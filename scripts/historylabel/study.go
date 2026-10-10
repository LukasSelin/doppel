package main

// The cost study: do the pairs doppel flags at revision T cost their
// maintainers anything over T..pin, measured against pairs doppel did not flag
// drawn from the same T tree and matched on locality and size?
//
//	historylabel study -repo <full clone> -snapshot <analyze --format json at T>
//	    -report <analyze text report at T, same flags> -corpus cobra
//	    -since <T> -pin <pinned commit> -out cobra.cost.json
//
// The verdict machinery is judge's, unchanged: the walk simply starts at T
// (walk's since), so every edit judge sees is one inside the window and its
// "pin" is the end of the window. What this file adds is the population —
// treated pairs by report rank, matched controls — and the per-pair counts
// the summary turns into rates. See examples/cost-study.md for the
// pre-registered design.

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"math/bits"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	top50Max = 50  // ranks 1..50 are the top50 group
	tailMax  = 500 // ranks 51..500 are the tail group
	// maxDraws bounds the rejection sampler per treated pair. A stratum that
	// cannot supply the controls inside it is reported short, never widened:
	// widening the match would compare against a different population.
	maxDraws = 4000
)

type studySnapshot struct {
	Params json.RawMessage `json:"params"`
	Units  []struct {
		Key, Package, Name, File string
		Line                     int
	} `json:"units"`
	Pairs []struct{ A, B string } `json:"pairs"`
}

// sunit is one function of the T snapshot, with what matching needs.
type sunit struct {
	key, pkg, name, file, dir, top string
	lines, bucket                  int
}

// rankedPair is one entry of the text report: its rank and kind are exactly
// what the report printed, which no snapshot field carries.
type rankedPair struct {
	rank         int
	aFile, bFile string
	aLine, bLine int
	kind         string
}

var (
	rankLine = regexp.MustCompile(`^#(\d+)\s+code-shape:`)
	sideLine = regexp.MustCompile(`^  ([AB])  (\S+):(\d+)\s+\S+\s*$`)
	kindLine = regexp.MustCompile(`^  kind: (.+?)(?: — .*)?$`)
)

// parseReport reads `doppel analyze` text output into ranked pairs.
func parseReport(p string) ([]rankedPair, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []rankedPair
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		l := sc.Text()
		if m := rankLine.FindStringSubmatch(l); m != nil {
			r, _ := strconv.Atoi(m[1])
			out = append(out, rankedPair{rank: r})
			continue
		}
		if len(out) == 0 {
			continue
		}
		cur := &out[len(out)-1]
		if m := sideLine.FindStringSubmatch(l); m != nil {
			n, _ := strconv.Atoi(m[3])
			if m[1] == "A" && cur.aFile == "" {
				cur.aFile, cur.aLine = m[2], n
			} else if m[1] == "B" && cur.bFile == "" {
				cur.bFile, cur.bLine = m[2], n
			}
			continue
		}
		if m := kindLine.FindStringSubmatch(l); m != nil && cur.kind == "" {
			cur.kind = m[1]
		}
	}
	return out, sc.Err()
}

// bucketOf is the log2 size bucket of a body line count: 1, 2-3, 4-7, ...,
// 64+.
func bucketOf(lines int) int {
	if lines < 1 {
		lines = 1
	}
	return min(bits.Len(uint(lines))-1, 6)
}

func topOf(file string) string {
	d := path.Dir(file)
	if d == "." {
		return "."
	}
	t, _, _ := strings.Cut(d, "/")
	return t
}

// locality classes, from closest to farthest.
const (
	locFile    = "file"
	locPackage = "package"
	locTop     = "top"
	locCross   = "cross"
)

func localityOf(a, b *sunit) string {
	switch {
	case a.file == b.file:
		return locFile
	case a.dir == b.dir:
		return locPackage
	case a.top == b.top:
		return locTop
	}
	return locCross
}

type upair struct{ a, b string }

func normPair(a, b string) upair {
	if a > b {
		a, b = b, a
	}
	return upair{a, b}
}

// lcg is the deterministic draw: the same 64-bit generator internal/calibrate
// uses, seeded per treated pair so a control set never depends on the order
// the pairs are visited in.
type lcg struct{ x uint64 }

func (r *lcg) intn(n int) int {
	r.x = r.x*6364136223846793005 + 1442695040888963407
	return int((r.x >> 33) % uint64(n))
}

func seedOf(parts ...string) uint64 {
	h := fnv.New64a()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return h.Sum64()
}

// studyPair is one row of the output: a treated pair or one of its controls,
// with every count the summary reads.
type studyPair struct {
	Set          int        `json:"set"`   // the treated pair's rank; its controls share it
	Group        string     `json:"group"` // top50, tail or control
	Rank         int        `json:"rank,omitempty"`
	Kind         string     `json:"kind,omitempty"`
	A            string     `json:"a"`
	B            string     `json:"b"`
	Locality     string     `json:"locality"`
	LinesA       int        `json:"linesA"`
	LinesB       int        `json:"linesB"`
	EditsA       int        `json:"editsA"` // non-noisy body modifications in the window
	EditsB       int        `json:"editsB"`
	Cochanges    int        `json:"cochanges"`
	Lagged       int        `json:"lagged"`
	Unpropagated int        `json:"unpropagated"`
	Extracted    int        `json:"extracted"`
	Consolidated int        `json:"consolidated"`
	GoneA        bool       `json:"goneA"` // absent at the pin: deleted, renamed or its file renamed
	GoneB        bool       `json:"goneB"`
	Evidence     []evidence `json:"evidence,omitempty"`
}

type studyOut struct {
	Corpus        string          `json:"corpus"`
	Since         string          `json:"since"`
	SinceDate     string          `json:"sinceDate"`
	Pin           string          `json:"pin"`
	PinDate       string          `json:"pinDate"`
	WindowCommits int             `json:"windowCommits"` // replayed: non-merge, touching a tracked directory
	Params        json.RawMessage `json:"params"`
	Functions     int             `json:"functions"`
	Unresolved    int             `json:"unresolved"` // snapshot units historylabel could not find at T
	Reported      int             `json:"reported"`   // pairs in the report
	Skipped       int             `json:"skipped"`    // treated-range pairs dropped: unresolved side or same-site
	ControlsPer   int             `json:"controlsPer"`
	Short         int             `json:"short"` // treated pairs that got fewer than ControlsPer controls
	Pairs         []studyPair     `json:"pairs"`
}

// window resolves T and the pin to full SHAs and commit dates.
func (r *repo) window(since, pin string) (sinceSHA, sinceDate, pinSHA, pinDate string, err error) {
	rev := func(s string) (string, string, error) {
		out, err := r.git("log", "-1", "--format=%H %cs", s)
		if err != nil {
			return "", "", err
		}
		f := strings.Fields(out)
		return f[0], f[1], nil
	}
	if sinceSHA, sinceDate, err = rev(since); err != nil {
		return
	}
	pinSHA, pinDate, err = rev(pin)
	return
}

// resolveAt finds one unit as historylabel sees it at T. A unit it cannot
// find there (a non-Go file, a receiver it renders differently) yields nil and
// bumps unresolved: nothing about its history could be read.
func resolveAt(ts *tables, sinceSHA, key, pkg, name, file string, unresolved *int) (*sunit, error) {
	if !strings.HasSuffix(file, ".go") {
		*unresolved++
		return nil, nil
	}
	dir := path.Dir(file)
	t, _, err := ts.at(sinceSHA, dir)
	if err != nil {
		return nil, err
	}
	f := t.lookup(histName(name), path.Base(file))
	if f == nil {
		if *unresolved++; *unresolved <= 5 {
			fmt.Fprintf(os.Stderr, "not found at T: %s (%s)\n", key, file)
		}
		return nil, nil
	}
	su := &sunit{key: key, pkg: pkg, name: histName(name), file: file, dir: dir, top: topOf(file), lines: len(f.Lines())}
	su.bucket = bucketOf(su.lines)
	return su, nil
}

func studyMain(args []string) {
	fs := flag.NewFlagSet("study", flag.ExitOnError)
	repoDir := fs.String("repo", "", "full-history clone of the corpus")
	snapPath := fs.String("snapshot", "", "doppel analyze --format json at -since")
	reportPath := fs.String("report", "", "doppel analyze text report at -since, same flags as -snapshot")
	corpus := fs.String("corpus", "", "corpus name")
	since := fs.String("since", "", "revision T the snapshot was taken at")
	pin := fs.String("pin", "", "end of the window")
	k := fs.Int("controls", 3, "matched controls per treated pair")
	outPath := fs.String("out", "", "write the study JSON here")
	fs.IntVar(&minCochanges, "min-cochanges", minCochanges, "as for labelling")
	fs.Parse(args)
	if *repoDir == "" || *snapPath == "" || *reportPath == "" || *since == "" || *pin == "" || *outPath == "" {
		fs.Usage()
		os.Exit(2)
	}
	if err := runStudy(*repoDir, *snapPath, *reportPath, *corpus, *since, *pin, *k, *outPath); err != nil {
		fmt.Fprintln(os.Stderr, "historylabel study:", err)
		os.Exit(1)
	}
}

func runStudy(repoDir, snapPath, reportPath, corpus, since, pin string, k int, outPath string) error {
	var snap studySnapshot
	if err := readJSON(snapPath, &snap); err != nil {
		return err
	}
	ranked, err := parseReport(reportPath)
	if err != nil {
		return err
	}
	r, err := openRepo(repoDir)
	if err != nil {
		return err
	}
	defer r.close()
	sinceSHA, sinceDate, pinSHA, pinDate, err := r.window(since, pin)
	if err != nil {
		return err
	}
	out := studyOut{Corpus: corpus, Since: sinceSHA, SinceDate: sinceDate, Pin: pinSHA, PinDate: pinDate,
		Params: snap.Params, Functions: len(snap.Units), Reported: len(ranked), ControlsPer: k}

	// Every unit as historylabel sees it at T. An unresolved one can be
	// neither treated nor a control.
	ts := newTables(r)
	units := map[string]*sunit{}
	byLoc := map[string]*sunit{}
	var keys []string
	for _, u := range snap.Units {
		su, err := resolveAt(ts, sinceSHA, u.Key, u.Package, u.Name, u.File, &out.Unresolved)
		if err != nil {
			return err
		}
		if su == nil {
			continue
		}
		units[u.Key] = su
		byLoc[u.File+":"+strconv.Itoa(u.Line)] = su
		keys = append(keys, u.Key)
	}
	sort.Strings(keys)
	sameSite := func(a, b *sunit) bool { return a == b || a.name == b.name && a.file == b.file }

	reported := map[upair]bool{}
	for _, p := range snap.Pairs {
		reported[normPair(p.A, p.B)] = true
	}
	type treated struct {
		rp   rankedPair
		a, b *sunit
	}
	var tr []treated
	for _, rp := range ranked {
		a, b := byLoc[rp.aFile+":"+strconv.Itoa(rp.aLine)], byLoc[rp.bFile+":"+strconv.Itoa(rp.bLine)]
		if a != nil && b != nil {
			reported[normPair(a.key, b.key)] = true
		}
		if rp.rank > tailMax {
			continue
		}
		if a == nil || b == nil || sameSite(a, b) {
			out.Skipped++
			continue
		}
		tr = append(tr, treated{rp, a, b})
	}

	// Indexes for the matched draw: by bucket, and by bucket within a file,
	// a directory and a top-level directory.
	byBucket := map[int][]*sunit{}
	byFile := map[string][]*sunit{}
	byDir := map[string][]*sunit{}
	byTop := map[string][]*sunit{}
	bk := func(s string, b int) string { return s + "\x00" + strconv.Itoa(b) }
	for _, key := range keys {
		u := units[key]
		byBucket[u.bucket] = append(byBucket[u.bucket], u)
		byFile[bk(u.file, u.bucket)] = append(byFile[bk(u.file, u.bucket)], u)
		byDir[bk(u.dir, u.bucket)] = append(byDir[bk(u.dir, u.bucket)], u)
		byTop[bk(u.top, u.bucket)] = append(byTop[bk(u.top, u.bucket)], u)
	}

	var rows []studyPair
	row := func(set int, group string, a, b *sunit) studyPair {
		if a.key > b.key {
			a, b = b, a
		}
		return studyPair{Set: set, Group: group, A: a.key, B: b.key, Locality: localityOf(a, b),
			LinesA: a.lines, LinesB: b.lines}
	}
	for _, t := range tr {
		g := "top50"
		if t.rp.rank > top50Max {
			g = "tail"
		}
		tp := row(t.rp.rank, g, t.a, t.b)
		tp.Rank, tp.Kind = t.rp.rank, t.rp.kind
		if tp.Kind == "" {
			tp.Kind = "none"
		}
		rows = append(rows, tp)

		loc := localityOf(t.a, t.b)
		rng := lcg{seedOf(corpus, sinceSHA, tp.A, tp.B)}
		first, second := t.a.bucket, t.b.bucket
		got := map[upair]bool{}
		for draw := 0; draw < maxDraws && len(got) < k; draw++ {
			xs := byBucket[first]
			if len(xs) == 0 {
				break
			}
			x := xs[rng.intn(len(xs))]
			var pool []*sunit
			switch loc {
			case locFile:
				pool = byFile[bk(x.file, second)]
			case locPackage:
				pool = byDir[bk(x.dir, second)]
			case locTop:
				pool = byTop[bk(x.top, second)]
			default:
				pool = byBucket[second]
			}
			if len(pool) == 0 {
				continue
			}
			y := pool[rng.intn(len(pool))]
			if sameSite(x, y) || localityOf(x, y) != loc {
				continue
			}
			p := normPair(x.key, y.key)
			if reported[p] || got[p] {
				continue
			}
			got[p] = true
			rows = append(rows, row(t.rp.rank, "control", x, y))
		}
		if len(got) < k {
			out.Short++
		}
	}

	// One life per function in any row, then the window replay.
	h := &history{r: r, ts: ts, lives: map[string]*life{}, pinSHA: pinSHA}
	dirSet := map[string]bool{}
	for _, p := range rows {
		for _, key := range []string{p.A, p.B} {
			if h.lives[key] != nil {
				continue
			}
			u := units[key]
			h.lives[key] = &life{key: key, dir: u.dir, name: u.name, file: path.Base(u.file)}
			dirSet[u.dir] = true
		}
	}
	var dirs []string
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	fmt.Fprintf(os.Stderr, "%s: %d treated pairs, %d rows, %d functions in %d directories; replaying %s..%s\n",
		corpus, len(tr), len(rows), len(h.lives), len(dirs), sinceSHA[:9], pinSHA[:9])
	if err := h.walk(sinceSHA, pinSHA, pinSHA, dirs); err != nil {
		return err
	}
	out.WindowCommits = len(h.commits)

	for i := range rows {
		judgeInto(h, &rows[i])
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Set != rows[j].Set {
			return rows[i].Set < rows[j].Set
		}
		ci, cj := rows[i].Group == "control", rows[j].Group == "control"
		if ci != cj {
			return cj
		}
		if rows[i].A != rows[j].A {
			return rows[i].A < rows[j].A
		}
		return rows[i].B < rows[j].B
	})
	out.Pairs = rows
	return writeJSON(outPath, out)
}

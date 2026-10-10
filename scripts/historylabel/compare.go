package main

// The ranker outcome comparison: rank one tree at revision T under doppel and
// every baseline (internal/bench TestRankingsAt writes the lists), then judge
// each listed pair on what happened to it over T..pin. See
// examples/ranker-outcomes.md for the pre-registered design.
//
//	historylabel compare -repo <full clone> -rankings <corpus>.rankings.json \
//	    -since <T> -pin <pinned commit> -out <corpus>.outcomes.json
//
// It differs from `study` in one place only: the pairs come from several
// ranked lists instead of one report plus matched controls. The walk, the
// judge and every exclusion are study's, unchanged, and each distinct pair is
// judged once however many lists hold it.

import (
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
)

// rankingsIn is internal/bench's rankingsFile.
type rankingsIn struct {
	Corpus      string `json:"corpus"`
	Calibration string `json:"calibration"`
	Functions   int    `json:"functions"`
	Union       int    `json:"union"`
	Report      int    `json:"report"`
	Units       []struct {
		Key, Package, Name, File string
		Line                     int
	} `json:"units"`
	Lists []struct {
		Method string `json:"method"`
		Pairs  []struct {
			Rank int    `json:"rank"`
			A    string `json:"a"`
			B    string `json:"b"`
		} `json:"pairs"`
	} `json:"lists"`
}

// outcomeList is one method's list after dropping pairs history cannot read:
// Pairs are indexes into compareOut.Pairs, in rank order.
type outcomeList struct {
	Method  string `json:"method"`
	Ranks   []int  `json:"ranks"` // the bench rank of each kept pair
	Pairs   []int  `json:"pairs"`
	Dropped int    `json:"dropped"` // unresolved side or same site
}

type compareOut struct {
	Corpus        string        `json:"corpus"`
	Since         string        `json:"since"`
	SinceDate     string        `json:"sinceDate"`
	Pin           string        `json:"pin"`
	PinDate       string        `json:"pinDate"`
	WindowCommits int           `json:"windowCommits"`
	SweepScope    string        `json:"sweepScope,omitempty"` // set only under -sweep-scope commit
	Calibration   string        `json:"calibration"`
	Functions     int           `json:"functions"`
	Union         int           `json:"union"`
	Report        int           `json:"report"`
	Unresolved    int           `json:"unresolved"` // listed units historylabel could not find at T
	Lists         []outcomeList `json:"lists"`
	Pairs         []studyPair   `json:"pairs"`
}

func compareMain(args []string) {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	repoDir := fs.String("repo", "", "full-history clone of the corpus")
	rankPath := fs.String("rankings", "", "TestRankingsAt output for the tree at -since")
	since := fs.String("since", "", "revision T the rankings were taken at")
	pin := fs.String("pin", "", "end of the window")
	outPath := fs.String("out", "", "write the outcomes JSON here")
	fs.IntVar(&minCochanges, "min-cochanges", minCochanges, "as for labelling")
	minDoppel := fs.Int("min-doppel", 0, "refuse (exit 3) before reading the window when doppel's list has fewer pairs")
	fs.StringVar(&sweepScope, "sweep-scope", sweepScope, "what a commit's sweep, family and campaign counts read: walked (the tracked directories, every earlier study's judge) or commit (the commit's whole Go diff, so a pair's outcome does not depend on the other pairs judged)")
	fs.Parse(args)
	if !validScope(sweepScope) {
		fmt.Fprintf(os.Stderr, "historylabel compare: -sweep-scope %q: want %s or %s\n", sweepScope, scopeWalked, scopeCommit)
		os.Exit(2)
	}
	if *repoDir == "" || *rankPath == "" || *since == "" || *pin == "" || *outPath == "" {
		fs.Usage()
		os.Exit(2)
	}
	if *minDoppel > 0 {
		var in rankingsIn
		if err := readJSON(*rankPath, &in); err != nil {
			fmt.Fprintln(os.Stderr, "historylabel compare:", err)
			os.Exit(1)
		}
		n := -1
		for _, l := range in.Lists {
			if l.Method == doppelMethod {
				n = len(l.Pairs)
			}
		}
		if n < *minDoppel {
			fmt.Fprintf(os.Stderr, "historylabel compare: inadmissible: doppel lists %d pairs, fewer than %d\n", n, *minDoppel)
			os.Exit(3)
		}
	}
	if err := runCompare(*repoDir, *rankPath, *since, *pin, *outPath); err != nil {
		fmt.Fprintln(os.Stderr, "historylabel compare:", err)
		os.Exit(1)
	}
}

func runCompare(repoDir, rankPath, since, pin, outPath string) error {
	var in rankingsIn
	if err := readJSON(rankPath, &in); err != nil {
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
	out := compareOut{Corpus: in.Corpus, Since: sinceSHA, SinceDate: sinceDate, Pin: pinSHA, PinDate: pinDate,
		Calibration: in.Calibration, Functions: in.Functions, Union: in.Union, Report: in.Report}

	// Every listed unit as historylabel sees it at T, exactly as study
	// resolves the snapshot's.
	ts := newTables(r)
	units := map[string]*sunit{}
	for _, u := range in.Units {
		su, err := resolveAt(ts, sinceSHA, u.Key, u.Package, u.Name, u.File, &out.Unresolved)
		if err != nil {
			return err
		}
		if su != nil {
			units[u.Key] = su
		}
	}
	sameSite := func(a, b *sunit) bool { return a == b || a.name == b.name && a.file == b.file }

	// The distinct pairs, in first-seen order over the lists.
	index := map[upair]int{}
	for _, l := range in.Lists {
		ol := outcomeList{Method: l.Method}
		for _, p := range l.Pairs {
			a, b := units[p.A], units[p.B]
			if a == nil || b == nil || sameSite(a, b) {
				ol.Dropped++
				continue
			}
			k := normPair(a.key, b.key)
			i, ok := index[k]
			if !ok {
				if a.key > b.key {
					a, b = b, a
				}
				i = len(out.Pairs)
				index[k] = i
				out.Pairs = append(out.Pairs, studyPair{A: a.key, B: b.key, Locality: localityOf(a, b),
					LinesA: a.lines, LinesB: b.lines})
			}
			ol.Ranks = append(ol.Ranks, p.Rank)
			ol.Pairs = append(ol.Pairs, i)
		}
		out.Lists = append(out.Lists, ol)
	}

	h := &history{r: r, ts: ts, lives: map[string]*life{}, pinSHA: pinSHA, scope: sweepScope}
	if sweepScope == scopeCommit {
		out.SweepScope = scopeCommit
	}
	dirSet := map[string]bool{}
	for _, p := range out.Pairs {
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
	fmt.Fprintf(os.Stderr, "%s: %d lists, %d distinct pairs, %d functions in %d directories; replaying %s..%s\n",
		in.Corpus, len(out.Lists), len(out.Pairs), len(h.lives), len(dirs), sinceSHA[:9], pinSHA[:9])
	if err := h.walk(sinceSHA, pinSHA, pinSHA, dirs); err != nil {
		return err
	}
	out.WindowCommits = len(h.commits)
	for i := range out.Pairs {
		judgeInto(h, &out.Pairs[i])
	}
	return writeJSON(outPath, out)
}

// judgeInto fills one pair's outcome counts from the window replay: study's
// arithmetic, shared so the two subcommands cannot count differently.
func judgeInto(h *history, p *studyPair) {
	a, b := h.lives[p.A], h.lives[p.B]
	edits := func(l *life) int {
		n := 0
		for _, e := range l.edits {
			if e.modified() && !e.c.noisy() {
				n++
			}
		}
		return n
	}
	v := h.judge(a, b)
	p.EditsA, p.EditsB = edits(a), edits(b)
	p.GoneA, p.GoneB = h.atPin(a) == nil, h.atPin(b) == nil
	for _, e := range v.Evidence {
		switch e.Kind {
		case "co-change":
			p.Cochanges++
		case "lagged-sync":
			p.Lagged++
		case "unpropagated-fix":
			p.Unpropagated++
		case "extracted":
			p.Extracted++
		case "consolidated":
			p.Consolidated++
		default:
			continue // diverged: not an outcome here
		}
		p.Evidence = append(p.Evidence, e)
	}
}

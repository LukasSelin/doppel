// Command historylabel derives golden labels for doppel's benchmark from a
// repository's git history instead of from a reviewer's reading of two bodies.
//
// A reviewer can say two functions look alike. Whether the duplication is
// drift or a decision is not in the bodies at all — it is in what the
// maintainers did with them: removed one in favour of the other, applied the
// same change to both, fixed one and left the other carrying the bug. That is
// evidence doppel never reads (it reads no git history, by construction),
// which is what makes it usable as an oracle for doppel rather than a mirror
// of it.
//
// It lives outside the doppel module for the reason scripts/timeline.sh does:
// the module must not know git exists.
//
//	historylabel -repo <full clone> -snapshot <analyze --format json at pin> \
//	             -pin v1.10.2 [-until HEAD] [-labels hand.labels.json] [-out history.labels.json]
//
// The pairs judged are the snapshot's candidate set plus every hand-labelled
// pair. Functions are followed by name: history before a rename is invisible,
// which costs evidence and never invents it.
//
// Two subcommands reuse the same walk for the cost study (study.go,
// summarize.go): `historylabel study` measures one corpus over a window,
// `historylabel summarize` turns the per-corpus files into rates and ratios.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

type snapshot struct {
	Units []struct {
		Key, Package, Name, File string
	} `json:"units"`
	Pairs []struct {
		A, B string
	} `json:"pairs"`
}

type handLabel struct {
	A, B, AFile, BFile, Class, Note string
}

type handFile struct {
	Corpus     string      `json:"corpus"`
	Population string      `json:"population"`
	Labels     []handLabel `json:"labels"`
}

// outLabel is internal/bench's Label shape plus the evidence behind it. The
// bench decoder ignores fields it does not know, so the file scores as-is.
type outLabel struct {
	A        string     `json:"a"`
	B        string     `json:"b"`
	AFile    string     `json:"aFile,omitempty"`
	BFile    string     `json:"bFile,omitempty"`
	Class    string     `json:"class"`
	Note     string     `json:"note"`
	Verdict  string     `json:"verdict"`
	Origin   string     `json:"origin"`
	Evidence []evidence `json:"evidence"`
}

type outFile struct {
	Corpus     string     `json:"corpus"`
	Reviewed   string     `json:"reviewed"`
	Population string     `json:"population"`
	Source     string     `json:"source"`
	Labels     []outLabel `json:"labels"`
}

// classOf maps a verdict onto the benchmark's classes. Stated once and not
// tuned against the hand labels — that would make the agreement number below
// a measurement of the tuning.
//
//	consolidated -> merge: the maintainers replaced one with the other, the
//	    only evidence that the whole bodies were one thing
//	extracted -> refactor: code moved out of both into one helper is the
//	    refactor itself, done; it shows a shared *part*, not that the rest of
//	    the two bodies is the same function
//	synced -> coupled: a change that had to land in both, in separate
//	    commits, proves the two are kept in step and nothing about whether
//	    they should be one. Mirror and lifecycle pairs (Create/Delete,
//	    Encode/Decode) co-change exactly as clones do, and a hand review
//	    calls those false positives, so a co-change cannot be a refactor
//	    claim without contradicting it
//	diverged -> false_positive, only under -weak
//	lagged, synced-once -> nothing: reported only, see verdictOrder
//	unpropagated -> nothing: reported for review, never a label. History says
//	    one side was fixed and the other still has the old code, but not
//	    which side was wrong — on cobra the fixed side was the one that had
//	    been copied badly (PrintErrln printing to stdout like Println).
func classOf(v string, weak bool) string {
	switch v {
	case vConsolidated:
		return "merge"
	case vExtracted:
		return "refactor"
	case vSynced:
		return "coupled"
	case vDiverged:
		if weak {
			return "false_positive"
		}
	}
	return ""
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "study":
			studyMain(os.Args[2:])
			return
		case "compare":
			compareMain(os.Args[2:])
			return
		case "compare-summary":
			compareSummaryMain(os.Args[2:])
			return
		case "rolling-summary":
			rollingSummaryMain(os.Args[2:])
			return
		case "summarize":
			summarizeMain(os.Args[2:])
			return
		}
	}
	repoDir := flag.String("repo", "", "full-history clone of the corpus")
	snapPath := flag.String("snapshot", "", "doppel analyze --format json --top 0 --max-per-func 0, run at -pin")
	pin := flag.String("pin", "", "revision the snapshot was taken at (the benchmark's pinned commit)")
	until := flag.String("until", "HEAD", "how far past the pin to look for consolidations")
	handPath := flag.String("labels", "", "hand-reviewed labels to measure agreement against")
	outPath := flag.String("out", "", "write history labels here (bench labels format)")
	corpus := flag.String("corpus", "", "corpus name for the output file (default: the hand labels' corpus)")
	weak := flag.Bool("weak", false, "emit diverged pairs as false_positive")
	flag.IntVar(&minCochanges, "min-cochanges", minCochanges, "separate commits that must apply the same change to both sides before a pair is labelled synced")
	flag.Parse()
	if *repoDir == "" || *snapPath == "" || *pin == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*repoDir, *snapPath, *pin, *until, *handPath, *outPath, *corpus, *weak); err != nil {
		fmt.Fprintln(os.Stderr, "historylabel:", err)
		os.Exit(1)
	}
}

func run(repoDir, snapPath, pin, until, handPath, outPath, corpus string, weak bool) error {
	var snap snapshot
	if err := readJSON(snapPath, &snap); err != nil {
		return err
	}
	var hand handFile
	if handPath != "" {
		if err := readJSON(handPath, &hand); err != nil {
			return err
		}
	}
	if corpus == "" {
		corpus = hand.Corpus
	}

	type unit struct{ key, pkg, name, file string }
	byKey := map[string]unit{}
	for _, u := range snap.Units {
		byKey[u.Key] = unit{u.Key, u.Package, u.Name, u.File}
	}
	// A hand label names a side as package.Name, plus a file when the name
	// alone is ambiguous; resolve it to the snapshot key.
	resolve := func(qual, file string) (string, bool) {
		if u, ok := byKey[qual]; ok && (file == "" || u.file == file) {
			return qual, true
		}
		for _, u := range snap.Units {
			if u.Package+"."+u.Name == qual && (file == "" || u.File == file) {
				return u.Key, true
			}
		}
		return "", false
	}

	type pair struct{ a, b string }
	norm := func(a, b string) pair {
		if a > b {
			a, b = b, a
		}
		return pair{a, b}
	}
	pairs := map[pair]bool{}
	sameSite := 0
	for _, p := range snap.Pairs {
		// Two declarations of one name in one file (init, which Go allows
		// repeatedly) are distinct snapshot keys but one side to the bench
		// labels format, which names a side by package.Name and file — and
		// one function to the history walk, which follows a name through a
		// file. Neither could say which init a verdict is about.
		ua, ub := byKey[p.A], byKey[p.B]
		if p.A == p.B || ua.pkg == ub.pkg && ua.name == ub.name && ua.file == ub.file {
			sameSite++
			continue
		}
		pairs[norm(p.A, p.B)] = true
	}
	if sameSite > 0 {
		fmt.Fprintf(os.Stderr, "%d pairs between same-named functions in one file skipped: no label can tell their sides apart\n", sameSite)
	}
	handBy := map[pair]handLabel{}
	for _, l := range hand.Labels {
		a, okA := resolve(l.A, l.AFile)
		b, okB := resolve(l.B, l.BFile)
		if !okA || !okB {
			fmt.Fprintf(os.Stderr, "hand label %s <-> %s: not in the snapshot, skipped\n", l.A, l.B)
			continue
		}
		p := norm(a, b)
		pairs[p] = true
		handBy[p] = l
	}

	r, err := openRepo(repoDir)
	if err != nil {
		return err
	}
	defer r.close()
	pinSHA, err := r.git("rev-parse", pin+"^{commit}")
	if err != nil {
		return err
	}
	h := &history{r: r, ts: newTables(r), lives: map[string]*life{}, pinSHA: strings.TrimSpace(pinSHA)}
	dirSet := map[string]bool{}
	for p := range pairs {
		for _, k := range []string{p.a, p.b} {
			u := byKey[k]
			if !strings.HasSuffix(u.file, ".go") {
				continue
			}
			l := &life{key: k, dir: path.Dir(u.file), name: histName(u.name), file: path.Base(u.file)}
			h.lives[k] = l
			dirSet[l.dir] = true
		}
	}
	var dirs []string
	for d := range dirSet {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	fmt.Fprintf(os.Stderr, "walking history of %d directories for %d functions in %d pairs\n", len(dirs), len(h.lives), len(pairs))
	if err := h.walk("", pin, until, dirs); err != nil {
		return err
	}
	sweeps := 0
	for _, c := range h.commits {
		if c.sweep {
			sweeps++
		}
	}
	fmt.Fprintf(os.Stderr, "%d commits replayed (%d after the pin), %d treated as sweeps\n",
		len(h.commits), countPost(h.commits), sweeps)

	keys := make([]pair, 0, len(pairs))
	for p := range pairs {
		keys = append(keys, p)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].a != keys[j].a {
			return keys[i].a < keys[j].a
		}
		return keys[i].b < keys[j].b
	})

	verdicts := map[pair]verdict{}
	for _, p := range keys {
		a, b := h.lives[p.a], h.lives[p.b]
		if a == nil || b == nil {
			continue
		}
		verdicts[p] = h.judge(a, b)
	}

	report(os.Stdout, keys, verdicts, handBy, len(snap.Pairs), weak)

	if outPath == "" {
		return nil
	}
	of := outFile{
		Corpus:     corpus,
		Reviewed:   time.Now().Format("2006-01-02"),
		Population: hand.Population,
		Source:     fmt.Sprintf("git history of %s up to %s, pairs from the snapshot at %s", path.Base(repoDir), until, pin),
	}
	if of.Population == "" {
		of.Population = "exclude"
	}
	for _, p := range keys {
		v, ok := verdicts[p]
		cls := classOf(v.Verdict, weak)
		if !ok || cls == "" {
			continue
		}
		l := outLabel{Class: cls, Verdict: v.Verdict, Origin: v.Origin, Evidence: v.Evidence}
		l.A, l.AFile = labelSide(byKey[p.a].key, byKey[p.a].pkg, byKey[p.a].name, byKey[p.a].file)
		l.B, l.BFile = labelSide(byKey[p.b].key, byKey[p.b].pkg, byKey[p.b].name, byKey[p.b].file)
		l.Note = noteOf(v)
		of.Labels = append(of.Labels, l)
	}
	return writeJSON(outPath, of)
}

// labelSide writes a side the way the bench labels do: package.Name, with the
// file only when the snapshot had to disambiguate the key.
func labelSide(key, pkg, name, file string) (string, string) {
	if strings.Contains(key, "@") {
		return pkg + "." + name, file
	}
	return key, ""
}

func noteOf(v verdict) string {
	var parts []string
	for _, e := range v.Evidence {
		parts = append(parts, e.Kind+" "+strings.Join(e.Commits, ","))
	}
	return "history: " + strings.Join(parts, "; ") + " (" + v.Origin + ")"
}

func countPost(cs []*commit) int {
	n := 0
	for _, c := range cs {
		if c.post {
			n++
		}
	}
	return n
}

// writeJSON writes v indented with a trailing newline, the one form every file
// this command writes takes.
func writeJSON(p string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o644)
}

func readJSON(p string, v any) error {
	data, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s: %w", p, err)
	}
	return nil
}

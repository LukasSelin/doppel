package main

// historylabel summarize turns `historylabel study` files into the cost
// study's tables: per-group rates, treated/control ratios with a bootstrap
// CI, the pre-registered decision, and the descriptive splits.
//
//	historylabel summarize -out summary.md [-samples samples.md] cobra.cost.json gin.cost.json ...
//
// Everything is deterministic: the bootstrap draws from a fixed-seed LCG, and
// corpora appear in argument order.

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
)

const (
	bootReps = 2000
	// costRatio and the CI bound are the pre-registered rule; see
	// examples/cost-study.md. They are not tunable.
	costRatio   = 2.0
	costCorpora = 4
)

// matchedSet is one treated pair and its controls.
type matchedSet struct {
	corpus   string
	treated  studyPair
	controls []studyPair
}

type metric struct {
	name, label string
	f           func(p studyPair) float64
	primary     bool
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

var metrics = []metric{
	{"M1", "normalised co-change (cochanges / min edits)", m1, true},
	{"M2", "lagged or unpropagated (share of pairs)", func(p studyPair) float64 { return b2f(p.Lagged+p.Unpropagated > 0) }, true},
	{"co", "any co-change", func(p studyPair) float64 { return b2f(p.Cochanges > 0) }, false},
	{"synced", "synced (>= 2 co-changes)", func(p studyPair) float64 { return b2f(p.Cochanges >= minCochanges) }, false},
	{"lagged", "lagged", func(p studyPair) float64 { return b2f(p.Lagged > 0) }, false},
	{"unprop", "unpropagated", func(p studyPair) float64 { return b2f(p.Unpropagated > 0) }, false},
	{"shared", "extracted or consolidated", func(p studyPair) float64 { return b2f(p.Extracted+p.Consolidated > 0) }, false},
	{"both", "both sides edited", func(p studyPair) float64 { return b2f(p.EditsA > 0 && p.EditsB > 0) }, false},
	{"edits", "mean edits per side", func(p studyPair) float64 { return float64(p.EditsA+p.EditsB) / 2 }, false},
	{"gone", "a side gone by the pin", func(p studyPair) float64 { return b2f(p.GoneA || p.GoneB) }, false},
}

// estimate is a treated/control comparison over a list of matched sets.
type estimate struct {
	n, nc         int
	treated, ctrl float64
	ratio, lo, hi float64
	undefined     int // bootstrap replicates where both means were 0
}

func means(sets []matchedSet, f func(studyPair) float64, idx []int) (float64, float64) {
	var st, sc float64
	nt, nc := 0, 0
	for _, i := range idx {
		s := sets[i]
		st += f(s.treated)
		nt++
		for _, c := range s.controls {
			sc += f(c)
			nc++
		}
	}
	if nt == 0 || nc == 0 {
		return 0, 0
	}
	return st / float64(nt), sc / float64(nc)
}

func ratioOf(t, c float64) float64 {
	switch {
	case c > 0:
		return t / c
	case t > 0:
		return math.Inf(1)
	}
	return math.NaN()
}

// estimateOf resamples matched sets with replacement inside each corpus, so a
// pooled estimate keeps every corpus's weight. Sets with no control cannot
// be compared and are left out.
func estimateOf(all []matchedSet, f func(studyPair) float64, seedLabel string) estimate {
	var sets []matchedSet
	for _, s := range all {
		if len(s.controls) > 0 {
			sets = append(sets, s)
		}
	}
	var e estimate
	e.n = len(sets)
	strata := map[string][]int{}
	var order []string
	idx := make([]int, len(sets))
	for i, s := range sets {
		idx[i] = i
		if _, ok := strata[s.corpus]; !ok {
			order = append(order, s.corpus)
		}
		strata[s.corpus] = append(strata[s.corpus], i)
		e.nc += len(s.controls)
	}
	e.treated, e.ctrl = means(sets, f, idx)
	e.ratio = ratioOf(e.treated, e.ctrl)
	if e.n == 0 {
		e.lo, e.hi = math.NaN(), math.NaN()
		return e
	}
	rng := lcg{seedOf("cost-study bootstrap", seedLabel)}
	var reps []float64
	draw := make([]int, 0, len(sets))
	for r := 0; r < bootReps; r++ {
		draw = draw[:0]
		for _, c := range order {
			st := strata[c]
			for range st {
				draw = append(draw, st[rng.intn(len(st))])
			}
		}
		t, c := means(sets, f, draw)
		q := ratioOf(t, c)
		if math.IsNaN(q) {
			e.undefined++
			continue
		}
		reps = append(reps, q)
	}
	if len(reps) == 0 {
		e.lo, e.hi = math.NaN(), math.NaN()
		return e
	}
	sort.Float64s(reps)
	e.lo = reps[int(0.025*float64(len(reps)-1))]
	e.hi = reps[int(math.Ceil(0.975*float64(len(reps)-1)))]
	return e
}

func fmtRatio(x float64) string {
	switch {
	case math.IsNaN(x):
		return "—"
	case math.IsInf(x, 1):
		return "∞"
	}
	return fmt.Sprintf("%.2f", x)
}

func (e estimate) cell() string {
	if e.n == 0 {
		return "—"
	}
	s := fmtRatio(e.ratio) + " [" + fmtRatio(e.lo) + ", " + fmtRatio(e.hi) + "]"
	if e.undefined > 0 {
		s += fmt.Sprintf(" (%d/%d undefined)", e.undefined, bootReps)
	}
	return s
}

// showsCost is the pre-registered per-corpus test.
func (e estimate) showsCost() bool {
	return e.n > 0 && e.ratio >= costRatio && e.lo > 1
}

type studyParams struct {
	Threshold float64 `json:"threshold"`
	StructMin float64 `json:"structMin"`
	Calibrate float64 `json:"calibrate"`
}

func setsOf(o studyOut, groups ...string) []matchedSet {
	want := map[string]bool{}
	for _, g := range groups {
		want[g] = true
	}
	var out []matchedSet
	byRank := map[int]int{}
	for _, p := range o.Pairs {
		if p.Group == "control" {
			if i, ok := byRank[p.Set]; ok {
				out[i].controls = append(out[i].controls, p)
			}
			continue
		}
		if want[p.Group] {
			byRank[p.Set] = len(out)
			out = append(out, matchedSet{corpus: o.Corpus, treated: p})
		}
	}
	return out
}

func summarizeMain(args []string) {
	fs := flag.NewFlagSet("summarize", flag.ExitOnError)
	outPath := fs.String("out", "", "write the markdown summary here (default stdout)")
	samplesPath := fs.String("samples", "", "write, per corpus, up to 10 treated pairs carrying a cost event, for hand-checking")
	fs.Parse(args)
	var studies []studyOut
	for _, p := range fs.Args() {
		var o studyOut
		if err := readJSON(p, &o); err != nil {
			fmt.Fprintln(os.Stderr, "historylabel summarize:", err)
			os.Exit(1)
		}
		studies = append(studies, o)
	}
	w := io.Writer(os.Stdout)
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "historylabel summarize:", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}
	summarize(w, studies)
	if *samplesPath != "" {
		f, err := os.Create(*samplesPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "historylabel summarize:", err)
			os.Exit(1)
		}
		defer f.Close()
		samples(f, studies)
	}
}

func summarize(w io.Writer, studies []studyOut) {
	fmt.Fprintf(w, "### Operating points\n\n")
	fmt.Fprintf(w, "| corpus | T | T date | pin date | window commits | functions | threshold | struct-min | reported | top50 | tail | short of controls |\n")
	fmt.Fprintf(w, "| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, o := range studies {
		var p studyParams
		json.Unmarshal(o.Params, &p)
		fmt.Fprintf(w, "| %s | `%s` | %s | %s | %d | %d | %.2f | %.2f | %d | %d | %d | %d |\n",
			o.Corpus, o.Since[:9], o.SinceDate, o.PinDate, o.WindowCommits, o.Functions, p.Threshold, p.StructMin,
			o.Reported, len(setsOf(o, "top50")), len(setsOf(o, "tail")), o.Short)
	}

	fmt.Fprintf(w, "\n### Decision (pre-registered: top50, M1 or M2 ratio >= %.1f with CI lower bound > 1, on >= %d corpora)\n\n", costRatio, costCorpora)
	fmt.Fprintf(w, "| corpus | M1 ratio [95%% CI] | M2 ratio [95%% CI] | shows cost | top50 pairs with an M1/M2 event | distinct commits behind them |\n")
	fmt.Fprintf(w, "| --- | --- | --- | --- | ---: | ---: |\n")
	showing := 0
	for _, o := range studies {
		sets := setsOf(o, "top50")
		m1 := estimateOf(sets, metrics[0].f, o.Corpus+"/top50/M1")
		m2 := estimateOf(sets, metrics[1].f, o.Corpus+"/top50/M2")
		yes := m1.showsCost() || m2.showsCost()
		if yes {
			showing++
		}
		pairs, commits := clustering(sets)
		fmt.Fprintf(w, "| %s | %s | %s | %s | %d | %d |\n", o.Corpus, m1.cell(), m2.cell(),
			map[bool]string{true: "**yes**", false: "no"}[yes], pairs, commits)
	}
	verdict := "**no evidence of cost**"
	if showing >= costCorpora {
		verdict = "**evidence of cost**"
	}
	fmt.Fprintf(w, "\n%d of %d corpora show cost: %s.\n", showing, len(studies), verdict)

	table := func(title string, sets map[string][]matchedSet, label string) {
		fmt.Fprintf(w, "\n%s\n\n", title)
		fmt.Fprintf(w, "| metric | top50 treated | control | ratio [95%% CI] | tail treated | control | ratio [95%% CI] |\n")
		fmt.Fprintf(w, "| --- | ---: | ---: | --- | ---: | ---: | --- |\n")
		for _, m := range metrics {
			name := m.label
			if m.primary {
				name = "**" + m.name + "** " + m.label
			}
			var cells []string
			for _, g := range []string{"top50", "tail"} {
				e := estimateOf(sets[g], m.f, label+"/"+g+"/"+m.name)
				if e.n == 0 {
					cells = append(cells, "—", "—", "—")
					continue
				}
				cells = append(cells, fmt.Sprintf("%.3f", e.treated), fmt.Sprintf("%.3f", e.ctrl), e.cell())
			}
			fmt.Fprintf(w, "| %s | %s |\n", name, strings.Join(cells, " | "))
		}
	}
	pooled := map[string][]matchedSet{}
	for _, o := range studies {
		per := map[string][]matchedSet{"top50": setsOf(o, "top50"), "tail": setsOf(o, "tail")}
		pooled["top50"] = append(pooled["top50"], per["top50"]...)
		pooled["tail"] = append(pooled["tail"], per["tail"]...)
		table(fmt.Sprintf("### %s (%d + %d treated pairs, %d controls each)", o.Corpus, len(per["top50"]), len(per["tail"]), o.ControlsPer), per, o.Corpus)
	}
	if len(studies) > 1 {
		table("### Pooled (resampled within corpus)", pooled, "pooled")
	}

	// Descriptive splits over every treated pair (top50 and tail together):
	// one corpus's top 50 is too few to split by anything.
	var every []matchedSet
	for _, o := range studies {
		every = append(every, setsOf(o, "top50", "tail")...)
	}
	split := func(title, label string, key func(studyPair) string) {
		groups := map[string][]matchedSet{}
		for _, s := range every {
			groups[key(s.treated)] = append(groups[key(s.treated)], s)
		}
		var names []string
		for k := range groups {
			names = append(names, k)
		}
		sort.Strings(names)
		fmt.Fprintf(w, "\n%s\n\n", title)
		fmt.Fprintf(w, "| %s | pairs | M1 treated / control | M1 ratio [95%% CI] | M2 treated / control | M2 ratio [95%% CI] | any co-change ratio [95%% CI] |\n", label)
		fmt.Fprintf(w, "| --- | ---: | --- | --- | --- | --- | --- |\n")
		for _, k := range names {
			sets := groups[k]
			m1 := estimateOf(sets, metrics[0].f, label+"/"+k+"/M1")
			m2 := estimateOf(sets, metrics[1].f, label+"/"+k+"/M2")
			co := estimateOf(sets, metrics[2].f, label+"/"+k+"/co")
			fmt.Fprintf(w, "| %s | %d | %.3f / %.3f | %s | %.3f / %.3f | %s | %s |\n",
				k, len(sets), m1.treated, m1.ctrl, m1.cell(), m2.treated, m2.ctrl, m2.cell(), co.cell())
		}
	}
	split("### By pair kind (pooled, ranks 1-500)", "kind", func(p studyPair) string { return p.Kind })
	split("### By locality (pooled, ranks 1-500)", "locality", func(p studyPair) string { return p.Locality })
}

// clustering counts the treated pairs carrying an M1 or M2 event and the
// distinct commits those events start from. Descriptive: the bootstrap
// resamples pairs, and one commit editing a family of four functions alike is
// six pairs' worth of events, so this is how a reader sees how few
// independent observations a ratio rests on.
func clustering(sets []matchedSet) (int, int) {
	pairs := 0
	commits := map[string]bool{}
	for _, s := range sets {
		p := s.treated
		if p.Cochanges+p.Lagged+p.Unpropagated == 0 {
			continue
		}
		pairs++
		for _, e := range p.Evidence {
			switch e.Kind {
			case "co-change", "lagged-sync", "unpropagated-fix":
				commits[e.Commits[0]] = true
			}
		}
	}
	return pairs, len(commits)
}

// samples lists, per corpus, the best-ranked treated pairs that carry any
// cost event, with the evidence, so a reader can open the commits.
func samples(w io.Writer, studies []studyOut) {
	for _, o := range studies {
		fmt.Fprintf(w, "## %s\n\n", o.Corpus)
		n := 0
		for _, p := range o.Pairs {
			if p.Group == "control" || len(p.Evidence) == 0 {
				continue
			}
			fmt.Fprintf(w, "- #%d %s ↔ %s (%s, %s; edits %d/%d)\n", p.Rank, p.A, p.B, p.Kind, p.Locality, p.EditsA, p.EditsB)
			for _, e := range p.Evidence {
				fmt.Fprintf(w, "  - %s %s: %s\n", e.Kind, strings.Join(e.Commits, ","), e.Detail)
			}
			if n++; n == 10 {
				break
			}
		}
		fmt.Fprintln(w)
	}
}

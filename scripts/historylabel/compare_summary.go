package main

// historylabel compare-summary turns `historylabel compare` files into the
// ranker outcome tables: per-method rates at the pre-registered depths, the
// doppel-minus-baseline differences with a bootstrap CI, and the decision.
//
//	historylabel compare-summary -out summary.md [-cost-dir examples/cost-study] \
//	    cobra.outcomes.json gin.outcomes.json ...
//
// The constants below are the pre-registered rule in
// examples/ranker-outcomes.md and are not tunable. Everything is
// deterministic: the bootstrap draws from a fixed-seed LCG per comparison.

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	primaryK      = 100 // M1 at this depth decides
	shallowK      = 50  // descriptive
	deepK         = 500 // ranks primaryK+1..deepK are descriptive
	beatsCorpora  = 3   // corpora doppel must win, with no loss, to beat a baseline
	doppelMethod  = "doppel"
	randomPrefix  = "random (seed "
	pooledRandom  = "random (3 seeds)"
	validityShare = 0.90
)

func compareSummaryMain(args []string) {
	fs := flag.NewFlagSet("compare-summary", flag.ExitOnError)
	outPath := fs.String("out", "", "write the markdown here (default stdout)")
	costDir := fs.String("cost-dir", "", "directory of the cost study's <corpus>.cost.json, for the agreement check")
	fs.Parse(args)
	if fs.NArg() == 0 {
		fs.Usage()
		os.Exit(2)
	}
	var corpora []compareOut
	for _, p := range fs.Args() {
		var c compareOut
		if err := readJSON(p, &c); err != nil {
			fmt.Fprintln(os.Stderr, "historylabel compare-summary:", err)
			os.Exit(1)
		}
		corpora = append(corpora, c)
	}
	var w io.Writer = os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "historylabel compare-summary:", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}
	writeCompareSummary(w, corpora, *costDir)
}

// m1 is the cost study's M1 for one pair, shared with summarize's metric table.
func m1(p studyPair) float64 {
	m := min(p.EditsA, p.EditsB)
	if m == 0 {
		return 0
	}
	return float64(p.Cochanges) / float64(m)
}

func anyEvent(p studyPair) bool {
	return p.Cochanges+p.Lagged+p.Unpropagated+p.Extracted+p.Consolidated > 0
}

// methodView is one method's kept pairs, in rank order, for one corpus.
type methodView struct {
	name  string
	pairs [][]int // per list (one, or three for pooled random): pair indexes
}

// slice is ranks lo..hi (1-based, inclusive) of every list, concatenated.
func (v methodView) slice(lo, hi int) []int {
	var out []int
	for _, l := range v.pairs {
		for i := lo - 1; i < hi && i < len(l); i++ {
			out = append(out, l[i])
		}
	}
	return out
}

func viewsOf(c compareOut) []methodView {
	var out []methodView
	var random methodView
	random.name = pooledRandom
	for _, l := range c.Lists {
		if strings.HasPrefix(l.Method, randomPrefix) {
			random.pairs = append(random.pairs, l.Pairs)
			continue
		}
		out = append(out, methodView{name: l.Method, pairs: [][]int{l.Pairs}})
	}
	if len(random.pairs) > 0 {
		out = append(out, random)
	}
	return out
}

// rates is every metric over one set of pairs.
type rates struct {
	n                 int
	m1, m2, e, m1clus float64
	commits           int
}

// rateOf reads the metrics over idx. The clustered M1 divides each co-change
// commit's credit among the pairs of idx that it touches.
func rateOf(c compareOut, idx []int) rates {
	r := rates{n: len(idx)}
	if r.n == 0 {
		return r
	}
	touches := map[string]int{}
	commits := map[string]bool{}
	for _, i := range idx {
		p := c.Pairs[i]
		for _, e := range p.Evidence {
			for _, sha := range e.Commits {
				commits[sha] = true
				if e.Kind == "co-change" {
					touches[sha]++
				}
			}
		}
	}
	for _, i := range idx {
		p := c.Pairs[i]
		r.m1 += m1(p)
		if p.Lagged+p.Unpropagated > 0 {
			r.m2++
		}
		if anyEvent(p) {
			r.e++
		}
		if m := min(p.EditsA, p.EditsB); m > 0 {
			credit := 0.0
			for _, e := range p.Evidence {
				if e.Kind == "co-change" && len(e.Commits) > 0 {
					credit += 1 / float64(touches[e.Commits[0]])
				}
			}
			r.m1clus += credit / float64(m)
		}
	}
	n := float64(r.n)
	r.m1, r.m2, r.e, r.m1clus = r.m1/n, r.m2/n, r.e/n, r.m1clus/n
	r.commits = len(commits)
	return r
}

// diffCI is M1(a) - M1(b) with a 95% percentile bootstrap CI, each side
// resampled independently.
func diffCI(c compareOut, a, b []int, seed string) (d, lo, hi float64) {
	va := make([]float64, len(a))
	vb := make([]float64, len(b))
	for i, x := range a {
		va[i] = m1(c.Pairs[x])
	}
	for i, x := range b {
		vb[i] = m1(c.Pairs[x])
	}
	mean := func(v []float64) float64 {
		s := 0.0
		for _, x := range v {
			s += x
		}
		return s / float64(len(v))
	}
	if len(va) == 0 || len(vb) == 0 {
		return math.NaN(), math.NaN(), math.NaN()
	}
	d = mean(va) - mean(vb)
	rng := lcg{seedOf("ranker-outcomes bootstrap", c.Corpus, seed)}
	reps := make([]float64, 0, bootReps)
	for range bootReps {
		sa, sb := 0.0, 0.0
		for range va {
			sa += va[rng.intn(len(va))]
		}
		for range vb {
			sb += vb[rng.intn(len(vb))]
		}
		reps = append(reps, sa/float64(len(va))-sb/float64(len(vb)))
	}
	sort.Float64s(reps)
	lo = reps[int(0.025*float64(len(reps)-1))]
	hi = reps[int(math.Ceil(0.975*float64(len(reps)-1)))]
	return d, lo, hi
}

func writeCompareSummary(w io.Writer, corpora []compareOut, costDir string) {
	fmt.Fprintf(w, "### Operating points\n\n")
	fmt.Fprintf(w, "| corpus | T | T date | pin date | window commits | functions at T | calibration | union | listed units unresolved |\n")
	fmt.Fprintf(w, "| --- | --- | --- | --- | ---: | ---: | --- | ---: | ---: |\n")
	for _, c := range corpora {
		fmt.Fprintf(w, "| %s | `%s` | %s | %s | %d | %d | %s | %d | %d |\n", c.Corpus, c.Since[:9], c.SinceDate, c.PinDate,
			c.WindowCommits, c.Functions, c.Calibration, c.Union, c.Unresolved)
	}
	fmt.Fprintln(w)

	if costDir != "" {
		fmt.Fprintf(w, "### Validity: agreement with the cost study's top 50\n\n")
		fmt.Fprintf(w, "| corpus | doppel (struct-min filtered) top 50 also in the cost study's top 50 | share | at least %.0f%% |\n", validityShare*100)
		fmt.Fprintf(w, "| --- | ---: | ---: | --- |\n")
		for _, c := range corpora {
			var cs studyOut
			if err := readJSON(filepath.Join(costDir, c.Corpus+".cost.json"), &cs); err != nil {
				fmt.Fprintf(w, "| %s | — | — | %v |\n", c.Corpus, err)
				continue
			}
			theirs := map[upair]bool{}
			for _, p := range cs.Pairs {
				if p.Group == "top50" {
					theirs[normPair(p.A, p.B)] = true
				}
			}
			var mine []int
			for _, v := range viewsOf(c) {
				if v.name == "doppel (struct-min filtered)" {
					mine = v.slice(1, shallowK)
				}
			}
			hit := 0
			for _, i := range mine {
				if theirs[normPair(c.Pairs[i].A, c.Pairs[i].B)] {
					hit++
				}
			}
			share := 0.0
			if len(mine) > 0 {
				share = float64(hit) / float64(len(mine))
			}
			ok := "yes"
			if share < validityShare {
				ok = "**no**"
			}
			fmt.Fprintf(w, "| %s | %d of %d | %.2f | %s |\n", c.Corpus, hit, len(mine), share, ok)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "### Rates per method\n\n")
	fmt.Fprintf(w, "M1 is mean co-changes per edit of the less-edited side; M2 the share with a lagged or unpropagated event; E the share with any event; C the distinct commits behind those events; M1 clustered divides each co-change commit among the pairs of the list it touches. Pooled random holds three lists, so its pair counts are three times the depth.\n\n")
	for _, c := range corpora {
		fmt.Fprintf(w, "#### %s\n\n", c.Corpus)
		fmt.Fprintf(w, "| method | dropped | n@%d | **M1@%d** | M2@%d | E@%d | C@%d | M1 clustered@%d | M1@%d | E@%d | M1 %d-%d | E %d-%d |\n",
			primaryK, primaryK, primaryK, primaryK, primaryK, primaryK, shallowK, shallowK, primaryK+1, deepK, primaryK+1, deepK)
		fmt.Fprintf(w, "| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
		dropped := map[string]int{}
		for _, l := range c.Lists {
			name := l.Method
			if strings.HasPrefix(name, randomPrefix) {
				name = pooledRandom
			}
			dropped[name] += l.Dropped
		}
		for _, v := range viewsOf(c) {
			p := rateOf(c, v.slice(1, primaryK))
			s := rateOf(c, v.slice(1, shallowK))
			d := rateOf(c, v.slice(primaryK+1, deepK))
			fmt.Fprintf(w, "| %s | %d | %d | **%.4f** | %.3f | %.3f | %d | %.4f | %.4f | %.3f | %.4f | %.3f |\n",
				v.name, dropped[v.name], p.n, p.m1, p.m2, p.e, p.commits, p.m1clus, s.m1, s.e, d.m1, d.e)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "### Primary comparison: M1@%d, doppel minus baseline\n\n", primaryK)
	type cell struct{ d, lo, hi float64 }
	var baselines []string
	cells := map[string]map[string]cell{}
	for _, c := range corpora {
		var dop []int
		for _, v := range viewsOf(c) {
			if v.name == doppelMethod {
				dop = v.slice(1, primaryK)
			}
		}
		for _, v := range viewsOf(c) {
			if v.name == doppelMethod {
				continue
			}
			if cells[v.name] == nil {
				cells[v.name] = map[string]cell{}
				baselines = append(baselines, v.name)
			}
			d, lo, hi := diffCI(c, dop, v.slice(1, primaryK), v.name)
			cells[v.name][c.Corpus] = cell{d, lo, hi}
		}
	}
	fmt.Fprintf(w, "| baseline |")
	for _, c := range corpora {
		fmt.Fprintf(w, " %s |", c.Corpus)
	}
	fmt.Fprintf(w, " doppel wins | baseline wins | verdict |\n| --- |")
	for range corpora {
		fmt.Fprintf(w, " --- |")
	}
	fmt.Fprintf(w, " ---: | ---: | --- |\n")
	for _, b := range baselines {
		fmt.Fprintf(w, "| %s |", b)
		win, loss := 0, 0
		for _, c := range corpora {
			x, ok := cells[b][c.Corpus]
			if !ok || math.IsNaN(x.d) {
				fmt.Fprintf(w, " — |")
				continue
			}
			mark := ""
			switch {
			case x.lo > 0:
				win++
				mark = " ✓"
			case x.hi < 0:
				loss++
				mark = " ✗"
			}
			fmt.Fprintf(w, " %+.4f [%+.4f, %+.4f]%s |", x.d, x.lo, x.hi, mark)
		}
		verdict := "not distinguishable"
		switch {
		case win >= beatsCorpora && loss == 0:
			verdict = "**doppel beats it**"
		case loss >= beatsCorpora && win == 0:
			verdict = "**it beats doppel**"
		}
		fmt.Fprintf(w, " %d | %d | %s |\n", win, loss, verdict)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "✓ doppel's CI lower bound is above 0 on that corpus; ✗ the upper bound is below 0. A verdict needs %d corpora one way and none the other.\n", beatsCorpora)
}

package main

// historylabel size-summary scores the size-aware rank variants of
// examples/size-aware-rank.md. Each input file is one unit — one corpus at one
// origin — whose lists were all judged in one `compare -sweep-scope commit`
// run, so every comparison at a unit reads one judgment per pair.
//
//	historylabel size-summary -out summary.md [-origins <dir of <corpus>.origins.tsv>] \
//	    a.size-outcomes.json a.o2.size-outcomes.json ...
//
// The constants are the pre-registered rule and are not tunable. Every draw is
// a fixed-seed LCG.

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"sort"
	"strings"
)

const (
	// sizeVariantPrefix names a variant list; everything else is doppel or a
	// detector.
	sizeVariantPrefix = "variant: "
	// sizeDevPool is the development list holding the whole union. It is
	// never a method.
	sizeDevPool = "union (development pool)"
	// minEventClusters is the floor on independent event clusters — clusters
	// of pairs joined by a shared evidence commit, holding at least one pair
	// with M1 > 0 — that a comparison's frame at a unit needs for the unit to
	// count in that comparison's pooled estimate. Below it the stratified
	// cluster bootstrap cannot resolve a difference (examples/outcome-
	// reanalysis.md), and the unit is reported but not pooled.
	minEventClusters = 3
	// smallSideLines: a pair whose smaller side has fewer lines at T is
	// below a clone detector's floor, for the small-family cost. 23 lines is
	// the median length of a 100-node function (dupl's default floor, in
	// syntax nodes) over the five development trees.
	smallSideLines = 23
)

// sizeDetectors are the clone detectors the gap criterion reads, in the
// clone-outcome study's names.
var sizeDetectors = []string{
	"dupl (t=100, default)", "dupl (t=50)",
	"token clones (≥100 nodes)", "token clones (≥50 nodes)", "code-shape (≥50 nodes)",
}

// sizeUnit is one corpus at one origin.
type sizeUnit struct {
	c     compareOut
	k     int
	views map[string]methodView
	names []string // methods in file order, the development pool left out
}

func (u sizeUnit) label() string { return fmt.Sprintf("%s o%d", u.c.Corpus, u.k) }

func (u sizeUnit) top(method string, depth int) []int {
	v, ok := u.views[method]
	if !ok {
		return nil
	}
	return v.slice(1, depth)
}

func (u sizeUnit) length(method string) int {
	v, ok := u.views[method]
	if !ok || len(v.pairs) == 0 {
		return 0
	}
	return len(v.pairs[0])
}

func sizeSummaryMain(args []string) {
	fs := flag.NewFlagSet("size-summary", flag.ExitOnError)
	outPath := fs.String("out", "", "write the markdown here (default stdout)")
	origins := fs.String("origins", "", "directory of <corpus>.origins.tsv; a file whose since is not in it is origin 1")
	fs.Parse(args)
	if fs.NArg() == 0 {
		fs.Usage()
		os.Exit(2)
	}
	var units []sizeUnit
	var corpora []string
	for _, p := range fs.Args() {
		var c compareOut
		if err := readJSON(p, &c); err != nil {
			fmt.Fprintln(os.Stderr, "historylabel size-summary:", err)
			os.Exit(1)
		}
		u := sizeUnit{c: c, k: 1, views: map[string]methodView{}}
		for _, v := range viewsOf(c) {
			if v.name == sizeDevPool {
				continue
			}
			u.views[v.name] = v
			u.names = append(u.names, v.name)
		}
		units = append(units, u)
		if !slices.Contains(corpora, c.Corpus) {
			corpora = append(corpora, c.Corpus)
		}
	}
	if *origins != "" {
		rows := readOrigins(*origins, corpora)
		for i := range units {
			for _, o := range rows {
				if o.corpus == units[i].c.Corpus && o.since == units[i].c.Since {
					fmt.Sscan(o.k, &units[i].k)
				}
			}
		}
	}
	var w io.Writer = os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "historylabel size-summary:", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}
	writeSizeSummary(w, units)
}

// sizeCell is one comparison at one unit, at matched depth.
type sizeCell struct {
	depth    int
	d        float64
	reps     []float64
	clusters int // independent event clusters in the frame
	lo, hi   float64
}

// compareAt is M1@depth(x) − M1@depth(y) at unit u, depth the shorter of the
// two lists and cap, with the paired-clustered bootstrap. ok is false when
// either list is empty there.
func compareAt(u sizeUnit, x, y string, depthCap int) (sizeCell, bool) {
	depth := min(u.length(x), u.length(y), depthCap)
	if depth == 0 {
		return sizeCell{}, false
	}
	la, lb := u.top(x, depth), u.top(y, depth)
	d, reps := pairedReps(u.c, la, lb, bootPairedClustered, seedOf("size-aware-rank", u.c.Corpus, fmt.Sprint(u.k), x, y, fmt.Sprint(depth)))
	cell := sizeCell{depth: depth, d: d, reps: reps, clusters: eventClusters(u.c, la, lb)}
	cell.lo, cell.hi = percentile95(slices.Clone(reps))
	return cell, true
}

// eventClusters counts the frame's clusters (pairs joined by a shared
// evidence commit) that hold a pair with M1 > 0.
func eventClusters(c compareOut, la, lb []int) int {
	u := unionOf(c, la, lb, true)
	n := 0
	for _, cl := range u.clusters {
		for _, i := range cl {
			if m1(c.Pairs[u.pairs[i]]) > 0 {
				n++
				break
			}
		}
	}
	return n
}

// sizePooled is a comparison pooled over units: the mean over corpora of the
// mean over that corpus's counting units, with the replicate-wise pooled
// percentile CI. Units below floor event clusters are left out.
type sizePooled struct {
	d, lo, hi      float64
	units, corpora int
}

func poolCells(units []sizeUnit, cells []*sizeCell, floor int) sizePooled {
	var corpora []string
	var keep []int
	for i, c := range cells {
		if c == nil || c.clusters < floor {
			continue
		}
		keep = append(keep, i)
		if !slices.Contains(corpora, units[i].c.Corpus) {
			corpora = append(corpora, units[i].c.Corpus)
		}
	}
	p := sizePooled{units: len(keep), corpora: len(corpora)}
	if len(keep) == 0 {
		p.d, p.lo, p.hi = math.NaN(), math.NaN(), math.NaN()
		return p
	}
	combine := func(val func(i int) float64) float64 {
		t := 0.0
		for _, corpus := range corpora {
			s, n := 0.0, 0
			for _, i := range keep {
				if units[i].c.Corpus == corpus {
					s += val(i)
					n++
				}
			}
			t += s / float64(n)
		}
		return t / float64(len(corpora))
	}
	p.d = combine(func(i int) float64 { return cells[i].d })
	reps := make([]float64, bootReps)
	for r := range reps {
		reps[r] = combine(func(i int) float64 { return cells[i].reps[r] })
	}
	p.lo, p.hi = percentile95(reps)
	return p
}

// sizeVerdict reads a pooled CI for "x minus y".
func sizeVerdict(p sizePooled, x, y string) string {
	switch {
	case math.IsNaN(p.d):
		return "not scored"
	case p.lo > 0:
		return x + " beats " + y
	case p.hi < 0:
		return y + " beats " + x
	}
	return "not distinguishable"
}

// verdictOrder ranks a verdict for x: 1 beats, 0 level, −1 beaten.
func verdictOrderOf(p sizePooled) int {
	switch {
	case math.IsNaN(p.d):
		return 0
	case p.lo > 0:
		return 1
	case p.hi < 0:
		return -1
	}
	return 0
}

func fmtPooled(p sizePooled) string {
	if math.IsNaN(p.d) {
		return "—"
	}
	return fmt.Sprintf("%+.4f [%+.4f, %+.4f]", p.d, p.lo, p.hi)
}

func writeSizeSummary(w io.Writer, units []sizeUnit) {
	var variants, methods []string
	for _, u := range units {
		for _, n := range u.names {
			if !slices.Contains(methods, n) {
				methods = append(methods, n)
			}
			if strings.HasPrefix(n, sizeVariantPrefix) && !slices.Contains(variants, n) {
				variants = append(variants, n)
			}
		}
	}

	fmt.Fprintf(w, "### Units\n\n")
	fmt.Fprintf(w, "| unit | T | T date | end date | window commits replayed | sweep scope | functions at T | calibration | union | unresolved |\n")
	fmt.Fprintf(w, "| --- | --- | --- | --- | ---: | --- | ---: | --- | ---: | ---: |\n")
	for _, u := range units {
		scope := u.c.SweepScope
		if scope == "" {
			scope = scopeWalked
		}
		fmt.Fprintf(w, "| %s | `%s` | %s | %s | %d | %s | %d | %s | %d | %d |\n", u.label(), u.c.Since[:9], u.c.SinceDate, u.c.PinDate,
			u.c.WindowCommits, scope, u.c.Functions, u.c.Calibration, u.c.Union, u.c.Unresolved)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "### M1@%d per unit\n\n", primaryK)
	fmt.Fprintf(w, "Each cell is M1@%d over the list's own top %d (n in brackets when shorter), the pairs with any event, the distinct commits behind them, and the median smaller side in lines.\n\n", primaryK, primaryK)
	fmt.Fprintf(w, "| method |")
	for _, u := range units {
		fmt.Fprintf(w, " %s |", u.label())
	}
	fmt.Fprintf(w, "\n| --- |")
	for range units {
		fmt.Fprintf(w, " ---: |")
	}
	fmt.Fprintln(w)
	for _, m := range methods {
		fmt.Fprintf(w, "| %s |", m)
		for _, u := range units {
			idx := u.top(m, primaryK)
			if len(idx) == 0 {
				fmt.Fprintf(w, " — |")
				continue
			}
			r := rateOf(u.c, idx)
			n := ""
			if r.n < primaryK {
				n = fmt.Sprintf(" (%d)", r.n)
			}
			fmt.Fprintf(w, " %.3f%s · %d · %d · %d |", r.m1, n, int(math.Round(r.e*float64(r.n))), r.commits, medianSmallSide(u.c, idx))
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)

	// The primary comparison: each variant against doppel.
	type pooledRow struct {
		x, y   string
		cells  []*sizeCell
		pooled sizePooled
	}
	compareAll := func(x, y string, depthCap, floor int) pooledRow {
		row := pooledRow{x: x, y: y, cells: make([]*sizeCell, len(units))}
		for i, u := range units {
			if c, ok := compareAt(u, x, y, depthCap); ok {
				row.cells[i] = &c
			}
		}
		row.pooled = poolCells(units, row.cells, floor)
		return row
	}
	writeRows := func(rows []pooledRow, verdictFor func(pooledRow) string) {
		fmt.Fprintf(w, "| comparison |")
		for _, u := range units {
			fmt.Fprintf(w, " %s |", u.label())
		}
		fmt.Fprintf(w, " units · corpora | pooled D [95%% CI] | verdict |\n| --- |")
		for range units {
			fmt.Fprintf(w, " --- |")
		}
		fmt.Fprintf(w, " ---: | --- | --- |\n")
		for _, r := range rows {
			fmt.Fprintf(w, "| %s − %s |", r.x, r.y)
			for _, c := range r.cells {
				if c == nil {
					fmt.Fprintf(w, " — |")
					continue
				}
				mark := ""
				switch {
				case c.lo > 0:
					mark = " ✓"
				case c.hi < 0:
					mark = " ✗"
				}
				out := ""
				if c.clusters < minEventClusters {
					out = " ·"
				}
				depth := ""
				if c.depth < primaryK {
					depth = fmt.Sprintf(" @%d", c.depth)
				}
				fmt.Fprintf(w, " %+.3f [%+.3f, %+.3f]%s (%d%s)%s |", c.d, c.lo, c.hi, mark, c.clusters, out, depth)
			}
			fmt.Fprintf(w, " %d · %d | %s | %s |\n", r.pooled.units, r.pooled.corpora, fmtPooled(r.pooled), verdictFor(r))
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "### Primary: each variant against doppel, M1@%d\n\n", primaryK)
	fmt.Fprintf(w, "Per unit: the difference at matched depth (each list's top min(nX, nY, %d)), its paired-clustered 95%% CI (%d replicates), ✓/✗ when the CI excludes 0, and in brackets the independent event clusters in the frame; · marks a unit below the floor of %d, which is not pooled; @n a depth below %d. Pooled D is the mean over corpora of the mean over each corpus's counting units, with the CI read off the replicate-wise pooled replicates.\n\n",
		primaryK, bootReps, minEventClusters, primaryK)
	var primary []pooledRow
	for _, v := range variants {
		primary = append(primary, compareAll(v, doppelMethod, primaryK, minEventClusters))
	}
	writeRows(primary, func(r pooledRow) string { return sizeVerdict(r.pooled, r.x, r.y) })

	fmt.Fprintf(w, "### The gap to the clone detectors\n\n")
	fmt.Fprintf(w, "Every method against each detector, at matched depth, under the same rule.\n\n")
	type gap struct {
		det          string
		base         pooledRow
		vs           []pooledRow
		change       []sizePooled
		changeCounts []int
	}
	var gaps []gap
	var detRows []pooledRow
	for _, x := range sizeDetectors {
		g := gap{det: x, base: compareAll(doppelMethod, x, primaryK, minEventClusters)}
		detRows = append(detRows, g.base)
		for _, v := range variants {
			r := compareAll(v, x, primaryK, minEventClusters)
			g.vs = append(g.vs, r)
			detRows = append(detRows, r)
			// The gap change: the variant against doppel at the detector's
			// matched depth, every unit where the detector lists a pair.
			ch := pooledRow{x: v, y: doppelMethod, cells: make([]*sizeCell, len(units))}
			for i, u := range units {
				if n := u.length(x); n > 0 {
					if c, ok := compareAt(u, v, doppelMethod, min(n, primaryK)); ok {
						ch.cells[i] = &c
					}
				}
			}
			ch.pooled = poolCells(units, ch.cells, 0)
			g.change = append(g.change, ch.pooled)
		}
		gaps = append(gaps, g)
	}
	writeRows(detRows, func(r pooledRow) string { return sizeVerdict(r.pooled, r.x, r.y) })

	fmt.Fprintf(w, "### Gap change per detector\n\n")
	fmt.Fprintf(w, "M1(variant) − M1(doppel) at the detector's matched depth, pooled over every unit where the detector lists a pair (no cluster floor): how far the variant moves doppel's difference with that detector. Then the two verdicts against the detector.\n\n")
	fmt.Fprintf(w, "| variant | detector | gap change [95%% CI] | doppel vs detector | variant vs detector |\n| --- | --- | --- | --- | --- |\n")
	for vi, v := range variants {
		for _, g := range gaps {
			fmt.Fprintf(w, "| %s | %s | %s | %s | %s |\n", v, g.det, fmtPooled(g.change[vi]),
				sizeVerdict(g.base.pooled, doppelMethod, g.det), sizeVerdict(g.vs[vi].pooled, v, g.det))
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "### Criteria (a) and (b)\n\n")
	fmt.Fprintf(w, "(a) the variant beats doppel: the pooled lower bound above 0. (b) the gap closes: the gap change is above 0 for at least 4 of the 5 detectors, and against no detector is the variant's verdict worse than doppel's.\n\n")
	fmt.Fprintf(w, "| variant | (a) | gap change > 0 | verdicts worse | (b) |\n| --- | --- | ---: | ---: | --- |\n")
	for vi, v := range variants {
		a := "no"
		if primary[vi].pooled.lo > 0 {
			a = "**yes**"
		}
		up, worse := 0, 0
		for _, g := range gaps {
			if g.change[vi].d > 0 {
				up++
			}
			if verdictOrderOf(g.vs[vi].pooled) < verdictOrderOf(g.base.pooled) {
				worse++
			}
		}
		b := "no"
		if up >= len(sizeDetectors)-1 && worse == 0 {
			b = "**yes**"
		}
		fmt.Fprintf(w, "| %s | %s | %d of %d | %d | %s |\n", v, a, up, len(sizeDetectors), worse, b)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "### Small-family cost\n\n")
	fmt.Fprintf(w, "Pairs in doppel's top %d whose smaller side has fewer than %d lines at T (below a clone detector's floor), and how many of them each variant's top %d drops; with an event in brackets.\n\n", primaryK, smallSideLines, primaryK)
	fmt.Fprintf(w, "| variant |")
	for _, u := range units {
		fmt.Fprintf(w, " %s |", u.label())
	}
	fmt.Fprintf(w, "\n| --- |")
	for range units {
		fmt.Fprintf(w, " --- |")
	}
	fmt.Fprintf(w, "\n| doppel: small pairs in top %d |", primaryK)
	for _, u := range units {
		n, ev := 0, 0
		for _, i := range u.top(doppelMethod, primaryK) {
			if smallPair(u.c.Pairs[i]) {
				n++
				if m1(u.c.Pairs[i]) > 0 {
					ev++
				}
			}
		}
		fmt.Fprintf(w, " %d (%d) |", n, ev)
	}
	fmt.Fprintln(w)
	for _, v := range variants {
		fmt.Fprintf(w, "| %s: dropped |", v)
		for _, u := range units {
			in := map[int]bool{}
			for _, i := range u.top(v, primaryK) {
				in[i] = true
			}
			n, ev := 0, 0
			for _, i := range u.top(doppelMethod, primaryK) {
				if smallPair(u.c.Pairs[i]) && !in[i] {
					n++
					if m1(u.c.Pairs[i]) > 0 {
						ev++
					}
				}
			}
			fmt.Fprintf(w, " %d (%d) |", n, ev)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "### Overlap of each top %d with doppel's (descriptive)\n\n", primaryK)
	fmt.Fprintf(w, "| method |")
	for _, u := range units {
		fmt.Fprintf(w, " %s |", u.label())
	}
	fmt.Fprintf(w, "\n| --- |")
	for range units {
		fmt.Fprintf(w, " ---: |")
	}
	fmt.Fprintln(w)
	for _, m := range methods {
		if m == doppelMethod {
			continue
		}
		fmt.Fprintf(w, "| %s |", m)
		for _, u := range units {
			dop := map[int]bool{}
			for _, i := range u.top(doppelMethod, primaryK) {
				dop[i] = true
			}
			n := 0
			for _, i := range u.top(m, primaryK) {
				if dop[i] {
					n++
				}
			}
			fmt.Fprintf(w, " %d |", n)
		}
		fmt.Fprintln(w)
	}
}

func smallPair(p studyPair) bool { return min(p.LinesA, p.LinesB) < smallSideLines }

// medianSmallSide is the median of the smaller side's lines over idx.
func medianSmallSide(c compareOut, idx []int) int {
	v := make([]int, len(idx))
	for i, x := range idx {
		v[i] = min(c.Pairs[x].LinesA, c.Pairs[x].LinesB)
	}
	sort.Ints(v)
	return v[len(v)/2]
}

package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// TestCalleeDrift measures two candidate drift signals before either becomes
// a kind, and asserts nothing.
//
//	DOPPEL_BENCH_CALLEEDRIFT=1 go test ./internal/bench/ -v -run TestCalleeDrift
//
// DOPPEL_BENCH_CALLEEDRIFT_EXTRA is an OS path list of further corpus roots;
// doppel's own tree is always measured.
//
// Substitution. Two bodies that are alike but call different things read today
// as "different calls" — the shape is a scaffold, the work is elsewhere. The
// other reading is drift: A routes through oldRetry, B through newRetry, and
// the two callees are themselves near-duplicates. For every scored pair whose
// code-shape clears comparator.MergeShapeFloor, the resolved internal callees
// only one side calls are matched greedily across the pair by code-shape; a
// match at or above the substitution floor is a substitution. The question is
// how often substitution explains a pair's callee gap, and in particular how
// often it explains a pair analyzer.DifferentCalls labels. Unresolved and
// external calls are out of reach (no body to compare) and are not counted.
//
// Locality is reported beside it, never folded in: the Jaccard of the two
// callers' depth-2 call-graph balls (each with its own identity added, the
// query locality rule), and whether they share a package. A drift finding
// is only actionable where the two callers are near each other; far apart it
// is a parallel implementation.
//
// Caller split. For every scored pair of near-duplicate bodies (code-shape at
// or above analyzer.ForkShapeFloor) with resolved callers on both sides, the
// corpus is using two versions of one thing. The census reports how many such
// pairs exist, how lopsided the split is (minority share of callers), how
// many callers call both, and whether the two caller populations live in the
// same packages.
func TestCalleeDrift(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_CALLEEDRIFT") != "1" {
		t.Skip("set DOPPEL_BENCH_CALLEEDRIFT=1 to measure callee drift")
	}
	type target struct{ name, root string }
	var targets []target
	for _, c := range Corpora {
		if !Present(c) {
			continue
		}
		root, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		targets = append(targets, target{c.Name, root})
	}
	targets = append(targets, target{"doppel", repoRoot(t)})
	if extra := os.Getenv("DOPPEL_BENCH_CALLEEDRIFT_EXTRA"); extra != "" {
		for _, root := range filepath.SplitList(extra) {
			targets = append(targets, target{filepath.Base(root), root})
		}
	}
	labels := committedLabels(t)

	for _, tg := range targets {
		pop := PopExclude
		lf, labeled := labels[tg.name]
		if labeled && lf.Population != "" {
			pop = Population(lf.Population)
		}
		units, err := Load(tg.root, pop)
		if err != nil || len(units) == 0 {
			t.Logf("[%s] load: %v (%d units)", tg.name, err, len(units))
			continue
		}
		run := Analyze(units, retriever.DefaultOptions())
		d := newDriftCorpus(run)
		d.substitution(t, tg.name, lf, labeled)
		d.callerSplit(t, tg.name)
		d.subsystemKind(t, tg.name, lf, labeled)
	}
}

// substitutionFloors are the code-shape values a callee pair is tried at.
// The middle one is the reported operating point: analyzer.ForkShapeFloor,
// the family edge cut, so "these two callees are alike" means what it means
// everywhere else in the tool.
var substitutionFloors = []float64{0.40, analyzer.ForkShapeFloor, 0.80}

type driftCorpus struct {
	run    *Run
	index  map[string]int // qualified name → unit index, ambiguous names absent
	pairs  map[[2]int]bool
	balls  map[int]map[string]bool
	scored map[[2]int]float64
}

func newDriftCorpus(run *Run) *driftCorpus {
	d := &driftCorpus{
		run:    run,
		index:  make(map[string]int, len(run.Units)),
		pairs:  make(map[[2]int]bool, len(run.Pairs)),
		balls:  map[int]map[string]bool{},
		scored: map[[2]int]float64{},
	}
	seen := map[string]int{}
	for i, u := range run.Units {
		seen[concepter.QualifiedName(u)]++
		d.index[concepter.QualifiedName(u)] = i
	}
	for k, n := range seen {
		if n > 1 {
			delete(d.index, k)
		}
	}
	for _, p := range run.Pairs {
		d.pairs[orderedIdx(p.AIdx, p.BIdx)] = true
	}
	return d
}

func orderedIdx(a, b int) [2]int {
	if a > b {
		a, b = b, a
	}
	return [2]int{a, b}
}

// shape is the production code-shape of two units, memoized.
func (d *driftCorpus) shape(a, b int) float64 {
	k := orderedIdx(a, b)
	if s, ok := d.scored[k]; ok {
		return s
	}
	s := fingerprint.Similarity(d.run.Units[a].Fingerprint, d.run.Units[b].Fingerprint, d.run.WL).Score
	d.scored[k] = s
	return s
}

// ball is a unit's depth-2 call-graph ball with its own identity added.
func (d *driftCorpus) ball(i int) map[string]bool {
	if b, ok := d.balls[i]; ok {
		return b
	}
	qn := concepter.QualifiedName(d.run.Units[i])
	b := map[string]bool{qn: true}
	for _, n := range d.run.Graph.Neighborhood(qn) {
		b[n] = true
	}
	d.balls[i] = b
	return b
}

// locality is the Jaccard of two units' balls. Two units with no edges at all
// share only nothing and read 0.
func (d *driftCorpus) locality(a, b int) float64 {
	ba, bb := d.ball(a), d.ball(b)
	inter := 0
	for n := range ba {
		if bb[n] {
			inter++
		}
	}
	union := len(ba) + len(bb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func (d *driftCorpus) callees(i int) []int {
	var out []int
	for _, q := range d.run.Graph.Callees[concepter.QualifiedName(d.run.Units[i])] {
		if j, ok := d.index[q]; ok {
			out = append(out, j)
		}
	}
	return out
}

func (d *driftCorpus) callers(i int) []int {
	var out []int
	for _, q := range d.run.Graph.Callers[concepter.QualifiedName(d.run.Units[i])] {
		if j, ok := d.index[q]; ok {
			out = append(out, j)
		}
	}
	return out
}

// subMatch is one substituted callee pair.
type subMatch struct {
	a, b      int
	shape     float64
	retrieved bool
}

// substitutions matches the callees only a calls against those only b calls,
// greedily by code-shape (desc, then indices), keeping matches at floor.
func (d *driftCorpus) substitutions(a, b int, floor float64) (onlyA, onlyB int, subs []subMatch) {
	ca, cb := d.callees(a), d.callees(b)
	inB := map[int]bool{}
	for _, c := range cb {
		inB[c] = true
	}
	inA := map[int]bool{}
	for _, c := range ca {
		inA[c] = true
	}
	var xa, xb []int
	for _, c := range ca {
		if !inB[c] {
			xa = append(xa, c)
		}
	}
	for _, c := range cb {
		if !inA[c] {
			xb = append(xb, c)
		}
	}
	type cand struct {
		x, y int
		s    float64
	}
	var cs []cand
	for _, x := range xa {
		for _, y := range xb {
			if x == y || x == a || x == b || y == a || y == b {
				continue
			}
			if s := d.shape(x, y); s >= floor {
				cs = append(cs, cand{x, y, s})
			}
		}
	}
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].s != cs[j].s {
			return cs[i].s > cs[j].s
		}
		if cs[i].x != cs[j].x {
			return cs[i].x < cs[j].x
		}
		return cs[i].y < cs[j].y
	})
	usedX, usedY := map[int]bool{}, map[int]bool{}
	for _, c := range cs {
		if usedX[c.x] || usedY[c.y] {
			continue
		}
		usedX[c.x], usedY[c.y] = true, true
		subs = append(subs, subMatch{c.x, c.y, c.s, d.pairs[orderedIdx(c.x, c.y)]})
	}
	return len(xa), len(xb), subs
}

// driftClass is a pair's substitution reading.
func driftClass(onlyA, onlyB int, subs []subMatch) string {
	switch {
	case onlyA+onlyB == 0:
		return "no gap"
	case len(subs) == 0:
		return "unexplained"
	case 2*len(subs) == onlyA+onlyB:
		return "full sub"
	default:
		return "partial sub"
	}
}

var driftClasses = []string{"no gap", "full sub", "partial sub", "unexplained"}

func (d *driftCorpus) substitution(t *testing.T, name string, lf LabelsFile, labeled bool) {
	run := d.run
	type row struct {
		p            analyzer.SimilarPair
		onlyA, onlyB int
		subs         []subMatch
		class        string
		loc          float64
		samePkg      bool
		different    bool
	}
	var eligible, withCallees int
	byFloor := make([]map[string]int, len(substitutionFloors))
	diffByFloor := make([]map[string]int, len(substitutionFloors))
	var opRows []row
	for fi, floor := range substitutionFloors {
		byFloor[fi], diffByFloor[fi] = map[string]int{}, map[string]int{}
		for _, p := range run.Pairs {
			if p.Score < 0.4 {
				continue
			}
			if fi == 0 {
				eligible++
			}
			ua, ub := run.Units[p.AIdx], run.Units[p.BIdx]
			if len(d.callees(p.AIdx))+len(d.callees(p.BIdx)) == 0 {
				continue
			}
			if fi == 0 {
				withCallees++
			}
			oa, ob, subs := d.substitutions(p.AIdx, p.BIdx, floor)
			cl := driftClass(oa, ob, subs)
			diff := analyzer.DifferentCalls(ua, ub) != nil
			byFloor[fi][cl]++
			if diff {
				diffByFloor[fi][cl]++
				diffByFloor[fi]["total"]++
			}
			if floor == analyzer.ForkShapeFloor {
				opRows = append(opRows, row{p, oa, ob, subs, cl, d.locality(p.AIdx, p.BIdx), ua.Package == ub.Package && ua.Package != "", diff})
			}
		}
	}
	t.Logf("[%s] %5d funcs  %6d pairs  %5d at code-shape >= 0.40  %5d with a resolved internal callee", name, len(run.Units), len(run.Pairs), eligible, withCallees)
	for fi, floor := range substitutionFloors {
		parts := make([]string, 0, len(driftClasses))
		dparts := make([]string, 0, len(driftClasses))
		for _, cl := range driftClasses {
			parts = append(parts, fmt.Sprintf("%s %d", cl, byFloor[fi][cl]))
			dparts = append(dparts, fmt.Sprintf("%s %d", cl, diffByFloor[fi][cl]))
		}
		t.Logf("[%s]   sub floor %.2f  all: %s", name, floor, strings.Join(parts, ", "))
		t.Logf("[%s]                    of %d 'different calls': %s", name, diffByFloor[fi]["total"], strings.Join(dparts, ", "))
	}

	// Locality and retrieval at the operating floor, per class.
	for _, cl := range driftClasses {
		var locs []float64
		same, retrieved, subs := 0, 0, 0
		for _, r := range opRows {
			if r.class != cl {
				continue
			}
			locs = append(locs, r.loc)
			if r.samePkg {
				same++
			}
			for _, s := range r.subs {
				subs++
				if s.retrieved {
					retrieved++
				}
			}
		}
		if len(locs) == 0 {
			continue
		}
		sort.Float64s(locs)
		line := fmt.Sprintf("[%s]   %-12s n %5d  same pkg %5.1f%%  locality p25 %.2f p50 %.2f p75 %.2f  zero %5.1f%%",
			name, cl, len(locs), pct(same, len(locs)), quantile(locs, 0.25), quantile(locs, 0.5), quantile(locs, 0.75), pct(countZero(locs), len(locs)))
		if subs > 0 {
			line += fmt.Sprintf("  callee pairs %d, retrieved %.0f%%", subs, pct(retrieved, subs))
		}
		t.Logf("%s", line)
	}

	// The strongest substitution pairs, so the class can be read by eye.
	var subRows []row
	subKinds := map[string]int{}
	for _, r := range opRows {
		if len(r.subs) > 0 {
			subRows = append(subRows, r)
			subKinds[pairKindName(run, r.p)]++
		}
	}
	if len(subRows) > 0 {
		t.Logf("[%s]   substitution pairs by existing kind: %s", name, kindTally(subKinds))
	}
	sort.Slice(subRows, func(i, j int) bool {
		ki, kj := RankKey(run, subRows[i].p), RankKey(run, subRows[j].p)
		if ki != kj {
			return ki > kj
		}
		return orderedLess(subRows[i].p, subRows[j].p)
	})
	for i, r := range subRows {
		if i == 12 {
			break
		}
		var ss []string
		for _, s := range r.subs {
			ret := ""
			if !s.retrieved {
				ret = " unretrieved"
			}
			ss = append(ss, fmt.Sprintf("%s ~ %s (%.2f%s)", qualifiedName(run.Units[s.a]), qualifiedName(run.Units[s.b]), s.shape, ret))
		}
		flag := ""
		if r.different {
			flag = "  [different calls]"
		}
		t.Logf("[%s]     %.2f  loc %.2f  %s / %s%s  {%s}", name, r.p.Score, r.loc, qualifiedName(run.Units[r.p.AIdx]), qualifiedName(run.Units[r.p.BIdx]), flag, pairKindName(run, r.p))
		t.Logf("[%s]            %s  (gap %d+%d)", name, strings.Join(ss, "; "), r.onlyA, r.onlyB)
	}

	if !labeled {
		return
	}
	byKey := map[string]row{}
	for _, r := range opRows {
		byKey[pairKey(qualifiedName(run.Units[r.p.AIdx]), qualifiedName(run.Units[r.p.BIdx]))] = r
	}
	tally := map[string]map[string]int{}
	for _, l := range lf.Labels {
		cl := "absent"
		if r, ok := byKey[pairKey(l.A, l.B)]; ok {
			cl = r.class
		}
		if tally[l.Class] == nil {
			tally[l.Class] = map[string]int{}
		}
		tally[l.Class][cl]++
	}
	classes := make([]string, 0, len(tally))
	for c := range tally {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	for _, c := range classes {
		parts := []string{}
		for _, cl := range append(append([]string{}, driftClasses...), "absent") {
			if n := tally[c][cl]; n > 0 {
				parts = append(parts, fmt.Sprintf("%s %d", cl, n))
			}
		}
		t.Logf("[%s]   label %-15s %s", name, c, strings.Join(parts, ", "))
	}
}

// pairKindName is the naming or mirror kind a pair already carries, or "-".
// The lens kinds that read context (thin wrappers, different calls) are left
// out: different calls is what substitution is measured against.
func pairKindName(run *Run, p analyzer.SimilarPair) string {
	a, b := run.Units[p.AIdx], run.Units[p.BIdx]
	if k := analyzer.ClassifyPairWith(a, b, p.Score, analyzer.ForkShapeFloor); k != nil {
		return k.Kind
	}
	if k := analyzer.Mirror(a, b); k != nil {
		return k.Kind
	}
	return "-"
}

func kindTally(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", k, m[k]))
	}
	return strings.Join(parts, ", ")
}

func countZero(sorted []float64) int {
	n := 0
	for _, x := range sorted {
		if x == 0 {
			n++
		}
	}
	return n
}

func orderedLess(a, b analyzer.SimilarPair) bool {
	if a.AIdx != b.AIdx {
		return a.AIdx < b.AIdx
	}
	return a.BIdx < b.BIdx
}

func (d *driftCorpus) callerSplit(t *testing.T, name string) {
	run := d.run
	type split struct {
		p                 analyzer.SimilarPair
		na, nb, both      int
		minority, pkgJacc float64
		samePkg           bool
		kind              string
	}
	var near, bothUsed, oneUsed, neither int
	var rows []split
	for _, p := range run.Pairs {
		if p.Score < analyzer.ForkShapeFloor {
			continue
		}
		near++
		ca, cb := d.callers(p.AIdx), d.callers(p.BIdx)
		switch {
		case len(ca) == 0 && len(cb) == 0:
			neither++
			continue
		case len(ca) == 0 || len(cb) == 0:
			oneUsed++
			continue
		}
		bothUsed++
		setA := map[int]bool{}
		pa, pb := map[string]bool{}, map[string]bool{}
		for _, c := range ca {
			setA[c] = true
			pa[run.Units[c].Package] = true
		}
		both := 0
		for _, c := range cb {
			if setA[c] {
				both++
			}
			pb[run.Units[c].Package] = true
		}
		pi := 0
		for k := range pa {
			if pb[k] {
				pi++
			}
		}
		na, nb := len(ca), len(cb)
		mn := na
		if nb < mn {
			mn = nb
		}
		rows = append(rows, split{
			p: p, na: na, nb: nb, both: both,
			minority: float64(mn) / float64(na+nb),
			pkgJacc:  float64(pi) / float64(len(pa)+len(pb)-pi),
			samePkg:  run.Units[p.AIdx].Package == run.Units[p.BIdx].Package,
			kind:     pairKindName(run, p),
		})
	}
	t.Logf("[%s]   caller split: %d pairs at code-shape >= %.2f — both sides called %d, one side %d, neither %d",
		name, near, analyzer.ForkShapeFloor, bothUsed, oneUsed, neither)
	if len(rows) == 0 {
		return
	}
	var mins, pjs []float64
	shared, disjointPkgs, lopsided := 0, 0, 0
	for _, r := range rows {
		mins = append(mins, r.minority)
		pjs = append(pjs, r.pkgJacc)
		if r.both > 0 {
			shared++
		}
		if r.pkgJacc == 0 {
			disjointPkgs++
		}
		if r.minority <= 0.2 {
			lopsided++
		}
	}
	sort.Float64s(mins)
	sort.Float64s(pjs)
	t.Logf("[%s]     minority share p25 %.2f p50 %.2f p75 %.2f  lopsided (<= 0.20) %d  a caller calls both %d  caller pkgs disjoint %d  caller-pkg jaccard p50 %.2f",
		name, quantile(mins, 0.25), quantile(mins, 0.5), quantile(mins, 0.75), lopsided, shared, disjointPkgs, quantile(pjs, 0.5))

	// By locality of the two definitions, and by the kind each already carries.
	for _, same := range []bool{true, false} {
		kinds := map[string]int{}
		n, lop, disj := 0, 0, 0
		for _, r := range rows {
			if r.samePkg != same {
				continue
			}
			n++
			kinds[r.kind]++
			if r.minority <= 0.2 {
				lop++
			}
			if r.pkgJacc == 0 {
				disj++
			}
		}
		where := "same package"
		if !same {
			where = "cross package"
		}
		t.Logf("[%s]     %-13s %4d  lopsided %3d  caller pkgs disjoint %3d  kinds: %s", name, where, n, lop, disj, kindTally(kinds))
	}

	// The largest splits by total callers, minority share >= 0.1 so one stray
	// caller does not lead the list, unlabelled by any kind, per locality.
	sort.Slice(rows, func(i, j int) bool {
		ti, tj := rows[i].na+rows[i].nb, rows[j].na+rows[j].nb
		if ti != tj {
			return ti > tj
		}
		return orderedLess(rows[i].p, rows[j].p)
	})
	for _, same := range []bool{true, false} {
		shown := 0
		for _, r := range rows {
			if r.samePkg != same || r.minority < 0.1 || r.kind != "-" {
				continue
			}
			if shown == 8 {
				break
			}
			shown++
			where := "same "
			if !same {
				where = "cross"
			}
			t.Logf("[%s]     %s %.2f  callers %3d / %-3d both %2d  pkg-jacc %.2f  %s / %s",
				name, where, r.p.Score, r.na, r.nb, r.both, r.pkgJacc, qualifiedName(run.Units[r.p.AIdx]), qualifiedName(run.Units[r.p.BIdx]))
		}
	}
}

// subsystemKind counts where the production rule, analyzer.SubsystemCopies,
// fires over the compared pairs at the default fork floor, what kind those
// pairs carried before it existed, and which labels it lands on.
func (d *driftCorpus) subsystemKind(t *testing.T, name string, lf LabelsFile, labeled bool) {
	run := d.run
	ctx := func(p analyzer.SimilarPair) analyzer.PairContext {
		da, db := run.Docs[p.AIdx], run.Docs[p.BIdx]
		return analyzer.PairContext{
			ResolvedA: da.ResolvedCallees, ResolvedB: db.ResolvedCallees,
			CallersA: da.Callers, CallersB: db.Callers,
			CallerPkgsA: da.CallerPackages, CallerPkgsB: db.CallerPackages,
		}
	}
	var hits []analyzer.SimilarPair
	displaced := map[string]int{}
	for _, p := range run.Pairs {
		a, b := run.Units[p.AIdx], run.Units[p.BIdx]
		if analyzer.SubsystemCopies(a, b, p.Score, analyzer.ForkShapeFloor, ctx(p)) == nil {
			continue
		}
		hits = append(hits, p)
		// What the pair would read without the new rule: the naming kinds
		// win over it by precedence, so only the lens kinds can be displaced.
		prev := "-"
		if k := analyzer.Mirror(a, b); k != nil {
			prev = k.Kind
		} else if k := analyzer.DifferentCalls(a, b); k != nil {
			prev = k.Kind
		}
		if analyzer.ClassifyPairWith(a, b, p.Score, analyzer.ForkShapeFloor) != nil {
			prev = "(naming kind wins)"
		}
		displaced[prev]++
	}
	sort.Slice(hits, func(i, j int) bool {
		ki, kj := RankKey(run, hits[i]), RankKey(run, hits[j])
		if ki != kj {
			return ki > kj
		}
		return orderedLess(hits[i], hits[j])
	})
	t.Logf("[%s]   subsystem copies: %d of %d pairs (previously: %s)", name, len(hits), len(run.Pairs), kindTally(displaced))
	// Reach: a side is shared when anything outside its own package calls it.
	// Both local is each package keeping its own private helper — parallel
	// implementations; one shared is a local copy beside a helper the rest of
	// the corpus already uses.
	reach := map[string]int{}
	shared := func(u parser.CodeUnit, pkgs []string) bool {
		for _, p := range pkgs {
			if p != u.Package {
				return true
			}
		}
		return false
	}
	reachOf := func(p analyzer.SimilarPair) string {
		sa := shared(run.Units[p.AIdx], run.Docs[p.AIdx].CallerPackages)
		sb := shared(run.Units[p.BIdx], run.Docs[p.BIdx].CallerPackages)
		switch {
		case sa && sb:
			return "both shared"
		case sa || sb:
			return "one shared"
		}
		return "both local"
	}
	for _, p := range hits {
		reach[reachOf(p)]++
	}
	t.Logf("[%s]   subsystem copies by reach: %s", name, kindTally(reach))
	rank := map[[2]int]int{}
	ranked := append([]analyzer.SimilarPair(nil), run.Pairs...)
	ranked, _ = analyzer.SortForReport(ranked, run.Units, 0, 0)
	for i, p := range ranked {
		rank[orderedIdx(p.AIdx, p.BIdx)] = i + 1
	}
	shown := 0
	for _, p := range hits {
		if shown == 10 {
			break
		}
		if reachOf(p) == "both local" {
			continue
		}
		shown++
		k := analyzer.SubsystemCopies(run.Units[p.AIdx], run.Units[p.BIdx], p.Score, analyzer.ForkShapeFloor, ctx(p))
		t.Logf("[%s]     rank %5d  %.2f  %-11s %s (%d in %s) / %s (%d in %s)", name, rank[orderedIdx(p.AIdx, p.BIdx)], p.Score, reachOf(p),
			k.Names[0], k.CallerCounts[0], strings.Join(k.CallerPackages[0], ","),
			k.Names[1], k.CallerCounts[1], strings.Join(k.CallerPackages[1], ","))
	}
	if !labeled {
		return
	}
	hit := map[string]bool{}
	for _, p := range hits {
		hit[pairKey(qualifiedName(run.Units[p.AIdx]), qualifiedName(run.Units[p.BIdx]))] = true
	}
	tally := map[string]int{}
	for _, l := range lf.Labels {
		if hit[pairKey(l.A, l.B)] {
			tally[l.Class]++
			t.Logf("[%s]     labelled %s: %s / %s", name, l.Class, l.A, l.B)
		}
	}
	t.Logf("[%s]   subsystem copies on labels: %s", name, kindTally(tally))
}

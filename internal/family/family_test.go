package family

import (
	"reflect"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

// units builds n functions. Every fingerprint is distinct, so no edge is ever
// completed unless a test asks for one by giving two units the same shape —
// the zero fingerprint would work too, but a distinct one proves completion is
// scoring rather than defaulting.
func units(n int) []parser.CodeUnit {
	out := make([]parser.CodeUnit, n)
	for i := range out {
		out[i] = parser.CodeUnit{
			Name:        string(rune('A' + i)),
			Package:     "p",
			File:        "p/a.go",
			Fingerprint: shape(uint64(i * 1000)),
		}
	}
	return out
}

// shape is a fingerprint whose content is decided by seed: two units built
// with the same seed score 1.0 against each other, two with different seeds
// share no shingles and no types.
func shape(seed uint64) fingerprint.Fingerprint {
	sh := make([]uint64, 0, 8)
	for i := uint64(0); i < 8; i++ {
		sh = append(sh, seed+i)
	}
	flow := make([]int, len(fingerprint.FlowLabels))
	flow[0] = 1
	return fingerprint.Fingerprint{
		Shingles: sh,
		Flow:     flow,
		Types:    []string{string(rune('a' + seed%26))},
		Nodes:    40,
	}
}

func pair(a, b int, score float64) analyzer.SimilarPair {
	return analyzer.SimilarPair{AIdx: a, BIdx: b, Score: score}
}

func members(f Family) []int { return f.Members }

// The whole design rests on this. A~B at 0.7 and B~C at 0.7 says nothing about
// A~C, and single-linkage clustering would chain them into a "family" whose
// two ends have nothing in common — a claim no reader could check.
func TestChainingIsNotAFamily(t *testing.T) {
	u := units(3)
	pairs := []analyzer.SimilarPair{pair(0, 1, 0.70), pair(1, 2, 0.70)}

	fams, stats := Build(u, pairs, DefaultOptions())

	if len(fams) != 0 {
		t.Errorf("chained pairs formed %d families, want none: %+v", len(fams), fams)
	}
	if stats.Components != 1 {
		t.Errorf("Components = %d, want 1 — the component exists, it just holds no clique", stats.Components)
	}
	if stats.Members != 0 {
		t.Errorf("Members = %d, want 0", stats.Members)
	}
}

// Retrieval keeps a bounded top-K per function, so an edge can be missing
// because it fell out of a budget rather than because the two are unalike.
// Completion is what stops that gap from splitting a real family.
func TestCompletionRecoversTheMissingEdge(t *testing.T) {
	u := units(3)
	// All three are the same shape; retrieval only ever proposed two of the
	// three edges.
	for i := range u {
		u[i].Fingerprint = shape(7)
	}
	pairs := []analyzer.SimilarPair{pair(0, 1, 1.0), pair(1, 2, 1.0)}

	fams, stats := Build(u, pairs, DefaultOptions())

	if len(fams) != 1 {
		t.Fatalf("got %d families, want 1: %+v", len(fams), fams)
	}
	if got := members(fams[0]); !reflect.DeepEqual(got, []int{0, 1, 2}) {
		t.Errorf("members = %v, want [0 1 2]", got)
	}
	if fams[0].Completed != 1 {
		t.Errorf("Completed = %d, want 1 — the 0-2 edge was scored here", fams[0].Completed)
	}
	if stats.Completed != 1 {
		t.Errorf("Stats.Completed = %d, want 1", stats.Completed)
	}
	if fams[0].MinEdge != 1.0 {
		t.Errorf("MinEdge = %.2f, want 1.00", fams[0].MinEdge)
	}
}

// Completion must score, not assume. Two units that were never paired and are
// genuinely unalike stay unconnected, which is the same case as
// TestChainingIsNotAFamily seen from the completion side.
func TestCompletionDoesNotInventEdges(t *testing.T) {
	u := units(3) // three distinct shapes
	pairs := []analyzer.SimilarPair{pair(0, 1, 0.90), pair(1, 2, 0.90)}

	fams, stats := Build(u, pairs, DefaultOptions())

	if len(fams) != 0 {
		t.Errorf("unalike units were joined into %d families: %+v", len(fams), fams)
	}
	if stats.Completed != 0 {
		t.Errorf("Stats.Completed = %d, want 0", stats.Completed)
	}
}

// A function can sit in more than one maximal clique, and choosing between
// them would be a judgement the tool cannot justify. Both are reported, and
// the shared member is counted once.
func TestOverlappingFamiliesAreBothReported(t *testing.T) {
	// {0,1,2} and {2,3,4} are complete; nothing else is.
	u := units(5)
	pairs := []analyzer.SimilarPair{
		pair(0, 1, 0.90), pair(0, 2, 0.90), pair(1, 2, 0.90),
		pair(2, 3, 0.80), pair(2, 4, 0.80), pair(3, 4, 0.80),
	}

	fams, stats := Build(u, pairs, DefaultOptions())

	if len(fams) != 2 {
		t.Fatalf("got %d families, want 2: %+v", len(fams), fams)
	}
	// Equal size, so the tighter one leads.
	if !reflect.DeepEqual(members(fams[0]), []int{0, 1, 2}) {
		t.Errorf("families[0] = %v, want [0 1 2] (higher MinEdge first)", members(fams[0]))
	}
	if !reflect.DeepEqual(members(fams[1]), []int{2, 3, 4}) {
		t.Errorf("families[1] = %v, want [2 3 4]", members(fams[1]))
	}
	if stats.Members != 5 {
		t.Errorf("Members = %d, want 5 — the shared member counts once", stats.Members)
	}
}

// The repo's invariant is that an unchanged tree produces byte-identical
// output. Map iteration is the usual way that breaks, and this package is full
// of maps.
func TestBuildIsDeterministic(t *testing.T) {
	u := units(6)
	for i := range u {
		u[i].Fingerprint = shape(uint64(i / 3)) // two groups of three
	}
	var pairs []analyzer.SimilarPair
	for _, p := range [][2]int{{0, 1}, {1, 2}, {0, 2}, {3, 4}, {4, 5}, {3, 5}} {
		pairs = append(pairs, pair(p[0], p[1], 1.0))
	}

	want, wantStats := Build(u, pairs, DefaultOptions())
	for i := 0; i < 20; i++ {
		// Reverse the input each round: Build must not depend on pair order,
		// which SortForReport changes in place on the real slice.
		rev := make([]analyzer.SimilarPair, len(pairs))
		for j := range pairs {
			rev[j] = pairs[len(pairs)-1-j]
		}
		got, gotStats := Build(u, rev, DefaultOptions())
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d differed:\n got %+v\nwant %+v", i, got, want)
		}
		if !reflect.DeepEqual(gotStats, wantStats) {
			t.Fatalf("run %d stats differed:\n got %+v\nwant %+v", i, gotStats, wantStats)
		}
	}
}

func TestBuildDoesNotMutateItsInput(t *testing.T) {
	u := units(3)
	for i := range u {
		u[i].Fingerprint = shape(7)
	}
	pairs := []analyzer.SimilarPair{pair(0, 1, 1.0), pair(1, 2, 1.0)}
	before := append([]analyzer.SimilarPair(nil), pairs...)

	Build(u, pairs, DefaultOptions())

	if !reflect.DeepEqual(pairs, before) {
		t.Error("Build reordered or rewrote its input pair slice")
	}
}

// A guard that silently drops work reads as "there was nothing here". The
// component is counted and its size recorded so the caller can say so.
func TestOversizedComponentIsSkippedAndCounted(t *testing.T) {
	u := units(6)
	for i := range u {
		u[i].Fingerprint = shape(7)
	}
	var pairs []analyzer.SimilarPair
	for i := 0; i < 6; i++ {
		for j := i + 1; j < 6; j++ {
			pairs = append(pairs, pair(i, j, 1.0))
		}
	}
	o := DefaultOptions()
	o.MaxComponent = 5

	fams, stats := Build(u, pairs, o)

	if len(fams) != 0 {
		t.Errorf("got %d families past the component guard", len(fams))
	}
	if !reflect.DeepEqual(stats.Skipped, []int{6}) {
		t.Errorf("Skipped = %v, want [6]", stats.Skipped)
	}
	if stats.Components != 1 {
		t.Errorf("Components = %d, want 1", stats.Components)
	}
}

// The search budget is the second guard, for a component small enough to
// complete but dense enough to be pathological.
func TestSearchBudgetSkipsRatherThanTruncates(t *testing.T) {
	u := units(6)
	for i := range u {
		u[i].Fingerprint = shape(7)
	}
	var pairs []analyzer.SimilarPair
	for i := 0; i < 6; i++ {
		for j := i + 1; j < 6; j++ {
			pairs = append(pairs, pair(i, j, 1.0))
		}
	}
	o := DefaultOptions()
	o.MaxSearch = 1

	fams, stats := Build(u, pairs, o)

	if len(fams) != 0 {
		t.Errorf("a partial enumeration was reported as the answer: %+v", fams)
	}
	if len(stats.Skipped) != 1 {
		t.Errorf("Skipped = %v, want one entry", stats.Skipped)
	}
}

// Pairs below the cut are not edges, so a family is never assembled out of
// resemblances the report would not show.
func TestEdgesBelowTheCutAreIgnored(t *testing.T) {
	u := units(3)
	for i := range u {
		u[i].Fingerprint = shape(7)
	}
	pairs := []analyzer.SimilarPair{pair(0, 1, 0.30), pair(1, 2, 0.30)}
	o := DefaultOptions()
	o.Min = 0.95

	// Completion is per component, and with no edge at the cut there is no
	// component to complete: three identical functions nobody retrieved above
	// 0.95 stay invisible.
	if fams, stats := Build(u, pairs, o); len(fams) != 0 || stats.Components != 0 {
		t.Errorf("got %d families in %d components, want 0 in 0", len(fams), stats.Components)
	}
}

// The census orders by size first: a six-member family is a bigger fact about
// the corpus than a tighter three-member one.
func TestFamiliesOrderBySizeThenTightness(t *testing.T) {
	u := units(7)
	for i := range u {
		if i < 4 {
			u[i].Fingerprint = shape(1)
		} else {
			u[i].Fingerprint = shape(2)
		}
	}
	var pairs []analyzer.SimilarPair
	for i := 0; i < 4; i++ {
		for j := i + 1; j < 4; j++ {
			pairs = append(pairs, pair(i, j, 0.70))
		}
	}
	for i := 4; i < 7; i++ {
		for j := i + 1; j < 7; j++ {
			pairs = append(pairs, pair(i, j, 0.99))
		}
	}

	fams, _ := Build(u, pairs, DefaultOptions())

	if len(fams) != 2 {
		t.Fatalf("got %d families, want 2", len(fams))
	}
	if len(fams[0].Members) != 4 {
		t.Errorf("the tighter but smaller family led: %+v", fams)
	}
}

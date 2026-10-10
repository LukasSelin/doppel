package main

import (
	"bytes"
	"fmt"
	"math"
	"testing"
)

// sizeFixture is one unit: pairs 0..n-1, pair i carrying M1 = 1 and its own
// evidence commit when events[i], and lists naming pair indexes.
func sizeFixture(corpus string, n int, events map[int]bool, lists map[string][]int) sizeUnit {
	c := compareOut{Corpus: corpus, Since: "0123456789abcdef", SinceDate: "2020-01-01", PinDate: "2022-01-01"}
	for i := range n {
		p := studyPair{A: fmt.Sprintf("p.A%d", i), B: fmt.Sprintf("p.B%d", i), EditsA: 1, EditsB: 1, LinesA: 10 + i, LinesB: 30}
		if events[i] {
			p.Cochanges = 1
			p.Evidence = []evidence{{Kind: "co-change", Commits: []string{fmt.Sprintf("c%04d", i)}}}
		}
		c.Pairs = append(c.Pairs, p)
	}
	u := sizeUnit{c: c, k: 1, views: map[string]methodView{}}
	for _, name := range []string{doppelMethod, "variant: X", "dupl (t=50)"} {
		l, ok := lists[name]
		if !ok {
			continue
		}
		c.Lists = append(c.Lists, outcomeList{Method: name, Pairs: l})
		u.views[name] = methodView{name: name, pairs: [][]int{l}}
		u.names = append(u.names, name)
	}
	u.c = c
	return u
}

func seq(lo, hi int) []int {
	var out []int
	for i := lo; i < hi; i++ {
		out = append(out, i)
	}
	return out
}

// TestCompareAtMatchedDepth: a list of two is compared with the other list's
// top two, never its top hundred, and identical lists read exactly 0.
func TestCompareAtMatchedDepth(t *testing.T) {
	ev := map[int]bool{0: true, 50: true, 60: true, 70: true}
	u := sizeFixture("x", 200, ev, map[string][]int{
		doppelMethod:  seq(0, 100),
		"variant: X":  seq(0, 100),
		"dupl (t=50)": {150, 151},
	})
	c, ok := compareAt(u, doppelMethod, "dupl (t=50)", primaryK)
	if !ok || c.depth != 2 {
		t.Fatalf("depth %d (ok %v), want 2", c.depth, ok)
	}
	// doppel's top 2 holds pair 0 (M1 1) and pair 1 (0): 0.5 against 0.
	if c.d != 0.5 {
		t.Errorf("d = %v, want 0.5", c.d)
	}
	same, _ := compareAt(u, "variant: X", doppelMethod, primaryK)
	if same.d != 0 || same.lo != 0 || same.hi != 0 {
		t.Errorf("identical lists: %v [%v, %v], want exactly 0 [0, 0]", same.d, same.lo, same.hi)
	}
	if same.clusters != 4 {
		t.Errorf("event clusters %d, want 4", same.clusters)
	}
	again, _ := compareAt(u, doppelMethod, "dupl (t=50)", primaryK)
	if fmt.Sprint(again.reps) != fmt.Sprint(c.reps) {
		t.Error("the bootstrap is not deterministic")
	}
}

// TestPoolCellsFloorAndWeights: a unit below the event-cluster floor is left
// out, and corpora weigh equally however many units each has.
func TestPoolCellsFloorAndWeights(t *testing.T) {
	units := []sizeUnit{{c: compareOut{Corpus: "a"}}, {c: compareOut{Corpus: "a"}}, {c: compareOut{Corpus: "b"}}, {c: compareOut{Corpus: "c"}}}
	mk := func(d float64, clusters int) *sizeCell {
		reps := make([]float64, bootReps)
		for i := range reps {
			reps[i] = d
		}
		return &sizeCell{d: d, reps: reps, clusters: clusters}
	}
	cells := []*sizeCell{mk(0.3, 5), mk(0.1, 3), mk(0.6, 4), mk(-9, 2)}
	p := poolCells(units, cells, minEventClusters)
	// Corpus a: (0.3 + 0.1) / 2 = 0.2; b: 0.6; c below the floor.
	if p.units != 3 || p.corpora != 2 || math.Abs(p.d-0.4) > 1e-12 || math.Abs(p.lo-0.4) > 1e-12 || math.Abs(p.hi-0.4) > 1e-12 {
		t.Errorf("pooled %+v, want D 0.4 [0.4, 0.4] over 3 units in 2 corpora", p)
	}
	if none := poolCells(units, []*sizeCell{nil, nil, nil, nil}, 0); !math.IsNaN(none.d) {
		t.Errorf("no cells pooled to %v, want NaN", none.d)
	}
}

// TestSizeSummaryRenders runs the whole summary over two fixture units and
// checks it is deterministic.
func TestSizeSummaryRenders(t *testing.T) {
	ev := map[int]bool{1: true, 3: true, 5: true, 7: true, 120: true}
	units := []sizeUnit{
		sizeFixture("x", 200, ev, map[string][]int{doppelMethod: seq(0, 100), "variant: X": seq(100, 200), "dupl (t=50)": seq(0, 10)}),
		sizeFixture("y", 200, ev, map[string][]int{doppelMethod: seq(0, 100), "variant: X": seq(0, 100)}),
	}
	var a, b bytes.Buffer
	writeSizeSummary(&a, units)
	writeSizeSummary(&b, units)
	if a.String() != b.String() {
		t.Fatal("size-summary is not deterministic")
	}
	if !bytes.Contains(a.Bytes(), []byte("### Criteria (a) and (b)")) {
		t.Error("no criteria table")
	}
}

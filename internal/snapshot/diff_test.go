package snapshot

import "testing"

func snap(mut func(*Snapshot)) Snapshot {
	s := Snapshot{
		Schema:    Schema,
		Doppel:    "test",
		Ontology:  "1.0.0",
		Params:    Params{Threshold: 0.6, MinNodes: 12, TestsMode: "exclude"},
		Functions: 2,
		Units: []Unit{
			{Key: "a.One", Digest: "d1"},
			{Key: "b.Two", Digest: "d2"},
		},
		Pairs: []Pair{{A: "a.One", B: "b.Two", Score: 0.80, Overlap: 0.5, MergeWorthy: true}},
	}
	if mut != nil {
		mut(&s)
	}
	return s
}

func keysOf(units []Unit) []string {
	out := make([]string, len(units))
	for i, u := range units {
		out[i] = u.Key
	}
	return out
}

func pairKeysOf(pairs []PairChange) []string {
	out := make([]string, len(pairs))
	for i, p := range pairs {
		out[i] = p.Key()
	}
	return out
}

func eq(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s = %v, want %v", label, got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s = %v, want %v", label, got, want)
			return
		}
	}
}

func TestDiffIdenticalIsEmpty(t *testing.T) {
	d := Diff(snap(nil), snap(nil))
	if !d.Comparable {
		t.Fatalf("identical snapshots reported incomparable: %s", d.Reason)
	}
	if !d.Empty() {
		t.Errorf("identical snapshots produced a non-empty delta: %+v", d)
	}
}

func TestDiffUnitAddedAndRemoved(t *testing.T) {
	base := snap(nil)
	head := snap(func(s *Snapshot) {
		s.Units = []Unit{{Key: "a.One", Digest: "d1"}, {Key: "c.Three", Digest: "d3"}}
		s.Functions = 2
	})

	d := Diff(base, head)
	eq(t, "UnitsAdded", keysOf(d.UnitsAdded), []string{"c.Three"})
	eq(t, "UnitsRemoved", keysOf(d.UnitsRemoved), []string{"b.Two"})
	if len(d.BodiesChanged) != 0 {
		t.Errorf("BodiesChanged = %v, want none", keysOf(d.BodiesChanged))
	}
}

func TestDiffBodyChange(t *testing.T) {
	head := snap(func(s *Snapshot) { s.Units[1].Digest = "CHANGED" })

	d := Diff(snap(nil), head)
	eq(t, "BodiesChanged", keysOf(d.BodiesChanged), []string{"b.Two"})
	if len(d.UnitsAdded) != 0 || len(d.UnitsRemoved) != 0 {
		t.Error("a body change must not be reported as an add or a remove")
	}
}

// A declaration with no body digests to the empty string. Two of those are not
// evidence of an identical body, so they must not be compared at all.
func TestDiffIgnoresEmptyDigests(t *testing.T) {
	base := snap(func(s *Snapshot) { s.Units[0].Digest = "" })
	head := snap(func(s *Snapshot) { s.Units[0].Digest = "" })

	if d := Diff(base, head); len(d.BodiesChanged) != 0 {
		t.Errorf("BodiesChanged = %v, want none", keysOf(d.BodiesChanged))
	}
}

func TestDiffPairAddedIsAttributableOnlyWhenAnEditExplainsIt(t *testing.T) {
	// A new unit whose arrival brings a new pair: attributable.
	head := snap(func(s *Snapshot) {
		s.Units = append(s.Units, Unit{Key: "c.Three", Digest: "d3"})
		s.Functions = 3
		s.Pairs = append(s.Pairs, Pair{A: "a.One", B: "c.Three", Score: 0.7})
	})
	d := Diff(snap(nil), head)
	eq(t, "PairsAdded", pairKeysOf(d.PairsAdded), []string{"a.One <-> c.Three"})
	if !d.PairsAdded[0].Attributable {
		t.Error("a pair involving a newly added unit must be attributable")
	}

	// The same new pair with no unit added or edited: retrieval re-ranked
	// around untouched code, which the session did not cause.
	churn := snap(func(s *Snapshot) {
		s.Pairs = append(s.Pairs, Pair{A: "a.One", B: "b.Two2", Score: 0.7})
	})
	d = Diff(snap(nil), churn)
	if len(d.PairsAdded) != 1 || d.PairsAdded[0].Attributable {
		t.Errorf("pair change with no edited side must not be attributable: %+v", d.PairsAdded)
	}
}

// The renderer prints only a bounded prefix, so ordering decides what a reader
// ever sees: changes the session caused first, and the actionable verdict
// ahead of high-scoring coincidences.
func TestPairChangeOrderingLeadsWithActionable(t *testing.T) {
	head := snap(func(s *Snapshot) {
		s.Units = append(s.Units, Unit{Key: "c.New", Digest: "d3"})
		s.Pairs = append(s.Pairs,
			Pair{A: "a.One", B: "c.New", Score: 0.70, MergeWorthy: true},
			Pair{A: "b.Two", B: "c.New", Score: 0.99},
			Pair{A: "x.Idle", B: "y.Idle", Score: 0.95},
		)
	})
	d := Diff(snap(nil), head)
	eq(t, "PairsAdded", pairKeysOf(d.PairsAdded), []string{
		"a.One <-> c.New",   // attributable + merge-worthy, despite the lowest score
		"b.Two <-> c.New",   // attributable
		"x.Idle <-> y.Idle", // not attributable, despite scoring 0.95
	})
}

func TestDiffScoreDriftFloor(t *testing.T) {
	tests := []struct {
		name      string
		score     float64
		wantDrift bool
	}{
		{"below floor", 0.80 + driftFloor/2, false},
		{"at floor", 0.80 + driftFloor, true},
		{"well above floor", 0.95, true},
		{"unchanged", 0.80, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			head := snap(func(s *Snapshot) { s.Pairs[0].Score = tc.score })
			d := Diff(snap(nil), head)
			if got := len(d.Drift) > 0; got != tc.wantDrift {
				t.Errorf("drift reported = %v, want %v (score %v -> %v)", got, tc.wantDrift, 0.80, tc.score)
			}
		})
	}
}

// Incomparability is a result, not an error: every mismatch means the two runs
// measured different things, and a diff across them would be confidently wrong.
func TestDiffIncomparable(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*Snapshot)
	}{
		{"schema", func(s *Snapshot) { s.Schema = Schema + 1 }},
		{"doppel build", func(s *Snapshot) { s.Doppel = "other" }},
		{"ontology", func(s *Snapshot) { s.Ontology = "9.9.9" }},
		{"threshold", func(s *Snapshot) { s.Params.Threshold = 0.9 }},
		{"min nodes", func(s *Snapshot) { s.Params.MinNodes = 99 }},
		{"tests mode", func(s *Snapshot) { s.Params.TestsMode = "include" }},
		{"channel k", func(s *Snapshot) { s.Params.ChannelK = 99 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := Diff(snap(nil), snap(tc.mut))
			if d.Comparable {
				t.Fatal("mismatched snapshots reported as comparable")
			}
			if d.Reason == "" {
				t.Error("incomparable delta must explain itself")
			}
			// The counts still describe both runs; only the attribution is
			// withheld, because that is the part that would be wrong.
			if len(d.UnitsAdded) != 0 || len(d.PairsAdded) != 0 || len(d.Drift) != 0 {
				t.Error("an incomparable delta must not claim specific changes")
			}
		})
	}
}

func TestDiffScoreboardCounts(t *testing.T) {
	head := snap(func(s *Snapshot) {
		s.Functions = 5
		s.Pairs[0].MergeWorthy = false
	})
	d := Diff(snap(nil), head)

	if d.FunctionsBefore != 2 || d.FunctionsAfter != 5 {
		t.Errorf("functions %d -> %d, want 2 -> 5", d.FunctionsBefore, d.FunctionsAfter)
	}
	if d.MergeBefore != 1 || d.MergeAfter != 0 {
		t.Errorf("merge-worthy %d -> %d, want 1 -> 0", d.MergeBefore, d.MergeAfter)
	}
}

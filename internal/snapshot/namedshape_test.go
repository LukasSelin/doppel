package snapshot

import "testing"

// Named-shape reaches --format json at the storage resolution, and a pair
// nothing measured it for reads -1: a consumer must be able to tell "these
// two bodies share no named structure" from "nobody looked".
func TestBuildCarriesNamedShape(t *testing.T) {
	u, d, p, c := sampleInputs()
	params := Params{Threshold: 0.6, MinNodes: 12, TestsMode: "exclude"}

	s := Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})
	if got := s.Pairs[0].NamedShape; got != -1 {
		t.Fatalf("unmeasured named-shape stored as %v, want -1", got)
	}

	p[0].NamedShape, p[0].HasNamedShape = 0.31234, true
	s = Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})
	if got := s.Pairs[0].NamedShape; got != 0.31 {
		t.Fatalf("named-shape stored as %v, want 0.31", got)
	}

	// A measured zero stays a zero.
	p[0].NamedShape = 0
	s = Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})
	if got := s.Pairs[0].NamedShape; got != 0 {
		t.Fatalf("measured zero stored as %v", got)
	}
}

// The annotation contract at the schema boundary: named-shape is weighted by
// this run's corpus, so a delta that noticed it would blame a session for
// names somebody else added.
func TestDiffIgnoresNamedShape(t *testing.T) {
	u, d, p, c := sampleInputs()
	params := Params{Threshold: 0.6, MinNodes: 12, TestsMode: "exclude"}

	p[0].NamedShape, p[0].HasNamedShape = 0.9, true
	before := Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})
	p[0].NamedShape = 0.2
	after := Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})
	p[0].HasNamedShape = false
	unmeasured := Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})

	for name, s := range map[string]Snapshot{"moved": after, "unmeasured": unmeasured} {
		delta := Diff(before, s)
		if !delta.Comparable {
			t.Fatalf("%s: runs reported incomparable: %s", name, delta.Reason)
		}
		if !delta.Empty() {
			t.Fatalf("%s: a changed named-shape produced a non-empty delta: %+v", name, delta)
		}
	}
}

package snapshot

import (
	"testing"
)

// TestBuildCarriesExplain pins the JSON payload's half of the annotation: the
// sentence a report prints has to survive into --format json, which is the
// only machine-readable place a consumer can read it back.
func TestBuildCarriesExplain(t *testing.T) {
	u, d, p, c := sampleInputs()
	p[0].Explain = "identical after rename, commutative-reorder"
	s := Build(u, d, p, c, "", "test", Params{Threshold: 0.6, MinNodes: 12, TestsMode: "exclude"}, CorpusMetrics{})

	if len(s.Pairs) != 1 {
		t.Fatalf("want 1 pair, got %d", len(s.Pairs))
	}
	if got, want := s.Pairs[0].Explain, "identical after rename, commutative-reorder"; got != want {
		t.Fatalf("Pair.Explain = %q, want %q", got, want)
	}
}

// TestDiffIgnoresExplain is the annotation contract at the schema boundary.
// The sentence describes the corpus as much as the pair — a rule firing
// elsewhere can reword it — so a delta that noticed it would blame a session
// for text nobody wrote.
func TestDiffIgnoresExplain(t *testing.T) {
	u, d, p, c := sampleInputs()
	params := Params{Threshold: 0.6, MinNodes: 12, TestsMode: "exclude"}

	p[0].Explain = "identical after rename"
	before := Build(u, d, p, c, "", "test", params, CorpusMetrics{})
	p[0].Explain = "differs by one extra defer"
	after := Build(u, d, p, c, "", "test", params, CorpusMetrics{})

	delta := Diff(before, after)
	if !delta.Comparable {
		t.Fatalf("runs reported incomparable: %s", delta.Reason)
	}
	if n := len(delta.PairsAdded) + len(delta.PairsRemoved) + len(delta.Drift); n != 0 {
		t.Fatalf("a reworded explanation moved %d pairs in the delta", n)
	}
	if !delta.Empty() {
		t.Fatal("a reworded explanation produced a non-empty delta")
	}
}

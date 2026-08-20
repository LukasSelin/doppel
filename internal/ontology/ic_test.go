package ontology

import (
	"math"
	"testing"
)

// toyCounts is the worked corpus the IC design is documented against:
// 100 tag occurrences, dominated by error_wrapping the way real Go code is.
func toyCounts() map[TermID]int {
	return map[TermID]int{
		ConErrorWrapping: 60,
		ConDBAccess:      10,
		ConCaching:       5,
		ConTransaction:   3,
		ConRetry:         2,
		ConHTTPCall:      5,
		ConValidation:    5,
		ConMapping:       5,
		ConConcurrency:   5,
	}
}

// With add-one smoothing over the 9 leaves, N=100 gives a root frequency of
// 109 and per-term frequencies of count+1, summed up the tree.
func TestNewCorpusICToyValues(t *testing.T) {
	ic := NewCorpusIC(Default(), toyCounts())
	root := 109.0
	tests := []struct {
		id   TermID
		freq float64
	}{
		{ConErrorWrapping, 61},
		{ConErrorHandling, 61}, // unary parent: same frequency, same IC
		{ConDBAccess, 11},
		{ConCaching, 6},
		{ConTransaction, 4},
		{ConDataStoreAccess, 21}, // 11 + 6 + 4
		{ConHTTPCall, 6},
		{ConRemoteIO, 6},
		{ConIOOperation, 27}, // 21 + 6
		{ConValidation, 6},
		{ConMapping, 6},
		{ConDataTransformation, 12},
		{ConConcurrency, 6},
		{ConRetry, 3},
		{ConFaultTolerance, 3},
		{ConControlFlow, 9},
	}
	for _, tt := range tests {
		want := math.Log(root / tt.freq)
		if got := ic.Of(tt.id); !closeTo(got, want) {
			t.Errorf("IC(%q) = %v, want ln(109/%g) = %v", tt.id, got, tt.freq, want)
		}
	}

	// The root's IC must be exactly +0.0, not merely small: it is what makes
	// cross-branch Lin similarity exactly zero.
	if got := ic.Of(ConConcept); got != 0 {
		t.Errorf("IC(root) = %v, want exactly 0", got)
	}

	// Every concrete leaf and every non-root node is strictly positive.
	for _, term := range Default().TermsOfKind(KindConcept) {
		if term.ID == ConConcept {
			continue
		}
		if got := ic.Of(term.ID); got <= 0 {
			t.Errorf("IC(%q) = %v, want > 0", term.ID, got)
		}
	}

	// An unknown term is treated as maximally rare: pseudo-count 1.
	if got, want := ic.Of("grpc_call"), math.Log(root); !closeTo(got, want) {
		t.Errorf("IC(unknown) = %v, want ln(109) = %v", got, want)
	}
}

// An empty corpus degenerates to add-one smoothing alone: every leaf has the
// same pseudo-count, hence the same IC — behaviorally uniform, like today.
func TestNewCorpusICEmptyCorpus(t *testing.T) {
	ic := NewCorpusIC(Default(), nil)
	want := math.Log(9.0)
	leaves := []TermID{
		ConRetry, ConHTTPCall, ConDBAccess, ConValidation, ConMapping,
		ConTransaction, ConCaching, ConConcurrency, ConErrorWrapping,
	}
	for _, id := range leaves {
		if got := ic.Of(id); !closeTo(got, want) {
			t.Errorf("IC(%q) on empty corpus = %v, want ln(9) = %v", id, got, want)
		}
	}
	if got := ic.Of(ConConcept); got != 0 {
		t.Errorf("IC(root) on empty corpus = %v, want exactly 0", got)
	}
}

// Counts keyed by abstract or unknown terms must not leak into the totals.
func TestNewCorpusICIgnoresNonLeafKeys(t *testing.T) {
	withJunk := toyCounts()
	withJunk[ConIOOperation] = 500 // abstract: never emitted by the tagger
	withJunk["not_a_term"] = 500
	clean := NewCorpusIC(Default(), toyCounts())
	junk := NewCorpusIC(Default(), withJunk)
	for _, id := range []TermID{ConErrorWrapping, ConDBAccess, ConIOOperation, ConConcept} {
		if a, b := clean.Of(id), junk.Of(id); a != b {
			t.Errorf("IC(%q) moved from %v to %v when junk keys were added", id, a, b)
		}
	}
}

func TestNewCorpusICDeterministic(t *testing.T) {
	first := NewCorpusIC(Default(), toyCounts())
	for i := 0; i < 100; i++ {
		again := NewCorpusIC(Default(), toyCounts())
		for _, term := range Default().TermsOfKind(KindConcept) {
			if first.Of(term.ID) != again.Of(term.ID) {
				t.Fatalf("run %d: IC(%q) differs", i, term.ID)
			}
		}
	}
}

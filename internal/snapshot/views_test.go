package snapshot

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/comparator"
)

// TestBuildCarriesViews pins the JSON payload's half of the views: the five
// numbers a report prints survive into --format json, rounded to the storage
// resolution.
func TestBuildCarriesViews(t *testing.T) {
	u, d, p, c := sampleInputs()
	p[0].Evidence = &comparator.StructuralEvidence{OverlapScore: 0.5, Views: comparator.ConceptViews{
		Shape: 0.666666, Corpus: 0.41234, Feature: 0.125, AInB: 0.9, BInA: 0.3, HasFeature: true, Disagree: true,
	}}
	s := Build(u, d, p, c, nil, "", "test", Params{Threshold: 0.6, MinNodes: 12, TestsMode: "exclude"}, CorpusMetrics{})
	if len(s.Pairs) != 1 {
		t.Fatalf("want 1 pair, got %d", len(s.Pairs))
	}
	got := s.Pairs[0]
	if got.ViewShape != 0.67 || got.ViewCorpus != 0.41 || got.ViewFeature != 0.13 || got.ViewAInB != 0.9 || got.ViewBInA != 0.3 || !got.ViewsDisagree {
		t.Fatalf("views stored as %+v", got)
	}
}

// An unmeasured feature view is -1, never 0: a schema consumer must be able
// to tell "these vocabularies share nothing" from "no vocabulary was read".
func TestBuildViewsAbsentIsMinusOne(t *testing.T) {
	u, d, p, c := sampleInputs()
	params := Params{Threshold: 0.6, MinNodes: 12, TestsMode: "exclude"}

	p[0].Evidence = nil
	s := Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})
	if got := s.Pairs[0]; got.ViewFeature != -1 || got.ViewAInB != -1 || got.ViewBInA != -1 || got.ViewsDisagree {
		t.Fatalf("no evidence: %+v", got)
	}

	p[0].Evidence = &comparator.StructuralEvidence{Views: comparator.ConceptViews{Shape: 0.5, Corpus: 0.25}}
	s = Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})
	if got := s.Pairs[0]; got.ViewShape != 0.5 || got.ViewCorpus != 0.25 || got.ViewFeature != -1 || got.ViewAInB != -1 || got.ViewBInA != -1 {
		t.Fatalf("no vocabulary: %+v", got)
	}
}

// TestDiffIgnoresViews is the annotation contract at the schema boundary: the
// views are corpus-relative, so a delta that noticed them would blame a
// session for a vocabulary it did not learn.
func TestDiffIgnoresViews(t *testing.T) {
	u, d, p, c := sampleInputs()
	params := Params{Threshold: 0.6, MinNodes: 12, TestsMode: "exclude"}

	p[0].Evidence = &comparator.StructuralEvidence{OverlapScore: 0.5, Views: comparator.ConceptViews{Shape: 0.6, Feature: 0.1, HasFeature: true, Disagree: true}}
	before := Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})
	p[0].Evidence = &comparator.StructuralEvidence{OverlapScore: 0.5, Views: comparator.ConceptViews{Shape: 0.1, Feature: 0.9, HasFeature: true}}
	after := Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})

	delta := Diff(before, after)
	if !delta.Comparable {
		t.Fatalf("runs reported incomparable: %s", delta.Reason)
	}
	if !delta.Empty() {
		t.Fatalf("moved views produced a non-empty delta: %+v", delta)
	}
}

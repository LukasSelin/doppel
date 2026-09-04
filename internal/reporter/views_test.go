package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/ontology"
)

func viewsPair(v comparator.ConceptViews) analyzer.SimilarPair {
	p := samplePair(nil)
	p.Evidence = &comparator.StructuralEvidence{OverlapScore: 0.41, Views: v}
	return p
}

var measuredViews = comparator.ConceptViews{
	Shape: 0, Corpus: 0, Feature: 0.62, AInB: 0.91, BInA: 0.34,
	SharedVocabulary: []ontology.WeightedFeature{{Name: "sel:sql.Open", Weight: 3}, {Name: "call:store.Get", Weight: 2}, {Name: "id:que|ry", Weight: 1}},
	HasFeature:       true, Disagree: true,
}

func TestPrintConceptViewsLine(t *testing.T) {
	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{viewsPair(measuredViews)}, Meta{})
	out := b.String()
	want := "  concept views: shape 0.00  corpus 0.00  feature 0.62  a-in-b 0.91  b-in-a 0.34 (taxonomy misses shared vocabulary)\n" +
		"    shared vocabulary: sel:sql.Open, call:store.Get, id:que|ry\n" +
		"  structural overlap: 0.41\n"
	if !strings.Contains(out, want) {
		t.Errorf("text report missing the views block:\n%s", out)
	}
}

func TestPrintConceptViewsOtherDirection(t *testing.T) {
	v := comparator.ConceptViews{Shape: 0.67, Corpus: 0.5, Feature: 0.05, HasFeature: true, Disagree: true}
	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{viewsPair(v)}, Meta{})
	if !strings.Contains(b.String(), "feature 0.05  a-in-b 0.00  b-in-a 0.00 (taxonomy asserts kinship the vocabularies lack)\n") {
		t.Errorf("text report missing the taxonomy-only clause:\n%s", b.String())
	}
	if strings.Contains(b.String(), "shared vocabulary:") {
		t.Errorf("a pair with no shared features printed a vocabulary line:\n%s", b.String())
	}
}

func TestPrintConceptViewsAgreeingHasNoClause(t *testing.T) {
	v := comparator.ConceptViews{Shape: 1, Corpus: 1, Feature: 1, AInB: 1, BInA: 1, HasFeature: true}
	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{viewsPair(v)}, Meta{})
	if !strings.Contains(b.String(), "  concept views: shape 1.00  corpus 1.00  feature 1.00  a-in-b 1.00  b-in-a 1.00\n") {
		t.Errorf("agreeing views rendered with a clause:\n%s", b.String())
	}
}

// A run that measured no feature view — a library caller with no vocabulary
// table — prints no views at all: an unmeasured view is not a zero.
func TestPrintOmitsUnmeasuredViews(t *testing.T) {
	v := comparator.ConceptViews{Shape: 0.67, Corpus: 0.5}
	var b, md strings.Builder
	Print(&b, []analyzer.SimilarPair{viewsPair(v)}, Meta{})
	PrintMarkdown(&md, []analyzer.SimilarPair{viewsPair(v)}, Meta{})
	for _, out := range []string{b.String(), md.String()} {
		if strings.Contains(out, "oncept views") || strings.Contains(out, "hared vocabulary") {
			t.Errorf("unmeasured views were rendered:\n%s", out)
		}
	}
	if !strings.Contains(b.String(), "structural overlap: 0.41") {
		t.Errorf("overlap line lost:\n%s", b.String())
	}
}

func TestPrintMarkdownConceptViews(t *testing.T) {
	var b strings.Builder
	PrintMarkdown(&b, []analyzer.SimilarPair{viewsPair(measuredViews)}, Meta{})
	out := b.String()
	want := "**Concept views:** shape `0.00`, corpus `0.00`, feature `0.62`, a-in-b `0.91`, b-in-a `0.34` — taxonomy misses shared vocabulary\n\n" +
		"**Shared vocabulary:** `sel:sql.Open`, `call:store.Get`, `id:que\\|ry`\n\n" +
		"**Structural overlap:** `0.41` (not merge-worthy)\n\n"
	if !strings.Contains(out, want) {
		t.Errorf("markdown missing the views block:\n%s", out)
	}
}

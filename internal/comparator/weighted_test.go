package comparator

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// weightedComparator builds a Comparator over a corpus where error_wrapping
// dominates — the shape real Go code has, and the reason IC weighting exists.
func weightedComparator() *Comparator {
	o := ontology.Default()
	return New(ontology.NewScorer(o, ontology.NewCorpusIC(o, map[ontology.TermID]int{
		ontology.ConErrorWrapping: 60,
		ontology.ConDBAccess:      10,
		ontology.ConCaching:       5,
		ontology.ConTransaction:   3,
		ontology.ConRetry:         2,
		ontology.ConHTTPCall:      5,
		ontology.ConValidation:    5,
		ontology.ConMapping:       5,
		ontology.ConConcurrency:   5,
	})))
}

func TestFreeCompareEqualsUniformInstance(t *testing.T) {
	a := concepter.ConceptDoc{Name: "a", Package: "p", Role: "utility",
		Concepts: parser.Certain("db_access", "error_wrapping"), Callees: []string{"x"}}
	b := concepter.ConceptDoc{Name: "b", Package: "p", Role: "utility",
		Concepts: parser.Certain("caching", "error_wrapping"), Callees: []string{"x"}}
	free := Compare(a, b)
	inst := New(ontology.NewScorer(ontology.Default(), nil)).Compare(a, b)
	if free.OverlapScore != inst.OverlapScore {
		t.Errorf("free Compare = %v, uniform instance = %v", free.OverlapScore, inst.OverlapScore)
	}
	if strings.Join(free.Reasons, "\n") != strings.Join(inst.Reasons, "\n") {
		t.Errorf("free Compare and uniform instance produced different reasons")
	}
}

// Sharing only the ubiquitous tag must score below sharing only a rare one —
// the two are indistinguishable to the uniform comparator.
func TestWeightedCompareDiscountsUbiquitousSharedTags(t *testing.T) {
	comp := weightedComparator()
	doc := func(name string, tags ...string) concepter.ConceptDoc {
		return concepter.ConceptDoc{Name: name, Package: "p", Role: "utility", Concepts: parser.Certain(tags...)}
	}

	ubiquitous := comp.Compare(doc("a", "error_wrapping", "mapping"), doc("b", "error_wrapping", "concurrency"))
	rare := comp.Compare(doc("c", "db_access", "mapping"), doc("d", "db_access", "concurrency"))

	if ubiquitous.PatternRelatedness >= rare.PatternRelatedness {
		t.Errorf("ubiquitous shared tag scored %v, rare shared tag %v — want ubiquitous lower",
			ubiquitous.PatternRelatedness, rare.PatternRelatedness)
	}
	if ubiquitous.OverlapScore >= rare.OverlapScore {
		t.Errorf("composite: ubiquitous %v >= rare %v", ubiquitous.OverlapScore, rare.OverlapScore)
	}
	// The uniform comparator cannot tell these apart — that is the gap IC closes.
	if u, r := Compare(doc("a", "error_wrapping", "mapping"), doc("b", "error_wrapping", "concurrency")),
		Compare(doc("c", "db_access", "mapping"), doc("d", "db_access", "concurrency")); u.OverlapScore != r.OverlapScore {
		t.Errorf("fixture broken: uniform comparator already distinguishes them (%v vs %v)", u.OverlapScore, r.OverlapScore)
	}
}

// The merge-signal gate must stay corpus-independent. On this corpus the Lin
// similarity of the http_call/db_access cousins rises above 0.5 — but cousins
// must not become a merge signal just because both tags happen to be rare here.
func TestGateIgnoresCorpusWeighting(t *testing.T) {
	comp := weightedComparator()
	doc := func(name, tag string) concepter.ConceptDoc {
		return concepter.ConceptDoc{Name: name, Package: "p", Role: "utility", Concepts: parser.Certain(tag)}
	}

	ev := comp.Compare(doc("a", "http_call"), doc("b", "db_access"))
	if len(ev.RelatedPatterns) != 1 || ev.RelatedPatterns[0].Score <= 0.5 {
		t.Fatalf("fixture broken: want a Lin cousin match above 0.5, got %+v", ev.RelatedPatterns)
	}
	if ev.PatternSignalBest >= 0.5 {
		t.Errorf("PatternSignalBest = %v: the gate leaked corpus weighting (Wu-Palmer cousins are 0.33)", ev.PatternSignalBest)
	}
	if got, want := countSignals(ev), countSignals(Compare(doc("a", "http_call"), doc("b", "db_access"))); got != want {
		t.Errorf("signal count = %d weighted vs %d uniform — the gate must not move", got, want)
	}

	// And the mirror image: on a transaction-dominated corpus, sibling Lin
	// similarity collapses far below 0.5, but siblings must keep counting.
	o := ontology.Default()
	txHeavy := New(ontology.NewScorer(o, ontology.NewCorpusIC(o, map[ontology.TermID]int{
		ontology.ConTransaction: 80, ontology.ConDBAccess: 1, ontology.ConCaching: 1,
	})))
	sib := txHeavy.Compare(doc("a", "db_access"), doc("b", "caching"))
	if len(sib.RelatedPatterns) != 1 || sib.RelatedPatterns[0].Score >= 0.5 {
		t.Fatalf("fixture broken: want a Lin sibling match below 0.5, got %+v", sib.RelatedPatterns)
	}
	if sib.PatternSignalBest < 0.5 {
		t.Errorf("PatternSignalBest = %v: siblings stopped counting under corpus weighting", sib.PatternSignalBest)
	}
}

// Evidence must keep explaining the score: the weighted matches carry Lin
// scores into the same related-patterns bullet.
func TestWeightedCompareEvidenceUsesLinScores(t *testing.T) {
	comp := weightedComparator()
	ev := comp.Compare(
		concepter.ConceptDoc{Name: "a", Package: "p", Role: "leaf", Concepts: parser.Certain("db_access")},
		concepter.ConceptDoc{Name: "b", Package: "p", Role: "leaf", Concepts: parser.Certain("caching")},
	)
	if len(ev.RelatedPatterns) != 1 {
		t.Fatalf("want one match, got %+v", ev.RelatedPatterns)
	}
	if m := ev.RelatedPatterns[0]; m.LCA != ontology.ConDataStoreAccess {
		t.Errorf("match LCA = %q, want data_store_access", m.LCA)
	}
	var found bool
	for _, r := range ev.Reasons {
		if strings.Contains(r, "related patterns:") && strings.Contains(r, "data_store_access") {
			found = true
		}
	}
	if !found {
		t.Errorf("no related-patterns bullet in %v", ev.Reasons)
	}
}

func TestWeightedCompareIsDeterministic(t *testing.T) {
	comp := weightedComparator()
	a := concepter.ConceptDoc{Name: "a", Package: "svc", Exported: true, Role: "passthrough",
		Callees:        []string{"save", "log"},
		Concepts:       parser.Certain("db_access", "error_wrapping", "concurrency"),
		CallerConcepts: parser.Certain("http_call", "validation"), CalleeConcepts: parser.Certain("transaction", "mapping")}
	b := concepter.ConceptDoc{Name: "b", Package: "svc", Exported: true, Role: "utility",
		Callees:        []string{"save", "flush"},
		Concepts:       parser.Certain("caching", "error_wrapping", "retry"),
		CallerConcepts: parser.Certain("db_access", "validation"), CalleeConcepts: parser.Certain("caching", "mapping")}
	first := comp.Compare(a, b)
	want := strings.Join(first.Reasons, "\n")
	for i := 0; i < 500; i++ {
		ev := comp.Compare(a, b)
		if ev.OverlapScore != first.OverlapScore || strings.Join(ev.Reasons, "\n") != want {
			t.Fatalf("run %d diverged", i)
		}
	}
}

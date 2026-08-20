package ontology

import (
	"sort"
	"strings"
	"testing"
)

// The invariant the rest of the pipeline leans on. If this fails, a term is
// misdeclared and scores are already wrong somewhere downstream.
func TestDefaultOntologyIsValid(t *testing.T) {
	errs := Default().Validate()
	for _, err := range errs {
		t.Errorf("axiom violation: %v", err)
	}
	if n := len(Default().Terms()); n == 0 {
		t.Fatal("the default ontology is empty")
	}
}

func TestValidateErrorsAreSorted(t *testing.T) {
	// Several violations at once, so the ordering is observable.
	o := New([]Term{
		{ID: "Zebra", Kind: KindConcept},
		{ID: "alpha", Kind: KindConcept, Parent: "nowhere"},
	})
	errs := o.Validate()
	if len(errs) < 2 {
		t.Fatalf("got %d errors, want several", len(errs))
	}
	msgs := make([]string, len(errs))
	for i, err := range errs {
		msgs[i] = err.Error()
	}
	if !sort.StringsAreSorted(msgs) {
		t.Errorf("Validate returned unsorted errors: %v", msgs)
	}
}

func hasError(errs []error, substr string) bool {
	for _, err := range errs {
		if strings.Contains(err.Error(), substr) {
			return true
		}
	}
	return false
}

// Negative cases. Each builds a deliberately inconsistent vocabulary and checks
// the matching axiom actually fires — a validator nobody has seen fail is a
// validator nobody knows works.
func TestValidateCatchesInconsistencies(t *testing.T) {
	documented := func(id TermID, kind Kind, parent TermID) Term {
		return Term{ID: id, Kind: kind, Parent: parent, Label: "L", Def: "D."}
	}

	tests := []struct {
		name  string
		terms []Term
		want  string
	}{
		{
			name: "parent that does not exist",
			terms: []Term{
				documented("concept", KindConcept, ""),
				documented("http_call", KindConcept, "remote_io"),
			},
			want: `has parent "remote_io", which does not exist`,
		},
		{
			name: "parent cycle",
			terms: []Term{
				documented("alpha", KindConcept, "beta"),
				documented("beta", KindConcept, "alpha"),
			},
			want: "parent cycle",
		},
		{
			name: "parent in another kind",
			terms: []Term{
				documented("concept", KindConcept, ""),
				documented("role", KindRole, ""),
				documented("leaf", KindRole, "concept"),
			},
			want: `is role but its parent "concept" is concept`,
		},
		{
			name: "two roots in one kind",
			terms: []Term{
				documented("concept", KindConcept, ""),
				documented("other_concept", KindConcept, ""),
			},
			want: "kind concept has 2 roots",
		},
		{
			name: "no root in a kind",
			terms: []Term{
				documented("alpha", KindConcept, "beta"),
				documented("beta", KindConcept, "alpha"),
			},
			want: "kind concept has no root",
		},
		{
			name:  "identifier that is not snake_case",
			terms: []Term{documented("httpCall", KindConcept, "")},
			want:  `term "httpCall" is not snake_case`,
		},
		{
			name:  "term with no definition",
			terms: []Term{{ID: "concept", Kind: KindConcept, Label: "Concept"}},
			want:  `term "concept" has no definition`,
		},
		{
			name:  "term with no label",
			terms: []Term{{ID: "concept", Kind: KindConcept, Def: "D."}},
			want:  `term "concept" has no label`,
		},
		{
			name: "relation whose range does not resolve",
			terms: []Term{
				documented("relation", KindRelation, ""),
				{ID: "exhibits", Kind: KindRelation, Parent: "relation", Label: "L", Def: "D.",
					Domain: "relation", Range: "concept", Weight: 1.0},
			},
			want: `has range "concept", which does not exist`,
		},
		{
			name: "inverse that is not symmetric",
			terms: []Term{
				documented("relation", KindRelation, ""),
				{ID: "calls", Kind: KindRelation, Parent: "relation", Label: "L", Def: "D.",
					Domain: "relation", Range: "relation", Inverse: "called_by", Weight: 1.0},
				{ID: "called_by", Kind: KindRelation, Parent: "relation", Label: "L", Def: "D.",
					Domain: "relation", Range: "relation", Inverse: "calls_wrongly"},
			},
			want: "declares inverse",
		},
		{
			name: "weights that do not sum to one",
			terms: []Term{
				documented("relation", KindRelation, ""),
				{ID: "calls", Kind: KindRelation, Parent: "relation", Label: "L", Def: "D.",
					Domain: "relation", Range: "relation", Weight: 0.25},
			},
			want: "relation weights sum to",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := New(tt.terms).Validate()
			if !hasError(errs, tt.want) {
				t.Errorf("Validate did not report %q; got %v", tt.want, errs)
			}
		})
	}
}

// Axiom 7 is what keeps the comparator's composite on a 0.0-1.0 scale without
// renormalizing, so it is worth asserting directly and not only through
// Validate.
func TestRelationWeightsSumToOne(t *testing.T) {
	var sum float64
	var scored int
	for _, term := range Default().TermsOfKind(KindRelation) {
		sum += term.Weight
		if term.Weight > 0 {
			scored++
		}
	}
	if !closeTo(sum, 1.0) {
		t.Errorf("relation weights sum to %v, want 1.0", sum)
	}
	if scored != 12 {
		t.Errorf("got %d scored relations, want 12", scored)
	}
}

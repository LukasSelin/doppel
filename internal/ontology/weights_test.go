package ontology

import (
	"math"
	"testing"
)

func TestWithWeightsEmptyIsDefault(t *testing.T) {
	for _, overrides := range []map[TermID]float64{nil, {}} {
		o, err := WithWeights(overrides)
		if err != nil {
			t.Fatalf("WithWeights(%v): %v", overrides, err)
		}
		// Not merely equal weights: the exact same vocabulary. Scaling by a
		// computed 1.0/sum would perturb last bits, so the implementation must
		// short-circuit.
		if o != Default() {
			t.Errorf("WithWeights(%v) built a copy; want Default() itself", overrides)
		}
	}
}

func TestWithWeightsZeroesAndRenormalizes(t *testing.T) {
	o, err := WithWeights(map[TermID]float64{RelCalls: 0})
	if err != nil {
		t.Fatal(err)
	}
	if w := o.Weight(RelCalls); w != 0 {
		t.Errorf("ablated calls weight = %v, want 0", w)
	}
	var sum float64
	for _, rel := range o.ScoredRelations() {
		sum += o.Weight(rel)
	}
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("reweighted sum = %v, want 1.0", sum)
	}
	// The remaining eleven scale uniformly: exhibits/has_role keep their ratio.
	def := Default()
	gotRatio := o.Weight(RelExhibits) / o.Weight(RelHasRole)
	wantRatio := def.Weight(RelExhibits) / def.Weight(RelHasRole)
	if math.Abs(gotRatio-wantRatio) > 1e-12 {
		t.Errorf("renormalization changed relative weights: %v vs %v", gotRatio, wantRatio)
	}
	// And they scale up, not down: mass from calls went somewhere.
	if o.Weight(RelExhibits) <= def.Weight(RelExhibits) {
		t.Errorf("exhibits did not absorb ablated mass: %v <= %v",
			o.Weight(RelExhibits), def.Weight(RelExhibits))
	}
	if errs := o.Validate(); len(errs) > 0 {
		t.Errorf("reweighted vocabulary fails validation: %v", errs)
	}
	// The default is untouched.
	if def.Weight(RelCalls) != 0.210 {
		t.Fatalf("WithWeights mutated the default vocabulary: calls = %v", def.Weight(RelCalls))
	}
}

func TestWithWeightsFullOverride(t *testing.T) {
	rels := Default().ScoredRelations()
	overrides := make(map[TermID]float64, len(rels))
	for _, rel := range rels {
		overrides[rel] = 1.0 / float64(len(rels))
	}
	o, err := WithWeights(overrides)
	if err != nil {
		t.Fatalf("uniform full override rejected: %v", err)
	}
	for _, rel := range rels {
		if w := o.Weight(rel); math.Abs(w-1.0/12) > 1e-12 {
			t.Errorf("weight %s = %v, want uniform", rel, w)
		}
	}
}

func TestWithWeightsErrors(t *testing.T) {
	cases := []struct {
		name      string
		overrides map[TermID]float64
	}{
		{"unknown id", map[TermID]float64{"no_such_relation": 0.1}},
		{"abstract root", map[TermID]float64{RelRelation: 0.1}},
		{"not a relation", map[TermID]float64{"http_call": 0.1}},
		{"negative", map[TermID]float64{RelCalls: -0.1}},
		{"sum above one", map[TermID]float64{RelCalls: 0.7, RelExhibits: 0.4}},
		{"full override not summing to one", func() map[TermID]float64 {
			m := map[TermID]float64{}
			for _, rel := range Default().ScoredRelations() {
				m[rel] = 0.05
			}
			return m
		}()},
	}
	for _, tc := range cases {
		if _, err := WithWeights(tc.overrides); err == nil {
			t.Errorf("%s: accepted", tc.name)
		}
	}
}

func TestScoredRelationsAreTheTwelve(t *testing.T) {
	rels := Default().ScoredRelations()
	if len(rels) != 12 {
		t.Fatalf("ScoredRelations returned %d, want 12", len(rels))
	}
	// Declaration order, RelCalls first — the order the ablation report reads
	// in and the fitter's vector layout.
	if rels[0] != RelCalls {
		t.Errorf("first scored relation = %s, want calls", rels[0])
	}
	seen := map[TermID]bool{}
	for _, rel := range rels {
		term, ok := Default().Get(rel)
		if !ok || term.Kind != KindRelation || term.Abstract {
			t.Errorf("%s is not a concrete relation", rel)
		}
		if seen[rel] {
			t.Errorf("%s listed twice", rel)
		}
		seen[rel] = true
	}
}

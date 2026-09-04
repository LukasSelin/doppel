package ontology

import (
	"fmt"
	"math"
)

// WithWeights returns a vocabulary identical to Default() except for the
// relation weights — WithWeightsOver applied to the seed vocabulary. See that
// function for the contract; this form exists so a caller with no corpus in
// hand (the package's own tests) can reweight the authored table.
//
// It is the wrong form for anything that has analyzed a corpus: a run reasons
// over a per-corpus vocabulary whose concept leaves are learned (WithConcepts),
// and Default() does not contain them. internal/bench reweights the run's own
// vocabulary through WithWeightsOver, never this.
func WithWeights(overrides map[TermID]float64) (*Ontology, error) {
	return WithWeightsOver(Default(), overrides)
}

// WithWeightsOver returns a vocabulary identical to base except for the
// relation weights: each override replaces that relation's weight, and every
// scored relation not named is scaled uniformly so the total stays exactly 1.0
// — axiom 7 holds by construction, and Validate is still run to prove it.
// Entity, concept and role terms are carried over untouched and in base's own
// declaration order, so a learned concept leaf keeps its place in the taxonomy
// and LCA answers exactly what it answered under base.
//
// This is the measurement seam for the ablation and fitting harness in
// internal/bench: it lets a run score pairs under modified weights without
// touching relations.go, whose committed values remain the one production
// truth. Nothing in the production pipeline calls this. base is the run's own
// vocabulary (Run.Onto) — reweighting Default() instead would hand the scorer a
// taxonomy missing the run's concepts, and every non-identical pair of them
// would lose its shared-ancestor credit.
//
// An empty or nil override map returns base itself — not a scaled copy,
// because the default literals sum a few ulps off 1.0 and scaling by
// 1.0/thatSum would perturb every weight's last bit. Overriding all twelve
// requires the overrides themselves to sum to 1.0, since no remaining mass
// exists to absorb the residual.
func WithWeightsOver(base *Ontology, overrides map[TermID]float64) (*Ontology, error) {
	if len(overrides) == 0 {
		return base, nil
	}
	// Terms returns value copies in declaration order; editing the relation
	// entries in place and rebuilding from the one slice keeps base's order
	// exactly, which a per-kind rebuild would not (WithConcepts appends the
	// concept table last).
	terms := base.Terms()

	idx := make(map[TermID]int, len(terms))
	for i, t := range terms {
		if t.Kind == KindRelation {
			idx[t.ID] = i
		}
	}

	var overridden float64
	for id, w := range overrides {
		i, ok := idx[id]
		if !ok || terms[i].Abstract {
			return nil, fmt.Errorf("ontology: %q is not a scored relation", id)
		}
		if w < 0 {
			return nil, fmt.Errorf("ontology: negative weight %v for %q", w, id)
		}
		overridden += w
	}
	residual := 1.0 - overridden
	if residual < -weightEpsilon {
		return nil, fmt.Errorf("ontology: overridden weights sum to %v, above 1.0", overridden)
	}

	var restSum float64
	for _, t := range terms {
		if t.Kind != KindRelation || t.Abstract {
			continue
		}
		if _, ok := overrides[t.ID]; !ok {
			restSum += t.Weight
		}
	}
	if restSum == 0 && math.Abs(residual) > weightEpsilon {
		return nil, fmt.Errorf("ontology: every relation overridden but weights sum to %v, not 1.0", overridden)
	}

	for i := range terms {
		if terms[i].Kind != KindRelation || terms[i].Abstract {
			continue
		}
		if w, ok := overrides[terms[i].ID]; ok {
			terms[i].Weight = w
		} else if restSum > 0 {
			terms[i].Weight *= residual / restSum
		}
	}

	o := New(terms)
	if errs := o.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("ontology: reweighted vocabulary fails validation: %v", errs[0])
	}
	return o, nil
}

// ScoredRelations returns the IDs of every weight-carrying relation, in
// declaration order — the exact set WithWeightsOver accepts as override keys,
// so an ablation sweep never hardcodes the twelve.
func (o *Ontology) ScoredRelations() []TermID {
	var out []TermID
	for _, id := range o.order {
		if t := o.terms[id]; t.Kind == KindRelation && !t.Abstract {
			out = append(out, id)
		}
	}
	return out
}

package ontology

import (
	"fmt"
	"math"
)

// WithWeights returns a vocabulary identical to Default() except for the
// relation weights: each override replaces that relation's weight, and every
// scored relation not named is scaled uniformly so the total stays exactly 1.0
// — axiom 7 holds by construction, and Validate is still run to prove it.
//
// This is the measurement seam for the ablation and fitting harness in
// internal/bench: it lets a run score pairs under modified weights without
// touching relations.go, whose committed values remain the one production
// truth. Nothing in the production pipeline calls this.
//
// An empty or nil override map returns Default() itself — not a scaled copy,
// because the default literals sum a few ulps off 1.0 and scaling by
// 1.0/thatSum would perturb every weight's last bit. Overriding all twelve
// requires the overrides themselves to sum to 1.0, since no remaining mass
// exists to absorb the residual.
func WithWeights(overrides map[TermID]float64) (*Ontology, error) {
	if len(overrides) == 0 {
		return Default(), nil
	}
	terms := make([]Term, len(relationTerms))
	copy(terms, relationTerms)

	idx := make(map[TermID]int, len(terms))
	for i, t := range terms {
		idx[t.ID] = i
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
		if t.Abstract {
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
		if terms[i].Abstract {
			continue
		}
		if w, ok := overrides[terms[i].ID]; ok {
			terms[i].Weight = w
		} else if restSum > 0 {
			terms[i].Weight *= residual / restSum
		}
	}

	o := New(entityTerms, terms, conceptTerms, roleTerms)
	if errs := o.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("ontology: reweighted vocabulary fails validation: %v", errs[0])
	}
	return o, nil
}

// ScoredRelations returns the IDs of every weight-carrying relation, in
// declaration order — the exact set WithWeights accepts as override keys, so
// an ablation sweep never hardcodes the twelve.
func (o *Ontology) ScoredRelations() []TermID {
	var out []TermID
	for _, id := range o.order {
		if t := o.terms[id]; t.Kind == KindRelation && !t.Abstract {
			out = append(out, id)
		}
	}
	return out
}

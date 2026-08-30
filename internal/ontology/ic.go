package ontology

import "math"

// IC holds the information content of every concept term, derived from how
// often each tag occurs in the analysed corpus: IC(c) = ln(P(c)⁻¹), with P(c)
// the probability that a randomly drawn tag occurrence falls under c.
//
// This is what makes relatedness corpus-aware. Under pure Wu-Palmer every leaf
// is equally informative, but on a real codebase some tags are near-universal —
// two functions sharing one says almost nothing — while sharing a rare tag is
// strong evidence. IC is recomputed from the corpus on every run: fully
// offline, no state carried between runs, and deterministic because the same
// tree always yields the same counts.
type IC struct {
	of       map[TermID]float64
	unknown  float64 // IC assigned to terms outside the taxonomy (pseudo-count 1)
	rootFreq float64
}

// NewCorpusIC computes information content from tag occurrence counts, keyed by
// concept term ID. counts is typically "number of units whose Patterns contains
// the tag"; keys that are not concrete concept leaves are ignored.
//
// Add-one smoothing: every concrete leaf gets a pseudo-count, so no term ever
// has zero probability and every leaf's IC is strictly positive. Frequencies
// propagate to ancestors — freq(c) sums the smoothed counts of every concrete
// leaf at or below c — so the root's frequency is the whole corpus and its IC
// is exactly zero, which is what makes cross-branch Lin similarity exactly
// zero, mirroring Wu-Palmer's root guard.
//
// Determinism: all accumulation is integer, iterating terms in declaration
// order; the only floating-point step is one math.Log per term.
func NewCorpusIC(o *Ontology, counts map[TermID]int) *IC {
	mass := make(map[TermID]float64, len(counts))
	for id, c := range counts {
		mass[id] = float64(c)
	}
	return NewCorpusICMass(o, mass)
}

// NewCorpusICMass is the graded form: mass is the summed membership confidence
// per concept rather than a member count, which is what a learned lexicon
// produces. Everything else is identical — add-one smoothing, ancestor
// accumulation, a root at exactly zero.
//
// Accumulation is floating point now rather than integer, so it is done in
// declaration order (TermsOfKind's own order) and never over a map, because a
// float sum whose order varies is a report that varies.
func NewCorpusICMass(o *Ontology, mass map[TermID]float64) *IC {
	freq := make(map[TermID]float64)
	rootFreq := 0.0
	for _, term := range o.TermsOfKind(KindConcept) {
		if term.Abstract {
			continue
		}
		f := mass[term.ID] + 1 // add-one smoothing
		rootFreq += f
		freq[term.ID] += f
		for _, anc := range o.Ancestors(term.ID) {
			freq[anc] += f
		}
	}
	if rootFreq == 0 {
		// Degenerate ontology with no concrete concepts; nothing to weight.
		return &IC{of: map[TermID]float64{}, unknown: 0, rootFreq: 0}
	}

	ic := &IC{
		of:       make(map[TermID]float64, len(freq)),
		unknown:  math.Log(rootFreq),
		rootFreq: rootFreq,
	}
	for _, term := range o.TermsOfKind(KindConcept) {
		f := freq[term.ID]
		if f <= 0 {
			continue
		}
		ic.of[term.ID] = math.Log(rootFreq / f)
	}
	return ic
}

// Of returns a term's information content. The root is exactly 0. A term the
// taxonomy does not know gets the maximum: a single pseudo-occurrence, i.e.
// ln(rootFreq) — an unregistered tag is treated as maximally rare rather than
// silently weightless, so a future tagger rule added before its concept term
// still carries weight when matched against itself.
func (ic *IC) Of(id TermID) float64 {
	if v, ok := ic.of[id]; ok {
		return v
	}
	return ic.unknown
}

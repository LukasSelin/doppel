package comparator

import "math"

// BlendMode is how ViewBlend combines the concept views into the exhibits
// slot of the composite.
type BlendMode int

const (
	// BlendWeighted is Σ w·view over the available views, normalised by the
	// available weights. The zero ViewBlend — no weights at all — is the
	// corpus view alone, which is what the exhibits slot was before views
	// existed and what every measurement is a difference from.
	BlendWeighted BlendMode = iota
	// BlendGeometric is the n-th root of the product of the available views:
	// a pair must look related from every angle, and one zero is a zero.
	BlendGeometric
	// BlendMax is the most generous view: any angle that sees kinship counts.
	BlendMax
)

// ViewBlend decides what the exhibits slot of OverlapScore reads. Views are
// always computed and reported unblended; this is the one place they are
// combined, and only because the composite needs one number per relation.
type ViewBlend struct {
	Shape, Corpus, Feature float64 // BlendWeighted weights; ignored by the other modes
	Mode                   BlendMode
}

// IsZero reports whether the blend is the incumbent: the corpus view alone.
func (b ViewBlend) IsZero() bool {
	return b.Mode == BlendWeighted && b.Shape == 0 && b.Corpus == 0 && b.Feature == 0
}

// Apply combines the views. "Available" means every view that was measured:
// shape and corpus always, feature only when the scorer carried a vocabulary.
// A view that was not measured is not a zero — leaving it out is what keeps
// a free comparator (no vocabulary, no IC) reading exactly as it always has.
// Every arithmetic path is a fixed sequence of operations in a fixed order,
// so the same views always give the same bits.
func (b ViewBlend) Apply(v ConceptViews) float64 {
	if b.IsZero() {
		return v.Corpus
	}
	switch b.Mode {
	case BlendGeometric:
		prod := v.Shape * v.Corpus
		n := 2.0
		if v.HasFeature {
			prod *= v.Feature
			n = 3
		}
		if prod <= 0 {
			return 0
		}
		return math.Pow(prod, 1/n)
	case BlendMax:
		m := v.Shape
		if v.Corpus > m {
			m = v.Corpus
		}
		if v.HasFeature && v.Feature > m {
			m = v.Feature
		}
		return m
	}
	sum := b.Shape*v.Shape + b.Corpus*v.Corpus
	norm := b.Shape + b.Corpus
	if v.HasFeature {
		sum += b.Feature * v.Feature
		norm += b.Feature
	}
	if norm <= 0 {
		return v.Corpus
	}
	return sum / norm
}

// Options are the comparator's knobs. Like retriever's and lexicon's, none is
// a flag: what the exhibits slot reads is a property of the tool, measured in
// internal/bench, not an operating point a user tunes.
type Options struct {
	Exhibits ViewBlend
}

// DefaultOptions is the production blend: the corpus view alone, which is
// what the exhibits slot read before the views existed.
//
// That is a measured result, not an omission. internal/bench's
// TestViewsBlend scores every candidate — the weighted simplex in 0.1 steps,
// the geometric mean, the max — against the cobra labels, and none is
// distinguishable from the corpus view: merge mean 5.3 (6/6) under every one
// of them, refactor 12.8 against 12.9, false-positive 50.5 against 51.0, the
// largest movement anywhere one pair by one rank. Under the stated selection
// rule (zero violations, merge not worse, false-positive not lower, then best
// refactor mean, ties to the least departure from the incumbent) the corpus
// view wins. It is a function rather than a constant so the day a second
// labeled corpus can tell the blends apart, adopting one is a one-line change
// whose measurement already exists.
func DefaultOptions() Options {
	return Options{}
}

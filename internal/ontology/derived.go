package ontology

import "sort"

// DerivedConcept is one learned concept asking to be placed in the taxonomy.
//
// The concept leaves in concepts.go were fourteen assertions about what any
// codebase does; internal/lexicon derives the leaves from the corpus instead.
// What survives that change is the *interior* of the tree — io_operation,
// data_store_access, data_transformation — which was never a claim about a
// codebase, only about how kinds of work relate. Wu-Palmer depth, Lin
// similarity and the concept retrieval channel's ancestor postings all read
// that interior, so a derived vocabulary hangs from it rather than replacing
// it.
type DerivedConcept struct {
	ID string // the learned name, e.g. "sql.Open+QueryRow"

	// Seed is the built-in leaf this concept grew from, if any. The concept
	// attaches under that leaf's *parent*, inheriting its place and depth:
	// a concept seeded from db_access is a kind of data_store_access, whatever
	// this corpus turned out to mean by it.
	Seed TermID

	// AnchorSeed places an emergent concept — one no seed accounts for — beside
	// the seeded concept it shares the most vocabulary with. Empty attaches it
	// directly under the concept root, which is the honest answer when it
	// resembles nothing the taxonomy names.
	AnchorSeed TermID

	Def string // one-sentence definition, rendered by doppel ontology
}

// DerivedConceptTerms builds the concept table for a learned lexicon: every
// abstract term of the base tree, unchanged, plus one concrete leaf per derived
// concept. base is the vocabulary the interior comes from, normally Default().
//
// Terms are emitted in a fixed order — the base's own declaration order for the
// interior, then derived concepts by ID — because Ontology.order is the only
// iteration order the package exposes and a report must not depend on how a
// lexicon happened to be assembled.
func DerivedConceptTerms(base *Ontology, derived []DerivedConcept) []Term {
	var out []Term
	for _, t := range base.TermsOfKind(KindConcept) {
		if t.Abstract {
			out = append(out, t)
		}
	}

	sorted := append([]DerivedConcept(nil), derived...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	for _, d := range sorted {
		out = append(out, Term{
			ID:     TermID(d.ID),
			Kind:   KindConcept,
			Label:  d.ID,
			Def:    d.Def,
			Parent: derivedParent(base, d),
		})
	}
	return out
}

// derivedParent resolves where a learned concept hangs. A seed's own parent
// when it has one, the anchor's parent when it does not, and the concept root
// as the floor — never a concrete leaf, because the built-in leaves are gone
// from the derived tree and a parent pointer into one would leave the term
// unreachable from the root, which Validate would reject and Wu-Palmer would
// silently score as unrelated to everything.
func derivedParent(base *Ontology, d DerivedConcept) TermID {
	for _, seed := range []TermID{d.Seed, d.AnchorSeed} {
		if seed == "" {
			continue
		}
		t, ok := base.Get(seed)
		if !ok || t.Parent == "" {
			continue
		}
		if p, ok := base.Get(t.Parent); ok && p.Abstract {
			return t.Parent
		}
	}
	return ConConcept
}

// WithConcepts returns a vocabulary identical to base except for its concept
// tree. Entity, relation and role terms — including the relation weights the
// comparator's scoring table lives in — are carried over untouched, so a
// per-corpus concept vocabulary cannot disturb anything else the ontology
// asserts.
func WithConcepts(base *Ontology, concepts []Term) *Ontology {
	var others []Term
	for _, t := range base.Terms() {
		if t.Kind != KindConcept {
			others = append(others, t)
		}
	}
	return New(others, concepts)
}

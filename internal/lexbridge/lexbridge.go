// Package lexbridge carries a learned lexicon across to the ontology.
//
// lexicon may not import ontology — it learns names, it does not reason about
// a vocabulary — so something has to translate a run's learned concepts into
// taxonomy placements and a feature side table. cmd and internal/bench both
// need that translation, and each used to carry its own copy; doppel found the
// two as exact clones of each other. This package is the one definition.
package lexbridge

import (
	"github.com/LukasSelin/doppel/internal/lexicon"
	"github.com/LukasSelin/doppel/internal/ontology"
)

// DerivedConcepts translates the learned lexicon into taxonomy placements.
//
// A seeded concept hangs where its seed's leaf hung — a concept grown from
// db_access is a kind of data_store_access, whatever this corpus turned out to
// mean by it — and an emergent one hangs beside whichever seeded concept it
// most resembles, or from the root when it resembles none. That is the whole of
// what the authored vocabulary still asserts: the shape of the interior, and
// where a learned leaf plausibly belongs in it.
func DerivedConcepts(lex *lexicon.Model) []ontology.DerivedConcept {
	concepts := lex.Concepts()
	out := make([]ontology.DerivedConcept, len(concepts))
	for i, c := range concepts {
		out[i] = ontology.DerivedConcept{
			ID:         c.ID,
			Seed:       ontology.TermID(c.Seed),
			AnchorSeed: ontology.TermID(c.Anchor),
			Def:        c.Definition(),
		}
	}
	return out
}

// Vocabulary carries each learned concept's feature vocabulary across to the
// ontology's side table, so the comparator's feature view can read what two
// concepts are made of.
func Vocabulary(lex *lexicon.Model) *ontology.Vocabulary {
	concepts := lex.Concepts()
	entries := make([]ontology.VocabularyEntry, len(concepts))
	for i, c := range concepts {
		feats := make([]ontology.WeightedFeature, len(c.Features))
		for j, f := range c.Features {
			feats[j] = ontology.WeightedFeature{Name: f.Name, Weight: f.Weight, Opaque: lexicon.Opaque(f.Name)}
		}
		entries[i] = ontology.VocabularyEntry{ID: ontology.TermID(c.ID), Features: feats}
	}
	return ontology.NewVocabulary(entries)
}

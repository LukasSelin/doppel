package retriever

import (
	"sort"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// conceptIndex is the concept channel: an inverted index over the units'
// learned concepts and their non-root taxonomy ancestors. Ancestor postings are
// what let two functions with no concept in common meet under a shared parent.
// Expansion is for enumeration only; pair evidence is always the graded shared
// information of the two membership sets (Σ min(conf)·IC(LCS) over the scorer's
// best matching), so a pairing that only meets at a shallow ancestor earns only
// that ancestor's small IC — and a pairing where either side barely carries the
// concept earns only as much as that side claims.
//
// Postings are unweighted: a unit posts under a concept it is a member of at
// all, so the df window keeps meaning "how many functions carry this", the
// quantity the cap is stated in. Confidence bends the evidence, not the index. This channel has
// no similarity floor and no size gate — admitting structurally dissimilar
// pairs with informative shared meaning is its entire purpose.
//
// A BelowFloor membership does not post. It is a membership the unit did not
// earn — the concept explains more of it than any other does, but not enough —
// and it reaches this channel the way it reaches every other consumer that
// counts rather than weights: not at all. It still contributes to a pair's
// *evidence* if some other posting proposed the pair, because that arithmetic
// is graded and a barely-held membership earns barely anything. Proposing is
// the boolean act; scoring is the graded one.
type conceptIndex struct {
	scorer   *ontology.Scorer
	concepts [][]parser.Concept        // per unit: its memberships, as learned
	expanded [][]ontology.TermID       // per unit: concepts + non-root ancestors, sorted unique
	postings map[ontology.TermID][]int // expanded term → unit indices, ascending
	mass     map[pairKey]float64       // memoized shared information per pair
}

func buildConceptIndex(units []parser.CodeUnit, onto *ontology.Ontology,
	scorer *ontology.Scorer, opt Options) *conceptIndex {

	x := &conceptIndex{
		scorer:   scorer,
		concepts: make([][]parser.Concept, len(units)),
		expanded: make([][]ontology.TermID, len(units)),
		postings: make(map[ontology.TermID][]int),
		mass:     make(map[pairKey]float64),
	}
	for i := range units {
		x.concepts[i] = units[i].Concepts
		if len(units[i].Concepts) == 0 {
			continue
		}
		seen := make(map[ontology.TermID]bool)
		for _, c := range units[i].Concepts {
			if c.BelowFloor {
				continue
			}
			id := ontology.TermID(c.ID)
			seen[id] = true
			for _, anc := range onto.Ancestors(id) {
				// The concept root carries zero IC and would post every
				// tagged unit into one bucket; skip it.
				if onto.Depth(anc) == 0 {
					continue
				}
				seen[anc] = true
			}
		}
		terms := make([]ontology.TermID, 0, len(seen))
		for id := range seen {
			terms = append(terms, id)
		}
		sort.Slice(terms, func(a, b int) bool { return terms[a] < terms[b] })
		x.expanded[i] = terms
		for _, id := range terms {
			x.postings[id] = append(x.postings[id], i)
		}
	}
	return x
}

// admitPairs runs per-function retrieval: every unit posting under a shared
// term (leaf or ancestor) with a posting list inside the df window is a
// neighbor candidate; candidates are scored by raw shared information and the
// top ChannelK by (mass desc, idx asc) are admitted. Zero-mass pairs — sets
// that meet only cross-branch — are never admitted.
func (x *conceptIndex) admitPairs(opt Options) []pairKey {
	var pairs []pairKey
	for a := range x.expanded {
		pairs = append(pairs, x.admitFor(a, opt)...)
	}
	return pairs
}

// admitFor is one function's turn of the admitPairs loop, factored out so a
// single probe unit can be retrieved without the all-pairs pass.
func (x *conceptIndex) admitFor(a int, opt Options) []pairKey {
	if len(x.expanded[a]) == 0 {
		return nil
	}
	nbrSet := make(map[int]bool)
	for _, term := range x.expanded[a] {
		posting := x.postings[term]
		if len(posting) < 2 || len(posting) > opt.MaxConceptDF {
			continue
		}
		for _, b := range posting {
			if b != a {
				nbrSet[b] = true
			}
		}
	}
	if len(nbrSet) == 0 {
		return nil
	}
	cand := make([]int, 0, len(nbrSet))
	for b := range nbrSet {
		cand = append(cand, b)
	}
	sort.Ints(cand)
	neighbors := make([]neighborMass, 0, len(cand))
	for _, b := range cand {
		if mass := x.sharedMass(a, b); mass > 0 {
			neighbors = append(neighbors, neighborMass{idx: b, mass: mass})
		}
	}
	var pairs []pairKey
	for _, nb := range topK(neighbors, opt.ChannelK) {
		pairs = append(pairs, orderPair(a, nb.idx))
	}
	return pairs
}

// sharedMass is the definitive concept evidence for a pair: the shared
// information of the two membership sets, each concept counted only as far as
// the weaker side asserts it, memoized per unordered pair.
func (x *conceptIndex) sharedMass(a, b int) float64 {
	k := orderPair(a, b)
	if m, ok := x.mass[k]; ok {
		return m
	}
	m, _ := x.scorer.SharedInformationW(
		concepter.Graded(x.concepts[k[0]]), concepter.Graded(x.concepts[k[1]]))
	x.mass[k] = m
	return m
}

package retriever

import (
	"sort"

	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// conceptIndex is the concept channel: an inverted index over tagger tags and
// their non-root taxonomy ancestors. Ancestor postings are what let a
// db_access-only function meet a caching-only one — they share no leaf, but
// both post under data_store_access. Expansion is for enumeration only; pair
// evidence is always the raw shared information of the two leaf tag sets
// (Σ IC(LCS) over the scorer's best matching), so a pairing that only meets
// at a shallow ancestor earns only that ancestor's small IC. This channel has
// no similarity floor and no size gate — admitting structurally dissimilar
// pairs with informative shared meaning is its entire purpose.
type conceptIndex struct {
	scorer   *ontology.Scorer
	patterns [][]string                // per unit: its leaf tags, as tagged
	expanded [][]ontology.TermID       // per unit: tags + non-root ancestors, sorted unique
	postings map[ontology.TermID][]int // expanded term → unit indices, ascending
	mass     map[pairKey]float64       // memoized SharedInformation per pair
}

func buildConceptIndex(units []parser.CodeUnit, onto *ontology.Ontology,
	scorer *ontology.Scorer, opt Options) *conceptIndex {

	x := &conceptIndex{
		scorer:   scorer,
		patterns: make([][]string, len(units)),
		expanded: make([][]ontology.TermID, len(units)),
		postings: make(map[ontology.TermID][]int),
		mass:     make(map[pairKey]float64),
	}
	for i := range units {
		x.patterns[i] = units[i].Patterns
		if len(units[i].Patterns) == 0 {
			continue
		}
		seen := make(map[ontology.TermID]bool)
		for _, tag := range units[i].Patterns {
			id := ontology.TermID(tag)
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

// sharedMass is the definitive concept evidence for a pair: the raw shared
// information of the two leaf tag sets, memoized per unordered pair.
func (x *conceptIndex) sharedMass(a, b int) float64 {
	k := orderPair(a, b)
	if m, ok := x.mass[k]; ok {
		return m
	}
	m, _ := x.scorer.SharedInformation(x.patterns[k[0]], x.patterns[k[1]])
	x.mass[k] = m
	return m
}

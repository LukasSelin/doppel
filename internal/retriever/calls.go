package retriever

import (
	"math"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
)

// callIndex is the call channel: an inverted index over resolved call tokens
// with per-token IDF. Two token sources, both fully resolved:
//
//   - repo-internal callees from the call graph (qualified names), and
//   - external calls whose selector receiver is an import binding of the
//     calling file, keyed by full import path ("database/sql.Open") so two
//     packages importing the same API meet on the same token.
//
// Bare names and variable-receiver method calls are never tokens — unresolved
// matching is exactly what the resolved call graph exists to avoid. Ubiquity
// handles itself: fmt.Sprintf-scale tokens exceed MaxCallDF and drop out of
// the index entirely, mid-frequency helpers get smoothly small IDF, and rare
// API combinations carry the mass.
type callIndex struct {
	surviving [][]string         // per unit: tokens that survived the df cap, ascending
	idf       map[string]float64 // ln(nUnits / df) for surviving tokens
	postings  map[string][]int   // surviving token → unit indices, ascending
	energy    []float64          // per unit: Σ idf over surviving tokens (informative call energy)
}

func buildCallIndex(units []parser.CodeUnit, g *concepter.Graph, opt Options) *callIndex {
	x := &callIndex{
		surviving: make([][]string, len(units)),
		idf:       make(map[string]float64),
		postings:  make(map[string][]int),
	}

	internal := concepter.QualifiedNames(units)

	tokens := make([][]string, len(units))
	df := make(map[string]int)
	for i := range units {
		tokens[i] = concepter.CallTokens(units[i], g, internal)
		for _, t := range tokens[i] {
			df[t]++
		}
	}

	for i := range units {
		var surv []string
		for _, t := range tokens[i] {
			if n := df[t]; n >= 2 && n <= opt.MaxCallDF {
				surv = append(surv, t)
				x.postings[t] = append(x.postings[t], i)
			}
		}
		x.surviving[i] = surv
	}
	for t, n := range df {
		if n >= 2 && n <= opt.MaxCallDF {
			x.idf[t] = math.Log(float64(len(units)) / float64(n))
		}
	}
	// Informative call energy per unit, mirroring the shape index: the Dice
	// over these energies (callSim) says how much of two functions' call
	// behavior is mutual — for a pair of tests, whether they exercise the
	// same machinery or merely share a driver skeleton.
	x.energy = make([]float64, len(units))
	for i := range units {
		var e float64
		for _, t := range x.surviving[i] { // ascending: fixed float order
			e += x.idf[t]
		}
		x.energy[i] = e
	}
	return x
}

// callSim is the call-channel Dice: 2·sharedMass/(E_A+E_B) over informative
// call energy, 0 on a zero denominator.
func (x *callIndex) callSim(a, b int, sharedMass float64) float64 {
	denom := x.energy[a] + x.energy[b]
	if denom <= 0 {
		return 0
	}
	return 2 * sharedMass / denom
}

// admitPairs runs per-function retrieval over shared surviving tokens:
// accumulate Σ idf per neighbor, keep the top ChannelK by (mass desc, idx
// asc). No similarity floor and no size gate — a syntactically alien pair
// sharing rare resolved calls is exactly what this channel exists to admit.
func (x *callIndex) admitPairs(opt Options) []pairKey {
	var pairs []pairKey
	for a := range x.surviving {
		pairs = append(pairs, x.admitFor(a, opt)...)
	}
	return pairs
}

// admitFor is one function's turn of the admitPairs loop, factored out so a
// single probe unit can be retrieved without the all-pairs pass.
func (x *callIndex) admitFor(a int, opt Options) []pairKey {
	if len(x.surviving[a]) == 0 {
		return nil
	}
	acc := make(map[int]float64)
	for _, t := range x.surviving[a] {
		w := x.idf[t]
		for _, b := range x.postings[t] {
			if b != a {
				acc[b] += w
			}
		}
	}
	neighbors := make([]neighborMass, 0, len(acc))
	for b, mass := range acc {
		// A token in every unit has idf 0; zero shared mass is zero
		// evidence, not a candidate.
		if mass > 0 {
			neighbors = append(neighbors, neighborMass{idx: b, mass: mass})
		}
	}
	var pairs []pairKey
	for _, nb := range topK(neighbors, opt.ChannelK) {
		pairs = append(pairs, orderPair(a, nb.idx))
	}
	return pairs
}

// sharedMass is the definitive call evidence for a pair: Σ idf over the
// intersection of the two surviving token sets, summed in ascending token
// order.
func (x *callIndex) sharedMass(a, b int) float64 {
	sa, sb := x.surviving[a], x.surviving[b]
	var mass float64
	i, j := 0, 0
	for i < len(sa) && j < len(sb) {
		switch {
		case sa[i] < sb[j]:
			i++
		case sa[i] > sb[j]:
			j++
		default:
			mass += x.idf[sa[i]]
			i++
			j++
		}
	}
	return mass
}

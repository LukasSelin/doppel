package retriever

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"sort"

	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

// largeBucketSize is the diagnostic threshold for "common structural idiom":
// an exact pattern-multiset identity bucket with more members than this is
// counted in Stats.LargeBuckets. It does not gate anything — suppression is
// the df cap's job — it exists so the stderr summary can show how much idiom
// mass the corpus carries.
const largeBucketSize = 20

// shapeIndex is the structural channel: an inverted index over the
// multi-level trophic pattern multiset (fingerprint.Pattern — tokens,
// expressions, actions, motif chains) with per-pattern IDF from presence df.
// A pattern present in more than MaxPatternDF eligible units (or in only
// one) carries no evidence, so corpus-wide idioms drop out entirely. Shared
// evidence between two functions is Σ idf·min(count) over the shared
// multiset — the shared structural energy: a match has weight because of
// what it shares, not the fraction that matches.
type shapeIndex struct {
	surviving    [][]survPattern    // per unit: patterns that survived the df cap, ascending hash
	idf          map[uint64]float64 // ln(nEligible / presence-df) for surviving patterns
	postings     map[uint64][]posting
	energy       []float64            // per unit: Σ idf·count over surviving patterns
	meta         map[uint64]chainMeta // level>=2 surviving patterns: render for explanations
	suppressed   int
	largeBuckets int
}

type survPattern struct {
	hash  uint64
	count uint16
}

type posting struct {
	idx   int
	count uint16
}

type chainMeta struct {
	level  uint8
	render string
}

// SharedPattern is one shared high-level structure between two functions —
// the explanation of where their shared energy comes from.
type SharedPattern struct {
	Level  int
	Energy float64 // idf · min(count)
	Render string
}

func buildShapeIndex(units []parser.CodeUnit, opt Options) *shapeIndex {
	x := &shapeIndex{
		surviving: make([][]survPattern, len(units)),
		idf:       make(map[uint64]float64),
		postings:  make(map[uint64][]posting),
		energy:    make([]float64, len(units)),
		meta:      make(map[uint64]chainMeta),
	}

	eligible := make([]bool, len(units))
	nEligible := 0
	for i := range units {
		fp := units[i].Fingerprint
		if fp.Nodes >= opt.MinNodes && fp.Nodes > 0 {
			eligible[i] = true
			nEligible++
		}
	}

	df := make(map[uint64]int)
	buckets := make(map[uint64]int)
	for i := range units {
		if !eligible[i] {
			continue
		}
		for _, p := range units[i].Fingerprint.Patterns {
			df[p.Hash]++ // presence df: Patterns is deduped per unit
		}
		buckets[patternBucket(units[i].Fingerprint.Patterns)]++
	}
	for _, members := range buckets {
		if members > largeBucketSize {
			x.largeBuckets++
		}
	}

	for i := range units {
		if !eligible[i] {
			continue
		}
		patterns := units[i].Fingerprint.Patterns
		var surv []survPattern
		for _, p := range patterns {
			if n := df[p.Hash]; n >= 2 && n <= opt.MaxPatternDF {
				surv = append(surv, survPattern{hash: p.Hash, count: p.Count})
				x.postings[p.Hash] = append(x.postings[p.Hash], posting{idx: i, count: p.Count})
				if p.Level >= fingerprint.LevelAction && p.Render != "" {
					if _, ok := x.meta[p.Hash]; !ok {
						x.meta[p.Hash] = chainMeta{level: p.Level, render: p.Render}
					}
				}
			}
		}
		x.surviving[i] = surv
		if len(surv) == 0 && len(patterns) > 0 {
			x.suppressed++
		}
	}
	for h, n := range df {
		if n >= 2 && n <= opt.MaxPatternDF {
			x.idf[h] = math.Log(float64(nEligible) / float64(n))
		}
	}
	// A unit's energy is its cap-surviving (informative) structure, so
	// trophic similarity is the Dice over informative energy: exact clones of
	// a rich function score 1.0, a fully-capped idiom bucket scores 0/0 = 0,
	// and everything between reads as the fraction of informative structure
	// the pair shares. Common idioms count on neither side by construction.
	for i := range units {
		var e float64
		for _, sp := range x.surviving[i] { // ascending hash: fixed float order
			e += x.idf[sp.hash] * float64(sp.count)
		}
		x.energy[i] = e
	}
	return x
}

// patternBucket hashes the full pattern-multiset identity (ordered
// hash+count pairs — Patterns is already canonically sorted), for the
// large-bucket diagnostic.
func patternBucket(patterns []fingerprint.Pattern) uint64 {
	h := fnv.New64a()
	var buf [10]byte
	for _, p := range patterns {
		binary.LittleEndian.PutUint64(buf[:8], p.Hash)
		binary.LittleEndian.PutUint16(buf[8:], p.Count)
		_, _ = h.Write(buf[:])
	}
	return h.Sum64()
}

// admitPairs runs per-function retrieval: accumulate shared energy against
// every unit sharing a surviving pattern, then probe neighbors in
// (mass desc, idx asc) order with the exact fingerprint score, admitting up
// to ChannelK pairs at or above Threshold. Probing stops after
// 4*ChannelK scored neighbors either way — inside a dense sub-threshold
// cluster a function may fill fewer than K slots, which is accepted: those
// clusters are precisely the junk being suppressed, and an unbounded probe
// would degenerate back toward O(n²).
func (x *shapeIndex) admitPairs(sim *simCache, opt Options) []pairKey {
	var pairs []pairKey
	for a := range x.surviving {
		if len(x.surviving[a]) == 0 {
			continue
		}
		acc := make(map[int]float64)
		for _, sp := range x.surviving[a] {
			w := x.idf[sp.hash]
			for _, po := range x.postings[sp.hash] {
				if po.idx != a {
					acc[po.idx] += w * float64(minCount(sp.count, po.count))
				}
			}
		}
		neighbors := make([]neighborMass, 0, len(acc))
		for b, mass := range acc {
			// A pattern in every eligible unit has idf 0; zero shared mass is
			// zero evidence, not a candidate.
			if mass > 0 {
				neighbors = append(neighbors, neighborMass{idx: b, mass: mass})
			}
		}
		neighbors = topK(neighbors, 0) // full deterministic order; probe bound below
		maxProbe := 4 * opt.ChannelK
		admitted, probed := 0, 0
		for _, nb := range neighbors {
			if admitted >= opt.ChannelK || probed >= maxProbe {
				break
			}
			probed++
			if sim.get(a, nb.idx).Score >= opt.Threshold {
				pairs = append(pairs, orderPair(a, nb.idx))
				admitted++
			}
		}
	}
	return pairs
}

func minCount(a, b uint16) uint16 {
	if a < b {
		return a
	}
	return b
}

// pairEvidence computes the definitive structural view of one pair in a
// single sorted-intersection pass: shared energy (Σ idf·min(count)), trophic
// similarity (weighted Dice, 2·shared/(E_A+E_B)), and the highest-energy
// shared high-level chains as the explanation. Chains sort by (energy desc,
// level desc, render asc) — byte-stable ties — and truncate to chainTopN.
func (x *shapeIndex) pairEvidence(a, b, chainTopN int) (float64, float64, []SharedPattern) {
	sa, sb := x.surviving[a], x.surviving[b]
	var mass float64
	var chains []SharedPattern
	i, j := 0, 0
	for i < len(sa) && j < len(sb) {
		switch {
		case sa[i].hash < sb[j].hash:
			i++
		case sa[i].hash > sb[j].hash:
			j++
		default:
			e := x.idf[sa[i].hash] * float64(minCount(sa[i].count, sb[j].count))
			mass += e
			if m, ok := x.meta[sa[i].hash]; ok && e > 0 {
				chains = append(chains, SharedPattern{Level: int(m.level), Energy: e, Render: m.render})
			}
			i++
			j++
		}
	}
	var trophic float64
	if denom := x.energy[a] + x.energy[b]; denom > 0 {
		trophic = 2 * mass / denom
	}
	sort.SliceStable(chains, func(p, q int) bool {
		if chains[p].Energy != chains[q].Energy {
			return chains[p].Energy > chains[q].Energy
		}
		if chains[p].Level != chains[q].Level {
			return chains[p].Level > chains[q].Level
		}
		return chains[p].Render < chains[q].Render
	})
	if chainTopN > 0 && len(chains) > chainTopN {
		chains = chains[:chainTopN]
	}
	return mass, trophic, chains
}

package retriever

import (
	"encoding/binary"
	"hash/fnv"
	"math"

	"github.com/lukse/doppel/internal/parser"
)

// largeBucketSize is the diagnostic threshold for "common structural idiom":
// an exact-fingerprint identity bucket with more members than this is counted
// in Stats.LargeBuckets. It does not gate anything — suppression is the df
// cap's job — it exists so the stderr summary can show how much idiom mass
// the corpus carries.
const largeBucketSize = 20

// shapeIndex is the structural channel: an inverted index over shingle hashes
// with per-shingle IDF. A shingle present in more than MaxShingleDF eligible
// units (or in only one) is dropped entirely, so corpus-wide idioms — the 130
// identical Error() bodies — contribute no postings and no evidence. What
// survives is exactly the rare shared structure, and Σ idf over it is the
// channel's evidence mass: tiny bodies have few shingles and common shapes
// have low-IDF ones, so mass encodes rarity and non-triviality at once.
type shapeIndex struct {
	surviving    [][]uint64         // per unit: shingles that survived the df cap, ascending
	idf          map[uint64]float64 // ln(nEligible / df) for surviving shingles
	postings     map[uint64][]int   // surviving shingle → unit indices, ascending
	suppressed   int
	largeBuckets int
}

func buildShapeIndex(units []parser.CodeUnit, opt Options) *shapeIndex {
	x := &shapeIndex{
		surviving: make([][]uint64, len(units)),
		idf:       make(map[uint64]float64),
		postings:  make(map[uint64][]int),
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
		for _, s := range units[i].Fingerprint.Shingles {
			df[s]++
		}
		buckets[fingerprintBucket(units[i].Fingerprint.Shingles)]++
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
		shingles := units[i].Fingerprint.Shingles
		var surv []uint64
		for _, s := range shingles {
			if n := df[s]; n >= 2 && n <= opt.MaxShingleDF {
				surv = append(surv, s)
				x.postings[s] = append(x.postings[s], i)
			}
		}
		x.surviving[i] = surv
		if len(surv) == 0 && len(shingles) > 0 {
			x.suppressed++
		}
	}
	for s, n := range df {
		if n >= 2 && n <= opt.MaxShingleDF {
			x.idf[s] = math.Log(float64(nEligible) / float64(n))
		}
	}
	return x
}

// fingerprintBucket hashes the full shingle multiset identity, for the
// large-bucket diagnostic. Shingles are already sorted and deduped, so the
// hash is canonical.
func fingerprintBucket(shingles []uint64) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	for _, s := range shingles {
		binary.LittleEndian.PutUint64(buf[:], s)
		h.Write(buf[:])
	}
	return h.Sum64()
}

// admitPairs runs per-function retrieval: accumulate shared-IDF mass against
// every unit sharing a surviving shingle, then probe neighbors in
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
		for _, s := range x.surviving[a] {
			w := x.idf[s]
			for _, b := range x.postings[s] {
				if b != a {
					acc[b] += w
				}
			}
		}
		neighbors := make([]neighborMass, 0, len(acc))
		for b, mass := range acc {
			// A shingle in every eligible unit has idf 0; zero shared mass is
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

// sharedMass is the definitive structural evidence for a pair: Σ idf over the
// intersection of the two surviving shingle sets, summed in ascending shingle
// order. Zero for ineligible or fully-suppressed units.
func (x *shapeIndex) sharedMass(a, b int) float64 {
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

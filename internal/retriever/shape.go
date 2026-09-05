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
// an exact label-bag identity bucket with more members than this is counted in
// Stats.LargeBuckets. It does not gate anything — suppression is the df cap's
// job — it exists so the stderr summary can show how much idiom mass the
// corpus carries.
const largeBucketSize = 20

// shapeIndex is the structural channel: an inverted index over the
// Weisfeiler-Lehman label multiset (fingerprint.WLBag — one label per node per
// refinement round, so a body's whole subtree vocabulary at four resolutions)
// with per-label IDF from presence df. A label present in more than MaxLabelDF
// eligible units (or in only one) carries no evidence, so corpus-wide idioms
// drop out entirely. Shared evidence between two functions is Σ idf·min(count)
// over the shared multiset — the shared structural energy: a match has weight
// because of what it shares, not the fraction that matches.
//
// # Why WL labels and not the pattern multiset they replaced
//
// The channel used to index a hand-built five-level pattern hierarchy — token
// n-grams, call and operator shapes, statement renders, loop summaries and
// statement bigrams, def-use role edges. Every level was a separate extractor
// with its own vocabulary, its own idea of what counted as salient, and its
// own way of being wrong about it, and the levels overlapped: an L0 3-gram
// over a return statement and the L2 render of that same return are two
// spellings of one fact, both admitted, both weighted.
//
// A WL bag is the same idea taken to its conclusion. label_h(v) summarises
// everything within h edges below v, so one uniform recurrence produces the
// whole ladder — h=0 is a node kind (the L1 vocabulary), h=1 is a statement
// with its operand shapes (the L2 vocabulary), h=2..3 are guards and loop
// bodies whole (what the L3 motifs approximated) — with no extractor deciding
// what is worth naming. It is also the same multiset the *score* already reads
// since T3, so the channel that retrieves a pair and the number that ranks it
// are computed over one feature set rather than two that can disagree.
//
// What is lost is the render: a pattern's Render string was the hash's own
// serialization, so an explanation could not drift from what was counted. A
// label is a hash of a subtree and has no short faithful name. See
// fingerprint.DescribeLabel for what the explanation block says instead.
type shapeIndex struct {
	surviving    [][]survLabel      // per unit: labels that survived the df cap, ascending label
	idf          map[uint64]float64 // ln(nEligible / presence-df) for surviving labels
	postings     map[uint64][]posting
	energy       []float64 // per unit: Σ idf·count over surviving labels
	suppressed   int
	largeBuckets int
	cap          int // the df cap actually applied
}

// survLabel is one cap-surviving label of one unit, carrying the two facts
// that let it be named — see fingerprint.LabelCount, from which it is copied
// verbatim. Sixteen bytes, so the intersection pass reads two cache lines per
// eight labels exactly as the pattern version did.
type survLabel struct {
	label uint64
	count int32
	h     uint8
	kind  fingerprint.LabelKind
}

type posting struct {
	idx   int
	count int32
}

// SharedLabel is one shared Weisfeiler-Lehman label between two functions —
// the explanation of where their shared energy comes from. Depth is the
// refinement round and Kind the node kind it was computed at; Render is
// fingerprint.DescribeLabel of the two, so a consumer that only prints does
// not need the vocabulary.
type SharedLabel struct {
	Label  uint64 // the label itself: the identity behind the number
	Depth  int    // WL refinement round, 0..3
	Count  int    // min(count) — how many times both bodies carry it
	Energy float64
	Render string
}

func buildShapeIndex(units []parser.CodeUnit, opt Options) *shapeIndex {
	x := &shapeIndex{
		surviving: make([][]survLabel, len(units)),
		idf:       make(map[uint64]float64),
		postings:  make(map[uint64][]posting),
		energy:    make([]float64, len(units)),
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

	x.cap, _ = effectiveCap(opt.MaxLabelDF, nEligible, opt.MinIDF)
	df := make(map[uint64]int)
	buckets := make(map[uint64]int)
	for i := range units {
		if !eligible[i] {
			continue
		}
		for _, lc := range units[i].Fingerprint.WL {
			df[lc.Label]++ // presence df: a WL bag has each label once
		}
		buckets[labelBucket(units[i].Fingerprint.WL)]++
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
		bag := units[i].Fingerprint.WL
		var surv []survLabel
		for _, lc := range bag {
			if n := df[lc.Label]; n >= 2 && n <= x.cap {
				surv = append(surv, survLabel{label: lc.Label, count: lc.Count, h: lc.H, kind: lc.Kind})
				x.postings[lc.Label] = append(x.postings[lc.Label], posting{idx: i, count: lc.Count})
			}
		}
		x.surviving[i] = surv
		if len(surv) == 0 && len(bag) > 0 {
			x.suppressed++
		}
	}
	for h, n := range df {
		if n >= 2 && n <= x.cap {
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
		for _, sl := range x.surviving[i] { // ascending label: fixed float order
			e += x.idf[sl.label] * float64(sl.count)
		}
		x.energy[i] = e
	}
	return x
}

// labelBucket hashes the full label-bag identity (ordered label+count pairs —
// a WL bag is already canonically sorted), for the large-bucket diagnostic.
func labelBucket(bag []fingerprint.LabelCount) uint64 {
	h := fnv.New64a()
	var buf [12]byte
	for _, lc := range bag {
		binary.LittleEndian.PutUint64(buf[:8], lc.Label)
		binary.LittleEndian.PutUint32(buf[8:], uint32(lc.Count))
		_, _ = h.Write(buf[:])
	}
	return h.Sum64()
}

// admitPairs runs per-function retrieval: accumulate shared energy against
// every unit sharing a surviving label, then probe neighbors in
// (mass desc, idx asc) order with the exact fingerprint score, admitting up
// to ChannelK pairs at or above Threshold. Probing stops after
// 4*ChannelK scored neighbors either way — inside a dense sub-threshold
// cluster a function may fill fewer than K slots, which is accepted: those
// clusters are precisely the junk being suppressed, and an unbounded probe
// would degenerate back toward O(n²).
func (x *shapeIndex) admitPairs(sim *simCache, opt Options) []pairKey {
	return concatByIndex(len(x.surviving), func(a int) []pairKey { return x.admitFor(a, sim, opt) })
}

// admitFor is one function's turn of the admitPairs loop, factored out so a
// single probe unit can be retrieved without the all-pairs pass.
func (x *shapeIndex) admitFor(a int, sim *simCache, opt Options) []pairKey {
	if len(x.surviving[a]) == 0 {
		return nil
	}
	var pairs []pairKey
	acc := make(map[int]float64)
	for _, sl := range x.surviving[a] {
		w := x.idf[sl.label]
		for _, po := range x.postings[sl.label] {
			if po.idx != a {
				acc[po.idx] += w * float64(minCount(sl.count, po.count))
			}
		}
	}
	neighbors := make([]neighborMass, 0, len(acc))
	for b, mass := range acc {
		// A label in every eligible unit has idf 0; zero shared mass is
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
	return pairs
}

func minCount(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

// pairEvidence computes the definitive structural view of one pair in a
// single sorted-intersection pass: shared energy (Σ idf·min(count)), trophic
// similarity (weighted Dice, 2·shared/(E_A+E_B)), and the highest-energy
// shared labels as the explanation. Labels sort by (energy desc, depth desc,
// label asc) — byte-stable ties — and truncate to chainTopN.
//
// Depth descending is the WL analogue of the level ranking the pattern
// channel used: a label at h=3 folds in three edges of context, so it is a
// more specific claim about the pair than an h=0 node kind at the same
// energy, and the block should lead with the most specific structure. The
// label itself is the last tie-break because it is the only total order left
// once energy and depth agree — it decides nothing a reader sees except which
// of two equally-strong equally-deep labels prints first, and it must decide
// it the same way on every run. The three keys are a strict total order over
// the intersection (a label appears in it once), so there is nothing for a
// stable sort to stabilise.
//
// # The top N is selected, not sorted
//
// The pattern channel could sort its candidate chains outright: only levels
// 2 and up carried a render, so a pair offered a handful. Every shared WL
// label is a candidate, and a pair of substantial functions shares hundreds —
// sorting them all, per pair, over tens of thousands of pairs, is a real cost
// in a stage that exists to be cheap. insertChain keeps the running best
// chainTopN (3, or 20 under --debug) in order instead, which is one compare
// against the incumbent worst for the overwhelming majority of labels.
func (x *shapeIndex) pairEvidence(a, b, chainTopN int) (float64, float64, []SharedLabel) {
	sa, sb := x.surviving[a], x.surviving[b]
	var mass float64
	var chains []SharedLabel
	i, j := 0, 0
	for i < len(sa) && j < len(sb) {
		switch {
		case sa[i].label < sb[j].label:
			i++
		case sa[i].label > sb[j].label:
			j++
		default:
			n := minCount(sa[i].count, sb[j].count)
			e := x.idf[sa[i].label] * float64(n)
			mass += e
			if e > 0 && chainTopN != 0 {
				chains = insertChain(chains, SharedLabel{
					Label:  sa[i].label,
					Depth:  int(sa[i].h),
					Count:  int(n),
					Energy: e,
					Render: fingerprint.DescribeLabel(sa[i].h, sa[i].kind),
				}, chainTopN)
			}
			i++
			j++
		}
	}
	var trophic float64
	if denom := x.energy[a] + x.energy[b]; denom > 0 {
		trophic = 2 * mass / denom
	}
	if chainTopN < 0 {
		// Unbounded: insertChain kept everything, in no particular order.
		sort.Slice(chains, func(p, q int) bool { return betterChain(chains[p], chains[q]) })
	}
	return mass, trophic, chains
}

// betterChain is the shared-structure order: energy desc, depth desc, label
// asc. Strict and total over one pair's shared labels.
func betterChain(p, q SharedLabel) bool {
	if p.Energy != q.Energy {
		return p.Energy > q.Energy
	}
	if p.Depth != q.Depth {
		return p.Depth > q.Depth
	}
	return p.Label < q.Label
}

// insertChain keeps the best n chains in betterChain order. n < 0 means
// unbounded, in which case it appends and the caller sorts once.
func insertChain(chains []SharedLabel, c SharedLabel, n int) []SharedLabel {
	if n < 0 {
		return append(chains, c)
	}
	if len(chains) == n && !betterChain(c, chains[len(chains)-1]) {
		return chains
	}
	if len(chains) < n {
		chains = append(chains, c)
	} else {
		chains[len(chains)-1] = c
	}
	for i := len(chains) - 1; i > 0 && betterChain(chains[i], chains[i-1]); i-- {
		chains[i], chains[i-1] = chains[i-1], chains[i]
	}
	return chains
}

// Package calibrate sets score thresholds from the corpus instead of from
// constants: it scores a deterministic sample of random, unrelated function
// pairs — the null distribution — and returns the score that a chosen
// fraction of them exceed. A threshold of 0.60 means different things on a
// corpus of 81 functions and one of 8000; "admit 1% of random pairs" means
// the same thing on both.
//
// Everything here is deterministic by construction. The sample is a bottom-k
// sample over pair priorities hashed from each function's identity, the pairs
// are scored in ascending index order, and the quantile is a rank, not an
// interpolation. An unchanged tree calibrates to the same numbers every run,
// which is what lets the derived thresholds live in a snapshot's Params.
//
// The sample is also stable under change, which a seeded generator was not.
// The old sampler seeded an LCG from a hash over every name in the corpus and
// drew positions in the canonical order, so adding any function - a one-line
// accessor in an unrelated package - redrew all 20 000 null pairs, and the
// derived floors moved by a hundredth as often as not, dropping every pair
// that sat between the old cut and the new one. Measured on doppel's own
// tree, a one-line method moved threshold 0.35 -> 0.36 and struct-min
// 0.40 -> 0.41 and took 135 of 1 842 reported pairs with it. Under bottom-k a
// pair's priority depends on its two functions alone, so an added function
// can only displace the existing sample where one of its own pairs outbids it.
//
// The package imports parser, fingerprint, comparator and concepter only; it
// never imports cmd, analyzer or bench.
package calibrate

import (
	"hash/fnv"
	"math"
	"sort"
	"strconv"

	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parallel"
	"github.com/LukasSelin/doppel/internal/parser"
)

// Options are the calibration knobs.
type Options struct {
	Rate         float64 // fraction of null pairs a threshold admits; 0 = off
	MaxPairs     int     // null pairs sampled per distribution (20000 in production)
	MinNodes     int     // code-shape null mirrors the shape channel's eligibility gate
	MinNullPairs int     // below this many null pairs the calibration is declined
	Weights      fingerprint.Weights
}

// DefaultOptions returns the production sampling sizes for a given rate.
func DefaultOptions(rate float64, minNodes int) Options {
	return Options{Rate: rate, MaxPairs: 20000, MinNodes: minNodes, MinNullPairs: 1000}
}

// Result is one calibration. Threshold and StructMin are the values to use,
// rounded up to 0.01; Declined names the reason they were not derived, in
// which case both are zero and the caller keeps its defaults.
type Result struct {
	Rate         float64
	ShapePairs   int // null pairs scored for code-shape
	OverlapPairs int // null pairs scored for structural overlap
	Threshold    float64
	StructMin    float64
	Declined     string
}

// Applied reports whether the calibration produced thresholds.
func (r Result) Applied() bool { return r.Declined == "" }

// Run calibrates over a corpus. comp is the run's own comparator and wl the
// run's own label weighting, so both nulls are corpus-weighted exactly like
// the pairs they will gate. Passing a different wl than the run scores with
// would calibrate a threshold against a distribution nothing is measured on.
func Run(units []parser.CodeUnit, docs []concepter.ConceptDoc, comp *comparator.Comparator,
	wl *fingerprint.LabelIDF, o Options) Result {
	res := Result{Rate: o.Rate}
	if o.Rate <= 0 || o.Rate >= 1 {
		res.Declined = "rate must be in (0, 1)"
		return res
	}
	w := o.Weights
	if w == (fingerprint.Weights{}) {
		w = fingerprint.DefaultWeights()
	}
	order := canonicalOrder(units)
	ids := identities(units, order)

	// Code-shape null: over units the shape channel would consider, so the
	// threshold is calibrated at the gate's own operating point.
	var shapeIdx []int
	for _, i := range order {
		if fp := units[i].Fingerprint; fp.Nodes >= o.MinNodes && fp.Nodes > 0 {
			shapeIdx = append(shapeIdx, i)
		}
	}
	shapePairs := samplePopulation(units, shapeIdx, o.MaxPairs, ids, shapeSalt)
	res.ShapePairs = len(shapePairs)
	if len(shapePairs) < o.MinNullPairs {
		res.Declined = declined(len(shapePairs), o.MinNullPairs, "shape")
		return res
	}
	// Scored across cores, by index. Every null pair is independent and
	// SimilarityWith is pure — two fingerprints, the corpus label weights and
	// the blend, all read-only here — so nothing is shared but the result
	// slice, and each slot is written once. The sort below makes even the fill
	// order immaterial; writing by index keeps it exact anyway.
	shape := make([]float64, len(shapePairs))
	parallel.Blocks(len(shapePairs), nullBlock, minPairsPerNullWorker, func(i int) {
		p := shapePairs[i]
		shape[i] = fingerprint.SimilarityWith(units[p[0]].Fingerprint, units[p[1]].Fingerprint, wl, w).Score
	})

	// Overlap null: every unit, the way the comparator sees pairs.
	overlapPairs := samplePopulation(units, order, o.MaxPairs, ids, overlapSalt)
	res.OverlapPairs = len(overlapPairs)
	if len(overlapPairs) < o.MinNullPairs || comp == nil || len(docs) != len(units) {
		res.Declined = declined(len(overlapPairs), o.MinNullPairs, "overlap")
		return res
	}
	// The same fan-out, with one forked comparator per worker — the comparator
	// carries the vocabulary's profile scratch, which is the only mutable state
	// in a Compare and the reason Fork exists. This is the null distribution of
	// exactly the comparator the run will use, so it must be that comparator
	// and not a fresh one.
	overlap := make([]float64, len(overlapPairs))
	parallel.BlocksWith(len(overlapPairs), nullBlock, minPairsPerNullWorker,
		comp.Fork,
		func(c *comparator.Comparator, i int) {
			p := overlapPairs[i]
			overlap[i] = c.Compare(docs[p[0]], docs[p[1]]).OverlapScore
		})

	sort.Float64s(shape)
	sort.Float64s(overlap)
	res.Threshold = roundUp(Quantile(shape, 1-o.Rate))
	res.StructMin = roundUp(Quantile(overlap, 1-o.Rate))
	return res
}

func declined(have, need int, which string) string {
	return "only " + itoa(have) + " eligible " + which + " null pairs (need " + itoa(need) + ")"
}

// canonicalOrder sorts unit indices by (package.name, file, line) so the
// sample never depends on walk order.
func canonicalOrder(units []parser.CodeUnit) []int {
	order := make([]int, len(units))
	for i := range order {
		order[i] = i
	}
	key := func(i int) string {
		u := units[i]
		return u.Package + "." + u.Name + "\x00" + u.File + "\x00" + itoa(u.StartLine)
	}
	sort.SliceStable(order, func(a, b int) bool { return key(order[a]) < key(order[b]) })
	return order
}

// Seed derives a seed from the corpus: FNV-1a over the canonical unit names.
// Run no longer uses it - a seed over the whole population is exactly what
// made every added function redraw the sample - and it survives for callers
// of SamplePairs that bring their own units and want one fixed draw per tree.
func Seed(units []parser.CodeUnit) uint64 {
	order := canonicalOrder(units)
	h := fnv.New64a()
	for _, i := range order {
		u := units[i]
		_, _ = h.Write([]byte(u.Package + "." + u.Name))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}

// The two nulls are independent samples over overlapping populations, so
// their priorities are salted apart.
const (
	shapeSalt   uint64 = 0
	overlapSalt uint64 = 0x9e3779b97f4a7c15
)

// identities hashes each unit's identity: package, name and file, plus an
// ordinal for the second and later declarations of one name in one file
// (init, or functions in a bundled script) - the snapshot key's rule. Never
// the line: a line number moves whenever code above it does, and a priority
// keyed on it would reshuffle the sample on edits nobody made to the unit.
// order is canonicalOrder, which is what makes the ordinal deterministic.
func identities(units []parser.CodeUnit, order []int) []uint64 {
	ids := make([]uint64, len(units))
	seen := make(map[string]int, len(units))
	for _, i := range order {
		u := units[i]
		key := u.Package + "." + u.Name + "\x00" + u.File
		n := seen[key]
		seen[key] = n + 1
		if n > 0 {
			key += "\x00" + itoa(n)
		}
		h := fnv.New64a()
		_, _ = h.Write([]byte(key))
		ids[i] = h.Sum64()
	}
	return ids
}

// mix is splitmix64's finalizer: a bijection on uint64 whose output bits each
// depend on every input bit, which is what lets a priority built from two
// FNV hashes read as uniform.
func mix(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}

// priority is a pair's place in the bottom-k sample: symmetric in the two
// identities, and a function of them and the salt alone.
func priority(a, b, salt uint64) uint64 {
	if a > b {
		a, b = b, a
	}
	return mix(a ^ mix(b^salt))
}

// samplePopulation is a bottom-k sample: of every unordered pair over the
// given unit indices that stays inside one build unit (cross test/production
// and cross-target pairs are never merge candidates, so never part of the
// null either), the k with the lowest priority. When the population has at
// most k such pairs, that is all of them. The result is sorted ascending so
// every consumer scores in one fixed order.
//
// Finding the k lowest without holding every pair: priorities are uniform, so
// a cut at about k/total of the range keeps about k pairs. Rows are scanned
// across cores against the cut, and a cut that kept too few - the build-unit
// rule rejects some - is doubled and the scan repeated. Cost is one mix per
// pair, O(m^2): 29M on moby's 7 658 functions.
func samplePopulation(units []parser.CodeUnit, idx []int, k int, ids []uint64, salt uint64) [][2]int {
	m := len(idx)
	if m < 2 || k <= 0 {
		return nil
	}
	type cand struct {
		prio uint64
		pair [2]int
	}
	total := float64(m) * float64(m-1) / 2
	cut := uint64(math.MaxUint64)
	if f := 1.25 * float64(k) / total; f < 0.5 { // past half, scanning everything is as cheap
		cut = uint64(f * math.MaxUint64)
	}
	var cands []cand
	for {
		rows := make([][]cand, m)
		parallel.Blocks(m, sampleRowBlock, minRowsPerSampleWorker, func(a int) {
			i := idx[a]
			var row []cand
			for b := a + 1; b < m; b++ {
				j := idx[b]
				p := priority(ids[i], ids[j], salt)
				if p > cut || !parser.SameBuildUnit(units[i], units[j]) {
					continue
				}
				row = append(row, cand{p, orderPair(i, j)})
			}
			rows[a] = row
		})
		cands = cands[:0]
		for _, row := range rows {
			cands = append(cands, row...)
		}
		if len(cands) >= k || cut == math.MaxUint64 {
			break
		}
		if cut > math.MaxUint64/2 {
			cut = math.MaxUint64
		} else {
			cut *= 2
		}
	}
	sort.Slice(cands, func(a, b int) bool {
		x, y := cands[a], cands[b]
		if x.prio != y.prio {
			return x.prio < y.prio
		}
		if x.pair[0] != y.pair[0] {
			return x.pair[0] < y.pair[0]
		}
		return x.pair[1] < y.pair[1]
	})
	if len(cands) > k {
		cands = cands[:k]
	}
	pairs := make([][2]int, len(cands))
	for i, c := range cands {
		pairs[i] = c.pair
	}
	sort.Slice(pairs, func(a, b int) bool {
		if pairs[a][0] != pairs[b][0] {
			return pairs[a][0] < pairs[b][0]
		}
		return pairs[a][1] < pairs[b][1]
	})
	return pairs
}

// SamplePairs draws up to k distinct unordered index pairs from [0, n) - the
// sampler without any population rule, for callers that bring their own
// units (the bench self-weighting experiment). Identities are the indices
// salted by seed, so one seed is one fixed draw.
func SamplePairs(n, k int, seed uint64) [][2]int {
	units := make([]parser.CodeUnit, n)
	idx := make([]int, n)
	ids := make([]uint64, n)
	for i := range idx {
		idx[i] = i
		ids[i] = mix(seed ^ mix(uint64(i)))
	}
	return samplePopulation(units, idx, k, ids, 0)
}

func orderPair(i, j int) [2]int {
	if i > j {
		i, j = j, i
	}
	return [2]int{i, j}
}

// Quantile is the nearest-rank upper quantile of an ascending slice: the
// value at rank ceil(q·n), clamped to [1, n]. A rank, not an interpolation,
// so the answer is always a score some null pair actually had. Ties resolve
// upward by construction — under a tie spike the admitted fraction can
// exceed the rate, which is why Run rounds the result up afterwards.
func Quantile(sorted []float64, q float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	r := int(math.Ceil(q * float64(n)))
	if r < 1 {
		r = 1
	}
	if r > n {
		r = n
	}
	return sorted[r-1]
}

// roundUp rounds a threshold up to the next 0.01, so the printed value is
// the used value and the admitted null fraction is at most the rate.
func roundUp(v float64) float64 {
	return math.Ceil(v*100-1e-9) / 100
}

func itoa(n int) string { return strconv.Itoa(n) }

// One null pair costs about what one candidate comparison does, so these mirror
// the comparison stage's two knobs. MinNullPairs (1000) is above the sequential
// floor, so a calibration that runs at all runs across cores.
const (
	nullBlock             = 64
	minPairsPerNullWorker = 512

	// A sampler row is a scan of up to m pairs, so a block of rows is real
	// work; rows shorten toward the end, which the block counter absorbs.
	sampleRowBlock         = 16
	minRowsPerSampleWorker = 64
)

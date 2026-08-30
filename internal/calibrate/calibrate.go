// Package calibrate sets score thresholds from the corpus instead of from
// constants: it scores a deterministic sample of random, unrelated function
// pairs — the null distribution — and returns the score that a chosen
// fraction of them exceed. A threshold of 0.60 means different things on a
// corpus of 81 functions and one of 8000; "admit 1% of random pairs" means
// the same thing on both.
//
// Everything here is deterministic by construction. The sample is drawn by a
// seeded generator whose seed is derived from the corpus's own names in a
// canonical order, the pairs are scored in ascending index order, and the
// quantile is a rank, not an interpolation. An unchanged tree calibrates to
// the same numbers every run, which is what lets the derived thresholds live
// in a snapshot's Params.
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

// Run calibrates over a corpus. comp is the run's own comparator, so the
// overlap null is corpus-weighted exactly like the pairs it will gate.
func Run(units []parser.CodeUnit, docs []concepter.ConceptDoc, comp *comparator.Comparator, o Options) Result {
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
	seed := Seed(units)

	// Code-shape null: over units the shape channel would consider, so the
	// threshold is calibrated at the gate's own operating point.
	var shapeIdx []int
	for _, i := range order {
		if fp := units[i].Fingerprint; fp.Nodes >= o.MinNodes && fp.Nodes > 0 {
			shapeIdx = append(shapeIdx, i)
		}
	}
	shapePairs := samplePopulation(units, shapeIdx, o.MaxPairs, seed)
	res.ShapePairs = len(shapePairs)
	if len(shapePairs) < o.MinNullPairs {
		res.Declined = declined(len(shapePairs), o.MinNullPairs, "shape")
		return res
	}
	shape := make([]float64, 0, len(shapePairs))
	for _, p := range shapePairs {
		shape = append(shape, fingerprint.SimilarityWith(units[p[0]].Fingerprint, units[p[1]].Fingerprint, w).Score)
	}

	// Overlap null: every unit, the way the comparator sees pairs.
	overlapPairs := samplePopulation(units, order, o.MaxPairs, seed^0x9e3779b97f4a7c15)
	res.OverlapPairs = len(overlapPairs)
	if len(overlapPairs) < o.MinNullPairs || comp == nil || len(docs) != len(units) {
		res.Declined = declined(len(overlapPairs), o.MinNullPairs, "overlap")
		return res
	}
	overlap := make([]float64, 0, len(overlapPairs))
	for _, p := range overlapPairs {
		overlap = append(overlap, comp.Compare(docs[p[0]], docs[p[1]]).OverlapScore)
	}

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

// Seed derives the sampler seed from the corpus: FNV-1a over the canonical
// unit names. Corpus-derived and order-independent, so the same tree always
// draws the same null sample.
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

// samplePopulation draws up to k distinct unordered pairs from the given
// unit indices, rejecting pairs that cross the test/production boundary
// (never merge candidates, so never part of the null either). When the
// population has at most k pairs it is enumerated instead. The result is
// sorted ascending so every consumer scores in one fixed order.
func samplePopulation(units []parser.CodeUnit, idx []int, k int, seed uint64) [][2]int {
	m := len(idx)
	if m < 2 {
		return nil
	}
	sameUnit := func(i, j int) bool { return parser.SameBuildUnit(units[i], units[j]) }
	var pairs [][2]int
	if m*(m-1)/2 <= k {
		for a := 0; a < m; a++ {
			for b := a + 1; b < m; b++ {
				i, j := idx[a], idx[b]
				if !sameUnit(i, j) {
					continue
				}
				pairs = append(pairs, orderPair(i, j))
			}
		}
	} else {
		seen := make(map[[2]int]bool, k)
		x := seed
		next := func() int {
			x = x*6364136223846793005 + 1442695040888963407
			return int((x >> 33) % uint64(m))
		}
		for draws := 0; len(pairs) < k && draws < 8*k; draws++ {
			a, b := next(), next()
			if a == b {
				continue
			}
			i, j := idx[a], idx[b]
			if !sameUnit(i, j) {
				continue
			}
			p := orderPair(i, j)
			if seen[p] {
				continue
			}
			seen[p] = true
			pairs = append(pairs, p)
		}
	}
	sort.Slice(pairs, func(a, b int) bool {
		if pairs[a][0] != pairs[b][0] {
			return pairs[a][0] < pairs[b][0]
		}
		return pairs[a][1] < pairs[b][1]
	})
	return pairs
}

// SamplePairs draws up to k distinct unordered index pairs from [0, n) with
// the package's generator — the sampler without any population rule, for
// callers that bring their own units (the bench self-weighting experiment).
func SamplePairs(n, k int, seed uint64) [][2]int {
	units := make([]parser.CodeUnit, n)
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	return samplePopulation(units, idx, k, seed)
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

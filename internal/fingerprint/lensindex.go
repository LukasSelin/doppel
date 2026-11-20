package fingerprint

import (
	"slices"

	"github.com/LukasSelin/doppel/internal/detmath"
	"github.com/LukasSelin/doppel/internal/parallel"
)

// LensIndex is one lens's reading of a whole corpus, reduced to what a pair
// comparison needs: for every unit, the labels some other unit also carries,
// and the weighted mass of the unit's whole bag.
//
// It exists so a lens other than shape can be scored on every pair without
// holding that lens's bags and weight table for the rest of a run. A lens
// that keeps names produces mostly labels no second unit carries — a deep
// label is a hash of a subtree with its identifiers in it — and such a label
// can never be shared, so all a comparison needs from it is its weight in its
// own unit's total. Summing those once per unit and dropping the labels is
// what this type does. The comparison is then a merge over two short sorted
// rows with no map lookup in it.
//
// The result is not an approximation. Jaccard(a, b) is bit for bit
// WLOverlap(bags[a], bags[b], LabelWeights(bags)): a unit's mass is summed in
// ascending label order, which is the order wlOverlap adds that side's terms
// in however the two bags interleave, and the shared mass runs over the same
// labels in the same order. TestLensIndexMatchesOverlap pins it.
//
// Read-only once built, so one index may be read from any number of
// goroutines.
//
// # Why rows hold ids and not labels
//
// A row entry is a dense label id and a count, eight bytes, and the weights
// live once in a table the ids index. Carrying the 64-bit label and its
// weight on every entry was three times that, and the rows are what a run
// retains: measured on moby with DOPPEL_MEMSTATS, 27MB of live heap in that
// form. Ids are handed out in ascending label order, so a row sorted by id
// is sorted by label and the merge adds its terms in the order it always did.
type LensIndex struct {
	rows    [][]lensLabel
	mass    []float64
	weights []float64 // by label id: ln(N/df) of every label two or more units carry
}

// lensLabel is one shared label of a unit's row: which label, and how many
// times the unit carries it.
type lensLabel struct {
	id    uint32
	count int32
}

// NewLensIndex indexes a population's bags, one per unit and in unit order;
// a nil bag is a unit without a body. The weights are the lens's own, counted
// over exactly these bags, which is the rule Lens states: a lens's labels are
// a different vocabulary from the shape lens's, so nothing here may read the
// run's shape weights.
//
// It counts document frequency itself instead of calling LabelWeights, for
// the allocation: that table keeps a weight for every label, and here most
// labels are carried by one unit and never looked up again — their weight is
// ln(N), the same for all of them. The arithmetic is LabelWeights' own (N is
// the units that have a body, the weight is detmath.Log(N/df)), and the test
// that pins this type against WLOverlap would fail on any drift.
func NewLensIndex(bags [][]LabelCount) *LensIndex {
	n := 0
	df := make(map[uint64]int32)
	for _, bag := range bags {
		if len(bag) == 0 {
			continue
		}
		n++
		for _, lc := range bag {
			df[lc.Label]++
		}
	}

	// Ids in ascending label order, for the labels a comparison can share.
	shared := make([]uint64, 0, len(df))
	for label, d := range df {
		if d >= 2 {
			shared = append(shared, label)
		}
	}
	slices.Sort(shared)
	x := &LensIndex{
		rows:    make([][]lensLabel, len(bags)),
		mass:    make([]float64, len(bags)),
		weights: make([]float64, len(shared)),
	}
	ids := make(map[uint64]uint32, len(shared))
	for id, label := range shared {
		ids[label] = uint32(id)
		x.weights[id] = detmath.Log(float64(n) / float64(df[label]))
	}
	// A label one unit carries is the rarest thing the corpus can express.
	alone := 0.0
	if n > 0 {
		alone = detmath.Log(float64(n))
	}

	// The per-unit pass reads the two tables above and writes its own row and
	// mass, so it fans out; each worker keeps one scratch row to size the
	// stored one exactly without looking a label up twice.
	parallel.BlocksWith(len(bags), lensBlock, minUnitsPerLensWorker,
		func() *[]lensLabel { return new([]lensLabel) },
		func(scratch *[]lensLabel, i int) {
			row := (*scratch)[:0]
			var mass float64
			for _, lc := range bags[i] {
				w := alone
				if id, ok := ids[lc.Label]; ok {
					w = x.weights[id]
					row = append(row, lensLabel{id: id, count: lc.Count})
				}
				mass += float64(w * float64(lc.Count))
			}
			*scratch = row
			x.mass[i] = mass
			// A unit that shares nothing keeps a nil row: cloning an empty
			// slice would hand back one that still points into the scratch.
			if len(row) > 0 {
				x.rows[i] = slices.Clone(row)
			}
		})
	return x
}

// The per-unit pass's fan-out: a block small enough that one very large body
// strands nobody, and a floor that keeps a small corpus sequential.
const (
	lensBlock             = 32
	minUnitsPerLensWorker = 256
)

// Len is how many units the index covers.
func (x *LensIndex) Len() int {
	if x == nil {
		return 0
	}
	return len(x.rows)
}

// Jaccard is the corpus-weighted multiset Jaccard of units a and b under this
// lens: Σ w·min(a,b) / Σ w·max(a,b), the metric the code-shape score's wl
// component uses, over this lens's labels and weights. 0 for an index out of
// range, a unit without a body, or a pair whose every label is in every unit.
func (x *LensIndex) Jaccard(a, b int) float64 {
	if x == nil || a < 0 || b < 0 || a >= len(x.rows) || b >= len(x.rows) {
		return 0
	}
	if a == b {
		// The one comparison where a dropped label is shared: a unit has
		// every label it carries, the ones nobody else does included.
		if x.mass[a] > 0 {
			return 1
		}
		return 0
	}
	ra, rb := x.rows[a], x.rows[b]
	var shared float64
	i, j := 0, 0
	for i < len(ra) && j < len(rb) {
		switch {
		case ra[i].id < rb[j].id:
			i++
		case rb[j].id < ra[i].id:
			j++
		default:
			shared += float64(x.weights[ra[i].id] * float64(min(ra[i].count, rb[j].count)))
			i++
			j++
		}
	}
	if union := x.mass[a] + x.mass[b] - shared; union > 0 {
		return shared / union
	}
	return 0
}

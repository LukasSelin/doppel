package fingerprint

import (
	"math/rand"
	"reflect"
	"runtime"
	"slices"
	"testing"

	"github.com/LukasSelin/doppel/internal/parallel"
)

// vocabBags is every lens fixture read under the vocab lens: a small
// population whose bodies share some named structure and not the rest.
func vocabBags(t *testing.T) [][]LabelCount {
	t.Helper()
	vocab := Lenses()[LensVocab]
	var bags [][]LabelCount
	for _, fx := range lensFixtures {
		bags = append(bags, vocab.Bag(parseFn(t, fx.a)), vocab.Bag(parseFn(t, fx.b)))
	}
	return append(bags, nil) // a unit without a body
}

// randomBags is a seeded population over a small label alphabet, so labels
// land at every document frequency from one unit to all of them.
func randomBags(units, alphabet int) [][]LabelCount {
	rng := rand.New(rand.NewSource(1))
	bags := make([][]LabelCount, units)
	for i := range bags {
		counts := map[uint64]int32{}
		for k := rng.Intn(40); k > 0; k-- {
			counts[uint64(rng.Intn(alphabet))]++
		}
		counts[0]++ // one label every unit carries: weight 0
		for label, c := range counts {
			bags[i] = append(bags[i], LabelCount{Label: label, Count: c})
		}
		slices.SortFunc(bags[i], func(a, b LabelCount) int {
			switch {
			case a.Label < b.Label:
				return -1
			case a.Label > b.Label:
				return 1
			}
			return 0
		})
	}
	return bags
}

// The index drops every label no second unit carries, and must still answer
// exactly what the full bags and the full weight table answer.
func TestLensIndexMatchesOverlap(t *testing.T) {
	for name, bags := range map[string][][]LabelCount{
		"fixtures": vocabBags(t),
		"random":   randomBags(60, 300),
	} {
		idf := LabelWeights(bags)
		x := NewLensIndex(bags)
		if x.Len() != len(bags) {
			t.Fatalf("%s: Len = %d, want %d", name, x.Len(), len(bags))
		}
		nonzero := 0
		for a := range bags {
			for b := range bags {
				want, _ := WLOverlap(bags[a], bags[b], idf)
				if got := x.Jaccard(a, b); got != want {
					t.Errorf("%s: Jaccard(%d, %d) = %v, want %v", name, a, b, got, want)
				}
				if a != b && want > 0 {
					nonzero++
				}
			}
		}
		if nonzero == 0 {
			t.Errorf("%s: no two units share anything, so the comparison proves nothing", name)
		}
	}
}

func TestLensIndexKeepsOnlySharedLabels(t *testing.T) {
	bags := randomBags(60, 300)
	idf := LabelWeights(bags)
	x := NewLensIndex(bags)
	kept, total := 0, 0
	for i, row := range x.rows {
		total += len(bags[i])
		kept += len(row)
		// Every kept label is one two or more units carry, and a row keeps
		// its bag's order: ids ascend exactly as the labels behind them do.
		at := 0
		for _, lc := range bags[i] {
			if idf.DF(lc.Label) < 2 {
				continue
			}
			if at >= len(row) || row[at].count != lc.Count || x.weights[row[at].id] != idf.Weight(lc.Label) {
				t.Fatalf("unit %d: row entry %d does not carry label %d", i, at, lc.Label)
			}
			if at > 0 && row[at-1].id >= row[at].id {
				t.Fatalf("unit %d: row ids are not ascending at %d", i, at)
			}
			at++
		}
		if at != len(row) {
			t.Fatalf("unit %d keeps %d labels, want the %d that two or more units carry", i, len(row), at)
		}
	}
	if kept == 0 || kept >= total {
		t.Errorf("kept %d of %d labels: the fixture has to hold both shared and unshared ones", kept, total)
	}
}

func TestLensIndexEdges(t *testing.T) {
	bags := vocabBags(t)
	x := NewLensIndex(bags)
	last := len(bags) - 1 // the unit without a body
	for _, c := range [][2]int{{-1, 0}, {0, -1}, {0, len(bags)}, {last, 0}, {0, last}, {last, last}} {
		if got := x.Jaccard(c[0], c[1]); got != 0 {
			t.Errorf("Jaccard(%d, %d) = %v, want 0", c[0], c[1], got)
		}
	}
	// A body compared with itself is the whole of itself.
	if got := x.Jaccard(0, 0); got != 1 {
		t.Errorf("Jaccard(0, 0) = %v, want 1", got)
	}
	var none *LensIndex
	if none.Len() != 0 || none.Jaccard(0, 1) != 0 {
		t.Error("a nil index must answer 0")
	}
}

// The per-unit pass fans out above a size floor no other fixture here
// reaches, so build one that does and require the same index a single
// goroutine builds.
func TestLensIndexParallelMatchesSequential(t *testing.T) {
	bags := randomBags(4*minUnitsPerLensWorker, 2000)
	if runtime.GOMAXPROCS(0) > 1 && parallel.Workers(len(bags), minUnitsPerLensWorker) < 2 {
		t.Fatalf("fixture of %d units does not reach the parallel path", len(bags))
	}
	got := NewLensIndex(bags)

	prev := runtime.GOMAXPROCS(1)
	want := NewLensIndex(bags)
	runtime.GOMAXPROCS(prev)

	if !reflect.DeepEqual(got, want) {
		t.Fatal("an index built across cores differs from one built on a single goroutine")
	}
	idf := LabelWeights(bags)
	for a := 0; a < len(bags); a += 37 {
		for b := 1; b < len(bags); b += 53 {
			if j, _ := WLOverlap(bags[a], bags[b], idf); got.Jaccard(a, b) != j {
				t.Fatalf("Jaccard(%d, %d) = %v, want %v", a, b, got.Jaccard(a, b), j)
			}
		}
	}
}

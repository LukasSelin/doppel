package calibrate

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// labelWeights counts the WL surprisals over a corpus the way cmd's index()
// does. Calibration must read the same weights the run it calibrates will
// score with, or the null distribution belongs to a different question.
func labelWeights(units []parser.CodeUnit) *fingerprint.LabelIDF {
	bags := make([][]fingerprint.LabelCount, len(units))
	for i := range units {
		bags[i] = units[i].Fingerprint.WL
	}
	return fingerprint.LabelWeights(bags)
}

// fixture builds n synthetic units with varied fingerprints: half big (shape
// eligible), half tiny, every fifth one a test file. No real corpus names.
func fixture(n int) ([]parser.CodeUnit, []concepter.ConceptDoc) {
	units := make([]parser.CodeUnit, n)
	docs := make([]concepter.ConceptDoc, n)
	for i := range units {
		u := parser.CodeUnit{Name: fmt.Sprintf("F%03d", i), Package: "p", File: "p/a.go", StartLine: i + 1}
		if i%5 == 0 {
			u.File = "p/a_test.go"
		}
		nodes := 40
		if i%2 == 1 {
			nodes = 4 // below min-nodes
		}
		u.Fingerprint.Nodes = nodes
		for s := 0; s < 6; s++ {
			u.Fingerprint.Shingles = append(u.Fingerprint.Shingles, uint64(i%7*10+s))
		}
		u.Fingerprint.Flow = []int{i % 3, 1}
		u.Fingerprint.Depth = []int{1, i % 2}
		u.Fingerprint.Types = []string{fmt.Sprintf("in:t%d", i%4)}
		units[i] = u
		docs[i] = concepter.ConceptDoc{Role: []string{"leaf", "utility"}[i%2], Package: "p", Callees: []string{fmt.Sprintf("c%d", i%5)}}
	}
	return units, docs
}

func comp() *comparator.Comparator {
	return comparator.New(ontology.NewScorer(ontology.Default(), nil))
}

func TestQuantile(t *testing.T) {
	s := make([]float64, 100)
	for i := range s {
		s[i] = float64(i + 1)
	}
	if q := Quantile(s, 0.99); q != 99 {
		t.Errorf("Quantile(1..100, 0.99) = %v, want 99", q)
	}
	if q := Quantile(s, 0.5); q != 50 {
		t.Errorf("Quantile(1..100, 0.5) = %v, want 50", q)
	}
	if q := Quantile(s, 1); q != 100 {
		t.Errorf("Quantile(1..100, 1) = %v, want 100", q)
	}
	// A tie spike at the cut resolves upward: the value at the rank is the
	// tied value, and everything equal to it is admitted.
	spike := []float64{0.1, 0.2, 0.3, 0.9, 0.9, 0.9, 0.9, 0.9, 0.9, 0.9}
	if q := Quantile(spike, 0.99); q != 0.9 {
		t.Errorf("tie spike quantile = %v, want 0.9", q)
	}
	if Quantile(nil, 0.5) != 0 {
		t.Error("empty quantile must be 0")
	}
}

func TestRoundUp(t *testing.T) {
	for _, tc := range []struct{ in, want float64 }{{0.63, 0.63}, {0.631, 0.64}, {0.6299999, 0.63}, {0, 0}, {1, 1}} {
		if got := roundUp(tc.in); got != tc.want {
			t.Errorf("roundUp(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestSamplePairsProperties(t *testing.T) {
	pairs := SamplePairs(200, 500, 7)
	if len(pairs) != 500 {
		t.Fatalf("got %d pairs, want 500", len(pairs))
	}
	seen := map[[2]int]bool{}
	for i, p := range pairs {
		if p[0] >= p[1] {
			t.Fatalf("pair %v is not i<j", p)
		}
		if seen[p] {
			t.Fatalf("duplicate pair %v", p)
		}
		seen[p] = true
		if i > 0 && (pairs[i-1][0] > p[0] || (pairs[i-1][0] == p[0] && pairs[i-1][1] >= p[1])) {
			t.Fatalf("pairs not ascending at %d", i)
		}
	}
	if !reflect.DeepEqual(pairs, SamplePairs(200, 500, 7)) {
		t.Error("same seed, different sample")
	}
	if reflect.DeepEqual(pairs, SamplePairs(200, 500, 8)) {
		t.Error("different seed, same sample")
	}
	// Small populations enumerate rather than sample.
	if all := SamplePairs(5, 100, 1); len(all) != 10 {
		t.Errorf("5 units enumerate to %d pairs, want 10", len(all))
	}
}

func TestRunMirrorsEligibilityAndPopulation(t *testing.T) {
	units, docs := fixture(60)
	o := Options{Rate: 0.05, MaxPairs: 5000, MinNodes: 12, MinNullPairs: 10}
	// Reach into the sampler with the same rules Run uses.
	order := canonicalOrder(units)
	var shapeIdx []int
	for _, i := range order {
		if units[i].Fingerprint.Nodes >= o.MinNodes {
			shapeIdx = append(shapeIdx, i)
		}
	}
	for _, p := range samplePopulation(units, shapeIdx, o.MaxPairs, Seed(units)) {
		for _, i := range p {
			if units[i].Fingerprint.Nodes < o.MinNodes {
				t.Fatalf("shape null sampled a unit below min-nodes: %d", i)
			}
		}
		if (units[p[0]].File == "p/a_test.go") != (units[p[1]].File == "p/a_test.go") {
			t.Fatalf("cross test/prod pair sampled: %v", p)
		}
	}
	res := Run(units, docs, comp(), labelWeights(units), o)
	if !res.Applied() {
		t.Fatalf("declined: %s", res.Declined)
	}
	if res.Threshold <= 0 || res.Threshold > 1 || res.StructMin <= 0 || res.StructMin > 1 {
		t.Errorf("thresholds out of range: %+v", res)
	}
	// Overlap null covers all units, so it has more pairs than the shape null.
	if res.OverlapPairs <= res.ShapePairs {
		t.Errorf("overlap pairs %d should exceed shape pairs %d", res.OverlapPairs, res.ShapePairs)
	}
}

func TestRunDeterministic(t *testing.T) {
	units, docs := fixture(120)
	o := Options{Rate: 0.01, MaxPairs: 3000, MinNodes: 12, MinNullPairs: 100}
	first := Run(units, docs, comp(), labelWeights(units), o)
	for i := 0; i < 25; i++ {
		if again := Run(units, docs, comp(), labelWeights(units), o); again != first {
			t.Fatalf("run %d differs: %+v vs %+v", i, again, first)
		}
	}
	// Walk order must not matter: reverse the units and the docs together.
	rev := make([]parser.CodeUnit, len(units))
	revDocs := make([]concepter.ConceptDoc, len(docs))
	for i := range units {
		rev[len(units)-1-i], revDocs[len(units)-1-i] = units[i], docs[i]
	}
	if got := Run(rev, revDocs, comp(), labelWeights(rev), o); got != first {
		t.Errorf("reversed unit order changed the calibration: %+v vs %+v", got, first)
	}
}

func TestRunDeclinesSmallCorpora(t *testing.T) {
	units, docs := fixture(20)
	res := Run(units, docs, comp(), labelWeights(units), DefaultOptions(0.01, 12))
	if res.Applied() {
		t.Fatalf("20-unit corpus calibrated: %+v", res)
	}
	if res.Threshold != 0 || res.StructMin != 0 {
		t.Error("declined result must carry zero thresholds")
	}
	if bad := Run(units, docs, comp(), labelWeights(units), Options{Rate: 0}); bad.Applied() {
		t.Error("rate 0 must decline")
	}
}

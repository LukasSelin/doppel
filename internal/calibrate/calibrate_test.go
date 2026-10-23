package calibrate

import (
	"fmt"
	"path/filepath"
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
	files := identityPaths(units, o.Root)
	order := canonicalOrder(units, files)
	var shapeIdx []int
	for _, i := range order {
		if units[i].Fingerprint.Nodes >= o.MinNodes {
			shapeIdx = append(shapeIdx, i)
		}
	}
	for _, p := range samplePopulation(units, shapeIdx, o.MaxPairs, identities(units, order, files), shapeSalt) {
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
	res := Run(units, docs, comp(), labelWeights(units), DefaultOptions(0.01, 12, ""))
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

// TestSampleStableUnderInsertion pins what bottom-k buys over a seeded
// generator: adding one function to the population leaves the null sample
// almost untouched. Only pairs involving the new function can enter, and each
// entrant displaces one incumbent, so the churn is bounded by that function's
// share of the pairs — about 2/m of the sample — instead of the whole draw.
func TestSampleStableUnderInsertion(t *testing.T) {
	units, _ := fixture(300)
	k := 2000
	key := func(us []parser.CodeUnit, p [2]int) string {
		return us[p[0]].Name + "|" + us[p[1]].Name
	}
	draw := func(us []parser.CodeUnit) map[string]bool {
		files := identityPaths(us, "")
		order := canonicalOrder(us, files)
		got := map[string]bool{}
		for _, p := range samplePopulation(us, order, k, identities(us, order, files), shapeSalt) {
			got[key(us, p)] = true
		}
		return got
	}
	before := draw(units)
	grown := append(append([]parser.CodeUnit{}, units...), parser.CodeUnit{Name: "Added", Package: "q", File: "q/b.go", StartLine: 1})
	after := draw(grown)
	kept := 0
	for p := range before {
		if after[p] {
			kept++
		}
	}
	// 2/m of 2000 is ~13 displaced pairs; allow generous slack.
	if lost := len(before) - kept; lost > 60 {
		t.Errorf("adding one unit displaced %d of %d null pairs; bottom-k should displace ~%d", lost, len(before), 2*k/len(units))
	}
	if len(after) != len(before) {
		t.Errorf("sample size moved %d -> %d", len(before), len(after))
	}
}

// TestIdentityIgnoresPathSpelling pins that the null sample, and so the
// derived floors, belong to the tree and not to how its paths were spelled.
// The walk hands calibration backslash paths on Windows and absolute ones when
// the root is given absolute, and both used to reach the identity hash raw:
// one pinned corpus calibrated to struct-min 0.30 on Linux and 0.31 on
// Windows. filepath.FromSlash only produces backslashes on Windows, so the
// native case can only fail there, which is why CI runs this test on every OS;
// the absolute-root case fails on all of them.
func TestIdentityIgnoresPathSpelling(t *testing.T) {
	base, docs := fixture(240)
	for i := range base {
		// Spread the units over directories, so the file is part of what
		// tells two identities apart rather than a prefix every unit shares.
		name := "a.go"
		if i%5 == 0 {
			name = "a_test.go"
		}
		base[i].File = fmt.Sprintf("p/d%d/%s", i%6, name)
		// And spread the overlap scores, so the top 1% of a sample depends on
		// which pairs it drew: under the raw-path hash, Windows separators
		// calibrated this fixture to struct-min 0.44 against the slash
		// spelling's 0.46.
		docs[i].Callees = []string{fmt.Sprintf("c%d", i%11), fmt.Sprintf("e%d", i%7), fmt.Sprintf("g%d", i%13)}
		docs[i].Callers = []string{fmt.Sprintf("k%d", i%9)}
	}
	abs := t.TempDir()
	respell := func(f func(string) string) []parser.CodeUnit {
		us := append([]parser.CodeUnit(nil), base...)
		for i := range us {
			us[i].File = f(base[i].File)
		}
		return us
	}
	spellings := []struct {
		name  string
		root  string
		units []parser.CodeUnit
	}{
		{"slash, root .", ".", base},
		{"native separators, root .", ".", respell(filepath.FromSlash)},
		{"absolute root", abs, respell(func(f string) string { return filepath.Join(abs, filepath.FromSlash(f)) })},
	}
	o := Options{Rate: 0.01, MaxPairs: 3000, MinNodes: 12, MinNullPairs: 100}
	sample := func(us []parser.CodeUnit, root string) [][2]int {
		files := identityPaths(us, root)
		order := canonicalOrder(us, files)
		return samplePopulation(us, order, o.MaxPairs, identities(us, order, files), overlapSalt)
	}
	run := func(us []parser.CodeUnit, root string) Result {
		ro := o
		ro.Root = root
		return Run(us, docs, comp(), labelWeights(us), ro)
	}
	want, wantRun := sample(base, "."), run(base, ".")
	if !wantRun.Applied() {
		t.Fatalf("fixture declined: %s", wantRun.Declined)
	}
	for _, sp := range spellings[1:] {
		if got := sample(sp.units, sp.root); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: the null sample differs from the slash spelling's", sp.name)
		}
		if got := run(sp.units, sp.root); got != wantRun {
			t.Errorf("%s: calibrated %+v, the slash spelling %+v", sp.name, got, wantRun)
		}
	}
}

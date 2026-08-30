package retriever

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// parseUnits parses one or more fixture files (path, source alternating) and
// returns the concatenated units in file-then-declaration order, mirroring
// the pipeline's walk order.
func parseUnits(t *testing.T, files ...string) []parser.CodeUnit {
	t.Helper()
	if len(files)%2 != 0 {
		t.Fatal("parseUnits needs (path, source) pairs")
	}
	var units []parser.CodeUnit
	for i := 0; i < len(files); i += 2 {
		parsed, err := parser.ParseSource(files[i], []byte(files[i+1]))
		if err != nil {
			t.Fatalf("parse %s: %v", files[i], err)
		}
		units = append(units, parsed...)
	}
	return units
}

// retrieveAll wires the same corpus statistics the pipeline builds: call
// graph from the units, IC from their (possibly hand-assigned) patterns.
func retrieveAll(t *testing.T, units []parser.CodeUnit, opt Options) ([]Candidate, Stats) {
	t.Helper()
	g := concepter.BuildCallGraph(units)
	onto := ontology.Default()
	counts := make(map[ontology.TermID]int)
	for i := range units {
		for _, tag := range parser.ConceptIDs(units[i].Concepts) {
			counts[ontology.TermID(tag)]++
		}
	}
	ic := ontology.NewCorpusIC(onto, counts)
	return Retrieve(units, g, onto, ic, opt)
}

func findCandidate(cands []Candidate, a, b int) (Candidate, bool) {
	if a > b {
		a, b = b, a
	}
	for _, c := range cands {
		if c.AIdx == a && c.BIdx == b {
			return c, true
		}
	}
	return Candidate{}, false
}

func unitIndex(t *testing.T, units []parser.CodeUnit, name string) int {
	t.Helper()
	for i := range units {
		if units[i].Name == name {
			return i
		}
	}
	t.Fatalf("no unit named %q", name)
	return -1
}

// A repetitive cluster in one package must not starve an isolated pair
// elsewhere of retrieval: per-function top-K gives every function its own
// slots, unlike the old global top-N over raw fingerprint score.
func TestPerFunctionRetrievalIsNotStarvedByClusters(t *testing.T) {
	var cluster strings.Builder
	cluster.WriteString("package clusterpkg\n")
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&cluster, `
func Clone%d(xs []int) int {
	total := 0
	for _, x := range xs {
		if x%%2 == 0 {
			total += x
		}
	}
	return total
}
`, i)
	}
	isolated := `package elsewhere

func CollapseRuns(xs []int) []int {
	var out []int
	for i, x := range xs {
		if i == 0 || xs[i-1] != x {
			out = append(out, x)
		}
	}
	for i := range out {
		out[i] *= 2
	}
	return out
}

func SquashRepeats(vals []int) []int {
	var kept []int
	for j, v := range vals {
		if j == 0 || vals[j-1] != v {
			kept = append(kept, v)
		}
	}
	for j := range kept {
		kept[j] *= 2
	}
	return kept
}
`
	units := parseUnits(t, "cluster.go", cluster.String(), "elsewhere.go", isolated)
	opt := DefaultOptions()
	opt.ChannelK = 3
	opt.MinNodes = 8
	opt.MaxPatternDF = 20

	cands, _ := retrieveAll(t, units, opt)
	a := unitIndex(t, units, "CollapseRuns")
	b := unitIndex(t, units, "SquashRepeats")
	if _, ok := findCandidate(cands, a, b); !ok {
		t.Fatalf("isolated pair CollapseRuns/SquashRepeats not retrieved; cluster consumed the candidate set")
	}
}

// Retrieval must be deterministic: identical corpus, identical candidates in
// identical order, and equal-evidence ties must break toward the lower index.
func TestRetrieveDeterministic(t *testing.T) {
	src := `package fix

func AlphaWalk(xs []int) int {
	total := 0
	for _, x := range xs {
		if x > 3 {
			total += x
		}
	}
	return total
}

func BetaWalk(ys []int) int {
	sum := 0
	for _, y := range ys {
		if y > 3 {
			sum += y
		}
	}
	return sum
}

func GammaWalk(zs []int) int {
	acc := 0
	for _, z := range zs {
		if z > 3 {
			acc += z
		}
	}
	return acc
}
`
	units := parseUnits(t, "fix.go", src)
	units[0].Concepts = parser.Certain("db_access")
	units[1].Concepts = parser.Certain("db_access")
	units[2].Concepts = parser.Certain("db_access")

	opt := DefaultOptions()
	opt.MinNodes = 8

	first, firstStats := retrieveAll(t, units, opt)
	for i := 0; i < 25; i++ {
		cands, stats := retrieveAll(t, units, opt)
		if !reflect.DeepEqual(cands, first) || stats != firstStats {
			t.Fatalf("run %d diverged from first run", i)
		}
	}
}

// With ChannelK=1 and three identical functions, every function's two
// neighbors carry exactly equal mass on every channel; the tie must resolve
// to the lower index, giving exactly {(0,1), (0,2)}.
func TestRetrieveTieBreaksTowardLowerIndex(t *testing.T) {
	src := `package fix

func AlphaWalk(xs []int) int {
	total := 0
	for _, x := range xs {
		if x > 3 {
			total += x
		}
	}
	return total
}

func BetaWalk(ys []int) int {
	sum := 0
	for _, y := range ys {
		if y > 3 {
			sum += y
		}
	}
	return sum
}

func GammaWalk(zs []int) int {
	acc := 0
	for _, z := range zs {
		if z > 3 {
			acc += z
		}
	}
	return acc
}

func DifferentShape(m map[string]int) string {
	best := ""
	for k, v := range m {
		switch {
		case v > 10:
			best = k
		case k == "":
			best = "empty"
		}
	}
	return best
}
`
	units := parseUnits(t, "fix.go", src)
	opt := DefaultOptions()
	opt.ChannelK = 1
	opt.MinNodes = 8

	cands, _ := retrieveAll(t, units, opt)
	var got [][2]int
	for _, c := range cands {
		got = append(got, [2]int{c.AIdx, c.BIdx})
	}
	want := [][2]int{{0, 1}, {0, 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates = %v, want %v (ties break toward the lower index)", got, want)
	}
}

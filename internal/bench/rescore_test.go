package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// A tiny synthetic corpus: two renamed clones sharing a resolved internal
// callee, so both the shape and call channels admit the pair. No real corpus
// identifiers, per the leak rule.
const rescoreFixture = `package corpus

import "fmt"

func helper(s string) string { return fmt.Sprintf("x-%s", s) }

func AlphaProcess(items []string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		if it == "" {
			continue
		}
		out = append(out, helper(it))
	}
	return out
}

func BetaProcess(values []string) []string {
	res := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		res = append(res, helper(v))
	}
	return res
}
`

// TestRescoreRoundTrip pins the seam's core guarantee: Rescore under a
// reweighted vocabulary changes overlap scores, and Rescore back to the
// default restores them exactly — bit-identical, because weight overrides
// touch nothing upstream of the comparator.
func TestRescoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "corpus.go"), []byte(rescoreFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	units, err := Load(dir, PopInclude)
	if err != nil {
		t.Fatal(err)
	}
	run := Analyze(units, retriever.DefaultOptions())
	if len(run.Pairs) == 0 {
		t.Fatal("fixture produced no pairs; the round-trip has nothing to check")
	}

	original := make([]float64, len(run.Pairs))
	for i, p := range run.Pairs {
		original[i] = p.Evidence.OverlapScore
	}

	ablated, err := ontology.WithWeights(map[ontology.TermID]float64{ontology.RelCalls: 0})
	if err != nil {
		t.Fatal(err)
	}
	run.Rescore(ablated)
	changed := false
	for i, p := range run.Pairs {
		if p.Evidence.OverlapScore != original[i] {
			changed = true
			break
		}
	}
	if !changed {
		t.Error("zeroing the calls weight moved no overlap score; the seam is not reaching Compare")
	}

	run.Rescore(ontology.Default())
	for i, p := range run.Pairs {
		if p.Evidence.OverlapScore != original[i] {
			t.Fatalf("pair %d: overlap %v after round-trip, want %v exactly",
				i, p.Evidence.OverlapScore, original[i])
		}
	}
}

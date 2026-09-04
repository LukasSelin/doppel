package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LukasSelin/doppel/internal/fingerprint"
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
	saved := run.Onto

	ablated, err := ontology.WithWeightsOver(saved, map[ontology.TermID]float64{ontology.RelCalls: 0})
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

	run.Rescore(saved)
	for i, p := range run.Pairs {
		if p.Evidence.OverlapScore != original[i] {
			t.Fatalf("pair %d: overlap %v after round-trip, want %v exactly",
				i, p.Evidence.OverlapScore, original[i])
		}
	}
}

// Reretrieve under a different fingerprint blend moves code-shape scores
// (and therefore admission), and Reretrieve back to the defaults restores
// every Breakdown and retrieval mass bit-identically — tags, IC, the graph
// and the docs were never touched.
func TestReretrieveRoundTrip(t *testing.T) {
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
		t.Fatal("fixture produced no pairs")
	}
	type snap struct {
		score, total float64
	}
	original := make([]snap, len(run.Pairs))
	for i, p := range run.Pairs {
		original[i] = snap{p.Score, p.Retrieval.Total}
	}

	// The fixture pair is an exact renamed clone — every component is 1.0,
	// so any blend summing to 1 still scores 1.0. Probe with a blend that
	// does not sum to 1, which only the seam can see.
	opt := retriever.DefaultOptions()
	opt.Weights = fingerprint.Weights{WL: 0.3, Flow: 0.2, Depth: 0.05, Signature: 0.15}
	run.Reretrieve(opt)
	changed := false
	for i, p := range run.Pairs {
		if i < len(original) && p.Score != original[i].score {
			changed = true
		}
	}
	if !changed {
		t.Error("halving the AST weight moved no code-shape score; the seam is not reaching the sim cache")
	}

	run.Reretrieve(retriever.DefaultOptions())
	if len(run.Pairs) != len(original) {
		t.Fatalf("round-trip changed the pair count: %d vs %d", len(run.Pairs), len(original))
	}
	for i, p := range run.Pairs {
		if p.Score != original[i].score || p.Retrieval.Total != original[i].total {
			t.Fatalf("pair %d not restored: %v/%v vs %v/%v", i, p.Score, p.Retrieval.Total, original[i].score, original[i].total)
		}
	}
}

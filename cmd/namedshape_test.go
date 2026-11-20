package cmd

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
)

// A pair's named-shape is the vocab lens scored against that lens's own
// corpus weights, on every pair the run hands back; and a run that opts out
// of it — every hook does — differs in nothing else. Runs on doppel's own
// tree, like the decisions test.
func TestNamedShapeIsTheVocabLens(t *testing.T) {
	root := repoRoot(t)
	p := Params{Threshold: 0.60, MinNodes: 16, ChannelK: 5, TestsMode: "exclude", Generated: "exclude"}

	with, err := analyze(root, p, io.Discard)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(with.Pairs) == 0 {
		t.Fatal("no pairs on doppel's own tree")
	}

	// The reference reading, from an index-only run that still has the trees:
	// the full bags, the full weight table, the plain overlap.
	ref, err := index(root, p, io.Discard, nil)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if len(ref.Units) != len(with.Units) {
		t.Fatalf("index read %d units, analyze %d", len(ref.Units), len(with.Units))
	}
	bags := analyzer.BuildVocab(ref.Units)
	idf := fingerprint.LabelWeights(bags)
	below := 0
	for _, pr := range with.Pairs {
		if !pr.HasNamedShape {
			t.Fatalf("pair %d-%d carries no named-shape", pr.AIdx, pr.BIdx)
		}
		want, _ := fingerprint.WLOverlap(bags[pr.AIdx], bags[pr.BIdx], idf)
		if pr.NamedShape != want {
			t.Fatalf("pair %d-%d: named-shape %v, the vocab lens reads %v", pr.AIdx, pr.BIdx, pr.NamedShape, want)
		}
		if pr.NamedShape < pr.Breakdown.WL {
			below++
		}
	}
	if below == 0 {
		t.Error("no pair reads lower with names kept than without: the lens is not keeping them")
	}

	p.NoNamedShape = true
	without, err := analyze(root, p, io.Discard)
	if err != nil {
		t.Fatalf("analyze (no named-shape): %v", err)
	}
	if without.Named != nil {
		t.Error("an opted-out run still built the index")
	}
	for _, pr := range without.Pairs {
		if pr.HasNamedShape {
			t.Fatalf("pair %d-%d carries a named-shape in an opted-out run", pr.AIdx, pr.BIdx)
		}
	}
	// Nothing else moves: kinds included, which read the same small bodies'
	// bags whichever way they were built.
	measured := snapshotOf(with, with.Pairs)
	for i := range measured.Pairs {
		if measured.Pairs[i].NamedShape < 0 {
			t.Fatalf("snapshot pair %d stored an unmeasured named-shape", i)
		}
		measured.Pairs[i].NamedShape = -1
	}
	a, _ := json.Marshal(measured)
	b, _ := json.Marshal(snapshotOf(without, without.Pairs))
	if string(a) != string(b) {
		t.Fatal("opting out of named-shape changed something else in the run")
	}
}

// Hooks never show the number, so they never pay for it.
func TestHookParamsSkipNamedShape(t *testing.T) {
	p, err := hookParams(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !p.NoNamedShape {
		t.Error("hook runs must opt out of the named-shape index")
	}
}

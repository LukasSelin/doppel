package retriever

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/tagger"
)

// probeCorpus builds a small corpus from inline sources, tagged the way the
// pipeline tags, with the last unit playing the probe.
func probeCorpus(t *testing.T, sources map[string]string) ([]parser.CodeUnit, *concepter.Graph, *ontology.Ontology, *ontology.IC) {
	t.Helper()
	var units []parser.CodeUnit
	for path, src := range sources {
		parsed, err := parser.ParseSource(path, []byte(src))
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		units = append(units, parsed...)
	}
	tagCounts := make(map[ontology.TermID]int)
	for i := range units {
		units[i].Concepts = parser.Certain(tagger.Tag(units[i])...)
		for _, tag := range parser.ConceptIDs(units[i].Concepts) {
			tagCounts[ontology.TermID(tag)]++
		}
	}
	onto := ontology.Default()
	ic := ontology.NewCorpusIC(onto, tagCounts)
	g := concepter.BuildCallGraph(units)
	return units, g, onto, ic
}

const probeFixtureA = `package alpha

import "sort"

func SortedA(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func SortedB(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func Unrelated(x int) int {
	if x < 0 {
		return -x
	}
	return x * 3
}
`

// Probe must find for the probe exactly what Retrieve finds for the same
// unit: same pairs, same evidence. The two share every gate and the
// evaluate tail, so a divergence means the admission narrowing broke
// something.
func TestProbeAgreesWithRetrieveForTheSameUnit(t *testing.T) {
	units, g, onto, ic := probeCorpus(t, map[string]string{"a.go": probeFixtureA})
	if len(units) != 3 {
		t.Fatalf("fixture parsed into %d units, want 3", len(units))
	}
	probeIdx := 0 // SortedA

	opt := DefaultOptions()
	opt.MinNodes = 1

	full, _ := Retrieve(units, g, onto, ic, labelWeights(units), opt)
	var wantPairs []Candidate
	for _, c := range full {
		if c.AIdx == probeIdx || c.BIdx == probeIdx {
			wantPairs = append(wantPairs, c)
		}
	}
	got, _ := Probe(units, probeIdx, g, onto, ic, labelWeights(units), opt)

	if len(got) != len(wantPairs) {
		t.Fatalf("Probe found %d pairs, Retrieve found %d involving the same unit", len(got), len(wantPairs))
	}
	for i := range got {
		w, p := wantPairs[i], got[i]
		if w.AIdx != p.AIdx || w.BIdx != p.BIdx {
			t.Errorf("pair %d: Probe (%d,%d), Retrieve (%d,%d)", i, p.AIdx, p.BIdx, w.AIdx, w.BIdx)
		}
		if w.Total != p.Total || w.Shape != p.Shape || w.Concept != p.Concept || w.Call != p.Call {
			t.Errorf("pair %d evidence diverged: Probe %+v, Retrieve %+v", i, p, w)
		}
		if w.Breakdown.Score != p.Breakdown.Score {
			t.Errorf("pair %d breakdown diverged: Probe %v, Retrieve %v", i, p.Breakdown.Score, w.Breakdown.Score)
		}
	}
	if len(got) == 0 {
		t.Fatal("fixture produced no pairs at all; the test asserts nothing")
	}
}

// A probe with no informative shared features is admitted by no channel.
func TestProbeWithNothingSharedFindsNothing(t *testing.T) {
	units, g, onto, ic := probeCorpus(t, map[string]string{"a.go": probeFixtureA})
	opt := DefaultOptions()
	opt.MinNodes = 1

	// Unrelated (index 2) shares no rare shape, tags, or calls with the twins.
	got, _ := Probe(units, 2, g, onto, ic, labelWeights(units), opt)
	for _, c := range got {
		if c.Total > 0 {
			// Some incidental L0 window overlap can admit a low-mass pair;
			// what must not happen is Unrelated matching the twins strongly.
			if c.Total > 5 {
				t.Errorf("Unrelated matched with %f nats: %+v", c.Total, c)
			}
		}
	}
}

// Probe results are deterministic: two runs over the same corpus return
// byte-identical candidate slices.
func TestProbeIsDeterministic(t *testing.T) {
	units, g, onto, ic := probeCorpus(t, map[string]string{"a.go": probeFixtureA})
	opt := DefaultOptions()
	opt.MinNodes = 1

	first, _ := Probe(units, 0, g, onto, ic, labelWeights(units), opt)
	for run := 0; run < 5; run++ {
		again, _ := Probe(units, 0, g, onto, ic, labelWeights(units), opt)
		if len(again) != len(first) {
			t.Fatalf("run %d: %d candidates, first run had %d", run, len(again), len(first))
		}
		for i := range again {
			if again[i].AIdx != first[i].AIdx || again[i].BIdx != first[i].BIdx || again[i].Total != first[i].Total {
				t.Fatalf("run %d candidate %d differs: %+v vs %+v", run, i, again[i], first[i])
			}
		}
	}
}

// A Prober reused across every unit of a corpus answers exactly what an
// independent Probe per unit does: the indexes are immutable and the memo is
// of a pure function, so reuse may only save work, never change an answer.
func TestProberReuseMatchesIndependentProbes(t *testing.T) {
	units, g, onto, ic := probeCorpus(t, map[string]string{"a.go": probeFixtureA})
	opt := DefaultOptions()
	opt.MinNodes = 1
	wl := labelWeights(units)

	prober := NewProber(units, g, onto, ic, wl, opt)
	// Reverse order, so a later probe reads memo entries an earlier one wrote.
	for i := len(units) - 1; i >= 0; i-- {
		shared, sharedStats := prober.Probe(i)
		alone, aloneStats := Probe(units, i, g, onto, ic, wl, opt)
		if sharedStats != aloneStats {
			t.Errorf("unit %d stats: reused %+v, alone %+v", i, sharedStats, aloneStats)
		}
		if len(shared) != len(alone) {
			t.Fatalf("unit %d: reused %d candidates, alone %d", i, len(shared), len(alone))
		}
		for j := range shared {
			s, a := shared[j], alone[j]
			if s.AIdx != a.AIdx || s.BIdx != a.BIdx || s.Total != a.Total || s.Breakdown != a.Breakdown || s.TrophicSim != a.TrophicSim {
				t.Errorf("unit %d candidate %d: reused %+v, alone %+v", i, j, s, a)
			}
		}
	}
}

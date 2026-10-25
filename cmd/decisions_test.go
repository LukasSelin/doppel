package cmd

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
)

// The decisions log must account for every pair retrieval proposed, exactly
// once, and keeping the dropped pairs for it must change nothing the run
// reports. Runs on doppel's own tree with a struct-min filter on, so every
// fate but the cross-build ones is reachable.
func TestDecisionsCoverEveryRetrievedPair(t *testing.T) {
	root := repoRoot(t)
	p := Params{Threshold: 0.60, MinNodes: 16, ChannelK: 5, StructMin: 0.4, TestsMode: "exclude", Generated: "exclude"}

	plain, err := analyze(root, p, io.Discard)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	p.KeepDropped = true
	kept, err := analyze(root, p, io.Discard)
	if err != nil {
		t.Fatalf("analyze (keep dropped): %v", err)
	}
	if plain.Dropped != nil {
		t.Errorf("KeepDropped off still kept %d dropped pairs", len(plain.Dropped))
	}

	a, _ := json.Marshal(snapshotOf(plain, plain.Pairs))
	b, _ := json.Marshal(snapshotOf(kept, kept.Pairs))
	if string(a) != string(b) {
		t.Fatal("KeepDropped changed the run's pairs")
	}

	analyzer.SortForReport(kept.Pairs, kept.Units, 20, 2)
	log := buildDecisions(&kept, kept.Pairs, 20, 2)
	c := log.Counts
	if c.Retrieved != kept.Retrieval.Union {
		t.Errorf("log holds %d pairs, retrieval proposed %d", c.Retrieved, kept.Retrieval.Union)
	}
	if c.Reported+c.Capped+c.BelowTop+c.BelowStructMin+c.CrossBuildUnit+c.CrossTarget != c.Retrieved {
		t.Errorf("fates do not partition the log: %+v", c)
	}
	if c.Reported != 20 || c.BelowStructMin == 0 {
		t.Errorf("want 20 reported and some struct-min drops on doppel's own tree, got %+v", c)
	}

	seen := map[string]bool{}
	for _, d := range log.Pairs {
		k := d.A + "@" + d.AFile + "|" + d.B + "@" + d.BFile
		if seen[k] {
			t.Errorf("pair %s logged twice", k)
		}
		seen[k] = true
		if d.Overlap != nil && d.MergeWorthy != (len(d.Blocked) == 0) {
			t.Errorf("%s: mergeWorthy %v with blocked %v", k, d.MergeWorthy, d.Blocked)
		}
		if d.Fate == dropBelowStructMin && (d.Overlap == nil || *d.Overlap > p.StructMin) {
			t.Errorf("%s: dropped under struct-min at overlap %v", k, d.Overlap)
		}
		if d.Fate == analyzer.FateCapped && d.CappedBy != d.A && d.CappedBy != d.B {
			t.Errorf("%s: capped by %q, not one of its sides", k, d.CappedBy)
		}
	}
}

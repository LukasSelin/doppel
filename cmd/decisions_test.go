package cmd

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/reporter"
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
	log := buildDecisions(&kept, kept.Pairs, 20, 2, reporter.ListInclude)
	c := log.Counts
	if c.Retrieved != kept.Retrieval.Union {
		t.Errorf("log holds %d pairs, retrieval proposed %d", c.Retrieved, kept.Retrieval.Union)
	}
	if c.Reported+c.Capped+c.BelowTop+c.BelowStructMin+c.CrossBuildUnit+c.CrossTarget != c.Retrieved {
		t.Errorf("fates do not partition the log: %+v", c)
	}
	if c.Reported != 20 || c.BelowStructMin == 0 || c.NotListed != 0 {
		t.Errorf("want 20 reported, some struct-min drops and nothing unlisted on doppel's own tree, got %+v", c)
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

// Under the default --non-merges the log and the report are cut the same
// way: every pair whose kind says it is not a merge is logged as not listed,
// no such pair is ranked, and the ranks are the report's own.
func TestDecisionsFollowTheListCut(t *testing.T) {
	root := repoRoot(t)
	p := Params{Threshold: 0.60, MinNodes: 16, ChannelK: 5, StructMin: 0.4, TestsMode: "exclude", Generated: "exclude", KeepDropped: true}
	res, err := analyze(root, p, io.Discard)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	analyzer.SortForReport(res.Pairs, res.Units, 20, 2)

	// What runAnalyze prints: the cut, then the cap walk.
	visible, left := cutList(res.Pairs, reporter.ListExclude)
	if left.Total == 0 {
		t.Fatal("doppel's own tree has no pair to leave out, so this test proves nothing")
	}
	listed, _ := analyzer.SortForReport(visible, res.Units, 20, 2)

	log := buildDecisions(&res, res.Pairs, 20, 2, reporter.ListExclude)
	c := log.Counts
	if c.NotListed != left.Total {
		t.Errorf("log leaves out %d pairs, the report %d", c.NotListed, left.Total)
	}
	if c.Reported+c.Capped+c.BelowTop+c.NotListed+c.BelowStructMin+c.CrossBuildUnit+c.CrossTarget != c.Retrieved {
		t.Errorf("fates do not partition the log: %+v", c)
	}
	if c.Reported != len(listed) {
		t.Fatalf("log reports %d pairs, the report lists %d", c.Reported, len(listed))
	}
	if log.Point.NonMerges != reporter.ListExclude {
		t.Errorf("operating point records mode %q", log.Point.NonMerges)
	}
	for _, d := range log.Pairs {
		nonMerge := analyzer.NotAMerge(d.Kind)
		switch d.Fate {
		case fateNotListed:
			if !nonMerge || d.Rank != 0 || d.Position == 0 {
				t.Errorf("%s <-> %s: not listed with kind %q, rank %d, position %d", d.A, d.B, d.Kind, d.Rank, d.Position)
			}
		case analyzer.FateReported:
			if nonMerge {
				t.Errorf("%s <-> %s: reported with kind %q", d.A, d.B, d.Kind)
			}
			// Rank r is the report's r-th pair.
			want := newDecision(&res, listed[d.Rank-1], analyzer.FateReported, analyzer.DefaultRankOptions())
			if d.A != want.A || d.B != want.B || d.AFile != want.AFile || d.BFile != want.BFile {
				t.Errorf("rank %d is %s <-> %s in the log and %s <-> %s in the report", d.Rank, d.A, d.B, want.A, want.B)
			}
		case analyzer.FateCapped, analyzer.FateBelowTop:
			if nonMerge {
				t.Errorf("%s <-> %s: %s with kind %q, which the list never considered", d.A, d.B, d.Fate, d.Kind)
			}
		}
	}

	// `only` is the same cut the other way round.
	only := buildDecisions(&res, res.Pairs, 20, 2, reporter.ListOnly)
	for _, d := range only.Pairs {
		if d.Fate == analyzer.FateReported && !analyzer.NotAMerge(d.Kind) {
			t.Errorf("%s <-> %s: reported under only with kind %q", d.A, d.B, d.Kind)
		}
		if d.Fate == fateNotListed && analyzer.NotAMerge(d.Kind) {
			t.Errorf("%s <-> %s: left out under only with kind %q", d.A, d.B, d.Kind)
		}
	}
}

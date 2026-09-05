package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestMemStatsNoiseFloor measures how reproducible the memstats numbers are, so
// a footprint claim can be checked against the floor rather than asserted.
//
//	DOPPEL_BENCH_MEMSTATS=<corpus path> go test ./cmd/ -run TestMemStatsNoiseFloor -v
//
// It asserts the floor rather than merely printing it, because the whole point
// of this harness is that its numbers are exact: reachable-heap accounting is a
// runtime counter, not a sample, so repeated runs of one binary must agree to
// well within anything worth optimising. The bar is deliberately loose against
// what is actually measured (spreads of ~0.02MB on the settled stages) so it
// fails on a regression in the harness, not on a busy machine.
//
// The one stage that legitimately moves is any taken while culture is still
// running on its own goroutine: how much of its scratch is live at that instant
// depends on the scheduler. That is a true fact about an overlapped pipeline,
// not noise in the instrument, so the bar below is per-stage and generous.
func TestMemStatsNoiseFloor(t *testing.T) {
	corpus := os.Getenv("DOPPEL_BENCH_MEMSTATS")
	if corpus == "" {
		t.Skip("set DOPPEL_BENCH_MEMSTATS=<corpus path> to run")
	}
	const runs = 3
	all := make([][]MemRecord, runs)
	for i := range runs {
		path := filepath.Join(t.TempDir(), fmt.Sprintf("ms%d.json", i))
		all[i] = runMemStats(t, corpus, path)
	}

	for s := range all[0] {
		stage := all[0][s].Stage
		var live, alloc []float64
		for r := range runs {
			if all[r][s].Stage != stage {
				t.Fatalf("run %d stage %d is %q, want %q — the stage list is not stable",
					r, s, all[r][s].Stage, stage)
			}
			live = append(live, mb(all[r][s].Live))
			alloc = append(alloc, mb(all[r][s].Alloc))
		}
		ls, as := spread(live), spread(alloc)
		t.Logf("%-22s live spread %6.3fMB   alloc spread %6.3fMB", stage, ls, as)

		// 5MB is ~200x the measured spread on a settled stage and still an
		// order of magnitude tighter than the sampled heap profile this
		// replaced, which read 129.6/137.8/143.2MB for one stage of one binary.
		if ls > 5 {
			t.Errorf("%s: live spread %.3fMB across %d runs — the harness is not exact",
				stage, ls, runs)
		}
		if as > 5 {
			t.Errorf("%s: alloc spread %.3fMB across %d runs", stage, as, runs)
		}
	}
}

func spread(v []float64) float64 {
	lo, hi := v[0], v[0]
	for _, x := range v {
		lo, hi = min(lo, x), max(hi, x)
	}
	return hi - lo
}

func runMemStats(t *testing.T, corpus, out string) []MemRecord {
	t.Helper()
	t.Setenv("DOPPEL_MEMSTATS", out)
	// A fresh recorder per run: the shared one is a process-wide singleton, so
	// a test driving several runs in one process has to reset it.
	resetMemRecorder()

	p := Params{
		Threshold: defaultThreshold, TopN: 20, MinNodes: defaultMinNodes,
		ChannelK: 5, MaxPerFunc: 2, TestsMode: "exclude", Generated: "exclude",
		Calibrate: defaultCalibrateRate,
	}
	if _, err := analyze(corpus, p, nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var recs []MemRecord
	if err := json.Unmarshal(b, &recs); err != nil {
		t.Fatal(err)
	}
	if len(recs) == 0 {
		t.Fatal("no records written")
	}
	return recs
}

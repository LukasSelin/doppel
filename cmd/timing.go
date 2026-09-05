package cmd

import (
	"fmt"
	"io"
	"os"
	"time"
)

// DOPPEL_TIMING=1 prints the wall-clock cost of each pipeline stage next to the
// progress lines that already mark those stage boundaries.
//
// An env gate rather than a flag, following the DOPPEL_BENCH_* convention: this
// is a measurement seam, not an operating point, and it must not travel in
// Params or reach a config key — what a run cost has no bearing on what it
// measured, and a baseline must not go incomparable because somebody timed it.
//
// It writes to the pipeline's own progress writer, which is what keeps a hook
// silent: hook runs pass io.Discard, so even with the variable exported into a
// session's environment no hook can emit a byte on stderr. Unset — the default
// on every path — a timer emits nothing at all, so an untimed run is
// byte-identical on stderr to one from before this file existed.
//
// The numbers are wall-clock and single-run: they say where a run's seconds
// went, not what a stage costs on average. `task bench` is the repeatable
// measurement; this is the one that works on the corpus actually in front of
// you, including the stages bench does not model.
type stageTimer struct {
	w     io.Writer
	start time.Time
	last  time.Time
	on    bool
}

func newStageTimer(progress io.Writer) *stageTimer {
	now := time.Now()
	return &stageTimer{w: progressOr(progress), start: now, last: now, on: timingEnabled()}
}

func timingEnabled() bool {
	v := os.Getenv("DOPPEL_TIMING")
	return v != "" && v != "0"
}

// mark reports the time since the previous mark as the cost of the stage that
// just finished, and the elapsed total beside it. Stage names are the pipeline
// vocabulary, not the function names, so a reader can line a timing line up
// with the stage list in CLAUDE.md.
func (t *stageTimer) mark(stage string) {
	if !t.on {
		return
	}
	now := time.Now()
	fmt.Fprintf(t.w, "  timing: %-22s %7.2fs  (total %6.2fs)\n",
		stage, now.Sub(t.last).Seconds(), now.Sub(t.start).Seconds())
	t.last = now
}

// total closes the report with the run's own end-to-end figure, which is what a
// reader compares the stage lines against. It is deliberately not the number
// `time doppel analyze` prints: process start, flag parsing and report
// rendering all sit outside the pipeline, and pretending otherwise would make
// the stage lines fail to sum.
func (t *stageTimer) total(label string) {
	if !t.on {
		return
	}
	fmt.Fprintf(t.w, "  timing: %-22s %7.2fs\n", label, time.Since(t.start).Seconds())
}

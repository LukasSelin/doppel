package bench

import (
	"os"
	"testing"

	"github.com/LukasSelin/doppel/internal/retriever"
)

// thresholdLadder is the set of code-shape floors TestThresholdLadder scores.
// It spans the range the null calibration derives across the public ladder
// (prometheus 0.33 to chi 0.45 at rate 0.01) plus the historical 0.60, so the
// default's derivation is re-runnable rather than a number in a commit message.
func thresholdLadder() []float64 {
	return []float64{0.30, 0.33, 0.35, 0.38, 0.41, 0.44, 0.45, 0.50, 0.60}
}

// TestThresholdLadder scores every labeled corpus at each candidate
// --threshold, re-running retrieval through the Reretrieve seam. It exists
// because the shipped default is derived from `--calibrate 0.01` medians over
// the public corpus ladder, and a median over corpora is only defensible if
// the labeled corpus is shown not to regress at the value chosen.
//
// It asserts nothing — like TestSweep, TestMinIDF and TestCalibrate, this is a
// measurement, not a gate. `task golden` is the gate.
//
//	DOPPEL_BENCH_THRESHOLD=1 go test ./internal/bench/ -v -run TestThresholdLadder
func TestThresholdLadder(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_THRESHOLD") != "1" {
		t.Skip("set DOPPEL_BENCH_THRESHOLD=1 to run the threshold ladder")
	}
	corpora := loadLabeledCorpora(t)
	if len(corpora) == 0 {
		t.Skip("no labeled corpora available; run `task corpora` or set DOPPEL_BENCH_CORPUS/DOPPEL_BENCH_LABELS")
	}

	for _, lc := range corpora {
		saved := snapshotRetrieval(lc.run)
		for _, th := range thresholdLadder() {
			opt := retriever.DefaultOptions()
			opt.Threshold = th
			lc.run.Reretrieve(opt)
			sc := Score(lc.run, lc.lf)
			t.Logf("[%s] threshold %.2f  pairs %5d  %s", lc.name, th, len(lc.run.Pairs), scLine(sc))
		}
		saved.restore(lc.run)
	}
}

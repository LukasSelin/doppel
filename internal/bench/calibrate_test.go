package bench

import (
	"os"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/calibrate"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// TestCalibrate measures null calibration against the golden labels: for
// each rate, derive the thresholds from the corpus, re-retrieve at the
// calibrated code-shape threshold, and score — once on the full candidate
// set (what retrieval admits) and once after dropping pairs below the
// calibrated struct-min (what the report shows). It asserts nothing.
//
//	DOPPEL_BENCH_CALIBRATE=1 go test ./internal/bench/ -v -run TestCalibrate
func TestCalibrate(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_CALIBRATE") != "1" {
		t.Skip("set DOPPEL_BENCH_CALIBRATE=1 to run the calibration measurement")
	}
	corpora := loadLabeledCorpora(t)
	if len(corpora) == 0 {
		t.Skip("no labeled corpora available; run `task corpora` or set DOPPEL_BENCH_CORPUS/DOPPEL_BENCH_LABELS")
	}
	for i := range corpora {
		lc := &corpora[i]
		base := Score(lc.run, lc.lf)
		t.Logf("[%s] fixed 0.60/none          %s", lc.name, scLine(base))
		saved := snapshotRetrieval(lc.run)
		for _, rate := range []float64{0.005, 0.01, 0.02, 0.05} {
			r := calibrate.Run(lc.run.Units, lc.run.Docs, lc.run.Comp, lc.run.WL,
				calibrate.DefaultOptions(rate, retriever.DefaultOptions().MinNodes))
			if !r.Applied() {
				t.Logf("[%s] rate %-5g declined: %s", lc.name, rate, r.Declined)
				continue
			}
			opt := retriever.DefaultOptions()
			opt.Threshold = r.Threshold
			lc.run.Reretrieve(opt)
			full := Score(lc.run, lc.lf)
			moved, _, _ := movement(base, full)

			// The report view: pairs below the calibrated struct-min are gone.
			all := lc.run.Pairs
			kept := make([]analyzer.SimilarPair, 0, len(all))
			for _, p := range all {
				if p.Evidence != nil && p.Evidence.OverlapScore >= r.StructMin {
					kept = append(kept, p)
				}
			}
			lc.run.Pairs = kept
			filtered := Score(lc.run, lc.lf)
			lc.run.Pairs = all

			t.Logf("[%s] rate %-5g thr %.2f sm %.2f  retrieved: %s", lc.name, rate, r.Threshold, r.StructMin, scLine(full))
			t.Logf("[%s]                            filtered:  %s  (%d of %d pairs kept)", lc.name, scLine(filtered), len(kept), len(all))
			if len(moved) > 0 {
				t.Logf("[%s]                            moved: %v", lc.name, moved)
			}
			saved.restore(lc.run)
		}
	}
}

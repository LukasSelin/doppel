package bench

import (
	"os"
	"testing"

	"github.com/LukasSelin/doppel/internal/retriever"
)

// TestMinIDF measures the information floor against the absolute df caps:
// for each floor, re-retrieve with derived caps and report the caps, the
// channel statistics and the labeled rankings. It asserts nothing; the
// adoption rule is written in CLAUDE.md next to the measured table.
//
//	DOPPEL_BENCH_MINIDF=1 go test ./internal/bench/ -v -run TestMinIDF
func TestMinIDF(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_MINIDF") != "1" {
		t.Skip("set DOPPEL_BENCH_MINIDF=1 to run the information-floor measurement")
	}
	corpora := loadLabeledCorpora(t)
	if len(corpora) == 0 {
		t.Skip("no labeled corpora available; run `task corpora` or set DOPPEL_BENCH_CORPUS/DOPPEL_BENCH_LABELS")
	}
	for i := range corpora {
		lc := &corpora[i]
		base := Score(lc.run, lc.lf)
		st := lc.run.Stats
		t.Logf("[%s] fixed caps 50/50      union %5d  suppressed %4d  patterns %6d  %s",
			lc.name, st.Union, st.Suppressed, st.SurvivingPatterns, scLine(base))
		saved := snapshotRetrieval(lc.run)
		for _, floor := range []float64{1, 1.5, 2, 3, 4} {
			opt := retriever.DefaultOptions()
			opt.MinIDF = floor
			lc.run.Reretrieve(opt)
			sc := Score(lc.run, lc.lf)
			st := lc.run.Stats
			moved, _, _ := movement(base, sc)
			t.Logf("[%s] min-idf %-4g caps %3d/%-4d union %5d  suppressed %4d  patterns %6d  %s",
				lc.name, floor, st.PatternCap, st.CallCap, st.Union, st.Suppressed, st.SurvivingPatterns, scLine(sc))
			if len(moved) > 0 {
				t.Logf("[%s]              moved: %v", lc.name, moved)
			}
			saved.restore(lc.run)
		}
	}
}

// TestMinIDFLadder prints the channel statistics under each floor for every
// fetched corpus, labeled or not — the caps' effect on a corpus is visible
// in union size and suppression long before any label can judge it.
func TestMinIDFLadder(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_MINIDF") != "1" {
		t.Skip("set DOPPEL_BENCH_MINIDF=1 to run the information-floor measurement")
	}
	for _, c := range Corpora {
		if !Present(c) {
			continue
		}
		dir, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		units, err := Load(dir, PopExclude)
		if err != nil {
			t.Fatal(err)
		}
		run := Analyze(units, retriever.DefaultOptions())
		st := run.Stats
		t.Logf("[%s] %5d funcs  fixed 50/50      union %6d  shape %5d  suppressed %4d  patterns %6d",
			c.Name, len(units), st.Union, st.ShapePairs, st.Suppressed, st.SurvivingPatterns)
		for _, floor := range []float64{1, 1.5, 2, 3} {
			opt := retriever.DefaultOptions()
			opt.MinIDF = floor
			run.Reretrieve(opt)
			st := run.Stats
			t.Logf("[%s]             min-idf %-4g caps %4d/%-5d union %6d  shape %5d  suppressed %4d  patterns %6d",
				c.Name, floor, st.PatternCap, st.CallCap, st.Union, st.ShapePairs, st.Suppressed, st.SurvivingPatterns)
		}
	}
}

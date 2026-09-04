package bench

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/calibrate"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/ontology"
)

// TestSelfWeight is the label-free weighting experiment: which comparator
// signals actually separate retrieved candidates from random pairs on this
// corpus? Fisher ratio per signal, (μc−μn)²/(σc²+σn²+ε), over the candidate
// set versus a deterministic null sample, normalized to weights; plus an
// entropy variant (weight ∝ variance over candidates) for contrast. Both
// derived vectors are then scored against the golden labels through
// Rescore(WithWeights). Measurement only: nothing here changes relations.go.
//
//	DOPPEL_BENCH_SELFWEIGHT=1 go test ./internal/bench/ -v -run TestSelfWeight
func TestSelfWeight(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_SELFWEIGHT") != "1" {
		t.Skip("set DOPPEL_BENCH_SELFWEIGHT=1 to run the self-weighting experiment")
	}
	corpora := loadLabeledCorpora(t)
	if len(corpora) == 0 {
		t.Skip("no labeled corpora available; run `task corpora` or set DOPPEL_BENCH_CORPUS/DOPPEL_BENCH_LABELS")
	}
	def := ontology.Default()
	for i := range corpora {
		lc := &corpora[i]
		run := lc.run

		// Candidate arm: every scored pair, read by its own indices.
		var cand [][comparator.SignalCount]float64
		for _, p := range run.Pairs {
			if p.Evidence == nil {
				continue
			}
			cand = append(cand, comparator.SignalVector(*p.Evidence, run.Docs[p.AIdx], run.Docs[p.BIdx]))
		}
		// Null arm: random pairs over all units, cross test/prod dropped,
		// compared with the run's own comparator.
		var null [][comparator.SignalCount]float64
		for _, pr := range calibrate.SamplePairs(len(run.Units), 20000, calibrate.Seed(run.Units)) {
			if isTest(run.Units[pr[0]]) != isTest(run.Units[pr[1]]) {
				continue
			}
			ev := run.Comp.Compare(run.Docs[pr[0]], run.Docs[pr[1]])
			null = append(null, comparator.SignalVector(ev, run.Docs[pr[0]], run.Docs[pr[1]]))
		}

		fisher := make(map[ontology.TermID]float64)
		entropy := make(map[ontology.TermID]float64)
		var fSum, eSum float64
		for s := 0; s < comparator.SignalCount; s++ {
			mc, vc := meanVar(cand, s)
			mn, vn := meanVar(null, s)
			f := (mc - mn) * (mc - mn) / (vc + vn + 1e-9)
			fisher[comparator.SignalOrder[s]] = f
			entropy[comparator.SignalOrder[s]] = vc
			fSum += f
			eSum += vc
		}
		for _, rel := range def.ScoredRelations() {
			if fSum > 0 {
				fisher[rel] /= fSum
			}
			if eSum > 0 {
				entropy[rel] /= eSum
			}
		}

		base := Score(run, lc.lf)
		t.Logf("[%s] %d candidates vs %d null pairs — default %s", lc.name, len(cand), len(null), scLine(base))
		t.Logf("[%s] %-22s %8s %8s %8s", lc.name, "relation", "default", "fisher", "entropy")
		for _, rel := range def.ScoredRelations() {
			t.Logf("[%s] %-22s %8.4f %8.4f %8.4f", lc.name, rel, def.Weight(rel), fisher[rel], entropy[rel])
		}
		for _, variant := range []struct {
			name string
			w    map[ontology.TermID]float64
		}{{"fisher", fisher}, {"entropy", entropy}} {
			onto, err := ontology.WithWeightsOver(lc.onto, variant.w)
			if err != nil {
				t.Logf("[%s] %s: %v", lc.name, variant.name, err)
				continue
			}
			run.Rescore(onto)
			sc := Score(run, lc.lf)
			moved, _, _ := movement(base, sc)
			t.Logf("[%s] %-8s weights: %s", lc.name, variant.name, scLine(sc))
			if len(moved) > 0 {
				t.Logf("[%s]          moved: %s", lc.name, strings.Join(moved, "; "))
			}
		}
		run.Rescore(lc.onto)
	}
}

func meanVar(rows [][comparator.SignalCount]float64, s int) (mean, variance float64) {
	if len(rows) == 0 {
		return 0, 0
	}
	for _, r := range rows {
		mean += r[s]
	}
	mean /= float64(len(rows))
	for _, r := range rows {
		d := r[s] - mean
		variance += d * d
	}
	variance /= float64(len(rows))
	return mean, math.Max(variance, 0)
}

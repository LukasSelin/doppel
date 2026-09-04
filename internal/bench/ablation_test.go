package bench

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// labeledCorpus is one (analyzed corpus, labels) pair the sweep harness scores
// against: every committed labels file whose corpus is fetched, plus the
// private DOPPEL_BENCH_CORPUS/DOPPEL_BENCH_LABELS pair when set.
type labeledCorpus struct {
	name string
	run  *Run
	lf   LabelsFile
	// onto is the vocabulary the run was analyzed under — the per-corpus one
	// whose concept leaves are learned — captured before any Rescore replaces
	// run.Onto. Every weight variant reweights this (ontology.WithWeightsOver)
	// and restores it afterwards.
	onto *ontology.Ontology
}

// loadLabeledCorpora analyzes each available labeled corpus once. The Runs are
// meant to be Rescored repeatedly, under one rule: a variant reweights lc.onto,
// never ontology.Default(), and Rescores back to lc.onto when it is done. The
// default vocabulary does not contain the run's learned concepts, so scoring
// under it collapses every non-exact concept match as well as changing the
// weight — and leaving the run there would put every later variant against a
// baseline scorecard measured under a different taxonomy.
func loadLabeledCorpora(t *testing.T) []labeledCorpus {
	t.Helper()
	var out []labeledCorpus

	analyze := func(name, corpus string, lf LabelsFile) {
		units, err := Load(corpus, Population(lf.Population))
		if err != nil {
			t.Fatalf("%s: load corpus: %v", name, err)
		}
		if len(units) == 0 {
			t.Fatalf("%s: corpus yielded no functions", name)
		}
		t.Logf("%s: %d functions, %d labels", name, len(units), len(lf.Labels))
		run := Analyze(units, retriever.DefaultOptions())
		out = append(out, labeledCorpus{name: name, run: run, lf: lf, onto: run.Onto})
	}

	entries, err := filepath.Glob(filepath.Join(repoRoot(t), "examples", "labels", "*.labels.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range entries {
		name := strings.TrimSuffix(filepath.Base(path), ".labels.json")
		c, ok := Find(name)
		if !ok {
			t.Fatalf("%s: no corpus in the ladder is named %q", filepath.Base(path), name)
		}
		if !Present(c) {
			t.Logf("%s not fetched; skipping (run `task corpora`)", name)
			continue
		}
		corpus, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lf, err := ParseLabels(data)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		analyze(name, corpus, lf)
	}

	if corpus, labels := os.Getenv("DOPPEL_BENCH_CORPUS"), os.Getenv("DOPPEL_BENCH_LABELS"); corpus != "" && labels != "" {
		data, err := os.ReadFile(labels)
		if err != nil {
			t.Fatalf("read private labels: %v", err)
		}
		lf, err := ParseLabels(data)
		if err != nil {
			t.Fatalf("parse private labels: %v", err)
		}
		// The private corpus stays private: log under a fixed alias, never its
		// path or the labels file's own corpus name.
		analyze("private", corpus, lf)
	}
	return out
}

// violations counts a scorecard's hard-assertion failures.
func violations(sc Scorecard) int {
	return len(sc.MergeMissing) + len(sc.FPAboveMerge) + len(sc.FPInTop20)
}

// scLine is the one-line scorecard summary the sweep logs per variant.
func scLine(sc Scorecard) string {
	mean := func(class string) string {
		if sc.Present[class] == 0 {
			return "-"
		}
		return fmt.Sprintf("%.1f", sc.MeanRank[class])
	}
	return fmt.Sprintf("merge %s (%d/%d, top50 %d)  refactor %s  fp %s  violations %d",
		mean("merge"), sc.MergePresent, sc.MergeTotal, sc.MergeInTop50,
		mean("refactor"), mean("false_positive"), violations(sc))
}

// TestAblation zeroes each of the twelve relation weights in turn (the other
// eleven renormalized to keep the sum at 1.0), rescores every available
// labeled corpus, and logs the movement. It never asserts: the output is a
// measurement of which signals earn their weight, not a gate. Guarded because
// it wants fetched corpora and a few hundred compare passes:
//
//	DOPPEL_BENCH_ABLATION=1 go test ./internal/bench/ -v -run TestAblation
func TestAblation(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_ABLATION") != "1" {
		t.Skip("set DOPPEL_BENCH_ABLATION=1 to run the weight ablation")
	}
	corpora := loadLabeledCorpora(t)
	if len(corpora) == 0 {
		t.Skip("no labeled corpora available; run `task corpora` or set DOPPEL_BENCH_CORPUS/DOPPEL_BENCH_LABELS")
	}

	rels := ontology.Default().ScoredRelations()
	for _, lc := range corpora {
		base := Score(lc.run, lc.lf)
		t.Logf("[%s] baseline          %s", lc.name, scLine(base))

		for _, rel := range rels {
			onto, err := ontology.WithWeightsOver(lc.onto, map[ontology.TermID]float64{rel: 0})
			if err != nil {
				t.Fatalf("ablate %s: %v", rel, err)
			}
			lc.run.Rescore(onto)
			sc := Score(lc.run, lc.lf)

			// Rank deltas per label, so a signal that moves nothing is visibly
			// dead and one that saves a single pair is visibly load-bearing.
			moved, maxShift, presenceChanged := 0, 0, false
			for i, r := range sc.Results {
				if r.Rank == 0 || base.Results[i].Rank == 0 {
					if r.Rank != base.Results[i].Rank || r.Absent != base.Results[i].Absent {
						moved++
						presenceChanged = true
					}
					continue
				}
				if shift := r.Rank - base.Results[i].Rank; shift != 0 {
					moved++
					if abs(shift) > abs(maxShift) {
						maxShift = shift
					}
				}
			}
			shiftDesc := fmt.Sprintf("max shift %+d", maxShift)
			if presenceChanged {
				shiftDesc = "presence changed"
			} else if moved == 0 {
				shiftDesc = "no movement"
			}
			t.Logf("[%s] -%-22s %s  (%d/%d labels moved, %s)",
				lc.name, rel, scLine(sc), moved, len(sc.Results), shiftDesc)
		}
		lc.run.Rescore(lc.onto)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// TestFitWeights searches the 12-weight simplex for a vector that improves the
// labeled rankings: seeded random restarts around the default, then coordinate
// descent. The objective is the summed mean merge rank across corpora plus
// heavy penalties for hard-assertion violations, so a "better" vector that
// breaks a golden guarantee always loses.
//
// EXPERIMENTAL, measurement-only: the result is a logged report. It never
// writes weights anywhere — adopting a fitted vector is a hand edit of
// relations.go with its own golden run, and with 18-odd labeled pairs the
// honest reading of any fit is "direction to investigate", not "truth".
//
//	DOPPEL_BENCH_FIT=1 go test ./internal/bench/ -v -run TestFitWeights -timeout 60m
func TestFitWeights(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_FIT") != "1" {
		t.Skip("set DOPPEL_BENCH_FIT=1 to run the weight fitter")
	}
	corpora := loadLabeledCorpora(t)
	if len(corpora) == 0 {
		t.Skip("no labeled corpora available; run `task corpora` or set DOPPEL_BENCH_CORPUS/DOPPEL_BENCH_LABELS")
	}

	rels := ontology.Default().ScoredRelations()
	defaults := make([]float64, len(rels))
	for i, rel := range rels {
		defaults[i] = ontology.Default().Weight(rel)
	}

	// evaluate rescoreable objective for one weight vector (normalized here).
	evals := 0
	evaluate := func(w []float64) float64 {
		var sum float64
		for _, v := range w {
			sum += v
		}
		overrides := make(map[ontology.TermID]float64, len(rels))
		for i, rel := range rels {
			overrides[rel] = w[i] / sum
		}
		var obj float64
		for _, lc := range corpora {
			// Per corpus, because each run's vocabulary is its own: the
			// weights are shared, the taxonomy they are applied to is not.
			onto, err := ontology.WithWeightsOver(lc.onto, overrides)
			if err != nil {
				t.Fatalf("fit weights: %v", err)
			}
			lc.run.Rescore(onto)
			sc := Score(lc.run, lc.lf)
			if sc.Present["merge"] > 0 {
				obj += sc.MeanRank["merge"]
			}
			obj += 1000*float64(len(sc.MergeMissing)) +
				250*float64(len(sc.FPAboveMerge)) +
				250*float64(len(sc.FPInTop20))
		}
		evals++
		return obj
	}

	best := append([]float64(nil), defaults...)
	bestObj := evaluate(best)
	t.Logf("baseline objective %.2f over %d corpora", bestObj, len(corpora))

	// Seeded, so two runs of the fitter on the same labels agree.
	rng := rand.New(rand.NewSource(42))

	// Log-normal perturbations around the incumbent: multiplicative, so no
	// weight can go negative and small weights explore proportionally.
	const restarts = 60
	for i := 0; i < restarts; i++ {
		cand := make([]float64, len(best))
		for j := range cand {
			cand[j] = best[j] * math.Exp(0.35*rng.NormFloat64())
		}
		if obj := evaluate(cand); obj < bestObj {
			bestObj = obj
			best = cand
			t.Logf("random step %d: objective %.2f", i, obj)
		}
	}

	// Coordinate descent: halve and double each coordinate until a full sweep
	// finds nothing.
	for sweep := 0; sweep < 5; sweep++ {
		improved := false
		for j := range best {
			for _, scale := range []float64{0.5, 2.0} {
				cand := append([]float64(nil), best...)
				cand[j] *= scale
				if obj := evaluate(cand); obj < bestObj {
					bestObj = obj
					best = cand
					improved = true
					t.Logf("sweep %d %s x%.1f: objective %.2f", sweep, rels[j], scale, obj)
				}
			}
		}
		if !improved {
			break
		}
	}

	var sum float64
	for _, v := range best {
		sum += v
	}
	t.Logf("EXPERIMENTAL fitted weights after %d evaluations (objective %.2f, baseline re-check below):", evals, bestObj)
	for i, rel := range rels {
		t.Logf("  %-22s %.4f  (default %.4f)", rel, best[i]/sum, defaults[i])
	}

	// Leave every Run rescored under its own vocabulary and log the final
	// per-corpus scorecards under both vectors for the report.
	overrides := make(map[ontology.TermID]float64, len(rels))
	for i, rel := range rels {
		overrides[rel] = best[i] / sum
	}
	for _, lc := range corpora {
		fitted, err := ontology.WithWeightsOver(lc.onto, overrides)
		if err != nil {
			t.Fatal(err)
		}
		lc.run.Rescore(fitted)
		t.Logf("[%s] fitted   %s", lc.name, scLine(Score(lc.run, lc.lf)))
		lc.run.Rescore(lc.onto)
		t.Logf("[%s] default  %s", lc.name, scLine(Score(lc.run, lc.lf)))
	}
}

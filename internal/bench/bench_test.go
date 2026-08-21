package bench

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/retriever"
)

// The loader is exercised in CI with a synthetic fixture; no real corpus
// identifiers exist anywhere in this repository.
func TestLabelsFileWellFormed(t *testing.T) {
	good := `{"corpus":"example","reviewed":"2026-01-01","labels":[
		{"a":"alpha.DoThing","b":"beta.*Widget.DoThing","class":"merge","note":"clone"},
		{"a":"alpha.Helper","b":"alpha.Assist","class":"refactor","note":"extract helper"},
		{"a":"gamma.Render","b":"delta.Paint","class":"false_positive","note":"shared vocabulary only"}]}`
	if _, err := ParseLabels([]byte(good)); err != nil {
		t.Fatalf("valid labels rejected: %v", err)
	}
	withPop := `{"corpus":"example","reviewed":"2026-01-01","population":"exclude","labels":[
		{"a":"alpha.DoThing","b":"beta.DoThing","class":"merge","note":"clone"}]}`
	if lf, err := ParseLabels([]byte(withPop)); err != nil || lf.Population != "exclude" {
		t.Fatalf("population field mishandled: %v (%+v)", err, lf.Population)
	}
	if lf, _ := ParseLabels([]byte(good)); lf.Population != "include" {
		t.Errorf("empty population should default to include, got %q", lf.Population)
	}
	bad := []string{
		`{"corpus":"x","reviewed":"2026-01-01","labels":[{"a":"a.F","b":"b.G","class":"maybe","note":""}]}`,
		`{"corpus":"x","reviewed":"2026-01-01","labels":[{"a":"a.F","b":"b.G","class":"merge","note":""},{"a":"b.G","b":"a.F","class":"merge","note":"dup reversed"}]}`,
		`{"corpus":"x","labels":[]}`,
		`{"corpus":"x","reviewed":"2026-01-01","population":"sometimes","labels":[{"a":"a.F","b":"b.G","class":"merge","note":""}]}`,
	}
	for i, src := range bad {
		if _, err := ParseLabels([]byte(src)); err == nil {
			t.Errorf("bad labels %d accepted", i)
		}
	}
}

// TestGoldenRanking scores a private review: both the corpus and the labels
// file arrive by environment variable, so nothing about them is committed.
func TestGoldenRanking(t *testing.T) {
	corpus := os.Getenv("DOPPEL_BENCH_CORPUS")
	labelsPath := os.Getenv("DOPPEL_BENCH_LABELS")
	if corpus == "" || labelsPath == "" {
		t.Skip("set DOPPEL_BENCH_CORPUS and DOPPEL_BENCH_LABELS to run the golden benchmark")
	}
	data, err := os.ReadFile(labelsPath)
	if err != nil {
		t.Fatalf("read labels: %v", err)
	}
	lf, err := ParseLabels(data)
	if err != nil {
		t.Fatalf("parse labels: %v", err)
	}
	scoreLabels(t, corpus, lf)
}

// TestGoldenCorpora scores the reviews that ARE committed: examples/labels/
// <corpus>.labels.json against the matching rung of the public ladder. Each
// labeled pair names two functions in a pinned public tree, so any reader can
// open them and disagree — which is the only way a golden benchmark on
// somebody else's judgment is worth anything. Corpora that have not been
// fetched are skipped.
func TestGoldenCorpora(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "examples", "labels")
	entries, err := filepath.Glob(filepath.Join(dir, "*.labels.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Skip("no committed labels files")
	}
	for _, path := range entries {
		name := strings.TrimSuffix(filepath.Base(path), ".labels.json")
		t.Run(name, func(t *testing.T) {
			c, ok := Find(name)
			if !ok {
				t.Fatalf("%s: no corpus in the ladder is named %q", filepath.Base(path), name)
			}
			if !Present(c) {
				t.Skipf("%s not fetched; run `task corpora`", name)
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
			scoreLabels(t, corpus, lf)
		})
	}
}

// scoreLabels runs the pipeline over corpus and scores lf against the
// resulting ranking. The scoring itself lives in Score (scorecard.go) so the
// ablation and fitting harness can reuse it; this wrapper only loads, logs
// the scorecard, and turns the three violation lists into hard assertions.
func scoreLabels(t *testing.T, corpus string, lf LabelsFile) {
	// The pipeline's ranking-relevant stages, as a library, at production
	// defaults. Culture/habitat/arena are skipped: they annotate, never rank.
	// The labels declare which population they describe; Load mirrors the
	// pipeline's --tests filter before any corpus statistic is computed, and
	// Analyze drops cross test/prod pairs the way the pipeline does.
	units, err := Load(corpus, Population(lf.Population))
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}
	if len(units) == 0 {
		t.Fatal("corpus yielded no functions")
	}
	t.Logf("corpus %s (population %s): %d functions", lf.Corpus, lf.Population, len(units))

	run := Analyze(units, retriever.DefaultOptions())
	sc := Score(run, lf)
	logScorecard(t, sc)

	// Hard assertions. Failure messages enumerate the offending pairs so a
	// known-limitation pair is identifiable next to a real regression.
	if len(sc.MergeMissing) > 0 {
		t.Errorf("AssertMergeAccounted: merge pairs never retrieved: %v", sc.MergeMissing)
	}
	if len(sc.FPAboveMerge) > 0 {
		t.Errorf("AssertNoFPAboveMerge: %v", sc.FPAboveMerge)
	}
	if len(sc.FPInTop20) > 0 {
		t.Errorf("AssertZeroFPInTop20: %v", sc.FPInTop20)
	}
}

// logScorecard prints one Scorecard the way scoreLabels always has.
func logScorecard(t *testing.T, sc Scorecard) {
	t.Helper()
	t.Logf("ranked %d pairs (%d suppressed by max-per-func=2)", sc.Ranked, sc.Suppressed)
	for _, r := range sc.Results {
		if r.Rank > 0 {
			t.Logf("%-14s rank %-6d key %8.1f  %s / %s  — %s",
				r.Label.Class, r.Rank, r.Key, r.Label.A, r.Label.B, r.Label.Note)
		} else {
			t.Logf("%-14s %-11s          %s / %s  — %s",
				r.Label.Class, r.Absent, r.Label.A, r.Label.B, r.Label.Note)
		}
	}
	for _, class := range []string{"merge", "refactor", "false_positive"} {
		if sc.Present[class] == 0 {
			continue
		}
		t.Logf("mean rank %s: %.1f over %d present", class, sc.MeanRank[class], sc.Present[class])
	}
	t.Logf("aggregates: fp_in_top20=%d merge_present=%d/%d merge_in_top50=%d",
		len(sc.FPInTop20), sc.MergePresent, sc.MergeTotal, sc.MergeInTop50)
}

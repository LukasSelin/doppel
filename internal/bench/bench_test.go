package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/retriever"
)

type label struct {
	A     string `json:"a"`
	B     string `json:"b"`
	Class string `json:"class"`
	Note  string `json:"note"`
}

type labelsFile struct {
	Corpus     string  `json:"corpus"`
	Reviewed   string  `json:"reviewed"`
	Population string  `json:"population"` // include (default when empty) | exclude | only
	Labels     []label `json:"labels"`
}

// pairKey is the canonical unordered identity of a labeled pair.
func pairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "\x00" + b
}

func parseLabels(data []byte) (labelsFile, error) {
	var lf labelsFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return lf, err
	}
	if lf.Corpus == "" || lf.Reviewed == "" || len(lf.Labels) == 0 {
		return lf, fmt.Errorf("labels file needs corpus, reviewed, and at least one label")
	}
	switch lf.Population {
	case "":
		lf.Population = "include"
	case "include", "exclude", "only":
	default:
		return lf, fmt.Errorf("invalid population %q: want include, exclude, or only", lf.Population)
	}
	seen := map[string]bool{}
	for i, l := range lf.Labels {
		switch l.Class {
		case "merge", "refactor", "false_positive":
		default:
			return lf, fmt.Errorf("label %d: invalid class %q", i, l.Class)
		}
		if l.A == "" || l.B == "" {
			return lf, fmt.Errorf("label %d: empty qualified name", i)
		}
		k := pairKey(l.A, l.B)
		if seen[k] {
			return lf, fmt.Errorf("label %d: duplicate pair %s / %s", i, l.A, l.B)
		}
		seen[k] = true
	}
	return lf, nil
}

// The loader is exercised in CI with a synthetic fixture; no real corpus
// identifiers exist anywhere in this repository.
func TestLabelsFileWellFormed(t *testing.T) {
	good := `{"corpus":"example","reviewed":"2026-01-01","labels":[
		{"a":"alpha.DoThing","b":"beta.*Widget.DoThing","class":"merge","note":"clone"},
		{"a":"alpha.Helper","b":"alpha.Assist","class":"refactor","note":"extract helper"},
		{"a":"gamma.Render","b":"delta.Paint","class":"false_positive","note":"shared vocabulary only"}]}`
	if _, err := parseLabels([]byte(good)); err != nil {
		t.Fatalf("valid labels rejected: %v", err)
	}
	withPop := `{"corpus":"example","reviewed":"2026-01-01","population":"exclude","labels":[
		{"a":"alpha.DoThing","b":"beta.DoThing","class":"merge","note":"clone"}]}`
	if lf, err := parseLabels([]byte(withPop)); err != nil || lf.Population != "exclude" {
		t.Fatalf("population field mishandled: %v (%+v)", err, lf.Population)
	}
	if lf, _ := parseLabels([]byte(good)); lf.Population != "include" {
		t.Errorf("empty population should default to include, got %q", lf.Population)
	}
	bad := []string{
		`{"corpus":"x","reviewed":"2026-01-01","labels":[{"a":"a.F","b":"b.G","class":"maybe","note":""}]}`,
		`{"corpus":"x","reviewed":"2026-01-01","labels":[{"a":"a.F","b":"b.G","class":"merge","note":""},{"a":"b.G","b":"a.F","class":"merge","note":"dup reversed"}]}`,
		`{"corpus":"x","labels":[]}`,
		`{"corpus":"x","reviewed":"2026-01-01","population":"sometimes","labels":[{"a":"a.F","b":"b.G","class":"merge","note":""}]}`,
	}
	for i, src := range bad {
		if _, err := parseLabels([]byte(src)); err == nil {
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
	lf, err := parseLabels(data)
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
			lf, err := parseLabels(data)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			scoreLabels(t, corpus, lf)
		})
	}
}

// scoreLabels runs the pipeline over corpus and scores lf against the
// resulting ranking: every labeled pair gets a rank or an absence reason,
// three assertions are hard, and the rest is a logged scorecard.
func scoreLabels(t *testing.T, corpus string, lf labelsFile) {
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
	pairs := run.Pairs

	retrieved := make(map[string]bool, len(pairs))
	for _, p := range pairs {
		retrieved[pairKey(qualifiedName(p.A), qualifiedName(p.B))] = true
	}

	kept, suppressed := analyzer.SortForReport(pairs, 0, 2)
	t.Logf("ranked %d pairs (%d suppressed by max-per-func=2)", len(kept), suppressed)

	// Best (lowest) rank per unordered pair identity — duplicate qualified
	// names (several units sharing one name) resolve to the best rank.
	rankOf := make(map[string]int, len(kept))
	keyOf := make(map[string]float64, len(kept))
	for i, p := range kept {
		k := pairKey(qualifiedName(p.A), qualifiedName(p.B))
		if _, ok := rankOf[k]; !ok {
			rankOf[k] = i + 1
			keyOf[k] = RankKey(p)
		}
	}

	type result struct {
		label  label
		rank   int    // 0 when absent
		absent string // AbsentReason
	}
	var results []result
	classRanks := map[string][]int{}
	fpInTop20, mergePresent, mergeTotal, mergeInTop50 := 0, 0, 0, 0
	var fpTop20Names, fpAboveMergeNames, mergeMissing []string

	for _, l := range lf.Labels {
		k := pairKey(l.A, l.B)
		r := result{label: l}
		if rank, ok := rankOf[k]; ok {
			r.rank = rank
			classRanks[l.Class] = append(classRanks[l.Class], rank)
		} else if retrieved[k] {
			r.absent = "suppressed_by_max_per_func"
		} else {
			r.absent = "not_retrieved"
		}
		results = append(results, r)

		switch l.Class {
		case "merge":
			mergeTotal++
			if r.rank > 0 {
				mergePresent++
				if r.rank <= 50 {
					mergeInTop50++
				}
			} else if r.absent == "not_retrieved" {
				mergeMissing = append(mergeMissing, l.A+" / "+l.B)
			}
		case "false_positive":
			if r.rank > 0 && r.rank <= 20 {
				fpInTop20++
				fpTop20Names = append(fpTop20Names, fmt.Sprintf("%s / %s (rank %d)", l.A, l.B, r.rank))
			}
		}
	}

	// Scorecard.
	for _, r := range results {
		if r.rank > 0 {
			t.Logf("%-14s rank %-6d key %8.1f  %s / %s  — %s",
				r.label.Class, r.rank, keyOf[pairKey(r.label.A, r.label.B)], r.label.A, r.label.B, r.label.Note)
		} else {
			t.Logf("%-14s %-11s          %s / %s  — %s",
				r.label.Class, r.absent, r.label.A, r.label.B, r.label.Note)
		}
	}
	for _, class := range []string{"merge", "refactor", "false_positive"} {
		ranks := classRanks[class]
		if len(ranks) == 0 {
			continue
		}
		sum := 0
		for _, r := range ranks {
			sum += r
		}
		t.Logf("mean rank %s: %.1f over %d present", class, float64(sum)/float64(len(ranks)), len(ranks))
	}
	t.Logf("aggregates: fp_in_top20=%d merge_present=%d/%d merge_in_top50=%d",
		fpInTop20, mergePresent, mergeTotal, mergeInTop50)

	// Hard assertions. Failure messages enumerate the offending pairs so a
	// known-limitation pair is identifiable next to a real regression.
	if len(mergeMissing) > 0 {
		t.Errorf("AssertMergeAccounted: merge pairs never retrieved: %v", mergeMissing)
	}
	worstMerge := 0
	for _, r := range classRanks["merge"] {
		if r > worstMerge {
			worstMerge = r
		}
	}
	sort.Ints(classRanks["false_positive"])
	for _, l := range lf.Labels {
		if l.Class != "false_positive" {
			continue
		}
		if rank, ok := rankOf[pairKey(l.A, l.B)]; ok && worstMerge > 0 && rank < worstMerge {
			fpAboveMergeNames = append(fpAboveMergeNames,
				fmt.Sprintf("%s / %s (rank %d, worst merge %d)", l.A, l.B, rank, worstMerge))
		}
	}
	if len(fpAboveMergeNames) > 0 {
		t.Errorf("AssertNoFPAboveMerge: %v", fpAboveMergeNames)
	}
	if fpInTop20 > 0 {
		t.Errorf("AssertZeroFPInTop20: %v", fpTop20Names)
	}
}

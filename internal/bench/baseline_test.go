package bench

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/retriever"
)

// TestGenerateBaseline is T0's "number to beat": it scores every committed
// labels file against its matching corpus exactly as TestGoldenCorpora does
// — same Load/Analyze/Score path, same population — then hashes every
// example report currently on disk, and writes both into
// examples/baseline.json.
//
// It deliberately does not regenerate examples/ itself: `task examples` is
// its own documented step, and this test only reads whatever is on disk
// afterward. That is what makes two consecutive `task baseline` runs on an
// unchanged tree byte-identical — no rebuild, no re-parse, no wall clock in
// the file this test writes.
//
// A labeled corpus that fails its own golden hard assertions still gets
// recorded (AssertionsPassed: false) rather than aborting the whole
// baseline — a regression is exactly the kind of thing a baseline exists to
// catch — but the test itself fails too, so `task baseline` cannot silently
// launder a broken ranking into "the new normal" without a human noticing.
//
//	task examples   # regenerate examples/*.md first (own step, own task)
//	task baseline   # DOPPEL_BENCH_BASELINE=1 go test ... -run TestGenerateBaseline
func TestGenerateBaseline(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_BASELINE") == "" {
		t.Skip("set DOPPEL_BENCH_BASELINE=1 to (re)generate examples/baseline.json")
	}
	root := repoRoot(t)

	labelsDir := filepath.Join(root, "examples", "labels")
	entries, err := filepath.Glob(filepath.Join(labelsDir, "*.labels.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Skip("no committed labels files")
	}

	var corpora []CorpusBaseline
	for _, path := range entries {
		name := strings.TrimSuffix(filepath.Base(path), ".labels.json")
		c, ok := Find(name)
		if !ok {
			t.Fatalf("%s: no corpus in the ladder is named %q", filepath.Base(path), name)
		}
		if !Present(c) {
			t.Fatalf("%s not fetched; run `task corpora` before `task baseline`", name)
		}
		corpusDir, err := Path(c)
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
		units, err := Load(corpusDir, Population(lf.Population))
		if err != nil {
			t.Fatalf("%s: load corpus: %v", name, err)
		}
		if len(units) == 0 {
			t.Fatalf("%s: corpus yielded no functions", name)
		}
		run := Analyze(units, retriever.DefaultOptions())
		sc := Score(run, lf)
		t.Run(name, func(t *testing.T) { logScorecard(t, sc) })

		cb := corpusBaselineOf(name, lf, sc)
		if !cb.AssertionsPassed {
			t.Errorf("%s: golden hard assertions failed — recording the regression, not hiding it "+
				"(merge missing %v, fp above merge %v, fp in top20 %v)",
				name, sc.MergeMissing, sc.FPAboveMerge, sc.FPInTop20)
		}
		corpora = append(corpora, cb)
	}
	sort.Slice(corpora, func(i, j int) bool { return corpora[i].Corpus < corpora[j].Corpus })

	examplesDir := filepath.Join(root, "examples")
	checks, err := exampleChecksums(examplesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) == 0 {
		t.Fatal("no examples/<corpus>.md found on disk; run `task examples` first")
	}

	out, err := MarshalBaseline(Baseline{Corpora: corpora, Examples: checks})
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(examplesDir, "baseline.json")
	if err := os.WriteFile(dst, out, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s (%d bytes, %d labeled corpora, %d example checksums)", dst, len(out), len(corpora), len(checks))
}

// baselineTimingsCorpora are the fast rungs TestGenerateBaselineTimings times.
// moby/prometheus/hugo's own BenchmarkCorpus subtests alone run
// seconds-to-minutes — that is what `task bench` is for. A baseline
// regeneration should stay on the order of a few seconds, so this list is
// deliberately the four small-to-mid corpora only.
var baselineTimingsCorpora = []string{"cobra", "chi", "conc", "gin"}

// baselineTimingsPayload is examples/baseline-timings.json's shape: the raw
// `go test -bench` output plus enough machine identity to explain why a
// different run reads differently. Every field here can vary run to run or
// machine to machine by design — see the Note — which is exactly why these
// numbers live in their own file instead of inside baseline.json.
type baselineTimingsPayload struct {
	Note      string   `json:"note"`
	GOOS      string   `json:"goos"`
	GOARCH    string   `json:"goarch"`
	NumCPU    int      `json:"numCPU"`
	Corpora   []string `json:"corpora"`
	GoTestCmd string   `json:"goTestCmd"`
	Output    string   `json:"output"`
}

// TestGenerateBaselineTimings captures per-stage pipeline timings the same
// way `task bench` does — literally by invoking BenchmarkCorpus, restricted
// to the fast corpora in baselineTimingsCorpora — and writes them to
// examples/baseline-timings.json.
//
// This file is EXPLICITLY EXCLUDED from baseline.json's determinism
// guarantee: wall-clock timings vary by machine, load, and thermal state, by
// design, which is why they are not in the deterministic file at all.
//
//	task baseline   # also runs this, alongside TestGenerateBaseline
func TestGenerateBaselineTimings(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_BASELINE") == "" {
		t.Skip("set DOPPEL_BENCH_BASELINE=1 to (re)generate examples/baseline-timings.json")
	}
	root := repoRoot(t)

	var present []string
	for _, name := range baselineTimingsCorpora {
		if c, ok := Find(name); ok && Present(c) {
			present = append(present, name)
		}
	}
	if len(present) == 0 {
		t.Skipf("none of %v are fetched; run `task corpora`", baselineTimingsCorpora)
	}

	benchPattern := "BenchmarkCorpus/(" + strings.Join(present, "|") + ")"
	cmd := exec.Command("go", "test", "./internal/bench/", "-run", "^$",
		"-bench", benchPattern, "-benchtime=1x")
	cmd.Dir = root
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run BenchmarkCorpus: %v\n%s", err, stderr.String())
	}

	payload := baselineTimingsPayload{
		Note: "Machine- and load-dependent. NOT covered by baseline.json's determinism guarantee — " +
			"expect different numbers on a different machine, or the same machine under load. " +
			"Restricted to the fast corpora (cobra, chi, conc, gin); `task bench` times the full " +
			"ladder including moby/prometheus/hugo, which cost seconds to minutes each.",
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, NumCPU: runtime.NumCPU(),
		Corpora:   present,
		GoTestCmd: strings.Join(cmd.Args, " "),
		Output:    string(normalizeEOL(out.Bytes())),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')

	dst := filepath.Join(root, "examples", "baseline-timings.json")
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s (%d bytes, corpora %v)", dst, len(data), present)
}

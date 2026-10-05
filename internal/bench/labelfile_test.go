package bench

import (
	"path/filepath"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/parser"
)

// A same-name pair is only a label when both sides say which file they mean,
// and a file is only a file when it is a clean path relative to the root.
func TestLabelFilesValidated(t *testing.T) {
	good := `{"corpus":"x","reviewed":"2026-01-01","labels":[
		{"a":"cmd.init","aFile":"cmd/a.go","b":"cmd.init","bFile":"cmd/b.go","class":"false_positive","kind":"entrypoint","note":"flag registration"},
		{"a":"cmd.init","aFile":"cmd/a.go","b":"cmd.init","bFile":"cmd/c.go","class":"false_positive","note":"same, third file"},
		{"a":"alpha.F","b":"beta.F","bFile":"beta/f.go","class":"merge","note":"one side pinned"}]}`
	lf, err := ParseLabels([]byte(good))
	if err != nil {
		t.Fatalf("valid file-pinned labels rejected: %v", err)
	}
	if got := lf.Labels[0].Pair(); got != "cmd.init@cmd/a.go / cmd.init@cmd/b.go" {
		t.Errorf("Pair() = %q", got)
	}

	label := func(fields string) string {
		return `{"corpus":"x","reviewed":"2026-01-01","labels":[{` + fields + `,"class":"merge","note":""}]}`
	}
	bad := map[string]string{
		"same name, no files":     label(`"a":"cmd.init","b":"cmd.init"`),
		"same name, one file":     label(`"a":"cmd.init","aFile":"cmd/a.go","b":"cmd.init"`),
		"same name, same file":    label(`"a":"cmd.init","aFile":"cmd/a.go","b":"cmd.init","bFile":"cmd/a.go"`),
		"absolute file":           label(`"a":"a.F","aFile":"/abs/a.go","b":"b.G"`),
		"parent-relative file":    label(`"a":"a.F","aFile":"../a.go","b":"b.G"`),
		"unclean file":            label(`"a":"a.F","aFile":"./a.go","b":"b.G"`),
		"backslash file":          label(`"a":"a.F","aFile":"cmd\\a.go","b":"b.G"`),
		"reversed duplicate pair": `{"corpus":"x","reviewed":"2026-01-01","labels":[{"a":"cmd.init","aFile":"cmd/a.go","b":"cmd.init","bFile":"cmd/b.go","class":"merge","note":""},{"a":"cmd.init","aFile":"cmd/b.go","b":"cmd.init","bFile":"cmd/a.go","class":"merge","note":""}]}`,
	}
	for name, src := range bad {
		if _, err := ParseLabels([]byte(src)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// Three init functions in one package, paired every way: a label pinning
// files must find its own pair in either orientation, and a label pinning the
// wrong file must not borrow another pair's rank.
func TestScoreMatchesPinnedFiles(t *testing.T) {
	root := t.TempDir()
	unit := func(pkg, name string, file ...string) parser.CodeUnit {
		return parser.CodeUnit{Package: pkg, Name: name, File: filepath.Join(append([]string{root}, file...)...)}
	}
	units := []parser.CodeUnit{
		unit("cmd", "init", "cmd", "a.go"),
		unit("cmd", "init", "cmd", "b.go"),
		unit("cmd", "init", "cmd", "c.go"),
		unit("alpha", "F", "alpha", "f.go"),
		unit("beta", "F", "beta", "f.go"),
	}
	pair := func(a, b int, total float64) analyzer.SimilarPair {
		return analyzer.SimilarPair{AIdx: a, BIdx: b, Score: 1,
			Retrieval: &analyzer.Retrieval{Total: total, TrophicSim: 1}}
	}
	run := &Run{Units: units, Root: root, Pairs: []analyzer.SimilarPair{
		pair(0, 1, 40), // a/b: rank 1
		pair(3, 4, 30), // alpha.F / beta.F: rank 2
		pair(0, 2, 20), // a/c: rank 3
		pair(1, 2, 10), // b/c: rank 4
	}}
	lf := LabelsFile{Corpus: "x", Reviewed: "2026-01-01", Labels: []Label{
		{A: "cmd.init", AFile: "cmd/a.go", B: "cmd.init", BFile: "cmd/b.go", Class: "false_positive"},
		{A: "cmd.init", AFile: "cmd/c.go", B: "cmd.init", BFile: "cmd/a.go", Class: "false_positive"}, // reversed
		{A: "cmd.init", AFile: "cmd/b.go", B: "cmd.init", BFile: "cmd/c.go", Class: "refactor"},
		{A: "beta.F", B: "alpha.F", Class: "merge"},                             // no files: names suffice
		{A: "alpha.F", AFile: "alpha/other.go", B: "beta.F", Class: "refactor"}, // wrong file
	}}
	want := []int{1, 3, 4, 2, 0}
	sc := Score(run, lf)
	for i, r := range sc.Results {
		if r.Rank != want[i] {
			t.Errorf("%s: rank %d, want %d (absent %q)", r.Label.Pair(), r.Rank, want[i], r.Absent)
		}
	}
	if got := sc.Results[4].Absent; got != AbsentNotRetrieved {
		t.Errorf("wrong-file label: absent %q, want %q", got, AbsentNotRetrieved)
	}

	// With no root, a pinned file matches the recorded path as a whole
	// trailing path, and never as a partial segment.
	run.Root = ""
	lf.Labels = []Label{
		{A: "cmd.init", AFile: "cmd/a.go", B: "cmd.init", BFile: "cmd/b.go", Class: "merge"},
		{A: "cmd.init", AFile: "md/a.go", B: "cmd.init", BFile: "cmd/b.go", Class: "merge"},
	}
	sc = Score(run, lf)
	if sc.Results[0].Rank != 1 {
		t.Errorf("rootless trailing-path match: rank %d, want 1", sc.Results[0].Rank)
	}
	if sc.Results[1].Rank != 0 {
		t.Errorf("partial segment md/a.go matched cmd/a.go at rank %d", sc.Results[1].Rank)
	}
}

// TestCoupledIsScoredNotAsserted pins what the coupled class is for: a pair
// history says is kept in step gets a rank and a mean like any class, and —
// because it claims nothing about merging — never enters the merge or
// false-positive accounting the hard assertions read. A coupled pair at rank
// 1 above an unretrieved merge would otherwise read as a violation.
func TestCoupledIsScoredNotAsserted(t *testing.T) {
	lf, err := ParseLabels([]byte(`{"corpus":"x","reviewed":"2026-01-01","labels":[
		{"a":"alpha.Create","b":"alpha.Delete","class":"coupled","note":"co-changed twice"}]}`))
	if err != nil {
		t.Fatalf("coupled label rejected: %v", err)
	}
	units := []parser.CodeUnit{
		{Package: "alpha", Name: "Create", File: "alpha/a.go"},
		{Package: "alpha", Name: "Delete", File: "alpha/a.go"},
	}
	run := &Run{Units: units, Pairs: []analyzer.SimilarPair{{AIdx: 0, BIdx: 1, Score: 1,
		Retrieval: &analyzer.Retrieval{Total: 10, TrophicSim: 1}}}}
	sc := Score(run, lf)
	if sc.Results[0].Rank != 1 || sc.Present["coupled"] != 1 || sc.MeanRank["coupled"] != 1 {
		t.Errorf("coupled not scored: rank %d, present %d, mean %.1f",
			sc.Results[0].Rank, sc.Present["coupled"], sc.MeanRank["coupled"])
	}
	if sc.MergeTotal != 0 || len(sc.FPInTop20) != 0 || len(sc.FPAboveMerge) != 0 || len(sc.MergeMissing) != 0 {
		t.Errorf("coupled leaked into the asserted accounting: %+v", sc)
	}
}

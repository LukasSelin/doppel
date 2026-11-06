package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/labels"
)

func TestVerdictClassFor(t *testing.T) {
	cases := []struct {
		outcome, class, kind, blocker string
		want                          string
		wantErr                       bool
	}{
		{"merged", "", "", "", "merge", false},
		{"merged", "refactor", "", "", "refactor", false},
		{"deferred", "", "", "", "merge", false},
		{"left", "", "mirror", "", "false_positive", false},
		{"left", "", "", "layering", "merge", false},
		{"left", "refactor", "", "design", "refactor", false},
		{"left", "", "", "", "", true},
		{"left", "", "mirror", "design", "", true},
	}
	for _, c := range cases {
		got, err := verdictClassFor(c.outcome, c.class, c.kind, c.blocker)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("%s class=%q kind=%q blocker=%q: got %q, %v; want %q (error %v)",
				c.outcome, c.class, c.kind, c.blocker, got, err, c.want, c.wantErr)
		}
	}
}

func TestSplitSide(t *testing.T) {
	name, file, err := splitSide("cmd.init@cmd/diff.go")
	if err != nil || name != "cmd.init" || file != "cmd/diff.go" {
		t.Errorf("hook key: %q %q %v", name, file, err)
	}
	name, file, err = splitSide("reporter.*Comparator.Compare")
	if err != nil || name != "reporter.*Comparator.Compare" || file != "" {
		t.Errorf("bare name: %q %q %v", name, file, err)
	}
	if _, _, err := splitSide("config.init@config/loader.go#2"); err == nil {
		t.Error("an ordinal key was accepted; a label cannot name the second init in a file")
	}
	if _, _, err := splitSide("@cmd/x.go"); err == nil {
		t.Error("a side with no name was accepted")
	}
}

// TestUpsertVerdictKeepsHistory pins that recording a pair again replaces its
// verdict without erasing the decision before it, keeps a reviewer's note,
// and drops a stale explicit weight so the new decision decides it.
func TestUpsertVerdictKeepsHistory(t *testing.T) {
	first := labels.Decision{Outcome: "left", By: "agent", Date: "2026-10-10", Reason: "shares only setup", Commit: "abc1234"}
	f := labels.File{Labels: []labels.Label{
		{A: "main.run", B: "main.runStudy", Class: "merge", Blocker: "too-small", Note: "reviewer note", Weight: 0.5, Decision: &first},
	}}
	again := labels.Label{A: "main.runStudy", B: "main.run", Class: "refactor",
		Decision: &labels.Decision{Outcome: "merged", By: "agent", Date: "2026-10-11", Reason: "extracted the shared lookup"}}

	prev := upsertVerdict(&f, again)
	if prev == nil || prev.Outcome != "left" {
		t.Fatalf("replaced decision not returned: %+v", prev)
	}
	if len(f.Labels) != 1 {
		t.Fatalf("the reversed pair was appended instead of replaced: %d labels", len(f.Labels))
	}
	got := f.Labels[0]
	if got.Decision.Outcome != "merged" || len(got.Earlier) != 1 || got.Earlier[0] != first {
		t.Errorf("history lost: decision %+v, earlier %+v", got.Decision, got.Earlier)
	}
	if got.Note != "reviewer note" || got.Weight != 0 || got.Blocker != "" {
		t.Errorf("note kept, weight and blocker dropped: note %q weight %g blocker %q", got.Note, got.Weight, got.Blocker)
	}

	upsertVerdict(&f, labels.Label{A: "a.X", B: "a.Y", Class: "merge",
		Decision: &labels.Decision{Outcome: "deferred", By: "user", Date: "2026-10-12", Reason: "later"}})
	if len(f.Labels) != 2 {
		t.Errorf("a new pair was not appended: %d labels", len(f.Labels))
	}
}

// TestRunVerdictWritesALabelsFile records two decisions into a fresh root and
// checks the result is a labels file the benchmark would accept, then that a
// file which does not parse is refused rather than overwritten.
func TestRunVerdictWritesALabelsFile(t *testing.T) {
	root := t.TempDir()
	set := func(reason, kind, blocker string) {
		verdictReason, verdictKind, verdictBlocker = reason, kind, blocker
		verdictClass, verdictTarget, verdictBy, verdictCommit, verdictRoot, verdictFile = "", "", "agent", "", root, ""
	}
	t.Cleanup(func() { set("", "", ""); verdictRoot = "." })

	var out bytes.Buffer
	verdictCmd.SetOut(&out)
	t.Cleanup(func() { verdictCmd.SetOut(nil) })

	set("one helper now", "", "")
	if err := runVerdict(verdictCmd, []string{"merged", "cmd.displayRoot", "cmd.workdirName"}); err != nil {
		t.Fatal(err)
	}
	set("each command registers its own flags", "entrypoint", "")
	if err := runVerdict(verdictCmd, []string{"left", "cmd.init@cmd/diff.go", "cmd.init@cmd/moves.go"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "recorded left cmd.init@cmd/diff.go / cmd.init@cmd/moves.go as false_positive") {
		t.Errorf("confirmation line missing: %q", out.String())
	}

	path := filepath.Join(root, filepath.FromSlash(verdictPath))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := labels.Parse(data)
	if err != nil {
		t.Fatalf("written verdicts do not parse as labels: %v\n%s", err, data)
	}
	if f.Corpus != filepath.Base(root) || f.Population != "exclude" || len(f.Labels) != 2 {
		t.Errorf("header or labels wrong: %+v", f)
	}
	if w := f.Labels[0].EffectiveWeight(); w != labels.MergedWeight {
		t.Errorf("a merged verdict weighs %g, want %g", w, float64(labels.MergedWeight))
	}

	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	set("x", "", "")
	if err := runVerdict(verdictCmd, []string{"merged", "a.X", "a.Y"}); err == nil {
		t.Fatal("a broken verdicts file was overwritten")
	}
	if data, _ := os.ReadFile(path); string(data) != "{not json" {
		t.Errorf("a broken verdicts file was modified: %q", data)
	}
}

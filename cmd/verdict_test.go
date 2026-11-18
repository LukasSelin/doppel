package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

// TestVerdictsAreSharedAcrossWorktrees pins where the default file lives: a
// verdict recorded in a linked worktree lands in the main checkout's
// .doppel/verdicts.json, the same file the main checkout writes, at the same
// subdirectory when --root is one, and --file still names any other.
func TestVerdictsAreSharedAcrossWorktrees(t *testing.T) {
	r := newGitRepo(t)
	r.commit("a")
	wt := filepath.Join(t.TempDir(), "session")
	r.run(nil, "worktree", "add", "-q", "--detach", wt)

	sameDir := func(what, got, want string) {
		t.Helper()
		gi, err := os.Stat(got)
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		wi, err := os.Stat(want)
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if !os.SameFile(gi, wi) {
			t.Errorf("%s: home %s, want %s", what, got, want)
		}
	}
	sameDir("linked worktree", verdictHome(wt), r.dir)
	sameDir("main checkout", verdictHome(r.dir), r.dir)

	sub := filepath.Join(wt, "corpus", "inner")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := verdictHome(sub), filepath.Join(verdictHome(wt), "corpus", "inner"); got != want {
		t.Errorf("subdirectory root: home %s, want %s", got, want)
	}
	outside := t.TempDir()
	if got := verdictHome(outside); got != outside {
		t.Errorf("outside a repository: home %s, want the root itself", got)
	}

	set := func(root, file string) {
		verdictReason, verdictKind, verdictBlocker = "one helper now", "", ""
		verdictClass, verdictTarget, verdictBy, verdictCommit, verdictRoot, verdictFile = "", "", "agent", "", root, file
	}
	t.Cleanup(func() { set(".", ""); verdictReason = "" })
	verdictCmd.SetOut(io.Discard)
	t.Cleanup(func() { verdictCmd.SetOut(nil) })

	set(wt, "")
	if err := runVerdict(verdictCmd, []string{"merged", "a.X", "a.Y"}); err != nil {
		t.Fatal(err)
	}
	set(r.dir, "")
	if err := runVerdict(verdictCmd, []string{"merged", "b.X", "b.Y"}); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "elsewhere.json")
	set(wt, other)
	if err := runVerdict(verdictCmd, []string{"merged", "c.X", "c.Y"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(wt, filepath.FromSlash(verdictPath))); err == nil {
		t.Error("the linked worktree got a verdicts file of its own")
	}
	data, err := os.ReadFile(filepath.Join(r.dir, filepath.FromSlash(verdictPath)))
	if err != nil {
		t.Fatalf("the main checkout has no verdicts file: %v", err)
	}
	f, err := labels.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Labels) != 2 || f.Labels[0].A != "a.X" || f.Labels[1].A != "b.X" {
		t.Errorf("want the worktree's and the main checkout's verdicts in one file, got %+v", f.Labels)
	}
	if f.Corpus != filepath.Base(r.dir) {
		t.Errorf("corpus named %q, want the main checkout's %q", f.Corpus, filepath.Base(r.dir))
	}
	if f.Labels[0].Decision.Commit != gitHead(wt) {
		t.Errorf("decision commit %q, want the worktree's HEAD %q", f.Labels[0].Decision.Commit, gitHead(wt))
	}
	if data, err := os.ReadFile(other); err != nil || !strings.Contains(string(data), "c.X") {
		t.Errorf("--file was not honoured: %v %s", err, data)
	}
}

// TestConcurrentVerdictsKeepEveryLabel records many verdicts into one file at
// once and checks none is lost: each writer re-reads under the lock, so the
// last rename carries every earlier label.
func TestConcurrentVerdictsKeepEveryLabel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, filepath.FromSlash(verdictPath))
	const n = 16
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l := labels.Label{A: fmt.Sprintf("p.A%d", i), B: fmt.Sprintf("p.B%d", i), Class: "merge",
				Decision: &labels.Decision{Outcome: "deferred", By: "agent", Date: "2026-11-18", Reason: "later"}}
			_, errs[i] = recordVerdict(path, dir, l)
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("writer %d: %v", i, err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := labels.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Labels) != n {
		t.Errorf("%d concurrent verdicts left %d labels", n, len(f.Labels))
	}
	if _, err := os.Stat(path + ".lock"); err == nil {
		t.Error("the lock was left behind")
	}
}

// TestLockVerdictsBreaksAStaleLock pins that a lock left by a writer that
// died is taken over rather than blocking every later verdict.
func TestLockVerdictsBreaksAStaleLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "verdicts.json")
	if err := os.WriteFile(path+".lock", nil, 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * verdictLockStale)
	if err := os.Chtimes(path+".lock", old, old); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	unlock, err := lockVerdicts(path)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	if d := time.Since(start); d > verdictLockWait/2 {
		t.Errorf("a stale lock was waited on for %s", d)
	}
}

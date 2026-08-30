package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/identity"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// writeSnapshot serialises a snapshot to the exact bytes `doppel analyze
// --format json` writes, so the command under test reads a real file rather
// than a struct handed to it directly.
func writeSnapshot(t *testing.T, dir, name string, s snapshot.Snapshot) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// corpusFrom runs the real pipeline over a temp tree and snapshots it, which
// is what makes this an end-to-end test: the bags, the label dictionary and
// the digests are all produced by the pipeline, not by the test.
func corpusFrom(t *testing.T, files map[string]string) snapshot.Snapshot {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	p := Params{Threshold: 0.60, MinNodes: 12, ChannelK: 5, TestsMode: "exclude", Generated: "exclude"}
	res, err := analyze(dir, p, discardWriter{})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	return snapshotOf(res, res.Pairs)
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

const diffSrcBefore = `package svc

import (
	"fmt"
	"strconv"
	"strings"
)

func Total(lines []string) int {
	total := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		n, err := strconv.Atoi(trimmed)
		if err != nil {
			continue
		}
		total += n
	}
	return total
}

func Describe(name string, n int) string {
	if n == 0 {
		return fmt.Sprintf("%s: empty", name)
	}
	if n < 0 {
		return fmt.Sprintf("%s: negative (%d)", name, n)
	}
	return fmt.Sprintf("%s: %d entries", name, n)
}
`

// diffSrcAfter renames Total to Sum in the same package, leaving Describe
// untouched — one renamed finding and one unchanged one.
var diffSrcAfter = strings.Replace(diffSrcBefore, "func Total(", "func Sum(", 1)

// diffSrcSession is the gate fixture's session: one function renamed, and one
// copied verbatim under a new name. The copy is what creates a pair, and the
// pair's stored explanation is what the delta report has to carry.
var diffSrcSession = diffSrcAfter + `
func Report(name string, n int) string {
	if n == 0 {
		return fmt.Sprintf("%s: empty", name)
	}
	if n < 0 {
		return fmt.Sprintf("%s: negative (%d)", name, n)
	}
	return fmt.Sprintf("%s: %d entries", name, n)
}
`

func runDiffCmd(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	// Flags are package-level and cobra does not reset them between runs.
	diffFormat, diffUnchanged = "text", false
	code := exitDiffOK
	prev := diffExit
	diffExit = func(c int) { code = c }
	t.Cleanup(func() { diffExit = prev })

	// Drive the root command, not diffCmd: cobra routes a child's Execute
	// through its parent anyway, so calling it on the child prints the root's
	// usage instead of running the subcommand.
	var out, errb bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errb)
	rootCmd.SetArgs(append([]string{"diff"}, args...))
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	_ = rootCmd.Execute()
	return out.String(), errb.String(), code
}

// TestDiffCommandEndToEnd is the command's own gate: two snapshots written by
// the real pipeline, read back off disk, and rendered.
func TestDiffCommandEndToEnd(t *testing.T) {
	dir := t.TempDir()
	before := writeSnapshot(t, dir, "before.json", corpusFrom(t, map[string]string{"svc/svc.go": diffSrcBefore}))
	after := writeSnapshot(t, dir, "after.json", corpusFrom(t, map[string]string{"svc/svc.go": diffSrcAfter}))

	out, errOut, code := runDiffCmd(t, before, after)
	if code != exitDiffOK {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "renamed 1") {
		t.Errorf("expected one rename:\n%s", out)
	}
	if !strings.Contains(out, "svc.Total") || !strings.Contains(out, "svc.Sum") {
		t.Errorf("the rename line must name both sides:\n%s", out)
	}
	if !strings.Contains(out, "unchanged 1") {
		t.Errorf("Describe was untouched:\n%s", out)
	}
	// Evidence, not just a verdict.
	if !strings.Contains(out, "jaccard") || !strings.Contains(out, "containment") || !strings.Contains(out, "digests") {
		t.Errorf("every line must print its evidence:\n%s", out)
	}
}

func TestDiffCommandJSON(t *testing.T) {
	dir := t.TempDir()
	before := writeSnapshot(t, dir, "before.json", corpusFrom(t, map[string]string{"svc/svc.go": diffSrcBefore}))
	after := writeSnapshot(t, dir, "after.json", corpusFrom(t, map[string]string{"svc/svc.go": diffSrcAfter}))

	out, errOut, code := runDiffCmd(t, before, after, "--format", "json")
	if code != exitDiffOK {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	var r identity.Result
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("--format json did not emit valid JSON: %v\n%s", err, out)
	}
	if !r.Comparable || r.Count(identity.Renamed) != 1 {
		t.Errorf("want one rename in the JSON payload, got %+v", r.Counts)
	}
	// Every class is always present, so a consumer sees a stable shape.
	if len(r.Counts) != 8 {
		t.Errorf("want all eight class counts, got %d", len(r.Counts))
	}
	// The JSON keeps the unchanged findings the text report only counts.
	if len(r.Changes) != r.OldFunctions {
		t.Errorf("JSON must carry every change including unchanged: %d changes, %d functions", len(r.Changes), r.OldFunctions)
	}

	// Rerunning must produce the same bytes.
	again, _, _ := runDiffCmd(t, before, after, "--format", "json")
	if again != out {
		t.Error("--format json is not byte-identical across runs")
	}
}

func TestDiffCommandUnreadableExitsOne(t *testing.T) {
	dir := t.TempDir()
	good := writeSnapshot(t, dir, "good.json", corpusFrom(t, map[string]string{"svc/svc.go": diffSrcBefore}))

	_, errOut, code := runDiffCmd(t, filepath.Join(dir, "missing.json"), good)
	if code != exitDiffUnreadable {
		t.Errorf("a missing file must exit %d, got %d", exitDiffUnreadable, code)
	}
	if !strings.Contains(errOut, "read ") {
		t.Errorf("stderr should say what could not be read: %q", errOut)
	}

	notJSON := filepath.Join(dir, "junk.json")
	if err := os.WriteFile(notJSON, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, errOut, code = runDiffCmd(t, notJSON, good)
	if code != exitDiffUnreadable {
		t.Errorf("malformed JSON must exit %d, got %d", exitDiffUnreadable, code)
	}
	if !strings.Contains(errOut, "parse ") {
		t.Errorf("stderr should say the file could not be parsed: %q", errOut)
	}
}

func TestDiffCommandIncomparableExitsTwo(t *testing.T) {
	dir := t.TempDir()
	s := corpusFrom(t, map[string]string{"svc/svc.go": diffSrcBefore})
	before := writeSnapshot(t, dir, "before.json", s)
	s.RuleSet += "-next"
	after := writeSnapshot(t, dir, "after.json", s)

	_, errOut, code := runDiffCmd(t, before, after)
	if code != exitDiffIncomparabl {
		t.Errorf("a rule-set mismatch must exit %d, got %d", exitDiffIncomparabl, code)
	}
	if !strings.Contains(errOut, "not comparable") || !strings.Contains(errOut, "rule set") {
		t.Errorf("stderr must say why: %q", errOut)
	}
}

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// A corpus with one true copy (SumPositive and TotalAbove) and one pair of
// plain functions that are an operation and its inverse (encodeHeader and
// decodeHeader), plus enough else that the labels they share are informative.
var listFixture = map[string]string{
	"go.mod": "module example.com/fx\n\ngo 1.25\n",
	"codec/codec.go": `package codec

import (
	"errors"
	"strings"
)

type header struct {
	name  string
	value string
	sent  bool
}

func encodeHeader(h *header, sep string) (string, error) {
	if h == nil {
		return "", errors.New("no header")
	}
	if h.name == "" {
		return "", errors.New("unnamed header")
	}
	out := strings.ToLower(h.name) + sep + strings.TrimSpace(h.value)
	h.sent = true
	return out, nil
}

func decodeHeader(h *header, sep string) (string, error) {
	if h == nil {
		return "", errors.New("no header")
	}
	if h.value == "" {
		return "", errors.New("empty header")
	}
	out := strings.ToUpper(h.value) + sep + strings.TrimSpace(h.name)
	h.sent = false
	return out, nil
}

func splitPair(s, sep string) (string, string) {
	i := strings.Index(s, sep)
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+len(sep):]
}
`,
	"stats/stats.go": `package stats

import "sort"

func SumPositive(nums []int) int {
	total := 0
	for _, n := range nums {
		if n > 0 {
			total += n
		}
	}
	return total
}

func TotalAbove(vals []int) int {
	acc := 0
	for _, v := range vals {
		if v > 0 {
			acc += v
		}
	}
	return acc
}

func Median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid]
	}
	return (s[mid-1] + s[mid]) / 2
}

func Clamp(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func Describe(xs []float64) map[string]float64 {
	out := map[string]float64{}
	if len(xs) == 0 {
		return out
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	out["mean"] = sum / float64(len(xs))
	out["median"] = Median(xs)
	return out
}
`,
}

func listFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range listFixture {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return dir
}

// resetAnalyzeRun restores every analyze flag to its registered default,
// for a test that executes the command. The command is a package-level
// singleton and cobra keeps a flag's value between runs, so a run must not
// leak into the next. A slice flag is emptied through its own Replace: its
// DefValue is the display string "[]", which Set would read as one element
// — a language nobody registered.
func resetAnalyzeRun(t *testing.T) {
	t.Helper()
	for _, name := range append([]string{"config"}, analyzeFlagNames...) {
		f := analyzeCmd.Flags().Lookup(name)
		if f == nil {
			t.Fatalf("flag %q is not registered", name)
		}
		if s, ok := f.Value.(interface{ Replace([]string) error }); ok {
			if err := s.Replace(nil); err != nil {
				t.Fatalf("reset flag %q: %v", name, err)
			}
		} else if err := f.Value.Set(f.DefValue); err != nil {
			t.Fatalf("reset flag %q: %v", name, err)
		}
		f.Changed = false
	}
}

func runAnalyzeCmd(t *testing.T, args ...string) (stdout, stderr string) {
	t.Helper()
	resetAnalyzeRun(t)
	var out, errb bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errb)
	rootCmd.SetArgs(append([]string{"analyze"}, args...))
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
		resetAnalyzeRun(t)
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("analyze %v: %v\n%s", args, err, errb.String())
	}
	return out.String(), errb.String()
}

// The text and markdown lists leave out a pair whose kind says it is not a
// merge, say so, and give its place to the next finding; include and only
// are the other two cuts; and --format json carries every pair whatever the
// mode.
func TestAnalyzeListLeavesOutNonMerges(t *testing.T) {
	dir := listFixtureDir(t)
	const (
		mirrorA = "codec.encodeHeader"
		cloneA  = "stats.SumPositive"
		said    = "Not listed: 1 pair that would have ranked here (mirror operations 1), which doppel reads as something other than a merge. --non-merges include lists every pair."
	)

	// Default: the copy is #1, the mirror is absent, and the report says so.
	out, _ := runAnalyzeCmd(t, dir, "--moves=false")
	if !strings.Contains(out, said) {
		t.Errorf("default report does not say what it left out:\n%s", out)
	}
	if strings.Contains(out, mirrorA) || !strings.Contains(out, cloneA) {
		t.Errorf("default report should list the copy and not the mirror:\n%s", out)
	}
	if !strings.Contains(out, "#1    code-shape: 1.0000") || strings.Contains(out, "#2 ") {
		t.Errorf("default report should hold exactly the copy, at #1:\n%s", out)
	}

	// include: both, the mirror named for what it is, and nothing to say.
	out, _ = runAnalyzeCmd(t, dir, "--moves=false", "--non-merges", "include")
	if strings.Contains(out, "Not listed") || !strings.Contains(out, mirrorA) || !strings.Contains(out, cloneA) {
		t.Errorf("include should list both pairs and say nothing:\n%s", out)
	}
	if !strings.Contains(out, "kind: mirror operations — encodeHeader and decodeHeader in package codec are one operation run in opposite directions") {
		t.Errorf("include should label the mirror:\n%s", out)
	}

	// only: the complement, and it says that too.
	out, _ = runAnalyzeCmd(t, dir, "--moves=false", "--non-merges", "only")
	if !strings.Contains(out, "Listing only the 1 of 2 scored pairs doppel reads as something other than a merge. --non-merges exclude lists the other 1.") {
		t.Errorf("only does not say what it lists:\n%s", out)
	}
	if !strings.Contains(out, mirrorA) || strings.Contains(out, cloneA) {
		t.Errorf("only should list the mirror alone:\n%s", out)
	}

	// Markdown is cut like text.
	md := filepath.Join(t.TempDir(), "report.md")
	runAnalyzeCmd(t, dir, "--moves=false", "-o", md)
	doc, err := os.ReadFile(md)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(doc), "_Not listed: 1 pair that would have ranked here (mirror operations 1), which doppel reads as something other than a merge. `--non-merges include` lists every pair._") {
		t.Errorf("markdown does not say what it left out:\n%s", doc)
	}
	if strings.Contains(string(doc), "encodeHeader") || !strings.Contains(string(doc), "**Pairs found:** 1") {
		t.Errorf("markdown should list the copy alone:\n%s", doc)
	}

	// JSON is the uncut ranking under every mode, kind attached.
	var want string
	for _, mode := range []string{"exclude", "include", "only"} {
		js, _ := runAnalyzeCmd(t, dir, "--moves=false", "--format", "json", "--non-merges", mode)
		if want == "" {
			want = js
		}
		if js != want {
			t.Errorf("--format json differs under --non-merges %s", mode)
		}
		var s snapshot.Snapshot
		if err := json.Unmarshal([]byte(js), &s); err != nil {
			t.Fatalf("json under %s: %v", mode, err)
		}
		kinds := map[string]bool{}
		for _, p := range s.Pairs {
			kinds[p.Kind] = true
			if p.NamedShape < 0 {
				t.Errorf("json pair %s <-> %s carries no named-shape", p.A, p.B)
			}
		}
		if len(s.Pairs) != 2 || !kinds[analyzer.KindMirror] {
			t.Errorf("json under %s should carry both pairs, the mirror labelled: %+v", mode, s.Pairs)
		}
	}
}

func TestAnalyzeRejectsBadNonMerges(t *testing.T) {
	resetAnalyzeRun(t)
	var out, errb bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errb)
	rootCmd.SetArgs([]string{"analyze", t.TempDir(), "--non-merges", "sometimes"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
		resetAnalyzeRun(t)
	})
	if err := rootCmd.Execute(); err == nil || !strings.Contains(err.Error(), "--non-merges") {
		t.Errorf("want an error naming --non-merges, got %v", err)
	}
}

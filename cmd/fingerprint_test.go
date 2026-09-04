package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
)

const fingerprintCmdSrc = `package svc

import "fmt"

func Sum(xs []int) int {
	total := 0
	for _, x := range xs {
		if x > 0 {
			total += x
		}
	}
	return total
}

func Total(vals []int) int {
	acc := 0
	for _, v := range vals {
		if v > 0 {
			acc += v
		}
	}
	return acc
}

func Other(s string) error {
	if s == "" {
		return fmt.Errorf("empty")
	}
	return nil
}

type Box struct{ n int }

func (b *Box) Get() int { return b.n }
func (b *Box) Set(n int) { b.n = n }
`

const fingerprintCmdSrcTwo = `package other

func Get() int { return 2 }
`

func fingerprintCorpusDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range map[string]string{"svc/svc.go": fingerprintCmdSrc, "other/other.go": fingerprintCmdSrcTwo} {
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

func runFingerprintCmd(t *testing.T, args ...string) (stdout string, err error) {
	t.Helper()
	// Flags are package-level and cobra does not reset them between runs.
	fpLabels, fpLabel, fpTests, fpGenerated, fpLanguages, fpConfig = 20, nil, "exclude", "exclude", nil, ""
	var out, errb bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errb)
	rootCmd.SetArgs(append([]string{"fingerprint"}, args...))
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	err = rootCmd.Execute()
	return out.String(), err
}

func TestFingerprintCommandSingle(t *testing.T) {
	dir := fingerprintCorpusDir(t)
	out, err := runFingerprintCmd(t, dir, "svc.Other", "--labels", "0")
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}
	for _, want := range []string{"fingerprint: svc.Other", "svc/svc.go:", "flow: if 1  return 2", "types: in:string out:error", "weights ln(N/df)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestFingerprintCommandPair(t *testing.T) {
	dir := fingerprintCorpusDir(t)
	out, err := runFingerprintCmd(t, dir, "Sum", "Total")
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}
	for _, want := range []string{"A: svc.Sum", "B: svc.Total", "code-shape: 1.0000", "jaccard = shared / union = 1.0000", "only in A: 0 labels"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestFingerprintCommandMethodNameKeepsTheStar(t *testing.T) {
	dir := fingerprintCorpusDir(t)
	out, err := runFingerprintCmd(t, dir, "svc.*Box.Set")
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}
	if !strings.Contains(out, "fingerprint: svc.*Box.Set") {
		t.Errorf("method not found by its qualified starred name:\n%s", out)
	}
}

func TestFingerprintCommandAmbiguousAndMissing(t *testing.T) {
	dir := fingerprintCorpusDir(t)
	// Get is a method in svc and a function in other; a bare name must not
	// pick one silently.
	if _, err := runFingerprintCmd(t, dir, "Get"); err == nil {
		t.Fatal("ambiguous bare name resolved without complaint")
	} else if s := err.Error(); !strings.Contains(s, "ambiguous") || !strings.Contains(s, "other.Get") || !strings.Contains(s, "svc.*Box.Get") {
		t.Errorf("ambiguity error should list both matches, got: %v", err)
	}
	if _, err := runFingerprintCmd(t, dir, "Nope"); err == nil || !strings.Contains(err.Error(), "no function named") {
		t.Errorf("missing name should error, got: %v", err)
	}
}

func TestFindUnitPrefersExactQualifiedMatch(t *testing.T) {
	units, err := parser.ParseSource("svc.go", []byte(fingerprintCmdSrc))
	if err != nil || len(units) == 0 {
		t.Fatalf("parse: units=%d err=%v", len(units), err)
	}
	i, err := findUnit(units, "svc.Sum")
	if err != nil || units[i].Name != "Sum" {
		t.Fatalf("findUnit(svc.Sum) = %d, %v", i, err)
	}
	i, err = findUnit(units, "Other")
	if err != nil || units[i].Name != "Other" {
		t.Fatalf("findUnit(Other) = %d, %v", i, err)
	}
}

// labelFromView pulls the hash (without its #) off the first bag row whose
// label reads as want — "depth-3 RANGE" — the way a reader would copy it.
func labelFromView(t *testing.T, out, want string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if i := strings.Index(line, "  #"); i >= 0 && strings.Contains(line, want) {
			return strings.TrimSpace(line[i+3:])
		}
	}
	t.Fatalf("no %s row in:\n%s", want, out)
	return ""
}

func TestFingerprintCommandLabelSingle(t *testing.T) {
	dir := fingerprintCorpusDir(t)
	view, err := runFingerprintCmd(t, dir, "svc.Sum", "--labels", "0")
	if err != nil {
		t.Fatalf("view: %v", err)
	}
	hash := labelFromView(t, view, "depth-3 RANGE")
	out, err := runFingerprintCmd(t, dir, "svc.Sum", "--labels", "1", "--label", hash)
	if err != nil {
		t.Fatalf("--label: %v", err)
	}
	for _, want := range []string{
		"labels in svc.Sum (code shown is the canonical form",
		"label #" + hash + ": depth-3 RANGE",
		": range, subtree ",
		"for _, x",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// Only the label section: the view above it legitimately says "N total".
	if sec := out[strings.Index(out, "labels in"):]; strings.Contains(sec, "total") {
		t.Errorf("the render must be the canonical form, not the source's names:\n%s", sec)
	}
}

func TestFingerprintCommandLabelPair(t *testing.T) {
	dir := fingerprintCorpusDir(t)
	view, err := runFingerprintCmd(t, dir, "Sum", "Total", "--labels", "0")
	if err != nil {
		t.Fatalf("view: %v", err)
	}
	hash := labelFromView(t, view, "depth-2 IF")
	out, err := runFingerprintCmd(t, dir, "Sum", "Total", "--label", "#"+hash)
	if err != nil {
		t.Fatalf("--label: %v", err)
	}
	for _, want := range []string{"labels in A svc.Sum and B svc.Total", "A ×1  B ×1", "    A svc.Sum\n", "    B svc.Total\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestFingerprintCommandLabelBadHex(t *testing.T) {
	dir := fingerprintCorpusDir(t)
	if _, err := runFingerprintCmd(t, dir, "svc.Sum", "--label", "zz"); err == nil || !strings.Contains(err.Error(), "not a label hash") {
		t.Errorf("bad hex should error, got: %v", err)
	}
}

func TestParseLabelFlagsDedupesInOrder(t *testing.T) {
	got, err := parseLabelFlags([]string{"#ff", "10", "ff", " 0a "})
	if err != nil {
		t.Fatal(err)
	}
	want := []uint64{0xff, 0x10, 0x0a}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// TestLabelSourceForRefusesAChangedFile: the re-derived tree must carry the
// bag the corpus holds, or the lookup would name nodes not in it.
func TestLabelSourceForRefusesAChangedFile(t *testing.T) {
	units, err := parser.ParseSource("svc.go", []byte(fingerprintCmdSrc))
	if err != nil || len(units) == 0 {
		t.Fatalf("parse: units=%d err=%v", len(units), err)
	}
	u := units[0] // Sum
	dir := t.TempDir()
	path := filepath.Join(dir, "svc.go")
	changed := strings.Replace(fingerprintCmdSrc, "total += x", "total += x * 2", 1)
	if err := os.WriteFile(path, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	u.File = path
	if _, err := labelSourceFor(u); err == nil || !strings.Contains(err.Error(), "different label bag") {
		t.Errorf("a changed body must be refused, got: %v", err)
	}
	if err := os.WriteFile(path, []byte(fingerprintCmdSrc), 0o600); err != nil {
		t.Fatal(err)
	}
	src, err := labelSourceFor(u)
	if err != nil {
		t.Fatalf("unchanged file: %v", err)
	}
	if src.Tree == nil || len(src.Renders) == 0 {
		t.Fatal("unchanged file must yield a tree and renders")
	}
	u.Lang = "python"
	src, err = labelSourceFor(u)
	if err != nil || src.Renders != nil || src.Tree != u.Canonical {
		t.Errorf("a non-Go unit must use its own canonical tree with no renders: %v", err)
	}
}

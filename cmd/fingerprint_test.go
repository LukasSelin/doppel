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
	fpLabels, fpTests, fpGenerated, fpLanguages, fpConfig = 20, "exclude", "exclude", nil, ""
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

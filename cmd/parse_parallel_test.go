package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/parallel"
	"github.com/LukasSelin/doppel/internal/parser"
)

// TestParseAllMatchesSequential pins the two things parallelism is allowed to
// leave alone: the order units arrive in, and the order warnings are printed
// in. Both are observable — units are addressed positionally for the whole
// rest of the pipeline, and stderr is read by the examples wrapper.
func TestParseAllMatchesSequential(t *testing.T) {
	dir := t.TempDir()
	// Enough files to clear minFilesPerWorker and actually run in parallel.
	const n = 200
	var paths []string
	for i := range n {
		p := filepath.Join(dir, fmt.Sprintf("f%03d.go", i))
		src := fmt.Sprintf("package p\n\nfunc F%03d(a int) int {\n\tif a > %d {\n\t\treturn a\n\t}\n\treturn %d\n}\n", i, i, i)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	// A directory named like a source file: os.ReadFile fails on it, which is
	// the warn-and-skip path, and it works the same on every platform.
	for _, i := range []int{7, 150} {
		bad := filepath.Join(dir, fmt.Sprintf("bad%03d.go", i))
		if err := os.Mkdir(bad, 0o755); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, bad)
	}
	// parseAll is given paths in the order the walk would produce them.
	sortPaths(paths)

	if got := parallel.Workers(len(paths), minFilesPerWorker); got < 2 {
		t.Skipf("%d workers on this machine; the parallel path is not exercised", got)
	}

	var wantWarn bytes.Buffer
	var want []parser.CodeUnit
	for _, p := range paths {
		u, err := parser.Parse(p)
		if err != nil {
			fmt.Fprintf(&wantWarn, "  warn: %s: %v\n", p, err)
			continue
		}
		want = append(want, u...)
	}

	var gotWarn bytes.Buffer
	got := parseAll(paths, &gotWarn)

	if len(got) != len(want) {
		t.Fatalf("parseAll returned %d units, sequential parse returned %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i].Name || got[i].File != want[i].File {
			t.Fatalf("unit %d is %s (%s), sequential had %s (%s) — order is not preserved",
				i, got[i].Name, got[i].File, want[i].Name, want[i].File)
		}
	}
	if gotWarn.String() != wantWarn.String() {
		t.Errorf("warnings differ\n got: %q\nwant: %q", gotWarn.String(), wantWarn.String())
	}
	if strings.Count(gotWarn.String(), "warn:") != 2 {
		t.Errorf("expected 2 warnings, got %q", gotWarn.String())
	}
}

func sortPaths(p []string) {
	for i := 1; i < len(p); i++ {
		for j := i; j > 0 && p[j] < p[j-1]; j-- {
			p[j], p[j-1] = p[j-1], p[j]
		}
	}
}

package bench

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// exampleTop is how many pairs an example report shows. Ten is the largest
// number that still reads as an example rather than as output.
const exampleTop = 10

// repoRoot resolves the module root from this test file's location.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(self), "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func gitHead(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// buildDoppel builds the doppel binary into a fresh temp dir and returns its
// path alongside the git rev that identifies the tree it was built from.
// Extracted so TestGenerateExamples and the baseline checksum generator
// (baseline_test.go) build it exactly once, the same way, rather than each
// keeping its own copy of the build invocation.
func buildDoppel(t *testing.T, root string) (bin, rev string) {
	t.Helper()
	bin = filepath.Join(t.TempDir(), "doppel")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = root
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("build doppel: %v", err)
	}
	return bin, gitHead(root)
}

// buildExampleReport runs bin over corpus c exactly the way `task examples`
// does — same flags, same corpus-relative working directory — and returns
// the full example-report content: the metadata table, the run diagnostics,
// then the markdown report body. This is the exact byte sequence
// TestGenerateExamples writes to examples/<corpus>.md, extracted so the
// baseline checksum generator can produce identical content without ever
// touching examples/ itself.
func buildExampleReport(t *testing.T, bin, doppelRev string, c Corpus) []byte {
	t.Helper()
	dir, err := Path(c)
	if err != nil {
		t.Fatal(err)
	}
	md := filepath.Join(t.TempDir(), "report.md")
	args := []string{"analyze", ".", "--tests", "exclude",
		"--top", fmt.Sprint(exampleTop), "--output", md}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("analyze %s: %v\n%s", c.Name, err, stderr.String())
	}
	body, err := os.ReadFile(md)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	fmt.Fprintf(&out, "# %s\n\n", c.Name)
	fmt.Fprintf(&out, "%s\n\n", c.Character)
	fmt.Fprintf(&out, "**What this rung shows:** %s\n\n", c.Exercises)
	fmt.Fprintf(&out, "| | |\n|---|---|\n")
	fmt.Fprintf(&out, "| Corpus | [%s](%s) |\n", c.Name, strings.TrimSuffix(c.Repo, ".git"))
	fmt.Fprintf(&out, "| Pinned at | `%s` (`%s`) |\n", c.Tag, c.Commit)
	fmt.Fprintf(&out, "| Project since | %d |\n", c.Since)
	fmt.Fprintf(&out, "| doppel | `%s` |\n", doppelRev)
	fmt.Fprintf(&out, "| Command | `doppel %s` |\n\n", strings.Join(args[:len(args)-2], " "))
	fmt.Fprintf(&out, "Run from the corpus root, so every path below is corpus-relative.\n")
	fmt.Fprintf(&out, "Regenerate with `task examples`.\n\n")
	fmt.Fprintf(&out, "## Run diagnostics\n\n")
	fmt.Fprintf(&out, "The corpus-level models doppel builds before ranking anything, as printed to stderr:\n\n")
	fmt.Fprintf(&out, "```\n%s```\n\n", diagnostics(stderr.String()))
	out.Write(body)
	return normalizeEOL(out.Bytes())
}

// TestGenerateExamples regenerates examples/<corpus>.md for every fetched
// corpus.
//
// It drives the built binary rather than the library on purpose: an example
// report should be what a reader gets from the documented command, culture
// and habitat and arena annotations included, not a reconstruction that can
// drift from cmd. The binary runs with its working directory set to the
// corpus so the report carries corpus-relative paths — an absolute path from
// whoever regenerated the file would be both noise and a small privacy leak.
func TestGenerateExamples(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_EXAMPLES") == "" {
		t.Skip("set DOPPEL_BENCH_EXAMPLES=1 to regenerate examples/ from the fetched corpora")
	}
	root := repoRoot(t)
	bin, doppelRev := buildDoppel(t, root)

	outDir := filepath.Join(root, "examples")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	any := false
	for _, c := range Corpora {
		if !Present(c) {
			t.Logf("skipping %s: not fetched", c.Name)
			continue
		}
		any = true
		t.Run(c.Name, func(t *testing.T) {
			content := buildExampleReport(t, bin, doppelRev, c)
			dst := filepath.Join(outDir, c.Name+".md")
			if err := os.WriteFile(dst, content, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("wrote %s (%d bytes)", dst, len(content))
		})
	}
	if !any {
		t.Fatal("no corpora fetched; run `task corpora` first")
	}
}

// diagnostics keeps the corpus-level stderr summary and drops the one line
// that names a path on the machine that regenerated the file.
func diagnostics(stderr string) string {
	var b strings.Builder
	for _, line := range strings.Split(string(normalizeEOL([]byte(stderr))), "\n") {
		if line == "" || strings.HasPrefix(line, "Markdown report written to") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// normalizeEOL keeps regenerated examples byte-identical across platforms;
// .gitattributes forces LF for markdown and the binary emits \n already, but
// stderr capture on Windows can carry \r\n through.
func normalizeEOL(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}

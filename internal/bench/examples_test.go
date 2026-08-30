package bench

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// exampleTop is how many pairs an example report shows. Ten is the largest
// number that still reads as an example rather than as output.
const exampleTop = 10

// The ladder table in examples/README.md is generated between these markers.
// Everything outside them is prose and stays hand-written — including the
// performance table, which is one machine's stopwatch and cannot be measured
// from a report.
const (
	ladderBegin = "<!-- BEGIN generated ladder -->"
	ladderEnd   = "<!-- END generated ladder -->"
)

// provenanceRow is the one line in a report that changes on every commit:
// the revision doppel was built from. Comparisons ignore it, so regenerating
// on an unchanged ranking is a no-op and CI does not commit seven files per
// push. The consequence is that the recorded revision means "the commit at
// which this report's content last changed", which is the more useful reading.
const provenanceRow = "| doppel |"

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

<<<<<<< HEAD
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
=======
// ladderRow is one rung's line in the examples/README.md summary table: the
// manifest coordinates plus the six quantities the run measured.
type ladderRow struct {
	Corpus   Corpus
	Funcs    int
	Pairs    int
	Kept     int
	Floor    string
	Concepts int
	Habitats int
>>>>>>> origin/master
}

// TestGenerateExamples regenerates examples/<corpus>.md for every fetched
// corpus, and the ladder table in examples/README.md when every rung is
// fetched.
//
// It drives the built binary rather than the library on purpose: an example
// report should be what a reader gets from the documented command, culture
// and habitat and arena annotations included, not a reconstruction that can
// drift from cmd. The binary runs with its working directory set to the
// corpus so the report carries corpus-relative paths — an absolute path from
// whoever regenerated the file would be both noise and a small privacy leak.
//
// With DOPPEL_BENCH_EXAMPLES_CHECK=1 it compares instead of writing and fails
// naming whatever drifted: the read-only half, for a pre-flight before a
// ranking change is pushed.
func TestGenerateExamples(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_EXAMPLES") == "" {
		t.Skip("set DOPPEL_BENCH_EXAMPLES=1 to regenerate examples/ from the fetched corpora")
	}
	check := os.Getenv("DOPPEL_BENCH_EXAMPLES_CHECK") != ""
	root := repoRoot(t)
	bin, doppelRev := buildDoppel(t, root)

	outDir := filepath.Join(root, "examples")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var rows []ladderRow
	for _, c := range Corpora {
		if !Present(c) {
			t.Logf("skipping %s: not fetched", c.Name)
			continue
		}
		t.Run(c.Name, func(t *testing.T) {
<<<<<<< HEAD
			content := buildExampleReport(t, bin, doppelRev, c)
			dst := filepath.Join(outDir, c.Name+".md")
			if err := os.WriteFile(dst, content, 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("wrote %s (%d bytes)", dst, len(content))
=======
			body, row := generateReport(t, bin, doppelRev, c)
			rows = append(rows, row)

			dst := filepath.Join(outDir, c.Name+".md")
			if check {
				if drifted(t, dst, body) {
					t.Errorf("%s.md is stale — run `task examples`", c.Name)
				}
				return
			}
			wrote, err := writeIfChanged(dst, body)
			if err != nil {
				t.Fatal(err)
			}
			if wrote {
				t.Logf("wrote %s (%d bytes)", dst, len(body))
			} else {
				t.Logf("%s is already current", dst)
			}
>>>>>>> origin/master
		})
	}
	if len(rows) == 0 {
		t.Fatal("no corpora fetched; run `task corpora` first")
	}

	// A partial fetch must leave the table alone: dropping the rungs that
	// happen not to be checked out would publish a shorter ladder than the
	// manifest describes, which reads as a deliberate claim.
	if len(rows) != len(Corpora) {
		t.Logf("ladder table left alone: %d of %d rungs fetched", len(rows), len(Corpora))
		return
	}
	readme := filepath.Join(outDir, "README.md")
	next, err := ladderREADME(readme, rows)
	if err != nil {
		t.Fatal(err)
	}
	if check {
		if drifted(t, readme, next) {
			t.Error("examples/README.md ladder table is stale — run `task examples`")
		}
		return
	}
	wrote, err := writeIfChanged(readme, next)
	if err != nil {
		t.Fatal(err)
	}
	if wrote {
		t.Logf("wrote %s", readme)
	} else {
		t.Logf("%s ladder table is already current", readme)
	}
}

// generateReport runs the built binary over one corpus and renders the whole
// example file, returning it alongside the row it contributes to the ladder.
func generateReport(t *testing.T, bin, doppelRev string, c Corpus) ([]byte, ladderRow) {
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
	diag := diagnostics(stderr.String())
	row, err := parseDiagnostics(c, diag)
	if err != nil {
		t.Fatalf("%s: %v\n%s", c.Name, err, diag)
	}

	var out bytes.Buffer
	fmt.Fprintf(&out, "# %s\n\n", c.Name)
	fmt.Fprintf(&out, "%s\n\n", c.Character)
	fmt.Fprintf(&out, "**What this rung shows:** %s\n\n", c.Exercises)
	fmt.Fprintf(&out, "| | |\n|---|---|\n")
	fmt.Fprintf(&out, "| Corpus | [%s](%s) |\n", c.Name, strings.TrimSuffix(c.Repo, ".git"))
	fmt.Fprintf(&out, "| Pinned at | `%s` (`%s`) |\n", c.Tag, c.Commit)
	fmt.Fprintf(&out, "| Project since | %d |\n", c.Since)
	fmt.Fprintf(&out, "%s `%s` |\n", provenanceRow, doppelRev)
	fmt.Fprintf(&out, "| Command | `doppel %s` |\n\n", strings.Join(args[:len(args)-2], " "))
	fmt.Fprintf(&out, "Run from the corpus root, so every path below is corpus-relative.\n")
	fmt.Fprintf(&out, "Regenerate with `task examples`; CI regenerates on every push to master.\n\n")
	fmt.Fprintf(&out, "## Run diagnostics\n\n")
	fmt.Fprintf(&out, "The corpus-level models doppel builds before ranking anything, as printed to stderr:\n\n")
	fmt.Fprintf(&out, "```\n%s```\n\n", diag)
	out.Write(body)

	return normalizeEOL(out.Bytes()), row
}

// The six quantities the ladder table reports, each already printed to stderr
// by a run. Parsing the diagnostics block rather than re-deriving them keeps
// the table and the reports describing the same run by construction.
var (
	reFuncs    = regexp.MustCompile(`(?m)^Found (\d+) functions`)
	rePairs    = regexp.MustCompile(`(?m)-> (\d+) unique pairs`)
	reKept     = regexp.MustCompile(`(?m)^\s*(\d+) pairs remain after struct-min=`)
	reFloor    = regexp.MustCompile(`(?m)^Calibration: .*-> threshold (\d+\.\d+)`)
	reDeclined = regexp.MustCompile(`(?m)^Calibration: .* declined`)
	reConcepts = regexp.MustCompile(`(?m)^Culture: (\d+) concepts modeled`)
	reHabitats = regexp.MustCompile(`(?m)^Habitats: (\d+) modeled`)
)

// parseDiagnostics reads one run's stderr block into a ladder row. A line
// that does not match is an error rather than a zero: a changed stderr format
// must fail the generator loudly, not publish wrong numbers into a table
// whose whole point is that nobody typed them.
func parseDiagnostics(c Corpus, diag string) (ladderRow, error) {
	row := ladderRow{Corpus: c}
	var err error
	if row.Funcs, err = matchInt(reFuncs, diag, "Found N functions"); err != nil {
		return row, err
	}
	if row.Pairs, err = matchInt(rePairs, diag, "N unique pairs"); err != nil {
		return row, err
	}
	if row.Concepts, err = matchInt(reConcepts, diag, "Culture: N concepts modeled"); err != nil {
		return row, err
	}
	if row.Habitats, err = matchInt(reHabitats, diag, "Habitats: N modeled"); err != nil {
		return row, err
	}
	// No struct-min line means no overlap filter ran, so every compared pair
	// was kept. That is a fact about the run, not a missing measurement.
	if m := reKept.FindStringSubmatch(diag); m != nil {
		if row.Kept, err = strconv.Atoi(m[1]); err != nil {
			return row, err
		}
	} else {
		row.Kept = row.Pairs
	}
	switch {
	case reFloor.MatchString(diag):
		row.Floor = reFloor.FindStringSubmatch(diag)[1]
	case reDeclined.MatchString(diag):
		// Too few null pairs to measure anything; the run kept the flag
		// defaults. Saying so is honest where printing 0.60 would present a
		// default as a measurement.
		row.Floor = "declined"
	default:
		return row, fmt.Errorf("no Calibration line in diagnostics")
	}
	return row, nil
}

func matchInt(re *regexp.Regexp, diag, want string) (int, error) {
	m := re.FindStringSubmatch(diag)
	if m == nil {
		return 0, fmt.Errorf("diagnostics carry no %q line", want)
	}
	return strconv.Atoi(m[1])
}

// renderLadder is the generated block of examples/README.md.
func renderLadder(rows []ladderRow) string {
	var b strings.Builder
	b.WriteString("| Corpus | Since | Pinned | Functions | Pairs compared | Kept | Code-shape floor | Concepts modeled | Habitats |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| [%s](%s.md) | %d | `%s` | %d | %d | %d | %s | %d | %d |\n",
			r.Corpus.Name, r.Corpus.Name, r.Corpus.Since, r.Corpus.Tag,
			r.Funcs, r.Pairs, r.Kept, r.Floor, r.Concepts, r.Habitats)
	}
	return b.String()
}

// ladderREADME returns examples/README.md with the generated block replaced.
func ladderREADME(path string, rows []ladderRow) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(normalizeEOL(data))
	begin := strings.Index(text, ladderBegin)
	end := strings.Index(text, ladderEnd)
	if begin < 0 || end < begin {
		return nil, fmt.Errorf("%s: missing %s / %s markers", path, ladderBegin, ladderEnd)
	}
	head := text[:begin+len(ladderBegin)]
	tail := text[end:]
	return []byte(head + "\n" + renderLadder(rows) + tail), nil
}

// writeIfChanged writes next to dst unless the committed file already says
// the same thing modulo the provenance row, and reports whether it wrote.
func writeIfChanged(dst string, next []byte) (bool, error) {
	old, err := os.ReadFile(dst)
	if err == nil && bytes.Equal(stripProvenance(normalizeEOL(old)), stripProvenance(next)) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return true, os.WriteFile(dst, next, 0o644)
}

// drifted reports whether the committed file differs from next, ignoring the
// provenance row, and logs the first differing line so a failure names
// something a reader can act on.
func drifted(t *testing.T, dst string, next []byte) bool {
	t.Helper()
	old, err := os.ReadFile(dst)
	if err != nil {
		t.Logf("%s: %v", dst, err)
		return true
	}
	want := strings.Split(string(stripProvenance(normalizeEOL(old))), "\n")
	got := strings.Split(string(stripProvenance(next)), "\n")
	for i := 0; i < len(want) || i < len(got); i++ {
		w, g := line(want, i), line(got, i)
		if w != g {
			t.Logf("%s line %d:\n  committed: %s\n  generated: %s", dst, i+1, w, g)
			return true
		}
	}
	return false
}

func line(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return "<end of file>"
}

// stripProvenance drops the one row that moves with every commit.
func stripProvenance(b []byte) []byte {
	var out bytes.Buffer
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(l, provenanceRow) {
			continue
		}
		out.WriteString(l)
		out.WriteByte('\n')
	}
	return out.Bytes()
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

// TestExamplesManifest is the offline half of the drift check: it needs no
// corpus and no network, so it runs in a plain `go test ./...` and catches
// the cheap failure — a rung added or repinned without its report being
// regenerated. What it cannot see is a ranking change, which only running the
// tool over the ladder can measure; that is what the CI suite is for.
func TestExamplesManifest(t *testing.T) {
	root := repoRoot(t)
	outDir := filepath.Join(root, "examples")

	readme, err := os.ReadFile(filepath.Join(outDir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{ladderBegin, ladderEnd} {
		if !bytes.Contains(readme, []byte(marker)) {
			t.Errorf("examples/README.md has no %s marker; the ladder table cannot be generated", marker)
		}
	}

	for _, c := range Corpora {
		path := filepath.Join(outDir, c.Name+".md")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s is in the ladder but has no example report: %v", c.Name, err)
			continue
		}
		want := fmt.Sprintf("| Pinned at | `%s` (`%s`) |", c.Tag, c.Commit)
		if !bytes.Contains(normalizeEOL(body), []byte(want)) {
			t.Errorf("examples/%s.md does not report the pinned commit %s (%s) — run `task examples`",
				c.Name, c.Tag, c.Commit)
		}
	}
}

// TestLadderMatchesReports closes the loop between the two generated things
// offline: the ladder table is re-derived from the diagnostics blocks the
// committed reports quote, and must come out byte-identical to the table
// committed in examples/README.md. It exercises parseDiagnostics and
// renderLadder against real output, and it fails if somebody edits one of the
// two by hand.
func TestLadderMatchesReports(t *testing.T) {
	root := repoRoot(t)
	outDir := filepath.Join(root, "examples")

	var rows []ladderRow
	for _, c := range Corpora {
		body, err := os.ReadFile(filepath.Join(outDir, c.Name+".md"))
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		diag, err := reportDiagnostics(string(normalizeEOL(body)))
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		row, err := parseDiagnostics(c, diag)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		rows = append(rows, row)
	}

	readme, err := os.ReadFile(filepath.Join(outDir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(normalizeEOL(readme))
	begin := strings.Index(text, ladderBegin)
	end := strings.Index(text, ladderEnd)
	if begin < 0 || end < begin {
		t.Fatal("examples/README.md has no ladder markers")
	}
	got := text[begin+len(ladderBegin)+1 : end]
	if want := renderLadder(rows); got != want {
		t.Errorf("ladder table disagrees with the reports — run `task examples`\ncommitted:\n%s\nfrom reports:\n%s", got, want)
	}
}

// reportDiagnostics recovers the fenced stderr block an example report quotes
// under "## Run diagnostics".
func reportDiagnostics(report string) (string, error) {
	const heading = "## Run diagnostics"
	i := strings.Index(report, heading)
	if i < 0 {
		return "", fmt.Errorf("report has no %q section", heading)
	}
	rest := report[i:]
	open := strings.Index(rest, "```\n")
	if open < 0 {
		return "", fmt.Errorf("report has no fenced diagnostics block")
	}
	rest = rest[open+len("```\n"):]
	closing := strings.Index(rest, "```")
	if closing < 0 {
		return "", fmt.Errorf("report has an unterminated diagnostics block")
	}
	return rest[:closing], nil
}

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/dashboard"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

func runTimelineCmd(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	// Flags are package-level and cobra does not reset them between runs.
	timelineOutput, timelineFormat, timelineTarget = "", "text", "fixture"
	timelineLabels = nil
	code := exitDiffOK
	prev := diffExit
	diffExit = func(c int) { code = c }
	t.Cleanup(func() { diffExit = prev })

	var out, errb bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errb)
	rootCmd.SetArgs(append([]string{"timeline"}, args...))
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	_ = rootCmd.Execute()
	return out.String(), errb.String(), code
}

// The three revisions the command tests run over: a rename, then a move plus a
// deletion. Same shape as the identity fixtures and for the same reason — a
// corpus with too little variety in it makes every label weight zero.
const tlFiller = `
func Encode(v string) string {
	out := ""
	for _, r := range v {
		out += string(r)
	}
	return out
}

func Clamp(v, lo, hi int) int {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	}
	return v
}

func Count(xs []int, want int) int {
	n := 0
	for _, x := range xs {
		if x == want {
			n++
		}
	}
	return n
}
`

const tlTotal = `
func Total(xs []int) int {
	sum := 0
	for _, x := range xs {
		if x > 0 {
			sum += x
		}
	}
	return sum
}
`

var tlSum = strings.Replace(tlTotal, "func Total(", "func Sum(", 1)

const tlHelper = `
func Helper(a, b int) int {
	if a > b {
		return a
	}
	return b
}
`

// tlSeries writes three snapshots of one refactor and returns their paths.
func tlSeries(t *testing.T) (dir string, paths []string) {
	t.Helper()
	dir = t.TempDir()
	v1 := corpusFrom(t, map[string]string{"alpha/a.go": "package alpha\n" + tlFiller + tlTotal + tlHelper})
	v2 := corpusFrom(t, map[string]string{"alpha/a.go": "package alpha\n" + tlFiller + tlSum + tlHelper})
	v3 := corpusFrom(t, map[string]string{
		"alpha/a.go": "package alpha\n" + tlFiller,
		"beta/b.go":  "package beta\n" + tlSum,
	})
	return dir, []string{
		writeSnapshot(t, dir, "0000-aaaaaaa.json", v1),
		writeSnapshot(t, dir, "0001-bbbbbbb.json", v2),
		writeSnapshot(t, dir, "0002-ccccccc.json", v3),
	}
}

func TestTimelineCommandEndToEnd(t *testing.T) {
	dir, paths := tlSeries(t)
	page := filepath.Join(dir, "timeline.html")

	stdout, stderr, code := runTimelineCmd(t, append(paths, "-o", page)...)
	if code != exitDiffOK {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	for _, want := range []string{"3 revisions", "aaaaaaa", "bbbbbbb", "ccccccc"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("text output is missing %q:\n%s", want, stdout)
		}
	}
	// The classification, not just the counts: the rename is the finding the
	// per-step diff exists to produce.
	if !strings.Contains(stdout, "1 renamed") {
		t.Errorf("the rename was not reported:\n%s", stdout)
	}

	data, err := os.ReadFile(page)
	if err != nil {
		t.Fatalf("read page: %v", err)
	}
	if !bytes.Contains(data, []byte("doppel-data")) {
		t.Error("the rendered page carries no payload")
	}
	if !bytes.Contains(data, []byte("track-cell")) {
		t.Error("the timeline stylesheet was not inlined")
	}
}

// TestTimelineCommandLabelsDefaultToFileNames pins the naming convention: an
// index prefix exists to keep a shell glob in series order and is not part of
// the label a reader sees.
func TestTimelineCommandLabelsDefaultToFileNames(t *testing.T) {
	got := stepLabels([]string{"runs/0000-abc1234.json", "runs/0012_def5678.json", "plain.json"})
	want := []string{"abc1234", "def5678", "plain"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("label %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTimelineCommandExplicitLabels(t *testing.T) {
	_, paths := tlSeries(t)
	stdout, stderr, code := runTimelineCmdWithLabels(t, []string{"v1", "v2", "v3"}, paths...)
	if code != exitDiffOK {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "v2") {
		t.Errorf("explicit labels were not used:\n%s", stdout)
	}
}

func runTimelineCmdWithLabels(t *testing.T, labels []string, paths ...string) (string, string, int) {
	t.Helper()
	args := append([]string{}, paths...)
	args = append(args, "--labels", strings.Join(labels, ","))
	return runTimelineCmd(t, args...)
}

func TestTimelineCommandJSON(t *testing.T) {
	_, paths := tlSeries(t)
	stdout, stderr, code := runTimelineCmd(t, append(paths, "--format", "json")...)
	if code != exitDiffOK {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	var p dashboard.TimelinePayload
	if err := json.Unmarshal([]byte(stdout), &p); err != nil {
		t.Fatalf("output does not parse as JSON: %v", err)
	}
	if p.Schema != dashboard.TimelineSchema {
		t.Errorf("Schema = %d, want %d", p.Schema, dashboard.TimelineSchema)
	}
	if len(p.Steps) != 3 || len(p.Changes) != 2 {
		t.Fatalf("payload has %d steps and %d changes, want 3 and 2", len(p.Steps), len(p.Changes))
	}
	// Every step carries all eight classes, zeros included — a stable shape.
	if len(p.Changes[0].Counts) != 8 {
		t.Errorf("a transition carries %d class counts, want 8", len(p.Changes[0].Counts))
	}
	var renamed bool
	for _, c := range p.Changes[0].Changes {
		if c.Class == "renamed" {
			renamed = true
		}
		if c.Class == "unchanged" {
			t.Error("unchanged functions are counted, never listed")
		}
	}
	if !renamed {
		t.Error("the rename is missing from the first transition")
	}
}

func TestTimelineCommandUnreadableExitsOne(t *testing.T) {
	dir, paths := tlSeries(t)
	missing := filepath.Join(dir, "nope.json")
	_, stderr, code := runTimelineCmd(t, paths[0], missing)
	if code != exitDiffUnreadable {
		t.Errorf("exit code = %d, want %d\nstderr: %s", code, exitDiffUnreadable, stderr)
	}
}

// TestTimelineRefusesAMovedOperatingPoint is this command's own check, and the
// reason it is stricter than internal/identity. A series puts each run's pair
// counts on one axis, and those are exactly the corpus-relative numbers a
// moved threshold invalidates.
func TestTimelineRefusesAMovedOperatingPoint(t *testing.T) {
	dir, paths := tlSeries(t)

	var s snapshot.Snapshot
	data, err := os.ReadFile(paths[1])
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("parse: %v", err)
	}
	s.Params.Threshold += 0.1
	shifted := writeSnapshot(t, dir, "shifted.json", s)

	_, stderr, code := runTimelineCmd(t, paths[0], shifted)
	if code != exitDiffIncomparabl {
		t.Errorf("exit code = %d, want %d\nstderr: %s", code, exitDiffIncomparabl, stderr)
	}
	if !strings.Contains(stderr, "operating point") {
		t.Errorf("the refusal does not name the cause:\n%s", stderr)
	}
}

// TestTimelineRefusesACalibratedSeries is the same rule for the default case,
// which is the one that would actually bite: calibration is on by default, so a
// series produced without pinned thresholds derives a different floor per
// revision.
func TestTimelineRefusesACalibratedSeries(t *testing.T) {
	dir, paths := tlSeries(t)

	var a, b snapshot.Snapshot
	for i, p := range []string{paths[0], paths[1]} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		target := &a
		if i == 1 {
			target = &b
		}
		if err := json.Unmarshal(data, target); err != nil {
			t.Fatalf("parse: %v", err)
		}
		target.Params.Calibrate = 0.01
	}
	pa := writeSnapshot(t, dir, "cal-a.json", a)
	pb := writeSnapshot(t, dir, "cal-b.json", b)

	_, stderr, code := runTimelineCmd(t, pa, pb)
	if code != exitDiffIncomparabl {
		t.Errorf("exit code = %d, want %d\nstderr: %s", code, exitDiffIncomparabl, stderr)
	}
	if !strings.Contains(stderr, "calibrated") {
		t.Errorf("the refusal does not name calibration:\n%s", stderr)
	}
}

// TestTimelineWarnsAboutReportCaps pins the bound a reader could not otherwise
// see: `analyze --format json` stores the *ranked* pair list, so a series left
// at the default --top carries twenty pairs per revision however large the
// corpus, and its pair half would read as a quiet history.
func TestTimelineWarnsAboutReportCaps(t *testing.T) {
	dir, paths := tlSeries(t)

	var capped []string
	for i, p := range paths[:2] {
		var s snapshot.Snapshot
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if err := json.Unmarshal(data, &s); err != nil {
			t.Fatalf("parse: %v", err)
		}
		s.Params.Top = 20
		s.Params.MaxPerFunc = 2
		capped = append(capped, writeSnapshot(t, dir, "capped-"+string(rune('a'+i))+".json", s))
	}

	_, stderr, code := runTimelineCmd(t, capped...)
	if code != exitDiffOK {
		t.Fatalf("exit code = %d, want 0\nstderr: %s", code, stderr)
	}
	if !strings.Contains(stderr, "--top 0 --max-per-func 0") {
		t.Errorf("no warning about the report caps:\n%s", stderr)
	}
}

// TestTimelineNeedsTwoSnapshots asserts on the returned error rather than the
// exit code: an argument-count failure is cobra's own, so RunE never runs and
// the shared exit seam is never reached. main exits 1 on it like any other
// usage error.
func TestTimelineNeedsTwoSnapshots(t *testing.T) {
	_, paths := tlSeries(t)
	timelineOutput, timelineFormat, timelineTarget = "", "text", "fixture"
	timelineLabels = nil
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"timeline", paths[0]})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	if err := rootCmd.Execute(); err == nil {
		t.Error("a one-snapshot series should not render a history")
	}
}

// TestTimelinePageIsDeterministic is the repo's own invariant at this command's
// boundary: an unchanged series must render byte-identical HTML.
func TestTimelinePageIsDeterministic(t *testing.T) {
	dir, paths := tlSeries(t)
	one := filepath.Join(dir, "one.html")
	two := filepath.Join(dir, "two.html")

	if _, stderr, code := runTimelineCmd(t, append(append([]string{}, paths...), "-o", one)...); code != exitDiffOK {
		t.Fatalf("first render failed: %s", stderr)
	}
	if _, stderr, code := runTimelineCmd(t, append(append([]string{}, paths...), "-o", two)...); code != exitDiffOK {
		t.Fatalf("second render failed: %s", stderr)
	}
	a, err := os.ReadFile(one)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(two)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("two renders of one series differ; the page must be reproducible")
	}
}

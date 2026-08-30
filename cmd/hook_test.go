package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// TestBaselinePathIsContained is the security property of the whole feature:
// session_id arrives from outside and is used to build a filesystem path.
// Hashing it makes traversal, absolute paths, drive letters, reserved device
// names and length limits impossible by construction, so the test asserts
// containment rather than checking a blocklist somebody has to keep correct.
func TestBaselinePathIsContained(t *testing.T) {
	hostile := []string{
		"",
		".",
		"..",
		"../../../etc/passwd",
		`..\..\..\Windows\System32\config\SAM`,
		"/etc/passwd",
		`C:\Windows\System32`,
		"con",
		"NUL",
		"a:b:c",
		strings.Repeat("x", 10_000),
		"héllo wörld/../..",
		"sess\x00id",
		"*",
	}
	for _, id := range hostile {
		got := baselinePath(id)
		if dir := filepath.Dir(got); dir != baselineDir() {
			t.Errorf("session id %q escaped the baseline dir: %q", short(id), got)
		}
		if !strings.HasSuffix(got, ".json") {
			t.Errorf("session id %q produced a non-json path: %q", short(id), got)
		}
	}
}

func TestBaselinePathIsStableAndDistinct(t *testing.T) {
	if baselinePath("abc") != baselinePath("abc") {
		t.Error("the same session id must map to the same baseline path")
	}
	if baselinePath("abc") == baselinePath("abd") {
		t.Error("different session ids must map to different baseline paths")
	}
}

func TestDeltaPathSitsBesideItsBaseline(t *testing.T) {
	b := baselinePath("s1")
	d := deltaPathFor(b)
	if filepath.Dir(d) != filepath.Dir(b) {
		t.Errorf("delta path %q is not beside baseline %q", d, b)
	}
	if !strings.HasSuffix(d, ".impact.json") {
		t.Errorf("delta path %q does not end in .impact.json", d)
	}
}

func TestReadHookInput(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr bool
		wantID  string
	}{
		{"full payload", `{"session_id":"s1","cwd":"/repo","source":"startup"}`, false, "s1"},
		{"unknown fields ignored", `{"session_id":"s1","future_field":{"a":1}}`, false, "s1"},
		{"missing fields", `{}`, false, ""},
		{"empty", ``, true, ""},
		{"malformed", `{not json`, true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in, err := readHookInput(strings.NewReader(tc.payload))
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && in.SessionID != tc.wantID {
				t.Errorf("SessionID = %q, want %q", in.SessionID, tc.wantID)
			}
		})
	}
}

func TestBaselineRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "b.json")
	want := baselineFile{
		Root:      "C:/repo",
		CreatedAt: "2026-08-21T00:00:00Z",
		Snapshot: snapshot.Snapshot{
			Schema: snapshot.Schema, Doppel: "test", Ontology: "1.0.0", Functions: 2,
			Units: []snapshot.Unit{{Key: "a.One", Digest: "d1"}},
		},
	}
	if err := writeJSONAtomic(path, want); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := readBaseline(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Root != want.Root || got.Snapshot.Functions != 2 || got.Snapshot.Units[0].Key != "a.One" {
		t.Errorf("round trip lost data: %+v", got)
	}
}

func TestReadBaselineMissingAndCorrupt(t *testing.T) {
	dir := t.TempDir()

	if _, err := readBaseline(filepath.Join(dir, "absent.json")); err == nil {
		t.Error("a missing baseline must be an error, so the caller stays silent")
	}

	corrupt := filepath.Join(dir, "corrupt.json")
	if err := os.WriteFile(corrupt, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readBaseline(corrupt); err == nil {
		t.Fatal("a corrupt baseline must be an error")
	}
	// It must also be removed: diffing against garbage is worse than having
	// no origin at all, so the next session starts clean.
	if _, err := os.Stat(corrupt); !os.IsNotExist(err) {
		t.Error("a corrupt baseline must be deleted, not left to be re-read")
	}
}

// A hook must never block a session over a measurement, so every failure path
// exits zero, writes nothing to stderr, and emits either nothing or valid JSON.
func TestHookCommandsFailSilently(t *testing.T) {
	tests := []struct {
		name    string
		cmdName string
		payload string
	}{
		{"session-start with malformed payload", "session-start", "{not json"},
		{"session-start with empty payload", "session-start", ""},
		{"stop with malformed payload", "stop", "{not json"},
		{"stop with no baseline", "stop", `{"session_id":"no-such-session-at-all","cwd":"."}`},
		{"stop with unreadable root", "stop", `{"session_id":"x","cwd":"/nonexistent-path-xyz"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			root := rootCmd
			root.SetIn(strings.NewReader(tc.payload))
			root.SetOut(&out)
			root.SetErr(&errOut)
			root.SetArgs([]string{"hook", tc.cmdName})

			if err := root.Execute(); err != nil {
				t.Fatalf("hook returned an error: %v", err)
			}
			if errOut.Len() != 0 {
				t.Errorf("hook wrote to stderr: %q", errOut.String())
			}
			if out.Len() > 0 {
				var v any
				if err := json.Unmarshal(out.Bytes(), &v); err != nil {
					t.Errorf("hook stdout is not valid JSON: %q", out.String())
				}
			}
		})
	}
}

func short(s string) string {
	if len(s) > 40 {
		return s[:40] + "..."
	}
	return s
}

// The Stop hook's output reaches the model only by continuing the turn, so the
// harness re-enters this hook on the very turn it just continued. Speaking
// again would continue it again; the loop ends only when Claude Code overrides
// it with a warning. stop_hook_active is the guard, so it has to be parsed.
func TestReadHookInputParsesStopHookActive(t *testing.T) {
	in, err := readHookInput(strings.NewReader(`{"session_id":"s","stop_hook_active":true}`))
	if err != nil {
		t.Fatalf("readHookInput: %v", err)
	}
	if !in.StopHookActive {
		t.Error("StopHookActive = false, want true")
	}
	// Absent must read as false, or the very first Stop of a turn would be
	// treated as a re-entry and the hook would never say anything at all.
	in, err = readHookInput(strings.NewReader(`{"session_id":"s"}`))
	if err != nil {
		t.Fatalf("readHookInput: %v", err)
	}
	if in.StopHookActive {
		t.Error("StopHookActive = true for an absent field, want false")
	}
}

func TestUnreportedDropsAlreadySurfacedFindings(t *testing.T) {
	findings := []reporter.Finding{
		{Key: "new:a|b", Line: "a <-> b"},
		{Key: "gate:c|d", Line: "c <-> d"},
	}
	got := unreported(findings, []string{"new:a|b"})
	if len(got) != 1 || got[0].Key != "gate:c|d" {
		t.Fatalf("unreported = %+v, want only gate:c|d", got)
	}
	if got := unreported(findings, []string{"new:a|b", "gate:c|d"}); len(got) != 0 {
		t.Fatalf("want nothing left once both are reported, got %+v", got)
	}
}

// A delta is cumulative against the session-start origin, so a finding that is
// not remembered is re-reported every turn — and in agent mode continues every
// turn. The ledger is what makes the feature bearable.
func TestWithReportedAccumulatesWithoutMovingTheOrigin(t *testing.T) {
	base := baselineFile{
		Root:      "/repo",
		CreatedAt: "2026-01-01T00:00:00Z",
		Snapshot:  snapshot.Snapshot{Schema: snapshot.Schema, Functions: 7},
		Reported:  []string{"new:a|b"},
	}
	got := withReported(base, []reporter.Finding{
		{Key: "gate:c|d"},
		{Key: "new:a|b"}, // already there; must not duplicate
	})

	want := []string{"gate:c|d", "new:a|b"} // sorted
	if len(got.Reported) != len(want) {
		t.Fatalf("Reported = %v, want %v", got.Reported, want)
	}
	for i := range want {
		if got.Reported[i] != want[i] {
			t.Fatalf("Reported = %v, want %v (sorted, deduped)", got.Reported, want)
		}
	}
	if got.Snapshot.Functions != base.Snapshot.Functions || got.CreatedAt != base.CreatedAt {
		t.Error("withReported moved the measurement origin; it must only add to the ledger")
	}
}

func TestHookNotifyDefaultsAndValidates(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    string
		wantErr bool
	}{
		{"no config file", "", NotifyAgent, false},
		{"absent key", `{"threshold":0.7}`, NotifyAgent, false},
		{"explicit user", `{"hook-notify":"user"}`, NotifyUser, false},
		{"explicit off", `{"hook-notify":"off"}`, NotifyOff, false},
		{"invalid", `{"hook-notify":"shout"}`, NotifyAgent, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.body != "" {
				if err := os.WriteFile(filepath.Join(dir, ".doppel.json"), []byte(tc.body), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			got, err := hookNotify(dir)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("hookNotify = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReadHookInputParsesPromptAndToolFields(t *testing.T) {
	in, err := readHookInput(strings.NewReader(
		`{"session_id":"s","prompt":"fix internal/culture","tool_name":"Edit","tool_input":{"file_path":"C:/repo/a.go","old_string":"x"}}`))
	if err != nil {
		t.Fatalf("readHookInput: %v", err)
	}
	if in.Prompt != "fix internal/culture" {
		t.Errorf("Prompt = %q", in.Prompt)
	}
	if in.ToolName != "Edit" || in.ToolInput.FilePath != "C:/repo/a.go" {
		t.Errorf("tool fields = %q / %q", in.ToolName, in.ToolInput.FilePath)
	}
}

func TestRelativeToRootRejectsOutsiders(t *testing.T) {
	root := filepath.Join("C:", "repo")
	if rel, ok := relativeToRoot(root, filepath.Join(root, "internal", "a.go")); !ok || rel != "internal/a.go" {
		t.Errorf("inside file: rel=%q ok=%v", rel, ok)
	}
	if _, ok := relativeToRoot(root, filepath.Join("C:", "elsewhere", "b.go")); ok {
		t.Error("file outside the analyzed tree was accepted")
	}
}

// The digest prints a bounded head of the findings, and the ledger retires
// whatever it is handed. Handing it the whole list therefore suppresses, for
// the rest of the session, findings nobody was ever shown — so the ledger gets
// what AgentDigest rendered, and the remainder leads the next turn.
func TestLedgerRetiresOnlyTheFindingsShown(t *testing.T) {
	var all []reporter.Finding
	for _, n := range []string{"one", "two", "three", "four", "five"} {
		all = append(all, reporter.Finding{Key: "new:" + n, Line: n + " <-> other"})
	}
	base := baselineFile{Root: "/repo", Snapshot: snapshot.Snapshot{Schema: snapshot.Schema}}

	note, shown := reporter.AgentDigest(unreported(all, base.Reported))
	if note == "" {
		t.Fatal("no digest rendered for five findings")
	}
	base = withReported(base, shown)

	left := unreported(all, base.Reported)
	if len(left) != len(all)-len(shown) {
		t.Fatalf("%d findings left for the next turn, want %d", len(left), len(all)-len(shown))
	}
	for _, f := range left {
		if strings.Contains(note, f.Line) {
			t.Errorf("finding %q was shown yet not retired", f.Line)
		}
	}

	// The second turn surfaces the remainder, and then there is nothing left.
	_, shown2 := reporter.AgentDigest(left)
	base = withReported(base, shown2)
	if rest := unreported(all, base.Reported); len(rest) != 0 {
		t.Errorf("%d findings still unreported after two turns, want 0", len(rest))
	}
}

// A later turn measures at the operating point the baseline was written at.
//
// Deriving it again each turn is what this prevents: the session is editing the
// corpus, so the null distribution moves, and a threshold that shifts by a
// hundredth makes Params unequal and silences the Stop hook for a turn that
// nothing was wrong with.
func TestPinThresholdsSuppliesTheBaselineOperatingPoint(t *testing.T) {
	p := Params{Threshold: 0.60, Calibrate: defaultCalibrateRate, NoOverlapFilter: true}
	pinned := &snapshot.Params{Threshold: 0.48, StructMin: 0.39, Calibrate: 0.01}

	got := pinThresholds(p, pinned)

	if got.Threshold != 0.48 {
		t.Errorf("Threshold = %v, want the baseline's 0.48", got.Threshold)
	}
	if !got.Pinned {
		t.Error("Pinned = false: the run would derive its own thresholds")
	}
	if got.Calibrate != 0.01 {
		t.Errorf("Calibrate = %v, want the baseline's 0.01", got.Calibrate)
	}
	// The baseline records the overlap floor for comparability, but a hook run
	// never applies one — it diffs the full candidate set.
	if got.StructMin != 0 {
		t.Errorf("StructMin = %v, want 0: a hook must not gain an overlap filter", got.StructMin)
	}
}

// No baseline means derive: session start is the run that establishes the
// operating point, and user-prompt scopes a digest without diffing anything.
func TestPinThresholdsWithoutABaselineDerives(t *testing.T) {
	p := Params{Threshold: 0.60, Calibrate: defaultCalibrateRate}

	got := pinThresholds(p, nil)

	if got.Pinned {
		t.Error("Pinned = true with no baseline: nothing supplied the thresholds")
	}
	if got.Calibrate != defaultCalibrateRate {
		t.Errorf("Calibrate = %v, want %v: the run must derive its own", got.Calibrate, defaultCalibrateRate)
	}
}

// A pinned run and the baseline it inherited from must compare as the same
// question. This is the property the whole pinning mechanism exists for.
func TestPinnedRunStaysComparableToItsBaseline(t *testing.T) {
	base := snapshot.Params{
		Threshold: 0.48, MinNodes: 12, ChannelK: 5,
		TestsMode: "exclude", Generated: "exclude", Calibrate: 0.01,
	}
	p, err := hookParams(t.TempDir())
	if err != nil {
		t.Fatalf("hookParams: %v", err)
	}
	got := pinThresholds(p, &base)

	head := snapshot.Params{
		Threshold: got.Threshold, Top: got.TopN, MinNodes: got.MinNodes,
		StructMin: got.StructMin, ChannelK: got.ChannelK, MaxPerFunc: got.MaxPerFunc,
		TestsMode: got.TestsMode, Generated: got.Generated, Calibrate: got.Calibrate,
	}
	if !head.Equal(base) {
		t.Errorf("pinned params differ from the baseline:\n head %+v\n base %+v", head, base)
	}
}

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

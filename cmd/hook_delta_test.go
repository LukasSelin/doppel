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

// hookResponse is the subset of a hook's stdout the tests read.
type hookResponse struct {
	SystemMessage      string `json:"systemMessage"`
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

// runHook drives one hook subcommand through the root command, the way the
// harness does, and returns its two streams.
func runHook(t *testing.T, sub, payload string) (hookResponse, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	rootCmd.SetIn(strings.NewReader(payload))
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errb)
	rootCmd.SetArgs([]string{"hook", sub})
	t.Cleanup(func() {
		rootCmd.SetIn(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook %s returned an error: %v", sub, err)
	}
	var resp hookResponse
	if out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
			t.Fatalf("hook %s stdout is not valid JSON: %q", sub, out.String())
		}
	}
	return resp, out.String(), errb.String()
}

func hookPayload(t *testing.T, sessionID, cwd string) string {
	t.Helper()
	data, err := json.Marshal(map[string]any{"session_id": sessionID, "cwd": cwd, "source": "startup"})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// writeCorpus puts one file in a fresh directory and returns the directory.
func writeCorpus(t *testing.T, dir, rel, src string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestStopHookDeltaReportGate is the task's fixture end to end through the
// hook: session-start records a baseline, the session renames one function and
// copies another, and `hook stop` reports the rename as `renamed` and the copy
// as a created pair carrying the stored explanation.
//
// It drives the real subcommands over a real tree, so the baseline, the
// operating point, the analysis and both digests are the shipped ones.
func TestStopHookDeltaReportGate(t *testing.T) {
	dir := t.TempDir()
	writeCorpus(t, dir, "svc/svc.go", gateSrcBefore)

	sessionID := "doppel-t9-gate-" + filepath.Base(dir)
	baseline := baselinePath(sessionID)
	t.Cleanup(func() {
		os.Remove(baseline)
		os.Remove(deltaPathFor(baseline))
	})

	payload := hookPayload(t, sessionID, dir)
	if _, _, errOut := runHook(t, "session-start", payload); errOut != "" {
		t.Fatalf("session-start wrote to stderr: %q", errOut)
	}
	if _, err := os.Stat(baseline); err != nil {
		t.Fatalf("session-start did not record a baseline: %v", err)
	}

	// The session: one rename, one verbatim copy.
	writeCorpus(t, dir, "svc/svc.go", gateSrcAfter)

	resp, _, errOut := runHook(t, "stop", payload)
	if errOut != "" {
		t.Fatalf("stop wrote to stderr: %q", errOut)
	}

	msg := resp.SystemMessage
	if msg == "" {
		t.Fatal("stop said nothing about a session that renamed and copied a function")
	}
	// The delta report leads.
	if !strings.HasPrefix(msg, "doppel delta since the session baseline:") {
		t.Errorf("the delta report must lead the user digest:\n%s", msg)
	}
	for _, want := range []string{
		"renamed 1",
		"svc.Total",
		"svc.Sum",
		"svc.Clip <-> svc.Trim",
		"explain: identical after rename",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("user digest is missing %q:\n%s", want, msg)
		}
	}

	// The agent note is sent (a new merge-worthy attributable pair is a
	// notable finding) and the delta report leads it too.
	note := resp.HookSpecificOutput.AdditionalContext
	if note == "" {
		t.Fatalf("no agent note for a new merge-worthy pair; response was %+v", resp)
	}
	if !strings.HasPrefix(note, "doppel matched this session's functions against the session baseline:") {
		t.Errorf("the delta report must lead the agent note:\n%s", note)
	}
	if !strings.Contains(note, "explain: identical after rename") {
		t.Errorf("the agent note must carry the created pair's explanation:\n%s", note)
	}
	if !strings.Contains(note, "This is a measurement, not a request.") {
		t.Errorf("the agent note must keep its closing bound:\n%s", note)
	}
}

// The delta is cumulative against the session-start origin, so the identity
// findings need the same ledger the notable ones have: without it a rename is
// restated in every later agent note for the rest of the session.
//
// The user digest is deliberately not ledgered — it lets the turn end, and it
// is a cumulative statement of where the session stands.
func TestStopHookLedgersIdentityFindings(t *testing.T) {
	dir := t.TempDir()
	writeCorpus(t, dir, "svc/svc.go", gateSrcBefore)

	sessionID := "doppel-t9-ledger-" + filepath.Base(dir)
	baseline := baselinePath(sessionID)
	t.Cleanup(func() {
		os.Remove(baseline)
		os.Remove(deltaPathFor(baseline))
	})

	payload := hookPayload(t, sessionID, dir)
	runHook(t, "session-start", payload)
	writeCorpus(t, dir, "svc/svc.go", gateSrcAfter)

	first, _, _ := runHook(t, "stop", payload)
	if first.HookSpecificOutput.AdditionalContext == "" {
		t.Fatal("the first stop must produce an agent note")
	}

	base, err := readBaseline(baseline)
	if err != nil {
		t.Fatalf("read baseline: %v", err)
	}
	var sawClass, sawCreated bool
	for _, k := range base.Reported {
		if strings.HasPrefix(k, "class:") {
			sawClass = true
		}
		if strings.HasPrefix(k, "created:") {
			sawCreated = true
		}
	}
	if !sawClass || !sawCreated {
		t.Errorf("identity findings were not ledgered: %v", base.Reported)
	}

	// The second turn measures the same cumulative delta. The user digest is
	// unchanged — it is a statement of where the session stands, not a feed —
	// and the agent note does not repeat what the ledger already retired.
	second, _, _ := runHook(t, "stop", payload)
	if second.SystemMessage != first.SystemMessage {
		t.Errorf("the user digest is not stable for an unchanged tree:\n%q\n%q",
			first.SystemMessage, second.SystemMessage)
	}
	if note := second.HookSpecificOutput.AdditionalContext; note != "" {
		t.Errorf("the second turn re-reported findings the ledger holds:\n%s", note)
	}
}

// A hook must never break a session over a measurement, so the identity pass
// degrades to no section rather than to an error or a panic. A corrupt label
// dictionary is the reachable failure: everything else identity refuses on,
// snapshot.Diff refuses on first.
func TestIdentityDeltaDegradesOnACorruptSnapshot(t *testing.T) {
	corrupt := snapshot.Snapshot{
		Schema:  snapshot.Schema,
		RuleSet: "x",
		Labels:  "not base64 at all !!!",
		Units:   []snapshot.Unit{{Key: "a.One", Digest: "d", WL: "AAAA"}},
	}
	got := identityDelta(corrupt, corrupt)
	if got.Comparable || len(got.Changes) != 0 {
		t.Errorf("a corrupt snapshot must yield no delta, got %+v", got)
	}
	// And the digest built from it says nothing rather than half a sentence.
	if len(got.Created) != 0 || len(got.Dissolved) != 0 {
		t.Error("a refused identity delta must carry no pair changes")
	}
	var _ identity.Delta = got
}

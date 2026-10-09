package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/reporter"
)

// execHookView drives `hook view` the way the plugin's mod does: a hook-shaped
// payload on stdin, the view (or nothing) on stdout.
func execHookView(t *testing.T, payload string) (reporter.SessionView, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	rootCmd.SetIn(strings.NewReader(payload))
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errb)
	rootCmd.SetArgs([]string{"hook", "view"})
	t.Cleanup(func() {
		rootCmd.SetIn(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook view returned an error: %v", err)
	}
	var v reporter.SessionView
	if out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &v); err != nil {
			t.Fatalf("hook view stdout is not valid JSON: %q", out.String())
		}
	}
	return v, out.String(), errb.String()
}

// TestHookViewShowsWhatStopRecorded is the mod's data path end to end: the
// Stop hook measures a turn and records the user-side view in its impact
// report, and `hook view` hands that view back without analyzing anything.
//
// The pane's report must be the delta report as `doppel diff` prints it — the
// identity classes and the created pair with its stored explanation — because
// the mod draws it verbatim and renders none of it itself.
func TestHookViewShowsWhatStopRecorded(t *testing.T) {
	dir := t.TempDir()
	writeCorpus(t, dir, "svc/svc.go", gateSrcBefore)

	sessionID := "doppel-view-" + filepath.Base(dir)
	baseline := baselinePath(sessionID)
	t.Cleanup(func() {
		os.Remove(baseline)
		os.Remove(deltaPathFor(baseline))
	})

	payload := hookPayload(t, sessionID, dir)
	runHook(t, "session-start", payload)

	// Before any Stop there is nothing to show, and nothing is said.
	if _, out, errOut := execHookView(t, payload); out != "" || errOut != "" {
		t.Fatalf("view before any stop: stdout %q, stderr %q; want both empty", out, errOut)
	}

	writeCorpus(t, dir, "svc/svc.go", gateSrcAfter)
	runHook(t, "stop", payload)

	v, _, errOut := execHookView(t, payload)
	if errOut != "" {
		t.Fatalf("hook view wrote to stderr: %q", errOut)
	}
	if !strings.HasPrefix(v.Band, "doppel: ") || strings.Contains(v.Band, "\n") {
		t.Errorf("band must be one line led by the tool's name, got %q", v.Band)
	}
	for _, want := range []string{"renamed 1", "pairs created ", " — since session start"} {
		if !strings.Contains(v.Band, want) {
			t.Errorf("band %q is missing %q", v.Band, want)
		}
	}
	for _, want := range []string{
		"Delta since the baseline",
		"svc.Total",
		"svc.Sum",
		"svc.Clip <-> svc.Trim",
		"explain: identical after rename",
	} {
		if !strings.Contains(v.Report, want) {
			t.Errorf("report is missing %q:\n%s", want, v.Report)
		}
	}

	// The impact report keeps every key it carried before the view joined it:
	// the delta is flattened, not nested, so a reader of the old shape is
	// unaffected.
	data, err := os.ReadFile(deltaPathFor(baseline))
	if err != nil {
		t.Fatalf("read impact report: %v", err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"comparable", "functionsBefore", "pairsAdded", "view"} {
		if _, ok := top[key]; !ok {
			t.Errorf("impact report lost top-level key %q", key)
		}
	}

	// The session undoes itself: the Stop hook says nothing, and the band must
	// not go on describing a delta the session no longer has.
	writeCorpus(t, dir, "svc/svc.go", gateSrcBefore)
	runHook(t, "stop", payload)
	if _, out, _ := execHookView(t, payload); out != "" {
		t.Errorf("view after the session reverted itself: %q; want nothing", out)
	}
}

// hook-notify: off silences the band as well as the transcript. Off was a
// statement about this measurement, not about which surface shows it.
func TestHookViewSilentUnderNotifyOff(t *testing.T) {
	dir := t.TempDir()
	writeCorpus(t, dir, "svc/svc.go", gateSrcBefore)

	sessionID := "doppel-view-off-" + filepath.Base(dir)
	baseline := baselinePath(sessionID)
	t.Cleanup(func() {
		os.Remove(baseline)
		os.Remove(deltaPathFor(baseline))
	})

	payload := hookPayload(t, sessionID, dir)
	runHook(t, "session-start", payload)
	writeCorpus(t, dir, "svc/svc.go", gateSrcAfter)
	runHook(t, "stop", payload)
	if v, _, _ := execHookView(t, payload); v.Band == "" {
		t.Fatal("precondition: the band should show before notify is turned off")
	}

	writeCorpus(t, dir, ".doppel.json", `{"hook-notify": "off"}`)
	resp, _, _ := runHook(t, "stop", payload)
	if resp.SystemMessage != "" {
		t.Errorf("stop spoke under hook-notify off: %q", resp.SystemMessage)
	}
	if _, out, _ := execHookView(t, payload); out != "" {
		t.Errorf("view under hook-notify off: %q; want nothing", out)
	}
}

// Same contract as every hook subcommand: whatever arrives, exit zero, write
// nothing to stderr, and print either nothing or valid JSON.
func TestHookViewFailsSilently(t *testing.T) {
	corrupt := "doppel-view-corrupt"
	t.Cleanup(func() { os.Remove(deltaPathFor(baselinePath(corrupt))) })
	if err := os.MkdirAll(baselineDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(deltaPathFor(baselinePath(corrupt)), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, payload := range map[string]string{
		"malformed":          "{not json",
		"empty":              "",
		"empty object":       `{}`,
		"unknown session":    `{"session_id":"no-such-session-at-all"}`,
		"corrupt report":     `{"session_id":"` + corrupt + `"}`,
		"hostile session":    `{"session_id":"../../../etc/passwd"}`,
		"report has no view": `{"session_id":"doppel-view-noview"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if name == "report has no view" {
				p := deltaPathFor(baselinePath("doppel-view-noview"))
				t.Cleanup(func() { os.Remove(p) })
				// The shape a pre-view build wrote: a bare snapshot.Delta.
				if err := os.WriteFile(p, []byte(`{"comparable":true}`), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			_, out, errOut := execHookView(t, payload)
			if errOut != "" {
				t.Errorf("stderr: %q", errOut)
			}
			if out != "" {
				t.Errorf("stdout: %q; want nothing", out)
			}
		})
	}
}

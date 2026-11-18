package cmd

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

func scopeSnap() snapshot.Snapshot {
	return snapshot.Snapshot{
		Units: []snapshot.Unit{
			{Key: "billing.Post", Package: "billing", File: "svc/internal/billing/service.go"},
			{Key: "shipping.Get", Package: "shipping", File: "svc/internal/shipping/client.go"},
			{Key: "culture.Build", Package: "culture", File: "internal/culture/culture.go"},
		},
	}
}

func TestScopedPackagesMatchesPaths(t *testing.T) {
	s := scopeSnap()
	tests := []struct {
		name   string
		prompt string
		want   []string // package names, in order
	}{
		{"at-mention path", "look at @svc/internal/billing please", []string{"billing"}},
		{"path with trailing comma", "fix svc/internal/billing, then test", []string{"billing"}},
		{"deep-only suffix", "check internal/billing for dupes", []string{"billing"}},
		{"file inside the package", "the bug is in svc/internal/billing/service.go", []string{"billing"}},
		{"backslash path", `open svc\internal\shipping\client.go`, []string{"shipping"}},
		{"punctuation-wrapped path", "is `internal/culture` still duplicated?", []string{"culture"}},
		{"prose only", "make the tests faster", nil},
		{"unknown package", "look at stripe/webhooks", nil},
		{"two mentions keep order", "compare internal/culture with svc/internal/shipping", []string{"culture", "shipping"}},
		{"duplicate mention collapses", "svc/internal/billing and again svc/internal/billing", []string{"billing"}},
		// A package name is an ordinary word, and the hook fired on exactly
		// these across a month of sessions ("git", "main") without one note
		// being acted on. Only a path names a place in the tree.
		{"bare package name is prose", "the billing integration is broken", nil},
		{"punctuation-wrapped bare name is prose", "is `billing` still duplicated?", nil},
		{"url is not a mention", "see https://example.com/svc/internal/billing", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scopedPackages(tc.prompt, s)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d packages %v, want %v", len(got), got, tc.want)
			}
			for i, w := range tc.want {
				if got[i].Package != w {
					t.Errorf("position %d: %q, want %q (full: %v)", i, got[i].Package, w, got)
				}
			}
		})
	}
}

// A prompt naming many packages is a survey, not a target: the cap keeps the
// digest from becoming the corpus dump it exists to replace.
func TestScopedPackagesCapsAtThree(t *testing.T) {
	s := snapshot.Snapshot{Units: []snapshot.Unit{
		{Key: "a.F", Package: "a", File: "a/f.go"},
		{Key: "b.F", Package: "b", File: "b/f.go"},
		{Key: "c.F", Package: "c", File: "c/f.go"},
		{Key: "d.F", Package: "d", File: "d/f.go"},
	}}
	got := scopedPackages("a/ b/ c/ d/", s)
	if len(got) != scopeMaxPackages {
		t.Fatalf("got %d packages, want %d", len(got), scopeMaxPackages)
	}
	for i, w := range []string{"a", "b", "c"} {
		if got[i].Package != w {
			t.Errorf("position %d: %q, want %q", i, got[i].Package, w)
		}
	}
}

// UserPromptSubmit fires on text the harness delivers as well as on what the
// user typed: a background task finishing, a subagent's report, another
// session's message, a system reminder. Paths inside those name the reported
// work, not a target, so only the typed text is read.
func TestScopedPackagesIgnoresHarnessEnvelopes(t *testing.T) {
	s := scopeSnap()
	tests := []struct {
		name   string
		prompt string
		want   []string
	}{
		{"task notification", "<task-notification>\n<task-id>a1</task-id>\n<summary>wrote svc/internal/billing/service.go</summary>\n</task-notification>", nil},
		{"another session's message", "Another Claude session sent a message:\n<agent-message from=\"a1\">\nI changed internal/culture and svc/internal/shipping.\n</agent-message>", nil},
		{"cross-session message", `<cross-session-message from="local_1">see internal/culture</cross-session-message>`, nil},
		{"system reminder", "<system-reminder>\nLinked sessions now: svc/internal/billing\n</system-reminder>", nil},
		{"unclosed envelope runs to the end", "<task-notification>\n<summary>svc/internal/billing", nil},
		{"typed text beside an envelope still counts", "<system-reminder>internal/culture</system-reminder>\nnow look at svc/internal/shipping", []string{"shipping"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scopedPackages(tc.prompt, s)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d packages %v, want %v", len(got), got, tc.want)
			}
			for i, w := range tc.want {
				if got[i].Package != w {
					t.Errorf("position %d: %q, want %q (full: %v)", i, got[i].Package, w, got)
				}
			}
		})
	}
}

// The hook decides whether to run an analysis at all from pathMentions alone,
// so a prompt with nothing path-shaped in its typed text must yield none.
func TestPathMentionsGatesTheAnalysis(t *testing.T) {
	for _, prompt := range []string{
		"",
		"make the git hook faster in main",
		"<task-notification><summary>internal/culture done</summary></task-notification>",
		"read https://example.com/docs/page",
	} {
		if got := pathMentions(prompt); len(got) != 0 {
			t.Errorf("pathMentions(%q) = %v, want none", prompt, got)
		}
	}
	got := pathMentions("check @internal/culture, and cmd\\hook.go")
	if len(got) != 2 || got[0].norm != "internal/culture" || got[1].norm != "cmd/hook.go" {
		t.Errorf("pathMentions = %v, want internal/culture then cmd/hook.go", got)
	}
}

// A shallow mention matching several packages must resolve them in sorted
// order, not map order: which packages win the cap has to be the same on
// every run of the same prompt.
func TestScopedPackagesMultiMatchIsDeterministic(t *testing.T) {
	s := snapshot.Snapshot{Units: []snapshot.Unit{
		{Key: "zeta.F", Package: "zeta", File: "svc/zeta/f.go"},
		{Key: "alpha.F", Package: "alpha", File: "svc/alpha/f.go"},
		{Key: "mid.F", Package: "mid", File: "svc/mid/f.go"},
	}}
	first := scopedPackages("everything under svc/ needs review", s)
	if len(first) != 3 {
		t.Fatalf("got %d packages, want 3: %v", len(first), first)
	}
	for i, w := range []string{"alpha", "mid", "zeta"} {
		if first[i].Package != w {
			t.Fatalf("order %v, want alphabetical alpha,mid,zeta", first)
		}
	}
	for run := 0; run < 5; run++ {
		again := scopedPackages("everything under svc/ needs review", s)
		for i := range first {
			if again[i].Package != first[i].Package {
				t.Fatalf("run %d: order changed: %v vs %v", run, again, first)
			}
		}
	}
}

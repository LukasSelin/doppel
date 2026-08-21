package cmd

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

func scopeSnap() snapshot.Snapshot {
	return snapshot.Snapshot{
		Units: []snapshot.Unit{
			{Key: "hubspot.Post", Package: "hubspot", File: "backend/internal/hubspot/service.go"},
			{Key: "aws.Get", Package: "aws", File: "backend/internal/aws/client.go"},
			{Key: "culture.Build", Package: "culture", File: "internal/culture/culture.go"},
		},
	}
}

func TestScopedPackagesMatchesPathsAndBareNames(t *testing.T) {
	s := scopeSnap()
	tests := []struct {
		name   string
		prompt string
		want   []string // package names, in order
	}{
		{"at-mention path", "look at @backend/internal/hubspot please", []string{"hubspot"}},
		{"path with trailing comma", "fix backend/internal/hubspot, then test", []string{"hubspot"}},
		{"bare package name", "the hubspot integration is broken", []string{"hubspot"}},
		{"deep-only suffix", "check internal/hubspot for dupes", []string{"hubspot"}},
		{"backslash path", `open backend\internal\aws\client.go`, []string{"aws"}},
		{"prose only", "make the tests faster", nil},
		{"unknown package", "look at stripe/webhooks", nil},
		{"two mentions keep order", "compare internal/culture with backend/internal/aws", []string{"culture", "aws"}},
		{"duplicate mention collapses", "hubspot hubspot hubspot", []string{"hubspot"}},
		{"punctuation-wrapped", "is `hubspot` still duplicated?", []string{"hubspot"}},
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
	got := scopedPackages("a b c d", s)
	if len(got) != scopeMaxPackages {
		t.Fatalf("got %d packages, want %d", len(got), scopeMaxPackages)
	}
	for i, w := range []string{"a", "b", "c"} {
		if got[i].Package != w {
			t.Errorf("position %d: %q, want %q", i, got[i].Package, w)
		}
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

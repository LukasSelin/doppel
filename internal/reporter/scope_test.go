package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

func scopeFixture() snapshot.Snapshot {
	return snapshot.Snapshot{
		Units: []snapshot.Unit{
			{Key: "hubspot.PostClaim", Package: "hubspot", File: "internal/hubspot/service.go"},
			{Key: "hubspot.PostLost", Package: "hubspot", File: "internal/hubspot/service.go"},
			{Key: "hubspot.PostInbox", Package: "hubspot", File: "internal/hubspot/inbox.go"},
			{Key: "aws.Send", Package: "aws", File: "internal/aws/send.go"},
			{Key: "quiet.F", Package: "quiet", File: "internal/quiet/f.go"},
		},
		Pairs: []snapshot.Pair{
			{A: "hubspot.PostClaim", B: "hubspot.PostLost", Score: 1.00, Overlap: 0.86, MergeWorthy: true},
			{A: "aws.Send", B: "hubspot.PostInbox", Score: 0.90, Overlap: 0.55, MergeWorthy: true},
			{A: "hubspot.PostClaim", B: "hubspot.PostInbox", Score: 0.70, Overlap: 0.30, MergeWorthy: false},
		},
	}
}

func TestScopeDigestReportsWithinAndCrossPairs(t *testing.T) {
	s := scopeFixture()
	got := ScopeDigest(s, []ScopedPackage{{Package: "hubspot", Mention: "internal/hubspot"}})

	for _, want := range []string{
		"doppel: internal/hubspot (3 functions)",
		"hubspot.PostClaim <-> hubspot.PostLost  shape 1.00  overlap 0.86",
		"1 merge-worthy pair connects this package to others.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("digest missing %q:\n%s", want, got)
		}
	}
	// The non-merge-worthy pair is below the bar everywhere; it must not
	// appear just because the package is in scope.
	if strings.Contains(got, "PostClaim <-> hubspot.PostInbox") {
		t.Errorf("non-merge-worthy pair leaked into the digest:\n%s", got)
	}
}

// A mentioned package with no duplication findings earns no lines at all: a
// header followed by nothing reads as "something to see here" when there is
// nothing.
func TestScopeDigestSilentForCleanPackage(t *testing.T) {
	s := scopeFixture()
	if got := ScopeDigest(s, []ScopedPackage{{Package: "quiet", Mention: "quiet"}}); got != "" {
		t.Errorf("clean package produced output:\n%s", got)
	}
	if got := ScopeDigest(s, []ScopedPackage{{Package: "ghost", Mention: "ghost"}}); got != "" {
		t.Errorf("unknown package produced output:\n%s", got)
	}
}

func TestAdviceDigestScopesToTheFile(t *testing.T) {
	s := scopeFixture()
	got := AdviceDigest(s, "internal/hubspot/service.go")

	if !strings.Contains(got, "as of session start") {
		t.Errorf("advisory does not label its facts' age:\n%s", got)
	}
	if !strings.Contains(got, "hubspot.PostClaim <-> hubspot.PostLost") {
		t.Errorf("advisory missing the file's own twin pair:\n%s", got)
	}
	// inbox.go's cross pair involves no function in service.go, and the
	// non-merge-worthy pair is below the bar.
	if strings.Contains(got, "aws.Send") {
		t.Errorf("pair from another file leaked in:\n%s", got)
	}
}

func TestAdviceDigestSilentWhenNothingToSay(t *testing.T) {
	s := scopeFixture()
	if got := AdviceDigest(s, "internal/quiet/f.go"); got != "" {
		t.Errorf("file with no merge-worthy pairs produced output:\n%s", got)
	}
	if got := AdviceDigest(s, "internal/unknown/x.go"); got != "" {
		t.Errorf("unknown file produced output:\n%s", got)
	}
}

package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

func scopeFixture() snapshot.Snapshot {
	return snapshot.Snapshot{
		Units: []snapshot.Unit{
			{Key: "billing.PostClaim", Package: "billing", File: "internal/billing/service.go"},
			{Key: "billing.PostLost", Package: "billing", File: "internal/billing/service.go"},
			{Key: "billing.PostInbox", Package: "billing", File: "internal/billing/inbox.go"},
			{Key: "shipping.Send", Package: "shipping", File: "internal/shipping/send.go"},
			{Key: "quiet.F", Package: "quiet", File: "internal/quiet/f.go"},
		},
		Pairs: []snapshot.Pair{
			{A: "billing.PostClaim", B: "billing.PostLost", Score: 1.00, Overlap: 0.86, MergeWorthy: true},
			{A: "shipping.Send", B: "billing.PostInbox", Score: 0.90, Overlap: 0.55, MergeWorthy: true},
			{A: "billing.PostClaim", B: "billing.PostInbox", Score: 0.70, Overlap: 0.30, MergeWorthy: false},
		},
	}
}

func TestScopeDigestReportsWithinAndCrossPairs(t *testing.T) {
	s := scopeFixture()
	got := ScopeDigest(s, []ScopedPackage{{Package: "billing", Mention: "internal/billing"}})

	for _, want := range []string{
		"doppel: internal/billing (3 functions)",
		"billing.PostClaim <-> billing.PostLost  shape 1.00  overlap 0.86",
		"1 merge-worthy pair connects this package to others.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("digest missing %q:\n%s", want, got)
		}
	}
	// The non-merge-worthy pair is below the bar everywhere; it must not
	// appear just because the package is in scope.
	if strings.Contains(got, "PostClaim <-> billing.PostInbox") {
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
	got := AdviceDigest(s, "internal/billing/service.go")

	if !strings.Contains(got, "as of session start") {
		t.Errorf("advisory does not label its facts' age:\n%s", got)
	}
	if !strings.Contains(got, "billing.PostClaim <-> billing.PostLost") {
		t.Errorf("advisory missing the file's own twin pair:\n%s", got)
	}
	// inbox.go's cross pair involves no function in service.go, and the
	// non-merge-worthy pair is below the bar.
	if strings.Contains(got, "shipping.Send") {
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

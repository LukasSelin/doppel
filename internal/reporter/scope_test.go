package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
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

	// One line, counts only: the non-merge-worthy pair is below the bar and
	// is not counted, and no pair is listed — a month of sessions acted on
	// none of the listed ones.
	want := "doppel: internal/billing (3 functions): 1 merge-worthy pair within the package and 1 connecting it to others.\n"
	if got != want {
		t.Errorf("digest:\n%q\nwant:\n%q", got, want)
	}
}

// Each half of the line stands alone when the other is zero.
func TestScopeDigestNamesOnlyTheNonZeroCount(t *testing.T) {
	s := scopeFixture()
	got := ScopeDigest(s, []ScopedPackage{{Package: "shipping", Mention: "internal/shipping"}})
	want := "doppel: internal/shipping (1 function): 1 merge-worthy pair connecting it to others.\n"
	if got != want {
		t.Errorf("digest:\n%q\nwant:\n%q", got, want)
	}

	s.Pairs = s.Pairs[:1]
	got = ScopeDigest(s, []ScopedPackage{{Package: "billing", Mention: "internal/billing"}})
	want = "doppel: internal/billing (3 functions): 1 merge-worthy pair within the package.\n"
	if got != want {
		t.Errorf("digest:\n%q\nwant:\n%q", got, want)
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

// A pair whose kind says it is not a merge — here two wrappers already
// delegating to one helper — is left out of the advisory, and a kind that is
// still a merge is named on the line.
func TestAdviceDigestLeavesOutNonMergeKinds(t *testing.T) {
	s := scopeFixture()
	s.Pairs[0].Kind = analyzer.KindThinWrappers
	if got := AdviceDigest(s, "internal/billing/service.go"); got != "" {
		t.Errorf("thin-wrapper pair reached the advisory:\n%s", got)
	}
	s.Pairs[0].Kind = analyzer.KindFork
	if got := AdviceDigest(s, "internal/billing/service.go"); !strings.Contains(got, "[diverged copy]") {
		t.Errorf("advisory does not name the pair's kind:\n%s", got)
	}
}

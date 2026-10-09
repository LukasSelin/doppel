package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/identity"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// The band is silent exactly when the user digest is, so the two user-side
// surfaces can never disagree about whether the session did anything.
func TestSessionViewSilentWhenDigestIs(t *testing.T) {
	empty := snapshot.Delta{Comparable: true}
	if _, ok := SessionViewOf(identity.Delta{}, empty); ok {
		t.Error("an empty delta must produce no view")
	}
	if SessionDigest(identity.Delta{}, empty, "") != "" {
		t.Fatal("precondition: the digest is silent for an empty delta")
	}
	incomparable := snapshot.Delta{Comparable: false, Reason: "schema"}
	if _, ok := SessionViewOf(identity.Delta{}, incomparable); ok {
		t.Error("an incomparable delta must produce no view")
	}
}

// With the identity pass degraded to its zero value the view falls back to the
// impact half rather than to a "not comparable" line for a comparison that
// snapshot.Diff did make.
func TestSessionViewFallsBackToImpact(t *testing.T) {
	d := snapshot.Delta{
		Comparable:      true,
		FunctionsBefore: 2,
		FunctionsAfter:  3,
		UnitsAdded:      []snapshot.Unit{{Key: "a.New"}},
	}
	v, ok := SessionViewOf(identity.Delta{}, d)
	if !ok {
		t.Fatal("a non-empty delta must produce a view")
	}
	if !strings.HasPrefix(v.Band, "doppel: functions 2 -> 3") || strings.Contains(v.Band, "\n") {
		t.Errorf("band = %q", v.Band)
	}
	if strings.Contains(v.Report, "Not comparable") {
		t.Errorf("fallback report must not claim incomparability:\n%s", v.Report)
	}
}

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

// The overview is the pane's first screen: merge-worthy pairs one line each,
// changes one line each without evidence, every other pair a count — and every
// list bounded with what it left out.
func TestSessionOverviewIsBounded(t *testing.T) {
	mem := func(key, file string, line int) identity.Member {
		return identity.Member{Function: identity.Function{Key: key, File: file, Line: line}}
	}
	id := identity.Delta{
		Result: identity.Result{
			Comparable:   true,
			OldFunctions: 3,
			NewFunctions: 4,
			Counts: []identity.ClassCount{
				{Class: identity.Renamed, Count: 1}, {Class: identity.Added, Count: 1}, {Class: identity.Unchanged, Count: 2},
			},
			Changes: []identity.Change{
				{Class: identity.Renamed, Old: []identity.Member{mem("svc.Total", "svc/svc.go", 9)}, New: []identity.Member{mem("svc.Sum", "svc/svc.go", 9)}, Jaccard: 1, Containment: 1, DigestEqual: true},
				{Class: identity.Added, New: []identity.Member{mem("svc.Trim", "svc/svc.go", 49)}},
			},
		},
		Created: []identity.PairChange{
			{A: "svc.Clip", B: "svc.Trim", Score: 1, Overlap: 0.52, MergeWorthy: true, Explain: "identical after rename", BClass: identity.Added},
			{A: "svc.Clip", B: "svc.Sum", Score: 0.33, Overlap: 0.37, Explain: "differs", BClass: identity.Renamed},
			{A: "x.A", B: "x.B", Score: 0.5, Overlap: 0.4, AClass: identity.Unchanged, BClass: identity.Unchanged},
		},
		Dissolved: []identity.PairChange{
			{A: "svc.Clip", B: "svc.Total", Score: 0.33, Overlap: 0.37, BClass: identity.Renamed},
		},
	}
	got := sessionOverview(id)
	want := `renamed 1, new 1; functions 3 -> 4

merge-worthy pairs created 1
  svc.Clip <-> svc.Trim  shape 1.00  (svc.Trim new)

functions changed 2
  renamed  svc.Total -> svc.Sum
  new      svc.Trim  svc/svc.go:49

other pairs from these changes: 1 created, 1 dissolved (below merge-worthy)
pairs no change explains: 1 created, 0 dissolved (retrieval re-ranking)
`
	if got != want {
		t.Errorf("overview:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "explain:") || strings.Contains(got, "jaccard") {
		t.Error("evidence lines belong to the full report")
	}

	for i := 0; i < 12; i++ {
		id.Changes = append(id.Changes, identity.Change{Class: identity.Edited,
			Old: []identity.Member{mem("e.F", "e.go", i)}, New: []identity.Member{mem("e.F", "e.go", i)}})
	}
	if got := sessionOverview(id); !strings.Contains(got, "  … 6 more in the full report\n") {
		t.Errorf("a long change list must be capped and say so:\n%s", got)
	}
}

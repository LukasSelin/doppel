package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/identity"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// sampleDelta is one rename plus one copied function, the shape the gate
// fixture produces: exactly what a session that renames and copies looks like
// to the Stop hook.
func sampleDelta() identity.Delta {
	changes := []identity.Change{
		{
			Class:       identity.Renamed,
			Old:         []identity.Member{{Function: identity.Function{Key: "svc.Total", File: "svc/svc.go", Line: 9}}},
			New:         []identity.Member{{Function: identity.Function{Key: "svc.Sum", File: "svc/svc.go", Line: 9}}},
			Jaccard:     1,
			Containment: 1,
			DigestEqual: true,
			NameChanged: true,
		},
		{
			Class: identity.Added,
			New:   []identity.Member{{Function: identity.Function{Key: "svc.Trim", File: "svc/svc.go", Line: 50}}},
		},
		{
			Class: identity.Unchanged,
			Old:   []identity.Member{{Function: identity.Function{Key: "svc.Clip"}}},
			New:   []identity.Member{{Function: identity.Function{Key: "svc.Clip"}}},
		},
	}
	counts := []identity.ClassCount{
		{Class: identity.Split}, {Class: identity.Merged}, {Class: identity.Moved},
		{Class: identity.Renamed, Count: 1}, {Class: identity.Edited},
		{Class: identity.Added, Count: 1}, {Class: identity.Deleted},
		{Class: identity.Unchanged, Count: 1},
	}
	return identity.Delta{
		Result: identity.Result{
			Comparable: true, OldFunctions: 3, NewFunctions: 4,
			Counts: counts, Changes: changes,
		},
		Created: []identity.PairChange{{
			A: "svc.Clip", B: "svc.Trim", Score: 1, Overlap: 0.52, MergeWorthy: true,
			Explain: "identical after rename",
			AClass:  identity.Unchanged, BClass: identity.Added,
		}},
		Dissolved: []identity.PairChange{{
			A: "svc.Clip", B: "svc.Total", Score: 0.33, Overlap: 0.37,
			Explain: "differs by three extra assign",
			AClass:  identity.Unchanged, BClass: identity.Renamed,
		}},
	}
}

func TestDeltaSectionLeadsWithTheCensusAndCarriesExplain(t *testing.T) {
	got := DeltaSection(sampleDelta(), maxDeltaChanges, maxDeltaPairs)

	if !strings.HasPrefix(got, "doppel delta since the session baseline: renamed 1, new 1; pairs created 1, dissolved 1.\n") {
		t.Fatalf("census line wrong:\n%s", got)
	}
	for _, want := range []string{
		"svc.Total (svc/svc.go:9) -> svc.Sum (svc/svc.go:9)",
		"svc.Trim (svc/svc.go:50)",
		"NEW  svc.Clip <-> svc.Trim  shape 1.00  overlap 0.52  (merge-worthy)",
		"explain: identical after rename",
		"GONE svc.Clip <-> svc.Total",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("section is missing %q:\n%s", want, got)
		}
	}
	// unchanged is the bulk of any nearby comparison and says nothing about
	// the session; it is counted nowhere in this section and listed nowhere.
	if strings.Contains(got, "unchanged") {
		t.Errorf("the section must not list or count unchanged functions:\n%s", got)
	}
}

func TestDeltaSectionIsSilentWhenNothingHappened(t *testing.T) {
	empty := identity.Delta{Result: identity.Result{
		Comparable: true,
		Counts:     []identity.ClassCount{{Class: identity.Unchanged, Count: 9}},
	}}
	if got := DeltaSection(empty, maxDeltaChanges, maxDeltaPairs); got != "" {
		t.Errorf("an empty delta must render nothing, got %q", got)
	}
	refused := identity.Delta{Result: identity.Result{Comparable: false, Reason: "schema"}}
	if got := DeltaSection(refused, maxDeltaChanges, maxDeltaPairs); got != "" {
		t.Errorf("a refused delta must render nothing, got %q", got)
	}
}

// The section is bounded, and it says how much it left out rather than
// silently truncating.
func TestDeltaSectionIsBounded(t *testing.T) {
	d := sampleDelta()
	for i := 0; i < 5; i++ {
		d.Created = append(d.Created, identity.PairChange{
			A: "svc.A", B: "svc.B", AClass: identity.Added,
		})
	}
	got := DeltaSection(d, 1, 2)
	if !strings.Contains(got, "(1 more classified functions, not listed)") {
		t.Errorf("the change list must report what it omitted:\n%s", got)
	}
	if !strings.Contains(got, "(4 more, not listed)") {
		t.Errorf("the pair list must report what it omitted:\n%s", got)
	}
}

// The line the bar is drawn at: the identity classification never buys a model
// turn on its own. Notable alone decides whether the note is sent.
func TestAgentDeltaDigestNeedsANotableFinding(t *testing.T) {
	note, shown := AgentDeltaDigest(sampleDelta(), nil)
	if note != "" || shown != nil {
		t.Errorf("a rename with no notable finding must not continue the turn: %q", note)
	}
}

// The note says which side of each pair the session wrote, in place, instead
// of leading with the delta report: that preamble was most of the text Stop
// notes cost an agent, and none of what it listed was acted on.
func TestAgentDeltaDigestLabelsEachSide(t *testing.T) {
	notable := []Finding{{
		Key:  "new:svc.Clip|svc.Trim",
		Line: "svc.Clip <-> svc.Trim  shape 1.00  overlap 0.52",
		A:    "svc.Clip", B: "svc.Trim",
	}}
	note, shown := AgentDeltaDigest(sampleDelta(), notable)

	if !strings.Contains(note, "  svc.Clip [unchanged] <-> svc.Trim [new]  shape 1.00  overlap 0.52\n") {
		t.Errorf("the pair line must label both sides:\n%s", note)
	}
	if strings.Contains(note, "doppel matched this session's functions") || strings.Contains(note, "renamed 1") {
		t.Errorf("the delta report must not reach the agent note:\n%s", note)
	}
	if !strings.HasPrefix(note, "doppel measured this session's effect") {
		t.Errorf("the note must open with the findings:\n%s", note)
	}

	// The ledger keeps the findings as Notable made them: keys are what
	// suppress a repeat, and a relabelled line must not become a new finding.
	if len(shown) != 1 || shown[0].Key != notable[0].Key || shown[0].Line != notable[0].Line {
		t.Errorf("shown = %+v, want the notable finding unchanged", shown)
	}
}

// A finding about something other than one pair, or one whose sides the delta
// cannot place, is said exactly as Notable wrote it.
func TestAgentDeltaDigestLeavesOtherLinesAlone(t *testing.T) {
	notable := []Finding{
		{Key: "new:a|b", Line: "a <-> b  shape 1.00  overlap 0.52"},
		{Key: "drift:x|y", Line: "x <-> y  shape 0.40 -> 0.70", A: "y", B: "x"},
	}
	note, _ := AgentDeltaDigest(sampleDelta(), notable)
	for _, want := range []string{"  a <-> b  shape 1.00", "  x <-> y  shape 0.40 -> 0.70"} {
		if !strings.Contains(note, want) {
			t.Errorf("note lost %q:\n%s", want, note)
		}
	}

	incomparable := sampleDelta()
	incomparable.Comparable = false
	labelled := []Finding{{Key: "new:svc.Clip|svc.Trim", Line: "svc.Clip <-> svc.Trim  shape 1.00", A: "svc.Clip", B: "svc.Trim"}}
	got, _ := AgentDeltaDigest(incomparable, labelled)
	want, _ := AgentDigest(labelled)
	if got != want {
		t.Errorf("an incomparable delta must leave the plain note:\n%q\nwant\n%q", got, want)
	}
}

// SessionDigest is silent exactly when ImpactDigest is: an identity finding
// always implies a snapshot delta, so the impact half is the stricter test.
func TestSessionDigestFollowsTheImpactHalfsSilence(t *testing.T) {
	nothing := snapshot.Delta{Comparable: true}
	if got := SessionDigest(sampleDelta(), nothing, ""); got != "" {
		t.Errorf("an empty impact delta must silence the whole digest, got %q", got)
	}

	moved := snapshot.Delta{
		Comparable:     true,
		FunctionsAfter: 4, FunctionsBefore: 3,
		UnitsAdded: []snapshot.Unit{{Key: "svc.Trim"}},
	}
	got := SessionDigest(sampleDelta(), moved, "")
	if !strings.HasPrefix(got, "doppel delta since the session baseline:") {
		t.Fatalf("the delta report must lead the user digest:\n%s", got)
	}
	if !strings.Contains(got, "doppel impact this session:") {
		t.Errorf("the impact half must follow it:\n%s", got)
	}
	if len(got) > digestMaxChars+len("\n  (truncated)\n") {
		t.Errorf("the composed digest exceeded its budget: %d chars", len(got))
	}
}

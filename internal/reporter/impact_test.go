package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

// The Stop hook runs at the end of every turn. An empty digest is what keeps it
// silent, and a "nothing changed" line repeated after each turn would train the
// reader to skip the place real findings appear.
func TestImpactDigestEmptyDeltaRendersNothing(t *testing.T) {
	d := snapshot.Delta{Comparable: true, FunctionsBefore: 10, FunctionsAfter: 10}
	if got := ImpactDigest(d, "/tmp/x.json"); got != "" {
		t.Errorf("ImpactDigest of an empty delta = %q, want the empty string", got)
	}
}

func TestImpactDigestIncomparableExplainsItself(t *testing.T) {
	d := snapshot.Delta{Comparable: false, Reason: "baseline used ontology 1.0.0, current 2.0.0"}
	got := ImpactDigest(d, "")
	if !strings.Contains(got, "not comparable") || !strings.Contains(got, "ontology") {
		t.Errorf("digest did not explain incomparability: %q", got)
	}
}

func TestImpactDigestLeadsWithAttributableChanges(t *testing.T) {
	d := snapshot.Delta{
		Comparable:      true,
		FunctionsBefore: 10, FunctionsAfter: 11,
		PairsBefore: 5, PairsAfter: 7,
		MergeBefore: 1, MergeAfter: 2,
		UnitsAdded: []snapshot.Unit{{Key: "p.New", Digest: "d"}},
		PairsAdded: []snapshot.PairChange{
			{Pair: snapshot.Pair{A: "p.New", B: "q.Old", Score: 0.9, Overlap: 0.5, MergeWorthy: true}, Attributable: true},
			{Pair: snapshot.Pair{A: "x.Idle", B: "y.Idle", Score: 0.95}, Attributable: false},
		},
	}
	got := ImpactDigest(d, "/tmp/delta.json")

	if !strings.Contains(got, "NEW  p.New <-> q.Old") {
		t.Errorf("attributable pair missing from digest:\n%s", got)
	}
	if strings.Contains(got, "x.Idle") {
		t.Errorf("non-attributable pair should be counted, not listed:\n%s", got)
	}
	if !strings.Contains(got, "1 further pair changes involve no function edited") {
		t.Errorf("non-attributable churn not counted:\n%s", got)
	}
	if !strings.Contains(got, "functions 10 -> 11") || !strings.Contains(got, "merge-worthy 1 -> 2") {
		t.Errorf("scoreboard missing:\n%s", got)
	}
	if !strings.Contains(got, "/tmp/delta.json") {
		t.Errorf("delta path missing:\n%s", got)
	}
}

// A single edit can add a pair against every look-alike in the repo, so the
// list is capped and the remainder counted rather than printed.
func TestImpactDigestCapsTheList(t *testing.T) {
	d := snapshot.Delta{Comparable: true, UnitsAdded: []snapshot.Unit{{Key: "p.New"}}}
	for i := 0; i < maxImpactListed+4; i++ {
		d.PairsAdded = append(d.PairsAdded, snapshot.PairChange{
			Pair:         snapshot.Pair{A: "p.New", B: string(rune('a'+i)) + ".Other", Score: 0.9},
			Attributable: true,
		})
	}
	got := ImpactDigest(d, "")

	if n := strings.Count(got, "  NEW  "); n != maxImpactListed {
		t.Errorf("listed %d pairs, want %d", n, maxImpactListed)
	}
	if !strings.Contains(got, "4 more pair changes") {
		t.Errorf("remainder not counted:\n%s", got)
	}
}

func TestConceptDigestNamesPresentAndAbsentConcepts(t *testing.T) {
	s := snapshot.Snapshot{
		Functions: 3,
		Params:    snapshot.Params{Threshold: 0.6, TestsMode: "exclude"},
		Concepts:  []snapshot.TagCount{{Tag: "db_access", Count: 2}, {Tag: "retry", Count: 1}},
		Roles:     []snapshot.RoleCount{{Role: "leaf", Count: 3}},
		Units: []snapshot.Unit{
			{Key: "store.Save", Package: "store", Patterns: []string{"db_access"}},
		},
		Pairs: []snapshot.Pair{{A: "store.Save", B: "store.Load", Score: 0.8, Overlap: 0.5, MergeWorthy: true}},
	}
	got := ConceptDigest(s, "myrepo")

	for _, want := range []string{
		"myrepo",
		"3 Go functions",
		"test functions excluded",
		"db_access 2",
		"of which 1 are merge-worthy",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("digest missing %q:\n%s", want, got)
		}
	}
	// Per-target findings deliberately do not appear at SessionStart: the
	// pair listing moved to the user-prompt and pre-tool hooks, where a
	// target exists to scope them to, and the concepts-by-package survey
	// (alphabetical, capped) is gone outright.
	for _, gone := range []string{"store.Save <-> store.Load", "Concepts by package"} {
		if strings.Contains(got, gone) {
			t.Errorf("digest still contains %q, which moved out of SessionStart:\n%s", gone, got)
		}
	}
	// The absent list is the direct answer to "is there already something
	// doing this here?" when the answer is no.
	absentLine := ""
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "Concept tags with no occurrence") {
			absentLine = line
		}
	}
	if absentLine == "" {
		t.Fatalf("no absent-concepts line:\n%s", got)
	}
	if !strings.Contains(absentLine, "http_call") || strings.Contains(absentLine, "db_access") {
		t.Errorf("absent line wrong: %q", absentLine)
	}
}

func TestTruncateKeepsWholeLines(t *testing.T) {
	long := strings.Repeat("some line of digest text\n", 1000)
	got := truncate(long)

	if len(got) > digestMaxChars+len("\n  (truncated)\n") {
		t.Errorf("truncate returned %d chars, over the cap", len(got))
	}
	if !strings.HasSuffix(got, "(truncated)\n") {
		t.Error("truncated output must say so")
	}
	if strings.Contains(got, "some line of digest tex\n") {
		t.Error("truncate cut mid-line")
	}
}

func pc(a, b string, score, overlap float64, merge, attr bool) snapshot.PairChange {
	return snapshot.PairChange{
		Pair:         snapshot.Pair{A: a, B: b, Score: score, Overlap: overlap, MergeWorthy: merge},
		Attributable: attr,
	}
}

// The bar for interrupting the model is far higher than the bar for a line the
// user chose to look at. Anything below the merge-worthy gate, and anything the
// session did not cause, must not reach it.
func TestNotableAdmitsOnlyMergeWorthyAttributablePairs(t *testing.T) {
	d := snapshot.Delta{Comparable: true, PairsAdded: []snapshot.PairChange{
		pc("a.Keep", "b.Keep", 1.00, 0.71, true, true),
		pc("a.Low", "b.Low", 0.99, 0.12, false, true),     // below the gate
		pc("a.Churn", "b.Churn", 1.00, 0.80, true, false), // no edit caused it
	}}
	got := Notable(d)
	if len(got) != 1 {
		t.Fatalf("want 1 notable finding, got %d: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Line, "a.Keep") {
		t.Errorf("wrong finding surfaced: %q", got[0].Line)
	}
	if got[0].Key != "new:a.Keep|b.Keep" {
		t.Errorf("key = %q, want a stable new: key", got[0].Key)
	}
}

// A gate crossing is reportable even with no score movement — that is the whole
// reason Drift admission was widened.
func TestNotableReportsGateCrossings(t *testing.T) {
	d := snapshot.Delta{Comparable: true, Drift: []snapshot.Drift{{
		A: "a.One", B: "b.Two",
		ScoreBefore: 0.8, ScoreAfter: 0.8,
		OverlapBefore: 0.31, OverlapAfter: 0.44,
		MergeWorthyBefore: false, MergeWorthyAfter: true,
		Attributable: true,
	}}}
	got := Notable(d)
	if len(got) != 1 {
		t.Fatalf("want 1 finding, got %d", len(got))
	}
	if !strings.Contains(got[0].Line, "became merge-worthy") {
		t.Errorf("line does not name the crossing: %q", got[0].Line)
	}
	if got[0].Key != "gate:a.One|b.Two" {
		t.Errorf("key = %q, want a gate: key", got[0].Key)
	}
}

// Drift that neither crosses the gate nor moves visibly is not worth a turn.
func TestNotableIgnoresSmallDrift(t *testing.T) {
	d := snapshot.Delta{Comparable: true, Drift: []snapshot.Drift{{
		A: "a.One", B: "b.Two",
		ScoreBefore: 0.80, ScoreAfter: 0.81, // below notableDrift
		MergeWorthyBefore: true, MergeWorthyAfter: true,
		Attributable: true,
	}}}
	if got := Notable(d); len(got) != 0 {
		t.Fatalf("want no findings, got %+v", got)
	}
}

func TestNotableIgnoresIncomparableDelta(t *testing.T) {
	d := snapshot.Delta{Comparable: false, PairsAdded: []snapshot.PairChange{
		pc("a.X", "b.Y", 1.0, 0.9, true, true),
	}}
	if got := Notable(d); got != nil {
		t.Fatalf("want nil for an incomparable delta, got %+v", got)
	}
}

func TestAgentDigestEmptyFindingsRenderNothing(t *testing.T) {
	if got := AgentDigest(nil); got != "" {
		t.Errorf("AgentDigest(nil) = %q, want the empty string", got)
	}
}

// The note arrives while the turn is being deliberately continued, so it must
// read as a measurement and not as an instruction to go refactor.
func TestAgentDigestStatesFactsAndCapsTheList(t *testing.T) {
	var f []Finding
	for _, n := range []string{"one", "two", "three", "four", "five"} {
		f = append(f, Finding{Key: "new:" + n, Line: n + " <-> other"})
	}
	got := AgentDigest(f)
	if strings.Count(got, "<->") != maxAgentListed {
		t.Errorf("listed %d findings, want %d:\n%s", strings.Count(got, "<->"), maxAgentListed, got)
	}
	if !strings.Contains(got, "2 further findings not listed") {
		t.Errorf("missing the roll-up line:\n%s", got)
	}
	if !strings.Contains(got, "No change is required.") {
		t.Errorf("digest does not bound itself as a measurement:\n%s", got)
	}
}

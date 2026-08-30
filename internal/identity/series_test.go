package identity

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

// The fixtures below are three stages of one refactor, over a corpus with
// enough variety in it to be measurable.
//
// The filler matters and is not padding. Label weight is ln(N/df) over the
// union of both snapshots' bags, so on a corpus of two near-identical functions
// every label has df == N, every weight is 0, and the weighted Jaccard of any
// two bodies is 0/0 — the matcher then refuses every similarity match and the
// classification collapses to "everything was deleted and something new
// arrived". That is correct behaviour on a corpus with no information in it,
// and it makes a two-function fixture useless for testing a matcher.
const filler = `
func Encode(v string) string {
	out := ""
	for _, r := range v {
		out += string(r)
	}
	return out
}

func Lookup(m []string, want string) bool {
	for _, s := range m {
		if s == want {
			return true
		}
	}
	return false
}

func Clamp(v, lo, hi int) int {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	}
	return v
}

func Join(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

func Count(xs []int, want int) int {
	n := 0
	for _, x := range xs {
		if x == want {
			n++
		}
	}
	return n
}

func First(xs []string) (string, bool) {
	for _, x := range xs {
		if x != "" {
			return x, true
		}
	}
	return "", false
}
`

const (
	total = `
func Total(xs []int) int {
	sum := 0
	for _, x := range xs {
		if x > 0 {
			sum += x
		}
	}
	return sum
}
`
	// The same body under a new name.
	sum = `
func Sum(xs []int) int {
	sum := 0
	for _, x := range xs {
		if x > 0 {
			sum += x
		}
	}
	return sum
}
`
	helper = `
func Helper(a, b int) int {
	if a > b {
		return a
	}
	return b
}
`
)

// v1 -> v2 renames Total to Sum in place. v2 -> v3 moves Sum to another
// package and deletes Helper.
var (
	srcV1      = "package alpha\n" + filler + total + helper
	srcV2      = "package alpha\n" + filler + sum + helper
	srcV3Alpha = "package alpha\n" + filler
	srcV3Beta  = "package beta\n" + sum
)

func chain(t *testing.T, snaps ...snapshot.Snapshot) Series {
	t.Helper()
	s, err := Chain(snaps, Options{})
	if err != nil {
		t.Fatalf("Chain: %v", err)
	}
	if !s.Comparable {
		t.Fatalf("Chain refused: %s", s.Reason)
	}
	return s
}

// trackFor returns the single track whose points mention key, failing when
// there is not exactly one.
func trackFor(t *testing.T, s Series, key string) Track {
	t.Helper()
	var hits []Track
	for _, tr := range s.Tracks {
		for _, p := range tr.Points {
			if p.Key == key {
				hits = append(hits, tr)
				break
			}
		}
	}
	if len(hits) != 1 {
		t.Fatalf("expected exactly one track mentioning %q, got %d", key, len(hits))
	}
	return hits[0]
}

// TestChainFollowsARenameAndAMove is the whole reason a track exists: a
// function that changes its name at one step and its package at the next is
// one lifeline, not three arrivals and two departures.
func TestChainFollowsARenameAndAMove(t *testing.T) {
	v1 := corpus(t, map[string]string{"a.go": srcV1})
	v2 := corpus(t, map[string]string{"a.go": srcV2})
	v3 := corpus(t, map[string]string{"a.go": srcV3Alpha, "b.go": srcV3Beta})

	s := chain(t, v1, v2, v3)

	if len(s.Deltas) != 2 {
		t.Fatalf("Deltas = %d, want 2 for three snapshots", len(s.Deltas))
	}

	tr := trackFor(t, s, "alpha.Total")
	if len(tr.Points) != 3 {
		t.Fatalf("the renamed-then-moved function has %d points, want 3: %+v", len(tr.Points), tr.Points)
	}
	want := []struct {
		key   string
		class Class
	}{
		{"alpha.Total", ""},
		{"alpha.Sum", Renamed},
		{"beta.Sum", Moved},
	}
	for i, w := range want {
		if tr.Points[i].Step != i {
			t.Errorf("point %d is at step %d, want %d", i, tr.Points[i].Step, i)
		}
		if tr.Points[i].Key != w.key {
			t.Errorf("point %d key = %q, want %q", i, tr.Points[i].Key, w.key)
		}
		if tr.Points[i].Class != w.class {
			t.Errorf("point %d class = %q, want %q", i, tr.Points[i].Class, w.class)
		}
	}
	if tr.Fate != "" {
		t.Errorf("a track reaching the last step has Fate %q, want empty", tr.Fate)
	}
}

// TestChainRecordsAFate pins the other end of a lifeline: a function that goes
// away stops at the step before, and the ending is named rather than left as a
// silent gap.
func TestChainRecordsAFate(t *testing.T) {
	v1 := corpus(t, map[string]string{"a.go": srcV1})
	v2 := corpus(t, map[string]string{"a.go": srcV2})
	v3 := corpus(t, map[string]string{"a.go": srcV3Alpha, "b.go": srcV3Beta})

	s := chain(t, v1, v2, v3)

	tr := trackFor(t, s, "alpha.Helper")
	if len(tr.Points) != 2 {
		t.Fatalf("the deleted function has %d points, want 2", len(tr.Points))
	}
	if tr.Points[len(tr.Points)-1].Step != 1 {
		t.Errorf("the track ends at step %d, want 1", tr.Points[len(tr.Points)-1].Step)
	}
	if tr.Fate != Deleted {
		t.Errorf("Fate = %q, want %q", tr.Fate, Deleted)
	}
}

// TestChainCoversEveryFunctionExactlyOnce is the invariant that makes the page
// a partition rather than a selection: every unit of every snapshot appears in
// exactly one track, at exactly one point.
func TestChainCoversEveryFunctionExactlyOnce(t *testing.T) {
	v1 := corpus(t, map[string]string{"a.go": srcV1})
	v2 := corpus(t, map[string]string{"a.go": srcV2})
	v3 := corpus(t, map[string]string{"a.go": srcV3Alpha, "b.go": srcV3Beta})
	snaps := []snapshot.Snapshot{v1, v2, v3}

	s := chain(t, snaps...)

	type at struct {
		step int
		key  string
	}
	seen := map[at]int{}
	for _, tr := range s.Tracks {
		for _, p := range tr.Points {
			seen[at{p.Step, p.Key}]++
		}
	}
	total := 0
	for i, sn := range snaps {
		total += len(sn.Units)
		for _, u := range sn.Units {
			if seen[at{i, u.Key}] != 1 {
				t.Errorf("unit %s at step %d appears in %d track points, want 1", u.Key, i, seen[at{i, u.Key}])
			}
		}
	}
	points := 0
	for _, tr := range s.Tracks {
		points += len(tr.Points)
	}
	if points != total {
		t.Errorf("tracks hold %d points for %d units across the series", points, total)
	}
}

// TestChainPointsAreContiguousAndAscending pins what a track claims. It is the
// transitive closure of *consecutive* matches, so nothing may skip a step —
// a function that vanished and came back is two tracks, which is the honest
// reading since nothing observed it in between.
func TestChainPointsAreContiguousAndAscending(t *testing.T) {
	v1 := corpus(t, map[string]string{"a.go": srcV1})
	v2 := corpus(t, map[string]string{"a.go": srcV2})
	v3 := corpus(t, map[string]string{"a.go": srcV3Alpha, "b.go": srcV3Beta})

	s := chain(t, v1, v2, v3)
	for _, tr := range s.Tracks {
		if len(tr.Points) == 0 {
			t.Fatalf("track %d has no points", tr.ID)
		}
		for i := 1; i < len(tr.Points); i++ {
			if tr.Points[i].Step != tr.Points[i-1].Step+1 {
				t.Errorf("track %d jumps from step %d to %d", tr.ID, tr.Points[i-1].Step, tr.Points[i].Step)
			}
		}
	}
}

// TestChainIsDeterministic mirrors the repo's own invariant: the same series
// must produce the same tracks, in the same order, with the same IDs.
func TestChainIsDeterministic(t *testing.T) {
	v1 := corpus(t, map[string]string{"a.go": srcV1})
	v2 := corpus(t, map[string]string{"a.go": srcV2})
	v3 := corpus(t, map[string]string{"a.go": srcV3Alpha, "b.go": srcV3Beta})

	for i := 0; i < 8; i++ {
		a := chain(t, v1, v2, v3)
		b := chain(t, v1, v2, v3)
		if len(a.Tracks) != len(b.Tracks) {
			t.Fatalf("track counts differ between runs: %d vs %d", len(a.Tracks), len(b.Tracks))
		}
		for j := range a.Tracks {
			if a.Tracks[j].ID != b.Tracks[j].ID || lastKey(a.Tracks[j]) != lastKey(b.Tracks[j]) {
				t.Fatalf("track %d differs between runs", j)
			}
		}
	}
}

// TestChainRefusesOnOneBadStep pins that one refusing boundary refuses the
// whole series. A page that stepped over a gap would present two
// incommensurable halves as one history.
func TestChainRefusesOnOneBadStep(t *testing.T) {
	v1 := corpus(t, map[string]string{"a.go": srcV1})
	v2 := corpus(t, map[string]string{"a.go": srcV2})
	bad := corpus(t, map[string]string{"a.go": srcV2})
	bad.RuleSet = "not-the-shipped-rule-set"

	s, err := Chain([]snapshot.Snapshot{v1, v2, bad}, Options{})
	if err != nil {
		t.Fatalf("Chain: %v", err)
	}
	if s.Comparable {
		t.Fatal("a series with a mismatched canon rule set was accepted")
	}
	if s.Reason == "" {
		t.Error("the refusal carries no reason")
	}
	// The reason must name the boundary; a reader has to know which file.
	if !strings.Contains(s.Reason, "step 1 to 2") {
		t.Errorf("Reason = %q, want it to name the offending step", s.Reason)
	}
	if len(s.Tracks) != 0 || len(s.Deltas) != 0 {
		t.Error("a refused series must carry no findings")
	}
}

func TestChainNeedsTwoSnapshots(t *testing.T) {
	v1 := corpus(t, map[string]string{"a.go": srcV1})
	if _, err := Chain([]snapshot.Snapshot{v1}, Options{}); err == nil {
		t.Error("a one-snapshot series should be an error, not an empty history")
	}
}

func lastKey(t Track) string { return t.Points[len(t.Points)-1].Key }

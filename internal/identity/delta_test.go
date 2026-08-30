package identity

import (
	"bytes"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

// The pair half of a Delta is a set difference over stored records, so these
// fixtures attach pair lists to real snapshots rather than running a
// retrieval. The units, the bags and the digests are still the pipeline's —
// only the pair list is stated, which is exactly the data Since reads.
const deltaSrcOld = `package svc

import "strings"

func Total(lines []string) int {
	total := 0
	for _, line := range lines {
		if len(strings.TrimSpace(line)) < 1 {
			continue
		}
		total = total - 1
	}
	return total
}

func Describe(name string) string {
	if len(name) < 1 {
		return "empty"
	}
	return strings.ToUpper(name)
}
`

var deltaSrcNew = strings.Replace(deltaSrcOld, "func Total(", "func Sum(", 1)

func withPairs(s snapshot.Snapshot, ps ...snapshot.Pair) snapshot.Snapshot {
	s.Pairs = ps
	return s
}

// A rename dissolves the pairs its old key held and creates the same pairs
// under its new key — and both changes are attributed to the rename, which is
// the claim snapshot.Diff cannot make.
func TestSinceAttributesPairChangesToTheRename(t *testing.T) {
	old := withPairs(corpus(t, map[string]string{"svc.go": deltaSrcOld}),
		snapshot.Pair{A: "svc.Describe", B: "svc.Total", Score: 0.4, Overlap: 0.3, Explain: "differs by one extra if"})
	head := withPairs(corpus(t, map[string]string{"svc.go": deltaSrcNew}),
		snapshot.Pair{A: "svc.Describe", B: "svc.Sum", Score: 0.4, Overlap: 0.3, Explain: "differs by one extra if"})

	d, err := Since(old, head, Options{})
	if err != nil {
		t.Fatalf("Since: %v", err)
	}
	if !d.Comparable {
		t.Fatalf("refused: %s", d.Reason)
	}
	if d.Count(Renamed) != 1 {
		t.Fatalf("want one rename, got %+v", d.Counts)
	}
	if len(d.Created) != 1 || len(d.Dissolved) != 1 {
		t.Fatalf("want one created and one dissolved pair, got %d and %d", len(d.Created), len(d.Dissolved))
	}

	created := d.Created[0]
	if created.B != "svc.Sum" || created.BClass != Renamed {
		t.Errorf("created pair must be attributed to the renamed function: %+v", created)
	}
	if created.AClass != Unchanged {
		t.Errorf("the untouched side must be classified unchanged: %+v", created)
	}
	if !created.Attributable() {
		t.Error("a pair with a renamed side is attributable")
	}
	if created.Explain != "differs by one extra if" {
		t.Errorf("Explain must come off the stored field, got %q", created.Explain)
	}

	dissolved := d.Dissolved[0]
	if dissolved.B != "svc.Total" || dissolved.BClass != Renamed {
		t.Errorf("dissolved pair must read the old side's key and class: %+v", dissolved)
	}
}

// A pair present in one run and not the other with nothing classified on
// either side is corpus churn, and says so rather than being blamed on the
// session.
func TestSinceMarksUnattributedChurn(t *testing.T) {
	src := corpus(t, map[string]string{"svc.go": deltaSrcOld})
	old := withPairs(src)
	head := withPairs(src, snapshot.Pair{A: "svc.Describe", B: "svc.Total", Score: 0.4})

	d, err := Since(old, head, Options{})
	if err != nil {
		t.Fatalf("Since: %v", err)
	}
	if len(d.Created) != 1 {
		t.Fatalf("want one created pair, got %d", len(d.Created))
	}
	p := d.Created[0]
	if p.Attributable() {
		t.Errorf("nothing was classified, so nothing attributes this pair: %+v", p)
	}
	if got := CauseLine(p); !strings.Contains(got, "retrieval re-ranking") {
		t.Errorf("CauseLine = %q, want the unattributed wording", got)
	}
	// Every function is unchanged and only a pair moved, so the delta is not
	// empty — but the classification half of it is silent.
	if d.Empty() {
		t.Error("a created pair makes a delta non-empty")
	}
}

func TestDeltaEmpty(t *testing.T) {
	src := corpus(t, map[string]string{"svc.go": deltaSrcOld})
	d, err := Since(src, src, Options{})
	if err != nil {
		t.Fatalf("Since: %v", err)
	}
	if !d.Empty() {
		t.Errorf("an unchanged tree must produce an empty delta: %+v", d)
	}
}

// A refusal carries no pair changes: a comparison the matcher declined cannot
// attribute a pair change to anything either.
func TestSinceRefusalCarriesNoPairs(t *testing.T) {
	old := withPairs(corpus(t, map[string]string{"svc.go": deltaSrcOld}),
		snapshot.Pair{A: "svc.Describe", B: "svc.Total"})
	head := withPairs(corpus(t, map[string]string{"svc.go": deltaSrcNew}),
		snapshot.Pair{A: "svc.Describe", B: "svc.Sum"})
	head.RuleSet += "-next"

	d, err := Since(old, head, Options{})
	if err != nil {
		t.Fatalf("Since: %v", err)
	}
	if d.Comparable {
		t.Fatal("a rule-set mismatch must refuse")
	}
	if len(d.Created) != 0 || len(d.Dissolved) != 0 {
		t.Errorf("a refused delta must carry no pair changes: %+v", d)
	}
}

// The order mirrors snapshot.sortPairChanges: attributable, then merge-worthy,
// then score, then names. Only the first key means something different here —
// see PairChange.Attributable.
func TestSortPairChangesOrder(t *testing.T) {
	ps := []PairChange{
		{A: "z", B: "z", Score: 0.9},                                   // churn, high score
		{A: "c", B: "d", Score: 0.5, AClass: Edited},                   // attributable
		{A: "a", B: "b", Score: 0.5, AClass: Added, MergeWorthy: true}, // attributable + merge-worthy
		{A: "e", B: "f", Score: 0.7, BClass: Moved},                    // attributable, higher score
	}
	sortPairChanges(ps)

	want := []string{"a", "e", "c", "z"}
	for i, w := range want {
		if ps[i].A != w {
			t.Fatalf("order = %v, want %v", keysOf(ps), want)
		}
	}
}

func keysOf(ps []PairChange) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.A
	}
	return out
}

// Both renderings must be byte-stable, which is the repo's invariant and the
// only thing that makes a hook digest safe to compare across turns.
func TestDeltaRenderingIsDeterministic(t *testing.T) {
	old := withPairs(corpus(t, map[string]string{"svc.go": deltaSrcOld}),
		snapshot.Pair{A: "svc.Describe", B: "svc.Total", Score: 0.4, Explain: "differs by one extra if"})
	head := withPairs(corpus(t, map[string]string{"svc.go": deltaSrcNew}),
		snapshot.Pair{A: "svc.Describe", B: "svc.Sum", Score: 0.4, Explain: "differs by one extra if"})

	for i := 0; i < 3; i++ {
		d1, err := Since(old, head, Options{})
		if err != nil {
			t.Fatal(err)
		}
		d2, err := Since(old, head, Options{})
		if err != nil {
			t.Fatal(err)
		}
		var a, b bytes.Buffer
		PrintDelta(&a, d1, false)
		PrintDelta(&b, d2, false)
		if a.String() != b.String() {
			t.Fatalf("PrintDelta is not byte-identical:\n%s\n---\n%s", a.String(), b.String())
		}
		a.Reset()
		b.Reset()
		MarkdownDelta(&a, d1, false)
		MarkdownDelta(&b, d2, false)
		if a.String() != b.String() {
			t.Fatalf("MarkdownDelta is not byte-identical:\n%s\n---\n%s", a.String(), b.String())
		}
	}
}

// The text report leads with the classification and follows with the pairs,
// and the census line carries both counts including the zeros.
func TestPrintDeltaShape(t *testing.T) {
	old := withPairs(corpus(t, map[string]string{"svc.go": deltaSrcOld}))
	head := withPairs(corpus(t, map[string]string{"svc.go": deltaSrcNew}),
		snapshot.Pair{A: "svc.Describe", B: "svc.Sum", Score: 0.4, Explain: "identical after rename"})

	d, err := Since(old, head, Options{})
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	PrintDelta(&b, d, false)
	got := b.String()

	if !strings.HasPrefix(got, "Delta since the baseline\n") {
		t.Errorf("missing the report title:\n%s", got)
	}
	iClass := strings.Index(got, "renamed 1")
	iPairs := strings.Index(got, "pairs created 1, dissolved 0")
	if iClass < 0 || iPairs < 0 || iClass > iPairs {
		t.Errorf("the classification must precede the pair section:\n%s", got)
	}
	if !strings.Contains(got, "    explain: identical after rename\n") {
		t.Errorf("the pair line must carry its stored explanation:\n%s", got)
	}
}

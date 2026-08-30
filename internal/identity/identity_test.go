package identity

import (
	"bytes"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/canon"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// corpus builds a real snapshot from real Go source.
//
// The bags are what the pipeline itself would produce: parser.ParseSource runs
// the same canonicalizer and the same fingerprint.WLBag every analyze run
// does, and snapshot.Build encodes them against a real label dictionary. A
// hand-written bag would let a fixture assert a class the tool could never
// reach on real code, which is the one thing these tests exist to rule out.
//
// files is filename -> source. Sorted before parsing so unit order — and
// therefore every index in the matcher — depends on nothing but the map's
// contents.
func corpus(t *testing.T, files map[string]string) snapshot.Snapshot {
	t.Helper()
	var units []parser.CodeUnit
	for _, name := range sortedNames(files) {
		us, err := parser.ParseSource(name, []byte(files[name]))
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		units = append(units, us...)
	}
	docs := make([]concepter.ConceptDoc, len(units))
	return snapshot.Build(units, docs, nil, map[ontology.TermID]int{}, "", "test",
		snapshot.Params{Threshold: 0.6, MinNodes: 12, TestsMode: "exclude"}, snapshot.CorpusMetrics{})
}

func sortedNames(files map[string]string) []string {
	out := make([]string, 0, len(files))
	for k := range files {
		out = append(out, k)
	}
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func compare(t *testing.T, old, new snapshot.Snapshot) Result {
	t.Helper()
	r, err := Compare(old, new, Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if !r.Comparable {
		t.Fatalf("Compare refused: %s", r.Reason)
	}
	return r
}

// find returns the single change whose old or new side carries key, failing
// when there is not exactly one.
func find(t *testing.T, r Result, key string) Change {
	t.Helper()
	var hits []Change
	for _, c := range r.Changes {
		for _, m := range append(append([]Member{}, c.Old...), c.New...) {
			if m.Key == key {
				hits = append(hits, c)
				break
			}
		}
	}
	if len(hits) != 1 {
		t.Fatalf("expected exactly one change mentioning %q, got %d:\n%s", key, len(hits), render(r))
	}
	return hits[0]
}

func render(r Result) string {
	var b bytes.Buffer
	Print(&b, r, true)
	return b.String()
}

// wantClass asserts the class of the finding that mentions key, printing the
// whole report on failure — a wrong class is almost always explained by some
// other line.
func wantClass(t *testing.T, r Result, key string, want Class) Change {
	t.Helper()
	c := find(t, r, key)
	if c.Class != want {
		t.Errorf("%s classified %s, want %s\n%s", key, c.Class, want, render(r))
	}
	return c
}

// The bodies below are deliberately substantial and distinct. A three-line
// function's WL bag is dominated by labels every function in the fixture
// carries, which makes every pair look alike; these are the size real
// functions are, which is the regime the thresholds were chosen for.

const bodyScan = `
func Scan(lines []string) (int, error) {
	total := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		n, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, fmt.Errorf("scan %q: %w", trimmed, err)
		}
		if n < 0 {
			return 0, fmt.Errorf("negative %d", n)
		}
		total += n
	}
	return total, nil
}
`

// bodyScanEdited is bodyScan with one guard added — the "small edit" the
// rename-plus-edit fixture needs: enough to move the digest, not enough to
// drop the pair below the rename floor.
const bodyScanEdited = `
func Scan(lines []string) (int, error) {
	total := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "#" {
			continue
		}
		n, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, fmt.Errorf("scan %q: %w", trimmed, err)
		}
		if n < 0 {
			return 0, fmt.Errorf("negative %d", n)
		}
		total += n
	}
	return total, nil
}
`

const bodyFormat = `
func Format(rows [][]string, sep string) string {
	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		for j, cell := range row {
			if j > 0 {
				b.WriteString(sep)
			}
			b.WriteString(strings.ToUpper(cell))
		}
	}
	return b.String()
}
`

// ballast keeps the label IDF from degenerating. With two functions in the
// corpus every label either appears in both (weight ln(2/2)=0) or in one
// (weight ln 2), which is a corpus so small the weighting says nothing. These
// four unrelated functions are present in every fixture on both sides, so
// they contribute an unchanging population and a realistic df spread.
const ballast = `package ballast

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func LoadNames(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	names := strings.Split(string(data), "\n")
	sort.Strings(names)
	return names, nil
}

func Longest(words []string) string {
	best := ""
	for _, w := range words {
		if len(w) > len(best) {
			best = w
		}
	}
	return best
}

func Counts(words []string) map[string]int {
	out := make(map[string]int)
	for _, w := range words {
		out[strings.ToLower(w)]++
	}
	return out
}

func Render(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%d\n", k, m[k])
	}
	return b.String()
}
`

// pkg wraps one or more bodies in a package clause plus the imports they all
// need. Unused imports are fine: go/parser does not type-check.
func pkg(name string, bodies ...string) string {
	return "package " + name + `

import (
	"fmt"
	"strconv"
	"strings"
)
` + strings.Join(bodies, "\n")
}

func sides(t *testing.T, oldSrc, newSrc map[string]string) (snapshot.Snapshot, snapshot.Snapshot) {
	t.Helper()
	withBallast := func(m map[string]string) map[string]string {
		out := map[string]string{"ballast/ballast.go": ballast}
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	return corpus(t, withBallast(oldSrc)), corpus(t, withBallast(newSrc))
}

// --- the eight classes -----------------------------------------------------

func TestUnchanged(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan)},
		map[string]string{"a/a.go": pkg("a", bodyScan)})
	r := compare(t, old, new)
	c := wantClass(t, r, "a.Scan", Unchanged)
	if !c.DigestEqual {
		t.Error("unchanged must report equal digests")
	}
	// Every ballast function is unchanged too, so the whole comparison is.
	if n := len(r.Changes); n != r.Count(Unchanged) {
		t.Errorf("%d changes but %d unchanged; identical inputs must classify everything unchanged\n%s",
			n, r.Count(Unchanged), render(r))
	}
}

func TestEdited(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan)},
		map[string]string{"a/a.go": pkg("a", bodyScanEdited)})
	r := compare(t, old, new)
	c := wantClass(t, r, "a.Scan", Edited)
	if c.DigestEqual {
		t.Error("edited must report differing digests")
	}
	if c.Jaccard <= 0 {
		t.Errorf("edited pair must carry positive shared mass as evidence, got %v", c.Jaccard)
	}
}

func TestRenamed(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan)},
		map[string]string{"a/a.go": pkg("a", strings.Replace(bodyScan, "func Scan(", "func Total(", 1))})
	r := compare(t, old, new)
	c := wantClass(t, r, "a.Scan", Renamed)
	if !c.DigestEqual {
		t.Error("a pure rename leaves the body untouched, so the digests must agree")
	}
	if !c.NameChanged || c.PackageChanged {
		t.Errorf("rename must record NameChanged and not PackageChanged, got %+v", c)
	}
}

func TestMoved(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan)},
		map[string]string{"b/b.go": pkg("b", bodyScan)})
	r := compare(t, old, new)
	c := wantClass(t, r, "a.Scan", Moved)
	if !c.DigestEqual {
		t.Error("a pure move leaves the body untouched, so the digests must agree")
	}
	if c.New[0].Key != "b.Scan" {
		t.Errorf("moved to %q, want b.Scan", c.New[0].Key)
	}
}

func TestNew(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan)},
		map[string]string{"a/a.go": pkg("a", bodyScan, bodyFormat)})
	r := compare(t, old, new)
	wantClass(t, r, "a.Format", Added)
	wantClass(t, r, "a.Scan", Unchanged)
}

func TestDeleted(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan, bodyFormat)},
		map[string]string{"a/a.go": pkg("a", bodyScan)})
	r := compare(t, old, new)
	wantClass(t, r, "a.Format", Deleted)
	wantClass(t, r, "a.Scan", Unchanged)
}

// bodyPipeline is one body doing two separable jobs, and bodyPipelineSum and
// bodyPipelineJoin are the two functions it splits into — each one carrying
// the corresponding loop verbatim, which is what an extraction refactor
// actually produces.
//
// The loops carry no return statement, deliberately. A piece extracted into
// its own function gains a return the original never had, and a return inside
// the extracted region would differ on both sides and take its whole
// Weisfeiler-Lehman label chain with it. That is a real limit of the rule and
// not something the fixture should pretend away — see the split note in
// CLAUDE.md — but the fixture's job is to pin the class on a clean case, and
// an extraction of a self-contained loop is the clean case.
const bodyPipeline = `
func Pipeline(lines []string, sep string) (int, string) {
	total := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		n, err := strconv.Atoi(trimmed)
		if err != nil {
			continue
		}
		if n < 0 {
			n = -n
		}
		total += n
	}
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteString(sep)
		}
		if line == "" {
			continue
		}
		b.WriteString(strings.ToUpper(strings.TrimSpace(line)))
	}
	return total, b.String()
}
`

const bodyPipelineSum = `
func Sum(lines []string) int {
	total := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		n, err := strconv.Atoi(trimmed)
		if err != nil {
			continue
		}
		if n < 0 {
			n = -n
		}
		total += n
	}
	return total
}
`

const bodyPipelineJoin = `
func Join(lines []string, sep string) string {
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteString(sep)
		}
		if line == "" {
			continue
		}
		b.WriteString(strings.ToUpper(strings.TrimSpace(line)))
	}
	return b.String()
}
`

func TestSplit(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyPipeline)},
		map[string]string{"a/a.go": pkg("a", bodyPipelineSum, bodyPipelineJoin)})
	r := compare(t, old, new)
	c := wantClass(t, r, "a.Pipeline", Split)
	if len(c.New) != 2 {
		t.Fatalf("split into %d bodies, want 2\n%s", len(c.New), render(r))
	}
	if c.New[0].Key != "a.Join" || c.New[1].Key != "a.Sum" {
		t.Errorf("split parts %q, %q; want a.Join, a.Sum sorted by key", c.New[0].Key, c.New[1].Key)
	}
	for _, m := range c.New {
		if m.Containment < 0.8 {
			t.Errorf("part %s admitted at containment %v, below the 0.8 floor", m.Key, m.Containment)
		}
	}
	// Absorption: neither half may also be reported as new.
	if got := r.Count(Added); got != 0 {
		t.Errorf("%d functions reported new; a split's parts must be absorbed\n%s", got, render(r))
	}
}

func TestMerged(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyPipelineSum, bodyPipelineJoin)},
		map[string]string{"a/a.go": pkg("a", bodyPipeline)})
	r := compare(t, old, new)
	c := wantClass(t, r, "a.Pipeline", Merged)
	if len(c.Old) != 2 {
		t.Fatalf("merged from %d bodies, want 2\n%s", len(c.Old), render(r))
	}
	if c.Old[0].Key != "a.Join" || c.Old[1].Key != "a.Sum" {
		t.Errorf("merge sources %q, %q; want a.Join, a.Sum sorted by key", c.Old[0].Key, c.Old[1].Key)
	}
	if got := r.Count(Deleted); got != 0 {
		t.Errorf("%d functions reported deleted; a merge's sources must be absorbed\n%s", got, render(r))
	}
}

// --- the two composites ----------------------------------------------------

// TestRenamePlusSmallEdit pins the documented precedence: rule 2 fires, the
// class is renamed, and the edit rides along as a secondary fact rather than
// as a second finding.
func TestRenamePlusSmallEdit(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan)},
		map[string]string{"a/a.go": pkg("a", strings.Replace(bodyScanEdited, "func Scan(", "func Total(", 1))})
	r := compare(t, old, new)
	c := wantClass(t, r, "a.Scan", Renamed)
	if c.DigestEqual {
		t.Error("the body was edited too; the digests must differ")
	}
	if c.New[0].Key != "a.Total" {
		t.Errorf("renamed to %q, want a.Total", c.New[0].Key)
	}
	if c.Jaccard < 0.5 {
		t.Errorf("jaccard %v is below the rename floor; the fixture's edit is meant to be small", c.Jaccard)
	}
	// The secondary fact must be printed, or the collapsed label loses
	// information the reader needs.
	lines := strings.Join(Lines(c), "\n")
	if !strings.Contains(lines, "body edited") {
		t.Errorf("rename+edit line must say the body was edited:\n%s", lines)
	}
}

// TestMovePlusRename pins rule 1: the package change wins, and the rename is
// printed on the same line.
func TestMovePlusRename(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan)},
		map[string]string{"b/b.go": pkg("b", strings.Replace(bodyScan, "func Scan(", "func Total(", 1))})
	r := compare(t, old, new)
	c := wantClass(t, r, "a.Scan", Moved)
	if !c.NameChanged || !c.PackageChanged {
		t.Errorf("move+rename must record both facts, got %+v", c)
	}
	if !c.DigestEqual {
		t.Error("the body was untouched; the digests must agree")
	}
	lines := strings.Join(Lines(c), "\n")
	if !strings.Contains(lines, "renamed Scan -> Total") {
		t.Errorf("move+rename line must name the rename:\n%s", lines)
	}
}

// --- edges -----------------------------------------------------------------

func TestIdenticalInputsAreAllUnchanged(t *testing.T) {
	s := corpus(t, map[string]string{
		"ballast/ballast.go": ballast,
		"a/a.go":             pkg("a", bodyScan, bodyFormat),
	})
	r := compare(t, s, s)
	for _, c := range r.Changes {
		if c.Class != Unchanged {
			t.Errorf("comparing a snapshot with itself produced %s for %s", c.Class, primaryKey(c))
		}
	}
	if r.Count(Unchanged) != len(s.Units) {
		t.Errorf("%d unchanged, want %d units", r.Count(Unchanged), len(s.Units))
	}
}

func TestEmptyOld(t *testing.T) {
	old := corpus(t, map[string]string{"empty/e.go": "package empty\n\nvar X = 1\n"})
	new := corpus(t, map[string]string{"a/a.go": pkg("a", bodyScan)})
	r, err := Compare(old, new, Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if !r.Comparable {
		t.Fatalf("refused: %s", r.Reason)
	}
	if r.Count(Added) != len(new.Units) || len(r.Changes) != len(new.Units) {
		t.Errorf("an empty old snapshot must make everything new; got %s", render(r))
	}
}

func TestEmptyNew(t *testing.T) {
	old := corpus(t, map[string]string{"a/a.go": pkg("a", bodyScan)})
	new := corpus(t, map[string]string{"empty/e.go": "package empty\n\nvar X = 1\n"})
	r, err := Compare(old, new, Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if r.Count(Deleted) != len(old.Units) || len(r.Changes) != len(old.Units) {
		t.Errorf("an empty new snapshot must make everything deleted; got %s", render(r))
	}
}

func TestRuleSetMismatchRefused(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan)},
		map[string]string{"a/a.go": pkg("a", bodyScan)})
	new.RuleSet = canon.Version + "-next"
	r, err := Compare(old, new, Options{})
	if err != nil {
		t.Fatalf("a refusal is a Result, not an error; got %v", err)
	}
	if r.Comparable {
		t.Fatal("a canon rule-set mismatch must refuse: every WL label moves under a different rule set")
	}
	if !strings.Contains(r.Reason, "rule set") {
		t.Errorf("reason %q should name the rule set", r.Reason)
	}
	if len(r.Changes) != 0 {
		t.Error("a refusal must produce no changes; an empty list would otherwise read as 'nothing happened'")
	}
	var b bytes.Buffer
	Print(&b, r, false)
	if !strings.HasPrefix(b.String(), "Not comparable:") {
		t.Errorf("the text report must lead with the refusal, got %q", b.String())
	}
}

func TestSchemaMismatchRefused(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan)},
		map[string]string{"a/a.go": pkg("a", bodyScan)})
	old.Schema = snapshot.Schema - 1
	r, err := Compare(old, new, Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if r.Comparable {
		t.Fatal("a schema mismatch must refuse")
	}
}

// TestParamsMismatchIsNoted is the documented loosening: identity reads no
// Pair, so a param difference is recorded and matched anyway.
func TestParamsMismatchIsNoted(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan)},
		map[string]string{"a/a.go": pkg("a", bodyScan)})
	new.Params.Threshold = 0.9
	new.Ontology = ontology.Version + "-next"
	r, err := Compare(old, new, Options{})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if !r.Comparable {
		t.Fatalf("params and ontology mismatches must not refuse; got %s", r.Reason)
	}
	if len(r.Notes) != 2 {
		t.Errorf("want a note per allowed mismatch, got %v", r.Notes)
	}
	if r.Count(Unchanged) != len(old.Units) {
		t.Errorf("the corpora are identical; everything should still be unchanged\n%s", render(r))
	}
}

// TestUnrelatedFunctionsAreNotRenames pins the rename floor. Two functions
// with nothing in common must not be paired off just because they are the
// only leftovers.
func TestUnrelatedFunctionsAreNotRenames(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan)},
		map[string]string{"a/a.go": pkg("a", bodyFormat)})
	r := compare(t, old, new)
	wantClass(t, r, "a.Scan", Deleted)
	wantClass(t, r, "a.Format", Added)
}

// TestClassesArePartition is the structural invariant every fixture rests on:
// each function on each side appears in exactly one finding.
func TestClassesArePartition(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyPipeline, bodyScan, bodyFormat)},
		map[string]string{
			"a/a.go": pkg("a", bodyPipelineSum, bodyPipelineJoin, bodyScanEdited),
			"c/c.go": pkg("c", bodyFormat),
		})
	r := compare(t, old, new)
	seenOld, seenNew := map[string]int{}, map[string]int{}
	for _, c := range r.Changes {
		for _, m := range c.Old {
			seenOld[m.Key]++
		}
		for _, m := range c.New {
			seenNew[m.Key]++
		}
	}
	for _, u := range old.Units {
		if seenOld[u.Key] != 1 {
			t.Errorf("old %s appears in %d findings, want 1\n%s", u.Key, seenOld[u.Key], render(r))
		}
	}
	for _, u := range new.Units {
		if seenNew[u.Key] != 1 {
			t.Errorf("new %s appears in %d findings, want 1\n%s", u.Key, seenNew[u.Key], render(r))
		}
	}
	if len(seenOld) != len(old.Units) || len(seenNew) != len(new.Units) {
		t.Errorf("finding sides cover %d old and %d new, want %d and %d",
			len(seenOld), len(seenNew), len(old.Units), len(new.Units))
	}
}

// TestSymmetry checks that the label population really is symmetric: swapping
// the two snapshots must invert every class rather than change the matching.
func TestSymmetry(t *testing.T) {
	old, new := sides(t,
		map[string]string{"a/a.go": pkg("a", bodyScan, bodyFormat)},
		map[string]string{"b/b.go": pkg("b", strings.Replace(bodyScanEdited, "func Scan(", "func Total(", 1))})
	fwd := compare(t, old, new)
	rev := compare(t, new, old)
	if got := find(t, fwd, "a.Scan"); got.Class != Moved {
		t.Fatalf("forward: a.Scan is %s, want moved\n%s", got.Class, render(fwd))
	}
	if got := find(t, rev, "b.Total"); got.Class != Moved {
		t.Fatalf("reverse: b.Total is %s, want moved\n%s", got.Class, render(rev))
	}
	f, rv := find(t, fwd, "a.Scan"), find(t, rev, "b.Total")
	if f.Jaccard != rv.Jaccard {
		t.Errorf("jaccard is not symmetric: %v forward, %v reverse", f.Jaccard, rv.Jaccard)
	}
}

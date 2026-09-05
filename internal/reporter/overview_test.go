package reporter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
)

func sampleOverview() *Overview {
	return &Overview{
		Functions: 314,
		Packages:  16,
		TestsMode: "exclude",
		Threshold: 0.60,
		Roles:     []RoleRow{{Role: "leaf", Count: 181}, {Role: "utility", Count: 88}},
		Concepts: []TagRow{
			{Tag: "error_wrapping", Count: 6, Convention: 0.40, Prototyped: true},
			{Tag: "mapping", Count: 3},
		},
		Absent: []string{"db_access", "http_call"},
		SeedMap: []TaxonomyNode{
			{ID: "concept", Abstract: true},
			{ID: "io_operation", Parent: "concept", Abstract: true},
			{ID: "http_call", Parent: "io_operation"},
			{ID: "file_io", Parent: "io_operation", Count: 12},
		},
		Taxonomy: []TaxonomyNode{
			{ID: "concept", Abstract: true},
			{ID: "io_operation", Parent: "concept", Abstract: true},
			{ID: "http_call", Parent: "io_operation"},
			{ID: "mapping", Parent: "concept", Count: 3},
		},
		Habitats: []HabitatRow{
			{Package: "snapshot", Functions: 20, Norm: 0.74, Misfits: 6},
			{Package: "bench", Functions: 18, Norm: 0.96},
		},
		MostUniform: "bench", MostUniformNorm: 0.96,
		MostDiverse: "snapshot", MostDiverseNorm: 0.74,
		Misfits:       8,
		ArenaProfiled: 181, ArenaDominance: 181,
		ShapePairs: 157, ConceptPairs: 51, CallPairs: 709, UnionPairs: 865,
		OnlyCallPairs: 659, OnlyConceptPairs: 35,
		Links:   []PackageLink{{A: "culture", B: "mapper", Weight: 3}},
		SelfDup: map[string]float64{"culture": 18},
	}
}

func TestPrintMarkdownOverview(t *testing.T) {
	var b strings.Builder
	PrintMarkdownOverview(&b, sampleOverview())
	out := b.String()

	if !strings.Contains(out, "## What doppel sees") {
		t.Errorf("section heading missing:\n%s", out)
	}
	if !strings.Contains(out, "**314 functions** across **16 packages** — test functions excluded.") {
		t.Errorf("corpus sentence missing or malformed:\n%s", out)
	}
	// The absent line is the one most likely to change a decision. It names
	// seeds that grew nothing, not learned concepts, which cannot be absent.
	if !strings.Contains(out, "**No practice here for** `db_access`, `http_call`") {
		t.Errorf("unused seeds not stated:\n%s", out)
	}
	// The retrieval mix is what tells a reader why these pairs and not others.
	if !strings.Contains(out, "**865 candidate pairs** (shape 157, concept 51, call 709)") {
		t.Errorf("retrieval mix missing:\n%s", out)
	}
	if !strings.Contains(out, "Most uniform is `bench`") {
		t.Errorf("habitat superlatives missing:\n%s", out)
	}
	if n := strings.Count(out, "```mermaid"); n != 4 {
		t.Errorf("got %d diagrams, want 4 (seed map, learned concepts, duplication, habitats):\n%s", n, out)
	}
}

// A caller that builds no overview must get exactly the document it got before
// this section existed.
func TestNilOverviewRendersNothing(t *testing.T) {
	var b strings.Builder
	PrintMarkdownOverview(&b, nil)
	if got := b.String(); got != "" {
		t.Errorf("nil overview rendered %q, want nothing", got)
	}

	var zero strings.Builder
	PrintMarkdownOverview(&zero, &Overview{})
	if got := zero.String(); got != "" {
		t.Errorf("zero overview rendered %q, want nothing", got)
	}

	// And through the report itself: a Meta without an Overview is unchanged.
	var with, without strings.Builder
	pair := samplePair(nil)
	PrintMarkdown(&with, []analyzer.SimilarPair{pair}, sampleUnits, Meta{})
	PrintMarkdown(&without, []analyzer.SimilarPair{pair}, sampleUnits, Meta{Overview: nil})
	if with.String() != without.String() {
		t.Error("an absent overview changed the report bytes")
	}
	if strings.Contains(with.String(), "What doppel sees") {
		t.Error("the overview section rendered without an Overview")
	}
}

// Bounded diagrams must say what they dropped: a graph that silently shows 12
// of 168 packages reads as "this is all of them".
func TestOverviewDisclosesBounds(t *testing.T) {
	ov := sampleOverview()
	ov.HabitatsMore = 156
	ov.LinksMore = 40

	var b strings.Builder
	PrintMarkdownOverview(&b, ov)
	out := b.String()

	if !strings.Contains(out, "_156 further packages are modeled and not drawn._") {
		t.Errorf("dropped habitats not disclosed:\n%s", out)
	}
	if !strings.Contains(out, "40 further package pairs") {
		t.Errorf("dropped links not disclosed:\n%s", out)
	}
}

// Absent concepts are the finding in the taxonomy diagram, so they carry the
// only colour in it.
func TestTaxonomyDiagramMarksAbsentConcepts(t *testing.T) {
	var b strings.Builder
	PrintMarkdownOverview(&b, sampleOverview())
	out := b.String()

	if !strings.Contains(out, `c2["http_call<br/>absent"]`) {
		t.Errorf("absent leaf not marked in the diagram:\n%s", out)
	}
	if !strings.Contains(out, `c3["mapping<br/>3"]`) {
		t.Errorf("present leaf missing its count:\n%s", out)
	}
	if !strings.Contains(out, "class c2 hot") {
		t.Errorf("absent leaf not coloured:\n%s", out)
	}
	// Abstract nodes are scaffolding, not findings — stadium shape, no count.
	if !strings.Contains(out, `c0(["concept"])`) {
		t.Errorf("abstract node not rendered as abstract:\n%s", out)
	}
}

// <br/> is markup this package emits, not content. Escaping a composed label
// turns the line break into a visible #lt;br/#gt; — which shipped once.
func TestDiagramLabelsKeepTheirLineBreaks(t *testing.T) {
	var b strings.Builder
	PrintMarkdownOverview(&b, sampleOverview())
	if out := b.String(); strings.Contains(out, "#lt;br") {
		t.Errorf("a composed label was escaped whole:\n%s", out)
	}
}

func TestOverviewIsDeterministic(t *testing.T) {
	var first strings.Builder
	PrintMarkdownOverview(&first, sampleOverview())
	for i := 0; i < 20; i++ {
		var b strings.Builder
		PrintMarkdownOverview(&b, sampleOverview())
		if b.String() != first.String() {
			t.Fatalf("run %d differed from the first", i)
		}
	}
}

func TestSortHabitatsPutsTheLeastUniformFirst(t *testing.T) {
	rows := []HabitatRow{
		{Package: "b", Norm: 0.9},
		{Package: "a", Norm: 0.4},
		{Package: "c", Norm: 0.9},
	}
	got, more := SortHabitats(rows)
	if more != 0 {
		t.Errorf("more = %d, want 0", more)
	}
	// Worst first, because that is where attention is worth spending; ties on
	// the package name so the order is total.
	if got[0].Package != "a" || got[1].Package != "b" || got[2].Package != "c" {
		t.Errorf("order = %v", got)
	}
}

func TestSortHabitatsBounds(t *testing.T) {
	var rows []HabitatRow
	for i := 0; i < maxOverviewNodes+5; i++ {
		rows = append(rows, HabitatRow{Package: string(rune('a' + i)), Norm: 0.5})
	}
	got, more := SortHabitats(rows)
	if len(got) != maxOverviewNodes || more != 5 {
		t.Errorf("got %d rows and %d dropped, want %d and 5", len(got), more, maxOverviewNodes)
	}
}

// The habitat tail is four optional clauses. Appending each with its own
// trailing space left a double space wherever a middle one was skipped, which
// is exactly what happens on a corpus whose misfits are all excused.
func TestHabitatTailJoinsCleanly(t *testing.T) {
	ov := sampleOverview()
	ov.Misfits = 0
	ov.MisfitsExcused = 7
	ov.HabitatsMore = 0

	var b strings.Builder
	PrintMarkdownOverview(&b, ov)
	out := b.String()

	if strings.Contains(out, "`).  A further") {
		t.Errorf("double space between clauses:\n%s", out)
	}
	// Zero misfits and seven excused is a different codebase from neither, and
	// the bare count cannot say so.
	if !strings.Contains(out, "A further 7 fit poorly in their package but match the wider subsystem") {
		t.Errorf("excused misfits not disclosed:\n%s", out)
	}
	if strings.Contains(out, "0 functions are alien") {
		t.Errorf("an empty misfit clause was rendered:\n%s", out)
	}
}

// The taxonomy diagram is a tree, so it is bounded per branch rather than
// globally: a learned vocabulary hangs hundreds of leaves off eight authored
// parents, and a global top-N would draw one crowded branch and seven bare
// ones. moby learns 519 concepts, which unbounded is a 527-node picture.
func TestBoundTaxonomyKeepsTheLargestLeafPerBranch(t *testing.T) {
	nodes := []TaxonomyNode{
		{ID: "concept", Abstract: true},
		{ID: "io_operation", Parent: "concept", Abstract: true},
		{ID: "data_transformation", Parent: "concept", Abstract: true},
	}
	for i := 0; i < maxTaxonomyLeaves+4; i++ {
		nodes = append(nodes, TaxonomyNode{ID: fmt.Sprintf("io%d", i), Parent: "io_operation", Count: i})
	}
	nodes = append(nodes, TaxonomyNode{ID: "lonely", Parent: "data_transformation", Count: 1})

	got, more := BoundTaxonomy(nodes)
	if more != 4 {
		t.Errorf("dropped %d leaves, want 4", more)
	}
	if len(got) != 3+maxTaxonomyLeaves+1 {
		t.Fatalf("kept %d nodes, want %d", len(got), 3+maxTaxonomyLeaves+1)
	}
	// Every abstract node survives, and the thin branch keeps its one leaf:
	// the interior is the map, and a branch drawn empty says something false.
	var kept []string
	for _, n := range got {
		kept = append(kept, n.ID)
	}
	want := []string{"concept", "io_operation", "data_transformation", "io4", "io5", "io6", "lonely"}
	if strings.Join(kept, ",") != strings.Join(want, ",") {
		t.Errorf("kept %v, want %v", kept, want)
	}
}

// What is dropped is counted in the prose, the rule every other diagram in
// this package follows.
func TestTaxonomyDiagramReportsWhatItLeftOut(t *testing.T) {
	ov := sampleOverview()
	ov.TaxonomyMore = 507
	var b strings.Builder
	PrintMarkdownOverview(&b, ov)
	if out := b.String(); !strings.Contains(out, "**507 further concepts**") {
		t.Errorf("bounded diagram does not say what it dropped:\n%s", out)
	}
}

// The section carries two concept diagrams and they must be told apart. Ids
// are per-block so they could collide harmlessly, but a report where c4 names
// one concept in the first picture and another in the second cannot be checked
// against the corpus by whoever has to check it.
func TestConceptDiagramsUseDistinctNodeIDs(t *testing.T) {
	var b strings.Builder
	PrintMarkdownOverview(&b, sampleOverview())
	out := b.String()
	for _, want := range []string{`s0(["concept"])`, `s3["file_io<br/>12"]`, `c0(["concept"])`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s — the two concept diagrams share a prefix:\n%s", want, out)
		}
	}
}

// The seed map is the picture the report opened with before the vocabulary
// became corpus-derived: the authored tree, absent leaves in red.
func TestSeedMapDrawsAbsentSeedsInRed(t *testing.T) {
	var b strings.Builder
	PrintMarkdownOverview(&b, sampleOverview())
	out := b.String()
	if !strings.Contains(out, `s2["http_call<br/>absent"]`) {
		t.Errorf("a seed that grew nothing is not marked absent:\n%s", out)
	}
	if !strings.Contains(out, "class s2 hot") {
		t.Errorf("an absent seed is not coloured:\n%s", out)
	}
}

// The map is one picture read three ways, so the numbers on it must say which
// reading they came from: a reader holding two reports must not be able to
// compare an edge weight from one against an edge weight from the other
// without noticing they are different quantities.
func TestDuplicationMapMetricIsStated(t *testing.T) {
	for _, tc := range []struct {
		metric MapMetric
		edge   string // the weight as it must appear on the edge
		self   string // and on the node
		noun   string
	}{
		{MapMergeWorthy, "3", "18", "merge-worthy pairs"},
		{MapPairs, "3", "18", "candidate pairs"},
		{MapEvidence, "3.00", "18.00", "corroborated evidence"},
	} {
		ov := sampleOverview()
		ov.Metric = tc.metric

		var b strings.Builder
		PrintMarkdownOverview(&b, ov)
		out := b.String()

		if !strings.Contains(out, "Weights are **"+tc.noun+"**") {
			t.Errorf("%s: metric not named in the prose:\n%s", tc.metric, out)
		}
		if !strings.Contains(out, `---|"`+tc.edge+`"|`) {
			t.Errorf("%s: edge weight %q missing:\n%s", tc.metric, tc.edge, out)
		}
		if !strings.Contains(out, tc.self+" internal") {
			t.Errorf("%s: self-duplication weight %q missing:\n%s", tc.metric, tc.self, out)
		}
	}
}

// An Overview built before the metric existed — or by a caller that never set
// one — draws the map it always drew.
func TestZeroMapMetricIsMergeWorthy(t *testing.T) {
	var chosen, unset strings.Builder
	ov := sampleOverview()
	ov.Metric = MapMergeWorthy
	PrintMarkdownOverview(&chosen, ov)
	PrintMarkdownOverview(&unset, sampleOverview()) // Metric is the zero value
	if chosen.String() != unset.String() {
		t.Error("an unset metric drew a different map than the default")
	}
	if got := MapMetric("nonsense").Resolve(); got != DefaultMapMetric {
		t.Errorf("unknown metric resolved to %q, want the default", got)
	}
}

// A bad metric names the alternatives rather than silently drawing one.
func TestParseMapMetric(t *testing.T) {
	for _, m := range MapMetrics() {
		if got, err := ParseMapMetric(string(m)); err != nil || got != m {
			t.Errorf("ParseMapMetric(%q) = %q, %v", m, got, err)
		}
	}
	if got, err := ParseMapMetric(""); err != nil || got != DefaultMapMetric {
		t.Errorf(`ParseMapMetric("") = %q, %v; want the default`, got, err)
	}
	err := func() error { _, err := ParseMapMetric("mergeworthy"); return err }()
	if err == nil {
		t.Fatal("an unknown metric was accepted")
	}
	for _, m := range MapMetrics() {
		if !strings.Contains(err.Error(), string(m)) {
			t.Errorf("error %q does not name %q", err, m)
		}
	}
}

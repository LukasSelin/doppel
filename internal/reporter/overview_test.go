package reporter

import (
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
		Links:   []PackageLink{{A: "culture", B: "mapper", Pairs: 3}},
		SelfDup: map[string]int{"culture": 18},
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
	if n := strings.Count(out, "```mermaid"); n != 3 {
		t.Errorf("got %d diagrams, want 3 (concepts, duplication, habitats):\n%s", n, out)
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
	PrintMarkdown(&with, []analyzer.SimilarPair{pair}, Meta{})
	PrintMarkdown(&without, []analyzer.SimilarPair{pair}, Meta{Overview: nil})
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

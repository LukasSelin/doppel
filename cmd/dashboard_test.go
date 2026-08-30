package cmd

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/dashboard"
	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

func unit(pkg, name, file string, line int, body string, tags ...string) parser.CodeUnit {
	return parser.CodeUnit{
		Name: name, Package: pkg, File: file, StartLine: line, Body: body,
		Patterns: tags, Fingerprint: fingerprint.Fingerprint{Nodes: 20},
	}
}

func pair(units []parser.CodeUnit, a, b int, shape, total float64) analyzer.SimilarPair {
	ev := comparator.StructuralEvidence{OverlapScore: 0.5}
	return analyzer.SimilarPair{
		A: units[a], B: units[b], AIdx: a, BIdx: b, Score: shape,
		Breakdown: fingerprint.Breakdown{AST: shape, Flow: 1, Depth: 1, Signature: 1, SizeRatio: 0.9},
		Evidence:  &ev,
		Retrieval: &analyzer.Retrieval{Total: total, TrophicSim: 1, Channels: []string{"shape"}},
	}
}

func sampleResult() Result {
	units := []parser.CodeUnit{
		unit("beta", "Zeta", "beta/z.go", 1, "func Zeta() {}", "retry"),
		unit("alpha", "Alpha", "alpha/a.go", 1, "func Alpha() {}", "db_access", "retry"),
		unit("alpha", "Beta", "alpha/a.go", 20, "func Beta() {}"),
	}
	res := Result{
		Root:   ".",
		Params: Params{Threshold: 0.6, StructMin: 0.4, TestsMode: "exclude", Generated: "exclude"},
		Units:  units,
		Docs: []concepter.ConceptDoc{
			{Role: "leaf"}, {Role: "utility"}, {Role: "leaf"},
		},
		Graph: &concepter.Graph{
			Callers: map[string][]string{"alpha.Alpha": {"beta.Zeta", "alpha.Beta"}},
			Callees: map[string][]string{"beta.Zeta": {"alpha.Alpha"}},
		},
	}
	res.Pairs = []analyzer.SimilarPair{
		pair(units, 1, 2, 0.70, 10), // lower rank
		pair(units, 0, 1, 0.90, 90), // higher rank
	}
	return res
}

func TestBuildDashboardWiresTheCorpus(t *testing.T) {
	p := buildDashboard(sampleResult(), nil, nil, familyStatsZero(), nil, 0)

	if p.Schema != dashboard.Schema {
		t.Errorf("Schema = %d, want %d", p.Schema, dashboard.Schema)
	}
	if len(p.Units) != 3 {
		t.Fatalf("Units = %d, want 3", len(p.Units))
	}
	// Units[i].ID == i is the payload's whole identity scheme.
	for i, u := range p.Units {
		if u.ID != i {
			t.Errorf("Units[%d].ID = %d, want %d", i, u.ID, i)
		}
	}
	if p.Units[1].Key != "alpha.Alpha" {
		t.Errorf("Key = %q, want alpha.Alpha", p.Units[1].Key)
	}
	if p.Units[1].Role != "utility" {
		t.Errorf("Role = %q, want utility", p.Units[1].Role)
	}
	if p.Units[1].FanIn != 2 || p.Units[1].FanOut != 0 {
		t.Errorf("fan-in/out = %d/%d, want 2/0", p.Units[1].FanIn, p.Units[1].FanOut)
	}
	if p.Units[0].FanOut != 1 {
		t.Errorf("beta.Zeta fan-out = %d, want 1", p.Units[0].FanOut)
	}
	// No culture model, so the concept falls back to the first tag.
	if p.Units[1].Concept != "db_access" {
		t.Errorf("Concept = %q, want db_access (first tag)", p.Units[1].Concept)
	}
	if p.Units[2].Concept != "" {
		t.Errorf("untagged unit got concept %q", p.Units[2].Concept)
	}
	// Habitat fit is -1, not 0, when nothing modeled it: 0 is a real fit.
	if p.Units[0].Fit != -1 {
		t.Errorf("Fit = %v, want -1 for an unmodeled unit", p.Units[0].Fit)
	}
}

func TestBuildDashboardSortsPackagesAndConcepts(t *testing.T) {
	p := buildDashboard(sampleResult(), nil, nil, familyStatsZero(), nil, 0)

	if len(p.Packages) != 2 || p.Packages[0].Name != "alpha" || p.Packages[1].Name != "beta" {
		t.Errorf("packages not sorted by name: %+v", p.Packages)
	}
	if p.Packages[0].Functions != 2 || p.Packages[1].Functions != 1 {
		t.Errorf("package counts wrong: %+v", p.Packages)
	}
	if p.Packages[0].Norm != -1 {
		t.Errorf("Norm = %v, want -1 for an unmodeled package", p.Packages[0].Norm)
	}
	want := []string{"db_access", "retry"}
	if len(p.Concepts) != len(want) {
		t.Fatalf("Concepts = %v, want %v", p.Concepts, want)
	}
	for i := range want {
		if p.Concepts[i] != want[i] {
			t.Errorf("Concepts = %v, want %v", p.Concepts, want)
		}
	}
}

// Edges lead with the best-corroborated pair, so every per-unit neighbour list
// in the page inherits that order without a second sort.
func TestBuildDashboardRanksEdges(t *testing.T) {
	p := buildDashboard(sampleResult(), nil, nil, familyStatsZero(), nil, 0)

	if len(p.Edges) != 2 {
		t.Fatalf("Edges = %d, want 2", len(p.Edges))
	}
	if p.Edges[0].Rank <= p.Edges[1].Rank {
		t.Errorf("edges not rank-descending: %v then %v", p.Edges[0].Rank, p.Edges[1].Rank)
	}
	e := p.Edges[0]
	if e.A != 0 || e.B != 1 {
		t.Errorf("top edge = %d-%d, want 0-1", e.A, e.B)
	}
	if !e.Cross {
		t.Error("a pair spanning alpha and beta should be marked cross-package")
	}
	if e.Shape != 0.90 || e.Overlap != 0.5 || e.Total != 90 {
		t.Errorf("scores not carried: %+v", e)
	}
	if e.Breakdown[0] != 0.90 || e.Breakdown[2] != 1 {
		t.Errorf("breakdown order wrong: %v", e.Breakdown)
	}
	if p.Edges[1].Cross {
		t.Error("a pair wholly inside alpha should not be cross-package")
	}
}

// A < B always, so the page never has to normalise an endpoint pair.
func TestBuildDashboardNormalisesEndpointOrder(t *testing.T) {
	res := sampleResult()
	res.Pairs[0].AIdx, res.Pairs[0].BIdx = 2, 1 // deliberately reversed
	p := buildDashboard(res, nil, nil, familyStatsZero(), nil, 0)
	for _, e := range p.Edges {
		if e.A >= e.B {
			t.Errorf("edge %d-%d is not in ascending endpoint order", e.A, e.B)
		}
	}
}

func TestDashboardBodiesFollowRankAndReportTheBound(t *testing.T) {
	res := sampleResult()
	p := buildDashboard(res, nil, nil, familyStatsZero(), nil, 0)

	if len(p.Bodies) != 3 || p.Facts.BodiesOmitted != 0 {
		t.Fatalf("Bodies = %d, omitted = %d, want 3 and 0", len(p.Bodies), p.Facts.BodiesOmitted)
	}
	// Emitted in unit order, whatever order they were admitted in.
	for i, b := range p.Bodies {
		if b.Unit != i {
			t.Errorf("Bodies[%d].Unit = %d, want %d", i, b.Unit, i)
		}
	}
}

// The bound falls on the least corroborated pairs first, and says how many it
// dropped rather than leaving a reader to wonder why a body is missing.
func TestDashboardBodiesBudgetDropsLowestRankedFirst(t *testing.T) {
	res := sampleResult()
	// Sized so two fit and the third does not.
	big := strings.Repeat("x", maxBodyBytes/3+1)
	for i := range res.Units {
		res.Units[i].Body = big
	}
	p := buildDashboard(res, nil, nil, familyStatsZero(), nil, 0)

	if len(p.Bodies) != 2 || p.Facts.BodiesOmitted != 1 {
		t.Fatalf("Bodies = %d, omitted = %d, want 2 and 1", len(p.Bodies), p.Facts.BodiesOmitted)
	}
	// The top edge is 0-1, so unit 2 — reachable only through the weaker pair —
	// is the one that loses its body.
	for _, b := range p.Bodies {
		if b.Unit == 2 {
			t.Error("the lowest-ranked function kept its body while a higher-ranked one was dropped")
		}
	}
}

func TestDashboardBodiesSkipUnpairedFunctions(t *testing.T) {
	res := sampleResult()
	res.Units = append(res.Units, unit("gamma", "Lonely", "gamma/g.go", 1, "func Lonely() {}"))
	res.Docs = append(res.Docs, concepter.ConceptDoc{Role: "leaf"})
	p := buildDashboard(res, nil, nil, familyStatsZero(), nil, 0)

	for _, b := range p.Bodies {
		if b.Unit == 3 {
			t.Error("a function in no pair has no neighbourhood, so its body is weight")
		}
	}
	if p.Facts.BodiesOmitted != 0 {
		t.Errorf("BodiesOmitted = %d; never-wanted bodies are not omissions", p.Facts.BodiesOmitted)
	}
}

func TestBuildDashboardSurvivesANilOverview(t *testing.T) {
	// --output is what builds the Overview; a caller without one must still
	// get a page rather than a panic.
	p := buildDashboard(sampleResult(), nil, nil, familyStatsZero(), nil, 0)
	if p.Facts.Functions != 3 || p.Facts.Packages != 0 {
		t.Errorf("facts without an overview: %+v", p.Facts)
	}
	if p.Facts.Threshold != 0.6 || p.Facts.TestsMode != "excluded" {
		t.Errorf("params did not reach the facts: %+v", p.Facts)
	}
}

func TestBuildDashboardRendersEndToEnd(t *testing.T) {
	p := buildDashboard(sampleResult(), nil, nil, familyStatsZero(), nil, 0)
	var b strings.Builder
	if err := dashboard.Print(&b, p); err != nil {
		t.Fatalf("Print: %v", err)
	}
	if !strings.Contains(b.String(), "alpha.Alpha") {
		t.Error("the page does not name the corpus it describes")
	}
}

func TestIsHTMLPath(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"report.html", true}, {"report.htm", true}, {"REPORT.HTML", true},
		{"a/b/c.Html", true}, {"report.md", false}, {"report", false},
		{"report.html.md", false}, {"", false},
	} {
		if got := isHTMLPath(tc.path); got != tc.want {
			t.Errorf("isHTMLPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func familyStatsZero() family.Stats { return family.Stats{} }

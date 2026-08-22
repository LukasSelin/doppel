package reporter

import (
	"strings"
	"testing"
)

func sampleHTMLReport() HTMLReport {
	return HTMLReport{
		Target:    "moby",
		Functions: 7644,
		Packages:  232,
		Threshold: 0.60,
		TestsMode: "excluded",

		PairsFound:     10,
		CandidatePairs: 17471,
		ShapePairs:     3983,
		ConceptPairs:   2410,
		CallPairs:      12442,
		CallOnlyPct:    64,
		ConceptOnlyPct: 13,

		FamilyCount:   656,
		FamilyFuncs:   1522,
		FamilyLargest: 44,
		EdgesScored:   2814,
		MisfitCount:   101,
		MisfitExcused: 152,

		Strips: []HTMLStrip{{
			Title:    "Family 1",
			File:     "libnetwork/networkdb/networkdbdiagnostic.go",
			Tag:      "logging",
			Note:     "10 functions declared one after another in a single file.",
			MinLabel: "0.68",
			Members: []HTMLStripMember{
				{Name: "dbJoin", Line: 38, Pct: 47, SpanLabel: "33 lines", HasSpan: true},
				{Name: "dbNetworkStats", Line: 420, SpanLabel: "span unknown — last in file"},
			},
			Pairs: []HTMLPairCard{{
				Label: "#1", A: "networkdb.dbCreateEntry", B: "networkdb.dbUpdateEntry",
				ShapeLabel: "0.9535", OverlapLabel: "0.79",
				Components: []HTMLComponent{
					{Name: "ast", Pct: 92, Label: "0.92", Hot: true},
					{Name: "size", Pct: 60, Label: "0.60"},
				},
				Footer: "19 shared callees · neighbourhood 0.91",
			}},
		}},
		Taxonomy: []HTMLTaxon{
			{Label: "concept", Abstract: true},
			{Label: "grpc_call", Indent: 60, Absent: true, CountLabel: "absent"},
			{Label: "caching", Indent: 60, CountLabel: "145", ConvPct: 57},
		},
		AbsentConcepts: []string{"circuit_breaker", "grpc_call"},
		Habitats:       []HTMLHabitat{{Name: "vfs", NormPct: 56, MisfitPct: 0, Meta: "27 fn"}},
		HabitatsMore:   154,
		MostUniform:    "checker", MostUniformN: 0.98,
		MostVaried: "vfs", MostVariedN: 0.56,
		Drift: []DriftRow{
			{Name: "client.*Client.Events", File: "client/events.go", Line: 18,
				Tag: "concurrency", Typicality: 0.14, Median: 0.34, Unpaired: true},
		},
		DriftMore: 103,
		Families: []HTMLFamily{{
			N: 1, What: "interface implementations", Where: "libnetwork", Tag: "logging",
			Members: 10, MinLabel: "0.68", EvidenceLabel: "41964", AddedLabel: "+4",
		}},
		FamiliesMore: 651,
	}
}

func renderHTML(t *testing.T, r HTMLReport) string {
	t.Helper()
	var b strings.Builder
	if err := PrintHTML(&b, r); err != nil {
		t.Fatalf("PrintHTML: %v", err)
	}
	return b.String()
}

// The point of rendering in Go rather than shipping the canvas prototype is
// that the result is one file: no runtime, no fetch, nothing to serve. A report
// someone can mail to a colleague has to open from file://.
func TestHTMLIsSelfContained(t *testing.T) {
	out := renderHTML(t, sampleHTMLReport())

	if strings.Contains(out, "<script") {
		t.Error("the report carries a script; it must render without one")
	}
	if strings.Contains(out, "src=") || strings.Contains(out, `href="`) {
		t.Errorf("the report references an external file:\n%s", firstMatch(out, "src=", `href="`))
	}
	// The design system's font import is the one permitted outbound reference,
	// and the token stack falls back to system-ui without it.
	if n := strings.Count(out, "https://"); n != 1 {
		t.Errorf("got %d external URLs, want exactly the font import", n)
	}
	if !strings.Contains(out, "fonts.googleapis.com") {
		t.Error("the font import went missing")
	}
	// The stylesheet is inlined, not linked.
	if !strings.Contains(out, "--color-accent: #0088b0") {
		t.Error("the Broadsheet tokens were not inlined")
	}
}

func TestHTMLRendersEachSection(t *testing.T) {
	out := renderHTML(t, sampleHTMLReport())

	for _, want := range []string{
		"Where this codebase repeats itself",
		"Repetition has a silhouette",
		"What this codebase does",
		"How settled each package is",
		"Carries the tag, looks nothing like it",
		"Families, not pairs",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("section %q missing", want)
		}
	}
	// The headline figure is the plate numeral, and every plate repeat has to
	// carry the same text or the misregistration reads as a typo.
	if n := strings.Count(out, "7,644"); n != 4 {
		t.Errorf("plate numeral repeated %d times, want 4 (paper + three plates)", n)
	}
}

// Every list in the design carries its own remainder line. They have to be
// wired to real counts, or a bounded view reads as a complete one.
func TestHTMLDisclosesRemainders(t *testing.T) {
	out := renderHTML(t, sampleHTMLReport())

	for _, want := range []string{
		"651 further families are not listed",
		"103 further unusual realisations are not listed",
		"A further 152 functions fit poorly",
		"of 154 more modelled",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("remainder %q missing:\n", want)
		}
	}
}

// Names come from analysed source, so the escaping is not optional. This is
// what html/template buys and why the renderer does not concatenate markup.
func TestHTMLEscapesAnalysedNames(t *testing.T) {
	r := sampleHTMLReport()
	r.Target = `moby<script>alert("x")</script>`
	r.Strips[0].Members[0].Name = `Foo<&>"Bar"`
	r.Families[0].Where = "pkg<b>"

	out := renderHTML(t, r)

	if strings.Contains(out, "<script>alert") {
		t.Error("a script tag survived from an analysed name")
	}
	if !strings.Contains(out, "Foo&lt;&amp;&gt;") {
		t.Errorf("member name not escaped:\n%s", firstMatch(out, "Foo"))
	}
	if strings.Contains(out, "pkg<b>") {
		t.Error("package name not escaped")
	}
}

// A run with nothing to show still has to produce a page rather than panic or
// emit half a document.
func TestHTMLZeroReportIsStillAPage(t *testing.T) {
	out := renderHTML(t, HTMLReport{})

	if !strings.HasPrefix(out, "<!DOCTYPE html>") || !strings.Contains(out, "</html>") {
		t.Errorf("zero report did not render a whole document:\n%s", out)
	}
	// The optional sections drop out entirely rather than rendering empty
	// tables, the same rule the markdown report follows.
	for _, gone := range []string{"Repetition has a silhouette", "Families, not pairs", "Carries the tag"} {
		if strings.Contains(out, gone) {
			t.Errorf("section %q rendered with no data", gone)
		}
	}
}

func TestHTMLIsDeterministic(t *testing.T) {
	first := renderHTML(t, sampleHTMLReport())
	for i := 0; i < 10; i++ {
		if got := renderHTML(t, sampleHTMLReport()); got != first {
			t.Fatalf("render %d differed from the first", i)
		}
	}
}

func TestComma(t *testing.T) {
	for _, tt := range []struct {
		in   int
		want string
	}{{0, "0"}, {999, "999"}, {1000, "1,000"}, {7644, "7,644"}, {1234567, "1,234,567"}, {-4321, "-4,321"}} {
		if got := comma(tt.in); got != tt.want {
			t.Errorf("comma(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// firstMatch returns a window around the first of any needle, for error
// messages that point at the problem rather than dumping 50KB of markup.
func firstMatch(s string, needles ...string) string {
	for _, n := range needles {
		if i := strings.Index(s, n); i >= 0 {
			end := i + 120
			if end > len(s) {
				end = len(s)
			}
			return s[i:end]
		}
	}
	return ""
}

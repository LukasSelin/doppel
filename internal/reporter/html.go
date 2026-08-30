package reporter

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"strings"
)

// broadsheetCSS is the vendored design system, inlined into every report so the
// file opens from file:// with nothing to fetch and nothing to serve.
//
//go:embed broadsheet.css
var broadsheetCSS string

// HTMLReport is one run rendered as the Similarity Report page.
//
// It mirrors the data contract of the design's own doppel-run.json rather than
// inventing a second shape, so the canvas prototype and this renderer stay
// comparable. Everything here is presorted and rendered-ready: the template
// does no arithmetic beyond what a Go template can express, and every
// percentage is computed here where it can be tested.
type HTMLReport struct {
	Target    string
	Functions int
	Packages  int
	Threshold float64
	TestsMode string

	PairsFound     int
	PairsHeldBack  int
	CandidatePairs int
	ShapePairs     int
	ConceptPairs   int
	CallPairs      int
	CallOnlyPct    int
	ConceptOnlyPct int

	ArenasSettled      int
	ArenaSingle        int
	ArenaCoalition     int
	ArenaContradictory int

	FamilyCount    int
	FamilyFuncs    int
	FamilyLargest  int
	EdgesScored    int
	MisfitCount    int
	MisfitExcused  int
	Roles          []RoleRow
	Strips         []HTMLStrip
	Taxonomy       []HTMLTaxon
	AbsentConcepts []string
	Habitats       []HTMLHabitat
	HabitatsMore   int
	MostUniform    string
	MostUniformN   float64
	MostVaried     string
	MostVariedN    float64
	Drift          []DriftRow
	DriftMore      int
	Families       []HTMLFamily
	FamiliesMore   int

	// Metrics carries the two T10 corpus-health numbers. HasMetrics gates the
	// section: the zero CorpusMetrics of a report built with no Overview
	// must render nothing, exactly as the markdown preamble does.
	Metrics    CorpusMetrics
	HasMetrics bool
}

// HTMLStrip is one family whose members all live in a single file, drawn as a
// column of bars at declaration length.
//
// The strip only exists for same-file families: the bar is the gap to the next
// declaration, which means nothing across files. The design's own caption says
// the bar is a reading aid rather than a measurement, and this is why.
type HTMLStrip struct {
	Title    string
	File     string
	Tag      string
	Note     string
	MinLabel string
	Members  []HTMLStripMember
	Pairs    []HTMLPairCard
}

// HTMLStripMember is one function in a strip.
type HTMLStripMember struct {
	Name      string
	Line      int
	Pct       int    // bar width, 0 when the span is unknown
	SpanLabel string // "33 lines", or why there is no bar
	HasSpan   bool
}

// HTMLPairCard is one reported pair, shown under the strip that produced it.
type HTMLPairCard struct {
	Label      string
	A, B       string
	ShapeLabel string
	// ContainmentLabel sits between the two scores in the card header
	// because that is what it is: a third reported quantity about the pair,
	// blended into neither. Shape divides shared structural information by
	// the pair's union, containment by the smaller side alone, so a helper
	// inlined into a long function reads low on the first and high on the
	// second.
	ContainmentLabel string
	OverlapLabel     string
	Components       []HTMLComponent
	Footer           string
}

// HTMLComponent is one of the fingerprint components behind a pair's
// code-shape score, plus the reported-but-unscored size ratio.
type HTMLComponent struct {
	Name  string
	Pct   int
	Label string
	Hot   bool // >= 0.90: the design tints these with the second accent
}

// HTMLTaxon is one concept term positioned in the taxonomy.
type HTMLTaxon struct {
	Label      string
	Indent     int
	Abstract   bool
	Absent     bool
	CountLabel string
	ConvPct    int
	Loose      bool
}

// HTMLHabitat is one modeled package.
type HTMLHabitat struct {
	Name      string
	NormPct   int
	MisfitPct int
	Meta      string
}

// HTMLFamily is one row of the census.
//
// What holds the design's editorial `what` column. doppel cannot write
// "HTTP diagnostic handlers", so this carries the pair-kind label where one
// applies and is empty otherwise — an empty cell being the honest rendering of
// "this family has no name doppel can justify".
type HTMLFamily struct {
	N             int
	What          string
	Where         string
	Tag           string
	Members       int
	MinLabel      string
	EvidenceLabel string
	AddedLabel    string
}

// PrintHTML writes the report as one self-contained HTML document.
//
// Self-contained is the point: the CSS is inlined, there is no script, and the
// only external reference is the design system's own font import, which falls
// back to system-ui offline. The canvas prototype this implements fetches its
// data at runtime and needs a server; a report a developer can mail to someone
// cannot.
func PrintHTML(w io.Writer, r HTMLReport) error {
	return htmlTemplate.Execute(w, htmlView{Report: r, CSS: template.CSS(broadsheetCSS)})
}

// htmlView is what the template sees. The CSS is typed so html/template inlines
// it into the <style> element instead of escaping it as text.
type htmlView struct {
	Report HTMLReport
	CSS    template.CSS
}

// pct renders a percentage as a bare number for a width: attribute. Values are
// clamped, because a width over 100% silently breaks the strip's grid.
func pct(v float64) int {
	n := int(v*100 + 0.5)
	if n > 100 {
		return 100
	}
	if n < 0 {
		return 0
	}
	return n
}

// comma groups thousands, the way the design's numerals are set.
func comma(n int) string {
	s := fmt.Sprint(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, d := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(d)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

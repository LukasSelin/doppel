package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/parser"
)

// The names in a generated diagram come from the analysed source, so the
// escaping has to survive whatever Go allows. Every input here is a real shape
// from the committed examples.
func TestMermaidLabelEscapesRealIdentifiers(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"cobra.*Command.getOut", "cobra.*Command.getOut"},
		{"pool.*ResultPool[T].WithMaxGoroutines", "pool.*ResultPool[T].WithMaxGoroutines"},
		{"internal/reporter", "internal/reporter"},
		// Mermaid has no escape character inside a quoted label, so these have
		// to become HTML entities or the label ends early.
		{`say "hi"`, "say #quot;hi#quot;"},
		{"tag#1", "tag#35;1"},
		{"a<b>c", "a#lt;b#gt;c"},
		{"two\nlines", "two lines"},
	} {
		if got := mermaidLabel(tt.in); got != tt.want {
			t.Errorf("mermaidLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// A "#" already inside the input must not re-escape the entities produced for
// the quotes beside it.
func TestMermaidLabelDoesNotDoubleEscape(t *testing.T) {
	got := mermaidLabel(`#"`)
	if got != "#35;#quot;" {
		t.Errorf("mermaidLabel(`#\"`) = %q, want %q", got, "#35;#quot;")
	}
}

// Ids are bare tokens, not labels: a name mangled into an id would collide the
// moment `a.b` and `a_b` both exist, so ids are positional and the readable
// name lives in the label.
func TestMermaidIDIsATokenNotAName(t *testing.T) {
	id := mermaidID("h", 12)
	if id != "h12" {
		t.Errorf("mermaidID = %q, want h12", id)
	}
	if strings.ContainsAny(id, "./*[]- ") {
		t.Errorf("id %q contains characters mermaid cannot parse", id)
	}
}

func TestHeatClassBuckets(t *testing.T) {
	for _, tt := range []struct {
		v    float64
		want string
	}{{0.95, "good"}, {0.75, "good"}, {0.74, "warn"}, {0.5, "warn"}, {0.49, "hot"}, {0, "hot"}} {
		if got := heatClass(tt.v); got != tt.want {
			t.Errorf("heatClass(%.2f) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func famUnits(n int) []parser.CodeUnit {
	out := make([]parser.CodeUnit, n)
	for i := range out {
		out[i] = parser.CodeUnit{Name: string(rune('a' + i)), Package: "p"}
	}
	return out
}

func famOf(n int) family.Family {
	f := family.Family{MinEdge: 0.8}
	for i := 0; i < n; i++ {
		f.Members = append(f.Members, i)
	}
	return f
}

// The diagram exists to show the one property the prose claims — every member
// connected to every other. A star or a chain would depict the failure mode
// this whole feature avoids, so it is all-pairs or nothing.
func TestFamilyDiagramDrawsEveryEdge(t *testing.T) {
	var b strings.Builder
	mdFamilyDiagram(&b, famOf(4), famUnits(4))
	out := b.String()

	if !strings.Contains(out, "```mermaid") {
		t.Fatalf("no diagram emitted:\n%s", out)
	}
	if n := strings.Count(out, " --- "); n != 6 { // 4*3/2
		t.Errorf("drew %d edges, want 6:\n%s", n, out)
	}
}

// Edges grow as n(n-1)/2, so past a limit the picture tells a reader strictly
// less than the sentence above it. The limit is stated, never silent.
func TestFamilyDiagramRefusesToDrawAHairball(t *testing.T) {
	var b strings.Builder
	mdFamilyDiagram(&b, famOf(maxFamilyDiagram+1), famUnits(maxFamilyDiagram+1))
	out := b.String()

	if strings.Contains(out, "```mermaid") {
		t.Errorf("drew a diagram past the limit:\n%s", out)
	}
	if !strings.Contains(out, "9 members is 36 connections") {
		t.Errorf("limit not explained:\n%s", out)
	}

	var at strings.Builder
	mdFamilyDiagram(&at, famOf(maxFamilyDiagram), famUnits(maxFamilyDiagram))
	if !strings.Contains(at.String(), "```mermaid") {
		t.Errorf("refused to draw a family exactly at the limit:\n%s", at.String())
	}
}

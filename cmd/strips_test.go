package cmd

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/parser"
)

func stripUnits() []parser.CodeUnit {
	return []parser.CodeUnit{
		{Name: "dbJoin", Package: "networkdb", File: "a/diag.go", StartLine: 38},
		{Name: "dbPeers", Package: "networkdb", File: "a/diag.go", StartLine: 71},
		{Name: "dbLast", Package: "networkdb", File: "a/diag.go", StartLine: 128},
		{Name: "elsewhere", Package: "other", File: "b/other.go", StartLine: 10},
	}
}

// The bar is the gap to the next declaration in the same file, so the last
// declaration in a file has none. That is a fact about the measurement, not a
// missing value, and the strip says so rather than drawing a zero-width bar.
func TestDeclarationSpansStopAtTheLastInFile(t *testing.T) {
	res := Result{Units: stripUnits()}
	spans := declarationSpans(res)

	if spans[0] != 33 {
		t.Errorf("dbJoin span = %d, want 33", spans[0])
	}
	if spans[1] != 57 {
		t.Errorf("dbPeers span = %d, want 57", spans[1])
	}
	if _, ok := spans[2]; ok {
		t.Error("the last declaration in a file was given a span")
	}
	if _, ok := spans[3]; ok {
		t.Error("the only declaration in a file was given a span")
	}
}

// A strip is a column of bars measured between declarations, which means
// nothing across files. Those families are still in the census; they just have
// no silhouette to draw.
func TestStripsOnlyForSingleFileFamilies(t *testing.T) {
	res := Result{Units: stripUnits()}

	sameFile := family.Family{Members: []int{0, 1, 2}, MinEdge: 0.68}
	if strips := htmlStrips(res, []family.Family{sameFile}, nil); len(strips) != 1 {
		t.Fatalf("got %d strips for a single-file family, want 1", len(strips))
	}

	spread := family.Family{Members: []int{0, 3}, MinEdge: 0.68}
	if strips := htmlStrips(res, []family.Family{spread}, nil); len(strips) != 0 {
		t.Errorf("a family spanning two files produced %d strips, want 0", len(strips))
	}
}

// The silhouette is only readable in declaration order. Family members are held
// by unit index, which is file-walk order and only coincidentally the same.
func TestStripMembersAreInDeclarationOrder(t *testing.T) {
	res := Result{Units: stripUnits()}
	f := family.Family{Members: []int{2, 0, 1}, MinEdge: 0.68}

	strips := htmlStrips(res, []family.Family{f}, nil)
	if len(strips) != 1 {
		t.Fatalf("got %d strips, want 1", len(strips))
	}
	got := []string{}
	for _, m := range strips[0].Members {
		got = append(got, m.Name)
	}
	want := []string{"dbJoin", "dbPeers", "dbLast"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("member order = %v, want %v", got, want)
		}
	}
	if strips[0].Members[2].HasSpan {
		t.Error("the last member drew a bar despite having no successor")
	}
}

func TestIsHTMLPath(t *testing.T) {
	for _, tt := range []struct {
		path string
		want bool
	}{
		{"report.html", true}, {"report.htm", true}, {"REPORT.HTML", true},
		{"report.md", false}, {"report", false}, {"a.html/b.md", false},
	} {
		if got := isHTMLPath(tt.path); got != tt.want {
			t.Errorf("isHTMLPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

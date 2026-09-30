package fingerprint

import (
	"reflect"
	"testing"

	"github.com/LukasSelin/doppel/internal/gofront"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// parseFn is parseFunc keeping the whole declaration, so a lens can choose
// between the canonical shape and the body as written.
func parseFn(t *testing.T, src string) *syntax.Func {
	t.Helper()
	f, err := gofront.Parse("snippet.go", []byte("package p\n"+src))
	if err != nil {
		t.Fatalf("parse snippet: %v", err)
	}
	if f == nil || len(f.Funcs) == 0 {
		t.Fatalf("no function declaration in snippet")
	}
	return &f.Funcs[0]
}

func profileOf(t *testing.T, a, b string) LensProfile {
	t.Helper()
	fa, fb := parseFn(t, a), parseFn(t, b)
	var p LensProfile
	for i, l := range Lenses() {
		p[i] = ScoreLens(l.Bag(fa), l.Bag(fb), nil)
	}
	return p
}

// lensFixtures are one pair per class. Each is the smallest edit that should
// land a pair on that rung and nowhere stricter.
var lensFixtures = []struct {
	class, a, b string
}{
	{"verbatim",
		"func f(xs []int) int {\n\tn := 0\n\tfor _, x := range xs {\n\t\tn += x\n\t}\n\treturn n\n}",
		"func g(xs []int) int {\n\tn := 0\n\tfor _, x := range xs {\n\t\tn += x\n\t}\n\treturn n\n}"},
	{"renamed",
		"func f(xs []int) int {\n\tn := 0\n\tfor _, x := range xs {\n\t\tn += x\n\t}\n\treturn n\n}",
		"func g(vals []int) int {\n\ttotal := 0\n\tfor _, v := range vals {\n\t\ttotal += v\n\t}\n\treturn total\n}"},
	{"inverse", "func f" + gather[len("func f"):], "func f" + scatter[len("func f"):]},
	{"parameterized",
		"func f(u *User) string {\n\tif u.Name == \"\" {\n\t\treturn \"anon\"\n\t}\n\treturn u.Name\n}",
		"func f(u *User) string {\n\tif u.Email == \"\" {\n\t\treturn \"none\"\n\t}\n\treturn u.Email\n}"},
	{"template",
		"func f(c *Conn) error {\n\tif err := c.Open(); err != nil {\n\t\treturn err\n\t}\n\treturn nil\n}",
		"func f(c *Conn) error {\n\tif err := c.Flush(); err != nil {\n\t\treturn err\n\t}\n\treturn nil\n}"},
	{"",
		"func f(xs []int) int {\n\treturn len(xs)\n}",
		"func f(m map[string]int) {\n\tfor k := range m {\n\t\tdelete(m, k)\n\t}\n}"},
}

func TestLensClasses(t *testing.T) {
	for _, fx := range lensFixtures {
		p := profileOf(t, fx.a, fx.b)
		if got := p.Class(); got != fx.class {
			t.Errorf("class %q: got %q (exact skeleton=%v shape=%v ordered=%v vocab=%v verbatim=%v)",
				fx.class, got, p[LensSkeleton].Exact, p[LensShape].Exact, p[LensOrdered].Exact,
				p[LensVocab].Exact, p[LensVerbatim].Exact)
		}
	}
}

// The lattice edges Lens documents, checked over every fixture pair in both
// orders. A lens that stopped implying the one below it would make Class
// read a pattern that cannot occur and miss one that can.
func TestLensLatticeEdges(t *testing.T) {
	var srcs []string
	for _, fx := range lensFixtures {
		srcs = append(srcs, fx.a, fx.b)
	}
	implies := [][2]int{
		{LensVerbatim, LensVocab},
		{LensVocab, LensShape},
		{LensOrdered, LensShape},
		{LensShape, LensSkeleton},
	}
	for i := range srcs {
		for j := range srcs {
			p := profileOf(t, srcs[i], srcs[j])
			for _, e := range implies {
				if p[e[0]].Exact && !p[e[1]].Exact {
					t.Errorf("fixtures %d,%d: exact under %s but not %s",
						i, j, Lenses()[e[0]].Name, Lenses()[e[1]].Name)
				}
			}
		}
	}
}

// The shape lens is the production bag, exactly: it is what retrieval and the
// code-shape score read, so a lens table that drifted from it would report a
// "shape" number nothing else in the tool computed.
func TestShapeLensIsProduction(t *testing.T) {
	fn := parseFn(t, gather)
	if !reflect.DeepEqual(Lenses()[LensShape].Bag(fn), WLBag(fn)) {
		t.Fatal("the shape lens must be byte-identical to WLBag")
	}
}

func TestLensNilBody(t *testing.T) {
	for _, l := range Lenses() {
		if l.Bag(nil) != nil || l.Bag(&syntax.Func{}) != nil {
			t.Errorf("%s: a function without a body must yield a nil bag", l.Name)
		}
	}
	if ScoreLens(nil, nil, nil).Exact {
		t.Error("two empty bags are not an exact match")
	}
}

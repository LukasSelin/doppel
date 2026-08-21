package concepter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
)

// unitFromSource parses a snippet and returns all its units, tagged with real
// parser output so the resolver sees exactly what the pipeline sees.
func unitsFromSource(t *testing.T, path, src string) []parser.CodeUnit {
	t.Helper()
	units, err := parser.ParseSource(path, []byte(src))
	if err != nil {
		t.Fatalf("ParseSource(%s): %v", path, err)
	}
	return units
}

// corpus assembles a multi-file corpus in unit-slice order, the same shape
// cmd/analyze.go builds by walking the tree.
func corpus(t *testing.T, files map[string]string) []parser.CodeUnit {
	t.Helper()
	// Deterministic file order, like WalkDir's lexical order.
	var paths []string
	for p := range files {
		paths = append(paths, p)
	}
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			if paths[j] < paths[i] {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}
	var units []parser.CodeUnit
	for _, p := range paths {
		units = append(units, unitsFromSource(t, p, files[p])...)
	}
	return units
}

func callersOf(g *Graph, key string) []string { return g.Callers[key] }

func assertEdges(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("edges = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("edges = %v, want %v", got, want)
		}
	}
}

func TestQualifiedName(t *testing.T) {
	fn := parser.CodeUnit{Name: "Map", Package: "mapper"}
	if got := QualifiedName(fn); got != "mapper.Map" {
		t.Errorf("QualifiedName(function) = %q", got)
	}
	m := parser.CodeUnit{Name: "*Comparator.Compare", Package: "comparator"}
	if got := QualifiedName(m); got != "comparator.*Comparator.Compare" {
		t.Errorf("QualifiedName(method) = %q", got)
	}
	if got := KeyPackage("comparator.*Comparator.Compare"); got != "comparator" {
		t.Errorf("KeyPackage split at the wrong dot: %q", got)
	}
}

func TestResolveSamePackageBareCall(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
func caller() { helper() }
func helper() {}`,
	})
	g := BuildCallGraph(units)
	assertEdges(t, callersOf(g, "alpha.helper"), "alpha.caller")
	assertEdges(t, g.Callees["alpha.caller"], "alpha.helper")
}

// The collision this whole change exists to kill: two packages each declaring
// New must not share caller edges.
func TestResolveDoesNotConflateSameBareNameAcrossPackages(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
func New() {}
func useAlpha() { New() }`,
		"b/b.go": `package beta
func New() {}`,
	})
	g := BuildCallGraph(units)
	assertEdges(t, callersOf(g, "alpha.New"), "alpha.useAlpha")
	assertEdges(t, callersOf(g, "beta.New"))
}

func TestResolvePackageQualifiedCall(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
import "example.com/mod/beta"
func caller() { beta.Work() }`,
		"b/b.go": `package beta
func Work() {}`,
	})
	g := BuildCallGraph(units)
	assertEdges(t, callersOf(g, "beta.Work"), "alpha.caller")
}

// An aliased import binds the alias, and the alias wins over any package that
// happens to share the local name — the same way the compiler resolves it.
func TestResolveAliasedImport(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
import gamma "example.com/mod/beta"
func caller() { gamma.Work() }`,
		"b/b.go": `package beta
func Work() {}`,
		"g/g.go": `package gamma
func Work() {}`,
	})
	g := BuildCallGraph(units)
	// gamma.Work in the caller's file means the beta package, not package gamma.
	assertEdges(t, callersOf(g, "beta.Work"), "alpha.caller")
	assertEdges(t, callersOf(g, "gamma.Work"))
}

// A method call on a variable resolves only when the corpus has exactly one
// method of that name — the receiver's type is unknowable without go/types.
func TestResolveUniqueMethod(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
type Server struct{}
func (s *Server) Start() {}
func run(srv *Server) { srv.Start() }`,
	})
	g := BuildCallGraph(units)
	assertEdges(t, callersOf(g, "alpha.*Server.Start"), "alpha.run")
}

func TestResolveAmbiguousMethodDropped(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
type Server struct{}
func (s *Server) Start() {}
type Worker struct{}
func (w *Worker) Start() {}
func run(x *Server) { x.Start() }`,
	})
	g := BuildCallGraph(units)
	assertEdges(t, callersOf(g, "alpha.*Server.Start"))
	assertEdges(t, callersOf(g, "alpha.*Worker.Start"))
}

// extractCallees collapses chained selectors (c.inner.Work) to the bare method
// name; the unique-method rule recovers the edge.
func TestResolveChainedSelectorCollapse(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
type Inner struct{}
func (i *Inner) Work() {}
type Outer struct{ inner *Inner }
func (o *Outer) Run() { o.inner.Work() }`,
	})
	g := BuildCallGraph(units)
	assertEdges(t, callersOf(g, "alpha.*Inner.Work"), "alpha.*Outer.Run")
}

func TestResolveBuiltinsAndExternalsDrop(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
import "fmt"
func caller(xs []int) {
	fmt.Println(len(xs))
	xs = append(xs, 1)
}`,
	})
	g := BuildCallGraph(units)
	if got := g.Callees["alpha.caller"]; len(got) != 0 {
		t.Errorf("builtin/external calls resolved to %v, want none", got)
	}
}

func TestResolveRecursionExcluded(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
func loop(n int) {
	if n > 0 {
		loop(n - 1)
	}
}`,
	})
	g := BuildCallGraph(units)
	assertEdges(t, callersOf(g, "alpha.loop"))
	if got := g.Callees["alpha.loop"]; len(got) != 0 {
		t.Errorf("self-edge survived: %v", got)
	}
}

// Two internal packages with the same package name and the same function name:
// refuse to guess rather than pick one.
func TestResolveDuplicatePackageNameDropped(t *testing.T) {
	units := corpus(t, map[string]string{
		"x/util/u.go": `package util
func Frob() {}`,
		"y/util/u.go": `package util
func Frob() {}`,
		"a/a.go": `package alpha
import "example.com/mod/x/util"
func caller() { util.Frob() }`,
	})
	g := BuildCallGraph(units)
	// Both util.Frob keys collapse to the same qualified name; the resolver
	// treats the pair as unresolvable and records no edge for either.
	assertEdges(t, callersOf(g, "util.Frob"))
}

// Dot-imported functions arrive as bare names from a different package and are
// documented as unresolvable. This test pins the accepted behavior: a miss,
// not a false edge.
func TestResolveDotImportMissesDocumented(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
import . "example.com/mod/beta"
func caller() { Work() }`,
		"b/b.go": `package beta
func Work() {}`,
	})
	g := BuildCallGraph(units)
	assertEdges(t, callersOf(g, "beta.Work"))
}

func TestGraphOutputSorted(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
func zebra() { shared() }
func apple() { shared() }
func shared() {}`,
	})
	g := BuildCallGraph(units)
	got := callersOf(g, "alpha.shared")
	assertEdges(t, got, "alpha.apple", "alpha.zebra")
	for key, edges := range g.Callers {
		for i := 1; i < len(edges); i++ {
			if edges[i-1] >= edges[i] {
				t.Errorf("Callers[%q] not strictly sorted: %v", key, edges)
			}
		}
	}
}

// Diamond: a calls b and c, b and c call d. N2(a) must reach d through both
// arms while excluding a itself.
func TestNeighborhoodDiamond(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
func a() { b(); c() }
func b() { d() }
func c() { d() }
func d() {}`,
	})
	g := BuildCallGraph(units)
	assertEdges(t, g.Neighborhood("alpha.a"), "alpha.b", "alpha.c", "alpha.d")
	// d's ball: direct callers b,c, plus their caller a.
	assertEdges(t, g.Neighborhood("alpha.d"), "alpha.a", "alpha.b", "alpha.c")
	// An isolated function has no neighborhood at all.
	iso := corpus(t, map[string]string{"i/i.go": "package iso\nfunc alone() {}"})
	if got := BuildCallGraph(iso).Neighborhood("iso.alone"); got != nil {
		t.Errorf("isolated neighborhood = %v, want nil", got)
	}
}

func TestBuildCallGraphDeterministic(t *testing.T) {
	units := corpus(t, map[string]string{
		"a/a.go": `package alpha
import "example.com/mod/beta"
type T struct{}
func (t *T) M() { beta.Work() }
func f() { g(); h() }
func g() { h() }
func h() {}`,
		"b/b.go": `package beta
func Work() {}`,
	})
	render := func(g *Graph) string {
		var sb strings.Builder
		var keys []string
		for k := range g.Callers {
			keys = append(keys, k)
		}
		// deterministic render for comparison
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				if keys[j] < keys[i] {
					keys[i], keys[j] = keys[j], keys[i]
				}
			}
		}
		for _, k := range keys {
			sb.WriteString(k + "<-[" + strings.Join(g.Callers[k], ",") + "] ")
			sb.WriteString(k + "->[" + strings.Join(g.Callees[k], ",") + "]\n")
		}
		return sb.String()
	}
	want := render(BuildCallGraph(units))
	for i := 0; i < 100; i++ {
		if got := render(BuildCallGraph(units)); got != want {
			t.Fatalf("run %d diverged:\n%s\nwant:\n%s", i, got, want)
		}
	}
}

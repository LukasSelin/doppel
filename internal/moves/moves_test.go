package moves

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeModule writes files, keyed by slash-separated path, into a new module
// named m and returns its directory.
func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	files["go.mod"] = "module m\n\ngo 1.22\n"
	for name, src := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func loadGraph(t *testing.T, files map[string]string) *Graph {
	t.Helper()
	fset, pkgs, err := Load(context.Background(), writeModule(t, files), nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	return Build(fset, pkgs)
}

func findMoves(t *testing.T, opt Options, files map[string]string) (*Graph, map[string]Move) {
	t.Helper()
	g := loadGraph(t, files)
	byName := map[string]Move{}
	for _, m := range Find(g, opt) {
		byName[m.Func.Qualified()] = m
	}
	return g, byName
}

func sym(t *testing.T, g *Graph, qualified string) *Sym {
	t.Helper()
	for _, s := range g.Syms {
		if s.Qualified() == qualified {
			return s
		}
	}
	t.Fatalf("no symbol %s", qualified)
	return nil
}

func names(syms []*Sym) string {
	var s []string
	for _, x := range syms {
		s = append(s, x.Name())
	}
	return strings.Join(s, ",")
}

func TestFind(t *testing.T) {
	_, moves := findMoves(t, DefaultOptions(), map[string]string{
		"a/a.go": `package a

// F is used only by b, and brings its private helpers along.
func F() int { return helper() + limit + (&counter{}).next() }

func helper() int { return twice(1) }
func twice(n int) int { return 2 * n }

const limit = 3

type counter struct{ n int }

func (c *counter) next() int { c.n++; return c.n }

// G is used only by b but shares shared with H, which stays.
func G() int { return shared() }
func H() int { return shared() }
func shared() int { return 1 }

// Used is referred to from inside a, so it stays.
func Used() int { return 1 }
func caller() int { return Used() }

var _ = caller

type T struct{}

// M is a method; methods cannot leave their type's package.
func (T) M() {}
`,
		"a/a_test.go": `package a

import "testing"

func TestTwice(t *testing.T) { _ = twice(2) }
`,
		"b/b.go": `package b

import "m/a"

func Use() int { return a.F() + a.G() + a.Used() + a.H() }

func UseM() { a.T{}.M() }
`,
		"b/b_test.go": `package b

import (
	"testing"

	aa "m/a"
)

func TestG(t *testing.T) { _ = aa.G() }
`,
	})

	f, ok := moves["a.F"]
	if !ok {
		t.Fatalf("no move for a.F; got %v", moves)
	}
	if f.To.Path != "m/b" || names(f.Users) != "Use" || f.Share != 1 || f.Staying != nil {
		t.Errorf("a.F moves to %s for %s (share %v, staying %s), want m/b for Use alone",
			f.To.Path, names(f.Users), f.Share, names(f.Staying))
	}
	if got := names(f.With); got != "helper,limit,counter,twice" {
		t.Errorf("a.F moves with %s", got)
	}
	if len(f.Blocked) > 0 {
		t.Errorf("a.F blocked: %v", f.Blocked)
	}
	if len(f.TestFiles) != 1 || filepath.Base(f.TestFiles[0]) != "a_test.go" {
		t.Errorf("a.F named in %v, want a_test.go, which uses twice", f.TestFiles)
	}

	g := moves["a.G"]
	if len(g.Blocked) != 1 || !strings.Contains(g.Blocked[0], "unexported shared") {
		t.Errorf("a.G blocked: %v, want the shared helper", g.Blocked)
	}
	if g.With != nil {
		t.Errorf("a.G moves with %s, but shared stays", names(g.With))
	}
	// An aliased import of a from b's tests still names G.
	if len(g.TestFiles) != 1 || filepath.Base(g.TestFiles[0]) != "b_test.go" {
		t.Errorf("a.G named in %v, want b_test.go", g.TestFiles)
	}
	for _, name := range []string{"a.Used", "a.T.M", "b.Use", "a.shared"} {
		if _, ok := moves[name]; ok {
			t.Errorf("unexpected move for %s", name)
		}
	}
}

// Methods are never candidates, even when one package is their only caller.
func TestFindSkipsMethods(t *testing.T) {
	_, moves := findMoves(t, DefaultOptions(), map[string]string{
		"a/a.go": `package a

type T struct{}

func (T) M() int { return 1 }
func (*T) P() int { return 2 }
`,
		"b/b.go": `package b

import "m/a"

func Use() int { var t a.T; return t.M() + t.P() }
`,
	})
	if len(moves) != 0 {
		t.Errorf("moves = %v, want none", moves)
	}
}

// A function whose signature names a type of its own package that stays is
// a constructor or an entry point of that type: reported, with the anchor.
func TestFindAnchors(t *testing.T) {
	_, moves := findMoves(t, DefaultOptions(), map[string]string{
		"a/a.go": `package a

type Repo struct{ path string }

func Open(path string) (*Repo, error) { return &Repo{path: path}, nil }

func (r *Repo) Path() string { return r.path }
`,
		"b/b.go": `package b

import "m/a"

func Use() string { r, _ := a.Open("."); return r.Path() }
`,
	})
	m := moves["a.Open"]
	if names(m.Anchors) != "Repo" {
		t.Errorf("a.Open anchors = %q, want Repo", names(m.Anchors))
	}
	// Open sets an unexported field of Repo, which Repo's own method reads.
	if len(m.Blocked) != 1 || !strings.Contains(m.Blocked[0], "field Repo.path") {
		t.Errorf("a.Open blocked: %q", m.Blocked)
	}
}

// Below Share 1 the referrers that stay must import the target. Here the one
// staying referrer is in d, which b already imports: moving F into b would
// close the cycle d -> b -> d.
func TestFindRelaxedShareCycle(t *testing.T) {
	files := map[string]string{
		"a/a.go": `package a

func F() int { return 1 }
`,
		"b/b.go": `package b

import (
	"m/a"
	"m/d"
)

func One() int { return a.F() }
func Two() int { return a.F() + d.D() }
`,
		"d/d.go": `package d

import "m/a"

func D() int { return a.F() }
`,
	}
	_, strict := findMoves(t, DefaultOptions(), files)
	if _, ok := strict["a.F"]; ok {
		t.Errorf("a.F is a move at share 1, though d refers to it too")
	}
	_, relaxed := findMoves(t, Options{Share: 0.6}, files)
	m, ok := relaxed["a.F"]
	if !ok {
		t.Fatalf("no move for a.F at share 0.6")
	}
	if m.To.Path != "m/b" || names(m.Staying) != "D" {
		t.Errorf("a.F -> %s staying %s, want m/b staying D", m.To.Path, names(m.Staying))
	}
	if len(m.Blocked) != 1 || m.Blocked[0] != "import cycle: m/d -> m/b -> m/d" {
		t.Errorf("a.F blocked: %q", m.Blocked)
	}
}

// A function moving into its only user's package cannot close a cycle: that
// package already imports the function's, which imports all it needs. The
// check still guards a target that does not import the function's package
// yet, as here, where b does not use K.
func TestBlockersImportCycle(t *testing.T) {
	g := loadGraph(t, map[string]string{
		"a/a.go": `package a

import "m/d"

func K() int { return d.D() }
`,
		"b/b.go": `package b

var Name = "b"
`,
		"d/d.go": `package d

import "m/b"

var _ = b.Name

func D() int { return 1 }
`,
	})
	k := sym(t, g, "a.K")
	got := g.blockers(Move{Func: k, To: g.Pkgs["m/b"]}, closure(k))
	if len(got) != 1 || got[0] != "import cycle: m/b -> m/d -> m/b" {
		t.Errorf("blockers = %q", got)
	}
}

func TestFindInternal(t *testing.T) {
	_, moves := findMoves(t, DefaultOptions(), map[string]string{
		// P needs x/internal/y, which outside/ may not import.
		"x/p/p.go": `package p

import "m/x/internal/y"

func P() int { return y.Y }
`,
		"x/internal/y/y.go": `package y

const Y = 1
`,
		"outside/o.go": `package outside

import "m/x/p"

var O = p.P()
`,
	})
	p := moves["p.P"]
	if len(p.Blocked) != 1 || !strings.Contains(p.Blocked[0], "internal package m/x/internal/y") {
		t.Errorf("p.P blocked: %q", p.Blocked)
	}
}

// The target already declares a name that moves along.
func TestFindNameCollision(t *testing.T) {
	_, moves := findMoves(t, DefaultOptions(), map[string]string{
		"a/a.go": `package a

func F() int { return helper() }
func helper() int { return 1 }
`,
		"b/b.go": `package b

import "m/a"

func helper() int { return a.F() }
`,
	})
	f := moves["a.F"]
	if len(f.Blocked) != 1 || f.Blocked[0] != "b already declares helper" {
		t.Errorf("a.F blocked: %q", f.Blocked)
	}
}

// A file this build does not compile is invisible to the type checker, so a
// reference from it is found by name and blocks the move.
func TestFindIgnoredFile(t *testing.T) {
	_, moves := findMoves(t, DefaultOptions(), map[string]string{
		"a/a.go": `package a

func F() int { return 1 }
`,
		"b/b.go": `package b

import "m/a"

func Use() int { return a.F() }
`,
		"c/c.go": `package c
`,
		"c/c_never.go": `//go:build never

package c

import "m/a"

var X = a.F()
`,
	})
	f := moves["a.F"]
	if len(f.Blocked) != 1 || f.Blocked[0] != "named in c_never.go, which this build does not compile" {
		t.Errorf("a.F blocked: %q", f.Blocked)
	}
}

func TestDispatch(t *testing.T) {
	g := loadGraph(t, map[string]string{
		"a/a.go": `package a

type Shape interface{ Area() int }

type square struct{ n int }

func (s square) Area() int { return s.n * s.n }

type circle struct{ r int }

func (c *circle) Area() int { return 3 * c.r * c.r }

type notShape struct{}

func (notShape) Area() string { return "" }

func New() Shape { return square{2} }

func Total(s Shape) int { return s.Area() }

func Generic[S Shape](s S) int { return s.Area() }
`,
	})
	dynamic := func(from string) string {
		var to []string
		for _, u := range sym(t, g, from).Out {
			if u.Dynamic {
				to = append(to, u.To.Name())
			}
		}
		return strings.Join(to, ",")
	}
	if got := dynamic("a.Total"); got != "square.Area,*circle.Area" {
		t.Errorf("Total dispatches to %q, want square.Area,*circle.Area", got)
	}
	if got := dynamic("a.Generic"); got != "square.Area,*circle.Area" {
		t.Errorf("Generic dispatches to %q, want square.Area,*circle.Area", got)
	}
	// Dispatch never makes a package a user: Area is reached only
	// dynamically, and nothing here is a move.
	if moves := Find(g, DefaultOptions()); len(moves) != 0 {
		t.Errorf("moves = %v", moves)
	}
}

// Fields reached through an embedded struct count against the embedded type,
// though the source never names it.
func TestEmbeddedFieldUse(t *testing.T) {
	g := loadGraph(t, map[string]string{
		"a/a.go": `package a

type inner struct{ n int }

type Outer struct{ inner }

func Get(o Outer) int { return o.n }
`,
	})
	found := false
	for _, u := range sym(t, g, "a.Get").Out {
		if u.To.Name() == "inner" {
			found = true
		}
	}
	if !found {
		t.Errorf("Get has no use of inner: %v", sym(t, g, "a.Get").Out)
	}
}

func TestInternalAllowed(t *testing.T) {
	for _, c := range []struct {
		importer, dep string
		want          bool
	}{
		{"m/x/p", "m/x/internal/y", true},
		{"m/x", "m/x/internal", true},
		{"m/outside", "m/x/internal/y", false},
		{"m/xy", "m/x/internal/y", false},
		{"m/outside", "m/x/y", true},
	} {
		if got := internalAllowed(c.importer, c.dep); got != c.want {
			t.Errorf("internalAllowed(%s, %s) = %v", c.importer, c.dep, got)
		}
	}
}

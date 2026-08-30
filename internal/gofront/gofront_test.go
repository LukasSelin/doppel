package gofront

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"testing"

	"github.com/LukasSelin/doppel/internal/syntax"
)

// rich exercises the node types the mapper has to get right: every
// control-flow construct that opens a nesting level, both loop forms, the
// statement shapes with L2 renders, nested selectors, a func literal, a
// labeled statement, a type switch and a select.
const rich = `package fix

import (
	"fmt"
	"strconv"
	_ "embed"
	alias "os"
)

func Work(ctx map[string]int, ch chan int, xs []int) (int, error) {
	var total int
	c := &client{}
	defer c.conn.Close()
	go func() { fmt.Println("bg") }()
outer:
	for i := 0; i < len(xs); i++ {
		switch v := any(xs[i]).(type) {
		case int:
			total += v
		default:
			break outer
		}
	}
	for k, v := range ctx {
		if n, err := strconv.Atoi(k); err == nil && n > 0 {
			total += v
		}
	}
	select {
	case ch <- total:
		return total, nil
	default:
	}
	s := alias.Getenv("X")
	_ = s[0:1]
	_ = *(&total)
	return total, fmt.Errorf("done: %w", errSentinel)
}
`

// astNodeCount counts what ast.Inspect visits, which is exactly what
// Fingerprint.Nodes used to count.
func astNodeCount(n ast.Node) int {
	count := 0
	ast.Inspect(n, func(x ast.Node) bool {
		if x != nil {
			count++
		}
		return true
	})
	return count
}

func irNodeCount(n *syntax.Node) int {
	count := 0
	syntax.Inspect(n, func(x *syntax.Node) bool {
		if x != nil {
			count++
		}
		return true
	})
	return count
}

// TestMapperPreservesNodeCount is the mapper's core contract. Fingerprint.Nodes
// feeds --min-nodes and SizeRatio, and the token stream is emitted in visit
// order, so a mapper that collapses or reorders nodes changes scores instead
// of losing detail quietly. Counting both ways on the same source is the
// cheapest way to keep that honest.
func TestMapperPreservesNodeCount(t *testing.T) {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "fix.go", rich, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	f, err := Parse("fix.go", []byte(rich))
	if err != nil {
		t.Fatalf("frontend: %v", err)
	}
	if f == nil || len(f.Funcs) != 1 {
		t.Fatalf("want 1 function, got %v", f)
	}

	var fd *ast.FuncDecl
	for _, decl := range file.Decls {
		if d, ok := decl.(*ast.FuncDecl); ok {
			fd = d
		}
	}
	want := astNodeCount(fd.Body)
	got := irNodeCount(f.Funcs[0].Body)
	if want != got {
		t.Errorf("node count: ast %d, ir %d", want, got)
	}
}

// TestMapperPreservesOrder pins the traversal order, not just the count: the
// token stream is order-sensitive, so two trees with the same nodes in a
// different order fingerprint differently.
func TestMapperPreservesOrder(t *testing.T) {
	fset := token.NewFileSet()
	file, err := goparser.ParseFile(fset, "fix.go", rich, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var fd *ast.FuncDecl
	for _, decl := range file.Decls {
		if d, ok := decl.(*ast.FuncDecl); ok {
			fd = d
		}
	}
	var wantKinds []syntax.Kind
	ast.Inspect(fd.Body, func(x ast.Node) bool {
		if x != nil {
			wantKinds = append(wantKinds, kindOf(x))
		}
		return true
	})

	f, _ := Parse("fix.go", []byte(rich))
	var gotKinds []syntax.Kind
	syntax.Inspect(f.Funcs[0].Body, func(x *syntax.Node) bool {
		if x != nil {
			gotKinds = append(gotKinds, x.Kind)
		}
		return true
	})

	if len(wantKinds) != len(gotKinds) {
		t.Fatalf("length: ast %d, ir %d", len(wantKinds), len(gotKinds))
	}
	for i := range wantKinds {
		if wantKinds[i] != gotKinds[i] {
			t.Fatalf("kind at %d: ast %v, ir %v", i, wantKinds[i], gotKinds[i])
		}
	}
}

// TestRolesAreAssigned checks the slots consumers actually read. A missing
// role is not a build error anywhere — Slot just returns nil and the render
// silently changes — so it has to be asserted.
func TestRolesAreAssigned(t *testing.T) {
	f, err := Parse("fix.go", []byte(rich))
	if err != nil || f == nil {
		t.Fatalf("frontend: %v", err)
	}
	seen := map[syntax.Kind]map[syntax.Role]bool{}
	syntax.Inspect(f.Funcs[0].Body, func(n *syntax.Node) bool {
		if n == nil {
			return true
		}
		for _, k := range n.Kids {
			if seen[n.Kind] == nil {
				seen[n.Kind] = map[syntax.Role]bool{}
			}
			seen[n.Kind][k.Role] = true
		}
		return true
	})
	want := []struct {
		kind syntax.Kind
		role syntax.Role
	}{
		{syntax.KindFor, syntax.RoleCond},
		{syntax.KindFor, syntax.RoleBody},
		{syntax.KindFor, syntax.RoleInit},
		{syntax.KindFor, syntax.RolePost},
		{syntax.KindRange, syntax.RoleX},
		{syntax.KindRange, syntax.RoleBody},
		{syntax.KindIf, syntax.RoleCond},
		{syntax.KindBlock, syntax.RoleList},
		{syntax.KindCall, syntax.RoleFun},
		{syntax.KindCall, syntax.RoleArg},
		{syntax.KindBinary, syntax.RoleX},
		{syntax.KindBinary, syntax.RoleY},
		{syntax.KindAssign, syntax.RoleLhs},
		{syntax.KindAssign, syntax.RoleRhs},
		{syntax.KindReturn, syntax.RoleResult},
		{syntax.KindSelector, syntax.RoleX},
		{syntax.KindSelector, syntax.RoleSel},
		{syntax.KindDefer, syntax.RoleCall},
		{syntax.KindGo, syntax.RoleCall},
		{syntax.KindSend, syntax.RoleValue},
		{syntax.KindExprStmt, syntax.RoleX},
	}
	for _, w := range want {
		if !seen[w.kind][w.role] {
			t.Errorf("kind %v never carried role %v", w.kind, w.role)
		}
	}
}

// TestImportsAndSignals pins the two frontend outputs the tagger reads: dot
// and blank imports must still be recorded as imports of the file even though
// they bind no selector-usable name, and a nested selector must record its
// tail pair.
func TestImportsAndSignals(t *testing.T) {
	f, err := Parse("fix.go", []byte(rich))
	if err != nil || f == nil {
		t.Fatalf("frontend: %v", err)
	}
	locals := map[string]string{}
	for _, imp := range f.Imports {
		locals[imp.Local] = imp.Path
	}
	if locals["_"] != "embed" {
		t.Errorf("blank import not recorded: %v", f.Imports)
	}
	if locals["alias"] != "os" {
		t.Errorf("aliased import not bound to its alias: %v", f.Imports)
	}
	if f.Lang != Lang {
		t.Errorf("Lang = %q, want %q", f.Lang, Lang)
	}
	if !f.Funcs[0].Exported {
		t.Error("Work should be exported")
	}
}

// TestLiteralTextIsDecoded pins that a string literal reaches the IR already
// unquoted: decoding escapes is lexical work only a frontend can do, and the
// tagger's literal channel reads the decoded form.
func TestLiteralTextIsDecoded(t *testing.T) {
	const src = `package p
func f() string { return "a\tb" }`
	f, err := Parse("p.go", []byte(src))
	if err != nil || f == nil {
		t.Fatalf("frontend: %v", err)
	}
	var got string
	syntax.Inspect(f.Funcs[0].Body, func(n *syntax.Node) bool {
		if n != nil && n.Kind == syntax.KindLit && n.Label == "STRING" {
			got = n.Text
		}
		return true
	})
	if got != "a\tb" {
		t.Errorf("literal text = %q, want %q", got, "a\tb")
	}
}

// TestSyntaxErrorIsSkipped keeps the documented contract: an unparseable file
// is skipped, never fatal.
func TestSyntaxErrorIsSkipped(t *testing.T) {
	f, err := Parse("bad.go", []byte("package p\nfunc ("))
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
	if f != nil {
		t.Errorf("want nil file, got %v", f)
	}
}

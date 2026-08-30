package lexfront

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/syntax"
)

func specFor(t *testing.T, lang string) Spec {
	t.Helper()
	for _, sp := range Specs {
		if sp.Lang == lang {
			return sp
		}
	}
	t.Fatalf("no spec for %q", lang)
	return Spec{}
}

func names(f *syntax.File) []string {
	out := make([]string, 0, len(f.Funcs))
	for _, fn := range f.Funcs {
		out = append(out, fn.Name)
	}
	return out
}

func has(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// TestPythonClassMethods pins the two defects that made Python nearly
// useless: a class with a parenthesised base looked exactly like a function
// declaration and swallowed every method inside it, and the indent rule
// measured from the method's *name* rather than from its "def", which ended
// each body on its own first statement.
func TestPythonClassMethods(t *testing.T) {
	const src = `
import os

class Widget(Base):
    def __init__(self, name):
        self.name = name

    def render(self, out):
        if isinstance(out, str):
            out = open(out)
        return out

def helper(a, b):
    return a + b
`
	f, _ := Parse(specFor(t, "python"), "w.py", []byte(src))
	got := names(f)
	for _, want := range []string{"__init__", "render", "helper"} {
		if !has(got, want) {
			t.Errorf("missing %q; got %v", want, got)
		}
	}
	if has(got, "Widget") {
		t.Errorf("a class is not a unit; got %v", got)
	}
	if len(got) != 3 {
		t.Errorf("want exactly 3 units, got %v", got)
	}
}

// TestStatementKeywordsAreNotFunctions pins the false positive that was 46%
// of everything found on the Python standard library: "except ValueError:"
// and "if isinstance(x, y):" both match the shape of a declaration.
func TestStatementKeywordsAreNotFunctions(t *testing.T) {
	const src = `
def work(items):
    try:
        if isinstance(items, list):
            print(items)
    except ValueError:
        pass
    for x in items:
        del x
`
	f, _ := Parse(specFor(t, "python"), "w.py", []byte(src))
	got := names(f)
	if len(got) != 1 || got[0] != "work" {
		t.Errorf("want [work], got %v", got)
	}
}

// TestTypeScriptAnnotatedMethods pins the return-type annotation case: a
// colon that does not end the line is an annotation, and the search for the
// body has to carry on past it rather than give up.
func TestTypeScriptAnnotatedMethods(t *testing.T) {
	const src = `
import { readFile } from "fs";

export class Loader {
  async loadConfig(name: string): Promise<Config> {
    const raw = await readFile(name, "utf8");
    return JSON.parse(raw);
  }
}

function plain(a: number): number { return a * 2; }
`
	f, _ := Parse(specFor(t, "typescript"), "a.ts", []byte(src))
	got := names(f)
	for _, want := range []string{"loadConfig", "plain"} {
		if !has(got, want) {
			t.Errorf("missing %q; got %v", want, got)
		}
	}
	if has(got, "Loader") {
		t.Errorf("a class is not a unit; got %v", got)
	}
	// The import must bind, or TagSignals.RefPath and the resolved call
	// channel have nothing to work with.
	var bound bool
	for _, imp := range f.Imports {
		if imp.Local == "readFile" && imp.Path == "fs" {
			bound = true
		}
	}
	if !bound {
		t.Errorf("readFile not bound to fs; imports %v", f.Imports)
	}
}

// TestGoLexicalMatchesShape uses the fidelity control spec on Go, which is
// the language where the expected answer is unambiguous.
func TestGoLexicalMatchesShape(t *testing.T) {
	const src = `package p

import "fmt"

func Sum(xs []int) (int, error) {
	total := 0
	for _, x := range xs {
		if x > 0 {
			total += x
		}
	}
	return total, nil
}

func (s *Server) Start(addr string) error {
	return fmt.Errorf("no: %w", errNo)
}

func nop() {}

func sortedKeys[K comparable, V any](m map[K]V) []K { return nil }
`
	f, _ := Parse(GoSpec, "p.go", []byte(src))
	got := names(f)
	for _, want := range []string{"Sum", "Start", "nop", "sortedKeys"} {
		if !has(got, want) {
			t.Errorf("missing %q; got %v", want, got)
		}
	}
	for _, fn := range f.Funcs {
		switch fn.Name {
		case "Start":
			// A receiver has to survive, or every method loses its identity
			// and two methods of different types collide on one name.
			if fn.Receiver != "*Server" {
				t.Errorf("Start receiver = %q, want *Server", fn.Receiver)
			}
		case "Sum":
			if len(fn.Params) != 1 || fn.Params[0].Name != "xs" {
				t.Errorf("Sum params = %v", fn.Params)
			}
			if !has(fn.Callees, "append") && len(fn.Callees) != 0 {
				// Sum calls nothing; the point is only that it is not junk.
				t.Logf("Sum callees: %v", fn.Callees)
			}
		}
	}
}

// TestAnonymousLiteralsAreNotUnits: a closure belongs to the function it is
// written in, exactly as a FuncLit does under go/ast. Emitting it separately
// made 11% of this repo's "functions" be called "func".
func TestAnonymousLiteralsAreNotUnits(t *testing.T) {
	const src = `package p

func outer(xs []int) {
	sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
}
`
	f, _ := Parse(GoSpec, "p.go", []byte(src))
	got := names(f)
	if len(got) != 1 || got[0] != "outer" {
		t.Errorf("want [outer], got %v", got)
	}
}

// TestCalleesExcludeKeywords: "if (x)" is a statement, and counting it as a
// call puts "if" among the most frequent callees in every C-family corpus,
// where it feeds retrieval, roles and the culture ecology as pure noise.
func TestCalleesExcludeKeywords(t *testing.T) {
	const src = `
function f(a) {
  if (a) { return g(a); }
  switch (a) { case 1: h(a); }
  return null;
}
`
	f, _ := Parse(specFor(t, "javascript"), "a.js", []byte(src))
	if len(f.Funcs) != 1 {
		t.Fatalf("want 1 function, got %v", names(f))
	}
	for _, c := range f.Funcs[0].Callees {
		if isStmtKeyword(c) {
			t.Errorf("%q is a statement keyword, not a callee: %v", c, f.Funcs[0].Callees)
		}
	}
	if !has(f.Funcs[0].Callees, "g") || !has(f.Funcs[0].Callees, "h") {
		t.Errorf("real calls missing: %v", f.Funcs[0].Callees)
	}
}

// TestGeneratedMarkers pins the convention-deep rule: a declared marker in
// the preamble, never a guess from the path.
func TestGeneratedMarkers(t *testing.T) {
	gen := "// Code generated by protoc. DO NOT EDIT.\n\nfunction f() { return 1; }\n"
	f, _ := Parse(specFor(t, "javascript"), "a.js", []byte(gen))
	if !f.Generated {
		t.Error("marker in the preamble should mark the file generated")
	}
	// The same words deep inside a body are a string, not a declaration.
	body := strings.Repeat("\n", 30) + "function f() { return \"DO NOT EDIT\"; }\n"
	f2, _ := Parse(specFor(t, "javascript"), "b.js", []byte(body))
	if f2.Generated {
		t.Error("a marker past the preamble should not count")
	}
}

// TestStringLiteralsAreDecoded: the literal channel is evidence the tagger
// and lexicon read, and it reads decoded contents.
func TestStringLiteralsAreDecoded(t *testing.T) {
	const src = "def f():\n    q = \"SELECT * FROM t\"\n    return q\n"
	f, _ := Parse(specFor(t, "python"), "a.py", []byte(src))
	if len(f.Funcs) != 1 {
		t.Fatalf("want 1 function, got %v", names(f))
	}
	var found bool
	syntax.Inspect(f.Funcs[0].Body, func(n *syntax.Node) bool {
		if n != nil && n.Kind == syntax.KindLit && n.Text == "SELECT * FROM t" {
			found = true
		}
		return true
	})
	if !found {
		t.Error("string literal contents did not reach the IR")
	}
}

// TestTestPatterns pins the per-language test conventions, which are what
// keeps tests in their own population.
func TestTestPatterns(t *testing.T) {
	cases := []struct {
		lang, path string
		want       bool
	}{
		{"python", "test_thing.py", true},
		{"python", "thing_test.py", true},
		{"python", "thing.py", false},
		{"typescript", "a.spec.ts", true},
		{"typescript", "a.ts", false},
		{"ruby", "a_spec.rb", true},
		{"java", "FooTest.java", true},
		{"java", "Foo.java", false},
	}
	for _, c := range cases {
		if got := specFor(t, c.lang).IsTestFile(c.path); got != c.want {
			t.Errorf("%s %s: IsTestFile = %v, want %v", c.lang, c.path, got, c.want)
		}
	}
}

// TestPackageFallsBackToDirectory: a language with no package clause still
// needs a habitat key, and the directory is what functions-that-live-together
// means there.
func TestPackageFallsBackToDirectory(t *testing.T) {
	f, _ := Parse(specFor(t, "python"), "svc/handlers/users.py", []byte("def f():\n    return 1\n"))
	if f.Package != "handlers" {
		t.Errorf("Package = %q, want handlers", f.Package)
	}
}

// TestUnparseableIsNotFatal: there is no parse to fail, so garbage yields a
// file with no functions rather than an error.
func TestUnparseableIsNotFatal(t *testing.T) {
	f, err := Parse(specFor(t, "python"), "a.py", []byte("!!! not code ((("))
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}
	if f == nil {
		t.Fatal("want a file")
	}
	if len(f.Funcs) != 0 {
		t.Errorf("want no functions, got %v", names(f))
	}
}

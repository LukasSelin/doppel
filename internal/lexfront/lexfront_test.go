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

// TestRustPatternsAreNotFunctions pins the defect that filled doppel's hook
// notes in Rust repositories with pairs of functions called Some. A pattern
// binding has the shape of a declaration — a name, a parenthesised group, a
// body — so "if let Some(x) = f() {" read as a function Some with parameter
// x, and so did let-else, a match arm after a block arm, a binding
// "n @ (0 | 1) => {", "let (a, b) = match … {" (a function called let), and
// the closure type in "|p: &dyn Fn(&str) -> bool| {". On a 1 031-function
// tree 49 units were these, paired with each other at code-shape 1.00.
//
// Patterns inside an ordinary fn do not reproduce it on their own: the fn is
// found and dropNested discards everything inside it. In the field the fn was
// lost first, because "Game<'_>" lexes the lifetime as a string running to
// end of line, and its patterns surfaced as top-level units. So the fixture
// carries that signature, and the test also reads segment directly, before
// nesting can hide anything.
func TestRustPatternsAreNotFunctions(t *testing.T) {
	const src = `
pub fn tick(game: &mut Game<'_>, at: [i32; 3]) {
    if let Some(sensor) = sensors::sensor(&game.id) {
        sense(game, at, sensor);
    }
    while let Some(next) = queue.pop() {
        run(next);
    }
    let Some(told) = told(game) else {
        return;
    };
    let Ok(count) = u8::try_from(len(at)) else {
        return;
    };
    match reading(game) {
        Ok(value) => {
            show(value);
        }
        Err(error) => {
            warn(error);
        }
    }
    match level(at) {
        _ => {}
        quiet @ (0 | 1) => {
            hush(quiet);
        }
    }
    let (a, b) = match pair(at) {
        x => x,
    };
    let part = |block, property: &dyn Fn(&str) -> Option<bool>| {
        part(block, property)
    };
}

pub fn helper(x: i32) -> i32 {
    x + 1
}
`
	sp := specFor(t, "rust")
	bogus := map[string]bool{"Some": true, "Ok": true, "Err": true, "None": true,
		"Fn": true, "let": true, "@": true}

	toks := lex([]byte(src), sp)
	for _, s := range segment(toks, sp) {
		if bogus[s.name] || toks[s.nameTok-1].text != "fn" {
			t.Errorf("segment found %q at line %d without fn before it",
				s.name, toks[s.nameTok].line)
		}
	}

	f, _ := Parse(sp, "blocks.rs", []byte(src))
	got := names(f)
	for _, n := range got {
		if bogus[n] {
			t.Errorf("pattern binding became a unit %q; got %v", n, got)
		}
	}
	if !has(got, "helper") {
		t.Errorf("missing helper; got %v", got)
	}
}

// TestRustLifetimesAreNames pins the defect that lost about a tenth of a Rust
// tree's functions: "'" was a string quote, so "Game<'_>" opened a string that
// ran to end of line and took the header's "(" with it. A lifetime or label is
// one identifier token; a char literal — escaped, unicode, or a quote — is
// still a literal.
func TestRustLifetimesAreNames(t *testing.T) {
	const src = `
pub fn tick(game: &mut Game<'_>, name: &'static str) -> &'a str {
    let c = 'x';
    let q = '\'';
    let s = '"';
    let u = '\u{1F600}';
    'outer: loop {
        break 'outer;
    }
    helper(c, q, s, u)
}

impl<'a> View<'a> {
    pub fn get(&'a self, key: &'a str) -> Option<&'a str> {
        self.map.get(key)
    }
}
`
	sp := specFor(t, "rust")
	var lifetimes, chars []string
	for _, tok := range lex([]byte(src), sp) {
		switch {
		case tok.kind == tokIdent && tok.text[0] == '\'':
			lifetimes = append(lifetimes, tok.text)
		case tok.kind == tokString:
			chars = append(chars, tok.val)
		}
	}
	if want := "'_ 'static 'a 'outer 'outer 'a 'a 'a 'a 'a"; strings.Join(lifetimes, " ") != want {
		t.Errorf("lifetimes = %v, want %s", lifetimes, want)
	}
	if want := []string{"x", "'", `"`, "u{1F600}"}; strings.Join(chars, "|") != strings.Join(want, "|") {
		t.Errorf("char literals = %q, want %q", chars, want)
	}

	f, _ := Parse(sp, "view.rs", []byte(src))
	params := map[string][]string{}
	for _, fn := range f.Funcs {
		for _, p := range fn.Params {
			params[fn.Name] = append(params[fn.Name], p.Name)
		}
	}
	if got := strings.Join(params["tick"], ","); got != "game,name" {
		t.Errorf("tick params = %q, want game,name; units %v", got, names(f))
	}
	// "&'a self" names self: a lifetime is never a parameter's name.
	if got := strings.Join(params["get"], ","); got != "self,key" {
		t.Errorf("get params = %q, want self,key; units %v", got, names(f))
	}
}

// TestRustGenericFunctions pins the other half of the Rust recall loss: only
// "[…]" was allowed between a name and its "(", so "fn f<T: Clone>(" was
// never a declaration. The list may nest, hold brackets, arrows, lifetimes
// and const parameters, and span lines.
func TestRustGenericFunctions(t *testing.T) {
	const src = `
fn plain<T: Clone>(x: T) -> T {
    x.clone()
}

pub fn nested<T: Into<Vec<u8>>>(x: T) -> usize {
    x.into().len()
}

pub fn bounded<'a, F: Fn(&'a str) -> [u8; 4], const N: usize>(f: F) -> [u8; N] {
    pad(f)
}

fn multi<
    A,
    B,
>(a: A, b: B) {
    both(a, b);
}
`
	f, _ := Parse(specFor(t, "rust"), "generic.rs", []byte(src))
	got := names(f)
	for _, want := range []string{"plain", "nested", "bounded", "multi"} {
		if !has(got, want) {
			t.Errorf("missing %q; got %v", want, got)
		}
	}
	if len(got) != 4 {
		t.Errorf("want exactly 4 units, got %v", got)
	}
}

// TestAngleEnd pins where a type-parameter list may end, and the comparison
// it must not swallow.
func TestAngleEnd(t *testing.T) {
	sp := specFor(t, "rust")
	cases := []struct {
		src  string
		want string // the token right after the list, or "" when there is none
	}{
		{"f<T>(x)", "("},
		{"f<T: Into<Vec<u8>>>(x)", "("},
		{"f<F: Fn(u8) -> [u8; 4]>(x)", "("},
		{"f<A, B,>(x)", "("},
		{"a < b; c > (d)", ""},
		{"a < b) && c > (d)", ""},
		{"f<const N: usize = { 3 }>(x)", ""},
	}
	for _, c := range cases {
		toks := lex([]byte(c.src), sp)
		end := angleEnd(toks, 1)
		got := ""
		if end >= 0 {
			got = toks[end+1].text
		}
		if got != c.want {
			t.Errorf("%s: after the list %q, want %q", c.src, got, c.want)
		}
	}
}

// TestRustMultilineAndRawStrings pins the third defect: a string ended at its
// first newline, so a multi-line literal's remaining lines lexed as code. In
// horizon those literals are GLSL and WGSL shader sources, which declare
// functions of their own, and each became a unit (main, vs, hz_depthTexture).
// Raw strings fence with "#" so they can hold a bare quote, and a byte string
// escapes its quotes like any other.
func TestRustMultilineAndRawStrings(t *testing.T) {
	const src = `
const SHADER: &str = r#"
@vertex
fn vs(@builtin(vertex_index) i: u32) -> vec4<f32> {
    return vec4<f32>(0.0, "quoted");
}
"#;

const GLSL: &str = "
void main() {
    gl_FragColor = vec4(1.0);
}
";

const BYTES: &[u8] = b"say \"hi\" { fn fake() {} }";
const RAW: &[u8] = br##"a "# b"##;

pub fn compile(src: &str) -> Module {
    let banner = "multi
line { not a body";
    build(src, banner)
}
`
	f, _ := Parse(specFor(t, "rust"), "shaders.rs", []byte(src))
	got := names(f)
	if len(got) != 1 || got[0] != "compile" {
		t.Fatalf("want [compile], got %v", got)
	}
	if line := f.Funcs[0].StartLine; line != 18 {
		t.Errorf("compile starts on line %d, want 18: line counting lost a multi-line literal", line)
	}
	var lits []string
	syntax.Inspect(f.Funcs[0].Body, func(n *syntax.Node) bool {
		if n != nil && n.Kind == syntax.KindLit && n.Label == "STRING" {
			lits = append(lits, n.Text)
		}
		return true
	})
	if len(lits) != 1 || lits[0] != "multi\nline { not a body" {
		t.Errorf("literals in compile = %q, want the two-line banner", lits)
	}

	var vals []string
	for _, tok := range lex([]byte(src), specFor(t, "rust")) {
		if tok.kind == tokString {
			vals = append(vals, tok.val)
		}
	}
	if len(vals) != 5 || vals[2] != `say "hi" { fn fake() {} }` || vals[3] != `a "# b` {
		t.Errorf("string literals = %q", vals)
	}
}

// TestTypeScriptGenericFunctions: TypeScript writes its type parameters where
// Rust does, and lost the same declarations.
func TestTypeScriptGenericFunctions(t *testing.T) {
	const src = `
function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

export class Api {
  private async request<T>(
    endpoint: string,
  ): Promise<T> {
    return fetchJSON<T>(endpoint);
  }
}
`
	f, _ := Parse(specFor(t, "typescript"), "api.ts", []byte(src))
	got := names(f)
	for _, want := range []string{"clone", "request"} {
		if !has(got, want) {
			t.Errorf("missing %q; got %v", want, got)
		}
	}
}

// TestTypeScriptKeywordlessMethods pins the reason Spec.KeywordRequired is
// per-language rather than implied by FuncKeywords: TypeScript declares
// "function", and a class method needs no keyword at all.
func TestTypeScriptKeywordlessMethods(t *testing.T) {
	const src = `
export class DateHelper {
  static isInPast(date: DateTime) {
    return date < now();
  }
  toISO(): string {
    return format(this.date);
  }
}
`
	f, _ := Parse(specFor(t, "typescript"), "DateHelper.ts", []byte(src))
	got := names(f)
	for _, want := range []string{"isInPast", "toISO"} {
		if !has(got, want) {
			t.Errorf("missing %q; got %v", want, got)
		}
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

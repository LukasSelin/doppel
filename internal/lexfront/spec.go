// Package lexfront is doppel's language-agnostic frontend: it reads code with
// no grammar for the language it is reading.
//
// It exists because a per-language grammar does not scale. internal/gofront
// is ~400 lines of go/ast knowledge, and the same again would be needed for
// every language — with a parser dependency each. What doppel actually needs
// from a frontend is much less than a parser gives: a function's boundaries,
// a shallow shape for its body, the names it calls, its literals, and its
// imports. All of that is recoverable lexically.
//
// The trade is stated rather than hidden. A lexical frontend gets the token
// stream (L0), call and operator shapes (L1), statement renders (L2), loop
// summaries (L3) and simple def-use edges (L4) — but not real types, so
// Fingerprint.Types degrades to the header's own text, and not a guarantee
// that every function was found. internal/bench measures both against the Go
// frontend on Go corpora, where the right answer is known.
//
// What stays per-language is exactly what cannot be guessed: which extensions
// to claim, how comments and strings are delimited, which keywords introduce
// a function, and how tests are named. That is a table, not a parser.
package lexfront

import (
	"path/filepath"
	"sort"
	"strings"
)

// blockStyle is how a language delimits a function body. Both are supported
// in one pass and a Spec need not declare which it uses: the segmenter looks
// at what actually follows the header — "{" or ":" — because several
// languages allow both and a declaration would just be a second place to be
// wrong.

// Spec is everything language-specific in this package.
type Spec struct {
	Lang string
	Exts []string

	// LineComment starts a comment that runs to end of line.
	LineComment []string
	// BlockOpen/BlockClose delimit a multi-line comment; empty means none.
	BlockOpen, BlockClose string
	// StringQuotes are the characters that open and close a string literal.
	// Triple-quoting is handled generically for any repeated quote.
	StringQuotes string
	// RawPrefixes are string prefixes that suppress escape processing
	// (Python's r"", C#'s @"").
	RawPrefixes string
	// StringPrefixes mark a literal's type without suppressing escapes:
	// Rust's b"" and c"". They may combine with a raw prefix — br"" is raw
	// because it carries r — and a prefix run with no raw letter in it is an
	// ordinary escaped literal. Reading b"" as raw ends "b\"key\"" at its
	// first escaped quote, and the rest of the line, or with MultilineQuotes
	// the rest of the file, lexes inside out.
	StringPrefixes string
	// RawFence is a character a raw literal may be fenced with on both sides,
	// so that it can hold a bare quote: Rust's r#"…"# and r##"…"##. The run
	// stands between the prefix and the quote, and the literal ends only at a
	// quote followed by the same run. Zero means the language has no fence.
	RawFence byte
	// MultilineQuotes are the quotes whose literal may run across lines. Every
	// other quoted literal ends at a newline, so that one unterminated quote
	// cannot swallow the rest of the file; a backtick always spans lines. In
	// Rust every "…" literal may span lines, and ending one at the newline
	// lexed its remaining lines as code — horizon's GLSL and WGSL shader
	// sources became units called main and hz_depthTexture.
	MultilineQuotes string
	// Lifetimes says "'" opens a literal only when it closes one character
	// later — a char literal 'x', '\n', '\u{1F600}' — and otherwise starts a
	// name: a Rust lifetime ('a, '_, 'static) or loop label ('outer). The
	// name lexes as one identifier token, quote included, so it collapses to
	// an identifier like any other and is never taken for a parameter name.
	// Without it "Game<'_>" opened a string running to end of line, the fn
	// whose header it sat in was lost, and a tenth of a Rust tree's functions
	// with it. Only a language whose "'" never opens a longer literal can say
	// this: C's 'ab' is a multi-character constant, and in Python or
	// JavaScript "'" quotes strings like any other quote.
	Lifetimes bool

	// FuncKeywords introduce a function declaration. A language whose
	// functions are declared by type rather than keyword (Java, C++) simply
	// lists none and relies on the bare `name(...)` + body shape.
	FuncKeywords []string
	// KeywordRequired says every function this language declares has its
	// keyword immediately before the name — Rust's "fn name(" — so the
	// segmenter accepts nothing else. In a multi-word entry the keyword is
	// the last word: "pub fn" is a modifier then the keyword "fn".
	//
	// The other shapes the segmenter accepts (a type or modifier before the
	// name, a closing bracket, a line start) exist for languages that declare
	// functions by type (Java, C) or allow methods without the keyword
	// (TypeScript's "static isInPast(", Swift's "init("). In a language with
	// neither, those shapes only ever match something that is not a function.
	// In Rust that was every pattern binding: "if let Some(x) = f() {",
	// "let Ok(n) = parse(s) else {", a match arm "Err(e) => {" after a block
	// arm, a closure type "dyn Fn(&str) -> bool| {". Each became a unit called
	// Some, Ok, Err or Fn, and they paired with each other at code-shape 1.00.
	// A language with keyword-less declarations must leave this off, or it
	// loses them.
	KeywordRequired bool
	// AngleTypeParams says type parameters are written "<…>" between a
	// function's name and its parameter list: Rust's "fn f<T: Clone>(",
	// TypeScript's "function f<T>(". "[…]" is accepted there for every
	// language (Go, Scala); "<" is an operator token, so reading it as a
	// bracket is a claim about the language, and a wrong one would make
	// "a < b && c > (d)" a declaration with a type-parameter list.
	AngleTypeParams bool
	// ImportKeywords begin a line that binds a name to a module.
	ImportKeywords []string

	// ParamNameFirst says a parameter is written name-then-type ("xs []int",
	// "x: string") rather than type-then-name ("int x", "String name").
	// Without it a Go parameter's name is read as its type, and the def-use
	// pass then attaches the "param" role to a type name that no statement
	// in the body ever mentions — losing every param→call edge.
	ParamNameFirst bool

	// ContainerKeywords introduce a type or module declaration — something
	// that holds functions but is not one. They matter because a container
	// with a parenthesised head ("class Foo(Base):") otherwise looks exactly
	// like a function declaration and swallows every method inside it.
	ContainerKeywords []string

	// TestPatterns are matched against the base filename. A "*" matches any
	// run of characters; anything else is literal.
	TestPatterns []string
}

// IsTestFile reports whether a base filename matches any test pattern.
func (s Spec) IsTestFile(path string) bool {
	base := filepath.Base(path)
	for _, pat := range s.TestPatterns {
		if matchGlob(pat, base) {
			return true
		}
	}
	return false
}

// matchGlob is a minimal two-star glob: enough for "*_test.py" and
// "*.spec.ts" and deliberately not a general matcher, which would invite
// patterns nobody can reason about.
func matchGlob(pattern, s string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	rest := s[len(parts[0]):]
	for i := 1; i < len(parts)-1; i++ {
		idx := strings.Index(rest, parts[i])
		if idx < 0 {
			return false
		}
		rest = rest[idx+len(parts[i]):]
	}
	return strings.HasSuffix(rest, parts[len(parts)-1])
}

// cFamily is the shared shape of every brace-and-slash-comment language.
func cFamily(lang string, exts []string, keywords []string, tests []string) Spec {
	return Spec{
		Lang:              lang,
		Exts:              exts,
		LineComment:       []string{"//"},
		BlockOpen:         "/*",
		BlockClose:        "*/",
		StringQuotes:      "\"'`",
		FuncKeywords:      keywords,
		ContainerKeywords: []string{"class", "struct", "interface", "enum", "impl", "trait", "object", "namespace", "record", "module", "union", "protocol", "extension"},
		ImportKeywords:    []string{"import", "using", "require", "use", "include"},
		TestPatterns:      tests,
	}
}

// Specs is the shipped language table.
//
// Only the Go frontend is measured against golden labels, and Go is not in
// this table — it has a real parser. These entries are the machinery pointed
// at the languages whose conventions are well enough known to write down;
// what they are NOT is validated, and internal/bench's fidelity harness is
// how any of them would become so. Adding one is a table entry, which is the
// point of the design.
var Specs = []Spec{
	{
		Lang:              "python",
		Exts:              []string{".py", ".pyi"},
		LineComment:       []string{"#"},
		StringQuotes:      "\"'",
		RawPrefixes:       "rRbBfFuU",
		ParamNameFirst:    true,
		FuncKeywords:      []string{"def", "async def", "lambda"},
		ContainerKeywords: []string{"class"},
		ImportKeywords:    []string{"import", "from"},
		TestPatterns:      []string{"test_*.py", "*_test.py"},
	},
	angleTypeParams(nameFirst(cFamily("typescript", []string{".ts", ".tsx", ".mts", ".cts"},
		[]string{"function", "async function"},
		[]string{"*.test.ts", "*.spec.ts", "*.test.tsx", "*.spec.tsx"}))),
	nameFirst(cFamily("javascript", []string{".js", ".jsx", ".mjs", ".cjs"},
		[]string{"function", "async function"},
		[]string{"*.test.js", "*.spec.js", "*.test.jsx", "*.spec.jsx"})),
	rust(nameFirst(cFamily("rust", []string{".rs"},
		[]string{"fn", "pub fn", "async fn"},
		[]string{"*_test.rs", "tests.rs"}))),
	cFamily("java", []string{".java"}, nil,
		[]string{"*Test.java", "Test*.java", "*Tests.java"}),
	cFamily("csharp", []string{".cs"}, nil,
		[]string{"*Test.cs", "*Tests.cs"}),
	nameFirst(cFamily("kotlin", []string{".kt", ".kts"},
		[]string{"fun", "suspend fun"},
		[]string{"*Test.kt", "*Tests.kt"})),
	nameFirst(cFamily("swift", []string{".swift"},
		[]string{"func"},
		[]string{"*Tests.swift", "*Test.swift"})),
	cFamily("php", []string{".php"},
		[]string{"function"},
		[]string{"*Test.php"}),
	{
		Lang:              "ruby",
		Exts:              []string{".rb"},
		LineComment:       []string{"#"},
		StringQuotes:      "\"'",
		ParamNameFirst:    true,
		FuncKeywords:      []string{"def"},
		ContainerKeywords: []string{"class", "module"},
		ImportKeywords:    []string{"require", "require_relative", "load"},
		TestPatterns:      []string{"*_test.rb", "*_spec.rb", "test_*.rb"},
	},
	cFamily("c", []string{".c", ".h"}, nil, []string{"*_test.c", "test_*.c"}),
	cFamily("cpp", []string{".cc", ".cpp", ".cxx", ".hpp", ".hh"}, nil,
		[]string{"*_test.cc", "*_test.cpp", "test_*.cc"}),
	nameFirst(cFamily("scala", []string{".scala"},
		[]string{"def"},
		[]string{"*Test.scala", "*Spec.scala"})),
}

func sortStrings(s []string) {
	sort.Strings(s)
}

// nameFirst marks the languages that write a parameter name before its type.
func nameFirst(s Spec) Spec { s.ParamNameFirst = true; return s }

// angleTypeParams marks the languages that write "f<T>(". Swift and C# do
// too, and are left off until a tree of either has been measured; Kotlin
// writes "fun <T> f(", before the name, which this does not reach.
func angleTypeParams(s Spec) Spec { s.AngleTypeParams = true; return s }

// rust is the C-family shape plus the facts Rust does not share with it: every
// declaration carries its keyword, "'" before a name is a lifetime, strings
// span lines and take byte, C-string and raw prefixes with a "#" fence, and
// type parameters sit in "<…>" after the name. Each is a Spec field of its
// own because each is a separate claim; this is only where Rust makes them.
func rust(s Spec) Spec {
	s = angleTypeParams(s)
	s.KeywordRequired = true
	s.Lifetimes = true
	s.MultilineQuotes = `"`
	s.RawPrefixes = "r"
	s.StringPrefixes = "bc"
	s.RawFence = '#'
	return s
}

// GoSpec is the fidelity control, and is deliberately NOT in Specs.
//
// Go has a real frontend, so nothing should ever route .go files here. It
// exists so internal/bench can run the lexical path over Go corpora and
// score it against go/ast — measuring what the AST actually buys, on the one
// language where doppel already knows the right answer. A claim about
// lexical fidelity that cannot be measured is not a claim.
var GoSpec = Spec{
	Lang:              "go-lexical",
	Exts:              nil,
	LineComment:       []string{"//"},
	BlockOpen:         "/*",
	BlockClose:        "*/",
	StringQuotes:      "\"'`",
	ParamNameFirst:    true,
	FuncKeywords:      []string{"func"},
	ContainerKeywords: []string{"type"},
	ImportKeywords:    []string{"import"},
	TestPatterns:      []string{"*_test.go"},
}

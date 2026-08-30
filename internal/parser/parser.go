package parser

import (
	"os"
	"strings"

	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// CodeUnit represents a single extracted function or method.
// Concept is one learned concept membership: which concept, and how strongly
// this unit belongs to it.
//
// Membership used to be a bare tag string, asserted by a hand-written rule the
// moment any one of its channels matched. It is a confidence now because that
// is what the evidence actually is — internal/lexicon derives both the concept
// and the number from the corpus, so a unit carrying a concept's whole learned
// vocabulary and one carrying the least of it that still counts are no longer
// indistinguishable.
//
// Confidence is in (0,1] and saturating: 0.5 is a unit carrying the concept's
// typical evidence for this corpus, not a probability.
type Concept struct {
	ID         string
	Confidence float64
}

// ConceptIDs projects memberships back to bare IDs, ascending. It is the
// boolean view of a graded fact, and every caller of it is a place that must
// not see corpus-relative weights — the merge-signal gate above all.
func ConceptIDs(cs []Concept) []string {
	if len(cs) == 0 {
		return nil
	}
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.ID
	}
	return out
}

// ConfidenceOf returns the unit's confidence in one concept, or 0.
func ConfidenceOf(cs []Concept, id string) float64 {
	for _, c := range cs {
		if c.ID == id {
			return c.Confidence
		}
	}
	return 0
}

type CodeUnit struct {
	Name         string
	File         string
	StartLine    int
	Body         string
	Signature    string                  // parameter + return types, e.g. "(ctx context.Context) (User, error)"
	Package      string                  // Go package name
	Concepts     []Concept               // learned concept memberships with confidence, ascending by ID
	DocComment   string                  // godoc comment above the declaration
	Exported     bool                    // true if the function name is exported
	ReceiverType string                  // e.g. "*Server"; empty for plain functions
	Lang         string                  // the frontend that produced this unit; see gofront.Lang
	Callees      []string                // frontend-derived outgoing call names
	Fingerprint  fingerprint.Fingerprint // deterministic static summary of the body
	Signals      TagSignals              // AST-level evidence channels the tagger reads
	Generated    bool                    // the file carries Go's "Code generated ... DO NOT EDIT." marker
}

// MethodName returns the bare method name of a method unit ("Start" for
// "*Server.Start") and the plain name of a function. The one place to split a
// unit name at its receiver: the receiver can carry dots of its own in a
// generic instantiation, so "last dot" is not a safe rule anywhere else.
func MethodName(u CodeUnit) string {
	if u.ReceiverType == "" {
		return u.Name
	}
	return strings.TrimPrefix(u.Name, u.ReceiverType+".")
}

// Parse extracts all CodeUnits from the file at the given path, dispatching
// on its extension to the frontend that claims it. A file no frontend claims
// returns nil, nil — that is how the extension allowlist keeps prose, config
// and data out of the corpus by construction rather than by a heuristic that
// tries to recognise code.
func Parse(path string) ([]CodeUnit, error) {
	f, ok := frontendFor(path)
	if !ok {
		return nil, nil
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseWith(f, path, src)
}

// ParseSource extracts CodeUnits from in-memory source, choosing the frontend
// by the path's extension. It exists so tests — here, in the tagger and in
// the lexicon — can parse inline snippets without touching disk, and so
// `doppel query` can read a proposed function from stdin.
func ParseSource(path string, src []byte) ([]CodeUnit, error) {
	f, ok := frontendFor(path)
	if !ok {
		return nil, nil
	}
	return parseWith(f, path, src)
}

func parseWith(fe Frontend, path string, src []byte) ([]CodeUnit, error) {
	f, err := fe.Parse(path, src)
	if err != nil || f == nil {
		return nil, err
	}
	return unitsFrom(*f), nil
}

// ShouldSkipDir reports whether a directory is outside the population — what
// the go tool itself ignores, directories whose name starts with "." or "_"
// (which is what keeps _examples/ demo trees out of a library's population),
// plus vendor, testdata and build.
//
// It lives here, next to what it feeds, rather than in cmd: the bench harness
// walks the same tree and must apply the same rule, and it kept a byte-identical
// copy precisely to avoid depending on cmd. One definition, no such dependency.
func ShouldSkipDir(name string) bool {
	if name != "" && (name[0] == '.' || name[0] == '_') {
		return true
	}
	switch name {
	case "vendor", "testdata", "build":
		return true
	}
	return false
}

// Certain builds memberships asserted without reservation — confidence 1.0.
//
// It exists for the callers that legitimately have bare IDs and no grading:
// test fixtures pinning behavior that has nothing to do with confidence, and
// any consumer handed a plain list. Production tagging never uses it; the
// lexicon derives its confidences from the corpus.
func Certain(ids ...string) []Concept {
	if len(ids) == 0 {
		return nil
	}
	out := make([]Concept, len(ids))
	for i, id := range ids {
		out[i] = Concept{ID: id, Confidence: 1}
	}
	return out
}

// unitsFrom projects a parsed file onto CodeUnits.
//
// This is the neutral half of the frontend contract and the whole reason the
// IR exists: everything above it is language-specific, everything below it —
// the fingerprint, the signals, the call graph, the lexicon, and every
// corpus statistic — reads only what this produces. A frontend that fills a
// syntax.File gets all of it without writing any of it.
func unitsFrom(f syntax.File) []CodeUnit {
	var units []CodeUnit
	for _, fn := range f.Funcs {
		units = append(units, CodeUnit{
			Name:         qualifyName(fn),
			File:         f.Path,
			Lang:         f.Lang,
			StartLine:    fn.StartLine,
			Body:         fn.Source,
			Signature:    signatureOf(fn),
			Package:      f.Package,
			DocComment:   fn.Doc,
			Exported:     fn.Exported,
			ReceiverType: fn.Receiver,
			Callees:      fn.Callees,
			Fingerprint:  fingerprint.Build(&fn),
			Signals:      extractSignals(fn, f),
			Generated:    f.Generated,
		})
	}
	return units
}

// qualifyName is the "*Server.Start" naming scheme: a method keeps its
// receiver, a plain function is its own name. MethodName is the only safe
// inverse.
func qualifyName(fn syntax.Func) string {
	if fn.Receiver == "" {
		return fn.Name
	}
	return fn.Receiver + "." + fn.Name
}

// extractSignature returns "(params) (results)" for a function declaration:
// parameter and result types in declaration order, names dropped, one entry
// per declared name ("a, b int" is "int, int"), results parenthesized whenever
// any exist — "([]int) (int)", "(context.Context) (error)", "()".
//
// It prints each field's Type expression individually. The earlier version
// handed the whole *ast.FieldList to go/printer, which accepts only Expr,
// Stmt, Decl, Spec and File nodes and silently wrote nothing — so every unit
// carried an empty signature and every report's Signature column was blank.
// This string is rendered text only; fingerprint.Types is what scores.
func signatureOf(fn syntax.Func) string {
	sig := "(" + paramTypes(fn.Params) + ")"
	if len(fn.Results) > 0 {
		sig += " (" + paramTypes(fn.Results) + ")"
	}
	return sig
}

// paramTypes renders parameter types, comma-separated. Params already carry
// one entry per declared name, so arity survives without counting names here.
// An unrenderable type prints "?" rather than being dropped silently, which is
// the one way this differs from the type set the fingerprint scores.
func paramTypes(params []syntax.Param) string {
	if len(params) == 0 {
		return ""
	}
	parts := make([]string, 0, len(params))
	for _, p := range params {
		if p.Type == "" {
			parts = append(parts, "?")
			continue
		}
		parts = append(parts, p.Type)
	}
	return strings.Join(parts, ", ")
}

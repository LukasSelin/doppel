package parser

import (
	"go/ast"
	"path/filepath"
	"strings"

	"github.com/LukasSelin/doppel/internal/canon"
	"github.com/LukasSelin/doppel/internal/fingerprint"
)

// CodeUnit represents a single extracted function or method.
type CodeUnit struct {
	Name         string
	File         string
	StartLine    int
	Body         string
	Signature    string                  // parameter + return types, e.g. "(ctx context.Context) (User, error)"
	Package      string                  // Go package name
	Patterns     []string                // detected intent tags, e.g. ["retry", "http_call"]
	DocComment   string                  // godoc comment above the declaration
	Exported     bool                    // true if the function name is exported
	ReceiverType string                  // e.g. "*Server"; empty for plain functions
	Callees      []string                // AST-derived outgoing call names
	Fingerprint  fingerprint.Fingerprint // deterministic static summary of the body
	Signals      TagSignals              // AST-level evidence channels the tagger reads
	Generated    bool                    // the file carries Go's "Code generated ... DO NOT EDIT." marker

	// Canonical is the function rewritten into canon's canonical shape — a
	// deep copy, never the tree Fingerprint and Signals were built from.
	// nil when the declaration has no body.
	//
	// Nothing reads it yet. It is produced here rather than later because
	// this is where the AST exists: the same reason fingerprint.Build and
	// extractSignals run in this loop.
	//
	// The declaration keeps its own name — that is the unit's identity in
	// the package, not a binding inside it, and the call graph and every
	// report key on it. A consumer comparing two canonical trees as shapes
	// has to set the name aside itself.
	Canonical *ast.FuncDecl

	// CanonRules names the canonicalization rules that fired on this
	// function, in canon's declaration order. It is the evidence half of
	// Canonical: whatever the canonical tree is later used to claim, this
	// says which normalizations were needed to get there.
	CanonRules []canon.RuleID
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

// Parse extracts all CodeUnits from the Go file at the given path.
// Non-.go files return nil, nil.
func Parse(path string) ([]CodeUnit, error) {
	if filepath.Ext(path) != ".go" {
		return nil, nil
	}
	return parseGo(path)
}

// ParseSource extracts CodeUnits from in-memory Go source. The path is used
// only for position information and the File field. This exists so tests —
// here and in the tagger — can parse inline snippets without touching disk.
func ParseSource(path string, src []byte) ([]CodeUnit, error) {
	return parseGoSource(path, src)
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

package parser

import (
	"path/filepath"
	"strings"

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

package parser

import "path/filepath"

// CodeUnit represents a single extracted function or method.
type CodeUnit struct {
	Name         string
	File         string
	StartLine    int
	Body         string
	Signature    string   // parameter + return types, e.g. "(ctx context.Context) (User, error)"
	Package      string   // Go package name
	Patterns     []string // detected intent tags, e.g. ["retry", "http_call"]
	DocComment   string   // godoc comment above the declaration
	Exported     bool     // true if the function name is exported
	ReceiverType string   // e.g. "*Server"; empty for plain functions
	Callees      []string // AST-derived outgoing call names
}

// Parse extracts all CodeUnits from the Go file at the given path.
// Non-.go files return nil, nil.
func Parse(path string) ([]CodeUnit, error) {
	if filepath.Ext(path) != ".go" {
		return nil, nil
	}
	return parseGo(path)
}

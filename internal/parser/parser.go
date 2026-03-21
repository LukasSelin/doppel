package parser

import (
	"path/filepath"
	"strings"
)

// CodeUnit represents a single extracted function or method.
type CodeUnit struct {
	Name      string
	File      string
	StartLine int
	Body      string
	Language  string
	Signature string   // parameter + return types, e.g. "(ctx context.Context) (User, error)"; empty for non-Go
	Package   string   // Go package name; empty for non-Go
	Patterns  []string // detected intent tags, e.g. ["retry", "http_call"]
}

// Parse extracts all CodeUnits from the file at the given path.
func Parse(path string) ([]CodeUnit, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return parseGo(path)
	case ".py":
		return parseGeneric(path, "python")
	case ".js", ".mjs", ".cjs":
		return parseGeneric(path, "javascript")
	case ".ts", ".tsx":
		return parseGeneric(path, "typescript")
	case ".java":
		return parseGeneric(path, "java")
	case ".rs":
		return parseGeneric(path, "rust")
	case ".cs":
		return parseGeneric(path, "csharp")
	case ".cpp", ".cc", ".cxx":
		return parseGeneric(path, "cpp")
	case ".c":
		return parseGeneric(path, "c")
	case ".rb":
		return parseGeneric(path, "ruby")
	default:
		return nil, nil
	}
}

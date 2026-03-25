package concepter

import (
	"strings"

	"github.com/lukse/doppel/internal/parser"
)

// ConceptDoc is a structured representation of a single CodeUnit, suitable for
// embedding as an architecture-level semantic signal.
type ConceptDoc struct {
	Name         string
	Package      string
	Inputs       []string // parameter types; from Go signature
	Outputs      []string // return types; from Go signature
	Dependencies []string // external packages/services
	Callers      []string // functions that call this one; from call graph
	Patterns     []string // tagger tags
}

// Format renders the ConceptDoc into a flat text block used as the embedding input.
// Sections with zero items are omitted entirely.
func (d ConceptDoc) Format() string {
	var sb strings.Builder

	sb.WriteString("Name: " + d.Name + "\n")
	if d.Package != "" {
		sb.WriteString("Package: " + d.Package + "\n")
	}
	writeList(&sb, "Inputs", d.Inputs)
	writeList(&sb, "Outputs", d.Outputs)
	writeList(&sb, "Dependencies", d.Dependencies)
	writeList(&sb, "Callers", d.Callers)
	writeList(&sb, "Patterns", d.Patterns)

	return sb.String()
}

func writeList(sb *strings.Builder, header string, items []string) {
	if len(items) == 0 {
		return
	}
	sb.WriteString(header + ":\n")
	for _, item := range items {
		sb.WriteString("- " + item + "\n")
	}
}

// Concepter generates ConceptDocs for CodeUnits using static analysis.
type Concepter struct{}

// New creates a Concepter.
func New() *Concepter { return &Concepter{} }

// Generate produces a static ConceptDoc for the given unit.
// Callers are not set here; use the mapper to enrich with call graph data.
func (c *Concepter) Generate(unit parser.CodeUnit) ConceptDoc {
	return ConceptDoc{
		Name:     unit.Name,
		Package:  unit.Package,
		Patterns: append([]string(nil), unit.Patterns...),
	}
}

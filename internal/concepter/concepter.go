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
	DocComment   string   // godoc for the function
	Exported     bool     // whether it is an exported symbol
	ReceiverType string   // receiver type for methods
	Inputs       []string // parameter types; from Go signature
	Outputs      []string // return types; from Go signature
	Dependencies []string // external packages/services
	Callers        []string // functions that call this one; from call graph
	Callees        []string // AST-derived outgoing call edges
	Patterns       []string // tagger tags
	Role           string   // structural role: leaf, utility, orchestrator, passthrough
	CallerPatterns []string // aggregated intent tags from caller functions
	CalleePatterns []string // aggregated intent tags from callee functions
	CallerPackages []string // packages of caller functions
	CalleePackages []string // packages of callee functions
}

// Format renders the ConceptDoc into a flat text block used as the embedding input.
// Sections with zero items are omitted entirely.
func (d ConceptDoc) Format() string {
	var sb strings.Builder

	sb.WriteString("Name: " + d.Name + "\n")
	if d.DocComment != "" {
		sb.WriteString("Doc: " + d.DocComment + "\n")
	}
	if d.Package != "" {
		sb.WriteString("Package: " + d.Package + "\n")
	}
	if d.Exported {
		sb.WriteString("Exported: true\n")
	}
	if d.ReceiverType != "" {
		sb.WriteString("Receiver: " + d.ReceiverType + "\n")
	}
	writeList(&sb, "Inputs", d.Inputs)
	writeList(&sb, "Outputs", d.Outputs)
	writeList(&sb, "Dependencies", d.Dependencies)
	writeList(&sb, "Callers", d.Callers)
	writeList(&sb, "Callees", d.Callees)
	writeList(&sb, "Patterns", d.Patterns)
	if d.Role != "" {
		sb.WriteString("Role: " + d.Role + "\n")
	}
	writeList(&sb, "CallerPatterns", d.CallerPatterns)
	writeList(&sb, "CalleePatterns", d.CalleePatterns)
	writeList(&sb, "CallerPackages", d.CallerPackages)
	writeList(&sb, "CalleePackages", d.CalleePackages)

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
		Name:         unit.Name,
		Package:      unit.Package,
		DocComment:   unit.DocComment,
		Exported:     unit.Exported,
		ReceiverType: unit.ReceiverType,
		Callees:      append([]string(nil), unit.Callees...),
		Patterns:     append([]string(nil), unit.Patterns...),
	}
}

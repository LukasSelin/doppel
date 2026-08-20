package concepter

import (
	"github.com/LukasSelin/doppel/internal/parser"
)

// ConceptDoc is a structured representation of a single CodeUnit: the
// architectural context around a function, as opposed to the body itself.
// It is what the comparator scores structural overlap on.
type ConceptDoc struct {
	Name            string
	Package         string
	DocComment      string   // godoc for the function
	Exported        bool     // whether it is an exported symbol
	ReceiverType    string   // receiver type for methods
	Callers         []string // qualified names of functions that call this one; from call graph
	Callees         []string // AST-derived outgoing call edges, raw strings incl. stdlib
	ResolvedCallees []string // qualified names of repo-internal callees; from call graph
	Neighborhood    []string // depth-2 call-graph ball, qualified names, self excluded
	Patterns        []string // tagger tags
	Role            string   // structural role: leaf, utility, orchestrator, passthrough
	CallerPatterns  []string // aggregated intent tags from caller functions
	CalleePatterns  []string // aggregated intent tags from callee functions
	CallerPackages  []string // packages of caller functions
	CalleePackages  []string // packages of callee functions
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

package concepter

import (
	"github.com/LukasSelin/doppel/internal/ontology"
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
	Role            string   // structural role: leaf, utility, orchestrator, passthrough
	CallerPackages  []string // packages of caller functions
	CalleePackages  []string // packages of callee functions

	// Concepts is the unit's own learned concept memberships, ascending by ID.
	// CallerConcepts and CalleeConcepts aggregate the same from the resolved
	// call edges, each concept keeping the strongest confidence any neighbour
	// asserted — a context signal is about what the neighbourhood does at all,
	// so the surest neighbour is the one that speaks for it.
	Concepts       []parser.Concept
	CallerConcepts []parser.Concept
	CalleeConcepts []parser.Concept
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
		Concepts:     append([]parser.Concept(nil), unit.Concepts...),
	}
}

// Graded converts learned concept memberships into the weighted terms the
// ontology scorer reasons over. Confidence travels with the concept: two
// functions each barely carrying one share less evidence than two that
// unmistakably do, which the bare-tag era could not express.
//
// It lives here, rather than in each caller, because the comparator and the
// retriever both need it and doppel does not keep two copies of a conversion —
// it found four of its own and consolidated them.
func Graded(cs []parser.Concept) []ontology.WeightedTerm {
	if len(cs) == 0 {
		return nil
	}
	out := make([]ontology.WeightedTerm, len(cs))
	for i, c := range cs {
		out[i] = ontology.WeightedTerm{ID: ontology.TermID(c.ID), Weight: c.Confidence}
	}
	return out
}

package mapper

import (
	"github.com/lukse/doppel/internal/concepter"
	"github.com/lukse/doppel/internal/parser"
)

// Map converts CodeUnits into ConceptDocs, enriching each with caller
// information from the call graph.
func Map(units []parser.CodeUnit, cg concepter.CallGraph, c *concepter.Concepter) []concepter.ConceptDoc {
	docs := make([]concepter.ConceptDoc, len(units))
	for i, u := range units {
		doc := c.Generate(u)
		doc.Callers = cg[u.Name]
		docs[i] = doc
	}
	return docs
}

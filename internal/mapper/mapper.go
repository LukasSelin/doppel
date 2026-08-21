package mapper

import (
	"sort"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
)

// Map converts CodeUnits into ConceptDocs, enriching each with caller
// information, structural role, and architectural context from the call graph.
//
// Everything graph-derived speaks qualified names (concepter.QualifiedName),
// so two functions sharing a bare name in different packages no longer share
// callers, patterns, or packages. Callees deliberately stays the raw
// extractCallees strings — shared stdlib calls are weak-but-real shape
// evidence for the comparator — while ResolvedCallees carries the repo-
// internal edges the role and context signals are computed from.
func Map(units []parser.CodeUnit, g *concepter.Graph, c *concepter.Concepter) []concepter.ConceptDoc {
	// Lookup: qualified name → intent tags. Package comes from the key itself.
	patternsByQN := make(map[string][]string, len(units))
	for _, u := range units {
		patternsByQN[concepter.QualifiedName(u)] = u.Patterns
	}

	// Derive the role thresholds from this corpus's own degree distribution
	// before classifying anything: "high fan-in" should mean high for this
	// repo, not high for a hard-coded idea of one.
	fanIn := make([]int, len(units))
	fanOut := make([]int, len(units))
	for i, u := range units {
		qn := concepter.QualifiedName(u)
		fanIn[i] = len(g.Callers[qn])
		fanOut[i] = len(g.Callees[qn])
	}
	th := concepter.ThresholdsFromDegrees(fanIn, fanOut)

	docs := make([]concepter.ConceptDoc, len(units))
	for i, u := range units {
		doc := c.Generate(u)
		qn := concepter.QualifiedName(u)
		doc.Callers = g.Callers[qn]
		doc.ResolvedCallees = g.Callees[qn]
		doc.Neighborhood = g.Neighborhood(qn)

		doc.Role = concepter.ClassifyRoleAt(len(doc.Callers), len(doc.ResolvedCallees), th)
		doc.CallerPatterns = collectPatterns(doc.Callers, patternsByQN)
		doc.CalleePatterns = collectPatterns(doc.ResolvedCallees, patternsByQN)
		doc.CallerPackages = collectPackages(doc.Callers)
		doc.CalleePackages = collectPackages(doc.ResolvedCallees)

		docs[i] = doc
	}
	return docs
}

// collectPatterns aggregates and deduplicates intent tags from the named
// units, returning them sorted for deterministic output.
func collectPatterns(qualified []string, patternsByQN map[string][]string) []string {
	seen := make(map[string]bool)
	for _, qn := range qualified {
		for _, p := range patternsByQN[qn] {
			seen[p] = true
		}
	}
	return sortedKeys(seen)
}

// collectPackages derives the deduplicated, sorted package set of the named
// units straight from their qualified keys, so it stays total even for a name
// no index entry survived for.
func collectPackages(qualified []string) []string {
	seen := make(map[string]bool)
	for _, qn := range qualified {
		if pkg := concepter.KeyPackage(qn); pkg != "" {
			seen[pkg] = true
		}
	}
	return sortedKeys(seen)
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

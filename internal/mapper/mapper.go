package mapper

import (
	"sort"

	"github.com/lukse/doppel/internal/concepter"
	"github.com/lukse/doppel/internal/parser"
)

type unitInfo struct {
	patterns []string
	pkg      string
}

// Map converts CodeUnits into ConceptDocs, enriching each with caller
// information, structural role, and architectural context from the call graph.
func Map(units []parser.CodeUnit, cg concepter.CallGraph, c *concepter.Concepter) []concepter.ConceptDoc {
	// Build lookup: qualified name → patterns + package.
	index := make(map[string]unitInfo, len(units))
	for _, u := range units {
		qn := concepter.QualifiedName(u)
		index[qn] = unitInfo{patterns: u.Patterns, pkg: u.Package}
	}

	docs := make([]concepter.ConceptDoc, len(units))
	for i, u := range units {
		doc := c.Generate(u)
		doc.Callers = cg[concepter.QualifiedName(u)]

		doc.Role = concepter.ClassifyRole(len(doc.Callers), len(doc.Callees))
		doc.CallerPatterns = collectPatterns(doc.Callers, index)
		doc.CalleePatterns = collectPatterns(doc.Callees, index)
		doc.CallerPackages = collectPackages(doc.Callers, index)
		doc.CalleePackages = collectPackages(doc.Callees, index)

		docs[i] = doc
	}
	return docs
}

// collectPatterns aggregates and deduplicates intent tags from the named
// functions, returning them sorted for deterministic output.
func collectPatterns(names []string, index map[string]unitInfo) []string {
	seen := make(map[string]bool)
	for _, name := range names {
		if info, ok := index[name]; ok {
			for _, p := range info.patterns {
				seen[p] = true
			}
		}
	}
	return sortedKeys(seen)
}

// collectPackages aggregates and deduplicates package names from the named
// functions, returning them sorted for deterministic output.
func collectPackages(names []string, index map[string]unitInfo) []string {
	seen := make(map[string]bool)
	for _, name := range names {
		if info, ok := index[name]; ok && info.pkg != "" {
			seen[info.pkg] = true
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

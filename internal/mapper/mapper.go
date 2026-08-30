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
	// Lookup: qualified name → learned concepts. Package comes from the key itself.
	conceptsByQN := make(map[string][]parser.Concept, len(units))
	for _, u := range units {
		conceptsByQN[concepter.QualifiedName(u)] = u.Concepts
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
		doc.CallerConcepts = collectConcepts(doc.Callers, conceptsByQN)
		doc.CalleeConcepts = collectConcepts(doc.ResolvedCallees, conceptsByQN)
		doc.CallerPackages = collectPackages(doc.Callers)
		doc.CalleePackages = collectPackages(doc.ResolvedCallees)

		docs[i] = doc
	}
	return docs
}

// contextConcepts bounds an aggregated caller/callee concept set. A function
// with forty callers can otherwise inherit most of the corpus vocabulary, which
// is both meaningless as a signal — everything is related to everything — and
// quadratically expensive, since the scorer matches the two sets pairwise. The
// strongest few are what "the neighbourhood does this kind of work" can honestly
// mean.
const contextConcepts = 8

// collectConcepts aggregates the learned concepts of the named units, keeping
// the strongest confidence any of them asserted, bounded to the strongest
// contextConcepts and returned sorted by ID for deterministic output.
//
// Strongest rather than mean: this is a signal about what the neighbourhood
// does at all, and averaging over a wide caller set would dilute a concept one
// caller unmistakably carries into nothing.
func collectConcepts(qualified []string, conceptsByQN map[string][]parser.Concept) []parser.Concept {
	best := make(map[string]float64)
	for _, qn := range qualified {
		for _, c := range conceptsByQN[qn] {
			if c.Confidence > best[c.ID] {
				best[c.ID] = c.Confidence
			}
		}
	}
	if len(best) == 0 {
		return nil
	}
	ids := make([]string, 0, len(best))
	for id := range best {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) > contextConcepts {
		// Strongest first to choose, ties on ID so the cut is deterministic,
		// then back to ID order because that is the invariant the field
		// documents and the scorer's tie-breaks assume.
		sort.SliceStable(ids, func(i, j int) bool {
			if best[ids[i]] != best[ids[j]] {
				return best[ids[i]] > best[ids[j]]
			}
			return ids[i] < ids[j]
		})
		ids = ids[:contextConcepts]
		sort.Strings(ids)
	}
	out := make([]parser.Concept, len(ids))
	for i, id := range ids {
		out[i] = parser.Concept{ID: id, Confidence: best[id]}
	}
	return out
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

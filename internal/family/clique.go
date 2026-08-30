package family

import "github.com/LukasSelin/doppel/internal/clique"

// The enumerator itself lives in internal/clique: internal/lexicon needs the
// same maximal-clique guarantee over a different graph, and doppel does not
// keep two copies of a rule it has already written down once. What stays here
// is the adapter — the family graph presenting itself as a clique.Graph.

// Has reports whether the two vertices are adjacent. Part of clique.Graph.
func (g *graph) Has(a, b int) bool { return g.has(a, b) }

// Neighbors returns a's neighbours in ascending order. Part of clique.Graph.
func (g *graph) Neighbors(a int) []int { return g.neighbors(a) }

// maximalCliques enumerates the component's maximal cliques, returning false
// when the search exceeds budget so the caller can record the component as
// skipped rather than report a partial enumeration as if it were the answer.
func (g *graph) maximalCliques(comp []int, budget int) ([][]int, bool) {
	return clique.Maximal(g, comp, budget)
}

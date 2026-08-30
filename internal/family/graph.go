package family

import (
	"sort"

	"github.com/LukasSelin/doppel/internal/clique"
)

// edge is one unordered pair, keyed low-index-first so a pair has exactly one
// spelling.
type edge struct{ a, b int }

func key(a, b int) edge {
	if a > b {
		a, b = b, a
	}
	return edge{a, b}
}

// graph is an undirected weighted graph over unit indices.
//
// Adjacency is stored as sets for membership tests and materialized into
// sorted slices for every traversal. Nothing in this package may iterate a map
// to decide an order: doppel's invariant is that an unchanged tree produces
// byte-identical output, and Go randomises map iteration.
type graph struct {
	n         int
	adj       []map[int]bool
	weight    map[edge]float64
	evidence  map[edge]float64 // retrieval evidence mass (nats); 0 for completed edges
	completed map[edge]bool
}

func newGraph(n int) *graph {
	return &graph{
		n:         n,
		adj:       make([]map[int]bool, n),
		weight:    make(map[edge]float64),
		evidence:  make(map[edge]float64),
		completed: make(map[edge]bool),
	}
}

// add records an undirected edge. The heaviest weight and evidence win if a
// pair somehow arrives twice, so the stored values are never worse than what
// was observed. ev is the pair's retrieval evidence mass; completion-scored
// edges carry zero, which is the point — an edge retrieval never proposed
// contributes shape to a family's guarantee but no informative energy to its
// rank.
func (g *graph) add(a, b int, w, ev float64) {
	if a == b || a < 0 || b < 0 || a >= g.n || b >= g.n {
		return
	}
	if g.adj[a] == nil {
		g.adj[a] = make(map[int]bool)
	}
	if g.adj[b] == nil {
		g.adj[b] = make(map[int]bool)
	}
	g.adj[a][b] = true
	g.adj[b][a] = true
	k := key(a, b)
	if w > g.weight[k] {
		g.weight[k] = w
	}
	if ev > g.evidence[k] {
		g.evidence[k] = ev
	}
}

func (g *graph) has(a, b int) bool {
	if a < 0 || a >= g.n || g.adj[a] == nil {
		return false
	}
	return g.adj[a][b]
}

func (g *graph) markCompleted(a, b int) { g.completed[key(a, b)] = true }

// neighbors returns a's neighbours in ascending order.
func (g *graph) neighbors(a int) []int {
	if a < 0 || a >= g.n || g.adj[a] == nil {
		return nil
	}
	out := make([]int, 0, len(g.adj[a]))
	for b := range g.adj[a] {
		out = append(out, b)
	}
	sort.Ints(out)
	return out
}

// components returns the connected components with at least one edge, each
// sorted ascending, the whole list ordered by first member.
//
// Vertices are scanned in index order and each component's frontier is walked
// in index order, so the result never depends on map iteration.
func (g *graph) components() [][]int {
	return clique.Components(g.n, g.neighbors)
}

// describe turns a clique into a Family, reading the weights back off the
// graph so MinEdge is the real guarantee rather than a remembered one.
func (g *graph) describe(members []int) Family {
	f := Family{Members: append([]int(nil), members...), MinEdge: 1.0}
	sort.Ints(f.Members)
	sum, count := 0.0, 0
	for i := 0; i < len(f.Members); i++ {
		for j := i + 1; j < len(f.Members); j++ {
			k := key(f.Members[i], f.Members[j])
			w := g.weight[k]
			if w < f.MinEdge {
				f.MinEdge = w
			}
			sum += w
			count++
			if g.completed[k] {
				f.Completed++
			}
			f.Evidence += g.evidence[k]
		}
	}
	if count > 0 {
		f.MeanEdge = sum / float64(count)
	} else {
		f.MinEdge = 0
	}
	return f
}

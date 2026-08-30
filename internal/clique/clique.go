// Package clique enumerates maximal cliques and connected components over an
// arbitrary integer-vertex graph, deterministically.
//
// It was extracted from internal/family, which needed it to state a checkable
// guarantee about a near-duplicate group: every member is similar to every
// other member, rather than merely chained to one by a sequence of pairwise
// resemblances. internal/lexicon needs exactly the same guarantee for a
// different graph — features that co-occur beyond chance — and for the same
// reason, so the enumerator lives here rather than being copied. Single-linkage
// or component clustering chains through non-transitive links until a cluster
// spans things with nothing in common; a maximal clique cannot.
//
// Determinism is the whole contract. Candidate vertices are held in ascending
// slices rather than sets, the degeneracy ordering breaks ties on the lowest
// index, and the pivot is tie-broken on the lowest index. Nothing here iterates
// a map to decide an order.
package clique

import "sort"

// Graph is the minimal adjacency view the enumerator needs. Neighbors must
// return ascending indices, and Has must agree with it.
type Graph interface {
	Has(a, b int) bool
	Neighbors(a int) []int
}

// Maximal enumerates the component's maximal cliques with Bron-Kerbosch,
// pivoted, over a degeneracy ordering. It returns false when the search
// exceeds budget, so the caller can record the component as skipped rather
// than report a partial enumeration as if it were the answer.
//
// Maximal cliques, not a partition: a vertex can belong to several, and
// resolving that would mean choosing one cluster over another on evidence the
// caller does not have.
//
// Degeneracy ordering is what makes this affordable on the real input. The
// graphs are sparse — a few thousand edges over thousands of vertices — and for
// a graph of degeneracy d the outer loop is O(d*n*3^(d/3)), which is the
// difference between milliseconds and the worst case the exponent advertises.
func Maximal(g Graph, comp []int, budget int) ([][]int, bool) {
	if len(comp) == 0 {
		return nil, true
	}
	inComp := make(map[int]bool, len(comp))
	for _, v := range comp {
		inComp[v] = true
	}
	nbr := func(v int) []int {
		all := g.Neighbors(v)
		out := make([]int, 0, len(all))
		for _, u := range all {
			if inComp[u] {
				out = append(out, u)
			}
		}
		return out
	}

	var out [][]int
	calls := 0
	// Guard as a closure so every recursion path pays the same check.
	overBudget := func() bool {
		calls++
		return calls > budget
	}

	var expand func(r, p, x []int) bool
	expand = func(r, p, x []int) bool {
		if overBudget() {
			return false
		}
		if len(p) == 0 && len(x) == 0 {
			c := append([]int(nil), r...)
			sort.Ints(c)
			out = append(out, c)
			return true
		}
		pivot := choosePivot(g, p, x)
		for _, v := range Without(p, nbr(pivot)) {
			vn := nbr(v)
			// r is copied rather than appended in place: append would share
			// the backing array across sibling recursions and quietly rewrite
			// a clique already recorded.
			next := make([]int, len(r), len(r)+1)
			copy(next, r)
			if !expand(append(next, v), Intersect(p, vn), Intersect(x, vn)) {
				return false
			}
			p = Remove(p, v)
			x = Insert(x, v)
		}
		return true
	}

	order := DegeneracyOrder(comp, nbr)
	pos := make(map[int]int, len(order))
	for i, v := range order {
		pos[v] = i
	}
	for _, v := range order {
		// P is the neighbours later in the ordering, X those already handled —
		// the standard degeneracy split, which is what bounds the recursion.
		var p, x []int
		for _, u := range nbr(v) {
			if pos[u] > pos[v] {
				p = Insert(p, u)
			} else {
				x = Insert(x, u)
			}
		}
		if !expand([]int{v}, p, x) {
			return nil, false
		}
	}

	// Sorted so the caller's output does not depend on enumeration order.
	sort.SliceStable(out, func(i, j int) bool {
		if len(out[i]) != len(out[j]) {
			return len(out[i]) > len(out[j])
		}
		for k := range out[i] {
			if out[i][k] != out[j][k] {
				return out[i][k] < out[j][k]
			}
		}
		return false
	})
	return out, true
}

// Components returns the connected components with at least one edge, each
// sorted ascending, the whole list ordered by first member. n is the vertex
// count and neighbors must return ascending indices.
//
// Vertices are scanned in index order and each component's frontier is walked
// in index order, so the result never depends on map iteration.
func Components(n int, neighbors func(int) []int) [][]int {
	seen := make([]bool, n)
	var out [][]int
	for v := 0; v < n; v++ {
		if seen[v] || len(neighbors(v)) == 0 {
			continue
		}
		var comp []int
		stack := []int{v}
		seen[v] = true
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			comp = append(comp, cur)
			for _, nb := range neighbors(cur) {
				if !seen[nb] {
					seen[nb] = true
					stack = append(stack, nb)
				}
			}
		}
		sort.Ints(comp)
		out = append(out, comp)
	}
	return out
}

// choosePivot returns the vertex of P union X with the most neighbours in P,
// ties broken on the lowest index. The pivot is what stops the recursion from
// re-deriving the same clique through every permutation of its members.
func choosePivot(g Graph, p, x []int) int {
	best, bestCount := -1, -1
	for _, cand := range append(append([]int(nil), p...), x...) {
		n := 0
		for _, v := range p {
			if g.Has(cand, v) {
				n++
			}
		}
		if n > bestCount || (n == bestCount && cand < best) {
			best, bestCount = cand, n
		}
	}
	return best
}

// DegeneracyOrder repeatedly removes a lowest-degree vertex, ties broken on
// the lowest index.
func DegeneracyOrder(comp []int, nbr func(int) []int) []int {
	deg := make(map[int]int, len(comp))
	live := make(map[int]bool, len(comp))
	for _, v := range comp {
		live[v] = true
	}
	for _, v := range comp {
		n := 0
		for _, u := range nbr(v) {
			if live[u] {
				n++
			}
		}
		deg[v] = n
	}

	out := make([]int, 0, len(comp))
	for len(out) < len(comp) {
		pick, pickDeg := -1, 0
		for _, v := range comp { // ascending: the tie-break, not a scan artifact
			if !live[v] {
				continue
			}
			if pick == -1 || deg[v] < pickDeg {
				pick, pickDeg = v, deg[v]
			}
		}
		live[pick] = false
		out = append(out, pick)
		for _, u := range nbr(pick) {
			if live[u] {
				deg[u]--
			}
		}
	}
	return out
}

// The four slice-set helpers below all keep ascending order, which is what
// makes every traversal above order-independent of insertion history.

// Intersect returns the ascending intersection of two ascending slices.
func Intersect(a, b []int) []int {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	in := make(map[int]bool, len(b))
	for _, v := range b {
		in[v] = true
	}
	out := make([]int, 0, len(a))
	for _, v := range a { // a is already ascending
		if in[v] {
			out = append(out, v)
		}
	}
	return out
}

// Without returns the members of a that are not in b, order preserved.
func Without(a, b []int) []int {
	if len(b) == 0 {
		return append([]int(nil), a...)
	}
	skip := make(map[int]bool, len(b))
	for _, v := range b {
		skip[v] = true
	}
	out := make([]int, 0, len(a))
	for _, v := range a {
		if !skip[v] {
			out = append(out, v)
		}
	}
	return out
}

// Remove returns a without v.
func Remove(a []int, v int) []int {
	out := make([]int, 0, len(a))
	for _, x := range a {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

// Insert returns a with v inserted in ascending position, or a unchanged when
// v is already present.
func Insert(a []int, v int) []int {
	i := sort.SearchInts(a, v)
	if i < len(a) && a[i] == v {
		return a
	}
	out := make([]int, 0, len(a)+1)
	out = append(out, a[:i]...)
	out = append(out, v)
	out = append(out, a[i:]...)
	return out
}

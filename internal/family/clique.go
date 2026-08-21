package family

import "sort"

// maximalCliques enumerates the component's maximal cliques with
// Bron-Kerbosch, pivoted, over a degeneracy ordering. It returns false when
// the search exceeds budget, so the caller can record the component as skipped
// rather than report a partial enumeration as if it were the answer.
//
// Maximal cliques, not a partition: a function can belong to several, and
// resolving that would mean choosing one family over another on evidence the
// tool does not have. The census reports both and counts the function once.
//
// Determinism comes from three places, all load-bearing: candidate vertices
// are always held in ascending-index slices rather than sets, the degeneracy
// ordering breaks its ties on the lowest index, and the pivot is the candidate
// with the most neighbours in P, again tie-broken on the lowest index. Nothing
// here iterates a map to decide an order.
//
// Degeneracy ordering is what makes this affordable on the real input. The
// pair graph is sparse — a few thousand edges over thousands of functions —
// and for a graph of degeneracy d the outer loop is O(d*n*3^(d/3)), which is
// the difference between milliseconds and the worst case the exponent
// advertises.
func (g *graph) maximalCliques(comp []int, budget int) ([][]int, bool) {
	if len(comp) == 0 {
		return nil, true
	}
	inComp := make(map[int]bool, len(comp))
	for _, v := range comp {
		inComp[v] = true
	}
	nbr := func(v int) []int {
		out := make([]int, 0, len(g.adj[v]))
		for _, u := range g.neighbors(v) {
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
			clique := append([]int(nil), r...)
			sort.Ints(clique)
			out = append(out, clique)
			return true
		}
		pivot := choosePivot(g, p, x)
		for _, v := range without(p, nbr(pivot)) {
			vn := nbr(v)
			// r is copied rather than appended in place: append would share
			// the backing array across sibling recursions and quietly rewrite
			// a clique already recorded.
			next := make([]int, len(r), len(r)+1)
			copy(next, r)
			if !expand(append(next, v), intersect(p, vn), intersect(x, vn)) {
				return false
			}
			p = remove(p, v)
			x = insert(x, v)
		}
		return true
	}

	order := degeneracyOrder(comp, nbr)
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
				p = insert(p, u)
			} else {
				x = insert(x, u)
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

// choosePivot returns the vertex of P union X with the most neighbours in P,
// ties broken on the lowest index. The pivot is what stops the recursion from
// re-deriving the same clique through every permutation of its members.
func choosePivot(g *graph, p, x []int) int {
	best, bestCount := -1, -1
	for _, cand := range append(append([]int(nil), p...), x...) {
		n := 0
		for _, v := range p {
			if g.has(cand, v) {
				n++
			}
		}
		if n > bestCount || (n == bestCount && cand < best) {
			best, bestCount = cand, n
		}
	}
	return best
}

// degeneracyOrder repeatedly removes a lowest-degree vertex, ties broken on
// the lowest index.
func degeneracyOrder(comp []int, nbr func(int) []int) []int {
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

func intersect(a, b []int) []int {
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

func without(a, b []int) []int {
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

func remove(a []int, v int) []int {
	out := make([]int, 0, len(a))
	for _, x := range a {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

func insert(a []int, v int) []int {
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

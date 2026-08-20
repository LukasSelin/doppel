package ontology

import (
	"math"
	"testing"
)

// optimalMatchSum is a brute-force maximum-weight bipartite matching, used only
// as a test oracle against the greedy matcher in SetRelatedness. Subset DP:
// process the elements of bs one at a time, tracking which elements of as are
// consumed as a bitmask. Exponential in len(as), which is fine — tag sets are
// bounded by the 9 concrete concepts.
//
// pairScore returns the value of matching as[i] with bs[j]; pairs scoring <= 0
// are never matched, mirroring the production matcher's candidate filter.
func optimalMatchSum(as, bs []string, pairScore func(a, b string) float64) float64 {
	if len(as) == 0 || len(bs) == 0 {
		return 0
	}
	m := len(as)
	size := 1 << m
	neg := math.Inf(-1)
	dp := make([]float64, size)
	for i := 1; i < size; i++ {
		dp[i] = neg
	}
	for _, b := range bs {
		next := make([]float64, size)
		copy(next, dp) // leaving b unmatched is always allowed
		for mask := 0; mask < size; mask++ {
			if dp[mask] == neg {
				continue
			}
			for i := 0; i < m; i++ {
				if mask&(1<<i) != 0 {
					continue
				}
				s := pairScore(as[i], b)
				if s <= 0 {
					continue
				}
				if v := dp[mask] + s; v > next[mask|1<<i] {
					next[mask|1<<i] = v
				}
			}
		}
		dp = next
	}
	best := 0.0
	for _, v := range dp {
		if v > best {
			best = v
		}
	}
	return best
}

// conceptLeaves returns the 9 concrete concept IDs in declaration order.
func conceptLeaves(t *testing.T) []string {
	t.Helper()
	var leaves []string
	for _, term := range Default().TermsOfKind(KindConcept) {
		if !term.Abstract {
			leaves = append(leaves, string(term.ID))
		}
	}
	if len(leaves) != 9 {
		t.Fatalf("got %d concept leaves, want 9", len(leaves))
	}
	return leaves
}

// allSubsets enumerates every non-empty subset of the given elements, in a
// deterministic order.
func allSubsets(elems []string) [][]string {
	var out [][]string
	for mask := 1; mask < 1<<len(elems); mask++ {
		var set []string
		for i := range elems {
			if mask&(1<<i) != 0 {
				set = append(set, elems[i])
			}
		}
		out = append(out, set)
	}
	return out
}

// The greedy matcher was chosen over an exact assignment algorithm on the
// claim that, on this taxonomy under uniform (Wu-Palmer) similarity, greedy
// already achieves the optimum. This test is that claim, checked exhaustively:
// every pair of non-empty subsets of the 9 leaves, 511 x 511 = 261,121 cases.
// If a taxonomy change ever breaks the equality, the fix is an exact matcher
// in production, not a relaxed test.
func TestGreedyMatchingIsOptimalUnderUniformSimilarity(t *testing.T) {
	o := Default()
	leaves := conceptLeaves(t)

	// Precompute the 9x9 similarity matrix once; Relatedness allocates per
	// call and the oracle would otherwise dominate the test suite's runtime.
	sim := make(map[[2]string]float64, len(leaves)*len(leaves))
	for _, a := range leaves {
		for _, b := range leaves {
			sim[[2]string{a, b}] = o.Relatedness(TermID(a), TermID(b))
		}
	}
	pairScore := func(a, b string) float64 { return sim[[2]string{a, b}] }

	subsets := allSubsets(leaves)
	var checked int
	for _, as := range subsets {
		for _, bs := range subsets {
			got, _ := o.SetRelatedness(as, bs)
			denom := len(as)
			if len(bs) > denom {
				denom = len(bs)
			}
			want := optimalMatchSum(as, bs, pairScore) / float64(denom)
			if math.Abs(got-want) > 1e-9 {
				t.Fatalf("SetRelatedness(%v, %v) = %.9f, optimal = %.9f", as, bs, got, want)
			}
			checked++
		}
	}
	t.Logf("greedy == optimal on %d subset pairs", checked)
}

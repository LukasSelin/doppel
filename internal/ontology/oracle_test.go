package ontology

import (
	"math"
	"math/rand"
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

// conceptLeaves returns the 14 concrete concept IDs in declaration order.
func conceptLeaves(t *testing.T) []string {
	t.Helper()
	var leaves []string
	for _, term := range Default().TermsOfKind(KindConcept) {
		if !term.Abstract {
			leaves = append(leaves, string(term.ID))
		}
	}
	if len(leaves) != 14 {
		t.Fatalf("got %d concept leaves, want 14", len(leaves))
	}
	return leaves
}

// coveringLeaves is the fixed 9-leaf subset the exhaustive oracles run over.
// Exhaustion over all 14 leaves would be (2^14-1)^2 ≈ 268M subset pairs —
// intractable — so the exhaustive half runs on a subset chosen to cover every
// structural shape the matcher can encounter: same-parent siblings at depth 3
// (http_call/grpc_call, db_access/caching, retry/circuit_breaker — the last
// pair being where the adversarial corpus stages its inversion), same-parent
// siblings at depth 2 (serialization/mapping), a depth-2 leaf against depth-3
// siblings in its own subtree (concurrency vs retry/circuit_breaker),
// cross-subtree cousins within one branch (http_call vs db_access), and
// cross-branch zeros everywhere between. file_io and logging repeat shapes
// already present (a depth-2 leaf under an abstract parent), so they are left
// to the seeded sampling over the full 14-leaf space that accompanies each
// exhaustive pass.
func coveringLeaves(t *testing.T) []string {
	t.Helper()
	subset := []TermID{
		ConHTTPCall, ConGRPCCall, ConDBAccess, ConCaching, ConCircuitBreaker,
		ConSerialization, ConMapping, ConRetry, ConConcurrency,
	}
	o := Default()
	out := make([]string, len(subset))
	for i, id := range subset {
		term, ok := o.Get(id)
		if !ok || term.Abstract || term.Kind != KindConcept {
			t.Fatalf("covering subset member %q is not a concrete concept", id)
		}
		out[i] = string(id)
	}
	return out
}

// sampledSubsetPairs yields deterministic (as, bs) subset pairs over the full
// leaf set: the left side capped at maxLeft elements so the exponential oracle
// stays affordable, the right side unrestricted. Seeded, so every run checks
// the same pairs.
func sampledSubsetPairs(leaves []string, n, maxLeft int, seed int64) [][2][]string {
	rng := rand.New(rand.NewSource(seed))
	full := 1 << len(leaves)
	pick := func(mask int) []string {
		var set []string
		for i := range leaves {
			if mask&(1<<i) != 0 {
				set = append(set, leaves[i])
			}
		}
		return set
	}
	var out [][2][]string
	for len(out) < n {
		a := rng.Intn(full-1) + 1
		if popcount(a) > maxLeft {
			continue
		}
		b := rng.Intn(full-1) + 1
		out = append(out, [2][]string{pick(a), pick(b)})
	}
	return out
}

func popcount(v int) int {
	n := 0
	for ; v != 0; v &= v - 1 {
		n++
	}
	return n
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
// already achieves the optimum. This test is that claim, checked two ways:
// exhaustively over the covering subset (511 x 511 = 261,121 cases) and by
// seeded sampling over the full 14-leaf space. If a taxonomy change ever
// breaks the equality, the fix is an exact matcher in production, not a
// relaxed test.
func TestGreedyMatchingIsOptimalUnderUniformSimilarity(t *testing.T) {
	o := Default()
	leaves := conceptLeaves(t)

	// Precompute the similarity matrix once; Relatedness allocates per call
	// and the oracle would otherwise dominate the test suite's runtime.
	sim := make(map[[2]string]float64, len(leaves)*len(leaves))
	for _, a := range leaves {
		for _, b := range leaves {
			sim[[2]string{a, b}] = o.Relatedness(TermID(a), TermID(b))
		}
	}
	pairScore := func(a, b string) float64 { return sim[[2]string{a, b}] }

	check := func(as, bs []string) {
		got, _ := o.SetRelatedness(as, bs)
		denom := len(as)
		if len(bs) > denom {
			denom = len(bs)
		}
		want := optimalMatchSum(as, bs, pairScore) / float64(denom)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("SetRelatedness(%v, %v) = %.9f, optimal = %.9f", as, bs, got, want)
		}
	}

	subsets := allSubsets(coveringLeaves(t))
	var checked int
	for _, as := range subsets {
		for _, bs := range subsets {
			check(as, bs)
			checked++
		}
	}
	sampled := sampledSubsetPairs(leaves, 50_000, 7, 1)
	for _, p := range sampled {
		check(p[0], p[1])
	}
	t.Logf("greedy == optimal on %d exhaustive + %d sampled subset pairs", checked, len(sampled))
}

package bench

import (
	"os"
	"slices"
	"testing"
)

// TestFusionRankingsAt lists doppel, the clone detectors it is fused with, and
// their Reciprocal Rank Fusion, for `historylabel compare` to judge on T..pin.
// The inputs are cloneOutcomeLists' lists unchanged, so they reproduce the
// clone-outcome study pair for pair. It scores nothing. See
// examples/fusion-outcomes.md for the pre-registered design.
//
//	DOPPEL_BENCH_FUSION_AT=<tree at T> DOPPEL_BENCH_FUSION_CORPUS=<name> \
//	DOPPEL_BENCH_FUSION_CLONES=<dir of dupl clone files for that tree> \
//	DOPPEL_BENCH_FUSION_OUT=<file> go test ./internal/bench/ -run TestFusionRankingsAt -v
func TestFusionRankingsAt(t *testing.T) {
	root := os.Getenv("DOPPEL_BENCH_FUSION_AT")
	if root == "" {
		t.Skip("set DOPPEL_BENCH_FUSION_AT to a tree to rank")
	}
	corpus := os.Getenv("DOPPEL_BENCH_FUSION_CORPUS")
	out := os.Getenv("DOPPEL_BENCH_FUSION_OUT")
	clones := os.Getenv("DOPPEL_BENCH_FUSION_CLONES")
	if corpus == "" || out == "" || clones == "" {
		t.Fatal("DOPPEL_BENCH_FUSION_CORPUS, _OUT and _CLONES are required")
	}
	br := rankingsRun(t, root, corpus)
	dupl, _, err := readCloneLists(br, corpus, clones, cloneOutcomeDetectors)
	if err != nil {
		t.Fatal(err)
	}
	if len(dupl) != len(cloneOutcomeDetectors) {
		t.Fatalf("[%s] %d of %d dupl runs found under %s", corpus, len(dupl), len(cloneOutcomeDetectors), clones)
	}
	all, ranked := cloneOutcomeLists(br, dupl, rankingsTop)
	byName := map[string][]ref{}
	for i, n := range all {
		byName[n] = ranked[i]
	}
	names := []string{fusionBase}
	lists := [][]ref{byName[fusionBase]}
	for _, in := range fusionInputs {
		if byName[in] == nil {
			t.Fatalf("[%s] no %q list", corpus, in)
		}
		names = append(names, in)
		lists = append(lists, byName[in])
	}
	for _, in := range fusionInputs {
		name := "RRF(" + fusionBase + ", " + in + ")"
		fused := rrfFuse([][]ref{byName[fusionBase], byName[in]}, rrfK, rankingsTop)
		both, base, other := provenance(fused[:min(len(fused), 100)], byName[fusionBase], byName[in])
		t.Logf("[%s] %s: %d pairs listed; top 100: %d in both inputs, %d doppel only, %d %s only",
			corpus, name, len(fused), both, base, other, in)
		names = append(names, name)
		lists = append(lists, fused)
	}
	writeRankingsFile(t, out, rankingsFileOf(br, corpus, root, names, lists, rankingsTop))
}

// fusionBase is fused with each of fusionInputs, the pre-registered clone
// detectors, under their clone-outcome study names.
const fusionBase = "doppel"

var fusionInputs = []string{"dupl (t=100, default)", "dupl (t=50)", "token clones (≥50 nodes)"}

// rrfK is Reciprocal Rank Fusion's k, the published default, fixed in advance.
const rrfK = 60

// rrfFuse merges ranked lists by Reciprocal Rank Fusion (Cormack, Clarke &
// Büttcher 2009): each pair scores Σ 1/(k + rank) over the lists that hold it,
// rank 1-based within each list cut at top. Ties break on the better best
// single rank, then on (a, b). The sum runs in list order, so it is the same
// float on every run. rankBy is not reused because its only tie-break is
// (a, b).
func rrfFuse(lists [][]ref, k float64, top int) []ref {
	type acc struct {
		score float64
		best  int
	}
	seen := map[ref]*acc{}
	var order []ref
	for _, l := range lists {
		for r, x := range l[:min(len(l), top)] {
			a := seen[x]
			if a == nil {
				a = &acc{best: r + 1}
				seen[x] = a
				order = append(order, x)
			}
			a.score += 1 / (k + float64(r+1))
			a.best = min(a.best, r+1)
		}
	}
	slices.SortFunc(order, func(x, y ref) int {
		ax, ay := seen[x], seen[y]
		switch {
		case ax.score > ay.score:
			return -1
		case ax.score < ay.score:
			return 1
		}
		if ax.best != ay.best {
			return ax.best - ay.best
		}
		if x.a != y.a {
			return x.a - y.a
		}
		return x.b - y.b
	})
	return order[:min(len(order), top)]
}

// provenance counts the pairs of l that are in both inputs, in a only, and in
// b only.
func provenance(l, a, b []ref) (both, onlyA, onlyB int) {
	inA, inB := map[ref]bool{}, map[ref]bool{}
	for _, x := range a {
		inA[x] = true
	}
	for _, x := range b {
		inB[x] = true
	}
	for _, x := range l {
		switch {
		case inA[x] && inB[x]:
			both++
		case inA[x]:
			onlyA++
		case inB[x]:
			onlyB++
		}
	}
	return both, onlyA, onlyB
}

// TestRRFFuse pins the fusion to a hand-computed example and each tie-break.
func TestRRFFuse(t *testing.T) {
	p, q, r, s := ref{0, 1}, ref{2, 3}, ref{4, 5}, ref{6, 7}
	// p: 1/61 + 1/62; q: 1/62 + 1/61 (equal sum, equal best rank 1: (a, b)
	// decides); r: 1/63 only; s: 1/61 only in the second list, cut away at top 2.
	a := []ref{p, q, r}
	b := []ref{q, p, s}
	got := rrfFuse([][]ref{a, b}, 60, 3)
	if want := []ref{p, q, r}; !slices.Equal(got, want) {
		t.Errorf("fused %v, want %v", got, want)
	}
	// Cut at 2: r and s drop out of their lists, so only p and q remain.
	if got := rrfFuse([][]ref{a, b}, 60, 2); !slices.Equal(got, []ref{p, q}) {
		t.Errorf("top 2: fused %v, want [p q]", got)
	}

	// Agreement outranks a single top rank: x at rank 30 in both scores
	// 2/90 > 1/61, the rank-1 pair of one list.
	var l1, l2 []ref
	for i := range 30 {
		l1 = append(l1, ref{100 + i, 200 + i})
		l2 = append(l2, ref{300 + i, 400 + i})
	}
	x := ref{1, 2}
	l1 = append(l1[:29], x)
	l2 = append(l2[:29], x)
	if got := rrfFuse([][]ref{l1, l2}, 60, 500); got[0] != x {
		t.Errorf("first fused pair %v, want the agreed pair %v", got[0], x)
	}

	// Equal score and equal best rank: u at rank 1 of one list, v at rank 1 of
	// the other, so (a, b) decides.
	u, v := ref{5, 9}, ref{3, 9}
	if got := rrfFuse([][]ref{{u}, {v}}, 60, 500); !slices.Equal(got, []ref{v, u}) {
		t.Errorf("equal score and best rank: %v, want (a, b) order [v u]", got)
	}
	// Equal score, different best rank: with k = 0, y at rank 2 of both lists
	// scores 1/2 + 1/2 = 1, and z and w at rank 1 of one list score 1/1 = 1.
	// The best single rank puts z and w (rank 1) before y (rank 2) although y's
	// unit index is lowest; between z and w, (a, b) decides.
	y, z, w := ref{0, 9}, ref{8, 9}, ref{7, 8}
	if got := rrfFuse([][]ref{{z, y}, {w, y}}, 0, 500); !slices.Equal(got, []ref{w, z, y}) {
		t.Errorf("equal score: %v, want [w z y]", got)
	}

	// Determinism: the same inputs fuse to the same list every time.
	first := rrfFuse([][]ref{l1, l2, a, b}, 60, 500)
	for range 20 {
		if again := rrfFuse([][]ref{l1, l2, a, b}, 60, 500); !slices.Equal(again, first) {
			t.Fatal("fusion is not deterministic")
		}
	}
}

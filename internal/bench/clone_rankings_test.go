package bench

import (
	"os"
	"slices"
	"testing"

	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parallel"
	"github.com/LukasSelin/doppel/internal/parser"
)

// TestCloneRankingsAt ranks one tree at revision T under doppel and under
// clone detectors that find their own pairs — dupl, and all-pairs token and
// code-shape search with a size floor — and writes the lists in
// TestRankingsAt's format, for `historylabel compare` to judge on T..pin. It
// scores nothing. See examples/clone-outcomes.md for the pre-registered
// design.
//
//	DOPPEL_BENCH_CLONERANK_AT=<tree at T> DOPPEL_BENCH_CLONERANK_CORPUS=<name> \
//	DOPPEL_BENCH_CLONERANK_CLONES=<dir of dupl clone files for that tree> \
//	DOPPEL_BENCH_CLONERANK_OUT=<file> go test ./internal/bench/ -run TestCloneRankingsAt -v
//
// Unlike TestRankingsAt, no list but doppel's is restricted to doppel's
// retrieval union: each detector searches the whole population.
func TestCloneRankingsAt(t *testing.T) {
	root := os.Getenv("DOPPEL_BENCH_CLONERANK_AT")
	if root == "" {
		t.Skip("set DOPPEL_BENCH_CLONERANK_AT to a tree to rank")
	}
	corpus := os.Getenv("DOPPEL_BENCH_CLONERANK_CORPUS")
	out := os.Getenv("DOPPEL_BENCH_CLONERANK_OUT")
	clones := os.Getenv("DOPPEL_BENCH_CLONERANK_CLONES")
	if corpus == "" || out == "" || clones == "" {
		t.Fatal("DOPPEL_BENCH_CLONERANK_CORPUS, _OUT and _CLONES are required")
	}
	br := rankingsRun(t, root, corpus)
	lists, notes, err := readCloneLists(br, corpus, clones, cloneOutcomeDetectors)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range notes {
		t.Logf("[%s] %s", corpus, n)
	}
	if len(lists) != len(cloneOutcomeDetectors) {
		t.Fatalf("[%s] %d of %d dupl runs found under %s", corpus, len(lists), len(cloneOutcomeDetectors), clones)
	}
	names, ranked := cloneOutcomeLists(br, lists, rankingsTop)
	for i, n := range names {
		t.Logf("[%s] %s: %d pairs listed", corpus, n, min(len(ranked[i]), rankingsTop))
	}
	writeRankingsFile(t, out, rankingsFileOf(br, corpus, root, names, ranked, rankingsTop))
}

// cloneOutcomeDetectors are the dupl runs the study scores: the tool's
// default threshold and the pre-registered sensitivity one.
var cloneOutcomeDetectors = []cloneMethod{
	{"dupl (t=100, default)", "dupl-t100"},
	{"dupl (t=50)", "dupl-t50"},
}

// cloneOutcomeLists is every method's list, in the pre-registered order: doppel
// over its union exactly as the ranker-outcome study lists it, the dupl runs,
// then the all-pairs searches. The size floors are in syntax nodes because
// dupl's tokens are serialized AST nodes.
func cloneOutcomeLists(br *baselineRun, dupl []cloneList, top int) ([]string, [][]ref) {
	r := br.run
	names := []string{"doppel"}
	lists := [][]ref{doppelOrder(r, r.Pairs)}
	for _, c := range dupl {
		names = append(names, c.name)
		lists = append(lists, rankBy(c.refs, c.key))
	}
	shingles := func(a, b int) float64 {
		return shingleJaccard(r.Units[a].Fingerprint.Shingles, r.Units[b].Fingerprint.Shingles)
	}
	shape := func(a, b int) float64 {
		return fingerprint.Similarity(r.Units[a].Fingerprint, r.Units[b].Fingerprint, r.WL).Score
	}
	for _, m := range []struct {
		name  string
		floor int
		score func(a, b int) float64
	}{
		{"token clones (≥100 nodes)", 100, shingles},
		{"token clones (≥50 nodes)", 50, shingles},
		{"token clones (no floor)", 0, shingles},
		{"code-shape (≥50 nodes)", 50, shape},
	} {
		names = append(names, m.name)
		lists = append(lists, allPairsTop(r.Units, m.floor, m.score, top))
	}
	return names, lists
}

// allPairsTop is an exhaustive detector: every same-build-unit pair whose two
// sides both have at least floor syntax nodes, scored, and the top pairs with
// a score above 0 kept in rankBy's order (score descending, then (a, b)).
//
// It is exact without holding every pair: each row a keeps its own top
// partners among b > a, written at index a, and the global top is drawn from
// the union of the rows' — a pair outside its row's top cannot be in the
// global one.
func allPairsTop(units []parser.CodeUnit, floor int, score func(a, b int) float64, top int) []ref {
	var eligible []int
	for i, u := range units {
		if u.Fingerprint.Nodes >= floor {
			eligible = append(eligible, i)
		}
	}
	rows := make([][]ref, len(eligible))
	rowKeys := make([][]float64, len(eligible))
	parallel.BlocksWith(len(eligible), 8, 1, func() *[]rowHit { return new([]rowHit) }, func(scratch *[]rowHit, i int) {
		a := eligible[i]
		hits := (*scratch)[:0]
		for _, b := range eligible[i+1:] {
			if !parser.SameBuildUnit(units[a], units[b]) {
				continue
			}
			if k := score(a, b); k > 0 {
				hits = append(hits, rowHit{b, k})
			}
		}
		// Within a row a is fixed, so rankBy's order is key desc, then b asc.
		slices.SortFunc(hits, func(x, y rowHit) int {
			switch {
			case x.k > y.k:
				return -1
			case x.k < y.k:
				return 1
			}
			return x.b - y.b
		})
		hits = hits[:min(len(hits), top)]
		rows[i] = make([]ref, len(hits))
		rowKeys[i] = make([]float64, len(hits))
		for j, h := range hits {
			rows[i][j] = ref{a, h.b}
			rowKeys[i][j] = h.k
		}
		*scratch = hits
	})
	var refs []ref
	var keys []float64
	for i := range rows {
		refs = append(refs, rows[i]...)
		keys = append(keys, rowKeys[i]...)
	}
	l := rankBy(refs, keys)
	return l[:min(len(l), top)]
}

type rowHit struct {
	b int
	k float64
}

// TestAllPairsTopIsExact pins allPairsTop to the plain exhaustive ranking:
// every eligible pair scored, rankBy, cut — ties included, since the keys
// below repeat on purpose.
func TestAllPairsTopIsExact(t *testing.T) {
	const n = 60
	units := make([]parser.CodeUnit, n)
	for i := range units {
		units[i].Fingerprint.Nodes = 10 + i%7*10
	}
	score := func(a, b int) float64 { return float64((a*31+b*17)%11) / 10 }
	for _, floor := range []int{0, 40} {
		for _, top := range []int{1, 5, 37, 2000} {
			var refs []ref
			var keys []float64
			for a := range n {
				for b := a + 1; b < n; b++ {
					if units[a].Fingerprint.Nodes < floor || units[b].Fingerprint.Nodes < floor {
						continue
					}
					if k := score(a, b); k > 0 {
						refs = append(refs, ref{a, b})
						keys = append(keys, k)
					}
				}
			}
			want := rankBy(refs, keys)
			want = want[:min(len(want), top)]
			if got := allPairsTop(units, floor, score, top); !slices.Equal(got, want) {
				t.Errorf("floor %d top %d: got %d pairs, want %d (first diff at %v)", floor, top, len(got), len(want), firstDiff(got, want))
			}
		}
	}
}

func firstDiff(a, b []ref) int {
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return i
		}
	}
	return min(len(a), len(b))
}

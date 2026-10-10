package bench

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// TestSizeRankingsAt lists the clone-outcome study's methods beside the
// size-aware rank variants of examples/size-aware-rank.md, for `historylabel
// compare -sweep-scope commit` to judge on T..pin. It scores nothing.
//
//	DOPPEL_BENCH_SIZERANK_AT=<tree at T> DOPPEL_BENCH_SIZERANK_CORPUS=<name> \
//	DOPPEL_BENCH_SIZERANK_CLONES=<dir of dupl clone files for that tree> \
//	DOPPEL_BENCH_SIZERANK_OUT=<file> go test ./internal/bench/ -run TestSizeRankingsAt -v
//
// DOPPEL_BENCH_SIZERANK_FEATURES=<file> is the development seam: it adds the
// whole retrieval union as one more list, so every union pair is judged once,
// and writes each listed pair's rank-key factors and size features to the
// file, so a candidate key can be scored offline against the judged outcomes.
// Under the commit sweep scope a pair's judgment does not depend on what else
// is listed, which is what makes judging the union once legitimate.
func TestSizeRankingsAt(t *testing.T) {
	root := os.Getenv("DOPPEL_BENCH_SIZERANK_AT")
	if root == "" {
		t.Skip("set DOPPEL_BENCH_SIZERANK_AT to a tree to rank")
	}
	corpus := os.Getenv("DOPPEL_BENCH_SIZERANK_CORPUS")
	out := os.Getenv("DOPPEL_BENCH_SIZERANK_OUT")
	clones := os.Getenv("DOPPEL_BENCH_SIZERANK_CLONES")
	if corpus == "" || out == "" || clones == "" {
		t.Fatal("DOPPEL_BENCH_SIZERANK_CORPUS, _OUT and _CLONES are required")
	}
	br := rankingsRun(t, root, corpus)
	dupl, _, err := readCloneLists(br, corpus, clones, cloneOutcomeDetectors)
	if err != nil {
		t.Fatal(err)
	}
	if len(dupl) != len(cloneOutcomeDetectors) {
		t.Fatalf("[%s] %d of %d dupl runs found under %s", corpus, len(dupl), len(cloneOutcomeDetectors), clones)
	}
	names, lists := cloneOutcomeLists(br, dupl, rankingsTop)
	sz := newSizeFeatures(br.run)
	for _, v := range sizeVariants {
		names = append(names, v.name)
		lists = append(lists, v.list(br.run, sz, rankingsTop))
	}
	for i, n := range names {
		t.Logf("[%s] %s: %d pairs listed", corpus, n, min(len(lists[i]), rankingsTop))
	}
	for i := range lists {
		lists[i] = lists[i][:min(len(lists[i]), rankingsTop)]
	}
	if feat := os.Getenv("DOPPEL_BENCH_SIZERANK_FEATURES"); feat != "" {
		pool := make([]ref, len(br.run.Pairs))
		for i, p := range br.run.Pairs {
			pool[i] = refOf(p)
		}
		names = append(names, sizeDevPool)
		lists = append(lists, pool)
		writeSizeFeatures(t, feat, br, sz, lists)
	}
	writeRankingsFile(t, out, rankingsFileOf(br, corpus, root, names, lists, 0))
}

// sizeDevPool names the development list that holds the whole union.
const sizeDevPool = "union (development pool)"

// sizeVariant is one rank variant: a key over the union's pairs, ranked by
// analyzer.SortForReportBy with no diversity cap, as doppel's own list is.
type sizeVariant struct {
	name string
	key  func(r *Run, sz *sizeFeatures, p analyzer.SimilarPair) float64
}

func (v sizeVariant) list(r *Run, sz *sizeFeatures, top int) []ref {
	cp := slices.Clone(r.Pairs)
	kept, _ := analyzer.SortForReportBy(cp, v.memo(r, sz), top, 0)
	out := make([]ref, len(kept))
	for i, p := range kept {
		out[i] = refOf(p)
	}
	return out
}

// memo is v's key computed once per pair: a sort asks for a pair's key at
// every comparison, and the subtree cover is too costly to recompute there.
func (v sizeVariant) memo(r *Run, sz *sizeFeatures) func(analyzer.SimilarPair) float64 {
	keys := make(map[ref]float64, len(r.Pairs))
	for _, p := range r.Pairs {
		keys[refOf(p)] = v.key(r, sz, p)
	}
	return func(p analyzer.SimilarPair) float64 { return keys[refOf(p)] }
}

// sizeVariants are the three variants examples/size-aware-rank.md locked
// before the test windows were read. Each is the production key (code-shape
// and trophic squared, retrieval mass, overlap, the test discount) times one
// size term; the names carry historylabel size-summary's variant prefix.
//
//   - K-size: the nodes of the smaller body the larger one also holds,
//     containment × the smaller node count. Cheap: both factors are on every
//     production pair already.
//   - K-subtree: the nodes in maximal canonical subtrees of at least 10 nodes
//     both bodies hold verbatim, plus 10, so a pair sharing none keeps a key.
//     The rename-proof analogue of a clone detector's clone length.
//   - K-floor: K-subtree's size as a soft clone floor at 50 nodes (dupl's
//     sensitivity threshold): (covered nodes + 10) / 50, at most 1. It
//     rewards no size above the floor; it only discounts a pair below it. The
//     one development candidate that held every golden-label guardrail.
var sizeVariants = []sizeVariant{
	{"variant: K-size", sizedKey(2, containedNodes)},
	{"variant: K-subtree", sizedKey(2, plus(coveredNodes(10), 10))},
	{"variant: K-floor", sizedKey(2, capped(plus(coveredNodes(10), 10), 50))},
}

// sizedKey is the production rank key with code-shape at shapePower,
// multiplied by a duplicated-size term. Everything else in the key —
// retrieval mass, overlap, trophic², the test discount — is RankKey's own.
func sizedKey(shapePower float64, size func(sz *sizeFeatures, p analyzer.SimilarPair) float64) func(r *Run, sz *sizeFeatures, p analyzer.SimilarPair) float64 {
	o := analyzer.DefaultRankOptions()
	o.ShapePower = shapePower
	return func(r *Run, sz *sizeFeatures, p analyzer.SimilarPair) float64 {
		return analyzer.RankKey(p, o, r.Units) * size(sz, p)
	}
}

// containedNodes is how many of the smaller body's nodes the larger one also
// has, by the WL containment: containment × the smaller node count.
func containedNodes(sz *sizeFeatures, p analyzer.SimilarPair) float64 {
	return p.Breakdown.Containment * float64(min(sz.units[p.AIdx].Fingerprint.Nodes, sz.units[p.BIdx].Fingerprint.Nodes))
}

// coveredNodes is sharedSubtree at a minimum subtree size.
func coveredNodes(minSize int) func(sz *sizeFeatures, p analyzer.SimilarPair) float64 {
	return func(sz *sizeFeatures, p analyzer.SimilarPair) float64 {
		return float64(sz.sharedSubtree(p.AIdx, p.BIdx, minSize))
	}
}

// sizeCandidates is every key the development log tried in Go, the locked
// variants among them, so the golden guardrail can be re-run on each.
var sizeCandidates = []sizeVariant{
	{"shape¹", sizedKey(1, func(*sizeFeatures, analyzer.SimilarPair) float64 { return 1 })},
	{"shape² × contained nodes", sizedKey(2, containedNodes)},
	{"shape¹ × contained nodes", sizedKey(1, containedNodes)},
	{"shape² × covered nodes (≥5)", sizedKey(2, coveredNodes(5))},
	{"shape¹ × covered nodes (≥5)", sizedKey(1, coveredNodes(5))},
	{"shape² × covered nodes (≥10)", sizedKey(2, coveredNodes(10))},
	{"shape¹ × covered nodes (≥10)", sizedKey(1, coveredNodes(10))},
	{"shape² × √covered nodes (≥10)", sizedKey(2, powered(coveredNodes(10), 0.5))},
	{"shape² × √contained nodes", sizedKey(2, powered(containedNodes, 0.5))},
	{"shape² × (covered nodes (≥10) + 10)", sizedKey(2, plus(coveredNodes(10), 10))},
	{"shape² × (covered nodes (≥5) + 10)", sizedKey(2, plus(coveredNodes(5), 10))},
	{"shape² × (contained nodes + 10)", sizedKey(2, plus(containedNodes, 10))},
	{"shape² × min(1, (covered nodes (≥10) + 10) / 100)", sizedKey(2, capped(plus(coveredNodes(10), 10), 100))},
	{"shape² × min(1, (covered nodes (≥10) + 10) / 200)", sizedKey(2, capped(plus(coveredNodes(10), 10), 200))},
	{"shape² × min(1, (contained nodes + 10) / 100)", sizedKey(2, capped(plus(containedNodes, 10), 100))},
	{"shape² × min(1, (contained nodes + 10) / 200)", sizedKey(2, capped(plus(containedNodes, 10), 200))},
	{"shape² × min(1, smaller nodes / 100)", sizedKey(2, capped(smallerNodes, 100))},
	{"shape² × min(1, (covered nodes (≥10) + 10) / 50)", sizedKey(2, capped(plus(coveredNodes(10), 10), 50))},
	{"shape² × min(1, smaller nodes / 50)", sizedKey(2, capped(smallerNodes, 50))},
}

// smallerNodes is the smaller body's syntax-node count.
func smallerNodes(sz *sizeFeatures, p analyzer.SimilarPair) float64 {
	return float64(min(sz.units[p.AIdx].Fingerprint.Nodes, sz.units[p.BIdx].Fingerprint.Nodes))
}

// capped is a size term as a soft clone floor: the fraction of floor it
// reaches, at most 1. Pairs at or above floor rank as production does; a pair
// below it is discounted in proportion.
func capped(size func(sz *sizeFeatures, p analyzer.SimilarPair) float64, floor float64) func(sz *sizeFeatures, p analyzer.SimilarPair) float64 {
	return func(sz *sizeFeatures, p analyzer.SimilarPair) float64 { return min(1, size(sz, p)/floor) }
}

// plus adds a pseudo-count to a size term, so a pair with none of it keeps a
// key rather than falling to 0.
func plus(size func(sz *sizeFeatures, p analyzer.SimilarPair) float64, c float64) func(sz *sizeFeatures, p analyzer.SimilarPair) float64 {
	return func(sz *sizeFeatures, p analyzer.SimilarPair) float64 { return size(sz, p) + c }
}

// powered raises a size term to p.
func powered(size func(sz *sizeFeatures, p analyzer.SimilarPair) float64, pow float64) func(sz *sizeFeatures, p analyzer.SimilarPair) float64 {
	return func(sz *sizeFeatures, p analyzer.SimilarPair) float64 { return math.Pow(size(sz, p), pow) }
}

// TestSizeVariantsGolden scores the production key, every development
// candidate and every locked variant against the committed cobra hand labels
// through ScoreBy — the same scorer, cap and assertions TestGoldenCorpora
// uses — and logs each scorecard. It asserts nothing: a variant failing the
// guardrail is a finding of examples/size-aware-rank.md, not a broken build.
func TestSizeVariantsGolden(t *testing.T) {
	c, ok := Find("cobra")
	if !ok || !Present(c) {
		t.Skip("cobra not fetched; run `task corpora`")
	}
	corpus, err := Path(c)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "examples", "labels", "cobra.labels.json"))
	if err != nil {
		t.Fatal(err)
	}
	lf, err := ParseLabels(data)
	if err != nil {
		t.Fatal(err)
	}
	units, err := Load(corpus, Population(lf.Population))
	if err != nil {
		t.Fatal(err)
	}
	run := Analyze(units, retriever.DefaultOptions())
	run.Root = corpus
	sz := newSizeFeatures(run)
	rows := sizeCheckRows()[:1] // production
	rows = append(rows, sizeCandidates...)
	rows = append(rows, sizeVariants...)
	t.Logf("| key | merge mean | refactor mean | false-positive mean | merges in top 50 | merges never retrieved | FP above worst merge | FP in top 20 |")
	for _, v := range rows {
		sc := ScoreBy(run, lf, v.memo(run, sz))
		t.Logf("| %s | %.1f | %.1f | %.1f | %d of %d | %d | %d | %d |", v.name,
			sc.MeanRank["merge"], sc.MeanRank["refactor"], sc.MeanRank["false_positive"],
			sc.MergeInTop50, sc.MergeTotal, len(sc.MergeMissing), len(sc.FPAboveMerge), len(sc.FPInTop20))
		if testing.Verbose() {
			ranks := map[string][]int{}
			for _, r := range sc.Results {
				ranks[r.Label.Class] = append(ranks[r.Label.Class], r.Rank)
			}
			t.Logf("    ranks: merge %v refactor %v false_positive %v", ranks["merge"], ranks["refactor"], ranks["false_positive"])
		}
	}
}

// sizeFeatures is each unit's canonical subtree inventory: every node's
// hash-cons identity and size, for the shared-subtree features.
type sizeFeatures struct {
	units []parser.CodeUnit
	trees []consTree
}

// consTree is one canonical body with every node's Merkle hash (fingerprint.
// Cons, whose post-order this walk mirrors) and subtree size.
type consTree struct {
	root *syntax.Node
	hash map[*syntax.Node]uint64
	size map[*syntax.Node]int
}

func newSizeFeatures(r *Run) *sizeFeatures {
	sz := &sizeFeatures{units: r.Units, trees: make([]consTree, len(r.Units))}
	for i, u := range r.Units {
		sz.trees[i] = consTreeOf(u.Canonical)
	}
	return sz
}

// consTreeOf pairs fingerprint.Cons's hashes with the nodes they belong to:
// Cons emits one hash per node in syntax.Inspect's post-order, and this walk
// visits the nodes in the same order, so the i-th hash is the i-th node's.
func consTreeOf(root *syntax.Node) consTree {
	t := consTree{root: root, hash: map[*syntax.Node]uint64{}, size: map[*syntax.Node]int{}}
	if root == nil {
		return t
	}
	hashes := fingerprint.Cons(root)
	var stack []*syntax.Node
	var post []*syntax.Node
	syntax.Inspect(root, func(n *syntax.Node) bool {
		if n != nil {
			stack = append(stack, n)
			return true
		}
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		s := 1
		for _, k := range top.Kids {
			s += t.size[k.Node]
		}
		t.size[top] = s
		post = append(post, top)
		return false
	})
	if len(post) != len(hashes) {
		panic("consTreeOf: walk and fingerprint.Cons disagree on the node count")
	}
	for i, n := range post {
		t.hash[n] = hashes[i]
	}
	return t
}

// cover is how many of b's nodes lie in maximal subtrees of at least minSize
// nodes that a also holds verbatim (canonical form, so renames and canon's
// rewrites do not separate them): b is walked top down, and a subtree whose
// hash a still has an unmatched copy of is taken whole and its descendants
// skipped. It is the rename-proof analogue of a clone detector's clone
// length, summed over every clone the two bodies share.
func cover(a, b consTree, minSize int) int {
	avail := map[uint64]int{}
	for n, h := range a.hash {
		if a.size[n] >= minSize {
			avail[h]++
		}
	}
	got := 0
	var walk func(n *syntax.Node)
	walk = func(n *syntax.Node) {
		if n == nil || b.size[n] < minSize {
			return
		}
		if h := b.hash[n]; avail[h] > 0 {
			avail[h]--
			got += b.size[n]
			return
		}
		for _, k := range n.Kids {
			walk(k.Node)
		}
	}
	walk(b.root)
	return got
}

// sharedSubtree is the symmetric cover: the smaller of the two directions.
func (sz *sizeFeatures) sharedSubtree(a, b, minSize int) int {
	return min(cover(sz.trees[a], sz.trees[b], minSize), cover(sz.trees[b], sz.trees[a], minSize))
}

// largestSubtree is the size of the largest canonical subtree both bodies hold.
func (sz *sizeFeatures) largestSubtree(a, b int) int {
	in := map[uint64]bool{}
	for _, h := range sz.trees[a].hash {
		in[h] = true
	}
	best := 0
	for n, h := range sz.trees[b].hash {
		if in[h] {
			best = max(best, sz.trees[b].size[n])
		}
	}
	return best
}

// wlShared is Σ min(count) over the labels two WL bags share, per refinement
// round: at h = 0 the shared node kinds, at h = 3 the nodes whose whole
// depth-3 neighbourhood agrees.
func wlShared(a, b []fingerprint.LabelCount) [4]int {
	var s [4]int
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i].Label < b[j].Label:
			i++
		case a[i].Label > b[j].Label:
			j++
		default:
			if h := int(a[i].H); h < len(s) {
				s[h] += int(min(a[i].Count, b[j].Count))
			}
			i++
			j++
		}
	}
	return s
}

// sizeFeatureRow is one listed pair's features, for offline development.
type sizeFeatureRow struct {
	A, B           string
	InUnion        bool
	ProdKey        float64
	Total          float64
	Shape          float64
	Concept        float64
	Call           float64
	Overlap        float64
	Score          float64
	Trophic        float64
	Containment    float64
	NodesA, NodesB int
	LinesA, LinesB int
	WLShared       [4]int
	Cover5         int
	Cover10        int
	Cover20        int
	Largest        int
}

func writeSizeFeatures(t *testing.T, path string, br *baselineRun, sz *sizeFeatures, lists [][]ref) {
	t.Helper()
	r := br.run
	byRef := map[ref]analyzer.SimilarPair{}
	for _, p := range r.Pairs {
		byRef[refOf(p)] = p
	}
	seen := map[ref]bool{}
	var rows []sizeFeatureRow
	for _, l := range lists {
		for _, x := range l {
			if seen[x] {
				continue
			}
			seen[x] = true
			ua, ub := r.Units[x.a], r.Units[x.b]
			row := sizeFeatureRow{A: br.keys[x.a], B: br.keys[x.b],
				NodesA: ua.Fingerprint.Nodes, NodesB: ub.Fingerprint.Nodes,
				LinesA: unitEnd(ua) - ua.StartLine + 1, LinesB: unitEnd(ub) - ub.StartLine + 1,
				WLShared: wlShared(ua.Fingerprint.WL, ub.Fingerprint.WL),
				Cover5:   sz.sharedSubtree(x.a, x.b, 5), Cover10: sz.sharedSubtree(x.a, x.b, 10),
				Cover20: sz.sharedSubtree(x.a, x.b, 20), Largest: sz.largestSubtree(x.a, x.b)}
			if p, ok := byRef[x]; ok {
				row.InUnion = true
				row.ProdKey = analyzer.RankKey(p, analyzer.DefaultRankOptions(), r.Units)
				row.Score, row.Containment = p.Score, p.Breakdown.Containment
				if p.Retrieval != nil {
					row.Total, row.Shape, row.Concept, row.Call = p.Retrieval.Total, p.Retrieval.Shape, p.Retrieval.Concept, p.Retrieval.Call
					row.Trophic = p.Retrieval.TrophicSim
				}
				if p.Evidence != nil {
					row.Overlap = p.Evidence.OverlapScore
				}
			} else {
				bd := fingerprint.Similarity(ua.Fingerprint, ub.Fingerprint, r.WL)
				row.Score, row.Containment = bd.Score, bd.Containment
			}
			if math.IsNaN(row.Score) {
				row.Score = 0
			}
			rows = append(rows, row)
		}
	}
	data, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d feature rows to %s", len(rows), path)
}

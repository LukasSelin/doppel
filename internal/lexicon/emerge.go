package lexicon

import (
	"math"
	"sort"

	"github.com/LukasSelin/doppel/internal/clique"
	"github.com/LukasSelin/doppel/internal/parallel"
)

// emergeConcepts discovers concepts no seed accounts for.
//
// This is the path that makes the learner work on a corpus whose vocabulary
// nobody wrote a rule for — and, with an empty seed set, the only path there
// is. Features that survived the information window and no seeded concept
// claimed are clustered on their own co-occurrence: an edge when two features
// appear together in at least MinSupport units and at least MinLift nats beyond
// chance, then maximal cliques over that graph.
//
// Cliques rather than connected components, for the reason internal/family
// documents at length: co-occurrence is not transitive, and single-linkage
// chains through it until a "concept" spans features with nothing in common.
// Every feature in a clique co-occurs with every other, so the founding member
// set is a set of functions that genuinely do the same several things.
//
// Three bounds, all reported rather than silent. The candidate set is the most
// frequent MaxEmergentFeatures survivors — the corpus's most common shapes,
// which is exactly where a recurring practice would be — one unit contributes
// at most MaxUnitFeatures of them to the co-occurrence count, because that
// count is quadratic in a unit's feature count and a large corpus has units
// with hundreds, and each feature keeps only its EdgeK strongest associations,
// without which the graph is one blob (see buildFeatureGraph).
func emergeConcepts(c *corpus, claimed map[string]bool, stats *Stats, opt Options) ([]Concept, [][]int) {
	cand := candidateFeatures(c, claimed, opt)
	if len(cand) < opt.MinCliqueSize {
		return nil, nil
	}
	index := make(map[string]int, len(cand))
	for i, f := range cand {
		index[f] = i
	}

	g := buildFeatureGraph(c, cand, index, opt)
	stats.Edges = g.edges()
	post := postings(c, cand, index)
	hits := make([]int, c.n) // scratch for foundingMembers, reset by it

	// Enumeration first, across cores, then the selection loop unchanged.
	//
	// The split is forced by a real sequential dependency: duplicate() below
	// tests a candidate member set against the sets already kept *earlier in
	// this loop*, so which of two overlapping cliques survives depends on the
	// order they are visited in. That order is the vocabulary, so it cannot be
	// left to a scheduler. clique.Maximal, on the other hand, is a pure
	// function of the graph and one feature's neighbourhood — g is read-only
	// here and Neighbors sorts its own output — so the expensive half moves out
	// and the deciding half stays exactly where it was.
	//
	// The search is one feature's own neighbourhood, never a connected
	// component. A top-K co-occurrence graph is sparse but overwhelmingly
	// connected — its giant component is most of the vocabulary — so
	// enumerating per component means enumerating over thousands of features,
	// tripping every guard and returning nothing. A neighbourhood is at most
	// EdgeK+1 vertices, so enumeration is bounded by construction rather than
	// by a budget, and the guard below only ever catches a pathology. That
	// bound is also what makes holding every feature's result at once cheap.
	type enumeration struct {
		cliques [][]int
		ok      bool
		skipped bool
	}
	enumerated := make([]enumeration, len(cand))
	parallel.Blocks(len(cand), cliqueBlock, minFeaturesPerCliqueWorker, func(a int) {
		nbrs := g.Neighbors(a)
		if len(nbrs) < opt.MinCliqueSize-1 {
			enumerated[a].skipped = true
			return
		}
		local := clique.Insert(append([]int(nil), nbrs...), a)
		enumerated[a].cliques, enumerated[a].ok = clique.Maximal(g, local, opt.MaxSearch)
	})

	var out []Concept
	var founders [][]int
	var kept [][]int // founding member sets already turned into a concept
	for a := 0; a < len(cand); a++ {
		e := enumerated[a]
		if e.skipped {
			continue
		}
		if !e.ok {
			stats.Skipped++
			continue
		}
		for _, cl := range e.cliques { // size desc: the strongest cluster claims its members first
			// Every maximal clique containing a lies inside a's neighbourhood
			// — an extending vertex would have to be adjacent to a — so each is
			// globally maximal, and each is rediscovered once per member. The
			// lowest-indexed member owns it; without that rule a clique of
			// seven becomes seven identical concepts.
			if len(cl) < opt.MinCliqueSize || cl[0] != a {
				continue
			}
			members := foundingMembers(post, hits, cl)
			if len(members) < opt.MinMembers {
				stats.EmergentDropped++
				continue
			}
			if duplicate(kept, members, opt) {
				stats.EmergentDropped++
				continue
			}
			features, scale, floor := fit(c, members, opt)
			if len(features) == 0 {
				stats.EmergentDropped++
				continue
			}
			kept = append(kept, members)
			out = append(out, Concept{Features: features, Scale: scale, Floor: floor})
			founders = append(founders, members)
			stats.Emergent++
		}
	}
	return out, founders
}

// duplicate reports whether a candidate member set is close enough to one
// already kept to be the same concept said twice.
//
// Overlapping cliques are the norm, not the exception — one function does
// several things, so its features sit in several clusters — and exact-signature
// deduplication misses the case that actually crowds the output: two maximal
// cliques over almost the same functions, differing by a member or two. Jaccard
// at MaxOverlap is the bar. Earlier-kept wins, and the iteration order is fixed
// (feature index ascending, then clique size descending), so which survives is
// deterministic rather than a race between equals.
func duplicate(kept [][]int, members []int, opt Options) bool {
	for _, k := range kept {
		if jaccard(k, members) >= opt.MaxOverlap {
			return true
		}
	}
	return false
}

// jaccard is the overlap of two ascending index slices.
func jaccard(a, b []int) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	i, j, shared := 0, 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			shared++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return float64(shared) / float64(len(a)+len(b)-shared)
}

// buildFeatureGraph is the co-occurrence graph the clusters are found in:
// features that appear together in at least MinSupport units and at least
// MinLift nats beyond chance.
//
// The top-K sparsification is not an optimization, it is what makes the graph
// mean anything. Features co-occur far more freely than functions resemble each
// other — every function in a package shares its package's vocabulary — so the
// unbounded graph is one dense blob: on doppel's own corpus it produced a
// single component of thousands of features that tripped MaxComponent and
// yielded no concepts at all. Keeping each feature's EdgeK strongest partners
// is the same bounded-top-K idiom the retrieval channels use, and for the same
// reason: recall is per-item, not global.
func buildFeatureGraph(c *corpus, cand []string, index map[string]int, opt Options) *featureGraph {
	co := coOccurrence(c, cand, index, opt)
	n := float64(c.n)

	// One PMI pass, accumulated per feature so each can keep its strongest
	// partners. Ascending pair order throughout: no map decides anything.
	type link struct {
		to  int
		pmi float64
	}
	links := make([][]link, len(cand))
	for _, k := range sortedPairKeys(co) {
		count := co[k]
		if count < opt.MinSupport {
			continue
		}
		dfA, dfB := c.df[cand[k.a]], c.df[cand[k.b]]
		pmi := math.Log(n * float64(count) / (float64(dfA) * float64(dfB)))
		if pmi < opt.MinLift {
			continue
		}
		links[k.a] = append(links[k.a], link{to: k.b, pmi: pmi})
		links[k.b] = append(links[k.b], link{to: k.a, pmi: pmi})
	}

	g := newFeatureGraph(len(cand))
	for a := range links {
		l := links[a]
		sort.SliceStable(l, func(i, j int) bool {
			if l[i].pmi != l[j].pmi {
				return l[i].pmi > l[j].pmi
			}
			return l[i].to < l[j].to // ties on the lower index, as everywhere
		})
		if opt.EdgeK > 0 && len(l) > opt.EdgeK {
			l = l[:opt.EdgeK]
		}
		for _, e := range l {
			g.add(a, e.to)
		}
	}
	return g
}

// candidateFeatures returns the unclaimed survivors that may seed a cluster —
// see seedChannels for why that is not every survivor — most frequent first and
// bounded by MaxEmergentFeatures, then sorted by name so the returned index
// space does not depend on the frequency tie order.
func candidateFeatures(c *corpus, claimed map[string]bool, opt Options) []string {
	var cand []string
	for _, f := range c.surviving {
		if !claimed[f] && canSeed(f) {
			cand = append(cand, f)
		}
	}
	if opt.MaxEmergentFeatures > 0 && len(cand) > opt.MaxEmergentFeatures {
		sort.SliceStable(cand, func(i, j int) bool {
			if c.df[cand[i]] != c.df[cand[j]] {
				return c.df[cand[i]] > c.df[cand[j]]
			}
			return cand[i] < cand[j]
		})
		cand = cand[:opt.MaxEmergentFeatures]
	}
	sort.Strings(cand)
	return cand
}

// coOccurrence counts how many units carry each candidate feature pair. A unit
// contributes only its MaxUnitFeatures most informative candidates, so one
// enormous function cannot dominate the graph or the runtime.
func coOccurrence(c *corpus, cand []string, index map[string]int, opt Options) map[pair]int {
	co := make(map[pair]int)
	for i := 0; i < c.n; i++ {
		var have []int
		for _, f := range c.features[i] {
			if j, ok := index[f]; ok {
				have = append(have, j)
			}
		}
		if opt.MaxUnitFeatures > 0 && len(have) > opt.MaxUnitFeatures {
			sort.SliceStable(have, func(x, y int) bool {
				ix, iy := c.idf[cand[have[x]]], c.idf[cand[have[y]]]
				if ix != iy {
					return ix > iy
				}
				return have[x] < have[y]
			})
			have = have[:opt.MaxUnitFeatures]
			sort.Ints(have)
		}
		for x := 0; x < len(have); x++ {
			for y := x + 1; y < len(have); y++ {
				co[pair{have[x], have[y]}]++
			}
		}
	}
	return co
}

// postings is the inverted index over candidate features: which units carry
// each, ascending. Built once, because foundingMembers is called for every
// clique and scanning every unit's whole feature set per clique is what made
// the emergent pass quadratic in practice.
func postings(c *corpus, cand []string, index map[string]int) [][]int {
	post := make([][]int, len(cand))
	for i := 0; i < c.n; i++ {
		for _, f := range c.features[i] {
			if j, ok := index[f]; ok {
				post[j] = append(post[j], i) // ascending: i increases
			}
		}
	}
	return post
}

// foundingMembers are the units carrying at least two of the clique's
// features. Two rather than all: a clique states that its features co-occur
// beyond chance, not that they are inseparable, and requiring the full set
// would shrink most clusters to the handful of units that happen to do
// everything. For a two-feature clique — the common case — two means both.
//
// hits is caller-owned scratch of length n, left zeroed on return.
func foundingMembers(post [][]int, hits []int, cl []int) []int {
	var touched []int
	for _, f := range cl {
		for _, u := range post[f] {
			if hits[u] == 0 {
				touched = append(touched, u)
			}
			hits[u]++
		}
	}
	sort.Ints(touched)
	var members []int
	for _, u := range touched {
		if hits[u] >= 2 {
			members = append(members, u)
		}
		hits[u] = 0
	}
	return members
}

// pair is one unordered feature-index pair, low index first.
type pair struct{ a, b int }

func sortedPairKeys(m map[pair]int) []pair {
	out := make([]pair, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].a != out[j].a {
			return out[i].a < out[j].a
		}
		return out[i].b < out[j].b
	})
	return out
}

// featureGraph is the undirected co-occurrence graph over candidate feature
// indices, presenting itself as a clique.Graph.
type featureGraph struct {
	adj []map[int]bool
}

func newFeatureGraph(n int) *featureGraph {
	return &featureGraph{adj: make([]map[int]bool, n)}
}

func (g *featureGraph) add(a, b int) {
	if a == b {
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
}

// Has reports adjacency. Part of clique.Graph.
func (g *featureGraph) Has(a, b int) bool {
	if a < 0 || a >= len(g.adj) || g.adj[a] == nil {
		return false
	}
	return g.adj[a][b]
}

// Neighbors returns a's neighbours ascending. Part of clique.Graph.
func (g *featureGraph) Neighbors(a int) []int {
	if a < 0 || a >= len(g.adj) || g.adj[a] == nil {
		return nil
	}
	out := make([]int, 0, len(g.adj[a]))
	for b := range g.adj[a] {
		out = append(out, b)
	}
	sort.Ints(out)
	return out
}

// edges counts the undirected edges, for the diagnostic line.
func (g *featureGraph) edges() int {
	n := 0
	for _, a := range g.adj {
		n += len(a)
	}
	return n / 2
}

// One feature's neighbourhood is at most EdgeK+1 vertices, so a single
// enumeration is small and a block amortises the atomic over many of them.
const (
	cliqueBlock                = 16
	minFeaturesPerCliqueWorker = 64
)

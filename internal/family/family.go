// Package family finds near-duplicate families — groups of three or more
// functions that are all alike, not merely chained together by a sequence of
// pairwise resemblances.
//
// Everything upstream of the report is already n-ary: tag frequencies,
// information content, the culture model, role thresholds, and the retrieval
// channels, whose posting lists are sets of arbitrary size. Only the 12-signal
// comparison and the report are pairwise, and neither is pairwise for a
// representational reason — the metrics simply have no canonical extension to
// k items. So this package does not generalize any score. It keeps pairwise
// scoring, whose quadratic cost the pipeline has already paid, and clusters
// afterward on the surviving pair graph.
//
// Two design constraints make that honest rather than a staircase failure.
//
// Non-transitivity. A~B at 0.7 and B~C at 0.7 says nothing whatever about
// A~C. Single-linkage clustering chains through those links until a "family"
// spans functions with nothing in common — the classic clone-detection
// failure. Families here are maximal cliques, so every member is similar to
// every other member and the report can state a guarantee a reader can
// falsify by opening two files.
//
// Retrieval gaps. Each function keeps a bounded top-K per channel, so an edge
// can be missing because it fell out of a budget rather than because the two
// functions are unalike. Enumerating cliques on the retrieved graph alone
// would read that gap as "not a family" and split real families
// systematically. Build therefore completes each component's missing edges
// before enumerating — see completeComponent.
package family

import (
	"sort"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

// Options are the family stage's knobs.
type Options struct {
	// Min is the edge cut on code-shape. An edge exists between two functions
	// when they are at least this alike, which is what lets a family state one
	// checkable sentence: every member is >= Min to every other member.
	Min float64

	// MinSize is the smallest family reported. Three, because a 2-clique is a
	// pair and the pair list already shows those.
	MinSize int

	// MaxComponent bounds edge completion and clique enumeration. Completion
	// is O(m^2) in a component's size and Bron-Kerbosch is exponential in the
	// worst case; a component past this is abandoned into Stats.Skipped rather
	// than run for minutes on a bucket of generated code.
	//
	// 128 is measured, not guessed: at 64 the two largest public corpora each
	// skipped two or three components; at 128 neither skips any, for about half
	// a second on an 8k-function corpus. MaxSearch is the guard that actually
	// matters, since density rather than size is what makes enumeration
	// expensive.
	MaxComponent int

	// MaxSearch bounds Bron-Kerbosch recursion per component, for the dense
	// components that are small enough to complete but still pathological.
	MaxSearch int
}

// DefaultOptions are the shipped settings.
func DefaultOptions() Options {
	return Options{Min: 0.60, MinSize: 3, MaxComponent: 128, MaxSearch: 200000}
}

// Family is one group of mutually similar functions.
//
// Members are unit indices, the positional index space the whole pipeline uses
// — docs[i] describes units[i], and SimilarPair carries AIdx/BIdx into it.
// Resolving families by name instead has already caused one silent-miss bug in
// this codebase.
type Family struct {
	Members []int // ascending

	// MinEdge is the guarantee: the weakest pairwise code-shape inside the
	// family. It is the number the report should state, because it is the one
	// every member satisfies.
	MinEdge  float64
	MeanEdge float64

	// Completed counts this family's edges that retrieval never proposed and
	// this package scored directly. Worth surfacing: without it a reader
	// checking a family against the pair list finds members with no pair
	// between them and concludes the family is invented.
	Completed int

	// Kind labels a family every member pair of which satisfies one of the
	// pair kinds — interface implementations of one method, or diverged
	// copies sharing one stem — so the census can say why a family exists.
	// Nil when unlabeled. See analyzer.ClassifyFamily.
	Kind *analyzer.KindNote

	// Evidence is the family's total retrieval evidence mass in nats —
	// Σ Retrieval.Total over the clique edges retrieval proposed (completed
	// edges contribute zero). It is what ranks the census: a 44-member family
	// of mutex-guarded getters is factually a family, but its members share
	// corpus idiom, its edges carry little informative mass, and most of them
	// exist only because completion stitched them; a doc-generator family's
	// few edges each carry hundreds of nats. Ranking by size instead put the
	// idiom families first on every large corpus.
	Evidence float64
}

// Stats is the run's family accounting, for the stderr summary.
type Stats struct {
	Components int   // connected components of the pair graph at this cut
	Families   int   // maximal cliques reported
	Members    int   // distinct functions in at least one family
	Completed  int   // edges scored here that retrieval never proposed
	Skipped    []int // sizes of components abandoned by a guard, ascending
}

// Build finds the families in a scored pair set.
//
// pairs is the full comparator-scored, filtered candidate list — never the
// ranked one. The --max-per-func cap is a report-time device applied after
// scoring, so the pair list a reader sees may show two of a function's edges
// while the family stage legitimately sees eight.
//
// The input is never mutated: analyzer.SortForReport sorts its argument in
// place, and this must produce the same answer whether or not that has run.
func Build(units []parser.CodeUnit, pairs []analyzer.SimilarPair, o Options) ([]Family, Stats) {
	if o.MinSize < 2 {
		o.MinSize = 2
	}
	g := newGraph(len(units))
	for _, p := range pairs {
		if p.Score >= o.Min {
			ev := 0.0
			if p.Retrieval != nil {
				ev = p.Retrieval.Total
			}
			g.add(p.AIdx, p.BIdx, p.Score, ev)
		}
	}

	var (
		out   []Family
		stats Stats
	)
	seen := make(map[int]bool)
	for _, comp := range g.components() {
		stats.Components++
		if len(comp) > o.MaxComponent {
			stats.Skipped = append(stats.Skipped, len(comp))
			continue
		}
		stats.Completed += completeComponent(units, g, comp, o.Min)

		cliques, ok := g.maximalCliques(comp, o.MaxSearch)
		if !ok {
			stats.Skipped = append(stats.Skipped, len(comp))
			continue
		}
		for _, c := range cliques {
			if len(c) < o.MinSize {
				continue
			}
			f := g.describe(c)
			out = append(out, f)
			for _, m := range c {
				seen[m] = true
			}
		}
	}

	sortFamilies(out)
	for i := range out {
		out[i].Kind = analyzer.ClassifyFamily(units, out[i].Members, out[i].MinEdge)
	}
	stats.Families = len(out)
	stats.Members = len(seen)
	sort.Ints(stats.Skipped)
	return out, stats
}

// completeComponent scores every member pair the retriever never proposed and
// adds the ones that clear the cut, returning how many it added.
//
// This is the step that makes clique enumeration meaningful. Retrieval keeps a
// bounded top-K per function per channel, so within a family of six the two
// weakest members may never have met — not because they are unalike but
// because each filled the other's budget with someone else. Without this, the
// six-member family enumerates as two overlapping fours.
//
// It is affordable because fingerprints are built during parsing and kept on
// the unit: this is arithmetic over sorted slices, not re-parsing. The zero
// fingerprint (a declaration with no body) already scores zero against
// everything, so body-less units cannot be dragged in.
func completeComponent(units []parser.CodeUnit, g *graph, comp []int, min float64) int {
	added := 0
	for i := 0; i < len(comp); i++ {
		for j := i + 1; j < len(comp); j++ {
			a, b := comp[i], comp[j]
			if g.has(a, b) {
				continue
			}
			s := fingerprint.Similarity(units[a].Fingerprint, units[b].Fingerprint).Score
			if s >= min {
				g.add(a, b, s, 0) // completion carries no retrieval evidence
				g.markCompleted(a, b)
				added++
			}
		}
	}
	return added
}

// sortFamilies puts the census in reading order: most informative first —
// total retrieval evidence mass, the same currency the pair list ranks by —
// then biggest, then tightest. Size ordering alone put the corpus's idiom
// families (44 mutex-guarded getters) above its genuine clone families on
// every large corpus; see Family.Evidence.
//
// The trailing member comparison makes the order total, which the repo's
// byte-identical-output invariant requires — two families of equal size and
// equal edge weights are otherwise free to swap between runs.
func sortFamilies(f []Family) {
	sort.SliceStable(f, func(i, j int) bool {
		a, b := f[i], f[j]
		if a.Evidence != b.Evidence {
			return a.Evidence > b.Evidence
		}
		if len(a.Members) != len(b.Members) {
			return len(a.Members) > len(b.Members)
		}
		if a.MinEdge != b.MinEdge {
			return a.MinEdge > b.MinEdge
		}
		if a.MeanEdge != b.MeanEdge {
			return a.MeanEdge > b.MeanEdge
		}
		for k := 0; k < len(a.Members) && k < len(b.Members); k++ {
			if a.Members[k] != b.Members[k] {
				return a.Members[k] < b.Members[k]
			}
		}
		return false
	})
}

package main

// The paired and paired-clustered bootstraps examples/outcome-reanalysis.md
// pre-registered: a sensitivity analysis of the two outcome studies' primary
// comparison, behind compare-summary's -bootstrap flag. The default
// (independent) is diffCI, unchanged.

import (
	"fmt"
	"io"
	"math"
	"sort"
)

const (
	bootIndependent     = "independent"
	bootPaired          = "paired"
	bootPairedClustered = "paired-clustered"
)

// pairedUnion is the comparison's resampling frame: the distinct pairs of two
// lists, how often each list holds each, and the clusters over them.
type pairedUnion struct {
	pairs    []int // pair indexes into compareOut.Pairs, ascending
	a, b     []int // multiplicity of pairs[i] in each list
	clusters [][]int
	strata   [3][]int // cluster indexes: A only, B only, both
}

// unionOf builds the frame. clustered joins pairs whose recorded evidence
// shares a commit; otherwise every pair is its own cluster.
func unionOf(c compareOut, la, lb []int, clustered bool) pairedUnion {
	ma, mb := map[int]int{}, map[int]int{}
	for _, x := range la {
		ma[x]++
	}
	for _, x := range lb {
		mb[x]++
	}
	var u pairedUnion
	for x := range ma {
		u.pairs = append(u.pairs, x)
	}
	for x := range mb {
		if ma[x] == 0 {
			u.pairs = append(u.pairs, x)
		}
	}
	sort.Ints(u.pairs)
	u.a = make([]int, len(u.pairs))
	u.b = make([]int, len(u.pairs))
	for i, x := range u.pairs {
		u.a[i], u.b[i] = ma[x], mb[x]
	}

	parent := make([]int, len(u.pairs))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	if clustered {
		first := map[string]int{}
		for i, x := range u.pairs {
			for _, e := range c.Pairs[x].Evidence {
				for _, sha := range e.Commits {
					j, ok := first[sha]
					if !ok {
						first[sha] = i
						continue
					}
					ri, rj := find(i), find(j)
					if ri != rj {
						parent[max(ri, rj)] = min(ri, rj)
					}
				}
			}
		}
	}
	// Roots are the smallest member, so clusters come out ordered by their
	// smallest pair index.
	at := map[int]int{}
	for i := range u.pairs {
		r := find(i)
		k, ok := at[r]
		if !ok {
			k = len(u.clusters)
			at[r] = k
			u.clusters = append(u.clusters, nil)
		}
		u.clusters[k] = append(u.clusters[k], i)
	}
	for k, cl := range u.clusters {
		inA, inB := false, false
		for _, i := range cl {
			inA = inA || u.a[i] > 0
			inB = inB || u.b[i] > 0
		}
		s := 2
		switch {
		case !inB:
			s = 0
		case !inA:
			s = 1
		}
		u.strata[s] = append(u.strata[s], k)
	}
	return u
}

// pairedDiffCI is M1(a) - M1(b) with the pre-registered stratified cluster
// bootstrap. The point estimate is diffCI's.
func pairedDiffCI(c compareOut, la, lb []int, scheme, seed string) (d, lo, hi float64) {
	if len(la) == 0 || len(lb) == 0 {
		return math.NaN(), math.NaN(), math.NaN()
	}
	u := unionOf(c, la, lb, scheme == bootPairedClustered)
	// Per cluster: the sums of m1 and of entries, each side.
	n := len(u.clusters)
	sa, na, sb, nb := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
	for k, cl := range u.clusters {
		for _, i := range cl {
			v := m1(c.Pairs[u.pairs[i]])
			sa[k] += float64(u.a[i]) * v
			na[k] += float64(u.a[i])
			sb[k] += float64(u.b[i]) * v
			nb[k] += float64(u.b[i])
		}
	}
	diff := func(draw func(int) int) float64 {
		ta, ca, tb, cb := 0.0, 0.0, 0.0, 0.0
		for _, st := range u.strata {
			for range st {
				k := st[draw(len(st))]
				ta, ca, tb, cb = ta+sa[k], ca+na[k], tb+sb[k], cb+nb[k]
			}
		}
		return ta/ca - tb/cb
	}
	rng := lcg{seedOf("outcome-reanalysis bootstrap", scheme, c.Corpus, seed)}
	reps := make([]float64, 0, bootReps)
	for range bootReps {
		reps = append(reps, diff(rng.intn))
	}
	lo, hi = percentile95(reps)
	return m1Mean(c, la) - m1Mean(c, lb), lo, hi
}

// doppelTop is doppel's top primaryK, nil when the corpus has no doppel list.
func doppelTop(c compareOut) []int {
	for _, v := range viewsOf(c) {
		if v.name == doppelMethod {
			return v.slice(1, primaryK)
		}
	}
	return nil
}

// writeClusterTable is the descriptive frame per comparison: union, shared
// pairs, clusters and the largest one.
func writeClusterTable(w io.Writer, corpora []compareOut, scheme string) {
	fmt.Fprintf(w, "\n### Resampling frame at %d (%s)\n\n", primaryK, scheme)
	fmt.Fprintf(w, "| corpus | baseline | union | shared | clusters | largest cluster |\n")
	fmt.Fprintf(w, "| --- | --- | ---: | ---: | ---: | ---: |\n")
	for _, c := range corpora {
		dop := doppelTop(c)
		for _, v := range viewsOf(c) {
			if v.name == doppelMethod {
				continue
			}
			u := unionOf(c, dop, v.slice(1, primaryK), scheme == bootPairedClustered)
			shared, largest := 0, 0
			for i := range u.pairs {
				if u.a[i] > 0 && u.b[i] > 0 {
					shared++
				}
			}
			for _, cl := range u.clusters {
				largest = max(largest, len(cl))
			}
			fmt.Fprintf(w, "| %s | %s | %d | %d | %d | %d |\n", c.Corpus, v.name, len(u.pairs), shared, len(u.clusters), largest)
		}
	}
}

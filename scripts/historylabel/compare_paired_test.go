package main

import (
	"fmt"
	"testing"
)

// fixturePairs builds a compareOut whose pair i co-changed once in commit
// commits[i] (no event when commits[i] is ""), with one edit a side.
func fixturePairs(commits []string) compareOut {
	c := compareOut{Corpus: "fixture"}
	for i, sha := range commits {
		p := studyPair{A: fmt.Sprintf("a%d", i), B: fmt.Sprintf("b%d", i), EditsA: 1, EditsB: 1}
		if sha != "" {
			p.Cochanges = 1
			p.Evidence = []evidence{{Kind: "co-change", Commits: []string{sha}}}
		}
		c.Pairs = append(c.Pairs, p)
	}
	return c
}

func span(lo, hi int) []int {
	var out []int
	for i := lo; i < hi; i++ {
		out = append(out, i)
	}
	return out
}

func TestPairedIdenticalListsGiveZero(t *testing.T) {
	commits := make([]string, 30)
	for i := range commits {
		if i%3 == 0 {
			commits[i] = fmt.Sprintf("c%d", i%2) // two commits, shared by many pairs
		}
	}
	c := fixturePairs(commits)
	l := span(0, 30)
	for _, scheme := range []string{bootPaired, bootPairedClustered} {
		d, lo, hi := pairedDiffCI(c, l, l, scheme, "x")
		if d != 0 || lo != 0 || hi != 0 {
			t.Errorf("%s: identical lists read %v [%v, %v], want exactly 0 [0, 0]", scheme, d, lo, hi)
		}
	}
}

func TestPairedIsDeterministic(t *testing.T) {
	commits := make([]string, 60)
	for i := range commits {
		if i%4 == 0 {
			commits[i] = fmt.Sprintf("c%d", i%3)
		}
	}
	c := fixturePairs(commits)
	a, b := span(0, 40), span(20, 60)
	for _, scheme := range []string{bootPaired, bootPairedClustered} {
		d1, lo1, hi1 := pairedDiffCI(c, a, b, scheme, "x")
		d2, lo2, hi2 := pairedDiffCI(c, a, b, scheme, "x")
		if d1 != d2 || lo1 != lo2 || hi1 != hi2 {
			t.Errorf("%s: two runs differ: %v [%v, %v] against %v [%v, %v]", scheme, d1, lo1, hi1, d2, lo2, hi2)
		}
		if want := m1Mean(c, a) - m1Mean(c, b); d1 != want {
			t.Errorf("%s: point estimate %v, want the independent one %v", scheme, d1, want)
		}
	}
}

// With no commit shared between pairs, clustering joins nothing: the
// clustered frame is the paired one, cluster for cluster and stratum for
// stratum. (The two schemes still seed differently.)
func TestClusteredWithoutSharedCommitsIsPaired(t *testing.T) {
	commits := make([]string, 40)
	for i := range commits {
		if i%5 == 0 {
			commits[i] = fmt.Sprintf("c%d", i)
		}
	}
	c := fixturePairs(commits)
	a, b := span(0, 25), span(15, 40)
	p, q := unionOf(c, a, b, false), unionOf(c, a, b, true)
	if fmt.Sprint(p) != fmt.Sprint(q) {
		t.Errorf("frames differ:\npaired    %v\nclustered %v", p, q)
	}
	if got := [3]int{len(p.strata[0]), len(p.strata[1]), len(p.strata[2])}; got != [3]int{15, 15, 10} {
		t.Errorf("strata sizes %v, want [15 15 10]", got)
	}
}

// One commit co-changes six of A's pairs and drives the whole difference.
// Paired treats them as six observations; clustered as one, so its interval
// must be wider.
func TestClusteredWidensWhenOneCommitDrivesTheDifference(t *testing.T) {
	commits := make([]string, 40)
	for i := 0; i < 6; i++ {
		commits[i] = "sweep"
	}
	commits[30] = "lone" // one event on B's side, its own commit
	c := fixturePairs(commits)
	a, b := span(0, 20), span(20, 40)
	d, plo, phi := pairedDiffCI(c, a, b, bootPaired, "x")
	_, clo, chi := pairedDiffCI(c, a, b, bootPairedClustered, "x")
	if d <= 0 {
		t.Fatalf("fixture: difference %v, want positive", d)
	}
	if u := unionOf(c, a, b, true); len(u.clusters) != 35 {
		t.Fatalf("clusters: %d, want 35 (the six sweep pairs as one)", len(u.clusters))
	}
	if chi-clo <= phi-plo {
		t.Errorf("clustered [%v, %v] is not wider than paired [%v, %v]", clo, chi, plo, phi)
	}
}

// Shared pairs cancel under paired: two lists differing by one pair each
// read a narrower interval than the independent bootstrap gives them.
func TestPairedNarrowsWhenListsShare(t *testing.T) {
	commits := make([]string, 42)
	for i := 0; i < 40; i += 2 {
		commits[i] = fmt.Sprintf("c%d", i)
	}
	commits[40] = "x40"
	c := fixturePairs(commits)
	a := append(span(0, 40), 40)
	b := append(span(0, 40), 41)
	_, ilo, ihi := diffCI(c, a, b, "x")
	_, plo, phi := pairedDiffCI(c, a, b, bootPaired, "x")
	if phi-plo >= ihi-ilo {
		t.Errorf("paired [%v, %v] is not narrower than independent [%v, %v]", plo, phi, ilo, ihi)
	}
}

package bench

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// signalNames are comparator.SignalVector's twelve slots, abbreviated for a
// table row, in the same order.
var signalNames = [comparator.SignalCount]string{
	"calls", "exhib", "calby", "role", "pkg", "fromC", "intoC", "vis", "recv", "nbhd", "fromP", "intoP",
}

// overlapVariant is one change to the rank key's overlap half: either a
// relation reweighting (the comparator re-scores every pair) or a transform
// applied to each pair before ranking.
type overlapVariant struct {
	name    string
	weights map[ontology.TermID]float64 // nil = production weights
	pair    func(p analyzer.SimilarPair) analyzer.SimilarPair
	rank    func(*analyzer.RankOptions) // nil = DefaultRankOptions
}

func zeroed(rels ...ontology.TermID) map[ontology.TermID]float64 {
	m := map[ontology.TermID]float64{}
	for _, r := range rels {
		m[r] = 0
	}
	return m
}

// withOverlap returns p with its evidence copied and OverlapScore replaced, so
// a transform never writes through the run's shared evidence pointer.
func withOverlap(p analyzer.SimilarPair, o float64) analyzer.SimilarPair {
	if p.Evidence == nil {
		return p
	}
	ev := *p.Evidence
	ev.OverlapScore = o
	p.Evidence = &ev
	return p
}

var overlapVariants = []overlapVariant{
	{name: "production"},
	{name: "no bound_to", weights: zeroed(ontology.RelBoundTo)},
	{name: "no declared_in", weights: zeroed(ontology.RelDeclaredIn)},
	{name: "no locality (recv+pkg+vis)", weights: zeroed(ontology.RelBoundTo, ontology.RelDeclaredIn, ontology.RelHasVisibility)},
	{name: "no calls", weights: zeroed(ontology.RelCalls)},
	{name: "no called_by", weights: zeroed(ontology.RelCalledBy)},
	{name: "no neighborhood", weights: zeroed(ontology.RelSharesNeighborhood)},
	{name: "no exhibits", weights: zeroed(ontology.RelExhibits)},
	{name: "key without overlap", pair: func(p analyzer.SimilarPair) analyzer.SimilarPair { return withOverlap(p, 1) }},
	{name: "key overlap^0.5", pair: func(p analyzer.SimilarPair) analyzer.SimilarPair {
		if p.Evidence == nil {
			return p
		}
		return withOverlap(p, math.Sqrt(p.Evidence.OverlapScore))
	}},
	{name: "key shape^1 (pre-ShapePower)", rank: func(o *analyzer.RankOptions) { o.ShapePower = 1 }},
	{name: "key shape^3", rank: func(o *analyzer.RankOptions) { o.ShapePower = 3 }},
}

// TestOverlapRank asks where the labeled false positives get their rank. The
// first half is a diagnosis: every labeled pair the run retrieved, with each
// factor of analyzer.RankKey (retrieval Total, OverlapScore, code-shape,
// trophic) and the twelve overlap signals behind OverlapScore, then the mean
// of each per label class. The second half ranks the labels under variants
// that remove one overlap signal or reshape the key — ShapePower included, so
// the measurement that set it stays re-runnable — and logs the scorecard
// and the labels that moved. Retrieval is untouched throughout. Asserts
// nothing.
//
//	DOPPEL_BENCH_OVERLAPRANK=1 go test ./internal/bench/ -v -run TestOverlapRank
//
// Labeled corpora come from labeledTargets: committed reviews plus private ones
// named by DOPPEL_BENCH_LENSES_LABELS and DOPPEL_BENCH_LENSES_EXTRA.
func TestOverlapRank(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_OVERLAPRANK") != "1" {
		t.Skip("set DOPPEL_BENCH_OVERLAPRANK=1 to measure the overlap half of the rank key")
	}
	targets := labeledTargets(t)

	type pooled struct {
		violations        int
		merge, ref, fp    float64
		nMerge, nRef, nFP int
	}
	pool := make([]pooled, len(overlapVariants))

	for _, tg := range targets {
		units, err := Load(tg.root, Population(tg.lf.Population))
		if err != nil || len(units) == 0 {
			t.Logf("[%s] load: %v", tg.name, err)
			continue
		}
		run := Analyze(units, retriever.DefaultOptions())
		base := run.Onto
		diagnose(t, tg.name, run, tg.lf)

		var prod Scorecard
		for vi, v := range overlapVariants {
			if v.weights != nil {
				onto, err := ontology.WithWeightsOver(base, v.weights)
				if err != nil {
					t.Fatalf("%s: %v", v.name, err)
				}
				run.Rescore(onto)
			}
			r := *run
			if v.pair != nil {
				r.Pairs = make([]analyzer.SimilarPair, len(run.Pairs))
				for i, p := range run.Pairs {
					r.Pairs[i] = v.pair(p)
				}
			}
			ro := analyzer.DefaultRankOptions()
			if v.rank != nil {
				v.rank(&ro)
			}
			sc := ScoreWith(&r, tg.lf, ro)
			if v.weights != nil {
				run.Rescore(base)
			}
			t.Logf("[%s] %-28s %s", tg.name, v.name, scLine(sc))
			pl := &pool[vi]
			pl.violations += violations(sc)
			if sc.Present["merge"] > 0 {
				pl.merge += sc.MeanRank["merge"]
				pl.nMerge++
			}
			if sc.Present["refactor"] > 0 {
				pl.ref += sc.MeanRank["refactor"]
				pl.nRef++
			}
			if sc.Present["false_positive"] > 0 {
				pl.fp += sc.MeanRank["false_positive"]
				pl.nFP++
			}
			if vi == 0 {
				prod = sc
				continue
			}
			for i, res := range sc.Results {
				was := prod.Results[i]
				if res.Rank == was.Rank || (res.Rank > 20 && was.Rank > 20) {
					continue // only movement into, out of or within the top 20
				}
				t.Logf("      %-14s %4d -> %-4d %s <-> %s", res.Label.Class, was.Rank, res.Rank, res.Label.A, res.Label.B)
			}
		}
	}

	t.Logf("pooled over %d labeled corpora (per-corpus class means, averaged; lower merge and higher fp are better):", len(targets))
	avg := func(s float64, n int) string {
		if n == 0 {
			return "-"
		}
		return fmt.Sprintf("%.1f", s/float64(n))
	}
	for vi, v := range overlapVariants {
		pl := pool[vi]
		t.Logf("  %-28s merge %5s  refactor %5s  fp %6s  violations %d",
			v.name, avg(pl.merge, pl.nMerge), avg(pl.ref, pl.nRef), avg(pl.fp, pl.nFP), pl.violations)
	}
}

// diagnose logs the rank-key factors and overlap signals of every labeled
// pair the run retrieved, and their means per label class.
func diagnose(t *testing.T, name string, run *Run, lf LabelsFile) {
	t.Helper()
	byKey := map[string]int{}
	for i, p := range run.Pairs {
		k := pairKey(qualifiedName(run.Units[p.AIdx]), qualifiedName(run.Units[p.BIdx]))
		if _, ok := byKey[k]; !ok {
			byKey[k] = i
		}
	}
	type acc struct {
		n                              float64
		logTotal, overlap, shape, trop float64
		sig                            [comparator.SignalCount]float64
	}
	sums := map[string]*acc{}
	header := "  class          lnTot  ovl  shape troph |"
	for _, s := range signalNames {
		header += fmt.Sprintf(" %5s", s)
	}
	t.Logf("[%s] labeled pairs, rank-key factors and overlap signals", name)
	t.Log(header)
	for _, l := range lf.Labels {
		i, ok := byKey[pairKey(l.A, l.B)]
		if !ok {
			continue
		}
		p := run.Pairs[i]
		if p.Retrieval == nil || p.Evidence == nil {
			continue
		}
		sig := comparator.SignalVector(*p.Evidence, run.Docs[p.AIdx], run.Docs[p.BIdx])
		var b strings.Builder
		lt := math.Log(math.Max(p.Retrieval.Total, 1e-9))
		fmt.Fprintf(&b, "  %-14s %5.1f %4.2f  %4.2f  %4.2f |", l.Class, lt, p.Evidence.OverlapScore, p.Score, p.Retrieval.TrophicSim)
		for _, v := range sig {
			fmt.Fprintf(&b, " %5.2f", v)
		}
		fmt.Fprintf(&b, "  %s <-> %s", l.A, l.B)
		t.Log(b.String())
		a := sums[l.Class]
		if a == nil {
			a = &acc{}
			sums[l.Class] = a
		}
		a.n++
		a.logTotal += lt
		a.overlap += p.Evidence.OverlapScore
		a.shape += p.Score
		a.trop += p.Retrieval.TrophicSim
		for j, v := range sig {
			a.sig[j] += v
		}
	}
	for _, c := range []string{"merge", "refactor", "false_positive"} {
		a := sums[c]
		if a == nil {
			continue
		}
		var b strings.Builder
		fmt.Fprintf(&b, "  mean %-9s %5.1f %4.2f  %4.2f  %4.2f |", c, a.logTotal/a.n, a.overlap/a.n, a.shape/a.n, a.trop/a.n)
		for _, v := range a.sig {
			fmt.Fprintf(&b, " %5.2f", v/a.n)
		}
		fmt.Fprintf(&b, "  n=%.0f", a.n)
		t.Log(b.String())
	}
}

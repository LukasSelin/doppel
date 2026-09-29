package bench

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// lensVariant rewrites the wl component of the code-shape score from a pair's
// per-lens Jaccards. Everything else — retrieval, the other three components,
// overlap, trophic — is the production run, so a variant moves exactly one
// number per pair and a scorecard change is attributable to it.
type lensVariant struct {
	name string
	wl   func(p analyzer.SimilarPair, prof fingerprint.LensProfile) float64
}

var lensVariants = []lensVariant{
	{"shape (production)", func(p analyzer.SimilarPair, _ fingerprint.LensProfile) float64 { return p.Breakdown.WL }},
	{"skeleton", func(_ analyzer.SimilarPair, pr fingerprint.LensProfile) float64 {
		return pr[fingerprint.LensSkeleton].Jaccard
	}},
	{"ordered", func(_ analyzer.SimilarPair, pr fingerprint.LensProfile) float64 {
		return pr[fingerprint.LensOrdered].Jaccard
	}},
	{"mean(shape, skeleton)", func(p analyzer.SimilarPair, pr fingerprint.LensProfile) float64 {
		return (p.Breakdown.WL + pr[fingerprint.LensSkeleton].Jaccard) / 2
	}},
	{"max(shape, skeleton)", func(p analyzer.SimilarPair, pr fingerprint.LensProfile) float64 {
		return math.Max(p.Breakdown.WL, pr[fingerprint.LensSkeleton].Jaccard)
	}},
	{"skeleton, ordered-gated", func(_ analyzer.SimilarPair, pr fingerprint.LensProfile) float64 {
		// Skeleton credit, discounted by how far the ordered lens falls
		// below the shape lens: a mirror pair keeps its structure but pays
		// for its reversed data flow.
		gap := pr[fingerprint.LensShape].Jaccard - pr[fingerprint.LensOrdered].Jaccard
		return math.Max(0, pr[fingerprint.LensSkeleton].Jaccard-gap)
	}},
}

// TestLensRank asks whether reading the code-shape score's wl component
// through a different fingerprint.Lens ranks labeled pairs better than the
// production shape lens. Retrieval is untouched, so a pair the production run
// never retrieved stays absent under every variant — this measures ranking,
// not recall. Per labeled corpus and variant it logs the scorecard and the
// labels whose rank moved; a pooled line per variant sums violations and
// averages the per-corpus class means. Asserts nothing.
//
//	DOPPEL_BENCH_LENSRANK=1 go test ./internal/bench/ -v -run TestLensRank
//
// Committed labels are scored against the fetched ladder. Private labels are
// read from DOPPEL_BENCH_LENSES_LABELS, a directory holding
// <name>.labels.json, matched by base name against the roots in
// DOPPEL_BENCH_LENSES_EXTRA. Nothing about a private corpus is committed.
func TestLensRank(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_LENSRANK") != "1" {
		t.Skip("set DOPPEL_BENCH_LENSRANK=1 to rank the labels under each lens")
	}
	type target struct {
		name, root string
		lf         LabelsFile
	}
	var targets []target
	committed := committedLabels(t)
	for _, c := range Corpora {
		lf, ok := committed[c.Name]
		if !ok || !Present(c) {
			continue
		}
		root, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		targets = append(targets, target{c.Name, root, lf})
	}
	if dir := os.Getenv("DOPPEL_BENCH_LENSES_LABELS"); dir != "" {
		for _, root := range filepath.SplitList(os.Getenv("DOPPEL_BENCH_LENSES_EXTRA")) {
			name := filepath.Base(root)
			data, err := os.ReadFile(filepath.Join(dir, name+".labels.json"))
			if err != nil {
				continue
			}
			lf, err := ParseLabels(data)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			targets = append(targets, target{name, root, lf})
		}
	}
	if len(targets) == 0 {
		t.Skip("no labeled corpus available")
	}

	type pooled struct {
		violations        int
		merge, ref, fp    float64
		nMerge, nRef, nFP int
	}
	pool := make([]pooled, len(lensVariants))
	lenses := fingerprint.Lenses()

	for _, tg := range targets {
		units, err := Load(tg.root, Population(tg.lf.Population))
		if err != nil || len(units) == 0 {
			t.Logf("[%s] load: %v", tg.name, err)
			continue
		}
		run := Analyze(units, retriever.DefaultOptions())
		fns, _ := lensFuncs(run.Units)
		bags := make([][][]fingerprint.LabelCount, len(lenses))
		idfs := make([]*fingerprint.LabelIDF, len(lenses))
		for li, l := range lenses {
			bags[li] = make([][]fingerprint.LabelCount, len(fns))
			for i := range fns {
				bags[li][i] = l.Bag(fns[i])
			}
			idfs[li] = fingerprint.LabelWeights(bags[li])
		}
		profs := make([]fingerprint.LensProfile, len(run.Pairs))
		for i, p := range run.Pairs {
			for li := range lenses {
				profs[i][li] = fingerprint.ScoreLens(bags[li][p.AIdx], bags[li][p.BIdx], idfs[li])
			}
		}
		wW := fingerprint.DefaultWeights().WL

		var base Scorecard
		for vi, v := range lensVariants {
			pairs := make([]analyzer.SimilarPair, len(run.Pairs))
			for i, p := range run.Pairs {
				wl := v.wl(p, profs[i])
				p.Score = math.Min(1, math.Max(0, p.Score-wW*p.Breakdown.WL+wW*wl))
				p.Breakdown.WL = wl
				pairs[i] = p
			}
			r := *run
			r.Pairs = pairs
			sc := Score(&r, tg.lf)
			t.Logf("[%s] %-24s %s", tg.name, v.name, scLine(sc))
			pl := &pool[vi]
			pl.violations += violations(sc)
			if n := sc.Present["merge"]; n > 0 {
				pl.merge += sc.MeanRank["merge"]
				pl.nMerge++
			}
			if n := sc.Present["refactor"]; n > 0 {
				pl.ref += sc.MeanRank["refactor"]
				pl.nRef++
			}
			if n := sc.Present["false_positive"]; n > 0 {
				pl.fp += sc.MeanRank["false_positive"]
				pl.nFP++
			}
			if vi == 0 {
				base = sc
				continue
			}
			var moved []string
			for i, res := range sc.Results {
				was := base.Results[i]
				if res.Rank == was.Rank {
					continue
				}
				moved = append(moved, fmt.Sprintf("%-14s %4d -> %-4d %s <-> %s",
					res.Label.Class, was.Rank, res.Rank, res.Label.A, res.Label.B))
			}
			sort.Strings(moved)
			for _, m := range moved {
				t.Logf("      %s", m)
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
	for vi, v := range lensVariants {
		pl := pool[vi]
		t.Logf("  %-24s merge %5s  refactor %5s  fp %5s  violations %d",
			v.name, avg(pl.merge, pl.nMerge), avg(pl.ref, pl.nRef), avg(pl.fp, pl.nFP), pl.violations)
	}
}

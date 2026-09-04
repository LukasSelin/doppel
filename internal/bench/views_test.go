package bench

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// The concept-views measurement, in two halves.
//
// TestViewsLadder is the label-free half: on every fetched rung, how often
// the three views of the concept signal disagree, in which direction, and
// what the joint distribution of the shape and feature views looks like —
// the numbers that decide whether comparator.ViewDisagreeSpread is a
// sensible constant. TestViewsBlend is the labeled half: every candidate
// blend for the exhibits slot of OverlapScore, scored against the golden
// labels through RescoreWith, with the selection rule stated in the log so
// the adopted DefaultOptions is a measurement and not a preference.
//
//	DOPPEL_BENCH_VIEWS=1 go test ./internal/bench/ -v -run 'TestViews'
//
// Both assert nothing. A candidate blend is rescored against lc.onto — the
// run's own learned vocabulary, snapshotted before any Rescore — and restored
// to it under production options, the rule every bench variant follows.

func TestViewsLadder(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_VIEWS") != "1" {
		t.Skip("set DOPPEL_BENCH_VIEWS=1 to run the concept-views measurement")
	}
	labels := labelsByCorpus(t)
	for _, c := range Corpora {
		if !Present(c) {
			continue
		}
		dir, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		units, err := Load(dir, PopExclude)
		if err != nil {
			t.Fatal(err)
		}
		run := Analyze(units, retriever.DefaultOptions())

		var compared, measured, disagree, featureOnly, taxonomyOnly int
		var sumShape, sumCorpus, sumFeature float64
		var features []float64
		var grid [4][4]int // shape bin × feature bin, quarters
		for _, p := range run.Pairs {
			if p.Evidence == nil {
				continue
			}
			compared++
			v := p.Evidence.Views
			if !v.HasFeature {
				continue
			}
			measured++
			sumShape += v.Shape
			sumCorpus += v.Corpus
			sumFeature += v.Feature
			features = append(features, v.Feature)
			grid[bin4(v.Shape)][bin4(v.Feature)]++
			if v.Disagree {
				disagree++
				if v.Feature > v.Shape {
					featureOnly++
				} else {
					taxonomyOnly++
				}
			}
		}
		sort.Float64s(features)
		t.Logf("[%s] %5d funcs  compared %6d  measured %6d  disagree %5d (%.1f%%: %d vocabulary-only, %d taxonomy-only)  spread %.2f",
			c.Name, len(units), compared, measured, disagree, pct(disagree, measured), featureOnly, taxonomyOnly, comparator.ViewDisagreeSpread)
		if measured > 0 {
			t.Logf("[%s]   mean shape %.3f  corpus %.3f  feature %.3f   feature p50 %.3f  p90 %.3f",
				c.Name, sumShape/float64(measured), sumCorpus/float64(measured), sumFeature/float64(measured),
				quantile(features, 0.5), quantile(features, 0.9))
			t.Logf("[%s]   shape\\feature   [0,.25)  [.25,.5)  [.5,.75)  [.75,1]", c.Name)
			for s := 0; s < 4; s++ {
				t.Logf("[%s]   %-14s %8d %9d %9d %9d", c.Name, binLabel(s), grid[s][0], grid[s][1], grid[s][2], grid[s][3])
			}
		}
		if lf, ok := labels[c.Name]; ok {
			byKey := make(map[string]comparator.ConceptViews, len(run.Pairs))
			for _, p := range run.Pairs {
				if p.Evidence != nil {
					byKey[pairKey(qualifiedName(p.A), qualifiedName(p.B))] = p.Evidence.Views
				}
			}
			for _, l := range lf.Labels {
				v, ok := byKey[pairKey(l.A, l.B)]
				if !ok {
					t.Logf("[%s]   %-15s %s / %s  not retrieved", c.Name, l.Class, l.A, l.B)
					continue
				}
				flag := ""
				if v.Disagree {
					flag = "  disagree"
				}
				t.Logf("[%s]   %-15s %s / %s  shape %.2f  corpus %.2f  feature %.2f  a-in-b %.2f  b-in-a %.2f%s",
					c.Name, l.Class, l.A, l.B, v.Shape, v.Corpus, v.Feature, v.AInB, v.BInA, flag)
			}
		}
	}
}

// blendVariant is one candidate for the exhibits slot.
type blendVariant struct {
	name string
	opt  comparator.Options
}

// blendVariants enumerates the candidates: the weighted simplex in 0.1
// steps (66 points, the corpus-only corner included as the baseline check),
// the geometric mean and the max.
func blendVariants() []blendVariant {
	var vs []blendVariant
	for s := 0; s <= 10; s++ {
		for c := 0; c <= 10-s; c++ {
			f := 10 - s - c
			vs = append(vs, blendVariant{
				name: fmt.Sprintf("w %.1f/%.1f/%.1f", float64(s)/10, float64(c)/10, float64(f)/10),
				opt:  comparator.Options{Exhibits: comparator.ViewBlend{Shape: float64(s) / 10, Corpus: float64(c) / 10, Feature: float64(f) / 10}},
			})
		}
	}
	vs = append(vs,
		blendVariant{"geometric", comparator.Options{Exhibits: comparator.ViewBlend{Mode: comparator.BlendGeometric}}},
		blendVariant{"max", comparator.Options{Exhibits: comparator.ViewBlend{Mode: comparator.BlendMax}}},
	)
	return vs
}

// admissible is the selection rule: zero violations, merge mean not worse,
// false-positive mean not lower. Among admissible variants the best refactor
// mean wins, ties going to the one closest to the corpus-only corner — the
// least departure from the incumbent that the labels cannot distinguish
// from a larger one.
func admissible(base, sc Scorecard) bool {
	return violations(sc) == 0 &&
		sc.MergePresent == base.MergePresent &&
		sc.MeanRank["merge"] <= base.MeanRank["merge"] &&
		sc.MeanRank["false_positive"] >= base.MeanRank["false_positive"]
}

func departure(opt comparator.Options) float64 {
	b := opt.Exhibits
	if b.Mode != comparator.BlendWeighted {
		return 2 // further than any point of the simplex
	}
	norm := b.Shape + b.Corpus + b.Feature
	if norm == 0 {
		return 0
	}
	return math.Abs(b.Shape/norm) + math.Abs(b.Corpus/norm-1) + math.Abs(b.Feature/norm)
}

func TestViewsBlend(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_VIEWS") != "1" {
		t.Skip("set DOPPEL_BENCH_VIEWS=1 to run the concept-views measurement")
	}
	corpora := loadLabeledCorpora(t)
	if len(corpora) == 0 {
		t.Skip("no labeled corpora available; run `task corpora` or set DOPPEL_BENCH_CORPUS/DOPPEL_BENCH_LABELS")
	}
	for i := range corpora {
		lc := &corpora[i]
		base := Score(lc.run, lc.lf)
		lc.run.RescoreWith(lc.onto, comparator.Options{})
		corpusOnly := Score(lc.run, lc.lf)
		if scLine(corpusOnly) != scLine(base) {
			t.Logf("[%s] NOTE: DefaultOptions is no longer the corpus-only corner; production %s, corpus-only %s",
				lc.name, scLine(base), scLine(corpusOnly))
		}
		t.Logf("[%s] %d functions, %d labels — production %s", lc.name, len(lc.run.Units), len(lc.lf.Labels), scLine(base))
		t.Logf("[%s] %-16s %-13s %-4s %s", lc.name, "blend", "verdict", "ok", "scorecard / moved labels")

		type outcome struct {
			v  blendVariant
			sc Scorecard
		}
		var chosen *outcome
		for _, v := range blendVariants() {
			lc.run.RescoreWith(lc.onto, v.opt)
			sc := Score(lc.run, lc.lf)
			moved, presence, _ := movement(base, sc)
			vd := verdict(base, sc, presence, len(moved))
			ok := admissible(base, sc)
			mark := "-"
			if ok {
				mark = "ok"
			}
			t.Logf("[%s] %-16s %-13s %-4s %s", lc.name, v.name, vd, mark, scLine(sc))
			if len(moved) > 0 {
				t.Logf("[%s] %-16s %-13s %-4s   %s", lc.name, "", "", "", strings.Join(moved, "; "))
			}
			if !ok {
				continue
			}
			o := outcome{v: v, sc: sc}
			if chosen == nil || better(o.sc, o.v.opt, chosen.sc, chosen.v.opt) {
				chosen = &o
			}
		}
		lc.run.RescoreWith(lc.onto, comparator.DefaultOptions())
		if chosen == nil {
			t.Logf("[%s] no admissible blend: nothing beats the corpus view on this corpus's labels", lc.name)
			continue
		}
		t.Logf("[%s] selected: %s — %s", lc.name, chosen.v.name, scLine(chosen.sc))
	}
}

// better orders admissible outcomes: refactor mean asc, then departure from
// the corpus-only corner asc.
func better(a Scorecard, ao comparator.Options, b Scorecard, bo comparator.Options) bool {
	ra, rb := a.MeanRank["refactor"], b.MeanRank["refactor"]
	if ra != rb {
		return ra < rb
	}
	return departure(ao) < departure(bo)
}

func labelsByCorpus(t *testing.T) map[string]LabelsFile {
	t.Helper()
	out := map[string]LabelsFile{}
	for _, lc := range loadLabeledCorpora(t) {
		out[lc.name] = lc.lf
	}
	return out
}

func bin4(x float64) int {
	b := int(x * 4)
	if b > 3 {
		b = 3
	}
	if b < 0 {
		b = 0
	}
	return b
}

func binLabel(b int) string {
	return [4]string{"[0,.25)", "[.25,.5)", "[.5,.75)", "[.75,1]"}[b]
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}

// quantile is the nearest-rank quantile of a sorted slice, the convention
// calibrate uses: a value some pair actually had, never an interpolation.
func quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(math.Ceil(q*float64(len(sorted)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(sorted) {
		i = len(sorted) - 1
	}
	return sorted[i]
}

package bench

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// BenchmarkCorpus times each pipeline stage on every fetched corpus, at the
// production population (--tests exclude) and default options.
//
// Corpora that have not been fetched are skipped, so the benchmark is useful
// after cloning one rung and complete after cloning all of them. Two derived
// metrics are reported alongside ns/op because raw ns/op across corpora of
// wildly different sizes says nothing on its own:
//
//	funcs   corpus size, so a reader can see what the ns/op bought
//	ns/func stage cost per function — the number that should stay flat as the
//	        corpus grows, and does not, for the quadratic-ish stages
//
// Parsing is timed separately from everything else: it is disk-bound and
// dominates a cold run, which would otherwise hide the stages that actually
// have interesting complexity.
func BenchmarkCorpus(b *testing.B) {
	for _, c := range Corpora {
		if !Present(c) {
			continue
		}
		dir, err := Path(c)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(c.Name, func(b *testing.B) {
			units, err := Load(dir, PopExclude)
			if err != nil {
				b.Fatalf("load %s: %v", c.Name, err)
			}
			if len(units) == 0 {
				b.Skipf("%s: no functions under population exclude", c.Name)
			}
			nf := float64(len(units))
			perFunc := func(b *testing.B) {
				b.ReportMetric(nf, "funcs")
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/nf, "ns/func")
			}

			b.Run("parse", func(b *testing.B) {
				for b.Loop() {
					if _, err := Load(dir, PopExclude); err != nil {
						b.Fatal(err)
					}
				}
				perFunc(b)
			})

			// Each stage below is timed with its predecessors already
			// computed, so the number is the stage's own cost — which means
			// the order here must be AnalyzeWith's own. It is not decoration:
			// StageTag learns the lexicon from resolved calls, so timing it
			// before StageGraph does not measure a cheaper tag stage, it
			// dereferences a nil graph.
			run := &Run{Units: units}
			b.Run("wl", func(b *testing.B) {
				for b.Loop() {
					run.StageWL()
				}
				perFunc(b)
			})
			run.StageWL()
			b.Run("callgraph", func(b *testing.B) {
				for b.Loop() {
					run.StageGraph()
				}
				perFunc(b)
			})
			run.StageGraph()
			b.Run("tag", func(b *testing.B) {
				for b.Loop() {
					run.StageTag()
				}
				perFunc(b)
			})
			run.StageTag()
			b.Run("map", func(b *testing.B) {
				for b.Loop() {
					run.StageMap()
				}
				perFunc(b)
			})
			run.StageMap()
			b.Run("retrieve", func(b *testing.B) {
				for b.Loop() {
					run.StageRetrieve(retriever.DefaultOptions())
				}
				perFunc(b)
			})
			run.StageRetrieve(retriever.DefaultOptions())
			run.StagePairs()
			np := float64(len(run.Pairs))
			b.Run("compare", func(b *testing.B) {
				for b.Loop() {
					run.StageCompare()
				}
				b.ReportMetric(np, "pairs")
				b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/max(np, 1), "ns/pair")
			})
			run.StageCompare()
			b.Run("rank", func(b *testing.B) {
				for b.Loop() {
					analyzer.SortForReport(run.Pairs, run.Units, 20, 2)
				}
				b.ReportMetric(np, "pairs")
			})

			// The whole ranking half end to end, parsing excluded: the
			// number a user feels when they re-run doppel on a warm tree.
			b.Run("analyze", func(b *testing.B) {
				for b.Loop() {
					r := Analyze(units, retriever.DefaultOptions())
					analyzer.SortForReport(r.Pairs, r.Units, 20, 2)
				}
				perFunc(b)
			})
		})
	}
}

package bench

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// The sensitivity sweep: vary each hand-set constant one at a time, re-run
// only the stages it reaches, and report how the labeled rankings move. It
// asserts nothing — its output is the answer to "which constants is this
// tool actually sensitive to", which is what decides whether a constant is
// worth calibrating, worth keeping, or worth deleting.
//
//	DOPPEL_BENCH_SWEEP=1 go test ./internal/bench/ -v -run TestSweep
//
// Verdicts: inert — no label changed rank or presence; moves — ranks
// shifted without a violation and the merge mean moved under 1.0;
// load-bearing — a violation, a presence change, or a merge-mean shift of
// at least 1.0. ChainTopN is not swept (explanation only, never ranks);
// struct-min and family-min are not swept (bench ranks the unfiltered set
// and no label scores the census); ForkShapeFloor is an annotation.

// retrievalState is what a retrieval-reaching variant replaces; snapshotting
// and reassigning it is exact because the stages build fresh slices.
type retrievalState struct {
	cands []retriever.Candidate
	stats retriever.Stats
	pairs []analyzer.SimilarPair
}

func snapshotRetrieval(r *Run) retrievalState {
	return retrievalState{cands: r.Cands, stats: r.Stats, pairs: r.Pairs}
}

func (s retrievalState) restore(r *Run) { r.Cands, r.Stats, r.Pairs = s.cands, s.stats, s.pairs }

type sweepVariant struct {
	constant string
	variant  string
	run      func(lc *labeledCorpus) Scorecard // must leave lc.run as it found it
}

func sweepVariants() []sweepVariant {
	var vs []sweepVariant
	retrieval := func(constant, variant string, mutate func(o *retriever.Options)) {
		vs = append(vs, sweepVariant{constant, variant, func(lc *labeledCorpus) Scorecard {
			saved := snapshotRetrieval(lc.run)
			opt := retriever.DefaultOptions()
			mutate(&opt)
			lc.run.Reretrieve(opt)
			sc := Score(lc.run, lc.lf)
			saved.restore(lc.run)
			return sc
		}})
	}
	retrieval("ChannelK 5", "3", func(o *retriever.Options) { o.ChannelK = 3 })
	retrieval("ChannelK 5", "8", func(o *retriever.Options) { o.ChannelK = 8 })
	retrieval("Threshold 0.60", "0.30", func(o *retriever.Options) { o.Threshold = 0.30 })
	retrieval("Threshold 0.60", "0.90", func(o *retriever.Options) { o.Threshold = 0.90 })
	// 12 is kept as a variant rather than dropped: it is the value this knob
	// held while the shape channel indexed the pattern multiset, and the
	// sweep is the record of why it moved.
	retrieval("MinNodes 18", "12", func(o *retriever.Options) { o.MinNodes = 12 })
	retrieval("MinNodes 18", "24", func(o *retriever.Options) { o.MinNodes = 24 })
	retrieval("MaxLabelDF 50", "25", func(o *retriever.Options) { o.MaxLabelDF = 25 })
	retrieval("MaxLabelDF 50", "100", func(o *retriever.Options) { o.MaxLabelDF = 100 })
	retrieval("MaxCallDF 50", "25", func(o *retriever.Options) { o.MaxCallDF = 25 })
	retrieval("MaxCallDF 50", "100", func(o *retriever.Options) { o.MaxCallDF = 100 })
	retrieval("MaxConceptDF 250", "125", func(o *retriever.Options) { o.MaxConceptDF = 125 })
	retrieval("MaxConceptDF 250", "500", func(o *retriever.Options) { o.MaxConceptDF = 500 })
	for i, name := range []string{"fp.AST 0.60", "fp.Flow 0.20", "fp.Depth 0.05", "fp.Signature 0.15"} {
		i := i
		for _, f := range []float64{0.5, 2} {
			f := f
			retrieval(name, fmt.Sprintf("x%g", f), func(o *retriever.Options) { o.Weights = fingerprint.DefaultWeights().Scaled(i, f) })
		}
	}

	def := ontology.Default()
	for _, rel := range def.ScoredRelations() {
		rel := rel
		w := def.Weight(rel)
		for _, f := range []float64{0.5, 2} {
			f := f
			vs = append(vs, sweepVariant{fmt.Sprintf("rel.%s %.4g", rel, w), fmt.Sprintf("x%g", f), func(lc *labeledCorpus) Scorecard {
				// Reweight the run's own vocabulary: its concept leaves are
				// learned, and Default() does not have them. w is the same
				// number under either — WithConcepts carries weights over.
				onto, err := ontology.WithWeightsOver(lc.onto, map[ontology.TermID]float64{rel: math.Min(w*f, 1)})
				if err != nil {
					panic(err)
				}
				lc.run.Rescore(onto)
				sc := Score(lc.run, lc.lf)
				lc.run.Rescore(lc.onto)
				return sc
			}})
		}
	}

	rank := func(constant, variant string, ro analyzer.RankOptions) {
		vs = append(vs, sweepVariant{constant, variant, func(lc *labeledCorpus) Scorecard { return ScoreWith(lc.run, lc.lf, ro) }})
	}
	rank("TrophicPower 2", "1", analyzer.RankOptions{TrophicPower: 1, TestCallDiscount: true})
	rank("TrophicPower 2", "3", analyzer.RankOptions{TrophicPower: 3, TestCallDiscount: true})
	rank("TestCallDiscount on", "off", analyzer.RankOptions{TrophicPower: 2, TestCallDiscount: false})
	return vs
}

// movement describes how the labels moved between two scorecards.
func movement(base, sc Scorecard) (moved []string, presenceChanged bool, maxShift int) {
	for i, r := range sc.Results {
		b := base.Results[i]
		if r.Rank == 0 || b.Rank == 0 {
			if r.Rank != b.Rank || r.Absent != b.Absent {
				presenceChanged = true
				moved = append(moved, fmt.Sprintf("%s/%s %s", r.Label.A, r.Label.B, presenceWord(b, r)))
			}
			continue
		}
		if r.Rank != b.Rank {
			moved = append(moved, fmt.Sprintf("%s/%s %d→%d", r.Label.A, r.Label.B, b.Rank, r.Rank))
			if d := r.Rank - b.Rank; abs(d) > abs(maxShift) {
				maxShift = d
			}
		}
	}
	return moved, presenceChanged, maxShift
}

func presenceWord(b, r LabelResult) string {
	if r.Rank == 0 {
		return fmt.Sprintf("%d→%s", b.Rank, r.Absent)
	}
	return fmt.Sprintf("%s→%d", b.Absent, r.Rank)
}

func verdict(base, sc Scorecard, presenceChanged bool, moved int) string {
	dMerge := math.Abs(sc.MeanRank["merge"] - base.MeanRank["merge"])
	switch {
	case violations(sc) > violations(base) || presenceChanged || dMerge >= 1.0:
		return "load-bearing"
	case moved > 0:
		return "moves"
	}
	return "inert"
}

func TestSweep(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_SWEEP") != "1" {
		t.Skip("set DOPPEL_BENCH_SWEEP=1 to run the sensitivity sweep")
	}
	corpora := loadLabeledCorpora(t)
	if len(corpora) == 0 {
		t.Skip("no labeled corpora available; run `task corpora` or set DOPPEL_BENCH_CORPUS/DOPPEL_BENCH_LABELS")
	}
	variants := sweepVariants()
	for i := range corpora {
		lc := &corpora[i]
		base := Score(lc.run, lc.lf)
		t.Logf("[%s] %d functions, %d labels — baseline %s", lc.name, len(lc.run.Units), len(lc.lf.Labels), scLine(base))
		t.Logf("[%s] %-28s %-6s %-13s %s", lc.name, "constant", "var", "verdict", "scorecard / moved labels")
		for _, v := range variants {
			sc := v.run(lc)
			moved, presence, _ := movement(base, sc)
			vd := verdict(base, sc, presence, len(moved))
			t.Logf("[%s] %-28s %-6s %-13s %s", lc.name, v.constant, v.variant, vd, scLine(sc))
			if len(moved) > 0 {
				t.Logf("[%s] %-28s %-6s %-13s   %s", lc.name, "", "", "", strings.Join(moved, "; "))
			}
		}
		t.Logf("[%s] not swept: ChainTopN (explanation only), struct-min and family-min (no bench analogue / census only), ForkShapeFloor (annotation)", lc.name)
	}
}

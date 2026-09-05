package bench

import (
	"fmt"

	"github.com/LukasSelin/doppel/internal/analyzer"
)

// Absence reasons a labeled pair can carry instead of a rank.
const (
	AbsentSuppressed   = "suppressed_by_max_per_func"
	AbsentNotRetrieved = "not_retrieved"
)

// LabelResult is one labeled pair's outcome against a ranking.
type LabelResult struct {
	Label  Label
	Rank   int     // 1-based; 0 when absent
	Key    float64 // RankKey at that rank; 0 when absent
	Absent string  // "", AbsentSuppressed, or AbsentNotRetrieved
}

// Scorecard is one scoring pass of a labels file against a completed Run,
// as plain data: what scoreLabels used to log and assert inline, extracted so
// the ablation and fitting harness can consume it programmatically. The three
// hard-assertion violation lists (MergeMissing, FPAboveMerge, FPInTop20) carry
// the same formatted pair descriptions the test failures print.
type Scorecard struct {
	Functions  int // corpus size under the labels' population
	Ranked     int
	Suppressed int

	Results []LabelResult // one per label, in labels-file order

	// Per-class aggregates over present (ranked) pairs.
	MeanRank map[string]float64
	Present  map[string]int

	MergeTotal   int
	MergePresent int
	MergeInTop50 int

	// Hard-assertion violations; all empty on a passing run.
	MergeMissing []string // merge pairs never retrieved
	FPAboveMerge []string // false positives ranked above the worst present merge
	FPInTop20    []string // false positives at rank <= 20
}

// Score ranks a completed Run and scores lf against it: every labeled pair
// gets a rank or an absence reason. Ranking mirrors scoreLabels' historical
// call exactly — SortForReport unbounded, max-per-func 2, best (lowest) rank
// per unordered name pair when duplicate qualified names exist.
func Score(run *Run, lf LabelsFile) Scorecard {
	return ScoreWith(run, lf, analyzer.DefaultRankOptions())
}

// ScoreWith is Score under an explicit rank key, for the sensitivity sweep.
func ScoreWith(run *Run, lf LabelsFile, ro analyzer.RankOptions) Scorecard {
	pairs := run.Pairs

	retrieved := make(map[string]bool, len(pairs))
	for _, p := range pairs {
		retrieved[pairKey(qualifiedName(run.Units[p.AIdx]), qualifiedName(run.Units[p.BIdx]))] = true
	}

	kept, suppressed := analyzer.SortForReportWith(pairs, run.Units, 0, 2, ro)

	rankOf := make(map[string]int, len(kept))
	keyOf := make(map[string]float64, len(kept))
	for i, p := range kept {
		k := pairKey(qualifiedName(run.Units[p.AIdx]), qualifiedName(run.Units[p.BIdx]))
		if _, ok := rankOf[k]; !ok {
			rankOf[k] = i + 1
			keyOf[k] = analyzer.RankKey(p, ro, run.Units)
		}
	}

	sc := Scorecard{
		Functions:  len(run.Units),
		Ranked:     len(kept),
		Suppressed: suppressed,
		MeanRank:   map[string]float64{},
		Present:    map[string]int{},
	}
	classSum := map[string]int{}

	var worstMerge int
	for _, l := range lf.Labels {
		k := pairKey(l.A, l.B)
		r := LabelResult{Label: l}
		if rank, ok := rankOf[k]; ok {
			r.Rank = rank
			r.Key = keyOf[k]
			sc.Present[l.Class]++
			classSum[l.Class] += rank
			if l.Class == "merge" && rank > worstMerge {
				worstMerge = rank
			}
		} else if retrieved[k] {
			r.Absent = AbsentSuppressed
		} else {
			r.Absent = AbsentNotRetrieved
		}
		sc.Results = append(sc.Results, r)

		switch l.Class {
		case "merge":
			sc.MergeTotal++
			if r.Rank > 0 {
				sc.MergePresent++
				if r.Rank <= 50 {
					sc.MergeInTop50++
				}
			} else if r.Absent == AbsentNotRetrieved {
				sc.MergeMissing = append(sc.MergeMissing, l.A+" / "+l.B)
			}
		case "false_positive":
			if r.Rank > 0 && r.Rank <= 20 {
				sc.FPInTop20 = append(sc.FPInTop20, fmt.Sprintf("%s / %s (rank %d)", l.A, l.B, r.Rank))
			}
		}
	}
	for class, n := range sc.Present {
		sc.MeanRank[class] = float64(classSum[class]) / float64(n)
	}

	// The FP-above-merge check needs the worst merge rank, so it runs after
	// the first pass — in labels-file order, like the assertions always did.
	for _, r := range sc.Results {
		if r.Label.Class != "false_positive" || r.Rank == 0 {
			continue
		}
		if worstMerge > 0 && r.Rank < worstMerge {
			sc.FPAboveMerge = append(sc.FPAboveMerge,
				fmt.Sprintf("%s / %s (rank %d, worst merge %d)", r.Label.A, r.Label.B, r.Rank, worstMerge))
		}
	}
	return sc
}

package bench

import (
	"fmt"
	"strings"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/snapshot"
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

	// Per-kind aggregates over present false positives that carry a Kind.
	KindMeanRank map[string]float64
	KindPresent  map[string]int

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

	// Candidates are indexed by their unordered name pair, each list in the
	// order it was built, so a label naming files can pick the unit it means
	// out of several that share a qualified name — and one naming none keeps
	// the old rule, the first (best-ranked) pair under those names.
	retrieved := make(map[string][]scoredPair, len(pairs))
	for _, p := range pairs {
		sp := run.scoredPair(p)
		retrieved[sp.key] = append(retrieved[sp.key], sp)
	}

	kept, suppressed := analyzer.SortForReportWith(pairs, run.Units, 0, 2, ro)

	ranked := make(map[string][]scoredPair, len(kept))
	for i, p := range kept {
		sp := run.scoredPair(p)
		sp.rank = i + 1
		sp.rankKey = analyzer.RankKey(p, ro, run.Units)
		ranked[sp.key] = append(ranked[sp.key], sp)
	}

	sc := Scorecard{
		Functions:    len(run.Units),
		Ranked:       len(kept),
		Suppressed:   suppressed,
		MeanRank:     map[string]float64{},
		Present:      map[string]int{},
		KindMeanRank: map[string]float64{},
		KindPresent:  map[string]int{},
	}
	classSum := map[string]int{}
	kindSum := map[string]int{}

	var worstMerge int
	for _, l := range lf.Labels {
		k := pairKey(l.A, l.B)
		r := LabelResult{Label: l}
		if sp, ok := firstMatch(ranked[k], l, run.Root); ok {
			rank := sp.rank
			r.Rank = rank
			r.Key = sp.rankKey
			sc.Present[l.Class]++
			classSum[l.Class] += rank
			if l.Kind != "" {
				sc.KindPresent[l.Kind]++
				kindSum[l.Kind] += rank
			}
			if l.Class == "merge" && rank > worstMerge {
				worstMerge = rank
			}
		} else if _, ok := firstMatch(retrieved[k], l, run.Root); ok {
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
				sc.MergeMissing = append(sc.MergeMissing, l.Pair())
			}
		case "false_positive":
			if r.Rank > 0 && r.Rank <= 20 {
				sc.FPInTop20 = append(sc.FPInTop20, fmt.Sprintf("%s (rank %d)", l.Pair(), r.Rank))
			}
		}
	}
	for class, n := range sc.Present {
		sc.MeanRank[class] = float64(classSum[class]) / float64(n)
	}
	for kind, n := range sc.KindPresent {
		sc.KindMeanRank[kind] = float64(kindSum[kind]) / float64(n)
	}

	// The FP-above-merge check needs the worst merge rank, so it runs after
	// the first pass — in labels-file order, like the assertions always did.
	for _, r := range sc.Results {
		if r.Label.Class != "false_positive" || r.Rank == 0 {
			continue
		}
		if worstMerge > 0 && r.Rank < worstMerge {
			sc.FPAboveMerge = append(sc.FPAboveMerge,
				fmt.Sprintf("%s (rank %d, worst merge %d)", r.Label.Pair(), r.Rank, worstMerge))
		}
	}
	return sc
}

// scoredPair is one candidate pair as the labels see it: its two sides' names
// and corpus-relative files, and — once ranked — its rank and rank key.
type scoredPair struct {
	key          string // pairKey of the two names
	nameA, nameB string
	fileA, fileB string
	rank         int
	rankKey      float64
	idx          int // position in run.Pairs, for a caller that matched a label and wants the pair back
}

func (r *Run) scoredPair(p analyzer.SimilarPair) scoredPair {
	a, b := r.Units[p.AIdx], r.Units[p.BIdx]
	return scoredPair{
		key:   pairKey(qualifiedName(a), qualifiedName(b)),
		nameA: qualifiedName(a), nameB: qualifiedName(b),
		fileA: snapshot.RelSlash(r.Root, a.File), fileB: snapshot.RelSlash(r.Root, b.File),
	}
}

// firstMatch is the first candidate under the label's names whose sides agree
// with every file the label pins, in either orientation.
func firstMatch(cands []scoredPair, l Label, root string) (scoredPair, bool) {
	for _, c := range cands {
		if sideMatches(c.nameA, c.fileA, l.A, l.AFile, root) && sideMatches(c.nameB, c.fileB, l.B, l.BFile, root) ||
			sideMatches(c.nameA, c.fileA, l.B, l.BFile, root) && sideMatches(c.nameB, c.fileB, l.A, l.AFile, root) {
			return c, true
		}
	}
	return scoredPair{}, false
}

// sideMatches reports whether a unit named name in file is the side a label
// names. With the corpus root known, a pinned file must equal the unit's
// root-relative path. Without one (a Run built from units, not from Load) the
// unit's path is whatever the loader recorded, so a pinned file matches it as
// a whole trailing path — never a partial path segment.
func sideMatches(name, file, wantName, wantFile, root string) bool {
	if name != wantName {
		return false
	}
	if wantFile == "" || file == wantFile {
		return true
	}
	return root == "" && strings.HasSuffix(file, "/"+wantFile)
}

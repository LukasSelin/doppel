package analyzer

import (
	"reflect"
	"testing"

	"github.com/lukse/doppel/internal/comparator"
)

func pairWith(aIdx, bIdx int, score float64, ret *Retrieval) SimilarPair {
	return SimilarPair{AIdx: aIdx, BIdx: bIdx, Score: score, Retrieval: ret}
}

func TestSortByEvidenceOrdersByTotalThenScoreThenIndex(t *testing.T) {
	pairs := []SimilarPair{
		pairWith(0, 1, 1.00, &Retrieval{Total: 0.2}), // perfect score, tiny evidence
		pairWith(2, 3, 0.70, &Retrieval{Total: 5.0}), // moderate score, strong evidence
		pairWith(4, 5, 0.90, &Retrieval{Total: 5.0}), // evidence tie, higher score
		pairWith(1, 6, 0.90, &Retrieval{Total: 5.0}), // full tie with above on evidence+score
		pairWith(7, 8, 0.95, nil),                    // nil retrieval counts as zero
	}
	got := SortByEvidence(pairs, 0)

	var order [][2]int
	for _, p := range got {
		order = append(order, [2]int{p.AIdx, p.BIdx})
	}
	want := [][2]int{
		{1, 6}, // 5.0 evidence, 0.90 score, lower AIdx
		{4, 5}, // 5.0 evidence, 0.90 score
		{2, 3}, // 5.0 evidence, 0.70 score
		{0, 1}, // 0.2 evidence beats nil despite lower score than the nil pair
		{7, 8}, // nil retrieval sorts last
	}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func corroborated(aIdx, bIdx int, total, overlap, score float64) SimilarPair {
	return corroboratedT(aIdx, bIdx, total, overlap, score, 1.0)
}

func corroboratedT(aIdx, bIdx int, total, overlap, score, trophic float64) SimilarPair {
	return SimilarPair{
		AIdx: aIdx, BIdx: bIdx, Score: score,
		Retrieval: &Retrieval{Total: total, TrophicSim: trophic},
		Evidence:  &comparator.StructuralEvidence{OverlapScore: overlap},
	}
}

// The ground-truth pin, using the real numbers from the human-reviewed
// corpus report: the production clone (moderate evidence, high overlap, high
// shape) must outrank the verbose-vocabulary false positives (huge evidence,
// low overlap, low shape), and the cross-package true clone must stay above
// them too.
func TestSortForReportCorroboratedOrdering(t *testing.T) {
	pairs := []SimilarPair{
		corroboratedT(0, 1, 1936.00, 0.18, 0.41, 0.55), // vocabulary FP
		corroboratedT(2, 3, 546.35, 0.59, 0.92, 0.93),  // production clone
		corroboratedT(4, 5, 1810.83, 0.26, 0.39, 0.55), // vocabulary FP
		corroboratedT(6, 7, 755.00, 0.37, 0.72, 0.91),  // cross-package clone
	}
	kept, suppressed := SortForReport(pairs, 0, 0)
	var order []int
	for _, p := range kept {
		order = append(order, p.AIdx)
	}
	if want := []int{2, 6, 4, 0}; !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v (clone, cross-package clone, then FPs)", order, want)
	}
	if suppressed != 0 {
		t.Errorf("suppressed = %d, want 0 with cap off", suppressed)
	}
}

// The family-skeleton pin, with the real numbers that motivated the squared
// trophic factor: a compose-send skeleton sibling (big evidence, decent
// overlap, but trophic 0.77 because each side carries a large unshared body)
// must rank BELOW every genuine family clone (trophic 0.97-1.0). Under a
// linear trophic factor the skeleton sibling beats the weakest clone by a
// fraction of a percent; squared, the margin is ~22%.
func TestSortForReportFamilySkeletonBelowFamilyClones(t *testing.T) {
	pairs := []SimilarPair{
		corroboratedT(0, 1, 939.84, 0.72, 0.6859, 0.77), // skeleton FP (BookFreight~InsuranceClaim shape)
		corroboratedT(2, 3, 456.72, 0.83, 0.9400, 1.00), // weakest family clone (ClaimUPS~DPDClaim shape)
		corroboratedT(4, 5, 456.72, 0.83, 0.9397, 0.97), // family clone
		corroboratedT(6, 7, 485.42, 0.83, 0.9400, 1.00), // family clone
	}
	kept, _ := SortForReport(pairs, 0, 0)
	if kept[len(kept)-1].AIdx != 0 {
		var order []int
		for _, p := range kept {
			order = append(order, p.AIdx)
		}
		t.Errorf("order = %v; the skeleton sibling must rank last", order)
	}
}

func TestSortForReportNilFallbacks(t *testing.T) {
	noEvidence := SimilarPair{AIdx: 0, BIdx: 1, Score: 0.5, Retrieval: &Retrieval{Total: 100, TrophicSim: 1}}
	noRetrieval := SimilarPair{AIdx: 2, BIdx: 3, Score: 0.99}
	small := corroborated(4, 5, 100, 0.4, 0.5) // key 20 < nil-Evidence key 50
	kept, _ := SortForReport([]SimilarPair{noRetrieval, small, noEvidence}, 0, 0)
	if kept[0].AIdx != 0 || kept[1].AIdx != 4 || kept[2].AIdx != 2 {
		t.Errorf("order = %d,%d,%d; want nil-Evidence (Total×Score) first, nil-Retrieval last",
			kept[0].AIdx, kept[1].AIdx, kept[2].AIdx)
	}
}

// A hub function in four pairs keeps exactly two; suppressed pairs are
// backfilled so topN stays full; both endpoints accrue.
func TestSortForReportDiversityCap(t *testing.T) {
	hub := []SimilarPair{
		corroborated(0, 1, 500, 1, 1), // hub=0, kept
		corroborated(0, 2, 400, 1, 1), // hub=0, kept (2nd appearance)
		corroborated(0, 3, 300, 1, 1), // hub full → suppressed
		corroborated(0, 4, 200, 1, 1), // suppressed
		corroborated(5, 6, 100, 1, 1), // backfills
	}
	kept, suppressed := SortForReport(hub, 3, 2)
	var order [][2]int
	for _, p := range kept {
		order = append(order, [2]int{p.AIdx, p.BIdx})
	}
	if want := [][2]int{{0, 1}, {0, 2}, {5, 6}}; !reflect.DeepEqual(order, want) {
		t.Errorf("kept = %v, want %v", order, want)
	}
	if suppressed != 2 {
		t.Errorf("suppressed = %d, want 2", suppressed)
	}

	// Both endpoints accrue: appearing as B counts too.
	both := []SimilarPair{
		corroborated(1, 9, 500, 1, 1), // 9 as B
		corroborated(9, 2, 400, 1, 1), // 9 as A (2nd)
		corroborated(3, 9, 300, 1, 1), // 9 full → suppressed
	}
	kept, suppressed = SortForReport(both, 0, 2)
	if len(kept) != 2 || suppressed != 1 {
		t.Errorf("kept %d / suppressed %d, want 2/1 (A and B appearances both count)", len(kept), suppressed)
	}

	// Cap 0 disables; topN still truncates.
	kept, suppressed = SortForReport(hub, 2, 0)
	if len(kept) != 2 || suppressed != 0 {
		t.Errorf("cap=0: kept %d / suppressed %d, want 2/0", len(kept), suppressed)
	}

	// topN 0 with cap active: cap still applies across the whole list.
	kept, suppressed = SortForReport(hub, 0, 2)
	if len(kept) != 3 || suppressed != 2 {
		t.Errorf("topN=0: kept %d / suppressed %d, want 3/2", len(kept), suppressed)
	}
}

func TestSortForReportTieDeterminism(t *testing.T) {
	tied := []SimilarPair{
		corroborated(4, 5, 100, 0.5, 0.8),
		corroborated(0, 6, 100, 0.5, 0.8),
		corroborated(0, 1, 100, 0.5, 0.9), // higher Score wins first
	}
	kept, _ := SortForReport(tied, 0, 0)
	if kept[0].BIdx != 1 || kept[1].BIdx != 6 || kept[2].AIdx != 4 {
		t.Errorf("tie order = %v; want Score desc then AIdx/BIdx asc", kept)
	}
}

func TestSortByEvidenceTruncates(t *testing.T) {
	pairs := []SimilarPair{
		pairWith(0, 1, 0.9, &Retrieval{Total: 3}),
		pairWith(2, 3, 0.9, &Retrieval{Total: 2}),
		pairWith(4, 5, 0.9, &Retrieval{Total: 1}),
	}
	if got := SortByEvidence(pairs, 2); len(got) != 2 || got[1].AIdx != 2 {
		t.Errorf("topN=2 kept %d pairs, want the two highest-evidence ones", len(got))
	}
	pairs2 := []SimilarPair{pairWith(0, 1, 0.9, nil)}
	if got := SortByEvidence(pairs2, 0); len(got) != 1 {
		t.Errorf("topN=0 must not truncate")
	}
}

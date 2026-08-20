package analyzer

import (
	"reflect"
	"testing"
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

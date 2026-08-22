package analyzer

import (
	"reflect"
	"testing"

	"github.com/LukasSelin/doppel/internal/comparator"
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

// SUT-aware test discounting: two tests with near-identical driver skeletons
// but no shared informative calls (CallSim 0 — they exercise different
// machinery) must sink below both production pairs and same-SUT test pairs.
// Production pairs never carry the CallSim factor.
func TestSortForReportSUTAwareTestDiscount(t *testing.T) {
	testPair := func(aIdx, bIdx int, total, overlap, score, trophic, callSim float64) SimilarPair {
		p := corroboratedT(aIdx, bIdx, total, overlap, score, trophic)
		p.Retrieval.CallSim = callSim
		p.A.File = "pkg/a_test.go"
		p.B.File = "pkg/b_test.go"
		return p
	}
	pairs := []SimilarPair{
		// The old-#6 shape: huge evidence, high overlap, decent trophic —
		// but zero shared informative calls.
		testPair(0, 1, 1812.48, 0.68, 0.53, 0.72, 0.0),
		// A same-SUT test pair: modest numbers but real shared call mass.
		testPair(2, 3, 400.00, 0.60, 0.80, 0.85, 0.60),
		// A production pair with low CallSim that must NOT be discounted.
		func() SimilarPair {
			p := corroboratedT(4, 5, 300.00, 0.59, 0.92, 0.93)
			p.Retrieval.CallSim = 0.0
			p.A.File = "pkg/a.go"
			p.B.File = "pkg/b.go"
			return p
		}(),
	}
	kept, _ := SortForReport(pairs, 0, 0)
	var order []int
	for _, p := range kept {
		order = append(order, p.AIdx)
	}
	// prod pair ≈ 141; same-SUT test pair ≈ 83; zero-CallSim test pair = 0.
	if want := []int{4, 2, 0}; !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v (prod, same-SUT test, skeleton test last)", order, want)
	}
	// A mixed test/prod pair is not discounted either.
	mixed := corroboratedT(6, 7, 100, 0.5, 0.5, 0.5)
	mixed.Retrieval.CallSim = 0.0
	mixed.A.File = "pkg/a_test.go"
	mixed.B.File = "pkg/b.go"
	kept, _ = SortForReport([]SimilarPair{mixed}, 0, 0)
	if len(kept) != 1 || kept[0].Retrieval.Total != 100 {
		t.Fatal("mixed pair mangled")
	}
	// Its key must be nonzero (no CallSim factor): verify it outranks a
	// true test pair with CallSim 0.
	zeroTest := corroboratedT(8, 9, 100, 0.5, 0.5, 0.5)
	zeroTest.Retrieval.CallSim = 0.0
	zeroTest.A.File = "x_test.go"
	zeroTest.B.File = "y_test.go"
	kept, _ = SortForReport([]SimilarPair{zeroTest, mixed}, 0, 0)
	if kept[0].AIdx != 6 {
		t.Errorf("mixed pair should outrank the zero-CallSim test pair, got %v first", kept[0].AIdx)
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

// One rank key: SortForReport and RankKey(DefaultRankOptions) must agree,
// and the default trophic term must be the exact t*t form.
func TestRankKeyMatchesSortForReport(t *testing.T) {
	mk := func(a, b int, total, score, trophic, overlap float64) SimilarPair {
		ev := comparator.StructuralEvidence{OverlapScore: overlap}
		return SimilarPair{AIdx: a, BIdx: b, Score: score,
			Retrieval: &Retrieval{Total: total, TrophicSim: trophic}, Evidence: &ev}
	}
	pairs := []SimilarPair{mk(0, 1, 10, 0.9, 0.5, 0.4), mk(1, 2, 8, 0.95, 0.9, 0.6), mk(2, 3, 30, 0.7, 0.3, 0.5), mk(0, 3, 30, 0.7, 0.3, 0.5)}
	for _, p := range pairs {
		t2 := p.Retrieval.TrophicSim * p.Retrieval.TrophicSim
		want := p.Retrieval.Total * p.Score * t2 * p.Evidence.OverlapScore
		if got := RankKey(p, DefaultRankOptions()); got != want {
			t.Errorf("RankKey = %v, want %v (exact t*t form)", got, want)
		}
	}
	a, _ := SortForReport(append([]SimilarPair(nil), pairs...), 0, 0)
	b, _ := SortForReportWith(append([]SimilarPair(nil), pairs...), 0, 0, DefaultRankOptions())
	for i := range a {
		if a[i].AIdx != b[i].AIdx || a[i].BIdx != b[i].BIdx {
			t.Fatalf("orders differ at %d", i)
		}
	}
	// A different power reorders: with power 1 the high-trophic pair no
	// longer out-keys the high-mass one by as much.
	if k1, k2 := RankKey(pairs[1], RankOptions{TrophicPower: 1, TestCallDiscount: true}), RankKey(pairs[1], DefaultRankOptions()); k1 <= k2 {
		t.Errorf("power 1 should key higher than power 2 for trophic < 1: %v vs %v", k1, k2)
	}
	if RankKey(SimilarPair{}, DefaultRankOptions()) != 0 {
		t.Error("nil Retrieval must key 0")
	}
}

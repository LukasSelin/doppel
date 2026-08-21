package analyzer

import "sort"

// SortByEvidence orders pairs for the final report: retrieval evidence mass
// descending (a nil Retrieval counts as zero), then fingerprint score
// descending, then AIdx and BIdx ascending so ties are byte-stable across
// runs. When topN > 0 the result is truncated to that many pairs. The slice
// is sorted in place and returned for convenience.
func SortByEvidence(pairs []SimilarPair, topN int) []SimilarPair {
	total := func(p SimilarPair) float64 {
		if p.Retrieval == nil {
			return 0
		}
		return p.Retrieval.Total
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		ti, tj := total(pairs[i]), total(pairs[j])
		if ti != tj {
			return ti > tj
		}
		if pairs[i].Score != pairs[j].Score {
			return pairs[i].Score > pairs[j].Score
		}
		if pairs[i].AIdx != pairs[j].AIdx {
			return pairs[i].AIdx < pairs[j].AIdx
		}
		return pairs[i].BIdx < pairs[j].BIdx
	})
	if topN > 0 && len(pairs) > topN {
		pairs = pairs[:topN]
	}
	return pairs
}

// SortForReport is the pipeline's final ranking: corroborated evidence.
// The rank key is Retrieval.Total × Evidence.OverlapScore × Score — evidence
// mass discounted by architectural corroboration and structural similarity —
// which demotes pairs whose mass comes from a verbose shared vocabulary with
// no other agreement (the drawing-API failure mode) without touching any
// displayed quantity. A nil Evidence contributes factor 1 (evidence×score);
// a nil Retrieval ranks 0, matching SortByEvidence. Ties break on Score
// desc, then AIdx, BIdx.
//
// After sorting, a per-function diversity cap walks the ranked list keeping
// a pair only while both endpoints (by unit index, positional as always)
// have appeared in fewer than maxPerFunc kept pairs — a boilerplate hub is
// one finding, not four — until topN pairs are kept (topN 0 = unbounded;
// maxPerFunc 0 disables the cap). Suppressed pairs are skipped and
// backfilled by lower-ranked ones; the count of skips is returned for the
// stderr accounting. SortByEvidence remains the simple library ranking.
func SortForReport(pairs []SimilarPair, topN, maxPerFunc int) ([]SimilarPair, int) {
	key := func(p SimilarPair) float64 {
		if p.Retrieval == nil {
			return 0
		}
		k := p.Retrieval.Total * p.Score
		if p.Evidence != nil {
			k *= p.Evidence.OverlapScore
		}
		return k
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		ki, kj := key(pairs[i]), key(pairs[j])
		if ki != kj {
			return ki > kj
		}
		if pairs[i].Score != pairs[j].Score {
			return pairs[i].Score > pairs[j].Score
		}
		if pairs[i].AIdx != pairs[j].AIdx {
			return pairs[i].AIdx < pairs[j].AIdx
		}
		return pairs[i].BIdx < pairs[j].BIdx
	})

	if maxPerFunc <= 0 {
		if topN > 0 && len(pairs) > topN {
			return pairs[:topN], 0
		}
		return pairs, 0
	}

	seen := make(map[int]int)
	kept := make([]SimilarPair, 0, len(pairs))
	suppressed := 0
	for _, p := range pairs {
		if topN > 0 && len(kept) == topN {
			break
		}
		if seen[p.AIdx] >= maxPerFunc || seen[p.BIdx] >= maxPerFunc {
			suppressed++
			continue
		}
		seen[p.AIdx]++
		seen[p.BIdx]++
		kept = append(kept, p)
	}
	return kept, suppressed
}

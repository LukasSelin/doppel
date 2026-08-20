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

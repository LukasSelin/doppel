package analyzer

import (
	"sort"

	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

// SimilarPair holds two code units and their static similarity score.
type SimilarPair struct {
	A, B       parser.CodeUnit
	AIdx, BIdx int                            // positions in the units slice, for looking up parallel data
	Score      float64                        // composite fingerprint similarity, 0.0-1.0
	Breakdown  fingerprint.Breakdown          // per-component scores behind Score
	Evidence   *comparator.StructuralEvidence // populated by structural comparison stage; nil until then
}

// FindSimilar compares every pair of function fingerprints and returns those
// above threshold, sorted by score descending, limited to topN results.
// Units whose body has fewer than minNodes AST nodes are excluded: trivial
// accessors match each other perfectly and drown out real candidates.
func FindSimilar(units []parser.CodeUnit, threshold float64, topN, minNodes int) []SimilarPair {
	// Collect the indices worth comparing once, rather than re-testing inside
	// the O(n^2) loop.
	var idx []int
	for i := range units {
		if units[i].Fingerprint.Nodes >= minNodes && units[i].Fingerprint.Nodes > 0 {
			idx = append(idx, i)
		}
	}

	var pairs []SimilarPair
	for a := 0; a < len(idx); a++ {
		for b := a + 1; b < len(idx); b++ {
			i, j := idx[a], idx[b]
			bd := fingerprint.Similarity(units[i].Fingerprint, units[j].Fingerprint)
			if bd.Score >= threshold {
				pairs = append(pairs, SimilarPair{
					A:         units[i],
					B:         units[j],
					AIdx:      i,
					BIdx:      j,
					Score:     bd.Score,
					Breakdown: bd,
				})
			}
		}
	}

	// Ties break on position so the report is byte-identical across runs.
	sort.SliceStable(pairs, func(i, j int) bool {
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

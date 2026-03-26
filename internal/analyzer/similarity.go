package analyzer

import (
	"math"
	"sort"

	"github.com/lukse/doppel/internal/comparator"
	"github.com/lukse/doppel/internal/parser"
)

// SimilarPair holds two code units and their cosine similarity score.
type SimilarPair struct {
	A, B        parser.CodeUnit
	Score       float64
	Explanation string                       // populated by reflector when --reflect-model is set; empty otherwise
	Evidence    *comparator.StructuralEvidence // populated by structural comparison pass; nil until then
}

// ProgressFunc is called periodically to report progress.
type ProgressFunc func(done, total int)

// FindSimilar computes pairwise cosine similarity and returns pairs above threshold,
// sorted by score descending, limited to topN results.
// If progress is non-nil it is called periodically with the number of comparisons completed.
func FindSimilar(units []parser.CodeUnit, embeddings [][]float64, threshold float64, topN int, progress ProgressFunc) []SimilarPair {
	var pairs []SimilarPair
	n := len(units)
	total := n * (n - 1) / 2
	done := 0

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			score := cosine(embeddings[i], embeddings[j])
			if score >= threshold {
				pairs = append(pairs, SimilarPair{
					A:     units[i],
					B:     units[j],
					Score: score,
				})
			}
			done++
			if progress != nil && (done%1000 == 0 || done == total) {
				progress(done, total)
			}
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Score > pairs[j].Score
	})

	if topN > 0 && len(pairs) > topN {
		pairs = pairs[:topN]
	}
	return pairs
}

func cosine(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

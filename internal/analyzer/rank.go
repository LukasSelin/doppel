package analyzer

import (
	"math"
	"sort"

	"github.com/LukasSelin/doppel/internal/parser"
)

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
// The rank key is Retrieval.Total × Evidence.OverlapScore × Score² ×
// TrophicSim² — evidence mass discounted by architectural corroboration,
// structural similarity, and squared trophic similarity (Score is squared
// for the reason DefaultRankOptions gives). The first two
// factors demote pairs whose mass comes from a verbose shared vocabulary
// with no other agreement (the drawing-API failure mode); the squared
// trophic factor separates genuine clone families from family-skeleton
// siblings that share a large common prologue but carry substantial unshared
// bodies. Squared, not linear: the Dice 2S/(E_A+E_B) approximates, when
// squared, the product of the two per-side shared fractions S/E_A · S/E_B —
// the pair is discounted once per side that does its own thing. (A linear
// factor verifiably leaves a skeleton sibling within a fraction of a percent
// of a true family clone; the golden benchmark pins the separation.)
// Pairs where both sides live in _test.go files carry one further linear
// factor, CallSim — SUT-aware test discounting; see the comment at the key.
// No displayed quantity changes. A nil Evidence contributes factor 1;
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
func SortForReport(pairs []SimilarPair, units []parser.CodeUnit, topN, maxPerFunc int) ([]SimilarPair, int) {
	return SortForReportWith(pairs, units, topN, maxPerFunc, DefaultRankOptions())
}

// RankOptions parameterizes the rank key for measurement. Production always
// ranks with DefaultRankOptions; the bench sensitivity sweep varies these
// per call, which is why they are an argument and not a global.
type RankOptions struct {
	TrophicPower     float64 // exponent on TrophicSim; 2 in production
	ShapePower       float64 // exponent on the code-shape Score; 2 in production, 0 reads as 1
	TestCallDiscount bool    // multiply test/test pairs by CallSim
}

// DefaultRankOptions is the shipped key: code-shape and trophic squared,
// tests discounted.
//
// ShapePower is 2 because code-shape is the factor that separates labelled
// merges from labelled false positives, and a linear factor let retrieval
// mass and overlap outvote it: sibling and mirror methods on one receiver
// (Get/Set/Delete, Encode/Decode) carry as much overlap as a real merge and
// reached the top 20 on bodies that are half alike. Measured over cobra and
// three private corpora (TestOverlapRank): pooled violations 39 -> 34,
// merge mean rank 10.5 -> 8.5, false-positive mean 212.8 -> 215.1, cobra's
// hard assertions green. The price is the refactor class, 9.8 -> 11.5 —
// pairs whose bodies differ by definition sink — and that trade, merges
// before refactors, was chosen deliberately. 3 moves further both ways and
// was not taken.
func DefaultRankOptions() RankOptions {
	return RankOptions{TrophicPower: 2, ShapePower: 2, TestCallDiscount: true}
}

// shapeFactor is the code-shape Score raised to the key's ShapePower. s*s
// rather than math.Pow for the default, for the reason the trophic factor
// gives. A zero power reads as linear — the key before the option existed —
// so an options literal that predates the field never drops shape from the
// key by omission.
func shapeFactor(s, power float64) float64 {
	switch power {
	case 0, 1:
		return s
	case 2:
		return s * s
	}
	return math.Pow(s, power)
}

// RankKey is the corroborated-evidence ordering quantity SortForReport uses,
// exposed so a scorecard can print it and a sweep can vary it — there is one
// definition, so the bench copy cannot drift. A nil Retrieval keys 0; a nil
// Evidence contributes factor 1.
//
// units is the slice a pair's AIdx/BIdx index, and is read for one thing only:
// whether both sides are test units. Pass nil when there is none — a pair with
// no units behind it then ranks without the test discount, which is exactly
// what it did when the units were embedded and a pair built without them
// carried two zero-value CodeUnits whose empty File is not a test file.
func RankKey(p SimilarPair, o RankOptions, units []parser.CodeUnit) float64 {
	if p.Retrieval == nil {
		return 0
	}
	t := p.Retrieval.TrophicSim
	// t*t, not math.Pow(t, 2): Pow is not guaranteed bit-equal, and the
	// default key must stay byte-identical to what every baseline ranked by.
	var trophic float64
	if o.TrophicPower == 2 {
		trophic = t * t
	} else {
		trophic = math.Pow(t, o.TrophicPower)
	}
	k := p.Retrieval.Total * shapeFactor(p.Score, o.ShapePower) * trophic
	if p.Evidence != nil {
		k *= p.Evidence.OverlapScore
	}
	// SUT-aware test discounting: two tests are related through what
	// they exercise, not through their driver skeleton — near-identical
	// table-driven harnesses over different functions share no
	// informative call tokens and key to zero, while tests of the same
	// machinery keep their shared call mass. The test rule is the same one
	// --tests uses, asked of the unit's own frontend. Production pairs are
	// untouched; under --tests only, the whole hygiene view becomes
	// SUT-aware, which is the point.
	if o.TestCallDiscount && bothTests(p, units) {
		k *= p.Retrieval.CallSim
	}
	return k
}

// SortForReportWith is SortForReport under an explicit rank key.
func SortForReportWith(pairs []SimilarPair, units []parser.CodeUnit, topN, maxPerFunc int, o RankOptions) ([]SimilarPair, int) {
	key := func(p SimilarPair) float64 { return RankKey(p, o, units) }
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

// bothTests reports whether both sides of a pair are test units.
//
// Out-of-range or absent indices answer false rather than panicking: a pair
// can legitimately be built without a units slice behind it (FindSimilar's
// callers, the bench harness's synthetic pairs, every rank test), and such a
// pair must rank exactly as it did when the units travelled inside it — where
// a zero-value CodeUnit's empty File was simply not a test file.
func bothTests(p SimilarPair, units []parser.CodeUnit) bool {
	if p.AIdx < 0 || p.BIdx < 0 || p.AIdx >= len(units) || p.BIdx >= len(units) {
		return false
	}
	return parser.IsTestUnit(units[p.AIdx]) && parser.IsTestUnit(units[p.BIdx])
}

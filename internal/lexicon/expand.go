package lexicon

import (
	"math"
	"sort"
)

// fit learns one concept's vocabulary from a founding member set, and is the
// single place the learner decides what "characteristic of a concept" means.
// Both paths use it: a seeded concept's members are the units its rule fired
// on, an emergent concept's are the units carrying a co-occurring feature
// clique.
//
// A feature earns its place by lift over the corpus base rate,
//
//	PMI(c,f) = ln( P(f | members) / P(f) )
//
// the same "beyond chance" arithmetic culture's ecology applies to tag~call
// pairs, generalized to concept~feature. Prevalence alone would not do: nearly
// every Go function returns and branches, so an unfiltered vocabulary reports
// facts about the language rather than about the concept — the lesson the
// report's practice section already learned. The weight is lift × idf, so a
// feature must be both distinctive *to the concept* and rare *in the corpus*.
//
// It returns the vocabulary, the median founding evidence (the half-point the
// reported confidence saturates around) and the membership floor at
// Options.FloorQuantile — see assign for why those are two numbers.
func fit(c *corpus, members []int, opt Options) ([]Feature, float64, float64) {
	if len(members) == 0 {
		return nil, 0, 0
	}
	counts := make(map[string]int)
	for _, m := range members {
		for _, f := range c.features[m] {
			if _, ok := c.idf[f]; ok { // surviving features only
				counts[f]++
			}
		}
	}

	m := float64(len(members))
	n := float64(c.n)
	var features []Feature
	for _, f := range sortedKeys(counts) { // ascending: no map order decides anything
		count := counts[f]
		if count < opt.MinSupport {
			continue
		}
		lift := math.Log((float64(count) / m) / (float64(c.df[f]) / n))
		if lift < opt.MinLift {
			continue
		}
		features = append(features, Feature{
			Name:   f,
			Weight: lift * c.idf[f],
			DF:     c.df[f],
			Count:  count,
		})
	}
	if len(features) == 0 {
		return nil, 0, 0
	}
	sort.Slice(features, func(i, j int) bool {
		if features[i].Weight != features[j].Weight {
			return features[i].Weight > features[j].Weight
		}
		return features[i].Name < features[j].Name
	})

	w := weightsOf(features)
	ev := make([]float64, 0, len(members))
	for _, mi := range members {
		ev = append(ev, c.evidence(mi, w))
	}
	sort.Float64s(ev)
	scale := median(ev)
	if scale <= 0 {
		// Every founding member scored zero, which can only happen when the
		// vocabulary is empty; guard anyway so confidence never divides by
		// zero and every member does not read 1.0.
		scale = math.SmallestNonzeroFloat64
	}
	floor := quantile(ev, opt.FloorQuantile)
	if floor <= 0 {
		// A concept whose bar is zero would admit every function carrying any
		// of its vocabulary at all, which is not membership in anything. The
		// weakest positive founding evidence is the honest floor.
		floor = math.SmallestNonzeroFloat64
		for _, e := range ev {
			if e > 0 {
				floor = e
				break
			}
		}
	}
	return features, scale, floor
}

// expandSeeds turns each seed label into a concept whose vocabulary is learned
// rather than declared. The rule that produced the seed contributes nothing
// beyond the member set: which functions to look at. What the concept *is*, on
// this corpus, is whatever those functions share and the rest of the corpus
// does not.
//
// A seed whose members share no distinctive feature is dropped, and that is a
// finding rather than a failure — the rule fired on things this codebase has no
// common way of writing, so there is no concept there to report.
//
// Features the seeded concepts use are recorded in claimed, which bounds only
// what may *seed* an emergent cluster. Vocabularies are free to overlap: fit
// always searches the whole surviving feature set.
func expandSeeds(c *corpus, seeds [][]string, claimed map[string]bool,
	stats *Stats, opt Options) []Concept {

	if len(seeds) == 0 {
		return nil
	}
	members := make(map[string][]int)
	for i := 0; i < c.n && i < len(seeds); i++ {
		for _, tag := range seeds[i] {
			members[tag] = append(members[tag], i)
		}
	}

	var out []Concept
	for _, tag := range sortedKeys(members) {
		features, scale, floor := fit(c, members[tag], opt)
		if len(features) == 0 {
			stats.SeedsDropped++
			continue
		}
		for _, f := range features {
			claimed[f.Name] = true
		}
		out = append(out, Concept{Seed: tag, Features: features, Scale: scale, Floor: floor})
		stats.Seeded++
	}
	return out
}

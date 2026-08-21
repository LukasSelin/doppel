// Package retriever generates candidate pairs for the expensive structural
// comparison stage. It treats candidate generation as an information-retrieval
// problem: three cheap channels — structural shape, ontology concepts, and
// resolved calls — each retrieve per-function top-K neighbors weighted by
// corpus rarity, and the union goes to the comparator. The first-stage number
// is therefore evidence mass (how much rare, informative material two
// functions share), not duplicate confidence: a perfect match on a ubiquitous
// trivial shape carries almost no evidence, while a moderate match supported
// by rare shingles, rare tags, or rare callees carries a lot.
package retriever

import (
	"sort"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// Channel names, in the fixed order they appear in Candidate.Channels.
const (
	ChannelShape   = "shape"
	ChannelConcept = "concept"
	ChannelCall    = "call"
)

// Options tunes retrieval. The df caps are not exposed as flags — they exist
// on Options so tests can shrink them without needing 50+-function fixtures.
type Options struct {
	ChannelK     int     // per-function, per-channel top-K
	Threshold    float64 // structural-channel floor on the exact fingerprint score
	MinNodes     int     // structural-channel eligibility gate on Fingerprint.Nodes
	MaxPatternDF int     // structural patterns present in more units than this carry no evidence
	MaxCallDF    int     // call tokens present in more units than this carry no evidence
	MaxConceptDF int     // concept postings larger than this are skipped for enumeration
	ChainTopN    int     // shared-structure explanations kept per pair
}

// DefaultOptions returns the production defaults. ChannelK mirrors the
// --channel-k flag default; the caps are fixed constants chosen so that
// corpus-wide idioms (Error() shapes, fmt.Sprintf) drop out of the indexes
// entirely while genuinely shared machinery stays in.
func DefaultOptions() Options {
	return Options{
		ChannelK:     5,
		Threshold:    0.60,
		MinNodes:     12,
		MaxPatternDF: 50,
		MaxCallDF:    50,
		MaxConceptDF: 250,
		ChainTopN:    3,
	}
}

// Candidate is one retrieved pair. AIdx < BIdx always; both index the units
// slice passed to Retrieve. The three evidence fields are Σ ln(N/df) over the
// shared rare features of each channel — nats of log-evidence over the same
// corpus, which is what makes summing them into Total coherent. Evidence is
// computed definitively for every union pair regardless of which channel
// admitted it, so a call-admitted pair still reports its shape mass.
type Candidate struct {
	AIdx, BIdx int
	Breakdown  fingerprint.Breakdown // exact fingerprint similarity, always computed
	Shape      float64               // shared structural energy, Σ IC·min(count) over shared patterns
	Concept    float64               // shared tag information, Σ IC(LCS) over the best matching
	Call       float64               // shared rare-call IDF mass
	Total      float64               // Shape + Concept + Call, summed in that order
	TrophicSim float64               // 2·SharedEnergy/(E_A+E_B): weighted Dice over pattern energy
	CallSim    float64               // call-channel Dice: mutual fraction of informative call energy
	Channels   []string              // admission provenance, subset of {shape, concept, call}
	Chains     []SharedPattern       // highest-energy shared structures, the explanation
}

// Stats describes one retrieval run, for the stderr summary and evaluation.
type Stats struct {
	ShapePairs        int // distinct pairs admitted by the structural channel
	ConceptPairs      int // distinct pairs admitted by the concept channel
	CallPairs         int // distinct pairs admitted by the call channel
	Union             int // unique pairs across all channels
	OnlyConcept       int // pairs only the concept channel admitted
	OnlyCall          int // pairs only the call channel admitted
	Suppressed        int // shape-eligible units whose every pattern was df-capped out
	LargeBuckets      int // exact pattern-multiset identity buckets with > largeBucketSize members
	SurvivingPatterns int // distinct structural patterns carrying evidence
}

// pairKey orders a pair as (min, max) so both admission directions collide.
type pairKey [2]int

func orderPair(a, b int) pairKey {
	if a > b {
		a, b = b, a
	}
	return pairKey{a, b}
}

// admission records which channels admitted a pair.
type admission struct {
	shape, concept, call bool
}

// Retrieve runs all three channels over the corpus and returns the deduped
// union with definitive per-pair evidence, sorted by (AIdx, BIdx). Ranking by
// evidence happens downstream, after the comparator — retriever output order
// is positional so the pipeline's positional doc lookup stays obvious.
func Retrieve(units []parser.CodeUnit, g *concepter.Graph,
	onto *ontology.Ontology, ic *ontology.IC, opt Options) ([]Candidate, Stats) {

	scorer := ontology.NewScorer(onto, ic)
	sim := newSimCache(units)

	shapes := buildShapeIndex(units, opt)
	calls := buildCallIndex(units, g, opt)
	concepts := buildConceptIndex(units, onto, scorer, opt)

	admitted := make(map[pairKey]*admission)
	admit := func(pairs []pairKey, channel string) int {
		distinct := 0
		for _, k := range pairs {
			a := admitted[k]
			if a == nil {
				a = &admission{}
				admitted[k] = a
			}
			var seen *bool
			switch channel {
			case ChannelShape:
				seen = &a.shape
			case ChannelConcept:
				seen = &a.concept
			case ChannelCall:
				seen = &a.call
			}
			if !*seen {
				*seen = true
				distinct++
			}
		}
		return distinct
	}

	var stats Stats
	stats.ShapePairs = admit(shapes.admitPairs(sim, opt), ChannelShape)
	stats.ConceptPairs = admit(concepts.admitPairs(opt), ChannelConcept)
	stats.CallPairs = admit(calls.admitPairs(opt), ChannelCall)
	stats.Union = len(admitted)
	stats.Suppressed = shapes.suppressed
	stats.LargeBuckets = shapes.largeBuckets
	stats.SurvivingPatterns = len(shapes.idf)

	cands := evaluate(admitted, shapes, concepts, calls, sim, opt, &stats)
	return cands, stats
}

// Probe retrieves the corpus functions most related to units[probeIdx]. It
// runs the same three channels, the same gates, and the same definitive
// evidence arithmetic as Retrieve — only the admission loop is narrowed to
// the probe's turn, so a query costs index building plus one function's
// retrieval rather than the corpus's.
//
// The probe must already be a member of units: every index is positional, and
// scoring a unit against statistics it is excluded from would misrepresent
// how it sits in this corpus. The caller appends it before tagging and graph
// building, which also hands it resolved callees for free.
func Probe(units []parser.CodeUnit, probeIdx int, g *concepter.Graph,
	onto *ontology.Ontology, ic *ontology.IC, opt Options) ([]Candidate, Stats) {

	scorer := ontology.NewScorer(onto, ic)
	sim := newSimCache(units)

	shapes := buildShapeIndex(units, opt)
	calls := buildCallIndex(units, g, opt)
	concepts := buildConceptIndex(units, onto, scorer, opt)

	admitted := make(map[pairKey]*admission)
	// Same counting semantics as Retrieve's admit closure: distinct counts
	// pairs newly marked for this channel, even when another channel admitted
	// the pair first.
	admitOne := func(pairs []pairKey, seen func(*admission) *bool) int {
		distinct := 0
		for _, k := range pairs {
			a := admitted[k]
			if a == nil {
				a = &admission{}
				admitted[k] = a
			}
			if f := seen(a); !*f {
				*f = true
				distinct++
			}
		}
		return distinct
	}

	var stats Stats
	stats.ShapePairs = admitOne(shapes.admitFor(probeIdx, sim, opt), func(a *admission) *bool { return &a.shape })
	stats.ConceptPairs = admitOne(concepts.admitFor(probeIdx, opt), func(a *admission) *bool { return &a.concept })
	stats.CallPairs = admitOne(calls.admitFor(probeIdx, opt), func(a *admission) *bool { return &a.call })
	stats.Union = len(admitted)
	stats.Suppressed = shapes.suppressed
	stats.LargeBuckets = shapes.largeBuckets
	stats.SurvivingPatterns = len(shapes.idf)

	cands := evaluate(admitted, shapes, concepts, calls, sim, opt, &stats)
	return cands, stats
}

// evaluate computes the definitive evidence for every admitted pair, in
// (AIdx, BIdx) order. Shared by Retrieve and Probe so the two cannot drift.
func evaluate(admitted map[pairKey]*admission, shapes *shapeIndex, concepts *conceptIndex,
	calls *callIndex, sim *simCache, opt Options, stats *Stats) []Candidate {

	keys := make([]pairKey, 0, len(admitted))
	for k := range admitted {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})

	cands := make([]Candidate, 0, len(keys))
	for _, k := range keys {
		a := admitted[k]
		shapeMass, trophic, chains := shapes.pairEvidence(k[0], k[1], opt.ChainTopN)
		callMass := calls.sharedMass(k[0], k[1])
		c := Candidate{
			AIdx:       k[0],
			BIdx:       k[1],
			Breakdown:  sim.get(k[0], k[1]),
			Shape:      shapeMass,
			Concept:    concepts.sharedMass(k[0], k[1]),
			Call:       callMass,
			TrophicSim: trophic,
			CallSim:    calls.callSim(k[0], k[1], callMass),
			Chains:     chains,
		}
		c.Total = c.Shape + c.Concept + c.Call
		if a.shape {
			c.Channels = append(c.Channels, ChannelShape)
		}
		if a.concept {
			c.Channels = append(c.Channels, ChannelConcept)
		}
		if a.call {
			c.Channels = append(c.Channels, ChannelCall)
		}
		if a.concept && !a.shape && !a.call {
			stats.OnlyConcept++
		}
		if a.call && !a.shape && !a.concept {
			stats.OnlyCall++
		}
		cands = append(cands, c)
	}
	return cands
}

// simCache memoizes exact fingerprint similarity per unordered pair, so the
// structural channel's probing and the union's definitive Breakdown never
// compute the same pair twice.
type simCache struct {
	units []parser.CodeUnit
	seen  map[pairKey]fingerprint.Breakdown
}

func newSimCache(units []parser.CodeUnit) *simCache {
	return &simCache{units: units, seen: make(map[pairKey]fingerprint.Breakdown)}
}

func (c *simCache) get(a, b int) fingerprint.Breakdown {
	k := orderPair(a, b)
	if bd, ok := c.seen[k]; ok {
		return bd
	}
	bd := fingerprint.Similarity(c.units[k[0]].Fingerprint, c.units[k[1]].Fingerprint)
	c.seen[k] = bd
	return bd
}

// topK sorts neighbor masses by (mass desc, index asc) and returns at most k
// indices. The index tie-break is what keeps equal-evidence retrieval
// byte-stable across runs.
type neighborMass struct {
	idx  int
	mass float64
}

func topK(neighbors []neighborMass, k int) []neighborMass {
	sort.Slice(neighbors, func(i, j int) bool {
		if neighbors[i].mass != neighbors[j].mass {
			return neighbors[i].mass > neighbors[j].mass
		}
		return neighbors[i].idx < neighbors[j].idx
	})
	if k > 0 && len(neighbors) > k {
		neighbors = neighbors[:k]
	}
	return neighbors
}

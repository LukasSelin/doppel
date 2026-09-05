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
	"math"
	"sort"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parallel"
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
	MaxLabelDF   int     // WL labels present in more units than this carry no evidence
	MaxCallDF    int     // call tokens present in more units than this carry no evidence
	MaxConceptDF int     // concept postings larger than this are skipped for enumeration
	ChainTopN    int     // shared-label explanations kept per pair; <0 unbounded, 0 none

	// MinIDF, when > 0, replaces the absolute label and call df caps with an
	// information floor in nats: a feature counts only if ln(N/df) >= MinIDF,
	// i.e. cap = floor(N·e^-MinIDF) with each channel's own N (shape-eligible
	// units for labels, all units for calls). A cap of 50 is 62% of conc's
	// functions and 0.6% of moby's; one floor means one thing everywhere. A
	// derived cap below 2 is not clamped up — it means nothing in that channel
	// both pairs and carries the floor, and Stats says so. 0 = absolute caps.
	MinIDF float64

	// Weights is the fingerprint blend the exact code-shape score uses. The
	// zero value means fingerprint.DefaultWeights — the production path never
	// sets it; the bench sensitivity sweep does, per run, with no global.
	Weights fingerprint.Weights
}

// weights resolves the blend, defaulting the zero value.
func (o Options) weights() fingerprint.Weights {
	if o.Weights == (fingerprint.Weights{}) {
		return fingerprint.DefaultWeights()
	}
	return o.Weights
}

// DefaultOptions returns the production defaults. ChannelK, Threshold and
// MinNodes mirror the --channel-k, --threshold and --min-nodes flag defaults;
// the caps are fixed constants chosen so that corpus-wide idioms (Error()
// shapes, fmt.Sprintf) drop out of the indexes entirely while genuinely
// shared machinery stays in.
//
// Threshold is 0.38, the median of the code-shape floors `--calibrate 0.01`
// derives across the public ladder (prometheus 0.33, moby and hugo 0.35, gin
// 0.41, cobra 0.44, chi 0.45; conc declines for want of null pairs). It was
// 0.60, which admitted far fewer than 1% of random pairs on every corpus
// measured — one number that meant a different strictness on each. The
// labeled corpus is flat across the whole 0.30..0.60 range, so this buys
// shape-channel recall at no measured labeled cost; see TestThresholdLadder.
//
// MinNodes was 12 while the shape channel indexed the pattern multiset, then
// 18 when it moved to WL labels. A body produces wlRounds+1 labels per node,
// so a *trivial* body that happens to be corpus-unique earns maximal-IDF
// evidence at the deep rounds — where the pattern hierarchy gave it nothing
// at all, a one-liner having no loop summary, no statement bigram and no
// def-use edge to offer. This gate is the only thing that ever suppressed
// those bodies.
//
// It is 16 because that is the lowest floor that still holds the pin 18 was
// set for. Cobra's `commandSorterByName.Less ↔ doc.byName.Less` false
// positive is 15 nodes a side and must stay out; conc's `ResultContextPool.
// Wait ↔ ResultErrorPool.Wait` clone family is 16 and should be let in. The
// two pins are one node apart, so 16 separates them exactly: 16, 17 and 18
// score identically on the cobra labels (merge 4.5, refactor 13.7, fp 47.0,
// no violations) while 15 and below admit the false positive at rank 20. See
// TestMinNodesLadder. 18 was closing the shape channel on small corpora for
// no labeled benefit — conc retrieved 3 shape candidates at 18 and 14 at 16.
func DefaultOptions() Options {
	return Options{
		ChannelK:     5,
		Threshold:    0.38,
		MinNodes:     16,
		MaxLabelDF:   50,
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
	Shape      float64               // shared structural energy, Σ IC·min(count) over shared WL labels
	Concept    float64               // shared tag information, Σ IC(LCS) over the best matching
	Call       float64               // shared rare-call IDF mass
	Total      float64               // Shape + Concept + Call, summed in that order
	TrophicSim float64               // 2·SharedEnergy/(E_A+E_B): weighted Dice over WL label energy
	CallSim    float64               // call-channel Dice: mutual fraction of informative call energy
	Channels   []string              // admission provenance, subset of {shape, concept, call}
	Chains     []SharedLabel         // highest-energy shared labels, the explanation
}

// Stats describes one retrieval run, for the stderr summary and evaluation.
type Stats struct {
	ShapePairs      int // distinct pairs admitted by the structural channel
	ConceptPairs    int // distinct pairs admitted by the concept channel
	CallPairs       int // distinct pairs admitted by the call channel
	Union           int // unique pairs across all channels
	OnlyConcept     int // pairs only the concept channel admitted
	OnlyCall        int // pairs only the call channel admitted
	Suppressed      int // shape-eligible units whose every WL label was df-capped out
	LargeBuckets    int // exact WL-bag identity buckets with > largeBucketSize members
	SurvivingLabels int // distinct WL labels carrying evidence

	LabelCap    int  // the df cap the shape channel used (derived or absolute)
	CallCap     int  // the df cap the call channel used
	CapsDerived bool // true when MinIDF derived the caps
}

// effectiveCap is the df cap a channel uses: the absolute one, or with a
// MinIDF floor the largest df still carrying that many nats over n — never
// clamped, so a floor no feature can meet reads as the empty channel it is.
func effectiveCap(absolute, n int, minIDF float64) (cap int, derived bool) {
	if minIDF <= 0 {
		return absolute, false
	}
	return int(math.Floor(float64(n) * math.Exp(-minIDF))), true
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
	onto *ontology.Ontology, ic *ontology.IC, wl *fingerprint.LabelIDF,
	opt Options) ([]Candidate, Stats) {

	scorer := ontology.NewScorer(onto, ic)
	sim := newSimCache(units, wl, opt.weights())

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
	stats.SurvivingLabels = len(shapes.idf)
	stats.LabelCap, stats.CallCap, stats.CapsDerived = shapes.cap, calls.cap, opt.MinIDF > 0

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
	onto *ontology.Ontology, ic *ontology.IC, wl *fingerprint.LabelIDF,
	opt Options) ([]Candidate, Stats) {

	scorer := ontology.NewScorer(onto, ic)
	sim := newSimCache(units, wl, opt.weights())

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
	stats.LabelCap, stats.CallCap, stats.CapsDerived = shapes.cap, calls.cap, opt.MinIDF > 0
	stats.Union = len(admitted)
	stats.Suppressed = shapes.suppressed
	stats.LargeBuckets = shapes.largeBuckets
	stats.SurvivingLabels = len(shapes.idf)

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

	// One candidate per key, filled by index: every quantity below is a pure
	// function of the pair and the (now immutable) indexes, and the two memos
	// are concurrency-safe. The channel flags and the two stats counters are
	// folded in afterwards, sequentially, so the counters cannot depend on who
	// finished first — the same compute-then-merge split buildArenas makes.
	cands := make([]Candidate, len(keys))
	parallel.Blocks(len(keys), evaluateBlock, minPairsPerEvaluateWorker, func(i int) {
		k := keys[i]
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
		cands[i] = c
	})

	for i, k := range keys {
		a := admitted[k]
		if a.shape {
			cands[i].Channels = append(cands[i].Channels, ChannelShape)
		}
		if a.concept {
			cands[i].Channels = append(cands[i].Channels, ChannelConcept)
		}
		if a.call {
			cands[i].Channels = append(cands[i].Channels, ChannelCall)
		}
		if a.concept && !a.shape && !a.call {
			stats.OnlyConcept++
		}
		if a.call && !a.shape && !a.concept {
			stats.OnlyCall++
		}
	}
	return cands
}

// evaluateBlock and minPairsPerEvaluateWorker mirror the comparison stage's
// two knobs: one union pair costs about what one comparison does.
const (
	evaluateBlock             = 64
	minPairsPerEvaluateWorker = 512
)

// simCache memoizes exact fingerprint similarity per unordered pair, so the
// structural channel's probing and the union's definitive Breakdown never
// compute the same pair twice.
type simCache struct {
	units   []parser.CodeUnit
	wl      *fingerprint.LabelIDF
	weights fingerprint.Weights
	seen    *pairMemo[fingerprint.Breakdown]
}

func newSimCache(units []parser.CodeUnit, wl *fingerprint.LabelIDF, w fingerprint.Weights) *simCache {
	return &simCache{units: units, wl: wl, weights: w, seen: newPairMemo[fingerprint.Breakdown]()}
}

func (c *simCache) get(a, b int) fingerprint.Breakdown {
	k := orderPair(a, b)
	return c.seen.get(k, func() fingerprint.Breakdown {
		return fingerprint.SimilarityWith(c.units[k[0]].Fingerprint, c.units[k[1]].Fingerprint, c.wl, c.weights)
	})
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

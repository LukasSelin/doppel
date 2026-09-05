// Package lexicon learns the corpus's own concept vocabulary instead of
// asserting one.
//
// The tagger it replaces was fourteen hand-written rules over string channels,
// and every term in it — "httpClient", "SELECT ", "gobreaker" — was a guess
// about how *some* codebase writes something. It did not scale: not to more
// concepts, not to a repository that names its database wrapper "store", and
// certainly not to another language, where every rule would have to be written
// again by hand.
//
// Everything else in doppel is already corpus-relative: information content,
// role thresholds, culture prototypes, habitat temperature, the PMI ecology.
// This package makes concepts behave the same way. The rules survive as
// *seeds* — a starting member set, not an answer — and the corpus supplies the
// vocabulary that actually distinguishes those members from everything else.
// Features no seed claims are clustered into emergent concepts on their own
// co-occurrence, so a corpus with no seeds at all still produces a lexicon.
//
// Membership is a confidence, not a boolean. A function either carries enough
// of a concept's learned vocabulary to be one of its members or it does not,
// and how much is exactly the quantity the old rule table threw away the
// moment any one of its channels matched.
//
// Determinism, as everywhere in doppel: integer counting, sorted iteration, no
// persisted state, no randomness. An unchanged tree yields an identical lexicon.
package lexicon

import (
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parallel"
	"github.com/LukasSelin/doppel/internal/parser"
)

// FloorRule selects where a concept's membership bar comes from. The zero
// value is the shipped rule; the others are measurement seams, Options-only and
// never reached by cmd, in the same shape retriever.Options.MinIDF has — and
// with the same verdict recorded beside them in CLAUDE.md.
type FloorRule int

const (
	// FloorFounding is the shipped rule: the FloorQuantile of the concept's own
	// founding members' coverage.
	//
	// It is open to an obvious objection — founding members are by construction
	// the units that already carry the concept, so a quantile of their coverage
	// asks the elite where the entry line is, and one quantile over each
	// concept's own sample leaves the floors spread 5-6x from p10 to p90 across
	// the ladder. The alternatives below are that objection, made concrete and
	// measured. None of them was adopted; see floors.
	FloorFounding FloorRule = iota

	// FloorRelMax anchors the bar to the concept's own best-covered unit: a
	// member must be covered at least RelMaxFraction as much as the unit this
	// concept explains the most of. The only measured rule that narrows the
	// floor spread (to 1.7-2.2x) rather than widening it.
	FloorRelMax

	// FloorTouched is the upper quantile, at TouchedQuantile, of the coverage
	// of every unit the concept reaches at all.
	FloorTouched
)

// Options tunes the learner. Like culture's, the cutoffs prefer silence over
// noise: a feature too rare to recur or too common to discriminate is not
// evidence, and a concept whose vocabulary does not separate its members from
// the corpus is not a concept.
type Options struct {
	// MinDF is the smallest document frequency a feature may have. Two: a
	// feature in one function can never relate two functions.
	MinDF int

	// MinIDF is the information floor in nats, and replaces an absolute df cap
	// for the reason retriever.Options.MinIDF documents — a cap of 50 is 1.5
	// nats on a small corpus and 5 on a large one, so the same number means
	// different things. The derived cap is floor(N·e^-MinIDF).
	MinIDF float64

	// MinSupport is the co-occurrence count floor for a concept~feature
	// association, and for an edge in the emergent feature graph.
	MinSupport int

	// MinLift is the pointwise mutual information floor, in nats, above which a
	// feature is characteristic of a concept rather than merely present in it.
	// ln 2 — the same "twice chance" bar culture's ecology uses.
	MinLift float64

	// MinMembers is the smallest membership an emergent concept may have.
	MinMembers int

	// FloorQuantile places each concept's membership bar inside its own
	// founding coverage distribution: the quantile of founding-member coverage
	// a unit must reach to be a member. 0.25 admits the founding set bar its
	// weakest quarter, plus everyone else the concept explains as much of.
	//
	// A fixed confidence cut was the first design and it was wrong. Confidence
	// saturates around the *median* founding member, so "confidence >= 0.5"
	// means "at least the median founding member" — which throws away half of
	// every concept's own seed set by construction, and did: on this repo it
	// left 527 of 546 functions with no concept at all.
	//
	// Lowering this quantile is not the repair for a corpus that is largely
	// unlabelled, and measuring it is what moved the bar into coverage: with an
	// evidence floor, 0.25 left 451 of 866 functions unlabelled here and 0.15
	// let one concept swallow 649 of them. The bar was not too high, it was
	// stated in a quantity that is not comparable between two functions.
	//
	// It is read only under FloorRule FloorFounding now. The shipped rule draws
	// the bar from the corpus instead — see floors — because a quantile of the
	// founders asks the elite where the entry line is, and because one quantile
	// over each concept's own sample still produced floors spread 5-6x from p10
	// to p90 across the ladder.
	FloorQuantile float64

	// FloorRule selects where the membership bar comes from. Zero value
	// FloorKnee is what production runs.
	FloorRule FloorRule

	// TouchedQuantile is the quantile FloorTouched applies to a concept's reach,
	// and that FloorCliff falls back to for a curve with no cliff in it.
	TouchedQuantile float64

	// RelMaxFraction is FloorRelMax's anchor.
	RelMaxFraction float64

	// BackfillN gives a unit its best N concepts whatever their floors say,
	// at whatever confidence it earned. 0, the default, is off.
	//
	// The case for it: membership is a hard cut in a system whose every other
	// quantity is graded, and a unit clearing no floor gets nothing at all —
	// not because no concept describes it, but because none describes it
	// *enough*. A low-confidence membership is cheap to a consumer that
	// weights it; a missing one is invisible to every consumer there is.
	//
	// The case against it is which consumers those are. Five read a membership
	// as a bare boolean — see parser.ConceptIDs, which lists them — and to
	// those, a membership admitted because the unit had nothing else is not
	// cheap at all. BackfillVisible is that question, as a knob.
	BackfillN int

	// BackfillAlways applies BackfillN to every unit rather than only to one
	// that would otherwise carry no concept at all.
	BackfillAlways bool

	// BackfillVisible emits backfilled memberships unmarked, so the five
	// boolean consumers see them as ordinary. Off, they are marked BelowFloor
	// and only the consumers that weight a membership see them.
	BackfillVisible bool

	// MaxMemberships bounds how many concepts one unit may belong to, keeping
	// its strongest by coverage. 0 is unbounded, which is not a working
	// setting: coverage removes the size term from the membership bar and with
	// it the accidental ceiling that bar was providing, so the tail runs away
	// on a wide corpus. Measured unbounded across the pinned ladder, hugo
	// assigns 7.4 concepts per function and grows a 1 268-member concept — 22%
	// of the corpus, which is not a practice — while moby reaches 4.8 per
	// function and gin lets one concept claim 37%.
	//
	// Three, and the value is the same bounded-per-item idiom the retrieval
	// channels apply to their postings: recall is per item. Every K measured
	// (2, 3, 4, 6) fixes the runaway tail and none of them moves a single
	// labeled pair on cobra — merge 5.3 (6/6), refactor 12.8, fp 50.5, no
	// violations, identically — so the labels do not decide it and the corpus
	// statistics do: at 3, hugo's largest concept is 4.5% of the corpus and its
	// assignments 2.4 per function. Below 3 a function may not do more than a
	// couple of things, which MaxOverlap already assumes it does.
	MaxMemberships int

	// EdgeK bounds the emergent feature graph: each feature keeps its EdgeK
	// strongest associations. Without it the graph is not sparse — features
	// co-occur far more freely than functions resemble each other — and the
	// whole vocabulary collapses into one component that trips MaxComponent
	// and produces no concepts at all. Measured: the first version of this
	// package emitted a single emergent concept on doppel's own corpus, and
	// none with an empty seed set.
	EdgeK int

	// MaxEmergentFeatures bounds the emergent pass to the most frequent
	// surviving features, and MaxUnitFeatures bounds one unit's contribution to
	// the co-occurrence count. Both are reported in Stats rather than applied
	// silently: feature co-occurrence is quadratic in a unit's feature count,
	// and a large corpus has enough of both to matter.
	MaxEmergentFeatures int
	MaxUnitFeatures     int

	// MinCliqueSize is the smallest feature clique that may seed an emergent
	// concept.
	//
	// Two, and the choice was measured rather than assumed. Three sounds
	// safer — "a pair is not a practice" — but the practices worth finding are
	// overwhelmingly pairs of calls: Get/Decode, Marshal/Unmarshal,
	// Open/Close. At three, a store wrapper reached through exactly two calls
	// is invisible, which is the case this package exists for. The strictness
	// is elsewhere and it is real: an edge needs MinSupport co-occurrences at
	// MinLift nats, and foundingMembers requires a unit to carry two of the
	// clique's features, so for a pair that means both.
	MinCliqueSize int

	// MaxOverlap is the founding-member Jaccard at which a new emergent
	// concept is judged to be one already found. Cliques overlap heavily —
	// one function does several things — so without this the same group of
	// functions is reported under a dozen slightly different names.
	MaxOverlap float64

	// MaxSearch guards the clique enumeration, as family's does. Enumeration
	// runs over one feature's neighbourhood — at most EdgeK+1 vertices — so
	// this is a backstop against a pathology rather than a working limit; a
	// tripped guard records the feature in Stats.Skipped and emits nothing for
	// it, never a partial enumeration presented as the answer.
	MaxSearch int
}

// DefaultOptions returns the production settings.
func DefaultOptions() Options {
	return Options{
		MinDF:               2,
		MinIDF:              1.0,
		MinSupport:          3,
		MinLift:             math.Ln2,
		MinMembers:          5,
		FloorQuantile:       0.25,
		TouchedQuantile:     0.25,
		RelMaxFraction:      0.5,
		EdgeK:               8,
		MaxEmergentFeatures: 2000,
		MaxUnitFeatures:     64,
		MaxMemberships:      3,
		MinCliqueSize:       2,
		MaxOverlap:          0.6,
		MaxSearch:           200000,
	}
}

// Feature is one weighted term of a concept's learned vocabulary.
type Feature struct {
	Name   string  // channel-prefixed, e.g. "sel:sql.Open"
	Weight float64 // lift × idf: distinctive to the concept and rare in the corpus
	DF     int     // units carrying the feature, corpus-wide
	Count  int     // units carrying it among the concept's founding members
}

// Concept is one learned concept: an identity derived from its own vocabulary,
// the vocabulary itself, and the evidence scale that turns a unit's evidence
// into a confidence.
type Concept struct {
	ID       string    // derived name, e.g. "sql.Open+QueryRow"
	Seed     string    // the seed tag it grew from; "" when emergent
	Anchor   string    // for an emergent concept: the seed of the concept it most resembles
	Features []Feature // sorted by (Weight desc, Name asc)
	Scale    float64   // median founding-member coverage: the confidence half-point
	Floor    float64   // coverage a unit must reach to be a member at all
	Members  int       // units at or above Floor
}

// Stats summarizes one build for the stderr diagnostic line.
type Stats struct {
	Units             int
	FeaturesTotal     int // distinct features before the window
	FeaturesSurviving int // features inside the df window
	FeatureCap        int // the df cap MinIDF derived
	Seeded            int // seeded concepts that kept a vocabulary
	SeedsDropped      int // seeds whose members shared no distinctive feature
	Emergent          int // concepts from the unclaimed-feature cliques
	EmergentDropped   int // cliques that failed the membership floor
	FloorDropped      int // concepts whose own founders could not clear the corpus bar
	Skipped           int // feature neighbourhoods abandoned by the search guard
	Edges             int // surviving edges in the emergent feature graph
	Assignments       int // (unit, concept) memberships
	Untagged          int // units carrying no concept at all
}

// Model is the built lexicon: the concepts, and each unit's memberships.
type Model struct {
	concepts []Concept
	assign   [][]parser.Concept
	unused   []string // seed labels a concept grew from
	stats    Stats
}

// Concepts returns the learned concepts, sorted by ID.
func (m *Model) Concepts() []Concept { return m.concepts }

// Assignments returns each unit's memberships, ascending by concept ID. The
// slice is positional: index i describes the unit at index i of the slice given
// to Build, the same convention docs[i]/units[i] uses everywhere else.
func (m *Model) Assignments() [][]parser.Concept { return m.assign }

// GrownSeeds are the seed labels that produced a concept, sorted.
//
// A seed fires on functions; a concept needs those functions to share a way of
// being written. A seed that grew nothing — because it fired on nothing, or
// because what it fired on has no common shape here — is a finding about the
// corpus rather than a failure, and it is what still answers "does this
// repository already do X" now that concept names are the corpus's own and no
// longer a fixed checklist. The caller holds the seed vocabulary and subtracts.
func (m *Model) GrownSeeds() []string { return m.unused }

// Stats returns the build summary.
func (m *Model) Stats() Stats { return m.stats }

// Get returns a concept by ID.
func (m *Model) Get(id string) (Concept, bool) {
	i := sort.Search(len(m.concepts), func(i int) bool { return m.concepts[i].ID >= id })
	if i < len(m.concepts) && m.concepts[i].ID == id {
		return m.concepts[i], true
	}
	return Concept{}, false
}

// corpus is the intermediate the stages share: the feature matrix, its document
// frequencies, and the surviving vocabulary.
type corpus struct {
	n         int
	features  [][]string         // per unit, sorted
	df        map[string]int     // feature → units carrying it
	idf       map[string]float64 // surviving features only
	surviving []string           // sorted
	cap       int
	mass      []float64 // per unit, Σ idf over its surviving features
}

// Build learns the lexicon. seeds[i] is the seed labels the rule tagger fired
// on unit i — the founding member sets — and may be nil throughout, in which
// case every concept comes from the emergent path. g is the resolved call
// graph, used for the call-token channel.
func Build(units []parser.CodeUnit, g *concepter.Graph, seeds [][]string, opt Options) *Model {
	c := buildCorpus(units, g, opt)

	m := &Model{stats: Stats{
		Units:             c.n,
		FeaturesTotal:     len(c.df),
		FeaturesSurviving: len(c.surviving),
		FeatureCap:        c.cap,
	}}

	claimed := make(map[string]bool)
	concepts, founders := expandSeeds(c, seeds, claimed, &m.stats, opt)
	emerged, emergedFounders := emergeConcepts(c, claimed, &m.stats, opt)
	concepts = append(concepts, emerged...)
	founders = append(founders, emergedFounders...)

	// The membership bar comes from the corpus, so it can only be derived once
	// every concept's vocabulary exists — and a concept whose own founders
	// cannot clear a corpus-relative bar was not distinctive enough to be one,
	// which is the same finding SeedsDropped reports one stage earlier. The
	// drop runs before naming and sorting so the parallel founder slice never
	// has to be permuted alongside a reorder.
	if opt.FloorRule != FloorFounding {
		floors(c, concepts, opt)
		concepts = dropUnfounded(c, concepts, founders, &m.stats, opt)
	}

	// grownSeeds is counted here rather than before the emergent pass, because
	// a seeded concept can now die at the floor: reporting it as grown would
	// put a practice this corpus does not have into Result.UnusedSeeds, the
	// report overview and the session-start digest — the one surface that
	// answers "does this repository already do X".
	m.unused = grownSeeds(concepts)

	anchorEmergent(concepts)
	nameConcepts(concepts)
	sort.Slice(concepts, func(i, j int) bool { return concepts[i].ID < concepts[j].ID })

	m.concepts = concepts
	m.assign = assign(c, concepts, opt)
	for i := range m.assign {
		m.stats.Assignments += len(m.assign[i])
		if len(m.assign[i]) == 0 {
			m.stats.Untagged++
		}
	}
	// Members is read back off the assignment so it is the real count rather
	// than a remembered one — the discipline family's describe follows.
	byID := make(map[string]int, len(m.concepts))
	for i := range m.concepts {
		byID[m.concepts[i].ID] = i
	}
	for i := range m.assign {
		for _, a := range m.assign[i] {
			m.concepts[byID[a.ID]].Members++
		}
	}
	return m
}

// buildCorpus extracts every unit's features, counts document frequencies, and
// applies the information window. Features below MinDF cannot pair anything;
// features above the derived cap are corpus idiom and carry no evidence — the
// same two-sided window the retrieval channels use, for the same reason.
func buildCorpus(units []parser.CodeUnit, g *concepter.Graph, opt Options) *corpus {
	internal := concepter.QualifiedNames(units)
	c := &corpus{
		n:        len(units),
		features: make([][]string, len(units)),
		df:       make(map[string]int),
		idf:      make(map[string]float64),
	}
	// Feature extraction is per unit and pure; the df tally is one shared map,
	// so it stays sequential. Splitting them costs one extra pass over the
	// feature slices and takes the expensive half across cores.
	parallel.Blocks(len(units), featureBlock, minUnitsPerFeatureWorker, func(i int) {
		c.features[i] = unitFeatures(units[i], g, internal)
	})
	for i := range units {
		for _, f := range c.features[i] {
			c.df[f]++
		}
	}

	c.cap = c.n
	if opt.MinIDF > 0 {
		c.cap = int(math.Floor(float64(c.n) * math.Exp(-opt.MinIDF)))
	}
	minDF := opt.MinDF
	if minDF < 2 {
		minDF = 2
	}
	for _, f := range sortedKeys(c.df) {
		d := c.df[f]
		if d < minDF || d > c.cap {
			continue
		}
		c.surviving = append(c.surviving, f)
		c.idf[f] = math.Log(float64(c.n) / float64(d))
	}

	// Every unit's own information, in the same currency a concept's evidence
	// is paid in. It can only be summed once the window is settled, because a
	// feature outside it has no idf and contributes to neither side — the same
	// "surviving features only" rule fit counts under.
	c.mass = make([]float64, c.n)
	parallel.Blocks(c.n, featureBlock, minUnitsPerFeatureWorker, func(i int) {
		for _, f := range c.features[i] { // ascending: addition order is fixed
			c.mass[i] += c.idf[f]
		}
	})
	return c
}

// evidence sums a concept's feature weights over the features a unit carries.
func (c *corpus) evidence(idx int, weights map[string]float64) float64 {
	if len(weights) == 0 {
		return 0
	}
	sum := 0.0
	for _, f := range c.features[idx] { // ascending: addition order is fixed
		sum += weights[f]
	}
	return sum
}

// cover is evidence as a fraction of the unit's own information: how much of
// what this function carries the concept explains.
//
// It is what membership is decided on, and the reason is that a bare sum is
// not comparable between two functions. Evidence scales with how many features
// a unit has, so an absolute bar is largely a bar on size: measured on doppel's
// own corpus the labelled units carried a median of 48 features and the
// unlabelled ones 21, and 447 of the 451 unlabelled functions carried real
// evidence for some concept and simply could not reach its floor. Dividing by
// the unit's mass removes the length term from both sides — a five-line
// function that does one thing is fully explained by one concept, and a long
// one is not merely by being long.
//
// Zero mass means the unit carries nothing inside the information window, so
// no concept explains any of it: the honest cover is 0, not an infinity.
func (c *corpus) cover(idx int, weights map[string]float64) float64 {
	if c.mass[idx] <= 0 {
		return 0
	}
	return c.evidence(idx, weights) / c.mass[idx]
}

// weightsOf indexes a concept's vocabulary for evidence lookups.
func weightsOf(features []Feature) map[string]float64 {
	w := make(map[string]float64, len(features))
	for _, f := range features {
		w[f.Name] = f.Weight
	}
	return w
}

// forEachCoverage walks every unit's coverage of every concept whose vocabulary
// it touches at all, in ascending unit order and ascending concept order.
//
// It is one loop because there are two callers — floors derives each concept's
// membership bar from the corpus's own distribution, assign then applies it —
// and two hand-maintained copies of an accumulation whose *order* is the
// determinism guarantee is exactly the clone this tool exists to find.
//
// Inverted, not nested. The direct form — every unit against every concept's
// vocabulary — is units × concepts × features, and on prometheus (5.5k
// functions, ~380 concepts) that alone cost most of a minute. Walking each
// unit's own features into the concepts that use them is the same arithmetic in
// the order the data is sparse in.
//
// The callback receives the concept indices the unit touched and their
// coverages, positionally aligned; both slices are scratch and must not be
// retained.
// fn may be called from several goroutines at once, and the slices it is handed
// are the calling worker's scratch — valid for the call and reused after it. A
// callback must therefore confine its writes to unit, or serialise them itself;
// floors and dropUnfounded, which accumulate per concept rather than per unit,
// take the second route (they run only under a non-default FloorRule, so the
// lock is never on a hot path).
func forEachCoverage(c *corpus, concepts []Concept, fn func(unit int, touched []int, cover []float64)) {
	type weighted struct {
		concept int
		weight  float64
	}
	byFeature := make(map[string][]weighted)
	for j := range concepts {
		for _, f := range concepts[j].Features {
			byFeature[f.Name] = append(byFeature[f.Name], weighted{concept: j, weight: f.Weight})
		}
	}

	// The three scratch buffers were one set reused down the loop; they become
	// one set per worker. byFeature is read-only from here on.
	type scratch struct {
		evidence []float64
		touched  []int
		cover    []float64
	}
	parallel.BlocksWith(c.n, coverageBlock, minUnitsPerCoverageWorker,
		func() *scratch { return &scratch{evidence: make([]float64, len(concepts))} },
		func(s *scratch, i int) {
			s.touched, s.cover = s.touched[:0], s.cover[:0]
			for _, f := range c.features[i] { // ascending: the addition order is fixed
				for _, w := range byFeature[f] {
					if s.evidence[w.concept] == 0 {
						s.touched = append(s.touched, w.concept)
					}
					s.evidence[w.concept] += w.weight
				}
			}
			sort.Ints(s.touched) // concepts are sorted by ID, so this is ID order
			for _, j := range s.touched {
				e := s.evidence[j]
				s.evidence[j] = 0 // reset only what was touched
				cov := 0.0
				if c.mass[i] > 0 {
					cov = e / c.mass[i] // one division at the end: accumulation order is untouched
				}
				s.cover = append(s.cover, cov)
			}
			fn(i, s.touched, s.cover)
		})
}

// floors sets every concept's membership bar from the corpus's own distribution
// of coverage for it, rather than from its founding members'. It runs only
// under a non-default FloorRule: **this was measured and not adopted**, and the
// seam is kept so the question is re-runnable rather than re-argued.
//
// The objection it answers is real. Founding members are by construction the
// units that already carry the concept, so a quantile of *their* coverage asks
// the elite where the entry line is; and because each concept's founders are a
// different sample, one FloorQuantile produces a different kind of number per
// concept — measured across the pinned ladder, the founding floors spread 5-6x
// from p10 to p90 (moby 0.44 to 2.27). Deriving the bar from the population
// being judged is what internal/calibrate does for a threshold, and it is the
// obvious repair.
//
// What the measurement found is that the reached population is the wrong
// reference, because it is dominated by units carrying a trace of the
// vocabulary. Any rank-based bar over it lands at 2-5% coverage, and membership
// stops meaning anything: FloorTouched at every quantile tried takes moby and
// hugo to **0.0% unlabelled** with every function saturating MaxMemberships,
// mean confidence falling 0.49 -> 0.40, and the largest concept growing (cobra
// 19.3% -> 33.5% of the corpus, chi 15.3% -> 23.5%). A knee — largest relative
// drop in the curve — was tried first and is worse in the goal's own terms: it
// widens the floor spread to 700x, because a curve with no cliff still has a
// largest step.
//
// FloorRelMax is the one that works as intended. Anchored to the concept's own
// best-covered unit at 0.25 it narrows the spread to 1.7-2.2x — the only rule
// that moves the number this change exists to move — at neutral coverage (moby
// 9.4% -> 8.8% unlabelled) and neutral labels (cobra merge 5.3 -> 5.2, refactor
// 12.8 -> 13.1, fp 50.5 -> 51.0, no violations). It was still not adopted: it
// drops ~30% of the learned vocabulary on the large corpora (moby 519 -> 365
// concepts, whose founders cannot clear a bar set by one dominant member) and
// grows the largest concept where the founding rule shrank it (moby 9.0% ->
// 12.3%, hugo 4.5% -> 7.9%). One labeled corpus cannot certify that trade, and
// a comparable floor is a property rather than a result — nothing downstream
// was shown to get better for having one.
//
// Zeros are not part of the distribution. A unit touching none of a concept's
// vocabulary covers none of it, which is a statement about the unit and not a
// point on the curve the concept's own members lie on.
func floors(c *corpus, concepts []Concept, opt Options) {
	if len(concepts) == 0 {
		return
	}
	curves := make([][]float64, len(concepts))
	var mu sync.Mutex // curves accumulates per concept, not per unit
	forEachCoverage(c, concepts, func(_ int, touched []int, cover []float64) {
		mu.Lock()
		for k, j := range touched {
			if cover[k] > 0 {
				curves[j] = append(curves[j], cover[k])
			}
		}
		mu.Unlock()
	})
	for j := range concepts {
		sort.Float64s(curves[j])
		concepts[j].Floor = floorOf(curves[j], opt)
	}
}

// dropUnfounded removes concepts whose own founding members cannot clear the
// corpus-derived bar, in the number MinMembers asks of an emergent concept.
//
// founders is positional with concepts. The result keeps concepts' order, so
// nothing downstream has to know a drop happened.
func dropUnfounded(c *corpus, concepts []Concept, founders [][]int, stats *Stats, opt Options) []Concept {
	kept := retainedFounders(c, concepts, founders)
	out := concepts[:0]
	for j := range concepts {
		if kept[j] < opt.MinMembers {
			stats.FloorDropped++
			continue
		}
		out = append(out, concepts[j])
	}
	return out
}

// retainedFounders counts, per concept, how many of its founding members clear
// the floor now that the floor comes from the corpus.
//
// The lookup is inverted for the same reason every other pass here is: a
// per-unit list of the concepts it founded is the sparse direction, where a
// concepts × units membership table is not. A founder always carries at least
// two of its concept's clique features, so it is always among the unit's
// touched concepts and the binary search always finds it.
func retainedFounders(c *corpus, concepts []Concept, founders [][]int) []int {
	founded := make([][]int, c.n)
	for j := range founders {
		for _, u := range founders[j] {
			founded[u] = append(founded[u], j)
		}
	}
	kept := make([]int, len(concepts))
	var mu sync.Mutex // kept counts per concept, not per unit
	forEachCoverage(c, concepts, func(i int, touched []int, cover []float64) {
		mu.Lock()
		for _, j := range founded[i] {
			k := sort.SearchInts(touched, j)
			if k < len(touched) && touched[k] == j && cover[k] >= concepts[j].Floor {
				kept[j]++
			}
		}
		mu.Unlock()
	})
	return kept
}

// floorOf turns one concept's coverage curve into its membership bar under a
// corpus-derived rule. v must be ascending; it is not modified.
func floorOf(v []float64, opt Options) float64 {
	if len(v) == 0 {
		// The concept's vocabulary reaches nobody. Any positive bar is
		// equivalent; the smallest keeps the field non-zero, which every reader
		// of Floor already assumes.
		return math.SmallestNonzeroFloat64
	}
	if opt.FloorRule == FloorRelMax {
		return v[len(v)-1] * opt.RelMaxFraction
	}
	return upperQuantile(v, opt.TouchedQuantile)
}

// assign computes every unit's memberships. Two corpus-derived quantities do
// two different jobs, which is what keeps either from having to do both badly,
// and both are stated in *coverage* — the fraction of a unit's own information
// a concept explains — rather than in raw evidence. See corpus.cover for why.
//
// Floor decides membership, and floors derives it from the corpus's own
// distribution of coverage for that concept rather than from the concept's
// founders.
//
// Scale decides what the confidence reads: conf = C/(C+Scale) with Scale the
// median founding coverage, so a unit the concept explains as much of as it
// explains of its typical member reads about 0.5, and one it explains far more
// of approaches 1. Founding-derived on purpose, unlike Floor: Scale is the
// *grading* reference, and a concept's founders are exactly its typical
// members. Saturating rather than normalized, because coverage has no natural
// maximum either — a concept's weights are lift×idf, not idf, so a unit can be
// explained past its own mass — and pretending it had one would make the number
// a rank in disguise.
func assign(c *corpus, concepts []Concept, opt Options) [][]parser.Concept {
	out := make([][]parser.Concept, c.n)
	forEachCoverage(c, concepts, func(i int, touched []int, cover []float64) {
		var got []parser.Concept
		var covers []float64
		for k, j := range touched {
			cov := cover[k]
			if cov < concepts[j].Floor {
				continue
			}
			got = append(got, parser.Concept{
				ID:         concepts[j].ID,
				Confidence: cov / (cov + concepts[j].Scale),
			})
			covers = append(covers, cov)
		}
		got, covers = backfill(got, covers, concepts, touched, cover, opt)
		out[i] = topMemberships(got, covers, opt.MaxMemberships)
	})
	return out
}

// backfill tops a unit's memberships up to Options.BackfillN from the concepts
// it touched but did not clear, strongest coverage first.
//
// It appends rather than competes: topMemberships ranks on coverage and a
// backfilled membership is by construction below its concept's floor, so it can
// only occupy space a floor-cleared membership did not want. The result is
// re-sorted by ID, which is the order every consumer of Concepts expects.
//
// Ties on coverage go to the lower concept index, which is ID order — the same
// rule topMemberships uses, so the two cannot disagree about which of two
// equals is stronger.
func backfill(got []parser.Concept, covers []float64, concepts []Concept,
	touched []int, cover []float64, opt Options) ([]parser.Concept, []float64) {

	if opt.BackfillN <= 0 {
		return got, covers
	}
	if len(got) >= opt.BackfillN {
		return got, covers
	}
	if len(got) > 0 && !opt.BackfillAlways {
		return got, covers
	}

	// The below-floor candidates, strongest first. touched is ascending, so a
	// stable sort leaves ties in ID order.
	type cand struct {
		j   int
		cov float64
	}
	var below []cand
	for k, j := range touched {
		if cover[k] < concepts[j].Floor {
			below = append(below, cand{j: j, cov: cover[k]})
		}
	}
	if len(below) == 0 {
		return got, covers
	}
	sort.SliceStable(below, func(a, b int) bool { return below[a].cov > below[b].cov })

	want := opt.BackfillN - len(got)
	if want > len(below) {
		want = len(below)
	}
	for _, cd := range below[:want] {
		got = append(got, parser.Concept{
			ID:         concepts[cd.j].ID,
			Confidence: cd.cov / (cd.cov + concepts[cd.j].Scale),
			BelowFloor: !opt.BackfillVisible,
		})
		covers = append(covers, cd.cov)
	}
	// Back to ascending ID. The two slices are positional, so they sort as one.
	idx := make([]int, len(got))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return got[idx[a]].ID < got[idx[b]].ID })
	sortedGot := make([]parser.Concept, len(got))
	sortedCovers := make([]float64, len(covers))
	for i, j := range idx {
		sortedGot[i], sortedCovers[i] = got[j], covers[j]
	}
	return sortedGot, sortedCovers
}

// topMemberships keeps a unit's k strongest memberships by coverage, the same
// bounded-per-item rule the retrieval channels apply to their postings — a
// function does several things, not several dozen, and an unbounded assignment
// lets one broad concept claim a third of a corpus.
//
// Ties go to the lower concept ID, which is the order got already carries, so
// the selection is stable without a second key. k <= 0 keeps everything.
func topMemberships(got []parser.Concept, covers []float64, k int) []parser.Concept {
	if k <= 0 || len(got) <= k {
		return got
	}
	idx := make([]int, len(got))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return covers[idx[a]] > covers[idx[b]] })
	keep := append([]int(nil), idx[:k]...)
	sort.Ints(keep) // back to ascending concept ID, the order every consumer expects
	out := make([]parser.Concept, k)
	for i, j := range keep {
		out[i] = got[j]
	}
	return out
}

// median returns the median of a sorted slice.
func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	mid := len(v) / 2
	if len(v)%2 == 1 {
		return v[mid]
	}
	return (v[mid-1] + v[mid]) / 2
}

// upperQuantile is the nearest-rank upper quantile of an ascending slice: the
// value at rank ceil(q·n). It is the mirror of quantile below, and it is
// deliberately not calibrate.Quantile, which is the same arithmetic — that
// package imports comparator, so depending on it from here would run the
// dependency the wrong way through a package that has to finish before
// retrieval starts. Two directions of one idea in two packages that cannot see
// each other is not the kind of clone the shared helpers list exists to
// prevent.
func upperQuantile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	r := int(math.Ceil(q * float64(len(v))))
	if r < 1 {
		r = 1
	}
	if r > len(v) {
		r = len(v)
	}
	return v[r-1]
}

// quantile is the nearest-rank lower quantile of a sorted slice: a value some
// member actually had, never an interpolation between two. Same convention
// internal/calibrate uses for its thresholds, and for the same reason — the
// number is a bar real functions are measured against, so it should be a
// number a real function produced.
func quantile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	if q <= 0 {
		return v[0]
	}
	i := int(math.Ceil(q*float64(len(v)))) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(v) {
		i = len(v) - 1
	}
	return v[i]
}

// sortedKeys drains a map in ascending key order — the package-wide determinism
// idiom.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// anchorEmergent records, for each emergent concept, the seed of the seeded
// concept it shares the most vocabulary with. It is the only thing the
// taxonomy can be told about a concept nobody named: where in the existing
// hierarchy it most nearly belongs.
//
// Overlap is the shared vocabulary's weight, counted at the weaker side —
// the same "you share only as much as the lesser claim" rule the graded
// scorer uses. No overlap leaves the anchor empty, and the concept hangs from
// the root, which is the honest placement for something the vocabulary has no
// word for.
func anchorEmergent(concepts []Concept) {
	var seeded []int
	for i := range concepts {
		if concepts[i].Seed != "" {
			seeded = append(seeded, i)
		}
	}
	if len(seeded) == 0 {
		return
	}
	weights := make(map[int]map[string]float64, len(seeded))
	for _, i := range seeded {
		weights[i] = weightsOf(concepts[i].Features)
	}
	for i := range concepts {
		if concepts[i].Seed != "" {
			continue
		}
		bestSeed, best := "", 0.0
		for _, j := range seeded {
			var shared float64
			for _, f := range concepts[i].Features { // weight desc, name asc: fixed order
				if w, ok := weights[j][f.Name]; ok {
					if f.Weight < w {
						w = f.Weight
					}
					shared += w
				}
			}
			if shared > best || (shared == best && shared > 0 && concepts[j].Seed < bestSeed) {
				bestSeed, best = concepts[j].Seed, shared
			}
		}
		concepts[i].Anchor = bestSeed
	}
}

// Definition is the one-sentence description the ontology requires of every
// term, written from what was actually measured rather than from anything
// anyone asserted about the concept.
func (c Concept) Definition() string {
	var b strings.Builder
	b.WriteString("Learned concept")
	if c.Seed != "" {
		b.WriteString(", seeded from ")
		b.WriteString(c.Seed)
	}
	b.WriteString("; strongest evidence ")
	for i, f := range c.Features {
		if i == 3 {
			break
		}
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(f.Name)
	}
	b.WriteString(".")
	return b.String()
}

// grownSeeds lists the seed labels a concept grew from.
func grownSeeds(concepts []Concept) []string {
	grown := make(map[string]bool, len(concepts))
	for _, c := range concepts {
		if c.Seed != "" {
			grown[c.Seed] = true
		}
	}
	return sortedKeys(grown)
}

// Block sizes for the two per-unit fan-outs in this file. A unit's feature
// extraction and its coverage pass are both a few microseconds, between a pair
// comparison and a file parse, so the blocks sit where the other stages' do.
const (
	featureBlock              = 32
	minUnitsPerFeatureWorker  = 128
	coverageBlock             = 32
	minUnitsPerCoverageWorker = 128
)

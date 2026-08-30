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

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
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
	FloorQuantile float64

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
	concepts := expandSeeds(c, seeds, claimed, &m.stats, opt)
	m.unused = grownSeeds(concepts)
	concepts = append(concepts, emergeConcepts(c, claimed, &m.stats, opt)...)

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
	for i := range units {
		c.features[i] = unitFeatures(units[i], g, internal)
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
	for i := range c.features {
		for _, f := range c.features[i] { // ascending: addition order is fixed
			c.mass[i] += c.idf[f]
		}
	}
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

// assign computes every unit's memberships. Two corpus-derived quantities do
// two different jobs, which is what keeps either from having to do both badly,
// and both are stated in *coverage* — the fraction of a unit's own information
// a concept explains — rather than in raw evidence. See corpus.cover for why.
//
// Floor decides membership: the concept's own founding coverage at
// FloorQuantile, so the bar is set by how much of its members a concept
// actually accounts for rather than by a number chosen in advance.
//
// Scale decides what the confidence reads: conf = C/(C+Scale) with Scale the
// median founding coverage, so a unit the concept explains as much of as it
// explains of its typical member reads about 0.5, and one it explains far more
// of approaches 1. Saturating rather than normalized, because coverage has no
// natural maximum either — a concept's weights are lift×idf, not idf, so a unit
// can be explained past its own mass — and pretending it had one would make the
// number a rank in disguise.
//
// The number that admits a unit and the number that grades it are therefore the
// same quantity. Deciding on coverage and grading on evidence would leave the
// size bias in the grade after removing it from the gate.
func assign(c *corpus, concepts []Concept, opt Options) [][]parser.Concept {
	// Inverted, not nested. The direct form — every unit against every
	// concept's vocabulary — is units × concepts × features, and on prometheus
	// (5.5k functions, ~380 concepts) that alone cost most of a minute. Walking
	// each unit's own features into the concepts that use them is the same
	// arithmetic in the order the data is sparse in.
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

	out := make([][]parser.Concept, c.n)
	evidence := make([]float64, len(concepts))
	for i := 0; i < c.n; i++ {
		var touched []int
		for _, f := range c.features[i] { // ascending: the addition order is fixed
			for _, w := range byFeature[f] {
				if evidence[w.concept] == 0 {
					touched = append(touched, w.concept)
				}
				evidence[w.concept] += w.weight
			}
		}
		sort.Ints(touched) // concepts are sorted by ID, so this is ID order
		var got []parser.Concept
		var covers []float64
		for _, j := range touched {
			e := evidence[j]
			evidence[j] = 0 // reset only what was touched
			cov := 0.0
			if c.mass[i] > 0 {
				cov = e / c.mass[i] // one division at the end: accumulation order is untouched
			}
			if cov < concepts[j].Floor {
				continue
			}
			got = append(got, parser.Concept{
				ID:         concepts[j].ID,
				Confidence: cov / (cov + concepts[j].Scale),
			})
			covers = append(covers, cov)
		}
		out[i] = topMemberships(got, covers, opt.MaxMemberships)
	}
	return out
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

// Package culture models the repository's local conceptual practice from
// counts alone: which concepts, roles, and calls co-occur beyond chance
// (ecology), and how each concept is normally realized here (prototypes and
// typicality). The ontology says what a concept is; culture says how this
// particular corpus tends to express it. Everything is deterministic
// counting — no models, no embeddings, no randomness — so an unchanged tree
// yields a deeply-equal Model.
package culture

import (
	"math"
	"sort"

	"github.com/lukse/doppel/internal/concepter"
	"github.com/lukse/doppel/internal/parser"
)

// Options tunes the culture model. The cutoffs prefer silence over noise:
// a concept with too few members gets no prototype, and an association with
// too little support is not an association.
type Options struct {
	MinConceptMembers int     // concepts with fewer members get no prototype or typicality
	MinPairSupport    int     // c(a,b) floor for reporting a positive association
	MinExpected       float64 // expected-co-occurrence floor for reporting a negative association
	MaxCallTokenDF    int     // ecology only: call tokens in more units are corpus idiom, not culture
	AtypicalFactor    float64 // atypical when typicality < factor × the concept's median
	MinHabitatMembers int     // packages with fewer functions get no habitat model
	MisfitFactor      float64 // misfit when strain > factor × the habitat temperature (median strain)
	SurvivorMass      float64 // equilibrium mass floor for an arena survivor
	DominanceMass     float64 // top-survivor mass at or above which the state is dominance
	MinArenaEvidence  float64 // weak floor on an arena's total evidence, nats
}

// DefaultOptions returns the production defaults.
func DefaultOptions() Options {
	return Options{
		MinConceptMembers: 5,
		MinPairSupport:    3,
		MinExpected:       3.0,
		MaxCallTokenDF:    50,
		AtypicalFactor:    0.5,
		MinHabitatMembers: 5,
		MisfitFactor:      2.0,
		SurvivorMass:      0.05,
		DominanceMass:     0.6,
		MinArenaEvidence:  math.Ln2,
	}
}

// ChannelTyp is one channel's contribution to a member's typicality.
type ChannelTyp struct {
	Name       string
	Typicality float64
}

// Stats summarizes one build for the stderr diagnostic lines. Superlative
// fields are empty strings (and their norms/strengths 0) when nothing
// qualifies; ties resolve to the lexicographically smaller name.
type Stats struct {
	ConceptsModeled     int // concepts with enough members for a prototype
	AssociationCount    int // associations surviving the cutoffs
	UnusualRealizations int // (member, concept) pairs flagged atypical corpus-wide

	HabitatsModeled             int    // packages with enough functions for a habitat model
	HabitatMisfits              int    // units flagged Misfit corpus-wide
	MostUniformHabitat          string // habitat with the highest norm ("coldest")
	MostUniformNorm             float64
	MostDiverseHabitat          string // habitat with the lowest norm ("hottest")
	MostDiverseNorm             float64
	StrongestConvention         string // prototyped tag with the highest convention strength
	StrongestConventionStrength float64
	LoosestConvention           string // prototyped tag with the lowest convention strength
	LoosestConventionStrength   float64

	ArenaProfiled  int // units with a concept-arena profile (sum of the four states)
	ArenaDominance int
	ArenaCoalition int
	ArenaConflict  int
	ArenaWeak      int
}

// Model is the built culture model. Query it by unit index (into the units
// slice given to Build) and concept tag.
type Model struct {
	opt           Options
	associations  []Association
	concepts      map[string]*conceptModel // keyed by tag; only prototyped concepts
	habitats      map[string]*habitatModel // keyed by package; only modeled habitats
	habitatByUnit map[int]*habitatModel    // modeled members only
	arenas        map[int]ArenaProfile     // units with a non-empty candidate set only
	stats         Stats
}

type conceptModel struct {
	members    []int // ascending unit indices
	typ        map[int]float64
	chTyp      map[int][]float64 // per member, aligned with channel order
	median     float64
	prototype  Prototype
	convention float64 // concentration of practice, 1 - normalized entropy
}

// unitFeatures is the per-unit feature material shared by prototypes and
// habitats, computed once per build.
type unitFeatures struct {
	tokens         [][]string // resolved call tokens
	sortedPatterns [][]string // sorted unique tags
	flowFeats      [][]string // binarized control-flow labels, sorted
}

// Build computes the whole culture model in one pass. docs[i] describes
// units[i]; g is the resolved call graph.
func Build(units []parser.CodeUnit, docs []concepter.ConceptDoc,
	g *concepter.Graph, opt Options) *Model {

	internal := concepter.QualifiedNames(units)
	uf := &unitFeatures{
		tokens:         make([][]string, len(units)),
		sortedPatterns: make([][]string, len(units)),
		flowFeats:      make([][]string, len(units)),
	}
	for i := range units {
		uf.tokens[i] = concepter.CallTokens(units[i], g, internal)
		uf.sortedPatterns[i] = sortedUniqueTags(units[i].Patterns)
		uf.flowFeats[i] = flowFeatures(units[i])
	}

	m := &Model{
		opt:           opt,
		concepts:      make(map[string]*conceptModel),
		habitats:      make(map[string]*habitatModel),
		habitatByUnit: make(map[int]*habitatModel),
		arenas:        make(map[int]ArenaProfile),
	}
	m.associations = buildAssociations(units, docs, uf.tokens, opt)
	m.stats.AssociationCount = len(m.associations)
	buildPrototypes(m, units, docs, uf, opt)
	buildHabitats(m, units, docs, uf, opt)
	buildArenas(m, units, docs, uf, opt)
	return m
}

// Typicality reports how normally unit idx realizes concept tag, in [0,1].
// ok is false when the concept has no prototype or idx does not carry the tag.
func (m *Model) Typicality(idx int, tag string) (float64, bool) {
	c := m.concepts[tag]
	if c == nil {
		return 0, false
	}
	t, ok := c.typ[idx]
	return t, ok
}

// ChannelTypicality returns the per-channel typicalities behind Typicality,
// in the fixed channel order, or nil when Typicality would report ok=false.
func (m *Model) ChannelTypicality(idx int, tag string) []ChannelTyp {
	c := m.concepts[tag]
	if c == nil {
		return nil
	}
	vals, ok := c.chTyp[idx]
	if !ok {
		return nil
	}
	out := make([]ChannelTyp, len(channelNames))
	for i := range channelNames {
		out[i] = ChannelTyp{Name: channelNames[i], Typicality: vals[i]}
	}
	return out
}

// Median returns the concept's median member typicality.
func (m *Model) Median(tag string) (float64, bool) {
	c := m.concepts[tag]
	if c == nil {
		return 0, false
	}
	return c.median, true
}

// Atypical reports whether unit idx is an unusual realization of tag: it
// carries the tag, the concept has a coherent norm (median > 0), and the
// unit's typicality sits below AtypicalFactor times that norm. Relative to
// the median rather than absolute, so a legitimately diverse concept lowers
// its own bar and a tight concept can flag nobody.
func (m *Model) Atypical(idx int, tag string) bool {
	c := m.concepts[tag]
	if c == nil || c.median <= 0 {
		return false
	}
	t, ok := c.typ[idx]
	return ok && t < m.opt.AtypicalFactor*c.median
}

// Associations returns the corpus association list, cutoffs applied:
// positives first by PMI descending, then negatives by PMI ascending, ties
// on (Kind, A, B).
func (m *Model) Associations() []Association { return m.associations }

// Prototype returns the concept's per-channel feature distributions.
func (m *Model) Prototype(tag string) (Prototype, bool) {
	c := m.concepts[tag]
	if c == nil {
		return Prototype{}, false
	}
	return c.prototype, true
}

// Stats returns the build summary.
func (m *Model) Stats() Stats { return m.stats }

// sortedStrings returns a map's keys in ascending order — the package-wide
// determinism idiom for draining a map.
func sortedStrings(set map[string]bool) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

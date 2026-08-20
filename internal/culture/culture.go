// Package culture models the repository's local conceptual practice from
// counts alone: which concepts, roles, and calls co-occur beyond chance
// (ecology), and how each concept is normally realized here (prototypes and
// typicality). The ontology says what a concept is; culture says how this
// particular corpus tends to express it. Everything is deterministic
// counting — no models, no embeddings, no randomness — so an unchanged tree
// yields a deeply-equal Model.
package culture

import (
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
}

// DefaultOptions returns the production defaults.
func DefaultOptions() Options {
	return Options{
		MinConceptMembers: 5,
		MinPairSupport:    3,
		MinExpected:       3.0,
		MaxCallTokenDF:    50,
		AtypicalFactor:    0.5,
	}
}

// ChannelTyp is one channel's contribution to a member's typicality.
type ChannelTyp struct {
	Name       string
	Typicality float64
}

// Stats summarizes one build for the stderr diagnostic line.
type Stats struct {
	ConceptsModeled     int // concepts with enough members for a prototype
	AssociationCount    int // associations surviving the cutoffs
	UnusualRealizations int // (member, concept) pairs flagged atypical corpus-wide
}

// Model is the built culture model. Query it by unit index (into the units
// slice given to Build) and concept tag.
type Model struct {
	opt          Options
	associations []Association
	concepts     map[string]*conceptModel // keyed by tag; only prototyped concepts
	stats        Stats
}

type conceptModel struct {
	members   []int // ascending unit indices
	typ       map[int]float64
	chTyp     map[int][]float64 // per member, aligned with channel order
	median    float64
	prototype Prototype
}

// Build computes the whole culture model in one pass. docs[i] describes
// units[i]; g is the resolved call graph.
func Build(units []parser.CodeUnit, docs []concepter.ConceptDoc,
	g *concepter.Graph, opt Options) *Model {

	internal := concepter.QualifiedNames(units)
	tokens := make([][]string, len(units))
	for i := range units {
		tokens[i] = concepter.CallTokens(units[i], g, internal)
	}

	m := &Model{
		opt:      opt,
		concepts: make(map[string]*conceptModel),
	}
	m.associations = buildAssociations(units, docs, tokens, opt)
	m.stats.AssociationCount = len(m.associations)
	buildPrototypes(m, units, docs, tokens, opt)
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

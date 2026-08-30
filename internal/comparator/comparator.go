package comparator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

const (
	mergeThreshold  = 0.4
	minMergeSignals = 2

	// relatedEnough is the relatedness at which a non-identical concept match
	// counts as a merge signal rather than merely nudging the score. Sibling
	// concepts clear it; cousins, which top out at 0.33 in the current
	// taxonomy, do not.
	relatedEnough = 0.5
)

// MergeShapeFloor is the code-shape score below which no amount of shared
// context makes two functions merge candidates. It mirrors mergeThreshold
// deliberately: the verdict asserts both halves, so both are gated at 0.4.
//
// The comparator cannot apply this itself — it sees two ConceptDocs and no
// fingerprint — so the caller supplies the shape score. That is the whole
// reason the verdict is split across two functions rather than computed in
// Compare: shared architectural context structurally favours same-package
// siblings, which share callers and callees by construction, so context alone
// reaches 0.4 on pairs whose bodies have almost nothing in common.
const MergeShapeFloor = 0.4

// MergeWorthy is the whole verdict: enough shared context, and enough shared
// shape to act on it.
func MergeWorthy(ev StructuralEvidence, shape float64) bool {
	return ev.ContextMergeWorthy && shape >= MergeShapeFloor
}

// The weight of each signal lives on its relation term in the ontology rather
// than as a constant here, so the scoring table and the vocabulary cannot drift
// apart and an axiom can assert the weights sum to 1.0. Compare reads them
// through the Comparator's own vocabulary (the scorer's), so this default is
// only the corpus-independent path's; the bench harness swaps in a reweighted
// one via ontology.WithWeights.
var onto = ontology.Default()

// Comparator scores structural overlap through a Scorer, which decides whether
// concept matching is corpus-weighted. The pipeline builds one with an
// information-content table derived from the parsed tree, so a near-universal
// tag contributes little evidence and a rare one a lot.
type Comparator struct {
	scorer *ontology.Scorer
	onto   *ontology.Ontology // the scorer's vocabulary; carries the relation weights
}

// New creates a Comparator over the given scorer. The relation weights come
// from the scorer's own vocabulary, so a scorer built over a reweighted
// ontology (ontology.WithWeights — the bench harness's ablation seam) scores
// with those weights while the default path is unchanged.
func New(scorer *ontology.Scorer) *Comparator {
	return &Comparator{scorer: scorer, onto: scorer.Ontology()}
}

// defaultComparator is corpus-independent: plain Wu-Palmer matching, the
// behavior every regression test pins.
var defaultComparator = New(ontology.NewScorer(onto, nil))

// Compare computes the structural overlap between two ConceptDocs with the
// corpus-independent default scorer. The pipeline uses a Comparator instance
// instead; this remains for callers and tests that need no corpus weighting.
func Compare(a, b concepter.ConceptDoc) StructuralEvidence {
	return defaultComparator.Compare(a, b)
}

// StructuralEvidence summarises the structural overlap between two ConceptDocs.
//
// The graded fields sit alongside the boolean and set-valued ones rather than
// replacing them: the booleans still say whether two things are literally the
// same, which is what the report's plain evidence bullets are about, while the
// graded fields carry what the score is actually computed from.
type StructuralEvidence struct {
	SharedCallees    []string
	SharedCallers    []string
	SharedPatterns   []string
	SameRole         bool
	RoleA, RoleB     string
	SamePackage      bool
	SameVisibility   bool
	SameReceiver     bool
	ReceiverA        string
	ReceiverB        string
	SharedCallerPkgs []string
	SharedCalleePkgs []string

	// Graded signals, from reasoning over the ontology.
	PatternRelatedness       float64          // soft overlap of the two intent-tag sets, corpus-weighted when IC is loaded
	RelatedPatterns          []ontology.Match // the pairings behind it, matcher order
	PatternSignalBest        float64          // best taxonomy-only pattern match; feeds countSignals (see Compare)
	RoleRelatedness          float64          // shared fan-in/fan-out axes
	ReceiverRelatedness      float64          // 1.0 same, 0.5 both methods, 0.0 mixed
	EntityKindA              string           // "function" or "method"
	EntityKindB              string
	CallerConceptRelatedness float64 // what the two functions' callers do
	RelatedCallerConcepts    []ontology.Match
	CalleeConceptRelatedness float64 // what the two functions' callees do
	RelatedCalleeConcepts    []ontology.Match
	SharedNeighborhood       []string // depth-2 call-graph names both sit near
	NeighborhoodOverlap      float64  // ratio behind the shares_neighborhood signal

	OverlapScore float64 // 0.0–1.0 weighted composite

	// ContextMergeWorthy is the architectural half of the merge verdict, and
	// only that half: high overlap plus multiple signals. It says the two
	// functions sit in the same part of the system, never that their bodies
	// resemble each other — the comparator has no fingerprint to say so with.
	// Use comparator.MergeWorthy, or analyzer.SimilarPair.MergeWorthy, for the
	// verdict itself.
	ContextMergeWorthy bool

	Reasons []string // human-readable evidence bullets
}

// Compare computes the structural overlap between two ConceptDocs.
//
// Concept and role signals are scored through the ontology rather than by
// string equality. Two functions tagged http_call and db_access used to score
// zero on intent — the same as two functions with nothing in common — even
// though both are I/O; they now score partial credit as cousins under
// io_operation. The same goes for roles that share one of their two axes.
//
// The merge-signal gate deliberately does not follow the scorer: it reads a
// taxonomy-only (Wu-Palmer) best match over bare concept IDs, recorded in
// PatternSignalBest. Under corpus weighting, sibling and cousin similarities
// move with concept frequencies elsewhere in the tree, and a pair's merge
// verdict must not flip because unrelated code shifted the statistics. The same
// argument now covers membership confidence, which is corpus-relative twice
// over — the concept's learned vocabulary and its evidence scale both move with
// the corpus — so the gate sees membership as the boolean it once was. IC and
// confidence bend the score; the signal count stays corpus-independent.
func (c *Comparator) Compare(a, b concepter.ConceptDoc) StructuralEvidence {
	ev := StructuralEvidence{
		SharedCallees:  intersect(a.Callees, b.Callees),
		SharedCallers:  intersect(a.Callers, b.Callers),
		SharedPatterns: intersect(parser.ConceptIDs(a.Concepts), parser.ConceptIDs(b.Concepts)),
		SameRole:       a.Role == b.Role && a.Role != "",
		RoleA:          a.Role,
		RoleB:          b.Role,
		SamePackage:    a.Package == b.Package && a.Package != "",
		SameVisibility: a.Exported == b.Exported,
		// Normalized, so a value-receiver and a pointer-receiver method on one
		// type count as bound to the same thing. The parser keeps the star in
		// the name, so these arrive here as "Server" and "*Server".
		SameReceiver:     ontology.NormalizeReceiver(a.ReceiverType) == ontology.NormalizeReceiver(b.ReceiverType),
		ReceiverA:        a.ReceiverType,
		ReceiverB:        b.ReceiverType,
		SharedCallerPkgs: intersect(a.CallerPackages, b.CallerPackages),
		SharedCalleePkgs: intersect(a.CalleePackages, b.CalleePackages),

		RoleRelatedness:     ontology.RoleRelatedness(a.Role, b.Role),
		ReceiverRelatedness: ontology.ReceiverRelatedness(a.ReceiverType, b.ReceiverType),
		EntityKindA:         string(ontology.EntityKindOf(a.ReceiverType)),
		EntityKindB:         string(ontology.EntityKindOf(b.ReceiverType)),
	}
	// Each side's depth-2 neighborhood, with the counterpart excluded: if a
	// calls b then b sits in a's ball but never its own, so without the
	// exclusion every directly-connected pair pays a systematic penalty in the
	// symmetric difference. (The other direction of that asymmetry is inherent:
	// a's ball then contains b's whole 1-neighborhood, mildly favoring adjacent
	// pairs. Acceptable at this weight — adjacent nodes are structurally close.)
	nbrA := excludeName(a.Neighborhood, qualifiedDocName(b))
	nbrB := excludeName(b.Neighborhood, qualifiedDocName(a))
	ev.SharedNeighborhood = intersect(nbrA, nbrB)
	ev.NeighborhoodOverlap = overlapRatio(nbrA, nbrB, ev.SharedNeighborhood)

	ev.PatternRelatedness, ev.RelatedPatterns = c.scorer.SetRelatednessW(concepter.Graded(a.Concepts), concepter.Graded(b.Concepts))
	ev.CallerConceptRelatedness, ev.RelatedCallerConcepts = c.scorer.SetRelatednessW(concepter.Graded(a.CallerConcepts), concepter.Graded(b.CallerConcepts))
	ev.CalleeConceptRelatedness, ev.RelatedCalleeConcepts = c.scorer.SetRelatednessW(concepter.Graded(a.CalleeConcepts), concepter.Graded(b.CalleeConcepts))

	if c.scorer.Weighted() {
		// Recompute the pattern matches taxonomy-only, and unweighted, for the
		// gate: membership is a bare ID here, not a confidence.
		_, taxonomyMatches := c.onto.SetRelatedness(parser.ConceptIDs(a.Concepts), parser.ConceptIDs(b.Concepts))
		ev.PatternSignalBest = ontology.BestMatch(taxonomyMatches)
	} else {
		// The unweighted matches already are taxonomy-only.
		ev.PatternSignalBest = ontology.BestMatch(ev.RelatedPatterns)
	}

	// One explicit ordered expression, deliberately. Summing over a map of
	// weights would let iteration order change the low bits of a float sum,
	// which is enough to move a pair across the merge threshold or the
	// --struct-min cutoff and make the report vary between runs.
	ev.OverlapScore = 0 +
		c.onto.Weight(ontology.RelCalls)*overlapRatio(a.Callees, b.Callees, ev.SharedCallees) +
		c.onto.Weight(ontology.RelExhibits)*ev.PatternRelatedness +
		c.onto.Weight(ontology.RelCalledBy)*overlapRatio(a.Callers, b.Callers, ev.SharedCallers) +
		c.onto.Weight(ontology.RelSharesNeighborhood)*ev.NeighborhoodOverlap +
		c.onto.Weight(ontology.RelHasRole)*ev.RoleRelatedness +
		c.onto.Weight(ontology.RelDeclaredIn)*boolFloat(ev.SamePackage) +
		c.onto.Weight(ontology.RelCalledFromConcept)*ev.CallerConceptRelatedness +
		c.onto.Weight(ontology.RelCallsIntoConcept)*ev.CalleeConceptRelatedness +
		c.onto.Weight(ontology.RelHasVisibility)*boolFloat(ev.SameVisibility) +
		c.onto.Weight(ontology.RelBoundTo)*ev.ReceiverRelatedness +
		c.onto.Weight(ontology.RelCalledFromPackage)*overlapRatio(a.CallerPackages, b.CallerPackages, ev.SharedCallerPkgs) +
		c.onto.Weight(ontology.RelCallsIntoPackage)*overlapRatio(a.CalleePackages, b.CalleePackages, ev.SharedCalleePkgs)

	if ev.OverlapScore > 1.0 {
		ev.OverlapScore = 1.0
	}

	// Build reasons and determine merge-worthiness.
	ev.Reasons = buildReasons(ev)
	signals := countSignals(ev)
	ev.ContextMergeWorthy = ev.OverlapScore >= mergeThreshold && signals >= minMergeSignals

	return ev
}

// overlapRatio returns |shared| / max(|a|, |b|), or 0 if both are empty.
func overlapRatio(a, b, shared []string) float64 {
	m := len(a)
	if len(b) > m {
		m = len(b)
	}
	if m == 0 {
		return 0
	}
	return float64(len(shared)) / float64(m)
}

func boolFloat(v bool) float64 {
	if v {
		return 1.0
	}
	return 0.0
}

// countSignals counts how many distinct merge-supporting signals are present.
//
// Five of the twelve scored signals can count. Visibility, receiver binding,
// the two package-overlap signals and the two caller/callee concept signals
// raise the score but never the count: they are context, and context alone is
// not a reason to merge two functions.
func countSignals(ev StructuralEvidence) int {
	n := 0
	if len(ev.SharedCallees) > 0 {
		n++
	}
	if len(ev.SharedCallers) > 0 {
		n++
	}
	// Judged on the best single taxonomy-only pairing, not on the aggregate
	// ratio and not on the corpus-weighted matches — see the gate note on
	// Compare. Thresholding the ratio would be a regression rather than a
	// guard: three tags with one exact match average to 0.33, so a pair that
	// counts today would stop counting at an unchanged score. Any exact match
	// scores 1.0, so this is a strict generalization of "the two share a tag".
	if ev.PatternSignalBest >= relatedEnough {
		n++
	}
	if ev.SameRole {
		n++
	}
	if ev.SamePackage {
		n++
	}
	return n
}

func buildReasons(ev StructuralEvidence) []string {
	var reasons []string
	if len(ev.SharedCallees) > 0 {
		reasons = append(reasons, fmt.Sprintf("share %d callees: [%s]", len(ev.SharedCallees), strings.Join(ev.SharedCallees, ", ")))
	}
	if len(ev.SharedCallers) > 0 {
		reasons = append(reasons, fmt.Sprintf("share %d callers: [%s]", len(ev.SharedCallers), strings.Join(ev.SharedCallers, ", ")))
	}
	// Count only: depth-2 name lists run long enough to bury a report, and the
	// names are still on the struct for anyone who asks.
	if ev.NeighborhoodOverlap > 0 {
		reasons = append(reasons, fmt.Sprintf("overlapping call-graph neighborhoods (%.2f): %d shared",
			ev.NeighborhoodOverlap, len(ev.SharedNeighborhood)))
	}
	if len(ev.SharedPatterns) > 0 {
		reasons = append(reasons, fmt.Sprintf("share patterns: [%s]", strings.Join(ev.SharedPatterns, ", ")))
	}
	// Without this line a pair of near-miss tags would raise the overlap score
	// with nothing in the report to account for it.
	if near := describeNearMatches(ev.RelatedPatterns); near != "" {
		reasons = append(reasons, "related patterns: "+near)
	}
	if ev.SameRole {
		reasons = append(reasons, fmt.Sprintf("both are %s functions", ev.RoleA))
	} else if ev.RoleRelatedness > 0 {
		reasons = append(reasons, fmt.Sprintf("related roles: %s ≈ %s (%s, %.2f)",
			ev.RoleA, ev.RoleB, sharedAxis(ev.RoleA, ev.RoleB), ev.RoleRelatedness))
	}
	if ev.SamePackage {
		reasons = append(reasons, "same package")
	}
	// The score goes before the list, not after it. A bullet ending in
	// "[...] (0.67)" is link syntax in the Markdown report, and the score would
	// silently become an href.
	if ev.CallerConceptRelatedness > 0 {
		reasons = append(reasons, fmt.Sprintf("callers do related work (%.2f): [%s]",
			ev.CallerConceptRelatedness, describeMatches(ev.RelatedCallerConcepts)))
	}
	if ev.CalleeConceptRelatedness > 0 {
		reasons = append(reasons, fmt.Sprintf("callees do related work (%.2f): [%s]",
			ev.CalleeConceptRelatedness, describeMatches(ev.RelatedCalleeConcepts)))
	}
	if ev.SameVisibility {
		reasons = append(reasons, "same visibility")
	}
	if ev.SameReceiver {
		recv := "plain functions"
		if r := ontology.NormalizeReceiver(ev.ReceiverA); r != "" {
			recv = r
		}
		reasons = append(reasons, fmt.Sprintf("same receiver type: %s", recv))
	} else if ev.ReceiverRelatedness > 0 {
		reasons = append(reasons, fmt.Sprintf("both are methods, on %s and %s", ev.ReceiverA, ev.ReceiverB))
	}
	if len(ev.SharedCallerPkgs) > 0 {
		reasons = append(reasons, fmt.Sprintf("called from same packages: [%s]", strings.Join(ev.SharedCallerPkgs, ", ")))
	}
	if len(ev.SharedCalleePkgs) > 0 {
		reasons = append(reasons, fmt.Sprintf("call into same packages: [%s]", strings.Join(ev.SharedCalleePkgs, ", ")))
	}
	return reasons
}

// describeNearMatches renders only the pairings between different concepts,
// naming the ancestor that relates them. Exact matches are omitted: they are
// already covered by the shared-patterns bullet.
func describeNearMatches(matches []ontology.Match) string {
	var parts []string
	for _, m := range matches {
		if m.Exact() {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s ≈ %s (both %s, %.2f)", m.A, m.B, m.LCA, m.Score))
	}
	return strings.Join(parts, "; ")
}

// describeMatches renders every pairing compactly, exact ones as a single term.
func describeMatches(matches []ontology.Match) string {
	var parts []string
	for _, m := range matches {
		if m.Exact() {
			parts = append(parts, m.A)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s ≈ %s", m.A, m.B))
	}
	return strings.Join(parts, ", ")
}

// sharedAxis names why two different roles are related at all, which is always
// exactly one of the two axes — a pair agreeing on both would be the same role.
func sharedAxis(a, b string) string {
	ax, okA := ontology.AxesFor(ontology.TermID(a))
	bx, okB := ontology.AxesFor(ontology.TermID(b))
	if !okA || !okB {
		return "related"
	}
	if ax.HighFanIn && bx.HighFanIn {
		return "both high fan-in"
	}
	if ax.HighFanOut && bx.HighFanOut {
		return "both high fan-out"
	}
	return "related"
}

// qualifiedDocName mirrors concepter.QualifiedName for a ConceptDoc.
func qualifiedDocName(d concepter.ConceptDoc) string {
	if d.Package == "" {
		return d.Name
	}
	return d.Package + "." + d.Name
}

// excludeName returns names without the given entry, sharing the input slice
// when the entry is absent.
func excludeName(names []string, drop string) []string {
	for i, n := range names {
		if n == drop {
			out := make([]string, 0, len(names)-1)
			out = append(out, names[:i]...)
			return append(out, names[i+1:]...)
		}
	}
	return names
}

// intersect returns the sorted intersection of two sorted string slices.
func intersect(a, b []string) []string {
	set := make(map[string]struct{}, len(a))
	for _, s := range a {
		set[s] = struct{}{}
	}
	var out []string
	for _, s := range b {
		if _, ok := set[s]; ok {
			out = append(out, s)
			delete(set, s) // avoid duplicates
		}
	}
	sort.Strings(out)
	return out
}

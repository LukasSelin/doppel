package snapshot

import (
	"fmt"
	"math"
	"sort"
)

// driftFloor is the smallest score movement worth reporting. The text report
// prints scores to four decimals and the digest to three, so anything smaller
// than this is a difference nobody can see.
const driftFloor = 0.005

// Delta is what changed between two snapshots.
//
// The fields are ordered by how much they can be trusted, and the renderer
// leads with the same order.
//
// UnitsAdded, UnitsRemoved and BodiesChanged are the solid ground: they come
// from names and from the fingerprint digest, both of which depend only on a
// function's own AST. If a body digest moved, that function was edited, full
// stop.
//
// PairsAdded and PairsRemoved are one tier down. Retrieval keeps a bounded
// number of neighbours per function, so a pair can enter or leave the candidate
// set without either side being touched. Each pair change therefore carries an
// Attributable bit saying whether at least one of its two sides is in the
// touched set; the renderer leads with the attributable ones, because those are
// the changes this session actually caused.
//
// Everything else corpus-relative — role changes, caller and callee counts,
// overlap movement, tag totals — is deliberately absent. Those move when code
// nobody touched moves, and reporting them would be reporting the session for
// something it did not do.
type Delta struct {
	// Comparable is false when the two snapshots answer different questions.
	// Callers should say so and stop, not report a partial diff.
	Comparable bool   `json:"comparable"`
	Reason     string `json:"reason,omitempty"`

	FunctionsBefore int `json:"functionsBefore"`
	FunctionsAfter  int `json:"functionsAfter"`
	PairsBefore     int `json:"pairsBefore"`
	PairsAfter      int `json:"pairsAfter"`
	MergeBefore     int `json:"mergeWorthyBefore"`
	MergeAfter      int `json:"mergeWorthyAfter"`

	UnitsAdded    []Unit `json:"unitsAdded,omitempty"`   // sorted by key
	UnitsRemoved  []Unit `json:"unitsRemoved,omitempty"` // sorted by key
	BodiesChanged []Unit `json:"bodiesChanged,omitempty"`

	PairsAdded   []PairChange `json:"pairsAdded,omitempty"`   // sorted: attributable first, then score desc
	PairsRemoved []PairChange `json:"pairsRemoved,omitempty"` // sorted: attributable first, then score desc
	Drift        []Drift      `json:"drift,omitempty"`
}

// PairChange is a pair that entered or left the candidate set, carrying whether
// the change can be traced to a function this session actually edited.
type PairChange struct {
	Pair
	// Attributable is true when at least one side was added or had its body
	// changed. When false the pair moved because retrieval re-ranked around it,
	// which is corpus churn rather than a consequence of the edit.
	Attributable bool `json:"attributable"`
}

// Drift is a pair present in both runs whose standing moved.
//
// "Standing" is deliberately wider than "score". Merge-worthiness is gated on
// overlap, not on shape score, so a pair can cross into or out of merge-worthy
// with its score untouched. Admitting only score movement would make exactly
// the most consequential drift — the kind that changes what you should do
// about a pair — the kind that never gets recorded.
type Drift struct {
	A                 string  `json:"a"`
	B                 string  `json:"b"`
	ScoreBefore       float64 `json:"scoreBefore"`
	ScoreAfter        float64 `json:"scoreAfter"`
	OverlapBefore     float64 `json:"overlapBefore"`
	OverlapAfter      float64 `json:"overlapAfter"`
	MergeWorthyBefore bool    `json:"mergeWorthyBefore"`
	MergeWorthyAfter  bool    `json:"mergeWorthyAfter"`
	Attributable      bool    `json:"attributable"` // at least one side's body changed
}

// CrossedGate reports whether this pair changed merge-worthiness. It is the
// only drift that changes a decision, so it is what a short report leads with.
func (d Drift) CrossedGate() bool { return d.MergeWorthyBefore != d.MergeWorthyAfter }

// ScoreMoved is the absolute shape-score movement.
func (d Drift) ScoreMoved() float64 { return math.Abs(d.ScoreAfter - d.ScoreBefore) }

// AttributablePairs returns the added and removed pairs this session's edits
// actually caused, which is what a report should lead with.
func (d Delta) AttributablePairs() (added, removed []PairChange) {
	for _, p := range d.PairsAdded {
		if p.Attributable {
			added = append(added, p)
		}
	}
	for _, p := range d.PairsRemoved {
		if p.Attributable {
			removed = append(removed, p)
		}
	}
	return added, removed
}

// Empty reports whether the delta says nothing worth telling anyone. A hook
// that renders an empty delta should emit no output at all: a "nothing
// changed" line after every turn is worse than silence.
func (d Delta) Empty() bool {
	return len(d.UnitsAdded) == 0 && len(d.UnitsRemoved) == 0 && len(d.BodiesChanged) == 0 &&
		len(d.PairsAdded) == 0 && len(d.PairsRemoved) == 0 && len(d.Drift) == 0
}

// Diff compares a baseline against a later run.
//
// Incomparability is a result, not an error: the hook has exactly one behaviour
// for every version of "these cannot be compared", and returning an error would
// tempt a caller into exiting non-zero over it. A mismatched schema, build,
// ontology or param set means the two runs measured different things, and a
// diff across them would be confidently wrong rather than merely unavailable.
//
// The full refusal list is Schema, Doppel, Ontology, RuleSet and Params — the
// union of two development lines' conditions, kept whole because each line
// refused for something the other could not see.
func Diff(base, head Snapshot) Delta {
	d := Delta{
		Comparable:      true,
		FunctionsBefore: base.Functions,
		FunctionsAfter:  head.Functions,
		PairsBefore:     len(base.Pairs),
		PairsAfter:      len(head.Pairs),
		MergeBefore:     base.MergeWorthy(),
		MergeAfter:      head.MergeWorthy(),
	}

	if why := incomparable(base, head); why != "" {
		d.Comparable = false
		d.Reason = why
		return d
	}

	baseUnits, headUnits := base.UnitByKey(), head.UnitByKey()

	for _, u := range head.Units {
		prev, ok := baseUnits[u.Key]
		if !ok {
			d.UnitsAdded = append(d.UnitsAdded, u)
			continue
		}
		// An empty digest means "no body"; two of those are not a match, so
		// only compare when both sides actually have one.
		if u.Digest != "" && prev.Digest != "" && u.Digest != prev.Digest {
			d.BodiesChanged = append(d.BodiesChanged, u)
		}
	}
	for _, u := range base.Units {
		if _, ok := headUnits[u.Key]; !ok {
			d.UnitsRemoved = append(d.UnitsRemoved, u)
		}
	}

	// A unit counts as touched if it is new or its body changed. That set is
	// what makes a pair change attributable to this session's edits.
	touched := d.Touched()

	basePairs := pairIndex(base.Pairs)
	headPairs := pairIndex(head.Pairs)

	for _, p := range head.Pairs {
		prev, ok := basePairs[p.Key()]
		if !ok {
			d.PairsAdded = append(d.PairsAdded, PairChange{Pair: p, Attributable: touched[p.A] || touched[p.B]})
			continue
		}
		if math.Abs(p.Score-prev.Score) >= driftFloor || prev.MergeWorthy != p.MergeWorthy {
			d.Drift = append(d.Drift, Drift{
				A:                 p.A,
				B:                 p.B,
				ScoreBefore:       prev.Score,
				ScoreAfter:        p.Score,
				OverlapBefore:     prev.Overlap,
				OverlapAfter:      p.Overlap,
				MergeWorthyBefore: prev.MergeWorthy,
				MergeWorthyAfter:  p.MergeWorthy,
				Attributable:      touched[p.A] || touched[p.B],
			})
		}
	}
	for _, p := range base.Pairs {
		if _, ok := headPairs[p.Key()]; !ok {
			d.PairsRemoved = append(d.PairsRemoved, PairChange{Pair: p, Attributable: touched[p.A] || touched[p.B]})
		}
	}

	sortUnits(d.UnitsAdded)
	sortUnits(d.UnitsRemoved)
	sortUnits(d.BodiesChanged)
	sortPairChanges(d.PairsAdded)
	sortPairChanges(d.PairsRemoved)
	sort.Slice(d.Drift, func(i, j int) bool {
		// Gate crossings first: they are the only drift that changes a decision,
		// and a crossing can carry zero score movement, which would otherwise
		// sort it last.
		if ci, cj := d.Drift[i].CrossedGate(), d.Drift[j].CrossedGate(); ci != cj {
			return ci
		}
		di, dj := d.Drift[i].ScoreMoved(), d.Drift[j].ScoreMoved()
		if di != dj {
			return di > dj
		}
		if d.Drift[i].A != d.Drift[j].A {
			return d.Drift[i].A < d.Drift[j].A
		}
		return d.Drift[i].B < d.Drift[j].B
	})

	return d
}

// Touched reports whether a pair change can be attributed to an edit in this
// session, rather than to corpus drift. Under an unfiltered snapshot every
// added or removed pair should be attributable; one that is not indicates the
// snapshot was taken with a top-N or struct-min filter applied.
func (d Delta) Touched() map[string]bool {
	m := make(map[string]bool, len(d.UnitsAdded)+len(d.BodiesChanged))
	for _, u := range d.UnitsAdded {
		m[u.Key] = true
	}
	for _, u := range d.BodiesChanged {
		m[u.Key] = true
	}
	return m
}

func incomparable(base, head Snapshot) string {
	switch {
	case base.Schema != head.Schema:
		return fmt.Sprintf("baseline schema %d, current schema %d", base.Schema, head.Schema)
	case base.Doppel != head.Doppel:
		return fmt.Sprintf("baseline built by doppel %s, current %s", base.Doppel, head.Doppel)
	case base.Ontology != head.Ontology:
		// Every relation weight and the taxonomy shape feed OverlapScore, so a
		// vocabulary change moves scores that no edit touched.
		return fmt.Sprintf("baseline used ontology %s, current %s", base.Ontology, head.Ontology)
	case base.RuleSet != head.RuleSet:
		// A canonicalization rule added or changed moves every WL label a
		// body produces, so the same two untouched functions would score a
		// different WL Jaccard and Containment under the two rule sets —
		// exactly the schema-4-vs-5 failure this check's siblings exist to
		// prevent, one layer lower.
		return fmt.Sprintf("baseline used canon rule set %s, current %s", base.RuleSet, head.RuleSet)
	case !base.Params.Equal(head.Params):
		return fmt.Sprintf("baseline params %+v, current %+v", base.Params, head.Params)
	}
	return ""
}

func pairIndex(pairs []Pair) map[string]Pair {
	m := make(map[string]Pair, len(pairs))
	for _, p := range pairs {
		m[p.Key()] = p
	}
	return m
}

func sortUnits(u []Unit) {
	sort.Slice(u, func(i, j int) bool { return u[i].Key < u[j].Key })
}

// sortPairChanges orders by how much a reader needs to see the line, because
// the renderer prints only a bounded prefix of it.
//
// Attributable first: a change this session caused must never be pushed off the
// list by corpus churn that happens to score higher. Then merge-worthy, which
// is the only verdict that asks anyone to do something — a single actionable
// pair buried under nine high-shape-score coincidences is a report nobody acts
// on. Score breaks the remaining ties.
func sortPairChanges(p []PairChange) {
	sort.Slice(p, func(i, j int) bool {
		if p[i].Attributable != p[j].Attributable {
			return p[i].Attributable
		}
		if p[i].MergeWorthy != p[j].MergeWorthy {
			return p[i].MergeWorthy
		}
		if p[i].Score != p[j].Score {
			return p[i].Score > p[j].Score
		}
		if p[i].A != p[j].A {
			return p[i].A < p[j].A
		}
		return p[i].B < p[j].B
	})
}

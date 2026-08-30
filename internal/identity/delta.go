package identity

import (
	"sort"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

// A Delta is the whole of what happened between two runs: every function's
// classification, and the near-duplicate pairs those changes created or
// dissolved.
//
// # Why the pair half lives here and not in snapshot.Diff
//
// snapshot.Diff already reports pairs that entered or left the candidate set,
// and it attributes each one to "a unit that was added or whose digest moved" —
// the only attribution key equality can offer. That is deliberately conservative
// and it stays as it is. This type asks the same question with the richer key:
// a pair is attributed when either of its two functions participates in a
// classification that is not `unchanged`, which includes the renames and moves
// Diff can only see as a deletion plus an arrival.
//
// The two never disagree about a fact, only about how much they can name. A
// renamed function's old pair is dissolved and its new pair created under both
// readings; only this one can say the rename is why.
//
// # What a pair change may claim
//
// The same caveat Delta's own doc carries, one tier down: retrieval keeps a
// bounded top-K per function, so a pair can enter or leave the candidate set
// with neither body touched. Such a change carries no class on either side and
// is reported as exactly that — the renderers put it after everything the
// session can be held responsible for, and label it.
//
// Nothing here is recomputed. A created pair's Explain is the sentence the new
// run stored; a dissolved pair's is the old run's. Scores likewise. This type
// compares two records; it never re-scores anything.
// Result is embedded rather than named, so the JSON payload gains the two pair
// lists without moving a single field a consumer already reads. A machine that
// only knows about the classification keeps working; one that wants the pair
// half finds it beside the fields it came from.
type Delta struct {
	Result

	// Created exists in the new snapshot and not the old; Dissolved the
	// inverse. Both are sorted by sortPairChanges.
	Created   []PairChange `json:"created,omitempty"`
	Dissolved []PairChange `json:"dissolved,omitempty"`
}

// PairChange is one near-duplicate pair present in exactly one of the two
// snapshots, carrying the stored numbers of the run that had it and the
// classification of each of its two sides.
//
// AClass and BClass are the classes of the changes A and B participate in, on
// the side of the comparison this pair came from: a created pair's keys are new
// keys and its classes come from the changes' New members, a dissolved pair's
// from their Old members. An empty class means the key named no unit in that
// snapshot, which a well-formed pair list never does.
type PairChange struct {
	A           string  `json:"a"`
	B           string  `json:"b"`
	Score       float64 `json:"score"`
	Overlap     float64 `json:"overlap"`
	MergeWorthy bool    `json:"mergeWorthy"`

	// Explain is the stored pair sentence — analyzer.Explain's output as the
	// run that held this pair recorded it. It is read back, never recomputed:
	// the sentence names the canonicalization rules that fired on two specific
	// bodies, and this package has neither body.
	Explain string `json:"explain,omitempty"`

	AClass Class `json:"aClass,omitempty"`
	BClass Class `json:"bClass,omitempty"`
}

// Attributable reports whether a classified change on either side explains this
// pair. When false the pair moved because retrieval re-ranked around it, which
// is corpus churn rather than a consequence of anything the session did — the
// same distinction snapshot.PairChange.Attributable draws, decided on the wider
// evidence this package has.
func (p PairChange) Attributable() bool {
	return classifiesAChange(p.AClass) || classifiesAChange(p.BClass)
}

func classifiesAChange(c Class) bool { return c != "" && c != Unchanged }

// A Cause is one classified function that explains a pair change.
type Cause struct {
	Key   string
	Class Class
}

// Causes names the classified functions that explain this pair, in A-then-B
// order, or nothing when neither side was classified.
func (p PairChange) Causes() []Cause {
	var out []Cause
	if classifiesAChange(p.AClass) {
		out = append(out, Cause{Key: p.A, Class: p.AClass})
	}
	if classifiesAChange(p.BClass) {
		out = append(out, Cause{Key: p.B, Class: p.BClass})
	}
	return out
}

// Empty reports whether the delta has nothing to say: no function classified as
// anything but unchanged, and no pair created or dissolved. A renderer that has
// an empty delta should print nothing — a "nothing changed" line after every
// turn trains the reader to skip the place real findings appear.
func (d Delta) Empty() bool {
	if len(d.Created) > 0 || len(d.Dissolved) > 0 {
		return false
	}
	for _, cc := range d.Counts {
		if cc.Class != Unchanged && cc.Count > 0 {
			return false
		}
	}
	return true
}

// Since compares a baseline run against a later one and returns both halves of
// the answer.
//
// Incomparability is a Delta, not an error, exactly as it is a Result in
// Compare: the pair halves are left empty, because a comparison the matcher
// refused cannot attribute a pair change to anything either. An unreadable WL
// bag is still an error — that is a corrupt file, not a refusal.
func Since(base, head snapshot.Snapshot, opt Options) (Delta, error) {
	r, err := Compare(base, head, opt)
	if err != nil {
		return Delta{}, err
	}
	d := Delta{Result: r}
	if !r.Comparable {
		return d, nil
	}

	oldClass, newClass := classIndex(r)
	basePairs := pairKeys(base.Pairs)
	headPairs := pairKeys(head.Pairs)

	for _, p := range head.Pairs {
		if _, ok := basePairs[p.Key()]; ok {
			continue
		}
		d.Created = append(d.Created, pairChange(p, newClass))
	}
	for _, p := range base.Pairs {
		if _, ok := headPairs[p.Key()]; ok {
			continue
		}
		d.Dissolved = append(d.Dissolved, pairChange(p, oldClass))
	}

	sortPairChanges(d.Created)
	sortPairChanges(d.Dissolved)
	return d, nil
}

func pairChange(p snapshot.Pair, class map[string]Class) PairChange {
	return PairChange{
		A: p.A, B: p.B,
		Score:       p.Score,
		Overlap:     p.Overlap,
		MergeWorthy: p.MergeWorthy,
		Explain:     p.Explain,
		AClass:      class[p.A],
		BClass:      class[p.B],
	}
}

// classIndex maps each snapshot's unit keys to the class of the change that
// unit participates in. Two maps, because a key is only meaningful on its own
// side: `svc.Total` names the old function and `svc.Sum` the new one, and a
// rename is the case where looking one up in the other's index would be
// silently wrong.
//
// Both are plain lookup maps: every read is by key, nothing iterates them, and
// no order depends on them.
func classIndex(r Result) (oldSide, newSide map[string]Class) {
	oldSide = make(map[string]Class)
	newSide = make(map[string]Class)
	for _, c := range r.Changes {
		for _, m := range c.Old {
			oldSide[m.Key] = c.Class
		}
		for _, m := range c.New {
			newSide[m.Key] = c.Class
		}
	}
	return oldSide, newSide
}

func pairKeys(ps []snapshot.Pair) map[string]struct{} {
	m := make(map[string]struct{}, len(ps))
	for _, p := range ps {
		m[p.Key()] = struct{}{}
	}
	return m
}

// sortPairChanges mirrors snapshot.sortPairChanges, which is the established
// order for a list of pair changes a renderer prints only a bounded prefix of:
// attributable first, so a change the session caused is never pushed off the
// list by corpus churn that happens to score higher; then merge-worthy, the
// only verdict that asks anyone to do something; then score. The trailing key
// comparisons make the order total, which the byte-identical-output invariant
// requires.
//
// The one difference from snapshot's is what "attributable" means — see
// PairChange.Attributable.
func sortPairChanges(ps []PairChange) {
	sort.Slice(ps, func(i, j int) bool {
		a, b := ps[i], ps[j]
		if ai, bi := a.Attributable(), b.Attributable(); ai != bi {
			return ai
		}
		if a.MergeWorthy != b.MergeWorthy {
			return a.MergeWorthy
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.A != b.A {
			return a.A < b.A
		}
		return a.B < b.B
	})
}

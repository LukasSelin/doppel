// Package identity matches functions across two snapshots and says what
// happened to each one.
//
// # This is not snapshot.Diff
//
// internal/snapshot's Diff answers "what did this session do to the pair
// list", and it answers it by *key equality alone*: a unit whose
// package.Name is in both runs is the same unit, one that is not was added or
// removed. That is exactly the right rule for the hook it feeds — it never
// claims anything it cannot attribute — but it means every rename reads as one
// deletion plus one addition, and a function moved between packages reads the
// same way.
//
// This package asks the harder question. Given two snapshots it matches
// function *bodies* across them, using the Weisfeiler-Lehman label bags
// schema 6 stores on every unit, and reports one label per function: it
// survived untouched, it was edited, it was renamed, it moved, it was split
// into several, several were merged into it, it is new, it is gone. Diff
// stays as it is; nothing here feeds a hook, a score, or a ranking.
//
// # What it may claim, and on what evidence
//
// Two kinds of evidence, and the classification names which it used:
//
//   - snapshot.Unit.Digest is exact. It hashes a function's own fingerprint
//     and nothing about the corpus, so two equal non-empty digests mean the
//     same body, full stop. Body identity is decided here and nowhere else.
//   - The WL bag decides *similarity*, which is what matching needs and what
//     digests cannot give: a renamed-and-slightly-edited function shares no
//     digest with its predecessor but shares almost all of its labels.
//     fingerprint.WLOverlap turns two bags into the weighted Jaccard and the
//     containment, both in the corpus's own information units.
//
// Every classification prints the evidence that produced it — jaccard,
// containment, and whether the digests agreed — so a reader can falsify any
// line by opening two files.
//
// # The label population is the union of both runs
//
// The weighted Jaccard needs a label IDF, and an IDF needs a stated
// population. This package counts document frequency over the union of both
// snapshots' decoded bags rather than over either side's own.
//
// The reason is that any other choice makes the answer asymmetric.
// ln(N_old/df_old) and ln(N_new/df_new) are different numbers for the same
// label, so scoring the old-to-new direction under the old corpus's norm and
// the new-to-old direction under the new one would let a pair be each other's
// best match in one direction and not the other — and the greedy matcher would
// then depend on which snapshot was passed first. The union is symmetric by
// construction, deterministic (it is a multiset of bags, and
// fingerprint.LabelWeights documents that only the multiset matters, never the
// order), and it is the honest reading of the question being asked: not "how
// rare is this label in the old corpus" but "how rare is it across the two
// corpora being compared".
//
// A consequence worth stating: these numbers are therefore not the ones either
// snapshot's own Pair records carry. A stored Pair.Score was computed under one
// run's own label weights. Nothing here reads Pair at all.
package identity

import (
	"fmt"
	"sort"

	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// Class is what happened to one function between two snapshots. Exactly one
// class is assigned per finding, and every function on either side appears in
// exactly one finding.
type Class string

// The eight classes. The Go identifier for "new" is Added, because New is
// what a constructor is called in Go and this is not one; the wire value is
// "new".
const (
	Unchanged Class = "unchanged"
	Edited    Class = "edited"
	Renamed   Class = "renamed"
	Moved     Class = "moved"
	Split     Class = "split"
	Merged    Class = "merged"
	Added     Class = "new"
	Deleted   Class = "deleted"
)

// classOrder is the report order, and the order Result.Counts is emitted in.
// Both surfaces read it, so a class cannot be printed in one order and counted
// in another.
//
// Structural relocations lead: split, merged and moved are the findings a
// key-equality diff cannot produce at all, and they are the reason this
// package exists. Renames follow, then edits, then the population changes a
// plain diff already reports, and unchanged last — it is the bulk of any real
// comparison and the least informative line in it.
var classOrder = []Class{Split, Merged, Moved, Renamed, Edited, Added, Deleted, Unchanged}

func classRank(c Class) int {
	for i, k := range classOrder {
		if k == c {
			return i
		}
	}
	return len(classOrder)
}

// Options are the matcher's thresholds. The zero value means DefaultOptions,
// so a caller that does not care passes Options{} — the same convention
// retriever.Options uses.
type Options struct {
	// RenameFloor is the weighted-Jaccard admission floor for a match made on
	// *similarity* rather than on identity. A pair whose keys are equal, or
	// whose digests are equal, is matched regardless: those are identity
	// facts and no similarity number can improve on them. Everything else has
	// to clear this floor, because below half the shared informative mass,
	// calling one function a rename of another is a claim the evidence does
	// not support — and the matcher would otherwise pair off whatever
	// leftovers happened to share one label.
	RenameFloor float64

	// SplitContainment is how much of a body has to be covered for the split
	// and merged rules to fire. 0.8 is the task's own definition of the
	// classes and is not derived from anything.
	SplitContainment float64

	// CandidateK bounds how many counterparts each function keeps from
	// candidate generation. It is a recall bound and nothing else: every
	// number reported is the exact WLOverlap of the two bags, never an
	// approximation from the accumulator that proposed the pair.
	CandidateK int

	// MaxLabelDF drops a label from candidate *generation* when more than
	// this many functions on the indexed side carry it. Such a label is a
	// language idiom (a bare block, an identifier) whose ln(N/df) is near
	// zero, so it moves almost no mass while costing a posting list
	// proportional to the corpus. It bounds accumulation at
	// MaxLabelDF x (total bag entries) instead of the product of the two
	// populations. It never reaches a reported number.
	MaxLabelDF int
}

// DefaultOptions are the shipped thresholds.
func DefaultOptions() Options {
	return Options{RenameFloor: 0.5, SplitContainment: 0.8, CandidateK: 8, MaxLabelDF: 200}
}

func (o Options) withDefaults() Options {
	d := DefaultOptions()
	if o.RenameFloor == 0 {
		o.RenameFloor = d.RenameFloor
	}
	if o.SplitContainment == 0 {
		o.SplitContainment = d.SplitContainment
	}
	if o.CandidateK == 0 {
		o.CandidateK = d.CandidateK
	}
	if o.MaxLabelDF == 0 {
		o.MaxLabelDF = d.MaxLabelDF
	}
	return o
}

// Function is one function as one of the two snapshots recorded it.
type Function struct {
	Key     string `json:"key"`
	Package string `json:"package"`
	Name    string `json:"name"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Digest  string `json:"digest"`
}

// Member is one function's participation in a Change, carrying the evidence
// specific to that participation.
//
// For the one-to-one classes there is exactly one Member on each side and the
// evidence lives on the Change. For split and merged the pivot side has one
// Member and the other side has two or more, each carrying its own
// Containment against the pivot — which is the whole evidence for the class,
// and the thing a reader checks.
type Member struct {
	Function
	Jaccard     float64 `json:"jaccard"`
	Containment float64 `json:"containment"`
}

// Change is one classified finding: one function's fate, or one split or
// merge.
//
// Old and New are sorted by key and are never both empty. Every one-to-one
// class has exactly one of each; Added has only New, Deleted only Old; Split
// has one Old and two or more New, Merged the inverse.
type Change struct {
	Class Class    `json:"class"`
	Old   []Member `json:"old,omitempty"`
	New   []Member `json:"new,omitempty"`

	// The one-to-one evidence. Zero for Split, Merged, Added and Deleted,
	// where the per-Member numbers are what carries the claim.
	Jaccard     float64 `json:"jaccard"`
	Containment float64 `json:"containment"`
	DigestEqual bool    `json:"digestEqual"`

	// The secondary facts the precedence order collapsed. A moved function
	// that was also renamed reports Class Moved with NameChanged true, and
	// the renderer prints both.
	NameChanged    bool `json:"nameChanged"`
	PackageChanged bool `json:"packageChanged"`
}

// ClassCount is one class and how many findings carry it. Result emits all
// eight, always, in classOrder — a stable shape beats a compact one for a
// documented machine payload.
type ClassCount struct {
	Class Class `json:"class"`
	Count int   `json:"count"`
}

// Result is one identity comparison.
type Result struct {
	// Comparable is false when the two snapshots cannot be matched at all;
	// Reason says which check refused. See Compare.
	Comparable bool   `json:"comparable"`
	Reason     string `json:"reason,omitempty"`

	// Notes records the mismatches that were *allowed*. They are not
	// warnings the caller has to act on, but they change how a reader should
	// weigh the result, so they travel with it rather than going to stderr.
	Notes []string `json:"notes,omitempty"`

	OldFunctions int `json:"oldFunctions"`
	NewFunctions int `json:"newFunctions"`

	Counts  []ClassCount `json:"counts"`
	Changes []Change     `json:"changes"`
}

// Compare matches every function in base against every function in head and
// classifies each one.
//
// # Refusals, and the two that are deliberately not refusals
//
// Incomparability is a Result, not an error: Comparable is false and Reason
// says why, mirroring snapshot.Diff's contract so the two surfaces cannot
// disagree about what "these cannot be compared" means. Two checks refuse:
//
//   - Schema. A snapshot older than 6 carries no WL bag and no label
//     dictionary at all, so there is nothing to match on; a later one may
//     mean something different by the bytes it does carry.
//   - RuleSet. canon's rule set decides the canonical tree every WL label is
//     computed over, so the same two untouched bodies produce different bags
//     under two rule sets. Matching across them would report an entire corpus
//     as edited, which is a confidently wrong answer rather than a missing
//     one.
//
// Params, Ontology and Doppel build mismatches are *allowed*, and noted. This
// is deliberately looser than snapshot.Diff, and the reason is that the two
// surfaces read different fields. Diff reads Pairs, whose every number is
// corpus-relative — a different threshold or ontology genuinely makes the two
// runs answers to different questions. This package never reads a Pair. It
// reads Units, and a Unit's identity (package, name), body digest and WL bag
// are all properties of that function's own AST:
//
//   - Params decides which functions enter the corpus and which pairs get
//     retrieved and reported. The pair half is irrelevant here. The
//     population half is not hidden either — comparing a --tests exclude run
//     against a --tests include one reports every test function as new, which
//     is a true statement about the two snapshots, and the note says the
//     params differed so the reader knows why.
//   - Ontology fixes concept relatedness and the overlap score. Neither
//     reaches a bag, a digest, or a name.
//   - The doppel build is the loosest of the three and the note matters most
//     there: a rebuild can move WL hashing without moving Schema or RuleSet,
//     and two plain `go build` binaries both report "(devel)", so this check
//     could not catch that case even if it refused. Refusing on it would
//     block the ordinary use — comparing a release binary's snapshot against
//     a local one — while still not catching the failure it is aimed at.
//
// An unreadable bag or label dictionary is a different thing entirely and is
// returned as an error: the file is corrupt or was not written by this codec.
func Compare(base, head snapshot.Snapshot, opt Options) (Result, error) {
	opt = opt.withDefaults()

	r := Result{
		Comparable:   true,
		OldFunctions: len(base.Units),
		NewFunctions: len(head.Units),
	}
	if why := refuse(base, head); why != "" {
		r.Comparable = false
		r.Reason = why
		r.Counts = zeroCounts()
		return r, nil
	}
	r.Notes = notes(base, head)

	a, err := decode(base, "old")
	if err != nil {
		return Result{}, err
	}
	b, err := decode(head, "new")
	if err != nil {
		return Result{}, err
	}

	bags := make([][]fingerprint.LabelCount, 0, len(a.bags)+len(b.bags))
	bags = append(bags, a.bags...)
	bags = append(bags, b.bags...)
	idf := fingerprint.LabelWeights(bags)

	m := &matcher{a: a, b: b, idf: idf, opt: opt, scores: map[pair]score{}}
	m.generateCandidates()
	m.matchByKey()
	m.matchByDigest()
	m.matchBySimilarity()
	changes := m.classify()

	r.Changes = changes
	r.Counts = countsOf(changes)
	return r, nil
}

// refuse mirrors snapshot.Diff's incomparable(), narrowed to the two checks
// that actually bear on matching bodies. Reason strings are phrased the same
// way, so the two surfaces read alike.
func refuse(base, head snapshot.Snapshot) string {
	switch {
	case base.Schema != head.Schema:
		return fmt.Sprintf("old snapshot schema %d, new schema %d", base.Schema, head.Schema)
	case base.Schema < 6:
		return fmt.Sprintf("snapshot schema %d carries no Weisfeiler-Lehman label bags; identity matching needs schema %d or later", base.Schema, 6)
	case base.RuleSet != head.RuleSet:
		return fmt.Sprintf("old snapshot used canon rule set %s, new used %s", base.RuleSet, head.RuleSet)
	}
	return ""
}

// notes states the mismatches Compare allowed. Sorted by construction: the
// order here is the order they are appended in, which is fixed.
func notes(base, head snapshot.Snapshot) []string {
	var out []string
	if base.Doppel != head.Doppel {
		out = append(out, fmt.Sprintf("built by different doppel versions (%s, %s); label hashing is not versioned separately from the schema", base.Doppel, head.Doppel))
	}
	if base.Ontology != head.Ontology {
		out = append(out, fmt.Sprintf("different ontology versions (%s, %s); identity reads no concept, so this is recorded rather than refused", base.Ontology, head.Ontology))
	}
	if base.Params != head.Params {
		out = append(out, fmt.Sprintf("different analysis params (%+v, %+v); a population change shows up as new and deleted functions", base.Params, head.Params))
	}
	return out
}

// side is one snapshot reduced to what matching reads: its units, in the
// snapshot's own key order, and their decoded bags at the same indices.
type side struct {
	label string
	units []snapshot.Unit
	bags  [][]fingerprint.LabelCount
}

func decode(s snapshot.Snapshot, label string) (side, error) {
	dict, err := fingerprint.DecodeLabelDict(s.Labels)
	if err != nil {
		return side{}, fmt.Errorf("identity: %s snapshot: %w", label, err)
	}
	out := side{label: label, units: s.Units, bags: make([][]fingerprint.LabelCount, len(s.Units))}
	for i, u := range s.Units {
		bag, err := fingerprint.DecodeWLBagIndexed(u.WL, dict)
		if err != nil {
			return side{}, fmt.Errorf("identity: %s snapshot, unit %s: %w", label, u.Key, err)
		}
		out.bags[i] = bag
	}
	return out, nil
}

// pair is one old index and one new index. Both are positions into a side's
// units, which snapshot.Build sorted by key, so a pair is a stable name.
type pair struct{ a, b int }

type score struct{ jaccard, containment float64 }

type matcher struct {
	a, b side
	idf  *fingerprint.LabelIDF
	opt  Options

	// scores holds the exact WLOverlap of every pair any stage looked at.
	// It is a map because it is a memo, never an order: every read is by
	// key and every iteration over it in this file is over a separately
	// sorted slice.
	scores map[pair]score
	cands  []pair // sorted, deduped

	partnerOf  []int // old index -> new index, -1 when unmatched
	partnerRev []int // new index -> old index, -1 when unmatched
	absorbedA  []bool
	absorbedB  []bool
}

// scoreOf returns the exact weighted Jaccard and containment of two bodies,
// memoized. It is the only place a number this package reports comes from.
func (m *matcher) scoreOf(p pair) score {
	if s, ok := m.scores[p]; ok {
		return s
	}
	j, c := fingerprint.WLOverlap(m.a.bags[p.a], m.b.bags[p.b], m.idf)
	s := score{jaccard: j, containment: c}
	m.scores[p] = s
	return s
}

// generateCandidates proposes the pairs worth scoring exactly.
//
// The all-pairs alternative is O(|old| x |new|) bag merges, which is fine on
// a few hundred functions and is minutes on a corpus the size of moby. So
// this is the retrieval shape the rest of the tool already uses: an inverted
// index over labels, accumulate approximate shared mass per counterpart, keep
// each function's top CandidateK.
//
// Run in both directions and unioned, because a top-K kept only from the old
// side would silently drop a new function that is nobody's top choice but has
// exactly one plausible ancestor. The accumulator's mass is an approximation
// (it skips high-df labels and never computes the union denominator) and is
// used for nothing but ordering the top-K: every pair that survives is scored
// again, exactly, by scoreOf.
func (m *matcher) generateCandidates() {
	seen := make(map[pair]struct{})
	add := func(p pair) {
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		m.cands = append(m.cands, p)
	}
	for _, p := range topK(m.a.bags, m.b.bags, m.idf, m.opt) {
		add(pair{a: p[0], b: p[1]})
	}
	for _, p := range topK(m.b.bags, m.a.bags, m.idf, m.opt) {
		add(pair{a: p[1], b: p[0]})
	}
	sort.Slice(m.cands, func(i, j int) bool {
		if m.cands[i].a != m.cands[j].a {
			return m.cands[i].a < m.cands[j].a
		}
		return m.cands[i].b < m.cands[j].b
	})
	for _, p := range m.cands {
		m.scoreOf(p)
	}
}

type posting struct {
	unit  int
	count int32
}

// topK indexes `to` and, for every bag in `from`, returns the CandidateK
// counterparts carrying the most shared informative mass. The result is
// sorted, so nothing downstream sees the map the index is built in.
func topK(from, to [][]fingerprint.LabelCount, idf *fingerprint.LabelIDF, opt Options) [][2]int {
	index := make(map[uint64][]posting)
	for j, bag := range to {
		for _, lc := range bag {
			index[lc.Label] = append(index[lc.Label], posting{unit: j, count: lc.Count})
		}
	}

	acc := make([]float64, len(to))
	var touched []int
	var out [][2]int
	for i, bag := range from {
		for _, t := range touched {
			acc[t] = 0
		}
		touched = touched[:0]
		for _, lc := range bag {
			w := idf.Weight(lc.Label)
			// A label every function in the union carries weighs exactly 0
			// and can move no mass; skipping it is exact, not a heuristic.
			if w == 0 {
				continue
			}
			post := index[lc.Label]
			if len(post) > opt.MaxLabelDF {
				continue
			}
			for _, p := range post {
				if acc[p.unit] == 0 {
					touched = append(touched, p.unit)
				}
				acc[p.unit] += w * float64(min(lc.Count, p.count))
			}
		}
		// Ascending index breaks mass ties, so the kept set depends on
		// nothing but the two bag slices.
		sort.Slice(touched, func(x, y int) bool {
			if acc[touched[x]] != acc[touched[y]] {
				return acc[touched[x]] > acc[touched[y]]
			}
			return touched[x] < touched[y]
		})
		keep := touched
		if len(keep) > opt.CandidateK {
			keep = keep[:opt.CandidateK]
		}
		for _, j := range keep {
			out = append(out, [2]int{i, j})
		}
	}
	return out
}

func (m *matcher) initMatching() {
	if m.partnerOf != nil {
		return
	}
	m.partnerOf = fill(len(m.a.units), -1)
	m.partnerRev = fill(len(m.b.units), -1)
	m.absorbedA = make([]bool, len(m.a.units))
	m.absorbedB = make([]bool, len(m.b.units))
}

func fill(n, v int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func (m *matcher) join(a, b int) {
	m.partnerOf[a] = b
	m.partnerRev[b] = a
	m.scoreOf(pair{a: a, b: b})
}

// matchByKey is the first pass, and the strongest one: a function whose
// snapshot key is unchanged is the same function, whatever its body now says.
// Key is package.Name (disambiguated by file when that collides), so an equal
// key means the same name in the same package, and no similarity number can
// improve on that — a body rewritten from scratch under an unchanged key is
// an edit, not a deletion and an unrelated arrival.
//
// It is also what keeps this package's answer consistent with snapshot.Diff's
// on the cases Diff can see.
func (m *matcher) matchByKey() {
	m.initMatching()
	byKey := make(map[string]int, len(m.b.units))
	for j, u := range m.b.units {
		byKey[u.Key] = j
	}
	// m.a.units is sorted by key (snapshot.Build's contract), so this loop
	// visits in a fixed order; keys are unique within a snapshot, so no
	// contention arises anyway.
	for i, u := range m.a.units {
		j, ok := byKey[u.Key]
		if !ok || m.partnerRev[j] >= 0 {
			continue
		}
		m.join(i, j)
	}
}

// matchByDigest is the second pass: equal non-empty fingerprint digests mean
// the same body under a different name, a different package, or both. It runs
// before similarity matching because it is an identity fact and similarity is
// an estimate — and because a digest bucket with several members must be
// consumed in one documented order rather than by whatever the weighted
// Jaccard happens to tie on (identical bodies score identically against every
// counterpart, so the greedy pass would be deciding by index anyway).
//
// Empty digests never match: a declaration with no body is not evidence of
// anything, which is the rule snapshot.Diff already applies.
func (m *matcher) matchByDigest() {
	m.initMatching()
	buckets := make(map[string][]int)
	for j, u := range m.b.units {
		if m.partnerRev[j] >= 0 || u.Digest == "" {
			continue
		}
		buckets[u.Digest] = append(buckets[u.Digest], j)
	}
	cursor := make(map[string]int, len(buckets))
	for i, u := range m.a.units {
		if m.partnerOf[i] >= 0 || u.Digest == "" {
			continue
		}
		free := buckets[u.Digest]
		k := cursor[u.Digest]
		for k < len(free) && m.partnerRev[free[k]] >= 0 {
			k++
		}
		cursor[u.Digest] = k
		if k >= len(free) {
			continue
		}
		m.join(i, free[k])
		cursor[u.Digest] = k + 1
	}
}

// matchBySimilarity is the greedy bipartite pass over what identity could not
// settle.
//
// Candidates are every generated pair with both sides still free whose exact
// weighted Jaccard clears RenameFloor, consumed in one total order:
//
//	jaccard desc, containment desc, old key asc, new key asc
//
// Greedy rather than optimal (Hungarian, say) on purpose. An assignment that
// maximises total similarity can move a pair off its own best match to
// improve someone else's, which produces a report whose individual lines
// cannot be checked in isolation: "why is A called a rename of B when A looks
// far more like C" would have "because of D" as its answer. Greedy's
// invariant is local and printable — every match was the best still available
// to both sides at the moment it was made — which is the property this whole
// tool is built on.
//
// The floor is applied at admission rather than after matching, so a weak
// pair never consumes a side that a later, better pair could have used.
func (m *matcher) matchBySimilarity() {
	m.initMatching()
	type cand struct {
		p pair
		s score
	}
	list := make([]cand, 0, len(m.cands))
	for _, p := range m.cands {
		if m.partnerOf[p.a] >= 0 || m.partnerRev[p.b] >= 0 {
			continue
		}
		s := m.scoreOf(p)
		if s.jaccard < m.opt.RenameFloor {
			continue
		}
		list = append(list, cand{p: p, s: s})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].s.jaccard != list[j].s.jaccard {
			return list[i].s.jaccard > list[j].s.jaccard
		}
		if list[i].s.containment != list[j].s.containment {
			return list[i].s.containment > list[j].s.containment
		}
		ka, kb := m.a.units[list[i].p.a].Key, m.a.units[list[j].p.a].Key
		if ka != kb {
			return ka < kb
		}
		return m.b.units[list[i].p.b].Key < m.b.units[list[j].p.b].Key
	})
	for _, c := range list {
		if m.partnerOf[c.p.a] >= 0 || m.partnerRev[c.p.b] >= 0 {
			continue
		}
		m.join(c.p.a, c.p.b)
	}
}

// classOf is the precedence order, stated as a total order over three
// independent facts about a matched pair: whether the package changed,
// whether the name changed, and whether the body changed. First rule wins;
// the facts the winning rule did not use are still recorded on the Change and
// printed, so nothing is lost by collapsing to one label.
//
//  1. moved     — the packages differ. A function that changed package is a
//     move whatever else happened to it: relocation is the fact a
//     reader most needs, because it is the one that makes the
//     function unfindable where it was. A move that also renamed
//     or also edited reports Moved with the secondary facts on
//     the line.
//  2. renamed   — same package, different name. An edit alongside a rename
//     reports Renamed, for the same reason: the name is how the
//     function is found.
//  3. edited    — same package, same name, digests differ.
//  4. unchanged — same package, same name, equal digests. Two empty digests
//     (declarations with no body) count as equal, which is the
//     only reading under which a body-less function can ever be
//     reported as unchanged.
func classOf(o, n snapshot.Unit) Class {
	switch {
	case o.Package != n.Package:
		return Moved
	case o.Name != n.Name:
		return Renamed
	case o.Digest != n.Digest:
		return Edited
	}
	return Unchanged
}

// classify turns the matching into findings: the split and merged rules first,
// then every surviving match, then the leftovers.
func (m *matcher) classify() []Change {
	m.initMatching()
	changes := m.detectSplits()
	changes = append(changes, m.detectMerges()...)

	for i, o := range m.a.units {
		if m.absorbedA[i] {
			continue
		}
		j := m.partnerOf[i]
		if j < 0 || m.absorbedB[j] {
			// A partner absorbed by a split or a merge releases this side,
			// which then stands on its own.
			changes = append(changes, Change{Class: Deleted, Old: []Member{m.oldMember(i, score{})}})
			continue
		}
		n := m.b.units[j]
		s := m.scoreOf(pair{a: i, b: j})
		changes = append(changes, Change{
			Class:          classOf(o, n),
			Old:            []Member{m.oldMember(i, s)},
			New:            []Member{m.newMember(j, s)},
			Jaccard:        s.jaccard,
			Containment:    s.containment,
			DigestEqual:    o.Digest == n.Digest,
			NameChanged:    o.Name != n.Name,
			PackageChanged: o.Package != n.Package,
		})
	}
	for j := range m.b.units {
		if m.absorbedB[j] {
			continue
		}
		i := m.partnerRev[j]
		if i >= 0 && !m.absorbedA[i] {
			continue // already reported from the old side
		}
		changes = append(changes, Change{Class: Added, New: []Member{m.newMember(j, score{})}})
	}

	sortChanges(changes)
	return changes
}

// detectSplits finds one old body whose informative mass covers two or more
// new bodies, and detectMerges is its exact inverse.
//
// # The rule, precisely
//
// An old function F is split when at least SplitContainment-worth of two or
// more distinct new bodies is shared with F — that is, when
// fingerprint.WLOverlap(F, G) returns containment >= 0.8 for two or more
// candidate G. Containment there is the shared informative mass over the
// *smaller* side's mass, which is symmetric; for a genuine split the pieces
// are smaller than the original, so the number read is exactly "how much of
// this new piece was already in the old body", which is the claim the class
// makes.
//
// Three exclusions, each because the class would otherwise be false, and each
// decided on digests rather than on a second threshold:
//
//   - F must not survive. If any function in the new snapshot carries F's
//     digest, F's body still exists byte-for-byte somewhere, so it was not
//     divided — something was copied out of it, which is duplication and is
//     what the rest of this tool reports.
//   - No participating G may carry F's digest either. A piece identical to
//     the whole is not a piece. This is the exclusion that keeps an
//     exact-clone family from reading as a split: when the corpus already
//     held three copies of one body and one of them was renamed, the rule's
//     bare form finds an old body covering two new ones at containment
//     1.0000 each and calls it a split, which is a confidently wrong answer
//     about code nobody divided.
//   - No participating G may be unchanged. A helper that already existed and
//     was merely reused is not something F produced.
//
// Everything else is eligible, matched or not. A function can be edited *and*
// split, and "an unmatched-or-even-matched old body" is exactly that case:
// the largest piece usually keeps the original's name and therefore matches
// it in an earlier pass, and requiring the residue would make the rule fire
// only on the splits that renamed everything.
//
// When the rule fires, F and every participating G are absorbed: they are
// reported once, in the split, and nowhere else. F's own match is dissolved.
// If F's partner is not among the participants it is released and stands
// alone as a new function, which is honest — nothing claims it descends from
// F any more.
//
// Splits are detected before merges, over old functions in key order, and a
// function absorbed by a split cannot join a merge. That ordering is the
// tie-break that makes the two rules a total order rather than a race.
func (m *matcher) detectSplits() []Change {
	byOld := make(map[int][]int)
	for _, p := range m.cands {
		byOld[p.a] = append(byOld[p.a], p.b)
	}
	survives := digestSet(m.b.units)
	var out []Change
	for i := range m.a.units {
		pivot := m.a.units[i]
		if m.absorbedA[i] || survives[pivot.Digest] {
			continue
		}
		var parts []int
		// byOld's slices come from m.cands, which is sorted, so parts comes
		// out in ascending new index.
		for _, j := range byOld[i] {
			if m.absorbedB[j] || m.isUnchangedNew(j) || sameBody(pivot, m.b.units[j]) {
				continue
			}
			if m.scoreOf(pair{a: i, b: j}).containment >= m.opt.SplitContainment {
				parts = append(parts, j)
			}
		}
		if len(parts) < 2 {
			continue
		}
		out = append(out, m.absorbSplit(i, parts))
	}
	return out
}

func (m *matcher) detectMerges() []Change {
	byNew := make(map[int][]int)
	for _, p := range m.cands {
		byNew[p.b] = append(byNew[p.b], p.a)
	}
	survived := digestSet(m.a.units)
	var out []Change
	for j := range m.b.units {
		pivot := m.b.units[j]
		if m.absorbedB[j] || survived[pivot.Digest] {
			continue
		}
		var parts []int
		for _, i := range byNew[j] {
			if m.absorbedA[i] || m.isUnchanged(i) || sameBody(pivot, m.a.units[i]) {
				continue
			}
			if m.scoreOf(pair{a: i, b: j}).containment >= m.opt.SplitContainment {
				parts = append(parts, i)
			}
		}
		if len(parts) < 2 {
			continue
		}
		out = append(out, m.absorbMerge(j, parts))
	}
	return out
}

// digestSet indexes the non-empty body digests one side carries, so "did this
// body survive anywhere" is a lookup rather than a scan. The empty digest is
// never a member: a declaration with no body is not evidence that any body
// survived.
func digestSet(us []snapshot.Unit) map[string]bool {
	set := make(map[string]bool, len(us))
	for _, u := range us {
		if u.Digest != "" {
			set[u.Digest] = true
		}
	}
	return set
}

// sameBody reports whether two units are the same body under possibly
// different names. Empty digests never match, mirroring snapshot.Diff.
func sameBody(a, b snapshot.Unit) bool {
	return a.Digest != "" && a.Digest == b.Digest
}

func (m *matcher) absorbSplit(i int, parts []int) Change {
	c := Change{Class: Split, Old: []Member{m.oldMember(i, score{})}}
	m.absorbedA[i] = true
	for _, j := range parts {
		c.New = append(c.New, m.newMember(j, m.scoreOf(pair{a: i, b: j})))
		m.absorbedB[j] = true
	}
	sortMembers(c.New)
	return c
}

func (m *matcher) absorbMerge(j int, parts []int) Change {
	c := Change{Class: Merged, New: []Member{m.newMember(j, score{})}}
	m.absorbedB[j] = true
	for _, i := range parts {
		c.Old = append(c.Old, m.oldMember(i, m.scoreOf(pair{a: i, b: j})))
		m.absorbedA[i] = true
	}
	sortMembers(c.Old)
	return c
}

// isUnchanged reports whether old function i survived into the new snapshot
// byte-identically — the one state the split and merged rules refuse to build
// on.
func (m *matcher) isUnchanged(i int) bool {
	j := m.partnerOf[i]
	return j >= 0 && classOf(m.a.units[i], m.b.units[j]) == Unchanged
}

func (m *matcher) isUnchangedNew(j int) bool {
	i := m.partnerRev[j]
	return i >= 0 && classOf(m.a.units[i], m.b.units[j]) == Unchanged
}

func (m *matcher) oldMember(i int, s score) Member {
	return Member{Function: functionOf(m.a.units[i]), Jaccard: s.jaccard, Containment: s.containment}
}

func (m *matcher) newMember(j int, s score) Member {
	return Member{Function: functionOf(m.b.units[j]), Jaccard: s.jaccard, Containment: s.containment}
}

func functionOf(u snapshot.Unit) Function {
	return Function{Key: u.Key, Package: u.Package, Name: u.Name, File: u.File, Line: u.Line, Digest: u.Digest}
}

func sortMembers(ms []Member) {
	sort.Slice(ms, func(i, j int) bool { return ms[i].Key < ms[j].Key })
}

// sortChanges is the report's total order: class first (classOrder, so the
// grouped renderer walks the slice once), then the old side's key, then the
// new side's. A change with no old side sorts on its new key in the same
// position, which is well-defined because Added is the only such class and it
// never shares a class rank with another.
func sortChanges(cs []Change) {
	sort.Slice(cs, func(i, j int) bool {
		if ri, rj := classRank(cs[i].Class), classRank(cs[j].Class); ri != rj {
			return ri < rj
		}
		if ki, kj := primaryKey(cs[i]), primaryKey(cs[j]); ki != kj {
			return ki < kj
		}
		return secondaryKey(cs[i]) < secondaryKey(cs[j])
	})
}

func primaryKey(c Change) string {
	if len(c.Old) > 0 {
		return c.Old[0].Key
	}
	if len(c.New) > 0 {
		return c.New[0].Key
	}
	return ""
}

func secondaryKey(c Change) string {
	if len(c.New) > 0 {
		return c.New[0].Key
	}
	return ""
}

func zeroCounts() []ClassCount {
	out := make([]ClassCount, 0, len(classOrder))
	for _, c := range classOrder {
		out = append(out, ClassCount{Class: c})
	}
	return out
}

func countsOf(changes []Change) []ClassCount {
	out := zeroCounts()
	for _, ch := range changes {
		out[classRank(ch.Class)].Count++
	}
	return out
}

// Count returns how many findings carry a class.
func (r Result) Count(c Class) int {
	for _, cc := range r.Counts {
		if cc.Class == c {
			return cc.Count
		}
	}
	return 0
}

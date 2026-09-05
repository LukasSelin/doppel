package analyzer

import (
	"fmt"
	"slices"
	"strings"

	"github.com/LukasSelin/doppel/internal/canon"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// Explain says, in one sentence, why two bodies came out as alike as they did
// — which normalizations were needed to bring them together, or what is left
// between them once those normalizations have run.
//
// It annotates and nothing else. No score reads it, no ranking key contains
// it, no filter consults it: it is a string attached to a pair after every
// number about that pair has been decided. That is the same contract culture
// notes, habitat notes, profiles and kinds hold, and it is the reason this
// can be a *readable* claim rather than a defensible one — a sentence that
// ranked something would have to be a number.
//
// # The two tiers
//
// A pair is either shape-identical after canonicalization, or it is not.
//
//	identical after rename, commutative-reorder
//	differs by one extra defer, two extra if
//
// The first form names the canon rules that fired on either side, in canon's
// declaration order, under friendly names. It is the answer to "these two
// score 1.00 — what was actually different about them?", and it is checkable:
// every rule named fired on at least one of the two functions, and the
// canonical trees agree, so undoing any of them is the only thing that could
// put a difference back. That is what "a rule whose reversal would lower the
// score" means here, and it is asserted rather than measured: literally
// reversing a rule would mean re-canonicalizing under a modified rule set and
// re-scoring, per rule, per pair.
//
// One refinement is deliberately skipped. A rule that fired on one function
// and would have been a no-op on the other still gets credit, because
// distinguishing the two needs the per-rule before/after trees that
// canon.Canonicalize does not keep. The claim stays true as stated — the rule
// did fire, and the trees do agree — it is just not minimal.
//
// # Shape-identical is decided on the WL bags
//
// Equal bags is a proxy for equal canonical trees, and a cheap one: the bags
// are already built, already sorted, and comparing them is a walk of two
// slices. It is a proxy in one direction only — two different trees can share
// a bag by hash collision, or by being genuinely WL-indistinguishable, which
// is the known limit of the refinement. Both would overstate the tier-one
// claim on one pair's sentence. Neither can move a number, since the score,
// the ranking and the filters were all decided before this runs.
func Explain(a, b parser.CodeUnit) string { return ExplainWith(a, b, nil) }

// ExplainWith is Explain against a naming table built once for a whole corpus
// rather than once per pair. A nil table means build one from these two units,
// which is exactly what Explain does and what every test pins.
//
// The seam exists because the per-pair form's cost premise stopped being true:
// see BuildLabelKinds.
func ExplainWith(a, b parser.CodeUnit, lk *LabelKinds) string {
	abag, bbag := a.Fingerprint.WL, b.Fingerprint.WL
	if len(abag) == 0 || len(bbag) == 0 {
		return "no body to compare"
	}
	if slices.Equal(abag, bbag) {
		rules := firedUnion(a.CanonRules, b.CanonRules)
		if len(rules) == 0 {
			return "identical structure, no normalization needed"
		}
		return "identical after " + strings.Join(rules, ", ")
	}
	return residual(a, b, lk)
}

// firedUnion maps the rules that fired on either side to their friendly names,
// in canon's declaration order.
//
// Declaration order rather than firing order, matching canon.Result.Fired's
// own choice: what a reader wants is which normalizations were needed, and the
// round a rule happened to catch in is an artefact of the fixed-point loop.
// The IDs arrive as plain strings, because syntax.Func carries them that way:
// a canonicalizer belongs to a frontend, so the IR cannot be typed on any one
// frontend's rule enum. canon's IDs are documented as stable API precisely so
// that projection is lossless, and a rule from a canonicalizer this switch
// does not know renders under its own name.
func firedUnion(as, bs []string) []string {
	fired := make(map[string]bool, len(as)+len(bs))
	for _, id := range as {
		fired[id] = true
	}
	for _, id := range bs {
		fired[id] = true
	}
	var out []string
	for _, r := range canon.Rules() {
		if fired[string(r.ID)] {
			out = append(out, CanonWord(string(r.ID)))
			delete(fired, string(r.ID))
		}
	}
	// Anything left fired under a canonicalizer canon does not own. Sorted,
	// because there is no declaration order to inherit for it.
	rest := make([]string, 0, len(fired))
	for id := range fired {
		rest = append(rest, id)
	}
	slices.Sort(rest)
	return append(out, rest...)
}

// CanonWord is the word a report uses for a canonicalization rule.
//
// The rule IDs are the vocabulary canon publishes and are stable API there;
// these are the reader-facing half, and they say what the normalization did
// to the *reader's* code rather than what the rewrite was called. An ID with
// no entry renders as itself, so a rule added to canon appears in explanations
// immediately, under its own name, rather than vanishing from them.
//
// Exported for the fingerprint view, which names the rules that fired on one
// body; one table, so a rule cannot be called one thing in an explain
// sentence and another in the view. It takes the plain string
// parser.CodeUnit.CanonRules carries rather than a canon.RuleID, because the
// view lives in reporter and reporter must not import canon — that package
// is go/ast-typed and reaches the rest of the tool only through gofront and
// this one.
func CanonWord(id string) string {
	switch canon.RuleID(id) {
	case canon.RuleAlphaRename:
		return "rename"
	case canon.RuleUnwrapBlock:
		return "block-unwrap"
	case canon.RuleNegatedIf:
		return "negation-flip"
	case canon.RuleGuardReturn:
		return "guard-form"
	case canon.RuleIncDec:
		return "increment-form"
	case canon.RuleCommutativeSort:
		return "commutative-reorder"
	}
	return string(id)
}

// explainMaxKinds bounds the residual sentence. An explanation is a sentence,
// not a dump: past three kinds a reader has stopped reading and the rest is
// better stated as a count.
const explainMaxKinds = 3

// residual describes what is left between two bodies whose canonical trees
// did not converge.
//
// # It counts h=0 labels and detects on h=1
//
// An h=0 label is a node's kind and nothing else, so the h=0 multiset
// difference is exactly "one more defer, two more ifs" — the quantity the
// sentence claims. The h=1 labels are not counted, and that is a decision
// rather than an omission: an h=1 label folds in a node's immediate children,
// so a single extra statement changes the label of its parent, its
// grandparent and every enclosing block. Counting those too would report one
// added defer as "one extra defer, one extra for, one extra block", which is
// three true statements adding up to a false impression.
//
// What h=1 is good for is the case the h=0 counts cannot see. When the kind
// multisets agree exactly, the two bodies use the same statements in a
// different arrangement, and h=1 says whether that rearrangement is local —
// some node has different immediate children — or further out, beyond what
// these shallow labels can name.
//
// # Sides are not named
//
// The difference is symmetric: "one extra defer" means the two bodies differ
// by one defer, not that A has it. Naming the side would double the sentence's
// length to carry a fact the reader gets from opening either file, and a
// residual mixing extras in both directions reads as a contradiction when
// each half is attributed.
func residual(a, b parser.CodeUnit, lk *LabelKinds) string {
	kinds := lk.lookup()
	if kinds == nil {
		if a.Canonical == nil && b.Canonical == nil {
			// Neither a table nor a tree to name the difference with. The
			// pipeline releases the canonical trees once its corpus-wide table
			// is built (see cmd.analyze), so a caller of the free Explain over
			// units that have been through it lands here. Say only what is
			// still true rather than falling through to the h=0 tier, which
			// would read an empty diff as "the same statement kinds" and be
			// confidently wrong.
			return "structures differ; no canonical tree to name the difference"
		}
		kinds = labelKinds(a.Canonical, b.Canonical)
	}

	counts := make(map[string]int)
	diffAtH1 := false
	for _, d := range bagDiff(a.Fingerprint.WL, b.Fingerprint.WL) {
		lk, ok := kinds[d.label]
		if !ok {
			// A label from a bag whose canonical tree this process no longer
			// holds — h >= 2, or a unit that reached here without one. Deep
			// labels are the common case and are not nameable anyway.
			continue
		}
		switch lk.H {
		case 0:
			counts[lk.Kind] += d.n
		case 1:
			diffAtH1 = true
		}
	}

	if len(counts) == 0 {
		if diffAtH1 {
			return "same statement kinds, different local shapes"
		}
		return "same statement kinds, different arrangement"
	}
	return "differs by " + strings.Join(kindPhrases(counts), ", ")
}

// kindPhrases renders the counted residual as ordered phrases, capped.
func kindPhrases(counts map[string]int) []string {
	type row struct {
		kind string
		n    int
	}
	rows := make([]row, 0, len(counts))
	for kind, n := range counts {
		rows = append(rows, row{kind: kind, n: n})
	}
	// Statement kinds lead, then expressions, then everything else; within a
	// tier the larger difference leads and the kind name breaks the tie. The
	// tiers exist because the raw counts are dominated by identifiers and
	// blocks — true, and never what a reader wanted told first.
	slices.SortFunc(rows, func(x, y row) int {
		if rx, ry := kindTier(x.kind), kindTier(y.kind); rx != ry {
			return rx - ry
		}
		if x.n != y.n {
			return y.n - x.n
		}
		return strings.Compare(x.kind, y.kind)
	})

	shown := rows
	rest := 0
	if len(shown) > explainMaxKinds {
		rest = len(shown) - explainMaxKinds
		shown = shown[:explainMaxKinds]
	}
	out := make([]string, 0, len(shown)+1)
	for _, r := range shown {
		out = append(out, fmt.Sprintf("%s extra %s", countWord(r.n), r.kind))
	}
	if rest == 1 {
		out = append(out, "and 1 more kind")
	} else if rest > 1 {
		out = append(out, fmt.Sprintf("and %d more kinds", rest))
	}
	return out
}

// statementKinds and expressionKinds are the display names — fingerprint's
// kindName output — of the node kinds a reader recognises as a statement and
// as an expression. Anything absent from both is scaffolding (blocks, field
// lists, bare identifiers, and the statement wrappers around an expression or
// a declaration) and sorts last: real, countable, and never the headline.
//
// The wrappers are the reason "expression statement" and "declaration
// statement" are absent here rather than listed. Each one always accompanies
// the thing it wraps, so admitting them would let a sentence spend two of its
// three slots saying the same difference twice.
var statementKinds = map[string]bool{
	"assign": true, "branch": true, "case": true, "comm": true,
	"declaration": true, "defer": true, "empty": true,
	"for": true, "go": true, "if": true, "increment": true,
	"labeled": true, "range": true, "return": true, "select": true,
	"send": true, "switch": true, "typeswitch": true,
}

var expressionKinds = map[string]bool{
	"binary": true, "call": true, "composite literal": true, "ellipsis": true,
	"function literal": true, "index": true, "indexlist": true, "key-value": true,
	"literal": true, "paren": true, "selector": true, "slice": true,
	"star": true, "type assertion": true, "unary": true,
}

func kindTier(kind string) int {
	switch {
	case statementKinds[kind]:
		return 0
	case expressionKinds[kind]:
		return 1
	}
	return 2
}

// countWord spells the small counts a residual almost always has. Past nine
// the numeral is shorter and reads no worse.
func countWord(n int) string {
	words := [...]string{"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"}
	if n > 0 && n < len(words) {
		return words[n]
	}
	return fmt.Sprint(n)
}

// labelKinds is the label -> (round, kind) table for one pair, built from the
// two units' own canonical trees. It is the fallback path: the free Explain,
// and every caller holding two units and no corpus.
//
// It was once the only path, on the reasoning that a shared table would be "a
// second index over every label in the corpus, kept alive for a sentence
// printed on a few dozen pairs". The premise was measured and is false — the
// pipeline annotates every surviving pair, 18 461 of them on moby, so this ran
// 36 922 times over at most 7 658 distinct trees and Explain became the single
// largest allocator in the tool at 797MB, 21% of a run. BuildLabelKinds is the
// alternative that comment rejected, adopted on that measurement.
//
// A collision between the two sides resolves to A's naming, since A is merged
// first. Both name the same kind unless the hash collided, and the ordering
// makes the outcome the same on every run either way.
func labelKinds(a, b *syntax.Node) map[uint64]fingerprint.LowLabel {
	out := make(map[uint64]fingerprint.LowLabel)
	for _, tree := range []*syntax.Node{a, b} {
		for _, lk := range fingerprint.LowLabels(tree) {
			if _, seen := out[lk.Label]; !seen {
				out[lk.Label] = lk
			}
		}
	}
	return out
}

// labelDelta is one label the two bags disagree about, and by how much.
type labelDelta struct {
	label uint64
	n     int
}

// bagDiff is the min-count multiset difference of two WL bags: every label
// whose counts disagree, with the size of the disagreement. Both bags are
// sorted ascending by label — WLBag's contract — so this is one merge.
func bagDiff(a, b []fingerprint.LabelCount) []labelDelta {
	var out []labelDelta
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i].Label < b[j].Label:
			out = append(out, labelDelta{label: a[i].Label, n: int(a[i].Count)})
			i++
		case a[i].Label > b[j].Label:
			out = append(out, labelDelta{label: b[j].Label, n: int(b[j].Count)})
			j++
		default:
			if d := a[i].Count - b[j].Count; d != 0 {
				if d < 0 {
					d = -d
				}
				out = append(out, labelDelta{label: a[i].Label, n: int(d)})
			}
			i++
			j++
		}
	}
	for ; i < len(a); i++ {
		out = append(out, labelDelta{label: a[i].Label, n: int(a[i].Count)})
	}
	for ; j < len(b); j++ {
		out = append(out, labelDelta{label: b[j].Label, n: int(b[j].Count)})
	}
	return out
}

// LabelKinds is the label -> (round, kind) naming table residual reads, built
// once over a corpus instead of once per pair.
//
// The zero value and a nil pointer both mean "no table", which is what makes
// ExplainWith(a, b, nil) identical to Explain.
type LabelKinds struct {
	m map[uint64]fingerprint.LowLabel
}

func (lk *LabelKinds) lookup() map[uint64]fingerprint.LowLabel {
	if lk == nil {
		return nil
	}
	return lk.m
}

// BuildLabelKinds builds the naming table over the units at the given indices,
// which the caller supplies so that only the units actually appearing in a pair
// are walked.
//
// # Why this is not the per-pair table repeated
//
// A unit's h<=1 labels are a pure function of its own canonical tree, so the
// per-pair form recomputed the same walk once per pair the unit appears in.
// This walks each unit once. The lookup itself cannot gain a hit from the
// widening: wlHash takes the round as an input, so an h>=2 label in a bag can
// only collide with an h<=1 label from another unit by 64-bit hash collision —
// the same caveat Explain's bag-equality tier already carries and the same one
// that made a per-pair collision resolve arbitrarily.
//
// # Determinism
//
// idx is sorted ascending and merged first-wins, so a collision between two
// units resolves to the lower index. The per-pair form resolved it to A. Both
// are fixed orders; they differ only on a genuine hash collision between two
// distinct (round, kind) pairs, which is the case neither form can name
// correctly anyway.
// A nil idx names every unit, which is what a caller that intends to release
// the canonical trees afterwards wants: the table has to be complete before the
// trees it was derived from can go.
func BuildLabelKinds(units []parser.CodeUnit, idx []int) *LabelKinds {
	lk := &LabelKinds{m: make(map[uint64]fingerprint.LowLabel)}
	if idx == nil {
		idx = make([]int, len(units))
		for i := range idx {
			idx[i] = i
		}
	}
	seen := make(map[int]bool, len(idx))
	ordered := append([]int(nil), idx...)
	slices.Sort(ordered)
	for _, i := range ordered {
		if i < 0 || i >= len(units) || seen[i] {
			continue
		}
		seen[i] = true
		for _, l := range fingerprint.LowLabels(units[i].Canonical) {
			if _, dup := lk.m[l.Label]; !dup {
				lk.m[l.Label] = l
			}
		}
	}
	return lk
}

// Package canon rewrites a Go function into a canonical shape, so that two
// functions which differ only in incidental choices — variable names, the
// order of operands around a commutative operator, whether a guard was
// written as an early return or as an else — reduce to the same tree.
//
// It is a leaf package by construction: it imports nothing from this module
// and works on *ast.FuncDecl directly, exactly as fingerprint does. parser
// calls it during parsing, while the AST is in hand.
//
// # Not semantics-preserving
//
// This is a *similarity* canonicalization, not a refactoring. The canonical
// tree is a comparison key; it is never compiled, never printed back into
// the user's source, and never presented as a suggested rewrite. Several
// rules change what the code would do if it were run:
//
//   - Sorting the operands of && and || discards short-circuit order, so a
//     nil check that guarded the operand next to it no longer guards it.
//   - Sorting the operands of + reverses string concatenation, and reorders
//     any operand whose evaluation has a side effect.
//   - Swapping the branches of a negated if, and lifting an else block out
//     from behind an early return, both preserve control flow — but only
//     because the rules refuse to fire when they would not.
//
// Two functions that canonicalize to the same tree are alike in shape. They
// are not thereby interchangeable, and nothing downstream may claim they are.
//
// # Rules, in application order
//
// The order below is the declaration order of Rules(), which is also the
// order the fixed-point loop applies them in. It is fixed rather than
// incidental, because the rules interact:
//
//  1. RuleAlphaRename     — bound identifiers become x0, x1, … in binding order.
//  2. RuleUnwrapBlock     — a bare { … } inside a block is spliced into it.
//  3. RuleNegatedIf       — if !c {A} else {B} becomes if c {B} else {A}.
//  4. RuleGuardReturn     — an if/else with one leaving branch becomes an early return.
//  5. RuleIncDec          — x = x + 1 and x += 1 become x++ (likewise --).
//  6. RuleCommutativeSort — operands of + * == != & | ^ && || are ordered.
//
// Alpha-renaming leads because the commutative sort orders operands by their
// rendered form, and renaming changes that form: sorting first would freeze
// an order that reflects the original names. RuleNegatedIf precedes
// RuleGuardReturn so that `if !c {A} else {return}` reaches the early-return
// form in one round rather than two. RuleIncDec precedes RuleCommutativeSort
// for the same reason renaming does, and additionally matches `1 + x` as well
// as `x + 1`, so it is correct whichever side the sort would have chosen.
//
// The two branch rules are the pair that interact, and the interaction is
// load-bearing rather than incidental: RuleNegatedIf declines to swap a
// guard that already sits in the then-branch, because doing so would move it
// where RuleGuardReturn can no longer lift anything out. Each rule's doc
// carries the argument; changing one without the other reintroduces a case
// where the same guard written two ways canonicalizes two ways.
//
// # Termination
//
// Every rule is individually terminating, each on its own strictly
// decreasing measure: RuleUnwrapBlock removes a block, RuleNegatedIf removes
// a leading "!" from an if condition, RuleGuardReturn removes an else,
// RuleIncDec removes an assignment, RuleCommutativeSort is a single
// bottom-up pass that leaves every commutative node ordered, and
// RuleAlphaRename is a fixed point after one application because binding
// order is preserved by every other rule except the branch swap.
//
// The loop nonetheless carries a bound — maxRounds(fd) = AST depth + 8 —
// because the rules interact and a proof of joint termination is not a thing
// this package can assert. Depth is the right shape for the constant: a
// cascade (an else-if chain collapsing into successive early returns) needs
// at most one round per nesting level, and the +8 covers the rename/swap
// interaction and leaves slack. Hitting the bound is a bug. In tests it is
// asserted against; in production the loop stops and records Result.Capped
// rather than panicking, because a report is worth more than a crash.
package canon

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
)

// Version identifies this rule set. It changes whenever a rule is added,
// removed, or altered in a way that can move a canonical tree. Consumers
// that persist canonical forms compare on it and refuse to compare across
// versions — two trees canonicalized under different rule sets answer
// different questions.
const Version = "1"

// A RuleID names one rewrite. The strings are stable API: they are recorded
// per function, will be serialized, and are rendered in explanations. Rename
// a rule and every stored record of it becomes unreadable.
type RuleID string

const (
	RuleAlphaRename     RuleID = "alpha-rename"
	RuleUnwrapBlock     RuleID = "unwrap-block"
	RuleNegatedIf       RuleID = "negated-if"
	RuleGuardReturn     RuleID = "guard-return"
	RuleIncDec          RuleID = "incdec"
	RuleCommutativeSort RuleID = "commutative-sort"
)

// A Rule is one named rewrite over a function declaration. Apply mutates the
// declaration in place and reports whether it changed anything; a rule that
// reports false must have left the tree untouched, because that is what ends
// the fixed-point loop.
type Rule struct {
	ID    RuleID
	Doc   string
	Apply func(*ast.FuncDecl) bool
}

// Rules returns the rule set in application order. The slice is rebuilt per
// call so a caller cannot reorder the shared one.
func Rules() []Rule {
	return []Rule{
		{
			ID:    RuleAlphaRename,
			Doc:   "identifiers bound inside the function become x0, x1, … in first-binding order, parameters first",
			Apply: alphaRename,
		},
		{
			ID:    RuleUnwrapBlock,
			Doc:   "a bare block statement is spliced into the statement list containing it",
			Apply: unwrapBlocks,
		},
		{
			ID:    RuleNegatedIf,
			Doc:   "if !c { A } else { B } becomes if c { B } else { A }",
			Apply: unnegateIfs,
		},
		{
			ID:    RuleGuardReturn,
			Doc:   "an if/else with exactly one leaving branch becomes an early return followed by the other",
			Apply: flattenGuards,
		},
		{
			ID:    RuleIncDec,
			Doc:   "x = x + 1, x = 1 + x and x += 1 become x++ (and the -- forms likewise)",
			Apply: toIncDec,
		},
		{
			ID:    RuleCommutativeSort,
			Doc:   "operands of + * == != & | ^ && || are placed in rendered order",
			Apply: sortCommutative,
		},
	}
}

// Result is one canonicalization: the rewritten tree, which rules fired, and
// how the loop ended.
type Result struct {
	// Decl is the canonical tree — always a deep copy, never the input.
	// It is nil when the input was nil or had no body.
	Decl *ast.FuncDecl

	// Fired lists the rules that changed something, in application order,
	// deduplicated. A rule that fired in three separate rounds appears once:
	// what a reader wants to know is which normalizations this function
	// needed, not how many rounds the loop took to settle.
	Fired []RuleID

	// Rounds is how many passes over the rule set ran, including the final
	// pass that changed nothing.
	Rounds int

	// Capped is true when the loop hit its iteration bound without reaching
	// a fixed point. Always false in practice; see the package doc.
	Capped bool
}

// Canonicalize deep-copies fd and rewrites the copy to a fixed point.
//
// The input is never mutated. That is not a convenience: parser hands the
// same *ast.FuncDecl to fingerprint.Build and to the tagger's signal
// extractor, so an in-place rewrite here would silently change every score
// and every tag in the tool.
func Canonicalize(fd *ast.FuncDecl) Result {
	if fd == nil || fd.Body == nil {
		return Result{}
	}
	decl := Clone(fd)
	res := Result{Decl: decl}

	rules := Rules()
	var fired []RuleID
	seen := make(map[RuleID]bool, len(rules))
	limit := maxRounds(decl)
	for res.Rounds < limit {
		res.Rounds++
		changed := false
		for _, r := range rules {
			if r.Apply(decl) {
				changed = true
				if !seen[r.ID] {
					seen[r.ID] = true
					fired = append(fired, r.ID)
				}
			}
		}
		if !changed {
			res.Fired = orderFired(rules, seen, fired)
			return res
		}
	}
	res.Capped = true
	res.Fired = orderFired(rules, seen, fired)
	return res
}

// orderFired returns the fired rules in declaration order rather than
// first-firing order, so the recorded list reads like the documented
// pipeline and does not depend on which round a rule happened to catch.
func orderFired(rules []Rule, seen map[RuleID]bool, fired []RuleID) []RuleID {
	if len(fired) == 0 {
		return nil
	}
	out := make([]RuleID, 0, len(fired))
	for _, r := range rules {
		if seen[r.ID] {
			out = append(out, r.ID)
		}
	}
	return out
}

// maxRounds is the iteration bound: the depth of the declaration plus a
// constant. See the package doc for why depth is the right shape.
func maxRounds(fd *ast.FuncDecl) int {
	return depth(fd) + 8
}

// depth returns the maximum nesting depth of the AST, counting every node.
func depth(n ast.Node) int {
	if n == nil {
		return 0
	}
	best := 0
	var stack []int
	cur := 0
	ast.Inspect(n, func(node ast.Node) bool {
		if node == nil {
			if last := len(stack) - 1; last >= 0 {
				cur = stack[last]
				stack = stack[:last]
			}
			return false
		}
		stack = append(stack, cur)
		cur++
		if cur > best {
			best = cur
		}
		return true
	})
	return best
}

// Print renders a canonical tree as text. It is the comparison form used by
// the idempotence property test and the natural thing to hash later: the
// tree carries no positions, so go/printer lays it out canonically and two
// structurally equal trees print identically.
func Print(fd *ast.FuncDecl) string {
	if fd == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), fd); err != nil {
		return ""
	}
	return buf.String()
}

package analyzer

import (
	"slices"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
)

// KindSubsystemCopies labels two near-duplicate functions in different
// packages, each called, whose callers live in disjoint sets of packages:
// every subsystem that needs the thing calls its own copy, and no caller
// anywhere has seen both. That is erosion by locally reasonable edit in its
// plainest form — writing a helper next to its first caller was cheaper than
// finding the one two packages over — and it is the case the merge verdict
// cannot see on its own, because shared callers and a shared package are
// exactly what such a pair lacks, so its architectural overlap reads low.
//
// It is the production form of the caller-split census in bench's
// TestCalleeDrift, which found it the one split shape that consistently
// surfaced copies a maintainer would consolidate — doppel's own
// concepter.QualifiedName beside four local qualifiedNames, hugo's
// helpers.IsWhitespace beside parse.isSpace. Same-package splits were sibling
// accessors (Flags beside PersistentFlags) and are left to the other kinds.
//
// Like every kind it annotates and never ranks or filters.
const KindSubsystemCopies = "subsystem copies"

// SubsystemCopies labels a pair of near-duplicate plain functions (code-shape
// at or above forkFloor, the floor the fork rule and the family edge cut read)
// declared in different packages, both called from inside the corpus, by
// callers whose packages share nothing — where at least one side is already
// called from outside its own package, and neither calls the other.
//
// Each condition was measured in TestCalleeDrift and each removes a class that
// is not this finding:
//
//   - Both sides called only from their own package is each package keeping a
//     private helper — ipvlan beside macvlan, one discovery plugin beside the
//     next. Those are parallel implementations, and a package's privacy is the
//     locality a merge would break.
//   - Methods are bound to their types. Two one-line setters on unrelated types
//     read alike at 0.85 and are not one helper said twice; a method cannot be
//     the shared helper a local copy should have called.
//   - A function delegating to its twin is a wrapper, not a second copy.
//
// Packages are compared by package clause, the key the call graph's qualified
// names carry; two package mains in different directories therefore count as
// one caller package, which errs toward silence.
func SubsystemCopies(a, b parser.CodeUnit, score, forkFloor float64, ctx PairContext) *KindNote {
	if score < forkFloor || relation(a, b) == RelationSamePackage {
		return nil
	}
	if a.ReceiverType != "" || b.ReceiverType != "" {
		return nil
	}
	if len(ctx.CallersA) == 0 || len(ctx.CallersB) == 0 {
		return nil
	}
	if !disjoint(ctx.CallerPkgsA, ctx.CallerPkgsB) {
		return nil
	}
	if !reachesOut(a.Package, ctx.CallerPkgsA) && !reachesOut(b.Package, ctx.CallerPkgsB) {
		return nil
	}
	qa, qb := concepter.QualifiedName(a), concepter.QualifiedName(b)
	if slices.Contains(ctx.CallersA, qb) || slices.Contains(ctx.CallersB, qa) {
		return nil
	}
	return &KindNote{
		Kind:           KindSubsystemCopies,
		Names:          []string{qa, qb},
		Packages:       distinct(a.Package, b.Package),
		Relation:       relation(a, b),
		CallerCounts:   []int{len(ctx.CallersA), len(ctx.CallersB)},
		CallerPackages: [][]string{slices.Clone(ctx.CallerPkgsA), slices.Clone(ctx.CallerPkgsB)},
	}
}

// reachesOut reports whether anything outside pkg calls the function: it
// is already some other package's helper, not only its own.
func reachesOut(pkg string, callerPkgs []string) bool {
	for _, p := range callerPkgs {
		if p != pkg {
			return true
		}
	}
	return false
}

// disjoint reports whether two sets share no element. The inputs are small
// (one function's caller packages), so a scan is cheaper than relying on a
// sort order every producer would have to keep.
func disjoint(a, b []string) bool {
	for _, x := range a {
		if slices.Contains(b, x) {
			return false
		}
	}
	return true
}

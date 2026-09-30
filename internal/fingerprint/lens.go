package fingerprint

import "github.com/LukasSelin/doppel/internal/syntax"

// Lens is one way of reading a function's tree into a Weisfeiler–Lehman bag:
// a set of WLOptions, and which tree to read — the canonical shape or the body
// as written.
//
// # Why several, and why never one blend
//
// The production bag answers one question — do these two bodies share shape
// once canonicalization has removed incidental choices — and a single number
// cannot say *which* sameness a pair has. A verbatim copy, a renamed copy, a
// template instantiated over different APIs and a gather against its scatter
// all score code-shape 1.00 or near it, and they are four different findings.
// Reading the same tree under several lenses separates them, because each
// lens forgets a different thing:
//
//	skeleton  canonical, callee names dropped     same structure, any API
//	shape     canonical, production labels        the code-shape score's bag
//	ordered   canonical, assignment sides kept    shape with data direction
//	vocab     canonical, names and values kept    shape over the same vocabulary
//	verbatim  as written, names and values kept   the code a reader sees
//
// The lenses form a lattice, not a ladder, and its edges are a claim the
// tests pin: exact under verbatim implies exact under vocab (canonicalizing
// two identical trees gives identical trees), exact under vocab or ordered
// implies exact under shape (each keeps strictly more than shape), and exact
// under shape implies exact under skeleton. vocab and ordered are
// incomparable — each keeps something the other forgets. What a pair is
// follows from where its agreement stops (see LensProfile.Class).
//
// Like the concept views, the lens scores are reported side by side and never
// blended: production retrieval and ranking read the shape lens alone, and a
// lens only enters a score once a measurement against the labels says so.
//
// # Each lens needs its own corpus weights
//
// A lens's labels are a different vocabulary, so its IDF must be counted over
// its own bags (LabelWeights over the lens's bags for the whole population).
// Labels are deliberately *not* seeded per lens: the ordered lens changes only
// the labels at and above a directed node, so most of its labels coincide with
// the shape lens's, which is correct — they describe the same subtrees — and
// keeps a shared label dictionary small. Nothing pools two lenses' bags.
type Lens struct {
	Name string
	Opt  WLOptions
	// Written reads the body as written instead of the canonical shape. For
	// a frontend with no canonicalizer the two trees are the same.
	Written bool
}

// Lens indices into Lenses(), for a caller that reads one by name.
const (
	LensSkeleton = iota
	LensShape
	LensOrdered
	LensVocab
	LensVerbatim
	NumLenses
)

// Lenses returns the lens ladder, most forgetful first. A fresh slice each
// call, so no caller can reconfigure another's lens.
func Lenses() []Lens {
	return []Lens{
		LensSkeleton: {Name: "skeleton", Opt: WLOptions{DropCallees: true}},
		LensShape:    {Name: "shape"},
		LensOrdered:  {Name: "ordered", Opt: WLOptions{Roles: WLRolesAssign}},
		LensVocab:    {Name: "vocab", Opt: WLOptions{KeepNames: true}},
		LensVerbatim: {Name: "verbatim", Opt: WLOptions{KeepNames: true}, Written: true},
	}
}

// Bag is the lens's reading of fn. nil for a function without a body.
func (l Lens) Bag(fn *syntax.Func) []LabelCount {
	if fn == nil || fn.Body == nil {
		return nil
	}
	root := fn.Shape()
	if l.Written {
		root = fn.Body
	}
	return WLBagWith(root, l.Opt)
}

// LensScore is one lens's verdict on a pair: the corpus-weighted Jaccard and
// containment of the two bags, and whether the bags are identical. Exact is
// decided on the bags themselves, never on Jaccard == 1, because a pair whose
// every label is corpus-universal reads Jaccard 0 while being identical.
type LensScore struct {
	Jaccard     float64
	Containment float64
	Exact       bool
}

// ScoreLens compares two bags under one lens's weights.
func ScoreLens(a, b []LabelCount, idf *LabelIDF) LensScore {
	j, c := wlOverlap(a, b, idf)
	return LensScore{Jaccard: j, Containment: c, Exact: len(a) > 0 && equalBags(a, b)}
}

// equalBags reports whether two sorted bags carry the same labels at the same
// counts.
func equalBags(a, b []LabelCount) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Label != b[i].Label || a[i].Count != b[i].Count {
			return false
		}
	}
	return true
}

// LensProfile is a pair read under every lens, in Lenses() order.
type LensProfile [NumLenses]LensScore

// Class names what an exact agreement pattern says a pair is. It is the
// clone-type ladder of the clone-detection literature read off the lenses,
// plus the one class that ladder has no rung for:
//
//	verbatim       identical as written                       (Type-1)
//	renamed        identical once canonicalization has renamed
//	               the locals and normalized spellings         (Type-2)
//	inverse        identical in shape, but the assignment sides
//	               are the other way round: gather and scatter
//	parameterized  identical in shape over different free names
//	               or constant values                          (Type-2, blind)
//	template       identical skeleton over different callees
//
// "" when no lens reads the pair as exact. Only exact agreement is classified:
// a graded pattern ("skeleton 0.95, shape 0.40") is a measurement to report,
// not a rule to name, until the labels say where its cut belongs.
func (p LensProfile) Class() string {
	switch {
	case p[LensVerbatim].Exact:
		return "verbatim"
	case p[LensVocab].Exact:
		// Identical on the canonical tree with names kept: alpha-renaming
		// made the bound names agree, so what differed as written was
		// binding names or a canonicalized spelling.
		return "renamed"
	case p[LensShape].Exact && !p[LensOrdered].Exact:
		return "inverse"
	case p[LensShape].Exact:
		return "parameterized"
	case p[LensSkeleton].Exact:
		return "template"
	}
	return ""
}

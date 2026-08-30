package fingerprint

import (
	"go/ast"
	"math"
	"slices"
)

// wlRounds is how many Weisfeiler–Lehman refinement rounds run past the raw
// node kind, so labels exist for h = 0..wlRounds and a node's deepest label
// summarises everything within wlRounds edges below it.
//
// Three is where a label stops describing a node and starts describing a
// region: at h=3 an if-statement's label already folds in its condition, that
// condition's operands, and their operands in turn — a whole guard. Going
// deeper mostly manufactures labels that occur once in the corpus, which
// carry maximal ln(N/df) and pair with nothing.
const wlRounds = 3

// LabelCount is one Weisfeiler–Lehman label and how many times a body carries
// it. A bag is a slice of these sorted ascending by Label, with no label
// repeated.
//
// A sorted slice rather than the map T2 shipped, now that something scores on
// it. Every other multiset on a Fingerprint — Shingles, Patterns — is already
// a sorted slice, for the reason this one now is too: scoring a pair is a
// single merge of two sorted runs, and sorting inside the pair loop would
// dominate a stage that runs on tens of thousands of pairs. Keeping the map
// as well and deriving this from it would be two spellings of one multiset,
// which is the drift this package refuses everywhere else.
type LabelCount struct {
	Label uint64
	Count int
}

// wlLabels holds one node's labels for every round, index h.
type wlLabels [wlRounds + 1]uint64

// wlFrame is one open node during the walk: its own label_0, and where its
// children's label vectors start in the shared arena.
type wlFrame struct {
	label0 uint64
	start  int
}

// WLBag computes the Weisfeiler–Lehman label multiset of a function body.
//
// The recurrence is the standard one — label_0(v) is the node's kind, and
//
//	label_h(v) = hash(label_{h-1}(v), sorted multiset of label_{h-1}(children))
//
// — evaluated for h = 1..wlRounds. Every label at every round goes into one
// merged multiset keyed by hash: a coarse label and a refined one are both
// just evidence about the body, and the round that produced a label is
// already folded into its hash, so nothing needs to separate them. Two
// labels from different rounds can only collide the way any two FNV values
// can, which is the collision budget the rest of this package already runs on.
//
// # It is meant for the canonical tree
//
// The caller passes canon's canonical form, not the parsed declaration: the
// bag is a shape key, and canonicalization is what makes two functions that
// differ only in incidental choices produce the same shape. Nothing here
// requires it — a raw declaration yields a perfectly well-formed bag — but
// the labels then carry whichever accidents canon exists to remove.
//
// Only the body is walked, exactly as walk() does for the token stream. The
// signature is not shape: it has its own Fingerprint.Types component, and
// folding it in here would count it twice. A nil declaration or one without
// a body yields a nil bag, mirroring the zero Fingerprint's "no body".
//
// # Cost
//
// One post-order pass. Every round is computed at the node on the way back
// up, when its children's whole label vectors are already known, so the tree
// is walked once rather than once per round. Children's vectors live in a
// single arena reused across siblings, so a function costs one map and two
// slices no matter how deep it nests.
func WLBag(fd *ast.FuncDecl) []LabelCount {
	if fd == nil || fd.Body == nil {
		return nil
	}
	bag := make(map[uint64]int)
	var frames []wlFrame
	var kids []wlLabels
	var buf []uint64

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if n != nil {
			frames = append(frames, wlFrame{label0: wlLabel0(n), start: len(kids)})
			return true
		}
		// ast.Inspect calls f(nil) after a node's children, and only for
		// nodes whose f returned true. Returning true unconditionally above
		// is what pairs every push with exactly one pop; the guard is for
		// the day that stops being true, because a panic here would take
		// down a parse over a bag nothing reads yet.
		last := len(frames) - 1
		if last < 0 {
			return false
		}
		fr := frames[last]
		frames = frames[:last]

		var lab wlLabels
		lab[0] = fr.label0
		children := kids[fr.start:]
		for h := 1; h <= wlRounds; h++ {
			buf = buf[:0]
			for i := range children {
				buf = append(buf, children[i][h-1])
			}
			// Sorted, so sibling order never reaches a label: the
			// recurrence is over a multiset of children, not a sequence.
			slices.Sort(buf)
			lab[h] = wlHash(h, lab[h-1], buf)
		}
		// The children have been read; their arena slots are now free and
		// the node takes the first of them as its own slot in its parent's
		// child run.
		kids = append(kids[:fr.start], lab)

		for h := 0; h <= wlRounds; h++ {
			bag[lab[h]]++
		}
		return false
	})
	// The map is a counting scratchpad and never escapes: the result is
	// sorted, so nothing downstream can depend on Go's map order.
	out := make([]LabelCount, 0, len(bag))
	for label, n := range bag {
		out = append(out, LabelCount{Label: label, Count: n})
	}
	slices.SortFunc(out, func(a, b LabelCount) int {
		switch {
		case a.Label < b.Label:
			return -1
		case a.Label > b.Label:
			return 1
		}
		return 0
	})
	return out
}

// FNV-1a, inlined over uint64 accumulators rather than through hash/fnv.
// This is the same hash the rest of the package computes — same offset
// basis, same prime, same byte order — but a hasher allocated per node per
// round is four allocations a node, which on a large corpus is the whole
// cost of the bag.
const (
	fnvOffset64 = 14695981039346656037
	fnvPrime64  = 1099511628211
)

func fnvByte(h uint64, b byte) uint64 { return (h ^ uint64(b)) * fnvPrime64 }

func fnvU64(h, v uint64) uint64 {
	for shift := 56; shift >= 0; shift -= 8 {
		h = fnvByte(h, byte(v>>uint(shift)))
	}
	return h
}

func fnvString(h uint64, s string) uint64 {
	for i := 0; i < len(s); i++ {
		h = fnvByte(h, s[i])
	}
	return h
}

// wlHash serialises one refinement step. Every field is fixed-width and the
// child count is written before the children, so no (self, children) pair
// can serialise to the same bytes as a different one — the ambiguity plain
// concatenation would allow. The round leads, so the same node with the same
// children cannot produce one label at two rounds.
func wlHash(round int, self uint64, children []uint64) uint64 {
	h := fnvByte(fnvOffset64, byte(round))
	h = fnvU64(h, self)
	h = fnvU64(h, uint64(len(children)))
	for _, c := range children {
		h = fnvU64(h, c)
	}
	return h
}

// wlKind hashes a label_0: a kind and the one name that is part of it. The
// two parts are hashed with a separator rather than concatenated into a
// string, because "BIN:" + op allocates once per node and this is the hot
// path. Round byte 0 leads, matching wlHash.
func wlKind(kind, name string) uint64 {
	h := fnvByte(fnvOffset64, 0)
	h = fnvString(h, kind)
	h = fnvByte(h, 0)
	return fnvString(h, name)
}

// wlLabel0 is the initial label: the node's kind.
//
// Two rules decide what "kind" means, and both are inherited rather than
// invented:
//
//   - Where go/ast folds several syntactic constructs into one struct
//     behind a token field, that token is part of the kind. walk() already
//     splits ASSIGN, BRANCH, BIN, UNARY and LIT this way; := and = are not
//     the same statement and const and var are not the same declaration.
//   - A call keeps its selector name — CALL/Errorf, CALL/Lock — with the
//     receiver expression dropped, exactly as walk() does. The name a
//     function calls is intent; the variable it calls it on is arbitrary.
//
// # Identifiers collapse to ID, deliberately
//
// Every *ast.Ident labels as ID, with no name, which is walk()'s rule for
// the token stream and is what makes a renamed copy produce an identical
// bag. It matters twice over on a canonical tree, because canon has already
// split identifiers into two populations and neither should be labelled by
// name:
//
//   - Bound identifiers are positional after alpha-renaming — x0, x1, … in
//     binding order. Labelling by name would put binding *order* into every
//     label above them, so two functions doing the same work with their
//     first two locals declared the other way round would share nothing.
//   - Free identifiers keep their source names, which is the shared
//     vocabulary two functions are compared on — but that vocabulary is
//     already carried where it is intent-bearing, on the CALL labels. An
//     identifier in any other position is a type, a package qualifier or a
//     field, and admitting those names here would make the bag a lexical
//     index rather than a structural one, which is the thing this tool
//     refuses to be everywhere else.
//
// The name of the declaration itself never arises: only the body is walked.
func wlLabel0(n ast.Node) uint64 {
	switch node := n.(type) {
	// Kinds the token stream already names, spelled the same way.
	case *ast.Ident:
		return wlKind("ID", "")
	case *ast.CallExpr:
		return wlKind("CALL", calleeName(node))
	case *ast.BinaryExpr:
		return wlKind("BIN", node.Op.String())
	case *ast.UnaryExpr:
		return wlKind("UNARY", node.Op.String())
	case *ast.AssignStmt:
		return wlKind("ASSIGN", node.Tok.String())
	case *ast.BranchStmt:
		return wlKind("BRANCH", node.Tok.String())
	case *ast.BasicLit:
		return wlKind("LIT", node.Kind.String())
	case *ast.IfStmt:
		return wlKind("IF", "")
	case *ast.ForStmt:
		return wlKind("FOR", "")
	case *ast.RangeStmt:
		return wlKind("RANGE", "")
	case *ast.SwitchStmt:
		return wlKind("SWITCH", "")
	case *ast.TypeSwitchStmt:
		return wlKind("TYPESWITCH", "")
	case *ast.SelectStmt:
		return wlKind("SELECT", "")
	case *ast.ReturnStmt:
		return wlKind("RETURN", "")
	case *ast.DeferStmt:
		return wlKind("DEFER", "")
	case *ast.GoStmt:
		return wlKind("GO", "")
	case *ast.FuncLit:
		return wlKind("FUNCLIT", "")
	case *ast.IncDecStmt:
		return wlKind("INCDEC", node.Tok.String())
	case *ast.IndexExpr:
		return wlKind("INDEX", "")
	case *ast.SliceExpr:
		return wlKind("SLICE", "")
	case *ast.StarExpr:
		return wlKind("STAR", "")
	case *ast.TypeAssertExpr:
		return wlKind("ASSERT", "")
	case *ast.CompositeLit:
		return wlKind("COMPOSITE", "")
	case *ast.KeyValueExpr:
		return wlKind("KV", "")
	case *ast.SelectorExpr:
		return wlKind("SEL", "")

	// Kinds the token stream drops. WL needs a label for every node, since
	// a node with no label would silently vanish from its parent's child
	// multiset and make two different shapes agree.
	case *ast.BlockStmt:
		return wlKind("BLOCK", "")
	case *ast.ExprStmt:
		return wlKind("EXPRSTMT", "")
	case *ast.EmptyStmt:
		return wlKind("EMPTY", "")
	case *ast.LabeledStmt:
		return wlKind("LABELED", "")
	case *ast.SendStmt:
		return wlKind("SEND", "")
	case *ast.DeclStmt:
		return wlKind("DECLSTMT", "")
	case *ast.CaseClause:
		return wlKind("CASE", "")
	case *ast.CommClause:
		return wlKind("COMM", "")
	case *ast.ParenExpr:
		return wlKind("PAREN", "")
	case *ast.Ellipsis:
		return wlKind("ELLIPSIS", "")
	case *ast.IndexListExpr:
		return wlKind("INDEXLIST", "")
	case *ast.ArrayType:
		return wlKind("ARRAYTYPE", "")
	case *ast.StructType:
		return wlKind("STRUCTTYPE", "")
	case *ast.FuncType:
		return wlKind("FUNCTYPE", "")
	case *ast.InterfaceType:
		return wlKind("INTERFACETYPE", "")
	case *ast.MapType:
		return wlKind("MAPTYPE", "")
	case *ast.ChanType:
		return wlKind("CHANTYPE", chanDir(node.Dir))
	case *ast.Field:
		return wlKind("FIELD", "")
	case *ast.FieldList:
		return wlKind("FIELDLIST", "")
	case *ast.GenDecl:
		return wlKind("GENDECL", node.Tok.String())
	case *ast.ValueSpec:
		return wlKind("VALUESPEC", "")
	case *ast.TypeSpec:
		return wlKind("TYPESPEC", "")
	case *ast.ImportSpec:
		return wlKind("IMPORTSPEC", "")
	case *ast.Comment:
		return wlKind("COMMENT", "")
	case *ast.CommentGroup:
		return wlKind("COMMENTGROUP", "")
	case *ast.FuncDecl:
		return wlKind("FUNCDECL", "")
	case *ast.File:
		return wlKind("FILE", "")
	case *ast.BadExpr:
		return wlKind("BADEXPR", "")
	case *ast.BadStmt:
		return wlKind("BADSTMT", "")
	case *ast.BadDecl:
		return wlKind("BADDECL", "")
	}
	// go/ast's node set is closed today; a kind added to it later lands
	// here rather than disappearing from its parent's child multiset.
	return wlKind("NODE", "")
}

// chanDir names a channel direction for the label. ast.ChanDir is a bit set,
// so the bidirectional case is both bits.
func chanDir(d ast.ChanDir) string {
	switch d {
	case ast.SEND:
		return "send"
	case ast.RECV:
		return "recv"
	}
	return "both"
}

// LabelIDF is the corpus surprisal of every Weisfeiler–Lehman label: how many
// functions carry it, and what sharing it is therefore worth.
//
// The weight is ln(N/df), the same information measure the retrieval channels
// use and in the same unit — nats — so a WL mass can be summed with a shape,
// concept or call mass without normalising first. df is *presence* df: a
// label repeated forty times inside one function still counts once, because
// the question is how many functions could have shared it. A label every
// function carries weighs exactly 0.
//
// N is the number of functions that have a body. A declaration without one
// carries no labels and cannot fail to carry any either; counting it would
// add a constant to every weight that says nothing about the corpus.
type LabelIDF struct {
	n  int
	df map[uint64]int
	w  map[uint64]float64 // ln(N/df), precomputed
}

// LabelWeights counts label document frequency across a population's bags.
// Pass one bag per function, in any order and including nil ones; the result
// depends only on the multiset of bags, never on their order.
//
// The ln(N/df) of every label is computed once, here. Weight is called twice
// per label per scored pair and the pair count is quadratic-ish in the corpus;
// a math.Log per call is more expensive than the lookup that finds it.
func LabelWeights(bags [][]LabelCount) *LabelIDF {
	w := &LabelIDF{df: make(map[uint64]int)}
	for _, bag := range bags {
		if len(bag) == 0 {
			continue
		}
		w.n++
		for _, lc := range bag {
			w.df[lc.Label]++
		}
	}
	w.w = make(map[uint64]float64, len(w.df))
	// Iterating a map is safe here and only here: every entry is computed
	// independently from its own key and value, so no ordering decides
	// anything.
	for label, df := range w.df {
		w.w[label] = math.Log(float64(w.n) / float64(df))
	}
	return w
}

// N is the number of functions the weights were counted over.
func (w *LabelIDF) N() int {
	if w == nil {
		return 0
	}
	return w.n
}

// DF is how many of those functions carry the label; 0 for one the corpus
// has never seen.
func (w *LabelIDF) DF(label uint64) int {
	if w == nil {
		return 0
	}
	return w.df[label]
}

// Weight is ln(N/df), the evidence in nats that two functions sharing this
// label supply. A label every function carries weighs exactly 0.
//
// # An unseen label counts as df 1, the rarest thing the corpus can express
//
// T2 returned 0 here, on the reading that a label the corpus has never seen is
// absence of evidence. Scoring makes that reading unsafe, so it is now ln(N).
//
// A weight of 0 does not make a label neutral in a weighted Jaccard — it makes
// it *invisible*, dropping out of the numerator and the denominator alike. Two
// functions built entirely of labels this table has never seen would then
// divide 0 by 0 and be reported as unlike, or worse, share one incidental
// label and be reported as identical. Under ln(N) an unseen label lands in the
// union and depresses the score, so the failure direction is "less similar
// than they are", which a reader can act on. Both readings are defensible for
// a lookup; only one is safe for a ratio.
//
// This is a fallback, not a path. Every unit the pipeline scores is in the
// population LabelWeights counted, query probes included — they join the
// corpus before the weights are built. A label reaches this branch only when a
// caller scores a fingerprint from outside the corpus it passed.
func (w *LabelIDF) Weight(label uint64) float64 {
	if w == nil || w.n == 0 {
		return 0
	}
	if weight, ok := w.w[label]; ok {
		return weight
	}
	return math.Log(float64(w.n))
}

// Labels returns every label the corpus carries, ascending. It exists so a
// consumer never has to range over the internal map: this is the only
// ordered view, and it is a total order.
func (w *LabelIDF) Labels() []uint64 {
	if w == nil || len(w.df) == 0 {
		return nil
	}
	out := make([]uint64, 0, len(w.df))
	for label := range w.df {
		out = append(out, label)
	}
	slices.Sort(out)
	return out
}

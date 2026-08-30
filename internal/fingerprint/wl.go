package fingerprint

import (
	"math"
	"slices"
	"strconv"

	"github.com/LukasSelin/doppel/internal/syntax"
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

// LabelCount is one Weisfeiler–Lehman label, how many times a body carries
// it, and the two facts that let it be named to a reader. A bag is a slice of
// these sorted ascending by Label, with no label repeated.
//
// A sorted slice rather than a map, now that something scores on it. Every
// other multiset on a Fingerprint — Shingles — is already a sorted slice, for
// the reason this one is too: scoring a pair is a single merge of two sorted
// runs, and sorting inside the pair loop would dominate a stage that runs on
// tens of thousands of pairs. Keeping a map as well and deriving this from it
// would be two spellings of one multiset, which is the drift this package
// refuses everywhere else.
//
// # H and Kind are free, and that is why they are here
//
// A WL label is an opaque hash: nothing downstream can say what a shared
// label *was*, which is what the retrieval channel's explanation block needs.
// H (the refinement round) and Kind (the node kind the label was computed at)
// are both determined by the label — the round leads every wlHash and the
// kind bottoms out every chain at label_0 — so recording them at construction
// is bookkeeping, not new information.
//
// They cost nothing. Count is int32 rather than int precisely so the struct
// stays at 16 bytes: H and Kind live in padding the old two-field version
// already wasted, and the merge loop in wlOverlap reads the same cache lines
// it did before. A body cannot carry one label two billion times.
type LabelCount struct {
	Label uint64
	Count int32
	H     uint8     // refinement round, 0..wlRounds
	Kind  LabelKind // node kind the label was computed at
}

// LabelKind is the node kind a Weisfeiler–Lehman label was computed at — the
// label_0 vocabulary, which every deeper label inherits through its self
// chain.
//
// It is this package's own enum rather than syntax.Kind directly, for one
// reason: the label is a hash of the kind's *name*, and the name has to be
// pinned here where the scoring lives. syntax.Kind is the IR's vocabulary and
// may grow a kind or reword one without meaning to move every score in the
// tool; labelKindNames is the table that decides what a label is, and
// wlLabel0 is the one place the two vocabularies meet.
type LabelKind uint8

// The label_0 vocabulary. Values are positional and are never serialized to
// anything durable — the hash is computed over the *name*, so reordering this
// block changes no label. Appending is safe; renaming a constant's string in
// labelKindNames changes every label under it, which is a scoring change.
const (
	KindNode LabelKind = iota // the open-world fallback; see wlLabel0
	KindIdent
	KindCall
	KindBinary
	KindUnary
	KindAssign
	KindBranch
	KindLit
	KindIf
	KindFor
	KindRange
	KindSwitch
	KindTypeSwitch
	KindSelect
	KindReturn
	KindDefer
	KindGo
	KindFuncLit
	KindIncDec
	KindIndex
	KindSlice
	KindStar
	KindAssert
	KindComposite
	KindKeyValue
	KindSelector
	KindBlock
	KindExprStmt
	KindEmpty
	KindLabeled
	KindSend
	KindDeclStmt
	KindCase
	KindComm
	KindParen
	KindEllipsis
	KindIndexList
	KindArrayType
	KindStructType
	KindFuncType
	KindInterfaceType
	KindMapType
	KindChanType
	KindField
	KindFieldList
	KindGenDecl
	KindValueSpec
	KindTypeSpec
	KindImportSpec
	KindBadExpr
	KindBadStmt
	KindBadDecl
	numLabelKinds
)

// labelKindNames is the hash input; index-aligned with the constants above.
// TestLabelKindNames pins the length against numLabelKinds so a kind added
// without a name fails the build's tests rather than hashing as "".
var labelKindNames = [numLabelKinds]string{
	"NODE", "ID", "CALL", "BIN", "UNARY", "ASSIGN", "BRANCH", "LIT",
	"IF", "FOR", "RANGE", "SWITCH", "TYPESWITCH", "SELECT", "RETURN",
	"DEFER", "GO", "FUNCLIT", "INCDEC", "INDEX", "SLICE", "STAR",
	"ASSERT", "COMPOSITE", "KV", "SEL", "BLOCK", "EXPRSTMT", "EMPTY",
	"LABELED", "SEND", "DECLSTMT", "CASE", "COMM", "PAREN", "ELLIPSIS",
	"INDEXLIST", "ARRAYTYPE", "STRUCTTYPE", "FUNCTYPE", "INTERFACETYPE",
	"MAPTYPE", "CHANTYPE", "FIELD", "FIELDLIST", "GENDECL", "VALUESPEC",
	"TYPESPEC", "IMPORTSPEC", "BADEXPR", "BADSTMT", "BADDECL",
}

// labelKindWords is the reader-facing word for each kind, index-aligned with
// labelKindNames. It is display-only and never reaches a hash, which is why
// it can read like English where the hash name reads like a token.
//
// It is a table rather than a derivation. The words used to come from
// reflecting on go/ast's type names with a small override map, which was
// exactly right while this package was Go-typed and is not available at all
// now that it walks a neutral tree — and a table is the honest replacement,
// because a word chosen for a reader is not something a naming convention can
// be trusted to produce.
var labelKindWords = [numLabelKinds]string{
	"node", "ident", "call", "binary", "unary", "assign", "branch", "literal",
	"if", "for", "range", "switch", "typeswitch", "select", "return",
	"defer", "go", "function literal", "increment", "index", "slice", "star",
	"type assertion", "composite literal", "key-value", "selector", "block",
	"expression statement", "empty", "labeled", "send", "declaration statement",
	"case", "comm", "paren", "ellipsis", "indexlist", "arraytype", "structtype",
	"functype", "interfacetype", "maptype", "chantype", "field", "fieldlist",
	"declaration", "valuespec", "typespec", "importspec", "bad", "bad", "baddecl",
}

// String is the kind's name — the same bytes wlKind hashes.
func (k LabelKind) String() string {
	if int(k) >= len(labelKindNames) {
		return "NODE"
	}
	return labelKindNames[k]
}

// Word is the kind's reader-facing name, for a report that has to say what
// two bodies differ by.
func (k LabelKind) Word() string {
	if int(k) >= len(labelKindWords) {
		return "node"
	}
	return labelKindWords[k]
}

// DescribeLabel names a label for a reader: the refinement round it was
// produced at and the node kind it was computed at — "depth-2 IF".
//
// # This is a weaker explanation than a rendered pattern would be
//
// A WL label has no faithful short string: it is a hash of a whole subtree,
// and the only honest short name for it is where it sits and what it sits on.
// "depth-2 IF" says a guard three levels deep matched exactly, which is a real
// and checkable claim about a pair, but it does not say *which* guard. Naming
// the subtree would mean rendering it, which is a second serialization of the
// thing the hash already is — a drift this package refuses.
//
// One definition, here, so the retriever's block and any later explanation
// layer print a label the same way.
func DescribeLabel(h uint8, k LabelKind) string {
	return "depth-" + strconv.Itoa(int(h)) + " " + k.String()
}

// wlLabels holds one node's labels for every round, index h.
type wlLabels [wlRounds + 1]uint64

// wlFrame is one open node during the walk: its own label_0 and kind, and
// where its children's label vectors start in the shared arena.
type wlFrame struct {
	label0 uint64
	kind   LabelKind
	start  int
}

// labelAcc is one label's accumulator during the walk: how many times it has
// been seen, and the (round, kind) of the first node that produced it.
//
// First sighting wins, and the walk is post-order and deterministic, so no
// map iteration decides which meta a label carries. The tie only arises under
// a hash collision between two labels of different rounds or kinds, which is
// the same collision budget the rest of this package runs on: the count is
// then shared too, and the meta names one of the two colliding shapes rather
// than a third thing.
type labelAcc struct {
	count int32
	h     uint8
	kind  LabelKind
}

// WLBag computes the Weisfeiler–Lehman label multiset of a function's shape.
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
// # It walks the canonical shape when there is one
//
// The tree is syntax.Func.Shape: the frontend's canonical body where the
// frontend has a canonicalizer, and the body as written where it does not.
// The bag is a shape key, and canonicalization is what makes two functions
// that differ only in incidental choices produce the same shape. Nothing here
// requires it — a raw body yields a perfectly well-formed bag — but the labels
// then carry whichever accidents a canonicalizer exists to remove. That is
// what a language without one loses: recall on those pairs, never correctness.
//
// Only the body is walked, exactly as walk() does for the token stream. The
// signature is not shape: it has its own Fingerprint.Types component, and
// folding it in here would count it twice. A nil function or one without a
// body yields a nil bag, mirroring the zero Fingerprint's "no body".
//
// # Cost
//
// One post-order pass. Every round is computed at the node on the way back
// up, when its children's whole label vectors are already known, so the tree
// is walked once rather than once per round. Children's vectors live in a
// single arena reused across siblings, so a function costs one map and two
// slices no matter how deep it nests.
func WLBag(fn *syntax.Func) []LabelCount {
	return wlBagOf(fn.Shape())
}

// wlBagOf is WLBag over an already-chosen tree, so a caller holding only the
// canonical body (the snapshot round-trip, the corpus statistics) does not
// have to fabricate a syntax.Func to reach it.
func wlBagOf(root *syntax.Node) []LabelCount {
	if root == nil {
		return nil
	}
	bag := make(map[uint64]*labelAcc)
	var frames []wlFrame
	var kids []wlLabels
	var buf []uint64

	syntax.Inspect(root, func(n *syntax.Node) bool {
		if n != nil {
			kind, name := wlLabel0(n)
			frames = append(frames, wlFrame{
				label0: wlKind(kind.String(), name),
				kind:   kind,
				start:  len(kids),
			})
			return true
		}
		// syntax.Inspect calls f(nil) after a node's children, and only for
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
			if acc, ok := bag[lab[h]]; ok {
				acc.count++
				continue
			}
			bag[lab[h]] = &labelAcc{count: 1, h: uint8(h), kind: fr.kind}
		}
		return false
	})
	// The map is a counting scratchpad and never escapes: the result is
	// sorted, so nothing downstream can depend on Go's map order.
	out := make([]LabelCount, 0, len(bag))
	for label, acc := range bag {
		out = append(out, LabelCount{Label: label, Count: acc.count, H: acc.h, Kind: acc.kind})
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

// wlLabel0Hash is wlLabel0 for a caller that wants only the label — Cons,
// which shares the label_0 vocabulary but keeps no per-label bookkeeping.
func wlLabel0Hash(n *syntax.Node) uint64 {
	kind, name := wlLabel0(n)
	return wlKind(kind.String(), name)
}

// wlLabel0 is the initial label: the node's kind, plus the one name that is
// part of that kind.
//
// It is the one place syntax.Kind and LabelKind meet, and the mapping is
// total and one-to-one by design: an IR kind that collapsed several
// constructs into one label would not lose them from the bag, it would merge
// them into their parents' child multisets and make two different shapes
// agree. A kind the IR does not name arrives as syntax.KindOther and labels
// as NODE — still a node, still counted, just not distinguished.
//
// Two rules decide what "part of the kind" means, and both are inherited from
// the token stream rather than invented here:
//
//   - Where a language folds several constructs into one node behind a token,
//     that token is part of the kind. The IR records it in Label, and ASSIGN,
//     BRANCH, BIN, UNARY, LIT, INCDEC, GENDECL and CHANTYPE all read it: :=
//     and = are not the same statement and const and var are not the same
//     declaration.
//   - A call keeps its callee name — CALL/Errorf, CALL/Lock — with the
//     receiver expression dropped, exactly as walk() does. The name a
//     function calls is intent; the variable it calls it on is arbitrary.
//
// # Identifiers collapse to ID, deliberately
//
// Every identifier labels as ID with no name, which is walk()'s rule for the
// token stream and is what makes a renamed copy produce an identical bag. It
// matters twice over on a canonical tree, because canonicalization has
// already split identifiers into two populations and neither should be
// labelled by name:
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
func wlLabel0(n *syntax.Node) (LabelKind, string) {
	switch n.Kind {
	// Kinds carrying the one token that is part of them.
	case syntax.KindCall:
		return KindCall, calleeName(n)
	case syntax.KindBinary:
		return KindBinary, n.Label
	case syntax.KindUnary:
		return KindUnary, n.Label
	case syntax.KindAssign:
		return KindAssign, n.Label
	case syntax.KindBranch:
		return KindBranch, n.Label
	case syntax.KindLit:
		return KindLit, n.Label
	case syntax.KindIncDec:
		return KindIncDec, n.Label
	case syntax.KindGenDecl:
		return KindGenDecl, n.Label
	case syntax.KindChanType:
		return KindChanType, n.Label

	// Kinds that are their kind and nothing else. An identifier is here
	// rather than above precisely because its Label is dropped.
	case syntax.KindIdent:
		return KindIdent, ""
	case syntax.KindIf:
		return KindIf, ""
	case syntax.KindFor:
		return KindFor, ""
	case syntax.KindRange:
		return KindRange, ""
	case syntax.KindSwitch:
		return KindSwitch, ""
	case syntax.KindTypeSwitch:
		return KindTypeSwitch, ""
	case syntax.KindSelect:
		return KindSelect, ""
	case syntax.KindReturn:
		return KindReturn, ""
	case syntax.KindDefer:
		return KindDefer, ""
	case syntax.KindGo:
		return KindGo, ""
	case syntax.KindFuncLit:
		return KindFuncLit, ""
	case syntax.KindIndex:
		return KindIndex, ""
	case syntax.KindSlice:
		return KindSlice, ""
	case syntax.KindStar:
		return KindStar, ""
	case syntax.KindAssert:
		return KindAssert, ""
	case syntax.KindComposite:
		return KindComposite, ""
	case syntax.KindKeyValue:
		return KindKeyValue, ""
	case syntax.KindSelector:
		return KindSelector, ""
	case syntax.KindBlock:
		return KindBlock, ""
	case syntax.KindExprStmt:
		return KindExprStmt, ""
	case syntax.KindEmpty:
		return KindEmpty, ""
	case syntax.KindLabeled:
		return KindLabeled, ""
	case syntax.KindSend:
		return KindSend, ""
	case syntax.KindDeclStmt:
		return KindDeclStmt, ""
	case syntax.KindCaseClause:
		return KindCase, ""
	case syntax.KindCommClause:
		return KindComm, ""
	case syntax.KindParen:
		return KindParen, ""
	case syntax.KindEllipsis:
		return KindEllipsis, ""
	case syntax.KindIndexList:
		return KindIndexList, ""
	case syntax.KindArrayType:
		return KindArrayType, ""
	case syntax.KindStructType:
		return KindStructType, ""
	case syntax.KindFuncType:
		return KindFuncType, ""
	case syntax.KindInterfaceType:
		return KindInterfaceType, ""
	case syntax.KindMapType:
		return KindMapType, ""
	case syntax.KindField:
		return KindField, ""
	case syntax.KindFieldList:
		return KindFieldList, ""
	case syntax.KindValueSpec:
		return KindValueSpec, ""
	case syntax.KindTypeSpec:
		return KindTypeSpec, ""
	case syntax.KindImportSpec:
		return KindImportSpec, ""
	case syntax.KindBadExpr:
		return KindBadExpr, ""
	case syntax.KindBadStmt:
		return KindBadStmt, ""
	case syntax.KindBadDecl:
		return KindBadDecl, ""
	}
	// syntax.KindOther, and any kind the IR grows before this switch does.
	return KindNode, ""
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

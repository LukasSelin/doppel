package fingerprint

import (
	"go/ast"
	"math"
	"slices"
	"strconv"
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
// A sorted slice rather than the map T2 shipped, now that something scores on
// it. Every other multiset on a Fingerprint — Shingles — is already a sorted
// slice, for the reason this one is too: scoring a pair is a single merge of
// two sorted runs, and sorting inside the pair loop would dominate a stage
// that runs on tens of thousands of pairs. Keeping the map as well and
// deriving this from it would be two spellings of one multiset, which is the
// drift this package refuses everywhere else.
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
// It is an enum rather than the string wlLabel0 used to build inline so that
// a bag carries it for free (see LabelCount), and it is *the* vocabulary
// rather than a second one: wlLabel0 returns a LabelKind and the hash is
// computed over its String(), so a kind cannot exist in the switch without a
// name or gain a name that hashes differently than it reads.
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
	KindComment
	KindCommentGroup
	KindFuncDecl
	KindFile
	KindBadExpr
	KindBadStmt
	KindBadDecl
	numLabelKinds
)

// labelKindNames is the hash input and the display name at once — the whole
// point of routing both through one table. Index-aligned with the constants
// above; TestLabelKindNames pins the length against numLabelKinds so a kind
// added without a name fails the build's tests rather than hashing as "".
var labelKindNames = [numLabelKinds]string{
	"NODE", "ID", "CALL", "BIN", "UNARY", "ASSIGN", "BRANCH", "LIT",
	"IF", "FOR", "RANGE", "SWITCH", "TYPESWITCH", "SELECT", "RETURN",
	"DEFER", "GO", "FUNCLIT", "INCDEC", "INDEX", "SLICE", "STAR",
	"ASSERT", "COMPOSITE", "KV", "SEL", "BLOCK", "EXPRSTMT", "EMPTY",
	"LABELED", "SEND", "DECLSTMT", "CASE", "COMM", "PAREN", "ELLIPSIS",
	"INDEXLIST", "ARRAYTYPE", "STRUCTTYPE", "FUNCTYPE", "INTERFACETYPE",
	"MAPTYPE", "CHANTYPE", "FIELD", "FIELDLIST", "GENDECL", "VALUESPEC",
	"TYPESPEC", "IMPORTSPEC", "COMMENT", "COMMENTGROUP", "FUNCDECL",
	"FILE", "BADEXPR", "BADSTMT", "BADDECL",
}

// String is the kind's name — the same bytes wlKind hashes.
func (k LabelKind) String() string {
	if int(k) >= len(labelKindNames) {
		return "NODE"
	}
	return labelKindNames[k]
}

// DescribeLabel names a label for a reader: the refinement round it was
// produced at and the node kind it was computed at — "depth-2 IF".
//
// # This is a weaker explanation than what it replaced, and deliberately so
//
// The pattern multiset this channel used to index carried a *render* — the
// hash's own serialization, so "if(bin:!=(id,nil))" could not drift from the
// thing that was counted. A WL label has no such string: it is a hash of a
// whole subtree, and the only honest short name for it is where it sits and
// what it sits on. "depth-2 IF" says a guard three levels deep matched
// exactly, which is a real and checkable claim about a pair, but it does not
// say *which* guard. Naming the subtree would mean rendering it, which is a
// second serialization of the thing the hash already is — the drift this
// package spent the pattern levels avoiding.
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
	bag := make(map[uint64]*labelAcc)
	var frames []wlFrame
	var kids []wlLabels
	var buf []uint64

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if n != nil {
			kind, name := wlLabel0(n)
			frames = append(frames, wlFrame{
				label0: wlKind(kind.String(), name),
				kind:   kind,
				start:  len(kids),
			})
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
func wlLabel0Hash(n ast.Node) uint64 {
	kind, name := wlLabel0(n)
	return wlKind(kind.String(), name)
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
//
// It returns the kind and the one name that is part of it, rather than the
// finished hash, so the kind reaches the bag as well as the hash. wlKind is
// still what turns the pair into a label, over kind.String() — the same bytes
// the string switch used to pass, so no label moved when this became an enum.
func wlLabel0(n ast.Node) (LabelKind, string) {
	switch node := n.(type) {
	// Kinds the token stream already names, spelled the same way.
	case *ast.Ident:
		return KindIdent, ""
	case *ast.CallExpr:
		return KindCall, calleeName(node)
	case *ast.BinaryExpr:
		return KindBinary, node.Op.String()
	case *ast.UnaryExpr:
		return KindUnary, node.Op.String()
	case *ast.AssignStmt:
		return KindAssign, node.Tok.String()
	case *ast.BranchStmt:
		return KindBranch, node.Tok.String()
	case *ast.BasicLit:
		return KindLit, node.Kind.String()
	case *ast.IfStmt:
		return KindIf, ""
	case *ast.ForStmt:
		return KindFor, ""
	case *ast.RangeStmt:
		return KindRange, ""
	case *ast.SwitchStmt:
		return KindSwitch, ""
	case *ast.TypeSwitchStmt:
		return KindTypeSwitch, ""
	case *ast.SelectStmt:
		return KindSelect, ""
	case *ast.ReturnStmt:
		return KindReturn, ""
	case *ast.DeferStmt:
		return KindDefer, ""
	case *ast.GoStmt:
		return KindGo, ""
	case *ast.FuncLit:
		return KindFuncLit, ""
	case *ast.IncDecStmt:
		return KindIncDec, node.Tok.String()
	case *ast.IndexExpr:
		return KindIndex, ""
	case *ast.SliceExpr:
		return KindSlice, ""
	case *ast.StarExpr:
		return KindStar, ""
	case *ast.TypeAssertExpr:
		return KindAssert, ""
	case *ast.CompositeLit:
		return KindComposite, ""
	case *ast.KeyValueExpr:
		return KindKeyValue, ""
	case *ast.SelectorExpr:
		return KindSelector, ""

	// Kinds the token stream drops. WL needs a label for every node, since
	// a node with no label would silently vanish from its parent's child
	// multiset and make two different shapes agree.
	case *ast.BlockStmt:
		return KindBlock, ""
	case *ast.ExprStmt:
		return KindExprStmt, ""
	case *ast.EmptyStmt:
		return KindEmpty, ""
	case *ast.LabeledStmt:
		return KindLabeled, ""
	case *ast.SendStmt:
		return KindSend, ""
	case *ast.DeclStmt:
		return KindDeclStmt, ""
	case *ast.CaseClause:
		return KindCase, ""
	case *ast.CommClause:
		return KindComm, ""
	case *ast.ParenExpr:
		return KindParen, ""
	case *ast.Ellipsis:
		return KindEllipsis, ""
	case *ast.IndexListExpr:
		return KindIndexList, ""
	case *ast.ArrayType:
		return KindArrayType, ""
	case *ast.StructType:
		return KindStructType, ""
	case *ast.FuncType:
		return KindFuncType, ""
	case *ast.InterfaceType:
		return KindInterfaceType, ""
	case *ast.MapType:
		return KindMapType, ""
	case *ast.ChanType:
		return KindChanType, chanDir(node.Dir)
	case *ast.Field:
		return KindField, ""
	case *ast.FieldList:
		return KindFieldList, ""
	case *ast.GenDecl:
		return KindGenDecl, node.Tok.String()
	case *ast.ValueSpec:
		return KindValueSpec, ""
	case *ast.TypeSpec:
		return KindTypeSpec, ""
	case *ast.ImportSpec:
		return KindImportSpec, ""
	case *ast.Comment:
		return KindComment, ""
	case *ast.CommentGroup:
		return KindCommentGroup, ""
	case *ast.FuncDecl:
		return KindFuncDecl, ""
	case *ast.File:
		return KindFile, ""
	case *ast.BadExpr:
		return KindBadExpr, ""
	case *ast.BadStmt:
		return KindBadStmt, ""
	case *ast.BadDecl:
		return KindBadDecl, ""
	}
	// go/ast's node set is closed today; a kind added to it later lands
	// here rather than disappearing from its parent's child multiset.
	return KindNode, ""
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

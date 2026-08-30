// Package syntax is doppel's language-neutral intermediate representation: a
// generic syntax tree that carries exactly what the fingerprint stages read,
// and nothing about any one language.
//
// It exists because internal/fingerprint used to be typed on *ast.FuncDecl.
// Build, walk, extractPatterns and extractDefUse all took Go AST directly, so
// the similarity score, the retrieval shape channel and all five pattern
// levels were Go-only — a second language frontend could not produce a
// Fingerprint at all, whatever else it filled in.
//
// The contract a frontend must honour is deliberately narrow but it is not
// loose: a Node exists for every node the frontend's own traversal would
// visit, in that traversal's order. Node identity and order are observable —
// Fingerprint.Nodes counts them, the token stream is emitted in visit order,
// and the nesting-depth histogram is driven by the push/pop pairing. A
// frontend that collapses nodes changes scores rather than losing detail
// quietly, so mirroring the source traversal exactly is the requirement, not
// an optimization.
//
// This package imports nothing from this module, the same rule ontology and
// clique follow.
package syntax

// Kind is the node vocabulary. It is a union of what every consumer switches
// on: the ten control-flow kinds the flow histogram counts, the expression
// and statement shapes the token stream names, the type-expression and
// declaration kinds the Weisfeiler-Lehman label bag distinguishes, and
// KindOther for everything a frontend visits but nobody names — those still
// count toward Nodes, which is why they are represented rather than dropped.
//
// The vocabulary is wider than the token stream needs because the label bag
// reads *every* node's kind: a node whose kind collapses to KindOther does
// not vanish from its parent's child multiset, it merges with every other
// unnamed node there. That coarsens the bag rather than shrinking it, so a
// frontend that can tell an array type from a map type should say so.
//
// The values are positional and are never serialized: the label bag hashes
// a kind's *name*, and nothing else stores one. Reordering this block is
// safe; renaming a kind is a scoring change.
type Kind uint8

const (
	KindOther Kind = iota

	// Statements.
	KindIf
	KindFor
	KindRange
	KindSwitch
	KindTypeSwitch
	KindSelect
	KindReturn
	KindDefer
	KindGo
	KindBlock
	KindAssign
	KindBranch
	KindIncDec
	KindSend
	KindExprStmt
	KindDeclStmt
	KindLabeled
	KindEmpty
	KindCaseClause
	KindCommClause
	KindBadStmt

	// Expressions.
	KindCall
	KindBinary
	KindUnary
	KindIdent
	KindLit
	KindSelector
	KindIndex
	KindSlice
	KindStar
	KindAssert
	KindComposite
	KindKeyValue
	KindFuncLit
	KindParen
	KindEllipsis
	KindIndexList

	// Type expressions. A frontend without a type grammar leaves these
	// KindOther; a frontend that has one names them, because the structural
	// label bag reads every node's kind and a collapsed vocabulary is a
	// coarser bag rather than a smaller one.
	KindArrayType
	KindStructType
	KindFuncType
	KindInterfaceType
	KindMapType
	KindChanType

	// Neither, but visited.
	KindValueSpec
	KindTypeSpec
	KindImportSpec
	KindField
	KindFieldList
	KindGenDecl
	KindBadExpr
	KindBadDecl
)

// IsStmt reports whether the kind is a statement. It mirrors Go's ast.Stmt
// interface because extractPatterns' default branch is gated on exactly that
// question: a node that is a statement gets an L2 render attempt, and one
// that is not is skipped. Kinds a frontend cannot classify are KindOther,
// which is not a statement — an unclassified node renders nothing rather
// than rendering something wrong.
func (k Kind) IsStmt() bool {
	switch k {
	case KindIf, KindFor, KindRange, KindSwitch, KindTypeSwitch, KindSelect,
		KindReturn, KindDefer, KindGo, KindBlock, KindAssign, KindBranch,
		KindIncDec, KindSend, KindExprStmt, KindDeclStmt, KindLabeled,
		KindEmpty, KindCaseClause, KindCommClause, KindBadStmt:
		return true
	}
	return false
}

// Role names which slot of its parent a child fills. Position alone cannot
// carry this: a Go for-loop omits absent Init/Cond/Post entirely, so the
// third child of a for is not reliably its Post. Frontends tag the slots they
// can identify and leave the rest RoleNone; consumers that need a slot ask
// for it by name and handle absence.
type Role uint8

const (
	RoleNone Role = iota
	RoleInit
	RoleCond
	RolePost
	RoleBody
	RoleElse
	RoleTag
	RoleX
	RoleY
	RoleSel
	RoleFun
	RoleArg
	RoleLhs
	RoleRhs
	RoleResult
	RoleCall
	RoleValue
	RoleChan
	RoleName
	RoleList
	RoleType
	RoleKey
	RoleElt
	RoleIndex
)

// Child is one parented node together with the slot it fills.
type Child struct {
	Role Role
	Node *Node
}

// Node is one syntax node.
//
// Label carries the one piece of lexical detail a kind needs, and what it
// means is fixed per kind: the operator for KindBinary and KindUnary, the
// assignment, branch, increment or declaration token for KindAssign,
// KindBranch, KindIncDec and KindGenDecl, the direction ("send", "recv",
// "both") for KindChanType, the literal kind
// ("STRING", "INT", …) for KindLit, the name for KindIdent, and the selected
// name for KindSelector. Every other kind leaves it empty.
//
// Text is the literal's decoded value for KindLit — a string literal's
// contents with its quoting and escapes already resolved, because decoding
// them is the frontend's lexical business and not something a consumer can
// redo without knowing the language. Empty for every other kind.
type Node struct {
	Kind  Kind
	Label string
	Text  string
	Kids  []Child
}

// Add appends a child in a slot. It returns the receiver so a frontend can
// chain, and tolerates a nil child so callers need not guard optional slots.
func (n *Node) Add(role Role, child *Node) *Node {
	if child == nil {
		return n
	}
	n.Kids = append(n.Kids, Child{Role: role, Node: child})
	return n
}

// Slot returns the first child in the given slot, or nil.
func (n *Node) Slot(role Role) *Node {
	if n == nil {
		return nil
	}
	for _, k := range n.Kids {
		if k.Role == role {
			return k.Node
		}
	}
	return nil
}

// Slots returns every child in the given slot, in order.
func (n *Node) Slots(role Role) []*Node {
	if n == nil {
		return nil
	}
	var out []*Node
	for _, k := range n.Kids {
		if k.Role == role {
			out = append(out, k.Node)
		}
	}
	return out
}

// Children returns every child in order, slots ignored.
func (n *Node) Children() []*Node {
	if n == nil {
		return nil
	}
	out := make([]*Node, 0, len(n.Kids))
	for _, k := range n.Kids {
		out = append(out, k.Node)
	}
	return out
}

// Inspect walks the tree depth-first in child order, calling f for each node.
// When f returns true the node's children are walked and f is then called
// once with nil, which is the after-children hook — the same contract as Go's
// ast.Inspect, and load-bearing here: fingerprint.walk pairs each nil with a
// pushed nesting level to know when a construct's children have ended.
func Inspect(n *Node, f func(*Node) bool) {
	if n == nil {
		return
	}
	if !f(n) {
		return
	}
	for _, k := range n.Kids {
		Inspect(k.Node, f)
	}
	f(nil)
}

// Param is one declared parameter or result. Type is already rendered text —
// the IR never models a type system, because no consumer reads one: the
// similarity score takes the set of type strings, and the signature line
// takes them in order.
type Param struct {
	Name string
	Type string
}

// Func is one extracted function or method.
//
// Body is nil for a declaration without one (external, forward-declared, or a
// frontend that could not segment it), which is what makes the zero
// Fingerprint mean "no body".
//
// Type is the signature as a subtree, when the frontend has one. It is
// deliberately separate from Body: nothing in the fingerprint walks it, so it
// never reaches the node count or the token stream, and it exists for the
// evidence channels that must see a type the body never mentions — a function
// taking a channel is coordinating concurrent work whether or not it says so
// in its body.
//
// Canon is the body rewritten into the frontend's canonical shape, and
// CanonRules names the normalizations that got it there. Both are optional:
// canonicalization is a *semantics* question — whether two spellings of a
// guard mean the same thing — so it can only be answered by the language,
// which is why it belongs to the frontend and not here. A frontend without
// one leaves Canon nil, and Shape then falls back to the body as written.
type Func struct {
	Name        string
	Receiver    string
	Doc         string
	Params      []Param
	Results     []Param
	Body        *Node
	Type        *Node
	Canon       *Node
	CanonRules  []string
	Source      string
	StartLine   int
	StartOffset int
	EndOffset   int
	Exported    bool
	Callees     []string
}

// Shape is the tree a structural key is computed over: the canonical body
// when the frontend produced one, and the body as written otherwise.
//
// The fallback is not a degradation to nothing. A bag over the raw body is a
// perfectly well-formed shape key; it simply carries whichever incidental
// choices a canonicalizer would have removed, so two spellings of one shape
// stay apart. What a language gains by having a canonicalizer is recall on
// exactly those pairs — never correctness.
func (f *Func) Shape() *Node {
	if f == nil {
		return nil
	}
	if f.Canon != nil {
		return f.Canon
	}
	return f.Body
}

// Import is one resolved import binding: the name it is reachable by inside
// the file, and the path it names.
type Import struct {
	Local string
	Path  string
}

// File is one parsed source file.
//
// Package is whatever partitions functions into habitats for this language —
// a package clause where one exists, and the containing directory where it
// does not. Consumers treat it as an opaque key.
type File struct {
	Path      string
	Package   string
	Lang      string
	Imports   []Import
	Generated bool
	Funcs     []Func
}

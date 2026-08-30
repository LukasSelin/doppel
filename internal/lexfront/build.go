package lexfront

import (
	"strings"

	"github.com/LukasSelin/doppel/internal/syntax"
)

// keywordKind maps the control-flow words that recur across languages onto
// the IR's kinds. It is shared rather than per-language on purpose: these
// spellings are near-universal, and a language that uses a different one
// simply produces KindOther, which still counts as a node and still carries
// its children.
var keywordKind = map[string]syntax.Kind{
	"if": syntax.KindIf, "elif": syntax.KindIf, "elsif": syntax.KindIf,
	"for": syntax.KindFor, "while": syntax.KindFor, "foreach": syntax.KindFor,
	"loop": syntax.KindFor, "until": syntax.KindFor,
	"switch": syntax.KindSwitch, "match": syntax.KindSwitch, "case": syntax.KindSwitch,
	"when":   syntax.KindSwitch,
	"return": syntax.KindReturn, "yield": syntax.KindReturn,
	"defer": syntax.KindDefer, "ensure": syntax.KindDefer, "finally": syntax.KindDefer,
	"go": syntax.KindGo, "spawn": syntax.KindGo, "await": syntax.KindGo,
	"select": syntax.KindSelect,
	"break":  syntax.KindBranch, "continue": syntax.KindBranch, "goto": syntax.KindBranch,
	"pass": syntax.KindEmpty,
}

// assignOps are the operators that make a statement an assignment.
var assignOps = map[string]bool{
	"=": true, ":=": true, "+=": true, "-=": true, "*=": true, "/=": true,
	"%=": true, "&=": true, "|=": true, "^=": true, "<<=": true, ">>=": true,
	"**=": true, "??=": true, "&&=": true, "||=": true,
}

// builder turns a token span into a shallow syntax tree.
//
// "Shallow" is the honest word: this is not a parser and does not resolve
// precedence, so `a + b * c` becomes one binary node over its operands rather
// than the correct tree. What it does guarantee is *consistency* — the same
// token shape always produces the same node shape — which is what a
// similarity score actually needs. Two functions written the same way
// fingerprint the same way whether or not the tree is the one a compiler
// would build.
type builder struct {
	toks []token
	sp   Spec
}

// block builds a KindBlock over the statements in [from, to).
func (b *builder) block(from, to int) *syntax.Node {
	n := &syntax.Node{Kind: syntax.KindBlock}
	for i := from; i < to; {
		end := b.stmtEnd(i, to)
		if end <= i {
			i++
			continue
		}
		if s := b.stmt(i, end); s != nil {
			n.Add(syntax.RoleList, s)
		}
		i = end
	}
	return n
}

// stmtEnd finds where the statement starting at i ends: a ";" at depth zero,
// a "}" closing a nested block, or a newline that is not a continuation.
func (b *builder) stmtEnd(i, to int) int {
	depth := 0
	for j := i; j < to; j++ {
		t := b.toks[j]
		switch t.kind {
		case tokOpen:
			if t.text == "{" && depth == 0 && j > i {
				// A brace-delimited block belongs to this statement.
				if c := matchBracket(b.toks, j); c >= 0 && c < to {
					return c + 1
				}
			}
			depth++
		case tokClose:
			depth--
			if depth < 0 {
				return j
			}
		case tokSemi:
			if depth == 0 && t.text == ";" {
				return j + 1
			}
		}
		// A newline at depth zero ends a statement in languages without
		// terminators, unless the previous token clearly continues it.
		if j+1 < to && b.toks[j+1].nlPrev && depth == 0 && !continues(b.toks[j]) {
			return j + 1
		}
	}
	return to
}

// continues reports whether a token cannot be the last of a statement, which
// is how a wrapped expression stays one statement in a language with no
// terminator.
func continues(t token) bool {
	if t.kind == tokOp {
		return t.text != "++" && t.text != "--"
	}
	return t.kind == tokSemi || t.kind == tokOpen
}

// stmt classifies one statement and builds it.
func (b *builder) stmt(from, to int) *syntax.Node {
	from = b.skipTrivia(from, to)
	if from >= to {
		return nil
	}
	head := b.toks[from]

	// A bare block.
	if head.kind == tokOpen && head.text == "{" {
		if c := matchBracket(b.toks, from); c >= 0 {
			return b.block(from+1, c)
		}
	}

	if head.kind == tokIdent {
		if kind, ok := keywordKind[head.text]; ok {
			return b.keywordStmt(kind, head.text, from, to)
		}
	}

	// An assignment: an operator from assignOps at depth zero.
	if op, at := b.topLevelAssign(from, to); at >= 0 {
		n := &syntax.Node{Kind: syntax.KindAssign, Label: assignLabel(op)}
		for _, e := range b.exprList(from, at) {
			n.Add(syntax.RoleLhs, e)
		}
		for _, e := range b.exprList(at+1, to) {
			n.Add(syntax.RoleRhs, e)
		}
		return n
	}

	// Otherwise an expression statement.
	n := &syntax.Node{Kind: syntax.KindExprStmt}
	if e := b.expr(from, to); e != nil {
		n.Add(syntax.RoleX, e)
	}
	return n
}

// keywordStmt builds the control-flow statements, splitting the header from
// the body so the slots the pattern renders read are populated.
func (b *builder) keywordStmt(kind syntax.Kind, word string, from, to int) *syntax.Node {
	n := &syntax.Node{Kind: kind}
	if kind == syntax.KindBranch {
		n.Label = word
		return n
	}

	// Find the body: a "{" at depth zero, or the whole rest for suites.
	bodyOpen := -1
	depth := 0
	for j := from + 1; j < to; j++ {
		t := b.toks[j]
		if t.kind == tokOpen {
			if t.text == "{" && depth == 0 {
				bodyOpen = j
				break
			}
			depth++
		} else if t.kind == tokClose {
			depth--
		}
	}

	headerEnd := to
	if bodyOpen >= 0 {
		headerEnd = bodyOpen
	}

	// The header, minus the keyword itself.
	switch kind {
	case syntax.KindReturn:
		for _, e := range b.exprList(from+1, headerEnd) {
			n.Add(syntax.RoleResult, e)
		}
	case syntax.KindDefer, syntax.KindGo:
		if e := b.expr(from+1, headerEnd); e != nil {
			n.Add(syntax.RoleCall, e)
		}
	case syntax.KindSwitch:
		if e := b.expr(from+1, headerEnd); e != nil {
			n.Add(syntax.RoleTag, e)
		}
	case syntax.KindFor:
		// A three-clause header splits on ";"; anything else is one
		// condition, which is right for while/foreach/range loops.
		parts := b.splitTop(from+1, headerEnd, ";")
		switch len(parts) {
		case 3:
			roles := []syntax.Role{syntax.RoleInit, syntax.RoleCond, syntax.RolePost}
			for i, p := range parts {
				if e := b.expr(p[0], p[1]); e != nil {
					n.Add(roles[i], e)
				}
			}
		default:
			if e := b.expr(from+1, headerEnd); e != nil {
				n.Add(syntax.RoleX, e)
				// A loop with one header expression reads as both the thing
				// iterated and the condition; the pattern renders look for
				// RoleCond on a for and RoleX on a range, so give both rather
				// than guess which language this is.
				n.Add(syntax.RoleCond, e)
			}
		}
	default:
		if e := b.expr(from+1, headerEnd); e != nil {
			n.Add(syntax.RoleCond, e)
		}
	}

	if bodyOpen >= 0 {
		if c := matchBracket(b.toks, bodyOpen); c >= 0 {
			n.Add(syntax.RoleBody, b.block(bodyOpen+1, c))
			// Anything after the closing brace is an else/elsif chain.
			if c+1 < to {
				if e := b.stmt(c+1, to); e != nil {
					n.Add(syntax.RoleElse, e)
				}
			}
		}
	}
	return n
}

// topLevelAssign finds an assignment operator at bracket depth zero.
func (b *builder) topLevelAssign(from, to int) (string, int) {
	depth := 0
	for j := from; j < to; j++ {
		t := b.toks[j]
		switch t.kind {
		case tokOpen:
			depth++
		case tokClose:
			depth--
		case tokOp:
			if depth == 0 && assignOps[t.text] {
				return t.text, j
			}
		}
	}
	return "", -1
}

// assignLabel normalises an assignment operator to the token text Go's own
// frontend would produce, so a plain "=" reads the same across frontends.
func assignLabel(op string) string { return op }

// splitTop splits [from,to) on a separator at depth zero.
func (b *builder) splitTop(from, to int, sep string) [][2]int {
	var out [][2]int
	depth, start := 0, from
	for j := from; j < to; j++ {
		t := b.toks[j]
		switch t.kind {
		case tokOpen:
			depth++
		case tokClose:
			depth--
		case tokSemi:
			if depth == 0 && t.text == sep {
				out = append(out, [2]int{start, j})
				start = j + 1
			}
		}
	}
	out = append(out, [2]int{start, to})
	return out
}

// exprList splits on commas at depth zero and builds each part.
func (b *builder) exprList(from, to int) []*syntax.Node {
	var out []*syntax.Node
	for _, p := range b.splitTop(from, to, ",") {
		if e := b.expr(p[0], p[1]); e != nil {
			out = append(out, e)
		}
	}
	return out
}

// expr builds an expression over [from, to).
//
// The shape recognised, in order: a binary operator at depth zero splits the
// range; a trailing call applies; a dotted name is a selector; everything
// else is a leaf. Precedence is deliberately ignored — see builder.
func (b *builder) expr(from, to int) *syntax.Node {
	from = b.skipTrivia(from, to)
	to = b.trimTrivia(from, to)
	if from >= to {
		return nil
	}

	// Parenthesised whole: unwrap into a Paren node.
	if b.toks[from].kind == tokOpen && b.toks[from].text == "(" {
		if c := matchBracket(b.toks, from); c == to-1 {
			n := &syntax.Node{Kind: syntax.KindParen}
			n.Add(syntax.RoleX, b.expr(from+1, c))
			return n
		}
	}

	// Binary operator at depth zero. The last one wins, which makes the tree
	// left-heavy and — more importantly — deterministic.
	if op, at := b.topLevelBinary(from, to); at >= 0 {
		n := &syntax.Node{Kind: syntax.KindBinary, Label: op}
		n.Add(syntax.RoleX, b.expr(from, at))
		n.Add(syntax.RoleY, b.expr(at+1, to))
		return n
	}

	// Unary prefix.
	if b.toks[from].kind == tokOp && to-from > 1 {
		n := &syntax.Node{Kind: syntax.KindUnary, Label: b.toks[from].text}
		n.Add(syntax.RoleX, b.expr(from+1, to))
		return n
	}

	// A call: the range ends with a balanced "(...)".
	if b.toks[to-1].kind == tokClose && b.toks[to-1].text == ")" {
		if open := b.openerOf(from, to-1); open > from {
			n := &syntax.Node{Kind: syntax.KindCall}
			n.Add(syntax.RoleFun, b.expr(from, open))
			for _, a := range b.exprList(open+1, to-1) {
				n.Add(syntax.RoleArg, a)
			}
			return n
		}
	}

	// Index or slice: ends with a balanced "[...]".
	if b.toks[to-1].kind == tokClose && b.toks[to-1].text == "]" {
		if open := b.openerOf(from, to-1); open > from {
			n := &syntax.Node{Kind: syntax.KindIndex}
			n.Add(syntax.RoleX, b.expr(from, open))
			n.Add(syntax.RoleIndex, b.expr(open+1, to-1))
			return n
		}
	}

	// Composite literal: a whole "{...}" or "[...]".
	if b.toks[from].kind == tokOpen && (b.toks[from].text == "{" || b.toks[from].text == "[") {
		if c := matchBracket(b.toks, from); c == to-1 {
			n := &syntax.Node{Kind: syntax.KindComposite}
			for _, e := range b.exprList(from+1, c) {
				n.Add(syntax.RoleElt, e)
			}
			return n
		}
	}

	// A selector: "<expr> . name".
	if to-from >= 3 && b.toks[to-2].kind == tokOp &&
		(b.toks[to-2].text == "." || b.toks[to-2].text == "?." || b.toks[to-2].text == "::") &&
		b.toks[to-1].kind == tokIdent {
		n := &syntax.Node{Kind: syntax.KindSelector, Label: b.toks[to-1].text}
		n.Add(syntax.RoleX, b.expr(from, to-2))
		n.Add(syntax.RoleSel, &syntax.Node{Kind: syntax.KindIdent, Label: b.toks[to-1].text})
		return n
	}

	// Leaves.
	t := b.toks[from]
	switch t.kind {
	case tokIdent:
		if to-from == 1 {
			return &syntax.Node{Kind: syntax.KindIdent, Label: t.text}
		}
	case tokString:
		if to-from == 1 {
			return &syntax.Node{Kind: syntax.KindLit, Label: "STRING", Text: t.val}
		}
	case tokNumber:
		if to-from == 1 {
			return &syntax.Node{Kind: syntax.KindLit, Label: numKind(t.text)}
		}
	}

	// A run this shape rule does not recognise still has to become nodes:
	// dropping it would change the node count and lose its identifiers,
	// which are the evidence the tagger and lexicon read.
	n := &syntax.Node{Kind: syntax.KindOther}
	for j := from; j < to; j++ {
		switch b.toks[j].kind {
		case tokIdent:
			n.Add(syntax.RoleNone, &syntax.Node{Kind: syntax.KindIdent, Label: b.toks[j].text})
		case tokString:
			n.Add(syntax.RoleNone, &syntax.Node{Kind: syntax.KindLit, Label: "STRING", Text: b.toks[j].val})
		case tokNumber:
			n.Add(syntax.RoleNone, &syntax.Node{Kind: syntax.KindLit, Label: numKind(b.toks[j].text)})
		}
	}
	if len(n.Kids) == 0 {
		return nil
	}
	return n
}

func numKind(s string) string {
	if strings.ContainsAny(s, ".eE") && !strings.HasPrefix(s, "0x") {
		return "FLOAT"
	}
	return "INT"
}

// binaryOps are the operators that split an expression. Assignment is not
// here — a statement-level assignment is handled by stmt, and an embedded one
// is rare enough to fall through to the unrecognised-run case.
var binaryOps = map[string]bool{
	"+": true, "-": true, "*": true, "/": true, "%": true, "**": true,
	"==": true, "!=": true, "<": true, ">": true, "<=": true, ">=": true,
	"===": true, "!==": true, "&&": true, "||": true, "&": true, "|": true,
	"^": true, "<<": true, ">>": true, "??": true, "and": true, "or": true,
	"<-": true,
}

func (b *builder) topLevelBinary(from, to int) (string, int) {
	depth := 0
	op, at := "", -1
	for j := from; j < to; j++ {
		t := b.toks[j]
		switch t.kind {
		case tokOpen:
			depth++
		case tokClose:
			depth--
		case tokOp:
			// A leading operator is unary, not binary.
			if depth == 0 && binaryOps[t.text] && j > from {
				op, at = t.text, j
			}
		case tokIdent:
			// Python's word operators.
			if depth == 0 && binaryOps[t.text] && j > from && j < to-1 {
				op, at = t.text, j
			}
		}
	}
	return op, at
}

// openerOf returns the index of the bracket that close matches, provided it
// opens at or after from.
func (b *builder) openerOf(from, close int) int {
	depth := 0
	for j := close; j >= from; j-- {
		switch b.toks[j].kind {
		case tokClose:
			depth++
		case tokOpen:
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

func (b *builder) skipTrivia(from, to int) int {
	for from < to && b.toks[from].kind == tokSemi {
		from++
	}
	return from
}

func (b *builder) trimTrivia(from, to int) int {
	for to > from && b.toks[to-1].kind == tokSemi {
		to--
	}
	return to
}

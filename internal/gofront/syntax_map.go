package gofront

import (
	"go/ast"
	"strconv"

	"github.com/LukasSelin/doppel/internal/syntax"
)

// toSyntax maps a Go AST subtree onto the neutral IR.
//
// The traversal order and the node count are both observable downstream —
// Fingerprint.Nodes counts nodes, the token stream is emitted in visit order,
// and the depth histogram is driven by the push/pop pairing — so this builds
// the tree *from* ast.Inspect rather than reimplementing ast.Walk's per-type
// field ordering. Inheriting the order that way makes it correct by
// construction: there is no list of field orderings here to fall out of sync
// with go/ast when a node type gains a field.
//
// Roles cannot be inherited the same way, because position does not determine
// them: a for-loop with no Init has its Cond first. They are recovered by
// comparing each child against its parent's named fields, which is exact.
func toSyntax(root ast.Node) *syntax.Node {
	if root == nil {
		return nil
	}
	var out *syntax.Node
	var stack []*syntax.Node
	var parents []ast.Node

	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			if last := len(stack) - 1; last >= 0 {
				stack = stack[:last]
				parents = parents[:last]
			}
			return false
		}
		sn := &syntax.Node{Kind: kindOf(n), Label: labelOf(n), Text: textOf(n)}
		if len(stack) == 0 {
			out = sn
		} else {
			top := stack[len(stack)-1]
			top.Kids = append(top.Kids, syntax.Child{
				Role: roleOf(parents[len(parents)-1], n),
				Node: sn,
			})
		}
		stack = append(stack, sn)
		parents = append(parents, n)
		return true
	})
	return out
}

// kindOf classifies a Go node. Everything the scoring stages do not name maps
// to KindOther, which still exists as a node — dropping it would change
// Fingerprint.Nodes and with it the --min-nodes gate and SizeRatio.
func kindOf(n ast.Node) syntax.Kind {
	switch n.(type) {
	case *ast.IfStmt:
		return syntax.KindIf
	case *ast.ForStmt:
		return syntax.KindFor
	case *ast.RangeStmt:
		return syntax.KindRange
	case *ast.SwitchStmt:
		return syntax.KindSwitch
	case *ast.TypeSwitchStmt:
		return syntax.KindTypeSwitch
	case *ast.SelectStmt:
		return syntax.KindSelect
	case *ast.ReturnStmt:
		return syntax.KindReturn
	case *ast.DeferStmt:
		return syntax.KindDefer
	case *ast.GoStmt:
		return syntax.KindGo
	case *ast.BlockStmt:
		return syntax.KindBlock
	case *ast.AssignStmt:
		return syntax.KindAssign
	case *ast.BranchStmt:
		return syntax.KindBranch
	case *ast.IncDecStmt:
		return syntax.KindIncDec
	case *ast.SendStmt:
		return syntax.KindSend
	case *ast.ExprStmt:
		return syntax.KindExprStmt
	case *ast.DeclStmt:
		return syntax.KindDeclStmt
	case *ast.LabeledStmt:
		return syntax.KindLabeled
	case *ast.EmptyStmt:
		return syntax.KindEmpty
	case *ast.CaseClause:
		return syntax.KindCaseClause
	case *ast.CommClause:
		return syntax.KindCommClause
	case *ast.BadStmt:
		return syntax.KindBadStmt
	case *ast.CallExpr:
		return syntax.KindCall
	case *ast.BinaryExpr:
		return syntax.KindBinary
	case *ast.UnaryExpr:
		return syntax.KindUnary
	case *ast.Ident:
		return syntax.KindIdent
	case *ast.BasicLit:
		return syntax.KindLit
	case *ast.SelectorExpr:
		return syntax.KindSelector
	case *ast.IndexExpr:
		return syntax.KindIndex
	case *ast.SliceExpr:
		return syntax.KindSlice
	case *ast.StarExpr:
		return syntax.KindStar
	case *ast.TypeAssertExpr:
		return syntax.KindAssert
	case *ast.CompositeLit:
		return syntax.KindComposite
	case *ast.KeyValueExpr:
		return syntax.KindKeyValue
	case *ast.FuncLit:
		return syntax.KindFuncLit
	case *ast.ParenExpr:
		return syntax.KindParen
	case *ast.ValueSpec:
		return syntax.KindValueSpec
	case *ast.ChanType:
		return syntax.KindChanType
	case *ast.Ellipsis:
		return syntax.KindEllipsis
	case *ast.IndexListExpr:
		return syntax.KindIndexList
	case *ast.ArrayType:
		return syntax.KindArrayType
	case *ast.StructType:
		return syntax.KindStructType
	case *ast.FuncType:
		return syntax.KindFuncType
	case *ast.InterfaceType:
		return syntax.KindInterfaceType
	case *ast.MapType:
		return syntax.KindMapType
	case *ast.Field:
		return syntax.KindField
	case *ast.FieldList:
		return syntax.KindFieldList
	case *ast.GenDecl:
		return syntax.KindGenDecl
	case *ast.TypeSpec:
		return syntax.KindTypeSpec
	case *ast.ImportSpec:
		return syntax.KindImportSpec
	case *ast.BadExpr:
		return syntax.KindBadExpr
	case *ast.BadDecl:
		return syntax.KindBadDecl
	}
	return syntax.KindOther
}

// labelOf carries the one lexical detail each kind needs. The strings are
// hashed downstream (walk's token stream, the label bag's label_0), so they
// must stay exactly what the Go-typed code used to read: Op.String(),
// Tok.String(), Kind.String(), and the bare identifier name.
//
// IncDecStmt, GenDecl and ChanType are here for the label bag alone — the
// token stream emits none of the three, or emits them without their token.
// Where go/ast folds several constructs into one struct behind a token
// field, that token is part of the node's kind: ++ and -- are not one
// statement, const and var are not one declaration, and a receive-only
// channel is not a send-only one.
func labelOf(n ast.Node) string {
	switch v := n.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.BasicLit:
		return v.Kind.String()
	case *ast.BinaryExpr:
		return v.Op.String()
	case *ast.UnaryExpr:
		return v.Op.String()
	case *ast.AssignStmt:
		return v.Tok.String()
	case *ast.BranchStmt:
		return v.Tok.String()
	case *ast.IncDecStmt:
		return v.Tok.String()
	case *ast.GenDecl:
		return v.Tok.String()
	case *ast.ChanType:
		return chanDir(v.Dir)
	case *ast.SelectorExpr:
		if v.Sel != nil {
			return v.Sel.Name
		}
	}
	return ""
}

// chanDir names a channel direction. ast.ChanDir is a bit set, so the
// bidirectional case is both bits.
func chanDir(d ast.ChanDir) string {
	switch d {
	case ast.SEND:
		return "send"
	case ast.RECV:
		return "recv"
	}
	return "both"
}

// textOf decodes a literal's value. Unquoting is lexical work only the
// frontend can do — escape rules are the language's — so the IR carries the
// result rather than the raw lexeme. A string that will not unquote keeps its
// raw form, which is the pre-existing behaviour for backtick and malformed
// literals.
func textOf(n ast.Node) string {
	lit, ok := n.(*ast.BasicLit)
	if !ok {
		return ""
	}
	if lit.Kind.String() != "STRING" {
		return lit.Value
	}
	if v, err := strconv.Unquote(lit.Value); err == nil {
		return v
	}
	return lit.Value
}

// roleOf recovers which slot of parent the child fills, by identity against
// the parent's named fields. Anything unnamed here is RoleNone, which is not
// a defect: a consumer asking for a slot it did not get handles absence, and
// the node is still present for counting and tokenizing.
func roleOf(parent, child ast.Node) syntax.Role {
	switch v := parent.(type) {
	case *ast.IfStmt:
		switch {
		case same(v.Init, child):
			return syntax.RoleInit
		case same(v.Cond, child):
			return syntax.RoleCond
		case same(v.Body, child):
			return syntax.RoleBody
		case same(v.Else, child):
			return syntax.RoleElse
		}
	case *ast.ForStmt:
		switch {
		case same(v.Init, child):
			return syntax.RoleInit
		case same(v.Cond, child):
			return syntax.RoleCond
		case same(v.Post, child):
			return syntax.RolePost
		case same(v.Body, child):
			return syntax.RoleBody
		}
	case *ast.RangeStmt:
		switch {
		case same(v.Key, child):
			return syntax.RoleKey
		case same(v.Value, child):
			return syntax.RoleValue
		case same(v.X, child):
			return syntax.RoleX
		case same(v.Body, child):
			return syntax.RoleBody
		}
	case *ast.SwitchStmt:
		switch {
		case same(v.Init, child):
			return syntax.RoleInit
		case same(v.Tag, child):
			return syntax.RoleTag
		case same(v.Body, child):
			return syntax.RoleBody
		}
	case *ast.TypeSwitchStmt:
		switch {
		case same(v.Init, child):
			return syntax.RoleInit
		case same(v.Assign, child):
			return syntax.RoleTag
		case same(v.Body, child):
			return syntax.RoleBody
		}
	case *ast.SelectStmt:
		if same(v.Body, child) {
			return syntax.RoleBody
		}
	case *ast.ReturnStmt:
		if anySame(v.Results, child) {
			return syntax.RoleResult
		}
	case *ast.AssignStmt:
		switch {
		case anySame(v.Lhs, child):
			return syntax.RoleLhs
		case anySame(v.Rhs, child):
			return syntax.RoleRhs
		}
	case *ast.DeferStmt:
		if same(v.Call, child) {
			return syntax.RoleCall
		}
	case *ast.GoStmt:
		if same(v.Call, child) {
			return syntax.RoleCall
		}
	case *ast.SendStmt:
		switch {
		case same(v.Chan, child):
			return syntax.RoleChan
		case same(v.Value, child):
			return syntax.RoleValue
		}
	case *ast.ExprStmt:
		if same(v.X, child) {
			return syntax.RoleX
		}
	case *ast.IncDecStmt:
		if same(v.X, child) {
			return syntax.RoleX
		}
	case *ast.BlockStmt:
		if anyStmtSame(v.List, child) {
			return syntax.RoleList
		}
	case *ast.CaseClause:
		switch {
		case anySame(v.List, child):
			return syntax.RoleList
		case anyStmtSame(v.Body, child):
			return syntax.RoleBody
		}
	case *ast.CommClause:
		switch {
		case same(v.Comm, child):
			return syntax.RoleInit
		case anyStmtSame(v.Body, child):
			return syntax.RoleBody
		}
	case *ast.LabeledStmt:
		switch {
		case same(v.Label, child):
			return syntax.RoleName
		case same(v.Stmt, child):
			return syntax.RoleBody
		}
	case *ast.CallExpr:
		switch {
		case same(v.Fun, child):
			return syntax.RoleFun
		case anySame(v.Args, child):
			return syntax.RoleArg
		}
	case *ast.BinaryExpr:
		switch {
		case same(v.X, child):
			return syntax.RoleX
		case same(v.Y, child):
			return syntax.RoleY
		}
	case *ast.UnaryExpr:
		if same(v.X, child) {
			return syntax.RoleX
		}
	case *ast.ParenExpr:
		if same(v.X, child) {
			return syntax.RoleX
		}
	case *ast.StarExpr:
		if same(v.X, child) {
			return syntax.RoleX
		}
	case *ast.SelectorExpr:
		switch {
		case same(v.X, child):
			return syntax.RoleX
		case same(v.Sel, child):
			return syntax.RoleSel
		}
	case *ast.IndexExpr:
		switch {
		case same(v.X, child):
			return syntax.RoleX
		case same(v.Index, child):
			return syntax.RoleIndex
		}
	case *ast.SliceExpr:
		if same(v.X, child) {
			return syntax.RoleX
		}
	case *ast.TypeAssertExpr:
		switch {
		case same(v.X, child):
			return syntax.RoleX
		case same(v.Type, child):
			return syntax.RoleType
		}
	case *ast.CompositeLit:
		switch {
		case same(v.Type, child):
			return syntax.RoleType
		case anySame(v.Elts, child):
			return syntax.RoleElt
		}
	case *ast.KeyValueExpr:
		switch {
		case same(v.Key, child):
			return syntax.RoleKey
		case same(v.Value, child):
			return syntax.RoleValue
		}
	case *ast.FuncLit:
		switch {
		case same(v.Type, child):
			return syntax.RoleType
		case same(v.Body, child):
			return syntax.RoleBody
		}
	case *ast.ValueSpec:
		switch {
		case anyIdentSame(v.Names, child):
			return syntax.RoleName
		case same(v.Type, child):
			return syntax.RoleType
		case anySame(v.Values, child):
			return syntax.RoleValue
		}
	}
	return syntax.RoleNone
}

// same compares a possibly-absent field against a child. A nil field is a nil
// interface and never matches, which is exactly the "slot absent" case.
func same(field, child ast.Node) bool {
	if field == nil || child == nil {
		return false
	}
	return field == child
}

func anySame(fields []ast.Expr, child ast.Node) bool {
	for _, f := range fields {
		if same(f, child) {
			return true
		}
	}
	return false
}

func anyStmtSame(fields []ast.Stmt, child ast.Node) bool {
	for _, f := range fields {
		if same(f, child) {
			return true
		}
	}
	return false
}

func anyIdentSame(fields []*ast.Ident, child ast.Node) bool {
	for _, f := range fields {
		if f != nil && ast.Node(f) == child {
			return true
		}
	}
	return false
}

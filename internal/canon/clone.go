package canon

import (
	"go/ast"
	"go/token"
)

// Clone returns a deep, position-free copy of a function declaration.
//
// It exists because parser.Parse hands the same *ast.FuncDecl to
// fingerprint.Build and to extractSignals, and canonicalization rewrites
// nodes. Rewriting in place would silently change every existing score and
// every tag. The stdlib has no ast deep-clone and this module takes no
// dependency beyond Cobra, so the clone is written out here, one case per
// node kind go/ast defines.
//
// Two things are deliberately dropped rather than copied:
//
//   - Every position is set to token.NoPos. The canonical tree is a shape,
//     not a location: keeping offsets from the original file would make two
//     structurally identical functions compare unequal through their
//     positions, and would make the printed form depend on where in a file
//     the function happened to sit. go/printer renders a position-free tree
//     in a canonical layout, which is exactly what a canonical form wants.
//   - Comments (Doc and the trailing Comment fields, and File-level comment
//     maps) are not carried over. The tagger already refuses to read comments
//     as evidence; a canonical shape has no room for them either.
//
// ast.Object and ast.Scope are the deprecated resolver fields; they are left
// nil, since nothing in this module reads them and copying them would share
// mutable state with the original tree.
func Clone(fd *ast.FuncDecl) *ast.FuncDecl {
	return cloneFuncDecl(fd)
}

func cloneFuncDecl(fd *ast.FuncDecl) *ast.FuncDecl {
	if fd == nil {
		return nil
	}
	return &ast.FuncDecl{
		Recv: cloneFieldList(fd.Recv),
		Name: cloneIdent(fd.Name),
		Type: cloneFuncType(fd.Type),
		Body: cloneBlock(fd.Body),
	}
}

func cloneIdent(id *ast.Ident) *ast.Ident {
	if id == nil {
		return nil
	}
	return &ast.Ident{NamePos: token.NoPos, Name: id.Name}
}

func cloneIdents(ids []*ast.Ident) []*ast.Ident {
	if ids == nil {
		return nil
	}
	out := make([]*ast.Ident, len(ids))
	for i, id := range ids {
		out[i] = cloneIdent(id)
	}
	return out
}

func cloneFuncType(ft *ast.FuncType) *ast.FuncType {
	if ft == nil {
		return nil
	}
	return &ast.FuncType{
		Func:       token.NoPos,
		TypeParams: cloneFieldList(ft.TypeParams),
		Params:     cloneFieldList(ft.Params),
		Results:    cloneFieldList(ft.Results),
	}
}

func cloneFieldList(fl *ast.FieldList) *ast.FieldList {
	if fl == nil {
		return nil
	}
	out := &ast.FieldList{Opening: token.NoPos, Closing: token.NoPos}
	if fl.List != nil {
		out.List = make([]*ast.Field, len(fl.List))
		for i, f := range fl.List {
			out.List[i] = cloneField(f)
		}
	}
	return out
}

func cloneField(f *ast.Field) *ast.Field {
	if f == nil {
		return nil
	}
	return &ast.Field{
		Names: cloneIdents(f.Names),
		Type:  cloneExpr(f.Type),
		Tag:   cloneBasicLit(f.Tag),
	}
}

func cloneBasicLit(l *ast.BasicLit) *ast.BasicLit {
	if l == nil {
		return nil
	}
	return &ast.BasicLit{ValuePos: token.NoPos, Kind: l.Kind, Value: l.Value}
}

func cloneExprs(list []ast.Expr) []ast.Expr {
	if list == nil {
		return nil
	}
	out := make([]ast.Expr, len(list))
	for i, e := range list {
		out[i] = cloneExpr(e)
	}
	return out
}

// cloneExpr covers every ast.Expr implementation. An unknown kind (there is
// none today; the compiler cannot prove that, since ast.Expr is an open
// interface) returns nil rather than aliasing the original node — sharing a
// node would reintroduce exactly the in-place mutation this file exists to
// prevent.
func cloneExpr(e ast.Expr) ast.Expr {
	switch n := e.(type) {
	case nil:
		return nil
	case *ast.BadExpr:
		return &ast.BadExpr{From: token.NoPos, To: token.NoPos}
	case *ast.Ident:
		return cloneIdent(n)
	case *ast.Ellipsis:
		return &ast.Ellipsis{Ellipsis: token.NoPos, Elt: cloneExpr(n.Elt)}
	case *ast.BasicLit:
		return cloneBasicLit(n)
	case *ast.FuncLit:
		return &ast.FuncLit{Type: cloneFuncType(n.Type), Body: cloneBlock(n.Body)}
	case *ast.CompositeLit:
		return &ast.CompositeLit{
			Type:       cloneExpr(n.Type),
			Lbrace:     token.NoPos,
			Elts:       cloneExprs(n.Elts),
			Rbrace:     token.NoPos,
			Incomplete: n.Incomplete,
		}
	case *ast.ParenExpr:
		return &ast.ParenExpr{Lparen: token.NoPos, X: cloneExpr(n.X), Rparen: token.NoPos}
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{X: cloneExpr(n.X), Sel: cloneIdent(n.Sel)}
	case *ast.IndexExpr:
		return &ast.IndexExpr{
			X: cloneExpr(n.X), Lbrack: token.NoPos, Index: cloneExpr(n.Index), Rbrack: token.NoPos,
		}
	case *ast.IndexListExpr:
		return &ast.IndexListExpr{
			X: cloneExpr(n.X), Lbrack: token.NoPos, Indices: cloneExprs(n.Indices), Rbrack: token.NoPos,
		}
	case *ast.SliceExpr:
		return &ast.SliceExpr{
			X: cloneExpr(n.X), Lbrack: token.NoPos,
			Low: cloneExpr(n.Low), High: cloneExpr(n.High), Max: cloneExpr(n.Max),
			Slice3: n.Slice3, Rbrack: token.NoPos,
		}
	case *ast.TypeAssertExpr:
		return &ast.TypeAssertExpr{
			X: cloneExpr(n.X), Lparen: token.NoPos, Type: cloneExpr(n.Type), Rparen: token.NoPos,
		}
	case *ast.CallExpr:
		return &ast.CallExpr{
			Fun: cloneExpr(n.Fun), Lparen: token.NoPos, Args: cloneExprs(n.Args),
			Ellipsis: ellipsisFlag(n.Ellipsis), Rparen: token.NoPos,
		}
	case *ast.StarExpr:
		return &ast.StarExpr{Star: token.NoPos, X: cloneExpr(n.X)}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{OpPos: token.NoPos, Op: n.Op, X: cloneExpr(n.X)}
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{X: cloneExpr(n.X), OpPos: token.NoPos, Op: n.Op, Y: cloneExpr(n.Y)}
	case *ast.KeyValueExpr:
		return &ast.KeyValueExpr{Key: cloneExpr(n.Key), Colon: token.NoPos, Value: cloneExpr(n.Value)}
	case *ast.ArrayType:
		return &ast.ArrayType{Lbrack: token.NoPos, Len: cloneExpr(n.Len), Elt: cloneExpr(n.Elt)}
	case *ast.StructType:
		return &ast.StructType{Struct: token.NoPos, Fields: cloneFieldList(n.Fields), Incomplete: n.Incomplete}
	case *ast.FuncType:
		return cloneFuncType(n)
	case *ast.InterfaceType:
		return &ast.InterfaceType{Interface: token.NoPos, Methods: cloneFieldList(n.Methods), Incomplete: n.Incomplete}
	case *ast.MapType:
		return &ast.MapType{Map: token.NoPos, Key: cloneExpr(n.Key), Value: cloneExpr(n.Value)}
	case *ast.ChanType:
		return &ast.ChanType{Begin: token.NoPos, Arrow: token.NoPos, Dir: n.Dir, Value: cloneExpr(n.Value)}
	}
	return nil
}

// ellipsisFlag preserves the *presence* of a call's "..." (go/printer only
// checks CallExpr.Ellipsis against token.NoPos) without preserving the offset.
func ellipsisFlag(p token.Pos) token.Pos {
	if p.IsValid() {
		return 1
	}
	return token.NoPos
}

func cloneBlock(b *ast.BlockStmt) *ast.BlockStmt {
	if b == nil {
		return nil
	}
	return &ast.BlockStmt{Lbrace: token.NoPos, List: cloneStmts(b.List), Rbrace: token.NoPos}
}

func cloneStmts(list []ast.Stmt) []ast.Stmt {
	if list == nil {
		return nil
	}
	out := make([]ast.Stmt, len(list))
	for i, s := range list {
		out[i] = cloneStmt(s)
	}
	return out
}

// cloneStmt covers every ast.Stmt implementation, with the same
// no-aliasing rule as cloneExpr.
func cloneStmt(s ast.Stmt) ast.Stmt {
	switch n := s.(type) {
	case nil:
		return nil
	case *ast.BadStmt:
		return &ast.BadStmt{From: token.NoPos, To: token.NoPos}
	case *ast.DeclStmt:
		return &ast.DeclStmt{Decl: cloneDecl(n.Decl)}
	case *ast.EmptyStmt:
		return &ast.EmptyStmt{Semicolon: token.NoPos, Implicit: n.Implicit}
	case *ast.LabeledStmt:
		return &ast.LabeledStmt{Label: cloneIdent(n.Label), Colon: token.NoPos, Stmt: cloneStmt(n.Stmt)}
	case *ast.ExprStmt:
		return &ast.ExprStmt{X: cloneExpr(n.X)}
	case *ast.SendStmt:
		return &ast.SendStmt{Chan: cloneExpr(n.Chan), Arrow: token.NoPos, Value: cloneExpr(n.Value)}
	case *ast.IncDecStmt:
		return &ast.IncDecStmt{X: cloneExpr(n.X), TokPos: token.NoPos, Tok: n.Tok}
	case *ast.AssignStmt:
		return &ast.AssignStmt{
			Lhs: cloneExprs(n.Lhs), TokPos: token.NoPos, Tok: n.Tok, Rhs: cloneExprs(n.Rhs),
		}
	case *ast.GoStmt:
		return &ast.GoStmt{Go: token.NoPos, Call: cloneCall(n.Call)}
	case *ast.DeferStmt:
		return &ast.DeferStmt{Defer: token.NoPos, Call: cloneCall(n.Call)}
	case *ast.ReturnStmt:
		return &ast.ReturnStmt{Return: token.NoPos, Results: cloneExprs(n.Results)}
	case *ast.BranchStmt:
		return &ast.BranchStmt{TokPos: token.NoPos, Tok: n.Tok, Label: cloneIdent(n.Label)}
	case *ast.BlockStmt:
		return cloneBlock(n)
	case *ast.IfStmt:
		return &ast.IfStmt{
			If: token.NoPos, Init: cloneStmt(n.Init), Cond: cloneExpr(n.Cond),
			Body: cloneBlock(n.Body), Else: cloneStmt(n.Else),
		}
	case *ast.CaseClause:
		return &ast.CaseClause{
			Case: token.NoPos, List: cloneExprs(n.List), Colon: token.NoPos, Body: cloneStmts(n.Body),
		}
	case *ast.SwitchStmt:
		return &ast.SwitchStmt{
			Switch: token.NoPos, Init: cloneStmt(n.Init), Tag: cloneExpr(n.Tag), Body: cloneBlock(n.Body),
		}
	case *ast.TypeSwitchStmt:
		return &ast.TypeSwitchStmt{
			Switch: token.NoPos, Init: cloneStmt(n.Init), Assign: cloneStmt(n.Assign), Body: cloneBlock(n.Body),
		}
	case *ast.CommClause:
		return &ast.CommClause{
			Case: token.NoPos, Comm: cloneStmt(n.Comm), Colon: token.NoPos, Body: cloneStmts(n.Body),
		}
	case *ast.SelectStmt:
		return &ast.SelectStmt{Select: token.NoPos, Body: cloneBlock(n.Body)}
	case *ast.ForStmt:
		return &ast.ForStmt{
			For: token.NoPos, Init: cloneStmt(n.Init), Cond: cloneExpr(n.Cond),
			Post: cloneStmt(n.Post), Body: cloneBlock(n.Body),
		}
	case *ast.RangeStmt:
		return &ast.RangeStmt{
			For: token.NoPos, Key: cloneExpr(n.Key), Value: cloneExpr(n.Value),
			TokPos: token.NoPos, Tok: n.Tok, Range: token.NoPos,
			X: cloneExpr(n.X), Body: cloneBlock(n.Body),
		}
	}
	return nil
}

func cloneCall(c *ast.CallExpr) *ast.CallExpr {
	if c == nil {
		return nil
	}
	cloned, _ := cloneExpr(c).(*ast.CallExpr)
	return cloned
}

// cloneDecl covers the declarations reachable from inside a function body:
// a DeclStmt wraps a GenDecl (var, const, type). FuncDecl is handled by
// cloneFuncDecl and never nests.
func cloneDecl(d ast.Decl) ast.Decl {
	switch n := d.(type) {
	case nil:
		return nil
	case *ast.BadDecl:
		return &ast.BadDecl{From: token.NoPos, To: token.NoPos}
	case *ast.GenDecl:
		out := &ast.GenDecl{
			TokPos: token.NoPos, Tok: n.Tok, Lparen: parenFlag(n.Lparen), Rparen: parenFlag(n.Rparen),
		}
		if n.Specs != nil {
			out.Specs = make([]ast.Spec, len(n.Specs))
			for i, s := range n.Specs {
				out.Specs[i] = cloneSpec(s)
			}
		}
		return out
	case *ast.FuncDecl:
		return cloneFuncDecl(n)
	}
	return nil
}

// parenFlag preserves whether a GenDecl was parenthesized — go/printer tests
// Lparen.IsValid() to decide between "var x int" and "var (\n\tx int\n)".
func parenFlag(p token.Pos) token.Pos {
	if p.IsValid() {
		return 1
	}
	return token.NoPos
}

func cloneSpec(s ast.Spec) ast.Spec {
	switch n := s.(type) {
	case nil:
		return nil
	case *ast.ImportSpec:
		return &ast.ImportSpec{
			Name: cloneIdent(n.Name), Path: cloneBasicLit(n.Path), EndPos: token.NoPos,
		}
	case *ast.ValueSpec:
		return &ast.ValueSpec{
			Names: cloneIdents(n.Names), Type: cloneExpr(n.Type), Values: cloneExprs(n.Values),
		}
	case *ast.TypeSpec:
		return &ast.TypeSpec{
			Name: cloneIdent(n.Name), TypeParams: cloneFieldList(n.TypeParams),
			Assign: assignFlag(n.Assign), Type: cloneExpr(n.Type),
		}
	}
	return nil
}

// assignFlag preserves whether a TypeSpec is an alias ("type A = B").
func assignFlag(p token.Pos) token.Pos {
	if p.IsValid() {
		return 1
	}
	return token.NoPos
}

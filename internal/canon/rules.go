package canon

import (
	"go/ast"
	"go/token"
)

// forEachStmtList calls f with a pointer to every statement list in the
// declaration: block bodies, and the case and comm clause bodies that hold
// statements without a block around them.
//
// Mutating the slice through the pointer is safe mid-walk: ast.Walk reads a
// node's children after the visitor returns, so a spliced-in statement is
// visited in the same pass rather than waiting for the next round.
func forEachStmtList(fd *ast.FuncDecl, f func(*[]ast.Stmt)) {
	ast.Inspect(fd, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BlockStmt:
			f(&x.List)
		case *ast.CaseClause:
			f(&x.Body)
		case *ast.CommClause:
			f(&x.Body)
		}
		return true
	})
}

// forEachStmtSlot calls f with a pointer to every place a single statement
// sits: each element of every statement list, plus the init/post/labelled
// slots that hold one statement outside any list. A slot holding nil is
// skipped.
func forEachStmtSlot(fd *ast.FuncDecl, f func(*ast.Stmt)) {
	visit := func(p *ast.Stmt) {
		if p != nil && *p != nil {
			f(p)
		}
	}
	ast.Inspect(fd, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BlockStmt:
			for i := range x.List {
				visit(&x.List[i])
			}
		case *ast.CaseClause:
			for i := range x.Body {
				visit(&x.Body[i])
			}
		case *ast.CommClause:
			visit(&x.Comm)
			for i := range x.Body {
				visit(&x.Body[i])
			}
		case *ast.IfStmt:
			visit(&x.Init)
		case *ast.ForStmt:
			visit(&x.Init)
			visit(&x.Post)
		case *ast.SwitchStmt:
			visit(&x.Init)
		case *ast.TypeSwitchStmt:
			// Assign is the type-switch binding, not a plain statement.
			visit(&x.Init)
		case *ast.LabeledStmt:
			visit(&x.Stmt)
		}
		return true
	})
}

// unwrapBlocks splices a bare block statement into the statement list that
// holds it. `{ a; b }` sitting inside another block is a scoping device or a
// leftover, not a shape: two functions that differ only by one of them
// having braced a section should not read as different.
//
// It applies to bare blocks of any length, not only single-statement ones —
// the brace is the incidental part, and its content length has nothing to do
// with that. Scoping is genuinely discarded, which is one more reason this
// package's output is a comparison key and not a rewrite.
//
// Measure: each firing removes one BlockStmt node, so it terminates.
func unwrapBlocks(fd *ast.FuncDecl) bool {
	changed := false
	forEachStmtList(fd, func(list *[]ast.Stmt) {
		if !hasBareBlock(*list) {
			return
		}
		out := make([]ast.Stmt, 0, len(*list))
		for _, s := range *list {
			if b, ok := s.(*ast.BlockStmt); ok {
				out = append(out, b.List...)
				changed = true
				continue
			}
			out = append(out, s)
		}
		*list = out
	})
	return changed
}

func hasBareBlock(list []ast.Stmt) bool {
	for _, s := range list {
		if _, ok := s.(*ast.BlockStmt); ok {
			return true
		}
	}
	return false
}

// unnegateIfs rewrites `if !c { A } else { B }` into `if c { B } else { A }`.
// Which branch a reader put first is a style choice; the shape is the same
// either way. This one is control-flow preserving.
//
// It fires only when the else is a plain block. An `else if` chain holds an
// IfStmt where a BlockStmt would have to go, so swapping would build an
// invalid tree, and the negation there is not incidental anyway — it orders
// a chain.
//
// It also declines when the then-branch already leaves the block and the
// else does not. That guard is not an optimization; without it the two
// branch rules fight. `if !ok(x) { return }` with an else is already in the
// early-return shape RuleGuardReturn normalizes towards, and swapping it
// moves the terminating branch into the else, where RuleGuardReturn can no
// longer lift anything out — so the same guard written two ways would
// canonicalize two ways. When *both* branches terminate the guard does not
// apply and the swap goes ahead, which strips the "!" and still leaves
// RuleGuardReturn a terminating then-branch to work with.
//
// Measure: each firing removes one leading "!" from an if condition and adds
// none, so it terminates. `if !!c {A} else {B}` unwinds in two firings.
func unnegateIfs(fd *ast.FuncDecl) bool {
	changed := false
	ast.Inspect(fd, func(n ast.Node) bool {
		ifs, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		not, ok := ifs.Cond.(*ast.UnaryExpr)
		if !ok || not.Op != token.NOT {
			return true
		}
		alt, ok := ifs.Else.(*ast.BlockStmt)
		if !ok {
			return true
		}
		if terminates(ifs.Body) && !terminates(alt) {
			return true
		}
		ifs.Cond = not.X
		ifs.Body, ifs.Else = alt, ifs.Body
		changed = true
		return true
	})
	return changed
}

// flattenGuards rewrites an if/else in which one branch always leaves the
// block into the early-return form: the terminating branch stays behind the
// condition, and the other branch is lifted out to follow the if. The two
// spellings describe the same control flow, and a function that differs from
// another only in which one it chose should not read as different.
//
//	if c { … return } else { B }   ->  if c { … return }; B
//	if c { A } else { … return }   ->  if !c { … return }; A
//
// The second direction is what makes the rule a canonicalization rather than
// a preference. Without it the two spellings settle in different places:
// RuleNegatedIf reaches the first form only when the condition was already
// negated, so a plainly-worded condition with the guard in the else would
// keep its else forever.
//
// Conditions, each of which the rule would be wrong without:
//
//   - A branch must end in a statement that leaves the block for good —
//     return, break, continue or goto. Otherwise the lifted branch would
//     run twice.
//   - Exactly one branch may terminate in the second direction. When both
//     do, the first direction applies and negating nothing is the smaller
//     change.
//   - The if must have no init statement. `if x := f(); c { return } else
//     { … x … }` puts x out of scope when the else is lifted out, and worse
//     for this tool's purposes, would let that x merge with an unrelated
//     outer x during alpha-renaming.
//   - An `else if` is spliced as the single statement it is; a plain else
//     block contributes its statements. The second direction requires a
//     plain else block, since an else-if is a chain rather than a guard.
//
// Measure: each firing removes one else branch and adds none — the negation
// the second direction introduces cannot feed RuleNegatedIf, which fires
// only on an if that still has an else. So it terminates.
func flattenGuards(fd *ast.FuncDecl) bool {
	changed := false
	forEachStmtList(fd, func(list *[]ast.Stmt) {
		for {
			i := guardIndex(*list)
			if i < 0 {
				return
			}
			ifs := (*list)[i].(*ast.IfStmt)
			var tail []ast.Stmt
			if b, ok := ifs.Else.(*ast.BlockStmt); ok {
				tail = b.List
			} else {
				tail = []ast.Stmt{ifs.Else}
			}
			if !terminates(ifs.Body) {
				// Second direction: the else is the guard. Negate the
				// condition and swap, so the guard ends up where the first
				// direction would have put it.
				alt := ifs.Else.(*ast.BlockStmt)
				ifs.Cond = negate(ifs.Cond)
				tail = ifs.Body.List
				ifs.Body = alt
			}
			ifs.Else = nil
			out := make([]ast.Stmt, 0, len(*list)+len(tail))
			out = append(out, (*list)[:i+1]...)
			out = append(out, tail...)
			out = append(out, (*list)[i+1:]...)
			*list = out
			changed = true
		}
	})
	return changed
}

// negate wraps a condition in "!", parenthesizing anything whose own
// operator binds looser than unary not so the rendered tree stays readable
// and re-parses to the same shape.
func negate(cond ast.Expr) ast.Expr {
	switch cond.(type) {
	case *ast.Ident, *ast.CallExpr, *ast.SelectorExpr, *ast.ParenExpr, *ast.IndexExpr:
		return &ast.UnaryExpr{Op: token.NOT, X: cond}
	}
	return &ast.UnaryExpr{Op: token.NOT, X: &ast.ParenExpr{X: cond}}
}

// guardIndex returns the index of the first statement in list that is an
// if/else with exactly the shape flattenGuards rewrites, or -1.
func guardIndex(list []ast.Stmt) int {
	for i, s := range list {
		ifs, ok := s.(*ast.IfStmt)
		if !ok || ifs.Else == nil || ifs.Init != nil {
			continue
		}
		if terminates(ifs.Body) {
			return i
		}
		// Second direction: only a plain else block, and only when it is
		// the branch that leaves.
		if alt, ok := ifs.Else.(*ast.BlockStmt); ok && terminates(alt) {
			return i
		}
	}
	return -1
}

// terminates reports whether a block's last statement leaves it for good.
// It is deliberately shallow — a block ending in an if/else where both
// branches return also terminates, but recognising that is dataflow, and
// the conservative answer only means the rule declines to fire.
func terminates(b *ast.BlockStmt) bool {
	if b == nil || len(b.List) == 0 {
		return false
	}
	switch last := b.List[len(b.List)-1].(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BranchStmt:
		// fallthrough is only legal as the last statement of a case clause,
		// so it never appears here; break, continue and goto all leave.
		return last.Tok != token.FALLTHROUGH
	}
	return false
}

// toIncDec rewrites `x = x + 1`, `x = 1 + x` and `x += 1` into `x++`, and
// the subtracting forms into `x--`. Both spellings are the same increment;
// only one of them should survive into the canonical tree.
//
// The target is matched on its rendered form, so `s.n = s.n + 1` folds as
// readily as `i = i + 1`. That is where the rule stops being
// semantics-preserving: if the target contains a call — `m[k()] = m[k()] +
// 1` — the fold evaluates it once where the source evaluated it twice.
// Accepted, and recorded here, for the reason the package doc gives.
//
// Both operand orders are accepted for +, so the rule is correct whichever
// side RuleCommutativeSort would later have chosen. It runs before the sort
// regardless, so in practice it sees the source order.
//
// Measure: each firing replaces an assignment with an increment, so the
// assignment count strictly decreases and it terminates.
func toIncDec(fd *ast.FuncDecl) bool {
	changed := false
	forEachStmtSlot(fd, func(p *ast.Stmt) {
		as, ok := (*p).(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return
		}
		target := as.Lhs[0]
		tok, ok := incDecTok(as, target)
		if !ok {
			return
		}
		*p = &ast.IncDecStmt{X: target, Tok: tok}
		changed = true
	})
	return changed
}

// incDecTok reports the ++/-- token an assignment folds to, if any.
func incDecTok(as *ast.AssignStmt, target ast.Expr) (token.Token, bool) {
	switch as.Tok {
	case token.ADD_ASSIGN:
		if isOne(as.Rhs[0]) {
			return token.INC, true
		}
	case token.SUB_ASSIGN:
		if isOne(as.Rhs[0]) {
			return token.DEC, true
		}
	case token.ASSIGN:
		bin, ok := as.Rhs[0].(*ast.BinaryExpr)
		if !ok {
			return token.ILLEGAL, false
		}
		key := exprKey(target)
		switch bin.Op {
		case token.ADD:
			if (exprKey(bin.X) == key && isOne(bin.Y)) || (exprKey(bin.Y) == key && isOne(bin.X)) {
				return token.INC, true
			}
		case token.SUB:
			// Subtraction does not commute: only `x - 1` is a decrement.
			if exprKey(bin.X) == key && isOne(bin.Y) {
				return token.DEC, true
			}
		}
	}
	return token.ILLEGAL, false
}

// isOne reports whether an expression is the integer literal 1.
func isOne(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "1"
}

// commutativeOps are the binary operators whose operand order carries no
// shape. Arithmetic + and *, the two equality comparisons, the three bitwise
// operators, and the two logical connectives.
//
// The logical pair and string + are the reason this package's doc says the
// canonical form is not semantics-preserving: `p != nil && p.ok` and
// `p.ok && p != nil` are the same shape and not the same program, and
// `"a" + b` is not `b + "a"`. Ordering relations (<, <=, >, >=) are not
// here: reversing them without also flipping the operator changes the
// meaning, and flipping the operator changes the shape.
var commutativeOps = map[token.Token]bool{
	token.ADD:  true,
	token.MUL:  true,
	token.EQL:  true,
	token.NEQ:  true,
	token.AND:  true,
	token.OR:   true,
	token.XOR:  true,
	token.LAND: true,
	token.LOR:  true,
}

// sortCommutative puts the operands of every commutative binary expression
// in ascending rendered order, bottom-up: a node is ordered only after its
// children are, so the keys it compares are already canonical.
//
// Measure: one bottom-up pass leaves every commutative node ordered, and a
// second pass over an unchanged tree swaps nothing, so the rule is its own
// fixed point. It can fire again in a later round only because
// RuleAlphaRename changed the names the keys are built from, which happens
// at most once.
func sortCommutative(fd *ast.FuncDecl) bool {
	c := &commuter{}
	c.block(fd.Body)
	return c.changed
}

type commuter struct{ changed bool }

// block orders every commutative expression in a block, innermost function
// literals first. The two phases matter: exprKey renders a function literal
// by printing it, so an operand containing one must have that literal
// already ordered before the operand's key is compared. Doing it in the
// other order would still converge, one extra round later.
func (c *commuter) block(b *ast.BlockStmt) {
	if b == nil {
		return
	}
	ast.Inspect(b, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		c.block(lit.Body) // recursion handles literals nested inside it
		return false
	})
	ast.Inspect(b, func(n ast.Node) bool {
		bin, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		c.order(bin)
		// order already descended through the whole operand tree, and
		// phase one already handled the function literals in it. Returning
		// true would re-key every nested binary once per enclosing level.
		return false
	})
}

// order returns an expression's key, ordering it and everything below it on
// the way back up.
func (c *commuter) order(e ast.Expr) string {
	bin, ok := e.(*ast.BinaryExpr)
	if !ok {
		return exprKey(e)
	}
	x, y := c.order(bin.X), c.order(bin.Y)
	if commutativeOps[bin.Op] && x > y {
		bin.X, bin.Y = bin.Y, bin.X
		x, y = y, x
		c.changed = true
	}
	return "(" + x + bin.Op.String() + y + ")"
}

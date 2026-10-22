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

// negate returns the logical negation of a condition in the spelling a person
// would have written it: a leading "!" is removed rather than doubled, a
// comparison flips its operator (`x == nil` becomes `x != nil`, `n > 0`
// becomes `n <= 0`), and anything else gains a "!", parenthesized when its own
// operator binds looser than unary not so the rendered tree stays readable and
// re-parses to the same shape.
//
// The first two cases are what let a negation a rule wrote meet the guard a
// person wrote. `if found { … }` lifted by RuleTailGuard must read
// `if !found { return }`, and `if x == nil { … }` must read
// `if x != nil { return }`: wrapping either in a fresh "!" would be correct and
// would canonicalize the two spellings apart, which defeats the rule. Flipping
// an ordering comparison is not exact — `!(a < b)` and `a >= b` disagree when
// either side is NaN — which is the package doc's caveat once more. && and ||
// are not pushed through by De Morgan: a person negating a conjunction writes
// it either way, so there is no one spelling to aim for.
//
// Parentheses around the condition, and around a stripped "!"'s operand, are
// dropped: they are how the condition was written, not what it is, and the
// hand-written guard does not carry them.
func negate(cond ast.Expr) ast.Expr {
	switch c := unparen(cond).(type) {
	case *ast.UnaryExpr:
		if c.Op == token.NOT {
			return unparen(c.X)
		}
	case *ast.BinaryExpr:
		if op, ok := negatedComparison[c.Op]; ok {
			return &ast.BinaryExpr{X: c.X, Op: op, Y: c.Y}
		}
	case *ast.Ident, *ast.CallExpr, *ast.SelectorExpr, *ast.IndexExpr:
		return &ast.UnaryExpr{Op: token.NOT, X: c}
	}
	return &ast.UnaryExpr{Op: token.NOT, X: &ast.ParenExpr{X: unparen(cond)}}
}

// negatedComparison maps each comparison operator to the one that holds
// exactly when it does not (NaN aside; see negate).
var negatedComparison = map[token.Token]token.Token{
	token.EQL: token.NEQ,
	token.NEQ: token.EQL,
	token.LSS: token.GEQ,
	token.GEQ: token.LSS,
	token.GTR: token.LEQ,
	token.LEQ: token.GTR,
}

// unparen strips every layer of parentheses around an expression.
func unparen(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

// liftTailIfs rewrites an if with no else that ends a block whose end is a
// bare exit into the early-return form: the condition is negated behind that
// exit, and the if's body is lifted out to follow it.
//
//	func f() { …; if c { B } }   ->  func f() { …; if !c { return }; B }
//	for … { …; if c { B } }      ->  for … { …; if !c { continue }; B }
//
// It is RuleGuardReturn's implicit-else case. Falling off the end of a
// function with no results does what a bare return does, and falling off the
// end of a loop body does what continue does, so a trailing if with no else
// already has an else — the empty rest of the block — and that else is the
// branch that leaves. RuleGuardReturn cannot see it, because it works on an
// IfStmt's Else field and there is none, so without this rule a body written
// with the guard (`if !found { return }; …`) and the same body written nested
// (`if found { … }`) never meet. The direction is RuleGuardReturn's: towards
// the early return, never into the nest.
//
// Conditions, each of which the rule would be wrong without:
//
//   - The if must be the last statement of its block. Anywhere else the
//     statements after it run whether or not c holds, and lifting B in
//     front of them behind an exit would skip them when it does not.
//   - The block must be a function body (declaration or literal) with no
//     results, or the body of a for or range loop. Those are the blocks
//     whose end is a single statement: return, which needs no operands only
//     when there are no results, and continue. A function with results
//     cannot end in an if with no else at all — the language requires a
//     terminating statement — and a bare return written into one would be a
//     statement the language rejects. Every other block either has no
//     statement that leaves just it (an if body, a bare block) or has one
//     nobody writes a guard with (break in a case clause), so there is no
//     second spelling there to converge with.
//   - The if must have no else. With one, the path where c is false is the
//     else rather than the rest of the block, so the premise is gone; an
//     if/else with a leaving branch is RuleGuardReturn's, and an else-if is
//     a chain rather than a guard.
//   - The if must have no init statement. `if x := f(); c { … x … }` puts x
//     out of scope once the body is lifted out, and the hand-written guard
//     cannot have the shape the rule would produce anyway: it has to bind x
//     before the guard, as a statement of its own.
//   - The body must not already leave. `if err != nil { return }` as the
//     last statement is the guard shape itself; lifting it would negate an
//     already-canonical guard into `if err == nil { return }; return`. It is
//     also the condition the measure below rests on. In a loop it costs one
//     equivalence, deliberately: `for … { if c { B; return x } }` has a
//     second spelling, `if !c { continue }; B; return x`, which this rule does
//     not reach. Reaching it would rewrite every trailing
//     `if err != nil { return err }` in a loop into a negated continue — a
//     spelling nobody writes — to meet one rarer than the form it replaced.
//
// The block is re-examined after each firing, so a nest of trailing ifs
// flattens into a run of guards in one pass, the last lifted body's own tail
// being next in line. Binding order is unchanged — B is moved to sit directly
// after the if, the exit binds nothing — so RuleAlphaRename stays a fixed point.
//
// Measure: each firing turns one else-less if whose body does not leave into
// one whose body does, moving the statements it lifts without rewriting them,
// so the number of else-less ifs with a non-leaving body strictly decreases.
// No other rule raises it — RuleGuardReturn's output ifs keep a leaving body,
// RuleNegatedIf works only on ifs that keep an else, and RuleUnwrapBlock can
// only turn a body ending in a bare block into one ending in what it held —
// and this rule writes no else, so it never feeds RuleGuardReturn's measure,
// nor puts a "!" where RuleNegatedIf, which needs an else, could see it. So it
// terminates.
func liftTailIfs(fd *ast.FuncDecl) bool {
	changed := false
	ast.Inspect(fd, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Body != nil && x.Type.Results.NumFields() == 0 {
				changed = liftTail(&x.Body.List, token.RETURN) || changed
			}
		case *ast.FuncLit:
			if x.Type.Results.NumFields() == 0 {
				changed = liftTail(&x.Body.List, token.RETURN) || changed
			}
		case *ast.ForStmt:
			changed = liftTail(&x.Body.List, token.CONTINUE) || changed
		case *ast.RangeStmt:
			changed = liftTail(&x.Body.List, token.CONTINUE) || changed
		}
		// Inspect walks a node's children after this returns, so the lifted
		// statements are visited in this same pass.
		return true
	})
	return changed
}

// liftTail applies liftTailIfs's rewrite to one block until its last statement
// no longer qualifies. exit is the statement the block's end stands for:
// token.RETURN or token.CONTINUE.
func liftTail(list *[]ast.Stmt, exit token.Token) bool {
	changed := false
	for {
		n := len(*list)
		if n == 0 {
			return changed
		}
		ifs, ok := (*list)[n-1].(*ast.IfStmt)
		if !ok || ifs.Else != nil || ifs.Init != nil || terminates(ifs.Body) {
			return changed
		}
		lifted := ifs.Body.List
		var leave ast.Stmt = &ast.ReturnStmt{}
		if exit == token.CONTINUE {
			leave = &ast.BranchStmt{Tok: token.CONTINUE}
		}
		ifs.Cond = negate(ifs.Cond)
		ifs.Body.List = []ast.Stmt{leave}
		out := make([]ast.Stmt, 0, n+len(lifted))
		out = append(out, (*list)...)
		*list = append(out, lifted...)
		changed = true
	}
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

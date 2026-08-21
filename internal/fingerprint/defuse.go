package fingerprint

import "go/ast"

// Crude single-hop def-use flow, as LevelFlow patterns.
//
// Nothing in the pipeline could previously distinguish a value that flows
// param → call → return from one that is computed and dropped. This pass adds
// the cheapest honest version of that dimension: role edges rendered as
// patterns (`flow:param→call:Errorf`, `flow:call:Atoi→cond`,
// `flow:call:Open→call:Close`, `flow:param→return`). Joining the pattern
// multiset buys three things at once — the shape channel's df caps and IDF
// score the edges (`flow:param→return` self-suppresses as corpus idiom, a
// rare chain is high-IDF evidence), the renders surface in the report's
// `shared structure:` block with zero reporter changes, and snapshot digests
// are untouched because Patterns are not hashed.
//
// Rename-invariant by construction: renders name *roles*, never identifiers.
// A def source is either a parameter ("param") or a binding whose right-hand
// side contains a call ("call:<name>", the first call in the expression); a
// use sink is a call the value is passed to or invoked on ("call:<name>"), a
// return ("return"), or a condition — if/for condition, switch tag, range
// operand ("cond").
//
// Deliberately NOT captured, in v1 and possibly ever (each would cost a
// resolution pass out of proportion for evidence rendering):
//
//   - shadowing: bindings are keyed by name within the function, so a
//     shadowed name merges with its shadower; the first binding's role wins
//     (parameters before body bindings, then walk order);
//   - aliasing through pointers, closures capturing variables, and struct
//     field reads/writes (x.f = v defines nothing here);
//   - multi-hop chains (a→b→c emits a→b and b→c, never a→c) and tuple
//     position (x, err := f() binds both names to call:f — which makes the
//     err-check idiom flow:call:X→cond fall out for free);
//   - control-flow sensitivity (a def in one branch reaches uses in another)
//     and anything cross-function.
func extractDefUse(fd *ast.FuncDecl, add func(level uint8, render string)) {
	if fd.Body == nil {
		return
	}

	roles := map[string]string{}
	def := func(name, role string) {
		if name == "" || name == "_" {
			return
		}
		if _, exists := roles[name]; !exists {
			roles[name] = role
		}
	}
	if fd.Type != nil && fd.Type.Params != nil {
		for _, field := range fd.Type.Params.List {
			for _, name := range field.Names {
				def(name.Name, "param")
			}
		}
	}

	// firstCall names the first call inside an expression, or "" when the
	// expression calls nothing.
	firstCall := func(e ast.Expr) string {
		label := ""
		ast.Inspect(e, func(n ast.Node) bool {
			if label != "" {
				return false
			}
			if call, ok := n.(*ast.CallExpr); ok {
				label = "call:" + callLabel(call)
				return false
			}
			return true
		})
		return label
	}
	bind := func(names []string, rhs []ast.Expr) {
		switch {
		case len(rhs) == 1:
			// One RHS for every name — including x, err := f(), which binds
			// both to call:f.
			if role := firstCall(rhs[0]); role != "" {
				for _, name := range names {
					def(name, role)
				}
			}
		case len(rhs) == len(names):
			for i, name := range names {
				if role := firstCall(rhs[i]); role != "" {
					def(name, role)
				}
			}
		}
	}

	// Pass one: bindings. Textual order is fine — within a function a name
	// cannot be used before some binding exists, and the name-keyed merge is
	// documented as crude.
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			names := make([]string, len(node.Lhs))
			for i, lhs := range node.Lhs {
				if id, ok := lhs.(*ast.Ident); ok {
					names[i] = id.Name
				}
			}
			bind(names, node.Rhs)
		case *ast.ValueSpec:
			if len(node.Values) > 0 {
				names := make([]string, len(node.Names))
				for i, id := range node.Names {
					names[i] = id.Name
				}
				bind(names, node.Values)
			}
		}
		return true
	})

	emit := func(src, sink string) { add(LevelFlow, "flow:"+src+"→"+sink) }
	useIdent := func(e ast.Expr, sink string) {
		if id, ok := e.(*ast.Ident); ok {
			if role, ok := roles[id.Name]; ok {
				emit(role, sink)
			}
		}
	}
	// condIdents finds role-carrying identifiers in a condition expression.
	// Call subtrees are skipped — their arguments are the call sink's
	// business, handled by the use pass — and only a selector's receiver is
	// considered, never its field name (a field can collide with a binding).
	condIdents := func(e ast.Expr, sink string) {
		ast.Inspect(e, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.CallExpr:
				return false
			case *ast.SelectorExpr:
				useIdent(node.X, sink)
				return false
			case *ast.Ident:
				useIdent(node, sink)
			}
			return true
		})
	}

	// Pass two: uses.
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			sink := "call:" + callLabel(node)
			for _, arg := range node.Args {
				useIdent(arg, sink)
			}
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				useIdent(sel.X, sink)
			}
		case *ast.ReturnStmt:
			for _, res := range node.Results {
				useIdent(res, "return")
			}
		case *ast.IfStmt:
			if node.Cond != nil {
				condIdents(node.Cond, "cond")
			}
		case *ast.ForStmt:
			if node.Cond != nil {
				condIdents(node.Cond, "cond")
			}
		case *ast.SwitchStmt:
			if node.Tag != nil {
				condIdents(node.Tag, "cond")
			}
		case *ast.RangeStmt:
			condIdents(node.X, "cond")
		}
		return true
	})
}

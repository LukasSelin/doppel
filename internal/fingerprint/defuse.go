package fingerprint

import "github.com/LukasSelin/doppel/internal/syntax"

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
func extractDefUse(fn *syntax.Func, add func(level uint8, render string)) {
	if fn == nil || fn.Body == nil {
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
	for _, p := range fn.Params {
		def(p.Name, "param")
	}

	// firstCall names the first call inside an expression, or "" when the
	// expression calls nothing.
	firstCall := func(e *syntax.Node) string {
		label := ""
		syntax.Inspect(e, func(n *syntax.Node) bool {
			if label != "" || n == nil {
				return false
			}
			if n.Kind == syntax.KindCall {
				label = "call:" + callLabel(n)
				return false
			}
			return true
		})
		return label
	}
	bind := func(names []string, rhs []*syntax.Node) {
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
	syntax.Inspect(fn.Body, func(n *syntax.Node) bool {
		if n == nil {
			return true
		}
		switch n.Kind {
		case syntax.KindAssign:
			lhs := n.Slots(syntax.RoleLhs)
			names := make([]string, len(lhs))
			for i, l := range lhs {
				if l.Kind == syntax.KindIdent {
					names[i] = l.Label
				}
			}
			bind(names, n.Slots(syntax.RoleRhs))
		case syntax.KindValueSpec:
			values := n.Slots(syntax.RoleValue)
			if len(values) > 0 {
				ids := n.Slots(syntax.RoleName)
				names := make([]string, len(ids))
				for i, id := range ids {
					names[i] = id.Label
				}
				bind(names, values)
			}
		}
		return true
	})

	emit := func(src, sink string) { add(LevelFlow, "flow:"+src+"→"+sink) }
	useIdent := func(e *syntax.Node, sink string) {
		if e != nil && e.Kind == syntax.KindIdent {
			if role, ok := roles[e.Label]; ok {
				emit(role, sink)
			}
		}
	}
	// condIdents finds role-carrying identifiers in a condition expression.
	// Call subtrees are skipped — their arguments are the call sink's
	// business, handled by the use pass — and only a selector's receiver is
	// considered, never its field name (a field can collide with a binding).
	condIdents := func(e *syntax.Node, sink string) {
		syntax.Inspect(e, func(n *syntax.Node) bool {
			if n == nil {
				return true
			}
			switch n.Kind {
			case syntax.KindCall:
				return false
			case syntax.KindSelector:
				useIdent(n.Slot(syntax.RoleX), sink)
				return false
			case syntax.KindIdent:
				useIdent(n, sink)
			}
			return true
		})
	}

	// Pass two: uses.
	syntax.Inspect(fn.Body, func(n *syntax.Node) bool {
		if n == nil {
			return true
		}
		switch n.Kind {
		case syntax.KindCall:
			sink := "call:" + callLabel(n)
			for _, arg := range n.Slots(syntax.RoleArg) {
				useIdent(arg, sink)
			}
			if fun := n.Slot(syntax.RoleFun); fun != nil && fun.Kind == syntax.KindSelector {
				useIdent(fun.Slot(syntax.RoleX), sink)
			}
		case syntax.KindReturn:
			for _, res := range n.Slots(syntax.RoleResult) {
				useIdent(res, "return")
			}
		case syntax.KindIf:
			if cond := n.Slot(syntax.RoleCond); cond != nil {
				condIdents(cond, "cond")
			}
		case syntax.KindFor:
			if cond := n.Slot(syntax.RoleCond); cond != nil {
				condIdents(cond, "cond")
			}
		case syntax.KindSwitch:
			if tag := n.Slot(syntax.RoleTag); tag != nil {
				condIdents(tag, "cond")
			}
		case syntax.KindRange:
			condIdents(n.Slot(syntax.RoleX), "cond")
		}
		return true
	})
}

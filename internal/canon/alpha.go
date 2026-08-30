package canon

import (
	"go/ast"
	"go/token"
	"strconv"
)

// alphaRename gives every identifier bound inside the function a positional
// name — x0, x1, … — in first-binding order, the receiver and parameters
// first. A copy with every variable renamed canonicalizes to the same tree.
//
// # What counts as a binding
//
// The receiver, parameters and named results, then, in source order through
// the body: the left-hand identifiers of a ":=", the names in a local var,
// const or type declaration, a range clause's key and value when it declares
// them, and a function literal's own parameters and results. Type-switch and
// select bindings arrive as ":=" assignments and need no separate case. The
// blank identifier is never renamed.
//
// # What is deliberately left alone
//
// Everything not bound here is free — package names, package-level
// functions and types, imported names, the universe scope — and free
// identifiers must keep their names, because they are the shared vocabulary
// two functions are compared on. Beyond that, four positions are skipped
// even when the name matches a binding:
//
//   - the Sel of a selector: x.Foo never renames Foo, only x;
//   - a bare-identifier key of a composite literal element, which is a
//     struct field name (or a constant), not a reference to a local;
//   - struct field and interface method declarations inside a local type;
//   - labels, on both the LabeledStmt and the BranchStmt that jumps to it —
//     labels live in their own namespace and renaming them into the value
//     namespace could merge two unrelated names.
//
// Struct tags are basic literals and are never touched by anything here.
//
// # Two documented crudenesses
//
// The map is keyed by name, not by declaration, so shadowing merges: an
// inner `err` and an outer `err` become the same x_i, first binding winning.
// Resolving them properly needs scope tracking that go/types does for free
// and that this module deliberately does not depend on — the same trade the
// L4 def-use pass in fingerprint documents.
//
// And a *free* identifier that happens to be literally named x0 merges with
// the first binding. Choosing an exotic prefix would trade a rare collision
// for an unreadable canonical form; the collision is recorded here instead.
func alphaRename(fd *ast.FuncDecl) bool {
	names := collectBindings(fd)
	if len(names) == 0 {
		return false
	}
	rename := make(map[string]string, len(names))
	identical := true
	for i, name := range names {
		to := "x" + strconv.Itoa(i)
		rename[name] = to
		if name != to {
			identical = false
		}
	}
	if identical {
		return false
	}
	return applyRename(fd, rename)
}

// collectBindings returns the distinct names bound inside fd, in
// first-binding order: receiver, type parameters, parameters, results, then
// the body in source order.
func collectBindings(fd *ast.FuncDecl) []string {
	var order []string
	seen := make(map[string]bool)
	add := func(id *ast.Ident) {
		if id == nil || id.Name == "" || id.Name == "_" || seen[id.Name] {
			return
		}
		seen[id.Name] = true
		order = append(order, id.Name)
	}
	addFields := func(fl *ast.FieldList) {
		if fl == nil {
			return
		}
		for _, f := range fl.List {
			for _, n := range f.Names {
				add(n)
			}
		}
	}

	addFields(fd.Recv)
	if fd.Type != nil {
		addFields(fd.Type.TypeParams)
		addFields(fd.Type.Params)
		addFields(fd.Type.Results)
	}

	// ast.Inspect is pre-order, which is source order for these constructs:
	// a range clause's key and value are recorded when the RangeStmt is
	// reached, before the walk descends into the ranged expression, which
	// is where they are written in the source.
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if x.Tok == token.DEFINE {
				for _, lhs := range x.Lhs {
					if id, ok := lhs.(*ast.Ident); ok {
						add(id)
					}
				}
			}
		case *ast.ValueSpec:
			for _, id := range x.Names {
				add(id)
			}
		case *ast.TypeSpec:
			add(x.Name)
		case *ast.RangeStmt:
			if x.Tok == token.DEFINE {
				if id, ok := x.Key.(*ast.Ident); ok {
					add(id)
				}
				if id, ok := x.Value.(*ast.Ident); ok {
					add(id)
				}
			}
		case *ast.FuncLit:
			if x.Type != nil {
				addFields(x.Type.TypeParams)
				addFields(x.Type.Params)
				addFields(x.Type.Results)
			}
		}
		return true
	})
	return order
}

// applyRename rewrites every identifier whose name is in the map, except at
// the positions skipMap collects. Each identifier node is visited once and
// its name read from the map before any write, so a rename that permutes
// names (x1 → x0 while x0 → x1) applies simultaneously rather than in two
// steps.
func applyRename(fd *ast.FuncDecl, rename map[string]string) bool {
	skip := skipSet(fd)
	changed := false
	ast.Inspect(fd, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		if skip[id] {
			return true
		}
		to, ok := rename[id.Name]
		if !ok || to == id.Name {
			return true
		}
		id.Name = to
		changed = true
		return true
	})
	return changed
}

// skipSet collects the identifier nodes alphaRename must not touch. It works
// by node identity rather than by name, which is what lets `x.x` rename the
// receiver and leave the field alone.
func skipSet(fd *ast.FuncDecl) map[*ast.Ident]bool {
	skip := make(map[*ast.Ident]bool)
	mark := func(id *ast.Ident) {
		if id != nil {
			skip[id] = true
		}
	}
	markFields := func(fl *ast.FieldList) {
		if fl == nil {
			return
		}
		for _, f := range fl.List {
			for _, n := range f.Names {
				mark(n)
			}
		}
	}
	ast.Inspect(fd, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			mark(x.Sel)
		case *ast.CompositeLit:
			for _, elt := range x.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					if id, ok := kv.Key.(*ast.Ident); ok {
						mark(id)
					}
				}
			}
		case *ast.StructType:
			markFields(x.Fields)
		case *ast.InterfaceType:
			markFields(x.Methods)
		case *ast.LabeledStmt:
			mark(x.Label)
		case *ast.BranchStmt:
			mark(x.Label)
		}
		return true
	})
	// The declaration's own name is not a binding inside it — it is the
	// function's identity, and the corpus refers to it by that name.
	mark(fd.Name)
	return skip
}

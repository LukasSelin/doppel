package canon

import (
	"go/ast"
	"strings"
)

// exprKey renders an expression to a deterministic string. It is the total
// order RuleCommutativeSort places operands in, and the equality test
// RuleIncDec uses to decide that `x` and `x` in `x = x + 1` are the same
// target.
//
// It is written out here rather than delegated to go/printer for one
// reason: a parent's key is composed from its children's keys, so a whole
// expression tree is rendered in a single bottom-up pass. Calling
// printer.Fprint on each operand of each binary node instead would re-render
// every nested subtree once per enclosing level. The exception is a function
// literal, where composing a key would mean rendering statements as well;
// those are rare inside a commutative operand, and Print handles them.
//
// The rendering is not Go source and does not need to be. It only needs to
// be deterministic, and to distinguish expressions that differ.
func exprKey(e ast.Expr) string {
	switch n := e.(type) {
	case nil:
		return ""
	case *ast.Ident:
		return n.Name
	case *ast.BasicLit:
		return n.Value
	case *ast.SelectorExpr:
		return exprKey(n.X) + "." + n.Sel.Name
	case *ast.CallExpr:
		s := exprKey(n.Fun) + "(" + exprKeys(n.Args) + ")"
		if n.Ellipsis.IsValid() {
			s += "..."
		}
		return s
	case *ast.BinaryExpr:
		return "(" + exprKey(n.X) + n.Op.String() + exprKey(n.Y) + ")"
	case *ast.UnaryExpr:
		return n.Op.String() + exprKey(n.X)
	case *ast.StarExpr:
		return "*" + exprKey(n.X)
	case *ast.ParenExpr:
		return "(" + exprKey(n.X) + ")"
	case *ast.IndexExpr:
		return exprKey(n.X) + "[" + exprKey(n.Index) + "]"
	case *ast.IndexListExpr:
		return exprKey(n.X) + "[" + exprKeys(n.Indices) + "]"
	case *ast.SliceExpr:
		s := exprKey(n.X) + "[" + exprKey(n.Low) + ":" + exprKey(n.High)
		if n.Slice3 {
			s += ":" + exprKey(n.Max)
		}
		return s + "]"
	case *ast.TypeAssertExpr:
		return exprKey(n.X) + ".(" + exprKey(n.Type) + ")"
	case *ast.CompositeLit:
		return exprKey(n.Type) + "{" + exprKeys(n.Elts) + "}"
	case *ast.KeyValueExpr:
		return exprKey(n.Key) + ":" + exprKey(n.Value)
	case *ast.Ellipsis:
		return "..." + exprKey(n.Elt)
	case *ast.ArrayType:
		return "[" + exprKey(n.Len) + "]" + exprKey(n.Elt)
	case *ast.MapType:
		return "map[" + exprKey(n.Key) + "]" + exprKey(n.Value)
	case *ast.ChanType:
		switch {
		case n.Dir == ast.SEND:
			return "chan<-" + exprKey(n.Value)
		case n.Dir == ast.RECV:
			return "<-chan" + exprKey(n.Value)
		}
		return "chan" + exprKey(n.Value)
	case *ast.StructType:
		return "struct{" + fieldKeys(n.Fields) + "}"
	case *ast.InterfaceType:
		return "interface{" + fieldKeys(n.Methods) + "}"
	case *ast.FuncType:
		return "func(" + fieldKeys(n.Params) + ")(" + fieldKeys(n.Results) + ")"
	case *ast.FuncLit:
		// Rare inside a commutative operand, and the only case where a key
		// would have to render statements. Print is exact and costs one
		// traversal; ordering two distinct literals with equal keys would
		// simply leave them unswapped, which is still deterministic.
		return "funclit" + Print(&ast.FuncDecl{Name: ast.NewIdent("_"), Type: n.Type, Body: n.Body})
	case *ast.BadExpr:
		return "?"
	}
	return "?"
}

func exprKeys(list []ast.Expr) string {
	if len(list) == 0 {
		return ""
	}
	parts := make([]string, len(list))
	for i, e := range list {
		parts[i] = exprKey(e)
	}
	return strings.Join(parts, ",")
}

func fieldKeys(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fl.List))
	for _, f := range fl.List {
		names := make([]string, 0, len(f.Names))
		for _, n := range f.Names {
			names = append(names, n.Name)
		}
		parts = append(parts, strings.Join(names, " ")+" "+exprKey(f.Type))
	}
	return strings.Join(parts, ";")
}

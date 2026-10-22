// Package gofront is doppel's Go frontend: the one place in the module that
// turns Go source into the neutral internal/syntax IR by way of go/ast.
//
// It exists as its own package so that the language-specific half is
// separable from everything that consumes it. internal/parser depends on it
// to build CodeUnits; internal/fingerprint does not depend on it at all, and
// only its tests do — which is the shape that makes "a frontend fills a
// syntax.File and gets the rest free" a fact about the build graph rather
// than an aspiration in a comment.
package gofront

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/LukasSelin/doppel/internal/canon"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// Lang is the language tag every unit from this frontend carries.
const Lang = "go"

// ruleNames projects canon's rule IDs onto the neutral strings syntax.Func
// carries. The IDs are already stable strings — canon documents them as API
// precisely because they are recorded and rendered — so this is a conversion
// and not an encoding.
func ruleNames(ids []canon.RuleID) []string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

// ParseFile reads and parses one Go file.
func ParseFile(filename string) (*syntax.File, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return Parse(filename, src)
}

// Parse turns Go source into the neutral IR. Everything after it is
// language-independent. A syntax error yields (nil, nil) — the file is warned
// about and skipped rather than failing the run.
func Parse(filename string, src []byte) (*syntax.File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		// Return partial results if the file has syntax errors
		return nil, nil
	}

	out := &syntax.File{
		Path:    filename,
		Package: f.Name.Name,
		Lang:    Lang,
		// Go's own convention (https://go.dev/s/generatedcode), checked by the
		// stdlib: a "// Code generated ... DO NOT EDIT." line before the first
		// non-comment text. Recorded per unit so --generated can pick the
		// population the same way --tests does — a convention the ecosystem
		// already declares, not a path or name heuristic.
		Generated: ast.IsGenerated(f),
		Imports:   goImports(f),
		Targets:   Targets(filename, f),
	}

	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		var docComment string
		if fd.Doc != nil {
			docComment = strings.TrimRight(fd.Doc.Text(), "\n")
		}
		fn := syntax.Func{
			Name:      fd.Name.Name,
			Receiver:  extractReceiverType(fd),
			Doc:       docComment,
			StartLine: fset.Position(fd.Pos()).Line,
			Exported:  fd.Name.IsExported(),
			Callees:   extractCallees(fd),
		}
		if fd.Type != nil {
			fn.Params = goParams(fd.Type.Params)
			fn.Results = goParams(fd.Type.Results)
			fn.Type = toSyntax(fd.Type)
		}
		if fd.Body != nil {
			fn.Body = toSyntax(fd.Body)
			// Canonicalization is Go semantics, so it happens here, in the
			// Go frontend, on go/ast — and its *output* is what crosses into
			// the neutral IR. Two orderings were possible and only this one
			// is honest: a rule like "an if with a negated condition and an
			// else is the same shape as the swapped one" is a claim about
			// what Go code means, and the IR deliberately carries no
			// semantics to make it with.
			//
			// canon works on a deep copy and never touches fd, which matters
			// exactly here: fn.Body, the token stream, the signals and the
			// callees all read the same declaration, so an in-place rewrite
			// would change every score and every concept in the tool without
			// a line of scoring code changing. internal/canon owns that
			// guarantee and proves it over every function in this repo.
			res := canon.Canonicalize(fd)
			if res.Decl != nil && res.Decl.Body != nil {
				fn.Canon = toSyntax(res.Decl.Body)
			}
			fn.CanonRules = ruleNames(res.Fired)
			fn.StartOffset = fset.Position(fd.Pos()).Offset
			fn.EndOffset = fset.Position(fd.End()).Offset
		} else {
			fn.StartOffset = fset.Position(fd.Pos()).Offset
			fn.EndOffset = fset.Position(fd.End()).Offset
		}
		fn.Source = clampSource(src, fn.StartOffset, fn.EndOffset)
		out.Funcs = append(out.Funcs, fn)
	}
	return out, nil
}

// goImports records the file's import bindings. Dot and blank imports are kept
// with their binding name so the neutral signals pass can apply the one rule
// that cares — they bind no selector-usable name — while still counting as
// imports of the file.
func goImports(f *ast.File) []syntax.Import {
	var out []syntax.Import
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		local := path.Base(p)
		if imp.Name != nil {
			local = imp.Name.Name
		}
		out = append(out, syntax.Import{Local: local, Path: p})
	}
	return out
}

// goParams renders a field list as one Param per declared name, so arity
// survives — "a, b int" is two parameters. An unnamed field is one parameter
// with no name. Type is the raw rendering: the signature line substitutes "?"
// for an unrenderable type and fingerprint.typeStrings drops it, and keeping
// the raw form here lets each apply its own rule.
func goParams(fields *ast.FieldList) []syntax.Param {
	if fields == nil {
		return nil
	}
	var out []syntax.Param
	for _, field := range fields.List {
		t := printType(field.Type)
		if len(field.Names) == 0 {
			out = append(out, syntax.Param{Type: t})
			continue
		}
		for _, n := range field.Names {
			out = append(out, syntax.Param{Name: n.Name, Type: t})
		}
	}
	return out
}

func clampSource(src []byte, start, end int) string {
	if start < 0 || start > len(src) {
		return ""
	}
	if end > len(src) {
		end = len(src)
	}
	if end < start {
		return ""
	}
	return string(src[start:end])
}

// extractReceiverType returns the printed receiver type (e.g. "*Server") or "" for functions.
func extractReceiverType(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	var buf bytes.Buffer
	printer.Fprint(&buf, token.NewFileSet(), fd.Recv.List[0].Type)
	return buf.String()
}

// printType renders a type expression with go/printer and collapses internal
// whitespace, so a multi-line struct or func type is one token. It used to
// live in fingerprint and be exported so this package could share it; now that
// fingerprint works on the neutral IR and never sees a type expression, it
// belongs here, where the go/ast dependency already is. An unprintable
// expression yields "".
//
// Parameter and result names inside a function type are dropped first (see
// unnameFuncTypes), so the text is the type and not the type as spelled: this
// one string is both what the signature component of code-shape scores and
// what the sig: line prints.
func printType(expr ast.Expr) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), unnameFuncTypes(expr)); err != nil {
		return ""
	}
	return strings.Join(strings.Fields(buf.String()), " ")
}

// unnameFuncTypes returns expr with the parameter and result names removed
// from every function type inside it, one unnamed field per declared name:
// func(f *flag.FlagSet, name string) flag.NormalizedName reads
// func(*flag.FlagSet, string) flag.NormalizedName. Go type identity ignores
// those names, and keeping them made a rename-only copy of a function with a
// func-typed parameter score 0.85 on code-shape instead of 1.00. Struct field
// names, method names in an interface and embedded types are part of the type
// and are kept.
//
// It copies the nodes on the way down and never writes to expr, which is the
// declaration every other reader of this file shares. canon.Clone would copy
// too, but it zeroes positions, and go/printer reads position validity to
// decide layout — struct{} prints as "struct { }" from a position-free tree.
// Keeping positions keeps every type with no function inside it
// byte-identical to its old rendering.
func unnameFuncTypes(expr ast.Expr) ast.Expr {
	switch n := expr.(type) {
	case *ast.FuncType:
		c := *n
		c.Params = unnamedFields(n.Params)
		c.Results = unnamedFields(n.Results)
		return &c
	case *ast.StarExpr:
		c := *n
		c.X = unnameFuncTypes(n.X)
		return &c
	case *ast.ParenExpr:
		c := *n
		c.X = unnameFuncTypes(n.X)
		return &c
	case *ast.Ellipsis:
		c := *n
		c.Elt = unnameFuncTypes(n.Elt)
		return &c
	case *ast.ArrayType:
		c := *n
		c.Elt = unnameFuncTypes(n.Elt)
		return &c
	case *ast.MapType:
		c := *n
		c.Key = unnameFuncTypes(n.Key)
		c.Value = unnameFuncTypes(n.Value)
		return &c
	case *ast.ChanType:
		c := *n
		c.Value = unnameFuncTypes(n.Value)
		return &c
	case *ast.StructType:
		c := *n
		c.Fields = namedFields(n.Fields)
		return &c
	case *ast.InterfaceType:
		c := *n
		c.Methods = namedFields(n.Methods)
		return &c
	case *ast.IndexExpr:
		c := *n
		c.Index = unnameFuncTypes(n.Index)
		return &c
	case *ast.IndexListExpr:
		c := *n
		c.Indices = make([]ast.Expr, len(n.Indices))
		for i, x := range n.Indices {
			c.Indices[i] = unnameFuncTypes(x)
		}
		return &c
	}
	return expr
}

// unnamedFields is a function type's parameter or result list with its names
// gone. "a, b int" becomes two unnamed int fields rather than one, so arity
// survives exactly as goParams keeps it for the declaration's own parameters.
func unnamedFields(fl *ast.FieldList) *ast.FieldList {
	if fl == nil {
		return nil
	}
	out := &ast.FieldList{Opening: fl.Opening, Closing: fl.Closing}
	for _, f := range fl.List {
		t := unnameFuncTypes(f.Type)
		for range max(len(f.Names), 1) {
			out.List = append(out.List, &ast.Field{Type: t})
		}
	}
	return out
}

// namedFields copies a struct's fields or an interface's methods with their
// names, tags and comments intact, rewriting only the function types inside
// their types.
func namedFields(fl *ast.FieldList) *ast.FieldList {
	if fl == nil {
		return nil
	}
	out := &ast.FieldList{Opening: fl.Opening, Closing: fl.Closing, List: make([]*ast.Field, len(fl.List))}
	for i, f := range fl.List {
		c := *f
		c.Type = unnameFuncTypes(f.Type)
		out.List[i] = &c
	}
	return out
}

// extractCallees walks the function body and returns a sorted, deduplicated list
// of function/method names found in call expressions.
func extractCallees(fd *ast.FuncDecl) []string {
	if fd.Body == nil {
		return nil
	}
	seen := make(map[string]struct{})
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			seen[fn.Name] = struct{}{}
		case *ast.SelectorExpr:
			if x, ok := fn.X.(*ast.Ident); ok {
				seen[x.Name+"."+fn.Sel.Name] = struct{}{}
			} else {
				seen[fn.Sel.Name] = struct{}{}
			}
		}
		return true
	})
	if len(seen) == 0 {
		return nil
	}
	callees := make([]string, 0, len(seen))
	for name := range seen {
		callees = append(callees, name)
	}
	sort.Strings(callees)
	return callees
}

package gofront

import (
	"bytes"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/printer"
	"go/token"
	"strings"

	"github.com/LukasSelin/doppel/internal/canon"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// CanonicalRenders re-derives one function's canonical tree from its source
// and renders every node of it as Go text, so a debug view can show what a
// Weisfeiler-Lehman label hashed.
//
// startLine is the line of the func keyword — syntax.Func.StartLine as this
// frontend records it — and name is the bare declared name (fd.Name.Name;
// parser.MethodName's result, which the caller supplies because this
// package must not import parser). Both must match, so two declarations on
// one line cannot be confused.
//
// tree is the canonical body mapped exactly as Parse maps it, and renders[i]
// pairs with the i-th non-nil node syntax.Inspect visits in tree. That
// pairing rests on the mapper being built from ast.Inspect: one syntax.Node
// per ast node, in visit order, which TestMapperPreservesOrder pins and
// TestCanonicalRendersPairWithTree re-checks here. A render is "" for a
// node go/printer cannot print on its own (a field, a field list), which
// the caller treats as "show the outline instead".
//
// # What the render is, and is not
//
// It is the canonical form: canon.Clone drops every position, so go/printer
// lays the tree out afresh, with bound identifiers alpha-renamed to x0, x1,
// … and every other rule's rewrite in place. That is the tree the bag was
// built over, so it is the honest thing to show beside a label — and it is
// not the source as written. A caller wanting the original bytes has
// CodeUnit.Body; nothing here maps a canonical node back to a span, because
// the clone deliberately keeps no span to map to.
//
// Unlike Parse, a syntax error is an error here. The file parsed when it was
// indexed, so failing to parse now means it changed underneath the run.
func CanonicalRenders(filename string, src []byte, startLine int, name string) (tree *syntax.Node, renders []string, err error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, filename, src, goparser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	var fd *ast.FuncDecl
	for _, decl := range f.Decls {
		d, ok := decl.(*ast.FuncDecl)
		if !ok || d.Name.Name != name || fset.Position(d.Pos()).Line != startLine {
			continue
		}
		fd = d
		break
	}
	if fd == nil {
		return nil, nil, fmt.Errorf("no function %s declared at %s:%d", name, filename, startLine)
	}
	if fd.Body == nil {
		return nil, nil, fmt.Errorf("%s at %s:%d has no body", name, filename, startLine)
	}
	res := canon.Canonicalize(fd)
	if res.Decl == nil || res.Decl.Body == nil {
		return nil, nil, fmt.Errorf("%s at %s:%d canonicalized to nothing", name, filename, startLine)
	}
	tree = toSyntax(res.Decl.Body)
	ast.Inspect(res.Decl.Body, func(n ast.Node) bool {
		if n != nil {
			renders = append(renders, renderNode(n))
		}
		return true
	})
	return tree, renders, nil
}

// renderNode prints one canonical node. go/printer accepts expressions,
// statements, declarations, specs and files and returns an error for
// anything else, which becomes the empty render. A statement nested under a
// label prints indented, so the common leading-tab run is stripped to keep
// every render flush left; the trailing newline goes for the same reason.
func renderNode(n ast.Node) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), n); err != nil {
		return ""
	}
	return dedent(strings.TrimRight(buf.String(), "\n"))
}

// dedent removes the longest run of leading tabs common to every non-empty
// line.
func dedent(s string) string {
	lines := strings.Split(s, "\n")
	common := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		n := len(line) - len(strings.TrimLeft(line, "\t"))
		if common < 0 || n < common {
			common = n
		}
	}
	if common <= 0 {
		return s
	}
	for i, line := range lines {
		if len(line) >= common {
			lines[i] = line[common:]
		}
	}
	return strings.Join(lines, "\n")
}

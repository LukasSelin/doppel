package parser

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"sort"
	"strings"
)

func parseGo(path string) ([]CodeUnit, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		// Return partial results if the file has syntax errors
		return nil, nil
	}

	pkg := f.Name.Name
	var units []CodeUnit
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		recvType := extractReceiverType(fd)
		name := funcName(fd, recvType)
		startLine := fset.Position(fd.Pos()).Line
		body := extractSource(fset, fd, src)
		sig := extractSignature(fset, fd)

		var docComment string
		if fd.Doc != nil {
			docComment = strings.TrimRight(fd.Doc.Text(), "\n")
		}

		units = append(units, CodeUnit{
			Name:         name,
			File:         path,
			StartLine:    startLine,
			Body:         body,
			Signature:    sig,
			Package:      pkg,
			DocComment:   docComment,
			Exported:     fd.Name.IsExported(),
			ReceiverType: recvType,
			Callees:      extractCallees(fd),
		})
	}
	return units, nil
}

// funcName returns "ReceiverType.MethodName" for methods, "FuncName" for functions.
func funcName(fd *ast.FuncDecl, recvType string) string {
	if recvType == "" {
		return fd.Name.Name
	}
	return recvType + "." + fd.Name.Name
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

// extractSignature returns "(params) (results)" for a function declaration.
func extractSignature(fset *token.FileSet, fd *ast.FuncDecl) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, fd.Type.Params)
	if fd.Type.Results != nil && len(fd.Type.Results.List) > 0 {
		buf.WriteByte(' ')
		printer.Fprint(&buf, fset, fd.Type.Results)
	}
	return buf.String()
}

// extractSource returns the source text of the node.
func extractSource(fset *token.FileSet, node ast.Node, src []byte) string {
	start := fset.Position(node.Pos()).Offset
	end := fset.Position(node.End()).Offset
	if end > len(src) {
		end = len(src)
	}
	return string(src[start:end])
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

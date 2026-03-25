package parser

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
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

		name := funcName(fd)
		startLine := fset.Position(fd.Pos()).Line
		body := extractSource(fset, fd, src)
		sig := extractSignature(fset, fd)

		units = append(units, CodeUnit{
			Name:      name,
			File:      path,
			StartLine: startLine,
			Body:      body,
			Signature: sig,
			Package:   pkg,
		})
	}
	return units, nil
}

// funcName returns "ReceiverType.MethodName" for methods, "FuncName" for functions.
func funcName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	recv := fd.Recv.List[0].Type
	var buf bytes.Buffer
	printer.Fprint(&buf, token.NewFileSet(), recv)
	return buf.String() + "." + fd.Name.Name
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

package moves

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// fileIndex is every file the type checker did not see — each graph
// package's _test.go files and the files this build leaves out — reduced to
// the names it mentions. Moving a function breaks a test that calls it and a
// build-constrained file that does; neither is in the reference graph, so
// they are found by name, which can only over-report.
type fileIndex struct {
	byPkg map[*Package][]*scanned
}

type scanned struct {
	path    string
	test    bool
	pkgName string
	imports map[string]string // import path -> the name it is bound to in this file
	plain   map[string]bool   // identifiers not on the right of a selector
	quals   map[string]bool   // "x.Name" for every selector on an identifier
}

func newFileIndex(g *Graph) *fileIndex {
	idx := &fileIndex{byPkg: map[*Package][]*scanned{}}
	fset := token.NewFileSet()
	for _, p := range g.Pkgs {
		if !p.InGraph {
			continue
		}
		var files []*scanned
		for _, path := range p.Tests {
			if f := scan(fset, path, true); f != nil {
				files = append(files, f)
			}
		}
		for _, path := range p.Ignored {
			if f := scan(fset, path, strings.HasSuffix(path, "_test.go")); f != nil {
				files = append(files, f)
			}
		}
		idx.byPkg[p] = files
	}
	return idx
}

func scan(fset *token.FileSet, path string, test bool) *scanned {
	f, err := goparser.ParseFile(fset, path, nil, goparser.SkipObjectResolution)
	if err != nil {
		return nil
	}
	s := &scanned{
		path:    path,
		test:    test,
		pkgName: f.Name.Name,
		imports: map[string]string{},
		plain:   map[string]bool{},
		quals:   map[string]bool{},
	}
	for _, imp := range f.Imports {
		p, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		name := ""
		if imp.Name != nil {
			name = imp.Name.Name
		}
		s.imports[p] = name
	}
	sels := map[*ast.Ident]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			sels[n.Sel] = true
			if x, ok := n.X.(*ast.Ident); ok {
				s.quals[x.Name+"."+n.Sel.Name] = true
			}
		case *ast.Ident:
			if !sels[n] {
				s.plain[n.Name] = true
			}
		}
		return true
	})
	return s
}

// naming returns the test files that name one of the group's symbols, and
// adds a blocker for each build-constrained file that does.
func (idx *fileIndex) naming(home *Package, gr *group, blocked []string) ([]string, []string) {
	var names []string
	for _, x := range gr.order {
		if x.Owner == nil {
			names = append(names, x.Obj.Name())
		}
	}
	var tests []string
	for _, p := range sortedPackages(idx.byPkg) {
		for _, f := range idx.byPkg[p] {
			if !f.names(home, p == home, names) {
				continue
			}
			if f.test {
				tests = append(tests, f.path)
			} else {
				blocked = append(blocked, fmt.Sprintf("named in %s, which this build does not compile", filepath.Base(f.path)))
			}
		}
	}
	return tests, blocked
}

// names reports whether f refers to one of names declared in home: plainly
// from inside home, through home's import name from anywhere else.
func (f *scanned) names(home *Package, inHome bool, names []string) bool {
	if inHome && f.pkgName == home.Name {
		for _, n := range names {
			if f.plain[n] {
				return true
			}
		}
		return false
	}
	bound, ok := f.imports[home.Path]
	if !ok {
		return false
	}
	switch bound {
	case "_":
		return false
	case ".":
		for _, n := range names {
			if f.plain[n] {
				return true
			}
		}
		return false
	case "":
		bound = home.Name
	}
	for _, n := range names {
		if f.quals[bound+"."+n] {
			return true
		}
	}
	return false
}

func sortedPackages(m map[*Package][]*scanned) []*Package {
	out := slices.Collect(maps.Keys(m))
	slices.SortFunc(out, func(a, b *Package) int { return strings.Compare(a.Path, b.Path) })
	return out
}

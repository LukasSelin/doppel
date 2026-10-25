// Package moves finds functions that live in the wrong package: ones whose
// every static reference comes from one other package, and that could move
// there together with the unexported helpers only they use.
//
// It is the one part of doppel that reads Go through the type checker rather
// than through the language-neutral IR, because the question needs exact
// references and the IR's call graph resolves by name (see
// concepter.BuildCallGraph): a missed edge there drops a merge signal, a
// missed edge here suggests moving a function out from under a caller nobody
// saw. So it is Go-only, it needs the go toolchain on PATH, and it needs a
// module that type-checks. It uses the standard library alone — go/types over
// `go list -export`, and a class hierarchy analysis written against go/types
// — rather than golang.org/x/tools, so cobra stays the only direct dependency.
package moves

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	goimporter "go/importer"
	goparser "go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Package is one type-checked package of the main module.
type Package struct {
	Path string // import path
	Name string
	Dir  string

	Files     []*ast.File
	Filenames []string // absolute, aligned with Files
	Types     *types.Package
	Info      *types.Info

	// Imports are the resolved import paths of the package's compiled files.
	Imports []string
	// Tests are the package's _test.go files and Ignored the files this build
	// does not compile (another GOOS, a build tag). Neither is type-checked, so
	// they are scanned by name only. Absolute paths.
	Tests   []string
	Ignored []string

	// InGraph is false for a package that was type-checked only because a
	// graph package imports it: outside the path analysed, or in a directory
	// the walk rule skips.
	InGraph bool

	generated map[*ast.File]bool
}

// listed is the part of a `go list -json` record the loader reads.
type listed struct {
	Dir            string
	ImportPath     string
	Name           string
	Export         string
	Standard       bool
	DepOnly        bool
	GoFiles        []string
	CgoFiles       []string
	TestGoFiles    []string
	XTestGoFiles   []string
	IgnoredGoFiles []string
	Imports        []string
	ImportMap      map[string]string
	Module         *struct{ Main bool }
	Error          *struct{ Err string }
}

const listFields = "Dir,ImportPath,Name,Export,Standard,DepOnly,GoFiles,CgoFiles,TestGoFiles," +
	"XTestGoFiles,IgnoredGoFiles,Imports,ImportMap,Module,Error"

// Load type-checks the packages under root and everything of the main module
// they import. Packages outside the main module are read from the export data
// `go list -export` leaves in the build cache, so nothing outside the module
// is type-checked from source.
//
// skip is the walk rule: a package whose directory it skips (given relative
// to root, slash-separated, and by base name) is type-checked if imported but
// kept out of the graph, so the population matches what analyze reads. Type
// errors are warned on progress and never fatal, the way parse errors are
// everywhere else in doppel; a package with errors is still graphed from what
// the checker did resolve.
func Load(ctx context.Context, root string, skip func(rel, name string) bool, progress io.Writer) (*token.FileSet, []*Package, error) {
	records, err := goList(ctx, root)
	if err != nil {
		return nil, nil, err
	}
	fset := token.NewFileSet()
	exports := map[string]string{}
	for _, r := range records {
		if r.Export != "" {
			exports[r.ImportPath] = r.Export
		}
	}
	gc := goimporter.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		file, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("no export data for %s", path)
		}
		return os.Open(file)
	})
	local := map[string]*types.Package{}

	var pkgs []*Package
	// -deps lists every package after the packages it imports, so one pass in
	// listed order type-checks each main-module package after its imports.
	for _, r := range records {
		if r.Standard || r.Module == nil || !r.Module.Main {
			continue
		}
		if r.Error != nil {
			fmt.Fprintf(progress, "warning: %s: %s\n", r.ImportPath, strings.TrimSpace(r.Error.Err))
		}
		p, err := check(fset, r, gc, local, progress)
		if err != nil {
			fmt.Fprintf(progress, "warning: %s: %v\n", r.ImportPath, err)
			continue
		}
		local[p.Path] = p.Types
		p.InGraph = !r.DepOnly && !skipped(root, r.Dir, skip)
		pkgs = append(pkgs, p)
	}
	if len(pkgs) == 0 {
		return nil, nil, fmt.Errorf("no Go packages of a main module under %s", root)
	}
	return fset, pkgs, nil
}

func goList(ctx context.Context, root string) ([]listed, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-e", "-export", "-deps", "-json="+listFields, "./...")
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list in %s: %v: %s", root, err, strings.TrimSpace(stderr.String()))
	}
	var out []listed
	dec := json.NewDecoder(&stdout)
	for {
		var r listed
		if err := dec.Decode(&r); errors.Is(err, io.EOF) {
			return out, nil
		} else if err != nil {
			return nil, fmt.Errorf("go list in %s: %w", root, err)
		}
		out = append(out, r)
	}
}

// skipped applies the walk rule to every directory between root and dir, as
// the walk would have met them on the way down.
func skipped(root, dir string, skip func(rel, name string) bool) bool {
	if skip == nil {
		return false
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return false
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i, name := range parts {
		if skip(strings.Join(parts[:i+1], "/"), name) {
			return true
		}
	}
	return false
}

func check(fset *token.FileSet, r listed, gc types.Importer, local map[string]*types.Package, progress io.Writer) (*Package, error) {
	p := &Package{
		Path:      r.ImportPath,
		Name:      r.Name,
		Dir:       r.Dir,
		generated: map[*ast.File]bool{},
	}
	for _, name := range append(append([]string{}, r.GoFiles...), r.CgoFiles...) {
		path := filepath.Join(r.Dir, name)
		f, err := goparser.ParseFile(fset, path, nil, goparser.ParseComments|goparser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		p.Files = append(p.Files, f)
		p.Filenames = append(p.Filenames, path)
		p.generated[f] = ast.IsGenerated(f)
	}
	for _, name := range append(append([]string{}, r.TestGoFiles...), r.XTestGoFiles...) {
		p.Tests = append(p.Tests, filepath.Join(r.Dir, name))
	}
	for _, name := range r.IgnoredGoFiles {
		p.Ignored = append(p.Ignored, filepath.Join(r.Dir, name))
	}
	p.Imports = r.Imports

	p.Info = &types.Info{
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	var errs []error
	conf := types.Config{
		Importer: importerFunc(func(path string) (*types.Package, error) {
			if mapped, ok := r.ImportMap[path]; ok {
				path = mapped
			}
			if t, ok := local[path]; ok {
				return t, nil
			}
			return gc.Import(path)
		}),
		FakeImportC: true,
		Error:       func(err error) { errs = append(errs, err) },
	}
	p.Types, _ = conf.Check(r.ImportPath, fset, p.Files, p.Info)
	if len(errs) > 0 {
		fmt.Fprintf(progress, "warning: %s: %d type error(s), first: %v\n", r.ImportPath, len(errs), errs[0])
	}
	return p, nil
}

type importerFunc func(path string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }

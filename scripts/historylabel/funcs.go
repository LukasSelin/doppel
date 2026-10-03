package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/scanner"
	"go/token"
	"sort"
	"strings"
)

// fn is one function body as it stood in one blob. Body is go/printer's
// rendering of the block with comments dropped, so whitespace and comment
// edits never read as a change.
type fn struct {
	Name  string // "Name" or "*Recv.Method", doppel's unit naming
	File  string // base name within its directory
	Body  string
	toks  []string
	lines []string
}

// Tokens is the body as a token stream: identifiers and literals by value,
// keywords and operators by spelling, grouping punctuation dropped (it is
// implied by the rest and only inflates every multiset equally).
func (f *fn) Tokens() []string {
	if f.toks == nil {
		f.toks = tokenize(f.Body)
	}
	return f.toks
}

// Lines is the body's statement lines, whitespace-trimmed.
func (f *fn) Lines() []string {
	if f.lines == nil {
		for _, l := range strings.Split(f.Body, "\n") {
			if l = strings.TrimSpace(l); l != "" {
				f.lines = append(f.lines, l)
			}
		}
	}
	return f.lines
}

func tokenize(src string) []string {
	var s scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	s.Init(file, []byte(src), nil, 0)
	var out []string
	for {
		_, tok, lit := s.Scan()
		switch tok {
		case token.EOF:
			return out
		case token.LBRACE, token.RBRACE, token.LPAREN, token.RPAREN,
			token.LBRACK, token.RBRACK, token.COMMA, token.SEMICOLON, token.COLON:
			continue
		}
		if lit != "" {
			out = append(out, lit)
		} else {
			out = append(out, tok.String())
		}
	}
}

// parseFuncs extracts every function with a body from one Go source file.
// A file that does not parse yields nothing: history has broken revisions,
// and a function that cannot be read at one commit simply has no state there.
func parseFuncs(file string, src []byte) []*fn {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, src, parser.SkipObjectResolution)
	if err != nil {
		return nil
	}
	var out []*fn
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		name := fd.Name.Name
		if fd.Recv != nil && len(fd.Recv.List) > 0 {
			name = recvName(fd.Recv.List[0].Type) + "." + name
		}
		var buf bytes.Buffer
		if err := cfg.Fprint(&buf, fset, fd.Body); err != nil {
			continue
		}
		out = append(out, &fn{Name: name, File: file, Body: buf.String()})
	}
	return out
}

func recvName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return "*" + recvName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return recvName(t.X)
	case *ast.IndexListExpr:
		return recvName(t.X)
	case *ast.ParenExpr:
		return recvName(t.X)
	}
	return "?"
}

// table is every function of one directory at one revision, by name. A name
// can carry several entries: build-constrained files (command_win.go beside
// command_notwin.go) each declare their own.
type table map[string][]*fn

// lookup finds a function by name, preferring the copy in file when the name
// is declared more than once.
func (t table) lookup(name, file string) *fn {
	fs := t[name]
	if len(fs) == 0 {
		return nil
	}
	for _, f := range fs {
		if f.File == file {
			return f
		}
	}
	return fs[0]
}

// names is the table's function names, sorted.
func (t table) names() []string {
	out := make([]string, 0, len(t))
	for n := range t {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// tables builds and memoizes directory tables. Trees and blobs are keyed by
// sha, so a commit that touches one file re-parses one file.
type tables struct {
	r     *repo
	blobs map[string][]*fn
	trees map[string]table
}

func newTables(r *repo) *tables {
	return &tables{r: r, blobs: map[string][]*fn{}, trees: map[string]table{}}
}

// maxCached bounds the parse caches. Nothing compares functions by pointer —
// bodies are compared as text — so dropping the caches costs only re-parsing.
const maxCached = 50000

// trim drops the caches once they outgrow maxCached. A history of a hundred
// thousand commits would otherwise hold every version of every tracked file.
func (ts *tables) trim() {
	if len(ts.blobs) > maxCached || len(ts.trees) > maxCached {
		ts.blobs = map[string][]*fn{}
		ts.trees = map[string]table{}
	}
}

// treeSHA is the sha of commit's tree at dir ("." is the root), or "" when the
// directory does not exist there.
func (ts *tables) treeSHA(commit, dir string) (string, []byte, error) {
	spec := commit + "^{tree}"
	if dir != "." {
		spec = commit + ":" + dir
	}
	sha, typ, data, ok, err := ts.r.object(spec)
	if err != nil || !ok || typ != "tree" {
		return "", nil, err
	}
	return sha, data, nil
}

// at returns the function table of dir at commit (nil when the directory is
// absent). Test files are skipped, matching the population the labels use.
func (ts *tables) at(commit, dir string) (table, string, error) {
	sha, data, err := ts.treeSHA(commit, dir)
	if err != nil || sha == "" {
		return nil, "", err
	}
	if t, ok := ts.trees[sha]; ok {
		return t, sha, nil
	}
	t := table{}
	for _, e := range parseTree(data) {
		if !strings.HasPrefix(e.mode, "100") || !strings.HasSuffix(e.name, ".go") || strings.HasSuffix(e.name, "_test.go") {
			continue
		}
		// Keyed on the name too: a renamed file is the same blob under a new
		// File, and lookup prefers by file.
		bk := e.sha + "/" + e.name
		fns, ok := ts.blobs[bk]
		if !ok {
			_, _, src, found, err := ts.r.object(e.sha)
			if err != nil {
				return nil, "", err
			}
			if found {
				fns = parseFuncs(e.name, src)
			}
			ts.blobs[bk] = fns
		}
		for _, f := range fns {
			t[f.Name] = append(t[f.Name], f)
		}
	}
	ts.trees[sha] = t
	return t, sha, nil
}

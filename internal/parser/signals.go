package parser

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
)

// TagSignals is the AST-level evidence the tagger reads instead of scanning
// raw source text. Substring-matching the body meant a comment saying "COMMIT"
// tagged a function transaction and mtx.Lock() matched the keyword "tx.";
// these fields separate the channels so each rule can look only where its
// evidence actually lives.
//
// Everything is deduplicated and sorted, so downstream consumers inherit
// deterministic ordering for free.
type TagSignals struct {
	Imports    []string // import paths of the enclosing file, unquoted
	Selectors  []string // every x.Sel expression in the body, call or not (sync.Map, sql.DB, atomic.AddInt64)
	StringLits []string // contents of string literals in the body, unquoted
	IdentNames []string // every identifier in the body, plus the function's own name
	HasGoStmt  bool     // go statement anywhere in the body
	HasSelect  bool     // select statement anywhere in the body
	HasChan    bool     // channel type in the body or the signature
}

// extractSignals walks the declaration once and collects every channel of tag
// evidence. Kept separate from extractCallees on purpose: Callees feeds the
// call graph and the shared-callee comparison signal, and its exact output —
// including its quirk of qualifying a call only when the receiver is a bare
// identifier — must not drift when tagging needs change.
func extractSignals(fd *ast.FuncDecl, file *ast.File) TagSignals {
	sig := TagSignals{}

	importSet := make(map[string]struct{})
	for _, imp := range file.Imports {
		if path, err := strconv.Unquote(imp.Path.Value); err == nil {
			importSet[path] = struct{}{}
		}
	}
	sig.Imports = sortedSet(importSet)

	selectors := make(map[string]struct{})
	literals := make(map[string]struct{})
	idents := make(map[string]struct{})

	// A function *named* retryFetch is evidence about the function even though
	// its own name never appears inside its body.
	idents[fd.Name.Name] = struct{}{}

	inspect := func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			if x, ok := node.X.(*ast.Ident); ok {
				selectors[x.Name+"."+node.Sel.Name] = struct{}{}
			}
		case *ast.BasicLit:
			if node.Kind.String() == "STRING" {
				if v, err := strconv.Unquote(node.Value); err == nil {
					literals[v] = struct{}{}
				} else {
					literals[node.Value] = struct{}{}
				}
			}
		case *ast.Ident:
			idents[node.Name] = struct{}{}
		case *ast.GoStmt:
			sig.HasGoStmt = true
		case *ast.SelectStmt:
			sig.HasSelect = true
		case *ast.ChanType:
			sig.HasChan = true
		}
		return true
	}
	if fd.Body != nil {
		ast.Inspect(fd.Body, inspect)
	}
	// A function that takes or returns a channel is coordinating concurrent
	// work even if its body never mentions one.
	if fd.Type != nil {
		ast.Inspect(fd.Type, func(n ast.Node) bool {
			if _, ok := n.(*ast.ChanType); ok {
				sig.HasChan = true
			}
			return true
		})
	}

	sig.Selectors = sortedSet(selectors)
	sig.StringLits = sortedSet(literals)
	sig.IdentNames = sortedSet(idents)
	return sig
}

func sortedSet(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// AnySelector reports whether any of the wanted selectors occurs verbatim.
func (s TagSignals) AnySelector(wanted ...string) bool {
	return anyEqual(s.Selectors, wanted)
}

// AnyReceiver reports whether any selector has one of the wanted receivers,
// matched exactly — "tx" matches tx.Commit but not mtx.Lock or ctx.Done.
func (s TagSignals) AnyReceiver(wanted ...string) bool {
	for _, sel := range s.Selectors {
		dot := strings.IndexByte(sel, '.')
		if dot < 0 {
			continue
		}
		recv := sel[:dot]
		for _, w := range wanted {
			if recv == w {
				return true
			}
		}
	}
	return false
}

// AnyMethod reports whether any selector's method part, or any bare identifier
// (which is how a plain call's name arrives), equals one of the wanted names.
func (s TagSignals) AnyMethod(wanted ...string) bool {
	for _, sel := range s.Selectors {
		if dot := strings.IndexByte(sel, '.'); dot >= 0 {
			method := sel[dot+1:]
			for _, w := range wanted {
				if method == w {
					return true
				}
			}
		}
	}
	return anyEqual(s.IdentNames, wanted)
}

// AnyImport reports whether any import path contains one of the fragments.
func (s TagSignals) AnyImport(fragments ...string) bool {
	return anyContains(s.Imports, fragments)
}

// AnyLiteral reports whether any string literal's contents contain one of the
// fragments.
func (s TagSignals) AnyLiteral(fragments ...string) bool {
	return anyContains(s.StringLits, fragments)
}

// AnyIdent reports whether any identifier name contains one of the fragments.
// Substring, not equality: the retry evidence lives in names like
// retryWithBackoff and maxRetries.
func (s TagSignals) AnyIdent(fragments ...string) bool {
	return anyContains(s.IdentNames, fragments)
}

func anyEqual(haystack, wanted []string) bool {
	for _, h := range haystack {
		for _, w := range wanted {
			if h == w {
				return true
			}
		}
	}
	return false
}

func anyContains(haystack, fragments []string) bool {
	for _, h := range haystack {
		for _, f := range fragments {
			if strings.Contains(h, f) {
				return true
			}
		}
	}
	return false
}

package parser

import (
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/syntax"
)

// TagSignals is the structural evidence the tagger reads instead of scanning
// raw source text. Substring-matching the body meant a comment saying "COMMIT"
// tagged a function transaction and mtx.Lock() matched the keyword "tx.";
// these fields separate the channels so each rule can look only where its
// evidence actually lives.
//
// Everything is deduplicated and sorted, so downstream consumers inherit
// deterministic ordering for free.
type TagSignals struct {
	Imports     []string     // import paths of the enclosing file, unquoted
	PackageRefs []PackageRef // local import name -> path, sorted by Local; see PackageRef
	Selectors   []string     // every x.Sel expression in the body, call or not (sync.Map, sql.DB, atomic.AddInt64)
	StringLits  []string     // contents of string literals in the body, unquoted
	IdentNames  []string     // every identifier in the body, plus the function's own name
	HasGoStmt   bool         // go statement anywhere in the body
	HasSelect   bool         // select statement anywhere in the body
	HasChan     bool         // channel type in the body or the signature
}

// PackageRef records one import binding of the enclosing file: the local name
// a package is reachable under, and the path it resolves to. For an unaliased
// import the local name is the path's last segment; an alias wins over that;
// dot and blank imports bind no usable name and are skipped.
//
// This is what lets a resolver ask the question the substring era could not:
// is the receiver in "x.Sel" an imported package or a local variable? A sorted
// pair slice rather than a map, keeping TagSignals' documented invariant that
// every field is deduplicated and sorted.
type PackageRef struct {
	Local string
	Path  string
}

// RefPath returns the import path bound to a local name, if any. Import lists
// are tiny; a linear scan is fine, and callers needing O(1) build their own map.
func (s TagSignals) RefPath(local string) (string, bool) {
	for _, ref := range s.PackageRefs {
		if ref.Local == local {
			return ref.Path, true
		}
	}
	return "", false
}

// extractSignals walks the declaration once and collects every channel of tag
// evidence. Kept separate from extractCallees on purpose: Callees feeds the
// call graph and the shared-callee comparison signal, and its exact output —
// including its quirk of qualifying a call only when the receiver is a bare
// identifier — must not drift when tagging needs change.
func extractSignals(fn syntax.Func, file syntax.File) TagSignals {
	sig := TagSignals{}

	importSet := make(map[string]struct{})
	refs := make(map[string]string)
	for _, imp := range file.Imports {
		importSet[imp.Path] = struct{}{}
		if imp.Local == "." || imp.Local == "_" {
			continue // dot and blank imports bind no selector-usable name
		}
		refs[imp.Local] = imp.Path
	}
	sig.Imports = sortedSet(importSet)
	if len(refs) > 0 {
		sig.PackageRefs = make([]PackageRef, 0, len(refs))
		for local, p := range refs {
			sig.PackageRefs = append(sig.PackageRefs, PackageRef{Local: local, Path: p})
		}
		sort.Slice(sig.PackageRefs, func(i, j int) bool {
			return sig.PackageRefs[i].Local < sig.PackageRefs[j].Local
		})
	}

	selectors := make(map[string]struct{})
	literals := make(map[string]struct{})
	idents := make(map[string]struct{})

	// A function *named* retryFetch is evidence about the function even though
	// its own name never appears inside its body.
	idents[fn.Name] = struct{}{}

	inspect := func(n *syntax.Node) bool {
		if n == nil {
			return true
		}
		switch n.Kind {
		case syntax.KindSelector:
			// A selector's Label is the name it selects, so both shapes read
			// the same two fields. For a nested selector like c.httpClient.Do
			// the receiver is itself a selector and its Label is the field
			// name, which records the tail pair ("httpClient.Do"): the inner
			// pair alone used to be recorded and the outer call dropped, which
			// is how a wrapper-client codebase reported zero http_call. One
			// level deep, additive, and only for the tagger — Callees and the
			// call graph never read Selectors.
			if x := n.Slot(syntax.RoleX); x != nil {
				switch x.Kind {
				case syntax.KindIdent, syntax.KindSelector:
					selectors[x.Label+"."+n.Label] = struct{}{}
				}
			}
		case syntax.KindLit:
			if n.Label == "STRING" {
				literals[n.Text] = struct{}{}
			}
		case syntax.KindIdent:
			idents[n.Label] = struct{}{}
		case syntax.KindGo:
			sig.HasGoStmt = true
		case syntax.KindSelect:
			sig.HasSelect = true
		case syntax.KindChanType:
			sig.HasChan = true
		}
		return true
	}
	syntax.Inspect(fn.Body, inspect)
	// A function that takes or returns a channel is coordinating concurrent
	// work even if its body never mentions one.
	syntax.Inspect(fn.Type, func(n *syntax.Node) bool {
		if n != nil && n.Kind == syntax.KindChanType {
			sig.HasChan = true
		}
		return true
	})

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

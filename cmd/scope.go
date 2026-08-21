package cmd

import (
	"path"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// scopeMaxPackages bounds how many mentioned packages one prompt can scope.
// A prompt that names ten packages is a survey, not a target, and the digest
// must not become the corpus dump it exists to replace.
const scopeMaxPackages = 3

// scopedPackages resolves a prompt's package mentions against a snapshot.
//
// Two mention forms match, both requiring the corpus to confirm them:
//
//   - a token containing '/' whose slash-normalized form is a path prefix of
//     some unit's file directory — "backend/internal/hubspot" style, matching
//     however deep the mention reaches;
//   - a bare token exactly equal to a package name in the corpus.
//
// Everything else in the prompt is prose. First-mention order is kept, capped
// at scopeMaxPackages, one entry per package even when mentioned twice.
func scopedPackages(prompt string, s snapshot.Snapshot) []reporter.ScopedPackage {
	if prompt == "" || len(s.Units) == 0 {
		return nil
	}

	// Directory → package, and the plain package-name set, both from the
	// snapshot so the matcher confirms mentions against what was analyzed
	// rather than against the filesystem.
	dirPkg := make(map[string]string)
	pkgs := make(map[string]bool)
	for _, u := range s.Units {
		if u.Package == "" {
			continue
		}
		pkgs[u.Package] = true
		dirPkg[path.Dir(u.File)] = u.Package
	}

	seen := make(map[string]bool)
	var out []reporter.ScopedPackage
	add := func(pkg, mention string) {
		if pkg == "" || seen[pkg] || len(out) >= scopeMaxPackages {
			return
		}
		seen[pkg] = true
		out = append(out, reporter.ScopedPackage{Package: pkg, Mention: mention})
	}

	for _, raw := range strings.Fields(prompt) {
		tok := trimMention(raw)
		if tok == "" {
			continue
		}
		if strings.ContainsAny(tok, "/\\") {
			norm := strings.Trim(path.Clean(strings.ReplaceAll(tok, "\\", "/")), "/")
			if norm == "" || norm == "." {
				continue
			}
			// The mention may name the directory from the repo root or from
			// deeper up the tree than the analysis root; match any unit
			// directory the mention is a suffix-aligned prefix of. Matches
			// are collected and sorted before adding — dirPkg is a map, and
			// map order deciding which package wins the cap would make the
			// digest differ between runs of the same prompt.
			var hits []string
			for dir, pkg := range dirPkg {
				// Four alignments: the mention names the directory exactly,
				// names it from deeper up the tree (suffix), names a parent
				// of it (prefix), or names a file inside it.
				if dir == norm || strings.HasSuffix(dir, "/"+norm) ||
					strings.HasPrefix(dir, norm+"/") || strings.HasPrefix(norm, dir+"/") {
					hits = append(hits, pkg)
				}
			}
			sort.Strings(hits)
			for _, pkg := range hits {
				add(pkg, tok)
			}
			continue
		}
		if pkgs[tok] {
			add(tok, tok)
		}
	}
	return out
}

// trimMention strips the decoration a prompt wraps around a path or name:
// a leading @ (file mentions), and trailing punctuation from prose
// ("internal/culture," or "hubspot?").
func trimMention(tok string) string {
	tok = strings.TrimPrefix(tok, "@")
	tok = strings.Trim(tok, "\"'`([{")
	return strings.TrimRight(tok, ".,;:!?)]}\"'`")
}

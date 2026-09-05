package parser

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// DefaultExcludes is the directory blocklist every walk starts from, on top of
// the go tool's own rule (dot- and underscore-prefixed names).
//
// It exists because the extension allowlist alone is not a scope rule on a
// polyglot tree. node_modules is a few thousand third-party JavaScript files
// whose extensions a frontend genuinely claims, so without this list a Node
// repository's corpus is mostly its dependencies: the learned lexicon names
// concepts after somebody else's vocabulary, IC and every df are counted over
// code nobody in the repo maintains, and the report's findings are duplication
// between two copies of a package.
//
// Two kinds of directory qualify, and both restate an argument doppel already
// makes rather than introducing a new one:
//
//   - Installed dependencies. This is "vendor" — on the list from the
//     beginning — spelled the way a dozen other ecosystems spell it. Code a
//     repository did not write and cannot merge is not a merge candidate.
//   - Build output. Near-identical by construction and unactionable, which is
//     exactly what --generated exclude says about a generated file; these are
//     the directories whose whole contents are that.
//
// It is deliberately a fixed list of names, never a content heuristic and
// never a path shape: the tagger and the retriever refuse to guess what a file
// is for, and a walk rule that guessed would be the same mistake one stage
// earlier. A name here that a particular repository uses for real source is
// what an "!name" exclude pattern is for.
//
// Names are compared case-insensitively. Directory names are case-insensitive
// on two of the three platforms doppel ships binaries for, and a walk rule
// that was not would put one tree in two different corpora depending on where
// it was analysed.
//
// Five obvious names are deliberately absent, under one rule rather than five
// judgments: a default may not shadow first-party source in a language doppel
// actually reads.
//
//   - "deps" is Elixir's install directory, and doppel has no Elixir frontend
//     — so it earns nothing, while hugo's own deps/deps.go is a Go package by
//     that name that the list would have deleted from the corpus. That one was
//     measured against the pinned ladder; the rest are the same argument.
//   - "packages" is legacy NuGet, and it is also where every pnpm, Lerna and
//     Yarn workspace keeps the repository's own code.
//   - "bin" is .NET build output, whose extensions no frontend claims anyway,
//     and it is where an npm package keeps bin/cli.js.
//   - "godeps" and "virtualenv" are dead weight: pre-modules Go is gone, and a
//     virtualenv is named venv or .venv.
//
// Each is one exclude pattern away for a repository where it does apply.
var DefaultExcludes = []string{
	// Installed dependencies.
	"bower_components", // JavaScript, legacy
	"carthage",         // Swift, Objective-C
	"jspm_packages",    // JavaScript, legacy
	"node_modules",     // JavaScript, TypeScript
	"pods",             // CocoaPods: Swift, Objective-C
	"site-packages",    // Python
	"third_party",
	"thirdparty",
	"vendor", // Go, Composer, Bundler
	"venv",   // Python; .venv is already covered by the dot rule

	// Build output.
	"build",
	"coverage",
	"deriveddata", // Xcode
	"dist",
	"obj",    // .NET, including generated sources
	"out",    // TypeScript outDir
	"target", // Rust, Maven, Gradle
	"testdata",
}

// defaultExcluded is DefaultExcludes as a lookup, lowercased once.
var defaultExcluded = func() map[string]bool {
	m := make(map[string]bool, len(DefaultExcludes))
	for _, name := range DefaultExcludes {
		m[strings.ToLower(name)] = true
	}
	return m
}()

// ShouldSkipDir reports whether a directory is outside the population under
// the default rules alone — what the go tool itself ignores, directories whose
// name starts with "." or "_" (which is what keeps _examples/ demo trees out
// of a library's population), plus everything in DefaultExcludes.
//
// It lives here, next to what it feeds, rather than in cmd: the bench harness
// walks the same tree and must apply the same rule, and it kept a byte-identical
// copy precisely to avoid depending on cmd. One definition, no such dependency.
//
// A run that configured excludes walks with an Excludes instead, which is this
// rule plus that configuration — so the defaults are still stated once.
func ShouldSkipDir(name string) bool {
	if name != "" && (name[0] == '.' || name[0] == '_') {
		return true
	}
	return defaultExcluded[strings.ToLower(name)]
}

// Excludes is the walk's directory rule for one run: the defaults above, plus
// whatever --exclude and the config's exclude key added or took back.
//
// The zero value is exactly ShouldSkipDir, so a command that configures
// nothing walks the tree it always did.
type Excludes struct {
	// patterns are kept in declaration order, which is not load-bearing: a
	// negation wins wherever it appears, so the answer does not depend on
	// where a pattern sits in the list. See SkipDir.
	patterns []excludePattern
}

type excludePattern struct {
	raw string
	// pat is lowercased, and is a glob in path.Match's sense.
	pat string
	// onPath says the pattern was written with a "/" in it and is matched
	// against the directory's root-relative slash path rather than its base
	// name. path.Match's "*" does not cross a separator, which is what makes
	// the two forms genuinely different rules rather than one loose one.
	onPath bool
	// negate says this pattern re-admits what another rule excluded.
	negate bool
}

// NewExcludes compiles a run's exclusion patterns.
//
// A pattern is a glob (path.Match syntax) over a directory's base name, or —
// when it contains a "/" — over the directory's path relative to the analysis
// root, slash-separated. A leading "!" negates: it re-admits a directory the
// defaults or another pattern would have skipped, which is the escape hatch
// for a repository whose real source lives somewhere DefaultExcludes calls
// build output.
//
// Both sides of every comparison are lowercased, matching DefaultExcludes.
//
// A malformed glob is an error rather than a pattern that silently matches
// nothing: an exclusion decides what the corpus is, and a corpus quietly
// changed by a typo is the kind of thing nobody notices until the report is
// already wrong.
func NewExcludes(patterns []string) (Excludes, error) {
	var e Excludes
	for _, raw := range patterns {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		neg := false
		if strings.HasPrefix(p, "!") {
			neg, p = true, strings.TrimSpace(p[1:])
		}
		p = strings.Trim(p, "/")
		if p == "" {
			return Excludes{}, fmt.Errorf("invalid exclude %q: names no directory", raw)
		}
		lower := strings.ToLower(p)
		if _, err := path.Match(lower, "x"); err != nil {
			return Excludes{}, fmt.Errorf("invalid exclude %q: %w", raw, err)
		}
		e.patterns = append(e.patterns, excludePattern{
			raw:    strings.TrimSpace(raw),
			pat:    lower,
			onPath: strings.Contains(lower, "/"),
			negate: neg,
		})
	}
	return e, nil
}

// SkipDir reports whether a directory is outside the population. rel is its
// path relative to the analysis root, slash-separated; name is its base name.
//
// Negation wins outright rather than by position: a "!" pattern re-admits a
// directory whatever else matched it, so the answer does not depend on the
// order patterns were written in — and the same set of patterns therefore
// means the same thing whether it arrived from a config file, a flag, or both.
//
// The root itself is the caller's business, not this rule's: doppel analyze .
// hands the walker a directory literally named ".", and a user who points
// doppel at node_modules/ has already made the call.
func (e Excludes) SkipDir(rel, name string) bool {
	lowerRel, lowerName := strings.ToLower(rel), strings.ToLower(name)
	skip := false
	for _, p := range e.patterns {
		if !p.matches(lowerRel, lowerName) {
			continue
		}
		if p.negate {
			return false
		}
		skip = true
	}
	if skip {
		return true
	}
	return ShouldSkipDir(name)
}

func (p excludePattern) matches(rel, name string) bool {
	subject := name
	if p.onPath {
		subject = rel
	}
	ok, err := path.Match(p.pat, subject)
	return err == nil && ok
}

// Patterns lists the configured patterns, sorted and deduplicated.
//
// It is what a snapshot records, and it is deliberately only the configured
// ones: the defaults are a property of the doppel build, like the walk rule
// they extend and like the frontend set behind Selection.Names(), and a
// baseline already refuses to compare across builds.
func (e Excludes) Patterns() []string {
	if len(e.patterns) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(e.patterns))
	out := make([]string, 0, len(e.patterns))
	for _, p := range e.patterns {
		if seen[p.raw] {
			continue
		}
		seen[p.raw] = true
		out = append(out, p.raw)
	}
	sort.Strings(out)
	return out
}

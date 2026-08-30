package parser

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/syntax"
)

// Frontend turns one source file into the neutral IR and answers the few
// questions about a file that only its language can answer.
//
// The interface is deliberately small. Everything doppel actually reasons
// with — the fingerprint, the evidence channels, the call graph, the learned
// vocabulary, every corpus statistic — is computed from syntax.File and knows
// nothing about any language. What must stay per-language is exactly this:
// how to parse, which extensions to claim, and which files are tests.
type Frontend interface {
	// Lang is the tag every unit from this frontend carries. It is what the
	// cross-language rule compares, so it must be stable.
	Lang() string

	// Extensions are the file extensions this frontend claims, with the
	// leading dot. Two frontends must not claim the same one.
	Extensions() []string

	// Parse turns source into the neutral IR. A file this frontend cannot
	// parse yields (nil, nil): unparseable files are skipped, never fatal.
	Parse(path string, src []byte) (*syntax.File, error)

	// IsTestFile reports whether a path is a test by this language's own
	// convention. Tests are conventionally similar by design, so they form
	// their own population — see the --tests flag.
	IsTestFile(path string) bool
}

// registry maps an extension to the frontend that claims it. It is built once
// at init and never mutated, so lookups need no synchronisation and the walk
// order of a corpus cannot change which frontend handles a file.
var registry = map[string]Frontend{}

// frontends is the registration order, which is also the order Languages()
// reports. Deterministic by construction rather than by sorting a map.
var frontends []Frontend

func register(f Frontend) {
	frontends = append(frontends, f)
	for _, ext := range f.Extensions() {
		registry[ext] = f
	}
}

// frontendFor returns the frontend claiming a path's extension, if any.
func frontendFor(path string) (Frontend, bool) {
	f, ok := registry[strings.ToLower(filepath.Ext(path))]
	return f, ok
}

// frontendLang returns the frontend registered for a language tag.
func frontendLang(lang string) (Frontend, bool) {
	for _, f := range frontends {
		if f.Lang() == lang {
			return f, true
		}
	}
	return nil, false
}

// Languages lists the registered language tags in registration order.
func Languages() []string {
	out := make([]string, 0, len(frontends))
	for _, f := range frontends {
		out = append(out, f.Lang())
	}
	return out
}

// Extensions lists every claimed extension, sorted, for help text and for the
// config file's own documentation.
func Extensions() []string {
	out := make([]string, 0, len(registry))
	for ext := range registry {
		out = append(out, ext)
	}
	sort.Strings(out)
	return out
}

// IsTestFile reports whether a path is a test file, asking the frontend that
// claims it. This is the one definition: it used to be a `_test.go` suffix
// check copy-pasted into the population filter, the null sampler, the rank
// key and the bench harness, which is both a clone doppel would flag on
// itself and four places to miss when a second language arrives.
func IsTestFile(path string) bool {
	f, ok := frontendFor(path)
	if !ok {
		return false
	}
	return f.IsTestFile(path)
}

// IsTestUnit is IsTestFile for a unit, keyed on the language that produced it
// rather than re-deriving it from the extension.
func IsTestUnit(u CodeUnit) bool {
	if f, ok := frontendLang(u.Lang); ok {
		return f.IsTestFile(u.File)
	}
	return IsTestFile(u.File)
}

// SameBuildUnit reports whether two units could sensibly be merged into one.
//
// Two rules, one predicate. Test and production code are different build
// units, so a test helper and the function it exercises are never merge
// candidates however alike they look. Two languages are the same thing again,
// one step out: a Go function and a Python one cannot be merged, their
// bodies do not compare on shape, and mixing them would make every corpus
// statistic — IC, culture, habitats, the calibration null — a two-population
// mixture describing neither.
func SameBuildUnit(a, b CodeUnit) bool {
	if a.Lang != b.Lang {
		return false
	}
	return IsTestUnit(a) == IsTestUnit(b)
}

// LangOf reports which language claims a path, and whether any does. It is
// what a corpus walk uses to apply the language allowlist before reading a
// file: deciding by extension costs nothing, where deciding by parse result
// would read every file in the tree.
func LangOf(path string) (string, bool) {
	f, ok := frontendFor(path)
	if !ok {
		return "", false
	}
	return f.Lang(), true
}

// Selection is a set of language tags, and the corpus's extension allowlist
// made explicit. The zero Selection admits every registered language, which
// is the default: doppel reads the code it can read.
//
// It exists so "code-specific" stays a rule rather than a heuristic. A file
// is in the corpus because a frontend claims its extension and the selection
// admits that language — never because something inspected the contents and
// judged them code-like.
type Selection map[string]bool

// NewSelection builds a selection from language tags. Unknown tags are
// returned so a caller can reject a config naming a language doppel has no
// frontend for, rather than silently analysing nothing.
func NewSelection(langs []string) (Selection, []string) {
	if len(langs) == 0 {
		return nil, nil
	}
	sel := Selection{}
	var unknown []string
	for _, l := range langs {
		l = strings.ToLower(strings.TrimSpace(l))
		if l == "" {
			continue
		}
		if _, ok := frontendLang(l); !ok {
			unknown = append(unknown, l)
			continue
		}
		sel[l] = true
	}
	return sel, unknown
}

// Admits reports whether a path is in the corpus.
func (s Selection) Admits(path string) bool {
	lang, ok := LangOf(path)
	if !ok {
		return false
	}
	if s == nil {
		return true
	}
	return s[lang]
}

// Names lists the selected languages, sorted. A nil selection reports every
// registered language, so what a run recorded is always the concrete list it
// actually admitted rather than an empty field meaning "whatever was built in".
func (s Selection) Names() []string {
	if s == nil {
		out := Languages()
		sort.Strings(out)
		return out
	}
	out := make([]string, 0, len(s))
	for l := range s {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

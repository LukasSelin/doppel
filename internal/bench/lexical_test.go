package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/LukasSelin/doppel/internal/gofront"
	"github.com/LukasSelin/doppel/internal/lexfront"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// TestLexicalFidelity measures what the AST buys.
//
// doppel's Go frontend and its language-agnostic one are pointed at the same
// Go corpus, and the lexical result is scored against go/ast's. This is the
// one place the cost of having no grammar becomes a number instead of a
// caveat: Go is the language where the right answer is already known, so it
// is the honest control for a frontend meant to be used on languages where
// it is not.
//
// It asserts a floor and reports everything else. The floor is recall: a
// frontend that cannot find the functions cannot find duplicates among them,
// whatever else it gets right.
//
// Guarded, because it is a measurement rather than a regression test:
//
//	DOPPEL_BENCH_LEXICAL=1 go test ./internal/bench/ -run Lexical -v
func TestLexicalFidelity(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_LEXICAL") == "" {
		t.Skip("set DOPPEL_BENCH_LEXICAL=1 to measure lexical-frontend fidelity")
	}
	root := corpusRoot(t)

	var (
		files                        int
		astFuncs, lexFuncs           int
		matched                      int
		astOnly, lexOnly             []string
		nameMismatch                 int
		bodyless                     int
		nodeRatioSum                 float64
		nodeRatioN                   int
		callsExact, callsSubset      int
		paramsMatched, paramsTotal   int
		receiverMatched, receiverTot int
	)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && parser.ShouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		af, err := gofront.Parse(path, src)
		if err != nil || af == nil {
			return nil
		}
		lf, _ := lexfront.Parse(lexfront.GoSpec, path, src)
		files++

		// Keyed by start line, not by name: a package routinely declares the
		// same method name on several types, and a name-keyed map silently
		// collapses them into one — which reads as the lexer having missed
		// functions it actually found. Start lines are exact and unique
		// within a file, so they are the honest identity here.
		astByName := map[int]syntax.Func{}
		for _, fn := range af.Funcs {
			// A declaration with no body — assembly-implemented, or an
			// external linkname — is not a comparable unit: its Fingerprint
			// is the zero value and it never matches anything. The lexical
			// frontend does not emit one at all, so counting them as misses
			// would measure a difference that changes no result. runtime is
			// where this matters: 10% of its declarations have no Go body.
			if fn.Body == nil {
				bodyless++
				continue
			}
			astByName[fn.StartLine] = fn
		}
		lexByName := map[int]syntax.Func{}
		for _, fn := range lf.Funcs {
			lexByName[fn.StartLine] = fn
		}
		astFuncs += len(astByName)
		lexFuncs += len(lf.Funcs)

		for line, a := range astByName {
			l, ok := lexByName[line]
			if !ok {
				if len(astOnly) < 25 {
					astOnly = append(astOnly, fmt.Sprintf("%s:%d:%s", relOf(root, path), line, a.Name))
				}
				continue
			}
			matched++
			if a.Name != l.Name {
				nameMismatch++
			}

			an, ln := countNodes(a.Body), countNodes(l.Body)
			if an > 0 {
				nodeRatioSum += float64(ln) / float64(an)
				nodeRatioN++
			}

			as, ls := set(a.Callees), set(l.Callees)
			if equalSets(as, ls) {
				callsExact++
			} else if subset(as, ls) || subset(ls, as) {
				callsSubset++
			}

			paramsTotal++
			if len(a.Params) == len(l.Params) {
				paramsMatched++
			}
			if a.Receiver != "" {
				receiverTot++
				if l.Receiver != "" {
					receiverMatched++
				}
			}
		}
		for line, l := range lexByName {
			if _, ok := astByName[line]; !ok && len(lexOnly) < 25 {
				lexOnly = append(lexOnly, fmt.Sprintf("%s:%d:%s", relOf(root, path), line, l.Name))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if astFuncs == 0 {
		t.Fatalf("no Go functions under %s", root)
	}

	recall := float64(matched) / float64(astFuncs)
	precision := float64(matched) / float64(lexFuncs)

	t.Logf("corpus %s: %d files, %d comparable functions by go/ast (%d bodyless declarations excluded)", root, files, astFuncs, bodyless)
	t.Logf("segmentation: %d found lexically, %d matched by name", lexFuncs, matched)
	t.Logf("  recall    %.3f   (functions go/ast found that the lexer also found)", recall)
	t.Logf("  precision %.3f   (functions the lexer found that go/ast agrees exist)", precision)
	t.Logf("names: %d of %d matched functions disagreed on the name", nameMismatch, matched)
	if nodeRatioN > 0 {
		t.Logf("body size: lexical tree is %.2fx the AST's node count on average", nodeRatioSum/float64(nodeRatioN))
	}
	t.Logf("callees: %d exact, %d one-sided subset, of %d matched", callsExact, callsSubset, matched)
	t.Logf("params: %d of %d matched on arity", paramsMatched, paramsTotal)
	t.Logf("receivers: %d of %d methods kept a receiver", receiverMatched, receiverTot)
	sort.Strings(astOnly)
	sort.Strings(lexOnly)
	if len(astOnly) > 0 {
		t.Logf("missed by the lexer (sample): %v", astOnly)
	}
	if len(lexOnly) > 0 {
		t.Logf("invented by the lexer (sample): %v", lexOnly)
	}

	// The floor. A lexical frontend that finds nine in ten functions is
	// usable on a language with no parser; one that finds half is not, and
	// no amount of good scoring downstream would rescue it.
	if recall < 0.90 {
		t.Errorf("segmentation recall %.3f is below the 0.90 floor", recall)
	}
	if precision < 0.90 {
		t.Errorf("segmentation precision %.3f is below the 0.90 floor", precision)
	}
}

// corpusRoot picks the corpus to measure: an explicitly named one, else the
// repository itself, which is always present.
func corpusRoot(t *testing.T) string {
	if c := os.Getenv("DOPPEL_BENCH_CORPUS"); c != "" {
		return c
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func relOf(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(r)
}

func countNodes(n *syntax.Node) int {
	c := 0
	syntax.Inspect(n, func(x *syntax.Node) bool {
		if x != nil {
			c++
		}
		return true
	})
	return c
}

func set(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

func equalSets(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	return subset(a, b)
}

func subset(a, b map[string]bool) bool {
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// This file is an external test package on purpose. It reuses
// parser.ShouldSkipDir — the one definition of the walk rule, which this
// repo keeps shared precisely because doppel found a hand-copied second one
// — and parser imports canon, so an internal test file could not.
package canon_test

import (
	"bytes"
	"go/ast"
	goparser "go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/LukasSelin/doppel/internal/canon"
	"github.com/LukasSelin/doppel/internal/parser"
)

// corpus is one tree of Go source to run the properties over.
type corpus struct {
	name string
	dir  string
}

// corpora returns the trees available on this machine: always this repo,
// plus the pinned cobra rung of the public ladder when it has been fetched.
// A missing corpus is skipped rather than failed — `task corpora` is
// optional and a few hundred megabytes.
func corpora(t *testing.T) []corpus {
	t.Helper()
	out := []corpus{{name: "doppel", dir: repoRoot(t)}}
	if dir := cobraDir(); dir != "" {
		out = append(out, corpus{name: "cobra", dir: dir})
	} else {
		t.Log("cobra corpus not fetched; running the properties over this repo only (run `task corpora` for the wider check)")
	}
	return out
}

// repoRoot walks up from this file to the directory holding go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate this test file")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		up := filepath.Dir(dir)
		if up == dir {
			t.Fatal("no go.mod above this test file")
		}
		dir = up
	}
}

// cobraDir returns the fetched cobra corpus, or "" when it is absent. It
// mirrors internal/bench's own root rule ($DOPPEL_CORPORA, else the user
// cache directory) without importing bench, which pulls in the whole
// pipeline.
func cobraDir() string {
	roots := []string{}
	if r := os.Getenv("DOPPEL_CORPORA"); r != "" {
		roots = append(roots, r)
	}
	if cache, err := os.UserCacheDir(); err == nil {
		roots = append(roots, filepath.Join(cache, "doppel-corpora"))
	}
	for _, r := range roots {
		dir := filepath.Join(r, "cobra")
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir
		}
	}
	return ""
}

// fn is one parsed function, kept with the fileset that positions it so the
// mutation check can print it exactly as written.
type fn struct {
	file string
	name string
	fset *token.FileSet
	decl *ast.FuncDecl
}

// collect parses every .go file under dir — test files included, since a
// canonicalizer that mishandles table-driven tests is still broken — and
// returns every function declaration with a body.
func collect(t *testing.T, dir string) []fn {
	t.Helper()
	var out []fn
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && parser.ShouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		fset := token.NewFileSet()
		f, err := goparser.ParseFile(fset, path, nil, goparser.ParseComments)
		if err != nil {
			return nil // the pipeline skips unparseable files too
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			out = append(out, fn{file: path, name: fd.Name.Name, fset: fset, decl: fd})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	if len(out) == 0 {
		t.Fatalf("no functions found under %s", dir)
	}
	return out
}

// printAt renders a node against the fileset that positions it, which is
// what makes it an exact record of the input for the mutation check.
func printAt(fset *token.FileSet, n ast.Node) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, n); err != nil {
		return ""
	}
	return buf.String()
}

// TestIdempotence is the property the whole fixed-point loop exists to have:
// canonicalizing a canonical tree changes nothing and fires nothing. It runs
// over every function in this repo and, when fetched, in cobra — tens of
// thousands of real functions rather than the handful a table test can hold.
//
// It also asserts the loop never hits its round bound. In production that is
// recorded and survived; here it is a bug, because a rule set that does not
// settle on real code is not a canonicalization.
func TestIdempotence(t *testing.T) {
	for _, c := range corpora(t) {
		t.Run(c.name, func(t *testing.T) {
			funcs := collect(t, c.dir)
			checked, everFired := 0, 0
			for _, f := range funcs {
				first := canon.Canonicalize(f.decl)
				if first.Capped {
					t.Fatalf("%s: %s hit the round bound after %d rounds", f.file, f.name, first.Rounds)
				}
				if first.Decl == nil {
					t.Fatalf("%s: %s canonicalized to nothing", f.file, f.name)
				}
				if len(first.Fired) > 0 {
					everFired++
				}

				second := canon.Canonicalize(first.Decl)
				if second.Capped {
					t.Fatalf("%s: %s hit the round bound on the second pass", f.file, f.name)
				}
				if got, want := canon.Print(second.Decl), canon.Print(first.Decl); got != want {
					t.Fatalf("%s: %s is not idempotent\nonce:\n%s\n\ntwice:\n%s", f.file, f.name, want, got)
				}
				if len(second.Fired) != 0 {
					t.Fatalf("%s: %s fired %v on the second pass; a canonical tree must be a fixed point",
						f.file, f.name, second.Fired)
				}
				checked++
			}
			t.Logf("%d functions, %d needed at least one rule", checked, everFired)
		})
	}
}

// TestOriginalIsNeverMutated is the constraint the clone exists to satisfy,
// checked at corpus scale: parser.Parse hands the same declaration to
// fingerprint.Build and to the tagger's signal extractor, so a rewrite that
// leaked into the input would move every score and every tag in the tool
// silently. Printed form before against printed form after, per function.
func TestOriginalIsNeverMutated(t *testing.T) {
	for _, c := range corpora(t) {
		t.Run(c.name, func(t *testing.T) {
			for _, f := range collect(t, c.dir) {
				before := printAt(f.fset, f.decl)
				canon.Canonicalize(f.decl)
				if after := printAt(f.fset, f.decl); after != before {
					t.Fatalf("%s: Canonicalize mutated %s\nbefore:\n%s\n\nafter:\n%s",
						f.file, f.name, before, after)
				}
			}
		})
	}
}

// TestRuleCensus reports how often each rule fires on real code. It asserts
// nothing about the individual counts, and the reason is the point of the
// package: three of the six rules normalize spellings that idiomatic,
// vetted, gofmt'd Go does not contain. Nobody writes `i = i + 1`, a bare
// block, or `if !ok { … } else { … }` in a linted repo — which is exactly
// why a canonicalizer is worth having, since the clone written in the other
// spelling is the one a text-based tool misses. Requiring every rule to
// fire here would delete the rules that matter most.
//
// The per-rule before/after tests in canon_test.go are where each rule's
// behaviour is pinned. This one exists so a change in the mix is visible in
// the log, and so the two rules that do carry the corpus — alpha-rename and
// the commutative sort — cannot silently stop firing.
func TestRuleCensus(t *testing.T) {
	counts := make(map[canon.RuleID]int)
	total := 0
	for _, c := range corpora(t) {
		for _, f := range collect(t, c.dir) {
			total++
			for _, id := range canon.Canonicalize(f.decl).Fired {
				counts[id]++
			}
		}
	}
	for _, r := range canon.Rules() {
		t.Logf("%-18s fired on %d of %d functions", r.ID, counts[r.ID], total)
	}
	// The two rules that apply to essentially all Go. A zero here is a
	// broken traversal, not a corpus that happens not to need them.
	for _, id := range []canon.RuleID{canon.RuleAlphaRename, canon.RuleCommutativeSort} {
		if counts[id] == 0 {
			t.Errorf("rule %s never fired across %d functions; the traversal is broken", id, total)
		}
	}
}

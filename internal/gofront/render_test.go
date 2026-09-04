package gofront

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/syntax"
)

// TestCanonicalRendersPairWithTree is the pairing contract: one render per
// IR node, in pre-order, and the tree is the same canonical body Parse
// produces. Fields and field lists have no standalone printed form and
// render empty; everything else renders to something.
func TestCanonicalRendersPairWithTree(t *testing.T) {
	f, err := Parse("fix.go", []byte(rich))
	if err != nil || f == nil || len(f.Funcs) != 1 {
		t.Fatalf("Parse: funcs=%d err=%v", len(f.Funcs), err)
	}
	fn := f.Funcs[0]
	tree, renders, err := CanonicalRenders("fix.go", []byte(rich), fn.StartLine, fn.Name)
	if err != nil {
		t.Fatalf("CanonicalRenders: %v", err)
	}
	if got, want := irNodeCount(tree), irNodeCount(fn.Canon); got != want {
		t.Fatalf("re-derived tree has %d nodes, Parse's canonical body has %d", got, want)
	}
	if len(renders) != irNodeCount(tree) {
		t.Fatalf("%d renders for %d nodes", len(renders), irNodeCount(tree))
	}
	i := 0
	syntax.Inspect(tree, func(n *syntax.Node) bool {
		if n == nil {
			return false
		}
		r := renders[i]
		switch n.Kind {
		case syntax.KindField, syntax.KindFieldList:
			if r != "" {
				t.Errorf("node %d (%v) rendered %q; a field has no standalone form", i, n.Kind, r)
			}
		default:
			if r == "" {
				t.Errorf("node %d (%v) rendered empty", i, n.Kind)
			}
		}
		i++
		return true
	})
	if strings.Contains(renders[0], "\n\t") && !strings.HasPrefix(renders[0], "{") {
		t.Errorf("root render is not flush left:\n%s", renders[0])
	}
}

// TestCanonicalRendersIsCanonical: the render is the canonical form — alpha
// renamed and with the negated-if rule applied — not the source as written.
func TestCanonicalRendersIsCanonical(t *testing.T) {
	src := `package p

func F(ok bool) int {
	count := 1
	if !ok {
		count = 2
	} else {
		count = 3
	}
	return count
}
`
	tree, renders, err := CanonicalRenders("f.go", []byte(src), 3, "F")
	if err != nil {
		t.Fatalf("CanonicalRenders: %v", err)
	}
	if tree == nil || len(renders) == 0 {
		t.Fatal("empty result")
	}
	body := renders[0]
	if strings.Contains(body, "count") || strings.Contains(body, "!") {
		t.Errorf("render is not canonical (expected x0-style names and the guard flipped):\n%s", body)
	}
	if !strings.Contains(body, "if x") {
		t.Errorf("render lost the if:\n%s", body)
	}
}

func TestCanonicalRendersErrors(t *testing.T) {
	src := "package p\n\nfunc A() {}\nfunc B()\n"
	if _, _, err := CanonicalRenders("f.go", []byte(src), 3, "Nope"); err == nil || !strings.Contains(err.Error(), "no function Nope") {
		t.Errorf("wrong name: %v", err)
	}
	if _, _, err := CanonicalRenders("f.go", []byte(src), 99, "A"); err == nil || !strings.Contains(err.Error(), "no function A") {
		t.Errorf("wrong line: %v", err)
	}
	if _, _, err := CanonicalRenders("f.go", []byte(src), 4, "B"); err == nil || !strings.Contains(err.Error(), "no body") {
		t.Errorf("bodyless: %v", err)
	}
	if _, _, err := CanonicalRenders("f.go", []byte("package p\nfunc broken( {{{"), 2, "broken"); err == nil {
		t.Error("a syntax error must be an error here, unlike Parse")
	}
	if _, _, err := CanonicalRenders("f.go", []byte(src), 3, "A"); err != nil {
		t.Errorf("A at line 3: %v", err)
	}
}

func TestDedent(t *testing.T) {
	if got := dedent("\t\ta\n\t\t\tb\n\n\t\tc"); got != "a\n\tb\n\nc" {
		t.Errorf("dedent = %q", got)
	}
	if got := dedent("a\n\tb"); got != "a\n\tb" {
		t.Errorf("no common indent must be untouched, got %q", got)
	}
}

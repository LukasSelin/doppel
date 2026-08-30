package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/canon"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// These tests sit at the seam where canonicalization becomes visible to
// everything downstream: CodeUnit.Canonical and CodeUnit.CanonRules.
//
// They read the neutral tree rather than go/ast, deliberately. Canonicalization
// itself is Go semantics and runs inside gofront, where internal/canon's own
// tests prove the rewrites; what a *consumer* of a CodeUnit can see is a
// syntax.Node, and that is what the WL bag, the hash-cons and the explanation
// layer all walk. Asserting on the go/ast form here would test a tree nobody
// downstream is handed.

const canonSrc = `package p

import "fmt"

// Add sums two numbers the long way round.
func Add(left, right int) int {
	total := 0
	total = total + left
	calls = calls + 1
	if !ok(right) {
		return total
	} else {
		total = total + right
	}
	return total
}

func Bare() {}

type T struct{}

func (t *T) Method(n int) string {
	return fmt.Sprintf("%d", n)
}
`

// TestParseAttachesCanonicalForm: every unit with a body carries a canonical
// tree and the list of rules that produced it.
func TestParseAttachesCanonicalForm(t *testing.T) {
	units, err := ParseSource("p.go", []byte(canonSrc))
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 3 {
		t.Fatalf("expected 3 units, got %d", len(units))
	}
	for _, u := range units {
		if u.Canonical == nil {
			t.Errorf("%s: no canonical tree", u.Name)
			continue
		}
		if countNodes(u.Canonical) == 0 {
			t.Errorf("%s: canonical tree has no nodes", u.Name)
		}
	}

	add := units[0]
	if add.Name != "Add" {
		t.Fatalf("expected Add first, got %s", add.Name)
	}
	// The normalizations this body needs, and no claim about others.
	// RuleNegatedIf deliberately does *not* fire: the guard is already in
	// the then-branch, which is the shape RuleGuardReturn normalizes to.
	for _, want := range []canon.RuleID{canon.RuleAlphaRename, canon.RuleGuardReturn, canon.RuleIncDec} {
		found := false
		for _, id := range add.CanonRules {
			if id == string(want) {
				found = true
			}
		}
		if !found {
			t.Errorf("expected %s to fire on Add; fired %v", want, add.CanonRules)
		}
	}
	for _, gone := range []string{"left", "right"} {
		if hasIdent(add.Canonical, gone) {
			t.Errorf("parameter %q was not renamed in the canonical tree:\n%s", gone, render(add.Canonical))
		}
	}

	// Bare has a body, an empty one: a canonical tree with nothing to do,
	// not a missing one.
	bare := units[1]
	if bare.Canonical == nil {
		t.Error("Bare should still carry a canonical tree")
	}
	if len(bare.CanonRules) != 0 {
		t.Errorf("Bare needs no rules, fired %v", bare.CanonRules)
	}
}

// TestCanonicalIsSeparateFromTheScoredTree is the whole reason canon clones.
// The fields the pipeline already scores — the token stream, Signals, Body,
// Signature, Callees — must be exactly what they were before a canonical tree
// existed. Comparing two parses of the same source cannot show that, so this
// checks the property that would break: the canonical tree is genuinely
// rewritten while the recorded body text still reads as written.
func TestCanonicalIsSeparateFromTheScoredTree(t *testing.T) {
	units, err := ParseSource("p.go", []byte(canonSrc))
	if err != nil {
		t.Fatal(err)
	}
	add := units[0]

	// The source text kept for the report is untouched.
	for _, want := range []string{"left", "right", "total", "if !ok(right)", "else"} {
		if !strings.Contains(add.Body, want) {
			t.Errorf("Body no longer contains %q; canonicalization leaked into the recorded source", want)
		}
	}
	// The signature is rendered from the original declaration.
	if add.Signature != "(int, int) (int)" {
		t.Errorf("Signature = %q, want %q", add.Signature, "(int, int) (int)")
	}
	// The callees the call graph resolves on come from the original names.
	joined := strings.Join(add.Callees, ",")
	if !strings.Contains(joined, "ok") {
		t.Errorf("Callees = %v, expected the original call name", add.Callees)
	}
	// And the canonical tree really was rewritten, so the checks above are
	// not passing because nothing happened.
	if len(add.CanonRules) == 0 {
		t.Fatal("no rules fired, so this test proves nothing")
	}
	if hasElse(add.Canonical) {
		t.Errorf("canonical tree still has an else:\n%s", render(add.Canonical))
	}
}

// TestCanonicalTreesMatchForRenamedClones is the property the canonical form
// exists for, checked at the seam where a consumer sees it: two functions that
// differ only in incidental spelling canonicalize to the same tree.
//
// The declaration's own name never arises, which is why no name has to be set
// aside here the way it did when the canonical form was a whole FuncDecl:
// Canonical is the *body*, and a function's identity in its package is not a
// binding inside it.
func TestCanonicalTreesMatchForRenamedClones(t *testing.T) {
	src := `package p

func first(items []string) int {
	count := 0
	for _, item := range items {
		if !valid(item) {
			continue
		}
		count = count + 1
	}
	return count
}

func second(entries []string) int {
	n := 0
	for _, entry := range entries {
		if !valid(entry) {
			continue
		} else {
			n += 1
		}
	}
	return n
}
`
	units, err := ParseSource("p.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}
	a, b := render(units[0].Canonical), render(units[1].Canonical)
	if a != b {
		t.Errorf("renamed-and-respelled clones did not converge\nfirst:\n%s\n\nsecond:\n%s", a, b)
	}
}

// render is a total, deterministic serialization of a syntax tree: every
// node's kind, its label and its children's slots, indented. It is a test
// helper and nothing else — the production code hashes trees rather than
// printing them — but it makes a failure readable, which a hash cannot.
func render(n *syntax.Node) string {
	var b strings.Builder
	renderInto(&b, n, 0)
	return b.String()
}

func renderInto(b *strings.Builder, n *syntax.Node, depth int) {
	if n == nil {
		return
	}
	fmt.Fprintf(b, "%s%d", strings.Repeat("  ", depth), n.Kind)
	if n.Label != "" {
		fmt.Fprintf(b, "(%s)", n.Label)
	}
	b.WriteByte('\n')
	for _, k := range n.Kids {
		fmt.Fprintf(b, "%s#%d\n", strings.Repeat("  ", depth+1), k.Role)
		renderInto(b, k.Node, depth+2)
	}
}

func countNodes(n *syntax.Node) int {
	count := 0
	syntax.Inspect(n, func(x *syntax.Node) bool {
		if x != nil {
			count++
		}
		return true
	})
	return count
}

func hasIdent(n *syntax.Node, name string) bool {
	found := false
	syntax.Inspect(n, func(x *syntax.Node) bool {
		if x != nil && x.Kind == syntax.KindIdent && x.Label == name {
			found = true
		}
		return true
	})
	return found
}

func hasElse(n *syntax.Node) bool {
	found := false
	syntax.Inspect(n, func(x *syntax.Node) bool {
		if x != nil && x.Kind == syntax.KindIf && x.Slot(syntax.RoleElse) != nil {
			found = true
		}
		return true
	})
	return found
}

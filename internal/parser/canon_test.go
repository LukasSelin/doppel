package parser

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/canon"
)

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
		if got := canon.Print(u.Canonical); got == "" {
			t.Errorf("%s: canonical tree does not print", u.Name)
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
			if id == want {
				found = true
			}
		}
		if !found {
			t.Errorf("expected %s to fire on Add; fired %v", want, add.CanonRules)
		}
	}
	if printed := canon.Print(add.Canonical); strings.Contains(printed, "left") || strings.Contains(printed, "right") {
		t.Errorf("parameters were not renamed in the canonical tree:\n%s", printed)
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
// The fields the pipeline already scores — Fingerprint, Signals, Body,
// Signature, Callees — must be exactly what they were before a canonical
// tree existed. Comparing two parses of the same source cannot show that,
// so this checks the property that would break: the canonical tree is a
// different object, and it is genuinely rewritten while the recorded body
// text still reads as written.
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
	if strings.Contains(canon.Print(add.Canonical), "else") {
		t.Errorf("canonical tree still has an else:\n%s", canon.Print(add.Canonical))
	}
}

// TestCanonicalTreesMatchForRenamedClones is the property the canonical form
// is being built for, checked at the seam where it is now produced: two
// functions that differ only in incidental spelling must canonicalize to the
// same tree, while the fingerprints — which nothing in this task rewired —
// stay whatever they already were.
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
	a, b := shapeOf(units[0].Canonical), shapeOf(units[1].Canonical)
	if a != b {
		t.Errorf("renamed-and-respelled clones did not converge\nfirst:\n%s\n\nsecond:\n%s", a, b)
	}
}

// shapeOf prints a canonical tree with its own name replaced. The
// declaration name is not renamed by canonicalization and must not be: it is
// the function's identity in the package, not a binding inside it, and the
// call graph and every report refer to units by it. A consumer comparing two
// canonical trees as shapes has to set it aside, which is what this does.
func shapeOf(fd *ast.FuncDecl) string {
	if fd == nil {
		return ""
	}
	return canon.Print(&ast.FuncDecl{
		Recv: fd.Recv,
		Name: ast.NewIdent("_"),
		Type: fd.Type,
		Body: fd.Body,
	})
}

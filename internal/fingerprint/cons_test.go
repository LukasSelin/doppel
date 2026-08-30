package fingerprint

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/syntax"
)

func consOf(t *testing.T, src string) []uint64 {
	t.Helper()
	return Cons(parseFunc(t, src))
}

// TestConsNoBody: a declaration without a body has no subtrees, mirroring
// WLBag's rule.
func TestConsNoBody(t *testing.T) {
	if got := Cons(nil); got != nil {
		t.Errorf("nil declaration: got %v, want nil", got)
	}
	if got := consOf(t, `func f()`); got != nil {
		t.Errorf("body-less declaration: got %v, want nil", got)
	}
}

// TestConsDeterministic: the same source always produces the same hashes in
// the same order, since Cons walks in one fixed post-order and never sorts.
func TestConsDeterministic(t *testing.T) {
	first := consOf(t, srcServe)
	for i := 0; i < 20; i++ {
		got := consOf(t, srcServe)
		if len(got) != len(first) {
			t.Fatalf("run %d: %d hashes, want %d", i, len(got), len(first))
		}
		for j := range got {
			if got[j] != first[j] {
				t.Fatalf("run %d differed at node %d", i, j)
			}
		}
	}
}

// TestConsIdenticalSubtreesCollapse is the whole point of hash-consing: two
// occurrences of the same statement, written identically, must hash to the
// same value so a corpus-wide dedup counts them once.
func TestConsIdenticalSubtreesCollapse(t *testing.T) {
	hashes := consOf(t, `
func f(a, b bool) {
	if a {
		g()
	}
	if a {
		g()
	}
}`)
	seen := map[uint64]int{}
	for _, h := range hashes {
		seen[h]++
	}
	dup := 0
	for _, n := range seen {
		if n > 1 {
			dup++
		}
	}
	if dup == 0 {
		t.Fatal("two identical `if a { g() }` statements produced no repeated hash")
	}
}

// TestConsOrderMatters is what separates a hash-cons from WL's shape channel:
// WL sorts a node's children into a multiset, so `a - b` and `b - a` are the
// same shape. A hash-cons answers "is this literally the same subtree", so
// swapping the operands of a non-commutative expression must change the hash
// of that subtree (and everything above it) even though the two functions
// are the same size and use the same tokens.
//
// Both operands have to be structurally distinguishable for this to test
// anything: a bare identifier's label carries no name (wlLabel0 collapses
// every *ast.Ident to plain "ID"), so `x - y` and `y - x` would hash
// identically under either convention. `x - g()` vs `g() - x` keeps the two
// operand kinds (ID, CALL) apart, so only their order is left to matter.
func TestConsOrderMatters(t *testing.T) {
	a := consOf(t, `func f(x int) int { return x - g() }`)
	b := consOf(t, `func f(x int) int { return g() - x }`)
	if len(a) != len(b) {
		t.Fatalf("different node counts: %d vs %d", len(a), len(b))
	}
	// The root (ReturnStmt) hash must differ: it is the last hash produced,
	// since Cons appends bottom-up and post-order visits the root last.
	if a[len(a)-1] == b[len(b)-1] {
		t.Error("`x - g()` and `g() - x` hash-consed to the same subtree")
	}
}

// TestConsCorpusRatio ties Cons to the corpus aggregate: a body that is just
// two copies of the same statement compresses (ratio > 1), an empty corpus
// does not divide by zero, and a corpus of nothing but distinct one-node
// bodies still compresses on their shared leaf kinds (every bare identifier
// is the same ID leaf).
func TestConsCorpusRatio(t *testing.T) {
	repeated := parseFunc(t, `
func f(a, b bool) {
	if a {
		g()
	}
	if a {
		g()
	}
}`)
	stats := ConsCorpus([]*syntax.Node{repeated})
	if stats.TotalNodes == 0 {
		t.Fatal("expected some nodes")
	}
	if stats.UniqueSubtrees >= stats.TotalNodes {
		t.Errorf("unique subtrees %d, total nodes %d: expected the repeated `if a { g() }` to compress",
			stats.UniqueSubtrees, stats.TotalNodes)
	}
	if ratio := stats.Ratio(); ratio <= 1.0 {
		t.Errorf("Ratio() = %v, want > 1.0", ratio)
	}

	empty := ConsCorpus(nil)
	if empty.TotalNodes != 0 || empty.UniqueSubtrees != 0 {
		t.Errorf("empty corpus: got %+v, want zero", empty)
	}
	if got := empty.Ratio(); got != 0 {
		t.Errorf("Ratio() of an empty corpus = %v, want 0 (not a division by zero)", got)
	}

	// A nil body among real ones contributes nothing, mirroring Cons's own
	// nil rule, and does not panic.
	mixed := ConsCorpus([]*syntax.Node{repeated, nil})
	if mixed != stats {
		t.Errorf("a nil body changed the corpus totals: %+v vs %+v", mixed, stats)
	}
}

// TestConsCrossFunctionCollapse: two different functions whose bodies are
// identical after parsing (nothing for canon to normalize here) must share
// every subtree hash, which is the corpus-wide half of hash-consing — it
// dedups across functions, not just within one.
//
// The two functions' own hash sets are compared against each other rather
// than against len(ha), because a repeated leaf (both statements below touch
// the same identifier, which collapses to the unnamed "ID" label) already
// dedups *within* one function — that is Cons doing its job, not a test bug.
func TestConsCrossFunctionCollapse(t *testing.T) {
	a := parseFunc(t, `func f() { total := 0; total++ }`)
	b := parseFunc(t, `func g() { total := 0; total++ }`)
	ha, hb := Cons(a), Cons(b)
	if len(ha) != len(hb) {
		t.Fatalf("identical bodies produced different node counts: %d vs %d", len(ha), len(hb))
	}
	withinA := uniqueCount(ha)
	stats := ConsCorpus([]*syntax.Node{a, b})
	if stats.UniqueSubtrees != withinA {
		t.Errorf("UniqueSubtrees = %d, want %d (b should add no new shape over a)",
			stats.UniqueSubtrees, withinA)
	}
	if stats.TotalNodes != len(ha)+len(hb) {
		t.Errorf("TotalNodes = %d, want %d", stats.TotalNodes, len(ha)+len(hb))
	}
}

func uniqueCount(hashes []uint64) int {
	seen := map[uint64]struct{}{}
	for _, h := range hashes {
		seen[h] = struct{}{}
	}
	return len(seen)
}

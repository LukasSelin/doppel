package syntax

import (
	"reflect"
	"testing"
)

func leaf(k Kind, label string) *Node { return &Node{Kind: k, Label: label} }

// TestInspectEmitsAfterChildrenHook pins the one property fingerprint.walk
// depends on for its nesting depth: every node f returns true for is followed
// by exactly one nil call, after its children. Without it the depth histogram
// never pops and every node reads as more deeply nested than it is.
func TestInspectEmitsAfterChildrenHook(t *testing.T) {
	root := &Node{Kind: KindBlock}
	root.Add(RoleList, leaf(KindIdent, "a"))
	root.Add(RoleList, leaf(KindIdent, "b"))

	var seq []string
	Inspect(root, func(n *Node) bool {
		if n == nil {
			seq = append(seq, "nil")
			return true
		}
		seq = append(seq, n.Label+"/"+string(rune('0'+int(n.Kind))))
		return true
	})
	// block, a, nil(a), b, nil(b), nil(block)
	if len(seq) != 6 || seq[len(seq)-1] != "nil" {
		t.Fatalf("unexpected visit sequence: %v", seq)
	}
	nils := 0
	for _, s := range seq {
		if s == "nil" {
			nils++
		}
	}
	if nils != 3 {
		t.Errorf("want one nil per visited node (3), got %d in %v", nils, seq)
	}
}

// TestInspectSkipsSubtreeOnFalse pins the other half of the contract: a false
// return prunes the subtree and emits no nil for that node, which is what lets
// def-use skip call subtrees without unbalancing anything.
func TestInspectSkipsSubtreeOnFalse(t *testing.T) {
	inner := &Node{Kind: KindCall}
	inner.Add(RoleArg, leaf(KindIdent, "hidden"))
	root := &Node{Kind: KindBlock}
	root.Add(RoleList, inner)

	var seen []string
	Inspect(root, func(n *Node) bool {
		if n == nil {
			return true
		}
		seen = append(seen, n.Label)
		return n.Kind != KindCall
	})
	for _, s := range seen {
		if s == "hidden" {
			t.Fatal("pruned subtree was still visited")
		}
	}
}

func TestSlotAndSlots(t *testing.T) {
	n := &Node{Kind: KindCall}
	n.Add(RoleFun, leaf(KindIdent, "f"))
	n.Add(RoleArg, leaf(KindIdent, "x"))
	n.Add(RoleArg, leaf(KindIdent, "y"))

	if got := n.Slot(RoleFun); got == nil || got.Label != "f" {
		t.Errorf("Slot(RoleFun) = %v", got)
	}
	if got := n.Slot(RoleCond); got != nil {
		t.Errorf("absent slot should be nil, got %v", got)
	}
	var args []string
	for _, a := range n.Slots(RoleArg) {
		args = append(args, a.Label)
	}
	if !reflect.DeepEqual(args, []string{"x", "y"}) {
		t.Errorf("Slots(RoleArg) = %v, want [x y]", args)
	}
	// Add tolerates a nil child so frontends need not guard optional slots.
	before := len(n.Kids)
	n.Add(RoleCond, nil)
	if len(n.Kids) != before {
		t.Error("Add(nil) should be a no-op")
	}
}

// TestNilReceiversAreSafe: consumers call Slot on the result of Slot, so the
// nil case has to be total rather than a panic waiting for an odd corpus.
func TestNilReceiversAreSafe(t *testing.T) {
	var n *Node
	if n.Slot(RoleX) != nil || n.Slots(RoleX) != nil || n.Children() != nil {
		t.Error("nil node accessors should return nil")
	}
	Inspect(nil, func(*Node) bool { t.Fatal("f called for nil root"); return true })
}

// TestIsStmt guards the gate extractPatterns' default branch reads: an
// unclassified node must not be treated as a statement, or it renders
// something a frontend never claimed.
func TestIsStmt(t *testing.T) {
	for _, k := range []Kind{KindIf, KindFor, KindReturn, KindBlock, KindAssign, KindExprStmt} {
		if !k.IsStmt() {
			t.Errorf("%v should be a statement", k)
		}
	}
	for _, k := range []Kind{KindOther, KindCall, KindIdent, KindBinary, KindLit, KindValueSpec} {
		if k.IsStmt() {
			t.Errorf("%v should not be a statement", k)
		}
	}
}

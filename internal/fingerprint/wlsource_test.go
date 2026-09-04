package fingerprint

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/gofront"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// chainSrc has a call with a name, an assignment token, a loop, a guard and
// a nested closure — enough depth that most nodes sit deeper than some round.
const chainSrc = `
func Serve(xs []int) error {
	total := 0
	for _, x := range xs {
		if x > 0 {
			total += x
		}
	}
	go func() { println(total) }()
	if total == 0 {
		return fmt.Errorf("empty: %d", total)
	}
	return nil
}
`

// shapeOf parses a snippet and returns the tree the bag is built over —
// Shape, the canonical body — because that is the tree a label lookup walks.
func shapeOf(t *testing.T, src string) *syntax.Node {
	t.Helper()
	f, err := gofront.Parse("snippet.go", []byte("package p\n"+src))
	if err != nil || f == nil || len(f.Funcs) == 0 {
		t.Fatalf("parse snippet: funcs=%v err=%v", f != nil && len(f.Funcs) > 0, err)
	}
	return f.Funcs[0].Shape()
}

func preorder(root *syntax.Node) []*syntax.Node {
	var out []*syntax.Node
	syntax.Inspect(root, func(n *syntax.Node) bool {
		if n != nil {
			out = append(out, n)
		}
		return true
	})
	return out
}

// TestLabelChainsSumToBag is the contract that makes a chain trustworthy:
// pouring every node's labels back into one multiset is the bag, count for
// count, on both the canonical tree and the body as written.
func TestLabelChainsSumToBag(t *testing.T) {
	for _, tree := range []*syntax.Node{shapeOf(t, chainSrc), parseFunc(t, chainSrc), shapeOf(t, srcSum)} {
		chains := LabelChains(tree)
		got := make(map[uint64]int)
		for _, c := range chains {
			for h := 0; h <= WLRounds; h++ {
				got[c.Labels[h]]++
			}
		}
		if !sameBag(t, got, asMap(WLBagOf(tree))) {
			t.Fatal("chains do not reproduce the bag")
		}
	}
}

func TestLabelChainsPreOrder(t *testing.T) {
	tree := shapeOf(t, chainSrc)
	nodes := preorder(tree)
	chains := LabelChains(tree)
	if len(chains) != len(nodes) {
		t.Fatalf("%d chains for %d nodes", len(chains), len(nodes))
	}
	for i, n := range nodes {
		kind, name := wlLabel0(n)
		if chains[i].Kind != kind || chains[i].Name != name {
			t.Errorf("node %d: chain says %s/%q, tree says %s/%q", i, chains[i].Kind, chains[i].Name, kind, name)
		}
		if chains[i].Labels[0] != wlKind(kind.String(), name) {
			t.Errorf("node %d: label_0 is not wlKind of its own kind and name", i)
		}
	}
	// The root is the block: its subtree is the whole tree.
	if chains[0].Nodes != len(nodes) {
		t.Errorf("root subtree size %d, want %d", chains[0].Nodes, len(nodes))
	}
}

func TestLabelChainsDepthAndSize(t *testing.T) {
	tree := shapeOf(t, chainSrc)
	nodes := preorder(tree)
	chains := LabelChains(tree)
	rootDepth := 0
	for i, n := range nodes {
		if len(n.Kids) == 0 {
			if chains[i].Depth != 0 || chains[i].Nodes != 1 {
				t.Errorf("leaf %d: depth %d size %d, want 0 and 1", i, chains[i].Depth, chains[i].Nodes)
			}
		}
		rootDepth = max(rootDepth, chains[i].Depth)
	}
	if chains[0].Depth != rootDepth {
		t.Errorf("root depth %d is not the maximum %d", chains[0].Depth, rootDepth)
	}
	if rootDepth < 4 {
		t.Errorf("fixture is only %d deep; it exists to exercise labels shallower than the tree", rootDepth)
	}
}

func TestLabelChainsNil(t *testing.T) {
	if LabelChains(nil) != nil {
		t.Fatal("nil tree must yield nil chains")
	}
	if Outline(nil, 2) != nil {
		t.Fatal("nil tree must yield nil outline")
	}
}

func TestOutlineNamesAndTruncates(t *testing.T) {
	tree := shapeOf(t, chainSrc)
	full := Outline(tree, -1)
	joined := strings.Join(full, "\n")
	for _, want := range []string{"CALL/Errorf", "ASSIGN/:=", "BIN/>", "RANGE", "FUNCLIT"} {
		if !strings.Contains(joined, want) {
			t.Errorf("outline missing %q:\n%s", want, joined)
		}
	}
	if len(full) != len(preorder(tree)) {
		t.Errorf("unbounded outline has %d lines for %d nodes", len(full), len(preorder(tree)))
	}
	zero := Outline(tree, 0)
	if len(zero) != 1 || zero[0] != "BLOCK" {
		t.Errorf("levels 0 must be the root alone, got %v", zero)
	}
	one := Outline(tree, 1)
	if len(one) != 1+len(tree.Kids) {
		t.Errorf("levels 1 must be the root and its children: %d lines for %d children", len(one), len(tree.Kids))
	}
	for _, line := range one[1:] {
		if !strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "    ") {
			t.Errorf("child line indented wrong: %q", line)
		}
	}
}

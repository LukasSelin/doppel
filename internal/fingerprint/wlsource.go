package fingerprint

import (
	"slices"
	"strings"

	"github.com/LukasSelin/doppel/internal/syntax"
)

// This file exists so a debug view can say *which node* a Weisfeiler-Lehman
// label came from. The bag deliberately forgets that: WLBag merges every
// node's labels into one multiset because scoring is a merge of two sorted
// runs and node identity would only be weight in it. But a reader holding a
// hash from the fingerprint view and asking "what shape is this" needs the
// node back, and the node is still in the tree — only the bag dropped it.
//
// LabelChains is the same walk as wlBagOf, keyed by node instead of by
// label. It calls wlLabel0, wlKind and wlHash directly, as wlexplain.go
// does and for the reason wlexplain.go states: a copy of the recurrence
// would be one edit away from naming nodes whose labels are in no bag. The
// per-node vector it returns is exactly the lab[] wlBagOf computes and then
// pours into the map — and TestLabelChainsSumToBag pins that pouring the
// chains back in reproduces the bag, count for count.
//
// Outline is the other half of the same question. A round-h label folds in
// h levels below its node and nothing further; a rendering of the whole
// subtree therefore overstates what the hash saw whenever the subtree is
// deeper than h. The outline is the hashed extent itself — the label_0
// vocabulary, truncated at h — so it is the one rendering that cannot claim
// more than the label does. It is language-neutral for the same reason the
// bag is.

// NodeLabels is one node's Weisfeiler-Lehman labels at every round, with
// what a reader needs to place them: the node's kind and label_0 name, and
// how much tree sits below it.
type NodeLabels struct {
	Kind LabelKind
	// Name is the one token that is part of the kind — "Errorf" for a call,
	// ":=" for an assignment — and empty where the kind is the whole label.
	// It is the second argument wlLabel0 returned, kept so an outline can
	// print CALL/Errorf rather than CALL.
	Name string
	// Labels[h] is the node's label at round h. Labels[0] is wlKind over
	// Kind and Name; each later round folds in one more level of children.
	Labels [WLRounds + 1]uint64
	// Depth is how many levels of tree sit below the node: 0 for a leaf.
	// A round-h label with h >= Depth has folded in the whole subtree.
	Depth int
	// Nodes is the subtree size, the node itself included.
	Nodes int
}

// wlChainSlot is one finished node's contribution to its parent, in the
// arena LabelChains reuses across siblings: the label vector wlBagOf's
// arena holds, plus the two subtree facts NodeLabels reports.
type wlChainSlot struct {
	lab   wlLabels
	depth int
	nodes int
}

// wlChainFrame is one open node during the walk: what wlFrame carries, plus
// the node's index in the pre-order output so the entry can be filled on
// the way back up.
type wlChainFrame struct {
	label0 uint64
	kind   LabelKind
	name   string
	start  int
	index  int
}

// LabelChains returns one NodeLabels per node of the tree, in pre-order:
// entry i describes the i-th non-nil node syntax.Inspect visits, which is
// the same order a frontend's own traversal produced the tree in. A nil
// tree yields nil, mirroring WLBag's "no body".
//
// Pouring every entry's Labels back into one multiset reproduces WLBagOf
// over the same tree exactly.
func LabelChains(root *syntax.Node) []NodeLabels {
	if root == nil {
		return nil
	}
	var out []NodeLabels
	var frames []wlChainFrame
	var kids []wlChainSlot
	var buf []uint64

	syntax.Inspect(root, func(n *syntax.Node) bool {
		if n != nil {
			kind, name := wlLabel0(n)
			frames = append(frames, wlChainFrame{
				label0: wlKind(kind.String(), name),
				kind:   kind,
				name:   name,
				start:  len(kids),
				index:  len(out),
			})
			out = append(out, NodeLabels{Kind: kind, Name: name})
			return true
		}
		last := len(frames) - 1
		if last < 0 {
			return false
		}
		fr := frames[last]
		frames = frames[:last]

		var lab wlLabels
		lab[0] = fr.label0
		children := kids[fr.start:]
		depth, nodes := 0, 1
		for h := 1; h <= wlRounds; h++ {
			buf = buf[:0]
			for i := range children {
				buf = append(buf, children[i].lab[h-1])
			}
			// Sorted, so sibling order never reaches a label — the same
			// line wlBagOf has, for the same reason.
			slices.Sort(buf)
			lab[h] = wlHash(h, lab[h-1], buf)
		}
		for i := range children {
			depth = max(depth, children[i].depth+1)
			nodes += children[i].nodes
		}
		kids = append(kids[:fr.start], wlChainSlot{lab: lab, depth: depth, nodes: nodes})

		e := &out[fr.index]
		e.Labels = lab
		e.Depth = depth
		e.Nodes = nodes
		return false
	})
	return out
}

// Outline renders a subtree in the label_0 vocabulary: one line per node,
// indented by depth, the kind's hash name plus "/name" where the kind
// carries one (CALL/Errorf, ASSIGN/:=). levels bounds how far below root
// the outline goes — pass the round of the label being explained and the
// outline is exactly the extent that label hashed; a negative value is
// unbounded. A nil root yields nil.
//
// Sibling order is the tree's, not the sorted order the hash folds children
// in. The hash is over a multiset, so the order shown is not part of the
// claim, but a reader matching the outline against code wants the code's
// order.
func Outline(root *syntax.Node, levels int) []string {
	if root == nil {
		return nil
	}
	var out []string
	var walk func(n *syntax.Node, depth int)
	walk = func(n *syntax.Node, depth int) {
		kind, name := wlLabel0(n)
		line := kind.String()
		if name != "" {
			line += "/" + name
		}
		out = append(out, strings.Repeat("  ", depth)+line)
		if levels >= 0 && depth >= levels {
			return
		}
		for _, k := range n.Kids {
			walk(k.Node, depth+1)
		}
	}
	walk(root, 0)
	return out
}

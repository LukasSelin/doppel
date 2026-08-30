package fingerprint

import (
	"slices"
	"strings"

	"github.com/LukasSelin/doppel/internal/syntax"
)

// This file exists so a report can say *what* two bodies differ by, in words,
// without any consumer of the WL bag having to know how a label is built.
//
// A Fingerprint.WL bag is a multiset of opaque uint64s. That is exactly right
// for scoring — a merge of two sorted runs, no strings touched — and exactly
// useless for explaining: "the bags differ in three labels" tells a reader
// nothing they can check. LowLabels re-derives the shallow end of the same
// recurrence *with* the node each label was computed at, so a differing label
// can be named ("defer", "if") instead of printed as a hash.
//
// # Only h <= 1, and that is the point
//
// A label at h=0 is a node's kind and nothing else, so the h=0 multiset is
// literally "how many defers, how many ifs" — the quantity an English
// sentence about a difference wants. At h=1 a label additionally folds in the
// kinds of the node's immediate children, so it changes for a node whose
// shape moved even when the kind counts agree, and it changes for every
// ancestor of such a node too. That cascade is why the h=1 labels are useful
// for *detecting* a difference and misleading for *counting* one; the caller
// is expected to use them that way, and analyzer.Explain documents the split.
//
// Going deeper is pointless here: an h=3 label describes a whole region, and
// no short phrase names a region.
//
// # Consistency with WLBag
//
// The recurrence, the hash and the label_0 function are WLBag's own, called
// directly rather than re-implemented: a copy would be one edit away from
// producing labels that name nodes in a bag they do not appear in. Even the
// display name is WLBag's, now that a label carries its LabelKind:
// LabelKind.Word is the one table, so a kind cannot be named one thing here
// and hashed as another there.

// A LowLabel is one Weisfeiler-Lehman label from the shallow rounds together
// with the kind of node that produced it.
//
// The mapping is a function rather than a relation: a label at h=0 is a hash
// of the kind itself, and a label at h=1 is a hash of that label, so no two
// kinds can produce one label — except by hash collision, which is the same
// collision budget every other multiset in this package runs on. A collision
// would mislabel one word in one sentence; it cannot move a score, because
// nothing here is scored.
type LowLabel struct {
	Label uint64
	H     int    // 0 or 1
	Kind  string // display name of the node the label describes
}

// LowLabels returns the h=0 and h=1 labels of a function body, each paired
// with the kind of node it was computed at, sorted ascending by (Label, H,
// Kind) and deduplicated on Label.
//
// The caller passes the canonical tree — syntax.Func.Shape, which is what
// parser stores as CodeUnit.Canonical — for the same reason WLBag does: these
// labels are meant to be looked up in a bag built from that tree, and a body
// as written produces labels that are in no bag anyone holds.
//
// A nil tree yields nil, mirroring the zero Fingerprint's "no body".
func LowLabels(root *syntax.Node) []LowLabel {
	if root == nil {
		return nil
	}

	// One frame per open node, and one arena slot per finished node holding
	// its label_0 — which is all an h=1 label needs from its children. The
	// arena is reused across siblings exactly as WLBag reuses its own: a
	// node overwrites its children's slots with its single slot on the way
	// back up, so the whole walk costs one slice no matter how deep it nests.
	type frame struct {
		label0 uint64
		kind   string
		start  int
	}
	var frames []frame
	var kids []uint64
	var buf []uint64
	var out []LowLabel

	syntax.Inspect(root, func(n *syntax.Node) bool {
		if n != nil {
			kind, name := wlLabel0(n)
			frames = append(frames, frame{label0: wlKind(kind.String(), name), kind: kind.Word(), start: len(kids)})
			return true
		}
		last := len(frames) - 1
		if last < 0 {
			// Unreachable while the push above is unconditional; guarded for
			// the same reason WLBag guards it, since a panic here would take
			// down a report over a sentence.
			return false
		}
		fr := frames[last]
		frames = frames[:last]

		buf = append(buf[:0], kids[fr.start:]...)
		// Sorted, so sibling order never reaches a label — the recurrence is
		// over a multiset of children, not a sequence.
		slices.Sort(buf)
		label1 := wlHash(1, fr.label0, buf)

		kids = append(kids[:fr.start], fr.label0)
		out = append(out,
			LowLabel{Label: fr.label0, H: 0, Kind: fr.kind},
			LowLabel{Label: label1, H: 1, Kind: fr.kind},
		)
		return false
	})

	slices.SortFunc(out, compareLowLabel)
	return slices.CompactFunc(out, func(a, b LowLabel) bool { return a.Label == b.Label })
}

// compareLowLabel is the total order LowLabels returns. Label leads because
// that is the key a caller looks up; H and Kind break ties so a hash collision
// resolves the same way on every run rather than by walk order.
func compareLowLabel(a, b LowLabel) int {
	switch {
	case a.Label < b.Label:
		return -1
	case a.Label > b.Label:
		return 1
	case a.H != b.H:
		return a.H - b.H
	}
	return strings.Compare(a.Kind, b.Kind)
}

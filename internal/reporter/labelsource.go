package reporter

import (
	"fmt"
	"io"
	"strings"

	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// This file is the --label half of the fingerprint view: given a label a
// reader copied out of a bag row, show the node(s) that produced it.
//
// The bag row says "depth-2 IF, df 25, weight 3.64" and nothing more,
// because a WL label is a hash with no short faithful name (see
// fingerprint.DescribeLabel). The tree the bag was built over is still in
// hand, though, and fingerprint.LabelChains re-derives every node's label
// vector from it under the same recurrence. So the lookup is a scan: every
// (node, round) whose label matches is an occurrence, and the occurrence is
// shown two ways.
//
// The *render* is Go text from the frontend, and it is the canonical form —
// alpha-renamed, rules applied — never the source as written, because the
// canonical tree keeps no positions to map back through. The section says
// so once. The *outline* is the label_0 vocabulary truncated at the label's
// round, which is exactly the extent the hash folded in; it prints whenever
// there is no render or the render shows more tree than the label saw.
// The render is the convenience; the outline is the claim.

// LabelSource is what a label lookup reads for one unit: the tree its bag
// was computed over, and optionally a rendering per node.
type LabelSource struct {
	// Tree is CodeUnit.Canonical, or a frontend's re-derivation of it that
	// the caller has checked carries the unit's bag.
	Tree *syntax.Node
	// Renders pairs with Tree's nodes in pre-order — entry i is the i-th
	// non-nil node syntax.Inspect visits — and is nil when no frontend can
	// render this unit's language. An entry may be "" for a node that has
	// no standalone printed form.
	Renders []string
}

// PrintLabelOccurrences renders, for each requested label, where in one
// function it occurs.
func PrintLabelOccurrences(w io.Writer, u parser.CodeUnit, src LabelSource, labels []uint64, idf *fingerprint.LabelIDF, meta FingerprintMeta) {
	fmt.Fprintf(w, "labels in %s%s\n", qualifiedName(u), canonicalFormClause(src))
	side := newLabelSide(u, src, idf)
	for _, label := range labels {
		fmt.Fprintf(w, "  label #%x: %s\n", label, side.headline(label, ""))
		side.printOccurrences(w, label, "    ", meta.LabelTop)
	}
}

// PrintLabelOccurrencesPair renders the same lookup on both sides of a pair,
// so a label from the shared or only-in-X sections can be seen on each body.
func PrintLabelOccurrencesPair(w io.Writer, a, b parser.CodeUnit, srcA, srcB LabelSource, labels []uint64, idf *fingerprint.LabelIDF, meta FingerprintMeta) {
	fmt.Fprintf(w, "labels in A %s and B %s%s\n", qualifiedName(a), qualifiedName(b), canonicalFormClause(srcA, srcB))
	sa, sb := newLabelSide(a, srcA, idf), newLabelSide(b, srcB, idf)
	for _, label := range labels {
		fmt.Fprintf(w, "  label #%x: %s\n", label, pairHeadline(sa, sb, label))
		fmt.Fprintf(w, "    A %s\n", qualifiedName(a))
		sa.printOccurrences(w, label, "      ", meta.LabelTop)
		fmt.Fprintf(w, "    B %s\n", qualifiedName(b))
		sb.printOccurrences(w, label, "      ", meta.LabelTop)
	}
}

func qualifiedName(u parser.CodeUnit) string {
	if u.Package == "" {
		return u.Name
	}
	return u.Package + "." + u.Name
}

// canonicalFormClause is the one-line caveat every label section opens
// with when any side has a render: the code shown is what the hash saw.
func canonicalFormClause(srcs ...LabelSource) string {
	for _, s := range srcs {
		if s.Renders != nil {
			return " (code shown is the canonical form the label hashed, not the source as written)"
		}
	}
	return ""
}

// labelSide is one unit prepared for lookups: its chains and nodes in
// pre-order, its bag rows by label, and the corpus weights.
type labelSide struct {
	unit    parser.CodeUnit
	src     LabelSource
	chains  []fingerprint.NodeLabels
	nodes   []*syntax.Node
	rows    map[uint64]bagRow
	idf     *fingerprint.LabelIDF
	hasBody bool
}

func newLabelSide(u parser.CodeUnit, src LabelSource, idf *fingerprint.LabelIDF) *labelSide {
	s := &labelSide{unit: u, src: src, idf: idf, hasBody: u.Fingerprint.Nodes > 0 && src.Tree != nil}
	if !s.hasBody {
		return s
	}
	s.chains = fingerprint.LabelChains(src.Tree)
	syntax.Inspect(src.Tree, func(n *syntax.Node) bool {
		if n != nil {
			s.nodes = append(s.nodes, n)
		}
		return true
	})
	s.rows = make(map[uint64]bagRow, len(u.Fingerprint.WL))
	for _, r := range bagRows(u.Fingerprint.WL, idf) {
		s.rows[r.label] = r
	}
	return s
}

// occurrence is one (node, round) carrying the label.
type occurrence struct {
	idx int
	h   int
}

func (s *labelSide) occurrences(label uint64) []occurrence {
	var out []occurrence
	for i, c := range s.chains {
		for h := 0; h <= fingerprint.WLRounds; h++ {
			if c.Labels[h] == label {
				out = append(out, occurrence{idx: i, h: h})
			}
		}
	}
	return out
}

// headline is the bag-row meta for a label on this side: what the view's
// row said, so the two can be matched by eye. countWord names the side
// when a pair headline composes two of these.
func (s *labelSide) headline(label uint64, countWord string) string {
	r, ok := s.rows[label]
	if !ok {
		return fmt.Sprintf("not in this function's bag (df %s: %s)", dfCell(s.idf.DF(label)), corpusPresence(s.idf, label))
	}
	return fmt.Sprintf("%s  df %s  weight %.2f  %scount %d",
		fingerprint.DescribeLabel(r.h, r.kind), dfCell(r.df), r.weight, countWord, r.count)
}

// pairHeadline states the label once and counts it per side.
func pairHeadline(a, b *labelSide, label uint64) string {
	ra, okA := a.rows[label]
	rb, okB := b.rows[label]
	switch {
	case !okA && !okB:
		return fmt.Sprintf("in neither bag (df %s: %s)", dfCell(a.idf.DF(label)), corpusPresence(a.idf, label))
	case okA && okB:
		return fmt.Sprintf("%s  df %s  weight %.2f  A ×%d  B ×%d",
			fingerprint.DescribeLabel(ra.h, ra.kind), dfCell(ra.df), ra.weight, ra.count, rb.count)
	case okA:
		return fmt.Sprintf("%s  df %s  weight %.2f  A ×%d  B absent",
			fingerprint.DescribeLabel(ra.h, ra.kind), dfCell(ra.df), ra.weight, ra.count)
	}
	return fmt.Sprintf("%s  df %s  weight %.2f  A absent  B ×%d",
		fingerprint.DescribeLabel(rb.h, rb.kind), dfCell(rb.df), rb.weight, rb.count)
}

func corpusPresence(idf *fingerprint.LabelIDF, label uint64) string {
	if idf.DF(label) == 0 {
		return "unknown to the corpus"
	}
	return "carried elsewhere in the corpus"
}

// printOccurrences lists every (node, round) carrying the label, capped at
// top (0 = all) with the count of the rest, and shows each as a render
// and/or an outline.
func (s *labelSide) printOccurrences(w io.Writer, label uint64, indent string, top int) {
	if !s.hasBody {
		fmt.Fprintf(w, "%sno body: nothing carries a label\n", indent)
		return
	}
	occ := s.occurrences(label)
	if len(occ) == 0 {
		fmt.Fprintf(w, "%sabsent\n", indent)
		return
	}
	shown := occ
	if top > 0 && len(occ) > top {
		shown = occ[:top]
	}
	for _, o := range shown {
		c := s.chains[o.idx]
		fmt.Fprintf(w, "%snode %d of %d: %s, subtree %d deep (%d nodes); %s\n",
			indent, o.idx+1, len(s.chains), c.Kind.Word(), c.Depth, c.Nodes, coverageClause(o.h, c.Depth))
		render := ""
		if s.src.Renders != nil && o.idx < len(s.src.Renders) {
			render = s.src.Renders[o.idx]
		}
		if render != "" {
			for _, line := range strings.Split(render, "\n") {
				fmt.Fprintf(w, "%s  %s\n", indent, line)
			}
		}
		if render == "" || c.Depth > o.h {
			fmt.Fprintf(w, "%s  hashed extent (%s; identifier names and literal values are not part of it):\n",
				indent, levelsWord(o.h))
			for _, line := range fingerprint.Outline(s.nodes[o.idx], o.h) {
				fmt.Fprintf(w, "%s    %s\n", indent, line)
			}
		}
	}
	if rest := len(occ) - len(shown); rest > 0 {
		fmt.Fprintf(w, "%s… and %d more\n", indent, rest)
	}
}

// coverageClause says how much of the subtree the label at round h folded
// in — the fact a reader needs before trusting a whole-subtree render.
func coverageClause(h, depth int) string {
	if h >= depth {
		return fmt.Sprintf("depth-%d covers all of it", h)
	}
	return fmt.Sprintf("depth-%d folds %d of those %d levels", h, h, depth)
}

func levelsWord(h int) string {
	if h == 1 {
		return "1 level"
	}
	return fmt.Sprintf("%d levels", h)
}

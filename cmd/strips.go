package cmd

import (
	"fmt"
	"sort"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// maxStrips bounds the strip view. Each strip is a column of bars plus a card
// per reported pair, so three is already most of a screen; the census table
// below carries the rest.
const maxStrips = 3

// htmlStrips renders the families whose members all live in one file.
//
// The strip only means something within a file. Its bar is the distance from
// one declaration to the next, so a family spread across packages has no
// silhouette to show — those families are still in the census, they just have
// no strip. On the corpora this was built against the same-file families are
// the ones worth drawing anyway: a file that declares ten near-identical
// handlers in a row is exactly what the view is for.
func htmlStrips(res Result, fams []family.Family, pairs []analyzer.SimilarPair) []reporter.HTMLStrip {
	spans := declarationSpans(res)

	var out []reporter.HTMLStrip
	for i, f := range fams {
		if len(out) == maxStrips {
			break
		}
		file, ok := singleFile(res, f)
		if !ok {
			continue
		}
		out = append(out, buildStrip(res, f, i+1, file, spans, pairs))
	}
	return out
}

// declarationSpans measures every unit's declaration length as the gap to the
// next declaration in the same file.
//
// This is derived, not measured. doppel's fingerprint counts AST nodes, which
// is the real size of a body; the gap between declarations includes comments,
// blank lines and anything else the file puts between them. The design says so
// itself — the bar is a reading aid and the scores beneath it are the analysis
// — and the last declaration in a file has no successor, so it has no bar.
func declarationSpans(res Result) map[int]int {
	byFile := map[string][]int{}
	for i, u := range res.Units {
		byFile[u.File] = append(byFile[u.File], i)
	}
	spans := make(map[int]int, len(res.Units))
	for _, idx := range byFile {
		sort.Slice(idx, func(a, b int) bool {
			return res.Units[idx[a]].StartLine < res.Units[idx[b]].StartLine
		})
		for n := 0; n+1 < len(idx); n++ {
			gap := res.Units[idx[n+1]].StartLine - res.Units[idx[n]].StartLine
			if gap > 0 {
				spans[idx[n]] = gap
			}
		}
	}
	return spans
}

// singleFile reports the one file a family lives in, or false when it spans
// several.
func singleFile(res Result, f family.Family) (string, bool) {
	file := ""
	for _, m := range f.Members {
		if m < 0 || m >= len(res.Units) {
			return "", false
		}
		switch {
		case file == "":
			file = res.Units[m].File
		case res.Units[m].File != file:
			return "", false
		}
	}
	return file, file != ""
}

func buildStrip(res Result, f family.Family, n int, file string,
	spans map[int]int, pairs []analyzer.SimilarPair) reporter.HTMLStrip {

	members := append([]int(nil), f.Members...)
	// Declaration order, which is what makes the silhouette readable: the
	// family's own member order is by unit index, which is file-walk order and
	// only coincidentally the same thing.
	sort.Slice(members, func(a, b int) bool {
		return res.Units[members[a]].StartLine < res.Units[members[b]].StartLine
	})

	s := reporter.HTMLStrip{
		Title:    fmt.Sprintf("Family %d", n),
		File:     snapshot.RelSlash(res.Root, file),
		Tag:      dominantTag(res, f),
		MinLabel: fmt.Sprintf("%.2f", f.MinEdge),
	}
	if f.Kind != nil {
		s.Title = fmt.Sprintf("Family %d — %s", n, f.Kind.Kind)
	}

	for _, m := range members {
		u := res.Units[m]
		mem := reporter.HTMLStripMember{Name: u.Name, Line: u.StartLine}
		if span, ok := spans[m]; ok {
			mem.HasSpan = true
			mem.Pct = stripPct(span)
			mem.SpanLabel = fmt.Sprintf("%d lines", span)
		} else {
			mem.SpanLabel = "span unknown — last in file"
		}
		s.Members = append(s.Members, mem)
	}

	s.Pairs = stripPairs(f, pairs)
	s.Note = stripNote(len(members), f.MinEdge, len(s.Pairs))
	return s
}

func stripPct(span int) int {
	p := int(float64(span)/float64(stripFullSpan)*100 + 0.5)
	if p > 100 {
		return 100
	}
	if p < 1 {
		return 1
	}
	return p
}

// stripPairs finds the reported pairs both of whose sides are family members,
// so the cards under a strip are the pairs that strip actually produced.
func stripPairs(f family.Family, pairs []analyzer.SimilarPair) []reporter.HTMLPairCard {
	member := make(map[int]bool, len(f.Members))
	for _, m := range f.Members {
		member[m] = true
	}
	var out []reporter.HTMLPairCard
	for i, p := range pairs {
		if !member[p.AIdx] || !member[p.BIdx] {
			continue
		}
		out = append(out, pairCard(i+1, p))
	}
	return out
}

func pairCard(n int, p analyzer.SimilarPair) reporter.HTMLPairCard {
	c := reporter.HTMLPairCard{
		Label:      fmt.Sprintf("#%d", n),
		A:          concepter.QualifiedName(p.A),
		B:          concepter.QualifiedName(p.B),
		ShapeLabel: fmt.Sprintf("%.4f", p.Score),
		// Reported on every card, whether or not the pair has evidence:
		// containment comes from the same two bags the shape score does, so
		// there is no pair that has one and not the other.
		ContainmentLabel: fmt.Sprintf("%.2f", p.Breakdown.Containment),
		Explain:          p.Explain,
	}
	// "nesting" is the design's name for the depth component; the fingerprint
	// calls the field Depth and its own comment says it reports as nesting.
	for _, comp := range []struct {
		name string
		v    float64
	}{
		{"wl", p.Breakdown.WL},
		{"flow", p.Breakdown.Flow},
		{"nesting", p.Breakdown.Depth},
		{"sig", p.Breakdown.Signature},
		{"size", p.Breakdown.SizeRatio},
	} {
		c.Components = append(c.Components, reporter.HTMLComponent{
			Name:  comp.name,
			Pct:   int(comp.v*100 + 0.5),
			Label: fmt.Sprintf("%.2f", comp.v),
			Hot:   comp.v >= 0.90,
		})
	}
	if p.Evidence != nil {
		c.OverlapLabel = fmt.Sprintf("%.2f", p.Evidence.OverlapScore)
		c.Footer = fmt.Sprintf("%d shared callees · neighbourhood %.2f",
			len(p.Evidence.SharedCallees), p.Evidence.NeighborhoodOverlap)
	}
	if p.Retrieval != nil && len(p.Retrieval.Chains) > 0 {
		chain := ""
		for i, ch := range p.Retrieval.Chains {
			if i == 2 {
				break
			}
			if i > 0 {
				chain += " → "
			}
			chain += ch.Render
		}
		if c.Footer != "" {
			c.Footer += " · "
		}
		c.Footer += chain
	}
	return c
}

// stripNote states what the strip shows, in the register the rest of the report
// uses. The design's own notes are hand-written for the mockup; this says the
// same kind of thing from the numbers, and claims nothing it cannot count.
func stripNote(members int, minEdge float64, pairs int) string {
	s := fmt.Sprintf("%d functions declared one after another in a single file. "+
		"Every pair of them is at least %.2f alike", members, minEdge)
	switch pairs {
	case 0:
		return s + "; none of the pairs reached the report's own cutoff, so the family is the finding."
	case 1:
		return s + "; the report names one of the pairs outright."
	}
	return fmt.Sprintf("%s; the report names %d of the pairs outright.", s, pairs)
}

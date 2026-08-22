package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/reporter"
)

// stripFullSpan is the declaration length a strip bar draws at full width.
//
// Fixed rather than scaled per strip, deliberately: the strip view exists so a
// reader can see one family's bars line up against another's, and normalising
// each strip to its own longest member would make every family look equally
// even. Longer declarations clamp, which is honest — the bar stops meaning
// anything precise well before it runs out of room.
const stripFullSpan = 70

// buildHTMLReport assembles the Similarity Report page from a finished run.
//
// Everything except the strips already exists on the Overview the markdown
// report uses, so this is a second consumer of that model rather than a second
// computation of it.
func buildHTMLReport(res Result, ov *reporter.Overview, fams []family.Family,
	famStats family.Stats, pairs []analyzer.SimilarPair, suppressed int) reporter.HTMLReport {

	r := reporter.HTMLReport{
		Target:    targetName(res.Root),
		Threshold: res.Params.Threshold,

		PairsFound:    len(pairs),
		PairsHeldBack: suppressed,

		FamilyCount:   famStats.Families,
		FamilyFuncs:   famStats.Members,
		FamilyLargest: largestFamily(fams),
		EdgesScored:   famStats.Completed,
	}
	if ov == nil {
		return r
	}

	r.Functions = ov.Functions
	r.Packages = ov.Packages
	r.TestsMode = testsWord(ov.TestsMode)
	r.CandidatePairs = ov.UnionPairs
	r.ShapePairs = ov.ShapePairs
	r.ConceptPairs = ov.ConceptPairs
	r.CallPairs = ov.CallPairs
	r.CallOnlyPct = share(ov.OnlyCallPairs, ov.UnionPairs)
	r.ConceptOnlyPct = share(ov.OnlyConceptPairs, ov.UnionPairs)
	r.ArenasSettled = ov.ArenaProfiled
	r.ArenaSingle = ov.ArenaDominance
	r.ArenaCoalition = ov.ArenaCoalition
	r.ArenaContradictory = ov.ArenaConflict
	r.MisfitCount = ov.Misfits
	r.MisfitExcused = ov.MisfitsExcused
	r.Roles = ov.Roles
	r.AbsentConcepts = ov.Absent
	r.Drift = ov.Drift
	r.DriftMore = ov.DriftMore
	r.HabitatsMore = ov.HabitatsMore
	r.MostUniform, r.MostUniformN = ov.MostUniform, ov.MostUniformNorm
	r.MostVaried, r.MostVariedN = ov.MostDiverse, ov.MostDiverseNorm

	r.Taxonomy = htmlTaxonomy(ov)
	r.Habitats = htmlHabitats(ov)
	r.Families, r.FamiliesMore = htmlFamilies(res, fams, famStats)
	r.Strips = htmlStrips(res, fams, pairs)
	return r
}

func testsWord(mode string) string {
	switch mode {
	case "only":
		return "only"
	case "include":
		return "included"
	}
	return "excluded"
}

func share(n, total int) int {
	if total == 0 {
		return 0
	}
	return int(100*float64(n)/float64(total) + 0.5)
}

func largestFamily(fams []family.Family) int {
	n := 0
	for _, f := range fams {
		if len(f.Members) > n {
			n = len(f.Members)
		}
	}
	return n
}

// htmlTaxonomy flattens the concept tree with its indent depth and this
// corpus's convention strength.
//
// Depth is walked from Parent rather than stored: the Overview keeps the tree
// as parent links so the markdown diagram can draw edges, and an indent is a
// rendering concern that only this page has.
func htmlTaxonomy(ov *reporter.Overview) []reporter.HTMLTaxon {
	parent := make(map[string]string, len(ov.Taxonomy))
	for _, n := range ov.Taxonomy {
		parent[n.ID] = n.Parent
	}
	depth := func(id string) int {
		d := 0
		for p := parent[id]; p != ""; p = parent[p] {
			d++
			if d > len(ov.Taxonomy) { // a cycle would hang the render
				break
			}
		}
		return d
	}

	conv := make(map[string]reporter.TagRow, len(ov.Concepts))
	for _, c := range ov.Concepts {
		conv[c.Tag] = c
	}

	out := make([]reporter.HTMLTaxon, 0, len(ov.Taxonomy))
	for _, n := range ov.Taxonomy {
		t := reporter.HTMLTaxon{
			Label:    n.ID,
			Indent:   depth(n.ID) * 20,
			Abstract: n.Abstract,
			Absent:   !n.Abstract && n.Count == 0,
		}
		switch {
		case t.Absent:
			t.CountLabel = "absent"
		case n.Count > 0:
			t.CountLabel = fmt.Sprint(n.Count)
		}
		if c, ok := conv[n.ID]; ok && c.Prototyped {
			t.ConvPct = int(c.Convention*100 + 0.5)
			t.Loose = c.Convention < 0.5
		}
		out = append(out, t)
	}
	return out
}

func htmlHabitats(ov *reporter.Overview) []reporter.HTMLHabitat {
	out := make([]reporter.HTMLHabitat, 0, len(ov.Habitats))
	for _, h := range ov.Habitats {
		meta := fmt.Sprintf("%d fn", h.Functions)
		if h.Misfits > 0 {
			meta += fmt.Sprintf(" · %d", h.Misfits)
		}
		out = append(out, reporter.HTMLHabitat{
			Name:      h.Package,
			NormPct:   int(h.Norm*100 + 0.5),
			MisfitPct: share(h.Misfits, h.Functions),
			Meta:      meta,
		})
	}
	return out
}

// htmlFamilies renders the census rows.
//
// The design's "what repeats" column is editorial prose in the mockup, and
// doppel cannot write it. What it can say is the pair kind — interface
// implementations, a diverged copy — so that is the column, and a family with
// no kind gets an empty cell rather than an invented description.
func htmlFamilies(res Result, fams []family.Family, stats family.Stats) ([]reporter.HTMLFamily, int) {
	shown := fams
	if len(shown) > familiesN && familiesN > 0 {
		shown = shown[:familiesN]
	}
	out := make([]reporter.HTMLFamily, 0, len(shown))
	for i, f := range shown {
		row := reporter.HTMLFamily{
			N:             i + 1,
			Members:       len(f.Members),
			MinLabel:      fmt.Sprintf("%.2f", f.MinEdge),
			EvidenceLabel: fmt.Sprintf("%.0f", f.Evidence),
			AddedLabel:    "—",
			Where:         joinPackages(familyPackages(res, f)),
			Tag:           dominantTag(res, f),
		}
		if f.Completed > 0 {
			row.AddedLabel = fmt.Sprintf("+%d", f.Completed)
		}
		if f.Kind != nil {
			row.What = f.Kind.Kind
		}
		out = append(out, row)
	}
	return out, stats.Families - len(out)
}

// familyPackages lists the distinct packages a family spans, in first-member
// order so the column reads as the family is listed.
func familyPackages(res Result, f family.Family) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range f.Members {
		if m < 0 || m >= len(res.Units) {
			continue
		}
		p := res.Units[m].Package
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func joinPackages(pkgs []string) string {
	const max = 4
	if len(pkgs) > max {
		return fmt.Sprintf("%s +%d more", joinComma(pkgs[:max]), len(pkgs)-max)
	}
	return joinComma(pkgs)
}

func joinComma(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}

// dominantTag is the concept most of a family's members carry, or "" when they
// share none. Ties break on the tag name so the column is stable.
func dominantTag(res Result, f family.Family) string {
	counts := map[string]int{}
	for _, m := range f.Members {
		if m < 0 || m >= len(res.Units) {
			continue
		}
		for _, t := range res.Units[m].Patterns {
			counts[t]++
		}
	}
	tags := make([]string, 0, len(counts))
	for t := range counts {
		tags = append(tags, t)
	}
	sort.Strings(tags) // before the count comparison, so map order never decides
	best, bestN := "", 0
	for _, t := range tags {
		if counts[t] > bestN {
			best, bestN = t, counts[t]
		}
	}
	// A tag one member happens to carry is not what the family is about.
	if bestN*2 < len(f.Members) {
		return ""
	}
	return best
}

// targetName is what the page calls the corpus.
//
// The root as typed is usually "." or an absolute path, and neither reads as
// the name of a codebase in a headline. The directory's own name does.
func targetName(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	if base := filepath.Base(abs); base != "" && base != "." && base != string(filepath.Separator) {
		return base
	}
	return root
}

// isHTMLPath decides the report format from the output path.
//
// An extension rather than a flag: --format already means "what goes to
// stdout", and someone writing report.html has said what they want plainly
// enough that asking them to say it twice is ceremony.
func isHTMLPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".html" || ext == ".htm"
}

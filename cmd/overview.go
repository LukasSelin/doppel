package cmd

import (
	"sort"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
)

// buildOverview assembles what doppel understands about the corpus.
//
// All of it already exists at this point in the run — Result carries the
// culture model, the ontology, the tag counts and the call graph — and until
// now almost none of it reached the document. The querying lives here rather
// than in reporter so that reporter never learns about culture or ontology; it
// receives plain presorted rows.
func buildOverview(res Result, suppressed int) *reporter.Overview {
	if len(res.Units) == 0 {
		return nil
	}
	ov := &reporter.Overview{
		Root:       res.Root,
		Functions:  len(res.Units),
		TestsMode:  res.Params.TestsMode,
		Threshold:  res.Params.Threshold,
		Suppressed: suppressed,
		SelfDup:    map[string]int{},

		ShapePairs:       res.Retrieval.ShapePairs,
		ConceptPairs:     res.Retrieval.ConceptPairs,
		CallPairs:        res.Retrieval.CallPairs,
		UnionPairs:       res.Retrieval.Union,
		OnlyConceptPairs: res.Retrieval.OnlyConcept,
		OnlyCallPairs:    res.Retrieval.OnlyCall,
	}

	pkgFuncs := map[string]int{}
	for _, u := range res.Units {
		if u.Package != "" {
			pkgFuncs[u.Package]++
		}
	}
	ov.Packages = len(pkgFuncs)

	ov.Concepts, ov.Absent, ov.Taxonomy = conceptRows(res)
	ov.Roles = roleRows(res)
	overviewCulture(ov, res, pkgFuncs)
	overviewDuplication(ov, res)
	return ov
}

// conceptRows returns the concepts this corpus uses, the ones it never does,
// and the taxonomy flattened parents-first for drawing.
//
// Absent concepts are a first-class answer, not a gap in a table: "nothing here
// is tagged retry" settles a question that the list of present tags only
// narrows. The hook's session-start digest already leads with the same fact.
func conceptRows(res Result) ([]reporter.TagRow, []string, []reporter.TaxonomyNode) {
	var rows []reporter.TagRow
	var absent []string
	var tree []reporter.TaxonomyNode

	// Declaration order from the ontology, so the tree is emitted parents-first
	// without a recursive walk and without any map deciding the order.
	for _, term := range res.Onto.TermsOfKind(ontology.KindConcept) {
		count := res.TagCounts[term.ID]
		tree = append(tree, reporter.TaxonomyNode{
			ID:       string(term.ID),
			Parent:   string(term.Parent),
			Abstract: term.Abstract,
			Count:    count,
		})
		if term.Abstract {
			continue
		}
		if count == 0 {
			absent = append(absent, string(term.ID))
			continue
		}
		row := reporter.TagRow{Tag: string(term.ID), Count: count}
		if res.Culture != nil {
			if c, ok := res.Culture.ConventionStrength(string(term.ID)); ok {
				row.Convention, row.Prototyped = c, true
			}
		}
		rows = append(rows, row)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}
		return rows[i].Tag < rows[j].Tag
	})
	sort.Strings(absent)
	return rows, absent, tree
}

func roleRows(res Result) []reporter.RoleRow {
	counts := map[string]int{}
	for _, d := range res.Docs {
		if d.Role != "" {
			counts[d.Role]++
		}
	}
	out := make([]reporter.RoleRow, 0, len(counts))
	for role, n := range counts {
		out = append(out, reporter.RoleRow{Role: role, Count: n})
	}
	// By role name, never by count: count ties are the common case and would
	// leave the order dependent on map iteration.
	sort.Slice(out, func(i, j int) bool { return out[i].Role < out[j].Role })
	return out
}

// overviewCulture fills in the habitat rows and the corpus superlatives.
//
// HabitatNorm and HabitatTemperature are queried per package rather than read
// from Stats, which carries only the two extremes. Both accessors return false
// for a package too small to model, which is how unmodeled packages are skipped
// without this code knowing the minimum.
func overviewCulture(ov *reporter.Overview, res Result, pkgFuncs map[string]int) {
	if res.Culture == nil {
		return
	}
	cs := res.Culture.Stats()
	ov.Misfits = cs.HabitatMisfits
	ov.ConceptsModeled = cs.ConceptsModeled
	ov.Unusual = cs.UnusualRealizations
	ov.MostUniform, ov.MostUniformNorm = cs.MostUniformHabitat, cs.MostUniformNorm
	ov.MostDiverse, ov.MostDiverseNorm = cs.MostDiverseHabitat, cs.MostDiverseNorm
	ov.Strongest, ov.StrongestStrength = cs.StrongestConvention, cs.StrongestConventionStrength
	ov.Loosest, ov.LoosestC = cs.LoosestConvention, cs.LoosestConventionStrength
	ov.ArenaProfiled = cs.ArenaProfiled
	ov.ArenaDominance, ov.ArenaCoalition = cs.ArenaDominance, cs.ArenaCoalition
	ov.ArenaConflict, ov.ArenaWeak = cs.ArenaConflict, cs.ArenaWeak

	misfitsPerPkg := map[string]int{}
	for i, u := range res.Units {
		if res.Culture.Misfit(i) {
			misfitsPerPkg[u.Package]++
		}
	}

	pkgs := make([]string, 0, len(pkgFuncs))
	for p := range pkgFuncs {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs) // before any weight comparison, so map order never survives

	var rows []reporter.HabitatRow
	for _, p := range pkgs {
		norm, ok := res.Culture.HabitatNorm(p)
		if !ok {
			continue
		}
		temp, _ := res.Culture.HabitatTemperature(p)
		rows = append(rows, reporter.HabitatRow{
			Package:     p,
			Functions:   pkgFuncs[p],
			Norm:        norm,
			Temperature: temp,
			Misfits:     misfitsPerPkg[p],
		})
	}
	ov.Habitats, ov.HabitatsMore = reporter.SortHabitats(rows)
}

// overviewDuplication folds merge-worthy pairs up to their packages.
//
// The pair list answers "which two functions"; nothing answered "which parts of
// this system keep growing the same code". A cross-package edge is the finding —
// two packages independently solving one problem — where an intra-package count
// is usually a family of deliberate siblings, so the two are counted separately
// and rendered differently.
func overviewDuplication(ov *reporter.Overview, res Result) {
	links := map[[2]string]int{}
	for _, p := range res.Pairs {
		if !p.MergeWorthy() {
			continue
		}
		a, b := packageOf(p.A), packageOf(p.B)
		if a == "" || b == "" {
			continue
		}
		if a == b {
			ov.SelfDup[a]++
			continue
		}
		if a > b {
			a, b = b, a
		}
		links[[2]string{a, b}]++
	}
	out := make([]reporter.PackageLink, 0, len(links))
	for k, n := range links {
		out = append(out, reporter.PackageLink{A: k[0], B: k[1], Pairs: n})
	}
	ov.Links, ov.LinksMore = reporter.SortLinks(out)
}

// packageOf is the package half of a unit's identity, taken through the same
// helper the call graph keys on rather than by reading u.Package directly, so
// the report and the graph cannot disagree about what a package is.
func packageOf(u parser.CodeUnit) string {
	return concepter.KeyPackage(concepter.QualifiedName(u))
}

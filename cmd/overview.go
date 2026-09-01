package cmd

import (
	"math"
	"slices"
	"sort"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/culture"
	"github.com/LukasSelin/doppel/internal/lexicon"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/snapshot"
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

		Metrics: reporter.CorpusMetrics{
			TotalNodes:           res.ConsStats.TotalNodes,
			UniqueSubtrees:       res.ConsStats.UniqueSubtrees,
			NNTotal:              res.NN.Total,
			NNScored:             res.NN.Scored,
			NNP50:                res.NN.P50,
			NNP90:                res.NN.P90,
			NNP99:                res.NN.P99,
			NNAtOrAboveThreshold: res.NN.AtOrAboveThreshold,
		},
	}

	pkgFuncs := map[string]int{}
	for _, u := range res.Units {
		if u.Package != "" {
			pkgFuncs[u.Package]++
		}
	}
	ov.Packages = len(pkgFuncs)

	ov.Concepts, ov.Absent, ov.Taxonomy = conceptRows(res)
	// Bounded here rather than in conceptRows for the same reason the habitat
	// and link diagrams are: how many nodes fit in a picture is a rendering
	// decision, and reporter owns the number.
	ov.Taxonomy, ov.TaxonomyMore = reporter.BoundTaxonomy(ov.Taxonomy)
	ov.SeedMap = seedMap(res)
	ov.Roles = roleRows(res)
	overviewCulture(ov, res, pkgFuncs)
	overviewDuplication(ov, res)
	buildPractice(ov, res)
	return ov
}

// conceptRows returns the concepts this corpus uses, the ones it never does,
// and the taxonomy flattened parents-first for drawing.
//
// The absent list is a first-class answer, not a gap in a table: "this codebase
// has no retry practice" settles a question the list of present concepts only
// narrows. It names *seeds*, not learned concepts — a learned concept cannot be
// absent, since it exists only because some function carries it — so the seed
// vocabulary is the one fixed list left to measure absence against. The hook's
// session-start digest reports the same fact from the snapshot.
func conceptRows(res Result) ([]reporter.TagRow, []string, []reporter.TaxonomyNode) {
	var rows []reporter.TagRow
	var absent []string
	var tree []reporter.TaxonomyNode

	// Declaration order from the ontology, so the tree is emitted parents-first
	// without a recursive walk and without any map deciding the order.
	absent = append(absent, res.UnusedSeeds...)
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
			// Unreachable for a learned vocabulary — a concept exists because
			// functions carry it — and kept as the guard that says so. Absence
			// is a question only the seeds can answer, and seedMap draws it.
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

// seedMap is the authored seed taxonomy with this run's yield on every leaf:
// the map the report opened with before the vocabulary became corpus-derived,
// restored and now measured rather than asserted.
//
// It walks ontology.Default() and not res.Onto, deliberately. The run's
// ontology carries the learned concepts as its concrete leaves — hundreds of
// them, which is a wall rather than a picture — where the authored tree is the
// same 8 abstract nodes and 14 seed leaves on every corpus. That fixedness is
// the whole value: it is the one concept diagram two runs, or two
// repositories, can be held beside each other.
//
// Declaration order from the ontology, so the tree is emitted parents-first
// without a recursive walk and without any map deciding the order — the same
// rule conceptRows follows.
func seedMap(res Result) []reporter.TaxonomyNode {
	// No lexicon means nothing grew, and every leaf reads absent. That is the
	// truthful picture for such a run rather than a missing diagram.
	var grown map[string]int
	if res.Lexicon != nil {
		grown = seedYield(res.Lexicon.Concepts(), res.Lexicon.Assignments())
	}
	var tree []reporter.TaxonomyNode
	for _, term := range ontology.Default().TermsOfKind(ontology.KindConcept) {
		tree = append(tree, reporter.TaxonomyNode{
			ID:       string(term.ID),
			Parent:   string(term.Parent),
			Abstract: term.Abstract,
			// Zero for an abstract node, which never renders a count, and for
			// a seed that grew nothing — which is exactly the `absent` the
			// diagram colours and the same set Result.UnusedSeeds holds. A
			// concept exists only because founders carry it, so "grew a
			// concept" and "has at least one member" are one condition, and
			// the red leaves cannot disagree with the sentence below them.
			Count: grown[string(term.ID)],
		})
	}
	return tree
}

// seedYield counts, per seed, how many distinct functions carry at least one
// concept that seed grew.
//
// Distinct functions rather than a sum of lexicon.Concept.Members: several
// concepts can grow from one seed and one function can be a member of more
// than one of them, so summing would report more http_call functions than the
// corpus has.
//
// Two rules the count depends on:
//
//   - A BelowFloor membership does not count. It is a backfilled membership the
//     unit did not earn, and every other boolean reader of a membership skips
//     it — that is what parser.ConceptIDs exists for. BackfillN is 0 by
//     default, so this is a no-op today and would be a silent overcount the
//     day it is not.
//   - Concept.Seed, never Concept.Anchor. An emergent concept's anchor decides
//     where it hangs in the taxonomy, not where it came from; counting anchored
//     concepts here would credit a seed with a practice it did not find.
//
// It takes the two slices rather than a Result so the counting rule can be
// tested on its own: a lexicon.Model is only constructible by lexicon.Build,
// and a corpus large enough to grow two concepts from one seed is not a unit
// test.
func seedYield(concepts []lexicon.Concept, assign [][]parser.Concept) map[string]int {
	seedOf := make(map[string]string)
	for _, c := range concepts {
		if c.Seed != "" {
			seedOf[c.ID] = c.Seed
		}
	}
	counts := make(map[string]int)
	for _, memberships := range assign {
		// A function counts once per seed however many of that seed's concepts
		// it carries, so the seeds are collected before anything is counted.
		var seen []string
		for _, m := range memberships {
			if m.BelowFloor {
				continue
			}
			s, ok := seedOf[m.ID]
			if !ok {
				continue
			}
			if !slices.Contains(seen, s) {
				seen = append(seen, s)
			}
		}
		for _, s := range seen {
			counts[s]++
		}
	}
	return counts
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
	ov.MisfitsExcused = cs.MisfitsExcused
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

// maxPracticeFeatures bounds how many things are listed per prototype channel.
// A prototype's calls channel can hold every call any member makes; the top few
// are the house style and the tail is noise.
const maxPracticeFeatures = 5

// minPracticeP and minPracticeLift decide what counts as house style.
//
// Presence alone does not: nearly every Go function has a return and an if, so
// a prototype unfiltered reports "533 of 533 error_wrapping functions return",
// which is a fact about Go rather than about this codebase. A feature has to be
// carried by a real share of the concept's members *and* carried more than the
// corpus at large carries it.
//
// The lift floor mirrors the ecology's own cutoff — twice the base rate, the
// same ln 2 that decides whether an association is worth reporting — so the two
// halves of this section agree on what "beyond chance" means. The presence
// floor is lower than the old half-the-members rule, because a feature in a
// third of a concept's members that is almost absent elsewhere is a sharper
// observation than one in two thirds that is everywhere.
const (
	minPracticeP    = 0.25
	minPracticeLift = 2.0
)

// buildPractice fills in how this corpus writes things, from the two models
// that had no caller outside their own tests.
//
// The prototypes say what a concept looks like here; the PMI ecology says what
// travels with what; Atypical says who is doing neither. Together they are the
// only part of doppel that describes the codebase rather than scoring it.
func buildPractice(ov *reporter.Overview, res Result) {
	if res.Culture == nil {
		return
	}
	practiceConcepts(ov, res)
	practiceMatrix(ov, res)
	practiceAssociations(ov, res)
	practiceDrift(ov, res)
}

// practiceConcepts renders each prototyped concept's feature distribution.
//
// Concepts are described biggest-first: a concept with thirty members says more
// about house style than one that barely cleared the modeling minimum.
func practiceConcepts(ov *reporter.Overview, res Result) {
	for _, row := range ov.Concepts { // already count desc, then tag
		if !row.Prototyped {
			continue
		}
		proto, ok := res.Culture.Prototype(row.Tag)
		if !ok {
			continue
		}
		cp := reporter.ConceptPractice{Tag: row.Tag, Members: row.Count}
		for _, ch := range proto.Channels {
			pc := reporter.PracticeChannel{Name: ch.Name, Weight: practiceWeight(ch.Name)}
			for _, f := range ch.Features { // sorted (P desc, Name asc) by culture
				base, ok := res.Culture.BaseRate(ch.Name, f.Name)
				if !ok || base <= 0 {
					continue
				}
				lift := f.P / base
				if f.P < minPracticeP || lift < minPracticeLift {
					continue
				}
				// P is an exact k/m over the concept's members, so rounding
				// recovers k without drift.
				pc.Features = append(pc.Features, reporter.PracticeFeature{
					Name:  f.Name,
					Count: int(math.Round(f.P * float64(row.Count))),
					P:     f.P,
					Lift:  lift,
				})
			}
			// Most distinctive first: the ordering culture supplies is by
			// prevalence, which is the thing the base rate exists to discount.
			sort.SliceStable(pc.Features, func(i, j int) bool {
				if pc.Features[i].Lift != pc.Features[j].Lift {
					return pc.Features[i].Lift > pc.Features[j].Lift
				}
				return pc.Features[i].Name < pc.Features[j].Name
			})
			if len(pc.Features) > maxPracticeFeatures {
				pc.Features = pc.Features[:maxPracticeFeatures]
			}
			if len(pc.Features) > 0 {
				cp.Channels = append(cp.Channels, pc)
			}
		}
		// A concept whose members do nothing the rest of the corpus does not is
		// itself the finding, and a silently omitted concept cannot say it.
		ov.Practice = append(ov.Practice, cp)
	}
}

// practiceWeight mirrors the prototype channel weights so the report can say
// which lines carry the argument. They live unexported in culture; duplicating
// four integers is cheaper than widening that package's API, but it does mean
// this table has to track it.
func practiceWeight(channel string) int {
	switch channel {
	case "calls":
		return 40
	case "flow":
		return 20
	case "cotags", "role":
		return 15
	case "package":
		return 10
	}
	return 0
}

// practiceAssociations groups the corpus's co-occurrence habits by kind and
// bounds each group on its own.
//
// Grouping is the point. There are far more call tokens than concepts, so a
// single PMI-ordered list is all tag~call — doppel's own report showed zero
// concept-to-concept associations, not because it has none but because none
// reached the cut against a few hundred call rows.
func practiceAssociations(ov *reporter.Overview, res Result) {
	pos := map[string][]reporter.AssocRow{}
	neg := map[string][]reporter.AssocRow{}
	for _, a := range res.Culture.Associations() {
		row := reporter.AssocRow{
			Kind:    a.Kind.String(),
			A:       a.A,
			B:       a.B,
			Count:   a.Count,
			AOf:     res.TagCounts[ontology.TermID(a.A)],
			BOf:     tagPopulation(res, a.Kind, a.B),
			Missing: a.Expected,
		}
		if math.IsInf(a.PMI, -1) {
			// Count 0 has no ratio to print; culture's contract is the word.
			row.Never = true
			neg[row.Kind] = append(neg[row.Kind], row)
			continue
		}
		row.Ratio = math.Exp(a.PMI)
		if a.PMI > 0 {
			pos[row.Kind] = append(pos[row.Kind], row)
		} else {
			neg[row.Kind] = append(neg[row.Kind], row)
		}
	}
	ov.Travels = assocGroups(pos, false)
	ov.Avoids = assocGroups(neg, true)
}

// tagPopulation returns B's own population when B is a tag, so a tag~tag row
// can state itself against whichever side is smaller. Roles and call tokens
// have populations too, but they are not what a reader is asking about.
func tagPopulation(res Result, kind culture.AssocKind, b string) int {
	if kind != culture.TagTag {
		return 0
	}
	return res.TagCounts[ontology.TermID(b)]
}

// assocKindOrder is the reading order: concepts first because they are the most
// interpretable, calls last because they are the most numerous.
var assocKindOrder = []string{"tag~tag", "tag~role", "tag~call"}

// maxAssocRows bounds each kind, in each direction, on its own budget.
const maxAssocRows = 6

// assocGroups ranks each kind and bounds it.
//
// Ranking is lift weighted by evidence, not lift alone. A 126x association on
// three functions outranked a 31x one on six under plain PMI order, which reads
// backwards: the second is the stronger finding. ln(1+count) is enough to fix
// the order without letting a weak-but-common relationship lead. This is a
// presentation key only — culture's own ordering contract is untouched.
//
// Absences rank ahead of merely-rare pairings, ordered by how many co-occurrences
// chance alone would have produced: the more expected, the louder the silence.
func assocGroups(byKind map[string][]reporter.AssocRow, negative bool) []reporter.AssocGroup {
	var out []reporter.AssocGroup
	for _, kind := range assocKindOrder { // fixed order, never map order
		rows := byKind[kind]
		if len(rows) == 0 {
			continue
		}
		sort.SliceStable(rows, func(i, j int) bool {
			a, b := rows[i], rows[j]
			if negative && a.Never != b.Never {
				return a.Never
			}
			if negative && a.Never && b.Never {
				if a.Missing != b.Missing {
					return a.Missing > b.Missing
				}
			} else if ka, kb := assocRank(a), assocRank(b); ka != kb {
				return ka > kb
			}
			if a.A != b.A {
				return a.A < b.A
			}
			return a.B < b.B
		})
		g := reporter.AssocGroup{Kind: kind, Rows: rows}
		if len(rows) > maxAssocRows {
			g.Rows, g.More = rows[:maxAssocRows], len(rows)-maxAssocRows
		}
		out = append(out, g)
	}
	return out
}

// assocRank is lift weighted by how many functions carry the finding.
func assocRank(a reporter.AssocRow) float64 {
	lift := a.Ratio
	if lift < 1 && lift > 0 {
		lift = 1 / lift // a negative association is strong when the ratio is small
	}
	return math.Log(lift) * math.Log(1+float64(a.Count))
}

// practiceMatrix builds the concept-to-concept co-occurrence grid.
//
// Unlike every other list in this section it is not a sample: the vocabulary is
// a fixed, small set of concrete concepts, so the grid is bounded by
// construction and can show every cell — including the blank ones, which are the ordinary company
// that a ranked list never has room to mention.
func practiceMatrix(ov *reporter.Overview, res Result) {
	tags := make([]string, 0, len(ov.Concepts))
	for _, c := range ov.Concepts {
		tags = append(tags, c.Tag)
	}
	if len(tags) < 2 {
		return
	}
	sort.Strings(tags)
	at := make(map[string]int, len(tags))
	for i, t := range tags {
		at[t] = i
	}

	cells := make([][]string, len(tags))
	for i := range cells {
		cells[i] = make([]string, len(tags))
	}
	for _, a := range res.Culture.Associations() {
		if a.Kind != culture.TagTag {
			continue
		}
		i, iok := at[a.A]
		j, jok := at[a.B]
		if !iok || !jok {
			continue
		}
		state := ""
		switch {
		case math.IsInf(a.PMI, -1):
			state = "never"
		case a.PMI < 0:
			state = "−"
		case math.Exp(a.PMI) >= 4:
			state = "++"
		case a.PMI > 0:
			state = "+"
		}
		// Symmetric: fill both, the renderer reads the lower triangle.
		cells[i][j], cells[j][i] = state, state
	}
	ov.Matrix = &reporter.ConceptMatrix{Tags: tags, Cells: cells}
}

// practiceDrift names the functions realizing a concept unlike their peers.
//
// The Unpaired bit is the point. A drifting function that appears in a reported
// pair already gets a culture note beside that pair; one that appears in no
// pair has been a stderr tally and nothing more — and it is the more
// interesting case, because nothing in the report explains it.
func practiceDrift(ov *reporter.Overview, res Result) {
	paired := make(map[int]bool, len(res.Pairs)*2)
	for _, p := range res.Pairs {
		paired[p.AIdx], paired[p.BIdx] = true, true
	}

	var rows []reporter.DriftRow
	for i, u := range res.Units {
		for _, tag := range parser.ConceptIDs(u.Concepts) { // ascending by ID, stable
			if !res.Culture.Atypical(i, tag) {
				continue
			}
			typ, _ := res.Culture.Typicality(i, tag)
			med, _ := res.Culture.Median(tag)
			rows = append(rows, reporter.DriftRow{
				Name:       concepter.QualifiedName(u),
				File:       snapshot.RelSlash(res.Root, u.File),
				Line:       u.StartLine,
				Tag:        tag,
				Typicality: typ,
				Median:     med,
				Unpaired:   !paired[i],
			})
		}
	}

	// Unexplained drift first, then the furthest from its concept's median.
	// Both keys are meaningful; the trailing name keeps the order total.
	sort.SliceStable(rows, func(a, b int) bool {
		x, y := rows[a], rows[b]
		if x.Unpaired != y.Unpaired {
			return x.Unpaired
		}
		if dx, dy := x.Median-x.Typicality, y.Median-y.Typicality; dx != dy {
			return dx > dy
		}
		if x.Name != y.Name {
			return x.Name < y.Name
		}
		return x.Tag < y.Tag
	})
	if len(rows) > maxDriftRows {
		ov.Drift, ov.DriftMore = rows[:maxDriftRows], len(rows)-maxDriftRows
		return
	}
	ov.Drift = rows
}

// maxDriftRows matches the renderer's table cap.
const maxDriftRows = 10

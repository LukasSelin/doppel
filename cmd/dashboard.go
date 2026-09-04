package cmd

import (
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/dashboard"
	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// The two payload budgets.
//
// Most of a payload scales with the number of findings, which is what a reader
// came for. Two parts scale with the corpus instead, and on a wide corpus they
// dominate the file: the source text of every function in a scored pair, and
// the per-pair prose evidence. Both are detail for *one* pair a reader has
// opened, so both are admitted in descending edge-rank order — what someone is
// most likely to open survives the bound — and both report what they dropped.
//
// Measured on moby (7 644 functions, 17 473 pairs): unbounded and unrounded,
// the page was 17.5 MB, of which Evidence.Reasons alone was 3.9 MB. That is
// the same bloat snapshot Schema 2 removed when it dropped Reasons — free-text
// English restating counts is expensive per byte of meaning.
const (
	maxBodyBytes   = 3 << 20
	maxDetailBytes = 2 << 20
)

// round4 trims a score to four decimal places.
//
// Float64 round-trips at 17 significant digits, and 0.7504423449238352 costs
// 18 bytes to say what the page renders as 0.75. Four places is far more
// resolution than any surface here shows and still separates any two scores a
// reader could act on. It is a pure function of its input, so nothing about
// determinism changes. On moby this alone is ~3 MB.
func round4(v float64) float64 { return math.Round(v*1e4) / 1e4 }

// buildDashboard assembles the page payload from a finished run.
//
// It is the only place presentation-adjacent decisions are made in Go, and it
// makes as few as it can: the payload carries raw scores, counts and
// identifiers, and the page's own assets decide what a colour or a radius
// means. The one thing computed here that could have been computed there is
// Edge.Rank — analyzer.RankKey is this repo's single definition of corroborated
// evidence, and a second one in JavaScript would drift from it silently.
func buildDashboard(res Result, ov *reporter.Overview, fams []family.Family,
	famStats family.Stats, reported []analyzer.SimilarPair, suppressed int) dashboard.Payload {

	p := dashboard.Payload{
		Schema: dashboard.Schema,
		Target: targetName(res.Root),
	}

	p.Units = dashboardUnits(res)
	p.Packages = dashboardPackages(res, p.Units)
	p.Concepts = dashboardConcepts(p.Units)

	// The page draws the full comparator-scored, struct-min-filtered set, not
	// the ranked one. Same reasoning as the family stage: --top and
	// --max-per-func are report-time devices, and a neighbourhood built on a
	// diversity-capped list would hide the very neighbour a reader clicked in
	// to find.
	p.Edges = dashboardEdges(res)
	p.Families = dashboardFamilies(res, fams)

	detailDropped := boundEdgeDetail(p.Edges)
	bodies, bodiesDropped := dashboardBodies(res, p.Edges)
	p.Bodies = bodies

	p.Facts = dashboardFacts(res, ov, famStats, fams, len(p.Edges), len(reported), suppressed)
	p.Facts.BodiesOmitted = bodiesDropped
	p.Facts.DetailOmitted = detailDropped
	return p
}

// boundEdgeDetail keeps the per-pair prose — the shared-structure chains and
// the comparator's reasons — for as many of the best-corroborated pairs as the
// budget allows, and strips it from the rest.
//
// Detail is what a reader sees after opening one pair, so bounding it by rank
// costs nothing on the pairs anyone opens and saves several megabytes on a wide
// corpus. Edges arrive rank-descending, so this is a single pass; the scores
// themselves are never dropped, only the explanation, and a stripped edge still
// draws on the map and still lists as a neighbour.
func boundEdgeDetail(edges []dashboard.Edge) int {
	spent, dropped := 0, 0
	for i := range edges {
		cost := 0
		for _, c := range edges[i].Chains {
			cost += len(c.Render) + 24
		}
		for _, r := range edges[i].Reasons {
			cost += len(r) + 4
		}
		if cost == 0 {
			continue
		}
		if spent+cost > maxDetailBytes {
			edges[i].Chains = nil
			edges[i].Reasons = nil
			dropped++
			continue
		}
		spent += cost
	}
	return dropped
}

// dashboardUnits converts every analysed function.
//
// Units[i].ID == i is the payload's whole identity scheme — Edge endpoints are
// positions into this slice, exactly as SimilarPair's AIdx/BIdx already are.
// A payload describes one run and never crosses runs, so it needs none of
// snapshot's name keying.
func dashboardUnits(res Result) []dashboard.Unit {
	out := make([]dashboard.Unit, len(res.Units))
	for i, u := range res.Units {
		qn := concepter.QualifiedName(u)
		d := dashboard.Unit{
			ID:      i,
			Key:     qn,
			Name:    u.Name,
			Package: u.Package,
			File:    snapshot.RelSlash(res.Root, u.File),
			Line:    u.StartLine,
			Nodes:   u.Fingerprint.Nodes,
			Fit:     -1,
			Test:    isTestUnit(u),
		}
		if i < len(res.Docs) {
			d.Role = res.Docs[i].Role
		}
		if res.Graph != nil {
			d.FanIn = len(res.Graph.Callers[qn])
			d.FanOut = len(res.Graph.Callees[qn])
		}
		d.Concepts = dashboardUnitConcepts(u.Concepts)
		d.Concept = dominantConcept(d.Concepts)
		if res.Culture != nil {
			if fit, ok := res.Culture.HabitatFit(i); ok {
				d.Fit = round4(fit)
			}
			d.Misfit = res.Culture.Misfit(i)
		}
		out[i] = d
	}
	return out
}

// dashboardUnitConcepts ranks a unit's memberships strongest first.
//
// The same order the text report's concept line uses, and for the same reason:
// the stored order is ascending by ID, which is right for a set and wrong for
// a reader — what carries the unit should lead. Uncapped, unlike the report
// line, because the page has room and the tail is one click away rather than a
// wall of text.
func dashboardUnitConcepts(cs []parser.Concept) []dashboard.UnitConcept {
	if len(cs) == 0 {
		return nil
	}
	ranked := append([]parser.Concept(nil), cs...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Confidence != ranked[j].Confidence {
			return ranked[i].Confidence > ranked[j].Confidence
		}
		return ranked[i].ID < ranked[j].ID
	})
	out := make([]dashboard.UnitConcept, len(ranked))
	for i, c := range ranked {
		out[i] = dashboard.UnitConcept{ID: c.ID, Confidence: c.Confidence}
	}
	return out
}

// dominantConcept is the map's colour channel: a unit's strongest membership.
//
// The arena's equilibrium was used here first, and is deliberately not used
// now. Its job was suppressing the fixed tagger's false positives — a unit
// tagged validation, db_access and mapping off fixture strings equilibrating to
// the one concept its surrounding evidence supports. A learned lexicon already
// does that job, with a confidence, at membership time. What the arena adds on
// top is invasion: a concept can win a function through an association without
// the function carrying it, which is a real finding and a bad colour. Measured
// on this repo it painted 111 functions with a concept only 5 of them carry,
// and a legend cannot honestly say "leads 111, carried by 5".
//
// So: the head of the ranked memberships, which keeps dominant a subset of
// carried by construction. Ties on ID, since two equal confidences must not let
// slice order decide a colour. A unit carrying nothing is simply uncoloured.
func dominantConcept(cs []dashboard.UnitConcept) string {
	if len(cs) == 0 {
		return ""
	}
	return cs[0].ID
}

// dashboardPackages aggregates the territories the map draws.
func dashboardPackages(res Result, units []dashboard.Unit) []dashboard.Package {
	counts := map[string]*dashboard.Package{}
	for _, u := range units {
		p := counts[u.Package]
		if p == nil {
			p = &dashboard.Package{Name: u.Package, Norm: -1}
			counts[u.Package] = p
		}
		p.Functions++
		if u.Misfit {
			p.Misfits++
		}
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names) // before anything reads the map, so its order never decides
	out := make([]dashboard.Package, 0, len(names))
	for _, name := range names {
		p := counts[name]
		if res.Culture != nil {
			if norm, ok := res.Culture.HabitatNorm(name); ok {
				p.Norm = round4(norm)
			}
		}
		out = append(out, *p)
	}
	return out
}

// dashboardConcepts is the legend: every learned concept in use, ranked by how
// many units it actually colours.
//
// Ranked rather than alphabetical because the vocabulary is learned and so its
// size is a property of the corpus — no palette can be assumed to cover it, and
// the page pools whatever falls off the end. Ties on ID, so an unchanged tree
// assigns the same colour to the same concept every run.
func dashboardConcepts(units []dashboard.Unit) []dashboard.ConceptRow {
	carried, dominant := map[string]int{}, map[string]int{}
	for _, u := range units {
		for _, c := range u.Concepts {
			carried[c.ID]++
		}
		if u.Concept != "" {
			dominant[u.Concept]++
			if _, ok := carried[u.Concept]; !ok {
				carried[u.Concept] = 0
			}
		}
	}
	ids := make([]string, 0, len(carried))
	for id := range carried {
		ids = append(ids, id)
	}
	sort.Strings(ids) // before the count comparison, so map order never decides
	out := make([]dashboard.ConceptRow, 0, len(ids))
	for _, id := range ids {
		out = append(out, dashboard.ConceptRow{ID: id, Carried: carried[id], Dominant: dominant[id]})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Dominant != out[j].Dominant {
			return out[i].Dominant > out[j].Dominant
		}
		if out[i].Carried != out[j].Carried {
			return out[i].Carried > out[j].Carried
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// dashboardEdges converts every scored pair, ranked.
func dashboardEdges(res Result) []dashboard.Edge {
	opts := analyzer.DefaultRankOptions()
	out := make([]dashboard.Edge, 0, len(res.Pairs))
	for _, pair := range res.Pairs {
		a, b := pair.AIdx, pair.BIdx
		if a > b {
			a, b = b, a
		}
		e := dashboard.Edge{
			A:           a,
			B:           b,
			Shape:       round4(pair.Score),
			Rank:        round4(analyzer.RankKey(pair, opts)),
			Merge:       pair.MergeWorthy(),
			Cross:       pair.A.Package != pair.B.Package,
			Containment: round4(pair.Breakdown.Containment),
			Explain:     pair.Explain,
			Breakdown: [6]float64{
				round4(pair.Breakdown.WL), round4(pair.Breakdown.Flow), round4(pair.Breakdown.Depth),
				round4(pair.Breakdown.Signature), round4(pair.Breakdown.SizeRatio),
				round4(pair.Breakdown.Containment),
			},
		}
		e.Views = [5]float64{0, 0, -1, -1, -1}
		if pair.Evidence != nil {
			e.Overlap = round4(pair.Evidence.OverlapScore)
			e.Reasons = pair.Evidence.Reasons
			v := pair.Evidence.Views
			e.Views[0], e.Views[1] = round4(v.Shape), round4(v.Corpus)
			if v.HasFeature {
				e.Views[2], e.Views[3], e.Views[4] = round4(v.Feature), round4(v.AInB), round4(v.BInA)
				e.ViewsDisagree = v.Disagree
				for _, f := range v.SharedVocabulary {
					e.SharedVocab = append(e.SharedVocab, f.Name)
				}
			}
		}
		if pair.Retrieval != nil {
			e.Total = round4(pair.Retrieval.Total)
			e.Trophic = round4(pair.Retrieval.TrophicSim)
			e.CallSim = round4(pair.Retrieval.CallSim)
			e.Channels = pair.Retrieval.Channels
			for _, c := range pair.Retrieval.Chains {
				e.Chains = append(e.Chains, dashboard.Chain{
					Depth: c.Depth, Energy: round4(c.Energy), Render: c.Render,
				})
			}
		}
		if pair.Kind != nil {
			e.Kind = pair.Kind.Kind
			e.KindNote = reporter.KindClause(pair.Kind)
		}
		out = append(out, e)
	}
	// Rank descending, so every per-unit neighbour list in the page inherits
	// the order without a second sort. Ties break on the endpoints, which is
	// what keeps an unchanged tree byte-identical.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Rank != out[j].Rank {
			return out[i].Rank > out[j].Rank
		}
		if out[i].A != out[j].A {
			return out[i].A < out[j].A
		}
		return out[i].B < out[j].B
	})
	return out
}

// dashboardBodies inlines the source of the functions a reader is most likely
// to open, and says how many it left out.
//
// Admission follows the edge order, which is rank order, so the bound falls on
// the least corroborated pairs first. It is deterministic for the same reason
// the ranking is.
func dashboardBodies(res Result, edges []dashboard.Edge) ([]dashboard.Body, int) {
	wanted := map[int]bool{}
	var order []int
	admit := func(idx int) {
		if idx < 0 || idx >= len(res.Units) || wanted[idx] {
			return
		}
		wanted[idx] = true
		order = append(order, idx)
	}
	for _, e := range edges {
		admit(e.A)
		admit(e.B)
	}

	out := make([]dashboard.Body, 0, len(order))
	spent, omitted := 0, 0
	kept := map[int]bool{}
	for _, idx := range order {
		text := res.Units[idx].Body
		if text == "" {
			omitted++
			continue
		}
		if spent+len(text) > maxBodyBytes {
			omitted++
			continue
		}
		spent += len(text)
		kept[idx] = true
	}
	// Emitted in unit order rather than admission order: the payload is
	// looked up by unit id, and a sorted slice is one less thing that can
	// vary between runs.
	ids := make([]int, 0, len(kept))
	for idx := range kept {
		ids = append(ids, idx)
	}
	sort.Ints(ids)
	for _, idx := range ids {
		out = append(out, dashboard.Body{Unit: idx, Text: res.Units[idx].Body})
	}
	return out, omitted
}

func dashboardFamilies(res Result, fams []family.Family) []dashboard.Family {
	out := make([]dashboard.Family, 0, len(fams))
	for _, f := range fams {
		d := dashboard.Family{
			Members:  f.Members,
			MinEdge:  round4(f.MinEdge),
			MeanEdge: round4(f.MeanEdge),
			Evidence: round4(f.Evidence),
			Added:    f.Completed,
			Tag:      dominantTag(res, f),
		}
		if f.Kind != nil {
			d.Kind = f.Kind.Kind
		}
		out = append(out, d)
	}
	return out
}

// dashboardFacts is the run's own header. Everything except the pair counts
// comes off the Overview the markdown report already builds, so this is a
// second consumer of that model rather than a second computation of it.
func dashboardFacts(res Result, ov *reporter.Overview, famStats family.Stats,
	fams []family.Family, edges, reported, suppressed int) dashboard.Facts {

	f := dashboard.Facts{
		Functions:     len(res.Units),
		Threshold:     res.Params.Threshold,
		StructMin:     res.Params.StructMin,
		TestsMode:     testsWord(res.Params.TestsMode),
		Generated:     res.Params.Generated,
		Calibrate:     res.Params.Calibrate,
		Debug:         res.Params.Debug,
		Pairs:         edges,
		Reported:      reported,
		Suppressed:    suppressed,
		FamilyCount:   famStats.Families,
		FamilyFuncs:   famStats.Members,
		FamilyLargest: largestFamily(fams),
		EdgesAdded:    famStats.Completed,

		// From Result directly, not from the Overview: these are corpus
		// statistics index() computed, and they exist on a run whose caller
		// asked for no overview at all.
		Compression: round4(res.ConsStats.Ratio()),
		NNScored:    res.NN.Scored,
		NNP50:       round4(res.NN.P50),
		NNP90:       round4(res.NN.P90),
	}
	if ov == nil {
		return f
	}
	f.Packages = ov.Packages
	f.CandidatePairs = ov.UnionPairs
	f.ShapePairs = ov.ShapePairs
	f.ConceptPairs = ov.ConceptPairs
	f.CallPairs = ov.CallPairs
	f.OnlyConcept = ov.OnlyConceptPairs
	f.OnlyCall = ov.OnlyCallPairs
	f.Misfits = ov.Misfits
	f.MisfitsExcused = ov.MisfitsExcused
	f.ArenaProfiled = ov.ArenaProfiled
	f.ArenaDominance = ov.ArenaDominance
	f.ArenaCoalition = ov.ArenaCoalition
	f.ArenaConflict = ov.ArenaConflict
	return f
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

func largestFamily(fams []family.Family) int {
	n := 0
	for _, f := range fams {
		if len(f.Members) > n {
			n = len(f.Members)
		}
	}
	return n
}

// dominantTag is the concept most of a family's members carry, or "" when they
// share none. Membership is the graded fact's boolean view here deliberately:
// the question is how many members share a concept, not how hard any one of
// them means it. Ties break on the name so the column is stable.
func dominantTag(res Result, f family.Family) string {
	counts := map[string]int{}
	for _, m := range f.Members {
		if m < 0 || m >= len(res.Units) {
			continue
		}
		for _, c := range res.Units[m].Concepts {
			counts[c.ID]++
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

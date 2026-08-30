package reporter

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Overview is what doppel understands about a corpus, as plain presorted data.
//
// The report used to open with three numbers — functions, threshold, pairs —
// and then hand the reader a list of matches. Everything doppel had actually
// learned about the codebase went to stderr and died there: the concept
// vocabulary it found, which concepts are absent, how uniform each package is,
// how settled each function's concept profile is, which retrieval channel
// found the candidates. A reader needs some of that to weigh the list they are
// being shown, and the strongest evidence is that examples/ already pastes the
// stderr block into every committed report to compensate.
//
// This type is deliberately dumb. Every field arrives sorted and rendered-ready
// so that reporter never learns about culture, ontology or the call graph — cmd
// queries those and fills this in. The zero value renders nothing.
type Overview struct {
	Root       string
	Functions  int
	Packages   int
	TestsMode  string
	Threshold  float64
	Suppressed int // pairs dropped by the per-function diversity cap

	Concepts []TagRow // concepts present, count desc then tag
	Absent   []string // seed concepts this corpus grew no practice for
	Roles    []RoleRow

	Taxonomy []TaxonomyNode // the concept tree, parents before children

	Habitats     []HabitatRow // packages with a habitat model, norm asc (worst first)
	HabitatsMore int          // habitats beyond the rendered bound
	Misfits      int
	// MisfitsExcused are package misfits that match their wider subsystem and so
	// are not reported. Counting them separately keeps "0 misfits" honest: it
	// can mean nothing was odd, or that everything odd was odd in company.
	MisfitsExcused int

	MostUniform, MostDiverse         string
	MostUniformNorm, MostDiverseNorm float64
	Strongest, Loosest               string
	StrongestStrength, LoosestC      float64

	ArenaProfiled, ArenaDominance   int
	ArenaCoalition, ArenaConflict   int
	ArenaWeak                       int
	ConceptsModeled, Unusual        int
	ShapePairs, ConceptPairs        int
	CallPairs, UnionPairs           int
	OnlyConceptPairs, OnlyCallPairs int

	Links     []PackageLink // merge-worthy duplication between packages
	LinksMore int
	SelfDup   map[string]int // package -> merge-worthy pairs wholly inside it

	// Local practice: not what the corpus contains, but how it writes things.
	Practice  []ConceptPractice // how each modeled concept is realized here
	Matrix    *ConceptMatrix    // concept-to-concept co-occurrence, whole and bounded
	Travels   []AssocGroup      // co-occur beyond chance, grouped by kind
	Avoids    []AssocGroup      // co-occur far less than chance, grouped by kind
	Drift     []DriftRow        // functions realizing a concept unlike their peers
	DriftMore int
}

// AssocGroup is one kind of association, bounded on its own.
//
// Grouping is not cosmetic. There are far more call tokens than concepts, so on
// a single PMI-ordered list the tag~call rows crowd out every tag~tag row —
// doppel's own report showed zero concept-to-concept associations, not because
// it has none but because none reached the cut. Each kind now gets its own
// budget.
type AssocGroup struct {
	Kind string // tag~tag, tag~role, tag~call
	Rows []AssocRow
	More int
}

// ConceptMatrix is the whole concept-to-concept co-occurrence structure.
//
// It is bounded by construction — the vocabulary is a fixed, small set of
// concrete concepts and a corpus uses some subset — so unlike every other list
// here it can show everything rather than a strongest-first sample. Widening the
// taxonomy widens this table; it does not turn it into a sample. Cells[i][j] is meaningful
// for i > j: the lower triangle, since co-occurrence is symmetric.
type ConceptMatrix struct {
	Tags  []string
	Cells [][]string // "", "+", "++", "-", "never"
}

// ConceptPractice is how this corpus normally realizes one concept — the
// prototype, rendered. It is the most direct answer doppel can give to "what
// does a transaction look like around here".
type ConceptPractice struct {
	Tag      string
	Members  int
	Channels []PracticeChannel
}

// PracticeChannel is one evidence channel of a prototype, with the weight it
// carries so a reader knows which lines to believe.
type PracticeChannel struct {
	Name     string
	Weight   int // percent of the typicality score
	Features []PracticeFeature
}

// PracticeFeature is one thing members of a concept do, and how many of them
// do it.
type PracticeFeature struct {
	Name string
	// Count over P: "4 of 6" is honest where "67%" is false precision on six
	// members. P is kept for the bar, which needs the fraction.
	Count int
	P     float64
	// Lift is how much more this concept's members carry the feature than the
	// corpus at large does. It is what separates house style from the ambient
	// properties of Go — a return in every member is lift 1 and says nothing.
	Lift float64
}

// AssocRow is one corpus association: two features that co-occur far more, or
// far less, than chance would give.
type AssocRow struct {
	Kind  string // tag~tag, tag~role, tag~call
	A, B  string
	Count int
	// AOf is how many functions carry A at all, which turns a lift into a
	// conditional: "13 of 14 http_call functions call NewRequest" is what a
	// reader acts on, where "416x chance" only says why it is notable.
	// A is always the tag — see culture's buildAssociations.
	AOf int
	// BOf is B's own population when B is a tag too. For a tag~tag pair either
	// side could be the denominator, and the smaller one is the sharper
	// statement: "16 of 33 retry functions" beats "16 of 436 concurrency
	// functions" for the same fact. Zero when B is a role or a call token.
	BOf     int
	Ratio   float64 // exp(PMI): how many times chance
	Never   bool    // the two never co-occur here at all
	Missing float64 // expected count, for the negative case
}

// DriftRow is one function that carries a concept but realizes it unlike
// every other member.
type DriftRow struct {
	Name       string
	File       string
	Line       int
	Tag        string
	Typicality float64
	Median     float64
	Unpaired   bool // in no reported pair: drift with no duplicate to explain it
}

// TagRow is one concept the corpus uses, with how settled its practice is.
type TagRow struct {
	Tag        string
	Count      int
	Convention float64 // 0 when the concept has too few members to prototype
	Prototyped bool
}

// RoleRow is one structural role and how many functions hold it.
type RoleRow struct {
	Role  string
	Count int
}

// TaxonomyNode is one concept term positioned in the tree, flattened so the
// renderer needs no recursion and the order cannot depend on a map.
type TaxonomyNode struct {
	ID       string
	Parent   string
	Abstract bool
	Count    int // leaf occurrences in this corpus; 0 for abstract nodes
}

// HabitatRow is one package doppel modeled as a habitat.
type HabitatRow struct {
	Package     string
	Functions   int
	Norm        float64 // mean member fit: how uniform the package is
	Temperature float64 // median member strain: how much deviation it tolerates
	Misfits     int
}

// PackageLink is duplication pressure between two packages.
type PackageLink struct {
	A, B  string // A < B
	Pairs int    // merge-worthy pairs with one side in each
}

// maxOverviewNodes bounds every package-scoped diagram. Past a dozen or so
// nodes a mermaid graph stops being a picture and becomes a hairball, and moby
// has 168 habitats. What is dropped is always counted in the prose.
const maxOverviewNodes = 12

// PrintMarkdownOverview writes the corpus overview, or nothing for a zero
// Overview.
//
// Rendering nothing for an empty overview is what keeps every existing report
// byte-identical: a caller that does not build one gets exactly the document
// it got before this section existed.
func PrintMarkdownOverview(w io.Writer, ov *Overview) {
	if ov == nil || ov.Functions == 0 {
		return
	}
	fmt.Fprintf(w, "## What doppel sees\n\n")
	fmt.Fprintf(w, "%s\n\n", corpusSentence(ov))

	overviewConcepts(w, ov)
	overviewDuplication(w, ov)
	overviewHabitats(w, ov)
	overviewRetrieval(w, ov)

	fmt.Fprintf(w, "---\n\n")
}

// corpusSentence states what was measured before anything is claimed about it.
func corpusSentence(ov *Overview) string {
	pop := "test functions excluded"
	switch ov.TestsMode {
	case "only":
		pop = "test functions only"
	case "include":
		pop = "tests and production together"
	}
	s := fmt.Sprintf("**%d functions** across **%d packages** — %s.", ov.Functions, ov.Packages, pop)
	if len(ov.Roles) > 0 {
		parts := make([]string, 0, len(ov.Roles))
		for _, r := range ov.Roles {
			parts = append(parts, fmt.Sprintf("%d %s", r.Count, r.Role))
		}
		s += " Structural roles: " + strings.Join(parts, ", ") + "."
	}
	return s
}

// overviewConcepts renders the vocabulary doppel reasons in and where this
// corpus sits inside it.
//
// The absent line is the one most likely to change a decision and the cheapest
// to compute — "nothing in this corpus is tagged retry" is a complete answer,
// where the list of present tags only narrows a search. It is the same fact the
// session-start hook digest leads with.
func overviewConcepts(w io.Writer, ov *Overview) {
	if len(ov.Taxonomy) == 0 && len(ov.Concepts) == 0 {
		return
	}
	fmt.Fprintf(w, "### Concepts\n\n")
	fmt.Fprintf(w, "These concepts were **learned from this corpus**, not read off a fixed list: "+
		"each one is a group of functions that share a way of being written, named after the "+
		"evidence that identified it. They hang from an authored interior, so two functions "+
		"under the same *branch* score partial credit rather than nothing. "+
		"Counts below are members; membership is graded, and a function can carry several.\n\n")

	if len(ov.Taxonomy) > 0 {
		fmt.Fprintf(w, "```mermaid\nflowchart LR\n")
		idOf := make(map[string]string, len(ov.Taxonomy))
		for i, n := range ov.Taxonomy {
			idOf[n.ID] = mermaidID("c", i)
		}
		for _, n := range ov.Taxonomy {
			id := idOf[n.ID]
			switch {
			case n.Abstract:
				fmt.Fprintf(w, "    %s([\"%s\"])\n", id, mermaidLabel(n.ID))
			case n.Count == 0:
				fmt.Fprintf(w, "    %s[\"%s<br/>absent\"]\n", id, mermaidLabel(n.ID))
			default:
				fmt.Fprintf(w, "    %s[\"%s<br/>%d\"]\n", id, mermaidLabel(n.ID), n.Count)
			}
		}
		for i, n := range ov.Taxonomy {
			if n.Parent == "" {
				continue
			}
			if pid, ok := idOf[n.Parent]; ok {
				fmt.Fprintf(w, "    %s --> %s\n", pid, mermaidID("c", i))
			}
		}
		// Absent leaves are the finding, so they get the one colour here.
		var hot []string
		for i, n := range ov.Taxonomy {
			if !n.Abstract && n.Count == 0 {
				hot = append(hot, mermaidID("c", i))
			}
		}
		if len(hot) > 0 {
			fmt.Fprint(w, mermaidClassDefs)
			fmt.Fprintf(w, "    class %s hot\n", strings.Join(hot, ","))
		}
		fmt.Fprintf(w, "```\n\n")
	}

	if len(ov.Absent) > 0 {
		fmt.Fprintf(w, "**No practice here for** `%s`. ", strings.Join(ov.Absent, "`, `"))
		fmt.Fprintf(w, "Concepts are learned from this corpus, so one can never be absent — it exists because functions carry it. These are the *seeds* the search started from that grew nothing: a direct answer to \"does this codebase already do X\".\n\n")
	}

	if len(ov.Concepts) > 0 {
		fmt.Fprintf(w, "| Concept | Functions | Convention |\n|---|---:|---|\n")
		for _, c := range ov.Concepts {
			conv := "—"
			if c.Prototyped {
				conv = fmt.Sprintf("`%.2f` %s", c.Convention, conventionWord(c.Convention))
			}
			fmt.Fprintf(w, "| `%s` | %d | %s |\n", mdEscape(c.Tag), c.Count, conv)
		}
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Convention is how uniformly this corpus realizes a concept: `1.00` means every "+
			"function carrying the tag does it the same way, and a low number means the tag covers "+
			"several unrelated habits. A concept with fewer than five members is not modeled.\n\n")
	}
}

func conventionWord(v float64) string {
	switch {
	case v >= 0.75:
		return "(unanimous)"
	case v >= 0.5:
		return "(settled)"
	default:
		return "(loose)"
	}
}

// overviewDuplication draws where the merge-worthy duplication actually sits.
//
// The pair list answers "which two functions"; nothing answers "which parts of
// the system keep growing the same code". A cross-package edge is the finding —
// two packages that both implement the same thing — while an intra-package
// count is usually a family of siblings and much less interesting.
func overviewDuplication(w io.Writer, ov *Overview) {
	if len(ov.Links) == 0 && len(ov.SelfDup) == 0 {
		return
	}
	fmt.Fprintf(w, "### Where the duplication is\n\n")
	fmt.Fprintf(w, "Merge-worthy pairs folded up to their packages. An edge means two packages keep "+
		"solving the same problem separately; a count on a node means the repetition is inside one package.\n\n")

	if len(ov.Links) > 0 {
		fmt.Fprintf(w, "```mermaid\nflowchart LR\n")
		seen := make(map[string]string)
		next := 0
		id := func(pkg string) string {
			if v, ok := seen[pkg]; ok {
				return v
			}
			v := mermaidID("p", next)
			next++
			seen[pkg] = v
			// Escape the identifier, then compose: <br/> is markup this code
			// emits, not content, and escaping the finished string would turn
			// the line break into a visible #lt;br/#gt;.
			label := mermaidLabel(pkg)
			if n := ov.SelfDup[pkg]; n > 0 {
				label += fmt.Sprintf("<br/>%d internal", n)
			}
			fmt.Fprintf(w, "    %s[\"%s\"]\n", v, label)
			return v
		}
		for _, l := range ov.Links {
			a, b := id(l.A), id(l.B)
			fmt.Fprintf(w, "    %s ---|\"%d\"| %s\n", a, l.Pairs, b)
		}
		fmt.Fprintf(w, "```\n\n")
		if ov.LinksMore > 0 {
			fmt.Fprintf(w, "_%d further package pairs are connected by merge-worthy duplication and are not drawn._\n\n", ov.LinksMore)
		}
	}
}

// overviewHabitats renders how uniform each package's practice is.
//
// Worst first, deliberately: a package whose members barely resemble each other
// is where a reader's attention is worth spending, and the uniform ones are
// only interesting as the contrast that makes the number mean something.
func overviewHabitats(w io.Writer, ov *Overview) {
	if len(ov.Habitats) == 0 {
		return
	}
	fmt.Fprintf(w, "### How settled each package is\n\n")
	fmt.Fprintf(w, "A package with at least five functions gets a habitat model: doppel learns what is "+
		"normal there and measures how surprising each member is against it. **Norm** is how uniform "+
		"the package's practice is. A **misfit** is a function alien to its package *and* to the "+
		"wider subsystem around it — one that fits its neighbours a directory up is normal for this "+
		"codebase and is not reported.\n\n")

	fmt.Fprintf(w, "```mermaid\nflowchart TD\n")
	for i, h := range ov.Habitats {
		// Escaped part first, then composed — see the note in overviewDuplication.
		label := fmt.Sprintf("%s<br/>%d functions · norm %.2f", mermaidLabel(h.Package), h.Functions, h.Norm)
		if h.Misfits > 0 {
			label += fmt.Sprintf("<br/>%d misfit", h.Misfits)
			if h.Misfits > 1 {
				label += "s"
			}
		}
		fmt.Fprintf(w, "    %s[\"%s\"]\n", mermaidID("h", i), label)
	}
	fmt.Fprint(w, mermaidClassDefs)
	byClass := map[string][]string{}
	for i, h := range ov.Habitats {
		c := heatClass(h.Norm)
		byClass[c] = append(byClass[c], mermaidID("h", i))
	}
	for _, c := range []string{"good", "warn", "hot"} { // fixed order, never map order
		if ids := byClass[c]; len(ids) > 0 {
			fmt.Fprintf(w, "    class %s %s\n", strings.Join(ids, ","), c)
		}
	}
	fmt.Fprintf(w, "```\n\n")

	// Joined rather than concatenated: each clause is optional, and appending
	// them with their own trailing spaces left a double space wherever one in
	// the middle was skipped.
	var tail []string
	if ov.HabitatsMore > 0 {
		tail = append(tail, fmt.Sprintf("_%d further packages are modeled and not drawn._", ov.HabitatsMore))
	}
	if ov.MostUniform != "" {
		tail = append(tail, fmt.Sprintf("Most uniform is `%s` (norm `%.2f`); most varied is `%s` (norm `%.2f`).",
			mdEscape(ov.MostUniform), ov.MostUniformNorm, mdEscape(ov.MostDiverse), ov.MostDiverseNorm))
	}
	if ov.Misfits > 0 {
		tail = append(tail, fmt.Sprintf("%d functions are alien to their package and to the subsystem around it.", ov.Misfits))
	}
	// The excused count earns a clause of its own: a corpus with no misfits and
	// seven excused ones is a different codebase from one with neither, and a
	// bare "0" cannot tell them apart.
	if ov.MisfitsExcused > 0 {
		tail = append(tail, fmt.Sprintf("A further %d fit poorly in their package but match the wider subsystem, so they are not reported.", ov.MisfitsExcused))
	}
	fmt.Fprint(w, strings.Join(tail, " "))
	fmt.Fprintf(w, "\n\n")
}

// overviewRetrieval explains why this pair list and not some other one.
//
// Recall is bounded by the three channels: a pair sharing no rare structure, no
// concept and no resolved call is never compared, however alike it is. A reader
// weighing the report is entitled to know which channel did the work.
func overviewRetrieval(w io.Writer, ov *Overview) {
	if ov.UnionPairs == 0 && ov.ConceptsModeled == 0 {
		return
	}
	fmt.Fprintf(w, "### How these candidates were found\n\n")
	if ov.UnionPairs > 0 {
		pct := func(n int) float64 { return 100 * float64(n) / float64(ov.UnionPairs) }
		fmt.Fprintf(w, "Three channels propose candidates independently — shared rare *structure*, shared "+
			"*concepts*, shared *calls* — and their union is what gets compared. This run: "+
			"**%d candidate pairs** (shape %d, concept %d, call %d), of which %.0f%% arrived on call "+
			"evidence alone and %.0f%% on concept evidence alone. A pair sharing none of the three is "+
			"never compared, however alike it looks.\n\n",
			ov.UnionPairs, ov.ShapePairs, ov.ConceptPairs, ov.CallPairs,
			pct(ov.OnlyCallPairs), pct(ov.OnlyConceptPairs))
	}
	if ov.ArenaProfiled > 0 {
		fmt.Fprintf(w, "Each function is also an arena where its candidate concepts compete for its evidence. "+
			"%d functions reached an equilibrium: **%d** settled on a single concept, **%d** on a coalition, "+
			"**%d** hold concepts this corpus says do not go together.\n\n",
			ov.ArenaProfiled, ov.ArenaDominance, ov.ArenaCoalition, ov.ArenaConflict)
	}
	if ov.Suppressed > 0 {
		fmt.Fprintf(w, "_%d further pairs were held back so no single function fills the report._\n\n", ov.Suppressed)
	}
}

// SortHabitats orders habitat rows worst-norm first and bounds them, returning
// the count dropped. Exported so cmd can bound before rendering without
// duplicating the ordering rule.
func SortHabitats(rows []HabitatRow) ([]HabitatRow, int) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Norm != rows[j].Norm {
			return rows[i].Norm < rows[j].Norm
		}
		return rows[i].Package < rows[j].Package
	})
	if len(rows) > maxOverviewNodes {
		return rows[:maxOverviewNodes], len(rows) - maxOverviewNodes
	}
	return rows, 0
}

// SortLinks orders package duplication links by weight and bounds them.
func SortLinks(links []PackageLink) ([]PackageLink, int) {
	sort.SliceStable(links, func(i, j int) bool {
		if links[i].Pairs != links[j].Pairs {
			return links[i].Pairs > links[j].Pairs
		}
		if links[i].A != links[j].A {
			return links[i].A < links[j].A
		}
		return links[i].B < links[j].B
	})
	if len(links) > maxOverviewNodes {
		return links[:maxOverviewNodes], len(links) - maxOverviewNodes
	}
	return links, 0
}

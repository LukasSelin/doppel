// Package dashboard renders one analysis run as a self-contained HTML page.
//
// The split from the report it replaced is the point. The old renderer carried
// a flat struct of render-ready percentages and label strings, so every visual
// decision lived in Go and adding one meant editing a template literal, a
// struct and an assembler. Here Go emits a semantic payload — raw scores,
// counts and identifiers — and the page's own assets decide what a bar, a
// colour or a node radius means. Iterating on the visuals is an asset edit.
package dashboard

// Schema is the payload version.
//
// It exists for the page, not for comparability: a stale asset bundle reading a
// newer payload should say so rather than render half a screen. Nothing diffs
// two payloads, and nothing should — that is snapshot.Schema's job.
const Schema = 1

// Payload is one analysis run, as the page receives it.
//
// Two rules hold it together, both inherited from snapshot for the same reason:
//
//   - No maps, anywhere. Map iteration order would make the page bytes vary
//     between runs of an unchanged tree, and a nondeterministic report is a bug
//     in this tool. TestPayloadHasNoMaps enforces it by reflection.
//   - Identity is the unit's index, not its name. A payload describes exactly
//     one run and never crosses runs, so it needs none of snapshot's name
//     keying — Edge endpoints are positions into Units, exactly as
//     analyzer.SimilarPair's AIdx/BIdx already are.
type Payload struct {
	Schema int    `json:"schema"`
	Target string `json:"target"` // the corpus's own directory name
	Facts  Facts  `json:"facts"`

	Packages []Package    `json:"packages"` // by name
	Units    []Unit       `json:"units"`    // Units[i].ID == i
	Edges    []Edge       `json:"edges"`    // by rank desc, then A, then B
	Bodies   []Body       `json:"bodies"`   // by unit ID; bounded, see Facts.BodiesOmitted
	Families []Family     `json:"families"` // as family.Build ordered them
	Concepts []ConceptRow `json:"concepts"` // the legend, most-carried first
}

// ConceptRow is one learned concept, with the two counts the legend needs.
//
// A learned vocabulary is not a fixed fourteen: its size is a property of the
// corpus, so the page cannot assume a palette covers it. Ranking by Dominant
// lets it colour the concepts that actually decide a territory and pool the
// long tail, rather than cycling a palette until two unrelated concepts share
// a hue.
type ConceptRow struct {
	ID       string `json:"id"`
	Carried  int    `json:"carried"`  // units carrying it at any confidence
	Dominant int    `json:"dominant"` // units it is the strongest concept for
}

// Facts is the run's own header: what was analysed, under what settings, and
// how the candidate set was found. Numbers only — the page formats them.
type Facts struct {
	Functions int     `json:"functions"`
	Packages  int     `json:"packages"`
	Threshold float64 `json:"threshold"`
	StructMin float64 `json:"structMin"`
	TestsMode string  `json:"testsMode"`
	Generated string  `json:"generated"`
	Calibrate float64 `json:"calibrate"` // 0 when thresholds were fixed
	Debug     bool    `json:"debug"`     // --debug widens Chains from 3 to 20

	// Pairs is what the page actually draws: the full comparator-scored,
	// struct-min-filtered set. Reported is what the text report showed after
	// --top and --max-per-func, and Suppressed is what that cap held back.
	// The page draws the uncapped set deliberately — a neighbourhood built on
	// a diversity-capped list would hide the neighbour a reader clicked in to
	// find — so it states the difference rather than implying there is none.
	Pairs      int `json:"pairs"`
	Reported   int `json:"reported"`
	Suppressed int `json:"suppressed"`

	CandidatePairs int `json:"candidatePairs"` // the retrieval union, pre-filter
	ShapePairs     int `json:"shapePairs"`
	ConceptPairs   int `json:"conceptPairs"`
	CallPairs      int `json:"callPairs"`
	OnlyConcept    int `json:"onlyConcept"`
	OnlyCall       int `json:"onlyCall"`

	FamilyCount   int `json:"familyCount"`
	FamilyFuncs   int `json:"familyFuncs"`
	FamilyLargest int `json:"familyLargest"`
	EdgesAdded    int `json:"edgesAdded"` // family edges supplied by completion

	Misfits        int `json:"misfits"`
	MisfitsExcused int `json:"misfitsExcused"`

	// The two corpus-health numbers, carried on the page for the same reason
	// they are carried in the markdown preamble and the JSON snapshot: they
	// describe the corpus the findings were drawn from, and a reader weighing
	// a pair list is entitled to know how repetitive the corpus is at all.
	// Neither ever moved a pair, a score or a rank.
	//
	// Compression is canonical AST nodes over distinct subtree shapes — 1.0
	// would be a corpus with no repeated structure anywhere. NNP50/NNP90 are
	// nearest-rank percentiles of each function's best code-shape score, over
	// the NNScored functions retrieval actually paired: a recall-bounded
	// population, not an exhaustive nearest-neighbour search, which is why
	// NNScored is carried beside them rather than assumed to be Functions.
	Compression float64 `json:"compression"`
	NNScored    int     `json:"nnScored"`
	NNP50       float64 `json:"nnP50"`
	NNP90       float64 `json:"nnP90"`

	ArenaProfiled  int `json:"arenaProfiled"`
	ArenaDominance int `json:"arenaDominance"`
	ArenaCoalition int `json:"arenaCoalition"`
	ArenaConflict  int `json:"arenaConflict"`

	// BodiesOmitted and DetailOmitted are the two payload bounds: how many
	// edge-participating functions had their source dropped, and how many
	// pairs lost their chains and reasons. Both fall on the least corroborated
	// pairs first, and both are reported rather than silent — like every other
	// bound in this tool.
	BodiesOmitted int `json:"bodiesOmitted"`
	DetailOmitted int `json:"detailOmitted"`
}

// Package is one Go package, drawn as a territory on the map.
type Package struct {
	Name      string  `json:"name"`
	Functions int     `json:"functions"`
	Misfits   int     `json:"misfits"`
	Norm      float64 `json:"norm"` // habitat norm; -1 when the package has no model
}

// Unit is one function or method.
type Unit struct {
	ID      int    `json:"id"` // == its index in Payload.Units
	Key     string `json:"key"`
	Name    string `json:"name"`
	Package string `json:"package"`
	File    string `json:"file"` // relative to the analysis root, slash-separated
	Line    int    `json:"line"`
	Role    string `json:"role"`

	// Concepts are the unit's learned memberships, strongest first — the same
	// order and the same graded fact the text report prints, uncapped here
	// because the page can afford to show what a line cannot.
	Concepts []UnitConcept `json:"concepts,omitempty"`

	// Concept is the map's colour channel: the head of Concepts, so it is
	// always one the unit actually carries. The map tints a region by the
	// concept most of its members carry, and the legend counts the same field.
	Concept string `json:"concept,omitempty"`

	FanIn  int `json:"fanIn"`  // resolved callers
	FanOut int `json:"fanOut"` // resolved internal callees
	Nodes  int `json:"nodes"`  // fingerprint node count

	Fit    float64 `json:"fit"` // habitat fit; -1 when the package has no model
	Misfit bool    `json:"misfit"`
	Test   bool    `json:"test"`
}

// Edge is one scored pair.
//
// Rank is computed here rather than in the page: analyzer.RankKey is the one
// definition of corroborated evidence in this repo, and a second one in
// JavaScript would drift from it silently.
type Edge struct {
	A int `json:"a"` // unit IDs, always A < B
	B int `json:"b"`

	Shape   float64 `json:"shape"`   // fingerprint code-shape
	Overlap float64 `json:"overlap"` // comparator structural overlap
	Total   float64 `json:"total"`   // retrieval evidence mass, nats
	Trophic float64 `json:"trophic"`
	CallSim float64 `json:"callSim"`
	Rank    float64 `json:"rank"`

	Channels []string `json:"channels,omitempty"`
	Kind     string   `json:"kind,omitempty"`     // "interface implementations", "diverged copy"
	KindNote string   `json:"kindNote,omitempty"` // the rendered clause
	Merge    bool     `json:"merge"`
	Cross    bool     `json:"cross"` // the two sides live in different packages

	// Containment is the shared Weisfeiler-Lehman mass over the *smaller*
	// side's, so a small function wholly absorbed into a large one reads near
	// 1.00 where Shape — a symmetric Jaccard — reads low. Reported beside
	// Shape and never blended into it: "these two are alike" and "this one is
	// inside that one" are different findings, and one number cannot say both.
	Containment float64 `json:"containment"`

	// Explain is the rule-attributed sentence about the pair: what the
	// canonicalizer had to do to the two bodies before they matched. It is the
	// only place the canonical rewrite is legible to a reader of this page.
	Explain string `json:"explain,omitempty"`

	// Breakdown is the six fingerprint components in their fixed order:
	// wl, flow, nesting, sig, size, containment. An array rather than a struct
	// because the page draws them as one bar list and the names are constant.
	//
	// The first component is a corpus-weighted multiset Jaccard over the two
	// Weisfeiler-Lehman label bags — it was a Jaccard over token 3-grams when
	// this payload was first written, which is why the name changed with it.
	Breakdown [6]float64 `json:"breakdown"`

	Chains  []Chain  `json:"chains,omitempty"`
	Reasons []string `json:"reasons,omitempty"`
}

// BreakdownNames labels Edge.Breakdown, in its fixed order.
var BreakdownNames = [6]string{"wl", "flow", "nesting", "sig", "size", "containment"}

// UnitConcept is one graded concept membership.
//
// Confidence is carried, not rounded away to a boolean: a learned concept is
// derived from this corpus's own vocabulary, so carrying it is a matter of
// degree, and a reader deciding whether two functions really do the same work
// needs to see whether both sides mean it.
type UnitConcept struct {
	ID         string  `json:"id"`
	Confidence float64 `json:"confidence"`
}

// Chain is one shared structural label behind a pair — the evidence its score
// actually rests on.
//
// Render names the label rather than reproducing it (`depth-2 IF`): a
// Weisfeiler-Lehman label is a hash of a whole subtree, and the fingerprint
// keeps no mapping back to the tokens that produced it. So the page lists
// chains beside the bodies rather than highlighting them inside, and Depth is
// how much structure the shared label covers — the refinement round, not a
// pattern level.
type Chain struct {
	Depth  int     `json:"depth"`
	Energy float64 `json:"energy"`
	Render string  `json:"render"`
}

// Body is one function's source text, for the side-by-side view.
type Body struct {
	Unit int    `json:"unit"`
	Text string `json:"text"`
}

// Family is one near-duplicate clique.
type Family struct {
	Members  []int   `json:"members"` // unit IDs, ascending
	MinEdge  float64 `json:"minEdge"`
	MeanEdge float64 `json:"meanEdge"`
	Evidence float64 `json:"evidence"`
	Added    int     `json:"added"` // edges supplied by completion, not retrieval
	Kind     string  `json:"kind,omitempty"`
	Tag      string  `json:"tag,omitempty"`
}

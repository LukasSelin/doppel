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
	// always one the unit actually carries. Denormalised rather than derived in
	// the page because the map reads it per function per redraw.
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

	// Breakdown is the five fingerprint components in their fixed order:
	// ast, flow, nesting, sig, size. An array rather than a struct because the
	// page draws them as one bar list and the names are constant.
	Breakdown [5]float64 `json:"breakdown"`

	Chains  []Chain  `json:"chains,omitempty"`
	Reasons []string `json:"reasons,omitempty"`
}

// BreakdownNames labels Edge.Breakdown, in its fixed order.
var BreakdownNames = [5]string{"ast", "flow", "nesting", "sig", "size"}

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

// Chain is one shared structural pattern behind a pair — the evidence its
// score actually rests on.
//
// Render is a motif string (`if(bin:!=(id,nil))`), not a source span: the
// fingerprint hashes patterns and keeps no mapping back to the tokens that
// produced them, so the page lists chains beside the bodies rather than
// highlighting them inside.
type Chain struct {
	Level  int     `json:"level"`
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

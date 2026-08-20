package ontology

// Role terms: where a function sits in the call graph. The IDs are exactly the
// strings concepter.ClassifyRole returns.
const (
	RoleRole         TermID = "role"
	RoleLeaf         TermID = "leaf"
	RoleUtility      TermID = "utility"
	RoleOrchestrator TermID = "orchestrator"
	RolePassthrough  TermID = "passthrough"
)

// RoleAxes is the pair of independent booleans a role decomposes into.
// ClassifyRole already computed roles this way; naming the axes is what makes
// graded role matching possible, since two roles can now agree on one axis.
//
// A struct of bools, deliberately, not a set: nothing here may introduce a map
// whose iteration order could reach a score or a report line.
type RoleAxes struct {
	HighFanIn  bool // many callers
	HighFanOut bool // many callees
}

var roleTerms = []Term{
	{ID: RoleRole, Kind: KindRole, Abstract: true,
		Label: "Role", Def: "A position in the call graph, from fan-in and fan-out."},
	{ID: RoleLeaf, Kind: KindRole, Parent: RoleRole,
		Label: "Leaf", Def: "Few callers and few callees; standalone or isolated."},
	{ID: RoleUtility, Kind: KindRole, Parent: RoleRole,
		Label: "Utility", Def: "Many callers and few callees; a shared helper."},
	{ID: RoleOrchestrator, Kind: KindRole, Parent: RoleRole,
		Label: "Orchestrator", Def: "Few callers and many callees; a handler or controller."},
	{ID: RolePassthrough, Kind: KindRole, Parent: RoleRole,
		Label: "Passthrough", Def: "Many callers and many callees; middleware or delegation."},
}

// roleAxes is the single truth table mapping roles to axes. concepter reads it
// through RoleFor rather than keeping its own switch, so there is one
// definition to change when a fifth role appears.
var roleAxes = []struct {
	id   TermID
	axes RoleAxes
}{
	{RoleLeaf, RoleAxes{false, false}},
	{RoleUtility, RoleAxes{true, false}},
	{RoleOrchestrator, RoleAxes{false, true}},
	{RolePassthrough, RoleAxes{true, true}},
}

// RoleFor returns the role occupying a combination of axes. The table is total
// over the four combinations (axiom 9), so the fallback is unreachable.
func RoleFor(axes RoleAxes) TermID {
	for _, entry := range roleAxes {
		if entry.axes == axes {
			return entry.id
		}
	}
	return RoleLeaf
}

// AxesFor decomposes a role back into its axes.
func AxesFor(id TermID) (RoleAxes, bool) {
	for _, entry := range roleAxes {
		if entry.id == id {
			return entry.axes, true
		}
	}
	return RoleAxes{}, false
}

// RoleRelatedness scores how alike two roles are, as the Jaccard overlap of the
// axes on which they are HIGH.
//
//	identical                     1.00
//	utility      vs passthrough   0.50   (both high fan-in)
//	orchestrator vs passthrough   0.50   (both high fan-out)
//	leaf         vs orchestrator  0.00
//	orchestrator vs utility       0.00
//
// Only shared positive attributes count. Crediting agreement on a low axis
// would be worse than useless: Callees counts every call expression including
// stdlib, so fan-out skews high and leaf mostly means "we could not tell".
// Scoring two functions for jointly not being utilities is noise, and it would
// raise the floor under every unrelated pair in the report.
//
// Two leaves are the one both-empty case, and they are identical roles, so they
// score 1.0. Note the opposite convention in SetRelatedness, where two empty
// tag sets score 0.0. The two Jaccard-shaped functions must not be merged into
// one helper.
func RoleRelatedness(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	ax, okA := AxesFor(TermID(a))
	bx, okB := AxesFor(TermID(b))
	if !okA || !okB {
		return 0
	}
	var inter, union int
	for _, axis := range [][2]bool{{ax.HighFanIn, bx.HighFanIn}, {ax.HighFanOut, bx.HighFanOut}} {
		if axis[0] && axis[1] {
			inter++
		}
		if axis[0] || axis[1] {
			union++
		}
	}
	if union == 0 {
		return 1
	}
	return float64(inter) / float64(union)
}

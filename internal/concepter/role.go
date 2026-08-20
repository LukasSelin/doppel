package concepter

import "github.com/lukse/doppel/internal/ontology"

// Role names, re-exported from the ontology so existing callers keep compiling
// and so there is one place these strings are defined.
const (
	RoleLeaf         = string(ontology.RoleLeaf)
	RoleUtility      = string(ontology.RoleUtility)
	RoleOrchestrator = string(ontology.RoleOrchestrator)
	RolePassthrough  = string(ontology.RolePassthrough)
)

// roleThreshold is the fan-in and fan-out count at which a function counts as
// high on that axis. Inclusive.
const roleThreshold = 2

// ClassifyRole returns a structural role based on fan-in (callers) and
// fan-out (callees) counts.
//
//	leaf:         few callers, few callees — standalone/isolated
//	utility:      many callers, few callees — shared helper
//	orchestrator: few callers, many callees — handler/controller
//	passthrough:  many callers, many callees — middleware/delegation
//
// The two counts were always independent; the ontology names them as axes so
// the comparator can score two roles that agree on one of them. The mapping
// from axes to role lives there, in a single table, rather than being restated
// as a switch here.
func ClassifyRole(callerCount, calleeCount int) string {
	return string(ontology.RoleFor(ontology.RoleAxes{
		HighFanIn:  callerCount >= roleThreshold,
		HighFanOut: calleeCount >= roleThreshold,
	}))
}

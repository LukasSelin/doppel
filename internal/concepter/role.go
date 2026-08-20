package concepter

import (
	"sort"

	"github.com/lukse/doppel/internal/ontology"
)

// Role names, re-exported from the ontology so existing callers keep compiling
// and so there is one place these strings are defined.
const (
	RoleLeaf         = string(ontology.RoleLeaf)
	RoleUtility      = string(ontology.RoleUtility)
	RoleOrchestrator = string(ontology.RoleOrchestrator)
	RolePassthrough  = string(ontology.RolePassthrough)
)

// roleThreshold is the default fan-in and fan-out count at which a function
// counts as high on that axis. Inclusive. The pipeline derives per-corpus
// thresholds instead (see RoleThresholds); this constant is the floor they can
// never drop below, and what ClassifyRole uses on its own.
const roleThreshold = 2

// RoleThresholds is the per-axis cutoff for "high", derived from the corpus
// being analysed rather than fixed, because fan-in and fan-out have very
// different distributions and both shift with repo size and style.
type RoleThresholds struct {
	FanIn  int
	FanOut int
}

// DefaultRoleThresholds is the fixed floor both axes share.
func DefaultRoleThresholds() RoleThresholds {
	return RoleThresholds{FanIn: roleThreshold, FanOut: roleThreshold}
}

// ThresholdsFromDegrees derives per-axis thresholds from the corpus's resolved
// degree distributions: high means strictly above the axis's median degree,
// floored at the default so sparse graphs behave exactly as the fixed
// threshold always did.
//
// Zero-degree units are deliberately included — they are the population, and
// excluding them would redefine "high fan-in" as "above the median of
// connected units", inflating the threshold precisely on the sparse graphs
// where resolution is strictest. On most repos, this one included, resolved
// median degrees are 0 or 1 and both thresholds sit at the floor; the adaptive
// branch is dormant until a genuinely dense graph raises the median, which is
// exactly when the fixed 2 would be mislabeling half the corpus "high". Do not
// simplify the dormant branch away.
func ThresholdsFromDegrees(fanIn, fanOut []int) RoleThresholds {
	return RoleThresholds{
		FanIn:  maxInt(roleThreshold, upperMedian(fanIn)+1),
		FanOut: maxInt(roleThreshold, upperMedian(fanOut)+1),
	}
}

// upperMedian returns sorted[n/2] (0-indexed), the upper median, or 0 for an
// empty distribution. The input is not modified.
func upperMedian(degrees []int) int {
	if len(degrees) == 0 {
		return 0
	}
	sorted := make([]int, len(degrees))
	copy(sorted, degrees)
	sort.Ints(sorted)
	return sorted[len(sorted)/2]
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

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
	return ClassifyRoleAt(callerCount, calleeCount, DefaultRoleThresholds())
}

// ClassifyRoleAt classifies against explicit per-axis thresholds, which the
// pipeline derives from the corpus's own degree distribution.
func ClassifyRoleAt(callerCount, calleeCount int, th RoleThresholds) string {
	return string(ontology.RoleFor(ontology.RoleAxes{
		HighFanIn:  callerCount >= th.FanIn,
		HighFanOut: calleeCount >= th.FanOut,
	}))
}

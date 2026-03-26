package concepter

const (
	RoleLeaf         = "leaf"
	RoleUtility      = "utility"
	RoleOrchestrator = "orchestrator"
	RolePassthrough  = "passthrough"

	roleThreshold = 2
)

// ClassifyRole returns a structural role based on fan-in (callers) and
// fan-out (callees) counts.
//
//	leaf:         few callers, few callees — standalone/isolated
//	utility:      many callers, few callees — shared helper
//	orchestrator: few callers, many callees — handler/controller
//	passthrough:  many callers, many callees — middleware/delegation
func ClassifyRole(callerCount, calleeCount int) string {
	highIn := callerCount >= roleThreshold
	highOut := calleeCount >= roleThreshold

	switch {
	case highIn && highOut:
		return RolePassthrough
	case highIn:
		return RoleUtility
	case highOut:
		return RoleOrchestrator
	default:
		return RoleLeaf
	}
}

package retriever

// OtherIdx is the side of a probe candidate that is not the probe.
func OtherIdx(c Candidate, probeIdx int) int {
	if c.AIdx == probeIdx {
		return c.BIdx
	}
	return c.AIdx
}

// ProbeLess is the order `doppel query` reports a probe's matches in:
// Total×(1+Locality) desc — evidence boosted by architectural nearness, never
// punished by distance — then Total desc, then code-shape desc, then the
// candidate's corpus index asc. A total order, so a fixed corpus and snippet
// always print the same report. li and lj are the two candidates' localities
// (concepter.Locality against the probe's ball).
//
// It lives beside Probe rather than in cmd so internal/bench's
// plant-and-recover harness ranks probes by the very order the command uses.
func ProbeLess(ci, cj Candidate, li, lj float64, probeIdx int) bool {
	ki := ci.Total * (1 + li)
	kj := cj.Total * (1 + lj)
	if ki != kj {
		return ki > kj
	}
	if ci.Total != cj.Total {
		return ci.Total > cj.Total
	}
	// An exact-clone family ties on evidence mass; code-shape separates the
	// byte-identical twin from the merely similar sibling.
	if ci.Breakdown.Score != cj.Breakdown.Score {
		return ci.Breakdown.Score > cj.Breakdown.Score
	}
	return OtherIdx(ci, probeIdx) < OtherIdx(cj, probeIdx)
}

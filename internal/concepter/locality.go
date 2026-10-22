package concepter

import "github.com/LukasSelin/doppel/internal/parser"

// NeighborhoodSet is u's depth-2 call-graph ball as a set; nil when the ball
// is empty.
//
// It and Locality are `doppel query`'s locality weight, kept here rather than
// in cmd so that internal/bench's plant-and-recover harness ranks a probe
// exactly as the command does: bench cannot import cmd, and a second copy of
// the ranking would drift from the one it measures.
func NeighborhoodSet(g *Graph, u parser.CodeUnit) map[string]bool {
	ball := g.Neighborhood(QualifiedName(u))
	if len(ball) == 0 {
		return nil
	}
	set := make(map[string]bool, len(ball))
	for _, n := range ball {
		set[n] = true
	}
	return set
}

// Locality measures how much of the probe's architectural neighborhood the
// candidate inhabits: |ball(probe) ∩ (ball(c) ∪ {c})| / |ball(probe)|.
//
// The candidate's own identity joins its ball because a function the probe
// directly calls is maximally near and must not score below its own
// neighbors. An empty probe ball — a snippet that resolved no calls — makes
// every locality 0: honestly neutral, and the report says so rather than
// letting a column of zeros imply the corpus is far away.
func Locality(probeBall map[string]bool, g *Graph, c parser.CodeUnit) float64 {
	if len(probeBall) == 0 {
		return 0
	}
	qn := QualifiedName(c)
	hits := 0
	if probeBall[qn] {
		hits++
	}
	for _, n := range g.Neighborhood(qn) {
		if n != qn && probeBall[n] {
			hits++
		}
	}
	// hits can exceed the intersection only if qn were in its own ball, which
	// Neighborhood excludes by construction.
	if hits > len(probeBall) {
		hits = len(probeBall)
	}
	return float64(hits) / float64(len(probeBall))
}

package comparator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lukse/doppel/internal/concepter"
)

// Weights for each structural signal in the composite OverlapScore.
const (
	weightSharedCallees    = 0.25
	weightSharedCallers    = 0.15
	weightSharedPatterns   = 0.20
	weightSameRole         = 0.15
	weightSamePackage      = 0.10
	weightSameVisibility   = 0.05
	weightSameReceiver     = 0.05
	weightSharedCallerPkgs = 0.025
	weightSharedCalleePkgs = 0.025

	mergeThreshold   = 0.4
	minMergeSignals  = 2
)

// StructuralEvidence summarises the structural overlap between two ConceptDocs.
type StructuralEvidence struct {
	SharedCallees    []string
	SharedCallers    []string
	SharedPatterns   []string
	SameRole         bool
	RoleA, RoleB     string
	SamePackage      bool
	SameVisibility   bool
	SameReceiver     bool
	ReceiverA        string
	ReceiverB        string
	SharedCallerPkgs []string
	SharedCalleePkgs []string
	OverlapScore     float64  // 0.0–1.0 weighted composite
	MergeWorthy      bool     // heuristic: high overlap + multiple signals
	Reasons          []string // human-readable evidence bullets
}

// Compare computes the structural overlap between two ConceptDocs.
func Compare(a, b concepter.ConceptDoc) StructuralEvidence {
	ev := StructuralEvidence{
		SharedCallees:    intersect(a.Callees, b.Callees),
		SharedCallers:    intersect(a.Callers, b.Callers),
		SharedPatterns:   intersect(a.Patterns, b.Patterns),
		SameRole:         a.Role == b.Role && a.Role != "",
		RoleA:            a.Role,
		RoleB:            b.Role,
		SamePackage:      a.Package == b.Package && a.Package != "",
		SameVisibility:   a.Exported == b.Exported,
		SameReceiver:     a.ReceiverType == b.ReceiverType,
		ReceiverA:        a.ReceiverType,
		ReceiverB:        b.ReceiverType,
		SharedCallerPkgs: intersect(a.CallerPackages, b.CallerPackages),
		SharedCalleePkgs: intersect(a.CalleePackages, b.CalleePackages),
	}

	// Compute weighted overlap score.
	ev.OverlapScore = 0 +
		weightSharedCallees*overlapRatio(a.Callees, b.Callees, ev.SharedCallees) +
		weightSharedCallers*overlapRatio(a.Callers, b.Callers, ev.SharedCallers) +
		weightSharedPatterns*overlapRatio(a.Patterns, b.Patterns, ev.SharedPatterns) +
		weightSameRole*boolFloat(ev.SameRole) +
		weightSamePackage*boolFloat(ev.SamePackage) +
		weightSameVisibility*boolFloat(ev.SameVisibility) +
		weightSameReceiver*boolFloat(ev.SameReceiver) +
		weightSharedCallerPkgs*overlapRatio(a.CallerPackages, b.CallerPackages, ev.SharedCallerPkgs) +
		weightSharedCalleePkgs*overlapRatio(a.CalleePackages, b.CalleePackages, ev.SharedCalleePkgs)

	if ev.OverlapScore > 1.0 {
		ev.OverlapScore = 1.0
	}

	// Build reasons and determine merge-worthiness.
	ev.Reasons = buildReasons(ev)
	signals := countSignals(ev)
	ev.MergeWorthy = ev.OverlapScore >= mergeThreshold && signals >= minMergeSignals

	return ev
}

// overlapRatio returns |shared| / max(|a|, |b|), or 0 if both are empty.
func overlapRatio(a, b, shared []string) float64 {
	m := len(a)
	if len(b) > m {
		m = len(b)
	}
	if m == 0 {
		return 0
	}
	return float64(len(shared)) / float64(m)
}

func boolFloat(v bool) float64 {
	if v {
		return 1.0
	}
	return 0.0
}

// countSignals counts how many distinct merge-supporting signals are present.
func countSignals(ev StructuralEvidence) int {
	n := 0
	if len(ev.SharedCallees) > 0 {
		n++
	}
	if len(ev.SharedCallers) > 0 {
		n++
	}
	if len(ev.SharedPatterns) > 0 {
		n++
	}
	if ev.SameRole {
		n++
	}
	if ev.SamePackage {
		n++
	}
	return n
}

func buildReasons(ev StructuralEvidence) []string {
	var reasons []string
	if len(ev.SharedCallees) > 0 {
		reasons = append(reasons, fmt.Sprintf("share %d callees: [%s]", len(ev.SharedCallees), strings.Join(ev.SharedCallees, ", ")))
	}
	if len(ev.SharedCallers) > 0 {
		reasons = append(reasons, fmt.Sprintf("share %d callers: [%s]", len(ev.SharedCallers), strings.Join(ev.SharedCallers, ", ")))
	}
	if len(ev.SharedPatterns) > 0 {
		reasons = append(reasons, fmt.Sprintf("share patterns: [%s]", strings.Join(ev.SharedPatterns, ", ")))
	}
	if ev.SameRole {
		reasons = append(reasons, fmt.Sprintf("both are %s functions", ev.RoleA))
	}
	if ev.SamePackage {
		reasons = append(reasons, "same package")
	}
	if ev.SameVisibility {
		reasons = append(reasons, "same visibility")
	}
	if ev.SameReceiver {
		recv := "plain functions"
		if ev.ReceiverA != "" {
			recv = ev.ReceiverA
		}
		reasons = append(reasons, fmt.Sprintf("same receiver type: %s", recv))
	}
	if len(ev.SharedCallerPkgs) > 0 {
		reasons = append(reasons, fmt.Sprintf("called from same packages: [%s]", strings.Join(ev.SharedCallerPkgs, ", ")))
	}
	if len(ev.SharedCalleePkgs) > 0 {
		reasons = append(reasons, fmt.Sprintf("call into same packages: [%s]", strings.Join(ev.SharedCalleePkgs, ", ")))
	}
	return reasons
}

// intersect returns the sorted intersection of two sorted string slices.
func intersect(a, b []string) []string {
	set := make(map[string]struct{}, len(a))
	for _, s := range a {
		set[s] = struct{}{}
	}
	var out []string
	for _, s := range b {
		if _, ok := set[s]; ok {
			out = append(out, s)
			delete(set, s) // avoid duplicates
		}
	}
	sort.Strings(out)
	return out
}

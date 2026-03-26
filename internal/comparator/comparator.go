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

	mergeThreshold  = 0.4
	minMergeSignals = 2
)

// DimensionScore holds per-dimension similarity, contradiction, and evidence.
type DimensionScore struct {
	Name          string  // e.g. "SharedCallees", "SameRole"
	Weight        float64 // weight used in composite scoring
	Similarity    float64 // 0.0–1.0 agreement
	Contradiction float64 // 0.0–1.0 disagreement
	Evidence      string  // human-readable explanation
	Active        bool    // true when at least one side has data for this dimension
}

// StructuralEvidence summarises the structural overlap between two ConceptDocs.
type StructuralEvidence struct {
	Dimensions   []DimensionScore
	OverlapScore float64 // 0.0–1.0 weighted composite of Similarity scores
	MergeWorthy  bool    // heuristic: high overlap + multiple signals
}

// Compare computes the structural overlap between two ConceptDocs.
func Compare(a, b concepter.ConceptDoc) StructuralEvidence {
	dims := []DimensionScore{
		listDimension("SharedCallees", weightSharedCallees, a.Callees, b.Callees),
		listDimension("SharedCallers", weightSharedCallers, a.Callers, b.Callers),
		listDimension("SharedPatterns", weightSharedPatterns, a.Patterns, b.Patterns),
		roleDimension(a.Role, b.Role),
		stringMatchDimension("SamePackage", weightSamePackage, a.Package, b.Package, "package"),
		boolDimension("SameVisibility", weightSameVisibility, a.Exported, b.Exported, "exported", "unexported"),
		receiverDimension(a.ReceiverType, b.ReceiverType),
		listDimension("SharedCallerPkgs", weightSharedCallerPkgs, a.CallerPackages, b.CallerPackages),
		listDimension("SharedCalleePkgs", weightSharedCalleePkgs, a.CalleePackages, b.CalleePackages),
	}

	// Compute weighted average across active dimensions only.
	// Inactive dimensions (no data on either side) do not penalise the score.
	var weightedSum, activeWeight float64
	for _, d := range dims {
		if d.Active {
			weightedSum += d.Weight * d.Similarity
			activeWeight += d.Weight
		}
	}
	var score float64
	if activeWeight > 0 {
		score = weightedSum / activeWeight
	}
	if score > 1.0 {
		score = 1.0
	}

	signals := countSignals(dims)

	return StructuralEvidence{
		Dimensions:   dims,
		OverlapScore: score,
		MergeWorthy:  score >= mergeThreshold && signals >= minMergeSignals,
	}
}

// listDimension scores a pair of string slices by overlap ratio.
func listDimension(name string, weight float64, a, b []string) DimensionScore {
	shared := intersect(a, b)
	sim := overlapRatio(a, b, shared)
	active := len(a) > 0 || len(b) > 0

	var contra float64
	if len(a) > 0 && len(b) > 0 {
		contra = 1.0 - sim
	}

	evidence := listEvidence(name, a, b, shared)

	return DimensionScore{
		Name:          name,
		Weight:        weight,
		Similarity:    sim,
		Contradiction: contra,
		Evidence:      evidence,
		Active:        active,
	}
}

// roleDimension scores the Role field.
func roleDimension(roleA, roleB string) DimensionScore {
	var sim, contra float64
	var evidence string
	active := roleA != "" || roleB != ""

	if roleA != "" && roleB != "" {
		if roleA == roleB {
			sim = 1.0
			evidence = fmt.Sprintf("both are %s functions", roleA)
		} else {
			contra = 1.0
			evidence = fmt.Sprintf("roles differ: %s vs %s", roleA, roleB)
		}
	}

	return DimensionScore{
		Name:          "SameRole",
		Weight:        weightSameRole,
		Similarity:    sim,
		Contradiction: contra,
		Evidence:      evidence,
		Active:        active,
	}
}

// stringMatchDimension scores two string fields that either match or don't.
func stringMatchDimension(name string, weight float64, a, b, label string) DimensionScore {
	var sim, contra float64
	var evidence string
	active := a != "" || b != ""

	if a != "" && b != "" {
		if a == b {
			sim = 1.0
			evidence = fmt.Sprintf("both in %s %s", label, a)
		} else {
			contra = 1.0
			evidence = fmt.Sprintf("different %ss: %s vs %s", label, a, b)
		}
	}

	return DimensionScore{
		Name:          name,
		Weight:        weight,
		Similarity:    sim,
		Contradiction: contra,
		Evidence:      evidence,
		Active:        active,
	}
}

// boolDimension scores two boolean fields.
func boolDimension(name string, weight float64, a, b bool, trueLabel, falseLabel string) DimensionScore {
	var sim, contra float64
	var evidence string

	if a == b {
		sim = 1.0
		label := falseLabel
		if a {
			label = trueLabel
		}
		evidence = fmt.Sprintf("both %s", label)
	} else {
		contra = 1.0
		evidence = fmt.Sprintf("visibility differs: %s vs %s", boolLabel(a, trueLabel, falseLabel), boolLabel(b, trueLabel, falseLabel))
	}

	return DimensionScore{
		Name:          name,
		Weight:        weight,
		Similarity:    sim,
		Contradiction: contra,
		Evidence:      evidence,
		Active:        true, // booleans are always defined
	}
}

// receiverDimension scores the ReceiverType field.
func receiverDimension(a, b string) DimensionScore {
	var sim, contra float64
	var evidence string

	if a == b {
		sim = 1.0
		if a == "" {
			evidence = "both plain functions"
		} else {
			evidence = fmt.Sprintf("same receiver type: %s", a)
		}
	} else if a != "" && b != "" {
		contra = 1.0
		evidence = fmt.Sprintf("different receivers: %s vs %s", a, b)
	}
	// If one has a receiver and one doesn't: sim=0, contra=0 (not comparable)

	return DimensionScore{
		Name:          "SameReceiver",
		Weight:        weightSameReceiver,
		Similarity:    sim,
		Contradiction: contra,
		Evidence:      evidence,
		Active:        true, // always relevant — "both plain" is meaningful
	}
}

func boolLabel(v bool, trueLabel, falseLabel string) string {
	if v {
		return trueLabel
	}
	return falseLabel
}

// listEvidence builds a human-readable evidence string for list dimensions.
func listEvidence(name string, a, b, shared []string) string {
	if len(a) == 0 && len(b) == 0 {
		return ""
	}

	label := dimensionLabel(name)

	if len(shared) == 0 {
		onlyA := setDifference(a, b)
		onlyB := setDifference(b, a)
		return fmt.Sprintf("no shared %s (A:%d B:%d); differ on: [%s] vs [%s]",
			label, len(a), len(b), strings.Join(onlyA, ", "), strings.Join(onlyB, ", "))
	}

	parts := fmt.Sprintf("share %d %s (A:%d B:%d): [%s]",
		len(shared), label, len(a), len(b), strings.Join(shared, ", "))

	onlyA := setDifference(a, shared)
	onlyB := setDifference(b, shared)
	if len(onlyA) > 0 || len(onlyB) > 0 {
		parts += fmt.Sprintf("; only A: [%s]; only B: [%s]",
			strings.Join(onlyA, ", "), strings.Join(onlyB, ", "))
	}

	return parts
}

// dimensionLabel converts a dimension name to a human-readable plural label.
func dimensionLabel(name string) string {
	switch name {
	case "SharedCallees":
		return "callees"
	case "SharedCallers":
		return "callers"
	case "SharedPatterns":
		return "patterns"
	case "SharedCallerPkgs":
		return "caller packages"
	case "SharedCalleePkgs":
		return "callee packages"
	default:
		return name
	}
}

// countSignals counts how many distinct merge-supporting signals are present.
func countSignals(dims []DimensionScore) int {
	n := 0
	for _, d := range dims {
		if d.Similarity > 0 {
			n++
		}
	}
	return n
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

// setDifference returns elements in a that are not in b, sorted.
func setDifference(a, b []string) []string {
	set := make(map[string]struct{}, len(b))
	for _, s := range b {
		set[s] = struct{}{}
	}
	var out []string
	for _, s := range a {
		if _, ok := set[s]; !ok {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// intersect returns the sorted intersection of two string slices.
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

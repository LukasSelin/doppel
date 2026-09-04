package comparator

import (
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
)

// SignalCount is the number of graded signals Compare blends — the scored
// relations, in ontology declaration order.
const SignalCount = 12

// SignalVector returns the twelve graded signals behind ev.OverlapScore, in
// ontology.ScoredRelations() order, so the weighted sum of this vector in
// that order reproduces the composite exactly (before the 1.0 clamp). The
// exhibits slot is ev.Exhibits — whatever blend of the concept views the
// comparator was built with — never a single view, so the reproduction holds
// under every Options. Eight
// are stored on the evidence; the four set-overlap ratios are recomputed from
// the stored shared slices and the two docs, exactly as Compare did.
// NeighborhoodOverlap is read from ev rather than re-derived: Compare
// excludes each side's counterpart from the balls first, and that exclusion
// is not reproducible from the docs alone.
//
// This exists for measurement — the bench self-weighting experiment asks
// which signals actually discriminate candidates from random pairs on a
// corpus. Nothing in the production pipeline calls it.
func SignalVector(ev StructuralEvidence, a, b concepter.ConceptDoc) [SignalCount]float64 {
	return [SignalCount]float64{
		overlapRatio(a.Callees, b.Callees, ev.SharedCallees), // calls
		ev.Exhibits, // exhibits: the Options blend of the views
		overlapRatio(a.Callers, b.Callers, ev.SharedCallers), // called_by
		ev.RoleRelatedness,           // has_role
		boolFloat(ev.SamePackage),    // declared_in
		ev.CallerConceptRelatedness,  // called_from_concept
		ev.CalleeConceptRelatedness,  // calls_into_concept
		boolFloat(ev.SameVisibility), // has_visibility
		ev.ReceiverRelatedness,       // bound_to
		ev.NeighborhoodOverlap,       // shares_neighborhood
		overlapRatio(a.CallerPackages, b.CallerPackages, ev.SharedCallerPkgs), // called_from_package
		overlapRatio(a.CalleePackages, b.CalleePackages, ev.SharedCalleePkgs), // calls_into_package
	}
}

// SignalOrder is the relation each SignalVector slot scores, for rendering.
var SignalOrder = [SignalCount]ontology.TermID{
	ontology.RelCalls, ontology.RelExhibits, ontology.RelCalledBy, ontology.RelHasRole,
	ontology.RelDeclaredIn, ontology.RelCalledFromConcept, ontology.RelCallsIntoConcept,
	ontology.RelHasVisibility, ontology.RelBoundTo, ontology.RelSharesNeighborhood,
	ontology.RelCalledFromPackage, ontology.RelCallsIntoPackage,
}

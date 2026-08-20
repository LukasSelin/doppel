package ontology

// Relation terms. Each names one edge type between entities, and carries the
// weight that edge's overlap contributes to comparator's composite score.
//
// The weights live here rather than as constants in comparator so that the
// scoring table and the vocabulary cannot drift apart, and so axiom 7 can
// assert they sum to exactly 1.0.
const (
	RelRelation          TermID = "relation"
	RelCalls             TermID = "calls"
	RelExhibits          TermID = "exhibits"
	RelCalledBy          TermID = "called_by"
	RelHasRole           TermID = "has_role"
	RelDeclaredIn        TermID = "declared_in"
	RelCalledFromConcept TermID = "called_from_concept"
	RelCallsIntoConcept  TermID = "calls_into_concept"
	RelHasVisibility     TermID = "has_visibility"
	RelBoundTo           TermID = "bound_to"
	RelCalledFromPackage TermID = "called_from_package"
	RelCallsIntoPackage  TermID = "calls_into_package"
)

// The nine pre-existing weights are their historical values scaled uniformly by
// 0.9, making room for called_from_concept and calls_into_concept at 0.05 each.
// Relative ordering is untouched: headroom was made, judgment was not applied.
var relationTerms = []Term{
	{ID: RelRelation, Kind: KindRelation, Abstract: true,
		Label: "Relation", Def: "A typed edge between two entities."},

	{ID: RelCalls, Kind: KindRelation, Parent: RelRelation, Weight: 0.225,
		Domain: EntCallable, Range: EntCallable, Inverse: RelCalledBy,
		Label: "calls", Def: "The subject invokes the object."},
	{ID: RelExhibits, Kind: KindRelation, Parent: RelRelation, Weight: 0.180,
		Domain: EntCallable, Range: ConConcept,
		Label: "exhibits", Def: "The subject's body shows the object's intent pattern."},
	{ID: RelCalledBy, Kind: KindRelation, Parent: RelRelation, Weight: 0.135,
		Domain: EntCallable, Range: EntCallable, Inverse: RelCalls,
		Label: "called by", Def: "The subject is invoked by the object."},
	{ID: RelHasRole, Kind: KindRelation, Parent: RelRelation, Weight: 0.135,
		Domain: EntCallable, Range: RoleRole,
		Label: "has role", Def: "The subject occupies the object's structural role."},
	{ID: RelDeclaredIn, Kind: KindRelation, Parent: RelRelation, Weight: 0.090,
		Domain: EntCallable, Range: EntPackage,
		Label: "declared in", Def: "The subject is declared in the object package."},
	{ID: RelCalledFromConcept, Kind: KindRelation, Parent: RelRelation, Weight: 0.050,
		Domain: EntCallable, Range: ConConcept,
		Label: "called from concept", Def: "Some caller of the subject exhibits the object's intent pattern."},
	{ID: RelCallsIntoConcept, Kind: KindRelation, Parent: RelRelation, Weight: 0.050,
		Domain: EntCallable, Range: ConConcept,
		Label: "calls into concept", Def: "Some callee of the subject exhibits the object's intent pattern."},
	{ID: RelHasVisibility, Kind: KindRelation, Parent: RelRelation, Weight: 0.045,
		Domain: EntCallable, Range: EntVisibility,
		Label: "has visibility", Def: "The subject is exported, or is not."},
	{ID: RelBoundTo, Kind: KindRelation, Parent: RelRelation, Weight: 0.045,
		Domain: EntMethod, Range: EntReceiverType,
		Label: "bound to", Def: "The subject method is declared on the object receiver type."},
	{ID: RelCalledFromPackage, Kind: KindRelation, Parent: RelRelation, Weight: 0.0225,
		Domain: EntCallable, Range: EntPackage,
		Label: "called from package", Def: "Some caller of the subject is declared in the object package."},
	{ID: RelCallsIntoPackage, Kind: KindRelation, Parent: RelRelation, Weight: 0.0225,
		Domain: EntCallable, Range: EntPackage,
		Label: "calls into package", Def: "Some callee of the subject is declared in the object package."},
}

// Weight returns a relation's contribution to the composite overlap score, or 0
// for a term that is not a scored relation.
func (o *Ontology) Weight(id TermID) float64 {
	return o.terms[id].Weight
}

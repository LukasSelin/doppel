package ontology

import (
	"fmt"
	"math"
	"regexp"
	"sort"
)

// weightEpsilon is the tolerance on the weight-sum axiom. The weights are
// written as decimal literals that are not exactly representable in binary, so
// their sum lands a few ulps off 1.0.
const weightEpsilon = 1e-9

var snakeCase = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// Validate checks the vocabulary against its axioms and returns every
// violation. An empty result is the invariant the rest of the pipeline relies
// on: a typo'd term or a rule pointing at a concept that does not exist would
// otherwise fail silently and change scores.
//
// Axioms 1-7 and 9 live here. Axiom 8, that the tagger's rules and the concept
// leaves are in exact correspondence, lives in the tagger's own test — the
// check needs the rule table, and importing tagger here would be a cycle.
//
// The returned errors are sorted. A validation pass that reported findings in
// map order would itself be the class of bug it exists to catch.
func (o *Ontology) Validate() []error {
	var errs []error
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	// Axiom 1: IDs are unique, non-empty and — where the vocabulary declares
	// them by hand — snake_case. Uniqueness across all Kinds is already
	// enforced by the ID-keyed map, so a collision shows up as a term that is
	// missing rather than duplicated; count instead.
	//
	// Concrete concept leaves are exempt from the spelling rule, and the
	// exemption is the point rather than a loophole. They are no longer
	// authored: internal/lexicon derives them from the corpus and names them
	// after the evidence that produced them ("sql.Open+QueryRow"), so a
	// spelling convention meant to keep a hand-written vocabulary tidy would
	// only force those names to be mangled into something a reader cannot match
	// against the code. Everything still declared by hand — every abstract
	// term, and every entity, relation and role — keeps the rule.
	if len(o.order) != len(o.terms) {
		add("axiom 1: %d declared terms collapsed to %d unique IDs", len(o.order), len(o.terms))
	}
	for _, id := range o.order {
		if id == "" {
			add("axiom 1: a term has an empty ID")
			continue
		}
		t := o.terms[id]
		if t.Kind == KindConcept && !t.Abstract {
			continue // corpus-derived name; see above
		}
		if !snakeCase.MatchString(string(id)) {
			add("axiom 1: term %q is not snake_case", id)
		}
	}

	// Axiom 2: parents exist, share their child's Kind, and each Kind has
	// exactly one root.
	roots := make(map[Kind][]TermID)
	for _, id := range o.order {
		t := o.terms[id]
		if t.Parent == "" {
			roots[t.Kind] = append(roots[t.Kind], id)
			continue
		}
		parent, ok := o.terms[t.Parent]
		if !ok {
			add("axiom 2: term %q has parent %q, which does not exist", id, t.Parent)
			continue
		}
		if parent.Kind != t.Kind {
			add("axiom 2: term %q is %s but its parent %q is %s", id, t.Kind, t.Parent, parent.Kind)
		}
	}
	for _, k := range Kinds {
		switch n := len(roots[k]); {
		case n == 0:
			add("axiom 2: kind %s has no root", k)
		case n > 1:
			sort.Slice(roots[k], func(i, j int) bool { return roots[k][i] < roots[k][j] })
			add("axiom 2: kind %s has %d roots: %v", k, n, roots[k])
		}
	}

	// Axiom 3: no cycles in any parent chain.
	for _, id := range o.order {
		if o.hasCycle(id) {
			add("axiom 3: term %q is in a parent cycle", id)
		}
	}

	// Axiom 4: every term is documented. An undefined term cannot be reviewed,
	// and `doppel ontology --defs` would print a blank line where the meaning
	// should be.
	for _, id := range o.order {
		t := o.terms[id]
		if t.Label == "" {
			add("axiom 4: term %q has no label", id)
		}
		if t.Def == "" {
			add("axiom 4: term %q has no definition", id)
		}
	}

	// Axiom 5: relations declare a domain and range that resolve.
	for _, id := range o.order {
		t := o.terms[id]
		if t.Kind != KindRelation || t.Abstract {
			continue
		}
		if _, ok := o.terms[t.Domain]; !ok {
			add("axiom 5: relation %q has domain %q, which does not exist", id, t.Domain)
		}
		if _, ok := o.terms[t.Range]; !ok {
			add("axiom 5: relation %q has range %q, which does not exist", id, t.Range)
		}
	}

	// Axiom 6: inverses are symmetric.
	for _, id := range o.order {
		t := o.terms[id]
		if t.Inverse == "" {
			continue
		}
		inv, ok := o.terms[t.Inverse]
		if !ok {
			add("axiom 6: relation %q has inverse %q, which does not exist", id, t.Inverse)
			continue
		}
		if inv.Inverse != id {
			add("axiom 6: %q declares inverse %q, but %q declares inverse %q", id, t.Inverse, t.Inverse, inv.Inverse)
		}
	}

	// Axiom 7: scored relation weights sum to 1.0. This is what lets the
	// comparator's composite stay on a 0.0-1.0 scale without renormalizing.
	var sum float64
	for _, id := range o.order {
		if t := o.terms[id]; t.Kind == KindRelation {
			sum += t.Weight
		}
	}
	if math.Abs(sum-1.0) > weightEpsilon {
		add("axiom 7: relation weights sum to %v, want 1.0", sum)
	}

	// Axiom 9: the role terms cover all four axis combinations exactly once, so
	// RoleFor is total and AxesFor is injective.
	counts := map[RoleAxes]int{}
	for _, entry := range roleAxes {
		counts[entry.axes]++
		if _, ok := o.terms[entry.id]; !ok {
			add("axiom 9: role %q has axes but no term", entry.id)
		}
	}
	for _, axes := range []RoleAxes{{false, false}, {true, false}, {false, true}, {true, true}} {
		if n := counts[axes]; n != 1 {
			add("axiom 9: axes {in:%t out:%t} map to %d roles, want exactly 1", axes.HighFanIn, axes.HighFanOut, n)
		}
	}
	for _, id := range o.order {
		t := o.terms[id]
		if t.Kind != KindRole || t.Abstract {
			continue
		}
		if _, ok := AxesFor(id); !ok {
			add("axiom 9: role %q has no axes", id)
		}
	}

	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errs
}

// hasCycle walks parent pointers further than the tree can possibly be deep.
// Reaching that bound means the walk is going round.
func (o *Ontology) hasCycle(id TermID) bool {
	cur := id
	for steps := 0; steps <= len(o.order); steps++ {
		t, ok := o.terms[cur]
		if !ok || t.Parent == "" {
			return false
		}
		cur = t.Parent
	}
	return true
}

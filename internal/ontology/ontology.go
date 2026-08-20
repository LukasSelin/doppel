// Package ontology is Doppel's formal vocabulary: the entity kinds, typed
// relations, intent concepts and structural roles the rest of the pipeline
// reasons about, declared once with definitions and machine-checkable axioms.
//
// Before this package the vocabulary existed only as bare strings scattered
// across tagger, concepter and comparator, which cost two things. There was no
// reasoning: http_call and db_access compared exactly equal to unrelated tags,
// so a real near-miss scored the same as nothing in common. And there was no
// integrity: a typo'd role or an unregistered tag failed silently and changed
// scores. Validate covers the second; the taxonomy plus Relatedness covers the
// first.
//
// The package holds no state beyond its own term tables, imports nothing from
// the rest of the module, and does no I/O. Every exported function is pure and
// deterministic — see the ordering rules on SetRelatedness, which the report's
// byte-for-byte reproducibility depends on.
package ontology

import "sort"

// Version identifies the vocabulary itself, so a change in term IDs or
// taxonomy shape is nameable in a report or a changelog.
const Version = "1.0.0"

// Kind partitions terms into four disjoint families, each a rooted tree.
type Kind string

const (
	KindEntity   Kind = "entity"
	KindRelation Kind = "relation"
	KindConcept  Kind = "concept"
	KindRole     Kind = "role"
)

// Kinds lists every Kind in a fixed order, for callers that walk all families.
var Kinds = []Kind{KindEntity, KindRelation, KindConcept, KindRole}

// TermID is the stable identifier of a term. Concept leaf IDs are exactly the
// strings tagger.Tag emits, and role IDs exactly the strings ClassifyRole
// returns, so the ontology can be introduced without changing any output.
type TermID string

// Term is one entry in the vocabulary.
type Term struct {
	ID       TermID
	Kind     Kind
	Label    string // human-readable name
	Def      string // one-sentence definition
	Parent   TermID // "" only for the root of a Kind
	Abstract bool   // classifies only; never asserted of a real code unit

	// Relation terms only.
	Domain  TermID
	Range   TermID
	Inverse TermID
	Weight  float64 // contribution to comparator's composite overlap score
}

// Ontology is an immutable, indexed view over the term tables.
type Ontology struct {
	terms    map[TermID]Term
	order    []TermID            // declaration order; the only iteration order
	depth    map[TermID]int      // root = 0
	children map[TermID][]TermID // sorted by ID
}

var defaultOntology = build()

// Default returns the shared vocabulary. The value is immutable, so callers
// must not retain and mutate the Terms they receive.
func Default() *Ontology { return defaultOntology }

// New assembles an Ontology from explicit term tables. Callers outside tests
// want Default; this exists so validate_test can construct deliberately broken
// vocabularies and check that Validate catches them.
func New(tables ...[]Term) *Ontology {
	o := &Ontology{
		terms:    make(map[TermID]Term),
		depth:    make(map[TermID]int),
		children: make(map[TermID][]TermID),
	}
	for _, table := range tables {
		for _, t := range table {
			if _, dup := o.terms[t.ID]; !dup {
				o.order = append(o.order, t.ID)
			}
			o.terms[t.ID] = t
		}
	}
	for _, id := range o.order {
		if p := o.terms[id].Parent; p != "" {
			o.children[p] = append(o.children[p], id)
		}
	}
	for parent := range o.children {
		sort.Slice(o.children[parent], func(i, j int) bool {
			return o.children[parent][i] < o.children[parent][j]
		})
	}
	for _, id := range o.order {
		o.depth[id] = o.computeDepth(id)
	}
	return o
}

func build() *Ontology {
	return New(entityTerms, relationTerms, conceptTerms, roleTerms)
}

// computeDepth walks parent pointers to the root. The step cap is a cycle
// guard: a malformed table must not hang Validate, which is the very thing
// meant to report the cycle.
func (o *Ontology) computeDepth(id TermID) int {
	d := 0
	for cur := id; ; d++ {
		t, ok := o.terms[cur]
		if !ok || t.Parent == "" || d > len(o.order) {
			return d
		}
		cur = t.Parent
	}
}

// Get returns the term with the given ID.
func (o *Ontology) Get(id TermID) (Term, bool) {
	t, ok := o.terms[id]
	return t, ok
}

// Terms returns every term in declaration order. Never range the underlying
// map instead: the report must be byte-identical across runs.
func (o *Ontology) Terms() []Term {
	out := make([]Term, 0, len(o.order))
	for _, id := range o.order {
		out = append(out, o.terms[id])
	}
	return out
}

// TermsOfKind returns the terms of one family, in declaration order.
func (o *Ontology) TermsOfKind(k Kind) []Term {
	var out []Term
	for _, id := range o.order {
		if t := o.terms[id]; t.Kind == k {
			out = append(out, t)
		}
	}
	return out
}

// Children returns the direct children of a term, sorted by ID.
func (o *Ontology) Children(id TermID) []TermID {
	return append([]TermID(nil), o.children[id]...)
}

// Root returns the single parentless term of a Kind.
func (o *Ontology) Root(k Kind) (TermID, bool) {
	for _, id := range o.order {
		if t := o.terms[id]; t.Kind == k && t.Parent == "" {
			return id, true
		}
	}
	return "", false
}

// Roots returns one root per Kind, in Kinds order.
func (o *Ontology) Roots() []TermID {
	var out []TermID
	for _, k := range Kinds {
		if id, ok := o.Root(k); ok {
			out = append(out, id)
		}
	}
	return out
}

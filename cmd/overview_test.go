package cmd

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/lexicon"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// A seed's leaf on the map counts functions, not memberships. Several concepts
// can grow from one seed and one function can carry more than one of them, so
// summing lexicon.Concept.Members would report more http_call functions than
// the corpus has.
func TestSeedYieldCountsEachFunctionOnce(t *testing.T) {
	concepts := []lexicon.Concept{
		{ID: "http.Get+Do", Seed: "http_call"},
		{ID: "http.NewRequest+Do", Seed: "http_call"},
		{ID: "sql.Open+QueryRow", Seed: "db_access"},
		// Emergent: no seed, so it credits nobody. Its Anchor decides only
		// where it hangs in the taxonomy, never where it came from.
		{ID: "buf.Len+buf.String", Anchor: "mapping"},
	}
	assign := [][]parser.Concept{
		// Both http concepts on one function: one function, one http_call.
		{{ID: "http.Get+Do"}, {ID: "http.NewRequest+Do"}},
		{{ID: "http.Get+Do"}, {ID: "sql.Open+QueryRow"}},
		{{ID: "buf.Len+buf.String"}},
		{},
	}

	got := seedYield(concepts, assign)
	if got["http_call"] != 2 {
		t.Errorf("http_call = %d, want 2", got["http_call"])
	}
	if got["db_access"] != 1 {
		t.Errorf("db_access = %d, want 1", got["db_access"])
	}
	if got["mapping"] != 0 {
		t.Errorf("mapping = %d, want 0 — an anchor is not provenance", got["mapping"])
	}
}

// A backfilled membership is one the unit did not earn, and every boolean
// reader of a membership skips it. BackfillN is 0 by default, so this is a
// no-op today and would be a silent overcount the day it is not.
func TestSeedYieldIgnoresBackfilledMemberships(t *testing.T) {
	concepts := []lexicon.Concept{{ID: "sql.Open+QueryRow", Seed: "db_access"}}
	assign := [][]parser.Concept{
		{{ID: "sql.Open+QueryRow"}},
		{{ID: "sql.Open+QueryRow", BelowFloor: true}},
	}
	if got := seedYield(concepts, assign); got["db_access"] != 1 {
		t.Errorf("db_access = %d, want 1", got["db_access"])
	}
}

// The map is the authored tree whatever the run found, so a run with no
// lexicon draws the whole thing with every leaf absent. A missing diagram
// would be the one answer that is not true.
func TestSeedMapDrawsTheAuthoredTreeWithoutALexicon(t *testing.T) {
	tree := seedMap(Result{})
	want := len(ontology.Default().TermsOfKind(ontology.KindConcept))
	if len(tree) != want {
		t.Fatalf("got %d nodes, want the whole authored taxonomy (%d)", len(tree), want)
	}
	var leaves, absent int
	for _, n := range tree {
		if n.Abstract {
			continue
		}
		leaves++
		if n.Count == 0 {
			absent++
		}
	}
	if leaves == 0 || leaves != absent {
		t.Errorf("%d of %d leaves absent, want all of them", absent, leaves)
	}
}

// A red leaf on the map and a name in the "No practice here for" sentence
// below it are two renderings of one fact, and they must not be able to
// disagree. The map says absent when no function reached a concept that seed
// grew; the sentence says it when the seed grew no concept at all. Those are
// the same set only because a concept always has founders and so always has at
// least one member — this pins the assumption, since a seeded concept nobody
// carries is the one shape that would split them.
func TestSeedMapAbsenceRestsOnConceptsHavingMembers(t *testing.T) {
	concepts := []lexicon.Concept{
		{ID: "sql.Open+QueryRow", Seed: "db_access"},
		{ID: "errors.Wrap+%w", Seed: "error_wrapping"},
		// Grew, per GrownSeeds — but nobody is assigned to it.
		{ID: "cache.Get+cache.Set", Seed: "caching"},
	}
	grown := seedYield(concepts, [][]parser.Concept{
		{{ID: "sql.Open+QueryRow"}, {ID: "errors.Wrap+%w"}},
	})

	for _, seed := range []string{"db_access", "error_wrapping"} {
		if grown[seed] == 0 {
			t.Errorf("%s grew a concept somebody carries and must not read absent", seed)
		}
	}
	if grown["retry"] != 0 {
		t.Error("a seed that grew nothing must read absent")
	}
	if grown["caching"] != 0 {
		t.Error("the memberless-concept case changed; the map and the sentence can now disagree")
	}
}

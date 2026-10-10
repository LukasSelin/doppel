package cmd

import (
	"math"
	"reflect"
	"testing"

	"github.com/LukasSelin/doppel/internal/culture"
)

// The grid's axis is drawn from the strongest cells, both ends at a time, so a
// shown concept's strongest partner is on the grid beside it — and the sample
// does not depend on the order associations arrive in.
func TestStrongestConceptsTakesCellsStrongestFirst(t *testing.T) {
	assocs := []culture.Association{
		{Kind: culture.TagTag, A: "weak1", B: "weak2", Count: 3, Expected: 1, PMI: math.Ln2},
		{Kind: culture.TagTag, A: "never1", B: "never2", Count: 0, Expected: 40, PMI: math.Inf(-1)},
		{Kind: culture.TagTag, A: "strong1", B: "strong2", Count: 30, Expected: 1, PMI: 3},
	}
	got := strongestConcepts(assocs, 1000, 3)
	want := []string{"never1", "never2", "strong1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	rev := []culture.Association{assocs[2], assocs[1], assocs[0]}
	if again := strongestConcepts(rev, 1000, 3); !reflect.DeepEqual(again, got) {
		t.Errorf("input order changed the sample: %v against %v", again, got)
	}
}

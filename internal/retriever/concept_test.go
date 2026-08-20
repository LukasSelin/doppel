package retriever

import (
	"math"
	"reflect"
	"testing"

	"github.com/lukse/doppel/internal/ontology"
)

// Two structurally dissimilar functions with identical rare tag sets, in a
// corpus of untagged filler. The structural channel cannot admit them (the
// threshold is set above their similarity); the concept channel must.
func TestConceptChannelAdmitsStructurallyDissimilarPairs(t *testing.T) {
	src := `package fix

func StoreOrder(id int, rows []string) error {
	for _, r := range rows {
		if r == "" {
			return nil
		}
	}
	return nil
}

func PersistShipment(labels map[string]int) int {
	total := 0
	for label, n := range labels {
		switch {
		case n > 10:
			total += n
		case label == "":
			total--
		}
	}
	return total
}

func UnrelatedFiller(x int) int {
	y := x * 2
	if y > 10 {
		y = y - x
	}
	for i := 0; i < 3; i++ {
		y += i
	}
	return y
}

func OtherFiller(s string) string {
	out := ""
	for _, c := range s {
		if c != ' ' {
			out += string(c)
		}
	}
	return out
}
`
	units := parseUnits(t, "fix.go", src)
	a := unitIndex(t, units, "StoreOrder")
	b := unitIndex(t, units, "PersistShipment")
	units[a].Patterns = []string{"transaction", "db_access"}
	units[b].Patterns = []string{"transaction", "db_access"}

	opt := DefaultOptions()
	opt.MinNodes = 8
	opt.Threshold = 0.99 // structural channel cannot admit anything non-identical

	cands, stats := retrieveAll(t, units, opt)
	c, ok := findCandidate(cands, a, b)
	if !ok {
		t.Fatal("tagged pair not retrieved by the concept channel")
	}
	if !reflect.DeepEqual(c.Channels, []string{ChannelConcept}) {
		t.Errorf("Channels = %v, want [concept] only", c.Channels)
	}
	if c.Concept <= 0 {
		t.Errorf("Concept evidence = %v, want > 0", c.Concept)
	}
	if stats.OnlyConcept < 1 {
		t.Errorf("OnlyConcept = %d, want >= 1", stats.OnlyConcept)
	}
}

// Sibling tags share no leaf, but both post under their taxonomy parent, so
// a db_access-only unit and a caching-only unit still meet — with evidence
// exactly the parent's IC.
func TestConceptChannelRetrievesSiblingTagsThroughAncestors(t *testing.T) {
	src := `package fix

func ReadThrough(keys []string) int {
	hits := 0
	for _, k := range keys {
		if k != "" {
			hits++
		}
	}
	return hits
}

func QueryRows(ids []int) int {
	found := 0
	for _, id := range ids {
		if id > 0 {
			found += id
		}
	}
	return found
}
`
	units := parseUnits(t, "fix.go", src)
	a := unitIndex(t, units, "ReadThrough")
	b := unitIndex(t, units, "QueryRows")
	units[a].Patterns = []string{"caching"}
	units[b].Patterns = []string{"db_access"}

	opt := DefaultOptions()
	opt.MinNodes = 8
	opt.Threshold = 0.99

	cands, _ := retrieveAll(t, units, opt)
	c, ok := findCandidate(cands, a, b)
	if !ok {
		t.Fatal("sibling-tagged pair not retrieved through the ancestor posting")
	}

	onto := ontology.Default()
	counts := map[ontology.TermID]int{ontology.ConCaching: 1, ontology.ConDBAccess: 1}
	ic := ontology.NewCorpusIC(onto, counts)
	if want := ic.Of(ontology.ConDataStoreAccess); math.Abs(c.Concept-want) > 1e-9 {
		t.Errorf("Concept evidence = %v, want IC(data_store_access) = %v", c.Concept, want)
	}
}

// Sharing a tag is necessary but not sufficient: two untagged functions have
// zero concept evidence and the channel admits nothing for them.
func TestConceptChannelIgnoresUntaggedUnits(t *testing.T) {
	src := `package fix

func A1(x int) int {
	if x > 0 {
		x++
	}
	for i := 0; i < 4; i++ {
		x += i
	}
	return x
}

func B1(y int) int {
	if y > 0 {
		y++
	}
	for j := 0; j < 4; j++ {
		y += j
	}
	return y
}
`
	units := parseUnits(t, "fix.go", src)
	opt := DefaultOptions()
	opt.MinNodes = 8

	cands, stats := retrieveAll(t, units, opt)
	if stats.ConceptPairs != 0 {
		t.Errorf("ConceptPairs = %d, want 0 for an untagged corpus", stats.ConceptPairs)
	}
	if c, ok := findCandidate(cands, 0, 1); ok && c.Concept != 0 {
		t.Errorf("untagged pair has concept evidence %v, want 0", c.Concept)
	}
}

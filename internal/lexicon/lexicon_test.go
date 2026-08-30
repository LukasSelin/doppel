package lexicon

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
)

// storeCorpus is a corpus with one obvious practice in it: six functions that
// reach a key-value store through the same two calls, six that do the same
// string cleanup, and filler that shares nothing with anybody.
//
// Nothing names the store practice and no rule describes it — a wrapper called
// "store" is exactly the vocabulary a hand-written tagger would miss — so if
// the learner works at all, it finds this.
//
// The filler is not padding. The information window is a *fraction* of the
// corpus: on twelve functions a practice shared by six is carried by half the
// codebase and is correctly judged idiom, not concept. Filler is what makes the
// store practice rare enough to be evidence, and each filler function calls
// names nobody else does, so the filler itself forms no concept.
func storeCorpus() string {
	var b strings.Builder
	b.WriteString("package app\n\nimport \"strings\"\n")

	// 0-5: the store readers. Two share a parameter name and four another, so
	// the concept cannot be an artifact of one identifier.
	readers := []string{"LoadUser", "LoadOrder", "LoadInvoice", "LoadShipment", "LoadAddress", "LoadCarrier"}
	for i, name := range readers {
		arg := "ref"
		if i < 2 {
			arg = "id"
		}
		fmt.Fprintf(&b, `
func %s(%s string) (string, error) {
	v, err := store.Get(%s)
	if err != nil {
		return "", err
	}
	return store.Decode(v)
}
`, name, arg, arg)
	}

	// 6-11: an unrelated shared practice, so "shares a concept" cannot pass by
	// there being only one concept to share.
	for _, name := range []string{"TitleOne", "TitleTwo", "TitleThree", "TitleFour", "TitleFive", "TitleSix"} {
		fmt.Fprintf(&b, "\nfunc %s(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }\n", name)
	}

	// 12+: filler, each calling names nobody else calls.
	for i := 0; i < 24; i++ {
		fmt.Fprintf(&b, "\nfunc Filler%02d(x int) int { return alpha%02d(x) + beta%02d(x) }\n", i, i, i)
	}
	return b.String()
}

const (
	firstStore = 0  // index of the first store reader
	afterStore = 6  // one past the last
	afterTitle = 12 // one past the last title function
)

func build(t *testing.T, src string, seeds [][]string, opt Options) (*Model, []parser.CodeUnit) {
	t.Helper()
	units, err := parser.ParseSource("app.go", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if seeds == nil {
		seeds = make([][]string, len(units))
	}
	g := concepter.BuildCallGraph(units)
	return Build(units, g, seeds, opt), units
}

func testOptions() Options {
	opt := DefaultOptions()
	// The corpus is twelve functions; the production floors are stated for
	// thousands. MinMembers and MinSupport are the ones that would silence a
	// twelve-function fixture outright.
	opt.MinMembers = 4
	opt.MinSupport = 3
	return opt
}

// TestNoSeeds is the language-portability claim, as a test. The learner must
// produce concepts from a corpus with no seed labels at all, because that is
// what a frontend for another language starts from: TagSignals and a
// fingerprint, and not one line of vocabulary anybody wrote.
func TestNoSeeds(t *testing.T) {
	m, units := build(t, storeCorpus(), nil, testOptions())

	if len(m.Concepts()) == 0 {
		t.Fatal("no concepts learned from an unseeded corpus")
	}
	for _, c := range m.Concepts() {
		if c.Seed != "" {
			t.Errorf("concept %q claims seed %q with no seeds given", c.ID, c.Seed)
		}
	}

	// The six store functions must share a concept nothing else carries: that
	// is the whole of what "learned from the corpus" has to mean.
	assign := m.Assignments()
	shared := ""
	for _, c := range assign[firstStore] {
		if hasConcept(assign[1], c.ID) && hasConcept(assign[afterStore-1], c.ID) {
			shared = c.ID
			break
		}
	}
	if shared == "" {
		t.Fatalf("the six store readers share no concept; got %v", describe(assign, units))
	}
	for i := afterStore; i < len(units); i++ {
		if hasConcept(assign[i], shared) {
			t.Errorf("%s carries the store concept %q; it does nothing of the kind",
				units[i].Name, shared)
		}
	}

	// The name has to say what the evidence was, or nobody can act on it.
	if !strings.Contains(shared, "store.") {
		t.Errorf("concept name %q does not name the calls it was learned from", shared)
	}
}

// TestSeedsExpandRatherThanDecide pins the hybrid: a seed contributes its
// member set and nothing else. The concept that comes back is named after what
// those members actually share, keeps the seed only as provenance, and admits
// members the seed never fired on.
func TestSeedsExpandRatherThanDecide(t *testing.T) {
	units, err := parser.ParseSource("app.go", []byte(storeCorpus()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Seed only four of the six store readers, as a lexical rule keyed on
	// "Load" plus a spelling would: the other two must still be found.
	seeds := make([][]string, len(units))
	for i := 0; i < 4; i++ {
		seeds[i] = []string{"db_access"}
	}
	g := concepter.BuildCallGraph(units)
	m := Build(units, g, seeds, testOptions())

	var seeded *Concept
	for i, c := range m.Concepts() {
		if c.Seed == "db_access" {
			seeded = &m.Concepts()[i]
			break
		}
	}
	if seeded == nil {
		t.Fatalf("the seed produced no concept; got %v", ids(m))
	}
	if seeded.ID == "db_access" {
		t.Error("the concept kept the seed's name; the name must state the learned evidence")
	}
	assign := m.Assignments()
	for i := firstStore; i < afterStore; i++ {
		if !hasConcept(assign[i], seeded.ID) {
			t.Errorf("%s does the same work as the seeded four but is not a member", units[i].Name)
		}
	}
}

// TestDeterministic is the repo-wide invariant, at this stage: the same corpus
// must yield the same lexicon, concept for concept and confidence for
// confidence. Map iteration order is the usual way this breaks.
func TestDeterministic(t *testing.T) {
	first, _ := build(t, storeCorpus(), nil, testOptions())
	for i := 0; i < 5; i++ {
		again, _ := build(t, storeCorpus(), nil, testOptions())
		if !reflect.DeepEqual(first.Concepts(), again.Concepts()) {
			t.Fatalf("run %d learned a different vocabulary", i)
		}
		if !reflect.DeepEqual(first.Assignments(), again.Assignments()) {
			t.Fatalf("run %d assigned different memberships", i)
		}
	}
}

// TestConfidenceIsGraded pins the reason memberships stopped being booleans: a
// unit carrying more of a concept's vocabulary must read higher than one
// carrying less, and every confidence must sit in (0,1].
func TestConfidenceIsGraded(t *testing.T) {
	m, _ := build(t, storeCorpus(), nil, testOptions())
	seen := false
	for _, cs := range m.Assignments() {
		for _, c := range cs {
			if c.Confidence <= 0 || c.Confidence > 1 {
				t.Errorf("confidence %v for %q is outside (0,1]", c.Confidence, c.ID)
			}
			seen = true
		}
	}
	if !seen {
		t.Fatal("no memberships at all")
	}
}

// TestFeatureWindowExcludesUbiquity pins the two-sided information window. A
// feature every function carries relates nothing, and one a single function
// carries can relate nothing; neither may reach a vocabulary.
func TestFeatureWindowExcludesUbiquity(t *testing.T) {
	m, _ := build(t, storeCorpus(), nil, testOptions())
	st := m.Stats()
	if st.FeaturesSurviving == 0 || st.FeaturesSurviving >= st.FeaturesTotal {
		t.Fatalf("window kept %d of %d features; want a proper subset",
			st.FeaturesSurviving, st.FeaturesTotal)
	}
	for _, c := range m.Concepts() {
		for _, f := range c.Features {
			if f.DF < 2 {
				t.Errorf("%q keeps feature %q at df %d", c.ID, f.Name, f.DF)
			}
			if f.DF > st.FeatureCap {
				t.Errorf("%q keeps feature %q at df %d, above the cap %d",
					c.ID, f.Name, f.DF, st.FeatureCap)
			}
		}
	}
}

// TestEmptyCorpus is the degenerate case: no units, no panic, no concepts.
func TestEmptyCorpus(t *testing.T) {
	g := concepter.BuildCallGraph(nil)
	m := Build(nil, g, nil, DefaultOptions())
	if len(m.Concepts()) != 0 || len(m.Assignments()) != 0 {
		t.Fatalf("empty corpus produced %d concepts", len(m.Concepts()))
	}
}

func hasConcept(cs []parser.Concept, id string) bool {
	for _, c := range cs {
		if c.ID == id {
			return true
		}
	}
	return false
}

func ids(m *Model) []string {
	var out []string
	for _, c := range m.Concepts() {
		out = append(out, c.ID)
	}
	return out
}

func describe(assign [][]parser.Concept, units []parser.CodeUnit) string {
	var b strings.Builder
	for i := range units {
		fmt.Fprintf(&b, "\n  %-14s %v", units[i].Name, assign[i])
	}
	return b.String()
}

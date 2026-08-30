package culture

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
)

// Build twice on the same corpus: associations, typicalities, prototypes,
// and stats must be deeply equal — the model is pure counting.
func TestBuildDeterministic(t *testing.T) {
	units, docs := cloneAlienFixture()

	snapshot := func() string {
		m := buildOn(t, units, docs, DefaultOptions())
		var out string
		out += fmt.Sprintf("%+v\n", m.Associations())
		out += fmt.Sprintf("%+v\n", m.Stats())
		for i := range units {
			for _, tag := range parser.ConceptIDs(units[i].Concepts) {
				typ, ok := m.Typicality(i, tag)
				out += fmt.Sprintf("%d/%s %v %v %+v\n", i, tag, typ, ok, m.ChannelTypicality(i, tag))
			}
		}
		if p, ok := m.Prototype("db_access"); ok {
			out += fmt.Sprintf("%+v\n", p)
		}
		return out
	}

	first := snapshot()
	for run := 0; run < 25; run++ {
		if got := snapshot(); got != first {
			t.Fatalf("run %d diverged:\n%s\nvs first:\n%s", run, got, first)
		}
	}

	a := buildOn(t, units, docs, DefaultOptions())
	b := buildOn(t, units, docs, DefaultOptions())
	if !reflect.DeepEqual(a.Associations(), b.Associations()) {
		t.Fatal("association slices not deeply equal across builds")
	}
	if a.Stats() != b.Stats() {
		t.Fatal("stats differ across builds")
	}
}

func TestBuildEmptyCorpus(t *testing.T) {
	m := buildOn(t, nil, nil, DefaultOptions())
	if s := m.Stats(); s != (Stats{}) {
		t.Errorf("empty corpus stats = %+v, want zero", s)
	}
	if m.Associations() != nil {
		t.Errorf("empty corpus has associations: %+v", m.Associations())
	}
}

// BaseRate is what lets a caller tell house style from the ambient properties
// of Go. A prototype says a fraction of a concept's members do something; only
// the corpus rate says whether that is a fact about the concept or about the
// language.
func TestBaseRateCountsTheWholeCorpus(t *testing.T) {
	units, docs := cloneAlienFixture()
	m := buildOn(t, units, docs, DefaultOptions())

	// Every unit in the fixture has a package, so a package's rate is its share
	// of the corpus and never zero for a package that exists.
	seen := map[string]bool{}
	for _, u := range units {
		if u.Package != "" {
			seen[u.Package] = true
		}
	}
	for pkg := range seen {
		r, ok := m.BaseRate("package", pkg)
		if !ok {
			t.Fatalf("BaseRate has no package channel")
		}
		if r <= 0 || r > 1 {
			t.Errorf("BaseRate(package, %q) = %v, want a fraction in (0,1]", pkg, r)
		}
	}

	// A feature nothing carries rates zero rather than reporting absence as an
	// unmodeled channel — the distinction matters to a caller dividing by it.
	if r, ok := m.BaseRate("calls", "nothing.CallsThis"); !ok || r != 0 {
		t.Errorf("BaseRate for an absent feature = (%v, %v), want (0, true)", r, ok)
	}
	if _, ok := m.BaseRate("no-such-channel", "x"); ok {
		t.Error("BaseRate reported a channel it does not model")
	}
}

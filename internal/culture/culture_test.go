package culture

import (
	"fmt"
	"reflect"
	"testing"
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
			for _, tag := range units[i].Patterns {
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

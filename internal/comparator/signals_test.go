package comparator

import (
	"math"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
)

// The signal vector, weighted in declaration order, is the composite: the
// pin that the measurement sees exactly what Compare blends.
func TestSignalVectorReproducesOverlap(t *testing.T) {
	rels := ontology.Default().ScoredRelations()
	if len(rels) != SignalCount {
		t.Fatalf("%d scored relations, SignalCount is %d", len(rels), SignalCount)
	}
	for i, rel := range rels {
		if SignalOrder[i] != rel {
			t.Errorf("SignalOrder[%d] = %s, ScoredRelations has %s", i, SignalOrder[i], rel)
		}
	}
	docs := []struct{ a, b concepter.ConceptDoc }{
		{
			concepter.ConceptDoc{Name: "foo", Package: "pkg", Exported: true, Role: "utility",
				Callees: []string{"bar", "baz"}, Callers: []string{"main"}, Patterns: []string{"retry", "http_call"},
				CallerPackages: []string{"cmd"}, CalleePackages: []string{"util"}, Neighborhood: []string{"pkg.x", "pkg.y"}},
			concepter.ConceptDoc{Name: "foo2", Package: "pkg", Exported: true, Role: "utility",
				Callees: []string{"bar", "qux"}, Callers: []string{"main", "other"}, Patterns: []string{"retry", "db_access"},
				CallerPackages: []string{"cmd", "api"}, CalleePackages: []string{"util"}, Neighborhood: []string{"pkg.x", "pkg.z"}},
		},
		{
			concepter.ConceptDoc{Name: "alpha", Package: "pkgA", Exported: true, Role: "leaf", Callees: []string{"x"}, Patterns: []string{"retry"}},
			concepter.ConceptDoc{Name: "beta", Package: "pkgB", Exported: false, Role: "orchestrator", Callees: []string{"y"}, Patterns: []string{"db_access"}},
		},
	}
	o := ontology.Default()
	for _, d := range docs {
		ev := Compare(d.a, d.b)
		v := SignalVector(ev, d.a, d.b)
		var sum float64
		for i, rel := range rels {
			sum += o.Weight(rel) * v[i]
		}
		if math.Abs(sum-ev.OverlapScore) > 1e-12 {
			t.Errorf("%s/%s: weighted signal vector = %v, OverlapScore = %v", d.a.Name, d.b.Name, sum, ev.OverlapScore)
		}
	}
}

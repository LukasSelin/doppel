package analyzer

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
)

func funcUnit(pkg, name string) parser.CodeUnit {
	return parser.CodeUnit{Package: pkg, Name: name, File: pkg + "/x.go"}
}

func TestSubsystemCopies(t *testing.T) {
	a, b := funcUnit("concepter", "QualifiedName"), funcUnit("reporter", "qualifiedName")
	own := PairContext{
		CallersA:    []string{"cmd.run", "family.key", "mapper.Map"},
		CallerPkgsA: []string{"cmd", "family", "mapper"},
		CallersB:    []string{"reporter.Print"},
		CallerPkgsB: []string{"reporter"},
	}
	k := SubsystemCopies(a, b, 0.9, ForkShapeFloor, own)
	if k == nil || k.Kind != KindSubsystemCopies {
		t.Fatalf("got %+v, want subsystem copies", k)
	}
	if k.Names[0] != "concepter.QualifiedName" || k.CallerCounts[0] != 3 || k.CallerCounts[1] != 1 {
		t.Errorf("facts: %+v", k)
	}

	shared := own
	shared.CallerPkgsB = []string{"cmd", "reporter"}
	cases := []struct {
		name  string
		a, b  parser.CodeUnit
		score float64
		ctx   PairContext
	}{
		{"below the fork floor", a, b, ForkShapeFloor - 0.01, own},
		{"one package", a, funcUnit("concepter", "qualifiedName"), 0.9, own},
		{"a caller package in common", a, b, 0.9, shared},
		{"one side uncalled", a, b, 0.9, PairContext{CallersA: own.CallersA, CallerPkgsA: own.CallerPkgsA}},
		{"both private to their package", a, b, 0.9, PairContext{
			CallersA: []string{"concepter.x"}, CallerPkgsA: []string{"concepter"},
			CallersB: own.CallersB, CallerPkgsB: own.CallerPkgsB,
		}},
		{"methods", methodUnit("concepter", "*T", "Name"), methodUnit("reporter", "*U", "Name"), 0.9, own},
		{"one side calls the other", a, b, 0.9, PairContext{
			CallersA: own.CallersA, CallerPkgsA: own.CallerPkgsA,
			CallersB: []string{"concepter.QualifiedName"}, CallerPkgsB: []string{"concepter"},
		}},
	}
	for _, c := range cases {
		if k := SubsystemCopies(c.a, c.b, c.score, ForkShapeFloor, c.ctx); k != nil {
			t.Errorf("%s: labelled %+v", c.name, k)
		}
	}

	// It outranks the shape-level lens kinds: two copies whose callees
	// differ are still two copies.
	a.Callees, b.Callees = []string{"p"}, []string{"q"}
	if k := ClassifyPairIn(a, b, 0.9, ForkShapeFloor, own); k == nil || k.Kind != KindSubsystemCopies {
		t.Errorf("got %+v, want subsystem copies before different calls", k)
	}
}

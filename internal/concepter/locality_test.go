package concepter

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
)

func localityFixture(t *testing.T) ([]parser.CodeUnit, *Graph) {
	t.Helper()
	src := `package p

func Probe() { Mid() }
func Mid()   { Leaf() }
func Leaf()  {}
func Far()   {}
`
	units, err := parser.ParseSource("p.go", []byte(src))
	if err != nil || len(units) != 4 {
		t.Fatalf("fixture: units=%d err=%v", len(units), err)
	}
	return units, BuildCallGraph(units)
}

func TestLocalityGradesByNeighborhoodMembership(t *testing.T) {
	units, g := localityFixture(t)
	ball := NeighborhoodSet(g, units[0]) // Probe's ball: {Mid, Leaf}
	if len(ball) != 2 {
		t.Fatalf("probe ball = %v, want {p.Mid, p.Leaf}", ball)
	}

	// Mid is in the ball and its own ball covers it entirely: maximal.
	if got := Locality(ball, g, units[1]); got != 1.0 {
		t.Errorf("Locality(Mid) = %v, want 1.0", got)
	}
	// Leaf is in the ball; its ball {Mid, Probe} adds Mid: also full coverage.
	if got := Locality(ball, g, units[2]); got != 1.0 {
		t.Errorf("Locality(Leaf) = %v, want 1.0", got)
	}
	// Far shares nothing.
	if got := Locality(ball, g, units[3]); got != 0.0 {
		t.Errorf("Locality(Far) = %v, want 0.0", got)
	}
}

func TestLocalityEmptyBallIsNeutral(t *testing.T) {
	units, g := localityFixture(t)
	if got := Locality(nil, g, units[1]); got != 0.0 {
		t.Errorf("Locality with empty probe ball = %v, want 0.0", got)
	}
}

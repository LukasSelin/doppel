package comparator

import (
	"strings"
	"testing"

	"github.com/lukse/doppel/internal/concepter"
	"github.com/lukse/doppel/internal/ontology"
)

func nbrDoc(name string, neighborhood ...string) concepter.ConceptDoc {
	return concepter.ConceptDoc{Name: name, Package: "p", Role: "leaf", Neighborhood: neighborhood}
}

// The signal is weight × overlap ratio, on top of whatever else the docs share.
func TestNeighborhoodSignalRaisesScoreByWeightTimesRatio(t *testing.T) {
	base := Compare(nbrDoc("a"), nbrDoc("b"))
	half := Compare(
		nbrDoc("a", "p.x", "p.y"),
		nbrDoc("b", "p.x", "p.z"),
	)
	full := Compare(
		nbrDoc("a", "p.x", "p.y"),
		nbrDoc("b", "p.x", "p.y"),
	)

	w := ontology.Default().Weight(ontology.RelSharesNeighborhood)
	if got, want := half.OverlapScore-base.OverlapScore, w*0.5; !approx(got, want) {
		t.Errorf("half overlap raised the score by %v, want %v", got, want)
	}
	if got, want := full.OverlapScore-base.OverlapScore, w*1.0; !approx(got, want) {
		t.Errorf("full overlap raised the score by %v, want %v", got, want)
	}
	if half.NeighborhoodOverlap != 0.5 || len(half.SharedNeighborhood) != 1 {
		t.Errorf("overlap = %v with %v shared, want 0.5 with 1", half.NeighborhoodOverlap, half.SharedNeighborhood)
	}
}

// Empty neighborhoods contribute nothing — this is what keeps every hand-built
// regression doc at its pinned score.
func TestNeighborhoodEmptyContributesZero(t *testing.T) {
	ev := Compare(nbrDoc("a"), nbrDoc("b"))
	if ev.NeighborhoodOverlap != 0 || len(ev.SharedNeighborhood) != 0 {
		t.Errorf("empty neighborhoods scored %v / %v", ev.NeighborhoodOverlap, ev.SharedNeighborhood)
	}
	one := Compare(nbrDoc("a", "p.x"), nbrDoc("b"))
	if one.NeighborhoodOverlap != 0 {
		t.Errorf("one-sided neighborhood scored %v, want 0", one.NeighborhoodOverlap)
	}
}

// If a calls b, then b is in a's ball but never in its own — without the
// counterpart exclusion every directly-connected pair pays a systematic
// penalty in the symmetric difference.
func TestNeighborhoodExcludesTheComparedCounterpart(t *testing.T) {
	// a and b are adjacent: b appears in a's neighborhood and vice versa, and
	// both also sit near p.shared.
	a := nbrDoc("a", "p.b", "p.shared")
	b := nbrDoc("b", "p.a", "p.shared")
	ev := Compare(a, b)
	// After dropping each other, both neighborhoods are exactly {p.shared}.
	if ev.NeighborhoodOverlap != 1.0 {
		t.Errorf("overlap = %v, want 1.0 once the pair itself is excluded", ev.NeighborhoodOverlap)
	}
	if len(ev.SharedNeighborhood) != 1 || ev.SharedNeighborhood[0] != "p.shared" {
		t.Errorf("SharedNeighborhood = %v, want [p.shared]", ev.SharedNeighborhood)
	}
}

// Context, not a merge signal: neighborhood overlap must never raise the
// signal count.
func TestNeighborhoodIsNotACountableSignal(t *testing.T) {
	without := Compare(nbrDoc("a"), nbrDoc("b"))
	with := Compare(
		nbrDoc("a", "p.x", "p.y", "p.z"),
		nbrDoc("b", "p.x", "p.y", "p.z"),
	)
	if countSignals(with) != countSignals(without) {
		t.Errorf("neighborhood overlap changed the signal count: %d vs %d",
			countSignals(with), countSignals(without))
	}
}

func TestNeighborhoodReasonIsCountOnly(t *testing.T) {
	ev := Compare(
		nbrDoc("a", "p.x", "p.y"),
		nbrDoc("b", "p.x", "p.y"),
	)
	var bullet string
	for _, r := range ev.Reasons {
		if strings.Contains(r, "neighborhood") {
			bullet = r
		}
	}
	if bullet == "" {
		t.Fatalf("no neighborhood bullet in %v", ev.Reasons)
	}
	if want := "overlapping call-graph neighborhoods (1.00): 2 shared"; bullet != want {
		t.Errorf("bullet = %q, want %q", bullet, want)
	}
	if strings.Contains(bullet, "p.x") {
		t.Errorf("bullet leaks neighbor names: %q", bullet)
	}
}

func TestNeighborhoodSignalIsSymmetric(t *testing.T) {
	a := nbrDoc("a", "p.b", "p.x", "p.y")
	b := nbrDoc("b", "p.a", "p.x")
	ab, ba := Compare(a, b), Compare(b, a)
	if ab.NeighborhoodOverlap != ba.NeighborhoodOverlap {
		t.Errorf("asymmetric: %v vs %v", ab.NeighborhoodOverlap, ba.NeighborhoodOverlap)
	}
	if ab.OverlapScore != ba.OverlapScore {
		t.Errorf("composite asymmetric: %v vs %v", ab.OverlapScore, ba.OverlapScore)
	}
}

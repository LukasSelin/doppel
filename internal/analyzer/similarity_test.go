package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
)

const fixture = `package fixture

func Sum(nums []int) int {
	total := 0
	for _, n := range nums {
		if n > 0 {
			total += n
		}
	}
	return total
}

func Total(values []int) int {
	acc := 0
	for _, v := range values {
		if v > 0 {
			acc += v
		}
	}
	return acc
}

func Serve(addr string) error {
	srv := newServer(addr)
	defer srv.Close()
	go srv.Listen()
	select {
	case <-srv.Done():
		return nil
	}
}

func (s *Server) Addr() string { return s.addr }

func (s *Server) Host() string { return s.host }
`

// parseFixture returns the fixture units in declaration order:
// Sum, Total, Serve, Addr, Host.
func parseFixture(t *testing.T) []parser.CodeUnit {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.go")
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	units, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if len(units) != 5 {
		t.Fatalf("parsed %d units, want 5", len(units))
	}
	return units
}

func TestFindSimilarMatchesRenamedCopy(t *testing.T) {
	units := parseFixture(t)
	pairs := FindSimilar(units, 0.9, 0, 12)

	if len(pairs) != 1 {
		t.Fatalf("got %d pairs, want 1: %+v", len(pairs), pairs)
	}
	p := pairs[0]
	if p.A.Name != "Sum" || p.B.Name != "Total" {
		t.Errorf("matched %s/%s, want Sum/Total", p.A.Name, p.B.Name)
	}
	// Indices must address the same units the pair carries, since the caller
	// uses them to look up parallel concept docs.
	if units[p.AIdx].Name != p.A.Name || units[p.BIdx].Name != p.B.Name {
		t.Errorf("indices %d/%d point at %s/%s, want %s/%s",
			p.AIdx, p.BIdx, units[p.AIdx].Name, units[p.BIdx].Name, p.A.Name, p.B.Name)
	}
	if p.Breakdown.Score != p.Score {
		t.Errorf("Breakdown.Score = %v, Score = %v; want equal", p.Breakdown.Score, p.Score)
	}
}

// The trivial accessors match each other perfectly, so the guard is the only
// thing keeping them out of the report.
func TestFindSimilarMinNodesExcludesTrivialAccessors(t *testing.T) {
	units := parseFixture(t)

	withGuard := FindSimilar(units, 0.9, 0, 12)
	for _, p := range withGuard {
		if p.A.Name == "*Server.Addr" || p.B.Name == "*Server.Addr" {
			t.Errorf("trivial accessor survived min-nodes=12: %s/%s", p.A.Name, p.B.Name)
		}
	}

	withoutGuard := FindSimilar(units, 0.9, 0, 0)
	var found bool
	for _, p := range withoutGuard {
		if p.A.Name == "*Server.Addr" && p.B.Name == "*Server.Host" {
			found = true
		}
	}
	if !found {
		t.Errorf("min-nodes=0 should surface the accessor pair, got %+v", withoutGuard)
	}
}

func TestFindSimilarThresholdAndTopN(t *testing.T) {
	units := parseFixture(t)

	if pairs := FindSimilar(units, 1.01, 0, 0); len(pairs) != 0 {
		t.Errorf("threshold above 1.0 returned %d pairs, want 0", len(pairs))
	}

	all := FindSimilar(units, 0.0, 0, 0)
	if len(all) < 2 {
		t.Fatalf("threshold 0 returned %d pairs, want at least 2", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].Score < all[i].Score {
			t.Fatalf("pairs not sorted descending at %d: %v < %v", i, all[i-1].Score, all[i].Score)
		}
	}

	if capped := FindSimilar(units, 0.0, 2, 0); len(capped) != 2 {
		t.Errorf("topN=2 returned %d pairs, want 2", len(capped))
	}
}

func TestFindSimilarEmptyInput(t *testing.T) {
	if pairs := FindSimilar(nil, 0.5, 10, 12); pairs != nil {
		t.Errorf("FindSimilar(nil) = %+v, want nil", pairs)
	}
}

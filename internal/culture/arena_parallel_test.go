package culture

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/LukasSelin/doppel/internal/parallel"
	"github.com/LukasSelin/doppel/internal/parser"
)

// TestArenaParallelMatchesSequential pins the fan-out in buildArenas against
// the same corpus built one unit at a time.
//
// TestArenaStatsAndDeterminism already re-runs a build 25 times, but its
// fixture is twelve units — below minUnitsPerWorker, so it stays on the
// sequential path and cannot see this. The fixture here is deliberately past
// the floor, and the sequential reference is obtained by pinning GOMAXPROCS to
// 1, which is what makes parallel.Workers return 1 and the primitive run the
// loop inline.
func TestArenaParallelMatchesSequential(t *testing.T) {
	// Past minUnitsPerWorker*2, or the "parallel" build is not parallel.
	const n = 1200
	var units []parser.CodeUnit
	for i := range n {
		switch i % 4 {
		case 0:
			units = append(units, sqlUnit(fmt.Sprintf("db%04d", i), "store", "db_access"))
		case 1:
			units = append(units, unit(fmt.Sprintf("tx%04d", i), "store", "transaction", "caching"))
		case 2:
			units = append(units, sqlUnit(fmt.Sprintf("mix%04d", i), "web", "db_access", "error_wrapping"))
		default:
			units = append(units, unit(fmt.Sprintf("plain%04d", i), "web"))
		}
	}
	docs := docsWithRole(len(units), "leaf")

	if got := parallel.Workers(n, minUnitsPerWorker); got < 2 {
		t.Skipf("%d workers on this machine; the parallel path is not exercised", got)
	}

	snapshot := func(m *Model) string {
		out := fmt.Sprintf("%+v\n", m.Stats())
		for i := range units {
			p, ok := m.ArenaProfile(i)
			out += fmt.Sprintf("%d %v %+v\n", i, ok, p)
		}
		return out
	}

	parallelBuild := snapshot(buildOn(t, units, docs, DefaultOptions()))

	prev := runtime.GOMAXPROCS(1)
	sequential := snapshot(buildOn(t, units, docs, DefaultOptions()))
	runtime.GOMAXPROCS(prev)

	if parallelBuild != sequential {
		t.Error("the parallel arena build disagrees with the sequential one")
		// Report the first differing line rather than two 1200-line dumps.
		pl, sl := splitLines(parallelBuild), splitLines(sequential)
		for i := range max(len(pl), len(sl)) {
			var a, b string
			if i < len(pl) {
				a = pl[i]
			}
			if i < len(sl) {
				b = sl[i]
			}
			if a != b {
				t.Fatalf("first difference at line %d:\nparallel:   %s\nsequential: %s", i, a, b)
			}
		}
	}

	// The counters must still be a partition of what was profiled, whichever
	// path filled them.
	s := buildOn(t, units, docs, DefaultOptions()).Stats()
	if s.ArenaProfiled == 0 {
		t.Fatal("fixture broken: nothing profiled")
	}
	if sum := s.ArenaDominance + s.ArenaCoalition + s.ArenaConflict + s.ArenaWeak; s.ArenaProfiled != sum {
		t.Errorf("profiled %d != state sum %d", s.ArenaProfiled, sum)
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := range len(s) {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

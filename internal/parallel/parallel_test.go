package parallel

import (
	"runtime"
	"sync/atomic"
	"testing"
)

func TestBlocksCoversEveryIndexExactlyOnce(t *testing.T) {
	for _, n := range []int{0, 1, 7, 63, 64, 65, 1000} {
		for _, block := range []int{1, 4, 64} {
			seen := make([]int32, n)
			Blocks(n, block, 8, func(i int) { atomic.AddInt32(&seen[i], 1) })
			for i, c := range seen {
				if c != 1 {
					t.Fatalf("n=%d block=%d: index %d ran %d times, want 1", n, block, i, c)
				}
			}
		}
	}
}

// Each worker must get its own state, and the same one for every item it runs.
func TestBlocksWithGivesEachWorkerItsOwnState(t *testing.T) {
	const n = 4096
	var made atomic.Int64
	owner := make([]int64, n)
	BlocksWith(n, 4, 8,
		func() int64 { return made.Add(1) },
		func(state int64, i int) { owner[i] = state })

	if made.Load() < 2 && Workers(n, 8) > 1 {
		t.Fatalf("newState called %d times with %d workers", made.Load(), Workers(n, 8))
	}
	if int(made.Load()) > Workers(n, 8) {
		t.Errorf("newState called %d times, more than the %d workers", made.Load(), Workers(n, 8))
	}
	for i := range owner {
		if owner[i] == 0 {
			t.Fatalf("index %d never ran", i)
		}
	}
}

// A stateless nil newState must not panic and must still cover the range.
func TestBlocksNilState(t *testing.T) {
	got := make([]int, 100)
	BlocksWith(100, 8, 4, nil, func(_ struct{}, i int) { got[i] = i })
	for i := range got {
		if got[i] != i {
			t.Fatalf("index %d = %d", i, got[i])
		}
	}
}

func TestWorkersFloor(t *testing.T) {
	if got := Workers(10, 512); got != 1 {
		t.Errorf("Workers(10, 512) = %d, want 1 (below the floor stays sequential)", got)
	}
	if got := Workers(1<<20, 1); got != runtime.GOMAXPROCS(0) {
		t.Errorf("Workers(huge, 1) = %d, want GOMAXPROCS %d", got, runtime.GOMAXPROCS(0))
	}
	if got := Workers(0, 0); got < 1 {
		t.Errorf("Workers(0,0) = %d, want at least 1", got)
	}
}

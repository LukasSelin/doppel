// Package parallel is the one fan-out primitive the pipeline shares.
//
// It exists because the same loop had been written three times — file parsing,
// pair comparison, and the per-unit concept arenas — and the third copy could
// not have been shared any other way: internal/culture cannot import cmd,
// where the first two live. That is the situation the module's other
// deliberately shared helpers (parser.ShouldSkipDir, snapshot.RelSlash,
// internal/clique) were extracted for, and doppel finds this kind of clone on
// itself.
//
// It imports nothing from this module, the same rule syntax, ontology and
// clique follow.
//
// # What it does not do
//
// It has no opinion about ordering, and it cannot give one: determinism in
// every caller comes from writing results at a fixed index and reading them
// back in index order afterwards, never from the order work completes in. A
// caller that accumulates into shared state instead is using the wrong tool.
package parallel

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// BlocksWith runs fn(state, i) for every i in [0, n), spread across workers
// that each claim block consecutive indices per atomic bump.
//
// Each worker calls newState once and passes that value to every fn it runs, so
// a caller needing per-goroutine scratch — a reusable buffer, a forked scorer —
// gets it without the primitive knowing what it is. newState may be nil when
// the work is stateless; fn then receives the zero value.
//
// Work is claimed dynamically rather than partitioned up front because the
// items are not equally expensive: measured on the comparison stage, a static
// contiguous split reached 9.7x on 24 threads where this reaches 13.5x, the
// difference being workers left idle on a tail of expensive items. A channel of
// items balances just as well and measured slower (9.9x), paying a send and a
// receive per item where this pays one atomic per block.
//
// fn must be safe to call from several goroutines and must confine its writes
// to index i.
func BlocksWith[T any](n, block, minPerWorker int, newState func() T, fn func(state T, i int)) {
	if n <= 0 {
		return
	}
	if block < 1 {
		block = 1
	}
	if workers := Workers(n, minPerWorker); workers > 1 {
		var next atomic.Int64
		var wg sync.WaitGroup
		wg.Add(workers)
		for range workers {
			go func() {
				defer wg.Done()
				var state T
				if newState != nil {
					state = newState()
				}
				for {
					lo := int(next.Add(int64(block))) - block
					if lo >= n {
						return
					}
					for i := lo; i < min(lo+block, n); i++ {
						fn(state, i)
					}
				}
			}()
		}
		wg.Wait()
		return
	}
	// Sequential: one state, no goroutine, so a small corpus pays nothing for
	// the machinery. This is a real case — conc is 81 functions.
	var state T
	if newState != nil {
		state = newState()
	}
	for i := range n {
		fn(state, i)
	}
}

// Blocks is BlocksWith for work that needs no per-worker state.
func Blocks(n, block, minPerWorker int, fn func(i int)) {
	BlocksWith(n, block, minPerWorker, nil, func(_ struct{}, i int) { fn(i) })
}

// Workers is how many goroutines BlocksWith would use for n items: one per
// core, capped so that each has at least minPerWorker items to do. It is
// exported because callers test whether a fixture is large enough to exercise
// the parallel path at all.
func Workers(n, minPerWorker int) int {
	w := runtime.GOMAXPROCS(0)
	if minPerWorker > 0 {
		if byLoad := n / minPerWorker; byLoad < w {
			w = byLoad
		}
	}
	if w < 1 {
		return 1
	}
	return w
}

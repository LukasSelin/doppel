package cmd

import (
	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parallel"
)

// compareAll attaches structural evidence to every candidate pair, in parallel.
//
// # Why this stage and not the others
//
// Scoring a pair reads two ConceptDocs and writes one Evidence pointer at a
// known index: no accumulation, no shared counter, no order dependence. The one
// piece of mutable state the scorer had — the vocabulary's profile merge
// buffers — is handed out per worker by comparator.Fork. So the output is
// identical rather than merely equivalent: every pair is scored by the same
// pure function and each result lands in the slot the sequential loop would
// have put it in. Determinism is a property of writing by index, and holds
// whatever order the scheduler runs the blocks in. Verified byte-identical
// `--format json` on all seven pinned corpora.
//
// # Why an atomic block counter, and not a channel or a static split
//
// All three were measured on moby (18 461 pairs, 24 threads, best of five —
// TestCompareParallelShape, guard DOPPEL_BENCH_COMPARE). The fan-out itself is
// parallel.BlocksWith, which carries that measurement's conclusion; what stays
// here is why this stage may be parallel at all, and the two constants:
//
//	workers=8    static 0.117s (5.5x)   chan 0.097s (6.6x)   atomic 0.093s (7.0x)
//	workers=16   static 0.084s (7.7x)   chan 0.066s (9.7x)   atomic 0.056s (11.5x)
//	workers=24   static 0.067s (9.7x)   chan 0.065s (9.9x)   atomic 0.048s (13.5x)
//
// The static split loses because pairs are not equally expensive — a pair whose
// two sides carry three concepts each costs several times one with a single
// concept apiece — so contiguous chunks leave workers idle at the end. Both
// dynamic schemes fix that; the channel then pays a send and a receive on a
// contended queue per unit of work, and the atomic counter does not. A channel
// would be the right tool for a stream of unknown length, and this is a slice
// whose length is known before the first goroutine starts.
//
// blockSize is the same measurement: 8 and 32 read 11.5x, 64 reads 12.4x, 128
// falls back to 11.4x and 4096 collapses to 4.0x as the tail block strands a
// worker. Anywhere in 8..128 is within noise of the best; 64 is the middle of
// that plateau rather than a tuned value.
func compareAll(pairs []analyzer.SimilarPair, docs []concepter.ConceptDoc, comp *comparator.Comparator) {
	parallel.BlocksWith(len(pairs), blockSize, minPairsPerWorker,
		// One forked comparator per worker: shares the ontology, the IC and the
		// interned vocabulary, owns its own scratch.
		comp.Fork,
		func(c *comparator.Comparator, i int) {
			ev := c.Compare(docs[pairs[i].AIdx], docs[pairs[i].BIdx])
			pairs[i].Evidence = &ev
		})
}

// blockSize is how many consecutive pairs a worker claims per atomic bump.
const blockSize = 64

// minPairsPerWorker keeps a small corpus sequential. Below it the goroutine and
// fork costs are a larger share of the stage than the parallelism saves, and
// conc (81 functions, 152 pairs) is a real corpus rather than a hypothetical.
const minPairsPerWorker = 512

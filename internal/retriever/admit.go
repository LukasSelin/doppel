package retriever

import "github.com/LukasSelin/doppel/internal/parallel"

// concatByIndex runs one function's admission turn for every unit across all
// cores and returns the results concatenated in unit order.
//
// All three channels share this shape: admitFor(a) is a pure function of the
// index, the corpus statistics and the two memos, which are the only mutable
// state and are safe for concurrent use (see pairMemo). Nothing accumulates,
// so the loop partitions with nothing to synchronise.
//
// The order is preserved even though nothing downstream needs it: `admit` folds
// these into a map and `evaluate` re-sorts the union's keys, so the output
// would be identical from any order. Keeping it costs one slice of slices and
// removes a whole class of question about whether that is still true.
//
// admitBlock is smaller than the parse and compare blocks because one turn is
// much more expensive than one pair — it walks a unit's postings and scores
// every candidate neighbour — so a smaller block balances better and the atomic
// is still nothing beside the work.
const admitBlock = 8

// minUnitsPerAdmitWorker keeps a small corpus sequential, as the other stages do.
const minUnitsPerAdmitWorker = 64

func concatByIndex(n int, turn func(a int) []pairKey) []pairKey {
	if n == 0 {
		return nil
	}
	per := make([][]pairKey, n)
	parallel.Blocks(n, admitBlock, minUnitsPerAdmitWorker, func(a int) {
		per[a] = turn(a)
	})
	total := 0
	for _, p := range per {
		total += len(p)
	}
	if total == 0 {
		return nil
	}
	out := make([]pairKey, 0, total)
	for _, p := range per {
		out = append(out, p...)
	}
	return out
}

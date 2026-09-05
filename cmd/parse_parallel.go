package cmd

import (
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/LukasSelin/doppel/internal/parser"
)

// parseAll parses every admitted path and returns the units in path order.
//
// # Order is the contract
//
// Everything downstream is positional — docs[i] describes units[i], a pair
// carries AIdx/BIdx into that slice, and the snapshot's pair sides are ordered
// by name precisely because indices are walk positions. So the parallelism here
// is allowed exactly one freedom, which file gets parsed when, and none at all
// over where its units land: results go into a slot per path and are
// concatenated in path order afterwards. filepath.WalkDir is lexical, so that
// is the same order the sequential loop produced.
//
// Warnings are collected and printed in the same path order rather than as
// they happen, for two reasons: the progress writer is not safe for concurrent
// use, and a run's stderr is read by the examples wrapper, so reordering it
// would be a visible change with no cause behind it.
//
// # Safety
//
// parser.Parse builds a token.FileSet per call and the frontend registry is
// written only by package init, so nothing here is shared but the results
// slice, which each index is written by exactly one goroutine. Verified under
// -race over the ladder.
func parseAll(paths []string, progress io.Writer) []parser.CodeUnit {
	if len(paths) == 0 {
		return nil
	}
	type result struct {
		units []parser.CodeUnit
		err   error
	}
	results := make([]result, len(paths))

	parse := func(i int) { results[i].units, results[i].err = parser.Parse(paths[i]) }
	if workers := parseWorkers(len(paths)); workers <= 1 {
		for i := range paths {
			parse(i)
		}
	} else {
		var next atomic.Int64
		var wg sync.WaitGroup
		wg.Add(workers)
		for w := 0; w < workers; w++ {
			go func() {
				defer wg.Done()
				for {
					lo := int(next.Add(parseBlock)) - parseBlock
					if lo >= len(paths) {
						return
					}
					for i := lo; i < min(lo+parseBlock, len(paths)); i++ {
						parse(i)
					}
				}
			}()
		}
		wg.Wait()
	}

	n := 0
	for i := range results {
		n += len(results[i].units)
	}
	units := make([]parser.CodeUnit, 0, n)
	for i := range results {
		if results[i].err != nil {
			// Unreadable file or a frontend that refused it: warned and
			// skipped, never fatal, exactly as the sequential walk did.
			fmt.Fprintf(progress, "  warn: %s: %v\n", paths[i], results[i].err)
			continue
		}
		units = append(units, results[i].units...)
	}
	return units
}

// parseBlock is how many consecutive files a worker claims per atomic bump.
// One file is a few hundred microseconds against an atomic of a few
// nanoseconds, so the counter is free at any block size and the only thing
// that matters is not stranding a worker on a tail of large files; 4 keeps the
// bookkeeping negligible without coarsening the balance.
const parseBlock = 4

// minFilesPerWorker keeps a small tree sequential, where the goroutines would
// cost more than the walk does.
const minFilesPerWorker = 16

func parseWorkers(n int) int {
	w := runtime.GOMAXPROCS(0)
	if byLoad := n / minFilesPerWorker; byLoad < w {
		w = byLoad
	}
	if w < 1 {
		return 1
	}
	return w
}

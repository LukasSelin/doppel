package cmd

import (
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
)

// compareAllChan is the channel-per-pair variant, kept only so the shape
// comparison in TestCompareParallelShape is a measurement rather than an
// assertion. It is not used by the pipeline.
func compareAllChan(pairs []analyzer.SimilarPair, docs []concepter.ConceptDoc, comp *comparator.Comparator, workers int) {
	idx := make(chan int, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := comp.Fork()
			for i := range idx {
				ev := c.Compare(docs[pairs[i].AIdx], docs[pairs[i].BIdx])
				pairs[i].Evidence = &ev
			}
		}()
	}
	for i := range pairs {
		idx <- i
	}
	close(idx)
	wg.Wait()
}

// compareAllAtomic hands out contiguous blocks through an atomic counter:
// dynamic balancing like the channel, without a queue on the hot path.
func compareAllAtomic(pairs []analyzer.SimilarPair, docs []concepter.ConceptDoc, comp *comparator.Comparator, workers, block int) {
	var next atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := comp.Fork()
			for {
				lo := int(next.Add(int64(block))) - block
				if lo >= len(pairs) {
					return
				}
				for i := lo; i < min(lo+block, len(pairs)); i++ {
					ev := c.Compare(docs[pairs[i].AIdx], docs[pairs[i].BIdx])
					pairs[i].Evidence = &ev
				}
			}
		}()
	}
	wg.Wait()
}

func compareAllPool(pairs []analyzer.SimilarPair, docs []concepter.ConceptDoc, comp *comparator.Comparator, workers int) {
	var wg sync.WaitGroup
	chunk := (len(pairs) + workers - 1) / workers
	for start := 0; start < len(pairs); start += chunk {
		end := min(start+chunk, len(pairs))
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			c := comp.Fork()
			for i := lo; i < hi; i++ {
				ev := c.Compare(docs[pairs[i].AIdx], docs[pairs[i].BIdx])
				pairs[i].Evidence = &ev
			}
		}(start, end)
	}
	wg.Wait()
}

// TestCompareParallelShape times the pool against the channel variant on a real
// corpus, and checks that both reproduce the sequential evidence exactly.
//
//	DOPPEL_BENCH_COMPARE=<corpus path> go test ./cmd/ -run TestCompareParallelShape -v
func TestCompareParallelShape(t *testing.T) {
	root := os.Getenv("DOPPEL_BENCH_COMPARE")
	if root == "" {
		t.Skip("set DOPPEL_BENCH_COMPARE=<corpus path> to run")
	}
	p := Params{
		Threshold: defaultThreshold, TopN: 20, MinNodes: defaultMinNodes,
		ChannelK: 5, MaxPerFunc: 2, TestsMode: "exclude", Generated: "exclude",
		Calibrate: defaultCalibrateRate,
	}
	res, err := index(root, p, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err = finishAnalyze(res, p, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	pairs, docs := res.Pairs, res.Docs
	if len(pairs) == 0 {
		t.Skip("no pairs")
	}
	comp := comparator.New(ontology.NewScorer(res.Onto, res.IC).WithVocabulary(res.Vocab))

	want := make([]comparator.StructuralEvidence, len(pairs))
	seq := timeIt(func() {
		for i := range pairs {
			want[i] = comp.Compare(docs[pairs[i].AIdx], docs[pairs[i].BIdx])
		}
	})
	t.Logf("%-28s %8.3fs   %d pairs, GOMAXPROCS=%d", "sequential", seq, len(pairs), runtime.GOMAXPROCS(0))

	check := func(name string) {
		for i := range pairs {
			if pairs[i].Evidence == nil || !reflect.DeepEqual(*pairs[i].Evidence, want[i]) {
				t.Fatalf("%s: pair %d differs from the sequential evidence", name, i)
			}
			pairs[i].Evidence = nil
		}
	}
	// Each variant is run repeatedly and the best time kept: these are
	// sub-second stages on a loaded desktop, so the minimum is the least
	// noisy estimator of the cost, and the spread between repeats is what
	// says whether a difference is real.
	best := func(f func()) float64 {
		b := math.Inf(1)
		for r := 0; r < 5; r++ {
			if d := timeIt(f); d < b {
				b = d
			}
		}
		return b
	}
	for _, w := range []int{4, 8, 16, runtime.GOMAXPROCS(0)} {
		if w > runtime.GOMAXPROCS(0) {
			continue
		}
		pool := best(func() { compareAllPool(pairs, docs, comp, w) })
		check(fmt.Sprintf("pool/%d", w))
		ch := best(func() { compareAllChan(pairs, docs, comp, w) })
		check(fmt.Sprintf("chan/%d", w))
		at := best(func() { compareAllAtomic(pairs, docs, comp, w, 64) })
		check(fmt.Sprintf("atomic/%d", w))
		t.Logf("workers=%-3d  static %6.3fs (%.1fx)   chan %6.3fs (%.1fx)   atomic-64 %6.3fs (%.1fx)",
			w, pool, seq/pool, ch, seq/ch, at, seq/at)
	}

	// Block size for the adopted strategy: large enough to amortise the atomic,
	// small enough that the last block cannot leave a worker idle for long.
	w := runtime.GOMAXPROCS(0)
	for _, block := range []int{8, 32, 64, 128, 512, 4096} {
		d := best(func() { compareAllAtomic(pairs, docs, comp, w, block) })
		check(fmt.Sprintf("atomic/%d", block))
		t.Logf("atomic block=%-5d %6.3fs (%.1fx)", block, d, seq/d)
	}
}

func timeIt(f func()) float64 {
	start := time.Now()
	f()
	return time.Since(start).Seconds()
}

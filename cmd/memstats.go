package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
)

// DOPPEL_MEMSTATS=1 records exact heap accounting at every pipeline stage;
// DOPPEL_MEMSTATS=<path>.json writes the same records there instead of to the
// progress writer, so two runs can be diffed by a machine.
//
// # Why this exists beside DOPPEL_HEAPPROFILE
//
// A heap profile is *sampled* — one allocation per MemProfileRate bytes, 512KB
// by default — so its inuse_space is an estimate, and the estimate moves. The
// same binary dumping the same stage three times read 129.6, 137.8 and 143.2MB:
// a 10% spread, which is larger than most changes worth making. Peak RSS is
// worse, because it also depends on when the collector happened to run: six runs
// of one binary spanned 624-755MB. A footprint change measured against either
// number alone cannot be believed.
//
// runtime.MemStats is not sampled. HeapAlloc, TotalAlloc, Mallocs and Frees are
// counters the runtime maintains exactly, so this harness answers the two
// questions a footprint change actually asks — how much is *held* at a stage,
// and how much was *allocated* to get there — with no sampling error at all.
// See TestMemStatsNoiseFloor in cmd for the measured floors.
//
// # What it costs, and why that is fine
//
// Every stage boundary forces a GC, so a run under this flag is slower and its
// collector schedule is not a normal run's. That makes RSS measured *during*
// such a run meaningless, and it is deliberately not reported: use the OS
// working set for that, knowing its spread. Unset — the default everywhere —
// nothing here runs and both streams are byte-identical.
type memRecorder struct {
	w    io.Writer
	path string // non-empty: write JSON here at the end instead of printing
	// base is TotalAlloc when recording began, so Alloc below is bytes this
	// *run* allocated rather than this *process*. The difference is invisible
	// in a normal run, which does one analysis, and decisive for a caller that
	// does several in one process — the noise-floor test read a 5GB spread on
	// what was really run 2 carrying run 1's total.
	base uint64
	recs []MemRecord
}

// MemRecord is one stage's exact accounting.
type MemRecord struct {
	Stage string `json:"stage"`
	// Live is HeapAlloc after a forced GC: bytes of reachable heap objects.
	Live uint64 `json:"live"`
	// Alloc is cumulative bytes allocated since this run's recording began. It
	// never decreases and is insensitive to when the collector ran, which makes
	// it the most reproducible number here.
	Alloc uint64 `json:"alloc"`
	// Objects is Mallocs-Frees: reachable object count, the allocation-shape
	// companion to Live — a change that halves object count without moving
	// Live has moved the GC's scanning cost, which is what scanobject reads.
	Objects uint64 `json:"objects"`
	// HeapSys is what the runtime has taken from the OS for the heap. It is the
	// closest in-process proxy for RSS and moves with collector scheduling, so
	// it is recorded but never the number to judge a change by.
	HeapSys uint64 `json:"heapSys"`
}

// The recorder is process-wide, not per timer. index() and finishAnalyze() each
// build their own stageTimer — deliberately, so `doppel query` reports a total
// no caller has to subtract — but the records are one timeline of one process,
// and two recorders writing the same JSON path would have the second silently
// overwrite the first half of the run. A single instance also means the printed
// and written forms cannot disagree about what a run did.
var (
	memOnce   sync.Once
	memShared *memRecorder
)

func newMemRecorder(progress io.Writer) *memRecorder {
	memOnce.Do(func() {
		v := os.Getenv("DOPPEL_MEMSTATS")
		if v == "" || v == "0" {
			return
		}
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		memShared = &memRecorder{w: progressOr(progress), base: ms.TotalAlloc}
		if v != "1" {
			memShared.path = v
		}
	})
	return memShared
}

func (m *memRecorder) record(stage string) {
	if m == nil {
		return
	}
	// Forced, so Live counts what is reachable rather than what has not been
	// collected yet — the same reason stopProfiling and the heap dumper do it.
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	r := MemRecord{
		Stage:   stage,
		Live:    ms.HeapAlloc,
		Alloc:   ms.TotalAlloc - m.base,
		Objects: ms.Mallocs - ms.Frees,
		HeapSys: ms.HeapSys,
	}
	m.recs = append(m.recs, r)
	if m.path == "" {
		fmt.Fprintf(m.w, "  memstats: %-22s live %8.2fMB  alloc %9.2fMB  objects %7.2fM\n",
			stage, mb(r.Live), mb(r.Alloc), float64(r.Objects)/1e6)
	}
}

// flush writes the JSON form, when a path was given. Errors are reported and
// never returned: a measurement must not fail a run.
func (m *memRecorder) flush() {
	if m == nil || m.path == "" {
		return
	}
	f, err := os.Create(m.path)
	if err != nil {
		fmt.Fprintf(m.w, "  memstats: %v\n", err)
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m.recs); err != nil {
		fmt.Fprintf(m.w, "  memstats: %v\n", err)
	}
}

func mb(b uint64) float64 { return float64(b) / (1 << 20) }

// resetMemRecorder drops the process-wide recorder so a caller can start a
// fresh one. It exists for the noise-floor test, which drives several runs in
// one process; nothing in a normal run needs it.
func resetMemRecorder() {
	memOnce = sync.Once{}
	memShared = nil
}

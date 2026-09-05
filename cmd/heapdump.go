package cmd

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
)

// DOPPEL_HEAPPROFILE=<prefix> with DOPPEL_HEAPPROFILE_AT=<stage>[,<stage>...]
// writes a heap profile at the end of the named pipeline stages, while the data
// those stages built is still reachable.
//
// # Why --memprofile cannot answer this
//
// --memprofile fires from a defer in Execute, after the pipeline's Result has
// gone out of scope, so its inuse_space reads ~24MB on a run whose peak RSS is
// 800MB. That is the right profile for "what leaked" and useless for "what does
// the corpus model cost to hold" — which is the question that decides whether a
// field is worth keeping. Measured across the ladder, `doppel query` (index
// only) peaks at 463MB of a full run's 750MB, so the retained corpus is most of
// the footprint and no exit-time profile can apportion it.
//
// # Shape
//
// Env-gated and hooked into the stage marks the timer already emits, so there
// is exactly one list of stage names in the tool and a dump cannot name a stage
// that does not exist. Like DOPPEL_TIMING it must never travel in Params or
// reach a config key: what a run cost to hold has no bearing on what it
// measured. Unset, nothing here runs and nothing is written.
//
// A dump forces a GC first, so what it reports is live rather than whatever had
// not been collected yet — the same reason stopProfiling does.
//
// Failures are reported to the pipeline's progress writer and never returned: a
// profile that cannot be written must not turn a clean analysis into a failed
// one, and a hook passes io.Discard, so this stays silent there like everything
// else.
type heapDumper struct {
	prefix string
	at     map[string]bool
}

func newHeapDumper() *heapDumper {
	prefix := os.Getenv("DOPPEL_HEAPPROFILE")
	at := os.Getenv("DOPPEL_HEAPPROFILE_AT")
	if prefix == "" || at == "" {
		return nil
	}
	d := &heapDumper{prefix: prefix, at: map[string]bool{}}
	for _, s := range strings.Split(at, ",") {
		if s = strings.TrimSpace(s); s != "" {
			d.at[s] = true
		}
	}
	if len(d.at) == 0 {
		return nil
	}
	return d
}

// dump writes a heap profile if this stage was asked for.
func (d *heapDumper) dump(stage string, w io.Writer) {
	if d == nil || !d.at[stage] {
		return
	}
	path := d.prefix + "-" + slugStage(stage) + ".heap"
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(w, "  heap: %v\n", err)
		return
	}
	defer f.Close()
	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		fmt.Fprintf(w, "  heap: %v\n", err)
		return
	}
	fmt.Fprintf(w, "  heap: wrote %s\n", path)
}

// slugStage turns a stage name into a filename fragment. Stage names are prose
// ("walk + parse", "culture + habitats") because they are read in the timing
// output first and used as a key second.
func slugStage(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			if n := b.String(); n != "" && !strings.HasSuffix(n, "-") {
				b.WriteByte('-')
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

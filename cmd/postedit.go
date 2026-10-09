package cmd

import (
	"io"
	"path/filepath"
	"sort"
	"time"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/gofront"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/snapshot"
	"github.com/spf13/cobra"
)

// postEditShapeFloor is the code-shape a match must reach before the
// post-edit hook says anything about it.
//
// It is the family edge cut and the fork floor (analyzer.ForkShapeFloor), not
// the calibrated threshold, and the difference is the point: the calibrated
// value is the 99th percentile of random pairs, which is a fine gate for
// admitting a candidate to be ranked and far too weak a claim to interrupt an
// edit with. A note here says "you just wrote a near duplicate", and 0.60 is
// the value this tool already uses wherever it states that kind of guarantee.
// The replay that measured the firing rate at this floor is
// scripts/postedit-replay.sh.
const postEditShapeFloor = analyzer.ForkShapeFloor

// postEditDeadline bounds the hook's own work. It sits under the plugin's
// 30-second timeout so the hook gives up silently rather than being killed by
// the harness, which reports a killed hook to the user as a failure.
const postEditDeadline = 20 * time.Second

var postEditMinShape float64

var hookPostEditCmd = &cobra.Command{
	Use:   "post-edit",
	Short: "Probe the functions an edit wrote against the corpus for near duplicates",
	Args:  cobra.NoArgs,
	RunE:  runHookPostEdit,
}

func init() {
	// Hidden: a measurement seam for the replay script, which records matches
	// below the shipped floor so the floor itself can be chosen. No question
	// about a codebase is answered by setting it.
	hookPostEditCmd.Flags().Float64Var(&postEditMinShape, "min-shape", postEditShapeFloor, "Code-shape a match must reach to be reported")
	_ = hookPostEditCmd.Flags().MarkHidden("min-shape")
}

// runHookPostEdit probes every function an Edit or Write created or changed
// against the corpus as it stands after the edit, and tells the model about
// the near duplicates among them.
//
// After the edit rather than before it, because before it the file on disk
// still holds the old body: a changed function's nearest match would be its
// own former self, and fixing that would mean overlaying the edit inside
// index(). After it, a plain index() reads the true post-edit corpus, and
// retriever.Probe accepts any unit index, so no pipeline stage changes.
//
// The session baseline is the "before": a function is probed when its key is
// absent from the baseline or its digest differs, and a match that was
// already a pair in the baseline is dropped — the pre-tool advisory covered
// those, and the session did not create them. Like every hook it never exits
// non-zero, never writes to stderr, and never makes a permission decision.
func runHookPostEdit(cmd *cobra.Command, args []string) error {
	in, err := readHookInput(cmd.InOrStdin())
	if err != nil || in.ToolInput.FilePath == "" {
		return emitNothing()
	}

	path := baselinePath(in.SessionID)
	base, err := readBaseline(path)
	if err != nil || base.Snapshot.Schema != snapshot.Schema {
		// No baseline means no "before", and so no way to tell a function
		// this session wrote from one it merely sits beside.
		return emitNothing()
	}
	root := base.Root
	if root == "" {
		root = resolveRoot(in.Cwd)
	}
	if mode, err := hookProbe(root); err != nil || mode != ProbeOn {
		return emitNothing()
	}
	file := in.ToolInput.FilePath
	if !filepath.IsAbs(file) {
		file = filepath.Join(root, file)
	}

	probes, ok := probeEditWithin(postEditDeadline, root, base.Snapshot, file, postEditMinShape)
	if !ok {
		return emitNothing()
	}
	fresh := unprobed(probes, base.Reported)
	digest, shown := reporter.ProbeDigest(fresh)
	if digest == "" {
		return emitNothing()
	}

	// Record before emitting, like the Stop hook's ledger: staying silent
	// beats repeating a finding on every later edit of the file. Re-read the
	// baseline first, because the analysis took long enough for another hook
	// to have written it, and this write must not drop that one's ledger.
	latest, err := readBaseline(path)
	if err != nil {
		return emitNothing()
	}
	if err := writeJSONAtomic(path, withReported(latest, probeFindings(shown))); err != nil {
		return emitNothing()
	}

	return emitJSON(cmd, map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "PostToolUse",
			"additionalContext": digest,
		},
	})
}

// probeEditWithin runs probeEdit under a deadline and reports false when the
// deadline passed or the probe panicked. On expiry the goroutine is abandoned
// rather than cancelled: index() has no cancellation, and the process exits
// as soon as the hook returns, which is the cancellation.
func probeEditWithin(d time.Duration, root string, base snapshot.Snapshot, file string, minShape float64) ([]reporter.ProbeResult, bool) {
	type result struct {
		probes []reporter.ProbeResult
		ok     bool
	}
	done := make(chan result, 1)
	go func() {
		defer func() {
			if recover() != nil {
				done <- result{}
			}
		}()
		probes, ok := probeEdit(root, base, file, minShape)
		done <- result{probes, ok}
	}()
	select {
	case r := <-done:
		return r.probes, r.ok
	case <-time.After(d):
		return nil, false
	}
}

// probeEdit indexes root as it stands and probes the functions in file whose
// bodies the session wrote, at the baseline's operating point.
func probeEdit(root string, base snapshot.Snapshot, file string, minShape float64) ([]reporter.ProbeResult, bool) {
	p, err := hookParams(root)
	if err != nil {
		return nil, false
	}
	p = pinThresholds(p, &base.Params)

	// Cheap refusals before the index: a file no frontend reads, or a test
	// file outside the chosen population, can hold no probe.
	if !languageSelection(p).Admits(file) || (p.TestsMode == "exclude" && parser.IsTestFile(file)) {
		return nil, false
	}
	rel, ok := relativeToRoot(root, file)
	if !ok {
		return nil, false
	}

	res, err := index(root, p, io.Discard, nil)
	if err != nil || len(res.Units) == 0 {
		return nil, false
	}

	keys := snapshot.Keys(res.Units, root)
	before := base.UnitByKey()
	paired := make(map[string]bool, len(base.Pairs))
	for _, pr := range base.Pairs {
		paired[pr.Key()] = true
	}
	forkFloor := analyzer.ForkShapeFloor
	if p.Calibrate > 0 && p.Pinned {
		forkFloor = p.Threshold
	}
	opts := probeOptions(p)

	var out []reporter.ProbeResult
	for i, u := range res.Units {
		if snapshot.RelSlash(root, u.File) != rel {
			continue
		}
		digest := snapshot.Digest(u.Fingerprint)
		if digest == "" || unreferenceable(u) {
			continue
		}
		old, had := before[keys[i]]
		if had && old.Digest == digest {
			continue
		}
		probe := reporter.ProbeResult{Key: keys[i], File: rel, Line: u.StartLine, New: !had}
		skip := func(o int) bool { return !parser.SameBuildUnit(u, res.Units[o]) || unreferenceable(res.Units[o]) }
		for _, m := range probeMatches(res, i, opts, skip) {
			if m.Candidate.Breakdown.Score < minShape {
				continue
			}
			o := otherIdx(m.Candidate, i)
			a, b := keys[i], keys[o]
			if a > b {
				a, b = b, a
			}
			if paired[snapshot.Pair{A: a, B: b}.Key()] {
				continue
			}
			probe.Matches = append(probe.Matches, reporter.ProbeMatch{
				Key:         keys[o],
				File:        snapshot.RelSlash(root, m.Unit.File),
				Line:        m.Unit.StartLine,
				Score:       m.Candidate.Breakdown.Score,
				Containment: m.Candidate.Breakdown.Containment,
				Locality:    m.Locality,
				Kind:        reporter.KindClause(probeKind(res, i, o, m.Candidate.Breakdown.Score, forkFloor)),
			})
		}
		out = append(out, probe)
	}
	return out, true
}

// unreferenceable reports a function no Go code can call: init, and main in
// package main. The note's whole ask is "reuse the existing one", which the
// language forbids for these — and init bodies registering flags read alike
// in every command file, which the replay measured as the largest class of
// post-edit noise. A rule of the language, not a naming heuristic: Go code
// cannot refer to either. Other languages keep every function.
func unreferenceable(u parser.CodeUnit) bool {
	if u.Lang != gofront.Lang || u.ReceiverType != "" {
		return false
	}
	return u.Name == "init" || (u.Name == "main" && u.Package == "main")
}

// probeKind labels a probe match the way the pipeline labels a pair — the
// same rules, the same context — so a note can say "subsystem copies" where
// that is what the two functions are. Annotation only: nothing is filtered
// on it. The thin-wrapper vocab bags are built for the two units alone, since
// index() leaves every canonical tree in place.
func probeKind(res Result, a, b int, score, forkFloor float64) *analyzer.KindNote {
	ua, ub := res.Units[a], res.Units[b]
	vocab := analyzer.BuildThinVocab([]parser.CodeUnit{ua, ub})
	da, db := res.Docs[a], res.Docs[b]
	return analyzer.ClassifyPairIn(ua, ub, score, forkFloor, analyzer.PairContext{
		ResolvedA: da.ResolvedCallees, ResolvedB: db.ResolvedCallees,
		VocabA: vocab[0], VocabB: vocab[1],
		CallersA: da.Callers, CallersB: db.Callers,
		CallerPkgsA: da.CallerPackages, CallerPkgsB: db.CallerPackages,
	})
}

// newPairKey is reporter.Notable's ledger key for the pair of a and b, sides
// in snapshot order (A < B).
func newPairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return "new:" + a + "|" + b
}

// probeLedgerKey names one probe finding in the baseline's Reported ledger.
func probeLedgerKey(probe, match string) string { return "probe:" + probe + "|" + match }

// unprobed drops the matches this session has already been told about, and
// the probes left with none. A pair is told about once whichever side was
// probed: the `new:` key is unordered, so B probed against A is quiet after A
// was probed against B — and after the Stop hook reported the pair.
func unprobed(probes []reporter.ProbeResult, reported []string) []reporter.ProbeResult {
	seen := make(map[string]bool, len(reported))
	for _, k := range reported {
		seen[k] = true
	}
	var out []reporter.ProbeResult
	for _, p := range probes {
		var keep []reporter.ProbeMatch
		for _, m := range p.Matches {
			if !seen[probeLedgerKey(p.Key, m.Key)] && !seen[newPairKey(p.Key, m.Key)] {
				keep = append(keep, m)
			}
		}
		if len(keep) > 0 {
			p.Matches = keep
			out = append(out, p)
		}
	}
	return out
}

// probeFindings is the ledger entries for what a digest showed: the probe key
// itself, and the Stop hook's `new:` key for the same pair, so the end-of-turn
// note does not repeat what this one already said.
func probeFindings(shown []reporter.ProbeResult) []reporter.Finding {
	var out []reporter.Finding
	for _, p := range shown {
		for _, m := range p.Matches {
			out = append(out,
				reporter.Finding{Key: probeLedgerKey(p.Key, m.Key)},
				reporter.Finding{Key: newPairKey(p.Key, m.Key)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

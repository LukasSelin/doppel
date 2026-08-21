package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/snapshot"
	"github.com/spf13/cobra"
)

// baselineTTL bounds how long an abandoned baseline survives. Sessions end
// without notice — a crash, a closed terminal — so nothing can be relied on to
// delete its own file; session-start sweeps instead.
const baselineTTL = 7 * 24 * time.Hour

// hookInput is the subset of the Claude Code hook payload doppel reads.
// Unknown fields are ignored, so the harness can add its own without breaking
// this.
type hookInput struct {
	SessionID string `json:"session_id"`
	Cwd       string `json:"cwd"`
	Source    string `json:"source"`
	// StopHookActive is set by the harness when this Stop hook fires on a turn
	// that a Stop hook already continued. It is the loop guard: a hook that
	// keeps producing output every time it is re-entered never lets the turn
	// end, and Claude Code eventually overrides it with a warning.
	StopHookActive bool `json:"stop_hook_active"`
}

// baselineFile wraps a Snapshot with the things that must not live inside one.
//
// A Snapshot has to stay byte-identical for an unchanged tree, which rules out
// a timestamp or an absolute path. Both are still needed — the timestamp to
// sweep abandoned files, the root so that `hook stop` measures the same tree
// `hook session-start` did even if the working directory moved — so they live
// out here, where nothing compares them.
type baselineFile struct {
	Root      string            `json:"root"`
	CreatedAt string            `json:"createdAt"`
	Snapshot  snapshot.Snapshot `json:"snapshot"`
	// Reported is the ledger of findings already surfaced this session, sorted.
	//
	// A delta is cumulative against the session-start origin, so without this a
	// single new duplicate would be reported again on every subsequent turn —
	// and in agent mode would continue every subsequent turn. The ledger rides
	// in the baseline file rather than a second one on purpose: the conventions
	// allow exactly one persisted artifact, and this is a property of the
	// session the baseline already represents. It is never compared, and it
	// never touches Snapshot, which must stay the untouched origin.
	Reported []string `json:"reported,omitempty"`
}

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Claude Code hook entry points",
	Long: "Subcommands intended to be run by Claude Code hooks. Each reads the hook\n" +
		"payload as JSON on stdin and writes a hook response as JSON on stdout.",
}

var hookSessionStartCmd = &cobra.Command{
	Use:   "session-start",
	Short: "Record a baseline and describe the corpus's existing concepts",
	Args:  cobra.NoArgs,
	RunE:  runHookSessionStart,
}

var hookStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Report how much this session changed the duplication picture",
	Args:  cobra.NoArgs,
	RunE:  runHookStop,
}

var hookRoot string

func init() {
	for _, c := range []*cobra.Command{hookSessionStartCmd, hookStopCmd} {
		c.Flags().StringVar(&hookRoot, "root", "", "Directory to analyze (default: the cwd from the hook payload)")
		hookCmd.AddCommand(c)
	}
	rootCmd.AddCommand(hookCmd)
}

// runHookSessionStart records the session's measurement origin and hands back a
// description of what concepts the corpus already contains.
//
// The baseline is written only when there is not already one for this session.
// SessionStart fires on resume and after compaction too, and re-recording then
// would silently move the origin mid-session, so that the impact report would
// describe the last few minutes rather than the session. Checking for the file
// covers every source, including ones added later.
func runHookSessionStart(cmd *cobra.Command, args []string) error {
	in, err := readHookInput(cmd.InOrStdin())
	if err != nil {
		return emitNothing()
	}
	root := resolveRoot(in.Cwd)
	sweepBaselines()

	snap, ok := snapshotAt(root)
	if !ok {
		return emitNothing()
	}

	path := baselinePath(in.SessionID)
	if _, statErr := os.Stat(path); statErr != nil {
		_ = writeJSONAtomic(path, baselineFile{
			Root:      filepath.ToSlash(root),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Snapshot:  snap,
		})
	}

	return emitJSON(cmd, map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "SessionStart",
			"additionalContext": reporter.ConceptDigest(snap, displayRoot(root)),
		},
	})
}

// runHookStop reports the session's cumulative effect on the duplication
// picture.
//
// Two channels, and the difference between them is a hard constraint of the
// harness rather than a matter of taste. systemMessage reaches the user and
// lets the turn end. additionalContext reaches the model — and continues the
// turn: Claude Code appends the hook's message to the same list it treats as
// blocking output and re-enters the query loop. There is no third option; a
// Stop hook cannot put text in the model's context without the agent working
// again.
//
// So agent mode is opt-outable (hook-notify) and gated hard (reporter.Notable
// plus the Reported ledger), because every finding it emits costs a model
// turn. A measurement that interrupts on every count that moved would be worse
// than no measurement at all.
func runHookStop(cmd *cobra.Command, args []string) error {
	in, err := readHookInput(cmd.InOrStdin())
	if err != nil {
		return emitNothing()
	}

	// The turn we are being asked about is one this hook already continued.
	// Speaking again would continue it again, and the loop only ends when the
	// harness overrides us with a warning. Return before any analysis: this is
	// the guard, and it also spares the repo a second full pipeline run for a
	// turn already reported on.
	if in.StopHookActive {
		return emitNothing()
	}

	path := baselinePath(in.SessionID)
	base, err := readBaseline(path)
	if err != nil {
		// No baseline is the normal state for a session that began before the
		// plugin was installed, and a corrupt one is not worth a word to the
		// user. Either way there is nothing to compare against.
		return emitNothing()
	}

	// The root comes from the baseline so that a `cd` mid-session cannot
	// silently re-point the measurement at a different tree.
	root := base.Root
	if root == "" {
		root = resolveRoot(in.Cwd)
	}
	head, ok := snapshotAt(root)
	if !ok {
		return emitNothing()
	}

	delta := snapshot.Diff(base.Snapshot, head)
	if !delta.Comparable {
		// The binary, the vocabulary or the params moved under us, so the old
		// origin measures a different question. Replace it and stay quiet
		// rather than reporting a difference nobody caused.
		_ = writeJSONAtomic(path, baselineFile{
			Root:      base.Root,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Snapshot:  head,
		})
		return emitNothing()
	}

	mode, err := hookNotify(root)
	if err != nil || mode == NotifyOff {
		// A malformed hook-notify is the user's config error, but a hook is the
		// wrong place to learn about it: stderr here reads as a broken tool.
		return emitNothing()
	}

	digest := reporter.ImpactDigest(delta, "")
	if digest == "" {
		return emitNothing()
	}

	deltaPath := deltaPathFor(path)
	if err := writeJSONAtomic(deltaPath, delta); err == nil {
		digest = reporter.ImpactDigest(delta, deltaPath)
	}

	out := map[string]any{"systemMessage": digest}

	if mode == NotifyAgent {
		if fresh := unreported(reporter.Notable(delta), base.Reported); len(fresh) > 0 {
			if note := reporter.AgentDigest(fresh); note != "" {
				// Record before emitting. If the write fails we would rather
				// stay silent than repeat this finding on every future turn,
				// each time continuing the turn to do it.
				if err := writeJSONAtomic(path, withReported(base, fresh)); err == nil {
					out["hookSpecificOutput"] = map[string]any{
						"hookEventName":     "Stop",
						"additionalContext": note,
					}
				}
			}
		}
	}

	return emitJSON(cmd, out)
}

// unreported drops findings already surfaced this session.
func unreported(findings []reporter.Finding, reported []string) []reporter.Finding {
	if len(findings) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(reported))
	for _, k := range reported {
		seen[k] = true
	}
	var out []reporter.Finding
	for _, f := range findings {
		if !seen[f.Key] {
			out = append(out, f)
		}
	}
	return out
}

// withReported returns the baseline with these findings added to its ledger.
//
// Snapshot and CreatedAt are carried over untouched: this records what has been
// said about the session, and must not move the origin the session is measured
// against. The ledger is sorted and deduped so the file stays byte-stable for
// an unchanged set.
func withReported(base baselineFile, fresh []reporter.Finding) baselineFile {
	seen := make(map[string]bool, len(base.Reported)+len(fresh))
	var keys []string
	for _, k := range base.Reported {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for _, f := range fresh {
		if !seen[f.Key] {
			seen[f.Key] = true
			keys = append(keys, f.Key)
		}
	}
	sort.Strings(keys)
	base.Reported = keys
	return base
}

// snapshotAt analyzes root the way a hook must and returns the comparable
// record of it, or false when there is nothing worth reporting.
//
// The pair set is deliberately the full retrieved candidate list rather than
// the ranked one: top-N and the per-function diversity cap drop pairs for
// presentation reasons, and a pair that fell past rank 20 has not changed.
func snapshotAt(root string) (snapshot.Snapshot, bool) {
	p, err := hookParams(root)
	if err != nil {
		return snapshot.Snapshot{}, false
	}
	res, err := analyze(root, p, io.Discard)
	if err != nil || len(res.Units) == 0 {
		return snapshot.Snapshot{}, false
	}
	return snapshotOf(res, res.Pairs), true
}

func readHookInput(r io.Reader) (hookInput, error) {
	var in hookInput
	data, err := io.ReadAll(io.LimitReader(r, 1<<20))
	if err != nil {
		return in, err
	}
	if len(data) == 0 {
		return in, fmt.Errorf("empty hook payload")
	}
	return in, json.Unmarshal(data, &in)
}

// emitNothing exits successfully and silently.
//
// Every failure path in this file ends here. A hook that cannot analyze the
// repo has nothing useful to say, and saying it loudly would be worse than
// saying nothing: a non-zero exit or stderr output from a SessionStart hook
// surfaces to the user as a broken-tool notice, and blocking a session over a
// measurement would be indefensible.
func emitNothing() error { return nil }

func emitJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// resolveRoot picks the tree to analyze: an explicit --root wins, then the cwd
// the harness reported. In a worktree that cwd is the worktree, which is what
// should be measured — CLAUDE_PROJECT_DIR would still point at the original
// checkout.
func resolveRoot(cwd string) string {
	if hookRoot != "" {
		return hookRoot
	}
	if cwd != "" {
		return cwd
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// displayRoot renders the root for a human. The base name is enough context and
// keeps an absolute home-directory path out of the model's context window.
func displayRoot(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	return filepath.Base(abs)
}

func baselineDir() string { return filepath.Join(os.TempDir(), "doppel-baselines") }

// baselinePath maps a session id to its baseline file.
//
// The id is hashed rather than validated. It arrives from outside and is used
// to build a path, so the interesting question is not "is this id well-formed"
// but "can any id escape this directory" — and against a fixed-length hex
// digest, path traversal, absolute paths, drive letters, reserved device names
// and length limits are all impossible by construction rather than by a
// blocklist somebody has to keep correct.
func baselinePath(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return filepath.Join(baselineDir(), hex.EncodeToString(sum[:16])+".json")
}

func deltaPathFor(baseline string) string {
	return baseline[:len(baseline)-len(".json")] + ".impact.json"
}

// writeJSONAtomic writes via a temp file and a rename, so a reader never sees a
// half-written baseline. A torn file would be read as "no baseline" anyway, but
// only after the session it belonged to had already lost its origin.
func writeJSONAtomic(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".doppel-*")
	if err != nil {
		return err
	}
	enc := json.NewEncoder(tmp)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

func readBaseline(path string) (baselineFile, error) {
	var b baselineFile
	data, err := os.ReadFile(path)
	if err != nil {
		return b, err
	}
	if err := json.Unmarshal(data, &b); err != nil {
		// A corrupt baseline is worse than none: it would be diffed against
		// and produce nonsense. Drop it so the next session starts clean.
		os.Remove(path)
		return b, err
	}
	return b, nil
}

// sweepBaselines removes files no live session can still be using. Errors are
// ignored throughout: this is housekeeping, and failing it must never cost a
// session its own baseline.
func sweepBaselines() {
	entries, err := os.ReadDir(baselineDir())
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-baselineTTL)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = os.Remove(filepath.Join(baselineDir(), e.Name()))
	}
}

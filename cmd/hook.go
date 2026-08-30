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
	"strings"
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
	// Prompt is the user's message text, present on UserPromptSubmit.
	Prompt string `json:"prompt"`
	// ToolName and ToolInput are present on PreToolUse. ToolInput is the
	// tool's own argument object; only file_path is read here.
	ToolName  string `json:"tool_name"`
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
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
	// Advised is the pre-tool ledger: relative slash paths of files whose
	// twin advisory has already been given this session. Same mechanism and
	// rationale as Reported — the facts are session-cumulative, so an
	// unremembered advisory repeats on every edit of the same file.
	Advised []string `json:"advised,omitempty"`
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

var hookUserPromptCmd = &cobra.Command{
	Use:   "user-prompt",
	Short: "Scope the corpus's duplication facts to the packages a prompt mentions",
	Args:  cobra.NoArgs,
	RunE:  runHookUserPrompt,
}

var hookPreToolCmd = &cobra.Command{
	Use:   "pre-tool",
	Short: "Advise on merge-worthy twins in a file about to be edited",
	Args:  cobra.NoArgs,
	RunE:  runHookPreTool,
}

var hookStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Report how much this session changed the duplication picture",
	Args:  cobra.NoArgs,
	RunE:  runHookStop,
}

var hookRoot string

func init() {
	for _, c := range []*cobra.Command{hookSessionStartCmd, hookStopCmd, hookUserPromptCmd, hookPreToolCmd} {
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

	// A baseline already here means SessionStart fired again on a resume or
	// after compaction. Inherit its operating point rather than deriving a
	// second one, so the digest describes the corpus the same way the origin
	// does.
	snap, ok := snapshotAt(root, baselineParams(in.SessionID))
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
	head, ok := snapshotAt(root, &base.Snapshot.Params)
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
			// Only the findings the digest actually printed go in the ledger.
			// The rest were counted, not said, and retiring them here would
			// suppress them for the whole session without anyone having seen
			// them; they lead the next turn's list instead.
			if note, shown := reporter.AgentDigest(fresh); note != "" {
				// Record before emitting. If the write fails we would rather
				// stay silent than repeat this finding on every future turn,
				// each time continuing the turn to do it.
				if err := writeJSONAtomic(path, withReported(base, shown)); err == nil {
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
//
// pinned carries a baseline's already-derived thresholds, or nil to derive
// them here. See pinThresholds.

// baselineParams returns the operating point recorded for this session, or nil
// when there is no usable baseline to inherit one from.
func baselineParams(sessionID string) *snapshot.Params {
	base, err := readBaseline(baselinePath(sessionID))
	if err != nil {
		return nil
	}
	return &base.Snapshot.Params
}

// pinThresholds supplies a baseline's operating point to a hook run.
//
// Every hook subcommand has to measure at the same thresholds the baseline was
// written at, and calibration derives them from the corpus — which the session
// is busy editing. Recalibrating each turn lets an edit move the null
// distribution far enough to change the derived threshold by a hundredth, at
// which point Params equality fails and the Stop hook goes silent for a turn
// that nothing was wrong with. So session start derives once and every later
// turn supplies the result back.
//
// This is the one place a hook run reads the baseline for something other than
// diffing, and it stays on the right side of the no-caches rule by being a
// parameter rather than a result: the thresholds arrive exactly as --threshold
// would supply them, and every pipeline stage still runs from source. Nothing
// analytical is reused, and no stage is skipped except the derivation itself.
//
// A nil baseline means this run derives its own, which is right everywhere it
// can happen: session start has no baseline yet and is the run whose thresholds
// every later turn inherits, and user-prompt scopes a digest without diffing
// anything, so it wants thresholds fitted to the corpus in front of it. The Stop
// hook cannot reach it at all — a missing baseline returns before this point,
// because there is nothing to compare against.
func pinThresholds(p Params, pinned *snapshot.Params) Params {
	if pinned == nil {
		return p
	}
	p.Threshold = pinned.Threshold
	p.Calibrate = pinned.Calibrate
	p.Pinned = true
	// StructMin stays zero. A hook diffs the full candidate set, so the
	// overlap filter is deliberately off here; the calibrated value lives in
	// the baseline's Params for comparability, not as a filter to reapply.
	return p
}

func snapshotAt(root string, pinned *snapshot.Params) (snapshot.Snapshot, bool) {
	p, err := hookParams(root)
	if err != nil {
		return snapshot.Snapshot{}, false
	}
	p = pinThresholds(p, pinned)
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

// runHookUserPrompt scopes the corpus's duplication facts to the packages the
// prompt mentions — the point in the session where the target is first known
// and no edit exists yet.
//
// The full pipeline runs, once per user prompt, with the same uncapped params
// the other hooks use: the cost is one analysis at the moment the answer can
// still change what gets written. A prompt mentioning no known package — the
// overwhelmingly common case — costs the analysis and says nothing.
func runHookUserPrompt(cmd *cobra.Command, args []string) error {
	in, err := readHookInput(cmd.InOrStdin())
	if err != nil || in.Prompt == "" {
		return emitNothing()
	}

	snap, ok := snapshotAt(resolveRoot(in.Cwd), baselineParams(in.SessionID))
	if !ok {
		return emitNothing()
	}

	pkgs := scopedPackages(in.Prompt, snap)
	if len(pkgs) == 0 {
		return emitNothing()
	}
	digest := reporter.ScopeDigest(snap, pkgs)
	if digest == "" {
		return emitNothing()
	}

	return emitJSON(cmd, map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "UserPromptSubmit",
			"additionalContext": digest,
		},
	})
}

// runHookPreTool advises on the merge-worthy twins of the file about to be
// edited — the last responsible moment before the edit exists.
//
// Facts come from the session-start baseline, and the digest labels them so.
// This is a deliberate widening of the baseline's role — a fact sheet as well
// as a measurement origin — with the original boundary intact: analyze never
// reads it, and no pipeline stage is ever skipped because it exists. The
// advisory is not a pipeline; recomputing instead would cost a full analysis
// per Edit/Write, which on a large repo means seconds added to every edit.
//
// Advisory-only, permanently: additionalContext is emitted and
// permissionDecision never is. A blocking dedupe hook that misfires on a
// genuine near-duplicate — and near-duplicates are exactly what it would
// fire on — would be worse than no hook.
func runHookPreTool(cmd *cobra.Command, args []string) error {
	in, err := readHookInput(cmd.InOrStdin())
	if err != nil || in.ToolInput.FilePath == "" {
		return emitNothing()
	}

	path := baselinePath(in.SessionID)
	base, err := readBaseline(path)
	if err != nil || base.Snapshot.Schema != snapshot.Schema {
		return emitNothing()
	}

	rel, ok := relativeToRoot(base.Root, in.ToolInput.FilePath)
	if !ok {
		return emitNothing()
	}
	for _, a := range base.Advised {
		if a == rel {
			return emitNothing()
		}
	}

	digest := reporter.AdviceDigest(base.Snapshot, rel)
	if digest == "" {
		return emitNothing()
	}

	// Record before emitting, like the Stop hook's Reported ledger: if the
	// write fails, staying silent beats repeating this advisory on every
	// future edit of the file.
	base.Advised = append(base.Advised, rel)
	sort.Strings(base.Advised)
	if err := writeJSONAtomic(path, base); err != nil {
		return emitNothing()
	}

	return emitJSON(cmd, map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "PreToolUse",
			"additionalContext": digest,
		},
	})
}

// relativeToRoot rewrites an absolute tool path as the snapshot's own
// relative slash form, or reports that the file lives outside the analyzed
// tree. The snapshot's normalization is what makes this lookup possible.
func relativeToRoot(root, file string) (string, bool) {
	rel, err := filepath.Rel(root, file)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return filepath.ToSlash(rel), true
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

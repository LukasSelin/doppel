package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/dashboard"
	"github.com/LukasSelin/doppel/internal/identity"
	"github.com/LukasSelin/doppel/internal/snapshot"
	"github.com/spf13/cobra"
)

var (
	diffFormat    string
	diffUnchanged bool
	diffOutput    string
)

// Exit codes. `doppel diff` is a command a person or a script runs, not a
// hook, so unlike `doppel hook` it is allowed — and expected — to fail
// loudly. The three codes separate the three things that can go wrong, so a
// script can tell "you gave me a file I cannot read" from "these two runs
// cannot be compared" without parsing English.
const (
	exitDiffOK          = 0 // the comparison ran
	exitDiffUnreadable  = 1 // a file is missing, is not JSON, or its WL codec is corrupt
	exitDiffIncomparabl = 2 // the two snapshots refuse comparison; see identity.Compare
)

// incomparableError marks the refusal path so RunE can pick exit code 2
// without re-deriving the reason.
type incomparableError struct{ reason string }

func (e incomparableError) Error() string { return "not comparable: " + e.reason }

// diffExit is os.Exit behind a seam, so a test can assert the code without
// ending the test binary. Production never replaces it.
var diffExit = os.Exit

var diffCmd = &cobra.Command{
	Use:   "diff <old.json> <new.json>",
	Short: "Match functions across two analysis snapshots and say what happened to each",
	Long: `Reads two snapshots written by ` + "`doppel analyze --format json`" + ` and matches
their functions to each other by body, not by name.

Every function on either side lands in exactly one of eight classes:

  unchanged  same package and name, identical fingerprint digest
  edited     same package and name, the body moved
  renamed    same package, a different name, the same or a similar body
  moved      a different package
  split      one old body covering two or more new ones
  merged     two or more old bodies covered by one new one
  new        nothing in the old snapshot matched it
  deleted    nothing in the new snapshot matched it

Matching runs in three passes, strongest evidence first: an unchanged snapshot
key, then an identical fingerprint digest, then greedy bipartite matching on
the corpus-weighted Weisfeiler-Lehman overlap of the two bodies. Every line
prints the evidence that produced its class.

This is not what a session hook reports. ` + "`doppel hook`" + ` measures a session's
impact on the pair list and deliberately claims nothing it cannot attribute to
an edit; this command asks the wider question and answers it from bodies.

--output writes the same delta as markdown, or as a self-contained review page
when the path ends in .html: merge-worthy pairs introduced first, then the ones
removed, then what happened to each function.

Exit codes: 0 compared, 1 a file could not be read, 2 the two snapshots refuse
comparison (mismatched schema or canon rule set).`,
	Example: `  doppel analyze . --format json > before.json
  # ... edit ...
  doppel analyze . --format json > after.json
  doppel diff before.json after.json
  doppel diff before.json after.json --format json
  doppel diff before.json after.json -o review.html   # the review page`,
	Args: cobra.ExactArgs(2),
	// A snapshot that cannot be read is a data problem, not a CLI-usage
	// problem, and printing the flag table under it buries the message.
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := runDiff(cmd, args)
		if err == nil {
			return nil
		}
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		diffExit(diffExitCode(err))
		// Unreachable in production. Returning the error keeps the seam
		// honest for a test that replaced diffExit with a recorder.
		return err
	},
}

func diffExitCode(err error) int {
	var inc incomparableError
	if errors.As(err, &inc) {
		return exitDiffIncomparabl
	}
	return exitDiffUnreadable
}

func init() {
	diffCmd.Flags().StringVar(&diffFormat, "format", "text", "Output format: text or json")
	diffCmd.Flags().BoolVar(&diffUnchanged, "unchanged", false, "List unchanged functions individually instead of only counting them")
	diffCmd.Flags().StringVarP(&diffOutput, "output", "o", "", "Also write the delta report to this file: markdown, or the review page when it ends in .html")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	if diffFormat != "text" && diffFormat != "json" {
		return fmt.Errorf("--format must be text or json, got %q", diffFormat)
	}
	base, err := readSnapshot(args[0])
	if err != nil {
		return err
	}
	head, err := readSnapshot(args[1])
	if err != nil {
		return err
	}

	// Since is Compare plus the pair-set comparison: which near-duplicate pairs
	// the classified changes created and which they dissolved. Both snapshots
	// carry their pair lists, so this costs a set difference and no re-scoring.
	d, err := identity.Since(base, head, identity.Options{})
	if err != nil {
		return err
	}
	if !d.Comparable {
		// The refusal is a Result, not an error, inside the library — the
		// hook path needs it that way. At the CLI boundary it becomes a
		// non-zero exit, because a script piping this into anything must not
		// read an empty change list as "nothing happened".
		return incomparableError{reason: d.Reason}
	}

	if err := writeDeltaFile(cmd, args, base, head, d); err != nil {
		return err
	}

	if diffFormat == "json" {
		return identity.WriteJSONDelta(cmd.OutOrStdout(), d)
	}
	identity.PrintDelta(cmd.OutOrStdout(), d, diffUnchanged)
	return nil
}

// writeDeltaFile honours --output: the review page when the path ends .html,
// markdown otherwise — the extension rule `analyze --output` already uses.
//
// The page is not the single-run dashboard with a second run bolted on. That
// page's payload is a per-run index and crosses no runs; this one is the
// timeline's transition read as a review, a sibling of both. See
// dashboard.AuditPayload.
func writeDeltaFile(cmd *cobra.Command, paths []string, base, head snapshot.Snapshot, d identity.Delta) error {
	if diffOutput == "" {
		return nil
	}
	f, err := os.Create(diffOutput)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()
	if strings.EqualFold(filepath.Ext(diffOutput), ".html") {
		warnReportCaps(cmd.ErrOrStderr(), head.Params)
		labels := fileLabels(paths)
		if err := dashboard.PrintAudit(f, buildAudit(labels[0], labels[1], workdirName("doppel"), base, head, d)); err != nil {
			return err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Review page written to %s\n", diffOutput)
		return nil
	}
	identity.MarkdownDelta(f, d, diffUnchanged)
	fmt.Fprintf(cmd.ErrOrStderr(), "Delta report written to %s\n", diffOutput)
	return nil
}

// buildAudit flattens one delta into the review page's payload. Like
// buildTimeline it recomputes nothing: stepChange selects and bounds exactly
// as a timeline step does, and the verdict counts are read off the pair lists.
//
// The labels and the target are the caller's: `doppel diff` names the runs
// after their files, the Stop hook after the session ("session start", "now").
func buildAudit(baseLabel, headLabel, target string, base, head snapshot.Snapshot, d identity.Delta) dashboard.AuditPayload {
	p := dashboard.AuditPayload{
		Schema:       dashboard.AuditSchema,
		Target:       target,
		Base:         baseLabel,
		Head:         headLabel,
		Params:       paramsLine(head.Params),
		Notes:        d.Notes,
		OldFunctions: d.OldFunctions,
		NewFunctions: d.NewFunctions,
		Change:       stepChange(0, d),
		Bounds: dashboard.AuditBounds{
			MaxChanges:       maxStepChanges,
			MaxPairs:         maxStepPairs,
			ReportTop:        head.Params.Top,
			ReportMaxPerFunc: head.Params.MaxPerFunc,
		},
	}
	for _, pc := range d.Created {
		if pc.MergeWorthy {
			p.CreatedMergeWorthy++
		}
	}
	for _, pc := range d.Dissolved {
		if pc.MergeWorthy {
			p.DissolvedMergeWorthy++
		}
		if pc.Attributable() {
			p.DissolvedAttributable++
		}
	}
	p.Units = auditUnits(base, head, p.Change)
	return p
}

// auditUnits locates every key the listed pairs name: a created pair's sides
// in the head run, a dissolved pair's in the base, sorted by key so the
// payload is byte-stable.
func auditUnits(base, head snapshot.Snapshot, sc dashboard.StepChange) []dashboard.AuditUnit {
	index := func(s snapshot.Snapshot) map[string]snapshot.Unit {
		m := make(map[string]snapshot.Unit, len(s.Units))
		for _, u := range s.Units {
			m[u.Key] = u
		}
		return m
	}
	seen := map[string]dashboard.AuditUnit{}
	add := func(units map[string]snapshot.Unit, rows []dashboard.PairRow) {
		for _, r := range rows {
			for _, k := range []string{r.A, r.B} {
				if _, ok := seen[k]; ok {
					continue
				}
				if u, ok := units[k]; ok {
					seen[k] = dashboard.AuditUnit{Key: k, File: u.File, Line: u.Line}
				}
			}
		}
	}
	add(index(head), sc.Created)
	add(index(base), sc.Dissolved)
	out := make([]dashboard.AuditUnit, 0, len(seen))
	for _, u := range seen {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func readSnapshot(path string) (snapshot.Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return snapshot.Snapshot{}, fmt.Errorf("read %s: %w", path, err)
	}
	var s snapshot.Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return snapshot.Snapshot{}, fmt.Errorf("parse %s: %w (is it the output of `doppel analyze --format json`?)", path, err)
	}
	if s.Schema == 0 {
		return snapshot.Snapshot{}, fmt.Errorf("parse %s: no schema field — this is not a doppel snapshot", path)
	}
	return s, nil
}

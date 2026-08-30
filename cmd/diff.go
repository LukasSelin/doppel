package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/LukasSelin/doppel/internal/identity"
	"github.com/LukasSelin/doppel/internal/snapshot"
	"github.com/spf13/cobra"
)

var (
	diffFormat    string
	diffUnchanged bool
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

Exit codes: 0 compared, 1 a file could not be read, 2 the two snapshots refuse
comparison (mismatched schema or canon rule set).`,
	Example: `  doppel analyze . --format json > before.json
  # ... edit ...
  doppel analyze . --format json > after.json
  doppel diff before.json after.json
  doppel diff before.json after.json --format json`,
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

	res, err := identity.Compare(base, head, identity.Options{})
	if err != nil {
		return err
	}
	if !res.Comparable {
		// The refusal is a Result, not an error, inside the library — the
		// hook path needs it that way. At the CLI boundary it becomes a
		// non-zero exit, because a script piping this into anything must not
		// read an empty change list as "nothing happened".
		return incomparableError{reason: res.Reason}
	}

	if diffFormat == "json" {
		return identity.WriteJSON(cmd.OutOrStdout(), res)
	}
	identity.Print(cmd.OutOrStdout(), res, diffUnchanged)
	return nil
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

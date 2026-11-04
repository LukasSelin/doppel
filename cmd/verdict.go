package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/LukasSelin/doppel/internal/labels"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/spf13/cobra"
)

var (
	verdictReason  string
	verdictClass   string
	verdictKind    string
	verdictBlocker string
	verdictTarget  string
	verdictBy      string
	verdictCommit  string
	verdictRoot    string
	verdictFile    string
)

// verdictPath is where a repository's recorded verdicts live, under its root.
// The walk skips dot-directories, so nothing here can join a corpus.
const verdictPath = ".doppel/verdicts.json"

var verdictCmd = &cobra.Command{
	Use:   "verdict <merged|left|deferred> <a> <b>",
	Short: "Record what was done with a pair, and why, in .doppel/verdicts.json",
	Long: `Record a decision about a near-duplicate pair at the moment it is made.

A report or a hook note names a pair; somebody then merges it, leaves it, or
puts it off, usually with a one-line reason. That reason is the most useful
thing a label can carry, and without a place to put it, it is lost in a
transcript. This command saves it as a label in .doppel/verdicts.json, in the
format the golden benchmark reads, so the file is ground truth as it stands:

  DOPPEL_BENCH_CORPUS=<repo> DOPPEL_BENCH_LABELS=<repo>/.doppel/verdicts.json \
    go test ./internal/bench/ -run TestGoldenRanking -v

Outcomes:
  merged    carried out: one body left, or both now delegate to one helper
  left      looked at and kept as it is; say why with --kind or --blocker
  deferred  judged worth doing, not done yet

A side is a qualified name as the report prints it (pkg.Name, pkg.*Recv.Method),
or a key as the hook notes print it, with its file: pkg.Name@path/to/file.go.

The class defaults from the outcome: merged and deferred record a merge (pass
--class refactor when a shared helper was extracted and both functions stay),
left with --kind records a false positive (not a real duplicate: mirror,
entrypoint, skeleton, ...), and left with --blocker records a real duplicate
that stays (no-home, layering, design, too-small, ...).

The decision records today's date and the commit the tree sits on; for a merge,
record it before committing, so the commit is the last one that still has both
functions. Recording a pair again keeps the earlier decision under "earlier".
Nothing in analyze or the hooks reads this file.`,
	Args: cobra.ExactArgs(3),
	// An error here is about a value (an unknown kind, a missing reason), and
	// the caller is usually an agent: the message says what to fix, and the
	// usage block would bury it.
	SilenceUsage: true,
	RunE:         runVerdict,
}

func init() {
	f := verdictCmd.Flags()
	f.StringVar(&verdictReason, "reason", "", "Why, in one line (required)")
	f.StringVar(&verdictClass, "class", "", "merge, refactor or false_positive (default from the outcome)")
	f.StringVar(&verdictKind, "kind", "", "Why a pair is not a real duplicate: "+strings.Join(labels.FPKinds, ", "))
	f.StringVar(&verdictBlocker, "blocker", "", "Why a real duplicate stays: "+strings.Join(labels.Blockers, ", "))
	f.StringVar(&verdictTarget, "target", "", "What acting on the pair makes or would make, e.g. \"one generic over both types\"")
	f.StringVar(&verdictBy, "by", "agent", "Who decided: "+strings.Join(labels.Deciders, " or "))
	f.StringVar(&verdictCommit, "commit", "", "The commit the decision is about (default: HEAD of --root)")
	f.StringVar(&verdictRoot, "root", ".", "Repository root the names and files are relative to")
	f.StringVar(&verdictFile, "file", "", "Verdicts file (default: <root>/"+verdictPath+")")
	_ = verdictCmd.MarkFlagRequired("reason")
	rootCmd.AddCommand(verdictCmd)
}

func runVerdict(cmd *cobra.Command, args []string) error {
	outcome := args[0]
	if !slices.Contains(labels.Outcomes, outcome) {
		return fmt.Errorf("outcome %q: want one of %s", outcome, strings.Join(labels.Outcomes, ", "))
	}
	a, aFile, err := splitSide(args[1])
	if err != nil {
		return err
	}
	b, bFile, err := splitSide(args[2])
	if err != nil {
		return err
	}
	class, err := verdictClassFor(outcome, verdictClass, verdictKind, verdictBlocker)
	if err != nil {
		return err
	}
	commit := verdictCommit
	if commit == "" {
		commit = gitHead(verdictRoot)
	}

	l := labels.Label{
		A: a, AFile: aFile, B: b, BFile: bFile,
		Class: class, Kind: verdictKind, Blocker: verdictBlocker, Target: verdictTarget,
		Decision: &labels.Decision{
			Outcome: outcome,
			By:      verdictBy,
			Date:    time.Now().Format(time.DateOnly),
			Commit:  commit,
			Reason:  strings.TrimSpace(verdictReason),
		},
	}
	if err := l.Validate(); err != nil {
		return err
	}

	path := verdictFile
	if path == "" {
		path = filepath.Join(verdictRoot, filepath.FromSlash(verdictPath))
	}
	f, err := readVerdicts(path, verdictRoot)
	if err != nil {
		return err
	}
	prev := upsertVerdict(&f, l)
	if l.Decision.Date > f.Reviewed {
		f.Reviewed = l.Decision.Date
	}
	if err := writeVerdicts(path, f); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "recorded %s %s as %s in %s", outcome, l.Pair(), class, filepath.ToSlash(path))
	if prev != nil {
		fmt.Fprintf(cmd.OutOrStdout(), " (replaces %s from %s)", prev.Outcome, prev.Date)
	}
	fmt.Fprintln(cmd.OutOrStdout())
	return nil
}

// splitSide reads one side as the report or a hook note prints it: a
// qualified name, optionally pinned to its file with "@". A repeated
// declaration's ordinal ("#2") has no place in a label, which can only name
// a file.
func splitSide(key string) (name, file string, err error) {
	name, file, _ = strings.Cut(key, "@")
	if name == "" {
		return "", "", fmt.Errorf("side %q has no function name", key)
	}
	if strings.Contains(file, "#") {
		return "", "", fmt.Errorf("side %q is one of several same-named declarations in one file; a verdict can only name the file", key)
	}
	return name, file, nil
}

// verdictClassFor is the label class an outcome records when --class is not
// given. A pair left alone needs a reason a scorecard can group by: a kind
// when it is not a real duplicate, a blocker when it is one that stays.
func verdictClassFor(outcome, class, kind, blocker string) (string, error) {
	if class != "" {
		return class, nil
	}
	switch {
	case outcome != "left":
		return "merge", nil
	case kind != "" && blocker != "":
		return "", fmt.Errorf("pass --kind (not a real duplicate) or --blocker (a real duplicate that stays), not both")
	case kind != "":
		return "false_positive", nil
	case blocker != "":
		return "merge", nil
	}
	return "", fmt.Errorf("a pair left alone needs a reason a scorecard can group by: --kind %s for a pair that is not a real duplicate, or --blocker %s for one that is",
		strings.Join(labels.FPKinds, "|"), strings.Join(labels.Blockers, "|"))
}

// readVerdicts loads the verdicts file, or starts one named after the root.
// A file that does not parse is an error rather than something to overwrite:
// it is somebody's record.
func readVerdicts(path, root string) (labels.File, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		abs, err := filepath.Abs(root)
		if err != nil {
			return labels.File{}, err
		}
		return labels.File{Corpus: filepath.Base(abs), Population: "exclude"}, nil
	}
	if err != nil {
		return labels.File{}, err
	}
	f, err := labels.Parse(data)
	if err != nil {
		return labels.File{}, fmt.Errorf("%s: %w (fix or move it; it is not overwritten)", path, err)
	}
	return f, nil
}

// upsertVerdict puts l into f: in place of the label for the same pair, whose
// decision moves to l.Earlier, or at the end. An explicit weight on the old
// label is dropped, because a new decision decides the weight. It returns the
// decision l replaced, if any.
func upsertVerdict(f *labels.File, l labels.Label) *labels.Decision {
	k := l.Key()
	for i := range f.Labels {
		old := f.Labels[i]
		if old.Key() != k {
			continue
		}
		l.Earlier = slices.Clone(old.Earlier)
		if old.Decision != nil {
			l.Earlier = append(l.Earlier, *old.Decision)
		}
		if l.Note == "" {
			l.Note = old.Note
		}
		f.Labels[i] = l
		return old.Decision
	}
	f.Labels = append(f.Labels, l)
	return nil
}

// writeVerdicts writes the file atomically, readable like any file in the
// repository: it is meant to be committed.
func writeVerdicts(path string, f labels.File) error {
	var buf bytes.Buffer
	if err := reporter.WriteJSON(&buf, f); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := writeFileAtomic(path, buf.Bytes()); err != nil {
		return err
	}
	return os.Chmod(path, 0o644)
}

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

// verdictPath is where a repository's recorded verdicts live, under its root
// in the repository's main checkout (see verdictHome). The walk skips
// dot-directories, so nothing here can join a corpus.
const verdictPath = ".doppel/verdicts.json"

// Lock timing. The critical section is a read, a merge and a rename of one
// small file, so a live writer holds the lock for milliseconds: waiting
// seconds is generous, and a lock older than verdictLockStale was left by a
// process that died holding it.
const (
	verdictLockWait  = 10 * time.Second
	verdictLockStale = time.Minute
	verdictLockPoll  = 20 * time.Millisecond
)

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
Nothing in analyze or the hooks reads this file.

Every worktree of a repository shares one file: .doppel/verdicts.json in the
repository's main checkout, at --root's place in it, so a decision made in a
linked worktree outlives the worktree. It stays uncommitted there until
somebody commits it. Outside a git repository the file is under --root, and
--file names any other.`,
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
	f.StringVar(&verdictFile, "file", "", "Verdicts file (default: "+verdictPath+" in the main checkout of --root's repository, or under --root outside one)")
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

	path, home := verdictFile, verdictRoot
	if path == "" {
		home = verdictHome(verdictRoot)
		path = filepath.Join(home, filepath.FromSlash(verdictPath))
	}
	prev, err := recordVerdict(path, home, l)
	if err != nil {
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

// verdictHome is the directory whose .doppel/verdicts.json `doppel verdict`
// writes by default: root's place in its repository's main checkout, or root
// itself outside a repository.
//
// Why the main checkout. Agents work in linked worktrees, one per session
// (.claude/worktrees/<name>/ in this repository), removed when the session is
// done. A file under each worktree's own root would scatter one repository's
// decisions across as many files as there were sessions; each would reach
// the default branch only if its session happened to commit it, and would be
// deleted with the worktree otherwise. One file per repository is what the
// golden benchmark scores and what .doppel/README.md describes, and the main
// checkout is the one directory every worktree of a repository can name.
//
// The alternative was to keep per-worktree files and stage each into its
// session's commit. It loses on three counts: it writes to git, where doppel
// only reads which tree it sits on; a session that commits nothing still
// loses its decisions; and branches that each append to the end of one JSON
// array conflict on every merge of two of them. What this choice costs is
// that the file sits uncommitted in the main checkout until somebody commits
// it there, on whatever branch that checkout is on.
//
// root keeps its place: a corpus rooted at a subdirectory names files
// relative to it, so its verdicts go to the same subdirectory of the main
// checkout. A bare repository has no main checkout and a deleted one cannot
// hold a file; both, and any git failure, keep the file under root, where it
// always was. In the main checkout itself that is the same file as before.
func verdictHome(root string) string {
	main, prefix, ok := gitMainWorktree(root)
	if !ok {
		return root
	}
	if fi, err := os.Stat(main); err != nil || !fi.IsDir() {
		return root
	}
	return filepath.Join(main, filepath.FromSlash(prefix))
}

// recordVerdict puts l into the verdicts file at path, holding the file's
// lock from the read to the write, so a label another process wrote a moment
// earlier is part of what this one writes. home names a file that does not
// exist yet (see readVerdicts). It returns the decision l replaced, if any.
func recordVerdict(path, home string, l labels.Label) (*labels.Decision, error) {
	unlock, err := lockVerdicts(path)
	if err != nil {
		return nil, err
	}
	defer unlock()
	f, err := readVerdicts(path, home)
	if err != nil {
		return nil, err
	}
	prev := upsertVerdict(&f, l)
	if l.Decision.Date > f.Reviewed {
		f.Reviewed = l.Decision.Date
	}
	return prev, writeVerdicts(path, f)
}

// lockVerdicts takes path's lock, a sibling file only one process can create,
// and returns the function that releases it.
//
// Writing atomically is not enough once the file is shared. Two sessions
// recording a verdict at once would each read the file, each add their own
// label and each rename a whole file into place, and the second rename would
// drop the first session's label. One session does this too: agents often
// record several verdicts as parallel tool calls. A stale lock is broken
// rather than waited on forever; that is the one way two writers can still
// overlap, and it needs a writer to have died inside a critical section that
// lasts milliseconds.
func lockVerdicts(path string) (unlock func(), err error) {
	lock := path + ".lock"
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(verdictLockWait)
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			f.Close()
			return func() { _ = os.Remove(lock) }, nil
		}
		// Windows answers access denied, not exists, while another
		// process's removal of the lock is still pending.
		if !errors.Is(err, fs.ErrExist) && !errors.Is(err, fs.ErrPermission) {
			return nil, err
		}
		fi, serr := os.Stat(lock)
		if errors.Is(err, fs.ErrPermission) && errors.Is(serr, fs.ErrNotExist) {
			return nil, err // a directory this process may not write in, not a held lock
		}
		if serr == nil && time.Since(fi.ModTime()) > verdictLockStale && os.Remove(lock) == nil {
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%s is still held after %s: another doppel verdict is writing, or one died holding it; remove it if none is running",
				filepath.ToSlash(lock), verdictLockWait)
		}
		time.Sleep(verdictLockPoll)
	}
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

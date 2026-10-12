package cmd

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// The hook's one use of git, and the reason it is allowed one.
//
// doppel reads no history: authorship, age and churn are invisible to every
// score, by construction. This file does not change that. It reads one fact —
// which commit the analysed tree sits on — to decide whether the session
// baseline still describes the tree the session started from.
//
// That question exists because the Stop hook measures cumulatively against a
// baseline recorded at session start, and agents routinely move the branch
// under it: a rebase onto main, a merge of origin/main, a fast-forward, a
// branch switch. Every function that arrives that way reads as this session's
// work, and measured over a weekend of real sessions that was half of all
// agent notes — each one a continued turn spent explaining "that isn't mine".
// The Attributable bit cannot see it, because those functions really did
// change in the working tree.
//
// Every failure here — no git, not a repository, a timeout — answers "unknown",
// and unknown keeps the old behaviour. A hook must never fail on a measurement,
// and this is a guard on one.

// gitTimeout bounds each git call. The Stop hook already runs a full analysis;
// a hung git (a lock, a credential prompt) must not add to it unboundedly.
const gitTimeout = 5 * time.Second

// gitOutput runs git in root and returns its trimmed stdout, or false on any
// failure.
func gitOutput(root string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	c := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	out, err := c.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// gitHead is the commit root's working tree sits on, or "" when that cannot be
// read (no git, not a repository, an unborn branch).
func gitHead(root string) string {
	h, ok := gitOutput(root, "rev-parse", "--verify", "-q", "HEAD")
	if !ok {
		return ""
	}
	return h
}

// baseMoved reports whether HEAD's move from `from` to `to` brought in work
// that is not this session's.
//
// The session's own commits are the one way HEAD may move without invalidating
// the baseline, and they are recognisable without reading history as evidence:
// they extend the old HEAD linearly, they were committed after the baseline
// was recorded, and they were committed by the local git identity. Anything
// else moved the base:
//
//   - `from` is not an ancestor of `to` — a reset, a rebase, a branch switch;
//   - a merge commit in between — origin/main merged in;
//   - a commit committed before the baseline existed — a fast-forward onto
//     work that already existed when the session started;
//   - a commit by another committer — a squash-merged PR pulled down, whose
//     committer is the forge rather than the local user.
//
// The rules lean towards "moved", and that is the cheap direction: re-baselining
// only forgets session work not yet reported, while not re-baselining blames
// the session for someone else's code on every turn that follows.
//
// ok is false when git could not answer; the caller then leaves the baseline
// alone.
func baseMoved(root, from, to string, since time.Time) (moved, ok bool) {
	if _, isAncestor := gitOutput(root, "merge-base", "--is-ancestor", from, to); !isAncestor {
		// A non-zero exit is "not an ancestor" or "unknown commit", and both
		// mean the old origin is not underneath the new tree. A missing git
		// was already ruled out by the caller reading HEAD.
		return true, true
	}
	log, ok := gitOutput(root, "log", "--format=%P%x09%ct%x09%ce", from+".."+to)
	if !ok {
		return false, false
	}
	me, _ := gitOutput(root, "config", "user.email")
	for _, line := range strings.Split(log, "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 3 {
			return false, false
		}
		if len(strings.Fields(f[0])) > 1 {
			return true, true
		}
		ct, err := strconv.ParseInt(f[1], 10, 64)
		if err != nil {
			return false, false
		}
		if time.Unix(ct, 0).Before(since) {
			return true, true
		}
		if me != "" && !strings.EqualFold(f[2], me) {
			return true, true
		}
	}
	return false, true
}

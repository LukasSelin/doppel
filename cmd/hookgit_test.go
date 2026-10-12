package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

// gitRepo is a throwaway repository with a fixed local identity.
type gitRepo struct {
	t   *testing.T
	dir string
}

func newGitRepo(t *testing.T) gitRepo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	r := gitRepo{t: t, dir: t.TempDir()}
	r.run(nil, "init", "-q", "-b", "main")
	r.run(nil, "config", "user.email", "me@example.com")
	r.run(nil, "config", "user.name", "Me")
	r.run(nil, "config", "commit.gpgsign", "false")
	return r
}

// run runs git in the repository with extra environment, failing the test on
// error.
func (r gitRepo) run(env []string, args ...string) string {
	r.t.Helper()
	c := exec.Command("git", append([]string{"-C", r.dir}, args...)...)
	c.Env = append(os.Environ(), env...)
	out, err := c.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// commit writes name and commits it, with the committer identity and date
// overridable through env.
func (r gitRepo) commit(name string, env ...string) string {
	r.t.Helper()
	if err := os.WriteFile(filepath.Join(r.dir, name), []byte(name), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.run(nil, "add", name)
	r.run(env, "commit", "-q", "-m", name)
	return gitHead(r.dir)
}

// baselineAt is a baseline recorded at HEAD, one minute ago, so that a commit
// made now is after it and one dated in the past is before it.
func (r gitRepo) baselineAt() baselineFile {
	return baselineFile{
		Root:      r.dir,
		CreatedAt: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
		Snapshot:  snapshot.Snapshot{Schema: snapshot.Schema, Doppel: "origin"},
		Head:      gitHead(r.dir),
		Reported:  []string{"new:a|b"},
	}
}

var nowSnapshot = snapshot.Snapshot{Schema: snapshot.Schema, Doppel: "now"}

// The session committing its own work is the one way HEAD may move without
// moving the origin: the baseline stays, and only its reference HEAD follows.
func TestFollowHeadKeepsTheOriginForTheSessionsOwnCommits(t *testing.T) {
	r := newGitRepo(t)
	r.commit("a")
	base := r.baselineAt()
	head := r.commit("b")

	got, rebased := followHead(r.dir, base, nowSnapshot)
	if rebased {
		t.Fatal("the session's own commit re-baselined")
	}
	if got.Snapshot.Doppel != "origin" || got.Head != head {
		t.Errorf("origin %q head %q, want the origin kept and head %q", got.Snapshot.Doppel, got.Head, head)
	}
}

// Every way the branch moves under a session re-baselines it, keeping the
// ledger so nothing already said is said again.
func TestFollowHeadRebaselinesWhenTheBranchMoves(t *testing.T) {
	old := "GIT_COMMITTER_DATE=2001-01-01T00:00:00Z"
	cases := map[string]func(r gitRepo){
		"reset to another branch": func(r gitRepo) {
			r.run(nil, "checkout", "-q", "-b", "other", "HEAD~1")
			r.commit("other")
		},
		"merge commit": func(r gitRepo) {
			r.run(nil, "checkout", "-q", "-b", "side", "HEAD~1")
			r.commit("side")
			r.run(nil, "checkout", "-q", "main")
			r.run(nil, "merge", "-q", "--no-ff", "--no-edit", "side")
		},
		"fast-forward onto older work": func(r gitRepo) {
			r.commit("upstream", old)
		},
		"another committer": func(r gitRepo) {
			r.commit("pr", "GIT_COMMITTER_EMAIL=noreply@github.com")
		},
	}
	for name, move := range cases {
		t.Run(name, func(t *testing.T) {
			r := newGitRepo(t)
			r.commit("a")
			r.commit("b")
			base := r.baselineAt()
			move(r)

			got, rebased := followHead(r.dir, base, nowSnapshot)
			if !rebased {
				t.Fatal("the branch moved and the baseline did not follow")
			}
			if got.Snapshot.Doppel != "now" || got.Head != gitHead(r.dir) {
				t.Errorf("origin %q head %q, want the current tree at %q", got.Snapshot.Doppel, got.Head, gitHead(r.dir))
			}
			if len(got.Reported) != 1 {
				t.Errorf("ledger = %v, want it carried over", got.Reported)
			}
		})
	}
}

// Outside a repository, and for a baseline that never recorded a HEAD, nothing
// is known and nothing moves.
func TestFollowHeadIsInertWithoutAnAnswer(t *testing.T) {
	base := baselineFile{Root: t.TempDir(), CreatedAt: time.Now().UTC().Format(time.RFC3339), Snapshot: nowSnapshot}
	if got, rebased := followHead(base.Root, base, snapshot.Snapshot{}); rebased || got.Head != "" {
		t.Errorf("non-repository: rebased=%v head=%q", rebased, got.Head)
	}

	r := newGitRepo(t)
	r.commit("a")
	legacy := r.baselineAt()
	legacy.Head = ""
	got, rebased := followHead(r.dir, legacy, nowSnapshot)
	if rebased || got.Head != gitHead(r.dir) || got.Snapshot.Doppel != "origin" {
		t.Errorf("legacy baseline: rebased=%v head=%q origin=%q", rebased, got.Head, got.Snapshot.Doppel)
	}
}

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommitScopeIsPairLocal judges one pair twice, once alone and once beside
// a pair in another directory, under both sweep scopes. Under the default
// (walked) scope the second run tracks the other directory, so one commit
// becomes a sweep and one delta becomes a campaign, and the pair's outcome
// moves. Under commit scope it must not: the judgment reads the pair and the
// repository, never the rest of the list.
func TestCommitScopeIsPairLocal(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(rel, src string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// fns renders a file of functions, each with the given extra statements.
	fns := func(pkg string, names []string, body func(name string) string) string {
		var b strings.Builder
		fmt.Fprintf(&b, "package %s\n\n", pkg)
		for _, n := range names {
			fmt.Fprintf(&b, "func %s(x int) int {\n\ty := x * 2\n%s\treturn y\n}\n\n", n, body(n))
		}
		return b.String()
	}
	pair := []string{"F", "G"}
	var others []string
	for i := range 9 {
		others = append(others, fmt.Sprintf("H%d", i))
	}
	// aSteps is what F and G carry after each commit; cSteps what H0..H8 do.
	aSteps := []string{
		"",
		"\ty = y + compute(y, 1)\n",
		"\ty = y + compute(y, 1)\n\ty = y - adjust(y, 7)\n",
		"\ty = y + compute(y, 1)\n\ty = y - adjust(y, 7)\n\ty = y * scale(y, 3)\n",
	}
	cBody := func(step int) func(string) string {
		return func(n string) string {
			switch step {
			case 0:
				return ""
			case 1:
				// Each H changes its own way: no family, but eleven functions
				// modified in one commit.
				return fmt.Sprintf("\ty = y + %s_%d(y)\n", strings.ToLower(n), step)
			}
			// Step 3: five of them take the exact change F and G take in that
			// commit's twin, which makes it a campaign once c is counted.
			if n < "H5" {
				return fmt.Sprintf("\ty = y + %s_%d(y)\n\ty = y * scale(y, 3)\n", strings.ToLower(n), 1)
			}
			return fmt.Sprintf("\ty = y + %s_%d(y)\n", strings.ToLower(n), 1)
		}
	}
	git("init", "-q", "-b", "main")
	write("a/a.go", fns("a", pair, func(string) string { return aSteps[0] }))
	write("c/c.go", fns("c", others, cBody(0)))
	git("add", ".")
	git("commit", "-q", "-m", "base")
	since := git("rev-parse", "HEAD")

	// 1: F and G alike, H0..H8 each its own way — eleven functions.
	write("a/a.go", fns("a", pair, func(string) string { return aSteps[1] }))
	write("c/c.go", fns("c", others, cBody(1)))
	git("commit", "-q", "-am", "add compute step")
	// 2: F and G alike, alone.
	write("a/a.go", fns("a", pair, func(string) string { return aSteps[2] }))
	git("commit", "-q", "-am", "add adjust step")
	// 3: H0..H4 take the scale change, in c only.
	write("c/c.go", fns("c", others, cBody(3)))
	git("commit", "-q", "-am", "scale the c functions")
	// 4: F and G take the same scale change.
	write("a/a.go", fns("a", pair, func(string) string { return aSteps[3] }))
	git("commit", "-q", "-am", "scale the a functions")
	pin := git("rev-parse", "HEAD")

	r, err := openRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.close()

	judgeIn := func(scope string, withC bool) studyPair {
		h := &history{r: r, ts: newTables(r), lives: map[string]*life{}, pinSHA: pin, scope: scope}
		h.lives["a.F"] = &life{key: "a.F", dir: "a", name: "F", file: "a.go"}
		h.lives["a.G"] = &life{key: "a.G", dir: "a", name: "G", file: "a.go"}
		dirs := []string{"a"}
		if withC {
			h.lives["c.H0"] = &life{key: "c.H0", dir: "c", name: "H0", file: "c.go"}
			h.lives["c.H1"] = &life{key: "c.H1", dir: "c", name: "H1", file: "c.go"}
			dirs = append(dirs, "c")
		}
		if err := h.walk(since, pin, pin, dirs); err != nil {
			t.Fatal(err)
		}
		p := studyPair{A: "a.F", B: "a.G"}
		judgeInto(h, &p)
		return p
	}

	alone, beside := judgeIn(scopeWalked, false), judgeIn(scopeWalked, true)
	if alone.Cochanges == beside.Cochanges {
		t.Fatalf("walked scope: fixture does not exercise the dependence (co-changes %d alone and beside)", alone.Cochanges)
	}
	if alone.Cochanges != 3 || beside.Cochanges != 1 {
		t.Errorf("walked scope: co-changes %d alone and %d beside, want 3 and 1", alone.Cochanges, beside.Cochanges)
	}

	ca, cb := judgeIn(scopeCommit, false), judgeIn(scopeCommit, true)
	if fmt.Sprintf("%+v", ca) != fmt.Sprintf("%+v", cb) {
		t.Errorf("commit scope: the pair's judgment depends on the other pairs judged:\nalone  %+v\nbeside %+v", ca, cb)
	}
	if ca.Cochanges != 1 {
		t.Errorf("commit scope: %d co-changes, want 1 (the sweep and the campaign excluded)", ca.Cochanges)
	}
}

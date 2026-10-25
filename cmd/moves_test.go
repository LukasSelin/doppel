package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func movesModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range map[string]string{
		"go.mod": "module m\n\ngo 1.22\n",
		"a/a.go": `package a

func F() int { return helper() }
func helper() int { return 1 }

type Repo struct{}

func Open() *Repo { return &Repo{} }

func G() int { return shared() }
func H() int { return shared() }
func shared() int { return 2 }
`,
		"b/b.go": `package b

import "m/a"

func Use() int { _ = a.Open(); return a.F() + a.G() + a.H() }
`,
		"c/c.go": `package c

import "m/a"

var X = a.H()
`,
	} {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func runMovesCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	// Flags are package-level and cobra does not reset them between runs.
	movesShare, movesBlocked, movesAnchored, movesExclude, movesConfig, movesFormat = 1, false, false, nil, "", formatText
	var out, errb bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errb)
	rootCmd.SetArgs(append([]string{"moves"}, args...))
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	err := rootCmd.Execute()
	return out.String(), err
}

func TestMovesCommandText(t *testing.T) {
	dir := movesModule(t)
	out, err := runMovesCmd(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"#1  a.F  a/a.go:3", "-> b, used by b.Use", "with helper  a/a.go:4"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	// Open is anchored to Repo and G is blocked on shared: both hidden.
	for _, absent := range []string{"a.Open", "a.G"} {
		if strings.Contains(out, absent) {
			t.Errorf("%s shown without --anchored/--blocked:\n%s", absent, out)
		}
	}
}

// JSON carries every move with its status, and is the same bytes twice.
func TestMovesCommandJSON(t *testing.T) {
	dir := movesModule(t)
	first, err := runMovesCmd(t, dir, "--format", "json", "--share", "0.5")
	if err != nil {
		t.Fatal(err)
	}
	second, err := runMovesCmd(t, dir, "--format", "json", "--share", "0.5")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Errorf("two runs differ:\n%s\n---\n%s", first, second)
	}
	var got movesJSON
	if err := json.Unmarshal([]byte(first), &got); err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, m := range got.Moves {
		status[m.Func] = m.Status
	}
	want := map[string]string{"a.F": "clean", "a.Open": "anchored", "a.G": "blocked", "a.H": "blocked"}
	for name, s := range want {
		if status[name] != s {
			t.Errorf("%s status %q, want %q (all: %v)", name, status[name], s, status)
		}
	}
}

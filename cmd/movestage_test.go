package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/dashboard"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
)

// A tree that is not a Go module is the ordinary case for analyze, which reads
// fourteen languages: the stage must skip, return nothing to render, and say
// why in exactly one stderr line.
func TestMovesStageSkipsNonModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\n\nfunc F() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if m := runMovesStage(context.Background(), dir, parser.Excludes{}, 1, &stderr); m != nil {
		t.Fatalf("got a moves section for a tree with no go.mod: %+v", m)
	}
	out := stderr.String()
	if !strings.HasPrefix(out, "Moves: skipped (") || strings.Count(out, "\n") != 1 {
		t.Errorf("want one 'Moves: skipped' line, got:\n%s", out)
	}
}

func TestMovesShouldRun(t *testing.T) {
	goUnits := []parser.CodeUnit{{Name: "F", Lang: "go"}}
	pyUnits := []parser.CodeUnit{{Name: "f", Lang: "python"}}
	p := Params{TestsMode: "exclude"}
	for _, c := range []struct {
		name    string
		p       Params
		units   []parser.CodeUnit
		on      bool
		format  string
		output  string
		wantRun bool
	}{
		{"text report", p, goUnits, true, formatText, "", true},
		{"markdown file", p, goUnits, true, formatJSON, "r.md", true},
		{"bare json renders no section", p, goUnits, true, formatJSON, "", false},
		{"--moves=false", p, goUnits, false, formatText, "r.html", false},
		{"no Go in the population", p, pyUnits, true, formatText, "", false},
		{"--tests only", Params{TestsMode: "only"}, goUnits, true, formatText, "", false},
	} {
		if got := movesShouldRun(c.p, c.units, c.on, c.format, c.output); got != c.wantRun {
			t.Errorf("%s: run = %v, want %v", c.name, got, c.wantRun)
		}
	}
}

// On a module that loads, the stage reports every move with its status, the
// markdown section lists the clean one and counts the rest, and the dashboard
// links each move to its unit.
func TestMovesStageOnModule(t *testing.T) {
	dir := movesModule(t)
	var stderr bytes.Buffer
	m := runMovesStage(context.Background(), dir, parser.Excludes{}, 1, &stderr)
	if m == nil {
		t.Fatalf("no moves section; stderr:\n%s", stderr.String())
	}
	if m.Target != movesTarget || m.Packages != 3 {
		t.Errorf("target %q, packages %d; want %q, 3", m.Target, m.Packages, movesTarget)
	}
	status := map[string]string{}
	for _, r := range m.Rows {
		status[r.Func] = r.Status
	}
	for fn, want := range map[string]string{"a.F": reporter.MoveClean, "a.G": reporter.MoveBlocked, "a.Open": reporter.MoveAnchored} {
		if status[fn] != want {
			t.Errorf("%s: status %q, want %q (all: %v)", fn, status[fn], want, status)
		}
	}
	if !strings.HasPrefix(stderr.String(), "Moves: 3 packages") {
		t.Errorf("stderr:\n%s", stderr.String())
	}

	var md bytes.Buffer
	reporter.PrintMarkdownMoves(&md, m)
	for _, want := range []string{
		"## Functions out of place",
		"built for linux/amd64",
		"| `a.F` a/a.go:3 | `b` | `b.Use` | `helper` |",
		"1 is blocked",
		"1 is anchored",
		"`doppel moves --blocked --anchored`",
	} {
		if !strings.Contains(md.String(), want) {
			t.Errorf("markdown missing %q:\n%s", want, md.String())
		}
	}
	if strings.Contains(md.String(), "a.G") || strings.Contains(md.String(), "a.Open") {
		t.Errorf("markdown lists a blocked or anchored move:\n%s", md.String())
	}

	var txt bytes.Buffer
	reporter.PrintMoves(&txt, m)
	if want := "Functions out of place: 1 used only by one other package could move there as it stands (1 more blocked, 1 anchored"; !strings.Contains(txt.String(), want) {
		t.Errorf("text line %q lacks %q", txt.String(), want)
	}

	res, err := analyze(dir, Params{Threshold: 0.60, MinNodes: 12, ChannelK: 5, TestsMode: "exclude", Generated: "exclude"}, discardWriter{})
	if err != nil {
		t.Fatal(err)
	}
	p := buildDashboard(res, nil, nil, familyStatsZero(), nil, 0, m)
	if p.Moves == nil || p.Moves.Clean != 1 || p.Moves.Blocked != 1 || p.Moves.Anchored != 1 {
		t.Fatalf("payload moves: %+v", p.Moves)
	}
	for _, mv := range p.Moves.List {
		if mv.Unit < 0 {
			t.Errorf("%s not linked to a unit", mv.Func)
			continue
		}
		u := p.Units[mv.Unit]
		if u.Package+"."+u.Name != mv.Func || u.File != mv.File {
			t.Errorf("%s linked to unit %s.%s in %s", mv.Func, u.Package, u.Name, u.File)
		}
	}
	var page bytes.Buffer
	if err := dashboard.Print(&page, p); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.String(), `"moves":{"target":"linux/amd64"`) {
		t.Error("page carries no moves payload")
	}
}

// A run that skipped the stage leaves the page without a moves key at all:
// "not measured" is not "none found".
func TestDashboardWithoutMoves(t *testing.T) {
	p := buildDashboard(sampleResult(), nil, nil, familyStatsZero(), nil, 0, nil)
	if p.Moves != nil {
		t.Fatalf("moves without a stage: %+v", p.Moves)
	}
	var page bytes.Buffer
	if err := dashboard.Print(&page, p); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(page.String(), `"moves":`) {
		t.Error("page carries a moves key for a run that skipped the stage")
	}
}

func TestUncleanPackages(t *testing.T) {
	warnings := "warning: m/a: 2 type error(s), first: a.go:3: x\n" +
		"warning: m/a: go list error\n" +
		"warning: m/b: parse error\n" +
		"warning: m/dep: 1 type error(s), first: d.go:1: y\n"
	if got := uncleanPackages(warnings, map[string]bool{"m/dep": true}); got != 2 {
		t.Errorf("unclean = %d, want 2 (m/a once, m/b; m/dep is outside the graph)", got)
	}
}

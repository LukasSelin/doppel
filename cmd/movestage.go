package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/LukasSelin/doppel/internal/moves"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
)

// movesTarget is the build every analyze report type-checks for, whatever the
// host. `go list` resolves GOOS, GOARCH and cgo from the machine it runs on, so
// a tree with a _windows.go file or a cgo package reads a different file set on
// each platform, and a report must not: CI byte-compares the dashboard across
// Linux, Windows and macOS. Pinning one target is the honest answer; the
// section names it, and a file this build excludes still shows up as the
// "a build-excluded file names what moves" blocker. `doppel moves` reads the
// host's own build instead, because a reader running it asks about their
// machine.
const movesTarget = "linux/amd64"

// movesEnv pins the target and keeps the load offline. GOPROXY=off and
// GOTOOLCHAIN=local are the no-network rule: analyze may not download a module
// or a toolchain to describe a tree, so a module whose dependencies are not
// already in the cache fails to load and the section is skipped with a line
// saying so, exactly like a tree that does not build.
var movesEnv = []string{
	"GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0",
	"GOPROXY=off", "GOTOOLCHAIN=local",
}

// movesShouldRun decides whether analyze runs its moves stage at all. Only a
// surface that renders it pays for a `go list -export`: the text report, a
// markdown file or the dashboard, and not a bare --format json, whose snapshot
// does not carry moves. The population must hold a Go function — which also
// means the Go frontend was selected — so a Python tree does not get a line
// about a missing go.mod; and --tests only skips it, since that report is about
// the test suite and moves never reads test code.
func movesShouldRun(p Params, units []parser.CodeUnit, enabled bool, format, output string) bool {
	if !enabled || p.TestsMode == "only" {
		return false
	}
	if format == formatJSON && output == "" {
		return false
	}
	return slices.ContainsFunc(units, func(u parser.CodeUnit) bool { return u.Lang == "go" })
}

// runMovesStage is analyze's moves stage: the moves analysis over the tree at
// root, reduced to the plain data the report renders, or nil when it could not
// run.
//
// It runs from the analyze command, never inside analyze() or finishAnalyze():
// hooks call analyze() directly, and a stage there would put a `go list` into
// every turn and could write to stderr, which a hook must never do. It never
// fails a report either. analyze reads trees in fourteen languages and code that
// does not build, so every failure — no toolchain, no module, a load error, a
// module with no package in the graph — skips the section with one line on
// stderr saying why. Type errors are counted on one line too: the graph is
// still built from what the checker resolved, as `doppel moves` does.
func runMovesStage(ctx context.Context, root string, exc parser.Excludes, share float64, stderr io.Writer) *reporter.Moves {
	if ctx == nil {
		ctx = context.Background() // a command run outside Execute has none
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(stderr, "Moves: skipped (%v)\n", err)
		return nil
	}
	abs = moves.ResolveRoot(abs) // moveRows relativizes go list's resolved dirs against it
	var warnings bytes.Buffer
	fset, pkgs, err := moves.LoadFor(ctx, abs, movesEnv, exc.SkipDir, &warnings)
	if err != nil {
		fmt.Fprintf(stderr, "Moves: skipped (%s)\n", firstLine(err.Error()))
		return nil
	}
	g := moves.Build(fset, pkgs)
	m := &reporter.Moves{
		Target:    movesTarget,
		Share:     share,
		Functions: g.Functions(),
		Rows:      moveRows(abs, g, moves.Find(g, moves.Options{Share: share})),
	}
	outside := map[string]bool{}
	for _, p := range pkgs {
		if p.InGraph {
			m.Packages++
		} else {
			outside[p.Path] = true
		}
	}
	m.Unclean = uncleanPackages(warnings.String(), outside)
	if m.Packages == 0 {
		fmt.Fprintf(stderr, "Moves: skipped (no package under the root is in the graph)\n")
		return nil
	}
	clean, blocked, anchored := m.Counts()
	fmt.Fprintf(stderr, "Moves: %d packages, %d functions for %s at share %.2f; %d clean, %d blocked, %d anchored\n",
		m.Packages, m.Functions, movesTarget, share, clean, blocked, anchored)
	if m.Unclean > 0 {
		fmt.Fprintf(stderr, "  %d packages did not type-check cleanly, first: %s\n", m.Unclean, firstLine(warnings.String()))
	}
	return m
}

// uncleanPackages counts the distinct packages moves.Load warned about — a go
// list error, a parse failure or type errors, one "warning: <path>: …" line
// each and sometimes two — leaving out the ones outside the graph.
func uncleanPackages(warnings string, outside map[string]bool) int {
	seen := map[string]bool{}
	for _, line := range strings.Split(warnings, "\n") {
		rest, ok := strings.CutPrefix(line, "warning: ")
		if !ok {
			continue
		}
		if path, _, ok := strings.Cut(rest, ": "); ok && !outside[path] {
			seen[path] = true
		}
	}
	return len(seen)
}

// firstLine keeps one line of a message: go list's stderr can run to a page,
// and the stage has promised one line.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i]) + " …"
	}
	return s
}

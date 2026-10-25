package cmd

import (
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/LukasSelin/doppel/internal/moves"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/spf13/cobra"
)

var (
	movesShare    float64
	movesBlocked  bool
	movesAnchored bool
	movesExclude  []string
	movesConfig   string
	movesFormat   string
)

// movesCmd reports functions that live in the wrong package.
//
// It is a command of its own rather than a section of analyze because it
// reads the code differently: through go/types over a module that builds,
// with the go toolchain on PATH, where everything analyze does works on any
// tree in fourteen languages. A report section that failed whenever the tree
// did not type-check would be a worse report.
var movesCmd = &cobra.Command{
	Use:   "moves <path>",
	Short: "Find functions that only another package uses, and what would move with them",
	Long: `Report the functions whose every static reference comes from one other
package, so that moving them there shrinks the surface of the package they
live in.

A function written beside the code that first needed it, and then used only
from somewhere else, is erosion by locally reasonable edit: no rule broke
when it was written, and nothing short of every reference in the corpus
shows it. For each candidate the report names the package to move it to, the
unexported helpers that only it uses and would move along (a type brings its
methods), and the test files that name any of them.

A move is blocked when it would not compile as it stands: it uses unexported
symbols, fields or methods that code staying behind also uses, the target
already declares one of the names, the move would close an import cycle or
break the internal-package rule, or a file this build does not compile (another
GOOS, a build tag) names something that moves. Blocked moves are counted and
listed only with --blocked.

A function whose signature names a type of its own package that stays — a
constructor, an entry point — belongs beside that type however few packages
call it, so those are counted and listed only with --anchored.

Methods are never candidates, nor init, main, or generated code. Calls
through an interface are found by class hierarchy analysis and never block a
move. References from outside the path analysed, and from other modules, are
invisible: an exported function another module calls is not a move.

Go only. The module must type-check, and go list -export builds it once.`,
	Example: `  doppel moves .
  doppel moves . --blocked --anchored
  doppel moves . --share 0.8 --format json`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		path := movesConfig
		if path == "" {
			path = ".doppel.json"
		}
		cfg, err := loadConfig(path)
		if err != nil {
			return err
		}
		if cfg != nil {
			applyConfig(cmd, cfg)
		}
		switch movesFormat {
		case formatText, formatJSON:
		default:
			return fmt.Errorf("invalid --format %q: want %q or %q", movesFormat, formatText, formatJSON)
		}
		if movesShare <= 0 || movesShare > 1 {
			return fmt.Errorf("invalid --share %v: want a fraction in (0, 1]", movesShare)
		}
		return nil
	},
	RunE: runMoves,
}

func init() {
	movesCmd.Flags().Float64Var(&movesShare, "share", 1, "Fraction of a function's referring declarations that must sit in one other package (0–1]. Below 1 the referrers that stay are listed, and must import the target after the move")
	movesCmd.Flags().BoolVar(&movesBlocked, "blocked", false, "Also list moves that would not compile as they stand, and why")
	movesCmd.Flags().BoolVar(&movesAnchored, "anchored", false, "Also list functions whose signature names a type of their own package that stays")
	movesCmd.Flags().StringSliceVar(&movesExclude, "exclude", nil, "Directory patterns to skip, comma-separated, on top of the built-in blocklist. A skipped package is type-checked if imported but its references do not count. Repeatable.")
	movesCmd.Flags().StringVar(&movesConfig, "config", "", "Path to JSON config file (default: .doppel.json if present)")
	movesCmd.Flags().StringVar(&movesFormat, "format", formatText, "Stdout format: text or json")
	rootCmd.AddCommand(movesCmd)
}

func runMoves(cmd *cobra.Command, args []string) error {
	root, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	root = moves.ResolveRoot(root)
	exc, err := parser.NewExcludes(movesExclude)
	if err != nil {
		return err
	}
	stderr := cmd.ErrOrStderr()
	fmt.Fprintf(stderr, "Type-checking %s...\n", root)
	fset, pkgs, err := moves.Load(cmd.Context(), root, exc.SkipDir, stderr)
	if err != nil {
		return err
	}
	g := moves.Build(fset, pkgs)
	found := moves.Find(g, moves.Options{Share: movesShare})

	r := movesReport{root: root, g: g}
	for _, p := range pkgs {
		if p.InGraph {
			r.packages++
		}
	}
	r.functions = g.Functions()
	for _, m := range found {
		switch {
		case len(m.Anchors) > 0:
			r.anchored++
		case len(m.Blocked) > 0:
			r.blocked++
		default:
			r.clean++
		}
		if (len(m.Blocked) == 0 || movesBlocked) && (len(m.Anchors) == 0 || movesAnchored) {
			r.shown = append(r.shown, m)
		}
	}
	fmt.Fprintf(stderr, "Moves: %d packages, %d functions at share %.2f; %d clean, %d blocked, %d anchored to a type that stays\n",
		r.packages, r.functions, movesShare, r.clean, r.blocked, r.anchored)

	if movesFormat == formatJSON {
		return r.writeJSON(cmd.OutOrStdout(), found)
	}
	r.print(cmd.OutOrStdout())
	return nil
}

type movesReport struct {
	root                     string
	g                        *moves.Graph
	packages, functions      int
	clean, blocked, anchored int
	shown                    []moves.Move
}

func (r movesReport) where(s *moves.Sym) string {
	pos := r.g.Pos(s)
	return fmt.Sprintf("%s:%d", parser.RelSlash(r.root, pos.Filename), pos.Line)
}

// target names a package by its directory under the root, which is shorter
// than its import path and unambiguous where its name is not, with the package
// name when the directory does not say it.
func (r movesReport) target(p *moves.Package) string {
	d := parser.RelSlash(r.root, p.Dir)
	if path.Base(d) == p.Name {
		return d
	}
	return d + " (package " + p.Name + ")"
}

func qualifiedSyms(syms []*moves.Sym) []string {
	out := make([]string, len(syms))
	for i, s := range syms {
		out[i] = s.Qualified()
	}
	return out
}

func (r movesReport) print(w io.Writer) {
	if len(r.shown) == 0 {
		fmt.Fprintf(w, "\nNo moves to show among %d functions in %d packages", r.functions, r.packages)
		if hidden := r.blocked + r.anchored; hidden > 0 {
			fmt.Fprintf(w, " (%d hidden: --blocked, --anchored)", hidden)
		}
		fmt.Fprintln(w, ".")
		return
	}
	for i, m := range r.shown {
		fmt.Fprintf(w, "\n#%d  %s  %s\n", i+1, m.Func.Qualified(), r.where(m.Func))
		fmt.Fprintf(w, "    -> %s, used by %s", r.target(m.To), strings.Join(qualifiedSyms(m.Users), ", "))
		if m.Share < 1 {
			fmt.Fprintf(w, " (share %.2f; staying: %s)", m.Share, strings.Join(qualifiedSyms(m.Staying), ", "))
		}
		fmt.Fprintln(w)
		for _, x := range m.With {
			fmt.Fprintf(w, "    with %s  %s\n", x.Name(), r.where(x))
		}
		if len(m.Anchors) > 0 {
			var names []string
			for _, a := range m.Anchors {
				names = append(names, a.Name())
			}
			fmt.Fprintf(w, "    anchored to %s, which stays\n", strings.Join(names, ", "))
		}
		for _, f := range m.TestFiles {
			fmt.Fprintf(w, "    named in %s\n", parser.RelSlash(r.root, f))
		}
		for _, why := range m.Blocked {
			fmt.Fprintf(w, "    blocked: %s\n", why)
		}
	}
}

func (r movesReport) writeJSON(w io.Writer, found []moves.Move) error {
	return reporter.PrintMovesJSON(w, reporter.MovesJSON{
		Share: movesShare, Packages: r.packages, Functions: r.functions, Moves: moveRows(r.root, r.g, found),
	})
}

// moveRows converts every move to plain data, paths relative to root and
// slash-separated. The JSON form of `doppel moves` and the analyze report's
// section are both built from it, so the two cannot describe a move apart.
func moveRows(root string, g *moves.Graph, found []moves.Move) []reporter.MoveRow {
	out := []reporter.MoveRow{}
	for _, m := range found {
		pos := g.Pos(m.Func)
		j := reporter.MoveRow{
			Func:    m.Func.Qualified(),
			File:    parser.RelSlash(root, pos.Filename),
			Line:    pos.Line,
			Status:  reporter.MoveClean,
			To:      m.To.Path,
			ToDir:   parser.RelSlash(root, m.To.Dir),
			Share:   m.Share,
			Users:   qualifiedSyms(m.Users),
			Blocked: m.Blocked,
		}
		switch {
		case len(m.Anchors) > 0:
			j.Status = reporter.MoveAnchored
		case len(m.Blocked) > 0:
			j.Status = reporter.MoveBlocked
		}
		if len(m.Staying) > 0 {
			j.Staying = qualifiedSyms(m.Staying)
		}
		for _, x := range m.With {
			p := g.Pos(x)
			j.With = append(j.With, reporter.MoveSym{Name: x.Name(), File: parser.RelSlash(root, p.Filename), Line: p.Line})
		}
		for _, a := range m.Anchors {
			j.Anchors = append(j.Anchors, a.Name())
		}
		for _, f := range m.TestFiles {
			j.TestFiles = append(j.TestFiles, parser.RelSlash(root, f))
		}
		out = append(out, j)
	}
	return out
}

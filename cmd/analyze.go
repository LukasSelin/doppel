package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/mapper"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/tagger"
	"github.com/spf13/cobra"
)

var (
	threshold  float64
	topN       int
	minNodes   int
	outputFile string
	configFile string
	structMin  float64
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze <path>",
	Short: "Analyze a codebase for structurally similar functions",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		path := configFile
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
		return nil
	},
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().Float64VarP(&threshold, "threshold", "t", 0.60, "Minimum code similarity score (0.0–1.0)")
	analyzeCmd.Flags().IntVarP(&topN, "top", "n", 20, "Maximum number of pairs to show")
	analyzeCmd.Flags().IntVar(&minNodes, "min-nodes", 12, "Skip functions whose body has fewer than this many AST nodes")
	analyzeCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write report as markdown to this file (e.g. report.md). Stdout text report is still printed.")
	analyzeCmd.Flags().StringVar(&configFile, "config", "", "Path to JSON config file (default: .doppel.json if present)")
	analyzeCmd.Flags().Float64Var(&structMin, "struct-min", 0.0, "Minimum structural overlap score (0.0–1.0) to keep a pair")
	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	root := args[0]

	fmt.Fprintf(os.Stderr, "Scanning %s ...\n", root)
	var units []parser.CodeUnit
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() && shouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		parsed, err := parser.Parse(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  warn: %s: %v\n", path, err)
			return nil
		}
		units = append(units, parsed...)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", root, err)
	}

	if len(units) == 0 {
		fmt.Println("No functions found.")
		return nil
	}

	for i := range units {
		units[i].Patterns = tagger.Tag(units[i].Body)
	}

	// Build call graph and generate concept documents for every unit.
	// docs[i] describes units[i]; the pipeline relies on that alignment.
	cg := concepter.BuildCallGraph(units)

	fmt.Fprintf(os.Stderr, "Generating concept documents...\n")
	cptr := concepter.New()
	docs := mapper.Map(units, cg, cptr)

	fmt.Fprintf(os.Stderr, "Found %d functions. Comparing fingerprints...\n", len(units))
	pairs := analyzer.FindSimilar(units, threshold, topN, minNodes)

	// Attach structural evidence to each surviving pair.
	if len(pairs) > 0 {
		fmt.Fprintf(os.Stderr, "Running structural comparison on %d pairs...\n", len(pairs))
		for i := range pairs {
			ev := comparator.Compare(docs[pairs[i].AIdx], docs[pairs[i].BIdx])
			pairs[i].Evidence = &ev
		}

		// Filter by structural overlap threshold if set.
		if structMin > 0 {
			filtered := pairs[:0]
			for _, p := range pairs {
				if p.Evidence != nil && p.Evidence.OverlapScore >= structMin {
					filtered = append(filtered, p)
				}
			}
			pairs = filtered
			fmt.Fprintf(os.Stderr, "  %d pairs remain after struct-min=%.2f filter\n", len(pairs), structMin)
		}
	}

	reporter.Print(os.Stdout, pairs, threshold, len(units))

	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		reporter.PrintMarkdown(f, pairs, threshold, len(units))
		fmt.Fprintf(os.Stderr, "Markdown report written to %s\n", outputFile)
	}

	return nil
}

func shouldSkipDir(name string) bool {
	skip := map[string]bool{
		".git": true, ".claude": true, "vendor": true,
		"testdata": true, "build": true,
		".idea": true, ".vscode": true,
	}
	return skip[name]
}

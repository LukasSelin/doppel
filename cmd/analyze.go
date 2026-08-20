package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/lukse/doppel/internal/analyzer"
	"github.com/lukse/doppel/internal/comparator"
	"github.com/lukse/doppel/internal/concepter"
	"github.com/lukse/doppel/internal/mapper"
	"github.com/lukse/doppel/internal/ontology"
	"github.com/lukse/doppel/internal/parser"
	"github.com/lukse/doppel/internal/reporter"
	"github.com/lukse/doppel/internal/retriever"
	"github.com/lukse/doppel/internal/tagger"
	"github.com/spf13/cobra"
)

var (
	threshold  float64
	topN       int
	minNodes   int
	outputFile string
	configFile string
	structMin  float64
	channelK   int
	debugFlag  bool
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
	analyzeCmd.Flags().Float64VarP(&threshold, "threshold", "t", 0.60, "Minimum code-shape score for structural-channel candidates (0.0–1.0)")
	analyzeCmd.Flags().IntVarP(&topN, "top", "n", 20, "Maximum number of pairs in the final report")
	analyzeCmd.Flags().IntVar(&minNodes, "min-nodes", 12, "Exclude functions with fewer body AST nodes from structural retrieval")
	analyzeCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write report as markdown to this file (e.g. report.md). Stdout text report is still printed.")
	analyzeCmd.Flags().StringVar(&configFile, "config", "", "Path to JSON config file (default: .doppel.json if present)")
	analyzeCmd.Flags().Float64Var(&structMin, "struct-min", 0.0, "Minimum structural overlap score (0.0–1.0) to keep a pair")
	analyzeCmd.Flags().IntVar(&channelK, "channel-k", 5, "Candidates each function keeps per retrieval channel")
	analyzeCmd.Flags().BoolVar(&debugFlag, "debug", false, "Show per-pair retrieval provenance in the report")
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

	// Tag every unit, counting tag occurrences as we go: the counts become
	// the corpus statistics that weight concept matching. A tag most units
	// carry says little about any pair sharing it; a rare one says a lot.
	tagCounts := make(map[ontology.TermID]int)
	for i := range units {
		units[i].Patterns = tagger.Tag(units[i])
		for _, tag := range units[i].Patterns {
			tagCounts[ontology.TermID(tag)]++
		}
	}
	onto := ontology.Default()
	ic := ontology.NewCorpusIC(onto, tagCounts)
	scorer := ontology.NewScorer(onto, ic)
	comp := comparator.New(scorer)

	// Build call graph and generate concept documents for every unit.
	// docs[i] describes units[i]; the pipeline relies on that alignment.
	cg := concepter.BuildCallGraph(units)

	fmt.Fprintf(os.Stderr, "Generating concept documents...\n")
	cptr := concepter.New()
	docs := mapper.Map(units, cg, cptr)

	// Multi-channel candidate retrieval: structural shape, shared concepts,
	// and shared resolved calls each retrieve per-function top-K neighbors
	// weighted by corpus rarity; the union goes to the expensive comparator.
	fmt.Fprintf(os.Stderr, "Found %d functions. Retrieving candidates...\n", len(units))
	opts := retriever.DefaultOptions()
	opts.ChannelK = channelK
	opts.Threshold = threshold
	opts.MinNodes = minNodes
	cands, stats := retriever.Retrieve(units, cg, onto, ic, opts)
	printRetrievalStats(os.Stderr, stats)

	pairs := make([]analyzer.SimilarPair, len(cands))
	for i, c := range cands {
		pairs[i] = analyzer.SimilarPair{
			A:         units[c.AIdx],
			B:         units[c.BIdx],
			AIdx:      c.AIdx,
			BIdx:      c.BIdx,
			Score:     c.Breakdown.Score,
			Breakdown: c.Breakdown,
			Retrieval: &analyzer.Retrieval{
				Shape:    c.Shape,
				Concept:  c.Concept,
				Call:     c.Call,
				Total:    c.Total,
				Channels: c.Channels,
			},
		}
	}

	// Attach structural evidence to every candidate pair.
	if len(pairs) > 0 {
		fmt.Fprintf(os.Stderr, "Running structural comparison on %d pairs...\n", len(pairs))
		for i := range pairs {
			ev := comp.Compare(docs[pairs[i].AIdx], docs[pairs[i].BIdx])
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

	// Final ranking: retrieval evidence mass decides the report order; the
	// code-shape and overlap scores stay unblended, displayed per pair.
	pairs = analyzer.SortByEvidence(pairs, topN)

	meta := reporter.Meta{Threshold: threshold, TotalFuncs: len(units), Debug: debugFlag}
	reporter.Print(os.Stdout, pairs, meta)

	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		reporter.PrintMarkdown(f, pairs, meta)
		fmt.Fprintf(os.Stderr, "Markdown report written to %s\n", outputFile)
	}

	return nil
}

// printRetrievalStats summarizes one retrieval run on stderr: how much each
// channel contributed and how much trivial idiom mass was suppressed. This is
// diagnostic output for tuning and evaluation, never part of the report.
func printRetrievalStats(w io.Writer, s retriever.Stats) {
	fmt.Fprintf(w, "Retrieval: shape %d, concept %d, call %d -> %d unique pairs\n",
		s.ShapePairs, s.ConceptPairs, s.CallPairs, s.Union)
	pct := func(n int) float64 {
		if s.Union == 0 {
			return 0
		}
		return 100 * float64(n) / float64(s.Union)
	}
	fmt.Fprintf(w, "  concept-only %.1f%%  call-only %.1f%%  suppressed-shape functions: %d  large identity buckets: %d\n",
		pct(s.OnlyConcept), pct(s.OnlyCall), s.Suppressed, s.LargeBuckets)
}

func shouldSkipDir(name string) bool {
	skip := map[string]bool{
		".git": true, ".claude": true, "vendor": true,
		"testdata": true, "build": true,
		".idea": true, ".vscode": true,
	}
	return skip[name]
}

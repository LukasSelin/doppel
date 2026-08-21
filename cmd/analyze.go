package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/lukse/doppel/internal/analyzer"
	"github.com/lukse/doppel/internal/comparator"
	"github.com/lukse/doppel/internal/concepter"
	"github.com/lukse/doppel/internal/culture"
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

	// Model the corpus's own conceptual practice: which concepts/roles/calls
	// co-occur beyond chance, and how each concept is normally realized here.
	cult := culture.Build(units, docs, cg, culture.DefaultOptions())
	cs := cult.Stats()
	fmt.Fprintf(os.Stderr, "Culture: %d concepts modeled, %d associations, %d unusual realizations\n",
		cs.ConceptsModeled, cs.AssociationCount, cs.UnusualRealizations)
	printHabitatSummary(os.Stderr, cs)
	printArenaSummary(os.Stderr, cs)

	// Multi-channel candidate retrieval: structural shape, shared concepts,
	// and shared resolved calls each retrieve per-function top-K neighbors
	// weighted by corpus rarity; the union goes to the expensive comparator.
	fmt.Fprintf(os.Stderr, "Found %d functions. Retrieving candidates...\n", len(units))
	opts := retriever.DefaultOptions()
	opts.ChannelK = channelK
	opts.Threshold = threshold
	opts.MinNodes = minNodes
	if debugFlag {
		opts.ChainTopN = 20 // the "full list", bounded
	}
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
				Shape:      c.Shape,
				Concept:    c.Concept,
				Call:       c.Call,
				Total:      c.Total,
				TrophicSim: c.TrophicSim,
				Channels:   c.Channels,
				Chains:     sharedChains(c.Chains),
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

	// Annotate surviving pairs with unusual concept realizations and habitat
	// misfits — positional lookup, like Evidence attachment; never name-keyed.
	for i := range pairs {
		pairs[i].Culture = cultureNotes(cult, pairs[i].AIdx, pairs[i].BIdx,
			pairs[i].A.Patterns, pairs[i].B.Patterns)
		pairs[i].Habitat = habitatNotes(cult, pairs[i].AIdx, pairs[i].BIdx,
			pairs[i].A.Package, pairs[i].B.Package)
		pairs[i].Profile = profileNotes(cult, pairs[i].AIdx, pairs[i].BIdx,
			pairs[i].A.Patterns, pairs[i].B.Patterns)
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
	fmt.Fprintf(w, "  concept-only %.1f%%  call-only %.1f%%  suppressed-shape functions: %d  large identity buckets: %d  surviving patterns: %d\n",
		pct(s.OnlyConcept), pct(s.OnlyCall), s.Suppressed, s.LargeBuckets, s.SurvivingPatterns)
}

// sharedChains bridges retriever chain explanations into the analyzer's
// plain-data mirror.
func sharedChains(chains []retriever.SharedPattern) []analyzer.SharedChain {
	if len(chains) == 0 {
		return nil
	}
	out := make([]analyzer.SharedChain, len(chains))
	for i, c := range chains {
		out[i] = analyzer.SharedChain{Level: c.Level, Energy: c.Energy, Render: c.Render}
	}
	return out
}

// cultureNotes flags unusual concept realizations on a pair's shared tags:
// the pair report explains this pair, and "you both claim transaction but one
// side does it unlike anything else here" is the drift-vs-duplication signal.
// Tags ascend, side A precedes B, so note order is deterministic.
func cultureNotes(cult *culture.Model, aIdx, bIdx int, aTags, bTags []string) []analyzer.CultureNote {
	inB := make(map[string]bool, len(bTags))
	for _, t := range bTags {
		inB[t] = true
	}
	shared := make(map[string]bool)
	for _, t := range aTags {
		if inB[t] {
			shared[t] = true
		}
	}
	tags := make([]string, 0, len(shared))
	for t := range shared {
		tags = append(tags, t)
	}
	sort.Strings(tags)

	var notes []analyzer.CultureNote
	for _, tag := range tags {
		for _, side := range []struct {
			label string
			idx   int
		}{{"A", aIdx}, {"B", bIdx}} {
			if !cult.Atypical(side.idx, tag) {
				continue
			}
			typ, _ := cult.Typicality(side.idx, tag)
			median, _ := cult.Median(tag)
			convention, _ := cult.ConventionStrength(tag)
			note := analyzer.CultureNote{
				Tag:           tag,
				Side:          side.label,
				Typicality:    typ,
				ConceptMedian: median,
				Convention:    convention,
			}
			for _, ch := range cult.ChannelTypicality(side.idx, tag) {
				note.Channels = append(note.Channels, analyzer.CultureChannel{
					Name: ch.Name, Typicality: ch.Typicality,
				})
			}
			notes = append(notes, note)
		}
	}
	return notes
}

// habitatNotes flags pair sides that are notably out of place in their own
// package. Positional lookup, A before B; only misfits carry notes, so the
// default report stays lean.
func habitatNotes(cult *culture.Model, aIdx, bIdx int, aPkg, bPkg string) []analyzer.HabitatNote {
	var notes []analyzer.HabitatNote
	for _, side := range []struct {
		label string
		idx   int
		pkg   string
	}{{"A", aIdx, aPkg}, {"B", bIdx, bPkg}} {
		if !cult.Misfit(side.idx) {
			continue
		}
		fit, _ := cult.HabitatFit(side.idx)
		norm, _ := cult.HabitatNorm(side.pkg)
		note := analyzer.HabitatNote{
			Side:        side.label,
			Package:     side.pkg,
			Fit:         fit,
			PackageNorm: norm,
		}
		for _, ch := range cult.ChannelSurprise(side.idx) {
			note.Channels = append(note.Channels, analyzer.HabitatChannel{
				Name: ch.Name, Surprise: ch.Surprise,
			})
		}
		notes = append(notes, note)
	}
	return notes
}

// profileNotes attaches each side's arena equilibrium — positional lookup,
// A before B. A side qualifies when it carries at least one tag and has a
// profile, which is exactly when a tags: line already renders, so profiles
// add no lines to previously silent units.
func profileNotes(cult *culture.Model, aIdx, bIdx int, aTags, bTags []string) []analyzer.ProfileNote {
	var notes []analyzer.ProfileNote
	for _, side := range []struct {
		label string
		idx   int
		tags  []string
	}{{"A", aIdx, aTags}, {"B", bIdx, bTags}} {
		if len(side.tags) == 0 {
			continue
		}
		p, ok := cult.ArenaProfile(side.idx)
		if !ok {
			continue
		}
		note := analyzer.ProfileNote{
			Side:      side.label,
			State:     p.State,
			Rounds:    p.Rounds,
			Converged: p.Converged,
		}
		for _, cm := range p.Survivors {
			note.Concepts = append(note.Concepts, analyzer.ProfileMass{Tag: cm.Tag, Mass: cm.Mass})
		}
		for _, cm := range p.Extinct {
			note.Extinct = append(note.Extinct, analyzer.ProfileMass{Tag: cm.Tag, Mass: cm.Mass})
		}
		notes = append(notes, note)
	}
	return notes
}

// printArenaSummary emits the ecosystem stderr line.
func printArenaSummary(w io.Writer, s culture.Stats) {
	if s.ArenaProfiled == 0 {
		fmt.Fprintf(w, "Ecosystems: 0 profiled\n")
		return
	}
	fmt.Fprintf(w, "Ecosystems: %d profiled (%d dominance, %d coalition, %d conflict, %d weak)\n",
		s.ArenaProfiled, s.ArenaDominance, s.ArenaCoalition, s.ArenaConflict, s.ArenaWeak)
}

// printHabitatSummary emits the habitat and convention stderr lines in human
// vocabulary. Superlatives are omitted entirely when nothing is modeled.
func printHabitatSummary(w io.Writer, s culture.Stats) {
	if s.HabitatsModeled == 0 {
		fmt.Fprintf(w, "Habitats: 0 modeled\n")
	} else {
		fmt.Fprintf(w, "Habitats: %d modeled, %d misfits; most uniform %s (norm %.2f), most diverse %s (norm %.2f)\n",
			s.HabitatsModeled, s.HabitatMisfits,
			s.MostUniformHabitat, s.MostUniformNorm,
			s.MostDiverseHabitat, s.MostDiverseNorm)
	}
	if s.ConceptsModeled >= 1 {
		fmt.Fprintf(w, "Conventions: strongest %s (%.2f), loosest %s (%.2f)\n",
			s.StrongestConvention, s.StrongestConventionStrength,
			s.LoosestConvention, s.LoosestConventionStrength)
	}
}

func shouldSkipDir(name string) bool {
	skip := map[string]bool{
		".git": true, ".claude": true, "vendor": true,
		"testdata": true, "build": true,
		".idea": true, ".vscode": true,
	}
	return skip[name]
}

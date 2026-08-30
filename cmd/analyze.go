package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/calibrate"
	"github.com/LukasSelin/doppel/internal/culture"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/retriever"
	"github.com/spf13/cobra"
)

var (
	threshold     float64
	topN          int
	minNodes      int
	outputFile    string
	configFile    string
	structMin     float64
	channelK      int
	debugFlag     bool
	maxPerFunc    int
	testsMode     string
	genMode       string
	calibrateRate float64
	familiesN     int
	familyMin     float64

	outputFormat string
)

// Output formats for --format.
const (
	formatText = "text"
	formatJSON = "json"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze <path>",
	Short: "Report a codebase's erosion: duplicate work, diverged copies, functions out of place",
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
		// Validated after applyConfig so a bad value in .doppel.json is
		// rejected exactly like a bad value on the command line.
		switch outputFormat {
		case formatText, formatJSON:
		default:
			return fmt.Errorf("invalid --format %q: want %q or %q", outputFormat, formatText, formatJSON)
		}
		if err := validateMode("tests", testsMode); err != nil {
			return err
		}
		return validateMode("generated", genMode)
	},
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().Float64VarP(&threshold, "threshold", "t", 0.60, "Minimum code-shape score for structural-channel candidates (0.0–1.0)")
	analyzeCmd.Flags().IntVarP(&topN, "top", "n", 20, "Maximum number of pairs in the final report")
	analyzeCmd.Flags().IntVar(&minNodes, "min-nodes", 18, "Exclude functions with fewer body AST nodes from structural retrieval")
	analyzeCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write a report to this file. A .html path renders the full visual report; any other extension writes markdown. Stdout text report is still printed.")
	analyzeCmd.Flags().StringVar(&configFile, "config", "", "Path to JSON config file (default: .doppel.json if present)")
	analyzeCmd.Flags().Float64Var(&structMin, "struct-min", 0.0, "Minimum structural overlap score (0.0–1.0) to keep a pair")
	analyzeCmd.Flags().IntVar(&channelK, "channel-k", 5, "Candidates each function keeps per retrieval channel")
	analyzeCmd.Flags().BoolVar(&debugFlag, "debug", false, "Show per-pair retrieval provenance in the report")
	analyzeCmd.Flags().IntVar(&maxPerFunc, "max-per-func", 2, "Maximum pairs any one function may appear in in the final report (0 = no cap)")
	analyzeCmd.Flags().StringVar(&testsMode, "tests", "exclude", "Test-function population: include, exclude, or only. Tests are conventionally similar, so the default models production code; cross test/prod pairs are never reported.")
	analyzeCmd.Flags().StringVar(&genMode, "generated", "exclude", "Generated-file population: include, exclude, or only. Files carrying Go's \"Code generated ... DO NOT EDIT.\" marker are near-identical by construction and unactionable, so the default models hand-written code.")
	analyzeCmd.Flags().Float64Var(&calibrateRate, "calibrate", 0, "Set --threshold and --struct-min from the corpus: admit this fraction of random unrelated pairs (e.g. 0.01). Overrides both flags and sets --family-min to the same code-shape floor; 0 = off")
	analyzeCmd.Flags().StringVar(&outputFormat, "format", formatText, "Stdout format: text or json")
	analyzeCmd.Flags().IntVar(&familiesN, "families", 5, "Near-duplicate families to show in the report (0 = no families section)")
	analyzeCmd.Flags().Float64Var(&familyMin, "family-min", 0.60, "Minimum code-shape between every two members of a family (0.0–1.0)")
	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	p := Params{
		Threshold:  threshold,
		TopN:       topN,
		MinNodes:   minNodes,
		StructMin:  structMin,
		ChannelK:   channelK,
		MaxPerFunc: maxPerFunc,
		TestsMode:  testsMode,
		Generated:  genMode,
		Calibrate:  calibrateRate,
		Debug:      debugFlag,
	}

	res, err := analyze(args[0], p, cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	if len(res.Units) == 0 && outputFormat != formatJSON {
		fmt.Fprintln(cmd.OutOrStdout(), "No functions found.")
		return nil
	}

	// Families are built from the full filtered pair set, never the ranked
	// one: --max-per-func is a report-time device applied after scoring, so
	// the pair list may show two of a function's edges while a family
	// legitimately rests on eight. Built before SortForReport, which sorts
	// res.Pairs in place, so the census cannot depend on ranking order.
	fams, famStats := buildFamilies(res, cmd.ErrOrStderr())

	// Final ranking: corroborated evidence — retrieval mass discounted by
	// architectural corroboration and structural similarity — with a
	// per-function diversity cap. The displayed scores stay unblended.
	pairs, suppressed := analyzer.SortForReport(res.Pairs, topN, maxPerFunc)
	if suppressed > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "  %d pairs suppressed by max-per-func=%d\n", suppressed, maxPerFunc)
	}

	// One Meta for both renderers. They used to build their own literals, which
	// is how a new field ends up set on one surface and forgotten on the other.
	// The overview rides on it and is markdown-only: the text report has no way
	// to draw a diagram, and the terminal already gets these numbers on stderr.
	meta := reporter.Meta{Threshold: threshold, TotalFuncs: len(res.Units), Debug: debugFlag}
	if outputFile != "" {
		meta.Overview = buildOverview(res, suppressed)
	}

	if outputFormat == formatJSON {
		// The snapshot describes what this run reports, so it carries the same
		// ranked pair set the text report shows. A snapshot meant for diffing
		// is taken by `doppel hook`, which deliberately runs uncapped.
		if err := reporter.PrintJSON(cmd.OutOrStdout(), snapshotOf(res, pairs)); err != nil {
			return err
		}
	} else {
		reporter.Print(cmd.OutOrStdout(), pairs, meta)
		reporter.PrintFamilies(cmd.OutOrStdout(), fams, famStats, res.Units, familiesN)
	}

	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		// The extension picks the format. HTML is deliberately not a --format
		// value: --format selects what goes to stdout, and a page of markup on
		// stdout helps nobody — the file is the only place it makes sense.
		if isHTMLPath(outputFile) {
			r := buildHTMLReport(res, meta.Overview, fams, famStats, pairs, suppressed)
			if err := reporter.PrintHTML(f, r); err != nil {
				return fmt.Errorf("write html report: %w", err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "HTML report written to %s\n", outputFile)
		} else {
			reporter.PrintMarkdown(f, pairs, meta)
			reporter.PrintMarkdownFamilies(f, fams, famStats, res.Units, familiesN)
			fmt.Fprintf(cmd.ErrOrStderr(), "Markdown report written to %s\n", outputFile)
		}
	}

	return nil
}

// printRetrievalStats summarizes one retrieval run on stderr: how much each
// channel contributed and how much trivial idiom mass was suppressed.
//
// The channel mix also reaches the markdown report now, because it answers
// "why these pairs and not others" and a reader needs that to weigh the list.
// The tuning counters below it — suppressed functions, large buckets,
// surviving patterns — stay here: they help someone calibrating doppel and
// mean nothing to someone reading about their own code.
func printRetrievalStats(w io.Writer, s retriever.Stats) {
	fmt.Fprintf(w, "Retrieval: shape %d, concept %d, call %d -> %d unique pairs\n",
		s.ShapePairs, s.ConceptPairs, s.CallPairs, s.Union)
	pct := func(n int) float64 {
		if s.Union == 0 {
			return 0
		}
		return 100 * float64(n) / float64(s.Union)
	}
	fmt.Fprintf(w, "  concept-only %.1f%%  call-only %.1f%%  suppressed-shape functions: %d  large identity buckets: %d  surviving labels: %d\n",
		pct(s.OnlyConcept), pct(s.OnlyCall), s.Suppressed, s.LargeBuckets, s.SurvivingLabels)
	// Only when a nats floor derived the caps: the absolute caps are the
	// documented constants and need no line.
	if s.CapsDerived {
		fmt.Fprintf(w, "  caps: label df<=%d, call df<=%d%s\n", s.LabelCap, s.CallCap, emptyChannels(s))
	}
}

// emptyChannels names channels whose derived cap fell below 2 — no feature
// can both pair and meet the floor there.
func emptyChannels(s retriever.Stats) string {
	var out []string
	if s.LabelCap < 2 {
		out = append(out, "label")
	}
	if s.CallCap < 2 {
		out = append(out, "call")
	}
	if len(out) == 0 {
		return ""
	}
	return " (" + strings.Join(out, ", ") + " channel empty)"
}

// sharedChains bridges retriever chain explanations into the analyzer's
// plain-data mirror.
func sharedChains(chains []retriever.SharedLabel) []analyzer.SharedChain {
	if len(chains) == 0 {
		return nil
	}
	out := make([]analyzer.SharedChain, len(chains))
	for i, c := range chains {
		out[i] = analyzer.SharedChain{
			Depth:  c.Depth,
			Count:  c.Count,
			Energy: c.Energy,
			Render: c.Render,
			Label:  c.Label,
		}
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
		// A confirmed misfit with a modeled subsystem was alien there too;
		// the note says so, with the subsystem fit for contrast.
		if key, sfit, ok := cult.SubsystemFit(side.idx); ok {
			note.Subsystem, note.SubsystemFit = key, sfit
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

// printCalibration reports a null calibration: the derived thresholds, or
// why the corpus was too small to derive them. Printed only when
// --calibrate is on, so the default stderr stays byte-identical.
func printCalibration(w io.Writer, r calibrate.Result) {
	if r.Declined != "" {
		fmt.Fprintf(w, "Calibration: rate %g declined (%s); defaults kept\n", r.Rate, r.Declined)
		return
	}
	pairs := fmt.Sprintf("%d", r.ShapePairs)
	if r.OverlapPairs != r.ShapePairs {
		pairs = fmt.Sprintf("%d shape / %d overlap", r.ShapePairs, r.OverlapPairs)
	}
	fmt.Fprintf(w, "Calibration: rate %g over %s null pairs -> threshold %.2f, struct-min %.2f, family-min %.2f\n",
		r.Rate, pairs, r.Threshold, r.StructMin, r.Threshold)
}

// printHabitatSummary emits the habitat and convention stderr lines in human
// vocabulary. Superlatives are omitted entirely when nothing is modeled.
func printHabitatSummary(w io.Writer, s culture.Stats) {
	switch {
	case s.HabitatsModeled == 0:
		fmt.Fprintf(w, "Habitats: 0 modeled\n")
	case s.SubsystemsModeled == 0:
		fmt.Fprintf(w, "Habitats: %d modeled, %d misfits; most uniform %s (norm %.2f), most diverse %s (norm %.2f)\n",
			s.HabitatsModeled, s.HabitatMisfits,
			s.MostUniformHabitat, s.MostUniformNorm,
			s.MostDiverseHabitat, s.MostDiverseNorm)
	default:
		// With subsystems modeled, a package misfit that fits its parent
		// directory is excused rather than reported; the count keeps the
		// excuse visible.
		fmt.Fprintf(w, "Habitats: %d modeled, %d misfits (%d excused by subsystem), %d subsystems; most uniform %s (norm %.2f), most diverse %s (norm %.2f)\n",
			s.HabitatsModeled, s.HabitatMisfits, s.MisfitsExcused, s.SubsystemsModeled,
			s.MostUniformHabitat, s.MostUniformNorm,
			s.MostDiverseHabitat, s.MostDiverseNorm)
	}
	if s.ConceptsModeled >= 1 {
		fmt.Fprintf(w, "Conventions: strongest %s (%.2f), loosest %s (%.2f)\n",
			s.StrongestConvention, s.StrongestConventionStrength,
			s.LoosestConvention, s.LoosestConventionStrength)
	}
}

// isTestUnit reports whether a unit lives in a _test.go file — a
// compiler-recognized build distinction, not a naming heuristic.
func isTestUnit(u parser.CodeUnit) bool {
	return strings.HasSuffix(u.File, "_test.go")
}

// filterTestUnits applies the --tests population policy. Filtering preserves
// slice order, so downstream positional alignment is untouched.
func filterTestUnits(units []parser.CodeUnit, mode string) []parser.CodeUnit {
	if mode == "include" {
		return units
	}
	keepTests := mode == "only"
	kept := units[:0]
	for _, u := range units {
		if isTestUnit(u) == keepTests {
			kept = append(kept, u)
		}
	}
	return kept
}

// filterGeneratedUnits applies the --generated population policy, keyed on
// Go's own "Code generated ... DO NOT EDIT." convention detected at parse
// time. Same shape as filterTestUnits and run in the same place — before any
// corpus statistic — because generated code is near-identical by construction
// and, left in, owns the top of every large corpus's report (moby's entire
// top ten was protobuf Unmarshal methods).
func filterGeneratedUnits(units []parser.CodeUnit, mode string) []parser.CodeUnit {
	if mode == "include" {
		return units
	}
	keepGenerated := mode == "only"
	kept := units[:0]
	for _, u := range units {
		if u.Generated == keepGenerated {
			kept = append(kept, u)
		}
	}
	return kept
}

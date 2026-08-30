package cmd

import (
	"fmt"
	"io"

	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/spf13/cobra"
)

var (
	familiesThreshold float64
	familiesMinNodes  int
	familiesChannelK  int
	familiesTests     string
	familiesLanguages []string
	familiesGenerated string
	familiesCalibrate float64
	familiesMin       float64
	familiesConfig    string
	familiesFormat    string
)

// familiesCmd is the census view: every near-duplicate family in the corpus,
// with no presentation cutoff.
//
// It is a separate command rather than a flag on analyze because the two
// answer different questions. analyze answers "what should I look at", and
// bounds itself accordingly — top-N, the per-function diversity cap, a handful
// of families. A census answers "how much of this is there", and a census with
// a top-N is not a census.
var familiesCmd = &cobra.Command{
	Use:   "families <path>",
	Short: "Census of near-duplicate families: groups of 3+ mutually similar functions",
	Long: `Report every near-duplicate family in a corpus.

A family is a set of functions in which every member is at least --family-min
alike to every other member — not a chain of pairwise resemblances. A~B and
B~C says nothing about A~C, so families are maximal cliques and the report
states the weakest edge, which is the claim a reader can check.

A function may belong to more than one family; the counts report distinct
functions.`,
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		path := familiesConfig
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
		// After applyConfig, so a config-pinned threshold counts as explicit.
		calibrationOptOut(cmd, &familiesCalibrate)
		switch familiesFormat {
		case formatText, formatJSON:
		default:
			return fmt.Errorf("invalid --format %q: want %q or %q", familiesFormat, formatText, formatJSON)
		}
		if err := validateMode("tests", familiesTests); err != nil {
			return err
		}
		return validateMode("generated", familiesGenerated)
	},
	RunE: runFamilies,
}

func init() {
	familiesCmd.Flags().Float64VarP(&familiesThreshold, "threshold", "t", 0.60, "Pin the code-shape floor for structural-channel candidates (0.0–1.0), turning off --calibrate.")
	familiesCmd.Flags().IntVar(&familiesMinNodes, "min-nodes", 12, "Exclude functions with fewer body AST nodes from structural retrieval")
	familiesCmd.Flags().IntVar(&familiesChannelK, "channel-k", 5, "Candidates each function keeps per retrieval channel")
	familiesCmd.Flags().StringSliceVar(&familiesLanguages, "languages", nil, "Languages to read, comma-separated (default: every language doppel has a frontend for). The extension allowlist is the whole scope rule — a file is in the corpus because a frontend claims its extension, never because its contents looked like code.")
	familiesCmd.Flags().StringVar(&familiesTests, "tests", "exclude", "Test-function population: include, exclude, or only")
	familiesCmd.Flags().StringVar(&familiesGenerated, "generated", "exclude", "Generated-file population: include, exclude, or only")
	familiesCmd.Flags().Float64Var(&familiesCalibrate, "calibrate", defaultCalibrateRate, "Fraction of random unrelated pairs the thresholds may admit. Derives --threshold and --family-min from this corpus; 0 = use the fixed defaults")
	familiesCmd.Flags().Float64Var(&familiesMin, "family-min", 0.60, "Pin the code-shape between every two members of a family (0.0–1.0), turning off --calibrate.")
	familiesCmd.Flags().StringVar(&familiesConfig, "config", "", "Path to JSON config file (default: .doppel.json if present)")
	familiesCmd.Flags().StringVar(&familiesFormat, "format", formatText, "Stdout format: text or json")
	for _, name := range []string{"channel-k", "min-nodes"} {
		_ = familiesCmd.Flags().MarkHidden(name)
	}
	rootCmd.AddCommand(familiesCmd)
}

func runFamilies(cmd *cobra.Command, args []string) error {
	// TopN, MaxPerFunc and StructMin stay zero for the same reason a hook run
	// zeroes them: they drop pairs for presentation reasons, and a pair that
	// fell past rank 20 is still part of the corpus's duplication.
	p := Params{
		Threshold: familiesThreshold,
		MinNodes:  familiesMinNodes,
		ChannelK:  familiesChannelK,
		TestsMode: familiesTests,
		Languages: familiesLanguages,
		Generated: familiesGenerated,
		Calibrate: familiesCalibrate,
	}

	res, err := analyze(args[0], p, cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	if len(res.Units) == 0 && familiesFormat != formatJSON {
		fmt.Fprintln(cmd.OutOrStdout(), "No functions found.")
		return nil
	}

	o := family.DefaultOptions()
	o.Min = familyMinFor(res, familiesMin)
	fams, stats := family.Build(res.Units, res.Pairs, o)
	printFamilyStats(cmd.ErrOrStderr(), stats)

	if familiesFormat == formatJSON {
		return reporter.PrintFamiliesJSON(cmd.OutOrStdout(), fams, stats, res.Units, res.Root)
	}
	if len(fams) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nNo families found among %d functions. Nothing here has three or more mutually similar members at code-shape >= %.2f.\n",
			len(res.Units), familiesMin)
		return nil
	}
	// 0: a census lists everything it found.
	reporter.PrintFamilies(cmd.OutOrStdout(), fams, stats, res.Units, 0)
	return nil
}

// buildFamilies is the analyze command's family stage.
//
// Skipped entirely when the section is off, because the completion pass costs
// a fingerprint comparison per unscored pair inside every component and
// nothing should pay for output it disabled.
// familyMinFor is the family edge cut: the flag, unless the run calibrated
// its code-shape threshold, in which case the census moves with it.
func familyMinFor(res Result, flag float64) float64 {
	if res.Calibration != nil && res.Calibration.Applied() {
		return res.Calibration.Threshold
	}
	// Pinned runs carry an already-derived threshold and no Calibration
	// result. Following it here keeps "alike enough" one number per run
	// whichever way the threshold arrived.
	if res.Params.Pinned && res.Params.Calibrate > 0 {
		return res.Params.Threshold
	}
	return flag
}

func buildFamilies(res Result, progress io.Writer) ([]family.Family, family.Stats) {
	if familiesN <= 0 || outputFormat == formatJSON {
		// --format json marshals the snapshot, which families are deliberately
		// not part of; doppel families --format json is their machine-readable
		// home.
		return nil, family.Stats{}
	}
	o := family.DefaultOptions()
	o.Min = familyMinFor(res, familyMin)
	fams, stats := family.Build(res.Units, res.Pairs, o)
	printFamilyStats(progress, stats)
	return fams, stats
}

// printFamilyStats reports the census on stderr, next to the retrieval and
// culture summaries. The skipped-component line is not optional: a guard that
// drops work silently reads as "there was nothing there".
func printFamilyStats(w io.Writer, s family.Stats) {
	if s.Components == 0 {
		return
	}
	fmt.Fprintf(w, "Families: %d over %d components, %d functions in a family",
		s.Families, s.Components, s.Members)
	if s.Completed > 0 {
		fmt.Fprintf(w, ", %d edges completed", s.Completed)
	}
	fmt.Fprintln(w)
	if len(s.Skipped) > 0 {
		fmt.Fprintf(w, "  %d component(s) skipped as too large or too dense: sizes %v\n", len(s.Skipped), s.Skipped)
	}
}

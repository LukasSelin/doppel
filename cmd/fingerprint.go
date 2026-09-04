package cmd

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/gofront"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/spf13/cobra"
)

var (
	fpLabels    int
	fpLabel     []string
	fpTests     string
	fpGenerated string
	fpLanguages []string
	fpConfig    string
)

var fingerprintCmd = &cobra.Command{
	Use:   "fingerprint <path> <function> [<function>]",
	Short: "Show one function's fingerprint, or the merge two fingerprints score on",
	Long: `Prints what the similarity score reads for one function — its
Weisfeiler-Lehman label bag weighted by this corpus, its control-flow and
nesting histograms, its type set, and the canonicalization rules that fired
on it — or, given two functions, the full merge between their bags: every
component of the code-shape score with its blend weight, and the shared,
only-A and only-B label partitions whose masses are the Jaccard and the
containment. The totals on the page add up to the numbers the report prints.

--label takes a hash from a bag row and shows the node(s) that produced it:
the canonical subtree as Go text, and the exact extent the label hashed as
an outline truncated at the label's round. The code shown is the canonical
form — identifiers renamed, rules applied — because that is the tree the bag
was built over; the canonical tree keeps no source positions to map back to.

A function is named as package.Name, as a bare Name, or as *Receiver.Method
(the star is part of the name). An ambiguous name lists its matches and
stops. The corpus is read exactly as analyze reads it — the same population
filters, so the label weights are the ones a report would use.`,
	Example: `  doppel fingerprint . retriever.Retrieve
  doppel fingerprint . mapper.sortedKeys lexicon.sortedKeys --labels 0
  doppel fingerprint . mapper.sortedKeys lexicon.sortedKeys --label 80e9c3fe3ce5ff64`,
	Args:         cobra.RangeArgs(2, 3),
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		path := fpConfig
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
		if err := validateMode("tests", fpTests); err != nil {
			return err
		}
		return validateMode("generated", fpGenerated)
	},
	RunE: runFingerprint,
}

func init() {
	fingerprintCmd.Flags().IntVar(&fpLabels, "labels", 20, "Bag rows to print per section, heaviest first (0 = all)")
	fingerprintCmd.Flags().StringSliceVar(&fpLabel, "label", nil, "Show the subtree behind a label from a bag row (hex, with or without #; repeatable)")
	fingerprintCmd.Flags().StringSliceVar(&fpLanguages, "languages", nil, "Languages to read, comma-separated (default: every language doppel has a frontend for)")
	fingerprintCmd.Flags().StringVar(&fpTests, "tests", "exclude", "Test-function population: include, exclude, or only")
	fingerprintCmd.Flags().StringVar(&fpGenerated, "generated", "exclude", "Generated-file population: include, exclude, or only")
	fingerprintCmd.Flags().StringVar(&fpConfig, "config", "", "Path to JSON config file (default: .doppel.json if present)")
	rootCmd.AddCommand(fingerprintCmd)
}

func runFingerprint(cmd *cobra.Command, args []string) error {
	p := Params{
		Threshold: defaultThreshold,
		MinNodes:  defaultMinNodes,
		ChannelK:  5,
		TestsMode: fpTests,
		Languages: fpLanguages,
		Generated: fpGenerated,
	}
	res, err := index(args[0], p, cmd.ErrOrStderr(), nil)
	if err != nil {
		return err
	}
	if len(res.Units) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No functions found in the corpus.")
		return nil
	}

	labels, err := parseLabelFlags(fpLabel)
	if err != nil {
		return err
	}
	a, err := findUnit(res.Units, args[1])
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	meta := reporter.FingerprintMeta{CorpusFuncs: res.WL.N(), LabelTop: fpLabels}
	if len(args) == 2 {
		reporter.PrintFingerprint(out, res.Units[a], res.WL, meta)
		if len(labels) == 0 {
			return nil
		}
		src, err := labelSourceFor(res.Units[a])
		if err != nil {
			return err
		}
		fmt.Fprintln(out)
		reporter.PrintLabelOccurrences(out, res.Units[a], src, labels, res.WL, meta)
		return nil
	}
	b, err := findUnit(res.Units, args[2])
	if err != nil {
		return err
	}
	reporter.PrintFingerprintPair(out, res.Units[a], res.Units[b], res.WL, meta)
	if len(labels) == 0 {
		return nil
	}
	srcA, err := labelSourceFor(res.Units[a])
	if err != nil {
		return err
	}
	srcB, err := labelSourceFor(res.Units[b])
	if err != nil {
		return err
	}
	fmt.Fprintln(out)
	reporter.PrintLabelOccurrencesPair(out, res.Units[a], res.Units[b], srcA, srcB, labels, res.WL, meta)
	return nil
}

// parseLabelFlags reads --label values: hex, with or without the # the view
// prints, deduplicated in the order given so the output follows the command
// line.
func parseLabelFlags(in []string) ([]uint64, error) {
	var out []uint64
	for _, raw := range in {
		s := strings.TrimPrefix(strings.TrimSpace(raw), "#")
		v, err := strconv.ParseUint(s, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("--label %q: not a label hash (the view prints them as #hex)", raw)
		}
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out, nil
}

// labelSourceFor is what a label lookup on this unit reads.
//
// For Go, the frontend re-derives the canonical tree from the file and
// renders every node, and the re-derivation is checked against the bag the
// corpus holds before anything is mapped onto it: LabelCount is comparable,
// so slices.Equal compares label, count, round and kind alike, and a
// mismatch means the file changed since it was indexed or canonicalization
// is nondeterministic — the second is a bug, and either way the lookup would
// name nodes that are not in the bag. Any other language has no renderer
// and no re-parse: the unit's own canonical tree is the source, and the
// outline is the whole claim.
func labelSourceFor(u parser.CodeUnit) (reporter.LabelSource, error) {
	if u.Fingerprint.Nodes == 0 {
		return reporter.LabelSource{}, nil
	}
	if u.Lang != gofront.Lang {
		return reporter.LabelSource{Tree: u.Canonical}, nil
	}
	name := concepter.QualifiedName(u)
	src, err := os.ReadFile(u.File)
	if err != nil {
		return reporter.LabelSource{}, fmt.Errorf("--label: cannot re-derive %s: %w", name, err)
	}
	tree, renders, err := gofront.CanonicalRenders(u.File, src, u.StartLine, parser.MethodName(u))
	if err != nil {
		return reporter.LabelSource{}, fmt.Errorf("--label: cannot re-derive %s: %w", name, err)
	}
	if !slices.Equal(fingerprint.WLBagOf(tree), u.Fingerprint.WL) {
		return reporter.LabelSource{}, fmt.Errorf("--label: cannot re-derive %s from %s:%d: the re-derived canonical tree carries a different label bag than the corpus holds — the file changed since it was indexed, or canonicalization is nondeterministic, which is a bug",
			name, u.File, u.StartLine)
	}
	return reporter.LabelSource{Tree: tree, Renders: renders}, nil
}

// findUnit resolves a name to one corpus index. A qualified name
// (package.Name, or package.*Receiver.Method) must match exactly; a bare
// name matches a function of that name in any package, or a method of that
// name on any receiver — both, in one candidate set, so "Get" naming a
// function in one package and a method in another is reported as ambiguous
// rather than resolved to whichever the walk met first. Zero matches is an
// error; more than one is an error that lists them, sorted, so the caller
// can pick a qualified form.
//
// Matching by name rather than by file:line is deliberate: the name is what
// every report prints, so the way into this view is the string a reader
// already has in front of them.
func findUnit(units []parser.CodeUnit, name string) (int, error) {
	var hits []int
	for i := range units {
		if concepter.QualifiedName(units[i]) == name {
			return i, nil
		}
		if units[i].Name == name || parser.MethodName(units[i]) == name {
			hits = append(hits, i)
		}
	}
	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		return 0, fmt.Errorf("no function named %q in the corpus (names are package.Name, or *Receiver.Method with the star)", name)
	}
	names := make([]string, 0, len(hits))
	for _, i := range hits {
		names = append(names, concepter.QualifiedName(units[i]))
	}
	sort.Strings(names)
	return 0, fmt.Errorf("%q is ambiguous; use one of: %s", name, strings.Join(names, ", "))
}

package cmd

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/retriever"
	"github.com/spf13/cobra"
)

var (
	queryNear      string
	queryFile      string
	queryTop       int
	queryThreshold float64
	queryMinNodes  int
	queryChannelK  int
	queryTests     string
	queryLanguages []string
	queryExclude   []string
	queryGenerated string
	queryConfig    string
)

var queryCmd = &cobra.Command{
	Use:   "query <path>",
	Short: "Check a proposed function against a codebase's existing concepts",
	Long: `Reads a Go snippet — the function you are about to write — and reports the
corpus functions most related to it by structure, concept tags and calls,
with architecturally near code ranked above equally-similar code elsewhere.

The snippet arrives on stdin, or from --file. A bare function without a
package clause is wrapped in one; --near names that package, which is also
what connects the snippet to its intended home: bare-name calls resolve to
functions of that package, and resolved calls are what the locality weight
is built from. Include the snippet's import block — calls into imported
packages only count as evidence when the import that binds them is present.

The probe joins the corpus for the duration of the query, so the statistics
it is scored against differ from a plain analyze by exactly the probe's own
contribution. That is deliberate: the question is how this function would
sit in this corpus, not how the corpus looked without it.`,
	Example: `  doppel query --near billing . < draft.go
  doppel query --file draft.go --near billing .`,
	Args: cobra.ExactArgs(1),
	// A bad snippet is a data problem, not a CLI-usage problem; printing the
	// flag table under "could not parse the snippet" buries the message.
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		path := queryConfig
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
		if err := validateMode("tests", queryTests); err != nil {
			return err
		}
		return validateMode("generated", queryGenerated)
	},
	RunE: runQuery,
}

func init() {
	queryCmd.Flags().StringVar(&queryNear, "near", "query", "Package the proposed function will live in; bare snippets are wrapped in it and bare-name calls resolve to it")
	queryCmd.Flags().StringVar(&queryFile, "file", "", "Read the snippet from this file instead of stdin")
	queryCmd.Flags().IntVarP(&queryTop, "top", "n", 5, "Maximum related functions to report per probe")
	queryCmd.Flags().Float64VarP(&queryThreshold, "threshold", "t", defaultThreshold, "Minimum code-shape score for structural-channel candidates (0.0–1.0)")
	queryCmd.Flags().IntVar(&queryMinNodes, "min-nodes", defaultMinNodes, "Exclude functions with fewer body AST nodes from structural retrieval")
	queryCmd.Flags().IntVar(&queryChannelK, "channel-k", probeChannelK, "Candidates each function keeps per retrieval channel")
	queryCmd.Flags().StringSliceVar(&queryLanguages, "languages", nil, "Languages to read, comma-separated (default: every language doppel has a frontend for). The extension allowlist is the whole scope rule — a file is in the corpus because a frontend claims its extension, never because its contents looked like code.")
	queryCmd.Flags().StringSliceVar(&queryExclude, "exclude", nil, "Directory patterns to skip, comma-separated, on top of the built-in blocklist (node_modules, vendor, site-packages, Pods, third_party, target, dist, build, out, obj, coverage). A pattern is a glob over a directory name, or over its root-relative path when it contains a slash; a leading exclamation mark re-admits a directory the defaults would have skipped. Repeatable.")
	queryCmd.Flags().StringVar(&queryTests, "tests", "exclude", "Test-function population: include, exclude, or only")
	queryCmd.Flags().StringVar(&queryGenerated, "generated", "exclude", "Generated-file population: include, exclude, or only")
	queryCmd.Flags().StringVar(&queryConfig, "config", "", "Path to JSON config file (default: .doppel.json if present)")
	for _, name := range []string{"channel-k", "min-nodes"} {
		_ = queryCmd.Flags().MarkHidden(name)
	}
	rootCmd.AddCommand(queryCmd)
}

// packageClauseRe detects a package clause at the top of the snippet, allowing
// leading comments and blank lines. Anchored to line starts so the word
// "package" in a comment body does not count.
var packageClauseRe = regexp.MustCompile(`(?m)^[ \t]*package[ \t]+\p{L}`)

// wrapSnippet gives a bare snippet the package clause go/parser requires.
// Source that already declares a package is passed through untouched — its
// own clause wins over --near, because a snippet that says where it lives
// knows better than a default.
func wrapSnippet(src []byte, near string) []byte {
	if packageClauseRe.Match(src) {
		return src
	}
	return append([]byte("package "+near+"\n\n"), src...)
}

func runQuery(cmd *cobra.Command, args []string) error {
	var src []byte
	var err error
	if queryFile != "" {
		src, err = os.ReadFile(queryFile)
	} else {
		src, err = io.ReadAll(cmd.InOrStdin())
	}
	if err != nil {
		return fmt.Errorf("read snippet: %w", err)
	}
	if len(src) == 0 {
		return fmt.Errorf("no snippet: pipe a Go function on stdin or pass --file")
	}

	probes, err := parser.ParseSource("query.go", wrapSnippet(src, queryNear))
	if err != nil {
		return fmt.Errorf("parse snippet: %w", err)
	}
	if len(probes) == 0 {
		// ParseSource returns (nil, nil) on a syntax error, so an empty
		// result is the only signal there is. Say so, rather than reporting
		// zero matches for code that was never parsed.
		return fmt.Errorf("could not parse the snippet: no function declarations found — is it valid Go? (a bare function body is fine; a fragment of one is not)")
	}

	p := Params{
		Threshold: queryThreshold,
		MinNodes:  queryMinNodes,
		ChannelK:  queryChannelK,
		TestsMode: queryTests,
		Languages: queryLanguages,
		Exclude:   queryExclude,
		Generated: queryGenerated,
	}
	res, err := index(args[0], p, cmd.ErrOrStderr(), probes)
	if err != nil {
		return err
	}
	corpusN := len(res.Units) - len(probes)
	if corpusN <= 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No functions found in the corpus.")
		return nil
	}

	opts := probeOptions(p)
	opts.ChannelK = p.ChannelK
	prober := retriever.NewProber(res.Units, res.Graph, res.Onto, res.IC, res.WL, opts)

	for pi := range probes {
		probeIdx := corpusN + pi
		// Another probe from the same snippet is not a corpus finding.
		matches := probeMatches(res, prober, probeIdx, func(other int) bool { return other >= corpusN })
		if len(matches) > queryTop {
			matches = matches[:queryTop]
		}

		reporter.PrintQuery(cmd.OutOrStdout(), res.Units[probeIdx], res.Docs[probeIdx], matches, reporter.QueryMeta{
			CorpusFuncs:   corpusN,
			ResolvedCalls: len(res.Graph.Callees[concepter.QualifiedName(res.Units[probeIdx])]),
		})
	}
	return nil
}

// probeChannelK is the per-channel candidate budget a probe retrieves with —
// 10, not analyze's 5: a probe's retrieval costs one function's worth, so a
// wider net is nearly free, and an exact-clone family larger than K gets cut
// on an index tie-break, which is how the nearest match goes missing.
const probeChannelK = 10

// probeOptions is the retrieval configuration a probe runs under: the run's
// own shape-channel gates, so a probe is admitted exactly as a corpus function
// would be, at probeChannelK. A caller with its own budget overrides ChannelK.
func probeOptions(p Params) retriever.Options {
	opts := retriever.DefaultOptions()
	opts.ChannelK = probeChannelK
	opts.Threshold = p.Threshold
	opts.MinNodes = p.MinNodes
	return opts
}

// probeMatches retrieves the functions related to units[probeIdx] and ranks
// them by evidence boosted by locality (rankQueryMatches). The prober is built
// once per corpus by the caller, over res.Units, with probeOptions. skip drops a
// candidate by its unit index before ranking — query uses it to keep the
// snippet's other probes out, the post-edit hook to keep the edit's own file
// state out of what it reports. The probe may sit anywhere in res.Units:
// retriever.Probe is positional and indifferent to whether it was appended.
//
// Shared by `doppel query` and `doppel hook post-edit` so that the two ask the
// corpus the same question in the same way; nothing is truncated here, because
// the two callers bound their output differently.
func probeMatches(res Result, prober *retriever.Prober, probeIdx int, skip func(other int) bool) []reporter.QueryMatch {
	cands, _ := prober.Probe(probeIdx)
	matches := make([]reporter.QueryMatch, 0, len(cands))
	ball := concepter.NeighborhoodSet(res.Graph, res.Units[probeIdx])
	for _, c := range cands {
		other := retriever.OtherIdx(c, probeIdx)
		if skip != nil && skip(other) {
			continue
		}
		matches = append(matches, reporter.QueryMatch{
			Unit:      res.Units[other],
			Doc:       res.Docs[other],
			Candidate: c,
			Locality:  concepter.Locality(ball, res.Graph, res.Units[other]),
		})
	}
	rankQueryMatches(matches, probeIdx)
	return matches
}

// rankQueryMatches orders matches by retriever.ProbeLess: Total×(1+Locality),
// evidence boosted by architectural nearness and never punished by distance.
// The key ranks and is never displayed: the report shows evidence and locality
// unblended, per the house rule.
func rankQueryMatches(matches []reporter.QueryMatch, probeIdx int) {
	sort.SliceStable(matches, func(i, j int) bool {
		return retriever.ProbeLess(matches[i].Candidate, matches[j].Candidate,
			matches[i].Locality, matches[j].Locality, probeIdx)
	})
}

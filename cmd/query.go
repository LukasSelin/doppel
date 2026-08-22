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
		if err := validateTestsMode(queryTests); err != nil {
			return err
		}
		return validateGeneratedMode(queryGenerated)
	},
	RunE: runQuery,
}

func init() {
	queryCmd.Flags().StringVar(&queryNear, "near", "query", "Package the proposed function will live in; bare snippets are wrapped in it and bare-name calls resolve to it")
	queryCmd.Flags().StringVar(&queryFile, "file", "", "Read the snippet from this file instead of stdin")
	queryCmd.Flags().IntVarP(&queryTop, "top", "n", 5, "Maximum related functions to report per probe")
	queryCmd.Flags().Float64VarP(&queryThreshold, "threshold", "t", 0.60, "Minimum code-shape score for structural-channel candidates (0.0–1.0)")
	queryCmd.Flags().IntVar(&queryMinNodes, "min-nodes", 12, "Exclude functions with fewer body AST nodes from structural retrieval")
	// 10, not analyze's 5: a probe's retrieval costs one function's worth, so
	// a wider net is nearly free — and an exact-clone family larger than K gets
	// cut on an index tie-break, which is how the nearest match goes missing.
	queryCmd.Flags().IntVar(&queryChannelK, "channel-k", 10, "Candidates each function keeps per retrieval channel")
	queryCmd.Flags().StringVar(&queryTests, "tests", "exclude", "Test-function population: include, exclude, or only")
	queryCmd.Flags().StringVar(&queryGenerated, "generated", "exclude", "Generated-file population: include, exclude, or only")
	queryCmd.Flags().StringVar(&queryConfig, "config", "", "Path to JSON config file (default: .doppel.json if present)")
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

	opts := retriever.DefaultOptions()
	opts.ChannelK = p.ChannelK
	opts.Threshold = p.Threshold
	opts.MinNodes = p.MinNodes

	for pi := range probes {
		probeIdx := corpusN + pi
		cands, _ := retriever.Probe(res.Units, probeIdx, res.Graph, res.Onto, res.IC, opts)

		matches := make([]reporter.QueryMatch, 0, len(cands))
		ball := neighborhoodSet(res.Graph, res.Units[probeIdx])
		for _, c := range cands {
			other := c.AIdx
			if other == probeIdx {
				other = c.BIdx
			}
			// Another probe from the same snippet is not a corpus finding.
			if other >= corpusN {
				continue
			}
			matches = append(matches, reporter.QueryMatch{
				Unit:      res.Units[other],
				Doc:       res.Docs[other],
				Candidate: c,
				Locality:  locality(ball, res.Graph, res.Units[other]),
			})
		}
		rankQueryMatches(matches)
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

// neighborhoodSet is the probe's depth-2 call-graph ball as a set.
func neighborhoodSet(g *concepter.Graph, u parser.CodeUnit) map[string]bool {
	ball := g.Neighborhood(concepter.QualifiedName(u))
	if len(ball) == 0 {
		return nil
	}
	set := make(map[string]bool, len(ball))
	for _, n := range ball {
		set[n] = true
	}
	return set
}

// locality measures how much of the probe's architectural neighborhood the
// candidate inhabits: |ball(probe) ∩ (ball(c) ∪ {c})| / |ball(probe)|.
//
// The candidate's own identity joins its ball because a function the probe
// directly calls is maximally near and must not score below its own
// neighbors. An empty probe ball — a snippet that resolved no calls — makes
// every locality 0: honestly neutral, and the report says so rather than
// letting a column of zeros imply the corpus is far away.
func locality(probeBall map[string]bool, g *concepter.Graph, c parser.CodeUnit) float64 {
	if len(probeBall) == 0 {
		return 0
	}
	qn := concepter.QualifiedName(c)
	hits := 0
	if probeBall[qn] {
		hits++
	}
	for _, n := range g.Neighborhood(qn) {
		if n != qn && probeBall[n] {
			hits++
		}
	}
	// hits can exceed the intersection only if qn were in its own ball, which
	// Neighborhood excludes by construction.
	if hits > len(probeBall) {
		hits = len(probeBall)
	}
	return float64(hits) / float64(len(probeBall))
}

// rankQueryMatches orders matches by Total×(1+Locality) — evidence boosted by
// architectural nearness, never punished by distance. The key ranks and is
// never displayed: the report shows evidence and locality unblended, per the
// house rule. Ties fall back to Total desc then corpus index asc, a total
// order, so a fixed corpus and snippet always print the same report.
func rankQueryMatches(matches []reporter.QueryMatch) {
	sort.SliceStable(matches, func(i, j int) bool {
		ki := matches[i].Candidate.Total * (1 + matches[i].Locality)
		kj := matches[j].Candidate.Total * (1 + matches[j].Locality)
		if ki != kj {
			return ki > kj
		}
		if matches[i].Candidate.Total != matches[j].Candidate.Total {
			return matches[i].Candidate.Total > matches[j].Candidate.Total
		}
		// An exact-clone family ties on evidence mass; code-shape separates
		// the byte-identical twin from the merely similar sibling.
		if matches[i].Candidate.Breakdown.Score != matches[j].Candidate.Breakdown.Score {
			return matches[i].Candidate.Breakdown.Score > matches[j].Candidate.Breakdown.Score
		}
		oi, oj := otherIdx(matches[i].Candidate), otherIdx(matches[j].Candidate)
		return oi < oj
	})
}

// otherIdx is the corpus-side index of a probe candidate. The probe is always
// the appended unit, so the smaller index is the corpus function.
func otherIdx(c retriever.Candidate) int {
	if c.AIdx < c.BIdx {
		return c.AIdx
	}
	return c.BIdx
}

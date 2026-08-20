package reporter

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/lukse/doppel/internal/analyzer"
	"github.com/lukse/doppel/internal/fingerprint"
	"github.com/lukse/doppel/internal/parser"
)

// Meta carries the run-level context a report renders alongside the pairs.
type Meta struct {
	Threshold  float64 // the structural-channel floor, echoed in the header
	TotalFuncs int
	Debug      bool // when set, show per-pair retrieval provenance
}

// Print writes the similarity report to w. The headline number per pair is
// the code-shape score — how alike the two normalized bodies are — which is
// deliberately not named "score": pairs are ranked by retrieval evidence, and
// a 1.0 shape match on a trivial idiom is not a 1.0 verdict.
func Print(w io.Writer, pairs []analyzer.SimilarPair, meta Meta) {
	fmt.Fprintf(w, "\nCode Similarity Report\n")
	fmt.Fprintf(w, "======================\n")
	fmt.Fprintf(w, "Functions analyzed: %d  |  Threshold: %.2f\n\n", meta.TotalFuncs, meta.Threshold)

	if len(pairs) == 0 {
		fmt.Fprintf(w, "No similar function pairs found\n")
		return
	}

	for i, p := range pairs {
		fmt.Fprintf(w, "#%-3d  code-shape: %.4f\n", i+1, p.Score)
		printUnit(w, "  A", p.A)
		printUnit(w, "  B", p.B)
		fmt.Fprintf(w, "  %s\n", breakdownLine(p.Breakdown))
		if p.Retrieval != nil {
			fmt.Fprintf(w, "  evidence: %.2f  (shape %.2f  concept %.2f  call %.2f)\n",
				p.Retrieval.Total, p.Retrieval.Shape, p.Retrieval.Concept, p.Retrieval.Call)
			if meta.Debug {
				fmt.Fprintf(w, "  retrieved-via: %s\n", strings.Join(p.Retrieval.Channels, "+"))
			}
		}
		if p.Evidence != nil {
			fmt.Fprintf(w, "  structural overlap: %.2f", p.Evidence.OverlapScore)
			if p.Evidence.MergeWorthy {
				fmt.Fprintf(w, " (merge-worthy)")
			}
			fmt.Fprintln(w)
			for _, reason := range p.Evidence.Reasons {
				fmt.Fprintf(w, "    • %s\n", reason)
			}
		}
		fmt.Fprintln(w)
	}
}

// PrintMarkdown writes the similarity report as a Markdown document to w.
func PrintMarkdown(w io.Writer, pairs []analyzer.SimilarPair, meta Meta) {
	fmt.Fprintf(w, "# Code Similarity Report\n\n")
	fmt.Fprintf(w, "**Functions analyzed:** %d | **Threshold:** %.2f | **Pairs found:** %d\n\n", meta.TotalFuncs, meta.Threshold, len(pairs))
	fmt.Fprintf(w, "---\n\n")

	if len(pairs) == 0 {
		fmt.Fprintf(w, "_No similar function pairs found._\n")
		return
	}

	for i, p := range pairs {
		fmt.Fprintf(w, "## Match #%d — Code-shape: `%.4f`\n\n", i+1, p.Score)

		// Table header
		fmt.Fprintf(w, "| | Location | Function | Signature | Patterns |\n")
		fmt.Fprintf(w, "|---|---|---|---|---|\n")
		mdTableRow(w, "A", p.A)
		mdTableRow(w, "B", p.B)
		fmt.Fprintln(w)

		fmt.Fprintf(w, "**Code similarity:** `%s`\n\n", breakdownLine(p.Breakdown))

		if p.Retrieval != nil {
			fmt.Fprintf(w, "**Evidence:** `%.2f` (shape %.2f, concept %.2f, call %.2f)\n\n",
				p.Retrieval.Total, p.Retrieval.Shape, p.Retrieval.Concept, p.Retrieval.Call)
			if meta.Debug {
				fmt.Fprintf(w, "**Retrieved via:** %s\n\n", strings.Join(p.Retrieval.Channels, ", "))
			}
		}

		if p.Evidence != nil {
			label := "not merge-worthy"
			if p.Evidence.MergeWorthy {
				label = "merge-worthy"
			}
			fmt.Fprintf(w, "**Structural overlap:** `%.2f` (%s)\n\n", p.Evidence.OverlapScore, label)
			if len(p.Evidence.Reasons) > 0 {
				for _, reason := range p.Evidence.Reasons {
					// Escaped like the table cells. Reasons quote identifiers from
					// the analysed source, and now bracketed lists of them too.
					fmt.Fprintf(w, "- %s\n", mdEscape(reason))
				}
				fmt.Fprintln(w)
			}
		}

		fmt.Fprintf(w, "---\n\n")
	}
}

// breakdownLine renders the component scores behind a pair score, so a match
// can be inspected without re-reading both function bodies.
func breakdownLine(b fingerprint.Breakdown) string {
	return fmt.Sprintf("ast %.2f  flow %.2f  sig %.2f  size %.2f", b.AST, b.Flow, b.Signature, b.SizeRatio)
}

func mdTableRow(w io.Writer, label string, u parser.CodeUnit) {
	loc := fmt.Sprintf("`%s:%d`", filepath.ToSlash(u.File), u.StartLine)
	name := u.Name
	if u.Package != "" {
		name = u.Package + "." + name
	}
	sig := u.Signature
	if sig == "" {
		sig = "—"
	}
	patterns := "—"
	if len(u.Patterns) > 0 {
		patterns = strings.Join(u.Patterns, ", ")
	}
	fmt.Fprintf(w, "| **%s** | %s | `%s` | `%s` | %s |\n", label, loc, mdEscape(name), mdEscape(sig), patterns)
}

// mdEscape escapes pipe characters that would break markdown tables.
func mdEscape(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

func printUnit(w io.Writer, prefix string, u parser.CodeUnit) {
	loc := fmt.Sprintf("%s:%d", filepath.ToSlash(u.File), u.StartLine)
	name := u.Name
	if u.Package != "" {
		name = u.Package + "." + name
	}
	fmt.Fprintf(w, "%s  %-60s  %s\n", prefix, loc, name)
	if u.Signature != "" {
		fmt.Fprintf(w, "       sig: %s\n", u.Signature)
	}
	if len(u.Patterns) > 0 {
		fmt.Fprintf(w, "      tags: %s\n", strings.Join(u.Patterns, ", "))
	}
}

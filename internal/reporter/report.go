package reporter

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/lukse/doppel/internal/analyzer"
	"github.com/lukse/doppel/internal/parser"
)

// Print writes the similarity report to w.
func Print(w io.Writer, pairs []analyzer.SimilarPair, threshold float64, totalFuncs int) {
	fmt.Fprintf(w, "\nCode Similarity Report\n")
	fmt.Fprintf(w, "======================\n")
	fmt.Fprintf(w, "Functions analyzed: %d  |  Threshold: %.2f\n\n", totalFuncs, threshold)

	if len(pairs) == 0 {
		fmt.Fprintf(w, "No similar function pairs found above threshold %.2f\n", threshold)
		return
	}

	for i, p := range pairs {
		fmt.Fprintf(w, "#%-3d  score: %.4f\n", i+1, p.Score)
		printUnit(w, "  A", p.A)
		printUnit(w, "  B", p.B)
		if p.Evidence != nil {
			fmt.Fprintf(w, "  structural overlap: %.2f", p.Evidence.OverlapScore)
			if p.Evidence.MergeWorthy {
				fmt.Fprintf(w, " (merge-worthy)")
			}
			fmt.Fprintln(w)
			for _, dim := range p.Evidence.Dimensions {
				if dim.Evidence == "" {
					continue
				}
				fmt.Fprintf(w, "    • [sim=%.2f contra=%.2f] %s\n", dim.Similarity, dim.Contradiction, dim.Evidence)
			}
		}
		if p.Explanation != "" {
			fmt.Fprintf(w, "  → %s\n", p.Explanation)
		}
		fmt.Fprintln(w)
	}
}

// PrintMarkdown writes the similarity report as a Markdown document to w.
func PrintMarkdown(w io.Writer, pairs []analyzer.SimilarPair, threshold float64, totalFuncs int) {
	fmt.Fprintf(w, "# Code Similarity Report\n\n")
	fmt.Fprintf(w, "**Functions analyzed:** %d | **Threshold:** %.2f | **Pairs found:** %d\n\n", totalFuncs, threshold, len(pairs))
	fmt.Fprintf(w, "---\n\n")

	if len(pairs) == 0 {
		fmt.Fprintf(w, "_No similar function pairs found above threshold %.2f._\n", threshold)
		return
	}

	for i, p := range pairs {
		fmt.Fprintf(w, "## Match #%d — Score: `%.4f`\n\n", i+1, p.Score)

		// Table header
		fmt.Fprintf(w, "| | Location | Function | Signature | Patterns |\n")
		fmt.Fprintf(w, "|---|---|---|---|---|\n")
		mdTableRow(w, "A", p.A)
		mdTableRow(w, "B", p.B)
		fmt.Fprintln(w)

		if p.Evidence != nil {
			label := "not merge-worthy"
			if p.Evidence.MergeWorthy {
				label = "merge-worthy"
			}
			fmt.Fprintf(w, "**Structural overlap:** `%.2f` (%s)\n\n", p.Evidence.OverlapScore, label)
			fmt.Fprintf(w, "| Dimension | Sim | Contra | Evidence |\n")
			fmt.Fprintf(w, "|---|---|---|---|\n")
			for _, dim := range p.Evidence.Dimensions {
				if dim.Evidence == "" {
					continue
				}
				fmt.Fprintf(w, "| %s | %.2f | %.2f | %s |\n", dim.Name, dim.Similarity, dim.Contradiction, mdEscape(dim.Evidence))
			}
			fmt.Fprintln(w)
		}

		if p.Explanation != "" {
			// Wrap explanation in a blockquote; handle multi-line responses
			for _, line := range strings.Split(strings.TrimSpace(p.Explanation), "\n") {
				fmt.Fprintf(w, "> %s\n", line)
			}
			fmt.Fprintln(w)
		}

		fmt.Fprintf(w, "---\n\n")
	}
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

package identity

import (
	"fmt"
	"io"
	"strings"
)

// deltaTitle heads every rendering of a Delta, on all three surfaces.
//
// "baseline" is the older of the two runs, whether it arrived as a session
// hook's recorded origin or as the first file argument to `doppel diff`. The
// word is the same one the hook contract uses, and the two surfaces are
// answering the same question.
const deltaTitle = "Delta since the baseline"

// PrintDelta writes the delta report as text: the identity classification,
// then the pairs those changes created or dissolved.
//
// The classification comes first because it is the attribution the pair half
// is read through — a created pair is interesting because of *which* function
// arrived, and the reader needs the class list to know that. Print renders the
// classification exactly as it always has, so the two surfaces cannot drift.
func PrintDelta(w io.Writer, d Delta, listUnchanged bool) {
	fmt.Fprintf(w, "%s\n%s\n", deltaTitle, strings.Repeat("=", len(deltaTitle)))
	Print(w, d.Result, listUnchanged)
	if !d.Comparable {
		return
	}

	fmt.Fprintf(w, "\npairs created %d, dissolved %d\n", len(d.Created), len(d.Dissolved))
	printPairSection(w, "created", d.Created)
	printPairSection(w, "dissolved", d.Dissolved)
}

// printPairSection prints one pair list, or nothing when it is empty — the
// same rule Print applies to its class sections, and the reason the census line
// above carries both counts including the zeros.
func printPairSection(w io.Writer, title string, ps []PairChange) {
	if len(ps) == 0 {
		return
	}
	fmt.Fprintf(w, "\npairs %s %d\n", title, len(ps))
	for _, p := range ps {
		for _, line := range PairLines(p) {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}
}

// PairLines renders one pair change as a headline plus its indented evidence:
// what classified change explains it, and the stored explanation of why the two
// bodies are alike.
//
// Exported for the same reason Lines is — the command, the hook digest and the
// tests must read one rendering, not three spellings of one.
func PairLines(p PairChange) []string {
	out := []string{fmt.Sprintf("%s <-> %s  shape %.2f  overlap %.2f%s",
		p.A, p.B, p.Score, p.Overlap, mergeSuffix(p.MergeWorthy))}
	out = append(out, "    "+CauseLine(p))
	if p.Explain != "" {
		out = append(out, "    explain: "+p.Explain)
	}
	return out
}

// CauseLine states what this pair change is attributed to, or that it is
// attributed to nothing.
//
// The unattributed wording matters: the reader must never be left to assume
// their edit caused a pair to appear when retrieval's bounded top-K moved it.
// It is the same claim snapshot's attributionTag makes, said in the classes
// this package can name.
func CauseLine(p PairChange) string {
	causes := p.Causes()
	if len(causes) == 0 {
		return "no classified change on either side (retrieval re-ranking)"
	}
	parts := make([]string, 0, len(causes))
	for _, c := range causes {
		parts = append(parts, fmt.Sprintf("%s %s", c.Key, c.Class))
	}
	return strings.Join(parts, ", ")
}

func mergeSuffix(worthy bool) string {
	if worthy {
		return "  (merge-worthy)"
	}
	return ""
}

// MarkdownDelta writes the same report as markdown, for `doppel diff --output`.
//
// It is the whole document rather than a section of one, which is why the
// report is trivially first in it: there is no two-run dashboard and no other
// two-run markdown to lead. A one-run report cannot carry this section at all —
// `analyze` never reads a baseline, so a plain run has nothing to compare
// against.
func MarkdownDelta(w io.Writer, d Delta, listUnchanged bool) {
	fmt.Fprintf(w, "# %s\n\n", deltaTitle)
	if !d.Comparable {
		fmt.Fprintf(w, "The two runs are not comparable: %s\n", d.Reason)
		return
	}

	fmt.Fprintf(w, "%d functions before, %d after.\n\n", d.OldFunctions, d.NewFunctions)
	for _, n := range d.Notes {
		fmt.Fprintf(w, "> Note: %s\n\n", n)
	}

	fmt.Fprintf(w, "| class | count |\n| --- | ---: |\n")
	for _, cc := range d.Counts {
		fmt.Fprintf(w, "| %s | %d |\n", cc.Class, cc.Count)
	}
	fmt.Fprintf(w, "| pairs created | %d |\n| pairs dissolved | %d |\n\n", len(d.Created), len(d.Dissolved))

	for _, class := range classOrder {
		n := d.Count(class)
		if n == 0 {
			continue
		}
		fmt.Fprintf(w, "## %s (%d)\n\n", class, n)
		if class == Unchanged && !listUnchanged {
			fmt.Fprintf(w, "Not listed. Pass `--unchanged` to see them.\n\n")
			continue
		}
		for _, c := range d.Changes {
			if c.Class != class {
				continue
			}
			markdownBullets(w, Lines(c))
		}
		fmt.Fprintln(w)
	}

	markdownPairSection(w, "Pairs created", d.Created)
	markdownPairSection(w, "Pairs dissolved", d.Dissolved)
}

func markdownPairSection(w io.Writer, title string, ps []PairChange) {
	if len(ps) == 0 {
		return
	}
	fmt.Fprintf(w, "## %s (%d)\n\n", title, len(ps))
	for _, p := range ps {
		markdownBullets(w, PairLines(p))
	}
	fmt.Fprintln(w)
}

// markdownBullets renders one finding's text lines as a bullet and its
// sub-bullets. The text renderer's own indentation is what decides the nesting,
// so the two forms carry exactly the same lines and neither can gain a fact the
// other lacks.
func markdownBullets(w io.Writer, lines []string) {
	for i, l := range lines {
		trimmed := strings.TrimLeft(l, " ")
		if i == 0 {
			fmt.Fprintf(w, "- %s\n", mdEscapePipes(trimmed))
			continue
		}
		fmt.Fprintf(w, "  - %s\n", mdEscapePipes(trimmed))
	}
}

// mdEscapePipes keeps a rendered line from breaking a table cell if one of
// these bullets is ever moved into one. Nothing else in these lines is
// markdown-significant: keys, classes and floats.
func mdEscapePipes(s string) string { return strings.ReplaceAll(s, "|", `\|`) }

package reporter

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

// Meta carries the run-level context a report renders alongside the pairs.
type Meta struct {
	Threshold  float64 // the structural-channel floor, echoed in the header
	TotalFuncs int
	Debug      bool // when set, show per-pair retrieval provenance

	// Overview is what doppel understands about the corpus, rendered above the
	// matches in the markdown report only. Nil — which is what the text report
	// always passes — renders nothing, so a report without one is byte-identical
	// to one written before this field existed.
	Overview *Overview
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
		if p.Kind != nil {
			fmt.Fprintf(w, "  kind: %s\n", kindClause(p.Kind, false, false))
		}
		// Above the numbers rather than below them: the explain line says what
		// the canonicalizer did to these two bodies, which is the premise every
		// score under it is computed on.
		if p.Explain != "" {
			fmt.Fprintf(w, "  explain: %s\n", p.Explain)
		}
		for _, note := range p.Profile {
			fmt.Fprintf(w, "  profile %s: %s (%s)\n", note.Side, profileMassLine(note.Concepts, "  "), note.State)
			if meta.Debug {
				fmt.Fprintf(w, "    arena %s: %s\n", note.Side, arenaDebugLine(note))
			}
		}
		fmt.Fprintf(w, "  %s\n", breakdownLine(p.Breakdown))
		// Containment is a second reported quantity, not a component and not
		// a verdict — see containmentClause. It gets its own line for the
		// same reason trophic does: it explains the code-shape number above
		// rather than contributing to it.
		fmt.Fprintf(w, "  containment: %.2f%s\n",
			p.Breakdown.Containment, containmentClause(p.Breakdown))
		if p.Retrieval != nil {
			fmt.Fprintf(w, "  evidence: %.2f  (shape %.2f  concept %.2f  call %.2f)\n",
				p.Retrieval.Total, p.Retrieval.Shape, p.Retrieval.Concept, p.Retrieval.Call)
			fmt.Fprintf(w, "  trophic: %.2f\n", p.Retrieval.TrophicSim)
			if len(p.Retrieval.Chains) > 0 {
				fmt.Fprintf(w, "  shared structure:\n")
				for _, ch := range p.Retrieval.Chains {
					fmt.Fprintf(w, "    %.2f  %s%s\n", ch.Energy, ch.Render, chainTimes(ch))
				}
			}
			if meta.Debug {
				fmt.Fprintf(w, "  retrieved-via: %s\n", strings.Join(p.Retrieval.Channels, "+"))
			}
		}
		for _, note := range p.Culture {
			fmt.Fprintf(w, "  culture: %s realizes %s atypically (typicality %.2f, concept median %.2f, convention %.2f)\n",
				note.Side, note.Tag, note.Typicality, note.ConceptMedian, note.Convention)
			if meta.Debug {
				fmt.Fprintf(w, "    channels: %s\n", cultureChannelLine(note.Channels))
			}
		}
		for _, note := range p.Habitat {
			fmt.Fprintf(w, "  habitat: %s fits poorly in %s (fit %.2f, package norm %.2f%s)\n",
				note.Side, note.Package, note.Fit, note.PackageNorm, subsystemClause(note, false))
			if meta.Debug {
				fmt.Fprintf(w, "    surprise: %s\n", habitatChannelLine(note.Channels))
			}
		}
		if p.Evidence != nil {
			fmt.Fprintf(w, "  structural overlap: %.2f", p.Evidence.OverlapScore)
			if p.MergeWorthy() {
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

	// The corpus before the findings: a reader weighing a list of pairs needs to
	// know what kind of codebase produced it.
	PrintMarkdownOverview(w, meta.Overview)
	PrintMarkdownPractice(w, meta.Overview)

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

		if p.Kind != nil {
			fmt.Fprintf(w, "**Kind:** %s\n\n", kindClause(p.Kind, false, true))
		}
		if p.Explain != "" {
			// Escaped like every other sentence carrying names from the
			// analysed source: a residual names node kinds, but a rule name
			// or a future kind could carry a pipe.
			fmt.Fprintf(w, "**Explain:** %s\n\n", mdEscape(p.Explain))
		}
		for _, note := range p.Profile {
			fmt.Fprintf(w, "**Profile %s:** %s (%s)\n\n", note.Side, mdProfileMassLine(note.Concepts), note.State)
			if meta.Debug {
				fmt.Fprintf(w, "**Arena %s:** %s\n\n", note.Side, mdArenaDebugLine(note))
			}
		}

		fmt.Fprintf(w, "**Code similarity:** `%s`\n\n", breakdownLine(p.Breakdown))
		fmt.Fprintf(w, "**Containment:** `%.2f`%s\n\n",
			p.Breakdown.Containment, containmentClause(p.Breakdown))

		if p.Retrieval != nil {
			fmt.Fprintf(w, "**Evidence:** `%.2f` (shape %.2f, concept %.2f, call %.2f)\n\n",
				p.Retrieval.Total, p.Retrieval.Shape, p.Retrieval.Concept, p.Retrieval.Call)
			fmt.Fprintf(w, "**Trophic:** `%.2f`\n\n", p.Retrieval.TrophicSim)
			if len(p.Retrieval.Chains) > 0 {
				fmt.Fprintf(w, "**Shared structure:**\n\n")
				for _, ch := range p.Retrieval.Chains {
					fmt.Fprintf(w, "- `%.2f` — `%s`%s\n",
						ch.Energy, mdEscape(ch.Render), mdEscape(chainTimes(ch)))
				}
				fmt.Fprintln(w)
			}
			if meta.Debug {
				fmt.Fprintf(w, "**Retrieved via:** %s\n\n", strings.Join(p.Retrieval.Channels, ", "))
			}
		}

		for _, note := range p.Culture {
			fmt.Fprintf(w, "**Culture:** %s realizes `%s` atypically (typicality %.2f, concept median %.2f, convention %.2f)\n\n",
				note.Side, mdEscape(note.Tag), note.Typicality, note.ConceptMedian, note.Convention)
			if meta.Debug {
				fmt.Fprintf(w, "**Channels (%s/%s):** %s\n\n",
					note.Side, mdEscape(note.Tag), cultureChannelLine(note.Channels))
			}
		}

		for _, note := range p.Habitat {
			fmt.Fprintf(w, "**Habitat:** %s fits poorly in `%s` (fit %.2f, package norm %.2f%s)\n\n",
				note.Side, mdEscape(note.Package), note.Fit, note.PackageNorm, subsystemClause(note, true))
			if meta.Debug {
				fmt.Fprintf(w, "**Surprise (%s/%s):** %s\n\n",
					note.Side, mdEscape(note.Package), habitatChannelLine(note.Channels))
			}
		}

		if p.Evidence != nil {
			label := "not merge-worthy"
			if p.MergeWorthy() {
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

// subsystemClause extends a habitat note with the subsystem the unit was
// also alien in, when one was modeled: "; subsystem tpl/ fit 0.30".
func subsystemClause(note analyzer.HabitatNote, md bool) string {
	if note.Subsystem == "" {
		return ""
	}
	key := note.Subsystem
	if md {
		key = "`" + mdEscape(key) + "`"
	}
	return fmt.Sprintf("; subsystem %s fit %.2f", key, note.SubsystemFit)
}

// cultureChannelLine renders a note's per-channel typicalities in their
// fixed channel order.
func cultureChannelLine(channels []analyzer.CultureChannel) string {
	parts := make([]string, 0, len(channels))
	for _, ch := range channels {
		parts = append(parts, fmt.Sprintf("%s %.2f", ch.Name, ch.Typicality))
	}
	return strings.Join(parts, "  ")
}

// profileMassLine renders survivor masses as "tag 0.39" entries.
func profileMassLine(concepts []analyzer.ProfileMass, sep string) string {
	parts := make([]string, 0, len(concepts))
	for _, c := range concepts {
		parts = append(parts, fmt.Sprintf("%s %.2f", c.Tag, c.Mass))
	}
	return strings.Join(parts, sep)
}

func mdProfileMassLine(concepts []analyzer.ProfileMass) string {
	parts := make([]string, 0, len(concepts))
	for _, c := range concepts {
		parts = append(parts, fmt.Sprintf("`%s` %.2f", mdEscape(c.Tag), c.Mass))
	}
	return strings.Join(parts, ", ")
}

// arenaDebugLine renders the dynamics detail: rounds, convergence verb, and
// the extinct candidates at %.4f so near-extinction stays visible.
func arenaDebugLine(note analyzer.ProfileNote) string {
	verb := "capped"
	if note.Converged {
		verb = "converged"
	}
	out := fmt.Sprintf("%d rounds, %s", note.Rounds, verb)
	if len(note.Extinct) > 0 {
		parts := make([]string, 0, len(note.Extinct))
		for _, c := range note.Extinct {
			parts = append(parts, fmt.Sprintf("%s %.4f", c.Tag, c.Mass))
		}
		out += "; extinct: " + strings.Join(parts, "  ")
	}
	return out
}

func mdArenaDebugLine(note analyzer.ProfileNote) string {
	verb := "capped"
	if note.Converged {
		verb = "converged"
	}
	out := fmt.Sprintf("%d rounds, %s", note.Rounds, verb)
	if len(note.Extinct) > 0 {
		parts := make([]string, 0, len(note.Extinct))
		for _, c := range note.Extinct {
			parts = append(parts, fmt.Sprintf("`%s` %.4f", mdEscape(c.Tag), c.Mass))
		}
		out += "; extinct: " + strings.Join(parts, "  ")
	}
	return out
}

// habitatChannelLine renders a habitat note's per-channel surprises in their
// fixed channel order.
func habitatChannelLine(channels []analyzer.HabitatChannel) string {
	parts := make([]string, 0, len(channels))
	for _, ch := range channels {
		parts = append(parts, fmt.Sprintf("%s %.2f", ch.Name, ch.Surprise))
	}
	return strings.Join(parts, "  ")
}

// containmentGap is how far containment must exceed the WL Jaccard before the
// report says so in words. Containment always sits at or above the Jaccard —
// same numerator, a smaller denominator — so the bare comparison is never
// news; the size of the gap is.
// chainTimes suffixes a shared-structure line with the multiplicity when the
// label matched more than once — " ×3". The energy on the line is
// idf · min(count), so the count is the only factor of it a reader cannot
// otherwise see, and printing "×1" on the common case would be noise.
func chainTimes(ch analyzer.SharedChain) string {
	if ch.Count <= 1 {
		return ""
	}
	return " ×" + strconv.Itoa(ch.Count)
}

const containmentGap = 0.25

// containmentClause names the one reading of containment a pair list would
// otherwise bury. When containment is high and the Jaccard is well below it,
// the two bodies are not two versions of one function: most of the smaller
// one's structure is present in the larger one, which has a lot more besides.
// That is the shape of an inlined helper, and it is invisible in a
// code-shape number that divides by the union.
//
// The clause restates arithmetic already printed on the same pair and adds no
// judgement of its own — both numbers are above it, so a reader can check the
// subtraction. It never affects ranking, filtering or the merge verdict.
func containmentClause(b fingerprint.Breakdown) string {
	if b.Containment >= 0.60 && b.Containment-b.WL >= containmentGap {
		return " — most of the smaller body's shape is inside the larger"
	}
	return ""
}

// breakdownLine renders the component scores behind a pair score, so a match
// can be inspected without re-reading both function bodies.
//
// The first component reads `wl` rather than the `ast` it read before the
// 0.60 slot changed metric. It is a corpus-weighted multiset Jaccard over
// Weisfeiler-Lehman label bags now, not Jaccard over token 3-grams, and a
// reader comparing an old report to a new one should see that the number
// answers a different question rather than assume the code moved.
func breakdownLine(b fingerprint.Breakdown) string {
	return fmt.Sprintf("wl %.2f  flow %.2f  nesting %.2f  sig %.2f  size %.2f",
		b.WL, b.Flow, b.Depth, b.Signature, b.SizeRatio)
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

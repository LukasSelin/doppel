package reporter

import (
	"fmt"
	"io"
	"strings"
)

// Bounds for the practice section. Every list here is long-tailed — a wide
// corpus has hundreds of associations and thousands of atypical members — and
// past a screenful the section stops being an insight and becomes a dump.
const (
	maxPractice = 6  // concepts described
	maxDrift    = 10 // functions named as drifting
	barWidth    = 10 // cells in a proportion bar
)

// PrintMarkdownPractice writes how this codebase writes things, as opposed to
// what it contains.
//
// Everything here comes from two models that had no caller outside their own
// tests: the prototypes, which are what a concept looks like when this corpus
// writes one, and the PMI ecology, which is what tends to travel with what. The
// argument that kept them out of the report was that an association annotates
// the corpus and not a pair — which is exactly right, and is why they belong in
// a corpus-level section rather than hanging off a match.
//
// The register is the same as everywhere else in the report: state the fact,
// give the number behind it, ask for nothing.
func PrintMarkdownPractice(w io.Writer, ov *Overview) {
	if ov == nil {
		return
	}
	if len(ov.Practice) == 0 && len(ov.Travels) == 0 && len(ov.Avoids) == 0 &&
		len(ov.Drift) == 0 && ov.Matrix == nil {
		return
	}
	fmt.Fprintf(w, "## Local practice\n\n")
	fmt.Fprintf(w, "The vocabulary above says what a concept *is*. This says what one looks like when "+
		"**this** codebase writes it — learned from the corpus, so it describes the house style rather "+
		"than a rule from anywhere else.\n\n")

	practiceConcepts(w, ov)
	practiceMatrix(w, ov)
	practiceTravels(w, ov)
	practiceDrift(w, ov)

	fmt.Fprintf(w, "---\n\n")
}

// practiceConcepts renders each prototype as house style.
//
// Counts, not percentages. A concept qualifies for a prototype at five members,
// and "67%" of six is a number with more digits than evidence; "4 of 6" says
// exactly what was counted. The bar carries the proportion, which is what
// percentages were doing badly.
func practiceConcepts(w io.Writer, ov *Overview) {
	if len(ov.Practice) == 0 {
		return
	}
	fmt.Fprintf(w, "### How this codebase writes each concept\n\n")
	fmt.Fprintf(w, "Of the functions carrying each tag, how many do each thing. Weights are how much "+
		"a channel counts toward whether a member looks normal — calls 40, control flow 20, "+
		"co-occurring tags 15, role 15, package 10.\n\n")

	shown := ov.Practice
	if len(shown) > maxPractice {
		shown = shown[:maxPractice]
	}
	for _, p := range shown {
		fmt.Fprintf(w, "**`%s`** — %s\n\n", mdEscape(p.Tag), plural(p.Members, "function"))
		fmt.Fprintf(w, "| Channel | Feature | | Members |\n|---|---|---|---|\n")
		for _, ch := range p.Channels {
			for i, f := range ch.Features {
				channel := ""
				if i == 0 {
					channel = fmt.Sprintf("%s ×%d", ch.Name, ch.Weight)
				}
				fmt.Fprintf(w, "| %s | `%s` | %s | %d of %d |\n",
					channel, mdEscape(f.Name), bar(f.P), f.Count, p.Members)
			}
		}
		fmt.Fprintln(w)
	}
	if more := len(ov.Practice) - len(shown); more > 0 {
		fmt.Fprintf(w, "_%d further concepts are modeled and not described._\n\n", more)
	}
}

// bar renders a proportion as blocks. A number tells you 4 of 6; the bar lets
// you compare six rows without reading any of them.
func bar(p float64) string {
	n := int(p*float64(barWidth) + 0.5)
	if n > barWidth {
		n = barWidth
	}
	if n < 0 {
		n = 0
	}
	return "`" + strings.Repeat("█", n) + strings.Repeat("·", barWidth-n) + "`"
}

// practiceMatrix renders the whole concept-to-concept co-occurrence structure.
//
// This is the one table in the report that is not a sample. The vocabulary is a
// fixed, small set of concrete concepts, so the grid is bounded by construction
// and can show every cell — including the empty ones, which is the point. A `never` cell
// says two concepts this corpus uses are never written by the same function,
// and that is a statement about layering no ranked list would surface.
func practiceMatrix(w io.Writer, ov *Overview) {
	m := ov.Matrix
	if m == nil || len(m.Tags) < 2 {
		return
	}
	// A grid where every cell is blank is not a finding, it is a shrug. On a
	// corpus whose concepts never travel together beyond chance, silence is
	// the honest output — the same rule every other section here follows.
	any := false
	for i := range m.Cells {
		for j := 0; j < i && !any; j++ {
			any = m.Cells[i][j] != ""
		}
	}
	if !any {
		return
	}
	fmt.Fprintf(w, "### Which concepts share a function\n\n")
	fmt.Fprintf(w, "`++` at least four times chance, `+` at least twice, `−` at most half, "+
		"`never` not once. A blank cell is ordinary company — near chance, which is not culture.\n\n")

	fmt.Fprintf(w, "| |")
	for _, t := range m.Tags[:len(m.Tags)-1] {
		fmt.Fprintf(w, " `%s` |", mdEscape(t))
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "|---|")
	for range m.Tags[:len(m.Tags)-1] {
		fmt.Fprintf(w, "---|")
	}
	fmt.Fprintln(w)

	// Lower triangle: co-occurrence is symmetric, so the upper half would
	// repeat every cell and the diagonal says a concept co-occurs with itself.
	for i := 1; i < len(m.Tags); i++ {
		fmt.Fprintf(w, "| **`%s`** |", mdEscape(m.Tags[i]))
		for j := 0; j < len(m.Tags)-1; j++ {
			if j >= i {
				fmt.Fprintf(w, " |")
				continue
			}
			fmt.Fprintf(w, " %s |", m.Cells[i][j])
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)
}

// practiceTravels renders the corpus's co-occurrence habits, one list per kind.
//
// Each line leads with the conditional rather than the lift: "13 of 14
// http_call functions call NewRequest" is what a reader acts on, where "416x
// chance" only explains why it is worth printing. A count of zero has no finite
// ratio, so it renders as the word — culture's own contract.
func practiceTravels(w io.Writer, ov *Overview) {
	if len(ov.Travels) == 0 && len(ov.Avoids) == 0 {
		return
	}
	fmt.Fprintf(w, "### What travels with what\n\n")
	fmt.Fprintf(w, "Co-occurrence measured against chance across every function. Only relationships at "+
		"least twice — or at most half — as common as chance are reported; near-chance company is not "+
		"culture. Each kind is listed separately, because there are far more call tokens than "+
		"concepts and one shared list is all calls. Within a kind, strongest first means lift "+
		"weighted by how many functions carry it — a 100× relationship holding for three functions "+
		"is a weaker finding than a 30× one holding for thirty.\n\n")

	for _, g := range ov.Travels {
		fmt.Fprintf(w, "**Together more than chance — %s**\n\n", g.Kind)
		for _, a := range g.Rows {
			fmt.Fprintf(w, "- %s — %s chance\n", conditional(a), lift(a.Ratio))
		}
		if g.More > 0 {
			fmt.Fprintf(w, "- _%d more not listed_\n", g.More)
		}
		fmt.Fprintln(w)
	}
	for _, g := range ov.Avoids {
		fmt.Fprintf(w, "**Apart more than chance — %s**\n\n", g.Kind)
		for _, a := range g.Rows {
			if a.Never {
				fmt.Fprintf(w, "- **no** `%s` function has `%s` — chance alone would give about %.0f of %d\n",
					mdEscape(a.A), mdEscape(a.B), a.Missing, a.AOf)
				continue
			}
			fmt.Fprintf(w, "- %s — %s chance\n", conditional(a), lift(a.Ratio))
		}
		if g.More > 0 {
			fmt.Fprintf(w, "- _%d more not listed_\n", g.More)
		}
		fmt.Fprintln(w)
	}
}

// conditional states an association as a fraction of the tag's own population.
// A is always the tag, so the denominator is always well-formed.
func conditional(a AssocRow) string {
	of, subject, object := a.AOf, a.A, a.B
	// For tag~tag either side could be the denominator, and the smaller
	// population makes the sharper statement about the same fact: "16 of 33
	// retry functions" says more than "16 of 436 concurrency functions".
	if a.BOf > 0 && a.BOf < a.AOf {
		of, subject, object = a.BOf, a.B, a.A
	}
	if of <= 0 {
		return fmt.Sprintf("`%s` with `%s` — %s", mdEscape(a.A), mdEscape(a.B), plural(a.Count, "function"))
	}
	return fmt.Sprintf("%d of %d `%s` functions also `%s`", a.Count, of, mdEscape(subject), mdEscape(object))
}

// lift renders a ratio with enough precision to be believed. Whole numbers past
// ten, one decimal below it — where the gap between 4.4 and 4.6 is the gap
// between two rows.
func lift(r float64) string {
	if r >= 10 {
		return fmt.Sprintf("%.0f×", r)
	}
	return fmt.Sprintf("%.1f×", r)
}

// practiceDrift names the functions that claim a concept and then realize it
// unlike every other member.
//
// This closes a gap the tool has carried: a drifting function that happens to
// appear in a reported pair got a culture note, and one that appeared in no
// pair was a number on stderr and nothing else. Those are the interesting ones.
// A function that drifts *and* has a near-duplicate is usually one half of a
// copy; a function that drifts alone is a decision somebody made on their own.
func practiceDrift(w io.Writer, ov *Overview) {
	if len(ov.Drift) == 0 {
		return
	}
	fmt.Fprintf(w, "### Functions drifting from their own concept\n\n")
	fmt.Fprintf(w, "These carry a tag but look nothing like the other functions carrying it. "+
		"Typicality is measured against the concept's own median, so a genuinely varied concept "+
		"lowers its own bar and a tight one can flag nobody.\n\n")

	// The marker column exists only when something is marked. An always-present
	// empty column trains a reader to ignore it, and then it is worthless on
	// the corpus where it finally has something to say.
	anyAlone := false
	for _, d := range ov.Drift {
		if d.Unpaired {
			anyAlone = true
			break
		}
	}

	if anyAlone {
		fmt.Fprintf(w, "| Function | Concept | Typicality | Concept median | |\n|---|---|---:|---:|---|\n")
	} else {
		fmt.Fprintf(w, "| Function | Concept | Typicality | Concept median |\n|---|---|---:|---:|\n")
	}
	for _, d := range ov.Drift {
		fmt.Fprintf(w, "| `%s` <br/>`%s:%d` | `%s` | `%.2f` | `%.2f`",
			mdEscape(d.Name), mdEscape(d.File), d.Line, mdEscape(d.Tag), d.Typicality, d.Median)
		if anyAlone {
			alone := ""
			if d.Unpaired {
				alone = "no near-duplicate"
			}
			fmt.Fprintf(w, " | %s", alone)
		}
		fmt.Fprintf(w, " |\n")
	}
	fmt.Fprintln(w)
	if ov.DriftMore > 0 {
		fmt.Fprintf(w, "_%d more unusual realizations not listed._\n\n", ov.DriftMore)
	}
	if anyAlone {
		fmt.Fprintf(w, "A row marked _no near-duplicate_ appears in no reported pair: nothing else in this "+
			"report explains it, which makes it drift rather than duplication.\n\n")
	}
}

// plural renders a count with its noun, because "1 functions" in a report about
// code quality is the kind of thing a reader notices instead of the finding.
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

package reporter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/parser"
)

// conceptsShown bounds a rendered concept list. A learned vocabulary is not
// fourteen tags: a function can plausibly belong to a dozen concepts, and a
// line listing all of them is a wall rather than a finding. The strongest few
// are what a reader weighs; the rest are counted.
const conceptsShown = 4

// conceptList renders a unit's learned concepts for a report line:
//
//	sql.Open+QueryRow 0.82, sql.Tx.Commit 0.61, +3 more
//
// Strongest first, ties on ID — not the stored ascending-ID order, because the
// point of the cap is to show the concepts that carry the unit.
//
// The confidence is printed because it is the finding. A concept is derived
// from this corpus's own vocabulary, so "carries it" is a matter of degree, and
// a reader deciding whether two functions really do the same work needs to see
// whether both sides mean it. Two decimals: the number is a saturating scale,
// not a probability, and more digits would suggest a precision it does not have.
func conceptList(cs []parser.Concept) string {
	if len(cs) == 0 {
		return ""
	}
	ranked := append([]parser.Concept(nil), cs...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Confidence != ranked[j].Confidence {
			return ranked[i].Confidence > ranked[j].Confidence
		}
		return ranked[i].ID < ranked[j].ID
	})
	shown := ranked
	if len(shown) > conceptsShown {
		shown = shown[:conceptsShown]
	}
	parts := make([]string, 0, len(shown)+1)
	for _, c := range shown {
		parts = append(parts, fmt.Sprintf("%s %.2f", c.ID, c.Confidence))
	}
	if rest := len(ranked) - len(shown); rest > 0 {
		parts = append(parts, fmt.Sprintf("+%d more", rest))
	}
	return strings.Join(parts, ", ")
}

// conceptCell is conceptList for a markdown table, with the em dash the tables
// use for "nothing here".
func conceptCell(cs []parser.Concept) string {
	if s := conceptList(cs); s != "" {
		return mdEscape(s)
	}
	return "—"
}

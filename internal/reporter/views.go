package reporter

import (
	"fmt"
	"strings"

	"github.com/LukasSelin/doppel/internal/comparator"
)

// viewsLine renders the concept views of a pair for the text report, or ""
// when the run measured no feature view — an unmeasured view is not a zero,
// and a line saying 0.00 would claim one. The five numbers are printed
// unblended, like code-shape and overlap, and the clause names the direction
// of a disagreement rather than only its size: which of the two readings
// failed is what a reader acts on.
func viewsLine(v comparator.ConceptViews) string {
	if !v.HasFeature {
		return ""
	}
	return fmt.Sprintf("concept views: shape %.2f  corpus %.2f  feature %.2f  a-in-b %.2f  b-in-a %.2f%s",
		v.Shape, v.Corpus, v.Feature, v.AInB, v.BInA, viewsClause(v))
}

// viewsClause is the disagreement, in words, or "".
func viewsClause(v comparator.ConceptViews) string {
	switch {
	case !v.HasFeature || !v.Disagree:
		return ""
	case v.Feature > v.Shape:
		return " (taxonomy misses shared vocabulary)"
	default:
		return " (taxonomy asserts kinship the vocabularies lack)"
	}
}

// sharedVocabularyLine names the strongest features both sides' concepts
// carry — the feature view's evidence — or "" when there are none.
func sharedVocabularyLine(v comparator.ConceptViews) string {
	if !v.HasFeature || len(v.SharedVocabulary) == 0 {
		return ""
	}
	names := make([]string, len(v.SharedVocabulary))
	for i, f := range v.SharedVocabulary {
		names[i] = f.Name
	}
	return "shared vocabulary: " + strings.Join(names, ", ")
}

// mdViewsLine is the markdown twin of viewsLine, numbers in code spans, the
// clause after a dash rather than in parentheses.
func mdViewsLine(v comparator.ConceptViews) string {
	if !v.HasFeature {
		return ""
	}
	clause := strings.TrimSpace(viewsClause(v))
	if clause != "" {
		clause = " — " + strings.Trim(clause, "()")
	}
	return fmt.Sprintf("**Concept views:** shape `%.2f`, corpus `%.2f`, feature `%.2f`, a-in-b `%.2f`, b-in-a `%.2f`%s",
		v.Shape, v.Corpus, v.Feature, v.AInB, v.BInA, clause)
}

// mdSharedVocabularyLine is the markdown twin of sharedVocabularyLine, each
// feature in its own code span and escaped like every other identifier
// quoted from the analysed source.
func mdSharedVocabularyLine(v comparator.ConceptViews) string {
	if !v.HasFeature || len(v.SharedVocabulary) == 0 {
		return ""
	}
	names := make([]string, len(v.SharedVocabulary))
	for i, f := range v.SharedVocabulary {
		names[i] = "`" + mdEscape(f.Name) + "`"
	}
	return "**Shared vocabulary:** " + strings.Join(names, ", ")
}

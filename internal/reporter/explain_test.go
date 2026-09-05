package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
)

func explainPair(sentence string) analyzer.SimilarPair {
	p := samplePair(nil)
	p.Explain = sentence
	return p
}

func TestPrintRendersExplain(t *testing.T) {
	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{explainPair("identical after rename, commutative-reorder")}, sampleUnits, Meta{})
	if !strings.Contains(b.String(), "  explain: identical after rename, commutative-reorder\n") {
		t.Errorf("text report missing the explain line:\n%s", b.String())
	}
}

func TestPrintMarkdownRendersExplain(t *testing.T) {
	var b strings.Builder
	PrintMarkdown(&b, []analyzer.SimilarPair{explainPair("differs by one extra defer")}, sampleUnits, Meta{})
	if !strings.Contains(b.String(), "**Explain:** differs by one extra defer") {
		t.Errorf("markdown report missing the explain line:\n%s", b.String())
	}
}

// The HTML half of this file's coverage used to live here, against the
// broadsheet report's pair card. That report was replaced by the dashboard,
// which reporter does not render — the equivalent assertion is
// TestDashboardCarriesContainmentAndExplain in package cmd.

// A pair the pipeline never annotated renders nothing, so a report written
// before explanations existed is byte-identical to one written now.
func TestExplainOmittedWhenEmpty(t *testing.T) {
	var text, md strings.Builder
	Print(&text, []analyzer.SimilarPair{explainPair("")}, sampleUnits, Meta{})
	PrintMarkdown(&md, []analyzer.SimilarPair{explainPair("")}, sampleUnits, Meta{})
	if strings.Contains(text.String(), "explain:") {
		t.Errorf("text report rendered an empty explain line:\n%s", text.String())
	}
	if strings.Contains(md.String(), "**Explain:**") {
		t.Errorf("markdown report rendered an empty explain line:\n%s", md.String())
	}
}

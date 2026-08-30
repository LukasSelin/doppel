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
	Print(&b, []analyzer.SimilarPair{explainPair("identical after rename, commutative-reorder")}, Meta{})
	if !strings.Contains(b.String(), "  explain: identical after rename, commutative-reorder\n") {
		t.Errorf("text report missing the explain line:\n%s", b.String())
	}
}

func TestPrintMarkdownRendersExplain(t *testing.T) {
	var b strings.Builder
	PrintMarkdown(&b, []analyzer.SimilarPair{explainPair("differs by one extra defer")}, Meta{})
	if !strings.Contains(b.String(), "**Explain:** differs by one extra defer") {
		t.Errorf("markdown report missing the explain line:\n%s", b.String())
	}
}

func TestHTMLPairCardRendersExplain(t *testing.T) {
	r := sampleHTMLReport()
	r.Strips[0].Pairs[0].Explain = "differs by one extra defer, two extra if"
	out := renderHTML(t, r)
	if !strings.Contains(out, "differs by one extra defer, two extra if") {
		t.Errorf("pair card missing the explain sentence:\n%s", firstMatch(out, "Match #1"))
	}
}

// A pair the pipeline never annotated renders nothing, so a report written
// before explanations existed is byte-identical to one written now.
func TestExplainOmittedWhenEmpty(t *testing.T) {
	var text, md strings.Builder
	Print(&text, []analyzer.SimilarPair{explainPair("")}, Meta{})
	PrintMarkdown(&md, []analyzer.SimilarPair{explainPair("")}, Meta{})
	if strings.Contains(text.String(), "explain:") {
		t.Errorf("text report rendered an empty explain line:\n%s", text.String())
	}
	if strings.Contains(md.String(), "**Explain:**") {
		t.Errorf("markdown report rendered an empty explain line:\n%s", md.String())
	}
}

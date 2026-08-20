package reporter

import (
	"strings"
	"testing"

	"github.com/lukse/doppel/internal/analyzer"
	"github.com/lukse/doppel/internal/parser"
)

func samplePair(ret *analyzer.Retrieval) analyzer.SimilarPair {
	return analyzer.SimilarPair{
		A:         parser.CodeUnit{Name: "Alpha", File: "a.go", StartLine: 3, Package: "fix"},
		B:         parser.CodeUnit{Name: "Beta", File: "b.go", StartLine: 9, Package: "fix"},
		Score:     0.8125,
		Retrieval: ret,
	}
}

func TestPrintLabelsShapeScoreAndEvidence(t *testing.T) {
	var b strings.Builder
	pairs := []analyzer.SimilarPair{samplePair(&analyzer.Retrieval{
		Shape: 1.5, Concept: 0.25, Call: 0.75, Total: 2.5,
		Channels: []string{"shape", "call"},
	})}
	Print(&b, pairs, Meta{Threshold: 0.6, TotalFuncs: 10})
	out := b.String()

	if !strings.Contains(out, "code-shape: 0.8125") {
		t.Errorf("output missing code-shape label:\n%s", out)
	}
	if strings.Contains(out, "score:") {
		t.Errorf("output still uses the misleading score label:\n%s", out)
	}
	if !strings.Contains(out, "evidence: 2.50  (shape 1.50  concept 0.25  call 0.75)") {
		t.Errorf("output missing evidence line:\n%s", out)
	}
	if strings.Contains(out, "retrieved-via") {
		t.Errorf("provenance shown without Debug:\n%s", out)
	}
}

func TestPrintDebugShowsProvenance(t *testing.T) {
	var b strings.Builder
	pairs := []analyzer.SimilarPair{samplePair(&analyzer.Retrieval{
		Total: 1, Channels: []string{"shape", "call"},
	})}
	Print(&b, pairs, Meta{Debug: true})
	if !strings.Contains(b.String(), "retrieved-via: shape+call") {
		t.Errorf("debug output missing provenance:\n%s", b.String())
	}
}

func TestPrintNilRetrievalOmitsEvidence(t *testing.T) {
	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{samplePair(nil)}, Meta{})
	if strings.Contains(b.String(), "evidence:") {
		t.Errorf("nil-Retrieval pair rendered an evidence line:\n%s", b.String())
	}
}

func TestPrintMarkdownLabels(t *testing.T) {
	var b strings.Builder
	pairs := []analyzer.SimilarPair{samplePair(&analyzer.Retrieval{
		Shape: 1.5, Concept: 0.25, Call: 0.75, Total: 2.5,
		Channels: []string{"concept"},
	})}
	PrintMarkdown(&b, pairs, Meta{Threshold: 0.6, TotalFuncs: 10, Debug: true})
	out := b.String()

	if !strings.Contains(out, "## Match #1 — Code-shape: `0.8125`") {
		t.Errorf("markdown missing Code-shape header:\n%s", out)
	}
	if !strings.Contains(out, "**Evidence:** `2.50` (shape 1.50, concept 0.25, call 0.75)") {
		t.Errorf("markdown missing evidence line:\n%s", out)
	}
	if !strings.Contains(out, "**Retrieved via:** concept") {
		t.Errorf("markdown debug missing provenance:\n%s", out)
	}
}

func cultureNote() analyzer.CultureNote {
	return analyzer.CultureNote{
		Tag: "transaction", Side: "B", Typicality: 0.21, ConceptMedian: 0.68,
		Channels: []analyzer.CultureChannel{
			{Name: "calls", Typicality: 0.10}, {Name: "flow", Typicality: 0.45},
			{Name: "cotags", Typicality: 0.20}, {Name: "role", Typicality: 0.00},
			{Name: "package", Typicality: 0.30},
		},
	}
}

func TestPrintCultureNotes(t *testing.T) {
	pair := samplePair(nil)
	pair.Culture = []analyzer.CultureNote{cultureNote()}

	var plain strings.Builder
	Print(&plain, []analyzer.SimilarPair{pair}, Meta{})
	if !strings.Contains(plain.String(),
		"culture: B realizes transaction atypically (typicality 0.21, concept median 0.68)") {
		t.Errorf("plain report missing culture line:\n%s", plain.String())
	}
	if strings.Contains(plain.String(), "channels:") {
		t.Errorf("channel detail shown without Debug:\n%s", plain.String())
	}

	var debug strings.Builder
	Print(&debug, []analyzer.SimilarPair{pair}, Meta{Debug: true})
	if !strings.Contains(debug.String(),
		"    channels: calls 0.10  flow 0.45  cotags 0.20  role 0.00  package 0.30") {
		t.Errorf("debug report missing channels line:\n%s", debug.String())
	}
}

func TestPrintMarkdownCultureNotes(t *testing.T) {
	pair := samplePair(nil)
	pair.Culture = []analyzer.CultureNote{cultureNote()}

	var b strings.Builder
	PrintMarkdown(&b, []analyzer.SimilarPair{pair}, Meta{Debug: true})
	out := b.String()
	if !strings.Contains(out,
		"**Culture:** B realizes `transaction` atypically (typicality 0.21, concept median 0.68)") {
		t.Errorf("markdown missing culture line:\n%s", out)
	}
	if !strings.Contains(out, "**Channels (B/transaction):** calls 0.10  flow 0.45") {
		t.Errorf("markdown debug missing channels line:\n%s", out)
	}
}

func TestPrintNilCultureRendersNothingNew(t *testing.T) {
	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{samplePair(nil)}, Meta{Debug: true})
	if strings.Contains(b.String(), "culture:") {
		t.Errorf("nil-Culture pair rendered a culture line:\n%s", b.String())
	}
}

func TestPrintEmpty(t *testing.T) {
	var b strings.Builder
	Print(&b, nil, Meta{Threshold: 0.6, TotalFuncs: 5})
	if !strings.Contains(b.String(), "No similar function pairs found") {
		t.Errorf("empty report missing message:\n%s", b.String())
	}
}

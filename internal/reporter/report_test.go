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

func TestPrintEmpty(t *testing.T) {
	var b strings.Builder
	Print(&b, nil, Meta{Threshold: 0.6, TotalFuncs: 5})
	if !strings.Contains(b.String(), "No similar function pairs found") {
		t.Errorf("empty report missing message:\n%s", b.String())
	}
}

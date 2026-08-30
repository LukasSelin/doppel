package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
)

func queryFixture() (parser.CodeUnit, concepter.ConceptDoc, []QueryMatch) {
	probe := parser.CodeUnit{Package: "billing", Name: "ValidateRef", Patterns: []string{"validation"}}
	probeDoc := concepter.ConceptDoc{Role: "leaf"}
	m := QueryMatch{
		Unit: parser.CodeUnit{Package: "fedex", Name: "ValidateCreds", File: "internal/fedex/creds.go", StartLine: 41,
			Patterns: []string{"validation"}},
		Doc:      concepter.ConceptDoc{Role: "leaf"},
		Locality: 0.4,
	}
	m.Candidate = retriever.Candidate{Total: 14.2, Shape: 6.1, Concept: 3.0, Call: 5.1}
	m.Candidate.Breakdown.Score = 0.82
	m.Candidate.Chains = []retriever.SharedLabel{{Energy: 2.8, Depth: 2, Count: 3, Render: "depth-2 IF"}}
	return probe, probeDoc, m.wrap()
}

func (m QueryMatch) wrap() []QueryMatch { return []QueryMatch{m} }

func TestPrintQueryStatesEvidenceAndLocalityUnblended(t *testing.T) {
	probe, doc, matches := queryFixture()
	var b strings.Builder
	PrintQuery(&b, probe, doc, matches, QueryMeta{CorpusFuncs: 100, ResolvedCalls: 2})
	out := b.String()

	for _, want := range []string{
		"query: billing.ValidateRef",
		"tags: validation",
		"resolved calls: 2",
		"fedex.ValidateCreds",
		"internal/fedex/creds.go:41",
		"evidence: 14.2 nats (shape 6.1, concept 3.0, call 5.1)",
		"code-shape: 0.82",
		"locality: 0.40",
		"shared structure:",
		"2.80  depth-2 IF ×3",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// A probe with no resolved calls says so once, up front, instead of letting a
// column of zero localities imply everything in the corpus is far away.
func TestPrintQueryExplainsEmptyLocality(t *testing.T) {
	probe, doc, matches := queryFixture()
	var b strings.Builder
	PrintQuery(&b, probe, doc, matches, QueryMeta{CorpusFuncs: 100, ResolvedCalls: 0})
	if !strings.Contains(b.String(), "no resolved calls in the snippet") {
		t.Errorf("missing the empty-locality explanation:\n%s", b.String())
	}
}

func TestPrintQueryNoMatchesIsAnAnswer(t *testing.T) {
	probe, doc, _ := queryFixture()
	var b strings.Builder
	PrintQuery(&b, probe, doc, nil, QueryMeta{CorpusFuncs: 100, ResolvedCalls: 1})
	if !strings.Contains(b.String(), "No related functions found") {
		t.Errorf("no-match case says nothing useful:\n%s", b.String())
	}
}

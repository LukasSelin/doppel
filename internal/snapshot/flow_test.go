package snapshot

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/fingerprint"
)

// The flow view rides into --format json rounded, -1 when a pair was never
// annotated, and never reaches a delta.
func TestBuildCarriesFlowAndDiffIgnoresIt(t *testing.T) {
	u, d, p, c := sampleInputs()
	params := Params{Threshold: 0.6, MinNodes: 12, TestsMode: "exclude"}

	unannotated := Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})
	if got := unannotated.Pairs[0]; got.FlowSteps != -1 || got.FlowTypes != -1 {
		t.Fatalf("unannotated pair: flowSteps %v flowTypes %v, want -1", got.FlowSteps, got.FlowTypes)
	}

	p[0].Flow = &fingerprint.FlowScore{Steps: 0.8333, Types: 0.5}
	before := Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})
	if got := before.Pairs[0]; got.FlowSteps != 0.83 || got.FlowTypes != 0.5 {
		t.Fatalf("flowSteps %v flowTypes %v, want 0.83 and 0.5", got.FlowSteps, got.FlowTypes)
	}
	p[0].Flow = &fingerprint.FlowScore{Steps: 0.2, Types: 0.1}
	after := Build(u, d, p, c, nil, "", "test", params, CorpusMetrics{})
	if delta := Diff(before, after); !delta.Comparable || !delta.Empty() {
		t.Fatalf("a changed flow view moved the delta: comparable %v empty %v", delta.Comparable, delta.Empty())
	}
}

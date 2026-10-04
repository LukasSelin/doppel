package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

func flowUnits() []parser.CodeUnit {
	units := append([]parser.CodeUnit(nil), sampleUnits...)
	units[0].Fingerprint = fingerprint.Fingerprint{Nodes: 9, Steps: []fingerprint.FlowStep{
		{Op: fingerprint.FlowIf}, {Op: fingerprint.FlowCall, Name: "Get"}, {Op: fingerprint.FlowReturn}}}
	units[1].Fingerprint = fingerprint.Fingerprint{Nodes: 9, Steps: []fingerprint.FlowStep{
		{Op: fingerprint.FlowIf}, {Op: fingerprint.FlowCall, Name: "Delete"}, {Op: fingerprint.FlowReturn}}}
	return units
}

func flowPair(units []parser.CodeUnit) analyzer.SimilarPair {
	p := samplePair(nil)
	fs := fingerprint.FlowSimilarity(units[0].Fingerprint, units[1].Fingerprint)
	p.Flow = &fs
	return p
}

func TestPrintRendersFlow(t *testing.T) {
	units := flowUnits()
	var text, md strings.Builder
	Print(&text, []analyzer.SimilarPair{flowPair(units)}, units, Meta{})
	PrintMarkdown(&md, []analyzer.SimilarPair{flowPair(units)}, units, Meta{})
	if want := "  flow: steps 0.83 (2 same, 1 retargeted of 3 and 3)  types 1.00\n"; !strings.Contains(text.String(), want) {
		t.Errorf("text report missing %q:\n%s", want, text.String())
	}
	if want := "**Flow:** `steps 0.83 (2 same, 1 retargeted of 3 and 3)  types 1.00`"; !strings.Contains(md.String(), want) {
		t.Errorf("markdown report missing %q:\n%s", want, md.String())
	}
	// The alignment is --debug only: it is as long as the longer function.
	if strings.Contains(text.String(), "call Get") || strings.Contains(md.String(), "call Get") {
		t.Errorf("alignment rendered without --debug")
	}
}

func TestPrintDebugRendersFlowAlignment(t *testing.T) {
	units := flowUnits()
	var text strings.Builder
	Print(&text, []analyzer.SimilarPair{flowPair(units)}, units, Meta{Debug: true})
	if !strings.Contains(text.String(), "    ~ call Get                     call Delete\n") {
		t.Errorf("debug report missing the retargeted row:\n%s", text.String())
	}
}

// A pair the pipeline never annotated renders no flow line, so a library
// caller's report is unchanged.
func TestFlowOmittedWhenUnannotated(t *testing.T) {
	var text strings.Builder
	Print(&text, []analyzer.SimilarPair{samplePair(nil)}, sampleUnits, Meta{Debug: true})
	if strings.Contains(text.String(), "flow:") {
		t.Errorf("unannotated pair rendered a flow line:\n%s", text.String())
	}
}

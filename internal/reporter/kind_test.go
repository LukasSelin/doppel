package reporter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/parser"
)

func interfaceKind() *analyzer.KindNote {
	return &analyzer.KindNote{
		Kind: analyzer.KindInterfaceImpl, Method: "Validate", Signature: "(context.Context) (error)",
		Receivers: []string{"*AWS", "*GCP"}, Packages: []string{"aws", "gcp"}, Relation: analyzer.RelationSiblings,
	}
}

func forkKind() *analyzer.KindNote {
	return &analyzer.KindNote{
		Kind: analyzer.KindFork, Method: "evalCall",
		Names: []string{"*state.evalCallOld", "*state.evalCall"}, Packages: []string{"template"}, Relation: analyzer.RelationSamePackage,
	}
}

func TestPrintKindNote(t *testing.T) {
	p := samplePair(nil)
	p.Kind = interfaceKind()
	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{p}, Meta{})
	want := "  kind: interface implementations — both implement Validate(context.Context) (error) on *AWS and *GCP, sibling packages aws and gcp\n"
	if !strings.Contains(b.String(), want) {
		t.Errorf("missing kind line %q in:\n%s", want, b.String())
	}
	// The kind is a claim about the pair: it sits right under the unit lines,
	// before the breakdown.
	out := b.String()
	if strings.Index(out, "kind:") > strings.Index(out, "wl ") {
		t.Errorf("kind line must precede the breakdown:\n%s", out)
	}

	p.Kind = forkKind()
	b.Reset()
	Print(&b, []analyzer.SimilarPair{p}, Meta{})
	want = "  kind: diverged copy — *state.evalCallOld and *state.evalCall share the stem evalCall in package template\n"
	if !strings.Contains(b.String(), want) {
		t.Errorf("missing fork line %q in:\n%s", want, b.String())
	}
}

func TestPrintMarkdownKindNote(t *testing.T) {
	p := samplePair(nil)
	p.Kind = interfaceKind()
	var b strings.Builder
	PrintMarkdown(&b, []analyzer.SimilarPair{p}, Meta{})
	want := "**Kind:** interface implementations — both implement `Validate(context.Context) (error)` on `*AWS` and `*GCP`, sibling packages `aws` and `gcp`\n\n"
	if !strings.Contains(b.String(), want) {
		t.Errorf("missing markdown kind %q in:\n%s", want, b.String())
	}
}

func TestPrintNilKindRendersNothing(t *testing.T) {
	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{samplePair(nil)}, Meta{Debug: true})
	PrintMarkdown(&b, []analyzer.SimilarPair{samplePair(nil)}, Meta{Debug: true})
	if strings.Contains(b.String(), "kind:") || strings.Contains(b.String(), "Kind:") {
		t.Errorf("unlabeled pair rendered a kind line:\n%s", b.String())
	}
}

func TestPrintFamilyKind(t *testing.T) {
	units := []parser.CodeUnit{
		{Name: "*AWS.Validate", Package: "aws", File: "carrier/aws/a.go", StartLine: 1},
		{Name: "*GCP.Validate", Package: "gcp", File: "carrier/gcp/g.go", StartLine: 1},
		{Name: "*Azure.Validate", Package: "azure", File: "carrier/azure/z.go", StartLine: 1},
	}
	fam := family.Family{Members: []int{0, 1, 2}, MinEdge: 0.72, MeanEdge: 0.8, Evidence: 120}
	fam.Kind = &analyzer.KindNote{
		Kind: analyzer.KindInterfaceImpl, Method: "Validate", Signature: "(context.Context) (error)",
		Receivers: []string{"*AWS", "*GCP", "*Azure"}, Packages: []string{"aws", "gcp", "azure"}, Relation: analyzer.RelationSiblings,
	}
	fams, stats := []family.Family{fam}, family.Stats{Families: 1, Members: 3}

	var b strings.Builder
	PrintFamilies(&b, fams, stats, units, 5)
	if !strings.Contains(b.String(), "evidence 120   kind: interface implementations of Validate(context.Context) (error), sibling packages aws, gcp and azure\n") {
		t.Errorf("text F-line lacks the kind suffix:\n%s", b.String())
	}

	b.Reset()
	PrintMarkdownFamilies(&b, fams, stats, units, 5)
	if !strings.Contains(b.String(), "evidence `120`, interface implementations of `Validate(context.Context) (error)`, sibling packages `aws`, `gcp` and `azure`\n") {
		t.Errorf("markdown heading lacks the kind suffix:\n%s", b.String())
	}

	b.Reset()
	if err := PrintFamiliesJSON(&b, fams, stats, units, ""); err != nil {
		t.Fatal(err)
	}
	var out FamiliesJSON
	if err := json.Unmarshal([]byte(b.String()), &out); err != nil {
		t.Fatal(err)
	}
	if out.Families[0].Kind != analyzer.KindInterfaceImpl || !strings.HasPrefix(out.Families[0].KindLabel, "interface implementations of Validate") {
		t.Errorf("JSON kind = %q / %q", out.Families[0].Kind, out.Families[0].KindLabel)
	}

	// An unlabeled family renders no suffix and no JSON key.
	fam.Kind = nil
	b.Reset()
	PrintFamilies(&b, []family.Family{fam}, stats, units, 5)
	if strings.Contains(b.String(), "kind:") {
		t.Errorf("unlabeled family rendered a kind:\n%s", b.String())
	}
}

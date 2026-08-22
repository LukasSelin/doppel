package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/parser"
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

func TestPrintTrophicAndSharedStructure(t *testing.T) {
	pairs := []analyzer.SimilarPair{samplePair(&analyzer.Retrieval{
		Total: 25.1, Shape: 23.7, TrophicSim: 0.43,
		Chains: []analyzer.SharedChain{
			{Level: 3, Energy: 6.2, Render: "for{ call:TrimSpace call:Atoi call:append }"},
			{Level: 3, Energy: 1.8, Render: "seq[ assign:=(call:Atoi) ; if(bin:!=(id,nil)) ]"},
		},
	})}
	var b strings.Builder
	Print(&b, pairs, Meta{})
	out := b.String()
	if !strings.Contains(out, "trophic: 0.43") {
		t.Errorf("missing trophic line:\n%s", out)
	}
	if !strings.Contains(out, "shared structure:\n    6.20  for{ call:TrimSpace call:Atoi call:append }\n    1.80  seq[") {
		t.Errorf("missing shared-structure block:\n%s", out)
	}
}

func TestPrintNoChainsOmitsSharedStructure(t *testing.T) {
	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{samplePair(&analyzer.Retrieval{TrophicSim: 0.1})}, Meta{})
	if strings.Contains(b.String(), "shared structure:") {
		t.Errorf("empty chains rendered a shared-structure header:\n%s", b.String())
	}
	if !strings.Contains(b.String(), "trophic: 0.10") {
		t.Errorf("trophic line missing when chains empty:\n%s", b.String())
	}
}

func TestPrintMarkdownTrophic(t *testing.T) {
	pairs := []analyzer.SimilarPair{samplePair(&analyzer.Retrieval{
		TrophicSim: 0.43,
		Chains:     []analyzer.SharedChain{{Level: 3, Energy: 6.2, Render: "for{ call:Scan }"}},
	})}
	var b strings.Builder
	PrintMarkdown(&b, pairs, Meta{})
	out := b.String()
	if !strings.Contains(out, "**Trophic:** `0.43`") {
		t.Errorf("markdown missing trophic:\n%s", out)
	}
	if !strings.Contains(out, "- `6.20` — `for{ call:Scan }`") {
		t.Errorf("markdown missing shared-structure bullet:\n%s", out)
	}
}

func cultureNote() analyzer.CultureNote {
	return analyzer.CultureNote{
		Tag: "transaction", Side: "B", Typicality: 0.21, ConceptMedian: 0.68, Convention: 0.85,
		Channels: []analyzer.CultureChannel{
			{Name: "calls", Typicality: 0.10}, {Name: "flow", Typicality: 0.45},
			{Name: "cotags", Typicality: 0.20}, {Name: "role", Typicality: 0.00},
			{Name: "package", Typicality: 0.30},
		},
	}
}

func habitatNote() analyzer.HabitatNote {
	return analyzer.HabitatNote{
		Side: "B", Package: "queue", Fit: 0.21, PackageNorm: 0.84,
		Channels: []analyzer.HabitatChannel{
			{Name: "calls", Surprise: 3.20}, {Name: "flow", Surprise: 0.45},
			{Name: "tags", Surprise: 1.10}, {Name: "role", Surprise: 0.00},
		},
	}
}

func TestPrintHabitatNotes(t *testing.T) {
	pair := samplePair(nil)
	pair.Habitat = []analyzer.HabitatNote{habitatNote()}

	var plain strings.Builder
	Print(&plain, []analyzer.SimilarPair{pair}, Meta{})
	if !strings.Contains(plain.String(),
		"  habitat: B fits poorly in queue (fit 0.21, package norm 0.84)") {
		t.Errorf("plain report missing habitat line:\n%s", plain.String())
	}
	if strings.Contains(plain.String(), "surprise:") {
		t.Errorf("surprise detail shown without Debug:\n%s", plain.String())
	}

	var debug strings.Builder
	Print(&debug, []analyzer.SimilarPair{pair}, Meta{Debug: true})
	if !strings.Contains(debug.String(),
		"    surprise: calls 3.20  flow 0.45  tags 1.10  role 0.00") {
		t.Errorf("debug report missing surprise line:\n%s", debug.String())
	}
}

func TestPrintMarkdownHabitatNotes(t *testing.T) {
	pair := samplePair(nil)
	pair.Habitat = []analyzer.HabitatNote{habitatNote()}

	var b strings.Builder
	PrintMarkdown(&b, []analyzer.SimilarPair{pair}, Meta{Debug: true})
	out := b.String()
	if !strings.Contains(out, "**Habitat:** B fits poorly in `queue` (fit 0.21, package norm 0.84)") {
		t.Errorf("markdown missing habitat line:\n%s", out)
	}
	if !strings.Contains(out, "**Surprise (B/queue):** calls 3.20  flow 0.45  tags 1.10  role 0.00") {
		t.Errorf("markdown debug missing surprise line:\n%s", out)
	}
}

func TestPrintNilHabitatRendersNothing(t *testing.T) {
	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{samplePair(nil)}, Meta{Debug: true})
	if strings.Contains(b.String(), "habitat:") {
		t.Errorf("nil-Habitat pair rendered a habitat line:\n%s", b.String())
	}
}

func profileNote() analyzer.ProfileNote {
	return analyzer.ProfileNote{
		Side: "A", State: "coalition", Rounds: 17, Converged: true,
		Concepts: []analyzer.ProfileMass{
			{Tag: "transaction", Mass: 0.39}, {Tag: "db_access", Mass: 0.34},
			{Tag: "error_wrapping", Mass: 0.27},
		},
		Extinct: []analyzer.ProfileMass{{Tag: "validation", Mass: 0.0008}},
	}
}

func TestPrintProfileNotes(t *testing.T) {
	pair := samplePair(nil)
	pair.Profile = []analyzer.ProfileNote{profileNote()}

	var plain strings.Builder
	Print(&plain, []analyzer.SimilarPair{pair}, Meta{})
	out := plain.String()
	if !strings.Contains(out,
		"  profile A: transaction 0.39  db_access 0.34  error_wrapping 0.27 (coalition)") {
		t.Errorf("plain report missing profile line:\n%s", out)
	}
	if strings.Contains(out, "arena A:") {
		t.Errorf("arena detail shown without Debug:\n%s", out)
	}
	// Placement: profile precedes the breakdown line.
	if strings.Index(out, "profile A:") > strings.Index(out, "ast ") {
		t.Errorf("profile line rendered after the breakdown:\n%s", out)
	}

	var debug strings.Builder
	Print(&debug, []analyzer.SimilarPair{pair}, Meta{Debug: true})
	if !strings.Contains(debug.String(),
		"    arena A: 17 rounds, converged; extinct: validation 0.0008") {
		t.Errorf("debug report missing arena line:\n%s", debug.String())
	}
}

func TestPrintMarkdownProfileNotes(t *testing.T) {
	pair := samplePair(nil)
	pair.Profile = []analyzer.ProfileNote{profileNote()}

	var b strings.Builder
	PrintMarkdown(&b, []analyzer.SimilarPair{pair}, Meta{Debug: true})
	out := b.String()
	if !strings.Contains(out,
		"**Profile A:** `transaction` 0.39, `db_access` 0.34, `error_wrapping` 0.27 (coalition)") {
		t.Errorf("markdown missing profile line:\n%s", out)
	}
	if !strings.Contains(out, "**Arena A:** 17 rounds, converged; extinct: `validation` 0.0008") {
		t.Errorf("markdown debug missing arena line:\n%s", out)
	}
}

func TestPrintNilProfileRendersNothing(t *testing.T) {
	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{samplePair(nil)}, Meta{Debug: true})
	if strings.Contains(b.String(), "profile") {
		t.Errorf("nil-Profile pair rendered a profile line:\n%s", b.String())
	}
}

func TestPrintCultureNotes(t *testing.T) {
	pair := samplePair(nil)
	pair.Culture = []analyzer.CultureNote{cultureNote()}

	var plain strings.Builder
	Print(&plain, []analyzer.SimilarPair{pair}, Meta{})
	if !strings.Contains(plain.String(),
		"culture: B realizes transaction atypically (typicality 0.21, concept median 0.68, convention 0.85)") {
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
		"**Culture:** B realizes `transaction` atypically (typicality 0.21, concept median 0.68, convention 0.85)") {
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

// A confirmed misfit with a modeled subsystem shows the subsystem fit inside
// the parentheses; without one the line is exactly as before.
func TestPrintHabitatNoteWithSubsystem(t *testing.T) {
	note := habitatNote()
	note.Subsystem, note.SubsystemFit = "tpl/", 0.30
	p := samplePair(nil)
	p.Habitat = []analyzer.HabitatNote{note}

	var b strings.Builder
	Print(&b, []analyzer.SimilarPair{p}, Meta{})
	if want := "  habitat: B fits poorly in queue (fit 0.21, package norm 0.84; subsystem tpl/ fit 0.30)\n"; !strings.Contains(b.String(), want) {
		t.Errorf("missing %q in:\n%s", want, b.String())
	}
	b.Reset()
	PrintMarkdown(&b, []analyzer.SimilarPair{p}, Meta{})
	if want := "**Habitat:** B fits poorly in `queue` (fit 0.21, package norm 0.84; subsystem `tpl/` fit 0.30)\n\n"; !strings.Contains(b.String(), want) {
		t.Errorf("missing %q in:\n%s", want, b.String())
	}
}

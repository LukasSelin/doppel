package reporter

import (
	"strings"
	"testing"
)

func samplePractice() *Overview {
	return &Overview{
		Functions: 100,
		Practice: []ConceptPractice{{
			Tag:     "transaction",
			Members: 6,
			Channels: []PracticeChannel{
				{Name: "calls", Weight: 40, Features: []PracticeFeature{
					{Name: "tx.Commit", Count: 6, P: 1.0},
					{Name: "tx.Rollback", Count: 5, P: 0.8333},
				}},
				{Name: "package", Weight: 10, Features: []PracticeFeature{
					{Name: "store", Count: 4, P: 0.6667},
				}},
			},
		}},
		Matrix: &ConceptMatrix{
			Tags:  []string{"caching", "db_access", "transaction"},
			Cells: [][]string{{"", "++", ""}, {"++", "", "never"}, {"", "never", ""}},
		},
		Travels: []AssocGroup{{Kind: "tag~call", Rows: []AssocRow{
			{Kind: "tag~call", A: "http_call", B: "net/http.NewRequest", Count: 13, AOf: 14, Ratio: 416.3},
			{Kind: "tag~call", A: "concurrency", B: "math.Float64frombits", Count: 1, AOf: 9, Ratio: 3.2},
		}}},
		Avoids: []AssocGroup{{Kind: "tag~role", Rows: []AssocRow{
			{Kind: "tag~role", A: "transaction", B: "utility", AOf: 12, Never: true, Missing: 5.4},
		}}},
		Drift: []DriftRow{
			{Name: "a.One", File: "a/one.go", Line: 12, Tag: "validation", Typicality: 0.13, Median: 0.40},
		},
	}
}

func TestPrintMarkdownPractice(t *testing.T) {
	var b strings.Builder
	PrintMarkdownPractice(&b, samplePractice())
	out := b.String()

	if !strings.Contains(out, "## Local practice") {
		t.Errorf("section heading missing:\n%s", out)
	}
	if !strings.Contains(out, "**`transaction`** — 6 functions") {
		t.Errorf("concept header missing:\n%s", out)
	}
}

// Counts, not percentages: a concept qualifies for a prototype at five members,
// and "83%" of six is a number with more digits than evidence.
func TestPrototypeRendersCountsNotPercentages(t *testing.T) {
	var b strings.Builder
	PrintMarkdownPractice(&b, samplePractice())
	out := b.String()

	if !strings.Contains(out, "| calls ×40 | `tx.Commit` | `██████████` | 6 of 6 |") {
		t.Errorf("prototype row missing or malformed:\n%s", out)
	}
	if !strings.Contains(out, "| `tx.Rollback` | `████████··` | 5 of 6 |") {
		t.Errorf("bar did not track the proportion:\n%s", out)
	}
	if strings.Contains(out, "83%") || strings.Contains(out, "67%") {
		t.Errorf("a percentage survived where a count belongs:\n%s", out)
	}
	// The channel label appears once per channel, not once per feature.
	if n := strings.Count(out, "calls ×40"); n != 1 {
		t.Errorf("channel label repeated %d times, want 1:\n%s", n, out)
	}
}

// The conditional is what a reader acts on; the lift only says why it is worth
// printing.
func TestAssociationsLeadWithTheConditional(t *testing.T) {
	var b strings.Builder
	PrintMarkdownPractice(&b, samplePractice())
	out := b.String()

	if !strings.Contains(out, "13 of 14 `http_call` functions also `net/http.NewRequest` — 416× chance") {
		t.Errorf("association not stated as a conditional:\n%s", out)
	}
	// Kinds are listed separately, or the call rows bury every concept row.
	if !strings.Contains(out, "**Together more than chance — tag~call**") {
		t.Errorf("associations not grouped by kind:\n%s", out)
	}
}

// Count 0 has no finite ratio. culture's own contract is that it renders as a
// word rather than a number, and a report printing "0.0× chance" for it would
// be stating a measurement it does not have.
func TestNeverCoOccurringRendersAsAWord(t *testing.T) {
	var b strings.Builder
	PrintMarkdownPractice(&b, samplePractice())
	out := b.String()

	if !strings.Contains(out, "**no** `transaction` function has `utility` — chance alone would give about 5 of 12") {
		t.Errorf("the never case did not render as a word:\n%s", out)
	}
	if strings.Contains(out, "0.0× chance") {
		t.Errorf("an infinite PMI leaked out as a ratio:\n%s", out)
	}
}

// The matrix is the one table here that is not a sample: nine concepts, bounded
// by construction, so it can show every cell including the ordinary ones.
func TestConceptMatrixRendersTheLowerTriangle(t *testing.T) {
	var b strings.Builder
	PrintMarkdownPractice(&b, samplePractice())
	out := b.String()

	if !strings.Contains(out, "### Which concepts share a function") {
		t.Errorf("matrix section missing:\n%s", out)
	}
	if !strings.Contains(out, "| **`db_access`** | ++ | |") {
		t.Errorf("matrix cell missing or in the wrong column:\n%s", out)
	}
	if !strings.Contains(out, "| **`transaction`** |  | never |") {
		t.Errorf("never cell missing:\n%s", out)
	}
	// The last tag never heads a column: its column would be all upper triangle.
	if strings.Contains(out, "| | `caching` | `db_access` | `transaction` |") {
		t.Errorf("the final tag was given a column:\n%s", out)
	}
}

// An all-blank grid is not a finding, it is a shrug — and doppel's own corpus
// produces exactly that.
func TestEmptyMatrixRendersNothing(t *testing.T) {
	ov := samplePractice()
	ov.Matrix = &ConceptMatrix{
		Tags:  []string{"a", "b"},
		Cells: [][]string{{"", ""}, {"", ""}},
	}
	var b strings.Builder
	PrintMarkdownPractice(&b, ov)
	if strings.Contains(b.String(), "Which concepts share a function") {
		t.Errorf("an empty matrix rendered:\n%s", b.String())
	}
}

func TestPracticePluralises(t *testing.T) {
	var b strings.Builder
	PrintMarkdownPractice(&b, samplePractice())
	if out := b.String(); strings.Contains(out, "1 functions") {
		t.Errorf("count rendered with the wrong plural:\n%s", out)
	}
}

// The marker column carries the finding, so it exists only when something is
// marked — an always-empty column trains a reader to ignore it, and then it is
// worthless on the corpus where it finally matters.
func TestDriftMarkerColumnAppearsOnlyWhenUsed(t *testing.T) {
	ov := samplePractice()
	var none strings.Builder
	PrintMarkdownPractice(&none, ov)
	out := none.String()
	if !strings.Contains(out, "| Function | Concept | Typicality | Concept median |\n") {
		t.Errorf("expected the four-column table when nothing is marked:\n%s", out)
	}
	if strings.Contains(out, "no near-duplicate") {
		t.Errorf("the explanation rendered with nothing to explain:\n%s", out)
	}

	ov.Drift[0].Unpaired = true
	var some strings.Builder
	PrintMarkdownPractice(&some, ov)
	out = some.String()
	if !strings.Contains(out, "| `a.One` <br/>`a/one.go:12` | `validation` | `0.13` | `0.40` | no near-duplicate |") {
		t.Errorf("marked row missing:\n%s", out)
	}
	if !strings.Contains(out, "A row marked _no near-duplicate_") {
		t.Errorf("the explanation is missing where it is needed:\n%s", out)
	}
}

// A corpus with none of this says nothing. Every other section in the report
// follows the same rule.
func TestEmptyPracticeRendersNothing(t *testing.T) {
	var b strings.Builder
	PrintMarkdownPractice(&b, &Overview{Functions: 10})
	if got := b.String(); got != "" {
		t.Errorf("empty practice rendered %q, want nothing", got)
	}
	var nilb strings.Builder
	PrintMarkdownPractice(&nilb, nil)
	if got := nilb.String(); got != "" {
		t.Errorf("nil overview rendered %q, want nothing", got)
	}
}

func TestPracticeIsDeterministic(t *testing.T) {
	var first strings.Builder
	PrintMarkdownPractice(&first, samplePractice())
	for i := 0; i < 20; i++ {
		var b strings.Builder
		PrintMarkdownPractice(&b, samplePractice())
		if b.String() != first.String() {
			t.Fatalf("run %d differed from the first", i)
		}
	}
}

// For a tag~tag pair either side could be the denominator, and the smaller
// population is the sharper statement about the same fact.
func TestTagPairStatesItselfAgainstTheSmallerSide(t *testing.T) {
	ov := samplePractice()
	ov.Travels = []AssocGroup{{Kind: "tag~tag", Rows: []AssocRow{
		{Kind: "tag~tag", A: "concurrency", B: "retry", Count: 16, AOf: 436, BOf: 33, Ratio: 7.2},
	}}}

	var b strings.Builder
	PrintMarkdownPractice(&b, ov)
	out := b.String()

	if !strings.Contains(out, "16 of 33 `retry` functions also `concurrency`") {
		t.Errorf("stated against the larger population:\n%s", out)
	}
	if strings.Contains(out, "of 436") {
		t.Errorf("the weaker denominator was used:\n%s", out)
	}
}

// Lift needs enough precision to justify the order it is printed in: 4.4 and
// 4.6 both round to "4x", and two rows differing only there read as a bug.
func TestLiftKeepsADecimalUnderTen(t *testing.T) {
	if got := lift(4.44); got != "4.4×" {
		t.Errorf("lift(4.44) = %q, want 4.4×", got)
	}
	if got := lift(416.3); got != "416×" {
		t.Errorf("lift(416.3) = %q, want 416×", got)
	}
}

package reporter

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/parser"
)

func familyUnits() []parser.CodeUnit {
	return []parser.CodeUnit{
		{Name: "compensationContext", Package: "dhl", File: "carrier/dhl/ctx.go", StartLine: 14},
		{Name: "compensationContext", Package: "ups", File: "carrier/ups/ctx.go", StartLine: 9},
		{Name: "compensationContext", Package: "dsv", File: "carrier/dsv/ctx.go", StartLine: 21},
		{Name: "unrelated", Package: "other", File: "other/x.go", StartLine: 3},
	}
}

func sampleFamilies() ([]family.Family, family.Stats) {
	f := []family.Family{{
		Members:   []int{0, 1, 2},
		MinEdge:   0.72,
		MeanEdge:  0.88,
		Completed: 1,
	}}
	return f, family.Stats{Components: 1, Families: 1, Members: 3, Completed: 1}
}

func TestPrintFamilies(t *testing.T) {
	fams, stats := sampleFamilies()

	var b strings.Builder
	PrintFamilies(&b, fams, stats, familyUnits(), 5)
	out := b.String()

	// The guarantee is the weakest edge, not the mean: a family is a claim
	// about all its members.
	if !strings.Contains(out, "3 members   every pair >= 0.72 code-shape") {
		t.Errorf("family line missing or not stating the weakest edge:\n%s", out)
	}
	if strings.Contains(out, "0.88") {
		t.Errorf("the mean edge was printed as if it were the guarantee:\n%s", out)
	}
	// Completed edges must be named, or a reader checking the family against
	// the pair list finds members with no pair between them.
	if !strings.Contains(out, "(1 edge scored here)") {
		t.Errorf("completed edge not disclosed:\n%s", out)
	}
	if !strings.Contains(out, "3 functions in a family") {
		t.Errorf("census summary missing:\n%s", out)
	}
	for _, want := range []string{"dhl.compensationContext", "ups.compensationContext", "dsv.compensationContext"} {
		if !strings.Contains(out, want) {
			t.Errorf("member %q missing:\n%s", want, out)
		}
	}
	if strings.Contains(out, "other.unrelated") {
		t.Errorf("a non-member was listed:\n%s", out)
	}
}

// A corpus with no families says nothing at all. A "no families" line under
// every report trains the reader to skip the place real findings appear —
// the same rule the impact digest follows.
func TestPrintNoFamiliesRendersNothing(t *testing.T) {
	var b strings.Builder
	PrintFamilies(&b, nil, family.Stats{}, familyUnits(), 5)
	if got := b.String(); got != "" {
		t.Errorf("empty census rendered %q, want nothing", got)
	}

	var md strings.Builder
	PrintMarkdownFamilies(&md, nil, family.Stats{}, familyUnits(), 5)
	if got := md.String(); got != "" {
		t.Errorf("empty markdown census rendered %q, want nothing", got)
	}
}

func TestPrintFamiliesCapsTheList(t *testing.T) {
	var fams []family.Family
	for i := 0; i < 8; i++ {
		fams = append(fams, family.Family{Members: []int{0, 1, 2}, MinEdge: 0.70})
	}
	stats := family.Stats{Components: 8, Families: 8, Members: 3}

	var b strings.Builder
	PrintFamilies(&b, fams, stats, familyUnits(), 3)
	out := b.String()

	if n := strings.Count(out, "every pair >="); n != 3 {
		t.Errorf("listed %d families, want 3:\n%s", n, out)
	}
	if !strings.Contains(out, "(5 more families not listed)") {
		t.Errorf("remainder not counted:\n%s", out)
	}
}

// A guard that drops work silently reads as "there was nothing there".
func TestPrintFamiliesDisclosesSkippedComponents(t *testing.T) {
	fams, stats := sampleFamilies()
	stats.Skipped = []int{97, 212}

	var b strings.Builder
	PrintFamilies(&b, fams, stats, familyUnits(), 5)
	if out := b.String(); !strings.Contains(out, "sizes 97, 212") {
		t.Errorf("skipped components not disclosed:\n%s", out)
	}
}

func TestPrintMarkdownFamilies(t *testing.T) {
	fams, stats := sampleFamilies()

	var b strings.Builder
	PrintMarkdownFamilies(&b, fams, stats, familyUnits(), 5)
	out := b.String()

	if !strings.Contains(out, "## Families") {
		t.Errorf("section heading missing:\n%s", out)
	}
	if !strings.Contains(out, "### Family 1 — 3 members, every pair `>= 0.72` code-shape") {
		t.Errorf("family heading missing or malformed:\n%s", out)
	}
	if !strings.Contains(out, "`carrier/dhl/ctx.go:14`") {
		t.Errorf("member location missing:\n%s", out)
	}
}

func TestPrintFamiliesJSONIsNameKeyed(t *testing.T) {
	fams, stats := sampleFamilies()

	var b strings.Builder
	if err := PrintFamiliesJSON(&b, fams, stats, familyUnits(), ""); err != nil {
		t.Fatalf("PrintFamiliesJSON: %v", err)
	}
	out := b.String()

	if !strings.Contains(out, `"minEdge": 0.72`) {
		t.Errorf("minEdge missing:\n%s", out)
	}
	// Members are ordered by name, not by the file-walk index: positions shift
	// the moment a file is added.
	dsv := strings.Index(out, "dsv.compensationContext")
	ups := strings.Index(out, "ups.compensationContext")
	if dsv < 0 || ups < 0 || dsv > ups {
		t.Errorf("members are not name-ordered:\n%s", out)
	}
}

// A large corpus has families of fifty, and the bounded listing must not turn
// into a wall of names. The census view (show == 0) is where the full list
// lives, and that difference is the point of the two views.
func TestFamilyMemberListIsCappedOnlyWhenBounded(t *testing.T) {
	big := family.Family{MinEdge: 0.70}
	units := make([]parser.CodeUnit, 0, maxFamilyMembers+4)
	for i := 0; i < maxFamilyMembers+4; i++ {
		big.Members = append(big.Members, i)
		units = append(units, parser.CodeUnit{Name: string(rune('a' + i)), Package: "p", File: "p/x.go", StartLine: i})
	}
	fams := []family.Family{big}
	stats := family.Stats{Components: 1, Families: 1, Members: len(big.Members)}

	var bounded strings.Builder
	PrintFamilies(&bounded, fams, stats, units, 5)
	if n := strings.Count(bounded.String(), "p/x.go:"); n != maxFamilyMembers {
		t.Errorf("bounded listing showed %d members, want %d:\n%s", n, maxFamilyMembers, bounded.String())
	}
	if !strings.Contains(bounded.String(), "(4 more members not listed)") {
		t.Errorf("remainder not counted:\n%s", bounded.String())
	}

	var census strings.Builder
	PrintFamilies(&census, fams, stats, units, 0)
	if n := strings.Count(census.String(), "p/x.go:"); n != len(big.Members) {
		t.Errorf("census showed %d members, want all %d", n, len(big.Members))
	}
	if strings.Contains(census.String(), "more members not listed") {
		t.Errorf("the census truncated a family:\n%s", census.String())
	}
}

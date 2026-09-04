package reporter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/gofront"
	"github.com/LukasSelin/doppel/internal/parser"
)

// renderedSource re-derives a fixture unit's canonical tree with renders,
// the way cmd does for a Go unit, and checks the re-derivation carries the
// unit's bag before handing it over.
func renderedSource(t *testing.T, u parser.CodeUnit) LabelSource {
	t.Helper()
	tree, renders, err := gofront.CanonicalRenders("p.go", []byte(fingerprintFixture), u.StartLine, parser.MethodName(u))
	if err != nil {
		t.Fatalf("CanonicalRenders: %v", err)
	}
	got, want := fingerprint.WLBagOf(tree), u.Fingerprint.WL
	if len(got) != len(want) {
		t.Fatalf("re-derived bag has %d labels, unit's has %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("re-derived bag differs at %d", i)
		}
	}
	return LabelSource{Tree: tree, Renders: renders}
}

// labelOfKind picks a label from the unit's bag by round and kind, so a test
// can ask for "the depth-3 RANGE" without hardcoding a hash.
func labelOfKind(t *testing.T, u parser.CodeUnit, h uint8, kind fingerprint.LabelKind) uint64 {
	t.Helper()
	for _, lc := range u.Fingerprint.WL {
		if lc.H == h && lc.Kind == kind {
			return lc.Label
		}
	}
	t.Fatalf("no %s in %s's bag", fingerprint.DescribeLabel(h, kind), u.Name)
	return 0
}

func TestPrintLabelOccurrencesRendersEachHit(t *testing.T) {
	units, idf := fingerprintUnits(t)
	u := unitNamed(t, units, "Sum")
	src := renderedSource(t, u)
	label := labelOfKind(t, u, 3, fingerprint.KindRange)
	var out bytes.Buffer
	PrintLabelOccurrences(&out, u, src, []uint64{label}, idf, FingerprintMeta{})
	s := out.String()
	for _, want := range []string{
		"labels in p.Sum (code shown is the canonical form",
		"depth-3 RANGE  df ",
		"count 1",
		"node ",
		": range, subtree ",
		"for _, x",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
	if strings.Contains(s, "hashed extent") && !strings.Contains(s, "folds 3 of those") {
		t.Errorf("an outline printed without the render overstating the extent:\n%s", s)
	}
	// One occurrence for a count-1 label.
	if n := strings.Count(s, "\n    node "); n != 1 {
		t.Errorf("expected exactly one occurrence line, got %d:\n%s", n, s)
	}
}

// TestPrintLabelOccurrencesCountMatchesRows: for every label in the bag, the
// number of occurrence lines (uncapped) equals the bag's count — the view
// can never disagree with the count column above it.
func TestPrintLabelOccurrencesCountMatchesRows(t *testing.T) {
	units, idf := fingerprintUnits(t)
	u := unitNamed(t, units, "Other")
	src := renderedSource(t, u)
	for _, lc := range u.Fingerprint.WL {
		var out bytes.Buffer
		PrintLabelOccurrences(&out, u, src, []uint64{lc.Label}, idf, FingerprintMeta{})
		if n := strings.Count(out.String(), "\n    node "); n != int(lc.Count) {
			t.Errorf("label %x: %d occurrence lines, bag count %d", lc.Label, n, lc.Count)
		}
	}
}

func TestPrintLabelOccurrencesLabelTopCaps(t *testing.T) {
	units, idf := fingerprintUnits(t)
	u := unitNamed(t, units, "Sum")
	label := labelOfKind(t, u, 0, fingerprint.KindIdent) // on every identifier
	var out bytes.Buffer
	PrintLabelOccurrences(&out, u, LabelSource{Tree: u.Canonical}, []uint64{label}, idf, FingerprintMeta{LabelTop: 2})
	s := out.String()
	if n := strings.Count(s, "\n    node "); n != 2 {
		t.Errorf("cap of 2 printed %d occurrences:\n%s", n, s)
	}
	if !strings.Contains(s, "… and ") {
		t.Errorf("the rest must be counted:\n%s", s)
	}
}

func TestPrintLabelOccurrencesOutlineFallback(t *testing.T) {
	units, idf := fingerprintUnits(t)
	u := unitNamed(t, units, "Sum")
	label := labelOfKind(t, u, 2, fingerprint.KindIf)
	var out bytes.Buffer
	PrintLabelOccurrences(&out, u, LabelSource{Tree: u.Canonical}, []uint64{label}, idf, FingerprintMeta{})
	s := out.String()
	if strings.Contains(s, "canonical form") {
		t.Errorf("no renders, so no canonical-form caveat:\n%s", s)
	}
	for _, want := range []string{"hashed extent (2 levels", "      IF\n", "        BIN/>"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestPrintLabelOccurrencesLeafCoversAll(t *testing.T) {
	units, idf := fingerprintUnits(t)
	u := unitNamed(t, units, "Sum")
	label := labelOfKind(t, u, 3, fingerprint.KindIdent)
	var out bytes.Buffer
	PrintLabelOccurrences(&out, u, renderedSource(t, u), []uint64{label}, idf, FingerprintMeta{LabelTop: 1})
	if !strings.Contains(out.String(), "depth-3 covers all of it") {
		t.Errorf("a leaf's deep label covers its whole subtree:\n%s", out.String())
	}
	if strings.Contains(out.String(), "hashed extent") {
		t.Errorf("a rendered leaf needs no outline:\n%s", out.String())
	}
}

func TestPrintLabelOccurrencesAbsent(t *testing.T) {
	units, idf := fingerprintUnits(t)
	u := unitNamed(t, units, "Sum")
	other := labelOfKind(t, unitNamed(t, units, "Other"), 3, fingerprint.KindDefer)
	var out bytes.Buffer
	PrintLabelOccurrences(&out, u, LabelSource{Tree: u.Canonical}, []uint64{other, 0xdead}, idf, FingerprintMeta{})
	s := out.String()
	if !strings.Contains(s, "not in this function's bag (df 1: carried elsewhere in the corpus)") {
		t.Errorf("a label another function carries must say so:\n%s", s)
	}
	if !strings.Contains(s, "label #dead: not in this function's bag (df -: unknown to the corpus)") {
		t.Errorf("an unknown label must say so:\n%s", s)
	}
}

func TestPrintLabelOccurrencesNoBody(t *testing.T) {
	_, idf := fingerprintUnits(t)
	var out bytes.Buffer
	PrintLabelOccurrences(&out, parser.CodeUnit{Name: "Ext", Package: "p"}, LabelSource{}, []uint64{1}, idf, FingerprintMeta{})
	if !strings.Contains(out.String(), "no body: nothing carries a label") {
		t.Errorf("no body must be stated:\n%s", out.String())
	}
}

func TestPrintLabelOccurrencesPair(t *testing.T) {
	units, idf := fingerprintUnits(t)
	a, b := unitNamed(t, units, "Sum"), unitNamed(t, units, "Other")
	shared := labelOfKind(t, a, 0, fingerprint.KindIf)
	onlyB := labelOfKind(t, b, 3, fingerprint.KindDefer)
	var out bytes.Buffer
	PrintLabelOccurrencesPair(&out, a, b, renderedSource(t, a), renderedSource(t, b), []uint64{shared, onlyB}, idf, FingerprintMeta{})
	s := out.String()
	for _, want := range []string{
		"labels in A p.Sum and B p.Other",
		"depth-0 IF  df ",
		"A ×1  B ×1",
		"A absent  B ×1",
		"    A p.Sum\n",
		"    B p.Other\n",
		"      absent\n",
		"defer fmt.Println(",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestPrintLabelOccurrencesDeterministic(t *testing.T) {
	units, idf := fingerprintUnits(t)
	u := unitNamed(t, units, "Sum")
	src := renderedSource(t, u)
	labels := []uint64{labelOfKind(t, u, 0, fingerprint.KindIdent), labelOfKind(t, u, 3, fingerprint.KindRange)}
	var one, two bytes.Buffer
	PrintLabelOccurrences(&one, u, src, labels, idf, FingerprintMeta{})
	PrintLabelOccurrences(&two, u, src, labels, idf, FingerprintMeta{})
	if one.String() != two.String() {
		t.Fatal("two renders differ")
	}
}

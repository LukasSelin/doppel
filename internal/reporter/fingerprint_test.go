package reporter

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

// fingerprintFixture is a small corpus with a clone pair inside it: Sum and
// Total are the same loop under different names, Other is unrelated, and
// the filler functions give the label weights a population to be counted
// over — on a two-function corpus every label has df == N and weighs 0, the
// trap CLAUDE.md records under the timeline's fixture notes.
const fingerprintFixture = `package p

import "fmt"

func Sum(xs []int) int {
	total := 0
	for _, x := range xs {
		if x > 0 {
			total += x
		}
	}
	return total
}

func Total(vals []int) int {
	acc := 0
	for _, v := range vals {
		if v > 0 {
			acc += v
		}
	}
	return acc
}

func Other(s string) error {
	if s == "" {
		return fmt.Errorf("empty")
	}
	defer fmt.Println("done")
	return nil
}

func A() { fmt.Println("a") }
func B() { fmt.Println("b", 1) }
func C() int { return 1 }
func D(x int) int { return x * 2 }
`

func fingerprintUnits(t *testing.T) ([]parser.CodeUnit, *fingerprint.LabelIDF) {
	t.Helper()
	units, err := parser.ParseSource("p.go", []byte(fingerprintFixture))
	if err != nil || len(units) < 3 {
		t.Fatalf("fixture: units=%d err=%v", len(units), err)
	}
	bags := make([][]fingerprint.LabelCount, len(units))
	for i := range units {
		bags[i] = units[i].Fingerprint.WL
	}
	return units, fingerprint.LabelWeights(bags)
}

func unitNamed(t *testing.T, units []parser.CodeUnit, name string) parser.CodeUnit {
	t.Helper()
	for _, u := range units {
		if u.Name == name {
			return u
		}
	}
	t.Fatalf("no unit %q in fixture", name)
	return parser.CodeUnit{}
}

func TestPrintFingerprintRendersEveryComponent(t *testing.T) {
	units, idf := fingerprintUnits(t)
	var out bytes.Buffer
	PrintFingerprint(&out, unitNamed(t, units, "Other"), idf, FingerprintMeta{})
	s := out.String()
	for _, want := range []string{
		"fingerprint: p.Other  p.go:",
		"sig: (string) (error)",
		"nodes: ",
		"canonicalized: ",
		"flow: if 1  return 2  defer 1",
		"nesting: depth 0: 3  depth 1: 1",
		"types: in:string out:error",
		"weights ln(N/df), N = 7 bodies",
		"depth-0 IF",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
	if strings.Contains(s, "… and") {
		t.Errorf("LabelTop 0 must print every row, got a truncation line:\n%s", s)
	}
}

func TestPrintFingerprintLabelTopTruncatesAndCounts(t *testing.T) {
	units, idf := fingerprintUnits(t)
	u := unitNamed(t, units, "Sum")
	var out bytes.Buffer
	PrintFingerprint(&out, u, idf, FingerprintMeta{LabelTop: 3})
	s := out.String()
	rest := len(u.Fingerprint.WL) - 3
	if rest <= 0 {
		t.Fatalf("fixture bag too small to truncate: %d labels", len(u.Fingerprint.WL))
	}
	if !strings.Contains(s, "… and "+itoa(rest)+" more") {
		t.Errorf("expected %d rows to be counted rather than printed:\n%s", rest, s)
	}
}

func TestPrintFingerprintNoCorpusSaysSo(t *testing.T) {
	units, _ := fingerprintUnits(t)
	var out bytes.Buffer
	PrintFingerprint(&out, unitNamed(t, units, "Sum"), nil, FingerprintMeta{})
	if !strings.Contains(out.String(), "no corpus: every label weighs 1.0") {
		t.Errorf("a nil idf must be stated, not passed off as a corpus figure:\n%s", out.String())
	}
	// With uniform weights, mass is the plain count.
	u := unitNamed(t, units, "Sum")
	total := 0
	for _, lc := range u.Fingerprint.WL {
		total += int(lc.Count)
	}
	if !strings.Contains(out.String(), itoa(total)+" total, "+itoa(total)+".0 nats") {
		t.Errorf("uniform weights should make mass equal the label count %d:\n%s", total, out.String())
	}
}

func TestPrintFingerprintNoBody(t *testing.T) {
	var out bytes.Buffer
	PrintFingerprint(&out, parser.CodeUnit{Name: "Ext", Package: "p"}, nil, FingerprintMeta{})
	if !strings.Contains(out.String(), "no body") {
		t.Errorf("zero fingerprint must be named as such:\n%s", out.String())
	}
}

// TestFingerprintPairMergeReproducesTheScore is the view's own gate: the
// three partition masses it prints are the same three sums wlOverlap
// accumulates, so the Jaccard and containment a reader can compute from the
// page must equal the ones the score reported, exactly, on every pair of the
// fixture — including the pair whose shared mass is zero.
func TestFingerprintPairMergeReproducesTheScore(t *testing.T) {
	units, idf := fingerprintUnits(t)
	pairs := 0
	for i := range units {
		for j := i + 1; j < len(units); j++ {
			a, b := units[i].Fingerprint.WL, units[j].Fingerprint.WL
			wantJ, wantC := fingerprint.WLOverlap(a, b, idf)
			m := mergeBags(a, b, idf)
			gotJ := ratioOrZero(m.shared, m.massA+m.massB-m.shared)
			gotC := ratioOrZero(m.shared, math.Min(m.massA, m.massB))
			if math.Abs(gotJ-wantJ) > 1e-12 || math.Abs(gotC-wantC) > 1e-12 {
				t.Errorf("%s/%s: merge gives jaccard %.12f containment %.12f, score gives %.12f %.12f",
					units[i].Name, units[j].Name, gotJ, gotC, wantJ, wantC)
			}
			// The sections must add back into the side masses: shared plus a
			// side's surplus is that side's whole bag.
			var sh, oa, ob float64
			for _, r := range m.sharedRows {
				sh += r.mass
			}
			for _, r := range m.onlyA {
				oa += r.mass
			}
			for _, r := range m.onlyB {
				ob += r.mass
			}
			if math.Abs(sh+oa-m.massA) > 1e-9 || math.Abs(sh+ob-m.massB) > 1e-9 {
				t.Errorf("%s/%s: sections do not add up: shared %.6f + onlyA %.6f != A %.6f, or + onlyB %.6f != B %.6f",
					units[i].Name, units[j].Name, sh, oa, m.massA, ob, m.massB)
			}
			pairs++
		}
	}
	if pairs < 10 {
		t.Fatalf("only %d pairs checked; the fixture shrank", pairs)
	}
}

func TestPrintFingerprintPairShowsTheBlend(t *testing.T) {
	units, idf := fingerprintUnits(t)
	a, b := unitNamed(t, units, "Sum"), unitNamed(t, units, "Total")
	var out bytes.Buffer
	PrintFingerprintPair(&out, a, b, idf, FingerprintMeta{LabelTop: 4})
	s := out.String()
	bd := fingerprint.Similarity(a.Fingerprint, b.Fingerprint, idf)
	if bd.Score != 1.0 {
		t.Fatalf("fixture clones score %.4f, want 1.0 — canon or the fixture moved", bd.Score)
	}
	for _, want := range []string{
		"A: p.Sum  p.go:",
		"B: p.Total  p.go:",
		"code-shape: 1.0000",
		"wl       1.0000 × 0.60 = 0.6000",
		"flow     1.0000 × 0.20 = 0.2000",
		"nesting  1.0000 × 0.05 = 0.0500",
		"sig      1.0000 × 0.15 = 0.1500",
		"jaccard = shared / union = 1.0000",
		"containment = shared / min(A, B) = 1.0000",
		"only in A: 0 labels, 0.0 nats",
		"only in B: 0 labels, 0.0 nats",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestPrintFingerprintPairUnrelatedPartitions(t *testing.T) {
	units, idf := fingerprintUnits(t)
	a, b := unitNamed(t, units, "Sum"), unitNamed(t, units, "Other")
	var out bytes.Buffer
	PrintFingerprintPair(&out, a, b, idf, FingerprintMeta{})
	s := out.String()
	if !strings.Contains(s, "only in A: ") || !strings.Contains(s, "only in B: ") || !strings.Contains(s, "shared: ") {
		t.Errorf("all three partitions must render:\n%s", s)
	}
	if !strings.Contains(s, "sig      0.0000") {
		t.Errorf("disjoint type sets must show a 0 signature component:\n%s", s)
	}
}

func TestPrintFingerprintPairNoBody(t *testing.T) {
	units, idf := fingerprintUnits(t)
	var out bytes.Buffer
	PrintFingerprintPair(&out, unitNamed(t, units, "Sum"), parser.CodeUnit{Name: "Ext", Package: "p"}, idf, FingerprintMeta{})
	if !strings.Contains(out.String(), "no body on one side") {
		t.Errorf("a zero fingerprint on either side must stop the merge:\n%s", out.String())
	}
}

func TestFingerprintViewDeterministic(t *testing.T) {
	units, idf := fingerprintUnits(t)
	a, b := unitNamed(t, units, "Sum"), unitNamed(t, units, "Other")
	var one, two bytes.Buffer
	PrintFingerprintPair(&one, a, b, idf, FingerprintMeta{})
	PrintFingerprintPair(&two, a, b, idf, FingerprintMeta{})
	if one.String() != two.String() {
		t.Fatal("two renders of one pair differ")
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

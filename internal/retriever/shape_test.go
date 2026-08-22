package retriever

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// trivialCloneSource builds twelve identical (modulo receiver) Error-style
// methods plus two genuine renamed-copy pairs of non-trivial functions.
func trivialCloneSource() string {
	var b strings.Builder
	b.WriteString("package fix\n\nimport \"fmt\"\n")
	for i := 0; i < 12; i++ {
		fmt.Fprintf(&b, `
type T%d struct {
	msg  string
	code int
}

func (e T%d) Error() string {
	return fmt.Sprintf("boom %%s %%d", e.msg, e.code)
}
`, i, i)
	}
	b.WriteString(`
func SumEvens(xs []int) int {
	total := 0
	for _, x := range xs {
		if x%2 == 0 {
			total += x
		}
	}
	return total
}

func TotalEvens(vals []int) int {
	sum := 0
	for _, v := range vals {
		if v%2 == 0 {
			sum += v
		}
	}
	return sum
}

func FirstNegative(xs []int) (int, bool) {
	for i, x := range xs {
		if x < 0 {
			return i, true
		}
		if x == 0 {
			break
		}
	}
	return 0, false
}

func LeadingNegative(vals []int) (int, bool) {
	for j, v := range vals {
		if v < 0 {
			return j, true
		}
		if v == 0 {
			break
		}
	}
	return 0, false
}
`)
	return b.String()
}

// Requirement 1: many identical tiny shapes must not dominate the candidate
// list. With a df cap of 8, every shingle of the 12 Error clones is capped
// out — they are fully suppressed from the structural channel — while the two
// real pairs retrieve each other.
func TestShapeChannelSuppressesTrivialCloneBuckets(t *testing.T) {
	units := parseUnits(t, "fix.go", trivialCloneSource())
	opt := DefaultOptions()
	opt.MinNodes = 8
	opt.MaxPatternDF = 8
	// The clones also share fmt.Sprintf (df=12 in this 16-unit fixture); cap
	// it out so this test isolates the structural channel.
	opt.MaxCallDF = 3

	cands, stats := retrieveAll(t, units, opt)

	isClone := func(i int) bool { return strings.HasSuffix(units[i].Name, ".Error") }
	for _, c := range cands {
		if isClone(c.AIdx) && isClone(c.BIdx) {
			t.Errorf("clone pair %s/%s retrieved; common idiom bucket should be suppressed",
				units[c.AIdx].Name, units[c.BIdx].Name)
		}
	}
	if stats.Suppressed != 12 {
		t.Errorf("Suppressed = %d, want 12 (every Error clone)", stats.Suppressed)
	}

	for _, pair := range [][2]string{{"SumEvens", "TotalEvens"}, {"FirstNegative", "LeadingNegative"}} {
		a, b := unitIndex(t, units, pair[0]), unitIndex(t, units, pair[1])
		c, ok := findCandidate(cands, a, b)
		if !ok {
			t.Errorf("real pair %s/%s not retrieved", pair[0], pair[1])
			continue
		}
		if c.Shape <= 0 {
			t.Errorf("real pair %s/%s has zero shape evidence", pair[0], pair[1])
		}
	}
}

// Requirement 2: a rare structural match must carry more evidence than an
// equally similar ubiquitous shape. Both pairs are exact renamed copies
// (fingerprint score 1.0); the common pair's shape also lives in six filler
// clones, so its shingles have df=8 against the rare pair's df=2.
func TestShapeChannelRanksRareShapeAboveCommonShape(t *testing.T) {
	var b strings.Builder
	b.WriteString("package fix\n")
	// The rare pair: a distinctive shape appearing exactly twice.
	b.WriteString(`
func RareMerge(a, b []int) []int {
	var out []int
	for len(a) > 0 && len(b) > 0 {
		if a[0] < b[0] {
			out = append(out, a[0])
			a = a[1:]
		} else {
			out = append(out, b[0])
			b = b[1:]
		}
	}
	return append(out, append(a, b...)...)
}

func RareWeave(xs, ys []int) []int {
	var res []int
	for len(xs) > 0 && len(ys) > 0 {
		if xs[0] < ys[0] {
			res = append(res, xs[0])
			xs = xs[1:]
		} else {
			res = append(res, ys[0])
			ys = ys[1:]
		}
	}
	return append(res, append(xs, ys...)...)
}
`)
	// The common pair plus six fillers with the identical shape.
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&b, `
func Common%d(xs []int) int {
	total := 0
	for _, x := range xs {
		if x%%2 == 0 {
			total += x
		}
	}
	return total
}
`, i)
	}
	units := parseUnits(t, "fix.go", b.String())
	opt := DefaultOptions()
	opt.MinNodes = 8
	opt.MaxPatternDF = 20

	cands, _ := retrieveAll(t, units, opt)

	rare, ok := findCandidate(cands, unitIndex(t, units, "RareMerge"), unitIndex(t, units, "RareWeave"))
	if !ok {
		t.Fatal("rare pair not retrieved")
	}
	common, ok := findCandidate(cands, unitIndex(t, units, "Common0"), unitIndex(t, units, "Common1"))
	if !ok {
		t.Fatal("common pair not retrieved")
	}
	if rare.Breakdown.Score < 0.95 || common.Breakdown.Score < 0.95 {
		t.Fatalf("fixture broken: both pairs should be near-exact copies, got %.2f and %.2f",
			rare.Breakdown.Score, common.Breakdown.Score)
	}
	if rare.Shape <= common.Shape {
		t.Errorf("rare shape evidence %.3f should exceed common shape evidence %.3f",
			rare.Shape, common.Shape)
	}
	if rare.Total <= common.Total {
		t.Errorf("rare pair total %.3f should outrank common pair total %.3f", rare.Total, common.Total)
	}
}

func TestEffectiveCap(t *testing.T) {
	if c, d := effectiveCap(50, 1000, 0); c != 50 || d {
		t.Errorf("floor off: got %d derived=%v, want absolute 50", c, d)
	}
	// ln(1000/50) = 2.9957…: a floor just under it reproduces the cap.
	if c, _ := effectiveCap(50, 1000, math.Log(1000.0/50)-1e-9); c != 50 {
		t.Errorf("floor at ln(N/50) should derive 50, got %d", c)
	}
	if c, d := effectiveCap(50, 81, 2.0); !d || c != int(math.Floor(81*math.Exp(-2))) {
		t.Errorf("derived cap on 81 units at 2 nats = %d (derived %v)", c, d)
	}
	// Honest emptiness: a floor no df>=2 feature can carry derives < 2.
	if c, _ := effectiveCap(50, 10, 3.0); c >= 2 {
		t.Errorf("10 units at 3 nats should derive an empty channel, got cap %d", c)
	}
}

// A MinIDF set to exactly the absolute caps' information level reproduces
// the fixed-cap retrieval: same candidates, same masses.
func TestMinIDFReproducesAbsoluteCaps(t *testing.T) {
	var b strings.Builder
	b.WriteString("package fix\n")
	b.WriteString(strings.ReplaceAll(trophicScanLoop, "ReadIDs", "ReadA"))
	b.WriteString(strings.NewReplacer(
		"ReadIDs", "ReadB", "path", "fname", "file", "fh",
		"ids", "nums", "scanner", "sc", "line", "text", "id", "n",
	).Replace(trophicScanLoop))
	b.WriteString(`
func Filler(m map[string]int) string {
	best := ""
	for k, v := range m {
		switch {
		case v > 10:
			best = k
		case k == "":
			best = "none"
		}
	}
	return best
}
`)
	units := parseUnits(t, "fix.go", b.String())
	fixed := DefaultOptions()
	fixed.MinNodes = 8
	fixed.MaxPatternDF, fixed.MaxCallDF = 2, 2
	cFixed, sFixed := retrieveAll(t, units, fixed)

	// Three eligible units, three units: ln(3/2) is the information of df 2.
	floored := fixed
	floored.MinIDF = math.Log(3.0/2) - 1e-9
	cFloor, sFloor := retrieveAll(t, units, floored)
	if !sFloor.CapsDerived || sFloor.PatternCap != 2 || sFloor.CallCap != 2 {
		t.Fatalf("derived caps = %d/%d (derived %v), want 2/2", sFloor.PatternCap, sFloor.CallCap, sFloor.CapsDerived)
	}
	if len(cFixed) != len(cFloor) {
		t.Fatalf("candidate count differs: fixed %d, floored %d", len(cFixed), len(cFloor))
	}
	for i := range cFixed {
		if cFixed[i].AIdx != cFloor[i].AIdx || cFixed[i].BIdx != cFloor[i].BIdx || cFixed[i].Total != cFloor[i].Total {
			t.Fatalf("candidate %d differs: %+v vs %+v", i, cFixed[i], cFloor[i])
		}
	}
	if sFixed.Suppressed != sFloor.Suppressed || sFixed.SurvivingPatterns != sFloor.SurvivingPatterns {
		t.Errorf("stats differ: %+v vs %+v", sFixed, sFloor)
	}
}

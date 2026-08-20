package retriever

import (
	"fmt"
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
	opt.MaxShingleDF = 8
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
	opt.MaxShingleDF = 20

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

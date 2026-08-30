package retriever

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const trophicScanLoop = `
func ReadIDs(path string) ([]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var ids []int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		id, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, scanner.Err()
}
`

// The essay's negative case (DataSourceName ↔ Error): structurally identical
// Sprintf-return twins whose entire shape is a corpus idiom. With the idiom
// df-capped, their shared structure earns zero credit while their total
// energy stays real, so trophic reads ~0 despite ast 1.00. The pair reaches
// the union via the concept channel (shared tag), exactly how such trivia
// would surface in practice.
func TestTrophicLowForUbiquitousPatternTwins(t *testing.T) {
	var b strings.Builder
	b.WriteString("package fix\n\nimport \"fmt\"\n")
	for i := 0; i < 8; i++ {
		fmt.Fprintf(&b, `
type T%d struct {
	msg  string
	code int
}

func (e T%d) Label() string {
	return fmt.Sprintf("v %%s %%d", e.msg, e.code)
}
`, i, i)
	}
	// Two structurally rich functions so the corpus has real energy too.
	b.WriteString(strings.ReplaceAll(trophicScanLoop, "ReadIDs", "ReadA"))
	b.WriteString(strings.NewReplacer(
		"ReadIDs", "ReadB", "path", "fname", "file", "fh",
		"ids", "nums", "scanner", "sc", "line", "text", "id", "n",
	).Replace(trophicScanLoop))

	units := parseUnits(t, "fix.go", b.String())
	a := unitIndex(t, units, "T0.Label")
	bIdx := unitIndex(t, units, "T1.Label")
	units[a].Patterns = []string{"mapping"}
	units[bIdx].Patterns = []string{"mapping"}

	opt := DefaultOptions()
	opt.MinNodes = 8
	opt.MaxPatternDF = 6 // clone patterns have df=8: fully capped idiom
	opt.MaxCallDF = 3    // fmt.Sprintf df=8 capped out of the call channel

	cands, _ := retrieveAll(t, units, opt)

	twin, ok := findCandidate(cands, a, bIdx)
	if !ok {
		t.Fatal("tagged twins not retrieved via the concept channel")
	}
	if twin.Breakdown.Score < 0.99 {
		t.Fatalf("fixture broken: twins should be near-identical, got %.2f", twin.Breakdown.Score)
	}
	if twin.Shape != 0 {
		t.Errorf("idiom twins shape energy = %.3f, want 0 (fully capped)", twin.Shape)
	}
	if twin.TrophicSim >= 0.3 {
		t.Errorf("ubiquitous twins trophic = %.3f, want < 0.3 despite wl %.2f",
			twin.TrophicSim, twin.Breakdown.WL)
	}
}

// The positive case: renamed scan-loop copies share the entire informative
// chain — trophic near 1.0 and the explanation names the loop motif.
func TestTrophicHighForSharedChain(t *testing.T) {
	var b strings.Builder
	b.WriteString("package fix\n")
	b.WriteString(strings.ReplaceAll(trophicScanLoop, "ReadIDs", "ReadA"))
	b.WriteString(strings.NewReplacer(
		"ReadIDs", "ReadB", "path", "fname", "file", "fh",
		"ids", "nums", "scanner", "sc", "line", "text", "id", "n",
	).Replace(trophicScanLoop))
	// Unrelated filler so pattern df < nEligible and idf > 0.
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
	opt := DefaultOptions()
	opt.MinNodes = 8
	// This three-function corpus is degenerate: every shared pattern has the
	// same idf, so the L4 flow edges (count 2) tie or beat the loop motif
	// (count 1) on energy and would crowd it out of the default top 3. On a
	// real corpus idf separates them; here, widen the cap so the assertion
	// tests presence in the explanation, not victory in a rigged tie.
	opt.ChainTopN = 10

	cands, _ := retrieveAll(t, units, opt)
	pair, ok := findCandidate(cands, unitIndex(t, units, "ReadA"), unitIndex(t, units, "ReadB"))
	if !ok {
		t.Fatal("renamed scan-loop pair not retrieved")
	}
	if pair.TrophicSim < 0.99 {
		t.Errorf("exact renamed clones trophic = %.3f, want ~1.0", pair.TrophicSim)
	}
	if len(pair.Chains) == 0 {
		t.Fatal("no shared-chain explanation")
	}
	foundLoop := false
	for _, ch := range pair.Chains {
		if strings.HasPrefix(ch.Render, "for{ call:Scan") {
			foundLoop = true
		}
	}
	if !foundLoop {
		t.Errorf("chains %+v do not include the scan-loop motif", pair.Chains)
	}
	for i := 1; i < len(pair.Chains); i++ {
		if pair.Chains[i-1].Energy < pair.Chains[i].Energy {
			t.Errorf("chains not in energy-descending order: %+v", pair.Chains)
		}
	}
}

// ChainTopN caps the explanation list, and retrieval stays deterministic
// with the trophic fields included.
func TestSharedChainsCapAndDeterminism(t *testing.T) {
	var b strings.Builder
	b.WriteString("package fix\n")
	b.WriteString(strings.ReplaceAll(trophicScanLoop, "ReadIDs", "ReadA"))
	b.WriteString(strings.NewReplacer(
		"ReadIDs", "ReadB", "path", "fname", "file", "fh",
		"ids", "nums", "scanner", "sc", "line", "text", "id", "n",
	).Replace(trophicScanLoop))
	units := parseUnits(t, "fix.go", b.String())

	opt := DefaultOptions()
	opt.MinNodes = 8
	opt.ChainTopN = 1

	first, firstStats := retrieveAll(t, units, opt)
	for _, c := range first {
		if len(c.Chains) > 1 {
			t.Errorf("ChainTopN=1 but pair (%d,%d) carries %d chains", c.AIdx, c.BIdx, len(c.Chains))
		}
	}
	for i := 0; i < 25; i++ {
		cands, stats := retrieveAll(t, units, opt)
		if !reflect.DeepEqual(cands, first) || stats != firstStats {
			t.Fatalf("run %d diverged", i)
		}
	}
}

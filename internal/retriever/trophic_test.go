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

// trophicFiller is an unrelated third function, so a label carried by the two
// clones has df 2 out of three eligible units rather than df = nEligible.
// ln(N/df) is 0 when a label is universal, so a two-function corpus has no
// informative structure at all by the channel's own arithmetic.
const trophicFiller = `
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
	opt.MaxLabelDF = 6 // clone labels have df=8: fully capped idiom
	opt.MaxCallDF = 3  // fmt.Sprintf df=8 capped out of the call channel

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
// structure — trophic near 1.0, and the explanation reaches the loop.
func TestTrophicHighForSharedChain(t *testing.T) {
	var b strings.Builder
	b.WriteString("package fix\n")
	b.WriteString(strings.ReplaceAll(trophicScanLoop, "ReadIDs", "ReadA"))
	b.WriteString(strings.NewReplacer(
		"ReadIDs", "ReadB", "path", "fname", "file", "fh",
		"ids", "nums", "scanner", "sc", "line", "text", "id", "n",
	).Replace(trophicScanLoop))
	// Unrelated filler so label df < nEligible and idf > 0.
	b.WriteString(trophicFiller)
	units := parseUnits(t, "fix.go", b.String())
	opt := DefaultOptions()
	opt.MinNodes = 8
	// This three-function corpus is degenerate: every shared label has the
	// same idf, so scores of them tie on energy and the loop's own label
	// would be crowded out of the default top 3 by an arbitrary one of them.
	// On a real corpus idf separates them; here, lift the cap entirely so
	// the assertion tests presence in the explanation, not victory in a
	// rigged tie.
	opt.ChainTopN = -1

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
	// The pattern channel named the loop outright ("for{ call:Scan ... }").
	// A WL label has no such render, so the migrated assertion is the
	// strongest equivalent claim the vocabulary can make: the loop node
	// itself is shared, at a round deep enough to have folded its body in.
	// Requiring h >= 1 is what makes it a claim about *this* loop rather
	// than about the bare existence of a `for` keyword — at h=0 every
	// condition-only for loop in Go carries the same label.
	foundLoop := false
	for _, ch := range pair.Chains {
		if ch.Depth >= 1 && strings.HasSuffix(ch.Render, " FOR") {
			foundLoop = true
		}
	}
	if !foundLoop {
		t.Errorf("chains %+v do not include the scan loop at any depth", pair.Chains)
	}
	for i := 1; i < len(pair.Chains); i++ {
		if pair.Chains[i-1].Energy < pair.Chains[i].Energy {
			t.Errorf("chains not in energy-descending order: %+v", pair.Chains)
		}
	}
}

// TestSharedLabelsAreSelfDescribing pins what makes the shared-structure
// block checkable at all: every entry names a real refinement round, carries
// the multiplicity its energy was computed from, and renders as exactly the
// package-level description of its own (round, kind). A render composed
// anywhere else could drift from the label it claims to name.
func TestSharedLabelsAreSelfDescribing(t *testing.T) {
	var b strings.Builder
	b.WriteString("package fix\n")
	b.WriteString(strings.ReplaceAll(trophicScanLoop, "ReadIDs", "ReadA"))
	b.WriteString(strings.NewReplacer(
		"ReadIDs", "ReadB", "path", "fname", "file", "fh",
		"ids", "nums", "scanner", "sc", "line", "text", "id", "n",
	).Replace(trophicScanLoop))
	// Unrelated filler, so a label's df is below nEligible and its idf above
	// zero. With only the two clones in the corpus every label has
	// df = nEligible and ln(1) = 0 energy, and the block is empty by the
	// channel's own rule rather than by a defect.
	b.WriteString(trophicFiller)
	units := parseUnits(t, "fix.go", b.String())

	opt := DefaultOptions()
	opt.MinNodes = 8
	opt.ChainTopN = -1

	cands, _ := retrieveAll(t, units, opt)
	seen := 0
	for _, c := range cands {
		for _, ch := range c.Chains {
			seen++
			if ch.Depth < 0 || ch.Depth > 3 {
				t.Errorf("chain %+v has an impossible round", ch)
			}
			if ch.Count < 1 {
				t.Errorf("chain %+v claims a multiplicity below 1", ch)
			}
			if ch.Energy <= 0 {
				t.Errorf("chain %+v has no energy and should not be listed", ch)
			}
			if ch.Label == 0 {
				t.Errorf("chain %+v carries no label identity", ch)
			}
			if !strings.HasPrefix(ch.Render, "depth-") {
				t.Errorf("chain render %q is not a label description", ch.Render)
			}
		}
	}
	if seen == 0 {
		t.Fatal("no shared labels at all: the fixture no longer exercises the block")
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

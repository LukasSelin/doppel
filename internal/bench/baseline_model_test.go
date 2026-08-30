package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// classOrder is the fixed class ordering logScorecard already prints in, used
// here too so a CorpusBaseline's Classes slice never depends on Scorecard's
// map iteration order.
var classOrder = []string{"merge", "refactor", "false_positive"}

// ClassScore is one label class's aggregate outcome against a ranking — the
// two parallel maps Scorecard.MeanRank/Present carry, flattened into one
// sorted-by-declaration-order slice so the JSON baseline has a single
// deterministic shape.
type ClassScore struct {
	Class    string  `json:"class"`
	Present  int     `json:"present"`
	MeanRank float64 `json:"meanRank"`
}

// ViolationCounts is the size of each hard-assertion violation list — counts,
// not the pair descriptions scoreLabels' own failure messages carry, so the
// baseline stays a number to beat rather than a restatement of those
// messages.
type ViolationCounts struct {
	MergeMissing int `json:"mergeMissing"`
	FPAboveMerge int `json:"fpAboveMerge"`
	FPInTop20    int `json:"fpInTop20"`
}

// CorpusBaseline is one labeled corpus's golden scorecard, reduced to the
// numbers examples/README.md quotes and the ones TestGoldenCorpora's hard
// assertions gate.
type CorpusBaseline struct {
	Corpus           string          `json:"corpus"`
	Population       string          `json:"population"`
	Functions        int             `json:"functions"`
	Ranked           int             `json:"ranked"`
	Suppressed       int             `json:"suppressed"`
	Classes          []ClassScore    `json:"classes"`
	MergeTotal       int             `json:"mergeTotal"`
	MergePresent     int             `json:"mergePresent"`
	MergeInTop50     int             `json:"mergeInTop50"`
	Violations       ViolationCounts `json:"violations"`
	AssertionsPassed bool            `json:"assertionsPassed"`
}

// ExampleChecksum is one committed example report's content hash.
type ExampleChecksum struct {
	Corpus string `json:"corpus"`
	SHA256 string `json:"sha256"`
}

// Baseline is the T0 gate: a number to beat, taken from the golden-labels
// scorer (internal/bench's own reason-giving harness, not a re-derivation)
// plus a checksum of every regenerated example report. It excludes anything
// wall-clock or machine-dependent — see baseline-timings.json for that — so
// two runs on an unchanged tree produce byte-identical JSON.
type Baseline struct {
	Corpora  []CorpusBaseline  `json:"corpora"`
	Examples []ExampleChecksum `json:"examples"`
}

// classScoresOf flattens a Scorecard's two per-class maps into classOrder,
// keeping only the classes the labels file actually used — a class with zero
// present pairs never contributes a rank, so it is dropped rather than
// printed as meanRank 0, which would read as a real (and impossibly low) rank.
func classScoresOf(sc Scorecard) []ClassScore {
	var out []ClassScore
	for _, class := range classOrder {
		if sc.Present[class] == 0 {
			continue
		}
		out = append(out, ClassScore{Class: class, Present: sc.Present[class], MeanRank: sc.MeanRank[class]})
	}
	return out
}

// corpusBaselineOf reduces one labeled corpus's Scorecard to a CorpusBaseline.
func corpusBaselineOf(name string, lf LabelsFile, sc Scorecard) CorpusBaseline {
	return CorpusBaseline{
		Corpus:       name,
		Population:   lf.Population,
		Functions:    sc.Functions,
		Ranked:       sc.Ranked,
		Suppressed:   sc.Suppressed,
		Classes:      classScoresOf(sc),
		MergeTotal:   sc.MergeTotal,
		MergePresent: sc.MergePresent,
		MergeInTop50: sc.MergeInTop50,
		Violations: ViolationCounts{
			MergeMissing: len(sc.MergeMissing),
			FPAboveMerge: len(sc.FPAboveMerge),
			FPInTop20:    len(sc.FPInTop20),
		},
		AssertionsPassed: len(sc.MergeMissing) == 0 && len(sc.FPAboveMerge) == 0 && len(sc.FPInTop20) == 0,
	}
}

// sha256File hashes a file's exact bytes — the same bytes `task examples`
// wrote and git diffs against, so this checksum catches regeneration drift
// byte for byte, whitespace included.
func sha256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// exampleChecksums hashes every examples/<corpus>.md the ladder names that is
// present on disk, sorted by corpus name — independent of both Corpora's own
// ladder order and any filesystem directory order, so reordering the ladder
// or the directory listing cannot reorder this list.
func exampleChecksums(examplesDir string) ([]ExampleChecksum, error) {
	var out []ExampleChecksum
	for _, c := range Corpora {
		path := filepath.Join(examplesDir, c.Name+".md")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		sum, err := sha256File(path)
		if err != nil {
			return nil, err
		}
		out = append(out, ExampleChecksum{Corpus: c.Name, SHA256: sum})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Corpus < out[j].Corpus })
	return out, nil
}

// MarshalBaseline renders b the way examples/baseline.json is written:
// two-space indent and a single trailing newline, so `task baseline` run
// twice back to back on an unchanged tree diffs empty.
func MarshalBaseline(b Baseline) ([]byte, error) {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

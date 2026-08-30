package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
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

// ExampleChecksum is one example report's content hash, computed from a
// freshly regenerated report (buildExampleReport in examples_test.go) with
// its build-identity line normalized away — see normalizeForChecksum. It is
// never read off examples/<corpus>.md on disk: a baseline that depended on
// whatever happened to be committed there could report "unchanged" over a
// stale file, or "changed" over nothing but a rebuild.
type ExampleChecksum struct {
	Corpus string `json:"corpus"`
	SHA256 string `json:"sha256"`
}

// Baseline is the T0 gate: a number to beat, taken from the golden-labels
// scorer (internal/bench's own reason-giving harness, not a re-derivation)
// plus a checksum of every freshly regenerated example report. It excludes
// anything wall-clock or machine-dependent — see baseline-timings.json for
// that — so two runs on an unchanged tree, from a clean checkout, produce
// byte-identical JSON.
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

// checksumRevLine is what normalizeForChecksum substitutes for the
// "| doppel | `<rev>` |" metadata line before hashing an example report.
const checksumRevLine = "| doppel | `<normalized-for-checksum>` |"

// normalizeForChecksum replaces an example report's build-identity line —
// "| doppel | `<git-rev>` |", the one line in buildExampleReport's output
// that names the commit that generated it rather than anything about the
// corpus or the ranking — with a fixed placeholder before hashing.
//
// This is load-bearing, not cosmetic: examples/baseline.json exists so a
// later change can ask "did any report's content actually move", and
// without this normalization every single commit — including ones that
// touch nothing under internal/ or cmd/ — would flip every checksum,
// because doppelRev is `git rev-parse --short HEAD` at generation time. That
// would make the checksum indistinguishable from noise. Every other line —
// the corpus metadata, the run diagnostics, the whole pair and family
// list — is hashed byte for byte.
// It substitutes rather than deletes, where the drift check deletes: both are
// "the provenance row is not content", but a checksum over a report with the
// line removed and one over a report with it replaced are different hashes,
// and the substituted form keeps the line count stable so a diff of two
// normalized reports still lines up. What must not drift is *which* line that
// is, so both spellings key on provenanceRow, defined once in examples_test.go
// beside the generator that writes it.
func normalizeForChecksum(content []byte) []byte {
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, provenanceRow) {
			lines[i] = checksumRevLine
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// sha256Bytes hashes data in memory — used on a freshly regenerated,
// normalized report, never on a file read back off disk.
func sha256Bytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
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

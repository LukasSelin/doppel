package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/retriever"
)

func TestWrapSnippetAddsAClauseOnlyWhenMissing(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantPkg string
	}{
		{"bare function", "func F() {}", "billing"},
		{"own clause wins over --near", "package own\n\nfunc F() {}", "own"},
		{"leading comment then clause", "// doc\npackage own\n\nfunc F() {}", "own"},
		{"comment mentioning package is not a clause", "// this package does X\nfunc F() {}", "billing"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			units, err := parser.ParseSource("q.go", wrapSnippet([]byte(tc.src), "billing"))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(units) != 1 {
				t.Fatalf("got %d units, want 1", len(units))
			}
			if units[0].Package != tc.wantPkg {
				t.Errorf("package = %q, want %q", units[0].Package, tc.wantPkg)
			}
		})
	}
}

// parser.ParseSource returns (nil, nil) on a syntax error, so the empty unit
// slice is the only signal a snippet was garbage. The command must turn that
// into a real error rather than a report of zero matches.
func TestWrapSnippetGarbageYieldsNoUnits(t *testing.T) {
	units, err := parser.ParseSource("q.go", wrapSnippet([]byte("func broken( {{{"), "x"))
	if err != nil {
		t.Fatalf("ParseSource surfaced an error; the (nil, nil) contract changed and runQuery's empty-check may be dead: %v", err)
	}
	if len(units) != 0 {
		t.Fatalf("garbage parsed into %d units", len(units))
	}
}

// The ranking key boosts by locality but never punishes distance: between two
// matches of equal evidence the near one leads, and a strictly stronger
// distant match still beats a weak local one.
func TestRankQueryMatchesLocalityBoostsButEvidenceRules(t *testing.T) {
	mk := func(idx int, total, score, loc float64) reporter.QueryMatch {
		m := reporter.QueryMatch{Locality: loc}
		m.Candidate = retriever.Candidate{AIdx: idx, BIdx: 99, Total: total}
		m.Candidate.Breakdown.Score = score
		return m
	}
	matches := []reporter.QueryMatch{
		mk(0, 10, 0.9, 0.0), // distant, equal evidence
		mk(1, 10, 0.9, 0.5), // near, equal evidence — must lead
		mk(2, 30, 0.5, 0.0), // strictly stronger, distant — must lead them all
	}
	rankQueryMatches(matches, 99)
	if matches[0].Candidate.AIdx != 2 {
		t.Errorf("strongest evidence did not lead: order %d,%d,%d",
			matches[0].Candidate.AIdx, matches[1].Candidate.AIdx, matches[2].Candidate.AIdx)
	}
	if matches[1].Candidate.AIdx != 1 {
		t.Errorf("locality did not break the evidence tie: order %d,%d,%d",
			matches[0].Candidate.AIdx, matches[1].Candidate.AIdx, matches[2].Candidate.AIdx)
	}
}

// Ties on the full key fall to code-shape: an exact clone must not rank below
// a merely similar sibling because of file-walk order.
func TestRankQueryMatchesCodeShapeBreaksTies(t *testing.T) {
	mk := func(idx int, score float64) reporter.QueryMatch {
		m := reporter.QueryMatch{}
		m.Candidate = retriever.Candidate{AIdx: idx, BIdx: 99, Total: 20}
		m.Candidate.Breakdown.Score = score
		return m
	}
	matches := []reporter.QueryMatch{mk(0, 0.58), mk(1, 1.00), mk(2, 0.90)}
	rankQueryMatches(matches, 99)
	if matches[0].Candidate.AIdx != 1 || matches[1].Candidate.AIdx != 2 {
		t.Errorf("order %d,%d,%d; want 1,2,0",
			matches[0].Candidate.AIdx, matches[1].Candidate.AIdx, matches[2].Candidate.AIdx)
	}
}

// index() with extras appends them after the population filter, and analyze()
// (extras nil) must not see them — the split is a pure refactor for the
// existing path, which the byte-identical JSON check covers end to end; this
// covers the seam itself.
func TestIndexAppendsExtrasToTheCorpus(t *testing.T) {
	probe, err := parser.ParseSource("q.go", []byte("package zzz\n\nfunc Probe() {}\n"))
	if err != nil || len(probe) != 1 {
		t.Fatalf("probe fixture: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n\nfunc A() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	p := Params{Threshold: 0.6, MinNodes: 12, ChannelK: 5, TestsMode: "exclude", Generated: "exclude"}
	res, err := index(dir, p, nil, probe)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if len(res.Units) != 2 {
		t.Fatalf("units = %d, want corpus 1 + probe 1", len(res.Units))
	}
	last := res.Units[len(res.Units)-1]
	if last.Package != "zzz" || last.Name != "Probe" {
		t.Errorf("probe is not the appended unit: %s.%s", last.Package, last.Name)
	}
	if res.Onto == nil || res.IC == nil {
		t.Error("index did not expose Onto/IC; the query cannot build a scorer")
	}
	if len(res.Docs) != len(res.Units) {
		t.Errorf("docs (%d) not aligned with units (%d)", len(res.Docs), len(res.Units))
	}
}

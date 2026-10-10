package bench

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// TestRankingsAt ranks one tree — a corpus checked out at an earlier revision
// T — under doppel and every baseline, and writes the lists with the unit
// identities an outcome study needs to find each function in git history. It
// scores nothing: `historylabel compare` reads the file and judges the pairs
// on what happened after T. See examples/ranker-outcomes.md.
//
//	DOPPEL_BENCH_RANKINGS_AT=<tree at T> DOPPEL_BENCH_RANKINGS_CORPUS=<name> \
//	DOPPEL_BENCH_RANKINGS_OUT=<file> go test ./internal/bench/ -run TestRankingsAt -v
//
// The population is the cost study's: Go files only, tests and generated
// files excluded. The pool is the retrieval union at the calibrated threshold,
// so every method ranks the same pairs.
func TestRankingsAt(t *testing.T) {
	root := os.Getenv("DOPPEL_BENCH_RANKINGS_AT")
	if root == "" {
		t.Skip("set DOPPEL_BENCH_RANKINGS_AT to a tree to rank")
	}
	corpus := os.Getenv("DOPPEL_BENCH_RANKINGS_CORPUS")
	out := os.Getenv("DOPPEL_BENCH_RANKINGS_OUT")
	if corpus == "" || out == "" {
		t.Fatal("DOPPEL_BENCH_RANKINGS_CORPUS and DOPPEL_BENCH_RANKINGS_OUT are required")
	}
	br := rankingsRun(t, root, corpus)
	writeRankingsFile(t, out, rankingsAt(br, corpus, root, rankingsTop))
}

// rankingsRun analyses the tree at T over the cost study's population: Go
// files only, tests and generated files excluded.
func rankingsRun(t *testing.T, root, corpus string) *baselineRun {
	t.Helper()
	all, err := Load(root, PopExclude)
	if err != nil {
		t.Fatal(err)
	}
	units := slices.DeleteFunc(all, func(u parser.CodeUnit) bool { return u.Lang != "go" })
	br, err := prepareBaselineRunOf(root, units)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("[%s] %d functions, calibration %s, union %d pairs, report pool %d pairs",
		corpus, len(br.run.Units), br.calib, len(br.run.Pairs), len(br.report))
	return br
}

func writeRankingsFile(t *testing.T, out string, f rankingsFile) {
	t.Helper()
	data, err := json.MarshalIndent(f, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// rankingsTop is where every list is cut: the deepest rank the study reads.
const rankingsTop = 500

// rankingsRandomSeeds is how many random lists the study pools.
const rankingsRandomSeeds = 3

type rankedUnit struct {
	Key     string `json:"key"`
	Package string `json:"package"`
	Name    string `json:"name"`
	File    string `json:"file"`
	Line    int    `json:"line"`
}

type rankedPairOut struct {
	Rank int    `json:"rank"`
	A    string `json:"a"`
	B    string `json:"b"`
}

type rankedList struct {
	Method string          `json:"method"`
	Pairs  []rankedPairOut `json:"pairs"`
}

type rankingsFile struct {
	Corpus      string       `json:"corpus"`
	Calibration string       `json:"calibration"`
	Functions   int          `json:"functions"`
	Union       int          `json:"union"`
	Report      int          `json:"report"`
	Units       []rankedUnit `json:"units"`
	Lists       []rankedList `json:"lists"`
}

// rankingsAt is every method's list over the union pool, cut at top, plus the
// units those lists name.
func rankingsAt(br *baselineRun, corpus, root string, top int) rankingsFile {
	r := br.run
	pool := r.Pairs
	m := poolMethods(br, pool)

	refs := make([]ref, len(pool))
	call := make([]float64, len(pool))
	size := make([]float64, len(pool))
	for i, p := range pool {
		refs[i] = refOf(p)
		if p.Retrieval != nil {
			call[i] = p.Retrieval.Call
		}
		size[i] = math.Min(float64(r.Units[p.AIdx].Fingerprint.Nodes), float64(r.Units[p.BIdx].Fingerprint.Nodes))
	}
	m.add("call mass", rankBy(refs, call))
	m.add("size", rankBy(refs, size))
	names := slices.Clone(m.names)
	lists := slices.Clone(m.lists)
	for s := range rankingsRandomSeeds {
		names = append(names, fmt.Sprintf("random (seed %d)", s+1))
		lists = append(lists, m.random[s])
	}

	return rankingsFileOf(br, corpus, root, names, lists, top)
}

// rankingsFileOf writes named lists, each cut at top, with the units they
// name: the file `historylabel compare` reads, whichever methods made it.
func rankingsFileOf(br *baselineRun, corpus, root string, names []string, lists [][]ref, top int) rankingsFile {
	r := br.run
	f := rankingsFile{Corpus: corpus, Calibration: br.calib, Functions: len(r.Units),
		Union: len(r.Pairs), Report: len(br.report)}
	used := map[int]bool{}
	for i, l := range lists {
		if top > 0 && len(l) > top {
			l = l[:top]
		}
		rl := rankedList{Method: names[i], Pairs: make([]rankedPairOut, len(l))}
		for k, x := range l {
			a, b := br.keys[x.a], br.keys[x.b]
			if a > b {
				a, b = b, a
			}
			rl.Pairs[k] = rankedPairOut{k + 1, a, b}
			used[x.a], used[x.b] = true, true
		}
		f.Lists = append(f.Lists, rl)
	}
	for i, u := range r.Units {
		if !used[i] {
			continue
		}
		f.Units = append(f.Units, rankedUnit{Key: br.keys[i], Package: u.Package, Name: u.Name,
			File: snapshot.RelSlash(root, u.File), Line: u.StartLine})
	}
	return f
}

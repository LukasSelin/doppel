package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// sizeCheckRows are the keys examples/size-aware-check.md scores: production
// first, then the three locked size variants, K-size the one the verdict reads.
func sizeCheckRows() []sizeVariant {
	rows := []sizeVariant{{"production", func(r *Run, _ *sizeFeatures, p analyzer.SimilarPair) float64 {
		return analyzer.RankKey(p, analyzer.DefaultRankOptions(), r.Units)
	}}}
	return append(rows, sizeVariants...)
}

// TestSizeCheckLabels is check 1 of examples/size-aware-check.md: every
// labelled corpus labeledTargets finds — the committed ladder reviews and the
// private ones named by DOPPEL_BENCH_LENSES_LABELS and DOPPEL_BENCH_LENSES_EXTRA
// — scored with ScoreBy under production's key and each size variant's, at
// retrieval defaults (the run TestOverlapRank and TestSizeVariantsGolden
// score). It logs each scorecard, the pooled counts over the corpora that are
// not ladder rungs, and every labelled false positive K-size ranks higher
// than production does. Asserts nothing.
//
//	DOPPEL_BENCH_SIZECHECK=1 DOPPEL_BENCH_LENSES_LABELS=<dir> DOPPEL_BENCH_LENSES_EXTRA=<roots> \
//	go test ./internal/bench/ -run TestSizeCheckLabels -v
//
// The log names private pairs: keep it out of the repository.
func TestSizeCheckLabels(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_SIZECHECK") != "1" {
		t.Skip("set DOPPEL_BENCH_SIZECHECK=1 to score the size variants against every labelled corpus")
	}
	rows := sizeCheckRows()
	type pooled struct {
		violations, fpTop20, fpAbove int
		merge, ref, fp               float64
		nMerge, nRef, nFP            int
	}
	pool := make([]pooled, len(rows))
	private := 0
	for _, tg := range labeledTargets(t) {
		units, err := Load(tg.root, Population(tg.lf.Population))
		if err != nil || len(units) == 0 {
			t.Logf("[%s] load: %v", tg.name, err)
			continue
		}
		run := Analyze(units, retriever.DefaultOptions())
		run.Root = tg.root
		sz := newSizeFeatures(run)
		_, ladder := Find(tg.name)
		if !ladder {
			private++
		}
		nodes := map[string]int{}
		for _, u := range run.Units {
			if n, ok := nodes[qualifiedName(u)]; !ok || u.Fingerprint.Nodes < n {
				nodes[qualifiedName(u)] = u.Fingerprint.Nodes
			}
		}
		cards := make([]Scorecard, len(rows))
		for vi, v := range rows {
			sc := ScoreBy(run, tg.lf, v.memo(run, sz))
			cards[vi] = sc
			t.Logf("[%s] %-20s %s  fp-above-merge %d  fp-top20 %d", tg.name, v.name, scLine(sc), len(sc.FPAboveMerge), len(sc.FPInTop20))
			if ladder {
				continue
			}
			pl := &pool[vi]
			pl.violations += violations(sc)
			pl.fpTop20 += len(sc.FPInTop20)
			pl.fpAbove += len(sc.FPAboveMerge)
			if sc.Present["merge"] > 0 {
				pl.merge += sc.MeanRank["merge"]
				pl.nMerge++
			}
			if sc.Present["refactor"] > 0 {
				pl.ref += sc.MeanRank["refactor"]
				pl.nRef++
			}
			if sc.Present["false_positive"] > 0 {
				pl.fp += sc.MeanRank["false_positive"]
				pl.nFP++
			}
		}
		// Every label K-size moves, false positives that rise marked: the
		// mirror check reads these.
		prod, ks := cards[0], cards[1]
		for i, r := range ks.Results {
			was := prod.Results[i].Rank
			if r.Rank == was {
				continue
			}
			mark := ""
			if r.Label.Class == "false_positive" && r.Rank > 0 && (was == 0 || r.Rank < was) {
				mark = "  RISES"
			}
			t.Logf("    %-14s %4d -> %-4d nodes %d/%d  %s <-> %s%s", r.Label.Class, was, r.Rank,
				nodes[r.Label.A], nodes[r.Label.B], r.Label.A, r.Label.B, mark)
		}
	}
	avg := func(s float64, n int) string {
		if n == 0 {
			return "-"
		}
		return fmt.Sprintf("%.1f", s/float64(n))
	}
	t.Logf("pooled over %d corpora that are not ladder rungs (counts summed; class means averaged per corpus):", private)
	for vi, v := range rows {
		pl := pool[vi]
		t.Logf("  %-20s violations %d  fp-top20 %d  fp-above-merge %d  merge %s  refactor %s  fp %s",
			v.name, pl.violations, pl.fpTop20, pl.fpAbove, avg(pl.merge, pl.nMerge), avg(pl.ref, pl.nRef), avg(pl.fp, pl.nFP))
	}
}

// TestSizeCheckTop20 is check 3 of examples/size-aware-check.md: on each named
// ladder rung, the report's pool (default population, calibrated at 0.01, the
// union at or above the calibrated struct-min) ranked under production's key
// and under each size variant's, --max-per-func 2, top 20. It logs both top 20s
// and the pairs entering and leaving, and writes, for the blind
// classification, every K-size entering or leaving pair as one sorted list
// with no direction (blind.tsv) and the directions apart (directions.tsv).
//
//	DOPPEL_BENCH_SIZECHECK_TOP20=chi,conc,cobra,gin DOPPEL_BENCH_SIZECHECK_OUT=<dir> \
//	go test ./internal/bench/ -run TestSizeCheckTop20 -v
func TestSizeCheckTop20(t *testing.T) {
	names := os.Getenv("DOPPEL_BENCH_SIZECHECK_TOP20")
	if names == "" {
		t.Skip("set DOPPEL_BENCH_SIZECHECK_TOP20 to a comma-separated list of ladder rungs")
	}
	out := os.Getenv("DOPPEL_BENCH_SIZECHECK_OUT")
	const top = 20
	rows := sizeCheckRows()
	var blind, dirs []string
	for _, name := range strings.Split(names, ",") {
		c, ok := Find(name)
		if !ok || !Present(c) {
			t.Fatalf("%s is not a fetched ladder rung", name)
		}
		root, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		br, err := prepareBaselineRun(root, PopExclude)
		if err != nil {
			t.Fatal(err)
		}
		r := br.run
		sz := newSizeFeatures(r)
		t.Logf("[%s] %d functions, %s, report pool %d pairs", name, len(r.Units), br.calib, len(br.report))
		where := func(i int) string {
			u := r.Units[i]
			rel, err := filepath.Rel(root, u.File)
			if err != nil {
				rel = u.File
			}
			return fmt.Sprintf("%s (%s:%d, %d nodes)", qualifiedName(u), filepath.ToSlash(rel), u.StartLine, u.Fingerprint.Nodes)
		}
		lists := make([][]analyzer.SimilarPair, len(rows))
		for vi, v := range rows {
			kept, _ := analyzer.SortForReportBy(slices.Clone(br.report), v.memo(r, sz), top, 2)
			lists[vi] = kept
			t.Logf("[%s] %s top %d:", name, v.name, top)
			for i, p := range kept {
				t.Logf("    %2d  shape %.2f  cont %.2f  %s <-> %s", i+1, p.Score, p.Breakdown.Containment, where(p.AIdx), where(p.BIdx))
			}
		}
		in := func(l []analyzer.SimilarPair, p analyzer.SimilarPair) bool {
			return slices.ContainsFunc(l, func(q analyzer.SimilarPair) bool { return refOf(q) == refOf(p) })
		}
		for vi := 1; vi < len(rows); vi++ {
			var enter, leave []analyzer.SimilarPair
			for _, p := range lists[vi] {
				if !in(lists[0], p) {
					enter = append(enter, p)
				}
			}
			for _, p := range lists[0] {
				if !in(lists[vi], p) {
					leave = append(leave, p)
				}
			}
			t.Logf("[%s] %s: %d enter, %d leave", name, rows[vi].name, len(enter), len(leave))
			for _, p := range enter {
				t.Logf("    enter  %s <-> %s", where(p.AIdx), where(p.BIdx))
			}
			for _, p := range leave {
				t.Logf("    leave  %s <-> %s", where(p.AIdx), where(p.BIdx))
			}
			if vi != 1 {
				continue
			}
			for _, d := range []struct {
				dir   string
				pairs []analyzer.SimilarPair
			}{{"enter", enter}, {"leave", leave}} {
				for _, p := range d.pairs {
					line := fmt.Sprintf("%s\t%s\t%s", name, where(p.AIdx), where(p.BIdx))
					blind = append(blind, line)
					dirs = append(dirs, line+"\t"+d.dir)
				}
			}
		}
	}
	if out == "" {
		return
	}
	sort.Strings(blind)
	sort.Strings(dirs)
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	for f, lines := range map[string][]string{"blind.tsv": blind, "directions.tsv": dirs} {
		if err := os.WriteFile(filepath.Join(out, f), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

package bench

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// groupRows assigns each ranked pair a display row. A pair whose two sides
// both belong to a qualifying family that already has a row is shown under
// that row, as a child; any other pair opens a new row and becomes the head of
// every qualifying family it belongs to. The ranking itself is untouched —
// a child keeps its own rank and score, it only stops costing a row.
func groupRows(kept []analyzer.SimilarPair, nUnits int, fams []family.Family, ok func(family.Family) bool) (rows []int, head []bool) {
	famsOf := make([][]int, nUnits)
	for fi, f := range fams {
		if !ok(f) {
			continue
		}
		for _, m := range f.Members {
			famsOf[m] = append(famsOf[m], fi)
		}
	}
	headRow := map[int]int{}
	rows = make([]int, len(kept))
	head = make([]bool, len(kept))
	next := 0
	for i, p := range kept {
		var shared []int
		for _, fa := range famsOf[p.AIdx] {
			for _, fb := range famsOf[p.BIdx] {
				if fa == fb {
					shared = append(shared, fa)
				}
			}
		}
		row := 0
		for _, f := range shared {
			if r, ok := headRow[f]; ok && (row == 0 || r < row) {
				row = r
			}
		}
		if row == 0 {
			next++
			row = next
			head[i] = true
			for _, f := range shared {
				headRow[f] = row
			}
		}
		rows[i] = row
	}
	return rows, head
}

func TestGroupRows(t *testing.T) {
	// Units 0,1,2 form a family; 3,4 do not. Ranked: 0-1, 3-4, 0-2, 1-2.
	kept := []analyzer.SimilarPair{{AIdx: 0, BIdx: 1}, {AIdx: 3, BIdx: 4}, {AIdx: 0, BIdx: 2}, {AIdx: 1, BIdx: 2}}
	fams := []family.Family{{Members: []int{0, 1, 2}}}
	rows, head := groupRows(kept, 5, fams, func(family.Family) bool { return true })
	if fmt.Sprint(rows) != "[1 2 1 1]" || fmt.Sprint(head) != "[true true false false]" {
		t.Fatalf("rows %v head %v", rows, head)
	}
	rows, _ = groupRows(kept, 5, fams, func(family.Family) bool { return false })
	if fmt.Sprint(rows) != "[1 2 3 4]" {
		t.Fatalf("ungrouped rows %v", rows)
	}
}

// groupVariant is one grouping rule under test.
type groupVariant struct {
	name string
	ok   func(family.Family) bool
}

func groupVariants() []groupVariant {
	return []groupVariant{
		{"none", func(family.Family) bool { return false }},
		{"all families", func(family.Family) bool { return true }},
		{"min>=0.90", func(f family.Family) bool { return f.MinEdge >= 0.90 }},
		{"min>=0.9*mean", func(f family.Family) bool { return f.MinEdge >= 0.9*f.MeanEdge }},
		{"min>=0.9*mean,<=8", func(f family.Family) bool { return f.MinEdge >= 0.9*f.MeanEdge && len(f.Members) <= 8 }},
	}
}

// groupScore is a labels file scored against grouped rows.
type groupScore struct {
	rows                               int
	mergeMean, refactorMean, fpMean    float64
	mergeIn20, mergeTotal              int
	fpIn20, fpHeadsIn20                int
	tpIn20, tpHeadsIn20                int
	tpUnderFP, mixedRows, groupedPairs int
}

func scoreGrouped(run *Run, lf LabelsFile, cap int, fams []family.Family, ok func(family.Family) bool) groupScore {
	kept, _ := analyzer.SortForReportWith(run.Pairs, run.Units, 0, cap, analyzer.DefaultRankOptions())
	rows, head := groupRows(kept, len(run.Units), fams, ok)

	ranked := map[string][]scoredPair{}
	type slot struct {
		row  int
		head bool
	}
	slots := map[string][]slot{}
	for i, p := range kept {
		sp := run.scoredPair(p)
		sp.rank = i + 1
		ranked[sp.key] = append(ranked[sp.key], sp)
		slots[sp.key] = append(slots[sp.key], slot{rows[i], head[i]})
	}
	var gs groupScore
	if len(rows) > 0 {
		gs.rows = rows[len(rows)-1]
		for _, r := range rows {
			if r > gs.rows {
				gs.rows = r
			}
		}
	}
	for i := range head {
		if !head[i] {
			gs.groupedPairs++
		}
	}

	sum := map[string]int{}
	n := map[string]int{}
	rowClasses := map[int]map[string]bool{}
	type placed struct {
		row   int
		head  bool
		class string
	}
	var ps []placed
	for _, l := range lf.Labels {
		k := pairKey(l.A, l.B)
		sp, found := firstMatch(ranked[k], l, run.Root)
		if l.Class == "merge" {
			gs.mergeTotal++
		}
		if !found {
			continue
		}
		var s slot
		for j, c := range ranked[k] {
			if c.rank == sp.rank {
				s = slots[k][j]
			}
		}
		sum[l.Class] += s.row
		n[l.Class]++
		ps = append(ps, placed{s.row, s.head, l.Class})
		if rowClasses[s.row] == nil {
			rowClasses[s.row] = map[string]bool{}
		}
		tp := "tp"
		if l.Class == "false_positive" {
			tp = "fp"
		}
		rowClasses[s.row][tp] = true
		if s.row <= 20 && l.Class != "false_positive" {
			gs.tpIn20++
			if s.head {
				gs.tpHeadsIn20++
			}
		}
		if s.row <= 20 {
			switch l.Class {
			case "merge":
				gs.mergeIn20++
			case "false_positive":
				gs.fpIn20++
				if s.head {
					gs.fpHeadsIn20++
				}
			}
		}
	}
	// A true match shown as a child under a row whose head is labelled a
	// false positive is the case grouping can hurt: the reader may dismiss
	// the row by its head.
	fpHead := map[int]bool{}
	for _, p := range ps {
		if p.head && p.class == "false_positive" {
			fpHead[p.row] = true
		}
	}
	for _, p := range ps {
		if !p.head && p.class != "false_positive" && fpHead[p.row] {
			gs.tpUnderFP++
		}
	}
	for _, c := range rowClasses {
		if c["tp"] && c["fp"] {
			gs.mixedRows++
		}
	}
	mean := func(c string) float64 {
		if n[c] == 0 {
			return 0
		}
		return float64(sum[c]) / float64(n[c])
	}
	gs.mergeMean, gs.refactorMean, gs.fpMean = mean("merge"), mean("refactor"), mean("false_positive")
	return gs
}

// TestGroupedDisplay measures grouped display against labelled corpora given
// as DOPPEL_BENCH_GROUP="<corpus dir>=<labels file>[=<family-min>];…". The
// family edge cut defaults to family.DefaultOptions' static 0.60; pass the
// value `doppel analyze` calibrates for the corpus to measure what the tool
// would actually group. Rows are what a reader scans, so the headline
// metrics are what reaches the first 20 rows. Mean row ranks are printed
// too, but grouping compresses every class upward, so they flatter any
// variant that groups at all. Asserts nothing.
func TestGroupedDisplay(t *testing.T) {
	spec := os.Getenv("DOPPEL_BENCH_GROUP")
	if spec == "" {
		t.Skip("set DOPPEL_BENCH_GROUP=<corpus>=<labels>;... to measure grouped display")
	}
	for _, item := range strings.Split(spec, ";") {
		parts := strings.Split(item, "=")
		if len(parts) < 2 || len(parts) > 3 {
			t.Fatalf("bad entry %q", item)
		}
		corpus, labelsPath := parts[0], parts[1]
		fo := family.DefaultOptions()
		if len(parts) == 3 {
			if _, err := fmt.Sscanf(parts[2], "%g", &fo.Min); err != nil {
				t.Fatalf("bad family-min in %q: %v", item, err)
			}
		}
		data, err := os.ReadFile(labelsPath)
		if err != nil {
			t.Fatal(err)
		}
		lf, err := ParseLabels(data)
		if err != nil {
			t.Fatal(err)
		}
		units, err := Load(corpus, Population(lf.Population))
		if err != nil {
			t.Fatal(err)
		}
		run := Analyze(units, retriever.DefaultOptions())
		run.Root = corpus
		fams, _ := family.Build(run.Units, run.Pairs, run.WL, fo)
		t.Logf("== %s: %d functions, %d pairs, %d families at family-min %.2f", lf.Corpus, len(units), len(run.Pairs), len(fams), fo.Min)
		t.Logf("   %-3s %-18s %6s %6s %6s %6s %6s %6s %6s %6s %5s %5s %7s",
			"cap", "grouping", "rows", "m@20", "tp@20", "tpHd20", "fp@20", "fpHd20", "merge", "fp", "tp<fp", "mixed", "grouped")
		for _, cap := range []int{2, 0} {
			for _, v := range groupVariants() {
				g := scoreGrouped(run, lf, cap, fams, v.ok)
				t.Logf("   %-3d %-18s %6d %3d/%-2d %6d %6d %6d %6d %6.1f %6.1f %5d %5d %7d",
					cap, v.name, g.rows, g.mergeIn20, g.mergeTotal, g.tpIn20, g.tpHeadsIn20,
					g.fpIn20, g.fpHeadsIn20, g.mergeMean, g.fpMean, g.tpUnderFP, g.mixedRows, g.groupedPairs)
			}
		}
	}
}

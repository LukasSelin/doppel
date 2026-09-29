package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/gofront"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
	"github.com/LukasSelin/doppel/internal/syntax"
)

// TestLenses reads every compared pair under every fingerprint.Lens and
// reports what the lenses say that the one code-shape number cannot. It is
// measurement only: the production pipeline runs unchanged, and the lenses
// are scored beside it, each against its own corpus weights.
//
// Per corpus it logs how the compared pairs distribute over the exact
// classes (LensProfile.Class), a few examples of each, the per-lens reading
// of the report's top 20, and — where labels exist — every labeled pair's
// profile plus the per-lens mean Jaccard for each label class. That last
// table is the tuning input: a lens whose mean separates merge from
// false_positive better than the shape lens does is a candidate for the
// ranking, and one that does not is an annotation at most. Asserts nothing.
//
//	DOPPEL_BENCH_LENSES=1 go test ./internal/bench/ -v -run TestLenses
//
// DOPPEL_BENCH_LENSES_EXTRA is an OS path list of further corpus roots.
func TestLenses(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_LENSES") != "1" {
		t.Skip("set DOPPEL_BENCH_LENSES=1 to measure the fingerprint lenses")
	}
	type target struct{ name, root string }
	var targets []target
	for _, c := range Corpora {
		if !Present(c) {
			continue
		}
		root, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		targets = append(targets, target{c.Name, root})
	}
	if extra := os.Getenv("DOPPEL_BENCH_LENSES_EXTRA"); extra != "" {
		for _, root := range filepath.SplitList(extra) {
			targets = append(targets, target{filepath.Base(root), root})
		}
	}
	if len(targets) == 0 {
		t.Skip("no corpora fetched and no DOPPEL_BENCH_LENSES_EXTRA")
	}
	labels := committedLabels(t)
	lenses := fingerprint.Lenses()

	for _, tg := range targets {
		pop := PopExclude
		lf, labeled := labels[tg.name]
		if labeled {
			pop = Population(lf.Population)
		}
		units, err := Load(tg.root, pop)
		if err != nil || len(units) == 0 {
			t.Logf("[%s] load: %v (%d units)", tg.name, err, len(units))
			continue
		}
		run := Analyze(units, retriever.DefaultOptions())
		fns, missing := lensFuncs(run.Units)

		bags := make([][][]fingerprint.LabelCount, len(lenses))
		idfs := make([]*fingerprint.LabelIDF, len(lenses))
		for li, l := range lenses {
			bags[li] = make([][]fingerprint.LabelCount, len(fns))
			for i := range fns {
				bags[li][i] = l.Bag(fns[i])
			}
			idfs[li] = fingerprint.LabelWeights(bags[li])
		}
		profile := func(a, b int) fingerprint.LensProfile {
			var p fingerprint.LensProfile
			for li := range lenses {
				p[li] = fingerprint.ScoreLens(bags[li][a], bags[li][b], idfs[li])
			}
			return p
		}

		ranked := append([]analyzer.SimilarPair(nil), run.Pairs...)
		kept, _ := analyzer.SortForReport(ranked, run.Units, 0, 0)
		t.Logf("[%s] %d units (%d without a written body, verbatim reads the shape there), %d compared pairs",
			tg.name, len(run.Units), missing, len(kept))

		counts := map[string]int{}
		examples := map[string][]string{}
		for _, p := range kept {
			c := profile(p.AIdx, p.BIdx).Class()
			if c == "" {
				c = "-"
			}
			counts[c]++
			if len(examples[c]) < 4 {
				examples[c] = append(examples[c], wlPairName(run.Units, p))
			}
		}
		for _, c := range []string{"verbatim", "renamed", "inverse", "parameterized", "template", "-"} {
			t.Logf("  %-13s %6d", c, counts[c])
			if c == "-" {
				continue
			}
			for _, e := range examples[c] {
				t.Logf("      %s", e)
			}
		}

		t.Logf("  top 20 by report rank      skel  shape  ordr  vocab  verb  class")
		top, _ := analyzer.SortForReport(append([]analyzer.SimilarPair(nil), run.Pairs...), run.Units, 20, 2)
		for i, p := range top {
			t.Logf("  %2d %s", i+1, profileLine(profile(p.AIdx, p.BIdx), wlPairName(run.Units, p)))
		}

		if !labeled {
			continue
		}
		byName := map[string]int{}
		for i, u := range run.Units {
			byName[qualifiedName(u)] = i
		}
		sums := map[string]*[fingerprint.NumLenses + 1]float64{} // lens jaccards, then n
		t.Logf("  labeled pairs              skel  shape  ordr  vocab  verb  class")
		ls := append([]Label(nil), lf.Labels...)
		sort.Slice(ls, func(i, j int) bool {
			if ls[i].Class != ls[j].Class {
				return ls[i].Class < ls[j].Class
			}
			return pairKey(ls[i].A, ls[i].B) < pairKey(ls[j].A, ls[j].B)
		})
		for _, l := range ls {
			a, okA := byName[l.A]
			b, okB := byName[l.B]
			if !okA || !okB {
				t.Logf("  %-14s %s <-> %s: not in the population", l.Class, l.A, l.B)
				continue
			}
			p := profile(a, b)
			t.Logf("  %-14s %s", l.Class, profileLine(p, l.A+" <-> "+l.B))
			s := sums[l.Class]
			if s == nil {
				s = &[fingerprint.NumLenses + 1]float64{}
				sums[l.Class] = s
			}
			for li := range lenses {
				s[li] += p[li].Jaccard
			}
			s[fingerprint.NumLenses]++
		}
		t.Logf("  mean jaccard by class      skel  shape  ordr  vocab  verb     n")
		for _, c := range []string{"merge", "refactor", "false_positive"} {
			s := sums[c]
			if s == nil {
				continue
			}
			n := s[fingerprint.NumLenses]
			var b strings.Builder
			for li := range lenses {
				fmt.Fprintf(&b, "  %.2f", s[li]/n)
			}
			t.Logf("  %-24s%s  %4.0f", c, b.String(), n)
		}
	}
}

func profileLine(p fingerprint.LensProfile, name string) string {
	var b strings.Builder
	for _, s := range p {
		mark := " "
		if s.Exact {
			mark = "="
		}
		fmt.Fprintf(&b, " %.2f%s", s.Jaccard, mark)
	}
	c := p.Class()
	if c == "" {
		c = "-"
	}
	return fmt.Sprintf("%-60s%s  %s", trim(name, 60), b.String(), c)
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n+1:]
}

// lensFuncs rebuilds each unit's syntax.Func, which the verbatim lens needs:
// a CodeUnit keeps its canonical tree but not the body as written. Go files
// are re-parsed and matched by name and start line; a unit from a frontend
// with no canonicalizer already holds its written body as Canonical. A Go unit
// that cannot be matched keeps its canonical tree for both, and is counted.
func lensFuncs(units []parser.CodeUnit) ([]*syntax.Func, int) {
	type key struct {
		name string
		line int
	}
	files := map[string]map[key]*syntax.Func{}
	out := make([]*syntax.Func, len(units))
	missing := 0
	for i, u := range units {
		if u.Canonical == nil {
			continue
		}
		fallback := &syntax.Func{Body: u.Canonical, Canon: u.Canonical}
		if len(u.CanonRules) == 0 && !strings.HasSuffix(u.File, ".go") {
			out[i] = fallback
			continue
		}
		byKey, ok := files[u.File]
		if !ok {
			byKey = map[key]*syntax.Func{}
			if f, err := gofront.ParseFile(u.File); err == nil && f != nil {
				for j := range f.Funcs {
					fn := &f.Funcs[j]
					name := fn.Name
					if fn.Receiver != "" {
						name = fn.Receiver + "." + fn.Name
					}
					byKey[key{name, fn.StartLine}] = fn
				}
			}
			files[u.File] = byKey
		}
		if fn, ok := byKey[key{u.Name, u.StartLine}]; ok && fn.Body != nil {
			out[i] = fn
			continue
		}
		out[i] = fallback
		missing++
	}
	return out, missing
}

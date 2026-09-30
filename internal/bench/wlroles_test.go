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
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// TestWLRoles measures the role-aware WL recurrence (fingerprint.WLRoles)
// against the production one. The production bag sorts a node's children, so
// an assignment's two sides commute and a gather scores 1.00 against its
// scatter; the variants fold a child's slot in before the sort. Every unit's
// bag is rebuilt from its canonical tree under each variant and the whole
// ranking pipeline re-runs, the lexicon included — its act: channel reads WL
// labels, so a variant changes the vocabulary exactly as it would in
// production.
//
// Per corpus and variant it logs the candidate union, how many pairs read an
// exact code-shape, how much of the top 20 survives, and the pairs that were
// exact under production and no longer are — the list a reader eyeballs to
// decide whether the variant split mirror images or broke real clones. Where
// labels exist it logs the scorecard. It asserts nothing.
//
//	DOPPEL_BENCH_WLROLES=1 go test ./internal/bench/ -v -run TestWLRoles
//
// DOPPEL_BENCH_WLROLES_EXTRA is an OS path list of further corpus roots.
func TestWLRoles(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_WLROLES") != "1" {
		t.Skip("set DOPPEL_BENCH_WLROLES=1 to measure the role-aware WL variants")
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
	if extra := os.Getenv("DOPPEL_BENCH_WLROLES_EXTRA"); extra != "" {
		for _, root := range filepath.SplitList(extra) {
			targets = append(targets, target{filepath.Base(root), root})
		}
	}
	if len(targets) == 0 {
		t.Skip("no corpora fetched and no DOPPEL_BENCH_WLROLES_EXTRA")
	}
	labels := committedLabels(t)

	variants := []struct {
		name string
		mode fingerprint.WLRoles
	}{
		{"none", fingerprint.WLRolesNone},
		{"assign", fingerprint.WLRolesAssign},
		{"directed", fingerprint.WLRolesDirected},
	}

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
		var baseTop []string
		var baseExact [][2]int // unit indices; units keep one order across variants
		for _, v := range variants {
			us := make([]parser.CodeUnit, len(units))
			copy(us, units)
			for i := range us {
				us[i].Fingerprint.WL = fingerprint.WLBagWith(us[i].Canonical, fingerprint.WLOptions{Roles: v.mode})
			}
			run := Analyze(us, retriever.DefaultOptions())

			var exact [][2]int
			for _, p := range run.Pairs {
				if p.Score >= 0.999 {
					exact = append(exact, [2]int{p.AIdx, p.BIdx})
				}
			}
			ranked := append([]analyzer.SimilarPair(nil), run.Pairs...)
			kept, _ := analyzer.SortForReport(ranked, run.Units, 20, 2)
			top := make([]string, len(kept))
			for i, p := range kept {
				top[i] = wlPairName(run.Units, p)
			}

			line := fmt.Sprintf("[%s] %-8s union %6d  pairs %6d  exact %5d", tg.name, v.name, len(run.Cands), len(run.Pairs), len(exact))
			if labeled {
				line += "  | " + scLine(Score(run, lf))
			}
			if v.mode == fingerprint.WLRolesNone {
				baseTop, baseExact = top, exact
				t.Log(line)
				continue
			}
			t.Logf("%s  top20 kept %d/%d", line, overlap(baseTop, top), len(baseTop))
			for _, k := range difference(top, baseTop) {
				t.Logf("    + top20 %s", k)
			}
			for _, k := range difference(baseTop, top) {
				t.Logf("    - top20 %s", k)
			}
			// Scored directly rather than read off run.Pairs: a pair can
			// leave the union through a top-K tie shifting, which says
			// nothing about whether the variant split it.
			var split []string
			for _, e := range baseExact {
				a, b := run.Units[e[0]], run.Units[e[1]]
				sc := fingerprint.Similarity(a.Fingerprint, b.Fingerprint, run.WL).Score
				if sc < 0.999 {
					split = append(split, fmt.Sprintf("%.3f  %s", sc,
						wlPairName(run.Units, analyzer.SimilarPair{AIdx: e[0], BIdx: e[1]})))
				}
			}
			sort.Strings(split)
			t.Logf("    exact under none, below 1.00 now: %d of %d", len(split), len(baseExact))
			for i, s := range split {
				if i == 25 {
					t.Logf("      ... %d more", len(split)-i)
					break
				}
				t.Logf("      %s", s)
			}
		}
	}
}

func committedLabels(t *testing.T) map[string]LabelsFile {
	t.Helper()
	out := map[string]LabelsFile{}
	entries, _ := filepath.Glob(filepath.Join(repoRoot(t), "examples", "labels", "*.labels.json"))
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lf, err := ParseLabels(data)
		if err != nil {
			t.Fatal(err)
		}
		out[strings.TrimSuffix(filepath.Base(path), ".labels.json")] = lf
	}
	return out
}

// wlPairName is a pair's identity across variants: ordered qualified names
// with the file base, since a name alone collides across main packages.
func wlPairName(units []parser.CodeUnit, p analyzer.SimilarPair) string {
	a, b := units[p.AIdx], units[p.BIdx]
	na := qualifiedName(a) + "@" + filepath.Base(a.File)
	nb := qualifiedName(b) + "@" + filepath.Base(b.File)
	if nb < na {
		na, nb = nb, na
	}
	return na + " <-> " + nb
}

func overlap(a, b []string) int {
	in := map[string]bool{}
	for _, k := range a {
		in[k] = true
	}
	n := 0
	for _, k := range b {
		if in[k] {
			n++
		}
	}
	return n
}

func difference(a, b []string) []string {
	in := map[string]bool{}
	for _, k := range b {
		in[k] = true
	}
	var out []string
	for _, k := range a {
		if !in[k] {
			out = append(out, k)
		}
	}
	return out
}

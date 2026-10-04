package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// flowQuantities are the flow-view numbers the measurement reads per pair,
// beside the production code-shape they are meant to complement.
type flowQuantities struct {
	shape, steps, exact, types float64
}

// flowOf computes the flow view of one pair. exact is the alignment with
// retargeted steps counted at zero — 2·Same/(|A|+|B|) — which asks whether a
// pair does the same things to the same targets, where steps also credits
// doing the same kind of thing to a different one.
func flowOf(run *Run, p analyzer.SimilarPair) flowQuantities {
	a, b := run.Units[p.AIdx].Fingerprint, run.Units[p.BIdx].Fingerprint
	fs := fingerprint.FlowSimilarity(a, b)
	q := flowQuantities{shape: p.Score, steps: fs.Steps, types: fs.Types, exact: 1}
	if n := fs.LenA + fs.LenB; n > 0 {
		q.exact = 2 * float64(fs.Same) / float64(n)
	}
	return q
}

// flowVariant is one way of letting the flow view into the rank key. Every
// variant scales the pair's retrieval Total, which the key is linear in, so a
// factor f is exactly "key × f" and nothing else about the pair moves.
type flowVariant struct {
	name   string
	factor func(q flowQuantities) float64
}

func demoteBelow(read func(flowQuantities) float64, floor, to float64) func(flowQuantities) float64 {
	return func(q flowQuantities) float64 {
		if read(q) < floor {
			return to
		}
		return 1
	}
}

var flowVariants = []flowVariant{
	{"production", nil},
	{"x steps", func(q flowQuantities) float64 { return q.steps }},
	{"x steps^2", func(q flowQuantities) float64 { return q.steps * q.steps }},
	{"x exact", func(q flowQuantities) float64 { return q.exact }},
	{"x exact^2", func(q flowQuantities) float64 { return q.exact * q.exact }},
	{"x types", func(q flowQuantities) float64 { return q.types }},
	{"x steps*types", func(q flowQuantities) float64 { return q.steps * q.types }},
	{"steps<0.5 -> x0.25", demoteBelow(func(q flowQuantities) float64 { return q.steps }, 0.5, 0.25)},
	{"exact<0.5 -> x0.25", demoteBelow(func(q flowQuantities) float64 { return q.exact }, 0.5, 0.25)},
	{"exact<0.3 -> x0.25", demoteBelow(func(q flowQuantities) float64 { return q.exact }, 0.3, 0.25)},
	// Read off the labels — every labelled merge aligns at 0.89 or above — so
	// these two rows are fitted to what they are scored on, and say how much
	// a necessary-condition floor could buy at best, not what it would buy.
	{"steps<0.85 -> x0.25", demoteBelow(func(q flowQuantities) float64 { return q.steps }, 0.85, 0.25)},
	{"exact<0.85 -> x0.25", demoteBelow(func(q flowQuantities) float64 { return q.exact }, 0.85, 0.25)},
}

// flowTarget is one labelled corpus. Unlike labeledTargets it never skips on
// its own, so the spec list can supply every corpus.
func flowTargets(t *testing.T) []labeledTarget {
	t.Helper()
	var targets []labeledTarget
	committed := committedLabels(t)
	for _, c := range Corpora {
		lf, ok := committed[c.Name]
		if !ok || !Present(c) {
			continue
		}
		root, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		targets = append(targets, labeledTarget{c.Name, root, lf})
	}
	if dir := os.Getenv("DOPPEL_BENCH_LENSES_LABELS"); dir != "" {
		for _, root := range filepath.SplitList(os.Getenv("DOPPEL_BENCH_LENSES_EXTRA")) {
			name := filepath.Base(root)
			data, err := os.ReadFile(filepath.Join(dir, name+".labels.json"))
			if err != nil {
				continue
			}
			lf, err := ParseLabels(data)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			targets = append(targets, labeledTarget{name, root, lf})
		}
	}
	if spec := os.Getenv("DOPPEL_BENCH_FLOW_SPECS"); spec != "" {
		for _, s := range parseLabeledSpecs(t, spec) {
			targets = append(targets, labeledTarget{filepath.Base(s.root), s.root, s.lf})
		}
	}
	return targets
}

// TestFlowRank measures the flow view against the labels: whether ordered
// logic separates merges from false positives where code-shape does not, and
// what letting it into the rank key would do. Three tables per corpus — the
// class means of shape, steps, exact and types; the same by false-positive
// kind where the labels carry one; and the scorecard under each variant — then
// the per-quantity AUCs and the variant scorecards pooled across corpora.
// Asserts nothing.
//
// # Measured, and not adopted (2026-10-03)
//
// Five labelled corpora — cobra, this repository's .doppel/labels.json and
// three private ones; 160 labelled pairs retrieved (28 merge, 41 refactor,
// 91 false positive). Pooled AUC, merge against false positive: shape 0.916,
// steps 0.902, exact 0.905, types 0.683. Merge
// and refactor against false positive: shape 0.814, steps 0.717, exact 0.776,
// types 0.622. So the flow view separates the classes about as well as
// code-shape and never better — and within the band where shape is already
// high (shape >= 0.75: 26 merges, 18 refactors, 16 false positives), the
// question a new signal has to answer, steps reads 0.644 / 0.518 against
// shape's 0.644 / 0.536: no separation shape does not already give.
//
// The reason is in the per-kind table (this repository's labels, the only
// ones carrying kinds). Mirrors read steps 0.97, accessor families 1.00,
// already-factored helpers 0.90: the same logic in the same order, against a
// different target or over different data, which is what those kinds are.
// Where flow does read low — skeletons, 0.72 — shape reads lower still
// (0.59). Every labelled merge aligns at 0.89 or above, so ordered flow is a
// necessary condition for a merge, and one the key already enforces through
// shape².
//
// No variant is adoptable (pooled scorecard, production: merge 142.6,
// refactor 27.7, fp 200.3, 67 violations). A linear factor barely moves the
// violations (x steps 66, x exact 65) while doubling the refactor mean.
// Stronger factors buy violations only by pushing merges down with everything
// else: x exact² 58 with merge 233.5, x steps·types 44 with merge 289.4 —
// cobra's merge mean going 4.8 -> 24.8 and 66.5, and merges leaving its top
// 50. Demoting below steps 0.5 gives 63 at merge 149.7 (cobra 4.8 -> 11.5,
// this repository 21.7 -> 33.9). A floor fitted to the labels at 0.85, which
// every merge clears, gives 66. The view ships as a view: its value is
// showing *where* two bodies differ — the retargeted rows of a mirror — not
// ranking them.
//
//	DOPPEL_BENCH_FLOW=1 go test ./internal/bench/ -v -run TestFlowRank
//
// Corpora: the committed reviews of fetched rungs, private ones through
// DOPPEL_BENCH_LENSES_LABELS / DOPPEL_BENCH_LENSES_EXTRA (as TestLensRank), and
// any "<corpus>=<labels>" entries in DOPPEL_BENCH_FLOW_SPECS (';'-separated) —
// this repository's own .doppel/labels.json among them.
func TestFlowRank(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_FLOW") != "1" {
		t.Skip("set DOPPEL_BENCH_FLOW=1 to measure the flow view against the labels")
	}
	targets := flowTargets(t)
	if len(targets) == 0 {
		t.Skip("no labeled corpus available")
	}

	type pooled struct {
		violations        int
		merge, ref, fp    float64
		nMerge, nRef, nFP int
		fpTop20           int
	}
	pool := make([]pooled, len(flowVariants))
	// Every labelled pair's quantities, pooled, for the AUCs.
	byClass := map[string][]flowQuantities{}

	for _, tg := range targets {
		units, err := Load(tg.root, Population(tg.lf.Population))
		if err != nil || len(units) == 0 {
			t.Logf("[%s] load: %v", tg.name, err)
			continue
		}
		run := Analyze(units, retriever.DefaultOptions())
		quant := make([]flowQuantities, len(run.Pairs))
		for i, p := range run.Pairs {
			quant[i] = flowOf(run, p)
		}
		flowDiagnose(t, tg.name, run, tg.lf, quant, byClass)

		var prod Scorecard
		for vi, v := range flowVariants {
			r := *run
			if v.factor != nil {
				r.Pairs = make([]analyzer.SimilarPair, len(run.Pairs))
				for i, p := range run.Pairs {
					if p.Retrieval != nil {
						ret := *p.Retrieval
						ret.Total *= v.factor(quant[i])
						p.Retrieval = &ret
					}
					r.Pairs[i] = p
				}
			}
			sc := ScoreWith(&r, tg.lf, analyzer.DefaultRankOptions())
			t.Logf("[%s] %-20s %s  fp-in-top20 %d", tg.name, v.name, scLine(sc), len(sc.FPInTop20))
			pl := &pool[vi]
			pl.violations += violations(sc)
			pl.fpTop20 += len(sc.FPInTop20)
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
			if vi == 0 {
				prod = sc
				continue
			}
			for i, res := range sc.Results {
				was := prod.Results[i]
				if res.Rank == was.Rank || (res.Rank > 20 && was.Rank > 20) {
					continue // only movement into, out of or within the top 20
				}
				t.Logf("      %-14s %4d -> %-4d %s <-> %s", res.Label.Class, was.Rank, res.Rank, res.Label.A, res.Label.B)
			}
		}
	}

	t.Logf("pooled AUC over every labelled pair retrieved (P[positive reads higher than false positive], ties half):")
	read := []struct {
		name string
		get  func(flowQuantities) float64
	}{
		{"shape", func(q flowQuantities) float64 { return q.shape }},
		{"steps", func(q flowQuantities) float64 { return q.steps }},
		{"exact", func(q flowQuantities) float64 { return q.exact }},
		{"types", func(q flowQuantities) float64 { return q.types }},
	}
	// The second pair of columns is the question a new signal has to answer:
	// among pairs code-shape already rates highly, does it still separate?
	// A quantity that only agrees with shape adds nothing to a key that
	// already squares shape.
	const band = 0.75
	for _, r := range read {
		var merge, mergeRef, fp, bMerge, bMergeRef, bFP []float64
		for _, c := range []string{"merge", "refactor", "false_positive"} {
			for _, q := range byClass[c] {
				v, hi := r.get(q), q.shape >= band
				switch c {
				case "merge":
					merge, mergeRef = append(merge, v), append(mergeRef, v)
					if hi {
						bMerge, bMergeRef = append(bMerge, v), append(bMergeRef, v)
					}
				case "refactor":
					mergeRef = append(mergeRef, v)
					if hi {
						bMergeRef = append(bMergeRef, v)
					}
				default:
					fp = append(fp, v)
					if hi {
						bFP = append(bFP, v)
					}
				}
			}
		}
		t.Logf("  %-6s merge vs fp %.3f   merge+refactor vs fp %.3f   | shape >= %.2f (%d/%d/%d): %.3f   %.3f",
			r.name, auc(merge, fp), auc(mergeRef, fp), band, len(bMerge), len(bMergeRef)-len(bMerge), len(bFP),
			auc(bMerge, bFP), auc(bMergeRef, bFP))
	}

	t.Logf("pooled over %d labeled corpora (per-corpus class means, averaged; lower merge and higher fp are better):", len(targets))
	avg := func(s float64, n int) string {
		if n == 0 {
			return "-"
		}
		return fmt.Sprintf("%.1f", s/float64(n))
	}
	for vi, v := range flowVariants {
		pl := pool[vi]
		t.Logf("  %-20s merge %5s  refactor %5s  fp %6s  violations %3d  fp-in-top20 %3d",
			v.name, avg(pl.merge, pl.nMerge), avg(pl.ref, pl.nRef), avg(pl.fp, pl.nFP), pl.violations, pl.fpTop20)
	}
}

// flowDiagnose logs the flow quantities of every labelled pair the run
// retrieved, their means per class and per false-positive kind, and adds them
// to the pooled per-class lists.
func flowDiagnose(t *testing.T, name string, run *Run, lf LabelsFile, quant []flowQuantities, byClass map[string][]flowQuantities) {
	t.Helper()
	// Every retrieved pair, suppressed ones included, matched to labels by
	// the scorecard's own rule (firstMatch, file pins and all): a pair the
	// diversity cap hid still has a flow view worth counting.
	retrieved := map[string][]scoredPair{}
	for i, p := range run.Pairs {
		sp := run.scoredPair(p)
		sp.idx = i
		retrieved[sp.key] = append(retrieved[sp.key], sp)
	}
	type acc struct {
		n                          int
		shape, steps, exact, types float64
	}
	add := func(m map[string]*acc, k string, q flowQuantities) {
		a := m[k]
		if a == nil {
			a = &acc{}
			m[k] = a
		}
		a.n++
		a.shape += q.shape
		a.steps += q.steps
		a.exact += q.exact
		a.types += q.types
	}
	classes, kinds := map[string]*acc{}, map[string]*acc{}
	t.Logf("[%s] labelled pairs: shape  steps  exact  types", name)
	for _, l := range lf.Labels {
		sp, ok := firstMatch(retrieved[pairKey(l.A, l.B)], l, run.Root)
		if !ok {
			continue
		}
		q := quant[sp.idx]
		byClass[l.Class] = append(byClass[l.Class], q)
		add(classes, l.Class, q)
		if l.Kind != "" {
			add(kinds, l.Kind, q)
		}
		t.Logf("  %-14s %-15s %.2f   %.2f   %.2f   %.2f  %s <-> %s", l.Class, l.Kind,
			q.shape, q.steps, q.exact, q.types, l.A, l.B)
	}
	logMeans := func(title string, m map[string]*acc) {
		if len(m) == 0 {
			return
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		t.Logf("[%s] means by %s:", name, title)
		for _, k := range keys {
			a := m[k]
			n := float64(a.n)
			t.Logf("  %-18s n=%-3d shape %.2f  steps %.2f  exact %.2f  types %.2f",
				k, a.n, a.shape/n, a.steps/n, a.exact/n, a.types/n)
		}
	}
	logMeans("class", classes)
	logMeans("false-positive kind", kinds)
}

// auc is the Mann-Whitney probability that a value drawn from pos exceeds one
// drawn from neg, ties counting half. NaN-free: empty input reads 0.5.
func auc(pos, neg []float64) float64 {
	if len(pos) == 0 || len(neg) == 0 {
		return 0.5
	}
	var wins float64
	for _, p := range pos {
		for _, n := range neg {
			switch {
			case p > n:
				wins++
			case p == n:
				wins += 0.5
			}
		}
	}
	return wins / float64(len(pos)*len(neg))
}

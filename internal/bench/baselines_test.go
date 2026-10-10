package bench

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/calibrate"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// TestBaselines asks whether doppel's full ranking beats simple baselines on
// the labels the harness has: the committed hand review, the history-derived
// labels, and any private review handed in by env. It is the measurement
// examples/baselines.md reports, and that file's pre-registration states the
// methods, settings, metrics and decision rule this test implements. It
// changes nothing in the pipeline and asserts nothing.
//
//	DOPPEL_BENCH_BASELINES=1 go test ./internal/bench/ -v -run TestBaselines -timeout 60m
//
// Optional:
//
//	DOPPEL_HISTORY=<dir>                    history labels (<corpus>.history.labels.json); default the history script's cache
//	DOPPEL_BENCH_BASELINES_MD=<path>        write the public results tables as markdown
//	DOPPEL_BENCH_BASELINES_EXPORT=<dir>     write every method's ranked list as JSON, public corpora only
//	DOPPEL_BENCH_BASELINES_EXPORT_TOP=<n>   pairs per exported list (default 500, 0 = all)
//	DOPPEL_BENCH_BASELINES_CLONES=<dir>     method 8, an external clone detector's groups (scripts/clone-baseline.sh); see clones_test.go
//
// The export is the hand-off to a cost study: it names each pair by the two
// snapshot.Unit.Key values, so an outcome measured on doppel's pairs can be
// measured on every baseline's pairs the same way.
func TestBaselines(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_BASELINES") != "1" {
		t.Skip("set DOPPEL_BENCH_BASELINES=1 to rank the labels under every baseline")
	}
	targets := baselineTargets(t)
	if only := os.Getenv("DOPPEL_BENCH_BASELINES_ONLY"); only != "" {
		targets = slices.DeleteFunc(targets, func(tg baselineTarget) bool { return !slices.Contains(strings.Split(only, ","), tg.corpus) })
	}
	if len(targets) == 0 {
		t.Skip("no labeled corpora available; run `task corpora` and `task history-labels`")
	}

	md := &strings.Builder{}
	var verdicts []verdictRow
	runs := map[string]*baselineRun{}
	for _, tg := range targets {
		key := tg.root + "\x00" + tg.lf.Population
		br, ok := runs[key]
		if !ok {
			var err error
			br, err = prepareBaselineRun(tg.root, Population(tg.lf.Population))
			if err != nil {
				t.Logf("[%s] %v", tg.display, err)
				continue
			}
			runs[key] = br
			t.Logf("[%s] %d functions, calibration %s, union %d pairs, report pool %d pairs",
				tg.corpus, len(br.run.Units), br.calib, len(br.run.Pairs), len(br.report))
			if tg.public {
				attachClones(t, br, tg.corpus)
			}
		}

		settings := []struct {
			name string
			pool []analyzer.SimilarPair
		}{{"A-union", br.run.Pairs}, {"A-report", br.report}}
		for _, s := range settings {
			res := evaluatePool(br, s.pool, tg.lf)
			logEval(t, tg.display, s.name, len(s.pool), res)
			if tg.public {
				writeEvalMD(md, tg.display, s.name, len(s.pool), res)
				exportRankings(t, br, tg.corpus, s.name, s.pool)
			}
			verdicts = append(verdicts, verdictRow{tg.display, tg.source, s.name, res})
		}
		if allPairsCorpora[tg.corpus] {
			res, n := evaluateAllPairs(br, tg.lf)
			logEval(t, tg.display, "B-all-pairs", n, res)
			if tg.public {
				writeEvalMD(md, tg.display, "B-all-pairs", n, res)
			}
			verdicts = append(verdicts, verdictRow{tg.display, tg.source, "B-all-pairs", res})
		}
	}
	// conc carries no labels of any kind, so it is analysed only for the export.
	for _, c := range Corpora {
		if os.Getenv("DOPPEL_BENCH_BASELINES_EXPORT") == "" || os.Getenv("DOPPEL_BENCH_BASELINES_ONLY") != "" {
			break
		}
		if !allPairsCorpora[c.Name] || !Present(c) || slices.ContainsFunc(targets, func(tg baselineTarget) bool { return tg.corpus == c.Name }) {
			continue
		}
		root, _ := Path(c)
		br, err := prepareBaselineRun(root, PopExclude)
		if err != nil {
			continue
		}
		attachClones(t, br, c.Name)
		exportRankings(t, br, c.Name, "A-union", br.run.Pairs)
		t.Logf("[%s] no labels; ranked lists exported only", c.Name)
	}

	logVerdict(t, md, verdicts)
	writePooled(t, md, verdicts)
	if path := os.Getenv("DOPPEL_BENCH_BASELINES_MD"); path != "" {
		if err := os.WriteFile(path, []byte(md.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// allPairsCorpora are the rungs small enough to score every pair (setting B).
var allPairsCorpora = map[string]bool{"cobra": true, "chi": true, "conc": true, "gin": true}

// historyVerdictCorpora are the four larger corpora the decision rule's second
// condition reads.
var historyVerdictCorpora = []string{"gin", "prometheus", "hugo", "moby"}

type baselineTarget struct {
	corpus  string // ladder name, or the private root's base name
	display string // what the logs call it: corpus/source, or "private-N"
	source  string // "hand" | "history" | "private"
	root    string
	lf      LabelsFile
	public  bool
}

func historyDir() string {
	if d := os.Getenv("DOPPEL_HISTORY"); d != "" {
		return d
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cache, "doppel-git-history")
}

func baselineTargets(t *testing.T) []baselineTarget {
	t.Helper()
	var out []baselineTarget
	committed := committedLabels(t)
	hist := historyDir()
	for _, c := range Corpora {
		if !Present(c) {
			continue
		}
		root, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		if lf, ok := committed[c.Name]; ok {
			out = append(out, baselineTarget{c.Name, c.Name + "/hand", "hand", root, lf, true})
		}
		data, err := os.ReadFile(filepath.Join(hist, c.Name+".history.labels.json"))
		if err != nil {
			continue
		}
		lf, err := ParseLabels(data)
		if err != nil {
			t.Logf("[%s] history labels unusable: %v", c.Name, err)
			continue
		}
		out = append(out, baselineTarget{c.Name, c.Name + "/history", "history", root, lf, true})
	}

	// Private reviews: scored, logged as aggregates under an opaque name, and
	// never written to the markdown file or the export.
	n := 0
	addPrivate := func(root, labels string) {
		data, err := os.ReadFile(labels)
		if err != nil {
			return
		}
		lf, err := ParseLabels(data)
		if err != nil {
			t.Fatalf("private labels: %v", err)
		}
		n++
		out = append(out, baselineTarget{filepath.Base(root), "private-" + strconv.Itoa(n), "private", root, lf, false})
	}
	if root, labels := os.Getenv("DOPPEL_BENCH_CORPUS"), os.Getenv("DOPPEL_BENCH_LABELS"); root != "" && labels != "" {
		addPrivate(root, labels)
	}
	if dir := os.Getenv("DOPPEL_BENCH_LENSES_LABELS"); dir != "" {
		for _, root := range filepath.SplitList(os.Getenv("DOPPEL_BENCH_LENSES_EXTRA")) {
			addPrivate(root, filepath.Join(dir, filepath.Base(root)+".labels.json"))
		}
	}
	return out
}

// baselineRun is one corpus analysed at the production operating point.
type baselineRun struct {
	run    *Run
	calib  string
	report []analyzer.SimilarPair // the union at or above the calibrated struct-min
	keys   []string               // snapshot.Unit.Key per unit
	clones []cloneList            // method 8, public corpora with clone files only
}

func attachClones(t *testing.T, br *baselineRun, corpus string) {
	t.Helper()
	lists, notes, err := loadCloneLists(br, corpus)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range notes {
		t.Logf("[%s] %s", corpus, n)
	}
	br.clones = lists
}

func prepareBaselineRun(root string, pop Population) (*baselineRun, error) {
	units, err := Load(root, pop)
	if err != nil {
		return nil, err
	}
	return prepareBaselineRunOf(root, units)
}

// prepareBaselineRunOf is prepareBaselineRun over a population already loaded.
func prepareBaselineRunOf(root string, units []parser.CodeUnit) (*baselineRun, error) {
	if len(units) == 0 {
		return nil, fmt.Errorf("no functions under %s", root)
	}
	run := Analyze(units, retriever.DefaultOptions())
	run.Root = root
	opt := retriever.DefaultOptions()
	cr := calibrate.Run(run.Units, run.Docs, run.Comp, run.WL, calibrate.DefaultOptions(0.01, opt.MinNodes))
	br := &baselineRun{run: run}
	structMin := 0.0
	if cr.Applied() {
		opt.Threshold = cr.Threshold
		structMin = cr.StructMin
		run.Reretrieve(opt)
		br.calib = fmt.Sprintf("threshold %.2f struct-min %.2f", cr.Threshold, cr.StructMin)
	} else {
		br.calib = "declined (" + cr.Declined + "); static defaults"
	}
	for _, p := range run.Pairs {
		if p.Evidence != nil && p.Evidence.OverlapScore >= structMin {
			br.report = append(br.report, p)
		}
	}
	br.keys = snapshot.Keys(run.Units, root)
	return br, nil
}

// ref is one ranked pair, by unit index.
type ref struct{ a, b int }

func refOf(p analyzer.SimilarPair) ref {
	if p.AIdx > p.BIdx {
		return ref{p.BIdx, p.AIdx}
	}
	return ref{p.AIdx, p.BIdx}
}

// rankBy orders refs by key descending, ties by (a, b) ascending — the same
// tie-break for every method. A NaN key leaves the ref out of the ranking.
func rankBy(refs []ref, key []float64) []ref {
	idx := make([]int, 0, len(refs))
	for i := range refs {
		if !math.IsNaN(key[i]) {
			idx = append(idx, i)
		}
	}
	sort.SliceStable(idx, func(x, y int) bool {
		i, j := idx[x], idx[y]
		if key[i] != key[j] {
			return key[i] > key[j]
		}
		if refs[i].a != refs[j].a {
			return refs[i].a < refs[j].a
		}
		return refs[i].b < refs[j].b
	})
	out := make([]ref, len(idx))
	for k, i := range idx {
		out[k] = refs[i]
	}
	return out
}

func doppelOrder(r *Run, pool []analyzer.SimilarPair) []ref {
	cp := slices.Clone(pool)
	kept, _ := analyzer.SortForReportWith(cp, r.Units, 0, 0, analyzer.DefaultRankOptions())
	out := make([]ref, len(kept))
	for i, p := range kept {
		out[i] = refOf(p)
	}
	return out
}

// shingleJaccard is the token-clone baseline: set Jaccard over two sorted,
// deduped shingle lists.
func shingleJaccard(a, b []uint64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	i, j, inter := 0, 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			inter++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return float64(inter) / float64(len(a)+len(b)-inter)
}

// nameStems splits an identifier at camel-case, digit and underscore
// boundaries, lower-cased.
func nameStems(s string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			out = append(out, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	rs := []rune(s)
	for i, r := range rs {
		switch {
		case r == '_' || r == '.' || r == '*':
			flush()
			continue
		case unicode.IsUpper(r) && i > 0 && (unicode.IsLower(rs[i-1]) || (i+1 < len(rs) && unicode.IsLower(rs[i+1]) && unicode.IsUpper(rs[i-1]))):
			flush()
		case unicode.IsDigit(r) != (i > 0 && unicode.IsDigit(rs[i-1])) && i > 0:
			flush()
		}
		cur = append(cur, r)
	}
	flush()
	return out
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	cur := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			c := 1
			if ra[i-1] == rb[j-1] {
				c = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+c)
		}
		prev, cur = cur, prev
	}
	return prev[len(rb)]
}

// nameHeuristic is the cheap baseline: name similarity times size similarity,
// defined only within one package (same package clause, same directory). NaN
// outside it, which leaves the pair unranked.
func nameHeuristic(a, b parser.CodeUnit) float64 {
	if a.Package != b.Package || filepath.Dir(a.File) != filepath.Dir(b.File) {
		return math.NaN()
	}
	na, nb := parser.MethodName(a), parser.MethodName(b)
	sa, sb := nameStems(na), nameStems(nb)
	inter := 0
	set := map[string]bool{}
	for _, s := range sa {
		set[s] = true
	}
	seen := map[string]bool{}
	for _, s := range sb {
		if set[s] && !seen[s] {
			inter++
		}
		seen[s] = true
	}
	union := len(set) + len(seen) - inter
	jac := 0.0
	if union > 0 {
		jac = float64(inter) / float64(union)
	}
	la, lb := strings.ToLower(na), strings.ToLower(nb)
	edit := 0.0
	if m := max(len([]rune(la)), len([]rune(lb))); m > 0 {
		edit = 1 - float64(levenshtein(la, lb))/float64(m)
	}
	x, y := a.Fingerprint.Nodes, b.Fingerprint.Nodes
	size := 0.0
	if x > 0 && y > 0 {
		size = float64(min(x, y)) / float64(max(x, y))
	}
	return math.Max(jac, edit) * size
}

// methodSet is every ranked list of one evaluation, in display order, plus the
// random lists.
type methodSet struct {
	names  []string
	lists  [][]ref
	random [][]ref
}

const randomSeeds = 20

var methodOrder = []string{
	"doppel", "doppel (struct-min filtered)", "token clones", "code-shape",
	"retrieval mass", "overlap", "name heuristic",
	cloneMethods[0].name, cloneMethods[1].name,
}

func (m *methodSet) add(name string, l []ref) {
	m.names = append(m.names, name)
	m.lists = append(m.lists, l)
}

func shuffled(refs []ref, seed uint64) []ref {
	out := slices.Clone(refs)
	r := rand.New(rand.NewPCG(seed, 0x9e3779b97f4a7c15))
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// poolMethods ranks one pool under every method (setting A).
func poolMethods(br *baselineRun, pool []analyzer.SimilarPair) methodSet {
	r := br.run
	refs := make([]ref, len(pool))
	shingle := make([]float64, len(pool))
	shape := make([]float64, len(pool))
	mass := make([]float64, len(pool))
	overlap := make([]float64, len(pool))
	name := make([]float64, len(pool))
	for i, p := range pool {
		refs[i] = refOf(p)
		a, b := r.Units[p.AIdx], r.Units[p.BIdx]
		shingle[i] = shingleJaccard(a.Fingerprint.Shingles, b.Fingerprint.Shingles)
		shape[i] = p.Score
		if p.Retrieval != nil {
			mass[i] = p.Retrieval.Total
		}
		if p.Evidence != nil {
			overlap[i] = p.Evidence.OverlapScore
		}
		name[i] = nameHeuristic(a, b)
	}
	var m methodSet
	full := doppelOrder(r, pool)
	m.add("doppel", full)
	m.add("doppel (struct-min filtered)", filterToReport(br, full))
	m.add("token clones", rankBy(refs, shingle))
	m.add("code-shape", rankBy(refs, shape))
	m.add("retrieval mass", rankBy(refs, mass))
	m.add("overlap", rankBy(refs, overlap))
	m.add("name heuristic", rankBy(refs, name))
	in := make(map[ref]bool, len(refs))
	for _, x := range refs {
		in[x] = true
	}
	for _, c := range br.clones {
		m.add(c.name, rankBy(c.restrictTo(in)))
	}
	base := rankBy(refs, make([]float64, len(refs))) // index order, the shuffle's start
	for s := uint64(1); s <= randomSeeds; s++ {
		m.random = append(m.random, shuffled(base, s))
	}
	return m
}

func filterToReport(br *baselineRun, l []ref) []ref {
	in := make(map[ref]bool, len(br.report))
	for _, p := range br.report {
		in[refOf(p)] = true
	}
	out := make([]ref, 0, len(br.report))
	for _, x := range l {
		if in[x] {
			out = append(out, x)
		}
	}
	return out
}

// evalResult is one method's scorecard against one labels file.
type evalResult struct {
	name     string
	listLen  int
	present  map[string]int
	meanRank map[string]float64 // absent labels imputed, see impute
	relN     int
	relHave  int     // relevant labels the list ranks at all
	relMed   float64 // median rank of the relevant labels (post-hoc), absent ones imputed
	pool     int     // pairs the setting could rank (the imputation's n)
	relMean  float64
	p10, p20 float64
	p50      float64
	hits50   int
	hits100  int
	// hard assertions, meaningful where the labels carry merges and false positives
	mergeMissing, fpAboveMerge, fpTop20 int
}

// randomAgg is the mean and standard deviation over the random seeds.
type randomAgg struct {
	mean, sd evalResult
}

type poolEval struct {
	methods []evalResult
	random  randomAgg
}

// scoreList ranks lf against one ordered list. n is the number of pairs the
// setting could rank at all; an absent label takes the expected rank of a
// random completion of the list, (len+1+n)/2.
func scoreList(r *Run, l []ref, n int, lf LabelsFile, name string) evalResult {
	byKey := make(map[string][]scoredPair, len(l))
	for i, x := range l {
		sp := r.scoredPair(analyzer.SimilarPair{AIdx: x.a, BIdx: x.b})
		sp.rank = i + 1
		byKey[sp.key] = append(byKey[sp.key], sp)
	}
	impute := float64(len(l)+1+n) / 2
	res := evalResult{name: name, listLen: len(l), pool: n, present: map[string]int{}, meanRank: map[string]float64{}}
	sum := map[string]float64{}
	cnt := map[string]int{}
	relSum := 0.0
	var relRanks []float64
	worstMerge := 0
	var fps []int
	for _, lab := range lf.Labels {
		rank := 0
		if sp, ok := firstMatch(byKey[pairKey(lab.A, lab.B)], lab, r.Root); ok {
			rank = sp.rank
			res.present[lab.Class]++
		}
		rk := impute
		if rank > 0 {
			rk = float64(rank)
		}
		sum[lab.Class] += rk
		cnt[lab.Class]++
		relevant := lab.Class == "merge" || lab.Class == "refactor"
		if relevant {
			relSum += rk
			relRanks = append(relRanks, rk)
			res.relN++
			if rank > 0 {
				res.relHave++
			}
			if rank > 0 && rank <= 10 {
				res.p10++
			}
			if rank > 0 && rank <= 20 {
				res.p20++
			}
			if rank > 0 && rank <= 50 {
				res.p50++
				res.hits50++
			}
			if rank > 0 && rank <= 100 {
				res.hits100++
			}
		}
		switch lab.Class {
		case "merge":
			if rank == 0 {
				res.mergeMissing++
			} else if rank > worstMerge {
				worstMerge = rank
			}
		case "false_positive":
			if rank > 0 {
				fps = append(fps, rank)
				if rank <= 20 {
					res.fpTop20++
				}
			}
		}
	}
	for _, f := range fps {
		if worstMerge > 0 && f < worstMerge {
			res.fpAboveMerge++
		}
	}
	for c, s := range sum {
		res.meanRank[c] = s / float64(cnt[c])
	}
	if res.relN > 0 {
		res.relMean = relSum / float64(res.relN)
		sort.Float64s(relRanks)
		if k := len(relRanks); k%2 == 1 {
			res.relMed = relRanks[k/2]
		} else {
			res.relMed = (relRanks[k/2-1] + relRanks[k/2]) / 2
		}
	}
	res.p10 /= 10
	res.p20 /= 20
	res.p50 /= 50
	return res
}

func evaluate(r *Run, m methodSet, n int, lf LabelsFile) poolEval {
	var pe poolEval
	for i, l := range m.lists {
		pe.methods = append(pe.methods, scoreList(r, l, n, lf, m.names[i]))
	}
	var rs []evalResult
	for _, l := range m.random {
		rs = append(rs, scoreList(r, l, n, lf, "random"))
	}
	pe.random = aggregate(rs)
	return pe
}

func evaluatePool(br *baselineRun, pool []analyzer.SimilarPair, lf LabelsFile) poolEval {
	return evaluate(br.run, poolMethods(br, pool), len(pool), lf)
}

// evaluateAllPairs is setting B: every same-build-unit pair of the population.
// Methods that exist only on retrieved pairs rank the union and leave the rest
// unranked.
func evaluateAllPairs(br *baselineRun, lf LabelsFile) (poolEval, int) {
	r := br.run
	var refs []ref
	for a := range r.Units {
		for b := a + 1; b < len(r.Units); b++ {
			if parser.SameBuildUnit(r.Units[a], r.Units[b]) {
				refs = append(refs, ref{a, b})
			}
		}
	}
	shingle := make([]float64, len(refs))
	shape := make([]float64, len(refs))
	overlap := make([]float64, len(refs))
	name := make([]float64, len(refs))
	for i, x := range refs {
		a, b := r.Units[x.a], r.Units[x.b]
		shingle[i] = shingleJaccard(a.Fingerprint.Shingles, b.Fingerprint.Shingles)
		shape[i] = fingerprint.Similarity(a.Fingerprint, b.Fingerprint, r.WL).Score
		overlap[i] = r.Comp.Compare(r.Docs[x.a], r.Docs[x.b]).OverlapScore
		name[i] = nameHeuristic(a, b)
	}
	union := r.Pairs
	massRefs := make([]ref, len(union))
	mass := make([]float64, len(union))
	for i, p := range union {
		massRefs[i] = refOf(p)
		if p.Retrieval != nil {
			mass[i] = p.Retrieval.Total
		}
	}
	var m methodSet
	full := doppelOrder(r, union)
	m.add("doppel", full)
	m.add("doppel (struct-min filtered)", filterToReport(br, full))
	m.add("token clones", rankBy(refs, shingle))
	m.add("code-shape", rankBy(refs, shape))
	m.add("retrieval mass", rankBy(massRefs, mass))
	m.add("overlap", rankBy(refs, overlap))
	m.add("name heuristic", rankBy(refs, name))
	// Every clone pair is already a same-build-unit pair of the population,
	// so the detector ranks what it reported and leaves the rest unranked.
	for _, c := range br.clones {
		m.add(c.name, rankBy(c.refs, c.key))
	}
	for s := uint64(1); s <= randomSeeds; s++ {
		m.random = append(m.random, shuffled(refs, s))
	}
	return evaluate(r, m, len(refs), lf), len(refs)
}

func aggregate(rs []evalResult) randomAgg {
	var agg randomAgg
	if len(rs) == 0 {
		return agg
	}
	field := func(get func(e evalResult) float64) (float64, float64) {
		var s, ss float64
		for _, e := range rs {
			v := get(e)
			s += v
			ss += v * v
		}
		mean := s / float64(len(rs))
		return mean, math.Sqrt(math.Max(0, ss/float64(len(rs))-mean*mean))
	}
	agg.mean = evalResult{name: "random", present: rs[0].present, meanRank: map[string]float64{}, relN: rs[0].relN, relHave: rs[0].relHave, listLen: rs[0].listLen, pool: rs[0].pool}
	agg.sd = evalResult{name: "random sd", present: rs[0].present, meanRank: map[string]float64{}, relN: rs[0].relN}
	agg.mean.relMean, agg.sd.relMean = field(func(e evalResult) float64 { return e.relMean })
	agg.mean.relMed, agg.sd.relMed = field(func(e evalResult) float64 { return e.relMed })
	agg.mean.p10, agg.sd.p10 = field(func(e evalResult) float64 { return e.p10 })
	agg.mean.p20, agg.sd.p20 = field(func(e evalResult) float64 { return e.p20 })
	agg.mean.p50, agg.sd.p50 = field(func(e evalResult) float64 { return e.p50 })
	// Counts are reported as the seed mean, rounded.
	count := func(get func(e evalResult) int) int {
		m, _ := field(func(e evalResult) float64 { return float64(get(e)) })
		return int(math.Round(m))
	}
	agg.mean.hits50 = count(func(e evalResult) int { return e.hits50 })
	agg.mean.hits100 = count(func(e evalResult) int { return e.hits100 })
	agg.mean.fpTop20 = count(func(e evalResult) int { return e.fpTop20 })
	agg.mean.fpAboveMerge = count(func(e evalResult) int { return e.fpAboveMerge })
	agg.mean.mergeMissing = count(func(e evalResult) int { return e.mergeMissing })
	for _, c := range Classes {
		if _, ok := rs[0].meanRank[c]; !ok {
			continue
		}
		agg.mean.meanRank[c], agg.sd.meanRank[c] = field(func(e evalResult) float64 { return e.meanRank[c] })
	}
	return agg
}

// pickMethod finds one method's result in an evaluation; random is the seed
// mean. ok is false for a method the evaluation did not run.
func pickMethod(pe poolEval, name string) (evalResult, bool) {
	if name == "random" {
		return pe.random.mean, true
	}
	for _, e := range pe.methods {
		if e.name == name {
			return e, true
		}
	}
	return evalResult{}, false
}

func fmtMean(e evalResult, class string) string {
	v, ok := e.meanRank[class]
	if !ok {
		return "-"
	}
	return fmt.Sprintf("%.1f", v)
}

func classesIn(lf LabelsFile) []string {
	var out []string
	for _, c := range Classes {
		if slices.ContainsFunc(lf.Labels, func(l Label) bool { return l.Class == c }) {
			out = append(out, c)
		}
	}
	return out
}

func evalRows(pe poolEval) []evalResult {
	rows := slices.Clone(pe.methods)
	rows = append(rows, pe.random.mean)
	return rows
}

func logEval(t *testing.T, display, setting string, n int, pe poolEval) {
	t.Helper()
	t.Logf("[%s] %s (%d pairs)", display, setting, n)
	for _, e := range evalRows(pe) {
		t.Logf("  %-30s ranked %6d  rel %d mean %7.1f  P@10 %.2f P@20 %.2f P@50 %.2f  hits50 %d hits100 %d  merge %s refactor %s fp %s coupled %s  missing %d fp>merge %d fp@20 %d",
			e.name, e.listLen, e.relN, e.relMean, e.p10, e.p20, e.p50, e.hits50, e.hits100,
			fmtMean(e, "merge"), fmtMean(e, "refactor"), fmtMean(e, "false_positive"), fmtMean(e, "coupled"),
			e.mergeMissing, e.fpAboveMerge, e.fpTop20)
	}
	sd := pe.random.sd
	t.Logf("  %-30s rel mean ±%.1f  P@20 ±%.2f", "random (sd over 20 seeds)", sd.relMean, sd.p20)
}

func writeEvalMD(md *strings.Builder, display, setting string, n int, pe poolEval) {
	fmt.Fprintf(md, "#### %s — %s (%d pairs)\n\n", display, setting, n)
	md.WriteString("| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |\n")
	md.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, e := range evalRows(pe) {
		name := e.name
		rel := fmt.Sprintf("%.1f (%d/%d)", e.relMean, e.relHave, e.relN)
		med := fmt.Sprintf("%.1f", e.relMed)
		p20 := fmt.Sprintf("%.2f", e.p20)
		if e.name == "random" {
			name = "random (20 seeds, mean)"
			rel = fmt.Sprintf("%.1f ±%.1f (%d/%d)", e.relMean, pe.random.sd.relMean, e.relHave, e.relN)
			p20 = fmt.Sprintf("%.2f ±%.2f", e.p20, pe.random.sd.p20)
		}
		fmt.Fprintf(md, "| %s | %d | %s | %s | %.2f | %s | %.2f | %d | %d | %s | %s | %s | %s | %d | %d | %d |\n",
			name, e.listLen, rel, med, e.p10, p20, e.p50, e.hits50, e.hits100,
			fmtMean(e, "merge"), fmtMean(e, "refactor"), fmtMean(e, "false_positive"), fmtMean(e, "coupled"),
			e.mergeMissing, e.fpAboveMerge, e.fpTop20)
	}
	md.WriteString("\n")
}

type verdictRow struct {
	display, source, setting string
	pe                       poolEval
}

// beats applies the pre-registered comparison to one doppel/baseline pair of
// results. strict: precision@20 strictly higher; otherwise not lower.
func beats(d, b evalResult, strict, checkFP bool) bool {
	p20 := d.p20 > b.p20
	if !strict {
		p20 = d.p20 >= b.p20
	}
	ok := p20 && d.relN > 0 && d.relMean < b.relMean
	if checkFP {
		df, okd := d.meanRank["false_positive"]
		bf, okb := b.meanRank["false_positive"]
		if okd && okb && df < bf {
			ok = false
		}
	}
	return ok
}

func logVerdict(t *testing.T, md *strings.Builder, rows []verdictRow) {
	t.Helper()
	find := func(display, setting string) *poolEval {
		for i := range rows {
			if rows[i].display == display && rows[i].setting == setting {
				return &rows[i].pe
			}
		}
		return nil
	}
	baselines := append(slices.Clone(methodOrder[2:]), "random")
	pick := func(pe *poolEval, name string) evalResult {
		e, _ := pickMethod(*pe, name)
		return e
	}
	has := func(pe *poolEval, name string) bool {
		if pe == nil {
			return false
		}
		_, ok := pickMethod(*pe, name)
		return ok
	}
	md.WriteString("#### Decision rule, applied\n\n")
	md.WriteString("| baseline | cobra hand, A-union | history wins (A-report), strict | history wins, weak (P@20 not lower) | beaten (strict) | beaten (weak) |\n")
	md.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, bl := range baselines {
		// A method that was not run (method 8 without clone files) is left
		// out, not scored as an empty list.
		if !slices.ContainsFunc(rows, func(r verdictRow) bool { return has(&r.pe, bl) }) {
			continue
		}
		cobra := find("cobra/hand", "A-union")
		c1s, c1w := false, false
		parts := "no cobra labels"
		if cobra != nil && !has(cobra, bl) {
			parts = "not run on cobra"
		}
		if cobra != nil && has(cobra, bl) {
			d := pick(cobra, "doppel")
			b := pick(cobra, bl)
			c1s, c1w = beats(d, b, true, true), beats(d, b, false, true)
			mark := func(v bool) string {
				if v {
					return "✓"
				}
				return "✗"
			}
			parts = fmt.Sprintf("P@20 %.2f vs %.2f %s; rel %.1f vs %.1f %s; fp %.1f vs %.1f %s",
				d.p20, b.p20, mark(d.p20 > b.p20), d.relMean, b.relMean, mark(d.relMean < b.relMean),
				d.meanRank["false_positive"], b.meanRank["false_positive"],
				mark(d.meanRank["false_positive"] >= b.meanRank["false_positive"]))
		}
		var winsS, winsW, avail []string
		for _, c := range historyVerdictCorpora {
			pe := find(c+"/history", "A-report")
			if pe == nil || !has(pe, bl) {
				continue
			}
			avail = append(avail, c)
			d, b := pick(pe, "doppel"), pick(pe, bl)
			if beats(d, b, true, false) {
				winsS = append(winsS, c)
			}
			if beats(d, b, false, false) {
				winsW = append(winsW, c)
			}
		}
		yes := func(v bool) string {
			if v {
				return "yes"
			}
			return "no"
		}
		list := func(w []string) string {
			return fmt.Sprintf("%d/%d %s", len(w), len(avail), strings.Join(w, ","))
		}
		strict := c1s && len(winsS) >= 3
		weak := c1w && len(winsW) >= 3
		line := fmt.Sprintf("| %s | %s — strict %s, weak %s | %s | %s | **%s** | %s |",
			bl, parts, yes(c1s), yes(c1w), list(winsS), list(winsW), yes(strict), yes(weak))
		md.WriteString(line + "\n")
		t.Log(line)
	}
	md.WriteString("\n")
}

// exportRankings writes every method's ranked list for one corpus and pool,
// keyed by snapshot.Unit.Key, for a later study to score on its own outcome.
func exportRankings(t *testing.T, br *baselineRun, corpus, setting string, pool []analyzer.SimilarPair) {
	t.Helper()
	dir := os.Getenv("DOPPEL_BENCH_BASELINES_EXPORT")
	if dir == "" {
		return
	}
	top := 500
	if s := os.Getenv("DOPPEL_BENCH_BASELINES_EXPORT_TOP"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			top = n
		}
	}
	type pairOut struct {
		Rank int    `json:"rank"`
		A    string `json:"a"`
		B    string `json:"b"`
	}
	type listOut struct {
		Corpus string    `json:"corpus"`
		Pool   string    `json:"pool"`
		Method string    `json:"method"`
		Pairs  []pairOut `json:"pairs"`
	}
	m := poolMethods(br, pool)
	names := append(slices.Clone(m.names), "random (seed 1)")
	lists := append(slices.Clone(m.lists), m.random[0])
	out := make([]listOut, 0, len(lists))
	for i, l := range lists {
		if top > 0 && len(l) > top {
			l = l[:top]
		}
		lo := listOut{Corpus: corpus, Pool: setting, Method: names[i], Pairs: make([]pairOut, len(l))}
		for k, x := range l {
			a, b := br.keys[x.a], br.keys[x.b]
			if a > b {
				a, b = b, a
			}
			lo.Pairs[k] = pairOut{k + 1, a, b}
		}
		out = append(out, lo)
	}
	data, err := json.MarshalIndent(out, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, corpus+"."+setting+".rankings.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writePooled is the post-hoc pooled view: per method, the history label sets'
// relevant (refactor) and coupled mean ranks as a fraction of the pool, averaged
// over corpora so a large pool does not dominate; and setting B's hits summed
// over every labelled small corpus.
func writePooled(t *testing.T, md *strings.Builder, rows []verdictRow) {
	t.Helper()
	names := append(slices.Clone(methodOrder), "random")
	pick := func(pe poolEval, name string) evalResult {
		e, _ := pickMethod(pe, name)
		return e
	}
	for _, setting := range []string{"A-union", "A-report"} {
		fmt.Fprintf(md, "#### Pooled (post-hoc) — history labels, %s\n\n", setting)
		md.WriteString("Mean rank divided by pool size, averaged over the corpora listed; 0 is the top of the list, 0.5 is what random ordering gives.\n\n")
		md.WriteString("| method | refactor (normalised mean) | coupled (normalised mean) | corpora where it beats doppel on refactor mean |\n| --- | ---: | ---: | --- |\n")
		for _, name := range names {
			var rel, cpl float64
			var nr, nc int
			var corpora, wins []string
			for _, r := range rows {
				if r.source != "history" || r.setting != setting {
					continue
				}
				e := pick(r.pe, name)
				d := pick(r.pe, "doppel")
				c := strings.TrimSuffix(r.display, "/history")
				if e.relN > 0 && e.pool > 0 {
					rel += e.relMean / float64(e.pool)
					nr++
					corpora = append(corpora, c)
					if name != "doppel" && e.relMean < d.relMean {
						wins = append(wins, c)
					}
				}
				if v, ok := e.meanRank["coupled"]; ok && e.pool > 0 {
					cpl += v / float64(e.pool)
					nc++
				}
			}
			if nr == 0 {
				continue
			}
			w := "—"
			if name != "doppel" {
				w = fmt.Sprintf("%d/%d %s", len(wins), len(corpora), strings.Join(wins, ", "))
			}
			line := fmt.Sprintf("| %s | %.3f | %.3f | %s |", name, rel/float64(nr), cpl/float64(max(nc, 1)), w)
			md.WriteString(line + "\n")
			t.Logf("pooled %s %s", setting, line)
		}
		md.WriteString("\n")
	}
	md.WriteString("#### Pooled — setting B, labelled relevant pairs in the top 50 / top 100\n\n")
	md.WriteString("Summed over every label set scored all-pairs (cobra hand, and the cobra, chi and gin history labels).\n\n| method | top 50 | top 100 | of |\n| --- | ---: | ---: | ---: |\n")
	for _, name := range names {
		var h50, h100, n int
		ran := false
		for _, r := range rows {
			if r.setting != "B-all-pairs" {
				continue
			}
			e, ok := pickMethod(r.pe, name)
			ran = ran || ok
			h50 += e.hits50
			h100 += e.hits100
			n += e.relN
		}
		if !ran {
			continue
		}
		fmt.Fprintf(md, "| %s | %d | %d | %d |\n", name, h50, h100, n)
	}
	md.WriteString("\n")
}

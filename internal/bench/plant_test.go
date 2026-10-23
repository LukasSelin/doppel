package bench

import (
	"fmt"
	"go/ast"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/calibrate"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/gofront"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// TestPlant is plant-and-recover: it copies real functions of a real corpus,
// mutates the copies by operators whose effect is known by construction, and
// measures whether doppel finds each copy's donor. Truth comes from the
// construction, never from a judgment, which is what makes this tier cheap
// and hard where the hand labels are soft and the outcome studies slow.
//
//	DOPPEL_BENCH_PLANT=1 go test ./internal/bench/ -v -run TestPlant
//
// DOPPEL_BENCH_PLANT_SEED picks another donor draw (default 1). Hosts are the
// small fetched rungs — cobra, chi, conc, gin — and doppel's own tree; a rung
// that is not fetched is skipped.
//
// Operators come in three families (plant_ops_test.go): canon-covered
// rewrites one canon rule undoes, equivalence-preserving rewrites no rule
// undoes, and drift at a dose of 1, 2 and 4 edits.
//
// Two levels. At the query level each plant joins the corpus as an extra unit
// and is probed through retriever.Prober, ranked by retriever.ProbeLess —
// what `doppel query` does — and the donor's rank is recorded. At the report
// level the plant goes into the host as a file of its own, optionally with a
// fraction of the donor's callers rewired to call it (a fork with callers of
// its own), the pipeline runs at analyze's defaults (calibrated thresholds,
// top 20, two pairs per function), and the plant↔donor pair's report rank is
// recorded, or the stage that dropped it. Either way a pipeline run carries
// at most about 1% of the host in plants, so the plants do not move the
// corpus statistics they are scored against, and never two plants of one
// donor.
//
// It asserts two things only: a canon-covered plant's WL bag equals its
// donor's (and, for a rewrite that leaves the written body's histograms
// alone, so does its code-shape), and the same seed reproduces the smallest
// host's results byte for byte. Everything else is logged. Plants measure
// recall of known forks; they say nothing about false-positive precision,
// which stays with the hand labels.
func TestPlant(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_PLANT") != "1" {
		t.Skip("set DOPPEL_BENCH_PLANT=1 to run plant-and-recover")
	}
	opt := defaultPlantOptions()
	if s := os.Getenv("DOPPEL_BENCH_PLANT_SEED"); s != "" {
		seed, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			t.Fatalf("DOPPEL_BENCH_PLANT_SEED: %v", err)
		}
		opt.Seed = seed
	}

	type target struct{ name, root string }
	var targets []target
	for _, name := range []string{"cobra", "chi", "conc", "gin"} {
		c, ok := Find(name)
		if !ok || !Present(c) {
			t.Logf("[%s] not fetched, skipped", name)
			continue
		}
		root, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		targets = append(targets, target{name, root})
	}
	targets = append(targets, target{"doppel", repoRoot(t)})

	var all []plantResult
	smallest, smallestN := -1, math.MaxInt
	for i, tg := range targets {
		h, err := loadPlantHost(tg.name, tg.root)
		if err != nil {
			t.Fatalf("[%s] %v", tg.name, err)
		}
		start := time.Now()
		res := measurePlants(t, h, opt)
		t.Logf("[%s] %d functions, %d donors, %d query plants in %d runs, %d report plants in %d runs, %.1fs",
			tg.name, len(h.units), res.donors, len(res.query), res.queryRuns, len(res.report), res.reportRuns,
			time.Since(start).Seconds())
		t.Log("\n" + res.render())
		checkCanon(t, res)
		all = append(all, res)
		if len(h.units) < smallestN {
			smallest, smallestN = i, len(h.units)
		}
	}
	t.Log("\n" + renderPooled(all))

	// Determinism: the same seed must reproduce a host's results byte for
	// byte. Re-running the smallest host keeps the check inside the budget;
	// every code path the others take, it takes too.
	if smallest >= 0 {
		tg := targets[smallest]
		h, err := loadPlantHost(tg.name, tg.root)
		if err != nil {
			t.Fatal(err)
		}
		again := measurePlants(t, h, opt)
		if a, b := all[smallest].render(), again.render(); a != b {
			t.Errorf("[%s] a second run with seed %d rendered differently:\n--- first\n%s\n--- second\n%s", tg.name, opt.Seed, a, b)
		} else {
			t.Logf("[%s] determinism: a second run with seed %d is byte-identical", tg.name, opt.Seed)
		}
	}
}

// plantOptions are the harness's knobs, passed explicitly so a variant run is
// an argument and never a package global.
type plantOptions struct {
	Seed   uint64
	Donors int // donors drawn per host
	// MinPerOp tops up a rare canon-covered or equivalence operator: one
	// that found a site in fewer drawn donors than this walks on down the
	// same seeded donor order until it has this many plants or runs out.
	// Without it an operator like negated-if — a plain else with a bare
	// condition is uncommon Go — would rest the canon assertion on one plant.
	MinPerOp           int
	ReportDonors       int     // of those, how many are also planted at report level
	MinNodes, MaxNodes int     // donor body size, in syntax nodes
	Doses              []int   // drift edits per plant
	Growth             float64 // plants per pipeline run, as a fraction of the host (at least one)
	RewireFraction     float64 // of a donor's callers, how many a wired plant takes over
	Cuts               []int   // query recall cut-offs

	// The analyze defaults the report level runs at — cmd's own, restated
	// because bench cannot import cmd.
	Calibrate   float64
	MinNodesRet int
	ChannelK    int
	QueryK      int // doppel query's per-channel budget, cmd.probeChannelK
	Top         int
	MaxPerFunc  int
}

func defaultPlantOptions() plantOptions {
	return plantOptions{
		Seed:           1,
		Donors:         24,
		MinPerOp:       8,
		ReportDonors:   12,
		MinNodes:       20,
		MaxNodes:       60,
		Doses:          []int{1, 2, 4},
		Growth:         0.01,
		RewireFraction: 0.5,
		Cuts:           []int{1, 5, 20},
		Calibrate:      0.01,
		MinNodesRet:    retriever.DefaultOptions().MinNodes,
		ChannelK:       retriever.DefaultOptions().ChannelK,
		QueryK:         10,
		Top:            20,
		MaxPerFunc:     2,
	}
}

// plantHost is one host corpus, loaded once.
type plantHost struct {
	name, root string
	units      []parser.CodeUnit
	files      map[string][2]int // file -> its units' [lo, hi) in units
	graph      *concepter.Graph
	byQual     map[string]int // qualified name -> unit index, unique names only
	sib        *siblingTable
	src        map[string][]byte
}

func loadPlantHost(name, root string) (*plantHost, error) {
	units, err := Load(root, PopExclude)
	if err != nil {
		return nil, err
	}
	h := &plantHost{name: name, root: root, units: units,
		files: map[string][2]int{}, byQual: map[string]int{}, src: map[string][]byte{}}
	for i := 0; i < len(units); {
		j := i
		for j < len(units) && units[j].File == units[i].File {
			j++
		}
		h.files[units[i].File] = [2]int{i, j}
		i = j
	}
	h.graph = concepter.BuildCallGraph(units)
	seen := map[string]int{}
	for i, u := range units {
		seen[concepter.QualifiedName(u)]++
		h.byQual[concepter.QualifiedName(u)] = i
	}
	for q, n := range seen {
		if n > 1 {
			delete(h.byQual, q)
		}
	}
	h.sib = buildSiblings(h)
	return h, nil
}

func (h *plantHost) source(file string) []byte {
	if b, ok := h.src[file]; ok {
		return b
	}
	b, err := os.ReadFile(file)
	if err != nil {
		b = nil
	}
	h.src[file] = b
	return b
}

// key names a unit the way a reader finds it: package.Name @ root-relative
// file. It is what donor priorities hash, so a draw does not depend on where
// the host is checked out.
func (h *plantHost) key(i int) string {
	u := h.units[i]
	return u.Package + "." + u.Name + " @ " + parser.RelSlash(h.root, u.File)
}

func dirOf(file string) string { return filepath.Dir(file) }

// hash64 is FNV-1a over the parts, NUL-separated.
func hash64(parts ...string) uint64 {
	f := fnv.New64a()
	for _, p := range parts {
		_, _ = f.Write([]byte(p))
		_, _ = f.Write([]byte{0})
	}
	return f.Sum64()
}

// seeded is a PCG stream keyed on the seed and the parts: every random choice
// the harness makes is a function of what it is about, never of the order
// earlier choices were made in.
func seeded(seed uint64, parts ...string) *rand.Rand {
	return rand.New(rand.NewPCG(seed, hash64(parts...)))
}

// eligibleDonors is every unit a donor may be drawn from — Go, non-test,
// non-generated (Load drops both), a body of MinNodes..MaxNodes syntax nodes —
// ordered by a seeded priority over each unit's key. The first Donors are the
// draw; the rest are the top-up order.
func eligibleDonors(h *plantHost, opt plantOptions) []int {
	type cand struct {
		idx int
		pri uint64
	}
	var cands []cand
	for i, u := range h.units {
		n := u.Fingerprint.Nodes
		if u.Lang != gofront.Lang || n < opt.MinNodes || n > opt.MaxNodes || u.Name == "init" {
			continue
		}
		cands = append(cands, cand{i, seeded(opt.Seed, "donor", h.key(i)).Uint64()})
	}
	sort.Slice(cands, func(a, b int) bool {
		if cands[a].pri != cands[b].pri {
			return cands[a].pri < cands[b].pri
		}
		return cands[a].idx < cands[b].idx
	})
	out := make([]int, len(cands))
	for i, c := range cands {
		out[i] = c.idx
	}
	return out
}

// plant is one planted copy of a donor.
type plant struct {
	donor int
	op    *plantOp
	dose  int
	key   string
	unit  parser.CodeUnit
	wire  bool // report level: take over a fraction of the donor's callers
}

// plantVariant names an operator application before it is built.
type plantVariant struct {
	op   *plantOp
	dose int
}

// variants is every operator at every dose it takes, in table order.
func variants(opt plantOptions) []plantVariant {
	var out []plantVariant
	for i := range plantOps {
		op := &plantOps[i]
		if op.family == famDrift {
			for _, d := range opt.Doses {
				out = append(out, plantVariant{op, d})
			}
			continue
		}
		out = append(out, plantVariant{op, 1})
	}
	return out
}

// makePlant builds the plant: the donor's file re-parsed, the operator applied
// dose times, the declaration renamed to a name unique to serial and printed
// under the donor file's own header. ok is false when the operator has no
// site in this donor.
func makePlant(h *plantHost, donor int, v plantVariant, serial int, opt plantOptions) (p *plant, ok bool, err error) {
	u := h.units[donor]
	ds := donorSource{path: u.File, src: h.source(u.File), line: u.StartLine, name: parser.MethodName(u)}
	fset, f, fd, err := ds.parse()
	if err != nil {
		return nil, false, err
	}
	m := &mutation{
		fd: fd, ds: ds, file: f,
		rng:   seeded(opt.Seed, "mutate", h.key(donor), v.op.name, strconv.Itoa(v.dose)),
		names: identNames(f), sib: h.sib, pkg: u.Package, dir: dirOf(u.File),
		spent: map[ast.Node]bool{},
	}
	for e := 0; e < v.dose; e++ {
		if !v.op.apply(m) {
			return nil, false, nil
		}
	}
	name := ds.name + "Pl" + strconv.FormatInt(int64(serial), 36)
	fd.Name = ast.NewIdent(name)
	fd.Doc = nil
	body, err := printDecl(fset, fd)
	if err != nil {
		return nil, false, err
	}
	src := append(append(append([]byte{}, header(fset, f, ds.src)...), "\n\n"...), body...)
	src = append(src, '\n')
	file := filepath.Join(filepath.Dir(u.File), "zz_plant"+strconv.Itoa(serial)+"_"+filepath.Base(u.File))
	units, err := parser.ParseSource(file, src)
	if err != nil {
		return nil, false, err
	}
	for _, pu := range units {
		if parser.MethodName(pu) == name {
			return &plant{donor: donor, op: v.op, dose: v.dose, key: plantKey(h, donor, v), unit: pu}, true, nil
		}
	}
	return nil, false, fmt.Errorf("%s: plant %s did not re-parse:\n%s", h.key(donor), name, src)
}

func plantKey(h *plantHost, donor int, v plantVariant) string {
	k := h.key(donor) + " " + v.op.name
	if v.op.family == famDrift {
		k += "@" + strconv.Itoa(v.dose)
	}
	return k
}

// batches deals plants into pipeline runs: never two plants of one donor in a
// run, at most size plants per run. Plants arrive grouped per donor; the v-th
// plant of every donor forms a column, and columns are chunked.
func batches(perDonor [][]*plant, size int) [][]*plant {
	var out [][]*plant
	for v := 0; ; v++ {
		var col []*plant
		for _, ps := range perDonor {
			if v < len(ps) {
				col = append(col, ps[v])
			}
		}
		if len(col) == 0 {
			return out
		}
		for len(col) > 0 {
			n := min(size, len(col))
			out = append(out, col[:n])
			col = col[n:]
		}
	}
}

// batchSize is the growth budget in plants: Growth of the host, at least one.
func batchSize(h *plantHost, opt plantOptions) int {
	return max(1, int(math.Round(opt.Growth*float64(len(h.units)))))
}

// queryOutcome is one plant measured at the query level.
type queryOutcome struct {
	key    string
	op     *plantOp
	dose   int
	rank   int     // the donor's 1-based rank among the probe's matches; 0 when not retrieved
	shape  float64 // code-shape against the donor
	self   float64 // the donor's code-shape against itself: what an exact copy reads
	bagEq  bool    // the plant's WL bag is the donor's
	cands  int
	nodesP int
	nodesD int
	// top and topShape name the match that outranked the donor, when one did.
	top      string
	topShape float64
}

// reportOutcome is one plant measured at the report level.
type reportOutcome struct {
	key       string
	op        *plantOp
	dose      int
	wire      bool
	rewired   int    // callers moved to the plant
	stage     string // reported, below top, per-func cap, struct-min, not retrieved
	rank      int    // position in the capped report order; 0 when absent from it
	shape     float64
	overlap   float64
	structMin float64
	merge     bool
}

// Report-level stages, in pipeline order.
const (
	stageReported     = "reported"
	stageBelowTop     = "below top"
	stageCap          = "per-func cap"
	stageStructMin    = "struct-min"
	stageCrossBuild   = "cross build unit" // retrieved, then refused by parser.SameBuildUnit
	stageNotRetrieved = "not retrieved"
)

var reportStages = []string{stageReported, stageBelowTop, stageCap, stageStructMin, stageCrossBuild, stageNotRetrieved}

// plantResult is one host's measurement.
type plantResult struct {
	host       string
	funcs      int
	donors     int
	query      []queryOutcome
	report     []reportOutcome
	queryRuns  int
	reportRuns int
	notApplied map[string]int // operator variant -> drawn donors it found no site in
	toppedUp   map[string]int // operator variant -> plants added past the draw
	opt        plantOptions
}

func measurePlants(t *testing.T, h *plantHost, opt plantOptions) plantResult {
	t.Helper()
	eligible := eligibleDonors(h, opt)
	donors := eligible[:min(opt.Donors, len(eligible))]
	res := plantResult{host: h.name, funcs: len(h.units), donors: len(donors),
		notApplied: map[string]int{}, toppedUp: map[string]int{}, opt: opt}
	serial := 0
	vars := variants(opt)

	// Query level: every applicable variant of every drawn donor, then the
	// top-up for rare canon-covered and equivalence operators.
	perDonor := make([][]*plant, len(eligible))
	applied := map[string]int{}
	try := func(di int, v plantVariant) bool {
		p, ok, err := makePlant(h, eligible[di], v, serial, opt)
		if err != nil {
			t.Errorf("[%s] %v", h.name, err)
			return false
		}
		if !ok {
			return false
		}
		serial++
		perDonor[di] = append(perDonor[di], p)
		applied[variantName(v)]++
		return true
	}
	for di := range donors {
		for _, v := range vars {
			if !try(di, v) {
				res.notApplied[variantName(v)]++
			}
		}
	}
	for _, v := range vars {
		if v.op.family == famDrift {
			continue
		}
		for di := len(donors); di < len(eligible) && applied[variantName(v)] < opt.MinPerOp; di++ {
			if try(di, v) {
				res.toppedUp[variantName(v)]++
			}
		}
	}
	for _, b := range batches(perDonor, batchSize(h, opt)) {
		res.query = append(res.query, runQueryBatch(h, b, opt)...)
		res.queryRuns++
	}

	// Report level: per donor a canon representative, an equivalence
	// representative and drift at every dose, each orphaned and — where the
	// donor has callers — wired.
	nr := min(opt.ReportDonors, len(donors))
	perDonor = make([][]*plant, nr)
	for di := 0; di < nr; di++ {
		d := donors[di]
		var picked []plantVariant
		pick := func(cands []plantVariant) {
			for _, v := range cands {
				if _, ok, _ := makePlant(h, d, v, 0, opt); ok {
					picked = append(picked, v)
					return
				}
			}
		}
		var canon, equiv []plantVariant
		var drift [][]plantVariant // per dose, the drift ops starting from this donor's turn
		for _, v := range vars {
			switch v.op.family {
			case famCanon:
				canon = append(canon, v)
			case famEquiv:
				equiv = append(equiv, v)
			}
		}
		pick(canon)
		pick(equiv)
		driftOps := opsOf(famDrift)
		for _, dose := range opt.Doses {
			var cycle []plantVariant
			for k := range driftOps {
				cycle = append(cycle, plantVariant{driftOps[(di+k)%len(driftOps)], dose})
			}
			drift = append(drift, cycle)
		}
		for _, c := range drift {
			pick(c)
		}
		wired := len(h.graph.Callers[concepter.QualifiedName(h.units[d])]) > 0
		for _, wire := range []bool{false, true} {
			if wire && !wired {
				continue
			}
			for _, v := range picked {
				p, ok, err := makePlant(h, d, v, serial, opt)
				if err != nil || !ok {
					continue
				}
				serial++
				p.wire = wire
				perDonor[di] = append(perDonor[di], p)
			}
		}
	}
	for _, b := range batches(perDonor, batchSize(h, opt)) {
		res.report = append(res.report, runReportBatch(t, h, b, opt)...)
		res.reportRuns++
	}
	return res
}

func variantName(v plantVariant) string {
	if v.op.family == famDrift {
		return v.op.name + "@" + strconv.Itoa(v.dose)
	}
	return v.op.name
}

func opsOf(f plantFamily) []*plantOp {
	var out []*plantOp
	for i := range plantOps {
		if plantOps[i].family == f {
			out = append(out, &plantOps[i])
		}
	}
	return out
}

// indexRun builds the corpus statistics over units, exactly as cmd's index()
// does: WL weights, call graph, lexicon, concept docs.
func indexRun(units []parser.CodeUnit, root string) *Run {
	r := &Run{Units: units, Root: root}
	r.StageWL()
	r.StageGraph()
	r.StageTag()
	r.StageMap()
	return r
}

// runQueryBatch probes each plant of a batch the way `doppel query` probes a
// snippet's functions: the plants join the corpus as extra units after the
// population filter, one Prober serves every probe, another plant is never a
// match, and matches are ranked by retriever.ProbeLess over
// concepter.Locality.
func runQueryBatch(h *plantHost, batch []*plant, opt plantOptions) []queryOutcome {
	corpusN := len(h.units)
	units := make([]parser.CodeUnit, 0, corpusN+len(batch))
	units = append(units, h.units...)
	for _, p := range batch {
		units = append(units, p.unit)
	}
	r := indexRun(units, h.root)
	popt := retriever.DefaultOptions()
	popt.ChannelK = opt.QueryK
	popt.MinNodes = opt.MinNodesRet
	prober := retriever.NewProber(r.Units, r.Graph, r.Onto, r.IC, r.WL, popt)

	type match struct {
		c   retriever.Candidate
		loc float64
	}
	out := make([]queryOutcome, 0, len(batch))
	for pi, p := range batch {
		idx := corpusN + pi
		cands, _ := prober.Probe(idx)
		ball := concepter.NeighborhoodSet(r.Graph, r.Units[idx])
		var ms []match
		for _, c := range cands {
			other := retriever.OtherIdx(c, idx)
			if other >= corpusN {
				continue
			}
			ms = append(ms, match{c, concepter.Locality(ball, r.Graph, r.Units[other])})
		}
		sort.SliceStable(ms, func(i, j int) bool {
			return retriever.ProbeLess(ms[i].c, ms[j].c, ms[i].loc, ms[j].loc, idx)
		})
		rank := 0
		for i, m := range ms {
			if retriever.OtherIdx(m.c, idx) == p.donor {
				rank = i + 1
				break
			}
		}
		pf, df := r.Units[idx].Fingerprint, r.Units[p.donor].Fingerprint
		top, topShape := "", 0.0
		if rank != 1 && len(ms) > 0 {
			top, topShape = h.key(retriever.OtherIdx(ms[0].c, idx)), ms[0].c.Breakdown.Score
		}
		out = append(out, queryOutcome{
			top: top, topShape: topShape,
			key: p.key, op: p.op, dose: p.dose, rank: rank,
			shape:  fingerprint.Similarity(pf, df, r.WL).Score,
			self:   fingerprint.Similarity(df, df, r.WL).Score,
			bagEq:  equalBags(pf.WL, df.WL),
			cands:  len(ms),
			nodesP: pf.Nodes, nodesD: df.Nodes,
		})
	}
	return out
}

// equalBags reports whether two WL bags are the same multiset of labels.
func equalBags(a, b []fingerprint.LabelCount) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Label != b[i].Label || a[i].Count != b[i].Count {
			return false
		}
	}
	return true
}

// splice is one identifier rewrite at a byte offset in a file.
type splice struct {
	off  int
	old  string
	name string
}

// runReportBatch plants a batch into the host and runs analyze's ranking
// pipeline at its defaults: calibration at 1%, retrieval at the calibrated
// threshold, comparison, the calibrated struct-min filter, then the report
// order with its per-function cap. Wired plants first take over a fraction of
// their donor's callers, by renaming the call sites in the callers' source.
func runReportBatch(t *testing.T, h *plantHost, batch []*plant, opt plantOptions) []reportOutcome {
	t.Helper()
	corpusN := len(h.units)
	units := make([]parser.CodeUnit, 0, corpusN+len(batch))
	units = append(units, h.units...)

	inBatch := map[int]bool{}
	for _, p := range batch {
		inBatch[p.donor] = true
	}
	edits := map[string][]splice{}
	rewired := make([]int, len(batch))
	for pi, p := range batch {
		if !p.wire {
			continue
		}
		rewired[pi] = wireCallers(h, p, inBatch, edits, opt)
	}
	files := make([]string, 0, len(edits))
	for f := range edits {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		src := applySplices(h.source(f), edits[f])
		parsed, err := parser.ParseSource(f, src)
		if err != nil {
			t.Errorf("[%s] rewired %s: %v", h.name, f, err)
			continue
		}
		var kept []parser.CodeUnit
		for _, u := range parsed {
			if !u.Generated && !isTest(u) {
				kept = append(kept, u)
			}
		}
		span := h.files[f]
		if len(kept) != span[1]-span[0] {
			t.Errorf("[%s] rewired %s: %d units, want %d", h.name, f, len(kept), span[1]-span[0])
			continue
		}
		copy(units[span[0]:span[1]], kept)
	}
	for _, p := range batch {
		units = append(units, p.unit)
	}

	r := indexRun(units, h.root)
	ropt := retriever.DefaultOptions()
	ropt.ChannelK = opt.ChannelK
	ropt.MinNodes = opt.MinNodesRet
	structMin := 0.0
	cal := calibrate.Run(r.Units, r.Docs, r.Comp, r.WL, calibrate.DefaultOptions(opt.Calibrate, opt.MinNodesRet, h.root))
	if cal.Applied() {
		ropt.Threshold = cal.Threshold
		structMin = cal.StructMin
	}
	r.StageRetrieve(ropt)
	r.StagePairs()
	r.StageCompare()

	type pk struct{ a, b int }
	retrieved := map[pk]bool{}
	for _, c := range r.Cands {
		retrieved[pk{c.AIdx, c.BIdx}] = true
	}
	scored := map[pk]analyzer.SimilarPair{}
	var survivors []analyzer.SimilarPair
	for _, sp := range r.Pairs {
		scored[pk{sp.AIdx, sp.BIdx}] = sp
		if sp.Evidence != nil && sp.Evidence.OverlapScore >= structMin {
			survivors = append(survivors, sp)
		}
	}
	ranked, _ := analyzer.SortForReport(survivors, r.Units, 0, opt.MaxPerFunc)
	pos := map[pk]int{}
	for i, sp := range ranked {
		pos[pk{sp.AIdx, sp.BIdx}] = i + 1
	}

	out := make([]reportOutcome, 0, len(batch))
	for pi, p := range batch {
		idx := corpusN + pi
		k := pk{min(idx, p.donor), max(idx, p.donor)}
		o := reportOutcome{key: p.key, op: p.op, dose: p.dose, wire: p.wire, rewired: rewired[pi], structMin: structMin,
			shape: fingerprint.Similarity(r.Units[idx].Fingerprint, r.Units[p.donor].Fingerprint, r.WL).Score}
		sp, ok := scored[k]
		switch {
		case !retrieved[k]:
			o.stage = stageNotRetrieved
		case !ok:
			o.stage = stageCrossBuild
		default:
			o.overlap = sp.Evidence.OverlapScore
			o.merge = sp.MergeWorthy()
			switch rank := pos[k]; {
			case o.overlap < structMin:
				o.stage = stageStructMin
			case rank == 0:
				o.stage = stageCap
			case rank > opt.Top:
				o.stage, o.rank = stageBelowTop, rank
			default:
				o.stage, o.rank = stageReported, rank
			}
		}
		out = append(out, o)
	}
	return out
}

// wireCallers moves a fraction of the donor's callers to the plant: their call
// sites of the donor are renamed to the plant, in edits. Callers that are
// themselves donors in this batch are left alone — rewiring one would change
// a donor under its own plant. A donor with two or more callers always keeps
// one, so the fork has callers on both sides. It returns how many callers
// were moved.
func wireCallers(h *plantHost, p *plant, inBatch map[int]bool, edits map[string][]splice, opt plantOptions) int {
	donor := h.units[p.donor]
	var callers []int
	for _, q := range h.graph.Callers[concepter.QualifiedName(donor)] {
		if ci, ok := h.byQual[q]; ok && !inBatch[ci] {
			callers = append(callers, ci)
		}
	}
	if len(callers) == 0 {
		return 0
	}
	k := int(math.Ceil(opt.RewireFraction * float64(len(callers))))
	if len(callers) >= 2 && k >= len(callers) {
		k = len(callers) - 1
	}
	rng := seeded(opt.Seed, "wire", h.key(p.donor))
	rng.Shuffle(len(callers), func(i, j int) { callers[i], callers[j] = callers[j], callers[i] })
	plantName := parser.MethodName(p.unit)
	donorName := parser.MethodName(donor)
	moved := 0
	for _, ci := range callers[:k] {
		c := h.units[ci]
		ds := donorSource{path: c.File, src: h.source(c.File), line: c.StartLine, name: parser.MethodName(c)}
		fset, f, fd, err := ds.parse()
		if err != nil {
			continue
		}
		sites := callSites(f, fd, donor, donorName, c.Package == donor.Package && dirOf(c.File) == dirOf(donor.File))
		for _, id := range sites {
			edits[c.File] = append(edits[c.File], splice{fset.Position(id.Pos()).Offset, id.Name, plantName})
		}
		if len(sites) > 0 {
			moved++
		}
	}
	return moved
}

// callSites finds the identifiers naming the donor in fd's calls: a bare call
// from the donor's own package, a package-qualified call from another, or any
// call of the method name for a method donor — the same shapes the call-graph
// resolver accepts.
func callSites(f *ast.File, fd *ast.FuncDecl, donor parser.CodeUnit, name string, samePkg bool) []*ast.Ident {
	imports := map[string]bool{}
	for _, im := range f.Imports {
		p, _ := strconv.Unquote(im.Path.Value)
		local := path.Base(p)
		if im.Name != nil {
			local = im.Name.Name
		}
		imports[local] = true
	}
	var out []*ast.Ident
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			if donor.ReceiverType == "" && samePkg && fn.Name == name {
				out = append(out, fn)
			}
		case *ast.SelectorExpr:
			if fn.Sel.Name != name {
				return true
			}
			x, isIdent := fn.X.(*ast.Ident)
			switch {
			case donor.ReceiverType != "":
				out = append(out, fn.Sel)
			case isIdent && !samePkg && imports[x.Name]:
				out = append(out, fn.Sel)
			}
		}
		return true
	})
	return out
}

// applySplices renames identifiers at byte offsets, back to front so earlier
// offsets stay valid.
func applySplices(src []byte, edits []splice) []byte {
	sorted := append([]splice(nil), edits...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].off > sorted[j].off })
	out := append([]byte(nil), src...)
	last := len(out) + 1
	for _, e := range sorted {
		if e.off == last { // the same site named twice
			continue
		}
		if e.off+len(e.old) > len(out) || string(out[e.off:e.off+len(e.old)]) != e.old {
			continue
		}
		out = append(out[:e.off], append([]byte(e.name), out[e.off+len(e.old):]...)...)
		last = e.off
	}
	return out
}

// checkCanon is the hard assertion: a canon-covered plant's canonical tree is
// its donor's, so its WL bag must equal the donor's; and an exact operator,
// which leaves the written body's histograms alone too, must read the donor's
// own code-shape. A failure is a canon bug, not a ranking result.
func checkCanon(t *testing.T, res plantResult) {
	t.Helper()
	for _, o := range res.query {
		if o.op.family != famCanon {
			continue
		}
		if !o.bagEq {
			t.Errorf("[%s] canon bug: %s — WL bag differs from the donor's (code-shape %.4f)", res.host, o.key, o.shape)
			continue
		}
		if o.op.exact && o.shape != o.self {
			t.Errorf("[%s] canon bug: %s — code-shape %.17g, the donor reads %.17g against itself", res.host, o.key, o.shape, o.self)
		}
	}
}

// ---- rendering ----

// queryStats summarizes a set of query outcomes.
type queryStats struct {
	n      int
	hits   []int // per cut
	mrr    float64
	shape  float64
	shapes []float64
}

func summarizeQuery(outs []queryOutcome, cuts []int) queryStats {
	s := queryStats{hits: make([]int, len(cuts))}
	for _, o := range outs {
		s.n++
		if o.rank > 0 {
			s.mrr += 1 / float64(o.rank)
			for i, c := range cuts {
				if o.rank <= c {
					s.hits[i]++
				}
			}
		}
		s.shape += o.shape
		s.shapes = append(s.shapes, o.shape)
	}
	if s.n > 0 {
		s.mrr /= float64(s.n)
		s.shape /= float64(s.n)
	}
	sort.Float64s(s.shapes)
	return s
}

func (s queryStats) line(cuts []int) string {
	if s.n == 0 {
		return "n=0"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "n=%-4d", s.n)
	for i, c := range cuts {
		fmt.Fprintf(&b, "  r@%-2d %.2f", c, float64(s.hits[i])/float64(s.n))
	}
	fmt.Fprintf(&b, "  MRR %.3f  shape mean %.2f min %.2f", s.mrr, s.shape, s.shapes[0])
	return b.String()
}

func reportLine(outs []reportOutcome, top int) string {
	if len(outs) == 0 {
		return "n=0"
	}
	count := map[string]int{}
	rankSum, rankN := 0, 0
	for _, o := range outs {
		count[o.stage]++
		if o.stage == stageReported {
			rankSum += o.rank
			rankN++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "n=%-3d reported@%d %.2f", len(outs), top, float64(count[stageReported])/float64(len(outs)))
	if rankN > 0 {
		fmt.Fprintf(&b, " (mean rank %.1f)", float64(rankSum)/float64(rankN))
	}
	b.WriteString("  lost:")
	for _, st := range reportStages[1:] {
		// A plant file keeps its donor file's build constraints, so a
		// cross-build refusal would be a harness bug: shown only if it happens.
		if st == stageCrossBuild && count[st] == 0 {
			continue
		}
		fmt.Fprintf(&b, " %s %d,", st, count[st])
	}
	return strings.TrimSuffix(b.String(), ",")
}

func (res plantResult) render() string {
	var b strings.Builder
	cuts := res.opt.Cuts
	fmt.Fprintf(&b, "== %s: %d functions, %d donors (seed %d), growth budget %d plant(s) per run\n",
		res.host, res.funcs, res.donors, res.opt.Seed, max(1, int(math.Round(res.opt.Growth*float64(res.funcs)))))

	fmt.Fprintf(&b, "query level (probe, ranked as doppel query ranks):\n")
	for _, f := range plantFamilies {
		fmt.Fprintf(&b, "  %-14s %s\n", f, summarizeQuery(queryOf(res.query, f, nil, 0), cuts).line(cuts))
	}
	fmt.Fprintf(&b, "  per operator:\n")
	for _, v := range variants(res.opt) {
		outs := queryOf(res.query, "", v.op, v.dose)
		na := res.notApplied[variantName(v)]
		fmt.Fprintf(&b, "    %-28s %s  (no site in %d of %d drawn", variantName(v), summarizeQuery(outs, cuts).line(cuts), na, res.donors)
		if up := res.toppedUp[variantName(v)]; up > 0 {
			fmt.Fprintf(&b, ", %d topped up", up)
		}
		b.WriteString(")\n")
	}
	fmt.Fprintf(&b, "  drift dose-response (all drift operators):\n")
	for _, d := range res.opt.Doses {
		var outs []queryOutcome
		for _, o := range res.query {
			if o.op.family == famDrift && o.dose == d {
				outs = append(outs, o)
			}
		}
		fmt.Fprintf(&b, "    dose %d  %s\n", d, summarizeQuery(outs, cuts).line(cuts))
	}

	fmt.Fprintf(&b, "report level (analyze defaults: calibrated, top %d, %d per function):\n", res.opt.Top, res.opt.MaxPerFunc)
	if len(res.report) > 0 {
		fmt.Fprintf(&b, "  calibrated struct-min per run: %s\n", structMins(res.report))
	}
	for _, f := range plantFamilies {
		for _, wire := range []bool{false, true} {
			fmt.Fprintf(&b, "  %-14s %-7s %s\n", f, wiring(wire), reportLine(reportOf(res.report, f, wire, 0), res.opt.Top))
		}
	}
	fmt.Fprintf(&b, "  drift dose-response:\n")
	for _, d := range res.opt.Doses {
		for _, wire := range []bool{false, true} {
			fmt.Fprintf(&b, "    dose %d %-7s %s\n", d, wiring(wire), reportLine(reportOf(res.report, famDrift, wire, d), res.opt.Top))
		}
	}

	fmt.Fprintf(&b, "lost at the query level (donor not in the top %d):\n", cuts[len(cuts)-1])
	lost := 0
	for _, o := range res.query {
		if o.rank == 0 || o.rank > cuts[len(cuts)-1] {
			lost++
			where := "not retrieved"
			if o.rank > 0 {
				where = "rank " + strconv.Itoa(o.rank)
			}
			fmt.Fprintf(&b, "  %s: %s of %d matches, code-shape %.2f, nodes %d vs %d\n", o.key, where, o.cands, o.shape, o.nodesP, o.nodesD)
		}
	}
	if lost == 0 {
		b.WriteString("  none\n")
	}
	fmt.Fprintf(&b, "displaced at the query level (canon-covered and equivalence plants whose donor is retrieved but not first):\n")
	shown := 0
	for _, o := range res.query {
		if o.op.family == famDrift || o.rank <= 1 || o.rank > cuts[len(cuts)-1] {
			continue
		}
		shown++
		fmt.Fprintf(&b, "  %s: rank %d, code-shape %.2f; first is %s at code-shape %.2f\n", o.key, o.rank, o.shape, o.top, o.topShape)
	}
	if shown == 0 {
		b.WriteString("  none\n")
	}
	fmt.Fprintf(&b, "lost at the report level:\n")
	lost = 0
	for _, o := range res.report {
		if o.stage == stageReported {
			continue
		}
		lost++
		fmt.Fprintf(&b, "  %s [%s, %d caller(s) moved]: %s", o.key, wiring(o.wire), o.rewired, o.stage)
		if o.stage != stageNotRetrieved {
			fmt.Fprintf(&b, " — overlap %.2f vs struct-min %.2f, code-shape %.2f, merge-worthy %v", o.overlap, o.structMin, o.shape, o.merge)
		}
		if o.rank > 0 {
			fmt.Fprintf(&b, ", rank %d", o.rank)
		}
		b.WriteString("\n")
	}
	if lost == 0 {
		b.WriteString("  none\n")
	}
	return b.String()
}

func wiring(wire bool) string {
	if wire {
		return "wired"
	}
	return "orphan"
}

// structMins lists the distinct calibrated struct-min values the report runs
// used — the plants are within the growth budget, so they should barely move.
func structMins(outs []reportOutcome) string {
	seen := map[float64]int{}
	for _, o := range outs {
		seen[o.structMin]++
	}
	vals := make([]float64, 0, len(seen))
	for v := range seen {
		vals = append(vals, v)
	}
	sort.Float64s(vals)
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprintf("%.2f (%d plants)", v, seen[v])
	}
	return strings.Join(parts, ", ")
}

// queryOf filters outcomes by family, or by operator and dose.
func queryOf(outs []queryOutcome, f plantFamily, op *plantOp, dose int) []queryOutcome {
	var out []queryOutcome
	for _, o := range outs {
		if (f != "" && o.op.family != f) || (op != nil && (o.op != op || o.dose != dose)) {
			continue
		}
		out = append(out, o)
	}
	return out
}

// reportOf filters outcomes by family and wiring, and by dose when non-zero.
func reportOf(outs []reportOutcome, f plantFamily, wire bool, dose int) []reportOutcome {
	var out []reportOutcome
	for _, o := range outs {
		if o.op.family != f || o.wire != wire || (dose != 0 && o.dose != dose) {
			continue
		}
		out = append(out, o)
	}
	return out
}

// renderPooled pools every host per family, at both levels.
func renderPooled(all []plantResult) string {
	if len(all) == 0 {
		return ""
	}
	cuts := all[0].opt.Cuts
	var q []queryOutcome
	var r []reportOutcome
	var hosts []string
	for _, res := range all {
		q = append(q, res.query...)
		r = append(r, res.report...)
		hosts = append(hosts, res.host)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "== pooled over %s\n", strings.Join(hosts, ", "))
	fmt.Fprintf(&b, "query level:\n")
	for _, f := range plantFamilies {
		fmt.Fprintf(&b, "  %-14s %s\n", f, summarizeQuery(queryOf(q, f, nil, 0), cuts).line(cuts))
	}
	fmt.Fprintf(&b, "  per operator:\n")
	for _, v := range variants(all[0].opt) {
		fmt.Fprintf(&b, "    %-28s %s\n", variantName(v), summarizeQuery(queryOf(q, "", v.op, v.dose), cuts).line(cuts))
	}
	fmt.Fprintf(&b, "report level:\n")
	for _, f := range plantFamilies {
		for _, wire := range []bool{false, true} {
			fmt.Fprintf(&b, "  %-14s %-7s %s\n", f, wiring(wire), reportLine(reportOf(r, f, wire, 0), all[0].opt.Top))
		}
	}
	return b.String()
}

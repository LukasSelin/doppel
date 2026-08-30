package cmd

import (
	"fmt"
	"go/ast"
	"io"
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/calibrate"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/culture"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/mapper"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
	"github.com/LukasSelin/doppel/internal/snapshot"
	"github.com/LukasSelin/doppel/internal/tagger"
)

// Params are the knobs a run is parameterised by.
//
// They travel as one value rather than as eight arguments because comparability
// is an all-or-nothing property: two runs answer the same question only when
// every one of these matches, so the natural operation on them is struct
// equality.
type Params struct {
	Threshold  float64
	TopN       int
	MinNodes   int
	StructMin  float64
	ChannelK   int
	MaxPerFunc int
	TestsMode  string
	Generated  string  // generated-file population: include, exclude, or only
	Calibrate  float64 // null admission rate; > 0 derives Threshold and StructMin from the corpus
	Debug      bool
}

// Result is one complete analysis, up to but not including the final ranking.
//
// The positional alignment is the contract: docs[i] describes units[i], and
// every SimilarPair carries AIdx/BIdx into units. An earlier version of this
// pipeline resolved that relationship by name and silently missed, which is why
// nothing here is keyed by name until snapshot.Build converts at the boundary.
//
// Ranking is left to the caller. SortForReport applies a presentation cap
// (top-N and per-function diversity), and a snapshot taken for diffing wants
// the pre-cap set while the report wants the capped one.
type Result struct {
	Root        string
	Params      Params
	Units       []parser.CodeUnit
	Docs        []concepter.ConceptDoc
	Graph       *concepter.Graph
	Culture     *culture.Model
	Calibration *calibrate.Result // nil unless Params.Calibrate > 0
	TagCounts   map[ontology.TermID]int
	Onto        *ontology.Ontology
	IC          *ontology.IC
	Pairs       []analyzer.SimilarPair

	// WL is the corpus surprisal of every Weisfeiler-Lehman structural
	// label, counted over exactly the population above. It is a corpus
	// statistic like TagCounts and IC and is built in the same place, for
	// the same reason: it must model the population the report describes.
	// nil for an empty corpus.
	//
	// It is what makes code-shape corpus-dependent: every consumer that
	// scores a fingerprint pair — retrieval, calibration, family edge
	// completion, the query probe — is handed this one, so a run has exactly
	// one answer to what a shared structural label is worth.
	WL *fingerprint.LabelIDF

	// ConsStats is the corpus-wide hash-cons of every canonical function
	// body: total canonical AST nodes, and how many distinct subtree shapes
	// (by structural hash) those nodes reduce to. TotalNodes/UniqueSubtrees
	// is the compression ratio the markdown/HTML preamble and the JSON
	// snapshot report — see fingerprint.ConsStats.Ratio. It never feeds any
	// score: it is a corpus-health number, computed once alongside WL.
	ConsStats fingerprint.ConsStats

	// Retrieval is how the candidate set was found — which channels admitted
	// how much. It rides on Result because the report explains its own pair
	// list with it; before that it was computed, printed to stderr and dropped.
	Retrieval retriever.Stats

	// NN is the nearest-neighbour code-shape distribution: for each function,
	// its best code-shape score among the pairs retrieval actually scored —
	// the retrieval union, before any --struct-min filter, since that is the
	// full set every union pair got an exact fingerprint.Breakdown for. It is
	// deliberately NOT an exhaustive nearest-neighbour search (O(n^2) is not
	// acceptable at corpus scale): a function retrieval never paired with
	// anyone is excluded from the percentiles and counted separately. See
	// nnDistribution.
	NN NNStats
}

// NNStats is the corpus-wide nearest-neighbour code-shape summary — see
// Result.NN for what population it is (and is not) drawn from.
type NNStats struct {
	Total  int // functions in the run
	Scored int // of Total, how many appeared in at least one pair retrieval scored

	// P50/P90/P99 are nearest-rank percentiles (calibrate.Quantile's
	// convention: a rank, never an interpolation, so the reported value is a
	// score some function's best neighbour actually had) over the Scored
	// functions' best code-shape scores, ascending.
	P50, P90, P99 float64

	// AtOrAboveThreshold is, of the Scored functions, how many had a best
	// score >= the run's effective threshold (post-config, post-calibrate).
	AtOrAboveThreshold int
}

// analyze runs the pipeline over root and returns everything downstream stages
// need.
//
// progress receives the stage log and per-file parse warnings; pass io.Discard
// to run silently. That parameter is the whole reason this function is separate
// from runAnalyze: a hook must not write to stderr, because on SessionStart
// stderr surfaces to the user as a hook-error notice, and a routine parse
// warning is not an error anyone needs to see.
func analyze(root string, p Params, progress io.Writer) (Result, error) {
	res, err := index(root, p, progress, nil)
	if err != nil || len(res.Units) == 0 {
		return res, err
	}
	return finishAnalyze(res, p, progressOr(progress))
}

func progressOr(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

// index runs the corpus-building prefix of the pipeline: walk, parse, filter,
// tag, corpus IC, call graph, concept docs. It is everything a lookup needs
// and nothing a report does — no retrieval, no comparator, no culture.
//
// extra units are appended to the corpus before tagging, which is how a query
// probe joins: the call-graph resolver hands it resolved callees, mapper
// classifies its role against the same thresholds as everyone, and every
// corpus statistic sees it. The statistics therefore differ from a plain
// analyze by exactly the extras' own contribution, which is the honest way
// to ask how a proposed function would sit in this corpus.
func index(root string, p Params, progress io.Writer, extra []parser.CodeUnit) (Result, error) {
	progress = progressOr(progress)
	res := Result{Root: root, Params: p, TagCounts: map[ontology.TermID]int{}}

	if err := validateMode("tests", p.TestsMode); err != nil {
		return res, err
	}
	if err := validateMode("generated", p.Generated); err != nil {
		return res, err
	}

	fmt.Fprintf(progress, "Scanning %s ...\n", root)
	var units []parser.CodeUnit
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		// The root itself is exempt from the skip rules: `doppel analyze .`
		// hands the walker a directory literally named ".", and a user who
		// points doppel at _examples/ or .config/ has already made the call.
		if d.IsDir() && path != root && parser.ShouldSkipDir(d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		parsed, err := parser.Parse(path)
		if err != nil {
			fmt.Fprintf(progress, "  warn: %s: %v\n", path, err)
			return nil
		}
		units = append(units, parsed...)
		return nil
	})
	if err != nil {
		return res, fmt.Errorf("walk %s: %w", root, err)
	}

	// Population filters, applied before any corpus statistic exists: IC,
	// dfs, culture, habitats, and arenas all model exactly the population
	// the report describes. Tests are conventionally similar by design, so
	// they form their own population rather than diluting production's;
	// generated files are near-identical by construction and, by default,
	// not part of the code anyone maintains by hand.
	units = filterTestUnits(units, p.TestsMode)
	units = filterGeneratedUnits(units, p.Generated)
	// Extras join after the population filter: a probe is part of whatever
	// population was chosen, not a reason to change it.
	units = append(units, extra...)
	res.Units = units

	if len(units) == 0 {
		// An empty corpus is not an error, and it is not this function's job
		// to decide what to say about it: the text report prints a line, the
		// JSON report emits an empty snapshot, the hook stays silent.
		return res, nil
	}

	// Corpus surprisal of the Weisfeiler-Lehman structural labels: ln(N/df)
	// over presence df, the same information measure and the same unit
	// (nats) the retrieval channels use. It is computed here, with the other
	// corpus statistics and after the population filter, so that it models
	// exactly the population the report describes — and so that a query
	// probe, which joins the corpus just above, is counted like everyone
	// else. It is what weights the code-shape score.
	bags := make([][]fingerprint.LabelCount, len(units))
	for i := range units {
		bags[i] = units[i].Fingerprint.WL
	}
	res.WL = fingerprint.LabelWeights(bags)

	// Corpus-wide compression: hash-cons every canonical body's subtrees and
	// keep the two totals the ratio needs. Computed alongside WL because it
	// is the same kind of fact — a static property of the canonical AST
	// forest over exactly this population — and never feeds any score.
	canonical := make([]*ast.FuncDecl, len(units))
	for i := range units {
		canonical[i] = units[i].Canonical
	}
	res.ConsStats = fingerprint.ConsCorpus(canonical)

	// Tag every unit, counting tag occurrences as we go: the counts become
	// the corpus statistics that weight concept matching. A tag most units
	// carry says little about any pair sharing it; a rare one says a lot.
	tagCounts := make(map[ontology.TermID]int)
	for i := range units {
		units[i].Patterns = tagger.Tag(units[i])
		for _, tag := range units[i].Patterns {
			tagCounts[ontology.TermID(tag)]++
		}
	}
	res.TagCounts = tagCounts

	onto := ontology.Default()
	ic := ontology.NewCorpusIC(onto, tagCounts)
	res.Onto, res.IC = onto, ic

	// Build call graph and generate concept documents for every unit.
	// docs[i] describes units[i]; the pipeline relies on that alignment.
	cg := concepter.BuildCallGraph(units)
	res.Graph = cg

	fmt.Fprintf(progress, "Generating concept documents...\n")
	cptr := concepter.New()
	docs := mapper.Map(units, cg, cptr)
	res.Docs = docs

	return res, nil
}

// finishAnalyze is the reporting tail of the pipeline: culture modeling,
// retrieval, structural comparison, culture annotation. Split from index so a
// query can stop at the corpus without paying for any of it.
func finishAnalyze(res Result, p Params, progress io.Writer) (Result, error) {
	units, docs, cg := res.Units, res.Docs, res.Graph
	onto, ic := res.Onto, res.IC
	scorer := ontology.NewScorer(onto, ic)
	comp := comparator.New(scorer)

	// Model the corpus's own conceptual practice: which concepts/roles/calls
	// co-occur beyond chance, and how each concept is normally realized here.
	// Root lets habitats roll up into subsystems (parent directories).
	cultOpts := culture.DefaultOptions()
	cultOpts.Root = res.Root
	cult := culture.Build(units, docs, cg, cultOpts)
	res.Culture = cult
	cs := cult.Stats()
	fmt.Fprintf(progress, "Culture: %d concepts modeled, %d associations, %d unusual realizations\n",
		cs.ConceptsModeled, cs.AssociationCount, cs.UnusualRealizations)
	printHabitatSummary(progress, cs)
	printArenaSummary(progress, cs)

	// Null calibration, when asked for: derive the code-shape and overlap
	// thresholds from what random unrelated pairs score in this corpus. It
	// runs before retrieval because retrieval reads the threshold, and it
	// replaces --threshold and --struct-min outright — a half-calibrated run
	// would be the mixed question Params equality exists to forbid. The
	// effective values travel in Params so a snapshot compares on what was
	// actually used.
	forkFloor := analyzer.ForkShapeFloor
	if p.Calibrate > 0 {
		r := calibrate.Run(units, docs, comp, res.WL, calibrate.DefaultOptions(p.Calibrate, p.MinNodes))
		res.Calibration = &r
		printCalibration(progress, r)
		if r.Applied() {
			p.Threshold, p.StructMin = r.Threshold, r.StructMin
			forkFloor = r.Threshold
		}
	}
	res.Params = p

	// Multi-channel candidate retrieval: structural shape, shared concepts,
	// and shared resolved calls each retrieve per-function top-K neighbors
	// weighted by corpus rarity; the union goes to the expensive comparator.
	fmt.Fprintf(progress, "Found %d functions. Retrieving candidates...\n", len(units))
	opts := retriever.DefaultOptions()
	opts.ChannelK = p.ChannelK
	opts.Threshold = p.Threshold
	opts.MinNodes = p.MinNodes
	if p.Debug {
		opts.ChainTopN = 20 // the "full list", bounded
	}
	cands, stats := retriever.Retrieve(units, cg, onto, ic, res.WL, opts)
	res.Retrieval = stats
	printRetrievalStats(progress, stats)

	pairs := make([]analyzer.SimilarPair, 0, len(cands))
	crossDropped := 0
	for _, c := range cands {
		// A test and a production function are never merge candidates —
		// different build units. Only possible under --tests include.
		if isTestUnit(units[c.AIdx]) != isTestUnit(units[c.BIdx]) {
			crossDropped++
			continue
		}
		pairs = append(pairs, analyzer.SimilarPair{
			A:         units[c.AIdx],
			B:         units[c.BIdx],
			AIdx:      c.AIdx,
			BIdx:      c.BIdx,
			Score:     c.Breakdown.Score,
			Breakdown: c.Breakdown,
			Retrieval: &analyzer.Retrieval{
				Shape:      c.Shape,
				Concept:    c.Concept,
				Call:       c.Call,
				Total:      c.Total,
				TrophicSim: c.TrophicSim,
				CallSim:    c.CallSim,
				Channels:   c.Channels,
				Chains:     sharedChains(c.Chains),
			},
		})
	}
	if crossDropped > 0 {
		fmt.Fprintf(progress, "  %d cross test/prod pairs dropped\n", crossDropped)
	}

	// Attach structural evidence to every candidate pair.
	if len(pairs) > 0 {
		fmt.Fprintf(progress, "Running structural comparison on %d pairs...\n", len(pairs))
		for i := range pairs {
			ev := comp.Compare(docs[pairs[i].AIdx], docs[pairs[i].BIdx])
			pairs[i].Evidence = &ev
		}

		// Nearest-neighbour code-shape distribution, over exactly this set:
		// the retrieval union with every pair's exact fingerprint.Breakdown
		// already attached, before --struct-min can drop any of them. That
		// filter is a selection stage for the *report*, not a reason to
		// declare a function neighbourless.
		res.NN = nnDistribution(pairs, len(units), p.Threshold)

		if p.StructMin > 0 {
			pairs = filterByOverlap(pairs, p.StructMin)
			fmt.Fprintf(progress, "  %d pairs remain after struct-min=%.2f filter\n", len(pairs), p.StructMin)
		}
	} else {
		res.NN = nnDistribution(nil, len(units), p.Threshold)
	}

	// Annotate surviving pairs with unusual concept realizations and habitat
	// misfits — positional lookup, like Evidence attachment; never name-keyed.
	for i := range pairs {
		pairs[i].Culture = cultureNotes(cult, pairs[i].AIdx, pairs[i].BIdx,
			pairs[i].A.Patterns, pairs[i].B.Patterns)
		pairs[i].Habitat = habitatNotes(cult, pairs[i].AIdx, pairs[i].BIdx,
			pairs[i].A.Package, pairs[i].B.Package)
		pairs[i].Kind = analyzer.ClassifyPairWith(pairs[i].A, pairs[i].B, pairs[i].Score, forkFloor)
		pairs[i].Profile = profileNotes(cult, pairs[i].AIdx, pairs[i].BIdx,
			pairs[i].A.Patterns, pairs[i].B.Patterns)
	}
	res.Pairs = pairs

	return res, nil
}

// validateMode checks a population selector. --tests and --generated take the
// same three values and produce the same message, so they share one check
// parameterized by flag name rather than drifting as two copies.
func validateMode(flag, mode string) error {
	switch mode {
	case "include", "exclude", "only":
		return nil
	}
	return fmt.Errorf("invalid --%s value %q: want include, exclude, or only", flag, mode)
}

// filterByOverlap drops pairs below the structural overlap threshold. A
// non-positive minimum keeps everything.
//
// It allocates rather than reslicing in place. The obvious `filtered :=
// pairs[:0]` shares a backing array with its input, so filtering would quietly
// rewrite whatever else still holds the unfiltered slice.
func filterByOverlap(pairs []analyzer.SimilarPair, min float64) []analyzer.SimilarPair {
	if min <= 0 {
		return pairs
	}
	out := make([]analyzer.SimilarPair, 0, len(pairs))
	for _, p := range pairs {
		if p.Evidence != nil && p.Evidence.OverlapScore >= min {
			out = append(out, p)
		}
	}
	return out
}

// nnDistribution computes each function's best code-shape score among pairs,
// which must be the retrieval union with Score already set on every entry
// (true of both branches that call this: the full pre-struct-min set, and
// nil when retrieval found nothing to compare).
//
// A function's best score is the maximum Score over every pair it appears in
// as either side, tracked in a plain index-sized slice rather than a map: the
// result is a pointwise maximum, so it cannot depend on the order pairs
// arrive in, and no sort is needed to make that true. total is len(units) —
// the population Scored is drawn from and NOT how many were scored, since a
// function retrieval never paired with anyone contributes nothing here. That
// is the recall bound every renderer of NNStats must repeat: this is the
// distribution over what retrieval's three bounded channels actually found,
// never an exhaustive O(n^2) nearest-neighbour search.
func nnDistribution(pairs []analyzer.SimilarPair, total int, threshold float64) NNStats {
	best := make([]float64, total)
	has := make([]bool, total)
	for _, p := range pairs {
		for _, idx := range [2]int{p.AIdx, p.BIdx} {
			if idx < 0 || idx >= total {
				continue
			}
			if !has[idx] || p.Score > best[idx] {
				best[idx], has[idx] = p.Score, true
			}
		}
	}
	scores := make([]float64, 0, total)
	for i := 0; i < total; i++ {
		if has[i] {
			scores = append(scores, best[i])
		}
	}
	sort.Float64s(scores)

	atOrAbove := 0
	for _, s := range scores {
		if s >= threshold {
			atOrAbove++
		}
	}
	return NNStats{
		Total:              total,
		Scored:             len(scores),
		P50:                calibrate.Quantile(scores, 0.50),
		P90:                calibrate.Quantile(scores, 0.90),
		P99:                calibrate.Quantile(scores, 0.99),
		AtOrAboveThreshold: atOrAbove,
	}
}

// snapshotOf converts a run into the comparable record of it.
func snapshotOf(res Result, pairs []analyzer.SimilarPair) snapshot.Snapshot {
	return snapshot.Build(res.Units, res.Docs, pairs, res.TagCounts, res.Root, buildVersion(),
		snapshot.Params{
			Threshold:  res.Params.Threshold,
			Top:        res.Params.TopN,
			MinNodes:   res.Params.MinNodes,
			StructMin:  res.Params.StructMin,
			ChannelK:   res.Params.ChannelK,
			MaxPerFunc: res.Params.MaxPerFunc,
			TestsMode:  res.Params.TestsMode,
			Generated:  res.Params.Generated,
			Calibrate:  res.Params.Calibrate,
		},
		snapshot.CorpusMetrics{
			TotalNodes:           res.ConsStats.TotalNodes,
			UniqueSubtrees:       res.ConsStats.UniqueSubtrees,
			NNTotal:              res.NN.Total,
			NNScored:             res.NN.Scored,
			NNP50:                res.NN.P50,
			NNP90:                res.NN.P90,
			NNP99:                res.NN.P99,
			NNAtOrAboveThreshold: res.NN.AtOrAboveThreshold,
		})
}

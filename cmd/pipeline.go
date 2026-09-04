package cmd

import (
	"fmt"
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
	"github.com/LukasSelin/doppel/internal/lexicon"
	"github.com/LukasSelin/doppel/internal/mapper"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
	"github.com/LukasSelin/doppel/internal/snapshot"
	"github.com/LukasSelin/doppel/internal/syntax"
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
	Generated  string   // generated-file population: include, exclude, or only
	Languages  []string // language allowlist; empty means every registered frontend
	Calibrate  float64  // null admission rate; > 0 derives Threshold and StructMin from the corpus
	Debug      bool
	// Pinned says Threshold and StructMin were supplied at the rate in
	// Calibrate rather than derived by this run, so calibration is skipped.
	//
	// It exists for the hook subcommands, which must all measure at one
	// operating point: the Stop hook diffs against a session-start baseline,
	// and re-deriving a threshold every turn lets an edit move the null
	// distribution across a rounding boundary and make the baseline
	// incomparable through no pair's fault. Session start derives the
	// thresholds once; every later turn supplies them back.
	//
	// Deliberately absent from snapshot.Params: what a run measured is the
	// effective Threshold and StructMin, which are recorded, and where they
	// came from does not change the question being asked.
	Pinned bool
	// NoOverlapFilter says this run keeps every scored pair regardless of
	// architectural overlap, and that calibration must not change that.
	//
	// Hook runs set it. They diff the full candidate set on purpose — a pair
	// dropped for presentation reasons has not changed, and reporting it as a
	// session's impact would be a lie — and StructMin zero is how they say so.
	// Calibration derives an overlap floor along with the code-shape one, so
	// without this it would silently install a filter that the hook contract
	// says is not there.
	//
	// It does not make a run half-calibrated: the run has no overlap gate at
	// all, and StructMin zero is recorded in the snapshot, so two runs still
	// agree on what was measured exactly when they measured the same thing.
	NoOverlapFilter bool
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
	Calibration *calibrate.Result           // nil unless Params.Calibrate > 0
	TagCounts   map[ontology.TermID]int     // members per concept; the inventory's own counts
	ConceptMass map[ontology.TermID]float64 // summed membership confidence; what weights IC
	Lexicon     *lexicon.Model
	UnusedSeeds []string // seed concepts this corpus grew no practice for
	Onto        *ontology.Ontology
	IC          *ontology.IC
	Vocab       *ontology.Vocabulary // each learned concept's feature vocabulary; the feature view's substrate
	Pairs       []analyzer.SimilarPair

	// Views counts how the concept views agreed across the compared pairs —
	// the retrieval union, before --struct-min, the same population NN is
	// drawn over. It is the per-report form of the "inert interior" finding:
	// how often the taxonomy and the learned vocabularies disagree about
	// whether two functions do related work. Never feeds a score.
	Views ViewStats

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

// ViewStats is the corpus-wide agreement between the concept views — see
// Result.Views for the population.
type ViewStats struct {
	Compared    int // pairs the comparator scored
	WithFeature int // of Compared, how many had a measured feature view
	Disagree    int // of WithFeature, how many crossed comparator.ViewDisagreeSpread

	// The two directions of disagreement, summing to Disagree. FeatureOnly is
	// vocabulary the taxonomy cannot see (feature above shape); TaxonomyOnly
	// is kinship the vocabularies do not show (shape above feature).
	FeatureOnly  int
	TaxonomyOnly int
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
	res := Result{Root: root, Params: p,
		TagCounts:   map[ontology.TermID]int{},
		ConceptMass: map[ontology.TermID]float64{},
	}

	if err := validateMode("tests", p.TestsMode); err != nil {
		return res, err
	}
	if err := validateMode("generated", p.Generated); err != nil {
		return res, err
	}

	sel, unknown := parser.NewSelection(p.Languages)
	if len(unknown) > 0 {
		return res, fmt.Errorf("unknown languages %v: doppel has frontends for %v", unknown, parser.Languages())
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
		// The extension allowlist is the whole scope rule: a file is in the
		// corpus because a frontend claims it and the selection admits that
		// language. Nothing inspects contents to judge a file code-like.
		if !sel.Admits(path) {
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
	canonical := make([]*syntax.Node, len(units))
	for i := range units {
		canonical[i] = units[i].Canonical
	}
	res.ConsStats = fingerprint.ConsCorpus(canonical)

	// The two statistics above are structural: they read each unit's own
	// canonical AST and nothing else, so they are computed here, first, where
	// the population is already final and no concept vocabulary exists yet.
	// Everything below is conceptual and depends on the call graph.
	//
	// The call graph comes before the lexicon, because the lexicon reads it: a
	// function's resolved callees are one of the evidence channels a concept
	// can be learned from, and the strongest one on most corpora.
	cg := concepter.BuildCallGraph(units)
	res.Graph = cg

	// Learn this corpus's concept vocabulary. The rule tagger still runs, but
	// only to seed the search: it says which functions to look at, and the
	// corpus says what those functions actually have in common. Concepts no
	// seed accounts for are discovered from feature co-occurrence, so a
	// codebase whose vocabulary nobody wrote a rule for is not silent.
	fmt.Fprintf(progress, "Learning concept vocabulary...\n")
	seeds := make([][]string, len(units))
	for i := range units {
		seeds[i] = tagger.Tag(units[i])
	}
	lex := lexicon.Build(units, cg, seeds, lexicon.DefaultOptions())
	res.Lexicon = lex
	res.UnusedSeeds = unusedSeeds(lex)
	ls := lex.Stats()
	fmt.Fprintf(progress,
		"Lexicon: %d concepts (%d seeded, %d emergent), %d/%d features above %d df, %d functions unlabeled\n",
		ls.Seeded+ls.Emergent, ls.Seeded, ls.Emergent,
		ls.FeaturesSurviving, ls.FeaturesTotal, ls.FeatureCap, ls.Untagged)
	assignments := lex.Assignments()
	for i := range units {
		units[i].Concepts = assignments[i]
	}

	// Corpus statistics for concept matching, in two currencies. TagCounts is
	// members per concept, which is what an inventory reports; ConceptMass is
	// the summed confidence, which is what weights information content — a
	// concept carried firmly by twenty functions is more of the corpus than one
	// carried barely by twenty.
	tagCounts := make(map[ontology.TermID]int)
	conceptMass := make(map[ontology.TermID]float64)
	for i := range units {
		for _, c := range units[i].Concepts {
			id := ontology.TermID(c.ID)
			tagCounts[id]++
			conceptMass[id] += c.Confidence
		}
	}
	res.TagCounts, res.ConceptMass = tagCounts, conceptMass

	// The vocabulary is per-corpus now: the abstract interior of the taxonomy
	// survives, its fourteen authored leaves are replaced by what was learned.
	onto := ontology.WithConcepts(ontology.Default(),
		ontology.DerivedConceptTerms(ontology.Default(), derivedConcepts(lex)))
	ic := ontology.NewCorpusICMass(onto, conceptMass)
	res.Onto, res.IC = onto, ic
	res.Vocab = vocabularyOf(lex)

	// Generate concept documents for every unit.
	// docs[i] describes units[i]; the pipeline relies on that alignment.

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
	scorer := ontology.NewScorer(onto, ic).WithVocabulary(res.Vocab)
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
	if p.Calibrate > 0 && p.Pinned {
		// Supplied, not derived — the thresholds are already in p. The fork
		// floor still has to follow them, or "alike enough" would mean two
		// different things in one run.
		forkFloor = p.Threshold
	}
	if p.Calibrate > 0 && !p.Pinned {
		// res.WL is what makes the null distribution the run's own: the same
		// corpus-weighted code-shape metric scores the random pairs as scores
		// the real ones, so the derived floor is a quantile of the very
		// quantity it will be compared against.
		r := calibrate.Run(units, docs, comp, res.WL, calibrate.DefaultOptions(p.Calibrate, p.MinNodes))
		res.Calibration = &r
		printCalibration(progress, r)
		if r.Applied() {
			p.Threshold = r.Threshold
			forkFloor = r.Threshold
			if !p.NoOverlapFilter {
				p.StructMin = r.StructMin
			}
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
		// different build units. Nor are two functions in different
		// languages, one step further out: they do not compare on shape,
		// and merging them is not a thing anyone could do. Only reachable
		// under --tests include, or in a mixed-language corpus.
		if !parser.SameBuildUnit(units[c.AIdx], units[c.BIdx]) {
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
		res.Views = viewStats(pairs)
		if res.Views.WithFeature > 0 {
			fmt.Fprintf(progress, "  Concept views: %d of %d compared pairs disagree with the taxonomy (%d vocabulary the tree misses, %d kinship the vocabularies lack)",
				res.Views.Disagree, res.Views.WithFeature, res.Views.FeatureOnly, res.Views.TaxonomyOnly)
			if n := res.Vocab.Truncated(); n > 0 {
				fmt.Fprintf(progress, "; %d concepts cut to their strongest %d features", n, ontology.MaxVocabularyFeatures)
			}
			fmt.Fprintln(progress)
		}

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
			parser.ConceptIDs(pairs[i].A.Concepts), parser.ConceptIDs(pairs[i].B.Concepts))
		pairs[i].Habitat = habitatNotes(cult, pairs[i].AIdx, pairs[i].BIdx,
			pairs[i].A.Package, pairs[i].B.Package)
		pairs[i].Kind = analyzer.ClassifyPairWith(pairs[i].A, pairs[i].B, pairs[i].Score, forkFloor)
		pairs[i].Explain = analyzer.Explain(pairs[i].A, pairs[i].B)
		pairs[i].Profile = profileNotes(cult, pairs[i].AIdx, pairs[i].BIdx,
			parser.ConceptIDs(pairs[i].A.Concepts), parser.ConceptIDs(pairs[i].B.Concepts))
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

// languageSelection is the run's language allowlist. Errors are impossible
// here: index() rejects an unknown language before any unit is parsed, so by
// the time a snapshot exists the selection has already been validated.
func languageSelection(p Params) parser.Selection {
	sel, _ := parser.NewSelection(p.Languages)
	return sel
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
	return snapshot.Build(res.Units, res.Docs, pairs, res.TagCounts, res.UnusedSeeds, res.Root, buildVersion(),
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
			// Names() resolves an empty selection to the concrete list of
			// languages actually read, so a baseline never records "whatever
			// was built in" — which would compare equal across two builds
			// that read different corpora.
			Languages: languageSelection(res.Params).Names(),
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

// derivedConcepts translates the learned lexicon into taxonomy placements.
//
// A seeded concept hangs where its seed's leaf hung — a concept grown from
// db_access is a kind of data_store_access, whatever this corpus turned out to
// mean by it — and an emergent one hangs beside whichever seeded concept it
// most resembles, or from the root when it resembles none. That is the whole of
// what the authored vocabulary still asserts: the shape of the interior, and
// where a learned leaf plausibly belongs in it.
func derivedConcepts(lex *lexicon.Model) []ontology.DerivedConcept {
	concepts := lex.Concepts()
	out := make([]ontology.DerivedConcept, len(concepts))
	for i, c := range concepts {
		out[i] = ontology.DerivedConcept{
			ID:         c.ID,
			Seed:       ontology.TermID(c.Seed),
			AnchorSeed: ontology.TermID(c.Anchor),
			Def:        c.Definition(),
		}
	}
	return out
}

// vocabularyOf carries each learned concept's feature vocabulary across to the
// ontology's side table, so the comparator's feature view can read what two
// concepts are made of. lexicon may not import ontology, which is why this
// bridge lives here; internal/bench mirrors it and the two must move together.
func vocabularyOf(lex *lexicon.Model) *ontology.Vocabulary {
	concepts := lex.Concepts()
	entries := make([]ontology.VocabularyEntry, len(concepts))
	for i, c := range concepts {
		feats := make([]ontology.WeightedFeature, len(c.Features))
		for j, f := range c.Features {
			feats[j] = ontology.WeightedFeature{Name: f.Name, Weight: f.Weight, Opaque: lexicon.Opaque(f.Name)}
		}
		entries[i] = ontology.VocabularyEntry{ID: ontology.TermID(c.ID), Features: feats}
	}
	return ontology.NewVocabulary(entries)
}

// viewStats counts the concept views' agreement over the compared pairs.
func viewStats(pairs []analyzer.SimilarPair) ViewStats {
	var s ViewStats
	for _, p := range pairs {
		if p.Evidence == nil {
			continue
		}
		s.Compared++
		v := p.Evidence.Views
		if !v.HasFeature {
			continue
		}
		s.WithFeature++
		if !v.Disagree {
			continue
		}
		s.Disagree++
		if v.Feature > v.Shape {
			s.FeatureOnly++
		} else {
			s.TaxonomyOnly++
		}
	}
	return s
}

// unusedSeeds is the seed vocabulary minus the seeds that grew a concept: the
// kinds of work this corpus shows no practice for.
//
// It replaces the old "concept tags with no occurrence" list, which compared a
// run against the fourteen authored leaves. Those leaves are seeds now and
// never appear in a derived vocabulary, so that comparison would report all
// fourteen as absent on every corpus — confidently, and always wrongly.
func unusedSeeds(lex *lexicon.Model) []string {
	grown := make(map[string]bool)
	for _, s := range lex.GrownSeeds() {
		grown[s] = true
	}
	var out []string
	for _, c := range tagger.Concepts() {
		if !grown[c] {
			out = append(out, c)
		}
	}
	sort.Strings(out)
	return out
}

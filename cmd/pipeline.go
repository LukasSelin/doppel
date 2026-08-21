package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/culture"
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
	Generated  string // generated-file population: include, exclude, or only
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
	Root      string
	Params    Params
	Units     []parser.CodeUnit
	Docs      []concepter.ConceptDoc
	Graph     *concepter.Graph
	Culture   *culture.Model
	TagCounts map[ontology.TermID]int
	Onto      *ontology.Ontology
	IC        *ontology.IC
	Pairs     []analyzer.SimilarPair
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

	if err := validateTestsMode(p.TestsMode); err != nil {
		return res, err
	}
	if err := validateGeneratedMode(p.Generated); err != nil {
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
		if d.IsDir() && path != root && shouldSkipDir(d.Name()) {
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
	cult := culture.Build(units, docs, cg, culture.DefaultOptions())
	res.Culture = cult
	cs := cult.Stats()
	fmt.Fprintf(progress, "Culture: %d concepts modeled, %d associations, %d unusual realizations\n",
		cs.ConceptsModeled, cs.AssociationCount, cs.UnusualRealizations)
	printHabitatSummary(progress, cs)
	printArenaSummary(progress, cs)

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
	cands, stats := retriever.Retrieve(units, cg, onto, ic, opts)
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

		if p.StructMin > 0 {
			pairs = filterByOverlap(pairs, p.StructMin)
			fmt.Fprintf(progress, "  %d pairs remain after struct-min=%.2f filter\n", len(pairs), p.StructMin)
		}
	}

	// Annotate surviving pairs with unusual concept realizations and habitat
	// misfits — positional lookup, like Evidence attachment; never name-keyed.
	for i := range pairs {
		pairs[i].Culture = cultureNotes(cult, pairs[i].AIdx, pairs[i].BIdx,
			pairs[i].A.Patterns, pairs[i].B.Patterns)
		pairs[i].Habitat = habitatNotes(cult, pairs[i].AIdx, pairs[i].BIdx,
			pairs[i].A.Package, pairs[i].B.Package)
		pairs[i].Kind = analyzer.ClassifyPair(pairs[i].A, pairs[i].B, pairs[i].Score)
		pairs[i].Profile = profileNotes(cult, pairs[i].AIdx, pairs[i].BIdx,
			pairs[i].A.Patterns, pairs[i].B.Patterns)
	}
	res.Pairs = pairs

	return res, nil
}

func validateTestsMode(mode string) error {
	switch mode {
	case "include", "exclude", "only":
		return nil
	}
	return fmt.Errorf("invalid --tests value %q: want include, exclude, or only", mode)
}

func validateGeneratedMode(mode string) error {
	switch mode {
	case "include", "exclude", "only":
		return nil
	}
	return fmt.Errorf("invalid --generated value %q: want include, exclude, or only", mode)
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
		})
}

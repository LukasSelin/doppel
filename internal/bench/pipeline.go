package bench

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/mapper"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
	"github.com/LukasSelin/doppel/internal/tagger"
)

// Population mirrors the --tests flag: which functions the corpus statistics
// are computed over. The filter runs before tagging, exactly as the pipeline
// does it, so IC and every df model the population being described.
type Population string

const (
	PopInclude Population = "include"
	PopExclude Population = "exclude"
	PopOnly    Population = "only"
)

func (p Population) valid() bool {
	switch p {
	case PopInclude, PopExclude, PopOnly:
		return true
	}
	return false
}

func isTest(u parser.CodeUnit) bool { return strings.HasSuffix(u.File, "_test.go") }

// qualifiedName renders a unit the way the reporter does: Package + "." +
// Name, receiver stars kept.
func qualifiedName(u parser.CodeUnit) string {
	if u.Package == "" {
		return u.Name
	}
	return u.Package + "." + u.Name
}

// Load walks a corpus and returns its functions under the given population.
// Unreadable files and parse errors are skipped, as in the pipeline. Generated
// files are always excluded, mirroring the pipeline's --generated default:
// the labels the harness scores were reviewed against hand-written code, and
// generated clones would drown exactly the rankings they certify.
func Load(root string, pop Population) ([]parser.CodeUnit, error) {
	if !pop.valid() {
		return nil, fmt.Errorf("invalid population %q", pop)
	}
	var units []parser.CodeUnit
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && parser.ShouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		parsed, err := parser.Parse(path)
		if err != nil {
			return nil
		}
		units = append(units, parsed...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	kept := units[:0]
	keepTests := pop == PopOnly
	for _, u := range units {
		if u.Generated {
			continue
		}
		if pop != PopInclude && isTest(u) != keepTests {
			continue
		}
		kept = append(kept, u)
	}
	return kept, nil
}

// Run holds one corpus's intermediate pipeline state. Every field is filled
// by the stage named after it, so a benchmark can time one stage with its
// predecessors already computed.
type Run struct {
	Units     []parser.CodeUnit
	TagCounts map[ontology.TermID]int
	Onto      *ontology.Ontology
	IC        *ontology.IC
	Comp      *comparator.Comparator
	Graph     *concepter.Graph
	Docs      []concepter.ConceptDoc
	Cands     []retriever.Candidate
	Stats     retriever.Stats
	Pairs     []analyzer.SimilarPair
}

// The stages below are the pipeline's ranking-relevant half, as a library at
// production defaults. Culture, habitats and arenas are deliberately absent:
// they annotate a pair, they never move it. Each stage mutates the Run in
// place and is safe to call repeatedly on the same predecessors, which is
// what makes per-stage benchmarking honest.

// StageTag tags every unit and accumulates the corpus tag counts. A Run with
// Onto pre-set keeps it — the weight-override seam AnalyzeWith uses; nil means
// the production default.
func (r *Run) StageTag() {
	r.TagCounts = make(map[ontology.TermID]int)
	for i := range r.Units {
		r.Units[i].Patterns = tagger.Tag(r.Units[i])
		for _, tag := range r.Units[i].Patterns {
			r.TagCounts[ontology.TermID(tag)]++
		}
	}
	if r.Onto == nil {
		r.Onto = ontology.Default()
	}
	r.IC = ontology.NewCorpusIC(r.Onto, r.TagCounts)
	r.Comp = comparator.New(ontology.NewScorer(r.Onto, r.IC))
}

// StageGraph resolves the corpus call graph.
func (r *Run) StageGraph() { r.Graph = concepter.BuildCallGraph(r.Units) }

// StageMap builds and enriches the concept documents.
func (r *Run) StageMap() { r.Docs = mapper.Map(r.Units, r.Graph, concepter.New()) }

// StageRetrieve runs the three retrieval channels.
func (r *Run) StageRetrieve(opt retriever.Options) {
	r.Cands, r.Stats = retriever.Retrieve(r.Units, r.Graph, r.Onto, r.IC, opt)
}

// StagePairs materializes candidates into pairs, dropping cross test/prod
// pairs the way the pipeline does — different build units are never merge
// candidates, whatever the population.
func (r *Run) StagePairs() {
	pairs := make([]analyzer.SimilarPair, 0, len(r.Cands))
	for _, c := range r.Cands {
		if isTest(r.Units[c.AIdx]) != isTest(r.Units[c.BIdx]) {
			continue
		}
		pairs = append(pairs, analyzer.SimilarPair{
			A: r.Units[c.AIdx], B: r.Units[c.BIdx],
			AIdx: c.AIdx, BIdx: c.BIdx,
			Score: c.Breakdown.Score, Breakdown: c.Breakdown,
			Retrieval: &analyzer.Retrieval{
				Shape: c.Shape, Concept: c.Concept, Call: c.Call, Total: c.Total,
				TrophicSim: c.TrophicSim, CallSim: c.CallSim,
			},
		})
	}
	r.Pairs = pairs
}

// StageCompare scores every pair's architectural context.
func (r *Run) StageCompare() {
	for i := range r.Pairs {
		ev := r.Comp.Compare(r.Docs[r.Pairs[i].AIdx], r.Docs[r.Pairs[i].BIdx])
		r.Pairs[i].Evidence = &ev
	}
}

// Analyze runs every stage in pipeline order.
func Analyze(units []parser.CodeUnit, opt retriever.Options) *Run {
	return AnalyzeWith(units, opt, nil)
}

// AnalyzeWith runs every stage under a custom vocabulary — the seam the
// ablation and fitting harness scores through. A nil onto is Analyze exactly.
func AnalyzeWith(units []parser.CodeUnit, opt retriever.Options, onto *ontology.Ontology) *Run {
	r := &Run{Units: units, Onto: onto}
	r.StageTag()
	r.StageGraph()
	r.StageMap()
	r.StageRetrieve(opt)
	r.StagePairs()
	r.StageCompare()
	return r
}

// Rescore re-runs only the weight-sensitive tail — comparator construction and
// StageCompare — under a different vocabulary, reusing parse, tags, IC, the
// call graph and retrieval. That reuse is exact, not approximate: relation
// weights affect neither the taxonomy nor IC nor any retrieval channel; the
// only reader is Compare's composite sum. This is what makes a 12-way ablation
// cost twelve compare passes instead of twelve pipelines.
func (r *Run) Rescore(onto *ontology.Ontology) {
	r.Onto = onto
	r.Comp = comparator.New(ontology.NewScorer(onto, r.IC))
	r.StageCompare()
}

// RankKey is the corroborated-evidence ordering quantity SortForReport uses,
// so a scorecard can print it. One definition lives in analyzer; this is a
// call, not a copy, so the two cannot drift.
func RankKey(p analyzer.SimilarPair) float64 {
	return analyzer.RankKey(p, analyzer.DefaultRankOptions())
}

// Reretrieve re-runs the retrieval-sensitive tail — retrieval, pair
// materialization and comparison — under different retriever options,
// reusing tags, IC, the call graph and the concept docs. Exact: nothing in
// Options reaches those stages. This is what makes a sweep over retrieval
// knobs or fingerprint weights cost retrieve+compare per variant rather than
// a pipeline.
func (r *Run) Reretrieve(opt retriever.Options) {
	r.StageRetrieve(opt)
	r.StagePairs()
	r.StageCompare()
}

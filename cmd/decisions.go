package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
)

// decisionsLog is `analyze --decisions`: every pair retrieval proposed, and
// what the pipeline did with it — reported, capped by --max-per-func, ranked
// past --top, dropped under --struct-min, or dropped as two build units —
// with each gate of the merge verdict stated as passed or blocked.
//
// It exists for labelling. A golden review judges pairs one at a time, and the
// question it keeps asking is why a pair is (or is not) where it is; the report
// answers that only for the pairs it shows. Sides are named exactly as a labels
// file names them (qualified name plus root-relative slash file), so a record
// becomes a label by adding a class and a note.
//
// Presentation only. Nothing here feeds a score, and the pipeline flag behind
// it (Params.KeepDropped) is not in snapshot.Params, so writing the log cannot
// make a baseline incomparable. Like every report it is deterministic: every
// list is in a fixed order and no map decides one.
type decisionsLog struct {
	Doppel string          `json:"doppel"`
	Point  decisionsPoint  `json:"operatingPoint"`
	Counts decisionsCounts `json:"counts"`
	Pairs  []decision      `json:"pairs"`
}

// decisionsPoint is the operating point the fates were decided at — the
// effective values, after calibration.
type decisionsPoint struct {
	Threshold  float64 `json:"threshold"`
	StructMin  float64 `json:"structMin"`
	Calibrate  float64 `json:"calibrate"`
	Top        int     `json:"top"`
	MaxPerFunc int     `json:"maxPerFunc"`
}

// decisionsCounts is how many pairs met each fate, every count always emitted.
type decisionsCounts struct {
	Retrieved      int `json:"retrieved"`
	Reported       int `json:"reported"`
	Capped         int `json:"capped"`
	BelowTop       int `json:"belowTop"`
	BelowStructMin int `json:"belowStructMin"`
	CrossBuildUnit int `json:"crossBuildUnit"`
	CrossTarget    int `json:"crossBuildTarget"`
	MergeWorthy    int `json:"mergeWorthy"`
}

// decision is one pair's record.
type decision struct {
	A     string `json:"a"`
	AFile string `json:"aFile"`
	ALine int    `json:"aLine"`
	B     string `json:"b"`
	BFile string `json:"bFile"`
	BLine int    `json:"bLine"`

	// Fate is one of analyzer.Fate* or one of the pipeline's drop reasons.
	Fate string `json:"fate"`
	// Rank is a reported pair's report position. Position is the pair's place
	// in the ranked order of every pair that survived filtering — the number
	// to read a capped or below-top pair against. CappedBy is the function
	// whose --max-per-func quota was already full.
	Rank     int    `json:"rank,omitempty"`
	Position int    `json:"position,omitempty"`
	CappedBy string `json:"cappedBy,omitempty"`

	// The merge verdict, gate by gate (comparator.Verdict). Empty on a pair
	// that was never compared.
	MergeWorthy bool     `json:"mergeWorthy"`
	Passed      []string `json:"passed,omitempty"`
	Blocked     []string `json:"blocked,omitempty"`

	Kind string `json:"kind,omitempty"`

	CodeShape   float64  `json:"codeShape"`
	Overlap     *float64 `json:"overlap,omitempty"`
	Containment float64  `json:"containment"`
	Trophic     float64  `json:"trophic"`
	Mass        float64  `json:"mass"`
	RankKey     float64  `json:"rankKey"`
	Channels    []string `json:"channels,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
	Explain     string   `json:"explain,omitempty"`
}

// buildDecisions assembles the log. sorted must be res.Pairs after
// SortForReport, which sorts it in place, so analyzer.Place walks exactly the
// order the report was cut from.
func buildDecisions(res *Result, sorted []analyzer.SimilarPair, topN, maxPerFunc int) decisionsLog {
	log := decisionsLog{
		Doppel: buildVersion(),
		Point: decisionsPoint{
			Threshold:  res.Params.Threshold,
			StructMin:  res.Params.StructMin,
			Calibrate:  res.Params.Calibrate,
			Top:        topN,
			MaxPerFunc: maxPerFunc,
		},
	}
	ro := analyzer.DefaultRankOptions()

	for i, pl := range analyzer.Place(sorted, topN, maxPerFunc) {
		d := newDecision(res, sorted[i], pl.Fate, ro)
		d.Rank = pl.Rank
		d.Position = i + 1
		if pl.CappedBy >= 0 {
			d.CappedBy = concepter.QualifiedName(res.Units[pl.CappedBy])
		}
		log.Pairs = append(log.Pairs, d)
	}

	// Dropped pairs have no report order. Struct-min drops were compared and
	// keyed, so they follow the same key; build-unit drops key on retrieval
	// alone. Names and files break ties, so the order is the tree's, not the
	// walk's.
	dropped := make([]decision, 0, len(res.Dropped))
	for _, dp := range res.Dropped {
		dropped = append(dropped, newDecision(res, dp.Pair, dp.Reason, ro))
	}
	fateOrder := func(f string) int {
		switch f {
		case dropBelowStructMin:
			return 0
		case dropCrossBuildUnit:
			return 1
		}
		return 2
	}
	sort.SliceStable(dropped, func(i, j int) bool {
		x, y := dropped[i], dropped[j]
		if x.Fate != y.Fate {
			return fateOrder(x.Fate) < fateOrder(y.Fate)
		}
		if x.RankKey != y.RankKey {
			return x.RankKey > y.RankKey
		}
		if x.A != y.A {
			return x.A < y.A
		}
		if x.B != y.B {
			return x.B < y.B
		}
		if x.AFile != y.AFile {
			return x.AFile < y.AFile
		}
		return x.BFile < y.BFile
	})
	log.Pairs = append(log.Pairs, dropped...)

	c := &log.Counts
	for _, d := range log.Pairs {
		c.Retrieved++
		switch d.Fate {
		case analyzer.FateReported:
			c.Reported++
		case analyzer.FateCapped:
			c.Capped++
		case analyzer.FateBelowTop:
			c.BelowTop++
		case dropBelowStructMin:
			c.BelowStructMin++
		case dropCrossBuildUnit:
			c.CrossBuildUnit++
		case dropCrossBuildTarget:
			c.CrossTarget++
		}
		if d.MergeWorthy {
			c.MergeWorthy++
		}
	}
	return log
}

// newDecision is one pair's record, minus its report placement.
func newDecision(res *Result, p analyzer.SimilarPair, fate string, ro analyzer.RankOptions) decision {
	a, b := &res.Units[p.AIdx], &res.Units[p.BIdx]
	d := decision{
		A: concepter.QualifiedName(*a), AFile: parser.RelSlash(res.Root, a.File), ALine: a.StartLine,
		B: concepter.QualifiedName(*b), BFile: parser.RelSlash(res.Root, b.File), BLine: b.StartLine,
		Fate:        fate,
		CodeShape:   round4(p.Score),
		Containment: round4(p.Breakdown.Containment),
		RankKey:     round4(analyzer.RankKey(p, ro, res.Units)),
		Explain:     p.Explain,
	}
	if p.Retrieval != nil {
		d.Trophic = round4(p.Retrieval.TrophicSim)
		d.Mass = round4(p.Retrieval.Total)
		d.Channels = p.Retrieval.Channels
	}
	if p.Kind != nil {
		d.Kind = p.Kind.Kind
	}
	if ev := p.Evidence; ev != nil {
		o := round4(ev.OverlapScore)
		d.Overlap = &o
		d.MergeWorthy = p.MergeWorthy()
		d.Passed, d.Blocked = comparator.Verdict(*ev, p.Score)
		d.Evidence = ev.Reasons
	}
	return d
}

// writeDecisions renders the log to path and says so on progress.
func writeDecisions(path string, log decisionsLog, progress io.Writer) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create decisions file: %w", err)
	}
	defer f.Close()
	if err := reporter.WriteJSON(f, log); err != nil {
		return fmt.Errorf("write decisions: %w", err)
	}
	c := log.Counts
	fmt.Fprintf(progress, "Decisions written to %s (%d pairs: %d reported, %d capped, %d below top, %d below struct-min, %d cross build-unit, %d cross build-target; %d merge-worthy)\n",
		path, c.Retrieved, c.Reported, c.Capped, c.BelowTop, c.BelowStructMin, c.CrossBuildUnit, c.CrossTarget, c.MergeWorthy)
	return nil
}

package bench

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/lexicon"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
	"github.com/LukasSelin/doppel/internal/tagger"
)

// membershipVariant is one setting of the membership rule, named for the log.
type membershipVariant struct {
	name string
	opt  func(lexicon.Options) lexicon.Options
}

// membershipVariants are the companion bounds measured against coverage alone.
//
// Two questions have been asked of the membership rule and both are answered
// here rather than argued: how many concepts one unit may belong to
// (MaxMemberships, which is shipped at 3), and where a concept's bar comes from
// (FloorRule, which is shipped at FloorFounding despite the obvious objection
// to it — see lexicon.floors for what the alternatives measured).
//
// Re-running this is how either would be revisited. The floor columns are the
// ones the second question turns on: p10/p50/p90 across concepts and their
// ratio say whether one FloorQuantile is producing one kind of number, and
// `dropped` says how much vocabulary a corpus-relative bar costs.
func membershipVariants() []membershipVariant {
	return []membershipVariant{
		{"founding (shipped)", func(o lexicon.Options) lexicon.Options { o.FloorRule = lexicon.FloorFounding; return o }},
		{"touched .25", func(o lexicon.Options) lexicon.Options {
			o.FloorRule = lexicon.FloorTouched
			o.TouchedQuantile = 0.25
			return o
		}},
		{"touched .50", func(o lexicon.Options) lexicon.Options {
			o.FloorRule = lexicon.FloorTouched
			o.TouchedQuantile = 0.50
			return o
		}},
		{"touched .75", func(o lexicon.Options) lexicon.Options {
			o.FloorRule = lexicon.FloorTouched
			o.TouchedQuantile = 0.75
			return o
		}},
		{"touched .90", func(o lexicon.Options) lexicon.Options {
			o.FloorRule = lexicon.FloorTouched
			o.TouchedQuantile = 0.90
			return o
		}},
		{"relmax .25", func(o lexicon.Options) lexicon.Options {
			o.FloorRule = lexicon.FloorRelMax
			o.RelMaxFraction = 0.25
			return o
		}},
		{"relmax .50", func(o lexicon.Options) lexicon.Options {
			o.FloorRule = lexicon.FloorRelMax
			o.RelMaxFraction = 0.50
			return o
		}},
		{"relmax .75", func(o lexicon.Options) lexicon.Options {
			o.FloorRule = lexicon.FloorRelMax
			o.RelMaxFraction = 0.75
			return o
		}},
	}
}

// membershipLine summarizes one built lexicon: how much of the corpus it labels
// at all, how much it labels each function, and whether any one concept has
// grown large enough to mean nothing.
func membershipLine(m *lexicon.Model, n int) string {
	st := m.Stats()
	largest, largestID := 0, ""
	for _, c := range m.Concepts() {
		if c.Members > largest {
			largest, largestID = c.Members, c.ID
		}
	}
	sum, count := 0.0, 0
	for _, cs := range m.Assignments() {
		for _, c := range cs {
			sum += c.Confidence
			count++
		}
	}
	meanConf := 0.0
	if count > 0 {
		meanConf = sum / float64(count)
	}
	// The floor spread is the premise of any change to where the bar comes
	// from: one FloorQuantile over each concept's own founding set produces a
	// different kind of number per concept, and p90/p10 is how far apart those
	// numbers actually are.
	floors := make([]float64, 0, len(m.Concepts()))
	for _, c := range m.Concepts() {
		floors = append(floors, c.Floor)
	}
	sort.Float64s(floors)
	q := func(p float64) float64 {
		if len(floors) == 0 {
			return 0
		}
		i := int(p * float64(len(floors)-1))
		return floors[i]
	}
	ratio := 0.0
	if q(0.1) > 0 {
		ratio = q(0.9) / q(0.1)
	}
	return fmt.Sprintf("untagged %4d/%-5d (%4.1f%%)  assigns %6d (%.1f/fn)  concepts %4d  largest %4d (%4.1f%%) %-38s meanConf %.3f  floor p10/p50/p90 %.4f/%.4f/%.4f (x%.1f)  dropped %d",
		st.Untagged, n, 100*float64(st.Untagged)/float64(n),
		st.Assignments, float64(st.Assignments)/float64(n),
		len(m.Concepts()), largest, 100*float64(largest)/float64(n), truncID(largestID), meanConf,
		q(0.1), q(0.5), q(0.9), ratio, st.FloorDropped)
}

func truncID(s string) string {
	if len(s) > 38 {
		return s[:35] + "..."
	}
	return s
}

// buildLexicon builds one variant's lexicon over an already-loaded corpus.
func buildLexicon(units []parser.CodeUnit, v membershipVariant) *lexicon.Model {
	g := concepter.BuildCallGraph(units)
	seeds := make([][]string, len(units))
	for i := range units {
		seeds[i] = tagger.Tag(units[i])
	}
	return lexicon.Build(units, g, seeds, v.opt(lexicon.DefaultOptions()))
}

// TestLexiconMembershipLadder prints the membership statistics of every variant
// on every fetched corpus. It asserts nothing — the adoption rule is written in
// CLAUDE.md beside the measured table, as TestMinIDF's is.
//
//	DOPPEL_BENCH_LEXICON=1 go test ./internal/bench/ -v -run TestLexiconMembership
func TestLexiconMembershipLadder(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_LEXICON") != "1" {
		t.Skip("set DOPPEL_BENCH_LEXICON=1 to run the membership measurement")
	}
	variants := membershipVariants()
	for _, c := range Corpora {
		if !Present(c) {
			continue
		}
		dir, err := Path(c)
		if err != nil {
			t.Fatal(err)
		}
		units, err := Load(dir, PopExclude)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range variants {
			t.Logf("[%-10s] %-12s %s", c.Name, v.name, membershipLine(buildLexicon(units, v), len(units)))
		}
	}
}

// TestLexiconMembershipLabels scores each variant against the committed
// reviews. Coverage is bought at a price — more memberships means a louder
// concept retrieval channel — and the labels are what say whether the extra
// recall is signal.
func TestLexiconMembershipLabels(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_LEXICON") != "1" {
		t.Skip("set DOPPEL_BENCH_LEXICON=1 to run the membership measurement")
	}
	corpora := loadLabeledCorpora(t)
	if len(corpora) == 0 {
		t.Skip("no labeled corpora available; run `task corpora` or set DOPPEL_BENCH_CORPUS/DOPPEL_BENCH_LABELS")
	}
	for i := range corpora {
		lc := &corpora[i]
		for _, v := range membershipVariants() {
			// A fresh unit slice per variant: StageTag writes memberships onto
			// the units, so a shared slice would carry the previous variant's
			// concepts into the next one's corpus statistics.
			units := append([]parser.CodeUnit(nil), lc.run.Units...)
			run := AnalyzeLexicon(units, retriever.DefaultOptions(), v.opt(lexicon.DefaultOptions()))
			sc := Score(run, lc.lf)
			t.Logf("[%s] %-12s union %5d  %s  violations %d",
				lc.name, v.name, run.Stats.Union, scLine(sc), violations(sc))
		}
	}
}

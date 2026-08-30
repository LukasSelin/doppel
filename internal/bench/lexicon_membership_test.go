package bench

import (
	"fmt"
	"os"
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
// Coverage removed the size term from the membership bar — see
// lexicon.corpus.cover — and with it the accidental ceiling that bar provided,
// so unbounded assignment runs away on a wide corpus. MaxMemberships is the
// bound; the ladder is what picked its value, and re-running this is how a
// different value would be argued for.
func membershipVariants() []membershipVariant {
	return []membershipVariant{
		{"unbounded", func(o lexicon.Options) lexicon.Options { o.MaxMemberships = 0; return o }},
		{"top2", func(o lexicon.Options) lexicon.Options { o.MaxMemberships = 2; return o }},
		{"top3 (shipped)", func(o lexicon.Options) lexicon.Options { o.MaxMemberships = 3; return o }},
		{"top4", func(o lexicon.Options) lexicon.Options { o.MaxMemberships = 4; return o }},
		{"top6", func(o lexicon.Options) lexicon.Options { o.MaxMemberships = 6; return o }},
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
	return fmt.Sprintf("untagged %4d/%-5d (%4.1f%%)  assigns %6d (%.1f/fn)  concepts %4d  largest %4d (%4.1f%%) %-38s meanConf %.3f",
		st.Untagged, n, 100*float64(st.Untagged)/float64(n),
		st.Assignments, float64(st.Assignments)/float64(n),
		len(m.Concepts()), largest, 100*float64(largest)/float64(n), truncID(largestID), meanConf)
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

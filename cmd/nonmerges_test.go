package cmd

import (
	"reflect"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/reporter"
)

func kindPair(a, b int, kind string) analyzer.SimilarPair {
	p := analyzer.SimilarPair{AIdx: a, BIdx: b}
	if kind != "" {
		p.Kind = &analyzer.KindNote{Kind: kind}
	}
	return p
}

func TestCutList(t *testing.T) {
	pairs := []analyzer.SimilarPair{
		kindPair(0, 1, ""),
		kindPair(2, 3, analyzer.KindThinWrappers),
		kindPair(4, 5, analyzer.KindFork),
		kindPair(6, 7, analyzer.KindMirror),
		kindPair(8, 9, analyzer.KindThinWrappers),
		kindPair(10, 11, analyzer.KindInterfaceImpl),
	}
	idx := func(ps []analyzer.SimilarPair) []int {
		out := make([]int, len(ps))
		for i, p := range ps {
			out[i] = p.AIdx
		}
		return out
	}

	// include, and anything unrecognised, cuts nothing and says nothing.
	for _, mode := range []string{reporter.ListInclude, ""} {
		shown, left := cutList(pairs, mode)
		if len(shown) != len(pairs) || left.Total != 0 || left.Mode != "" {
			t.Errorf("mode %q: shown %v, left %+v", mode, idx(shown), left)
		}
	}

	// exclude drops the three not-a-merge kinds, keeps the order, and counts
	// what it dropped by kind in the kinds' own order. A fork and an
	// interface implementation are kinds too, and stay.
	shown, left := cutList(pairs, reporter.ListExclude)
	if got := idx(shown); !reflect.DeepEqual(got, []int{0, 4, 10}) {
		t.Errorf("exclude shows %v", got)
	}
	want := reporter.NotListed{Mode: reporter.ListExclude, Total: 3, Scored: 6}
	if !reflect.DeepEqual(left, want) {
		t.Errorf("exclude left %+v, want %+v", left, want)
	}
	// Of an uncut list holding the first four pairs, two are left out, and
	// they are counted by kind in the kinds' own order.
	n, kinds := displaced(pairs[:4], reporter.ListExclude)
	if n != 2 || !reflect.DeepEqual(kinds, []reporter.KindCount{
		{Kind: analyzer.KindMirror, Count: 1}, {Kind: analyzer.KindThinWrappers, Count: 1},
	}) {
		t.Errorf("displaced = %d %+v", n, kinds)
	}
	if n, kinds := displaced(pairs[:1], reporter.ListExclude); n != 0 || kinds != nil {
		t.Errorf("nothing displaced from a list of merge candidates, got %d %+v", n, kinds)
	}

	// only is the complement.
	shown, left = cutList(pairs, reporter.ListOnly)
	if got := idx(shown); !reflect.DeepEqual(got, []int{2, 6, 8}) {
		t.Errorf("only shows %v", got)
	}
	if !reflect.DeepEqual(left, reporter.NotListed{Mode: reporter.ListOnly, Total: 3, Scored: 6}) {
		t.Errorf("only left %+v", left)
	}

	// Nothing to leave out: the input itself, and a zero account, so a
	// corpus with no such pair reports as it always did.
	clean := []analyzer.SimilarPair{kindPair(0, 1, ""), kindPair(2, 3, analyzer.KindFork)}
	shown, left = cutList(clean, reporter.ListExclude)
	if len(shown) != 2 || !reflect.DeepEqual(left, reporter.NotListed{}) {
		t.Errorf("clean list: shown %v, left %+v", idx(shown), left)
	}
}

// A pair the list does not show must not hold a slot the next one could
// have: the cut comes before the per-function cap, not after it.
func TestCutListComesBeforeTheCap(t *testing.T) {
	ret := func(total float64) *analyzer.Retrieval { return &analyzer.Retrieval{Total: total, TrophicSim: 1} }
	mk := func(a, b int, total float64, kind string) analyzer.SimilarPair {
		p := kindPair(a, b, kind)
		p.Score, p.Retrieval = 1, ret(total)
		return p
	}
	// Function 0 is in three pairs. The strongest is a mirror.
	pairs := []analyzer.SimilarPair{
		mk(0, 1, 30, analyzer.KindMirror),
		mk(0, 2, 20, ""),
		mk(0, 3, 10, ""),
	}
	all, _ := analyzer.SortForReportBy(pairs, func(p analyzer.SimilarPair) float64 { return p.Retrieval.Total }, 0, 1)
	if len(all) != 1 || all[0].BIdx != 1 {
		t.Fatalf("uncut, the cap of one keeps the mirror: %+v", all)
	}
	visible, left := cutList(pairs, reporter.ListExclude)
	if left.Total != 1 {
		t.Fatalf("left %+v", left)
	}
	listed, suppressed := analyzer.SortForReportBy(visible, func(p analyzer.SimilarPair) float64 { return p.Retrieval.Total }, 0, 1)
	if len(listed) != 1 || listed[0].BIdx != 2 || suppressed != 1 {
		t.Errorf("cut first, the cap of one keeps the next finding: %+v (suppressed %d)", listed, suppressed)
	}
}

func TestSuppressedOnStdout(t *testing.T) {
	if got := suppressedOnStdout(formatText, 7, 3); got != 3 {
		t.Errorf("text reports the cut list's count, got %d", got)
	}
	if got := suppressedOnStdout(formatJSON, 7, 3); got != 7 {
		t.Errorf("json reports the uncut ranking's count, got %d", got)
	}
}

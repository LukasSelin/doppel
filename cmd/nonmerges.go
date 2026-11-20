package cmd

import (
	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/reporter"
)

// listShows reports whether a report's pair list shows p under a
// --non-merges mode: every pair, every pair but the ones whose kind says they
// are something other than a merge, or only those.
//
// This is a reader's default and nothing more. The pair was retrieved,
// compared and ranked like any other, a kind still never enters a rank key,
// and --format json and the dashboard carry it whatever the mode — the page
// has had the same three kinds off by default since it had a filter rail.
func listShows(p analyzer.SimilarPair, mode string) bool {
	nonMerge := p.Kind != nil && analyzer.NotAMerge(p.Kind.Kind)
	switch mode {
	case reporter.ListExclude:
		return !nonMerge
	case reporter.ListOnly:
		return nonMerge
	}
	return true
}

// cutList is the run's scored pairs as the report list may show them under
// mode, in the order given, with a count of what it left out. When nothing
// is left out it hands back pairs itself and a zero account, so a corpus
// with no such pair reports exactly as it did before the list could be cut.
//
// The cut comes before the ranking walk, not after it: a pair that is not
// listed must not use up a function's --max-per-func quota or a --top slot,
// or hiding it would leave a gap where the next finding belongs.
func cutList(pairs []analyzer.SimilarPair, mode string) ([]analyzer.SimilarPair, reporter.NotListed) {
	if mode != reporter.ListExclude && mode != reporter.ListOnly {
		return pairs, reporter.NotListed{}
	}
	shown := make([]analyzer.SimilarPair, 0, len(pairs))
	for _, p := range pairs {
		if listShows(p, mode) {
			shown = append(shown, p)
		}
	}
	left := len(pairs) - len(shown)
	if left == 0 {
		return pairs, reporter.NotListed{}
	}
	return shown, reporter.NotListed{Mode: mode, Total: left, Scored: len(pairs)}
}

// displaced counts the pairs of uncut — the list as it would have been with
// every kind in it — that mode leaves out, and those pairs by kind. They are
// the ones a reader would otherwise have been shown; the rest of what a cut
// leaves out ranks below the list either way.
func displaced(uncut []analyzer.SimilarPair, mode string) (int, []reporter.KindCount) {
	byKind := map[string]int{}
	n := 0
	for _, p := range uncut {
		if listShows(p, mode) {
			continue
		}
		n++
		if p.Kind != nil {
			byKind[p.Kind.Kind]++
		}
	}
	// Read by key in the kinds' own order; the map decides nothing.
	var kinds []reporter.KindCount
	for _, k := range analyzer.NonMergeKinds() {
		if c := byKind[k]; c > 0 {
			kinds = append(kinds, reporter.KindCount{Kind: k, Count: c})
		}
	}
	return n, kinds
}

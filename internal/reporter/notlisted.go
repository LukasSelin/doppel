package reporter

import (
	"fmt"
	"strings"
)

// Which pairs a report's list shows, by what their kind says about them. The
// values are the ones `analyze --non-merges` takes.
const (
	ListInclude = "include" // every pair
	ListExclude = "exclude" // leave out the kinds that say a pair is not a merge
	ListOnly    = "only"    // list those kinds and nothing else
)

// NotListed is what a report's pair list left out on purpose, so the report
// can say so. The zero value — which is what a list that hid nothing
// carries — renders nothing, so such a report is byte-identical to one
// written before the list could be cut.
//
// Leaving a pair out is a reader's default, not a judgement the pipeline
// acts on: the pair was retrieved, compared, ranked and counted like any
// other, and --format json and the dashboard still carry it.
type NotListed struct {
	Mode   string // ListExclude or ListOnly; anything else says nothing was left out
	Total  int    // pairs left out of the list
	Scored int    // pairs the run scored, listed or not

	// InList is how many of the pairs left out would have been in this list
	// had it not been cut — the ones whose place another finding now has —
	// and Kinds counts those by kind, in analyzer.NonMergeKinds order, zero
	// counts omitted. It is the number a reader of a top-20 wants first: the
	// total is mostly pairs ranked far below anything the list shows.
	InList int
	Kinds  []KindCount
}

// KindCount is how many pairs of one kind a list left out.
type KindCount struct {
	Kind  string
	Count int
}

// notListedLine is the one sentence a report prints about a cut list, or ""
// when nothing was cut. md wraps the flag in backticks.
//
//	Not listed: 3 pairs that would have ranked here (mirror operations 2, thin wrappers 1) and 1171 below them, which doppel reads as something other than a merge. --non-merges include lists every pair.
//	Not listed: 7771 of 13517 scored pairs, which doppel reads as something other than a merge; none would have ranked here. --non-merges include lists every pair.
//	Listing only the 25 of 261 scored pairs doppel reads as something other than a merge. --non-merges exclude lists the other 236.
func notListedLine(n NotListed, md bool) string {
	if n.Total == 0 {
		return ""
	}
	flag := func(mode string) string {
		f := "--non-merges " + mode
		if md {
			return "`" + f + "`"
		}
		return f
	}
	const what = "doppel reads as something other than a merge"
	switch n.Mode {
	case ListExclude:
		every := flag(ListInclude) + " lists every pair."
		if n.InList == 0 {
			return fmt.Sprintf("Not listed: %d of %s, which %s; none would have ranked here. %s",
				n.Total, plural(n.Scored, "scored pair"), what, every)
		}
		parts := make([]string, 0, len(n.Kinds))
		for _, k := range n.Kinds {
			parts = append(parts, fmt.Sprintf("%s %d", k.Kind, k.Count))
		}
		head := plural(n.InList, "pair") + " that would have ranked here"
		if len(parts) > 0 {
			head += " (" + strings.Join(parts, ", ") + ")"
		}
		if below := n.Total - n.InList; below > 0 {
			them := "them"
			if n.InList == 1 {
				them = "it"
			}
			head += fmt.Sprintf(" and %d below %s", below, them)
		}
		return fmt.Sprintf("Not listed: %s, which %s. %s", head, what, every)
	case ListOnly:
		return fmt.Sprintf("Listing only the %d of %s %s. %s lists the other %d.",
			n.Scored-n.Total, plural(n.Scored, "scored pair"), what, flag(ListExclude), n.Total)
	}
	return ""
}

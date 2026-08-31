package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/LukasSelin/doppel/internal/dashboard"
)

// printTimelineText is the terminal form of a series: one block per revision,
// with the transition into it.
//
// It exists so the command is usable without a browser, and it is deliberately
// a summary rather than the page in words. Per-function detail is what
// `doppel diff <a> <b>` already renders for one step, in a form this would
// otherwise be a second spelling of.
func printTimelineText(w io.Writer, p dashboard.TimelinePayload) {
	fmt.Fprintf(w, "Timeline: %s — %d revisions\n", p.Target, len(p.Steps))
	fmt.Fprintf(w, "Operating point: %s\n", p.Params)
	for _, n := range p.Notes {
		fmt.Fprintf(w, "Note: %s\n", n)
	}
	fmt.Fprintln(w)

	for i, s := range p.Steps {
		fmt.Fprintf(w, "%s  %d functions, %d pairs (%d merge-worthy)\n",
			s.Label, s.Functions, s.Pairs, s.MergeWorthy)
		if i == 0 {
			// Not "nothing changed" — nothing was observed before it, which
			// is a different statement and the one that is true.
			fmt.Fprintln(w, "    (first revision; no transition into it)")
			fmt.Fprintln(w)
			continue
		}
		c := p.Changes[i-1]
		var parts []string
		for _, cc := range c.Counts {
			if cc.Class == "unchanged" || cc.Count == 0 {
				continue
			}
			parts = append(parts, fmt.Sprintf("%d %s", cc.Count, cc.Class))
		}
		if len(parts) == 0 {
			fmt.Fprintln(w, "    no function changed")
		} else {
			fmt.Fprintf(w, "    %s\n", strings.Join(parts, ", "))
		}
		fmt.Fprintf(w, "    %d pairs created (%d attributable), %d dissolved\n",
			c.CreatedTotal, c.AttributableNew, c.DissolvedTotal)
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "%d tracks listed, %d ran the series unchanged", len(p.Tracks), p.Bounds.FlatTracks)
	if p.Bounds.TracksOmitted > 0 {
		fmt.Fprintf(w, ", %d held back by the display cap", p.Bounds.TracksOmitted)
	}
	fmt.Fprintln(w, ".")
}

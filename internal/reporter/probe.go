package reporter

import (
	"fmt"
	"strings"
)

// maxProbeMatches bounds the near matches listed per probed function. A body
// with an exact-clone family behind it matches every member, and the first
// three already answer the question the note asks.
const maxProbeMatches = 3

// ProbeMatch is one existing function an edited body reads as a near
// duplicate of. Every number is the pipeline's own for that pair, unblended:
// code-shape is fingerprint.Similarity, containment the share of the smaller
// body's structure the larger carries, locality the fraction of the probe's
// depth-2 call-graph ball the match inhabits. Kind is a pair-kind clause, or
// empty when no kind rule fired.
type ProbeMatch struct {
	Key         string
	File        string
	Line        int
	Score       float64
	Containment float64
	Locality    float64
	Kind        string
}

// ProbeResult is one function an edit created or changed, and the near
// matches it has in the corpus as it stands after the edit.
type ProbeResult struct {
	Key     string
	File    string
	Line    int
	New     bool // absent from the session baseline, rather than a changed body
	Matches []ProbeMatch
}

// ProbeDigest renders the post-edit hook's note: per probed function, its
// near matches, then one closing line.
//
// The phrasing follows ConceptDigest's rule — state facts, ask for little —
// because text reading as an out-of-band instruction gets surfaced to the user
// instead of used. The one ask it makes is the cheapest decision there is: a
// near duplicate was just written, so either reuse what exists or say in one
// line why not.
//
// The second result is what the note actually printed — each probe with its
// matches cut to the listed head — so the caller ledgers only findings
// somebody was shown, the rule AgentDigest's second result exists for.
func ProbeDigest(probes []ProbeResult) (string, []ProbeResult) {
	var b strings.Builder
	n := 0
	for _, p := range probes {
		if len(p.Matches) > 0 {
			n++
		}
	}
	if n == 0 {
		return "", nil
	}
	noun := "function"
	if n > 1 {
		noun = "functions"
	}
	var printed []ProbeResult
	fmt.Fprintf(&b, "doppel post-edit: %d %s just written read as near duplicates of existing code:\n", n, noun)
	for _, p := range probes {
		if len(p.Matches) == 0 {
			continue
		}
		state := "edited"
		if p.New {
			state = "new"
		}
		fmt.Fprintf(&b, "  %s (%s, %s:%d)\n", p.Key, state, p.File, p.Line)
		shown := p.Matches
		if len(shown) > maxProbeMatches {
			shown = shown[:maxProbeMatches]
		}
		q := p
		q.Matches = shown
		printed = append(printed, q)
		for _, m := range shown {
			fmt.Fprintf(&b, "    ~ %s  %s:%d  code-shape %.2f  containment %.2f  locality %.2f\n",
				m.Key, m.File, m.Line, m.Score, m.Containment, m.Locality)
			if m.Kind != "" {
				fmt.Fprintf(&b, "      kind: %s\n", m.Kind)
			}
		}
		if more := len(p.Matches) - len(shown); more > 0 {
			fmt.Fprintf(&b, "    (%d further matches not listed)\n", more)
		}
	}
	b.WriteString("Reusing or extending the existing function avoids a copy; if the new one is deliberately separate, one line saying why is enough.\n")
	return truncate(b.String()), printed
}

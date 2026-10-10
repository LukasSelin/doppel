package reporter

import (
	"strings"
	"testing"
)

func TestProbeDigestEmptyWhenNothingMatched(t *testing.T) {
	got, shown := ProbeDigest([]ProbeResult{{Key: "a.F", File: "a/f.go", Line: 3}})
	if got != "" || shown != nil {
		t.Errorf("digest for a probe with no matches = %q, %v; want silence", got, shown)
	}
}

// Only the listed head of each probe's matches is returned as shown, because
// the caller ledgers exactly what it returns: a match counted but not printed
// must stay eligible for a later note.
func TestProbeDigestBoundsMatchesAndReturnsWhatItPrinted(t *testing.T) {
	var ms []ProbeMatch
	for _, k := range []string{"b.One", "b.Two", "b.Three", "b.Four", "b.Five"} {
		ms = append(ms, ProbeMatch{Key: k, File: "b/b.go", Line: 1, Score: 0.9, Containment: 1, Locality: 0.5})
	}
	ms[0].Kind = "subsystem copies — something"
	got, shown := ProbeDigest([]ProbeResult{
		{Key: "a.F", File: "a/f.go", Line: 3, New: true, Matches: ms},
		{Key: "a.G", File: "a/f.go", Line: 9},
	})
	for _, want := range []string{
		"1 function just written",
		"a.F (new, a/f.go:3)",
		"~ b.One  b/b.go:1  code-shape 0.90  containment 1.00  locality 0.50",
		"kind: subsystem copies — something",
		"(2 further matches not listed)",
		"one line saying why",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("digest missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "b.Four") || strings.Contains(got, "a.G") {
		t.Errorf("digest lists past its bound or a probe with no matches:\n%s", got)
	}
	if len(shown) != 1 || len(shown[0].Matches) != maxProbeMatches {
		t.Errorf("shown = %+v; want one probe with %d matches", shown, maxProbeMatches)
	}
}

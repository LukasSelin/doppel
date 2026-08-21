package reporter

import (
	"fmt"
	"strings"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

// scopeMaxPairs bounds the within-package pair list per scoped package. The
// scope digest lands in context right before work starts, so it is held to
// the same standard as the impact digest: few lines, each one a decision.
const scopeMaxPairs = 5

// ScopedPackage is one package a prompt mentioned, resolved against the
// corpus: the package name as the snapshot spells it, and the mention that
// found it, for the header line.
type ScopedPackage struct {
	Package string // snapshot package name
	Mention string // the prompt token that matched, shown to the reader
}

// ScopeDigest renders the corpus facts scoped to the packages a prompt
// mentioned, or "" when there is nothing to say about any of them.
//
// This is the same information ConceptDigest used to spread over the whole
// corpus, spent on the packages actually in play: function count,
// merge-worthy pairs living entirely inside the package, and a count of
// merge-worthy pairs connecting it elsewhere. Phrasing is declarative
// throughout — every line states a fact and asks for nothing.
func ScopeDigest(s snapshot.Snapshot, pkgs []ScopedPackage) string {
	pkgOf := unitPackages(s)

	var b strings.Builder
	for _, sp := range pkgs {
		funcs := 0
		for _, u := range s.Units {
			if u.Package == sp.Package {
				funcs++
			}
		}
		if funcs == 0 {
			continue
		}

		var within []snapshot.Pair
		cross := 0
		for _, p := range s.Pairs {
			if !p.MergeWorthy {
				continue
			}
			ain, bin := pkgOf[p.A] == sp.Package, pkgOf[p.B] == sp.Package
			switch {
			case ain && bin:
				within = append(within, p)
			case ain || bin:
				cross++
			}
		}
		if len(within) == 0 && cross == 0 {
			// A package with no duplication findings earns no lines: the
			// header alone would read as "something to see here" when there
			// is nothing.
			continue
		}

		fmt.Fprintf(&b, "doppel: %s (%d functions)\n", sp.Mention, funcs)
		if len(within) > 0 {
			fmt.Fprintln(&b, "  merge-worthy pairs within this package:")
			shown := 0
			for _, p := range within {
				if shown == scopeMaxPairs {
					fmt.Fprintf(&b, "    (%d more not listed)\n", len(within)-shown)
					break
				}
				fmt.Fprintf(&b, "    %s <-> %s  shape %.2f  overlap %.2f\n", p.A, p.B, p.Score, p.Overlap)
				shown++
			}
		}
		if cross > 0 {
			noun := "pairs connect"
			if cross == 1 {
				noun = "pair connects"
			}
			fmt.Fprintf(&b, "  %d merge-worthy %s this package to others.\n", cross, noun)
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return truncate(b.String())
}

// AdviceDigest renders the merge-worthy twins of the functions in one file,
// or "" when they have none. Facts come from a session-start snapshot, and
// the label says so: the reader must know these predate the session's own
// edits.
func AdviceDigest(s snapshot.Snapshot, relFile string) string {
	inFile := make(map[string]bool)
	for _, u := range s.Units {
		if u.File == relFile {
			inFile[u.Key] = true
		}
	}
	if len(inFile) == 0 {
		return ""
	}

	var lines []string
	for _, p := range s.Pairs {
		if !p.MergeWorthy {
			continue
		}
		if inFile[p.A] || inFile[p.B] {
			lines = append(lines, fmt.Sprintf("  %s <-> %s  shape %.2f  overlap %.2f", p.A, p.B, p.Score, p.Overlap))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	if len(lines) > scopeMaxPairs {
		lines = append(lines[:scopeMaxPairs], fmt.Sprintf("  (%d more not listed)", len(lines)-scopeMaxPairs))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "doppel (as of session start): functions in %s have merge-worthy twins:\n", relFile)
	for _, l := range lines {
		fmt.Fprintln(&b, l)
	}
	return truncate(b.String())
}

// unitPackages maps unit key to package, so pair sides — which are keys —
// can be placed without re-deriving the key format anywhere else.
func unitPackages(s snapshot.Snapshot) map[string]string {
	m := make(map[string]string, len(s.Units))
	for _, u := range s.Units {
		m[u.Key] = u.Package
	}
	return m
}

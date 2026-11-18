package reporter

import (
	"fmt"
	"strings"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// scopeMaxPairs bounds the pre-edit advisory's pair list. It lands in context
// right before an edit, so it is held to the same standard as the impact
// digest: few lines, each one a decision.
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
// One line per package: its function count, how many merge-worthy pairs live
// entirely inside it, and how many connect it to other packages. It used to
// list up to five of the inside pairs as well, and across a month of sessions
// no listed pair was acted on — 827 pair lines, 0 merges — while the hook
// carried 44% of all the text doppel put in front of an agent. The counts are
// what changes how the work is approached; the pairs themselves are what the
// pre-edit and post-edit hooks name when a file in the package is touched.
// Phrasing is declarative — the line states a fact and asks for nothing.
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

		within, cross := 0, 0
		for _, p := range s.Pairs {
			if !p.MergeWorthy || analyzer.NotAMerge(p.Kind) {
				continue
			}
			ain, bin := pkgOf[p.A] == sp.Package, pkgOf[p.B] == sp.Package
			switch {
			case ain && bin:
				within++
			case ain || bin:
				cross++
			}
		}
		var facts string
		switch {
		case within > 0 && cross > 0:
			facts = fmt.Sprintf("%s within the package and %d connecting it to others",
				plural(within, "merge-worthy pair"), cross)
		case within > 0:
			facts = plural(within, "merge-worthy pair") + " within the package"
		case cross > 0:
			facts = plural(cross, "merge-worthy pair") + " connecting it to others"
		default:
			// A package with no duplication findings earns no line: a
			// header with zeros would read as "something to see here" when
			// there is nothing.
			continue
		}
		fmt.Fprintf(&b, "doppel: %s (%s): %s.\n", sp.Mention, plural(funcs, "function"), facts)
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
		if !p.MergeWorthy || analyzer.NotAMerge(p.Kind) {
			continue
		}
		if inFile[p.A] || inFile[p.B] {
			lines = append(lines, fmt.Sprintf("  %s <-> %s  shape %.2f  overlap %.2f%s", p.A, p.B, p.Score, p.Overlap, kindTag(p.Kind)))
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

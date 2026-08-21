package reporter

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// Hook output is capped at 10k characters by the harness, and anything past a
// screenful stops being read anyway. These bound the two digests well under it.
const (
	digestMaxChars = 8000
	maxListed      = 10
	// The impact digest is read at the end of every turn, so it is held
	// tighter than the one-off session-start inventory: a new function pairs
	// with every look-alike it has, and ten near-identical lines about one
	// edit bury the one line that asks for a decision.
	maxImpactListed = 6
	// The agent digest is not merely short, it is expensive: handing a Stop
	// hook's output to the model continues the turn, so every line here costs
	// part of a model turn. Three findings is enough to act on and few enough
	// that the cost stays proportionate.
	maxAgentListed = 3
	// notableDrift is the movement worth interrupting for, an order above
	// driftFloor. driftFloor only asks "is this visible at all"; this asks "is
	// this worth a turn".
	notableDrift = 0.05
)

// ConceptDigest describes a corpus in the terms doppel reasons about, for the
// question "does something like this already exist here?".
//
// The absent-tags line is the one most likely to change a decision and the
// cheapest to compute: "no function in this corpus is tagged retry" is a direct
// answer, where the present-tags list only narrows the search.
//
// Phrasing is deliberately declarative. Text that reads as an out-of-band
// instruction trips prompt-injection defences and gets surfaced to the user
// instead of used, so every line here states a fact about the corpus and asks
// for nothing.
func ConceptDigest(s snapshot.Snapshot, root string) string {
	var b strings.Builder

	// The root is passed in rather than read off the snapshot: an absolute
	// path inside a Snapshot would make its JSON machine-dependent and break
	// the byte-identical invariant that --format json inherits.
	label := root
	if label == "" {
		label = "this repository"
	}
	fmt.Fprintf(&b, "doppel corpus snapshot of %s — %d Go functions", label, s.Functions)
	if s.Params.TestsMode == "exclude" {
		b.WriteString(" (test functions excluded)")
	}
	b.WriteString(".\n")

	if len(s.Concepts) > 0 {
		present := make([]string, 0, len(s.Concepts))
		for _, c := range byCountDesc(s.Concepts) {
			present = append(present, fmt.Sprintf("%s %d", c.Tag, c.Count))
		}
		fmt.Fprintf(&b, "Concept tags present: %s.\n", strings.Join(present, ", "))
	}
	if absent := absentConcepts(s.Concepts); len(absent) > 0 {
		fmt.Fprintf(&b, "Concept tags with no occurrence in this corpus: %s.\n", strings.Join(absent, ", "))
	}

	if len(s.Roles) > 0 {
		roles := make([]string, 0, len(s.Roles))
		for _, r := range s.Roles {
			roles = append(roles, fmt.Sprintf("%s %d", r.Role, r.Count))
		}
		fmt.Fprintf(&b, "Structural roles: %s.\n", strings.Join(roles, ", "))
	}

	if byPkg := conceptsByPackage(s); len(byPkg) > 0 {
		fmt.Fprintf(&b, "Concepts by package: %s.\n", strings.Join(byPkg, "; "))
	}

	merge := s.MergeWorthy()
	fmt.Fprintf(&b, "Near-duplicate pairs reported at threshold %.2f: %d, of which %d are merge-worthy.\n",
		s.Params.Threshold, len(s.Pairs), merge)
	shown := 0
	for _, p := range s.Pairs {
		if !p.MergeWorthy {
			continue
		}
		if shown == maxListed {
			fmt.Fprintf(&b, "  (%d more merge-worthy pairs not listed)\n", merge-shown)
			break
		}
		fmt.Fprintf(&b, "  %s <-> %s  shape %.2f  overlap %.2f\n", p.A, p.B, p.Score, p.Overlap)
		shown++
	}

	// Declarative, like every line above: the command exists whether or not
	// anyone runs it. Without this sentence the query feature is invisible to
	// the one reader positioned to use it before writing code.
	b.WriteString("A proposed function can be checked against this corpus before writing it: doppel query --near <package> <root> reads a Go snippet on stdin and reports its nearest existing relatives.\n")

	return truncate(b.String())
}

// ImpactDigest renders what a session changed, or the empty string when it
// changed nothing.
//
// Emptiness matters: this runs at the end of every turn, and a "no changes"
// line repeated after each one is worse than silence, because it trains the
// reader to skip the place real findings appear.
func ImpactDigest(d snapshot.Delta, deltaPath string) string {
	if !d.Comparable {
		return fmt.Sprintf("doppel impact: baseline not comparable (%s). A new baseline will be taken.", d.Reason)
	}
	if d.Empty() {
		return ""
	}

	var b strings.Builder
	b.WriteString("doppel impact this session: ")
	b.WriteString(strings.Join(scoreboard(d), ", "))
	b.WriteString(".\n")

	added, removed := d.AttributablePairs()
	sortForReading(added)
	sortForReading(removed)
	for _, p := range head(added, maxImpactListed) {
		b.WriteString(fmt.Sprintf("  NEW  %s <-> %s  shape %.2f  overlap %.2f%s\n",
			p.A, p.B, p.Score, p.Overlap, mergeTag(p.MergeWorthy)))
	}
	for _, p := range head(removed, maxImpactListed) {
		b.WriteString(fmt.Sprintf("  GONE %s <-> %s  shape %.2f\n", p.A, p.B, p.Score))
	}

	// Pair movement that no edited function explains is retrieval re-ranking
	// around the change, not a consequence of it. It is worth one counted line
	// so the numbers above reconcile, and no more than that.
	if more := len(added) - min(len(added), maxImpactListed) + len(removed) - min(len(removed), maxImpactListed); more > 0 {
		fmt.Fprintf(&b, "  (%d more pair changes from functions edited this session, not listed)\n", more)
	}
	if churn := len(d.PairsAdded) - len(added) + len(d.PairsRemoved) - len(removed); churn > 0 {
		fmt.Fprintf(&b, "  %d further pair changes involve no function edited this session (retrieval re-ranking).\n", churn)
	}
	if deltaPath != "" {
		fmt.Fprintf(&b, "  Full delta: %s\n", deltaPath)
	}
	return truncate(b.String())
}

func scoreboard(d snapshot.Delta) []string {
	var out []string
	if n := d.FunctionsAfter - d.FunctionsBefore; n != 0 {
		out = append(out, fmt.Sprintf("functions %d -> %d", d.FunctionsBefore, d.FunctionsAfter))
	}
	if len(d.BodiesChanged) > 0 {
		out = append(out, fmt.Sprintf("%d function bodies changed", len(d.BodiesChanged)))
	}
	if d.PairsAfter != d.PairsBefore {
		out = append(out, fmt.Sprintf("candidate pairs %d -> %d", d.PairsBefore, d.PairsAfter))
	}
	if d.MergeAfter != d.MergeBefore {
		out = append(out, fmt.Sprintf("merge-worthy %d -> %d", d.MergeBefore, d.MergeAfter))
	}
	if len(out) == 0 {
		out = append(out, "no net change in counts")
	}
	return out
}

// PrintImpact writes the full delta as text, for someone who wants the detail
// the digest omits.
func PrintImpact(w io.Writer, d snapshot.Delta) {
	fmt.Fprintf(w, "\nDoppel Impact Report\n")
	fmt.Fprintf(w, "====================\n")
	if !d.Comparable {
		fmt.Fprintf(w, "Not comparable: %s\n", d.Reason)
		return
	}
	fmt.Fprintf(w, "Functions: %d -> %d  |  Candidate pairs: %d -> %d  |  Merge-worthy: %d -> %d\n\n",
		d.FunctionsBefore, d.FunctionsAfter, d.PairsBefore, d.PairsAfter, d.MergeBefore, d.MergeAfter)

	section(w, "Functions added", unitLines(d.UnitsAdded))
	section(w, "Functions removed", unitLines(d.UnitsRemoved))
	section(w, "Bodies changed", unitLines(d.BodiesChanged))
	section(w, "Pairs added", pairChangeLines(d.PairsAdded))
	section(w, "Pairs removed", pairChangeLines(d.PairsRemoved))

	var drift []string
	for _, dr := range d.Drift {
		drift = append(drift, fmt.Sprintf("%s <-> %s  %.4f -> %.4f%s",
			dr.A, dr.B, dr.ScoreBefore, dr.ScoreAfter, attributionTag(dr.Attributable)))
	}
	section(w, "Shape-score drift", drift)
}

func section(w io.Writer, title string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(w, "%s (%d)\n", title, len(lines))
	for _, l := range lines {
		fmt.Fprintf(w, "  %s\n", l)
	}
	fmt.Fprintln(w)
}

func unitLines(units []snapshot.Unit) []string {
	out := make([]string, 0, len(units))
	for _, u := range units {
		out = append(out, fmt.Sprintf("%s  %s:%d", u.Key, u.File, u.Line))
	}
	return out
}

func pairChangeLines(pairs []snapshot.PairChange) []string {
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, fmt.Sprintf("%s <-> %s  shape %.4f  overlap %.4f%s%s",
			p.A, p.B, p.Score, p.Overlap, mergeTag(p.MergeWorthy), attributionTag(p.Attributable)))
	}
	return out
}

func mergeTag(worthy bool) string {
	if worthy {
		return "  (merge-worthy)"
	}
	return ""
}

// attributionTag marks the changes no edit in this session explains, so a
// reader is never left to assume their work caused corpus churn.
func attributionTag(attributable bool) string {
	if attributable {
		return ""
	}
	return "  [not attributable to an edited function]"
}

func head[T any](s []T, n int) []T {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// sortForReading orders pair changes so the few the digest has room to print
// are the ones worth printing.
//
// Diff hands these over in snapshot order, which is score-first — fine when a
// session touches one function, useless on a corpus where dozens of pairs move
// at once and the six that get listed are then whichever happened to score
// highest on code shape alone. Merge-worthiness leads because it is the only
// bit here that claims a decision is warranted, and overlap breaks its ties
// because a shared architectural context is what separates a real merge
// candidate from two look-alike bodies in unrelated subsystems.
//
// This is presentation, and only presentation: the delta still carries every
// pair, the hook still diffs the full candidate set, and nothing here feeds a
// score. The trailing name keys make the order total, which the repo's
// byte-identical-output invariant requires — without them, two pairs equal on
// all three numbers could swap between runs.
func sortForReading(pairs []snapshot.PairChange) {
	sort.SliceStable(pairs, func(i, j int) bool {
		a, b := pairs[i], pairs[j]
		if a.MergeWorthy != b.MergeWorthy {
			return a.MergeWorthy
		}
		if a.Overlap != b.Overlap {
			return a.Overlap > b.Overlap
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.A != b.A {
			return a.A < b.A
		}
		return a.B < b.B
	})
}

// byCountDesc orders tags by how common they are, for reading. The snapshot
// itself stays sorted by tag: count ties are the common case, so count order is
// only ever a presentation choice and must not decide stored order.
func byCountDesc(tags []snapshot.TagCount) []snapshot.TagCount {
	out := append([]snapshot.TagCount(nil), tags...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Tag < out[j].Tag
	})
	return out
}

// absentConcepts lists the vocabulary's concrete concepts that nothing in the
// corpus carries — the direct answer to "is there already something doing this
// here?" when the answer is no.
func absentConcepts(present []snapshot.TagCount) []string {
	have := make(map[string]bool, len(present))
	for _, t := range present {
		have[t.Tag] = true
	}
	var out []string
	for _, term := range ontology.Default().TermsOfKind(ontology.KindConcept) {
		if term.Abstract || have[string(term.ID)] {
			continue
		}
		out = append(out, string(term.ID))
	}
	sort.Strings(out)
	return out
}

func conceptsByPackage(s snapshot.Snapshot) []string {
	byPkg := make(map[string]map[string]bool)
	for _, u := range s.Units {
		if len(u.Patterns) == 0 {
			continue
		}
		if byPkg[u.Package] == nil {
			byPkg[u.Package] = make(map[string]bool)
		}
		for _, p := range u.Patterns {
			byPkg[u.Package][p] = true
		}
	}
	pkgs := make([]string, 0, len(byPkg))
	for p := range byPkg {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	if len(pkgs) > maxListed {
		pkgs = pkgs[:maxListed]
	}
	out := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		tags := make([]string, 0, len(byPkg[p]))
		for t := range byPkg[p] {
			tags = append(tags, t)
		}
		sort.Strings(tags)
		out = append(out, fmt.Sprintf("%s — %s", p, strings.Join(tags, ", ")))
	}
	return out
}

func truncate(s string) string {
	if len(s) <= digestMaxChars {
		return s
	}
	cut := strings.LastIndex(s[:digestMaxChars], "\n")
	if cut <= 0 {
		cut = digestMaxChars
	}
	return s[:cut] + "\n  (truncated)\n"
}

// A Finding is one thing about the duplication surface worth an agent's
// attention. Key is stable across runs so a caller can remember what it has
// already surfaced; Line is the rendered sentence.
type Finding struct {
	Key  string
	Line string
}

// Notable selects the findings from a delta that justify occupying context.
//
// The bar is deliberately far higher than the user-facing digest's. That digest
// is read by someone who chose to look at it; this one interrupts. Two kinds
// qualify, and nothing else does:
//
//   - a new near-duplicate that is both attributable to an edit this session
//     and merge-worthy — code that was just written and already has a twin;
//   - a pair with an edited side that crossed the merge-worthy gate, or,
//     failing that, moved by notableDrift.
//
// Attribution gates the crossings as hard as it gates the new pairs, and
// measurement proved it has to. Merge-worthiness is corpus-weighted: adding two
// functions anywhere shifts the information content of every tag and nudges
// pairs nobody touched across the 0.4 boundary. Reporting those would blame a
// session for arithmetic it did not cause — the same reason Delta omits role
// changes and caller counts.
//
// Everything the impact digest also prints — count scoreboards, pairs below the
// gate, the retrieval re-ranking line — is excluded on purpose. A count that
// moved is not a finding, and a look-alike below the gate is not a decision.
//
// Results are ordered gate-crossings-last-in, findings-first: new duplication
// leads, because it is the only category describing something that did not
// exist before this session.
func Notable(d snapshot.Delta) []Finding {
	if !d.Comparable {
		return nil
	}
	var out []Finding

	added, _ := d.AttributablePairs()
	sortForReading(added)
	for _, p := range added {
		if !p.MergeWorthy {
			continue
		}
		out = append(out, Finding{
			Key:  "new:" + p.A + "|" + p.B,
			Line: fmt.Sprintf("%s <-> %s  shape %.2f  overlap %.2f", p.A, p.B, p.Score, p.Overlap),
		})
	}

	for _, dr := range d.Drift {
		switch {
		case dr.CrossedGate() && dr.Attributable:
			verb := "became merge-worthy"
			if !dr.MergeWorthyAfter {
				verb = "is no longer merge-worthy"
			}
			out = append(out, Finding{
				Key: "gate:" + dr.A + "|" + dr.B,
				Line: fmt.Sprintf("%s <-> %s  %s (overlap %.3f -> %.3f)",
					dr.A, dr.B, verb, dr.OverlapBefore, dr.OverlapAfter),
			})
		case dr.Attributable && dr.ScoreMoved() >= notableDrift:
			out = append(out, Finding{
				Key: "drift:" + dr.A + "|" + dr.B,
				Line: fmt.Sprintf("%s <-> %s  shape %.2f -> %.2f",
					dr.A, dr.B, dr.ScoreBefore, dr.ScoreAfter),
			})
		}
	}
	return out
}

// AgentDigest renders findings as a note for the model, or "" for none.
//
// Phrasing follows the same rule as ConceptDigest, and for a sharper reason:
// this text arrives while the turn is being deliberately continued, so a line
// that reads as an instruction would be an out-of-band order to go refactor
// something the user never asked about. Every line states a fact and asks for
// nothing. The closing sentence exists to bound that explicitly.
func AgentDigest(findings []Finding) string {
	if len(findings) == 0 {
		return ""
	}

	noun := "near-duplicate findings"
	if len(findings) == 1 {
		noun = "near-duplicate finding"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "doppel measured this session's effect on the repository's duplication surface and found %d new %s:\n",
		len(findings), noun)
	for _, f := range head(findings, maxAgentListed) {
		fmt.Fprintf(&b, "  %s\n", f.Line)
	}
	if more := len(findings) - maxAgentListed; more > 0 {
		fmt.Fprintf(&b, "  (%d further findings not listed)\n", more)
	}
	b.WriteString("This is a measurement, not a request. No change is required.\n")
	return truncate(b.String())
}

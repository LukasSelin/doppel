package reporter

import (
	"fmt"
	"strings"

	"github.com/LukasSelin/doppel/internal/identity"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// Bounds on the delta section, in the same spirit as maxImpactListed and
// maxAgentListed: the digests are read at the end of every turn, and a section
// that grows with the session stops being read long before it stops being
// printed.
const (
	// The user digest can afford the fuller list — it lets the turn end.
	maxDeltaChanges = 6
	maxDeltaPairs   = 6
	// The agent note costs part of a model turn per line, so it gets the same
	// three-finding budget the notable list has.
	maxAgentDelta = 3
)

// DeltaSection renders the delta report — what happened to each function since
// the baseline, then the pairs those changes created or dissolved — or the
// empty string when nothing happened.
//
// This is the section both Stop-hook digests lead with, and the reason it leads
// is attribution. snapshot.Delta can say a pair appeared and that one of its
// sides is new; only this can say the side is new *because a function was
// renamed into it*, which is the difference between a reader checking a finding
// and a reader dismissing one.
//
// Nothing here is recomputed or re-rendered: the class lines are
// identity.Lines and the pair lines identity.PairLines, exactly as `doppel
// diff` prints them. A digest that spelled these differently would be a second
// rendering to keep in step with the first.
func DeltaSection(d identity.Delta, maxChanges, maxPairs int) string {
	if !d.Comparable || d.Empty() {
		return ""
	}

	var b strings.Builder
	b.WriteString("doppel delta since the session baseline: ")
	b.WriteString(deltaScoreboard(d))
	b.WriteString(".\n")

	changes := classifiedChanges(d)
	for _, c := range head(changes, maxChanges) {
		writeIndented(&b, identity.Lines(c))
	}
	if more := len(changes) - min(len(changes), maxChanges); more > 0 {
		fmt.Fprintf(&b, "  (%d more classified functions, not listed)\n", more)
	}

	writePairs(&b, "NEW  ", d.Created, maxPairs)
	writePairs(&b, "GONE ", d.Dissolved, maxPairs)
	return b.String()
}

// writePairs prints a bounded head of one pair list, tagged so a created pair
// and a dissolved one are distinguishable at a glance — the two lists are
// otherwise identically shaped.
func writePairs(b *strings.Builder, tag string, ps []identity.PairChange, max int) {
	shown := head(ps, max)
	for _, p := range shown {
		lines := identity.PairLines(p)
		fmt.Fprintf(b, "  %s%s\n", tag, lines[0])
		writeIndented(b, lines[1:])
	}
	if more := len(ps) - len(shown); more > 0 {
		fmt.Fprintf(b, "  (%d more, not listed)\n", more)
	}
}

func writeIndented(b *strings.Builder, lines []string) {
	for _, l := range lines {
		fmt.Fprintf(b, "  %s\n", l)
	}
}

// deltaScoreboard is the one-line census: every class with a finding, then the
// pair counts. Unchanged is excluded — it is the bulk of any nearby comparison
// and says nothing about the session.
func deltaScoreboard(d identity.Delta) string {
	var parts []string
	for _, cc := range d.Counts {
		if cc.Class == identity.Unchanged || cc.Count == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d", cc.Class, cc.Count))
	}
	if len(parts) == 0 {
		parts = append(parts, "no function reclassified")
	}
	return strings.Join(parts, ", ") +
		fmt.Sprintf("; pairs created %d, dissolved %d", len(d.Created), len(d.Dissolved))
}

// classifiedChanges is every finding that is not `unchanged`, in the order
// identity already sorted them — structural relocations first, then renames,
// edits, and the population changes. That order is the digest's priority order
// too, which is why nothing re-sorts here.
func classifiedChanges(d identity.Delta) []identity.Change {
	out := make([]identity.Change, 0, len(d.Changes))
	for _, c := range d.Changes {
		if c.Class == identity.Unchanged {
			continue
		}
		out = append(out, c)
	}
	return out
}

// SessionDigest is the Stop hook's systemMessage: the delta report, then the
// impact scoreboard.
//
// Composing here rather than concatenating two finished digests is what keeps
// the whole thing inside one byte budget — truncate applied twice bounds each
// half at digestMaxChars and the sum at twice that, which is past what the
// harness will carry.
//
// It returns "" exactly when ImpactDigest would: an identity finding always
// implies a snapshot delta (every class but unchanged moves a key or a digest,
// which is UnitsAdded, UnitsRemoved or BodiesChanged), so the impact half is
// the stricter emptiness test and the one that decides silence.
func SessionDigest(id identity.Delta, d snapshot.Delta, deltaPath string) string {
	impact := impactBody(d, deltaPath)
	if impact == "" {
		return ""
	}
	return truncate(DeltaSection(id, maxDeltaChanges, maxDeltaPairs) + impact)
}

// DeltaFindings turns the delta report into ledgerable findings, so the agent
// note can say a thing once per session rather than on every turn that has
// anything to say at all.
//
// A delta is cumulative against the session-start origin, exactly like the
// notable findings the ledger already holds: without this, one rename would be
// restated in every later agent note for the rest of the session. Keys follow
// the existing shape — a prefix naming what kind of finding it is, then the
// identities joined by a pipe — and the prefixes are distinct from `new:`,
// `gate:` and `drift:` so two kinds of finding can never collide on one key.
//
// Unchanged functions are not findings and produce no keys.
func DeltaFindings(d identity.Delta) []Finding {
	if !d.Comparable {
		return nil
	}
	var out []Finding
	for _, c := range classifiedChanges(d) {
		out = append(out, Finding{
			Key:  "class:" + string(c.Class) + ":" + changeKey(c),
			Line: strings.Join(identity.Lines(c), "\n  "),
		})
	}
	for _, p := range d.Created {
		out = append(out, pairFinding("created", p))
	}
	for _, p := range d.Dissolved {
		out = append(out, pairFinding("dissolved", p))
	}
	return out
}

func pairFinding(kind string, p identity.PairChange) Finding {
	lines := identity.PairLines(p)
	if kind == "created" {
		lines[0] = "NEW  " + lines[0]
	} else {
		lines[0] = "GONE " + lines[0]
	}
	return Finding{
		Key:  kind + ":" + p.A + "|" + p.B,
		Line: strings.Join(lines, "\n  "),
	}
}

// changeKey names a finding by the identities it is about, old side then new.
// A split or a merge is keyed on its pivot and its first participant, which is
// enough to be stable across turns: the pivot cannot participate in a second
// finding, because absorption reports every function exactly once.
func changeKey(c identity.Change) string {
	var oldKey, newKey string
	if len(c.Old) > 0 {
		oldKey = c.Old[0].Key
	}
	if len(c.New) > 0 {
		newKey = c.New[0].Key
	}
	return oldKey + "|" + newKey
}

// AgentDeltaDigest renders the note that continues the turn: the delta report
// first, then the notable findings.
//
// The two arguments are not symmetric, and the asymmetry is the whole design.
// `notable` is what justifies interrupting — it is the caller's already-filtered
// Notable list, and an empty one means this function is never called. `fresh` is
// the delta report's own unreported findings, which change *what* the note says
// and never *whether* it is sent. A rename with no pair consequence is a fact
// the user digest carries and the model already knows: it does not buy a model
// turn.
//
// The second return is what the caller must ledger — the findings actually
// rendered, from both halves. Recording more would retire findings nobody was
// ever shown; recording less would repeat them.
func AgentDeltaDigest(d identity.Delta, fresh, notable []Finding) (string, []Finding) {
	if len(notable) == 0 {
		return "", nil
	}

	var b strings.Builder
	var shown []Finding

	if len(fresh) > 0 && d.Comparable {
		fmt.Fprintf(&b, "doppel matched this session's functions against the session baseline: %s.\n",
			deltaScoreboard(d))
		shown = head(fresh, maxAgentDelta)
		for _, f := range shown {
			fmt.Fprintf(&b, "  %s\n", f.Line)
		}
		if more := len(fresh) - len(shown); more > 0 {
			fmt.Fprintf(&b, "  (%d further changes not listed)\n", more)
		}
	}

	note, notableShown := AgentDigest(notable)
	b.WriteString(note)
	shown = append(shown, notableShown...)
	return truncate(b.String()), shown
}

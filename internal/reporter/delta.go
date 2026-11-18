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
	//
	// The two bounds are different numbers because the two lines cost
	// different amounts. A classified change is two lines; a pair change is
	// three, since it carries both its attribution and its stored
	// explanation. Three pairs each way is nine lines, which is the same
	// weight the impact half's six one-line entries carry. It is also enough:
	// a rename re-keys every pair its function held, so the tail of these
	// lists is one fact restated, and the head is where the new duplication
	// is.
	maxDeltaChanges = 6
	maxDeltaPairs   = 3
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
	return classTally(d) +
		fmt.Sprintf("; pairs created %d, dissolved %d", len(d.Created), len(d.Dissolved))
}

// classTally is the class half of the scoreboard. The pane's overview leads
// with it alone, because it states the pair half in its own sections.
func classTally(d identity.Delta) string {
	var parts []string
	for _, cc := range d.Counts {
		if cc.Class == identity.Unchanged || cc.Count == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d", cc.Class, cc.Count))
	}
	if len(parts) == 0 {
		return "no function reclassified"
	}
	return strings.Join(parts, ", ")
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

// AgentDeltaDigest renders the note that continues the turn: the notable
// findings, each pair side labelled with what happened to it since the
// session baseline.
//
// `notable` alone decides whether there is a note — it is the caller's
// already-filtered Notable list, and an empty one renders nothing. The identity
// delta changes only how the note reads.
//
// The note used to lead with the delta report itself — the classification
// scoreboard and the first few classified functions and created pairs, the
// section the user digest still leads with. Over a month of transcripts that
// preamble was 55% of the text Stop notes put in front of an agent, and no
// agent acted on anything it listed. What it was there for is attribution:
// telling a reader which side of a pair the session wrote. A label on each
// side says that in a word, at the place it is read — `svc.Clip [new] <->
// svc.Trim [unchanged]` is a new function beside an existing one, the shape
// most of the merges agents made had.
//
// The second return is what the caller must ledger: every notable finding, as
// AgentDigest retires them.
func AgentDeltaDigest(d identity.Delta, notable []Finding) (string, []Finding) {
	if len(notable) == 0 {
		return "", nil
	}
	if !d.Comparable {
		return AgentDigest(notable)
	}
	classes := sideClasses(d)
	labelled := make([]Finding, len(notable))
	for i, f := range notable {
		labelled[i] = withSides(f, classes)
	}
	note, _ := AgentDigest(labelled)
	return note, notable
}

// sideClasses maps each function's key in the session's current tree to what
// happened to it since the baseline. A key absent from the map was matched
// unchanged: identity reports every function in exactly one change.
func sideClasses(d identity.Delta) map[string]identity.Class {
	out := make(map[string]identity.Class)
	for _, c := range classifiedChanges(d) {
		for _, m := range c.New {
			out[m.Key] = c.Class
		}
	}
	return out
}

// withSides labels the two sides of a pair finding with their classes. A
// finding that is not about one pair, or whose line does not open with its
// sides, is returned as it was.
func withSides(f Finding, classes map[string]identity.Class) Finding {
	pair := f.A + " <-> " + f.B
	if f.A == "" || f.B == "" || !strings.HasPrefix(f.Line, pair) {
		return f
	}
	f.Line = sideLabel(f.A, classes) + " <-> " + sideLabel(f.B, classes) + f.Line[len(pair):]
	return f
}

func sideLabel(key string, classes map[string]identity.Class) string {
	c, ok := classes[key]
	if !ok {
		c = identity.Unchanged
	}
	return key + " [" + string(c) + "]"
}

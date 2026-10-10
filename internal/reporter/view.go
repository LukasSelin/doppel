package reporter

import (
	"fmt"
	"strings"

	"github.com/LukasSelin/doppel/internal/identity"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// SessionView is the user-side form of the Stop hook's measurement: one line
// for the band above the prompt, a bounded overview the pane opens on, and the
// full delta report the pane can switch to — all rendered here so the plugin's
// mod draws text it never composes.
//
// It exists because of a constraint the Stop hook cannot get around: text a
// Stop hook puts in front of the model continues the turn. A mod draws in the
// UI and never reaches the model's context, so it is a channel to the user that
// costs no token and no turn — which is why it can afford to show everything
// the session did, where the agent note shows only what clears Notable.
//
// Nothing here is a second rendering. The band is deltaScoreboard, the line
// that heads DeltaSection; the report is identity.PrintDelta, the text `doppel
// diff` prints. One rendering, three surfaces. The overview is the one text
// made for the pane alone, and it selects from the report rather than
// rewording it: a session's full report grows with every function it touched
// and every pair a rename re-keyed, so by the end of a long session it is a
// list to scroll rather than a picture to read.
type SessionView struct {
	Band     string `json:"band"`
	Overview string `json:"overview"`
	Report   string `json:"report"`
}

// SessionViewOf renders the view, or false exactly when SessionDigest would
// return "" — the band is silent when the user digest is, so the two user-side
// surfaces can never disagree about whether the session has done anything.
//
// The identity delta is preferred because it attributes; when it is
// unavailable (identityDelta degraded to its zero value) the view falls back to
// the impact half, untruncated, rather than showing a "not comparable" line for
// a comparison that snapshot.Diff did make. The impact half is already a
// bounded summary, so it serves as the overview too.
func SessionViewOf(id identity.Delta, d snapshot.Delta) (SessionView, bool) {
	impact := impactBody(d, "")
	if impact == "" || !d.Comparable {
		return SessionView{}, false
	}
	if id.Comparable && !id.Empty() {
		var b strings.Builder
		identity.PrintDelta(&b, id, false)
		return SessionView{
			Band:     "doppel: " + deltaScoreboard(id) + " — since session start",
			Overview: sessionOverview(id),
			Report:   b.String(),
		}, true
	}
	return SessionView{
		Band:     "doppel: " + strings.Join(scoreboard(d), ", ") + " — since session start",
		Overview: impact,
		Report:   impact,
	}, true
}

// The overview's bounds. Merge-worthy pairs are what a reader acts on, so each
// direction gets its own budget; the change list is capped harder because the
// full report carries every line of it with evidence.
const (
	overviewPairs   = 5
	overviewChanges = 8
)

// sessionOverview is the pane's first screen: what the session did, in the
// order a reader acts on it.
//
// Merge-worthy pairs lead, created then dissolved — duplication the session
// introduced, then duplication it removed. Then one line per classified
// function, without the evidence line the full report puts under each. Every
// other pair change is a count: a pair below the merge verdict is a candidate
// rather than a finding, and a rename restates each pair its function held as
// one created and one dissolved, so listing them is most of what makes the
// full report long. Pairs nothing in the session explains are counted apart,
// because they are retrieval re-ranking rather than the session's doing.
// Every list says how much it left out.
func sessionOverview(d identity.Delta) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s; functions %d -> %d\n", classTally(d), d.OldFunctions, d.NewFunctions)

	created, createdRest := splitMergeWorthy(d.Created)
	dissolved, dissolvedRest := splitMergeWorthy(d.Dissolved)
	overviewPairSection(&b, "merge-worthy pairs created", created)
	overviewPairSection(&b, "merge-worthy pairs dissolved", dissolved)

	if changes := classifiedChanges(d); len(changes) > 0 {
		fmt.Fprintf(&b, "\nfunctions changed %d\n", len(changes))
		for _, c := range head(changes, overviewChanges) {
			fmt.Fprintf(&b, "  %-8s %s\n", c.Class, changeSummary(c))
		}
		moreLine(&b, len(changes)-overviewChanges)
	}

	ca, cc := countAttribution(createdRest)
	da, dc := countAttribution(dissolvedRest)
	if ca+da+cc+dc > 0 {
		b.WriteString("\n")
	}
	if ca+da > 0 {
		fmt.Fprintf(&b, "other pairs from these changes: %d created, %d dissolved (below merge-worthy)\n", ca, da)
	}
	if cc+dc > 0 {
		fmt.Fprintf(&b, "pairs no change explains: %d created, %d dissolved (retrieval re-ranking)\n", cc, dc)
	}
	return b.String()
}

// splitMergeWorthy partitions one pair list, keeping each half in the order
// identity sorted it.
func splitMergeWorthy(ps []identity.PairChange) (worthy, rest []identity.PairChange) {
	for _, p := range ps {
		if p.MergeWorthy {
			worthy = append(worthy, p)
		} else {
			rest = append(rest, p)
		}
	}
	return worthy, rest
}

// countAttribution counts the pairs a classified change explains and the ones
// it does not.
func countAttribution(ps []identity.PairChange) (attributed, churn int) {
	for _, p := range ps {
		if p.Attributable() {
			attributed++
		} else {
			churn++
		}
	}
	return attributed, churn
}

// overviewPairSection is PairLines' headline with the cause folded onto it,
// one line per pair: the explain sentence is the full report's to carry.
func overviewPairSection(b *strings.Builder, title string, ps []identity.PairChange) {
	if len(ps) == 0 {
		return
	}
	fmt.Fprintf(b, "\n%s %d\n", title, len(ps))
	for _, p := range head(ps, overviewPairs) {
		fmt.Fprintf(b, "  %s <-> %s  shape %.2f  (%s)\n", p.A, p.B, p.Score, identity.CauseLine(p))
	}
	moreLine(b, len(ps)-overviewPairs)
}

// changeSummary is one change on one line: the keys it moved between, and a
// location only for a new function, the one case where the key cannot be
// looked up in code the reader already knows.
func changeSummary(c identity.Change) string {
	switch c.Class {
	case identity.Split:
		return c.Old[0].Key + " -> " + joinKeys(c.New)
	case identity.Merged:
		return joinKeys(c.Old) + " -> " + c.New[0].Key
	case identity.Added:
		return fmt.Sprintf("%s  %s:%d", c.New[0].Key, c.New[0].File, c.New[0].Line)
	case identity.Deleted:
		return c.Old[0].Key
	}
	if c.Old[0].Key != c.New[0].Key {
		return c.Old[0].Key + " -> " + c.New[0].Key
	}
	return c.New[0].Key
}

func joinKeys(ms []identity.Member) string {
	keys := make([]string, len(ms))
	for i, m := range ms {
		keys[i] = m.Key
	}
	return strings.Join(keys, ", ")
}

func moreLine(b *strings.Builder, n int) {
	if n > 0 {
		fmt.Fprintf(b, "  … %d more in the full report\n", n)
	}
}

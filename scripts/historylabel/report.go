package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// report prints the coverage of the candidate set and the agreement with the
// hand labels. Agreement is the number that decides whether history can be
// trusted as an oracle on this corpus at all, so it is printed both coarsely
// (worth acting on or not) and by class.
func report[P comparable](w io.Writer, keys []P, verdicts map[P]verdict, hand map[P]handLabel, candidates int, weak bool) {
	dist := map[string]int{}
	for _, p := range keys {
		if _, isHand := hand[p]; isHand {
			continue
		}
		dist[verdicts[p].Verdict]++
	}
	fmt.Fprintf(w, "## Candidate set (%d pairs, hand-labelled ones excluded)\n\n", candidates)
	for _, v := range append(verdictOrder, vNone) {
		name := v
		if name == "" {
			name = "no evidence"
		}
		fmt.Fprintf(w, "  %-14s %4d\n", name, dist[v])
	}

	if len(hand) == 0 {
		return
	}
	var hk []P
	for _, p := range keys {
		if _, ok := hand[p]; ok {
			hk = append(hk, p)
		}
	}
	fmt.Fprintf(w, "\n## Hand labels against history (%d pairs)\n\n", len(hk))
	type row struct{ class, line string }
	var rows []row
	confusion := map[string]map[string]int{}
	agreeAct, decidedAct, agreeClass, coupled := 0, 0, 0, 0
	for _, p := range hk {
		l, v := hand[p], verdicts[p]
		got := classOf(v.Verdict, weak)
		if confusion[l.Class] == nil {
			confusion[l.Class] = map[string]int{}
		}
		vn := v.Verdict
		if vn == "" {
			vn = "-"
		}
		confusion[l.Class][vn]++
		mark := " "
		if got == "coupled" {
			// coupled claims the two are kept in step, not whether they
			// should be merged, so it neither agrees nor disagrees with a
			// hand verdict on mergeability.
			mark = "~"
			coupled++
		} else if got != "" {
			decidedAct++
			if (got == "false_positive") == (l.Class == "false_positive") {
				agreeAct++
				mark = "="
			} else {
				mark = "x"
			}
			if got == l.Class {
				agreeClass++
			}
		}
		var ev []string
		for _, e := range v.Evidence {
			ev = append(ev, fmt.Sprintf("%s %s: %s", e.Kind, strings.Join(e.Commits, ","), e.Detail))
		}
		line := fmt.Sprintf("%s %-15s %-13s %s <-> %s\n      origin: %s\n", mark, l.Class, vn, l.A, l.B, v.Origin)
		for _, e := range ev {
			line += "      " + e + "\n"
		}
		rows = append(rows, row{l.Class, line})
	}
	order := map[string]int{"merge": 0, "refactor": 1, "false_positive": 2}
	sort.SliceStable(rows, func(i, j int) bool { return order[rows[i].class] < order[rows[j].class] })
	for _, r := range rows {
		fmt.Fprint(w, r.line)
	}

	fmt.Fprintf(w, "\n## Confusion (rows: hand class, columns: history verdict)\n\n")
	cols := append(append([]string{}, verdictOrder...), "-")
	fmt.Fprintf(w, "  %-15s", "")
	for _, c := range cols {
		fmt.Fprintf(w, " %13s", c)
	}
	fmt.Fprintln(w)
	for _, cls := range []string{"merge", "refactor", "false_positive"} {
		fmt.Fprintf(w, "  %-15s", cls)
		for _, c := range cols {
			fmt.Fprintf(w, " %13d", confusion[cls][c])
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "\n  history decided %d of %d hand-labelled pairs\n", decidedAct, len(hk))
	if decidedAct > 0 {
		fmt.Fprintf(w, "  actionable (merge|refactor vs false_positive) agreement: %d/%d\n", agreeAct, decidedAct)
		fmt.Fprintf(w, "  exact class agreement: %d/%d\n", agreeClass, decidedAct)
	}
}

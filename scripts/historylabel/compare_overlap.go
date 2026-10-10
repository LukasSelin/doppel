package main

// The two compare-summary sections examples/clone-outcomes.md added, both
// behind flags so the ranker-outcome summary renders exactly as it did.

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
)

// listKeys is one method's kept pairs as unordered key pairs, in rank order.
func listKeys(c compareOut, method string) []upair {
	for _, l := range c.Lists {
		if l.Method == method {
			out := make([]upair, len(l.Pairs))
			for i, x := range l.Pairs {
				out[i] = normPair(c.Pairs[x].A, c.Pairs[x].B)
			}
			return out
		}
	}
	return nil
}

// writeReferenceCheck states, per corpus, whether doppel's list is the
// earlier study's doppel list pair for pair and in order: the check that the
// two studies ranked the same tree with the same pipeline.
func writeReferenceCheck(w io.Writer, corpora []compareOut, refDir string) {
	fmt.Fprintf(w, "### Validity: doppel's list against the earlier study's\n\n")
	fmt.Fprintf(w, "| corpus | pairs | identical, in order |\n| --- | ---: | --- |\n")
	for _, c := range corpora {
		var ref compareOut
		if err := readJSON(filepath.Join(refDir, c.Corpus+".outcomes.json"), &ref); err != nil {
			fmt.Fprintf(w, "| %s | — | %v |\n", c.Corpus, err)
			continue
		}
		mine := listKeys(c, doppelMethod)
		fmt.Fprintf(w, "| %s | %d | %s |\n", c.Corpus, len(mine), sameList(mine, listKeys(ref, doppelMethod)))
	}
	fmt.Fprintln(w)
}

// sameList says whether two ranked lists are the same pairs in the same order.
func sameList(mine, theirs []upair) string {
	if len(mine) != len(theirs) {
		return fmt.Sprintf("**no** (%d against %d pairs)", len(mine), len(theirs))
	}
	for i := range mine {
		if mine[i] != theirs[i] {
			return fmt.Sprintf("**no** (first difference at rank %d)", i+1)
		}
	}
	return "yes"
}

// writeOverlap is the descriptive table: per method, how many of its top
// primaryK are in doppel's, the median smallest side in lines at T, and how
// many of its top primaryK carry any event.
func writeOverlap(w io.Writer, corpora []compareOut) {
	fmt.Fprintf(w, "\n### Overlap, size and event counts at %d\n\n", primaryK)
	fmt.Fprintf(w, "Overlap is the number of a method's top %d also in doppel's top %d. Median smallest side is in lines at T. Pairs with an event is a count, not a share.\n\n", primaryK, primaryK)
	fmt.Fprintf(w, "| corpus | method | n@%d | in doppel's top %d | median smallest side | pairs with an event |\n", primaryK, primaryK)
	fmt.Fprintf(w, "| --- | --- | ---: | ---: | ---: | ---: |\n")
	for _, c := range corpora {
		views := viewsOf(c)
		dop := map[upair]bool{}
		for _, v := range views {
			if v.name == doppelMethod {
				for _, i := range v.slice(1, primaryK) {
					dop[normPair(c.Pairs[i].A, c.Pairs[i].B)] = true
				}
			}
		}
		for _, v := range views {
			idx := v.slice(1, primaryK)
			in, events := 0, 0
			sizes := make([]int, 0, len(idx))
			for _, i := range idx {
				p := c.Pairs[i]
				if dop[normPair(p.A, p.B)] {
					in++
				}
				if anyEvent(p) {
					events++
				}
				sizes = append(sizes, min(p.LinesA, p.LinesB))
			}
			med := "—"
			if len(sizes) > 0 {
				sort.Ints(sizes)
				med = fmt.Sprint(sizes[(len(sizes)-1)/2])
			}
			fmt.Fprintf(w, "| %s | %s | %d | %d | %s | %d |\n", c.Corpus, v.name, len(idx), in, med, events)
		}
	}
}

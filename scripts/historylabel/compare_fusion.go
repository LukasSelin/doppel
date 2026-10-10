package main

// The two compare-summary sections examples/fusion-outcomes.md added, behind
// flags so the earlier studies' summaries render exactly as they did.

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// writeInputsCheck states, per corpus and per list this file shares with the
// clone-outcome study's file, whether the two lists are the same pairs in the
// same order: the check that the fusion's inputs are that study's lists.
func writeInputsCheck(w io.Writer, corpora []compareOut, dir string) {
	fmt.Fprintf(w, "### Validity: input lists against the clone-outcome study's\n\n")
	fmt.Fprintf(w, "| corpus | method | pairs | identical, in order |\n| --- | --- | ---: | --- |\n")
	for _, c := range corpora {
		var ref compareOut
		if err := readJSON(filepath.Join(dir, c.Corpus+".clone-outcomes.json"), &ref); err != nil {
			fmt.Fprintf(w, "| %s | — | — | %v |\n", c.Corpus, err)
			continue
		}
		for _, l := range c.Lists {
			theirs := listKeys(ref, l.Method)
			if theirs == nil {
				continue
			}
			mine := listKeys(c, l.Method)
			fmt.Fprintf(w, "| %s | %s | %d | %s |\n", c.Corpus, l.Method, len(mine), sameList(mine, theirs))
		}
	}
	fmt.Fprintln(w)
}

// fusedInputs reads the two inputs out of a fused list's name,
// "RRF(<a>, <b>)"; ok is false for any other list.
func fusedInputs(method string) (a, b string, ok bool) {
	inner, found := strings.CutPrefix(method, "RRF(")
	if !found || !strings.HasSuffix(inner, ")") {
		return "", "", false
	}
	return strings.Cut(strings.TrimSuffix(inner, ")"), ", ")
}

// writeFusion is the pre-registered primary comparison of the fusion study:
// for every fused list, M1@primaryK of the fused list minus each of its two
// inputs, with diffCI's independent bootstrap and the same decision rule.
func writeFusion(w io.Writer, corpora []compareOut) {
	fmt.Fprintf(w, "\n### Fusion: M1@%d, fused minus input\n\n", primaryK)
	type row struct{ fused, input string }
	var rows []row
	cells := map[row]map[string]cell{}
	for _, c := range corpora {
		top := map[string][]int{}
		for _, v := range viewsOf(c) {
			top[v.name] = v.slice(1, primaryK)
		}
		for _, l := range c.Lists {
			a, b, ok := fusedInputs(l.Method)
			if !ok {
				continue
			}
			for _, in := range []string{a, b} {
				r := row{l.Method, in}
				if cells[r] == nil {
					cells[r] = map[string]cell{}
					rows = append(rows, r)
				}
				d, lo, hi := diffCI(c, top[l.Method], top[in], "fusion "+l.Method+" - "+in)
				cells[r][c.Corpus] = cell{d, lo, hi}
			}
		}
	}
	fmt.Fprintf(w, "| fused | input |")
	for _, c := range corpora {
		fmt.Fprintf(w, " %s |", c.Corpus)
	}
	fmt.Fprintf(w, " fused wins | input wins | verdict |\n| --- | --- |")
	for range corpora {
		fmt.Fprintf(w, " --- |")
	}
	fmt.Fprintf(w, " ---: | ---: | --- |\n")
	for _, r := range rows {
		fmt.Fprintf(w, "| %s | %s |", r.fused, r.input)
		writeVerdictCells(w, corpora, cells[r], "**fused beats it**", "**it beats fused**")
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "✓ the fused list's CI lower bound is above 0 on that corpus; ✗ the upper bound is below 0. A verdict needs %d corpora one way and none the other.\n", beatsCorpora)
}

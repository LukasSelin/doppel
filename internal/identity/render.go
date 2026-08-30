package identity

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Print writes the text report: a header, then one section per class in
// classOrder, each headed by its count.
//
// Only classes with findings get a section — a report that lists six empty
// headings to say nothing happened is a report nobody reads to the end. The
// counts line at the top carries every class including the zeros, so the
// shape of the answer is still visible.
//
// unchanged is the one class rendered as a count and nothing else unless
// listUnchanged is set. It is the bulk of any comparison between two nearby
// states — eight hundred lines saying "still there" — and burying six real
// findings under it defeats the point. The JSON payload has no such cutoff:
// it carries every change, unchanged included, because a machine consumer
// filtering is trivial and a machine consumer missing data is not.
func Print(w io.Writer, r Result, listUnchanged bool) {
	if !r.Comparable {
		fmt.Fprintf(w, "Not comparable: %s\n", r.Reason)
		return
	}

	fmt.Fprintf(w, "%d functions before, %d after\n", r.OldFunctions, r.NewFunctions)
	fmt.Fprintf(w, "%s\n", countLine(r))
	for _, n := range r.Notes {
		fmt.Fprintf(w, "note: %s\n", n)
	}

	for _, class := range classOrder {
		n := r.Count(class)
		if n == 0 {
			continue
		}
		fmt.Fprintf(w, "\n%s %d\n", class, n)
		if class == Unchanged && !listUnchanged {
			continue
		}
		for _, c := range r.Changes {
			if c.Class != class {
				continue
			}
			for _, line := range Lines(c) {
				fmt.Fprintf(w, "  %s\n", line)
			}
		}
	}
}

// countLine is the one-line census, every class in classOrder including the
// zeros.
func countLine(r Result) string {
	parts := make([]string, 0, len(r.Counts))
	for _, cc := range r.Counts {
		parts = append(parts, fmt.Sprintf("%s %d", cc.Class, cc.Count))
	}
	return strings.Join(parts, ", ")
}

// Lines renders one Change as a headline plus an indented evidence line. It
// is exported so the command and the tests read the same rendering — a second
// spelling of this would be the drift this codebase refuses everywhere else.
//
// Every line states the evidence that produced the class: the weighted
// Jaccard and the containment for a one-to-one match, per-part containment
// for a split or a merge, and whether the two fingerprint digests agreed. A
// reader who doubts a line can open the two functions.
func Lines(c Change) []string {
	switch c.Class {
	case Split:
		return groupLines(c, c.Old[0], c.New, "->")
	case Merged:
		return groupLines(c, c.New[0], c.Old, "<-")
	case Added:
		return []string{location(c.New[0]) + "  (no counterpart above the match floor)"}
	case Deleted:
		return []string{location(c.Old[0]) + "  (no counterpart above the match floor)"}
	}

	head := location(c.Old[0])
	if c.Old[0].Key != c.New[0].Key {
		head += " -> " + location(c.New[0])
	}
	if extra := secondaryFacts(c); extra != "" {
		head += "  " + extra
	}
	return []string{head, "    " + evidence(c)}
}

// secondaryFacts names what the precedence order collapsed, so a moved
// function that was also renamed and also edited says all three on one line
// while carrying one class.
func secondaryFacts(c Change) string {
	var facts []string
	if c.Class != Renamed && c.NameChanged {
		facts = append(facts, "renamed "+c.Old[0].Name+" -> "+c.New[0].Name)
	}
	if c.Class != Moved && c.PackageChanged {
		facts = append(facts, "package "+c.Old[0].Package+" -> "+c.New[0].Package)
	}
	if c.Class != Edited && c.Class != Unchanged && !c.DigestEqual {
		facts = append(facts, "body edited")
	}
	if len(facts) == 0 {
		return ""
	}
	return "(" + strings.Join(facts, "; ") + ")"
}

func evidence(c Change) string {
	digest := "digests differ"
	if c.DigestEqual {
		digest = "digests equal"
	}
	return fmt.Sprintf("jaccard %.4f  containment %.4f  %s", c.Jaccard, c.Containment, digest)
}

// groupLines renders a split or a merge: the pivot, then one line per part
// with the containment that admitted it. arrow points from the pivot to the
// parts for a split and the other way for a merge, so the direction of the
// claim is legible without reading the class again.
func groupLines(c Change, pivot Member, parts []Member, arrow string) []string {
	out := []string{location(pivot) + "  " + arrow + " " + fmt.Sprint(len(parts)) + " bodies"}
	for _, p := range parts {
		out = append(out, fmt.Sprintf("    %s %s  containment %.4f", arrow, location(p), p.Containment))
	}
	return out
}

func location(m Member) string {
	return fmt.Sprintf("%s (%s:%d)", m.Key, m.File, m.Line)
}

// WriteJSON emits the machine shape: the same Result the text report renders,
// with every change including the unchanged ones. No maps, every slice
// already in a total order, so two runs over the same two files produce
// byte-identical bytes.
func WriteJSON(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteJSONDelta emits the classification and the pair changes together.
//
// Delta embeds Result, so this payload is the one WriteJSON produces with two
// fields appended: a consumer written against the classification alone keeps
// reading exactly the keys it always did. Both lists are already in a total
// order and neither contains a map, so two runs over the same two files produce
// byte-identical bytes.
func WriteJSONDelta(w io.Writer, d Delta) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(d)
}

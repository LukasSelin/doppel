package reporter

import (
	"fmt"
	"sort"
	"strings"
)

// Mermaid rendering helpers.
//
// The wiki's hand-authored diagrams need no escaping because every label there
// is prose someone typed. These labels are generated and carry Go identifiers:
// receivers keep their star, methods their dots, packages their slashes, and
// generic instantiations their brackets — `*ResultPool[T].WithMaxGoroutines` is
// a real name from the committed examples, and its brackets alone are enough to
// break an unquoted label.
//
// mdEscape must not be reused here. It replaces "|" with "\|", and "|" is
// mermaid's edge-label delimiter, so the escape would render as a literal
// backslash inside the node.

// mermaidLabel makes a string safe inside a double-quoted mermaid label.
//
// Mermaid wants HTML entities, not backslashes: there is no escape character to
// reach for, so a quote has to become #quot;. That makes "#" itself special,
// which is why this is one pass over the runes rather than chained replacement
// — replacing `"` first and `#` second would re-escape the entity just written.
func mermaidLabel(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '#':
			b.WriteString("#35;")
		case '"':
			b.WriteString("#quot;")
		case '<':
			b.WriteString("#lt;")
		case '>':
			b.WriteString("#gt;")
		case '\n', '\r':
			b.WriteString(" ")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// mermaidID makes a node identifier.
//
// Ids are not labels: mermaid parses them as bare tokens, so they may hold only
// letters, digits and underscores. Rather than mangling a name into an id and
// hoping the mangling stays unique — `a.b` and `a_b` collide the moment both
// exist — the caller supplies a prefix and an index, and the readable name goes
// in the label where it belongs.
func mermaidID(prefix string, i int) string {
	return fmt.Sprintf("%s%d", prefix, i)
}

// heatClass buckets a 0-1 goodness score into the three classes every generated
// diagram shares. Higher is better, so a low score is the hot one.
//
// The wiki's diagrams define no classes at all, and that is right for them: a
// hand-drawn diagram explains a mechanism, and colour would be decoration.
// These diagrams encode a measured value, and colour is the only channel
// mermaid gives for one.
func heatClass(v float64) string {
	switch {
	case v >= 0.75:
		return "good"
	case v >= 0.5:
		return "warn"
	default:
		return "hot"
	}
}

// mermaidClassDefs is the shared three-class palette, emitted by any diagram
// that colours its nodes. Fill colours are light enough to read black text on
// in a light theme and are left unset for stroke, so a dark-theme renderer
// still shows the shape.
const mermaidClassDefs = "    classDef good fill:#d7ecd9,color:#1b3d20\n" +
	"    classDef warn fill:#fbeecb,color:#4a3a12\n" +
	"    classDef hot fill:#f7d6d6,color:#4a1c1c\n"

// topByCount reduces a keyed count map to the largest n entries, returning them
// in descending count order with ties broken by key, plus how many were left
// out. Every diagram in this package is bounded, and every one of them says so.
func topByCount(counts map[string]int, n int) ([]string, int) {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	// Sorted before the count comparison so map order never survives into the
	// output: the tie-break is the key, and ties are common.
	sort.Strings(keys)
	sort.SliceStable(keys, func(i, j int) bool { return counts[keys[i]] > counts[keys[j]] })
	if n > 0 && len(keys) > n {
		return keys[:n], len(keys) - n
	}
	return keys, 0
}

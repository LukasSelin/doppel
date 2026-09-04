package reporter

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

// This file is the fingerprint view: one function's Fingerprint laid out so
// a reader can check it, and two functions' fingerprints merged so the
// code-shape number between them can be re-derived by hand.
//
// Every other surface shows a fingerprint only through a pair's score — a
// Breakdown line, a containment number, the top three shared labels. That is
// the right amount for a report and the wrong amount for verifying the
// fingerprint itself, which is the one stage every score in the tool rests
// on. This view prints the whole bag, weighted by the run's own corpus
// surprisal, and for a pair prints the partition — shared, only-A, only-B —
// whose masses *are* the Jaccard and the containment. The totals on the
// page add up to the numbers the pipeline reported, and a test pins that.
//
// Nothing here is recomputed under a second definition: the weights come
// from fingerprint.WeightOf, the overlap from fingerprint.WLOverlap, the
// rule words from analyzer.CanonWord, the label names from
// fingerprint.DescribeLabel. The view can only be as wrong as the score is.

// FingerprintMeta is the run context a fingerprint view states up front.
type FingerprintMeta struct {
	CorpusFuncs int // the population the label weights were counted over
	// LabelTop bounds how many bag rows print, highest mass first; 0 prints
	// every row. The single view's rows are the whole bag, so a substantial
	// body has hundreds, and a reader checking a score wants the heavy end.
	LabelTop int
}

// PrintFingerprint renders one function's fingerprint.
func PrintFingerprint(w io.Writer, u parser.CodeUnit, idf *fingerprint.LabelIDF, meta FingerprintMeta) {
	printFingerprintHead(w, "fingerprint", u)
	fp := u.Fingerprint
	if fp.Nodes == 0 {
		fmt.Fprintln(w, "  no body: the zero fingerprint, which never matches anything")
		return
	}
	fmt.Fprintf(w, "  nodes: %d (body as written)\n", fp.Nodes)
	fmt.Fprintf(w, "  canonicalized: %s\n", canonClause(u.CanonRules))
	fmt.Fprintf(w, "  flow: %s\n", flowLine(fp.Flow))
	fmt.Fprintf(w, "  nesting: %s\n", depthLine(fp.Depth))
	fmt.Fprintf(w, "  types: %s\n", typesLine(fp.Types))

	rows := bagRows(fp.WL, idf)
	var mass float64
	var total int
	for _, r := range rows {
		mass += r.mass
		total += r.count
	}
	fmt.Fprintf(w, "  labels: %d distinct, %d total, %.1f nats%s\n",
		len(rows), total, mass, weightingClause(idf, meta))
	printBagRows(w, rows, meta.LabelTop)
}

// PrintFingerprintPair renders two fingerprints and the merge between them.
//
// The Breakdown is the one fingerprint.Similarity computed, printed with the
// blend weight beside each component so the composite is a sum a reader can
// do. The label merge below it is the WL component opened up: the shared
// mass over the union mass is the Jaccard, the shared mass over the smaller
// side's mass is the containment, and both totals print next to the ratio
// they produce.
func PrintFingerprintPair(w io.Writer, a, b parser.CodeUnit, idf *fingerprint.LabelIDF, meta FingerprintMeta) {
	printFingerprintHead(w, "A", a)
	printFingerprintBrief(w, a)
	printFingerprintHead(w, "B", b)
	printFingerprintBrief(w, b)

	bd := fingerprint.Similarity(a.Fingerprint, b.Fingerprint, idf)
	if a.Fingerprint.Nodes == 0 || b.Fingerprint.Nodes == 0 {
		fmt.Fprintln(w, "\n  no body on one side: the zero fingerprint never matches anything")
		return
	}
	wt := fingerprint.DefaultWeights()
	fmt.Fprintf(w, "\n  code-shape: %.4f\n", bd.Score)
	fmt.Fprintf(w, "    wl       %.4f × %.2f = %.4f\n", bd.WL, wt.WL, bd.WL*wt.WL)
	fmt.Fprintf(w, "    flow     %.4f × %.2f = %.4f\n", bd.Flow, wt.Flow, bd.Flow*wt.Flow)
	fmt.Fprintf(w, "    nesting  %.4f × %.2f = %.4f\n", bd.Depth, wt.Depth, bd.Depth*wt.Depth)
	fmt.Fprintf(w, "    sig      %.4f × %.2f = %.4f\n", bd.Signature, wt.Signature, bd.Signature*wt.Signature)
	fmt.Fprintf(w, "  containment: %.4f%s\n", bd.Containment, containmentClause(bd))
	fmt.Fprintf(w, "  size: %.2f (%d / %d nodes; reported, not scored)\n",
		bd.SizeRatio, min(a.Fingerprint.Nodes, b.Fingerprint.Nodes), max(a.Fingerprint.Nodes, b.Fingerprint.Nodes))

	m := mergeBags(a.Fingerprint.WL, b.Fingerprint.WL, idf)
	union := m.massA + m.massB - m.shared
	fmt.Fprintf(w, "\n  label merge%s\n", weightingClause(idf, meta))
	fmt.Fprintf(w, "    mass A %.1f  mass B %.1f  shared %.1f  union %.1f\n", m.massA, m.massB, m.shared, union)
	fmt.Fprintf(w, "    jaccard = shared / union = %.4f\n", ratioOrZero(m.shared, union))
	fmt.Fprintf(w, "    containment = shared / min(A, B) = %.4f\n", ratioOrZero(m.shared, min(m.massA, m.massB)))

	printMergeSection(w, "shared", m.sharedRows, meta.LabelTop)
	printMergeSection(w, "only in A", m.onlyA, meta.LabelTop)
	printMergeSection(w, "only in B", m.onlyB, meta.LabelTop)
}

// printFingerprintHead is the one-line identity every section opens with:
// location, qualified name, signature.
func printFingerprintHead(w io.Writer, label string, u parser.CodeUnit) {
	name := u.Name
	if u.Package != "" {
		name = u.Package + "." + name
	}
	fmt.Fprintf(w, "%s: %s  %s:%d\n", label, name, filepath.ToSlash(u.File), u.StartLine)
	if u.Signature != "" {
		fmt.Fprintf(w, "  sig: %s\n", u.Signature)
	}
}

// printFingerprintBrief is the single view's body lines without the bag,
// for a pair page where the bag is shown merged rather than twice over.
func printFingerprintBrief(w io.Writer, u parser.CodeUnit) {
	fp := u.Fingerprint
	if fp.Nodes == 0 {
		fmt.Fprintln(w, "  no body")
		return
	}
	fmt.Fprintf(w, "  nodes: %d  labels: %d distinct  canonicalized: %s\n",
		fp.Nodes, len(fp.WL), canonClause(u.CanonRules))
	fmt.Fprintf(w, "  flow: %s\n", flowLine(fp.Flow))
	fmt.Fprintf(w, "  nesting: %s\n", depthLine(fp.Depth))
	fmt.Fprintf(w, "  types: %s\n", typesLine(fp.Types))
}

// canonClause names the rules that fired, in the order the frontend recorded
// them, under the same words the explain sentence uses.
func canonClause(rules []string) string {
	if len(rules) == 0 {
		return "no rules fired"
	}
	words := make([]string, len(rules))
	for i, r := range rules {
		words[i] = analyzer.CanonWord(r)
	}
	return strings.Join(words, ", ")
}

// flowLine renders the non-zero control-flow slots by name; "none" for a
// body with no control flow, which cosineInts scores as a match against any
// other such body.
func flowLine(flow []int) string {
	var parts []string
	for i, n := range flow {
		if n == 0 || i >= len(fingerprint.FlowLabels) {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d", fingerprint.FlowLabels[i], n))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "  ")
}

// depthLine renders the nesting-entry histogram as "depth N: count" for the
// non-empty buckets; the last bucket is the folded tail and says so.
func depthLine(depth []int) string {
	var parts []string
	for i, n := range depth {
		if n == 0 {
			continue
		}
		if i == len(depth)-1 {
			parts = append(parts, fmt.Sprintf("depth %d+: %d", i, n))
			continue
		}
		parts = append(parts, fmt.Sprintf("depth %d: %d", i, n))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "  ")
}

// typesLine is the normalized type set the signature component reads —
// the in:/out: strings themselves, since that is what Jaccard compares.
func typesLine(types []string) string {
	if len(types) == 0 {
		return "none (an empty set matches any other empty set at 1.0)"
	}
	return strings.Join(types, " ")
}

// weightingClause says which corpus the weights came from, or that there is
// none. A nil idf means every label weighs 1 and the numbers are the plain
// multiset arithmetic; the reader must not mistake that for a corpus figure.
func weightingClause(idf *fingerprint.LabelIDF, meta FingerprintMeta) string {
	if idf == nil || idf.N() == 0 {
		return " (no corpus: every label weighs 1.0)"
	}
	n := meta.CorpusFuncs
	if n == 0 {
		n = idf.N()
	}
	return fmt.Sprintf(" (weights ln(N/df), N = %d bodies)", n)
}

// bagRow is one label as the view prints it. mass is weight × count: the
// label's whole contribution to the side's mass in the overlap arithmetic.
type bagRow struct {
	label  uint64
	h      uint8
	kind   fingerprint.LabelKind
	count  int
	df     int
	weight float64
	mass   float64
}

// bagRows weights a bag and orders it heaviest first: mass desc, then round
// desc (a deeper label is the more specific claim at equal mass), then label
// asc — the same tie-break the retrieval chains use, so the two surfaces
// agree about which label leads.
func bagRows(bag []fingerprint.LabelCount, idf *fingerprint.LabelIDF) []bagRow {
	rows := make([]bagRow, 0, len(bag))
	for _, lc := range bag {
		w := fingerprint.WeightOf(idf, lc.Label)
		rows = append(rows, bagRow{
			label:  lc.Label,
			h:      lc.H,
			kind:   lc.Kind,
			count:  int(lc.Count),
			df:     idf.DF(lc.Label),
			weight: w,
			mass:   w * float64(lc.Count),
		})
	}
	sortBagRows(rows)
	return rows
}

func sortBagRows(rows []bagRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].mass != rows[j].mass {
			return rows[i].mass > rows[j].mass
		}
		if rows[i].h != rows[j].h {
			return rows[i].h > rows[j].h
		}
		return rows[i].label < rows[j].label
	})
}

// printBagRows prints up to top rows (0 = all) and counts the rest, so a
// truncated list never reads as the whole bag.
func printBagRows(w io.Writer, rows []bagRow, top int) {
	shown := rows
	if top > 0 && len(rows) > top {
		shown = rows[:top]
	}
	fmt.Fprintf(w, "    %8s  %6s  %5s  %4s  %s\n", "mass", "weight", "count", "df", "label")
	for _, r := range shown {
		fmt.Fprintf(w, "    %8.2f  %6.2f  %5d  %4s  %s  #%x\n",
			r.mass, r.weight, r.count, dfCell(r.df), fingerprint.DescribeLabel(r.h, r.kind), r.label)
	}
	if rest := len(rows) - len(shown); rest > 0 {
		fmt.Fprintf(w, "    … and %d more\n", rest)
	}
}

// dfCell renders a document frequency, or "-" when there is no corpus to
// have counted one.
func dfCell(df int) string {
	if df == 0 {
		return "-"
	}
	return fmt.Sprint(df)
}

// mergeRow is one label in a pair's merge: its count on each side and the
// mass it contributes to the shared, A-only or B-only partition.
type mergeRow struct {
	bagRow
	countA, countB int
}

// bagMerge is a pair's WL arithmetic laid out as rows. The three partitions
// together carry exactly Σ w·max over the union: sharedRows carry w·min,
// onlyA and onlyB carry the excess on each side, and a label present on
// both sides with unequal counts appears in sharedRows *and* in the side
// with the surplus — one row per partition, never one row saying two things.
type bagMerge struct {
	massA, massB, shared float64
	sharedRows           []mergeRow
	onlyA, onlyB         []mergeRow
}

// mergeBags is the single sorted merge fingerprint.wlOverlap does, kept as
// rows instead of three sums. The totals it accumulates are the same three
// quantities — TestFingerprintPairMergeReproducesTheScore pins that they
// reproduce WLOverlap's Jaccard and containment — and the rows are what a
// reader needs to see *which* labels the ratio is made of.
func mergeBags(a, b []fingerprint.LabelCount, idf *fingerprint.LabelIDF) bagMerge {
	var m bagMerge
	row := func(lc fingerprint.LabelCount, count int) bagRow {
		w := fingerprint.WeightOf(idf, lc.Label)
		return bagRow{label: lc.Label, h: lc.H, kind: lc.Kind, count: count,
			df: idf.DF(lc.Label), weight: w, mass: w * float64(count)}
	}
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case j == len(b) || (i < len(a) && a[i].Label < b[j].Label):
			r := row(a[i], int(a[i].Count))
			m.massA += r.mass
			m.onlyA = append(m.onlyA, mergeRow{bagRow: r, countA: r.count})
			i++
		case i == len(a) || b[j].Label < a[i].Label:
			r := row(b[j], int(b[j].Count))
			m.massB += r.mass
			m.onlyB = append(m.onlyB, mergeRow{bagRow: r, countB: r.count})
			j++
		default:
			ca, cb := int(a[i].Count), int(b[j].Count)
			w := fingerprint.WeightOf(idf, a[i].Label)
			m.massA += w * float64(ca)
			m.massB += w * float64(cb)
			s := row(a[i], min(ca, cb))
			m.shared += s.mass
			m.sharedRows = append(m.sharedRows, mergeRow{bagRow: s, countA: ca, countB: cb})
			switch {
			case ca > cb:
				m.onlyA = append(m.onlyA, mergeRow{bagRow: row(a[i], ca-cb), countA: ca, countB: cb})
			case cb > ca:
				m.onlyB = append(m.onlyB, mergeRow{bagRow: row(b[j], cb-ca), countA: ca, countB: cb})
			}
			i++
			j++
		}
	}
	sortMergeRows(m.sharedRows)
	sortMergeRows(m.onlyA)
	sortMergeRows(m.onlyB)
	return m
}

func sortMergeRows(rows []mergeRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].mass != rows[j].mass {
			return rows[i].mass > rows[j].mass
		}
		if rows[i].h != rows[j].h {
			return rows[i].h > rows[j].h
		}
		return rows[i].label < rows[j].label
	})
}

// printMergeSection prints one partition with its total, so the three
// section totals can be added back into the masses above them.
func printMergeSection(w io.Writer, title string, rows []mergeRow, top int) {
	var mass float64
	for _, r := range rows {
		mass += r.mass
	}
	fmt.Fprintf(w, "\n  %s: %d labels, %.1f nats\n", title, len(rows), mass)
	if len(rows) == 0 {
		return
	}
	shown := rows
	if top > 0 && len(rows) > top {
		shown = rows[:top]
	}
	fmt.Fprintf(w, "    %8s  %6s  %3s  %3s  %4s  %s\n", "mass", "weight", "A", "B", "df", "label")
	for _, r := range shown {
		fmt.Fprintf(w, "    %8.2f  %6.2f  %3d  %3d  %4s  %s  #%x\n",
			r.mass, r.weight, r.countA, r.countB, dfCell(r.df), fingerprint.DescribeLabel(r.h, r.kind), r.label)
	}
	if rest := len(rows) - len(shown); rest > 0 {
		fmt.Fprintf(w, "    … and %d more\n", rest)
	}
}

func ratioOrZero(num, den float64) float64 {
	if den <= 0 {
		return 0
	}
	return num / den
}

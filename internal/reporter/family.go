package reporter

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/family"
	"github.com/LukasSelin/doppel/internal/parser"
)

// maxFamilyMembers bounds how many members a *bounded* family listing prints.
// A large corpus has families of fifty, and five of those is a wall of names
// where the report wanted a summary. The census view (show == 0) lists
// everything, which is what makes it the census.
const maxFamilyMembers = 10

// PrintFamilies writes the family census as text, or nothing when the corpus
// has none.
//
// show bounds the list (0 = every family). Silence on an empty census is the
// same rule the impact digest follows: a "no families" line printed under
// every report trains the reader to skip the place real findings appear.
//
// One sentence per family carries the whole claim — "every pair >= 0.72" — and
// it is deliberately the *weakest* edge, not the mean. A family is a claim
// about all of its members, so the number a reader can check by opening any
// two files is the one worth printing.
func PrintFamilies(w io.Writer, fams []family.Family, stats family.Stats, units []parser.CodeUnit, show int) {
	if len(fams) == 0 {
		return
	}
	fmt.Fprintf(w, "\nFamilies\n")
	fmt.Fprintf(w, "--------\n")
	fmt.Fprintf(w, "%s\n\n", familySummary(fams, stats))

	shown := fams
	if show > 0 && len(shown) > show {
		shown = shown[:show]
	}
	for i, f := range shown {
		fmt.Fprintf(w, "F%-3d  %d members   every pair >= %.2f code-shape   evidence %.0f%s%s\n",
			i+1, len(f.Members), f.MinEdge, f.Evidence, completedNote(f.Completed), familyKindNote(f, false))
		listed := memberLimit(f, show)
		for _, m := range f.Members[:listed] {
			if m < 0 || m >= len(units) {
				continue
			}
			printFamilyMember(w, units[m])
		}
		if more := len(f.Members) - listed; more > 0 {
			fmt.Fprintf(w, "      (%d more members not listed)\n", more)
		}
		fmt.Fprintln(w)
	}
	if more := len(fams) - len(shown); more > 0 {
		fmt.Fprintf(w, "(%d more families not listed)\n\n", more)
	}
	printSkipped(w, "", stats)
}

// PrintMarkdownFamilies is the same census for the markdown report. It emits
// nothing for an empty census, so a report without families is byte-identical
// to one written before this section existed.
func PrintMarkdownFamilies(w io.Writer, fams []family.Family, stats family.Stats, units []parser.CodeUnit, show int) {
	if len(fams) == 0 {
		return
	}
	fmt.Fprintf(w, "## Families\n\n")
	fmt.Fprintf(w, "%s\n\n", familySummary(fams, stats))

	shown := fams
	if show > 0 && len(shown) > show {
		shown = shown[:show]
	}
	for i, f := range shown {
		fmt.Fprintf(w, "### Family %d — %d members, every pair `>= %.2f` code-shape, evidence `%.0f`%s%s\n\n",
			i+1, len(f.Members), f.MinEdge, f.Evidence, completedNote(f.Completed), familyKindNote(f, true))
		fmt.Fprintf(w, "| Location | Function | Signature | Patterns |\n")
		fmt.Fprintf(w, "|---|---|---|---|\n")
		listed := memberLimit(f, show)
		for _, m := range f.Members[:listed] {
			if m < 0 || m >= len(units) {
				continue
			}
			mdFamilyRow(w, units[m])
		}
		fmt.Fprintln(w)
		if more := len(f.Members) - listed; more > 0 {
			fmt.Fprintf(w, "_%d more members not listed._\n\n", more)
		}
	}
	if more := len(fams) - len(shown); more > 0 {
		fmt.Fprintf(w, "_%d more families not listed._\n\n", more)
	}
	printSkipped(w, "_", stats)
}

// memberLimit bounds a family's member list. show == 0 is the census: no
// cutoff anywhere, which is the whole difference between the two views.
func memberLimit(f family.Family, show int) int {
	if show == 0 || len(f.Members) <= maxFamilyMembers {
		return len(f.Members)
	}
	return maxFamilyMembers
}

// familySummary is the census line: the counts a reader wants before deciding
// whether to read the list.
//
// "functions in a family" counts distinct functions, because a function can
// belong to more than one maximal clique and summing member counts would
// report more duplication than the corpus has.
func familySummary(fams []family.Family, stats family.Stats) string {
	noun := "families"
	if len(fams) == 1 {
		noun = "family"
	}
	s := fmt.Sprintf("%d %s, %d functions in a family, largest %d members",
		len(fams), noun, stats.Members, len(fams[0].Members))
	if stats.Completed > 0 {
		s += fmt.Sprintf("; %d edges scored here that retrieval never proposed", stats.Completed)
	}
	return s
}

// completedNote names the edges this stage scored itself. Without it a reader
// checking a family against the pair list finds members with no pair between
// them and concludes the family was invented.
// familyKindNote is the F-line / heading suffix for a labeled family.
func familyKindNote(f family.Family, md bool) string {
	if f.Kind == nil {
		return ""
	}
	if md {
		return ", " + kindClause(f.Kind, true, true)
	}
	return "   kind: " + kindClause(f.Kind, true, false)
}

func completedNote(n int) string {
	switch {
	case n == 0:
		return ""
	case n == 1:
		return "  (1 edge scored here)"
	default:
		return fmt.Sprintf("  (%d edges scored here)", n)
	}
}

// printSkipped reports the components a guard abandoned. A guard that drops
// work silently reads as "there was nothing there".
func printSkipped(w io.Writer, wrap string, stats family.Stats) {
	if len(stats.Skipped) == 0 {
		return
	}
	sizes := make([]string, 0, len(stats.Skipped))
	for _, n := range stats.Skipped {
		sizes = append(sizes, fmt.Sprint(n))
	}
	fmt.Fprintf(w, "%s%d component(s) too large or too dense to enumerate (sizes %s); their families are not reported.%s\n\n",
		wrap, len(stats.Skipped), strings.Join(sizes, ", "), wrap)
}

// printFamilyMember is one line per member, deliberately narrower than
// printUnit's pair rendering. A family of seven is a list to scan, and the
// per-unit signature and tag lines that help when comparing exactly two
// functions turn a seven-member family into twenty-one lines of noise.
func printFamilyMember(w io.Writer, u parser.CodeUnit) {
	name := u.Name
	if u.Package != "" {
		name = u.Package + "." + name
	}
	fmt.Fprintf(w, "      %-46s  %s:%d\n", name, filepath.ToSlash(u.File), u.StartLine)
}

func mdFamilyRow(w io.Writer, u parser.CodeUnit) {
	loc := fmt.Sprintf("`%s:%d`", filepath.ToSlash(u.File), u.StartLine)
	name := u.Name
	if u.Package != "" {
		name = u.Package + "." + name
	}
	sig := u.Signature
	if sig == "" {
		sig = "—"
	}
	patterns := "—"
	if len(u.Patterns) > 0 {
		patterns = strings.Join(u.Patterns, ", ")
	}
	fmt.Fprintf(w, "| %s | `%s` | `%s` | %s |\n", loc, mdEscape(name), mdEscape(sig), patterns)
}

// FamiliesJSON is the machine-readable census: doppel families --format json.
//
// It is deliberately not part of snapshot.Snapshot. The snapshot's rule is
// that only what a consumer reads is stored, and the Stop hook rewrites that
// file every turn; families are a report, not a measurement origin.
type FamiliesJSON struct {
	Functions int          `json:"functions"`
	Families  []FamilyJSON `json:"families"`
	Members   int          `json:"members"` // distinct functions in >= 1 family
	Completed int          `json:"completed"`
	Skipped   []int        `json:"skipped,omitempty"`
}

// FamilyJSON is one family, keyed by name rather than by index: positions are
// file-walk artifacts and shift the moment a file is added.
type FamilyJSON struct {
	Size      int          `json:"size"`
	MinEdge   float64      `json:"minEdge"`
	MeanEdge  float64      `json:"meanEdge"`
	Evidence  float64      `json:"evidence"` // Σ retrieval evidence mass over retrieved edges; the census's rank key
	Completed int          `json:"completed"`
	Kind      string       `json:"kind,omitempty"`      // analyzer.KindInterfaceImpl or KindFork; absent when unlabeled
	KindLabel string       `json:"kindLabel,omitempty"` // the rendered clause, as the text report prints it
	Members   []MemberJSON `json:"members"`             // sorted by key
}

// MemberJSON locates one member for a human reading the JSON.
type MemberJSON struct {
	Key  string `json:"key"` // package.Name
	File string `json:"file"`
	Line int    `json:"line"`
}

// PrintFamiliesJSON writes the census as JSON.
func PrintFamiliesJSON(w io.Writer, fams []family.Family, stats family.Stats, units []parser.CodeUnit, root string) error {
	out := FamiliesJSON{
		Functions: len(units),
		Members:   stats.Members,
		Completed: stats.Completed,
		Skipped:   stats.Skipped,
		Families:  make([]FamilyJSON, 0, len(fams)),
	}
	for _, f := range fams {
		fj := FamilyJSON{
			Size:      len(f.Members),
			MinEdge:   f.MinEdge,
			MeanEdge:  f.MeanEdge,
			Evidence:  f.Evidence,
			Completed: f.Completed,
			Members:   make([]MemberJSON, 0, len(f.Members)),
		}
		if f.Kind != nil {
			fj.Kind = f.Kind.Kind
			fj.KindLabel = kindClause(f.Kind, true, false)
		}
		for _, m := range f.Members {
			if m < 0 || m >= len(units) {
				continue
			}
			u := units[m]
			name := u.Name
			if u.Package != "" {
				name = u.Package + "." + name
			}
			fj.Members = append(fj.Members, MemberJSON{
				Key:  name,
				File: relSlashPath(root, u.File),
				Line: u.StartLine,
			})
		}
		// By name, so the payload does not carry file-walk order.
		sort.Slice(fj.Members, func(i, j int) bool {
			if fj.Members[i].Key != fj.Members[j].Key {
				return fj.Members[i].Key < fj.Members[j].Key
			}
			return fj.Members[i].File < fj.Members[j].File
		})
		out.Families = append(out.Families, fj)
	}
	return encodeJSON(w, out)
}

// relSlashPath mirrors the snapshot's path rule: relative to the analysis
// root and slash-separated, so a census taken at an absolute cwd reads the
// same as one taken at ".".
func relSlashPath(root, path string) string {
	if root == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

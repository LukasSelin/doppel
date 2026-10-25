package reporter

import (
	"fmt"
	"io"
	"strings"
)

// Move statuses. A move is clean when it would compile as it stands, blocked
// when it would not, and anchored when its signature names a type of its own
// package that stays — a constructor or an entry point, which belongs beside
// its type however few packages call it. Anchored wins over blocked.
const (
	MoveClean    = "clean"
	MoveBlocked  = "blocked"
	MoveAnchored = "anchored"
)

// MoveRow is one function a moves analysis found, as plain data: the record
// `doppel moves --format json` prints, and the row the analyze report renders.
// Paths are relative to the analysis root and slash-separated; To is an import
// path and ToDir the target's directory under the root.
type MoveRow struct {
	Func      string    `json:"func"`
	File      string    `json:"file"`
	Line      int       `json:"line"`
	Status    string    `json:"status"` // clean, blocked or anchored
	To        string    `json:"to"`     // import path
	ToDir     string    `json:"toDir"`
	Share     float64   `json:"share"`
	Users     []string  `json:"users"`
	Staying   []string  `json:"staying,omitempty"`
	With      []MoveSym `json:"with,omitempty"`
	Anchors   []string  `json:"anchors,omitempty"`
	Blocked   []string  `json:"blocked,omitempty"`
	TestFiles []string  `json:"testFiles,omitempty"`
}

// Target names where the move goes for a reader: the directory under the
// root, or the import path when the target is the root package itself, whose
// directory reads as a bare ".".
func (r MoveRow) Target() string {
	if r.ToDir == "." {
		return r.To
	}
	return r.ToDir
}

// MoveSym is one helper that moves along with a function.
type MoveSym struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// Moves is a moves analysis as the analyze report carries it.
//
// It travels beside the pair findings rather than inside Meta.Overview because
// it is not drawn from the corpus model at all: cmd runs it through the type
// checker, after the pipeline, only when the tree is a Go module that loads.
// Nil renders nothing, which is what a run that skipped the stage passes.
type Moves struct {
	Target    string  // the GOOS/GOARCH the module was type-checked for
	Share     float64 // the referrer share a candidate needed
	Packages  int
	Functions int
	// Unclean counts the graph's packages that did not type-check cleanly:
	// a dependency missing from the module cache, or code that does not build
	// for Target. Their unresolved references are missing from the graph, so a
	// move can have a user nobody saw, and the section says so.
	Unclean int
	Rows    []MoveRow // every move found, in declaration order
}

// Counts splits Rows by status.
func (m *Moves) Counts() (clean, blocked, anchored int) {
	for _, r := range m.Rows {
		switch r.Status {
		case MoveClean:
			clean++
		case MoveBlocked:
			blocked++
		case MoveAnchored:
			anchored++
		}
	}
	return clean, blocked, anchored
}

// maxMoveRows bounds the markdown table. Clean moves are rare — four on
// doppel's own tree — so the bound exists for the corpus where they are not,
// and says what it left out.
const maxMoveRows = 25

// maxMoveUsers bounds the users cell: a function called from a dozen
// declarations of one package needs the package, not the dozen.
const maxMoveUsers = 4

// PrintMoves writes the one-line summary the text report carries. The list
// itself is `doppel moves`, which a terminal reader can run; a line is enough
// to say there is something to run it for.
func PrintMoves(w io.Writer, m *Moves) {
	if m == nil {
		return
	}
	clean, blocked, anchored := m.Counts()
	fmt.Fprintf(w, "\nFunctions out of place: %d used only by one other package could move there as %s (%d more blocked, %d anchored to a type that stays; doppel moves lists them)\n",
		clean, pick(clean, "it stands", "they stand"), blocked, anchored)
}

// PrintMarkdownMoves writes the "Functions out of place" section: the clean
// moves with where each would go, who uses it there, and what moves along.
// Blocked and anchored moves are counted and left to `doppel moves`, whose
// sentences say why each one is.
func PrintMarkdownMoves(w io.Writer, m *Moves) {
	if m == nil {
		return
	}
	clean, blocked, anchored := m.Counts()
	fmt.Fprintf(w, "\n## Functions out of place\n\n")
	fmt.Fprintf(w, "Functions whose every static reference comes from one other package, so that moving them there shrinks the package they live in. ")
	fmt.Fprintf(w, "Read through the type checker over %s and %s, built for %s",
		plural(m.Packages, "package"), plural(m.Functions, "function"), m.Target)
	if m.Share < 1 {
		fmt.Fprintf(w, ", at a referrer share of %.2f", m.Share)
	}
	fmt.Fprintf(w, ".")
	if m.Unclean > 0 {
		fmt.Fprintf(w, " **%d of the %s did not type-check cleanly** — a dependency missing from the module cache, or code that does not build for %s — so references the checker could not resolve are missing, and a move here can have a user it did not see.",
			m.Unclean, plural(m.Packages, "package"), m.Target)
	}
	fmt.Fprintf(w, "\n\n")

	if clean == 0 {
		fmt.Fprintf(w, "No function could move as it stands.")
	} else {
		fmt.Fprintf(w, "%s could move as %s, taking along the unexported helpers only %s use%s.",
			plural(clean, "function"), pick(clean, "it stands", "they stand"),
			pick(clean, "it", "they"), pick(clean, "s", ""))
	}
	var rest []string
	if blocked > 0 {
		rest = append(rest, fmt.Sprintf("%d %s blocked — the move would not compile as it stands", blocked, pick(blocked, "is", "are")))
	}
	if anchored > 0 {
		rest = append(rest, fmt.Sprintf("%d %s anchored to a type of %s own package that stays", anchored, pick(anchored, "is", "are"), pick(anchored, "its", "their")))
	}
	if len(rest) > 0 {
		fmt.Fprintf(w, " Of the other functions only one other package uses, %s; `doppel moves --blocked --anchored` lists them with the reasons.",
			strings.Join(rest, ", and "))
	}
	fmt.Fprintf(w, "\n")
	if clean == 0 {
		return
	}

	fmt.Fprintf(w, "\n| Function | Move to | Used by | Moves along |\n|---|---|---|---|\n")
	shown := 0
	for _, r := range m.Rows {
		if r.Status != MoveClean {
			continue
		}
		if shown == maxMoveRows {
			break
		}
		shown++
		users := r.Users
		more := ""
		if len(users) > maxMoveUsers {
			more = fmt.Sprintf(" and %d more", len(users)-maxMoveUsers)
			users = users[:maxMoveUsers]
		}
		with := make([]string, len(r.With))
		for i, x := range r.With {
			with[i] = "`" + mdEscape(x.Name) + "`"
		}
		along := strings.Join(with, ", ")
		if along == "" {
			along = "—"
		}
		fmt.Fprintf(w, "| `%s` %s:%d | `%s` | %s%s | %s |\n",
			mdEscape(r.Func), mdEscape(r.File), r.Line, mdEscape(r.Target()),
			codeList(users), more, along)
	}
	if clean > shown {
		fmt.Fprintf(w, "\n%d more clean moves are not shown; `doppel moves` lists every one.\n", clean-shown)
	}
}

func codeList(xs []string) string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = "`" + mdEscape(x) + "`"
	}
	return strings.Join(out, ", ")
}

func pick(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

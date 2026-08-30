package parser

import (
	"path/filepath"
	"strings"

	"github.com/LukasSelin/doppel/internal/fingerprint"
)

// CodeUnit represents a single extracted function or method.
// Concept is one learned concept membership: which concept, and how strongly
// this unit belongs to it.
//
// Membership used to be a bare tag string, asserted by a hand-written rule the
// moment any one of its channels matched. It is a confidence now because that
// is what the evidence actually is — internal/lexicon derives both the concept
// and the number from the corpus, so a unit carrying a concept's whole learned
// vocabulary and one carrying the least of it that still counts are no longer
// indistinguishable.
//
// Confidence is in (0,1] and saturating: 0.5 is a unit carrying the concept's
// typical evidence for this corpus, not a probability.
type Concept struct {
	ID         string
	Confidence float64
}

// ConceptIDs projects memberships back to bare IDs, ascending. It is the
// boolean view of a graded fact, and every caller of it is a place that must
// not see corpus-relative weights — the merge-signal gate above all.
func ConceptIDs(cs []Concept) []string {
	if len(cs) == 0 {
		return nil
	}
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.ID
	}
	return out
}

// ConfidenceOf returns the unit's confidence in one concept, or 0.
func ConfidenceOf(cs []Concept, id string) float64 {
	for _, c := range cs {
		if c.ID == id {
			return c.Confidence
		}
	}
	return 0
}

type CodeUnit struct {
	Name         string
	File         string
	StartLine    int
	Body         string
	Signature    string                  // parameter + return types, e.g. "(ctx context.Context) (User, error)"
	Package      string                  // Go package name
	Concepts     []Concept               // learned concept memberships with confidence, ascending by ID
	DocComment   string                  // godoc comment above the declaration
	Exported     bool                    // true if the function name is exported
	ReceiverType string                  // e.g. "*Server"; empty for plain functions
	Callees      []string                // AST-derived outgoing call names
	Fingerprint  fingerprint.Fingerprint // deterministic static summary of the body
	Signals      TagSignals              // AST-level evidence channels the tagger reads
	Generated    bool                    // the file carries Go's "Code generated ... DO NOT EDIT." marker
}

// MethodName returns the bare method name of a method unit ("Start" for
// "*Server.Start") and the plain name of a function. The one place to split a
// unit name at its receiver: the receiver can carry dots of its own in a
// generic instantiation, so "last dot" is not a safe rule anywhere else.
func MethodName(u CodeUnit) string {
	if u.ReceiverType == "" {
		return u.Name
	}
	return strings.TrimPrefix(u.Name, u.ReceiverType+".")
}

// Parse extracts all CodeUnits from the Go file at the given path.
// Non-.go files return nil, nil.
func Parse(path string) ([]CodeUnit, error) {
	if filepath.Ext(path) != ".go" {
		return nil, nil
	}
	return parseGo(path)
}

// ParseSource extracts CodeUnits from in-memory Go source. The path is used
// only for position information and the File field. This exists so tests —
// here and in the tagger — can parse inline snippets without touching disk.
func ParseSource(path string, src []byte) ([]CodeUnit, error) {
	return parseGoSource(path, src)
}

// ShouldSkipDir reports whether a directory is outside the population — what
// the go tool itself ignores, directories whose name starts with "." or "_"
// (which is what keeps _examples/ demo trees out of a library's population),
// plus vendor, testdata and build.
//
// It lives here, next to what it feeds, rather than in cmd: the bench harness
// walks the same tree and must apply the same rule, and it kept a byte-identical
// copy precisely to avoid depending on cmd. One definition, no such dependency.
func ShouldSkipDir(name string) bool {
	if name != "" && (name[0] == '.' || name[0] == '_') {
		return true
	}
	switch name {
	case "vendor", "testdata", "build":
		return true
	}
	return false
}

// Certain builds memberships asserted without reservation — confidence 1.0.
//
// It exists for the callers that legitimately have bare IDs and no grading:
// test fixtures pinning behavior that has nothing to do with confidence, and
// any consumer handed a plain list. Production tagging never uses it; the
// lexicon derives its confidences from the corpus.
func Certain(ids ...string) []Concept {
	if len(ids) == 0 {
		return nil
	}
	out := make([]Concept, len(ids))
	for i, id := range ids {
		out[i] = Concept{ID: id, Confidence: 1}
	}
	return out
}

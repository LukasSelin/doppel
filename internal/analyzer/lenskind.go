package analyzer

import (
	"regexp"
	"slices"
	"strings"

	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

// Three more pair kinds, each the production form of a signal bench's
// TestKindLenses measured against labelled corpora. Every one fired on no
// labelled merge or refactor across four self-review label sets, and every one
// failed as a ranking discount when held out on cobra — so they are what the
// two kinds above are: annotations that say why a pair exists, never a filter
// and never a rank.
//
//   - mirror operations — Encode beside Decode, Get beside Delete, MinOver
//     beside MaxOver, on one receiver. The bodies are alike because the
//     operations are one idea run in opposite directions, which is a reason
//     to keep them apart rather than to merge them.
//   - thin wrappers — two small bodies that already delegate to a shared
//     helper and differ in the names they pass it. The consolidation a merge
//     would buy has already happened; what is left is the call site.
//   - different calls — two bodies with a shared shape that call almost
//     nothing in common. The shape is a scaffold; the work is elsewhere.
const (
	KindMirror         = "mirror operations"
	KindThinWrappers   = "thin wrappers"
	KindDifferentCalls = "different calls"
)

// The thresholds the measurement set. ThinNodes bounds "small" on the
// larger side; ThinVocabCeiling is the vocab-lens Jaccard below which the two
// bodies name different things (merges read 1.00 there, wrappers about half);
// CallOverlapFloor is the callee Jaccard below which the calls differ.
const (
	ThinNodes        = 30
	ThinVocabCeiling = 0.8
	CallOverlapFloor = 0.25
)

// PairContext is what the lens kinds read beyond the two units: each side's
// resolved repo-internal callees (from the call graph, qualified names) and
// its vocab-lens bag (BuildThinVocab; nil for a body too large to be a thin
// wrapper, or one whose canonical tree is gone).
type PairContext struct {
	ResolvedA, ResolvedB []string
	VocabA, VocabB       []fingerprint.LabelCount
}

// ClassifyPairIn is ClassifyPairWith with the lens kinds after the naming
// kinds. Precedence, most specific first: a fork, then an interface
// implementation, then mirror operations, thin wrappers, different calls.
func ClassifyPairIn(a, b parser.CodeUnit, score, forkFloor float64, ctx PairContext) *KindNote {
	if k := ClassifyPairWith(a, b, score, forkFloor); k != nil {
		return k
	}
	if k := Mirror(a, b); k != nil {
		return k
	}
	if k := ThinWrappers(a, b, ctx); k != nil {
		return k
	}
	return DifferentCalls(a, b)
}

// opposites are word pairs that, with every other word of two names equal,
// make the names one operation and its opposite.
var opposites = map[string]string{}

func init() {
	for _, p := range [][2]string{
		{"encode", "decode"}, {"marshal", "unmarshal"}, {"read", "write"}, {"get", "set"},
		{"get", "delete"}, {"set", "delete"}, {"get", "put"}, {"put", "delete"}, {"add", "remove"},
		{"min", "max"}, {"super", "sub"}, {"old", "new"}, {"source", "sink"}, {"open", "close"},
		{"lock", "unlock"}, {"push", "pop"}, {"load", "save"}, {"load", "store"}, {"to", "from"},
		{"split", "merge"}, {"splits", "merges"}, {"union", "intersection"}, {"begin", "end"},
		{"start", "stop"}, {"enable", "disable"}, {"acquire", "release"}, {"lock", "release"},
		{"assemble", "split"}, {"compress", "decompress"}, {"serialize", "deserialize"},
		{"inc", "dec"}, {"up", "down"}, {"left", "right"}, {"first", "last"}, {"prev", "next"},
		{"before", "after"}, {"in", "out"}, {"import", "export"}, {"send", "receive"}, {"send", "recv"},
	} {
		opposites[p[0]+" "+p[1]] = p[1]
		opposites[p[1]+" "+p[0]] = p[0]
	}
}

var nameWordRE = regexp.MustCompile(`[A-Z]?[a-z]+|[A-Z]+(?:[a-z]+)?|\d+`)

func nameWords(name string) []string {
	var out []string
	for _, w := range nameWordRE.FindAllString(name, -1) {
		out = append(out, strings.ToLower(w))
	}
	return out
}

// oppositeNames reports whether two names differ in exactly one word and
// that word pair is an opposite.
func oppositeNames(a, b string) bool {
	wa, wb := nameWords(a), nameWords(b)
	if len(wa) != len(wb) {
		return false
	}
	diff := -1
	for i := range wa {
		if wa[i] != wb[i] {
			if diff >= 0 {
				return false
			}
			diff = i
		}
	}
	if diff < 0 {
		return false
	}
	_, ok := opposites[wa[diff]+" "+wb[diff]]
	return ok
}

// Mirror labels two methods on one receiver type in one package whose names
// are one operation and its opposite.
func Mirror(a, b parser.CodeUnit) *KindNote {
	if a.ReceiverType == "" || bareReceiver(a.ReceiverType) != bareReceiver(b.ReceiverType) {
		return nil
	}
	if relation(a, b) != RelationSamePackage {
		return nil
	}
	ma, mb := parser.MethodName(a), parser.MethodName(b)
	if !oppositeNames(ma, mb) {
		return nil
	}
	return &KindNote{
		Kind:      KindMirror,
		Names:     []string{ma, mb},
		Receivers: []string{a.ReceiverType},
		Packages:  []string{a.Package},
		Relation:  RelationSamePackage,
	}
}

// ThinWrappers labels two small bodies that call a shared repo-internal
// helper and name different things under the vocab lens.
func ThinWrappers(a, b parser.CodeUnit, ctx PairContext) *KindNote {
	na, nb := a.Fingerprint.Nodes, b.Fingerprint.Nodes
	if na == 0 || nb == 0 || max(na, nb) > ThinNodes {
		return nil
	}
	if len(ctx.VocabA) == 0 || len(ctx.VocabB) == 0 {
		return nil
	}
	var shared []string
	for _, c := range ctx.ResolvedA {
		if slices.Contains(ctx.ResolvedB, c) && !slices.Contains(shared, c) {
			shared = append(shared, c)
		}
	}
	if len(shared) == 0 {
		return nil
	}
	// Uniform weights: the rule asks whether the two small bodies name the
	// same things, which is a question about these two bags alone.
	if j, _ := fingerprint.WLOverlap(ctx.VocabA, ctx.VocabB, nil); j >= ThinVocabCeiling {
		return nil
	}
	slices.Sort(shared)
	return &KindNote{
		Kind:     KindThinWrappers,
		Shared:   shared,
		Packages: distinct(a.Package, b.Package),
		Relation: relation(a, b),
	}
}

// DifferentCalls labels two bodies whose distinct callees overlap below
// CallOverlapFloor. Two bodies that call nothing are not labelled: an empty
// overlap of empty sets says nothing about what the bodies do.
func DifferentCalls(a, b parser.CodeUnit) *KindNote {
	sa, sb := distinctSorted(a.Callees), distinctSorted(b.Callees)
	union := len(sa)
	shared := 0
	for _, c := range sb {
		if _, ok := slices.BinarySearch(sa, c); ok {
			shared++
		} else {
			union++
		}
	}
	if union == 0 {
		return nil
	}
	j := float64(shared) / float64(union)
	if j >= CallOverlapFloor {
		return nil
	}
	return &KindNote{
		Kind:     KindDifferentCalls,
		Overlap:  j,
		Packages: distinct(a.Package, b.Package),
		Relation: relation(a, b),
	}
}

func distinctSorted(xs []string) []string {
	out := slices.Clone(xs)
	slices.Sort(out)
	return slices.Compact(out)
}

// BuildThinVocab is each unit's vocab-lens bag — the canonical tree with
// names and literal values kept — for units small enough to be a thin
// wrapper, and nil for every other unit. It reads CodeUnit.Canonical, so it
// must run before a caller releases the trees.
func BuildThinVocab(units []parser.CodeUnit) [][]fingerprint.LabelCount {
	out := make([][]fingerprint.LabelCount, len(units))
	for i, u := range units {
		if u.Canonical == nil || u.Fingerprint.Nodes == 0 || u.Fingerprint.Nodes > ThinNodes {
			continue
		}
		out[i] = fingerprint.WLBagWith(u.Canonical, fingerprint.WLOptions{KeepNames: true})
	}
	return out
}

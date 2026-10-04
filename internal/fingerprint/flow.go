package fingerprint

import (
	"sort"

	"github.com/LukasSelin/doppel/internal/syntax"
)

// # The flow view
//
// Every structural quantity the score reads is order-free. The WL bag is a
// multiset of subtree labels, the control-flow and nesting components are
// histograms, and the signature component is a set. Two bodies that validate,
// fetch and map, against two that fetch, map and validate, share nearly every
// label and every histogram bucket, so code-shape cannot tell them apart.
//
// The flow view reads the same canonical tree as a *sequence*: one step per
// branch, loop, return, call, constructed literal and type assertion, in the
// order the frontend's traversal visits them. Two sequences are then aligned
// (FlowSimilarity), and the alignment says how much of the two functions'
// logic happens in the same order against the same targets. A second number,
// Types, is the Jaccard over the types each function touches: its signature
// plus every type it constructs or asserts.
//
// Like containment and the concept views it is reported and never scored: it
// enters no Score, no ranking key, no filter and no verdict. It earns a place
// in one only through a measurement against the labels, which is the rule
// every other view in this package has been held to — and bench TestFlowRank
// measured it and found nothing to adopt: ordered flow separates merges from
// false positives about as well as code-shape and no better, because the
// false positives that rank high (mirrors, accessor families) have the same
// logic by construction. What it adds is legibility: the alignment shows
// where two bodies differ, which a score cannot.
//
// It is coarser than a control-flow graph on purpose. A pre-order sequence
// does not mark where a block ends, so `if c { a() }; b()` and
// `if c { a(); b() }` read alike — nesting is what the depth histogram is for,
// and the alignment's job is order and target. Calls are named by callee
// only, the receiver dropped, for the same reason the WL labels drop it.

// FlowOp is the kind of one flow step.
type FlowOp uint8

const (
	FlowNone FlowOp = iota
	FlowIf
	FlowLoop // for and range alike: the loop is the step, its header is not
	FlowSwitch
	FlowTypeSwitch
	FlowSelect
	FlowReturn
	FlowDefer
	FlowGo
	FlowBranch // break, continue, goto, fallthrough — Name says which
	FlowFunc   // a function literal begins
	FlowSend
	FlowRecv
	FlowCall   // Name is the callee, receiver dropped; "" when unnamed
	FlowNew    // a composite literal; Name is its type
	FlowAssert // a type assertion; Name is the asserted type
)

var flowOpNames = [...]string{
	FlowNone: "?", FlowIf: "if", FlowLoop: "loop", FlowSwitch: "switch",
	FlowTypeSwitch: "typeswitch", FlowSelect: "select", FlowReturn: "return",
	FlowDefer: "defer", FlowGo: "go", FlowBranch: "branch", FlowFunc: "func",
	FlowSend: "send", FlowRecv: "recv", FlowCall: "call", FlowNew: "new",
	FlowAssert: "assert",
}

func (o FlowOp) String() string {
	if int(o) < len(flowOpNames) {
		return flowOpNames[o]
	}
	return "?"
}

// FlowStep is one step of a function's logic: what happens, and to what.
type FlowStep struct {
	Op   FlowOp
	Name string
}

// String renders a step as a reader sees it: "if", "call Get", "new Config".
func (s FlowStep) String() string {
	if s.Name == "" {
		return s.Op.String()
	}
	return s.Op.String() + " " + s.Name
}

// MaxFlowSteps bounds the sequence one function contributes to an alignment.
// Alignment is quadratic in sequence length, and a function longer than this
// is not one a reader compares step by step anyway; a pair that hit the bound
// says so (FlowScore.Truncated) rather than being scored as if it had not.
const MaxFlowSteps = 512

// flowSteps reads a tree into its step sequence.
//
// Two orders, because a tree's traversal order is not its execution order for
// everything. A construct that opens a region — if, loop, switch, select, a
// function literal, and defer and go, whose call runs elsewhere — is a step
// at the point it is entered, so it comes before what it governs. Everything
// that *does* something — a call, a construction, an assertion, a send, a
// receive, a return, a branch — is a step once its operands have been
// evaluated, so it comes after them: cmd.Flags().StringVar(…) is Flags then
// StringVar, and return fmt.Errorf(…) is the call then the return.
func flowSteps(root *syntax.Node) []FlowStep {
	var out []FlowStep
	var pending []FlowStep // per open node, the step it emits on exit; Op 0 for none
	syntax.Inspect(root, func(n *syntax.Node) bool {
		if n == nil {
			last := len(pending) - 1
			if s := pending[last]; s.Op != FlowNone {
				out = append(out, s)
			}
			pending = pending[:last]
			return false
		}
		var exit FlowStep
		switch n.Kind {
		case syntax.KindIf:
			out = append(out, FlowStep{Op: FlowIf})
		case syntax.KindFor, syntax.KindRange:
			out = append(out, FlowStep{Op: FlowLoop})
		case syntax.KindSwitch:
			out = append(out, FlowStep{Op: FlowSwitch})
		case syntax.KindTypeSwitch:
			out = append(out, FlowStep{Op: FlowTypeSwitch})
		case syntax.KindSelect:
			out = append(out, FlowStep{Op: FlowSelect})
		case syntax.KindDefer:
			out = append(out, FlowStep{Op: FlowDefer})
		case syntax.KindGo:
			out = append(out, FlowStep{Op: FlowGo})
		case syntax.KindFuncLit:
			out = append(out, FlowStep{Op: FlowFunc})
		case syntax.KindReturn:
			exit = FlowStep{Op: FlowReturn}
		case syntax.KindBranch:
			exit = FlowStep{Op: FlowBranch, Name: n.Label}
		case syntax.KindSend:
			exit = FlowStep{Op: FlowSend}
		case syntax.KindUnary:
			if n.Label == "<-" {
				exit = FlowStep{Op: FlowRecv}
			}
		case syntax.KindCall:
			exit = FlowStep{Op: FlowCall, Name: calleeName(n)}
		case syntax.KindComposite:
			exit = FlowStep{Op: FlowNew, Name: typeName(n.Slot(syntax.RoleType))}
		case syntax.KindAssert:
			// x.(type) in a type switch has no type child and is not a step:
			// the typeswitch above it already is.
			if t := n.Slot(syntax.RoleType); t != nil {
				exit = FlowStep{Op: FlowAssert, Name: typeName(t)}
			}
		}
		pending = append(pending, exit)
		return true
	})
	return out
}

// typeName names a type expression as briefly as it can be named honestly:
// the type's own name with any package qualifier dropped (the selector's
// selected name, like a callee), a pointer or slice marker kept, and the kind
// of an anonymous type otherwise. "" when there is no type expression at all —
// an elided element type in a nested literal.
func typeName(t *syntax.Node) string {
	if t == nil {
		return ""
	}
	switch t.Kind {
	case syntax.KindIdent, syntax.KindSelector:
		return t.Label
	case syntax.KindStar:
		if x := t.Slot(syntax.RoleX); x != nil {
			return "*" + typeName(x)
		}
	case syntax.KindIndex, syntax.KindIndexList:
		// A generic instantiation: List[int] is a List.
		if x := t.Slot(syntax.RoleX); x != nil {
			return typeName(x)
		}
	case syntax.KindArrayType:
		// The element type is the last child; a fixed array's length, when
		// there is one, comes before it.
		if len(t.Kids) > 0 {
			return "[]" + typeName(t.Kids[len(t.Kids)-1].Node)
		}
		return "[]"
	case syntax.KindMapType:
		return "map"
	case syntax.KindStructType:
		return "struct"
	case syntax.KindFuncType:
		return "func"
	case syntax.KindInterfaceType:
		return "interface"
	case syntax.KindChanType:
		return "chan"
	}
	return ""
}

// touchedTypes is the set a function's Types score compares: its signature
// (the in:/out: entries Fingerprint.Types already carries) plus every named
// type its body constructs (new:) or asserts (as:).
func touchedTypes(sig []string, steps []FlowStep) []string {
	seen := make(map[string]struct{}, len(sig)+len(steps))
	for _, s := range sig {
		seen[s] = struct{}{}
	}
	for _, s := range steps {
		if s.Name == "" {
			continue
		}
		switch s.Op {
		case FlowNew:
			seen["new:"+s.Name] = struct{}{}
		case FlowAssert:
			seen["as:"+s.Name] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// FlowScore is the flow view of one pair.
//
// Steps is 2·S / (|A| + |B|), where S is the weight of the best order-
// preserving alignment of the two step sequences: a step matched to an
// identical step weighs 1, a step matched to one of the same kind against a
// different target (call Get against call Delete, new Config against new
// Options) weighs ½, and nothing else may be matched. Identical sequences read
// 1.0; a pair sharing steps only in a different order reads low however much
// its multisets agree. Two bodies with no steps at all read 1.0, the same
// both-empty convention the flow histogram's cosine uses.
//
// Types is the Jaccard over the two touched-type sets — see touchedTypes —
// with the signature component's both-empty convention.
type FlowScore struct {
	Steps      float64
	Types      float64
	Same       int // steps aligned to an identical step
	Retargeted int // steps aligned to a same-kind step against a different target
	LenA, LenB int // sequence lengths after the MaxFlowSteps bound
	Truncated  bool
}

// FlowSimilarity aligns the two fingerprints' step sequences and compares
// their touched types. Symmetric, deterministic, and linear in memory.
func FlowSimilarity(a, b Fingerprint) FlowScore {
	sa, sb, trunc := bounded(a.Steps), bounded(b.Steps), len(a.Steps) > MaxFlowSteps || len(b.Steps) > MaxFlowSteps
	fs := FlowScore{LenA: len(sa), LenB: len(sb), Truncated: trunc, Types: jaccardStrings(a.Touched, b.Touched)}
	if len(sa)+len(sb) == 0 {
		fs.Steps = 1
		return fs
	}
	// Two rolling rows of (score, same, retargeted), scores in half units so
	// the arithmetic is integer and exact. Equal-score alignments are broken
	// toward more identical steps (two retargets and one exact match weigh the
	// same, and the exact match is the stronger claim), which makes the counts
	// a property of the pair rather than of which side came first.
	prev := make([]flowCell, len(sb)+1)
	cur := make([]flowCell, len(sb)+1)
	for i := 1; i <= len(sa); i++ {
		cur[0] = flowCell{}
		for j := 1; j <= len(sb); j++ {
			cur[j] = flowBest(prev[j], cur[j-1], prev[j-1], stepMatch(sa[i-1], sb[j-1]))
		}
		prev, cur = cur, prev
	}
	end := prev[len(sb)]
	fs.Steps = float64(end.s) / float64(len(sa)+len(sb))
	fs.Same, fs.Retargeted = int(end.same), int(end.re)
	return fs
}

// flowCell is one alignment prefix: its score in half units and how many of
// its matches were exact and retargeted.
type flowCell struct{ s, same, re int32 }

// better orders two prefixes: higher score, then more exact matches.
func (c flowCell) better(d flowCell) bool {
	return c.s > d.s || (c.s == d.s && c.same > d.same)
}

// flowBest is one alignment recurrence step: skip a step of A (up), skip a
// step of B (left), or match the two (diag) when they can be matched. Ties go
// to the earlier option, and FlowAlign's traceback relies on that order.
func flowBest(up, left, diag flowCell, match int) flowCell {
	best := up
	if left.better(best) {
		best = left
	}
	if match > 0 {
		d := diag
		d.s += int32(match)
		if match == 2 {
			d.same++
		} else {
			d.re++
		}
		if d.better(best) {
			best = d
		}
	}
	return best
}

// stepMatch weighs a pairing in half units: 2 identical, 1 same kind against
// a different target, 0 not alignable.
func stepMatch(a, b FlowStep) int {
	if a.Op != b.Op {
		return 0
	}
	if a.Name == b.Name {
		return 2
	}
	return 1
}

func bounded(s []FlowStep) []FlowStep {
	if len(s) > MaxFlowSteps {
		return s[:MaxFlowSteps]
	}
	return s
}

// FlowRow is one row of a rendered alignment: the step from each side, or
// none, and how the two were matched.
type FlowRow struct {
	A, B  int // index into each side's Steps; -1 when the row has no step from that side
	Match int // 2 identical, 1 retargeted, 0 unmatched
}

// FlowAlign is FlowSimilarity's alignment written out row by row, for a
// surface that shows it. It keeps the whole score matrix to trace back
// through, so it is quadratic in memory where FlowSimilarity is linear —
// callers render it for a bounded number of pairs, never for every pair.
// Its matched rows always agree with FlowSimilarity's counts: both maximise
// the same objective under the same tie order.
func FlowAlign(a, b Fingerprint) []FlowRow {
	sa, sb := bounded(a.Steps), bounded(b.Steps)
	n, m := len(sa), len(sb)
	w := m + 1
	cells := make([]flowCell, (n+1)*w)
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cells[i*w+j] = flowBest(cells[(i-1)*w+j], cells[i*w+j-1], cells[(i-1)*w+j-1], stepMatch(sa[i-1], sb[j-1]))
		}
	}
	// Trace back by re-asking which option flowBest chose, so the rendered
	// rows are exactly the alignment the counts describe.
	var rows []FlowRow
	i, j := n, m
	for i > 0 || j > 0 {
		switch {
		case i == 0:
			rows = append(rows, FlowRow{A: -1, B: j - 1})
			j--
		case j == 0:
			rows = append(rows, FlowRow{A: i - 1, B: -1})
			i--
		default:
			here, up, left := cells[i*w+j], cells[(i-1)*w+j], cells[i*w+j-1]
			mt := stepMatch(sa[i-1], sb[j-1])
			switch {
			case here == up:
				rows = append(rows, FlowRow{A: i - 1, B: -1})
				i--
			case here == left:
				rows = append(rows, FlowRow{A: -1, B: j - 1})
				j--
			default:
				rows = append(rows, FlowRow{A: i - 1, B: j - 1, Match: mt})
				i, j = i-1, j-1
			}
		}
	}
	for l, r := 0, len(rows)-1; l < r; l, r = l+1, r-1 {
		rows[l], rows[r] = rows[r], rows[l]
	}
	return rows
}

// Package fingerprint builds deterministic static summaries of function
// bodies and scores how similar two of them are. No network, no model, no
// cache: the same source always yields the same fingerprint and the same score.
//
// It works on internal/syntax, not on any language's own AST, so a frontend
// that can produce a syntax.Func gets a Fingerprint — and with it the
// similarity score, the retrieval shape channel and all five pattern levels —
// without this package learning anything about the language.
package fingerprint

import (
	"hash/fnv"
	"math"
	"sort"

	"github.com/LukasSelin/doppel/internal/syntax"
)

// Weights for each component of the composite similarity Score.
// They sum to exactly 1.00. Depth's 0.05 was carved entirely out of Flow
// (0.25 → 0.20): nesting is flow-adjacent information, so flow pays for it
// rather than taxing the AST or signature components.
const (
	weightAST       = 0.60
	weightFlow      = 0.20
	weightDepth     = 0.05
	weightSignature = 0.15
)

// shingleK is the sliding-window width over the AST token stream.
const shingleK = 3

// depthBuckets is the length of the nesting-depth histogram: control-flow
// entry depths 0-4 each get a bucket and everything deeper folds into the
// last. Without this histogram two functions with identical token bags but
// different nesting — a flat if/if against an if nested in an if — were
// indistinguishable: flattened tokens carry no depth and the flow histogram
// only counts kinds.
const depthBuckets = 6

// Control-flow histogram slots. Order is fixed; flowKinds must stay last.
const (
	flowIf = iota
	flowFor
	flowRange
	flowSwitch
	flowTypeSwitch
	flowSelect
	flowReturn
	flowDefer
	flowGo
	flowFuncLit
	flowKinds
)

// FlowLabels names the control-flow histogram slots, index-aligned with
// Fingerprint.Flow. The array length is tied to flowKinds so a new slot
// cannot be added without naming it.
var FlowLabels = [flowKinds]string{
	"if", "for", "range", "switch", "typeswitch",
	"select", "return", "defer", "go", "funclit",
}

// Fingerprint is a deterministic static summary of one function body.
// The zero value means "no body" and never matches anything.
type Fingerprint struct {
	Shingles []uint64  // sorted, deduped FNV-1a hashes of AST 3-grams
	Flow     []int     // control-flow node histogram, length flowKinds
	Depth    []int     // control-flow entry-depth histogram, length depthBuckets
	Types    []string  // sorted, deduped normalized param + result types
	Nodes    int       // syntax node count of the body (size / triviality guard)
	Patterns []Pattern // multi-level structural pattern multiset, sorted by hash
}

// Breakdown is the per-component result of comparing two Fingerprints.
type Breakdown struct {
	AST       float64 // Jaccard over AST 3-gram shingles
	Flow      float64 // cosine over the control-flow histogram
	Depth     float64 // cosine over the nesting-depth histogram (reported as nesting:)
	Signature float64 // Jaccard over normalized parameter and result types
	SizeRatio float64 // min(Nodes)/max(Nodes); reported, not scored
	Score     float64 // weighted composite, 0.0-1.0
}

// Build summarises a function. A nil function or one without a body
// (external, forward-declared, or unsegmentable by its frontend) yields the
// zero Fingerprint.
func Build(fn *syntax.Func) Fingerprint {
	if fn == nil || fn.Body == nil {
		return Fingerprint{}
	}
	tokens, flow, depth, nodes := walk(fn.Body)
	return Fingerprint{
		Shingles: shingle(tokens),
		Flow:     flow,
		Depth:    depth,
		Types:    typeStrings(fn),
		Nodes:    nodes,
		Patterns: extractPatterns(fn, tokens),
	}
}

// Weights blends the four Breakdown components into Score. The production
// path always uses DefaultWeights; the type exists so the bench harness can
// measure the blend (a sensitivity sweep) without a package-level mutable —
// the same seam idiom as ontology.WithWeights.
type Weights struct {
	AST, Flow, Depth, Signature float64
}

// DefaultWeights returns the shipped blend: 0.60 / 0.20 / 0.05 / 0.15.
func DefaultWeights() Weights {
	return Weights{AST: weightAST, Flow: weightFlow, Depth: weightDepth, Signature: weightSignature}
}

// Sum is the blend total, 1.00 for the defaults.
func (w Weights) Sum() float64 { return w.AST + w.Flow + w.Depth + w.Signature }

// Scaled multiplies component i (0 AST, 1 Flow, 2 Depth, 3 Signature) by f
// and renormalizes the other three uniformly so the sum stays what it was —
// the ontology.WithWeights idiom, so a sweep moves one knob at a time.
func (w Weights) Scaled(i int, f float64) Weights {
	c := [4]float64{w.AST, w.Flow, w.Depth, w.Signature}
	total := w.Sum()
	c[i] *= f
	rest := total - c[i]
	var restNow float64
	for j := range c {
		if j != i {
			restNow += c[j]
		}
	}
	if restNow > 0 && rest > 0 {
		for j := range c {
			if j != i {
				c[j] *= rest / restNow
			}
		}
	}
	return Weights{AST: c[0], Flow: c[1], Depth: c[2], Signature: c[3]}
}

// Similarity scores two Fingerprints. It is symmetric, and returns the zero
// Breakdown when either side has no body.
func Similarity(a, b Fingerprint) Breakdown {
	return SimilarityWith(a, b, DefaultWeights())
}

// SimilarityWith is Similarity under an explicit blend. With DefaultWeights
// it is bit-identical to Similarity: the same four-term sum in the same
// order, which is what keeps every pinned score and every baseline digest
// where it is.
func SimilarityWith(a, b Fingerprint, w Weights) Breakdown {
	if a.Nodes == 0 || b.Nodes == 0 {
		return Breakdown{}
	}
	bd := Breakdown{
		AST:       jaccardUint64(a.Shingles, b.Shingles),
		Flow:      cosineInts(a.Flow, b.Flow),
		Depth:     cosineInts(a.Depth, b.Depth),
		Signature: jaccardStrings(a.Types, b.Types),
		SizeRatio: ratio(a.Nodes, b.Nodes),
	}
	bd.Score = w.AST*bd.AST + w.Flow*bd.Flow + w.Depth*bd.Depth + w.Signature*bd.Signature
	if bd.Score > 1.0 {
		bd.Score = 1.0
	}
	return bd
}

// walk traverses the body in pre-order, emitting one short token per
// interesting node. Identifiers collapse to "ID" and literals to their kind, so
// a copy with every variable renamed still produces the same token stream.
// Call targets keep their selector name because it carries real intent
// (Errorf, Query, Lock).
//
// Alongside the tokens it histograms control-flow nesting: every flow-slot
// node records the depth it was entered at (bucketed, deep tails folded into
// the last bucket), and the seven statement-bearing constructs — if, for,
// range, switch, type switch, select, funclit — push a nesting level for
// their children. The bool stack pairs each push with the f(nil) call
// syntax.Inspect makes after a node's children, which is the only reliable
// after-children hook Inspect offers.
func walk(body *syntax.Node) (tokens []string, flow, depth []int, nodes int) {
	flow = make([]int, flowKinds)
	depth = make([]int, depthBuckets)
	nesting := 0
	var nests []bool
	enter := func() {
		b := nesting
		if b > depthBuckets-1 {
			b = depthBuckets - 1
		}
		depth[b]++
	}
	syntax.Inspect(body, func(n *syntax.Node) bool {
		if n == nil {
			if last := len(nests) - 1; last >= 0 {
				if nests[last] {
					nesting--
				}
				nests = nests[:last]
			}
			return false
		}
		nodes++
		opens := false
		switch n.Kind {
		case syntax.KindIf:
			flow[flowIf]++
			enter()
			opens = true
			tokens = append(tokens, "IF")
		case syntax.KindFor:
			flow[flowFor]++
			enter()
			opens = true
			tokens = append(tokens, "FOR")
		case syntax.KindRange:
			flow[flowRange]++
			enter()
			opens = true
			tokens = append(tokens, "RANGE")
		case syntax.KindSwitch:
			flow[flowSwitch]++
			enter()
			opens = true
			tokens = append(tokens, "SWITCH")
		case syntax.KindTypeSwitch:
			flow[flowTypeSwitch]++
			enter()
			opens = true
			tokens = append(tokens, "TYPESWITCH")
		case syntax.KindSelect:
			flow[flowSelect]++
			enter()
			opens = true
			tokens = append(tokens, "SELECT")
		case syntax.KindReturn:
			flow[flowReturn]++
			enter()
			tokens = append(tokens, "RETURN")
		case syntax.KindDefer:
			flow[flowDefer]++
			enter()
			tokens = append(tokens, "DEFER")
		case syntax.KindGo:
			flow[flowGo]++
			enter()
			tokens = append(tokens, "GO")
		case syntax.KindFuncLit:
			flow[flowFuncLit]++
			enter()
			opens = true
			tokens = append(tokens, "FUNCLIT")
		case syntax.KindCall:
			tokens = append(tokens, "CALL")
			if name := calleeName(n); name != "" {
				tokens = append(tokens, "CALL:"+name)
			}
		case syntax.KindBinary:
			tokens = append(tokens, "BIN:"+n.Label)
		case syntax.KindUnary:
			tokens = append(tokens, "UNARY:"+n.Label)
		case syntax.KindAssign:
			tokens = append(tokens, "ASSIGN:"+n.Label)
		case syntax.KindBranch:
			tokens = append(tokens, "BRANCH:"+n.Label)
		case syntax.KindLit:
			tokens = append(tokens, "LIT:"+n.Label)
		case syntax.KindIncDec:
			tokens = append(tokens, "INCDEC")
		case syntax.KindIndex:
			tokens = append(tokens, "INDEX")
		case syntax.KindSlice:
			tokens = append(tokens, "SLICE")
		case syntax.KindStar:
			tokens = append(tokens, "STAR")
		case syntax.KindAssert:
			tokens = append(tokens, "ASSERT")
		case syntax.KindComposite:
			tokens = append(tokens, "COMPOSITE")
		case syntax.KindKeyValue:
			tokens = append(tokens, "KV")
		case syntax.KindSelector:
			tokens = append(tokens, "SEL")
		case syntax.KindIdent:
			tokens = append(tokens, "ID")
		}
		if opens {
			nesting++
		}
		nests = append(nests, opens)
		return true
	})
	return tokens, flow, depth, nodes
}

// calleeName names the thing a call invokes, dropping the receiver: the
// variable a method is called on is arbitrary (e, s, cfg) while the method
// name is not. A frontend records the selected name in the selector node's
// Label, so both call shapes read the same field.
//
// This deliberately differs from the parser's own callee extraction, which
// keeps the receiver because the call graph resolves on it.
func calleeName(call *syntax.Node) string {
	fun := call.Slot(syntax.RoleFun)
	if fun == nil {
		return ""
	}
	switch fun.Kind {
	case syntax.KindIdent, syntax.KindSelector:
		return fun.Label
	}
	return ""
}

// shingle hashes every sliding window of shingleK tokens, then sorts and
// deduplicates. Token streams shorter than the window hash as a single shingle
// so that very small bodies still compare meaningfully.
func shingle(tokens []string) []uint64 {
	if len(tokens) == 0 {
		return nil
	}
	k := shingleK
	if len(tokens) < k {
		k = len(tokens)
	}
	seen := make(map[uint64]struct{}, len(tokens))
	for i := 0; i+k <= len(tokens); i++ {
		seen[hashWindow(tokens[i:i+k])] = struct{}{}
	}
	out := make([]uint64, 0, len(seen))
	for h := range seen {
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func hashWindow(window []string) uint64 {
	h := fnv.New64a()
	for _, tok := range window {
		_, _ = h.Write([]byte(tok))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}

// typeStrings renders the parameter and result types of a function,
// discarding the parameter names. Inputs and outputs are prefixed so that a
// parameter of type error does not match a returned error.
//
// The result is a set, which is why it can read one entry per declared name
// while the signature line reads one per declared name too: "a, b int"
// contributes "in:int" once either way. Frontends therefore need only one
// convention for Params, and the two consumers stay consistent.
func typeStrings(fn *syntax.Func) []string {
	if fn == nil {
		return nil
	}
	seen := make(map[string]struct{})
	collect := func(prefix string, params []syntax.Param) {
		for _, p := range params {
			if p.Type == "" {
				continue
			}
			seen[prefix+p.Type] = struct{}{}
		}
	}
	collect("in:", fn.Params)
	collect("out:", fn.Results)
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

// jaccardUint64 intersects two sorted, deduplicated slices in a single pass.
func jaccardUint64(a, b []uint64) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	var shared int
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			shared++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	union := len(a) + len(b) - shared
	if union == 0 {
		return 0
	}
	return float64(shared) / float64(union)
}

// jaccardStrings treats two empty sets as identical: a pair of functions that
// both take and return nothing genuinely agree on their signature.
func jaccardStrings(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	set := make(map[string]struct{}, len(a))
	for _, s := range a {
		set[s] = struct{}{}
	}
	var shared int
	for _, s := range b {
		if _, ok := set[s]; ok {
			shared++
			delete(set, s)
		}
	}
	union := len(a) + len(b) - shared
	if union == 0 {
		return 0
	}
	return float64(shared) / float64(union)
}

// cosineInts compares two control-flow histograms. Two bodies with no control
// flow at all are equally flat, so both-zero counts as a match.
func cosineInts(a, b []int) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		normA += x * x
		normB += y * y
	}
	if normA == 0 && normB == 0 {
		return 1
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}

func ratio(a, b int) float64 {
	if a == 0 || b == 0 {
		return 0
	}
	if a > b {
		a, b = b, a
	}
	return float64(a) / float64(b)
}

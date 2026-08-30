// Package fingerprint builds deterministic static summaries of function
// bodies and scores how similar two of them are. No network, no model, no
// cache: the same source always yields the same fingerprint and the same score.
//
// It works on internal/syntax, not on any language's own AST, so a frontend
// that can produce a syntax.Func gets a Fingerprint — and with it the
// similarity score, the retrieval shape channel, containment and the label
// bag's explanations — without this package learning anything about the
// language.
//
// # Weisfeiler-Lehman labels
//
// wl.go computes a Weisfeiler-Lehman label bag over a function's shape
// (WLBag), and LabelWeights turns a population of bags into the ln(N/df)
// surprisal of each label. The bag is built from syntax.Func.Shape — the
// frontend's canonical body where the frontend has a canonicalizer, and the
// body as written where it does not — and the 0.60 component of the composite
// Score is a corpus-weighted multiset Jaccard over two of them.
//
// That component used to be Jaccard over hashed 3-grams of a flattened token
// stream. Both are structural, but a shingle is a window over a linearisation
// — it cannot tell a condition from the statement it guards, and two shingles
// that overlap in the stream may be nowhere near each other in the tree. A WL
// label is a subtree summary, so sharing one is a claim about shape rather
// than about token adjacency, and the corpus weighting then says how much
// that particular shape is worth here. Shingles are still built: they feed
// snapshot.Digest, which answers "did this body change" about the code as
// written rather than about its canonical shape.
//
// One choice in it is worth stating up front, because it decides what the
// whole bag can mean: label_0 collapses every identifier to a single ID
// label, keeping no name. A call is the one exception — it keeps its callee
// name, exactly as the token stream does, because that name is intent.
//
// The alternative was to label an identifier by its text. On a canonical tree
// that text is one of two things and neither should be a label. A bound
// identifier has been alpha-renamed to x0, x1, ... in binding order, so
// labelling by it would put binding order into every label above it and two
// functions that declare the same two locals in the other order would share
// nothing. A free identifier keeps its source name, but that vocabulary is
// already carried where it is intent-bearing — on the call labels — and
// admitting type names, package qualifiers and field names besides would make
// the bag a lexical index rather than a structural one, which is what this
// tool refuses to be everywhere else.
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
// rather than taxing the WL or signature components.
//
// The 0.60 slot changed what it measures — token-shingle Jaccard became
// corpus-weighted multiset Jaccard over the Weisfeiler-Lehman label bags —
// but not how much it is worth. The blend was tuned against the old metric
// and re-tuning it in the same change as the metric would leave nothing to
// attribute a ranking move to.
const (
	weightWL        = 0.60
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
	Shingles []uint64 // sorted, deduped FNV-1a hashes of token 3-grams
	Flow     []int    // control-flow node histogram, length flowKinds
	Depth    []int    // control-flow entry-depth histogram, length depthBuckets
	Types    []string // sorted, deduped normalized param + result types
	Nodes    int      // syntax node count of the body (size / triviality guard)

	// WL is the Weisfeiler-Lehman label multiset of the body's canonical
	// shape, rounds 0..3 merged and sorted ascending by label — see WLBag.
	//
	// It is what the 0.60 component of Score measures and the only
	// structural multiset on a Fingerprint: it is also the shape retrieval
	// channel's feature set. Shingles is still built and still hashed into
	// snapshot.Digest — the digest answers "did this body change", which is
	// a question about the code as written, not about its canonical shape.
	WL []LabelCount
}

// Breakdown is the per-component result of comparing two Fingerprints.
//
// Two of its six numbers are reported and never scored. SizeRatio is not
// scored because Jaccard already penalises size mismatch through its union
// and damping again would double-count it. Containment is not scored because
// it answers a different question, and folding it into Score would destroy
// the distinction: a 40-line function whose whole shape reappears inside a
// 400-line one has containment near 1.0 and Jaccard near 0.1, and that is a
// finding — extracted-helper-shaped — that no single blended number states.
type Breakdown struct {
	WL          float64 // corpus-weighted multiset Jaccard over the WL label bags
	Flow        float64 // cosine over the control-flow histogram
	Depth       float64 // cosine over the nesting-depth histogram (reported as nesting:)
	Signature   float64 // Jaccard over normalized parameter and result types
	SizeRatio   float64 // min(Nodes)/max(Nodes); reported, not scored
	Containment float64 // shared WL mass over the smaller side's; reported, not scored
	Score       float64 // weighted composite, 0.0-1.0
}

// Build summarises a function. A nil function or one without a body
// (external, forward-declared, or unsegmentable by its frontend) yields the
// zero Fingerprint.
//
// Two trees are read, and which is which is load-bearing. Everything the
// token stream, the histograms and the node count measure comes from Body —
// the code as written — because that is the question those components answer.
// WL comes from Shape, the canonical body where the frontend has a
// canonicalizer, because a shape key should not carry the incidental choices
// canonicalization exists to remove.
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
		WL:       WLBag(fn),
	}
}

// Weights blends the four Breakdown components into Score. The production
// path always uses DefaultWeights; the type exists so the bench harness can
// measure the blend (a sensitivity sweep) without a package-level mutable —
// the same seam idiom as ontology.WithWeights.
type Weights struct {
	WL, Flow, Depth, Signature float64
}

// DefaultWeights returns the shipped blend: 0.60 / 0.20 / 0.05 / 0.15.
func DefaultWeights() Weights {
	return Weights{WL: weightWL, Flow: weightFlow, Depth: weightDepth, Signature: weightSignature}
}

// Sum is the blend total, 1.00 for the defaults.
func (w Weights) Sum() float64 { return w.WL + w.Flow + w.Depth + w.Signature }

// Scaled multiplies component i (0 WL, 1 Flow, 2 Depth, 3 Signature) by f
// and renormalizes the other three uniformly so the sum stays what it was —
// the ontology.WithWeights idiom, so a sweep moves one knob at a time.
func (w Weights) Scaled(i int, f float64) Weights {
	c := [4]float64{w.WL, w.Flow, w.Depth, w.Signature}
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
	return Weights{WL: c[0], Flow: c[1], Depth: c[2], Signature: c[3]}
}

// Similarity scores two Fingerprints against a corpus label weighting. It is
// symmetric, and returns the zero Breakdown when either side has no body.
//
// # Code shape is corpus-dependent now
//
// idf is what makes the WL component information-weighted: sharing a label
// every function in the corpus carries is worth ln(N/N) = 0, sharing one two
// functions carry is worth nearly ln(N). The same two bodies therefore score
// differently in different corpora, which is deliberate and is the same
// property structural overlap has had since it started reading the corpus IC.
// What "alike" means depends on what else is around, and a shape every
// function in the repo has is not a finding.
//
// A nil or empty idf falls back to weight 1 for every label, which is plain
// unweighted multiset Jaccard. That is the honest answer when there is no
// corpus to ask — not a silent zero — and it is what the library API and the
// package's own unit tests score under.
func Similarity(a, b Fingerprint, idf *LabelIDF) Breakdown {
	return SimilarityWith(a, b, idf, DefaultWeights())
}

// SimilarityWith is Similarity under an explicit blend. With DefaultWeights
// it is bit-identical to Similarity: the same four-term sum in the same
// order, which is what keeps the measurement seam a no-op at its defaults.
func SimilarityWith(a, b Fingerprint, idf *LabelIDF, w Weights) Breakdown {
	if a.Nodes == 0 || b.Nodes == 0 {
		return Breakdown{}
	}
	shape, contain := wlOverlap(a.WL, b.WL, idf)
	bd := Breakdown{
		WL:          shape,
		Flow:        cosineInts(a.Flow, b.Flow),
		Depth:       cosineInts(a.Depth, b.Depth),
		Signature:   jaccardStrings(a.Types, b.Types),
		SizeRatio:   ratio(a.Nodes, b.Nodes),
		Containment: contain,
	}
	bd.Score = w.WL*bd.WL + w.Flow*bd.Flow + w.Depth*bd.Depth + w.Signature*bd.Signature
	if bd.Score > 1.0 {
		bd.Score = 1.0
	}
	return bd
}

// wlOverlap computes both weighted quantities over two sorted label bags in
// one merge:
//
//	jaccard     = Σ w·min(a,b) / Σ w·max(a,b)
//	containment = Σ w·min(a,b) / min(Σ w·a, Σ w·b)
//
// Both are ratios of information, in nats, over the same shared mass. They
// differ only in what they divide by, which is why one pass yields both: a
// full merge visits every entry of both bags, so the two side masses and the
// shared mass all fall out of it. Σ w·max is not accumulated separately
// because max(x,y) = x + y − min(x,y) term by term, so it is exactly
// massA + massB − shared.
//
// Jaccard asks how much of the pair's combined structure is shared;
// containment asks how much of the *smaller* function's structure the larger
// one also has. A helper inlined into a long function scores low on the first
// and high on the second, and the report says both.
//
// Zero denominators are 0.0, not 1.0, matching the zero Fingerprint's rule
// that no body never matches anything. Both are reachable with non-empty
// bags: when every label a pair carries is carried by every function in the
// corpus, every weight is ln(N/N) = 0 and the pair shares no information at
// all. That is the same convention TrophicSimilarity already uses for a pair
// whose every pattern is corpus idiom.
func wlOverlap(a, b []LabelCount, idf *LabelIDF) (jaccard, containment float64) {
	var massA, massB, shared float64
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case j == len(b) || (i < len(a) && a[i].Label < b[j].Label):
			massA += weightOf(idf, a[i].Label) * float64(a[i].Count)
			i++
		case i == len(a) || b[j].Label < a[i].Label:
			massB += weightOf(idf, b[j].Label) * float64(b[j].Count)
			j++
		default:
			w := weightOf(idf, a[i].Label)
			ca, cb := a[i].Count, b[j].Count
			massA += w * float64(ca)
			massB += w * float64(cb)
			shared += w * float64(min(ca, cb))
			i++
			j++
		}
	}
	if union := massA + massB - shared; union > 0 {
		jaccard = shared / union
	}
	if smaller := math.Min(massA, massB); smaller > 0 {
		containment = shared / smaller
	}
	return jaccard, containment
}

// weightOf is the uniform-fallback rule in one place: no corpus means every
// label is worth the same, so the weighted Jaccard degenerates to the plain
// multiset one rather than to zero.
func weightOf(idf *LabelIDF, label uint64) float64 {
	if idf == nil || idf.N() == 0 {
		return 1
	}
	return idf.Weight(label)
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

// Package fingerprint builds deterministic static summaries of Go function
// bodies and scores how similar two of them are. No network, no model, no
// cache: the same source always yields the same fingerprint and the same score.
package fingerprint

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"hash/fnv"
	"math"
	"sort"
	"strings"
)

// Weights for each component of the composite similarity Score.
// They sum to exactly 1.00.
const (
	weightAST       = 0.60
	weightFlow      = 0.25
	weightSignature = 0.15
)

// shingleK is the sliding-window width over the AST token stream.
const shingleK = 3

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

// Fingerprint is a deterministic static summary of one function body.
// The zero value means "no body" and never matches anything.
type Fingerprint struct {
	Shingles []uint64 // sorted, deduped FNV-1a hashes of AST 3-grams
	Flow     []int    // control-flow node histogram, length flowKinds
	Types    []string // sorted, deduped normalized param + result types
	Nodes    int      // AST node count of the body (size / triviality guard)
}

// Breakdown is the per-component result of comparing two Fingerprints.
type Breakdown struct {
	AST       float64 // Jaccard over AST 3-gram shingles
	Flow      float64 // cosine over the control-flow histogram
	Signature float64 // Jaccard over normalized parameter and result types
	SizeRatio float64 // min(Nodes)/max(Nodes); reported, not scored
	Score     float64 // weighted composite, 0.0-1.0
}

// Build summarises a function declaration. A nil declaration or a declaration
// without a body (external or forward-declared) yields the zero Fingerprint.
func Build(fd *ast.FuncDecl) Fingerprint {
	if fd == nil || fd.Body == nil {
		return Fingerprint{}
	}
	tokens, flow, nodes := walk(fd.Body)
	return Fingerprint{
		Shingles: shingle(tokens),
		Flow:     flow,
		Types:    typeStrings(fd.Type),
		Nodes:    nodes,
	}
}

// Similarity scores two Fingerprints. It is symmetric, and returns the zero
// Breakdown when either side has no body.
func Similarity(a, b Fingerprint) Breakdown {
	if a.Nodes == 0 || b.Nodes == 0 {
		return Breakdown{}
	}
	bd := Breakdown{
		AST:       jaccardUint64(a.Shingles, b.Shingles),
		Flow:      cosineInts(a.Flow, b.Flow),
		Signature: jaccardStrings(a.Types, b.Types),
		SizeRatio: ratio(a.Nodes, b.Nodes),
	}
	bd.Score = weightAST*bd.AST + weightFlow*bd.Flow + weightSignature*bd.Signature
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
func walk(body *ast.BlockStmt) (tokens []string, flow []int, nodes int) {
	flow = make([]int, flowKinds)
	ast.Inspect(body, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		nodes++
		switch node := n.(type) {
		case *ast.IfStmt:
			flow[flowIf]++
			tokens = append(tokens, "IF")
		case *ast.ForStmt:
			flow[flowFor]++
			tokens = append(tokens, "FOR")
		case *ast.RangeStmt:
			flow[flowRange]++
			tokens = append(tokens, "RANGE")
		case *ast.SwitchStmt:
			flow[flowSwitch]++
			tokens = append(tokens, "SWITCH")
		case *ast.TypeSwitchStmt:
			flow[flowTypeSwitch]++
			tokens = append(tokens, "TYPESWITCH")
		case *ast.SelectStmt:
			flow[flowSelect]++
			tokens = append(tokens, "SELECT")
		case *ast.ReturnStmt:
			flow[flowReturn]++
			tokens = append(tokens, "RETURN")
		case *ast.DeferStmt:
			flow[flowDefer]++
			tokens = append(tokens, "DEFER")
		case *ast.GoStmt:
			flow[flowGo]++
			tokens = append(tokens, "GO")
		case *ast.FuncLit:
			flow[flowFuncLit]++
			tokens = append(tokens, "FUNCLIT")
		case *ast.CallExpr:
			tokens = append(tokens, "CALL")
			if name := calleeName(node); name != "" {
				tokens = append(tokens, "CALL:"+name)
			}
		case *ast.BinaryExpr:
			tokens = append(tokens, "BIN:"+node.Op.String())
		case *ast.UnaryExpr:
			tokens = append(tokens, "UNARY:"+node.Op.String())
		case *ast.AssignStmt:
			tokens = append(tokens, "ASSIGN:"+node.Tok.String())
		case *ast.BranchStmt:
			tokens = append(tokens, "BRANCH:"+node.Tok.String())
		case *ast.BasicLit:
			tokens = append(tokens, "LIT:"+node.Kind.String())
		case *ast.IncDecStmt:
			tokens = append(tokens, "INCDEC")
		case *ast.IndexExpr:
			tokens = append(tokens, "INDEX")
		case *ast.SliceExpr:
			tokens = append(tokens, "SLICE")
		case *ast.StarExpr:
			tokens = append(tokens, "STAR")
		case *ast.TypeAssertExpr:
			tokens = append(tokens, "ASSERT")
		case *ast.CompositeLit:
			tokens = append(tokens, "COMPOSITE")
		case *ast.KeyValueExpr:
			tokens = append(tokens, "KV")
		case *ast.SelectorExpr:
			tokens = append(tokens, "SEL")
		case *ast.Ident:
			tokens = append(tokens, "ID")
		}
		return true
	})
	return tokens, flow, nodes
}

// calleeName mirrors the selector handling in extractCallees over in the parser
// package, but drops the receiver expression: the variable a method is called
// on is arbitrary (e, s, cfg) while the method name is not.
func calleeName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
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

// typeStrings renders the parameter and result types of a function, discarding
// the parameter names. Inputs and outputs are prefixed so that a parameter of
// type error does not match a returned error.
func typeStrings(ft *ast.FuncType) []string {
	if ft == nil {
		return nil
	}
	seen := make(map[string]struct{})
	collect := func(prefix string, fields *ast.FieldList) {
		if fields == nil {
			return
		}
		for _, field := range fields.List {
			typ := printType(field.Type)
			if typ == "" {
				continue
			}
			seen[prefix+typ] = struct{}{}
		}
	}
	collect("in:", ft.Params)
	collect("out:", ft.Results)
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

func printType(expr ast.Expr) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), expr); err != nil {
		return ""
	}
	return strings.Join(strings.Fields(buf.String()), " ")
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

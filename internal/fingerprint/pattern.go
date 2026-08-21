package fingerprint

import (
	"go/ast"
	"hash/fnv"
	"sort"
	"strings"
)

// Pattern levels: each level metabolizes the one below it — tokens feed
// expressions, expressions feed actions, actions feed motif chains. Higher
// levels carry more behavioral meaning; corpus IC (computed downstream)
// decides how much evidence any one pattern is worth.
const (
	LevelToken  uint8 = iota // L0: token n-gram windows (widths in l0Widths)
	LevelExpr                // L1: call / binary-operator shapes
	LevelAction              // L2: statement-with-salient-structure
	LevelMotif               // L3: loop-body call summaries, statement bigrams
)

// l0ExtraWidths are the additional L0 window widths beside shingleK: 5-grams
// are fine matches that certify longer shared runs. One index carries every
// resolution and the retriever's presence-df cap discards whichever width
// turns out idiomatic. Width 2 was built, measured on the cobra golden
// labels, and left out: merge ranks were identical with and without it
// (4.8), but the surviving 2-gram mass pulled the false-positive mean rank
// from 41.3 up to 38.7 — on a small corpus too many 2-grams sit under the df
// cap and feed exactly the vocabulary-heavy pairs the ranking's thin margins
// already worry about. On large corpora the cap kills them anyway. Re-adding
// 2 here is one edit, worth re-measuring once more corpora are labeled.
// Two deliberate asymmetries against the legacy k=3 windows:
//
//   - Extra widths never clamp. shingleK's short-stream fallback would make
//     a 2-token body emit the same hash under k=3 and k=5, silently
//     collapsing in the accumulator — exactly the bodies the widths exist to
//     discriminate. A stream shorter than the width simply emits nothing for
//     it.
//   - Extra-width hashes carry a leading "w<k>" part, so a window can never
//     collide with a k=3 window over the same tokens. k=3 keeps its untagged
//     hash input byte-identical, so every pre-existing pattern df is
//     unchanged by the widening.
var l0ExtraWidths = [...]int{5}

// Pattern is one structural feature of a function body, at one level of the
// hierarchy. For levels 1-3 the Render string IS the canonical serialization
// the Hash is computed over, so hash and human-readable form cannot drift;
// level-0 windows hash their tokens directly and carry no render.
type Pattern struct {
	Hash   uint64 // FNV-1a over "L<level>|" + canonical serialization
	Level  uint8
	Count  uint16 // multiset count within the function, saturating
	Render string // canonical human-readable form; "" for levels 0-1
}

// loopSummaryCap bounds a loop-body call summary: at most this many distinct
// callee names, with a trailing "..." marker when more exist.
const loopSummaryCap = 8

func patternHash(level uint8, parts ...string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte{'L', '0' + level, '|'})
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}

// patternHashL0 hashes one token window, optionally width-tagged. An empty
// tag is byte-identical to patternHash(LevelToken, window...) — the legacy
// k=3 hash — and avoids the per-window slice allocation the variadic form
// would cost on this hot path.
func patternHashL0(tag string, window []string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte{'L', '0' + LevelToken, '|'})
	if tag != "" {
		_, _ = h.Write([]byte(tag))
		_, _ = h.Write([]byte{0})
	}
	for _, p := range window {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}

// widthTag names an extra L0 width in the hash input: "w2", "w5".
func widthTag(w int) string { return "w" + string(rune('0'+w)) }

// extractPatterns walks the body once and accumulates the multi-level pattern
// multiset. tokens is the walk() token stream, reused for the L0 windows.
// Output is sorted by (Hash, Level, Render) — a total order, so the
// accumulator map cannot leak iteration order.
func extractPatterns(body *ast.BlockStmt, tokens []string) []Pattern {
	acc := make(map[uint64]*Pattern)
	add := func(hash uint64, level uint8, render string) {
		if p, ok := acc[hash]; ok {
			if p.Count < ^uint16(0) {
				p.Count++
			}
			return
		}
		acc[hash] = &Pattern{Hash: hash, Level: level, Count: 1, Render: render}
	}
	addRendered := func(level uint8, render string) {
		keep := render
		if level < LevelAction {
			keep = "" // L1 renders are serializations only, never shown
		}
		add(patternHash(level, render), level, keep)
	}

	// L0: counted sliding windows over the token stream at every width in
	// {shingleK} ∪ l0ExtraWidths. The k=3 pass keeps shingle()'s short-stream
	// fallback and untagged hash input; the extra widths are width-tagged and
	// never clamp — see l0ExtraWidths. Counts matter here, unlike the deduped
	// Shingles.
	if len(tokens) > 0 {
		k := shingleK
		if len(tokens) < k {
			k = len(tokens)
		}
		for i := 0; i+k <= len(tokens); i++ {
			add(patternHashL0("", tokens[i:i+k]), LevelToken, "")
		}
		for _, w := range l0ExtraWidths {
			if len(tokens) < w {
				continue
			}
			tag := widthTag(w)
			for i := 0; i+w <= len(tokens); i++ {
				add(patternHashL0(tag, tokens[i:i+w]), LevelToken, "")
			}
		}
	}

	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			addRendered(LevelExpr, "call:"+callLabel(node))
		case *ast.BinaryExpr:
			addRendered(LevelExpr, binRender(node))
		case *ast.ForStmt:
			if r := loopSummary("for", node.Init, node.Cond, node.Post, node.Body); r != "" {
				addRendered(LevelMotif, r)
			}
		case *ast.RangeStmt:
			if r := loopSummary("range", node.X, node.Body); r != "" {
				addRendered(LevelMotif, r)
			}
		case *ast.BlockStmt:
			for _, r := range seqBigrams(node) {
				addRendered(LevelMotif, r)
			}
		default:
			if s, ok := n.(ast.Stmt); ok {
				if r := stmtRender(s); r != "" {
					addRendered(LevelAction, r)
				}
			}
		}
		return true
	})

	if len(acc) == 0 {
		return nil
	}
	out := make([]Pattern, 0, len(acc))
	for _, p := range acc {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Hash != out[j].Hash {
			return out[i].Hash < out[j].Hash
		}
		if out[i].Level != out[j].Level {
			return out[i].Level < out[j].Level
		}
		return out[i].Render < out[j].Render
	})
	return out
}

// callLabel names a call for pattern purposes: the callee name when one is
// syntactically visible, "funclit" for immediately-invoked literals, "?"
// otherwise. Receiver expressions stay dropped, same rule as calleeName.
func callLabel(call *ast.CallExpr) string {
	if _, ok := call.Fun.(*ast.FuncLit); ok {
		return "funclit"
	}
	if name := calleeName(call); name != "" {
		return name
	}
	return "?"
}

// exprKind is the L1 vocabulary for operand shapes. The universal constants
// nil/true/false keep their names — that is what makes the err != nil idiom
// fall out as its own pattern with no special case — while every other
// identifier collapses to "id".
func exprKind(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.ParenExpr:
		return exprKind(v.X)
	case *ast.Ident:
		switch v.Name {
		case "nil", "true", "false":
			return v.Name
		}
		return "id"
	case *ast.BasicLit:
		return "lit:" + v.Kind.String()
	case *ast.CallExpr:
		return "call:" + callLabel(v)
	case *ast.SelectorExpr:
		return "sel"
	case *ast.BinaryExpr:
		return "bin"
	case *ast.UnaryExpr:
		return "unary"
	case *ast.CompositeLit:
		return "composite"
	case *ast.FuncLit:
		return "funclit"
	case *ast.IndexExpr:
		return "index"
	case *ast.SliceExpr:
		return "slice"
	case *ast.StarExpr:
		return "star"
	case *ast.TypeAssertExpr:
		return "assert"
	}
	return "expr"
}

func binRender(b *ast.BinaryExpr) string {
	return "bin:" + b.Op.String() + "(" + exprKind(b.X) + "," + exprKind(b.Y) + ")"
}

// stmtRender is the L2 vocabulary: a statement kind with its salient
// structure. Loops and switches render nothing here — summarizing what
// happens inside them is the motif level's job.
func stmtRender(s ast.Stmt) string {
	switch v := s.(type) {
	case *ast.ReturnStmt:
		parts := make([]string, 0, len(v.Results))
		for _, r := range v.Results {
			parts = append(parts, exprKind(r))
		}
		return "return(" + strings.Join(parts, ",") + ")"
	case *ast.AssignStmt:
		parts := make([]string, 0, len(v.Rhs))
		for _, r := range v.Rhs {
			parts = append(parts, exprKind(r))
		}
		return "assign" + v.Tok.String() + "(" + strings.Join(parts, ",") + ")"
	case *ast.DeferStmt:
		return "defer(" + callInner(v.Call) + ")"
	case *ast.GoStmt:
		return "go(" + callInner(v.Call) + ")"
	case *ast.IfStmt:
		if cond, ok := v.Cond.(*ast.BinaryExpr); ok {
			return "if(" + binRender(cond) + ")"
		}
		return "if(" + exprKind(v.Cond) + ")"
	case *ast.SendStmt:
		return "send(" + exprKind(v.Value) + ")"
	case *ast.ExprStmt:
		if call, ok := v.X.(*ast.CallExpr); ok {
			return "do(" + callInner(call) + ")"
		}
	}
	return ""
}

// callInner renders a called thing for defer/go/do contexts: "funclit" for
// literals (the interesting fact is that a literal runs there), otherwise the
// call pattern.
func callInner(call *ast.CallExpr) string {
	if _, ok := call.Fun.(*ast.FuncLit); ok {
		return "funclit"
	}
	return "call:" + callLabel(call)
}

// loopSummary is the L3 signature of one loop: the distinct named callees
// anywhere in its header and body, in first-occurrence syntactic order,
// capped. The header matters — `for scanner.Scan()` puts the loop's defining
// call in the condition — so the summary reads the way the loop behaves:
// "for{ call:Scan call:TrimSpace call:Atoi call:append }". Empty for loops
// that call nothing named.
func loopSummary(keyword string, parts ...ast.Node) string {
	var names []string
	seen := make(map[string]bool)
	truncated := false
	for _, part := range parts {
		// Init/Cond/Post/X are interface fields: absent means a nil
		// interface, caught here. Body is a concrete pointer, so guard the
		// typed-nil wrap explicitly.
		if part == nil {
			continue
		}
		if b, ok := part.(*ast.BlockStmt); ok && b == nil {
			continue
		}
		ast.Inspect(part, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := calleeName(call)
			if name == "" || seen[name] {
				return true
			}
			seen[name] = true
			if len(names) < loopSummaryCap {
				names = append(names, "call:"+name)
			} else {
				truncated = true
			}
			return true
		})
	}
	if len(names) == 0 {
		return ""
	}
	if truncated {
		names = append(names, "...")
	}
	return keyword + "{ " + strings.Join(names, " ") + " }"
}

// seqBigrams is the other L3 form: adjacent-statement bigrams within one
// block, where each statement is its L2 render or a container keyword.
// Statements with neither break the window, so bigrams never bridge over
// unmodeled structure.
func seqBigrams(block *ast.BlockStmt) []string {
	items := make([]string, len(block.List))
	for i, s := range block.List {
		switch s.(type) {
		case *ast.ForStmt:
			items[i] = "for"
		case *ast.RangeStmt:
			items[i] = "range"
		case *ast.SwitchStmt:
			items[i] = "switch"
		case *ast.TypeSwitchStmt:
			items[i] = "typeswitch"
		case *ast.SelectStmt:
			items[i] = "select"
		default:
			items[i] = stmtRender(s)
		}
	}
	var out []string
	for i := 0; i+1 < len(items); i++ {
		if items[i] != "" && items[i+1] != "" {
			out = append(out, "seq[ "+items[i]+" ; "+items[i+1]+" ]")
		}
	}
	return out
}

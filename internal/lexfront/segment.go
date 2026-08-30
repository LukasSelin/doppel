package lexfront

import "strings"

// stmtKeywords can never be the name of a declaration. Without this list
// "except ValueError:" and "if cond:" both match the shape a Python function
// declaration has — an identifier, a parenthesised group, then a colon — and
// on the Python standard library they were 46% of everything found.
//
// The list is universal statement vocabulary, not per-language: a language
// that uses none of these loses nothing, and no language declares a function
// called "if".
var stmtKeywords = map[string]bool{
	"if": true, "elif": true, "elsif": true, "else": true, "while": true,
	"for": true, "foreach": true, "switch": true, "case": true, "when": true,
	"match": true, "with": true, "try": true, "except": true, "catch": true,
	"finally": true, "ensure": true, "rescue": true, "return": true,
	"yield": true, "assert": true, "del": true, "raise": true, "throw": true,
	"throws": true, "and": true, "or": true, "not": true, "in": true,
	"is": true, "as": true, "await": true, "async": true, "defer": true,
	"go": true, "select": true, "break": true, "continue": true, "goto": true,
	"import": true, "from": true, "using": true, "package": true, "new": true,
	"delete": true, "typeof": true, "instanceof": true, "sizeof": true,
	"print": true, "lambda": true, "do": true, "then": true, "end": true,
	"pass": true, "raise_": true, "global": true, "nonlocal": true,
	"unless": true, "until": true, "loop": true, "where": true, "guard": true,
	"repeat": true, "elseif": true, "fi": true, "esac": true,
}

// isStmtKeyword reports whether a word is universal statement vocabulary.
func isStmtKeyword(w string) bool { return stmtKeywords[w] }

// span is one detected function: where its header starts, where its body
// starts and ends, and the name and parameters recovered from the header.
type span struct {
	name      string
	receiver  string
	params    []string
	nameTok   int
	bodyStart int // first token of the body, just past "{" or the ":" newline
	bodyEnd   int // one past the last token of the body
	declTok   int // first token of the declaration, for source offsets
	blockOpen bool
	// container marks a type or module declaration: it is not emitted as a
	// unit, and — the reason it must be detected at all — it does not count
	// as an enclosing function, so its methods stay top-level units.
	container bool
}

// isContainer reports whether the declaration introducing this name is a type
// or module rather than a function.
func isContainer(toks []token, name int, container map[string]bool) bool {
	for j := name - 1; j >= 0 && j >= name-4; j-- {
		if toks[j].kind != tokIdent {
			break
		}
		if container[toks[j].text] {
			return true
		}
	}
	return false
}

// segment finds function-like declarations.
//
// The rule is the one thing a lexical frontend cannot avoid guessing at, so
// it is kept narrow and stated plainly: a name, a balanced parenthesised
// list, then a body introduced by "{" or by ":" at end of line. That covers
// keyword-led declarations (def/fn/func/function), type-led ones (Java, C++,
// C#), methods inside classes, and assigned function expressions.
//
// It will miss things, and the misses are the honest cost of having no
// grammar — internal/bench measures the miss rate against go/ast on Go
// corpora rather than leaving it to be discovered in the field.
func segment(toks []token, sp Spec) []span {
	kw := map[string]bool{}
	for _, k := range sp.FuncKeywords {
		for _, part := range strings.Fields(k) {
			kw[part] = true
		}
	}
	container := map[string]bool{}
	for _, k := range sp.ContainerKeywords {
		container[k] = true
	}

	var out []span
	for i := 0; i < len(toks); i++ {
		if toks[i].kind != tokIdent {
			continue
		}
		// A function keyword is not a name. "func(i, j int) bool { ... }" is
		// an anonymous literal, and its enclosing function already accounts
		// for it — gofront folds a FuncLit into its parent for the same
		// reason. Without this check every closure became a unit called
		// "func", which was 11% of everything found on this repo.
		if kw[toks[i].text] {
			continue
		}
		// The name is the identifier immediately before a "(", allowing a
		// type-parameter list in between: "func sortedKeys[K comparable](m M)".
		paren := i + 1
		if paren < len(toks) && toks[paren].kind == tokOpen && toks[paren].text == "[" {
			if c := matchBracket(toks, paren); c >= 0 {
				paren = c + 1
			}
		}
		if paren >= len(toks) || toks[paren].kind != tokOpen || toks[paren].text != "(" {
			continue
		}
		// A call, not a declaration: "foo(" preceded by "." or "=" or an
		// opening bracket is an expression. A declaration is preceded by a
		// keyword, a type, a modifier, or a line start.
		if stmtKeywords[toks[i].text] {
			continue
		}
		if !declContext(toks, i, kw) {
			continue
		}
		close := matchBracket(toks, paren)
		if close < 0 {
			continue
		}
		body, open := bodyStart(toks, close+1, toks[close].line)
		if body < 0 {
			continue
		}
		decl := declStart(toks, i)
		var end int
		if open {
			cb := matchBracket(toks, body-1)
			if cb < 0 {
				continue
			}
			end = cb
		} else {
			// The suite ends at the first line indented no further than the
			// *declaration*, which is where "def" sits — not where the name
			// sits. Measuring from the name made every indented method's
			// body end on its own first statement, because that statement is
			// indented less than the name it follows.
			end = indentEnd(toks, body, toks[decl].col)
		}
		// An empty brace body is still a body: "func nop() {}" is a real
		// function with an empty block, and gofront emits one. An empty
		// indent suite is not — nothing followed the colon — so the two
		// styles differ here on purpose.
		if open && end < body {
			continue
		}
		if !open && end <= body {
			continue
		}
		s := span{
			container: isContainer(toks, i, container),
			name:      toks[i].text,
			nameTok:   i,
			params:    paramNames(toks, paren, close, sp),
			bodyStart: body,
			bodyEnd:   end,
			declTok:   decl,
			blockOpen: open,
		}
		s.receiver = receiverOf(toks, decl, i)
		out = append(out, s)
		// Continue scanning from inside the body so nested functions and
		// methods inside classes are found too. Nested closures assigned to
		// names are functions in every language that has them.
		_ = close
	}
	return dropNested(out)
}

// declContext rejects the common false positive: an ordinary call.
func declContext(toks []token, i int, kw map[string]bool) bool {
	if i == 0 {
		return true
	}
	prev := toks[i-1]
	// A keyword introducing a declaration.
	if prev.kind == tokIdent && kw[prev.text] {
		return true
	}
	switch prev.kind {
	case tokOp:
		// "." is a method call; "=" and "=>" precede assigned functions,
		// which the caller detects by the body that follows.
		if prev.text == "." || prev.text == "?." || prev.text == "::" {
			return false
		}
		return false
	case tokOpen, tokSemi:
		return false
	case tokIdent:
		// A type or modifier before the name: "void run(", "int main(",
		// "public static String name(". This is what covers the languages
		// with no function keyword at all — but a statement keyword before
		// the name means a call, not a declaration: "if isinstance(x, y):"
		// otherwise declares a function called isinstance.
		return !stmtKeywords[prev.text]
	case tokClose:
		// ") name(" — a Go-style receiver or a C++ trailing construct.
		return true
	}
	// Start of a line with nothing before it.
	return toks[i].nlPrev
}

// declStart walks back over modifiers and types to the first token of the
// declaration, so the reported source and line cover the whole thing.
func declStart(toks []token, name int) int {
	i := name
	for i > 0 {
		p := toks[i-1]
		// A receiver or template group sits between the keyword and the
		// name: "func (s *Server) Start(", "template<T> void f(".
		if p.kind == tokClose {
			if o := openerBefore(toks, i-1); o >= 0 {
				i = o
				continue
			}
			break
		}
		if p.kind == tokIdent && !p.nlPrev {
			i--
			continue
		}
		if p.kind == tokIdent && p.nlPrev {
			return i - 1
		}
		break
	}
	return i
}

// openerBefore returns the index of the bracket opening the one at close.
func openerBefore(toks []token, close int) int {
	depth := 0
	for j := close; j >= 0; j-- {
		switch toks[j].kind {
		case tokClose:
			depth++
		case tokOpen:
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}

// receiverOf recovers a Go/Rust-style receiver or a qualified declarator:
// the identifier inside a parenthesised group before the name, or the type
// before "::"/"." in the declaration head.
func receiverOf(toks []token, start, name int) string {
	// A qualified declarator: "void Widget::render(", "Widget.prototype.f =".
	for i := start; i < name-1; i++ {
		if toks[i].kind == tokOp && (toks[i].text == "::" || toks[i].text == ".") &&
			i > start && toks[i-1].kind == tokIdent {
			return toks[i-1].text
		}
	}
	// A parenthesised receiver group before the name: Go's "func (s *Server)".
	// The type is the last identifier in the group, and a leading "*" is kept
	// because the qualified unit name carries it — parser.MethodName strips
	// it back off, and gofront does exactly the same.
	for i := start; i < name-1; i++ {
		if toks[i].kind != tokOpen || toks[i].text != "(" {
			continue
		}
		c := matchBracket(toks, i)
		if c < 0 || c >= name {
			break
		}
		star := ""
		last := ""
		for j := i + 1; j < c; j++ {
			if toks[j].kind == tokOp && toks[j].text == "*" {
				star = "*"
			}
			if toks[j].kind == tokIdent {
				last = toks[j].text
			}
		}
		if last != "" {
			return star + last
		}
		break
	}
	return ""
}

// bodyStart returns the first body token after the header, and whether the
// body is brace-delimited. A ":" that ends a line opens an indented suite.
// The scan is bounded to the header's own line plus one, and that bound is
// what stops a call from being mistaken for a declaration: an unbounded scan
// walked forward from `print("x")` until it met the colon of a completely
// unrelated `if` two lines later, and declared a function called print.
// Allman brace style is why the bound is one line rather than zero.
func bodyStart(toks []token, i, headLine int) (int, bool) {
	depth := 0
	for ; i < len(toks); i++ {
		t := toks[i]
		if t.kind == tokEOF || t.line > headLine+1 {
			return -1, false
		}
		if t.line > headLine && depth == 0 {
			// Only a brace may open a body on the following line.
			if t.kind == tokOpen && t.text == "{" {
				return i + 1, true
			}
			return -1, false
		}
		switch {
		case t.kind == tokOpen && t.text == "{" && depth == 0:
			return i + 1, true
		case t.kind == tokOpen:
			// A return type can bracket: "[]T", "(int, error)", "Map<K,V>".
			// Bailing on the first closing bracket cost 22% of Go's
			// functions before this depth counter existed.
			depth++
		case t.kind == tokClose:
			depth--
			if depth < 0 {
				return -1, false
			}
		case t.kind == tokOp && t.text == ":" && depth == 0:
			// A colon ending the line opens an indented suite. A colon
			// followed by more of the same line is a return-type annotation
			// — "async load(n: string): Promise<T> {" — so the scan must
			// carry on to the brace rather than give up, which is what cost
			// every annotated TypeScript method.
			if i+1 < len(toks) && toks[i+1].nlPrev {
				return i + 1, false
			}
		case t.kind == tokSemi && t.text == ";" && depth == 0:
			return -1, false // a declaration with no body
		}
	}
	return -1, false
}

// indentEnd finds where an indented suite stops: the first token on a new
// line at or left of the header's own column.
func indentEnd(toks []token, start, headerCol int) int {
	for i := start; i < len(toks); i++ {
		if toks[i].kind == tokEOF {
			return i
		}
		if toks[i].nlPrev && toks[i].col <= headerCol {
			return i
		}
	}
	return len(toks)
}

// matchBracket returns the index of the bracket closing the one at open, or
// -1 if it is unbalanced.
func matchBracket(toks []token, open int) int {
	if open >= len(toks) || toks[open].kind != tokOpen {
		return -1
	}
	depth := 0
	for i := open; i < len(toks); i++ {
		switch toks[i].kind {
		case tokOpen:
			depth++
		case tokClose:
			depth--
			if depth == 0 {
				return i
			}
		case tokEOF:
			return -1
		}
	}
	return -1
}

// paramNames recovers parameter names from the header.
//
// Without types there is no way to tell "int x" from "x int" by inspection,
// so the rule is positional and the Spec says which end to take: the first
// identifier of a group for name-first languages (Go, Rust, TypeScript), the
// last for type-first ones (C, Java, C#). A language whose parameters are
// bare names lands on the same token either way.
func paramNames(toks []token, open, close int, sp Spec) []string {
	var out []string
	depth := 0
	first, last := "", ""
	flush := func() {
		pick := last
		if sp.ParamNameFirst {
			pick = first
		}
		if pick != "" {
			out = append(out, pick)
		}
		first, last = "", ""
	}
	for i := open; i <= close && i < len(toks); i++ {
		t := toks[i]
		switch t.kind {
		case tokOpen:
			depth++
		case tokClose:
			depth--
			if depth == 0 {
				flush()
				return out
			}
		case tokSemi:
			if depth == 1 && t.text == "," {
				flush()
			}
		case tokIdent:
			if depth == 1 {
				if first == "" {
					first = t.text
				}
				last = t.text
			}
		case tokOp:
			// A default value or type annotation ends the name.
			if depth == 1 && (t.text == "=" || t.text == ":") {
				flush()
				// skip to the next comma at this depth
				for i++; i <= close && i < len(toks); i++ {
					if toks[i].kind == tokOpen {
						depth++
					} else if toks[i].kind == tokClose {
						depth--
						if depth == 0 {
							return out
						}
					} else if toks[i].kind == tokSemi && depth == 1 && toks[i].text == "," {
						break
					}
				}
			}
		}
	}
	flush()
	return out
}

// dropNested removes spans whose body lies entirely inside another's.
//
// A nested closure is real code, but it is already counted inside its parent
// — admitting both would double-count every callback-heavy file and make a
// function's own body look like a duplicate of the function it is declared
// in. gofront has the same behaviour for free: only top-level FuncDecls
// become units, and a FuncLit is part of its enclosing function.
func dropNested(spans []span) []span {
	var out []span
	for i, s := range spans {
		if s.container {
			continue // a class is not a unit
		}
		nested := false
		for j, o := range spans {
			if i == j || o.container {
				continue // ... and does not enclose one either
			}
			if s.declTok > o.bodyStart && s.bodyEnd <= o.bodyEnd {
				nested = true
				break
			}
		}
		if !nested {
			out = append(out, s)
		}
	}
	return out
}

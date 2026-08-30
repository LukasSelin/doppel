package lexfront

import "strings"

// tokKind is the lexical vocabulary. It is deliberately tiny: the shape
// stages downstream need to tell apart names, literals, operators and
// grouping, and nothing finer.
type tokKind uint8

const (
	tokIdent tokKind = iota
	tokString
	tokNumber
	tokOp
	tokOpen  // ( [ {
	tokClose // ) ] }
	tokSemi  // ; and , — statement and list separators
	tokEOF
)

type token struct {
	kind tokKind
	text string
	// val is a string literal's decoded contents. Escape handling is
	// approximate on purpose: a literal is evidence about what a function
	// mentions, and getting "\n" exactly right does not change which
	// functions look alike.
	val    string
	line   int
	col    int
	off    int
	nlPrev bool // a newline (and only whitespace) preceded this token
}

// lex turns source into tokens, dropping comments and whitespace but
// recording where lines break, because an indent-delimited body cannot be
// found without that.
func lex(src []byte, sp Spec) []token {
	var out []token
	s := string(src)
	i, line, lineStart := 0, 1, 0
	nl := false

	atLineStart := func() int { return i - lineStart }

	for i < len(s) {
		c := s[i]

		// Whitespace and newlines.
		if c == '\n' {
			line++
			i++
			lineStart = i
			nl = true
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}

		// Comments.
		if cut, adv := skipComment(s, i, sp); cut {
			for j := i; j < adv && j < len(s); j++ {
				if s[j] == '\n' {
					line++
					lineStart = j + 1
					nl = true
				}
			}
			i = adv
			continue
		}

		start := i
		col := atLineStart()

		// Strings, including any raw prefix the language allows.
		if p := stringStart(s, i, sp); p >= 0 {
			val, adv := scanString(s, p, i, sp)
			for j := i; j < adv && j < len(s); j++ {
				if s[j] == '\n' {
					line++
					lineStart = j + 1
				}
			}
			out = append(out, token{kind: tokString, text: s[start:adv], val: val,
				line: line, col: col, off: start, nlPrev: nl})
			nl = false
			i = adv
			continue
		}

		// Identifiers and keywords.
		if isIdentStart(c) {
			j := i
			for j < len(s) && isIdentPart(s[j]) {
				j++
			}
			out = append(out, token{kind: tokIdent, text: s[i:j],
				line: line, col: col, off: start, nlPrev: nl})
			nl = false
			i = j
			continue
		}

		// Numbers.
		if c >= '0' && c <= '9' {
			j := i
			for j < len(s) && (isIdentPart(s[j]) || s[j] == '.') {
				j++
			}
			out = append(out, token{kind: tokNumber, text: s[i:j],
				line: line, col: col, off: start, nlPrev: nl})
			nl = false
			i = j
			continue
		}

		// Grouping and separators.
		switch c {
		case '(', '[', '{':
			out = append(out, token{kind: tokOpen, text: string(c), line: line, col: col, off: start, nlPrev: nl})
			nl = false
			i++
			continue
		case ')', ']', '}':
			out = append(out, token{kind: tokClose, text: string(c), line: line, col: col, off: start, nlPrev: nl})
			nl = false
			i++
			continue
		case ';', ',':
			out = append(out, token{kind: tokSemi, text: string(c), line: line, col: col, off: start, nlPrev: nl})
			nl = false
			i++
			continue
		}

		// Operators, longest match first so "==" does not lex as two "=".
		op := scanOp(s, i)
		out = append(out, token{kind: tokOp, text: op, line: line, col: col, off: start, nlPrev: nl})
		nl = false
		i += len(op)
	}
	out = append(out, token{kind: tokEOF, line: line, off: len(s), nlPrev: nl})
	return out
}

// skipComment reports whether a comment starts at i and where it ends.
func skipComment(s string, i int, sp Spec) (bool, int) {
	for _, lc := range sp.LineComment {
		if lc != "" && strings.HasPrefix(s[i:], lc) {
			j := strings.IndexByte(s[i:], '\n')
			if j < 0 {
				return true, len(s)
			}
			return true, i + j
		}
	}
	if sp.BlockOpen != "" && strings.HasPrefix(s[i:], sp.BlockOpen) {
		j := strings.Index(s[i+len(sp.BlockOpen):], sp.BlockClose)
		if j < 0 {
			return true, len(s)
		}
		return true, i + len(sp.BlockOpen) + j + len(sp.BlockClose)
	}
	return false, i
}

// stringStart returns the index of the opening quote if a string literal
// starts at i (allowing a raw/format prefix), or -1.
func stringStart(s string, i int, sp Spec) int {
	if i < len(s) && strings.IndexByte(sp.StringQuotes, s[i]) >= 0 {
		return i
	}
	// A prefix run followed by a quote: r"...", f"...", b"...".
	j := i
	for j < len(s) && strings.IndexByte(sp.RawPrefixes, s[j]) >= 0 {
		j++
	}
	if j > i && j < len(s) && strings.IndexByte(sp.StringQuotes, s[j]) >= 0 {
		return j
	}
	return -1
}

// scanString consumes a string literal and returns its contents and the index
// just past it. Triple-quoted strings are handled for any repeated quote,
// which covers Python's docstrings without a Python-specific rule.
func scanString(s string, q, start int, sp Spec) (string, int) {
	quote := s[q]
	raw := q > start // a prefix was present
	// Triple quote?
	if q+2 < len(s) && s[q+1] == quote && s[q+2] == quote {
		end := strings.Index(s[q+3:], strings.Repeat(string(quote), 3))
		if end < 0 {
			return s[q+3:], len(s)
		}
		return s[q+3 : q+3+end], q + 6 + end
	}
	var b strings.Builder
	i := q + 1
	for i < len(s) {
		c := s[i]
		if c == '\\' && !raw && quote != '`' {
			if i+1 < len(s) {
				b.WriteByte(unescape(s[i+1]))
				i += 2
				continue
			}
			i++
			continue
		}
		if c == quote {
			return b.String(), i + 1
		}
		// An unterminated single-quoted string would otherwise swallow the
		// rest of the file; a newline ends it for every quote but a backtick.
		if c == '\n' && quote != '`' {
			return b.String(), i
		}
		b.WriteByte(c)
		i++
	}
	return b.String(), len(s)
}

func unescape(c byte) byte {
	switch c {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	case '0':
		return 0
	}
	return c
}

// multiOps are the operators that must lex as one token. Longest first.
var multiOps = []string{
	"<<=", ">>=", "...", "===", "!==", "??=", "**=", "&&=", "||=",
	"=>", "->", "::", "<=", ">=", "==", "!=", "&&", "||", "++", "--",
	"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<", ">>", "**",
	"??", "?.", ":=", "<-",
}

func scanOp(s string, i int) string {
	for _, op := range multiOps {
		if strings.HasPrefix(s[i:], op) {
			return op
		}
	}
	return string(s[i])
}

func isIdentStart(c byte) bool {
	return c == '_' || c == '$' || c == '@' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

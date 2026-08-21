package analyzer

import "strings"

// Name stems, for the diverged-copy kind.
//
// Two functions are a fork when their names agree once version markers are
// stripped: evalCallOld and evalCall, scrapeLoopAppender and
// scrapeLoopAppenderV2. The markers are lexical and so is this — no
// go/types, no history — which is why the rule only ever labels a pair that
// code-shape already scored alike, and why the guards below exist. They are
// what keeps sha256/sha512, Threshold/Thresh and Newton/ton apart.

// stemMarkers are stripped as a suffix or a prefix, longest first so "orig"
// can never eat "original".
var stemMarkers = []string{"Original", "Deprecated", "Legacy", "Orig", "Copy", "Old", "New"}

// minStem is the shortest remainder any strip may leave: "Go" is not a stem
// of "GoOld", and "est" is not a stem of "oldest".
const minStem = 3

// stemPair reports whether two distinct names share a stem, and what it is.
//
// Guard one is the numbered-variant rule: when both names end in a bare digit
// run (not a v/V-prefixed version) and the runs differ, the two are siblings
// in a numbered series — sha256 and sha512, utf8 and utf16 — never a fork.
func stemPair(x, y string) (string, bool) {
	if x == y {
		return "", false
	}
	if dx, ok1 := bareDigitSuffix(x); ok1 {
		if dy, ok2 := bareDigitSuffix(y); ok2 && dx != dy {
			return "", false
		}
	}
	s, t := stem(x), stem(y)
	if s != t || len(s) < minStem {
		return "", false
	}
	return s, true
}

// stem strips version markers to a fixed point. Each round tries, in order:
// a trailing underscore, a trailing bare digit run, a trailing v/V-digit
// version, a suffix marker, a prefix marker, a leading underscore. Every
// strip must leave at least minStem characters.
func stem(name string) string {
	for {
		next := stripOnce(name)
		if next == name {
			return name
		}
		name = next
	}
}

func stripOnce(name string) string {
	if r := strings.TrimSuffix(name, "_"); r != name && len(r) >= minStem {
		return r
	}
	if r, ok := stripDigitSuffix(name); ok {
		return r
	}
	if r, ok := stripVersionSuffix(name); ok {
		return r
	}
	for _, m := range stemMarkers {
		if r, ok := stripMarkerSuffix(name, m); ok {
			return r
		}
	}
	for _, m := range stemMarkers {
		if r, ok := stripMarkerPrefix(name, m); ok {
			return r
		}
	}
	if r := strings.TrimPrefix(name, "_"); r != name && len(r) >= minStem {
		return r
	}
	return name
}

// bareDigitSuffix returns a trailing digit run not preceded by v/V.
func bareDigitSuffix(name string) (string, bool) {
	i := len(name)
	for i > 0 && isDigit(name[i-1]) {
		i--
	}
	if i == len(name) || i == 0 {
		return "", false
	}
	if c := name[i-1]; c == 'v' || c == 'V' {
		return "", false
	}
	return name[i:], true
}

// stripDigitSuffix drops a trailing bare digit run when the remainder ends in
// a letter and is long enough — handler2 → handler, but never 256 → "".
func stripDigitSuffix(name string) (string, bool) {
	d, ok := bareDigitSuffix(name)
	if !ok {
		return name, false
	}
	r := name[:len(name)-len(d)]
	if len(r) < minStem || !isLetter(r[len(r)-1]) {
		return name, false
	}
	return r, true
}

// stripVersionSuffix drops a trailing v2/V3 under the same remainder rule.
func stripVersionSuffix(name string) (string, bool) {
	i := len(name)
	for i > 0 && isDigit(name[i-1]) {
		i--
	}
	if i == len(name) || i < 2 {
		return name, false
	}
	if c := name[i-1]; c != 'v' && c != 'V' {
		return name, false
	}
	r := name[:i-1]
	if len(r) < minStem || !isLetter(r[len(r)-1]) {
		return name, false
	}
	return r, true
}

// stripMarkerSuffix drops a marker that starts a camel-case token or follows
// an underscore: evalCallOld and foo_old strip, Threshold does not — a
// lower-case "old" inside a word is not a marker.
func stripMarkerSuffix(name, marker string) (string, bool) {
	if len(name) < len(marker)+minStem {
		return name, false
	}
	tail := name[len(name)-len(marker):]
	if !strings.EqualFold(tail, marker) {
		return name, false
	}
	r := name[:len(name)-len(marker)]
	if !(isUpper(tail[0]) || strings.HasSuffix(r, "_")) {
		return name, false
	}
	r = strings.TrimSuffix(r, "_")
	if len(r) < minStem {
		return name, false
	}
	return r, true
}

// stripMarkerPrefix drops a marker followed by a token boundary: NewClient
// and new_client strip, Newton and newline do not.
func stripMarkerPrefix(name, marker string) (string, bool) {
	if len(name) < len(marker)+minStem {
		return name, false
	}
	head := name[:len(marker)]
	if !strings.EqualFold(head, marker) {
		return name, false
	}
	next := name[len(marker)]
	if !(isUpper(next) || isDigit(next) || next == '_') {
		return name, false
	}
	r := strings.TrimPrefix(name[len(marker):], "_")
	if len(r) < minStem {
		return name, false
	}
	return r, true
}

func isDigit(c byte) bool  { return c >= '0' && c <= '9' }
func isUpper(c byte) bool  { return c >= 'A' && c <= 'Z' }
func isLetter(c byte) bool { return isUpper(c) || (c >= 'a' && c <= 'z') }

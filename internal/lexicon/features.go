package lexicon

import (
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

// Feature channels. The prefix is part of the feature name, so a selector
// "json.Marshal" and a call token that renders the same string can never be
// conflated, and a name always says which evidence produced it.
//
// Every channel is filled from material the frontend already produces —
// TagSignals, the resolved call graph, and the fingerprint's Weisfeiler-Lehman
// label bag. None of it is a rule about what any of these strings *mean*, which is the
// whole point: a new language frontend that fills TagSignals and Fingerprint
// gets concepts without anyone writing a vocabulary for it.
const (
	ChanSelector = "sel"  // x.Sel expressions, including the nested tail (httpClient.Do)
	ChanImport   = "imp"  // import paths of the enclosing file
	ChanIdent    = "id"   // identifier name stems, camel/underscore split
	ChanLiteral  = "lit"  // the leading token of a string literal
	ChanCall     = "call" // resolved call tokens, internal qualified or import-pathed
	ChanAction   = "act"  // Weisfeiler-Lehman labels of whole statements — what a body does, structurally
	ChanFlow     = "flow" // binarized control-flow labels plus the node-kind flags
)

// seedChannels are the channels an emergent concept may be *founded* on: what
// the code calls, what it imports, what it reaches through. Every channel still
// contributes to a concept's learned vocabulary once its members are known —
// identifier stems and statement shapes are real evidence — but none of them
// may start a cluster on its own.
//
// The distinction is the difference between a vocabulary and a clustering.
// Without it, doppel's own corpus produced concepts founded on stems like "and"
// and "code" and on the literal "%d": groups of functions that genuinely
// co-occur those tokens, and mean nothing. What a function *does* is the thing
// worth founding a concept on.
var seedChannels = []string{ChanSelector, ChanCall, ChanImport}

// nameableChannels are the channels a concept may take its name from, in
// preference order. An action render is real evidence but a poor label — it is
// a serialized statement shape, not something a reader recognises — so naming
// falls back to it only when nothing better carries weight.
var nameableChannels = []string{ChanSelector, ChanCall, ChanImport, ChanIdent, ChanLiteral}

// maxLiteralToken bounds a literal feature. A whole string literal is usually
// unique to one function and would never clear the df floor anyway; the
// leading token is the part that recurs ("SELECT", "required").
const maxLiteralToken = 32

// minStem is the shortest identifier stem worth keeping. Below three
// characters a stem is a loop variable or a receiver, never a concept.
const minStem = 3

// unitFeatures returns one unit's deduped, sorted feature set.
func unitFeatures(u parser.CodeUnit, g *concepter.Graph, internal map[string]bool) []string {
	set := make(map[string]struct{})
	add := func(channel, name string) {
		if name == "" {
			return
		}
		set[channel+":"+name] = struct{}{}
	}

	for _, sel := range u.Signals.Selectors {
		add(ChanSelector, sel)
	}
	for _, imp := range u.Signals.Imports {
		add(ChanImport, imp)
	}
	for _, id := range u.Signals.IdentNames {
		for _, stem := range stems(id) {
			add(ChanIdent, stem)
		}
	}
	for _, lit := range u.Signals.StringLits {
		add(ChanLiteral, literalToken(lit))
	}
	for _, tok := range concepter.CallTokens(u, g, internal) {
		add(ChanCall, tok)
	}
	for _, l := range u.Fingerprint.WL {
		if !isActionLabel(l) {
			continue
		}
		add(ChanAction, fingerprint.DescribeLabel(l.H, l.Kind)+"#"+strconv.FormatUint(l.Label, 16))
	}
	for k, n := range u.Fingerprint.Flow {
		if n > 0 && k < len(fingerprint.FlowLabels) {
			add(ChanFlow, fingerprint.FlowLabels[k])
		}
	}
	if u.Signals.HasGoStmt {
		add(ChanFlow, "go")
	}
	if u.Signals.HasSelect {
		add(ChanFlow, "select")
	}
	if u.Signals.HasChan {
		add(ChanFlow, "chan")
	}

	out := make([]string, 0, len(set))
	for f := range set {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// actionRound is the Weisfeiler-Lehman refinement round the action channel
// reads. At h=0 a label is a bare node kind and at h=1 it adds only the kinds
// of the immediate children — both far too coarse to be an action, and both
// carried by nearly every function, so the df window would discard them
// anyway. h=2 is where a label folds in the grandchildren, which for a
// statement node is the statement *with its structure*: the granularity the
// retired pattern multiset called L2.
//
// Only h=2, not h>=2. An h=3 label at a statement describes a region rather
// than a statement, and every h=3 label's evidence is already implied by the
// h=2 labels beneath it — indexing both would double-count one piece of
// structure in a channel whose weights are lifts.
const actionRound = 2

// actionKinds are the node kinds whose h=2 label is worth calling an action:
// the statement forms, plus calls and assignments, which are the two
// expression shapes a reader would describe as something a function does.
// Everything else — identifiers, literals, blocks, type nodes — is grammar
// the function had no choice about.
//
// This is the one judgement in this file, and it is deliberately about *shape*
// rather than about meaning: no entry here says what any construct is for.
var actionKinds = map[fingerprint.LabelKind]bool{
	fingerprint.KindCall:       true,
	fingerprint.KindAssign:     true,
	fingerprint.KindIf:         true,
	fingerprint.KindFor:        true,
	fingerprint.KindRange:      true,
	fingerprint.KindSwitch:     true,
	fingerprint.KindTypeSwitch: true,
	fingerprint.KindSelect:     true,
	fingerprint.KindReturn:     true,
	fingerprint.KindDefer:      true,
	fingerprint.KindGo:         true,
	fingerprint.KindSend:       true,
	fingerprint.KindBranch:     true,
}

// isActionLabel reports whether a WL label belongs in the action channel.
func isActionLabel(l fingerprint.LabelCount) bool {
	return l.H == actionRound && actionKinds[l.Kind]
}

// stems splits an identifier into lower-cased word stems on camel-case and
// underscore boundaries. "retryWithBackoff" yields retry, with, backoff; the
// corpus df window then discards "with" on its own, without anyone deciding it
// is uninteresting.
func stems(id string) []string {
	if id == "" {
		return nil
	}
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() >= minStem {
			out = append(out, strings.ToLower(cur.String()))
		}
		cur.Reset()
	}
	runes := []rune(id)
	for i, r := range runes {
		switch {
		case r == '_' || r == '-':
			flush()
			continue
		case unicode.IsUpper(r) && i > 0 && !unicode.IsUpper(runes[i-1]):
			// camelCase boundary: lower then upper.
			flush()
		case unicode.IsUpper(r) && i > 0 && i+1 < len(runes) && unicode.IsUpper(runes[i-1]) &&
			unicode.IsLower(runes[i+1]):
			// acronym boundary: HTTPServer splits before the S.
			flush()
		}
		cur.WriteRune(r)
	}
	flush()
	return out
}

// literalToken reduces a string literal to its leading token.
func literalToken(lit string) string {
	lit = strings.TrimSpace(lit)
	if lit == "" {
		return ""
	}
	if i := strings.IndexFunc(lit, unicode.IsSpace); i > 0 {
		lit = lit[:i]
	}
	if len(lit) < 2 || len(lit) > maxLiteralToken {
		return ""
	}
	return lit
}

// channelOf splits a feature name into its channel and bare name.
func channelOf(feature string) (channel, name string) {
	if i := strings.IndexByte(feature, ':'); i > 0 {
		return feature[:i], feature[i+1:]
	}
	return "", feature
}

// canSeed reports whether a feature may found an emergent concept.
func canSeed(feature string) bool {
	channel, _ := channelOf(feature)
	for _, c := range seedChannels {
		if c == channel {
			return true
		}
	}
	return false
}

// Opaque reports whether a feature's name is a hash rather than a term a
// reader can look up: the structural channel's Weisfeiler-Lehman labels. It is
// the rule nameableChannels already applies to naming a concept, exported so
// the evidence a report prints for a concept pair can prefer the same legible
// features naming does.
func Opaque(feature string) bool {
	channel, _ := channelOf(feature)
	return channel == ChanAction
}

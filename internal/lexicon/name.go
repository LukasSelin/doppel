package lexicon

import (
	"strconv"
	"strings"
)

// maxNamePart bounds one component of a derived name. Action renders are
// serialized statement shapes and can run long; a name is a label, not the
// evidence, which the report prints separately.
const maxNamePart = 28

// maxNameParts bounds how many features a name may join. Three: a name is a
// handle, and the vocabulary it stands for is printed in full elsewhere.
const maxNameParts = 3

// nameConcepts gives every concept an identity derived from its own
// vocabulary: its highest-weight features, joined by "+".
//
//	sql.Open+QueryRow      json.Marshal+Unmarshal      errors.Wrap+%w
//
// The seed's name is deliberately *not* the identity. After expansion a
// seeded concept is whatever its members turned out to share, and a corpus
// where the rule fired on three unrelated things should not get to inherit the
// rule's claim about what it found. Seed stays on the concept as provenance and
// the report renders it in parentheses, which is the honest arrangement: the
// name states the evidence, the parenthetical states where the search started.
//
// Deterministic by construction — features are already sorted by (weight desc,
// name asc), collisions extend with the next feature in that order, and the
// last resort is a numeric suffix in processing order.
func nameConcepts(concepts []Concept) {
	taken := make(map[string]bool, len(concepts))
	for i := range concepts {
		concepts[i].ID = uniqueName(concepts[i].Features, taken)
		taken[concepts[i].ID] = true
	}
}

// uniqueName builds a name from the top features, widening it until it is
// unused. Two concepts agreeing on their two strongest features are close
// enough that the third is the informative difference, so widening beats a
// counter — but only up to maxNameParts, because past that the name stops
// distinguishing anything a reader can hold and a numeric suffix is the more
// honest admission that these two concepts look alike.
func uniqueName(features []Feature, taken map[string]bool) string {
	parts := nameParts(features)
	if len(parts) == 0 {
		parts = []string{"concept"}
	}
	maxWidth := maxNameParts
	if len(parts) < maxWidth {
		maxWidth = len(parts)
	}
	for width := 2; width <= maxWidth; width++ {
		id := strings.Join(parts[:width], "+")
		if !taken[id] {
			return id
		}
	}
	base := strings.Join(parts[:maxWidth], "+")
	if !taken[base] {
		return base
	}
	for n := 2; ; n++ {
		id := base + "~" + strconv.Itoa(n)
		if !taken[id] {
			return id
		}
	}
}

// nameParts returns the short forms a name may be built from, in the order a
// name should prefer them: the nameable channels in their own priority order,
// and within each channel by feature weight.
//
// Channel before weight is the load-bearing half. A concept's heaviest feature
// is often an identifier stem — "ref", "decode" — because stems are numerous
// and their lift can beat a call's. Naming by raw weight therefore produced
// "ref+decode" for a concept whose actual evidence is store.Get and
// store.Decode: a name a reader cannot look up, describing a practice by the
// variables that happen to surround it. What the code *calls* names it; the
// rest is available when nothing calls anything.
//
// Duplicate short forms are dropped — "sel:json.Marshal" and
// "call:encoding/json.Marshal" both shorten to json.Marshal, and a name reading
// "json.Marshal+json.Marshal" would say less than one of them alone.
func nameParts(features []Feature) []string {
	seen := make(map[string]bool)
	var out []string
	push := func(s string) {
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	// features arrives sorted by (weight desc, name asc), so one pass per
	// channel keeps that order inside each bucket.
	for _, channel := range nameableChannels {
		for _, f := range features {
			if c, _ := channelOf(f.Name); c == channel {
				push(shortName(f.Name))
			}
		}
	}
	for _, f := range features {
		if c, _ := channelOf(f.Name); !isNameable(c) {
			push(shortName(f.Name))
		}
	}
	return out
}

func isNameable(channel string) bool {
	for _, c := range nameableChannels {
		if c == channel {
			return true
		}
	}
	return false
}

// shortName reduces a feature to its label form: the channel prefix goes, an
// import path keeps its last two segments at most, and anything longer than
// maxNamePart is truncated with an ellipsis so a name stays scannable.
func shortName(feature string) string {
	channel, name := channelOf(feature)
	switch channel {
	case ChanCall, ChanImport:
		// "database/sql.Open" and "github.com/pkg/errors.Wrap" are precise and
		// unreadable; the tail is what a reader recognises.
		if i := strings.LastIndexByte(name, '/'); i >= 0 {
			name = name[i+1:]
		}
	}
	name = strings.TrimSpace(name)
	if len(name) > maxNamePart {
		name = name[:maxNamePart-1] + "…"
	}
	return name
}

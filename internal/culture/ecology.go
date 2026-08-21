package culture

import (
	"math"
	"sort"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
)

// AssocKind names the feature families an association can relate.
type AssocKind int

const (
	TagTag AssocKind = iota
	TagRole
	TagCall
)

func (k AssocKind) String() string {
	switch k {
	case TagTag:
		return "tag~tag"
	case TagRole:
		return "tag~role"
	case TagCall:
		return "tag~call"
	}
	return "unknown"
}

// Association is one corpus-derived relationship: features a and b co-occur
// on units far more (positive PMI) or far less (negative) than chance.
// PMI = ln(N·Count / (c(a)·c(b))); Count == 0 stores math.Inf(-1), rendered
// as "never", not as a number.
type Association struct {
	Kind     AssocKind
	A, B     string
	Count    int     // units carrying both features
	Expected float64 // c(a)·c(b)/N
	PMI      float64
}

// minAbsPMI is the reporting floor on association strength in either
// direction: co-occurrence at least twice (or at most half) the chance rate.
// Near-chance co-occurrence is not culture.
var minAbsPMI = math.Ln2

// buildAssociations counts unit-level binary co-occurrence and returns the
// association list with cutoffs applied. Counting rules: N = len(units);
// a unit contributes at most once to any marginal or pair count.
func buildAssociations(units []parser.CodeUnit, docs []concepter.ConceptDoc,
	tokens [][]string, opt Options) []Association {

	n := len(units)
	if n == 0 {
		return nil
	}

	tagCount := make(map[string]int)
	roleCount := make(map[string]int)
	tokenCount := make(map[string]int)
	pairTagTag := make(map[[2]string]int)
	pairTagRole := make(map[[2]string]int)
	pairTagCall := make(map[[2]string]int)

	for i := range units {
		tags := sortedUniqueTags(units[i].Patterns)
		for _, t := range tags {
			tagCount[t]++
		}
		role := docs[i].Role
		roleCount[role]++
		for _, t := range tokens[i] {
			tokenCount[t]++
		}
		for a := 0; a < len(tags); a++ {
			for b := a + 1; b < len(tags); b++ {
				pairTagTag[[2]string{tags[a], tags[b]}]++
			}
			pairTagRole[[2]string{tags[a], role}]++
			for _, tok := range tokens[i] {
				pairTagCall[[2]string{tags[a], tok}]++
			}
		}
	}

	// Enumerate every marginal pair in deterministic order, keep those that
	// pass the cutoffs. Cross products are small: 9 tags × 4 roles × capped
	// tokens.
	var out []Association
	consider := func(kind AssocKind, a, b string, ca, cb int) {
		cnt := 0
		switch kind {
		case TagTag:
			cnt = pairTagTag[[2]string{a, b}]
		case TagRole:
			cnt = pairTagRole[[2]string{a, b}]
		case TagCall:
			cnt = pairTagCall[[2]string{a, b}]
		}
		expected := float64(ca) * float64(cb) / float64(n)
		pmi := math.Inf(-1)
		if cnt > 0 {
			pmi = math.Log(float64(n) * float64(cnt) / (float64(ca) * float64(cb)))
		}
		switch {
		case cnt >= opt.MinPairSupport && pmi >= minAbsPMI:
			// Positive: co-occur at least twice the chance rate, with support.
		case expected >= opt.MinExpected && float64(cnt) <= expected/2:
			// Negative: both marginals large enough that absence (or rarity)
			// is informative — Count <= Expected/2 is PMI <= -ln 2.
		default:
			return
		}
		out = append(out, Association{Kind: kind, A: a, B: b, Count: cnt, Expected: expected, PMI: pmi})
	}

	tags := sortedCountKeys(tagCount)
	roles := sortedCountKeys(roleCount)
	for i, a := range tags {
		for _, b := range tags[i+1:] {
			consider(TagTag, a, b, tagCount[a], tagCount[b])
		}
	}
	for _, a := range tags {
		for _, r := range roles {
			consider(TagRole, a, r, tagCount[a], roleCount[r])
		}
	}
	for _, a := range tags {
		for _, tok := range sortedCountKeys(tokenCount) {
			if df := tokenCount[tok]; df < 2 || df > opt.MaxCallTokenDF {
				continue // singleton or corpus idiom: not association material
			}
			consider(TagCall, a, tok, tagCount[a], tokenCount[tok])
		}
	}

	// Positives by PMI descending, then negatives by PMI ascending (never
	// first); float ties break on (Kind, A, B), a total order.
	sort.SliceStable(out, func(x, y int) bool {
		px, py := out[x].PMI >= minAbsPMI, out[y].PMI >= minAbsPMI
		if px != py {
			return px
		}
		if out[x].PMI != out[y].PMI {
			if px {
				return out[x].PMI > out[y].PMI
			}
			return out[x].PMI < out[y].PMI
		}
		if out[x].Kind != out[y].Kind {
			return out[x].Kind < out[y].Kind
		}
		if out[x].A != out[y].A {
			return out[x].A < out[y].A
		}
		return out[x].B < out[y].B
	})
	return out
}

func sortedUniqueTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	set := make(map[string]bool, len(tags))
	for _, t := range tags {
		set[t] = true
	}
	return sortedStrings(set)
}

func sortedCountKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

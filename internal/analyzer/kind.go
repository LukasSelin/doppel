package analyzer

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// Pair kinds: what a pair (or a family) is, stated so the reader knows why
// it exists. Two classes of finding used to crowd the top of every wide
// corpus without the report being able to say anything about them:
//
//   - interface implementations — a Validate per provider, a Go per pool
//     variant — near-identical by construction and unactionable. Proving
//     interface satisfaction needs go/types, which is out of proportion here;
//     the honest middle is a naming rule: same method name, same signature,
//     different receivers.
//   - diverged copies — hugo's evalCall beside evalCallOld, prometheus's v1
//     and v2 scrape appenders — real findings that are forks, not merge
//     candidates. The rule is code-shape already high plus names that agree
//     once version markers are stripped (stem.go), in the same or a sibling
//     package.
//
// A kind annotates. It never filters a pair, never enters ranking, and is
// not stored in the snapshot (hook digests never see it) — the same contract
// as culture, habitat and profile notes.
const (
	KindInterfaceImpl = "interface implementations"
	KindFork          = "diverged copy"
)

// ForkShapeFloor is the code-shape a pair must reach before two stem-sharing
// names are read as a diverged copy. It equals the family stage's edge cut so
// the two views agree on what "alike enough" means; family_test pins that.
const ForkShapeFloor = 0.60

// Package relations a kind reports.
const (
	RelationSamePackage = "same package"
	RelationSiblings    = "sibling packages"
	RelationUnrelated   = "unrelated packages"
)

// KindNote is one pair's (or family's) kind with the facts the label renders.
type KindNote struct {
	Kind      string   // KindInterfaceImpl or KindFork
	Method    string   // interface-impl: the shared method name; fork: the shared stem
	Signature string   // interface-impl: the identical signature text
	Receivers []string // interface-impl: the receiver types as written, member order
	Names     []string // fork: the differing unit names, member order
	Packages  []string // distinct package names, member order
	Relation  string   // RelationSamePackage, RelationSiblings or RelationUnrelated
}

// ClassifyPair labels a pair, or returns nil. Fork is tried first: when both
// rules hold — scrapeLoopAppender.append beside scrapeLoopAppenderV2.append
// is an interface implementation and a fork — the fork is the more specific
// claim, and the one a reader acts on.
func ClassifyPair(a, b parser.CodeUnit, score float64) *KindNote {
	return ClassifyPairWith(a, b, score, ForkShapeFloor)
}

// ClassifyPairWith is ClassifyPair under an explicit fork floor — the
// calibrated code-shape threshold when --calibrate is on, so the fork rule
// and the family edge cut move together.
func ClassifyPairWith(a, b parser.CodeUnit, score, forkFloor float64) *KindNote {
	if k := forkAt(a, b, score, forkFloor); k != nil {
		return k
	}
	return InterfaceImpl(a, b)
}

// InterfaceImpl labels two methods with the same name and the same signature
// on different receiver types. It fires for every package relation; the
// label states which, because the same method name across unrelated
// packages is a weaker claim than across siblings.
func InterfaceImpl(a, b parser.CodeUnit) *KindNote {
	if a.ReceiverType == "" || b.ReceiverType == "" {
		return nil
	}
	if parser.MethodName(a) != parser.MethodName(b) {
		return nil
	}
	// Same receiver name in the same package is one type — two methods on
	// it are overloads that Go does not have, so this is never reached for
	// real code, but pointer and value receivers normalize to one name and
	// must not pass. In different packages, two types named driver are two
	// types: moby's ipvlan and macvlan drivers each implement Join on their
	// own *driver.
	if bareReceiver(a.ReceiverType) == bareReceiver(b.ReceiverType) && a.Package == b.Package {
		return nil
	}
	if a.Signature == "" || a.Signature != b.Signature {
		return nil
	}
	return &KindNote{
		Kind:      KindInterfaceImpl,
		Method:    parser.MethodName(a),
		Signature: a.Signature,
		Receivers: []string{a.ReceiverType, b.ReceiverType},
		Packages:  distinct(a.Package, b.Package),
		Relation:  relation(a, b),
	}
}

// Fork labels two alike bodies whose names share a stem, in the same or a
// sibling package. Methods fork on either axis: the same method on receivers
// that share a stem (*Appender.append / *AppenderV2.append), or stem-sharing
// methods on one receiver (*state.evalCall / *state.evalCallOld).
func Fork(a, b parser.CodeUnit, score float64) *KindNote {
	return forkAt(a, b, score, ForkShapeFloor)
}

func forkAt(a, b parser.CodeUnit, score, floor float64) *KindNote {
	if score < floor || a.Name == b.Name {
		return nil
	}
	rel := relation(a, b)
	if rel == RelationUnrelated {
		return nil
	}
	stem, ok := forkStem(a, b)
	if !ok {
		return nil
	}
	return &KindNote{
		Kind:     KindFork,
		Method:   stem,
		Names:    []string{a.Name, b.Name},
		Packages: distinct(a.Package, b.Package),
		Relation: rel,
	}
}

// forkStem finds the stem two units share, on whichever naming axis differs.
func forkStem(a, b parser.CodeUnit) (string, bool) {
	switch {
	case a.ReceiverType == "" && b.ReceiverType == "":
		return stemPair(a.Name, b.Name)
	case a.ReceiverType == "" || b.ReceiverType == "":
		return "", false
	}
	ra, rb := bareReceiver(a.ReceiverType), bareReceiver(b.ReceiverType)
	ma, mb := parser.MethodName(a), parser.MethodName(b)
	if ma == mb {
		return stemPair(ra, rb)
	}
	if ra == rb {
		return stemPair(ma, mb)
	}
	return "", false
}

// ClassifyFamily labels a family whose every member pair satisfies one rule:
// a fork when all members share one stem, else an interface implementation
// when all share a method name and signature on pairwise-distinct receivers.
// minEdge is the family's code-shape guarantee, which stands in for each
// pair's score.
func ClassifyFamily(units []parser.CodeUnit, members []int, minEdge float64) *KindNote {
	if len(members) < 2 {
		return nil
	}
	for _, m := range members {
		if m < 0 || m >= len(units) {
			return nil
		}
	}
	if k := familyFork(units, members, minEdge); k != nil {
		return k
	}
	return familyInterfaceImpl(units, members)
}

func familyFork(units []parser.CodeUnit, members []int, minEdge float64) *KindNote {
	var stem string
	rel := RelationSamePackage
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			k := Fork(units[members[i]], units[members[j]], minEdge)
			if k == nil {
				return nil
			}
			if stem == "" {
				stem = k.Method
			} else if k.Method != stem {
				return nil
			}
			rel = widen(rel, k.Relation)
		}
	}
	note := &KindNote{Kind: KindFork, Method: stem, Relation: rel}
	for _, m := range members {
		note.Names = append(note.Names, units[m].Name)
		note.Packages = appendDistinct(note.Packages, units[m].Package)
	}
	return note
}

func familyInterfaceImpl(units []parser.CodeUnit, members []int) *KindNote {
	first := units[members[0]]
	note := &KindNote{Kind: KindInterfaceImpl, Relation: RelationSamePackage}
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			k := InterfaceImpl(units[members[i]], units[members[j]])
			if k == nil {
				return nil
			}
			note.Relation = widen(note.Relation, k.Relation)
		}
	}
	note.Method = parser.MethodName(first)
	note.Signature = first.Signature
	for _, m := range members {
		note.Receivers = append(note.Receivers, units[m].ReceiverType)
		note.Packages = appendDistinct(note.Packages, units[m].Package)
	}
	return note
}

// widen keeps the loosest relation seen across a family's pairs.
func widen(cur, next string) string {
	rank := map[string]int{RelationSamePackage: 0, RelationSiblings: 1, RelationUnrelated: 2}
	if rank[next] > rank[cur] {
		return next
	}
	return cur
}

// relation places two units: same package (same clause in the same
// directory — two directories can share a clause), sibling packages (their
// directories share a parent), or unrelated. Units without a File both sit
// in ".", so fixtures compare by package alone.
func relation(a, b parser.CodeUnit) string {
	da, db := unitDir(a), unitDir(b)
	if a.Package == b.Package && da == db {
		return RelationSamePackage
	}
	if path.Dir(da) == path.Dir(db) {
		return RelationSiblings
	}
	return RelationUnrelated
}

func unitDir(u parser.CodeUnit) string {
	return path.Dir(filepath.ToSlash(u.File))
}

// bareReceiver strips the pointer star and any generic instantiation, so
// *Pool[T] and Pool[U] compare as Pool.
func bareReceiver(r string) string {
	r = ontology.NormalizeReceiver(r)
	if i := strings.IndexByte(r, '['); i >= 0 {
		r = r[:i]
	}
	return r
}

func distinct(a, b string) []string {
	if a == b {
		return []string{a}
	}
	return []string{a, b}
}

func appendDistinct(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

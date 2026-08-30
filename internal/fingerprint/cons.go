package fingerprint

import "go/ast"

// Cons computes the hash-consing signature of every node in a function body:
// one structural hash per node, identifying the exact subtree rooted there.
//
// The recurrence is a plain bottom-up Merkle hash — hash(v) = fnv(label(v),
// len(children), hash(child) for each child, in source order) — evaluated
// once per node on the way back up, exactly the traversal WLBag already uses
// (push a frame on entry, pop and compute on the matching nil callback).
// label(v) is wlLabel0(v): the same node-kind vocabulary WL's label_0 uses,
// reused rather than reinvented, because "what kind of node is this" is one
// question this package should only answer once.
//
// # This is not WL
//
// WL sorts each node's children into a multiset before hashing, because its
// job is a *shape* signature that should not care whether two structurally
// interchangeable branches were written in one order or the other. A
// hash-cons answers a stricter question — "is this literally the same
// subtree" — so child order is kept: `a - b` and `b - a` must hash
// differently even though they are the same shape under WL. That is also
// why there is only one round here where WL has several: a hash-cons wants
// the whole subtree collapsed into one identity, not a family of
// bounded-radius approximations to it.
//
// # It is meant for the canonical tree
//
// As with WLBag, the caller passes canon's canonical form, not the parsed
// declaration: two hand-written copies of the same logic should cons
// together, and canonicalization is what removes the accidental choices
// (bound-identifier naming, and everything else canon normalizes) that would
// otherwise keep them apart. Only the body is walked — the signature is not
// shape, and has its own Fingerprint.Types component. A nil declaration or
// one without a body yields no hashes, mirroring WLBag's rule.
//
// # Cost and collisions
//
// One post-order pass, one map-free arena exactly like WLBag's, so a
// function costs one slice no matter how deep it nests. Two different
// subtrees can hash identically on an FNV-64 collision, which conflates
// them into one entry in the corpus-wide count ConsCorpus keeps — the same
// collision budget WL already runs on, accepted for the same reason: a
// cryptographic hash would be disproportionate for a compression estimate.
func Cons(fd *ast.FuncDecl) []uint64 {
	if fd == nil || fd.Body == nil {
		return nil
	}
	var hashes []uint64
	var frames []consFrame
	var kids []uint64

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if n != nil {
			frames = append(frames, consFrame{label: wlLabel0(n), start: len(kids)})
			return true
		}
		last := len(frames) - 1
		if last < 0 {
			return false
		}
		fr := frames[last]
		frames = frames[:last]

		children := kids[fr.start:]
		h := consHash(fr.label, children)
		// Same arena-reuse trick as WLBag: the children have been folded into
		// h, so their slots are free and this node takes the first as its own.
		kids = append(kids[:fr.start], h)
		hashes = append(hashes, h)
		return false
	})
	return hashes
}

// consFrame is one open node during the walk: its label, captured at push
// time because ast.Inspect's matching pop callback receives nil rather than
// the node, and where its children's hashes start in the shared arena.
type consFrame struct {
	label uint64
	start int
}

// consHash serialises one node: its label, its child count, then each
// child's hash in source order. The count leads the children for the same
// reason wlHash writes it first — without it, a different (children)
// grouping could serialise to the same bytes as a different split.
func consHash(label uint64, children []uint64) uint64 {
	h := fnvU64(fnvOffset64, label)
	h = fnvU64(h, uint64(len(children)))
	for _, c := range children {
		h = fnvU64(h, c)
	}
	return h
}

// ConsStats is the corpus-wide result of hash-consing every canonical
// function body: how many nodes those bodies have in total, and how many
// distinct subtree shapes — by Cons's structural hash — those nodes reduce
// to.
type ConsStats struct {
	TotalNodes     int
	UniqueSubtrees int
}

// Ratio is the compression ratio: how many canonical AST nodes exist for
// every distinct subtree shape among them. It is always >= 1.0 for a
// non-empty corpus, since a subtree that occurs once still counts as one
// node over one shape; it grows with how much of the corpus's syntax repeats
// verbatim (after canonicalization) somewhere else in it. An empty corpus —
// no bodies at all — reports 0 rather than dividing by zero.
func (s ConsStats) Ratio() float64 {
	if s.UniqueSubtrees == 0 {
		return 0
	}
	return float64(s.TotalNodes) / float64(s.UniqueSubtrees)
}

// ConsCorpus hash-conses every body's subtrees and reduces the result to the
// two totals Ratio needs. Pass one canonical *ast.FuncDecl per function, in
// any order and including nil ones (a declaration with no body contributes
// nothing) — the result depends only on the multiset of hashes, never on
// their order, so the corpus-wide dedup is a plain set-size count and needs
// no sorting.
func ConsCorpus(bodies []*ast.FuncDecl) ConsStats {
	seen := make(map[uint64]struct{})
	total := 0
	for _, fd := range bodies {
		for _, h := range Cons(fd) {
			total++
			seen[h] = struct{}{}
		}
	}
	return ConsStats{TotalNodes: total, UniqueSubtrees: len(seen)}
}

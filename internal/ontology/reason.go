package ontology

import "sort"

// Match is one term-to-term pairing chosen by SetRelatedness, with the ancestor
// that explains it. Reported so the evidence in a report can say why two
// different tags scored, not just that they did.
type Match struct {
	A     string
	B     string
	Score float64
	LCA   TermID
}

// Exact reports whether the two terms are the same, in which case the match
// needs no explaining and is already covered by the shared-terms evidence.
func (m Match) Exact() bool { return m.A == m.B }

// Ancestors returns a term's ancestors, nearest first, excluding the term
// itself.
func (o *Ontology) Ancestors(id TermID) []TermID {
	chain := o.chain(id)
	if len(chain) == 0 {
		return nil
	}
	return chain[1:]
}

// chain returns the term itself followed by its ancestors, nearest first. The
// step cap guards against a cycle in a malformed table.
func (o *Ontology) chain(id TermID) []TermID {
	if _, ok := o.terms[id]; !ok {
		return nil
	}
	out := []TermID{id}
	for cur := id; len(out) <= len(o.order); {
		t := o.terms[cur]
		if t.Parent == "" {
			break
		}
		if _, ok := o.terms[t.Parent]; !ok {
			break
		}
		out = append(out, t.Parent)
		cur = t.Parent
	}
	return out
}

// IsA reports whether child is a descendant of ancestor. A term is not its own
// ancestor.
func (o *Ontology) IsA(child, ancestor TermID) bool {
	for _, id := range o.Ancestors(child) {
		if id == ancestor {
			return true
		}
	}
	return false
}

// Depth returns the distance from the root of a term's Kind. The root is 0.
func (o *Ontology) Depth(id TermID) int {
	d, ok := o.depth[id]
	if !ok {
		return -1
	}
	return d
}

// LCA returns the least common ancestor of two terms, which is a itself when a
// and b are equal.
func (o *Ontology) LCA(a, b TermID) (TermID, bool) {
	ta, okA := o.terms[a]
	tb, okB := o.terms[b]
	if !okA || !okB || ta.Kind != tb.Kind {
		return "", false
	}
	seen := make(map[TermID]bool)
	for _, id := range o.chain(a) {
		seen[id] = true
	}
	for _, id := range o.chain(b) {
		if seen[id] {
			return id, true
		}
	}
	return "", false
}

// Relatedness scores two terms by Wu-Palmer similarity,
// 2*depth(LCA) / (depth(a)+depth(b)), on a 0.0-1.0 scale.
//
// Three guards shape the result:
//
// Equal terms return 1.0 before any lookup, so a tag the ontology has not
// learned about yet still matches itself rather than silently scoring zero
// against its own twin.
//
// Terms in different Kinds, or terms that do not exist, return 0.0. Comparing a
// role to a concept is a bug in the caller, not a weak similarity.
//
// An LCA at the root returns 0.0. Under this depth convention that is what the
// arithmetic already produces, since the root sits at depth 0, and the check is
// therefore redundant today. It is written out anyway because the textbook
// statement of Wu-Palmer puts the root at depth 1: anyone "correcting" the
// depths to match would otherwise turn every cross-branch pair from 0.00 into a
// nonzero floor and quietly inflate every structural score in the report.
func (o *Ontology) Relatedness(a, b TermID) float64 {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	lca, ok := o.LCA(a, b)
	if !ok {
		return 0
	}
	dl := o.Depth(lca)
	if dl <= 0 {
		return 0
	}
	da, db := o.Depth(a), o.Depth(b)
	if da <= 0 || db <= 0 {
		return 0
	}
	return 2 * float64(dl) / float64(da+db)
}

// SetRelatedness scores how alike two sets of terms are, generalizing the exact
// intersection ratio the comparator used to compute. It returns the score and
// the pairings behind it.
//
// Terms are matched by global-descending greedy: every candidate pairing is
// scored, all of them are sorted by score and then by term, and the list is
// walked once consuming pairings whose endpoints are both still free. The score
// is the sum of the chosen pairings over the size of the larger set.
//
// The obvious alternative — walk the smaller set, give each term its best
// partner — is wrong, and quietly so. It lets a merely related term consume an
// exact match: {caching, transaction} against {mapping, transaction} scores
// 0.33 that way, below the 0.50 exact matching already gives, because caching
// takes transaction (0.67) before transaction can claim itself. That makes the
// function asymmetric in its arguments and lets a pair lose merge-worthiness by
// gaining a hierarchy. Sorting globally instead means exact matches, worth 1.0
// and never in contention with each other across deduplicated sets, are all
// consumed first, so the result is never below the old exact ratio.
//
// The sort tie-break on term names is load-bearing: cross-branch pairings all
// score exactly 0.0, so ties are the common case, and without a total order the
// chosen pairings — and the evidence lines naming them — would vary run to run.
//
// Two empty sets score 0.0, not 1.0. Beyond being the right reading (two
// functions that carry no tags have not been shown to agree on anything), the
// guard has to run before the division, which would otherwise be 0/0 and return
// a NaN that slips past both the clamp on the composite score and every
// threshold comparison downstream.
func (o *Ontology) SetRelatedness(a, b []string) (float64, []Match) {
	as, bs := sortedUnique(a), sortedUnique(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0, nil
	}

	type candidate struct {
		i, j  int
		score float64
	}
	var candidates []candidate
	for i := range as {
		for j := range bs {
			if s := o.Relatedness(TermID(as[i]), TermID(bs[j])); s > 0 {
				candidates = append(candidates, candidate{i: i, j: j, score: s})
			}
		}
	}
	// as and bs are sorted, so ordering by index is ordering by term name.
	sort.SliceStable(candidates, func(x, y int) bool {
		if candidates[x].score != candidates[y].score {
			return candidates[x].score > candidates[y].score
		}
		if candidates[x].i != candidates[y].i {
			return candidates[x].i < candidates[y].i
		}
		return candidates[x].j < candidates[y].j
	})

	usedA := make([]bool, len(as))
	usedB := make([]bool, len(bs))
	var sum float64
	var matched []Match
	for _, c := range candidates {
		if usedA[c.i] || usedB[c.j] {
			continue
		}
		usedA[c.i], usedB[c.j] = true, true
		sum += c.score
		lca, _ := o.LCA(TermID(as[c.i]), TermID(bs[c.j]))
		matched = append(matched, Match{A: as[c.i], B: bs[c.j], Score: c.score, LCA: lca})
	}

	denom := len(as)
	if len(bs) > denom {
		denom = len(bs)
	}
	return sum / float64(denom), matched
}

// BestMatch returns the highest score among a set of matches, or 0 for none.
// Used to judge evidence quality independently of set size: one exact match
// among three tags is strong evidence even though the aggregate ratio is 0.33.
func BestMatch(matches []Match) float64 {
	var best float64
	for _, m := range matches {
		if m.Score > best {
			best = m.Score
		}
	}
	return best
}

func sortedUnique(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

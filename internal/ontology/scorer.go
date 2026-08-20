package ontology

import "sort"

// Scorer scores term and set relatedness, optionally weighted by corpus-derived
// information content.
//
// With a nil IC it delegates, literally, to the Ontology's own Wu-Palmer
// methods — byte-identical behavior by construction, so every test pinning
// those methods also pins this path. With an IC it switches to Lin similarity
// and information-weighted set matching, where a ubiquitous tag contributes
// little and a rare one a lot.
type Scorer struct {
	o  *Ontology
	ic *IC
}

// NewScorer pairs an ontology with an optional IC table. A nil ic yields the
// corpus-independent scorer.
func NewScorer(o *Ontology, ic *IC) *Scorer { return &Scorer{o: o, ic: ic} }

// Weighted reports whether this scorer carries corpus weights.
func (s *Scorer) Weighted() bool { return s.ic != nil }

// Relatedness scores two terms. Without IC: Wu-Palmer. With IC: Lin similarity,
// 2·IC(LCS) / (IC(a) + IC(b)) — how much of the two terms' combined information
// their most specific common ancestor carries.
//
// Guard order matches the Wu-Palmer path and is load-bearing: a == b returns
// 1.0 before any lookup so an unknown term still matches its own twin; unknown
// or cross-kind pairs are 0; an LCS at the root is 0 because IC(root) is
// exactly zero, so cross-branch pairs stay at zero just as under Wu-Palmer.
func (s *Scorer) Relatedness(a, b TermID) float64 {
	if s.ic == nil {
		return s.o.Relatedness(a, b)
	}
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	lcs, ok := s.o.LCA(a, b)
	if !ok {
		return 0
	}
	shared := s.ic.Of(lcs)
	if shared <= 0 {
		return 0
	}
	denom := s.ic.Of(a) + s.ic.Of(b)
	if denom <= 0 {
		return 0
	}
	return 2 * shared / denom
}

// SetRelatedness scores how alike two sets of terms are. Without IC it is the
// Ontology's soft overlap. With IC, the objective collapses to something
// pleasingly direct: under Lin, a matched pair's similarity times its average
// weight equals IC(LCS) exactly — Lin's denominator cancels the pair weight —
// so the score is
//
//	Σ over matched pairs of IC(LCS)  /  max(Σ IC over a, Σ IC over b)
//
// the fraction of the larger side's information content that the two sets
// share. Sharing only a near-universal tag now scores close to zero; sharing a
// rare one scores close to one. With uniform weights this reduces to the
// unweighted soft overlap, which is why the nil-IC path can delegate.
//
// The matcher is the same greedy as the unweighted path but sorted by
// contribution (IC of the pair's LCS) rather than by similarity. Under IC the
// two orders genuinely differ — a pair can be more *similar* while sharing a
// less *informative* ancestor — and sorting by contribution is what stays
// optimal: IC decomposes laminarly over the tree, so consuming
// deepest-information ancestors first achieves every subtree's matching bound
// at once (re-verified against the exhaustive oracle in tests). Exact matches
// carry their leaf's full IC, which strictly exceeds any rival pairing for the
// same term, so they are always consumed and the weighted score never falls
// below the weighted exact-share.
//
// Matches are reported in consumption order — descending contribution — and
// Match.Score carries the Lin similarity, since that is the 0-to-1 number the
// evidence lines render. Two empty sets score 0, guarded before the division,
// same convention and same NaN hazard as the unweighted path.
func (s *Scorer) SetRelatedness(a, b []string) (float64, []Match) {
	if s.ic == nil {
		return s.o.SetRelatedness(a, b)
	}
	as, bs := sortedUnique(a), sortedUnique(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0, nil
	}

	type candidate struct {
		i, j    int
		contrib float64 // IC of the pair's most specific common ancestor
		sim     float64 // Lin similarity, for the Match record
		lcs     TermID
	}
	var candidates []candidate
	for i := range as {
		for j := range bs {
			ta, tb := TermID(as[i]), TermID(bs[j])
			contrib, lcs, ok := s.pairContribution(ta, tb)
			if !ok || contrib <= 0 {
				continue
			}
			candidates = append(candidates, candidate{
				i: i, j: j,
				contrib: contrib,
				sim:     s.Relatedness(ta, tb),
				lcs:     lcs,
			})
		}
	}
	// as and bs are sorted, so ordering by index is ordering by term name; the
	// tie-break keeps evidence lines byte-stable across runs.
	sort.SliceStable(candidates, func(x, y int) bool {
		if candidates[x].contrib != candidates[y].contrib {
			return candidates[x].contrib > candidates[y].contrib
		}
		if candidates[x].i != candidates[y].i {
			return candidates[x].i < candidates[y].i
		}
		return candidates[x].j < candidates[y].j
	})

	usedA := make([]bool, len(as))
	usedB := make([]bool, len(bs))
	var shared float64
	var matched []Match
	for _, c := range candidates {
		if usedA[c.i] || usedB[c.j] {
			continue
		}
		usedA[c.i], usedB[c.j] = true, true
		shared += c.contrib
		matched = append(matched, Match{A: as[c.i], B: bs[c.j], Score: c.sim, LCA: c.lcs})
	}

	var sumA, sumB float64
	for _, t := range as {
		sumA += s.ic.Of(TermID(t))
	}
	for _, t := range bs {
		sumB += s.ic.Of(TermID(t))
	}
	denom := sumA
	if sumB > denom {
		denom = sumB
	}
	if denom <= 0 {
		return 0, matched
	}
	return shared / denom, matched
}

// pairContribution returns the information the two terms share: IC of their
// most specific common ancestor. Equal terms share their full IC — including
// terms the taxonomy does not know, which is the weighted analogue of the
// unknown-term self-match guard in Relatedness.
func (s *Scorer) pairContribution(a, b TermID) (float64, TermID, bool) {
	if a == b {
		lcs := a
		if _, known := s.o.Get(a); !known {
			lcs = ""
		}
		return s.ic.Of(a), lcs, true
	}
	lcs, ok := s.o.LCA(a, b)
	if !ok {
		return 0, "", false
	}
	return s.ic.Of(lcs), lcs, true
}

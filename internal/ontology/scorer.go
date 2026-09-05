package ontology

import (
	"cmp"
	"slices"
)

// Scorer scores term and set relatedness, optionally weighted by corpus-derived
// information content.
//
// With a nil IC it delegates, literally, to the Ontology's own Wu-Palmer
// methods — byte-identical behavior by construction, so every test pinning
// those methods also pins this path. With an IC it switches to Lin similarity
// and information-weighted set matching, where a ubiquitous tag contributes
// little and a rare one a lot.
type Scorer struct {
	o     *Ontology
	ic    *IC
	vocab *Vocabulary // nil: the feature view is not measured

	// scratch is per-goroutine reusable buffers, nil on every scorer the
	// pipeline builds. A nil scratch makes matchShared allocate exactly as it
	// always did, which is what keeps the shared scorer safe to read from
	// several goroutines at once; Fork is how a worker gets its own.
	scratch *scorerScratch
}

// scorerScratch is matchShared's working memory. Only the buffers that stay
// inside the call are here: the []Match it returns escapes into the evidence a
// caller keeps, so it is allocated fresh every time and always will be.
type scorerScratch struct {
	candidates []matchCandidate
	usedA      []bool
	usedB      []bool
}

// matchCandidate is one possible pairing, scored before the greedy consumes it.
type matchCandidate struct {
	i, j    int
	contrib float64 // IC of the pair's most specific common ancestor
	sim     float64 // Lin similarity, for the Match record
	lcs     TermID
}

// Fork returns a scorer that shares this one's immutable tables — the
// ontology, the IC, the interned vocabulary — and owns its own mutable
// scratch, which is what makes it safe to use from one goroutine while another
// uses the original.
//
// It is the whole concurrency contract of this package: a *Scorer with no
// scratch and a *Vocabulary that is never profiled are read-only and shareable;
// everything mutable is reached only through a Fork. Scoring is otherwise
// unchanged, so a forked scorer produces bit-identical numbers — there is no
// accumulation across calls for a buffer to carry between them.
func (s *Scorer) Fork() *Scorer {
	cp := *s
	cp.scratch = &scorerScratch{}
	if s.vocab != nil {
		cp.vocab = s.vocab.fork()
	}
	return &cp
}

// boolBuf resizes a scratch bool slice to n and zeroes it.
func boolBuf(buf *[]bool, n int) []bool {
	if cap(*buf) < n {
		*buf = make([]bool, n)
	}
	b := (*buf)[:n]
	clear(b)
	return b
}

// NewScorer pairs an ontology with an optional IC table. A nil ic yields the
// corpus-independent scorer.
func NewScorer(o *Ontology, ic *IC) *Scorer { return &Scorer{o: o, ic: ic} }

// Weighted reports whether this scorer carries corpus weights.
func (s *Scorer) Weighted() bool { return s.ic != nil }

// Ontology returns the vocabulary this scorer reasons over. The comparator
// reads its relation weights through this rather than through the package
// default, which is what lets the bench harness score pairs under overridden
// weights (WithWeights) without touching production state.
func (s *Scorer) Ontology() *Ontology { return s.o }

// WithVocabulary returns a copy of the scorer carrying the learned concept
// vocabularies, which is what FeatureRelatednessW reads. NewScorer is left as
// it is because retrieval builds its own scorer and has no use for the table:
// the concept channel is information mass, and stays so.
func (s *Scorer) WithVocabulary(v *Vocabulary) *Scorer {
	cp := *s
	cp.vocab = v
	return &cp
}

// Vocabulary returns the table WithVocabulary attached, or nil.
func (s *Scorer) Vocabulary() *Vocabulary { return s.vocab }

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
	shared, matched := s.matchShared(as, bs)

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

// SharedInformation returns the un-normalized numerator of the weighted
// SetRelatedness: Σ IC(LCS) over the greedy contribution-ordered matching of
// the two term sets, plus the matches themselves. It is the "how much
// information do these sets share" quantity, in nats, before any division by
// either side's total — which is what candidate retrieval wants: a pair
// sharing one rare tag carries more evidence than a pair sharing one
// ubiquitous tag, even though both normalize to the same set relatedness.
// A nil-IC scorer has no information table and returns (0, nil).
func (s *Scorer) SharedInformation(a, b []string) (float64, []Match) {
	if s.ic == nil {
		return 0, nil
	}
	as, bs := sortedUnique(a), sortedUnique(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0, nil
	}
	return s.matchShared(as, bs)
}

// matchShared runs the contribution-ordered greedy matching over two
// sorted-unique term sets and returns the total shared information and the
// matches in consumption order. Both SetRelatedness and SharedInformation
// build on it, so the matching (and its tie-breaks) cannot drift between the
// normalized and raw views.
func (s *Scorer) matchShared(as, bs []string) (float64, []Match) {
	var candidates []matchCandidate
	if s.scratch != nil {
		candidates = s.scratch.candidates[:0]
	}
	for i := range as {
		for j := range bs {
			ta, tb := TermID(as[i]), TermID(bs[j])
			contrib, lcs, ok := s.pairContribution(ta, tb)
			if !ok || contrib <= 0 {
				continue
			}
			candidates = append(candidates, matchCandidate{
				i: i, j: j,
				contrib: contrib,
				sim:     s.Relatedness(ta, tb),
				lcs:     lcs,
			})
		}
	}
	// as and bs are sorted, so ordering by index is ordering by term name; the
	// tie-break keeps evidence lines byte-stable across runs.
	// A typed comparator rather than sort.SliceStable, whose reflect-based
	// swapper is an allocation per call and showed up as such in the heap
	// profile. Same order, by construction: the two predicates are the same
	// three comparisons in the same sequence.
	slices.SortStableFunc(candidates, func(x, y matchCandidate) int {
		if x.contrib != y.contrib {
			return cmp.Compare(y.contrib, x.contrib) // descending
		}
		if x.i != y.i {
			return cmp.Compare(x.i, y.i)
		}
		return cmp.Compare(x.j, y.j)
	})

	var usedA, usedB []bool
	if s.scratch != nil {
		usedA = boolBuf(&s.scratch.usedA, len(as))
		usedB = boolBuf(&s.scratch.usedB, len(bs))
	} else {
		usedA = make([]bool, len(as))
		usedB = make([]bool, len(bs))
	}
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
	if s.scratch != nil {
		s.scratch.candidates = candidates // keep the grown capacity
	}
	return shared, matched
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

// WeightedTerm is a term asserted with a confidence, which is what a learned
// lexicon produces instead of a bare tag list. Weight is in (0,1]; the
// string-set methods above are the special case where every weight is 1.
type WeightedTerm struct {
	ID     TermID
	Weight float64
}

// SetRelatednessW is SetRelatedness over graded assertions:
//
//	Σ over matched pairs of min(w_a, w_b)·IC(LCS)  /  max(Σ w·IC over a, Σ w·IC over b)
//
// min rather than a product or a mean because sharing is bounded by the weaker
// of the two claims: a concept one side barely carries cannot be strong shared
// evidence however sure the other side is.
//
// The *matching* is deliberately left un-weighted — it is the same
// contribution-ordered greedy over the same term IDs, and weights only rescale
// the pairs it chose. That is not a shortcut: the greedy's optimality rests on
// IC decomposing laminarly over the tree, an argument confidences break (a
// down-weighted exact match can be worth less than two related pairs, which is
// the classic case where greedy matching is not optimal). Keeping the choice of
// pairs weight-free preserves the property the exhaustive oracle test verifies,
// and confines confidence to what it honestly is: how much the chosen evidence
// counts.
func (s *Scorer) SetRelatednessW(a, b []WeightedTerm) (float64, []Match) {
	if s.ic == nil {
		return s.o.SetRelatedness(termIDs(a), termIDs(b))
	}
	as, aw := split(a)
	bs, bw := split(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0, nil
	}
	_, matched := s.matchShared(as, bs)
	shared := s.weighAll(matched, as, aw, bs, bw)

	denom := s.weighedTotal(as, aw)
	if sumB := s.weighedTotal(bs, bw); sumB > denom {
		denom = sumB
	}
	if denom <= 0 {
		return 0, matched
	}
	return shared / denom, matched
}

// SharedInformationW is the un-normalized numerator of SetRelatednessW: the
// graded shared information in nats, which is what candidate retrieval wants.
func (s *Scorer) SharedInformationW(a, b []WeightedTerm) (float64, []Match) {
	if s.ic == nil {
		return 0, nil
	}
	as, aw := split(a)
	bs, bw := split(b)
	if len(as) == 0 || len(bs) == 0 {
		return 0, nil
	}
	_, matched := s.matchShared(as, bs)
	return s.weighAll(matched, as, aw, bs, bw), matched
}

// weighAll rescales a matching's contributions by the weaker side's weight.
//
// Weights arrive as each side's split, looked up by ID rather than indexed by
// the Match, because a Match names terms and not positions — that is what makes
// it renderable evidence, and re-deriving the position here is a binary search
// over a handful of sorted strings.
func (s *Scorer) weighAll(matched []Match, as []string, aw []float64, bs []string, bw []float64) float64 {
	var shared float64
	for _, m := range matched { // consumption order: the addition order is fixed
		contrib, _, ok := s.pairContribution(TermID(m.A), TermID(m.B))
		if !ok || contrib <= 0 {
			continue
		}
		w := weightOf(as, aw, m.A)
		if wb := weightOf(bs, bw, m.B); wb < w {
			w = wb
		}
		shared += w * contrib
	}
	return shared
}

// weighedTotal is one side's own information, each term counted only as far as
// it is asserted.
func (s *Scorer) weighedTotal(terms []string, w []float64) float64 {
	var sum float64
	for i, t := range terms { // sorted: the addition order is fixed
		sum += w[i] * s.ic.Of(TermID(t))
	}
	return sum
}

// split separates graded terms into the sorted-unique ID slice the matcher
// works on and that slice's weights, positionally aligned. A term asserted
// twice keeps its strongest weight, which cannot happen from the lexicon but
// must not silently pick one at random if it ever does. Empty IDs are dropped,
// which was sortedUnique's rule and has to survive replacing it.
//
// Two parallel slices rather than a map, because this runs up to eight times
// per compared pair — three SetRelatednessW, the feature profile, and
// retrieval's SharedInformationW, each over both sides — and the map form
// allocated two maps per call (its own and sortedUnique's), measured as the
// single largest allocation site in the tool at 306MB of a 3.26GB moby run.
// Every reader wants either a positional walk (weighedTotal, Vocabulary.
// profile) or a lookup over a handful of sorted entries (weighAll), and
// neither needs hashing.
//
// The common input is already sorted and unique — CodeUnit.Concepts is held
// ascending by ID, and concepter.Graded preserves that — so the ordered case
// is filled in one pass with no sort at all. The general path exists because
// nothing in the type says so.
func split(ts []WeightedTerm) ([]string, []float64) {
	if len(ts) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(ts))
	ws := make([]float64, 0, len(ts))
	ordered := true
	for _, t := range ts {
		if t.ID == "" {
			continue
		}
		// Against the last *kept* id, not the previous element: a skipped
		// empty ID would otherwise let ["b", "", "a"] pass as ordered.
		if n := len(ids); n > 0 {
			switch {
			case string(t.ID) < ids[n-1]:
				ordered = false
			case ids[n-1] == string(t.ID):
				if t.Weight > ws[n-1] {
					ws[n-1] = t.Weight
				}
				continue
			}
			if !ordered {
				break
			}
		}
		ids = append(ids, string(t.ID))
		ws = append(ws, t.Weight)
	}
	if ordered {
		return ids, ws
	}
	// Unordered: sort a copy of the terms and re-run the same collapse. Not
	// slices.SortFunc over two parallel slices, which has no way to swap them
	// together, and not sort.Slice, whose reflect-based swapper is itself an
	// allocation this function exists to avoid.
	buf := make([]WeightedTerm, 0, len(ts))
	for _, t := range ts {
		if t.ID != "" {
			buf = append(buf, t)
		}
	}
	slices.SortStableFunc(buf, func(x, y WeightedTerm) int { return cmp.Compare(x.ID, y.ID) })
	ids, ws = ids[:0], ws[:0]
	for _, t := range buf {
		if n := len(ids); n > 0 && ids[n-1] == string(t.ID) {
			if t.Weight > ws[n-1] {
				ws[n-1] = t.Weight
			}
			continue
		}
		ids = append(ids, string(t.ID))
		ws = append(ws, t.Weight)
	}
	return ids, ws
}

// weightOf looks a term's weight up in a split's parallel slices. Absent reads
// zero, which is what the map form returned for a term the side never asserted.
func weightOf(ids []string, ws []float64, id string) float64 {
	if i, ok := slices.BinarySearch(ids, id); ok {
		return ws[i]
	}
	return 0
}

// termIDs drops the weights, for the corpus-independent path that has no use
// for them.
func termIDs(ts []WeightedTerm) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = string(t.ID)
	}
	return out
}

package ontology

import "sort"

// WeightedFeature is one term of a learned concept's vocabulary: a
// channel-prefixed feature name ("sel:sql.Open", "call:store.Get") and the
// weight the lexicon gave it (lift × idf).
type WeightedFeature struct {
	Name   string
	Weight float64

	// Opaque marks a feature whose name is a hash rather than a term a reader
	// can look up in the source — the lexicon's structural channel. It scores
	// exactly like any other feature; it only yields its place in the shared
	// list to a legible one, the same rule that keeps it from naming a
	// concept.
	Opaque bool
}

// VocabularyEntry is one concept's vocabulary, as the lexicon learned it.
type VocabularyEntry struct {
	ID       TermID
	Features []WeightedFeature
}

// Vocabulary is the corpus-derived side table beside the Ontology: what each
// learned concept is made of. It travels beside the vocabulary rather than
// inside it for the same reason IC does — Term carries no payload, the
// Ontology is authored and immutable, and internal/lexicon (which produces the
// features) may not import this package, so cmd bridges the two.
//
// It is the substrate of the concept signal's *feature view*: a relatedness
// that never touches the taxonomy, so two concepts hanging from the concept
// root — invisible to every LCA-routed matcher, which drops zero pairings —
// can still be seen to share what they are made of.
//
// Feature names are interned once, so a profile is a slice of (id, weight)
// pairs sorted by id and every comparison is a merge join over integers.
// The scratch buffers below are the one piece of mutable state here, so a
// *Vocabulary is shareable only for reading: a goroutine that profiles needs
// its own, which is what fork (reached through Scorer.Fork) hands out.
type Vocabulary struct {
	names     []string              // interned feature names, sorted; index is the feature id
	of        map[TermID][]weighted // per concept: sorted by feature id, unique, Weight > 0, at most MaxVocabularyFeatures
	truncated int                   // concepts whose vocabulary was cut to MaxVocabularyFeatures

	scratch [2][2][]weighted // one pair of merge buffers per side, reused across comparisons
}

// fork returns a Vocabulary sharing this one's interned names and per-concept
// feature lists — both read-only once built — with its own profile scratch, so
// two goroutines can profile at the same time. See Scorer.Fork.
func (v *Vocabulary) fork() *Vocabulary {
	cp := *v
	cp.scratch = [2][2][]weighted{}
	return &cp
}

type weighted struct {
	f      int
	w      float64
	opaque bool
}

// MaxVocabularyFeatures bounds each concept's vocabulary to its strongest
// features by weight. It is a guard against a pathological corpus, not a
// tuning: on the pinned ladder the largest learned concept carries 2 479
// features (moby; the median is 44), so nothing there is cut, and the value
// sits above it the way family.MaxComponent sits above the largest component
// the ladder produces. Truncated reports how many concepts it touched.
//
// A tighter bound was measured and not taken. The cost of the feature view
// is the profile merge, and once that merge stopped sorting per pair the size
// of the vocabulary barely matters: moby runs in 12.1s unbounded against
// 10.3s without the view, 11.8s at 512, 11.4s at 256 and 11.1s at 128. What a
// tight cap does change is the finding the view exists for — pairs the
// taxonomy scores 0 whose concepts are made of the same things — because the
// shared vocabulary of two large concepts is largely their tails: moby
// reports 400 such pairs unbounded, 230 at 512, 88 at 256 and 24 at 128.
// Weight is lift × idf, so every feature in a tail still cleared the
// lexicon's information window; a cap that drops it trades a second of
// moby's run for most of the signal.
const MaxVocabularyFeatures = 4096

// NewVocabulary builds the table. Features with a non-positive weight are
// dropped; a feature asserted twice for one concept keeps its strongest weight
// (it cannot happen from the lexicon, but must not silently pick one if it
// ever does); a concept keeps at most MaxVocabularyFeatures of them, by
// (weight desc, name asc); an entry with no surviving feature is absent, not
// empty. Input order — of entries and of features — does not reach the
// result.
func NewVocabulary(entries []VocabularyEntry) *Vocabulary {
	seen := make(map[string]struct{})
	for _, e := range entries {
		for _, f := range e.Features {
			if f.Weight > 0 {
				seen[f.Name] = struct{}{}
			}
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	index := make(map[string]int, len(names))
	for i, n := range names {
		index[n] = i
	}

	v := &Vocabulary{names: names, of: make(map[TermID][]weighted, len(entries))}
	for _, e := range entries {
		list := v.of[e.ID]
		for _, f := range e.Features {
			if f.Weight > 0 {
				list = append(list, weighted{f: index[f.Name], w: f.Weight, opaque: f.Opaque})
			}
		}
		list = dedupMax(list)
		if len(list) > MaxVocabularyFeatures {
			// Cut by strength, then restore id order for the merge join.
			sort.Slice(list, func(i, j int) bool {
				if list[i].w != list[j].w {
					return list[i].w > list[j].w
				}
				return v.names[list[i].f] < v.names[list[j].f]
			})
			list = list[:MaxVocabularyFeatures]
			sort.Slice(list, func(i, j int) bool { return list[i].f < list[j].f })
			v.truncated++
		}
		if len(list) > 0 {
			v.of[e.ID] = list
		}
	}
	return v
}

// Truncated is how many concepts had more than MaxVocabularyFeatures
// features and were cut to their strongest.
func (v *Vocabulary) Truncated() int {
	if v == nil {
		return 0
	}
	return v.truncated
}

// dedupMax sorts by feature id and keeps the strongest weight per id.
func dedupMax(list []weighted) []weighted {
	if len(list) == 0 {
		return nil
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].f != list[j].f {
			return list[i].f < list[j].f
		}
		return list[i].w > list[j].w
	})
	out := list[:1]
	for _, x := range list[1:] {
		if x.f != out[len(out)-1].f {
			out = append(out, x)
		}
	}
	return out
}

// Len is the number of concepts with a vocabulary.
func (v *Vocabulary) Len() int {
	if v == nil {
		return 0
	}
	return len(v.of)
}

// Has reports whether the concept has a vocabulary.
func (v *Vocabulary) Has(id TermID) bool {
	if v == nil {
		return false
	}
	_, ok := v.of[id]
	return ok
}

// Features returns a concept's vocabulary by name, weight descending then
// name ascending — the lexicon's own order — or nil.
func (v *Vocabulary) Features(id TermID) []WeightedFeature {
	if v == nil {
		return nil
	}
	list, ok := v.of[id]
	if !ok {
		return nil
	}
	out := make([]WeightedFeature, len(list))
	for i, x := range list {
		out[i] = WeightedFeature{Name: v.names[x.f], Weight: x.w, Opaque: x.opaque}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Weight != out[j].Weight {
			return out[i].Weight > out[j].Weight
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// profile merges one side's concept vocabularies into a single feature
// profile: P(f) = max over its concepts c of conf_c · w(c, f). Max rather
// than sum because overlapping concept vocabularies are the norm (MaxOverlap
// collapses only near-duplicate concepts), and summing would count one
// feature twice for carrying two names for it. Sorted by feature id.
//
// Each concept's list is already id-sorted, so the merge is a pairwise
// sorted merge over at most MaxMemberships lists into the side's scratch
// buffers — no sort and no allocation per comparison. The result aliases
// the scratch and is valid until the same side is profiled again.
func (v *Vocabulary) profile(terms []WeightedTerm, side int) []weighted {
	ids, conf := split(terms)
	cur, nxt := v.scratch[side][0][:0], v.scratch[side][1][:0]
	for i, id := range ids { // sorted: the merge order is fixed
		c := conf[i]
		list := v.of[TermID(id)]
		if c <= 0 || len(list) == 0 {
			continue
		}
		nxt = nxt[:0]
		i, j := 0, 0
		for i < len(cur) || j < len(list) {
			switch {
			case j >= len(list) || (i < len(cur) && cur[i].f < list[j].f):
				nxt = append(nxt, cur[i])
				i++
			case i >= len(cur) || list[j].f < cur[i].f:
				nxt = append(nxt, weighted{f: list[j].f, w: c * list[j].w, opaque: list[j].opaque})
				j++
			default:
				x := cur[i]
				if w := c * list[j].w; w > x.w {
					x.w = w
				}
				nxt = append(nxt, x)
				i++
				j++
			}
		}
		cur, nxt = nxt, cur
	}
	v.scratch[side][0], v.scratch[side][1] = cur, nxt
	return cur
}

// FeatureTopN is how many shared features FeatureRelatedness names — the
// explanation, selected by insertion the way retrieval's shared labels are.
const FeatureTopN = 3

// FeatureRelatedness is the concept signal seen through the learned
// vocabularies alone: how much of what the two sides' concepts are made of is
// the same, and in which direction.
type FeatureRelatedness struct {
	Score  float64           // weighted Jaccard of the two profiles: Σmin / Σmax
	AInB   float64           // Σmin / ΣA — how much of A's vocabulary B's concepts also carry
	BInA   float64           // Σmin / ΣB
	Shared []WeightedFeature // the top FeatureTopN shared features, min weight desc then name asc
}

// FeatureRelatednessW scores two graded concept sets on their vocabularies.
// ok is false when the scorer carries no Vocabulary — the view was not
// measured, which is a different fact from measuring zero.
//
// Each side is merged into one profile (see profile) and the two are joined
// once: Σmin is the shared vocabulary mass, Σmax the union's, and each side's
// own mass is the denominator of its containment. Confidence scales a
// concept's whole vocabulary, so an identical single concept asserted at 0.9
// and 0.5 reads 0.56 here exactly as it does under SetRelatednessW — the two
// views agree on what a weaker claim is worth and differ only in what they
// count. Empty sides, and a pair whose vocabularies do not meet, score 0.
func (s *Scorer) FeatureRelatednessW(a, b []WeightedTerm) (FeatureRelatedness, bool) {
	if s.vocab == nil {
		return FeatureRelatedness{}, false
	}
	pa, pb := s.vocab.profile(a, 0), s.vocab.profile(b, 1)
	var r FeatureRelatedness
	var shared, union, massA, massB float64
	i, j := 0, 0
	for i < len(pa) || j < len(pb) {
		switch {
		case j >= len(pb) || (i < len(pa) && pa[i].f < pb[j].f):
			union += pa[i].w
			massA += pa[i].w
			i++
		case i >= len(pa) || pb[j].f < pa[i].f:
			union += pb[j].w
			massB += pb[j].w
			j++
		default:
			lo, hi := pa[i].w, pb[j].w
			if lo > hi {
				lo, hi = hi, lo
			}
			shared += lo
			union += hi
			massA += pa[i].w
			massB += pb[j].w
			r.Shared = insertShared(r.Shared, WeightedFeature{Name: s.vocab.names[pa[i].f], Weight: lo, Opaque: pa[i].opaque}, FeatureTopN)
			i++
			j++
		}
	}
	if union > 0 {
		r.Score = shared / union
	}
	if massA > 0 {
		r.AInB = shared / massA
	}
	if massB > 0 {
		r.BInA = shared / massB
	}
	return r, true
}

// insertShared keeps the top n shared features by (legible before opaque,
// weight desc, name asc),
// selected by insertion because a substantial pair shares hundreds and sorting
// them all per pair is real cost in a stage that exists to be cheap.
func insertShared(top []WeightedFeature, f WeightedFeature, n int) []WeightedFeature {
	if n <= 0 {
		return top
	}
	pos := len(top)
	for pos > 0 && betterShared(f, top[pos-1]) {
		pos--
	}
	if pos >= n {
		return top
	}
	if len(top) < n {
		top = append(top, WeightedFeature{})
	}
	copy(top[pos+1:], top[pos:])
	top[pos] = f
	return top
}

func betterShared(a, b WeightedFeature) bool {
	if a.Opaque != b.Opaque {
		return !a.Opaque
	}
	if a.Weight != b.Weight {
		return a.Weight > b.Weight
	}
	return a.Name < b.Name
}

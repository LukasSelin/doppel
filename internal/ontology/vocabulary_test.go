package ontology

import (
	"fmt"
	"math"
	"testing"
)

func wf(name string, w float64) WeightedFeature { return WeightedFeature{Name: name, Weight: w} }

func certain(ids ...TermID) []WeightedTerm {
	out := make([]WeightedTerm, len(ids))
	for i, id := range ids {
		out[i] = WeightedTerm{ID: id, Weight: 1}
	}
	return out
}

// twoRootConcepts is a derived vocabulary with two emergent leaves hanging
// directly from the concept root — the case the feature view exists for.
func twoRootConcepts(t *testing.T) *Ontology {
	t.Helper()
	terms := DerivedConceptTerms(Default(), []DerivedConcept{{ID: "x", Def: "x"}, {ID: "y", Def: "y"}})
	o := WithConcepts(Default(), terms)
	if err := o.Validate(); err != nil {
		t.Fatal(err)
	}
	return o
}

func TestFeatureRelatednessAbsentWithoutVocabulary(t *testing.T) {
	s := NewScorer(Default(), nil)
	if _, ok := s.FeatureRelatednessW(certain("x"), certain("x")); ok {
		t.Fatal("a scorer with no vocabulary reported a feature view")
	}
	if s.Vocabulary() != nil {
		t.Fatal("bare scorer carries a vocabulary")
	}
}

func TestFeatureRelatednessIdenticalAndDisjoint(t *testing.T) {
	v := NewVocabulary([]VocabularyEntry{
		{ID: "x", Features: []WeightedFeature{wf("sel:a", 2), wf("call:b", 1)}},
		{ID: "y", Features: []WeightedFeature{wf("sel:c", 3)}},
	})
	s := NewScorer(Default(), nil).WithVocabulary(v)

	same, ok := s.FeatureRelatednessW(certain("x"), certain("x"))
	if !ok || same.Score != 1 || same.AInB != 1 || same.BInA != 1 {
		t.Fatalf("identical concept: %+v ok=%v", same, ok)
	}
	if len(same.Shared) != 2 || same.Shared[0].Name != "sel:a" || same.Shared[1].Name != "call:b" {
		t.Fatalf("shared features of an identical pair: %+v", same.Shared)
	}
	none, _ := s.FeatureRelatednessW(certain("x"), certain("y"))
	if none.Score != 0 || none.AInB != 0 || none.BInA != 0 || none.Shared != nil {
		t.Fatalf("disjoint vocabularies: %+v", none)
	}
	empty, ok := s.FeatureRelatednessW(nil, certain("y"))
	if !ok || empty.Score != 0 {
		t.Fatalf("empty side: %+v ok=%v", empty, ok)
	}
}

func TestFeatureRelatednessJaccardAndContainment(t *testing.T) {
	// y's vocabulary is a subset of x's: containment is asymmetric.
	v := NewVocabulary([]VocabularyEntry{
		{ID: "x", Features: []WeightedFeature{wf("a", 2), wf("b", 1), wf("c", 1)}},
		{ID: "y", Features: []WeightedFeature{wf("a", 1), wf("b", 1)}},
	})
	s := NewScorer(Default(), nil).WithVocabulary(v)
	r, _ := s.FeatureRelatednessW(certain("x"), certain("y"))
	// Σmin = 1 + 1 = 2, Σmax = 2 + 1 + 1 = 4, ΣA = 4, ΣB = 2.
	if r.Score != 0.5 || r.AInB != 0.5 || r.BInA != 1 {
		t.Fatalf("got %+v", r)
	}
	if len(r.Shared) != 2 || r.Shared[0].Name != "a" || r.Shared[0].Weight != 1 || r.Shared[1].Name != "b" {
		t.Fatalf("shared: %+v", r.Shared)
	}
	// Symmetric quantities do not depend on the order of the sides.
	rev, _ := s.FeatureRelatednessW(certain("y"), certain("x"))
	if rev.Score != r.Score || rev.AInB != r.BInA || rev.BInA != r.AInB {
		t.Fatalf("asymmetric under side swap: %+v vs %+v", r, rev)
	}
}

func TestFeatureRelatednessConfidenceScalesLikeCorpusView(t *testing.T) {
	v := NewVocabulary([]VocabularyEntry{{ID: "x", Features: []WeightedFeature{wf("a", 3), wf("b", 1)}}})
	s := NewScorer(Default(), nil).WithVocabulary(v)
	r, _ := s.FeatureRelatednessW(
		[]WeightedTerm{{ID: "x", Weight: 0.9}}, []WeightedTerm{{ID: "x", Weight: 0.5}})
	if want := 0.5 / 0.9; math.Abs(r.Score-want) > 1e-12 {
		t.Fatalf("score %v, want %v", r.Score, want)
	}
}

func TestFeatureProfileMergesOverlappingConceptsByMax(t *testing.T) {
	// One side carries two concepts that both name feature a: the profile keeps
	// the strongest, never the sum, so a shared feature is not counted twice.
	v := NewVocabulary([]VocabularyEntry{
		{ID: "x", Features: []WeightedFeature{wf("a", 1)}},
		{ID: "y", Features: []WeightedFeature{wf("a", 3)}},
		{ID: "z", Features: []WeightedFeature{wf("a", 3)}},
	})
	s := NewScorer(Default(), nil).WithVocabulary(v)
	r, _ := s.FeatureRelatednessW(certain("x", "y"), certain("z"))
	if r.Score != 1 {
		t.Fatalf("max-merged profile should equal z exactly, got %+v", r)
	}
}

func TestVocabularyConstructorNormalizes(t *testing.T) {
	v := NewVocabulary([]VocabularyEntry{
		{ID: "x", Features: []WeightedFeature{wf("b", 1), wf("a", 2), wf("a", 5), wf("dead", 0), wf("neg", -1)}},
		{ID: "x", Features: []WeightedFeature{wf("c", 1)}}, // a second entry for the same concept merges
		{ID: "empty", Features: []WeightedFeature{wf("dead", 0)}},
	})
	if v.Len() != 1 || !v.Has("x") || v.Has("empty") || v.Has("nope") {
		t.Fatalf("Len %d Has(x) %v Has(empty) %v", v.Len(), v.Has("x"), v.Has("empty"))
	}
	got := v.Features("x")
	want := []WeightedFeature{wf("a", 5), wf("b", 1), wf("c", 1)}
	if len(got) != len(want) {
		t.Fatalf("features %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("features %+v, want %+v", got, want)
		}
	}
	var nilV *Vocabulary
	if nilV.Len() != 0 || nilV.Has("x") || nilV.Features("x") != nil {
		t.Fatal("nil Vocabulary is not empty")
	}
}

func TestFeatureRelatednessIsInputOrderIndependent(t *testing.T) {
	a := NewVocabulary([]VocabularyEntry{
		{ID: "x", Features: []WeightedFeature{wf("a", 2), wf("b", 1), wf("c", 0.5)}},
		{ID: "y", Features: []WeightedFeature{wf("c", 0.5), wf("b", 1), wf("d", 4)}},
	})
	b := NewVocabulary([]VocabularyEntry{
		{ID: "y", Features: []WeightedFeature{wf("d", 4), wf("b", 1), wf("c", 0.5)}},
		{ID: "x", Features: []WeightedFeature{wf("c", 0.5), wf("a", 2), wf("b", 1)}},
	})
	ra, _ := NewScorer(Default(), nil).WithVocabulary(a).FeatureRelatednessW(certain("x"), certain("y"))
	rb, _ := NewScorer(Default(), nil).WithVocabulary(b).FeatureRelatednessW(certain("x"), certain("y"))
	if ra.Score != rb.Score || ra.AInB != rb.AInB || ra.BInA != rb.BInA || len(ra.Shared) != len(rb.Shared) {
		t.Fatalf("input order reached the result: %+v vs %+v", ra, rb)
	}
	for i := range ra.Shared {
		if ra.Shared[i] != rb.Shared[i] {
			t.Fatalf("shared order differs: %+v vs %+v", ra.Shared, rb.Shared)
		}
	}
	// b (1.0) before c (0.5): weight desc, then name asc.
	if ra.Shared[0].Name != "b" || ra.Shared[1].Name != "c" {
		t.Fatalf("shared order: %+v", ra.Shared)
	}
}

func TestSharedFeaturesAreBoundedAndOrdered(t *testing.T) {
	feats := []WeightedFeature{wf("a", 1), wf("b", 1), wf("c", 1), wf("d", 1), wf("e", 5)}
	v := NewVocabulary([]VocabularyEntry{{ID: "x", Features: feats}})
	r, _ := NewScorer(Default(), nil).WithVocabulary(v).FeatureRelatednessW(certain("x"), certain("x"))
	if len(r.Shared) != FeatureTopN {
		t.Fatalf("shared list not bounded at %d: %+v", FeatureTopN, r.Shared)
	}
	if r.Shared[0].Name != "e" || r.Shared[1].Name != "a" || r.Shared[2].Name != "b" {
		t.Fatalf("shared order: %+v", r.Shared)
	}
}

// The point of the view: two concepts the taxonomy cannot relate at all —
// both hang from the concept root, so every LCA-routed matcher drops the
// pairing — still read as related through what they are made of.
func TestFeatureViewSeesRootHungConcepts(t *testing.T) {
	o := twoRootConcepts(t)
	ic := NewCorpusIC(o, map[TermID]int{"x": 5, "y": 5})
	v := NewVocabulary([]VocabularyEntry{
		{ID: "x", Features: []WeightedFeature{wf("sel:store.Get", 3), wf("call:Decode", 2), wf("id:key", 1)}},
		{ID: "y", Features: []WeightedFeature{wf("sel:store.Get", 3), wf("call:Decode", 2), wf("id:ttl", 1)}},
	})
	s := NewScorer(o, ic).WithVocabulary(v)

	corpus, matches := s.SetRelatednessW(certain("x"), certain("y"))
	if corpus != 0 || len(matches) != 0 {
		t.Fatalf("corpus view saw a root-hung pair: %v %+v", corpus, matches)
	}
	if shape, _ := o.SetRelatedness([]string{"x"}, []string{"y"}); shape != 0 {
		t.Fatalf("shape view saw a root-hung pair: %v", shape)
	}
	feat, ok := s.FeatureRelatednessW(certain("x"), certain("y"))
	if !ok || feat.Score <= 0.5 {
		t.Fatalf("feature view: %+v ok=%v", feat, ok)
	}
	if feat.Shared[0].Name != "sel:store.Get" {
		t.Fatalf("shared vocabulary: %+v", feat.Shared)
	}
}

func TestWithVocabularyIsACopy(t *testing.T) {
	base := NewScorer(Default(), nil)
	with := base.WithVocabulary(NewVocabulary(nil))
	if base.Vocabulary() != nil || with.Vocabulary() == nil {
		t.Fatal("WithVocabulary mutated the receiver or lost the table")
	}
	if with.Ontology() != base.Ontology() || with.Weighted() != base.Weighted() {
		t.Fatal("copy dropped the ontology or the IC")
	}
}

// A hash-named feature scores like any other but yields its place in the
// evidence to a legible one, however heavy it is.
func TestSharedFeaturesPreferLegibleOverOpaque(t *testing.T) {
	feats := []WeightedFeature{
		{Name: "act:depth-2 IF#abc", Weight: 9, Opaque: true},
		{Name: "sel:store.Get", Weight: 2},
		{Name: "call:Decode", Weight: 1},
	}
	v := NewVocabulary([]VocabularyEntry{{ID: "x", Features: feats}})
	r, _ := NewScorer(Default(), nil).WithVocabulary(v).FeatureRelatednessW(certain("x"), certain("x"))
	if r.Score != 1 {
		t.Fatalf("an opaque feature must still score: %+v", r)
	}
	if len(r.Shared) != 3 || r.Shared[0].Name != "sel:store.Get" || r.Shared[1].Name != "call:Decode" || !r.Shared[2].Opaque {
		t.Fatalf("shared order: %+v", r.Shared)
	}
}

// A concept keeps its strongest MaxVocabularyFeatures features, and says so.
func TestVocabularyIsBounded(t *testing.T) {
	var feats []WeightedFeature
	for i := 0; i < MaxVocabularyFeatures+10; i++ {
		feats = append(feats, WeightedFeature{Name: fmt.Sprintf("f%04d", i), Weight: float64(i + 1)})
	}
	v := NewVocabulary([]VocabularyEntry{{ID: "big", Features: feats}, {ID: "small", Features: feats[:3]}})
	if v.Truncated() != 1 {
		t.Fatalf("truncated %d, want 1", v.Truncated())
	}
	got := v.Features("big")
	if len(got) != MaxVocabularyFeatures || got[0].Name != fmt.Sprintf("f%04d", MaxVocabularyFeatures+9) || got[len(got)-1].Name != "f0010" {
		t.Fatalf("kept %d features, first %s last %s", len(got), got[0].Name, got[len(got)-1].Name)
	}
	// The bound is by weight, and the cut list is still a valid profile.
	r, _ := NewScorer(Default(), nil).WithVocabulary(v).FeatureRelatednessW(certain("big"), certain("big"))
	if r.Score != 1 {
		t.Fatalf("self-similarity after the cut: %+v", r)
	}
}

// Profiling one side must not disturb the other's scratch: the two sides of
// a comparison are held at once.
func TestProfilesOfBothSidesCoexist(t *testing.T) {
	v := NewVocabulary([]VocabularyEntry{
		{ID: "x", Features: []WeightedFeature{wf("a", 1), wf("b", 2)}},
		{ID: "y", Features: []WeightedFeature{wf("b", 2), wf("c", 3)}},
		{ID: "z", Features: []WeightedFeature{wf("a", 1), wf("c", 3), wf("d", 4)}},
	})
	s := NewScorer(Default(), nil).WithVocabulary(v)
	// x∪y = {a:1, b:2, c:3}; z = {a:1, c:3, d:4}. Σmin = 4, Σmax = 1+2+3+4 = 10.
	r, _ := s.FeatureRelatednessW(certain("x", "y"), certain("z"))
	if r.Score != 0.4 || r.AInB != 4.0/6 || r.BInA != 0.5 {
		t.Fatalf("got %+v", r)
	}
	// And again, so a stale scratch would show.
	r2, _ := s.FeatureRelatednessW(certain("z"), certain("x", "y"))
	if r2.Score != r.Score || r2.AInB != r.BInA {
		t.Fatalf("second call differs: %+v vs %+v", r2, r)
	}
}

package comparator

import (
	"math"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// learnedOntology is a derived vocabulary of the kind a run reasons over:
// two seeded siblings under data_store_access, and two emergent concepts
// hanging from the concept root because they resemble no seed.
func learnedOntology(t *testing.T) *ontology.Ontology {
	t.Helper()
	terms := ontology.DerivedConceptTerms(ontology.Default(), []ontology.DerivedConcept{
		{ID: "sql.Open+QueryRow", Seed: ontology.ConDBAccess, Def: "db"},
		{ID: "cache.Get+Set", Seed: ontology.ConCaching, Def: "cache"},
		{ID: "store.Get+Decode", Def: "emergent store"},
		{ID: "store.Put+Encode", Def: "emergent store, the other way"},
	})
	o := ontology.WithConcepts(ontology.Default(), terms)
	if err := o.Validate(); err != nil {
		t.Fatal(err)
	}
	return o
}

func learnedVocabulary() *ontology.Vocabulary {
	f := func(name string, w float64) ontology.WeightedFeature {
		return ontology.WeightedFeature{Name: name, Weight: w}
	}
	return ontology.NewVocabulary([]ontology.VocabularyEntry{
		{ID: "sql.Open+QueryRow", Features: []ontology.WeightedFeature{f("sel:sql.Open", 4), f("sel:db.QueryRow", 3), f("id:row", 1)}},
		{ID: "cache.Get+Set", Features: []ontology.WeightedFeature{f("sel:cache.Get", 4), f("sel:cache.Set", 3), f("id:ttl", 1)}},
		{ID: "store.Get+Decode", Features: []ontology.WeightedFeature{f("sel:store.Get", 4), f("call:Decode", 3), f("id:key", 1), f("id:store", 1)}},
		{ID: "store.Put+Encode", Features: []ontology.WeightedFeature{f("sel:store.Put", 4), f("call:Encode", 3), f("id:key", 1), f("id:store", 1)}},
	})
}

func learnedScorer(t *testing.T) *ontology.Scorer {
	t.Helper()
	o := learnedOntology(t)
	ic := ontology.NewCorpusIC(o, map[ontology.TermID]int{
		"sql.Open+QueryRow": 10, "cache.Get+Set": 6, "store.Get+Decode": 4, "store.Put+Encode": 4,
	})
	return ontology.NewScorer(o, ic)
}

func conceptDoc(name string, concepts ...string) concepter.ConceptDoc {
	return concepter.ConceptDoc{Name: name, Package: "p", Role: "utility", Concepts: parser.Certain(concepts...)}
}

// The free comparator has no vocabulary: the feature view is absent, not
// zero, and the shape and corpus views coincide with the number the composite
// always read.
func TestFreeCompareViewsAreShapeOnly(t *testing.T) {
	ev := Compare(conceptDoc("a", "db_access"), conceptDoc("b", "caching"))
	if ev.Views.HasFeature || ev.Views.Disagree {
		t.Fatalf("free comparator measured a feature view: %+v", ev.Views)
	}
	if ev.Views.Shape != ev.PatternRelatedness || ev.Views.Corpus != ev.PatternRelatedness {
		t.Fatalf("free views %+v, PatternRelatedness %v", ev.Views, ev.PatternRelatedness)
	}
	if ev.Exhibits != ev.PatternRelatedness {
		t.Fatalf("exhibits %v, corpus view %v", ev.Exhibits, ev.PatternRelatedness)
	}
	if len(ev.TaxonomyPatterns) != 1 || ev.TaxonomyPatterns[0].LCA != ontology.ConDataStoreAccess {
		t.Fatalf("taxonomy pairings: %+v", ev.TaxonomyPatterns)
	}
}

// The whole point: two emergent concepts the taxonomy hangs from the root
// score zero on both tree views and positive on the feature view — and adding
// the vocabulary changes nothing the composite reads under the incumbent
// blend.
func TestViewsSeeRootHungConcepts(t *testing.T) {
	s := learnedScorer(t)
	a, b := conceptDoc("get", "store.Get+Decode"), conceptDoc("put", "store.Put+Encode")

	without := NewWith(s, Options{}).Compare(a, b)
	with := NewWith(s.WithVocabulary(learnedVocabulary()), Options{}).Compare(a, b)

	if without.Views.HasFeature {
		t.Fatal("no vocabulary, yet a feature view")
	}
	v := with.Views
	if !v.HasFeature || v.Shape != 0 || v.Corpus != 0 || v.Feature <= 0 {
		t.Fatalf("views %+v", v)
	}
	// Σmin = id:key 1 + id:store 1 = 2; Σmax = 4+3+1+1 + 4+3 = 16.
	if math.Abs(v.Feature-2.0/16) > 1e-12 || math.Abs(v.AInB-2.0/9) > 1e-12 || math.Abs(v.BInA-2.0/9) > 1e-12 {
		t.Fatalf("feature arithmetic: %+v", v)
	}
	if len(v.SharedVocabulary) != 2 || v.SharedVocabulary[0].Name != "id:key" || v.SharedVocabulary[1].Name != "id:store" {
		t.Fatalf("shared vocabulary: %+v", v.SharedVocabulary)
	}
	if v.Disagree {
		t.Fatalf("a spread of %.2f must not trip the %.2f flag", v.Feature-v.Shape, ViewDisagreeSpread)
	}
	if with.OverlapScore != without.OverlapScore || with.PatternSignalBest != without.PatternSignalBest ||
		strings.Join(with.Reasons, "\n") != strings.Join(without.Reasons, "\n") {
		t.Fatal("the vocabulary moved the composite, the gate or the reasons under the incumbent blend")
	}
}

func TestViewsDisagreeInBothDirections(t *testing.T) {
	s := learnedScorer(t).WithVocabulary(learnedVocabulary())
	comp := NewWith(s, Options{})

	// Seeded siblings: the taxonomy says 0.67, the vocabularies say nothing.
	sib := comp.Compare(conceptDoc("db", "sql.Open+QueryRow"), conceptDoc("cache", "cache.Get+Set")).Views
	if !approx(sib.Shape, 2.0/3.0) || sib.Feature != 0 || !sib.Disagree {
		t.Fatalf("siblings with disjoint vocabularies: %+v", sib)
	}

	// Root-hung twins made of the same things: the taxonomy says 0, the
	// vocabularies say identical.
	twin := ontology.NewVocabulary([]ontology.VocabularyEntry{
		{ID: "store.Get+Decode", Features: []ontology.WeightedFeature{{Name: "sel:store.Get", Weight: 4}}},
		{ID: "store.Put+Encode", Features: []ontology.WeightedFeature{{Name: "sel:store.Get", Weight: 4}}},
	})
	root := NewWith(learnedScorer(t).WithVocabulary(twin), Options{}).
		Compare(conceptDoc("get", "store.Get+Decode"), conceptDoc("put", "store.Put+Encode")).Views
	if root.Shape != 0 || root.Feature != 1 || !root.Disagree {
		t.Fatalf("root-hung twins: %+v", root)
	}

	// Identical concepts agree from every angle.
	same := comp.Compare(conceptDoc("a", "sql.Open+QueryRow"), conceptDoc("b", "sql.Open+QueryRow")).Views
	if same.Shape != 1 || same.Corpus != 1 || same.Feature != 1 || same.AInB != 1 || same.BInA != 1 || same.Disagree {
		t.Fatalf("identical concepts: %+v", same)
	}
}

func TestBlendsMoveOnlyTheExhibitsSlot(t *testing.T) {
	s := learnedScorer(t).WithVocabulary(learnedVocabulary())
	a, b := conceptDoc("db", "sql.Open+QueryRow"), conceptDoc("cache", "cache.Get+Set")
	base := NewWith(s, Options{}).Compare(a, b)
	if base.Exhibits != base.Views.Corpus {
		t.Fatalf("zero options read %v, corpus view is %v", base.Exhibits, base.Views.Corpus)
	}
	corpusOnly := NewWith(s, Options{Exhibits: ViewBlend{Corpus: 1}}).Compare(a, b)
	if corpusOnly.OverlapScore != base.OverlapScore || corpusOnly.Exhibits != base.Exhibits {
		t.Fatal("an explicit corpus-only weighting is not bit-identical to the incumbent")
	}

	w := ontology.Default().Weight(ontology.RelExhibits)
	for _, tt := range []struct {
		name string
		opt  ViewBlend
		want float64
	}{
		{"weighted", ViewBlend{Shape: 1, Corpus: 1, Feature: 1}, (base.Views.Shape + base.Views.Corpus + base.Views.Feature) / 3},
		{"geometric", ViewBlend{Mode: BlendGeometric}, 0}, // feature is 0 for this pair
		{"max", ViewBlend{Mode: BlendMax}, math.Max(base.Views.Shape, math.Max(base.Views.Corpus, base.Views.Feature))},
	} {
		ev := NewWith(s, Options{Exhibits: tt.opt}).Compare(a, b)
		if math.Abs(ev.Exhibits-tt.want) > 1e-12 {
			t.Errorf("%s: exhibits %v, want %v", tt.name, ev.Exhibits, tt.want)
		}
		wantScore := base.OverlapScore - w*base.Exhibits + w*ev.Exhibits
		if math.Abs(ev.OverlapScore-wantScore) > 1e-12 {
			t.Errorf("%s: overlap %v, want %v — the blend reached something other than the exhibits slot", tt.name, ev.OverlapScore, wantScore)
		}
		if ev.Views.Shape != base.Views.Shape || ev.Views.Corpus != base.Views.Corpus || ev.Views.Feature != base.Views.Feature {
			t.Errorf("%s: the blend changed the reported views", tt.name)
		}
		// The measurement vector still reproduces the composite under the blend.
		v := SignalVector(ev, a, b)
		var sum float64
		for i, rel := range ontology.Default().ScoredRelations() {
			sum += ontology.Default().Weight(rel) * v[i]
		}
		if math.Abs(sum-ev.OverlapScore) > 1e-12 {
			t.Errorf("%s: signal vector sums to %v, composite is %v", tt.name, sum, ev.OverlapScore)
		}
	}
}

func TestBlendWithoutFeatureViewUsesAvailableViews(t *testing.T) {
	v := ConceptViews{Shape: 0.6, Corpus: 0.2}
	if got := (ViewBlend{Shape: 1, Corpus: 1, Feature: 8}).Apply(v); math.Abs(got-0.4) > 1e-12 {
		t.Errorf("weighted without feature: %v, want 0.4 (the feature weight must not count as a zero view)", got)
	}
	if got := (ViewBlend{Mode: BlendGeometric}).Apply(v); math.Abs(got-math.Sqrt(0.12)) > 1e-12 {
		t.Errorf("geometric without feature: %v", got)
	}
	if got := (ViewBlend{Mode: BlendMax}).Apply(v); got != 0.6 {
		t.Errorf("max without feature: %v", got)
	}
	if got := (ViewBlend{Feature: 1}).Apply(v); got != 0.2 {
		t.Errorf("a blend with no available weight falls back to the corpus view: %v", got)
	}
	withF := ConceptViews{Shape: 0.6, Corpus: 0.2, Feature: 0.9, HasFeature: true}
	if got := (ViewBlend{Mode: BlendGeometric}).Apply(withF); math.Abs(got-math.Cbrt(0.6*0.2*0.9)) > 1e-12 {
		t.Errorf("geometric with feature: %v", got)
	}
}

func TestViewsAreDeterministic(t *testing.T) {
	s := learnedScorer(t).WithVocabulary(learnedVocabulary())
	comp := NewWith(s, Options{Exhibits: ViewBlend{Shape: 1, Corpus: 2, Feature: 3}})
	a := conceptDoc("a", "sql.Open+QueryRow", "store.Get+Decode")
	b := conceptDoc("b", "cache.Get+Set", "store.Put+Encode")
	first := comp.Compare(a, b)
	for i := 0; i < 200; i++ {
		ev := comp.Compare(a, b)
		if ev.OverlapScore != first.OverlapScore || ev.Views.Feature != first.Views.Feature ||
			ev.Views.AInB != first.Views.AInB || len(ev.Views.SharedVocabulary) != len(first.Views.SharedVocabulary) {
			t.Fatalf("run %d differs: %+v vs %+v", i, ev.Views, first.Views)
		}
		for j := range ev.Views.SharedVocabulary {
			if ev.Views.SharedVocabulary[j] != first.Views.SharedVocabulary[j] {
				t.Fatalf("run %d: shared vocabulary order moved", i)
			}
		}
	}
}

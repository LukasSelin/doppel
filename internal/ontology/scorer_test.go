package ontology

import (
	"math"
	"strings"
	"testing"
)

func toyScorer() *Scorer {
	o := Default()
	return NewScorer(o, NewCorpusIC(o, toyCounts()))
}

// adversarialScorer is the corpus on which sorting candidates by similarity is
// provably suboptimal: retry and concurrency are both rare while retry's
// fault_tolerance sibling circuit_breaker is rarer still, so
// Lin(retry, concurrency) > Lin(retry, circuit_breaker) even though the
// retry/circuit_breaker pair shares strictly more information.
//
// The original inversion lived on the io_operation branch (db_access against
// caching and http_call), but the file_io and logging leaves added in 1.1.0
// dilute io_operation's IC through their pseudo-counts and dissolve it.
// control_flow gained only circuit_breaker, so the same construction survives
// there.
func adversarialScorer() *Scorer {
	o := Default()
	return NewScorer(o, NewCorpusIC(o, map[TermID]int{
		ConRetry:         6,
		ConConcurrency:   2,
		ConErrorWrapping: 94,
	}))
}

// The nil-IC scorer must be the Ontology's own methods, not a reimplementation.
func TestScorerWithoutICDelegates(t *testing.T) {
	o := Default()
	s := NewScorer(o, nil)
	if s.Weighted() {
		t.Fatal("nil-IC scorer reports Weighted")
	}
	pairs := [][2]TermID{
		{ConDBAccess, ConCaching}, {ConHTTPCall, ConRetry}, {ConHTTPCall, ConHTTPCall},
	}
	for _, p := range pairs {
		if got, want := s.Relatedness(p[0], p[1]), o.Relatedness(p[0], p[1]); got != want {
			t.Errorf("Relatedness(%q, %q) = %v via scorer, %v via ontology", p[0], p[1], got, want)
		}
	}
	sets := [][2][]string{
		{{"db_access", "retry"}, {"caching", "retry"}},
		{{"error_wrapping"}, {"error_wrapping"}},
		{nil, nil},
	}
	for _, p := range sets {
		gs, gm := s.SetRelatedness(p[0], p[1])
		ws, wm := o.SetRelatedness(p[0], p[1])
		if gs != ws || len(gm) != len(wm) {
			t.Errorf("SetRelatedness(%v, %v) = (%v, %d matches) via scorer, (%v, %d) via ontology",
				p[0], p[1], gs, len(gm), ws, len(wm))
		}
	}
}

// Lin similarity on the toy corpus. Values are computed from the frequency
// table pinned in ic_test.go, with two documentation-level literals as a
// cross-check against the design notes.
func TestLinRelatednessToyValues(t *testing.T) {
	s := toyScorer()
	lin := func(fLCS, fA, fB float64) float64 {
		return 2 * math.Log(114/fLCS) / (math.Log(114/fA) + math.Log(114/fB))
	}
	tests := []struct {
		name string
		a, b TermID
		want float64
	}{
		{"identical stays 1.0 regardless of ubiquity", ConErrorWrapping, ConErrorWrapping, 1.0},
		{"siblings", ConDBAccess, ConCaching, lin(21, 11, 6)},
		{"cousins", ConHTTPCall, ConDBAccess, lin(30, 6, 11)},
		{"cross-branch is exactly zero", ConHTTPCall, ConRetry, 0.0},
		{"shallow siblings", ConMapping, ConValidation, lin(13, 6, 6)},
		{"unmentioned leaf carries only its pseudo-count", ConHTTPCall, ConGRPCCall, lin(7, 6, 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.Relatedness(tt.a, tt.b)
			if !closeTo(got, tt.want) {
				t.Errorf("Lin(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
			if rev := s.Relatedness(tt.b, tt.a); rev != got {
				t.Errorf("Lin is asymmetric on (%q, %q)", tt.a, tt.b)
			}
		})
	}
	// The two literals the design write-up quotes.
	if got := s.Relatedness(ConDBAccess, ConCaching); math.Abs(got-0.640454) > 1e-4 {
		t.Errorf("Lin(db_access, caching) = %v, want ~0.640454", got)
	}
	if got := s.Relatedness(ConHTTPCall, ConDBAccess); math.Abs(got-0.505420) > 1e-4 {
		t.Errorf("Lin(http_call, db_access) = %v, want ~0.505420", got)
	}
}

func TestLinRelatednessGuards(t *testing.T) {
	s := toyScorer()
	tests := []struct {
		name string
		a, b TermID
		want float64
	}{
		{"empty left", "", ConHTTPCall, 0},
		{"both empty", "", "", 0},
		{"unknown vs known", "soap_call", ConHTTPCall, 0},
		{"unknown vs itself", "soap_call", "soap_call", 1},
		{"cross kind", ConHTTPCall, RoleLeaf, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Relatedness(tt.a, tt.b); !closeTo(got, tt.want) {
				t.Errorf("Lin(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// The point of the whole change, pinned: sharing only the ubiquitous tag now
// scores far below sharing only a rare one — both are 0.5 unweighted.
func TestWeightedSetRelatednessDiscountsUbiquity(t *testing.T) {
	s := toyScorer()

	ubiquitous, _ := s.SetRelatedness(
		[]string{"error_wrapping", "mapping"}, []string{"error_wrapping", "concurrency"})
	rare, _ := s.SetRelatedness(
		[]string{"db_access", "mapping"}, []string{"db_access", "concurrency"})

	if math.Abs(ubiquitous-0.175173) > 1e-4 {
		t.Errorf("sharing only error_wrapping scored %v, want ~0.1752", ubiquitous)
	}
	if math.Abs(rare-0.442631) > 1e-4 {
		t.Errorf("sharing only db_access scored %v, want ~0.4426", rare)
	}
	if ubiquitous >= rare {
		t.Errorf("ubiquitous shared tag (%v) should score below rare shared tag (%v)", ubiquitous, rare)
	}

	// Documented residual limitation: singleton sets cannot be discounted —
	// the shared information and the total information are the same quantity.
	if got, _ := s.SetRelatedness([]string{"error_wrapping"}, []string{"error_wrapping"}); got != 1.0 {
		t.Errorf("{ew} vs {ew} = %v, want exactly 1.0", got)
	}
}

func TestWeightedSetRelatednessToyValues(t *testing.T) {
	s := toyScorer()
	ic := s.ic
	tests := []struct {
		name string
		a, b []string
		want float64
	}{
		{"identical multi-tag sets", []string{"error_wrapping", "db_access"}, []string{"db_access", "error_wrapping"}, 1.0},
		{"shared ubiquitous plus sibling pair",
			[]string{"error_wrapping", "db_access"}, []string{"error_wrapping", "caching"},
			(ic.Of(ConErrorWrapping) + ic.Of(ConDataStoreAccess)) /
				(ic.Of(ConErrorWrapping) + ic.Of(ConCaching))},
		{"both empty", nil, nil, 0},
		{"one empty", []string{"retry"}, nil, 0},
		{"cross-branch only", []string{"retry"}, []string{"db_access"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := s.SetRelatedness(tt.a, tt.b)
			if !closeTo(got, tt.want) {
				t.Errorf("SetRelatedness(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
			rev, _ := s.SetRelatedness(tt.b, tt.a)
			if rev != got {
				t.Errorf("weighted SetRelatedness is asymmetric on (%v, %v)", tt.a, tt.b)
			}
		})
	}
	// Documentation-level literal from the design write-up.
	got, _ := s.SetRelatedness([]string{"error_wrapping", "db_access"}, []string{"error_wrapping", "caching"})
	if math.Abs(got-0.649063) > 1e-4 {
		t.Errorf("{ew,db} vs {ew,caching} = %v, want ~0.649063", got)
	}
}

// Matches come back in consumption order — descending contribution, which under
// IC is not descending similarity: the sibling pair carries more information
// than the exact match on the ubiquitous tag, so it is consumed first.
func TestWeightedMatchesOrderedByContribution(t *testing.T) {
	s := toyScorer()
	_, matches := s.SetRelatedness(
		[]string{"error_wrapping", "db_access"}, []string{"error_wrapping", "caching"})
	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2", len(matches))
	}
	if matches[0].A != "db_access" || matches[0].B != "caching" {
		t.Errorf("first match = %s/%s, want db_access/caching (higher contribution)", matches[0].A, matches[0].B)
	}
	if matches[1].A != "error_wrapping" || !matches[1].Exact() {
		t.Errorf("second match = %s/%s, want the exact error_wrapping pair", matches[1].A, matches[1].B)
	}
	// Score carries the Lin similarity for the evidence lines, so the exact
	// match still reads 1.00 even though it was consumed second.
	if matches[1].Score != 1.0 {
		t.Errorf("exact match Score = %v, want 1.0", matches[1].Score)
	}
	if matches[0].LCA != ConDataStoreAccess {
		t.Errorf("sibling match LCA = %q, want data_store_access", matches[0].LCA)
	}
}

// The adversarial pin: greedy by similarity would pair retry with concurrency
// (more similar) and score ~0.2802; pairing it with circuit_breaker (more
// shared information) is optimal at ~0.3180.
func TestWeightedMatcherPrefersContributionOverSimilarity(t *testing.T) {
	s := adversarialScorer()

	if simConc, simCB := s.Relatedness(ConRetry, ConConcurrency), s.Relatedness(ConRetry, ConCircuitBreaker); simConc <= simCB {
		t.Fatalf("fixture broken: need Lin(retry,concurrency)=%v > Lin(retry,circuit_breaker)=%v for the inversion", simConc, simCB)
	}

	got, matches := s.SetRelatedness([]string{"retry"}, []string{"circuit_breaker", "concurrency"})
	if math.Abs(got-0.318027) > 1e-4 {
		t.Errorf("score = %v, want ~0.318027 (the optimal pairing)", got)
	}
	if len(matches) != 1 || matches[0].B != "circuit_breaker" {
		t.Fatalf("matched %+v, want retry paired with circuit_breaker", matches)
	}
}

// weightedExactShare is the IC-weighted analogue of the old exact ratio: the
// information of the literal intersection over the larger side's information.
func weightedExactShare(ic *IC, a, b []string) float64 {
	as, bs := sortedUnique(a), sortedUnique(b)
	inSet := map[string]bool{}
	for _, t := range as {
		inSet[t] = true
	}
	var shared, sumA, sumB float64
	for _, t := range bs {
		if inSet[t] {
			shared += ic.Of(TermID(t))
		}
		sumB += ic.Of(TermID(t))
	}
	for _, t := range as {
		sumA += ic.Of(TermID(t))
	}
	denom := sumA
	if sumB > denom {
		denom = sumB
	}
	if denom <= 0 {
		return 0
	}
	return shared / denom
}

// The weighted analogue of the uniform soft >= exact invariant: hierarchy can
// only add shared information on top of the literal intersection, never
// destroy it. Exhaustive over subsets of size <= 3 on the toy corpus.
func TestWeightedSetRelatednessNeverFallsBelowWeightedExactShare(t *testing.T) {
	s := toyScorer()
	leaves := conceptLeaves(t)
	var sets [][]string
	for i := range leaves {
		sets = append(sets, []string{leaves[i]})
		for j := i + 1; j < len(leaves); j++ {
			sets = append(sets, []string{leaves[i], leaves[j]})
			for k := j + 1; k < len(leaves); k++ {
				sets = append(sets, []string{leaves[i], leaves[j], leaves[k]})
			}
		}
	}
	var checked int
	for _, a := range sets {
		for _, b := range sets {
			got, _ := s.SetRelatedness(a, b)
			if want := weightedExactShare(s.ic, a, b); got < want-1e-9 {
				t.Fatalf("SetRelatedness(%v, %v) = %v, below weighted exact share %v", a, b, got, want)
			}
			if got > 1.0+1e-9 {
				t.Fatalf("SetRelatedness(%v, %v) = %v, above 1.0", a, b, got)
			}
			checked++
		}
	}
	t.Logf("checked %d set pairs", checked)
}

// The contribution-sorted greedy must equal the exact optimum under IC weights,
// checked against the same oracle that certifies the uniform matcher. Full
// exhaustion runs on the covering subset — which contains the adversarial
// corpus's whole inversion (retry, circuit_breaker, concurrency), where
// similarity-sorted greedy is known to fail on thousands of subset pairs —
// with seeded sampling over the full 14-leaf space alongside.
func TestWeightedGreedyMatchingIsOptimal(t *testing.T) {
	for _, tc := range []struct {
		name string
		s    *Scorer
		full bool
	}{
		{"adversarial corpus", adversarialScorer(), true},
		{"toy corpus", toyScorer(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.s
			leaves := conceptLeaves(t)

			contrib := make(map[[2]string]float64, len(leaves)*len(leaves))
			for _, a := range leaves {
				for _, b := range leaves {
					c, _, ok := s.pairContribution(TermID(a), TermID(b))
					if !ok {
						c = 0
					}
					contrib[[2]string{a, b}] = c
				}
			}
			pairScore := func(a, b string) float64 { return contrib[[2]string{a, b}] }

			check := func(as, bs []string) {
				got, _ := s.SetRelatedness(as, bs)
				var sumA, sumB float64
				for _, x := range as {
					sumA += s.ic.Of(TermID(x))
				}
				for _, x := range bs {
					sumB += s.ic.Of(TermID(x))
				}
				denom := sumA
				if sumB > denom {
					denom = sumB
				}
				want := optimalMatchSum(as, bs, pairScore) / denom
				if math.Abs(got-want) > 1e-9 {
					t.Fatalf("SetRelatedness(%v, %v) = %.9f, optimal = %.9f", as, bs, got, want)
				}
			}

			subsets := allSubsets(coveringLeaves(t))
			if !tc.full {
				var small [][]string
				for _, set := range subsets {
					if len(set) <= 4 {
						small = append(small, set)
					}
				}
				subsets = small
			}
			var checked int
			for _, as := range subsets {
				for _, bs := range subsets {
					check(as, bs)
					checked++
				}
			}
			sampled := sampledSubsetPairs(leaves, 20_000, 7, 2)
			for _, p := range sampled {
				check(p[0], p[1])
			}
			t.Logf("weighted greedy == optimal on %d exhaustive + %d sampled subset pairs", checked, len(sampled))
		})
	}
}

// SharedInformation is the raw numerator of the weighted SetRelatedness, so
// the two must agree through the max-side-IC denominator on every input.
func TestSharedInformationIsSetRelatednessNumerator(t *testing.T) {
	s := toyScorer()
	sets := [][2][]string{
		{{"error_wrapping", "db_access"}, {"error_wrapping", "caching"}},
		{{"db_access"}, {"caching", "http_call"}},
		{{"retry"}, {"db_access"}},
		{{"error_wrapping"}, {"error_wrapping"}},
	}
	for _, p := range sets {
		mass, massMatches := s.SharedInformation(p[0], p[1])
		score, scoreMatches := s.SetRelatedness(p[0], p[1])
		var sumA, sumB float64
		for _, x := range sortedUnique(p[0]) {
			sumA += s.ic.Of(TermID(x))
		}
		for _, x := range sortedUnique(p[1]) {
			sumB += s.ic.Of(TermID(x))
		}
		denom := sumA
		if sumB > denom {
			denom = sumB
		}
		if want := score * denom; math.Abs(mass-want) > 1e-9 {
			t.Errorf("SharedInformation(%v, %v) = %v, want score*denom = %v", p[0], p[1], mass, want)
		}
		if len(massMatches) != len(scoreMatches) {
			t.Errorf("SharedInformation(%v, %v) returned %d matches, SetRelatedness %d",
				p[0], p[1], len(massMatches), len(scoreMatches))
		}
	}
}

// The retrieval-side point of the raw view: two singleton sets both normalize
// to 1.0, but the rare tag carries strictly more shared information than the
// ubiquitous one.
func TestSharedInformationDiscountsUbiquity(t *testing.T) {
	s := toyScorer()
	rare, _ := s.SharedInformation([]string{"db_access"}, []string{"db_access"})
	ubiquitous, _ := s.SharedInformation([]string{"error_wrapping"}, []string{"error_wrapping"})
	if rare <= ubiquitous {
		t.Errorf("mass(db_access)=%v should exceed mass(error_wrapping)=%v", rare, ubiquitous)
	}
	if want := s.ic.Of(ConDBAccess); !closeTo(rare, want) {
		t.Errorf("singleton exact-match mass = %v, want IC(db_access) = %v", rare, want)
	}
}

// An unknown term matching itself must contribute its (unknown-sentinel) IC —
// pinned here because recomputing mass externally as Σ ic.Of(m.LCA) would hit
// Of("") instead, which is why SharedInformation exists at all.
func TestSharedInformationUnknownTermSelfMatch(t *testing.T) {
	s := toyScorer()
	mass, matches := s.SharedInformation([]string{"grpc_call"}, []string{"grpc_call"})
	if want := s.ic.Of("grpc_call"); !closeTo(mass, want) {
		t.Errorf("unknown-term self-match mass = %v, want ic.Of(grpc_call) = %v", mass, want)
	}
	if len(matches) != 1 || matches[0].A != "grpc_call" {
		t.Fatalf("matches = %+v, want the single grpc_call self-match", matches)
	}
}

func TestSharedInformationGuards(t *testing.T) {
	if mass, matches := NewScorer(Default(), nil).SharedInformation(
		[]string{"db_access"}, []string{"db_access"}); mass != 0 || matches != nil {
		t.Errorf("nil-IC SharedInformation = (%v, %v), want (0, nil)", mass, matches)
	}
	s := toyScorer()
	if mass, _ := s.SharedInformation(nil, []string{"db_access"}); mass != 0 {
		t.Errorf("empty-side SharedInformation = %v, want 0", mass)
	}
	if mass, _ := s.SharedInformation([]string{"retry"}, []string{"db_access"}); mass != 0 {
		t.Errorf("cross-branch SharedInformation = %v, want 0", mass)
	}
}

func TestWeightedSetRelatednessDeterministic(t *testing.T) {
	s := toyScorer()
	a := []string{"error_wrapping", "db_access", "concurrency"}
	b := []string{"caching", "error_wrapping", "retry"}
	firstScore, firstMatches := s.SetRelatedness(a, b)
	render := func(ms []Match) string {
		var parts []string
		for _, m := range ms {
			parts = append(parts, m.A+"~"+m.B+":"+string(m.LCA))
		}
		return strings.Join(parts, ";")
	}
	want := render(firstMatches)
	for i := 0; i < 1000; i++ {
		score, matches := s.SetRelatedness(a, b)
		if score != firstScore || render(matches) != want {
			t.Fatalf("run %d diverged: %v %q vs %v %q", i, score, render(matches), firstScore, want)
		}
	}
}

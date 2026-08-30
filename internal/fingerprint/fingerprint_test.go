package fingerprint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

const (
	srcSum = `
func Sum(nums []int) int {
	total := 0
	for _, n := range nums {
		if n > 0 {
			total += n
		}
	}
	return total
}`

	// Same logic as srcSum with every identifier renamed.
	srcSumRenamed = `
func Total(values []int) int {
	acc := 0
	for _, v := range values {
		if v > 0 {
			acc += v
		}
	}
	return acc
}`

	srcServe = `
func Serve(addr string) error {
	srv := newServer(addr)
	defer srv.Close()
	go srv.Listen()
	select {
	case <-srv.Done():
		return nil
	}
}`

	srcGetter = `
func (s *Server) Addr() string {
	return s.addr
}`
)

// build parses a single function declaration and fingerprints it.
//
// It fills WL itself, because Build deliberately does not: in production the
// parser sets it from canon's canonical tree, and this package cannot import
// canon. Bagging the parsed declaration instead is the right substitute here
// — these tests are about the scoring arithmetic over two bags, and what
// canonicalization does to a bag's contents is internal/canon's and
// internal/parser's to prove.
func build(t *testing.T, src string) Fingerprint {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "snippet.go", "package p\n"+src, 0)
	if err != nil {
		t.Fatalf("parse snippet: %v", err)
	}
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			fp := Build(fd)
			fp.WL = WLBag(fd)
			return fp
		}
	}
	t.Fatalf("no function declaration in snippet")
	return Fingerprint{}
}

// Unit tests score with nil weights — no corpus, so every label is worth 1
// and the shape component is a plain multiset Jaccard. The corpus-weighted
// path is exercised where a corpus exists: retriever, calibrate and the
// golden benchmark.
func sim(a, b Fingerprint) Breakdown { return SimilarityWith(a, b, nil, DefaultWeights()) }

func TestSimilarityIdentical(t *testing.T) {
	fp := build(t, srcSum)
	got := sim(fp, fp)
	if got.Score != 1.0 {
		t.Errorf("Score = %v, want 1.0 (breakdown %+v)", got.Score, got)
	}
	if got.SizeRatio != 1.0 {
		t.Errorf("SizeRatio = %v, want 1.0", got.SizeRatio)
	}
}

// The depth histogram records each control-flow node's entry depth: for srcSum
// the range enters at 0, the if inside it at 1, and the return at 0.
func TestDepthHistogram(t *testing.T) {
	fp := build(t, srcSum)
	want := make([]int, depthBuckets)
	want[0] = 2 // range, return
	want[1] = 1 // if inside the range
	if len(fp.Depth) != depthBuckets {
		t.Fatalf("Depth has %d buckets, want %d", len(fp.Depth), depthBuckets)
	}
	for i := range want {
		if fp.Depth[i] != want[i] {
			t.Fatalf("Depth = %v, want %v", fp.Depth, want)
		}
	}
	// A body with no control flow at all leaves the histogram empty, so two
	// such bodies compare 1.0 on the component (cosine's both-empty rule) —
	// flatness agreeing with flatness.
	if getter := build(t, srcGetter); getter.Depth[0] != 1 { // the return
		t.Errorf("getter Depth = %v, want one depth-0 entry", getter.Depth)
	}
}

// Two functions with identical token bags but different nesting were
// indistinguishable before the depth histogram: flattened tokens carry no
// depth and the flow histogram only counts kinds. Sequential ifs against
// nested ifs must now score below 1.0 — and only the Depth component moves.
func TestDepthSeparatesFlatFromNested(t *testing.T) {
	flat := build(t, `
func flat(a, b bool) int {
	x := 0
	if a {
		x = 1
	}
	if b {
		x = 2
	}
	return x
}`)
	nested := build(t, `
func nested(a, b bool) int {
	x := 0
	if a {
		if b {
			x = 2
		}
		x = 1
	}
	return x
}`)
	bd := sim(flat, nested)
	if bd.Depth >= 1.0 {
		t.Errorf("Depth = %v, want < 1.0 for different nesting", bd.Depth)
	}
	if bd.Flow < 0.999 { // cosine of equal vectors can land an ulp under 1.0
		t.Errorf("Flow = %v, want ~1.0 — same node kinds, so only Depth should separate them", bd.Flow)
	}
	if bd.Score >= 1.0 {
		t.Errorf("Score = %v, want < 1.0 (breakdown %+v)", bd.Score, bd)
	}
}

// Depths past the last bucket fold into it rather than falling off.
func TestDepthDeepTailFolds(t *testing.T) {
	fp := build(t, `
func deep(a bool) {
	if a {
		if a {
			if a {
				if a {
					if a {
						if a {
							if a {
								println()
							}
						}
					}
				}
			}
		}
	}
}`)
	if fp.Depth[depthBuckets-1] != 2 { // the ifs entered at depths 5 and 6
		t.Errorf("Depth = %v, want the two deepest ifs folded into the last bucket", fp.Depth)
	}
}

// Renaming every variable must not change the score: canonicalizing
// identifiers is the whole point of the AST token stream.
func TestSimilarityRenamedVariables(t *testing.T) {
	got := sim(build(t, srcSum), build(t, srcSumRenamed))
	if got.Score < 0.95 {
		t.Errorf("Score = %v, want >= 0.95 (breakdown %+v)", got.Score, got)
	}
}

func TestSimilarityUnrelated(t *testing.T) {
	got := sim(build(t, srcSum), build(t, srcServe))
	if got.Score >= 0.3 {
		t.Errorf("Score = %v, want < 0.3 (breakdown %+v)", got.Score, got)
	}
}

// Containment divides the shared mass by the smaller side's rather than by
// the union, so a body whose whole shape reappears inside a much longer one
// reads high on containment and low on the Jaccard. That gap is the finding
// the pair list could not previously state.
func TestContainmentSeesTheInlinedHelper(t *testing.T) {
	helper := build(t, `
func helper(nums []int) int {
	total := 0
	for _, n := range nums {
		if n > 0 {
			total += n
		}
	}
	return total
}`)
	// The same loop, inlined into a function that does a great deal more.
	host := build(t, `
func host(nums []int, names []string) (int, string) {
	total := 0
	for _, n := range nums {
		if n > 0 {
			total += n
		}
	}
	joined := ""
	for _, s := range names {
		if s != "" {
			joined += s
		}
	}
	switch {
	case total > 100:
		joined += "big"
	case total > 10:
		joined += "medium"
	}
	if joined == "" {
		joined = "none"
	}
	return total, joined
}`)
	bd := sim(helper, host)
	if bd.Containment < bd.WL {
		t.Fatalf("containment %.3f below the Jaccard %.3f — same numerator, smaller denominator",
			bd.Containment, bd.WL)
	}
	if bd.Containment-bd.WL < 0.15 {
		t.Errorf("containment %.3f vs Jaccard %.3f: the fixture no longer shows the gap",
			bd.Containment, bd.WL)
	}
	// And the thing the gap is for: containment is not folded into Score.
	if bd.Score >= bd.Containment {
		t.Errorf("Score %.3f >= containment %.3f — containment must not be blended in",
			bd.Score, bd.Containment)
	}
}

// Identical bags contain each other entirely, whatever the weighting.
func TestContainmentOfIdenticalBodies(t *testing.T) {
	fp := build(t, srcSum)
	if got := sim(fp, fp).Containment; got != 1.0 {
		t.Errorf("self-containment = %v, want 1.0", got)
	}
}

// Corpus weighting is the semantic shift: the same two bodies score
// differently depending on how ordinary their shared structure is where they
// live. Against a corpus of nothing but copies of themselves, every label is
// universal, every weight is ln(N/N) = 0, and the pair shares no information
// at all — the same 0/0 convention TrophicSimilarity uses for a pair whose
// every pattern is corpus idiom.
func TestWeightingIsCorpusRelative(t *testing.T) {
	a, b := build(t, srcSum), build(t, srcSumRenamed)
	if plain := sim(a, b).WL; plain != 1.0 {
		t.Fatalf("unweighted Jaccard of a renamed copy = %v, want 1.0", plain)
	}

	// A corpus that is nothing but these two: every label is in every bag.
	idiom := LabelWeights([][]LabelCount{a.WL, b.WL})
	if got := Similarity(a, b, idiom); got.WL != 0 || got.Containment != 0 {
		t.Errorf("in a corpus of nothing but this shape, wl = %v / containment = %v, want 0 and 0",
			got.WL, got.Containment)
	}

	// Add bodies that share none of it and the shape becomes informative again.
	corpus := LabelWeights([][]LabelCount{a.WL, b.WL,
		build(t, srcServe).WL, build(t, srcGetter).WL})
	if got := Similarity(a, b, corpus).WL; got <= 0 {
		t.Errorf("wl = %v in a mixed corpus, want > 0 — the shape is rare there", got)
	}
}

// An unseen label is treated as df 1 rather than weightless, so it lands in
// the union and depresses the score. Weightless would make it invisible,
// which is the unsafe direction — see LabelIDF.Weight.
func TestUnseenLabelsDepressRatherThanVanish(t *testing.T) {
	a, b := build(t, srcSum), build(t, srcServe)
	// Weights counted over a corpus neither body belongs to.
	foreign := LabelWeights([][]LabelCount{
		build(t, srcGetter).WL,
		build(t, `func g() { println(1) }`).WL,
	})
	got := Similarity(a, b, foreign)
	if got.WL < 0 || got.WL > 1 {
		t.Errorf("wl = %v, outside [0,1]", got.WL)
	}
	if got.WL >= sim(a, b).WL {
		t.Errorf("wl %v against a foreign corpus is not below the unweighted %v",
			got.WL, sim(a, b).WL)
	}
}

func TestSimilaritySymmetric(t *testing.T) {
	a, b := build(t, srcSum), build(t, srcServe)
	if forward, reverse := sim(a, b), sim(b, a); forward != reverse {
		t.Errorf("Similarity not symmetric: %+v vs %+v", forward, reverse)
	}
}

func TestSimilarityNoBody(t *testing.T) {
	var empty Fingerprint
	if got := sim(empty, build(t, srcSum)); got != (Breakdown{}) {
		t.Errorf("Similarity with empty fingerprint = %+v, want zero Breakdown", got)
	}
	if got := sim(empty, empty); got != (Breakdown{}) {
		t.Errorf("Similarity of two empty fingerprints = %+v, want zero Breakdown", got)
	}
}

func TestBuildNilDecl(t *testing.T) {
	if got := Build(nil); got.Nodes != 0 {
		t.Errorf("Build(nil).Nodes = %d, want 0", got.Nodes)
	}
}

func TestBuildDeterministic(t *testing.T) {
	first, second := build(t, srcSum), build(t, srcSum)
	if len(first.Shingles) != len(second.Shingles) {
		t.Fatalf("shingle count differs: %d vs %d", len(first.Shingles), len(second.Shingles))
	}
	for i := range first.Shingles {
		if first.Shingles[i] != second.Shingles[i] {
			t.Fatalf("shingle %d differs: %x vs %x", i, first.Shingles[i], second.Shingles[i])
		}
	}
}

// Trivial accessors must stay under the default --min-nodes guard, otherwise
// they pairwise match at 1.0 and flood the report.
//
// The constant tracks retriever.DefaultOptions().MinNodes, which moved 12 → 18
// when the shape channel started indexing WL labels. It is spelled out rather
// than imported because fingerprint must not depend on retriever, and the
// accessor this asserts about is well under either value — the test is about
// the body being trivial, not about the exact floor.
const defaultMinNodes = 18

func TestBuildTrivialAccessorIsSmall(t *testing.T) {
	if nodes := build(t, srcGetter).Nodes; nodes >= defaultMinNodes {
		t.Errorf("trivial accessor has %d nodes, expected fewer than the default min-nodes of %d",
			nodes, defaultMinNodes)
	}
}

func TestTypeStringsSeparatesInputsFromOutputs(t *testing.T) {
	a := build(t, "func A(err error) {}")
	b := build(t, "func B() error { return nil }")
	if got := jaccardStrings(a.Types, b.Types); got != 0 {
		t.Errorf("jaccardStrings = %v, want 0: a param error must not match a returned error", got)
	}
}

// FlowLabels is index-aligned with the slot constants; pin the two ends so a
// reordering cannot slip through silently.
func TestFlowLabelsAlignWithSlots(t *testing.T) {
	if FlowLabels[flowIf] != "if" {
		t.Errorf("FlowLabels[flowIf] = %q, want if", FlowLabels[flowIf])
	}
	if FlowLabels[flowFuncLit] != "funclit" {
		t.Errorf("FlowLabels[flowFuncLit] = %q, want funclit", FlowLabels[flowFuncLit])
	}
}

// SimilarityWith under the default blend is Similarity, bit for bit: the
// seam exists for measurement and must not move a single score.
func TestSimilarityWithDefaultIsIdentical(t *testing.T) {
	fps := []Fingerprint{build(t, srcSum), build(t, srcSumRenamed), build(t, srcServe), build(t, srcGetter)}
	for _, a := range fps {
		for _, b := range fps {
			if Similarity(a, b, nil) != SimilarityWith(a, b, nil, DefaultWeights()) {
				t.Fatalf("SimilarityWith(DefaultWeights) differs from Similarity")
			}
		}
	}
	if s := DefaultWeights().Sum(); s != weightWL+weightFlow+weightDepth+weightSignature {
		t.Errorf("DefaultWeights().Sum() = %v", s)
	}
}

// Scaled moves one component and renormalizes the rest, keeping the total.
func TestWeightsScaled(t *testing.T) {
	w := DefaultWeights().Scaled(0, 0.5)
	if w.WL != 0.30 {
		t.Errorf("WL = %v, want 0.30", w.WL)
	}
	if d := w.Sum() - 1.0; d > 1e-12 || d < -1e-12 {
		t.Errorf("Sum = %v, want 1.0", w.Sum())
	}
	// The other three scale uniformly: flow/sig ratio unchanged.
	if r := w.Flow / w.Signature; r < 0.2/0.15-1e-9 || r > 0.2/0.15+1e-9 {
		t.Errorf("flow/sig ratio moved: %v", r)
	}
	if a, b := sim(build(t, srcSum), build(t, srcServe)), SimilarityWith(build(t, srcSum), build(t, srcServe), nil, w); a.Score == b.Score {
		t.Error("a different blend left the score unchanged")
	}
}

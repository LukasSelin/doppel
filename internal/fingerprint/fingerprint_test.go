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
func build(t *testing.T, src string) Fingerprint {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "snippet.go", "package p\n"+src, 0)
	if err != nil {
		t.Fatalf("parse snippet: %v", err)
	}
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			return Build(fd)
		}
	}
	t.Fatalf("no function declaration in snippet")
	return Fingerprint{}
}

func TestSimilarityIdentical(t *testing.T) {
	fp := build(t, srcSum)
	got := Similarity(fp, fp)
	if got.Score != 1.0 {
		t.Errorf("Score = %v, want 1.0 (breakdown %+v)", got.Score, got)
	}
	if got.SizeRatio != 1.0 {
		t.Errorf("SizeRatio = %v, want 1.0", got.SizeRatio)
	}
}

// Renaming every variable must not change the score: canonicalizing
// identifiers is the whole point of the AST token stream.
func TestSimilarityRenamedVariables(t *testing.T) {
	got := Similarity(build(t, srcSum), build(t, srcSumRenamed))
	if got.Score < 0.95 {
		t.Errorf("Score = %v, want >= 0.95 (breakdown %+v)", got.Score, got)
	}
}

func TestSimilarityUnrelated(t *testing.T) {
	got := Similarity(build(t, srcSum), build(t, srcServe))
	if got.Score >= 0.3 {
		t.Errorf("Score = %v, want < 0.3 (breakdown %+v)", got.Score, got)
	}
}

func TestSimilaritySymmetric(t *testing.T) {
	a, b := build(t, srcSum), build(t, srcServe)
	if forward, reverse := Similarity(a, b), Similarity(b, a); forward != reverse {
		t.Errorf("Similarity not symmetric: %+v vs %+v", forward, reverse)
	}
}

func TestSimilarityNoBody(t *testing.T) {
	var empty Fingerprint
	if got := Similarity(empty, build(t, srcSum)); got != (Breakdown{}) {
		t.Errorf("Similarity with empty fingerprint = %+v, want zero Breakdown", got)
	}
	if got := Similarity(empty, empty); got != (Breakdown{}) {
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
func TestBuildTrivialAccessorIsSmall(t *testing.T) {
	if nodes := build(t, srcGetter).Nodes; nodes >= 12 {
		t.Errorf("trivial accessor has %d nodes, expected fewer than the default min-nodes of 12", nodes)
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

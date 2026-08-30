package analyzer

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
)

// parseTwo parses one source file and returns the two units it declares, in
// declaration order.
//
// Real parsing rather than hand-built ASTs, deliberately: Explain reads
// Fingerprint.WL and CanonRules, both of which parser fills during the parse,
// so a hand-built CodeUnit would test the sentence against inputs no pipeline
// ever produces.
func parseTwo(t *testing.T, src string) (parser.CodeUnit, parser.CodeUnit) {
	t.Helper()
	units, err := parser.ParseSource("explain_test_input.go", []byte(src))
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("want 2 units, got %d", len(units))
	}
	return units[0], units[1]
}

// TestExplainIdenticalAfterRenameAndReorder is the gate: two bodies that
// differ only by their identifiers and by the order of one commutative
// operand must be reported as identical, naming exactly those two
// normalizations.
//
// The two functions are built so that only one of them needs the reorder —
// alpha's operands are already in canonical order, beta's are not — which is
// what makes this a test of "fired on either side" rather than on both.
func TestExplainIdenticalAfterRenameAndReorder(t *testing.T) {
	a, b := parseTwo(t, `package p

func alpha(first int, second int) int {
	total := first * second
	return first + total
}

func beta(left int, right int) int {
	product := right * left
	return left + product
}
`)

	if got, want := Explain(a, b), "identical after rename, commutative-reorder"; got != want {
		t.Fatalf("Explain = %q, want %q", got, want)
	}
	// Symmetric: the sentence is about the pair, not about an order.
	if got := Explain(b, a); got != "identical after rename, commutative-reorder" {
		t.Errorf("Explain(b, a) = %q, want the same sentence", got)
	}
}

// TestExplainResidualNamesTheExtraStatement pins the residual's headline: an
// extra defer is named as a defer, and it leads, ahead of the call and
// selector it necessarily drags in with it.
func TestExplainResidualNamesTheExtraStatement(t *testing.T) {
	a, b := parseTwo(t, `package p

func withCleanup(c closer) error {
	defer c.Close()
	return nil
}

func withoutCleanup(c closer) error {
	return nil
}
`)

	got := Explain(a, b)
	if !strings.HasPrefix(got, "differs by one extra defer") {
		t.Fatalf("Explain = %q, want it to lead with the extra defer", got)
	}
	if got != Explain(b, a) {
		t.Errorf("Explain is not symmetric: %q vs %q", got, Explain(b, a))
	}
}

// TestExplainSameKindsDifferentArrangement covers the case the kind counts
// cannot see: both bodies use exactly the same statements, arranged
// differently. The h=0 multisets agree, so there is nothing to count, and the
// sentence has to say so rather than claiming a difference of zero.
func TestExplainSameKindsDifferentArrangement(t *testing.T) {
	a, b := parseTwo(t, `package p

func guardedFirst(ok bool) {
	if ok {
		first()
	}
	second()
}

func guardedSecond(ok bool) {
	if ok {
		second()
	}
	first()
}
`)

	got := Explain(a, b)
	if !strings.HasPrefix(got, "same statement kinds,") {
		t.Fatalf("Explain = %q, want a same-kinds sentence", got)
	}
}

// TestExplainIdenticalWithoutNormalization is the tier-one case where no rule
// fired: two bodies that were already canonical and already the same.
func TestExplainIdenticalWithoutNormalization(t *testing.T) {
	a, b := parseTwo(t, `package p

func one() error {
	return nil
}

func two() error {
	return nil
}
`)

	if got, want := Explain(a, b), "identical structure, no normalization needed"; got != want {
		t.Fatalf("Explain = %q, want %q", got, want)
	}
}

// TestExplainNoBody covers a declaration the fingerprint calls empty — an
// assembly stub or an interface-only declaration. There is nothing to explain
// and the sentence must not claim there is.
func TestExplainNoBody(t *testing.T) {
	units, err := parser.ParseSource("explain_test_input.go", []byte(`package p

func stub()

func real() error {
	return nil
}
`))
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	var stub, real parser.CodeUnit
	for _, u := range units {
		if u.Name == "stub" {
			stub = u
		} else {
			real = u
		}
	}
	if got, want := Explain(stub, real), "no body to compare"; got != want {
		t.Fatalf("Explain = %q, want %q", got, want)
	}
}

// TestExplainDeterministic runs the same pair repeatedly. The residual is
// summarised out of a map, so a sentence that depended on map order would be
// a report that differed between runs.
func TestExplainDeterministic(t *testing.T) {
	a, b := parseTwo(t, `package p

func left(xs []int) int {
	sum := 0
	for _, x := range xs {
		if x > 0 {
			sum += x
		}
	}
	defer trace()
	return sum
}

func right(ys []int) int {
	total := 0
	for _, y := range ys {
		total += y
	}
	return total
}
`)

	want := Explain(a, b)
	for i := 0; i < 200; i++ {
		if got := Explain(a, b); got != want {
			t.Fatalf("run %d: Explain = %q, want %q", i, got, want)
		}
	}
	if want == "" {
		t.Fatal("Explain returned an empty sentence")
	}
}

// TestExplainCapsTheSentence checks the bound: a residual with many differing
// kinds says three of them and counts the rest, rather than printing a bag.
func TestExplainCapsTheSentence(t *testing.T) {
	a, b := parseTwo(t, `package p

func busy(xs []int, c closer) (int, error) {
	defer c.Close()
	sum := 0
	for _, x := range xs {
		switch {
		case x > 0:
			sum += x
		default:
			sum -= x
		}
	}
	go background()
	select {
	case <-done():
		return sum, nil
	}
}

func quiet(xs []int) (int, error) {
	return len(xs), nil
}
`)

	got := Explain(a, b)
	if !strings.HasPrefix(got, "differs by ") {
		t.Fatalf("Explain = %q, want a residual", got)
	}
	if n := strings.Count(got, " extra "); n > explainMaxKinds {
		t.Fatalf("Explain names %d kinds, want at most %d: %q", n, explainMaxKinds, got)
	}
	if !strings.Contains(got, "more kind") {
		t.Fatalf("Explain = %q, want it to count the kinds it did not name", got)
	}
}

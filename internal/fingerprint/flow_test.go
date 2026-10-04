package fingerprint

import (
	"reflect"
	"strings"
	"testing"
)

func stepsOf(t *testing.T, src string) string {
	t.Helper()
	var parts []string
	for _, s := range Build(parseFn(t, src)).Steps {
		parts = append(parts, s.String())
	}
	return strings.Join(parts, ", ")
}

func TestFlowSteps(t *testing.T) {
	src := `func f(c *Conn, v any) (*Result, error) {
	if err := c.Open(); err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer c.Close()
	for _, r := range c.Rows() {
		if r == nil {
			continue
		}
	}
	s := v.(Stringer)
	_ = s
	cmd.Flags().StringVar(&s, "x", "", "")
	_ = []string{}
	return &Result{}, nil
}`
	want := "if, call Open, call Errorf, return, defer, call Close, loop, call Rows, if, branch continue, " +
		"assert Stringer, call Flags, call StringVar, new []string, new Result, return"
	if got := stepsOf(t, src); got != want {
		t.Errorf("steps:\n got %s\nwant %s", got, want)
	}
	fp := Build(parseFn(t, src))
	for _, w := range []string{"as:Stringer", "new:Result", "out:*Result", "in:*Conn"} {
		found := false
		for _, s := range fp.Touched {
			if s == w {
				found = true
			}
		}
		if !found {
			t.Errorf("Touched %v lacks %q", fp.Touched, w)
		}
	}
}

func TestFlowSimilarityIdenticalAndRenamed(t *testing.T) {
	a := Build(parseFn(t, "func f(c *Conn) error {\n\tif err := c.Open(); err != nil {\n\t\treturn err\n\t}\n\treturn nil\n}"))
	b := Build(parseFn(t, "func g(x *Conn) error {\n\tif e := x.Open(); e != nil {\n\t\treturn e\n\t}\n\treturn nil\n}"))
	fs := FlowSimilarity(a, b)
	if fs.Steps != 1 || fs.Types != 1 || fs.Same != len(a.Steps) || fs.Retargeted != 0 {
		t.Errorf("renamed copy: %+v", fs)
	}
}

// The case the view exists for: the same steps in a different order share
// every histogram bucket and nearly every WL label, and must not read alike
// here.
func TestFlowSimilaritySeesOrder(t *testing.T) {
	a := Build(parseFn(t, "func f(x int) int {\n\tvalidate(x)\n\ty := fetch(x)\n\treturn store(y)\n}"))
	b := Build(parseFn(t, "func f(x int) int {\n\ty := fetch(x)\n\tstore(y)\n\treturn validate(x)\n}"))
	fs := FlowSimilarity(a, b)
	if bd := Similarity(a, b, nil); bd.Flow != 1 {
		t.Fatalf("fixture premise: flow histograms should agree, got %.2f", bd.Flow)
	}
	if fs.Steps >= 0.8 {
		t.Errorf("reordered body reads %.2f on flow steps, want well below 1: %+v", fs.Steps, fs)
	}
}

// Same kind, different target counts half: a mirror method (Get against
// Delete) is not the same logic as a clone, nor wholly different logic.
func TestFlowSimilarityRetargeted(t *testing.T) {
	a := Build(parseFn(t, "func f(s *S, k string) error {\n\treturn s.Get(k)\n}"))
	b := Build(parseFn(t, "func f(s *S, k string) error {\n\treturn s.Delete(k)\n}"))
	fs := FlowSimilarity(a, b)
	if fs.Same != 1 || fs.Retargeted != 1 || fs.Steps != 0.75 {
		t.Errorf("retargeted: %+v", fs)
	}
}

func TestFlowSimilaritySymmetricAndAlignAgrees(t *testing.T) {
	srcs := []string{
		"func f(xs []int) int {\n\tn := 0\n\tfor _, x := range xs {\n\t\tn += g(x)\n\t}\n\treturn n\n}",
		"func f(m map[string]int) {\n\tfor k := range m {\n\t\tif k == \"\" {\n\t\t\tdelete(m, k)\n\t\t}\n\t}\n}",
		"func f(c *Conn) error {\n\tdefer c.Close()\n\tif err := c.Flush(); err != nil {\n\t\treturn err\n\t}\n\treturn c.Sync()\n}",
		"func f() {}",
		// Two retargets or one exact match weigh the same; the tie must not
		// be decided by which side came first.
		"func f() {\n\ta()\n\tb()\n}",
		"func f() {\n\tb()\n\tc()\n}",
	}
	for _, x := range srcs {
		for _, y := range srcs {
			a, b := Build(parseFn(t, x)), Build(parseFn(t, y))
			ab, ba := FlowSimilarity(a, b), FlowSimilarity(b, a)
			if ab.Steps != ba.Steps || ab.Types != ba.Types || ab.Same != ba.Same || ab.Retargeted != ba.Retargeted {
				t.Errorf("asymmetric: %+v vs %+v", ab, ba)
			}
			var same, re int
			rows := FlowAlign(a, b)
			seenA, seenB := 0, 0
			for _, r := range rows {
				switch r.Match {
				case 2:
					same++
				case 1:
					re++
				}
				if r.A >= 0 {
					seenA++
				}
				if r.B >= 0 {
					seenB++
				}
			}
			if same != ab.Same || re != ab.Retargeted {
				t.Errorf("FlowAlign matched %d/%d, FlowSimilarity %d/%d", same, re, ab.Same, ab.Retargeted)
			}
			if seenA != len(a.Steps) || seenB != len(b.Steps) {
				t.Errorf("alignment does not cover every step: %d/%d of %d/%d", seenA, seenB, len(a.Steps), len(b.Steps))
			}
		}
	}
}

func TestFlowSimilarityEmpty(t *testing.T) {
	a := Build(parseFn(t, "func f() {}"))
	if fs := FlowSimilarity(a, a); fs.Steps != 1 {
		t.Errorf("two step-free bodies: %+v", fs)
	}
	b := Build(parseFn(t, "func f() { g() }"))
	if fs := FlowSimilarity(a, b); fs.Steps != 0 {
		t.Errorf("empty against non-empty: %+v", fs)
	}
	if !reflect.DeepEqual(FlowAlign(a, a), []FlowRow(nil)) {
		t.Errorf("empty alignment should be nil")
	}
}

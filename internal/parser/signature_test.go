package parser

import (
	"reflect"
	"testing"

	"github.com/LukasSelin/doppel/internal/fingerprint"
)

// Signature is rendered text: parameter and result types in order, names
// dropped, one entry per declared name, results parenthesized when present.
func TestSignatureRendersTypes(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"func f() {}", "()"},
		{"func f(nums []int) int { return 0 }", "([]int) (int)"},
		{"func f(a, b int, s string) {}", "(int, int, string)"},
		{"func f(xs ...string) (n int, err error) { return 0, nil }", "(...string) (int, error)"},
		{"func (s *Server) Start(ctx context.Context) error { return nil }", "(context.Context) (error)"},
		{"func (p *Pool[T]) Go(f func(T) error) {}", "(func(T) error)"},
		{"func f(m map[string][]int) (chan<- int, bool) { return nil, false }", "(map[string][]int) (chan<- int, bool)"},
	}
	for _, tc := range cases {
		units, err := ParseSource("sig.go", []byte("package p\nimport \"context\"\ntype Server struct{}\ntype Pool[T any] struct{}\n"+tc.src))
		if err != nil || len(units) != 1 {
			t.Fatalf("%s: parse: %v (%d units)", tc.src, err, len(units))
		}
		if got := units[0].Signature; got != tc.want {
			t.Errorf("%s: Signature = %q, want %q", tc.src, got, tc.want)
		}
	}
}

// Names inside a function-typed parameter are not part of its type, so a copy
// that renames them — or drops them, or renames the outer parameter too — is
// the same signature and scores 1.00 on the signature component. They used to
// be kept, and a rename-only copy of such a function read code-shape 0.85.
func TestSignatureIgnoresNamesInFuncTypes(t *testing.T) {
	src := []byte(`package p

import "flag"

type Command struct{}

func (c *Command) A(n func(f *flag.FlagSet, name string) flag.NormalizedName) (done func() (ok bool)) {
	return nil
}

func (c *Command) B(m func(g *flag.FlagSet, s string) flag.NormalizedName) (fin func() (b bool)) {
	return nil
}

func (c *Command) C(func(*flag.FlagSet, string) flag.NormalizedName) func() bool {
	return nil
}
`)
	units, err := ParseSource("cmd.go", src)
	if err != nil || len(units) != 3 {
		t.Fatalf("parse: %v (%d units)", err, len(units))
	}
	const wantSig = "(func(*flag.FlagSet, string) flag.NormalizedName) (func() bool)"
	wantTypes := []string{"in:func(*flag.FlagSet, string) flag.NormalizedName", "out:func() bool"}
	for _, u := range units {
		if u.Signature != wantSig {
			t.Errorf("%s: Signature = %q, want %q", u.Name, u.Signature, wantSig)
		}
		if !reflect.DeepEqual(u.Fingerprint.Types, wantTypes) {
			t.Errorf("%s: Fingerprint.Types = %v, want %v", u.Name, u.Fingerprint.Types, wantTypes)
		}
	}
	for _, other := range units[1:] {
		bd := fingerprint.Similarity(units[0].Fingerprint, other.Fingerprint, nil)
		if bd.Signature != 1.0 || bd.Score != 1.0 {
			t.Errorf("A vs %s = %+v, want signature and code-shape exactly 1.0", other.Name, bd)
		}
	}
}

// The signature string is presentation only. Fingerprint.Types — what the
// similarity score actually reads — is computed from the AST independently
// and must not move with it.
func TestSignatureDoesNotChangeFingerprint(t *testing.T) {
	src := []byte("package p\nfunc Sum(nums []int) int {\n\ttotal := 0\n\tfor _, n := range nums {\n\t\ttotal += n\n\t}\n\treturn total\n}\n")
	units, err := ParseSource("sum.go", src)
	if err != nil || len(units) != 1 {
		t.Fatalf("parse: %v", err)
	}
	u := units[0]
	if u.Signature == "" || u.Signature == " " {
		t.Fatalf("signature still empty: %q", u.Signature)
	}
	if want := []string{"in:[]int", "out:int"}; !reflect.DeepEqual(u.Fingerprint.Types, want) {
		t.Errorf("Fingerprint.Types = %v, want %v", u.Fingerprint.Types, want)
	}
	// nil weights: no corpus to ask, so every WL label is worth 1 and the
	// shape component is a plain multiset Jaccard. A fingerprint against
	// itself is still exactly 1.0 under either weighting.
	if bd := fingerprint.Similarity(u.Fingerprint, u.Fingerprint, nil); bd.Signature != 1.0 || bd.Score != 1.0 {
		t.Errorf("self-similarity = %+v, want exact 1.0", bd)
	}
}

package parser

import "testing"

func unitOf(t *testing.T, name, src string) CodeUnit {
	t.Helper()
	units, err := ParseSource(name, []byte(src))
	if err != nil || len(units) != 1 {
		t.Fatalf("parse %s: %v (%d units)", name, err, len(units))
	}
	return units[0]
}

// Per-architecture kernels are near-identical by necessity and no build
// compiles two of them together, so they are never a merge candidate; a plain
// file stays pairable with each of them, and the zero value — every language
// without conditional compilation — keeps the old answer.
func TestSameBuildUnitBuildTargets(t *testing.T) {
	const body = "package p\n\nfunc Sum(x []float32) (s float32) {\n\tfor _, v := range x {\n\t\ts += v\n\t}\n\treturn\n}\n"
	amd64 := unitOf(t, "kern_amd64.go", body)
	arm64 := unitOf(t, "kern_arm64.go", body)
	generic := unitOf(t, "kern_generic.go", "//go:build !amd64 && !arm64\n\n"+body)
	plain := unitOf(t, "util.go", body)
	linux := unitOf(t, "sys_linux.go", body)

	cases := []struct {
		name string
		a, b CodeUnit
		want bool
	}{
		{"amd64 vs arm64", amd64, arm64, false},
		{"amd64 vs generic fallback", amd64, generic, false},
		{"amd64 vs plain", amd64, plain, true},
		{"linux vs amd64", linux, amd64, true},
		{"same target", amd64, amd64, true},
		{"zero value", CodeUnit{Lang: "go"}, CodeUnit{Lang: "go"}, true},
	}
	for _, c := range cases {
		if got := SameBuildUnit(c.a, c.b); got != c.want {
			t.Errorf("%s: SameBuildUnit = %v, want %v", c.name, got, c.want)
		}
		if got := SameBuildUnit(c.b, c.a); got != c.want {
			t.Errorf("%s (swapped): SameBuildUnit = %v, want %v", c.name, got, c.want)
		}
	}
}

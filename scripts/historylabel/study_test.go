package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestParseReport(t *testing.T) {
	report := `
Code Similarity Report
======================
Functions analyzed: 3  |  Threshold: 0.40

#1    code-shape: 1.0000
  A  a/x.go:10                                    a.F
       sig: ()
  B  a/x.go:20                                    a.G
       sig: ()
  kind: mirror operations — F and G on T are one operation run in opposite directions
  explain: identical

#2    code-shape: 0.5000
  A  a/x.go:10                                    a.F
  B  b/y.go:3                                     b.*T.H
`
	p := filepath.Join(t.TempDir(), "r.txt")
	if err := os.WriteFile(p, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := parseReport(p)
	if err != nil {
		t.Fatal(err)
	}
	want := []rankedPair{
		{rank: 1, aFile: "a/x.go", aLine: 10, bFile: "a/x.go", bLine: 20, kind: "mirror operations"},
		{rank: 2, aFile: "a/x.go", aLine: 10, bFile: "b/y.go", bLine: 3},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d pairs, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pair %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestBucketAndLocality(t *testing.T) {
	for lines, want := range map[int]int{0: 0, 1: 0, 2: 1, 3: 1, 4: 2, 63: 5, 64: 6, 1000: 6} {
		if got := bucketOf(lines); got != want {
			t.Errorf("bucketOf(%d) = %d, want %d", lines, got, want)
		}
	}
	u := func(file string) *sunit { return &sunit{file: file, dir: filepathDir(file), top: topOf(file)} }
	cases := []struct{ a, b, want string }{
		{"x.go", "x.go", locFile},
		{"x.go", "y.go", locPackage},
		{"a/b/x.go", "a/c/y.go", locTop},
		{"a/x.go", "b/x.go", locCross},
		{"x.go", "a/x.go", locCross},
	}
	for _, c := range cases {
		if got := localityOf(u(c.a), u(c.b)); got != c.want {
			t.Errorf("localityOf(%s, %s) = %s, want %s", c.a, c.b, got, c.want)
		}
	}
}

func filepathDir(f string) string { return filepath.ToSlash(filepath.Dir(f)) }

// TestEstimate pins the ratio-of-means arithmetic and the bootstrap's
// determinism.
func TestEstimate(t *testing.T) {
	one := func(c int) studyPair { return studyPair{Cochanges: c, EditsA: 1, EditsB: 1} }
	var sets []matchedSet
	for i := 0; i < 20; i++ {
		s := matchedSet{corpus: "x", treated: one(i % 2)}
		s.controls = []studyPair{one(0), one(0), one(i % 4 / 3)}
		sets = append(sets, s)
	}
	e := estimateOf(sets, metrics[0].f, "t")
	if e.treated != 0.5 || math.Abs(e.ctrl-5.0/60) > 1e-12 {
		t.Fatalf("means %v / %v, want 0.5 / %v", e.treated, e.ctrl, 5.0/60)
	}
	if math.Abs(e.ratio-6) > 1e-9 {
		t.Fatalf("ratio %v, want 6", e.ratio)
	}
	if again := estimateOf(sets, metrics[0].f, "t"); again != e {
		t.Fatalf("bootstrap not deterministic: %+v vs %+v", again, e)
	}
	if !(e.lo <= e.ratio && e.ratio <= e.hi) {
		t.Fatalf("CI [%v, %v] excludes the point estimate %v", e.lo, e.hi, e.ratio)
	}
}

func TestHistName(t *testing.T) {
	for in, want := range map[string]string{
		"*Stack[T].Pop":    "*Stack.Pop",
		"Map[K, V].Get":    "Map.Get",
		"*Command.Execute": "*Command.Execute",
		"plainFunc":        "plainFunc",
	} {
		if got := histName(in); got != want {
			t.Errorf("histName(%q) = %q, want %q", in, got, want)
		}
	}
}

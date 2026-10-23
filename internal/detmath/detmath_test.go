package detmath

import (
	"hash/fnv"
	"math"
	"math/rand/v2"
	"runtime"
	"testing"
)

// ratios is the shape every IDF and IC takes, ln(n/df): every n/df for
// 1 ≤ df ≤ n ≤ max.
func ratios(max int, fn func(x float64)) {
	for n := 1; n <= max; n++ {
		for df := 1; df <= n; df++ {
			fn(float64(n) / float64(df))
		}
	}
}

// normals draws positive normal floats with a uniform exponent, so every
// binade from the smallest normal to the largest finite is reached.
func normals(count int, fn func(x float64)) {
	rng := rand.New(rand.NewPCG(1, 2))
	for range count {
		exp := 1 + rng.Uint64N(2046)
		fn(math.Float64frombits(exp<<52 | rng.Uint64()&(1<<52-1)))
	}
}

// exponents draws Exp inputs over the whole finite domain and, densely, the
// range doppel actually reaches (fitness gaps, PMI, −MinIDF).
func exponents(count int, fn func(x float64)) {
	rng := rand.New(rand.NewPCG(3, 4))
	for range count {
		fn(float64(rng.Float64()*1460) - 750)
		fn(float64(rng.Float64()*80) - 40)
	}
}

var specials = []float64{
	0, math.Copysign(0, -1), -1, 1, 2, 0.5, math.E, math.Ln2, -math.Ln2,
	math.Inf(1), math.Inf(-1), math.NaN(), math.MaxFloat64, -math.MaxFloat64,
	0x1p-1022, 709.4, 709.5, 709.78, 709.79, -708.4, -744, -745.2, -746, -1e10,
	1e-300, -1e-300, 0x1p-28, -0x1p-28,
}

func same(a, b float64) bool {
	return math.Float64bits(a) == math.Float64bits(b) || (math.IsNaN(a) && math.IsNaN(b))
}

// TestLogMatchesAmd64 is the claim that replacing math.Log moved no amd64
// result: amd64's assembly is this algorithm, so the two agree on every
// positive normal input. Subnormals are excluded on purpose (see Log).
func TestLogMatchesAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skipf("math.Log is assembly only on amd64; on %s it is the fused Go version this replaces", runtime.GOARCH)
	}
	bad := 0
	check := func(x float64) {
		if got, want := Log(x), math.Log(x); !same(got, want) {
			if bad++; bad <= 5 {
				t.Errorf("Log(%x) = %x, math.Log = %x", math.Float64bits(x), math.Float64bits(got), math.Float64bits(want))
			}
		}
	}
	for _, x := range specials {
		if x == 0 || math.Abs(x) >= 0x1p-1022 || math.IsNaN(x) {
			check(x)
		}
	}
	ratios(2000, check)
	normals(2_000_000, check)
	if bad > 0 {
		t.Fatalf("%d inputs differ from math.Log", bad)
	}
}

// TestExpMatchesAmd64 is the same claim for Exp, which can only hold on a
// CPU where math.Exp takes its FMA branch. A CPU without FMA takes the plain
// branch instead — expPlain is that branch, so the test can say which it saw
// rather than fail on hardware the claim does not cover.
func TestExpMatchesAmd64(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skipf("math.Exp is amd64's assembly only on amd64; on %s it is another algorithm", runtime.GOARCH)
	}
	var bad, plain, n int
	check := func(x float64) {
		n++
		want := math.Exp(x)
		if same(expPlain(x), want) {
			plain++
		}
		if got := Exp(x); !same(got, want) {
			if bad++; bad <= 5 {
				t.Logf("Exp(%v) = %x, math.Exp = %x", x, math.Float64bits(got), math.Float64bits(want))
			}
		}
	}
	for _, x := range specials {
		check(x)
	}
	exponents(1_000_000, check)
	switch {
	case bad == 0:
	case plain == n:
		t.Skip("this CPU has no FMA, so math.Exp takes its plain branch and Exp is not expected to equal it")
	default:
		t.Fatalf("%d of %d inputs differ from math.Exp", bad, n)
	}
}

// expPlain is the non-FMA branch of amd64's math.Exp assembly, used only to
// recognise a CPU that takes it.
func expPlain(x float64) float64 {
	const (
		log2e    = 1.4426950408889634073599246810018920
		ln2U     = 0.69314718055966295651160180568695068359375
		ln2L     = 0.28235290563031577122588448175013436025525412068e-12
		overflow = 7.09782712893384e+02
	)
	switch {
	case math.IsNaN(x) || math.IsInf(x, 1):
		return x
	case math.IsInf(x, -1):
		return 0
	case x > overflow:
		return math.Inf(1)
	}
	t := log2e * x
	if t < -1076 {
		return 0
	}
	k := int(math.RoundToEven(t))
	fk := float64(k)
	x -= float64(fk * ln2U)
	x -= float64(fk * ln2L)
	x *= 0.0625
	p := 2.4801587301587301587e-5
	for _, c := range []float64{1.9841269841269841270e-4, 1.3888888888888888889e-3,
		8.3333333333333333333e-3, 4.1666666666666666667e-2, 1.6666666666666666667e-1, 0.5, 1} {
		p = float64(p*x) + c
	}
	x = float64(x * p)
	for range 4 {
		x = float64(x * (x + 2))
	}
	x += 1
	e := k + 0x3FF
	switch {
	case e <= 0:
		if e < -52 {
			return 0
		}
		x *= math.Float64frombits(uint64(e+0x3FE) << 52)
		e = 1
	case e >= 0x7FF:
		return math.Inf(1)
	}
	return x * math.Float64frombits(uint64(e)<<52)
}

// TestGolden pins the results themselves, on whatever architecture runs it:
// the hashes were taken on amd64, so an arm64 run passing this is the
// cross-architecture claim checked directly rather than by argument.
func TestGolden(t *testing.T) {
	for _, c := range []struct {
		name string
		fn   func(float64) float64
		x    float64
		want uint64
	}{
		{"Log", Log, 7.0 / 3.0, 0x3feb1d10670aae9a},
		{"Log", Log, 1411.0 / 2.0, 0x401a3c520e80dd01},
		{"Log", Log, 1, 0},
		{"Exp", Exp, -1, 0x3fd78b56362cef38},
		{"Exp", Exp, 1, 0x4005bf0a8b145769},
		{"Exp", Exp, -0.925, 0x3fd960c55130de3f},
	} {
		if got := c.fn(c.x); math.Float64bits(got) != c.want {
			t.Errorf("%s(%v) = %#x, want %#x", c.name, c.x, math.Float64bits(got), c.want)
		}
	}

	h := fnv.New64a()
	var buf [8]byte
	put := func(v float64) {
		b := math.Float64bits(v)
		for i := range buf {
			buf[i] = byte(b >> (8 * i))
		}
		h.Write(buf[:])
	}
	ratios(1000, func(x float64) { put(Log(x)) })
	normals(200_000, func(x float64) { put(Log(x)) })
	if got, want := h.Sum64(), uint64(0x433a19983461452b); got != want {
		t.Errorf("Log hash = %#x, want %#x", got, want)
	}
	h.Reset()
	for _, x := range specials {
		put(Exp(x))
	}
	exponents(200_000, func(x float64) { put(Exp(x)) })
	if got, want := h.Sum64(), uint64(0x12c5e98eb2f3e893); got != want {
		t.Errorf("Exp hash = %#x, want %#x", got, want)
	}
}

// Package detmath is the floating-point arithmetic that every architecture
// doppel ships for agrees on, bit for bit: Log and Exp, plus the rule the rest
// of the module follows for plain arithmetic.
//
// The standard library's two do not agree. math.Log is assembly on amd64 and
// Go on arm64, where the compiler fuses the Go version's multiply-adds into
// FMA instructions; on the integer ratios an IDF is made of (n/df, n ≤ 3000)
// about 1.2% came out a bit apart. math.Exp is assembly on both, as two
// different algorithms, and amd64's own picks between an FMA branch and a
// plain one at run time by CPU. Every corpus statistic is a Log, so a run on
// an arm64 Mac and a run on amd64 Linux disagreed in the last bit of a score —
// enough to put a pair either side of a threshold, and to make a baseline
// written on one machine incomparable with a run on the other.
//
// Neither function here is a new approximation. Log is math/log.go's
// algorithm, which amd64's assembly implements operation for operation with no
// fusion; written with every product rounded explicitly, it is that same
// sequence of IEEE operations on every architecture, so on amd64 it returns
// exactly what math.Log returned. Exp is amd64's FMA branch, with each fused
// step written as math.FMA: one rounding by definition, a single instruction
// on arm64 and on every amd64 since 2013, and a correctly rounded software
// fallback elsewhere. On any amd64 with FMA — every machine the committed
// examples were generated on — it returns exactly what math.Exp returned.
// TestLogMatchesAmd64 and TestExpMatchesAmd64 pin both claims where they can
// be checked, and TestGolden pins the results themselves on every
// architecture CI runs.
//
// The rule for everything else: Go may fuse x*y + z into one FMA "possibly
// across statements" (the spec's words), and arm64 does where amd64 does not.
// An explicit float64(x*y) rounds the product and forbids the fusion, so a
// product that feeds an addition or subtraction in anything that reaches a
// score, a threshold or a snapshot is written that way. `task fma-check` (and
// CI) disassembles an arm64 build and fails on any fused instruction in this
// module's code outside this package, whose fused steps are explicit.
//
// detmath imports nothing from this module.
package detmath

import "math"

// Log returns the natural logarithm of x: math/log.go's algorithm (FreeBSD's
// e_log.c), with every intermediate product converted to float64 so that no
// compiler may fuse it. The special cases are math.Log's.
//
// On amd64 this is bit-identical to math.Log for every positive normal input,
// because amd64's assembly is this sequence of operations. Subnormal inputs
// are where they part: the assembly reads the exponent field directly and
// does not normalise first, which math.Frexp does. No count ratio or
// probability doppel takes a log of is subnormal.
func Log(x float64) float64 {
	const (
		Ln2Hi = 6.93147180369123816490e-01 /* 3fe62e42 fee00000 */
		Ln2Lo = 1.90821492927058770002e-10 /* 3dea39ef 35793c76 */
		L1    = 6.666666666666735130e-01   /* 3FE55555 55555593 */
		L2    = 3.999999999940941908e-01   /* 3FD99999 9997FA04 */
		L3    = 2.857142874366239149e-01   /* 3FD24924 94229359 */
		L4    = 2.222219843214978396e-01   /* 3FCC71C5 1D8E78AF */
		L5    = 1.818357216161805012e-01   /* 3FC74664 96CB03DE */
		L6    = 1.531383769920937332e-01   /* 3FC39A09 D078C69F */
		L7    = 1.479819860511658591e-01   /* 3FC2F112 DF3E5244 */
	)

	switch {
	case math.IsNaN(x) || math.IsInf(x, 1):
		return x
	case x < 0:
		return math.NaN()
	case x == 0:
		return math.Inf(-1)
	}

	// reduce
	f1, ki := math.Frexp(x)
	if f1 < math.Sqrt2/2 {
		f1 = float64(f1 * 2)
		ki--
	}
	f := f1 - 1
	k := float64(ki)

	// compute, one rounding per operation
	s := f / (2 + f)
	s2 := s * s
	s4 := s2 * s2
	t1 := float64(s2 * (L1 + float64(s4*(L3+float64(s4*(L5+float64(s4*L7)))))))
	t2 := float64(s4 * (L2 + float64(s4*(L4+float64(s4*L6)))))
	R := t1 + t2
	hfsq := float64(0.5 * f * f)
	return float64(k*Ln2Hi) - ((hfsq - (float64(s*(hfsq+R)) + float64(k*Ln2Lo))) - f)
}

// Exp returns e**x: the FMA branch of amd64's math.Exp assembly (after
// Shibata's SLEEF), with each fused step a math.FMA and each product that
// feeds an addition rounded explicitly. The special cases are that
// assembly's, including its overflow bound, which returns +Inf a little below
// math.MaxFloat64's logarithm.
func Exp(x float64) float64 {
	const (
		log2e    = 1.4426950408889634073599246810018920        // 1/ln 2
		ln2U     = 0.69314718055966295651160180568695068359375 // upper half of ln 2
		ln2L     = 0.28235290563031577122588448175013436025525412068e-12
		overflow = 7.09782712893384e+02

		c2 = 0.5
		c3 = 1.6666666666666666667e-1
		c4 = 4.1666666666666666667e-2
		c5 = 8.3333333333333333333e-3
		c6 = 1.3888888888888888889e-3
		c7 = 1.9841269841269841270e-4
		c8 = 2.4801587301587301587e-5
	)

	switch {
	case math.IsNaN(x) || math.IsInf(x, 1):
		return x
	case math.IsInf(x, -1):
		return 0
	case x > overflow:
		return math.Inf(1)
	}

	// The exponent, rounded to nearest even as CVTSD2SL does. Below -1075.5
	// the result underflows to zero whatever the reduction says, which also
	// keeps the conversion inside int32 where the assembly's would saturate.
	t := log2e * x
	if t < -1076 {
		return 0
	}
	k := int(math.RoundToEven(t))
	fk := float64(k)

	// reduce: x - k·ln 2 in two fused steps, then by 16 for the series
	x = math.FMA(-ln2U, fk, x)
	x = math.FMA(-ln2L, fk, x)
	x *= 0.0625

	// Taylor series of e**x − 1, then squared back up four times through
	// (e**x − 1)·(e**x + 1) = e**2x − 1
	p := c8
	p = math.FMA(p, x, c7)
	p = math.FMA(p, x, c6)
	p = math.FMA(p, x, c5)
	p = math.FMA(p, x, c4)
	p = math.FMA(p, x, c3)
	p = math.FMA(p, x, c2)
	p = math.FMA(p, x, 1)
	x = float64(x * p)
	x = float64(x * (x + 2))
	x = float64(x * (x + 2))
	x = float64(x * (x + 2))
	x = math.FMA(x, x+2, 1)

	// return x · 2**k, through a subnormal in two steps as the assembly does
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

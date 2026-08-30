package parser

import "testing"

// wlRenameSrc is one function written twice: the same work with every
// parameter and local renamed, and the bindings declared in the same order so
// that canon's positional names line up. Nothing else differs — the
// canonicalization-does-more claim has its own test below.
const wlRenameSrc = `package p

func Sum(nums []int) int {
	total := 0
	for _, n := range nums {
		if n > 0 {
			total += n
		}
	}
	return total
}

func Accumulate(values []int) int {
	acc := 0
	for _, v := range values {
		if v > 0 {
			acc += v
		}
	}
	return acc
}
`

// TestWLBagRenameInvariantThroughCanon is the first T2 gate, end to end on
// the path production actually takes: parse, canonicalize, bag. Two functions
// identical up to consistent renaming canonicalize to the same tree and so
// must carry the same bag, counts included.
func TestWLBagRenameInvariantThroughCanon(t *testing.T) {
	units, err := ParseSource("p.go", []byte(wlRenameSrc))
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}
	a, b := units[0].Fingerprint.WL, units[1].Fingerprint.WL
	if len(a) == 0 {
		t.Fatal("no WL bag on a function with a body")
	}
	if len(a) != len(b) {
		t.Fatalf("bags differ in size: %d vs %d", len(a), len(b))
	}
	for label, n := range a {
		if b[label] != n {
			t.Errorf("label %016x: %s has %d, %s has %d",
				label, units[0].Name, n, units[1].Name, b[label])
		}
	}
}

// TestWLBagPresentAndBounded: every unit with a body carries a bag, and one
// without does not — the same rule the zero Fingerprint follows.
func TestWLBagPresentAndBounded(t *testing.T) {
	units, err := ParseSource("p.go", []byte(canonSrc))
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range units {
		bag := u.Fingerprint.WL
		if u.Fingerprint.Nodes == 0 {
			if bag != nil {
				t.Errorf("%s: body-less unit carries a bag", u.Name)
			}
			continue
		}
		if len(bag) == 0 {
			t.Errorf("%s: no WL bag", u.Name)
		}
	}
}

// TestWLBagBuiltFromCanonicalTree: the bag is canon's tree, not the source
// tree. `x = x + 1` and `x++` are different code and the token stream keeps
// them apart, but RuleIncDec makes them one canonical shape — so the bags
// must agree where the shingles do not.
func TestWLBagBuiltFromCanonicalTree(t *testing.T) {
	src := `package p

func A(n int) int {
	i := 0
	for i < n {
		i = i + 1
	}
	return i
}

func B(limit int) int {
	k := 0
	for k < limit {
		k++
	}
	return k
}
`
	units, err := ParseSource("p.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}
	a, b := units[0].Fingerprint.WL, units[1].Fingerprint.WL
	for label, n := range a {
		if b[label] != n {
			t.Fatalf("label %016x: A has %d, B has %d — the bag is not being built from the canonical tree",
				label, n, b[label])
		}
	}
	if len(a) != len(b) {
		t.Fatalf("bags differ in size: %d vs %d", len(a), len(b))
	}

	// And the guard that this proves something: the un-canonicalized token
	// shingles do still tell the two apart.
	if equalUint64(units[0].Fingerprint.Shingles, units[1].Fingerprint.Shingles) {
		t.Error("shingles agree too, so the test proves nothing about canonicalization")
	}
}

func equalUint64(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

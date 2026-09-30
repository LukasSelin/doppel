package fingerprint

import (
	"reflect"
	"testing"
)

// A gather and its scatter, the byte-shuffle pair doppel reported at 1.00:
// the same statement with the strided index and the running one on opposite
// sides. Identifiers collapse to ID, so under the
// production recurrence the assignment's children are one unordered pair and
// the two bodies carry the same bag.
const (
	gather = `func f(dst, src []byte, n, m int) {
	k := 0
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			dst[k] = src[j*n+i]
			k++
		}
	}
}`
	scatter = `func f(dst, src []byte, n, m int) {
	k := 0
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			dst[j*n+i] = src[k]
			k++
		}
	}
}`
)

func TestWLRolesNoneIsProduction(t *testing.T) {
	root := parseFunc(t, gather)
	if !reflect.DeepEqual(WLBagWith(root, WLOptions{}), wlBagOf(root)) {
		t.Fatal("WLOptions{} must be byte-identical to the production bag")
	}
}

func TestWLRolesSeparateMirrorImages(t *testing.T) {
	g, s := parseFunc(t, gather), parseFunc(t, scatter)
	if !reflect.DeepEqual(wlBagOf(g), wlBagOf(s)) {
		t.Fatal("premise: the production bag cannot tell a gather from its scatter")
	}
	for _, mode := range []WLRoles{WLRolesAssign, WLRolesDirected} {
		a, b := WLBagWith(g, WLOptions{Roles: mode}), WLBagWith(s, WLOptions{Roles: mode})
		if reflect.DeepEqual(a, b) {
			t.Errorf("mode %d: gather and scatter still carry one bag", mode)
		}
		if j, _ := WLOverlap(a, b, nil); j >= 1 {
			t.Errorf("mode %d: jaccard %.3f, want below 1", mode, j)
		}
	}
}

// What the production recurrence is for must survive: statement order in a
// block is still a multiset under every mode.
func TestWLRolesKeepStatementOrderFree(t *testing.T) {
	a := parseFunc(t, "func f(x, y int) int {\n\tx++\n\ty--\n\treturn x\n}")
	b := parseFunc(t, "func f(x, y int) int {\n\ty--\n\tx++\n\treturn x\n}")
	for _, mode := range []WLRoles{WLRolesNone, WLRolesAssign, WLRolesDirected} {
		if !reflect.DeepEqual(WLBagWith(a, WLOptions{Roles: mode}), WLBagWith(b, WLOptions{Roles: mode})) {
			t.Errorf("mode %d: reordering two independent statements changed the bag", mode)
		}
	}
}

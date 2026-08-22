package analyzer

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
)

func method(pkg, file, recv, name, sig string) parser.CodeUnit {
	return parser.CodeUnit{Package: pkg, File: file, ReceiverType: recv, Name: recv + "." + name, Signature: sig}
}

func fn(pkg, file, name string) parser.CodeUnit {
	return parser.CodeUnit{Package: pkg, File: file, Name: name, Signature: "()"}
}

func TestInterfaceImpl(t *testing.T) {
	aws := method("aws", "carrier/aws/aws.go", "*AWS", "Validate", "(context.Context) (error)")
	gcp := method("gcp", "carrier/gcp/gcp.go", "*GCP", "Validate", "(context.Context) (error)")

	k := InterfaceImpl(aws, gcp)
	if k == nil || k.Kind != KindInterfaceImpl {
		t.Fatalf("sibling Validate methods not labeled: %+v", k)
	}
	if k.Method != "Validate" || k.Signature != "(context.Context) (error)" || k.Relation != RelationSiblings {
		t.Errorf("note = %+v", k)
	}
	if len(k.Receivers) != 2 || k.Receivers[0] != "*AWS" || len(k.Packages) != 2 {
		t.Errorf("receivers/packages = %v / %v", k.Receivers, k.Packages)
	}

	// Generic receivers compare by their bare type name.
	pool := method("pool", "pool/pool.go", "*Pool[T]", "Go", "(func(T) error)")
	set := method("pool", "pool/set.go", "*Set[T]", "Go", "(func(T) error)")
	if k := InterfaceImpl(pool, set); k == nil || k.Relation != RelationSamePackage {
		t.Errorf("generic receivers in one package: %+v", k)
	}

	// Two types that happen to share a name in sibling packages are two
	// types: moby's ipvlan and macvlan drivers both implement Join on their
	// own *driver.
	ipvlan := method("ipvlan", "drivers/ipvlan/join.go", "*driver", "Join", "(string) (error)")
	macvlan := method("macvlan", "drivers/macvlan/join.go", "*driver", "Join", "(string) (error)")
	if k := InterfaceImpl(ipvlan, macvlan); k == nil || k.Relation != RelationSiblings {
		t.Errorf("same-named receivers in sibling packages: %+v", k)
	}
	if k := ClassifyPair(ipvlan, macvlan, 0.9); k == nil || k.Kind != KindInterfaceImpl {
		t.Errorf("same-named receivers must not read as a fork: %+v", k)
	}

	// Unrelated directories still label, and say so.
	far := method("x", "internal/x/x.go", "*X", "Validate", "(context.Context) (error)")
	if k := InterfaceImpl(aws, far); k == nil || k.Relation != RelationUnrelated {
		t.Errorf("unrelated packages: %+v", k)
	}

	negatives := []struct {
		name string
		a, b parser.CodeUnit
	}{
		{"same receiver", aws, method("aws", "carrier/aws/aws.go", "AWS", "Validate", "(context.Context) (error)")},
		{"signature differs", aws, method("gcp", "carrier/gcp/gcp.go", "*GCP", "Validate", "() (error)")},
		{"method name differs", aws, method("gcp", "carrier/gcp/gcp.go", "*GCP", "Check", "(context.Context) (error)")},
		{"one plain function", aws, fn("gcp", "carrier/gcp/gcp.go", "Validate")},
		{"empty signature", method("a", "a/a.go", "*A", "F", ""), method("b", "b/b.go", "*B", "F", "")},
	}
	for _, tc := range negatives {
		if k := InterfaceImpl(tc.a, tc.b); k != nil {
			t.Errorf("%s: labeled %+v, want nil", tc.name, k)
		}
	}
}

func TestFork(t *testing.T) {
	old := method("template", "tpl/texttemplate/exec.go", "*state", "evalCallOld", "()")
	cur := method("template", "tpl/texttemplate/hugo_template.go", "*state", "evalCall", "()")
	k := Fork(old, cur, 0.85)
	if k == nil || k.Kind != KindFork || k.Method != "evalCall" || k.Relation != RelationSamePackage {
		t.Fatalf("evalCall fork: %+v", k)
	}
	if len(k.Names) != 2 || k.Names[0] != "*state.evalCallOld" {
		t.Errorf("names = %v", k.Names)
	}

	// The receiver axis: same method, stem-sharing receivers.
	v1 := method("scrape", "scrape/scrape.go", "*scrapeLoopAppender", "append", "([]byte) (error)")
	v2 := method("scrape", "scrape/v2.go", "*scrapeLoopAppenderV2", "append", "([]byte) (error)")
	if k := Fork(v1, v2, 0.83); k == nil || k.Method != "scrapeLoopAppender" {
		t.Errorf("receiver-axis fork: %+v", k)
	}

	// Plain functions in sibling packages.
	a := fn("md", "doc/md/gen.go", "GenTree")
	b := fn("rst", "doc/rst/gen.go", "GenTreeLegacy")
	if k := Fork(a, b, 0.7); k == nil || k.Relation != RelationSiblings {
		t.Errorf("sibling-package fork: %+v", k)
	}

	negatives := []struct {
		name  string
		a, b  parser.CodeUnit
		score float64
	}{
		{"below the shape floor", old, cur, ForkShapeFloor - 0.01},
		{"no marker", fn("b", "b/t.go", "decodeToml"), fn("b", "b/y.go", "decodeYAML"), 1.0},
		{"different names, no stem", fn("t", "tsdb/w.go", "loadWAL"), fn("t", "tsdb/w.go", "loadWBL"), 0.74},
		{"unrelated directories", fn("a", "x/a/a.go", "runOld"), fn("b", "y/b/b.go", "run"), 0.9},
		{"method vs function", old, fn("template", "tpl/texttemplate/f.go", "evalCall"), 0.9},
		{"both axes differ", v1, method("scrape", "scrape/v2.go", "*scrapeLoopAppenderV2", "appendOld", "()"), 0.9},
	}
	for _, tc := range negatives {
		if k := Fork(tc.a, tc.b, tc.score); k != nil {
			t.Errorf("%s: labeled %+v, want nil", tc.name, k)
		}
	}
}

// Both rules hold for the v1/v2 appenders; the fork is the claim a reader
// acts on, so it wins.
func TestForkWinsOverInterfaceImpl(t *testing.T) {
	v1 := method("scrape", "scrape/scrape.go", "*scrapeLoopAppender", "append", "([]byte) (error)")
	v2 := method("scrape", "scrape/v2.go", "*scrapeLoopAppenderV2", "append", "([]byte) (error)")
	if InterfaceImpl(v1, v2) == nil {
		t.Fatal("fixture broken: expected the interface rule to hold too")
	}
	if k := ClassifyPair(v1, v2, 0.83); k == nil || k.Kind != KindFork {
		t.Errorf("ClassifyPair = %+v, want the fork", k)
	}
	if k := ClassifyPair(v1, v2, 0.3); k == nil || k.Kind != KindInterfaceImpl {
		t.Errorf("below the floor the interface rule should remain: %+v", k)
	}
}

func TestClassifyFamily(t *testing.T) {
	units := []parser.CodeUnit{
		method("aws", "carrier/aws/aws.go", "*AWS", "Validate", "(context.Context) (error)"),
		method("gcp", "carrier/gcp/gcp.go", "*GCP", "Validate", "(context.Context) (error)"),
		method("azure", "carrier/azure/az.go", "*Azure", "Validate", "(context.Context) (error)"),
		fn("dhl", "carrier/dhl/c.go", "compensationContext"),
		fn("ups", "carrier/ups/c.go", "compensationContext"),
		fn("dsv", "carrier/dsv/c.go", "compensationContext"),
		method("t", "tpl/t/a.go", "*state", "evalCall", "()"),
		method("t", "tpl/t/b.go", "*state", "evalCallOld", "()"),
		method("t", "tpl/t/c.go", "*state", "evalCallLegacy", "()"),
	}
	if k := ClassifyFamily(units, []int{0, 1, 2}, 0.8); k == nil || k.Kind != KindInterfaceImpl ||
		k.Relation != RelationSiblings || len(k.Receivers) != 3 || len(k.Packages) != 3 {
		t.Errorf("Validate family: %+v", k)
	}
	// Three same-named plain functions are neither methods nor stem-distinct:
	// no kind, which is the honest answer.
	if k := ClassifyFamily(units, []int{3, 4, 5}, 0.9); k != nil {
		t.Errorf("plain same-name functions labeled %+v, want nil", k)
	}
	if k := ClassifyFamily(units, []int{6, 7, 8}, 0.8); k == nil || k.Kind != KindFork || k.Method != "evalCall" || len(k.Names) != 3 {
		t.Errorf("evalCall family: %+v", k)
	}
	// A family whose members fork on different stems is not one fork.
	mixed := append(units, method("t", "tpl/t/d.go", "*state", "renderOld", "()"), method("t", "tpl/t/e.go", "*state", "render", "()"))
	if k := ClassifyFamily(mixed, []int{6, 7, 9, 10}, 0.8); k != nil {
		t.Errorf("mixed stems labeled %+v, want nil", k)
	}
}

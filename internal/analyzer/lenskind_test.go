package analyzer

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/syntax"
)

func methodUnit(pkg, recv, name string) parser.CodeUnit {
	return parser.CodeUnit{Package: pkg, ReceiverType: recv, Name: recv + "." + name, File: pkg + "/x.go"}
}

func TestMirror(t *testing.T) {
	cases := []struct {
		a, b parser.CodeUnit
		want bool
	}{
		{methodUnit("zarr", "ShuffleCodec", "EncodeBytes"), methodUnit("zarr", "ShuffleCodec", "DecodeBytes"), true},
		{methodUnit("zarr", "*MemoryStore", "Get"), methodUnit("zarr", "*MemoryStore", "Delete"), true},
		{methodUnit("array", "*T", "MinOver"), methodUnit("array", "T", "MaxOver"), true}, // pointer and value receiver are one type
		{methodUnit("zarr", "*A", "Encode"), methodUnit("zarr", "*B", "Decode"), false},   // different receivers
		{methodUnit("a", "*T", "Encode"), methodUnit("b", "*T", "Decode"), false},         // different packages
		{methodUnit("zarr", "*T", "EncodeBytes"), methodUnit("zarr", "*T", "DecodeArray"), false},
		{methodUnit("zarr", "*T", "Get"), methodUnit("zarr", "*T", "GetRange"), false},
		{parser.CodeUnit{Package: "p", Name: "encode"}, parser.CodeUnit{Package: "p", Name: "decode"}, false}, // plain functions
		// Lifecycle verbs: an operation and the one that undoes it.
		{methodUnit("bridge", "*driver", "CreateEndpoint"), methodUnit("bridge", "*driver", "DeleteEndpoint"), true},
		{methodUnit("plugin", "*pluginRouter", "pullPlugin"), methodUnit("plugin", "*pluginRouter", "pushPlugin"), true},
		{methodUnit("libnetwork", "*Endpoint", "sbJoin"), methodUnit("libnetwork", "*Endpoint", "sbLeave"), true},
		{methodUnit("hugolib", "pageTree", "IsAncestor"), methodUnit("hugolib", "pageTree", "IsDescendant"), true},
		{methodUnit("fs", "*T", "Mount"), methodUnit("fs", "*T", "Unmount"), true},
		// One resource's verbs, not one operation run backwards.
		{methodUnit("cluster", "*Cluster", "CreateService"), methodUnit("cluster", "*Cluster", "UpdateService"), false},
		{methodUnit("daemon", "*Daemon", "ContainerRestart"), methodUnit("daemon", "*Daemon", "ContainerStop"), false},
	}
	for _, c := range cases {
		if got := Mirror(c.a, c.b) != nil; got != c.want {
			t.Errorf("Mirror(%s, %s) = %v, want %v", c.a.Name, c.b.Name, got, c.want)
		}
	}
	k := Mirror(methodUnit("zarr", "*S", "Read"), methodUnit("zarr", "*S", "Write"))
	if k.Kind != KindMirror || k.Names[0] != "Read" || k.Names[1] != "Write" || k.Receivers[0] != "*S" {
		t.Errorf("note = %+v", k)
	}
}

// callTree builds a canonical tree returning helper(name).
func callTree(helper, name string) *syntax.Node {
	call := (&syntax.Node{Kind: syntax.KindCall}).
		Add(syntax.RoleFun, &syntax.Node{Kind: syntax.KindIdent, Label: helper}).
		Add(syntax.RoleArg, &syntax.Node{Kind: syntax.KindLit, Label: "STRING", Text: name})
	ret := (&syntax.Node{Kind: syntax.KindReturn}).Add(syntax.RoleResult, call)
	return (&syntax.Node{Kind: syntax.KindBlock}).Add(syntax.RoleNone, ret)
}

func TestThinWrappers(t *testing.T) {
	unit := func(nodes int, tree *syntax.Node) parser.CodeUnit {
		return parser.CodeUnit{Package: "p", Name: "f", File: "p/x.go", Canonical: tree,
			Fingerprint: fingerprint.Fingerprint{Nodes: nodes}}
	}
	label, comment := unit(12, callTree("labelAxiom", `"label"`)), unit(12, callTree("labelAxiom", `"comment"`))
	bags := BuildThinVocab([]parser.CodeUnit{label, comment, unit(ThinNodes+1, callTree("x", "y"))})
	if bags[2] != nil {
		t.Error("a body above ThinNodes got a vocab bag")
	}
	helper := []string{"owl.labelAxiom"}
	ctx := PairContext{ResolvedA: helper, ResolvedB: helper, VocabA: bags[0], VocabB: bags[1]}
	k := ThinWrappers(label, comment, ctx)
	if k == nil || k.Kind != KindThinWrappers || len(k.Shared) != 1 || k.Shared[0] != "owl.labelAxiom" {
		t.Fatalf("different names over a shared helper: %+v", k)
	}
	// Identical vocabulary is a clone, not a wrapper pair: the vocab guard spares it.
	if ThinWrappers(label, label, PairContext{ResolvedA: helper, ResolvedB: helper, VocabA: bags[0], VocabB: bags[0]}) != nil {
		t.Error("identical vocabulary labelled thin wrappers")
	}
	if ThinWrappers(label, comment, PairContext{ResolvedA: helper, ResolvedB: []string{"owl.other"}, VocabA: bags[0], VocabB: bags[1]}) != nil {
		t.Error("no shared helper labelled thin wrappers")
	}
	if ThinWrappers(label, comment, PairContext{ResolvedA: helper, ResolvedB: helper}) != nil {
		t.Error("labelled without vocab bags")
	}
}

func TestDifferentCalls(t *testing.T) {
	u := func(calls ...string) parser.CodeUnit {
		return parser.CodeUnit{Package: "p", File: "p/x.go", Callees: calls}
	}
	if k := DifferentCalls(u("a", "b", "c", "d"), u("e", "f", "g", "a")); k == nil || k.Overlap >= CallOverlapFloor {
		t.Errorf("1 of 7 calls shared: %+v", k)
	}
	if DifferentCalls(u("a", "b"), u("a", "c")) != nil { // 1/3
		t.Error("a third of calls shared was labelled")
	}
	if DifferentCalls(u(), u()) != nil {
		t.Error("two bodies that call nothing were labelled")
	}
	if DifferentCalls(u("a", "a", "a", "b"), u("a", "c")) != nil { // distinct: {a,b} vs {a,c} = 1/3
		t.Error("repeated calls were counted as distinct")
	}
}

func TestClassifyPairInPrecedence(t *testing.T) {
	// An interface implementation keeps its label even when the calls differ.
	a := methodUnit("x", "*A", "Validate")
	b := methodUnit("y", "*B", "Validate")
	a.Signature, b.Signature = "() (error)", "() (error)"
	a.Callees, b.Callees = []string{"p"}, []string{"q"}
	if k := ClassifyPairIn(a, b, 0.5, ForkShapeFloor, PairContext{}); k == nil || k.Kind != KindInterfaceImpl {
		t.Errorf("got %+v, want interface implementations", k)
	}
	// A mirror pair with different calls is a mirror first.
	m1, m2 := methodUnit("z", "*S", "Encode"), methodUnit("z", "*S", "Decode")
	m1.Callees, m2.Callees = []string{"p"}, []string{"q"}
	if k := ClassifyPairIn(m1, m2, 0.9, ForkShapeFloor, PairContext{}); k == nil || k.Kind != KindMirror {
		t.Errorf("got %+v, want mirror operations", k)
	}
}

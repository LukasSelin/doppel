package ontology

import "testing"

func TestDefaultHasOneRootPerKind(t *testing.T) {
	o := Default()
	want := map[Kind]TermID{
		KindEntity:   EntEntity,
		KindRelation: RelRelation,
		KindConcept:  ConConcept,
		KindRole:     RoleRole,
	}
	for kind, wantRoot := range want {
		got, ok := o.Root(kind)
		if !ok {
			t.Errorf("kind %s has no root", kind)
			continue
		}
		if got != wantRoot {
			t.Errorf("root of %s = %q, want %q", kind, got, wantRoot)
		}
		if !o.terms[got].Abstract {
			t.Errorf("root %q should be abstract", got)
		}
	}
}

func TestGet(t *testing.T) {
	o := Default()
	tests := []struct {
		id           TermID
		wantKind     Kind
		wantParent   TermID
		wantAbstract bool
	}{
		{ConHTTPCall, KindConcept, ConRemoteIO, false},
		{ConRemoteIO, KindConcept, ConIOOperation, true},
		{ConIOOperation, KindConcept, ConConcept, true},
		{ConConcept, KindConcept, "", true},
		{ConErrorWrapping, KindConcept, ConErrorHandling, false},
		{RoleUtility, KindRole, RoleRole, false},
		{EntMethod, KindEntity, EntCallable, false},
		{RelCalls, KindRelation, RelRelation, false},
	}
	for _, tt := range tests {
		got, ok := o.Get(tt.id)
		if !ok {
			t.Errorf("Get(%q) not found", tt.id)
			continue
		}
		if got.Kind != tt.wantKind {
			t.Errorf("%q kind = %s, want %s", tt.id, got.Kind, tt.wantKind)
		}
		if got.Parent != tt.wantParent {
			t.Errorf("%q parent = %q, want %q", tt.id, got.Parent, tt.wantParent)
		}
		if got.Abstract != tt.wantAbstract {
			t.Errorf("%q abstract = %t, want %t", tt.id, got.Abstract, tt.wantAbstract)
		}
	}
	if _, ok := o.Get("no_such_term"); ok {
		t.Error("Get on an unknown ID reported found")
	}
}

// The nine concept leaves are the tagger's tags. Their IDs are the tool's
// output, so a change here is a change to every report and every config a user
// has written.
func TestConceptLeavesAreTheNineTags(t *testing.T) {
	want := []TermID{
		ConRetry, ConHTTPCall, ConDBAccess, ConValidation, ConMapping,
		ConTransaction, ConCaching, ConConcurrency, ConErrorWrapping,
	}
	got := map[TermID]bool{}
	for _, term := range Default().TermsOfKind(KindConcept) {
		if !term.Abstract {
			got[term.ID] = true
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d concrete concepts, want %d", len(got), len(want))
	}
	for _, id := range want {
		if !got[id] {
			t.Errorf("concrete concept %q is missing", id)
		}
	}
}

func TestChildrenAreSorted(t *testing.T) {
	o := Default()
	tests := []struct {
		parent TermID
		want   []TermID
	}{
		{ConDataStoreAccess, []TermID{ConCaching, ConDBAccess, ConTransaction}},
		{ConIOOperation, []TermID{ConDataStoreAccess, ConRemoteIO}},
		{ConRemoteIO, []TermID{ConHTTPCall}},
		{ConHTTPCall, nil},
		{RoleRole, []TermID{RoleLeaf, RoleOrchestrator, RolePassthrough, RoleUtility}},
	}
	for _, tt := range tests {
		got := o.Children(tt.parent)
		if len(got) != len(tt.want) {
			t.Errorf("Children(%q) = %v, want %v", tt.parent, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("Children(%q) = %v, want %v", tt.parent, got, tt.want)
				break
			}
		}
	}
}

func TestWeight(t *testing.T) {
	o := Default()
	if got := o.Weight(RelCalls); got != 0.225 {
		t.Errorf("Weight(calls) = %v, want 0.225", got)
	}
	if got := o.Weight(ConHTTPCall); got != 0 {
		t.Errorf("Weight of a non-relation = %v, want 0", got)
	}
	if got := o.Weight("no_such_term"); got != 0 {
		t.Errorf("Weight of an unknown term = %v, want 0", got)
	}
}

func TestEntityKindOf(t *testing.T) {
	tests := []struct {
		recv string
		want TermID
	}{
		{"", EntFunction},
		{"Server", EntMethod},
		{"*Server", EntMethod},
	}
	for _, tt := range tests {
		if got := EntityKindOf(tt.recv); got != tt.want {
			t.Errorf("EntityKindOf(%q) = %q, want %q", tt.recv, got, tt.want)
		}
	}
}

// The parser names methods "*Server.Start", star included, so a value receiver
// and a pointer receiver on one type arrive as different strings.
func TestReceiverRelatedness(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want float64
	}{
		{"both plain functions", "", "", 1.0},
		{"same receiver", "*Server", "*Server", 1.0},
		{"pointer and value receiver on one type", "Server", "*Server", 1.0},
		{"different receivers", "*Server", "*Client", 0.5},
		{"function and method", "", "*Server", 0.0},
		{"method and function", "*Server", "", 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReceiverRelatedness(tt.a, tt.b); got != tt.want {
				t.Errorf("ReceiverRelatedness(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
			if got := ReceiverRelatedness(tt.b, tt.a); got != tt.want {
				t.Errorf("ReceiverRelatedness is asymmetric on (%q, %q)", tt.a, tt.b)
			}
		})
	}
}

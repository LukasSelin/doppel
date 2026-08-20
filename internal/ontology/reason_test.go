package ontology

import (
	"math"
	"testing"
)

func closeTo(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

// The values the taxonomy is documented against. They are what the shape of the
// tree is for, so a reorganisation that changes them is a scoring change and
// has to be argued for rather than absorbed.
func TestRelatednessPinnedExamples(t *testing.T) {
	o := Default()
	tests := []struct {
		name string
		a, b TermID
		want float64
	}{
		{"identical", ConHTTPCall, ConHTTPCall, 1.0},
		{"siblings under data_store_access", ConDBAccess, ConCaching, 2.0 / 3.0},
		{"cousins under io_operation", ConHTTPCall, ConDBAccess, 1.0 / 3.0},
		{"different branches", ConHTTPCall, ConRetry, 0.0},
		{"shallower siblings", ConMapping, ConValidation, 0.5},
		{"uneven depth siblings", ConConcurrency, ConRetry, 0.4},
		{"leaf against its own parent", ConHTTPCall, ConRemoteIO, 0.8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := o.Relatedness(tt.a, tt.b)
			if !closeTo(got, tt.want) {
				t.Errorf("Relatedness(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
			if rev := o.Relatedness(tt.b, tt.a); !closeTo(rev, got) {
				t.Errorf("Relatedness is asymmetric on (%q, %q): %v vs %v", tt.a, tt.b, got, rev)
			}
		})
	}
}

func TestRelatednessGuards(t *testing.T) {
	o := Default()
	tests := []struct {
		name string
		a, b TermID
		want float64
	}{
		{"empty left", "", ConHTTPCall, 0},
		{"empty right", ConHTTPCall, "", 0},
		{"both empty", "", "", 0},
		{"unknown term", "grpc_call", ConHTTPCall, 0},
		// A tag the ontology has not learned about yet must still match its own
		// twin, or adding a tagger rule before a concept term would silently
		// drop identical-tag credit to zero.
		{"unknown term against itself", "grpc_call", "grpc_call", 1},
		{"across kinds", ConHTTPCall, RoleLeaf, 0},
		{"root against a leaf", ConConcept, ConHTTPCall, 0},
		{"abstract siblings", ConRemoteIO, ConDataStoreAccess, 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := o.Relatedness(tt.a, tt.b); !closeTo(got, tt.want) {
				t.Errorf("Relatedness(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestLCA(t *testing.T) {
	o := Default()
	tests := []struct {
		a, b   TermID
		want   TermID
		wantOK bool
	}{
		{ConDBAccess, ConCaching, ConDataStoreAccess, true},
		{ConHTTPCall, ConDBAccess, ConIOOperation, true},
		{ConHTTPCall, ConRetry, ConConcept, true},
		{ConHTTPCall, ConHTTPCall, ConHTTPCall, true},
		{ConHTTPCall, ConRemoteIO, ConRemoteIO, true},
		{RoleLeaf, RoleUtility, RoleRole, true},
		{ConHTTPCall, RoleLeaf, "", false},
		{ConHTTPCall, "grpc_call", "", false},
	}
	for _, tt := range tests {
		got, ok := o.LCA(tt.a, tt.b)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("LCA(%q, %q) = (%q, %t), want (%q, %t)", tt.a, tt.b, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestDepthAndAncestors(t *testing.T) {
	o := Default()
	depths := map[TermID]int{
		ConConcept: 0, ConIOOperation: 1, ConRemoteIO: 2, ConHTTPCall: 3,
		ConDBAccess: 3, ConMapping: 2, ConRetry: 3, ConErrorWrapping: 2,
	}
	for id, want := range depths {
		if got := o.Depth(id); got != want {
			t.Errorf("Depth(%q) = %d, want %d", id, got, want)
		}
	}
	if got := o.Depth("no_such_term"); got != -1 {
		t.Errorf("Depth of an unknown term = %d, want -1", got)
	}

	want := []TermID{ConRemoteIO, ConIOOperation, ConConcept}
	got := o.Ancestors(ConHTTPCall)
	if len(got) != len(want) {
		t.Fatalf("Ancestors(http_call) = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Ancestors(http_call) = %v, want %v (nearest first)", got, want)
		}
	}
	if a := o.Ancestors(ConConcept); len(a) != 0 {
		t.Errorf("Ancestors of a root = %v, want none", a)
	}
}

func TestIsA(t *testing.T) {
	o := Default()
	tests := []struct {
		child, ancestor TermID
		want            bool
	}{
		{ConHTTPCall, ConIOOperation, true},
		{ConHTTPCall, ConConcept, true},
		{ConCaching, ConDataStoreAccess, true},
		{ConCaching, ConRemoteIO, false},
		{ConHTTPCall, ConHTTPCall, false}, // a term is not its own ancestor
		{ConIOOperation, ConHTTPCall, false},
		{ConRetry, ConControlFlow, true},
	}
	for _, tt := range tests {
		if got := o.IsA(tt.child, tt.ancestor); got != tt.want {
			t.Errorf("IsA(%q, %q) = %t, want %t", tt.child, tt.ancestor, got, tt.want)
		}
	}
}

func TestSetRelatedness(t *testing.T) {
	o := Default()
	tests := []struct {
		name        string
		a, b        []string
		want        float64
		wantMatches int
	}{
		{"identical singletons", []string{"retry"}, []string{"retry"}, 1.0, 1},
		{"identical pairs", []string{"retry", "http_call"}, []string{"http_call", "retry"}, 1.0, 2},
		{"siblings", []string{"db_access"}, []string{"caching"}, 2.0 / 3.0, 1},
		{"cousins", []string{"http_call"}, []string{"db_access"}, 1.0 / 3.0, 1},
		{"unrelated", []string{"retry"}, []string{"db_access"}, 0.0, 0},
		{"subset", []string{"validation", "db_access"}, []string{"validation"}, 0.5, 1},
		{"both empty", nil, nil, 0.0, 0},
		{"one empty", []string{"retry"}, nil, 0.0, 0},
		{"duplicates collapse", []string{"retry", "retry"}, []string{"retry"}, 1.0, 1},
		{"empty strings ignored", []string{"", "retry"}, []string{"retry", ""}, 1.0, 1},
		// The case that makes the matcher global rather than per-term. Walking
		// one side greedily lets caching consume transaction, dropping this to
		// 0.33 -- below what plain exact matching already gives.
		{"exact match survives a tempting neighbour",
			[]string{"caching", "transaction"}, []string{"mapping", "transaction"}, 0.5, 1},
		{"exact match survives a zero-scoring rival",
			[]string{"retry", "http_call"}, []string{"http_call", "validation"}, 0.5, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, matches := o.SetRelatedness(tt.a, tt.b)
			if !closeTo(got, tt.want) {
				t.Errorf("SetRelatedness(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
			if len(matches) != tt.wantMatches {
				t.Errorf("got %d matches, want %d: %+v", len(matches), tt.wantMatches, matches)
			}
			rev, _ := o.SetRelatedness(tt.b, tt.a)
			if !closeTo(rev, got) {
				t.Errorf("SetRelatedness is asymmetric on (%v, %v): %v vs %v", tt.a, tt.b, got, rev)
			}
		})
	}
}

func TestSetRelatednessReportsTheAncestorBehindAMatch(t *testing.T) {
	_, matches := Default().SetRelatedness([]string{"db_access"}, []string{"caching"})
	if len(matches) != 1 {
		t.Fatalf("got %d matches, want 1", len(matches))
	}
	m := matches[0]
	// A comes from the first argument and B from the second, so the pairing
	// reads in call order rather than being jointly sorted.
	if m.A != "db_access" || m.B != "caching" {
		t.Errorf("match = %q/%q, want db_access/caching", m.A, m.B)
	}
	if m.LCA != ConDataStoreAccess {
		t.Errorf("match LCA = %q, want %q", m.LCA, ConDataStoreAccess)
	}
	if m.Exact() {
		t.Error("a match between two different terms reported itself as exact")
	}
}

// exactRatio is the intersection-over-larger-set score the comparator used
// before the taxonomy existed. Soft matching must never fall below it, or a
// pair could lose merge-worthiness purely by gaining a hierarchy.
func exactRatio(a, b []string) float64 {
	as, bs := sortedUnique(a), sortedUnique(b)
	set := map[string]bool{}
	for _, s := range as {
		set[s] = true
	}
	var shared int
	for _, s := range bs {
		if set[s] {
			shared++
			delete(set, s)
		}
	}
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	if n == 0 {
		return 0
	}
	return float64(shared) / float64(n)
}

func TestSetRelatednessNeverFallsBelowExactMatching(t *testing.T) {
	o := Default()
	leaves := []string{
		"retry", "http_call", "db_access", "validation", "mapping",
		"transaction", "caching", "concurrency", "error_wrapping",
	}
	var sets [][]string
	for i := range leaves {
		sets = append(sets, []string{leaves[i]})
		for j := i + 1; j < len(leaves); j++ {
			sets = append(sets, []string{leaves[i], leaves[j]})
			for k := j + 1; k < len(leaves); k++ {
				sets = append(sets, []string{leaves[i], leaves[j], leaves[k]})
			}
		}
	}
	var checked int
	for _, a := range sets {
		for _, b := range sets {
			got, _ := o.SetRelatedness(a, b)
			if want := exactRatio(a, b); got < want-1e-9 {
				t.Fatalf("SetRelatedness(%v, %v) = %v, below exact ratio %v", a, b, got, want)
			}
			if got > 1.0+1e-9 {
				t.Fatalf("SetRelatedness(%v, %v) = %v, above 1.0", a, b, got)
			}
			checked++
		}
	}
	t.Logf("checked %d set pairs", checked)
}

func TestRoleRelatedness(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want float64
	}{
		{"identical leaf", "leaf", "leaf", 1.0},
		{"identical utility", "utility", "utility", 1.0},
		{"identical passthrough", "passthrough", "passthrough", 1.0},
		{"both high fan-in", "utility", "passthrough", 0.5},
		{"both high fan-out", "orchestrator", "passthrough", 0.5},
		{"opposite axes", "orchestrator", "utility", 0.0},
		// Agreement on a low axis must not score. Fan-out counts every call
		// including stdlib, so leaf mostly means the counts could not tell us
		// anything, and crediting it would raise the floor under every
		// unrelated pair in the report.
		{"leaf and orchestrator share only a low axis", "leaf", "orchestrator", 0.0},
		{"leaf and utility share only a low axis", "leaf", "utility", 0.0},
		{"leaf and passthrough", "leaf", "passthrough", 0.0},
		{"empty role", "", "leaf", 0.0},
		{"both empty", "", "", 0.0},
		{"unknown role", "coordinator", "leaf", 0.0},
		{"unknown role against itself", "coordinator", "coordinator", 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoleRelatedness(tt.a, tt.b)
			if !closeTo(got, tt.want) {
				t.Errorf("RoleRelatedness(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
			if rev := RoleRelatedness(tt.b, tt.a); !closeTo(rev, got) {
				t.Errorf("RoleRelatedness is asymmetric on (%q, %q)", tt.a, tt.b)
			}
		})
	}
}

func TestRoleForAndAxesForRoundTrip(t *testing.T) {
	for _, axes := range []RoleAxes{{false, false}, {true, false}, {false, true}, {true, true}} {
		id := RoleFor(axes)
		back, ok := AxesFor(id)
		if !ok {
			t.Errorf("AxesFor(%q) not found", id)
			continue
		}
		if back != axes {
			t.Errorf("RoleFor(%+v) = %q, which decomposes to %+v", axes, id, back)
		}
	}
	if _, ok := AxesFor("coordinator"); ok {
		t.Error("AxesFor reported axes for an unknown role")
	}
}

func TestBestMatch(t *testing.T) {
	if got := BestMatch(nil); got != 0 {
		t.Errorf("BestMatch(nil) = %v, want 0", got)
	}
	matches := []Match{{Score: 0.33}, {Score: 1.0}, {Score: 0.67}}
	if got := BestMatch(matches); got != 1.0 {
		t.Errorf("BestMatch = %v, want 1.0", got)
	}
}

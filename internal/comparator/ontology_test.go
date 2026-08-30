package comparator

import (
	"math"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
)

func approx(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

// The five cases in comparator_test.go assert bounds, which is what makes them
// a durable regression guard. These pin the exact values those bounds now
// bracket, so a change in the weight table shows up as a diff here rather than
// silently using up the slack in a bound.
//
// Two weight changes separate these from the historical values. First the nine
// original weights were scaled by 0.9 for the caller/callee concept signals;
// then shares_neighborhood took 0.030 out of calls (0.210) and called_by
// (0.120). These documents carry no caller/callee patterns and no
// neighborhoods, so the new signals contribute nothing and the re-weighted
// calls/called_by terms are the whole difference from the previous pins.
func TestCompareExactScores(t *testing.T) {
	tests := []struct {
		name string
		a, b concepter.ConceptDoc
		want float64
	}{
		{
			name: "identical docs",
			a: concepter.ConceptDoc{Name: "foo", Package: "pkg", Exported: true, Role: "utility",
				Callees: []string{"bar", "baz"}, Callers: []string{"main"}, Concepts: parser.Certain("retry", "http_call")},
			b: concepter.ConceptDoc{Name: "foo2", Package: "pkg", Exported: true, Role: "utility",
				Callees: []string{"bar", "baz"}, Callers: []string{"main"}, Concepts: parser.Certain("retry", "http_call")},
			want: 0.825, // was 0.950, then 0.855
		},
		{
			name: "completely disjoint",
			a: concepter.ConceptDoc{Name: "alpha", Package: "pkgA", Exported: true, Role: "leaf",
				Callees: []string{"x"}, Concepts: parser.Certain("retry")},
			b: concepter.ConceptDoc{Name: "beta", Package: "pkgB", Exported: false, Role: "orchestrator",
				Callees: []string{"y"}, Concepts: parser.Certain("db_access")},
			want: 0.045, // was 0.050; no calls/called_by contribution, so both carves leave it
		},
		{
			name: "partial overlap",
			a: concepter.ConceptDoc{Name: "handler1", Package: "api", Exported: true, Role: "orchestrator",
				Callees: []string{"validate", "save", "notify"}, Concepts: parser.Certain("validation", "db_access")},
			b: concepter.ConceptDoc{Name: "handler2", Package: "api", Exported: true, Role: "utility",
				Callees: []string{"validate", "save", "log"}, Concepts: parser.Certain("validation")},
			want: 0.410, // was 0.4667, then 0.420; still clears the 0.4 merge threshold, now with 0.010 slack
		},
		{
			name: "empty slices",
			a:    concepter.ConceptDoc{Name: "empty1", Package: "pkg", Role: "leaf"},
			b:    concepter.ConceptDoc{Name: "empty2", Package: "pkg", Role: "leaf"},
			want: 0.315, // was 0.350; no calls/called_by contribution, so both carves leave it
		},
		{
			name: "same receiver type methods",
			a: concepter.ConceptDoc{Name: "Server.Start", Package: "http", Exported: true, ReceiverType: "*Server",
				Role: "orchestrator", Callees: []string{"listen", "serve"}, Concepts: parser.Certain("concurrency")},
			b: concepter.ConceptDoc{Name: "Server.Stop", Package: "http", Exported: true, ReceiverType: "*Server",
				Role: "orchestrator", Callees: []string{"shutdown", "serve"}, Concepts: parser.Certain("concurrency")},
			want: 0.600, // was 0.675, then 0.6075
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Compare(tt.a, tt.b).OverlapScore; !approx(got, tt.want) {
				t.Errorf("OverlapScore = %.4f, want %.4f", got, tt.want)
			}
		})
	}
}

// The point of the whole change: two functions doing related-but-not-identical
// work used to score zero on intent, indistinguishable from two functions with
// nothing in common.
func TestCompareGradedConcepts(t *testing.T) {
	doc := func(name, pattern string) concepter.ConceptDoc {
		return concepter.ConceptDoc{Name: name, Package: "store", Role: "utility", Concepts: parser.Certain(pattern)}
	}
	tests := []struct {
		name            string
		a, b            concepter.ConceptDoc
		wantRelatedness float64
		wantScore       float64
		wantMerge       bool
		wantReason      string
	}{
		{
			// Siblings clear the signal threshold, so this both scores higher
			// and gains a third signal, which is what tips it to merge-worthy.
			name: "siblings under data_store_access",
			a:    doc("loadFromDB", "db_access"), b: doc("loadFromCache", "caching"),
			wantRelatedness: 2.0 / 3.0, wantScore: 0.435, wantMerge: true,
			wantReason: "related patterns: db_access ≈ caching (both data_store_access, 0.67)",
		},
		{
			// Cousins nudge the score but must not count as a signal, or weak
			// evidence would start flipping pairs into merge-worthiness.
			name: "cousins under io_operation",
			a:    doc("fetchOverHTTP", "http_call"), b: doc("fetchFromDB", "db_access"),
			wantRelatedness: 1.0 / 3.0, wantScore: 0.375, wantMerge: false,
			wantReason: "related patterns: http_call ≈ db_access (both io_operation, 0.33)",
		},
		{
			name: "different branches stay at zero",
			a:    doc("retryIt", "retry"), b: doc("saveIt", "db_access"),
			wantRelatedness: 0, wantScore: 0.315, wantMerge: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := Compare(tt.a, tt.b)
			if !approx(ev.PatternRelatedness, tt.wantRelatedness) {
				t.Errorf("PatternRelatedness = %v, want %v", ev.PatternRelatedness, tt.wantRelatedness)
			}
			if !approx(ev.OverlapScore, tt.wantScore) {
				t.Errorf("OverlapScore = %.4f, want %.4f", ev.OverlapScore, tt.wantScore)
			}
			if ev.ContextMergeWorthy != tt.wantMerge {
				t.Errorf("ContextMergeWorthy = %t, want %t", ev.ContextMergeWorthy, tt.wantMerge)
			}
			if len(ev.SharedPatterns) != 0 {
				t.Errorf("SharedPatterns = %v, want none: these tags are not equal", ev.SharedPatterns)
			}
			if tt.wantReason != "" && !hasReason(ev.Reasons, tt.wantReason) {
				t.Errorf("reasons %v do not contain %q", ev.Reasons, tt.wantReason)
			}
		})
	}
}

// A raised score with no bullet explaining it would undercut the point of the
// evidence section, so every graded signal has to be able to speak.
func TestGradedSignalsAlwaysProduceAReason(t *testing.T) {
	ev := Compare(
		concepter.ConceptDoc{Name: "a", Package: "p", Role: "utility", Concepts: parser.Certain("db_access"),
			CallerConcepts: parser.Certain("http_call"), CalleeConcepts: parser.Certain("transaction")},
		concepter.ConceptDoc{Name: "b", Package: "p", Role: "passthrough", Concepts: parser.Certain("caching"),
			CallerConcepts: parser.Certain("http_call"), CalleeConcepts: parser.Certain("db_access")},
	)
	for _, want := range []string{
		"related patterns:",
		"related roles:",
		"callers do related work (",
		"callees do related work (",
	} {
		if !hasReason(ev.Reasons, want) {
			t.Errorf("reasons %v do not mention %q", ev.Reasons, want)
		}
	}
}

func TestCompareGradedRoles(t *testing.T) {
	doc := func(name, role string) concepter.ConceptDoc {
		return concepter.ConceptDoc{Name: name, Package: "p", Role: role}
	}
	tests := []struct {
		name            string
		roleA, roleB    string
		wantRelatedness float64
		wantSameRole    bool
	}{
		{"identical", "utility", "utility", 1.0, true},
		{"both high fan-in", "utility", "passthrough", 0.5, false},
		{"both high fan-out", "orchestrator", "passthrough", 0.5, false},
		{"opposite axes", "utility", "orchestrator", 0.0, false},
		// Sharing only a low axis must not score, or every unrelated pair of
		// leaf functions would collect free credit.
		{"only a low axis in common", "leaf", "orchestrator", 0.0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := Compare(doc("a", tt.roleA), doc("b", tt.roleB))
			if !approx(ev.RoleRelatedness, tt.wantRelatedness) {
				t.Errorf("RoleRelatedness = %v, want %v", ev.RoleRelatedness, tt.wantRelatedness)
			}
			if ev.SameRole != tt.wantSameRole {
				t.Errorf("SameRole = %t, want %t", ev.SameRole, tt.wantSameRole)
			}
			// A partially matching role is context, not a merge signal.
			if !tt.wantSameRole && tt.wantRelatedness > 0 && countSignals(ev) != 1 {
				t.Errorf("countSignals = %d, want 1 (same package only)", countSignals(ev))
			}
		})
	}
}

// The parser keeps the star in a method name, so a value receiver and a pointer
// receiver on one type used to arrive as two different strings and score zero
// against each other.
func TestCompareReceiverBinding(t *testing.T) {
	doc := func(recv string) concepter.ConceptDoc {
		return concepter.ConceptDoc{Name: "m", Package: "p", ReceiverType: recv}
	}
	tests := []struct {
		name            string
		recvA, recvB    string
		wantRelatedness float64
		wantSame        bool
		wantKindA       string
		wantKindB       string
	}{
		{"two plain functions", "", "", 1.0, true, "function", "function"},
		{"same pointer receiver", "*Server", "*Server", 1.0, true, "method", "method"},
		{"pointer and value receiver on one type", "Server", "*Server", 1.0, true, "method", "method"},
		{"methods on different types", "*Server", "*Client", 0.5, false, "method", "method"},
		{"a function and a method", "", "*Server", 0.0, false, "function", "method"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := Compare(doc(tt.recvA), doc(tt.recvB))
			if !approx(ev.ReceiverRelatedness, tt.wantRelatedness) {
				t.Errorf("ReceiverRelatedness = %v, want %v", ev.ReceiverRelatedness, tt.wantRelatedness)
			}
			if ev.SameReceiver != tt.wantSame {
				t.Errorf("SameReceiver = %t, want %t", ev.SameReceiver, tt.wantSame)
			}
			if ev.EntityKindA != tt.wantKindA || ev.EntityKindB != tt.wantKindB {
				t.Errorf("entity kinds = %s/%s, want %s/%s", ev.EntityKindA, ev.EntityKindB, tt.wantKindA, tt.wantKindB)
			}
		})
	}
}

// mapper computes CallerPatterns and CalleePatterns for every doc. Before the
// ontology nothing read them.
func TestCompareScoresCallerAndCalleeConcepts(t *testing.T) {
	bare := concepter.ConceptDoc{Name: "a", Package: "p", Role: "leaf"}
	withContext := func(caller, callee string) concepter.ConceptDoc {
		d := bare
		d.CallerConcepts = parser.Certain(caller)
		d.CalleeConcepts = parser.Certain(callee)
		return d
	}

	none := Compare(bare, bare)
	if none.CallerConceptRelatedness != 0 || none.CalleeConceptRelatedness != 0 {
		t.Errorf("documents with no context scored %v/%v, want 0/0",
			none.CallerConceptRelatedness, none.CalleeConceptRelatedness)
	}

	related := Compare(withContext("db_access", "http_call"), withContext("caching", "http_call"))
	if !approx(related.CallerConceptRelatedness, 2.0/3.0) {
		t.Errorf("CallerConceptRelatedness = %v, want 0.667", related.CallerConceptRelatedness)
	}
	if !approx(related.CalleeConceptRelatedness, 1.0) {
		t.Errorf("CalleeConceptRelatedness = %v, want 1.0", related.CalleeConceptRelatedness)
	}
	if related.OverlapScore <= none.OverlapScore {
		t.Errorf("shared context did not raise the score: %v vs %v", related.OverlapScore, none.OverlapScore)
	}
	// Context is not by itself a reason to merge two functions.
	if countSignals(related) != countSignals(none) {
		t.Errorf("context changed the signal count from %d to %d", countSignals(none), countSignals(related))
	}
}

// One exact match among several tags is strong evidence even though the
// aggregate ratio is low. Thresholding the ratio instead of the best pairing
// would drop this pair's signal count at an unchanged score.
func TestPatternSignalJudgesTheBestMatchNotTheAverage(t *testing.T) {
	a := concepter.ConceptDoc{Name: "a", Package: "pkgA", Role: "leaf",
		Concepts: parser.Certain("http_call", "concurrency", "error_wrapping")}
	b := concepter.ConceptDoc{Name: "b", Package: "pkgB", Role: "orchestrator",
		Concepts: parser.Certain("http_call", "mapping", "validation")}

	ev := Compare(a, b)
	if ev.PatternRelatedness > 0.5 {
		t.Fatalf("PatternRelatedness = %v, expected a low aggregate for this fixture", ev.PatternRelatedness)
	}
	if len(ev.SharedPatterns) != 1 {
		t.Fatalf("SharedPatterns = %v, want exactly one", ev.SharedPatterns)
	}
	if countSignals(ev) != 1 {
		t.Errorf("countSignals = %d, want 1: the exact match must still count", countSignals(ev))
	}
}

// The report has to be byte-identical for an unchanged tree, and --struct-min
// turns score jitter into pairs appearing and disappearing rather than digits
// moving. Map iteration order is the usual way that breaks, so run the whole
// comparison repeatedly. Worth running with -count=5: Go reseeds map ordering
// per process.
func TestCompareIsDeterministic(t *testing.T) {
	a := concepter.ConceptDoc{Name: "a", Package: "svc", Exported: true, ReceiverType: "*Server",
		Role: "passthrough", Callees: []string{"save", "log", "emit"}, Callers: []string{"main", "run"},
		Concepts:       parser.Certain("db_access", "error_wrapping", "concurrency"),
		CallerConcepts: parser.Certain("http_call", "validation"), CalleeConcepts: parser.Certain("transaction", "mapping"),
		CallerPackages: []string{"api", "cmd"}, CalleePackages: []string{"store"}}
	b := concepter.ConceptDoc{Name: "b", Package: "svc", Exported: true, ReceiverType: "*Client",
		Role: "utility", Callees: []string{"save", "emit", "flush"}, Callers: []string{"main", "serve"},
		Concepts:       parser.Certain("caching", "error_wrapping", "retry"),
		CallerConcepts: parser.Certain("db_access", "validation"), CalleeConcepts: parser.Certain("caching", "mapping"),
		CallerPackages: []string{"api", "web"}, CalleePackages: []string{"store", "cache"}}

	first := Compare(a, b)
	if len(first.RelatedPatterns) == 0 {
		t.Fatal("fixture produced no graded matches, so it cannot detect ordering bugs")
	}
	wantReasons := strings.Join(first.Reasons, "\n")
	for i := 0; i < 1000; i++ {
		ev := Compare(a, b)
		if ev.OverlapScore != first.OverlapScore {
			t.Fatalf("run %d scored %v, first run scored %v", i, ev.OverlapScore, first.OverlapScore)
		}
		if got := strings.Join(ev.Reasons, "\n"); got != wantReasons {
			t.Fatalf("run %d produced different reasons:\n%s\nwant:\n%s", i, got, wantReasons)
		}
	}
}

// Compare has always been symmetric apart from the A/B labels, and the graded
// matcher has to keep it that way.
func TestCompareIsSymmetric(t *testing.T) {
	a := concepter.ConceptDoc{Name: "a", Package: "p", Exported: true, Role: "utility",
		Callees: []string{"x", "y"}, Concepts: parser.Certain("caching", "transaction"),
		CallerConcepts: parser.Certain("http_call")}
	b := concepter.ConceptDoc{Name: "b", Package: "p", Exported: true, Role: "passthrough",
		Callees: []string{"y", "z"}, Concepts: parser.Certain("mapping", "transaction"),
		CallerConcepts: parser.Certain("db_access")}

	ab, ba := Compare(a, b), Compare(b, a)
	if ab.OverlapScore != ba.OverlapScore {
		t.Errorf("Compare(a,b) = %v but Compare(b,a) = %v", ab.OverlapScore, ba.OverlapScore)
	}
	if ab.ContextMergeWorthy != ba.ContextMergeWorthy {
		t.Errorf("ContextMergeWorthy differs by argument order")
	}
	if ab.PatternRelatedness != ba.PatternRelatedness {
		t.Errorf("PatternRelatedness differs by argument order: %v vs %v", ab.PatternRelatedness, ba.PatternRelatedness)
	}
}

func hasReason(reasons []string, substr string) bool {
	for _, r := range reasons {
		if strings.Contains(r, substr) {
			return true
		}
	}
	return false
}

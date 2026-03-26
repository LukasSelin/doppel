package comparator

import (
	"testing"

	"github.com/lukse/doppel/internal/concepter"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		name        string
		a, b        concepter.ConceptDoc
		wantMerge   bool
		wantMinOver float64
		wantMaxOver float64
	}{
		{
			name: "identical docs",
			a: concepter.ConceptDoc{
				Name:     "foo",
				Package:  "pkg",
				Exported: true,
				Role:     "utility",
				Callees:  []string{"bar", "baz"},
				Callers:  []string{"main"},
				Patterns: []string{"retry", "http_call"},
			},
			b: concepter.ConceptDoc{
				Name:     "foo2",
				Package:  "pkg",
				Exported: true,
				Role:     "utility",
				Callees:  []string{"bar", "baz"},
				Callers:  []string{"main"},
				Patterns: []string{"retry", "http_call"},
			},
			wantMerge:   true,
			wantMinOver: 0.8,
			wantMaxOver: 1.0,
		},
		{
			name: "completely disjoint",
			a: concepter.ConceptDoc{
				Name:     "alpha",
				Package:  "pkgA",
				Exported: true,
				Role:     "leaf",
				Callees:  []string{"x"},
				Patterns: []string{"retry"},
			},
			b: concepter.ConceptDoc{
				Name:     "beta",
				Package:  "pkgB",
				Exported: false,
				Role:     "orchestrator",
				Callees:  []string{"y"},
				Patterns: []string{"db_access"},
			},
			wantMerge:   false,
			wantMinOver: 0.0,
			wantMaxOver: 0.1,
		},
		{
			name: "partial overlap shared callees different role",
			a: concepter.ConceptDoc{
				Name:     "handler1",
				Package:  "api",
				Exported: true,
				Role:     "orchestrator",
				Callees:  []string{"validate", "save", "notify"},
				Patterns: []string{"validation", "db_access"},
			},
			b: concepter.ConceptDoc{
				Name:     "handler2",
				Package:  "api",
				Exported: true,
				Role:     "utility",
				Callees:  []string{"validate", "save", "log"},
				Patterns: []string{"validation"},
			},
			wantMerge:   true,
			wantMinOver: 0.3,
			wantMaxOver: 0.7,
		},
		{
			name: "empty slices",
			a: concepter.ConceptDoc{
				Name:    "empty1",
				Package: "pkg",
				Role:    "leaf",
			},
			b: concepter.ConceptDoc{
				Name:    "empty2",
				Package: "pkg",
				Role:    "leaf",
			},
			// With re-normalization, matching package/role/visibility/receiver
			// across only active dimensions yields a high score.
			wantMerge:   true,
			wantMinOver: 0.9,
			wantMaxOver: 1.0,
		},
		{
			name: "same receiver type methods",
			a: concepter.ConceptDoc{
				Name:         "Server.Start",
				Package:      "http",
				Exported:     true,
				ReceiverType: "*Server",
				Role:         "orchestrator",
				Callees:      []string{"listen", "serve"},
				Patterns:     []string{"concurrency"},
			},
			b: concepter.ConceptDoc{
				Name:         "Server.Stop",
				Package:      "http",
				Exported:     true,
				ReceiverType: "*Server",
				Role:         "orchestrator",
				Callees:      []string{"shutdown", "serve"},
				Patterns:     []string{"concurrency"},
			},
			wantMerge:   true,
			wantMinOver: 0.4,
			wantMaxOver: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := Compare(tt.a, tt.b)
			if ev.MergeWorthy != tt.wantMerge {
				t.Errorf("MergeWorthy = %v, want %v (score=%.3f)",
					ev.MergeWorthy, tt.wantMerge, ev.OverlapScore)
			}
			if ev.OverlapScore < tt.wantMinOver || ev.OverlapScore > tt.wantMaxOver {
				t.Errorf("OverlapScore = %.3f, want [%.2f, %.2f]",
					ev.OverlapScore, tt.wantMinOver, tt.wantMaxOver)
			}
			if len(ev.Dimensions) != 9 {
				t.Fatalf("expected 9 dimensions, got %d", len(ev.Dimensions))
			}
		})
	}
}

func TestDimensionScores_IdenticalDocs(t *testing.T) {
	a := concepter.ConceptDoc{
		Name:     "foo",
		Package:  "pkg",
		Exported: true,
		Role:     "utility",
		Callees:  []string{"bar", "baz"},
		Callers:  []string{"main"},
		Patterns: []string{"retry", "http_call"},
	}
	b := a // identical
	b.Name = "foo2"

	ev := Compare(a, b)

	for _, dim := range ev.Dimensions {
		// Empty list dimensions (CallerPkgs, CalleePkgs) have sim=0 because
		// there is no data — that's correct, not a disagreement.
		if dim.Name == "SharedCallerPkgs" || dim.Name == "SharedCalleePkgs" {
			if dim.Contradiction > 0.0 {
				t.Errorf("dimension %s: expected contradiction=0.0, got %.2f", dim.Name, dim.Contradiction)
			}
			continue
		}
		if dim.Similarity < 1.0 {
			t.Errorf("dimension %s: expected similarity=1.0, got %.2f", dim.Name, dim.Similarity)
		}
		if dim.Contradiction > 0.0 {
			t.Errorf("dimension %s: expected contradiction=0.0, got %.2f", dim.Name, dim.Contradiction)
		}
	}
}

func TestDimensionScores_Contradictions(t *testing.T) {
	a := concepter.ConceptDoc{
		Name:         "alpha",
		Package:      "pkgA",
		Exported:     true,
		ReceiverType: "*Foo",
		Role:         "leaf",
		Callees:      []string{"x"},
		Patterns:     []string{"retry"},
	}
	b := concepter.ConceptDoc{
		Name:         "beta",
		Package:      "pkgB",
		Exported:     false,
		ReceiverType: "*Bar",
		Role:         "orchestrator",
		Callees:      []string{"y"},
		Patterns:     []string{"db_access"},
	}

	ev := Compare(a, b)

	dimByName := make(map[string]DimensionScore)
	for _, d := range ev.Dimensions {
		dimByName[d.Name] = d
	}

	// Callees: completely disjoint, both non-empty → contra=1.0
	if d := dimByName["SharedCallees"]; d.Contradiction != 1.0 {
		t.Errorf("SharedCallees contradiction = %.2f, want 1.0", d.Contradiction)
	}
	if d := dimByName["SharedCallees"]; d.Similarity != 0.0 {
		t.Errorf("SharedCallees similarity = %.2f, want 0.0", d.Similarity)
	}

	// Role: both have roles but differ → contra=1.0
	if d := dimByName["SameRole"]; d.Contradiction != 1.0 {
		t.Errorf("SameRole contradiction = %.2f, want 1.0", d.Contradiction)
	}

	// Package: both have packages but differ → contra=1.0
	if d := dimByName["SamePackage"]; d.Contradiction != 1.0 {
		t.Errorf("SamePackage contradiction = %.2f, want 1.0", d.Contradiction)
	}

	// Visibility: differ → contra=1.0
	if d := dimByName["SameVisibility"]; d.Contradiction != 1.0 {
		t.Errorf("SameVisibility contradiction = %.2f, want 1.0", d.Contradiction)
	}

	// Receiver: both have receivers but differ → contra=1.0
	if d := dimByName["SameReceiver"]; d.Contradiction != 1.0 {
		t.Errorf("SameReceiver contradiction = %.2f, want 1.0", d.Contradiction)
	}
}

func TestDimensionScores_EmptyNoContradiction(t *testing.T) {
	// When both sides are empty, there should be no contradiction.
	a := concepter.ConceptDoc{Name: "a"}
	b := concepter.ConceptDoc{Name: "b"}

	ev := Compare(a, b)

	for _, dim := range ev.Dimensions {
		if dim.Name == "SameVisibility" {
			// Visibility is always defined (bool), so both false = sim=1.0
			continue
		}
		if dim.Name == "SameReceiver" {
			// Both empty receiver = both plain functions, sim=1.0
			continue
		}
		if dim.Contradiction > 0.0 {
			t.Errorf("dimension %s: expected contradiction=0.0 for empty data, got %.2f",
				dim.Name, dim.Contradiction)
		}
	}
}

func TestDimensionScores_PartialOverlap(t *testing.T) {
	a := concepter.ConceptDoc{
		Callees:  []string{"validate", "save", "notify"},
		Patterns: []string{"validation", "db_access"},
	}
	b := concepter.ConceptDoc{
		Callees:  []string{"validate", "save", "log"},
		Patterns: []string{"validation"},
	}

	ev := Compare(a, b)

	dimByName := make(map[string]DimensionScore)
	for _, d := range ev.Dimensions {
		dimByName[d.Name] = d
	}

	// Callees: 2/3 overlap
	d := dimByName["SharedCallees"]
	wantSim := 2.0 / 3.0
	if d.Similarity < wantSim-0.01 || d.Similarity > wantSim+0.01 {
		t.Errorf("SharedCallees similarity = %.4f, want ~%.4f", d.Similarity, wantSim)
	}
	wantContra := 1.0 / 3.0
	if d.Contradiction < wantContra-0.01 || d.Contradiction > wantContra+0.01 {
		t.Errorf("SharedCallees contradiction = %.4f, want ~%.4f", d.Contradiction, wantContra)
	}

	// Patterns: 1/2 overlap
	d = dimByName["SharedPatterns"]
	if d.Similarity < 0.49 || d.Similarity > 0.51 {
		t.Errorf("SharedPatterns similarity = %.4f, want ~0.50", d.Similarity)
	}
	if d.Contradiction < 0.49 || d.Contradiction > 0.51 {
		t.Errorf("SharedPatterns contradiction = %.4f, want ~0.50", d.Contradiction)
	}
}

func TestDimensionEvidence(t *testing.T) {
	a := concepter.ConceptDoc{
		Package: "api",
		Role:    "orchestrator",
		Callees: []string{"validate", "save"},
	}
	b := concepter.ConceptDoc{
		Package: "handler",
		Role:    "utility",
		Callees: []string{"validate", "log"},
	}

	ev := Compare(a, b)

	dimByName := make(map[string]DimensionScore)
	for _, d := range ev.Dimensions {
		dimByName[d.Name] = d
	}

	if d := dimByName["SameRole"]; d.Evidence != "roles differ: orchestrator vs utility" {
		t.Errorf("SameRole evidence = %q", d.Evidence)
	}
	if d := dimByName["SamePackage"]; d.Evidence != "different packages: api vs handler" {
		t.Errorf("SamePackage evidence = %q", d.Evidence)
	}
	if d := dimByName["SharedCallees"]; d.Evidence == "" {
		t.Error("SharedCallees evidence should not be empty")
	}
}

func TestIntersect(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want []string
	}{
		{"both empty", nil, nil, nil},
		{"one empty", []string{"a"}, nil, nil},
		{"no overlap", []string{"a"}, []string{"b"}, nil},
		{"full overlap", []string{"a", "b"}, []string{"a", "b"}, []string{"a", "b"}},
		{"partial", []string{"a", "b", "c"}, []string{"b", "d"}, []string{"b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := intersect(tt.a, tt.b)
			if len(got) != len(tt.want) {
				t.Fatalf("intersect = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("intersect[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

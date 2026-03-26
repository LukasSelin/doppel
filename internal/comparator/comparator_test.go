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
			wantMerge:   true, // shared callees + shared patterns + same package = 3 signals
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
			wantMerge:   false, // no shared callees/patterns → < 2 signals
			wantMinOver: 0.0,
			wantMaxOver: 0.5,
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
				t.Errorf("MergeWorthy = %v, want %v (score=%.3f, reasons=%v)",
					ev.MergeWorthy, tt.wantMerge, ev.OverlapScore, ev.Reasons)
			}
			if ev.OverlapScore < tt.wantMinOver || ev.OverlapScore > tt.wantMaxOver {
				t.Errorf("OverlapScore = %.3f, want [%.2f, %.2f]",
					ev.OverlapScore, tt.wantMinOver, tt.wantMaxOver)
			}
		})
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

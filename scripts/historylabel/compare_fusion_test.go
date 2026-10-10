package main

import "testing"

func TestFusedInputs(t *testing.T) {
	for _, tc := range []struct{ method, a, b string }{
		{"RRF(doppel, dupl (t=100, default))", "doppel", "dupl (t=100, default)"},
		{"RRF(doppel, token clones (≥50 nodes))", "doppel", "token clones (≥50 nodes)"},
	} {
		a, b, ok := fusedInputs(tc.method)
		if !ok || a != tc.a || b != tc.b {
			t.Errorf("%q: got %q, %q, %v", tc.method, a, b, ok)
		}
	}
	for _, m := range []string{"doppel", "dupl (t=50)", "RRF(doppel)"} {
		if _, _, ok := fusedInputs(m); ok {
			t.Errorf("%q read as fused", m)
		}
	}
}

package cmd

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
)

func testPopulation() []parser.CodeUnit {
	return []parser.CodeUnit{
		{Name: "Prod1", File: "a/service.go"},
		{Name: "TestProd1", File: "a/service_test.go"},
		{Name: "Prod2", File: "b/worker.go"},
		{Name: "TestProd2", File: "b/worker_test.go"},
	}
}

func names(units []parser.CodeUnit) []string {
	out := make([]string, len(units))
	for i, u := range units {
		out[i] = u.Name
	}
	return out
}

func TestFilterTestUnits(t *testing.T) {
	cases := []struct {
		mode string
		want []string
	}{
		{"include", []string{"Prod1", "TestProd1", "Prod2", "TestProd2"}},
		{"exclude", []string{"Prod1", "Prod2"}},
		{"only", []string{"TestProd1", "TestProd2"}},
	}
	for _, tc := range cases {
		got := names(filterTestUnits(testPopulation(), tc.mode))
		if len(got) != len(tc.want) {
			t.Errorf("%s: kept %v, want %v", tc.mode, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("%s: kept %v, want %v (order must be preserved)", tc.mode, got, tc.want)
				break
			}
		}
	}
}

func TestIsTestUnit(t *testing.T) {
	if !isTestUnit(parser.CodeUnit{File: "x/y_test.go"}) {
		t.Error("_test.go file not recognized")
	}
	if isTestUnit(parser.CodeUnit{File: "x/testify.go"}) {
		t.Error("non-test file with 'test' in the name misclassified")
	}
	if isTestUnit(parser.CodeUnit{File: "x/y_test.go.bak"}) {
		t.Error("suffix must match exactly")
	}
}

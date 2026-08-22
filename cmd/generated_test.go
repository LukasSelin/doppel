package cmd

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/parser"
)

func TestFilterGeneratedUnits(t *testing.T) {
	units := []parser.CodeUnit{
		{Name: "Hand"},
		{Name: "Machine", Generated: true},
	}
	if got := filterGeneratedUnits(units, "include"); len(got) != 2 {
		t.Errorf("include kept %d units, want 2", len(got))
	}
	units = []parser.CodeUnit{{Name: "Hand"}, {Name: "Machine", Generated: true}}
	if got := filterGeneratedUnits(units, "exclude"); len(got) != 1 || got[0].Name != "Hand" {
		t.Errorf("exclude kept %v, want just Hand", got)
	}
	units = []parser.CodeUnit{{Name: "Hand"}, {Name: "Machine", Generated: true}}
	if got := filterGeneratedUnits(units, "only"); len(got) != 1 || got[0].Name != "Machine" {
		t.Errorf("only kept %v, want just Machine", got)
	}
}

// The skip rule is the go tool's own: dot- and underscore-prefixed directories
// are not part of a build, so demo trees like chi's _examples never join the
// population. The walker exempts the root itself, or `doppel analyze .` would
// skip everything (covered here only by the rule's shape; the exemption lives
// at the call site).
func TestShouldSkipDirMatchesGoToolRule(t *testing.T) {
	for _, name := range []string{".git", ".claude", ".idea", "_examples", "_tools", "vendor", "testdata", "build"} {
		if !shouldSkipDir(name) {
			t.Errorf("shouldSkipDir(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"internal", "cmd", "examples", "builder", "x_y"} {
		if shouldSkipDir(name) {
			t.Errorf("shouldSkipDir(%q) = true, want false", name)
		}
	}
}

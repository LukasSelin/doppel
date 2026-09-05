package parser

import (
	"reflect"
	"testing"
)

// The default blocklist is a scope rule, so what it does and does not catch is
// behaviour rather than a list somebody keeps tidy.
func TestShouldSkipDirDefaults(t *testing.T) {
	skip := []string{
		".git", "_examples", "vendor", "testdata", "build",
		"node_modules", "bower_components", "jspm_packages",
		"site-packages", "venv", "Pods", "Carthage", "DerivedData",
		"third_party", "dist", "out", "target", "obj", "coverage",
		// Case-insensitively: the same tree must be the same corpus on
		// every platform doppel ships for.
		"Node_Modules", "VENDOR",
	}
	for _, name := range skip {
		if !ShouldSkipDir(name) {
			t.Errorf("ShouldSkipDir(%q) = false, want true", name)
		}
	}
	keep := []string{
		"internal", "cmd", "src", "lib", "pkg", "api", "app",
		// Deliberately not on the list: each shadows first-party source in
		// a language doppel reads. See DefaultExcludes.
		"deps", "packages", "bin",
		// Neighbours of a blocked name, which a prefix or substring rule
		// would have taken with it.
		"binary", "distribution", "outbound", "targeting", "vendors",
		"node_modules_helper", "objects",
	}
	for _, name := range keep {
		if ShouldSkipDir(name) {
			t.Errorf("ShouldSkipDir(%q) = true, want false", name)
		}
	}
}

// The zero Excludes is the walk doppel always had: a command that configures
// nothing must not walk a different tree than it used to.
func TestZeroExcludesIsTheDefaultRule(t *testing.T) {
	var e Excludes
	for _, name := range []string{"node_modules", "vendor", ".git", "internal", "src"} {
		if got, want := e.SkipDir(name, name), ShouldSkipDir(name); got != want {
			t.Errorf("zero Excludes.SkipDir(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestExcludesPatterns(t *testing.T) {
	cases := []struct {
		name     string
		patterns []string
		rel      string
		dir      string
		want     bool
	}{
		{"added name", []string{"generated"}, "internal/generated", "generated", true},
		{"added glob", []string{"*_gen"}, "api/proto_gen", "proto_gen", true},
		{"glob misses", []string{"*_gen"}, "api/gen_proto", "gen_proto", false},
		// A bare name is a name rule, so it fires wherever the directory sits.
		{"name anywhere", []string{"fixtures"}, "a/b/c/fixtures", "fixtures", true},
		// A slash makes it a path rule, anchored at the analysis root.
		{"path anchored", []string{"internal/generated"}, "internal/generated", "generated", true},
		{"path elsewhere", []string{"internal/generated"}, "cmd/generated", "generated", false},
		{"path glob", []string{"api/*/gen"}, "api/v1/gen", "gen", true},
		// path.Match's * does not cross a separator, which is the whole
		// reason the two forms are different rules.
		{"path glob one segment", []string{"api/*/gen"}, "api/v1/v2/gen", "gen", false},
		// Negation re-admits a default, wherever it appears in the list.
		{"negation", []string{"!dist"}, "dist", "dist", false},
		{"negation after exclude", []string{"tools", "!tools"}, "tools", "tools", false},
		{"negation before exclude", []string{"!tools", "tools"}, "tools", "tools", false},
		{"negation on a path", []string{"!scripts/dist"}, "scripts/dist", "dist", false},
		{"negation elsewhere", []string{"!scripts/dist"}, "app/dist", "dist", true},
		// The dot/underscore rule is a default like any other, so it can be
		// taken back too.
		{"negated dot dir", []string{"!.github"}, ".github", ".github", false},
		// Case-insensitive on both sides.
		{"case insensitive pattern", []string{"Generated"}, "internal/generated", "generated", true},
		{"case insensitive subject", []string{"generated"}, "internal/Generated", "Generated", true},
		// Untouched defaults still apply alongside configuration.
		{"defaults still apply", []string{"generated"}, "node_modules", "node_modules", true},
		{"unmatched keeps source", []string{"generated"}, "internal", "internal", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, err := NewExcludes(tc.patterns)
			if err != nil {
				t.Fatalf("NewExcludes(%v): %v", tc.patterns, err)
			}
			if got := e.SkipDir(tc.rel, tc.dir); got != tc.want {
				t.Errorf("SkipDir(%q, %q) = %v, want %v", tc.rel, tc.dir, got, tc.want)
			}
		})
	}
}

// A malformed glob must not become a pattern that silently matches nothing: an
// exclusion decides what the corpus is, and a corpus changed by a typo is what
// nobody notices until the report is already wrong.
func TestExcludesRejectMalformed(t *testing.T) {
	for _, pat := range []string{"[", "!", "!/", "/", "a/["} {
		if _, err := NewExcludes([]string{pat}); err == nil {
			t.Errorf("NewExcludes(%q) = nil error, want a rejection", pat)
		}
	}
	// Blank entries are how a comma-separated flag or a config list spells
	// "nothing here", and must not be an error.
	e, err := NewExcludes([]string{"", "  "})
	if err != nil {
		t.Fatalf("NewExcludes(blanks): %v", err)
	}
	if got := e.Patterns(); got != nil {
		t.Errorf("Patterns() = %v, want nil", got)
	}
}

// Patterns is what a snapshot records, so it must be a normal form: two runs
// that configured the same exclusions compare equal however they spelled the
// order.
func TestPatternsIsSortedAndDeduped(t *testing.T) {
	a, err := NewExcludes([]string{"zeta", "alpha", " zeta ", "!dist"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewExcludes([]string{"!dist", "alpha", "zeta"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"!dist", "alpha", "zeta"}
	if got := a.Patterns(); !reflect.DeepEqual(got, want) {
		t.Errorf("Patterns() = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(a.Patterns(), b.Patterns()) {
		t.Errorf("Patterns() differs by declaration order: %v vs %v", a.Patterns(), b.Patterns())
	}
}

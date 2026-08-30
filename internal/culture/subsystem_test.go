package culture

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
)

func TestSubsystemKey(t *testing.T) {
	cases := []struct {
		root, file, want string
	}{
		{".", "internal/culture/habitat.go", "internal/"},
		{".", filepath.Join("internal", "culture", "habitat.go"), "internal/"},
		{".", "a/b/c/x.go", "a/b/"},
		{".", "cmd/x.go", ""},  // directory sits directly under the root: no parent
		{".", "main.go", ""},   // top-level file
		{"", "a/b/c/x.go", ""}, // no root, no subsystems
		{"root", "root/sub/pkg/x.go", "sub/"},
		{"root", "elsewhere/x.go", ""}, // not under the root
		{filepath.Join("C:", "r"), filepath.Join("C:", "r", "tpl", "math", "init.go"), "tpl/"},
	}
	for _, tc := range cases {
		if got := subsystemKey(tc.root, tc.file); got != tc.want {
			t.Errorf("subsystemKey(%q, %q) = %q, want %q", tc.root, tc.file, got, tc.want)
		}
	}
}

// subsystemFixture: two sibling packages under root/sub. pkgA holds five
// clones and one alien; pkgB holds five members shaped exactly like the
// alien when bSharesAlien is set (so the alien is typical of the subsystem),
// or like pkgA's clones otherwise (so the alien is alien at both levels).
func subsystemFixture(bSharesAlien bool) ([]parser.CodeUnit, []concepter.ConceptDoc) {
	clone := func(nm, pkg string) (parser.CodeUnit, concepter.ConceptDoc) {
		u := parser.CodeUnit{Name: nm, Package: pkg, File: "root/sub/" + pkg + "/" + nm + ".go", Concepts: parser.Certain("db_access", "error_wrapping")}
		u.Fingerprint.Flow = flowOf(0, 6)
		u.Callees = []string{"sql.Open"}
		u.Signals = parser.TagSignals{PackageRefs: []parser.PackageRef{{Local: "sql", Path: "database/sql"}}}
		return u, concepter.ConceptDoc{Role: "leaf"}
	}
	alienLike := func(nm, pkg string) (parser.CodeUnit, concepter.ConceptDoc) {
		u := parser.CodeUnit{Name: nm, Package: pkg, File: "root/sub/" + pkg + "/" + nm + ".go", Concepts: parser.Certain("concurrency")}
		u.Fingerprint.Flow = flowOf(5, 8)
		return u, concepter.ConceptDoc{Role: "orchestrator"}
	}
	var units []parser.CodeUnit
	var docs []concepter.ConceptDoc
	add := func(u parser.CodeUnit, d concepter.ConceptDoc) { units = append(units, u); docs = append(docs, d) }
	for i := 0; i < 5; i++ {
		add(clone(name("clone", i), "pkga"))
	}
	add(alienLike("alien", "pkga")) // index 5
	for i := 0; i < 5; i++ {
		if bSharesAlien {
			add(alienLike(name("b", i), "pkgb"))
		} else {
			add(clone(name("b", i), "pkgb"))
		}
	}
	return units, docs
}

func subsystemOpts() Options {
	o := DefaultOptions()
	o.Root = "root"
	return o
}

// A package alien that is typical of its subsystem is excused: alien in
// pkga, but six of the eleven functions under sub/ look like it.
func TestSubsystemExcusesPackageAlien(t *testing.T) {
	units, docs := subsystemFixture(true)
	m := buildOn(t, units, docs, subsystemOpts())
	const alien = 5

	if !m.PackageMisfit(alien) {
		t.Fatal("alien is not a package-level misfit; fixture broken")
	}
	if m.Misfit(alien) {
		t.Error("alien reported as a misfit although its subsystem excuses it")
	}
	key, fit, ok := m.SubsystemFit(alien)
	if !ok || key != "sub/" {
		t.Fatalf("SubsystemFit = (%q, %v, %v), want sub/", key, fit, ok)
	}
	if fit != 1.0 {
		t.Errorf("alien subsystem fit = %v, want 1.0 — its shape is the subsystem majority", fit)
	}
	s := m.Stats()
	if s.SubsystemsModeled != 1 || s.MisfitsExcused != 1 || s.HabitatMisfits != 0 {
		t.Errorf("stats = subsystems %d, excused %d, misfits %d; want 1, 1, 0", s.SubsystemsModeled, s.MisfitsExcused, s.HabitatMisfits)
	}
	if s.HabitatsModeled != 2 {
		t.Errorf("HabitatsModeled = %d, want 2 (package level is unchanged by the rollup)", s.HabitatsModeled)
	}
}

// Alien at both levels stays a misfit, with a low subsystem fit to show.
func TestSubsystemConfirmsAlienAtBothLevels(t *testing.T) {
	units, docs := subsystemFixture(false)
	m := buildOn(t, units, docs, subsystemOpts())
	const alien = 5

	if !m.Misfit(alien) {
		t.Error("alien at both levels was not reported")
	}
	if _, fit, ok := m.SubsystemFit(alien); !ok || fit >= 0.5 {
		t.Errorf("subsystem fit = %v (ok %v), want a modeled, poor fit", fit, ok)
	}
	s := m.Stats()
	if s.HabitatMisfits != 1 || s.MisfitsExcused != 0 {
		t.Errorf("stats = misfits %d, excused %d; want 1, 0", s.HabitatMisfits, s.MisfitsExcused)
	}
}

// Without a Root there is no subsystem level: Misfit is the package rule and
// the new stats stay zero — which is what keeps every older pin valid.
func TestNoRootMeansNoSubsystems(t *testing.T) {
	units, docs := subsystemFixture(true)
	m := buildOn(t, units, docs, DefaultOptions())
	if _, _, ok := m.SubsystemFit(5); ok {
		t.Error("subsystem modeled without a Root")
	}
	if m.Misfit(5) != m.PackageMisfit(5) || !m.Misfit(5) {
		t.Error("Misfit must equal PackageMisfit without subsystems")
	}
	if s := m.Stats(); s.SubsystemsModeled != 0 || s.MisfitsExcused != 0 {
		t.Errorf("stats = %+v, want no subsystem fields set", s)
	}
}

// A subsystem below the member floor is silent, like a small package.
func TestSubsystemBelowMinIsSilent(t *testing.T) {
	units, docs := subsystemFixture(true)
	o := subsystemOpts()
	o.MinHabitatMembers = 12 // eleven under sub/
	m := buildOn(t, units, docs, o)
	if _, _, ok := m.SubsystemFit(5); ok {
		t.Error("subsystem modeled below the member floor")
	}
	if s := m.Stats(); s.SubsystemsModeled != 0 {
		t.Errorf("SubsystemsModeled = %d, want 0", s.SubsystemsModeled)
	}
}

func TestSubsystemDeterminism(t *testing.T) {
	units, docs := subsystemFixture(true)
	first := buildOn(t, units, docs, subsystemOpts())
	for i := 0; i < 25; i++ {
		again := buildOn(t, units, docs, subsystemOpts())
		if again.Stats() != first.Stats() {
			t.Fatalf("run %d: stats differ", i)
		}
		for idx := range units {
			k1, f1, ok1 := first.SubsystemFit(idx)
			k2, f2, ok2 := again.SubsystemFit(idx)
			if !reflect.DeepEqual([]any{k1, f1, ok1}, []any{k2, f2, ok2}) {
				t.Fatalf("run %d: SubsystemFit(%d) differs", i, idx)
			}
		}
	}
}

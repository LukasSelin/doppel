package culture

import (
	"math"
	"reflect"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
)

func TestHabitatChannelWeightsSum(t *testing.T) {
	sum := 0
	for _, w := range habitatChannelWeightPct {
		sum += w
	}
	if sum != 100 {
		t.Fatalf("habitat channel weights sum to %d, want exactly 100", sum)
	}
	if len(habitatChannelNames) != len(habitatChannelWeightPct) {
		t.Fatalf("%d names for %d weights", len(habitatChannelNames), len(habitatChannelWeightPct))
	}
}

// habitatCloneAlienFixture: one package "store" with five clone members
// sharing every feature and one alien sharing almost none.
func habitatCloneAlienFixture() ([]parser.CodeUnit, []concepter.ConceptDoc) {
	sqlCaller := func(nm string, flow []int, tags ...string) parser.CodeUnit {
		u := parser.CodeUnit{Name: nm, Package: "store", Concepts: parser.Certain(tags...)}
		u.Fingerprint.Flow = flow
		u.Callees = []string{"sql.Open"}
		u.Signals = parser.TagSignals{PackageRefs: []parser.PackageRef{{Local: "sql", Path: "database/sql"}}}
		return u
	}
	var units []parser.CodeUnit
	docs := make([]concepter.ConceptDoc, 0, 6)
	for i := 0; i < 5; i++ {
		units = append(units, sqlCaller(name("clone", i), flowOf(0, 6), "db_access", "error_wrapping"))
		docs = append(docs, concepter.ConceptDoc{Role: "leaf"})
	}
	alien := parser.CodeUnit{Name: "alien", Package: "store", Concepts: parser.Certain("db_access")}
	alien.Fingerprint.Flow = flowOf(5, 8) // select + go: unique flow features
	units = append(units, alien)
	docs = append(docs, concepter.ConceptDoc{Role: "orchestrator"})
	return units, docs
}

// Hand-computed strain/fit pins for the clone/alien habitat (m=6):
// clone strain = 0.915·ln(6/5); alien = 0.83·ln 6; temperature = the clone
// strain (median); clone fit exactly 1.0; alien fit = exp(-excess/T) tiny.
func TestHabitatStrainCloneVsAlien(t *testing.T) {
	units, docs := habitatCloneAlienFixture()
	m := buildOn(t, units, docs, DefaultOptions())

	cloneStrain := 0.915 * math.Log(6.0/5.0)
	alienStrain := 0.83 * math.Log(6.0)

	for i := 0; i < 5; i++ {
		s, ok := m.HabitatStrain(i)
		if !ok || math.Abs(s-cloneStrain) > 1e-12 {
			t.Errorf("clone %d strain = (%v, %v), want %v", i, s, ok, cloneStrain)
		}
		fit, _ := m.HabitatFit(i)
		if fit != 1.0 {
			t.Errorf("clone %d fit = %v, want exactly 1.0", i, fit)
		}
		if m.Misfit(i) {
			t.Errorf("clone %d flagged misfit", i)
		}
	}

	s, ok := m.HabitatStrain(5)
	if !ok || math.Abs(s-alienStrain) > 1e-12 {
		t.Errorf("alien strain = (%v, %v), want %v", s, ok, alienStrain)
	}
	temp, ok := m.HabitatTemperature("store")
	if !ok || math.Abs(temp-cloneStrain) > 1e-12 {
		t.Errorf("temperature = (%v, %v), want %v", temp, ok, cloneStrain)
	}
	fit, _ := m.HabitatFit(5)
	wantFit := math.Exp(-(alienStrain - cloneStrain) / cloneStrain)
	if math.Abs(fit-wantFit) > 1e-12 || fit > 0.05 {
		t.Errorf("alien fit = %v, want %v (tiny)", fit, wantFit)
	}
	if !m.Misfit(5) {
		t.Error("alien not flagged misfit")
	}
	norm, ok := m.HabitatNorm("store")
	if !ok || math.Abs(norm-(5.0+wantFit)/6.0) > 1e-12 {
		t.Errorf("norm = (%v, %v), want %v", norm, ok, (5.0+wantFit)/6.0)
	}

	channels := m.ChannelSurprise(5)
	if len(channels) != len(habitatChannelNames) {
		t.Fatalf("got %d channels, want %d", len(channels), len(habitatChannelNames))
	}
	// Alien per channel: calls (empty set) ln6, flow ln6, tags 0, role ln6.
	wantCh := []float64{math.Log(6), math.Log(6), 0, math.Log(6)}
	for ch := range channels {
		if math.Abs(channels[ch].Surprise-wantCh[ch]) > 1e-12 {
			t.Errorf("alien channel %s surprise = %v, want %v",
				channels[ch].Name, channels[ch].Surprise, wantCh[ch])
		}
	}

	stats := m.Stats()
	if stats.HabitatsModeled != 1 || stats.HabitatMisfits != 1 {
		t.Errorf("stats = %+v, want 1 habitat, 1 misfit", stats)
	}
}

// Packages below MinHabitatMembers stay silent everywhere.
func TestSmallHabitatSilence(t *testing.T) {
	var units []parser.CodeUnit
	for i := 0; i < 4; i++ {
		units = append(units, unit(name("f", i), "tiny", "retry"))
	}
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())

	if _, ok := m.HabitatStrain(0); ok {
		t.Error("HabitatStrain reported for a 4-member package")
	}
	if _, ok := m.HabitatFit(0); ok {
		t.Error("HabitatFit reported for a 4-member package")
	}
	if _, ok := m.HabitatNorm("tiny"); ok {
		t.Error("HabitatNorm reported for a 4-member package")
	}
	if m.Misfit(0) {
		t.Error("Misfit fired for a 4-member package")
	}
	if m.ChannelSurprise(0) != nil {
		t.Error("ChannelSurprise non-nil for a 4-member package")
	}
	if s := m.Stats(); s.HabitatsModeled != 0 || s.HabitatMisfits != 0 {
		t.Errorf("stats = %+v, want zero habitats", s)
	}
}

// A perfectly uniform habitat freezes at temperature 0: everyone fits 1.0,
// nobody misfits; adding one deviant makes it fit exactly 0.0 and misfit via
// the temperature-0 disjunct.
func TestUniformHabitatGuard(t *testing.T) {
	identical := func(n int) []parser.CodeUnit {
		var units []parser.CodeUnit
		for i := 0; i < n; i++ {
			u := unit(name("same", i), "frozen", "caching")
			u.Fingerprint.Flow = flowOf(0)
			units = append(units, u)
		}
		return units
	}

	units := identical(5)
	m := buildOn(t, units, docsWithRole(5, "leaf"), DefaultOptions())
	for i := 0; i < 5; i++ {
		if s, _ := m.HabitatStrain(i); s != 0 {
			t.Errorf("uniform member %d strain = %v, want exactly 0", i, s)
		}
		if fit, _ := m.HabitatFit(i); fit != 1.0 {
			t.Errorf("uniform member %d fit = %v, want exactly 1.0", i, fit)
		}
		if m.Misfit(i) {
			t.Errorf("uniform member %d flagged misfit", i)
		}
	}

	// The deviant must carry a SUPERSET of the clone features — extra flow on
	// top of the same if/tags/role — so the five clones keep strain exactly 0
	// (every one of their features stays universal) and the median freezes at
	// temperature 0 while the deviant alone carries positive strain.
	units = identical(5)
	alien := unit("alien", "frozen", "caching")
	alien.Fingerprint.Flow = flowOf(0, 5, 8) // if + select + go
	units = append(units, alien)
	m = buildOn(t, units, docsWithRole(6, "leaf"), DefaultOptions())
	if temp, _ := m.HabitatTemperature("frozen"); temp != 0 {
		t.Fatalf("temperature = %v, want 0 (median of five zeros)", temp)
	}
	if fit, _ := m.HabitatFit(5); fit != 0.0 {
		t.Errorf("frozen-habitat alien fit = %v, want exactly 0.0", fit)
	}
	if !m.Misfit(5) {
		t.Error("frozen-habitat alien not flagged misfit")
	}
}

// Superlatives rank by norm with ties resolving to the lexicographically
// smaller name.
func TestHabitatStatsSuperlatives(t *testing.T) {
	var units []parser.CodeUnit
	var docs []concepter.ConceptDoc
	addPkg := func(pkg string, withAlien bool) {
		for i := 0; i < 5; i++ {
			u := unit(name(pkg, i), pkg, "caching")
			u.Fingerprint.Flow = flowOf(0)
			units = append(units, u)
			docs = append(docs, concepter.ConceptDoc{Role: "leaf"})
		}
		if withAlien {
			a := unit(pkg+"Alien", pkg)
			a.Fingerprint.Flow = flowOf(5, 8)
			units = append(units, a)
			docs = append(docs, concepter.ConceptDoc{Role: "orchestrator"})
		}
	}
	addPkg("hot", true)
	addPkg("cold", false)

	m := buildOn(t, units, docs, DefaultOptions())
	s := m.Stats()
	if s.MostUniformHabitat != "cold" || s.MostUniformNorm != 1.0 {
		t.Errorf("most uniform = %s (%v), want cold (1.0)", s.MostUniformHabitat, s.MostUniformNorm)
	}
	if s.MostDiverseHabitat != "hot" || s.MostDiverseNorm >= 1.0 {
		t.Errorf("most diverse = %s (%v), want hot (< 1.0)", s.MostDiverseHabitat, s.MostDiverseNorm)
	}

	// Tie: two identical uniform packages — both norms 1.0, name-ascending wins.
	units, docs = nil, nil
	addPkg("beta", false)
	addPkg("alpha", false)
	m = buildOn(t, units, docs, DefaultOptions())
	s = m.Stats()
	if s.MostUniformHabitat != "alpha" || s.MostDiverseHabitat != "alpha" {
		t.Errorf("tie resolved to %s/%s, want alpha/alpha", s.MostUniformHabitat, s.MostDiverseHabitat)
	}
}

func TestHabitatDeterminism(t *testing.T) {
	units, docs := habitatCloneAlienFixture()
	first := buildOn(t, units, docs, DefaultOptions())
	for run := 0; run < 25; run++ {
		m := buildOn(t, units, docs, DefaultOptions())
		if m.Stats() != first.Stats() {
			t.Fatalf("run %d stats diverged", run)
		}
		for i := range units {
			fs, fok := first.HabitatStrain(i)
			s, ok := m.HabitatStrain(i)
			if fs != s || fok != ok || !reflect.DeepEqual(first.ChannelSurprise(i), m.ChannelSurprise(i)) {
				t.Fatalf("run %d diverged at unit %d", run, i)
			}
		}
	}
}

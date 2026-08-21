package culture

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"github.com/lukse/doppel/internal/parser"
)

// sqlUnit is a unit calling the external token database/sql.Open.
func sqlUnit(nm, pkg string, tags ...string) parser.CodeUnit {
	u := parser.CodeUnit{Name: nm, Package: pkg, Patterns: tags}
	u.Callees = []string{"sql.Open"}
	u.Signals = parser.TagSignals{PackageRefs: []parser.PackageRef{{Local: "sql", Path: "database/sql"}}}
	return u
}

// The extinction story: the subject carries db_access (strong call support:
// IC + TagCall PMI) and error_wrapping (IC only, nothing else to explain).
// N=12: E(db) = ln3 + ln3 ≈ 2.197, E(ew) = ln6 ≈ 1.792 — the 0.405-nat gap
// grinds error_wrapping below the survivor floor.
func TestArenaExtinctionOfUnsupportedTag(t *testing.T) {
	var units []parser.CodeUnit
	units = append(units, sqlUnit("subject", "p", "db_access", "error_wrapping"))
	for i := 0; i < 3; i++ {
		units = append(units, sqlUnit(name("db", i), "p", "db_access"))
	}
	units = append(units, unit("otherEW", "p", "error_wrapping"))
	for i := 0; i < 7; i++ {
		units = append(units, unit(name("f", i), "p"))
	}
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())

	p, ok := m.ArenaProfile(0)
	if !ok {
		t.Fatal("subject has no arena profile")
	}
	if want := 2*math.Log(3) + math.Log(6); math.Abs(p.TotalEvidence-want) > 1e-9 {
		t.Errorf("TotalEvidence = %v, want %v", p.TotalEvidence, want)
	}
	if p.State != StateDominance {
		t.Errorf("state = %s, want dominance", p.State)
	}
	if len(p.Survivors) != 1 || p.Survivors[0].Tag != "db_access" || p.Survivors[0].Mass < 0.9 {
		t.Errorf("survivors = %+v, want db_access alone above 0.9", p.Survivors)
	}
	if len(p.Extinct) != 1 || p.Extinct[0].Tag != "error_wrapping" || p.Extinct[0].Mass >= 0.05 {
		t.Errorf("extinct = %+v, want error_wrapping below the survivor floor", p.Extinct)
	}
	if p.Rounds > arenaMaxRounds {
		t.Errorf("rounds = %d exceeds cap", p.Rounds)
	}
}

// Mutually-reinforcing equals: transaction+db_access co-occur corpus-wide
// (positive TagTag), the subject carries both with symmetric evidence — the
// equilibrium is an exact 0.5/0.5 coalition that converges in one round.
func TestArenaCoalition(t *testing.T) {
	var units []parser.CodeUnit
	for i := 0; i < 4; i++ {
		units = append(units, unit(name("tx", i), "p", "transaction", "db_access"))
	}
	for i := 0; i < 8; i++ {
		units = append(units, unit(name("f", i), "p"))
	}
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())

	p, ok := m.ArenaProfile(0)
	if !ok {
		t.Fatal("no profile")
	}
	if p.State != StateCoalition {
		t.Errorf("state = %s, want coalition", p.State)
	}
	if len(p.Survivors) != 2 || p.Survivors[0].Mass != 0.5 || p.Survivors[1].Mass != 0.5 {
		t.Errorf("survivors = %+v, want exactly 0.5/0.5", p.Survivors)
	}
	if p.Extinct != nil {
		t.Errorf("extinct = %+v, want nil", p.Extinct)
	}
	if p.Rounds != 1 || !p.Converged {
		t.Errorf("rounds/converged = %d/%v, want 1/true (symmetric fixed point)", p.Rounds, p.Converged)
	}

	// The dominance knob: with the threshold lowered to the tie mass, the
	// same equilibrium reads dominance instead.
	opt := DefaultOptions()
	opt.DominanceMass = 0.5
	m = buildOn(t, units, docsWithRole(len(units), "leaf"), opt)
	if p, _ := m.ArenaProfile(0); p.State != StateDominance {
		t.Errorf("state with DominanceMass 0.5 = %s, want dominance", p.State)
	}
}

// Conflict via invasion: the subject is tagged validation only, but
// transaction invades through its call token (positive TagCall at exactly
// ln 2), and the two tags never co-occur corpus-wide (reported negative with
// Count 0 — the -Inf that must map to a finite -ln N repulsion). Symmetric
// evidence keeps both alive: conflict.
func TestArenaConflictWithInfiniteRepulsionAndInvasion(t *testing.T) {
	var units []parser.CodeUnit
	units = append(units, sqlUnit("subject", "p", "validation"))
	for i := 0; i < 6; i++ {
		units = append(units, unit(name("v", i), "p", "validation"))
	}
	for i := 0; i < 6; i++ {
		units = append(units, sqlUnit(name("tx", i), "p", "transaction"))
	}
	units = append(units, unit("filler", "p"))
	if len(units) != 14 {
		t.Fatalf("fixture broken: %d units, want 14", len(units))
	}
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())

	p, ok := m.ArenaProfile(0)
	if !ok {
		t.Fatal("no profile")
	}
	// transaction is not among the subject's tags: it invaded via the token.
	tags := []string{}
	for _, s := range p.Survivors {
		tags = append(tags, s.Tag)
	}
	if !reflect.DeepEqual(tags, []string{"transaction", "validation"}) &&
		!reflect.DeepEqual(tags, []string{"validation", "transaction"}) {
		t.Fatalf("survivors = %v, want transaction (invader) and validation", tags)
	}
	if p.State != StateConflict {
		t.Errorf("state = %s, want conflict (never-co-occurring pair maps to -lnN repulsion)", p.State)
	}
	if want := 2 * math.Ln2; math.Abs(p.TotalEvidence-want) > 1e-9 {
		t.Errorf("TotalEvidence = %v, want 2·ln2 = %v", p.TotalEvidence, want)
	}
}

// Weak precedes everything: a tag carried by every unit has zero IC and no
// associations, so even a mass-1.0 single candidate is a weak ecosystem.
func TestArenaWeakPrecedence(t *testing.T) {
	var units []parser.CodeUnit
	for i := 0; i < 12; i++ {
		units = append(units, unit(name("c", i), "p", "caching"))
	}
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())

	p, ok := m.ArenaProfile(0)
	if !ok {
		t.Fatal("no profile")
	}
	if p.TotalEvidence != 0 {
		t.Errorf("TotalEvidence = %v, want exactly 0 (IC of a universal tag)", p.TotalEvidence)
	}
	if p.State != StateWeak {
		t.Errorf("state = %s, want weak despite single mass-1.0 candidate", p.State)
	}
	if len(p.Survivors) != 1 || p.Survivors[0].Mass != 1.0 {
		t.Errorf("survivors = %+v, want the single candidate at 1.0", p.Survivors)
	}
}

// Untagged units with no matching associations are silent — no state.
func TestArenaSilence(t *testing.T) {
	units := []parser.CodeUnit{unit("bare", "p")}
	for i := 0; i < 5; i++ {
		units = append(units, unit(name("t", i), "p", "retry"))
	}
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())
	if _, ok := m.ArenaProfile(0); ok {
		t.Error("untagged, association-free unit has a profile; want silence")
	}
}

// Stats identity and determinism across 25 rebuilds.
func TestArenaStatsAndDeterminism(t *testing.T) {
	var units []parser.CodeUnit
	units = append(units, sqlUnit("subject", "p", "db_access", "error_wrapping"))
	for i := 0; i < 3; i++ {
		units = append(units, sqlUnit(name("db", i), "p", "db_access"))
	}
	for i := 0; i < 4; i++ {
		units = append(units, unit(name("tx", i), "p", "transaction", "caching"))
	}
	for i := 0; i < 4; i++ {
		units = append(units, unit(name("f", i), "p"))
	}
	docs := docsWithRole(len(units), "leaf")

	first := buildOn(t, units, docs, DefaultOptions())
	fs := first.Stats()
	if fs.ArenaProfiled != fs.ArenaDominance+fs.ArenaCoalition+fs.ArenaConflict+fs.ArenaWeak {
		t.Errorf("profiled %d != state sum %d", fs.ArenaProfiled,
			fs.ArenaDominance+fs.ArenaCoalition+fs.ArenaConflict+fs.ArenaWeak)
	}
	if fs.ArenaProfiled == 0 {
		t.Fatal("fixture broken: nothing profiled")
	}

	snapshot := func(m *Model) string {
		out := fmt.Sprintf("%+v\n", m.Stats())
		for i := range units {
			p, ok := m.ArenaProfile(i)
			out += fmt.Sprintf("%d %v %+v\n", i, ok, p)
		}
		return out
	}
	want := snapshot(first)
	for run := 0; run < 25; run++ {
		if got := snapshot(buildOn(t, units, docs, DefaultOptions())); got != want {
			t.Fatalf("run %d diverged:\n%s\nvs\n%s", run, got, want)
		}
	}
}

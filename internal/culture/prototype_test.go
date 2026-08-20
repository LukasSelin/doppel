package culture

import (
	"math"
	"reflect"
	"testing"

	"github.com/lukse/doppel/internal/concepter"
	"github.com/lukse/doppel/internal/fingerprint"
	"github.com/lukse/doppel/internal/parser"
)

func TestChannelWeightsSumToOne(t *testing.T) {
	sum := 0
	for _, w := range channelWeightPct {
		sum += w
	}
	if sum != 100 {
		t.Fatalf("channel weight percentages sum to %d, want exactly 100", sum)
	}
	if len(channelNames) != len(channelWeightPct) {
		t.Fatalf("%d names for %d weights", len(channelNames), len(channelWeightPct))
	}
}

// flowOf builds a length-10 histogram with the named slots set to 1.
func flowOf(slots ...int) []int {
	f := make([]int, len(fingerprint.FlowLabels))
	for _, s := range slots {
		f[s] = 1
	}
	return f
}

// cloneAlienFixture: five clone members of db_access sharing every feature,
// plus one alien member sharing none. Hand-computed: each clone scores
// (5−1)/(6−1) = 0.8 on every channel → typicality 0.8; the alien scores 0
// everywhere; median = 0.8.
func cloneAlienFixture() ([]parser.CodeUnit, []concepter.ConceptDoc) {
	sqlCaller := func(nm, pkg string, flow []int, tags ...string) parser.CodeUnit {
		u := parser.CodeUnit{Name: nm, Package: pkg, Patterns: tags}
		u.Fingerprint.Flow = flow
		u.Callees = []string{"sql.Open"}
		u.Signals = parser.TagSignals{PackageRefs: []parser.PackageRef{{Local: "sql", Path: "database/sql"}}}
		return u
	}
	var units []parser.CodeUnit
	docs := make([]concepter.ConceptDoc, 0, 6)
	for i := 0; i < 5; i++ {
		units = append(units, sqlCaller(name("clone", i), "store", flowOf(0, 6), "db_access", "error_wrapping"))
		docs = append(docs, concepter.ConceptDoc{Role: "leaf"})
	}
	alien := parser.CodeUnit{Name: "alien", Package: "exotic", Patterns: []string{"db_access"}}
	alien.Fingerprint.Flow = flowOf(5, 8) // select + go: unique flow features
	units = append(units, alien)
	docs = append(docs, concepter.ConceptDoc{Role: "orchestrator"})
	return units, docs
}

func TestTypicalityCloneVsAlien(t *testing.T) {
	units, docs := cloneAlienFixture()
	m := buildOn(t, units, docs, DefaultOptions())

	for i := 0; i < 5; i++ {
		typ, ok := m.Typicality(i, "db_access")
		if !ok || math.Abs(typ-0.8) > 1e-12 {
			t.Errorf("clone %d typicality = (%v, %v), want exactly 0.8", i, typ, ok)
		}
		if m.Atypical(i, "db_access") {
			t.Errorf("clone %d flagged atypical", i)
		}
	}

	typ, ok := m.Typicality(5, "db_access")
	if !ok || typ != 0 {
		t.Errorf("alien typicality = (%v, %v), want exactly 0", typ, ok)
	}
	if !m.Atypical(5, "db_access") {
		t.Error("alien not flagged atypical")
	}
	if med, ok := m.Median("db_access"); !ok || math.Abs(med-0.8) > 1e-12 {
		t.Errorf("median = (%v, %v), want 0.8", med, ok)
	}

	// Per-channel view: alien is 0 on every channel; unique features (its
	// select/go flow) contribute 0 under leave-one-out, never 1/m.
	channels := m.ChannelTypicality(5, "db_access")
	if len(channels) != len(channelNames) {
		t.Fatalf("got %d channels, want %d", len(channels), len(channelNames))
	}
	for _, ch := range channels {
		if ch.Typicality != 0 {
			t.Errorf("alien channel %s = %v, want 0", ch.Name, ch.Typicality)
		}
	}

	stats := m.Stats()
	// db_access (6 members) and error_wrapping (the 5 clones) both reach the
	// member floor; only the alien is unusual anywhere.
	if stats.ConceptsModeled != 2 {
		t.Errorf("ConceptsModeled = %d, want 2 (db_access and error_wrapping)", stats.ConceptsModeled)
	}
	if stats.UnusualRealizations != 1 {
		t.Errorf("UnusualRealizations = %d, want 1 (the alien)", stats.UnusualRealizations)
	}
}

func TestPrototypeDistributions(t *testing.T) {
	units, docs := cloneAlienFixture()
	m := buildOn(t, units, docs, DefaultOptions())

	proto, ok := m.Prototype("db_access")
	if !ok {
		t.Fatal("no prototype for db_access")
	}
	if got := len(proto.Channels); got != len(channelNames) {
		t.Fatalf("prototype has %d channels, want %d", got, len(channelNames))
	}
	calls := proto.Channels[0]
	if calls.Name != "calls" {
		t.Fatalf("first channel = %s, want calls", calls.Name)
	}
	want := []Feature{{Name: "database/sql.Open", P: 5.0 / 6.0}}
	if !reflect.DeepEqual(calls.Features, want) {
		t.Errorf("calls prototype = %+v, want %+v", calls.Features, want)
	}
	// Flow channel: if/return at 5/6 each, alien's go/select at 1/6; order is
	// (P desc, name asc).
	flow := proto.Channels[1]
	var names []string
	for _, f := range flow.Features {
		names = append(names, f.Name)
	}
	if !reflect.DeepEqual(names, []string{"if", "return", "go", "select"}) {
		t.Errorf("flow prototype order = %v, want [if return go select]", names)
	}
}

// Doing nothing can itself be the norm: with 4 of 6 members carrying no
// co-tags, a co-tag-less member scores (4−1)/(6−1) = 0.6 on that channel.
func TestTypicalityEmptySetConvention(t *testing.T) {
	var units []parser.CodeUnit
	for i := 0; i < 4; i++ {
		units = append(units, unit(name("bare", i), "p", "retry"))
	}
	units = append(units, unit("richA", "p", "retry", "http_call"))
	units = append(units, unit("richB", "p", "retry", "http_call"))
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())

	channels := m.ChannelTypicality(0, "retry")
	if channels == nil {
		t.Fatal("no channel typicality for member 0")
	}
	var cotags float64
	for _, ch := range channels {
		if ch.Name == "cotags" {
			cotags = ch.Typicality
		}
	}
	if math.Abs(cotags-0.6) > 1e-12 {
		t.Errorf("empty-cotags member scores %v, want 3/5 = 0.6", cotags)
	}
}

// Concepts below MinConceptMembers stay silent everywhere.
func TestSmallConceptSilence(t *testing.T) {
	var units []parser.CodeUnit
	for i := 0; i < 4; i++ {
		units = append(units, unit(name("t", i), "p", "transaction"))
	}
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())

	if _, ok := m.Typicality(0, "transaction"); ok {
		t.Error("Typicality reported for a 4-member concept")
	}
	if m.Atypical(0, "transaction") {
		t.Error("Atypical fired for a 4-member concept")
	}
	if _, ok := m.Prototype("transaction"); ok {
		t.Error("Prototype exists for a 4-member concept")
	}
	if s := m.Stats(); s.ConceptsModeled != 0 || s.UnusualRealizations != 0 {
		t.Errorf("stats = %+v, want zero concepts and realizations", s)
	}
}

// Identical members are perfectly typical, and an odd member count exercises
// the middle-element median.
func TestTypicalityIdenticalMembersOddMedian(t *testing.T) {
	var units []parser.CodeUnit
	for i := 0; i < 5; i++ {
		u := unit(name("same", i), "p", "caching")
		u.Fingerprint.Flow = flowOf(0)
		units = append(units, u)
	}
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())
	for i := 0; i < 5; i++ {
		if typ, ok := m.Typicality(i, "caching"); !ok || typ != 1.0 {
			t.Errorf("member %d typicality = (%v, %v), want exactly 1.0", i, typ, ok)
		}
	}
	if med, ok := m.Median("caching"); !ok || med != 1.0 {
		t.Errorf("median = (%v, %v), want 1.0", med, ok)
	}
}

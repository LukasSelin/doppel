package cmd

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/calibrate"
	"github.com/LukasSelin/doppel/internal/culture"
)

func TestPrintHabitatSummary(t *testing.T) {
	var b strings.Builder
	printHabitatSummary(&b, culture.Stats{
		ConceptsModeled:             3,
		HabitatsModeled:             14,
		HabitatMisfits:              23,
		MostUniformHabitat:          "cmd",
		MostUniformNorm:             0.97,
		MostDiverseHabitat:          "store",
		MostDiverseNorm:             0.62,
		StrongestConvention:         "transaction",
		StrongestConventionStrength: 0.93,
		LoosestConvention:           "retry",
		LoosestConventionStrength:   0.41,
	})
	out := b.String()
	if !strings.Contains(out,
		"Habitats: 14 modeled, 23 misfits; most uniform cmd (norm 0.97), most diverse store (norm 0.62)") {
		t.Errorf("missing habitats line:\n%s", out)
	}
	if !strings.Contains(out, "Conventions: strongest transaction (0.93), loosest retry (0.41)") {
		t.Errorf("missing conventions line:\n%s", out)
	}
}

func TestPrintHabitatSummaryEmpty(t *testing.T) {
	var b strings.Builder
	printHabitatSummary(&b, culture.Stats{})
	if got := b.String(); got != "Habitats: 0 modeled\n" {
		t.Errorf("empty summary = %q, want only the zero-habitats line", got)
	}
}

func TestPrintArenaSummary(t *testing.T) {
	var b strings.Builder
	printArenaSummary(&b, culture.Stats{
		ArenaProfiled: 412, ArenaDominance: 118, ArenaCoalition: 231,
		ArenaConflict: 12, ArenaWeak: 51,
	})
	if got := b.String(); got != "Ecosystems: 412 profiled (118 dominance, 231 coalition, 12 conflict, 51 weak)\n" {
		t.Errorf("summary = %q", got)
	}
	b.Reset()
	printArenaSummary(&b, culture.Stats{})
	if got := b.String(); got != "Ecosystems: 0 profiled\n" {
		t.Errorf("empty summary = %q", got)
	}
}

// With subsystems modeled the line carries the excused count, so the rollup
// never hides a package misfit silently.
func TestPrintHabitatSummaryWithSubsystems(t *testing.T) {
	var b strings.Builder
	printHabitatSummary(&b, culture.Stats{
		HabitatsModeled:    126,
		HabitatMisfits:     88,
		MisfitsExcused:     571,
		SubsystemsModeled:  23,
		MostUniformHabitat: "partials",
		MostUniformNorm:    0.98,
		MostDiverseHabitat: "page",
		MostDiverseNorm:    0.61,
	})
	want := "Habitats: 126 modeled, 88 misfits (571 excused by subsystem), 23 subsystems; most uniform partials (norm 0.98), most diverse page (norm 0.61)\n"
	if got := b.String(); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestPrintCalibration(t *testing.T) {
	var b strings.Builder
	printCalibration(&b, calibrate.Result{Rate: 0.01, ShapePairs: 20000, OverlapPairs: 20000, Threshold: 0.63, StructMin: 0.37})
	want := "Calibration: rate 0.01 over 20000 null pairs -> threshold 0.63, struct-min 0.37, family-min 0.63\n"
	if got := b.String(); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	b.Reset()
	printCalibration(&b, calibrate.Result{Rate: 0.01, ShapePairs: 17578, OverlapPairs: 20000, Threshold: 0.53, StructMin: 0.44})
	want = "Calibration: rate 0.01 over 17578 shape / 20000 overlap null pairs -> threshold 0.53, struct-min 0.44, family-min 0.53\n"
	if got := b.String(); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	b.Reset()
	printCalibration(&b, calibrate.Result{Rate: 0.01, ShapePairs: 780, Declined: "only 780 eligible shape null pairs (need 1000)"})
	want = "Calibration: rate 0.01 declined (only 780 eligible shape null pairs (need 1000)); defaults kept\n"
	if got := b.String(); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

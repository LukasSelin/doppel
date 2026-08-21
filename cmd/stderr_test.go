package cmd

import (
	"strings"
	"testing"

	"github.com/lukse/doppel/internal/culture"
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

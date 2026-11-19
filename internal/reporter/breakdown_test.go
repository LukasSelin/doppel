package reporter

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/fingerprint"
)

// An unmeasured component reads n/a, never 0.00: it is not part of the score,
// and a zero would read as two bodies disagreeing. A measured breakdown
// renders exactly as it always did.
func TestBreakdownLineMarksUnmeasured(t *testing.T) {
	measured := fingerprint.Breakdown{WL: 0.5, Flow: 1, Depth: 0.8, Signature: 0.25, SizeRatio: 0.4}
	if got, want := breakdownLine(measured), "wl 0.50  flow 1.00  nesting 0.80  sig 0.25  size 0.40"; got != want {
		t.Errorf("measured: %q, want %q", got, want)
	}
	absent := fingerprint.Breakdown{WL: 0.5, Flow: 1, SizeRatio: 0.4, NoDepth: true, NoSignature: true}
	if got, want := breakdownLine(absent), "wl 0.50  flow 1.00  nesting n/a  sig n/a  size 0.40"; got != want {
		t.Errorf("unmeasured: %q, want %q", got, want)
	}
}

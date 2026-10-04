package reporter

import (
	"fmt"
	"io"
	"strings"

	"github.com/LukasSelin/doppel/internal/fingerprint"
)

// flowViewLine renders the flow view of a pair, or "" when the pair carries none.
// The counts stay beside the ratio for the reason the practice section uses
// counts: "12 of 15 steps" is what was measured, and 0.80 alone hides whether
// a pair is two steps long or two hundred.
func flowViewLine(fs *fingerprint.FlowScore) string {
	if fs == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "flow: steps %.2f (%d same", fs.Steps, fs.Same)
	if fs.Retargeted > 0 {
		fmt.Fprintf(&b, ", %d retargeted", fs.Retargeted)
	}
	fmt.Fprintf(&b, " of %d and %d", fs.LenA, fs.LenB)
	if fs.Truncated {
		fmt.Fprintf(&b, ", truncated at %d", fingerprint.MaxFlowSteps)
	}
	fmt.Fprintf(&b, ")  types %.2f", fs.Types)
	return b.String()
}

// flowAlignWidth is the column the B side starts at in a rendered alignment.
const flowAlignWidth = 28

// printFlowAlignment writes the aligned step sequences, one row per step:
// "=" an identical step, "~" the same kind against a different target, and a
// blank where one side has a step the other does not. Only under --debug — it
// is as long as the longer function's logic.
func printFlowAlignment(w io.Writer, indent string, a, b fingerprint.Fingerprint) {
	for _, r := range fingerprint.FlowAlign(a, b) {
		mark, left, right := " ", "", ""
		switch r.Match {
		case 2:
			mark = "="
		case 1:
			mark = "~"
		}
		if r.A >= 0 {
			left = a.Steps[r.A].String()
		}
		if r.B >= 0 {
			right = b.Steps[r.B].String()
		}
		fmt.Fprintf(w, "%s%s %-*s %s\n", indent, mark, flowAlignWidth, left, right)
	}
}

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/calibrate"
	"github.com/LukasSelin/doppel/internal/reporter"
)

// A calibrated run's header states the threshold the run used, not the flag's
// static fallback. The pipeline writes the derived value into res.Params (the
// value snapshot.Params records), while the --threshold package variable keeps
// defaultThreshold — so reading the variable printed 0.38 over every report.
func TestReportMetaShowsCalibratedThreshold(t *testing.T) {
	old := threshold
	threshold = defaultThreshold
	t.Cleanup(func() { threshold = old })

	const calibrated = 0.41
	if calibrated == defaultThreshold {
		t.Fatal("fixture must differ from the static default to mean anything")
	}
	res := Result{
		Params:      Params{Threshold: calibrated, Calibrate: defaultCalibrateRate},
		Calibration: &calibrate.Result{Rate: defaultCalibrateRate, Threshold: calibrated},
	}
	meta := reportMeta(res, false)
	if meta.Threshold != calibrated {
		t.Fatalf("Meta.Threshold = %.2f, want the calibrated %.2f", meta.Threshold, calibrated)
	}

	var text, md bytes.Buffer
	reporter.Print(&text, nil, res.Units, meta)
	reporter.PrintMarkdown(&md, nil, res.Units, meta)
	if !strings.Contains(text.String(), "Threshold: 0.41") {
		t.Errorf("text header does not show the calibrated threshold:\n%s", text.String())
	}
	if !strings.Contains(md.String(), "**Threshold:** 0.41") {
		t.Errorf("markdown header does not show the calibrated threshold:\n%s", md.String())
	}
}

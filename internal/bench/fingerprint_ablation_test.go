package bench

import (
	"fmt"
	"os"
	"testing"

	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// fpComponent is one of the four fingerprint blend components (see
// fingerprint.DefaultWeights: AST 0.60, Flow 0.20, Depth 0.05, Signature
// 0.15), ablated by zeroing it outright.
//
// This is deliberately NOT fingerprint.Weights.Scaled, which renormalizes
// the other three so the blend still sums to 1.0 — the right question for a
// partial x0.5/x2 sensitivity move (see sweepVariants), but the wrong one
// for an ablation: renormalizing a zeroed component back up to a full 1.0
// would answer "what happens once the others compensate for its absence",
// not "does this component pull its own weight". Zeroing without
// renormalizing leaves the blend summing to less than 1.0 and answers the
// second question, which is the one an ablation table is for.
type fpComponent struct {
	name string
	zero func(fingerprint.Weights) fingerprint.Weights
}

func fpComponents() []fpComponent {
	return []fpComponent{
		{"WL 0.60", func(w fingerprint.Weights) fingerprint.Weights { w.WL = 0; return w }},
		{"Flow 0.20", func(w fingerprint.Weights) fingerprint.Weights { w.Flow = 0; return w }},
		{"Depth 0.05", func(w fingerprint.Weights) fingerprint.Weights { w.Depth = 0; return w }},
		{"Signature 0.15", func(w fingerprint.Weights) fingerprint.Weights { w.Signature = 0; return w }},
	}
}

// TestAblateFingerprint zeroes each of the four fingerprint blend components
// in turn — NOT renormalized, see fpComponent — and rescores every available
// labeled corpus through Reretrieve (retriever.Options.Weights is the
// measurement seam; the shape channel's admission gate and every
// code-shape score move with it, exactly as a production --threshold run
// would see a different blend). One row per component per corpus.
//
// It asserts nothing: like TestAblation (the ontology relation-weight
// ablation this deliberately parallels) and TestSweep, this is a
// measurement of which components carry their own weight, not a gate.
//
//	DOPPEL_BENCH_ABLATE=1 go test ./internal/bench/ -v -run TestAblateFingerprint
func TestAblateFingerprint(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_ABLATE") != "1" {
		t.Skip("set DOPPEL_BENCH_ABLATE=1 to run the fingerprint component ablation")
	}
	corpora := loadLabeledCorpora(t)
	if len(corpora) == 0 {
		t.Skip("no labeled corpora available; run `task corpora` or set DOPPEL_BENCH_CORPUS/DOPPEL_BENCH_LABELS")
	}

	t.Logf("each row zeroes one fingerprint blend component outright; the remaining three keep their default weight (not renormalized to 1.0)")
	for _, lc := range corpora {
		base := Score(lc.run, lc.lf)
		t.Logf("[%s] baseline                %s", lc.name, scLine(base))

		saved := snapshotRetrieval(lc.run)
		for _, comp := range fpComponents() {
			opt := retriever.DefaultOptions()
			opt.Weights = comp.zero(fingerprint.DefaultWeights())
			lc.run.Reretrieve(opt)
			sc := Score(lc.run, lc.lf)

			delta := "-"
			if base.Present["merge"] > 0 && sc.Present["merge"] > 0 {
				delta = fmt.Sprintf("%+.1f", sc.MeanRank["merge"]-base.MeanRank["merge"])
			}
			t.Logf("[%s] zero %-16s   %s  (merge mean delta vs baseline: %s)", lc.name, comp.name, scLine(sc), delta)
		}
		saved.restore(lc.run)
	}
}

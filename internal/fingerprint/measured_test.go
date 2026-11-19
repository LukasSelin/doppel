package fingerprint

import (
	"math"
	"testing"
)

// unmeasured marks a fingerprint the way the lexical frontend's do: no types
// read, and nesting not attached.
func unmeasured(fp Fingerprint, types, nesting bool) Fingerprint {
	fp.NoTypes, fp.NoNesting = types, nesting
	fp.Types = nil
	return fp
}

// A component one side could not measure is left out of Score, and a perfect
// match on what was measured still reaches the blend total. Before, two empty
// type sets and two flat depth histograms both read 1.0, so every pair the
// lexical frontend produced carried 0.20 of agreement nobody had measured.
func TestUnmeasuredComponentsAreLeftOut(t *testing.T) {
	a := unmeasured(build(t, srcSum), true, true)
	b := unmeasured(build(t, srcSumRenamed), true, true)

	bd := sim(a, a)
	if !bd.NoSignature || !bd.NoDepth {
		t.Fatalf("flags not set: %+v", bd)
	}
	if bd.Signature != 0 || bd.Depth != 0 {
		t.Errorf("unmeasured components read %v / %v, want 0", bd.Signature, bd.Depth)
	}
	if math.Abs(bd.Score-1) > 1e-12 {
		t.Errorf("self-similarity with two components unmeasured = %v, want 1", bd.Score)
	}

	// The renormalised blend is what Score is, term for term.
	bd = sim(a, b)
	w := DefaultWeights().Effective(bd)
	want := float64(w.WL*bd.WL) + float64(w.Flow*bd.Flow)
	if bd.Score != want {
		t.Errorf("Score = %v, want WL and flow under the effective blend = %v", bd.Score, want)
	}
}

// The floor this exists to remove: two bodies sharing no label evidence at
// all, with the same control flow. Measured, an untyped and unnested pair
// scores only the flow share of the blend; it used to score 0.40.
func TestUnmeasuredPairHasNoFreeFloor(t *testing.T) {
	a := unmeasured(build(t, srcSum), true, true)
	b := a
	b.WL = []LabelCount{{Label: 1, Count: 1}} // nothing in common with a's bag
	bd := sim(a, b)
	if bd.WL != 0 || math.Abs(bd.Flow-1) > 1e-12 {
		t.Fatalf("fixture: wl %v flow %v, want 0 and 1", bd.WL, bd.Flow)
	}
	flowShare := weightFlow / (weightWL + weightFlow)
	if math.Abs(bd.Score-flowShare) > 1e-9 {
		t.Errorf("Score = %v, want the flow share %v", bd.Score, flowShare)
	}

	// Typed, nested and otherwise identical, the same pair scores what it
	// always did: flow, nesting and signature all agree.
	m, n := build(t, srcSum), build(t, srcSum)
	n.WL = b.WL
	if got := sim(m, n).Score; math.Abs(got-(weightFlow+weightDepth+weightSignature)) > 1e-9 {
		t.Errorf("measured pair = %v, want %v", got, weightFlow+weightDepth+weightSignature)
	}
}

// One unmeasured side is enough: a component is part of the score only when
// both sides measured it.
func TestOneUnmeasuredSideLeavesComponentOut(t *testing.T) {
	a := build(t, srcSum)
	b := unmeasured(build(t, srcSumRenamed), true, false)
	bd := sim(a, b)
	if !bd.NoSignature || bd.NoDepth {
		t.Errorf("flags = sig %v depth %v, want true false", bd.NoSignature, bd.NoDepth)
	}
	if bd != sim(b, a) {
		t.Errorf("not symmetric with one side unmeasured")
	}
}

// A measured pair's score is the four-term sum it always was, bit for bit:
// Effective hands back the weights untouched, so no Go score can move.
func TestMeasuredScoreIsUnchanged(t *testing.T) {
	fps := []Fingerprint{build(t, srcSum), build(t, srcSumRenamed), build(t, srcServe), build(t, srcGetter)}
	w := DefaultWeights()
	for _, a := range fps {
		for _, b := range fps {
			bd := sim(a, b)
			if bd.NoDepth || bd.NoSignature {
				t.Fatalf("a gofront fingerprint read as unmeasured")
			}
			old := float64(w.WL*bd.WL) + float64(w.Flow*bd.Flow) + float64(w.Depth*bd.Depth) + float64(w.Signature*bd.Signature)
			if old > 1 {
				old = 1
			}
			if bd.Score != old {
				t.Fatalf("Score %v, want the pre-existing blend %v", bd.Score, old)
			}
		}
	}
	if DefaultWeights().Effective(Breakdown{}) != DefaultWeights() {
		t.Errorf("Effective moved the weights of a fully measured pair")
	}
}

// Effective is Scaled with a zero factor, once per absent component, so the
// blend total is kept and the measured components keep their ratios.
func TestEffectiveRenormalises(t *testing.T) {
	w := DefaultWeights().Effective(Breakdown{NoDepth: true, NoSignature: true})
	if w.Depth != 0 || w.Signature != 0 {
		t.Errorf("absent weights = %v / %v, want 0", w.Depth, w.Signature)
	}
	if d := w.Sum() - DefaultWeights().Sum(); math.Abs(d) > 1e-12 {
		t.Errorf("total = %v, want %v", w.Sum(), DefaultWeights().Sum())
	}
	if r := w.WL / w.Flow; math.Abs(r-weightWL/weightFlow) > 1e-12 {
		t.Errorf("WL:flow = %v, want %v", r, weightWL/weightFlow)
	}
}

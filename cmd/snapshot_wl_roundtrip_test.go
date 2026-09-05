package cmd

import (
	"encoding/json"
	"io"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/LukasSelin/doppel/internal/canon"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// repoRoot resolves doppel's own module root from this test file's location,
// so the round trip below runs on a real, fully-featured corpus (this repo
// carries the self-documented exact clones CLAUDE.md names — see
// cmd.validateMode et al.) regardless of the directory `go test` is invoked
// from.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// cmd/snapshot_wl_roundtrip_test.go -> repo root is one level up.
	return filepath.Dir(filepath.Dir(file))
}

// TestSnapshotWLBagRoundTrip is T6's gate.
//
// It runs the real pipeline on doppel's own tree, serializes the result to
// the exact bytes `--format json` and a session baseline both write,
// deserializes it back, and recomputes — from nothing but what the file
// stored — the two WL-based quantities every reported pair carries:
//
//   - the WL Jaccard component of fingerprint.Breakdown (code-shape's 0.60
//     term), by decoding every unit's bag and rebuilding a fresh LabelIDF
//     from exactly that population with fingerprint.LabelWeights, then
//     calling fingerprint.WLOverlap on the two sides' decoded bags;
//   - Containment, the same call's second return value.
//
// Both must be bit-identical (Go == on float64, not an epsilon compare) to
// what the live run computed. They are, because Snapshot stores every unit
// the run saw — the WL population Build's caller counted over is exactly
// the population LabelWeights sees again here — and wlOverlap (reached
// through WLOverlap) is a pure function of two bags and a weighting: same
// bags, same weights in, same floats out.
//
// Score cannot be recomputed this way and this test does not try: Flow,
// Depth and Signature are not part of the stored Unit (only Digest, an
// opaque hash, and the WL bag itself — "only what a consumer reads is
// stored"), so the composite four-term blend is unrecoverable from a
// snapshot by design. The stored Pair.Score and Pair.Containment floats
// themselves are checked separately, for exact JSON round-trip fidelity
// rather than recomputation.
func TestSnapshotWLBagRoundTrip(t *testing.T) {
	root := repoRoot(t)
	p := Params{Threshold: 0.60, MinNodes: 12, ChannelK: 5, TestsMode: "exclude", Generated: "exclude"}
	res, err := analyze(root, p, io.Discard)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(res.Pairs) == 0 {
		t.Fatal("no pairs retrieved analyzing doppel's own tree; the round trip needs at least one")
	}

	live := snapshotOf(res, res.Pairs)

	data, err := json.Marshal(live)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var parsed snapshot.Snapshot
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed.RuleSet != canon.Version {
		t.Errorf("RuleSet round-tripped as %q, want canon.Version %q", parsed.RuleSet, canon.Version)
	}
	if parsed.Schema != snapshot.Schema {
		t.Errorf("Schema round-tripped as %d, want %d", parsed.Schema, snapshot.Schema)
	}

	// Stored Pair.Score/Containment must round-trip as exact JSON numbers.
	// live.Pairs and parsed.Pairs are the same JSON array, so they are the
	// same length in the same order; comparing by index is exact rather than
	// keyed, on purpose, since this half of the gate is about serialization
	// fidelity, not about recomputing anything.
	if len(live.Pairs) != len(parsed.Pairs) {
		t.Fatalf("pair count changed across the round trip: %d -> %d", len(live.Pairs), len(parsed.Pairs))
	}
	for i := range live.Pairs {
		if live.Pairs[i].Score != parsed.Pairs[i].Score {
			t.Errorf("pair %d (%s<->%s): Score %v round-tripped as %v",
				i, live.Pairs[i].A, live.Pairs[i].B, live.Pairs[i].Score, parsed.Pairs[i].Score)
		}
		if live.Pairs[i].Containment != parsed.Pairs[i].Containment {
			t.Errorf("pair %d (%s<->%s): Containment %v round-tripped as %v",
				i, live.Pairs[i].A, live.Pairs[i].B, live.Pairs[i].Containment, parsed.Pairs[i].Containment)
		}
	}

	// Decode the run's shared label dictionary once, then every parsed
	// unit's bag against it, and rebuild the population's own label weights
	// from exactly that population — the snapshot's Units field is the
	// whole run's population (no top-N, no struct-min filter touches
	// Units), which is what makes this the same multiset cmd/pipeline.go's
	// index() built res.WL from. LabelWeights documents that only the
	// multiset of bags matters, never their order, so rebuilding it here
	// from Units (sorted by key, not by walk order) reproduces the exact
	// same ln(N/df) table.
	dict, err := fingerprint.DecodeLabelDict(parsed.Labels)
	if err != nil {
		t.Fatalf("decode label dictionary: %v", err)
	}
	bagByKey := make(map[string][]fingerprint.LabelCount, len(parsed.Units))
	bags := make([][]fingerprint.LabelCount, len(parsed.Units))
	for i, u := range parsed.Units {
		bag, err := fingerprint.DecodeWLBagIndexed(u.WL, dict)
		if err != nil {
			t.Fatalf("decode WL bag for %s: %v", u.Key, err)
		}
		bagByKey[u.Key] = bag
		bags[i] = bag
	}
	idf := fingerprint.LabelWeights(bags)

	// Correlate every live pair (which carries the exact fingerprint.Breakdown
	// the pipeline computed) to its snapshot key, via file+line — a location
	// is unique per function in a real repo, and both res.Units (via pr.A/B)
	// and live.Units (via the same snapshot.RelSlash the pipeline used) agree
	// on it.
	locToKey := make(map[string]string, len(live.Units))
	for _, u := range live.Units {
		locToKey[u.File+":"+strconv.Itoa(u.Line)] = u.Key
	}
	liveWL := make(map[string]float64, len(res.Pairs))
	unresolved := 0
	for _, pr := range res.Pairs {
		ua, ub := res.Units[pr.AIdx], res.Units[pr.BIdx]
		ka, oka := locToKey[snapshot.RelSlash(root, ua.File)+":"+strconv.Itoa(ua.StartLine)]
		kb, okb := locToKey[snapshot.RelSlash(root, ub.File)+":"+strconv.Itoa(ub.StartLine)]
		if !oka || !okb {
			unresolved++
			continue
		}
		if ka > kb {
			ka, kb = kb, ka
		}
		liveWL[ka+"|"+kb] = pr.Breakdown.WL
	}
	if unresolved > 0 {
		t.Fatalf("%d of %d live pairs could not be resolved to snapshot keys by location", unresolved, len(res.Pairs))
	}

	checked := 0
	for _, sp := range parsed.Pairs {
		a, ok1 := bagByKey[sp.A]
		b, ok2 := bagByKey[sp.B]
		if !ok1 || !ok2 {
			t.Fatalf("pair %s<->%s references a unit missing from parsed.Units", sp.A, sp.B)
		}
		gotJaccard, gotContainment := fingerprint.WLOverlap(a, b, idf)

		wantJaccard, ok := liveWL[sp.A+"|"+sp.B]
		if !ok {
			t.Fatalf("no live WL value found for pair %s<->%s", sp.A, sp.B)
		}
		if gotJaccard != wantJaccard {
			t.Errorf("pair %s<->%s: recomputed WL Jaccard %v, want bit-identical live value %v",
				sp.A, sp.B, gotJaccard, wantJaccard)
		}
		if gotContainment != sp.Containment {
			t.Errorf("pair %s<->%s: recomputed Containment %v, want bit-identical stored value %v",
				sp.A, sp.B, gotContainment, sp.Containment)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no pairs verified")
	}
	t.Logf("recomputed WL Jaccard + Containment for %d pairs from parsed bags alone, bit-identical to the live run", checked)
}

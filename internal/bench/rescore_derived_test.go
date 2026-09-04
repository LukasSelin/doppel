package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// derivedFixture is a synthetic corpus that grows a learned vocabulary under
// production lexicon settings, so a Rescore can be checked against the
// taxonomy a real run reasons over rather than the seed one. No real corpus
// identifiers, per the leak rule.
//
// Three practices of six functions each, every one clearing MinMembers (5)
// and MinSupport (3): readers that fire the db_access seed (receiver "db") and
// share two selectors nobody else uses; posters that fire the transaction seed
// (receiver "tx") and share two others; validators that fire validation
// (identifier "Validate") and share a third pair. The first two land under
// data_store_access, the third under data_transformation — which is what keeps
// IC(data_store_access) above zero, since an interior node every concept hangs
// from carries no information and its matches would be dropped as worthless.
//
// The filler is what makes each practice rare enough to be evidence (the
// information window is a fraction of the corpus), and each filler function
// calls names nobody else calls, so it founds nothing.
//
// Every reader and every poster calls stamp: a resolved internal callee with
// df 12, inside the call channel's window, so the channel retrieves
// reader/poster pairs (equal mass, lowest index first) whatever the concept
// channel's top-K decides. Every member calls it, so it dilutes nobody's
// coverage — an earlier draft gave stamp to one function per group, and those
// two fell below their concept's floor for carrying more than their siblings.
// A retrieved reader/poster pair carries one learned concept on each side and
// its concept evidence is their shared ancestor.
func derivedFixture() string {
	var b strings.Builder
	b.WriteString("package corpus\n\nfunc stamp(s string) string { return s + \"!\" }\n")

	names := []string{"One", "Two", "Three", "Four", "Five", "Six"}
	for _, n := range names {
		fmt.Fprintf(&b, `
func Fetch%s(db Conn, key string) (string, error) {
	row := db.QueryRow(stamp(key))
	v, err := repo.Fetch(row)
	if err != nil {
		return "", err
	}
	return repo.Unpack(v), nil
}
`, n)
	}
	for _, n := range names {
		fmt.Fprintf(&b, `
func Post%s(tx Tx, amount string) error {
	entry := ledger.Begin(stamp(amount))
	if err := ledger.Post(entry); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
`, n)
	}
	for _, n := range names {
		fmt.Fprintf(&b, `
func Validate%s(v int) error {
	if !check.Bound(v) {
		return check.Require(%q)
	}
	return nil
}
`, n, strings.ToLower(n))
	}
	for i := 0; i < 24; i++ {
		fmt.Fprintf(&b, "\nfunc Filler%02d(x int) int { return alpha%02d(x) + beta%02d(x) }\n", i, i, i)
	}
	return b.String()
}

// nonExact returns the matches in ms that pair two different concepts — the
// shared-ancestor credit a seed-vocabulary scorer cannot give.
func nonExact(ms []ontology.Match) []ontology.Match {
	var out []ontology.Match
	for _, m := range ms {
		if !m.Exact() {
			out = append(out, m)
		}
	}
	return out
}

// TestRescoreKeepsDerivedTaxonomy is the round-trip test that can see the
// vocabulary: under a reweighted copy of the run's own ontology a pair keeps
// every non-exact concept match and its PatternRelatedness bit for bit, while
// the same overrides applied to the seed vocabulary drop them — the defect the
// bench harness carried — and restoring the run's vocabulary restores every
// overlap score exactly.
func TestRescoreKeepsDerivedTaxonomy(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "corpus.go"), []byte(derivedFixture()), 0o644); err != nil {
		t.Fatal(err)
	}
	units, err := Load(dir, PopInclude)
	if err != nil {
		t.Fatal(err)
	}
	run := Analyze(units, retriever.DefaultOptions())
	saved := run.Onto

	// The probe: the first retrieved reader/poster pair. Positions are
	// file-walk order and the fixture is generated, so look it up by name
	// rather than assume an index.
	probe := -1
	for i, p := range run.Pairs {
		a, b := run.Units[p.AIdx].Name, run.Units[p.BIdx].Name
		fetchPost := strings.HasPrefix(a, "Fetch") && strings.HasPrefix(b, "Post")
		postFetch := strings.HasPrefix(a, "Post") && strings.HasPrefix(b, "Fetch")
		if fetchPost || postFetch {
			probe = i
			break
		}
	}
	if probe < 0 {
		t.Fatalf("no reader/poster pair was retrieved; %d pairs, concepts %v", len(run.Pairs), run.Lexicon.Concepts())
	}
	base := *run.Pairs[probe].Evidence
	related := nonExact(base.RelatedPatterns)
	if len(related) == 0 {
		t.Fatalf("probe pair has no non-exact concept match: patterns %+v; concepts A %v, B %v",
			base.RelatedPatterns, run.Units[run.Pairs[probe].AIdx].Concepts, run.Units[run.Pairs[probe].BIdx].Concepts)
	}
	for _, m := range related {
		if m.LCA != ontology.ConDataStoreAccess {
			t.Errorf("match %q~%q meets at %q, want %q", m.A, m.B, m.LCA, ontology.ConDataStoreAccess)
		}
		if _, ok := ontology.Default().Get(ontology.TermID(m.A)); ok {
			t.Errorf("%q is a seed concept, not a learned one; the fixture proves nothing", m.A)
		}
	}
	original := make([]float64, len(run.Pairs))
	for i, p := range run.Pairs {
		original[i] = p.Evidence.OverlapScore
	}

	// Reweighting the run's own vocabulary changes the composite and nothing
	// upstream of it: the matches and their relatedness are untouched.
	overrides := map[ontology.TermID]float64{ontology.RelCalls: 0}
	over, err := ontology.WithWeightsOver(saved, overrides)
	if err != nil {
		t.Fatal(err)
	}
	run.Rescore(over)
	got := *run.Pairs[probe].Evidence
	if !reflect.DeepEqual(got.RelatedPatterns, base.RelatedPatterns) {
		t.Errorf("reweighting the derived vocabulary changed the matches:\n got %+v\nwant %+v", got.RelatedPatterns, base.RelatedPatterns)
	}
	if got.PatternRelatedness != base.PatternRelatedness {
		t.Errorf("PatternRelatedness %v under reweighted derived vocabulary, want %v exactly", got.PatternRelatedness, base.PatternRelatedness)
	}
	changed := false
	for i, p := range run.Pairs {
		if p.Evidence.OverlapScore != original[i] {
			changed = true
			break
		}
	}
	if !changed {
		t.Error("zeroing the calls weight moved no overlap score; the seam is not reaching Compare")
	}

	// The contrast: the same overrides over the seed vocabulary lose every
	// non-exact match, because that taxonomy has never heard of the learned
	// leaves and LCA fails for any two distinct ones.
	seed, err := ontology.WithWeights(overrides)
	if err != nil {
		t.Fatal(err)
	}
	run.Rescore(seed)
	if left := nonExact(run.Pairs[probe].Evidence.RelatedPatterns); len(left) != 0 {
		t.Errorf("seed vocabulary kept non-exact matches %+v; the contrast this test rests on no longer holds", left)
	}

	run.Rescore(saved)
	for i, p := range run.Pairs {
		if p.Evidence.OverlapScore != original[i] {
			t.Fatalf("pair %d: overlap %v after round-trip, want %v exactly", i, p.Evidence.OverlapScore, original[i])
		}
	}
	if after := *run.Pairs[probe].Evidence; !reflect.DeepEqual(after.RelatedPatterns, base.RelatedPatterns) || after.PatternRelatedness != base.PatternRelatedness {
		t.Errorf("round trip did not restore the probe's matches: %+v", after.RelatedPatterns)
	}
}

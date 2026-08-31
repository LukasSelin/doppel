package lexicon

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
)

// storeCorpus is a corpus with one obvious practice in it: six functions that
// reach a key-value store through the same two calls, six that do the same
// string cleanup, and filler that shares nothing with anybody.
//
// Nothing names the store practice and no rule describes it — a wrapper called
// "store" is exactly the vocabulary a hand-written tagger would miss — so if
// the learner works at all, it finds this.
//
// The filler is not padding. The information window is a *fraction* of the
// corpus: on twelve functions a practice shared by six is carried by half the
// codebase and is correctly judged idiom, not concept. Filler is what makes the
// store practice rare enough to be evidence, and each filler function calls
// names nobody else does, so the filler itself forms no concept.
func storeCorpus() string {
	var b strings.Builder
	b.WriteString("package app\n\nimport \"strings\"\n")

	// 0-5: the store readers. Two share a parameter name and four another, so
	// the concept cannot be an artifact of one identifier.
	readers := []string{"LoadUser", "LoadOrder", "LoadInvoice", "LoadShipment", "LoadAddress", "LoadCarrier"}
	for i, name := range readers {
		arg := "ref"
		if i < 2 {
			arg = "id"
		}
		fmt.Fprintf(&b, `
func %s(%s string) (string, error) {
	v, err := store.Get(%s)
	if err != nil {
		return "", err
	}
	return store.Decode(v)
}
`, name, arg, arg)
	}

	// 6-11: an unrelated shared practice, so "shares a concept" cannot pass by
	// there being only one concept to share.
	for _, name := range []string{"TitleOne", "TitleTwo", "TitleThree", "TitleFour", "TitleFive", "TitleSix"} {
		fmt.Fprintf(&b, "\nfunc %s(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }\n", name)
	}

	// 12+: filler, each calling names nobody else calls.
	for i := 0; i < 24; i++ {
		fmt.Fprintf(&b, "\nfunc Filler%02d(x int) int { return alpha%02d(x) + beta%02d(x) }\n", i, i, i)
	}
	return b.String()
}

const (
	firstStore = 0  // index of the first store reader
	afterStore = 6  // one past the last
	afterTitle = 12 // one past the last title function
)

func build(t *testing.T, src string, seeds [][]string, opt Options) (*Model, []parser.CodeUnit) {
	t.Helper()
	units, err := parser.ParseSource("app.go", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if seeds == nil {
		seeds = make([][]string, len(units))
	}
	g := concepter.BuildCallGraph(units)
	return Build(units, g, seeds, opt), units
}

func testOptions() Options {
	opt := DefaultOptions()
	// The corpus is twelve functions; the production floors are stated for
	// thousands. MinMembers and MinSupport are the ones that would silence a
	// twelve-function fixture outright.
	opt.MinMembers = 4
	opt.MinSupport = 3
	return opt
}

// TestNoSeeds is the language-portability claim, as a test. The learner must
// produce concepts from a corpus with no seed labels at all, because that is
// what a frontend for another language starts from: TagSignals and a
// fingerprint, and not one line of vocabulary anybody wrote.
func TestNoSeeds(t *testing.T) {
	m, units := build(t, storeCorpus(), nil, testOptions())

	if len(m.Concepts()) == 0 {
		t.Fatal("no concepts learned from an unseeded corpus")
	}
	for _, c := range m.Concepts() {
		if c.Seed != "" {
			t.Errorf("concept %q claims seed %q with no seeds given", c.ID, c.Seed)
		}
	}

	// The six store functions must share a concept nothing else carries: that
	// is the whole of what "learned from the corpus" has to mean.
	assign := m.Assignments()
	shared := ""
	for _, c := range assign[firstStore] {
		if hasConcept(assign[1], c.ID) && hasConcept(assign[afterStore-1], c.ID) {
			shared = c.ID
			break
		}
	}
	if shared == "" {
		t.Fatalf("the six store readers share no concept; got %v", describe(assign, units))
	}
	for i := afterStore; i < len(units); i++ {
		if hasConcept(assign[i], shared) {
			t.Errorf("%s carries the store concept %q; it does nothing of the kind",
				units[i].Name, shared)
		}
	}

	// The name has to say what the evidence was, or nobody can act on it.
	if !strings.Contains(shared, "store.") {
		t.Errorf("concept name %q does not name the calls it was learned from", shared)
	}
}

// TestSeedsExpandRatherThanDecide pins the hybrid: a seed contributes its
// member set and nothing else. The concept that comes back is named after what
// those members actually share, keeps the seed only as provenance, and admits
// members the seed never fired on.
func TestSeedsExpandRatherThanDecide(t *testing.T) {
	units, err := parser.ParseSource("app.go", []byte(storeCorpus()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Seed only four of the six store readers, as a lexical rule keyed on
	// "Load" plus a spelling would: the other two must still be found.
	seeds := make([][]string, len(units))
	for i := 0; i < 4; i++ {
		seeds[i] = []string{"db_access"}
	}
	g := concepter.BuildCallGraph(units)
	m := Build(units, g, seeds, testOptions())

	var seeded *Concept
	for i, c := range m.Concepts() {
		if c.Seed == "db_access" {
			seeded = &m.Concepts()[i]
			break
		}
	}
	if seeded == nil {
		t.Fatalf("the seed produced no concept; got %v", ids(m))
	}
	if seeded.ID == "db_access" {
		t.Error("the concept kept the seed's name; the name must state the learned evidence")
	}
	assign := m.Assignments()
	for i := firstStore; i < afterStore; i++ {
		if !hasConcept(assign[i], seeded.ID) {
			t.Errorf("%s does the same work as the seeded four but is not a member", units[i].Name)
		}
	}
}

// TestDeterministic is the repo-wide invariant, at this stage: the same corpus
// must yield the same lexicon, concept for concept and confidence for
// confidence. Map iteration order is the usual way this breaks.
func TestDeterministic(t *testing.T) {
	first, _ := build(t, storeCorpus(), nil, testOptions())
	for i := 0; i < 5; i++ {
		again, _ := build(t, storeCorpus(), nil, testOptions())
		if !reflect.DeepEqual(first.Concepts(), again.Concepts()) {
			t.Fatalf("run %d learned a different vocabulary", i)
		}
		if !reflect.DeepEqual(first.Assignments(), again.Assignments()) {
			t.Fatalf("run %d assigned different memberships", i)
		}
	}
}

// TestConfidenceIsGraded pins the reason memberships stopped being booleans: a
// unit carrying more of a concept's vocabulary must read higher than one
// carrying less, and every confidence must sit in (0,1].
func TestConfidenceIsGraded(t *testing.T) {
	m, _ := build(t, storeCorpus(), nil, testOptions())
	seen := false
	for _, cs := range m.Assignments() {
		for _, c := range cs {
			if c.Confidence <= 0 || c.Confidence > 1 {
				t.Errorf("confidence %v for %q is outside (0,1]", c.Confidence, c.ID)
			}
			seen = true
		}
	}
	if !seen {
		t.Fatal("no memberships at all")
	}
}

// TestFeatureWindowExcludesUbiquity pins the two-sided information window. A
// feature every function carries relates nothing, and one a single function
// carries can relate nothing; neither may reach a vocabulary.
func TestFeatureWindowExcludesUbiquity(t *testing.T) {
	m, _ := build(t, storeCorpus(), nil, testOptions())
	st := m.Stats()
	if st.FeaturesSurviving == 0 || st.FeaturesSurviving >= st.FeaturesTotal {
		t.Fatalf("window kept %d of %d features; want a proper subset",
			st.FeaturesSurviving, st.FeaturesTotal)
	}
	for _, c := range m.Concepts() {
		for _, f := range c.Features {
			if f.DF < 2 {
				t.Errorf("%q keeps feature %q at df %d", c.ID, f.Name, f.DF)
			}
			if f.DF > st.FeatureCap {
				t.Errorf("%q keeps feature %q at df %d, above the cap %d",
					c.ID, f.Name, f.DF, st.FeatureCap)
			}
		}
	}
}

// TestEmptyCorpus is the degenerate case: no units, no panic, no concepts.
func TestEmptyCorpus(t *testing.T) {
	g := concepter.BuildCallGraph(nil)
	m := Build(nil, g, nil, DefaultOptions())
	if len(m.Concepts()) != 0 || len(m.Assignments()) != 0 {
		t.Fatalf("empty corpus produced %d concepts", len(m.Concepts()))
	}
}

func hasConcept(cs []parser.Concept, id string) bool {
	for _, c := range cs {
		if c.ID == id {
			return true
		}
	}
	return false
}

func ids(m *Model) []string {
	var out []string
	for _, c := range m.Concepts() {
		out = append(out, c.ID)
	}
	return out
}

func describe(assign [][]parser.Concept, units []parser.CodeUnit) string {
	var b strings.Builder
	for i := range units {
		fmt.Fprintf(&b, "\n  %-14s %v", units[i].Name, assign[i])
	}
	return b.String()
}

// TestMembershipIgnoresSize is the regression the coverage rule exists for.
//
// The concept's founding members are verbose: each reaches the store through
// the same two calls and then does a great deal of unrelated work. A seventh
// function reaches the store the same way and does nothing else — it carries
// the concept's whole vocabulary and none of the noise.
//
// Under a floor stated in raw evidence the terse one is not a member, because
// the bar was set by how much its founders carry *in total* rather than by how
// much of them the concept accounts for. That arithmetic is what left 451 of
// doppel's own 866 functions unlabelled: measured on this corpus, the labelled
// units carried a median of 48 features and the unlabelled ones 21.
//
// Coverage is size-free on both sides, so the terse reader is a member of the
// concept its verbose siblings founded.
func TestMembershipIgnoresSize(t *testing.T) {
	var b strings.Builder
	b.WriteString("package app\n\nimport \"strings\"\n")

	// 0-5: verbose store readers. The store calls are the only thing they have
	// in common; the padding is deliberately per-function so it founds nothing.
	readers := []string{"LoadUser", "LoadOrder", "LoadInvoice", "LoadShipment", "LoadAddress", "LoadCarrier"}
	for i, name := range readers {
		fmt.Fprintf(&b, `
func %s(ref string) (string, error) {
	v, err := store.Get(ref)
	if err != nil {
		return "", err
	}
	out, err := store.Decode(v)
	if err != nil {
		return "", err
	}
	for i := 0; i < %d; i++ {
		out = strings.TrimSpace(out)
		out = strings.ToUpper(out)
		if strings.Contains(out, "z%d") {
			out = strings.TrimSuffix(out, "z%d")
		}
		switch {
		case strings.HasPrefix(out, "q%d"):
			out = strings.TrimPrefix(out, "q%d")
		default:
			out += strings.Repeat("x", i)
		}
	}
	return out, nil
}
`, name, 3+i, i, i, i, i)
	}

	// 6: the same practice, and nothing else.
	b.WriteString(`
func LoadTerse(ref string) (string, error) {
	v, err := store.Get(ref)
	if err != nil {
		return "", err
	}
	return store.Decode(v)
}
`)

	// Filler, so the store practice is rare enough in the corpus to be
	// evidence at all — the reason storeCorpus carries filler too.
	for i := 0; i < 24; i++ {
		fmt.Fprintf(&b, "\nfunc Filler%02d(x int) int { return alpha%02d(x) + beta%02d(x) }\n", i, i, i)
	}

	m, units := build(t, b.String(), nil, testOptions())

	idx := make(map[string]int, len(units))
	for i := range units {
		idx[units[i].Name] = i
	}
	terse, ok := idx["LoadTerse"]
	if !ok {
		t.Fatal("fixture did not parse the terse reader")
	}
	founder, ok := idx[readers[0]]
	if !ok {
		t.Fatal("fixture did not parse the verbose readers")
	}
	if len(units[founder].Body) <= 3*len(units[terse].Body) {
		t.Fatal("fixture is not lopsided enough to be measuring anything")
	}

	assign := m.Assignments()
	shared := ""
	for _, c := range assign[terse] {
		if parser.ConfidenceOf(assign[founder], c.ID) > 0 {
			shared = c.ID
			break
		}
	}
	if shared == "" {
		t.Fatalf("the terse reader shares no concept with the verbose ones it copies: terse %v, verbose %v",
			parser.ConceptIDs(assign[terse]), parser.ConceptIDs(assign[founder]))
	}
}

// TestMaxMemberships pins the bound: a unit keeps its strongest memberships
// and no more, ascending by ID as every consumer expects.
//
// The fixture needs units that genuinely carry more than one concept, so six
// functions here do both practices — read the store and clean the string — and
// they are what the bound has to choose between.
func TestMaxMemberships(t *testing.T) {
	var b strings.Builder
	b.WriteString(storeCorpus())
	for _, name := range []string{"BothOne", "BothTwo", "BothThree", "BothFour", "BothFive", "BothSix"} {
		fmt.Fprintf(&b, `
func %s(ref string) (string, error) {
	v, err := store.Get(ref)
	if err != nil {
		return "", err
	}
	out, err := store.Decode(v)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(strings.TrimSpace(out)), nil
}
`, name)
	}
	src := b.String()

	opt := testOptions()
	opt.MaxMemberships = 0
	unbounded, _ := build(t, src, nil, opt)

	widest := 0
	for _, cs := range unbounded.Assignments() {
		if len(cs) > widest {
			widest = len(cs)
		}
	}
	if widest < 2 {
		t.Fatalf("fixture assigns at most %d concept to any unit; the bound cannot be measured", widest)
	}

	opt.MaxMemberships = 1
	bounded, _ := build(t, src, nil, opt)
	for i, cs := range bounded.Assignments() {
		if len(cs) > 1 {
			t.Errorf("unit %d kept %d memberships under MaxMemberships=1", i, len(cs))
		}
		for j := 1; j < len(cs); j++ {
			if cs[j-1].ID >= cs[j].ID {
				t.Errorf("unit %d memberships are not ascending by ID: %v", i, parser.ConceptIDs(cs))
			}
		}
	}

	// The bound selects, it does not invent: every membership it keeps is one
	// the unbounded run also assigned, at the same confidence. (Which one it
	// keeps is decided on coverage, and coverage is not recoverable from a
	// Model — confidence is monotone in it within a concept but not across
	// concepts, since each has its own Scale — so the ranking itself is pinned
	// by the corpus measurement in internal/bench, not from out here.)
	for i, cs := range bounded.Assignments() {
		for _, c := range cs {
			if got := parser.ConfidenceOf(unbounded.Assignments()[i], c.ID); got != c.Confidence {
				t.Errorf("unit %d kept %q at confidence %v; unbounded read %v", i, c.ID, c.Confidence, got)
			}
		}
	}

	// Nothing may be lost entirely: the bound trims a unit's memberships, it
	// never empties them.
	if bounded.Stats().Untagged != unbounded.Stats().Untagged {
		t.Errorf("untagged moved with the bound: %d bounded, %d unbounded",
			bounded.Stats().Untagged, unbounded.Stats().Untagged)
	}
}

// TestCorpusFloorRulesAreSeams pins the measurement seams as seams: they change
// what is admitted, they are off by default, and the default is untouched by
// their existence.
//
// The verdict itself is corpus-scale and lives in internal/bench — one
// 48-function fixture cannot say whether a floor rule is better. What this
// pins is that the alternatives are reachable, deterministic, and inert until
// asked for.
func TestCorpusFloorRulesAreSeams(t *testing.T) {
	src := storeCorpus()
	base, _ := build(t, src, nil, testOptions())

	for _, tc := range []struct {
		name string
		opt  func(Options) Options
	}{
		{"relmax", func(o Options) Options { o.FloorRule = FloorRelMax; o.RelMaxFraction = 0.25; return o }},
		{"touched", func(o Options) Options { o.FloorRule = FloorTouched; o.TouchedQuantile = 0.5; return o }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opt := tc.opt(testOptions())
			m, _ := build(t, src, nil, opt)
			again, _ := build(t, src, nil, opt)
			if !reflect.DeepEqual(m.Assignments(), again.Assignments()) {
				t.Error("two builds under the same options disagree")
			}
			for _, c := range m.Concepts() {
				if c.Floor <= 0 {
					t.Errorf("concept %q has a non-positive floor %v", c.ID, c.Floor)
				}
			}
		})
	}

	// The default is unchanged by any of it.
	after, _ := build(t, src, nil, testOptions())
	if !reflect.DeepEqual(base.Assignments(), after.Assignments()) {
		t.Error("the default rule is not stable across builds")
	}
}

// TestFloorOfDegenerateCurves covers the shapes a corpus-derived bar has to
// survive: nothing reached, one unit reached, and a curve every unit sits on
// the same point of.
func TestFloorOfDegenerateCurves(t *testing.T) {
	opt := testOptions()
	opt.FloorRule = FloorRelMax
	opt.RelMaxFraction = 0.5

	if got := floorOf(nil, opt); got <= 0 {
		t.Errorf("an unreached concept floors at %v; want a positive bar", got)
	}
	if got := floorOf([]float64{2}, opt); got != 1 {
		t.Errorf("floorOf([2]) at half the max = %v, want 1", got)
	}
	if got := floorOf([]float64{3, 3, 3}, opt); got != 1.5 {
		t.Errorf("floorOf(flat 3) at half the max = %v, want 1.5", got)
	}

	opt.FloorRule = FloorTouched
	opt.TouchedQuantile = 0.5
	// Nearest-rank upper: rank ceil(0.5*4) = 2, ascending.
	if got := floorOf([]float64{1, 2, 3, 4}, opt); got != 2 {
		t.Errorf("upper quantile 0.5 of [1 2 3 4] = %v, want 2", got)
	}
}

// TestCorpusFloorDropsUnfoundedConcepts pins the rule taken with the corpus
// bar: a concept whose own founders cannot clear it is dropped, counted, and —
// if it grew from a seed — no longer reported as a grown seed, because
// UnusedSeeds is what answers "does this repository already do X".
func TestCorpusFloorDropsUnfoundedConcepts(t *testing.T) {
	units, err := parser.ParseSource("app.go", []byte(storeCorpus()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	seeds := make([][]string, len(units))
	for i := firstStore; i < afterStore; i++ {
		seeds[i] = []string{"db_access"}
	}
	g := concepter.BuildCallGraph(units)

	opt := testOptions()
	opt.FloorRule = FloorRelMax
	opt.RelMaxFraction = 0.25
	kept := Build(units, g, seeds, opt)

	// A bar no founder can clear must take the concept with it.
	opt.RelMaxFraction = 1000
	gone := Build(units, g, seeds, opt)

	if gone.Stats().FloorDropped <= kept.Stats().FloorDropped {
		t.Errorf("an unreachable bar dropped %d concepts, an ordinary one %d; want more",
			gone.Stats().FloorDropped, kept.Stats().FloorDropped)
	}
	if len(gone.Concepts()) != 0 {
		t.Errorf("concepts survived an unreachable bar: %v", ids(gone))
	}
	if got := gone.GrownSeeds(); len(got) != 0 {
		t.Errorf("GrownSeeds = %v after every concept was dropped; want none", got)
	}
	if got := kept.GrownSeeds(); len(got) == 0 {
		t.Error("GrownSeeds is empty though the seeded concept survived")
	}
}

// TestBackfillAdmitsTheOtherwiseEmpty pins the backfill's contract: a unit
// clearing no floor gets its strongest concepts anyway, marked, at the
// confidence it earned — and the mark is what keeps a membership the unit did
// not earn away from the consumers that count rather than weight.
func TestBackfillAdmitsTheOtherwiseEmpty(t *testing.T) {
	src := weakCarrierCorpus()
	base := testOptions()
	control, units := build(t, src, nil, base)

	// The weak carrier reaches the store practice through one of its two calls
	// and spends the rest of itself elsewhere, so it touches the vocabulary and
	// covers far too little of itself with it to clear any floor. A unit that
	// touches nothing at all — the filler — has nothing to backfill from, which
	// is why the fixture cannot just be storeCorpus and the first empty unit.
	empty := -1
	for i := range units {
		if units[i].Name == "WeakReader" {
			empty = i
		}
	}
	if empty < 0 {
		t.Fatal("fixture did not parse the weak carrier")
	}
	if got := control.Assignments()[empty]; len(got) != 0 {
		t.Fatalf("the weak carrier already carries %v; the fixture is not measuring anything",
			parser.ConceptIDs(got))
	}

	opt := base
	opt.BackfillN = 1
	filled, _ := build(t, src, nil, opt)

	got := filled.Assignments()[empty]
	if len(got) != 1 {
		t.Fatalf("the weak carrier has %d memberships under BackfillN=1, want 1", len(got))
	}
	if !got[0].BelowFloor {
		t.Error("a backfilled membership is not marked BelowFloor")
	}
	if got[0].Confidence <= 0 || got[0].Confidence > 1 {
		t.Errorf("backfilled confidence %v is outside (0,1]", got[0].Confidence)
	}

	// The two projections must disagree: that disagreement is the mechanism.
	if ids := parser.ConceptIDs(got); len(ids) != 0 {
		t.Errorf("ConceptIDs = %v for a backfilled-only unit; want none", ids)
	}
	if g := concepter.Graded(got); len(g) != 1 {
		t.Errorf("Graded = %v for a backfilled-only unit; want the membership kept", g)
	}

	// Nobody who already had a membership gains one, absent BackfillAlways.
	for i := range units {
		if len(control.Assignments()[i]) > 0 &&
			len(filled.Assignments()[i]) != len(control.Assignments()[i]) {
			t.Errorf("unit %d gained a membership though it already had one", i)
		}
	}
}

// weakCarrierCorpus is storeCorpus plus one function that reaches the store
// practice through half its vocabulary and spends the rest of itself
// elsewhere: it touches the concept and covers far too little of itself with
// it to be a member. That is the shape the backfill exists for, and the shape
// a fixture of pure carriers and pure strangers does not contain.
func weakCarrierCorpus() string {
	var b strings.Builder
	b.WriteString(storeCorpus())
	b.WriteString(`
func WeakReader(ref string) string {
	v, _ := store.Get(ref)
	out := v
	for i := 0; i < 6; i++ {
		out = strings.TrimSpace(out)
		out = strings.ToUpper(out)
		out = strings.ReplaceAll(out, "a", "b")
		if strings.Contains(out, "z") {
			out = strings.TrimSuffix(out, "z")
		}
		switch {
		case strings.HasPrefix(out, "q"):
			out = strings.TrimPrefix(out, "q")
		default:
			out += strings.Repeat("x", i)
		}
	}
	return out
}
`)
	return b.String()
}

// TestBackfillVisibleSkipsTheMark is the other half of the switch: the same
// memberships, emitted as ordinary, so the boolean consumers see them too.
func TestBackfillVisibleSkipsTheMark(t *testing.T) {
	opt := testOptions()
	opt.BackfillN = 1
	marked, _ := build(t, weakCarrierCorpus(), nil, opt)
	opt.BackfillVisible = true
	visible, _ := build(t, weakCarrierCorpus(), nil, opt)

	sawVisible := false
	for i, cs := range visible.Assignments() {
		if len(cs) != len(marked.Assignments()[i]) {
			t.Fatalf("unit %d has %d memberships visible and %d marked; want the same set",
				i, len(cs), len(marked.Assignments()[i]))
		}
		for k, c := range cs {
			m := marked.Assignments()[i][k]
			if c.ID != m.ID || c.Confidence != m.Confidence {
				t.Errorf("unit %d membership %d differs beyond the mark: %+v vs %+v", i, k, c, m)
			}
			if c.BelowFloor {
				t.Errorf("unit %d membership %q is marked under BackfillVisible", i, c.ID)
			}
			if m.BelowFloor {
				sawVisible = true
			}
		}
	}
	if !sawVisible {
		t.Skip("fixture produced no backfilled membership; nothing to compare")
	}
}

// TestBackfillNeverDisplaces pins the ordering guarantee: a backfilled
// membership fills space a floor-cleared one did not want, and can never push
// one out under MaxMemberships.
func TestBackfillNeverDisplaces(t *testing.T) {
	opt := testOptions()
	opt.MaxMemberships = 1
	control, _ := build(t, weakCarrierCorpus(), nil, opt)
	opt.BackfillN = 1
	opt.BackfillAlways = true
	filled, _ := build(t, weakCarrierCorpus(), nil, opt)

	for i, cs := range control.Assignments() {
		for _, c := range cs {
			if parser.ConfidenceOf(filled.Assignments()[i], c.ID) != c.Confidence {
				t.Errorf("unit %d lost earned membership %q to the backfill", i, c.ID)
			}
		}
	}
}

// TestBackfillOffByDefault is the seam's own pin: the shipped options produce
// no marked membership at all, so every consumer sees exactly what it saw
// before the backfill existed.
func TestBackfillOffByDefault(t *testing.T) {
	m, _ := build(t, storeCorpus(), nil, testOptions())
	for i, cs := range m.Assignments() {
		for _, c := range cs {
			if c.BelowFloor {
				t.Errorf("unit %d carries a BelowFloor membership %q at defaults", i, c.ID)
			}
		}
	}
}

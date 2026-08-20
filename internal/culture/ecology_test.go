package culture

import (
	"math"
	"testing"

	"github.com/lukse/doppel/internal/concepter"
	"github.com/lukse/doppel/internal/parser"
)

// unit builds a minimal hand-constructed CodeUnit. Culture only reads
// Patterns, Fingerprint.Flow, Package, and (via CallTokens) Callees/Signals.
func unit(name, pkg string, tags ...string) parser.CodeUnit {
	return parser.CodeUnit{Name: name, Package: pkg, Patterns: tags}
}

// docsWithRole builds n concept docs all carrying the same role.
func docsWithRole(n int, role string) []concepter.ConceptDoc {
	docs := make([]concepter.ConceptDoc, n)
	for i := range docs {
		docs[i].Role = role
	}
	return docs
}

func buildOn(t *testing.T, units []parser.CodeUnit, docs []concepter.ConceptDoc, opt Options) *Model {
	t.Helper()
	if len(units) != len(docs) {
		t.Fatalf("fixture broken: %d units, %d docs", len(units), len(docs))
	}
	return Build(units, docs, concepter.BuildCallGraph(units), opt)
}

func findAssoc(assocs []Association, kind AssocKind, a, b string) (Association, bool) {
	for _, as := range assocs {
		if as.Kind == kind && as.A == a && as.B == b {
			return as, true
		}
	}
	return Association{}, false
}

// Hand-computed positive PMI pins. 8 units, db_access on 4, retry on 4:
// together on 3 → PMI = ln(8·3/16) = ln 1.5, below the ln 2 floor → silent;
// together on all 4 → PMI = ln 2 exactly → reported.
func TestAssociationPositiveFloor(t *testing.T) {
	makeCorpus := func(both int) []parser.CodeUnit {
		var units []parser.CodeUnit
		for i := 0; i < both; i++ {
			units = append(units, unit(name("ab", i), "p", "db_access", "retry"))
		}
		for i := both; i < 4; i++ {
			units = append(units, unit(name("a", i), "p", "db_access"))
			units = append(units, unit(name("b", i), "p", "retry"))
		}
		for len(units) < 8 {
			units = append(units, unit(name("x", len(units)), "p"))
		}
		return units
	}

	units := makeCorpus(3)
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())
	if as, ok := findAssoc(m.Associations(), TagTag, "db_access", "retry"); ok {
		t.Errorf("PMI ln1.5 pair reported (%+v); the ln2 floor should silence it", as)
	}

	units = makeCorpus(4)
	m = buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())
	as, ok := findAssoc(m.Associations(), TagTag, "db_access", "retry")
	if !ok {
		t.Fatal("PMI ln2 pair not reported")
	}
	if math.Abs(as.PMI-math.Ln2) > 1e-12 || as.Count != 4 {
		t.Errorf("association = %+v, want Count 4, PMI exactly ln2", as)
	}
}

func name(prefix string, i int) string {
	return prefix + string(rune('A'+i))
}

// High PMI without support stays silent: 2 co-occurrences in 100 units is an
// anecdote, not an association.
func TestAssociationSupportCutoff(t *testing.T) {
	var units []parser.CodeUnit
	units = append(units,
		unit("abA", "p", "caching", "transaction"),
		unit("abB", "p", "caching", "transaction"),
	)
	for i := 0; i < 98; i++ {
		units = append(units, unit(name("f", i%26)+name("g", i/26), "p"))
	}
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())
	if as, ok := findAssoc(m.Associations(), TagTag, "caching", "transaction"); ok {
		t.Errorf("count-2 pair reported (%+v) despite MinPairSupport", as)
	}
}

// Negative associations: informative absence needs big marginals. 12 units,
// two 6-unit tags that never meet → Expected 3.0, reported with PMI == -Inf;
// two 3-unit tags that never meet → Expected 0.75, silent.
func TestAssociationNegatives(t *testing.T) {
	var units []parser.CodeUnit
	for i := 0; i < 6; i++ {
		units = append(units, unit(name("a", i), "p", "validation"))
		units = append(units, unit(name("b", i), "p", "transaction"))
	}
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())
	as, ok := findAssoc(m.Associations(), TagTag, "transaction", "validation")
	if !ok {
		t.Fatal("never-co-occurring big pair not reported as negative")
	}
	if !math.IsInf(as.PMI, -1) || as.Count != 0 || math.Abs(as.Expected-3.0) > 1e-12 {
		t.Errorf("association = %+v, want Count 0, Expected 3.0, PMI -Inf", as)
	}

	units = nil
	for i := 0; i < 3; i++ {
		units = append(units, unit(name("a", i), "p", "validation"))
		units = append(units, unit(name("b", i), "p", "transaction"))
	}
	for i := 0; i < 6; i++ {
		units = append(units, unit(name("x", i), "p"))
	}
	m = buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())
	if as, ok := findAssoc(m.Associations(), TagTag, "transaction", "validation"); ok {
		t.Errorf("sparse zero reported (%+v); Expected 0.75 is below the floor", as)
	}
}

// tag~role counting: 4 orchestrator transaction units among 10 units, roles
// split 4/6 → PMI = ln(10·4/(4·4)) = ln 2.5, reported.
func TestAssociationTagRole(t *testing.T) {
	var units []parser.CodeUnit
	docs := make([]concepter.ConceptDoc, 0, 10)
	for i := 0; i < 4; i++ {
		units = append(units, unit(name("t", i), "p", "transaction"))
		docs = append(docs, concepter.ConceptDoc{Role: "orchestrator"})
	}
	for i := 0; i < 6; i++ {
		units = append(units, unit(name("x", i), "p"))
		docs = append(docs, concepter.ConceptDoc{Role: "leaf"})
	}
	m := buildOn(t, units, docs, DefaultOptions())
	as, ok := findAssoc(m.Associations(), TagRole, "transaction", "orchestrator")
	if !ok {
		t.Fatal("tag~role association not reported")
	}
	if want := math.Log(2.5); math.Abs(as.PMI-want) > 1e-12 {
		t.Errorf("PMI = %v, want ln2.5 = %v", as.PMI, want)
	}
}

// tag~call counting uses resolved call tokens, and the df cap excludes
// corpus-idiom tokens from association material.
func TestAssociationTagCallAndDFCap(t *testing.T) {
	sqlCaller := func(nm string, tags ...string) parser.CodeUnit {
		u := unit(nm, "p", tags...)
		u.Callees = []string{"sql.Open"}
		u.Signals = parser.TagSignals{PackageRefs: []parser.PackageRef{{Local: "sql", Path: "database/sql"}}}
		return u
	}
	var units []parser.CodeUnit
	for i := 0; i < 4; i++ {
		units = append(units, sqlCaller(name("d", i), "db_access"))
	}
	for i := 0; i < 4; i++ {
		units = append(units, unit(name("x", i), "p"))
	}
	// c(tag)=4, c(token)=4, together 4, N=8 → PMI = ln(8·4/16) = ln2.
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())
	as, ok := findAssoc(m.Associations(), TagCall, "db_access", "database/sql.Open")
	if !ok {
		t.Fatal("tag~call association not reported")
	}
	if math.Abs(as.PMI-math.Ln2) > 1e-12 {
		t.Errorf("PMI = %v, want ln2", as.PMI)
	}

	opt := DefaultOptions()
	opt.MaxCallTokenDF = 3 // token df=4 exceeds the cap → corpus idiom
	m = buildOn(t, units, docsWithRole(len(units), "leaf"), opt)
	if _, ok := findAssoc(m.Associations(), TagCall, "db_access", "database/sql.Open"); ok {
		t.Error("df-capped token still produced an association")
	}
}

// Equal-PMI ties order by (Kind, A, B): two disjoint tag pairs with the same
// count structure must come out in name order.
func TestAssociationTieOrdering(t *testing.T) {
	var units []parser.CodeUnit
	for i := 0; i < 4; i++ {
		units = append(units, unit(name("m", i), "p", "mapping", "validation"))
		units = append(units, unit(name("c", i), "p", "caching", "retry"))
	}
	m := buildOn(t, units, docsWithRole(len(units), "leaf"), DefaultOptions())
	var tagTag []Association
	for _, as := range m.Associations() {
		if as.Kind == TagTag {
			tagTag = append(tagTag, as)
		}
	}
	if len(tagTag) < 2 {
		t.Fatalf("expected at least the two positive tag pairs, got %+v", tagTag)
	}
	if tagTag[0].PMI != tagTag[1].PMI {
		t.Fatalf("fixture broken: tie expected, got PMI %v vs %v", tagTag[0].PMI, tagTag[1].PMI)
	}
	if tagTag[0].A != "caching" || tagTag[1].A != "mapping" {
		t.Errorf("tie order = (%s,%s) then (%s,%s), want caching/retry before mapping/validation",
			tagTag[0].A, tagTag[0].B, tagTag[1].A, tagTag[1].B)
	}
}

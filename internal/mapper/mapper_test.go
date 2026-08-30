package mapper

import (
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/tagger"
)

// buildCorpus parses files in lexical path order, tags every unit, and returns
// units + graph + docs — the same shape cmd/analyze.go assembles.
func buildCorpus(t *testing.T, files map[string]string) ([]parser.CodeUnit, []concepter.ConceptDoc) {
	t.Helper()
	var paths []string
	for p := range files {
		paths = append(paths, p)
	}
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			if paths[j] < paths[i] {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}
	var units []parser.CodeUnit
	for _, p := range paths {
		parsed, err := parser.ParseSource(p, []byte(files[p]))
		if err != nil {
			t.Fatalf("ParseSource(%s): %v", p, err)
		}
		units = append(units, parsed...)
	}
	for i := range units {
		units[i].Concepts = parser.Certain(tagger.Tag(units[i])...)
	}
	g := concepter.BuildCallGraph(units)
	docs := Map(units, g, concepter.New())
	return units, docs
}

func docByName(t *testing.T, units []parser.CodeUnit, docs []concepter.ConceptDoc, qualified string) concepter.ConceptDoc {
	t.Helper()
	for i := range units {
		if concepter.QualifiedName(units[i]) == qualified {
			return docs[i]
		}
	}
	t.Fatalf("no unit %q in corpus", qualified)
	return concepter.ConceptDoc{}
}

func TestMapCallersAreQualified(t *testing.T) {
	units, docs := buildCorpus(t, map[string]string{
		"a/a.go": `package alpha
import "example.com/mod/beta"
func caller() { beta.Work() }`,
		"b/b.go": `package beta
func Work() {}`,
	})
	work := docByName(t, units, docs, "beta.Work")
	if len(work.Callers) != 1 || work.Callers[0] != "alpha.caller" {
		t.Errorf("Callers = %v, want [alpha.caller]", work.Callers)
	}
	if len(work.CallerPackages) != 1 || work.CallerPackages[0] != "alpha" {
		t.Errorf("CallerPackages = %v, want [alpha]", work.CallerPackages)
	}
}

// The old index was keyed by bare name: two functions named New in different
// packages overwrote each other, so context aggregation could report the wrong
// package's tags. Qualified keys end that.
func TestMapIndexCollisionFixed(t *testing.T) {
	units, docs := buildCorpus(t, map[string]string{
		"a/a.go": `package alpha
func New() {
	for i := 0; i < maxRetries; i++ {
	}
}
var maxRetries = 3`,
		"b/b.go": `package beta
func New() {
	tx := begin()
	tx.Commit()
}
func begin() txn { return txn{} }
type txn struct{}
func (t txn) Commit() {}`,
		"c/c.go": `package gamma
import (
	"example.com/mod/alpha"
	"example.com/mod/beta"
)
func useBoth() {
	alpha.New()
	beta.New()
}`,
	})

	caller := docByName(t, units, docs, "gamma.useBoth")
	// Callee patterns must aggregate BOTH News' tags: alpha.New is retry,
	// beta.New is transaction. Under the bare-name index one clobbered the
	// other.
	wantTags := map[string]bool{"retry": true, "transaction": true}
	if len(caller.CalleeConcepts) != len(wantTags) {
		t.Fatalf("CalleeConcepts = %v, want retry+transaction", caller.CalleeConcepts)
	}
	for _, p := range caller.CalleeConcepts {
		if !wantTags[p.ID] {
			t.Errorf("unexpected callee concept %q", p.ID)
		}
	}
	wantPkgs := []string{"alpha", "beta"}
	if strings.Join(caller.CalleePackages, ",") != strings.Join(wantPkgs, ",") {
		t.Errorf("CalleePackages = %v, want %v", caller.CalleePackages, wantPkgs)
	}
}

// Cross-package callee context works for the first time: the old lookup hit
// only bare same-package names, so a cross-package callee's patterns and
// package were silently dropped.
func TestMapResolvedCalleesFeedContext(t *testing.T) {
	units, docs := buildCorpus(t, map[string]string{
		"a/a.go": `package alpha
import "example.com/mod/store"
func handler() { store.Save() }`,
		"s/s.go": `package store
import "database/sql"
func Save() {
	var db *sql.DB
	_ = db
}`,
	})
	h := docByName(t, units, docs, "alpha.handler")
	if len(h.ResolvedCallees) != 1 || h.ResolvedCallees[0] != "store.Save" {
		t.Fatalf("ResolvedCallees = %v, want [store.Save]", h.ResolvedCallees)
	}
	if strings.Join(parser.ConceptIDs(h.CalleeConcepts), ",") != "db_access" {
		t.Errorf("CalleeConcepts = %v, want [db_access]", h.CalleeConcepts)
	}
	if strings.Join(h.CalleePackages, ",") != "store" {
		t.Errorf("CalleePackages = %v, want [store]", h.CalleePackages)
	}
}

// Roles now measure resolved internal fan-out, not raw call expressions: a
// helper calling only builtins and stdlib is a leaf, no matter how many call
// expressions its body holds.
func TestMapRoleUsesResolvedEdges(t *testing.T) {
	units, docs := buildCorpus(t, map[string]string{
		"a/a.go": `package alpha
import (
	"fmt"
	"sort"
)
func stdlibOnly(xs []string) {
	sort.Strings(xs)
	fmt.Println(len(xs), append(xs, "x"))
}
func orchestrates() {
	one()
	two()
}
func one() {}
func two() {}`,
	})
	if got := docByName(t, units, docs, "alpha.stdlibOnly").Role; got != concepter.RoleLeaf {
		t.Errorf("stdlib-only helper role = %q, want leaf", got)
	}
	if got := docByName(t, units, docs, "alpha.orchestrates").Role; got != concepter.RoleOrchestrator {
		t.Errorf("two-internal-callee function role = %q, want orchestrator", got)
	}
}

// On a dense corpus the adaptive threshold rises above the floor: a unit with
// 2 callers is a "utility" under the historical fixed threshold but ordinary —
// leaf — when over half the corpus has 2 or more callers.
func TestMapAdaptiveThresholds(t *testing.T) {
	// Degrees: t1,t3,t4,t5 have 2 callers, t2 has 3, the callers have 0.
	// Sorted fan-in [0,0,0,2,2,2,2,3] has upper median 2, so the fan-in
	// threshold rises to 3: two callers is ordinary here.
	units, docs := buildCorpus(t, map[string]string{
		"a/a.go": `package alpha
func t1() {}
func t2() {}
func t3() {}
func t4() {}
func t5() {}
func c1() { t1(); t2(); t3(); t4(); t5() }
func c2() { t1(); t2(); t3(); t4(); t5() }
func c3() { t2() }`,
	})
	// Sanity: t1 has exactly 2 callers, which would be "high" at the floor.
	t1 := docByName(t, units, docs, "alpha.t1")
	if len(t1.Callers) != 2 {
		t.Fatalf("fixture broken: t1 has %d callers, want 2", len(t1.Callers))
	}
	if t1.Role != concepter.RoleLeaf {
		t.Errorf("t1 role = %q, want leaf: 2 callers is ordinary in this corpus", t1.Role)
	}
	t2 := docByName(t, units, docs, "alpha.t2")
	if t2.Role != concepter.RoleUtility {
		t.Errorf("t2 role = %q, want utility: 3 callers clears the raised threshold", t2.Role)
	}
}

func TestMapDeterministic(t *testing.T) {
	files := map[string]string{
		"a/a.go": `package alpha
import "example.com/mod/beta"
func f() { beta.Work(); g() }
func g() {}`,
		"b/b.go": `package beta
func Work() {}`,
	}
	render := func(docs []concepter.ConceptDoc) string {
		var sb strings.Builder
		for _, d := range docs {
			sb.WriteString(d.Package + "." + d.Name + "|" + d.Role + "|")
			sb.WriteString(strings.Join(d.Callers, ",") + "|")
			sb.WriteString(strings.Join(d.ResolvedCallees, ",") + "|")
			sb.WriteString(strings.Join(d.CalleePackages, ",") + "\n")
		}
		return sb.String()
	}
	_, first := buildCorpus(t, files)
	want := render(first)
	for i := 0; i < 100; i++ {
		_, docs := buildCorpus(t, files)
		if got := render(docs); got != want {
			t.Fatalf("run %d diverged:\n%s\nwant:\n%s", i, got, want)
		}
	}
}

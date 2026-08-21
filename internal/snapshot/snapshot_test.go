package snapshot

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

func unit(pkg, name, file string, line, nodes int, patterns ...string) parser.CodeUnit {
	return parser.CodeUnit{
		Name: name, Package: pkg, File: file, StartLine: line,
		Patterns: patterns,
		Fingerprint: fingerprint.Fingerprint{
			Shingles: []uint64{uint64(nodes), uint64(nodes) * 7},
			Flow:     []int{nodes % 3, 1},
			Types:    []string{"int"},
			Nodes:    nodes,
		},
	}
}

func docFor(role string, callers, callees int) concepter.ConceptDoc {
	d := concepter.ConceptDoc{Role: role}
	for i := 0; i < callers; i++ {
		d.Callers = append(d.Callers, "x.Caller")
	}
	for i := 0; i < callees; i++ {
		d.ResolvedCallees = append(d.ResolvedCallees, "x.Callee")
	}
	return d
}

func sampleInputs() ([]parser.CodeUnit, []concepter.ConceptDoc, []analyzer.SimilarPair, map[ontology.TermID]int) {
	units := []parser.CodeUnit{
		unit("beta", "Second", "beta/b.go", 10, 30, "mapping"),
		unit("alpha", "First", "alpha/a.go", 4, 20, "db_access", "retry"),
		unit("alpha", "Third", "alpha/c.go", 7, 25),
	}
	docs := []concepter.ConceptDoc{
		docFor("utility", 3, 0),
		docFor("leaf", 0, 1),
		docFor("orchestrator", 1, 4),
	}
	ev := comparator.StructuralEvidence{OverlapScore: 0.5, MergeWorthy: true, Reasons: []string{"both <read> & write"}}
	pairs := []analyzer.SimilarPair{{
		A: units[0], B: units[1], AIdx: 0, BIdx: 1, Score: 0.8,
		Breakdown: fingerprint.Breakdown{AST: 0.7, Flow: 0.9, Signature: 1, SizeRatio: 0.66, Score: 0.8},
		Evidence:  &ev,
	}}
	counts := map[ontology.TermID]int{"mapping": 1, "db_access": 1, "retry": 1}
	return units, docs, pairs, counts
}

func buildSample() Snapshot {
	u, d, p, c := sampleInputs()
	return Build(u, d, p, c, "", "test", Params{Threshold: 0.6, MinNodes: 12, TestsMode: "exclude"})
}

// TestBuildIsDeterministic is the invariant the whole tool rests on: an
// unchanged tree must produce byte-identical output. Go randomises map
// iteration per range statement, so repeating the build many times is what
// actually catches an unsorted map ranging into the schema.
func TestBuildIsDeterministic(t *testing.T) {
	want, err := json.Marshal(buildSample())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for i := 0; i < 50; i++ {
		got, err := json.Marshal(buildSample())
		if err != nil {
			t.Fatalf("marshal %d: %v", i, err)
		}
		if !bytes.Equal(want, got) {
			t.Fatalf("build %d differs from build 0:\n first: %s\n  this: %s", i, want, got)
		}
	}
}

// TestSchemaHasNoMaps keeps the determinism guarantee visible in the type
// rather than relying on encoding/json's key sorting. A map field would
// serialise deterministically but would still let Go code that reads a
// Snapshot iterate it in random order.
func TestSchemaHasNoMaps(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var walk func(t reflect.Type, path string)
	walk = func(rt reflect.Type, path string) {
		if seen[rt] {
			return
		}
		seen[rt] = true
		switch rt.Kind() {
		case reflect.Map:
			t.Errorf("%s is a map; the snapshot schema must use sorted slices", path)
		case reflect.Slice, reflect.Ptr, reflect.Array:
			walk(rt.Elem(), path+"[]")
		case reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				walk(f.Type, path+"."+f.Name)
			}
		}
	}
	walk(reflect.TypeOf(Snapshot{}), "Snapshot")
	walk(reflect.TypeOf(Delta{}), "Delta")
}

func TestBuildSortsAndCounts(t *testing.T) {
	s := buildSample()

	if s.Schema != Schema || s.Ontology != ontology.Version || s.Doppel != "test" {
		t.Errorf("identity fields wrong: %+v", s)
	}
	if s.Functions != 3 {
		t.Errorf("Functions = %d, want 3", s.Functions)
	}

	wantUnits := []string{"alpha.First", "alpha.Third", "beta.Second"}
	for i, u := range s.Units {
		if u.Key != wantUnits[i] {
			t.Errorf("Units[%d].Key = %q, want %q", i, u.Key, wantUnits[i])
		}
	}
	wantTags := []string{"db_access", "mapping", "retry"}
	for i, c := range s.Concepts {
		if c.Tag != wantTags[i] {
			t.Errorf("Concepts[%d].Tag = %q, want %q", i, c.Tag, wantTags[i])
		}
	}
	wantRoles := []string{"leaf", "orchestrator", "utility"}
	for i, r := range s.Roles {
		if r.Role != wantRoles[i] {
			t.Errorf("Roles[%d].Role = %q, want %q", i, r.Role, wantRoles[i])
		}
	}
	if s.MergeWorthy() != 1 {
		t.Errorf("MergeWorthy() = %d, want 1", s.MergeWorthy())
	}
}

// TestPairSidesAreNameOrdered pins that a pair has one spelling. FindSimilar
// emits AIdx < BIdx, which is file-walk order and shifts when a file is added;
// only name order lets the same pair be recognised across two runs.
func TestPairSidesAreNameOrdered(t *testing.T) {
	s := buildSample()
	if len(s.Pairs) != 1 {
		t.Fatalf("got %d pairs, want 1", len(s.Pairs))
	}
	p := s.Pairs[0]
	// units[0] is beta.Second and units[1] is alpha.First, so the walk order
	// and the name order genuinely disagree here.
	if p.A != "alpha.First" || p.B != "beta.Second" {
		t.Errorf("pair = %s <-> %s, want alpha.First <-> beta.Second", p.A, p.B)
	}
}

// TestUnitKeysDisambiguateCollisions covers init, which may be declared more
// than once per package, and two directories sharing a package clause. Without
// disambiguation those units share a key and a diff silently conflates them.
func TestUnitKeysDisambiguateCollisions(t *testing.T) {
	units := []parser.CodeUnit{
		unit("app", "init", "app/a.go", 1, 20),
		unit("app", "init", "app/b.go", 1, 20),
		unit("app", "Unique", "app/a.go", 9, 20),
	}
	keys := unitKeys(units, "")
	if keys[0] == keys[1] {
		t.Errorf("colliding init units share key %q", keys[0])
	}
	if !strings.HasPrefix(keys[0], "app.init@") || !strings.Contains(keys[0], "app/a.go") {
		t.Errorf("keys[0] = %q, want app.init@app/a.go", keys[0])
	}
	if keys[2] != "app.Unique" {
		t.Errorf("keys[2] = %q, want the bare qualified name when there is no collision", keys[2])
	}
}

// TestPathsAreRelativeAndSlashed pins the property that lets two runs be
// compared at all: a hook analyses an absolute cwd while `doppel analyze .`
// analyses a relative root, and both must describe the same file the same way.
//
// The paths are built with filepath.Join rather than written as literals so the
// test exercises whatever separator the host actually uses. Hard-coding
// backslashes only tests Windows, and on Linux it does not even fail
// meaningfully: `\` is an ordinary character there, so filepath.Rel cannot
// relativize the path and returns a `../` escape instead.
func TestPathsAreRelativeAndSlashed(t *testing.T) {
	root := filepath.Join(string(filepath.Separator)+"repo", "project")
	file := filepath.Join(root, "internal", "pkg", "a.go")
	units := []parser.CodeUnit{unit("app", "F", file, 1, 20)}

	s := Build(units, []concepter.ConceptDoc{{}}, nil, nil, root, "test", Params{})

	got := s.Units[0].File
	if want := "internal/pkg/a.go"; got != want {
		t.Fatalf("File = %q, want %q", got, want)
	}
	if strings.ContainsRune(got, filepath.Separator) && filepath.Separator != '/' {
		t.Errorf("File %q still carries a native separator", got)
	}
	if filepath.IsAbs(got) || strings.HasPrefix(got, "..") {
		t.Errorf("File %q is not contained within the analysis root", got)
	}
}

// A path the root cannot explain must still be recorded deterministically
// rather than dropped, and must still be slash-separated.
func TestPathsOutsideRootStaySlashed(t *testing.T) {
	units := []parser.CodeUnit{unit("app", "F", filepath.Join("elsewhere", "a.go"), 1, 20)}

	s := Build(units, []concepter.ConceptDoc{{}}, nil, nil, "", "test", Params{})

	if got := s.Units[0].File; got != "elsewhere/a.go" {
		t.Errorf("File = %q, want elsewhere/a.go", got)
	}
}

func TestDigestDetectsBodyChange(t *testing.T) {
	a := unit("p", "F", "p/a.go", 1, 20).Fingerprint
	b := unit("p", "F", "p/a.go", 1, 21).Fingerprint

	if Digest(a) == "" {
		t.Fatal("digest of a real fingerprint must not be empty")
	}
	if Digest(a) != Digest(a) {
		t.Error("digest is not stable")
	}
	if Digest(a) == Digest(b) {
		t.Error("digest did not change when the fingerprint did")
	}
	// A declaration with no body must not digest-match another one: the zero
	// fingerprint never matches anything, and the digest follows that rule.
	if Digest(fingerprint.Fingerprint{}) != "" {
		t.Error("zero fingerprint must digest to the empty string")
	}
}

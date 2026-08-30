package identity

import (
	"bytes"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// repoRoot resolves doppel's own module root from this file's location, so
// the determinism test runs on a real corpus of the size the command will
// actually be pointed at rather than on a handful of fixture bodies.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/identity/determinism_test.go -> two levels up.
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

// realCorpus walks doppel's own tree exactly as the pipeline does —
// parser.ShouldSkipDir for the directory rule, parser.Parse per .go file —
// and builds a snapshot from it.
func realCorpus(t *testing.T, root string) snapshot.Snapshot {
	t.Helper()
	var units []parser.CodeUnit
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && parser.ShouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		us, err := parser.Parse(path)
		if err != nil {
			return nil // a parse failure is skipped here exactly as the pipeline skips it
		}
		units = append(units, us...)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if len(units) < 200 {
		t.Fatalf("only %d functions found under %s; the determinism test needs a real corpus", len(units), root)
	}
	docs := make([]concepter.ConceptDoc, len(units))
	return snapshot.Build(units, docs, nil, map[ontology.TermID]int{}, nil, root, "test",
		snapshot.Params{Threshold: 0.6, MinNodes: 12, TestsMode: "include"}, snapshot.CorpusMetrics{})
}

// mutate produces the "after" snapshot: the same corpus with a rename, a move,
// a rename-and-move, and a handful of deletions.
//
// It edits the Unit records rather than the source, which is exactly right for
// what it is testing. A rename does not touch a body, so the WL bag and the
// digest are genuinely unchanged and the matcher has to find the pair the way
// it would on real renamed code; a deletion is a unit that is simply not
// there. Nothing here fabricates a bag.
func mutate(s snapshot.Snapshot) snapshot.Snapshot {
	out := s
	out.Units = make([]snapshot.Unit, 0, len(s.Units))
	for i, u := range s.Units {
		switch {
		case i%37 == 0: // deleted
			continue
		case i%23 == 0: // renamed
			u.Name += "V2"
			u.Key = u.Package + "." + u.Name
		case i%19 == 0: // moved
			u.Package = "relocated"
			u.Key = "relocated." + u.Name
		case i%29 == 0: // moved and renamed
			u.Package = "relocated"
			u.Name += "Ex"
			u.Key = "relocated." + u.Name
		}
		out.Units = append(out.Units, u)
	}
	// Build sorts by key and Compare's key index assumes it.
	sortUnitsByKey(out.Units)
	out.Functions = len(out.Units)
	// Pairs reference keys that may have moved. Nothing in this package reads
	// Pairs, but leaving stale ones behind would be a trap for the next reader.
	out.Pairs = nil
	return out
}

func sortUnitsByKey(us []snapshot.Unit) {
	for i := range us {
		for j := i + 1; j < len(us); j++ {
			if us[j].Key < us[i].Key {
				us[i], us[j] = us[j], us[i]
			}
		}
	}
}

// TestDeterminismOnRealCorpus is the invariant the whole tool rests on,
// applied to this command: the same two snapshots must render byte-identically
// every time.
//
// Go randomises map iteration per range statement, so repeating the comparison
// is what actually catches an unsorted map reaching an order — and this
// package builds four of them (the label index, the candidate set, the digest
// buckets, the split and merge candidate lists). Only repetition finds that.
func TestDeterminismOnRealCorpus(t *testing.T) {
	root := repoRoot(t)
	base := realCorpus(t, root)
	head := mutate(base)

	run := func() string {
		r, err := Compare(base, head, Options{})
		if err != nil {
			t.Fatalf("Compare: %v", err)
		}
		if !r.Comparable {
			t.Fatalf("refused: %s", r.Reason)
		}
		var b bytes.Buffer
		Print(&b, r, true)
		if err := WriteJSON(&b, r); err != nil {
			t.Fatalf("WriteJSON: %v", err)
		}
		return b.String()
	}

	want := run()
	for i := 1; i < 12; i++ {
		if got := run(); got != want {
			t.Fatalf("run %d differs from run 0; first divergence at %s", i, firstDiff(want, got))
		}
	}

	// A comparison that found nothing would pass the byte-identity check
	// vacuously, so assert the mutation actually landed in several classes.
	r, _ := Compare(base, head, Options{})
	for _, c := range []Class{Renamed, Moved, Deleted, Unchanged} {
		if r.Count(c) == 0 {
			t.Errorf("no %s findings; the mutation was meant to produce some\n%s", c, countLine(r))
		}
	}
	t.Logf("%d -> %d functions: %s", r.OldFunctions, r.NewFunctions, countLine(r))
}

func firstDiff(a, b string) string {
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(la) && i < len(lb); i++ {
		if la[i] != lb[i] {
			return "line " + itoa(i+1) + ":\n  first: " + la[i] + "\n  this:  " + lb[i]
		}
	}
	return "line " + itoa(min(len(la), len(lb))+1) + " (one output is longer)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

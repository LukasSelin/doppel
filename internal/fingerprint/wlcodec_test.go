package fingerprint

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/LukasSelin/doppel/internal/gofront"
)

// TestEncodeDecodeWLBagSynthetic covers the shapes real bags cannot easily
// exercise on their own: nil/empty, a single entry, adjacent labels (the
// delta-encoding's zero-width case is delta==0 only for a first entry whose
// label is 0 — every later entry has delta >= 1 because WLBag never repeats
// a label), and the uint64 range's extremes, which is where a varint
// implementation most often gets the byte count wrong.
func TestEncodeDecodeWLBagSynthetic(t *testing.T) {
	cases := [][]LabelCount{
		nil,
		{},
		{{Label: 0, Count: 1}},
		{{Label: 1, Count: 1}, {Label: 2, Count: 1}, {Label: 3, Count: 1}},
		{{Label: 0, Count: 1}, {Label: 1, Count: 1000000}},
		{{Label: math.MaxUint64, Count: 1}},
		{{Label: 0, Count: 1}, {Label: math.MaxUint64, Count: 1}},
		{{Label: 5, Count: 1}, {Label: 5 + 1<<32, Count: 1}, {Label: math.MaxUint64, Count: 1}},
	}
	for i, bag := range cases {
		roundTripBag(t, bag, i)
	}
}

// roundTripBag encodes, decodes, and asserts the result is exactly the input
// bag — same labels, same counts, same order.
func roundTripBag(t *testing.T, bag []LabelCount, id int) {
	t.Helper()
	encoded := EncodeWLBag(bag)
	if len(bag) == 0 && encoded != "" {
		t.Errorf("case %d: empty bag encoded to %q, want \"\"", id, encoded)
	}
	decoded, err := DecodeWLBag(encoded)
	if err != nil {
		t.Fatalf("case %d: decode: %v", id, err)
	}
	// The codec persists Label and Count only: H and Kind are in-memory
	// display meta no snapshot consumer reads back, so a decoded bag
	// carries their zero values.
	want := make([]LabelCount, 0, len(bag))
	for _, lc := range bag {
		want = append(want, LabelCount{Label: lc.Label, Count: lc.Count})
	}
	if len(want) == 0 {
		want = nil // DecodeWLBag's empty-bag convention is nil, not [].
	}
	if !slices.Equal(want, decoded) {
		t.Errorf("case %d: round trip mismatch:\n  in  = %+v\n  out = %+v", id, want, decoded)
	}
}

// TestDecodeWLBagRejectsGarbage: a corrupted or foreign string must return an
// error, never a silently wrong bag or a panic. Both are reachable from a
// hand-edited or truncated snapshot file.
func TestDecodeWLBagRejectsGarbage(t *testing.T) {
	cases := []string{
		"not-base64!!!",
		"====",
		// Valid base64 (a single 0xFF byte), but 0xFF alone is a truncated
		// varint: the continuation bit is set with nothing to continue into.
		"/w",
	}
	for _, s := range cases {
		if _, err := DecodeWLBag(s); err == nil {
			t.Errorf("DecodeWLBag(%q) returned no error, want one", s)
		}
	}
}

// TestEncodeDecodeWLBagOverRealCorpus is T6's gate: every function body in
// doppel's own tree, canonicalized exactly as the parser canonicalizes it,
// yields a bag that survives EncodeWLBag/DecodeWLBag unchanged. "Exhaustive"
// means every unit, not a sample — a codec bug that only shows up on one
// corpus body in a thousand is exactly the kind a hand-picked fixture set
// would miss.
func TestEncodeDecodeWLBagOverRealCorpus(t *testing.T) {
	root := repoRoot(t)
	checked := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (name == "vendor" || name == "testdata" ||
				(len(name) > 0 && (name[0] == '.' || name[0] == '_'))) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		// Through the real frontend, which is what canonicalizes now: the
		// bag under test is the one production would build for this body,
		// not a reconstruction of it.
		f, perr := gofront.ParseFile(path)
		if perr != nil {
			t.Fatalf("parse %s: %v", path, perr)
		}
		if f == nil {
			return nil
		}
		for i := range f.Funcs {
			fn := &f.Funcs[i]
			if fn.Body == nil {
				continue
			}
			bag := WLBag(fn)
			if len(bag) == 0 {
				t.Fatalf("%s: %s produced an empty WL bag; a non-nil body should never yield one",
					path, fn.Name)
			}
			roundTripBag(t, bag, checked)
			checked++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if checked < 100 {
		t.Fatalf("only checked %d functions from %s; the repo walk is not finding the real tree", checked, root)
	}
	t.Logf("round-tripped %d WL bags from doppel's own tree", checked)
}

// repoRoot finds the module root from this test file's own location, so the
// walk finds doppel's tree regardless of the working directory `go test`
// runs from.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/fingerprint/wlcodec_test.go -> repo root is two levels up.
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolved root %q has no go.mod: %v", root, err)
	}
	return root
}

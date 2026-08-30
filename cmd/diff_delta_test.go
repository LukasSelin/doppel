package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/identity"
)

// TestDiffDeltaReportGate is the task's fixture on the `doppel diff` side: a
// session that renames one function and copies another must produce a delta
// naming the rename as `renamed` and the copy as a created pair whose line
// carries the stored explanation.
//
// The explanation is the load-bearing half. It is read back off
// snapshot.Pair.Explain — the sentence the analyzing run recorded — and never
// recomputed here, which is why the fixture pins the exact words rather than
// merely asserting that some sentence is present.
func TestDiffDeltaReportGate(t *testing.T) {
	dir := t.TempDir()
	before := writeSnapshot(t, dir, "before.json", corpusFrom(t, map[string]string{"svc/svc.go": gateSrcBefore}))
	after := writeSnapshot(t, dir, "after.json", corpusFrom(t, map[string]string{"svc/svc.go": gateSrcAfter}))

	out, errOut, code := runDiffCmd(t, before, after)
	if code != exitDiffOK {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}

	if !strings.Contains(out, "renamed 1") {
		t.Errorf("the rename must be classified as renamed:\n%s", out)
	}
	if !strings.Contains(out, "svc.Total") || !strings.Contains(out, "svc.Sum") {
		t.Errorf("the rename must name both sides:\n%s", out)
	}
	if !strings.Contains(out, "pairs created") {
		t.Errorf("the copy must create a pair:\n%s", out)
	}
	if !strings.Contains(out, "svc.Clip <-> svc.Trim") {
		t.Errorf("the created pair must be the copy and its source:\n%s", out)
	}
	if !strings.Contains(out, "explain: identical after rename\n") {
		t.Errorf("the created pair must carry its stored explanation:\n%s", out)
	}
	// The pair the rename moved is named on both sides of the move, and
	// attributed to the rename rather than left as unexplained churn.
	if !strings.Contains(out, "pairs dissolved") || !strings.Contains(out, "svc.Clip <-> svc.Total") {
		t.Errorf("the rename must dissolve the old pair:\n%s", out)
	}
	if !strings.Contains(out, "svc.Sum renamed") {
		t.Errorf("a pair change must name the classified function that explains it:\n%s", out)
	}
}

// The markdown form is the same report, and its first section is the report —
// there is nothing else in the document.
func TestDiffMarkdownOutput(t *testing.T) {
	dir := t.TempDir()
	before := writeSnapshot(t, dir, "before.json", corpusFrom(t, map[string]string{"svc/svc.go": gateSrcBefore}))
	after := writeSnapshot(t, dir, "after.json", corpusFrom(t, map[string]string{"svc/svc.go": gateSrcAfter}))
	md := filepath.Join(dir, "delta.md")

	_, errOut, code := runDiffCmd(t, before, after, "--output", md)
	if code != exitDiffOK {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if !strings.Contains(errOut, "Delta report written to") {
		t.Errorf("stderr should name the file it wrote: %q", errOut)
	}
	data, err := os.ReadFile(md)
	if err != nil {
		t.Fatalf("read markdown: %v", err)
	}
	got := string(data)
	if !strings.HasPrefix(got, "# Delta since the baseline\n") {
		t.Errorf("the delta report must be the document's first section:\n%s", got)
	}
	for _, want := range []string{"## renamed (1)", "## Pairs created", "explain: identical after rename"} {
		if !strings.Contains(got, want) {
			t.Errorf("markdown is missing %q:\n%s", want, got)
		}
	}

	// Two runs of the same input must produce the same bytes.
	md2 := filepath.Join(dir, "delta2.md")
	if _, _, code := runDiffCmd(t, before, after, "--output", md2); code != exitDiffOK {
		t.Fatalf("second run exited %d", code)
	}
	again, err := os.ReadFile(md2)
	if err != nil {
		t.Fatalf("read second markdown: %v", err)
	}
	if string(again) != got {
		t.Error("the delta markdown is not byte-identical across runs")
	}
}

// The JSON payload gained the pair lists without moving a field a consumer of
// the classification already reads.
func TestDiffJSONCarriesPairChanges(t *testing.T) {
	dir := t.TempDir()
	before := writeSnapshot(t, dir, "before.json", corpusFrom(t, map[string]string{"svc/svc.go": gateSrcBefore}))
	after := writeSnapshot(t, dir, "after.json", corpusFrom(t, map[string]string{"svc/svc.go": gateSrcAfter}))

	out, _, code := runDiffCmd(t, before, after, "--format", "json")
	if code != exitDiffOK {
		t.Fatalf("exit %d", code)
	}
	var d identity.Delta
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	if d.Count(identity.Renamed) != 1 {
		t.Errorf("want one rename, got %+v", d.Counts)
	}
	if len(d.Created) == 0 || len(d.Dissolved) == 0 {
		t.Fatalf("want both pair lists populated: %d created, %d dissolved", len(d.Created), len(d.Dissolved))
	}
	// The classification fields still sit at the top level, unmoved.
	var r identity.Result
	if err := json.Unmarshal([]byte(out), &r); err != nil || r.Count(identity.Renamed) != 1 {
		t.Errorf("a consumer reading only the classification must still work: %v %+v", err, r.Counts)
	}
}

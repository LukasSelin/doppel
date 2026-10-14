package dashboard

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func sampleAudit() AuditPayload {
	return AuditPayload{
		Schema:       AuditSchema,
		Target:       "svc",
		Base:         "before",
		Head:         "after",
		Params:       "threshold 0.35 · struct-min 0.00",
		Notes:        []string{"the two runs were produced by different doppel builds"},
		OldFunctions: 10,
		NewFunctions: 11,
		Change: StepChange{
			From: 0, To: 1,
			Counts: []ClassCount{{Class: "renamed", Count: 1}, {Class: "new", Count: 1}, {Class: "unchanged", Count: 9}},
			Changes: []ChangeRow{
				{Class: "renamed", Old: []string{"svc.Sum"}, New: []string{"svc.Total"}, File: "svc/svc.go", Line: 4, Jaccard: 1, Containment: 1, DigestEqual: true, NameChanged: true},
				{Class: "new", New: []string{"svc.Clip"}, File: "svc/svc.go", Line: 20},
			},
			ChangesTotal: 2,
			Created: []PairRow{{A: "svc.Clip", B: "svc.Trim", Score: 1, Overlap: 0.7, MergeWorthy: true,
				Explain: "identical after rename", BClass: "new", Attributable: true}},
			Dissolved:       []PairRow{{A: "svc.Sum", B: "svc.Trim", Score: 0.8, Overlap: 0.5, AClass: "renamed", Attributable: true}},
			CreatedTotal:    1,
			DissolvedTotal:  1,
			AttributableNew: 1,
		},
		CreatedMergeWorthy:    1,
		DissolvedAttributable: 1,
		Units:                 []AuditUnit{{Key: "svc.Clip", File: "svc/svc.go", Line: 20}, {Key: "svc.Trim", File: "svc/svc.go", Line: 12}},
		Bounds:                AuditBounds{MaxChanges: 200, MaxPairs: 60},
	}
}

func renderAudit(t *testing.T, p AuditPayload) string {
	t.Helper()
	var b strings.Builder
	if err := PrintAudit(&b, p); err != nil {
		t.Fatalf("PrintAudit: %v", err)
	}
	return b.String()
}

func extractAudit(t *testing.T, page string) AuditPayload {
	t.Helper()
	m := timelineDataRe.FindStringSubmatch(page)
	if m == nil {
		t.Fatal("no payload script element in the page")
	}
	var got AuditPayload
	if err := json.Unmarshal([]byte(m[1]), &got); err != nil {
		t.Fatalf("payload does not parse as JSON: %v", err)
	}
	return got
}

// TestAuditPayloadHasNoMaps is the determinism rule every page payload holds.
func TestAuditPayloadHasNoMaps(t *testing.T) {
	assertNoMaps(t, reflect.TypeOf(AuditPayload{}), "AuditPayload")
}

func TestPrintAuditRoundTripsThePayload(t *testing.T) {
	want := sampleAudit()
	if got := extractAudit(t, renderAudit(t, want)); !reflect.DeepEqual(got, want) {
		t.Errorf("payload did not round-trip\n got: %+v\nwant: %+v", got, want)
	}
}

// The review page holds the dashboard's contract — nothing to fetch, opens from
// file:// — and is built from the same shared pieces: the design tokens, the
// shared component sheet and the payload-free helpers in ui.js.
func TestPrintAuditIsSelfContained(t *testing.T) {
	page := renderAudit(t, sampleAudit())
	for _, bad := range []string{"<script src=", "<link rel=\"stylesheet\"", "fetch(", "XMLHttpRequest"} {
		if strings.Contains(page, bad) {
			t.Errorf("page is not self-contained: found %q", bad)
		}
	}
	for _, want := range []string{"--color-accent", ".au-verdict", ".bar-shape", "var DOPPEL_UI", "au-introduced"} {
		if !strings.Contains(page, want) {
			t.Errorf("page is missing %q", want)
		}
	}
	if strings.Contains(page, "var DOPPEL =") {
		t.Error("the single-run dashboard's core.js reached the review page; it reads a payload this page does not carry")
	}
}

func TestPrintAuditEscapesScriptClose(t *testing.T) {
	p := sampleAudit()
	p.Change.Created[0].A = "svc.Clip</script><script>alert(1)</script>"
	p.Target = "wat</script>"
	page := renderAudit(t, p)
	if n := strings.Count(page, "</script>"); n != 2 {
		t.Errorf("</script> appears %d times, want 2 — the payload leaked out of its element", n)
	}
	if got := extractAudit(t, page); got.Change.Created[0].A != p.Change.Created[0].A {
		t.Errorf("key did not survive escaping: %q", got.Change.Created[0].A)
	}
}

func TestPrintAuditIsDeterministic(t *testing.T) {
	p := sampleAudit()
	if a, b := renderAudit(t, p), renderAudit(t, p); a != b {
		t.Error("two renders of one comparison differ; the page must be reproducible")
	}
}

func TestAuditDevAssetsMatchEmbedded(t *testing.T) {
	p := sampleAudit()
	embedded := renderAudit(t, p)
	t.Setenv(DevAssetsEnv, "assets")
	if dev := renderAudit(t, p); dev != embedded {
		t.Error("DOPPEL_DASHBOARD_ASSETS render differs from the embedded one")
	}
}

func TestEmptyAuditStillRenders(t *testing.T) {
	renderAudit(t, AuditPayload{Schema: AuditSchema})
}

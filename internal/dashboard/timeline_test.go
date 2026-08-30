package dashboard

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// sampleTimeline is a small but complete series: three revisions, one rename,
// one pair created and one dissolved, one track that survives the rename and
// one that is deleted.
func sampleTimeline() TimelinePayload {
	return TimelinePayload{
		Schema: TimelineSchema,
		Target: "sample",
		Params: "threshold 0.38 · struct-min 0.00",
		Steps: []Step{
			{Index: 0, Label: "aaaaaaa", Functions: 2, Pairs: 1, MergeWorthy: 1, Concepts: 3, Compression: 4.5, NNScored: 2, NNP50: 0.4, NNP90: 0.8},
			{Index: 1, Label: "bbbbbbb", Functions: 2, Pairs: 1, MergeWorthy: 1, Concepts: 3, Compression: 4.6, NNScored: 2, NNP50: 0.45, NNP90: 0.82, UnusedSeeds: []string{"retry"}},
			{Index: 2, Label: "ccccccc", Functions: 1, Pairs: 0, Concepts: 2, Compression: 4.1, NNScored: 1},
		},
		Changes: []StepChange{
			{
				From: 0, To: 1,
				Counts: []ClassCount{{Class: "renamed", Count: 1}, {Class: "unchanged", Count: 1}},
				Changes: []ChangeRow{{
					Class: "renamed", Old: []string{"alpha.First"}, New: []string{"alpha.Premier"},
					File: "a.go", Line: 3, Jaccard: 0.9, Containment: 0.95, DigestEqual: false,
				}},
				ChangesTotal: 1,
				Created: []PairRow{{
					A: "alpha.Premier", B: "alpha.Second", Score: 0.8, Overlap: 0.5,
					MergeWorthy: true, Explain: "identical after rename", AClass: "renamed",
					BClass: "unchanged", Attributable: true,
				}},
				CreatedTotal: 1, AttributableNew: 1,
			},
			{
				From: 1, To: 2,
				Counts:       []ClassCount{{Class: "deleted", Count: 1}, {Class: "unchanged", Count: 1}},
				Changes:      []ChangeRow{{Class: "deleted", Old: []string{"alpha.Second"}, File: "a.go", Line: 30}},
				ChangesTotal: 1,
				Dissolved: []PairRow{{
					A: "alpha.Premier", B: "alpha.Second", Score: 0.8, Overlap: 0.5,
					BClass: "deleted", Attributable: true,
				}},
				DissolvedTotal: 1,
			},
		},
		Tracks: []TimelineTrack{{
			ID: 0, First: 0, Last: 2, Label: "alpha.Premier",
			Points: []TrackStop{
				{Step: 0, Key: "alpha.First"},
				{Step: 1, Key: "alpha.Premier", Class: "renamed"},
				{Step: 2, Key: "alpha.Premier", Class: "unchanged"},
			},
		}, {
			ID: 1, First: 0, Last: 1, Label: "alpha.Second", Fate: "deleted",
			Points: []TrackStop{
				{Step: 0, Key: "alpha.Second"},
				{Step: 1, Key: "alpha.Second", Class: "unchanged"},
			},
		}},
		Notes:  []string{"the two runs were produced by different doppel builds"},
		Bounds: TimelineBounds{MaxChanges: 200, MaxPairs: 60, MaxTracks: 400, FlatTracks: 4},
	}
}

// TestTimelinePayloadHasNoMaps is TestPayloadHasNoMaps' rule applied to the
// second payload, for the same reason: map iteration order would make an
// unchanged series render different bytes.
func TestTimelinePayloadHasNoMaps(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var walk func(rt reflect.Type, path string)
	walk = func(rt reflect.Type, path string) {
		if seen[rt] {
			return
		}
		seen[rt] = true
		switch rt.Kind() {
		case reflect.Map:
			t.Errorf("%s is a map; the payload must use sorted slices", path)
		case reflect.Slice, reflect.Ptr, reflect.Array:
			walk(rt.Elem(), path+"[]")
		case reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				walk(f.Type, path+"."+f.Name)
			}
		}
	}
	walk(reflect.TypeOf(TimelinePayload{}), "TimelinePayload")
}

func renderTimeline(t *testing.T, p TimelinePayload) string {
	t.Helper()
	var b strings.Builder
	if err := PrintTimeline(&b, p); err != nil {
		t.Fatalf("PrintTimeline: %v", err)
	}
	return b.String()
}

var timelineDataRe = regexp.MustCompile(`(?s)<script type="application/json" id="doppel-data">(.*?)</script>`)

func extractTimeline(t *testing.T, page string) TimelinePayload {
	t.Helper()
	m := timelineDataRe.FindStringSubmatch(page)
	if m == nil {
		t.Fatal("no payload script element in the page")
	}
	var got TimelinePayload
	if err := json.Unmarshal([]byte(m[1]), &got); err != nil {
		t.Fatalf("payload does not parse as JSON: %v", err)
	}
	return got
}

func TestPrintTimelineRoundTripsThePayload(t *testing.T) {
	want := sampleTimeline()
	got := extractTimeline(t, renderTimeline(t, want))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("payload did not round-trip\n got: %+v\nwant: %+v", got, want)
	}
}

// TestPrintTimelineIsSelfContained holds the timeline page to exactly the
// contract the dashboard has: it opens from file:// with nothing to fetch, and
// no library rides along.
func TestPrintTimelineIsSelfContained(t *testing.T) {
	page := renderTimeline(t, sampleTimeline())
	for _, bad := range []string{"<script src=", "<link rel=\"stylesheet\"", "fetch(", "XMLHttpRequest"} {
		if strings.Contains(page, bad) {
			t.Errorf("page is not self-contained: found %q", bad)
		}
	}
	if !strings.Contains(page, "--color-accent") {
		t.Error("the design system tokens were not inlined")
	}
	if !strings.Contains(page, ".track-cell") {
		t.Error("the timeline stylesheet was not inlined")
	}
	if imports := strings.Count(page, "@import url("); imports != 1 {
		t.Errorf("@import count = %d, want exactly 1 (the font)", imports)
	}
}

// TestPrintTimelineEscapesScriptClose is the same trap as the dashboard's, one
// field along: this page carries function *keys* rather than bodies, and a key
// is just as capable of closing the element that carries it.
func TestPrintTimelineEscapesScriptClose(t *testing.T) {
	p := sampleTimeline()
	p.Tracks[0].Label = "alpha.Render</script><script>alert(1)</script>"
	p.Tracks[0].Points[2].Key = p.Tracks[0].Label
	p.Target = "wat</script>"

	page := renderTimeline(t, p)
	if n := strings.Count(page, "</script>"); n != 2 {
		t.Errorf("</script> appears %d times, want 2 — the payload leaked out of its element", n)
	}
	got := extractTimeline(t, page)
	if got.Tracks[0].Label != p.Tracks[0].Label {
		t.Errorf("label did not survive escaping: %q", got.Tracks[0].Label)
	}
	if strings.Contains(page, "<title>doppel timeline — wat</script>") {
		t.Error("target name reached the title unescaped")
	}
}

func TestPrintTimelineIsDeterministic(t *testing.T) {
	p := sampleTimeline()
	if a, b := renderTimeline(t, p), renderTimeline(t, p); a != b {
		t.Error("two renders of one series differ; the page must be reproducible")
	}
}

func TestTimelineDevAssetsMatchEmbedded(t *testing.T) {
	p := sampleTimeline()
	embedded := renderTimeline(t, p)
	t.Setenv(DevAssetsEnv, "assets")
	if dev := renderTimeline(t, p); dev != embedded {
		t.Error("DOPPEL_DASHBOARD_ASSETS render differs from the embedded one")
	}
}

// TestTimelineSchemaIsIndependent pins the reason there are two constants. The
// two payloads answer different questions, and a page written against one must
// be able to tell it was handed the other.
func TestTimelineSchemaIsIndependent(t *testing.T) {
	page := renderTimeline(t, sampleTimeline())
	if !strings.Contains(page, `"schema":1`) {
		t.Error("the timeline payload does not carry its schema")
	}
	if strings.Contains(page, "screen-map") {
		t.Error("the dashboard shell was rendered for a timeline payload")
	}
}

func TestEmptyTimelineStillRenders(t *testing.T) {
	page := renderTimeline(t, TimelinePayload{Schema: TimelineSchema, Target: "empty"})
	if !strings.Contains(page, "doppel-data") {
		t.Error("an empty series should still produce a page")
	}
}

func TestWriteTimelineJSON(t *testing.T) {
	var b strings.Builder
	if err := WriteTimelineJSON(&b, sampleTimeline()); err != nil {
		t.Fatalf("WriteTimelineJSON: %v", err)
	}
	var got TimelinePayload
	if err := json.Unmarshal([]byte(b.String()), &got); err != nil {
		t.Fatalf("output does not parse: %v", err)
	}
	if !reflect.DeepEqual(got, sampleTimeline()) {
		t.Error("the JSON form did not round-trip")
	}
}

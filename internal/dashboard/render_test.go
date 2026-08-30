package dashboard

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// samplePayload is a small but complete payload: two units, one edge between
// them, both bodies inlined.
func samplePayload() Payload {
	return Payload{
		Schema: Schema,
		Target: "sample",
		Facts:  Facts{Functions: 2, Packages: 1, Threshold: 0.6, TestsMode: "excluded", Pairs: 1},
		Packages: []Package{
			{Name: "alpha", Functions: 2, Norm: 0.8},
		},
		Units: []Unit{
			{ID: 0, Key: "alpha.First", Name: "First", Package: "alpha", File: "a.go", Line: 3,
				Role: "leaf", Tags: []string{"retry"}, Concept: "retry", FanIn: 1, Nodes: 20, Fit: 0.9},
			{ID: 1, Key: "alpha.Second", Name: "Second", Package: "alpha", File: "a.go", Line: 30,
				Role: "leaf", Concept: "retry", Nodes: 22, Fit: -1},
		},
		Edges: []Edge{{
			A: 0, B: 1, Shape: 0.8, Overlap: 0.5, Total: 12, Trophic: 0.7, Rank: 3.4,
			Channels: []string{"shape"}, Breakdown: [5]float64{0.8, 0.9, 1, 1, 0.9},
			Chains: []Chain{{Level: 2, Energy: 4.5, Render: "if(bin:!=(id,nil))"}},
		}},
		Bodies: []Body{
			{Unit: 0, Text: "func First() {}"},
			{Unit: 1, Text: "func Second() {}"},
		},
		Concepts: []string{"retry"},
	}
}

// TestPayloadHasNoMaps mirrors snapshot's own rule for the same reason: map
// iteration order would make an unchanged tree render different bytes, and a
// nondeterministic report is a bug in this tool.
func TestPayloadHasNoMaps(t *testing.T) {
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
	walk(reflect.TypeOf(Payload{}), "Payload")
}

func render(t *testing.T, p Payload) string {
	t.Helper()
	var b strings.Builder
	if err := Print(&b, p); err != nil {
		t.Fatalf("Print: %v", err)
	}
	return b.String()
}

var dataRe = regexp.MustCompile(`(?s)<script type="application/json" id="doppel-data">(.*?)</script>`)

func extractPayload(t *testing.T, page string) Payload {
	t.Helper()
	m := dataRe.FindStringSubmatch(page)
	if m == nil {
		t.Fatal("no payload script element in the page")
	}
	var got Payload
	if err := json.Unmarshal([]byte(m[1]), &got); err != nil {
		t.Fatalf("payload does not parse as JSON: %v", err)
	}
	return got
}

func TestPrintRoundTripsThePayload(t *testing.T) {
	want := samplePayload()
	got := extractPayload(t, render(t, want))
	if !reflect.DeepEqual(got, want) {
		t.Errorf("payload did not round-trip\n got: %+v\nwant: %+v", got, want)
	}
}

// TestPrintIsSelfContained is the contract the report has always had: it opens
// from file:// with nothing to fetch. The design system's font import is the
// one allowed external reference, and it falls back to system-ui offline.
func TestPrintIsSelfContained(t *testing.T) {
	page := render(t, samplePayload())
	for _, bad := range []string{"<script src=", "<link rel=\"stylesheet\"", "fetch(", "XMLHttpRequest"} {
		if strings.Contains(page, bad) {
			t.Errorf("page is not self-contained: found %q", bad)
		}
	}
	if !strings.Contains(page, "powerDiagram") {
		t.Error("the map geometry was not inlined")
	}
	// There is no vendored JavaScript, and that is a property worth pinning:
	// the map is a power diagram in plain SVG, so a graph library would be
	// several hundred KB in every report for a screen that does not need one.
	if strings.Contains(page, "cytoscape") {
		t.Error("a graph library reappeared in the page")
	}
	if !strings.Contains(page, "--color-accent") {
		t.Error("the design system tokens were not inlined")
	}
	// The only permitted remote reference.
	imports := strings.Count(page, "@import url(")
	if imports != 1 {
		t.Errorf("@import count = %d, want exactly 1 (the font)", imports)
	}
}

// TestPrintEscapesScriptClose is the trap this renderer exists next to. An
// analysed function whose body or name contains </script> would otherwise
// terminate the element carrying the payload and break the whole page.
func TestPrintEscapesScriptClose(t *testing.T) {
	p := samplePayload()
	p.Units[0].Name = "Render</script><script>alert(1)</script>"
	p.Units[0].Key = "alpha." + p.Units[0].Name
	p.Bodies[0].Text = "func Render() { return `</script>` } // < & >"
	p.Target = "wat</script>"

	page := render(t, p)

	// Exactly the two script elements the shell declares: payload and app.
	if n := strings.Count(page, "</script>"); n != 2 {
		t.Errorf("</script> appears %d times, want 2 — the payload leaked out of its element", n)
	}
	got := extractPayload(t, page)
	if got.Bodies[0].Text != p.Bodies[0].Text {
		t.Errorf("body did not survive escaping:\n got %q\nwant %q", got.Bodies[0].Text, p.Bodies[0].Text)
	}
	if got.Units[0].Name != p.Units[0].Name {
		t.Errorf("name did not survive escaping: %q", got.Units[0].Name)
	}
	// The title is html/template's job, not the JSON encoder's.
	if strings.Contains(page, "<title>doppel — wat</script>") {
		t.Error("target name reached the title unescaped")
	}
}

func TestPrintIsDeterministic(t *testing.T) {
	p := samplePayload()
	if a, b := render(t, p), render(t, p); a != b {
		t.Error("two renders of one payload differ; the page must be reproducible")
	}
}

// TestDevAssetsMatchEmbedded pins the dev seam to the shipped path: reading
// assets off disk must produce exactly the embedded render, or iterating with
// it would be iterating on something else.
func TestDevAssetsMatchEmbedded(t *testing.T) {
	p := samplePayload()
	embedded := render(t, p)
	t.Setenv(DevAssetsEnv, "assets")
	if dev := render(t, p); dev != embedded {
		t.Error("DOPPEL_DASHBOARD_ASSETS render differs from the embedded one")
	}
}

func TestDevAssetsRejectsABadDirectory(t *testing.T) {
	t.Setenv(DevAssetsEnv, "no/such/directory")
	if err := Print(&strings.Builder{}, samplePayload()); err == nil {
		t.Error("a missing dev asset directory should be an error, not a silent fallback")
	}
}

func TestEmptyPayloadStillRenders(t *testing.T) {
	page := render(t, Payload{Schema: Schema, Target: "empty"})
	if !strings.Contains(page, "doppel-data") {
		t.Error("an empty corpus should still produce a page")
	}
}

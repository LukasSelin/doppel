package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LukasSelin/doppel/internal/gofront"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/reporter"
)

// postEditFixture is a corpus big enough for corpus weights to mean
// something: on a corpus of two near-identical functions every label has
// df == N, every weight is 0 and nothing can match (see "The fixture trap").
// mergeCounts and tallyPairs are already twins, so the baseline holds them as
// a pair; the rest are deliberately unalike.
var postEditFixture = map[string]string{
	"store/merge.go": `package store

import "sort"

func mergeCounts(a, b map[string]int) []string {
	out := make(map[string]int, len(a)+len(b))
	for k, v := range a {
		out[k] += v
	}
	for k, v := range b {
		if v > 0 {
			out[k] += v
		}
	}
	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
`,
	"tally/pairs.go": `package tally

import "sort"

func tallyPairs(a, b map[string]int) []string {
	out := make(map[string]int, len(a)+len(b))
	for k, v := range a {
		out[k] += v
	}
	for k, v := range b {
		if v > 0 {
			out[k] += v
		}
	}
	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
`,
	"render/render.go": `package render

import (
	"fmt"
	"strings"
)

func Table(rows [][]string) string {
	var b strings.Builder
	for i, r := range rows {
		if i == 0 {
			b.WriteString(strings.ToUpper(strings.Join(r, " | ")))
		} else {
			b.WriteString(strings.Join(r, " | "))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func Bar(n, width int) string {
	if n > width {
		n = width
	}
	return fmt.Sprintf("[%s%s]", strings.Repeat("#", n), strings.Repeat(" ", width-n))
}

func Plural(n int, word string) string {
	switch {
	case n == 1:
		return "1 " + word
	case strings.HasSuffix(word, "s"):
		return fmt.Sprintf("%d %ses", n, word)
	default:
		return fmt.Sprintf("%d %ss", n, word)
	}
}
`,
	"net/retry.go": `package net

import (
	"errors"
	"time"
)

func Retry(attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(delay)
		delay *= 2
	}
	return errors.Join(errors.New("retry: gave up"), err)
}

func Backoff(base time.Duration, n int) time.Duration {
	d := base
	for i := 0; i < n && d < time.Minute; i++ {
		d *= 2
	}
	return d
}
`,
	"geo/geo.go": `package geo

import "math"

type Point struct{ X, Y float64 }

func Dist(a, b Point) float64 {
	dx, dy := a.X-b.X, a.Y-b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func Centroid(ps []Point) Point {
	var c Point
	if len(ps) == 0 {
		return c
	}
	for _, p := range ps {
		c.X += p.X
		c.Y += p.Y
	}
	c.X /= float64(len(ps))
	c.Y /= float64(len(ps))
	return c
}
`,
}

// cloneOfMerge is mergeCounts again under a new name in a new package: the
// near duplicate the post-edit hook exists to catch.
const cloneOfMerge = `package report

import "sort"

func unionKeys(a, b map[string]int) []string {
	out := make(map[string]int, len(a)+len(b))
	for k, v := range a {
		out[k] += v
	}
	for k, v := range b {
		if v > 0 {
			out[k] += v
		}
	}
	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
`

// newPostEditSession writes the fixture, opts it into probing, and records a
// session-start baseline for it. It returns the root and the session id.
func newPostEditSession(t *testing.T, probe string) (string, string) {
	t.Helper()
	root := t.TempDir()
	for rel, src := range postEditFixture {
		writeCorpus(t, root, rel, src)
	}
	writeCorpus(t, root, ".doppel.json", `{"hook-probe":"`+probe+`"}`)
	sid := "post-edit-test-" + t.Name()
	t.Cleanup(func() {
		os.Remove(baselinePath(sid))
		os.Remove(deltaPathFor(baselinePath(sid)))
	})
	_, out, errOut := runHook(t, "session-start", hookPayload(t, sid, root))
	if errOut != "" || out == "" {
		t.Fatalf("session-start: stdout %q stderr %q", short(out), errOut)
	}
	return root, sid
}

func postEdit(t *testing.T, root, sid, rel string) (string, string) {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"session_id": sid, "cwd": root, "tool_name": "Write",
		"tool_input": map[string]any{"file_path": filepath.Join(root, filepath.FromSlash(rel))},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, out, errOut := runHook(t, "post-edit", string(data))
	return out, errOut
}

// postEditContext is the additionalContext a post-edit run emitted, or "".
func postEditContext(t *testing.T, out string) string {
	t.Helper()
	if out == "" {
		return ""
	}
	var v hookResponse
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("stdout is not a hook response: %q", out)
	}
	if v.HookSpecificOutput.HookEventName != "PostToolUse" {
		t.Errorf("hookEventName = %q, want PostToolUse", v.HookSpecificOutput.HookEventName)
	}
	if strings.Contains(out, "permissionDecision") || strings.Contains(out, `"decision"`) {
		t.Errorf("post-edit made a decision; it may only advise: %s", out)
	}
	return v.HookSpecificOutput.AdditionalContext
}

func TestPostEditReportsAClone(t *testing.T) {
	root, sid := newPostEditSession(t, "on")
	writeCorpus(t, root, "report/union.go", cloneOfMerge)

	out, errOut := postEdit(t, root, sid, "report/union.go")
	if errOut != "" {
		t.Errorf("stderr: %q", errOut)
	}
	ctx := postEditContext(t, out)
	for _, want := range []string{"report.unionKeys (new, report/union.go:5)", "store.mergeCounts", "tally.tallyPairs", "code-shape 1.00"} {
		if !strings.Contains(ctx, want) {
			t.Errorf("note is missing %q:\n%s", want, ctx)
		}
	}

	// The same edit again: everything it found was already said.
	out, errOut = postEdit(t, root, sid, "report/union.go")
	if out != "" || errOut != "" {
		t.Errorf("a repeated edit spoke again: stdout %q stderr %q", short(out), errOut)
	}

	// And the Stop hook's ledger learned the same pairs, so it will not
	// repeat them at the end of the turn.
	base, err := readBaseline(baselinePath(sid))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"probe:report.unionKeys|store.mergeCounts": false,
		"new:report.unionKeys|store.mergeCounts":   false,
	}
	for _, k := range base.Reported {
		if _, ok := want[k]; ok {
			want[k] = true
		}
	}
	for k, seen := range want {
		if !seen {
			t.Errorf("ledger is missing %q (have %v)", k, base.Reported)
		}
	}
}

func TestPostEditSilentCases(t *testing.T) {
	t.Run("unrelated function", func(t *testing.T) {
		root, sid := newPostEditSession(t, "on")
		writeCorpus(t, root, "report/hello.go", "package report\n\nfunc Hello(name string) string { return \"hello \" + name }\n")
		assertSilent(t, postEditOf(t, root, sid, "report/hello.go"))
	})
	t.Run("unparseable file", func(t *testing.T) {
		root, sid := newPostEditSession(t, "on")
		writeCorpus(t, root, "report/union.go", strings.Replace(cloneOfMerge, "return keys\n}", "return keys", 1))
		assertSilent(t, postEditOf(t, root, sid, "report/union.go"))
	})
	t.Run("twin already paired in the baseline", func(t *testing.T) {
		root, sid := newPostEditSession(t, "on")
		// Silence must come from the baseline pair, not from the twins never
		// having been retrieved at all.
		base, err := readBaseline(baselinePath(sid))
		if err != nil {
			t.Fatal(err)
		}
		paired := false
		for _, p := range base.Snapshot.Pairs {
			paired = paired || p.Key() == "store.mergeCounts <-> tally.tallyPairs"
		}
		if !paired {
			t.Fatalf("fixture twins are not a baseline pair; pairs %v", base.Snapshot.Pairs)
		}
		// A changed body whose only near match was its twin before the
		// session began: the session did not create that pair.
		edited := strings.Replace(postEditFixture["store/merge.go"], "if v > 0 {", "if v >= 0 {", 1)
		writeCorpus(t, root, "store/merge.go", edited)
		assertSilent(t, postEditOf(t, root, sid, "store/merge.go"))
	})
	t.Run("probing off", func(t *testing.T) {
		root, sid := newPostEditSession(t, "off")
		writeCorpus(t, root, "report/union.go", cloneOfMerge)
		assertSilent(t, postEditOf(t, root, sid, "report/union.go"))
	})
	t.Run("no baseline", func(t *testing.T) {
		root, _ := newPostEditSession(t, "on")
		writeCorpus(t, root, "report/union.go", cloneOfMerge)
		assertSilent(t, postEditOf(t, root, "post-edit-test-no-such-session", "report/union.go"))
	})
	t.Run("non-code file", func(t *testing.T) {
		root, sid := newPostEditSession(t, "on")
		writeCorpus(t, root, "NOTES.md", "# notes\n")
		assertSilent(t, postEditOf(t, root, sid, "NOTES.md"))
	})
}

// DOPPEL_TIMING writes stage lines to the pipeline's progress writer, and a
// hook's is io.Discard. A firing run must stay off stderr with it exported.
func TestPostEditStaysOffStderrUnderTiming(t *testing.T) {
	t.Setenv("DOPPEL_TIMING", "1")
	root, sid := newPostEditSession(t, "on")
	writeCorpus(t, root, "report/union.go", cloneOfMerge)
	out, errOut := postEdit(t, root, sid, "report/union.go")
	if errOut != "" {
		t.Errorf("stderr under DOPPEL_TIMING: %q", errOut)
	}
	if postEditContext(t, out) == "" {
		t.Errorf("the clone did not fire under DOPPEL_TIMING")
	}
}

type hookRun struct{ out, err string }

func postEditOf(t *testing.T, root, sid, rel string) hookRun {
	out, err := postEdit(t, root, sid, rel)
	return hookRun{out, err}
}

func assertSilent(t *testing.T, r hookRun) {
	t.Helper()
	if r.out != "" || r.err != "" {
		t.Errorf("want silence, got stdout %q stderr %q", r.out, r.err)
	}
}

// A pair is said once whichever side was probed, and not again after the Stop
// hook reported it: both ledgers share the unordered `new:` key.
func TestUnprobedDedupesThePairNotTheDirection(t *testing.T) {
	probes := []reporter.ProbeResult{{Key: "b.G", Matches: []reporter.ProbeMatch{{Key: "a.F"}, {Key: "c.H"}}}}
	reported := []string{probeLedgerKey("a.F", "b.G"), reporter.NewPairKey("a.F", "b.G")}
	got := unprobed(probes, reported)
	if len(got) != 1 || len(got[0].Matches) != 1 || got[0].Matches[0].Key != "c.H" {
		t.Errorf("unprobed = %+v; want only b.G ~ c.H", got)
	}
	if reporter.NewPairKey("b.G", "a.F") != "new:a.F|b.G" {
		t.Errorf("NewPairKey is not order-free: %q", reporter.NewPairKey("b.G", "a.F"))
	}
}

func TestUnreferenceableIsTheLanguageRuleOnly(t *testing.T) {
	cases := []struct {
		u    parser.CodeUnit
		want bool
	}{
		{parser.CodeUnit{Lang: gofront.Lang, Name: "init", Package: "cmd"}, true},
		{parser.CodeUnit{Lang: gofront.Lang, Name: "main", Package: "main"}, true},
		{parser.CodeUnit{Lang: gofront.Lang, Name: "main", Package: "app"}, false},
		{parser.CodeUnit{Lang: gofront.Lang, Name: "init", Package: "cmd", ReceiverType: "*T"}, false},
		{parser.CodeUnit{Lang: "python", Name: "init", Package: "pkg"}, false},
	}
	for _, c := range cases {
		if got := unreferenceable(c.u); got != c.want {
			t.Errorf("unreferenceable(%s.%s lang %s recv %q) = %v, want %v", c.u.Package, c.u.Name, c.u.Lang, c.u.ReceiverType, got, c.want)
		}
	}
}

// walkSkips mirrors index()'s walk: a file under a directory the walk skips
// can hold no unit, so the hook answers before paying for the index.
func TestWalkSkipsMirrorsTheWalk(t *testing.T) {
	cases := []struct {
		exclude []string
		rel     string
		want    bool
	}{
		{nil, "main.go", false},
		{nil, "internal/store/store.go", false},
		{nil, "vendor/github.com/x/y.go", true},
		{nil, "web/node_modules/pkg/index.js", true},
		{nil, ".claude/worktrees/x/a.go", true},
		{[]string{"internal/proto"}, "internal/proto/gen.go", true},
		{[]string{"!vendor"}, "vendor/github.com/x/y.go", false},
	}
	for _, c := range cases {
		if got := walkSkips(Params{Exclude: c.exclude}, c.rel); got != c.want {
			t.Errorf("walkSkips(%v, %q) = %v, want %v", c.exclude, c.rel, got, c.want)
		}
	}
}

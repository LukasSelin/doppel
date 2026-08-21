package fingerprint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

// buildFrom parses one fixture and fingerprints its first function. Local to
// the pattern tests so the existing fixtures stay untouched.
func buildFrom(t *testing.T, src string) Fingerprint {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fix.go", "package fix\n"+src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			return Build(fd)
		}
	}
	t.Fatal("no function in fixture")
	return Fingerprint{}
}

func renders(fp Fingerprint, level uint8) []string {
	var out []string
	for _, p := range fp.Patterns {
		if p.Level == level && p.Render != "" {
			out = append(out, p.Render)
		}
	}
	return out
}

func hasRender(fp Fingerprint, level uint8, render string) bool {
	for _, p := range fp.Patterns {
		if p.Level == level && p.Render == render {
			return true
		}
	}
	return false
}

// A trivial Sprintf method: L1 call pattern, L2 return-of-call, and no motif
// patterns at all — there is no chain to summarize.
func TestPatternsTrivialSprintfMethod(t *testing.T) {
	fp := buildFrom(t, `
func (e T) Error() string {
	return fmt.Sprintf("boom %s %d", e.msg, e.code)
}
type T struct{ msg string; code int }
`)
	if !hasRender(fp, LevelAction, "return(call:Sprintf)") {
		t.Errorf("missing L2 return(call:Sprintf); actions = %v", renders(fp, LevelAction))
	}
	if got := renders(fp, LevelMotif); got != nil {
		t.Errorf("trivial method has motif patterns: %v", got)
	}
	// L1 renders are serialization-only, never stored.
	for _, p := range fp.Patterns {
		if p.Level == LevelExpr && p.Render != "" {
			t.Errorf("L1 pattern carries a render: %q", p.Render)
		}
	}
	hasL1Call := false
	for _, p := range fp.Patterns {
		if p.Level == LevelExpr && p.Hash == patternHash(LevelExpr, "call:Sprintf") {
			hasL1Call = true
		}
	}
	if !hasL1Call {
		t.Error("missing L1 call:Sprintf pattern hash")
	}
}

const scanLoopSrc = `
func readIDs(path string) ([]int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var ids []int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		id, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, scanner.Err()
}
`

// The essay's flagship motif: the scan/trim/parse/append loop must produce a
// legible loop summary and the call-then-errcheck bigram.
func TestPatternsScanLoopMotifs(t *testing.T) {
	fp := buildFrom(t, scanLoopSrc)

	wantLoop := "for{ call:Scan call:TrimSpace call:Text call:Atoi call:append }"
	if !hasRender(fp, LevelMotif, wantLoop) {
		t.Errorf("missing loop summary %q; motifs = %v", wantLoop, renders(fp, LevelMotif))
	}
	wantSeq := "seq[ assign:=(call:Atoi) ; if(bin:!=(id,nil)) ]"
	if !hasRender(fp, LevelMotif, wantSeq) {
		t.Errorf("missing bigram %q; motifs = %v", wantSeq, renders(fp, LevelMotif))
	}
	if !hasRender(fp, LevelAction, "defer(call:Close)") {
		t.Errorf("missing defer(call:Close); actions = %v", renders(fp, LevelAction))
	}
	if !hasRender(fp, LevelAction, "if(bin:!=(id,nil))") {
		t.Errorf("missing err-check if; actions = %v", renders(fp, LevelAction))
	}
}

// Multiset counts: the two identical err-check returns aggregate to Count 2.
func TestPatternsMultisetCounts(t *testing.T) {
	fp := buildFrom(t, scanLoopSrc)
	for _, p := range fp.Patterns {
		if p.Level == LevelAction && p.Render == "return(nil,id)" {
			if p.Count != 2 {
				t.Errorf("return(nil,id) count = %d, want 2", p.Count)
			}
			return
		}
	}
	t.Fatalf("return(nil,id) not found; actions = %v", renders(fp, LevelAction))
}

// A rename-only copy produces the identical pattern multiset.
func TestPatternsRenameInvariant(t *testing.T) {
	a := buildFrom(t, scanLoopSrc)
	b := buildFrom(t, strings.NewReplacer(
		"readIDs", "loadNumbers", "path", "fname", "file", "fh",
		"ids", "nums", "scanner", "sc", "line", "text", "id", "n",
	).Replace(scanLoopSrc))
	if !reflect.DeepEqual(a.Patterns, b.Patterns) {
		t.Error("renamed clone produced a different pattern multiset")
	}
}

func TestPatternsDeterministic(t *testing.T) {
	first := buildFrom(t, scanLoopSrc)
	for i := 0; i < 25; i++ {
		if got := buildFrom(t, scanLoopSrc); !reflect.DeepEqual(got.Patterns, first.Patterns) {
			t.Fatalf("run %d diverged", i)
		}
	}
	for i := 1; i < len(first.Patterns); i++ {
		if first.Patterns[i-1].Hash > first.Patterns[i].Hash {
			t.Fatal("patterns not sorted by hash")
		}
	}
}

// Loop summaries cap at 8 distinct callees plus a truncation marker.
func TestPatternsLoopSummaryCap(t *testing.T) {
	var b strings.Builder
	b.WriteString("func many(xs []int) {\n\tfor _, x := range xs {\n")
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&b, "\t\thelper%d(x)\n", i)
	}
	b.WriteString("\t}\n}\n")
	fp := buildFrom(t, b.String())

	motifs := renders(fp, LevelMotif)
	found := false
	for _, r := range motifs {
		if strings.HasPrefix(r, "range{ ") {
			found = true
			if !strings.HasSuffix(r, "... }") {
				t.Errorf("capped summary missing truncation marker: %q", r)
			}
			if got := strings.Count(r, "call:"); got != loopSummaryCap {
				t.Errorf("summary carries %d callees, want %d: %q", got, loopSummaryCap, r)
			}
		}
	}
	if !found {
		t.Fatalf("no range summary found; motifs = %v", motifs)
	}
}

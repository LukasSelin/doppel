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

// The multi-width L0 index: the k=3 windows keep their legacy untagged hash
// (every pre-existing df is unchanged by the widening), the extra widths are
// tagged so cross-width windows can never collide, and a stream shorter than
// an extra width emits nothing for it rather than clamping.
func TestPatternsMultiWidthL0(t *testing.T) {
	fp := buildFrom(t, `
func widths(a, b int) int {
	if a > b {
		return a
	}
	return b
}`)
	tokens, _, _, _ := walkFixture(t, `
func widths(a, b int) int {
	if a > b {
		return a
	}
	return b
}`)
	if len(tokens) < 5 {
		t.Fatalf("fixture stream too short: %v", tokens)
	}
	hashes := map[uint64]bool{}
	for _, p := range fp.Patterns {
		if p.Level == LevelToken {
			hashes[p.Hash] = true
		}
	}
	// Legacy k=3 hash input is byte-identical to the variadic form.
	legacy := patternHash(LevelToken, tokens[0:3]...)
	if legacy != patternHashL0("", tokens[0:3]) {
		t.Fatal("patternHashL0 with empty tag must equal the legacy k=3 hash")
	}
	if !hashes[legacy] {
		t.Errorf("legacy k=3 window missing from the multiset")
	}
	// Every extra width present, and tagged so it cannot collide with an
	// untagged window over the same tokens. (Width 2 was measured out — see
	// l0ExtraWidths — so only w5 exists today.)
	for _, w := range l0ExtraWidths {
		h := patternHashL0(widthTag(w), tokens[0:w])
		if !hashes[h] {
			t.Errorf("w%d window missing from the multiset", w)
		}
		if h == patternHashL0("", tokens[0:w]) {
			t.Errorf("w%d hash collides with an untagged window over the same tokens", w)
		}
	}
	// A stream shorter than an extra width emits nothing for it — never a
	// clamped window: every L0 hash in the short multiset must be
	// reproducible from the stream's own (possibly clamped) k=3 windows.
	short := buildFrom(t, `
func tiny() {
	println()
}`)
	stokens, _, _, _ := walkFixture(t, `
func tiny() {
	println()
}`)
	if len(stokens) >= 5 {
		t.Fatalf("short fixture is not short: %v", stokens)
	}
	want := map[uint64]bool{}
	k := shingleK
	if len(stokens) < k {
		k = len(stokens)
	}
	for i := 0; i+k <= len(stokens); i++ {
		want[patternHashL0("", stokens[i:i+k])] = true
	}
	for _, w := range l0ExtraWidths {
		if len(stokens) < w {
			continue
		}
		for i := 0; i+w <= len(stokens); i++ {
			want[patternHashL0(widthTag(w), stokens[i:i+w])] = true
		}
	}
	for _, p := range short.Patterns {
		if p.Level == LevelToken && !want[p.Hash] {
			t.Errorf("short stream emitted an unexpected L0 window (hash %x) — a clamped extra width?", p.Hash)
		}
	}
}

// The def-use flow edges, pinned on the idioms the pass exists for: a
// parameter flowing into a call, a call result deciding a condition (the
// errcheck idiom, via the tuple rule), a call result invoked on
// (open→close), and a parameter flowing straight to a return.
func TestPatternsDefUseFlowEdges(t *testing.T) {
	fp := buildFrom(t, `
func load(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readAll(f)
}`)
	for _, want := range []string{
		"flow:param→call:Open",      // path into os.Open
		"flow:call:Open→cond",       // err (tuple-bound to call:Open) in the if
		"flow:call:Open→call:Close", // f invoked on
		"flow:call:Open→call:readAll",
		"flow:call:Open→return", // err returned in the error branch
	} {
		if !hasRender(fp, LevelFlow, want) {
			t.Errorf("missing flow edge %q; have %v", want, renders(fp, LevelFlow))
		}
	}

	passthrough := buildFrom(t, `
func id(x int) int {
	return x
}`)
	if !hasRender(passthrough, LevelFlow, "flow:param→return") {
		t.Errorf("missing flow:param→return; have %v", renders(passthrough, LevelFlow))
	}

	// A value computed and dropped produces no edge: the whole point is that
	// it is now distinguishable from one that flows onward.
	dropped := buildFrom(t, `
func drop(x int) {
	y := compute(x)
	_ = y
}`)
	for _, r := range renders(dropped, LevelFlow) {
		if strings.HasPrefix(r, "flow:call:compute→") {
			t.Errorf("dropped value emitted an onward edge: %v", renders(dropped, LevelFlow))
		}
	}
}

// walkFixture exposes the walk() token stream for a fixture.
func walkFixture(t *testing.T, src string) ([]string, []int, []int, int) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fix.go", "package fix\n"+src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, decl := range file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			return walk(fd.Body)
		}
	}
	t.Fatal("no function in fixture")
	return nil, nil, nil, 0
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

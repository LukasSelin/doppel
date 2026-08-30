package canon

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// parseFunc parses a single function declaration from source text. The
// source is wrapped in a package clause so callers can write the function
// on its own.
func parseFunc(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", "package p\n"+src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			return fd
		}
	}
	t.Fatalf("no function declaration in %q", src)
	return nil
}

// canonOf canonicalizes source and returns the printed canonical form plus
// the rules that fired.
func canonOf(t *testing.T, src string) (string, []RuleID) {
	t.Helper()
	res := Canonicalize(parseFunc(t, src))
	if res.Capped {
		t.Fatalf("canonicalization hit its round bound on %q", src)
	}
	return Print(res.Decl), res.Fired
}

// fired reports whether id is in the list.
func fired(list []RuleID, id RuleID) bool {
	for _, r := range list {
		if r == id {
			return true
		}
	}
	return false
}

// TestRulesBeforeAfter is one before/after pair per rule: the "before" and
// "after" spellings must canonicalize to the same tree, and the rule under
// test must be among those that fired on the "before" side.
func TestRulesBeforeAfter(t *testing.T) {
	tests := []struct {
		name   string
		rule   RuleID
		before string
		after  string
	}{
		{
			name: "alpha-rename: parameters and locals become positional",
			rule: RuleAlphaRename,
			before: `func f(count int) int {
	total := count * 2
	return total
}`,
			after: `func f(n int) int {
	sum := n * 2
	return sum
}`,
		},
		{
			name: "alpha-rename: selector field names survive",
			rule: RuleAlphaRename,
			before: `func f(cfg Config) string {
	return cfg.Name
}`,
			after: `func f(other Config) string {
	return other.Name
}`,
		},
		{
			name: "unwrap-block: a bare block is spliced into its parent",
			rule: RuleUnwrapBlock,
			before: `func f() {
	a()
	{
		b()
		c()
	}
	d()
}`,
			after: `func f() {
	a()
	b()
	c()
	d()
}`,
		},
		{
			// Both branches return, so the guard that keeps an early
			// return in the then-branch does not apply and the swap goes
			// ahead, stripping the "!".
			name: "negated-if: the branches swap and the ! goes",
			rule: RuleNegatedIf,
			before: `func f(ok bool) int {
	if !ok {
		return 1
	} else {
		return 2
	}
}`,
			after: `func f(ok bool) int {
	if ok {
		return 2
	}
	return 1
}`,
		},
		{
			name: "negated-if: a non-leaving then-branch swaps against a leaving else",
			rule: RuleNegatedIf,
			before: `func f(ok bool) int {
	n := 0
	if !ok {
		n = 1
	} else {
		return 2
	}
	return n
}`,
			after: `func f(ok bool) int {
	n := 0
	if ok {
		return 2
	}
	n = 1
	return n
}`,
		},
		{
			// The second direction of the guard rule: the else is what
			// leaves, and the condition is not negated, so RuleNegatedIf
			// cannot reach it. Without this direction the two spellings
			// below would canonicalize differently forever.
			name: "guard-return: a leaving else negates the condition and swaps",
			rule: RuleGuardReturn,
			before: `func f(ok bool) int {
	if ok {
		work()
	} else {
		return 2
	}
	return 1
}`,
			after: `func f(ok bool) int {
	if !ok {
		return 2
	}
	work()
	return 1
}`,
		},
		{
			name: "guard-return: the else is lifted out",
			rule: RuleGuardReturn,
			before: `func f(err error) string {
	if err != nil {
		return "bad"
	} else {
		return "good"
	}
}`,
			after: `func f(err error) string {
	if err != nil {
		return "bad"
	}
	return "good"
}`,
		},
		{
			name: "guard-return: continue counts as leaving",
			rule: RuleGuardReturn,
			before: `func f(xs []int) {
	for _, x := range xs {
		if x == 0 {
			continue
		} else {
			use(x)
		}
	}
}`,
			after: `func f(xs []int) {
	for _, x := range xs {
		if x == 0 {
			continue
		}
		use(x)
	}
}`,
		},
		{
			name: "incdec: assignment folds to ++",
			rule: RuleIncDec,
			before: `func f() int {
	i := 0
	i = i + 1
	return i
}`,
			after: `func f() int {
	i := 0
	i++
	return i
}`,
		},
		{
			name: "incdec: += 1 and 1 + x fold to the same thing",
			rule: RuleIncDec,
			before: `func f() int {
	i := 0
	i += 1
	return i
}`,
			after: `func f() int {
	i := 0
	i = 1 + i
	return i
}`,
		},
		{
			name: "incdec: -= 1 folds to --",
			rule: RuleIncDec,
			before: `func f() int {
	i := 0
	i -= 1
	return i
}`,
			after: `func f() int {
	i := 0
	i = i - 1
	return i
}`,
		},
		{
			name: "commutative-sort: operand order stops mattering",
			rule: RuleCommutativeSort,
			before: `func f(a, b int) bool {
	return a+b == b*a
}`,
			after: `func f(a, b int) bool {
	return b+a == a*b
}`,
		},
		{
			name: "commutative-sort: && and == both order",
			rule: RuleCommutativeSort,
			before: `func f(p *T, q *T) bool {
	return q != nil && p != nil
}`,
			after: `func f(p *T, q *T) bool {
	return nil != p && nil != q
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBefore, firedBefore := canonOf(t, tt.before)
			gotAfter, _ := canonOf(t, tt.after)
			if gotBefore != gotAfter {
				t.Errorf("the two spellings did not converge\nbefore canonicalizes to:\n%s\n\nafter canonicalizes to:\n%s", gotBefore, gotAfter)
			}
			if !fired(firedBefore, tt.rule) {
				t.Errorf("rule %s did not fire on the before form; fired: %v", tt.rule, firedBefore)
			}
		})
	}
}

// TestRulesDeclineToFire pins the cases each rule must refuse, which are the
// cases where firing would be wrong rather than merely unhelpful.
func TestRulesDeclineToFire(t *testing.T) {
	tests := []struct {
		name string
		rule RuleID
		src  string
	}{
		{
			name: "negated-if leaves an else-if chain alone",
			rule: RuleNegatedIf,
			src: `func f(a, b bool) int {
	if !a {
		return 1
	} else if b {
		return 2
	}
	return 3
}`,
		},
		{
			name: "negated-if leaves a bare negated if alone",
			rule: RuleNegatedIf,
			src: `func f(a bool) int {
	if !a {
		return 1
	}
	return 2
}`,
		},
		{
			name: "guard-return leaves an if/else where neither branch leaves alone",
			rule: RuleGuardReturn,
			src: `func f(a bool) int {
	n := 0
	if a {
		n = 1
	} else {
		n = 2
	}
	return n
}`,
		},
		{
			// The guard that keeps the two branch rules from fighting: the
			// early-return form is already canonical, and swapping it would
			// put the guard where guard-return could not lift it out.
			name: "negated-if leaves an early return in the then-branch alone",
			rule: RuleNegatedIf,
			src: `func f(a bool) int {
	n := 0
	if !a {
		return 0
	} else {
		n = 1
	}
	return n
}`,
		},
		{
			name: "guard-return leaves an if with an init statement alone",
			rule: RuleGuardReturn,
			src: `func f() int {
	if v := get(); v > 0 {
		return v
	} else {
		return -v
	}
}`,
		},
		{
			name: "commutative-sort leaves ordering comparisons alone",
			rule: RuleCommutativeSort,
			src: `func f(a, b int) bool {
	return a < b
}`,
		},
		{
			name: "incdec leaves a step of 2 alone",
			rule: RuleIncDec,
			src: `func f() int {
	i := 0
	i = i + 2
	return i
}`,
		},
		{
			name: "incdec leaves 1 - x alone",
			rule: RuleIncDec,
			src: `func f() int {
	i := 0
	i = 1 - i
	return i
}`,
		},
		{
			name: "unwrap-block leaves an if body alone",
			rule: RuleUnwrapBlock,
			src: `func f(a bool) {
	if a {
		b()
	}
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, list := canonOf(t, tt.src)
			if fired(list, tt.rule) {
				t.Errorf("rule %s fired but should have declined; fired: %v", tt.rule, list)
			}
		})
	}
}

// TestAlphaRenameLeavesFreeIdentifiersAlone is the rule's central promise:
// free names are the shared vocabulary two functions are compared on, so
// they must survive. Bound names must not.
func TestAlphaRenameLeavesFreeIdentifiersAlone(t *testing.T) {
	got, _ := canonOf(t, `func f(w io.Writer, name string) error {
	msg := fmt.Sprintf("hello %s", name)
	_, err := w.Write([]byte(msg))
	return err
}`)

	for _, want := range []string{"io.Writer", "fmt.Sprintf", "Write", "byte", "error", "string"} {
		if !strings.Contains(got, want) {
			t.Errorf("free identifier %q did not survive:\n%s", want, got)
		}
	}
	// Word-boundary checks: "err" is a substring of the free name "error",
	// which must survive, so a bare Contains would pass for the wrong reason.
	for _, gone := range []string{"name", "msg", "err "} {
		if strings.Contains(got, gone) {
			t.Errorf("bound identifier %q survived:\n%s", gone, got)
		}
	}
	if strings.Contains(got, "return err\n") {
		t.Errorf("bound identifier \"err\" survived:\n%s", got)
	}
	// Four bindings: the two parameters, then msg and err.
	for _, want := range []string{"x0", "x1", "x2", "x3"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected positional name %q in:\n%s", want, got)
		}
	}
}

// TestAlphaRenameSkips pins the four positions the rule must not touch even
// when the name matches a binding.
func TestAlphaRenameSkips(t *testing.T) {
	got, _ := canonOf(t, `func f(x int) int {
	p := Point{x: x}
	if x > 0 {
		goto x
	}
x:
	return p.x
}`)
	// The parameter is renamed everywhere it is a value reference...
	if !strings.Contains(got, "x0") {
		t.Fatalf("parameter was not renamed:\n%s", got)
	}
	// ...but the composite-literal key, the selector, and the label are not.
	for _, want := range []string{"x: x0", "x1.x", "goto x", "x:"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q to survive renaming, in:\n%s", want, got)
		}
	}
}

// TestCanonicalizeDoesNotMutateInput is the invariant the whole clone file
// exists for: parser hands the same *ast.FuncDecl to fingerprint.Build and
// to the tagger's signal extractor, so an in-place rewrite here would move
// every score and every tag in the tool without touching a line of scoring
// code.
func TestCanonicalizeDoesNotMutateInput(t *testing.T) {
	srcs := []string{
		`func f(count int) int {
	if !ok(count) {
		return 0
	} else {
		count = count + 1
	}
	{
		log(count)
	}
	return count + zero
}`,
		`func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	for i, h := range s.handlers {
		if i == 0 {
			continue
		} else {
			h(w, r)
		}
	}
}`,
	}
	for _, src := range srcs {
		fd := parseFunc(t, src)
		before := Print(fd)
		res := Canonicalize(fd)
		after := Print(fd)
		if before != after {
			t.Errorf("Canonicalize mutated its input\nbefore:\n%s\nafter:\n%s", before, after)
		}
		if res.Decl == fd {
			t.Error("Canonicalize returned the input declaration rather than a copy")
		}
		if Print(res.Decl) == before {
			t.Errorf("nothing was canonicalized, so this case proves nothing:\n%s", before)
		}
	}
}

// TestCloneIsDeep walks the copy and the original together looking for a
// shared node. Any alias is a latent in-place mutation.
func TestCloneIsDeep(t *testing.T) {
	fd := parseFunc(t, `func f(a int) (b int) {
	var c = []int{1, 2}
	type t struct{ n int }
	switch v := any(a).(type) {
	case int:
		b = v
	}
	for i := range c {
		go func() { _ = i }()
	}
	select {
	case <-ch:
	default:
	}
	return b
}`)
	clone := Clone(fd)

	seen := make(map[ast.Node]bool)
	nodes := 0
	ast.Inspect(fd, func(n ast.Node) bool {
		if n != nil {
			seen[n] = true
			nodes++
		}
		return true
	})
	shared, cloned := 0, 0
	ast.Inspect(clone, func(n ast.Node) bool {
		if n != nil {
			cloned++
			if seen[n] {
				shared++
			}
		}
		return true
	})
	if shared != 0 {
		t.Errorf("clone shares %d of its %d nodes with the original", shared, cloned)
	}
	if cloned != nodes {
		t.Errorf("clone has %d nodes, original has %d — a node kind is missing from cloneExpr/cloneStmt", cloned, nodes)
	}

	// The clone cannot be compared to the original by printed text: the
	// original carries positions and go/printer honours them, so a struct
	// written on one line stays on one line there and is expanded here.
	// Round-tripping is the check that has content — the clone prints to
	// source that parses back to a declaration cloning to the same text.
	once := Print(clone)
	twice := Print(Clone(parseFunc(t, once)))
	if once != twice {
		t.Errorf("clone does not round-trip through source\nfirst:\n%s\nsecond:\n%s", once, twice)
	}
}

// TestClonePositionFree asserts the canonical tree carries no positions: a
// canonical form is a shape, and two identical shapes from different files
// must not differ through where they were written.
func TestClonePositionFree(t *testing.T) {
	clone := Clone(parseFunc(t, `func f(a int) int {
	if a > 0 {
		return a
	}
	return 0
}`))
	ast.Inspect(clone, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		// Ellipsis, GenDecl parens and TypeSpec assign are kept as
		// presence flags at position 1; nothing else may carry a position.
		if p := n.Pos(); p.IsValid() && p != 1 {
			t.Errorf("%T carries position %d", n, p)
		}
		return true
	})
}

// TestFiredIsDeclarationOrdered pins the contract on Result.Fired: rules in
// declaration order, deduplicated, regardless of which round caught them.
func TestFiredIsDeclarationOrdered(t *testing.T) {
	_, list := canonOf(t, `func f(count int) int {
	if !bad(count) {
		count = count + 1
	} else {
		return 0
	}
	{
		log(count)
	}
	return count + base
}`)
	order := make(map[RuleID]int)
	for i, r := range Rules() {
		order[r.ID] = i
	}
	if len(list) < 4 {
		t.Fatalf("expected several rules to fire, got %v", list)
	}
	for i := 1; i < len(list); i++ {
		if order[list[i-1]] >= order[list[i]] {
			t.Errorf("Fired is not in declaration order: %v", list)
		}
	}
}

// TestNoBodyIsSilent: a declaration without a body has nothing to
// canonicalize and must not produce a tree that later stages could mistake
// for one — the same "zero value means no body" convention fingerprint uses.
func TestNoBodyIsSilent(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", "package p\nfunc f()\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	res := Canonicalize(f.Decls[0].(*ast.FuncDecl))
	if res.Decl != nil || len(res.Fired) != 0 || res.Rounds != 0 {
		t.Errorf("expected a zero Result for a bodyless declaration, got %+v", res)
	}
	if r := Canonicalize(nil); r.Decl != nil {
		t.Errorf("expected a zero Result for a nil declaration, got %+v", r)
	}
}

// TestVersionIsSet guards the constant T6 will refuse to compare across.
func TestVersionIsSet(t *testing.T) {
	if Version == "" {
		t.Error("canon.Version must name the rule set")
	}
}

// TestRuleIDsAreUniqueAndNamed: the IDs are stable API — they get
// serialized and rendered — so a duplicate or an empty one is a defect in
// the table itself.
func TestRuleIDsAreUniqueAndNamed(t *testing.T) {
	seen := make(map[RuleID]bool)
	for _, r := range Rules() {
		if r.ID == "" {
			t.Error("a rule has no ID")
		}
		if r.Doc == "" {
			t.Errorf("rule %s has no Doc; the ID is rendered in explanations and needs one", r.ID)
		}
		if r.Apply == nil {
			t.Errorf("rule %s has no Apply", r.ID)
		}
		if seen[r.ID] {
			t.Errorf("duplicate rule ID %s", r.ID)
		}
		seen[r.ID] = true
	}
	if len(seen) != 6 {
		t.Errorf("expected 6 rules, got %d", len(seen))
	}
}

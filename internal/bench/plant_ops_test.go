package bench

import (
	"bytes"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/printer"
	"go/token"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
)

// This file is the plant harness's mutation half: go/ast rewrites that turn a
// donor function into a planted copy whose relation to the donor is known by
// construction. It is bench-only and deliberately never touches
// internal/canon — the canon-covered operators are written from canon's rule
// docs, so a plant canon fails to undo is a finding about canon, not an
// artefact of sharing its code.

// plantFamily groups operators by what the truth about a plant is.
type plantFamily string

const (
	// famCanon rewrites are undone by exactly one canon rule, so the plant's
	// canonical tree is the donor's: its WL bag must equal the donor's, and
	// where the rewrite leaves the written body's control-flow and nesting
	// histograms alone (plantOp.exact) its code-shape must equal the donor's
	// self-similarity. A failure is a canon bug.
	famCanon plantFamily = "canon-covered"
	// famEquiv rewrites preserve behaviour but no canon rule undoes them:
	// how far they read from the donor is the measured gap.
	famEquiv plantFamily = "equivalence"
	// famDrift rewrites change behaviour at a stated dose: the fork has aged.
	famDrift plantFamily = "drift"
)

var plantFamilies = []plantFamily{famCanon, famEquiv, famDrift}

// plantOp is one mutation operator. apply makes one edit and reports whether
// it found a site; a drift operator at dose d is apply run d times, and is not
// applicable to a donor where fewer than d edits succeed.
type plantOp struct {
	name   string
	family plantFamily
	// exact marks a canon-covered rewrite that leaves the written body's
	// control-flow and nesting histograms untouched — every one but
	// guard-else, whose lifted tail sits one level deeper as written. Those
	// histograms are read from the body as written, not the canonical tree,
	// so only an exact operator can be held to the donor's full code-shape.
	exact bool
	apply func(m *mutation) bool
}

// plantOps is the operator table, in the order every report lists them.
var plantOps = []plantOp{
	{"alpha-rename", famCanon, true, opAlphaRename},
	{"negated-if", famCanon, true, opNegatedIf},
	{"guard-else", famCanon, false, opGuardElse},
	{"incdec", famCanon, true, opIncDec},
	{"commutative-swap", famCanon, true, opCommutativeSwap},

	{"rename-in-type", famEquiv, false, opRenameInType},
	{"negated-if-paren", famEquiv, false, opNegatedIfParen},
	{"trailing-if-guard", famEquiv, false, opTrailingIfGuard},
	{"loop-if-continue", famEquiv, false, opLoopIfContinue},
	{"extract-local", famEquiv, false, opExtractLocal},
	{"split-define", famEquiv, false, opSplitDefine},
	{"var-decl", famEquiv, false, opVarDecl},
	{"else-after-negated-guard", famEquiv, false, opElseAfterNegatedGuard},

	{"insert-stmt", famDrift, false, opInsertStmt},
	{"swap-callee", famDrift, false, opSwapCallee},
	{"change-literal", famDrift, false, opChangeLiteral},
	{"drop-stmt", famDrift, false, opDropStmt},
}

// donorSource locates one donor declaration in its file. Every mutation
// re-parses the file, so each operator rewrites a tree of its own.
type donorSource struct {
	path string
	src  []byte
	line int    // the donor's StartLine: its func keyword
	name string // bare name, parser.MethodName
}

// parse parses the donor's file without comments and returns the donor's
// declaration. Comments are left out because the plant is printed from the
// tree, and a comment map would have to follow every rewrite.
func (d donorSource) parse() (*token.FileSet, *ast.File, *ast.FuncDecl, error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, d.path, d.src, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, decl := range f.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if ok && fd.Body != nil && fd.Name.Name == d.name && fset.Position(fd.Pos()).Line == d.line {
			return fset, f, fd, nil
		}
	}
	return nil, nil, nil, fmt.Errorf("%s:%d: no declaration of %s", d.path, d.line, d.name)
}

// header is everything in the donor's file up to the end of its imports:
// build constraints, package clause and import block. A plant file is this
// header plus the plant, so its calls into imported packages resolve exactly
// as the donor's do, and it builds for the same targets.
func header(fset *token.FileSet, f *ast.File, src []byte) []byte {
	end := f.Name.End()
	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			end = gd.End()
		}
	}
	return src[:fset.Position(end).Offset]
}

// mutation is one operator application's working state.
type mutation struct {
	fd    *ast.FuncDecl
	ds    donorSource
	twin  *ast.FuncDecl // see twinDecl
	file  *ast.File
	rng   *rand.Rand
	names map[string]bool // every identifier spelled in the file
	sib   *siblingTable
	pkg   string // the donor's package, for bare-call siblings
	dir   string // the donor's directory, which together with pkg is the Go package
	// spent marks nodes an earlier edit of this dose used — an inserted
	// statement, a swapped callee, a changed literal — so a dose of d is d
	// distinct edits and no node is placed in the tree twice.
	spent map[ast.Node]bool
}

// twinDecl is an independently parsed copy of the donor, a source of fresh
// nodes to insert. Parsed on first use: only insert-stmt needs one.
func (m *mutation) twinDecl() *ast.FuncDecl {
	if m.twin == nil {
		_, _, m.twin, _ = m.ds.parse() // the first parse of this source succeeded
	}
	return m.twin
}

// freshName returns an identifier spelled nowhere in the file.
func (m *mutation) freshName(prefix string) string {
	for i := 0; ; i++ {
		n := prefix + strconv.Itoa(i)
		if !m.names[n] {
			m.names[n] = true
			return n
		}
	}
}

// pick returns a uniformly drawn index into n candidates, or -1 for none.
func (m *mutation) pick(n int) int {
	if n == 0 {
		return -1
	}
	return m.rng.IntN(n)
}

// identNames collects every identifier spelled in a file.
func identNames(f *ast.File) map[string]bool {
	names := make(map[string]bool)
	ast.Inspect(f, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			names[id.Name] = true
		}
		return true
	})
	return names
}

// printDecl renders a declaration. A rewritten tree mixes original positions
// with zero ones, which go/printer lays out validly if not always prettily;
// the plant is re-parsed afterwards, so a rendering that did not round-trip
// would be caught there.
func printDecl(fset *token.FileSet, fd *ast.FuncDecl) ([]byte, error) {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, fd); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// stmtList is one statement list in the declaration — a block's, or a case
// or comm clause's — by pointer so an operator can splice it.
type stmtList struct{ list *[]ast.Stmt }

// stmtLists returns every statement list in fd, in source order.
func stmtLists(fd *ast.FuncDecl) []stmtList {
	var out []stmtList
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BlockStmt:
			out = append(out, stmtList{&x.List})
		case *ast.CaseClause:
			out = append(out, stmtList{&x.Body})
		case *ast.CommClause:
			out = append(out, stmtList{&x.Body})
		}
		return true
	})
	return out
}

// terminates reports whether a block's last statement leaves it for good —
// canon's guard rules' own condition, restated from their doc.
func terminates(b *ast.BlockStmt) bool {
	if b == nil || len(b.List) == 0 {
		return false
	}
	switch last := b.List[len(b.List)-1].(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BranchStmt:
		return last.Tok != token.FALLTHROUGH
	}
	return false
}

// simpleCond reports whether negating e needs no parentheses: the operand
// kinds canon's negate wraps bare.
func simpleCond(e ast.Expr) bool {
	switch e.(type) {
	case *ast.Ident, *ast.CallExpr, *ast.SelectorExpr, *ast.ParenExpr, *ast.IndexExpr:
		return true
	}
	return false
}

func isNot(e ast.Expr) bool {
	u, ok := e.(*ast.UnaryExpr)
	return ok && u.Op == token.NOT
}

// negated is how a person negates a condition: a leading "!" is dropped,
// a simple operand gains one, anything else is parenthesized first.
func negated(e ast.Expr) ast.Expr {
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.NOT {
		return u.X
	}
	if simpleCond(e) {
		return &ast.UnaryExpr{Op: token.NOT, X: e}
	}
	return &ast.UnaryExpr{Op: token.NOT, X: &ast.ParenExpr{X: e}}
}

// listedIfs is every if statement that is an element of a statement list —
// never an else-if, which sits in an else slot no list rule can reach.
type listedIf struct {
	sl  stmtList
	idx int
	ifs *ast.IfStmt
}

func listedIfs(fd *ast.FuncDecl) []listedIf {
	var out []listedIf
	for _, sl := range stmtLists(fd) {
		for i, s := range *sl.list {
			if ifs, ok := s.(*ast.IfStmt); ok {
				out = append(out, listedIf{sl, i, ifs})
			}
		}
	}
	return out
}

// ---- canon-covered ----

// opAlphaRename renames every identifier the function binds — receiver,
// parameters, results, locals, and those of its function literals — to a
// fresh name, consistently. canon's alpha-rename maps both spellings onto x0,
// x1, … in binding order. Names inside a type literal are left alone (see
// ownFields); opRenameInType renames those.
func opAlphaRename(m *mutation) bool {
	own := ownFields(m.fd)
	return renameBound(m, func(decl ast.Node) bool {
		f, isField := decl.(*ast.Field)
		return !isField || own[f]
	})
}

// opRenameInType renames the parameter names inside a function-typed
// parameter or result of the signature: `n func(f *flag.FlagSet)` becomes
// `n func(v0 *flag.FlagSet)`. Nothing binds those names at run time, but the
// signature component reads the type as written, names included, so the
// rename reads as a different signature.
func opRenameInType(m *mutation) bool {
	inType := make(map[*ast.Field]bool)
	for _, fl := range []*ast.FieldList{m.fd.Type.Params, m.fd.Type.Results} {
		if fl == nil {
			continue
		}
		for _, field := range fl.List {
			ast.Inspect(field.Type, func(n ast.Node) bool {
				if ft, ok := n.(*ast.FuncType); ok {
					for _, l := range []*ast.FieldList{ft.Params, ft.Results} {
						if l != nil {
							for _, f := range l.List {
								inType[f] = true
							}
						}
					}
				}
				return true
			})
		}
	}
	return renameBound(m, func(decl ast.Node) bool {
		f, isField := decl.(*ast.Field)
		return isField && inType[f]
	})
}

// ownFields are the fields whose names the function itself binds: its
// receiver, parameters and results, and those of every function literal in
// its body. A name declared in any other field list sits inside a type
// literal, where it binds nothing.
func ownFields(fd *ast.FuncDecl) map[*ast.Field]bool {
	own := make(map[*ast.Field]bool)
	add := func(fls ...*ast.FieldList) {
		for _, fl := range fls {
			if fl != nil {
				for _, f := range fl.List {
					own[f] = true
				}
			}
		}
	}
	add(fd.Recv, fd.Type.Params, fd.Type.Results)
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if lit, ok := n.(*ast.FuncLit); ok {
			add(lit.Type.Params, lit.Type.Results)
		}
		return true
	})
	return own
}

// renameBound renames, consistently and to fresh names, every variable
// declared inside fd whose declaring node keep admits. Composite-literal
// keys are skipped: go/parser resolves a key to a local of the same name,
// but a key is a field name, not a use of the local.
func renameBound(m *mutation, keep func(decl ast.Node) bool) bool {
	keys := make(map[*ast.Ident]bool)
	ast.Inspect(m.fd, func(n ast.Node) bool {
		if cl, ok := n.(*ast.CompositeLit); ok {
			for _, e := range cl.Elts {
				if kv, ok := e.(*ast.KeyValueExpr); ok {
					if id, ok := kv.Key.(*ast.Ident); ok {
						keys[id] = true
					}
				}
			}
		}
		return true
	})
	lo, hi := m.fd.Pos(), m.fd.End()
	renamed := make(map[*ast.Object]string)
	var order []*ast.Object
	ast.Inspect(m.fd, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok || id.Obj == nil || id.Obj.Kind != ast.Var || keys[id] || id.Name == "_" {
			return true
		}
		decl, ok := id.Obj.Decl.(ast.Node)
		if !ok || decl.Pos() < lo || decl.Pos() >= hi || !keep(decl) {
			return true
		}
		if _, seen := renamed[id.Obj]; !seen {
			renamed[id.Obj] = ""
			order = append(order, id.Obj)
		}
		return true
	})
	if len(order) == 0 {
		return false
	}
	for _, obj := range order {
		renamed[obj] = m.freshName("v")
	}
	ast.Inspect(m.fd, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Obj != nil && !keys[id] {
			if name := renamed[id.Obj]; name != "" {
				id.Name = name
			}
		}
		return true
	})
	return true
}

// opNegatedIf rewrites `if c {A} else {B}` into `if !c {B} else {A}`, which
// canon's negated-if (or, when only B leaves, its guard-return) maps back.
// Only a listed if with no init, a plain else block and a condition that
// negates without parentheses: an init or an else-if slot leaves the guard
// rule nothing to lift, and a parenthesized condition keeps its paren node
// after canon strips the "!".
func opNegatedIf(m *mutation) bool {
	var sites []*ast.IfStmt
	for _, li := range listedIfs(m.fd) {
		ifs := li.ifs
		if _, plain := ifs.Else.(*ast.BlockStmt); !plain || ifs.Init != nil {
			continue
		}
		if !simpleCond(ifs.Cond) || isNot(ifs.Cond) {
			continue
		}
		sites = append(sites, ifs)
	}
	i := m.pick(len(sites))
	if i < 0 {
		return false
	}
	ifs := sites[i]
	alt := ifs.Else.(*ast.BlockStmt)
	ifs.Cond = &ast.UnaryExpr{Op: token.NOT, X: ifs.Cond}
	ifs.Body, ifs.Else = alt, ifs.Body
	return true
}

// opNegatedIfParen is negated-if where the condition needs parentheses to be
// negated: `if a == b {A} else {B}` becomes `if !(a == b) {B} else {A}`.
// canon's negated-if strips the "!" and swaps back, but the parenthesis node
// stays in the canonical tree, which the donor's has no counterpart for.
func opNegatedIfParen(m *mutation) bool {
	var sites []*ast.IfStmt
	for _, li := range listedIfs(m.fd) {
		ifs := li.ifs
		if _, plain := ifs.Else.(*ast.BlockStmt); !plain || ifs.Init != nil || simpleCond(ifs.Cond) || isNot(ifs.Cond) {
			continue
		}
		sites = append(sites, ifs)
	}
	i := m.pick(len(sites))
	if i < 0 {
		return false
	}
	ifs := sites[i]
	alt := ifs.Else.(*ast.BlockStmt)
	ifs.Cond = &ast.UnaryExpr{Op: token.NOT, X: &ast.ParenExpr{X: ifs.Cond}}
	ifs.Body, ifs.Else = alt, ifs.Body
	return true
}

// guardSite is an early return followed by more statements: the shape
// guard-else wraps into an else.
func guardSites(fd *ast.FuncDecl, negatedTerminating bool) []listedIf {
	var out []listedIf
	for _, li := range listedIfs(fd) {
		ifs := li.ifs
		if ifs.Else != nil || ifs.Init != nil || !terminates(ifs.Body) || li.idx == len(*li.sl.list)-1 {
			continue
		}
		tail := &ast.BlockStmt{List: (*li.sl.list)[li.idx+1:]}
		// canon's negated-if swaps `if !c {return} else {tail}` when the tail
		// leaves too, and the result is a different canonical tree from the
		// early-return form — the case guard-else excludes and
		// else-after-negated-guard measures.
		gap := isNot(ifs.Cond) && terminates(tail)
		if gap == negatedTerminating {
			out = append(out, li)
		}
	}
	return out
}

// wrapTail moves everything after the guard into its else block.
func wrapTail(li listedIf) {
	list := *li.sl.list
	tail := append([]ast.Stmt(nil), list[li.idx+1:]...)
	li.ifs.Else = &ast.BlockStmt{List: tail}
	*li.sl.list = list[:li.idx+1]
}

// opGuardElse rewrites `if c {…return}; rest` into
// `if c {…return} else {rest}`, which canon's guard-return lifts back.
func opGuardElse(m *mutation) bool {
	sites := guardSites(m.fd, false)
	i := m.pick(len(sites))
	if i < 0 {
		return false
	}
	wrapTail(sites[i])
	return true
}

// opIncDec swaps `x++` and `x += 1` (and the -- forms), in either
// direction; canon's incdec folds both to the increment.
func opIncDec(m *mutation) bool {
	type site struct{ p *ast.Stmt }
	var sites []site
	visit := func(p *ast.Stmt) {
		if p == nil || *p == nil {
			return
		}
		switch s := (*p).(type) {
		case *ast.IncDecStmt:
			sites = append(sites, site{p})
		case *ast.AssignStmt:
			if (s.Tok == token.ADD_ASSIGN || s.Tok == token.SUB_ASSIGN) && len(s.Lhs) == 1 && len(s.Rhs) == 1 {
				if lit, ok := s.Rhs[0].(*ast.BasicLit); ok && lit.Kind == token.INT && lit.Value == "1" {
					sites = append(sites, site{p})
				}
			}
		}
	}
	ast.Inspect(m.fd.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BlockStmt:
			for i := range x.List {
				visit(&x.List[i])
			}
		case *ast.CaseClause:
			for i := range x.Body {
				visit(&x.Body[i])
			}
		case *ast.CommClause:
			for i := range x.Body {
				visit(&x.Body[i])
			}
		case *ast.IfStmt:
			visit(&x.Init)
		case *ast.ForStmt:
			visit(&x.Init)
			visit(&x.Post)
		case *ast.SwitchStmt:
			visit(&x.Init)
		case *ast.TypeSwitchStmt:
			visit(&x.Init)
		case *ast.LabeledStmt:
			visit(&x.Stmt)
		}
		return true
	})
	i := m.pick(len(sites))
	if i < 0 {
		return false
	}
	p := sites[i].p
	switch s := (*p).(type) {
	case *ast.IncDecStmt:
		tok := token.ADD_ASSIGN
		if s.Tok == token.DEC {
			tok = token.SUB_ASSIGN
		}
		*p = &ast.AssignStmt{Lhs: []ast.Expr{s.X}, Tok: tok, Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "1"}}}
	case *ast.AssignStmt:
		tok := token.INC
		if s.Tok == token.SUB_ASSIGN {
			tok = token.DEC
		}
		*p = &ast.IncDecStmt{X: s.Lhs[0], Tok: tok}
	}
	return true
}

// commutative are the operators canon's commutative-sort orders.
var commutative = map[token.Token]bool{
	token.ADD: true, token.MUL: true, token.EQL: true, token.NEQ: true,
	token.AND: true, token.OR: true, token.XOR: true, token.LAND: true, token.LOR: true,
}

// opCommutativeSwap swaps the operands of one commutative binary expression.
// Only where neither operand is itself binary: the printer does not insert
// parentheses, so swapping `a - b + c` would print a different tree.
func opCommutativeSwap(m *mutation) bool {
	var sites []*ast.BinaryExpr
	ast.Inspect(m.fd.Body, func(n ast.Node) bool {
		b, ok := n.(*ast.BinaryExpr)
		if !ok || !commutative[b.Op] {
			return true
		}
		_, xb := b.X.(*ast.BinaryExpr)
		_, yb := b.Y.(*ast.BinaryExpr)
		if !xb && !yb && render(b.X) != render(b.Y) {
			sites = append(sites, b)
		}
		return true
	})
	i := m.pick(len(sites))
	if i < 0 {
		return false
	}
	sites[i].X, sites[i].Y = sites[i].Y, sites[i].X
	return true
}

// render prints an expression without positions, for comparisons.
func render(e ast.Expr) string {
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), e)
	return buf.String()
}

// ---- equivalence-preserving, not covered by canon ----

// opTrailingIfGuard rewrites a void function's final `if c { B }` into
// `if !c { return }; B` — the known gap canon has no rule for.
func opTrailingIfGuard(m *mutation) bool {
	if m.fd.Type.Results != nil && len(m.fd.Type.Results.List) > 0 {
		return false
	}
	body := &m.fd.Body.List
	n := len(*body)
	if n == 0 {
		return false
	}
	ifs, ok := (*body)[n-1].(*ast.IfStmt)
	if !ok || ifs.Else != nil || ifs.Init != nil || len(ifs.Body.List) == 0 {
		return false
	}
	guard := &ast.IfStmt{Cond: negated(ifs.Cond), Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{}}}}
	*body = append(append((*body)[:n-1:n-1], guard), ifs.Body.List...)
	return true
}

// opLoopIfContinue rewrites a loop body's final `if c { B }` into
// `if !c { continue }; B`.
func opLoopIfContinue(m *mutation) bool {
	var sites []*ast.BlockStmt
	ast.Inspect(m.fd.Body, func(n ast.Node) bool {
		var body *ast.BlockStmt
		switch x := n.(type) {
		case *ast.ForStmt:
			body = x.Body
		case *ast.RangeStmt:
			body = x.Body
		}
		if body != nil && len(body.List) > 0 {
			if ifs, ok := body.List[len(body.List)-1].(*ast.IfStmt); ok && ifs.Else == nil && ifs.Init == nil && len(ifs.Body.List) > 0 {
				sites = append(sites, body)
			}
		}
		return true
	})
	i := m.pick(len(sites))
	if i < 0 {
		return false
	}
	body := sites[i]
	n := len(body.List)
	ifs := body.List[n-1].(*ast.IfStmt)
	guard := &ast.IfStmt{Cond: negated(ifs.Cond), Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.CONTINUE}}}}
	body.List = append(append(body.List[:n-1:n-1], guard), ifs.Body.List...)
	return true
}

// hasCall reports whether an expression contains a call.
func hasCall(n ast.Node) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		if _, ok := n.(*ast.CallExpr); ok {
			found = true
		}
		return !found
	})
	return found
}

// opExtractLocal hoists a call argument into a local declared just before the
// statement: `f(a, g(x))` becomes `v := g(x); f(a, v)`. Only the first call
// among the arguments of a statement-level call with at least two arguments
// (so the hoisted call is single-valued), and only when nothing evaluated
// before it in the call contains a call (so evaluation order is unchanged).
func opExtractLocal(m *mutation) bool {
	type site struct {
		sl   stmtList
		idx  int
		call *ast.CallExpr
		arg  int
	}
	var sites []site
	for _, sl := range stmtLists(m.fd) {
		for i, s := range *sl.list {
			var call *ast.CallExpr
			switch x := s.(type) {
			case *ast.ExprStmt:
				call, _ = x.X.(*ast.CallExpr)
			case *ast.ReturnStmt:
				if len(x.Results) == 1 {
					call, _ = x.Results[0].(*ast.CallExpr)
				}
			case *ast.AssignStmt:
				if len(x.Rhs) == 1 {
					call, _ = x.Rhs[0].(*ast.CallExpr)
				}
			}
			if call == nil || len(call.Args) < 2 || call.Ellipsis.IsValid() || hasCall(call.Fun) {
				continue
			}
			for a, arg := range call.Args {
				if _, ok := arg.(*ast.CallExpr); ok {
					sites = append(sites, site{sl, i, call, a})
					break
				}
				if hasCall(arg) {
					break
				}
			}
		}
	}
	k := m.pick(len(sites))
	if k < 0 {
		return false
	}
	s := sites[k]
	name := m.freshName("t")
	decl := &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(name)}, Tok: token.DEFINE, Rhs: []ast.Expr{s.call.Args[s.arg]}}
	s.call.Args[s.arg] = ast.NewIdent(name)
	list := *s.sl.list
	out := make([]ast.Stmt, 0, len(list)+1)
	out = append(out, list[:s.idx]...)
	out = append(out, decl)
	out = append(out, list[s.idx:]...)
	*s.sl.list = out
	return true
}

// opSplitDefine splits `a, b := x, y` into `a := x; b := y`, where no
// right-hand side mentions a name the statement declares.
func opSplitDefine(m *mutation) bool {
	type site struct {
		sl  stmtList
		idx int
	}
	var sites []site
	for _, sl := range stmtLists(m.fd) {
		for i, s := range *sl.list {
			as, ok := s.(*ast.AssignStmt)
			if !ok || as.Tok != token.DEFINE || len(as.Lhs) < 2 || len(as.Lhs) != len(as.Rhs) {
				continue
			}
			lhs := make(map[string]bool)
			allIdents := true
			for _, l := range as.Lhs {
				id, ok := l.(*ast.Ident)
				if !ok {
					allIdents = false
					break
				}
				lhs[id.Name] = true
			}
			if !allIdents {
				continue
			}
			clean := true
			for _, r := range as.Rhs {
				ast.Inspect(r, func(n ast.Node) bool {
					if id, ok := n.(*ast.Ident); ok && lhs[id.Name] {
						clean = false
					}
					return clean
				})
			}
			if clean {
				sites = append(sites, site{sl, i})
			}
		}
	}
	k := m.pick(len(sites))
	if k < 0 {
		return false
	}
	s := sites[k]
	list := *s.sl.list
	as := list[s.idx].(*ast.AssignStmt)
	out := make([]ast.Stmt, 0, len(list)+len(as.Lhs)-1)
	out = append(out, list[:s.idx]...)
	for j := range as.Lhs {
		out = append(out, &ast.AssignStmt{Lhs: []ast.Expr{as.Lhs[j]}, Tok: token.DEFINE, Rhs: []ast.Expr{as.Rhs[j]}})
	}
	out = append(out, list[s.idx+1:]...)
	*s.sl.list = out
	return true
}

// opVarDecl rewrites `x := v` as `var x = v`.
func opVarDecl(m *mutation) bool {
	type site struct {
		sl  stmtList
		idx int
	}
	var sites []site
	for _, sl := range stmtLists(m.fd) {
		for i, s := range *sl.list {
			as, ok := s.(*ast.AssignStmt)
			if !ok || as.Tok != token.DEFINE || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
				continue
			}
			if id, ok := as.Lhs[0].(*ast.Ident); ok && id.Name != "_" {
				sites = append(sites, site{sl, i})
			}
		}
	}
	k := m.pick(len(sites))
	if k < 0 {
		return false
	}
	s := sites[k]
	as := (*s.sl.list)[s.idx].(*ast.AssignStmt)
	(*s.sl.list)[s.idx] = &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{
		&ast.ValueSpec{Names: []*ast.Ident{as.Lhs[0].(*ast.Ident)}, Values: as.Rhs},
	}}}
	return true
}

// opElseAfterNegatedGuard is guard-else at the sites it excludes: a negated
// early return whose tail also leaves. canon's negated-if swaps that if/else
// before guard-return can lift it, so the two spellings settle on different
// canonical trees. It sits in the equivalence family because no canon rule
// undoes it today — if one comes to, it moves to the canon family.
func opElseAfterNegatedGuard(m *mutation) bool {
	sites := guardSites(m.fd, true)
	i := m.pick(len(sites))
	if i < 0 {
		return false
	}
	wrapTail(sites[i])
	return true
}

// ---- drift ----

// opInsertStmt inserts a copy of one of the function's own simple statements
// — a call, an assignment, an increment — before another statement. Drawn
// from the twin tree, so the copy shares no node with the original.
func opInsertStmt(m *mutation) bool {
	var pool []ast.Stmt
	ast.Inspect(m.twinDecl().Body, func(n ast.Node) bool {
		if m.spent[n] {
			return true
		}
		switch s := n.(type) {
		case *ast.ExprStmt:
			if _, ok := s.X.(*ast.CallExpr); ok {
				pool = append(pool, s)
			}
		case *ast.AssignStmt:
			if s.Tok != token.DEFINE {
				pool = append(pool, s)
			}
		case *ast.IncDecStmt:
			pool = append(pool, s)
		}
		return true
	})
	p := m.pick(len(pool))
	if p < 0 {
		return false
	}
	lists := stmtLists(m.fd)
	var slots []stmtList
	for _, sl := range lists {
		if len(*sl.list) > 0 {
			slots = append(slots, sl)
		}
	}
	s := m.pick(len(slots))
	if s < 0 {
		return false
	}
	sl := slots[s]
	at := m.pick(len(*sl.list))
	list := *sl.list
	out := make([]ast.Stmt, 0, len(list)+1)
	out = append(out, list[:at]...)
	out = append(out, pool[p])
	out = append(out, list[at:]...)
	*sl.list = out
	m.spent[pool[p]] = true
	return true
}

// siblingTable is the corpus-derived answer to "what else could this call
// have called": for a bare call, another function of the same package; for
// an import-qualified call, another name called through the same qualifier
// anywhere in the host.
type siblingTable struct {
	byPackage   map[string][]string // package+"\x00"+dir -> plain function names, sorted
	byQualifier map[string][]string // qualifier -> selected names, sorted
}

// opSwapCallee replaces one call's target with a sibling.
func opSwapCallee(m *mutation) bool {
	imports := make(map[string]bool)
	for _, im := range m.file.Imports {
		path, _ := strconv.Unquote(im.Path.Value)
		name := path[strings.LastIndex(path, "/")+1:]
		if im.Name != nil {
			name = im.Name.Name
		}
		imports[name] = true
	}
	type site struct {
		id   *ast.Ident
		alts []string
	}
	var sites []site
	ast.Inspect(m.fd.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		var id *ast.Ident
		var pool []string
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			id, pool = fn, m.sib.byPackage[m.pkg+"\x00"+m.dir]
		case *ast.SelectorExpr:
			if x, ok := fn.X.(*ast.Ident); ok && imports[x.Name] {
				id, pool = fn.Sel, m.sib.byQualifier[x.Name]
			}
		}
		if id == nil || m.spent[id] {
			return true
		}
		var alts []string
		for _, name := range pool {
			if name != id.Name {
				alts = append(alts, name)
			}
		}
		if len(alts) > 0 {
			sites = append(sites, site{id, alts})
		}
		return true
	})
	k := m.pick(len(sites))
	if k < 0 {
		return false
	}
	s := sites[k]
	s.id.Name = s.alts[m.pick(len(s.alts))]
	m.spent[s.id] = true
	return true
}

// opChangeLiteral edits one literal: a separator inside a string becomes a
// different separator, any other string gains a character, an integer moves
// by one.
func opChangeLiteral(m *mutation) bool {
	var sites []*ast.BasicLit
	ast.Inspect(m.fd.Body, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || m.spent[lit] {
			return true
		}
		switch lit.Kind {
		case token.STRING:
			if len(lit.Value) >= 2 {
				sites = append(sites, lit)
			}
		case token.INT:
			if _, err := strconv.ParseInt(lit.Value, 10, 64); err == nil {
				sites = append(sites, lit)
			}
		}
		return true
	})
	k := m.pick(len(sites))
	if k < 0 {
		return false
	}
	lit := sites[k]
	switch lit.Kind {
	case token.INT:
		v, _ := strconv.ParseInt(lit.Value, 10, 64)
		lit.Value = strconv.FormatInt(v+1, 10)
	case token.STRING:
		lit.Value = changeString(lit.Value)
	}
	m.spent[lit] = true
	return true
}

// separators are the characters a separator edit swaps between.
const separators = ",;:/.-_ |"

// changeString edits a quoted Go string literal and keeps it valid: the first
// separator becomes the next one in separators, or, with none present, an
// "x" goes in before the closing quote.
func changeString(q string) string {
	body := q[1 : len(q)-1]
	for i := 0; i < len(body); i++ {
		if j := strings.IndexByte(separators, body[i]); j >= 0 {
			next := separators[(j+1)%len(separators)]
			if q[0] == '`' && next == '`' {
				continue
			}
			return q[:1] + body[:i] + string(next) + body[i+1:] + q[len(q)-1:]
		}
	}
	return q[:len(q)-1] + "x" + q[len(q)-1:]
}

// simpleStmt reports whether a statement is one line of work with no
// statements nested in it — what one drift edit adds or removes.
func simpleStmt(s ast.Stmt) bool {
	switch s.(type) {
	case *ast.ExprStmt, *ast.AssignStmt, *ast.IncDecStmt, *ast.DeclStmt, *ast.SendStmt, *ast.DeferStmt, *ast.GoStmt:
		return true
	}
	return false
}

// opDropStmt removes one simple statement from a list that keeps at least one
// statement. Never a compound one: dropping an if would take its whole body
// with it, which is many edits counted as one.
func opDropStmt(m *mutation) bool {
	type site struct {
		sl  stmtList
		idx int
	}
	var sites []site
	for _, sl := range stmtLists(m.fd) {
		if len(*sl.list) < 2 {
			continue
		}
		for i, s := range *sl.list {
			if simpleStmt(s) {
				sites = append(sites, site{sl, i})
			}
		}
	}
	k := m.pick(len(sites))
	if k < 0 {
		return false
	}
	s := sites[k]
	list := *s.sl.list
	*s.sl.list = append(list[:s.idx:s.idx], list[s.idx+1:]...)
	return true
}

// buildSiblings derives the swap-callee table from the host's own calls.
func buildSiblings(h *plantHost) *siblingTable {
	t := &siblingTable{byPackage: map[string][]string{}, byQualifier: map[string][]string{}}
	pkgSet := map[string]map[string]bool{}
	qualSet := map[string]map[string]bool{}
	add := func(m map[string]map[string]bool, k, v string) {
		if m[k] == nil {
			m[k] = map[string]bool{}
		}
		m[k][v] = true
	}
	for _, u := range h.units {
		if u.ReceiverType == "" {
			add(pkgSet, u.Package+"\x00"+dirOf(u.File), u.Name)
		}
		for _, c := range u.Callees {
			if dot := strings.IndexByte(c, '.'); dot > 0 {
				add(qualSet, c[:dot], c[dot+1:])
			}
		}
	}
	flatten := func(src map[string]map[string]bool, dst map[string][]string) {
		for k, set := range src {
			names := make([]string, 0, len(set))
			for n := range set {
				names = append(names, n)
			}
			sort.Strings(names)
			dst[k] = names
		}
	}
	flatten(pkgSet, t.byPackage)
	flatten(qualSet, t.byQualifier)
	return t
}

package moves

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
)

// Graph is the package-level symbols of the graph packages and the
// references between them.
type Graph struct {
	Fset *token.FileSet
	// Pkgs is every loaded package by import path, graph or not: the import
	// cycle check walks the whole main module.
	Pkgs map[string]*Package
	// Syms is every symbol in declaration order: packages by import path,
	// then position. Every iteration that decides an output goes through it.
	Syms []*Sym

	byObj      map[types.Object]*Sym
	fieldOwner map[*types.Var]*Sym
	dispatched map[dispatchKey][]*Sym
	concrete   []*Sym // non-interface named types, the dispatch candidates
}

// Sym is one package-level declaration: a function, method, type, var or
// const.
type Sym struct {
	Obj     types.Object
	Pkg     *Package
	Owner   *Sym   // the receiver's type for a method; nil otherwise
	Methods []*Sym // a type's methods
	In      []Use
	Out     []Use

	generated bool // declared in a generated file
	body      bool // a function or method declared with a body
}

// Name is the symbol's name in doppel's convention: "Parse", or for a
// method its receiver type with the star kept, "*Indexer.Visit".
func (s *Sym) Name() string {
	fn, ok := s.Obj.(*types.Func)
	if !ok || s.Owner == nil {
		return s.Obj.Name()
	}
	recv := s.Owner.Obj.Name()
	if _, ptr := fn.Signature().Recv().Type().(*types.Pointer); ptr {
		recv = "*" + recv
	}
	return recv + "." + s.Obj.Name()
}

// Qualified is Name prefixed with the package name, as CodeUnits are keyed.
func (s *Sym) Qualified() string { return s.Pkg.Name + "." + s.Name() }

// Use is one reference from a declaration to a symbol. A declaration
// referring to a symbol several times is one Use: shares count referring
// declarations, not reference sites.
type Use struct {
	From, To *Sym
	// Obj is the object named: To.Obj, or a field of To's struct type.
	Obj types.Object
	// Dynamic marks a call through an interface that may reach To, by class
	// hierarchy analysis. It needs neither an import nor access to To, so it
	// never blocks a move and never makes a package a function's user.
	Dynamic bool
}

// Build builds the graph of the packages marked InGraph. The others are kept
// only for their imports.
func Build(fset *token.FileSet, pkgs []*Package) *Graph {
	g := &Graph{
		Fset:       fset,
		Pkgs:       map[string]*Package{},
		byObj:      map[types.Object]*Sym{},
		fieldOwner: map[*types.Var]*Sym{},
		dispatched: map[dispatchKey][]*Sym{},
	}
	var graphed []*Package
	for _, p := range pkgs {
		g.Pkgs[p.Path] = p
		if p.InGraph && p.Types != nil {
			graphed = append(graphed, p)
		}
	}
	sort.Slice(graphed, func(i, j int) bool { return graphed[i].Path < graphed[j].Path })
	for _, p := range graphed {
		g.declare(p)
	}
	for _, p := range graphed {
		for _, f := range p.Files {
			for _, d := range f.Decls {
				g.walkDecl(p, f, d)
			}
		}
	}
	sort.SliceStable(g.Syms, func(i, j int) bool { return symLess(g.Syms[i], g.Syms[j]) })
	for _, s := range g.Syms {
		sortUses(s.In, func(u Use) *Sym { return u.From })
		sortUses(s.Out, func(u Use) *Sym { return u.To })
	}
	return g
}

// declare adds a symbol for each package-level object of p and each method
// of its named types, and records which type owns each struct field.
func (g *Graph) declare(p *Package) {
	scope := p.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		s := g.add(obj, p, nil)
		tn, ok := obj.(*types.TypeName)
		if !ok || tn.IsAlias() {
			continue
		}
		named, ok := tn.Type().(*types.Named)
		if !ok {
			continue
		}
		for i := 0; i < named.NumMethods(); i++ {
			g.add(named.Method(i), p, s)
		}
		switch u := named.Underlying().(type) {
		case *types.Interface:
			for i := 0; i < u.NumExplicitMethods(); i++ {
				g.add(u.ExplicitMethod(i), p, s)
			}
		case *types.Struct:
			g.ownFields(u, s)
		}
		if _, iface := named.Underlying().(*types.Interface); !iface && named.TypeParams().Len() == 0 {
			g.concrete = append(g.concrete, s)
		}
	}
}

// ownFields records owner as the owner of st's fields, and of the fields of
// any struct literal types nested in them.
func (g *Graph) ownFields(st *types.Struct, owner *Sym) {
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		g.fieldOwner[f] = owner
		if inner, ok := f.Type().(*types.Struct); ok {
			g.ownFields(inner, owner)
		}
	}
}

func (g *Graph) add(obj types.Object, p *Package, owner *Sym) *Sym {
	if s := g.byObj[obj]; s != nil {
		return s
	}
	s := &Sym{Obj: obj, Pkg: p, Owner: owner}
	g.byObj[obj] = s
	g.Syms = append(g.Syms, s)
	if owner != nil {
		owner.Methods = append(owner.Methods, s)
	}
	return s
}

// walkDecl records what one top-level declaration refers to. Each name of a
// var or const spec is charged with everything the whole spec refers to.
//
// init functions and blank package-level names are not in the package scope,
// so they are added here: what they refer to is still a reference.
func (g *Graph) walkDecl(p *Package, f *ast.File, d ast.Decl) {
	gen := p.generated[f]
	switch d := d.(type) {
	case *ast.FuncDecl:
		if obj := p.Info.Defs[d.Name]; obj != nil {
			from := g.add(obj, p, g.recvOwner(obj))
			from.generated = gen
			from.body = d.Body != nil
			g.walk(p, from, d)
		}
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			switch sp := spec.(type) {
			case *ast.TypeSpec:
				if s := g.byObj[p.Info.Defs[sp.Name]]; s != nil {
					s.generated = gen
					g.walk(p, s, sp)
				}
			case *ast.ValueSpec:
				for _, name := range sp.Names {
					if obj := p.Info.Defs[name]; obj != nil {
						s := g.add(obj, p, nil)
						s.generated = gen
						g.walk(p, s, sp)
					}
				}
			}
		}
	}
}

// recvOwner is the symbol of a method's receiver type, so that a method the
// scope walk did not reach (one declared on an alias) still has its owner.
func (g *Graph) recvOwner(obj types.Object) *Sym {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Signature().Recv() == nil {
		return nil
	}
	t := fn.Signature().Recv().Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return g.byObj[named.Origin().Obj()]
	}
	return nil
}

func (g *Graph) walk(p *Package, from *Sym, root ast.Node) {
	ast.Inspect(root, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			if obj := p.Info.Uses[n]; obj != nil {
				g.use(from, obj)
			}
		case *ast.SelectorExpr:
			sel := p.Info.Selections[n]
			if sel == nil {
				break
			}
			// Promoted fields and methods reach through embedded fields that
			// the source never names.
			for _, f := range embeddedPath(sel) {
				g.use(from, f)
			}
			if sel.Kind() == types.MethodVal {
				for _, to := range g.dispatch(sel) {
					if to != from {
						g.link(Use{From: from, To: to, Obj: to.Obj, Dynamic: true})
					}
				}
			}
		}
		return true
	})
}

// use records that from refers to obj, if obj is a field or a symbol of one
// of the graph's packages.
func (g *Graph) use(from *Sym, obj types.Object) {
	var to *Sym
	switch o := obj.(type) {
	case *types.Var:
		if o.IsField() {
			obj = o.Origin()
			to = g.fieldOwner[obj.(*types.Var)]
		} else {
			to = g.byObj[o.Origin()]
		}
	case *types.Func:
		obj = o.Origin()
		to = g.byObj[obj]
	default:
		to = g.byObj[obj]
	}
	if to == nil || to == from {
		return
	}
	g.link(Use{From: from, To: to, Obj: obj})
}

func (g *Graph) link(u Use) {
	for _, have := range u.From.Out {
		if have == u {
			return
		}
	}
	u.From.Out = append(u.From.Out, u)
	u.To.In = append(u.To.In, u)
}

// embeddedPath returns the embedded fields sel passes through implicitly on
// its way to the field or method it selects.
func embeddedPath(sel *types.Selection) []*types.Var {
	var path []*types.Var
	t := sel.Recv()
	idx := sel.Index()
	for _, i := range idx[:len(idx)-1] {
		if ptr, ok := t.(*types.Pointer); ok {
			t = ptr.Elem()
		}
		st, ok := t.Underlying().(*types.Struct)
		if !ok {
			break
		}
		f := st.Field(i)
		path = append(path, f)
		t = f.Type()
	}
	return path
}

type dispatchKey struct {
	iface *types.Interface
	name  string
}

// dispatch is class hierarchy analysis over go/types: a method selected on
// an interface, or on a type parameter through its constraint, may reach the
// method of that name on every graph type whose value or pointer satisfies
// the interface. It is what golang.org/x/tools/go/callgraph/cha computes,
// restricted to the graph's types, without building SSA.
//
// Generic types are not candidates: whether an uninstantiated type implements
// an interface is unspecified in go/types, and guessing would be the wrong
// direction for a check whose edges never block anything anyway.
func (g *Graph) dispatch(sel *types.Selection) []*Sym {
	var iface *types.Interface
	satisfies := types.Implements
	switch r := sel.Recv().(type) {
	case *types.TypeParam:
		iface, _ = r.Constraint().Underlying().(*types.Interface)
		satisfies = types.Satisfies
	default:
		iface, _ = r.Underlying().(*types.Interface)
	}
	if iface == nil {
		return nil
	}
	m := sel.Obj().(*types.Func)
	key := dispatchKey{iface, m.Name()}
	if to, ok := g.dispatched[key]; ok {
		return to
	}
	var to []*Sym
	for _, t := range g.concrete {
		recv := t.Obj.Type()
		if !satisfies(recv, iface) {
			recv = types.NewPointer(recv)
			if !satisfies(recv, iface) {
				continue
			}
		}
		obj, _, _ := types.LookupFieldOrMethod(recv, true, m.Pkg(), m.Name())
		if fn, ok := obj.(*types.Func); ok {
			if s := g.byObj[fn.Origin()]; s != nil {
				to = append(to, s)
			}
		}
	}
	g.dispatched[key] = to
	return to
}

// Functions counts the functions and methods with a body, the population
// doppel's other commands count as units.
func (g *Graph) Functions() int {
	n := 0
	for _, s := range g.Syms {
		if s.body {
			n++
		}
	}
	return n
}

// Pos returns where s is declared.
func (g *Graph) Pos(s *Sym) token.Position { return g.Fset.Position(s.Obj.Pos()) }

func symLess(a, b *Sym) bool {
	if a.Pkg.Path != b.Pkg.Path {
		return a.Pkg.Path < b.Pkg.Path
	}
	return a.Obj.Pos() < b.Obj.Pos()
}

func sortUses(us []Use, end func(Use) *Sym) {
	sort.SliceStable(us, func(i, j int) bool { return symLess(end(us[i]), end(us[j])) })
}

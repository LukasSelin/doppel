package moves

import (
	"fmt"
	"go/types"
	"maps"
	"slices"
	"sort"
	"strings"
)

// Options are the analysis knobs.
type Options struct {
	// Share is the fraction of a function's static referrers that must sit in
	// one other package for the function to be a candidate. 1 (the default) is
	// "only that package refers to it"; below 1 the referrers that stay are
	// listed, and every one of them must import the target after the move.
	Share float64
}

// DefaultOptions is Share 1.
func DefaultOptions() Options { return Options{Share: 1} }

// Move suggests moving a function to the package that refers to it.
type Move struct {
	Func *Sym
	To   *Package
	// Users are the declarations in To that refer to Func, and Staying the
	// ones elsewhere (empty at Share 1). Share is len(Users) over both.
	Users   []*Sym
	Staying []*Sym
	Share   float64
	// With are the unexported symbols of Func's package that only Func and
	// each other refer to statically, so they move with it. A type brings its
	// methods, which are not listed.
	With []*Sym
	// Anchors are the types of Func's own package named in its signature that
	// do not move with it: a constructor or an entry point of a type that
	// stays, which belongs beside its type however few packages call it.
	Anchors []*Sym
	// Blocked says why the move would not compile as it stands: unexported
	// symbols it shares with code that stays, a name the target already
	// declares, an import cycle or internal-package rule it would break, or a
	// file this build does not compile that names what moves. Empty for a
	// clean move.
	Blocked []string
	// TestFiles are _test.go files that name Func or one of With, anywhere in
	// the graph. Test files are not type-checked, so these are found by name.
	TestFiles []string
}

// Find returns a move for every candidate function, in declaration order.
// Methods are never candidates: they cannot leave their receiver's package.
// Nor are init, main in package main, or a function in a generated file.
func Find(g *Graph, opt Options) []Move {
	if opt.Share <= 0 || opt.Share > 1 {
		opt.Share = 1
	}
	files := newFileIndex(g)
	var moves []Move
	for _, s := range g.Syms {
		if m, ok := g.candidate(s, opt, files); ok {
			moves = append(moves, m)
		}
	}
	return moves
}

func (g *Graph) candidate(s *Sym, opt Options, files *fileIndex) (Move, bool) {
	fn, ok := s.Obj.(*types.Func)
	if !ok || s.Owner != nil || s.generated || fn.Name() == "init" ||
		(fn.Name() == "main" && s.Pkg.Name == "main") || fn.Name() == "_" {
		return Move{}, false
	}
	byPkg := map[*Package][]*Sym{}
	total := 0
	for _, u := range s.In {
		if u.Dynamic {
			continue
		}
		byPkg[u.From.Pkg] = append(byPkg[u.From.Pkg], u.From)
		total++
	}
	var to *Package
	for p, users := range byPkg {
		if to == nil || len(users) > len(byPkg[to]) || (len(users) == len(byPkg[to]) && p.Path < to.Path) {
			to = p
		}
	}
	// A tie with the function's own package is not a reason to move it.
	if to == nil || to == s.Pkg || len(byPkg[s.Pkg]) >= len(byPkg[to]) {
		return Move{}, false
	}
	share := float64(len(byPkg[to])) / float64(total)
	if share < opt.Share-1e-9 {
		return Move{}, false
	}
	m := Move{Func: s, To: to, Users: byPkg[to], Share: share}
	for _, u := range s.In {
		if !u.Dynamic && u.From.Pkg != to {
			m.Staying = append(m.Staying, u.From)
		}
	}
	gr := closure(s)
	for _, x := range gr.order[1:] {
		if x.Owner == nil {
			m.With = append(m.With, x)
		}
	}
	m.Anchors = anchors(s, gr)
	m.Blocked = g.blockers(m, gr)
	m.TestFiles, m.Blocked = files.naming(s.Pkg, gr, m.Blocked)
	return m, true
}

// group is a set of symbols that move together.
type group struct {
	in    map[*Sym]bool
	order []*Sym // the function first, then what joined, methods after their type
}

// closure returns s and the unexported symbols of its package that only s
// and each other refer to statically, iterated until nothing more joins, so
// that nothing left behind needs them.
func closure(s *Sym) *group {
	gr := &group{in: map[*Sym]bool{s: true}, order: []*Sym{s}}
	for changed := true; changed; {
		changed = false
		for i := 0; i < len(gr.order); i++ {
			for _, u := range gr.order[i].Out {
				unit := u.To
				if unit.Owner != nil {
					unit = unit.Owner
				}
				if u.Dynamic || gr.in[unit] || unit.Pkg != s.Pkg || unit.Obj.Exported() || unit.Obj.Name() == "_" {
					continue
				}
				if gr.onlyUsedBy(unit) {
					gr.join(unit)
					changed = true
				}
			}
		}
	}
	return gr
}

// onlyUsedBy reports whether everything that refers statically to unit or
// one of its methods is in the group or part of unit itself.
func (gr *group) onlyUsedBy(unit *Sym) bool {
	for _, x := range unitSyms(unit) {
		for _, u := range x.In {
			if u.Dynamic || gr.in[u.From] || u.From == unit || u.From.Owner == unit {
				continue
			}
			return false
		}
	}
	return true
}

func (gr *group) join(unit *Sym) {
	for _, x := range unitSyms(unit) {
		gr.in[x] = true
		gr.order = append(gr.order, x)
	}
}

// unitSyms returns unit and, if it is a type, its methods.
func unitSyms(unit *Sym) []*Sym { return append([]*Sym{unit}, unit.Methods...) }

// anchors returns the named types of s's package in its signature that stay
// behind. An unexported one is also a blocker, through the reference the
// signature makes; an exported one is not, which is why it is its own field.
func anchors(s *Sym, gr *group) []*Sym {
	seen := map[*Sym]bool{}
	var out []*Sym
	sig := s.Obj.(*types.Func).Signature()
	for _, tuple := range []*types.Tuple{sig.Params(), sig.Results()} {
		for i := 0; i < tuple.Len(); i++ {
			namedIn(tuple.At(i).Type(), func(n *types.Named) {
				x := typeSym(s, n)
				if x != nil && !gr.in[x] && !seen[x] {
					seen[x] = true
					out = append(out, x)
				}
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return symLess(out[i], out[j]) })
	return out
}

// typeSym finds the symbol of n among the types s refers to, which is every
// type its signature names.
func typeSym(s *Sym, n *types.Named) *Sym {
	obj := n.Origin().Obj()
	if obj.Pkg() != s.Pkg.Types {
		return nil
	}
	for _, u := range s.Out {
		if u.To.Obj == obj {
			return u.To
		}
	}
	return nil
}

// namedIn calls fn for every named type t is built from.
func namedIn(t types.Type, fn func(*types.Named)) {
	switch t := t.(type) {
	case *types.Named:
		fn(t)
		if args := t.TypeArgs(); args != nil {
			for i := 0; i < args.Len(); i++ {
				namedIn(args.At(i), fn)
			}
		}
	case *types.Pointer:
		namedIn(t.Elem(), fn)
	case *types.Slice:
		namedIn(t.Elem(), fn)
	case *types.Array:
		namedIn(t.Elem(), fn)
	case *types.Map:
		namedIn(t.Key(), fn)
		namedIn(t.Elem(), fn)
	case *types.Chan:
		namedIn(t.Elem(), fn)
	}
}

// blockers says what keeps m's group from moving to m.To as it stands.
func (g *Graph) blockers(m Move, gr *group) []string {
	var why []string
	from := m.Func.Pkg
	shared := map[string]bool{}
	deps := map[string]bool{}
	for _, x := range gr.order {
		for _, u := range x.Out {
			if u.Dynamic || gr.in[u.To] {
				continue
			}
			if u.Obj.Pkg() == from.Types && !u.Obj.Exported() {
				shared[describe(u)] = true
			}
			// Only package-level names are written qualified, so only they
			// need an import once the group lives in m.To.
			if u.Obj == u.To.Obj && u.To.Owner == nil && u.To.Pkg != m.To {
				deps[u.To.Pkg.Path] = true
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(shared)) {
		why = append(why, fmt.Sprintf("uses unexported %s, which code staying in %s also uses", name, from.Name))
	}
	for _, x := range gr.order {
		if x.Owner == nil && m.To.Types.Scope().Lookup(x.Obj.Name()) != nil {
			why = append(why, fmt.Sprintf("%s already declares %s", m.To.Name, x.Obj.Name()))
		}
	}
	why = append(why, g.cycles(m, gr, deps)...)
	for _, dep := range slices.Sorted(maps.Keys(deps)) {
		if !internalAllowed(m.To.Path, dep) {
			why = append(why, fmt.Sprintf("%s may not import internal package %s", m.To.Path, dep))
		}
	}
	staying := map[string]bool{}
	for _, u := range m.Staying {
		staying[u.Pkg.Path] = true
	}
	for _, p := range slices.Sorted(maps.Keys(staying)) {
		if !internalAllowed(p, m.To.Path) {
			why = append(why, fmt.Sprintf("%s, which still uses %s, may not import internal package %s", p, m.Func.Name(), m.To.Path))
		}
	}
	return why
}

// cycles checks the import graph as it would be after the move. m.To gains
// an import of every package the group refers to, and loses its import of
// Func's package when nothing else of that package is left for it to use.
// Every package whose code still refers to Func gains an import of m.To. A
// gained edge closes a cycle when its target already reaches its source.
//
// At Share 1 this cannot fire, and the reason is worth keeping: m.To uses
// Func, so it imports Func's package, which imports every dependency already,
// so no dependency can lead back to m.To — and there is nobody staying to
// gain an edge. It guards relaxed shares, where the referrers that stay must
// import the target, and a direct call with a target that does not use Func.
func (g *Graph) cycles(m Move, gr *group, deps map[string]bool) []string {
	from := m.Func.Pkg
	edges := map[string][]string{}
	for path, p := range g.Pkgs {
		for _, imp := range p.Imports {
			if g.Pkgs[imp] != nil {
				edges[path] = append(edges[path], imp)
			}
		}
	}
	if !deps[from.Path] && !g.stillUses(m.To, from, gr) {
		edges[m.To.Path] = without(edges[m.To.Path], from.Path)
	}
	type edge struct{ from, to string }
	var gained []edge
	for _, dep := range slices.Sorted(maps.Keys(deps)) {
		gained = append(gained, edge{m.To.Path, dep})
		edges[m.To.Path] = append(edges[m.To.Path], dep)
	}
	stay := map[string]bool{}
	for _, u := range m.Staying {
		stay[u.Pkg.Path] = true
	}
	for _, p := range slices.Sorted(maps.Keys(stay)) {
		gained = append(gained, edge{p, m.To.Path})
		edges[p] = append(edges[p], m.To.Path)
	}
	for k := range edges {
		sort.Strings(edges[k])
	}
	var why []string
	seen := map[string]bool{}
	for _, e := range gained {
		if path := importPath(edges, e.to, e.from); path != nil {
			msg := "import cycle: " + strings.Join(append([]string{e.from}, path...), " -> ")
			if !seen[msg] {
				seen[msg] = true
				why = append(why, msg)
			}
		}
	}
	return why
}

// stillUses reports whether to refers to anything of from outside the group.
func (g *Graph) stillUses(to, from *Package, gr *group) bool {
	for _, s := range g.Syms {
		if s.Pkg != to {
			continue
		}
		for _, u := range s.Out {
			if !u.Dynamic && u.To.Pkg == from && !gr.in[u.To] {
				return true
			}
		}
	}
	return false
}

func without(xs []string, drop string) []string {
	var out []string
	for _, x := range xs {
		if x != drop {
			out = append(out, x)
		}
	}
	return out
}

// importPath returns the chain of imports leading from package from to
// package to, from first and to last, or nil if from does not reach to.
func importPath(edges map[string][]string, from, to string) []string {
	prev := map[string]string{from: ""}
	queue := []string{from}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if p == to {
			var path []string
			for ; p != ""; p = prev[p] {
				path = append([]string{p}, path...)
			}
			return path
		}
		for _, imp := range edges[p] {
			if _, seen := prev[imp]; !seen {
				prev[imp] = p
				queue = append(queue, imp)
			}
		}
	}
	return nil
}

// describe names what a use refers to, e.g. "helper" or "field T.name".
func describe(u Use) string {
	if v, ok := u.Obj.(*types.Var); ok && v.IsField() {
		return "field " + u.To.Name() + "." + v.Name()
	}
	return u.To.Name()
}

// internalAllowed reports whether the package importer may import dep under
// the internal package rule.
func internalAllowed(importer, dep string) bool {
	var parent string
	if i := strings.LastIndex(dep, "/internal/"); i >= 0 {
		parent = dep[:i]
	} else if p, ok := strings.CutSuffix(dep, "/internal"); ok {
		parent = p
	} else {
		return true
	}
	return importer == parent || strings.HasPrefix(importer, parent+"/")
}

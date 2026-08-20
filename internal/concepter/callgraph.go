package concepter

import (
	"path"
	"sort"
	"strings"

	"github.com/lukse/doppel/internal/parser"
)

// Graph is the repo-internal call graph over qualified unit names, in both
// directions. Every name in it identifies exactly one CodeUnit; external and
// unresolvable calls simply do not appear.
//
// This replaces a bare-name graph whose keys collided (New in two packages
// shared edges) and whose "AST fast path" was a string-equality lookup that
// could not see method calls at all, plus a substring-scan fallback that ran
// for every zero-callee function and manufactured garbage edges. Both are
// gone: edges now exist only where the resolver found a unique target.
type Graph struct {
	Callers map[string][]string // qualified callee → qualified callers, sorted
	Callees map[string][]string // qualified caller → resolved internal callees, sorted
}

// QualifiedName returns the corpus-unique identity of a unit:
// "package.Name", where Name already carries the receiver for methods, so a
// method key reads "comparator.*Comparator.Compare". The key is opaque — built
// here, compared for equality, and split only at the FIRST dot to recover the
// package (Go package names cannot contain dots). Never parse it anywhere else.
func QualifiedName(u parser.CodeUnit) string {
	if u.Package == "" {
		return u.Name
	}
	return u.Package + "." + u.Name
}

// KeyPackage returns the package half of a qualified name.
func KeyPackage(qualified string) string {
	if i := strings.IndexByte(qualified, '.'); i >= 0 {
		return qualified[:i]
	}
	return ""
}

// resolver answers "which unit, if any, does this callee string name?" from
// the three indexes a corpus supports without a type checker.
type resolver struct {
	// functions: package name + "." + plain function name → qualified key.
	// Only unambiguous entries survive; two internal packages sharing a name
	// make their same-named functions unresolvable rather than guessable.
	functions map[string]string
	// methods: bare method name → qualified keys of every unit declaring it.
	// Consulted only when exactly one exists, because the receiver of a call
	// like comp.Compare is a variable whose type the AST does not know.
	methods map[string][]string
}

func newResolver(units []parser.CodeUnit) *resolver {
	r := &resolver{
		functions: make(map[string]string),
		methods:   make(map[string][]string),
	}
	ambiguous := make(map[string]bool)
	for _, u := range units {
		qn := QualifiedName(u)
		if u.ReceiverType != "" {
			method := u.Name[strings.LastIndexByte(u.Name, '.')+1:]
			r.methods[method] = append(r.methods[method], qn)
			continue
		}
		key := u.Package + "." + u.Name
		if _, dup := r.functions[key]; dup {
			ambiguous[key] = true
		}
		r.functions[key] = qn
	}
	// Two same-named functions in two same-named packages: refuse to guess.
	for key := range ambiguous {
		delete(r.functions, key)
	}
	return r
}

// resolve maps one raw extractCallees string to a qualified unit key.
//
// The ladder, in order:
//
//   - "recv.sel" where recv is an import of the caller's file → a package-
//     qualified call. The target package name is approximated by the import
//     path's last segment, which is exact whenever the directory name matches
//     the package clause (violated by paths like gopkg.in/yaml.v2, where the
//     result is a miss, never a false edge).
//   - "recv.sel" where recv is not an import → a method call on a variable.
//     Without a type checker the receiver's type is unknowable, so this
//     resolves only when exactly one method in the corpus bears that name.
//   - bare "c" → a same-package function; failing that, the unique-method rule
//     again, because extractCallees collapses chained selectors (c.scorer.X)
//     to the bare method name. Builtins and externals fall out here.
//
// Known, documented imprecision, all AST-level and all shared with the tagger's
// signals (see the tx/mtx note on AnyReceiver): a local variable shadowing an
// import name is mistaken for the package; an external import whose base equals
// an internal package name can hit an internal function of the same name; dot
// imports make their functions arrive bare and unresolvable. Each failure
// needs a name coincidence and costs one edge; the cure for all of them is
// go/types, which is out of proportion for this tool.
func (r *resolver) resolve(caller parser.CodeUnit, callee string) (string, bool) {
	if dot := strings.IndexByte(callee, '.'); dot >= 0 {
		recv, sel := callee[:dot], callee[dot+1:]
		if refPath, ok := caller.Signals.RefPath(recv); ok {
			qn, ok := r.functions[path.Base(refPath)+"."+sel]
			return qn, ok
		}
		return r.uniqueMethod(sel)
	}
	if qn, ok := r.functions[caller.Package+"."+callee]; ok {
		return qn, true
	}
	return r.uniqueMethod(callee)
}

func (r *resolver) uniqueMethod(name string) (string, bool) {
	if candidates := r.methods[name]; len(candidates) == 1 {
		return candidates[0], true
	}
	return "", false
}

// BuildCallGraph resolves every unit's AST-derived callees against the corpus
// and returns the repo-internal call graph in both directions. Self-edges
// (recursion) are excluded: a function is not its own caller for fan-in or
// context purposes.
//
// Determinism: units are visited in slice order, Callees are already sorted by
// the parser, and every value slice is canonically sorted before returning.
func BuildCallGraph(units []parser.CodeUnit) *Graph {
	r := newResolver(units)
	g := &Graph{
		Callers: make(map[string][]string, len(units)),
		Callees: make(map[string][]string, len(units)),
	}
	for _, u := range units {
		qn := QualifiedName(u)
		if _, ok := g.Callers[qn]; !ok {
			g.Callers[qn] = nil
		}
	}
	for _, caller := range units {
		callerQN := QualifiedName(caller)
		for _, callee := range caller.Callees {
			target, ok := r.resolve(caller, callee)
			if !ok || target == callerQN {
				continue
			}
			g.Callers[target] = appendUnique(g.Callers[target], callerQN)
			g.Callees[callerQN] = appendUnique(g.Callees[callerQN], target)
		}
	}
	for _, edges := range g.Callers {
		sort.Strings(edges)
	}
	for _, edges := range g.Callees {
		sort.Strings(edges)
	}
	return g
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

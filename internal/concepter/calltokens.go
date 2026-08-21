package concepter

import (
	"path"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/parser"
)

// QualifiedNames returns the set of QualifiedName(u) over units — the
// repo-internal identity set that CallTokens uses to keep internal and
// external call forms from double-counting.
func QualifiedNames(units []parser.CodeUnit) map[string]bool {
	internal := make(map[string]bool, len(units))
	for i := range units {
		internal[QualifiedName(units[i])] = true
	}
	return internal
}

// CallTokens builds one unit's deduped, sorted resolved call-token set:
// qualified repo-internal callees from the call graph, plus external calls
// whose selector receiver is an import binding of the calling file, keyed by
// full import path ("database/sql.Open") so two packages importing the same
// API meet on the same token. Bare names and variable-receiver method calls
// are never tokens — unresolved matching is exactly what the resolved call
// graph exists to avoid. The internal-QN guard keeps an internal package
// called through its own import from counting twice — once as the resolved
// graph edge and once as an import-qualified external token for the same
// call.
func CallTokens(u parser.CodeUnit, g *Graph, internal map[string]bool) []string {
	set := make(map[string]bool)
	for _, callee := range g.Callees[QualifiedName(u)] {
		set[callee] = true
	}
	for _, raw := range u.Callees {
		dot := strings.IndexByte(raw, '.')
		if dot <= 0 {
			continue // bare name: unresolved, never a token
		}
		recv, sel := raw[:dot], raw[dot+1:]
		refPath, ok := u.Signals.RefPath(recv)
		if !ok {
			continue // variable receiver: unresolved, never a token
		}
		if internal[path.Base(refPath)+"."+sel] {
			continue // repo-internal: the resolved graph edge already covers it
		}
		set[refPath+"."+sel] = true
	}
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for t := range set {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

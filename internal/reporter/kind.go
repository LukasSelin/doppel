package reporter

import (
	"fmt"
	"strings"

	"github.com/LukasSelin/doppel/internal/analyzer"
)

// kindClause renders a KindNote as one readable clause, without its leading
// label. One renderer serves the text report, the markdown report and the
// family census so the three can never say different things; md wraps
// identifiers in backticks.
//
//	interface implementations — both implement Validate(context.Context) (error) on *AWS and *GCP, sibling packages aws and gcp
//	diverged copy — evalCallOld and evalCall share the stem evalCall in package template
func kindClause(k *analyzer.KindNote, family bool, md bool) string {
	code := func(s string) string {
		if md {
			return "`" + mdEscape(s) + "`"
		}
		return s
	}
	where := kindWhere(k, code)
	switch k.Kind {
	case analyzer.KindInterfaceImpl:
		sig := code(k.Method + k.Signature)
		if family {
			return fmt.Sprintf("%s of %s, %s", k.Kind, sig, where)
		}
		return fmt.Sprintf("%s — both implement %s on %s, %s", k.Kind, sig, joinAnd(k.Receivers, code), where)
	case analyzer.KindFork:
		if family {
			return fmt.Sprintf("diverged copies sharing the stem %s %s", code(k.Method), where)
		}
		return fmt.Sprintf("%s — %s share the stem %s %s", k.Kind, joinAnd(k.Names, code), code(k.Method), where)
	}
	return k.Kind
}

// kindWhere phrases the package relation: "in package template",
// "sibling packages aws and gcp", "packages a and b".
func kindWhere(k *analyzer.KindNote, code func(string) string) string {
	switch k.Relation {
	case analyzer.RelationSamePackage:
		if len(k.Packages) == 1 {
			return "in package " + code(k.Packages[0])
		}
	case analyzer.RelationSiblings:
		return "sibling packages " + joinAnd(k.Packages, code)
	}
	return "packages " + joinAnd(k.Packages, code)
}

// joinAnd renders "a", "a and b", "a, b and c".
func joinAnd(items []string, code func(string) string) string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = code(s)
	}
	switch len(out) {
	case 0:
		return ""
	case 1:
		return out[0]
	case 2:
		return out[0] + " and " + out[1]
	}
	return strings.Join(out[:len(out)-1], ", ") + " and " + out[len(out)-1]
}

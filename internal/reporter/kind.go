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
	case analyzer.KindMirror:
		return fmt.Sprintf("%s — %s on %s are one operation run in opposite directions", k.Kind, joinAnd(k.Names, code), code(k.Receivers[0]))
	case analyzer.KindThinWrappers:
		return fmt.Sprintf("%s — both are small bodies delegating to %s and naming different things, %s", k.Kind, joinAnd(k.Shared, code), where)
	case analyzer.KindSubsystemCopies:
		return fmt.Sprintf("%s — %s, %s; no caller uses both", k.Kind,
			callerSide(k.Names[0], k.CallerCounts[0], k.CallerPackages[0], code),
			callerSide(k.Names[1], k.CallerCounts[1], k.CallerPackages[1], code))
	case analyzer.KindDifferentCalls:
		return fmt.Sprintf("%s — the bodies share a shape but only %.0f%% of their calls, %s", k.Kind, 100*k.Overlap, where)
	}
	return k.Kind
}

// callerSide phrases one side of a subsystem-copies pair:
// "concepter.QualifiedName has 14 callers in cmd, family, reporter and 3 more".
func callerSide(name string, callers int, pkgs []string, code func(string) string) string {
	const shown = 3
	where := joinAnd(pkgs, code)
	if len(pkgs) > shown {
		parts := make([]string, 0, shown+1)
		for _, p := range pkgs[:shown] {
			parts = append(parts, code(p))
		}
		parts = append(parts, fmt.Sprintf("%d more", len(pkgs)-shown))
		where = joinAnd(parts, func(s string) string { return s })
	}
	noun := "callers"
	if callers == 1 {
		noun = "caller"
	}
	return fmt.Sprintf("%s has %d %s in %s", code(name), callers, noun, where)
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
	if len(k.Packages) == 1 {
		// One package clause in two unrelated directories: two package
		// mains, most often, which "packages main" would misread as one.
		return "package " + code(k.Packages[0]) + " in two directories"
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

// KindClause is kindClause as plain prose, for a consumer outside this package.
//
// Exported rather than duplicated: the dashboard shows the same sentence the
// text report does, and two renderers of one KindNote would eventually say
// different things about the same pair.
func KindClause(k *analyzer.KindNote) string {
	if k == nil {
		return ""
	}
	return kindClause(k, false, false)
}

package concepter

import "testing"

// CallTokens must not double-count an internal package called through its own
// import: the resolved graph edge covers it, and the import-qualified form is
// skipped.
func TestCallTokensDoNotDoubleCountInternalCalls(t *testing.T) {
	helperFile := `package helper

func Normalize(s string) string {
	out := ""
	for _, c := range s {
		if c != ' ' {
			out += string(c)
		}
	}
	return out
}
`
	callerFile := `package caller

import "example.com/proj/helper"

func Clean(s string) string {
	if s == "" {
		return s
	}
	return helper.Normalize(s)
}
`
	units := unitsFromSource(t, "helper.go", helperFile)
	units = append(units, unitsFromSource(t, "caller.go", callerFile)...)
	g := BuildCallGraph(units)
	internal := QualifiedNames(units)

	var caller int
	for i := range units {
		if units[i].Name == "Clean" {
			caller = i
		}
	}
	tokens := CallTokens(units[caller], g, internal)
	count := 0
	for _, tok := range tokens {
		if tok == "helper.Normalize" || tok == "example.com/proj/helper.Normalize" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("tokens = %v; internal call must appear exactly once, got %d forms", tokens, count)
	}
}

// Bare-name and variable-receiver calls never become tokens: unresolved
// matching is forbidden.
func TestCallTokensExcludeUnresolvedCalls(t *testing.T) {
	src := `package fix

func Use(tx interface{ Commit() error }) error {
	doWork()
	return tx.Commit()
}
`
	units := unitsFromSource(t, "fix.go", src)
	g := BuildCallGraph(units)
	tokens := CallTokens(units[0], g, map[string]bool{})
	if len(tokens) != 0 {
		t.Errorf("tokens = %v, want none: doWork is bare and tx.Commit has a variable receiver", tokens)
	}
}

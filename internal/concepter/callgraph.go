package concepter

import (
	"strings"

	"github.com/lukse/doppel/internal/parser"
)

// CallGraph maps each qualified function name (Package.Name) to the list of
// qualified caller names. Using qualified keys prevents false merges when
// different packages contain functions with the same bare name.
type CallGraph map[string][]string

// minCallerNameLen skips very short names to reduce false-positive caller edges.
const minCallerNameLen = 4

// QualifiedName returns "Package.Name" for a CodeUnit, or just Name if Package is empty.
func QualifiedName(u parser.CodeUnit) string {
	if u.Package == "" {
		return u.Name
	}
	return u.Package + "." + u.Name
}

// BuildCallGraph performs an O(n²) text scan across all units.
// For each (caller, callee) pair where the names differ, if callee.Name appears
// anywhere in caller.Body the caller is recorded as a caller of the callee.
// All keys and values use qualified names (Package.Name).
func BuildCallGraph(units []parser.CodeUnit) CallGraph {
	graph := make(CallGraph, len(units))

	// Build reverse index: bare name → list of qualified names.
	bareToQualified := make(map[string][]string)
	for _, u := range units {
		qn := QualifiedName(u)
		if _, ok := graph[qn]; !ok {
			graph[qn] = nil
		}
		bareToQualified[u.Name] = append(bareToQualified[u.Name], qn)
	}

	for _, caller := range units {
		callerQN := QualifiedName(caller)
		if len(caller.Callees) > 0 {
			// Use AST-derived callees (Go units): precise, no false positives.
			for _, calleeName := range caller.Callees {
				addCallerEdges(graph, bareToQualified, calleeName, callerQN, caller.Package)
			}
		} else {
			// Fallback: O(n) text scan for non-Go units.
			for _, callee := range units {
				calleeQN := QualifiedName(callee)
				if callerQN == calleeQN {
					continue
				}
				if len(callee.Name) < minCallerNameLen {
					continue
				}
				if strings.Contains(caller.Body, callee.Name) {
					graph[calleeQN] = appendUnique(graph[calleeQN], callerQN)
				}
			}
		}
	}
	return graph
}

// addCallerEdges resolves a bare or selector callee name to qualified graph
// keys and records the caller edge. For bare names it prefers same-package
// matches; for selector expressions (x.Y) it tries an exact qualified-key
// match first, then falls back to resolving the selector part.
func addCallerEdges(graph CallGraph, bareToQualified map[string][]string, calleeName, callerQN, callerPkg string) {
	// Try exact match on qualified key (handles selector expressions where
	// the import alias matches the package name, e.g. "json.Marshal" →
	// graph key "json.Marshal").
	if _, ok := graph[calleeName]; ok {
		graph[calleeName] = appendUnique(graph[calleeName], callerQN)
		return
	}

	// For selector expressions like "foo.Bar", also try resolving just "Bar".
	bareName := calleeName
	if idx := strings.LastIndex(calleeName, "."); idx >= 0 {
		bareName = calleeName[idx+1:]
	}

	candidates := bareToQualified[bareName]
	if len(candidates) == 0 {
		return
	}

	// Prefer same-package match for bare names (e.g. helper() calls within
	// the same package). For selector expressions the caller package won't
	// match anyway, so all candidates are tried.
	if bareName == calleeName {
		for _, cand := range candidates {
			if strings.HasPrefix(cand, callerPkg+".") {
				graph[cand] = appendUnique(graph[cand], callerQN)
				return
			}
		}
	}

	// No same-package match — record against all candidates.
	for _, cand := range candidates {
		graph[cand] = appendUnique(graph[cand], callerQN)
	}
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

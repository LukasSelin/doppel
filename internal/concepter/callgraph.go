package concepter

import (
	"strings"

	"github.com/lukse/doppel/internal/parser"
)

// CallGraph maps each function Name to the list of other function Names whose
// Body contains the callee's name as a substring.
// Key: callee Name. Value: deduplicated slice of caller Names.
type CallGraph map[string][]string

// minCallerNameLen skips very short names to reduce false-positive caller edges.
const minCallerNameLen = 4

// BuildCallGraph performs an O(n²) text scan across all units.
// For each (caller, callee) pair where the names differ, if callee.Name appears
// anywhere in caller.Body the caller is recorded as a caller of the callee.
func BuildCallGraph(units []parser.CodeUnit) CallGraph {
	graph := make(CallGraph, len(units))

	// Ensure every callee has an entry, even if no callers are found.
	for _, u := range units {
		if _, ok := graph[u.Name]; !ok {
			graph[u.Name] = nil
		}
	}

	for _, caller := range units {
		if len(caller.Callees) > 0 {
			// Use AST-derived callees (Go units): precise, no false positives.
			for _, calleeName := range caller.Callees {
				if _, ok := graph[calleeName]; ok {
					graph[calleeName] = appendUnique(graph[calleeName], caller.Name)
				}
			}
		} else {
			// Fallback: O(n) text scan for non-Go units.
			for _, callee := range units {
				if caller.Name == callee.Name {
					continue
				}
				if len(callee.Name) < minCallerNameLen {
					continue
				}
				if strings.Contains(caller.Body, callee.Name) {
					graph[callee.Name] = appendUnique(graph[callee.Name], caller.Name)
				}
			}
		}
	}
	return graph
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

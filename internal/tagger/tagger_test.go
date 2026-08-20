package tagger

import (
	"testing"

	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// Axiom 8, the tagger half of the vocabulary's integrity check: the rule table
// and the concrete concept terms must be in exact correspondence, both ways.
//
// It lives here rather than in ontology.Validate because the check needs the
// rule table, and importing tagger from ontology would be a cycle.
//
// Without it, drift fails silently and changes scores. A rule emitting a tag no
// concept term declares would score zero relatedness against everything except
// an identical tag, and a concept term with no rule would sit in the taxonomy
// affecting nothing while looking load-bearing.
func TestEveryRuleNamesAConcreteConcept(t *testing.T) {
	o := ontology.Default()
	for _, rule := range patternRules {
		term, ok := o.Get(rule.concept)
		if !ok {
			t.Errorf("rule %q names a concept that does not exist", rule.concept)
			continue
		}
		if term.Kind != ontology.KindConcept {
			t.Errorf("rule %q names a %s term, want a concept", rule.concept, term.Kind)
		}
		if term.Abstract {
			t.Errorf("rule %q names the abstract concept %q; only leaves can be asserted of real code", rule.concept, term.ID)
		}
		if rule.signalCount() == 0 {
			t.Errorf("rule %q declares no evidence channels, so it can never fire", rule.concept)
		}
	}
}

func TestEveryConcreteConceptHasExactlyOneRule(t *testing.T) {
	rules := map[ontology.TermID]int{}
	for _, rule := range patternRules {
		rules[rule.concept]++
	}
	for _, term := range ontology.Default().TermsOfKind(ontology.KindConcept) {
		if term.Abstract {
			if n := rules[term.ID]; n != 0 {
				t.Errorf("abstract concept %q has %d rules, want none", term.ID, n)
			}
			continue
		}
		if n := rules[term.ID]; n != 1 {
			t.Errorf("concrete concept %q has %d rules, want exactly 1", term.ID, n)
		}
	}
}

// tagOne parses a snippet expected to hold exactly one function and tags it.
func tagOne(t *testing.T, src string) []string {
	t.Helper()
	units, err := parser.ParseSource("fixture.go", []byte(src))
	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("fixture has %d functions, want 1", len(units))
	}
	return Tag(units[0])
}

func TestTag(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "no signals",
			src:  "package p\nfunc add(a, b int) int { return a + b }",
			want: nil,
		},
		{
			name: "wrapping via %w mid-string, the case the old rule missed",
			src: `package p
import "fmt"
func wrap(err error, name string) error {
	return fmt.Errorf("read %w: %s", err, name)
}`,
			want: []string{"error_wrapping"},
		},
		{
			name: "bare Errorf without %w no longer counts as wrapping",
			src: `package p
import "fmt"
func note(name string) error {
	return fmt.Errorf("bad name %q", name)
}`,
			want: nil,
		},
		{
			name: "error inspection is not wrapping",
			src: `package p
import ("errors"; "io")
func isEOF(err error) bool {
	return errors.Is(err, io.EOF)
}`,
			want: nil,
		},
		{
			name: "SQL in a string literal",
			src: `package p
func load(q queryer) error {
	return q.Scan("SELECT name FROM users")
}`,
			want: []string{"db_access"},
		},
		{
			name: "SQL in a comment does not fire, the false positive the AST move kills",
			src: `package p
// remove uses DELETE ROW semantics internally.
func remove(items []int, i int) []int {
	return append(items[:i], items[i+1:]...)
}`,
			want: nil,
		},
		{
			name: "mutex is concurrency, not transaction, despite containing tx",
			src: `package p
import "sync"
func guarded(mtx *sync.Mutex, f func()) {
	mtx.Lock()
	defer mtx.Unlock()
	f()
}`,
			want: []string{"concurrency"},
		},
		{
			name: "tx receiver is transaction",
			src: `package p
func commit(tx txn) error {
	return tx.Commit()
}`,
			want: []string{"transaction"},
		},
		{
			name: "go statement alone marks concurrency",
			src: `package p
func fire(f func()) {
	go f()
}`,
			want: []string{"concurrency"},
		},
		{
			name: "channel in signature alone marks concurrency",
			src: `package p
func drain(c <-chan int) {
	for range c {
	}
}`,
			want: []string{"concurrency"},
		},
		{
			name: "retry evidence is lexical",
			src: `package p
func retryWithBackoff(f func() error) error {
	var err error
	for i := 0; i < 3; i++ {
		if err = f(); err == nil {
			return nil
		}
	}
	return err
}`,
			want: []string{"retry"},
		},
		{
			name: "database/sql import alone is db evidence",
			src: `package p
import "database/sql"
func ping(conn *sql.DB) error {
	return conn.Ping()
}`,
			want: []string{"db_access"},
		},
		{
			name: "declaration order, not evidence order",
			src: `package p
import "net/http"
func fetchWithRetry(url string, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		if _, err := http.Get(url); err == nil {
			return nil
		}
	}
	return nil
}`,
			want: []string{"retry", "http_call"},
		},
		{
			name: "mapping via json",
			src: `package p
import "encoding/json"
func encode(v any) ([]byte, error) {
	return json.Marshal(v)
}`,
			want: []string{"mapping"},
		},
		{
			name: "caching via receiver convention",
			src: `package p
func lookup(cache store, key string) (string, bool) {
	return cache.Get(key)
}`,
			want: []string{"caching"},
		},
		{
			name: "validation via identifier",
			src: `package p
func validateInput(s string) error {
	if s == "" {
		return errEmpty
	}
	return nil
}`,
			want: []string{"validation"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tagOne(t, tt.src)
			if len(got) != len(tt.want) {
				t.Fatalf("Tag() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("Tag() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// The tags are the tool's public vocabulary: they appear in every report and in
// the docs. Moving to AST evidence must not have renamed any of them.
func TestTagNamesAreUnchanged(t *testing.T) {
	want := []string{
		"retry", "http_call", "db_access", "validation", "mapping",
		"transaction", "caching", "concurrency", "error_wrapping",
	}
	if len(patternRules) != len(want) {
		t.Fatalf("got %d rules, want %d", len(patternRules), len(want))
	}
	for i, rule := range patternRules {
		if string(rule.concept) != want[i] {
			t.Errorf("rule %d emits %q, want %q", i, rule.concept, want[i])
		}
	}
}

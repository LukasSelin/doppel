package tagger

import (
	"testing"

	"github.com/lukse/doppel/internal/ontology"
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
		if len(rule.keywords) == 0 {
			t.Errorf("rule %q has no keywords, so it can never fire", rule.concept)
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

func TestTag(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"no signals", "func add(a, b int) int { return a + b }", nil},
		{"single tag", "if err != nil { return fmt.Errorf(\"boom: %w\", err) }", []string{"error_wrapping"}},
		{"receiver-qualified call", "rows, err := db.Query(q)", []string{"db_access"}},
		// Declaration order, not match order: retry is declared before
		// error_wrapping, so it comes first even though the body mentions the
		// wrapping call earlier.
		{"two tags come back in declaration order",
			"fmt.Errorf(\"attempt failed\"); for i := 0; i < maxRetries; i++ {}",
			[]string{"retry", "error_wrapping"}},
		{"one keyword is enough, and a tag is never repeated",
			"retry(); retry(); backoff()",
			[]string{"retry"}},
		{"empty body", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tag(tt.body)
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
// the docs. Introducing the ontology must not have renamed any of them.
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

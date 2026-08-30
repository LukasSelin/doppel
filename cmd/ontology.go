package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/spf13/cobra"
)

var (
	ontologyKind string
	ontologyDefs bool
)

var ontologyCmd = &cobra.Command{
	Use:   "ontology",
	Short: "Print the vocabulary doppel reasons over, and check it is consistent",
	Long: `Print the entity kinds, relations, concepts and roles that scoring is based on,
then check them against the ontology's axioms.

The vocabulary decides how scores are computed: the concept taxonomy is what
lets two functions doing related kinds of work score partial credit as cousins
rather than zero, and the weight on each relation is that signal's contribution
to the structural overlap score. This command exists so both can be reviewed
without reading source.

The concept leaves shown here are SEEDS, not the vocabulary a run reasons over.
Concepts are learned per corpus: analyze replaces these fourteen leaves with
what it finds in the code, hanging the learned concepts from the same abstract
interior. Run "doppel analyze" to see a corpus's own vocabulary.

Exits non-zero if any axiom is violated.`,
	Args: cobra.NoArgs,
	RunE: runOntology,
}

func init() {
	ontologyCmd.Flags().StringVar(&ontologyKind, "kind", "", "Show only one family: entity, relation, concept or role")
	ontologyCmd.Flags().BoolVar(&ontologyDefs, "defs", false, "Include the definition of each term")
	rootCmd.AddCommand(ontologyCmd)
}

func runOntology(cmd *cobra.Command, args []string) error {
	o := ontology.Default()

	kinds := ontology.Kinds
	if ontologyKind != "" {
		k := ontology.Kind(strings.ToLower(ontologyKind))
		if _, ok := o.Root(k); !ok {
			return fmt.Errorf("unknown kind %q: want one of entity, relation, concept, role", ontologyKind)
		}
		kinds = []ontology.Kind{k}
	}

	w := os.Stdout
	fmt.Fprintf(w, "doppel ontology v%s\n", ontology.Version)
	for _, kind := range kinds {
		root, ok := o.Root(kind)
		if !ok {
			continue
		}
		fmt.Fprintf(w, "\n%s\n", strings.ToUpper(string(kind)))
		printTerm(w, o, root, 0)
	}

	errs := o.Validate()
	fmt.Fprintln(w)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "  violation: %v\n", err)
		}
		return fmt.Errorf("ontology is inconsistent: %d axiom violations", len(errs))
	}
	fmt.Fprintf(w, "ontology valid: %d terms\n", len(o.Terms()))
	return nil
}

// printTerm renders one subtree. Children come back sorted from the ontology,
// so the output is stable between runs.
func printTerm(w io.Writer, o *ontology.Ontology, id ontology.TermID, depth int) {
	term, ok := o.Get(id)
	if !ok {
		return
	}

	indent := strings.Repeat("  ", depth)
	line := indent + string(term.ID)
	if term.Abstract {
		// Abstract terms classify other terms but are never asserted of a real
		// function, so they can appear in a score's explanation and never in a
		// function's tags.
		line += " (abstract)"
	}
	if term.Weight > 0 {
		line += fmt.Sprintf("  weight %.4g", term.Weight)
	}
	if term.Domain != "" || term.Range != "" {
		line += fmt.Sprintf("  %s → %s", term.Domain, term.Range)
	}
	if term.Inverse != "" {
		line += fmt.Sprintf("  inverse %s", term.Inverse)
	}
	fmt.Fprintln(w, line)

	if ontologyDefs && term.Def != "" {
		fmt.Fprintf(w, "%s    %s\n", indent, term.Def)
	}

	for _, child := range o.Children(id) {
		printTerm(w, o, child, depth+1)
	}
}

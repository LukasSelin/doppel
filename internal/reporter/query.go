package reporter

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/parser"
	"github.com/LukasSelin/doppel/internal/retriever"
)

// QueryMatch is one corpus function related to a query probe: the function,
// its architectural context, the retrieval evidence connecting it to the
// probe, and how much of the probe's call-graph neighborhood it inhabits.
type QueryMatch struct {
	Unit      parser.CodeUnit
	Doc       concepter.ConceptDoc
	Candidate retriever.Candidate
	Locality  float64
}

// QueryMeta is the run context a query report states up front.
type QueryMeta struct {
	CorpusFuncs   int
	ResolvedCalls int // the probe's resolved internal callees; 0 disables locality
}

// PrintQuery renders one probe's related functions.
//
// The framing is deliberately asymmetric, unlike Print's A/B pairs: the probe
// is the question and the matches are the answer, so the probe is described
// once and each match names only itself. Evidence and locality print
// unblended — the ranking key that combined them ordered this list and is
// nowhere shown, per the same rule the analyze report follows.
func PrintQuery(w io.Writer, probe parser.CodeUnit, probeDoc concepter.ConceptDoc, matches []QueryMatch, meta QueryMeta) {
	name := probe.Name
	if probe.Package != "" {
		name = probe.Package + "." + name
	}
	fmt.Fprintf(w, "query: %s", name)
	if list := conceptList(probe.Concepts); list != "" {
		fmt.Fprintf(w, " — concepts: %s", list)
	}
	fmt.Fprintln(w)
	if probeDoc.Role != "" {
		fmt.Fprintf(w, "  role: %s", probeDoc.Role)
		if meta.ResolvedCalls > 0 {
			fmt.Fprintf(w, "   resolved calls: %d", meta.ResolvedCalls)
		}
		fmt.Fprintln(w)
	}
	if meta.ResolvedCalls == 0 {
		fmt.Fprintln(w, "  locality: no resolved calls in the snippet, so all locality reads 0.00")
	}

	if len(matches) == 0 {
		fmt.Fprintf(w, "\nNo related functions found among %d corpus functions. Nothing here shares informative structure, concepts, or calls with this snippet.\n", meta.CorpusFuncs)
		return
	}
	noun := "functions"
	if len(matches) == 1 {
		noun = "function"
	}
	fmt.Fprintf(w, "\nCorpus: %d functions. %d related %s:\n", meta.CorpusFuncs, len(matches), noun)

	for i, m := range matches {
		loc := fmt.Sprintf("%s:%d", filepath.ToSlash(m.Unit.File), m.Unit.StartLine)
		mname := m.Unit.Name
		if m.Unit.Package != "" {
			mname = m.Unit.Package + "." + mname
		}
		c := m.Candidate
		fmt.Fprintf(w, "\n#%d  %s  %s\n", i+1, mname, loc)
		fmt.Fprintf(w, "    evidence: %.1f nats (shape %.1f, concept %.1f, call %.1f)  code-shape: %.2f  locality: %.2f\n",
			c.Total, c.Shape, c.Concept, c.Call, c.Breakdown.Score, m.Locality)
		if list := conceptList(m.Unit.Concepts); list != "" || m.Doc.Role != "" {
			fmt.Fprint(w, "   ")
			if list != "" {
				fmt.Fprintf(w, " concepts: %s  ", list)
			}
			if m.Doc.Role != "" {
				fmt.Fprintf(w, " role: %s", m.Doc.Role)
			}
			fmt.Fprintln(w)
		}
		if len(c.Chains) > 0 {
			fmt.Fprintln(w, "    shared structure:")
			for _, ch := range c.Chains {
				times := ""
				if ch.Count > 1 {
					times = " ×" + strconv.Itoa(ch.Count)
				}
				fmt.Fprintf(w, "      %.2f  %s%s\n", ch.Energy, ch.Render, times)
			}
		}
	}
}

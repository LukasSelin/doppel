package dashboard

import "io"

// AuditSchema is the audit payload's version, independent of Schema and
// TimelineSchema for the reason TimelineSchema gives: a page written against
// one payload must fail loudly rather than half-render another.
const AuditSchema = 1

// AuditPayload is one comparison between two runs, read as a review: what a
// session, a branch or a pull request did to the duplication surface.
//
// It is the two-run page the timeline's single transition already is, said in
// the order a reviewer acts on it rather than the order a history is stepped
// through — merge-worthy pairs introduced first, then the ones removed, then
// what happened to each function. That is why it reuses StepChange rather than
// defining a second change shape: the findings are identical, only the reading
// differs, and two row types would be two renderings of one fact to keep in
// step.
//
// Like the timeline payload it carries findings and never the evidence to
// re-derive them. Every class, score, verdict and explanation sentence was
// settled by internal/identity or stored by the run that held the pair. No
// maps (TestAuditPayloadHasNoMaps), and every bound is stated on the page.
type AuditPayload struct {
	Schema int    `json:"schema"`
	Target string `json:"target"`

	// Base and Head name the two runs as the caller named them — by default
	// the snapshot files' base names.
	Base string `json:"base"`
	Head string `json:"head"`

	// Params is the head run's operating point, rendered for the reader. A
	// mismatch with the base run is not refused here — identity allows it and
	// records it in Notes — so the page shows the notes beside it.
	Params string   `json:"params"`
	Notes  []string `json:"notes,omitempty"`

	OldFunctions int `json:"oldFunctions"`
	NewFunctions int `json:"newFunctions"`

	// Change is the transition from Base to Head, bounded exactly as a
	// timeline step is.
	Change StepChange `json:"change"`

	// The verdict counts, over the whole pair lists rather than the bounded
	// rows: the page's headline must not shrink because a list was capped.
	CreatedMergeWorthy    int `json:"createdMergeWorthy"`
	DissolvedMergeWorthy  int `json:"dissolvedMergeWorthy"`
	DissolvedAttributable int `json:"dissolvedAttributable"`

	// Units locates every key a listed pair names, so the page can cite a
	// file:line beside it. Sorted by key; a created pair's sides are read from
	// the head run and a dissolved pair's from the base, head winning where a
	// key names a unit in both.
	Units []AuditUnit `json:"units,omitempty"`

	Bounds AuditBounds `json:"bounds"`
}

// AuditUnit is one function's location, as the run that held it recorded it.
type AuditUnit struct {
	Key  string `json:"key"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// AuditBounds is what the payload left out, so the page can say so.
type AuditBounds struct {
	MaxChanges int `json:"maxChanges"`
	MaxPairs   int `json:"maxPairs"`

	// ReportTop and ReportMaxPerFunc are the analysis-time presentation caps
	// the head run was written under. Non-zero means each snapshot stored the
	// ranked report list rather than the full candidate set, so a pair can
	// appear or vanish here because a cutoff moved — see TimelineBounds.
	ReportTop        int `json:"reportTop"`
	ReportMaxPerFunc int `json:"reportMaxPerFunc"`
}

var auditPage = page{
	name:  "audit",
	shell: "audit.html",
	css:   []string{broadsheetCSS, appCSS, "audit.css"},
	js:    []string{uiJS, "audit.js"},
}

// PrintAudit writes an audit payload as one self-contained HTML document,
// under exactly the same contract as Print and PrintTimeline.
func PrintAudit(w io.Writer, p AuditPayload) error {
	return printPage(w, auditPage, p.Target, p)
}

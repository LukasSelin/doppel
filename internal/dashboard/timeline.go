package dashboard

import (
	"encoding/json"
	"io"
)

// TimelineSchema is the timeline payload's version, independent of Schema.
//
// The two pages share a renderer and nothing else. A single-run payload and a
// series payload answer different questions, and a page written against one
// must fail loudly rather than half-render the other.
const TimelineSchema = 1

// TimelinePayload is a series of analysis runs, as the timeline page receives
// it.
//
// It is a sibling of Payload, not an extension of it, and the distinction is
// load-bearing. Payload's own contract says identity is a per-run index and
// that nothing crosses runs — which is right, and is exactly why a time axis
// could not be added to it. Here identity is a snapshot *key*, the only thing
// that survives a revision, and every cross-run judgment was already made by
// internal/identity before the payload was built. This type carries findings,
// never the evidence to re-derive them.
//
// The two inherited rules still hold: no maps anywhere (determinism,
// TestTimelinePayloadHasNoMaps), and every bound is stated on the page rather
// than applied silently.
//
// # What this payload deliberately does not carry
//
// Source bodies. They are the only part of the single-run payload that scales
// with corpus size rather than with findings, and N revisions of them would
// dominate the file outright. The page cites file:line and leaves bodies to the
// per-revision dashboard, which already renders them.
type TimelinePayload struct {
	Schema int    `json:"schema"`
	Target string `json:"target"`

	// Params is the operating point every step in the series was analysed at,
	// rendered for the reader. It is one string because the command refuses a
	// series whose steps disagree about it — see cmd/timeline.go.
	Params string `json:"params"`

	Steps   []Step          `json:"steps"`   // in series order; Step.Index == its position
	Changes []StepChange    `json:"changes"` // len(Steps)-1; Changes[i] is Steps[i] to Steps[i+1]
	Tracks  []TimelineTrack `json:"tracks"`

	// Notes are the mismatches the matcher allowed across the series —
	// ontology or build differences between steps. Not warnings, but they
	// change how a reader should weigh a step, so they ride on the page.
	Notes  []string       `json:"notes,omitempty"`
	Bounds TimelineBounds `json:"bounds"`
}

// TimelineBounds is what the payload left out, so the page can say so.
//
// Every bound in this tool is reported. A timeline that silently dropped the
// tail of a step's findings would read as "that revision was quiet" — the one
// misreading a history page must not invite.
type TimelineBounds struct {
	// MaxChanges, MaxPairs and MaxTracks are the caps that were applied.
	MaxChanges int `json:"maxChanges"`
	MaxPairs   int `json:"maxPairs"`
	MaxTracks  int `json:"maxTracks"`

	// FlatTracks is how many function lifelines ran the whole series with
	// nothing but `unchanged` on them, and are therefore absent from Tracks.
	// They are the bulk of any real history and the least informative part of
	// it, which is the same argument identity's text report makes for
	// counting unchanged rather than listing it.
	FlatTracks int `json:"flatTracks"`

	// TracksOmitted is how many interesting tracks the MaxTracks cap still
	// held back after the flat ones were already excluded.
	TracksOmitted int `json:"tracksOmitted"`

	// ReportTop and ReportMaxPerFunc are the *analysis*-time presentation caps
	// every step in the series ran under, carried here because they bound the
	// pair half of this page in a way nothing on it could otherwise reveal.
	//
	// `analyze --format json` stores the ranked report list, not the full
	// candidate set: at the shipped defaults that is twenty pairs however large
	// the corpus, so a series produced without `--top 0 --max-per-func 0` shows
	// a pair list that barely moves and reads as a quiet history. Zero on both
	// means the snapshots carry the whole struct-min-filtered set and the pair
	// half is complete.
	ReportTop        int `json:"reportTop"`
	ReportMaxPerFunc int `json:"reportMaxPerFunc"`
}

// Step is one analysis run in the series: its label and its own header numbers.
//
// Every number here is corpus-relative in the way CLAUDE.md documents — the
// learned concept vocabulary, roles, habitat fit and the nearest-neighbour
// percentiles all move with the corpus, so a reader stepping through them is
// watching a measurement of each revision, not one measurement over time. The
// page states that rather than drawing a trend line through it.
type Step struct {
	Index int    `json:"index"`
	Label string `json:"label"` // the revision this run analysed, as the caller named it

	Functions   int `json:"functions"`
	Pairs       int `json:"pairs"`
	MergeWorthy int `json:"mergeWorthy"`
	Concepts    int `json:"concepts"` // size of the learned vocabulary

	// Compression is canonical AST nodes over distinct subtree shapes; 1.0
	// would be a corpus with no repeated structure anywhere. NNP50/NNP90 are
	// nearest-rank percentiles of each function's best code-shape score over
	// the NNScored functions retrieval actually paired — a recall-bounded
	// population, which is why NNScored travels beside them.
	Compression float64 `json:"compression"`
	NNScored    int     `json:"nnScored"`
	NNP50       float64 `json:"nnP50"`
	NNP90       float64 `json:"nnP90"`

	// UnusedSeeds are the seed concepts this revision grew no practice for.
	UnusedSeeds []string `json:"unusedSeeds,omitempty"`
}

// StepChange is everything that happened between two consecutive steps.
type StepChange struct {
	From int `json:"from"`
	To   int `json:"to"`

	// Counts carries all eight classes, always, zeros included — a stable
	// shape beats a compact one, the same choice identity.Result makes.
	Counts []ClassCount `json:"counts"`

	// Changes are the named findings, `unchanged` excluded and the list
	// bounded. ChangesTotal is how many there were before the bound.
	Changes      []ChangeRow `json:"changes"`
	ChangesTotal int         `json:"changesTotal"`

	// Created and Dissolved are the near-duplicate pairs these changes
	// brought into and out of the candidate set, in identity's own order:
	// attributable first, then merge-worthy, then score. Bounded, with the
	// pre-bound totals beside them.
	Created         []PairRow `json:"created,omitempty"`
	Dissolved       []PairRow `json:"dissolved,omitempty"`
	CreatedTotal    int       `json:"createdTotal"`
	DissolvedTotal  int       `json:"dissolvedTotal"`
	AttributableNew int       `json:"attributableNew"` // created pairs a classified change explains
}

// ClassCount is one identity class and how many findings carry it.
type ClassCount struct {
	Class string `json:"class"`
	Count int    `json:"count"`
}

// ChangeRow is one classified finding, flattened for the page.
//
// Old and New are snapshot keys — `package.Name`, disambiguated with @file.
// Both are lists because split and merged have several on one side; every
// other class has exactly one on each.
type ChangeRow struct {
	Class string   `json:"class"`
	Old   []string `json:"old,omitempty"`
	New   []string `json:"new,omitempty"`

	File string `json:"file,omitempty"` // the new side's file, or the old side's for a deletion
	Line int    `json:"line,omitempty"`

	// The evidence, so a reader can falsify the class by opening two files.
	// Zero on split and merged, where the per-member containment carries the
	// claim and this page shows the member list instead.
	Jaccard     float64 `json:"jaccard"`
	Containment float64 `json:"containment"`
	DigestEqual bool    `json:"digestEqual"`

	// The secondary facts the class precedence collapsed: a moved function
	// that was also renamed reports `moved` with NameChanged set.
	NameChanged    bool `json:"nameChanged,omitempty"`
	PackageChanged bool `json:"packageChanged,omitempty"`
}

// PairRow is one near-duplicate pair that appeared or vanished at a step.
//
// Nothing here is recomputed: the score, the verdict and the sentence are what
// the run that held the pair recorded. Explain names the canonicalization rules
// that fired on two specific bodies, and neither this package nor the one that
// produced the row has either body.
type PairRow struct {
	A string `json:"a"`
	B string `json:"b"`

	Score       float64 `json:"score"`
	Overlap     float64 `json:"overlap"`
	MergeWorthy bool    `json:"mergeWorthy"`
	Explain     string  `json:"explain,omitempty"`

	AClass string `json:"aClass,omitempty"`
	BClass string `json:"bClass,omitempty"`

	// Attributable is false when neither side was classified as anything but
	// unchanged — the pair moved because retrieval re-ranked around it, which
	// is corpus churn rather than a consequence of the revision. The page
	// separates the two rather than presenting churn as history.
	Attributable bool `json:"attributable"`
}

// TimelineTrack is one function's lifeline across the series.
//
// A track is joined only through one-to-one matches, so its steps are
// contiguous and its claim is checkable: at every step boundary it crosses,
// the matcher matched exactly one function to exactly one other. Splits and
// merges end a track rather than continuing arbitrarily into one part — see
// identity.Series for that argument.
type TimelineTrack struct {
	ID     int         `json:"id"`
	Points []TrackStop `json:"points"`

	// First and Last bracket the track, so the page can lay it out without
	// walking Points.
	First int `json:"first"`
	Last  int `json:"last"`

	// Fate is the class of the change that ended the track before the series
	// did: deleted, split or merged. Empty when it reaches the last step.
	Fate string `json:"fate,omitempty"`

	// Label is the track's most recent key, which is what a reader looks it
	// up by.
	Label string `json:"label"`
}

// TrackStop is one function at one step. Class is how it was reached from the
// step before, and is empty only at step 0.
type TrackStop struct {
	Step  int    `json:"step"`
	Key   string `json:"key"`
	Class string `json:"class,omitempty"`
}

// WriteTimelineJSON writes the payload as indented JSON.
//
// It lives here rather than in cmd because the payload is this package's type
// and its encoding is part of what the type promises. HTML escaping is left at
// encoding/json's default OFF for this surface — unlike the inlined copy inside
// the page, a standalone file is not sitting in a script element, and a machine
// consumer should read the keys a function actually has.
func WriteTimelineJSON(w io.Writer, p TimelinePayload) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(p)
}

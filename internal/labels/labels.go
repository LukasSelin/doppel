// Package labels is the schema of a labels file: verdicts on pairs of
// functions, written by a reviewer reading two bodies or by a session deciding
// what to do with a pair, and read by the measurement harness as ground truth.
//
// It is its own package because it has two sides that must agree on every
// word. internal/bench reads labels files and scores a ranking against them;
// `doppel verdict` writes one (.doppel/verdicts.json) at the moment a decision
// is made. One schema and one validator serve both, so a vocabulary added on
// one side cannot be missing on the other.
//
// It imports nothing from this module.
package labels

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
	"time"
)

// Label is one verdict on a pair. See internal/bench's doc.go for the file
// format and how a scorecard reads it.
type Label struct {
	A string `json:"a"`
	B string `json:"b"`
	// AFile and BFile pin a side to the unit declared in that file: a
	// slash-separated path relative to the corpus root, as the report prints
	// it. Optional, and required on both sides when A and B are the same
	// qualified name — two init functions, a helper copied between two
	// scripts — because a name alone cannot say which unit is meant.
	AFile string `json:"aFile,omitempty"`
	BFile string `json:"bFile,omitempty"`
	Class string `json:"class"`
	// Kind says why a false positive is not worth acting on (FPKinds). It is
	// optional, allowed only on false_positive labels, and never ranks or
	// asserts: it lets a scorecard report mean rank per failure mode, which
	// is what a change aimed at one mode has to be measured against.
	Kind string `json:"kind,omitempty"`
	Note string `json:"note"`
	// Weight is how much the label counts toward its class's mean rank. Zero
	// reads as EffectiveWeight's default: 1, or what the Decision implies. It
	// bends the two mean ranks and nothing else — presence counts, merge
	// recall and the three hard assertions take every label as one label, so
	// a weight can never hide a violation.
	Weight float64 `json:"weight,omitempty"`
	// Target is what acting on the pair would make: one generic over the two
	// types, a shared helper in some package, one function delegating to the
	// other. Free text; it stops "merge" from hiding several refactors.
	Target string `json:"target,omitempty"`
	// Blocker says why a real duplicate stays (Blockers): the counterpart of
	// Kind for merge and refactor labels. A false positive is not a
	// duplicate, so it takes a Kind; a merge nobody can carry out takes a
	// Blocker.
	Blocker string `json:"blocker,omitempty"`
	// Decision is what somebody did with the pair, when someone did. A label
	// without one is a reading of two bodies; a label with one is also a
	// revealed verdict.
	Decision *Decision `json:"decision,omitempty"`
	// Earlier holds decisions this one replaced, oldest first. A pair left
	// alone at one session and merged at a later one is information a single
	// verdict would erase.
	Earlier []Decision `json:"earlier,omitempty"`
}

// Decision records what somebody did with a pair and why.
type Decision struct {
	Outcome string `json:"outcome"` // Outcomes
	By      string `json:"by"`      // Deciders
	Date    string `json:"date"`    // YYYY-MM-DD
	// Commit is the commit the tree sat on when the decision was made. For a
	// merge that is the last commit still holding both functions, which is
	// the tree a scorecard can find the pair in. Empty when unknown.
	Commit string `json:"commit,omitempty"`
	Reason string `json:"reason"`
}

// Outcomes is the closed vocabulary of Decision.Outcome.
var Outcomes = []string{
	"merged",   // carried out: one body left, or both now delegate to one helper
	"left",     // looked at and kept as it is, for a stated reason
	"deferred", // judged worth doing, not done yet
}

// Deciders is the closed vocabulary of Decision.By.
var Deciders = []string{"user", "agent"}

// Blockers is the closed vocabulary of Label.Blocker.
var Blockers = []string{
	"no-home",         // no package both sides could import is a natural home
	"layering",        // a dependency rule forbids the import the merge needs
	"separate-module", // separate modules or binaries with no shared package
	"too-small",       // the copy costs less than the helper would
	"design",          // kept apart on purpose: paired renderers, per-type APIs
	"convention",      // the repository's way of writing this kind of function
	"out-of-scope",    // real, but not this change's business
	"diverged",        // the difference between the two is correct and load-bearing
	"hot-path",        // performance-critical code kept specialised
	"other",
}

// Classes is the closed vocabulary of Label.Class, in the order a scorecard
// logs it.
//
// merge, refactor and false_positive are verdicts on what to do with a pair.
// coupled is not: it says the two are kept in step — the same change landed
// in both, in separate commits — and nothing about whether they should be one
// function, so it is scored for rank and never asserted on. It is what the
// history labeller emits for a co-change, because mirror and lifecycle pairs
// (Create/Delete, Encode/Decode) co-change exactly as clones do, and a hand
// review calls those false positives.
var Classes = []string{"merge", "refactor", "false_positive", "coupled"}

// FPKinds is the closed vocabulary of Label.Kind, in the order a scorecard
// logs it.
var FPKinds = []string{
	"mirror",            // inverse operations: encode/decode, read/write, min/max
	"entrypoint",        // main/init/run boilerplate, flag registration
	"separate-programs", // two binaries with no shared home to merge into
	"skeleton",          // shared driver scaffold around different payloads
	"already-factored",  // thin wrappers already delegating to one helper
	"accessor-family",   // parallel trivial accessors/builders across types
	"vocabulary",        // shared calls or vocabulary, unrelated logic
	"other",
}

// The weights a decision implies on a merge or refactor label that sets none.
// They are a judgement, not a measurement: a merge somebody carried out is a
// stronger verdict than one reading of two bodies, and one somebody looked at
// and left is a weaker one. An explicit Weight always wins.
const (
	MergedWeight = 3
	LeftWeight   = 0.5
)

// EffectiveWeight is the label's contribution to its class's and kind's mean
// rank: its Weight when set, else what its Decision implies for a merge or
// refactor label, else 1. A false positive somebody left alone is a
// confirmation, not a doubt, so a decision never lowers one.
func (l Label) EffectiveWeight() float64 {
	if l.Weight != 0 {
		return l.Weight
	}
	if l.Decision != nil && (l.Class == "merge" || l.Class == "refactor") {
		switch l.Decision.Outcome {
		case "merged":
			return MergedWeight
		case "left":
			return LeftWeight
		}
	}
	return 1
}

// File is one corpus's worth of labels.
type File struct {
	Corpus     string `json:"corpus"`
	Reviewed   string `json:"reviewed"`
	Population string `json:"population"` // include (default when empty) | exclude | only
	// Tree is the commit the labels describe. A label is a statement about
	// the code at one commit: a pair merged since then survives as two
	// wrappers that a ranking rightly buries, and scoring the old label
	// against the new tree reads that as a badly ranked merge.
	Tree   string  `json:"tree,omitempty"`
	Labels []Label `json:"labels"`
}

// PairKey is the canonical unordered identity of a pair of sides.
func PairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "\x00" + b
}

// SideID is one side of a label as an identity: the qualified name, or the
// name and its file when the label pins one.
func SideID(name, file string) string {
	if file == "" {
		return name
	}
	return name + "@" + file
}

// Key is the label's unordered identity, files included.
func (l Label) Key() string {
	return PairKey(SideID(l.A, l.AFile), SideID(l.B, l.BFile))
}

// Pair renders the label's two sides for a log line or a violation message.
func (l Label) Pair() string {
	return SideID(l.A, l.AFile) + " / " + SideID(l.B, l.BFile)
}

// ValidFile reports whether f is a clean, relative, slash-separated path.
func ValidFile(f string) bool {
	return f != "" && !strings.Contains(f, "\\") && !path.IsAbs(f) &&
		path.Clean(f) == f && f != "." && !strings.HasPrefix(f, "../")
}

// Parse decodes and validates a labels file. Duplicate pairs (reversed counts
// as duplicate), unknown classes, kinds, blockers, outcomes or deciders,
// unknown populations, empty names, malformed files, a same-name pair without
// two files to tell its sides apart, and a decision that contradicts its
// label are all hard errors — a labels file is ground truth, and a malformed
// one must fail loudly rather than silently score fewer pairs.
func Parse(data []byte) (File, error) {
	var lf File
	if err := json.Unmarshal(data, &lf); err != nil {
		return lf, err
	}
	if lf.Corpus == "" || lf.Reviewed == "" || len(lf.Labels) == 0 {
		return lf, fmt.Errorf("labels file needs corpus, reviewed, and at least one label")
	}
	switch lf.Population {
	case "":
		lf.Population = "include"
	case "include", "exclude", "only":
	default:
		return lf, fmt.Errorf("invalid population %q: want include, exclude, or only", lf.Population)
	}
	seen := map[string]bool{}
	for i, l := range lf.Labels {
		if err := l.Validate(); err != nil {
			return lf, fmt.Errorf("label %d: %w", i, err)
		}
		k := l.Key()
		if seen[k] {
			return lf, fmt.Errorf("label %d: duplicate pair %s", i, l.Pair())
		}
		seen[k] = true
	}
	return lf, nil
}

// Validate checks one label on its own: every vocabulary, every file pin, and
// that its decision agrees with its class.
func (l Label) Validate() error {
	if !slices.Contains(Classes, l.Class) {
		return fmt.Errorf("invalid class %q", l.Class)
	}
	if l.Kind != "" {
		if l.Class != "false_positive" {
			return fmt.Errorf("kind %q on a %s label; kind describes false positives only", l.Kind, l.Class)
		}
		if !slices.Contains(FPKinds, l.Kind) {
			return fmt.Errorf("invalid kind %q", l.Kind)
		}
	}
	if l.Blocker != "" {
		if l.Class != "merge" && l.Class != "refactor" {
			return fmt.Errorf("blocker %q on a %s label; a blocker says why a merge or refactor stays undone", l.Blocker, l.Class)
		}
		if !slices.Contains(Blockers, l.Blocker) {
			return fmt.Errorf("invalid blocker %q", l.Blocker)
		}
	}
	if l.A == "" || l.B == "" {
		return fmt.Errorf("empty qualified name")
	}
	if l.Weight < 0 {
		return fmt.Errorf("negative weight %g", l.Weight)
	}
	for _, f := range []string{l.AFile, l.BFile} {
		if f != "" && !ValidFile(f) {
			return fmt.Errorf("file %q must be a clean slash-separated path relative to the corpus root", f)
		}
	}
	if l.A == l.B && (l.AFile == "" || l.BFile == "" || l.AFile == l.BFile) {
		return fmt.Errorf("both sides are %s; name two different files with aFile and bFile", l.A)
	}
	for _, d := range l.Earlier {
		if err := d.Validate(); err != nil {
			return fmt.Errorf("earlier decision: %w", err)
		}
	}
	if l.Decision == nil {
		return nil
	}
	if err := l.Decision.Validate(); err != nil {
		return fmt.Errorf("decision: %w", err)
	}
	real := l.Class == "merge" || l.Class == "refactor"
	switch l.Decision.Outcome {
	case "merged":
		if !real {
			return fmt.Errorf("decision merged on a %s label; a merged pair is a merge or a refactor", l.Class)
		}
		if l.Blocker != "" {
			return fmt.Errorf("decision merged with blocker %q; a merged pair was not blocked", l.Blocker)
		}
	case "left":
		if real && l.Blocker == "" {
			return fmt.Errorf("a %s left alone needs a blocker saying why (%s)", l.Class, strings.Join(Blockers, ", "))
		}
	}
	return nil
}

// Validate checks one decision's fields.
func (d Decision) Validate() error {
	if !slices.Contains(Outcomes, d.Outcome) {
		return fmt.Errorf("invalid outcome %q: want %s", d.Outcome, strings.Join(Outcomes, ", "))
	}
	if !slices.Contains(Deciders, d.By) {
		return fmt.Errorf("invalid decider %q: want %s", d.By, strings.Join(Deciders, ", "))
	}
	if _, err := time.Parse(time.DateOnly, d.Date); err != nil {
		return fmt.Errorf("date %q is not YYYY-MM-DD", d.Date)
	}
	if strings.TrimSpace(d.Reason) == "" {
		return fmt.Errorf("a decision needs a reason")
	}
	if d.Commit != "" && !isHex(d.Commit) {
		return fmt.Errorf("commit %q is not a hexadecimal object name", d.Commit)
	}
	return nil
}

func isHex(s string) bool {
	if len(s) < 4 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}

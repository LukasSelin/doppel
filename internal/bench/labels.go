package bench

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
)

// Label is one human verdict on a ranked pair. See doc.go for the file format.
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
}

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

// LabelsFile is one reviewed corpus's worth of labels.
type LabelsFile struct {
	Corpus     string  `json:"corpus"`
	Reviewed   string  `json:"reviewed"`
	Population string  `json:"population"` // include (default when empty) | exclude | only
	Labels     []Label `json:"labels"`
}

// pairKey is the canonical unordered identity of a labeled pair.
func pairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "\x00" + b
}

// sideID is one side of a label as an identity: the qualified name, or the
// name and its file when the label pins one.
func sideID(name, file string) string {
	if file == "" {
		return name
	}
	return name + "@" + file
}

// Pair renders the label's two sides for a log line or a violation message.
func (l Label) Pair() string {
	return sideID(l.A, l.AFile) + " / " + sideID(l.B, l.BFile)
}

// validLabelFile reports whether f is a clean, relative, slash-separated path.
func validLabelFile(f string) bool {
	return f != "" && !strings.Contains(f, "\\") && !path.IsAbs(f) &&
		path.Clean(f) == f && f != "." && !strings.HasPrefix(f, "../")
}

// ParseLabels decodes and validates a labels file. Duplicate pairs (reversed
// counts as duplicate), unknown classes, unknown populations, empty names,
// malformed files and a same-name pair without two files to tell its sides
// apart are all hard errors — a labels file is ground truth, and a malformed one
// must fail loudly rather than silently score fewer pairs.
func ParseLabels(data []byte) (LabelsFile, error) {
	var lf LabelsFile
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
		switch l.Class {
		case "merge", "refactor", "false_positive":
		default:
			return lf, fmt.Errorf("label %d: invalid class %q", i, l.Class)
		}
		if l.Kind != "" {
			if l.Class != "false_positive" {
				return lf, fmt.Errorf("label %d: kind %q on a %s label; kind describes false positives only", i, l.Kind, l.Class)
			}
			if !slices.Contains(FPKinds, l.Kind) {
				return lf, fmt.Errorf("label %d: invalid kind %q", i, l.Kind)
			}
		}
		if l.A == "" || l.B == "" {
			return lf, fmt.Errorf("label %d: empty qualified name", i)
		}
		for _, f := range []string{l.AFile, l.BFile} {
			if f != "" && !validLabelFile(f) {
				return lf, fmt.Errorf("label %d: file %q must be a clean slash-separated path relative to the corpus root", i, f)
			}
		}
		if l.A == l.B && (l.AFile == "" || l.BFile == "" || l.AFile == l.BFile) {
			return lf, fmt.Errorf("label %d: both sides are %s; name two different files with aFile and bFile", i, l.A)
		}
		k := pairKey(sideID(l.A, l.AFile), sideID(l.B, l.BFile))
		if seen[k] {
			return lf, fmt.Errorf("label %d: duplicate pair %s", i, l.Pair())
		}
		seen[k] = true
	}
	return lf, nil
}

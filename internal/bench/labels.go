package bench

import (
	"encoding/json"
	"fmt"
	"slices"
)

// Label is one human verdict on a ranked pair. See doc.go for the file format.
type Label struct {
	A     string `json:"a"`
	B     string `json:"b"`
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

// ParseLabels decodes and validates a labels file. Duplicate pairs (reversed
// counts as duplicate), unknown classes, unknown populations and empty names
// are all hard errors — a labels file is ground truth, and a malformed one
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
		k := pairKey(l.A, l.B)
		if seen[k] {
			return lf, fmt.Errorf("label %d: duplicate pair %s / %s", i, l.A, l.B)
		}
		seen[k] = true
	}
	return lf, nil
}

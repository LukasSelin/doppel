package bench

import (
	"encoding/json"
	"fmt"
)

// Label is one human verdict on a ranked pair. See doc.go for the file format.
type Label struct {
	A     string `json:"a"`
	B     string `json:"b"`
	Class string `json:"class"`
	Note  string `json:"note"`
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

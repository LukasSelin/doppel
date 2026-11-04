package bench

import "github.com/LukasSelin/doppel/internal/labels"

// The labels schema lives in internal/labels, which `doppel verdict` also
// writes through; these names keep the harness reading it as it always has.

// Label is one verdict on a ranked pair. See doc.go for the file format.
type Label = labels.Label

// LabelsFile is one reviewed corpus's worth of labels.
type LabelsFile = labels.File

// Classes and FPKinds are the closed vocabularies of Label.Class and
// Label.Kind, in the order a scorecard logs them.
var (
	Classes = labels.Classes
	FPKinds = labels.FPKinds
)

// ParseLabels decodes and validates a labels file; see labels.Parse.
func ParseLabels(data []byte) (LabelsFile, error) { return labels.Parse(data) }

// pairKey is the canonical unordered identity of a pair of sides.
func pairKey(a, b string) string { return labels.PairKey(a, b) }

// sideID is one side of a label as an identity.
func sideID(name, file string) string { return labels.SideID(name, file) }

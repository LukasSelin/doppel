package analyzer

import (
	"sort"

	"github.com/LukasSelin/doppel/internal/comparator"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/parser"
)

// SimilarPair holds two code units and their static similarity score.
type SimilarPair struct {
	A, B       parser.CodeUnit
	AIdx, BIdx int                            // positions in the units slice, for looking up parallel data
	Score      float64                        // composite fingerprint similarity, 0.0-1.0
	Breakdown  fingerprint.Breakdown          // per-component scores behind Score
	Evidence   *comparator.StructuralEvidence // populated by structural comparison stage; nil until then
	Retrieval  *Retrieval                     // multi-channel retrieval evidence; nil for FindSimilar-produced pairs
	Culture    []CultureNote                  // unusual concept realizations; nil when none — set by the pipeline
	Habitat    []HabitatNote                  // habitat misfits; nil when neither side misfits — set by the pipeline
	Profile    []ProfileNote                  // equilibrium concept profiles; nil when neither side qualifies
	Kind       *KindNote                      // what the pair is — interface implementations, a diverged copy; nil when unlabeled
}

// MergeWorthy is the whole merge verdict, and SimilarPair is the only type
// that can state it: enough shared architectural context, from the comparator,
// and enough shared code shape, from the fingerprint. False when Evidence is
// nil — a FindSimilar-produced pair has never been through the comparator, so
// nothing here has judged its context.
func (p SimilarPair) MergeWorthy() bool {
	return p.Evidence != nil && comparator.MergeWorthy(*p.Evidence, p.Score)
}

// ProfileNote is one side's concept-arena equilibrium: which concepts
// survived the competition for the function's evidence, and what kind of
// ecosystem they form. Profiles annotate; they never affect ranking.
type ProfileNote struct {
	Side      string        // "A" or "B"
	State     string        // dominance | coalition | conflict | weak
	Concepts  []ProfileMass // survivors, (Mass desc, Tag asc)
	Extinct   []ProfileMass // candidates that died; rendered only under --debug
	Rounds    int
	Converged bool
}

// ProfileMass is one concept's equilibrium mass.
type ProfileMass struct {
	Tag  string
	Mass float64
}

// HabitatNote flags one side of a pair as notably out of place in its own
// package: its features are far more surprising there than the package's
// norm tolerates. Habitat annotates; it never affects ranking.
type HabitatNote struct {
	Side        string           // "A" or "B"
	Package     string           // the unit's habitat
	Fit         float64          // 0-1, excess-strain Boltzmann factor
	PackageNorm float64          // the habitat's mean member fit, for contrast
	Channels    []HabitatChannel // per-channel surprise; rendered only under --debug
}

// HabitatChannel is one channel's contribution to a HabitatNote's strain.
type HabitatChannel struct {
	Name     string
	Surprise float64
}

// CultureNote flags one side of a pair as an unusual realization of a shared
// concept: it carries the tag, but its typicality sits far below the corpus
// norm for that concept. Culture annotates; it never affects ranking.
type CultureNote struct {
	Tag           string
	Side          string // "A" or "B"
	Typicality    float64
	ConceptMedian float64
	Convention    float64          // the concept's convention strength, for context
	Channels      []CultureChannel // per-channel typicality; rendered only under --debug
}

// CultureChannel is one channel's contribution to a CultureNote's typicality.
type CultureChannel struct {
	Name       string
	Typicality float64
}

// Retrieval carries the candidate-retrieval evidence for a pair: per-channel
// shared-information masses in nats and the channels that admitted it. It is
// a third quantity next to Score and Evidence.OverlapScore — evidence mass
// ranks the report, while the two similarity scores stay unblended.
type Retrieval struct {
	Shape      float64       // shared structural energy, Σ IC·min(count) over shared patterns
	Concept    float64       // shared tag information, Σ IC(LCS)
	Call       float64       // shared rare-call IDF mass
	Total      float64       // Shape + Concept + Call
	TrophicSim float64       // 2·SharedEnergy/(E_A+E_B): how much of their structure is shared
	CallSim    float64       // call-channel Dice: mutual fraction of informative call energy
	Channels   []string      // which retrieval channels admitted the pair
	Chains     []SharedChain // highest-energy shared structures, the explanation
}

// SharedChain is one shared high-level structure behind a pair's shape
// energy — where the match's weight actually comes from.
type SharedChain struct {
	Level  int
	Energy float64
	Render string
}

// FindSimilar compares every pair of function fingerprints and returns those
// above threshold, sorted by score descending, limited to topN results.
// Units whose body has fewer than minNodes AST nodes are excluded: trivial
// accessors match each other perfectly and drown out real candidates.
func FindSimilar(units []parser.CodeUnit, threshold float64, topN, minNodes int) []SimilarPair {
	// Collect the indices worth comparing once, rather than re-testing inside
	// the O(n^2) loop.
	var idx []int
	for i := range units {
		if units[i].Fingerprint.Nodes >= minNodes && units[i].Fingerprint.Nodes > 0 {
			idx = append(idx, i)
		}
	}

	var pairs []SimilarPair
	for a := 0; a < len(idx); a++ {
		for b := a + 1; b < len(idx); b++ {
			i, j := idx[a], idx[b]
			bd := fingerprint.Similarity(units[i].Fingerprint, units[j].Fingerprint)
			if bd.Score >= threshold {
				pairs = append(pairs, SimilarPair{
					A:         units[i],
					B:         units[j],
					AIdx:      i,
					BIdx:      j,
					Score:     bd.Score,
					Breakdown: bd,
				})
			}
		}
	}

	// Ties break on position so the report is byte-identical across runs.
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].Score != pairs[j].Score {
			return pairs[i].Score > pairs[j].Score
		}
		if pairs[i].AIdx != pairs[j].AIdx {
			return pairs[i].AIdx < pairs[j].AIdx
		}
		return pairs[i].BIdx < pairs[j].BIdx
	})

	if topN > 0 && len(pairs) > topN {
		pairs = pairs[:topN]
	}
	return pairs
}

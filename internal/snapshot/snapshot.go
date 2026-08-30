// Package snapshot captures one complete doppel analysis run as plain data, so
// that two runs can be compared.
//
// Nothing else in the pipeline has a notion of "a run": comparator compares two
// functions, analyzer compares two fingerprints, reporter renders one result. A
// Snapshot is the missing noun — everything an impact comparison needs about a
// corpus at a point in time, and deliberately nothing more. It is derived
// output, never an input to any score.
//
// Three rules govern the schema, all load-bearing:
//
// Determinism. The repo's invariant is that an unchanged tree produces
// byte-identical output, so every map is flattened into a sorted slice before it
// reaches JSON and every slice has a total order. encoding/json does sort map
// keys, so a map field would marshal deterministically too; sorted slices are
// used anyway so the ordering is visible in the type rather than an emergent
// property of the encoder, and so the Go code that reads a snapshot sees the
// same order the file does.
//
// Identity over position. A diff is only meaningful if a function can be
// recognised across two runs, so units and pairs are keyed by name and never by
// index. Indices shift when any file changes, which is exactly the case a diff
// exists to describe. For the same reason paths are stored relative to the
// analysis root and slash-separated: `doppel analyze .` and a hook run rooted at
// an absolute cwd must produce comparable snapshots.
//
// No wall-clock, no environment. A timestamp, hostname or absolute root inside
// a Snapshot would break byte-identical reproducibility the moment --format json
// exists. Those belong to the baseline file wrapper, which is never compared.
package snapshot

import (
	"encoding/binary"
	"encoding/hex"
	"hash/fnv"
	"math"
	"path/filepath"
	"sort"

	"github.com/LukasSelin/doppel/internal/analyzer"
	"github.com/LukasSelin/doppel/internal/canon"
	"github.com/LukasSelin/doppel/internal/concepter"
	"github.com/LukasSelin/doppel/internal/fingerprint"
	"github.com/LukasSelin/doppel/internal/ontology"
	"github.com/LukasSelin/doppel/internal/parser"
)

// Schema is the snapshot format version. Bump it whenever a field's meaning
// changes, so an older baseline is discarded rather than misread.
//
// 2 dropped every field no consumer read. A snapshot is diffed and rendered,
// never browsed, so a field nothing reads is pure weight in a file the Stop
// hook rewrites and re-parses on every turn.
//
// 3 changed what MergeWorthy asserts: the verdict now floors on code shape as
// well as architectural overlap, so a baseline written by an older build counts
// merge-worthy pairs a newer one would not. No field moved, which is exactly
// why the bump matters — the two files would otherwise compare cleanly and
// report a drop nobody caused.
//
// 4 extended Digest with the fingerprint's nesting-depth histogram: the same
// body hashes differently under schema 3 and 4, so cross-schema diffs would
// report every function as changed. The histogram itself is not stored — no
// consumer reads it back, and rule four still holds.
//
// 5 and 6 were assigned twice, on two development lines that diverged for a
// while and did not know about each other. On one line, 5 changed what Score
// means and 6 moved shape retrieval to Weisfeiler-Lehman labels; on the other,
// 5 replaced the tag list with graded concept memberships. Both meanings are
// recorded below because both landed, and a reader of an old baseline deserves
// to know that a file claiming "5" could have been written by either. 7 is the
// first version that carries both worlds at once, and the first one whose
// number means exactly one thing.
//
// 5 (shape line) changed what Score means and added Containment. Code shape is a
// corpus-weighted multiset Jaccard over Weisfeiler-Lehman label bags now, not
// Jaccard over token 3-grams, so the same untouched pair scores differently
// under schema 4 and 5. Worse than differently: the new score is
// corpus-relative, so a schema-4 baseline and a schema-5 run would disagree
// about pairs nobody edited. The bump turns that into an incomparability
// result rather than a delta full of movement no session caused.
//
// 6 (shape line) is four changes shipped as one bump. The shape retrieval channel moved to
// the same Weisfeiler-Lehman labels, retiring the multi-level pattern multiset
// it used to index — retrieval decides which pairs exist at all, so a schema-5
// baseline and a schema-6 run hold different candidate sets: pairs appear and
// vanish with no body having changed, which is precisely the movement Delta's
// Attributable bit exists to avoid claiming. Every Unit gained its own WL
// label bag (Unit.WL, see EncodeWLBag) and the Snapshot gained RuleSet — a bag
// is what lets a stored run's WL Jaccard and Containment be recomputed rather
// than merely replayed, and a diff across two different canon rule-set
// versions would be the schema-4-vs-5 failure again (the same pair of
// untouched bodies canonicalizing differently), so RuleSet joins Schema,
// Doppel, Ontology and Params as a fifth thing Diff refuses to compare across.
// And Pair gained Explain, the rule-attributed sentence about a pair — the one
// prose field on the struct, earning its bytes the way rule four demands:
// --format json is a documented payload with readers outside this process, and
// the sentence is the only place the canonicalizer's work on a pair is legible
// at all. Diff does not diff it, for the reason Reasons was dropped in schema
// 2: it restates facts about the corpus in English, and a delta reporting that
// a sentence changed would blame a session for a rewording.
//
// 5 (concept line) replaced the tag list with graded concept memberships.
// Concepts are learned from the corpus now rather than asserted by a rule
// table, so a schema-4 baseline's tags are names from a vocabulary that no
// longer exists — they would not compare, they would silently fail to match
// anything.
//
// 7 is the reconciliation: the first schema carrying the WL fields (RuleSet,
// Labels, Unit.WL, Pair.Containment, Pair.Explain, CorpusMetrics) and the
// learned concept shape (Unit.Concepts as graded memberships, UnusedSeeds)
// together. Nothing was dropped from either line. The bump is not cosmetic
// even for a reader who has both: a baseline written by either predecessor is
// missing half of what 7 records, and the halves it is missing are exactly the
// corpus-relative ones a diff must not guess at.
//
// 6 (language line) added Params.Languages. Which languages a run reads
// decides what the corpus *is*, so it belongs with TestsMode and Generated
// among the parameters that make two runs comparable or not. A schema-5
// baseline records no language at all, and defaulting it would assert
// something the older run never measured — so it is a version bump rather
// than a field with a fallback.
//
// 8 is the second reconciliation, and it is a numbering bump rather than a
// metric one. The two lines collided again: 6 means "graded concepts plus
// Languages" on one and "graded concepts plus WL retrieval" on the other, so a
// file claiming 6 or 7 is again missing half of what a run now records. What
// it is *not* is a change to what a stored quantity means. Moving the
// Weisfeiler-Lehman bag onto the neutral IR was arranged to leave the label
// vocabulary exactly where it was — same label_0 names, same recurrence, same
// canonical tree — so a schema-7 Unit.WL and a schema-8 one over the same Go
// body are byte-identical bags. The bump exists because Params gained
// Languages, which a schema-7 baseline cannot supply and this run must not
// invent: a run that read one language and a run that read four measured
// different corpora, and every corpus-relative number in the file follows from
// which one it was.
//
// 9 is a meaning change with no shape change, the same kind of bump as 3.
// Concept membership is decided on *coverage* now — the fraction of a
// function's own information a concept explains — rather than on an
// unnormalized sum of feature weights, and Unit.Concepts[].Conf is stated in
// that quantity too. Nothing about the file's shape moves: the same field
// holds the same kind of number in the same place. What moves is every
// membership and every confidence in it, corpus-wide and for reasons no
// session caused, which is exactly what would otherwise let a schema-8
// baseline and a schema-9 run compare cleanly and report drift nobody made.
// A unit also keeps at most Options.MaxMemberships concepts, so the lists are
// shorter as well as differently graded.
const Schema = 9

// Snapshot is one full analysis run.
//
// A snapshot intended for diffing is built from the retrieved candidate set
// before the report's presentation cutoffs — no top-N truncation, no
// per-function diversity cap, no struct-min filter. Those cutoffs drop a pair
// for reasons that have nothing to do with any edit, so applying them first
// would manufacture differences; they belong at render time.
//
// Removing them narrows the noise but does not eliminate it, and the schema is
// honest about which. Retrieval keeps each function's top-K neighbours per
// channel, weighted by corpus rarity, so pair membership is corpus-relative:
// adding code elsewhere can push a pair out of a function's channel budget
// without either body changing. That is why Diff separates a pair change it can
// attribute to a fingerprint that actually moved from one it cannot — see
// Delta.
type Snapshot struct {
	Schema    int        `json:"schema"`
	Doppel    string     `json:"doppel"`   // doppel build version
	Ontology  string     `json:"ontology"` // ontology.Version the run reasoned with
	RuleSet   string     `json:"ruleSet"`  // canon.Version the run canonicalized bodies with
	Params    Params     `json:"params"`
	Functions int        `json:"functions"`
	Concepts  []TagCount `json:"concepts"` // sorted by tag

	// UnusedSeeds are the seed concepts this corpus grew no practice for,
	// sorted. Concepts are learned per corpus, so "absent" cannot be derived
	// from a fixed vocabulary any more: the only fixed list left is the seeds,
	// and the ones that grew nothing are the honest answer to "does this
	// codebase already do X". Fourteen short strings at most.
	UnusedSeeds []string    `json:"unusedSeeds,omitempty"`
	Roles       []RoleCount `json:"roles"` // sorted by role
	Units       []Unit      `json:"units"` // sorted by key
	Pairs       []Pair      `json:"pairs"` // sorted by score desc, then a, then b

	// Labels is the corpus-wide Weisfeiler-Lehman label dictionary — every
	// distinct label carried by any Unit.WL in this run, sorted ascending,
	// encoded once by fingerprint.EncodeLabelDict. Every Unit.WL is indexed
	// against exactly this slice (see EncodeWLBagIndexed), so a consumer
	// decoding any unit's bag must decode this field first. It exists purely
	// so Unit.WL is affordable: see fingerprint's package doc for the
	// measurement that makes a shared dictionary necessary rather than a
	// nicety. Empty for a corpus with no bodies.
	Labels string `json:"labels,omitempty"`

	// CorpusMetrics carries the two T10 corpus-health numbers: hash-cons
	// compression of the canonical AST forest, and the nearest-neighbour
	// code-shape distribution over the retrieval union. Additive to this
	// schema version — see the type doc for what it does and does not claim.
	CorpusMetrics CorpusMetrics `json:"corpusMetrics"`
}

// CorpusMetrics is the plain-data mirror of cmd's fingerprint.ConsStats and
// NNStats — flattened scalars, no maps, so TestSchemaHasNoMaps holds without
// special-casing this field.
//
// Neither number changes a pair, a score, or a ranking: both are read-only
// summaries of the run, computed once and carried here so a consumer of
// --format json sees exactly what the markdown/HTML preamble states in
// prose.
type CorpusMetrics struct {
	// TotalNodes / UniqueSubtrees are corpus totals across every canonical
	// function body, hash-consed by exact structural equality (same node
	// kind, same children, recursively). TotalNodes / UniqueSubtrees is the
	// compression ratio, always >= 1.0 for a non-empty corpus.
	TotalNodes     int `json:"totalNodes"`
	UniqueSubtrees int `json:"uniqueSubtrees"`

	// NNTotal is every function in the run; NNScored is how many of them
	// appeared in at least one pair the retrieval union actually scored — a
	// recall-bounded population, NOT the result of an exhaustive
	// nearest-neighbour search. NNP50/NNP90/NNP99 are nearest-rank
	// percentiles (no interpolation) of the Scored functions' best
	// code-shape score, and NNAtOrAboveThreshold is how many of them already
	// clear the run's own (post-calibration) threshold.
	NNTotal              int     `json:"nnTotal"`
	NNScored             int     `json:"nnScored"`
	NNP50                float64 `json:"nnP50"`
	NNP90                float64 `json:"nnP90"`
	NNP99                float64 `json:"nnP99"`
	NNAtOrAboveThreshold int     `json:"nnAtOrAboveThreshold"`
}

// Params records the knobs a run used. Diff compares them because every doppel
// score is corpus-relative: a baseline taken at a different threshold or
// min-nodes is not an earlier answer to the same question, it is an answer to a
// different one.
type Params struct {
	Threshold  float64  `json:"threshold"`
	Top        int      `json:"top"`
	MinNodes   int      `json:"minNodes"`
	StructMin  float64  `json:"structMin"`
	ChannelK   int      `json:"channelK"`
	MaxPerFunc int      `json:"maxPerFunc"`
	TestsMode  string   `json:"testsMode"`
	Generated  string   `json:"generated"` // generated-file population: include, exclude, or only
	Calibrate  float64  `json:"calibrate"` // null admission rate the thresholds were derived at; 0 = fixed thresholds
	Languages  []string `json:"languages"` // the languages this run read, sorted; corpus-defining like TestsMode
}

// Equal reports whether two runs asked the same question.
//
// It exists because Params stopped being a comparable struct when Languages
// arrived, and the alternative — storing the language list as one joined
// string to keep == working — would have hidden a real fact behind a
// convenience. Comparability is the whole purpose of this type, so it gets an
// explicit definition rather than an incidental one.
func (p Params) Equal(q Params) bool {
	if p.Threshold != q.Threshold || p.Top != q.Top || p.MinNodes != q.MinNodes ||
		p.StructMin != q.StructMin || p.ChannelK != q.ChannelK ||
		p.MaxPerFunc != q.MaxPerFunc || p.TestsMode != q.TestsMode ||
		p.Generated != q.Generated || p.Calibrate != q.Calibrate {
		return false
	}
	if len(p.Languages) != len(q.Languages) {
		return false
	}
	// Both sides are stored sorted, so this is order-sensitive on purpose:
	// two runs that read the same languages produce the same list.
	for i := range p.Languages {
		if p.Languages[i] != q.Languages[i] {
			return false
		}
	}
	return true
}

// TagCount is one concept tag and how many units carry it.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// RoleCount is one structural role and how many units were classified into it.
type RoleCount struct {
	Role  string `json:"role"`
	Count int    `json:"count"`
}

// Unit is one function or method as this run saw it.
//
// Only what a consumer reads is kept. Key and Digest are corpus-independent —
// they depend on this function's own AST alone — and together they are the
// whole of what Diff may claim: Key recognises a function across runs, Digest
// is the exact "this body changed" bit. Package and Concepts feed the concept
// inventory, File and Line locate a finding for a human, and Line is display
// only: inserting anything above a function shifts it.
//
// Concepts are corpus-derived and graded, so a unit's list can move when code
// nobody touched moves — the same caveat Role carries, and the reason Delta
// claims nothing from them. Confidence is rounded to two decimals: the file is
// rewritten every turn by the Stop hook, and full float precision would be
// bytes of noise in a diff nobody reads at that resolution.
//
// Role is corpus-relative and no internal consumer reads it. It survives
// because `analyze --format json` documents it, not because anything here
// needs it — see the --format row in README.md before removing it.
//
// WL is this unit's Weisfeiler-Lehman label bag (fingerprint.Fingerprint.WL),
// compactly encoded by fingerprint.EncodeWLBagIndexed against the run's
// shared Snapshot.Labels dictionary — decode that first. It earns its bytes
// the same way Containment did at schema 5: it is what a consumer of
// --format json needs to recompute the WL Jaccard and Containment
// components of a stored pair, rather than merely reading the two floats
// the pipeline already computed. A unit whose Fingerprint has no body —
// Nodes == 0 — carries no bag and encodes to "", mirroring Digest's
// empty-string convention.
//
// Earlier schemas also carried Qualified, Exported, Receiver, Nodes, Callers
// and Callees. Nothing ever read them.
type Unit struct {
	Key      string    `json:"key"` // stable cross-run identity; see unitKeys
	Package  string    `json:"package"`
	Name     string    `json:"name"`
	File     string    `json:"file"` // relative to root, slash-separated
	Line     int       `json:"line"` // display only, never diffed
	Concepts []Concept `json:"concepts,omitempty"`
	Digest   string    `json:"digest"`       // fingerprint hash: the exact "body changed" bit
	Role     string    `json:"role"`         // corpus-relative; documented output only
	WL       string    `json:"wl,omitempty"` // encoded Weisfeiler-Lehman label bag, indexed against Snapshot.Labels
}

// Concept is one graded membership as this run learned it.
type Concept struct {
	ID   string  `json:"id"`
	Conf float64 `json:"conf"`
}

// Pair is one reported near-duplicate. A and B are Unit keys, ordered A < B so
// a pair has exactly one spelling and can be matched across runs.
//
// Every number here is corpus-relative now. Score used to be the exception —
// fingerprint.Similarity read two fingerprints and nothing else — but its
// shape component weights each shared structural label by ln(N/df) over this
// run's corpus, so it moves when the corpus does. Overlap has always been
// corpus-weighted through this run's concept information content, and
// MergeWorthy is a pair of thresholds over both.
//
// That is not an argument for dropping them; it is the reason Params equality
// and the Schema version gate a diff at all. Two runs over the same tree with
// the same params still agree exactly, which is the only claim a delta makes.
//
// Containment is stored because it is now part of the documented --format
// json payload, which is rule four's whole test: a field earns its bytes by
// having a reader. It is reported and never scored — not in the rank key, not
// in the merge verdict, and Diff does not diff it.
//
// Earlier schemas also carried the four fingerprint.Breakdown components and
// the evidence Reasons strings. Neither was ever read back: the text report
// renders both straight off analyzer.SimilarPair, never through a snapshot.
// The Reasons in particular were a quarter of a baseline's bytes — free-text
// English restating counts that move with corpus churn.
type Pair struct {
	A           string  `json:"a"`
	B           string  `json:"b"`
	Score       float64 `json:"score"`
	Containment float64 `json:"containment"` // reported, never scored or diffed
	Overlap     float64 `json:"overlap"`     // corpus-relative
	MergeWorthy bool    `json:"mergeWorthy"` // half corpus-relative
	Explain     string  `json:"explain"`     // annotation; never diffed
}

// Build assembles a Snapshot from one pipeline run.
//
// docs[i] must describe units[i] and pairs must carry AIdx/BIdx into units:
// that positional alignment is the pipeline's existing contract, and resolving
// it by name instead has already caused one silent-miss bug in this codebase.
// Build converts to names once, here, at the boundary where positions stop
// being meaningful.
//
// metrics is copied straight through: it is already the plain-data form
// cmd computed alongside everything else in Result, and — unlike every other
// argument here — carries no per-unit or per-pair identity for Build to
// resolve.
func Build(units []parser.CodeUnit, docs []concepter.ConceptDoc, pairs []analyzer.SimilarPair,
	tagCounts map[ontology.TermID]int, unusedSeeds []string, root, version string, p Params, metrics CorpusMetrics) Snapshot {

	keys := unitKeys(units, root)
	dict := labelDict(units)

	s := Snapshot{
		Schema:        Schema,
		Doppel:        version,
		Ontology:      ontology.Version,
		RuleSet:       canon.Version,
		Params:        p,
		Functions:     len(units),
		Concepts:      tagCountsOf(tagCounts),
		UnusedSeeds:   append([]string(nil), unusedSeeds...),
		Units:         make([]Unit, 0, len(units)),
		Pairs:         make([]Pair, 0, len(pairs)),
		Labels:        fingerprint.EncodeLabelDict(dict),
		CorpusMetrics: metrics,
	}

	roleCounts := make(map[string]int)
	for i, u := range units {
		var doc concepter.ConceptDoc
		if i < len(docs) {
			doc = docs[i]
		}
		roleCounts[doc.Role]++
		s.Units = append(s.Units, Unit{
			Key:      keys[i],
			Package:  u.Package,
			Name:     u.Name,
			File:     RelSlash(root, u.File),
			Line:     u.StartLine,
			Concepts: concepts(u.Concepts),
			Digest:   Digest(u.Fingerprint),
			Role:     doc.Role,
			WL:       fingerprint.EncodeWLBagIndexed(u.Fingerprint.WL, dict),
		})
	}
	s.Roles = roleCountsOf(roleCounts)
	sort.Slice(s.Units, func(i, j int) bool { return s.Units[i].Key < s.Units[j].Key })

	for _, pr := range pairs {
		a, b := keyAt(keys, pr.AIdx), keyAt(keys, pr.BIdx)
		if a > b {
			a, b = b, a
		}
		rec := Pair{A: a, B: b, Score: pr.Score, Containment: pr.Breakdown.Containment, Explain: pr.Explain}
		if pr.Evidence != nil {
			rec.Overlap = pr.Evidence.OverlapScore
			rec.MergeWorthy = pr.MergeWorthy()
		}
		s.Pairs = append(s.Pairs, rec)
	}

	// Sort by name, not by the AIdx/BIdx tie-break analyzer.FindSimilar uses.
	// Those are file-walk positions: stable within a run, but they shift the
	// moment a file is added, which would reorder the whole pair list in a diff
	// for no reason the user caused.
	sort.Slice(s.Pairs, func(i, j int) bool {
		if s.Pairs[i].Score != s.Pairs[j].Score {
			return s.Pairs[i].Score > s.Pairs[j].Score
		}
		if s.Pairs[i].A != s.Pairs[j].A {
			return s.Pairs[i].A < s.Pairs[j].A
		}
		return s.Pairs[i].B < s.Pairs[j].B
	})

	return s
}

// Digest hashes a fingerprint into a short hex string: the exact,
// corpus-independent "this body changed" bit, and the most attributable signal
// a diff has.
//
// All five hashed fingerprint fields are already sorted and deduped by
// fingerprint.Build, so the hash is deterministic without any extra ordering
// work. Depth is included because a nesting change is a body change — the
// token stream can be identical while the structure is not, and that is
// exactly the blindness the Depth histogram exists to remove. The zero
// fingerprint (a declaration with no body) digests to the empty string rather
// than to a hash value, mirroring the rule that a zero fingerprint never
// matches anything: two body-less units must not be reported as having
// identical bodies.
func Digest(fp fingerprint.Fingerprint) string {
	if fp.Nodes == 0 {
		return ""
	}
	h := fnv.New64a()
	var buf [8]byte
	write := func(v uint64) {
		binary.LittleEndian.PutUint64(buf[:], v)
		_, _ = h.Write(buf[:])
	}
	write(uint64(fp.Nodes))
	// Field separators keep the concatenation unambiguous, so that moving a
	// value between Flow and Shingles could not produce the same digest.
	write(^uint64(0))
	for _, s := range fp.Shingles {
		write(s)
	}
	write(^uint64(0))
	for _, f := range fp.Flow {
		write(uint64(f))
	}
	write(^uint64(0))
	for _, d := range fp.Depth {
		write(uint64(d))
	}
	write(^uint64(0))
	for _, t := range fp.Types {
		_, _ = h.Write([]byte(t))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// RelSlash is the snapshot's path rule, exported because it is the rule every
// other surface mirrors: a path relative to the analysis root, slash-separated,
// so a run rooted at an absolute cwd reads the same as one run at ".". cmd and
// reporter call it rather than keeping their own copies.
func RelSlash(root, path string) string {
	if root == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// labelDict collects the corpus-wide Weisfeiler-Lehman label dictionary:
// every distinct label carried by any unit's bag, sorted ascending. It is
// what makes Unit.WL affordable — see fingerprint's package doc — and it
// must be built before any unit is encoded, so that every bag's labels are
// guaranteed present in it (EncodeWLBagIndexed's whole contract).
//
// The intermediate set is a plain map: it never reaches JSON, and the value
// that does — dict — is a sorted, deduped slice, so nothing here can make an
// unchanged tree's output depend on map iteration order.
func labelDict(units []parser.CodeUnit) []uint64 {
	seen := make(map[uint64]struct{})
	for _, u := range units {
		for _, lc := range u.Fingerprint.WL {
			seen[lc.Label] = struct{}{}
		}
	}
	dict := make([]uint64, 0, len(seen))
	for label := range seen {
		dict = append(dict, label)
	}
	sort.Slice(dict, func(i, j int) bool { return dict[i] < dict[j] })
	return dict
}

// unitKeys assigns every unit a corpus-unique identity.
//
// concepter.QualifiedName is "package.Name" over the package *clause*, which is
// unique for ordinary functions but not total: init may be declared repeatedly
// in a package, and two directories may share a package name. Colliding names
// get their file appended, which keeps the key stable when code moves within a
// file — the common edit — and unambiguous when it does not.
func unitKeys(units []parser.CodeUnit, root string) []string {
	counts := make(map[string]int, len(units))
	for _, u := range units {
		counts[concepter.QualifiedName(u)]++
	}
	keys := make([]string, len(units))
	for i, u := range units {
		qn := concepter.QualifiedName(u)
		if counts[qn] > 1 {
			keys[i] = qn + "@" + RelSlash(root, u.File)
			continue
		}
		keys[i] = qn
	}
	return keys
}

func keyAt(keys []string, i int) string {
	if i < 0 || i >= len(keys) {
		return ""
	}
	return keys[i]
}

func tagCountsOf(counts map[ontology.TermID]int) []TagCount {
	out := make([]TagCount, 0, len(counts))
	for tag, n := range counts {
		out = append(out, TagCount{Tag: string(tag), Count: n})
	}
	// Sort by tag, never by count: count ties are the common case and would
	// leave the order dependent on map iteration.
	sort.Slice(out, func(i, j int) bool { return out[i].Tag < out[j].Tag })
	return out
}

func roleCountsOf(counts map[string]int) []RoleCount {
	out := make([]RoleCount, 0, len(counts))
	for role, n := range counts {
		if role == "" {
			continue
		}
		out = append(out, RoleCount{Role: role, Count: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Role < out[j].Role })
	return out
}

// UnitByKey indexes the snapshot's units for lookup during a diff.
func (s Snapshot) UnitByKey() map[string]Unit {
	m := make(map[string]Unit, len(s.Units))
	for _, u := range s.Units {
		m[u.Key] = u
	}
	return m
}

// MergeWorthy counts the pairs the run judged worth merging.
func (s Snapshot) MergeWorthy() int {
	n := 0
	for _, p := range s.Pairs {
		if p.MergeWorthy {
			n++
		}
	}
	return n
}

// Key is the identity of a pair across runs.
func (p Pair) Key() string { return p.A + " <-> " + p.B }

// concepts copies a unit's memberships into the schema's plain form, rounding
// confidence to two decimals. The rounding is the storage rule, not a scoring
// one: nothing reads these back into a score, and two decimals is the
// resolution the digests are rendered at.
func concepts(cs []parser.Concept) []Concept {
	if len(cs) == 0 {
		return nil
	}
	out := make([]Concept, len(cs))
	for i, c := range cs {
		out[i] = Concept{ID: c.ID, Conf: math.Round(c.Confidence*100) / 100}
	}
	return out
}

# Doppel

Go CLI tool that detects structural similarities in a Go codebase to surface merge candidates — functions or methods that do similar enough work to warrant consolidation.

## Goal

Spot duplication that text-based tools miss. Doppel fingerprints each function from its AST and
cross-checks matches against call-graph context, so it finds pairs that share shape and role rather
than string overlap. The output is a ranked list of merge candidates with two scores and evidence
explaining *why* they are similar.

**The tool is fully self-contained.** No network, no LLM, no model, no cache. It was built around
Ollama embeddings originally; that entire pass was removed. Anything suggesting otherwise is stale.
The corollary worth internalizing: **every score is deterministic**, so an unchanged tree must
produce a byte-identical report. Treat a nondeterministic report as a bug (map iteration order is
the usual culprit — see `sortedKeys` in `mapper`, the tie-breaks in `retriever` (`topK`, union key
sort), and `analyzer.SortByEvidence`).

## Pipeline

Orchestrated end to end by `runAnalyze` in `cmd/analyze.go`. Stages in execution order:

1. **Walk & parse** — `filepath.WalkDir` + `shouldSkipDir`, then `parser.Parse` per `.go` file → `[]CodeUnit`. Unreadable files and parse errors are warned and skipped, never fatal. `fingerprint.Build` and `extractSignals` (the tagger's AST evidence) both run here, while the AST is still in hand.
2. **Tag** — `tagger.Tag(unit)` sets `unit.Patterns` (9 intent tags matched against the unit's AST signals — selectors, imports, string-literal contents, identifiers, node kinds — never against raw source text). Tag counts feed the corpus IC in the same loop.
3. **Build call graph** — `concepter.BuildCallGraph(units)` → `concepter.Graph`, both directions over **qualified names** (`package.Name`, methods keeping their receiver: `comparator.*Comparator.Compare`). A resolver maps each raw callee string to at most one unit: import-qualified selectors through the file's recorded import bindings (aliases included), variable-receiver method calls only when the method name is unique corpus-wide, bare names to the same-package function. Ambiguity drops the edge; recursion is excluded; only repo-internal edges exist. Happens *before* concept docs, because docs need caller lists.
4. **Generate + enrich concept docs** — `concepter.New()` makes bare docs; **`mapper.Map` does the real work**: attaches qualified callers, resolved internal callees, and the depth-2 call-graph neighborhood; derives per-corpus role thresholds from the resolved degree distribution and classifies; aggregates caller/callee patterns and packages from resolved edges. `culture.Build` then models the corpus's conceptual practice (see *Corpus culture*); its summary goes to stderr, and after the struct-min filter each surviving pair gets `Culture` notes for atypical realizations of its **shared** tags (positional attachment, like Evidence).
5. **Candidate retrieval** — `retriever.Retrieve` runs three per-function top-K channels (structural shingle-IDF, concept IC, resolved-call IDF — see *Candidate retrieval* below), unions and dedupes them, and computes definitive per-pair evidence masses plus the exact `fingerprint.Breakdown` for every union pair. Retrieval stats go to stderr. `cmd` materializes the candidates into `analyzer.SimilarPair`s (with `Retrieval` set). `analyzer.FindSimilar` still exists as the simple library API but the pipeline no longer calls it.
6. **Structural comparison** — a `comparator.Comparator` built over a corpus-weighted `ontology.Scorer` scores **every** candidate pair → `pair.Evidence`. Concept and role signals go through the ontology hierarchy, not string equality, and concept matching is weighted by information content computed from this run's tag counts — sharing a near-universal tag is weak evidence, sharing a rare one is strong.
7. **Structural filter** — when `--struct-min > 0`, pairs below that overlap score are **dropped**. This is a selection stage, not just annotation.
8. **Rank + report** — `analyzer.SortForReport` orders by corroborated evidence (`Total × OverlapScore × Score × TrophicSim²`, then code-shape, then `AIdx`/`BIdx`), applies the `--max-per-func` diversity cap greedily with backfill, and truncates to `--top`. `reporter.Print` to stdout always; `reporter.PrintMarkdown` to `--output` additionally. Both take a `reporter.Meta`; `--debug` adds per-pair retrieval provenance.

`docs[i]` describes `units[i]`, and `SimilarPair` carries `AIdx`/`BIdx` into that slice. Evidence
attachment is a positional lookup, deliberately — an earlier version keyed it on
`Package + "." + Name` while the call graph keyed on bare names, so lookups silently missed and
`--struct-min` then dropped the pair. Do not reintroduce name-keyed lookups between these stages.

## Module layout

```
main.go         Thin entry point → cmd.Execute()
cmd/            CLI commands (Cobra).
  root.go       rootCmd, Execute()
  analyze.go    Pipeline orchestrator; all flag registration
  config.go     .doppel.json loading (AnalysisConfig) and flag precedence
  ontology.go   doppel ontology: print the vocabulary, check its axioms
internal/
  parser/       parser.go is a thin dispatcher; go_parser.go does the go/ast work; signals.go extracts the tagger's evidence channels → CodeUnit
  fingerprint/  AST token shingles + control-flow histogram + signature types; the code-similarity score
  ontology/     The formal vocabulary: entity kinds, typed relations, concept taxonomy, roles, axioms
  tagger/       AST-signal intent detection → 9 pattern tags
  concepter/    ConceptDoc; callgraph.go (BuildCallGraph); role.go (ClassifyRole, role constants)
  mapper/       Where enrichment actually happens: callers, role classification, aggregated patterns/packages
  retriever/    Multi-channel candidate retrieval: shape.go / concept.go / calls.go inverted indexes, retriever.go union + evidence
  culture/      Corpus-culture model: ecology.go (PMI), prototype.go (prototypes + typicality), habitat.go (fit), convention.go
  analyzer/     SimilarPair + Retrieval types; FindSimilar (library API); SortByEvidence (final ranking)
  comparator/   Weighted structural overlap scoring (9 signals → 0.0–1.0 composite)
  reporter/     Plain-text (stdout) and Markdown (--output) formatting
```

Dependency directions that must hold: `analyzer` imports `comparator` (for the `Evidence` field), so
`comparator` must never import `analyzer`. `parser` imports `fingerprint`, so `fingerprint` must
never import `parser` — it works on `*ast.FuncDecl` directly. `ontology` imports nothing from this
module and must stay that way: `tagger`, `concepter` and `comparator` all depend on it. `retriever`
imports `parser`, `fingerprint`, `concepter`, `ontology` and must never import `analyzer` or
`comparator` — `cmd` bridges retriever candidates into `analyzer.SimilarPair`. `culture` imports
`parser`, `concepter`, `fingerprint` only (not even `ontology` — it is a leaf-tag count model) and
nothing imports it except `cmd`, which bridges its findings into `analyzer.CultureNote`.

## Two scores, deliberately unblended — and a third quantity that ranks

Each pair carries two independent similarity numbers, gated by two independent flags:

| Score | Source | Flag | Means |
| --- | --- | --- | --- |
| `Score` | `fingerprint.Similarity` | `--threshold` | how alike the two bodies are (reported as `code-shape:`) |
| `Evidence.OverlapScore` | `comparator.Compare` | `--struct-min` | how much architectural context they share |

Do not merge these into one number. High code score + low overlap is a *different finding* (lookalike
bodies in unrelated subsystems) from high on both (a real merge candidate), and collapsing them
destroys that distinction.

The report is **ranked by neither alone**: `analyzer.SortForReport` orders by **corroborated
evidence** — `Retrieval.Total × Evidence.OverlapScore × Score × TrophicSim²` — evidence mass
discounted by architectural corroboration, structural similarity, and squared trophic
similarity. Raw mass alone let verbose shared vocabularies (PDF drawing APIs) outrank a
self-documented production clone; adding overlap and code-shape fixed that but left
family-skeleton siblings (a shared compose-send prologue with large unshared bodies) beating
genuine family clones. Trophic² separates them: squared because the Dice, squared, approximates
the product of the two per-side shared fractions — one discount per side that does its own
thing. A linear trophic factor verifiably leaves a skeleton sibling within a fraction of a
percent of a true clone; the golden benchmark pins the separation. This is a *ranking key only*
— the displayed quantities stay unblended. A per-function diversity cap (`--max-per-func`, default 2) then bounds how many pairs
any one function fills, greedily in rank order with backfill; a suppression count goes to stderr.
`SortByEvidence` (plain `Retrieval.Total` ordering) remains the simple library API. The report
label was deliberately renamed from `score:` to `code-shape:` so nobody reads 1.0000 as a verdict.

## Development

`Taskfile.yml` ([go-task](https://taskfile.dev)) is the documented entry point — the PR template
and the pre-commit hook both assume it. Every task is a one-line wrapper, so raw `go` works too.

```bash
task setup    # git config core.hooksPath .githooks  — run this once after cloning
task build    # go build -o doppel .
task test     # go test ./...
task vet      # go vet ./...
task fmt      # gofmt -w .
```

Running the tool against this repo:

```bash
doppel analyze .
doppel analyze . --struct-min 0.4 --output report.md
doppel ontology --defs                                # print the vocabulary and check its axioms
```

**Hooks and CI:**

- `.githooks/pre-commit` checks `gofmt` on **staged** `.go` files, then runs `go vet ./...` across
  the whole module. It is only active after `task setup` — `core.hooksPath` is documented nowhere else.
- `.github/workflows/ci.yml` runs `go build/vet/test` on pushes and PRs to **`master`** (not `main`).
  Go version comes from `go-version-file: go.mod` (currently `1.25.0`).
- **CI does not check gofmt** — only the local hook does. This is why formatting drift is possible.
- `.gitattributes` forces LF for Go/shell/markdown/config so the bash hook works under Git Bash on Windows.

## Key types

- **CodeUnit** (`internal/parser/parser.go`) — one function/method from the AST: `Name`, `File`,
  `StartLine`, `Body`, `Signature`, `Package`, `Patterns`, `DocComment`, `Exported`, `ReceiverType`,
  `Callees`, `Fingerprint`. Methods are named `"*Server.Start"` — the receiver keeps its star.
- **Fingerprint** (`internal/fingerprint/fingerprint.go`) — `Shingles` (sorted, deduped 3-gram
  hashes), `Flow` (control-flow histogram), `Types` (normalized param/result types), `Nodes`, and
  `Patterns` (the multi-level trophic pattern multiset — see *Trophic structural energy*).
  `Shingles` still feeds the pinned `ast` Jaccard while `Patterns` feeds retrieval; the L0 overlap
  between them is deliberate — different dedup semantics, different consumers. The zero value
  means "no body" and never matches anything.
- **Term / Ontology** (`internal/ontology/ontology.go`) — the vocabulary: four disjoint rooted trees
  (`entity`, `relation`, `concept`, `role`) carrying definitions, relation weights, and `Validate()`.
  Concept leaf IDs are exactly the tagger's tag strings and role IDs exactly `ClassifyRole`'s return
  values, which is what let the ontology be introduced without changing any output.
- **ConceptDoc** (`internal/concepter/concepter.go`) — the architectural context the comparator
  scores. It is no longer rendered to text anywhere; `Format()` existed only to build embedding
  input and is gone.
- **SimilarPair** (`internal/analyzer/similarity.go`) — two `CodeUnit`s plus `AIdx`/`BIdx`, `Score`,
  `Breakdown` (per-component code similarity), `Evidence` (**nil until the structural
  comparison stage**), and `Retrieval` (**nil for `FindSimilar`-produced pairs** — set by the
  pipeline from retriever candidates).
- **StructuralEvidence** (`internal/comparator/comparator.go`) — the weighted overlap result.
- **Candidate / Stats / Options** (`internal/retriever/retriever.go`) — one retrieved pair with
  per-channel evidence masses, the run's channel statistics, and the retrieval knobs (the df caps
  live on `Options` so tests can shrink them; they are not flags).

### Candidate retrieval

Retrieval is an information-retrieval stage, not a similarity ranking: its job is recall — get every
pair with enough informative shared evidence in front of the comparator cheaply — and its unit is
**evidence mass in nats**, `Σ ln(N/df)` over shared rare features. All three channels share the
skeleton: build an inverted index, drop features with `df < 2` (can't pair) or `df > cap` (corpus
idiom, zero evidence), accumulate shared mass per neighbor in ascending feature order, keep each
function's top `--channel-k` (default 5) by `(mass desc, idx asc)`.

| Channel | Features | Cap (Options) | Extra gates |
| --- | --- | --- | --- |
| shape | multi-level trophic patterns (`fingerprint.Pattern`, presence-df IDF, min-count multiset mass) | `MaxPatternDF` 50 | `--min-nodes` eligibility; admits only pairs with exact `code-shape >= --threshold`, probing at most `4*ChannelK` neighbors |
| concept | tagger tags + non-root taxonomy ancestors (enumeration only) | `MaxConceptDF` 250 | none — evidence is `Scorer.SharedInformation` (raw `Σ IC(LCS)`) over the leaf tag sets |
| call | resolved internal callees (qualified) + import-qualified external calls via `RefPath` (full import path) | `MaxCallDF` 50 | none; bare names and variable-receiver calls are never tokens |

The union is deduped on `(min idx, max idx)`; every union pair then gets **definitive** evidence on
all three channels regardless of which admitted it, plus the exact `fingerprint.Breakdown`
(memoized). Summing the three masses into `Total` is coherent because all three are log-evidence
over the same corpus of N functions — do not normalize the components before summing.

Consequences worth knowing:

- A pattern/token in *every* unit has `idf = ln(N/N) = 0`; zero-mass neighbors are never admitted.
  The 130-clone `Error()` bucket exceeds the df cap entirely — those functions contribute no
  structural candidates and can only enter via concept/call evidence, which is the intended
  common-idiom suppression (no name-based heuristics anywhere).
- The concept channel indexes ancestors so `db_access`-only can meet `caching`-only through
  `data_store_access`, but the *evidence* is always `SharedInformation` on the leaf sets — a pair
  meeting only at a shallow ancestor earns only that ancestor's small IC.
- `ontology.Scorer.SharedInformation` exists precisely so retrieval never recomputes mass as
  `Σ ic.Of(m.LCA)` — that hits `Of("")` (the unknown sentinel) for unknown-term self-matches.
- `Stats.Suppressed` / `Stats.LargeBuckets` are diagnostics on stderr, not gates.
- On the large reference corpus (~8k functions): ~22k union pairs, a few seconds end to end (the
  old all-pairs scoring took ~20s), ~60% of pairs call-only, ~9% concept-only.

### Trophic structural energy

The shape channel's features are the **multi-level pattern multiset** extracted by
`fingerprint.extractPatterns` during `Build` (the AST exists only during parse): L0 token 3-gram
windows, L1 call/binary-operator shapes, L2 statements with salient structure
(`return(call:Sprintf)`, `defer(call:Close)`, `if(bin:!=(id,nil))` — nil/true/false keep their
names so the err-check idiom falls out with no special case), and L3 motifs — loop call summaries
covering header *and* body (`for{ call:Scan call:TrimSpace call:Atoi call:append }`, ≤ 8 callees)
and adjacent-statement bigrams (`seq[ assign:=(call:Atoi) ; if(bin:!=(id,nil)) ]`). For levels 1–3
the render string IS the hash serialization, so hash and explanation cannot drift; L2/L3 keep
their renders, L0/L1 do not.

Three quantities per pair, all from one sorted-intersection pass (`pairEvidence`):

- **Shape evidence** = `Σ idf·min(count)` over cap-surviving shared patterns — shared structural
  energy, the retrieval mass.
- **TrophicSimilarity** = `2·SharedEnergy / (E_A + E_B)`, reported as `trophic:`, where energy on
  both sides is **cap-surviving (informative) energy only**. Exact clones of a rich function read
  1.00; an idiom bucket whose every pattern exceeds the df cap reads 0/0 = 0.00 (`DataSourceName ↔
  Error`); everything between is the fraction of informative structure the pair shares. Two exact
  twins whose shape sits *between* df 2 and the cap legitimately read 1.00 with small energy — the
  energy ranks, trophic explains.
- **Shared chains** = the highest-energy shared L2/L3 patterns, `(energy desc, level desc,
  render asc)`, top `ChainTopN` (3 default, 20 under `--debug`) — rendered as the
  `shared structure:` block. A match has weight because of what it shares.

Trophic explains; it never ranks (`Total` stays Shape+Concept+Call) and never blends into
code-shape or overlap.

### Corpus culture

`internal/culture` models the repo's *local conceptual practice* from counts alone — the ontology
says what a concept is, culture says how this corpus normally expresses it. Built once per run by
`culture.Build(units, docs, cg, DefaultOptions())`; summarized on stderr as
`Culture: N concepts modeled, N associations, N unusual realizations`.

**Ecology** — PMI associations over unit-level binary co-occurrence, three kinds: tag~tag,
tag~role, tag~call (resolved call tokens, df ∈ [2, 50]). `PMI = ln(N·c(a,b)/(c(a)·c(b)))`.
Reported only when informative: positives need `count >= 3` and `PMI >= ln 2`; negatives need
`expected >= 3` and `count <= expected/2` (count 0 stores `PMI = -Inf`, rendered as "never" if
ever rendered). Ordering: positives by PMI desc, negatives by PMI asc, ties on (Kind, A, B).
**Associations are computed and pin-tested but deliberately unsurfaced per-pair** — an
association annotates the corpus, not a pair; a future `doppel culture` command is their home.

**Prototypes + typicality** — for each tag with **≥ 5 members**, five feature channels with
integer-percent weights (sum pinned at exactly 100): calls 40 (resolved call tokens, no df cap —
typicality measures normality, not rarity), flow 20 (binarized `FlowLabels`), cotags 15, role 15,
package 10. Channel typicality is **leave-one-out**: for member i with feature set F,
`T = Σ_{x∈F}(cnt(x)−1) / (|F|·(m−1))` — the mean over other members of `|F_i∩F_g|/|F_i|`,
computed with an integer numerator so it is order-independent. Empty F scores
`(#other empty members)/(m−1)` — doing nothing can be the norm, and no channel is ever skipped, so
no weight renormalization exists. A member never certifies its own normality; identical members
score exactly 1.0.

`Atypical(i,c) ⇔ median(c) > 0 && Typ(i,c) < 0.5·median(c)` — relative to the concept's own
median, so a legitimately diverse concept lowers its own bar and a tight concept can flag nobody.
Membership stays binary (the tag); typicality grades it. The report surfaces notes only on a
pair's **shared** tags ("you both claim transaction but B does it unlike anything else here" — the
drift-vs-duplication signal), one `culture:` line per note, per-channel detail under `--debug`.

### Habitats and conventions

The thermodynamic static layer, also in `internal/culture` (habitat.go, convention.go): how well a
function fits where it lives, and how strict each concept's practice is. Human vocabulary in
output (fit, surprise, convention); precise terms live here.

**Habitat** = a Go package with ≥ `MinHabitatMembers` (5) functions (empty package names are
skipped; smaller packages are silent). Channels are the culture channels minus `package`
(constant within a habitat): calls 44 / flow 22 / tags 17 / role 17 — integer weights summing to
exactly 100, pinned. **The smoothing identity is load-bearing**: leave-one-out counts plus one
pseudo-count collapse to the plain presence fraction, `P_i(x|h) = (cnt_-i(x)+1)/((m−1)+1) =
cnt(x)/m` — no surprisal is ever infinite, everything is integer counts over one denominator, and
a member never certifies its own normality beyond the pseudo-count.

- **Strain** = weighted mean feature surprisal (mean per channel, so richness is not penalized;
  an empty feature set scores the empty-set event — doing nothing can be the norm).
- **Temperature** = median member strain (one alien cannot heat the habitat enough to excuse
  itself).
- **Fit** = excess-energy Boltzmann factor: 1.0 when strain ≤ T; `exp(−(strain−T)/T)` above; the
  median member reads exactly 1.0 by construction. T = 0 (frozen habitat) makes any deviation
  fit 0.0 — branch order keeps all-identical habitats at 1.0.
- **Misfit** ⇔ strain > `MisfitFactor` (2.0) × T, or any positive strain when T = 0. At factor
  2.0 this is fit < e⁻¹. Only misfits produce `habitat:` report lines.
- **Norm** = mean member fit — the `package norm` contrast number and the stderr superlative
  ranking (median fit would be ≈ 1.0 always; the mean is dragged by outliers, which is the
  signal).

**Convention strength** per prototyped concept: 1 − the mass-weighted mean **Bernoulli** entropy
of feature presence across members, per channel, combined with the prototype's 40/20/15/15/10
weights. Bernoulli-per-feature rather than entropy over the mass distribution, deliberately: a
concept where every member does the same two things has uniform masses (maximal mass-entropy)
but zero presence-disorder — universal co-occurrence is unanimity, not diversity. Empty channels
score 1.0 (unanimity of absence). Shown as `convention` on culture notes; superlatives on stderr.

### The concept arena

`internal/culture/arena.go`: instead of five independent booleans, each function is an arena where
candidate concepts compete for its evidence under deterministic replicator dynamics, yielding an
equilibrium **concept profile** (masses summing to 1) and an **ecosystem state**. Everything is
corpus-derived — the PMI ecology is the arena's physics.

- **Candidates**: the unit's own tags ∪ concepts with a reported positive (PMI ≥ ln 2) tag~call
  association to any of its resolved call tokens ∪ positive tag~role associations to its role.
  Empty set → silence (`ArenaProfile` ok=false), not a state.
- **Evidence(f,c)** in nats, fixed order: tag presence → `IC(c) = ln(N/df)` (plain, matching the
  retriever idiom), + each positive tag~call PMI over the unit's tokens, + positive tag~role PMI.
  Deliberately not typicality-scaled (asymmetric for unprototyped concepts; future work).
- **Interactions**: reported TagTag PMI as-is; `-Inf` (never co-occur) maps to `−ln N`, the
  largest finite repulsion the corpus can express; unreported pairs 0; diagonal 0 (the
  replicator's own mass term carries self-reinforcement).
- **Dynamics** (consts, not Options: η 0.25, 64-round cap, 1e-9 convergence, 1e-6 extinction
  clamp): `x' ∝ x·exp(η·(F − maxF))` with `F = E + Σ M·x`, all in fixed ascending-candidate
  order; the max-shift makes overflow impossible and is scan-order independent; clamped concepts
  never revive.
- **States, pinned precedence**: **weak** (TotalEvidence < ln 2 — an equilibrium over noise is
  noise) → **conflict** (≥2 survivors with a reported-negative interaction — checked before
  dominance so the smell is never masked by a big top mass) → **dominance** (1 survivor or top ≥
  0.6) → **coalition**. Survivor floor 0.05 (max mass ≥ 1/9 so someone always survives).
- **Report**: `profile A: transaction 0.39  db_access 0.34 (coalition)` under the unit lines
  (the flat `tags:` line stays); extinct candidates + rounds under `--debug`; stderr
  `Ecosystems: N profiled (…)`. Profiles never rank.

On real corpora the flagship behavior is visible: units tagged `validation, db_access, mapping`
(fixture-string tags) equilibrate to `validation 1.00 (dominance)` — the unsupported tags go
extinct because they explain none of the surrounding evidence.

### Fingerprint scoring

`fingerprint.Similarity` blends three components; weights are constants and sum to exactly `1.00`.

| Component | Metric | Weight |
| --- | --- | --- |
| AST shingles | Jaccard over hashed 3-grams | 0.60 |
| Control flow | cosine over the node-kind histogram | 0.25 |
| Signature | Jaccard over normalized param/result types | 0.15 |

`SizeRatio` is reported in the `Breakdown` but **not** scored — Jaccard already penalizes size
mismatch through the union, so damping again would double-count it.

Two canonicalization rules do the heavy lifting in the token stream, and both are load-bearing:
identifiers collapse to `ID` (so renamed clones still match), while call *selector* names survive as
`CALL:Errorf` (intent-bearing) with the receiver expression dropped (`e`, `s`, `cfg` are arbitrary).

`--min-nodes` (default `12`) excludes tiny bodies from the **structural retrieval channel** (and
from `FindSimilar`). Without it one-line accessors match each other at 1.0 and flood the channel.
Concept and call retrieval deliberately ignore it — a small function with rare tag or call evidence
is still worth comparing.

### Comparator weights

Each weight lives on its relation term in `internal/ontology/relations.go`, not as a constant in the
comparator, so the scoring table and the vocabulary cannot drift apart. They sum to exactly `1.00` —
axiom 7 asserts it — and the composite is clamped to `1.0`.

| Signal | Relation | Weight | Graded |
| --- | --- | --- | --- |
| shared callees (raw, incl. stdlib) | `calls` | 0.210 | no |
| shared concepts | `exhibits` | 0.180 | **yes** |
| shared callers (resolved, qualified) | `called_by` | 0.120 | no |
| same role | `has_role` | 0.135 | **yes** |
| same package | `declared_in` | 0.090 | no |
| neighborhood overlap | `shares_neighborhood` | 0.030 | no |
| caller concepts | `called_from_concept` | 0.050 | **yes** |
| callee concepts | `calls_into_concept` | 0.050 | **yes** |
| same visibility | `has_visibility` | 0.045 | no |
| same receiver type | `bound_to` | 0.045 | **yes** |
| shared caller packages | `called_from_package` | 0.0225 | no |
| shared callee packages | `calls_into_package` | 0.0225 | no |

Weight provenance, two carves deep: the nine original weights were scaled uniformly by `0.9` for
the two concept-context signals (no judgment applied — a pair with no caller/callee concept overlap
scores ~10% lower, which moved two unrelated test-helper pairs out of merge-worthiness when it
landed); then `shares_neighborhood` took its `0.030` entirely from `calls` and `called_by`, because
a depth-2 neighborhood generalizes exactly what those two edges measure. That second carve is the
first change to relative order — `called_by` now sits below `has_role`, deliberately.

`MergeWorthy = OverlapScore >= 0.4 && countSignals >= 2`. **`countSignals` counts only 5 of the 12**
— callees, callers, concepts, role, package. Visibility, receiver, the two package-overlap signals
and the two concept-context signals raise the score but never the signal count: context is not by
itself a reason to merge two functions.

The concept signal counts when its *best single pairing* reaches `0.5`, not when the aggregate ratio
does. Thresholding the aggregate would be a regression rather than a guard — three tags with one
exact match average to `0.33`, so a pair that counts today would stop counting at an unchanged score.

Floor effect worth knowing: `SameVisibility` is true when both are unexported *or* both exported, and
the receiver signal scores `1.0` when both are plain functions. Any two plain unexported functions
therefore start with a free `0.09`.

### Roles

Fan-in counts resolved callers, fan-out counts **resolved internal callees** — stdlib and builtin
calls no longer inflate it. The pipeline classifies with `concepter.ClassifyRoleAt` against
per-corpus `RoleThresholds`: high on an axis means strictly above that axis's median resolved
degree, floored at 2, zero-degree units included in the median. On sparse graphs (most repos,
this one included) both thresholds sit at the floor and the adaptive branch is dormant — do not
simplify it away. `ClassifyRole` keeps the fixed threshold 2 for direct callers. The truth table
itself lives in `internal/ontology/roles.go`, so there is one definition to change when a fifth
role appears.

| | few callees (<2) | many callees (>=2) |
| --- | --- | --- |
| **few callers (<2)** | `leaf` | `orchestrator` |
| **many callers (>=2)** | `utility` | `passthrough` |

Roles decompose into two independent axes, `HighFanIn` and `HighFanOut`. The comparator scores two
roles as the Jaccard overlap of the axes on which **both** are high: `utility` vs `passthrough` is
`0.5` (both high fan-in), `orchestrator` vs `passthrough` is `0.5`, `leaf` vs `orchestrator` is
`0.0`. Agreement on a *low* axis deliberately scores nothing, which the caveat below explains.

Caveat: roles are now corpus-relative twice over — resolved degrees shift as code is added, and
the thresholds themselves follow the degree distribution — so a function's role (and the `SameRole`
merge signal) can move when unrelated code changes. That is what "high for this repo" means, and it
was chosen deliberately over a fixed absolute scale.

### Tagger patterns

Exactly 9, emitted in declaration order: `retry`, `http_call`, `db_access`, `validation`, `mapping`,
`transaction`, `caching`, `concurrency`, `error_wrapping`. The rules name `ontology` concept terms
rather than bare strings, so a rule pointing at a concept that does not exist stops compiling, and
`tagger_test` enforces the other direction: every concrete concept has exactly one rule.

Rules match **AST evidence** (`parser.TagSignals`), never raw source text, and each channel has its
own semantics: selectors exact (`http.Get`, `sync.Map`), methods exact on the method or bare-call
name, receivers exact on the receiver identifier (`tx.Commit` fires, `mtx.Lock` does not), imports
and string-literal contents and identifier names by substring, plus node-kind flags
(go/select/chan) for `concurrency`. Consequences worth knowing:

- A comment saying `COMMIT` or `DELETE` no longer tags anything — comments are not evidence.
- `error_wrapping` is deliberately tight: a `%w` verb anywhere in a format string (the old rule
  only matched `%w"`) or a pkg/errors wrap helper. Bare `fmt.Errorf` and `errors.As`/`errors.Is`
  no longer fire it, which makes the tag rare enough to be informative under IC.
- `retry` is the one tag with no structural handle — its evidence is lexical, in identifier names.
- String literals are still evidence, so a test whose fixture strings contain `%w` or `SELECT `
  earns those tags. A function carrying SQL strings is db-flavored even when it is a test.
- The pre-Go-only polyglot keywords (`axios`, `urllib`, `Promise.`, `await `) are gone.

### The ontology

`internal/ontology` is the vocabulary the comparator reasons over, and the reason a pair tagged
`http_call`/`db_access` no longer scores the same as a pair with nothing in common. The nine tags are
leaves of a taxonomy whose interior nodes are abstract and exist purely to relate them:

```
concept → io_operation → remote_io → http_call
                       → data_store_access → db_access, caching, transaction
        → data_transformation → mapping, validation
        → control_flow → concurrency, fault_tolerance → retry
        → error_handling → error_wrapping
```

Relatedness is Wu–Palmer, `2·depth(LCA) / (depth(a) + depth(b))`: identical `1.00`, siblings under
`data_store_access` `0.67`, cousins under `io_operation` `0.33`, different branches `0.00`. Sets of
tags are matched pairwise by a **global-descending greedy** — every candidate pairing sorted by score
and then by term name, consumed in that order. Walking one side greedily instead would let a merely
related tag consume an exact match and score *below* plain exact matching, which would let a pair
lose merge-worthiness purely by gaining a hierarchy. The tie-break on term name is equally
load-bearing: cross-branch pairings all score exactly `0.0`, so ties are the common case, and without
a total order the evidence lines would vary between runs.

Two empty tag sets score `0.0`, while two `leaf` roles score `1.0`. The opposite conventions are
deliberate — carrying no tags is not agreement, whereas two leaves are the same role — so the two
Jaccard-shaped functions must not be merged into one helper.

`ontology.Validate()` checks nine axioms and is exercised both by tests and by `doppel ontology`.
Axiom 8, the tagger/ontology correspondence, lives in `internal/tagger` instead: the check needs the
rule table, and importing `tagger` from `ontology` would be a cycle.

### Corpus-weighted relatedness

`ontology.NewCorpusIC` turns this run's tag counts into information content —
`IC(c) = ln(1/P(c))`, add-one smoothed, ancestors accumulating their leaves — and
`ontology.NewScorer` wraps the ontology with it. `cmd/analyze.go` builds the scorer from the tag
counts and hands it to `comparator.New`; the free `comparator.Compare` stays corpus-independent
(nil IC delegates literally to the Wu–Palmer methods) and is what the regression tests pin.

With IC loaded, term relatedness is **Lin** (`2·IC(LCS)/(IC(a)+IC(b))`) and set relatedness is
information-weighted: a matched pair contributes `IC(LCS)` exactly (Lin's denominator cancels the
pair weight), so the score is the fraction of the larger side's information that is shared. The
matcher is the same greedy **sorted by contribution, not similarity** — under IC a pair can be more
similar yet share less information, and contribution order is what stays optimal (verified against
a brute-force oracle in `oracle_test.go`/`scorer_test.go`, exhaustively).

Two invariants worth knowing:

- **The merge-signal gate never sees IC.** `countSignals` reads `PatternSignalBest`, a
  taxonomy-only Wu–Palmer best match. Under Lin, sibling/cousin similarities move with tag
  frequencies elsewhere in the tree; a pair's `MergeWorthy` must not flip because unrelated code
  shifted the statistics. (The `OverlapScore >= 0.4` half of the verdict does include the weighted
  score, so `MergeWorthy` is not fully corpus-independent — the signal count is.)
- **Singleton sets cannot be discounted.** `{error_wrapping}` vs `{error_wrapping}` is still 1.0 —
  the shared information and the total information are the same quantity. The discount only
  manifests in sets of ≥ 2 tags. Fixing it would mean IC-scaling the `exhibits` relation weight,
  which breaks axiom 7; documented instead.

## Configuration

`.doppel.json` at repo root, or `--config <path>`. A missing file is not an error; malformed JSON is.
**Keys are kebab-case** (they mirror the flag names), not snake_case:

```json
{
  "threshold": 0.65,
  "top": 10,
  "min-nodes": 12,
  "struct-min": 0.4,
  "output": "doppel-report.md",
  "channel-k": 5,
  "debug": false,
  "max-per-func": 2,
  "tests": "exclude"
}
```

Flag semantics after the retrieval redesign: `--threshold` floors code-shape for
**structural-channel admission only** (concept/call candidates bypass it); `--top` caps the
**final report** after comparison, filtering and evidence ranking — not the candidate set;
`--min-nodes` gates the structural channel only; `--channel-k` is the per-function per-channel
top-K; `--debug` adds retrieval provenance lines to the report; `--max-per-func` caps how many
final-report pairs any one function may appear in (0 disables); `--tests` picks the population
(`include`/`exclude`/`only`, default `exclude`) before any statistic is computed.

Every functional flag except `--config` has a config key. Precedence: `applyConfig` only calls
`Flags().Set` when `!Flags().Changed(name)`, so explicit CLI flags always win over the file.
Unknown keys are ignored rather than rejected, so a stale config file does not break a run.

## Conventions

- Go-only. All parsing uses `go/ast` — no external parsers, no multi-language support.
- No caches and no generated state files. If you find yourself adding one, that is a design change,
  not an optimization.
- Cobra is the only direct dependency. Keep it that way unless there is a strong reason.
- Skipped directories: `.git`, `.claude`, `vendor`, `testdata`, `build`, `.idea`, `.vscode`.
  `_test.go` files are always parsed; **`--tests` decides the population** (default `exclude`).
  Tests are conventionally similar by design, so they form their own population: `exclude`
  models production practice, `only` is test-suite hygiene mode, `include` mixes both but
  **cross test/prod pairs are never reported** (different build units are never merge
  candidates). The filter runs before tagging, so every corpus statistic — IC, dfs, culture,
  habitats, arenas — models exactly the population the report describes; filtering at report
  time instead would be the worst of both. `_test.go` is a compiler-recognized suffix, not a
  naming heuristic.
- Tested: `ontology`, `fingerprint`, `analyzer`, `comparator`, `tagger`, `parser`,
  `concepter/role`, `retriever`, `culture`, `reporter`, `cmd` config precedence. Untested and
  worth covering: `mapper`.
- **Golden ranking benchmark** (`internal/bench`): a corpus-agnostic, env-guarded harness scoring
  the final ranking against a human-reviewed labels file. Both inputs come from outside the repo
  (`DOPPEL_BENCH_CORPUS` + `DOPPEL_BENCH_LABELS`); **no corpus names, paths, or labeled pairs may
  ever be committed** — the labels JSON is a local artifact the user keeps. Run:
  `DOPPEL_BENCH_CORPUS=<corpus> DOPPEL_BENCH_LABELS=<labels.json> go test ./internal/bench/ -v`.
  One labeled semantic false positive keeps two assertions deliberately red (see rough edges).
  The harness analyzes the full population (`--tests include` semantics, cross pairs dropped)
  because label files may span test and production pairs.

## Rough edges

Known traps, documented so they aren't rediscovered. None are fixed:

- **Resolver imprecision without go/types.** Call-graph edges are resolved from the AST alone:
  a local variable shadowing an import name is mistaken for the package; an external import whose
  path base equals an internal package name can produce a false edge on a name coincidence; import
  paths whose base differs from the package clause (`yaml.v2`), dot imports, and ambiguous method
  names all fail toward a *missed* edge. Documented at the resolver in
  `internal/concepter/callgraph.go`; the cure for all of them is go/types, deliberately out of
  proportion for this tool.
- **Silent parse failures.** `parseGo` returns `nil, nil` on any syntax error, despite a comment
  claiming it returns partial results.
- **Wu–Palmer depth convention.** `ontology.Relatedness` puts the root at depth 0, so a pair whose
  least common ancestor is the root scores `0`. The textbook formulation puts the root at 1, and
  "correcting" the depths to match would hand every cross-branch pair a nonzero floor and inflate
  every structural score in the report. The redundant explicit guard in `Relatedness` exists to make
  that trap visible.
- **`remote_io` and `fault_tolerance` are unary.** A taxonomy node with one child adds no
  discriminative power and costs its leaf a level of depth, which *lowers* that leaf relatedness to
  everything. They are kept as the slot future siblings go in (`grpc_call`, `circuit_breaker`), so
  removing them is a scoring change rather than a simplification.
- **Adjacent pairs and the neighborhood signal.** Two directly-connected functions inherently
  share each other's 1-neighborhood inside their depth-2 balls, mildly favoring adjacent pairs on
  `shares_neighborhood`. The compared counterpart itself is excluded per pair (without that,
  adjacency would be *penalized* instead); the residual bias is accepted at weight 0.030.
- **Retrieval recall is bounded by the channels.** A pair with no shared rare shingle, no shared
  tag, and no shared resolved call is never compared, no matter how alike it is — that is the
  design trade (the old exhaustive `FindSimilar` pass remains available as a library call). The
  worst case of the inverted-index accumulation is `O(cap · postings)`, comfortably sub-quadratic
  at 10k functions (~2.5s on an 8.7k-function corpus, vs ~20s for the old all-pairs pass).
- **Arena equilibria are corpus-relative thrice over.** Tag dfs, association cutoffs, and the
  interaction matrix all move with unrelated code, so a function's profile and state can change
  when the corpus does. Convergence is capped at 64 rounds, not proven (small fitness gaps end
  `capped`, which the debug line shows honestly). A concept can only invade a function through a
  *reported* association — the ecology's cutoffs bound the arena's imagination. On the large
  reference corpus the states degenerate gracefully: ~98% dominance, few coalitions,
  conflict/weak ≈ 0 (any invaded
  candidate carries ≥ ln 2 evidence by construction, so the weak floor rarely triggers).
  Function-vs-function niches, invasion/speciation/overcrowding, dominant-set clustering, and
  Potts-style domains are named future work.
- **Habitat = package is crude.** One Go package can host several micro-habitats (handlers next
  to helpers next to tests — test functions in a production package legitimately read as
  misfits). Directory- or subsystem-level habitat rollup is future work, as are all the delta
  quantities: Δentropy/fragmentation, phase transitions, chemical potential (marginal duplication
  pressure), git-derived heat, inter-package JS divergence, free energy.
- **Convention entropy is a dispersion proxy, not realization clustering.** It measures how
  predictable each practice is across members, not how many distinct whole realizations exist.
  Habitat fit and misfit notes are corpus-relative like roles and typicality — a function's fit
  can move when unrelated code shifts its package.
- **Culture associations are computed but unsurfaced.** The ecology model (PMI associations) is
  built, tested, and reported only as a stderr count — per-pair surfacing was deliberately
  deferred because an association annotates the corpus, not a pair. A `doppel culture` command is
  the natural next home. Relatedly, an unusual realization on a function that appears in *no*
  retrieved pair is invisible in the report (stderr count only).
- **Typicality is corpus-relative, like roles.** A function's typicality — and whether a pair
  carries a culture note — can change when unrelated code shifts the concept's membership or the
  corpus norm. That is what "normal for this repo" means; same caveat as the role thresholds.
- **Nested loops double-count inner calls in loop summaries.** An inner loop's callees appear in
  both its own L3 summary and every enclosing loop's — each container is a real behavioral unit,
  and de-duplicating would cost a pass per nesting level for no scoring benefit. Accepted.
- **Trophic similarity of exact mid-frequency twins is 1.0.** Any normalized similarity gives
  identical inputs 1.0; trivia suppression relies on the df cap zeroing *both* sides of the Dice,
  which only engages once the idiom bucket exceeds `MaxPatternDF`. Between df=2 and the cap, exact
  twins read trophic 1.0 with *small* energy — the energy is what ranks, so this is display-level
  nuance, not a scoring bug.
- **Corroborated ranking has thin margins on vocabulary-heavy pairs.** A cross-package true
  clone and a vocabulary false positive can sit ~10-20% apart in key. The golden benchmark
  exists to watch exactly this; one labeled semantic false positive with genuinely high
  corroboration AND high trophic (sibling table-driven tests of different functions — the
  drivers really are near-clones; only the tested function differs) keeps the full-population
  benchmark's assertions deliberately red until a mechanism can express it (SUT-aware test
  discounting via the call graph is the named candidate). Trophic² also discounts non-identical
  true clones somewhat (the production clone sits mid-top-50, not top-20, in the full-population
  view) — the price of demoting skeleton siblings, watched by the same benchmark.

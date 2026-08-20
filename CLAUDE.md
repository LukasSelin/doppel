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
4. **Generate + enrich concept docs** — `concepter.New()` makes bare docs; **`mapper.Map` does the real work**: attaches qualified callers, resolved internal callees, and the depth-2 call-graph neighborhood; derives per-corpus role thresholds from the resolved degree distribution and classifies; aggregates caller/callee patterns and packages from resolved edges.
5. **Candidate retrieval** — `retriever.Retrieve` runs three per-function top-K channels (structural shingle-IDF, concept IC, resolved-call IDF — see *Candidate retrieval* below), unions and dedupes them, and computes definitive per-pair evidence masses plus the exact `fingerprint.Breakdown` for every union pair. Retrieval stats go to stderr. `cmd` materializes the candidates into `analyzer.SimilarPair`s (with `Retrieval` set). `analyzer.FindSimilar` still exists as the simple library API but the pipeline no longer calls it.
6. **Structural comparison** — a `comparator.Comparator` built over a corpus-weighted `ontology.Scorer` scores **every** candidate pair → `pair.Evidence`. Concept and role signals go through the ontology hierarchy, not string equality, and concept matching is weighted by information content computed from this run's tag counts — sharing a near-universal tag is weak evidence, sharing a rare one is strong.
7. **Structural filter** — when `--struct-min > 0`, pairs below that overlap score are **dropped**. This is a selection stage, not just annotation.
8. **Rank + report** — `analyzer.SortByEvidence` orders by retrieval evidence mass (desc), then code-shape score (desc), then `AIdx`/`BIdx`, and truncates to `--top`. `reporter.Print` to stdout always; `reporter.PrintMarkdown` to `--output` additionally. Both take a `reporter.Meta`; `--debug` adds per-pair retrieval provenance.

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
  analyzer/     SimilarPair + Retrieval types; FindSimilar (library API); SortByEvidence (final ranking)
  comparator/   Weighted structural overlap scoring (9 signals → 0.0–1.0 composite)
  reporter/     Plain-text (stdout) and Markdown (--output) formatting
```

Dependency directions that must hold: `analyzer` imports `comparator` (for the `Evidence` field), so
`comparator` must never import `analyzer`. `parser` imports `fingerprint`, so `fingerprint` must
never import `parser` — it works on `*ast.FuncDecl` directly. `ontology` imports nothing from this
module and must stay that way: `tagger`, `concepter` and `comparator` all depend on it. `retriever`
imports `parser`, `fingerprint`, `concepter`, `ontology` and must never import `analyzer` or
`comparator` — `cmd` bridges retriever candidates into `analyzer.SimilarPair`.

## Two scores, deliberately unblended — and a third quantity that ranks

Each pair carries two independent similarity numbers, gated by two independent flags:

| Score | Source | Flag | Means |
| --- | --- | --- | --- |
| `Score` | `fingerprint.Similarity` | `--threshold` | how alike the two bodies are (reported as `code-shape:`) |
| `Evidence.OverlapScore` | `comparator.Compare` | `--struct-min` | how much architectural context they share |

Do not merge these into one number. High code score + low overlap is a *different finding* (lookalike
bodies in unrelated subsystems) from high on both (a real merge candidate), and collapsing them
destroys that distinction.

The report is **ranked by neither**: ordering comes from `Retrieval.Total`, the candidate evidence
mass (see *Candidate retrieval*). Similarity says *how alike*, evidence says *how much rare,
informative material is shared* — a 1.0 code-shape match on a ubiquitous `Error()` idiom carries
near-zero evidence and sinks, while a 0.5-shape pair sharing rare calls and tags rises. The report
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
  hashes), `Flow` (control-flow histogram), `Types` (normalized param/result types), `Nodes`.
  The zero value means "no body" and never matches anything.
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
| shape | fingerprint shingle hashes, IDF over eligible units | `MaxShingleDF` 50 | `--min-nodes` eligibility; admits only pairs with exact `code-shape >= --threshold`, probing at most `4*ChannelK` neighbors |
| concept | tagger tags + non-root taxonomy ancestors (enumeration only) | `MaxConceptDF` 250 | none — evidence is `Scorer.SharedInformation` (raw `Σ IC(LCS)`) over the leaf tag sets |
| call | resolved internal callees (qualified) + import-qualified external calls via `RefPath` (full import path) | `MaxCallDF` 50 | none; bare names and variable-receiver calls are never tokens |

The union is deduped on `(min idx, max idx)`; every union pair then gets **definitive** evidence on
all three channels regardless of which admitted it, plus the exact `fingerprint.Breakdown`
(memoized). Summing the three masses into `Total` is coherent because all three are log-evidence
over the same corpus of N functions — do not normalize the components before summing.

Consequences worth knowing:

- A shingle/token in *every* unit has `idf = ln(N/N) = 0`; zero-mass neighbors are never admitted.
  The 130-clone `Error()` bucket exceeds the df cap entirely — those functions contribute no
  structural candidates and can only enter via concept/call evidence, which is the intended
  common-idiom suppression (no name-based heuristics anywhere).
- The concept channel indexes ancestors so `db_access`-only can meet `caching`-only through
  `data_store_access`, but the *evidence* is always `SharedInformation` on the leaf sets — a pair
  meeting only at a shallow ancestor earns only that ancestor's small IC.
- `ontology.Scorer.SharedInformation` exists precisely so retrieval never recomputes mass as
  `Σ ic.Of(m.LCA)` — that hits `Of("")` (the unknown sentinel) for unknown-term self-matches.
- `Stats.Suppressed` / `Stats.LargeBuckets` are diagnostics on stderr, not gates.
- On the ~8.7k-function Sendify corpus: ~22k union pairs, ~2.5s end to end (the old all-pairs
  scoring took ~20s), ~60% of pairs call-only, ~9% concept-only.

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
  "debug": false
}
```

Flag semantics after the retrieval redesign: `--threshold` floors code-shape for
**structural-channel admission only** (concept/call candidates bypass it); `--top` caps the
**final report** after comparison, filtering and evidence ranking — not the candidate set;
`--min-nodes` gates the structural channel only; `--channel-k` is the per-function per-channel
top-K; `--debug` adds retrieval provenance lines to the report.

Every functional flag except `--config` has a config key. Precedence: `applyConfig` only calls
`Flags().Set` when `!Flags().Changed(name)`, so explicit CLI flags always win over the file.
Unknown keys are ignored rather than rejected, so a stale config file does not break a run.

## Conventions

- Go-only. All parsing uses `go/ast` — no external parsers, no multi-language support.
- No caches and no generated state files. If you find yourself adding one, that is a design change,
  not an optimization.
- Cobra is the only direct dependency. Keep it that way unless there is a strong reason.
- Skipped directories: `.git`, `.claude`, `vendor`, `testdata`, `build`, `.idea`, `.vscode`.
  `_test.go` files are **not** skipped, and test functions legitimately dominate the top of the
  report on this repo.
- Tested: `ontology`, `fingerprint`, `analyzer`, `comparator`, `tagger`, `parser`,
  `concepter/role`, `retriever`, `reporter`, `cmd` config precedence. Untested and worth
  covering: `mapper`.

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
- **Repetitive scaffolding clusters can dominate the evidence ranking.** Fifteen `cmd/tool`
  `main.main` functions sharing ~100 mid-frequency setup calls each carry ~600 nats of call
  evidence and fill the Sendify top-20. That is real duplication, but it crowds the default view;
  per-function top-K keeps them from crowding the *candidate set*, and `--struct-min` or a larger
  `--top` gets past them in the report. A report-level diversity cap (max pairs per function) is
  the natural follow-up if it grates.

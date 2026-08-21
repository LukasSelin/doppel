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

The pipeline lives in `cmd/pipeline.go`, split in two: `index()` is the corpus-building prefix
(walk → parse → filter → tag → IC → call graph → mapper), `finishAnalyze()` is the reporting tail
(culture, retrieval, comparison, annotation), and `analyze()` is exactly the two in sequence — the
split is a pure refactoring, verified byte-identical on `--format json`. `runAnalyze` in
`cmd/analyze.go` is the CLI wrapper that supplies flags and renders, and `cmd/hook.go` is the other
`analyze()` caller. `cmd/query.go` calls only `index()`: a lookup needs the corpus, not the report.

`index()` takes `extra` units appended after the population filter — that is how a query probe
joins the corpus. Appending before anything runs is the load-bearing choice: the call-graph
resolver hands the probe resolved callees, mapper classifies its role against the same thresholds
as everyone, and every corpus statistic sees it — so query statistics differ from a plain analyze
by exactly the probe's own contribution, which is the honest way to ask how a proposed function
would sit in this corpus. `retriever.Probe` then runs the same three channels, gates and evidence
arithmetic as `Retrieve`, narrowed to the probe's admission turn (`admitFor`, extracted from each
channel's loop; the shared `evaluate` tail keeps the two from drifting). Query ranking is
`Total × (1 + Locality)` where locality is the fraction of the probe's depth-2 call-graph ball the
candidate inhabits — a ranking key only, displayed unblended, ties broken by code-shape then index.
`--near` names the package a bare snippet is wrapped in, which is what makes its bare-name calls
resolve (and locality light up); query's `--channel-k` defaults to 10, not 5, because a probe's
retrieval costs one function's worth and an exact-clone family larger than K gets cut on an index
tie-break. Stages 1-7 below are
`analyze()`; stage 8 belongs to the caller, because ranking is a presentation choice and the two
callers make it differently. `analyze()` takes a progress writer rather than using `os.Stderr`
directly: a hook must stay silent, since stderr from a SessionStart hook surfaces to the user as a
broken-tool notice. Stages in execution order:

1. **Walk & parse** — `filepath.WalkDir` + `shouldSkipDir`, then `parser.Parse` per `.go` file → `[]CodeUnit`. Unreadable files and parse errors are warned and skipped, never fatal. `fingerprint.Build` and `extractSignals` (the tagger's AST evidence) both run here, while the AST is still in hand.
2. **Tag** — `tagger.Tag(unit)` sets `unit.Patterns` (14 intent tags matched against the unit's AST signals — selectors, imports, string-literal contents, identifiers, node kinds — never against raw source text). Tag counts feed the corpus IC in the same loop.
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
  families.go   doppel families: the census view, plus analyze's family stage
  pipeline.go   index() + finishAnalyze(): the pipeline split into corpus-building prefix and reporting tail; filterByOverlap; snapshotOf
  query.go      doppel query: check a proposed function (a snippet on stdin) against the corpus, locality-weighted
  config.go     .doppel.json loading (AnalysisConfig), flag precedence, hookParams
  hook.go       doppel hook session-start / stop: Claude Code hook entry points, baseline file I/O
  version.go    build identity, for deciding whether a baseline is still comparable
  ontology.go   doppel ontology: print the vocabulary, check its axioms
internal/
  parser/       parser.go is a thin dispatcher; go_parser.go does the go/ast work; signals.go extracts the tagger's evidence channels → CodeUnit
  fingerprint/  AST token shingles + control-flow histogram + signature types; the code-similarity score
  ontology/     The formal vocabulary: entity kinds, typed relations, concept taxonomy, roles, axioms
  tagger/       AST-signal intent detection → 14 pattern tags
  concepter/    ConceptDoc; callgraph.go (BuildCallGraph); role.go (ClassifyRole, role constants)
  mapper/       Where enrichment actually happens: callers, role classification, aggregated patterns/packages
  retriever/    Multi-channel candidate retrieval: shape.go / concept.go / calls.go inverted indexes, retriever.go union + evidence
  culture/      Corpus-culture model: ecology.go (PMI), prototype.go (prototypes + typicality), habitat.go (fit), convention.go
  analyzer/     SimilarPair + Retrieval types; FindSimilar (library API); SortByEvidence (final ranking)
  comparator/   Weighted structural overlap scoring (9 signals → 0.0–1.0 composite)
  family/       Near-duplicate families: components + edge completion + maximal cliques over the pair graph
  snapshot/     One analysis run as comparable plain data: schema + Build, and Diff over two of them
  reporter/     Plain-text (stdout), Markdown (--output), JSON (--format json), and the two hook digests
  bench/        Measurement harness: golden-ranking scorer, the pinned public corpus ladder, per-stage benchmarks, example generator
examples/       Committed real reports for each corpus rung, plus labels/ (committed golden reviews) — see examples/README.md
```

Dependency directions that must hold: `analyzer` imports `comparator` (for the `Evidence` field), so
`comparator` must never import `analyzer`. `parser` imports `fingerprint`, so `fingerprint` must
never import `parser` — it works on `*ast.FuncDecl` directly. `ontology` imports nothing from this
module and must stay that way: `tagger`, `concepter` and `comparator` all depend on it. `retriever`
imports `parser`, `fingerprint`, `concepter`, `ontology` and must never import `analyzer` or
`comparator` — `cmd` bridges retriever candidates into `analyzer.SimilarPair`. `culture` imports
`parser`, `concepter`, `fingerprint` only (not even `ontology` — it is a leaf-tag count model) and
nothing imports it except `cmd`, which bridges its findings into `analyzer.CultureNote`. `family`
imports `parser`, `fingerprint`, `analyzer` and nothing else; `cmd` and `reporter` import it.

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
evidence** — `Retrieval.Total × Evidence.OverlapScore × Score × TrophicSim²`, with one further
linear factor `CallSim` (the call-channel Dice: the mutual fraction of the two functions'
informative call energy) **when both sides live in `_test.go` files** — SUT-aware test
discounting: two tests are related through what they exercise, not their driver skeleton.
Near-identical table-driven harnesses over different functions share no informative call tokens
and key to zero, while tests of the same machinery keep their shared call mass. Under
`--tests only` every pair is a test pair, so the hygiene view is SUT-aware globally. Raw mass alone let verbose shared vocabularies (PDF drawing APIs) outrank a
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

task dist          # goreleaser release --snapshot --clean  — the six release archives, unpublished
task release-check # goreleaser check                       — validate .goreleaser.yaml
task build-stamped # go build with -X cmd.version=$(git describe --tags --always --dirty)

task corpora  # fetch the pinned public corpus ladder (network, a few hundred MB)
task bench    # per-stage pipeline benchmarks over whatever is fetched
task golden   # score examples/labels/*.labels.json against the fetched corpora
task examples # regenerate examples/<corpus>.md
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
- `.github/workflows/ci.yml` runs `go build/vet/test` on pushes and PRs to **`master`** (not `main`),
  across a three-OS matrix (`ubuntu`/`windows`/`macos`) because releases ship binaries for all three.
  The determinism and hook-entry-point steps are guarded `if: matrix.os == 'ubuntu-latest'` — they
  assume `/tmp`, `printf` and `diff`, and determinism is platform-independent, so proving it once
  is enough. Go version comes from `go-version-file: go.mod` (currently `1.25.0`).
- **CI does not check gofmt** — only the local hook does. This is why formatting drift is possible.
- `.github/workflows/release.yml` fires on a `v*` tag push and runs GoReleaser (pinned `v2.17.1`)
  over `.goreleaser.yaml`. It **re-declares** vet/test/determinism in a `verify` job rather than
  reusing `ci.yml`: a tag push matches neither of `ci.yml`'s triggers, so `needs:` cannot reach it
  and `workflow_run` never fires. `contents: write` is scoped to the release job alone.
- The release build stamps `-X …/cmd.version=v{{ .Version }}`, **not** `{{ .Tag }}` — under
  `--snapshot` the latter resolves to the *previous* tag, so every local build of every tree would
  claim one identity, which is the stale-baseline-passes-comparability failure the check exists to
  catch. `task dist` runs the whole release locally into `dist/`; `task release-check` validates
  the config.
- `.github/workflows/wiki.yml` mirrors `.github/wiki/` to the repo's GitHub wiki on a push to
  `master`, path-filtered so it does not start for unrelated commits. **`.github/wiki/` is the
  source of truth and the wiki is a generated view** — a page edited in the wiki UI is overwritten
  by the next sync, and deleting a file here deletes the page, because the job clears the wiki's
  markdown before copying. Page name is the filename with its first letter upper-cased
  (`how-it-works.md` → `How-it-works`, which GitHub titles "How it works"), so the directory must
  stay **flat**: wiki page names cannot nest, and the job fails loudly rather than silently
  collapsing `a/page.md` and `b/page.md` onto one page. It pushes only when something changed.
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
`fingerprint.extractPatterns` during `Build` (the AST exists only during parse): L0 token n-gram
windows at widths 3 and 5 (k=3 keeps its legacy untagged hash so pre-widening dfs are unchanged;
w5 windows are width-tagged, never clamp on short streams, and certify longer shared runs — width
2 was built, measured on the cobra labels, and left out because its surviving mass fed
vocabulary-heavy false positives; see `l0ExtraWidths`), L1 call/binary-operator shapes, L2
statements with salient structure
(`return(call:Sprintf)`, `defer(call:Close)`, `if(bin:!=(id,nil))` — nil/true/false keep their
names so the err-check idiom falls out with no special case), L3 motifs — loop call summaries
covering header *and* body (`for{ call:Scan call:TrimSpace call:Atoi call:append }`, ≤ 8 callees)
and adjacent-statement bigrams (`seq[ assign:=(call:Atoi) ; if(bin:!=(id,nil)) ]`) — and L4
def-use flow edges (`defuse.go`): single-hop role edges from a def source (a parameter, or a
binding whose RHS contains a call) to a use sink (a call it is passed to or invoked on, a return,
or a condition) — `flow:param→call:Errorf`, `flow:call:Open→call:Close`, `flow:call:Atoi→cond`.
Renders name roles, never identifiers, so the edges are rename-invariant; the tuple rule
(`x, err := f()` binds both names to `call:f`) is what makes the errcheck idiom fall out free. A
value computed and dropped emits no onward edge — previously indistinguishable from one that
flows. For levels 1–4 the render string IS the hash serialization, so hash and explanation cannot
drift; L2/L3/L4 keep their renders, L0/L1 do not. In the `shared structure:` block L4 sorts below
L2/L3 at equal energy (`chainRank`): a role edge is a coarser explanation than a concrete
statement shape.

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

`fingerprint.Similarity` blends four components; weights are constants and sum to exactly `1.00`.

| Component | Metric | Weight |
| --- | --- | --- |
| AST shingles | Jaccard over hashed 3-grams | 0.60 |
| Control flow | cosine over the node-kind histogram | 0.20 |
| Nesting depth | cosine over the entry-depth histogram | 0.05 |
| Signature | Jaccard over normalized param/result types | 0.15 |

The depth histogram (`Fingerprint.Depth`, 6 buckets, deep tails folded into the last) records the
nesting depth each control-flow node is *entered* at; the seven statement-bearing constructs (if,
for, range, switch, type switch, select, funclit) push a level for their children. It exists
because flattened tokens carry no depth: sequential ifs and nested ifs used to have identical
token bags, identical flow histograms, and score 1.0. Depth's 0.05 was carved entirely out of
Flow (0.25 → 0.20) — nesting is flow-adjacent, so flow pays for it. Rendered as `nesting:` in the
breakdown line. A nesting change is a body change: `snapshot.Digest` hashes Depth (snapshot
Schema 3).

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

`ContextMergeWorthy = OverlapScore >= 0.4 && countSignals >= 2` — the **architectural half only**.
The verdict itself is `comparator.MergeWorthy(ev, shape) = ev.ContextMergeWorthy && shape >=
MergeShapeFloor` (0.4, mirroring the overlap gate), surfaced as `analyzer.SimilarPair.MergeWorthy()`,
which is the one type holding both halves. The split is forced: the comparator sees two ConceptDocs
and no fingerprint, so it cannot floor on shape itself. It is also the point — shared architectural
context structurally favours same-package siblings, which share callers and callees by construction,
so context alone reached 0.4 on pairs whose bodies had almost nothing in common (measured: a
merge-worthy label at code-shape 0.31). The field rename is deliberate: every reader of the old
`Evidence.MergeWorthy` is a compile error, so no surface can silently keep half a verdict.
**`countSignals` counts only 5 of the 12**
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

Exactly 14, emitted in declaration order: `retry`, `http_call`, `db_access`, `validation`, `mapping`,
`transaction`, `caching`, `concurrency`, `error_wrapping`, then the five added with ontology 1.1.0 —
`grpc_call`, `circuit_breaker`, `serialization`, `file_io`, `logging` — appended after the original
nine so every pre-existing tag keeps its emission position. The rules name `ontology` concept terms
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
- `retry` and `circuit_breaker` are the two tags with no structural handle — their evidence is
  lexical, in identifier names (`circuit_breaker` also matches a `gobreaker` import).
- String literals are still evidence, so a test whose fixture strings contain `%w` or `SELECT `
  earns those tags. A function carrying SQL strings is db-flavored even when it is a test.
- `json.Marshal`/`json.Unmarshal` moved from `mapping` to `serialization` when that leaf arrived —
  otherwise every json function would carry both tags forever. `mapping` is now purely the
  conversion vocabulary (`transform`, `convert`, `ToDTO`, …).
- `serialization`, `file_io` and `logging` are selector/method/receiver evidence only — an
  `encoding/json` or `os` import is file-level and near-universal, and an import-substring `"log"`
  would match `dialog` and half the module paths on earth.
- The pre-Go-only polyglot keywords (`axios`, `urllib`, `Promise.`, `await `) are gone.

### The ontology

`internal/ontology` is the vocabulary the comparator reasons over, and the reason a pair tagged
`http_call`/`db_access` no longer scores the same as a pair with nothing in common. The fourteen tags
are leaves of a taxonomy whose interior nodes are abstract and exist purely to relate them:

```
concept → io_operation → remote_io → http_call, grpc_call
                       → data_store_access → db_access, caching, transaction
                       → file_io, logging
        → data_transformation → mapping, validation, serialization
        → control_flow → concurrency, fault_tolerance → retry, circuit_breaker
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
  score, so `MergeWorthy` is not fully corpus-independent — the signal count and the shape floor
  are.)
- **Singleton sets cannot be discounted.** `{error_wrapping}` vs `{error_wrapping}` is still 1.0 —
  the shared information and the total information are the same quantity. The discount only
  manifests in sets of ≥ 2 tags. Fixing it would mean IC-scaling the `exhibits` relation weight,
  which breaks axiom 7; documented instead.

## Families

`internal/family` answers the census question the pair list cannot: how many near-duplicate
*groups* this corpus has. It generalizes nothing. Almost everything upstream of the report is
already n-ary — tag frequencies, IC, culture, role thresholds, and the retrieval channels, whose
posting lists are sets of arbitrary size — and the two stages that are genuinely pairwise (the
12-signal comparison, the report) are pairwise because Jaccard, cosine and nearest-shared-ancestor
have no canonical k-ary extension, not for a representational reason. So families keep pairwise
scoring, whose quadratic cost is already paid, and cluster afterward on the surviving pair graph.

Two constraints shape the whole design:

- **Non-transitivity.** A≈B at 0.7 and B≈C at 0.7 says nothing about A≈C. Single-linkage or
  connected-component clustering chains through those links until a "family" spans functions with
  nothing in common — the classic staircase failure in clone detection. A family here is a
  **maximal clique**, so every member is similar to every other member and the report can state a
  guarantee (`every pair >= 0.72 code-shape`) that a reader falsifies by opening two files.
- **Retrieval gaps.** Each function keeps a bounded top-K per channel, so an edge may be missing
  because it fell out of a budget rather than because the two functions are unalike. Enumerating on
  the retrieved graph alone would split real families. `completeComponent` therefore scores every
  unpaired member of a component directly with `fingerprint.Similarity` before enumeration —
  fingerprints are built during parsing, so this is arithmetic over sorted slices, not re-parsing.
  It is not cosmetic: on this repo it supplies 3 of the 21 edges in the 7-member `sorted*Keys`
  family, and on prometheus 6.5k edges overall.

Stages, in `Build`: adjacency over unit indices from pairs with `Score >= Min` → connected
components → per-component edge completion → Bron–Kerbosch with pivoting over a degeneracy
ordering → keep cliques of `>= MinSize` (3; a 2-clique is a pair) → order by size desc, `MinEdge`
desc, `MeanEdge` desc, members asc. Determinism is by construction: candidate sets are
ascending-index slices, the degeneracy order and the pivot both tie-break on lowest index, and no
map iteration decides anything.

Two guards, both reported and never silent: `MaxComponent` (128 — measured, the value at which
neither moby nor prometheus skips anything, for ~0.5s) and `MaxSearch` (200k recursive calls, for a
component small enough to complete but pathologically dense). A tripped guard records the component
size in `Stats.Skipped` and emits **no** families for it, rather than presenting a partial
enumeration as the answer.

**Where it runs is load-bearing.** The family stage lives in the `cmd` command functions, never in
`finishAnalyze`: `cmd/hook.go` calls `analyze()` directly and snapshots `res.Pairs`, so a stage
inside the pipeline would change every baseline and delta. It reads `res.Pairs` — the full
comparator-scored, struct-min-filtered set — never the ranked slice, because `--max-per-func` is a
report-time device applied *after* scoring and a family of seven rests on 21 edges the pair list
would never show. `analyze` renders `--families N` (default 5) of them after the pair report;
`doppel families` is the census, with no presentation cutoff and its own `--format json` payload.

Families are deliberately **not** in `snapshot`: only what a consumer reads is stored, and the Stop
hook rewrites that file every turn. `analyze --format json` remains the snapshot exactly as before.

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
  "tests": "exclude",
  "families": 5,
  "family-min": 0.60,
  "hook-notify": "agent"
}
```

Flag semantics after the retrieval redesign: `--threshold` floors code-shape for
**structural-channel admission only** (concept/call candidates bypass it); `--top` caps the
**final report** after comparison, filtering and evidence ranking — not the candidate set;
`--min-nodes` gates the structural channel only; `--channel-k` is the per-function per-channel
top-K; `--debug` adds retrieval provenance lines to the report; `--max-per-func` caps how many
final-report pairs any one function may appear in (0 disables); `--families` bounds the report's
family section (0 removes it) and `--family-min` is the code-shape every two members of a family
must reach — both presentation, so neither is in `Params` and neither can invalidate a baseline; `--tests` picks the population
(`include`/`exclude`/`only`, default `exclude`) before any statistic is computed.

`hook-notify` (`agent` | `user` | `off`) is read only by `doppel hook stop` and has no flag — there
is no CLI surface a hook setting would belong to. `format` (`text` or `json`) is a key like any
other. Every functional flag except `--config` has a
config key. Precedence: `applyConfig` only calls
`Flags().Set` when `!Flags().Changed(name)`, so explicit CLI flags always win over the file.
Unknown keys are ignored rather than rejected, so a stale config file does not break a run.

## Impact measurement and the Claude Code plugin

`internal/snapshot` gives the tool the one noun it lacked: a **run**. `comparator` compares two
functions, `analyzer` compares two fingerprints, `reporter` renders one result — nothing could
compare two *analyses*. A `Snapshot` is one run reduced to comparable plain data, `Diff` produces a
`Delta` from two of them, and `doppel hook` wires both to Claude Code.

Four rules hold the schema together. The first three exist because breaking them produces a
confidently wrong answer rather than a missing one; the fourth is what keeps the file small enough
to rewrite on every turn:

- **No maps, no wall-clock, no absolute paths.** Every map is flattened into a sorted slice before it
  reaches JSON, and unit paths are stored relative to the analysis root and slash-separated. A
  timestamp or an absolute root inside a `Snapshot` would break byte-identical reproducibility the
  moment `--format json` exists, so both live in the baseline file wrapper in `cmd/hook.go`, where
  nothing compares them. `TestSchemaHasNoMaps` enforces the first rule by reflection.
  Path normalisation is what lets a hook run rooted at an absolute cwd be compared to
  `doppel analyze .` at all.
- **Identity, never position.** Units are keyed by `package.Name`, disambiguated with `@file` when
  that collides (`init`, or two directories sharing a package clause). Pair sides are ordered by
  name, *not* by the `AIdx`/`BIdx` order `FindSimilar` emits — those are file-walk positions and
  shift the moment a file is added, which would reorder the whole pair list in a diff for no reason
  anyone caused.
- **Hooks diff the full candidate set.** `hookParams` honours the `.doppel.json` keys that define the
  *corpus* (`threshold`, `min-nodes`, `channel-k`, `tests`) and overrides the ones that only decide
  what gets *shown* (`top`, `max-per-func`, `struct-min`). A pair that fell past rank 20 has not
  changed; reporting it as a session's impact would be a lie.
- **Only what a consumer reads is stored.** `Schema` 2 dropped every field nothing read back:
  `Unit.Qualified`/`Exported`/`Receiver`/`Nodes`/`Callers`/`Callees`, and on `Pair` the four
  `fingerprint.Breakdown` components and the evidence `Reasons`. The text report renders both of
  those straight off `analyzer.SimilarPair` and never through a snapshot, so they were write-only
  — and `Reasons`, free-text English restating counts, was a *quarter* of a baseline's bytes.
  Removing them cut a 280-function baseline from 461KB to 189KB (59%). `Unit.Role` is the one
  deliberate exception: no internal consumer reads it either, but the `--format` row in `README.md`
  documents it as part of the `--format json` payload, so it stays until that promise changes.
  Anything added back needs a reader, or it is weight in a file the Stop hook rewrites every turn.
  `Schema` 3 added no field: it changed what `Pair.MergeWorthy` asserts (the shape floor). A
  meaning change with no shape change is exactly what the version exists for — the two files would
  otherwise compare cleanly and report a merge-worthy drop nobody caused.

**What a delta may and may not claim.** `UnitsAdded`, `UnitsRemoved` and `BodiesChanged` are solid:
they come from names and from `Digest`, an FNV-1a hash of the unit's own fingerprint, so nothing
outside a function can move them. Pair changes are one tier down and each carries an `Attributable`
bit, because retrieval keeps a bounded top-K per function and a pair can enter or leave the candidate
set without either side being touched. Everything else corpus-relative — role changes, caller/callee
counts, overlap movement, tag totals — is deliberately **absent from `Delta`**: those move when code
nobody touched moves, and reporting them would blame a session for something it did not do.

Incomparability is a result, not an error. A mismatched schema, doppel build, ontology version or
param set means the two runs measured different questions, so `Diff` sets `Comparable=false` with a
reason rather than returning a partial delta.

**The hook contract**, in `cmd/hook.go`:

- Both subcommands read the hook payload on stdin and write a hook response on stdout. **Neither ever
  exits non-zero or writes to stderr** — every failure path ends at `emitNothing`. A SessionStart
  hook's stderr surfaces to the user as a broken-tool notice, and blocking a session over a
  measurement would be indefensible.
- `session-start` emits `additionalContext` (the corpus concept inventory — deliberately only
  what is session-stable: tag counts, absent tags, roles, one pair-count line; per-target findings
  live in the two hooks below, where a target exists) and writes the baseline **only if one does
  not already exist**. SessionStart also fires on resume and after compaction; re-recording then
  would silently move the origin mid-session.
- `user-prompt` (UserPromptSubmit) scopes the corpus's duplication facts to the packages the
  prompt mentions — `@path/to/pkg` mentions and bare package names, confirmed against the corpus,
  capped at 3 in first-mention order (multi-matches resolved in sorted order; map order deciding
  who wins the cap would break run-to-run stability). It re-runs the full pipeline once per
  prompt, and is silent for prompts mentioning nothing it knows — the common case pays the
  analysis and says nothing.
- `pre-tool` (PreToolUse, matched on `Edit|Write`) advises on the merge-worthy twins of the file
  about to be edited, from the **session-start baseline**, labeled "as of session start". This is
  a deliberate widening of the baseline's role — fact sheet as well as measurement origin — with
  the original boundary intact: `analyze` never reads it and no pipeline stage is ever skipped
  because it exists; the advisory is not a pipeline, and recomputing instead would cost a full
  analysis per edit. **Advisory-only, permanently**: `additionalContext` is emitted,
  `permissionDecision` never is — a blocking dedupe hook misfiring on a genuine near-duplicate
  (exactly what it fires on) would be worse than none. An `Advised` ledger in the baseline
  wrapper (same mechanism as `Reported`) makes each file's advisory fire once per session.
- Both digests rank the pairs they have room to print by **corroborated evidence**,
  `shape x overlap` (`impactKey` in `reporter`), then shape, then names. Not by overlap, and not by
  the merge-worthy flag: shared context favours same-package siblings by construction, so overlap
  alone promotes intentional variants over cross-package copy-paste, and the flag is a boolean over
  a continuum that cannot separate a pair three signals past the gate from one barely over it. The
  flag stays on the rendered line. This is presentation only — `snapshot.Diff`'s own total order
  (attributable, merge-worthy, score) is untouched, and nothing here feeds a score.
- `stop` emits `systemMessage` to the user and, under `hook-notify: agent` (the default),
  `additionalContext` to the model. **`additionalContext` on a Stop hook continues the turn** —
  verified against the shipped Claude Code binary, because the public docs do not document Stop's
  decision-control fields at all. The hook's message is appended to the same list the harness
  returns as `blockingErrors`, and a non-empty list re-enters the query loop
  (`transition: {reason: "stop_hook_blocking"}`). There is no way to put text in the model's
  context from a Stop hook without the agent working again.

  Three things make that affordable, and removing any one of them makes the feature hostile:

  1. **The `stop_hook_active` guard.** The harness sets it on the re-entry; `runHookStop` returns
     before any analysis when it is true. Without it the turn never ends until Claude Code
     overrides it with a warning (`CLAUDE_CODE_STOP_HOOK_BLOCK_CAP`). It also spares a second full
     pipeline run per reported turn.
  2. **`reporter.Notable`, a much higher bar than the user digest's.** Only new merge-worthy
     attributable pairs, and gate crossings with an edited side. Attribution gates the crossings
     as hard as the pairs, and measurement proved it must: merge-worthiness is corpus-weighted, so
     adding two functions anywhere nudges untouched pairs across the 0.4 boundary.
  3. **The `Reported` ledger** in the baseline wrapper. A delta is cumulative against the
     session-start origin, so an unremembered finding is re-reported — and re-continues the turn —
     every turn thereafter. The ledger rides in the baseline file rather than a second one, and
     never touches its `Snapshot`. It records **only what the digest printed**, which is why
     `AgentDigest` returns the findings it rendered: the note prints a bounded head, and ledgering
     the rest would retire findings nobody was ever shown. They lead the next turn's list instead.

  `hook-notify` (`agent` | `user` | `off`) is the escape hatch; `user` is the pre-existing
  behaviour exactly. It is deliberately **not** in `Params`, because who gets told has no bearing
  on what was measured and must not invalidate a baseline.
- The session id is **hashed, not validated**, to build the baseline path. Against a fixed-length hex
  digest, traversal, absolute paths, drive letters and reserved device names are impossible by
  construction rather than by a blocklist somebody has to keep correct.

The plugin itself is `plugin/`, published through the one-entry marketplace at
`.claude-plugin/marketplace.json`. Its hooks use **exec form** (`command` + `args`), which takes no
shell and behaves identically on Windows and Unix, and which is also the only form where
`${user_config.*}` is substituted.

## Conventions

- Go-only. All parsing uses `go/ast` — no external parsers, no multi-language support.
- **No caches.** No run ever reads persisted state to avoid recomputation: `doppel hook stop`
  re-runs the whole pipeline from source every time, exactly as `analyze` does. The one persisted
  artifact is the **session baseline** written by `doppel hook session-start` — a measurement
  origin, not a cache. It is scoped to a single Claude Code session, lives in the OS temp
  directory under a hash of the session id, is discarded whenever the schema, the doppel build,
  the ontology version or the run params differ, and is swept after seven days. `analyze` must
  never read it, and no code path may short-circuit a stage because a baseline exists. If you find
  yourself adding a second state file, or reading this one to skip work, that is a design change,
  not an optimization. (`--format json` and `--output` write reports, not state.)
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
  `concepter/role`, `retriever`, `culture`, `reporter`, `snapshot`, `cmd` config precedence and
  hook baseline handling. Untested and worth covering: `mapper`.
- **Measurement harness** (`internal/bench`), four jobs in one package:
  - `scoreLabels` ranks a corpus and scores a human-reviewed labels file against it: every
    labeled pair gets a rank or an absence reason, three assertions are hard (merges retrieved,
    no false positive above the worst merge, no false positive in the top 20), the rest is a
    logged scorecard. A **partial** review is fine — only labeled pairs are scored, so a
    genuinely contested pair is better left unlabeled than guessed.
  - `TestGoldenRanking` scores a **private** review: both inputs come from outside the repo
    (`DOPPEL_BENCH_CORPUS` + `DOPPEL_BENCH_LABELS`) and **no name, path, or labeled pair from
    that corpus may ever be committed**. Run:
    `DOPPEL_BENCH_CORPUS=<corpus> DOPPEL_BENCH_LABELS=<labels.json> go test ./internal/bench/ -v`.
    Labels declare their own population; the harness mirrors the `--tests` filter before any
    corpus statistic, and drops cross test/prod pairs like the pipeline.
  - `TestGoldenCorpora` scores the **committed** reviews in `examples/labels/<corpus>.labels.json`
    against the matching rung of the public ladder, skipping corpora that are not fetched. The
    public/private split is the whole point: the committed reviews make the benchmark
    reproducible by anyone, the env-driven one keeps the private corpus private.
  - `Corpora` (corpora.go) pins seven public Go repos at release tags, ordered old-and-complex
    to new-and-narrow (moby 8003 funcs → conc 81). Only coordinates are committed; `Fetch`
    shallow-clones into `Root()` (`$DOPPEL_CORPORA`, else user cache) and verifies HEAD against
    the manifest, so a moved tag fails loudly instead of silently changing every example.
    `Root()` is deliberately outside the working tree — a corpus under it would be walked by
    `doppel analyze .` on this repo. Annotated tags need their **peeled** commit
    (`git ls-remote <repo> refs/tags/<tag>^{}`), which is what bit the first version of the
    manifest.
  - `pipeline.go` is the ranking-relevant pipeline as a library (`Load` + `Run` stages +
    `Analyze` + `RankKey`), shared by the golden scorer and `BenchmarkCorpus`. Culture, habitats
    and arenas are deliberately absent: they annotate, they never rank. `RankKey` duplicates
    `SortForReport`'s ordering quantity so a scorecard can print it — it must track it.
  - `TestGenerateExamples` (guard `DOPPEL_BENCH_EXAMPLES=1`) regenerates `examples/<name>.md` by
    running the **built binary** with `cmd.Dir` set to the corpus — not the library — so the
    committed reports are what the documented command actually prints, culture/habitat/arena
    annotations included, with corpus-relative paths. The one stderr line naming a local path is
    filtered out.

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
- **`remote_io` and `fault_tolerance` were unary until ontology 1.1.0.** A taxonomy node with one
  child adds no discriminative power and costs its leaf a level of depth, which *lowers* that leaf
  relatedness to everything. They were kept as the slot future siblings go in, and `grpc_call` and
  `circuit_breaker` are those siblings, arrived. `error_handling` remains unary over
  `error_wrapping` — same reasoning, same future slot.
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
- **The L4 def-use pass is deliberately crude.** Bindings are name-keyed within the function, so
  shadowing merges with the first binding winning; pointer/closure/field aliasing, field writes,
  multi-hop chains, tuple position, control-flow sensitivity and cross-function flow are all
  outside it — the full non-capture list is documented at `extractDefUse`. Each would cost a
  resolution pass out of proportion for evidence rendering; the cure, as with the call-graph
  resolver, is go/types.
- **Trophic similarity of exact mid-frequency twins is 1.0.** Any normalized similarity gives
  identical inputs 1.0; trivia suppression relies on the df cap zeroing *both* sides of the Dice,
  which only engages once the idiom bucket exceeds `MaxPatternDF`. Between df=2 and the cap, exact
  twins read trophic 1.0 with *small* energy — the energy is what ranks, so this is display-level
  nuance, not a scoring bug.
- **Corroborated ranking has thin margins on vocabulary-heavy pairs.** A cross-package true
  clone and a vocabulary false positive can sit ~10-20% apart in key; the golden benchmark
  watches exactly this. Trophic² also discounts non-identical true clones somewhat (the
  production clone sits mid-top-50, not top-20, in the full-population view) — the price of
  demoting skeleton siblings.
- **Generated code owns the top of a large old corpus.** moby's entire top ten is `.pb.go`
  protobuf `Unmarshal`/`skipX` methods, and half of prometheus's is; the pairs are factually
  near-identical and completely unactionable. doppel has no exclusion flag and no notion of
  "this directory is not the library" (chi's `_examples/` demo `main`s are the small-corpus
  version of the same problem), so the only answer today is pointing it at a hand-written
  subtree — `prometheus/tsdb` reports the real float/int histogram duplication instead. A
  generated-file or path-exclusion filter is the obvious fix and is deliberately not built.
- **Committed examples drift silently.** `examples/*.md` is real output from a pinned tree, but
  nothing verifies it: any ranking change makes every file stale until somebody runs
  `task examples`. Regenerating is cheap (~10s for all seven) — do it in the same change that
  moves ranking, and re-read the diff, because it is the fastest available review of what a
  scoring change actually did. `task golden` is the assertion-backed half of that check; the
  reports are the eyeball half. The measured numbers in `examples/README.md` are hand-copied
  from a `task bench` run on one machine and go stale the same way.

- **Families overlap, and that is reported rather than resolved.** A function can belong to
  several maximal cliques, so the family list can show it twice; the counts report *distinct*
  functions. Picking one clique per function would be a judgement the tool cannot justify, and a
  partition would have to break a tie no evidence decides. Family membership is also corpus-relative
  in the same way roles and typicality are — the pair graph moves when unrelated code moves — and it
  is bounded by the same retrieval recall the pair list is, except inside a component, where edge
  completion repairs it.
- **SUT-aware test discounting is only as good as call resolution.** A test pair with zero
  informative call tokens keys to zero even when genuinely duplicated (mock-heavy tests whose
  every call is variable-receiver read CallSim 0 — harsh but honest: no call evidence, no SUT
  corroboration). Both label sets pin the behavior; the historical semantic false positive
  (sibling table-driven tests of different calculations) is resolved by this factor, and both
  benchmarks' assertions are green.
- **A pair change is not always attributable to an edit.** Retrieval keeps a bounded top-K per
  function, so adding code anywhere can push a pair out of some other function's channel budget
  without either of its bodies changing. `Delta` marks these with `Attributable=false` and the
  digest counts rather than lists them, but the underlying imprecision is inherent to top-K
  retrieval — the only fix would be scoring all pairs, which is the O(n²) cost retrieval exists to
  avoid. `Digest`, by contrast, is exact: a changed digest always means that function changed.
- **The Stop hook runs a full analysis on every turn.** Cumulative comparison against a fixed
  baseline is what makes the report answer "what has this session done", but it means the whole
  pipeline runs each time the agent finishes responding. At a few hundred functions this is
  imperceptible; on a very large repo it is the first thing to feel. There is no incremental mode
  and adding one would mean caching, which the conventions rule out.
- **Two dev builds are indistinguishable to the comparability check.** `buildVersion` falls back to
  `(devel)` for a plain `go build`, so rebuilding doppel mid-session with changed scoring constants
  leaves a stale baseline looking comparable. Ldflags-stamped releases do not have this problem, and
  the fallback deliberately does not invent a value that would make every baseline stale instead.
  `task build-stamped` is the mitigation when you are mid-session and changing scoring code; plain
  `task build` deliberately stays unstamped, because `git describe --dirty` collapses every dirty
  tree to one string and would hide the failure rather than fix it.

- **`v0.1.0` exists but carries no binaries.** `go install github.com/LukasSelin/doppel@latest`
  works again — `v0.1.0` on `a268368` has the correct module path and the `hook` subcommands, so it
  supersedes `v0.0.1-alpha` (a lightweight tag on the *initial* commit `9c221eb`, whose `go.mod`
  still reads `module github.com/lukse/doppel`; kept rather than moved, because the bench manifest
  already learned that moved tags fail loudly for a reason). But `v0.1.0` predates
  `.github/workflows/release.yml`, so **no archives were ever built for it** — and both READMEs now
  point users at the releases page first. The next tag cut after this change is the first one that
  actually publishes binaries; until it exists, the download instructions describe a release that
  has no assets. Two things to get right when cutting it: use an **annotated** tag, and keep
  `plugin/.claude-plugin/plugin.json`'s `version` in lockstep with it, since the plugin is coupled
  to the binary's `hook <name>` contract.

- **`http_call` on wrapper clients is receiver-convention-deep only.** The historical defects are
  fixed — dead `http.Do` dropped, `http.NewRequestWithContext`/`PostForm`/`Head` added, and
  `extractSignals` now records the tail pair of a nested selector (`c.httpClient.Do` →
  `httpClient.Do`), one level deep — but the wrapper evidence is still the `httpClient` receiver
  name. A wrapper field named `api` or `transport` stays invisible, the nested tail goes exactly
  one level (`a.b.c.Do` records `c.Do`, never `b.c`), and the tag remains deliberately not
  import-based (servers import `net/http` too).

- **Interface implementations dominate merge-worthy output.** Methods satisfying a shared interface
  across sibling packages — a `Validate` per provider, say — are near-identical by construction and
  completely unactionable, and on a wide corpus they crowd the top of the list the same way
  generated code does. Recognising one needs `go/types`, which is out of proportion here, and a
  name-or-signature heuristic is exactly the kind of rule the tagger and retriever avoid
  everywhere else. The digest orders what it *shows* by merge-worthiness and overlap, which helps
  a six-line hook report and does nothing for a full `analyze` run.

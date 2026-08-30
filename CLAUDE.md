# Doppel

Go CLI tool that measures **architectural erosion** in a Go codebase — the gap between the structure
a project intends and the one it has — and surfaces it as merge candidates: functions or methods
that do similar enough work to warrant consolidation.

## Goal

**The target is erosion by locally reasonable edit.** Every duplicate this tool finds was defensible
when it was written: the author needed a retry loop and writing one cost less than finding the
existing one; a handler was forked for a second provider and the two aged apart. No rule was
violated, so nothing in review or in a linter could have caught it — erosion is a property of the
corpus, not of a diff, and that is the whole reason the pipeline reads every function at once rather
than reading a change. Weigh proposals against it: a feature that only helps someone judge a single
function in isolation is off-target, however good the signal.

Perry & Wolf (1992) split this in two — *erosion* is violating the architecture, *drift* is
insensitivity to it. Strictly, doppel is aimed at their drift; violations are what a linter is for.
These docs use "erosion" as the umbrella term because that is how it is generally read, and reserve
"drift" for the narrower per-function sense the culture and habitat notes carry.

**What that rules out, by construction.** No declared architecture is read, so a layering violation
is invisible. No git history, so authorship, age and churn are invisible. No config or deploy state,
so configuration drift is a different tool's problem. The corpus is the only norm doppel has — which
is exactly why every judgment it makes is corpus-relative (roles, typicality, habitat fit, IC), the
caveat that recurs throughout this file.

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
(walk → parse → filter → WL/cons corpus statistics → call graph → learn concepts → IC → mapper), `finishAnalyze()` is the reporting tail
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
tie-break. Stages 1-8 below are
`analyze()`; stage 9 belongs to the caller, because ranking is a presentation choice and the two
callers make it differently. `analyze()` takes a progress writer rather than using `os.Stderr`
directly: a hook must stay silent, since stderr from a SessionStart hook surfaces to the user as a
broken-tool notice. Stages in execution order:

1. **Walk & parse** — `filepath.WalkDir` + `parser.ShouldSkipDir`, then `parser.Parse` per file whose extension a frontend claims and `--languages` admits → `[]CodeUnit`. Unreadable files and parse errors are warned and skipped, never fatal. The frontend produces a `syntax.File` — and, where it has a canonicalizer, the canonical body alongside the body as written; `fingerprint.Build` (which fills the WL bag from `syntax.Func.Shape`) and `extractSignals` (the seed rules' evidence) then run over the IR.
2. **Structural corpus statistics** — `fingerprint.LabelWeights` over every unit's WL bag (`Result.WL`, the `ln(N/df)` label surprisal that makes code shape corpus-weighted) and `fingerprint.ConsCorpus` over every canonical body (`Result.ConsStats`, the compression ratio). Both read each unit's own canonical tree and nothing else, which is why they come first: the population is settled and no concept vocabulary exists yet. Everything below is conceptual and needs the call graph.
3. **Build call graph** — `concepter.BuildCallGraph(units)` → `concepter.Graph`, both directions over **qualified names** (`package.Name`, methods keeping their receiver: `comparator.*Comparator.Compare`). A resolver maps each raw callee string to at most one unit: import-qualified selectors through the file's recorded import bindings (aliases included), variable-receiver method calls only when the method name is unique corpus-wide, bare names to the same-package function. Ambiguity drops the edge; recursion is excluded; only repo-internal edges exist. It runs **before** concepts now, because resolved calls are one of the channels a concept is learned from — and still before concept docs, which need caller lists.
4. **Learn concepts** — `lexicon.Build(units, cg, seeds, …)` fills `unit.Concepts`: corpus-derived concepts with graded membership (see *The learned lexicon*). `tagger.Tag` still runs, but only to produce `seeds` — which functions a concept search starts from, and nothing else. Member counts and summed confidence feed the corpus IC in the same loop, and the learned concepts become the taxonomy's leaves for this run via `ontology.WithConcepts`.
5. **Generate + enrich concept docs** — `concepter.New()` makes bare docs; **`mapper.Map` does the real work**: attaches qualified callers, resolved internal callees, and the depth-2 call-graph neighborhood; derives per-corpus role thresholds from the resolved degree distribution and classifies; aggregates caller/callee concepts (as graded `[]parser.Concept`, each keeping the strongest confidence any neighbour asserted) and packages from resolved edges. `culture.Build` then models the corpus's conceptual practice (see *Corpus culture*); its summary goes to stderr, and after the struct-min filter each surviving pair gets `Culture` notes for atypical realizations of its **shared** tags (positional attachment, like Evidence).
6. **Candidate retrieval** — `retriever.Retrieve` runs three per-function top-K channels (WL-label IDF, concept IC, resolved-call IDF — see *Candidate retrieval* below), unions and dedupes them, and computes definitive per-pair evidence masses plus the exact `fingerprint.Breakdown` for every union pair. Retrieval stats go to stderr. `cmd` materializes the candidates into `analyzer.SimilarPair`s (with `Retrieval` set). `analyzer.FindSimilar` still exists as the simple library API but the pipeline no longer calls it.
7. **Structural comparison** — a `comparator.Comparator` built over a corpus-weighted `ontology.Scorer` scores **every** candidate pair → `pair.Evidence`. Concept and role signals go through the ontology hierarchy, not string equality, and concept matching is weighted by information content computed from this run's tag counts — sharing a near-universal tag is weak evidence, sharing a rare one is strong.
8. **Structural filter** — when `--struct-min > 0`, pairs below that overlap score are **dropped**. This is a selection stage, not just annotation.
9. **Rank + report** — `analyzer.SortForReport` orders by corroborated evidence (`Total × OverlapScore × Score × TrophicSim²`, then code-shape, then `AIdx`/`BIdx`), applies the `--max-per-func` diversity cap greedily with backfill, and truncates to `--top`. `reporter.Print` to stdout always; `--output` additionally writes markdown, or the dashboard when the path ends `.html`. Both take a `reporter.Meta`; `--debug` adds per-pair retrieval provenance.

`docs[i]` describes `units[i]`, and `SimilarPair` carries `AIdx`/`BIdx` into that slice. Evidence
attachment is a positional lookup, deliberately — an earlier version keyed it on
`Package + "." + Name` while the call graph keyed on bare names, so lookups silently missed and
`--struct-min` then dropped the pair. Do not reintroduce name-keyed lookups between these stages.

## Module layout

```
main.go         Thin entry point → cmd.Execute()
cmd/            CLI commands (Cobra).
  root.go       rootCmd, Execute()
  analyze.go    Pipeline orchestrator; analyze's own flags (each command registers its own in init)
  families.go   doppel families: the census view, plus analyze's family stage
  overview.go   Queries the corpus model (culture, ontology, call graph) into reporter.Overview
  dashboard.go  Assembles dashboard.Payload — the semantic model the HTML page draws
  pipeline.go   index() + finishAnalyze(): the pipeline split into corpus-building prefix and reporting tail; filterByOverlap; snapshotOf
  query.go      doppel query: check a proposed function (a snippet on stdin) against the corpus, locality-weighted
  config.go     .doppel.json loading (AnalysisConfig), flag precedence, hookParams
  hook.go       doppel hook session-start / user-prompt / pre-tool / stop: the four Claude Code
                hook entry points, and baseline file I/O
  diff.go       doppel diff: match two snapshot files' functions to each other; --output writes the delta report as markdown; exit codes 0/1/2
  version.go    build identity, for deciding whether a baseline is still comparable
  ontology.go   doppel ontology: print the vocabulary, check its axioms
internal/
  syntax/       The language-neutral IR: Kind/Role/Node/Func/File and Inspect. Imports nothing from this module
  gofront/      The Go frontend: the only package that imports go/ast. gofront.go maps *ast.File → syntax.File and runs canon; syntax_map.go is the node-for-node mapper
  lexfront/     The language-agnostic frontend: spec.go is the per-language table, lexer.go tokenizes, segment.go finds functions, build.go builds the shallow tree
  parser/       frontend.go owns the Frontend interface, the extension registry, IsTestFile and SameBuildUnit; parser.go is the neutral syntax.File → CodeUnit projection (and owns ShouldSkipDir, the walk rule cmd and bench share); signals.go extracts the tagger's evidence channels over the IR; go_parser.go and lex_parser.go are the two registry adapters
  canon/        Canonical Go AST shapes with a per-function rule log: canon.go + rules.go (the rule
                set and canon.Version), alpha.go (rename), clone.go, key.go. Go-only by nature;
                imported by gofront (to run it) and by analyzer (to name its rules)
  fingerprint/  Token shingles + control-flow histogram + signature types over the neutral IR; the code-similarity score
                wl.go is the WL label bag; wlexplain.go names its shallow labels for reports;
                cons.go hash-conses the canonical forest; wlcodec.go encodes bags for the snapshot
  ontology/     The formal vocabulary: entity kinds, typed relations, concept taxonomy, roles, axioms
  tagger/       The 14 seed rules: AST-signal matching → founding member sets for the lexicon
  lexicon/      Learns the corpus's own concepts: features.go (evidence channels), expand.go (seeded PMI expansion), emerge.go (clique clustering), name.go
  clique/       Deterministic maximal-clique enumeration and components, shared by family and lexicon
  concepter/    ConceptDoc; callgraph.go (BuildCallGraph); role.go (ClassifyRole, role constants);
                calltokens.go (the resolved-call token view retriever, culture and lexicon share)
  mapper/       Where enrichment actually happens: callers, role classification, aggregated concepts/packages
  retriever/    Multi-channel candidate retrieval: shape.go / concept.go / calls.go inverted indexes, retriever.go union + evidence
  culture/      Corpus-culture model: ecology.go (PMI), prototype.go (prototypes + typicality), habitat.go (fit), convention.go
  analyzer/     SimilarPair + Retrieval types; FindSimilar (library API); rank.go: SortForReport (the report's ranking) and SortByEvidence (the plain-Total library one); kind.go + stem.go (pair kinds); explain.go (rule-attributed pair sentences)
  comparator/   Weighted structural overlap scoring (12 signals → 0.0–1.0 composite)
  family/       Near-duplicate families: components + edge completion + maximal cliques over the pair graph
  snapshot/     One analysis run as comparable plain data: schema + Build, and Diff over two of them
  identity/     Matches two snapshots' functions to each other by WL bag and classifies each into one of eight classes; render.go is the text + JSON report
                delta.go adds the pairs those changes created or dissolved; deltarender.go is the delta report's text + markdown form
  reporter/     Plain-text (stdout), Markdown (--output), JSON (--format json), and the five hook digests
                (impact.go: ConceptDigest, ImpactDigest, AgentDigest; scope.go: ScopeDigest, AdviceDigest)
                overview.go + mermaid.go render the corpus model into the markdown report only
  dashboard/    The HTML dashboard (--output *.html): payload.go is the semantic payload,
                render.go inlines it into assets/ (shell.html + app.js + app.css + vendor/)
  bench/        Measurement harness: golden-ranking scorer, the pinned public corpus ladder, per-stage benchmarks, example generator
examples/       Committed real reports for each corpus rung, plus labels/ (committed golden reviews) — see examples/README.md
```

Six helpers are deliberately shared rather than copied, because doppel found each
of them as an exact clone of itself: `parser.ShouldSkipDir` (the walk rule — `cmd` walks
with it and `internal/bench` mirrored it by hand, which is how the harness could have
silently measured a different corpus than the tool), `snapshot.RelSlash` (the path rule
`cmd` and `reporter` both described as "mirrors the snapshot's"), `fingerprint.PrintType`
(the type rendering `parser` needs for `Signature`, wrapped there only to keep its `"?"`
fallback), `cmd.validateMode` (one check for `--tests` and `--generated`, parameterized
by flag name), `internal/clique` (the Bron–Kerbosch enumerator `family` needed for the pair
graph and `lexicon` needs for the feature graph, with the same non-transitivity argument),
and `concepter.Graded` (the `[]parser.Concept` → `[]ontology.WeightedTerm` conversion both
`comparator` and `retriever` need). Do not reintroduce a local copy of any of them.

Dependency directions that must hold: `analyzer` imports `comparator` (for the `Evidence` field), so
`comparator` must never import `analyzer`. `parser` imports `fingerprint`, so `fingerprint` must
never import `parser` — it works on `syntax.Node` directly, and imports no frontend. `syntax`
imports nothing from this module, the same rule `ontology` and `clique` follow. `gofront` imports
`syntax` and `canon` and must never import `parser`: a frontend depending on the registry that
dispatches to it is the cycle the split exists to avoid. `canon` imports nothing from this module
either — it is `go/ast`-typed and Go-only, imported by `gofront` (which runs it) and by `analyzer`
(which names its rules for a report), and by nothing else. `ontology` imports nothing from this
module and must stay that way: `tagger`, `concepter` and `comparator` all depend on it. `retriever`
imports `parser`, `fingerprint`, `concepter`, `ontology` and must never import `analyzer` or
`comparator` — `cmd` bridges retriever candidates into `analyzer.SimilarPair`. `culture` imports
`parser`, `concepter`, `fingerprint` only (not `ontology` directly — it is a count model over
concept names) and nothing imports it except `cmd`, which bridges its findings into
`analyzer.CultureNote`. `family` imports `parser`, `fingerprint`, `analyzer` and `clique`; `cmd`
and `reporter` import it. `lexicon` imports `parser`, `fingerprint`, `concepter` and `clique` and
must never import `ontology` or `tagger` — it learns names, it does not reason about a vocabulary,
and `cmd` bridges its concepts into an ontology term table. `clique` imports nothing. `dashboard`
imports **nothing from this module at all** — its payload is plain data and its renderer is
`html/template` plus `embed` — which is what keeps the page's data contract from quietly acquiring
pipeline types; `cmd` bridges a finished run into it, exactly as it does for `reporter.Overview`.
`identity` imports `fingerprint` and `snapshot` only, and `reporter` imports `identity` — that edge
is one-way and must stay so: `identity` models what happened between two runs, `reporter` renders
it, and an import the other way would put a digest's byte budget inside the matcher.

## Two scores, deliberately unblended — and a third quantity that ranks

Each pair carries two independent similarity numbers, gated by two independent flags:

| Score | Source | Flag | Means |
| --- | --- | --- | --- |
| `Score` | `fingerprint.Similarity` | `--threshold` | how alike the two bodies are (reported as `code-shape:`) |
| `Evidence.OverlapScore` | `comparator.Compare` | `--struct-min` | how much architectural context they share |

Do not merge these into one number. High code score + low overlap is a *different finding* (lookalike
bodies in unrelated subsystems) from high on both (a real merge candidate), and collapsing them
destroys that distinction.

A **third** reported quantity, `Breakdown.Containment`, is gated by no flag and blended into
neither: `Σ w·min(a,b) / min(Σ w·a, Σ w·b)` over the same WL bags and corpus weights `Score`'s
shape component reads. It answers how much of the *smaller* body's structure the larger one also
has, where the shape score divides by the union — so an inlined helper reads low on shape and
near 1.0 on containment. It is reported on all four surfaces — text (`containment:`), markdown,
`--format json` (`Pair.Containment`), and the dashboard, where it sits in the pair's score row and
as the sixth bar of the code-shape breakdown — and never enters ranking, filtering or
`MergeWorthy`. See *Fingerprint scoring*.

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
and the pre-commit hook both assume it. Nearly every task is a one-line wrapper, so raw `go` works too.

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

task dashboard     # analyze this repo to an HTML dashboard and open it
task dashboard-dev # the same, reading assets off disk (DOPPEL_DASHBOARD_ASSETS)
task snapshot      # analyze . --format json
task baseline      # the T0 record: golden scorecard + example checksums -> examples/baseline.json
task ablate        # zero each fingerprint blend component in turn and re-score the labels
```

`task dashboard` is the one task that is not a single command — it renders and then opens the
page.

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
- **CI checks gofmt** on the Linux leg, so formatting drift no longer reaches `master` when the
  local hook is inactive (it only exists after `task setup`, which nothing enforces).
- `.github/workflows/release.yml` fires on a `v*` tag push and runs GoReleaser (pinned `v2.17.1`)
  over `.goreleaser.yaml`. It **re-declares** vet/test/determinism in a `verify` job rather than
  reusing `ci.yml`: a tag push matches neither of `ci.yml`'s triggers, so `needs:` cannot reach it
  and `workflow_run` never fires. `contents: write` is scoped to the release job alone. `verify`
  also asserts that `-X …/cmd.version=$GITHUB_REF_NAME` actually reaches `doppel version`, because
  a binary shipping an unstamped identity degrades every comparability check silently instead of
  failing — that assertion and `.goreleaser.yaml`'s ldflags line must move together.
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
- `.github/workflows/pages.yml` publishes the site at `lukasselin.github.io/doppel`:
  `.github/pages/index.html` (the landing page) and `examples.html` (the ladder index, with one
  `@@CARDS@@` slot) are authored here, and the reports are generated — doppel's own source at
  `/report.html`, and every rung of the pinned ladder under `/examples/`. Those seven come from
  the **same corpora cache `examples.yml` writes**, restored under the same key, so the site and
  the committed Markdown reports describe the same seven trees; a rung that is not fetched fails
  the job rather than publishing a shorter ladder than the manifest pins. Output goes to `_site`
  because `parser.ShouldSkipDir` skips underscore-prefixed directories, so the site cannot join
  the corpus of the run writing into it. `@@COMMIT@@`/`@@BUILT@@` are substituted here and never
  inside a report, because an unchanged tree must render byte-identical HTML. Deliberately no
  `paths:` filter, unlike `examples.yml`: the landing page and the report both move with any
  commit, and a stale published page looks current.
- `.github/workflows/examples.yml` regenerates `examples/` from the pinned corpus ladder on
  every push to `master` that touches Go code, and commits the result as `github-actions[bot]`.
  The ladder is restored by `actions/cache` keyed on `internal/bench/corpora.go`, so the
  recurring cost is the analysis rather than a few hundred megabytes of clone, and a bumped
  rung invalidates the cache exactly when it should. It commits only when a report's *content*
  moved — the generator excludes the `| doppel |` provenance row from its comparison, or an
  unchanged ranking would rewrite all seven files on every push. A push made with
  `GITHUB_TOKEN` triggers no workflow, so the bot commit neither loops nor re-runs `ci.yml`.
  It runs on **push to `master` only, never on a pull request**, so a branch is never gated on
  its examples being current — but master's own copy self-heals within one push either way,
  which is why regenerating in the same change that moves ranking is still the rule.
- `.github/workflows/pages.yml` renders doppel's dashboard for doppel's own source
  (`go run . analyze . -o _site/report.html`) and publishes it to GitHub Pages on every push to
  `master`, with `.github/pages/index.html` as the landing page (`@@COMMIT@@`/`@@BUILT@@` are
  substituted at build time). Deliberately **not** path-filtered: any change to the tool can change
  the report. `_site/` is chosen because `parser.ShouldSkipDir` skips underscore-prefixed
  directories, so the output cannot join the corpus of the run that writes it. Provenance is
  stamped into the landing page only, never into the report — `ci.yml` diffs two `-o *.html` runs
  and a timestamp inside one would fail it.
- `.gitattributes` forces LF for Go/shell/markdown/config so the bash hook works under Git Bash on Windows.

## Key types

- **CodeUnit** (`internal/parser/parser.go`) — one function/method, projected from a
  `syntax.File` by `unitsFrom`: `Name`, `File`, `Lang`, `StartLine`, `Body`, `Signature`,
  `Package`, `Concepts`, `DocComment`, `Exported`, `ReceiverType`, `Callees`, `Fingerprint`,
  `Generated`, `Signals` (the seed rules' evidence channels), and `Canonical`/`CanonRules`
  (the frontend's canonical body and the log of what its canonicalizer rewrote — the WL bag
  and the hash-cons are built from the first, `analyzer.Explain` reads the second; both are
  empty for a language with no canonicalizer). `Lang` is what `SameBuildUnit` compares;
  `Package` falls back to the containing directory for a language with no package clause,
  which is all `culture`'s habitat model needs since it treats the key as opaque.
  Methods are named `"*Server.Start"` — the receiver keeps its
  star; `parser.MethodName` strips it back off. `Signature` is rendered text — `([]int) (int)`,
  types in order, names dropped, one entry per declared name — and is what the `sig:` line and the
  interface-implementation kind read; `Fingerprint.Types` (the sorted `in:`/`out:` type *set*) is
  what the similarity score reads. (For a long time `Signature` was empty on every unit: the old
  extractor handed an `*ast.FieldList` to `go/printer`, which rejects it silently.)
- **Concept** (`internal/parser/parser.go`) — `{ID string; Confidence float64}`, one graded
  membership; `CodeUnit.Concepts` holds them ascending by ID. It replaced a bare `Patterns
  []string`, and the rename was deliberate: every reader became a compile error, so none could
  silently keep a boolean view of a graded fact. `parser.ConceptIDs` is the boolean projection —
  every caller of it is a place that must *not* see corpus-relative weights, the merge-signal gate
  above all — and `parser.Certain` builds confidence-1 memberships for the callers that legitimately
  have bare IDs (test fixtures pinning behavior that has nothing to do with confidence).
- **Fingerprint** (`internal/fingerprint/fingerprint.go`) — `WL` (the Weisfeiler-Lehman label
  bag: rounds 0..3 merged, sorted ascending by label, built from `syntax.Func.Shape` — the
  frontend's canonical body where it has a canonicalizer, the body as written where it does
  not), `Shingles` (sorted, deduped 3-gram hashes),
  `Flow` (control-flow histogram), `Depth` (nesting-entry histogram), `Types` (normalized
  param/result types) and `Nodes`. **`WL` is the only structural multiset left**: it feeds the
  `wl` code-shape component *and* the shape retrieval channel (see *Trophic structural energy*),
  so the number that retrieves a pair and the number that ranks it read one feature set. Each
  `LabelCount` carries `H` (the refinement round) and `Kind` (the node kind the label was computed
  at) alongside the count, in padding the two-field struct already wasted — `Count` is `int32` to
  keep it at 16 bytes, which is why the merge loop's cost did not move. `Shingles` now feeds only
  `snapshot.Digest` — the digest answers "did this body change" about the code *as written*, which
  is a different question from the canonical shape the score reads. `Build` reads **two** trees
  for exactly that reason: `Body` for the token stream, the histograms and the node count, and
  `Shape()` for the bag. The zero value means "no body" and never matches anything.
  `fingerprint.LabelIDF` (built once per run in `index()`) is the corpus surprisal `ln(N/df)` of
  every WL label, over presence df; an unseen label answers `ln(N)` (df 1), never 0 — a zero
  weight does not make a label neutral in a ratio, it makes it invisible.
- **Term / Ontology** (`internal/ontology/ontology.go`) — the vocabulary: four disjoint rooted trees
  (`entity`, `relation`, `concept`, `role`) carrying definitions, relation weights, and `Validate()`.
  Role IDs are exactly `ClassifyRole`'s return values. Concept leaves are per-run: the abstract
  interior is authored, the concrete leaves come from `DerivedConceptTerms` — see *The ontology*.
  `WeightedTerm` is a term plus a confidence, which is what the graded scorer methods take.
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
- **Concept / Feature / Model / Options** (`internal/lexicon/lexicon.go`) — a learned concept with
  its weighted vocabulary, its `Scale` and `Floor`, and its `Seed`/`Anchor` provenance; `Model`
  carries the concepts, the per-unit assignments (positional, like `docs[i]`), and `Stats`. Like
  `culture`'s and `retriever`'s, none of `Options` is a flag.

### Candidate retrieval

Retrieval is an information-retrieval stage, not a similarity ranking: its job is recall — get every
pair with enough informative shared evidence in front of the comparator cheaply — and its unit is
**evidence mass in nats**, `Σ ln(N/df)` over shared rare features. All three channels share the
skeleton: build an inverted index, drop features with `df < 2` (can't pair) or `df > cap` (corpus
idiom, zero evidence), accumulate shared mass per neighbor in ascending feature order, keep each
function's top `--channel-k` (default 5) by `(mass desc, idx asc)`.

| Channel | Features | Cap (Options) | Extra gates |
| --- | --- | --- | --- |
| shape | Weisfeiler-Lehman labels (`fingerprint.WLBag`, presence-df IDF, min-count multiset mass) | `MaxLabelDF` 50 | `--min-nodes` eligibility; admits only pairs with exact `code-shape >= --threshold`, probing at most `4*ChannelK` neighbors |
| concept | learned concepts + non-root taxonomy ancestors (enumeration only) | `MaxConceptDF` 250 | none — evidence is `Scorer.SharedInformationW` (`Σ min(conf)·IC(LCS)`) over the membership sets |
| call | resolved internal callees (qualified) + import-qualified external calls via `RefPath` (full import path) | `MaxCallDF` 50 | none; bare names and variable-receiver calls are never tokens |

The union is deduped on `(min idx, max idx)`; every union pair then gets **definitive** evidence on
all three channels regardless of which admitted it, plus the exact `fingerprint.Breakdown`
(memoized). Summing the three masses into `Total` is coherent because all three are log-evidence
over the same corpus of N functions — do not normalize the components before summing.

Consequences worth knowing:

- **The absolute caps are not one number in nats, and that was measured and kept.** A cap of
  50 is `ln(N/50)` nats of required information: ≈1.5 on cobra, ≈5 on moby. `Options.MinIDF`
  replaces both caps with one floor — `cap = ⌊N·e^−MinIDF⌋` with each channel's own N
  (shape-eligible units for labels, all units for calls; a derived cap below 2 is not clamped,
  the channel is honestly empty and `Stats` says so) — and `TestMinIDF`/`TestMinIDFLadder`
  (guard `DOPPEL_BENCH_MINIDF=1`) measured it. Small corpora reproduce the fixed caps at 1–1.5
  nats (cobra: 826 candidates either way; a 1.0 floor reads merge 5.0 / fp 40.7 against the fixed
  5.2 / 40.0). The large corpora do not: a 1.0 floor derives caps of 2158/2812 on moby, grows its
  union 17 471 → 25 813 (+48%; prometheus +37%, hugo +17%), suppresses nobody, and reshuffles the
  top ten — no junk enters, but prometheus's `jaroWinkler` and `addBuckets` pairs leave its top
  eight — at half again the compare cost, with no label able to certify it as better. **Not
  adopted**: `MinIDF` is an Options-only measurement seam (no flag, default 0), and the absolute
  caps stay because what they do on the large rungs is visibly load-bearing and unlabeled. The
  adoption rule, should gin/chi labels arrive: golden green on every labeled corpus, cobra merge
  mean not worse and FP mean not lower, no corpus suppressing > 2× more functions than fixed, and
  the large-corpus top-20s reading at least as well.
- A label/token in *every* unit has `idf = ln(N/N) = 0`; zero-mass neighbors are never admitted.
  The 130-clone `Error()` bucket exceeds the df cap entirely — those functions contribute no
  structural candidates and can only enter via concept/call evidence, which is the intended
  common-idiom suppression (no name-based heuristics anywhere).
- **The df cap bites at the shallow rounds and almost never at the deep ones.** A depth-3 label
  is a near-fingerprint of the subtree under it, so on any corpus most h=3 labels have df 1 or 2
  and the `≤ 50` bound never engages there. Suppression is therefore a property of h=0/h=1, and
  the deep rounds are close to pure "is this an exact clone" evidence at maximal weight. That is
  what makes exact agreement superlinearly better rewarded than partial agreement — agreeing at
  h=3 on a node implies agreeing at h=0..2 on it, so a deep match is counted four times where a
  shallow one is counted once. It is honest under the channel's own arithmetic and it is why a
  trivial-but-unique body can now out-earn a substantial partial match; see *Trophic structural
  energy*.
- The concept channel indexes ancestors so two functions with no concept in common can still meet
  under a shared parent, but the *evidence* is always the graded shared information of the
  membership sets — a pair meeting only at a shallow ancestor earns only that ancestor's small IC,
  and a pair where either side barely carries the concept earns only what that side claims.
  **Postings are unweighted**: a unit posts under a concept it is a member of at all, so `df` keeps
  meaning "how many functions carry this", which is the quantity the cap is stated in. Confidence
  bends the evidence, not the index.
- **The channel got much louder when concepts became learned**, and that is the main behavioral
  shift of the change. On doppel's own corpus it went from 63 admitted pairs to ~1300, and
  concept-only pairs from 3% of the union to 43%; on moby, 2410 → 12050. A fixed fourteen-tag
  vocabulary simply could not distinguish enough functions to retrieve on. The golden benchmark is
  what judges whether that is recall or noise — see *Development*.
- `ontology.Scorer.SharedInformationW` exists precisely so retrieval never recomputes mass as
  `Σ ic.Of(m.LCA)` — that hits `Of("")` (the unknown sentinel) for unknown-term self-matches.
- `Stats.Suppressed` / `Stats.LargeBuckets` are diagnostics on stderr, not gates.
- On the large reference corpus (~8k functions): ~22k union pairs, a few seconds end to end (the
  old all-pairs scoring took ~20s), ~60% of pairs call-only, ~9% concept-only.

### Trophic structural energy

The shape channel's features are the **Weisfeiler-Lehman label bag** (`fingerprint.WLBag`, built
in `Build` from `syntax.Func.Shape` — the frontend's canonical body where it has a canonicalizer):
one label per node per refinement round h = 0..3, merged into one multiset. h=0 is the node kind
(`IF`, `RETURN`, `CALL/Errorf` — a call keeps its callee name, identifiers collapse to `ID`); each
further round folds in one more edge of children, so h=3 at an if-statement is that whole guard
and h=3 at a loop is its body. `LabelIDF` is the same presence-df `ln(N/df)` the other channels
use.

`wlLabel0` is the one place `syntax.Kind` and `fingerprint.LabelKind` meet, and the mapping is
total and one-to-one on purpose. `LabelKind` is this package's own enum rather than the IR's
because the label is a hash of the kind's *name*: the name has to be pinned where the scoring
lives, or the IR could reword a kind and silently move every score in the tool. `labelKindNames`
is the hash input, `labelKindWords` the reader-facing word `wlexplain` prints — one table each,
index-aligned, so a kind cannot be named one thing in a sentence and hashed as another in a bag.

**The multi-level pattern multiset this replaced is gone** — `fingerprint.Pattern`,
`extractPatterns`, `l0ExtraWidths`, `pattern.go` and `defuse.go` in full, which means the L0 token
n-gram windows at widths 3 and 5, the L1 call/operator shapes, the L2 statement renders
(`if(bin:!=(id,nil))`), the L3 loop-call summaries and statement bigrams, and the **entire L4
def-use flow pass** (`flow:call:Open→call:Close`) existed only as pattern features and were
deleted with them. Five hand-built extractors, each with its own idea of what was salient and its
own way of being wrong about it, and overlapping: an L0 3-gram over a return and the L2 render of
that same return were two spellings of one fact, both indexed, both weighted. A WL bag is that
ladder from one uniform recurrence, with no extractor deciding what is worth naming — and it is
the multiset the *score* already reads, so one feature set now serves retrieval and ranking.

What is lost is the **render**. A pattern's `Render` string was the hash's own serialization, so
explanation and hash could not drift. A WL label is a hash of a subtree and has no short faithful
name: the `shared structure:` block now prints `fingerprint.DescribeLabel` — `depth-2 IF ×3`, the
round and the node kind, with the multiplicity when above 1. That is a real, checkable claim ("a
guard two levels deep matched exactly, three times") and a strictly weaker one than
`if(bin:!=(id,nil))` was, because it does not say *which* guard. Naming the subtree would mean a
second serialization of the thing the hash already is, which is the drift the pattern levels spent
their renders avoiding. `LabelKind` is an enum whose `String()` is the hash input, so the
vocabulary cannot fork; `Label` travels on `SharedChain` so a consumer can join on the identity.

Three quantities per pair, all from one sorted-intersection pass (`pairEvidence`):

- **Shape evidence** = `Σ idf·min(count)` over cap-surviving shared labels — shared structural
  energy, the retrieval mass.
- **TrophicSimilarity** = `2·SharedEnergy / (E_A + E_B)`, reported as `trophic:`, where energy on
  both sides is **cap-surviving (informative) energy only**. Exact clones of a rich function read
  1.00; an idiom bucket whose every label exceeds the df cap reads 0/0 = 0.00 (`DataSourceName ↔
  Error`); everything between is the fraction of informative structure the pair shares. Two exact
  twins whose shape sits *between* df 2 and the cap legitimately read 1.00 with small energy — the
  energy ranks, trophic explains.
- **Shared labels** = the highest-energy shared labels, `(energy desc, depth desc, label asc)`,
  top `ChainTopN` (3 default, 20 under `--debug`; `-1` unbounded, `0` none). Depth descending is
  the WL analogue of the old level ranking — an h=3 label folds three edges of context, so it is a
  more specific claim than an h=0 node kind at equal energy. Every shared label is a candidate now
  (the pattern channel offered only its L2+ levels), so the top N is **selected by insertion**
  rather than sorted: a substantial pair shares hundreds, and sorting them all per pair over tens
  of thousands of pairs is real cost in a stage that exists to be cheap.

Trophic explains; it never ranks (`Total` stays Shape+Concept+Call) and never blends into
code-shape or overlap.

**The consequence that moved a default.** Because every body produces `wlRounds+1` labels per
node, a *trivial* body that happens to be corpus-unique earns maximal-IDF evidence at h≥2, where
the pattern hierarchy gave it nothing (a one-liner has no loop, no bigram and no def-use edge, so
it could only earn L0/L1 mass, which the df cap ate). On cobra this put `commandSorterByName.Less
↔ byName.Less` — a 16-node one-liner, code-shape 1.00, shape mass 107.8 of which every deep label
is df=2 — at rank 20, tripping the golden benchmark's no-false-positive-in-the-top-20 assertion.
**`--min-nodes` was recalibrated 12 → 18 in the same change** (and later relaxed to 16, the
lowest value that still holds the pin — see *Fingerprint scoring*), being the guard that exists for
exactly this ("one-line accessors match each other at 1.0 and flood the channel") and whose 12 was
calibrated against the retired feature set. See *Fingerprint scoring* for the recall nuance and
*Rough edges* for the alternatives that were measured and not taken.

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
**Associations are still deliberately unsurfaced per-pair** — an association annotates the corpus,
not a pair. That argument is what decided where they *do* belong: the markdown report's **Local
practice** section, which is corpus-level by construction. A `doppel culture` command would be the
next home for the full list, which the report bounds.

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
- **Subsystem rollup.** With `Options.Root` set (the pipeline passes the analysis root; tests
  leave it empty and see no subsystems), a second partition models **subsystems** — the parent
  of each file's directory, slash-relative with a trailing slash (`tpl/`, `internal/`;
  `subsystemKey`) — with the same features, weights and math (`buildHabitatModel` is
  partition-agnostic; a coarser habitat just has a larger m). A unit is a **Misfit** only when it
  is alien at every level that can judge it: a package misfit whose subsystem (when modeled)
  says it fits is **excused** — drift across a directory, not a function out of place. `Stats`
  carries `SubsystemsModeled` and `MisfitsExcused`; the stderr line becomes `Habitats: 126
  modeled, 538 misfits (121 excused by subsystem), 31 subsystems; …` and stays byte-identical
  to the old form when no subsystem is modeled; a confirmed misfit's report line adds `;
  subsystem tpl/ fit 0.30`. Package-level superlatives and every package pin are unchanged.
  `PackageMisfit` is the raw rule for diagnostics. On the ladder: hugo 659 → 538, prometheus
  340 → 195, moby 253 → 101.
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
  (the flat `concepts:` line stays); extinct candidates + rounds under `--debug`; stderr
  `Ecosystems: N profiled (…)`. Profiles never rank.

On real corpora the flagship behavior is visible: units tagged `validation, db_access, mapping`
(fixture-string tags) equilibrate to `validation 1.00 (dominance)` — the unsupported tags go
extinct because they explain none of the surrounding evidence.

### Fingerprint scoring

`fingerprint.Similarity(a, b, idf)` blends four components; weights are constants and sum to
exactly `1.00`.

| Component | Metric | Weight |
| --- | --- | --- |
| WL labels | corpus-weighted multiset Jaccard over the WL bags | 0.60 |
| Control flow | cosine over the node-kind histogram | 0.20 |
| Nesting depth | cosine over the entry-depth histogram | 0.05 |
| Signature | Jaccard over normalized param/result types | 0.15 |

The 0.60 slot used to be Jaccard over hashed 3-grams of a flattened token stream. A shingle is a
window over a *linearisation*: it cannot tell a condition from the statement it guards, and two
shingles adjacent in the stream may be nowhere near each other in the tree. A WL label is a
subtree summary, so sharing one is a claim about shape. The metric is

    jaccard = Σ w·min(a,b) / Σ w·max(a,b),   w = ln(N/df)

with `w` the label's surprisal in *this* corpus, so **code shape is corpus-dependent now** — a
structure every function in the repo carries weighs exactly 0 and is not a finding. The blend
weights themselves did not move: re-tuning them in the same change as the metric would leave
nothing to attribute a ranking move to.

`idf` is threaded explicitly rather than read from a global, so every call site says which corpus
it is scoring against. `index()` builds exactly one `LabelIDF` after the population filter and
hands it to retrieval, calibration, family edge completion and the query probe; a nil idf means
uniform `w = 1` (plain multiset Jaccard), which is the honest answer when there is no corpus and
what the package's own unit tests score under. `analyzer.FindSimilar` counts its own over the
units it is handed — the only population a library call can claim to know.

**Containment** is a third reported quantity, `Σ w·min(a,b) / min(Σ w·a, Σ w·b)` — the same
numerator over the smaller side alone. Both fall out of one sorted merge (`Σ w·max` is
`massA + massB − shared` term by term), so a pair costs one pass. It is **never** blended into
`Score`, never enters ranking, filtering or the merge verdict, and is reported in all four
formats. A helper inlined into a long function reads low on the Jaccard and high on containment,
which is a finding no single blended number states. Both quantities are `0.0` when the
denominator is — including the case where every label a pair carries is corpus-universal, the
same convention `TrophicSimilarity` uses for a pair whose every pattern is idiom.

The depth histogram (`Fingerprint.Depth`, 6 buckets, deep tails folded into the last) records the
nesting depth each control-flow node is *entered* at; the seven statement-bearing constructs (if,
for, range, switch, type switch, select, funclit) push a level for their children. It exists
because flattened tokens carry no depth: sequential ifs and nested ifs used to have identical
token bags, identical flow histograms, and score 1.0. Depth's 0.05 was carved entirely out of
Flow (0.25 → 0.20) — nesting is flow-adjacent, so flow pays for it. Rendered as `nesting:` in the
breakdown line. A nesting change is a body change: `snapshot.Digest` hashes Depth (snapshot
Schema 4).

`SizeRatio` is reported in the `Breakdown` but **not** scored — Jaccard already penalizes size
mismatch through the union, so damping again would double-count it.

Three mechanisms do the heavy lifting, and all are load-bearing. Two are canonicalization rules
shared by the token stream and the WL labels: identifiers collapse to `ID` (so renamed clones
still match), while call *selector* names survive as `CALL:Errorf` (intent-bearing) with the
receiver expression dropped (`e`, `s`, `cfg` are arbitrary). The third is the corpus weighting
above — the token stream had no way to say that a shared shape was unremarkable, so a body made
entirely of the repo's own idiom scored as high as a genuine clone.

`--min-nodes` (default `16`) excludes tiny bodies from the **structural retrieval channel** (and
from `FindSimilar`). Without it one-line accessors match each other at 1.0 and flood the channel.
Concept and call retrieval deliberately ignore it — a small function with rare tag or call evidence
is still worth comparing.

**It was `12`, went to `18` when the shape channel moved to WL labels, and is `16`.** The number
was always a property of the feature set, not of Go: under the pattern multiset a trivial body was
suppressed *implicitly*, because a one-liner has no loop summary, no statement bigram and no
def-use edge and could therefore only earn L0/L1 mass, which the df cap ate. A WL bag has no such
floor — every body emits `wlRounds+1` labels per node, and the deep ones are df 1 or 2 whenever
the body is corpus-unique, so a trivial-but-unique one-liner now earns maximal-IDF evidence. The
measured case is cobra's `commandSorterByName.Less ↔ doc.byName.Less` (`return c[i].Name() <
c[j].Name()`, **15** AST nodes a side, code-shape 1.00, trophic 1.00): at `12` it rose to rank 20
and tripped the golden benchmark's no-false-positive-in-the-top-20 assertion.

`18` cleared that pin but overshot, closing the shape channel on small corpora (see *Rough
edges*). **`16` is the lowest floor that still holds it**, and it is pinned from both sides now:
conc's `ResultContextPool.Wait ↔ ResultErrorPool.Wait` is 16 nodes a side and genuinely identical,
so 16 admits it where 18 did not. The two pins are one node apart, which is what makes an absolute
constant sufficient — `16`, `17` and `18` score identically on the cobra labels (merge 4.5,
refactor 13.7, fp 47.0, no violations) while `15` and below re-admit the false positive at rank
20. `TestMinNodesLadder` is that measurement.

The recall nuance is worth stating plainly, because it cuts the other way: **at `--min-nodes 12`
all 18 cobra labels reach the comparator; at the shipped `16` one does not — and the one that
leaves is the false positive.** Merge recall is 6/6 either way and the merge mean is 4.5 either
way. A gate measured only as "labelled pairs retrieved" would score the old value higher, which is
why the golden benchmark asserts on the false-positive side as well.

**It also interacts with corpus-derived floors, in one direction.** The shape null is drawn over
the `--min-nodes`-eligible units, so raising the floor shrinks the null population — on conc (81
functions) past the 1 000-pair minimum, which declines calibration there entirely. See
*Calibration*. The flag is hidden on every command (`MarkHidden`, alongside `--channel-k` and
`--max-per-func`) but still parses and still has its config key; `cmd.defaultMinNodes` is the one
definition.

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

### The learned lexicon

`internal/lexicon` learns this corpus's concept vocabulary instead of asserting one. It is the
answer to the question the tagger could not scale to: fourteen hand-written rules over string
channels are fourteen guesses about how *some* codebase writes something, they do not port to a
repository that calls its database wrapper `store`, and they do not port to another language at
all.

**The rules survive as seeds.** `tagger.Tag` still runs and still matches the same AST evidence,
but its output is a founding *member set* — which functions to look at — not a label. What the
concept turns out to be is whatever those functions share that the rest of the corpus does not.

Stages, all deterministic counting:

1. **Features.** Per unit, a channel-prefixed set from material the frontend already produces:
   `sel:` selectors (nested tail included, so `c.httpClient.Do` gives `httpClient.Do`), `imp:`
   import paths, `id:` identifier stems split on camel/underscore boundaries, `lit:` the leading
   token of each string literal, `call:` resolved call tokens, `act:` the fingerprint's
   Weisfeiler-Lehman labels at h=2 on statement kinds, `flow:` binarized control-flow labels plus
   the go/select/chan flags.

   The `act:` channel read the multi-level pattern multiset's L2/L3/L4 renders when it was
   written; that multiset is gone, and WL h=2 is its closest analogue — the round at which a
   label folds in a statement node's grandchildren is the statement *with its structure*, which is
   what L2 carried. `actionKinds` (`lexicon/features.go`) is the one judgement in that file, and
   it is about shape only: statements, calls and assignments, never what a construct is *for*. h=2
   and not h≥2, because an h=3 label at a statement describes a region and its evidence is already
   implied by the h=2 labels beneath it — indexing both would double-count one piece of structure
   in a channel whose weights are lifts. A feature name is `depth-2 IF#<hex>`: the hash keys it,
   the description keeps it minimally legible if it ever surfaces through `Concept.Definition`.
   `act:` is not in `seedChannels` and not in `nameableChannels`, so it can neither found a
   concept nor name one — it only ever adds vocabulary to a concept someone else's evidence found.
2. **Information window.** Keep features with `df >= MinDF` (2 — one function can relate nothing)
   and `ln(N/df) >= MinIDF` (1.0 nat). The upper bound is derived, `cap = ⌊N·e^−MinIDF⌋`, for the
   reason `retriever.Options.MinIDF` documents: an absolute cap of 50 means 1.5 nats on cobra and 5
   on moby, and the same number should mean the same thing.
3. **Seeded expansion** (`expand.go`). For each seed, every surviving feature is scored by lift
   over the corpus base rate — `PMI(c,f) = ln( P(f | members) / P(f) )`, kept at `>= ln 2` with
   `count >= MinSupport` — and weighted `lift × idf`: distinctive *to the concept* and rare *in the
   corpus*. This is `culture/ecology.go`'s arithmetic generalized from tag~call to concept~feature.
   A seed whose members share nothing distinctive produces no concept, which is a finding rather
   than a failure.
4. **Emergence** (`emerge.go`). Features no seed claimed are clustered on their own co-occurrence:
   an edge at `PMI >= ln 2` with `count >= MinSupport`, then **maximal cliques**, then the same
   `fit` as a seed. This is the path that makes an unseeded corpus work at all.
5. **Membership.** `E(u,c) = Σ w(c,f)` over the features the unit carries, divided by the unit's
   own information `Σ ln(N/df)` over its surviving features: **coverage**, the fraction of what a
   function carries that the concept explains. Two corpus-derived quantities then do two different
   jobs — `Floor` (the founding coverage at `FloorQuantile`, 0.25) decides membership, `Scale` (the
   founding median coverage) sets what the confidence reads, `conf = C/(C+Scale)` — and a unit keeps
   its `MaxMemberships` (3) strongest. Saturating, not normalized: a concept's weights are
   `lift × idf` rather than `idf`, so coverage has no natural maximum either, and pretending it had
   one would make the number a rank in disguise.

Seven decisions here were **measured, not assumed**, and each was wrong the first time:

- **A fixed confidence cut cannot decide membership.** Confidence saturates around the median
  founding member, so `conf >= 0.5` means "at least the median founding member" and discards half
  of every concept's own seed set by construction. On doppel's own corpus it left 527 of 546
  functions with no concept at all. Hence the separate quantile floor.
- **A membership bar cannot be stated in raw evidence, and lowering the quantile is not the
  repair.** An unnormalized `Σ w(c,f)` scales with how many features a unit has, so an absolute
  floor is largely a floor on *size*: measured on doppel's own corpus the labelled units carried a
  median of 48 features and the unlabelled 21, and **451 of 866 functions were unlabelled** — of
  which 447 carried real evidence for some concept and simply could not reach its bar (best
  `evidence/Floor` p50 0.37, p90 0.87). The floors themselves ranged from 1.285 to 94.8 across
  concepts, because `fit` draws them from a founding set that is by construction the units carrying
  the clique — so one `FloorQuantile` is at once far too strict and far too loose, and dropping it
  is a cliff rather than a dial: at 0.15 the `caching`-seeded `ontology+fingerprint` concept
  swallowed 649 functions. Coverage removes the length term from both sides and takes this corpus to
  197 unlabelled. `TestMembershipIgnoresSize` is the regression as a test: a terse function
  carrying a concept's whole vocabulary and nothing else, against verbose founders. On cobra's
  labels the move is neutral where it matters and better where it does not: merge 5.3 → 5.3
  (6/6), refactor 12.8 → 12.8, false-positive separation **43.5 → 50.5**.

  It is also *cheaper*, which is not the direction more coverage suggests. The union grows — moby
  compares 40 225 pairs against 28 065, +43%, because the concept channel has more memberships to
  retrieve on — and the run still halves, moby 11.0s → 5.2s end to end. An evidence floor admitted
  a big function to a great many concepts at once, and the stages downstream of membership
  (culture's prototypes, the arena, the comparator's concept signals) pay per membership rather
  than per pair.
- **Coverage needs a per-unit bound, and the raw-evidence bar had been supplying one by
  accident.** Unbounded, coverage assigns hugo **7.4 concepts per function** and grows a
  1 268-member concept (22% of that corpus); moby reaches 4.8 per function, gin lets one concept
  claim 37%. `MaxMemberships` (3, the bounded-per-item idiom the retrieval channels use) is the
  bound. `TestLexiconMembershipLadder`/`Labels` in `internal/bench` (guard
  `DOPPEL_BENCH_LEXICON=1`) is the measurement: every K tried (2, 3, 4, 6) fixes the tail and
  **none moves a single labeled pair on cobra** — merge 5.3 (6/6), refactor 12.8, fp 50.5, no
  violations, identically — so the labels do not choose K and the corpus statistics do. At 3,
  hugo's largest concept is 4.5% and its assignments 2.4 per function.

  Two alternatives were measured on the same harness and **not adopted**. Requiring *both* the
  coverage floor and the old evidence floor doubles the unlabelled share everywhere (moby 9.4% →
  50.0%, cobra 21.9% → 49.1%) — it is strictly stricter than either, so it can only reintroduce the
  bias coverage removes. Admitting on *either* floor is the more tempting one, and it does buy
  coverage (moby 9.4% → 7.0%, hugo 6.3% → 4.8%, gin 15.3% → 9.9%): it costs 4 points of
  false-positive separation on the one labeled corpus (fp 50.5 → 46.5, merge and refactor
  unchanged), and two of the three hard assertions are on the false-positive side. Both seams were
  deleted rather than kept, since neither is an operating point anybody should reach for; the
  variants live in the bench test.

  The flip side is real and is the price: a function that does the concept **and** a great deal
  else now covers less of itself with it and can lose the membership, where the old bar would have
  admitted it for sheer bulk. That is the intended reading of "how much of this function is this
  concept", and it is the direction the labels prefer.
- **The feature co-occurrence graph is not sparse.** Features co-occur far more freely than
  functions resemble each other, so the unbounded graph is one blob: it tripped `MaxComponent` and
  produced one emergent concept, and none at all with no seeds. Each feature keeps its `EdgeK` (8)
  strongest associations — the bounded per-item top-K idiom the retrieval channels already use.
- **Components are the wrong enumeration unit, even sparsified.** A top-K graph's giant component
  is most of the vocabulary. Enumeration runs over **one feature's neighbourhood** instead, at most
  `EdgeK+1` vertices, so it is bounded by construction rather than by a budget. Every maximal
  clique containing a feature lies inside its neighbourhood — an extending vertex would have to be
  adjacent to it — so these are globally maximal, and each is rediscovered once per member: the
  **lowest-indexed member owns it**, or a clique of seven becomes seven identical concepts.
- **Not every feature may found a concept.** Seeding is restricted to `seedChannels`
  (`sel`/`call`/`imp`) — what the code *does*. Every channel still contributes to a concept's
  vocabulary once its members are known, but without this restriction doppel's own corpus produced
  concepts founded on the stem `and` and the literal `%d`: groups of functions that genuinely
  co-occur those tokens and mean nothing. That one restriction took 245 emergent concepts to 55.
- **`MinCliqueSize` is 2, not 3.** "A pair is not a practice" sounds safer and is wrong: the
  practices worth finding are overwhelmingly pairs of calls — Get/Decode, Marshal/Unmarshal,
  Open/Close — and at three a store wrapper reached through exactly two calls is invisible. The
  strictness is elsewhere: an edge needs `MinSupport` co-occurrences at `MinLift` nats, and a
  founding member must carry two of the clique's features, which for a pair means both.

**Naming.** A concept's ID is its top-weight features, short form, joined by `+`:
`sql.Open+QueryRow`, `json.Marshal+Unmarshal`, `store.Get+store.Decode`. Ordering is **channel
priority before weight** — a concept's heaviest feature is often an identifier stem, and naming by
raw weight produced `ref+decode` for a concept whose evidence is `store.Get`/`store.Decode`: a name
a reader cannot look up. Widening stops at `maxNameParts` (3) and falls back to a `~2` suffix. The
seed's name is kept as `Seed` and rendered as provenance, **never as the identity** — after
expansion the concept is its learned vocabulary, and a corpus where a rule fired on three unrelated
things does not get to inherit the rule's claim.

Overlapping cliques are the norm (one function does several things), so `MaxOverlap` (founding-member
Jaccard 0.6) collapses concepts that are the same group of functions said twice; earlier-kept wins,
and the iteration order is fixed, so which survives is deterministic.

There are **no flags**. Every knob is on `lexicon.Options`, like `culture`'s and `retriever`'s, and
`cmd` passes `DefaultOptions()`. A learned vocabulary is not an operating point a user tunes; it is
what the corpus says.

### Seeds

The seed table is `internal/tagger`, unchanged as a matcher and re-documented as what it now is.
Exactly 14, emitted in declaration order: `retry`, `http_call`, `db_access`, `validation`, `mapping`,
`transaction`, `caching`, `concurrency`, `error_wrapping`, then the five added with ontology 1.1.0 —
`grpc_call`, `circuit_breaker`, `serialization`, `file_io`, `logging` — appended after the original
nine so every pre-existing seed keeps its emission position. The rules name `ontology` concept terms
rather than bare strings, so a rule pointing at a concept that does not exist stops compiling, and
`tagger_test` enforces the other direction: every concrete concept in the authored taxonomy has
exactly one rule. `tagger.Concepts()` exposes the full list, which is what makes "this corpus has no
HTTP practice" reportable — see *Absence* below.

Rules match **AST evidence** (`parser.TagSignals`), never raw source text, and each channel has its
own semantics: selectors exact (`http.Get`, `sync.Map`), methods exact on the method or bare-call
name, receivers exact on the receiver identifier (`tx.Commit` fires, `mtx.Lock` does not), imports
and string-literal contents and identifier names by substring, plus node-kind flags
(go/select/chan) for `concurrency`. Consequences worth knowing — all of them now bear on *which
functions a search starts from*, not on what anything is called:

- A comment saying `COMMIT` or `DELETE` no longer seeds anything — comments are not evidence.
- `error_wrapping` is deliberately tight: a `%w` verb anywhere in a format string (the old rule
  only matched `%w"`) or a pkg/errors wrap helper. Bare `fmt.Errorf` and `errors.As`/`errors.Is`
  no longer fire it.
- `retry` and `circuit_breaker` are the two seeds with no structural handle — their evidence is
  lexical, in identifier names (`circuit_breaker` also matches a `gobreaker` import).
- String literals are still evidence, so a test whose fixture strings contain `%w` or `SELECT `
  seeds those searches. A function carrying SQL strings is db-flavored even when it is a test.
- `json.Marshal`/`json.Unmarshal` moved from `mapping` to `serialization` when that leaf arrived.
  `mapping` is purely the conversion vocabulary (`transform`, `convert`, `ToDTO`, …).
- `serialization`, `file_io` and `logging` are selector/method/receiver evidence only — an
  `encoding/json` or `os` import is file-level and near-universal, and an import-substring `"log"`
  would match `dialog` and half the module paths on earth.
- The pre-Go-only polyglot keywords (`axios`, `urllib`, `Promise.`, `await `) are gone.

### Absence

A learned concept can never be absent — it exists *because* functions carry it — so "does this
codebase already do X" has to be answered against the one fixed list left: the seeds. `Model.
GrownSeeds()` reports which seeds produced a concept, `cmd.unusedSeeds` subtracts them from
`tagger.Concepts()`, and the remainder travels in `Result.UnusedSeeds`, `snapshot.UnusedSeeds`,
the report overview and the session-start digest.

The predecessor compared a run's tags against `ontology.Default()`'s fourteen leaves. Those leaves
are seeds now and never appear in a derived vocabulary, so that comparison would have reported all
fourteen absent on every corpus — confidently, and always wrongly.

### The ontology

`internal/ontology` is the vocabulary the comparator reasons over, and the reason two functions
doing related kinds of work no longer score the same as two with nothing in common.

**The interior is authored; the leaves are learned.** `concepts.go` still declares the tree below,
but its fourteen concrete leaves are now the *seed* vocabulary. Every run replaces them with what
`internal/lexicon` found — `ontology.DerivedConceptTerms` builds the table and
`ontology.WithConcepts` assembles the run's vocabulary, carrying entity, relation and role terms
over untouched. A seeded concept hangs under its seed leaf's **parent** (a concept grown from
`db_access` is a kind of `data_store_access`, whatever this corpus turned out to mean by it); an
emergent one hangs under the parent of the seeded concept it shares the most vocabulary with, or
under the concept root when it resembles none. `Ontology` was already constructible — `New(tables
…)` predates this and `Default()` is just `New(entity, relation, concept, role)` — so nothing new
was needed to make the vocabulary per-corpus.

The abstract interior is the part that was never a claim about a codebase, and it is what
Wu–Palmer depth, Lin similarity and the concept channel's ancestor postings all read:

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

`ontology.Validate()` checks eight of the nine axioms and is exercised both by tests and by `doppel ontology`.
Axiom 8, the tagger/ontology correspondence, lives in `internal/tagger` instead: the check needs the
rule table, and importing `tagger` from `ontology` would be a cycle.

**Axiom 1 exempts concrete concept leaves from snake_case**, and the exemption is the point rather
than a loophole. Learned leaves are named after the evidence that produced them
(`sql.Open+QueryRow`), so a spelling convention meant to keep a hand-written vocabulary tidy would
only force those names to be mangled into something a reader cannot match against the code.
Everything still authored by hand — every abstract term, and every entity, relation and role —
keeps the rule. `doppel ontology` prints the *seed* vocabulary and says so: the leaves it shows are
not what any run reasons over.

`Ontology.LCA` walks the deeper term up to the shallower and then both together, using the depth
table and parent pointers. The chain-and-set form it replaced allocated a map per call, which was
fine for fourteen tags and became a quarter of a run once a unit carries a dozen learned concepts.

### Corpus-weighted relatedness

`ontology.NewCorpusICMass` turns this run's **summed membership confidence** per concept into
information content — `IC(c) = ln(1/P(c))`, add-one smoothed, ancestors accumulating their leaves —
and `ontology.NewScorer` wraps the ontology with it. `NewCorpusIC` remains as the integer-count
form and delegates. Mass rather than a member count because membership is graded: a concept twenty
functions carry firmly is more of the corpus than one twenty functions barely carry. Accumulation
is floating point now, so it runs in declaration order and never over a map. `cmd/pipeline.go`
builds the scorer and hands it to `comparator.New`; the free `comparator.Compare` stays
corpus-independent (nil IC delegates literally to the Wu–Palmer methods) and is what the regression
tests pin.

With IC loaded, term relatedness is **Lin** (`2·IC(LCS)/(IC(a)+IC(b))`) and set relatedness is
information-weighted: a matched pair contributes `IC(LCS)` exactly (Lin's denominator cancels the
pair weight), so the score is the fraction of the larger side's information that is shared. The
matcher is the same greedy **sorted by contribution, not similarity** — under IC a pair can be more
similar yet share less information, and contribution order is what stays optimal (verified against
a brute-force oracle in `oracle_test.go`/`scorer_test.go`, exhaustively).

**Confidence rides on top of IC, and only on the score.** `SetRelatednessW` and
`SharedInformationW` take `[]ontology.WeightedTerm`: a matched pair contributes
`min(conf_a, conf_b)·IC(LCS)`, and each side's total is `Σ conf·IC`. `min` because sharing is
bounded by the weaker claim — a concept one side barely carries cannot be strong shared evidence
however sure the other side is. The **matching itself stays unweighted**: it is the same
contribution-ordered greedy over the same term IDs, and confidence only rescales the pairs it
chose. That is not a shortcut. The greedy's optimality rests on IC decomposing laminarly over the
tree, an argument confidences break — a down-weighted exact match can be worth less than two
related pairs, the classic case where greedy matching is not optimal — so keeping the *choice* of
pairs weight-free preserves exactly the property the exhaustive oracle test verifies.

Two invariants worth knowing:

- **The merge-signal gate never sees IC, and now never sees confidence either.** `countSignals`
  reads `PatternSignalBest`, a taxonomy-only Wu–Palmer best match over bare concept IDs. Under Lin,
  sibling/cousin similarities move with concept frequencies elsewhere in the tree; membership
  confidence is corpus-relative twice over (the learned vocabulary and the evidence scale both move
  with the corpus); a pair's `MergeWorthy` must not flip because unrelated code shifted either.
  (The `OverlapScore >= 0.4` half of the verdict does include the weighted score, so `MergeWorthy`
  is not fully corpus-independent — the signal count and the shape floor are.)
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

**The census ranks by evidence, not size.** `Family.Evidence` sums `Retrieval.Total` over the
clique edges retrieval proposed (completed edges contribute zero — an edge retrieval never found
informative enough to propose adds shape to the guarantee but no energy to the rank), and
`sortFamilies` orders by it first. Ranked by member count, every large corpus led with its biggest
*idiom* family — moby's was 44 mutex-guarded getters, mostly stitched by completion. Under
evidence the family section and the pair list rank in the same currency and tell one story.

**Where it runs is load-bearing.** The family stage lives in the `cmd` command functions, never in
`finishAnalyze`: `cmd/hook.go` calls `analyze()` directly and snapshots `res.Pairs`, so a stage
inside the pipeline would change every baseline and delta. It reads `res.Pairs` — the full
comparator-scored, struct-min-filtered set — never the ranked slice, because `--max-per-func` is a
report-time device applied *after* scoring and a family of seven rests on 21 edges the pair list
would never show. `analyze` renders `--families N` (default 5) of them after the pair report;
`doppel families` is the census, with no presentation cutoff and its own `--format json` payload.

Families are deliberately **not** in `snapshot`: only what a consumer reads is stored, and the Stop
hook rewrites that file every turn. `analyze --format json` remains the snapshot exactly as before.

## Pair kinds

`internal/analyzer/kind.go` labels what a pair *is* when a naming rule can say so, because two
classes of finding crowded every wide corpus without the report being able to explain them. A
kind annotates: it never filters, never enters ranking, and is not in the snapshot (hook digests
never see it) — the same contract as culture, habitat and profile notes. No kind, no line.

- **`interface implementations`** — both sides are methods with the same bare method name
  (`parser.MethodName`), the same `Signature` text, on different types (receiver names differ, or
  the same receiver name in different packages — moby's `ipvlan` and `macvlan` each implement
  `Join` on their own `*driver`; pointer and value receivers normalize to one name and never pass).
  Proving interface satisfaction needs `go/types`; this is the honest middle ground, and the label
  states the package relation (`in package x`, `sibling packages x and y`, `packages x and y`)
  because the same method across unrelated packages is the weaker claim.
- **`diverged copy`** — code-shape `>= ForkShapeFloor` (0.60, pinned equal to the family edge
  cut), same or sibling package, and names that agree once version markers are stripped:
  `evalCallOld`/`evalCall`, `scrapeLoopAppenderV2.append`/`scrapeLoopAppender.append`. Methods
  fork on either axis — same method on stem-sharing receivers, or stem-sharing methods on one
  receiver. The stem rule (`stem.go`) strips, to a fixed point, a trailing `_`, a trailing bare
  digit run, a trailing `v2`/`V3`, and the markers `Original Deprecated Legacy Orig Copy Old New`
  as suffix or prefix — with three guards that keep the near-misses apart: both names ending in
  *differing* bare digit runs are a numbered series, never a fork (`sha256`/`sha512`); a marker
  strips only at a token boundary (`evalCallOld` and `foo_old` yes, `Threshold` and `Newton` no);
  every strip must leave `>= 3` characters (`GoOld`/`Go` is nothing). `decodeToml`/`decodeYAML` and
  `loadWAL`/`loadWBL` have no marker and stay merge candidates.

Precedence: fork first. The v1/v2 appenders satisfy both rules, and the fork is the claim a reader
acts on. `ClassifyFamily` applies the same rules to every member pair of a family — one shared stem,
or one method and signature on pairwise-distinct types — and the census renders it as a suffix on
the F-line, in the markdown heading, and as `kind`/`kindLabel` in `doppel families --format json`
(families are not in the snapshot, so the JSON is their machine-readable home). On the ladder the
rules land where intended: hugo's `evalCall`/`evalField` pairs read `diverged copy`, conc's pool
`WithContext`/`Wait`, gin's `Render`, chi's `Flush` and moby's 11-member `UnmarshalJSON` family read
`interface implementations`.

## Pair explanations

`analyzer.Explain` puts one sentence on **every** reported pair, saying what canonicalization did
for it and what it left behind. It is the only place canon's work is legible: a pair scores 1.00
and the number alone cannot say whether the two bodies were identical or merely converged after a
rename and an operand swap.

```
explain: identical after rename, commutative-reorder
explain: differs by one extra defer, two extra if
```

Two tiers, decided on **WL bag equality**. Equal bags means the canonical trees agree, and the
sentence names the union of `CodeUnit.CanonRules` from both sides — "fired on either side" — in
canon's declaration order, under reader-facing names (`alpha-rename`→rename,
`unwrap-block`→block-unwrap, `negated-if`→negation-flip, `guard-return`→guard-form,
`incdec`→increment-form, `commutative-sort`→commutative-reorder). That is how "a rule whose
reversal would lower the score" is implemented: **asserted, not measured** — literally reversing a
rule would mean re-canonicalizing under a modified rule set and re-scoring, per rule, per pair.
Two known overstatements, both documented at the function and neither able to move a number: bag
equality is a proxy for tree equality (hash collisions, and WL's own indistinguishability limit),
and a rule that fired on one side into what would have been a no-op on the other still gets
credit, because `canon.Canonicalize` does not keep the per-rule before/after trees that would tell
them apart.

Unequal bags render the **residual**, and the h-split there is load-bearing.
`fingerprint.LowLabels` (in `wlexplain.go`, a new file — it calls `wlLabel0`/`wlHash` rather than
re-implementing them) re-derives the h≤1 labels of a canonical tree *with* the node kind each was
computed at, kind names taken from go/ast's own type names by reflection so no parallel switch can
drift from the node set. Counting is over **h=0 only** — a node's kind and nothing else, so the
multiset difference is literally "one more defer". h=1 labels fold in a node's immediate children
and therefore change for every *ancestor* of a changed node too; counting them would report one
added defer as "one extra defer, one extra for, one extra block". They are used only to split the
no-count case in two: same kinds arranged differently *locally*, or beyond what shallow labels can
name. Kinds sort statements first, then expressions, then scaffolding (blocks, bare identifiers,
and the `ExprStmt`/`DeclStmt` wrappers, which always accompany what they wrap), capped at three
plus a count — an explanation is a sentence, not a dump. Sides are not named: the difference is
symmetric.

Where it renders: the text pair list (`explain:`), the markdown pair list (`**Explain:**`), the
dashboard's *What the canonicalizer did* panel, and `snapshot.Pair.Explain` — which is part of why
the schema bumped (to **6** on the line that introduced it, and **7** after the merge). `Diff` does not
diff it, for the reason `Reasons` was dropped in schema 2. The hook digests deliberately do **not**
carry it (they are bounded summaries), and neither do the census F-lines. Like culture, habitat,
profile and kind notes, it annotates: no score, no ranking key and no filter reads it, and the
golden scorecard is byte-identical with it in place.

## The report overview

The markdown report (`--output`) opens with a **What doppel sees** section: the concept vocabulary
and which concepts are *absent*, a package duplication map, per-package habitat norms, the arena
ecosystem split, the corpus metrics and the retrieval channel mix. Three mermaid diagrams carry it
(concepts, package duplication, habitats); the metrics and the channel mix are prose and tables,
and the fourth diagram in the document belongs to the families section.

The point is that all of this was already computed and then discarded. `culture.Stats`,
`retriever.Stats` and `family.Stats` went to stderr and died there; `Model.HabitatNorm`,
`HabitatTemperature` and `ConventionStrength` had no non-test caller at all outside the per-pair
notes. `internal/bench/examples_test.go` had been pasting the stderr block into every committed
example under `## Run diagnostics` to compensate, which is the clearest evidence the information
belonged in the document.

Rules that hold it together:

- **Markdown only.** `Meta.Overview` is nil for the text report and for `--format json`, and a nil
  overview renders **nothing** — a report without one is byte-identical to one written before the
  section existed. Mermaid is meaningless in a terminal, and the JSON payload is a snapshot with a
  documented shape.
- **`reporter` never learns about `culture`.** `cmd/overview.go` queries the model and fills a
  struct of plain presorted rows. `Overview` carries no maps that decide an order.
- **What crosses over.** A fact belongs in the report if it changes how a reader weighs the
  findings, and on stderr if it only helps someone tuning doppel. The channel mix crosses;
  `Suppressed`, `LargeBuckets`, `SurvivingLabels` and parse warnings do not. Both surfaces keep
  their lines — stderr is unchanged, because the hook and the examples wrapper read it.
- **`retriever.Stats` now rides on `Result`.** It was created, printed and dropped; the report
  explains its own pair list with it.
- **Escaping is not `mdEscape`.** That helper turns `|` into `\|`, and `|` is mermaid's edge-label
  delimiter. `mermaidLabel` emits HTML entities (`#quot;`, `#35;`) because a quoted mermaid label
  has no escape character, and `mermaidID` is positional rather than a mangled name — `a.b` and
  `a_b` would otherwise collide. Escape the identifier, *then* compose with `<br/>`: escaping a
  finished label renders the line break as a visible `#lt;br/#gt;`.
- **Diagrams departing from the wiki's unstyled house style is deliberate**, and only on `classDef`.
  A hand-authored diagram explains a mechanism; these encode a measured value, and colour is the
  only channel mermaid offers for one.
- **Every diagram is bounded and says so.** Package diagrams cap at `maxOverviewNodes` (12 — moby
  has 168 habitats); family diagrams cap at 8 members, because the picture must draw every edge to
  show the clique property and 55 members is 1485 edges.

A **Corpus metrics** subsection carries two further numbers, in the markdown preamble, in the
dashboard's fact tiles, and in `--format json` as `corpusMetrics` (not in `snapshot.Schema`'s
Params or Pair/Unit types, so it never affects comparability). **The plain-text report does not
carry them, and that is the Overview rule rather than an oversight**: both numbers travel to
`reporter` inside `Meta.Overview`, which is nil for the text report by construction, so a terminal
report is byte-identical to one written before the section existed. Containment and `explain:` are
per-pair and reach all four surfaces — text, markdown, JSON and the dashboard — because they ride
on `SimilarPair`, not on the Overview. The two numbers are: **compression ratio**, total canonical AST nodes divided by the
count of distinct subtree shapes among them once `fingerprint.ConsCorpus` hash-conses every
canonical body (`internal/fingerprint/cons.go`, mirroring `wl.go`'s FNV idiom but preserving child
order — a hash-cons answers "is this literally the same subtree", not "the same shape", so `a - b`
and `b - a` must not collapse the way WL's sorted-children recurrence lets them); and the
**nearest-neighbour code-shape distribution**, each function's best code-shape score among the
pairs retrieval already scored (the union, before any `--struct-min` filter — every one of those
pairs already carries an exact `fingerprint.Breakdown`), reported as p50/p90/p99 (nearest-rank, no
interpolation) and the fraction at or above the run's own threshold. Both are computed once in
`cmd/pipeline.go` (`Result.ConsStats`, `Result.NN`) alongside the existing `WL` and `TagCounts`
corpus statistics, for the same reason — a corpus statistic belongs where the population is
settled — and neither feeds any score. The nearest-neighbour figure is deliberately **not** an
exhaustive search: that is O(n²) fingerprint comparisons, the same cost `--min-nodes` and the
retrieval channels already exist to avoid, so a function retrieval never paired with anyone is
excluded from the percentiles and counted separately rather than assumed to have no similar code
at all — every rendering of the number restates that caveat rather than letting the figure imply
more than it measured.

A second section, **Local practice**, describes how the corpus *writes* things rather than what it
contains, from the two models that had no caller outside their own tests:

- **How each concept is written here** — the prototype as a table of counts with a proportion bar,
  never percentages: a concept qualifies at five members, and "83%" of six is more digits than
  evidence where "5 of 6" is what was counted.

  **Prevalence alone is not house style, and filtering on it produced a section that said nothing.**
  Nearly every Go function has a `return` and an `if`, so an unfiltered prototype reported "533 of
  533 error_wrapping functions return" — a fact about the language. A feature now earns its row by
  **lift over the corpus base rate**: carried by the concept's members at least twice as often as
  by the corpus at large, the same `ln 2` the ecology uses to decide an association is beyond
  chance. `culture.BaseRate(channel, feature)` supplies the denominator, counted in `Build` from
  the same `unitFeatures` the prototypes use, so the two cannot drift. The presence floor dropped
  to 0.25 in exchange: a feature in a third of a concept's members that is nearly absent elsewhere
  says more than one in two thirds that is everywhere. Rows sort by lift, not prevalence.

  On moby this is the difference between `return` and `github.com/pkg/errors.Wrap` at 14× — the
  second is a real fact about how that codebase wraps errors. A concept with **no** distinctive
  feature says so explicitly rather than rendering an empty table: the tag groups its members, but
  no shared way of writing them exists, and that is a finding about the tag.
- **Which concepts share a function** — the tag~tag grid, and the one table in the report that is
  **not** a sample: a fixed concept vocabulary means it is bounded by construction, so it shows every
  cell including the ordinary ones. `never` cells are the layering signal. An all-blank grid
  renders nothing, which is what doppel's own corpus produces.
- **What travels with what** — the PMI ecology, both directions, **grouped by kind**. Grouping is
  not cosmetic: there are far more call tokens than concepts, so on one shared list the tag~call
  rows crowd out every tag~tag row — this report showed zero concept-to-concept associations until
  each kind got its own budget. Each line leads with the conditional rather than the lift, because
  "13 of 15 `http_call` functions also call `NewRequest`" is what a reader acts on where "416×
  chance" only says why it is worth printing. For a tag~tag pair the *smaller* population is the
  denominator: "16 of 33 retry" beats "16 of 436 concurrency" for the same fact. Count 0 has no
  finite ratio and renders as the word, honouring `ecology.go`'s own contract.
- **Ranking within a kind is lift weighted by evidence** (`ln(lift) · ln(1+count)`), not lift
  alone, which put a 126× finding on three functions above a 31× one on six. Presentation only —
  `culture`'s own ordering contract is untouched — and stated in the section, because a list whose
  displayed lifts are not monotonic otherwise reads as a bug.
- **Functions drifting from their own concept** — named, not counted, which closes a gap the tool
  carried: a drifting function in a reported pair got a culture note, and one in *no* pair was a
  stderr tally and nothing else. Those are the more interesting ones, so they sort first and carry
  a marker. The marker column is emitted only when something is marked.

`practiceWeight` in `cmd/overview.go` duplicates the five prototype channel weights, which live
unexported in `culture`. Four integers were cheaper than widening that package's API — but the
table has to track it.

The derived `RoleThresholds`, computed in `mapper` and discarded, stay unsurfaced.

## The dashboard

`--output report.html` writes the **dashboard**: one self-contained page, no fetch, opening from
`file://`. Any other extension still writes markdown. The format is chosen by extension rather than
by `--format`, because `--format` selects what goes to *stdout* and a page of markup there helps
nobody.

**The split with Go is the whole design.** It replaced a renderer whose page lived in a 276-line
`html/template` literal fed by a flat struct of render-ready percentages and label strings, so
every visual decision was Go's and adding one meant editing a template, a struct and an assembler
across two packages. Now `cmd/dashboard.go` emits a **semantic payload** — raw scores, counts and
identifiers, no percentages, no labels — and `internal/dashboard/assets/app.js` decides what a
colour, a radius or a bar means. Iterating on the visuals is an asset edit.

`internal/dashboard` owns the payload types, the renderer and the assets; `cmd/dashboard.go` builds
the payload. That is the same bridge `cmd/overview.go` makes for the markdown report — `dashboard`
never learns about `culture`, and `cmd` queries the model.

- **Two screens, and the two that are missing are missing for a reason.** *Map* is a political map
  of the corpus: every package (or concept — it is a toggle) is a polygon whose area is its share of
  the functions, polygons tile the canvas with shared borders, and each region is tinted by the
  concept most of its functions carry. Pair evidence is carried by the **borders** rather than by
  a line layer. Individual functions are deliberately **not** drawn: a per-function dot inside a
  region encodes nothing the region does not already say — its position is an arbitrary spiral —
  so it was clutter with a redundant click target, and the border/arc pair lists and the
  neighbourhood picker remain the ways into a single function. *Neighbourhood* takes
  one function and shows its ranked neighbours, both bodies side by side, and the pair's evidence.
  A **delta** screen and a **concept-drift** screen were scoped and dropped: both need a snapshot
  *series*, and there is exactly one baseline per session in tmpdir, no timestamp inside a
  `Snapshot`, and nothing from `culture` persisted at all. Neither is a small addition.
- **The page draws `res.Pairs`, not the ranked report list.** Same reasoning as the family stage:
  `--top` and `--max-per-func` are report-time devices, and a neighbourhood built on a
  diversity-capped list would hide the neighbour a reader clicked in to find. So those two flags do
  not bound this page — `--threshold` and `--struct-min` do — and the page's closing note states
  the difference rather than implying there is none.
- **`Edge.Rank` is computed in Go on purpose.** `analyzer.RankKey` is this repo's single definition
  of corroborated evidence; a second one in JavaScript would drift from it silently. It is the one
  quantity the payload carries that the page could have derived.
- **Determinism holds, and CI checks it.** `TestPayloadHasNoMaps` mirrors
  `snapshot.TestSchemaHasNoMaps` by reflection, every slice is sorted before it is emitted, and the
  ubuntu leg diffs two `-o *.html` runs beside the existing `--format json` one.
- **Escaping is `encoding/json` with `SetEscapeHTML` left ON** — the opposite of
  `reporter.encodeJSON`, which turns it off for snapshot byte-comparability. The payload rides
  inside a `<script>` element, so a `</script>` in an analysed function body must come out as
  `\u003c/script\u003e` or it ends the page. `TestPrintEscapesScriptClose` pins it, and fails
  loudly when the flag is flipped.
- **One vendored asset, not hand-edited** (`internal/dashboard/assets/vendor/`, with a README
  recording source and licence): `broadsheet.css`, the same design-system subset as before, moved.
  There is deliberately **no vendored JavaScript**. Cytoscape.js was vendored first and dropped when
  the map became a power diagram — that is plain SVG, and the neighbourhood screen never used a
  library at all, so it was ~370 KB in every report written for a screen that no longer needed it.
  `TestPrintIsSelfContained` pins its absence. The bar for bringing one back is that it does
  something the page cannot do in a few hundred lines of its own.
- **The map is a cartogram: a power diagram fitted to area shares.** Every package is a convex
  polygon, the polygons tile the canvas, and each one's **area is its share of the function count**.
  Cells fall out of successive half-plane clips (`clipHalfPlane`, `powerDiagram`), because a
  weighted-Voronoi bisector is still a straight line — only shifted by the weight difference — so
  weights cost nothing but a constant in the clip. `fitAreas` then alternates Lloyd relaxation with
  a weight step, both expressed in **radii** and both clamped against the distance to the nearest
  site. The radius formulation is load-bearing: written additively on raw area, with the clamp as
  `wᵢ − wⱼ ≤ |pᵢ − pⱼ|²`, it does not converge — that constraint binds hardest exactly where two
  sites are close, so a large region wedged beside a small one cannot grow. Measured on this repo,
  that version left `parser` at 0.02% of the canvas against a 4.9% target; the shipped one holds
  every region of four functions or more inside 0.1%, weighted mean error 0.2%. Residual error is
  concentrated in regions too small to draw honestly anyway.
  `cose` was tried before either and is wrong twice over: compound parents pin every function inside
  its own package box, so a force layout can only jiggle within a territory rather than pull related
  functions across one; and on a corpus with many disconnected components it detonates (measured at
  y ≈ 1.3e6 on this repo, off-screen and unfittable).
- **A shared border is geometry; the paint on it is the evidence.** `clipHalfPlane` carries a tag
  per polygon edge naming the site that produced it, so a finished cell knows which neighbour every
  border faces — recovering that afterwards by comparing coordinates would be slower and fuzzier
  than recording it at the moment it is created. Sites are seeded by a spring embedding of the
  region coupling graph, so related regions *tend* to adjoin, and where two related regions do meet
  the border between them is painted by the duplication crossing it. **Adjacency is a tendency, not
  a claim**: a planar map cannot realise every adjacency a corpus asks for, so related regions the
  packing could not seat together are drawn as dashed arcs between centroids instead. Without those
  the map would be quietly lossy, which is the failure mode a political map invites. The page says
  all of this in its own words, next to the map.
- **The partition is a control, not a rebuild.** `powerDiagram` does not care what a site is, so the
  same renderer draws territories by package or by concept. Concept territories force each function
  into one region where membership is genuinely multi-valued and graded — a real simplification, and
  the reason package is the default.
- **The colour channel is a unit's strongest membership, not its arena equilibrium.** The arena was
  used here first, when concepts were fourteen asserted tags and its job was suppressing the
  tagger's false positives. A learned lexicon does that job at membership time, with a confidence.
  What the arena still adds is *invasion* — a concept can win a function through an association
  without the function carrying it — which is a real finding and a bad colour: measured on this repo
  it painted 111 functions with a concept only 5 of them carry, and a legend cannot honestly say
  "leads 111, carried by 5". Taking the head of the ranked memberships keeps `Dominant` a subset of
  `Carried` by construction, which is the invariant the legend rests on.
- **The palette does not cycle.** The vocabulary is learned, so its size is a property of the corpus
  — 71 concepts on this repo against 13 palette entries. Cycling would give two unrelated concepts
  the same hue and say nothing about it, so the concepts that colour the most functions take the
  palette and the rest pool into one neutral that the legend counts out loud.
- **Seeding and fitting are deterministic by construction.** Fixed iteration counts, fixed order,
  sites started on a ring in size order — no seeded RNG, because there is no RNG. The same payload
  always draws the same map, which is what lets the page itself stay byte-identical.
- **Bodies are bounded and the bound is reported.** `parser.CodeUnit.Body` already holds full source
  text, but it is the only part of the payload that scales with corpus size rather than with the
  number of findings. Bodies are admitted in descending edge rank until `maxBodyBytes` (3 MB) is
  spent, so what a reader is most likely to open survives the bound; the count dropped goes to
  stderr and onto the page, and a function without one shows its `file:line` instead.
- **The residual-difference view is a text diff, and says so.** Screen 3 was specified to highlight
  the shared structural patterns inside both bodies. There is no data for that: a shared chain is a
  Weisfeiler-Lehman label, which is a hash of a whole subtree, so `SharedChain.Render` can *name*
  it (`depth-2 IF`, via `fingerprint.DescribeLabel`) but nothing maps it back to the tokens that
  produced it. The page lists the chains beside the bodies (the actual evidence) and runs a
  line-level LCS diff over the two sources, captioned as a textual comparison rather than the
  structural claim the score makes. Highlighting would mean carrying source spans through the
  fingerprint's hot path. What the page *can* say about the residual it now says in words, in the
  **What the canonicalizer did** panel: `SimilarPair.Explain` is the pipeline's own sentence about
  what had to be rewritten before the two bodies matched.
- **What the merge added to the page.** Three quantities from the WL line reach it, all additive
  and none of them scored: `Edge.Containment` (in the score row and as the sixth breakdown bar),
  `Edge.Explain` (its own panel), and the two corpus metrics — compression ratio and the
  nearest-neighbour median — as fact tiles. `Facts.Compression`/`NNScored`/`NNP50`/`NNP90` come
  from `Result` directly rather than through `reporter.Overview`, because a run can be built with
  no overview at all and the numbers exist either way. The breakdown's first component was renamed
  `ast` → `wl`, which is a rename of the label to match what the number has become: a
  corpus-weighted multiset Jaccard over WL bags, not a Jaccard over token 3-grams.
- **The dashboard is not fed by `--format json`.** It has its own `dashboard.Payload` (`Schema` 1,
  independent of `snapshot.Schema`), marshalled by Go and inlined into the page at render time.
  That is worth knowing before assuming the snapshot and the page must agree: they share no type,
  and a field added to one reaches the other only if somebody adds it there too. It also means the
  snapshot's schema-7 additions could not have broken the page even by accident.
- **`DOPPEL_DASHBOARD_ASSETS=<dir>`** reads the assets off disk instead of the embedded copy, so
  editing `app.js` and re-running a prebuilt binary is the whole cycle (`task dashboard-dev`).
  `TestDevAssetsMatchEmbedded` pins the two paths to the same bytes. Nothing ships depending on it.

Gone with the broadsheet report: the **strip view** (`cmd/strips.go`'s declaration-span
silhouettes, the one piece of genuinely new data that page invented) and the taxonomy, habitat and
census panels, which remain in the markdown report. The strips would port cleanly to a dashboard
panel if they turn out to be missed.

## Frontends

A frontend's whole job is to produce `internal/syntax.File`. Everything downstream — the
fingerprint, the Weisfeiler-Lehman label bag, the hash-cons, `TagSignals`, the call graph, the
lexicon, culture, habitats, calibration, the report — reads that and knows nothing about any
language.

**The IR's contract is narrow but not loose.** A `syntax.Node` must exist for every node the
frontend's own traversal would visit, in that traversal's order. Node identity and order are
observable: `Fingerprint.Nodes` counts them (feeding `--min-nodes` and `SizeRatio`), the token
stream is emitted in visit order, and the nesting-depth histogram is driven by the push/pop
pairing that `syntax.Inspect`'s nil-after-children callback provides. A frontend that collapses
nodes changes scores rather than losing detail quietly. `Role` exists because position cannot
carry slot identity — a for-loop with no init has its condition first — so slots are named.

**The kind vocabulary is wider than the token stream needs, and that is the label bag's doing.**
`walk` emits a token for two dozen kinds and silently ignores the rest; `WLBag` reads *every*
node's kind, because a node whose kind collapses to `KindOther` does not vanish from its
parent's child multiset — it merges with every other unnamed node there. A collapsed vocabulary
therefore coarsens the bag rather than shrinking it, which is why `syntax.Kind` names array,
map, struct, func, interface and channel types, fields and field lists, and general
declarations, none of which the token stream has ever emitted. A frontend that can tell an array
type from a map type should say so; one that cannot leaves them `KindOther` and loses
discrimination rather than correctness.

- **`internal/gofront`** maps `*ast.FuncDecl` onto the IR and is the only package in the module
  importing `go/*` outside `internal/canon`. It builds the tree **from `ast.Inspect`** rather
  than reimplementing `ast.Walk`'s per-type field ordering, so order and node count are correct
  by construction and there is no ordering table to fall out of sync with the stdlib; roles are
  recovered separately by identity against the parent's named fields. `TestMapperPreservesNodeCount`
  and `TestMapperPreservesOrder` pin both halves. It also **runs `internal/canon`** and maps the
  canonical declaration a second time into `syntax.Func.Canon` — see *Canonicalization is a
  frontend's job* below.
- **`internal/lexfront`** has no grammar: one tokenizer, one block rule, and a per-language table
  of the things that genuinely cannot be guessed — extensions, comment and string delimiters,
  which keywords introduce a function or a container, whether parameters are name-first, and how
  tests are named. Adding a language is a table entry.

**The trade is measured, not asserted.** `TestLexicalFidelity` (guard `DOPPEL_BENCH_LEXICAL=1`,
`DOPPEL_BENCH_CORPUS` to aim it) runs both frontends over the same **Go** corpus and scores the
lexical one against `go/ast` — the one language where the right answer is already known, which
is why it is the honest control for a frontend meant for languages where it is not. Measured on
the Go standard library:

| corpus | functions | recall | precision | node count | param arity |
| --- | --- | --- | --- | --- | --- |
| runtime | 7 101 | 0.995 | 1.000 | 0.98× | 100% |
| net/http | 2 904 | 0.996 | 1.000 | 0.94× | 100% |
| crypto | 3 934 | 0.999 | 1.000 | 0.96× | 100% |
| encoding/json | 1 235 | 0.999 | 1.000 | 0.94× | 100% |
| go/types | 1 055 | 0.994 | 1.000 | 0.98× | 100% |

Bodyless declarations (assembly-implemented, external linkname) are excluded from the
denominator: their `Fingerprint` is the zero value and they never match anything, so counting
them would measure a difference that changes no result.

**What the AST still buys is two things: types, and a canonicalizer.** `lexfront` fills
`syntax.Param.Type` with the empty string, so `Fingerprint.Types` is empty and the signature
component — 0.15 of the composite — contributes nothing, with `sig: (?)` in the report saying
so. If that 0.15 turns out to dominate, the fix is the existing `fingerprint.Weights` seam
(`SimilarityWith`), which is already a no-op at defaults. The second is the subject of the next
two sections.

### Canonicalization is a frontend's job

`internal/canon` stays go/ast-typed and Go-only. That is not a leftover: every rule it applies is
a claim about what Go code *means* — that `if !c {A} else {B}` and `if c {B} else {A}` are the
same shape, that `x = x + 1` and `x++` are, that operands of `&&` may be reordered. The IR
deliberately carries no semantics to make such a claim with, and a rule table shared across
thirteen languages would be asserting Go's semantics about all of them.

So `gofront` runs `canon.Canonicalize` on the `*ast.FuncDecl`, maps the **canonical** tree into
`syntax.Func.Canon`, and records the fired rule IDs in `syntax.Func.CanonRules`. `Func.Shape()`
returns `Canon` when there is one and `Body` otherwise, and that is the tree `WLBag` and `Cons`
walk. `parser.CodeUnit.Canonical` is `Shape()` projected; `CanonRules` is the plain string list,
because `syntax` cannot be typed on one frontend's rule enum. `analyzer.Explain` still imports
`canon` to map an ID to a reader-facing word, and renders an unknown ID under its own name.

**The move was arranged to change no Go score, and that was verified rather than argued.**
`label_0` is a hash of a kind's *name* plus the one token that is part of it, so as long as every
go/ast kind reaches the same `LabelKind` it did when the switch was typed on `ast.Node`, every
label at every round is the same 64 bits. That is what the widened `syntax.Kind` vocabulary and
the `IncDecStmt`/`GenDecl`/`ChanType` labels are for. Measured end to end: over all seven pinned
corpora (~19 400 functions), `doppel analyze . --format json --languages go` before and after
differs in exactly three top-level keys — `doppel`, `schema` and `params.languages`. Every unit,
every WL bag, every pair, every score and every explanation sentence is byte-identical.

### What a language without a canonicalizer gets

Every language gets a WL bag, and with it the code-shape score, the shape retrieval channel,
containment, the hash-cons compression ratio, and `analyzer.Explain`'s residual sentence. None
of that is Go-only any more. What a `lexfront` language does *not* get is the canonical tree the
bag is built over, so it is worth being exact about what its bag can and cannot claim.

**It can claim**: that two bodies share structure, weighted by how surprising that structure is
in this corpus. The recurrence is the same, the corpus weighting is the same, the bag is built
over whatever tree the frontend produced, and `LabelWeights` counts df over that same
population. Two Python functions with the same nesting and the same call names share labels, and
`Containment` still says whether one body's shape reappears inside a larger one.

**It cannot claim** the two things canonicalization buys. First, **rename invariance is only
partial**: `label_0` collapses every identifier to `ID` regardless of frontend, so a renamed copy
already matches — but alpha-renaming is what makes two functions that *bind* their locals in a
different order agree, and without it they do not. Second, **spelling variants stay apart**:
early return versus else, `x += 1` versus `x++`, `a && b` versus `b && a` are different shapes to
a bag with no canonicalizer, where in Go they are one. Both failures are in the safe direction —
less similar than they are, never more — which is the same direction `LabelIDF.Weight`'s
unseen-label fallback fails in, and for the same reason.

A shallow tree adds a third, independent limit that has nothing to do with canon: `lexfront`
builds a block tree, not a parse tree, so most interior nodes are `KindOther` and the deeper
refinement rounds fold in less. An h=3 label on a Go body summarises a whole guard; on a
lexically-segmented body it summarises a nesting region. The bag is real evidence at both ends,
but it is coarser at the second, and `CanonRules` being empty is the visible sign of it —
`analyzer.Explain`'s "identical after rename" tier keys on bag equality, which still works, and
simply reports "identical structure, no normalization needed" where a Go pair would have named
the rules that got there.

**Five defects the harness caught that review had not**, kept as tests in `internal/lexfront`
because each was silent: statement keywords matching the declaration shape (`except ValueError:`
was 46% of everything found on the Python standard library); a class with a parenthesised base
swallowing every method inside it; the indent rule measured from the method's name rather than
its `def`, ending each body on its own first statement; an unbounded body scan walking forward
from `print("x")` to an unrelated `if` two lines later; and anonymous literals emitted as units
called `func` (11% of this repo). A lexical frontend fails quietly by construction — there is no
parse to fail — so the measurement *is* the correctness argument.

**Cross-language pairs never exist.** `parser.SameBuildUnit` is one predicate over two rules:
test and production code are different build units, and so are two languages one step further
out. It replaced the `_test.go` suffix check that had been copy-pasted into `cmd/analyze.go`,
`internal/bench`, `internal/calibrate` and `internal/analyzer/rank.go` — a clone doppel would
have flagged on itself, and four places to miss when a second language arrived.

**Scope is an extension allowlist, never a content heuristic.** A file is in the corpus because
a frontend claims its extension and `--languages` admits that language. Prose, markdown, config
and data are out by construction; nothing inspects a file to judge it code-like, which is the
same refusal the tagger and retriever make everywhere else. `--languages` defaults to every
registered frontend, so a Go repository with a vendored `.js` asset now analyses that asset too
— which moves the calibrated thresholds, because the corpus genuinely changed. `--languages go`
or the `languages` config key restores the old population exactly.

## Configuration

`.doppel.json` at repo root, or `--config <path>`. A missing file is not an error; malformed JSON is.
**Keys are kebab-case** (they mirror the flag names), not snake_case:

```json
{
  "threshold": 0.38,
  "top": 10,
  "min-nodes": 16,
  "struct-min": 0.4,
  "output": "doppel-report.md",
  "channel-k": 5,
  "debug": false,
  "max-per-func": 2,
  "tests": "exclude",
  "generated": "exclude",
  "languages": ["go"],
  "calibrate": 0.01,
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
must reach — both presentation, so neither is in `Params` and neither can invalidate a baseline; `--languages` picks which
frontends are read (defaulting to all of them) and is corpus-defining, so it travels in `Params`
and in `snapshot.Params` and a run reading Go alone is correctly incomparable to one reading Go
and TypeScript; `--tests` and `--generated`
pick the population (`include`/`exclude`/`only` each, both defaulting `exclude`) before any
statistic is computed — tests because they are conventionally similar by design, generated files
(Go's "Code generated ... DO NOT EDIT." marker, detected via `ast.IsGenerated` at parse time)
because they are near-identical by construction and unactionable. `--calibrate <rate>` (config
key `calibrate`, **default 0.01**) derives `--threshold` and `--struct-min` from the corpus's own
null distribution and sets the family edge cut and fork floor to the same code-shape value — see
*Calibration*; it overrides both flags outright, and an explicitly pinned threshold turns it off
for the whole run instead. `--min-nodes`, `--channel-k` and `--max-per-func` are `MarkHidden` on
every command that registers them: they are retrieval and report budgets, not judgments about a
corpus, and no question about a codebase answers what to set them to. Hidden, never removed —
each still parses and still has a config key, so `internal/bench` and every script keep working;
only `--help` shrinks. The three similarity floors stay **visible** because they are the
documented escape hatch from calibration, and a hidden escape hatch is not one.

### Where the shipped floors come from

`--threshold` defaults to **0.38**: the median of the code-shape floors `--calibrate 0.01` derives
across the public ladder — prometheus 0.33, moby 0.35, hugo 0.35, gin 0.41, cobra 0.44, chi 0.45,
with conc declining for want of eligible null pairs (351, against the 1 000 the calibration
requires). The declined rung is **excluded from the median rather than counted at the old
default**: a declined calibration is missing data, and letting the incumbent value vote for its own
retention is not a measurement. Six values, so the median is the midpoint of 0.35 and 0.41.

The old `0.60` was not wrong so much as *differently strict on every corpus* — it admitted far
fewer than 1% of random unrelated pairs everywhere it was measured, which is exactly the
"`--threshold 0.60` is loose on 81 functions and strict on 8 000" problem *Calibration* exists to
name. `TestThresholdLadder` is the evidence that moving it is safe: on the cobra labels **every
metric is flat across the whole 0.30–0.60 range** (merge 4.5, refactor 13.7, fp 47.0, no
violations) while the candidate set grows 793 → 1 016. The threshold gates shape-channel admission
only, and the labeled pairs all reach the comparator through concept or call evidence regardless,
so the change buys recall at no measured labeled cost. Where it is visible is the small rungs:
conc's top ten used to bottom out on call-only pairs at code-shape 0.13–0.24 and now ends at
0.40–0.51.

**`--family-min` deliberately does *not* follow it, and stays `0.60`.** Under `--calibrate` it
does — `familyMinFor` moves the edge cut to the calibrated value — because that run has opted into
corpus-relative semantics for every "alike enough" in it. The static default is a different
question, because the two numbers are different *kinds* of number: `--threshold` is an admission
gate, where being generous costs compute and lets ranking sort it out, while `--family-min` is the
floor of a **guarantee the report prints** ("every pair >= 0.72 code-shape"). A cut at the 99th
percentile of random pairs would make that guarantee read "these functions are not quite random
with respect to each other", which is not a near-duplicate claim. Measured at 0.38 the census
degrades exactly as that argument predicts: moby goes 179 families / 636 functions to 670 / 1 554
with a **77-member** largest family and edge completion supplying 7 653 edges rather than 589, and
cobra's tightest family — the three `MarkFlags*` bodies at code-shape 1.00 — falls out of the
default top five, outranked by looser families whose evidence sums over more edges. `ForkShapeFloor`
stays pinned to it at 0.60 for the same reason.

`--struct-min` keeps its `0.0` default (off). Calibration derives it (0.30–0.44 across the ladder)
but it is a *selection* stage, and turning a filter on by default would drop pairs from every
report to no labeled benefit.

`hook-notify` (`agent` | `user` | `off`) is read only by `doppel hook stop` and has no flag — there
is no CLI surface a hook setting would belong to. `format` (`text` or `json`) is a key like any
other. Every functional flag except `--config` has a
config key. Precedence: `applyConfig` only calls
`Flags().Set` when `!Flags().Changed(name)`, so explicit CLI flags always win over the file.
Unknown keys are ignored rather than rejected, so a stale config file does not break a run.

## Calibration

`internal/calibrate` answers "what does a random, unrelated pair score *here*", so a threshold can
be stated as a rate instead of a number. `--threshold 0.60` is loose on a corpus of 81 functions
and strict on one of 8000; "admit 1% of random pairs" means the same thing on both. **This is the
default** (`defaultCalibrateRate` = 0.01, in `cmd/config.go`): the three similarity floors are
corpus-derived unless someone pins one, because a fixed floor is an operating point calibrated for
somebody else's repo and no end user has a basis for moving it.

Measured at rate 0.01 **on the merged tree**, where the code-shape null is drawn against the
corpus-weighted WL Jaccard rather than the old token-3-gram one: threshold **0.33 prometheus,
0.35 hugo, 0.35 moby, 0.41 gin, 0.44 cobra, 0.45 chi**, with struct-min from 0.29 to 0.51. The
whole ladder sits *lower* than the fixed 0.60 and lower than the same measurement on the
pre-merge tool (which read 0.45 moby / 0.53 cobra), which is what a WL score distribution looks
like: a WL bag weights every shared label by corpus surprisal, so two unrelated bodies agree on
far less of what is being counted than two unrelated token bags did, and the 99th percentile of
the null moves down with it. Nothing here says the operating point is *better*; it says the same
question now has a different answer, and pinning the old numbers by hand would be answering the
old one.

**`--min-nodes` and calibration are separate knobs, and the merge made that visible.** Calibration
derives a *score* floor from the null distribution; `--min-nodes` (16, raised from 12 for the WL
shape channel) is an *eligibility* rule about which functions the channel indexes at all. Neither
subsumes the other — a corpus of accessors has a null distribution made of accessors, so a rate
would happily admit them — and the shape null is drawn over exactly the `--min-nodes`-eligible
units, so raising the floor shrinks the null population. On **conc** (81 functions) that is
decisive: too few eligible shape null pairs remain against the 1 000 the calibration needs (351
measured at floor 18), so calibration is **declined** and conc runs at the static fallbacks
(threshold 0.38, struct-min 0.0). At `--min-nodes 12` the same corpus calibrates to 0.85. That is the guard doing its job — eight samples above a 1% cut is not
a calibration — and it is stated on stderr, not silent. Whether 16 is right for a corpus that
small is a measurement question, not a merge question; the seam is here for it.

Mechanics, all deterministic by construction: units are put in a canonical order (`package.name`,
file, line) so walk order cannot matter; the seed is FNV-1a over those names; a 64-bit LCG draws
up to 20 000 distinct unordered pairs (enumerated outright when the population is smaller),
rejecting cross test/production pairs like the pipeline does; pairs are scored in ascending index
order. The **code-shape null** is drawn over `--min-nodes`-eligible units — the shape channel's own
gate — and scored with `fingerprint.SimilarityWith`; the **overlap null** is drawn over all units
and scored with the run's own corpus-weighted comparator. Each threshold is the nearest-rank upper
quantile at `1 − rate` (a score some null pair actually had, never an interpolation), **rounded up
to 0.01** so the printed value is the used value and the admitted null fraction is at most the
rate. Below 1 000 eligible null pairs the calibration is **declined** and the defaults are kept —
eight samples above a 1% cut is not a calibration — and stderr says so.

When applied, `p.Threshold` and `p.StructMin` are replaced and the fork floor
(`analyzer.ClassifyPairWith`) and family edge cut (`familyMinFor`) follow the calibrated
code-shape value, so every "alike enough" in the run is one number. The effective values travel in
`Params` and therefore in `snapshot.Params` (which also records the rate, `calibrate`): a snapshot
compares on what was actually used, and a calibrated run against an uncalibrated baseline is
incomparable, correctly. Calibration **replaces both thresholds unconditionally** — `applyConfig`
sets flags through `Flags().Set`, which marks them Changed, so "explicit flag wins" is not
honestly implementable once a config has applied, and a half-calibrated run is the mixed question
Params equality exists to forbid. `doppel query` does not calibrate (it stops at `index()`), so it
alone still runs at the fixed 0.60.

**Opting out is all-or-nothing, and by design.** `calibrationOptOut` (`cmd/config.go`, called from
both `PreRunE`s after `applyConfig`) turns calibration off for the whole run when `--threshold`,
`--struct-min` or `--family-min` is `Changed` and `--calibrate` is not. Without it a default-on
calibration would accept an explicit `--threshold` and then silently discard it. Reading `Changed`
rather than the values is what makes it run *after* `applyConfig`, and that ordering is
load-bearing in the useful direction: config keys are applied through `Flags().Set`, which marks
them Changed, so **an existing `.doppel.json` pinning `threshold` or `struct-min` keeps its
behaviour byte-for-byte** and only a config naming `calibrate` opts back in. Verified: `analyze`
under `--calibrate 0`, under an explicit `--threshold 0.60`, and under a config pinning
`threshold` each reproduce the pre-change `--format json` byte-for-byte on a fixed tree.

The stderr calibration line is now part of **default** output — it was previously suppressed to
keep the default byte-identical, and that reason is gone. Printing it is the point: the operating
point is the first thing a reader needs when it is no longer a constant.

On cobra's golden labels calibration is neutral at every rate from 0.005 to 0.05: the retrieved
set grows from 816 to 1 029 candidates and the overlap gate keeps between 83 and 384 of them,
without a single labeled pair changing rank (`TestCalibrate`, guard `DOPPEL_BENCH_CALIBRATE=1`).
That is the evidence that calibration changes *what is admitted and shown*, not the ordering —
which is what made it safe to default on.

**Where it pays is the corpora the fixed floor got wrong, and doppel is not one of them.** On this
repo the derived floor lands just under the fixed 0.60 (0.48 at the time of writing; it moves as
the corpus does, which is the point) and the top 20 comes out **identical** to the fixed-threshold
run, pair for pair — 0.60 was already about right here, so calibration correctly changes nothing.
Re-check it with `doppel analyze . --format json` against `--calibrate 0` rather than trusting the
number above, which is exactly as stable as this repo is. conc is the opposite end and the real
argument: at
a fixed 0.60 its report is ten pairs of which seven are one-line builder methods,
`WithMaxGoroutines` alone four times over. Calibration measures that looking alike is *normal* in
that corpus, puts the floor at 0.85, and what surfaces instead is the `Go` methods across pool
types, `addErr` beside `resultAggregator.add`, and the `panics.Catcher` trio. Same rate, opposite
effect, because the two corpora genuinely differ — which is the whole claim. The gin/chi labels
are still what would turn a direction into a verdict.

**The `--struct-min` change is the largest behavioural one here.** Its flag default is `0.0`,
meaning no overlap filter ran at all by default; calibration gives it a real value (0.29–0.51
measured across the pinned ladder), so pairs are now dropped that previously reached the
report. That is the intended behaviour — 0.4 is what the config example and the merge gate always
suggested — but it is the thing to look at first in a regenerated example.

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
  *corpus* (`threshold`, `min-nodes`, `channel-k`, `tests`, `generated`) and overrides the ones that only decide
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

  **5 and 6 were assigned twice.** Two development lines diverged for a while and neither knew
  about the other, so a file claiming `"schema": 5` could have been written by either. On the
  *shape* line 5 changed what `Score` means and 6 moved shape retrieval to WL labels; on the
  *concept* line 5 replaced the tag list with graded concept memberships. Both narratives are kept
  below, and both are kept in the `Schema` const's own doc comment, because a reader holding an old
  baseline needs to know the number is ambiguous. **`Schema` 7 is the reconciliation** — the first
  version carrying both worlds, and the first whose number means exactly one thing. Nothing was
  dropped from either line: 7 has `RuleSet`, `Labels`, `Unit.WL`, `Pair.Containment`,
  `Pair.Explain` and `CorpusMetrics` from one, and graded `Unit.Concepts` plus `UnusedSeeds` from
  the other. The bump is not cosmetic even for someone who has both halves: a baseline written by
  either predecessor is missing half of what 7 records, and the missing halves are exactly the
  corpus-relative ones a diff must never guess at.

  **And then 6 was assigned a third time**, on the *language* line, where it added
  `Params.Languages` — which languages a run read decides what the corpus *is*, so it sits with
  `TestsMode` and `Generated` among the parameters that make two runs comparable, and a schema-5
  baseline records no language at all. `Params` gained an explicit `Equal` in the same change: a
  slice field stopped `==` from compiling, and joining the list into one string to keep the
  operator would have hidden a real fact behind a convenience. **`Schema` 8 is the second
  reconciliation**, and unlike 7 it is a numbering bump rather than a metric one. Nothing a
  snapshot stores changed meaning: moving the WL bag onto the neutral IR was arranged to leave the
  label vocabulary exactly where it was — same `label_0` names, same recurrence, same canonical
  tree — and verified byte-identical over all seven pinned corpora, so a schema-7 `Unit.WL` and a
  schema-8 one over the same Go body are the same bag. 8 exists because `Params` gained
  `Languages`, which a schema-7 baseline cannot supply and this run must not invent.

  `Schema` 5 (shape line) was the same kind of bump as 3, one step further: `Pair.Score` changed metric (token shingles → corpus-weighted WL
  Jaccard) *and* became corpus-relative, so a schema-4 baseline and a schema-5 run would disagree
  about pairs nobody edited. It also added `Containment`, which earns its bytes the way rule four
  requires: containment is reported on every pair in all four output formats now, and
  `--format json` is the machine-readable form of that same report, so the field has a reader
  rather than being write-only. `Schema` 6 (shape line) is four
  changes shipped as one bump. The shape retrieval channel indexes WL labels instead of the pattern
  multiset, so a schema-5 baseline and a schema-6 run hold *different candidate sets* — pairs appear
  and vanish with neither body touched, which is exactly the movement `Attributable` exists to avoid
  claiming, and unlike a top-K budget shift it is systematic rather than incidental, so the honest
  result is incomparability rather than a delta. The same bump adds
  `Unit.WL` — every unit's Weisfeiler-Lehman label bag — and `Snapshot.RuleSet` (`canon.Version`,
  recorded once per run rather than per unit, since every unit in a run shares one canonicalizer).
  The bag's encoding is not a plain per-unit delta-varint of raw labels: a WL label is an FNV hash,
  essentially a uniform random 64-bit value, so the gap between two labels in one function's own
  small bag is itself close to the full 64-bit range — sorting cannot cluster values that were
  never clustered — and measured naively this put moby's `--format json` at just over 5x its
  schema-5 size. `internal/fingerprint/wlcodec.go` instead builds one corpus-wide dictionary of
  every distinct label the run's bags carry (`Snapshot.Labels`, via `EncodeLabelDict` — 131,350
  entries behind 648,305 total bag occurrences on moby, a ~4.9x reduction before a byte is counted,
  because the same structural labels recur across many functions) and encodes each `Unit.WL`
  as delta-varint over that dictionary's *positions*, not the raw hashes (`EncodeWLBagIndexed`) —
  positions range over the dictionary's size rather than 2^64, so they compress far more tightly.
  base64.RawStdEncoding throughout — picked once, documented, fixed forever. Measured end to end:
  moby's `--format json` is 5,877,513 bytes against a schema-5 baseline of 2,077,641 — 2.83x,
  inside the `<3x` budget; the baseline-wrapper file (pretty-printed) is 1.60x. Both fields earn
  their bytes the way rule four requires: the bag is what lets a consumer of `--format json` or a
  session baseline recompute the WL Jaccard and Containment components of a stored pair from
  stored data alone, rather than only reading the two floats the pipeline already produced —
  `fingerprint.WLOverlap` (a thin export of the same `wlOverlap` the pipeline calls) plus a
  `LabelIDF` rebuilt with `fingerprint.LabelWeights` over every decoded bag in the snapshot's own
  `Units` is that computation, and the round trip is exact because that population is exactly what
  the live run's `res.WL` was built from. RuleSet is what makes a canon rule-set change an
  incomparability result instead of silent drift: adding or changing a canonicalization rule moves
  every WL label a body produces, which is the schema-4-vs-5 failure one layer lower — the same two untouched functions
  scoring a different WL Jaccard and Containment under two rule sets — so `Diff` refuses across it
  exactly like Schema, Doppel and Ontology. Score itself still cannot be recomputed from a
  snapshot: Flow, Depth and Signature are not stored (no reader needs them back), so the bag only
  ever reconstructs the WL component and Containment, never the four-term composite.

  `Schema` 5 (concept line) replaced `Unit.Patterns []string` with graded `Unit.Concepts`
  (confidence rounded to two decimals — the Stop hook rewrites this file every turn and full float
  precision is bytes of noise) and added `UnusedSeeds`. A schema-4 baseline's tags are names from a
  vocabulary that no longer exists: they would not fail to compare, they would silently match
  nothing. Both fields survive unchanged into 7.

  **`Diff`'s incomparability reasons are the union of both lines' refusal conditions**, which is
  the only safe resolution: each line refused for something the other could not see. A diff is
  declined when `Schema`, `Doppel`, `Ontology`, `RuleSet` or `Params` differ. `RuleSet` is the
  shape line's contribution and it is not implied by any of the others — a canonicalization rule
  can change with the doppel build string unstamped (see the `(devel)` trap below), and it moves
  every WL label two untouched bodies produce.

**What a delta may and may not claim.** `UnitsAdded`, `UnitsRemoved` and `BodiesChanged` are solid:
they come from names and from `Digest`, an FNV-1a hash of the unit's own fingerprint, so nothing
outside a function can move them. Pair changes are one tier down and each carries an `Attributable`
bit, because retrieval keeps a bounded top-K per function and a pair can enter or leave the candidate
set without either side being touched. Everything else corpus-relative — role changes, caller/callee
counts, overlap movement, tag totals — is deliberately **absent from `Delta`**: those move when code
nobody touched moves, and reporting them would blame a session for something it did not do.

Incomparability is a result, not an error. A mismatched schema, doppel build, ontology version,
canon rule-set version or param set means the two runs measured different questions, so `Diff` sets `Comparable=false` with a
reason rather than returning a partial delta.

**The hook contract**, in `cmd/hook.go`:

- All four subcommands read the hook payload on stdin and write a hook response on stdout. **None ever
  exits non-zero or writes to stderr** — every failure path ends at `emitNothing`. A SessionStart
  hook's stderr surfaces to the user as a broken-tool notice, and blocking a session over a
  measurement would be indefensible.
- **Every hook subcommand measures at one operating point, derived once.** `hookParams` sets
  `Calibrate` to the same 0.01 default and `NoOverlapFilter: true`; `session-start` derives the
  thresholds and they land in the baseline's `snapshot.Params`, and `stop` / `user-prompt` supply
  them back through `pinThresholds` (`Params.Pinned` skips the derivation). Recalibrating per turn
  is what this avoids: the session is editing the corpus, so the null distribution moves, a
  threshold shifts by a hundredth, `Params` compare unequal and the Stop hook goes silent for a
  turn nothing was wrong with. A nil baseline means derive — right in all three places it can
  happen (session start *is* the deriving run; `user-prompt` scopes a digest without diffing;
  `stop` returns before that point when there is no baseline).

  `Params.NoOverlapFilter` exists because calibration derives an overlap floor as well as a
  code-shape one, and a hook run must not gain one: it diffs the **full** candidate set, and
  `StructMin` zero is how it says so. It is not a half-calibrated run — there is no overlap gate
  at all, and the recorded `StructMin` is 0, so two runs still agree exactly when they measured
  the same thing.
- `session-start` emits `additionalContext` (the corpus concept inventory — deliberately only
  what is session-stable: concept member counts, the seeds this corpus grew no practice for, roles,
  one pair-count line; per-target findings live in the two hooks below, where a target exists) and writes the baseline **only if one does
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
- **Both `stop` digests lead with the delta report** — the identity classification since the
  baseline, then the pairs those changes created or dissolved, each line carrying its stored
  `explain:` sentence. See *The delta report* below for the model; what belongs here is where the
  line was drawn:

  **The classification never buys a model turn.** `reporter.Notable` alone decides whether
  `additionalContext` is emitted, and its bar is untouched. A rename with no pair consequence is a
  fact about code the model just wrote and the user digest already carries; paying a turn to
  restate it would make the feature hostile. What the classification buys, *once a notable finding
  has already justified the note*, is the attribution the note is read through. So identity changes
  what the note says and never whether it is sent — `AgentDeltaDigest` returns "" for an empty
  `notable` list however much the delta found.

  **The `Reported` ledger covers it, by the existing mechanism.** `reporter.DeltaFindings` turns
  the report into `Finding`s keyed `class:<class>:<old>|<new>`, `created:<a>|<b>` and
  `dissolved:<a>|<b>` — prefixes deliberately distinct from `new:`, `gate:` and `drift:`, so two
  kinds of finding cannot collide on one key. They go through the same `unreported` filter and the
  same `withReported` write, and `AgentDeltaDigest` returns the findings from *both* halves that it
  actually rendered, because ledgering what was only counted retires findings nobody was shown. The
  user digest is deliberately not ledgered: it lets the turn end, and it is a cumulative statement
  of where the session stands rather than a feed.

  **`identityDelta` in `cmd/hook.go` recovers.** An error (a corrupt WL codec) and a panic both
  degrade to no delta section, never to a non-zero exit or a byte on stderr. The identity pass can
  only *refuse* on schema or rule set, both of which `snapshot.Diff` refuses on first, so the
  refusal path is unreachable from the hook and the error path is the one that matters.

  **`SessionDigest` composes and truncates once.** Two finished digests concatenated would bound
  each half at `digestMaxChars` and the sum at twice it, past what the harness carries — so
  `ImpactDigest` was split into `impactBody` plus the truncation, and nothing was re-rendered. It
  returns "" exactly when `ImpactDigest` would: every class but `unchanged` moves a key or a digest,
  so an identity finding always implies a non-empty `snapshot.Delta` and the impact half is the
  stricter emptiness test.
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

## Identity matching — `doppel diff`

`internal/identity` answers the question `snapshot.Diff` deliberately refuses to: given two
snapshots, *what happened to each function*. Diff keys on `package.Name` alone, so every rename
reads as one deletion plus one addition and a moved function reads the same way — the right rule
for a hook that must never claim more than it can attribute, and useless for reading a refactor.
Identity matches **bodies**, using the Weisfeiler-Lehman label bags schema 6 put on every `Unit`,
and reports one of eight classes per function: `unchanged`, `edited`, `renamed`, `moved`, `split`,
`merged`, `new`, `deleted`. **`snapshot.Diff` is untouched and stays the hook path**; nothing here
feeds a hook, a score or a ranking. The command is `doppel diff <old.json> <new.json>`, over two
files written by `doppel analyze --format json`.

**Two kinds of evidence, and each line says which it used.** `Unit.Digest` is exact — it hashes a
function's own fingerprint and nothing about the corpus, so equal non-empty digests mean the same
body, full stop, and body identity is decided nowhere else. The WL bag decides *similarity*, which
is what matching needs and digests cannot give: `fingerprint.WLOverlap` turns two bags into the
weighted Jaccard and the containment. Every reported line prints jaccard, containment and whether
the digests agreed, so a reader falsifies it by opening two files.

**The label IDF is counted over the union of both snapshots' bags.** Any other population makes the
answer asymmetric — `ln(N_old/df_old)` and `ln(N_new/df_new)` are different numbers for the same
label, so scoring one direction under each side's own norm would let a pair be each other's best
match one way and not the other, and the greedy matcher would then depend on which file was passed
first. The consequence worth knowing: these numbers are *not* the ones either snapshot's `Pair`
records carry, which were computed under one run's own weights. Nothing here reads `Pair` at all.

**Three matching passes, strongest evidence first.** Key equality (an unchanged `package.Name` is
the same function, whatever its body now says — a rewrite under an unchanged key is an edit, not a
deletion plus an unrelated arrival); then equal non-empty digest (the same body under a new name or
package, consumed in ascending key order within a bucket because identical bodies score identically
against everything and greedy would be deciding by index anyway); then greedy bipartite matching on
the exact WL overlap, admitted at `RenameFloor` and consumed by `(jaccard desc, containment desc,
old key asc, new key asc)`. Greedy rather than optimal deliberately: an assignment maximising total
similarity moves a pair off its own best match to improve someone else's, and the resulting report
has lines that cannot be checked in isolation. Greedy's invariant is local and printable — every
match was the best still available to both sides when it was made.

**One label per function, by this total order** (first rule wins; the facts the winning rule did not
use are still recorded and printed on the same line):

1. **moved** — the packages differ. A function that changed package is a move whatever else happened
   to it: relocation is what makes it unfindable where it was. `moved` + renamed + edited all report
   `moved`, with `(renamed X -> Y; body edited)` as secondary evidence.
2. **renamed** — same package, different name. A rename plus a small edit reports `renamed`, with
   `(body edited)`.
3. **edited** — same package and name, digests differ.
4. **unchanged** — same package and name, equal digests. Two empty digests count as equal.

**split / merged** run after matching, over the same candidate table, and absorb their participants
so every function still appears in exactly one finding. F is *split* when two or more distinct new
bodies each read containment ≥ `SplitContainment` against F. Three exclusions, all decided on
digests rather than on a second threshold: F is ineligible if **any** new function carries F's
digest (a body that still exists byte-for-byte was not divided — something was copied out of it,
which is duplication and what the rest of the tool reports); a participant carrying F's digest is
ineligible (a piece identical to the whole is not a piece); and an `unchanged` participant is
ineligible (a helper that already existed was not produced by F). The second exclusion is
load-bearing and was measured: without it, a corpus already holding two copies of one body reads as
a split the moment one copy is renamed, at containment 1.0000 on both parts. Splits are detected
before merges, over old functions in key order, and a function absorbed by a split cannot join a
merge — that ordering is what makes the two rules a total order rather than a race.

**Floors and knobs** live on `identity.Options`, zero value meaning defaults (the `retriever.Options`
convention): `RenameFloor` 0.5, `SplitContainment` 0.8, `CandidateK` 8, `MaxLabelDF` 200. The last
two are a **recall** bound and nothing else — candidate generation is the same inverted-index shape
retrieval uses, because all-pairs bag merges are fine on a few hundred functions and minutes on
moby, and every number reported is the exact `WLOverlap` of two bags, never the accumulator's
approximation. Measured: 0.03s on cobra (269 functions), 0.85s on moby (7,644).

**What it refuses, and the three mismatches it does not.** Schema and `RuleSet` refuse — a
pre-schema-6 snapshot has no bags at all, and a different canon rule set makes the same two
untouched bodies produce different labels, so matching across it would report a whole corpus as
edited. `Params`, `Ontology` and the doppel build are **allowed and noted in the report**, which is
looser than `snapshot.Diff` on purpose: Diff reads `Pairs`, whose every number is corpus-relative,
and identity reads only `Units`, whose key, digest and bag are properties of that function's own
AST. A population change (`--tests exclude` against `--tests include`) is not hidden either — it
shows up as new and deleted functions, which is a true statement about the two files. The build
check is the loosest, and refusing on it would block comparing a release binary's snapshot against
a local one while still not catching what it aims at: two plain `go build` binaries both report
`(devel)`.

**Refusal is a `Result`, not an error, inside the library** — mirroring `snapshot.Diff` so the two
surfaces cannot disagree about what "cannot be compared" means. At the CLI boundary it becomes a
non-zero exit, because a script must not read an empty change list as "nothing happened". Codes:
**0** compared, **1** a file is missing, is not JSON, or its WL codec is corrupt, **2** the two
snapshots refuse comparison.

**Presentation.** The text report groups by class and prints `unchanged` as a count only unless
`--unchanged` is passed — it is the bulk of any nearby comparison and burying six real findings
under eight hundred "still there" lines defeats the point. `--format json` has no such cutoff: it
carries every change, unchanged included, because a machine consumer filtering is trivial and one
missing data is not. All eight class counts are always emitted, zeros included, so the payload
shape is stable.

## The delta report

`identity.Since` is `Compare` plus one set difference: the near-duplicate **pairs those changes
created or dissolved**. Both snapshots already carry their pair lists, so this costs a walk of two
key sets and no re-scoring. `identity.Delta` is the pair of halves, and it is what both the
`doppel diff` renderings and the Stop hook's two digests lead with.

**It exists only where two snapshots exist.** `analyze` never reads the session baseline — the
hard convention — so a plain run's markdown and HTML *cannot* carry this section, and do not.
Three homes, in the order they matter:

1. **The Stop hook digests.** Both of them lead with it; see the hook contract above.
2. **`doppel diff`'s text form**, where the classification is already the whole output and the pair
   sections follow it under one `Delta since the baseline` title.
3. **`doppel diff --output <file.md>`**, a markdown form whose first section is the report because
   it is the only section. The extension selects nothing there — markdown is the command's only
   file form. **A two-run HTML dashboard is deliberately not built**: the dashboard is a
   payload-driven page describing one run, and a two-run page would be a different artifact rather
   than a format of this one. Future work, noted here rather than half-done.

**Attribution is the reason the classification leads.** `snapshot.Diff` can say a pair appeared and
that one side is new; only this can say the side is new *because a function was renamed into it*.
So a `PairChange` carries the class of each of its two sides — from the changes' `New` members for
a created pair, their `Old` members for a dissolved one, which is the case where looking a key up in
the other side's index would be silently wrong — and `Attributable()` is "either side is classified
as something other than `unchanged`". That is strictly wider than `snapshot.Delta.Touched()` (added
or digest-moved), and the two never disagree about a fact, only about how much they can name.

**Ordering is `snapshot.sortPairChanges`, mirrored**: attributable first, then merge-worthy, then
score desc, then the two keys. Only the first key means something different. In the hook, "pairs
involving functions in files the session edited come first" is exactly what that first key does —
**the classification is the edit signal**, and no per-file edit tracking is invented to improve on
it. The baseline wrapper's `Advised` ledger is not one: it records only files whose pre-tool
advisory actually rendered, and only when that hook is installed, so ordering on it would be a
partial record presented as a complete one. `BodiesChanged` is a strict subset of what the
classification already reports as `edited`.

**Every line carries the stored `explain:` sentence** — a created pair's from the new snapshot's
`Pair.Explain`, a dissolved pair's from the old one's. Read back, never recomputed: the sentence
names the canonicalization rules that fired on two specific bodies, and this package has neither
body. That is also why `snapshot.Diff` still does not *diff* the field — a reworded sentence is not
a session's doing — while the delta report renders it verbatim.

**One rendering, three surfaces.** The class lines are `identity.Lines` and the pair lines
`identity.PairLines`, in the text report, the markdown and both digests alike. A digest that spelled
these differently would be a second rendering to keep in step with the first.

**`--format json` gained the two lists without moving a field.** `Delta` embeds `Result`, so the
payload is what `WriteJSON` produced with `created` and `dissolved` appended, and a consumer written
against the classification alone keeps reading exactly the keys it always did.

Known limits, both inherent: a rename **re-keys every pair its function held**, so a single rename
produces N created and N dissolved pair changes that are one fact restated — the bounded head is
where the new duplication is and the tail is the restatement, which is why the pair bound is
tighter than the class bound. And two snapshots from `analyze --format json` carry the *ranked* pair
set (top-N, `--max-per-func`), so a pair can appear in one list purely because a presentation cutoff
moved; the hook path does not have this, because it snapshots the full candidate set.

## Conventions

- **Language-neutral by construction, Go-first by fidelity.** Everything from `fingerprint`
  outward reads `internal/syntax`, never any language's AST, so a frontend that can fill a
  `syntax.File` gets the fingerprint, the evidence channels, the call graph, the learned
  vocabulary and every corpus statistic without writing any of them. Two frontends exist:
  `internal/gofront` (go/ast, full fidelity, the only package importing `go/*`) and
  `internal/lexfront` (a tokenizer and a block rule, no grammar, 13 languages). No external
  parser dependency, and none is wanted — `go/ast` ships with the toolchain and the lexical
  path has no dependency at all. See *Frontends* below for the measured trade.

  **`internal/canon` is the one deliberate exception, and it is a frontend's business.**
  Canonicalization is a claim about what code *means*, which the IR carries nothing to make, so
  it stays `go/ast`-typed and Go-only, runs inside `gofront`, and reaches everything else as
  `syntax.Func.Canon`. A language with no canonicalizer still gets a label bag, a code-shape
  score, containment and the compression ratio — over its body as written. What it loses is
  recall on pairs that differ only in a spelling canonicalization would have removed, never
  correctness: the failure direction is "less similar than they are". *Frontends* states the
  boundary exactly.
- **No caches.** No run ever reads persisted state to avoid recomputation: `doppel hook stop`
  re-runs the whole pipeline from source every time, exactly as `analyze` does. The one persisted
  artifact is the **session baseline** written by `doppel hook session-start` — a measurement
  origin, not a cache. It is scoped to a single Claude Code session, lives in the OS temp
  directory under a hash of the session id, is discarded whenever the schema, the doppel build,
  the ontology version or the run params differ, and is swept after seven days. `analyze` must
  never read it, and no code path may short-circuit a stage because a baseline exists. If you find
  yourself adding a second state file, or reading this one to skip work, that is a design change,
  not an optimization. (`--format json` and `--output` write reports, not state.)

  **One documented exception, `pinThresholds` in `cmd/hook.go`:** a hook run reads the baseline's
  recorded `Threshold` and `Calibrate` and skips the calibration derivation. It is a supplied
  *parameter*, not a cached *result* — the thresholds arrive exactly as `--threshold` would supply
  them, every pipeline stage still runs from source every turn, and nothing analytical is reused.
  The justification is that a session baseline exists to pin the question being asked, and the
  operating point is part of that question; the alternative is the mid-session incomparability
  described under the hook contract. It is the only such read, and widening it is a design change.
- Cobra is the only direct **Go** dependency. Keep it that way unless there is a strong reason.
  Browser assets are a separate question and live vendored under
  `internal/dashboard/assets/vendor/` with their version, source and licence recorded — they never
  reach `go.mod`, and they are never hand-edited.
- Skipped directories: anything dot- or underscore-prefixed (the go tool's own ignore rule, so
  `_examples/` demo trees never join the population), plus `vendor`, `testdata`, `build`. The
  walk root itself is exempt — `doppel analyze .` hands the walker a directory named `.`, and a
  user pointing doppel at `_examples/` directly has already made the call.
  `_test.go` files are always parsed; **`--tests` decides the population** (default `exclude`).
  Tests are conventionally similar by design, so they form their own population: `exclude`
  models production practice, `only` is test-suite hygiene mode, `include` mixes both but
  **cross test/prod pairs are never reported** (different build units are never merge
  candidates). The filter runs before the lexicon, so every corpus statistic — the learned
  vocabulary itself, IC, dfs, culture,
  habitats, arenas — models exactly the population the report describes; filtering at report
  time instead would be the worst of both. `_test.go` is a compiler-recognized suffix, not a
  naming heuristic.
- Tested: `ontology`, `fingerprint`, `analyzer`, `comparator`, `tagger`, `parser`,
  `concepter/role`, `retriever`, `culture`, `lexicon`, `reporter`, `snapshot`, `cmd` config
  precedence and hook baseline handling. Untested and worth covering: `clique`, `clique` (covered
  only through `family`'s tests).
- `lexicon`'s `TestNoSeeds` is the language-portability claim as a test: the learner must produce
  usable concepts from a corpus with **no seed labels at all**, because that is what a frontend for
  another language starts from. It is no longer a promissory note — the Python standard library
  reports `Lexicon: 108 concepts (5 seeded, 103 emergent)`, and the emergent path is carrying the
  whole vocabulary exactly as the test says it must. Do not let it become a formality.
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
    and arenas are deliberately absent: they annotate, they never rank. `bench.RankKey` is a
    call to `analyzer.RankKey(p, DefaultRankOptions())` — one definition, so the scorecard's
    printed key cannot drift from the ranking. `Reretrieve(opt)` re-runs retrieval → pairs →
    compare under different `retriever.Options`, reusing tags, IC, graph and docs (exact: none of
    Options reaches them); `Rescore(onto)` re-runs compare alone.
  - **Measurement seams**, all no-ops at defaults and pinned as such: `fingerprint.Weights` /
    `SimilarityWith` (the production path is `Similarity` = `DefaultWeights()`; the blend is the
    same four-term sum in the same order, bit-identical), `retriever.Options.Weights` (zero value
    = defaults; cmd never sets it), `analyzer.RankOptions{TrophicPower, TestCallDiscount}` /
    `SortForReportWith` (power 2 uses `t*t`, not `math.Pow`, so the default key is byte-identical),
    and `ontology.WithWeights`. Options and arguments, never package globals.
  - `TestSelfWeight` (guard `DOPPEL_BENCH_SELFWEIGHT=1`) is the label-free weighting experiment:
    `comparator.SignalVector` (the twelve graded signals in `ScoredRelations` order; pinned to
    reproduce `OverlapScore` exactly) over the candidate set versus a null sample
    (`calibrate.SamplePairs`, cross test/prod dropped), Fisher ratio per signal normalized to
    weights, an entropy variant for contrast, each scored via `Rescore(WithWeights)`. **Measured
    on cobra and not adopted:** Fisher loads 0.37 on `shares_neighborhood` and 0.29 on `calls`
    and cuts `exhibits` to 0.05, and the labels get worse (merge 5.5 / refactor 20.0 against
    5.2 / 16.1; entropy 5.3 / 19.6). The contrast is confounded: what separates *retrieved* pairs
    from random pairs is largely call-graph adjacency, which retrieval selected for — the
    experiment measures what retrieval wants, not what a reviewer judges. Label-free weighting
    from this corpus alone cannot replace the hand-set table; the harness that could is the
    labeled fitter, once more corpora are labeled.
  - `TestMinIDF` and `TestMinIDFLadder` (guard `DOPPEL_BENCH_MINIDF=1`) measure the information
    floor against the absolute df caps: derived caps, union size, suppressed functions and
    surviving labels per floor on every fetched corpus, plus the labeled rankings where labels
    exist. Asserts nothing; see *Candidate retrieval* for the measured result and why the caps
    stayed absolute.
  - `TestLexiconMembershipLadder` and `TestLexiconMembershipLabels` (guard
    `DOPPEL_BENCH_LEXICON=1`) measure the membership rule: per corpus and per variant, the
    unlabelled share, assignments per function, concept count and largest concept as a share of
    the corpus, plus the labeled rankings where labels exist. `Run.LexOpt` (and
    `AnalyzeLexicon`) is the seam they score through, the membership analogue of `AnalyzeWith`'s
    vocabulary seam. They assert nothing; see *The learned lexicon* for the measured result and
    why `MaxMemberships` is 3.
  - `TestCalibrate` (guard `DOPPEL_BENCH_CALIBRATE=1`) scores null calibration at rates 0.005,
    0.01, 0.02 and 0.05: re-retrieves at the calibrated threshold and scores both the candidate set
    and the struct-min-filtered view, listing the labels that moved. Asserts nothing.
  - `TestThresholdLadder` (guard `DOPPEL_BENCH_THRESHOLD=1`) and `TestMinNodesLadder` /
    `TestMinNodesDistribution` (guard `DOPPEL_BENCH_MINNODES=1`) are the two floor derivations,
    kept so the shipped defaults are re-runnable rather than numbers in a commit message. The
    threshold ladder scores 0.30–0.60 (flat on cobra, which is why the calibrated median was safe
    to adopt); the min-nodes pair prints each corpus's node quantiles with the two pins that bound
    the floor from both sides, and scores 12–20. Both assert nothing.
  - `TestAblateFingerprint` (guard `DOPPEL_BENCH_ABLATE=1`, `task ablate`) zeroes each of the four
    fingerprint blend components in turn **without** renormalizing the rest — the question is
    whether a component pulls its own weight, not what happens once the others compensate.
    **Measured on cobra: nothing is dead and nothing is harmful.** The merge pairs are completely
    inert (4.5, 6/6, no violations in every row); every single ablation *lowers* the false-positive
    mean, i.e. each component is pushing false positives down and removing any one brings them up
    — WL 47.0→31.0, Flow →39.5, Signature →40.0, Depth →41.0. Zeroing WL improves the refactor
    mean (13.7→11.9) while costing 16 points of false-positive separation, which is a compression
    of the whole ranking rather than a quality gain. Note the contrast with the shingle metric this
    replaced, where zeroing the 0.60 slot *improved* the label means: the T3 metric change is what
    made that slot load-bearing. Depth buys the most separation per unit of weight (6.0 points for
    0.05) and that is *not* a licence to raise it — a single-point ablation on one corpus is a
    direction, not a gradient.

    Two further reasons the table must not be read as a tuning gradient. **It is not stable under
    changes that are not weights**: re-run after the `--threshold`/`--min-nodes` floors moved, the
    same ablation reads Flow and Depth fully *inert* on the false-positive mean and zeroing
    Signature as a 2-point *improvement*, where before the floors moved all four cost separation.
    Only the WL row keeps its sign and magnitude across both. With 2 false-positive and 9 refactor
    labels on one corpus, a 2-point mean move is one pair shifting four ranks — noise, and the
    instability is the proof of it. **Halving the `wl` blend weight is the concrete proposal this
    rejects**: it reads refactor 13.7 → 12.2 (better) against fp 47.0 → 42.0 (worse), and two of
    the three hard assertions are on the false-positive side, so it trades the corroborated
    quantity for the uncorroborated one in the same direction the ablation shows WL is carrying.
  - `TestSweep` (guard `DOPPEL_BENCH_SWEEP=1`) is the sensitivity sweep: each hand-set constant
    varied one at a time (±50% or the natural alternatives), only the stages it reaches re-run,
    and the labeled rankings reported with a verdict — `inert` (no label moved), `moves`,
    `load-bearing` (a violation, a presence change, or a merge-mean shift ≥ 1.0) — plus the labels
    that moved. It asserts nothing, and it is what first priced the `--min-nodes` default: the WL
    retrieval change tripped `AssertZeroFPInTop20`, the sweep priced the three constants that
    could clear it, and 12 → 18 was adopted on that evidence (`TestMinNodesLadder` later narrowed
    it to 16, the lowest value holding the same pin). **Measured on cobra (18 labels):**
    the merge pairs never move under any variant; every sensitivity is in the
    refactor/false-positive tail. Inert in both directions: `MaxConceptDF`, `fp.Depth`. Inert in
    one: `ChannelK`→8, `Threshold`→0.30, `MaxCallDF`→100, `calls_into_concept`×0.5,
    `shares_neighborhood`×0.5, `calls_into_package`×0.5, `called_from_concept`×2;
    `TestCallDiscount` (no test pairs under `exclude`). Load-bearing: `MinNodes` in either
    direction (it changes which pairs retrieval sees at all — this is the knob, so it should be).
    Largest movers: `fp.AST`, `MaxLabelDF`, `calls`, `exhibits`, `TrophicPower`. Not swept: `ChainTopN`
    (explanation only), `struct-min`/`family-min` (no bench analogue / census only),
    `ForkShapeFloor` (annotation). One corpus is a direction, not a verdict; the gin/chi labels
    are what would make it one.
  - `TestGenerateExamples` (guard `DOPPEL_BENCH_EXAMPLES=1`) regenerates `examples/<name>.md` by
    running the **built binary** with `cmd.Dir` set to the corpus — not the library — so the
    committed reports are what the documented command actually prints, culture/habitat/arena
    annotations included, with corpus-relative paths. The one stderr line naming a local path is
    filtered out. It also rewrites the ladder table in `examples/README.md` between its two
    markers, from `parseDiagnostics` over the same stderr block each report quotes — so the
    table and the reports cannot describe different runs — and only when **every** rung is
    fetched, because a partial ladder reads as a claim rather than as a partial measurement.
    A file is written only when its content changed modulo the `| doppel |` provenance row,
    which makes the recorded revision mean "where this report's content last moved" and makes
    regeneration idempotent enough for CI to commit. `DOPPEL_BENCH_EXAMPLES_CHECK=1` compares
    instead of writing (`task examples-check`).
  - `TestGeneratePagesSite` (guard `DOPPEL_BENCH_PAGES=<site root>`, relative to the module)
    renders the published site's examples section from the same manifest: one dashboard per
    rung via the built binary, plus the index, whose per-corpus counts are read back out of
    the committed Markdown reports rather than measured again — one analysis per corpus, and a
    card that cannot disagree with the report it links to. `task pages` builds the whole site
    locally.
  - `TestExamplesManifest` and `TestLadderMatchesReports` are the offline half, unguarded and
    in every `go test ./...`: the first asserts each rung has a report quoting the manifest's
    pinned tag and commit and that the ladder markers exist, the second re-derives the whole
    ladder table from the committed reports' diagnostics blocks and requires it byte-identical
    to the committed table. Neither can see a ranking change — only running the tool over the
    ladder measures that — but between them a repinned rung, a hand-edited table and a broken
    diagnostics regex all fail without a clone.

## Rough edges

Known traps, documented so they aren't rediscovered. None are fixed:

- **Lexical fidelity is measured on Go and assumed elsewhere.** `TestLexicalFidelity` scores the
  lexical frontend against `go/ast` on Go corpora at 0.994–0.999 recall and 1.000 precision, but
  Go is a brace language with a keyword-led declaration syntax — the easy case for the block
  rule. Python is exercised by unit tests and by hand (1 761 of 1 813 `def`s on a 75-file
  standard-library corpus, the balance being nested closures, which are dropped by design); the
  other eleven languages in `lexfront.Specs` are the machinery pointed at conventions written
  down from knowledge, not from measurement. Labels for one non-Go corpus would turn a direction
  into a verdict, exactly as the gin/chi labels would for scoring.
- **The lexical frontend has no types, and one-line bodies are its blind spot.** A Python
  `def f(): return x` is not found: the block rule needs a colon that ends its line. The same
  goes for any construct whose body opens more than one line after the header. Both fail toward
  a *missing* unit rather than a wrong one, which is why precision stays at 1.000 while recall
  is what moves.
- **A non-Go label bag is real evidence, and it is coarser in two independent ways.** The bag,
  the code-shape score, containment, the compression ratio and the explanation layer all work on
  any frontend's tree. But no `lexfront` language has a canonicalizer, so two bodies differing
  only in a spelling Go would normalize away — early return versus else, `x += 1` versus `x++`,
  operands either way round a commutative operator, locals bound in a different order — stay
  apart. And the tree is a *block* tree, so most interior nodes are `KindOther` and the deeper
  refinement rounds fold in less: an h=3 label summarises a whole guard on a Go body and a
  nesting region on a lexically-segmented one. Both errors point the same way, "less similar
  than they are", which is the direction a reader can act on — but a cross-language recall
  comparison has never been measured, and the numbers in *Frontends* are recall of *units*, not
  of pairs. Nothing here is unfixable in principle: a per-language canonicalizer is a frontend
  concern, and the seam for one already exists (`syntax.Func.Canon`).
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
  concept, and no shared resolved call is never compared, no matter how alike it is — that is the
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
- **Habitat = package is crude, and the subsystem rollup is one level deep.** One Go package can
  host several micro-habitats (handlers next to helpers), and the rollup excuses in only one
  direction — up to the parent directory. A package directly under the root has no parent and
  can never be excused (most of hugo's misfits live in `hugolib`, `resources`, …, which is why
  hugo drops least), and a repo whose packages all sit under one umbrella (`internal/`, `pkg/`)
  gets that umbrella as its only subsystem, which is generous. The delta quantities remain future
  work: Δentropy/fragmentation, phase transitions, chemical potential (marginal duplication
  pressure), git-derived heat, inter-package JS divergence, free energy.
- **Convention entropy is a dispersion proxy, not realization clustering.** It measures how
  predictable each practice is across members, not how many distinct whole realizations exist.
  Habitat fit and misfit notes are corpus-relative like roles and typicality — a function's fit
  can move when unrelated code shifts its package.
- **A distinctive call token can carry a false name.** The call channel inherits the resolver's
  imprecision: a variable-receiver method call resolves only when the method name is unique
  corpus-wide, so on moby every `mu.Unlock()` resolves to `sdjournal.noCopy.Unlock`, the one
  declaration of that name. The lift is real — concurrency functions do unlock far more than the
  corpus does — but the label names one arbitrary declaration rather than the practice. Fixing it
  needs `go/types`; the practice section is now the most visible place this shows.
- **The practice section is bounded by row count, not by importance.** Six concepts, six
  associations per direction, ten drifting functions — and on prometheus that is 6 of 371
  associations and 10 of 32 unusual realizations. Strongest-first is a good proxy and not the same
  thing as most-relevant: a weak association between two subsystems that should never touch may
  matter more than a strong one inside a package. The full lists remain reachable only through the
  library.
- **Typicality is corpus-relative, like roles.** A function's typicality — and whether a pair
  carries a culture note — can change when unrelated code shifts the concept's membership or the
  corpus norm. That is what "normal for this repo" means; same caveat as the role thresholds.
- **The trivial-body floor is two labeled pins wide, and a third constant would still have
  served.** A WL bag gives every body `wlRounds+1` labels per node, and the deep ones are df 1 or
  2 whenever the body is corpus-unique, so a trivial-but-unique one-liner earns maximal-IDF
  evidence that the pattern hierarchy never granted it (see *Fingerprint scoring*).
  `--min-nodes 16` is the shipped answer, and it is now pinned from *both* sides rather than one:
  cobra's `Less` false positive is 15 nodes and must stay out, conc's `Wait` clone family is 16
  and should come in. It is still not the only answer the sweep found — at the old `12`,
  `MaxLabelDF 50→100` also pushed the false positive to rank 21 — but that measurement was taken
  when the pair was retrieved at all; at 16 it never reaches the comparator, so the cap is no
  longer doing that job and loosening it would be tuning a cap to fix a retrieval symptom the
  floor already fixes. Measured on the shipped floors, `MaxLabelDF 100` reads merge 4.5 / refactor
  13.3 / fp 49.0 / no violations — mildly *better* on the one labeled corpus, and still **not
  adopted**: it doubles a cap whose job is idiom suppression on the large unlabeled rungs, which is
  precisely the `MinIDF` situation and gets the same answer. What is missing before it could be
  taken is the large-corpus union-growth and suppression numbers (there is no flag for the cap, so
  it cannot be measured from the CLI) and a second labeled corpus. **One corpus is still a
  direction, not a verdict** — cobra is the only rung
  with labels, and neither pin has been corroborated on gin, chi or the large rungs. The deeper
  option, not attempted: count each *node* agreement once at its deepest agreeing round instead of
  once per round, which would remove the implied-label multiplication (agreeing at h=3 implies
  agreeing at h=0..2) rather than raising a floor to compensate for it. That changes the mass
  definition and needs `WLBag` to retain the node→label chain.
- **The floor is an absolute node count against a corpus-relative problem, and a relative one was
  measured and not needed.** `--min-nodes 18` used to cost conc its shape channel entirely: 81
  functions of three-to-five-line generic pool wrappers, almost none clearing 18 nodes, shape
  admissions **23 → 3**, family census **3 families over 12 functions → none**, and
  `ResultContextPool.Wait ↔ ResultErrorPool.Wait` — genuinely identical, code-shape 1.00 — gone
  from the report. The obvious repair was a quantile of the corpus's own node distribution, the
  way `calibrate` derives thresholds. `TestMinNodesDistribution` measured it and the quantile is
  **not** needed: the two pins are one node apart (15 out, 16 in), so the absolute **16** already
  separates them, and it restores conc to 14 shape admissions and one family. A relative floor
  would also have had to satisfy both pins at once — only `q ∈ [0.38, 0.58]` does — and at
  `q=0.40` it would set chi's floor to 23 and prometheus's to 24 while dropping gin's to 10,
  changing five unlabeled corpora to fix a problem an absolute constant fixes exactly. That is the
  `MinIDF` failure mode, so the same verdict applies: measured, documented, not adopted. What
  would revive it is a corpus whose pins genuinely straddle — a small library needing a floor
  below 16 while a large one needs it above.
- **Trophic similarity of exact mid-frequency twins is 1.0.** Any normalized similarity gives
  identical inputs 1.0; trivia suppression relies on the df cap zeroing *both* sides of the Dice,
  which only engages once the idiom bucket exceeds `MaxLabelDF`. Between df=2 and the cap, exact
  twins read trophic 1.0 with *small* energy — the energy is what ranks, so this is display-level
  nuance, not a scoring bug.
- **Corroborated ranking has thin margins on vocabulary-heavy pairs.** A cross-package true
  clone and a vocabulary false positive can sit ~10-20% apart in key; the golden benchmark
  watches exactly this. Trophic² also discounts non-identical true clones somewhat (the
  production clone sits mid-top-50, not top-20, in the full-population view) — the price of
  demoting skeleton siblings.
- **Generated-code and demo-tree suppression is convention-deep only.** The two historical
  dominators are fixed by the ecosystem's own declarations — `--generated exclude` (default)
  filters files carrying Go's "Code generated ... DO NOT EDIT." marker, and the walker skips
  dot-/underscore-prefixed directories exactly as the go tool does, which is what removed moby's
  all-`.pb.go` top ten and chi's `_examples/` `main`s. But a generator that omits the marker, or
  a demo tree named `examples/` without the underscore, is invisible to both rules, and that is
  deliberate: the alternative is path/name heuristics, which the tagger and retriever refuse
  everywhere else. Narrowing to a subtree remains the answer for focus (`prometheus/tsdb`
  reports the float/int histogram duplication by itself).
- **A calibrated threshold moves between separate `analyze` runs.** The hook half of this is
  fixed — `pinThresholds` derives once per session and supplies the result back, so a baseline
  cannot go incomparable through no pair's fault. Two plain `analyze` runs over a changed tree
  still get different thresholds, which is correct (the corpus changed) but means a report's
  numbers are only comparable to another report at the same stated operating point. Rounding to
  0.01 bounds how often it moves, it does not stop it. Under `--tests include` the null is also a
  two-population mixture (cross pairs are rejected, but test and production bodies are drawn
  together), so the derived floor there describes neither population exactly.
- **Committed examples are regenerated by CI, but only after the fact.** `examples.yml`
  re-runs the ladder on every push to `master` and commits what moved, so the reports and the
  ladder table no longer go stale silently. What it cannot do is tell you *before* you push:
  a PR's reports are whatever its branch inherited, and the bot commit lands afterwards. Run
  `task examples` in the same change that moves ranking and re-read the diff — it is the
  fastest available review of what a scoring change actually did, and `task examples-check`
  is the read-only form. `task golden` is the assertion-backed half of the check; the reports
  are the eyeball half. The **performance table** in `examples/README.md` is still hand-copied
  from a `task bench` run on one machine and still goes stale silently — no run can measure
  another machine's stopwatch. The prose around both tables is hand-written too, so a corpus
  whose character changes leaves the paragraph describing it wrong while every number beside
  it is right.

- **Families overlap, and that is reported rather than resolved.** A function can belong to
  several maximal cliques, so the family list can show it twice; the counts report *distinct*
  functions. Picking one clique per function would be a judgement the tool cannot justify, and a
  partition would have to break a tie no evidence decides. Family membership is also corpus-relative
  in the same way roles and typicality are — the pair graph moves when unrelated code moves — and it
  is bounded by the same retrieval recall the pair list is, except inside a component, where edge
  completion repairs it.
- **The report overview is bounded by node count, not by importance.** The package diagrams show
  the twelve least uniform habitats and the twelve heaviest duplication links, and count the rest.
  On a wide corpus that is a sample, not a summary — the thirteenth link may matter more than the
  first if it crosses a subsystem boundary the tool cannot see. Family diagrams stop at 8 members
  for the same arithmetic reason, so the largest families — exactly the ones a census exists to
  surface — are the ones shown only as prose.
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

  Two sharpenings, both measured. First, the `(devel)` in that sentence is the *last* fallback:
  on a modern toolchain `debug.ReadBuildInfo().Main.Version` usually resolves to a VCS
  pseudo-version first, so a plain build often does carry a commit. Second, **that commit can be
  the wrong one.** Built inside a `git worktree`, the stamp resolves against the shared git
  directory and reports the *main* worktree's HEAD, not the code being compiled — observed here
  as `v0.1.2-0.…-885cdfc` from a tree whose own HEAD was several commits elsewhere. So the
  fallback is not merely coarse, it can be confidently wrong in exactly the setup an agent uses.
  Hardcoding a base version into `cmd.version` would replace one wrong answer with a worse one
  (every dev build of every tree claiming one identity — the `{{ .Tag }}`-under-`--snapshot`
  failure the release config exists to avoid), so it stays as-is and `task build-stamped` remains
  the answer.

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

- **Interface implementations are labeled, not filtered.** Methods satisfying a shared interface
  across sibling packages — a `Validate` per provider — are near-identical by construction and
  unactionable, and on a wide corpus they still crowd the list. The `interface implementations`
  kind names them by a naming rule (same method, same signature, different types), which is
  acceptable as an *annotation* precisely because it cannot be wrong about what it claims (both
  sides do implement that method) while it can be wrong about whether an interface exists — that
  needs `go/types`. Filtering or demoting on the same rule would be the name heuristic the tagger
  and retriever refuse everywhere else, and is deliberately not done.
- **Fork stems are lexical.** `diverged copy` reads version markers in names, nothing else: a fork
  that kept the same name in two packages, or renamed freely, is invisible; `NewClient`/`newClient`
  at high shape is labeled (the rule strips `New`, case falls out); numbered series (`sha256`/
  `sha512`) are excluded by design; lower-case markers inside a word (`Threshold`) are not markers.
  The 0.60 shape floor and the same/sibling-package locality bound the damage.

- **A learned vocabulary costs real time, and the cost is corpus-derived.** moby went from ~1.8s
  end to end to ~9.7s, prometheus from ~2.6s to ~5.5s. Almost none of that is the lexicon itself
  (~1s on moby); it is the stages downstream that now do proportionate work — `culture` models 394
  concepts where it modeled 12, and retrieval's concept channel admits several times more pairs.
  Three pathologies were found and fixed along the way and are worth not reintroducing: the arena
  rebuilt its interaction matrix by map lookup inside the replicator loop (90% of a prometheus run
  once candidate sets grew), `lexicon.assign` scored every unit against every concept's vocabulary
  instead of inverting the index, and `ontology.LCA` allocated a map per call. What remains is
  proportionate, not pathological — but the Stop hook runs a full analysis every turn, so on a very
  large repo this is the first thing that will be felt.
- **A learned concept can be enormous.** moby's largest reads
  `Config.OpenStdin+Config.StdinOnce` and has 922 members — 12% of the corpus. It is real (those
  functions do touch container config) but it is weak evidence, and nothing bounds a concept's size
  directly. IC handles it automatically, which is why nothing does bound it: a concept a tenth of
  the corpus carries contributes almost nothing to any pair that shares it. The df window bounds
  *features*, not concepts, deliberately.
- **Concept identity is corpus-relative, and cross-corpus names do not compare.** Two repositories'
  `sql.Open+QueryRow` concepts are different objects that happen to share a spelling, and within one
  repository a concept's name can change when unrelated code shifts which feature weighs most. This
  is the same caveat roles, typicality and habitat fit already carry, widened to the vocabulary
  itself. `snapshot.Digest` is unaffected — it hashes the fingerprint alone — so `BodiesChanged` and
  the `Attributable` contract still mean exactly what they meant.
- **Seed bias remains.** Fourteen Go-shaped seeds still give some concepts a head start, and a
  seeded concept claims its vocabulary before the emergent pass runs (`claimed` bounds what may
  *found* a cluster, though never what a vocabulary may contain). `TestNoSeeds` bounds how much this
  matters by proving the unseeded path works; it does not remove the bias.
- **Emergent concepts are found per feature-neighbourhood, so recall is per-feature.** A practice
  whose features all fall outside each other's top `EdgeK` associations is not found, the same
  bounded-recall trade retrieval already makes. `MaxEmergentFeatures` (2000) and `MaxUnitFeatures`
  (64) bound the co-occurrence pass and are reported in `Stats` rather than applied silently, but a
  corpus with more than 2000 informative seedable features is sampled, not covered.
- **The graded matcher is not proven optimal.** `SetRelatednessW` keeps the matching itself
  unweighted precisely so the oracle test still certifies the pair *choice*; with confidences in the
  objective, greedy matching genuinely is not optimal in general. The scores it produces are
  therefore a defensible lower bound on the weighted objective, not the objective's maximum. It
  feeds ranking, never a gate.
- **The golden benchmark moved slightly the wrong way and was accepted.** On cobra, the only
  labeled corpus: merge mean rank 5.2 → 5.5, refactor 16.1 → 16.8, false-positive 40.0 → 39.0. All
  three hard assertions stay green (6/6 merges retrieved and in the top 50, no false positive in
  the top 20). One corpus is a direction, not a verdict, and the concept channel's recall changed by
  an order of magnitude — gin/chi labels are what would say whether the extra recall is worth the
  drift.

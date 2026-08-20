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
the usual culprit — see `sortedKeys` in `mapper` and the tie-break in `FindSimilar`).

## Pipeline

Orchestrated end to end by `runAnalyze` in `cmd/analyze.go`. Stages in execution order:

1. **Walk & parse** — `filepath.WalkDir` + `shouldSkipDir`, then `parser.Parse` per `.go` file → `[]CodeUnit`. Unreadable files and parse errors are warned and skipped, never fatal. `fingerprint.Build` runs here, while the AST is still in hand.
2. **Tag** — `tagger.Tag(unit.Body)` sets `unit.Patterns` (9 keyword-matched intent tags).
3. **Build call graph** — `concepter.BuildCallGraph(units)` → `map[calleeName][]callerName`. Note this happens *before* concept docs, because docs need caller lists.
4. **Generate + enrich concept docs** — `concepter.New()` makes bare docs; **`mapper.Map` does the real work**: attaches callers, calls `concepter.ClassifyRole`, and aggregates caller/callee patterns and packages.
5. **Compare fingerprints** — `analyzer.FindSimilar` scores the full O(n²) upper triangle, keeps `score >= threshold`, sorts descending, truncates to `--top`.
6. **Structural comparison** — `comparator.Compare` per surviving pair → `pair.Evidence`.
7. **Structural filter** — when `--struct-min > 0`, pairs below that overlap score are **dropped**. This is a selection stage, not just annotation.
8. **Report** — `reporter.Print` to stdout always; `reporter.PrintMarkdown` to `--output` additionally.

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
internal/
  parser/       parser.go is a thin dispatcher; go_parser.go does all go/ast work → CodeUnit
  fingerprint/  AST token shingles + control-flow histogram + signature types; the code-similarity score
  tagger/       Keyword-substring intent detection → 9 pattern tags
  concepter/    ConceptDoc; callgraph.go (BuildCallGraph); role.go (ClassifyRole, role constants)
  mapper/       Where enrichment actually happens: callers, role classification, aggregated patterns/packages
  analyzer/     Pairwise fingerprint scoring, threshold filtering, top-N sorting
  comparator/   Weighted structural overlap scoring (9 signals → 0.0–1.0 composite)
  reporter/     Plain-text (stdout) and Markdown (--output) formatting
```

Dependency directions that must hold: `analyzer` imports `comparator` (for the `Evidence` field), so
`comparator` must never import `analyzer`. `parser` imports `fingerprint`, so `fingerprint` must
never import `parser` — it works on `*ast.FuncDecl` directly.

## Two scores, deliberately unblended

Each pair carries two independent numbers, gated by two independent flags:

| Score | Source | Flag | Means |
| --- | --- | --- | --- |
| `Score` | `fingerprint.Similarity` | `--threshold` | how alike the two bodies are |
| `Evidence.OverlapScore` | `comparator.Compare` | `--struct-min` | how much architectural context they share |

Do not merge these into one number. High code score + low overlap is a *different finding* (lookalike
bodies in unrelated subsystems) from high on both (a real merge candidate), and collapsing them
destroys that distinction.

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
- **ConceptDoc** (`internal/concepter/concepter.go`) — the architectural context the comparator
  scores. It is no longer rendered to text anywhere; `Format()` existed only to build embedding
  input and is gone.
- **SimilarPair** (`internal/analyzer/similarity.go`) — two `CodeUnit`s plus `AIdx`/`BIdx`, `Score`,
  `Breakdown` (per-component code similarity), and `Evidence` (**nil until the structural
  comparison stage**).
- **StructuralEvidence** (`internal/comparator/comparator.go`) — the weighted overlap result.

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

`--min-nodes` (default `12`) excludes tiny bodies from comparison entirely. Without it one-line
accessors match each other at 1.0 and flood the report. On this repo the guard currently excludes 4
functions and changes no reported pair at the default threshold — it is preventive, so do not
"simplify" it away on the evidence of a clean run here.

### Comparator weights

Constants in `internal/comparator/comparator.go`; they sum to exactly `1.00` and the composite is clamped to `1.0`.

| Signal | Weight |
| --- | --- |
| shared callees | 0.25 |
| shared patterns | 0.20 |
| shared callers | 0.15 |
| same role | 0.15 |
| same package | 0.10 |
| same visibility | 0.05 |
| same receiver type | 0.05 |
| shared caller packages | 0.025 |
| shared callee packages | 0.025 |

`MergeWorthy = OverlapScore >= 0.4 && countSignals >= 2`. **`countSignals` counts only 5 of the 9**
— callees, callers, patterns, role, package. Visibility, receiver, and the two package-overlap
signals raise the score but never the signal count.

Floor effect worth knowing: `SameVisibility` is true when both are unexported *or* both exported,
and `SameReceiver` is true when both are plain functions (`"" == ""`). Any two plain unexported
functions therefore start with a free `0.10`.

### Roles

`concepter.ClassifyRole(callerCount, calleeCount)`, one shared `roleThreshold = 2`, inclusive:

| | few callees (<2) | many callees (>=2) |
| --- | --- | --- |
| **few callers (<2)** | `leaf` | `orchestrator` |
| **many callers (>=2)** | `utility` | `passthrough` |

Caveat: `Callees` counts *every* call expression including stdlib (`fmt.Errorf`, `len`, `append`),
so most non-trivial functions clear the fan-out threshold and roles skew to
`orchestrator`/`passthrough`, making `SameRole` fire often.

### Tagger patterns

Exactly 9, emitted in declaration order: `retry`, `http_call`, `db_access`, `validation`, `mapping`,
`transaction`, `caching`, `concurrency`, `error_wrapping`. Their keyword lists still contain
non-Go signals (`axios`, `urllib`, `Promise.`, `await `) left over from the pre-Go-only era — dead
weight, since only `.go` files are ever parsed.

## Configuration

`.doppel.json` at repo root, or `--config <path>`. A missing file is not an error; malformed JSON is.
**Keys are kebab-case** (they mirror the flag names), not snake_case:

```json
{
  "threshold": 0.65,
  "top": 10,
  "min-nodes": 12,
  "struct-min": 0.4,
  "output": "doppel-report.md"
}
```

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
- Tested: `fingerprint`, `analyzer`, `comparator`, `concepter/role`, `cmd` config precedence.
  Untested and worth covering: `parser`, `tagger`, `mapper`, `reporter`.

## Rough edges

Known traps, documented so they aren't rediscovered. None are fixed:

- **Name-key mismatch in the call graph.** `BuildCallGraph`, `mapper`'s index, and `extractCallees`
  all key on bare function names, so `New` in two different packages shares call-graph edges. The
  pipeline no longer depends on name keys (see above), but `Callers`, `CallerPackages` and the role
  classification derived from them are still affected by this conflation.
- **Silent parse failures.** `parseGo` returns `nil, nil` on any syntax error, despite a comment
  claiming it returns partial results.
- **Stale doc comment** on `BuildCallGraph` describes only the O(n²) text-scan fallback and predates
  the AST-callee fast path that now handles virtually every case. With Go-only parsing, `Callees` is
  always populated, so the text-scan branch is effectively dead code.
- **O(n²) with no blocking.** Every function pair is compared. Fine at the current scale (74
  functions here, ~2.7k pairs), but a large repo would want MinHash/LSH banding to cut the candidate
  set before exact scoring.

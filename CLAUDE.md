# Doppel

Go CLI tool that detects structural similarities in a Go codebase to surface merge candidates — functions or methods that do similar enough work to warrant consolidation.

## Goal

Spot structural duplication that text-based tools miss. Doppel uses semantic embeddings (via Ollama) and structural analysis to find function pairs that share intent, call patterns, and role — not just string overlap. The output is a ranked list of merge candidates with confidence scores and evidence explaining *why* they are similar.

## Pipeline

Orchestrated end to end by `runAnalyze` in `cmd/analyze.go`. Stages in execution order:

1. **Walk & parse** — `filepath.WalkDir` + `shouldSkipDir`, then `parser.Parse` per `.go` file → `[]CodeUnit`. Unreadable files and parse errors are warned and skipped, never fatal.
2. **Tag** — `tagger.Tag(unit.Body)` sets `unit.Patterns` (9 keyword-matched intent tags).
3. **Build call graph** — `concepter.BuildCallGraph(units)` → `map[calleeName][]callerName`. Note this happens *before* concept docs, because docs need caller lists.
4. **Generate + enrich concept docs** — `concepter.New()` makes bare docs; **`mapper.Map` does the real work**: attaches callers, calls `concepter.ClassifyRole`, and aggregates caller/callee patterns and packages.
5. **Render** — `ConceptDoc.Format()` produces the text block that is actually embedded.
6. **Embed** — `embedder.Embed` per doc via Ollama `/api/embed`, wrapped in `embedWithBackoff`; cache saved afterwards.
7. **Cosine similarity** — `analyzer.FindSimilar` scores the full O(n²) upper triangle, keeps `score >= threshold`, sorts descending, truncates to `--top`.
8. **Structural comparison** — `comparator.Compare` per surviving pair → `pair.Evidence`.
9. **Structural filter** — when `--struct-min > 0`, pairs below that overlap score are **dropped**. This is a selection stage, not just annotation.
10. **Optional LLM reflection** — when `--reflect-model` is set, `reflector.Explain` fills `pair.Explanation`. Per-pair failures warn and continue.
11. **Report** — `reporter.Print` to stdout always; `reporter.PrintMarkdown` to `--output` additionally.

**The single most important non-obvious fact:** what gets embedded is the *concept doc*
(`ConceptDoc.Format()`), not the function body. Only the reflector ever sends real bodies to the
LLM. The `--max-input` help text says "function body" and is wrong.

## Module layout

```
main.go         Thin entry point → cmd.Execute()
cmd/            CLI commands (Cobra).
  root.go       rootCmd, Execute()
  analyze.go    Pipeline orchestrator; all flag registration; embed backoff helpers
  config.go     .doppel.json loading (AnalysisConfig) and flag precedence
internal/
  parser/       parser.go is a thin dispatcher; go_parser.go does all go/ast work → CodeUnit
  tagger/       Keyword-substring intent detection → 9 pattern tags
  concepter/    ConceptDoc + Format(); callgraph.go (BuildCallGraph); role.go (ClassifyRole, role constants)
  mapper/       Where enrichment actually happens: callers, role classification, aggregated patterns/packages
  embedder/     Ollama /api/embed client with SHA-256 keyed on-disk cache
  analyzer/     Pairwise cosine similarity, threshold filtering, top-N sorting
  comparator/   Weighted structural overlap scoring (9 signals → 0.0–1.0 composite)
  reflector/    LLM merge rationale via Ollama /api/generate; injects structural evidence into the prompt
  reporter/     Plain-text (stdout) and Markdown (--output) formatting
```

`analyzer` imports `comparator` (for the `Evidence` field), so `comparator` must never import `analyzer`.

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
doppel analyze . --reflect-model llama3.2 --output report.md   # full run with LLM explanations
```

Requires Ollama running locally (`http://localhost:11434`) with an embedding model pulled
(default: `nomic-embed-text`).

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
  `Callees`. Methods are named `"*Server.Start"` — the receiver keeps its star.
- **ConceptDoc** (`internal/concepter/concepter.go`) — the semantic representation that gets embedded.
  Beware: `Inputs`, `Outputs`, and `Dependencies` are declared and rendered by `Format()` but **never
  populated by anything**, so those sections never appear in a real embedding input.
- **SimilarPair** (`internal/analyzer/similarity.go`) — two `CodeUnit`s plus `Score`, `Explanation`
  (empty unless `--reflect-model`), and `Evidence` (**nil until the structural comparison stage**).
- **StructuralEvidence** (`internal/comparator/comparator.go`) — the weighted overlap result.

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
  "threshold": 0.85,
  "top": 10,
  "reflect-model": "llama3.2",
  "output": "doppel-report.md"
}
```

Precedence: `applyConfig` only calls `Flags().Set` when `!Flags().Changed(name)`, so explicit CLI
flags always win over the file.

Three gaps to know about:

- `struct-min` is the only functional flag with **no config key** — CLI only.
- `concept-cache` is a config key with **no matching flag**; the resulting "no such flag" error is
  discarded, so setting it silently does nothing.
- `--concept-model` and `--concept-prompt-file` are registered and parsed but **never read** by
  `runAnalyze` — dead leftovers from the LLM-based concepter removed in `98541fd`.

## Conventions

- Go-only. All parsing uses `go/ast` — no external parsers, no multi-language support.
- One cache: `.embeddings.json`, keyed `sha256(model \x00 numCtx \x00 concept-doc-text)`.
  `--cache ""` disables it. There is **no** `.concepts.json` — that cache was deleted in `98541fd`.
- Skipped directories: `.git`, `.claude`, `vendor`, `testdata`, `build`, `.idea`, `.vscode`.
- Only two test files exist: `internal/concepter/role_test.go` and `internal/comparator/comparator_test.go`.
  Everything else is untested — new tests are welcome, especially for `parser`, `tagger`, `mapper`,
  and `cmd` config precedence.

## Rough edges

Known traps, documented so they aren't rediscovered. None are fixed:

- **Name-key mismatch.** `BuildCallGraph`, `mapper`'s index, and `extractCallees` all key on bare
  function names, so `New` in two different packages shares call-graph edges. But `docIndex` in
  `analyze.go` keys on `Package + "." + Name` — a different key space — so evidence attachment can
  silently miss, leaving `Evidence == nil`, which then makes `--struct-min` drop the pair.
- **No HTTP timeouts.** Both `embedder` and `reflector` use the package-level `http.Post` with no
  timeout, no `context.Context`, and no retry. A hung Ollama hangs the entire run.
- **Silent parse failures.** `parseGo` returns `nil, nil` on any syntax error, despite a comment
  claiming it returns partial results.
- **Formatting drift.** `gofmt -l .` currently reports 5 unformatted files (`cmd/analyze.go`,
  `cmd/config.go`, `internal/analyzer/similarity.go`, `internal/comparator/comparator.go`,
  `internal/concepter/concepter.go`). The pre-commit hook will block commits touching them until
  `task fmt` is run.
- **Stale doc comment** on `BuildCallGraph` describes only the O(n²) text-scan fallback and predates
  the AST-callee fast path that now handles virtually every case.

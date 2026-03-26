# doppel

A CLI tool that detects semantically similar functions across a Go codebase using local AI embeddings. It helps identify duplicate logic and refactoring opportunities by comparing function bodies with vector similarity rather than text matching.

## How it works

1. **Parse** — walks the target directory and extracts all Go function/method bodies, names, signatures, and line numbers using the `go/ast` package; non-`.go` files are skipped
2. **Tag** — scans each function body for intent patterns (`retry`, `http_call`, `db_access`, `validation`, `mapping`, `transaction`, `caching`, `concurrency`, `error_wrapping`) using keyword matching
3. **Build call graph** — maps which functions call which, used to enrich concept docs with caller context
4. **Generate concept docs** — creates a structured semantic summary per function (name, package, I/O types, external dependencies, callers, patterns); if `--concept-model` is provided the summary is written by an LLM, otherwise static analysis is used; cached in `.concepts.json`
5. **Embed** — sends each concept doc (not the raw function body) to a local [Ollama](https://ollama.com) embedding model to generate a semantic vector; cached in `.embeddings.json`
6. **Compare** — computes pairwise cosine similarity across all vectors
7. **Reflect** *(optional)* — uses a chat model (`--reflect-model`) to explain why each pair is similar and how they could be merged
8. **Report** — prints the most similar pairs above a configurable threshold to stdout and optionally saves a Markdown file

Results are printed to stdout as a plain-text report and optionally saved as a Markdown file.

## Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [Ollama](https://ollama.com) running locally (default: `http://localhost:11434`)
- An Ollama embedding model pulled, e.g.:
  ```bash
  ollama pull nomic-embed-text
  ```

## Installation

```bash
go install github.com/lukse/doppel@latest
```

Or build from source:

```bash
git clone https://github.com/lukse/doppel
cd doppel
go build -o doppel .
```

## Usage

```bash
doppel analyze <path> [flags]
```

### Examples

```bash
# Analyze current directory with defaults
doppel analyze .

# Lower the threshold to catch more subtle similarities
doppel analyze ./src --threshold 0.80

# Save a Markdown report and use a chat model to explain each match
doppel analyze . --reflect-model llama3.2 --output report.md

# Full semantic analysis with LLM concept docs and merge explanations
doppel analyze . --concept-model llama3.2 --reflect-model llama3.2 --output report.md

# Use a long-context embedding model for large functions
doppel analyze . --model qwen3-embedding-8b --ollama-num-ctx 32768
```

## Iterative Refactoring Loop

A single `doppel` run gives you a snapshot. Running it repeatedly after each refactoring session creates a compounding effect: merging two functions often unmasks a third pair that was previously hidden behind the noise. Over successive passes you can progressively tighten the threshold and reach a leaner, more consistent codebase.

**The reduction cycle:**

1. Run `doppel analyze` with a conservative threshold (e.g. `--threshold 0.90`) and save the report
2. Work through the top pairs — extract shared logic or consolidate the duplicates
3. Re-run; the embedding and concept-doc caches mean only changed files are re-processed, so each pass is fast
4. Lower the threshold slightly once the high-confidence pairs are gone (`0.90 → 0.85 → 0.80`) to surface the next layer
5. Repeat until the report comes back empty at your chosen floor

**Scheduling it:**

The loop works best when it runs automatically. Commit a `.doppel.json` to the repo with a standing configuration — threshold, output file, optional reflect model — so every run uses consistent settings:

```json
{
  "threshold": 0.85,
  "top": 10,
  "reflect-model": "llama3.2",
  "output": "doppel-report.md"
}
```

Then set up a recurring task (daily, post-merge, or pre-PR) that runs `doppel analyze .` and writes a fresh `doppel-report.md`. Each session you open the report, work through what's there, and commit the reduction. Because the cache persists between runs, revisited functions are instant — only new or changed code gets re-embedded, so the tool never slows you down as the codebase shrinks.

With Claude Code's scheduled-task support you can wire this directly into your workflow: a scheduled run generates the report, and your next coding session opens with a ready-made list of merge candidates to check off.

The goal is a threshold floor where new pairs no longer appear — at that point the codebase has reached its semantic minimum for the chosen embedding model.

### Flags

| Flag | Default | Description |
|---|---|---|
| `-t`, `--threshold` | `0.85` | Minimum cosine similarity score to report (0.0–1.0) |
| `-n`, `--top` | `20` | Maximum number of similar pairs to show |
| `-m`, `--model` | `nomic-embed-text` | Ollama embedding model to use |
| `--ollama-url` | `http://localhost:11434` | Ollama base URL |
| `--cache` | `.embeddings.json` | Embedding cache file (empty string disables caching) |
| `--max-input` | `8192` | Max bytes of each function body sent to the embedder |
| `--ollama-num-ctx` | `0` (server default) | Ollama `options.num_ctx` token limit |
| `--concept-model` | *(disabled)* | Ollama chat model for concept doc generation (e.g. `llama3.2`) |
| `--concept-cache` | `.concepts.json` | Concept doc cache file (empty string disables caching) |
| `--reflect-model` | *(disabled)* | Ollama chat model for merge explanations (e.g. `llama3.2`) |
| `-o`, `--output` | *(disabled)* | Write report as Markdown to this file |

## Embedding Cache

Embeddings are cached to `.embeddings.json` by default. The cache is keyed by a SHA-256 hash of the model name, `num_ctx`, and concept doc text. Re-runs on an unchanged codebase complete instantly without hitting Ollama. Pass `--cache ""` to disable caching.

## Concept Doc Cache

Concept docs are cached to `.concepts.json` by default. The cache is keyed by a SHA-256 hash of the model name and function body text. Pass `--concept-cache ""` to disable caching.

## Skipped Directories

The following directories are automatically skipped:
`.git`, `.claude`, `vendor`, `testdata`, `build`, `.idea`, `.vscode`

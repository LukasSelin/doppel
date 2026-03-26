# How Doppel Works

## Pipeline

1. **Parse** — walks the target directory and extracts all Go function/method bodies, names, signatures, doc comments, visibility, receiver types, and AST-derived callees using the `go/ast` package; non-`.go` files are skipped
2. **Tag** — scans each function body for intent patterns (`retry`, `http_call`, `db_access`, `validation`, `mapping`, `transaction`, `caching`, `concurrency`, `error_wrapping`) using keyword matching
3. **Generate concept docs** — creates a deterministic semantic summary per function (name, package, I/O types, external dependencies, patterns) from static analysis alone — no LLM or cache required
4. **Map** — builds a call graph across all parsed functions and enriches each concept doc with callers, callees, aggregated caller/callee patterns and packages, and a structural role (`leaf`, `utility`, `orchestrator`, or `passthrough`) based on fan-in/fan-out counts
5. **Embed** — sends each concept doc (not the raw function body) to a local [Ollama](https://ollama.com) embedding model to generate a semantic vector; cached in `.embeddings.json`
6. **Find similar** — computes pairwise cosine similarity across all vectors
7. **Structural comparison** — scores each embedding-matched pair across 9 weighted signals (shared callees 25%, patterns 20%, role 15%, callers 15%, package 10%, visibility 5%, receiver type 5%, callee packages 2.5%, caller packages 2.5%) producing a 0.0–1.0 overlap score and a merge-worthiness flag; pairs below `--struct-min` are dropped
8. **Reflect** *(optional)* — uses a chat model (`--reflect-model`) to explain why each pair is similar and how they could be merged; the structural evidence is included in the prompt for better-informed judgments
9. **Report** — prints the most similar pairs above a configurable threshold to stdout and optionally saves a Markdown file

Results are printed to stdout as a plain-text report and optionally saved as a Markdown file.

## Iterative Refactoring Loop

A single `doppel` run gives you a snapshot. Running it repeatedly after each refactoring session creates a compounding effect: merging two functions often unmasks a third pair that was previously hidden behind the noise. Over successive passes you can progressively tighten the threshold and reach a leaner, more consistent codebase.

**The reduction cycle:**

1. Run `doppel analyze` with a conservative threshold (e.g. `--threshold 0.90`) and save the report
2. Work through the top pairs — extract shared logic or consolidate the duplicates
3. Re-run; the embedding cache means only changed files are re-processed, so each pass is fast
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

The goal is a threshold floor where new pairs no longer appear — at that point the codebase has reached its semantic minimum for the chosen embedding model.

## Embedding Cache

Embeddings are cached to `.embeddings.json` by default. The cache is keyed by a SHA-256 hash of the model name, `num_ctx`, and concept doc text. Re-runs on an unchanged codebase complete instantly without hitting Ollama. Pass `--cache ""` to disable caching.

## Skipped Directories

The following directories are automatically skipped:
`.git`, `.claude`, `vendor`, `testdata`, `build`, `.idea`, `.vscode`

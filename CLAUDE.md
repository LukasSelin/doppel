# Doppel

Go CLI tool that detects structural similarities in a Go codebase to surface merge candidates — functions or methods that do similar enough work to warrant consolidation.

## Goal

Spot structural duplication that text-based tools miss. Doppel uses semantic embeddings (via Ollama) and structural analysis to find function pairs that share intent, call patterns, and role — not just string overlap. The output is a ranked list of merge candidates with evidence explaining *why* they are similar.
**Detect**  — Doppel scans a codebase and produces a report of structurally similar function pairs with confidence scores and structural evidence.

## Pipeline

Parse (go/ast) → Tag patterns → Build call graph → Generate concept docs → Embed (Ollama) → Cosine similarity → Structural comparison → Optional LLM reflection → Report

## Module layout

```
cmd/            CLI commands (Cobra). analyze.go is the pipeline orchestrator.
internal/
  parser/       AST extraction of Go functions → CodeUnit
  tagger/       Keyword-based intent pattern detection (retry, http_call, db_access, etc.)
  concepter/    ConceptDoc struct, call graph builder, role classification (leaf/utility/orchestrator/passthrough)
  mapper/       Enriches CodeUnits into ConceptDocs with call graph context
  embedder/     Ollama embedding client with SHA-256 keyed cache
  analyzer/     Pairwise cosine similarity, filtering, sorting
  comparator/   Weighted structural overlap scoring (9 signals, 0.0–1.0 composite)
  reflector/    LLM-based merge rationale generation via Ollama chat
  reporter/     Plain-text and Markdown output formatting
```

## Development

```bash
go build -o doppel .          # build
go test ./...                 # test
go vet ./...                  # lint
doppel analyze .              # run against this repo
doppel analyze . --reflect-model llama3.2 --output report.md  # full run with LLM explanations
```

Requires Ollama running locally (`http://localhost:11434`) with an embedding model pulled (default: `nomic-embed-text`).

## Key types

- **CodeUnit** (`internal/parser/parser.go`) — a single function/method extracted from AST: name, body, signature, callees, patterns, metadata.
- **ConceptDoc** (`internal/concepter/concepter.go`) — semantic representation of a CodeUnit enriched with call graph context, role, and aggregated patterns. This is what gets embedded.
- **SimilarPair** (`internal/analyzer/similarity.go`) — two CodeUnits with their cosine similarity score, structural evidence, and optional LLM explanation.
- **StructuralEvidence** (`internal/comparator/comparator.go`) — weighted overlap scoring across 9 signals (shared callees 25%, patterns 20%, callers 15%, role 15%, package 10%, etc.).

## Conventions

- Go-only. All parsing uses `go/ast` — no external parsers or multi-language support.
- Caches are JSON files (`.embeddings.json`, `.concepts.json`) keyed by SHA-256 hashes. They are gitignored artifacts.
- Skipped directories: `.git`, `.claude`, `vendor`, `testdata`, `build`, `.idea`, `.vscode`.
- Config via `.doppel.json` at repo root (all CLI flags as snake_case keys). CLI flags override config.

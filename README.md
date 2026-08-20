# doppel

A CLI tool that detects semantically similar functions across a Go codebase using local AI embeddings. It helps identify duplicate logic and refactoring opportunities by comparing structural concept documents — call graph context, intent patterns and role — with vector similarity rather than text matching.

For a detailed breakdown of the pipeline internals, see [How Doppel Works](.github/wiki/how-it-works.md).

## Quick Start

**Prerequisites:**

- [Go 1.25+](https://go.dev/dl/)
- [Ollama](https://ollama.com) running locally

**Pull the models and run:**

```bash
ollama pull qwen3-embedding:8b
ollama pull llama3.2

go run . analyze . --model qwen3-embedding:8b --reflect-model llama3.2 --output report.md
```

This scans the current directory, embeds every Go function, finds structurally similar pairs, uses `llama3.2` to explain why they match, and writes the results to `report.md`.

## Installation

Build from source:

```bash
git clone https://github.com/LukasSelin/doppel
cd doppel
go build -o doppel .
```

> `go install` is not yet supported: `go.mod` declares the module as
> `github.com/lukse/doppel`, which does not resolve to this repository. Enabling it
> requires renaming the module path to `github.com/LukasSelin/doppel`.

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

# Full run with LLM explanations and structural filtering
doppel analyze . --reflect-model llama3.2 --struct-min 0.4 --output report.md
```

### Flags

| Flag                | Default                  | Description                                                                         |
| ------------------- | ------------------------ | ----------------------------------------------------------------------------------- |
| `-t`, `--threshold` | `0.85`                   | Minimum cosine similarity score to report (0.0–1.0)                                 |
| `-n`, `--top`       | `20`                     | Maximum number of similar pairs to show                                             |
| `-m`, `--model`     | `nomic-embed-text`       | Ollama embedding model to use                                                       |
| `--ollama-url`      | `http://localhost:11434` | Ollama base URL                                                                     |
| `--cache`           | `.embeddings.json`       | Embedding cache file (empty string disables caching)                                |
| `--max-input`       | `8192`                   | Max UTF-8 bytes of each concept doc sent to the embedder (auto-shrinks on context errors) |
| `--ollama-num-ctx`  | `0` (server default)     | Ollama `options.num_ctx` token limit                                                |
| `--struct-min`      | `0.0`                    | Minimum structural overlap score (0.0–1.0) to keep a pair after embedding selection |
| `--reflect-model`   | *(disabled)*             | Ollama chat model for merge explanations (e.g. `llama3.2`)                          |
| `-o`, `--output`    | *(disabled)*             | Write report as Markdown to this file                                               |
| `--reflect-prompt-file` | *(built-in prompt)*  | `text/template` file overriding the reflect prompt (`{{.Score}}`, `{{.A.Name}}`, `{{.A.Body}}`, `{{.Evidence}}`, …) |
| `--config`          | `.doppel.json` if present | Path to a JSON config file                                                         |
| `--concept-model`   | *(inert)*                | Registered but currently unused — leftover from the removed LLM concepter           |
| `--concept-prompt-file` | *(inert)*            | Registered but currently unused — leftover from the removed LLM concepter           |

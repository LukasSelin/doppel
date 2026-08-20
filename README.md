# doppel

A CLI tool that detects structurally similar functions across a Go codebase. It helps identify duplicate logic and refactoring opportunities by fingerprinting each function from its AST and cross-checking matches against call-graph context — rather than by text matching.

Everything runs locally and offline: no models, no network, no cache. The same source always produces the same report.

For a detailed breakdown of the pipeline internals, see [How Doppel Works](.github/wiki/how-it-works.md).

## Quick Start

**Prerequisites:** [Go 1.25+](https://go.dev/dl/)

```bash
go run . analyze .
```

This scans the current directory, fingerprints every Go function, and prints the most similar pairs with the evidence behind each match. Add `--output report.md` to also write a Markdown report.

## Installation

Build from source:

```bash
git clone https://github.com/LukasSelin/doppel
cd doppel
go build -o doppel .
```

Or install directly:

```bash
go install github.com/LukasSelin/doppel@latest
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
doppel analyze ./src --threshold 0.50

# Keep only pairs that also share architectural context, and save a report
doppel analyze . --struct-min 0.4 --output report.md
```

### Two scores per pair

Every reported pair carries two independent numbers:

- **Code similarity** (`Score`, gated by `--threshold`) — how alike the two bodies are, from the AST fingerprint. The report breaks it into its components: `ast` (3-gram shingle overlap), `flow` (control-flow shape), `sig` (parameter and result types), and `size` (relative body size, shown for context but not scored).
- **Structural overlap** (gated by `--struct-min`) — how much architectural context the two share: callees, callers, intent patterns, role, package.

A high code score with low structural overlap means two lookalike bodies in unrelated parts of the system. High on both is the real merge candidate.

### Flags

| Flag                | Default | Description                                                                 |
| ------------------- | ------- | --------------------------------------------------------------------------- |
| `-t`, `--threshold` | `0.60`  | Minimum code similarity score to report (0.0–1.0)                           |
| `-n`, `--top`       | `20`    | Maximum number of pairs to show (`0` for no limit)                          |
| `--struct-min`      | `0.0`   | Minimum structural overlap score (0.0–1.0) to keep a pair                   |
| `--min-nodes`       | `12`    | Skip functions whose body has fewer than this many AST nodes. Guards against one-line accessors, which match each other perfectly and would otherwise flood the report |
| `-o`, `--output`    | *(disabled)* | Write report as Markdown to this file. The stdout report is still printed |
| `--config`          | `.doppel.json` if present | Path to a JSON config file                                |

### Configuration

Any flag above except `--config` can be set in a `.doppel.json` at the repo root. Keys are kebab-case, mirroring the flag names, and an explicit CLI flag always wins over the file:

```json
{
  "threshold": 0.65,
  "top": 10,
  "struct-min": 0.4,
  "output": "doppel-report.md"
}
```

A missing config file is not an error; malformed JSON is.

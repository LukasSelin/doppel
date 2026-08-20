# How Doppel Works

Doppel is fully self-contained: it parses Go with `go/ast`, scores every function pair from static data, and prints a report. There is no model, no network call, and no cache anywhere in the pipeline, so a given tree always yields the same output.

## Pipeline

1. **Parse** — walks the target directory and extracts all Go function/method bodies, names, signatures, doc comments, visibility, receiver types, and AST-derived callees using the `go/ast` package; non-`.go` files are skipped
2. **Fingerprint** — while the AST is still in hand, summarises each body into a `Fingerprint`: hashed 3-grams over a canonicalized AST token stream, a control-flow histogram, the normalized parameter/result types, and a node count
3. **Tag** — scans each function body for intent patterns (`retry`, `http_call`, `db_access`, `validation`, `mapping`, `transaction`, `caching`, `concurrency`, `error_wrapping`) using keyword matching
4. **Generate concept docs** — creates a deterministic architectural summary per function (name, package, visibility, receiver, patterns) from static analysis alone
5. **Map** — builds a call graph across all parsed functions and enriches each concept doc with callers, callees, aggregated caller/callee patterns and packages, and a structural role (`leaf`, `utility`, `orchestrator`, or `passthrough`) based on fan-in/fan-out counts
6. **Find similar** — compares every pair of fingerprints, keeps those at or above `--threshold`, sorts descending and truncates to `--top`. Functions with fewer than `--min-nodes` AST nodes are excluded from comparison entirely
7. **Structural comparison** — scores each matched pair across 9 weighted signals (shared callees 25%, patterns 20%, role 15%, callers 15%, package 10%, visibility 5%, receiver type 5%, callee packages 2.5%, caller packages 2.5%) producing a 0.0–1.0 overlap score and a merge-worthiness flag; pairs below `--struct-min` are dropped
8. **Report** — prints the surviving pairs to stdout and optionally saves a Markdown file

## The fingerprint

Step 2 is what replaces text matching. Each body is walked in pre-order and reduced to a stream of short tokens: statement and expression kinds (`IF`, `RANGE`, `RETURN`, `DEFER`), operators (`BIN:+`, `ASSIGN::=`), literal kinds (`LIT:STRING`), and call targets (`CALL:Errorf`).

Two rules matter:

- **Identifiers collapse to `ID`.** A copy of a function with every variable renamed produces exactly the same token stream, so renamed clones score as clones.
- **Call selector names survive.** `Errorf`, `Query` and `Lock` carry real intent, while the receiver variable they are called on (`e`, `s`, `cfg`) is arbitrary and is discarded.

Sliding windows of 3 tokens are hashed, deduplicated and sorted; two functions are compared by Jaccard overlap of those shingle sets, blended with control-flow cosine similarity and signature-type overlap:

| Component | Metric | Weight |
| --- | --- | --- |
| AST shingles | Jaccard | 0.60 |
| Control flow | cosine over the node-kind histogram | 0.25 |
| Signature | Jaccard over normalized param/result types | 0.15 |

The relative body size is reported alongside these but is not scored — Jaccard already penalizes size mismatch through the union.

## Two scores, kept separate

A pair carries a **code similarity** score (step 6, gated by `--threshold`) and a **structural overlap** score (step 7, gated by `--struct-min`). They are deliberately not blended: a high code score with low structural overlap means two lookalike bodies sitting in unrelated parts of the system, which is a different finding from two functions that both look alike *and* share callers, callees and role. Tighten `--struct-min` when you only want the latter.

## Iterative Refactoring Loop

A single `doppel` run gives you a snapshot. Running it repeatedly after each refactoring session creates a compounding effect: merging two functions often unmasks a third pair that was previously hidden behind the noise. Over successive passes you can progressively tighten the threshold and reach a leaner, more consistent codebase.

**The reduction cycle:**

1. Run `doppel analyze` with a conservative threshold (e.g. `--threshold 0.75`) and save the report
2. Work through the top pairs — extract shared logic or consolidate the duplicates
3. Re-run; analysis is pure local computation, so a pass costs seconds
4. Lower the threshold slightly once the high-confidence pairs are gone (`0.75 → 0.65 → 0.55`) to surface the next layer
5. Repeat until the report comes back empty at your chosen floor

**Scheduling it:**

The loop works best when it runs automatically. Commit a `.doppel.json` to the repo with a standing configuration so every run uses consistent settings:

```json
{
  "threshold": 0.65,
  "top": 10,
  "struct-min": 0.4,
  "output": "doppel-report.md"
}
```

Then set up a recurring task (daily, post-merge, or pre-PR) that runs `doppel analyze .` and writes a fresh `doppel-report.md`. Because the scoring is deterministic, an unchanged tree produces a byte-identical report — so any diff in the committed report is a real change in the shape of the codebase, which makes it reviewable in a PR.

## Skipped Directories

The following directories are automatically skipped:
`.git`, `.claude`, `vendor`, `testdata`, `build`, `.idea`, `.vscode`

Note that `_test.go` files are **not** skipped. Test functions are repetitive by nature and will legitimately dominate the top of the report on a well-tested repo; point `doppel analyze` at a subdirectory if you want production code only.

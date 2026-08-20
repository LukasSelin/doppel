# How Doppel Works

Doppel is fully self-contained: it parses Go with `go/ast`, scores every function pair from static data, and prints a report. There is no model, no network call, and no cache anywhere in the pipeline, so a given tree always yields the same output.

## Pipeline

1. **Parse** — walks the target directory and extracts all Go function/method bodies, names, signatures, doc comments, visibility, receiver types, and AST-derived callees using the `go/ast` package; non-`.go` files are skipped
2. **Fingerprint** — while the AST is still in hand, summarises each body into a `Fingerprint`: hashed 3-grams over a canonicalized AST token stream, a control-flow histogram, the normalized parameter/result types, and a node count
3. **Tag** — detects intent patterns (`retry`, `http_call`, `db_access`, `validation`, `mapping`, `transaction`, `caching`, `concurrency`, `error_wrapping`) from AST evidence: call selectors, imports, string-literal contents, identifier names, and node kinds. Comments are not evidence, so SQL in a comment tags nothing. Each tag is a leaf of the concept taxonomy described below, which is what lets two different tags still score against each other. Tag frequencies over the whole corpus then weight concept matching in step 7 — sharing a near-universal tag is weak evidence, sharing a rare one is strong
4. **Generate concept docs** — creates a deterministic architectural summary per function (name, package, visibility, receiver, patterns) from static analysis alone
5. **Map** — builds a resolved, repo-internal call graph (qualified names; import-aware; method calls resolved when unambiguous; no stdlib noise) and enriches each concept doc with callers, resolved callees, the depth-2 call-graph neighborhood, aggregated caller/callee patterns and packages, and a structural role (`leaf`, `utility`, `orchestrator`, or `passthrough`). Role thresholds adapt to the repo: high fan-in/fan-out means above this corpus's median resolved degree, floored at 2. The role is really two independent booleans, which is why two different roles can still partly agree
6. **Find similar** — compares every pair of fingerprints, keeps those at or above `--threshold`, sorts descending and truncates to `--top`. Functions with fewer than `--min-nodes` AST nodes are excluded from comparison entirely
7. **Structural comparison** — scores each matched pair across 12 weighted signals (shared callees 21%, concepts 18%, role 13.5%, callers 12%, package 9%, caller concepts 5%, callee concepts 5%, visibility 4.5%, receiver type 4.5%, neighborhood overlap 3%, caller packages 2.25%, callee packages 2.25%) producing a 0.0–1.0 overlap score and a merge-worthiness flag; pairs below `--struct-min` are dropped. Concepts, roles and receiver types are matched through the ontology rather than by string equality, so related-but-not-identical work earns partial credit, and two functions sitting in overlapping call-graph neighborhoods share context even when their direct edges differ
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

## The ontology

Step 7 used to compare strings. Two functions tagged `http_call` and `db_access` scored **zero** on
intent — exactly the same as two functions with nothing in common — even though both are I/O. Roles
were just as binary: `utility` and `passthrough` scored zero against each other despite both being
called from many places.

The nine tags are now leaves of a small taxonomy. Everything above them is abstract: those interior
nodes never describe a real function, they exist only to relate the leaves.

```
concept
├── io_operation
│   ├── remote_io → http_call
│   └── data_store_access → db_access, caching, transaction
├── data_transformation → mapping, validation
├── control_flow
│   ├── concurrency
│   └── fault_tolerance → retry
└── error_handling → error_wrapping
```

Two concepts are scored by how deep their nearest shared ancestor sits:

| Pair | Nearest shared ancestor | Score |
| --- | --- | --- |
| `db_access` / `db_access` | itself | 1.00 |
| `db_access` / `caching` | `data_store_access` | 0.67 |
| `http_call` / `db_access` | `io_operation` | 0.33 |
| `http_call` / `retry` | the root | 0.00 |

Roles work the same way on two axes. `utility` and `passthrough` are both high fan-in, so they score
0.5; `leaf` and `orchestrator` share only the *absence* of an axis, which scores nothing. And a pair
of methods on different receiver types now scores 0.5 rather than 0, while a value receiver and a
pointer receiver on the same type finally score 1.0.

None of this can raise a pair above what an exact match would give, and every graded match that
contributes to a score also produces an evidence line, so an elevated score always has a stated
reason:

```
related patterns: db_access ≈ caching (both data_store_access, 0.67)
related roles: utility ≈ passthrough (both high fan-in, 0.50)
callees do related work (1.00): [error_wrapping, mapping]
```

A near match nudges the score but only counts as a *merge signal* once it reaches 0.5, so a pair of
distant cousins cannot be flagged merge-worthy on that evidence alone.

Run `doppel ontology --defs` to print the whole vocabulary with definitions and check it against its
own consistency rules. Those rules are also enforced by the test suite, which is what stops the
tagger and the taxonomy from drifting apart.

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

# conc

structured concurrency library; generics-heavy, one idea, written recently and at once

**What this rung shows:** the small-corpus floor: 85 functions, where IC and df caps have almost nothing to work with

| | |
|---|---|
| Corpus | [conc](https://github.com/sourcegraph/conc) |
| Pinned at | `v0.3.0` (`7b8c8f2875cb861bb61844c9bcaa1aed070adbd4`) |
| Project since | 2023 |
| doppel | `bb0f86a` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 1 concepts modeled, 0 associations, 0 unusual realizations
Habitats: 4 modeled, 16 misfits; most uniform panics (norm 0.94), most diverse pool (norm 0.60)
Conventions: strongest concurrency (0.37), loosest concurrency (0.37)
Ecosystems: 8 profiled (8 dominance, 0 coalition, 0 conflict, 0 weak)
Calibration: rate 0.01 over 1830 shape / 3240 overlap null pairs -> threshold 0.85, struct-min 0.51, family-min 0.85
Found 81 functions. Retrieving candidates...
Retrieval: shape 22, concept 25, call 15 -> 60 unique pairs
  concept-only 40.0%  call-only 21.7%  suppressed-shape functions: 0  large identity buckets: 0  surviving patterns: 448
Running structural comparison on 60 pairs...
  5 pairs remain after struct-min=0.51 filter
```

# Code Similarity Report

**Functions analyzed:** 81 | **Threshold:** 0.60 | **Pairs found:** 5

---

## What doppel sees

**81 functions** across **5 packages** — test functions excluded. Structural roles: 72 leaf, 2 orchestrator, 7 utility.

### Concepts

doppel reads intent from the AST into a fixed vocabulary and reasons over the tree, so two functions that share a *branch* score partial credit rather than nothing. Leaf counts below are this corpus.

```mermaid
flowchart LR
    c0(["concept"])
    c1(["io_operation"])
    c2(["remote_io"])
    c3["http_call<br/>absent"]
    c4["grpc_call<br/>absent"]
    c5(["data_store_access"])
    c6["db_access<br/>absent"]
    c7["caching<br/>absent"]
    c8["transaction<br/>absent"]
    c9["file_io<br/>absent"]
    c10["logging<br/>absent"]
    c11(["data_transformation"])
    c12["mapping<br/>absent"]
    c13["validation<br/>absent"]
    c14["serialization<br/>absent"]
    c15(["control_flow"])
    c16["concurrency<br/>8"]
    c17(["fault_tolerance"])
    c18["retry<br/>absent"]
    c19["circuit_breaker<br/>absent"]
    c20(["error_handling"])
    c21["error_wrapping<br/>absent"]
    c0 --> c1
    c1 --> c2
    c2 --> c3
    c2 --> c4
    c1 --> c5
    c5 --> c6
    c5 --> c7
    c5 --> c8
    c1 --> c9
    c1 --> c10
    c0 --> c11
    c11 --> c12
    c11 --> c13
    c11 --> c14
    c0 --> c15
    c15 --> c16
    c15 --> c17
    c17 --> c18
    c17 --> c19
    c0 --> c20
    c20 --> c21
    classDef good fill:#d7ecd9,color:#1b3d20
    classDef warn fill:#fbeecb,color:#4a3a12
    classDef hot fill:#f7d6d6,color:#4a1c1c
    class c3,c4,c6,c7,c8,c9,c10,c12,c13,c14,c18,c19,c21 hot
```

**Nothing here is tagged** `caching`, `circuit_breaker`, `db_access`, `error_wrapping`, `file_io`, `grpc_call`, `http_call`, `logging`, `mapping`, `retry`, `serialization`, `transaction`, `validation`. That is a direct answer to "does this codebase already do X" — for those concepts, it does not.

| Concept | Functions | Convention |
|---|---:|---|
| `concurrency` | 8 | `0.37` (loose) |

Convention is how uniformly this corpus realizes a concept: `1.00` means every function carrying the tag does it the same way, and a low number means the tag covers several unrelated habits. A concept with fewer than five members is not modeled.

### Where the duplication is

Merge-worthy pairs folded up to their packages. An edge means two packages keep solving the same problem separately; a count on a node means the repetition is inside one package.

### How settled each package is

A package with at least five functions gets a habitat model: doppel learns what is normal there and measures how surprising each member is against it. **Norm** is how uniform the package's practice is. A **misfit** is a function alien to its package *and* to the wider subsystem around it — one that fits its neighbours a directory up is normal for this codebase and is not reported.

```mermaid
flowchart TD
    h0["pool<br/>50 functions · norm 0.60<br/>15 misfits"]
    h1["stream<br/>8 functions · norm 0.79<br/>1 misfit"]
    h2["iter<br/>9 functions · norm 0.93"]
    h3["panics<br/>10 functions · norm 0.94"]
    classDef good fill:#d7ecd9,color:#1b3d20
    classDef warn fill:#fbeecb,color:#4a3a12
    classDef hot fill:#f7d6d6,color:#4a1c1c
    class h1,h2,h3 good
    class h0 warn
```

Most uniform is `panics` (norm `0.94`); most varied is `pool` (norm `0.60`). 16 functions are alien to their package and to the subsystem around it.

### How these candidates were found

Three channels propose candidates independently — shared rare *structure*, shared *concepts*, shared *calls* — and their union is what gets compared. This run: **60 candidate pairs** (shape 22, concept 25, call 15), of which 22% arrived on call evidence alone and 40% on concept evidence alone. A pair sharing none of the three is never compared, however alike it looks.

Each function is also an arena where its candidate concepts compete for its evidence. 8 functions reached an equilibrium: **8** settled on a single concept, **0** on a coalition, **0** hold concepts this corpus says do not go together.

---

## Local practice

The vocabulary above says what a concept *is*. This says what one looks like when **this** codebase writes it — learned from the corpus, so it describes the house style rather than a rule from anywhere else.

### How this codebase writes each concept

Only what is **distinctive**. A feature earns a row by being carried by this concept's members at least twice as often as by the corpus at large — nearly every Go function has a `return` and an `if`, so prevalence alone would describe the language rather than this codebase. Weights are how much a channel counts toward whether a member looks normal — calls 40, control flow 20, co-occurring tags 15, role 15, package 10.

**`concurrency`** — 8 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `github.com/sourcegraph/conc/internal/multierror.Join` | `███·······` | 2 of 8 | 10× |
| flow ×20 | `funclit` | `██████····` | 5 of 8 | 3.6× |
|  | `if` | `█████·····` | 4 of 8 | 2.7× |
| role ×15 | `utility` | `███·······` | 2 of 8 | 2.9× |
| package ×10 | `iter` | `███·······` | 2 of 8 | 2.2× |

---

## Match #1 — Code-shape: `0.7259`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/result_context_pool.go:22` | `pool.*ResultContextPool[T].Go` | `(func(context.Context) (T, error))` | — |
| **B** | `pool/result_error_pool.go:25` | `pool.*ResultErrorPool[T].Go` | `(func() (T, error))` | — |

**Code similarity:** `ast 0.79  flow 1.00  nesting 1.00  sig 0.00  size 0.87`

**Evidence:** `146.48` (shape 143.19, concept 0.00, call 3.30)

**Trophic:** `0.94`

**Shared structure:**

- `3.42` — `seq[ assign:=(call:f) ; if(bin:\|\|(bin,sel)) ]`
- `3.42` — `seq[ if(bin:\|\|(bin,sel)) ; return(id) ]`
- `3.42` — `if(bin:\|\|(bin,sel))`

**Habitat:** A fits poorly in `pool` (fit 0.01, package norm 0.60)

**Habitat:** B fits poorly in `pool` (fit 0.01, package norm 0.60)

**Structural overlap:** `0.60` (merge-worthy)

- share 3 callees: [Go, add, f]
- overlapping call-graph neighborhoods (1.00): 2 shared
- both are leaf functions
- same package
- callees do related work (1.00): [concurrency]
- same visibility
- both are methods, on *ResultContextPool[T] and *ResultErrorPool[T]
- call into same packages: [pool]

---

## Match #2 — Code-shape: `0.2690`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/error_pool.go:85` | `pool.*ErrorPool.addErr` | `(error)` | concurrency |
| **B** | `pool/result_pool.go:89` | `pool.*resultAggregator[T].add` | `(T)` | concurrency |

**Profile A:** `concurrency` 1.00 (dominance)

**Profile B:** `concurrency` 1.00 (dominance)

**Code similarity:** `ast 0.45  flow 0.00  nesting 0.00  sig 0.00  size 0.50`

**Evidence:** `79.14` (shape 78.24, concept 0.89, call 0.00)

**Trophic:** `0.54`

**Shared structure:**

- `3.01` — `do(call:Lock)`
- `3.01` — `do(call:Unlock)`

**Habitat:** A fits poorly in `pool` (fit 0.00, package norm 0.60)

**Habitat:** B fits poorly in `pool` (fit 0.02, package norm 0.60)

**Structural overlap:** `0.63` (not merge-worthy)

- share 2 callees: [Lock, Unlock]
- share patterns: [concurrency]
- both are utility functions
- same package
- same visibility
- both are methods, on *ErrorPool and *resultAggregator[T]
- called from same packages: [pool]

---

## Match #3 — Code-shape: `0.3823`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/result_error_pool.go:25` | `pool.*ResultErrorPool[T].Go` | `(func() (T, error))` | — |
| **B** | `pool/result_pool.go:32` | `pool.*ResultPool[T].Go` | `(func() T)` | — |

**Code similarity:** `ast 0.41  flow 0.58  nesting 0.45  sig 0.00  size 0.54`

**Evidence:** `46.90` (shape 43.60, concept 0.00, call 3.30)

**Trophic:** `0.44`

**Shared structure:**

- `3.01` — `do(call:add)`
- `1.91` — `do(call:Go)`

**Habitat:** A fits poorly in `pool` (fit 0.01, package norm 0.60)

**Habitat:** B fits poorly in `pool` (fit 0.01, package norm 0.60)

**Structural overlap:** `0.60` (not merge-worthy)

- share 3 callees: [Go, add, f]
- overlapping call-graph neighborhoods (1.00): 2 shared
- both are leaf functions
- same package
- callees do related work (1.00): [concurrency]
- same visibility
- both are methods, on *ResultErrorPool[T] and *ResultPool[T]
- call into same packages: [pool]

---

## Match #4 — Code-shape: `0.3654`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/result_context_pool.go:22` | `pool.*ResultContextPool[T].Go` | `(func(context.Context) (T, error))` | — |
| **B** | `pool/result_pool.go:32` | `pool.*ResultPool[T].Go` | `(func() T)` | — |

**Code similarity:** `ast 0.38  flow 0.58  nesting 0.45  sig 0.00  size 0.47`

**Evidence:** `46.90` (shape 43.60, concept 0.00, call 3.30)

**Trophic:** `0.40`

**Shared structure:**

- `3.01` — `do(call:add)`
- `1.91` — `do(call:Go)`

**Habitat:** A fits poorly in `pool` (fit 0.01, package norm 0.60)

**Habitat:** B fits poorly in `pool` (fit 0.01, package norm 0.60)

**Structural overlap:** `0.60` (not merge-worthy)

- share 3 callees: [Go, add, f]
- overlapping call-graph neighborhoods (1.00): 2 shared
- both are leaf functions
- same package
- callees do related work (1.00): [concurrency]
- same visibility
- both are methods, on *ResultContextPool[T] and *ResultPool[T]
- call into same packages: [pool]

---

## Match #5 — Code-shape: `0.2852`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `iter/iter.go:59` | `iter.Iterator[T].ForEachIdx` | `([]T, func(int, *T))` | concurrency |
| **B** | `iter/map.go:48` | `iter.Mapper[T, R].MapErr` | `([]T, func(*T) (R, error)) ([]R, error)` | concurrency |

**Profile A:** `concurrency` 1.00 (dominance)

**Profile B:** `concurrency` 1.00 (dominance)

**Code similarity:** `ast 0.15  flow 0.58  nesting 0.98  sig 0.20  size 0.73`

**Evidence:** `53.58` (shape 52.68, concept 0.89, call 0.00)

**Trophic:** `0.23`

**Shared structure:**

- `3.01` — `flow:param→call:len`

**Structural overlap:** `0.55` (not merge-worthy)

- share 2 callees: [f, len]
- share patterns: [concurrency]
- both are leaf functions
- same package
- same visibility
- both are methods, on Iterator[T] and Mapper[T, R]
- call into same packages: [iter]

---


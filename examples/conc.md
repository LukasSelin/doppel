# conc

structured concurrency library; generics-heavy, one idea, written recently and at once

**What this rung shows:** the small-corpus floor: 85 functions, where IC and df caps have almost nothing to work with

| | |
|---|---|
| Corpus | [conc](https://github.com/sourcegraph/conc) |
| Pinned at | `v0.3.0` (`7b8c8f2875cb861bb61844c9bcaa1aed070adbd4`) |
| Project since | 2023 |
| doppel | `e61ea20` |
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
Found 81 functions. Retrieving candidates...
Retrieval: shape 43, concept 25, call 15 -> 79 unique pairs
  concept-only 29.1%  call-only 15.2%  suppressed-shape functions: 0  large identity buckets: 0  surviving patterns: 434
Running structural comparison on 79 pairs...
```

# Code Similarity Report

**Functions analyzed:** 81 | **Threshold:** 0.60 | **Pairs found:** 10

---

## Match #1 — Code-shape: `0.7259`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/result_context_pool.go:22` | `pool.*ResultContextPool[T].Go` | `—` | — |
| **B** | `pool/result_error_pool.go:25` | `pool.*ResultErrorPool[T].Go` | `—` | — |

**Code similarity:** `ast 0.79  flow 1.00  nesting 1.00  sig 0.00  size 0.87`

**Evidence:** `137.33` (shape 134.03, concept 0.00, call 3.30)

**Trophic:** `0.93`

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

## Match #2 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/result_error_pool.go:55` | `pool.*ResultErrorPool[T].WithContext` | ` ` | — |
| **B** | `pool/result_pool.go:63` | `pool.*ResultPool[T].WithContext` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `90.71` (shape 90.71, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `2.72` — `seq[ do(call:panicIfInitialized) ; return(unary) ]`
- `1.91` — `return(unary)`
- `1.07` — `do(call:panicIfInitialized)`

**Structural overlap:** `0.50` (merge-worthy)

- share 2 callees: [WithContext, p.panicIfInitialized]
- both are leaf functions
- same package
- same visibility
- both are methods, on *ResultErrorPool[T] and *ResultPool[T]

---

## Match #3 — Code-shape: `0.8889`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/error_pool.go:45` | `pool.*ErrorPool.WithContext` | ` ` | — |
| **B** | `pool/pool.go:138` | `pool.*Pool.WithContext` | ` ` | — |

**Code similarity:** `ast 0.81  flow 1.00  nesting 1.00  sig 1.00  size 0.91`

**Evidence:** `141.91` (shape 138.21, concept 0.00, call 3.70)

**Trophic:** `0.92`

**Shared structure:**

- `3.42` — `seq[ assign:=(call:WithCancel) ; return(unary) ]`
- `3.42` — `seq[ do(call:panicIfInitialized) ; assign:=(call:WithCancel) ]`
- `3.42` — `assign:=(call:WithCancel)`

**Habitat:** A fits poorly in `pool` (fit 0.01, package norm 0.60)

**Habitat:** B fits poorly in `pool` (fit 0.01, package norm 0.60)

**Structural overlap:** `0.40` (not merge-worthy)

- share 2 callees: [context.WithCancel, p.panicIfInitialized]
- both are leaf functions
- same package
- same visibility
- both are methods, on *ErrorPool and *Pool

---

## Match #4 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/result_context_pool.go:34` | `pool.*ResultContextPool[T].Wait` | ` ` | — |
| **B** | `pool/result_error_pool.go:37` | `pool.*ResultErrorPool[T].Wait` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `52.86` (shape 52.86, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.42` — `seq[ assign:=(call:Wait) ; return(sel,id) ]`
- `3.42` — `assign:=(call:Wait)`
- `3.42` — `return(sel,id)`

**Structural overlap:** `0.50` (merge-worthy)

- share 1 callees: [Wait]
- both are leaf functions
- same package
- same visibility
- both are methods, on *ResultContextPool[T] and *ResultErrorPool[T]

---

## Match #5 — Code-shape: `0.5864`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `iter/map.go:27` | `iter.Mapper[T, R].Map` | ` ` | — |
| **B** | `iter/map.go:48` | `iter.Mapper[T, R].MapErr` | ` ` | concurrency |

**Profile B:** `concurrency` 1.00 (dominance)

**Code similarity:** `ast 0.53  flow 0.82  nesting 0.89  sig 0.40  size 0.50`

**Evidence:** `158.79` (shape 155.09, concept 0.00, call 3.70)

**Trophic:** `0.71`

**Shared structure:**

- `3.42` — `assign=(call:f)`
- `3.01` — `do(call:ForEachIdx)`

**Structural overlap:** `0.49` (merge-worthy)

- share 4 callees: [ForEachIdx, f, len, make]
- overlapping call-graph neighborhoods (1.00): 1 shared
- both are leaf functions
- same package
- same visibility
- same receiver type: Mapper[T, R]
- call into same packages: [iter]

---

## Match #6 — Code-shape: `0.8500`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/context_pool.go:64` | `pool.*ContextPool.WithFirstError` | ` ` | — |
| **B** | `pool/result_context_pool.go:50` | `pool.*ResultContextPool[T].WithFirstError` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 0.00  size 1.00`

**Evidence:** `50.32` (shape 50.32, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.01` — `seq[ do(call:WithFirstError) ; return(id) ]`
- `3.01` — `seq[ do(call:panicIfInitialized) ; do(call:WithFirstError) ]`
- `3.01` — `do(call:WithFirstError)`

**Structural overlap:** `0.50` (merge-worthy)

- share 2 callees: [WithFirstError, p.panicIfInitialized]
- both are leaf functions
- same package
- same visibility
- both are methods, on *ContextPool and *ResultContextPool[T]

---

## Match #7 — Code-shape: `0.8500`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/context_pool.go:64` | `pool.*ContextPool.WithFirstError` | ` ` | — |
| **B** | `pool/result_error_pool.go:64` | `pool.*ResultErrorPool[T].WithFirstError` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 0.00  size 1.00`

**Evidence:** `50.32` (shape 50.32, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.01` — `seq[ do(call:WithFirstError) ; return(id) ]`
- `3.01` — `seq[ do(call:panicIfInitialized) ; do(call:WithFirstError) ]`
- `3.01` — `do(call:WithFirstError)`

**Structural overlap:** `0.50` (merge-worthy)

- share 2 callees: [WithFirstError, p.panicIfInitialized]
- both are leaf functions
- same package
- same visibility
- both are methods, on *ContextPool and *ResultErrorPool[T]

---

## Match #8 — Code-shape: `0.8500`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/result_context_pool.go:50` | `pool.*ResultContextPool[T].WithFirstError` | ` ` | — |
| **B** | `pool/result_error_pool.go:64` | `pool.*ResultErrorPool[T].WithFirstError` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 0.00  size 1.00`

**Evidence:** `50.32` (shape 50.32, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.01` — `seq[ do(call:WithFirstError) ; return(id) ]`
- `3.01` — `seq[ do(call:panicIfInitialized) ; do(call:WithFirstError) ]`
- `3.01` — `do(call:WithFirstError)`

**Structural overlap:** `0.50` (merge-worthy)

- share 2 callees: [WithFirstError, p.panicIfInitialized]
- both are leaf functions
- same package
- same visibility
- both are methods, on *ResultContextPool[T] and *ResultErrorPool[T]

---

## Match #9 — Code-shape: `0.9000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/context_pool.go:86` | `pool.*ContextPool.WithMaxGoroutines` | ` ` | — |
| **B** | `pool/error_pool.go:65` | `pool.*ErrorPool.WithMaxGoroutines` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 0.33  size 1.00`

**Evidence:** `46.22` (shape 46.22, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `2.50` — `seq[ do(call:WithMaxGoroutines) ; return(id) ]`
- `2.50` — `seq[ do(call:panicIfInitialized) ; do(call:WithMaxGoroutines) ]`
- `2.50` — `do(call:WithMaxGoroutines)`

**Structural overlap:** `0.50` (merge-worthy)

- share 2 callees: [WithMaxGoroutines, p.panicIfInitialized]
- both are leaf functions
- same package
- same visibility
- both are methods, on *ContextPool and *ErrorPool

---

## Match #10 — Code-shape: `0.9000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `pool/context_pool.go:86` | `pool.*ContextPool.WithMaxGoroutines` | ` ` | — |
| **B** | `pool/result_context_pool.go:67` | `pool.*ResultContextPool[T].WithMaxGoroutines` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 0.33  size 1.00`

**Evidence:** `46.22` (shape 46.22, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `2.50` — `seq[ do(call:WithMaxGoroutines) ; return(id) ]`
- `2.50` — `seq[ do(call:panicIfInitialized) ; do(call:WithMaxGoroutines) ]`
- `2.50` — `do(call:WithMaxGoroutines)`

**Structural overlap:** `0.50` (merge-worthy)

- share 2 callees: [WithMaxGoroutines, p.panicIfInitialized]
- both are leaf functions
- same package
- same visibility
- both are methods, on *ContextPool and *ResultContextPool[T]

---


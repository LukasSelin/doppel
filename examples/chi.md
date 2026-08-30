# chi

HTTP router; a narrow core with a middleware package beside it

**What this rung shows:** a corpus small enough to read end to end, where every reported pair can be checked

| | |
|---|---|
| Corpus | [chi](https://github.com/go-chi/chi) |
| Pinned at | `v5.3.2` (`38939062c5df4d3e8814aad1a488983112627ced`) |
| Project since | 2015 |
| doppel | `bc0615f` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 0 concepts modeled, 0 associations, 0 unusual realizations
Habitats: 2 modeled, 1 misfits; most uniform chi (norm 0.91), most diverse middleware (norm 0.87)
Ecosystems: 7 profiled (7 dominance, 0 coalition, 0 conflict, 0 weak)
Found 183 functions. Retrieving candidates...
Retrieval: shape 19, concept 5, call 357 -> 374 unique pairs
  concept-only 1.3%  call-only 93.6%  suppressed-shape functions: 0  large identity buckets: 0  surviving patterns: 2138
Running structural comparison on 374 pairs...
Families: 3 over 14 components, 17 functions in a family, 21 edges completed
  3 pairs suppressed by max-per-func=2
```

# Code Similarity Report

**Functions analyzed:** 183 | **Threshold:** 0.60 | **Pairs found:** 10

---

## What doppel sees

**183 functions** across **2 packages** — test functions excluded. Structural roles: 126 leaf, 27 orchestrator, 3 passthrough, 27 utility.

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
    c7["caching<br/>2"]
    c8["transaction<br/>absent"]
    c9["file_io<br/>1"]
    c10["logging<br/>absent"]
    c11(["data_transformation"])
    c12["mapping<br/>absent"]
    c13["validation<br/>2"]
    c14["serialization<br/>absent"]
    c15(["control_flow"])
    c16["concurrency<br/>1"]
    c17(["fault_tolerance"])
    c18["retry<br/>2"]
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
    class c3,c4,c6,c8,c10,c12,c14,c19,c21 hot
```

**Nothing here is tagged** `circuit_breaker`, `db_access`, `error_wrapping`, `grpc_call`, `http_call`, `logging`, `mapping`, `serialization`, `transaction`. That is a direct answer to "does this codebase already do X" — for those concepts, it does not.

| Concept | Functions | Convention |
|---|---:|---|
| `caching` | 2 | — |
| `retry` | 2 | — |
| `validation` | 2 | — |
| `concurrency` | 1 | — |
| `file_io` | 1 | — |

Convention is how uniformly this corpus realizes a concept: `1.00` means every function carrying the tag does it the same way, and a low number means the tag covers several unrelated habits. A concept with fewer than five members is not modeled.

### Where the duplication is

Merge-worthy pairs folded up to their packages. An edge means two packages keep solving the same problem separately; a count on a node means the repetition is inside one package.

### How settled each package is

A package with at least five functions gets a habitat model: doppel learns what is normal there and measures how surprising each member is against it. **Norm** is how uniform the package's practice is. A **misfit** is a function alien to its package *and* to the wider subsystem around it — one that fits its neighbours a directory up is normal for this codebase and is not reported.

```mermaid
flowchart TD
    h0["middleware<br/>106 functions · norm 0.87<br/>1 misfit"]
    h1["chi<br/>77 functions · norm 0.91"]
    classDef good fill:#d7ecd9,color:#1b3d20
    classDef warn fill:#fbeecb,color:#4a3a12
    classDef hot fill:#f7d6d6,color:#4a1c1c
    class h0,h1 good
```

Most uniform is `chi` (norm `0.91`); most varied is `middleware` (norm `0.87`). 1 functions are alien to their package and to the subsystem around it.

### How these candidates were found

Three channels propose candidates independently — shared rare *structure*, shared *concepts*, shared *calls* — and their union is what gets compared. This run: **374 candidate pairs** (shape 19, concept 5, call 357), of which 94% arrived on call evidence alone and 1% on concept evidence alone. A pair sharing none of the three is never compared, however alike it looks.

Each function is also an arena where its candidate concepts compete for its evidence. 7 functions reached an equilibrium: **7** settled on a single concept, **0** on a coalition, **0** hold concepts this corpus says do not go together.

_3 further pairs were held back so no single function fills the report._

### Corpus metrics

**Compression ratio:** `5.32`x — this corpus's canonical function bodies contain **11155 AST nodes** in total, which hash-cons (two nodes count as the same subtree exactly when their kind and every child match, all the way down) to **2097 distinct subtree shapes**; the ratio is nodes divided by shapes, always >= 1.0, and it never feeds any score.

**Nearest-neighbour code-shape:** of **183 functions**, **131** had a code-shape neighbour among the pairs retrieval actually scored — their best score's p50/p90/p99 are `0.41` / `1.00` / `1.00`, and 30% of them (39 of 131) already clear this run's threshold of `0.60`. This is **not an exhaustive nearest-neighbour search** (that would be a full pairwise comparison); it is bounded by the same three retrieval channels the pair list itself is bounded by, so the other 52 functions are excluded here as having no *scored* neighbour, not asserted to have none at all.

---

## Local practice

The vocabulary above says what a concept *is*. This says what one looks like when **this** codebase writes it — learned from the corpus, so it describes the house style rather than a rule from anywhere else.

---

## Match #1 — Code-shape: `0.6878`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/recoverer.go:132` | `middleware.prettyStack.decorateFuncCallLine` | `(string, bool, int) (string, error)` | — |
| **B** | `middleware/recoverer.go:172` | `middleware.prettyStack.decorateSourceLine` | `(string, bool, int) (string, error)` | — |

**Code similarity:** `wl 0.49  flow 1.00  nesting 0.90  sig 1.00  size 0.99`

**Containment:** `0.67`

**Evidence:** `803.08` (shape 787.95, concept 0.00, call 15.13)

**Trophic:** `0.71`

**Shared structure:**

- `14.36` — `flow:param→call:cW`
- `13.47` — `do(call:cW)`
- `7.76` — `flow:call:LastIndex→cond`

**Structural overlap:** `0.72` (merge-worthy)

- share 6 callees: [buf.String, cW, errors.New, string, strings.Index, strings.LastIndex]
- share 1 callers: [middleware.prettyStack.decorateLine]
- overlapping call-graph neighborhoods (1.00): 5 shared
- both are leaf functions
- same package
- same visibility
- same receiver type: prettyStack
- called from same packages: [middleware]
- call into same packages: [middleware]

---

## Match #2 — Code-shape: `0.7890`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tree.go:559` | `chi.*node.findEdge` | `(nodeTyp, byte) (*node)` | — |
| **B** | `tree.go:850` | `chi.nodes.findEdge` | `(byte) (*node)` | — |

**Code similarity:** `wl 0.77  flow 0.96  nesting 0.74  sig 0.67  size 0.80`

**Containment:** `0.97`

**Evidence:** `509.55` (shape 509.55, concept 0.00, call 0.00)

**Trophic:** `0.92`

**Shared structure:**

- `9.55` — `assign=(bin)`
- `4.28` — `seq[ assign:=(call:len) ; assign:=(lit:INT) ]`
- `4.28` — `seq[ assign=(bin) ; if(bin:>(id,sel)) ]`

**Structural overlap:** `0.50` (merge-worthy)

- share 1 callees: [len]
- both are leaf functions
- same package
- same visibility
- both are methods, on *node and nodes

---

## Match #3 — Code-shape: `0.8418`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `mux.go:203` | `chi.*Mux.NotFound` | `(http.HandlerFunc)` | — |
| **B** | `mux.go:223` | `chi.*Mux.MethodNotAllowed` | `(http.HandlerFunc)` | — |

**Code similarity:** `wl 0.74  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `0.85`

**Evidence:** `334.43` (shape 321.28, concept 0.00, call 13.14)

**Trophic:** `0.89`

**Shared structure:**

- `4.98` — `assign:=(id)`
- `4.82` — `assign=(sel)`
- `4.28` — `seq[ assign:=(id) ; if(bin:&&(sel,bin)) ]`

**Structural overlap:** `0.67` (merge-worthy)

- share 3 callees: [Chain, HandlerFunc, m.updateSubRoutes]
- share 1 callers: [chi.*Mux.Mount]
- overlapping call-graph neighborhoods (1.00): 12 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: Mux
- called from same packages: [chi]
- call into same packages: [chi]

---

## Match #4 — Code-shape: `0.6414`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/route_headers.go:48` | `middleware.HeaderRouter.Route` | `(string, string, func(next http.Handler) http.Handler) (HeaderRouter)` | — |
| **B** | `middleware/route_headers.go:58` | `middleware.HeaderRouter.RouteAny` | `(string, []string, func(next http.Handler) http.Handler) (HeaderRouter)` | — |

**Code similarity:** `wl 0.53  flow 0.82  nesting 1.00  sig 0.75  size 0.74`

**Containment:** `0.81` — most of the smaller body's shape is inside the larger

**Evidence:** `248.60` (shape 241.07, concept 0.00, call 7.53)

**Trophic:** `0.82`

**Shared structure:**

- `4.28` — `seq[ assign:=(index) ; if(bin:==(id,nil)) ]`
- `4.28` — `seq[ assign=(call:ToLower) ; assign:=(index) ]`
- `3.88` — `seq[ assign=(call:append) ; return(id) ]`

**Structural overlap:** `0.58` (merge-worthy)

- share 3 callees: [NewPattern, append, strings.ToLower]
- overlapping call-graph neighborhoods (1.00): 1 shared
- both are leaf functions
- same package
- same visibility
- same receiver type: HeaderRouter
- call into same packages: [middleware]

---

## Match #5 — Code-shape: `0.6090`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/content_encoding.go:10` | `middleware.AllowContentEncoding` | `(...string) (func(next http.Handler) http.Handler)` | — |
| **B** | `middleware/content_type.go:20` | `middleware.AllowContentType` | `(...string) (func(http.Handler) http.Handler)` | — |

**Code similarity:** `wl 0.52  flow 0.98  nesting 0.98  sig 0.33  size 0.98`

**Containment:** `0.70`

**Evidence:** `395.75` (shape 387.49, concept 0.00, call 8.27)

**Trophic:** `0.71`

**Shared structure:**

- `4.28` — `range{ call:TrimSpace call:ToLower }`
- `4.28` — `if(bin:==(sel,lit:INT))`
- `3.86` — `return()`

**Structural overlap:** `0.48` (merge-worthy)

- share 7 callees: [http.HandlerFunc, len, make, next.ServeHTTP, strings.ToLower, strings.TrimSpace, w.WriteHeader]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions

---

## Match #6 — Code-shape: `0.6883`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/strip.go:14` | `middleware.StripSlashes` | `(http.Handler) (http.Handler)` | — |
| **B** | `middleware/strip.go:41` | `middleware.RedirectSlashes` | `(http.Handler) (http.Handler)` | — |

**Code similarity:** `wl 0.49  flow 0.97  nesting 0.96  sig 1.00  size 0.83`

**Containment:** `0.74`

**Evidence:** `380.92` (shape 379.26, concept 0.00, call 1.65)

**Trophic:** `0.70`

**Shared structure:**

- `6.73` — `flow:call:RouteContext→cond`
- `4.98` — `if(bin:&&(bin,bin))`
- `4.82` — `assign=(sel)`

**Structural overlap:** `0.43` (merge-worthy)

- share 5 callees: [chi.RouteContext, http.HandlerFunc, len, next.ServeHTTP, r.Context]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions

---

## Match #7 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/wrap_writer.go:147` | `middleware.*flushWriter.Flush` | `()` | — |
| **B** | `middleware/wrap_writer.go:172` | `middleware.*flushHijackWriter.Flush` | `()` | — |

**Kind:** interface implementations — both implement `Flush()` on `*flushWriter` and `*flushHijackWriter`, in package `middleware`

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `81.38` (shape 81.38, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.59` — `seq[ assign:=(assert) ; do(call:Flush) ]`
- `3.59` — `seq[ assign=(true) ; assign:=(assert) ]`
- `3.37` — `do(call:Flush)`

**Structural overlap:** `0.50` (merge-worthy)

- share 1 callees: [fl.Flush]
- both are leaf functions
- same package
- same visibility
- both are methods, on *flushWriter and *flushHijackWriter

---

## Match #8 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/wrap_writer.go:147` | `middleware.*flushWriter.Flush` | `()` | — |
| **B** | `middleware/wrap_writer.go:194` | `middleware.*httpFancyWriter.Flush` | `()` | — |

**Kind:** interface implementations — both implement `Flush()` on `*flushWriter` and `*httpFancyWriter`, in package `middleware`

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `81.38` (shape 81.38, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.59` — `seq[ assign:=(assert) ; do(call:Flush) ]`
- `3.59` — `seq[ assign=(true) ; assign:=(assert) ]`
- `3.37` — `do(call:Flush)`

**Structural overlap:** `0.50` (merge-worthy)

- share 1 callees: [fl.Flush]
- both are leaf functions
- same package
- same visibility
- both are methods, on *flushWriter and *httpFancyWriter

---

## Match #9 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/wrap_writer.go:172` | `middleware.*flushHijackWriter.Flush` | `()` | — |
| **B** | `middleware/wrap_writer.go:194` | `middleware.*httpFancyWriter.Flush` | `()` | — |

**Kind:** interface implementations — both implement `Flush()` on `*flushHijackWriter` and `*httpFancyWriter`, in package `middleware`

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `81.38` (shape 81.38, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.59` — `seq[ assign:=(assert) ; do(call:Flush) ]`
- `3.59` — `seq[ assign=(true) ; assign:=(assert) ]`
- `3.37` — `do(call:Flush)`

**Structural overlap:** `0.50` (merge-worthy)

- share 1 callees: [fl.Flush]
- both are leaf functions
- same package
- same visibility
- both are methods, on *flushHijackWriter and *httpFancyWriter

---

## Match #10 — Code-shape: `0.6123`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/clean_path.go:12` | `middleware.CleanPath` | `(http.Handler) (http.Handler)` | — |
| **B** | `middleware/get_head.go:10` | `middleware.GetHead` | `(http.Handler) (http.Handler)` | — |

**Code similarity:** `wl 0.38  flow 0.98  nesting 0.79  sig 1.00  size 0.70`

**Containment:** `0.69` — most of the smaller body's shape is inside the larger

**Evidence:** `253.27` (shape 251.62, concept 0.00, call 1.65)

**Trophic:** `0.73`

**Shared structure:**

- `4.82` — `assign=(sel)`
- `4.28` — `seq[ assign:=(call:RouteContext) ; assign:=(sel) ]`
- `3.88` — `seq[ assign:=(sel) ; if(bin:==(id,lit:STRING)) ]`

**Structural overlap:** `0.45` (merge-worthy)

- share 4 callees: [chi.RouteContext, http.HandlerFunc, next.ServeHTTP, r.Context]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions

---

## Families

3 families, 17 functions in a family, largest 10 members; 21 edges scored here that retrieval never proposed

### Family 1 — 4 members, every pair `>= 1.00` code-shape, evidence `488`, interface implementations of `Flush()`, in package `middleware`

```mermaid
flowchart LR
    m0["middleware.*flushWriter.Flush"]
    m1["middleware.*flushHijackWriter.Flush"]
    m2["middleware.*httpFancyWriter.Flush"]
    m3["middleware.*http2FancyWriter.Flush"]
    m0 --- m1
    m0 --- m2
    m0 --- m3
    m1 --- m2
    m1 --- m3
    m2 --- m3
```

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `middleware/wrap_writer.go:147` | `middleware.*flushWriter.Flush` | `()` | — |
| `middleware/wrap_writer.go:172` | `middleware.*flushHijackWriter.Flush` | `()` | — |
| `middleware/wrap_writer.go:194` | `middleware.*httpFancyWriter.Flush` | `()` | — |
| `middleware/wrap_writer.go:239` | `middleware.*http2FancyWriter.Flush` | `()` | — |

### Family 2 — 3 members, every pair `>= 1.00` code-shape, evidence `197`, interface implementations of `Hijack() (net.Conn, *bufio.ReadWriter, error)`, in package `middleware`

```mermaid
flowchart LR
    m0["middleware.*hijackWriter.Hijack"]
    m1["middleware.*flushHijackWriter.Hijack"]
    m2["middleware.*httpFancyWriter.Hijack"]
    m0 --- m1
    m0 --- m2
    m1 --- m2
```

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `middleware/wrap_writer.go:160` | `middleware.*hijackWriter.Hijack` | `() (net.Conn, *bufio.ReadWriter, error)` | — |
| `middleware/wrap_writer.go:178` | `middleware.*flushHijackWriter.Hijack` | `() (net.Conn, *bufio.ReadWriter, error)` | — |
| `middleware/wrap_writer.go:200` | `middleware.*httpFancyWriter.Hijack` | `() (net.Conn, *bufio.ReadWriter, error)` | — |

### Family 3 — 10 members, every pair `>= 1.00` code-shape, evidence `63`  (21 edges scored here)

_Not drawn: 10 members is 45 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `mux.go:143` | `chi.*Mux.Connect` | `(string, http.HandlerFunc)` | — |
| `mux.go:149` | `chi.*Mux.Delete` | `(string, http.HandlerFunc)` | — |
| `mux.go:155` | `chi.*Mux.Get` | `(string, http.HandlerFunc)` | — |
| `mux.go:161` | `chi.*Mux.Head` | `(string, http.HandlerFunc)` | — |
| `mux.go:167` | `chi.*Mux.Options` | `(string, http.HandlerFunc)` | — |
| `mux.go:173` | `chi.*Mux.Patch` | `(string, http.HandlerFunc)` | — |
| `mux.go:179` | `chi.*Mux.Post` | `(string, http.HandlerFunc)` | — |
| `mux.go:185` | `chi.*Mux.Put` | `(string, http.HandlerFunc)` | — |
| `mux.go:191` | `chi.*Mux.Query` | `(string, http.HandlerFunc)` | — |
| `mux.go:197` | `chi.*Mux.Trace` | `(string, http.HandlerFunc)` | — |


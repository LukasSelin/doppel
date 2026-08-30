# chi

HTTP router; a narrow core with a middleware package beside it

**What this rung shows:** a corpus small enough to read end to end, where every reported pair can be checked

| | |
|---|---|
| Corpus | [chi](https://github.com/go-chi/chi) |
| Pinned at | `v5.3.2` (`38939062c5df4d3e8814aad1a488983112627ced`) |
| Project since | 2015 |
| doppel | `95071c4` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Learning concept vocabulary...
Lexicon: 21 concepts (0 seeded, 21 emergent), 717/1935 features above 67 df, 61 functions unlabeled
Generating concept documents...
Culture: 19 concepts modeled, 95 associations, 5 unusual realizations
Habitats: 2 modeled, 0 misfits; most uniform middleware (norm 0.90), most diverse chi (norm 0.89)
Conventions: strongest mx.handle+chi.*Mux.handle (0.65), loosest mx.inline+mx.handler (0.25)
Ecosystems: 133 profiled (103 dominance, 30 coalition, 0 conflict, 0 weak)
Calibration: rate 0.01 over 7875 shape / 16653 overlap null pairs -> threshold 0.45, struct-min 0.50, family-min 0.45
Found 183 functions. Retrieving candidates...
Retrieval: shape 77, concept 494, call 357 -> 755 unique pairs
  concept-only 45.6%  call-only 27.9%  suppressed-shape functions: 0  large identity buckets: 0  surviving labels: 1308
Running structural comparison on 755 pairs...
  113 pairs remain after struct-min=0.50 filter
Families: 6 over 15 components, 27 functions in a family, 22 edges completed
  1 pairs suppressed by max-per-func=2
```

# Code Similarity Report

**Functions analyzed:** 183 | **Threshold:** 0.60 | **Pairs found:** 10

---

## What doppel sees

**183 functions** across **2 packages** — test functions excluded. Structural roles: 126 leaf, 27 orchestrator, 3 passthrough, 27 utility.

### Concepts

These concepts were **learned from this corpus**, not read off a fixed list: each one is a group of functions that share a way of being written, named after the evidence that identified it. They hang from an authored interior, so two functions under the same *branch* score partial credit rather than nothing. Counts below are members; membership is graded, and a function can carry several.

```mermaid
flowchart LR
    c0(["concept"])
    c1(["io_operation"])
    c2(["remote_io"])
    c3(["data_store_access"])
    c4(["data_transformation"])
    c5(["control_flow"])
    c6(["fault_tolerance"])
    c7(["error_handling"])
    c8["URL.RawPath+chi.RouteContext<br/>8"]
    c9["b.ResponseWriter+b.discard<br/>29"]
    c10["buf.String+bytes.Buffer<br/>12"]
    c11["context.WithValue+r.WithContext<br/>8"]
    c12["cw.ResponseWriter+cw.writer<br/>20"]
    c13["fmt.Sprintf+r.Context<br/>10"]
    c14["h.handler+n.endpoints<br/>21"]
    c15["http.StatusUnsupportedMedia…+Header.Get<br/>10"]
    c16["http.StatusUnsupportedMedia…+Header.Get+context.WithValue<br/>17"]
    c17["http.StatusUnsupportedMedia…+chi.RouteContext<br/>29"]
    c18["http.StatusUnsupportedMedia…+strings.ToLower<br/>21"]
    c19["http.StatusUnsupportedMedia…+w.WriteHeader<br/>9"]
    c20["mx.handle+chi.*Mux.handle<br/>12"]
    c21["mx.inline+mx.handler<br/>6"]
    c22["mx.tree+rctx.RoutePath<br/>6"]
    c23["netip.Addr+context.WithValue<br/>15"]
    c24["r.Context+http.Handler<br/>9"]
    c25["rctx.URLParams+URLParams.Keys<br/>4"]
    c26["strings.Cut+chi.*Mux.Get<br/>5"]
    c27["strings.TrimSpace+space<br/>4"]
    c28["w.Header+http.Handler<br/>9"]
    c0 --> c1
    c1 --> c2
    c1 --> c3
    c0 --> c4
    c0 --> c5
    c5 --> c6
    c0 --> c7
    c0 --> c8
    c0 --> c9
    c0 --> c10
    c0 --> c11
    c0 --> c12
    c0 --> c13
    c0 --> c14
    c0 --> c15
    c0 --> c16
    c0 --> c17
    c0 --> c18
    c0 --> c19
    c0 --> c20
    c0 --> c21
    c0 --> c22
    c0 --> c23
    c0 --> c24
    c0 --> c25
    c0 --> c26
    c0 --> c27
    c0 --> c28
```

**No practice here for** `caching`, `circuit_breaker`, `concurrency`, `db_access`, `error_wrapping`, `file_io`, `grpc_call`, `http_call`, `logging`, `mapping`, `retry`, `serialization`, `transaction`, `validation`. Concepts are learned from this corpus, so one can never be absent — it exists because functions carry it. These are the *seeds* the search started from that grew nothing: a direct answer to "does this codebase already do X".

| Concept | Functions | Convention |
|---|---:|---|
| `b.ResponseWriter+b.discard` | 29 | `0.51` (settled) |
| `http.StatusUnsupportedMedia…+chi.RouteContext` | 29 | `0.49` (loose) |
| `h.handler+n.endpoints` | 21 | `0.54` (settled) |
| `http.StatusUnsupportedMedia…+strings.ToLower` | 21 | `0.47` (loose) |
| `cw.ResponseWriter+cw.writer` | 20 | `0.45` (loose) |
| `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` | 17 | `0.44` (loose) |
| `netip.Addr+context.WithValue` | 15 | `0.35` (loose) |
| `buf.String+bytes.Buffer` | 12 | `0.37` (loose) |
| `mx.handle+chi.*Mux.handle` | 12 | `0.65` (settled) |
| `fmt.Sprintf+r.Context` | 10 | `0.43` (loose) |
| `http.StatusUnsupportedMedia…+Header.Get` | 10 | `0.37` (loose) |
| `http.StatusUnsupportedMedia…+w.WriteHeader` | 9 | `0.37` (loose) |
| `r.Context+http.Handler` | 9 | `0.32` (loose) |
| `w.Header+http.Handler` | 9 | `0.57` (settled) |
| `URL.RawPath+chi.RouteContext` | 8 | `0.55` (settled) |
| `context.WithValue+r.WithContext` | 8 | `0.50` (loose) |
| `mx.inline+mx.handler` | 6 | `0.25` (loose) |
| `mx.tree+rctx.RoutePath` | 6 | `0.35` (loose) |
| `strings.Cut+chi.*Mux.Get` | 5 | `0.38` (loose) |
| `rctx.URLParams+URLParams.Keys` | 4 | — |
| `strings.TrimSpace+space` | 4 | — |

Convention is how uniformly this corpus realizes a concept: `1.00` means every function carrying the tag does it the same way, and a low number means the tag covers several unrelated habits. A concept with fewer than five members is not modeled.

### Where the duplication is

Merge-worthy pairs folded up to their packages. An edge means two packages keep solving the same problem separately; a count on a node means the repetition is inside one package.

### How settled each package is

A package with at least five functions gets a habitat model: doppel learns what is normal there and measures how surprising each member is against it. **Norm** is how uniform the package's practice is. A **misfit** is a function alien to its package *and* to the wider subsystem around it — one that fits its neighbours a directory up is normal for this codebase and is not reported.

```mermaid
flowchart TD
    h0["chi<br/>77 functions · norm 0.89"]
    h1["middleware<br/>106 functions · norm 0.90"]
    classDef good fill:#d7ecd9,color:#1b3d20
    classDef warn fill:#fbeecb,color:#4a3a12
    classDef hot fill:#f7d6d6,color:#4a1c1c
    class h0,h1 good
```

Most uniform is `middleware` (norm `0.90`); most varied is `chi` (norm `0.89`).

### How these candidates were found

Three channels propose candidates independently — shared rare *structure*, shared *concepts*, shared *calls* — and their union is what gets compared. This run: **755 candidate pairs** (shape 77, concept 494, call 357), of which 28% arrived on call evidence alone and 46% on concept evidence alone. A pair sharing none of the three is never compared, however alike it looks.

Each function is also an arena where its candidate concepts compete for its evidence. 133 functions reached an equilibrium: **103** settled on a single concept, **30** on a coalition, **0** hold concepts this corpus says do not go together.

_1 further pairs were held back so no single function fills the report._

### Corpus metrics

**Compression ratio:** `5.32`x — this corpus's canonical function bodies contain **11155 AST nodes** in total, which hash-cons (two nodes count as the same subtree exactly when their kind and every child match, all the way down) to **2097 distinct subtree shapes**; the ratio is nodes divided by shapes, always >= 1.0, and it never feeds any score.

**Nearest-neighbour code-shape:** of **183 functions**, **149** had a code-shape neighbour among the pairs retrieval actually scored — their best score's p50/p90/p99 are `0.42` / `0.90` / `1.00`, and 45% of them (67 of 149) already clear this run's threshold of `0.45`. This is **not an exhaustive nearest-neighbour search** (that would be a full pairwise comparison); it is bounded by the same three retrieval channels the pair list itself is bounded by, so the other 34 functions are excluded here as having no *scored* neighbour, not asserted to have none at all.

---

## Local practice

The vocabulary above says what a concept *is*. This says what one looks like when **this** codebase writes it — learned from the corpus, so it describes the house style rather than a rule from anywhere else.

### How this codebase writes each concept

Only what is **distinctive**. A feature earns a row by being carried by this concept's members at least twice as often as by the corpus at large — nearly every Go function has a `return` and an `if`, so prevalence alone would describe the language rather than this codebase. Weights are how much a channel counts toward whether a member looks normal — calls 40, control flow 20, co-occurring tags 15, role 15, package 10.

**`b.ResponseWriter+b.discard`** — 29 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| cotags ×15 | `cw.ResponseWriter+cw.writer` | `████······` | 13 of 29 | 4.1× |

**`http.StatusUnsupportedMedia…+chi.RouteContext`** — 29 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `net/http.HandlerFunc` | `██████████` | 28 of 29 | 5.0× |
| flow ×20 | `funclit` | `██████████` | 28 of 29 | 4.2× |
| cotags ×15 | `w.Header+http.Handler` | `███·······` | 9 of 29 | 6.3× |
|  | `URL.RawPath+chi.RouteContext` | `███·······` | 8 of 29 | 6.3× |
|  | `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` | `██████····` | 16 of 29 | 5.9× |
|  | `http.StatusUnsupportedMedia…+w.WriteHeader` | `███·······` | 8 of 29 | 5.6× |
|  | `http.StatusUnsupportedMedia…+strings.ToLower` | `██████····` | 17 of 29 | 5.1× |
| role ×15 | `orchestrator` | `███·······` | 10 of 29 | 2.3× |

**`h.handler+n.endpoints`** — 21 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| flow ×20 | `for` | `███·······` | 7 of 21 | 4.4× |
|  | `range` | `████······` | 8 of 21 | 2.7× |
| package ×10 | `chi` | `██████████` | 21 of 21 | 2.4× |

**`http.StatusUnsupportedMedia…+strings.ToLower`** — 21 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `strings.ToLower` | `███·······` | 7 of 21 | 6.8× |
|  | `net/http.HandlerFunc` | `█████████·` | 18 of 21 | 4.5× |
| flow ×20 | `funclit` | `█████████·` | 19 of 21 | 3.9× |
|  | `range` | `████······` | 9 of 21 | 3.0× |
| cotags ×15 | `w.Header+http.Handler` | `████······` | 9 of 21 | 8.7× |
|  | `http.StatusUnsupportedMedia…+Header.Get` | `████······` | 9 of 21 | 7.8× |
|  | `http.StatusUnsupportedMedia…+w.WriteHeader` | `████······` | 8 of 21 | 7.7× |
|  | `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` | `█████·····` | 10 of 21 | 5.1× |
|  | `http.StatusUnsupportedMedia…+chi.RouteContext` | `████████··` | 17 of 21 | 5.1× |

**`cw.ResponseWriter+cw.writer`** — 20 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `errors.New` | `███·······` | 5 of 20 | 9.2× |
| cotags ×15 | `buf.String+bytes.Buffer` | `████······` | 8 of 20 | 6.1× |
|  | `b.ResponseWriter+b.discard` | `███████···` | 13 of 20 | 4.1× |
| role ×15 | `orchestrator` | `████······` | 7 of 20 | 2.4× |

**`http.StatusUnsupportedMedia…+Header.Get+context.WithValue`** — 17 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `context.WithValue` | `█████·····` | 8 of 17 | 9.6× |
|  | `chi.*Mux.Get` | `████······` | 7 of 17 | 6.3× |
|  | `net/http.HandlerFunc` | `█████████·` | 16 of 17 | 4.9× |
| flow ×20 | `funclit` | `█████████·` | 16 of 17 | 4.1× |
|  | `range` | `████······` | 6 of 17 | 2.5× |
| cotags ×15 | `context.WithValue+r.WithContext` | `█████·····` | 8 of 17 | 11× |
|  | `http.StatusUnsupportedMedia…+w.WriteHeader` | `████······` | 7 of 17 | 8.4× |
|  | `http.StatusUnsupportedMedia…+Header.Get` | `████······` | 7 of 17 | 7.5× |
|  | `netip.Addr+context.WithValue` | `██████····` | 10 of 17 | 7.2× |
|  | `w.Header+http.Handler` | `████······` | 6 of 17 | 7.2× |
| role ×15 | `orchestrator` | `████······` | 7 of 17 | 2.8× |

_13 further concepts are modeled and not described._

### Which concepts share a function

`++` at least four times chance, `+` at least twice, `−` at most half, `never` not once. A blank cell is ordinary company — near chance, which is not culture.

| | `URL.RawPath+chi.RouteContext` | `b.ResponseWriter+b.discard` | `buf.String+bytes.Buffer` | `context.WithValue+r.WithContext` | `cw.ResponseWriter+cw.writer` | `fmt.Sprintf+r.Context` | `h.handler+n.endpoints` | `http.StatusUnsupportedMedia…+Header.Get` | `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` | `http.StatusUnsupportedMedia…+chi.RouteContext` | `http.StatusUnsupportedMedia…+strings.ToLower` | `http.StatusUnsupportedMedia…+w.WriteHeader` | `mx.handle+chi.*Mux.handle` | `mx.inline+mx.handler` | `mx.tree+rctx.RoutePath` | `netip.Addr+context.WithValue` | `r.Context+http.Handler` | `rctx.URLParams+URLParams.Keys` | `strings.Cut+chi.*Mux.Get` | `strings.TrimSpace+space` |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **`b.ResponseWriter+b.discard`** |  | | | | | | | | | | | | | | | | | | | |
| **`buf.String+bytes.Buffer`** |  |  | | | | | | | | | | | | | | | | | | |
| **`context.WithValue+r.WithContext`** |  |  |  | | | | | | | | | | | | | | | | | |
| **`cw.ResponseWriter+cw.writer`** |  | ++ | ++ |  | | | | | | | | | | | | | | | | |
| **`fmt.Sprintf+r.Context`** |  |  |  |  |  | | | | | | | | | | | | | | | |
| **`h.handler+n.endpoints`** |  | never |  |  |  | + | | | | | | | | | | | | | | |
| **`http.StatusUnsupportedMedia…+Header.Get`** |  |  |  |  |  |  |  | | | | | | | | | | | | | |
| **`http.StatusUnsupportedMedia…+Header.Get+context.WithValue`** |  |  |  | ++ |  |  |  | ++ | | | | | | | | | | | | |
| **`http.StatusUnsupportedMedia…+chi.RouteContext`** | ++ | − |  | ++ |  |  | − | ++ | ++ | | | | | | | | | | | |
| **`http.StatusUnsupportedMedia…+strings.ToLower`** |  |  |  |  |  |  |  | ++ | ++ | ++ | | | | | | | | | | |
| **`http.StatusUnsupportedMedia…+w.WriteHeader`** |  |  |  |  |  |  |  | ++ | ++ | ++ | ++ | | | | | | | | | |
| **`mx.handle+chi.*Mux.handle`** |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | |
| **`mx.inline+mx.handler`** |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | |
| **`mx.tree+rctx.RoutePath`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | |
| **`netip.Addr+context.WithValue`** |  |  |  | ++ |  |  |  |  | ++ | + |  |  |  |  |  | | | | | |
| **`r.Context+http.Handler`** |  |  | ++ |  |  |  |  |  | + | + | + |  |  |  |  | ++ | | | | |
| **`rctx.URLParams+URLParams.Keys`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | |
| **`strings.Cut+chi.*Mux.Get`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | |
| **`strings.TrimSpace+space`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | |
| **`w.Header+http.Handler`** |  |  |  |  |  |  |  | ++ | ++ | ++ | ++ | ++ |  |  |  |  | ++ |  |  |  |

### What travels with what

Co-occurrence measured against chance across every function. Only relationships at least twice — or at most half — as common as chance are reported; near-chance company is not culture. Each kind is listed separately, because there are far more call tokens than concepts and one shared list is all calls. Within a kind, strongest first means lift weighted by how many functions carry it — a 100× relationship holding for three functions is a weaker finding than a 30× one holding for thirty.

**Together more than chance — tag~tag**

- 8 of 8 `context.WithValue+r.WithContext` functions also `netip.Addr+context.WithValue` — 12× chance
- 8 of 8 `context.WithValue+r.WithContext` functions also `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` — 11× chance
- 16 of 17 `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` functions also `http.StatusUnsupportedMedia…+chi.RouteContext` — 5.9× chance
- 9 of 9 `w.Header+http.Handler` functions also `http.StatusUnsupportedMedia…+strings.ToLower` — 8.7× chance
- 9 of 10 `http.StatusUnsupportedMedia…+Header.Get` functions also `http.StatusUnsupportedMedia…+strings.ToLower` — 7.8× chance
- 10 of 15 `netip.Addr+context.WithValue` functions also `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` — 7.2× chance
- _24 more not listed_

**Together more than chance — tag~role**

- 7 of 17 `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` functions also `orchestrator` — 2.8× chance
- 10 of 29 `http.StatusUnsupportedMedia…+chi.RouteContext` functions also `orchestrator` — 2.3× chance
- 4 of 8 `context.WithValue+r.WithContext` functions also `orchestrator` — 3.4× chance
- 5 of 12 `buf.String+bytes.Buffer` functions also `orchestrator` — 2.8× chance
- 7 of 20 `cw.ResponseWriter+cw.writer` functions also `orchestrator` — 2.4× chance
- 3 of 6 `mx.tree+rctx.RoutePath` functions also `orchestrator` — 3.4× chance
- _4 more not listed_

**Together more than chance — tag~call**

- 12 of 12 `mx.handle+chi.*Mux.handle` functions also `chi.*Mux.handle` — 14× chance
- 8 of 8 `context.WithValue+r.WithContext` functions also `context.WithValue` — 20× chance
- 10 of 10 `fmt.Sprintf+r.Context` functions also `fmt.Sprintf` — 14× chance
- 5 of 5 `strings.Cut+chi.*Mux.Get` functions also `strings.Cut` — 30× chance
- 4 of 4 `strings.TrimSpace+space` functions also `strings.TrimSpace` — 37× chance
- 9 of 15 `netip.Addr+context.WithValue` functions also `context.WithValue` — 12× chance
- _39 more not listed_

**Apart more than chance — tag~tag**

- **no** `b.ResponseWriter+b.discard` function has `h.handler+n.endpoints` — chance alone would give about 3 of 29
- 1 of 29 `b.ResponseWriter+b.discard` functions also `http.StatusUnsupportedMedia…+chi.RouteContext` — 0.2× chance
- 1 of 21 `h.handler+n.endpoints` functions also `http.StatusUnsupportedMedia…+chi.RouteContext` — 0.3× chance

**Apart more than chance — tag~role**

- 1 of 29 `b.ResponseWriter+b.discard` functions also `utility` — 0.2× chance
- 1 of 29 `http.StatusUnsupportedMedia…+chi.RouteContext` functions also `utility` — 0.2× chance
- 1 of 6 `mx.tree+rctx.RoutePath` functions also `leaf` — 0.2× chance
- 2 of 6 `mx.inline+mx.handler` functions also `leaf` — 0.5× chance
- 1 of 21 `http.StatusUnsupportedMedia…+strings.ToLower` functions also `utility` — 0.3× chance

**Apart more than chance — tag~call**

- 1 of 29 `b.ResponseWriter+b.discard` functions also `net/http.HandlerFunc` — 0.2× chance
- 1 of 21 `h.handler+n.endpoints` functions also `net/http.HandlerFunc` — 0.2× chance

### Functions drifting from their own concept

These carry a tag but look nothing like the other functions carrying it. Typicality is measured against the concept's own median, so a genuinely varied concept lowers its own bar and a tight one can flag nobody.

| Function | Concept | Typicality | Concept median | |
|---|---|---:|---:|---|
| `chi.*Mux.routeHTTP` <br/>`mux.go:447` | `URL.RawPath+chi.RouteContext` | `0.19` | `0.68` | no near-duplicate |
| `chi.*Mux.routeHTTP` <br/>`mux.go:447` | `http.StatusUnsupportedMedia…+chi.RouteContext` | `0.21` | `0.54` | no near-duplicate |
| `middleware.*Compressor.Handler` <br/>`middleware/compress.go:199` | `b.ResponseWriter+b.discard` | `0.21` | `0.47` | no near-duplicate |
| `chi.*Mux.Method` <br/>`mux.go:127` | `mx.handle+chi.*Mux.handle` | `0.28` | `0.90` |  |
| `chi.*Mux.Handle` <br/>`mux.go:109` | `mx.handle+chi.*Mux.handle` | `0.35` | `0.90` |  |

A row marked _no near-duplicate_ appears in no reported pair: nothing else in this report explains it, which makes it drift rather than duplication.

---

## Match #1 — Code-shape: `0.6878`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `middleware/recoverer.go:132` | `middleware.prettyStack.decorateFuncCallLine` | `(string, bool, int) (string, error)` | buf.String+bytes.Buffer 0.74, cw.ResponseWriter+cw.writer 0.68, b.ResponseWriter+b.discard 0.44 |
| **B** | `middleware/recoverer.go:172` | `middleware.prettyStack.decorateSourceLine` | `(string, bool, int) (string, error)` | buf.String+bytes.Buffer 0.74, cw.ResponseWriter+cw.writer 0.68, b.ResponseWriter+b.discard 0.44 |

**Explain:** differs by four extra assign, one extra if, five extra binary, and 7 more kinds

**Profile A:** `cw.ResponseWriter+cw.writer` 0.77, `buf.String+bytes.Buffer` 0.23 (dominance)

**Profile B:** `cw.ResponseWriter+cw.writer` 0.77, `buf.String+bytes.Buffer` 0.23 (dominance)

**Code similarity:** `wl 0.49  flow 1.00  nesting 0.90  sig 1.00  size 0.99`

**Containment:** `0.67`

**Evidence:** `506.88` (shape 486.74, concept 5.01, call 15.13)

**Trophic:** `0.76`

**Shared structure:**

- `12.91` — `depth-1 EXPRSTMT` ×4
- `12.91` — `depth-0 CALL` ×4
- `11.08` — `depth-3 LIT` ×11

**Structural overlap:** `0.95` (merge-worthy)

- share 6 callees: [buf.String, cW, errors.New, string, strings.Index, strings.LastIndex]
- share 1 callers: [middleware.prettyStack.decorateLine]
- overlapping call-graph neighborhoods (1.00): 5 shared
- share patterns: [b.ResponseWriter+b.discard, buf.String+bytes.Buffer, cw.ResponseWriter+cw.writer]
- both are leaf functions
- same package
- callers do related work (1.00): [strings.TrimSpace+space, fmt.Sprintf+r.Context, buf.String+bytes.Buffer, cw.ResponseWriter+cw.writer]
- same visibility
- same receiver type: prettyStack
- called from same packages: [middleware]
- call into same packages: [middleware]

---

## Match #2 — Code-shape: `0.7890`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `tree.go:559` | `chi.*node.findEdge` | `(nodeTyp, byte) (*node)` | h.handler+n.endpoints 0.68 |
| **B** | `tree.go:850` | `chi.nodes.findEdge` | `(byte) (*node)` | h.handler+n.endpoints 0.54 |

**Explain:** differs by two extra case, one extra assign, one extra return, and 5 more kinds

**Profile A:** `h.handler+n.endpoints` 1.00 (dominance)

**Profile B:** `h.handler+n.endpoints` 1.00 (dominance)

**Code similarity:** `wl 0.77  flow 0.96  nesting 0.74  sig 0.67  size 0.80`

**Containment:** `0.97`

**Evidence:** `413.36` (shape 411.97, concept 1.39, call 0.00)

**Trophic:** `0.94`

**Shared structure:**

- `9.68` — `depth-3 SEL` ×3
- `9.68` — `depth-2 SEL` ×3
- `8.29` — `depth-3 BIN` ×2

**Structural overlap:** `0.65` (merge-worthy)

- share 1 callees: [len]
- share patterns: [h.handler+n.endpoints]
- both are leaf functions
- same package
- same visibility
- both are methods, on *node and nodes

---

## Match #3 — Code-shape: `0.8418`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `mux.go:203` | `chi.*Mux.NotFound` | `(http.HandlerFunc)` | mx.inline+mx.handler 0.55 |
| **B** | `mux.go:223` | `chi.*Mux.MethodNotAllowed` | `(http.HandlerFunc)` | mx.inline+mx.handler 0.55 |

**Explain:** differs by two extra call

**Profile A:** `mx.inline+mx.handler` 1.00 (dominance)

**Profile B:** `mx.inline+mx.handler` 1.00 (dominance)

**Code similarity:** `wl 0.74  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `0.85`

**Evidence:** `255.52` (shape 240.31, concept 2.07, call 13.14)

**Trophic:** `0.95`

**Shared structure:**

- `4.62` — `depth-1 ASSIGN` ×3
- `4.54` — `depth-3 ASSIGN` ×2
- `4.54` — `depth-2 ASSIGN` ×2

**Structural overlap:** `0.95` (merge-worthy)

- share 3 callees: [Chain, HandlerFunc, m.updateSubRoutes]
- share 1 callers: [chi.*Mux.Mount]
- overlapping call-graph neighborhoods (1.00): 12 shared
- share patterns: [mx.inline+mx.handler]
- both are orchestrator functions
- same package
- callers do related work (1.00): [rctx.URLParams+URLParams.Keys, mx.tree+rctx.RoutePath, fmt.Sprintf+r.Context, h.handler+n.endpoints, http.StatusUnsupportedMedia…+chi.RouteContext]
- callees do related work (1.00): [mx.tree+rctx.RoutePath]
- same visibility
- same receiver type: Mux
- called from same packages: [chi]
- call into same packages: [chi]

---

## Match #4 — Code-shape: `0.6090`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `middleware/content_encoding.go:10` | `middleware.AllowContentEncoding` | `(...string) (func(next http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.72, http.StatusUnsupportedMedia…+Header.Get 0.68, http.StatusUnsupportedMedia…+w.WriteHeader 0.68, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.61, +2 more |
| **B** | `middleware/content_type.go:20` | `middleware.AllowContentType` | `(...string) (func(http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.72, http.StatusUnsupportedMedia…+Header.Get 0.70, http.StatusUnsupportedMedia…+w.WriteHeader 0.69, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.63, +3 more |

**Explain:** differs by two extra assign, one extra range, two extra call, and 6 more kinds

**Profile A:** `http.StatusUnsupportedMedia…+Header.Get` 0.52, `http.StatusUnsupportedMedia…+w.WriteHeader` 0.46 (coalition)

**Profile B:** `http.StatusUnsupportedMedia…+Header.Get` 0.60, `http.StatusUnsupportedMedia…+w.WriteHeader` 0.40 (dominance)

**Code similarity:** `wl 0.52  flow 0.98  nesting 0.98  sig 0.33  size 0.98`

**Containment:** `0.70`

**Evidence:** `365.04` (shape 347.48, concept 9.29, call 8.27)

**Trophic:** `0.79`

**Shared structure:**

- `7.48` — `depth-3 STRUCTTYPE` ×2
- `7.48` — `depth-2 STRUCTTYPE` ×2
- `7.48` — `depth-1 STRUCTTYPE` ×2

**Structural overlap:** `0.60` (merge-worthy)

- share 7 callees: [http.HandlerFunc, len, make, next.ServeHTTP, strings.ToLower, strings.TrimSpace, w.WriteHeader]
- share patterns: [http.StatusUnsupportedMedia…+Header.Get, http.StatusUnsupportedMedia…+Header.Get+context.WithValue, http.StatusUnsupportedMedia…+chi.RouteContext, http.StatusUnsupportedMedia…+strings.ToLower, http.StatusUnsupportedMedia…+w.WriteHeader]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions

---

## Match #5 — Code-shape: `0.6883`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `middleware/strip.go:14` | `middleware.StripSlashes` | `(http.Handler) (http.Handler)` | http.StatusUnsupportedMedia…+chi.RouteContext 0.54, URL.RawPath+chi.RouteContext 0.51 |
| **B** | `middleware/strip.go:41` | `middleware.RedirectSlashes` | `(http.Handler) (http.Handler)` | http.StatusUnsupportedMedia…+chi.RouteContext 0.56, URL.RawPath+chi.RouteContext 0.54, fmt.Sprintf+r.Context 0.50 |

**Explain:** differs by one extra return, six extra literal, five extra call, and 6 more kinds

**Profile A:** `URL.RawPath+chi.RouteContext` 0.64, `http.StatusUnsupportedMedia…+chi.RouteContext` 0.36 (dominance)

**Profile B:** `fmt.Sprintf+r.Context` 1.00 (dominance)

**Code similarity:** `wl 0.49  flow 0.97  nesting 0.96  sig 1.00  size 0.83`

**Containment:** `0.74`

**Evidence:** `307.66` (shape 302.98, concept 3.03, call 1.65)

**Trophic:** `0.83`

**Shared structure:**

- `4.39` — `depth-2 BLOCK` ×2
- `4.14` — `depth-3 BIN`
- `4.14` — `depth-3 IF`

**Structural overlap:** `0.55` (merge-worthy)

- share 5 callees: [chi.RouteContext, http.HandlerFunc, len, next.ServeHTTP, r.Context]
- share patterns: [URL.RawPath+chi.RouteContext, http.StatusUnsupportedMedia…+chi.RouteContext]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions

---

## Match #6 — Code-shape: `0.6414`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `middleware/route_headers.go:48` | `middleware.HeaderRouter.Route` | `(string, string, func(next http.Handler) http.Handler) (HeaderRouter)` | http.StatusUnsupportedMedia…+strings.ToLower 0.50, http.StatusUnsupportedMedia…+Header.Get 0.48 |
| **B** | `middleware/route_headers.go:58` | `middleware.HeaderRouter.RouteAny` | `(string, []string, func(next http.Handler) http.Handler) (HeaderRouter)` | http.StatusUnsupportedMedia…+strings.ToLower 0.53, http.StatusUnsupportedMedia…+Header.Get 0.52 |

**Explain:** differs by two extra assign, one extra range, one extra call, and 4 more kinds

**Profile A:** `http.StatusUnsupportedMedia…+Header.Get` 0.75, `http.StatusUnsupportedMedia…+strings.ToLower` 0.24 (dominance)

**Profile B:** `http.StatusUnsupportedMedia…+Header.Get` 0.76, `http.StatusUnsupportedMedia…+strings.ToLower` 0.23 (dominance)

**Code similarity:** `wl 0.53  flow 0.82  nesting 1.00  sig 0.75  size 0.74`

**Containment:** `0.81` — most of the smaller body's shape is inside the larger

**Evidence:** `181.69` (shape 171.36, concept 2.81, call 7.53)

**Trophic:** `0.86`

**Shared structure:**

- `6.98` — `depth-3 INDEX` ×4
- `6.98` — `depth-2 INDEX` ×4
- `6.98` — `depth-1 INDEX` ×4

**Structural overlap:** `0.80` (merge-worthy)

- share 3 callees: [NewPattern, append, strings.ToLower]
- overlapping call-graph neighborhoods (1.00): 1 shared
- share patterns: [http.StatusUnsupportedMedia…+Header.Get, http.StatusUnsupportedMedia…+strings.ToLower]
- both are leaf functions
- same package
- callees do related work (1.00): [strings.Cut+chi.*Mux.Get]
- same visibility
- same receiver type: HeaderRouter
- call into same packages: [middleware]

---

## Match #7 — Code-shape: `0.5097`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `middleware/client_ip.go:92` | `middleware.ClientIPFromXFF` | `(...string) (func(http.Handler) http.Handler)` | netip.Addr+context.WithValue 0.78, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.63, context.WithValue+r.WithContext 0.63, http.StatusUnsupportedMedia…+chi.RouteContext 0.58, +1 more |
| **B** | `middleware/client_ip.go:149` | `middleware.ClientIPFromXFFTrustedProxies` | `(int) (func(http.Handler) http.Handler)` | netip.Addr+context.WithValue 0.75, context.WithValue+r.WithContext 0.63, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.62, http.StatusUnsupportedMedia…+chi.RouteContext 0.58 |

**Explain:** differs by one extra assign, one extra if, one extra increment, and 11 more kinds

**Profile A:** `context.WithValue+r.WithContext` 0.94, `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` 0.06 (dominance)

**Profile B:** `context.WithValue+r.WithContext` 0.95, `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` 0.05 (dominance)

**Code similarity:** `wl 0.36  flow 0.97  nesting 0.98  sig 0.33  size 0.87`

**Containment:** `0.56`

**Evidence:** `303.51` (shape 282.84, concept 7.37, call 13.29)

**Trophic:** `0.71`

**Shared structure:**

- `6.74` — `depth-0 FIELD` ×6
- `6.31` — `depth-1 FIELDLIST` ×4
- `5.38` — `depth-0 FIELDLIST` ×5

**Structural overlap:** `0.68` (merge-worthy)

- share 7 callees: [context.WithValue, h.ServeHTTP, http.HandlerFunc, parseHeaderAddr, r.Context, r.WithContext, walkXFF]
- overlapping call-graph neighborhoods (0.75): 3 shared
- share patterns: [context.WithValue+r.WithContext, http.StatusUnsupportedMedia…+Header.Get+context.WithValue, http.StatusUnsupportedMedia…+chi.RouteContext, netip.Addr+context.WithValue]
- both are orchestrator functions
- same package
- callees do related work (1.00): [strings.TrimSpace+space, netip.Addr+context.WithValue]
- same visibility
- same receiver type: plain functions
- call into same packages: [middleware]

---

## Match #8 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `middleware/wrap_writer.go:147` | `middleware.*flushWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.60 |
| **B** | `middleware/wrap_writer.go:172` | `middleware.*flushHijackWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.60 |

**Kind:** interface implementations — both implement `Flush()` on `*flushWriter` and `*flushHijackWriter`, in package `middleware`

**Explain:** identical after rename

**Profile A:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Profile B:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `63.19` (shape 61.82, concept 1.37, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.45` — `depth-3 BLOCK`
- `3.45` — `depth-2 BLOCK`
- `3.45` — `depth-1 BLOCK`

**Structural overlap:** `0.68` (merge-worthy)

- share 1 callees: [fl.Flush]
- share patterns: [b.ResponseWriter+b.discard]
- both are leaf functions
- same package
- same visibility
- both are methods, on *flushWriter and *flushHijackWriter

---

## Match #9 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `middleware/wrap_writer.go:147` | `middleware.*flushWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.60 |
| **B** | `middleware/wrap_writer.go:194` | `middleware.*httpFancyWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.60 |

**Kind:** interface implementations — both implement `Flush()` on `*flushWriter` and `*httpFancyWriter`, in package `middleware`

**Explain:** identical after rename

**Profile A:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Profile B:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `63.19` (shape 61.82, concept 1.37, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.45` — `depth-3 BLOCK`
- `3.45` — `depth-2 BLOCK`
- `3.45` — `depth-1 BLOCK`

**Structural overlap:** `0.68` (merge-worthy)

- share 1 callees: [fl.Flush]
- share patterns: [b.ResponseWriter+b.discard]
- both are leaf functions
- same package
- same visibility
- both are methods, on *flushWriter and *httpFancyWriter

---

## Match #10 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `middleware/wrap_writer.go:172` | `middleware.*flushHijackWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.60 |
| **B** | `middleware/wrap_writer.go:194` | `middleware.*httpFancyWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.60 |

**Kind:** interface implementations — both implement `Flush()` on `*flushHijackWriter` and `*httpFancyWriter`, in package `middleware`

**Explain:** identical after rename

**Profile A:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Profile B:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `63.19` (shape 61.82, concept 1.37, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.45` — `depth-3 BLOCK`
- `3.45` — `depth-2 BLOCK`
- `3.45` — `depth-1 BLOCK`

**Structural overlap:** `0.68` (merge-worthy)

- share 1 callees: [fl.Flush]
- share patterns: [b.ResponseWriter+b.discard]
- both are leaf functions
- same package
- same visibility
- both are methods, on *flushHijackWriter and *httpFancyWriter

---

## Families

6 families, 27 functions in a family, largest 10 members; 22 edges scored here that retrieval never proposed

### Family 1 — 4 members, every pair `>= 0.50` code-shape, evidence `922`

```mermaid
flowchart LR
    m0["middleware.CleanPath"]
    m1["middleware.GetHead"]
    m2["middleware.StripSlashes"]
    m3["middleware.RedirectSlashes"]
    m0 --- m1
    m0 --- m2
    m0 --- m3
    m1 --- m2
    m1 --- m3
    m2 --- m3
```

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `middleware/clean_path.go:12` | `middleware.CleanPath` | `(http.Handler) (http.Handler)` | URL.RawPath+chi.RouteContext 0.56, http.StatusUnsupportedMedia…+chi.RouteContext 0.55 |
| `middleware/get_head.go:10` | `middleware.GetHead` | `(http.Handler) (http.Handler)` | URL.RawPath+chi.RouteContext 0.60, http.StatusUnsupportedMedia…+chi.RouteContext 0.58 |
| `middleware/strip.go:14` | `middleware.StripSlashes` | `(http.Handler) (http.Handler)` | http.StatusUnsupportedMedia…+chi.RouteContext 0.54, URL.RawPath+chi.RouteContext 0.51 |
| `middleware/strip.go:41` | `middleware.RedirectSlashes` | `(http.Handler) (http.Handler)` | http.StatusUnsupportedMedia…+chi.RouteContext 0.56, URL.RawPath+chi.RouteContext 0.54, fmt.Sprintf+r.Context 0.50 |

### Family 2 — 3 members, every pair `>= 0.46` code-shape, evidence `542`  (1 edge scored here)

```mermaid
flowchart LR
    m0["middleware.ContentCharset"]
    m1["middleware.AllowContentEncoding"]
    m2["middleware.AllowContentType"]
    m0 --- m1
    m0 --- m2
    m1 --- m2
```

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `middleware/content_charset.go:11` | `middleware.ContentCharset` | `(...string) (func(next http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.72, http.StatusUnsupportedMedia…+Header.Get 0.70, http.StatusUnsupportedMedia…+w.WriteHeader 0.68, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.62, +1 more |
| `middleware/content_encoding.go:10` | `middleware.AllowContentEncoding` | `(...string) (func(next http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.72, http.StatusUnsupportedMedia…+Header.Get 0.68, http.StatusUnsupportedMedia…+w.WriteHeader 0.68, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.61, +2 more |
| `middleware/content_type.go:20` | `middleware.AllowContentType` | `(...string) (func(http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.72, http.StatusUnsupportedMedia…+Header.Get 0.70, http.StatusUnsupportedMedia…+w.WriteHeader 0.69, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.63, +3 more |

### Family 3 — 4 members, every pair `>= 1.00` code-shape, evidence `379`, interface implementations of `Flush()`, in package `middleware`

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

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `middleware/wrap_writer.go:147` | `middleware.*flushWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.60 |
| `middleware/wrap_writer.go:172` | `middleware.*flushHijackWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.60 |
| `middleware/wrap_writer.go:194` | `middleware.*httpFancyWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.60 |
| `middleware/wrap_writer.go:239` | `middleware.*http2FancyWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.60 |

### Family 4 — 3 members, every pair `>= 0.57` code-shape, evidence `306`

```mermaid
flowchart LR
    m0["middleware.SetHeader"]
    m1["middleware.PageRoute"]
    m2["middleware.PathRewrite"]
    m0 --- m1
    m0 --- m2
    m1 --- m2
```

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `middleware/content_type.go:9` | `middleware.SetHeader` | `(string, string) (func(http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.47, w.Header+http.Handler 0.44, http.StatusUnsupportedMedia…+chi.RouteContext 0.40 |
| `middleware/page_route.go:10` | `middleware.PageRoute` | `(string, http.Handler) (func(http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.48, http.StatusUnsupportedMedia…+chi.RouteContext 0.45, URL.RawPath+chi.RouteContext 0.40 |
| `middleware/path_rewrite.go:9` | `middleware.PathRewrite` | `(string, string) (func(http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.47, http.StatusUnsupportedMedia…+chi.RouteContext 0.40 |

### Family 5 — 3 members, every pair `>= 0.48` code-shape, evidence `218`

```mermaid
flowchart LR
    m0["middleware.*compressResponseWriter.Hijack"]
    m1["middleware.*compressResponseWriter.Push"]
    m2["middleware.*compressResponseWriter.Close"]
    m0 --- m1
    m0 --- m2
    m1 --- m2
```

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `middleware/compress.go:365` | `middleware.*compressResponseWriter.Hijack` | `() (net.Conn, *bufio.ReadWriter, error)` | b.ResponseWriter+b.discard 0.71, cw.ResponseWriter+cw.writer 0.58 |
| `middleware/compress.go:372` | `middleware.*compressResponseWriter.Push` | `(string, *http.PushOptions) (error)` | b.ResponseWriter+b.discard 0.66, cw.ResponseWriter+cw.writer 0.58 |
| `middleware/compress.go:379` | `middleware.*compressResponseWriter.Close` | `() (error)` | b.ResponseWriter+b.discard 0.63, cw.ResponseWriter+cw.writer 0.58 |

_1 more families not listed._


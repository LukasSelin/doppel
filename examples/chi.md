# chi

HTTP router; a narrow core with a middleware package beside it

**What this rung shows:** a corpus small enough to read end to end, where every reported pair can be checked

| | |
|---|---|
| Corpus | [chi](https://github.com/go-chi/chi) |
| Pinned at | `v5.3.2` (`38939062c5df4d3e8814aad1a488983112627ced`) |
| Project since | 2015 |
| doppel | `7c27a17` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`; CI regenerates on every push to master.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Learning concept vocabulary...
Lexicon: 21 concepts (0 seeded, 21 emergent), 770/2177 features above 67 df, 61 functions unlabeled
Generating concept documents...
Culture: 19 concepts modeled, 102 associations, 7 unusual realizations
Habitats: 2 modeled, 0 misfits; most uniform middleware (norm 0.90), most diverse chi (norm 0.89)
Conventions: strongest mx.handle+chi.*Mux.handle (0.60), loosest mx.inline+mx.handler (0.25)
Ecosystems: 133 profiled (105 dominance, 28 coalition, 0 conflict, 0 weak)
Calibration: rate 0.01 over 10440 shape / 16653 overlap null pairs -> threshold 0.53, struct-min 0.50, family-min 0.53
Found 183 functions. Retrieving candidates...
Retrieval: shape 86, concept 488, call 357 -> 750 unique pairs
  concept-only 44.7%  call-only 28.4%  suppressed-shape functions: 0  large identity buckets: 0  surviving patterns: 2138
Running structural comparison on 750 pairs...
  118 pairs remain after struct-min=0.50 filter
Families: 6 over 18 components, 27 functions in a family, 22 edges completed
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
    c11["context.WithValue+r.WithContext<br/>9"]
    c12["cw.ResponseWriter+cw.writer<br/>21"]
    c13["fmt.Sprintf+r.Context<br/>10"]
    c14["h.handler+n.endpoints<br/>21"]
    c15["http.StatusUnsupportedMedia…+Header.Get<br/>10"]
    c16["http.StatusUnsupportedMedia…+Header.Get+context.WithValue<br/>17"]
    c17["http.StatusUnsupportedMedia…+chi.RouteContext<br/>29"]
    c18["http.StatusUnsupportedMedia…+strings.ToLower<br/>24"]
    c19["http.StatusUnsupportedMedia…+w.WriteHeader<br/>7"]
    c20["mx.handle+chi.*Mux.handle<br/>13"]
    c21["mx.inline+mx.handler<br/>6"]
    c22["mx.tree+rctx.RoutePath<br/>6"]
    c23["netip.Addr+context.WithValue<br/>15"]
    c24["r.Context+http.Handler<br/>9"]
    c25["rctx.URLParams+URLParams.Keys<br/>4"]
    c26["strings.Cut+chi.*Mux.Get<br/>5"]
    c27["strings.TrimSpace+space<br/>4"]
    c28["w.Header+http.Handler<br/>8"]
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
| `b.ResponseWriter+b.discard` | 29 | `0.50` (settled) |
| `http.StatusUnsupportedMedia…+chi.RouteContext` | 29 | `0.50` (loose) |
| `http.StatusUnsupportedMedia…+strings.ToLower` | 24 | `0.47` (loose) |
| `cw.ResponseWriter+cw.writer` | 21 | `0.44` (loose) |
| `h.handler+n.endpoints` | 21 | `0.55` (settled) |
| `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` | 17 | `0.44` (loose) |
| `netip.Addr+context.WithValue` | 15 | `0.35` (loose) |
| `mx.handle+chi.*Mux.handle` | 13 | `0.60` (settled) |
| `buf.String+bytes.Buffer` | 12 | `0.37` (loose) |
| `fmt.Sprintf+r.Context` | 10 | `0.41` (loose) |
| `http.StatusUnsupportedMedia…+Header.Get` | 10 | `0.41` (loose) |
| `context.WithValue+r.WithContext` | 9 | `0.46` (loose) |
| `r.Context+http.Handler` | 9 | `0.32` (loose) |
| `URL.RawPath+chi.RouteContext` | 8 | `0.54` (settled) |
| `w.Header+http.Handler` | 8 | `0.54` (settled) |
| `http.StatusUnsupportedMedia…+w.WriteHeader` | 7 | `0.38` (loose) |
| `mx.inline+mx.handler` | 6 | `0.25` (loose) |
| `mx.tree+rctx.RoutePath` | 6 | `0.35` (loose) |
| `strings.Cut+chi.*Mux.Get` | 5 | `0.45` (loose) |
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

Three channels propose candidates independently — shared rare *structure*, shared *concepts*, shared *calls* — and their union is what gets compared. This run: **750 candidate pairs** (shape 86, concept 488, call 357), of which 28% arrived on call evidence alone and 45% on concept evidence alone. A pair sharing none of the three is never compared, however alike it looks.

Each function is also an arena where its candidate concepts compete for its evidence. 133 functions reached an equilibrium: **105** settled on a single concept, **28** on a coalition, **0** hold concepts this corpus says do not go together.

_1 further pairs were held back so no single function fills the report._

---

## Local practice

The vocabulary above says what a concept *is*. This says what one looks like when **this** codebase writes it — learned from the corpus, so it describes the house style rather than a rule from anywhere else.

### How this codebase writes each concept

Only what is **distinctive**. A feature earns a row by being carried by this concept's members at least twice as often as by the corpus at large — nearly every Go function has a `return` and an `if`, so prevalence alone would describe the language rather than this codebase. Weights are how much a channel counts toward whether a member looks normal — calls 40, control flow 20, co-occurring tags 15, role 15, package 10.

**`b.ResponseWriter+b.discard`** — 29 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| cotags ×15 | `cw.ResponseWriter+cw.writer` | `████······` | 13 of 29 | 3.9× |

**`http.StatusUnsupportedMedia…+chi.RouteContext`** — 29 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `net/http.HandlerFunc` | `██████████` | 28 of 29 | 5.0× |
| flow ×20 | `funclit` | `██████████` | 28 of 29 | 4.2× |
| cotags ×15 | `URL.RawPath+chi.RouteContext` | `███·······` | 8 of 29 | 6.3× |
|  | `w.Header+http.Handler` | `███·······` | 8 of 29 | 6.3× |
|  | `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` | `██████····` | 16 of 29 | 5.9× |
|  | `context.WithValue+r.WithContext` | `███·······` | 8 of 29 | 5.6× |
|  | `http.StatusUnsupportedMedia…+strings.ToLower` | `███████···` | 19 of 29 | 5.0× |
| role ×15 | `orchestrator` | `███·······` | 10 of 29 | 2.3× |

**`http.StatusUnsupportedMedia…+strings.ToLower`** — 24 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `strings.ToLower` | `███·······` | 8 of 24 | 6.8× |
|  | `chi.*Mux.Get` | `███·······` | 7 of 24 | 4.4× |
|  | `net/http.HandlerFunc` | `████████··` | 20 of 24 | 4.4× |
| flow ×20 | `funclit` | `█████████·` | 22 of 24 | 4.0× |
|  | `range` | `████······` | 10 of 24 | 2.9× |
| cotags ×15 | `http.StatusUnsupportedMedia…+Header.Get` | `████······` | 10 of 24 | 7.6× |
|  | `w.Header+http.Handler` | `███·······` | 7 of 24 | 6.7× |
|  | `http.StatusUnsupportedMedia…+w.WriteHeader` | `███·······` | 6 of 24 | 6.5× |
|  | `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` | `█████·····` | 12 of 24 | 5.4× |
|  | `http.StatusUnsupportedMedia…+chi.RouteContext` | `████████··` | 19 of 24 | 5.0× |
| role ×15 | `orchestrator` | `███·······` | 8 of 24 | 2.3× |

**`cw.ResponseWriter+cw.writer`** — 21 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| cotags ×15 | `buf.String+bytes.Buffer` | `████······` | 9 of 21 | 6.5× |
|  | `b.ResponseWriter+b.discard` | `██████····` | 13 of 21 | 3.9× |
| role ×15 | `orchestrator` | `███·······` | 7 of 21 | 2.3× |

**`h.handler+n.endpoints`** — 21 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| flow ×20 | `for` | `███·······` | 7 of 21 | 4.4× |
|  | `range` | `████······` | 8 of 21 | 2.7× |
| package ×10 | `chi` | `██████████` | 21 of 21 | 2.4× |

**`http.StatusUnsupportedMedia…+Header.Get+context.WithValue`** — 17 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `context.WithValue` | `█████·····` | 8 of 17 | 9.6× |
|  | `chi.*Mux.Get` | `████······` | 7 of 17 | 6.3× |
|  | `net/http.HandlerFunc` | `█████████·` | 16 of 17 | 4.9× |
| flow ×20 | `funclit` | `█████████·` | 16 of 17 | 4.1× |
|  | `range` | `████······` | 6 of 17 | 2.5× |
| cotags ×15 | `context.WithValue+r.WithContext` | `█████·····` | 9 of 17 | 11× |
|  | `http.StatusUnsupportedMedia…+w.WriteHeader` | `███·······` | 5 of 17 | 7.7× |
|  | `netip.Addr+context.WithValue` | `██████····` | 10 of 17 | 7.2× |
|  | `w.Header+http.Handler` | `███·······` | 5 of 17 | 6.7× |
|  | `http.StatusUnsupportedMedia…+Header.Get` | `████······` | 6 of 17 | 6.5× |
| role ×15 | `orchestrator` | `████······` | 7 of 17 | 2.8× |

_13 further concepts are modeled and not described._

### Which concepts share a function

`++` at least four times chance, `+` at least twice, `−` at most half, `never` not once. A blank cell is ordinary company — near chance, which is not culture.

| | `URL.RawPath+chi.RouteContext` | `b.ResponseWriter+b.discard` | `buf.String+bytes.Buffer` | `context.WithValue+r.WithContext` | `cw.ResponseWriter+cw.writer` | `fmt.Sprintf+r.Context` | `h.handler+n.endpoints` | `http.StatusUnsupportedMedia…+Header.Get` | `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` | `http.StatusUnsupportedMedia…+chi.RouteContext` | `http.StatusUnsupportedMedia…+strings.ToLower` | `http.StatusUnsupportedMedia…+w.WriteHeader` | `mx.handle+chi.*Mux.handle` | `mx.inline+mx.handler` | `mx.tree+rctx.RoutePath` | `netip.Addr+context.WithValue` | `r.Context+http.Handler` | `rctx.URLParams+URLParams.Keys` | `strings.Cut+chi.*Mux.Get` | `strings.TrimSpace+space` |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **`b.ResponseWriter+b.discard`** |  | | | | | | | | | | | | | | | | | | | |
| **`buf.String+bytes.Buffer`** |  |  | | | | | | | | | | | | | | | | | | |
| **`context.WithValue+r.WithContext`** |  |  |  | | | | | | | | | | | | | | | | | |
| **`cw.ResponseWriter+cw.writer`** |  | + | ++ |  | | | | | | | | | | | | | | | | |
| **`fmt.Sprintf+r.Context`** |  |  |  |  |  | | | | | | | | | | | | | | | |
| **`h.handler+n.endpoints`** |  | never |  |  |  | + | | | | | | | | | | | | | | |
| **`http.StatusUnsupportedMedia…+Header.Get`** |  |  |  |  | + |  |  | | | | | | | | | | | | | |
| **`http.StatusUnsupportedMedia…+Header.Get+context.WithValue`** |  |  |  | ++ |  |  |  | ++ | | | | | | | | | | | | |
| **`http.StatusUnsupportedMedia…+chi.RouteContext`** | ++ | − |  | ++ |  |  | − | + | ++ | | | | | | | | | | | |
| **`http.StatusUnsupportedMedia…+strings.ToLower`** | + |  |  | + |  |  |  | ++ | ++ | ++ | | | | | | | | | | |
| **`http.StatusUnsupportedMedia…+w.WriteHeader`** |  |  |  |  |  |  |  | ++ | ++ | ++ | ++ | | | | | | | | | |
| **`mx.handle+chi.*Mux.handle`** |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | |
| **`mx.inline+mx.handler`** |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | |
| **`mx.tree+rctx.RoutePath`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | |
| **`netip.Addr+context.WithValue`** |  |  |  | ++ |  |  |  |  | ++ | + | + |  |  |  |  | | | | | |
| **`r.Context+http.Handler`** |  |  | ++ |  | + |  |  |  | + | + | + |  |  |  |  | ++ | | | | |
| **`rctx.URLParams+URLParams.Keys`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | |
| **`strings.Cut+chi.*Mux.Get`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | |
| **`strings.TrimSpace+space`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | |
| **`w.Header+http.Handler`** |  |  |  |  |  |  |  |  | ++ | ++ | ++ | ++ |  |  |  |  | ++ |  |  |  |

### What travels with what

Co-occurrence measured against chance across every function. Only relationships at least twice — or at most half — as common as chance are reported; near-chance company is not culture. Each kind is listed separately, because there are far more call tokens than concepts and one shared list is all calls. Within a kind, strongest first means lift weighted by how many functions carry it — a 100× relationship holding for three functions is a weaker finding than a 30× one holding for thirty.

**Together more than chance — tag~tag**

- 9 of 9 `context.WithValue+r.WithContext` functions also `netip.Addr+context.WithValue` — 12× chance
- 9 of 9 `context.WithValue+r.WithContext` functions also `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` — 11× chance
- 16 of 17 `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` functions also `http.StatusUnsupportedMedia…+chi.RouteContext` — 5.9× chance
- 10 of 10 `http.StatusUnsupportedMedia…+Header.Get` functions also `http.StatusUnsupportedMedia…+strings.ToLower` — 7.6× chance
- 19 of 24 `http.StatusUnsupportedMedia…+strings.ToLower` functions also `http.StatusUnsupportedMedia…+chi.RouteContext` — 5.0× chance
- 10 of 15 `netip.Addr+context.WithValue` functions also `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` — 7.2× chance
- _28 more not listed_

**Together more than chance — tag~role**

- 5 of 10 `http.StatusUnsupportedMedia…+Header.Get` functions also `orchestrator` — 3.4× chance
- 7 of 17 `http.StatusUnsupportedMedia…+Header.Get+context.WithValue` functions also `orchestrator` — 2.8× chance
- 10 of 29 `http.StatusUnsupportedMedia…+chi.RouteContext` functions also `orchestrator` — 2.3× chance
- 5 of 12 `buf.String+bytes.Buffer` functions also `orchestrator` — 2.8× chance
- 8 of 24 `http.StatusUnsupportedMedia…+strings.ToLower` functions also `orchestrator` — 2.3× chance
- 4 of 9 `context.WithValue+r.WithContext` functions also `orchestrator` — 3.0× chance
- _4 more not listed_

**Together more than chance — tag~call**

- 13 of 13 `mx.handle+chi.*Mux.handle` functions also `chi.*Mux.handle` — 14× chance
- 8 of 9 `context.WithValue+r.WithContext` functions also `context.WithValue` — 18× chance
- 10 of 10 `fmt.Sprintf+r.Context` functions also `fmt.Sprintf` — 14× chance
- 8 of 10 `http.StatusUnsupportedMedia…+Header.Get` functions also `strings.ToLower` — 16× chance
- 5 of 5 `strings.Cut+chi.*Mux.Get` functions also `strings.Cut` — 30× chance
- 4 of 4 `strings.TrimSpace+space` functions also `strings.TrimSpace` — 37× chance
- _40 more not listed_

**Apart more than chance — tag~tag**

- **no** `b.ResponseWriter+b.discard` function has `h.handler+n.endpoints` — chance alone would give about 3 of 29
- 1 of 29 `b.ResponseWriter+b.discard` functions also `http.StatusUnsupportedMedia…+chi.RouteContext` — 0.2× chance
- 1 of 21 `h.handler+n.endpoints` functions also `http.StatusUnsupportedMedia…+chi.RouteContext` — 0.3× chance

**Apart more than chance — tag~role**

- 1 of 29 `b.ResponseWriter+b.discard` functions also `utility` — 0.2× chance
- 1 of 29 `http.StatusUnsupportedMedia…+chi.RouteContext` functions also `utility` — 0.2× chance
- 1 of 6 `mx.tree+rctx.RoutePath` functions also `leaf` — 0.2× chance
- 1 of 24 `http.StatusUnsupportedMedia…+strings.ToLower` functions also `utility` — 0.3× chance
- 2 of 6 `mx.inline+mx.handler` functions also `leaf` — 0.5× chance
- 1 of 21 `cw.ResponseWriter+cw.writer` functions also `utility` — 0.3× chance

**Apart more than chance — tag~call**

- 1 of 29 `b.ResponseWriter+b.discard` functions also `net/http.HandlerFunc` — 0.2× chance
- 1 of 21 `h.handler+n.endpoints` functions also `net/http.HandlerFunc` — 0.2× chance
- 2 of 21 `cw.ResponseWriter+cw.writer` functions also `net/http.HandlerFunc` — 0.5× chance

### Functions drifting from their own concept

These carry a tag but look nothing like the other functions carrying it. Typicality is measured against the concept's own median, so a genuinely varied concept lowers its own bar and a tight one can flag nobody.

| Function | Concept | Typicality | Concept median | |
|---|---|---:|---:|---|
| `chi.*Mux.Mount` <br/>`mux.go:295` | `mx.handle+chi.*Mux.handle` | `0.17` | `0.86` | no near-duplicate |
| `chi.*Mux.routeHTTP` <br/>`mux.go:447` | `URL.RawPath+chi.RouteContext` | `0.19` | `0.70` | no near-duplicate |
| `chi.*Mux.routeHTTP` <br/>`mux.go:447` | `http.StatusUnsupportedMedia…+chi.RouteContext` | `0.21` | `0.54` | no near-duplicate |
| `middleware.*Compressor.Handler` <br/>`middleware/compress.go:199` | `b.ResponseWriter+b.discard` | `0.21` | `0.47` | no near-duplicate |
| `middleware.*Compressor.selectEncoder` <br/>`middleware/compress.go:223` | `b.ResponseWriter+b.discard` | `0.23` | `0.47` | no near-duplicate |
| `chi.*Mux.Method` <br/>`mux.go:127` | `mx.handle+chi.*Mux.handle` | `0.32` | `0.86` |  |
| `chi.*Mux.Handle` <br/>`mux.go:109` | `mx.handle+chi.*Mux.handle` | `0.34` | `0.86` |  |

A row marked _no near-duplicate_ appears in no reported pair: nothing else in this report explains it, which makes it drift rather than duplication.

---

## Match #1 — Code-shape: `0.8396`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/recoverer.go:132` | `middleware.prettyStack.decorateFuncCallLine` | `(string, bool, int) (string, error)` | buf.String+bytes.Buffer 0.74, cw.ResponseWriter+cw.writer 0.67, b.ResponseWriter+b.discard 0.42 |
| **B** | `middleware/recoverer.go:172` | `middleware.prettyStack.decorateSourceLine` | `(string, bool, int) (string, error)` | buf.String+bytes.Buffer 0.74, cw.ResponseWriter+cw.writer 0.67, b.ResponseWriter+b.discard 0.42 |

**Profile A:** `cw.ResponseWriter+cw.writer` 0.77, `buf.String+bytes.Buffer` 0.23 (dominance)

**Profile B:** `cw.ResponseWriter+cw.writer` 0.78, `buf.String+bytes.Buffer` 0.22 (dominance)

**Code similarity:** `ast 0.74  flow 1.00  nesting 0.90  sig 1.00  size 0.99`

**Evidence:** `808.01` (shape 787.95, concept 4.93, call 15.13)

**Trophic:** `0.71`

**Shared structure:**

- `14.36` — `flow:param→call:cW`
- `13.47` — `do(call:cW)`
- `7.76` — `flow:call:LastIndex→cond`

**Structural overlap:** `0.95` (merge-worthy)

- share 6 callees: [buf.String, cW, errors.New, string, strings.Index, strings.LastIndex]
- share 1 callers: [middleware.prettyStack.decorateLine]
- overlapping call-graph neighborhoods (1.00): 5 shared
- share patterns: [b.ResponseWriter+b.discard, buf.String+bytes.Buffer, cw.ResponseWriter+cw.writer]
- both are leaf functions
- same package
- callers do related work (1.00): [strings.TrimSpace+space, buf.String+bytes.Buffer, cw.ResponseWriter+cw.writer]
- same visibility
- same receiver type: prettyStack
- called from same packages: [middleware]
- call into same packages: [middleware]

---

## Match #2 — Code-shape: `0.8367`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tree.go:559` | `chi.*node.findEdge` | `(nodeTyp, byte) (*node)` | h.handler+n.endpoints 0.65 |
| **B** | `tree.go:850` | `chi.nodes.findEdge` | `(byte) (*node)` | h.handler+n.endpoints 0.52 |

**Profile A:** `h.handler+n.endpoints` 1.00 (dominance)

**Profile B:** `h.handler+n.endpoints` 1.00 (dominance)

**Code similarity:** `ast 0.85  flow 0.96  nesting 0.74  sig 0.67  size 0.80`

**Evidence:** `510.89` (shape 509.55, concept 1.34, call 0.00)

**Trophic:** `0.92`

**Shared structure:**

- `9.55` — `assign=(bin)`
- `4.28` — `seq[ assign:=(call:len) ; assign:=(lit:INT) ]`
- `4.28` — `seq[ assign=(bin) ; if(bin:>(id,sel)) ]`

**Structural overlap:** `0.65` (merge-worthy)

- share 1 callees: [len]
- share patterns: [h.handler+n.endpoints]
- both are leaf functions
- same package
- same visibility
- both are methods, on *node and nodes

---

## Match #3 — Code-shape: `0.9163`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `mux.go:203` | `chi.*Mux.NotFound` | `(http.HandlerFunc)` | mx.inline+mx.handler 0.52 |
| **B** | `mux.go:223` | `chi.*Mux.MethodNotAllowed` | `(http.HandlerFunc)` | mx.inline+mx.handler 0.53 |

**Profile A:** `mx.inline+mx.handler` 1.00 (dominance)

**Profile B:** `mx.inline+mx.handler` 1.00 (dominance)

**Code similarity:** `ast 0.86  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `336.40` (shape 321.28, concept 1.98, call 13.14)

**Trophic:** `0.89`

**Shared structure:**

- `4.98` — `assign:=(id)`
- `4.82` — `assign=(sel)`
- `4.28` — `seq[ assign:=(id) ; if(bin:&&(sel,bin)) ]`

**Structural overlap:** `0.95` (merge-worthy)

- share 3 callees: [Chain, HandlerFunc, m.updateSubRoutes]
- share 1 callers: [chi.*Mux.Mount]
- overlapping call-graph neighborhoods (1.00): 12 shared
- share patterns: [mx.inline+mx.handler]
- both are orchestrator functions
- same package
- callers do related work (1.00): [rctx.URLParams+URLParams.Keys, mx.tree+rctx.RoutePath, fmt.Sprintf+r.Context, mx.handle+chi.*Mux.handle, h.handler+n.endpoints, http.StatusUnsupportedMedia…+chi.RouteContext]
- callees do related work (1.00): [mx.tree+rctx.RoutePath]
- same visibility
- same receiver type: Mux
- called from same packages: [chi]
- call into same packages: [chi]

---

## Match #4 — Code-shape: `0.7972`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/route_headers.go:48` | `middleware.HeaderRouter.Route` | `(string, string, func(next http.Handler) http.Handler) (HeaderRouter)` | http.StatusUnsupportedMedia…+Header.Get 0.50, http.StatusUnsupportedMedia…+strings.ToLower 0.50 |
| **B** | `middleware/route_headers.go:58` | `middleware.HeaderRouter.RouteAny` | `(string, []string, func(next http.Handler) http.Handler) (HeaderRouter)` | http.StatusUnsupportedMedia…+Header.Get 0.51, http.StatusUnsupportedMedia…+strings.ToLower 0.50 |

**Profile A:** `http.StatusUnsupportedMedia…+Header.Get` 0.83, `http.StatusUnsupportedMedia…+strings.ToLower` 0.17 (dominance)

**Profile B:** `http.StatusUnsupportedMedia…+Header.Get` 0.83, `http.StatusUnsupportedMedia…+strings.ToLower` 0.17 (dominance)

**Code similarity:** `ast 0.79  flow 0.82  nesting 1.00  sig 0.75  size 0.74`

**Evidence:** `251.43` (shape 241.07, concept 2.83, call 7.53)

**Trophic:** `0.82`

**Shared structure:**

- `4.28` — `seq[ assign:=(index) ; if(bin:==(id,nil)) ]`
- `4.28` — `seq[ assign=(call:ToLower) ; assign:=(index) ]`
- `3.88` — `seq[ assign=(call:append) ; return(id) ]`

**Structural overlap:** `0.75` (merge-worthy)

- share 3 callees: [NewPattern, append, strings.ToLower]
- overlapping call-graph neighborhoods (1.00): 1 shared
- share patterns: [http.StatusUnsupportedMedia…+Header.Get, http.StatusUnsupportedMedia…+strings.ToLower]
- both are leaf functions
- same package
- same visibility
- same receiver type: HeaderRouter
- call into same packages: [middleware]

---

## Match #5 — Code-shape: `0.6704`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/content_encoding.go:10` | `middleware.AllowContentEncoding` | `(...string) (func(next http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.71, http.StatusUnsupportedMedia…+Header.Get 0.68, http.StatusUnsupportedMedia…+w.WriteHeader 0.65, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.61, +2 more |
| **B** | `middleware/content_type.go:20` | `middleware.AllowContentType` | `(...string) (func(http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.72, http.StatusUnsupportedMedia…+Header.Get 0.69, http.StatusUnsupportedMedia…+w.WriteHeader 0.65, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.63, +3 more |

**Profile A:** `http.StatusUnsupportedMedia…+Header.Get` 0.52, `http.StatusUnsupportedMedia…+w.WriteHeader` 0.47 (coalition)

**Profile B:** `http.StatusUnsupportedMedia…+Header.Get` 0.93 (dominance)

**Code similarity:** `ast 0.62  flow 0.98  nesting 0.98  sig 0.33  size 0.98`

**Evidence:** `407.00` (shape 387.49, concept 11.25, call 8.27)

**Trophic:** `0.71`

**Shared structure:**

- `4.28` — `range{ call:TrimSpace call:ToLower }`
- `4.28` — `if(bin:==(sel,lit:INT))`
- `3.86` — `return()`

**Structural overlap:** `0.63` (merge-worthy)

- share 7 callees: [http.HandlerFunc, len, make, next.ServeHTTP, strings.ToLower, strings.TrimSpace, w.WriteHeader]
- share patterns: [http.StatusUnsupportedMedia…+Header.Get, http.StatusUnsupportedMedia…+Header.Get+context.WithValue, http.StatusUnsupportedMedia…+chi.RouteContext, http.StatusUnsupportedMedia…+strings.ToLower, http.StatusUnsupportedMedia…+w.WriteHeader, strings.TrimSpace+space]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions

---

## Match #6 — Code-shape: `0.7282`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/strip.go:14` | `middleware.StripSlashes` | `(http.Handler) (http.Handler)` | http.StatusUnsupportedMedia…+chi.RouteContext 0.56, URL.RawPath+chi.RouteContext 0.51 |
| **B** | `middleware/strip.go:41` | `middleware.RedirectSlashes` | `(http.Handler) (http.Handler)` | http.StatusUnsupportedMedia…+chi.RouteContext 0.58, URL.RawPath+chi.RouteContext 0.55, fmt.Sprintf+r.Context 0.51, http.StatusUnsupportedMedia…+strings.ToLower 0.42 |

**Profile A:** `URL.RawPath+chi.RouteContext` 0.63, `http.StatusUnsupportedMedia…+chi.RouteContext` 0.37 (dominance)

**Profile B:** `fmt.Sprintf+r.Context` 1.00 (dominance)

**Code similarity:** `ast 0.56  flow 0.97  nesting 0.96  sig 1.00  size 0.83`

**Evidence:** `383.99` (shape 379.26, concept 3.07, call 1.65)

**Trophic:** `0.70`

**Shared structure:**

- `6.73` — `flow:call:RouteContext→cond`
- `4.98` — `if(bin:&&(bin,bin))`
- `4.82` — `assign=(sel)`

**Structural overlap:** `0.52` (merge-worthy)

- share 5 callees: [chi.RouteContext, http.HandlerFunc, len, next.ServeHTTP, r.Context]
- share patterns: [URL.RawPath+chi.RouteContext, http.StatusUnsupportedMedia…+chi.RouteContext]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions

---

## Match #7 — Code-shape: `0.7918`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/clean_path.go:12` | `middleware.CleanPath` | `(http.Handler) (http.Handler)` | URL.RawPath+chi.RouteContext 0.54, http.StatusUnsupportedMedia…+chi.RouteContext 0.52 |
| **B** | `middleware/get_head.go:10` | `middleware.GetHead` | `(http.Handler) (http.Handler)` | URL.RawPath+chi.RouteContext 0.59, http.StatusUnsupportedMedia…+chi.RouteContext 0.56, http.StatusUnsupportedMedia…+strings.ToLower 0.44 |

**Profile A:** `URL.RawPath+chi.RouteContext` 0.68, `http.StatusUnsupportedMedia…+chi.RouteContext` 0.32 (dominance)

**Profile B:** `URL.RawPath+chi.RouteContext` 0.69, `http.StatusUnsupportedMedia…+chi.RouteContext` 0.30 (dominance)

**Code similarity:** `ast 0.68  flow 0.98  nesting 0.79  sig 1.00  size 0.70`

**Evidence:** `256.35` (shape 251.62, concept 3.08, call 1.65)

**Trophic:** `0.73`

**Shared structure:**

- `4.82` — `assign=(sel)`
- `4.28` — `seq[ assign:=(call:RouteContext) ; assign:=(sel) ]`
- `3.88` — `seq[ assign:=(sel) ; if(bin:==(id,lit:STRING)) ]`

**Structural overlap:** `0.58` (merge-worthy)

- share 4 callees: [chi.RouteContext, http.HandlerFunc, next.ServeHTTP, r.Context]
- share patterns: [URL.RawPath+chi.RouteContext, http.StatusUnsupportedMedia…+chi.RouteContext]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions

---

## Match #8 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/wrap_writer.go:147` | `middleware.*flushWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.65 |
| **B** | `middleware/wrap_writer.go:172` | `middleware.*flushHijackWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.65 |

**Kind:** interface implementations — both implement `Flush()` on `*flushWriter` and `*flushHijackWriter`, in package `middleware`

**Profile A:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Profile B:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `82.86` (shape 81.38, concept 1.48, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.59` — `seq[ assign:=(assert) ; do(call:Flush) ]`
- `3.59` — `seq[ assign=(true) ; assign:=(assert) ]`
- `3.37` — `do(call:Flush)`

**Structural overlap:** `0.68` (merge-worthy)

- share 1 callees: [fl.Flush]
- share patterns: [b.ResponseWriter+b.discard]
- both are leaf functions
- same package
- same visibility
- both are methods, on *flushWriter and *flushHijackWriter

---

## Match #9 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/wrap_writer.go:147` | `middleware.*flushWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.65 |
| **B** | `middleware/wrap_writer.go:194` | `middleware.*httpFancyWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.65 |

**Kind:** interface implementations — both implement `Flush()` on `*flushWriter` and `*httpFancyWriter`, in package `middleware`

**Profile A:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Profile B:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `82.86` (shape 81.38, concept 1.48, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.59` — `seq[ assign:=(assert) ; do(call:Flush) ]`
- `3.59` — `seq[ assign=(true) ; assign:=(assert) ]`
- `3.37` — `do(call:Flush)`

**Structural overlap:** `0.68` (merge-worthy)

- share 1 callees: [fl.Flush]
- share patterns: [b.ResponseWriter+b.discard]
- both are leaf functions
- same package
- same visibility
- both are methods, on *flushWriter and *httpFancyWriter

---

## Match #10 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/wrap_writer.go:172` | `middleware.*flushHijackWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.65 |
| **B** | `middleware/wrap_writer.go:194` | `middleware.*httpFancyWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.65 |

**Kind:** interface implementations — both implement `Flush()` on `*flushHijackWriter` and `*httpFancyWriter`, in package `middleware`

**Profile A:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Profile B:** `b.ResponseWriter+b.discard` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `82.86` (shape 81.38, concept 1.48, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `3.59` — `seq[ assign:=(assert) ; do(call:Flush) ]`
- `3.59` — `seq[ assign=(true) ; assign:=(assert) ]`
- `3.37` — `do(call:Flush)`

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

### Family 1 — 4 members, every pair `>= 0.62` code-shape, evidence `1311`

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

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `middleware/clean_path.go:12` | `middleware.CleanPath` | `(http.Handler) (http.Handler)` | URL.RawPath+chi.RouteContext 0.54, http.StatusUnsupportedMedia…+chi.RouteContext 0.52 |
| `middleware/get_head.go:10` | `middleware.GetHead` | `(http.Handler) (http.Handler)` | URL.RawPath+chi.RouteContext 0.59, http.StatusUnsupportedMedia…+chi.RouteContext 0.56, http.StatusUnsupportedMedia…+strings.ToLower 0.44 |
| `middleware/strip.go:14` | `middleware.StripSlashes` | `(http.Handler) (http.Handler)` | http.StatusUnsupportedMedia…+chi.RouteContext 0.56, URL.RawPath+chi.RouteContext 0.51 |
| `middleware/strip.go:41` | `middleware.RedirectSlashes` | `(http.Handler) (http.Handler)` | http.StatusUnsupportedMedia…+chi.RouteContext 0.58, URL.RawPath+chi.RouteContext 0.55, fmt.Sprintf+r.Context 0.51, http.StatusUnsupportedMedia…+strings.ToLower 0.42 |

### Family 2 — 3 members, every pair `>= 0.55` code-shape, evidence `579`  (1 edge scored here)

```mermaid
flowchart LR
    m0["middleware.ContentCharset"]
    m1["middleware.AllowContentEncoding"]
    m2["middleware.AllowContentType"]
    m0 --- m1
    m0 --- m2
    m1 --- m2
```

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `middleware/content_charset.go:11` | `middleware.ContentCharset` | `(...string) (func(next http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.71, http.StatusUnsupportedMedia…+Header.Get 0.68, http.StatusUnsupportedMedia…+w.WriteHeader 0.65, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.59, +1 more |
| `middleware/content_encoding.go:10` | `middleware.AllowContentEncoding` | `(...string) (func(next http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.71, http.StatusUnsupportedMedia…+Header.Get 0.68, http.StatusUnsupportedMedia…+w.WriteHeader 0.65, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.61, +2 more |
| `middleware/content_type.go:20` | `middleware.AllowContentType` | `(...string) (func(http.Handler) http.Handler)` | http.StatusUnsupportedMedia…+strings.ToLower 0.72, http.StatusUnsupportedMedia…+Header.Get 0.69, http.StatusUnsupportedMedia…+w.WriteHeader 0.65, http.StatusUnsupportedMedia…+Header.Get+context.WithValue 0.63, +3 more |

### Family 3 — 4 members, every pair `>= 1.00` code-shape, evidence `497`, interface implementations of `Flush()`, in package `middleware`

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
| `middleware/wrap_writer.go:147` | `middleware.*flushWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.65 |
| `middleware/wrap_writer.go:172` | `middleware.*flushHijackWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.65 |
| `middleware/wrap_writer.go:194` | `middleware.*httpFancyWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.65 |
| `middleware/wrap_writer.go:239` | `middleware.*http2FancyWriter.Flush` | `()` | b.ResponseWriter+b.discard 0.65 |

### Family 4 — 3 members, every pair `>= 0.64` code-shape, evidence `275`

```mermaid
flowchart LR
    m0["middleware.*compressResponseWriter.Hijack"]
    m1["middleware.*compressResponseWriter.Push"]
    m2["middleware.*compressResponseWriter.Close"]
    m0 --- m1
    m0 --- m2
    m1 --- m2
```

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `middleware/compress.go:365` | `middleware.*compressResponseWriter.Hijack` | `() (net.Conn, *bufio.ReadWriter, error)` | b.ResponseWriter+b.discard 0.68, cw.ResponseWriter+cw.writer 0.53 |
| `middleware/compress.go:372` | `middleware.*compressResponseWriter.Push` | `(string, *http.PushOptions) (error)` | b.ResponseWriter+b.discard 0.64, cw.ResponseWriter+cw.writer 0.53 |
| `middleware/compress.go:379` | `middleware.*compressResponseWriter.Close` | `() (error)` | b.ResponseWriter+b.discard 0.59, cw.ResponseWriter+cw.writer 0.53 |

### Family 5 — 3 members, every pair `>= 1.00` code-shape, evidence `201`, interface implementations of `Hijack() (net.Conn, *bufio.ReadWriter, error)`, in package `middleware`

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
| `middleware/wrap_writer.go:160` | `middleware.*hijackWriter.Hijack` | `() (net.Conn, *bufio.ReadWriter, error)` | b.ResponseWriter+b.discard 0.58 |
| `middleware/wrap_writer.go:178` | `middleware.*flushHijackWriter.Hijack` | `() (net.Conn, *bufio.ReadWriter, error)` | b.ResponseWriter+b.discard 0.58 |
| `middleware/wrap_writer.go:200` | `middleware.*httpFancyWriter.Hijack` | `() (net.Conn, *bufio.ReadWriter, error)` | b.ResponseWriter+b.discard 0.58 |

_1 more families not listed._


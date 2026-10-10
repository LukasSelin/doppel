# gin

HTTP framework; a small core surrounded by generated-looking binding and render variants

**What this rung shows:** family clones — the case corroborated ranking was tuned to separate

| | |
|---|---|
| Corpus | [gin](https://github.com/gin-gonic/gin) |
| Pinned at | `v1.12.0` (`73726dc606796a025971fe451f0aa6f1b9b847f6`) |
| Project since | 2014 |
| doppel | `a7601b6` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`; CI regenerates on every push to master.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Learning concept vocabulary...
Lexicon: 46 concepts (6 seeded, 40 emergent), 1266/3382 features above 182 df, 76 functions unlabeled
Generating concept documents...
Calibration: rate 0.01 over 20000 null pairs -> threshold 0.40, struct-min 0.47, family-min 0.40
Found 497 functions. Retrieving candidates...
Retrieval: shape 209, concept 1637, call 609 -> 2132 unique pairs
  concept-only 64.3%  call-only 17.2%  suppressed-shape functions: 0  large identity buckets: 0  surviving labels: 1934
Running structural comparison on 2132 pairs...
  Concept views: 70 of 2132 compared pairs disagree with the taxonomy (4 vocabulary the tree misses, 66 kinship the vocabularies lack)
  687 pairs remain after struct-min=0.47 filter
Culture: 37 concepts modeled, 153 associations, 31 unusual realizations
Habitats: 5 modeled, 15 misfits (0 excused by subsystem), 1 subsystems; most uniform binding (norm 0.92), most diverse json (norm 0.63)
Conventions: strongest c.MustBindWith+gin.*Context.MustBindWith (1.00), loosest gin.*Context.Header+gin.*Context.Set (0.22)
Ecosystems: 443 profiled (351 dominance, 74 coalition, 0 conflict, 18 weak)
Families: 33 over 62 components, 180 functions in a family, 249 edges completed
```

# Code Similarity Report

**Functions analyzed:** 497 | **Threshold:** 0.40 | **Pairs found:** 10

---

## What doppel sees

**497 functions** across **7 packages** — test functions excluded. Structural roles: 361 leaf, 57 orchestrator, 17 passthrough, 62 utility.

### Concepts

Two pictures of the same vocabulary. The first is what doppel **searched with**: an authored tree of fourteen seed practices, each leaf showing how many functions here ended up in a concept that seed grew. It is the same shape on every corpus, which is what makes it the one concept picture two runs can be compared on. The second is what this corpus **turned out to have**: concepts learned from the code itself, named after the evidence that identified them, hung from that same interior — so two functions under one *branch* score partial credit rather than nothing. Counts are members; membership is graded, and a function can carry several.

**What doppel looked for, and how much of it grew here.**

```mermaid
flowchart LR
    s0(["concept"])
    s1(["io_operation"])
    s2(["remote_io"])
    s3["http_call<br/>absent"]
    s4["grpc_call<br/>absent"]
    s5(["data_store_access"])
    s6["db_access<br/>absent"]
    s7["caching<br/>39"]
    s8["transaction<br/>absent"]
    s9["file_io<br/>11"]
    s10["logging<br/>3"]
    s11(["data_transformation"])
    s12["mapping<br/>absent"]
    s13["validation<br/>22"]
    s14["serialization<br/>33"]
    s15(["control_flow"])
    s16["concurrency<br/>150"]
    s17(["fault_tolerance"])
    s18["retry<br/>absent"]
    s19["circuit_breaker<br/>absent"]
    s20(["error_handling"])
    s21["error_wrapping<br/>absent"]
    s0 --> s1
    s1 --> s2
    s2 --> s3
    s2 --> s4
    s1 --> s5
    s5 --> s6
    s5 --> s7
    s5 --> s8
    s1 --> s9
    s1 --> s10
    s0 --> s11
    s11 --> s12
    s11 --> s13
    s11 --> s14
    s0 --> s15
    s15 --> s16
    s15 --> s17
    s17 --> s18
    s17 --> s19
    s0 --> s20
    s20 --> s21
    classDef good fill:#d7ecd9,color:#1b3d20
    classDef warn fill:#fbeecb,color:#4a3a12
    classDef hot fill:#f7d6d6,color:#4a1c1c
    class s3,s4,s6,s8,s12,s18,s19,s21 hot
```

**What it learned instead.**

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
    c8["URL.Path+c.Writer<br/>11"]
    c9["binding+nil<br/>150"]
    c10["c.formCache+c.queryCache<br/>39"]
    c11["c.requestHeader+gin.*Context.requestHeader<br/>53"]
    c12["delims.Left+delims.Right+engine.SetHTMLTemplate<br/>43"]
    c13["engine.MaxMultipartMemory+c.engine<br/>82"]
    c14["group.calculateAbsolutePath+group.engine<br/>22"]
    c15["json.Marshal+json.MarshalIndent<br/>33"]
    c16["n.nType+n.priority<br/>13"]
    c17["xml+runtime<br/>12"]
    c0 --> c1
    c1 --> c2
    c1 --> c3
    c0 --> c4
    c0 --> c5
    c5 --> c6
    c0 --> c7
    c1 --> c8
    c5 --> c9
    c3 --> c10
    c3 --> c11
    c4 --> c12
    c4 --> c13
    c1 --> c14
    c4 --> c15
    c1 --> c16
    c3 --> c17
```

The diagram draws the 3 largest concepts on each branch; **36 further concepts** are left out of the picture and listed in the table below.

**No practice here for** `circuit_breaker`, `db_access`, `error_wrapping`, `grpc_call`, `http_call`, `mapping`, `retry`, `transaction`. Concepts are learned from this corpus, so one can never be absent — it exists because functions carry it. These are the *seeds* the search started from that grew nothing: a direct answer to "does this codebase already do X".

| Concept | Functions | Convention |
|---|---:|---|
| `binding+nil` | 150 | `0.56` (settled) |
| `engine.MaxMultipartMemory+c.engine` | 82 | `0.56` (settled) |
| `c.requestHeader+gin.*Context.requestHeader` | 53 | `0.64` (settled) |
| `delims.Left+delims.Right+engine.SetHTMLTemplate` | 43 | `0.41` (loose) |
| `c.formCache+c.queryCache` | 39 | `0.47` (loose) |
| `json.Marshal+json.MarshalIndent` | 33 | `0.64` (settled) |
| `writermem.WriteHeaderNow+c.writermem` | 28 | `0.42` (loose) |
| `API.Marshal+bytesconv.StringToBytes` | 24 | `0.31` (loose) |
| `Request.URL+req.URL` | 22 | `0.44` (loose) |
| `delims.Left+delims.Right` | 22 | `0.32` (loose) |
| `group.calculateAbsolutePath+group.engine` | 22 | `0.44` (loose) |
| `value.Set+value.Type` | 20 | `0.44` (loose) |
| `gin+template` | 19 | `0.78` (unanimous) |
| `value.Addr+field.Tag` | 18 | `0.51` (settled) |
| `n.nType+n.priority` | 13 | `0.51` (settled) |
| `reflect.New+reflect.Array` | 12 | `0.32` (loose) |
| `xml+runtime` | 12 | `0.45` (loose) |
| `URL.Path+c.Writer` | 11 | `0.30` (loose) |
| `io.ReadAll+req.Body` | 11 | `0.40` (loose) |
| `tree.method+tree.root` | 11 | `0.33` (loose) |
| `reflect.Map+reflect.New` | 10 | `0.33` (loose) |
| `c.Next+c.Request` | 9 | `0.42` (loose) |
| `w.WriteHeaderNow+w.ResponseWriter` | 9 | `0.24` (loose) |
| `bytesconv.StringToBytes+json.API` | 8 | `0.23` (loose) |
| `cmp+httputil` | 8 | `0.44` (loose) |
| `field.Tag+Tag.Get` | 8 | `0.41` (loose) |
| `gin.*Context.Header+gin.*Context.Set` | 8 | `0.22` (loose) |
| `render.writeContentType+bytes` | 8 | `0.65` (settled) |
| `c.MustBindWith+gin.*Context.MustBindWith` | 7 | `1.00` (unanimous) |
| `c.ShouldBindWith+gin.*Context.ShouldBindWith` | 7 | `1.00` (unanimous) |
| `gin.IsDebugging+gin.debugPrint` | 7 | `0.28` (loose) |
| `gin.debugPrint+atomic` | 7 | `0.30` (loose) |
| `c.hasRequestContext+Request.Context` | 5 | `0.48` (loose) |
| `flag+atomic` | 5 | `0.57` (settled) |
| `http.Server+engine.Handler` | 5 | `0.93` (unanimous) |
| `strings.Split+reflect` | 5 | `0.45` (loose) |
| `subtle+base64` | 5 | `0.36` (loose) |
| `bytes.NewReader+bytes` | 4 | — |
| `bytesconv.BytesToString+bytesconv` | 4 | — |
| `c.Abort+gin.*Context.Abort` | 4 | — |
| `c.ShouldBindBodyWith+gin.*Context.ShouldBindBody…` | 4 | — |
| `fmt.Fprintf+runtime` | 4 | — |
| `reflect.Array+reflect.Slice` | 4 | — |
| `strings.TrimSpace+bytesconv` | 4 | — |
| `log` | 3 | — |
| `reflect.New+value.Type` | 2 | — |

Convention is how uniformly this corpus realizes a concept: `1.00` means every function carrying the tag does it the same way, and a low number means the tag covers several unrelated habits. A concept with fewer than five members is not modeled.

### Where the duplication is

Merge-worthy pairs are folded up to their packages: only pairs doppel judges worth consolidating are counted. An edge means two packages keep solving the same problem separately; a count on a node means the repetition is inside one package. Weights are **merge-worthy pairs**.

```mermaid
flowchart LR
    p0["fs"]
    p1["gin<br/>262 internal"]
    p0 ---|"1"| p1
```

### How settled each package is

A package with at least five functions gets a habitat model: doppel learns what is normal there and measures how surprising each member is against it. **Norm** is how uniform the package's practice is. A **misfit** is a function alien to its package *and* to the wider subsystem around it — one that fits its neighbours a directory up is normal for this codebase and is not reported.

```mermaid
flowchart TD
    h0["json<br/>24 functions · norm 0.63<br/>9 misfits"]
    h1["ginS<br/>25 functions · norm 0.75<br/>6 misfits"]
    h2["render<br/>42 functions · norm 0.85"]
    h3["gin<br/>324 functions · norm 0.89"]
    h4["binding<br/>79 functions · norm 0.92"]
    classDef good fill:#d7ecd9,color:#1b3d20
    classDef warn fill:#fbeecb,color:#4a3a12
    classDef hot fill:#f7d6d6,color:#4a1c1c
    class h1,h2,h3,h4 good
    class h0 warn
```

Most uniform is `binding` (norm `0.92`); most varied is `json` (norm `0.63`). 15 functions are alien to their package and to the subsystem around it.

### How these candidates were found

Three channels propose candidates independently — shared rare *structure*, shared *concepts*, shared *calls* — and their union is what gets compared. This run: **2132 candidate pairs** (shape 209, concept 1637, call 609), of which 17% arrived on call evidence alone and 64% on concept evidence alone. A pair sharing none of the three is never compared, however alike it looks.

The concept signal on each compared pair is read three ways — what the taxonomy asserts, what this corpus's frequencies say, and what the two sides' learned vocabularies share with no tree in between. On **70 of 2132** pairs the taxonomy and the vocabularies differ by at least 0.50: 4 where the vocabularies agree and the tree cannot see it, 66 where the tree asserts a kinship the vocabularies lack. Each such pair carries a `concept views` line saying which.

Each function is also an arena where its candidate concepts compete for its evidence. 443 functions reached an equilibrium: **351** settled on a single concept, **74** on a coalition, **0** hold concepts this corpus says do not go together.

### Corpus metrics

**Compression ratio:** `5.28`x — this corpus's canonical function bodies contain **17625 AST nodes** in total, which hash-cons (two nodes count as the same subtree exactly when their kind and every child match, all the way down) to **3336 distinct subtree shapes**; the ratio is nodes divided by shapes, always >= 1.0, and it never feeds any score.

**Nearest-neighbour code-shape:** of **497 functions**, **463** had a code-shape neighbour among the pairs retrieval actually scored — their best score's p50/p90/p99 are `0.48` / `1.00` / `1.00`, and 70% of them (323 of 463) already clear this run's threshold of `0.40`. This is **not an exhaustive nearest-neighbour search** (that would be a full pairwise comparison); it is bounded by the same three retrieval channels the pair list itself is bounded by, so the other 34 functions are excluded here as having no *scored* neighbour, not asserted to have none at all.

---

## Local practice

The vocabulary above says what a concept *is*. This says what one looks like when **this** codebase writes it — learned from the corpus, so it describes the house style rather than a rule from anywhere else.

### How this codebase writes each concept

Only what is **distinctive**. A feature earns a row by being carried by this concept's members at least twice as often as by the corpus at large — nearly every Go function has a `return` and an `if`, so prevalence alone would describe the language rather than this codebase. Weights are how much a channel counts toward whether a member looks normal — calls 40, control flow 20, co-occurring tags 15, role 15, package 10.

**`binding+nil`** — 150 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| cotags ×15 | `engine.MaxMultipartMemory+c.engine` | `███·······` | 51 of 150 | 2.1× |

**`engine.MaxMultipartMemory+c.engine`** — 82 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| cotags ×15 | `c.requestHeader+gin.*Context.requestHeader` | `██████····` | 48 of 82 | 5.5× |
|  | `c.formCache+c.queryCache` | `███·······` | 26 of 82 | 4.0× |
|  | `binding+nil` | `██████····` | 51 of 82 | 2.1× |

**`c.requestHeader+gin.*Context.requestHeader`** — 53 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| cotags ×15 | `engine.MaxMultipartMemory+c.engine` | `█████████·` | 48 of 53 | 5.5× |
|  | `c.formCache+c.queryCache` | `████······` | 20 of 53 | 4.8× |

**`delims.Left+delims.Right+engine.SetHTMLTemplate`** — 43 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| cotags ×15 | `Request.URL+req.URL` | `███·······` | 11 of 43 | 5.8× |
| role ×15 | `orchestrator` | `███·······` | 13 of 43 | 2.6× |

**`c.formCache+c.queryCache`** — 39 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| cotags ×15 | `c.requestHeader+gin.*Context.requestHeader` | `█████·····` | 20 of 39 | 4.8× |
|  | `engine.MaxMultipartMemory+c.engine` | `███████···` | 26 of 39 | 4.0× |

**`json.Marshal+json.MarshalIndent`** — 33 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| package ×10 | `json` | `███████···` | 24 of 33 | 15× |

_31 further concepts are modeled and not described._

### Which concepts share a function

`++` at least four times chance, `+` at least twice, `−` at most half, `never` not once. A blank cell is ordinary company — near chance, which is not culture.

_Showing 12 of 46 concepts — those in the strongest pairings, taken strongest first by lift weighted by how many functions it speaks for. Every cell between them is shown; the other 34 concepts are not on the grid._

| | `API.Marshal+bytesconv.StringToBytes` | `URL.Path+c.Writer` | `binding+nil` | `c.formCache+c.queryCache` | `c.requestHeader+gin.*Context.requestHeader` | `delims.Left+delims.Right+engine.SetHTMLTemplate` | `engine.MaxMultipartMemory+c.engine` | `gin+template` | `group.calculateAbsolutePath+group.engine` | `json.Marshal+json.MarshalIndent` | `reflect.New+reflect.Array` |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **`URL.Path+c.Writer`** |  | | | | | | | | | | |
| **`binding+nil`** |  | never | | | | | | | | | |
| **`c.formCache+c.queryCache`** |  |  |  | | | | | | | | |
| **`c.requestHeader+gin.*Context.requestHeader`** |  |  |  | ++ | | | | | | | |
| **`delims.Left+delims.Right+engine.SetHTMLTemplate`** |  |  | − | never | never | | | | | | |
| **`engine.MaxMultipartMemory+c.engine`** | never |  | + | ++ | ++ | never | | | | | |
| **`gin+template`** |  |  | never |  |  | + | never | | | | |
| **`group.calculateAbsolutePath+group.engine`** |  |  | never |  |  |  | never | + | | | |
| **`json.Marshal+json.MarshalIndent`** | ++ |  | − |  | never |  | never |  |  | | |
| **`reflect.New+reflect.Array`** |  |  | never |  |  |  |  |  |  |  | |
| **`writermem.WriteHeaderNow+c.writermem`** |  |  | − |  |  | + | never |  |  |  |  |

### What travels with what

Co-occurrence measured against chance across every function. Only relationships at least twice — or at most half — as common as chance are reported; near-chance company is not culture. Each kind is listed separately, because there are far more call tokens than concepts and one shared list is all calls. Within a kind, strongest first means lift weighted by how many functions carry it — a 100× relationship holding for three functions is a weaker finding than a 30× one holding for thirty.

**Together more than chance — tag~tag**

- 6 of 7 `gin.IsDebugging+gin.debugPrint` functions also `gin.debugPrint+atomic` — 61× chance
- 13 of 18 `value.Addr+field.Tag` functions also `value.Set+value.Type` — 18× chance
- 7 of 10 `reflect.Map+reflect.New` functions also `reflect.New+reflect.Array` — 29× chance
- 48 of 53 `c.requestHeader+gin.*Context.requestHeader` functions also `engine.MaxMultipartMemory+c.engine` — 5.5× chance
- 7 of 8 `bytesconv.StringToBytes+json.API` functions also `API.Marshal+bytesconv.StringToBytes` — 18× chance
- 7 of 10 `reflect.Map+reflect.New` functions also `value.Set+value.Type` — 17× chance
- _28 more not listed_

**Together more than chance — tag~role**

- 5 of 5 `http.Server+engine.Handler` functions also `orchestrator` — 8.7× chance
- 6 of 28 `writermem.WriteHeaderNow+c.writermem` functions also `passthrough` — 6.3× chance
- 5 of 8 `gin.*Context.Header+gin.*Context.Set` functions also `orchestrator` — 5.4× chance
- 13 of 43 `delims.Left+delims.Right+engine.SetHTMLTemplate` functions also `orchestrator` — 2.6× chance
- 4 of 8 `bytesconv.StringToBytes+json.API` functions also `orchestrator` — 4.4× chance
- 8 of 22 `delims.Left+delims.Right` functions also `utility` — 2.9× chance
- _8 more not listed_

**Together more than chance — tag~call**

- 7 of 7 `c.MustBindWith+gin.*Context.MustBindWith` functions also `gin.*Context.MustBindWith` — 55× chance
- 7 of 7 `c.ShouldBindWith+gin.*Context.ShouldBindWith` functions also `gin.*Context.ShouldBindWith` — 55× chance
- 8 of 10 `reflect.Map+reflect.New` functions also `reflect.ValueOf` — 40× chance
- 5 of 5 `http.Server+engine.Handler` functions also `gin.*Engine.isUnsafeTrustedProxies` — 83× chance
- 5 of 5 `http.Server+engine.Handler` functions also `gin.debugPrintError` — 83× chance
- 6 of 8 `gin.*Context.Header+gin.*Context.Set` functions also `gin.*Context.Header` — 47× chance
- _58 more not listed_

**Apart more than chance — tag~tag**

- **no** `delims.Left+delims.Right+engine.SetHTMLTemplate` function has `engine.MaxMultipartMemory+c.engine` — chance alone would give about 7 of 43
- **no** `binding+nil` function has `group.calculateAbsolutePath+group.engine` — chance alone would give about 7 of 150
- **no** `binding+nil` function has `gin+template` — chance alone would give about 6 of 150
- **no** `engine.MaxMultipartMemory+c.engine` function has `json.Marshal+json.MarshalIndent` — chance alone would give about 5 of 82
- **no** `engine.MaxMultipartMemory+c.engine` function has `writermem.WriteHeaderNow+c.writermem` — chance alone would give about 5 of 82
- **no** `c.requestHeader+gin.*Context.requestHeader` function has `delims.Left+delims.Right+engine.SetHTMLTemplate` — chance alone would give about 5 of 53
- _18 more not listed_

**Apart more than chance — tag~role**

- **no** `json.Marshal+json.MarshalIndent` function has `utility` — chance alone would give about 4 of 33
- **no** `json.Marshal+json.MarshalIndent` function has `orchestrator` — chance alone would give about 4 of 33
- **no** `http.Server+engine.Handler` function has `leaf` — chance alone would give about 4 of 5
- 8 of 150 `binding+nil` functions also `orchestrator` — 0.5× chance
- 1 of 82 `engine.MaxMultipartMemory+c.engine` functions also `orchestrator` — 0.1× chance
- 1 of 8 `gin.*Context.Header+gin.*Context.Set` functions also `leaf` — 0.2× chance
- _5 more not listed_

**Apart more than chance — tag~call**

- **no** `binding+nil` function has `gin.*RouterGroup.handle` — chance alone would give about 3 of 150
- **no** `engine.MaxMultipartMemory+c.engine` function has `render.writeContentType` — chance alone would give about 3 of 82
- **no** `binding+nil` function has `reflect.ValueOf` — chance alone would give about 3 of 150
- 1 of 150 `binding+nil` functions also `render.writeContentType` — 0.2× chance
- 1 of 150 `binding+nil` functions also `gin.*Context.Set` — 0.2× chance
- 2 of 150 `binding+nil` functions also `gin.debugPrint` — 0.4× chance

### Functions drifting from their own concept

These carry a tag but look nothing like the other functions carrying it. Typicality is measured against the concept's own median, so a genuinely varied concept lowers its own bar and a tight one can flag nobody.

| Function | Concept | Typicality | Concept median | |
|---|---|---:|---:|---|
| `gin.*Context.ClientIP` <br/>`context.go:975` | `c.requestHeader+gin.*Context.requestHeader` | `0.18` | `0.80` | no near-duplicate |
| `gin.*Context.Header` <br/>`context.go:1080` | `c.requestHeader+gin.*Context.requestHeader` | `0.23` | `0.80` | no near-duplicate |
| `binding.*multipartRequest.TrySet` <br/>`binding/multipart_form_mapping.go:27` | `engine.MaxMultipartMemory+c.engine` | `0.09` | `0.66` | no near-duplicate |
| `render.Redirect.WriteContentType` <br/>`render/redirect.go:29` | `bytesconv.StringToBytes+json.API` | `0.18` | `0.52` | no near-duplicate |
| `binding.setArray` <br/>`binding/form_mapping.go:490` | `binding+nil` | `0.12` | `0.38` | no near-duplicate |
| `gin.*responseWriter.Status` <br/>`response_writer.go:98` | `gin.*Context.Header+gin.*Context.Set` | `0.19` | `0.45` | no near-duplicate |
| `gin.*Engine.addRoute` <br/>`gin.go:364` | `tree.method+tree.root` | `0.12` | `0.36` | no near-duplicate |
| `binding.queryBinding.Bind` <br/>`binding/query.go:15` | `binding+nil` | `0.15` | `0.38` | no near-duplicate |
| `gin.*Context.Next` <br/>`context.go:188` | `binding+nil` | `0.17` | `0.38` | no near-duplicate |
| `gin.*Context.Stream` <br/>`context.go:1322` | `binding+nil` | `0.18` | `0.38` | no near-duplicate |

_21 more unusual realizations not listed._

A row marked _no near-duplicate_ appears in no reported pair: nothing else in this report explains it, which makes it drift rather than duplication.

---

## Match #1 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `auth.go:48` | `gin.BasicAuthForRealm` | `(Accounts, string) (HandlerFunc)` | gin.*Context.Header+gin.*Context.Set 0.50, subtle+base64 0.47, c.requestHeader+gin.*Context.requestHeader 0.35 |
| **B** | `auth.go:98` | `gin.BasicAuthForProxy` | `(Accounts, string) (HandlerFunc)` | gin.*Context.Header+gin.*Context.Set 0.50 |

**Explain:** identical after rename, commutative-reorder

**Profile A:** `gin.*Context.Header+gin.*Context.Set` 1.00 (dominance)

**Profile B:** `gin.*Context.Header+gin.*Context.Set` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `425.00` (shape 389.58, concept 2.16, call 33.26)

**Trophic:** `1.00`

**Shared structure:**

- `4.78` — `depth-3 CALL`
- `4.78` — `depth-3 ASSIGN`
- `4.78` — `depth-3 CALL`

**Concept views:** shape `0.33`, corpus `0.40`, feature `0.28`, a-in-b `0.28`, b-in-a `1.00`

**Shared vocabulary:** `id:found`, `call:gin.*Context.Header`, `call:gin.*Context.Set`

**Structural overlap:** `0.70` (merge-worthy)

- share 7 callees: [c.AbortWithStatus, c.Header, c.Set, c.requestHeader, pairs.searchCredential, processAccounts, strconv.Quote]
- overlapping call-graph neighborhoods (0.97): 32 shared
- share patterns: [gin.*Context.Header+gin.*Context.Set]
- both are orchestrator functions
- same package
- callees do related work (1.00): [c.Abort+gin.*Context.Abort, subtle+base64, bytesconv.StringToBytes+json.API, gin.*Context.Header+gin.*Context.Set, writermem.WriteHeaderNow+c.writermem, c.requestHeader+gin.*Context.requestHeader, engine.MaxMultipartMemory+c.engine, binding+nil]
- same visibility
- same receiver type: plain functions
- call into same packages: [gin]

---

## Match #2 — Code-shape: `0.6790`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `gin.go:540` | `gin.*Engine.Run` | `(...string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.61, http.Server+engine.Handler 0.49 |
| **B** | `gin.go:561` | `gin.*Engine.RunTLS` | `(string, string, string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.62, http.Server+engine.Handler 0.53 |

**Explain:** differs by one extra assign, four extra call, one extra selector, and 2 more kinds

**Profile A:** `http.Server+engine.Handler` 1.00 (dominance)

**Profile B:** `http.Server+engine.Handler` 1.00 (dominance)

**Code similarity:** `wl 0.63  flow 1.00  nesting 1.00  sig 0.33  size 0.87`

**Containment:** `0.85`

**Evidence:** `286.89` (shape 270.53, concept 4.09, call 12.27)

**Trophic:** `0.93`

**Shared structure:**

- `5.67` — `depth-1 EXPRSTMT` ×2
- `5.67` — `depth-0 CALL` ×2
- `4.78` — `depth-3 ASSIGN`

**Concept views:** shape `1.00`, corpus `0.95`, feature `0.96`, a-in-b `1.00`, b-in-a `0.96`

**Shared vocabulary:** `id:and`, `id:listener`, `id:listen`

**Structural overlap:** `0.69` (merge-worthy)

- share 4 callees: [debugPrint, debugPrintError, engine.Handler, engine.isUnsafeTrustedProxies]
- overlapping call-graph neighborhoods (0.86): 19 shared
- share patterns: [delims.Left+delims.Right+engine.SetHTMLTemplate, http.Server+engine.Handler]
- both are orchestrator functions
- same package
- callees do related work (0.69): [fmt.Fprintf+runtime, gin.debugPrint+atomic, gin.IsDebugging+gin.debugPrint, delims.Left+delims.Right+engine.SetHTMLTemplate]
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Match #3 — Code-shape: `0.9625`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `routergroup.go:147` | `gin.*RouterGroup.Any` | `(string, ...HandlerFunc) (IRoutes)` | group.calculateAbsolutePath+group.engine 0.53 |
| **B** | `routergroup.go:156` | `gin.*RouterGroup.Match` | `([]string, string, ...HandlerFunc) (IRoutes)` | group.calculateAbsolutePath+group.engine 0.53 |

**Kind:** thin wrappers — both are small bodies delegating to `gin.*RouterGroup.handle` and `gin.*RouterGroup.returnObj` and naming different things, in package `gin`

**Explain:** identical after rename

**Profile A:** `group.calculateAbsolutePath+group.engine` 1.00 (dominance)

**Profile B:** `group.calculateAbsolutePath+group.engine` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 0.75  size 1.00`

**Containment:** `1.00`

**Evidence:** `104.12` (shape 93.99, concept 1.90, call 8.23)

**Trophic:** `1.00`

**Shared structure:**

- `4.78` — `depth-3 BLOCK`
- `4.78` — `depth-3 EXPRSTMT`
- `4.78` — `depth-3 RANGE`

**Concept views:** shape `1.00`, corpus `0.99`, feature `0.99`, a-in-b `0.99`, b-in-a `1.00`

**Shared vocabulary:** `call:gin.*RouterGroup.calculateAbsolutePath`, `sel:group.calculateAbsolutePath`, `id:calculate`

**Structural overlap:** `0.81` (merge-worthy)

- share 2 callees: [group.handle, group.returnObj]
- overlapping call-graph neighborhoods (1.00): 16 shared
- share patterns: [group.calculateAbsolutePath+group.engine]
- both are orchestrator functions
- same package
- callees do related work (1.00): [group.calculateAbsolutePath+group.engine]
- same visibility
- same receiver type: RouterGroup
- call into same packages: [gin]

---

## Match #4 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `binding/toml.go:29` | `binding.decodeToml` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.62, binding+nil 0.44 |
| **B** | `binding/yaml.go:29` | `binding.decodeYAML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.62, binding+nil 0.44 |

**Explain:** identical after rename, commutative-reorder

**Profile A:** `delims.Left+delims.Right` 1.00 (dominance)

**Profile B:** `delims.Left+delims.Right` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `103.59` (shape 100.67, concept 2.93, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.38` — `depth-3 ASSIGN`
- `4.38` — `depth-3 BLOCK`
- `4.38` — `depth-3 CALL`

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `call:binding.mapForm`, `call:gin.*Engine.SetHTMLTemplate`, `id:lazyinit`

**Structural overlap:** `0.71` (merge-worthy)

- share 2 callees: [decoder.Decode, validate]
- share patterns: [binding+nil, delims.Left+delims.Right]
- both are utility functions
- same package
- callers do related work (1.00): [bytes.NewReader+bytes, io.ReadAll+req.Body]
- same visibility
- same receiver type: plain functions
- called from same packages: [binding]

---

## Match #5 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `binding/toml.go:29` | `binding.decodeToml` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.62, binding+nil 0.44 |
| **B** | `binding/xml.go:28` | `binding.decodeXML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.63, binding+nil 0.45 |

**Explain:** identical after rename, commutative-reorder

**Profile A:** `delims.Left+delims.Right` 1.00 (dominance)

**Profile B:** `delims.Left+delims.Right` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `103.59` (shape 100.67, concept 2.93, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.38` — `depth-3 ASSIGN`
- `4.38` — `depth-3 BLOCK`
- `4.38` — `depth-3 CALL`

**Culture:** B realizes `binding+nil` atypically (typicality 0.17, concept median 0.38, convention 0.56)

**Concept views:** shape `1.00`, corpus `0.99`, feature `0.99`, a-in-b `1.00`, b-in-a `0.99`

**Shared vocabulary:** `call:binding.mapForm`, `call:gin.*Engine.SetHTMLTemplate`, `id:lazyinit`

**Structural overlap:** `0.71` (merge-worthy)

- share 2 callees: [decoder.Decode, validate]
- share patterns: [binding+nil, delims.Left+delims.Right]
- both are utility functions
- same package
- callers do related work (0.98): [bytes.NewReader+bytes, io.ReadAll+req.Body]
- same visibility
- same receiver type: plain functions
- called from same packages: [binding]

---

## Match #6 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `binding/xml.go:28` | `binding.decodeXML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.63, binding+nil 0.45 |
| **B** | `binding/yaml.go:29` | `binding.decodeYAML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.62, binding+nil 0.44 |

**Explain:** identical after rename, commutative-reorder

**Profile A:** `delims.Left+delims.Right` 1.00 (dominance)

**Profile B:** `delims.Left+delims.Right` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `103.59` (shape 100.67, concept 2.93, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.38` — `depth-3 ASSIGN`
- `4.38` — `depth-3 BLOCK`
- `4.38` — `depth-3 CALL`

**Culture:** A realizes `binding+nil` atypically (typicality 0.17, concept median 0.38, convention 0.56)

**Concept views:** shape `1.00`, corpus `0.99`, feature `0.99`, a-in-b `0.99`, b-in-a `1.00`

**Shared vocabulary:** `call:binding.mapForm`, `call:gin.*Engine.SetHTMLTemplate`, `id:lazyinit`

**Structural overlap:** `0.71` (merge-worthy)

- share 2 callees: [decoder.Decode, validate]
- share patterns: [binding+nil, delims.Left+delims.Right]
- both are utility functions
- same package
- callers do related work (0.98): [bytes.NewReader+bytes, io.ReadAll+req.Body]
- same visibility
- same receiver type: plain functions
- called from same packages: [binding]

---

## Match #7 — Code-shape: `0.7507`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `gin.go:561` | `gin.*Engine.RunTLS` | `(string, string, string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.62, http.Server+engine.Handler 0.53 |
| **B** | `gin.go:630` | `gin.*Engine.RunQUIC` | `(string, string, string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.63, http.Server+engine.Handler 0.53 |

**Explain:** differs by one extra assign, two extra call, two extra key-value, and 4 more kinds

**Profile A:** `http.Server+engine.Handler` 1.00 (dominance)

**Profile B:** `http.Server+engine.Handler` 1.00 (dominance)

**Code similarity:** `wl 0.58  flow 1.00  nesting 1.00  sig 1.00  size 0.79`

**Containment:** `0.82`

**Evidence:** `225.00` (shape 208.43, concept 4.30, call 12.27)

**Trophic:** `0.86`

**Shared structure:**

- `5.67` — `depth-1 EXPRSTMT` ×2
- `5.67` — `depth-0 CALL` ×2
- `3.87` — `depth-3 CALL`

**Concept views:** shape `1.00`, corpus `0.99`, feature `0.98`, a-in-b `1.00`, b-in-a `0.98`

**Shared vocabulary:** `id:and`, `id:listener`, `id:listen`

**Structural overlap:** `0.76` (merge-worthy)

- share 4 callees: [debugPrint, debugPrintError, engine.Handler, engine.isUnsafeTrustedProxies]
- overlapping call-graph neighborhoods (1.00): 19 shared
- share patterns: [delims.Left+delims.Right+engine.SetHTMLTemplate, http.Server+engine.Handler]
- both are orchestrator functions
- same package
- callees do related work (1.00): [fmt.Fprintf+runtime, gin.debugPrint+atomic, gin.IsDebugging+gin.debugPrint, delims.Left+delims.Right+engine.SetHTMLTemplate]
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Match #8 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `render/toml.go:21` | `render.TOML.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.59, binding+nil 0.43 |
| **B** | `render/yaml.go:21` | `render.YAML.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.59, binding+nil 0.43 |

**Kind:** interface implementations — both implement `Render(http.ResponseWriter) (error)` on `TOML` and `YAML`, in package `render`

**Explain:** identical after rename, commutative-reorder

**Profile A:** `API.Marshal+bytesconv.StringToBytes` 0.74, `bytesconv.StringToBytes+json.API` 0.26 (dominance)

**Profile B:** `API.Marshal+bytesconv.StringToBytes` 0.74, `bytesconv.StringToBytes+json.API` 0.26 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `95.66` (shape 89.18, concept 2.76, call 3.72)

**Trophic:** `1.00`

**Shared structure:**

- `4.78` — `depth-3 BLOCK`
- `4.78` — `depth-3 CALL`
- `4.78` — `depth-3 ASSIGN`

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `call:bytesconv.StringToBytes`, `sel:API.Marshal`, `sel:bytesconv.StringToBytes`

**Structural overlap:** `0.71` (merge-worthy)

- share 2 callees: [r.WriteContentType, w.Write]
- overlapping call-graph neighborhoods (1.00): 12 shared
- share patterns: [API.Marshal+bytesconv.StringToBytes, binding+nil]
- both are leaf functions
- same package
- callees do related work (1.00): [w.WriteHeaderNow+w.ResponseWriter, writermem.WriteHeaderNow+c.writermem]
- same visibility
- both are methods, on TOML and YAML
- call into same packages: [gin]

---

## Match #9 — Code-shape: `0.6290`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `gin.go:581` | `gin.*Engine.RunUnix` | `(string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.58, http.Server+engine.Handler 0.47 |
| **B** | `gin.go:645` | `gin.*Engine.RunListener` | `(net.Listener) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.61, http.Server+engine.Handler 0.51 |

**Explain:** differs by two extra defer, one extra assign, one extra if, and 7 more kinds

**Profile A:** `http.Server+engine.Handler` 1.00 (dominance)

**Profile B:** `http.Server+engine.Handler` 1.00 (dominance)

**Code similarity:** `wl 0.57  flow 0.94  nesting 0.99  sig 0.33  size 0.69`

**Containment:** `0.84` — most of the smaller body's shape is inside the larger

**Evidence:** `294.88` (shape 278.72, concept 3.89, call 12.27)

**Trophic:** `0.85`

**Shared structure:**

- `5.67` — `depth-1 EXPRSTMT` ×2
- `5.67` — `depth-0 CALL` ×2
- `4.78` — `depth-3 UNARY`

**Concept views:** shape `1.00`, corpus `0.93`, feature `0.94`, a-in-b `1.00`, b-in-a `0.94`

**Shared vocabulary:** `id:and`, `id:listener`, `id:listen`

**Structural overlap:** `0.72` (merge-worthy)

- share 5 callees: [debugPrint, debugPrintError, engine.Handler, engine.isUnsafeTrustedProxies, server.Serve]
- overlapping call-graph neighborhoods (1.00): 19 shared
- share patterns: [delims.Left+delims.Right+engine.SetHTMLTemplate, http.Server+engine.Handler]
- both are orchestrator functions
- same package
- callees do related work (1.00): [fmt.Fprintf+runtime, gin.debugPrint+atomic, gin.IsDebugging+gin.debugPrint, delims.Left+delims.Right+engine.SetHTMLTemplate]
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Match #10 — Code-shape: `0.6576`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `gin.go:288` | `gin.*Engine.LoadHTMLFiles` | `(...string)` | delims.Left+delims.Right 0.61, delims.Left+delims.Right+engine.SetHTMLTemplate 0.60 |
| **B** | `gin.go:300` | `gin.*Engine.LoadHTMLFS` | `(http.FileSystem, ...string)` | delims.Left+delims.Right 0.60, delims.Left+delims.Right+engine.SetHTMLTemplate 0.59 |

**Explain:** differs by two extra call, two extra key-value, one extra composite literal, and 2 more kinds

**Profile A:** `delims.Left+delims.Right` 1.00 (dominance)

**Profile B:** `delims.Left+delims.Right` 1.00 (dominance)

**Code similarity:** `wl 0.55  flow 1.00  nesting 1.00  sig 0.50  size 0.87`

**Containment:** `0.75`

**Evidence:** `234.18` (shape 206.68, concept 3.79, call 23.71)

**Trophic:** `0.84`

**Shared structure:**

- `5.98` — `depth-3 KV` ×2
- `5.98` — `depth-2 KV` ×2
- `5.82` — `depth-1 KV` ×2

**Concept views:** shape `1.00`, corpus `0.98`, feature `0.98`, a-in-b `0.98`, b-in-a `1.00`

**Shared vocabulary:** `call:binding.mapForm`, `call:gin.*Engine.SetHTMLTemplate`, `id:lazyinit`

**Structural overlap:** `0.77` (merge-worthy)

- share 6 callees: [Delims, Funcs, IsDebugging, engine.SetHTMLTemplate, template.Must, template.New]
- overlapping call-graph neighborhoods (1.00): 11 shared
- share patterns: [delims.Left+delims.Right, delims.Left+delims.Right+engine.SetHTMLTemplate]
- both are orchestrator functions
- same package
- callees do related work (1.00): [gin.debugPrint+atomic, gin.IsDebugging+gin.debugPrint, tree.method+tree.root, writermem.WriteHeaderNow+c.writermem, delims.Left+delims.Right, delims.Left+delims.Right+engine.SetHTMLTemplate]
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Families

33 families, 180 functions in a family, largest 17 members; 249 edges scored here that retrieval never proposed

### Family 1 — 6 members, every pair `>= 0.48` code-shape, evidence `3517`

```mermaid
flowchart LR
    m0["gin.*Engine.Run"]
    m1["gin.*Engine.RunTLS"]
    m2["gin.*Engine.RunUnix"]
    m3["gin.*Engine.RunFd"]
    m4["gin.*Engine.RunQUIC"]
    m5["gin.*Engine.RunListener"]
    m0 --- m1
    m0 --- m2
    m0 --- m3
    m0 --- m4
    m0 --- m5
    m1 --- m2
    m1 --- m3
    m1 --- m4
    m1 --- m5
    m2 --- m3
    m2 --- m4
    m2 --- m5
    m3 --- m4
    m3 --- m5
    m4 --- m5
```

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `gin.go:540` | `gin.*Engine.Run` | `(...string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.61, http.Server+engine.Handler 0.49 |
| `gin.go:561` | `gin.*Engine.RunTLS` | `(string, string, string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.62, http.Server+engine.Handler 0.53 |
| `gin.go:581` | `gin.*Engine.RunUnix` | `(string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.58, http.Server+engine.Handler 0.47 |
| `gin.go:607` | `gin.*Engine.RunFd` | `(int) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.56 |
| `gin.go:630` | `gin.*Engine.RunQUIC` | `(string, string, string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.63, http.Server+engine.Handler 0.53 |
| `gin.go:645` | `gin.*Engine.RunListener` | `(net.Listener) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.61, http.Server+engine.Handler 0.51 |

### Family 2 — 8 members, every pair `>= 0.40` code-shape, evidence `758`  (13 edges scored here), interface implementations of `Render(http.ResponseWriter) (error)`, in package `render`

```mermaid
flowchart LR
    m0["render.BSON.Render"]
    m1["render.HTML.Render"]
    m2["render.IndentedJSON.Render"]
    m3["render.PureJSON.Render"]
    m4["render.ProtoBuf.Render"]
    m5["render.TOML.Render"]
    m6["render.XML.Render"]
    m7["render.YAML.Render"]
    m0 --- m1
    m0 --- m2
    m0 --- m3
    m0 --- m4
    m0 --- m5
    m0 --- m6
    m0 --- m7
    m1 --- m2
    m1 --- m3
    m1 --- m4
    m1 --- m5
    m1 --- m6
    m1 --- m7
    m2 --- m3
    m2 --- m4
    m2 --- m5
    m2 --- m6
    m2 --- m7
    m3 --- m4
    m3 --- m5
    m3 --- m6
    m3 --- m7
    m4 --- m5
    m4 --- m6
    m4 --- m7
    m5 --- m6
    m5 --- m7
    m6 --- m7
```

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `render/bson.go:21` | `render.BSON.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.58, binding+nil 0.45, json.Marshal+json.MarshalIndent 0.14 |
| `render/html.go:89` | `render.HTML.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.49, json.Marshal+json.MarshalIndent 0.15 |
| `render/json.go:78` | `render.IndentedJSON.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.61, bytesconv.StringToBytes+json.API 0.47, json.Marshal+json.MarshalIndent 0.18 |
| `render/json.go:184` | `render.PureJSON.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.50, json.Marshal+json.MarshalIndent 0.17 |
| `render/protobuf.go:21` | `render.ProtoBuf.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.56, binding+nil 0.42 |
| `render/toml.go:21` | `render.TOML.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.59, binding+nil 0.43 |
| `render/xml.go:20` | `render.XML.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.43, json.Marshal+json.MarshalIndent 0.27 |
| `render/yaml.go:21` | `render.YAML.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.59, binding+nil 0.43 |

### Family 3 — 5 members, every pair `>= 0.47` code-shape, evidence `717`

```mermaid
flowchart LR
    m0["binding.decodeJSON"]
    m1["binding.decodeMsgPack"]
    m2["binding.decodeToml"]
    m3["binding.decodeXML"]
    m4["binding.decodeYAML"]
    m0 --- m1
    m0 --- m2
    m0 --- m3
    m0 --- m4
    m1 --- m2
    m1 --- m3
    m1 --- m4
    m2 --- m3
    m2 --- m4
    m3 --- m4
```

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `binding/json.go:44` | `binding.decodeJSON` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.51 |
| `binding/msgpack.go:31` | `binding.decodeMsgPack` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.43, binding+nil 0.40 |
| `binding/toml.go:29` | `binding.decodeToml` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.62, binding+nil 0.44 |
| `binding/xml.go:28` | `binding.decodeXML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.63, binding+nil 0.45 |
| `binding/yaml.go:29` | `binding.decodeYAML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.62, binding+nil 0.44 |

### Family 4 — 3 members, every pair `>= 0.45` code-shape, evidence `621`

```mermaid
flowchart LR
    m0["gin.*Engine.LoadHTMLGlob"]
    m1["gin.*Engine.LoadHTMLFiles"]
    m2["gin.*Engine.LoadHTMLFS"]
    m0 --- m1
    m0 --- m2
    m1 --- m2
```

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `gin.go:272` | `gin.*Engine.LoadHTMLGlob` | `(string)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.60, delims.Left+delims.Right 0.59 |
| `gin.go:288` | `gin.*Engine.LoadHTMLFiles` | `(...string)` | delims.Left+delims.Right 0.61, delims.Left+delims.Right+engine.SetHTMLTemplate 0.60 |
| `gin.go:300` | `gin.*Engine.LoadHTMLFS` | `(http.FileSystem, ...string)` | delims.Left+delims.Right 0.60, delims.Left+delims.Right+engine.SetHTMLTemplate 0.59 |

### Family 5 — 7 members, every pair `>= 0.40` code-shape, evidence `590`  (5 edges scored here)

```mermaid
flowchart LR
    m0["gin.*Context.IndentedJSON"]
    m1["gin.*Context.SecureJSON"]
    m2["gin.*Context.String"]
    m3["gin.*Context.Redirect"]
    m4["gin.*Context.Data"]
    m5["gin.*Context.DataFromReader"]
    m6["gin.*Context.SSEvent"]
    m0 --- m1
    m0 --- m2
    m0 --- m3
    m0 --- m4
    m0 --- m5
    m0 --- m6
    m1 --- m2
    m1 --- m3
    m1 --- m4
    m1 --- m5
    m1 --- m6
    m2 --- m3
    m2 --- m4
    m2 --- m5
    m2 --- m6
    m3 --- m4
    m3 --- m5
    m3 --- m6
    m4 --- m5
    m4 --- m6
    m5 --- m6
```

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `context.go:1180` | `gin.*Context.IndentedJSON` | `(int, any)` | binding+nil 0.60, engine.MaxMultipartMemory+c.engine 0.27 |
| `context.go:1187` | `gin.*Context.SecureJSON` | `(int, any)` | binding+nil 0.47, engine.MaxMultipartMemory+c.engine 0.37 |
| `context.go:1248` | `gin.*Context.String` | `(int, string, ...any)` | binding+nil 0.55, c.formCache+c.queryCache 0.36, engine.MaxMultipartMemory+c.engine 0.23 |
| `context.go:1253` | `gin.*Context.Redirect` | `(int, string)` | Request.URL+req.URL 0.49, c.requestHeader+gin.*Context.requestHeader 0.41, engine.MaxMultipartMemory+c.engine 0.40 |
| `context.go:1262` | `gin.*Context.Data` | `(int, string, []byte)` | binding+nil 0.59, engine.MaxMultipartMemory+c.engine 0.26 |
| `context.go:1270` | `gin.*Context.DataFromReader` | `(int, int64, string, io.Reader, map[string]string)` | binding+nil 0.48 |
| `context.go:1313` | `gin.*Context.SSEvent` | `(string, any)` | binding+nil 0.63, c.requestHeader+gin.*Context.requestHeader 0.35, engine.MaxMultipartMemory+c.engine 0.29 |

_28 more families not listed._


# gin

HTTP framework; a small core surrounded by generated-looking binding and render variants

**What this rung shows:** family clones — the case corroborated ranking was tuned to separate

| | |
|---|---|
| Corpus | [gin](https://github.com/gin-gonic/gin) |
| Pinned at | `v1.12.0` (`73726dc606796a025971fe451f0aa6f1b9b847f6`) |
| Project since | 2014 |
| doppel | `89a3161` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Learning concept vocabulary...
Lexicon: 46 concepts (6 seeded, 40 emergent), 1348/3708 features above 182 df, 109 functions unlabeled
Generating concept documents...
Culture: 37 concepts modeled, 161 associations, 37 unusual realizations
Habitats: 5 modeled, 17 misfits (0 excused by subsystem), 1 subsystems; most uniform binding (norm 0.92), most diverse json (norm 0.63)
Conventions: strongest c.ShouldBindBodyWith+gin.*Context.ShouldBindBody… (1.00), loosest gin.IsDebugging+gin.debugPrint (0.24)
Ecosystems: 418 profiled (301 dominance, 28 coalition, 0 conflict, 89 weak)
Calibration: rate 0.01 over 20000 null pairs -> threshold 0.49, struct-min 0.50, family-min 0.49
Found 497 functions. Retrieving candidates...
Retrieval: shape 260, concept 1601, call 609 -> 2101 unique pairs
  concept-only 61.7%  call-only 15.7%  suppressed-shape functions: 0  large identity buckets: 0  surviving patterns: 3321
Running structural comparison on 2101 pairs...
  477 pairs remain after struct-min=0.50 filter
Families: 27 over 47 components, 155 functions in a family, 235 edges completed
```

# Code Similarity Report

**Functions analyzed:** 497 | **Threshold:** 0.60 | **Pairs found:** 10

---

## What doppel sees

**497 functions** across **7 packages** — test functions excluded. Structural roles: 361 leaf, 57 orchestrator, 17 passthrough, 62 utility.

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
    c8["API.Marshal+bytesconv.StringToBytes<br/>14"]
    c9["Request.URL+req.URL<br/>6"]
    c10["URL.Path+c.Writer<br/>8"]
    c11["binding+nil<br/>247"]
    c12["bytes.NewReader+bytes<br/>5"]
    c13["bytesconv.BytesToString+bytesconv<br/>5"]
    c14["bytesconv.StringToBytes+json.API<br/>8"]
    c15["c.Abort+gin.*Context.Abort<br/>4"]
    c16["c.MustBindWith+gin.*Context.MustBindWith<br/>8"]
    c17["c.Next+c.Request<br/>4"]
    c18["c.ShouldBindBodyWith+gin.*Context.ShouldBindBody…<br/>5"]
    c19["c.ShouldBindWith+gin.*Context.ShouldBindWith<br/>8"]
    c20["c.formCache+c.queryCache<br/>6"]
    c21["c.hasRequestContext+Request.Context<br/>5"]
    c22["c.requestHeader+gin.*Context.requestHeader<br/>7"]
    c23["cmp+httputil<br/>8"]
    c24["delims.Left+delims.Right<br/>22"]
    c25["delims.Left+delims.Right+engine.SetHTMLTemplate<br/>39"]
    c26["engine.MaxMultipartMemory+c.engine<br/>8"]
    c27["field.Tag+Tag.Get<br/>4"]
    c28["flag+atomic<br/>6"]
    c29["fmt.Fprintf+runtime<br/>4"]
    c30["gin+template<br/>20"]
    c31["gin.*Context.Header+gin.*Context.Set<br/>5"]
    c32["gin.IsDebugging+gin.debugPrint<br/>7"]
    c33["gin.debugPrint+atomic<br/>5"]
    c34["group.calculateAbsolutePath+group.engine<br/>19"]
    c35["http.Server+engine.Handler<br/>5"]
    c36["io.ReadAll+req.Body<br/>7"]
    c37["json.Marshal+json.MarshalIndent<br/>31"]
    c38["log<br/>3"]
    c39["n.nType+n.priority<br/>27"]
    c40["reflect.Array+reflect.Slice<br/>4"]
    c41["reflect.Map+reflect.New<br/>10"]
    c42["reflect.New+reflect.Array<br/>5"]
    c43["reflect.New+value.Type<br/>4"]
    c44["render.writeContentType+bytes<br/>6"]
    c45["strings.Split+reflect<br/>4"]
    c46["strings.TrimSpace+bytesconv<br/>4"]
    c47["subtle+base64<br/>5"]
    c48["tree.method+tree.root<br/>5"]
    c49["value.Addr+field.Tag<br/>21"]
    c50["value.Set+value.Type<br/>6"]
    c51["w.WriteHeaderNow+w.ResponseWriter<br/>10"]
    c52["writermem.WriteHeaderNow+c.writermem<br/>5"]
    c53["xml+runtime<br/>13"]
    c0 --> c1
    c1 --> c2
    c1 --> c3
    c0 --> c4
    c0 --> c5
    c5 --> c6
    c0 --> c7
    c4 --> c8
    c4 --> c9
    c1 --> c10
    c5 --> c11
    c4 --> c12
    c1 --> c13
    c4 --> c14
    c3 --> c15
    c3 --> c16
    c1 --> c17
    c3 --> c18
    c3 --> c19
    c3 --> c20
    c3 --> c21
    c3 --> c22
    c4 --> c23
    c4 --> c24
    c4 --> c25
    c4 --> c26
    c4 --> c27
    c3 --> c28
    c1 --> c29
    c4 --> c30
    c4 --> c31
    c4 --> c32
    c4 --> c33
    c1 --> c34
    c1 --> c35
    c1 --> c36
    c4 --> c37
    c1 --> c38
    c4 --> c39
    c4 --> c40
    c4 --> c41
    c4 --> c42
    c4 --> c43
    c4 --> c44
    c1 --> c45
    c4 --> c46
    c4 --> c47
    c4 --> c48
    c4 --> c49
    c4 --> c50
    c4 --> c51
    c4 --> c52
    c3 --> c53
```

**No practice here for** `circuit_breaker`, `db_access`, `error_wrapping`, `grpc_call`, `http_call`, `mapping`, `retry`, `transaction`. Concepts are learned from this corpus, so one can never be absent — it exists because functions carry it. These are the *seeds* the search started from that grew nothing: a direct answer to "does this codebase already do X".

| Concept | Functions | Convention |
|---|---:|---|
| `binding+nil` | 247 | `0.60` (settled) |
| `delims.Left+delims.Right+engine.SetHTMLTemplate` | 39 | `0.48` (loose) |
| `json.Marshal+json.MarshalIndent` | 31 | `0.52` (settled) |
| `n.nType+n.priority` | 27 | `0.47` (loose) |
| `delims.Left+delims.Right` | 22 | `0.39` (loose) |
| `value.Addr+field.Tag` | 21 | `0.49` (loose) |
| `gin+template` | 20 | `0.68` (settled) |
| `group.calculateAbsolutePath+group.engine` | 19 | `0.53` (settled) |
| `API.Marshal+bytesconv.StringToBytes` | 14 | `0.42` (loose) |
| `xml+runtime` | 13 | `0.56` (settled) |
| `reflect.Map+reflect.New` | 10 | `0.39` (loose) |
| `w.WriteHeaderNow+w.ResponseWriter` | 10 | `0.29` (loose) |
| `URL.Path+c.Writer` | 8 | `0.29` (loose) |
| `bytesconv.StringToBytes+json.API` | 8 | `0.43` (loose) |
| `c.MustBindWith+gin.*Context.MustBindWith` | 8 | `0.89` (unanimous) |
| `c.ShouldBindWith+gin.*Context.ShouldBindWith` | 8 | `0.89` (unanimous) |
| `cmp+httputil` | 8 | `0.40` (loose) |
| `engine.MaxMultipartMemory+c.engine` | 8 | `0.37` (loose) |
| `c.requestHeader+gin.*Context.requestHeader` | 7 | `0.43` (loose) |
| `gin.IsDebugging+gin.debugPrint` | 7 | `0.24` (loose) |
| `io.ReadAll+req.Body` | 7 | `0.50` (settled) |
| `Request.URL+req.URL` | 6 | `0.29` (loose) |
| `c.formCache+c.queryCache` | 6 | `0.34` (loose) |
| `flag+atomic` | 6 | `0.61` (settled) |
| `render.writeContentType+bytes` | 6 | `0.64` (settled) |
| `value.Set+value.Type` | 6 | `0.43` (loose) |
| `bytes.NewReader+bytes` | 5 | `0.86` (unanimous) |
| `bytesconv.BytesToString+bytesconv` | 5 | `0.38` (loose) |
| `c.ShouldBindBodyWith+gin.*Context.ShouldBindBody…` | 5 | `1.00` (unanimous) |
| `c.hasRequestContext+Request.Context` | 5 | `0.54` (settled) |
| `gin.*Context.Header+gin.*Context.Set` | 5 | `0.34` (loose) |
| `gin.debugPrint+atomic` | 5 | `0.46` (loose) |
| `http.Server+engine.Handler` | 5 | `0.91` (unanimous) |
| `reflect.New+reflect.Array` | 5 | `0.38` (loose) |
| `subtle+base64` | 5 | `0.27` (loose) |
| `tree.method+tree.root` | 5 | `0.29` (loose) |
| `writermem.WriteHeaderNow+c.writermem` | 5 | `0.30` (loose) |
| `c.Abort+gin.*Context.Abort` | 4 | — |
| `c.Next+c.Request` | 4 | — |
| `field.Tag+Tag.Get` | 4 | — |
| `fmt.Fprintf+runtime` | 4 | — |
| `reflect.Array+reflect.Slice` | 4 | — |
| `reflect.New+value.Type` | 4 | — |
| `strings.Split+reflect` | 4 | — |
| `strings.TrimSpace+bytesconv` | 4 | — |
| `log` | 3 | — |

Convention is how uniformly this corpus realizes a concept: `1.00` means every function carrying the tag does it the same way, and a low number means the tag covers several unrelated habits. A concept with fewer than five members is not modeled.

### Where the duplication is

Merge-worthy pairs folded up to their packages. An edge means two packages keep solving the same problem separately; a count on a node means the repetition is inside one package.

```mermaid
flowchart LR
    p0["gin<br/>188 internal"]
    p1["render<br/>78 internal"]
    p0 ---|"1"| p1
```

### How settled each package is

A package with at least five functions gets a habitat model: doppel learns what is normal there and measures how surprising each member is against it. **Norm** is how uniform the package's practice is. A **misfit** is a function alien to its package *and* to the wider subsystem around it — one that fits its neighbours a directory up is normal for this codebase and is not reported.

```mermaid
flowchart TD
    h0["json<br/>24 functions · norm 0.63<br/>9 misfits"]
    h1["ginS<br/>25 functions · norm 0.70<br/>8 misfits"]
    h2["render<br/>42 functions · norm 0.84"]
    h3["gin<br/>324 functions · norm 0.89"]
    h4["binding<br/>79 functions · norm 0.92"]
    classDef good fill:#d7ecd9,color:#1b3d20
    classDef warn fill:#fbeecb,color:#4a3a12
    classDef hot fill:#f7d6d6,color:#4a1c1c
    class h2,h3,h4 good
    class h0,h1 warn
```

Most uniform is `binding` (norm `0.92`); most varied is `json` (norm `0.63`). 17 functions are alien to their package and to the subsystem around it.

### How these candidates were found

Three channels propose candidates independently — shared rare *structure*, shared *concepts*, shared *calls* — and their union is what gets compared. This run: **2101 candidate pairs** (shape 260, concept 1601, call 609), of which 16% arrived on call evidence alone and 62% on concept evidence alone. A pair sharing none of the three is never compared, however alike it looks.

Each function is also an arena where its candidate concepts compete for its evidence. 418 functions reached an equilibrium: **301** settled on a single concept, **28** on a coalition, **0** hold concepts this corpus says do not go together.

---

## Local practice

The vocabulary above says what a concept *is*. This says what one looks like when **this** codebase writes it — learned from the corpus, so it describes the house style rather than a rule from anywhere else.

### How this codebase writes each concept

Only what is **distinctive**. A feature earns a row by being carried by this concept's members at least twice as often as by the corpus at large — nearly every Go function has a `return` and an `if`, so prevalence alone would describe the language rather than this codebase. Weights are how much a channel counts toward whether a member looks normal — calls 40, control flow 20, co-occurring tags 15, role 15, package 10.

**`binding+nil`** — 247 functions

Nothing distinctive: its members do what the rest of the corpus does. The tag groups them; a shared way of writing them does not.

**`delims.Left+delims.Right+engine.SetHTMLTemplate`** — 39 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| flow ×20 | `range` | `███·······` | 10 of 39 | 3.0× |
| role ×15 | `orchestrator` | `████······` | 17 of 39 | 3.8× |

**`json.Marshal+json.MarshalIndent`** — 31 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `gin.*responseWriter.Write` | `███·······` | 8 of 31 | 11× |
| cotags ×15 | `API.Marshal+bytesconv.StringToBytes` | `███·······` | 9 of 31 | 10× |
| package ×10 | `json` | `██████····` | 20 of 31 | 13× |
|  | `render` | `███·······` | 10 of 31 | 3.8× |

**`n.nType+n.priority`** — 27 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| flow ×20 | `for` | `████······` | 10 of 27 | 10× |
|  | `switch` | `███·······` | 7 of 27 | 5.2× |
|  | `range` | `████······` | 11 of 27 | 4.7× |
|  | `if` | `████████··` | 22 of 27 | 2.2× |
| cotags ×15 | `delims.Left+delims.Right+engine.SetHTMLTemplate` | `███·······` | 8 of 27 | 3.8× |

**`delims.Left+delims.Right`** — 22 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| flow ×20 | `if` | `█████████·` | 19 of 22 | 2.4× |
| cotags ×15 | `delims.Left+delims.Right+engine.SetHTMLTemplate` | `███·······` | 6 of 22 | 3.5× |
| role ×15 | `utility` | `███·······` | 7 of 22 | 2.6× |
| package ×10 | `binding` | `███████···` | 16 of 22 | 4.6× |

**`value.Addr+field.Tag`** — 21 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `reflect.ValueOf` | `███·······` | 7 of 21 | 17× |
|  | `gin.*Context.Set` | `███·······` | 7 of 21 | 11× |
| flow ×20 | `switch` | `███·······` | 7 of 21 | 6.6× |
|  | `if` | `█████████·` | 19 of 21 | 2.5× |
| cotags ×15 | `value.Set+value.Type` | `███·······` | 6 of 21 | 24× |
|  | `reflect.Map+reflect.New` | `████······` | 8 of 21 | 19× |
| package ×10 | `binding` | `██████████` | 20 of 21 | 6.0× |

_31 further concepts are modeled and not described._

### Which concepts share a function

`++` at least four times chance, `+` at least twice, `−` at most half, `never` not once. A blank cell is ordinary company — near chance, which is not culture.

| | `API.Marshal+bytesconv.StringToBytes` | `Request.URL+req.URL` | `URL.Path+c.Writer` | `binding+nil` | `bytes.NewReader+bytes` | `bytesconv.BytesToString+bytesconv` | `bytesconv.StringToBytes+json.API` | `c.Abort+gin.*Context.Abort` | `c.MustBindWith+gin.*Context.MustBindWith` | `c.Next+c.Request` | `c.ShouldBindBodyWith+gin.*Context.ShouldBindBody…` | `c.ShouldBindWith+gin.*Context.ShouldBindWith` | `c.formCache+c.queryCache` | `c.hasRequestContext+Request.Context` | `c.requestHeader+gin.*Context.requestHeader` | `cmp+httputil` | `delims.Left+delims.Right` | `delims.Left+delims.Right+engine.SetHTMLTemplate` | `engine.MaxMultipartMemory+c.engine` | `field.Tag+Tag.Get` | `flag+atomic` | `fmt.Fprintf+runtime` | `gin+template` | `gin.*Context.Header+gin.*Context.Set` | `gin.IsDebugging+gin.debugPrint` | `gin.debugPrint+atomic` | `group.calculateAbsolutePath+group.engine` | `http.Server+engine.Handler` | `io.ReadAll+req.Body` | `json.Marshal+json.MarshalIndent` | `log` | `n.nType+n.priority` | `reflect.Array+reflect.Slice` | `reflect.Map+reflect.New` | `reflect.New+reflect.Array` | `reflect.New+value.Type` | `render.writeContentType+bytes` | `strings.Split+reflect` | `strings.TrimSpace+bytesconv` | `subtle+base64` | `tree.method+tree.root` | `value.Addr+field.Tag` | `value.Set+value.Type` | `w.WriteHeaderNow+w.ResponseWriter` | `writermem.WriteHeaderNow+c.writermem` |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **`Request.URL+req.URL`** |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`URL.Path+c.Writer`** |  | ++ | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`binding+nil`** |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`bytes.NewReader+bytes`** |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`bytesconv.BytesToString+bytesconv`** |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`bytesconv.StringToBytes+json.API`** | ++ |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`c.Abort+gin.*Context.Abort`** |  |  |  | + |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`c.MustBindWith+gin.*Context.MustBindWith`** |  |  |  | + |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`c.Next+c.Request`** |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`c.ShouldBindBodyWith+gin.*Context.ShouldBindBody…`** |  |  |  | + |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`c.ShouldBindWith+gin.*Context.ShouldBindWith`** |  |  |  | + |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`c.formCache+c.queryCache`** |  |  |  | + |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`c.hasRequestContext+Request.Context`** |  |  |  | + |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`c.requestHeader+gin.*Context.requestHeader`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`cmp+httputil`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`delims.Left+delims.Right`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`delims.Left+delims.Right+engine.SetHTMLTemplate`** |  | ++ | ++ |  |  |  |  |  |  |  |  |  |  |  |  |  | + | | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`engine.MaxMultipartMemory+c.engine`** |  |  |  | + |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`field.Tag+Tag.Get`** |  |  |  | + |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`flag+atomic`** |  |  |  | + |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | | |
| **`fmt.Fprintf+runtime`** |  |  |  | + |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | | |
| **`gin+template`** |  |  |  | − |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | | |
| **`gin.*Context.Header+gin.*Context.Set`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | | |
| **`gin.IsDebugging+gin.debugPrint`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | | | |
| **`gin.debugPrint+atomic`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | ++ | | | | | | | | | | | | | | | | | | | | |
| **`group.calculateAbsolutePath+group.engine`** |  |  |  | − |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | | |
| **`http.Server+engine.Handler`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | ++ |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | | |
| **`io.ReadAll+req.Body`** |  |  |  | + |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | | |
| **`json.Marshal+json.MarshalIndent`** | ++ |  |  |  |  |  | ++ |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | | |
| **`log`** |  |  |  | + |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | | |
| **`n.nType+n.priority`** |  | ++ |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | + |  | ++ |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | | |
| **`reflect.Array+reflect.Slice`** |  |  |  | + |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | | |
| **`reflect.Map+reflect.New`** |  |  |  | + |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | | | | |
| **`reflect.New+reflect.Array`** |  |  |  | + |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | ++ | ++ | | | | | | | | | | | |
| **`reflect.New+value.Type`** |  |  |  | + |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | ++ | ++ | | | | | | | | | | |
| **`render.writeContentType+bytes`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | | |
| **`strings.Split+reflect`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | | |
| **`strings.TrimSpace+bytesconv`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | | |
| **`subtle+base64`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | | |
| **`tree.method+tree.root`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | ++ |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | | | | |
| **`value.Addr+field.Tag`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | ++ |  |  |  |  |  |  |  |  |  |  |  | + | ++ | ++ | ++ | ++ |  |  |  |  |  | | | | |
| **`value.Set+value.Type`** |  |  |  | + |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | ++ | ++ | ++ |  |  |  |  |  | ++ | | | |
| **`w.WriteHeaderNow+w.ResponseWriter`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | | |
| **`writermem.WriteHeaderNow+c.writermem`** |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | ++ |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  | |
| **`xml+runtime`** |  |  |  | − |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |  |

### What travels with what

Co-occurrence measured against chance across every function. Only relationships at least twice — or at most half — as common as chance are reported; near-chance company is not culture. Each kind is listed separately, because there are far more call tokens than concepts and one shared list is all calls. Within a kind, strongest first means lift weighted by how many functions carry it — a 100× relationship holding for three functions is a weaker finding than a 30× one holding for thirty.

**Together more than chance — tag~tag**

- 8 of 8 `bytesconv.StringToBytes+json.API` functions also `API.Marshal+bytesconv.StringToBytes` — 36× chance
- 5 of 5 `gin.debugPrint+atomic` functions also `gin.IsDebugging+gin.debugPrint` — 71× chance
- 4 of 5 `reflect.New+reflect.Array` functions also `value.Set+value.Type` — 66× chance
- 5 of 6 `value.Set+value.Type` functions also `reflect.Map+reflect.New` — 41× chance
- 8 of 10 `reflect.Map+reflect.New` functions also `value.Addr+field.Tag` — 19× chance
- 4 of 4 `reflect.New+value.Type` functions also `reflect.Map+reflect.New` — 50× chance
- _39 more not listed_

**Together more than chance — tag~role**

- 3 of 5 `reflect.New+reflect.Array` functions also `passthrough` — 18× chance
- 3 of 5 `writermem.WriteHeaderNow+c.writermem` functions also `passthrough` — 18× chance
- 5 of 5 `http.Server+engine.Handler` functions also `orchestrator` — 8.7× chance
- 17 of 39 `delims.Left+delims.Right+engine.SetHTMLTemplate` functions also `orchestrator` — 3.8× chance
- 3 of 6 `Request.URL+req.URL` functions also `passthrough` — 15× chance
- 3 of 6 `value.Set+value.Type` functions also `passthrough` — 15× chance
- _15 more not listed_

**Together more than chance — tag~call**

- 7 of 7 `c.requestHeader+gin.*Context.requestHeader` functions also `gin.*Context.requestHeader` — 71× chance
- 8 of 8 `c.MustBindWith+gin.*Context.MustBindWith` functions also `gin.*Context.MustBindWith` — 55× chance
- 8 of 8 `c.ShouldBindWith+gin.*Context.ShouldBindWith` functions also `gin.*Context.ShouldBindWith` — 55× chance
- 11 of 14 `API.Marshal+bytesconv.StringToBytes` functions also `gin.*responseWriter.Write` — 33× chance
- 5 of 5 `bytes.NewReader+bytes` functions also `bytes.NewReader` — 99× chance
- 5 of 5 `bytesconv.BytesToString+bytesconv` functions also `bytesconv.BytesToString` — 99× chance
- _70 more not listed_

**Apart more than chance — tag~tag**

- 1 of 20 `gin+template` functions also `binding+nil` — 0.1× chance
- 1 of 19 `group.calculateAbsolutePath+group.engine` functions also `binding+nil` — 0.1× chance
- 3 of 13 `xml+runtime` functions also `binding+nil` — 0.5× chance

**Apart more than chance — tag~role**

- **no** `json.Marshal+json.MarshalIndent` function has `utility` — chance alone would give about 4 of 31
- **no** `gin.*Context.Header+gin.*Context.Set` function has `leaf` — chance alone would give about 4 of 5
- **no** `http.Server+engine.Handler` function has `leaf` — chance alone would give about 4 of 5
- **no** `writermem.WriteHeaderNow+c.writermem` function has `leaf` — chance alone would give about 4 of 5
- 11 of 39 `delims.Left+delims.Right+engine.SetHTMLTemplate` functions also `leaf` — 0.4× chance
- 1 of 8 `bytesconv.StringToBytes+json.API` functions also `leaf` — 0.2× chance
- _8 more not listed_

**Apart more than chance — tag~call**

- 1 of 247 `binding+nil` functions also `render.writeContentType` — 0.1× chance
- 1 of 247 `binding+nil` functions also `gin.*RouterGroup.handle` — 0.2× chance

### Functions drifting from their own concept

These carry a tag but look nothing like the other functions carrying it. Typicality is measured against the concept's own median, so a genuinely varied concept lowers its own bar and a tight one can flag nobody.

| Function | Concept | Typicality | Concept median | |
|---|---|---:|---:|---|
| `gin.*RouterGroup.createStaticHandler` <br/>`routergroup.go:216` | `gin+template` | `0.06` | `0.94` | no near-duplicate |
| `gin.*Context.ClientIP` <br/>`context.go:975` | `c.hasRequestContext+Request.Context` | `0.26` | `0.86` | no near-duplicate |
| `gin.authPairs.searchCredential` <br/>`auth.go:32` | `API.Marshal+bytesconv.StringToBytes` | `0.24` | `0.55` | no near-duplicate |
| `gin.*RouterGroup.createStaticHandler` <br/>`routergroup.go:216` | `group.calculateAbsolutePath+group.engine` | `0.24` | `0.55` | no near-duplicate |
| `gin.*Engine.handleHTTPRequest` <br/>`gin.go:690` | `gin.*Context.Header+gin.*Context.Set` | `0.23` | `0.52` | no near-duplicate |
| `gin.*Context.initFormCache` <br/>`context.go:638` | `c.formCache+c.queryCache` | `0.23` | `0.48` | no near-duplicate |
| `gin.*Context.ClientIP` <br/>`context.go:975` | `engine.MaxMultipartMemory+c.engine` | `0.19` | `0.42` | no near-duplicate |
| `binding.decodePlain` <br/>`binding/plain.go:31` | `binding+nil` | `0.13` | `0.35` | no near-duplicate |
| `gin.New` <br/>`gin.go:202` | `delims.Left+delims.Right` | `0.19` | `0.39` | no near-duplicate |
| `binding.*defaultValidator.lazyinit` <br/>`binding/default_validator.go:90` | `delims.Left+delims.Right` | `0.19` | `0.39` | no near-duplicate |

_27 more unusual realizations not listed._

A row marked _no near-duplicate_ appears in no reported pair: nothing else in this report explains it, which makes it drift rather than duplication.

---

## Match #1 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `auth.go:48` | `gin.BasicAuthForRealm` | `(Accounts, string) (HandlerFunc)` | subtle+base64 0.66, gin.*Context.Header+gin.*Context.Set 0.57, c.requestHeader+gin.*Context.requestHeader 0.54 |
| **B** | `auth.go:98` | `gin.BasicAuthForProxy` | `(Accounts, string) (HandlerFunc)` | subtle+base64 0.66, gin.*Context.Header+gin.*Context.Set 0.57, c.requestHeader+gin.*Context.requestHeader 0.54 |

**Profile A:** `gin.*Context.Header+gin.*Context.Set` 1.00 (dominance)

**Profile B:** `gin.*Context.Header+gin.*Context.Set` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `603.72` (shape 562.58, concept 7.88, call 33.26)

**Trophic:** `1.00`

**Shared structure:**

- `4.93` — `seq[ assign:=(call:processAccounts) ; return(funclit) ]`
- `4.93` — `seq[ assign:=(call:searchCredential) ; if(unary) ]`
- `4.93` — `seq[ assign=(bin) ; assign:=(call:processAccounts) ]`

**Structural overlap:** `0.81` (merge-worthy)

- share 7 callees: [c.AbortWithStatus, c.Header, c.Set, c.requestHeader, pairs.searchCredential, processAccounts, strconv.Quote]
- overlapping call-graph neighborhoods (0.97): 32 shared
- share patterns: [c.requestHeader+gin.*Context.requestHeader, gin.*Context.Header+gin.*Context.Set, subtle+base64]
- both are orchestrator functions
- same package
- callees do related work (1.00): [c.Abort+gin.*Context.Abort, writermem.WriteHeaderNow+c.writermem, subtle+base64, bytesconv.StringToBytes+json.API, API.Marshal+bytesconv.StringToBytes, binding+nil]
- same visibility
- same receiver type: plain functions
- call into same packages: [gin]

---

## Match #2 — Code-shape: `0.8484`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:288` | `gin.*Engine.LoadHTMLFiles` | `(...string)` | delims.Left+delims.Right 0.79, delims.Left+delims.Right+engine.SetHTMLTemplate 0.79 |
| **B** | `gin.go:300` | `gin.*Engine.LoadHTMLFS` | `(http.FileSystem, ...string)` | delims.Left+delims.Right 0.79, delims.Left+delims.Right+engine.SetHTMLTemplate 0.79 |

**Profile A:** `delims.Left+delims.Right` 1.00 (dominance)

**Profile B:** `delims.Left+delims.Right` 1.00 (dominance)

**Code similarity:** `ast 0.87  flow 1.00  nesting 1.00  sig 0.50  size 0.87`

**Evidence:** `410.27` (shape 381.84, concept 4.72, call 23.71)

**Trophic:** `0.85`

**Shared structure:**

- `4.93` — `seq[ assign:=(call:Must) ; do(call:SetHTMLTemplate) ]`
- `4.93` — `seq[ if(call:IsDebugging) ; assign:=(call:Must) ]`
- `4.53` — `assign:=(call:Must)`

**Structural overlap:** `0.78` (merge-worthy)

- share 6 callees: [Delims, Funcs, IsDebugging, engine.SetHTMLTemplate, template.Must, template.New]
- overlapping call-graph neighborhoods (1.00): 11 shared
- share patterns: [delims.Left+delims.Right, delims.Left+delims.Right+engine.SetHTMLTemplate]
- both are orchestrator functions
- same package
- callees do related work (1.00): [delims.Left+delims.Right, delims.Left+delims.Right+engine.SetHTMLTemplate]
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Match #3 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `render/toml.go:21` | `render.TOML.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.60, binding+nil 0.35, json.Marshal+json.MarshalIndent 0.31 |
| **B** | `render/yaml.go:21` | `render.YAML.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.60, binding+nil 0.35, json.Marshal+json.MarshalIndent 0.31 |

**Kind:** interface implementations — both implement `Render(http.ResponseWriter) (error)` on `TOML` and `YAML`, in package `render`

**Profile A:** `API.Marshal+bytesconv.StringToBytes` 0.80, `bytesconv.StringToBytes+json.API` 0.20 (dominance)

**Profile B:** `API.Marshal+bytesconv.StringToBytes` 0.80, `bytesconv.StringToBytes+json.API` 0.20 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `177.95` (shape 170.58, concept 3.65, call 3.72)

**Trophic:** `1.00`

**Shared structure:**

- `7.09` — `flow:call:Marshal→return`
- `4.01` — `seq[ if(bin:!=(id,nil)) ; assign=(call:Write) ]`
- `3.68` — `seq[ assign:=(call:Marshal) ; if(bin:!=(id,nil)) ]`

**Structural overlap:** `0.71` (merge-worthy)

- share 2 callees: [r.WriteContentType, w.Write]
- overlapping call-graph neighborhoods (1.00): 12 shared
- share patterns: [API.Marshal+bytesconv.StringToBytes, binding+nil, json.Marshal+json.MarshalIndent]
- both are leaf functions
- same package
- callees do related work (1.00): [w.WriteHeaderNow+w.ResponseWriter]
- same visibility
- both are methods, on TOML and YAML
- call into same packages: [gin]

---

## Match #4 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `binding/toml.go:29` | `binding.decodeToml` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.66, binding+nil 0.35 |
| **B** | `binding/xml.go:28` | `binding.decodeXML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.66, binding+nil 0.35 |

**Profile A:** `delims.Left+delims.Right` 1.00 (dominance)

**Profile B:** `delims.Left+delims.Right` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `174.20` (shape 171.53, concept 2.67, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.53` — `seq[ assign:=(call:NewDecoder) ; if(bin:!=(id,nil)) ]`
- `4.24` — `assign:=(call:NewDecoder)`
- `4.24` — `flow:call:NewDecoder→call:Decode`

**Culture:** B realizes `binding+nil` atypically (typicality 0.18, concept median 0.35, convention 0.60)

**Structural overlap:** `0.71` (merge-worthy)

- share 2 callees: [decoder.Decode, validate]
- share patterns: [binding+nil, delims.Left+delims.Right]
- both are utility functions
- same package
- callers do related work (1.00): [bytes.NewReader+bytes]
- same visibility
- same receiver type: plain functions
- called from same packages: [binding]

---

## Match #5 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `binding/toml.go:29` | `binding.decodeToml` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.66, binding+nil 0.35 |
| **B** | `binding/yaml.go:29` | `binding.decodeYAML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.66, binding+nil 0.35 |

**Profile A:** `delims.Left+delims.Right` 1.00 (dominance)

**Profile B:** `delims.Left+delims.Right` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `174.20` (shape 171.53, concept 2.67, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.53` — `seq[ assign:=(call:NewDecoder) ; if(bin:!=(id,nil)) ]`
- `4.24` — `assign:=(call:NewDecoder)`
- `4.24` — `flow:call:NewDecoder→call:Decode`

**Structural overlap:** `0.71` (merge-worthy)

- share 2 callees: [decoder.Decode, validate]
- share patterns: [binding+nil, delims.Left+delims.Right]
- both are utility functions
- same package
- callers do related work (1.00): [bytes.NewReader+bytes]
- same visibility
- same receiver type: plain functions
- called from same packages: [binding]

---

## Match #6 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `binding/xml.go:28` | `binding.decodeXML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.66, binding+nil 0.35 |
| **B** | `binding/yaml.go:29` | `binding.decodeYAML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.66, binding+nil 0.35 |

**Profile A:** `delims.Left+delims.Right` 1.00 (dominance)

**Profile B:** `delims.Left+delims.Right` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `174.20` (shape 171.53, concept 2.67, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.53` — `seq[ assign:=(call:NewDecoder) ; if(bin:!=(id,nil)) ]`
- `4.24` — `assign:=(call:NewDecoder)`
- `4.24` — `flow:call:NewDecoder→call:Decode`

**Culture:** A realizes `binding+nil` atypically (typicality 0.18, concept median 0.35, convention 0.60)

**Structural overlap:** `0.71` (merge-worthy)

- share 2 callees: [decoder.Decode, validate]
- share patterns: [binding+nil, delims.Left+delims.Right]
- both are utility functions
- same package
- callers do related work (1.00): [bytes.NewReader+bytes]
- same visibility
- same receiver type: plain functions
- called from same packages: [binding]

---

## Match #7 — Code-shape: `0.7364`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `routergroup.go:181` | `gin.*RouterGroup.staticFileHandler` | `(string, HandlerFunc) (IRoutes)` | group.calculateAbsolutePath+group.engine 0.55 |
| **B** | `routergroup.go:203` | `gin.*RouterGroup.StaticFS` | `(string, http.FileSystem) (IRoutes)` | group.calculateAbsolutePath+group.engine 0.54 |

**Profile A:** `group.calculateAbsolutePath+group.engine` 1.00 (dominance)

**Profile B:** `group.calculateAbsolutePath+group.engine` 1.00 (dominance)

**Code similarity:** `ast 0.69  flow 1.00  nesting 1.00  sig 0.50  size 0.71`

**Evidence:** `315.38` (shape 293.23, concept 1.87, call 20.27)

**Trophic:** `0.88`

**Shared structure:**

- `9.05` — `flow:param→call:Contains`
- `4.93` — `seq[ do(call:GET) ; do(call:HEAD) ]`
- `4.93` — `seq[ do(call:HEAD) ; return(call:returnObj) ]`

**Structural overlap:** `0.60` (merge-worthy)

- share 5 callees: [group.GET, group.HEAD, group.returnObj, panic, strings.Contains]
- overlapping call-graph neighborhoods (0.50): 7 shared
- share patterns: [group.calculateAbsolutePath+group.engine]
- related roles: passthrough ≈ orchestrator (both high fan-out, 0.50)
- same package
- callees do related work (0.27): [group.calculateAbsolutePath+group.engine]
- same receiver type: RouterGroup
- called from same packages: [gin]
- call into same packages: [gin]

---

## Match #8 — Code-shape: `0.6573`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:581` | `gin.*Engine.RunUnix` | `(string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.78, http.Server+engine.Handler 0.52, binding+nil 0.35 |
| **B** | `gin.go:645` | `gin.*Engine.RunListener` | `(net.Listener) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.76, http.Server+engine.Handler 0.50 |

**Profile A:** `http.Server+engine.Handler` 1.00 (dominance)

**Profile B:** `http.Server+engine.Handler` 1.00 (dominance)

**Code similarity:** `ast 0.62  flow 0.94  nesting 0.99  sig 0.33  size 0.69`

**Evidence:** `357.02` (shape 340.41, concept 4.34, call 12.27)

**Trophic:** `0.80`

**Shared structure:**

- `5.97` — `do(call:debugPrint)`
- `4.93` — `seq[ assign:=(unary) ; assign=(call:Serve) ]`
- `4.93` — `seq[ assign=(call:Serve) ; return() ]`

**Structural overlap:** `0.71` (merge-worthy)

- share 5 callees: [debugPrint, debugPrintError, engine.Handler, engine.isUnsafeTrustedProxies, server.Serve]
- overlapping call-graph neighborhoods (1.00): 19 shared
- share patterns: [delims.Left+delims.Right+engine.SetHTMLTemplate, http.Server+engine.Handler]
- both are orchestrator functions
- same package
- callees do related work (1.00): [fmt.Fprintf+runtime, gin.IsDebugging+gin.debugPrint, delims.Left+delims.Right+engine.SetHTMLTemplate, binding+nil]
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Match #9 — Code-shape: `0.7320`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:561` | `gin.*Engine.RunTLS` | `(string, string, string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.77, http.Server+engine.Handler 0.52 |
| **B** | `gin.go:645` | `gin.*Engine.RunListener` | `(net.Listener) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.76, http.Server+engine.Handler 0.50 |

**Profile A:** `http.Server+engine.Handler` 1.00 (dominance)

**Profile B:** `http.Server+engine.Handler` 1.00 (dominance)

**Code similarity:** `ast 0.72  flow 1.00  nesting 1.00  sig 0.33  size 0.98`

**Evidence:** `298.76` (shape 282.16, concept 4.34, call 12.27)

**Trophic:** `0.80`

**Shared structure:**

- `5.97` — `do(call:debugPrint)`
- `4.93` — `seq[ if(call:isUnsafeTrustedProxies) ; assign:=(unary) ]`
- `4.01` — `seq[ do(call:debugPrint) ; defer(funclit) ]`

**Structural overlap:** `0.73` (merge-worthy)

- share 4 callees: [debugPrint, debugPrintError, engine.Handler, engine.isUnsafeTrustedProxies]
- overlapping call-graph neighborhoods (1.00): 19 shared
- share patterns: [delims.Left+delims.Right+engine.SetHTMLTemplate, http.Server+engine.Handler]
- both are orchestrator functions
- same package
- callees do related work (1.00): [fmt.Fprintf+runtime, gin.IsDebugging+gin.debugPrint, delims.Left+delims.Right+engine.SetHTMLTemplate, binding+nil]
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Match #10 — Code-shape: `0.9357`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `render/protobuf.go:21` | `render.ProtoBuf.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.60, binding+nil 0.35, json.Marshal+json.MarshalIndent 0.31 |
| **B** | `render/toml.go:21` | `render.TOML.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.60, binding+nil 0.35, json.Marshal+json.MarshalIndent 0.31 |

**Kind:** interface implementations — both implement `Render(http.ResponseWriter) (error)` on `ProtoBuf` and `TOML`, in package `render`

**Profile A:** `API.Marshal+bytesconv.StringToBytes` 0.80, `bytesconv.StringToBytes+json.API` 0.20 (dominance)

**Profile B:** `API.Marshal+bytesconv.StringToBytes` 0.80, `bytesconv.StringToBytes+json.API` 0.20 (dominance)

**Code similarity:** `ast 0.89  flow 1.00  nesting 1.00  sig 1.00  size 0.89`

**Evidence:** `173.02` (shape 165.65, concept 3.65, call 3.72)

**Trophic:** `0.92`

**Shared structure:**

- `7.09` — `flow:call:Marshal→return`
- `4.01` — `seq[ if(bin:!=(id,nil)) ; assign=(call:Write) ]`
- `3.68` — `seq[ assign:=(call:Marshal) ; if(bin:!=(id,nil)) ]`

**Structural overlap:** `0.71` (merge-worthy)

- share 2 callees: [r.WriteContentType, w.Write]
- overlapping call-graph neighborhoods (1.00): 12 shared
- share patterns: [API.Marshal+bytesconv.StringToBytes, binding+nil, json.Marshal+json.MarshalIndent]
- both are leaf functions
- same package
- callees do related work (1.00): [w.WriteHeaderNow+w.ResponseWriter]
- same visibility
- both are methods, on ProtoBuf and TOML
- call into same packages: [gin]

---

## Families

27 families, 155 functions in a family, largest 17 members; 235 edges scored here that retrieval never proposed

### Family 1 — 15 members, every pair `>= 0.55` code-shape, evidence `2751`  (49 edges scored here)

_Not drawn: 15 members is 105 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `context.go:1180` | `gin.*Context.IndentedJSON` | `(int, any)` | binding+nil 0.31 |
| `context.go:1187` | `gin.*Context.SecureJSON` | `(int, any)` | binding+nil 0.31 |
| `context.go:1205` | `gin.*Context.JSON` | `(int, any)` | binding+nil 0.31 |
| `context.go:1211` | `gin.*Context.AsciiJSON` | `(int, any)` | binding+nil 0.31 |
| `context.go:1217` | `gin.*Context.PureJSON` | `(int, any)` | binding+nil 0.31 |
| `context.go:1223` | `gin.*Context.XML` | `(int, any)` | binding+nil 0.31 |
| `context.go:1228` | `gin.*Context.YAML` | `(int, any)` | binding+nil 0.31 |
| `context.go:1233` | `gin.*Context.TOML` | `(int, any)` | binding+nil 0.31 |
| `context.go:1238` | `gin.*Context.ProtoBuf` | `(int, any)` | binding+nil 0.31 |
| `context.go:1243` | `gin.*Context.BSON` | `(int, any)` | binding+nil 0.31 |

_5 more members not listed._

### Family 2 — 5 members, every pair `>= 0.56` code-shape, evidence `2656`

```mermaid
flowchart LR
    m0["gin.*Engine.Run"]
    m1["gin.*Engine.RunTLS"]
    m2["gin.*Engine.RunUnix"]
    m3["gin.*Engine.RunQUIC"]
    m4["gin.*Engine.RunListener"]
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

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `gin.go:540` | `gin.*Engine.Run` | `(...string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.78, http.Server+engine.Handler 0.50 |
| `gin.go:561` | `gin.*Engine.RunTLS` | `(string, string, string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.77, http.Server+engine.Handler 0.52 |
| `gin.go:581` | `gin.*Engine.RunUnix` | `(string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.78, http.Server+engine.Handler 0.52, binding+nil 0.35 |
| `gin.go:630` | `gin.*Engine.RunQUIC` | `(string, string, string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.76, http.Server+engine.Handler 0.50 |
| `gin.go:645` | `gin.*Engine.RunListener` | `(net.Listener) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.76, http.Server+engine.Handler 0.50 |

### Family 3 — 5 members, every pair `>= 0.50` code-shape, evidence `2604`

```mermaid
flowchart LR
    m0["gin.*Engine.RunTLS"]
    m1["gin.*Engine.RunUnix"]
    m2["gin.*Engine.RunFd"]
    m3["gin.*Engine.RunQUIC"]
    m4["gin.*Engine.RunListener"]
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

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `gin.go:561` | `gin.*Engine.RunTLS` | `(string, string, string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.77, http.Server+engine.Handler 0.52 |
| `gin.go:581` | `gin.*Engine.RunUnix` | `(string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.78, http.Server+engine.Handler 0.52, binding+nil 0.35 |
| `gin.go:607` | `gin.*Engine.RunFd` | `(int) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.74, binding+nil 0.35 |
| `gin.go:630` | `gin.*Engine.RunQUIC` | `(string, string, string) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.76, http.Server+engine.Handler 0.50 |
| `gin.go:645` | `gin.*Engine.RunListener` | `(net.Listener) (error)` | delims.Left+delims.Right+engine.SetHTMLTemplate 0.76, http.Server+engine.Handler 0.50 |

### Family 4 — 5 members, every pair `>= 0.64` code-shape, evidence `1402`

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

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `binding/json.go:44` | `binding.decodeJSON` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.65, binding+nil 0.35 |
| `binding/msgpack.go:31` | `binding.decodeMsgPack` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.56, binding+nil 0.35 |
| `binding/toml.go:29` | `binding.decodeToml` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.66, binding+nil 0.35 |
| `binding/xml.go:28` | `binding.decodeXML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.66, binding+nil 0.35 |
| `binding/yaml.go:29` | `binding.decodeYAML` | `(io.Reader, any) (error)` | delims.Left+delims.Right 0.66, binding+nil 0.35 |

### Family 5 — 5 members, every pair `>= 0.62` code-shape, evidence `1304`, interface implementations of `Render(http.ResponseWriter) (error)`, in package `render`

```mermaid
flowchart LR
    m0["render.BSON.Render"]
    m1["render.IndentedJSON.Render"]
    m2["render.ProtoBuf.Render"]
    m3["render.TOML.Render"]
    m4["render.YAML.Render"]
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

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `render/bson.go:21` | `render.BSON.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.54, binding+nil 0.35, json.Marshal+json.MarshalIndent 0.31 |
| `render/json.go:78` | `render.IndentedJSON.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.55, bytesconv.StringToBytes+json.API 0.42, json.Marshal+json.MarshalIndent 0.41, binding+nil 0.35 |
| `render/protobuf.go:21` | `render.ProtoBuf.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.60, binding+nil 0.35, json.Marshal+json.MarshalIndent 0.31 |
| `render/toml.go:21` | `render.TOML.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.60, binding+nil 0.35, json.Marshal+json.MarshalIndent 0.31 |
| `render/yaml.go:21` | `render.YAML.Render` | `(http.ResponseWriter) (error)` | API.Marshal+bytesconv.StringToBytes 0.60, binding+nil 0.35, json.Marshal+json.MarshalIndent 0.31 |

_22 more families not listed._


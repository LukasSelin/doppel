# gin

HTTP framework; a small core surrounded by generated-looking binding and render variants

**What this rung shows:** family clones — the case corroborated ranking was tuned to separate

| | |
|---|---|
| Corpus | [gin](https://github.com/gin-gonic/gin) |
| Pinned at | `v1.12.0` (`73726dc606796a025971fe451f0aa6f1b9b847f6`) |
| Project since | 2014 |
| doppel | `706150c` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 4 concepts modeled, 9 associations, 1 unusual realizations
Habitats: 5 modeled, 13 misfits; most uniform binding (norm 0.92), most diverse json (norm 0.71)
Conventions: strongest mapping (0.73), loosest caching (0.37)
Ecosystems: 101 profiled (101 dominance, 0 coalition, 0 conflict, 0 weak)
Found 497 functions. Retrieving candidates...
Retrieval: shape 154, concept 180, call 609 -> 896 unique pairs
  concept-only 19.4%  call-only 63.2%  suppressed-shape functions: 1  large identity buckets: 0  surviving patterns: 1491
Running structural comparison on 896 pairs...
Families: 24 over 48 components, 109 functions in a family, 117 edges completed
```

# Code Similarity Report

**Functions analyzed:** 497 | **Threshold:** 0.60 | **Pairs found:** 10

---

## Match #1 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `auth.go:48` | `gin.BasicAuthForRealm` | ` ` | — |
| **B** | `auth.go:98` | `gin.BasicAuthForProxy` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `305.26` (shape 272.00, concept 0.00, call 33.26)

**Trophic:** `1.00`

**Shared structure:**

- `4.93` — `seq[ assign:=(call:processAccounts) ; return(funclit) ]`
- `4.93` — `seq[ assign:=(call:searchCredential) ; if(unary) ]`
- `4.93` — `seq[ assign=(bin) ; assign:=(call:processAccounts) ]`

**Structural overlap:** `0.63` (merge-worthy)

- share 7 callees: [c.AbortWithStatus, c.Header, c.Set, c.requestHeader, pairs.searchCredential, processAccounts, strconv.Quote]
- overlapping call-graph neighborhoods (0.97): 32 shared
- both are orchestrator functions
- same package
- callees do related work (1.00): [concurrency]
- same visibility
- same receiver type: plain functions
- call into same packages: [gin]

---

## Match #2 — Code-shape: `0.8484`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:288` | `gin.*Engine.LoadHTMLFiles` | `—` | validation |
| **B** | `gin.go:300` | `gin.*Engine.LoadHTMLFS` | `—` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.87  flow 1.00  sig 0.50  size 0.87`

**Evidence:** `203.04` (shape 178.51, concept 0.82, call 23.71)

**Trophic:** `0.86`

**Shared structure:**

- `4.93` — `seq[ assign:=(call:Must) ; do(call:SetHTMLTemplate) ]`
- `4.93` — `seq[ if(call:IsDebugging) ; assign:=(call:Must) ]`
- `4.53` — `assign:=(call:Must)`

**Structural overlap:** `0.78` (merge-worthy)

- share 6 callees: [Delims, Funcs, IsDebugging, engine.SetHTMLTemplate, template.Must, template.New]
- overlapping call-graph neighborhoods (1.00): 11 shared
- share patterns: [validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [concurrency]
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Match #3 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `binding/toml.go:29` | `binding.decodeToml` | ` ` | validation |
| **B** | `binding/xml.go:28` | `binding.decodeXML` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `73.97` (shape 73.14, concept 0.82, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.53` — `seq[ assign:=(call:NewDecoder) ; if(bin:!=(id,nil)) ]`
- `4.24` — `assign:=(call:NewDecoder)`
- `4.01` — `assign:=(call:Decode)`

**Structural overlap:** `0.66` (merge-worthy)

- share 2 callees: [decoder.Decode, validate]
- share patterns: [validation]
- both are utility functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [binding]

---

## Match #4 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `binding/toml.go:29` | `binding.decodeToml` | ` ` | validation |
| **B** | `binding/yaml.go:29` | `binding.decodeYAML` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `73.97` (shape 73.14, concept 0.82, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.53` — `seq[ assign:=(call:NewDecoder) ; if(bin:!=(id,nil)) ]`
- `4.24` — `assign:=(call:NewDecoder)`
- `4.01` — `assign:=(call:Decode)`

**Structural overlap:** `0.66` (merge-worthy)

- share 2 callees: [decoder.Decode, validate]
- share patterns: [validation]
- both are utility functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [binding]

---

## Match #5 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `binding/xml.go:28` | `binding.decodeXML` | ` ` | validation |
| **B** | `binding/yaml.go:29` | `binding.decodeYAML` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `73.97` (shape 73.14, concept 0.82, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.53` — `seq[ assign:=(call:NewDecoder) ; if(bin:!=(id,nil)) ]`
- `4.24` — `assign:=(call:NewDecoder)`
- `4.01` — `assign:=(call:Decode)`

**Structural overlap:** `0.66` (merge-worthy)

- share 2 callees: [decoder.Decode, validate]
- share patterns: [validation]
- both are utility functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [binding]

---

## Match #6 — Code-shape: `0.6549`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:581` | `gin.*Engine.RunUnix` | ` ` | — |
| **B** | `gin.go:645` | `gin.*Engine.RunListener` | ` ` | — |

**Code similarity:** `ast 0.62  flow 0.94  sig 0.33  size 0.69`

**Evidence:** `205.26` (shape 192.99, concept 0.00, call 12.27)

**Trophic:** `0.84`

**Shared structure:**

- `5.97` — `do(call:debugPrint)`
- `4.93` — `seq[ assign:=(unary) ; assign=(call:Serve) ]`
- `4.93` — `seq[ assign=(call:Serve) ; return() ]`

**Structural overlap:** `0.50` (merge-worthy)

- share 5 callees: [debugPrint, debugPrintError, engine.Handler, engine.isUnsafeTrustedProxies, server.Serve]
- overlapping call-graph neighborhoods (1.00): 19 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Match #7 — Code-shape: `0.6100`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:272` | `gin.*Engine.LoadHTMLGlob` | `—` | validation |
| **B** | `gin.go:288` | `gin.*Engine.LoadHTMLFiles` | `—` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.60  flow 1.00  sig 0.00  size 0.87`

**Evidence:** `171.27` (shape 146.74, concept 0.82, call 23.71)

**Trophic:** `0.75`

**Shared structure:**

- `4.53` — `assign:=(call:Must)`
- `4.53` — `do(call:SetHTMLTemplate)`
- `4.24` — `seq[ assign=(composite) ; return() ]`

**Structural overlap:** `0.75` (merge-worthy)

- share 6 callees: [Delims, Funcs, IsDebugging, engine.SetHTMLTemplate, template.Must, template.New]
- overlapping call-graph neighborhoods (0.92): 11 shared
- share patterns: [validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [concurrency]
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Match #8 — Code-shape: `0.7320`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:561` | `gin.*Engine.RunTLS` | ` ` | — |
| **B** | `gin.go:645` | `gin.*Engine.RunListener` | ` ` | — |

**Code similarity:** `ast 0.72  flow 1.00  sig 0.33  size 0.98`

**Evidence:** `171.66` (shape 159.39, concept 0.00, call 12.27)

**Trophic:** `0.83`

**Shared structure:**

- `5.97` — `do(call:debugPrint)`
- `4.93` — `seq[ if(call:isUnsafeTrustedProxies) ; assign:=(unary) ]`
- `4.01` — `seq[ do(call:debugPrint) ; defer(funclit) ]`

**Structural overlap:** `0.51` (merge-worthy)

- share 4 callees: [debugPrint, debugPrintError, engine.Handler, engine.isUnsafeTrustedProxies]
- overlapping call-graph neighborhoods (1.00): 19 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Match #9 — Code-shape: `0.7364`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `routergroup.go:181` | `gin.*RouterGroup.staticFileHandler` | ` ` | — |
| **B** | `routergroup.go:203` | `gin.*RouterGroup.StaticFS` | ` ` | — |

**Code similarity:** `ast 0.69  flow 1.00  sig 0.50  size 0.71`

**Evidence:** `170.37` (shape 150.10, concept 0.00, call 20.27)

**Trophic:** `0.92`

**Shared structure:**

- `4.93` — `seq[ do(call:GET) ; do(call:HEAD) ]`
- `4.93` — `seq[ do(call:HEAD) ; return(call:returnObj) ]`
- `4.93` — `do(call:GET)`

**Structural overlap:** `0.41` (merge-worthy)

- share 5 callees: [group.GET, group.HEAD, group.returnObj, panic, strings.Contains]
- overlapping call-graph neighborhoods (0.50): 7 shared
- related roles: passthrough ≈ orchestrator (both high fan-out, 0.50)
- same package
- same receiver type: RouterGroup
- called from same packages: [gin]
- call into same packages: [gin]

---

## Match #10 — Code-shape: `0.7447`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:561` | `gin.*Engine.RunTLS` | ` ` | — |
| **B** | `gin.go:630` | `gin.*Engine.RunQUIC` | ` ` | — |

**Code similarity:** `ast 0.57  flow 1.00  sig 1.00  size 0.79`

**Evidence:** `144.09` (shape 131.82, concept 0.00, call 12.27)

**Trophic:** `0.85`

**Shared structure:**

- `5.97` — `do(call:debugPrint)`
- `4.01` — `seq[ do(call:debugPrint) ; defer(funclit) ]`
- `3.83` — `seq[ defer(funclit) ; if(call:isUnsafeTrustedProxies) ]`

**Structural overlap:** `0.54` (merge-worthy)

- share 4 callees: [debugPrint, debugPrintError, engine.Handler, engine.isUnsafeTrustedProxies]
- overlapping call-graph neighborhoods (1.00): 19 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: Engine
- call into same packages: [gin]

---

## Families

24 families, 109 functions in a family, largest 14 members; 117 edges scored here that retrieval never proposed

### Family 1 — 14 members, every pair `>= 1.00` code-shape  (55 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `render/bson.go:32` | `render.BSON.WriteContentType` | `—` | — |
| `render/html.go:99` | `render.HTML.WriteContentType` | `—` | — |
| `render/json.go:62` | `render.JSON.WriteContentType` | `—` | — |
| `render/json.go:89` | `render.IndentedJSON.WriteContentType` | `—` | — |
| `render/json.go:112` | `render.SecureJSON.WriteContentType` | `—` | — |
| `render/json.go:150` | `render.JsonpJSON.WriteContentType` | `—` | — |
| `render/json.go:179` | `render.AsciiJSON.WriteContentType` | `—` | — |
| `render/json.go:192` | `render.PureJSON.WriteContentType` | `—` | — |
| `render/msgpack.go:29` | `render.MsgPack.WriteContentType` | `—` | — |
| `render/protobuf.go:34` | `render.ProtoBuf.WriteContentType` | `—` | — |

_4 more members not listed._

### Family 2 — 13 members, every pair `>= 0.74` code-shape  (33 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `context.go:1180` | `gin.*Context.IndentedJSON` | `—` | — |
| `context.go:1187` | `gin.*Context.SecureJSON` | `—` | — |
| `context.go:1205` | `gin.*Context.JSON` | `—` | — |
| `context.go:1211` | `gin.*Context.AsciiJSON` | `—` | — |
| `context.go:1217` | `gin.*Context.PureJSON` | `—` | — |
| `context.go:1223` | `gin.*Context.XML` | `—` | — |
| `context.go:1228` | `gin.*Context.YAML` | `—` | — |
| `context.go:1233` | `gin.*Context.TOML` | `—` | — |
| `context.go:1238` | `gin.*Context.ProtoBuf` | `—` | — |
| `context.go:1243` | `gin.*Context.BSON` | `—` | — |

_3 more members not listed._

### Family 3 — 10 members, every pair `>= 0.60` code-shape  (15 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `context.go:1180` | `gin.*Context.IndentedJSON` | `—` | — |
| `context.go:1205` | `gin.*Context.JSON` | `—` | — |
| `context.go:1211` | `gin.*Context.AsciiJSON` | `—` | — |
| `context.go:1217` | `gin.*Context.PureJSON` | `—` | — |
| `context.go:1223` | `gin.*Context.XML` | `—` | — |
| `context.go:1228` | `gin.*Context.YAML` | `—` | — |
| `context.go:1233` | `gin.*Context.TOML` | `—` | — |
| `context.go:1238` | `gin.*Context.ProtoBuf` | `—` | — |
| `context.go:1243` | `gin.*Context.BSON` | `—` | — |
| `context.go:1313` | `gin.*Context.SSEvent` | `—` | — |

### Family 4 — 8 members, every pair `>= 0.60` code-shape  (7 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `context.go:763` | `gin.*Context.BindJSON` | ` ` | — |
| `context.go:768` | `gin.*Context.BindXML` | ` ` | — |
| `context.go:773` | `gin.*Context.BindQuery` | ` ` | — |
| `context.go:778` | `gin.*Context.BindYAML` | ` ` | — |
| `context.go:783` | `gin.*Context.BindTOML` | ` ` | — |
| `context.go:788` | `gin.*Context.BindPlain` | ` ` | — |
| `context.go:793` | `gin.*Context.BindHeader` | ` ` | — |
| `deprecated.go:17` | `gin.*Context.BindWith` | ` ` | — |

### Family 5 — 7 members, every pair `>= 1.00` code-shape  (6 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `context.go:867` | `gin.*Context.ShouldBindJSON` | ` ` | — |
| `context.go:873` | `gin.*Context.ShouldBindXML` | ` ` | — |
| `context.go:879` | `gin.*Context.ShouldBindQuery` | ` ` | — |
| `context.go:885` | `gin.*Context.ShouldBindYAML` | ` ` | — |
| `context.go:891` | `gin.*Context.ShouldBindTOML` | ` ` | — |
| `context.go:897` | `gin.*Context.ShouldBindPlain` | ` ` | — |
| `context.go:903` | `gin.*Context.ShouldBindHeader` | ` ` | — |

_19 more families not listed._


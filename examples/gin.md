# gin

HTTP framework; a small core surrounded by generated-looking binding and render variants

**What this rung shows:** family clones — the case corroborated ranking was tuned to separate

| | |
|---|---|
| Corpus | [gin](https://github.com/gin-gonic/gin) |
| Pinned at | `v1.12.0` (`73726dc606796a025971fe451f0aa6f1b9b847f6`) |
| Project since | 2014 |
| doppel | `0fe7542` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 5 concepts modeled, 11 associations, 2 unusual realizations
Habitats: 5 modeled, 17 misfits; most uniform binding (norm 0.91), most diverse json (norm 0.63)
Conventions: strongest serialization (0.72), loosest caching (0.37)
Ecosystems: 128 profiled (128 dominance, 0 coalition, 0 conflict, 0 weak)
Found 497 functions. Retrieving candidates...
Retrieval: shape 154, concept 317, call 609 -> 1021 unique pairs
  concept-only 29.3%  call-only 54.6%  suppressed-shape functions: 1  large identity buckets: 0  surviving patterns: 1491
Running structural comparison on 1021 pairs...
```

# Code Similarity Report

**Functions analyzed:** 497 | **Threshold:** 0.60 | **Pairs found:** 10

---

## Match #1 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `auth.go:48` | `gin.BasicAuthForRealm` | ` ` | — |
| **B** | `auth.go:98` | `gin.BasicAuthForProxy` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

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

**Code similarity:** `ast 0.87  flow 1.00  nesting 1.00  sig 0.50  size 0.87`

**Evidence:** `203.54` (shape 178.51, concept 1.32, call 23.71)

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
| **B** | `binding/yaml.go:29` | `binding.decodeYAML` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `74.47` (shape 73.14, concept 1.32, call 0.00)

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

## Match #4 — Code-shape: `0.6573`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:581` | `gin.*Engine.RunUnix` | ` ` | file_io |
| **B** | `gin.go:645` | `gin.*Engine.RunListener` | ` ` | — |

**Profile A:** `file_io` 1.00 (dominance)

**Code similarity:** `ast 0.62  flow 0.94  nesting 0.99  sig 0.33  size 0.69`

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

## Match #5 — Code-shape: `0.6100`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:272` | `gin.*Engine.LoadHTMLGlob` | `—` | validation |
| **B** | `gin.go:288` | `gin.*Engine.LoadHTMLFiles` | `—` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.60  flow 1.00  nesting 1.00  sig 0.00  size 0.87`

**Evidence:** `171.77` (shape 146.74, concept 1.32, call 23.71)

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

## Match #6 — Code-shape: `0.7320`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:561` | `gin.*Engine.RunTLS` | ` ` | — |
| **B** | `gin.go:645` | `gin.*Engine.RunListener` | ` ` | — |

**Code similarity:** `ast 0.72  flow 1.00  nesting 1.00  sig 0.33  size 0.98`

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

## Match #7 — Code-shape: `0.7364`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `routergroup.go:181` | `gin.*RouterGroup.staticFileHandler` | ` ` | — |
| **B** | `routergroup.go:203` | `gin.*RouterGroup.StaticFS` | ` ` | — |

**Code similarity:** `ast 0.69  flow 1.00  nesting 1.00  sig 0.50  size 0.71`

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

## Match #8 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `binding/toml.go:29` | `binding.decodeToml` | ` ` | validation |
| **B** | `binding/xml.go:28` | `binding.decodeXML` | ` ` | validation, serialization |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `74.47` (shape 73.14, concept 1.32, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.53` — `seq[ assign:=(call:NewDecoder) ; if(bin:!=(id,nil)) ]`
- `4.24` — `assign:=(call:NewDecoder)`
- `4.01` — `assign:=(call:Decode)`

**Structural overlap:** `0.57` (merge-worthy)

- share 2 callees: [decoder.Decode, validate]
- share patterns: [validation]
- both are utility functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [binding]

---

## Match #9 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `binding/xml.go:28` | `binding.decodeXML` | ` ` | validation, serialization |
| **B** | `binding/yaml.go:29` | `binding.decodeYAML` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `74.47` (shape 73.14, concept 1.32, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.53` — `seq[ assign:=(call:NewDecoder) ; if(bin:!=(id,nil)) ]`
- `4.24` — `assign:=(call:NewDecoder)`
- `4.01` — `assign:=(call:Decode)`

**Structural overlap:** `0.57` (merge-worthy)

- share 2 callees: [decoder.Decode, validate]
- share patterns: [validation]
- both are utility functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [binding]

---

## Match #10 — Code-shape: `0.7447`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `gin.go:561` | `gin.*Engine.RunTLS` | ` ` | — |
| **B** | `gin.go:630` | `gin.*Engine.RunQUIC` | ` ` | — |

**Code similarity:** `ast 0.57  flow 1.00  nesting 1.00  sig 1.00  size 0.79`

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


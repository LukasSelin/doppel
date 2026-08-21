# chi

HTTP router; a narrow core with a middleware package beside it

**What this rung shows:** a corpus small enough to read end to end, where every reported pair can be checked

| | |
|---|---|
| Corpus | [chi](https://github.com/go-chi/chi) |
| Pinned at | `v5.3.2` (`38939062c5df4d3e8814aad1a488983112627ced`) |
| Project since | 2015 |
| doppel | `0fe7542` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 1 concepts modeled, 6 associations, 0 unusual realizations
Habitats: 3 modeled, 4 misfits; most uniform chi (norm 0.91), most diverse middleware (norm 0.85)
Conventions: strongest validation (0.41), loosest validation (0.41)
Ecosystems: 53 profiled (53 dominance, 0 coalition, 0 conflict, 0 weak)
Found 254 functions. Retrieving candidates...
Retrieval: shape 86, concept 36, call 542 -> 617 unique pairs
  concept-only 5.0%  call-only 80.6%  suppressed-shape functions: 0  large identity buckets: 0  surviving patterns: 1166
Running structural comparison on 617 pairs...
```

# Code Similarity Report

**Functions analyzed:** 254 | **Threshold:** 0.60 | **Pairs found:** 10

---

## Match #1 — Code-shape: `0.8396`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/recoverer.go:132` | `middleware.prettyStack.decorateFuncCallLine` | ` ` | — |
| **B** | `middleware/recoverer.go:172` | `middleware.prettyStack.decorateSourceLine` | ` ` | — |

**Code similarity:** `ast 0.74  flow 1.00  nesting 0.90  sig 1.00  size 0.99`

**Evidence:** `430.04` (shape 414.55, concept 0.00, call 15.49)

**Trophic:** `0.70`

**Shared structure:**

- `14.63` — `do(call:cW)`
- `5.57` — `assign:=(id)`
- `4.57` — `seq[ assign:=(call:LastIndex) ; if(bin:<(id,lit:INT)) ]`

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

## Match #2 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `_examples/todos-resource/todos.go:12` | `main.todosResource.Routes` | ` ` | — |
| **B** | `_examples/todos-resource/users.go:12` | `main.usersResource.Routes` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 0.90`

**Evidence:** `206.36` (shape 200.70, concept 0.00, call 5.66)

**Trophic:** `0.96`

**Shared structure:**

- `6.95` — `do(call:Put)`
- `5.12` — `do(call:Get)`
- `4.57` — `seq[ do(call:Post) ; do(call:Put) ]`

**Structural overlap:** `0.55` (merge-worthy)

- share 6 callees: [chi.NewRouter, r.Delete, r.Get, r.Post, r.Put, r.Route]
- overlapping call-graph neighborhoods (1.00): 6 shared
- both are orchestrator functions
- same package
- same visibility
- both are methods, on todosResource and usersResource
- call into same packages: [chi]

---

## Match #3 — Code-shape: `0.8367`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tree.go:559` | `chi.*node.findEdge` | ` ` | — |
| **B** | `tree.go:850` | `chi.nodes.findEdge` | ` ` | — |

**Code similarity:** `ast 0.85  flow 0.96  nesting 0.74  sig 0.67  size 0.80`

**Evidence:** `275.77` (shape 275.77, concept 0.00, call 0.00)

**Trophic:** `0.93`

**Shared structure:**

- `10.43` — `assign=(bin)`
- `4.57` — `seq[ assign:=(call:len) ; assign:=(lit:INT) ]`
- `4.57` — `seq[ assign=(bin) ; if(bin:>(id,sel)) ]`

**Structural overlap:** `0.50` (merge-worthy)

- share 1 callees: [len]
- both are leaf functions
- same package
- same visibility
- both are methods, on *node and nodes

---

## Match #4 — Code-shape: `0.9163`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `mux.go:203` | `chi.*Mux.NotFound` | `—` | — |
| **B** | `mux.go:223` | `chi.*Mux.MethodNotAllowed` | `—` | — |

**Code similarity:** `ast 0.86  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `173.04` (shape 158.92, concept 0.00, call 14.13)

**Trophic:** `0.87`

**Shared structure:**

- `5.57` — `assign:=(id)`
- `5.12` — `assign=(sel)`
- `4.57` — `seq[ assign:=(id) ; if(bin:&&(sel,bin)) ]`

**Structural overlap:** `0.67` (merge-worthy)

- share 3 callees: [Chain, HandlerFunc, m.updateSubRoutes]
- share 1 callers: [chi.*Mux.Mount]
- overlapping call-graph neighborhoods (1.00): 13 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: Mux
- called from same packages: [chi]
- call into same packages: [chi]

---

## Match #5 — Code-shape: `0.7972`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/route_headers.go:48` | `middleware.HeaderRouter.Route` | ` ` | — |
| **B** | `middleware/route_headers.go:58` | `middleware.HeaderRouter.RouteAny` | ` ` | — |

**Code similarity:** `ast 0.79  flow 0.82  nesting 1.00  sig 0.75  size 0.74`

**Evidence:** `136.30` (shape 128.22, concept 0.00, call 8.08)

**Trophic:** `0.82`

**Shared structure:**

- `4.57` — `seq[ assign:=(index) ; if(bin:==(id,nil)) ]`
- `4.57` — `seq[ assign=(call:ToLower) ; assign:=(index) ]`
- `4.17` — `seq[ assign=(call:append) ; return(id) ]`

**Structural overlap:** `0.58` (merge-worthy)

- share 3 callees: [NewPattern, append, strings.ToLower]
- overlapping call-graph neighborhoods (1.00): 1 shared
- both are leaf functions
- same package
- same visibility
- same receiver type: HeaderRouter
- call into same packages: [middleware]

---

## Match #6 — Code-shape: `0.8429`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `_examples/hello-world/main.go:10` | `main.main` | `—` | — |
| **B** | `_examples/todos-resource/main.go:14` | `main.main` | `—` | — |

**Code similarity:** `ast 0.74  flow 1.00  nesting 1.00  sig 1.00  size 0.69`

**Evidence:** `172.32` (shape 121.20, concept 0.00, call 51.12)

**Trophic:** `0.71`

**Shared structure:**

- `8.90` — `do(call:Use)`
- `6.64` — `seq[ do(call:Use) ; do(call:Use) ]`
- `3.07` — `seq[ do(call:Use) ; do(call:Get) ]`

**Structural overlap:** `0.52` (merge-worthy)

- share 5 callees: [chi.NewRouter, http.ListenAndServe, r.Get, r.Use, w.Write]
- overlapping call-graph neighborhoods (1.00): 33 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: plain functions
- call into same packages: [chi, main, middleware]

---

## Match #7 — Code-shape: `0.6704`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `middleware/content_encoding.go:10` | `middleware.AllowContentEncoding` | ` ` | — |
| **B** | `middleware/content_type.go:20` | `middleware.AllowContentType` | ` ` | — |

**Code similarity:** `ast 0.62  flow 0.98  nesting 0.98  sig 0.33  size 0.98`

**Evidence:** `204.55` (shape 195.53, concept 0.00, call 9.01)

**Trophic:** `0.76`

**Shared structure:**

- `4.57` — `range{ call:TrimSpace call:ToLower }`
- `4.57` — `if(bin:==(sel,lit:INT))`
- `3.88` — `seq[ assign:=(call:make) ; range ]`

**Structural overlap:** `0.48` (merge-worthy)

- share 7 callees: [http.HandlerFunc, len, make, next.ServeHTTP, strings.ToLower, strings.TrimSpace, w.WriteHeader]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions

---

## Match #8 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `_examples/rest/main.go:415` | `main.ErrInvalidRequest` | ` ` | validation |
| **B** | `_examples/rest/main.go:424` | `main.ErrRender` | ` ` | — |

**Profile A:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `61.27` (shape 61.27, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `4.17` — `return(unary)`

**Structural overlap:** `0.55` (merge-worthy)

- share 1 callees: [err.Error]
- overlapping call-graph neighborhoods (0.25): 2 shared
- both are utility functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [main]

---

## Match #9 — Code-shape: `0.6600`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `_examples/rest/main.go:155` | `main.CreateArticle` | `—` | validation |
| **B** | `_examples/rest/main.go:186` | `main.UpdateArticle` | `—` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.43  flow 1.00  nesting 1.00  sig 1.00  size 0.85`

**Evidence:** `138.02` (shape 119.87, concept 1.35, call 16.80)

**Trophic:** `0.73`

**Shared structure:**

- `6.64` — `do(call:Render)`
- `4.57` — `assign:=(call:Bind)`
- `4.17` — `seq[ assign:=(unary) ; if(bin:!=(id,nil)) ]`

**Structural overlap:** `0.69` (merge-worthy)

- share 4 callees: [ErrInvalidRequest, NewArticleResponse, render.Bind, render.Render]
- overlapping call-graph neighborhoods (0.33): 7 shared
- share patterns: [validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [validation]
- same visibility
- same receiver type: plain functions
- call into same packages: [main]

---

## Match #10 — Code-shape: `0.6800`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `_examples/rest/main.go:186` | `main.UpdateArticle` | `—` | validation |
| **B** | `_examples/rest/main.go:201` | `main.DeleteArticle` | `—` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.47  flow 1.00  nesting 1.00  sig 1.00  size 0.78`

**Evidence:** `121.14` (shape 105.06, concept 1.35, call 14.72)

**Trophic:** `0.72`

**Shared structure:**

- `6.64` — `do(call:Render)`
- `3.48` — `seq[ do(call:Render) ; return() ]`
- `1.94` — `if(bin:!=(id,nil))`

**Structural overlap:** `0.75` (merge-worthy)

- share 5 callees: [ErrInvalidRequest, NewArticleResponse, Value, r.Context, render.Render]
- overlapping call-graph neighborhoods (0.95): 20 shared
- share patterns: [validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [validation]
- same visibility
- same receiver type: plain functions
- call into same packages: [chi, main]

---


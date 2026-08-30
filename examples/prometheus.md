# prometheus

monitoring system; storage engine, query language, and scrape pipeline in one tree

**What this rung shows:** deep call graphs and role classification on a genuinely layered corpus

| | |
|---|---|
| Corpus | [prometheus](https://github.com/prometheus/prometheus) |
| Pinned at | `v3.14.0` (`d7598b7141418fa35be2b5ec5d0fefb634199610`) |
| Project since | 2012 |
| doppel | `8a7ede0` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 12 concepts modeled, 649 associations, 29 unusual realizations
Habitats: 90 modeled, 195 misfits (145 excused by subsystem), 16 subsystems; most uniform tracing (norm 0.97), most diverse testhelpers (norm 0.55)
Conventions: strongest error_wrapping (0.63), loosest retry (0.34)
Ecosystems: 2400 profiled (1707 dominance, 693 coalition, 0 conflict, 0 weak)
Found 5469 functions. Retrieving candidates...
Retrieval: shape 5416, concept 3669, call 8141 -> 15527 unique pairs
  concept-only 22.1%  call-only 42.3%  suppressed-shape functions: 12  large identity buckets: 0  surviving labels: 29106
Running structural comparison on 15527 pairs...
Families: 262 over 457 components, 739 functions in a family, 1907 edges completed
```

# Code Similarity Report

**Functions analyzed:** 5469 | **Threshold:** 0.38 | **Pairs found:** 10

---

## What doppel sees

**5469 functions** across **107 packages** — test functions excluded. Structural roles: 3303 leaf, 1126 orchestrator, 301 passthrough, 739 utility.

### Concepts

doppel reads intent from the AST into a fixed vocabulary and reasons over the tree, so two functions that share a *branch* score partial credit rather than nothing. Leaf counts below are this corpus.

```mermaid
flowchart LR
    c0(["concept"])
    c1(["io_operation"])
    c2(["remote_io"])
    c3["http_call<br/>19"]
    c4["grpc_call<br/>absent"]
    c5(["data_store_access"])
    c6["db_access<br/>88"]
    c7["caching<br/>139"]
    c8["transaction<br/>37"]
    c9["file_io<br/>128"]
    c10["logging<br/>250"]
    c11(["data_transformation"])
    c12["mapping<br/>57"]
    c13["validation<br/>154"]
    c14["serialization<br/>76"]
    c15(["control_flow"])
    c16["concurrency<br/>436"]
    c17(["fault_tolerance"])
    c18["retry<br/>33"]
    c19["circuit_breaker<br/>absent"]
    c20(["error_handling"])
    c21["error_wrapping<br/>352"]
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
    class c4,c19 hot
```

**Nothing here is tagged** `circuit_breaker`, `grpc_call`. That is a direct answer to "does this codebase already do X" — for those concepts, it does not.

| Concept | Functions | Convention |
|---|---:|---|
| `concurrency` | 436 | `0.48` (loose) |
| `error_wrapping` | 352 | `0.63` (settled) |
| `logging` | 250 | `0.51` (settled) |
| `validation` | 154 | `0.60` (settled) |
| `caching` | 139 | `0.48` (loose) |
| `file_io` | 128 | `0.50` (settled) |
| `db_access` | 88 | `0.40` (loose) |
| `serialization` | 76 | `0.46` (loose) |
| `mapping` | 57 | `0.50` (settled) |
| `transaction` | 37 | `0.45` (loose) |
| `retry` | 33 | `0.34` (loose) |
| `http_call` | 19 | `0.35` (loose) |

Convention is how uniformly this corpus realizes a concept: `1.00` means every function carrying the tag does it the same way, and a low number means the tag covers several unrelated habits. A concept with fewer than five members is not modeled.

### Where the duplication is

Merge-worthy pairs folded up to their packages. An edge means two packages keep solving the same problem separately; a count on a node means the repetition is inside one package.

```mermaid
flowchart LR
    p0["rules<br/>30 internal"]
    p1["scrape<br/>35 internal"]
    p0 ---|"34"| p1
    p2["agent<br/>19 internal"]
    p3["tsdb<br/>198 internal"]
    p2 ---|"22"| p3
    p4["aws<br/>130 internal"]
    p5["linode"]
    p4 ---|"10"| p5
    p6["moby<br/>8 internal"]
    p4 ---|"10"| p6
    p0 ---|"10"| p3
    p7["remote<br/>44 internal"]
    p7 ---|"9"| p0
    p8["azure<br/>5 internal"]
    p4 ---|"8"| p8
    p9["digitalocean<br/>1 internal"]
    p4 ---|"8"| p9
    p10["chunks<br/>11 internal"]
    p10 ---|"8"| p3
    p11["openstack<br/>3 internal"]
    p4 ---|"7"| p11
    p12["vultr"]
    p4 ---|"7"| p12
    p13["discovery<br/>4 internal"]
    p13 ---|"7"| p0
```

_264 further package pairs are connected by merge-worthy duplication and are not drawn._

### How settled each package is

A package with at least five functions gets a habitat model: doppel learns what is normal there and measures how surprising each member is against it. **Norm** is how uniform the package's practice is. A **misfit** is a function alien to its package *and* to the wider subsystem around it — one that fits its neighbours a directory up is normal for this codebase and is not reported.

```mermaid
flowchart TD
    h0["testhelpers<br/>109 functions · norm 0.55<br/>7 misfits"]
    h1["storage<br/>355 functions · norm 0.59<br/>147 misfits"]
    h2["uyuni<br/>17 functions · norm 0.69"]
    h3["features<br/>9 functions · norm 0.70"]
    h4["testutil<br/>22 functions · norm 0.74"]
    h5["chunkenc<br/>254 functions · norm 0.74"]
    h6["textparse<br/>92 functions · norm 0.74<br/>6 misfits"]
    h7["ovhcloud<br/>23 functions · norm 0.75"]
    h8["notifications<br/>6 functions · norm 0.76"]
    h9["compression<br/>23 functions · norm 0.77"]
    h10["gce<br/>10 functions · norm 0.77"]
    h11["vultr<br/>11 functions · norm 0.78"]
    classDef good fill:#d7ecd9,color:#1b3d20
    classDef warn fill:#fbeecb,color:#4a3a12
    classDef hot fill:#f7d6d6,color:#4a1c1c
    class h8,h9,h10,h11 good
    class h0,h1,h2,h3,h4,h5,h6,h7 warn
```

_78 further packages are modeled and not drawn._ Most uniform is `tracing` (norm `0.97`); most varied is `testhelpers` (norm `0.55`). 195 functions are alien to their package and to the subsystem around it. A further 145 fit poorly in their package but match the wider subsystem, so they are not reported.

### How these candidates were found

Three channels propose candidates independently — shared rare *structure*, shared *concepts*, shared *calls* — and their union is what gets compared. This run: **15527 candidate pairs** (shape 5416, concept 3669, call 8141), of which 42% arrived on call evidence alone and 22% on concept evidence alone. A pair sharing none of the three is never compared, however alike it looks.

Each function is also an arena where its candidate concepts compete for its evidence. 2400 functions reached an equilibrium: **1707** settled on a single concept, **693** on a coalition, **0** hold concepts this corpus says do not go together.

### Corpus metrics

**Compression ratio:** `9.07`x — this corpus's canonical function bodies contain **468473 AST nodes** in total, which hash-cons (two nodes count as the same subtree exactly when their kind and every child match, all the way down) to **51658 distinct subtree shapes**; the ratio is nodes divided by shapes, always >= 1.0, and it never feeds any score.

**Nearest-neighbour code-shape:** of **5469 functions**, **3780** had a code-shape neighbour among the pairs retrieval actually scored — their best score's p50/p90/p99 are `0.52` / `1.00` / `1.00`, and 79% of them (2985 of 3780) already clear this run's threshold of `0.38`. This is **not an exhaustive nearest-neighbour search** (that would be a full pairwise comparison); it is bounded by the same three retrieval channels the pair list itself is bounded by, so the other 1689 functions are excluded here as having no *scored* neighbour, not asserted to have none at all.

---

## Local practice

The vocabulary above says what a concept *is*. This says what one looks like when **this** codebase writes it — learned from the corpus, so it describes the house style rather than a rule from anywhere else.

### How this codebase writes each concept

Only what is **distinctive**. A feature earns a row by being carried by this concept's members at least twice as often as by the corpus at large — nearly every Go function has a `return` and an `if`, so prevalence alone would describe the language rather than this codebase. Weights are how much a channel counts toward whether a member looks normal — calls 40, control flow 20, co-occurring tags 15, role 15, package 10.

**`concurrency`** — 436 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `tsdbutil.*DirLocker.Lock` | `██████····` | 279 of 436 | 12× |
| flow ×20 | `defer` | `██████····` | 262 of 436 | 7.2× |
|  | `range` | `████······` | 165 of 436 | 2.0× |
| role ×15 | `orchestrator` | `█████·····` | 218 of 436 | 2.4× |

**`error_wrapping`** — 352 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `fmt.Errorf` | `██████████` | 352 of 352 | 9.5× |
| flow ×20 | `funclit` | `███·······` | 103 of 352 | 3.2× |
|  | `range` | `█████·····` | 160 of 352 | 2.4× |
| role ×15 | `orchestrator` | `████······` | 157 of 352 | 2.2× |

**`logging`** — 250 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `fmt.Errorf` | `███·······` | 79 of 250 | 3.0× |
| flow ×20 | `defer` | `████······` | 99 of 250 | 4.8× |
|  | `funclit` | `████······` | 95 of 250 | 4.2× |
|  | `range` | `█████·····` | 123 of 250 | 2.6× |
| cotags ×15 | `concurrency` | `████······` | 98 of 250 | 4.9× |
|  | `error_wrapping` | `███·······` | 69 of 250 | 4.3× |
| role ×15 | `orchestrator` | `██████····` | 141 of 250 | 2.7× |

**`validation`** — 154 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `errors.New` | `███·······` | 51 of 154 | 7.7× |
|  | `fmt.Errorf` | `████······` | 69 of 154 | 4.3× |
|  | `tsdb.*memChunk.len` | `████······` | 65 of 154 | 2.1× |
| flow ×20 | `if` | `██████████` | 150 of 154 | 2.0× |

**`caching`** — 139 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| flow ×20 | `funclit` | `████······` | 49 of 139 | 3.9× |
| cotags ×15 | `concurrency` | `███·······` | 39 of 139 | 3.5× |
| package ×10 | `kubernetes` | `███·······` | 42 of 139 | 14× |

**`file_io`** — 128 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `path/filepath.Join` | `███·······` | 39 of 128 | 29× |
|  | `fmt.Errorf` | `██████····` | 73 of 128 | 5.4× |
| flow ×20 | `defer` | `████······` | 46 of 128 | 4.3× |
|  | `funclit` | `████······` | 45 of 128 | 3.9× |
|  | `range` | `████······` | 57 of 128 | 2.4× |
|  | `if` | `██████████` | 125 of 128 | 2.0× |
| cotags ×15 | `error_wrapping` | `████······` | 54 of 128 | 6.6× |

_6 further concepts are modeled and not described._

### Which concepts share a function

`++` at least four times chance, `+` at least twice, `−` at most half, `never` not once. A blank cell is ordinary company — near chance, which is not culture.

| | `caching` | `concurrency` | `db_access` | `error_wrapping` | `file_io` | `http_call` | `logging` | `mapping` | `retry` | `serialization` | `transaction` |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **`concurrency`** | + | | | | | | | | | | |
| **`db_access`** |  | + | | | | | | | | | |
| **`error_wrapping`** | + |  | ++ | | | | | | | | |
| **`file_io`** |  |  | ++ | ++ | | | | | | | |
| **`http_call`** |  |  |  | ++ | ++ | | | | | | |
| **`logging`** | ++ | ++ | ++ | ++ | ++ | + | | | | | |
| **`mapping`** | ++ |  |  |  |  |  | + | | | | |
| **`retry`** |  | ++ |  |  |  |  | ++ |  | | | |
| **`serialization`** |  |  |  | + | ++ | ++ | + |  |  | | |
| **`transaction`** |  | + | ++ |  |  |  | ++ |  |  |  | |
| **`validation`** |  |  |  | + |  |  |  | ++ |  |  |  |

### What travels with what

Co-occurrence measured against chance across every function. Only relationships at least twice — or at most half — as common as chance are reported; near-chance company is not culture. Each kind is listed separately, because there are far more call tokens than concepts and one shared list is all calls. Within a kind, strongest first means lift weighted by how many functions carry it — a 100× relationship holding for three functions is a weaker finding than a 30× one holding for thirty.

**Together more than chance — tag~tag**

- 13 of 19 `http_call` functions also `file_io` — 29× chance
- 26 of 76 `serialization` functions also `file_io` — 15× chance
- 9 of 19 `http_call` functions also `serialization` — 34× chance
- 54 of 128 `file_io` functions also `error_wrapping` — 6.6× chance
- 98 of 250 `logging` functions also `concurrency` — 4.9× chance
- 69 of 250 `logging` functions also `error_wrapping` — 4.3× chance
- _21 more not listed_

**Together more than chance — tag~role**

- 141 of 250 `logging` functions also `orchestrator` — 2.7× chance
- 218 of 436 `concurrency` functions also `orchestrator` — 2.4× chance
- 157 of 352 `error_wrapping` functions also `orchestrator` — 2.2× chance
- 49 of 352 `error_wrapping` functions also `passthrough` — 2.5× chance
- 31 of 57 `mapping` functions also `orchestrator` — 2.6× chance
- 20 of 33 `retry` functions also `orchestrator` — 2.9× chance
- _3 more not listed_

**Together more than chance — tag~call**

- 13 of 19 `http_call` functions also `net/http.NewRequest` — 288× chance
- 20 of 76 `serialization` functions also `encoding/json.Unmarshal` — 72× chance
- 11 of 19 `http_call` functions also `io.Copy` — 186× chance
- 18 of 76 `serialization` functions also `encoding/json.Marshal` — 72× chance
- 27 of 128 `file_io` functions also `io.ReadAll` — 43× chance
- 26 of 128 `file_io` functions also `os.RemoveAll` — 43× chance
- _596 more not listed_

**Apart more than chance — tag~role**

- **no** `transaction` function has `utility` — chance alone would give about 5 of 37
- 105 of 352 `error_wrapping` functions also `leaf` — 0.5× chance
- 9 of 250 `logging` functions also `utility` — 0.3× chance
- 16 of 57 `mapping` functions also `leaf` — 0.5× chance
- 2 of 88 `db_access` functions also `utility` — 0.2× chance
- 9 of 139 `caching` functions also `utility` — 0.5× chance
- _2 more not listed_

**Apart more than chance — tag~call**

- **no** `concurrency` function has `chunkenc.*bstream.bytes` — chance alone would give about 3 of 436
- **no** `concurrency` function has `math.IsNaN` — chance alone would give about 3 of 436
- **no** `concurrency` function has `v1.stringSchema` — chance alone would give about 3 of 436

### Functions drifting from their own concept

These carry a tag but look nothing like the other functions carrying it. Typicality is measured against the concept's own median, so a genuinely varied concept lowers its own bar and a tight one can flag nobody.

| Function | Concept | Typicality | Concept median |
|---|---|---:|---:|
| `testutil.temporaryDirectory.Close` <br/>`util/testutil/directory.go:86` | `retry` | `0.10` | `0.32` |
| `promql.*evaluator.eval` <br/>`promql/engine.go:2051` | `error_wrapping` | `0.13` | `0.35` |
| `promql.*evaluator.eval` <br/>`promql/engine.go:2051` | `validation` | `0.14` | `0.35` |
| `wlog.*Watcher.Start` <br/>`tsdb/wlog/watcher.go:257` | `concurrency` | `0.11` | `0.31` |
| `scrape.*MetadataMetricsCollector.Describe` <br/>`scrape/metrics.go:353` | `concurrency` | `0.11` | `0.31` |
| `main.main` <br/>`cmd/prometheus/main.go:368` | `validation` | `0.16` | `0.35` |
| `main.*testGroup.test` <br/>`cmd/promtool/unittest.go:228` | `validation` | `0.16` | `0.35` |
| `remote.*MetadataWatcher.Start` <br/>`storage/remote/metadata_watcher.go:89` | `concurrency` | `0.12` | `0.31` |
| `web.*Handler.federation` <br/>`web/federate.go:55` | `validation` | `0.16` | `0.35` |
| `azure.*Discovery.addToCache` <br/>`discovery/azure/azure.go:827` | `logging` | `0.08` | `0.27` |

_19 more unusual realizations not listed._

---

## Match #1 — Code-shape: `0.7551`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `scrape/scrape.go:1589` | `scrape.*scrapeLoopAppender.append` | `([]byte, string, time.Time) (int, int, int, error)` | validation, mapping, caching |
| **B** | `scrape/scrape_append_v2.go:90` | `scrape.*scrapeLoopAppenderV2.append` | `([]byte, string, time.Time) (int, int, int, error)` | validation, mapping, caching |

**Kind:** diverged copy — `*scrapeLoopAppender.append` and `*scrapeLoopAppenderV2.append` share the stem `scrapeLoopAppender` in package `scrape`

**Explain:** differs by nine extra assign, three extra case, one extra branch, and 15 more kinds

**Profile A:** `mapping` 0.51, `caching` 0.49 (coalition)

**Profile B:** `mapping` 0.51, `caching` 0.49 (coalition)

**Code similarity:** `wl 0.59  flow 1.00  nesting 0.99  sig 1.00  size 0.91`

**Containment:** `0.78`

**Evidence:** `3452.21` (shape 3333.52, concept 8.41, call 110.28)

**Trophic:** `0.83`

**Shared structure:**

- `30.18` — `depth-3 IF` ×4
- `27.41` — `depth-2 IF` ×4
- `26.52` — `depth-3 BLOCK` ×4

**Culture:** A realizes `validation` atypically (typicality 0.17, concept median 0.35, convention 0.60)

**Culture:** B realizes `validation` atypically (typicality 0.17, concept median 0.35, convention 0.60)

**Structural overlap:** `0.74` (merge-worthy)

- share 42 callees: [Debug, Error, Inc, Warn, addDropped, addRef, app.Append, append, errors.Is, fmt.Errorf, get, getDropped, isSeriesPartOfFamily, iterDone, len, lset.Get, lset.Has, lset.Hash, lset.IsEmpty, lset.IsValid, lset.String, make, p.Exemplar, p.Help, p.Histogram, p.Labels, p.Next, p.Series, p.StartTimestamp, p.Type, p.Unit, setHelp, setType, setUnit, sl.checkAddError, sl.sampleMutator, slices.SortFunc, string, textparse.New, timestamp.FromTime, trackStaleness, verifyLabelLimits]
- overlapping call-graph neighborhoods (1.00): 1148 shared
- share patterns: [caching, mapping, validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [mapping, caching, validation, concurrency]
- same visibility
- both are methods, on *scrapeLoopAppender and *scrapeLoopAppenderV2
- call into same packages: [discovery, labels, scrape, textparse, timestamp, tsdb]

---

## Match #2 — Code-shape: `0.7474`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `discovery/kubernetes/endpoints.go:63` | `kubernetes.NewEndpoints` | `(*slog.Logger, cache.SharedIndexInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, bool, bool, bool, *prometheus.CounterVec) (*Endpoints)` | mapping, caching, logging |
| **B** | `discovery/kubernetes/endpointslice.go:62` | `kubernetes.NewEndpointSlice` | `(*slog.Logger, cache.SharedIndexInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, bool, bool, bool, *prometheus.CounterVec) (*EndpointSlice)` | mapping, caching, logging |

**Explain:** differs by five extra if, four extra assign, one extra range, and 17 more kinds

**Profile A:** `caching` 1.00 (dominance)

**Profile B:** `caching` 1.00 (dominance)

**Code similarity:** `wl 0.66  flow 0.99  nesting 1.00  sig 0.71  size 0.84`

**Containment:** `0.86`

**Evidence:** `2058.21` (shape 2022.76, concept 7.93, call 27.51)

**Trophic:** `0.88`

**Shared structure:**

- `48.37` — `depth-2 KV` ×10
- `48.37` — `depth-1 KV` ×10
- `38.68` — `depth-3 ASSIGN` ×6

**Structural overlap:** `0.93` (merge-worthy)

- share 20 callees: [AddEventHandler, Error, RoleService.String, convertToService, e.enqueue, e.enqueueNamespace, e.enqueueNode, eps.GetStore, eventCount.WithLabelValues, l.Error, namespacedName, nodeName, pod.GetStore, promslog.NewNopLogger, serviceUpdate, svc.GetStore, svcAddCount.Inc, svcDeleteCount.Inc, svcUpdateCount.Inc, workqueue.NewTypedWithConfig]
- share 1 callers: [kubernetes.*Discovery.Run]
- overlapping call-graph neighborhoods (0.99): 141 shared
- share patterns: [caching, logging, mapping]
- both are orchestrator functions
- same package
- callers do related work (1.00): [caching, logging, concurrency]
- callees do related work (0.75): [mapping, caching]
- same visibility
- same receiver type: plain functions
- called from same packages: [kubernetes]
- call into same packages: [discovery, kubernetes]

---

## Match #3 — Code-shape: `0.9172`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tsdb/head_append.go:1477` | `tsdb.*headAppenderBase.commitHistograms` | `(*appendBatch, *appenderCommitContext)` | concurrency |
| **B** | `tsdb/head_append.go:1579` | `tsdb.*headAppenderBase.commitFloatHistograms` | `(*appendBatch, *appenderCommitContext)` | concurrency |

**Explain:** differs by four extra call

**Profile A:** `concurrency` 1.00 (dominance)

**Profile B:** `concurrency` 1.00 (dominance)

**Code similarity:** `wl 0.86  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `0.93`

**Evidence:** `1231.53` (shape 1175.32, concept 1.41, call 54.81)

**Trophic:** `0.98`

**Shared structure:**

- `12.58` — `depth-3 IF` ×2
- `12.08` — `depth-2 IF` ×2
- `11.87` — `depth-3 IF` ×2

**Structural overlap:** `0.98` (merge-worthy)

- share 15 callees: [Load, acc.collectOOORecords, append, handleAppendableError, len, make, onChunkCreated, series.Lock, series.Unlock, series.cleanupAppendIDsBelow, series.insert, series.sampleState, updateNativeHistogramMetricsOnAppend, updateStaleSeriesMetricOnAppend, value.IsStaleNaN]
- share 1 callers: [tsdb.*headAppenderBase.Commit]
- overlapping call-graph neighborhoods (1.00): 1327 shared
- share patterns: [concurrency]
- both are orchestrator functions
- same package
- callers do related work (1.00): [transaction, logging, error_wrapping]
- callees do related work (1.00): [logging, error_wrapping, concurrency]
- same visibility
- same receiver type: headAppenderBase
- called from same packages: [tsdb]
- call into same packages: [rules, tsdb, tsdbutil, value]

---

## Match #4 — Code-shape: `0.6052`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tsdb/head_wal.go:81` | `tsdb.*Head.loadWAL` | `(*wlog.Reader, *labels.SymbolTable, map[chunks.HeadSeriesRef]chunks.HeadSeriesRef, map[chunks.HeadSeriesRef][]*mmappedChunk, map[chunks.HeadSeriesRef][]*mmappedChunk) (error)` | concurrency, error_wrapping, logging |
| **B** | `tsdb/head_wal.go:871` | `tsdb.*Head.loadWBL` | `(*wlog.Reader, *labels.SymbolTable, map[chunks.HeadSeriesRef]chunks.HeadSeriesRef, chunks.ChunkDiskMapperRef) (error)` | concurrency, error_wrapping, logging |

**Explain:** differs by 33 extra assign, 22 extra if, nine extra range, and 31 more kinds

**Profile A:** `logging` 1.00 (dominance)

**Profile B:** `logging` 1.00 (dominance)

**Code similarity:** `wl 0.43  flow 0.99  nesting 0.99  sig 0.67  size 0.54`

**Containment:** `0.84` — most of the smaller body's shape is inside the larger

**Evidence:** `4711.34` (shape 4663.14, concept 4.99, call 43.21)

**Trophic:** `0.69`

**Shared structure:**

- `55.43` — `depth-3 CALL` ×9
- `55.43` — `depth-2 CALL` ×9
- `39.77` — `depth-2 RANGE` ×6

**Structural overlap:** `0.68` (merge-worthy)

- share 38 callees: [Get, Put, Warn, append, clear, close, closeAndDrain, counterAddNonZero, dec.FloatHistogramSamples, dec.HistogramSamples, dec.Samples, dec.Type, float64, fmt.Errorf, getByID, len, make, min, panic, r.Err, r.Next, r.Offset, r.Record, r.Segment, record.NewDecoder, reuseBuf, reuseHistogramBuf, setup, uint64, unknownHistogramRefs.Add, unknownHistogramRefs.Load, unknownSampleRefs.Add, unknownSampleRefs.Load, unknownSeriesRefs.count, unknownSeriesRefs.merge, wg.Add, wg.Done, wg.Wait]
- overlapping call-graph neighborhoods (0.98): 1180 shared
- share patterns: [concurrency, error_wrapping, logging]
- both are orchestrator functions
- same package
- callees do related work (0.37): [concurrency]
- same visibility
- same receiver type: Head
- call into same packages: [discovery, record, rules, tsdb, wlog]

---

## Match #5 — Code-shape: `0.9000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `storage/remote/queue_manager.go:844` | `remote.*QueueManager.AppendHistograms` | `([]record.RefHistogramSample) (bool)` | retry, concurrency, logging |
| **B** | `storage/remote/queue_manager.go:906` | `remote.*QueueManager.AppendFloatHistograms` | `([]record.RefFloatHistogramSample) (bool)` | retry, concurrency, logging |

**Explain:** identical after rename, negation-flip, commutative-reorder

**Profile A:** `retry` 1.00 (dominance)

**Profile B:** `retry` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 0.33  size 1.00`

**Containment:** `1.00`

**Evidence:** `1031.28` (shape 998.52, concept 7.33, call 25.43)

**Trophic:** `1.00`

**Shared structure:**

- `22.40` — `depth-3 CALL` ×4
- `21.39` — `depth-3 CALL` ×4
- `21.39` — `depth-3 SEL` ×4

**Structural overlap:** `1.00` (merge-worthy)

- share 13 callees: [Inc, Info, Lock, Unlock, Warn, WithLabelValues, enqueue, incr, isSampleOld, model.Duration, time.Duration, time.Now, time.Sleep]
- share 1 callers: [wlog.*Watcher.readSegment]
- overlapping call-graph neighborhoods (1.00): 380 shared
- share patterns: [concurrency, logging, retry]
- both are orchestrator functions
- same package
- callers do related work (1.00): [logging]
- callees do related work (1.00): [logging, error_wrapping, concurrency]
- same visibility
- same receiver type: QueueManager
- called from same packages: [wlog]
- call into same packages: [discovery, remote, tsdbutil]

---

## Match #6 — Code-shape: `0.9246`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `discovery/aws/msk.go:226` | `aws.*MSKDiscovery.initMskClient` | `(context.Context) (error)` | caching, error_wrapping, logging |
| **B** | `discovery/aws/rds.go:339` | `aws.*RDSDiscovery.initRdsClient` | `(context.Context) (error)` | caching, error_wrapping, logging |

**Explain:** differs by four extra call

**Profile A:** `caching` 1.00 (dominance)

**Profile B:** `caching` 1.00 (dominance)

**Code similarity:** `wl 0.87  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `0.93`

**Evidence:** `1207.06` (shape 1115.84, concept 6.12, call 85.10)

**Trophic:** `0.98`

**Shared structure:**

- `32.43` — `depth-3 BIN` ×6
- `32.43` — `depth-2 BIN` ×6
- `14.72` — `depth-2 IF` ×3

**Structural overlap:** `0.79` (merge-worthy)

- share 18 callees: [Error, append, aws.NewCredentialsCache, aws.String, awsConfig.LoadDefaultConfig, awsConfig.WithCredentialsProvider, awsConfig.WithHTTPClient, awsConfig.WithRegion, awsConfig.WithSharedConfigProfile, cancel, config.NewClientFromConfig, context.WithTimeout, credentials.NewStaticCredentialsProvider, fmt.Errorf, loadRegion, string, sts.NewFromConfig, stscreds.NewAssumeRoleProvider]
- overlapping call-graph neighborhoods (0.57): 8 shared
- share patterns: [caching, error_wrapping, logging]
- both are orchestrator functions
- same package
- callers do related work (0.46): [concurrency]
- callees do related work (1.00): [error_wrapping]
- same visibility
- both are methods, on *MSKDiscovery and *RDSDiscovery
- called from same packages: [aws]
- call into same packages: [aws]

---

## Match #7 — Code-shape: `0.9000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `util/strutil/jarowinkler.go:57` | `strutil.jaroWinklerString` | `(string, string) (float64)` | — |
| **B** | `util/strutil/jarowinkler.go:125` | `strutil.jaroWinklerRunes` | `([]rune, []rune) (float64)` | — |

**Explain:** identical after rename, commutative-reorder

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 0.33  size 1.00`

**Containment:** `1.00`

**Evidence:** `1270.23` (shape 1270.23, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `14.23` — `depth-1 ASSIGN` ×3
- `13.70` — `depth-3 UNARY` ×2
- `13.70` — `depth-3 BIN` ×2

**Structural overlap:** `0.72` (merge-worthy)

- share 5 callees: [float64, len, make, max, min]
- share 1 callers: [strutil.*JaroWinklerMatcher.Score]
- overlapping call-graph neighborhoods (1.00): 1040 shared
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [strutil]
- call into same packages: [tsdb]

---

## Match #8 — Code-shape: `0.9248`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `discovery/aws/elasticache.go:315` | `aws.*ElasticacheDiscovery.initElasticacheClient` | `(context.Context) (error)` | caching, error_wrapping, logging |
| **B** | `discovery/aws/rds.go:339` | `aws.*RDSDiscovery.initRdsClient` | `(context.Context) (error)` | caching, error_wrapping, logging |

**Explain:** differs by four extra call

**Profile A:** `caching` 1.00 (dominance)

**Profile B:** `caching` 1.00 (dominance)

**Code similarity:** `wl 0.87  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `0.93`

**Evidence:** `1207.06` (shape 1115.84, concept 6.12, call 85.10)

**Trophic:** `0.98`

**Shared structure:**

- `32.43` — `depth-3 BIN` ×6
- `32.43` — `depth-2 BIN` ×6
- `14.72` — `depth-2 IF` ×3

**Structural overlap:** `0.76` (merge-worthy)

- share 18 callees: [Error, append, aws.NewCredentialsCache, aws.String, awsConfig.LoadDefaultConfig, awsConfig.WithCredentialsProvider, awsConfig.WithHTTPClient, awsConfig.WithRegion, awsConfig.WithSharedConfigProfile, cancel, config.NewClientFromConfig, context.WithTimeout, credentials.NewStaticCredentialsProvider, fmt.Errorf, loadRegion, string, sts.NewFromConfig, stscreds.NewAssumeRoleProvider]
- overlapping call-graph neighborhoods (0.40): 6 shared
- share patterns: [caching, error_wrapping, logging]
- both are orchestrator functions
- same package
- callers do related work (0.54): [error_wrapping, concurrency]
- callees do related work (0.39): [error_wrapping]
- same visibility
- both are methods, on *ElasticacheDiscovery and *RDSDiscovery
- called from same packages: [aws]
- call into same packages: [aws]

---

## Match #9 — Code-shape: `0.9243`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `discovery/aws/elasticache.go:315` | `aws.*ElasticacheDiscovery.initElasticacheClient` | `(context.Context) (error)` | caching, error_wrapping, logging |
| **B** | `discovery/aws/msk.go:226` | `aws.*MSKDiscovery.initMskClient` | `(context.Context) (error)` | caching, error_wrapping, logging |

**Explain:** differs by four extra call

**Profile A:** `caching` 1.00 (dominance)

**Profile B:** `caching` 1.00 (dominance)

**Code similarity:** `wl 0.87  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `0.93`

**Evidence:** `1207.06` (shape 1115.84, concept 6.12, call 85.10)

**Trophic:** `0.98`

**Shared structure:**

- `32.43` — `depth-3 BIN` ×6
- `32.43` — `depth-2 BIN` ×6
- `14.72` — `depth-2 IF` ×3

**Structural overlap:** `0.74` (merge-worthy)

- share 18 callees: [Error, append, aws.NewCredentialsCache, aws.String, awsConfig.LoadDefaultConfig, awsConfig.WithCredentialsProvider, awsConfig.WithHTTPClient, awsConfig.WithRegion, awsConfig.WithSharedConfigProfile, cancel, config.NewClientFromConfig, context.WithTimeout, credentials.NewStaticCredentialsProvider, fmt.Errorf, loadRegion, string, sts.NewFromConfig, stscreds.NewAssumeRoleProvider]
- overlapping call-graph neighborhoods (0.40): 6 shared
- share patterns: [caching, error_wrapping, logging]
- both are orchestrator functions
- same package
- callers do related work (0.25): [concurrency]
- callees do related work (0.39): [error_wrapping]
- same visibility
- both are methods, on *ElasticacheDiscovery and *MSKDiscovery
- called from same packages: [aws]
- call into same packages: [aws]

---

## Match #10 — Code-shape: `0.8777`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tsdb/chunkenc/float_histogram.go:335` | `chunkenc.expandFloatSpansAndBuckets` | `([]histogram.Span, []histogram.Span, []xorValue, []float64) ([]Insert, []Insert, bool)` | — |
| **B** | `tsdb/chunkenc/histogram.go:371` | `chunkenc.expandIntSpansAndBuckets` | `([]histogram.Span, []histogram.Span, []int64, []int64) ([]Insert, []Insert, bool)` | — |

**Explain:** differs by four extra assign, two extra selector, two extra ident

**Code similarity:** `wl 0.92  flow 1.00  nesting 1.00  sig 0.50  size 0.99`

**Containment:** `0.96`

**Evidence:** `1642.71` (shape 1635.49, concept 0.00, call 7.22)

**Trophic:** `0.96`

**Shared structure:**

- `30.18` — `depth-3 CALL` ×4
- `30.18` — `depth-3 ASSIGN` ×4
- `30.18` — `depth-2 ASSIGN` ×4

**Structural overlap:** `0.59` (merge-worthy)

- share 7 callees: [addInsert, advanceA, advanceB, ai.Next, append, bi.Next, newBucketIterator]
- overlapping call-graph neighborhoods (0.78): 7 shared
- both are utility functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [chunkenc]
- call into same packages: [chunkenc]

---

## Families

262 families, 739 functions in a family, largest 34 members; 1907 edges scored here that retrieval never proposed

### Family 1 — 6 members, every pair `>= 0.68` code-shape, evidence `16048`

```mermaid
flowchart LR
    m0["aws.*EC2Discovery.ec2Client"]
    m1["aws.*ECSDiscovery.initEcsClient"]
    m2["aws.*ElasticacheDiscovery.initElasticacheClient"]
    m3["aws.*LightsailDiscovery.lightsailClient"]
    m4["aws.*MSKDiscovery.initMskClient"]
    m5["aws.*RDSDiscovery.initRdsClient"]
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

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `discovery/aws/ec2.go:228` | `aws.*EC2Discovery.ec2Client` | `(context.Context) (ec2Client, error)` | caching, error_wrapping |
| `discovery/aws/ecs.go:260` | `aws.*ECSDiscovery.initEcsClient` | `(context.Context) (error)` | caching, error_wrapping, logging |
| `discovery/aws/elasticache.go:315` | `aws.*ElasticacheDiscovery.initElasticacheClient` | `(context.Context) (error)` | caching, error_wrapping, logging |
| `discovery/aws/lightsail.go:175` | `aws.*LightsailDiscovery.lightsailClient` | `(context.Context) (*lightsailClientAdapter, error)` | caching, error_wrapping |
| `discovery/aws/msk.go:226` | `aws.*MSKDiscovery.initMskClient` | `(context.Context) (error)` | caching, error_wrapping, logging |
| `discovery/aws/rds.go:339` | `aws.*RDSDiscovery.initRdsClient` | `(context.Context) (error)` | caching, error_wrapping, logging |

### Family 2 — 4 members, every pair `>= 0.71` code-shape, evidence `11842`

```mermaid
flowchart LR
    m0["chunkenc.*FloatHistogramAppender.AppendFloatHistogram"]
    m1["chunkenc.*FloatHistogramSTAppender.AppendFloatHistogram"]
    m2["chunkenc.*HistogramAppender.AppendHistogram"]
    m3["chunkenc.*HistogramSTAppender.AppendHistogram"]
    m0 --- m1
    m0 --- m2
    m0 --- m3
    m1 --- m2
    m1 --- m3
    m2 --- m3
```

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `tsdb/chunkenc/float_histogram.go:699` | `chunkenc.*FloatHistogramAppender.AppendFloatHistogram` | `(Appender, int64, int64, *histogram.FloatHistogram, bool) (Chunk, bool, Appender, error)` | — |
| `tsdb/chunkenc/float_histogram_st.go:325` | `chunkenc.*FloatHistogramSTAppender.AppendFloatHistogram` | `(Appender, int64, int64, *histogram.FloatHistogram, bool) (Chunk, bool, Appender, error)` | — |
| `tsdb/chunkenc/histogram.go:751` | `chunkenc.*HistogramAppender.AppendHistogram` | `(Appender, int64, int64, *histogram.Histogram, bool) (Chunk, bool, Appender, error)` | — |
| `tsdb/chunkenc/histogram_st.go:312` | `chunkenc.*HistogramSTAppender.AppendHistogram` | `(Appender, int64, int64, *histogram.Histogram, bool) (Chunk, bool, Appender, error)` | — |

### Family 3 — 15 members, every pair `>= 0.60` code-shape, evidence `11349`  (52 edges scored here)

_Not drawn: 15 members is 105 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_schemas.go:181` | `v1.*OpenAPIBuilder.simpleResponseBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:200` | `v1.*OpenAPIBuilder.statusOnlyResponseBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:215` | `v1.*OpenAPIBuilder.stringArrayResponseBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:235` | `v1.*OpenAPIBuilder.labelsArrayResponseBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:255` | `v1.*OpenAPIBuilder.metricMetadataArrayResponseBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:286` | `v1.*OpenAPIBuilder.notificationArrayResponseBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:683` | `v1.*OpenAPIBuilder.formatQueryOutputBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:847` | `v1.*OpenAPIBuilder.metadataOutputBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:949` | `v1.*OpenAPIBuilder.scrapePoolsDiscoverySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:983` | `v1.*OpenAPIBuilder.relabelStepSchema` | `() (*base.SchemaProxy)` | — |

_5 more members not listed._

### Family 4 — 7 members, every pair `>= 0.61` code-shape, evidence `10930`

```mermaid
flowchart LR
    m0["v1.*OpenAPIBuilder.labelsPath"]
    m1["v1.*OpenAPIBuilder.labelValuesPath"]
    m2["v1.*OpenAPIBuilder.searchMetricNamesPath"]
    m3["v1.*OpenAPIBuilder.searchLabelNamesPath"]
    m4["v1.*OpenAPIBuilder.searchLabelValuesPath"]
    m5["v1.*OpenAPIBuilder.seriesPath"]
    m6["v1.*OpenAPIBuilder.adminDeleteSeriesPath"]
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

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_paths.go:152` | `v1.*OpenAPIBuilder.labelsPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:180` | `v1.*OpenAPIBuilder.labelValuesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:214` | `v1.*OpenAPIBuilder.searchMetricNamesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:250` | `v1.*OpenAPIBuilder.searchLabelNamesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:285` | `v1.*OpenAPIBuilder.searchLabelValuesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:321` | `v1.*OpenAPIBuilder.seriesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:579` | `v1.*OpenAPIBuilder.adminDeleteSeriesPath` | `() (*v3.PathItem)` | — |

### Family 5 — 14 members, every pair `>= 0.60` code-shape, evidence `9499`  (46 edges scored here)

_Not drawn: 14 members is 91 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_schemas.go:181` | `v1.*OpenAPIBuilder.simpleResponseBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:215` | `v1.*OpenAPIBuilder.stringArrayResponseBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:235` | `v1.*OpenAPIBuilder.labelsArrayResponseBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:255` | `v1.*OpenAPIBuilder.metricMetadataArrayResponseBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:286` | `v1.*OpenAPIBuilder.notificationArrayResponseBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:847` | `v1.*OpenAPIBuilder.metadataOutputBodySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:949` | `v1.*OpenAPIBuilder.scrapePoolsDiscoverySchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:983` | `v1.*OpenAPIBuilder.relabelStepSchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:998` | `v1.*OpenAPIBuilder.relabelStepsResponseSchema` | `() (*base.SchemaProxy)` | — |
| `web/api/v1/openapi_schemas.go:1037` | `v1.*OpenAPIBuilder.ruleDiscoverySchema` | `() (*base.SchemaProxy)` | — |

_4 more members not listed._

_257 more families not listed._


# prometheus

monitoring system; storage engine, query language, and scrape pipeline in one tree

**What this rung shows:** deep call graphs and role classification on a genuinely layered corpus

| | |
|---|---|
| Corpus | [prometheus](https://github.com/prometheus/prometheus) |
| Pinned at | `v3.14.0` (`d7598b7141418fa35be2b5ec5d0fefb634199610`) |
| Project since | 2012 |
| doppel | `27da9f4` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 12 concepts modeled, 659 associations, 30 unusual realizations
Habitats: 90 modeled, 566 misfits; most uniform tracing (norm 0.97), most diverse testhelpers (norm 0.55)
Conventions: strongest error_wrapping (0.63), loosest retry (0.34)
Ecosystems: 2520 profiled (1738 dominance, 782 coalition, 0 conflict, 0 weak)
Found 6245 functions. Retrieving candidates...
Retrieval: shape 4365, concept 3885, call 8864 -> 15587 unique pairs
  concept-only 23.5%  call-only 47.6%  suppressed-shape functions: 272  large identity buckets: 5  surviving patterns: 15278
Running structural comparison on 15587 pairs...
```

# Code Similarity Report

**Functions analyzed:** 6245 | **Threshold:** 0.60 | **Pairs found:** 10

---

## Match #1 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `prompb/io/prometheus/write/v2/types.pb.go:2522` | `writev2.*Histogram.Unmarshal` | ` ` | validation |
| **B** | `prompb/types.pb.go:2784` | `prompb.*Histogram.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 0.97`

**Evidence:** `3190.07` (shape 3182.87, concept 2.23, call 4.98)

**Trophic:** `0.99`

**Shared structure:**

- `78.63` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`
- `78.63` — `seq[ if(bin:>=(id,id)) ; assign:=(index) ]`
- `78.63` — `seq[ if(bin:>=(id,lit:INT)) ; if(bin:>=(id,id)) ]`

**Habitat:** A fits poorly in `writev2` (fit 0.00, package norm 0.59)

**Habitat:** B fits poorly in `prompb` (fit 0.00, package norm 0.61)

**Structural overlap:** `0.71` (merge-worthy)

- share 16 callees: [Histogram_ResetHint, Uint64, Unmarshal, append, float64, fmt.Errorf, int, int32, int64, len, make, math.Float64frombits, skipTypes, uint, uint32, uint64]
- overlapping call-graph neighborhoods (1.00): 1168 shared
- share patterns: [validation]
- both are orchestrator functions
- callees do related work (1.00): [validation]
- same visibility
- same receiver type: Histogram
- call into same packages: [tsdb]

---

## Match #2 — Code-shape: `0.9151`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `model/textparse/openmetricslex.l.go:25` | `textparse.*openMetricsLexer.Lex` | ` ` | — |
| **B** | `model/textparse/promlex.l.go:39` | `textparse.*promlexer.Lex` | ` ` | — |

**Code similarity:** `ast 0.86  flow 1.00  sig 1.00  size 0.69`

**Evidence:** `7537.35` (shape 7537.35, concept 0.00, call 0.00)

**Trophic:** `0.78`

**Shared structure:**

- `306.00` — `assign=(call:next)`
- `77.16` — `if(false)`
- `73.69` — `seq[ assign=(id) ; return(id) ]`

**Structural overlap:** `0.37` (not merge-worthy)

- share 4 callees: [fmt.Errorf, l.next, len, panic]
- overlapping call-graph neighborhoods (1.00): 1168 shared
- same package
- same visibility
- both are methods, on *openMetricsLexer and *promlexer
- call into same packages: [tsdb]

---

## Match #3 — Code-shape: `0.9580`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `prompb/io/prometheus/client/metrics.pb.go:2792` | `io_prometheus_client.*Histogram.Unmarshal` | ` ` | validation |
| **B** | `prompb/types.pb.go:2784` | `prompb.*Histogram.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.93  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `2799.49` (shape 2792.29, concept 2.23, call 4.98)

**Trophic:** `0.91`

**Shared structure:**

- `78.63` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`
- `78.63` — `seq[ if(bin:>=(id,id)) ; assign:=(index) ]`
- `78.63` — `seq[ if(bin:>=(id,lit:INT)) ; if(bin:>=(id,id)) ]`

**Habitat:** A fits poorly in `io_prometheus_client` (fit 0.03, package norm 0.75)

**Habitat:** B fits poorly in `prompb` (fit 0.00, package norm 0.61)

**Structural overlap:** `0.68` (merge-worthy)

- share 14 callees: [Uint64, Unmarshal, append, float64, fmt.Errorf, int, int32, int64, len, make, math.Float64frombits, uint, uint32, uint64]
- overlapping call-graph neighborhoods (1.00): 1168 shared
- share patterns: [validation]
- both are orchestrator functions
- callees do related work (1.00): [validation]
- same visibility
- same receiver type: Histogram
- call into same packages: [tsdb]

---

## Match #4 — Code-shape: `0.9580`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `prompb/io/prometheus/client/metrics.pb.go:2792` | `io_prometheus_client.*Histogram.Unmarshal` | ` ` | validation |
| **B** | `prompb/io/prometheus/write/v2/types.pb.go:2522` | `writev2.*Histogram.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.93  flow 1.00  sig 1.00  size 0.98`

**Evidence:** `2799.49` (shape 2792.29, concept 2.23, call 4.98)

**Trophic:** `0.90`

**Shared structure:**

- `78.63` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`
- `78.63` — `seq[ if(bin:>=(id,id)) ; assign:=(index) ]`
- `78.63` — `seq[ if(bin:>=(id,lit:INT)) ; if(bin:>=(id,id)) ]`

**Habitat:** A fits poorly in `io_prometheus_client` (fit 0.03, package norm 0.75)

**Habitat:** B fits poorly in `writev2` (fit 0.00, package norm 0.59)

**Structural overlap:** `0.68` (merge-worthy)

- share 14 callees: [Uint64, Unmarshal, append, float64, fmt.Errorf, int, int32, int64, len, make, math.Float64frombits, uint, uint32, uint64]
- overlapping call-graph neighborhoods (1.00): 1168 shared
- share patterns: [validation]
- both are orchestrator functions
- callees do related work (1.00): [validation]
- same visibility
- same receiver type: Histogram
- call into same packages: [tsdb]

---

## Match #5 — Code-shape: `0.9961`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `prompb/io/prometheus/write/v2/types.pb.go:1257` | `writev2.*Histogram.MarshalToSizedBuffer` | ` ` | — |
| **B** | `prompb/types.pb.go:1401` | `prompb.*Histogram.MarshalToSizedBuffer` | ` ` | — |

**Code similarity:** `ast 0.99  flow 1.00  sig 1.00  size 0.96`

**Evidence:** `1751.24` (shape 1746.24, concept 0.00, call 5.00)

**Trophic:** `0.99`

**Shared structure:**

- `51.91` — `assign=(call:encodeVarintTypes)`
- `27.60` — `seq[ assign-=(lit:INT) ; do(call:PutUint64) ]`
- `27.23` — `assign-=(id)`

**Habitat:** A fits poorly in `writev2` (fit 0.01, package norm 0.59)

**Habitat:** B fits poorly in `prompb` (fit 0.00, package norm 0.61)

**Structural overlap:** `0.48` (merge-worthy)

- share 13 callees: [MarshalTo, MarshalToSizedBuffer, PutUint64, Size, copy, encodeVarintTypes, float64, len, make, math.Float64bits, uint32, uint64, uint8]
- overlapping call-graph neighborhoods (1.00): 1168 shared
- both are orchestrator functions
- same visibility
- same receiver type: Histogram
- call into same packages: [tsdb]

---

## Match #6 — Code-shape: `0.8257`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `scrape/scrape.go:1589` | `scrape.*scrapeLoopAppender.append` | ` ` | validation, mapping, caching |
| **B** | `scrape/scrape_append_v2.go:90` | `scrape.*scrapeLoopAppenderV2.append` | ` ` | validation, mapping, caching |

**Profile A:** `mapping` 0.52, `caching` 0.48 (coalition)

**Profile B:** `mapping` 0.52, `caching` 0.48 (coalition)

**Code similarity:** `ast 0.71  flow 1.00  sig 1.00  size 0.91`

**Evidence:** `1793.62` (shape 1673.12, concept 8.24, call 112.27)

**Trophic:** `0.83`

**Shared structure:**

- `21.07` — `seq[ if(bin:>(sel,lit:INT)) ; if(bin:>(sel,lit:INT)) ]`
- `20.45` — `if(bin:>(sel,lit:INT))`
- `20.40` — `return(id,id,id,id)`

**Structural overlap:** `0.74` (merge-worthy)

- share 42 callees: [Debug, Error, Inc, Warn, addDropped, addRef, app.Append, append, errors.Is, fmt.Errorf, get, getDropped, isSeriesPartOfFamily, iterDone, len, lset.Get, lset.Has, lset.Hash, lset.IsEmpty, lset.IsValid, lset.String, make, p.Exemplar, p.Help, p.Histogram, p.Labels, p.Next, p.Series, p.StartTimestamp, p.Type, p.Unit, setHelp, setType, setUnit, sl.checkAddError, sl.sampleMutator, slices.SortFunc, string, textparse.New, timestamp.FromTime, trackStaleness, verifyLabelLimits]
- overlapping call-graph neighborhoods (1.00): 1278 shared
- share patterns: [caching, mapping, validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [mapping, caching, validation, concurrency]
- same visibility
- both are methods, on *scrapeLoopAppender and *scrapeLoopAppenderV2
- call into same packages: [discovery, labels, scrape, textparse, timestamp, tsdb]

---

## Match #7 — Code-shape: `0.8218`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `discovery/kubernetes/endpoints.go:63` | `kubernetes.NewEndpoints` | ` ` | mapping, caching, logging |
| **B** | `discovery/kubernetes/endpointslice.go:62` | `kubernetes.NewEndpointSlice` | ` ` | mapping, caching, logging |

**Profile A:** `caching` 1.00 (dominance)

**Profile B:** `caching` 1.00 (dominance)

**Code similarity:** `ast 0.78  flow 0.99  sig 0.71  size 0.84`

**Evidence:** `1166.27` (shape 1130.23, concept 7.99, call 28.04)

**Trophic:** `0.90`

**Shared structure:**

- `39.71` — `assign:=(call:WithLabelValues)`
- `33.09` — `seq[ assign:=(call:WithLabelValues) ; assign:=(call:WithLabelValues) ]`
- `23.15` — `seq[ do(call:Inc) ; do(call:serviceUpdate) ]`

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

## Match #8 — Code-shape: `0.7659`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `discovery/kubernetes/endpoints.go:347` | `kubernetes.*Endpoints.buildEndpoints` | ` ` | logging |
| **B** | `discovery/kubernetes/endpointslice.go:308` | `kubernetes.*EndpointSlice.buildEndpointSlice` | ` ` | — |

**Profile A:** `logging` 1.00 (dominance)

**Code similarity:** `ast 0.78  flow 0.98  sig 0.33  size 0.84`

**Evidence:** `1720.40` (shape 1657.09, concept 0.00, call 63.31)

**Trophic:** `0.84`

**Shared structure:**

- `59.24` — `assign=(call:lv)`
- `43.86` — `seq[ assign=(call:lv) ; assign=(call:lv) ]`
- `19.85` — `range{ call:add }`

**Structural overlap:** `0.62` (merge-worthy)

- share 19 callees: [add, addNamespaceLabels, addNodeLabels, addObjectMetaLabels, append, e.addServiceLabels, e.resolvePodRef, hasSeenPort, len, lv, model.LabelName, namespacedName, net.JoinHostPort, podLabels, strconv.FormatBool, strconv.FormatUint, string, target.Merge, uint64]
- overlapping call-graph neighborhoods (0.99): 1196 shared
- both are orchestrator functions
- same package
- callers do related work (0.63): [caching, logging, concurrency]
- callees do related work (1.00): [logging]
- same visibility
- both are methods, on *Endpoints and *EndpointSlice
- called from same packages: [kubernetes]
- call into same packages: [kubernetes, tsdb]

---

## Match #9 — Code-shape: `0.9000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `util/strutil/jarowinkler.go:57` | `strutil.jaroWinklerString` | ` ` | — |
| **B** | `util/strutil/jarowinkler.go:125` | `strutil.jaroWinklerRunes` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  sig 0.33  size 1.00`

**Evidence:** `839.26` (shape 839.26, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `11.40` — `assign:=(call:max)`
- `10.83` — `assign=(id,id)`
- `10.23` — `seq[ assign:=(bin) ; assign:=(bin) ]`

**Structural overlap:** `0.72` (merge-worthy)

- share 5 callees: [float64, len, make, max, min]
- share 1 callers: [strutil.*JaroWinklerMatcher.Score]
- overlapping call-graph neighborhoods (1.00): 1170 shared
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [strutil]
- call into same packages: [tsdb]

---

## Match #10 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `util/runtime/statfs_linux_386.go:24` | `runtime.FsType` | ` ` | — |
| **B** | `util/runtime/statfs_uint32.go:23` | `runtime.FsType` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `1012.76` (shape 1000.90, concept 0.00, call 11.86)

**Trophic:** `1.00`

**Shared structure:**

- `15.43` — `return(call:Itoa)`
- `7.72` — `seq[ if(id) ; return(call:Itoa) ]`
- `6.80` — `seq[ assign:=(call:Statfs) ; if(bin:!=(id,nil)) ]`

**Structural overlap:** `0.52` (merge-worthy)

- share 3 callees: [int, strconv.Itoa, syscall.Statfs]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions

---


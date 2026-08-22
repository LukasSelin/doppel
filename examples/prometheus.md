# prometheus

monitoring system; storage engine, query language, and scrape pipeline in one tree

**What this rung shows:** deep call graphs and role classification on a genuinely layered corpus

| | |
|---|---|
| Corpus | [prometheus](https://github.com/prometheus/prometheus) |
| Pinned at | `v3.14.0` (`d7598b7141418fa35be2b5ec5d0fefb634199610`) |
| Project since | 2012 |
| doppel | `b730816` |
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
Retrieval: shape 3319, concept 3669, call 8141 -> 13847 unique pairs
  concept-only 25.0%  call-only 50.1%  suppressed-shape functions: 67  large identity buckets: 0  surviving patterns: 37732
Running structural comparison on 13847 pairs...
Families: 477 over 611 components, 1366 functions in a family, 4359 edges completed
```

# Code Similarity Report

**Functions analyzed:** 5469 | **Threshold:** 0.60 | **Pairs found:** 10

---

## Match #1 — Code-shape: `0.8253`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `scrape/scrape.go:1589` | `scrape.*scrapeLoopAppender.append` | `([]byte, string, time.Time) (int, int, int, error)` | validation, mapping, caching |
| **B** | `scrape/scrape_append_v2.go:90` | `scrape.*scrapeLoopAppenderV2.append` | `([]byte, string, time.Time) (int, int, int, error)` | validation, mapping, caching |

**Kind:** diverged copy — `*scrapeLoopAppender.append` and `*scrapeLoopAppenderV2.append` share the stem `scrapeLoopAppender` in package `scrape`

**Profile A:** `mapping` 0.51, `caching` 0.49 (coalition)

**Profile B:** `mapping` 0.51, `caching` 0.49 (coalition)

**Code similarity:** `ast 0.71  flow 1.00  nesting 0.99  sig 1.00  size 0.91`

**Evidence:** `4720.28` (shape 4601.59, concept 8.41, call 110.28)

**Trophic:** `0.81`

**Shared structure:**

- `43.32` — `flow:call:get→cond`
- `38.06` — `flow:call:Histogram→cond`
- `20.76` — `seq[ if(bin:>(sel,lit:INT)) ; if(bin:>(sel,lit:INT)) ]`

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

## Match #2 — Code-shape: `0.8224`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `discovery/kubernetes/endpoints.go:63` | `kubernetes.NewEndpoints` | `(*slog.Logger, cache.SharedIndexInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, bool, bool, bool, *prometheus.CounterVec) (*Endpoints)` | mapping, caching, logging |
| **B** | `discovery/kubernetes/endpointslice.go:62` | `kubernetes.NewEndpointSlice` | `(*slog.Logger, cache.SharedIndexInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, cache.SharedInformer, bool, bool, bool, *prometheus.CounterVec) (*EndpointSlice)` | mapping, caching, logging |

**Profile A:** `caching` 1.00 (dominance)

**Profile B:** `caching` 1.00 (dominance)

**Code similarity:** `ast 0.78  flow 0.99  nesting 1.00  sig 0.71  size 0.84`

**Evidence:** `3018.39` (shape 2982.95, concept 7.93, call 27.51)

**Trophic:** `0.90`

**Shared structure:**

- `45.60` — `flow:call:AddEventHandler→call:Error`
- `45.60` — `flow:call:AddEventHandler→cond`
- `39.08` — `assign:=(call:WithLabelValues)`

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

## Match #3 — Code-shape: `0.9000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `util/strutil/jarowinkler.go:57` | `strutil.jaroWinklerString` | `(string, string) (float64)` | — |
| **B** | `util/strutil/jarowinkler.go:125` | `strutil.jaroWinklerRunes` | `([]rune, []rune) (float64)` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 0.33  size 1.00`

**Evidence:** `2024.14` (shape 2024.14, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `22.84` — `flow:call:len→call:min`
- `11.20` — `assign:=(call:max)`
- `11.07` — `flow:call:len→call:float64`

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

## Match #4 — Code-shape: `0.7407`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tsdb/head_wal.go:81` | `tsdb.*Head.loadWAL` | `(*wlog.Reader, *labels.SymbolTable, map[chunks.HeadSeriesRef]chunks.HeadSeriesRef, map[chunks.HeadSeriesRef][]*mmappedChunk, map[chunks.HeadSeriesRef][]*mmappedChunk) (error)` | concurrency, error_wrapping, logging |
| **B** | `tsdb/head_wal.go:871` | `tsdb.*Head.loadWBL` | `(*wlog.Reader, *labels.SymbolTable, map[chunks.HeadSeriesRef]chunks.HeadSeriesRef, chunks.ChunkDiskMapperRef) (error)` | concurrency, error_wrapping, logging |

**Profile A:** `logging` 1.00 (dominance)

**Profile B:** `logging` 1.00 (dominance)

**Code similarity:** `ast 0.66  flow 0.99  nesting 0.99  sig 0.67  size 0.54`

**Evidence:** `6107.70` (shape 6059.51, concept 4.99, call 43.21)

**Trophic:** `0.65`

**Shared structure:**

- `40.18` — `flow:call:Get→call:len`
- `30.45` — `seq[ assign=(unary) ; return() ]`
- `30.45` — `do(call:counterAddNonZero)`

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

## Match #5 — Code-shape: `0.9090`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `prompb/io/prometheus/client/decoder.go:314` | `io_prometheus_client.*Metric.unmarshalWithoutLabels` | `(*MetricStreamingDecoder, []byte) (error)` | validation |
| **B** | `prompb/io/prometheus/client/decoder.go:596` | `io_prometheus_client.*MetricFamily.unmarshalWithoutMetrics` | `(*MetricStreamingDecoder, []byte) (error)` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.85  flow 1.00  nesting 1.00  sig 1.00  size 0.69`

**Evidence:** `4344.70` (shape 4342.26, concept 2.44, call 0.00)

**Trophic:** `0.75`

**Shared structure:**

- `64.29` — `if(bin:<(id,lit:INT))`
- `47.22` — `flow:call:int→cond`
- `43.24` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`

**Structural overlap:** `0.57` (merge-worthy)

- share 9 callees: [append, errors.New, fmt.Errorf, int, int32, len, skipMetrics, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 1039 shared
- share patterns: [validation]
- same package
- same visibility
- both are methods, on *Metric and *MetricFamily
- called from same packages: [io_prometheus_client]
- call into same packages: [tsdb]

---

## Match #6 — Code-shape: `0.7631`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `discovery/kubernetes/endpoints.go:347` | `kubernetes.*Endpoints.buildEndpoints` | `(*apiv1.Endpoints) (*targetgroup.Group)` | logging |
| **B** | `discovery/kubernetes/endpointslice.go:308` | `kubernetes.*EndpointSlice.buildEndpointSlice` | `(v1.EndpointSlice) (*targetgroup.Group)` | — |

**Profile A:** `logging` 1.00 (dominance)

**Code similarity:** `ast 0.78  flow 0.98  nesting 0.92  sig 0.33  size 0.84`

**Evidence:** `3719.38` (shape 3657.39, concept 0.00, call 61.98)

**Trophic:** `0.81`

**Shared structure:**

- `58.21` — `assign=(call:lv)`
- `43.24` — `seq[ assign=(call:lv) ; assign=(call:lv) ]`
- `19.54` — `range{ call:add }`

**Structural overlap:** `0.62` (merge-worthy)

- share 19 callees: [add, addNamespaceLabels, addNodeLabels, addObjectMetaLabels, append, e.addServiceLabels, e.resolvePodRef, hasSeenPort, len, lv, model.LabelName, namespacedName, net.JoinHostPort, podLabels, strconv.FormatBool, strconv.FormatUint, string, target.Merge, uint64]
- overlapping call-graph neighborhoods (0.99): 1066 shared
- both are orchestrator functions
- same package
- callers do related work (0.63): [caching, logging, concurrency]
- callees do related work (1.00): [logging]
- same visibility
- both are methods, on *Endpoints and *EndpointSlice
- called from same packages: [kubernetes]
- call into same packages: [kubernetes, tsdb]

---

## Match #7 — Code-shape: `0.9459`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `model/histogram/float_histogram.go:1420` | `histogram.addBuckets` | `(int32, float64, bool, []Span, []float64, []Span, []float64) ([]Span, []float64)` | — |
| **B** | `model/histogram/float_histogram.go:1545` | `histogram.kahanAddBuckets` | `(int32, float64, bool, []Span, []float64, []Span, []float64, []float64, []float64) ([]Span, []float64, []float64)` | — |

**Code similarity:** `ast 0.91  flow 1.00  nesting 0.99  sig 1.00  size 0.78`

**Evidence:** `3138.58` (shape 3124.84, concept 0.00, call 13.75)

**Trophic:** `0.86`

**Shared structure:**

- `20.76` — `seq[ assign=(call:append) ; do(call:copy) ]`
- `11.48` — `assign-=(bin)`
- `11.48` — `if(bin:<(id,call:len))`

**Structural overlap:** `0.50` (merge-worthy)

- share 7 callees: [IsExponentialSchema, append, copy, getBoundExponential, int, int32, len]
- overlapping call-graph neighborhoods (0.99): 1044 shared
- related roles: passthrough ≈ orchestrator (both high fan-out, 0.50)
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [histogram]
- call into same packages: [histogram, tsdb]

---

## Match #8 — Code-shape: `0.9000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `storage/remote/queue_manager.go:844` | `remote.*QueueManager.AppendHistograms` | `([]record.RefHistogramSample) (bool)` | retry, concurrency, logging |
| **B** | `storage/remote/queue_manager.go:906` | `remote.*QueueManager.AppendFloatHistograms` | `([]record.RefFloatHistogramSample) (bool)` | retry, concurrency, logging |

**Profile A:** `retry` 1.00 (dominance)

**Profile B:** `retry` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 0.33  size 1.00`

**Evidence:** `1211.21` (shape 1178.45, concept 7.33, call 25.43)

**Trophic:** `1.00`

**Shared structure:**

- `7.61` — `range{ call:isSampleOld call:Duration call:Inc call:WithLabelValues call:Warn call:Lock call:incr call:Info ... }`
- `7.61` — `seq[ if(call:isSampleOld) ; if(bin:&&(bin,bin)) ]`
- `7.21` — `seq[ do(call:Unlock) ; assign:=(call:Duration) ]`

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

## Match #9 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `util/runtime/statfs_linux_386.go:24` | `runtime.FsType` | `(string) (string)` | — |
| **B** | `util/runtime/statfs_uint32.go:23` | `runtime.FsType` | `(string) (string)` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `2054.36` (shape 2042.77, concept 0.00, call 11.59)

**Trophic:** `1.00`

**Shared structure:**

- `15.23` — `return(call:Itoa)`
- `7.61` — `seq[ if(id) ; return(call:Itoa) ]`
- `6.70` — `seq[ assign:=(call:Statfs) ; if(bin:!=(id,nil)) ]`

**Structural overlap:** `0.52` (merge-worthy)

- share 3 callees: [int, strconv.Itoa, syscall.Statfs]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions

---

## Match #10 — Code-shape: `0.8934`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tsdb/chunkenc/float_histogram.go:335` | `chunkenc.expandFloatSpansAndBuckets` | `([]histogram.Span, []histogram.Span, []xorValue, []float64) ([]Insert, []Insert, bool)` | — |
| **B** | `tsdb/chunkenc/histogram.go:371` | `chunkenc.expandIntSpansAndBuckets` | `([]histogram.Span, []histogram.Span, []int64, []int64) ([]Insert, []Insert, bool)` | — |

**Code similarity:** `ast 0.95  flow 1.00  nesting 1.00  sig 0.50  size 0.99`

**Evidence:** `2179.52` (shape 2172.30, concept 0.00, call 7.22)

**Trophic:** `0.96`

**Shared structure:**

- `30.45` — `assign=(call:addInsert)`
- `30.45` — `flow:call:Next→call:addInsert`
- `30.45` — `flow:call:append→call:addInsert`

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

477 families, 1366 functions in a family, largest 15 members; 4359 edges scored here that retrieval never proposed

### Family 1 — 15 members, every pair `>= 0.60` code-shape, evidence `46062`  (32 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_paths.go:28` | `v1.*OpenAPIBuilder.queryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:55` | `v1.*OpenAPIBuilder.queryRangePath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:84` | `v1.*OpenAPIBuilder.queryExemplarsPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:108` | `v1.*OpenAPIBuilder.formatQueryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:130` | `v1.*OpenAPIBuilder.parseQueryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:152` | `v1.*OpenAPIBuilder.labelsPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:180` | `v1.*OpenAPIBuilder.labelValuesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:214` | `v1.*OpenAPIBuilder.searchMetricNamesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:250` | `v1.*OpenAPIBuilder.searchLabelNamesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:285` | `v1.*OpenAPIBuilder.searchLabelValuesPath` | `() (*v3.PathItem)` | — |

_5 more members not listed._

### Family 2 — 17 members, every pair `>= 0.60` code-shape, evidence `44210`  (53 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_paths.go:28` | `v1.*OpenAPIBuilder.queryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:55` | `v1.*OpenAPIBuilder.queryRangePath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:84` | `v1.*OpenAPIBuilder.queryExemplarsPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:108` | `v1.*OpenAPIBuilder.formatQueryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:130` | `v1.*OpenAPIBuilder.parseQueryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:152` | `v1.*OpenAPIBuilder.labelsPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:180` | `v1.*OpenAPIBuilder.labelValuesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:250` | `v1.*OpenAPIBuilder.searchLabelNamesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:285` | `v1.*OpenAPIBuilder.searchLabelValuesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:321` | `v1.*OpenAPIBuilder.seriesPath` | `() (*v3.PathItem)` | — |

_7 more members not listed._

### Family 3 — 17 members, every pair `>= 0.60` code-shape, evidence `39215`  (56 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_paths.go:28` | `v1.*OpenAPIBuilder.queryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:55` | `v1.*OpenAPIBuilder.queryRangePath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:84` | `v1.*OpenAPIBuilder.queryExemplarsPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:108` | `v1.*OpenAPIBuilder.formatQueryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:130` | `v1.*OpenAPIBuilder.parseQueryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:152` | `v1.*OpenAPIBuilder.labelsPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:180` | `v1.*OpenAPIBuilder.labelValuesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:285` | `v1.*OpenAPIBuilder.searchLabelValuesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:321` | `v1.*OpenAPIBuilder.seriesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:349` | `v1.*OpenAPIBuilder.metadataPath` | `() (*v3.PathItem)` | — |

_7 more members not listed._

### Family 4 — 17 members, every pair `>= 0.65` code-shape, evidence `35639`  (57 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_paths.go:28` | `v1.*OpenAPIBuilder.queryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:55` | `v1.*OpenAPIBuilder.queryRangePath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:84` | `v1.*OpenAPIBuilder.queryExemplarsPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:108` | `v1.*OpenAPIBuilder.formatQueryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:130` | `v1.*OpenAPIBuilder.parseQueryPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:152` | `v1.*OpenAPIBuilder.labelsPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:180` | `v1.*OpenAPIBuilder.labelValuesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:321` | `v1.*OpenAPIBuilder.seriesPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:349` | `v1.*OpenAPIBuilder.metadataPath` | `() (*v3.PathItem)` | — |
| `web/api/v1/openapi_paths.go:377` | `v1.*OpenAPIBuilder.targetsPath` | `() (*v3.PathItem)` | — |

_7 more members not listed._

### Family 5 — 43 members, every pair `>= 0.61` code-shape, evidence `33714`  (731 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_examples.go:28` | `v1.queryPostExamples` | `() (*orderedmap.Map[string, *base.Example])` | — |
| `web/api/v1/openapi_examples.go:57` | `v1.queryRangePostExamples` | `() (*orderedmap.Map[string, *base.Example])` | — |
| `web/api/v1/openapi_examples.go:85` | `v1.queryExemplarsPostExamples` | `() (*orderedmap.Map[string, *base.Example])` | — |
| `web/api/v1/openapi_examples.go:106` | `v1.formatQueryPostExamples` | `() (*orderedmap.Map[string, *base.Example])` | — |
| `web/api/v1/openapi_examples.go:123` | `v1.parseQueryPostExamples` | `() (*orderedmap.Map[string, *base.Example])` | — |
| `web/api/v1/openapi_examples.go:140` | `v1.labelsPostExamples` | `() (*orderedmap.Map[string, *base.Example])` | — |
| `web/api/v1/openapi_examples.go:167` | `v1.searchMetricNamesPostExamples` | `() (*orderedmap.Map[string, *base.Example])` | — |
| `web/api/v1/openapi_examples.go:184` | `v1.searchLabelNamesPostExamples` | `() (*orderedmap.Map[string, *base.Example])` | — |
| `web/api/v1/openapi_examples.go:201` | `v1.searchLabelValuesPostExamples` | `() (*orderedmap.Map[string, *base.Example])` | — |
| `web/api/v1/openapi_examples.go:219` | `v1.seriesPostExamples` | `() (*orderedmap.Map[string, *base.Example])` | — |

_33 more members not listed._

_472 more families not listed._


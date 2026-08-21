# prometheus

monitoring system; storage engine, query language, and scrape pipeline in one tree

**What this rung shows:** deep call graphs and role classification on a genuinely layered corpus

| | |
|---|---|
| Corpus | [prometheus](https://github.com/prometheus/prometheus) |
| Pinned at | `v3.14.0` (`d7598b7141418fa35be2b5ec5d0fefb634199610`) |
| Project since | 2012 |
| doppel | `b6eeaeb` |
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
Retrieval: shape 4512, concept 3885, call 8864 -> 15776 unique pairs
  concept-only 23.3%  call-only 47.2%  suppressed-shape functions: 199  large identity buckets: 5  surviving patterns: 38503
Running structural comparison on 15776 pairs...
Families: 509 over 630 components, 1703 functions in a family, 8283 edges completed
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

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 0.97`

**Evidence:** `10150.99` (shape 10143.78, concept 2.23, call 4.98)

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

## Match #2 — Code-shape: `0.9571`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `prompb/io/prometheus/client/metrics.pb.go:2792` | `io_prometheus_client.*Histogram.Unmarshal` | ` ` | validation |
| **B** | `prompb/types.pb.go:2784` | `prompb.*Histogram.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.93  flow 1.00  nesting 0.98  sig 1.00  size 1.00`

**Evidence:** `9006.13` (shape 8998.92, concept 2.23, call 4.98)

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

## Match #3 — Code-shape: `0.9573`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `prompb/io/prometheus/client/metrics.pb.go:2792` | `io_prometheus_client.*Histogram.Unmarshal` | ` ` | validation |
| **B** | `prompb/io/prometheus/write/v2/types.pb.go:2522` | `writev2.*Histogram.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.93  flow 1.00  nesting 0.99  sig 1.00  size 0.98`

**Evidence:** `9019.98` (shape 9012.78, concept 2.23, call 4.98)

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

## Match #4 — Code-shape: `0.9153`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `model/textparse/openmetricslex.l.go:25` | `textparse.*openMetricsLexer.Lex` | ` ` | — |
| **B** | `model/textparse/promlex.l.go:39` | `textparse.*promlexer.Lex` | ` ` | — |

**Code similarity:** `ast 0.86  flow 1.00  nesting 1.00  sig 1.00  size 0.69`

**Evidence:** `15458.42` (shape 15458.42, concept 0.00, call 0.00)

**Trophic:** `0.77`

**Shared structure:**

- `306.00` — `assign=(call:next)`
- `102.00` — `flow:call:next→cond`
- `77.16` — `if(false)`

**Structural overlap:** `0.37` (not merge-worthy)

- share 4 callees: [fmt.Errorf, l.next, len, panic]
- overlapping call-graph neighborhoods (1.00): 1168 shared
- same package
- same visibility
- both are methods, on *openMetricsLexer and *promlexer
- call into same packages: [tsdb]

---

## Match #5 — Code-shape: `0.9961`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `prompb/io/prometheus/write/v2/types.pb.go:1257` | `writev2.*Histogram.MarshalToSizedBuffer` | ` ` | — |
| **B** | `prompb/types.pb.go:1401` | `prompb.*Histogram.MarshalToSizedBuffer` | ` ` | — |

**Code similarity:** `ast 0.99  flow 1.00  nesting 1.00  sig 1.00  size 0.96`

**Evidence:** `4805.14` (shape 4800.14, concept 0.00, call 5.00)

**Trophic:** `0.99`

**Shared structure:**

- `51.91` — `assign=(call:encodeVarintTypes)`
- `51.91` — `flow:call:len→call:encodeVarintTypes`
- `51.91` — `flow:param→call:encodeVarintTypes`

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

## Match #6 — Code-shape: `0.8980`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `prompb/io/prometheus/client/decoder.go:314` | `io_prometheus_client.*Metric.unmarshalWithoutLabels` | ` ` | validation |
| **B** | `prompb/io/prometheus/client/metrics.pb.go:3733` | `io_prometheus_client.*Metric.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.91  flow 1.00  nesting 1.00  sig 0.67  size 0.98`

**Evidence:** `3149.66` (shape 3141.40, concept 2.23, call 6.03)

**Trophic:** `0.98`

**Shared structure:**

- `37.00` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`
- `37.00` — `seq[ if(bin:>=(id,id)) ; assign:=(index) ]`
- `37.00` — `seq[ if(bin:>=(id,lit:INT)) ; if(bin:>=(id,id)) ]`

**Habitat:** A fits poorly in `io_prometheus_client` (fit 0.03, package norm 0.75)

**Habitat:** B fits poorly in `io_prometheus_client` (fit 0.04, package norm 0.75)

**Structural overlap:** `0.74` (merge-worthy)

- share 10 callees: [Unmarshal, append, fmt.Errorf, int, int32, int64, len, skipMetrics, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 1168 shared
- share patterns: [validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [validation]
- same receiver type: Metric
- call into same packages: [io_prometheus_client, tsdb]

---

## Match #7 — Code-shape: `0.8253`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `scrape/scrape.go:1589` | `scrape.*scrapeLoopAppender.append` | ` ` | validation, mapping, caching |
| **B** | `scrape/scrape_append_v2.go:90` | `scrape.*scrapeLoopAppenderV2.append` | ` ` | validation, mapping, caching |

**Profile A:** `mapping` 0.52, `caching` 0.48 (coalition)

**Profile B:** `mapping` 0.52, `caching` 0.48 (coalition)

**Code similarity:** `ast 0.71  flow 1.00  nesting 0.99  sig 1.00  size 0.91`

**Evidence:** `4730.38` (shape 4609.87, concept 8.24, call 112.27)

**Trophic:** `0.81`

**Shared structure:**

- `44.15` — `flow:call:get→cond`
- `38.58` — `flow:call:Histogram→cond`
- `21.07` — `seq[ if(bin:>(sel,lit:INT)) ; if(bin:>(sel,lit:INT)) ]`

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

## Match #8 — Code-shape: `0.8224`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `discovery/kubernetes/endpoints.go:63` | `kubernetes.NewEndpoints` | ` ` | mapping, caching, logging |
| **B** | `discovery/kubernetes/endpointslice.go:62` | `kubernetes.NewEndpointSlice` | ` ` | mapping, caching, logging |

**Profile A:** `caching` 1.00 (dominance)

**Profile B:** `caching` 1.00 (dominance)

**Code similarity:** `ast 0.78  flow 0.99  nesting 1.00  sig 0.71  size 0.84`

**Evidence:** `3059.43` (shape 3023.40, concept 7.99, call 28.04)

**Trophic:** `0.90`

**Shared structure:**

- `46.32` — `flow:call:AddEventHandler→call:Error`
- `46.32` — `flow:call:AddEventHandler→cond`
- `39.71` — `assign:=(call:WithLabelValues)`

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

## Match #9 — Code-shape: `0.9000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `util/strutil/jarowinkler.go:57` | `strutil.jaroWinklerString` | ` ` | — |
| **B** | `util/strutil/jarowinkler.go:125` | `strutil.jaroWinklerRunes` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 0.33  size 1.00`

**Evidence:** `1990.11` (shape 1990.11, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `23.15` — `flow:call:len→call:min`
- `11.40` — `assign:=(call:max)`
- `11.27` — `flow:call:len→call:float64`

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

## Match #10 — Code-shape: `0.7407`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tsdb/head_wal.go:81` | `tsdb.*Head.loadWAL` | ` ` | concurrency, error_wrapping, logging |
| **B** | `tsdb/head_wal.go:871` | `tsdb.*Head.loadWBL` | ` ` | concurrency, error_wrapping, logging |

**Profile A:** `logging` 1.00 (dominance)

**Profile B:** `logging` 1.00 (dominance)

**Code similarity:** `ast 0.66  flow 0.99  nesting 0.99  sig 0.67  size 0.54`

**Evidence:** `6014.62` (shape 5965.56, concept 5.06, call 44.00)

**Trophic:** `0.64`

**Shared structure:**

- `40.80` — `flow:call:Get→call:len`
- `30.86` — `seq[ assign=(unary) ; return() ]`
- `30.86` — `do(call:counterAddNonZero)`

**Structural overlap:** `0.68` (merge-worthy)

- share 38 callees: [Get, Put, Warn, append, clear, close, closeAndDrain, counterAddNonZero, dec.FloatHistogramSamples, dec.HistogramSamples, dec.Samples, dec.Type, float64, fmt.Errorf, getByID, len, make, min, panic, r.Err, r.Next, r.Offset, r.Record, r.Segment, record.NewDecoder, reuseBuf, reuseHistogramBuf, setup, uint64, unknownHistogramRefs.Add, unknownHistogramRefs.Load, unknownSampleRefs.Add, unknownSampleRefs.Load, unknownSeriesRefs.count, unknownSeriesRefs.merge, wg.Add, wg.Done, wg.Wait]
- overlapping call-graph neighborhoods (0.99): 1310 shared
- share patterns: [concurrency, error_wrapping, logging]
- both are orchestrator functions
- same package
- callees do related work (0.39): [concurrency]
- same visibility
- same receiver type: Head
- call into same packages: [discovery, record, rules, tsdb, wlog]

---

## Families

509 families, 1703 functions in a family, largest 55 members; 8283 edges scored here that retrieval never proposed

### Family 1 — 55 members, every pair `>= 0.60` code-shape  (1287 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_helpers.go:172` | `v1.warningsSchema` | ` ` | — |
| `web/api/v1/openapi_helpers.go:180` | `v1.infosSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:138` | `v1.*OpenAPIBuilder.errorSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:181` | `v1.*OpenAPIBuilder.simpleResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:200` | `v1.*OpenAPIBuilder.statusOnlyResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:215` | `v1.*OpenAPIBuilder.stringArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:235` | `v1.*OpenAPIBuilder.labelsArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:286` | `v1.*OpenAPIBuilder.notificationArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:308` | `v1.*OpenAPIBuilder.floatSampleSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:334` | `v1.*OpenAPIBuilder.histogramValueSchema` | ` ` | — |

_45 more members not listed._

### Family 2 — 54 members, every pair `>= 0.61` code-shape  (1230 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_schemas.go:138` | `v1.*OpenAPIBuilder.errorSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:181` | `v1.*OpenAPIBuilder.simpleResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:200` | `v1.*OpenAPIBuilder.statusOnlyResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:215` | `v1.*OpenAPIBuilder.stringArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:235` | `v1.*OpenAPIBuilder.labelsArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:255` | `v1.*OpenAPIBuilder.metricMetadataArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:286` | `v1.*OpenAPIBuilder.notificationArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:308` | `v1.*OpenAPIBuilder.floatSampleSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:334` | `v1.*OpenAPIBuilder.histogramValueSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:361` | `v1.*OpenAPIBuilder.histogramSampleSchema` | ` ` | — |

_44 more members not listed._

### Family 3 — 54 members, every pair `>= 0.60` code-shape  (1246 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_helpers.go:172` | `v1.warningsSchema` | ` ` | — |
| `web/api/v1/openapi_helpers.go:180` | `v1.infosSchema` | ` ` | — |
| `web/api/v1/openapi_helpers.go:188` | `v1.timestampSchema` | ` ` | — |
| `web/api/v1/openapi_helpers.go:206` | `v1.durationSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:138` | `v1.*OpenAPIBuilder.errorSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:181` | `v1.*OpenAPIBuilder.simpleResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:200` | `v1.*OpenAPIBuilder.statusOnlyResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:215` | `v1.*OpenAPIBuilder.stringArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:308` | `v1.*OpenAPIBuilder.floatSampleSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:334` | `v1.*OpenAPIBuilder.histogramValueSchema` | ` ` | — |

_44 more members not listed._

### Family 4 — 53 members, every pair `>= 0.64` code-shape  (1168 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_schemas.go:138` | `v1.*OpenAPIBuilder.errorSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:181` | `v1.*OpenAPIBuilder.simpleResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:200` | `v1.*OpenAPIBuilder.statusOnlyResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:215` | `v1.*OpenAPIBuilder.stringArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:235` | `v1.*OpenAPIBuilder.labelsArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:255` | `v1.*OpenAPIBuilder.metricMetadataArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:286` | `v1.*OpenAPIBuilder.notificationArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:308` | `v1.*OpenAPIBuilder.floatSampleSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:334` | `v1.*OpenAPIBuilder.histogramValueSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:361` | `v1.*OpenAPIBuilder.histogramSampleSchema` | ` ` | — |

_43 more members not listed._

### Family 5 — 53 members, every pair `>= 0.62` code-shape  (1168 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `web/api/v1/openapi_schemas.go:138` | `v1.*OpenAPIBuilder.errorSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:161` | `v1.*OpenAPIBuilder.responseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:181` | `v1.*OpenAPIBuilder.simpleResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:200` | `v1.*OpenAPIBuilder.statusOnlyResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:215` | `v1.*OpenAPIBuilder.stringArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:235` | `v1.*OpenAPIBuilder.labelsArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:255` | `v1.*OpenAPIBuilder.metricMetadataArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:286` | `v1.*OpenAPIBuilder.notificationArrayResponseBodySchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:308` | `v1.*OpenAPIBuilder.floatSampleSchema` | ` ` | — |
| `web/api/v1/openapi_schemas.go:334` | `v1.*OpenAPIBuilder.histogramValueSchema` | ` ` | — |

_43 more members not listed._

_504 more families not listed._


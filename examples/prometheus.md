# prometheus

monitoring system; storage engine, query language, and scrape pipeline in one tree

**What this rung shows:** deep call graphs and role classification on a genuinely layered corpus

| | |
|---|---|
| Corpus | [prometheus](https://github.com/prometheus/prometheus) |
| Pinned at | `v3.14.0` (`d7598b7141418fa35be2b5ec5d0fefb634199610`) |
| Project since | 2012 |
| doppel | `e61ea20` |
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
Retrieval: shape 4499, concept 3885, call 8864 -> 15764 unique pairs
  concept-only 23.3%  call-only 47.3%  suppressed-shape functions: 201  large identity buckets: 5  surviving patterns: 36313
Running structural comparison on 15764 pairs...
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

**Evidence:** `9868.83` (shape 9861.63, concept 2.23, call 4.98)

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

**Evidence:** `8766.68` (shape 8759.48, concept 2.23, call 4.98)

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

**Evidence:** `8780.53` (shape 8773.33, concept 2.23, call 4.98)

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

**Evidence:** `15356.42` (shape 15356.42, concept 0.00, call 0.00)

**Trophic:** `0.77`

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

## Match #5 — Code-shape: `0.9961`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `prompb/io/prometheus/write/v2/types.pb.go:1257` | `writev2.*Histogram.MarshalToSizedBuffer` | ` ` | — |
| **B** | `prompb/types.pb.go:1401` | `prompb.*Histogram.MarshalToSizedBuffer` | ` ` | — |

**Code similarity:** `ast 0.99  flow 1.00  nesting 1.00  sig 1.00  size 0.96`

**Evidence:** `4627.83` (shape 4622.83, concept 0.00, call 5.00)

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

## Match #6 — Code-shape: `0.8980`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `prompb/io/prometheus/client/decoder.go:314` | `io_prometheus_client.*Metric.unmarshalWithoutLabels` | ` ` | validation |
| **B** | `prompb/io/prometheus/client/metrics.pb.go:3733` | `io_prometheus_client.*Metric.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.91  flow 1.00  nesting 1.00  sig 0.67  size 0.98`

**Evidence:** `3015.31` (shape 3007.06, concept 2.23, call 6.03)

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

**Evidence:** `4327.97` (shape 4207.47, concept 8.24, call 112.27)

**Trophic:** `0.80`

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

## Match #8 — Code-shape: `0.8224`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `discovery/kubernetes/endpoints.go:63` | `kubernetes.NewEndpoints` | ` ` | mapping, caching, logging |
| **B** | `discovery/kubernetes/endpointslice.go:62` | `kubernetes.NewEndpointSlice` | ` ` | mapping, caching, logging |

**Profile A:** `caching` 1.00 (dominance)

**Profile B:** `caching` 1.00 (dominance)

**Code similarity:** `ast 0.78  flow 0.99  nesting 1.00  sig 0.71  size 0.84`

**Evidence:** `2787.80` (shape 2751.76, concept 7.99, call 28.04)

**Trophic:** `0.89`

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

## Match #9 — Code-shape: `0.9000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `util/strutil/jarowinkler.go:57` | `strutil.jaroWinklerString` | ` ` | — |
| **B** | `util/strutil/jarowinkler.go:125` | `strutil.jaroWinklerRunes` | ` ` | — |

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 0.33  size 1.00`

**Evidence:** `1938.64` (shape 1938.64, concept 0.00, call 0.00)

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

## Match #10 — Code-shape: `0.7407`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tsdb/head_wal.go:81` | `tsdb.*Head.loadWAL` | ` ` | concurrency, error_wrapping, logging |
| **B** | `tsdb/head_wal.go:871` | `tsdb.*Head.loadWBL` | ` ` | concurrency, error_wrapping, logging |

**Profile A:** `logging` 1.00 (dominance)

**Profile B:** `logging` 1.00 (dominance)

**Code similarity:** `ast 0.66  flow 0.99  nesting 0.99  sig 0.67  size 0.54`

**Evidence:** `5774.61` (shape 5725.55, concept 5.06, call 44.00)

**Trophic:** `0.65`

**Shared structure:**

- `30.86` — `seq[ assign=(unary) ; return() ]`
- `30.86` — `do(call:counterAddNonZero)`
- `23.15` — `seq[ do(call:counterAddNonZero) ; do(call:counterAddNonZero) ]`

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


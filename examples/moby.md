# moby

container engine; a decade of accretion across daemon, API, and plugin layers

**What this rung shows:** scale, df caps, and the common-idiom suppression the retrieval channels exist for

| | |
|---|---|
| Corpus | [moby](https://github.com/moby/moby) |
| Pinned at | `v28.5.2` (`89c5e8fd66634b6128fc4c0e6f1236e2540e46e0`) |
| Project since | 2013 |
| doppel | `fc06f91` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 12 concepts modeled, 790 associations, 113 unusual realizations
Habitats: 166 modeled, 253 misfits; most uniform checker (norm 0.98), most diverse vfs (norm 0.56)
Conventions: strongest error_wrapping (0.62), loosest db_access (0.37)
Ecosystems: 3997 profiled (3689 dominance, 308 coalition, 0 conflict, 0 weak)
Found 7644 functions. Retrieving candidates...
Retrieval: shape 3983, concept 2410, call 12442 -> 17471 unique pairs
  concept-only 13.1%  call-only 63.6%  suppressed-shape functions: 179  large identity buckets: 3  surviving patterns: 46104
Running structural comparison on 17471 pairs...
Families: 656 over 702 components, 1522 functions in a family, 2814 edges completed
  1 pairs suppressed by max-per-func=2
```

# Code Similarity Report

**Functions analyzed:** 7644 | **Threshold:** 0.60 | **Pairs found:** 10

---

## Match #1 — Code-shape: `0.9535`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/networkdb/networkdbdiagnostic.go:128` | `networkdb.*NetworkDB.dbCreateEntry` | `—` | logging |
| **B** | `libnetwork/networkdb/networkdbdiagnostic.go:177` | `networkdb.*NetworkDB.dbUpdateEntry` | `—` | logging |

**Profile A:** `logging` 1.00 (dominance)

**Profile B:** `logging` 1.00 (dominance)

**Code similarity:** `ast 0.92  flow 1.00  nesting 1.00  sig 1.00  size 0.99`

**Evidence:** `1966.98` (shape 1919.55, concept 3.05, call 44.38)

**Trophic:** `0.94`

**Shared structure:**

- `24.45` — `do(call:HTTPReply)`
- `23.88` — `flow:call:ParseHTTPFormOptions→call:HTTPReply`
- `23.62` — `flow:param→call:HTTPReply`

**Structural overlap:** `0.79` (merge-worthy)

- share 19 callees: [DecodeString, Error, String, WithFields, caller.Name, context.TODO, diagnostic.CommandSucceed, diagnostic.DebugHTTPForm, diagnostic.FailCommand, diagnostic.HTTPReply, diagnostic.ParseHTTPFormOptions, diagnostic.WrongCommand, fmt.Sprintf, len, log.G, logger.Error, logger.Info, logger.WithError, r.ParseForm]
- overlapping call-graph neighborhoods (0.91): 29 shared
- share patterns: [logging]
- both are orchestrator functions
- same package
- callees do related work (1.00): [serialization, concurrency]
- same visibility
- same receiver type: NetworkDB
- call into same packages: [caller, diagnostic, networkdb]

---

## Match #2 — Code-shape: `0.9297`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:631` | `dbclient.doWriteDeleteWaitLeaveJoin` | `—` | concurrency |
| **B** | `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:713` | `dbclient.doWriteWaitLeaveJoin` | `—` | concurrency |

**Profile A:** `concurrency` 1.00 (dominance)

**Profile B:** `concurrency` 1.00 (dominance)

**Code similarity:** `ast 0.88  flow 1.00  nesting 1.00  sig 1.00  size 0.92`

**Evidence:** `1633.01` (shape 1581.67, concept 1.20, call 50.14)

**Trophic:** `0.89`

**Shared structure:**

- `25.50` — `flow:call:WithTimeout→call:G`
- `19.79` — `flow:call:Atoi→call:waitWriters`
- `19.79` — `flow:call:make→call:waitWriters`

**Structural overlap:** `0.94` (merge-worthy)

- share 16 callees: [Infof, cancel, checkTable, close, context.Background, context.WithTimeout, fmt.Fprintf, joinNetwork, leaveNetwork, log.G, make, strconv.Atoi, strconv.Itoa, time.Duration, time.Sleep, waitWriters]
- share 1 callers: [dbclient.Client]
- overlapping call-graph neighborhoods (0.97): 76 shared
- share patterns: [concurrency]
- both are orchestrator functions
- same package
- callees do related work (1.00): [concurrency]
- same visibility
- same receiver type: plain functions
- called from same packages: [dbclient]
- call into same packages: [container, dbclient]

---

## Match #3 — Code-shape: `0.9443`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:464` | `dbclient.doWriteKeys` | `—` | concurrency |
| **B** | `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:497` | `dbclient.doDeleteKeys` | `—` | concurrency |

**Profile A:** `concurrency` 1.00 (dominance)

**Profile B:** `concurrency` 1.00 (dominance)

**Code similarity:** `ast 0.91  flow 1.00  nesting 1.00  sig 1.00  size 0.99`

**Evidence:** `1010.80` (shape 978.00, concept 1.20, call 31.60)

**Trophic:** `0.97`

**Shared structure:**

- `13.20` — `flow:call:Atoi→call:waitWriters`
- `13.20` — `flow:call:make→call:waitWriters`
- `9.93` — `flow:call:Atoi→cond`

**Structural overlap:** `0.94` (merge-worthy)

- share 14 callees: [Infof, cancel, checkTable, clientWatchTable, close, context.Background, context.TODO, context.WithTimeout, fmt.Fprintf, log.G, make, strconv.Atoi, strconv.Itoa, waitWriters]
- share 1 callers: [dbclient.Client]
- overlapping call-graph neighborhoods (0.97): 73 shared
- share patterns: [concurrency]
- both are orchestrator functions
- same package
- callees do related work (1.00): [concurrency]
- same visibility
- same receiver type: plain functions
- called from same packages: [dbclient]
- call into same packages: [container, dbclient]

---

## Match #4 — Code-shape: `0.9664`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/networkdb/networkdbdiagnostic.go:308` | `networkdb.*NetworkDB.dbJoinNetwork` | `—` | logging |
| **B** | `libnetwork/networkdb/networkdbdiagnostic.go:340` | `networkdb.*NetworkDB.dbLeaveNetwork` | `—` | logging |

**Profile A:** `logging` 1.00 (dominance)

**Profile B:** `logging` 1.00 (dominance)

**Code similarity:** `ast 0.94  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `1134.27` (shape 1086.84, concept 3.05, call 44.38)

**Trophic:** `0.99`

**Shared structure:**

- `18.34` — `do(call:HTTPReply)`
- `17.91` — `flow:call:ParseHTTPFormOptions→call:HTTPReply`
- `17.71` — `flow:param→call:HTTPReply`

**Structural overlap:** `0.79` (merge-worthy)

- share 18 callees: [Error, String, WithFields, caller.Name, context.TODO, diagnostic.CommandSucceed, diagnostic.DebugHTTPForm, diagnostic.FailCommand, diagnostic.HTTPReply, diagnostic.ParseHTTPFormOptions, diagnostic.WrongCommand, fmt.Sprintf, len, log.G, logger.Error, logger.Info, logger.WithError, r.ParseForm]
- overlapping call-graph neighborhoods (0.84): 27 shared
- share patterns: [logging]
- both are orchestrator functions
- same package
- callees do related work (1.00): [serialization, concurrency]
- same visibility
- same receiver type: NetworkDB
- call into same packages: [caller, diagnostic, networkdb]

---

## Match #5 — Code-shape: `0.9265`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:677` | `dbclient.doWriteWaitLeave` | `—` | concurrency |
| **B** | `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:713` | `dbclient.doWriteWaitLeaveJoin` | `—` | concurrency |

**Profile A:** `concurrency` 1.00 (dominance)

**Profile B:** `concurrency` 1.00 (dominance)

**Code similarity:** `ast 0.88  flow 0.99  nesting 1.00  sig 1.00  size 0.75`

**Evidence:** `1381.36` (shape 1335.18, concept 1.20, call 44.98)

**Trophic:** `0.84`

**Shared structure:**

- `19.12` — `flow:call:WithTimeout→call:G`
- `13.20` — `flow:call:Atoi→call:waitWriters`
- `13.20` — `flow:call:make→call:waitWriters`

**Structural overlap:** `0.92` (merge-worthy)

- share 15 callees: [Infof, cancel, checkTable, close, context.Background, context.WithTimeout, fmt.Fprintf, leaveNetwork, log.G, make, strconv.Atoi, strconv.Itoa, time.Duration, waitWriters, writeUniqueKeys]
- share 1 callers: [dbclient.Client]
- overlapping call-graph neighborhoods (0.99): 76 shared
- share patterns: [concurrency]
- both are orchestrator functions
- same package
- callees do related work (1.00): [concurrency]
- same visibility
- same receiver type: plain functions
- called from same packages: [dbclient]
- call into same packages: [container, dbclient]

---

## Match #6 — Code-shape: `0.9353`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:567` | `dbclient.doWriteUniqueKeys` | `—` | concurrency |
| **B** | `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:677` | `dbclient.doWriteWaitLeave` | `—` | concurrency |

**Profile A:** `concurrency` 1.00 (dominance)

**Profile B:** `concurrency` 1.00 (dominance)

**Code similarity:** `ast 0.89  flow 1.00  nesting 1.00  sig 1.00  size 0.95`

**Evidence:** `1178.57` (shape 1139.71, concept 1.20, call 37.65)

**Trophic:** `0.84`

**Shared structure:**

- `13.20` — `flow:call:Atoi→call:waitWriters`
- `13.20` — `flow:call:make→call:waitWriters`
- `12.75` — `flow:call:WithTimeout→call:G`

**Structural overlap:** `0.94` (merge-worthy)

- share 14 callees: [Infof, cancel, checkTable, close, context.Background, context.WithTimeout, fmt.Fprintf, log.G, make, strconv.Atoi, strconv.Itoa, time.Duration, waitWriters, writeUniqueKeys]
- share 1 callers: [dbclient.Client]
- overlapping call-graph neighborhoods (0.97): 74 shared
- share patterns: [concurrency]
- both are orchestrator functions
- same package
- callees do related work (1.00): [concurrency]
- same visibility
- same receiver type: plain functions
- called from same packages: [dbclient]
- call into same packages: [container, dbclient]

---

## Match #7 — Code-shape: `0.9081`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/drivers/ipvlan/ipvlan_joinleave.go:33` | `ipvlan.*driver.Join` | ` ` | — |
| **B** | `libnetwork/drivers/macvlan/macvlan_joinleave.go:21` | `macvlan.*driver.Join` | ` ` | — |

**Code similarity:** `ast 0.86  flow 1.00  nesting 0.85  sig 1.00  size 0.76`

**Evidence:** `1977.00` (shape 1919.05, concept 0.00, call 57.94)

**Trophic:** `0.88`

**Shared structure:**

- `48.11` — `flow:call:getNetwork→cond`
- `29.52` — `flow:call:getNetwork→return`
- `28.27` — `flow:call:getNetwork→call:Errorf`

**Structural overlap:** `0.51` (merge-worthy)

- share 26 callees: [Debugf, Start, String, attribute.String, d.getNetwork, d.storeUpdate, fmt.Errorf, iNames.SetNames, jinfo.DisableGatewayService, jinfo.InterfaceName, jinfo.SetGateway, jinfo.SetGatewayIPv6, len, log.G, n.endpoint, n.getSubnetforIPv4, n.getSubnetforIPv6, net.ParseCIDR, netlabel.GetIfname, netutils.GenerateIfaceName, ns.NlHandle, otel.Tracer, span.End, trace.WithAttributes, v4gw.String, v6gw.String]
- overlapping call-graph neighborhoods (0.98): 134 shared
- both are orchestrator functions
- callees do related work (1.00): [concurrency]
- same visibility
- same receiver type: driver
- call into same packages: [libnetwork, netlabel, netutils, ns, tailfile]

---

## Match #8 — Code-shape: `0.8508`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/networkdb/networkdbdiagnostic.go:225` | `networkdb.*NetworkDB.dbDeleteEntry` | `—` | logging |
| **B** | `libnetwork/networkdb/networkdbdiagnostic.go:262` | `networkdb.*NetworkDB.dbGetEntry` | `—` | logging |

**Profile A:** `logging` 1.00 (dominance)

**Profile B:** `logging` 1.00 (dominance)

**Code similarity:** `ast 0.76  flow 0.98  nesting 0.98  sig 1.00  size 0.82`

**Evidence:** `1381.44` (shape 1334.02, concept 3.05, call 44.38)

**Trophic:** `0.87`

**Shared structure:**

- `18.34` — `do(call:HTTPReply)`
- `17.91` — `flow:call:ParseHTTPFormOptions→call:HTTPReply`
- `17.71` — `flow:param→call:HTTPReply`

**Structural overlap:** `0.74` (merge-worthy)

- share 18 callees: [Error, String, WithFields, caller.Name, context.TODO, diagnostic.CommandSucceed, diagnostic.DebugHTTPForm, diagnostic.FailCommand, diagnostic.HTTPReply, diagnostic.ParseHTTPFormOptions, diagnostic.WrongCommand, fmt.Sprintf, len, log.G, logger.Error, logger.Info, logger.WithError, r.ParseForm]
- overlapping call-graph neighborhoods (0.78): 25 shared
- share patterns: [logging]
- both are orchestrator functions
- same package
- callees do related work (0.66): [serialization]
- same visibility
- same receiver type: NetworkDB
- call into same packages: [caller, diagnostic, networkdb]

---

## Match #9 — Code-shape: `0.6408`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `daemon/logger/loggertest/logreader.go:73` | `loggertest.Reader.testTail` | `—` | validation, logging |
| **B** | `daemon/logger/loggertest/logreader.go:196` | `loggertest.Reader.TestFollow` | `—` | validation, concurrency, logging |

**Profile A:** `logging` 1.00 (dominance)

**Profile B:** `logging` 1.00 (dominance)

**Code similarity:** `ast 0.56  flow 0.94  nesting 0.86  sig 0.50  size 0.56`

**Evidence:** `4483.96` (shape 4440.46, concept 5.08, call 38.42)

**Trophic:** `0.58`

**Shared structure:**

- `60.63` — `flow:call:ReadLogs→call:readAll`
- `60.63` — `flow:param→call:readAll`
- `60.58` — `do(call:Run)`

**Culture:** B realizes `validation` atypically (typicality 0.13, concept median 0.30, convention 0.59)

**Structural overlap:** `0.66` (merge-worthy)

- share 14 callees: [Add, Truncate, assert.DeepEqual, assert.NilError, context.TODO, factory, l.Close, logMessages, lw.ConsumerGone, makeTestMessages, readAll, t.Parallel, t.Run, tr.Factory]
- overlapping call-graph neighborhoods (0.92): 11 shared
- share patterns: [logging, validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [mapping, logging, validation]
- same receiver type: Reader
- call into same packages: [logger, loggertest]

---

## Match #10 — Code-shape: `0.8631`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `daemon/graphdriver/fuse-overlayfs/fuseoverlayfs.go:172` | `fuseoverlayfs.*Driver.create` | ` ` | file_io |
| **B** | `daemon/graphdriver/overlay2/overlay.go:345` | `overlay2.*Driver.create` | ` ` | file_io |

**Profile A:** `file_io` 1.00 (dominance)

**Profile B:** `file_io` 1.00 (dominance)

**Code similarity:** `ast 0.77  flow 1.00  nesting 0.99  sig 1.00  size 0.84`

**Evidence:** `1441.84` (shape 1399.43, concept 2.18, call 40.23)

**Trophic:** `0.89`

**Shared structure:**

- `60.58` — `flow:call:MkdirAllAndChown→cond`
- `60.58` — `flow:call:MkdirAllAndChown→return`
- `34.43` — `flow:call:RootPair→call:MkdirAndChown`

**Structural overlap:** `0.65` (merge-worthy)

- share 11 callees: [RootPair, d.dir, d.getLower, len, os.RemoveAll, os.Symlink, overlayutils.GenerateID, path.Dir, path.Join, user.MkdirAllAndChown, user.MkdirAndChown]
- overlapping call-graph neighborhoods (0.76): 34 shared
- share patterns: [file_io]
- both are orchestrator functions
- callees do related work (1.00): [retry, logging]
- same visibility
- same receiver type: Driver
- call into same packages: [idtools, overlayutils]

---

## Families

656 families, 1522 functions in a family, largest 44 members; 2814 edges scored here that retrieval never proposed

### Family 1 — 44 members, every pair `>= 0.61` code-shape  (741 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `builder/builder-next/adapters/containerimage/pull.go:874` | `containerimage.*jobs.isResolved` | ` ` | concurrency |
| `cmd/docker-proxy/udp_proxy_linux.go:69` | `main.*connTrackEntry.lastWrite` | ` ` | concurrency |
| `container/state.go:242` | `container.*State.IsRunning` | ` ` | concurrency |
| `container/state.go:249` | `container.*State.GetPID` | ` ` | concurrency |
| `container/state.go:354` | `container.*State.IsPaused` | ` ` | concurrency |
| `container/state.go:373` | `container.*State.IsRestarting` | ` ` | concurrency |
| `container/state.go:400` | `container.*State.IsRemovalInProgress` | ` ` | concurrency |
| `container/state.go:407` | `container.*State.IsDead` | ` ` | concurrency |
| `libnetwork/controller.go:269` | `libnetwork.*Controller.getAgent` | ` ` | concurrency |
| `libnetwork/diagnostic/server.go:125` | `diagnostic.*Server.Enabled` | ` ` | concurrency |

_34 more members not listed._

### Family 2 — 29 members, every pair `>= 0.62` code-shape  (277 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `integration/internal/container/ops.go:23` | `container.WithHostname` | ` ` | — |
| `integration/internal/container/ops.go:30` | `container.WithLinks` | ` ` | — |
| `integration/internal/container/ops.go:37` | `container.WithImage` | ` ` | — |
| `integration/internal/container/ops.go:44` | `container.WithCmd` | ` ` | — |
| `integration/internal/container/ops.go:51` | `container.WithNetworkMode` | ` ` | — |
| `integration/internal/container/ops.go:65` | `container.WithSysctls` | ` ` | — |
| `integration/internal/container/ops.go:92` | `container.WithTty` | ` ` | — |
| `integration/internal/container/ops.go:99` | `container.WithWorkingDir` | ` ` | — |
| `integration/internal/container/ops.go:234` | `container.WithUser` | ` ` | — |
| `integration/internal/container/ops.go:241` | `container.WithAdditionalGroups` | ` ` | — |

_19 more members not listed._

### Family 3 — 29 members, every pair `>= 0.61` code-shape  (277 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `integration/internal/container/ops.go:23` | `container.WithHostname` | ` ` | — |
| `integration/internal/container/ops.go:30` | `container.WithLinks` | ` ` | — |
| `integration/internal/container/ops.go:37` | `container.WithImage` | ` ` | — |
| `integration/internal/container/ops.go:44` | `container.WithCmd` | ` ` | — |
| `integration/internal/container/ops.go:51` | `container.WithNetworkMode` | ` ` | — |
| `integration/internal/container/ops.go:58` | `container.WithDNS` | ` ` | — |
| `integration/internal/container/ops.go:65` | `container.WithSysctls` | ` ` | — |
| `integration/internal/container/ops.go:92` | `container.WithTty` | ` ` | — |
| `integration/internal/container/ops.go:99` | `container.WithWorkingDir` | ` ` | — |
| `integration/internal/container/ops.go:234` | `container.WithUser` | ` ` | — |

_19 more members not listed._

### Family 4 — 17 members, every pair `>= 0.60` code-shape  (84 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `client/checkpoint_list.go:12` | `client.*Client.CheckpointList` | ` ` | serialization |
| `client/config_inspect.go:13` | `client.*Client.ConfigInspectWithRaw` | ` ` | serialization, file_io |
| `client/container_diff.go:12` | `client.*Client.ContainerDiff` | ` ` | serialization |
| `client/container_inspect.go:14` | `client.*Client.ContainerInspect` | ` ` | serialization |
| `client/container_inspect.go:32` | `client.*Client.ContainerInspectWithRaw` | ` ` | serialization, file_io |
| `client/container_top.go:13` | `client.*Client.ContainerTop` | ` ` | serialization |
| `client/container_update.go:11` | `client.*Client.ContainerUpdate` | ` ` | serialization |
| `client/network_inspect.go:20` | `client.*Client.NetworkInspectWithRaw` | ` ` | serialization, file_io |
| `client/node_inspect.go:13` | `client.*Client.NodeInspectWithRaw` | ` ` | serialization, file_io |
| `client/plugin_inspect.go:13` | `client.*Client.PluginInspectWithRaw` | ` ` | serialization, file_io |

_7 more members not listed._

### Family 5 — 17 members, every pair `>= 0.60` code-shape  (88 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `client/checkpoint_list.go:12` | `client.*Client.CheckpointList` | ` ` | serialization |
| `client/config_inspect.go:13` | `client.*Client.ConfigInspectWithRaw` | ` ` | serialization, file_io |
| `client/container_diff.go:12` | `client.*Client.ContainerDiff` | ` ` | serialization |
| `client/container_exec.go:71` | `client.*Client.ContainerExecInspect` | ` ` | serialization |
| `client/container_inspect.go:14` | `client.*Client.ContainerInspect` | ` ` | serialization |
| `client/container_inspect.go:32` | `client.*Client.ContainerInspectWithRaw` | ` ` | serialization, file_io |
| `client/container_top.go:13` | `client.*Client.ContainerTop` | ` ` | serialization |
| `client/container_update.go:11` | `client.*Client.ContainerUpdate` | ` ` | serialization |
| `client/node_inspect.go:13` | `client.*Client.NodeInspectWithRaw` | ` ` | serialization, file_io |
| `client/plugin_inspect.go:13` | `client.*Client.PluginInspectWithRaw` | ` ` | serialization, file_io |

_7 more members not listed._

_651 more families not listed._


# moby

container engine; a decade of accretion across daemon, API, and plugin layers

**What this rung shows:** scale, df caps, and the common-idiom suppression the retrieval channels exist for

| | |
|---|---|
| Corpus | [moby](https://github.com/moby/moby) |
| Pinned at | `v28.5.2` (`89c5e8fd66634b6128fc4c0e6f1236e2540e46e0`) |
| Project since | 2013 |
| doppel | `706150c` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 9 concepts modeled, 548 associations, 95 unusual realizations
Habitats: 168 modeled, 368 misfits; most uniform checker (norm 0.98), most diverse vfs (norm 0.56)
Conventions: strongest error_wrapping (0.62), loosest db_access (0.37)
Ecosystems: 3903 profiled (3712 dominance, 191 coalition, 0 conflict, 0 weak)
Found 8003 functions. Retrieving candidates...
Retrieval: shape 4137, concept 1271, call 12559 -> 16514 unique pairs
  concept-only 7.4%  call-only 67.3%  suppressed-shape functions: 376  large identity buckets: 4  surviving patterns: 19542
Running structural comparison on 16514 pairs...
Families: 603 over 706 components, 1536 functions in a family, 2963 edges completed
  6 pairs suppressed by max-per-func=2
```

# Code Similarity Report

**Functions analyzed:** 8003 | **Threshold:** 0.60 | **Pairs found:** 10

---

## Match #1 — Code-shape: `0.9514`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/networkdb/networkdb.pb.go:1944` | `networkdb.*NetworkEvent.Unmarshal` | ` ` | validation |
| **B** | `libnetwork/networkdb/networkdb.pb.go:2096` | `networkdb.*NetworkEntry.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.92  flow 1.00  sig 1.00  size 0.99`

**Evidence:** `1246.08` (shape 1230.35, concept 1.74, call 13.99)

**Trophic:** `0.95`

**Shared structure:**

- `27.90` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`
- `27.90` — `seq[ if(bin:>=(id,id)) ; assign:=(index) ]`
- `27.90` — `seq[ if(bin:>=(id,lit:INT)) ; if(bin:>=(id,id)) ]`

**Habitat:** A fits poorly in `networkdb` (fit 0.20, package norm 0.69)

**Habitat:** B fits poorly in `networkdb` (fit 0.20, package norm 0.69)

**Structural overlap:** `0.76` (merge-worthy)

- share 9 callees: [fmt.Errorf, github_com_hashicorp_serf_serf.LamportTime, int, int32, len, skipNetworkdb, string, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 106 shared
- share patterns: [validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [validation]
- same visibility
- both are methods, on *NetworkEvent and *NetworkEntry
- call into same packages: [ipbits, networkdb]

---

## Match #2 — Code-shape: `0.9231`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/networkdb/networkdb.pb.go:2096` | `networkdb.*NetworkEntry.Unmarshal` | ` ` | validation |
| **B** | `libnetwork/networkdb/networkdb.pb.go:2653` | `networkdb.*BulkSyncMessage.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.87  flow 1.00  sig 1.00  size 0.80`

**Evidence:** `1322.73` (shape 1307.00, concept 1.74, call 13.99)

**Trophic:** `0.92`

**Shared structure:**

- `27.90` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`
- `27.90` — `seq[ if(bin:>=(id,id)) ; assign:=(index) ]`
- `27.90` — `seq[ if(bin:>=(id,lit:INT)) ; if(bin:>=(id,id)) ]`

**Habitat:** A fits poorly in `networkdb` (fit 0.20, package norm 0.69)

**Habitat:** B fits poorly in `networkdb` (fit 0.20, package norm 0.69)

**Structural overlap:** `0.77` (merge-worthy)

- share 10 callees: [bool, fmt.Errorf, github_com_hashicorp_serf_serf.LamportTime, int, int32, len, skipNetworkdb, string, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 106 shared
- share patterns: [validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [validation]
- same visibility
- both are methods, on *NetworkEntry and *BulkSyncMessage
- call into same packages: [ipbits, networkdb]

---

## Match #3 — Code-shape: `0.9072`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `api/types/plugins/logdriver/entry.pb.go:360` | `logdriver.*LogEntry.Unmarshal` | ` ` | validation |
| **B** | `libnetwork/networkdb/networkdb.pb.go:2653` | `networkdb.*BulkSyncMessage.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.85  flow 1.00  sig 1.00  size 0.98`

**Evidence:** `1397.87` (shape 1396.13, concept 1.74, call 0.00)

**Trophic:** `0.93`

**Shared structure:**

- `33.48` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`
- `33.48` — `seq[ if(bin:>=(id,id)) ; assign:=(index) ]`
- `33.48` — `seq[ if(bin:>=(id,lit:INT)) ; if(bin:>=(id,id)) ]`

**Habitat:** A fits poorly in `logdriver` (fit 0.05, package norm 0.71)

**Habitat:** B fits poorly in `networkdb` (fit 0.20, package norm 0.69)

**Structural overlap:** `0.63` (merge-worthy)

- share 9 callees: [append, bool, fmt.Errorf, int, int32, len, string, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 106 shared
- share patterns: [validation]
- both are orchestrator functions
- callees do related work (1.00): [validation]
- same visibility
- both are methods, on *LogEntry and *BulkSyncMessage
- call into same packages: [ipbits]

---

## Match #4 — Code-shape: `0.8063`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/agent.pb.go:618` | `libnetwork.*EndpointRecord.Unmarshal` | ` ` | validation |
| **B** | `libnetwork/networkdb/networkdb.pb.go:2384` | `networkdb.*TableEvent.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.68  flow 1.00  sig 1.00  size 0.81`

**Evidence:** `1880.68` (shape 1878.93, concept 1.74, call 0.00)

**Trophic:** `0.83`

**Shared structure:**

- `50.21` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`
- `50.21` — `seq[ if(bin:>=(id,id)) ; assign:=(index) ]`
- `50.21` — `seq[ if(bin:>=(id,lit:INT)) ; if(bin:>=(id,id)) ]`

**Habitat:** B fits poorly in `networkdb` (fit 0.20, package norm 0.69)

**Structural overlap:** `0.63` (merge-worthy)

- share 8 callees: [append, fmt.Errorf, int, int32, len, string, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 106 shared
- share patterns: [validation]
- both are orchestrator functions
- callees do related work (1.00): [validation]
- same visibility
- both are methods, on *EndpointRecord and *TableEvent
- call into same packages: [ipbits]

---

## Match #5 — Code-shape: `0.9746`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/networkdb/networkdb.pb.go:1824` | `networkdb.*NodeEvent.Unmarshal` | ` ` | validation |
| **B** | `libnetwork/networkdb/networkdb.pb.go:1944` | `networkdb.*NetworkEvent.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.96  flow 1.00  sig 1.00  size 0.79`

**Evidence:** `1068.18` (shape 1052.45, concept 1.74, call 13.99)

**Trophic:** `0.91`

**Shared structure:**

- `22.32` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`
- `22.32` — `seq[ if(bin:>=(id,id)) ; assign:=(index) ]`
- `22.32` — `seq[ if(bin:>=(id,lit:INT)) ; if(bin:>=(id,id)) ]`

**Habitat:** A fits poorly in `networkdb` (fit 0.20, package norm 0.69)

**Habitat:** B fits poorly in `networkdb` (fit 0.20, package norm 0.69)

**Structural overlap:** `0.76` (merge-worthy)

- share 9 callees: [fmt.Errorf, github_com_hashicorp_serf_serf.LamportTime, int, int32, len, skipNetworkdb, string, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 106 shared
- share patterns: [validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [validation]
- same visibility
- both are methods, on *NodeEvent and *NetworkEvent
- call into same packages: [ipbits, networkdb]

---

## Match #6 — Code-shape: `0.8945`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `libnetwork/networkdb/networkdb.pb.go:1824` | `networkdb.*NodeEvent.Unmarshal` | ` ` | validation |
| **B** | `libnetwork/networkdb/networkdb.pb.go:2249` | `networkdb.*NetworkPushPull.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.82  flow 1.00  sig 1.00  size 0.86`

**Evidence:** `1031.56` (shape 1015.83, concept 1.74, call 13.99)

**Trophic:** `0.95`

**Shared structure:**

- `22.32` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`
- `22.32` — `seq[ if(bin:>=(id,id)) ; assign:=(index) ]`
- `22.32` — `seq[ if(bin:>=(id,lit:INT)) ; if(bin:>=(id,id)) ]`

**Habitat:** A fits poorly in `networkdb` (fit 0.20, package norm 0.69)

**Habitat:** B fits poorly in `networkdb` (fit 0.20, package norm 0.69)

**Structural overlap:** `0.75` (merge-worthy)

- share 9 callees: [fmt.Errorf, github_com_hashicorp_serf_serf.LamportTime, int, int32, len, skipNetworkdb, string, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 106 shared
- share patterns: [validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [validation]
- same visibility
- both are methods, on *NodeEvent and *NetworkPushPull
- call into same packages: [ipbits, networkdb]

---

## Match #7 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `api/types/plugins/logdriver/entry.pb.go:672` | `logdriver.skipEntry` | ` ` | validation |
| **B** | `daemon/cluster/internal/runtime/plugin.pb.go:725` | `runtime.skipPlugin` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `846.31` (shape 844.57, concept 1.74, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `30.16` — `return(lit:INT,id)`
- `25.39` — `assign+=(lit:INT)`
- `24.60` — `return(lit:INT,sel)`

**Habitat:** A fits poorly in `logdriver` (fit 0.06, package norm 0.71)

**Habitat:** B fits poorly in `runtime` (fit 0.06, package norm 0.70)

**Structural overlap:** `0.72` (merge-worthy)

- share 5 callees: [fmt.Errorf, int, len, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 106 shared
- share patterns: [validation]
- both are utility functions
- callers do related work (1.00): [validation]
- same visibility
- same receiver type: plain functions
- call into same packages: [ipbits]

---

## Match #8 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `api/types/plugins/logdriver/entry.pb.go:672` | `logdriver.skipEntry` | ` ` | validation |
| **B** | `libnetwork/agent.pb.go:1085` | `libnetwork.skipAgent` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `846.31` (shape 844.57, concept 1.74, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `30.16` — `return(lit:INT,id)`
- `25.39` — `assign+=(lit:INT)`
- `24.60` — `return(lit:INT,sel)`

**Habitat:** A fits poorly in `logdriver` (fit 0.06, package norm 0.71)

**Structural overlap:** `0.72` (merge-worthy)

- share 5 callees: [fmt.Errorf, int, len, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 106 shared
- share patterns: [validation]
- both are utility functions
- callers do related work (1.00): [validation]
- same visibility
- same receiver type: plain functions
- call into same packages: [ipbits]

---

## Match #9 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `daemon/cluster/internal/runtime/plugin.pb.go:725` | `runtime.skipPlugin` | ` ` | validation |
| **B** | `libnetwork/agent.pb.go:1085` | `libnetwork.skipAgent` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `846.31` (shape 844.57, concept 1.74, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `30.16` — `return(lit:INT,id)`
- `25.39` — `assign+=(lit:INT)`
- `24.60` — `return(lit:INT,sel)`

**Habitat:** A fits poorly in `runtime` (fit 0.06, package norm 0.70)

**Structural overlap:** `0.72` (merge-worthy)

- share 5 callees: [fmt.Errorf, int, len, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 106 shared
- share patterns: [validation]
- both are utility functions
- callers do related work (1.00): [validation]
- same visibility
- same receiver type: plain functions
- call into same packages: [ipbits]

---

## Match #10 — Code-shape: `0.8702`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `api/types/plugins/logdriver/entry.pb.go:360` | `logdriver.*LogEntry.Unmarshal` | ` ` | validation |
| **B** | `daemon/cluster/internal/runtime/plugin.pb.go:379` | `runtime.*PluginSpec.Unmarshal` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.78  flow 1.00  sig 1.00  size 0.94`

**Evidence:** `1338.05` (shape 1336.30, concept 1.74, call 0.00)

**Trophic:** `0.89`

**Shared structure:**

- `33.48` — `seq[ assign\|=(bin) ; if(bin:<(id,lit:INT)) ]`
- `33.48` — `seq[ if(bin:>=(id,id)) ; assign:=(index) ]`
- `33.48` — `seq[ if(bin:>=(id,lit:INT)) ; if(bin:>=(id,id)) ]`

**Habitat:** A fits poorly in `logdriver` (fit 0.05, package norm 0.71)

**Habitat:** B fits poorly in `runtime` (fit 0.06, package norm 0.70)

**Structural overlap:** `0.65` (merge-worthy)

- share 10 callees: [Unmarshal, append, bool, fmt.Errorf, int, int32, len, string, uint, uint64]
- overlapping call-graph neighborhoods (1.00): 106 shared
- share patterns: [validation]
- both are orchestrator functions
- callees do related work (1.00): [validation]
- same visibility
- both are methods, on *LogEntry and *PluginSpec
- call into same packages: [ipbits]

---

## Families

603 families, 1536 functions in a family, largest 44 members; 2963 edges scored here that retrieval never proposed

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

### Family 2 — 29 members, every pair `>= 1.00` code-shape  (276 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `api/types/plugins/logdriver/entry.pb.go:66` | `logdriver.*LogEntry.GetSource` | ` ` | — |
| `api/types/plugins/logdriver/entry.pb.go:147` | `logdriver.*PartialLogEntryMetadata.GetId` | ` ` | — |
| `daemon/cluster/internal/runtime/plugin.pb.go:68` | `runtime.*PluginSpec.GetName` | ` ` | — |
| `daemon/cluster/internal/runtime/plugin.pb.go:75` | `runtime.*PluginSpec.GetRemote` | ` ` | — |
| `daemon/cluster/internal/runtime/plugin.pb.go:144` | `runtime.*PluginPrivilege.GetName` | ` ` | — |
| `daemon/cluster/internal/runtime/plugin.pb.go:151` | `runtime.*PluginPrivilege.GetDescription` | ` ` | — |
| `libnetwork/agent.pb.go:111` | `libnetwork.*EndpointRecord.GetName` | ` ` | — |
| `libnetwork/agent.pb.go:118` | `libnetwork.*EndpointRecord.GetServiceName` | ` ` | — |
| `libnetwork/agent.pb.go:125` | `libnetwork.*EndpointRecord.GetServiceID` | ` ` | — |
| `libnetwork/agent.pb.go:132` | `libnetwork.*EndpointRecord.GetVirtualIP` | ` ` | — |

_19 more members not listed._

### Family 3 — 17 members, every pair `>= 0.86` code-shape  (66 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `api/types/plugins/logdriver/entry.pb.go:188` | `logdriver.*LogEntry.Marshal` | ` ` | — |
| `api/types/plugins/logdriver/entry.pb.go:252` | `logdriver.*PartialLogEntryMetadata.Marshal` | ` ` | — |
| `daemon/cluster/internal/runtime/plugin.pb.go:191` | `runtime.*PluginSpec.Marshal` | ` ` | — |
| `daemon/cluster/internal/runtime/plugin.pb.go:261` | `runtime.*PluginPrivilege.Marshal` | ` ` | — |
| `libnetwork/agent.pb.go:340` | `libnetwork.*EndpointRecord.Marshal` | ` ` | — |
| `libnetwork/agent.pb.go:440` | `libnetwork.*PortConfig.Marshal` | ` ` | — |
| `libnetwork/drivers/overlay/overlay.pb.go:141` | `overlay.*PeerRecord.Marshal` | ` ` | — |
| `libnetwork/drivers/windows/overlay/overlay.pb.go:101` | `overlay.*PeerRecord.Marshal` | ` ` | — |
| `libnetwork/networkdb/networkdb.pb.go:952` | `networkdb.*GossipMessage.Marshal` | ` ` | — |
| `libnetwork/networkdb/networkdb.pb.go:987` | `networkdb.*NodeEvent.Marshal` | ` ` | — |

_7 more members not listed._

### Family 4 — 17 members, every pair `>= 0.78` code-shape  (61 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `api/types/plugins/logdriver/entry.pb.go:360` | `logdriver.*LogEntry.Unmarshal` | ` ` | validation |
| `api/types/plugins/logdriver/entry.pb.go:551` | `logdriver.*PartialLogEntryMetadata.Unmarshal` | ` ` | validation |
| `daemon/cluster/internal/runtime/plugin.pb.go:379` | `runtime.*PluginSpec.Unmarshal` | ` ` | validation |
| `daemon/cluster/internal/runtime/plugin.pb.go:579` | `runtime.*PluginPrivilege.Unmarshal` | ` ` | validation |
| `libnetwork/agent.pb.go:618` | `libnetwork.*EndpointRecord.Unmarshal` | ` ` | validation |
| `libnetwork/agent.pb.go:946` | `libnetwork.*PortConfig.Unmarshal` | ` ` | validation |
| `libnetwork/drivers/overlay/overlay.pb.go:243` | `overlay.*PeerRecord.Unmarshal` | ` ` | validation |
| `libnetwork/drivers/windows/overlay/overlay.pb.go:197` | `overlay.*PeerRecord.Unmarshal` | ` ` | validation |
| `libnetwork/networkdb/networkdb.pb.go:1721` | `networkdb.*GossipMessage.Unmarshal` | ` ` | validation |
| `libnetwork/networkdb/networkdb.pb.go:1824` | `networkdb.*NodeEvent.Unmarshal` | ` ` | validation |

_7 more members not listed._

### Family 5 — 17 members, every pair `>= 0.61` code-shape  (74 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `api/types/plugins/logdriver/entry.pb.go:308` | `logdriver.*LogEntry.Size` | ` ` | — |
| `api/types/plugins/logdriver/entry.pb.go:335` | `logdriver.*PartialLogEntryMetadata.Size` | ` ` | — |
| `daemon/cluster/internal/runtime/plugin.pb.go:318` | `runtime.*PluginSpec.Size` | ` ` | — |
| `daemon/cluster/internal/runtime/plugin.pb.go:350` | `runtime.*PluginPrivilege.Size` | ` ` | — |
| `libnetwork/agent.pb.go:496` | `libnetwork.*EndpointRecord.Size` | ` ` | — |
| `libnetwork/agent.pb.go:546` | `libnetwork.*PortConfig.Size` | ` ` | — |
| `libnetwork/drivers/overlay/overlay.pb.go:196` | `overlay.*PeerRecord.Size` | ` ` | — |
| `libnetwork/drivers/windows/overlay/overlay.pb.go:146` | `overlay.*PeerRecord.Size` | ` ` | — |
| `libnetwork/networkdb/networkdb.pb.go:1387` | `networkdb.*GossipMessage.Size` | ` ` | — |
| `libnetwork/networkdb/networkdb.pb.go:1403` | `networkdb.*NodeEvent.Size` | ` ` | — |

_7 more members not listed._

_598 more families not listed._


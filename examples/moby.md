# moby

container engine; a decade of accretion across daemon, API, and plugin layers

**What this rung shows:** scale, df caps, and the common-idiom suppression the retrieval channels exist for

| | |
|---|---|
| Corpus | [moby](https://github.com/moby/moby) |
| Pinned at | `v28.5.2` (`89c5e8fd66634b6128fc4c0e6f1236e2540e46e0`) |
| Project since | 2013 |
| doppel | `783a0d4` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`; CI regenerates on every push to master.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Learning concept vocabulary...
Lexicon: 519 concepts (12 seeded, 507 emergent), 22469/57681 features above 2817 df, 721 functions unlabeled
Generating concept documents...
Calibration: rate 0.01 over 20000 null pairs -> threshold 0.36, struct-min 0.30, family-min 0.36
Found 7658 functions. Retrieving candidates...
Retrieval: shape 9451, concept 23375, call 12470 -> 40225 unique pairs
  concept-only 51.4%  call-only 22.4%  suppressed-shape functions: 11  large identity buckets: 2  surviving labels: 34380
  22 cross test/prod pairs dropped
  1788 cross build-target pairs dropped (no build compiles both files)
Running structural comparison on 38415 pairs...
  Concept views: 2893 of 38415 compared pairs disagree with the taxonomy (372 vocabulary the tree misses, 2521 kinship the vocabularies lack)
  18545 pairs remain after struct-min=0.30 filter
Culture: 488 concepts modeled, 3281 associations, 902 unusual realizations
Habitats: 167 modeled, 71 misfits (94 excused by subsystem), 59 subsystems; most uniform checker (norm 0.97), most diverse suite (norm 0.58)
Conventions: strongest nl.manager+nl.ns (0.98), loosest aSpace.allocated+netiputil.PrefixCompare (0.16)
Ecosystems: 7325 profiled (4541 dominance, 2781 coalition, 0 conflict, 3 weak)
Families: 866 over 818 components, 2209 functions in a family, 6581 edges completed
  2 component(s) skipped as too large or too dense: sizes [144 1054]
```

# Code Similarity Report

**Functions analyzed:** 7658 | **Threshold:** 0.36 | **Pairs found:** 10

---

## What doppel sees

**7658 functions** across **235 packages** — test functions excluded. Structural roles: 4307 leaf, 1840 orchestrator, 412 passthrough, 1099 utility.

### Concepts

Two pictures of the same vocabulary. The first is what doppel **searched with**: an authored tree of fourteen seed practices, each leaf showing how many functions here ended up in a concept that seed grew. It is the same shape on every corpus, which is what makes it the one concept picture two runs can be compared on. The second is what this corpus **turned out to have**: concepts learned from the code itself, named after the evidence that identified them, hung from that same interior — so two functions under one *branch* score partial credit rather than nothing. Counts are members; membership is graded, and a function can carry several.

**What doppel looked for, and how much of it grew here.**

```mermaid
flowchart LR
    s0(["concept"])
    s1(["io_operation"])
    s2(["remote_io"])
    s3["http_call<br/>27"]
    s4["grpc_call<br/>absent"]
    s5(["data_store_access"])
    s6["db_access<br/>25"]
    s7["caching<br/>251"]
    s8["transaction<br/>189"]
    s9["file_io<br/>407"]
    s10["logging<br/>238"]
    s11(["data_transformation"])
    s12["mapping<br/>218"]
    s13["validation<br/>437"]
    s14["serialization<br/>309"]
    s15(["control_flow"])
    s16["concurrency<br/>688"]
    s17(["fault_tolerance"])
    s18["retry<br/>85"]
    s19["circuit_breaker<br/>absent"]
    s20(["error_handling"])
    s21["error_wrapping<br/>636"]
    s0 --> s1
    s1 --> s2
    s2 --> s3
    s2 --> s4
    s1 --> s5
    s5 --> s6
    s5 --> s7
    s5 --> s8
    s1 --> s9
    s1 --> s10
    s0 --> s11
    s11 --> s12
    s11 --> s13
    s11 --> s14
    s0 --> s15
    s15 --> s16
    s15 --> s17
    s17 --> s18
    s17 --> s19
    s0 --> s20
    s20 --> s21
    classDef good fill:#d7ecd9,color:#1b3d20
    classDef warn fill:#fbeecb,color:#4a3a12
    classDef hot fill:#f7d6d6,color:#4a1c1c
    class s4,s19 hot
```

**What it learned instead.**

```mermaid
flowchart LR
    c0(["concept"])
    c1(["io_operation"])
    c2(["remote_io"])
    c3(["data_store_access"])
    c4(["data_transformation"])
    c5(["control_flow"])
    c6(["fault_tolerance"])
    c7(["error_handling"])
    c8["C.__u32+C.int<br/>636"]
    c9["Config.OpenStdin+Config.StdinOnce<br/>688"]
    c10["Healthcheck.Retries+c.callWithRetry<br/>85"]
    c11["Isolation.IsValid+PluginObj.PluginReference<br/>437"]
    c12["LittleEndian.Uint64+tx.onRollback<br/>189"]
    c13["Store.validateName+bytes.TrimSpace<br/>407"]
    c14["Task.GetContainer+enginemount.Mount<br/>218"]
    c15["Task.Runtime+c.callWithRetry<br/>309"]
    c16["c.cache+client.PruneInfo<br/>251"]
    c17["config.MetaHeaders+config.AuthConfig+p.repoName<br/>80"]
    c18["config.processIPAM+d.createNetwork<br/>71"]
    c19["ctr.terminateInvoked+diagnostic.TableObj<br/>238"]
    c20["d.Sock+time.Second<br/>57"]
    c21["daemon.containerdClient+daemon.imageService<br/>158"]
    c22["errdefs.Forbidden+errdefs.InvalidParameter<br/>204"]
    c23["http.StatusConflict+http.StatusNotImplemented<br/>50"]
    c24["img.RawJSON+img.OS<br/>146"]
    c25["n.hasSpecialDriver+n.ipamType<br/>95"]
    c26["nat.Port+fmt.Sprintf<br/>86"]
    c27["options.quota+quota.Size<br/>139"]
    c28["pd.layer+layer.DiffID<br/>93"]
    c0 --> c1
    c1 --> c2
    c1 --> c3
    c0 --> c4
    c0 --> c5
    c5 --> c6
    c0 --> c7
    c7 --> c8
    c5 --> c9
    c6 --> c10
    c4 --> c11
    c3 --> c12
    c1 --> c13
    c4 --> c14
    c4 --> c15
    c3 --> c16
    c6 --> c17
    c5 --> c18
    c1 --> c19
    c2 --> c20
    c7 --> c21
    c7 --> c22
    c2 --> c23
    c3 --> c24
    c5 --> c25
    c2 --> c26
    c1 --> c27
    c6 --> c28
```

The diagram draws the 3 largest concepts on each branch; **498 further concepts** are left out of the picture and listed in the table below.

**No practice here for** `circuit_breaker`, `grpc_call`. Concepts are learned from this corpus, so one can never be absent — it exists because functions carry it. These are the *seeds* the search started from that grew nothing: a direct answer to "does this codebase already do X".

| Concept | Functions | Convention |
|---|---:|---|
| `Config.OpenStdin+Config.StdinOnce` | 688 | `0.54` (settled) |
| `C.__u32+C.int` | 636 | `0.63` (settled) |
| `Isolation.IsValid+PluginObj.PluginReference` | 437 | `0.60` (settled) |
| `Store.validateName+bytes.TrimSpace` | 407 | `0.56` (settled) |
| `Task.Runtime+c.callWithRetry` | 309 | `0.55` (settled) |
| `c.cache+client.PruneInfo` | 251 | `0.62` (settled) |
| `ctr.terminateInvoked+diagnostic.TableObj` | 238 | `0.52` (settled) |
| `Task.GetContainer+enginemount.Mount` | 218 | `0.58` (settled) |
| `errdefs.Forbidden+errdefs.InvalidParameter` | 204 | `0.56` (settled) |
| `LittleEndian.Uint64+tx.onRollback` | 189 | `0.53` (settled) |
| `daemon.containerdClient+daemon.imageService` | 158 | `0.59` (settled) |
| `img.RawJSON+img.OS` | 146 | `0.62` (settled) |
| `options.quota+quota.Size` | 139 | `0.57` (settled) |
| `platforms.Parse+ocispec.Platform` | 139 | `0.55` (settled) |
| `strconv.FormatInt+strconv.Itoa` | 130 | `0.63` (settled) |
| `config.Backend+c.config` | 127 | `0.48` (loose) |
| `Config.Image+container.Name` | 123 | `0.56` (settled) |
| `NetworkSettings.Ports+NetworkSettings.SandboxID` | 118 | `0.57` (settled) |
| `container.StreamConfig+container.Config` | 107 | `0.51` (settled) |
| `container.RestartManager+container.*Container.Restar…` | 105 | `0.56` (settled) |
| `n.skipGwAllocIPv4+n.skipGwAllocIPv6+n.inDelete` | 101 | `0.42` (loose) |
| `n.addrSpace+n.skipGwAllocIPv4` | 99 | `0.37` (loose) |
| `distribution.Config+i.registryService` | 97 | `0.58` (settled) |
| `n.hasSpecialDriver+n.ipamType` | 95 | `0.43` (loose) |
| `pd.layer+layer.DiffID` | 93 | `0.59` (settled) |
| `nat.PortBinding+nat.PortMap` | 87 | `0.58` (settled) |
| `config.HnsID+d.name` | 86 | `0.53` (settled) |
| `nat.Port+fmt.Sprintf` | 86 | `0.53` (settled) |
| `options.size+driver.options` | 86 | `0.50` (settled) |
| `Healthcheck.Retries+c.callWithRetry` | 85 | `0.52` (settled) |
| `netlabel.MacAddress+options` | 82 | `0.52` (settled) |
| `config.MetaHeaders+config.AuthConfig+p.repoName` | 80 | `0.52` (settled) |
| `Gateway.IP+Pool.String` | 75 | `0.60` (settled) |
| `i.walkPresentChildren+c8dimages.IsIndexType` | 75 | `0.47` (loose) |
| `types.MediaTypeMultiplexedS…+types.MediaTypeRawStream` | 74 | `0.37` (loose) |
| `config.processIPAM+d.createNetwork` | 71 | `0.54` (settled) |
| `daemon.UsingSystemd+remote` | 71 | `0.50` (settled) |
| `ep.getDNSNames+ep.svcID` | 70 | `0.44` (loose) |
| `netlabel.Internal+netlabel.EnableIPv6+netlabel.EnableIPv4` | 70 | `0.38` (loose) |
| `os.Setenv+os.Getenv` | 70 | `0.51` (settled) |
| `TmpfsOptions.Mode+TmpfsOptions.SizeBytes+m.ReadOnly` | 69 | `0.56` (settled) |
| `typ.Prefix+typ.Capability` | 67 | `0.48` (loose) |
| `config.ProgressOutput+p.repoName` | 64 | `0.51` (settled) |
| `scope.Global+r.RegisterDriver` | 64 | `0.67` (settled) |
| `c.Name+log.Fields` | 63 | `0.36` (loose) |
| `ctr.ID+errors.New` | 62 | `0.54` (settled) |
| `nlh.AddrAdd+netlink.FAMILY_V4` | 62 | `0.36` (loose) |
| `types.MediaTypeMultiplexedS…+types.MediaTypeRawStream+httputils.DecodePlatform` | 62 | `0.41` (loose) |
| `Config.Hostname+container.Config` | 61 | `0.50` (settled) |
| `cfg.Root+filepath.Join` | 61 | `0.43` (loose) |
| `httputils.BoolValueOrDefault+types.MediaTypeMultiplexedS…` | 61 | `0.41` (loose) |
| `nlh.AddrAdd+netlink.FAMILY_V4+netlink.FAMILY_V6` | 61 | `0.38` (loose) |
| `m.Options+s.Root` | 60 | `0.49` (loose) |
| `types.NotImplementedErrorf+bitmap+overlayutils` | 59 | `0.71` (settled) |
| `bnd.HostIP+bnd.IP` | 58 | `0.51` (settled) |
| `d.Sock+time.Second` | 57 | `0.52` (settled) |
| `filepath.ToSlash+query.Set` | 56 | `0.34` (loose) |
| `ocispec.AnnotationRefName+ref.Name` | 56 | `0.47` (loose) |
| `d.secMap+k.tag` | 55 | `0.35` (loose) |
| `directory.Size+d.getDiffPath` | 55 | `0.52` (settled) |
| `n.Scope+scope.Global` | 55 | `0.48` (loose) |
| `types.NotImplementedErrorf+bitmap` | 55 | `0.76` (unanimous) |
| `Spec.BindOptions+mp.Spec` | 54 | `0.54` (settled) |
| `imagetypes.Summary+Config.Labels` | 54 | `0.55` (settled) |
| `n.skipGwAllocIPv4+n.skipGwAllocIPv6` | 54 | `0.41` (loose) |
| `daemon.UsingSystemd+initlayer` | 53 | `0.55` (settled) |
| `img.Platform+fmt.Errorf` | 52 | `0.63` (settled) |
| `Config.ExposedPorts+nat.Port` | 51 | `0.57` (settled) |
| `libnetwork.NetworkOptionIpam+libnetwork.IpamConf+netlabel.EnableIPv6` | 51 | `0.51` (settled) |
| `config.HnsID+d.name+d.getNetwork` | 50 | `0.52` (settled) |
| `http.StatusConflict+http.StatusNotImplemented` | 50 | `0.56` (settled) |
| `span.AddEvent+span.SetAttributes` | 49 | `0.41` (loose) |
| `portConfig.Name+Endpoint.Ports` | 47 | `0.51` (settled) |
| `builder.Stderr+c.ShellDependantCmdLine+c.String` | 46 | `0.44` (loose) |
| `discoverapi.Discover+driverapi.Driver+drvRegistry.WalkDrivers` | 46 | `0.42` (loose) |
| `fstype.GetFSMagic+overlayutils` | 46 | `0.48` (loose) |
| `ipam.RequestAddress+ipamapi.ErrNoAvailableIPs` | 46 | `0.47` (loose) |
| `netlabel.Internal+netlabel.EnableIPv6` | 46 | `0.53` (settled) |
| `reference.Path+reference.Domain` | 45 | `0.52` (settled) |
| `Process.Args+s.Root` | 44 | `0.53` (settled) |
| `cgroups.Mode+cgroups.Unified` | 44 | `0.58` (settled) |
| `config.ProgressOutput+config.ReferenceStore` | 44 | `0.47` (loose) |
| `containersReplica.ReserveNa…+cerrdefs.IsConflict` | 44 | `0.42` (loose) |
| `ctr.IsPaused+ctr.IsRestarting` | 44 | `0.40` (loose) |
| `daemon.setupLinkedContainers+Config.User` | 44 | `0.52` (settled) |
| `l.is+l.lss+is.Get` | 44 | `0.52` (settled) |
| `network.config+d.link` | 44 | `0.42` (loose) |
| `ref.Name+options.Platform` | 44 | `0.45` (loose) |
| `discoverapi.Discover+driverapi.Driver` | 42 | `0.57` (settled) |
| `Config.Image+Config.Labels` | 41 | `0.47` (loose) |
| `errors.Errorf+fluent` | 41 | `0.55` (settled) |
| `ld.desc+desc.Digest` | 41 | `0.53` (settled) |
| `task.Networks+Network.Spec` | 41 | `0.47` (loose) |
| `driverapi.NetworkPluginEndp…+ipamapi.PluginEndpointType+daemon.netController` | 40 | `0.45` (loose) |
| `ipamapi.PluginEndpointType+driverapi.NetworkPluginEndp…+context.TODO` | 40 | `0.55` (settled) |
| `time.RFC3339+time.Now` | 40 | `0.51` (settled) |
| `daemon.clusterProvider+fmt.Errorf` | 39 | `0.52` (settled) |
| `img.RawJSON+img.OS+path.Join` | 39 | `0.57` (settled) |
| `networktypes.EndpointSettin…+network.EndpointSettings` | 39 | `0.45` (loose) |
| `options.Signal+options.Timeout` | 39 | `0.51` (settled) |
| `r.FormValue+pr.backend` | 39 | `0.56` (settled) |
| `subnet.GatewayAddress+hcsshim.HNSNetworkRequest` | 39 | `0.39` (loose) |
| `metrics.ContainerActions+ContainerActions.WithValues` | 38 | `0.41` (loose) |
| `txn.First+containers.First` | 38 | `0.41` (loose) |
| `c.IsRunning+daemon.config` | 37 | `0.49` (loose) |
| `c.WalkNetworks+libnetwork.*Controller.Walk…` | 37 | `0.52` (settled) |
| `container.spec+c.spec` | 37 | `0.54` (settled) |
| `epConfig.IPAMConfig+containertypes.NetworkMode` | 37 | `0.41` (loose) |
| `label.Relabel+container.MountLabel` | 37 | `0.50` (loose) |
| `volumeopts.WithCreateRefere…+volumes.Create` | 37 | `0.54` (settled) |
| `cfg.AuthorizationPlugins+cfg.Experimental` | 36 | `0.35` (loose) |
| `containerdlabels.LabelDistr…+source.registryRef` | 36 | `0.49` (loose) |
| `globalLock.RLock+globalLock.RUnlock` | 36 | `0.51` (settled) |
| `i.StorageDriver+client.LeasesService` | 36 | `0.52` (settled) |
| `HostConfig.RestartPolicy+RestartPolicy.Name` | 35 | `0.55` (settled) |
| `IPAMConfig.IPv4Address+IPAMConfig.IPv6Address` | 35 | `0.65` (settled) |
| `a.plugin+logdriver` | 35 | `0.53` (settled) |
| `b.Stdout+b.options` | 35 | `0.50` (settled) |
| `cerrdefs.IsAlreadyExists+errors.Wrap` | 35 | `0.41` (loose) |
| `daemon.runAsHyperVContainer+errors.Wrap` | 35 | `0.51` (settled) |
| `httpstatus.FromError+r.Method` | 35 | `0.48` (loose) |
| `system.Runtime+system` | 35 | `0.60` (settled) |
| `compression.DecompressStream+log.G` | 34 | `0.59` (settled) |
| `errcode.ErrorCodeDenied+f.err` | 34 | `0.54` (settled) |
| `fmt.Fprintln+fmt.Fprintf` | 34 | `0.42` (loose) |
| `libnetwork.NetworkOptionIpam+libnetwork.IpamConf+netlabel.EnableIPv4` | 34 | `0.45` (loose) |
| `Spec.BindOptions+m.Propagation` | 33 | `0.45` (loose) |
| `canonical.Digest+reference.WithDigest` | 33 | `0.42` (loose) |
| `ctr.RWLayer+daemon.LogContainerEvent` | 33 | `0.42` (loose) |
| `gogotypes.DurationFromProto+types.DurationFromProto` | 33 | `0.51` (settled) |
| `oci.DefaultSpec+oci` | 33 | `0.53` (settled) |
| `rp.Name+n.IsHost` | 33 | `0.54` (settled) |
| `c.MountLabel+idtools.Identity` | 32 | `0.48` (loose) |
| `ipamapi.PluginEndpointType+driverapi.NetworkPluginEndp…` | 32 | `0.43` (loose) |
| `n.path+netns.GetFromPath` | 32 | `0.35` (loose) |
| `netlabel.ExposedPorts+types.TransportPort` | 32 | `0.56` (settled) |
| `network.config+d.link+config.EnableICC` | 32 | `0.39` (loose) |
| `b.options+state.imageID` | 31 | `0.54` (settled) |
| `ep.Type+ep.nid` | 31 | `0.65` (settled) |
| `h.Created+h.CreatedBy` | 31 | `0.52` (settled) |
| `libnetwork.Network+daemon.FindNetwork` | 31 | `0.48` (loose) |
| `p.HostPortEnd+p.HostIP+p.HostPort` | 31 | `0.42` (loose) |
| `sb.controller+sb.containerID+sb.Endpoints` | 31 | `0.40` (loose) |
| `sysinfo.New+os.Getenv` | 31 | `0.40` (loose) |
| `daemonCfg.Rootless+specs.Linux+HostConfig.Privileged` | 30 | `0.47` (loose) |
| `n.secure+context.TODO` | 30 | `0.36` (loose) |
| `sb.containerID+sb.Endpoints` | 30 | `0.36` (loose) |
| `m.RW+m.Propagation` | 29 | `0.46` (loose) |
| `na.tasks+t.Networks` | 29 | `0.56` (settled) |
| `ncfg.Type+ncfg.ID` | 29 | `0.60` (settled) |
| `archiver.Untar+archiver.IDMapping` | 28 | `0.47` (loose) |
| `daemonCfg.Rootless+specs.Linux` | 28 | `0.50` (loose) |
| `errdefs.Unknown+errdefs.System` | 28 | `0.43` (loose) |
| `l.is+l.lss` | 28 | `0.47` (loose) |
| `libnetwork.Network+nw.ID+nw.Name` | 28 | `0.45` (loose) |
| `sort.Strings+instructions` | 28 | `0.42` (loose) |
| `BindOptions.ReadOnlyNonRecu…+BindOptions.CreateMountpoint` | 27 | `0.55` (settled) |
| `Config.Hostname+ctr.Config` | 27 | `0.51` (settled) |
| `Created.Format+storage.DriverData` | 27 | `0.51` (settled) |
| `config.Tail+config.Follow` | 27 | `0.52` (settled) |
| `container.GetRootResourcePa…+container.*Container.GetRoo…` | 27 | `0.43` (loose) |
| `hostURL.Host+hostURL.Scheme` | 27 | `0.48` (loose) |
| `httputils.BoolValue+ir.backend` | 27 | `0.39` (loose) |
| `listener.Close+proxy.listener` | 27 | `0.54` (settled) |
| `ocispec.History+ocispec.RootFS` | 27 | `0.48` (loose) |
| `plugingetter.Release+plugins` | 27 | `0.49` (loose) |
| `resp.Status+http.NewRequestWithContext` | 27 | `0.44` (loose) |
| `s.backend+httputils.VersionFromContext` | 27 | `0.44` (loose) |
| `child.Name+linkIndex.children` | 26 | `0.43` (loose) |
| `libnetwork.Network+nw.ID` | 26 | `0.43` (loose) |
| `p.Manifest+tlsconfig` | 26 | `0.50` (loose) |
| `router.NewDeleteRoute+router.NewGetRoute` | 26 | `0.24` (loose) |
| `b.bytes+l.logStreamName` | 25 | `0.48` (loose) |
| `container.DetachAndUnmount+daemon.isOnlineFSOperationP…` | 25 | `0.42` (loose) |
| `db.store+store.Txn` | 25 | `0.40` (loose) |
| `fmt.Fprint+os.Exit` | 25 | `0.38` (loose) |
| `plugins.Handle+plugins.Client` | 25 | `0.47` (loose) |
| `tracing.Attribute+span.SetAttributes` | 25 | `0.37` (loose) |
| `Reference.String+p.platform` | 24 | `0.52` (settled) |
| `c8dimages.IsLayerType+desc.MediaType` | 24 | `0.39` (loose) |
| `manifestlist.DeserializedMa…+schema1.SignedManifest` | 24 | `0.47` (loose) |
| `p.platform+p.desc` | 24 | `0.54` (settled) |
| `runConfig.Cmd+container.Config+build` | 24 | `0.49` (loose) |
| `v2.*Plugin.GetRefCount+daemon.*Daemon.LogPluginEve…` | 24 | `0.39` (loose) |
| `GwModeIPv6.isolated+config.GwModeIPv6` | 23 | `0.44` (loose) |
| `c.getAgent+context.TODO` | 23 | `0.35` (loose) |
| `d.call+api` | 23 | `0.62` (settled) |
| `i.logImageEvent+events.ActionUnTag` | 23 | `0.34` (loose) |
| `i.walkPresentChildren+c8dimages.IsIndexType+c8dimages.IsManifestType` | 23 | `0.29` (loose) |
| `iPort.Protocol+iPort.PublishedPort` | 23 | `0.35` (loose) |
| `net.CIDRMask+sb.getGatewayEndpoint` | 23 | `0.38` (loose) |
| `nlHandle.LinkByName+n.nlHandle` | 23 | `0.36` (loose) |
| `nlh.LinkAdd+parentLink.Attrs` | 23 | `0.33` (loose) |
| `nr.config+c.nr` | 23 | `0.33` (loose) |
| `source.registryRef+reference.Path` | 23 | `0.40` (loose) |
| `IPNet.String+nlwrap.AddrList+addr.IPNet` | 22 | `0.38` (loose) |
| `Status.Err+task.Status` | 22 | `0.32` (loose) |
| `d.options+fmt.Errorf` | 22 | `0.57` (settled) |
| `ep.endpointInGWNetwork+sb.Endpoints` | 22 | `0.43` (loose) |
| `http.ProxyFromEnvironment+net.Dialer` | 22 | `0.50` (loose) |
| `opts.WithGetDriver+vol.Driver` | 22 | `0.41` (loose) |
| `ref.Name+tagged.Tag` | 22 | `0.40` (loose) |
| `st.Completed+progress.NewFromContext` | 22 | `0.50` (loose) |
| `t.Chains+t.Family` | 22 | `0.30` (loose) |
| `Privileges.AppArmor+Privileges.CredentialSpec` | 21 | `0.49` (loose) |
| `c.LeasesService+lm.Delete` | 21 | `0.42` (loose) |
| `container.GetResourcePath+stat.IsDir` | 21 | `0.42` (loose) |
| `daemon.logClusterEvent+task.Networks` | 21 | `0.30` (loose) |
| `hcsshim.GetLayerMountPath+d.getLayerChain` | 21 | `0.45` (loose) |
| `libnetwork.ErrNoSuchNetwork+libnetwork.Network` | 21 | `0.45` (loose) |
| `netip.PrefixFrom+netip.Prefix` | 21 | `0.39` (loose) |
| `opts.validator+opts.values` | 21 | `0.64` (settled) |
| `parentLink.Attrs+netlink.Bridge` | 21 | `0.36` (loose) |
| `plugins.ErrNotFound+plugins.Get` | 21 | `0.40` (loose) |
| `registry.DecodeAuthConfig+registry.AuthHeader` | 21 | `0.44` (loose) |
| `sb.Labels+sb.getEndpointInGWNetwork` | 21 | `0.41` (loose) |
| `Platform.Variant+Platform.OSVersion` | 20 | `0.41` (loose) |
| `backend.GetNetworks+backend.NetworkListConfig` | 20 | `0.26` (loose) |
| `builder.Stderr+c.ShellDependantCmdLine` | 20 | `0.46` (loose) |
| `c8dimages.MediaTypeDockerSc…+ocispec.MediaTypeImageLayer…` | 20 | `0.42` (loose) |
| `cfg.TLS+log.SetLevel` | 20 | `0.49` (loose) |
| `cleanups+idtools` | 20 | `0.56` (settled) |
| `distribution.Config+i.registryService+streamformatter.NewJSONProg…` | 20 | `0.41` (loose) |
| `hcsshim.DeactivateLayer+hcsshim.UnprepareLayer` | 20 | `0.47` (loose) |
| `srslog+loggerutils` | 20 | `0.51` (settled) |
| `swarmtypes.ServiceSpec+spec.TaskTemplate` | 20 | `0.42` (loose) |
| `Config.Linux+Config.Mounts` | 19 | `0.38` (loose) |
| `Spec.GetNetwork+ic.Range+network.IPAM` | 19 | `0.40` (loose) |
| `broadcaster.Write+nDB.broadcaster` | 19 | `0.36` (loose) |
| `containers.First+c.ImageID` | 19 | `0.33` (loose) |
| `dns.TypeAAAA+dns.TypePTR` | 19 | `0.42` (loose) |
| `eventsService.Log+i.eventsService` | 19 | `0.32` (loose) |
| `img.Author+img.OSFeatures` | 19 | `0.44` (loose) |
| `iptable.Exists+iptable.RawCombinedOutput` | 19 | `0.27` (loose) |
| `nftables.Enabled+fmt.Errorf` | 19 | `0.41` (loose) |
| `p.parseMountSpec+lazyregexp` | 19 | `0.51` (settled) |
| `path.IsAbs+path.Join` | 19 | `0.45` (loose) |
| `rootless.RunningWithRootles…+log.G` | 19 | `0.40` (loose) |
| `sb.controller+sb.containerID` | 19 | `0.41` (loose) |
| `signal.ParseSignal+errdefs.InvalidParameter` | 19 | `0.35` (loose) |
| `stats+bufio` | 19 | `0.51` (settled) |
| `testEnv.DaemonInfo+DaemonInfo.OSType` | 19 | `0.41` (loose) |
| `a.call+api` | 18 | `0.65` (settled) |
| `ipam.ReleaseAddress+context.TODO` | 18 | `0.28` (loose) |
| `link.Attrs+n.nlHandle` | 18 | `0.41` (loose) |
| `loggerutils.DefaultTemplate+info.ExtraAttributes` | 18 | `0.44` (loose) |
| `strconv.FormatBool+strconv.Itoa` | 18 | `0.36` (loose) |
| `subnetIP.Mask+ip.IP` | 18 | `0.43` (loose) |
| `HostInfoFunctions.WithValues+metrics.HostInfoFunctions` | 17 | `0.27` (loose) |
| `Spec.GetContainer+task.Spec` | 17 | `0.43` (loose) |
| `c.newNS+unix.Gettid` | 17 | `0.30` (loose) |
| `containertypes.Isolation+defaultIsolation.IsHyperV` | 17 | `0.39` (loose) |
| `d.keys+addr.Mask` | 17 | `0.29` (loose) |
| `dns.TypeAAAA+dns.A` | 17 | `0.46` (loose) |
| `driverapi.NetworkPluginEndp…+ipamapi.PluginEndpointType` | 17 | `0.39` (loose) |
| `ec.User+Container.ID` | 17 | `0.37` (loose) |
| `ep.Delete+context.WithoutCancel+fmt.Errorf` | 17 | `0.34` (loose) |
| `fs.GetMetadata+fs.SetMetadata` | 17 | `0.42` (loose) |
| `md.NDotsFrom+md.NSOverride` | 17 | `0.41` (loose) |
| `reference.ParseNamed+reference.Digested` | 17 | `0.33` (loose) |
| `GwModeIPv4.isolated+config.GwModeIPv4` | 16 | `0.46` (loose) |
| `addr.Unmap+netip.AddrFromSlice` | 16 | `0.35` (loose) |
| `addr.Unmap+netip.AddrFromSlice+fmt.Errorf` | 16 | `0.34` (loose) |
| `container.name+c.container` | 16 | `0.65` (settled) |
| `containers.List+daemon.containers` | 16 | `0.49` (loose) |
| `e.backend+e.dependencies` | 16 | `0.45` (loose) |
| `imgID.Digest+referenceStore.References` | 16 | `0.37` (loose) |
| `libnetwork.NetworkOptionIpam+libnetwork.IpamConf` | 16 | `0.34` (loose) |
| `nw.IPAM+Driver.Options` | 16 | `0.43` (loose) |
| `options+containers` | 16 | `0.55` (settled) |
| `p.DefaultCopyMode+VolumeOptions.NoCopy` | 16 | `0.43` (loose) |
| `sctp.ListenSCTP+net.ListenTCP` | 16 | `0.30` (loose) |
| `client.TaskList+inspect.State` | 15 | `0.46` (loose) |
| `connection.sysObj+sysObj.Call` | 15 | `0.40` (loose) |
| `http.StatusNoContent+io.Writer` | 15 | `0.40` (loose) |
| `i.Address+i.AddressIPv6` | 15 | `0.36` (loose) |
| `i.StorageDriver+client.LeasesService+client.SnapshotService` | 15 | `0.47` (loose) |
| `iface.provider+iface.err` | 15 | `0.37` (loose) |
| `iptables.Nat+iptables.Rule` | 15 | `0.25` (loose) |
| `mounttypes.TypeBind+m.Spec` | 15 | `0.34` (loose) |
| `n.InvokeFunc+osSbox.InvokeFunc` | 15 | `0.34` (loose) |
| `ncfg.ContainerIfacePrefix+ncfg.DefaultBindingIP+ncfg.DefaultBridge` | 15 | `0.50` (loose) |
| `netlabel.ExposedPorts+netlabel.PortMap` | 15 | `0.33` (loose) |
| `netlabel.ExposedPorts+netlabel.PortMap+types.TransportPort` | 15 | `0.40` (loose) |
| `ordinal.Load+r.maxOrdinal` | 15 | `0.35` (loose) |
| `os.NewFile+sctp` | 15 | `0.43` (loose) |
| `parentLink.Attrs+netlink.LinkAttrs` | 15 | `0.47` (loose) |
| `r.FormValue+output.Flushed` | 15 | `0.35` (loose) |
| `reference.IsNameOnly+reference.WithTag` | 15 | `0.36` (loose) |
| `ts.h+ts.tHash` | 15 | `0.58` (settled) |
| `txn.OnCommit+conf.IsValueSet` | 15 | `0.34` (loose) |
| `Runtimes.Default+cfg.Runtimes` | 14 | `0.54` (settled) |
| `cmd.Command+icmd.Cmd` | 14 | `0.34` (loose) |
| `config.RegistryService+endpoint.URL` | 14 | `0.34` (loose) |
| `ctr.MountLabel+ctr.ImagePlatform` | 14 | `0.44` (loose) |
| `daemon+v3` | 14 | `0.52` (settled) |
| `daemon.RawSysInfo+daemon.*Daemon.RawSysInfo` | 14 | `0.44` (loose) |
| `dockerversion.DockerUserAge…+http.Header` | 14 | `0.42` (loose) |
| `drvregistry.*IPAMs.Register…+ipamapi` | 14 | `0.42` (loose) |
| `layerStore.Get+rootFS.ChainID` | 14 | `0.34` (loose) |
| `list+pools` | 14 | `0.54` (settled) |
| `na.networks+n.ID` | 14 | `0.53` (settled) |
| `na.services+vip.Addr` | 14 | `0.48` (loose) |
| `net.SplitHostPort+net.ParseIP` | 14 | `0.44` (loose) |
| `nlh.LinkAdd+netlink.Bridge` | 14 | `0.34` (loose) |
| `otelhttp.NewTransport+http.Transport` | 14 | `0.40` (loose) |
| `plugingetter.CompatPlugin+plugins` | 14 | `0.41` (loose) |
| `strconv.ParseInt+fmt.Errorf` | 14 | `0.44` (loose) |
| `t.Task+remote.wrapError` | 14 | `0.58` (settled) |
| `v.value+v.deleting` | 14 | `0.39` (loose) |
| `Config.Domainname+Config.Hostname` | 13 | `0.42` (loose) |
| `Config.NetworkDisabled+ctr.Config` | 13 | `0.38` (loose) |
| `bufio.NewReader+bufio` | 13 | `0.50` (settled) |
| `daemon.runAsHyperVContainer+daemon.*Daemon.runAsHyperVC…` | 13 | `0.48` (loose) |
| `gogotypes.DurationProto+fmt.Errorf` | 13 | `0.37` (loose) |
| `h.fromsvc+h.tosvc` | 13 | `0.44` (loose) |
| `h.head+h.unselected` | 13 | `0.46` (loose) |
| `info.cg2Controllers+info.Warnings` | 13 | `0.50` (loose) |
| `n.IsBridge+n.IsDefault` | 13 | `0.67` (settled) |
| `n.Meta+s.Meta` | 13 | `0.39` (loose) |
| `net.IPAddr+sctp.SCTPAddr` | 13 | `0.37` (loose) |
| `netlink.SCOPE_UNIVERSE+nlHandle.RouteAdd` | 13 | `0.34` (loose) |
| `netlink.XFRM_MODE_TRANSPORT+netlink.XFRM_PROTO_ESP` | 13 | `0.34` (loose) |
| `network.NetworkHost+network.NetworkNone` | 13 | `0.39` (loose) |
| `once.Do+unix` | 13 | `0.43` (loose) |
| `out.WriteProgress+remotes.MakeRefKey` | 13 | `0.29` (loose) |
| `p.GetTypes+v2.*Plugin.GetTypes` | 13 | `0.30` (loose) |
| `path.Split+fmt.Errorf` | 13 | `0.40` (loose) |
| `pool.Addr+pool.Bits` | 13 | `0.33` (loose) |
| `r.RegisterDriver+scope.Global` | 13 | `0.62` (settled) |
| `s.config+s.mu` | 13 | `0.43` (loose) |
| `sb.makeHostsRecs+config.hostsPath` | 13 | `0.33` (loose) |
| `sctp.ListenSCTP+net.ListenTCP+net.ListenUDP` | 13 | `0.33` (loose) |
| `w.Opt+mod` | 13 | `0.83` (unanimous) |
| `Created.Unix+Config.Labels` | 12 | `0.53` (settled) |
| `Subnet.Addr+k.AddressSpace` | 12 | `0.21` (loose) |
| `addr.Mask+d.advertiseAddress` | 12 | `0.34` (loose) |
| `b.HostPort+b.Port` | 12 | `0.24` (loose) |
| `binding.HostIP+HostConfig.PortBindings` | 12 | `0.35` (loose) |
| `c.stdin+c.stdinPipe` | 12 | `0.33` (loose) |
| `config.Config4+config.Config6` | 12 | `0.45` (loose) |
| `config.IpvlanFlag+config.IpvlanMode` | 12 | `0.42` (loose) |
| `config.Mirrors+registry.IndexInfo` | 12 | `0.45` (loose) |
| `ctr.MountLabel+ctr.ImageID` | 12 | `0.43` (loose) |
| `io.NopCloser+bytes` | 12 | `0.39` (loose) |
| `mounttypes.TypeImage+m.Spec` | 12 | `0.41` (loose) |
| `nDB.networkNodes+v2` | 12 | `0.59` (settled) |
| `net.Conn+net` | 12 | `0.41` (loose) |
| `nl.manager+nl.ns` | 12 | `0.98` (unanimous) |
| `nlwrap.NewHandleAt+syscall.NETLINK_ROUTE` | 12 | `0.23` (loose) |
| `resolvconf.Parse+bytes.NewBuffer` | 12 | `0.25` (loose) |
| `tlsconfig.Client+tlsconfig.Options` | 12 | `0.34` (loose) |
| `types+network` | 12 | `0.47` (loose) |
| `v.Components+types.ComponentVersion` | 12 | `0.32` (loose) |
| `Err.Error+e.Err` | 11 | `0.67` (settled) |
| `b.HostPort+b.Port+a.IP` | 11 | `0.23` (loose) |
| `builder.commit+d.state` | 11 | `0.56` (settled) |
| `c.IPVersion+iptable.Raw` | 11 | `0.31` (loose) |
| `epi.dstName+epi.routes+epi.v4PoolID` | 11 | `0.42` (loose) |
| `filepath.Join+shimopts` | 11 | `0.43` (loose) |
| `http.Request+sha512` | 11 | `0.31` (loose) |
| `ipamutils.NetworkToSplit.Ov…+ipbits` | 11 | `0.39` (loose) |
| `lcs.authConfig+challenge` | 11 | `0.49` (loose) |
| `overlayutils.VXLANUDPPort+ns.NlHandle` | 11 | `0.30` (loose) |
| `r.cancelPull+r.checkClosed` | 11 | `0.39` (loose) |
| `s.FinishedAt+s.Pid` | 11 | `0.22` (loose) |
| `t.Method+t.NumMethod` | 11 | `0.34` (loose) |
| `IPAM.Config+network.IPAM` | 10 | `0.35` (loose) |
| `IPNet.String+nlwrap.AddrList` | 10 | `0.26` (loose) |
| `a.eMount+a.scopePath` | 10 | `0.81` (unanimous) |
| `client.SnapshotService+i.client` | 10 | `0.48` (loose) |
| `containersReplica.ReserveNa…+cerrdefs.IsConflict+daemon.containersReplica` | 10 | `0.28` (loose) |
| `d.create+opts.StorageOpt` | 10 | `0.42` (loose) |
| `daemon.logClusterEvent+Annotations.Name` | 10 | `0.37` (loose) |
| `identity.NewID+identity` | 10 | `0.40` (loose) |
| `ipbits.Add+bitmap.*Bitmap.Bits` | 10 | `0.28` (loose) |
| `nDB.RLock+nDB.RUnlock` | 10 | `0.31` (loose) |
| `net.HardwareAddr+fmt.Errorf` | 10 | `0.33` (loose) |
| `netip.AddrFromSlice+portmapper.*PortMapper.Unmap` | 10 | `0.25` (loose) |
| `os.IsPathSeparator+unsafe` | 10 | `0.31` (loose) |
| `portallocator.Get+portallocator` | 10 | `0.39` (loose) |
| `rm.hijacked+rm.rw` | 10 | `0.34` (loose) |
| `s.Meta+s.Endpoint` | 10 | `0.35` (loose) |
| `scanner.Err+scanner.Text` | 10 | `0.38` (loose) |
| `sha256.New+sha256` | 10 | `0.39` (loose) |
| `strings.SplitN+fmt.Errorf` | 10 | `0.39` (loose) |
| `table.Family+table.Name` | 10 | `0.68` (settled) |
| `windows.UTF16PtrFromString+windows` | 10 | `0.48` (loose) |
| `BridgeConfig.Iface+conf.BridgeConfig` | 9 | `0.40` (loose) |
| `C.__u64+errno.Error` | 9 | `0.50` (loose) |
| `Timestamp.UnixNano+msg.Timestamp` | 9 | `0.26` (loose) |
| `TmpfsOptions.Mode+TmpfsOptions.SizeBytes` | 9 | `0.56` (settled) |
| `bnd.HostIP+bnd.IP+defHostIP.To4` | 9 | `0.43` (loose) |
| `d.deleteNetwork+Pool.String` | 9 | `0.31` (loose) |
| `daemon.execCommands+term` | 9 | `0.31` (loose) |
| `epi.dstName+epi.routes` | 9 | `0.43` (loose) |
| `errhttp+versions` | 9 | `0.55` (settled) |
| `http.StatusNoContent+w.WriteHeader` | 9 | `0.46` (loose) |
| `info.ContainerImageName+info.ContainerImageID` | 9 | `0.48` (loose) |
| `jinfo.AddStaticRoute+iNames.SetNames` | 9 | `0.26` (loose) |
| `o.cfg+initlayer` | 9 | `0.68` (settled) |
| `pp.Call+plugins.WithRequestTimeout` | 9 | `0.75` (settled) |
| `runConfig.Cmd+container.Config` | 9 | `0.46` (loose) |
| `runtime.SetFinalizer+j.j` | 9 | `0.18` (loose) |
| `Meta.UpdatedAt+Version.Index` | 8 | `0.38` (loose) |
| `State.Health+c.State` | 8 | `0.33` (loose) |
| `ast+parser` | 8 | `0.43` (loose) |
| `backend.GetNetworks+backend.NetworkListConfig+cluster.GetNetworks` | 8 | `0.28` (loose) |
| `baggage.ContextWithBaggage+otelutil.MustNewBaggage` | 8 | `0.37` (loose) |
| `config.MacvlanMode+config.CreatedSlaveLink` | 8 | `0.52` (settled) |
| `container.StreamConfig+jsonfilelog` | 8 | `0.81` (unanimous) |
| `containerimage+exptypes` | 8 | `0.52` (settled) |
| `ctr.MountLabel+ctr.ImageID+ctr.ImagePlatform` | 8 | `0.45` (loose) |
| `discoverapi.NodeDiscoveryDa…+discoverapi.NodeDiscovery` | 8 | `0.26` (loose) |
| `errdefs.InvalidParameter+remotecontext.withDockerfil…` | 8 | `0.41` (loose) |
| `ev.Actor+filter.ExactMatch` | 8 | `0.43` (loose) |
| `expvar+pprof` | 8 | `0.55` (settled) |
| `fmt.Fprint+aec` | 8 | `0.43` (loose) |
| `h.cursor+h.b` | 8 | `0.83` (unanimous) |
| `i.nlh+i.Link` | 8 | `0.36` (loose) |
| `r.initRoutes+router` | 8 | `0.83` (unanimous) |
| `r.re+lazyregexp.*Regexp.re` | 8 | `0.80` (unanimous) |
| `runtime.GOARCH+types` | 8 | `0.35` (loose) |
| `slices.DeleteFunc+nlwrap` | 8 | `0.34` (loose) |
| `store.DeleteObject+store.PutObjectAtomic` | 8 | `0.72` (settled) |
| `store.PutObjectAtomic+datastore.Key` | 8 | `0.72` (settled) |
| `t.Proto+t.Port` | 8 | `0.54` (settled) |
| `types.GetIPNetCopy+net.IPNet` | 8 | `0.43` (loose) |
| `unix.EINTR+errors.Is` | 8 | `0.53` (settled) |
| `BindOptions.ReadOnlyNonRecu…+BindOptions.NonRecursive` | 7 | `0.31` (loose) |
| `Logger.SetOutput+L.Logger` | 7 | `0.31` (loose) |
| `Privileges.AppArmor+Privileges.CredentialSpec+Privileges.NoNewPrivileges` | 7 | `0.34` (loose) |
| `ef.fuzzyMatchName+events.*Filter.fuzzyMatchNa…` | 7 | `0.94` (unanimous) |
| `mountinfo.Mounted+mountinfo` | 7 | `0.44` (loose) |
| `netController.Networks+daemon.netController` | 7 | `0.34` (loose) |
| `ocispec.ImageConfig+ocispec.MediaTypeImageConfig` | 7 | `0.42` (loose) |
| `osl.createNetworkNamespace+osl.createNamespaceFile` | 7 | `0.55` (settled) |
| `p.OSVersion+p.Variant` | 7 | `0.37` (loose) |
| `p.s+labels` | 7 | `0.93` (unanimous) |
| `r.RegisterDriver+scope.Local` | 7 | `0.66` (settled) |
| `r.l+l.Name` | 7 | `0.35` (loose) |
| `server.Serve+http.NewServeMux` | 7 | `0.85` (unanimous) |
| `slices.Clone+slices` | 7 | `0.36` (loose) |
| `span.RecordError+codes.Error` | 7 | `0.33` (loose) |
| `ws.WriteFile+fm.ws+resolvconf.*ResolvConf.Writ…` | 7 | `0.44` (loose) |
| `y.hi+y.lo` | 7 | `0.51` (settled) |
| `archive.toArchiveOpt+tarheader` | 6 | `0.77` (unanimous) |
| `c.SandboxByID+sbs.dbExists` | 6 | `0.44` (loose) |
| `config.Routed+config.Unprotected` | 6 | `0.50` (settled) |
| `container.NetworkMode+network` | 6 | `0.28` (loose) |
| `errors.New+filepath.Join` | 6 | `0.67` (settled) |
| `eventtypes.Message+pubsub` | 6 | `0.30` (loose) |
| `format+unicode` | 6 | `0.56` (settled) |
| `httputils.ArchiveFormValues+v.Path` | 6 | `0.20` (loose) |
| `info.cgMounts+info.Warnings` | 6 | `0.35` (loose) |
| `ld.digest+ocischema` | 6 | `0.53` (settled) |
| `log.FatalLevel+log.PanicLevel` | 6 | `0.29` (loose) |
| `mountinfo.GetMounts+mountinfo` | 6 | `0.30` (loose) |
| `platforms.Normalize+platforms` | 6 | `0.30` (loose) |
| `sliceutil.Dedup+sliceutil` | 6 | `0.33` (loose) |
| `txn.Get+iter.Next` | 6 | `0.40` (loose) |
| `unix.Close+unix` | 6 | `0.38` (loose) |
| `v.s+v.v` | 6 | `0.57` (settled) |
| `ws.WriteFile+fm.ws` | 6 | `0.46` (loose) |
| `PLogMetaData.ID+PLogMetaData.Ordinal` | 5 | `0.24` (loose) |
| `Spec.GetNetwork+ic.Range` | 5 | `0.19` (loose) |
| `aSpace.allocated+netiputil.PrefixCompare` | 5 | `0.16` (loose) |
| `b.HostPort+b.HostIP` | 5 | `0.27` (loose) |
| `b.allowedBuildArgs+sort` | 5 | `0.24` (loose) |
| `b.lastRead+b.pos` | 5 | `0.46` (loose) |
| `c.NetworkList+network.ListOptions` | 5 | `0.43` (loose) |
| `cli.sendRequest+client.*Client.sendRequest` | 5 | `0.72` (settled) |
| `e.protectedElements+t.Helper` | 5 | `0.89` (unanimous) |
| `endpoint.srcName+netutils.GenerateIfaceName` | 5 | `0.34` (loose) |
| `ep.getSandbox+ep.ID` | 5 | `0.50` (settled) |
| `graphdriver.Register+fstype` | 5 | `0.87` (unanimous) |
| `i.AddressSpace+i.AuxAddresses` | 5 | `0.30` (loose) |
| `mfst.Layers+schema1` | 5 | `0.32` (loose) |
| `na.services+s.Meta` | 5 | `0.34` (loose) |
| `ncfg.ContainerIfacePrefix+ncfg.DefaultBindingIP` | 5 | `0.43` (loose) |
| `p.Values+csv` | 5 | `0.56` (settled) |
| `plugin+plugingetter` | 5 | `0.38` (loose) |
| `req.AddressSpace+req.Pool` | 5 | `0.28` (loose) |
| `resources.BlkioDeviceReadBps+resources.BlkioDeviceReadIO…` | 5 | `0.25` (loose) |
| `router.NewRoute+httputils` | 5 | `0.87` (unanimous) |
| `sa.InheritHandle+sa.Length` | 5 | `0.42` (loose) |
| `task.ID+c.task` | 5 | `0.47` (loose) |
| `version+useragent` | 5 | `0.28` (loose) |
| `EventsService.Log+daemon.EventsService` | 4 | — |
| `File.GID+File.UID` | 4 | — |
| `archive.ToArchiveOpt+archive` | 4 | — |
| `b.Port+b.HostPort` | 4 | — |
| `cerrdefs.IsInvalidArgument+daemon.GetContainer` | 4 | — |
| `config.getOriginResolvConfP…+rc.WriteFile` | 4 | — |
| `d.getNetworks+nw.config` | 4 | — |
| `dc.con+dc.wg` | 4 | — |
| `e.byID+e.mu` | 4 | — |
| `ep.Delete+context.WithoutCancel` | 4 | — |
| `fmt.Println+filepath` | 4 | — |
| `ip.To4+net` | 4 | — |
| `layerStore.GetRWLayer+layerStore.ReleaseRWLayer` | 4 | — |
| `n.Options+n.Driver` | 4 | — |
| `options.Details+options.ShowStderr` | 4 | — |
| `rand.Read+rand` | 4 | — |
| `strings.Count+fmt.Errorf` | 4 | — |
| `testutil.RunCommand+icmd.Success` | 4 | — |
| `unix.Kill+errors.Is` | 4 | — |
| `versions.compare+strconv` | 4 | — |
| `Driver.Name+Driver.Options` | 3 | — |
| `IP.Equal+sctp` | 3 | — |
| `config.MetaHeaders+config.AuthConfig` | 3 | — |
| `defaultipam.DriverName+defaultipam` | 3 | — |
| `imageStore.Children+referenceStore.References` | 3 | — |
| `n.EndpointByName+libnetwork.*Network.Endpoin…` | 3 | — |
| `os.Hostname+opts` | 3 | — |
| `p.HostPortEnd+p.HostIP` | 3 | — |
| `v.backend+v.cluster` | 3 | — |
| `daemon.PluginStore+daemon.netController` | 2 | — |
| `Annotations.Labels+Driver.Name` | 1 | — |

Convention is how uniformly this corpus realizes a concept: `1.00` means every function carrying the tag does it the same way, and a low number means the tag covers several unrelated habits. A concept with fewer than five members is not modeled.

### Where the duplication is

Merge-worthy pairs are folded up to their packages: only pairs doppel judges worth consolidating are counted. An edge means two packages keep solving the same problem separately; a count on a node means the repetition is inside one package. Weights are **merge-worthy pairs**.

```mermaid
flowchart LR
    p0["ipvlan<br/>27 internal"]
    p1["macvlan<br/>22 internal"]
    p0 ---|"104"| p1
    p2["container<br/>386 internal"]
    p3["swarm<br/>127 internal"]
    p2 ---|"44"| p3
    p4["image<br/>43 internal"]
    p2 ---|"40"| p4
    p5["plugin<br/>50 internal"]
    p2 ---|"29"| p5
    p6["brmanager<br/>7 internal"]
    p7["cnmallocator<br/>16 internal"]
    p6 ---|"27"| p7
    p8["diagnostic<br/>5 internal"]
    p9["libnetwork<br/>294 internal"]
    p8 ---|"27"| p9
    p10["ivmanager<br/>3 internal"]
    p6 ---|"25"| p10
    p11["main<br/>36 internal"]
    p9 ---|"25"| p11
    p12["mvmanager"]
    p6 ---|"24"| p12
    p4 ---|"24"| p3
    p7 ---|"23"| p10
    p7 ---|"22"| p12
```

_410 further package pairs are connected by duplication and are not drawn._

### How settled each package is

A package with at least five functions gets a habitat model: doppel learns what is normal there and measures how surprising each member is against it. **Norm** is how uniform the package's practice is. A **misfit** is a function alien to its package *and* to the wider subsystem around it — one that fits its neighbours a directory up is normal for this codebase and is not reported.

```mermaid
flowchart TD
    h0["suite<br/>7 functions · norm 0.58<br/>3 misfits"]
    h1["vfs<br/>27 functions · norm 0.59"]
    h2["boltdb<br/>8 functions · norm 0.63<br/>3 misfits"]
    h3["testutils<br/>33 functions · norm 0.66<br/>2 misfits"]
    h4["null<br/>23 functions · norm 0.69"]
    h5["lazyregexp<br/>13 functions · norm 0.69"]
    h6["oci<br/>11 functions · norm 0.70<br/>4 misfits"]
    h7["errdefs<br/>55 functions · norm 0.72<br/>16 misfits"]
    h8["progress<br/>13 functions · norm 0.73"]
    h9["syscall-test<br/>9 functions · norm 0.73"]
    h10["brmanager<br/>16 functions · norm 0.73<br/>5 misfits"]
    h11["ivmanager<br/>16 functions · norm 0.73<br/>5 misfits"]
    classDef good fill:#d7ecd9,color:#1b3d20
    classDef warn fill:#fbeecb,color:#4a3a12
    classDef hot fill:#f7d6d6,color:#4a1c1c
    class h0,h1,h2,h3,h4,h5,h6,h7,h8,h9,h10,h11 warn
```

_155 further packages are modeled and not drawn._ Most uniform is `checker` (norm `0.97`); most varied is `suite` (norm `0.58`). 71 functions are alien to their package and to the subsystem around it. A further 94 fit poorly in their package but match the wider subsystem, so they are not reported.

### How these candidates were found

Three channels propose candidates independently — shared rare *structure*, shared *concepts*, shared *calls* — and their union is what gets compared. This run: **40225 candidate pairs** (shape 9451, concept 23375, call 12470), of which 22% arrived on call evidence alone and 51% on concept evidence alone. A pair sharing none of the three is never compared, however alike it looks.

The concept signal on each compared pair is read three ways — what the taxonomy asserts, what this corpus's frequencies say, and what the two sides' learned vocabularies share with no tree in between. On **2893 of 38415** pairs the taxonomy and the vocabularies differ by at least 0.50: 372 where the vocabularies agree and the tree cannot see it, 2521 where the tree asserts a kinship the vocabularies lack. Each such pair carries a `concept views` line saying which.

Each function is also an arena where its candidate concepts compete for its evidence. 7325 functions reached an equilibrium: **4541** settled on a single concept, **2781** on a coalition, **0** hold concepts this corpus says do not go together.

### Corpus metrics

**Compression ratio:** `8.59`x — this corpus's canonical function bodies contain **570200 AST nodes** in total, which hash-cons (two nodes count as the same subtree exactly when their kind and every child match, all the way down) to **66390 distinct subtree shapes**; the ratio is nodes divided by shapes, always >= 1.0, and it never feeds any score.

**Nearest-neighbour code-shape:** of **7658 functions**, **7429** had a code-shape neighbour among the pairs retrieval actually scored — their best score's p50/p90/p99 are `0.45` / `1.00` / `1.00`, and 76% of them (5654 of 7429) already clear this run's threshold of `0.36`. This is **not an exhaustive nearest-neighbour search** (that would be a full pairwise comparison); it is bounded by the same three retrieval channels the pair list itself is bounded by, so the other 229 functions are excluded here as having no *scored* neighbour, not asserted to have none at all.

---

## Local practice

The vocabulary above says what a concept *is*. This says what one looks like when **this** codebase writes it — learned from the corpus, so it describes the house style rather than a rule from anywhere else.

### How this codebase writes each concept

Only what is **distinctive**. A feature earns a row by being carried by this concept's members at least twice as often as by the corpus at large — nearly every Go function has a `return` and an `if`, so prevalence alone would describe the language rather than this codebase. Weights are how much a channel counts toward whether a member looks normal — calls 40, control flow 20, co-occurring tags 15, role 15, package 10.

**`Config.OpenStdin+Config.StdinOnce`** — 688 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `sdjournal.noCopy.Unlock` | `███████···` | 458 of 688 | 7.7× |
| flow ×20 | `defer` | `█████·····` | 363 of 688 | 3.9× |

**`C.__u32+C.int`** — 636 functions

Nothing distinctive: its members do what the rest of the corpus does. The tag groups them; a shared way of writing them does not.

**`Isolation.IsValid+PluginObj.PluginReference`** — 437 functions

Nothing distinctive: its members do what the rest of the corpus does. The tag groups them; a shared way of writing them does not.

**`Store.validateName+bytes.TrimSpace`** — 407 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `path/filepath.Join` | `███·······` | 116 of 407 | 9.1× |
| flow ×20 | `defer` | `███·······` | 122 of 407 | 2.2× |
| cotags ×15 | `C.__u32+C.int` | `███·······` | 134 of 407 | 4.0× |

**`Task.Runtime+c.callWithRetry`** — 309 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `client.ensureReaderClosed` | `███·······` | 86 of 309 | 23× |
|  | `encoding/json.Marshal` | `███·······` | 80 of 309 | 18× |
| flow ×20 | `defer` | `███·······` | 88 of 309 | 2.1× |
| package ×10 | `client` | `████······` | 114 of 309 | 12× |

**`c.cache+client.PruneInfo`** — 251 functions

Nothing distinctive: its members do what the rest of the corpus does. The tag groups them; a shared way of writing them does not.

_482 further concepts are modeled and not described._

### Which concepts share a function

`++` at least four times chance, `+` at least twice, `−` at most half, `never` not once. A blank cell is ordinary company — near chance, which is not culture.

_Showing 12 of 519 concepts — those in the strongest pairings, taken strongest first by lift weighted by how many functions it speaks for. Every cell between them is shown; the other 507 concepts are not on the grid._

| | `C.__u32+C.int` | `Config.OpenStdin+Config.StdinOnce` | `Isolation.IsValid+PluginObj.PluginReference` | `config.MetaHeaders+config.AuthConfig+p.repoName` | `container.StreamConfig+container.Config` | `img.RawJSON+img.OS` | `n.addrSpace+n.skipGwAllocIPv4` | `n.hasSpecialDriver+n.ipamType` | `n.skipGwAllocIPv4+n.skipGwAllocIPv6+n.inDelete` | `nat.Port+fmt.Sprintf` | `options.size+driver.options` |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **`Config.OpenStdin+Config.StdinOnce`** |  | | | | | | | | | | |
| **`Isolation.IsValid+PluginObj.PluginReference`** |  | − | | | | | | | | | |
| **`config.MetaHeaders+config.AuthConfig+p.repoName`** | never | never | never | | | | | | | | |
| **`container.StreamConfig+container.Config`** | never | − | never |  | | | | | | | |
| **`img.RawJSON+img.OS`** |  | never | never |  |  | | | | | | |
| **`n.addrSpace+n.skipGwAllocIPv4`** | never |  | never |  |  |  | | | | | |
| **`n.hasSpecialDriver+n.ipamType`** | never | + | never |  |  |  | ++ | | | | |
| **`n.skipGwAllocIPv4+n.skipGwAllocIPv6+n.inDelete`** | never | − | never |  |  |  | ++ | ++ | | | |
| **`nat.Port+fmt.Sprintf`** | never | never | never |  |  |  |  |  |  | | |
| **`options.size+driver.options`** |  | never |  |  |  |  |  |  |  |  | |
| **`platforms.Parse+ocispec.Platform`** |  | never |  |  |  |  |  |  |  |  |  |

### What travels with what

Co-occurrence measured against chance across every function. Only relationships at least twice — or at most half — as common as chance are reported; near-chance company is not culture. Each kind is listed separately, because there are far more call tokens than concepts and one shared list is all calls. Within a kind, strongest first means lift weighted by how many functions carry it — a 100× relationship holding for three functions is a weaker finding than a 30× one holding for thirty.

**Together more than chance — tag~tag**

- 27 of 28 `daemonCfg.Rootless+specs.Linux` functions also `daemonCfg.Rootless+specs.Linux+HostConfig.Privileged` — 246× chance
- 41 of 44 `cgroups.Mode+cgroups.Unified` functions also `daemon.UsingSystemd+initlayer` — 135× chance
- 47 of 55 `types.NotImplementedErrorf+bitmap` functions also `types.NotImplementedErrorf+bitmap+overlayutils` — 111× chance
- 23 of 24 `Reference.String+p.platform` functions also `p.platform+p.desc` — 306× chance
- 19 of 20 `hcsshim.DeactivateLayer+hcsshim.UnprepareLayer` functions also `hcsshim.GetLayerMountPath+d.getLayerChain` — 346× chance
- 17 of 17 `dns.TypeAAAA+dns.A` functions also `dns.TypeAAAA+dns.TypePTR` — 403× chance
- _1042 more not listed_

**Together more than chance — tag~role**

- 51 of 62 `types.MediaTypeMultiplexedS…+types.MediaTypeRawStream+httputils.DecodePlatform` functions also `orchestrator` — 3.4× chance
- 48 of 61 `httputils.BoolValueOrDefault+types.MediaTypeMultiplexedS…` functions also `orchestrator` — 3.3× chance
- 34 of 39 `options.Signal+options.Timeout` functions also `orchestrator` — 3.6× chance
- 55 of 74 `types.MediaTypeMultiplexedS…+types.MediaTypeRawStream` functions also `orchestrator` — 3.1× chance
- 5 of 10 `s.Meta+s.Endpoint` functions also `passthrough` — 9.3× chance
- 21 of 27 `httputils.BoolValue+ir.backend` functions also `orchestrator` — 3.2× chance
- _143 more not listed_

**Together more than chance — tag~call**

- 20 of 25 `tracing.Attribute+span.SetAttributes` functions also `github.com/containerd/containerd/v2/pkg/tracing.StartSpan` — 245× chance
- 15 of 15 `client.TaskList+inspect.State` functions also `gotest.tools/v3/poll.Continue` — 403× chance
- 14 of 15 `client.TaskList+inspect.State` functions also `gotest.tools/v3/poll.Success` — 397× chance
- 12 of 12 `nl.manager+nl.ns` functions also `daemon.withDefaultNamespace` — 511× chance
- 13 of 20 `builder.Stderr+c.ShellDependantCmdLine` functions also `dockerfile.*Builder.commit` — 356× chance
- 9 of 9 `C.__u64+errno.Error` functions also `golang.org/x/sys/unix.Syscall` — 766× chance
- _1524 more not listed_

**Apart more than chance — tag~tag**

- **no** `Config.OpenStdin+Config.StdinOnce` function has `img.RawJSON+img.OS` — chance alone would give about 13 of 688
- **no** `Config.OpenStdin+Config.StdinOnce` function has `platforms.Parse+ocispec.Platform` — chance alone would give about 12 of 688
- **no** `C.__u32+C.int` function has `container.StreamConfig+container.Config` — chance alone would give about 9 of 636
- **no** `C.__u32+C.int` function has `n.skipGwAllocIPv4+n.skipGwAllocIPv6+n.inDelete` — chance alone would give about 8 of 636
- **no** `Isolation.IsValid+PluginObj.PluginReference` function has `img.RawJSON+img.OS` — chance alone would give about 8 of 437
- **no** `C.__u32+C.int` function has `n.addrSpace+n.skipGwAllocIPv4` — chance alone would give about 8 of 636
- _287 more not listed_

**Apart more than chance — tag~role**

- **no** `types.NotImplementedErrorf+bitmap+overlayutils` function has `orchestrator` — chance alone would give about 14 of 59
- **no** `types.NotImplementedErrorf+bitmap` function has `orchestrator` — chance alone would give about 13 of 55
- **no** `globalLock.RLock+globalLock.RUnlock` function has `orchestrator` — chance alone would give about 9 of 36
- **no** `types.NotImplementedErrorf+bitmap+overlayutils` function has `utility` — chance alone would give about 8 of 59
- **no** `parentLink.Attrs+netlink.LinkAttrs` function has `leaf` — chance alone would give about 8 of 15
- **no** `errcode.ErrorCodeDenied+f.err` function has `orchestrator` — chance alone would give about 8 of 34
- _233 more not listed_

**Apart more than chance — tag~call**

- **no** `Config.OpenStdin+Config.StdinOnce` function has `client.*Client.post` — chance alone would give about 4 of 688
- **no** `C.__u32+C.int` function has `types.InvalidParameterErrorf` — chance alone would give about 4 of 636
- **no** `Config.OpenStdin+Config.StdinOnce` function has `github.com/distribution/reference.FamiliarString` — chance alone would give about 4 of 688
- **no** `Config.OpenStdin+Config.StdinOnce` function has `versions.LessThan` — chance alone would give about 4 of 688
- **no** `C.__u32+C.int` function has `client.*Client.post` — chance alone would give about 4 of 636
- **no** `Config.OpenStdin+Config.StdinOnce` function has `authorization.*responseModifier.WriteHeader` — chance alone would give about 4 of 688
- _16 more not listed_

### Functions drifting from their own concept

These carry a tag but look nothing like the other functions carrying it. Typicality is measured against the concept's own median, so a genuinely varied concept lowers its own bar and a tight one can flag nobody.

| Function | Concept | Typicality | Concept median | |
|---|---|---:|---:|---|
| `service.volumeToAPIType` <br/>`volume/service/convert.go:83` | `time.RFC3339+time.Now` | `0.14` | `0.73` | no near-duplicate |
| `container.*View.transform` <br/>`container/view.go:297` | `IPAMConfig.IPv4Address+IPAMConfig.IPv6Address` | `0.18` | `0.77` | no near-duplicate |
| `plugin.parseHeaders` <br/>`api/server/router/plugin/plugin_routes.go:20` | `registry.DecodeAuthConfig+registry.AuthHeader` | `0.09` | `0.65` | no near-duplicate |
| `main.installConfigFlags` <br/>`cmd/dockerd/config_windows.go:9` | `BridgeConfig.Iface+conf.BridgeConfig` | `0.04` | `0.56` | no near-duplicate |
| `httpstatus.FromError` <br/>`api/server/httpstatus/status.go:16` | `http.StatusConflict+http.StatusNotImplemented` | `0.04` | `0.55` | no near-duplicate |
| `httputils.ParseForm` <br/>`api/server/httputils/httputils.go:100` | `registry.DecodeAuthConfig+registry.AuthHeader` | `0.14` | `0.65` | no near-duplicate |
| `containerd.translateRegistryError` <br/>`daemon/containerd/registry_errors.go:16` | `http.StatusConflict+http.StatusNotImplemented` | `0.06` | `0.55` | no near-duplicate |
| `environment.restoreDefaultBridge` <br/>`testutil/environment/protect_others.go:16` | `network.NetworkHost+network.NetworkNone` | `0.07` | `0.55` | no near-duplicate |
| `plugins.IsNotFound` <br/>`pkg/plugins/errors.go:20` | `http.StatusConflict+http.StatusNotImplemented` | `0.09` | `0.55` | no near-duplicate |
| `syslog.parseFacility` <br/>`daemon/logger/syslog/syslog.go:219` | `errors.Errorf+fluent` | `0.15` | `0.60` | no near-duplicate |

_892 more unusual realizations not listed._

A row marked _no near-duplicate_ appears in no reported pair: nothing else in this report explains it, which makes it drift rather than duplication.

---

## Match #1 — Code-shape: `0.9131`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `libnetwork/networkdb/networkdbdiagnostic.go:128` | `networkdb.*NetworkDB.dbCreateEntry` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.67 |
| **B** | `libnetwork/networkdb/networkdbdiagnostic.go:177` | `networkdb.*NetworkDB.dbUpdateEntry` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.67 |

**Explain:** differs by one extra assign, two extra call, two extra ident

**Profile A:** `ctr.terminateInvoked+diagnostic.TableObj` 1.00 (dominance)

**Profile B:** `ctr.terminateInvoked+diagnostic.TableObj` 1.00 (dominance)

**Code similarity:** `wl 0.86  flow 1.00  nesting 1.00  sig 1.00  size 0.99`

**Containment:** `0.93`

**Evidence:** `1160.17` (shape 1112.94, concept 2.83, call 44.39)

**Trophic:** `0.94`

**Shared structure:**

- `25.66` — `depth-3 ASSIGN` ×4
- `25.66` — `depth-2 ASSIGN` ×4
- `24.51` — `depth-3 BIN` ×4

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `call:loggertest.makeTestMessages`, `call:loggertest.readAll`, `call:overlayutils.NeedsUserXAttr`

**Structural overlap:** `0.79` (merge-worthy)

- share 19 callees: [DecodeString, Error, String, WithFields, caller.Name, context.TODO, diagnostic.CommandSucceed, diagnostic.DebugHTTPForm, diagnostic.FailCommand, diagnostic.HTTPReply, diagnostic.ParseHTTPFormOptions, diagnostic.WrongCommand, fmt.Sprintf, len, log.G, logger.Error, logger.Info, logger.WithError, r.ParseForm]
- overlapping call-graph neighborhoods (0.91): 29 shared
- share patterns: [ctr.terminateInvoked+diagnostic.TableObj]
- both are orchestrator functions
- same package
- callees do related work (1.00): [log.FatalLevel+log.PanicLevel, c.newNS+unix.Gettid, stats+bufio, fmt.Fprint+os.Exit, fmt.Fprintln+fmt.Fprintf, c.Name+log.Fields, ctr.terminateInvoked+diagnostic.TableObj, Config.OpenStdin+Config.StdinOnce]
- same visibility
- same receiver type: NetworkDB
- call into same packages: [caller, diagnostic, networkdb]

---

## Match #2 — Code-shape: `0.9660`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `contrib/syscall-test/ns.c:31` | `syscall-test.main` | `(?, ?)` | — |
| **B** | `contrib/syscall-test/userns.c:31` | `syscall-test.main` | `(?, ?)` | — |

**Explain:** differs by one extra binary, one extra ident

**Code similarity:** `wl 0.94  flow 1.00  nesting 1.00  sig 1.00  size 0.98`

**Containment:** `0.98`

**Evidence:** `1165.38` (shape 1158.52, concept 0.00, call 6.86)

**Trophic:** `0.99`

**Shared structure:**

- `28.01` — `depth-3 CALL` ×4
- `28.01` — `depth-3 EXPRSTMT` ×4
- `28.01` — `depth-2 CALL` ×4

**Concept views:** shape `0.00`, corpus `0.00`, feature `0.00`, a-in-b `0.00`, b-in-a `0.00`

**Structural overlap:** `0.63` (merge-worthy)

- share 6 callees: [clone, exit, fprintf, mmap, strerror, waitpid]
- overlapping call-graph neighborhoods (1.00): 5 shared
- both are leaf functions
- same package
- callees do related work (1.00): [Store.validateName+bytes.TrimSpace, C.__u32+C.int]
- same visibility
- same receiver type: plain functions
- call into same packages: [git]

---

## Match #3 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `libnetwork/drivers/ipvlan/ipvlan_store.go:254` | `ipvlan.*endpoint.UnmarshalJSON` | `([]byte) (error)` | Task.Runtime+c.callWithRetry 0.66 |
| **B** | `libnetwork/drivers/macvlan/macvlan_store.go:248` | `macvlan.*endpoint.UnmarshalJSON` | `([]byte) (error)` | Task.Runtime+c.callWithRetry 0.66 |

**Kind:** interface implementations — both implement `UnmarshalJSON([]byte) (error)` on `*endpoint` and `*endpoint`, sibling packages `ipvlan` and `macvlan`

**Explain:** identical after rename, commutative-reorder

**Profile A:** `Task.Runtime+c.callWithRetry` 1.00 (dominance)

**Profile B:** `Task.Runtime+c.callWithRetry` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `690.18` (shape 669.16, concept 2.60, call 18.42)

**Trophic:** `1.00`

**Shared structure:**

- `21.68` — `depth-3 BLOCK` ×3
- `21.68` — `depth-3 RETURN` ×3
- `21.68` — `depth-3 CALL` ×3

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `call:metadata.*v2MetadataService.diffIDKey`, `call:metadata.*v2MetadataService.diffIDNamespace`, `call:plugins.*Client.callWithRetry`

**Structural overlap:** `0.72` (merge-worthy)

- share 5 callees: [fmt.Errorf, json.Unmarshal, net.ParseMAC, types.InternalErrorf, types.ParseCIDR]
- overlapping call-graph neighborhoods (1.00): 26 shared
- share patterns: [Task.Runtime+c.callWithRetry]
- both are orchestrator functions
- callees do related work (1.00): [nw.IPAM+Driver.Options, p.HostPortEnd+p.HostIP+p.HostPort, time.RFC3339+time.Now, ipam.RequestAddress+ipamapi.ErrNoAvailableIPs, http.StatusConflict+http.StatusNotImplemented, C.__u32+C.int]
- same visibility
- same receiver type: endpoint
- call into same packages: [types]

---

## Match #4 — Code-shape: `0.9030`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `libnetwork/drivers/ipvlan/ipvlan_network.go:140` | `ipvlan.*driver.DeleteNetwork` | `(string) (error)` | d.deleteNetwork+Pool.String 0.50 |
| **B** | `libnetwork/drivers/macvlan/macvlan_network.go:155` | `macvlan.*driver.DeleteNetwork` | `(string) (error)` | d.deleteNetwork+Pool.String 0.50 |

**Kind:** interface implementations — both implement `DeleteNetwork(string) (error)` on `*driver` and `*driver`, sibling packages `ipvlan` and `macvlan`

**Explain:** differs by two extra assign, one extra if, two extra binary, and 4 more kinds

**Profile A:** `parentLink.Attrs+netlink.LinkAttrs` 0.40, `parentLink.Attrs+netlink.Bridge` 0.33, `nlh.LinkAdd+parentLink.Attrs` 0.27 (coalition)

**Profile B:** `parentLink.Attrs+netlink.LinkAttrs` 0.40, `parentLink.Attrs+netlink.Bridge` 0.33, `nlh.LinkAdd+parentLink.Attrs` 0.27 (coalition)

**Code similarity:** `wl 0.84  flow 1.00  nesting 0.96  sig 1.00  size 1.00`

**Containment:** `0.92`

**Evidence:** `871.95` (shape 857.96, concept 3.63, call 10.37)

**Trophic:** `1.00`

**Shared structure:**

- `15.84` — `depth-3 EXPRSTMT` ×2
- `15.84` — `depth-3 CALL` ×2
- `15.84` — `depth-2 CALL` ×2

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `sel:d.deleteNetwork`, `lit:ipv4`, `lit:0.0.0.0/0`

**Structural overlap:** `0.69` (merge-worthy)

- share 16 callees: [Debugf, LinkByName, LinkDel, Warnf, WithError, context.TODO, d.deleteNetwork, d.network, d.storeDelete, delDummyLink, delVlanLink, fmt.Errorf, getDummyName, log.G, ns.NlHandle, parentExists]
- overlapping call-graph neighborhoods (0.92): 60 shared
- share patterns: [d.deleteNetwork+Pool.String]
- both are orchestrator functions
- callees do related work (0.82): [path.Split+fmt.Errorf, parentLink.Attrs+netlink.LinkAttrs, nlh.LinkAdd+parentLink.Attrs, nlHandle.LinkByName+n.nlHandle, parentLink.Attrs+netlink.Bridge, n.path+netns.GetFromPath, Healthcheck.Retries+c.callWithRetry]
- same visibility
- same receiver type: driver
- call into same packages: [nlwrap, ns]

---

## Match #5 — Code-shape: `0.9134`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `libnetwork/networkdb/networkdbdiagnostic.go:308` | `networkdb.*NetworkDB.dbJoinNetwork` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.69 |
| **B** | `libnetwork/networkdb/networkdbdiagnostic.go:340` | `networkdb.*NetworkDB.dbLeaveNetwork` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.69 |

**Kind:** mirror operations — `dbJoinNetwork` and `dbLeaveNetwork` on `*NetworkDB` are one operation run in opposite directions

**Explain:** differs by two extra call

**Profile A:** `ctr.terminateInvoked+diagnostic.TableObj` 1.00 (dominance)

**Profile B:** `ctr.terminateInvoked+diagnostic.TableObj` 1.00 (dominance)

**Code similarity:** `wl 0.86  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `0.92`

**Evidence:** `717.51` (shape 670.19, concept 2.92, call 44.39)

**Trophic:** `0.99`

**Shared structure:**

- `18.14` — `depth-1 EXPRSTMT` ×3
- `17.34` — `depth-0 CALL` ×3
- `11.81` — `depth-3 CALL` ×2

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `call:loggertest.makeTestMessages`, `call:loggertest.readAll`, `call:overlayutils.NeedsUserXAttr`

**Structural overlap:** `0.79` (merge-worthy)

- share 18 callees: [Error, String, WithFields, caller.Name, context.TODO, diagnostic.CommandSucceed, diagnostic.DebugHTTPForm, diagnostic.FailCommand, diagnostic.HTTPReply, diagnostic.ParseHTTPFormOptions, diagnostic.WrongCommand, fmt.Sprintf, len, log.G, logger.Error, logger.Info, logger.WithError, r.ParseForm]
- overlapping call-graph neighborhoods (0.84): 27 shared
- share patterns: [ctr.terminateInvoked+diagnostic.TableObj]
- both are orchestrator functions
- same package
- callees do related work (0.99): [log.FatalLevel+log.PanicLevel, c.newNS+unix.Gettid, stats+bufio, fmt.Fprint+os.Exit, fmt.Fprintln+fmt.Fprintf, c.Name+log.Fields, ctr.terminateInvoked+diagnostic.TableObj, Config.OpenStdin+Config.StdinOnce]
- same visibility
- same receiver type: NetworkDB
- call into same packages: [caller, diagnostic, networkdb]

---

## Match #6 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `libnetwork/drivers/ipvlan/ipvlan_setup.go:96` | `ipvlan.createVlanLink` | `(string) (error)` | parentLink.Attrs+netlink.Bridge 0.58, nlh.LinkAdd+parentLink.Attrs 0.52, parentLink.Attrs+netlink.LinkAttrs 0.50 |
| **B** | `libnetwork/drivers/macvlan/macvlan_setup.go:76` | `macvlan.createVlanLink` | `(string) (error)` | parentLink.Attrs+netlink.Bridge 0.58, nlh.LinkAdd+parentLink.Attrs 0.52, parentLink.Attrs+netlink.LinkAttrs 0.50 |

**Explain:** identical after rename, commutative-reorder

**Profile A:** `parentLink.Attrs+netlink.LinkAttrs` 0.42, `parentLink.Attrs+netlink.Bridge` 0.38, `nlh.LinkAdd+parentLink.Attrs` 0.21 (coalition)

**Profile B:** `parentLink.Attrs+netlink.LinkAttrs` 0.42, `parentLink.Attrs+netlink.Bridge` 0.38, `nlh.LinkAdd+parentLink.Attrs` 0.21 (coalition)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `600.51` (shape 574.01, concept 10.66, call 15.84)

**Trophic:** `1.00`

**Shared structure:**

- `16.18` — `depth-3 SEL` ×3
- `16.18` — `depth-2 SEL` ×3
- `16.18` — `depth-1 SEL` ×3

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `sel:parentLink.Attrs`, `sel:netlink.Bridge`, `sel:nlh.LinkAdd`

**Structural overlap:** `0.76` (merge-worthy)

- share 11 callees: [Debugf, LinkAdd, LinkByName, LinkSetUp, context.TODO, fmt.Errorf, log.G, ns.NlHandle, parentLink.Attrs, parseVlan, strings.Contains]
- overlapping call-graph neighborhoods (0.96): 73 shared
- share patterns: [nlh.LinkAdd+parentLink.Attrs, parentLink.Attrs+netlink.Bridge, parentLink.Attrs+netlink.LinkAttrs]
- both are orchestrator functions
- callers do related work (0.94): [d.getNetworks+nw.config]
- callees do related work (1.00): [containerimage+exptypes, nlHandle.LinkByName+n.nlHandle, n.path+netns.GetFromPath, nlh.AddrAdd+netlink.FAMILY_V4, Healthcheck.Retries+c.callWithRetry, img.RawJSON+img.OS, c.cache+client.PruneInfo]
- same visibility
- same receiver type: plain functions
- call into same packages: [mobyexporter, nlwrap, ns]

---

## Match #7 — Code-shape: `0.7909`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `libnetwork/drivers/ipvlan/ipvlan_joinleave.go:33` | `ipvlan.*driver.Join` | `(context.Context, string, string, string, driverapi.JoinInfo, map[string]interface{}, map[string]interface{}) (error)` | jinfo.AddStaticRoute+iNames.SetNames 0.51, endpoint.srcName+netutils.GenerateIfaceName 0.50 |
| **B** | `libnetwork/drivers/macvlan/macvlan_joinleave.go:21` | `macvlan.*driver.Join` | `(context.Context, string, string, string, driverapi.JoinInfo, map[string]interface{}, map[string]interface{}) (error)` | jinfo.AddStaticRoute+iNames.SetNames 0.52, endpoint.srcName+netutils.GenerateIfaceName 0.52 |

**Kind:** interface implementations — both implement `Join(context.Context, string, string, string, driverapi.JoinInfo, map[string]interface{}, map[string]interface{}) (error)` on `*driver` and `*driver`, sibling packages `ipvlan` and `macvlan`

**Explain:** differs by six extra if, four extra assign, four extra return, and 9 more kinds

**Profile A:** `jinfo.AddStaticRoute+iNames.SetNames` 0.68, `endpoint.srcName+netutils.GenerateIfaceName` 0.32 (dominance)

**Profile B:** `jinfo.AddStaticRoute+iNames.SetNames` 0.68, `endpoint.srcName+netutils.GenerateIfaceName` 0.32 (dominance)

**Code similarity:** `wl 0.66  flow 1.00  nesting 0.85  sig 1.00  size 0.76`

**Containment:** `0.92` — most of the smaller body's shape is inside the larger

**Evidence:** `1380.62` (shape 1315.06, concept 7.60, call 57.96)

**Trophic:** `0.87`

**Shared structure:**

- `21.74` — `depth-3 CALL` ×4
- `21.42` — `depth-3 BIN` ×4
- `20.47` — `depth-3 IF` ×4

**Concept views:** shape `1.00`, corpus `0.96`, feature `0.97`, a-in-b `1.00`, b-in-a `0.97`

**Shared vocabulary:** `sel:jinfo.AddStaticRoute`, `call:netutils.GenerateIfaceName`, `sel:endpoint.srcName`

**Structural overlap:** `0.68` (merge-worthy)

- share 26 callees: [Debugf, Start, String, attribute.String, d.getNetwork, d.storeUpdate, fmt.Errorf, iNames.SetNames, jinfo.DisableGatewayService, jinfo.InterfaceName, jinfo.SetGateway, jinfo.SetGatewayIPv6, len, log.G, n.endpoint, n.getSubnetforIPv4, n.getSubnetforIPv6, net.ParseCIDR, netlabel.GetIfname, netutils.GenerateIfaceName, ns.NlHandle, otel.Tracer, span.End, trace.WithAttributes, v4gw.String, v6gw.String]
- overlapping call-graph neighborhoods (0.98): 134 shared
- share patterns: [endpoint.srcName+netutils.GenerateIfaceName, jinfo.AddStaticRoute+iNames.SetNames]
- both are orchestrator functions
- callees do related work (1.00): [endpoint.srcName+netutils.GenerateIfaceName, epi.dstName+epi.routes, epi.dstName+epi.routes+epi.v4PoolID, parentLink.Attrs+netlink.LinkAttrs, link.Attrs+n.nlHandle, nlh.LinkAdd+parentLink.Attrs, parentLink.Attrs+netlink.Bridge, Config.OpenStdin+Config.StdinOnce]
- same visibility
- same receiver type: driver
- call into same packages: [libnetwork, netlabel, netutils, ns, tailfile]

---

## Match #8 — Code-shape: `0.8769`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:464` | `dbclient.doWriteKeys` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.63 |
| **B** | `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:497` | `dbclient.doDeleteKeys` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.64 |

**Explain:** differs by two extra call, one extra index, one extra literal, and 1 more kind

**Profile A:** `Config.OpenStdin+Config.StdinOnce` 1.00 (dominance)

**Profile B:** `Config.OpenStdin+Config.StdinOnce` 1.00 (dominance)

**Code similarity:** `wl 0.79  flow 1.00  nesting 1.00  sig 1.00  size 0.99`

**Containment:** `0.89`

**Evidence:** `636.75` (shape 603.18, concept 1.97, call 31.60)

**Trophic:** `0.94`

**Shared structure:**

- `13.07` — `depth-3 CALL` ×2
- `13.07` — `depth-2 CALL` ×2
- `13.07` — `depth-1 CALL` ×2

**Concept views:** shape `1.00`, corpus `0.98`, feature `0.98`, a-in-b `1.00`, b-in-a `0.98`

**Shared vocabulary:** `call:bridge.*bridgeNetwork.releasePorts`, `call:client.*Client.negotiateAPIVersionPing`, `call:container.*controller.matchevent`

**Structural overlap:** `0.93` (merge-worthy)

- share 14 callees: [Infof, cancel, checkTable, clientWatchTable, close, context.Background, context.TODO, context.WithTimeout, fmt.Fprintf, log.G, make, strconv.Atoi, strconv.Itoa, waitWriters]
- share 1 callers: [dbclient.Client]
- overlapping call-graph neighborhoods (0.97): 73 shared
- share patterns: [Config.OpenStdin+Config.StdinOnce]
- both are orchestrator functions
- same package
- callees do related work (0.99): [strconv.FormatBool+strconv.Itoa, Healthcheck.Retries+c.callWithRetry, strconv.FormatInt+strconv.Itoa, Config.OpenStdin+Config.StdinOnce]
- same visibility
- same receiver type: plain functions
- called from same packages: [dbclient]
- call into same packages: [container, dbclient]

---

## Match #9 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `integration/plugin/logging/cmd/dummy/main.go:9` | `main.main` | `()` | Store.validateName+bytes.TrimSpace 0.53, ctr.terminateInvoked+diagnostic.TableObj 0.51, server.Serve+http.NewServeMux 0.50 |
| **B** | `integration/plugin/volumes/cmd/dummy/main.go:9` | `main.main` | `()` | Store.validateName+bytes.TrimSpace 0.53, ctr.terminateInvoked+diagnostic.TableObj 0.51, server.Serve+http.NewServeMux 0.50 |

**Explain:** identical after rename, commutative-reorder

**Profile A:** `server.Serve+http.NewServeMux` 1.00 (dominance)

**Profile B:** `server.Serve+http.NewServeMux` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `487.43` (shape 319.02, concept 7.90, call 160.52)

**Trophic:** `1.00`

**Shared structure:**

- `7.92` — `depth-3 BLOCK`
- `7.92` — `depth-2 BLOCK`
- `7.51` — `depth-3 COMPOSITE`

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `lit:/run/docker/plugins/plugin.sock`, `sel:server.Serve`, `call:net.Listen`

**Structural overlap:** `0.81` (merge-worthy)

- share 6 callees: [String, http.NewServeMux, l.Addr, net.Listen, panic, server.Serve]
- overlapping call-graph neighborhoods (1.00): 146 shared
- share patterns: [Store.validateName+bytes.TrimSpace, ctr.terminateInvoked+diagnostic.TableObj, server.Serve+http.NewServeMux]
- both are orchestrator functions
- same package
- callees do related work (1.00): [log.FatalLevel+log.PanicLevel, format+unicode, rm.hijacked+rm.rw, os.NewFile+sctp, fmt.Fprint+os.Exit, resp.Status+http.NewRequestWithContext, fmt.Fprintln+fmt.Fprintf, Config.OpenStdin+Config.StdinOnce]
- same visibility
- same receiver type: plain functions
- call into same packages: [authorization, dbclient, dbserver, diagnostic, main, reexec, trap, v2]

---

## Match #10 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `libnetwork/drivers/ipvlan/ipvlan_endpoint.go:62` | `ipvlan.*driver.DeleteEndpoint` | `(string, string) (error)` | Isolation.IsValid+PluginObj.PluginReference 0.57 |
| **B** | `libnetwork/drivers/macvlan/macvlan_endpoint.go:67` | `macvlan.*driver.DeleteEndpoint` | `(string, string) (error)` | Isolation.IsValid+PluginObj.PluginReference 0.56 |

**Kind:** interface implementations — both implement `DeleteEndpoint(string, string) (error)` on `*driver` and `*driver`, sibling packages `ipvlan` and `macvlan`

**Explain:** identical after rename, commutative-reorder

**Profile A:** `parentLink.Attrs+netlink.LinkAttrs` 0.40, `parentLink.Attrs+netlink.Bridge` 0.33, `nlh.LinkAdd+parentLink.Attrs` 0.27 (coalition)

**Profile B:** `parentLink.Attrs+netlink.LinkAttrs` 0.40, `parentLink.Attrs+netlink.Bridge` 0.33, `nlh.LinkAdd+parentLink.Attrs` 0.27 (coalition)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `553.52` (shape 541.13, concept 2.02, call 10.37)

**Trophic:** `1.00`

**Shared structure:**

- `10.79` — `depth-3 SEL` ×2
- `10.79` — `depth-2 SEL` ×2
- `10.79` — `depth-1 SEL` ×2

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `call:bitmap.*Bitmap.validateOrdinal`, `call:client.*Client.ContainerList`, `call:client.*Client.ImageList`

**Structural overlap:** `0.71` (merge-worthy)

- share 13 callees: [LinkByName, LinkDel, Warnf, WithError, context.TODO, d.network, d.storeDelete, fmt.Errorf, log.G, n.deleteEndpoint, n.endpoint, ns.NlHandle, validateID]
- overlapping call-graph neighborhoods (0.97): 59 shared
- share patterns: [Isolation.IsValid+PluginObj.PluginReference]
- both are orchestrator functions
- callees do related work (1.00): [d.deleteNetwork+Pool.String, addr.Mask+d.advertiseAddress, netlabel.ExposedPorts+netlabel.PortMap, nlHandle.LinkByName+n.nlHandle, n.path+netns.GetFromPath, nlh.AddrAdd+netlink.FAMILY_V4, Healthcheck.Retries+c.callWithRetry]
- same visibility
- same receiver type: driver
- call into same packages: [nlwrap, ns]

---

## Families

866 families, 2209 functions in a family, largest 52 members; 6581 edges scored here that retrieval never proposed

### Family 1 — 11 members, every pair `>= 0.47` code-shape, evidence `27605`  (9 edges scored here)

_Not drawn: 11 members is 55 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `libnetwork/cmd/networkdb-test/dummyclient/dummyClient.go:58` | `dummyclient.watchTableEntries` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.58 |
| `libnetwork/networkdb/networkdbdiagnostic.go:38` | `networkdb.*NetworkDB.dbJoin` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.68 |
| `libnetwork/networkdb/networkdbdiagnostic.go:71` | `networkdb.*NetworkDB.dbPeers` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.67 |
| `libnetwork/networkdb/networkdbdiagnostic.go:128` | `networkdb.*NetworkDB.dbCreateEntry` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.67 |
| `libnetwork/networkdb/networkdbdiagnostic.go:177` | `networkdb.*NetworkDB.dbUpdateEntry` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.67 |
| `libnetwork/networkdb/networkdbdiagnostic.go:225` | `networkdb.*NetworkDB.dbDeleteEntry` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.70 |
| `libnetwork/networkdb/networkdbdiagnostic.go:262` | `networkdb.*NetworkDB.dbGetEntry` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.68 |
| `libnetwork/networkdb/networkdbdiagnostic.go:308` | `networkdb.*NetworkDB.dbJoinNetwork` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.69 |
| `libnetwork/networkdb/networkdbdiagnostic.go:340` | `networkdb.*NetworkDB.dbLeaveNetwork` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.69 |
| `libnetwork/networkdb/networkdbdiagnostic.go:372` | `networkdb.*NetworkDB.dbGetTable` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.67 |

_1 more members not listed._

### Family 2 — 11 members, every pair `>= 0.45` code-shape, evidence `27560`  (10 edges scored here)

_Not drawn: 11 members is 55 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `libnetwork/diagnostic/server.go:182` | `diagnostic.stackTrace` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.62 |
| `libnetwork/networkdb/networkdbdiagnostic.go:38` | `networkdb.*NetworkDB.dbJoin` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.68 |
| `libnetwork/networkdb/networkdbdiagnostic.go:71` | `networkdb.*NetworkDB.dbPeers` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.67 |
| `libnetwork/networkdb/networkdbdiagnostic.go:128` | `networkdb.*NetworkDB.dbCreateEntry` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.67 |
| `libnetwork/networkdb/networkdbdiagnostic.go:177` | `networkdb.*NetworkDB.dbUpdateEntry` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.67 |
| `libnetwork/networkdb/networkdbdiagnostic.go:225` | `networkdb.*NetworkDB.dbDeleteEntry` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.70 |
| `libnetwork/networkdb/networkdbdiagnostic.go:262` | `networkdb.*NetworkDB.dbGetEntry` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.68 |
| `libnetwork/networkdb/networkdbdiagnostic.go:308` | `networkdb.*NetworkDB.dbJoinNetwork` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.69 |
| `libnetwork/networkdb/networkdbdiagnostic.go:340` | `networkdb.*NetworkDB.dbLeaveNetwork` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.69 |
| `libnetwork/networkdb/networkdbdiagnostic.go:372` | `networkdb.*NetworkDB.dbGetTable` | `(http.ResponseWriter, *http.Request)` | ctr.terminateInvoked+diagnostic.TableObj 0.67 |

_1 more members not listed._

### Family 3 — 11 members, every pair `>= 0.37` code-shape, evidence `15997`  (19 edges scored here)

_Not drawn: 11 members is 55 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:368` | `dbclient.doJoinNetwork` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.63 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:382` | `dbclient.doLeaveNetwork` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.64 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:396` | `dbclient.doNetworkPeers` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.58, Healthcheck.Retries+c.callWithRetry 0.47 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:464` | `dbclient.doWriteKeys` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.63 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:497` | `dbclient.doDeleteKeys` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.64 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:530` | `dbclient.doWriteDeleteUniqueKeys` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.63 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:567` | `dbclient.doWriteUniqueKeys` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.64 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:602` | `dbclient.doWriteDeleteLeaveJoin` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.64 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:631` | `dbclient.doWriteDeleteWaitLeaveJoin` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.64 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:677` | `dbclient.doWriteWaitLeave` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.65 |

_1 more members not listed._

### Family 4 — 20 members, every pair `>= 0.38` code-shape, evidence `14844`  (121 edges scored here)

_Not drawn: 20 members is 190 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `cmd/docker-proxy/main_linux.go:31` | `main.main` | `()` | fmt.Fprint+os.Exit 0.50, os.NewFile+sctp 0.46, ctr.terminateInvoked+diagnostic.TableObj 0.43 |
| `cmd/dockerd/docker.go:103` | `main.main` | `()` | ctr.terminateInvoked+diagnostic.TableObj 0.46, fmt.Fprint+os.Exit 0.42 |
| `contrib/apparmor/main.go:13` | `main.main` | `()` | Store.validateName+bytes.TrimSpace 0.58, ctr.terminateInvoked+diagnostic.TableObj 0.44 |
| `daemon/logger/awslogs/cloudwatchlogs.go:116` | `awslogs.init` | `()` | Isolation.IsValid+PluginObj.PluginReference 0.65, ctr.terminateInvoked+diagnostic.TableObj 0.59 |
| `daemon/logger/fluentd/fluentd.go:68` | `fluentd.init` | `()` | Isolation.IsValid+PluginObj.PluginReference 0.72, ctr.terminateInvoked+diagnostic.TableObj 0.67, errors.Errorf+fluent 0.58 |
| `daemon/logger/gcplogs/gcplogging.go:46` | `gcplogs.init` | `()` | Isolation.IsValid+PluginObj.PluginReference 0.71, ctr.terminateInvoked+diagnostic.TableObj 0.69 |
| `daemon/logger/gelf/gelf.go:28` | `gelf.init` | `()` | Isolation.IsValid+PluginObj.PluginReference 0.74, ctr.terminateInvoked+diagnostic.TableObj 0.67 |
| `daemon/logger/journald/journald.go:69` | `journald.init` | `()` | Isolation.IsValid+PluginObj.PluginReference 0.73, ctr.terminateInvoked+diagnostic.TableObj 0.68 |
| `daemon/logger/jsonfilelog/jsonfilelog.go:37` | `jsonfilelog.init` | `()` | Isolation.IsValid+PluginObj.PluginReference 0.73, ctr.terminateInvoked+diagnostic.TableObj 0.69 |
| `daemon/logger/local/local.go:55` | `local.init` | `()` | Isolation.IsValid+PluginObj.PluginReference 0.72, ctr.terminateInvoked+diagnostic.TableObj 0.67 |

_10 more members not listed._

### Family 5 — 13 members, every pair `>= 0.36` code-shape, evidence `14268`  (35 edges scored here)

_Not drawn: 13 members is 78 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:306` | `dbclient.doReady` | `([]string)` | Config.OpenStdin+Config.StdinOnce 0.64 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:320` | `dbclient.doJoin` | `([]string)` | Config.OpenStdin+Config.StdinOnce 0.57 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:336` | `dbclient.doClusterPeers` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.58, Healthcheck.Retries+c.callWithRetry 0.47 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:368` | `dbclient.doJoinNetwork` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.63 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:382` | `dbclient.doLeaveNetwork` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.64 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:396` | `dbclient.doNetworkPeers` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.58, Healthcheck.Retries+c.callWithRetry 0.47 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:464` | `dbclient.doWriteKeys` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.63 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:497` | `dbclient.doDeleteKeys` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.64 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:530` | `dbclient.doWriteDeleteUniqueKeys` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.63 |
| `libnetwork/cmd/networkdb-test/dbclient/ndbClient.go:567` | `dbclient.doWriteUniqueKeys` | `([]string, []string)` | Config.OpenStdin+Config.StdinOnce 0.64 |

_3 more members not listed._

_861 more families not listed._

_2 component(s) too large or too dense to enumerate (sizes 144, 1054); their families are not reported._


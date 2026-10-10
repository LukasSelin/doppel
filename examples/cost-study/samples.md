## cobra

- #17 cobra.*Command.InitDefaultHelpFlag ↔ cobra.*Command.InitDefaultVersionFlag (none, file; edits 2/1)
  - co-change 212ea4078: both edited alike (0.83): Include --help and --version flag in completion (#1813)
- #121 cobra.genBashComp ↔ cobra.writePreamble (thin wrappers, package; edits 14/4)
  - unpropagated-fix 4ba5566f5: fixed in cobra.genBashComp only; cobra.writePreamble still carries 1 of 1 replaced lines: fix(bash): nounset unbound file filter variable on empty extension (#2228)

## gin

- #2 gin.*Engine.Run ↔ gin.*Engine.RunTLS (none, file; edits 3/2)
  - co-change 93ff771e6: both edited alike (1.00): ci(sec): improve type safety and server organization in HTTP middleware (#4437)
- #8 gin.*Engine.RunListener ↔ gin.*Engine.RunTLS (none, file; edits 1/2)
  - co-change 93ff771e6: both edited alike (0.88): ci(sec): improve type safety and server organization in HTTP middleware (#4437)
- #10 gin.*Engine.RunTLS ↔ gin.*Engine.RunUnix (none, file; edits 2/1)
  - co-change 93ff771e6: both edited alike (0.88): ci(sec): improve type safety and server organization in HTTP middleware (#4437)
- #31 gin.*Engine.Run ↔ gin.*Engine.RunListener (none, file; edits 3/1)
  - co-change 93ff771e6: both edited alike (0.88): ci(sec): improve type safety and server organization in HTTP middleware (#4437)
- #32 gin.*Engine.RunListener ↔ gin.*Engine.RunUnix (none, file; edits 1/1)
  - co-change 93ff771e6: both edited alike (1.00): ci(sec): improve type safety and server organization in HTTP middleware (#4437)
- #35 gin.*Engine.Run ↔ gin.*Engine.RunUnix (none, file; edits 3/1)
  - co-change 93ff771e6: both edited alike (0.88): ci(sec): improve type safety and server organization in HTTP middleware (#4437)

## prometheus

- #1 kubernetes.NewEndpointSlice ↔ kubernetes.NewEndpoints (none, package; edits 5/5)
  - co-change b1c356bee: both edited alike (1.00): fix(discovery): Handle cache.DeletedFinalStateUnknown in node informers' DeleteFunc
  - co-change 228b94f6e: both edited alike (1.00): discovery/kubernetes: Support linked pod controllers
- #2 remote.*QueueManager.AppendFloatHistograms ↔ remote.*QueueManager.AppendHistograms (none, file; edits 3/3)
  - co-change 43c1535bd: both edited alike (0.88): fix(rw1): drop unsupported NHCB and log (#17146)
  - co-change 15e86825b: both edited alike (1.00): Add plumbing for ST histograms in RW
- #3 remote.*shards.sendSamplesWithBackoff ↔ remote.*shards.sendV2SamplesWithBackoff (none, file; edits 2/2)
  - co-change e1cb29bf8: both edited alike (1.00): create common struct and function to DRY
  - co-change 897ba10d1: both edited alike (1.00): remote write: fix sent_batch_duration_seconds measuring before the request is sent (#18214)
  - extracted e1cb29bf8: both moved code into new createBatchSpan: create common struct and function to DRY
- #5 storage.addFH ↔ storage.addH (none, file; edits 1/1)
  - co-change 1e77d9ded: both edited alike (1.00): storage/buffer.go: add ST to sample types and iterators
- #6 chunkenc.*FloatHistogramAppender.AppendFloatHistogram ↔ chunkenc.*HistogramAppender.AppendHistogram (none, package; edits 4/5)
  - co-change 8a9f4ff44: both edited alike (1.00): fix(tsdb): chunk overflow on ooo query (#18692)
  - co-change 2aefd00c2: both edited alike (0.78): tsdb/chunkenc: take a generic Appender as the prev parameter (#18857)
  - lagged-sync 7509e7ab1,bfe221c8e: chunkenc.*HistogramAppender.AppendHistogram changed first, chunkenc.*FloatHistogramAppender.AppendFloatHistogram caught up (1.00): "tsdb/chunkenc: make HistogramAppender.appendHistogram reusable" then "tsdb/chunkenc: make FloatHistogramAppender.appendFloatHistogram reusable"
  - lagged-sync 7d0aff14c,d46a510c6: chunkenc.*HistogramAppender.AppendHistogram changed first, chunkenc.*FloatHistogramAppender.AppendFloatHistogram caught up (0.78): "tsdb/chunkenc: return *HistogramAppender from recode" then "tsdb/chunkenc: return *FloatHistogramAppender from recode"
- #7 textparse.*OpenMetricsParser.Metric ↔ textparse.*PromParser.Metric (interface implementations, package; edits 1/1)
  - co-change 8bcb4d865: both edited alike (1.00): feat: normalize "le" and "quantile" labels values upon ingestion
- #8 remote.buildTimeSeries ↔ remote.buildV2TimeSeries (none, file; edits 1/1)
  - co-change 9e6a626da: both edited alike (1.00): create timeSeriesStats to reduce return variable
- #10 promql.funcHistogramStdDev ↔ promql.funcHistogramStdVar (none, file; edits 0/0)
  - extracted a51116461: both moved code into new histogramVariance: [REFACTOR] PromQL: DRY `stddev` and `stdvar` functions (#16451)
- #13 kubernetes.NewIngress ↔ kubernetes.NewService (none, package; edits 1/1)
  - co-change a9f6fdd91: both edited alike (1.00): feat(discovery/kubernetes): allow attaching namespace metadata to ingress and service roles.
- #19 moby.NewDiscovery ↔ moby.NewDockerDiscovery (none, package; edits 1/1)
  - co-change c14d5e2d6: both edited alike (1.00): discovery/moby: bound Docker API requests on all host schemes

## hugo

- #3 math.init ↔ strings.init (none, top; edits 2/1)
  - lagged-sync 325a0dba6,13a95b9c0: math.init changed first, strings.init caught up (0.50): "tpl/math: Add MaxInt64 function" then "tpl/strings: Add strings.ReplacePairs function"
- #6 cssjs.*postcssTransformation.Transform ↔ cssjs.*tailwindcssTransformation.Transform (interface implementations, package; edits 1/1)
  - unpropagated-fix a03a245f0: fixed in cssjs.*tailwindcssTransformation.Transform only; cssjs.*postcssTransformation.Transform still carries 1 of 1 replaced lines: Fix it so css.TailwindCSS inlineImports options isn't always enabled
- #7 goldmark.*hookedRenderer.renderImage ↔ goldmark.*hookedRenderer.renderLink (none, file; edits 3/3)
  - co-change f738669a4: both edited alike (1.00): Add Markdown render hooks for tables
  - co-change 588c9019c: both edited alike (1.00): deps: Upgrade github.com/yuin/goldmark v1.7.4 => v1.7.8
- #10 goldmark.*hookedRenderer.renderHeading ↔ goldmark.*hookedRenderer.renderLink (none, file; edits 4/3)
  - co-change f738669a4: both edited alike (1.00): Add Markdown render hooks for tables
  - co-change 588c9019c: both edited alike (1.00): deps: Upgrade github.com/yuin/goldmark v1.7.4 => v1.7.8
- #17 main.gofmt ↔ main.goimports (none, file; edits 1/1)
  - co-change a6bd67793: both edited alike (1.00): common/hexec: Remove github.com/cli/safeexec
- #22 goldmark.*hookedRenderer.renderHeading ↔ goldmark.*hookedRenderer.renderImage (none, file; edits 4/3)
  - co-change f738669a4: both edited alike (1.00): Add Markdown render hooks for tables
  - co-change 588c9019c: both edited alike (1.00): deps: Upgrade github.com/yuin/goldmark v1.7.4 => v1.7.8
- #38 babel.*babelTransformation.Transform ↔ cssjs.*postcssTransformation.Transform (interface implementations, top; edits 2/1)
  - co-change 9d66d513c: both edited alike (0.69): resources: Support babel/postcss config variants
- #98 codeblocks.*htmlRenderer.renderCodeBlock ↔ passthrough.*htmlRenderer.renderPassthroughBlock (none, top; edits 2/1)
  - co-change f738669a4: both edited alike (0.79): Add Markdown render hooks for tables
- #128 goldmark.*hookedRenderer.renderImageDefault ↔ goldmark.*hookedRenderer.renderLinkDefault (none, file; edits 2/2)
  - co-change 479fe6c65: both edited alike (0.67): Fix potential content XSS by escaping dangerous URLs in links and images
- #157 main.gofmt ↔ main.rewrite (none, file; edits 1/1)
  - co-change a6bd67793: both edited alike (1.00): common/hexec: Remove github.com/cli/safeexec

## moby

- #21 ipvlan.*driver.Join ↔ macvlan.*driver.Join (interface implementations, top; edits 3/2)
  - lagged-sync 17b863154,cd7240f6d: ipvlan.*driver.Join changed first, macvlan.*driver.Join caught up (1.00): "Enable DNS proxying for ipvlan-l3" then "Stop macvlan with no parent from using ext-dns"
  - lagged-sync d599cc584,390713607: macvlan.*driver.Join changed first, ipvlan.*driver.Join caught up (1.00): "Allow macvlan containers with no address" then "Allow ipvlan containers with no address"
- #42 ipvlan.*driver.initStore ↔ macvlan.*driver.initStore (interface implementations, top; edits 1/1)
  - co-change d21d0884a: both edited alike (0.92): libnetwork: share a single datastore with drivers
- #43 bridge.*driver.initStore ↔ ipvlan.*driver.initStore (interface implementations, top; edits 1/1)
  - co-change d21d0884a: both edited alike (0.92): libnetwork: share a single datastore with drivers
- #44 bridge.*driver.initStore ↔ macvlan.*driver.initStore (interface implementations, top; edits 1/1)
  - co-change d21d0884a: both edited alike (0.92): libnetwork: share a single datastore with drivers
- #59 ipvlan.*driver.CreateNetwork ↔ macvlan.*driver.CreateNetwork (interface implementations, top; edits 4/4)
  - lagged-sync 17b863154,cd7240f6d: ipvlan.*driver.CreateNetwork changed first, macvlan.*driver.CreateNetwork caught up (1.00): "Enable DNS proxying for ipvlan-l3" then "Stop macvlan with no parent from using ext-dns"
  - lagged-sync 8b13cde27,2f60d15dd: ipvlan.*driver.CreateNetwork changed first, macvlan.*driver.CreateNetwork caught up (1.00): "L3 and internal ipvlans don't need a gateway address" then "Internal macvlan networks don't need a gateway address."
  - lagged-sync a7a5de676,660e8118a: macvlan.*driver.CreateNetwork changed first, ipvlan.*driver.CreateNetwork caught up (1.00): "Allow no-IPv4 on a macvlan network" then "Allow no-IPv4 on an ipvlan network"
- #76 mounts.*linuxParser.validateMountConfigImpl ↔ mounts.*windowsParser.validateMountConfigReg (none, package; edits 4/1)
  - co-change bfb810445: both edited alike (0.92): volumes: Implement subpath mount
- #104 libnetwork.*Endpoint.addServiceInfoToCluster ↔ libnetwork.*Endpoint.deleteServiceInfoFromCluster (none, file; edits 1/1)
  - co-change 3bb13c7eb: both edited alike (0.86): libnet: Use Endpoint.dnsNames to create DNS records
- #121 client.*Client.NetworksPrune ↔ client.*Client.VolumesPrune (none, package; edits 1/1)
  - co-change 7faaa3afa: both edited alike (0.60): client: explicitly return zero-type on failures in prune functions
- #122 client.*Client.ContainersPrune ↔ client.*Client.VolumesPrune (none, package; edits 1/1)
  - co-change 7faaa3afa: both edited alike (0.60): client: explicitly return zero-type on failures in prune functions
- #123 client.*Client.ImagesPrune ↔ client.*Client.NetworksPrune (none, package; edits 1/1)
  - co-change 7faaa3afa: both edited alike (0.60): client: explicitly return zero-type on failures in prune functions


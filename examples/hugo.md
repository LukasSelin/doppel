# hugo

static site generator; a large monolith with heavy template and resource subsystems

**What this rung shows:** habitats — many packages large enough to have a temperature of their own

| | |
|---|---|
| Corpus | [hugo](https://github.com/gohugoio/hugo) |
| Pinned at | `v0.165.0` (`76a5e1880ab46688155b02e99bab9be2a6134492`) |
| Project since | 2013 |
| doppel | `103ec8f` |
| Command | `doppel analyze . --tests exclude --moves=false --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`; CI regenerates on every push to master.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Learning concept vocabulary...
Lexicon: 547 concepts (9 seeded, 538 emergent), 14869/41767 features above 2110 df, 358 functions unlabeled
Generating concept documents...
Calibration: rate 0.01 over 20000 null pairs -> threshold 0.34, struct-min 0.31, family-min 0.34
Found 5737 functions. Retrieving candidates...
Retrieval: shape 7055, concept 20730, call 8634 -> 32251 unique pairs
  concept-only 56.0%  call-only 18.8%  suppressed-shape functions: 19  large identity buckets: 0  surviving labels: 24438
  344 cross test/prod pairs dropped
Running structural comparison on 31907 pairs...
  Concept views: 2250 of 31907 compared pairs disagree with the taxonomy (161 vocabulary the tree misses, 2089 kinship the vocabularies lack)
  13593 pairs remain after struct-min=0.31 filter
Culture: 502 concepts modeled, 2587 associations, 1127 unusual realizations
Habitats: 129 modeled, 416 misfits (63 excused by subsystem), 33 subsystems; most uniform dartsass (norm 0.98), most diverse page (norm 0.66)
Conventions: strongest Symbol.toPrimitive+prim.call (1.00), loosest d.ExpiryDate+d.PublishDate (0.14)
Ecosystems: 5546 profiled (3227 dominance, 2312 coalition, 0 conflict, 7 weak)
Families: 515 over 615 components, 1894 functions in a family, 6390 edges completed
  3 component(s) skipped as too large or too dense: sizes [139 145 642]
```

# Code Similarity Report

**Functions analyzed:** 5737 | **Threshold:** 0.34 | **Pairs found:** 10

---

## What doppel sees

**5737 functions** across **167 packages** — test functions excluded. Structural roles: 3745 leaf, 904 orchestrator, 266 passthrough, 822 utility.

### Concepts

Two pictures of the same vocabulary. The first is what doppel **searched with**: an authored tree of fourteen seed practices, each leaf showing how many functions here ended up in a concept that seed grew. It is the same shape on every corpus, which is what makes it the one concept picture two runs can be compared on. The second is what this corpus **turned out to have**: concepts learned from the code itself, named after the evidence that identified them, hung from that same interior — so two functions under one *branch* score partial credit rather than nothing. Counts are members; membership is graded, and a function can carry several.

**What doppel looked for, and how much of it grew here.**

```mermaid
flowchart LR
    s0(["concept"])
    s1(["io_operation"])
    s2(["remote_io"])
    s3["http_call<br/>88"]
    s4["grpc_call<br/>absent"]
    s5(["data_store_access"])
    s6["db_access<br/>absent"]
    s7["caching<br/>242"]
    s8["transaction<br/>absent"]
    s9["file_io<br/>153"]
    s10["logging<br/>126"]
    s11(["data_transformation"])
    s12["mapping<br/>207"]
    s13["validation<br/>204"]
    s14["serialization<br/>50"]
    s15(["control_flow"])
    s16["concurrency<br/>210"]
    s17(["fault_tolerance"])
    s18["retry<br/>absent"]
    s19["circuit_breaker<br/>absent"]
    s20(["error_handling"])
    s21["error_wrapping<br/>256"]
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
    class s4,s6,s8,s18,s19 hot
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
    c8["Body.Close+api.OnLoadArgs<br/>153"]
    c9["C.BaseURL+conf.C<br/>233"]
    c10["Config.Sorted+c.C<br/>203"]
    c11["Config.Sorted+c.C+c.Languages<br/>108"]
    c12["Converters.GetHighlighter+c.applyTransformations<br/>207"]
    c13["ResourceSpec.FileCaches+c.isExpired<br/>242"]
    c14["c.getNestedKeyAndMap+c.root<br/>37"]
    c15["c.hasBeenInitialized+call.done<br/>210"]
    c16["fn.IsValid+htmltemplate.Must<br/>204"]
    c17["fw.Close+helpers.OpenFilesForWriting<br/>129"]
    c18["h.allSites+Conf.Watching<br/>33"]
    c19["http+io<br/>88"]
    c20["init.data+init.gitInfo<br/>162"]
    c21["m.Version+m.Owner<br/>95"]
    c22["mnt.init+typ.In<br/>256"]
    c23["pageOutput.pco+ps.pageOutput<br/>144"]
    c0 --> c1
    c1 --> c2
    c1 --> c3
    c0 --> c4
    c0 --> c5
    c5 --> c6
    c0 --> c7
    c1 --> c8
    c1 --> c9
    c3 --> c10
    c3 --> c11
    c4 --> c12
    c3 --> c13
    c5 --> c14
    c5 --> c15
    c4 --> c16
    c1 --> c17
    c5 --> c18
    c2 --> c19
    c7 --> c20
    c4 --> c21
    c7 --> c22
    c7 --> c23
```

The diagram draws the 3 largest concepts on each branch; **531 further concepts** are left out of the picture and listed in the table below.

**No practice here for** `circuit_breaker`, `db_access`, `grpc_call`, `retry`, `transaction`. Concepts are learned from this corpus, so one can never be absent — it exists because functions carry it. These are the *seeds* the search started from that grew nothing: a direct answer to "does this codebase already do X".

| Concept | Functions | Convention |
|---|---:|---|
| `mnt.init+typ.In` | 256 | `0.61` (settled) |
| `ResourceSpec.FileCaches+c.isExpired` | 242 | `0.65` (settled) |
| `C.BaseURL+conf.C` | 233 | `0.71` (settled) |
| `c.hasBeenInitialized+call.done` | 210 | `0.60` (settled) |
| `Converters.GetHighlighter+c.applyTransformations` | 207 | `0.65` (settled) |
| `fn.IsValid+htmltemplate.Must` | 204 | `0.55` (settled) |
| `Config.Sorted+c.C` | 203 | `0.78` (unanimous) |
| `init.data+init.gitInfo` | 162 | `0.62` (settled) |
| `Body.Close+api.OnLoadArgs` | 153 | `0.59` (settled) |
| `pageOutput.pco+ps.pageOutput` | 144 | `0.66` (settled) |
| `fw.Close+helpers.OpenFilesForWriting` | 129 | `0.62` (settled) |
| `Config.Sorted+c.copyStatic` | 126 | `0.56` (settled) |
| `pageConfigSource.Content+p.wrapError` | 120 | `0.59` (settled) |
| `cast.ToStringSlice+strings.ToLower` | 118 | `0.51` (settled) |
| `s.insertTemplate+opts.PathParser` | 116 | `0.57` (settled) |
| `Config.Sorted+c.C+c.Languages` | 108 | `0.59` (settled) |
| `c.pi+c.pco` | 106 | `0.63` (settled) |
| `d.Name+fs` | 103 | `0.62` (settled) |
| `m.Version+m.Owner` | 95 | `0.57` (settled) |
| `console.log+this.console` | 93 | `0.55` (settled) |
| `hugo.CurrentVersion+v.String` | 92 | `0.60` (settled) |
| `text.Position+text` | 91 | `0.65` (settled) |
| `Build.Render+pagemeta.Never` | 89 | `0.62` (settled) |
| `p.Path+hugofs.FileMetaInfo` | 89 | `0.58` (settled) |
| `vv.noBaseOf+s.treeShortcodes` | 89 | `0.55` (settled) |
| `http+io` | 88 | `0.61` (settled) |
| `output.Formats+f.Name` | 83 | `0.62` (settled) |
| `c.serverPorts+c.serverInterface` | 82 | `0.55` (settled) |
| `h.PathSpec+PathSpec.BaseFs` | 82 | `0.57` (settled) |
| `language.Lang+publisher` | 82 | `0.64` (settled) |
| `Proc.Cfg+Proc.Codec` | 81 | `0.52` (settled) |
| `a.m+cnh.GetIdentity` | 79 | `0.56` (settled) |
| `doctree.LockTypeRead+m.treePages+doctree.NodeShiftTreeWalker` | 77 | `0.54` (settled) |
| `loggers.TimeTrackf+l.WithField` | 77 | `0.53` (settled) |
| `b.configOptions+b.dependencyManager` | 73 | `0.59` (settled) |
| `conf.OutputFormats+init.taxonomies` | 73 | `0.57` (settled) |
| `i.ReadSeekCloser+i.Format` | 72 | `0.50` (settled) |
| `this.options+livereload.start` | 72 | `0.55` (settled) |
| `this.window+livereload.generateCacheBus…` | 67 | `0.53` (settled) |
| `i.getResourcePaths+i.hash` | 66 | `0.56` (settled) |
| `slices.Contains+fmt.Sprintf` | 66 | `0.52` (settled) |
| `hstrings.GetOrCompileRegexp+cast.ToIntE+hreflect.Indirect` | 65 | `0.51` (settled) |
| `pageConfigSource.Content+b.Bytes` | 64 | `0.53` (settled) |
| `d.Name+overlayfs` | 62 | `0.59` (settled) |
| `b.Grow+b.WriteString+utf8.DecodeRuneInString` | 61 | `0.42` (loose) |
| `strings.Trim+strings.Join` | 61 | `0.45` (loose) |
| `types.LowHigh+strings.Index` | 60 | `0.55` (settled) |
| `Conf.IsMultilingual+Conf.IsMultihost+s.Language` | 59 | `0.55` (settled) |
| `b.Grow+b.WriteString` | 59 | `0.41` (loose) |
| `console.log+this.console+this.options` | 58 | `0.58` (settled) |
| `s.s+s.Language` | 58 | `0.97` (unanimous) |
| `of.MediaType+goldmark_config` | 57 | `0.61` (settled) |
| `ExecHelper.Sec+rs.ExecHelper` | 54 | `0.57` (settled) |
| `d.Fs+lockedfile` | 53 | `0.60` (settled) |
| `doctree.NodeShiftTreeWalker+w.Walk` | 53 | `0.51` (settled) |
| `kinds.KindPage+kinds.KindSection` | 53 | `0.58` (settled) |
| `pageConfigSource.Content+Content.Markup` | 53 | `0.52` (settled) |
| `ShouldListLocal.And+pagePredicates.KindPage+pageMap.getPagesInSection` | 52 | `0.59` (settled) |
| `afero.Walk+info.IsDir` | 51 | `0.49` (loose) |
| `depsFinder.Contains+identity.FinderConfig` | 51 | `0.47` (loose) |
| `doctree.LockTypeRead+m.treePages` | 51 | `0.54` (settled) |
| `openapi3+resources` | 51 | `0.54` (settled) |
| `htmlFormat.MediaType+s.htmlFormat` | 50 | `0.52` (settled) |
| `json.MarshalIndent+json.NewDecoder` | 50 | `0.54` (settled) |
| `strings.ReplaceAll+hugofs.FileMetaInfo` | 50 | `0.53` (settled) |
| `testing.Short+t.Skip+t.Skipf~2` | 49 | `0.38` (loose) |
| `l.spec+mime` | 48 | `0.63` (settled) |
| `s.CheckReady+i18n` | 48 | `0.63` (settled) |
| `r.Cfg+Cfg.GetConfigSection` | 47 | `0.53` (settled) |
| `s.C+s.H` | 47 | `0.56` (settled) |
| `s.inPath+strings.Index` | 47 | `0.51` (settled) |
| `ast.Heading+n.Kind` | 46 | `0.53` (settled) |
| `pi.Ext+fi.Meta` | 46 | `0.63` (settled) |
| `Conf.NewIdentityManager+s.Conf` | 45 | `0.57` (settled) |
| `LoadingInfo.ConfigFiles+configs.LoadingInfo` | 45 | `0.45` (loose) |
| `Build.Render+m.cascadeCompiled` | 44 | `0.66` (settled) |
| `cfg.MarkupConfig+fmt.Errorf` | 44 | `0.52` (settled) |
| `content.pi+pi.contentSource` | 44 | `0.52` (settled) |
| `hugofs.NewFrom+afero.NewMemMapFs` | 44 | `0.48` (loose) |
| `pathInfo.Unnormalized+kinds.KindTerm` | 44 | `0.58` (settled) |
| `arguments.length+_add-to-unscopables` | 43 | `0.54` (settled) |
| `filepath.Abs+hugofs.Os` | 43 | `0.52` (settled) |
| `s.build+s.Cfg` | 43 | `0.53` (settled) |
| `Matrix.Languages+Sites.Matrix` | 41 | `0.51` (settled) |
| `Meta.InclusionFilter+Meta.Rename` | 41 | `0.47` (loose) |
| `lv.Index+lv.Kind+lv.Len` | 41 | `0.29` (loose) |
| `Source.MkdirAll+s.C` | 40 | `0.45` (loose) |
| `cast.ToFloat64+cast.ToInt` | 40 | `0.42` (loose) |
| `doctree.WalkContext+cnh.isBranchNode` | 40 | `0.44` (loose) |
| `p.Draft+p.Weight` | 40 | `0.58` (settled) |
| `c.disableLiveReload+c.tlsAuto` | 39 | `0.54` (settled) |
| `r.cfgDir+r.cfgFile` | 39 | `0.54` (settled) |
| `anim.GetFrames+h.img` | 38 | `0.52` (settled) |
| `fim.IsDir+meta.Filename` | 38 | `0.53` (settled) |
| `m.Owner+m.Version` | 38 | `0.47` (loose) |
| `r.Hugo+r.ConfigFromProvider` | 38 | `0.52` (settled) |
| `Lastmod.After+collections.*Namespace.After` | 37 | `0.54` (settled) |
| `c.getNestedKeyAndMap+c.root` | 37 | `0.42` (loose) |
| `l.warnl+warnl.Logf` | 37 | `0.46` (loose) |
| `parse.StringNode+parse.PipeNode` | 37 | `0.35` (loose) |
| `fmt.Fprint+fmt.Fprintf` | 36 | `0.44` (loose) |
| `l.startLine+l.width` | 36 | `0.38` (loose) |
| `pathInfo.Unnormalized+p.PathInfo` | 36 | `0.53` (settled) |
| `po.f+pco.po` | 36 | `0.50` (loose) |
| `s.inPath+treePages.LongestPrefix` | 36 | `0.46` (loose) |
| `image.NewPaletted+src.Bounds` | 35 | `0.42` (loose) |
| `m.f+f.FileInfo` | 35 | `0.55` (settled) |
| `pathInfo.Base+pi.Base` | 35 | `0.56` (settled) |
| `result.setVector+s.ForEachVector` | 35 | `0.51` (settled) |
| `s.TemplateStore+context.Background` | 35 | `0.54` (settled) |
| `ShouldListLocal.And+pagePredicates.KindPage` | 34 | `0.44` (loose) |
| `iter.Next+reflect.New` | 34 | `0.42` (loose) |
| `m.treeTaxonomyEntries+doctree.LockTypeNone` | 34 | `0.47` (loose) |
| `parse.DotNode+parse.NilNode` | 34 | `0.35` (loose) |
| `path.Split+path.Join` | 34 | `0.40` (loose) |
| `BaseFs.PublishFs+bufferpool` | 33 | `0.52` (settled) |
| `bytes.Index+afero.WriteFile` | 33 | `0.55` (settled) |
| `cmd.PersistentFlags+cobra.FixedCompletions` | 33 | `0.50` (loose) |
| `fi.fim+fim.IsDir` | 33 | `0.49` (loose) |
| `h.allSites+Conf.Watching` | 33 | `0.45` (loose) |
| `hugoSites.GetOrCreate+hugolib.NewHugoSites+r.newDepsConfig` | 33 | `0.48` (loose) |
| `paths.PathEscape+path.Join` | 33 | `0.49` (loose) |
| `Meta.Rename+filepath.Split` | 32 | `0.44` (loose) |
| `a.m+h.SourceSpec+a.h` | 32 | `0.42` (loose) |
| `b.WriteString+b.String` | 32 | `0.37` (loose) |
| `cmd.PersistentFlags+cmd.MarkFlagDirname` | 32 | `0.51` (settled) |
| `hstrings.GetOrCompileRegexp+cast.ToIntE` | 32 | `0.41` (loose) |
| `maps.Copy+maps` | 32 | `0.53` (settled) |
| `p.PathInfo+pathInfo.Unnormalized` | 32 | `0.52` (settled) |
| `template.errorf+html` | 32 | `0.45` (loose) |
| `a.m+h.SourceSpec` | 31 | `0.41` (loose) |
| `configs.IsMultihost+configs.Languages` | 31 | `0.55` (settled) |
| `errState.setBuildErr+errState.wasErr` | 31 | `0.44` (loose) |
| `identity.GenghisKhan+pageTrees.treePagesFromTemp…` | 31 | `0.45` (loose) |
| `p.PathInfo+pathInfo.Unnormalized+kinds.KindTaxonomy` | 31 | `0.58` (settled) |
| `resources.InternalResourceT…+identity.FirstIdentity` | 31 | `0.50` (settled) |
| `siteVector.Language+s.siteVector` | 31 | `0.53` (settled) |
| `Buffer.Len+ctx.Buffer` | 30 | `0.35` (loose) |
| `hooks.LinkRenderer+html.IsDangerousURL` | 30 | `0.57` (settled) |
| `hpaths.AbsPathify+paths.AbsPathify` | 30 | `0.44` (loose) |
| `image.Config+draw` | 30 | `0.50` (loose) |
| `meta.Name+Meta.InclusionFilter` | 30 | `0.46` (loose) |
| `sort.Slice+fmt.Errorf` | 30 | `0.44` (loose) |
| `treePages.LongestPrefix+s.home` | 30 | `0.56` (settled) |
| `Source.MkdirAll+s.atomicSavedFiles` | 29 | `0.41` (loose) |
| `ctx.pids+ctx.values` | 29 | `0.50` (loose) |
| `helpers.Exists+filepath.Join` | 29 | `0.47` (loose) |
| `overlayfs.OverlayFs+b.p+b.theBigFs` | 29 | `0.44` (loose) |
| `p.Base+pt.p` | 29 | `0.49` (loose) |
| `ti.Template+metrics` | 29 | `0.51` (settled) |
| `Matrix.IsZero+Sites.Matrix` | 28 | `0.52` (settled) |
| `l.content+bytes.HasPrefix` | 28 | `0.43` (loose) |
| `paths.ToSlashTrimLeading+opts.TargetPath` | 28 | `0.47` (loose) |
| `testing.Short+t.Skip` | 28 | `0.25` (loose) |
| `time.RFC3339+p.File` | 28 | `0.53` (settled) |
| `Conf.IsMultilingual+Conf.IsMultihost` | 27 | `0.55` (settled) |
| `bytes.Contains+text` | 27 | `0.45` (loose) |
| `fsnotify.Chmod+fsnotify.Rename` | 27 | `0.47` (loose) |
| `pi.itemsStep2+fmt.Errorf` | 27 | `0.38` (loose) |
| `best.w+best.templ` | 26 | `0.47` (loose) |
| `bytes.Equal+bytes` | 26 | `0.47` (loose) |
| `f.fs+f.name` | 26 | `0.47` (loose) |
| `filepath.Ext+strings.TrimPrefix` | 26 | `0.45` (loose) |
| `files.ComponentFolderData+files.ComponentFolderAssets` | 26 | `0.47` (loose) |
| `h.sitesVersionsRoles+doctree.NodeShiftTreeWalker` | 26 | `0.44` (loose) |
| `langs.Languages+l.Lang+fmt.Errorf` | 26 | `0.51` (settled) |
| `sort.Strings+c.mu` | 26 | `0.37` (loose) |
| `strconv.Atoi+strings.TrimSpace` | 26 | `0.44` (loose) |
| `testing.Short+t.Skip+t.Skipf` | 26 | `0.52` (settled) |
| `b.SourceFilesystems+hugofs.NoOpFs` | 25 | `0.47` (loose) |
| `c.partitions+lazycache` | 25 | `0.50` (settled) |
| `d.Site+d.Conf` | 25 | `0.68` (settled) |
| `hooks.LinkRenderer+lr.RenderLink` | 25 | `0.35` (loose) |
| `l.closingState+l.currShortcodeName` | 25 | `0.39` (loose) |
| `meta.PathInfo+pi.Base` | 25 | `0.49` (loose) |
| `Sites.Matrix+Matrix.IsZero` | 24 | `0.55` (settled) |
| `cmd.RunE+cmd.Long` | 24 | `0.65` (settled) |
| `fs.trackAndWrapFile+Fs.Open` | 24 | `0.60` (settled) |
| `fw.Close+helpers.OpenFilesForWriting+BaseFs.PublishFs` | 24 | `0.54` (settled) |
| `l.w+l.append` | 24 | `0.36` (loose) |
| `langs.Languages+l.Lang` | 24 | `0.57` (settled) |
| `lv.Index+lv.Kind` | 24 | `0.29` (loose) |
| `os.ErrNotExist+overlayfs` | 24 | `0.45` (loose) |
| `strings.EqualFold+strings.Contains` | 24 | `0.45` (loose) |
| `Conf.ConfiguredDimensions+sitesmatrix.IntSetsConfig` | 23 | `0.46` (loose) |
| `gohtml.EscapeString+h.cfg` | 23 | `0.47` (loose) |
| `hashing.HashString+hashing` | 23 | `0.52` (settled) |
| `hreflect.ToFloat64E+hreflect.ToStringE` | 23 | `0.40` (loose) |
| `id.High+id.Low` | 23 | `0.45` (loose) |
| `t.pipeline+t.peekNonSpace` | 23 | `0.41` (loose) |
| `TemplateStore.TextParse+fmt.Errorf` | 22 | `0.47` (loose) |
| `b.String+bp.GetBuffer` | 22 | `0.37` (loose) |
| `cfg.taxonomyTermDisabled+viewName.plural+viewName.pluralTreeKey` | 22 | `0.48` (loose) |
| `fe.position+errors.Unwrap` | 22 | `0.48` (loose) |
| `h.Fs+Fs.Source` | 22 | `0.46` (loose) |
| `l.cfg+cfg.Get` | 22 | `0.45` (loose) |
| `main.buildFlags+main.flagEnv` | 22 | `0.49` (loose) |
| `Internal.Watch+Base.Internal` | 21 | `0.55` (settled) |
| `conf.Name+idx.index` | 21 | `0.45` (loose) |
| `config.C+c.config` | 21 | `0.74` (settled) |
| `gohtml.EscapeString+html.EscapeString` | 21 | `0.52` (settled) |
| `helpers.WriteToDisk+filepath.Base` | 21 | `0.41` (loose) |
| `init.taxonomies+menus.Value` | 21 | `0.46` (loose) |
| `l.current+l.isInline` | 21 | `0.38` (loose) |
| `p.posIdentifierBaseof+p.posNameHigh+p.posIdentifierLayout` | 21 | `0.45` (loose) |
| `passthrough.KindPassthrough…+ast.WalkContinue` | 21 | `0.51` (settled) |
| `r.Sorted+version` | 21 | `0.53` (settled) |
| `s.sets+hashstructure` | 21 | `0.55` (settled) |
| `helpers.FilePathSeparator+filepath.Join` | 20 | `0.52` (settled) |
| `html.RenderAttributes+attributes.AttributesOwnerG…` | 20 | `0.39` (loose) |
| `l.LanguageConfig+l.Logger` | 20 | `0.47` (loose) |
| `l.startLine+l.line` | 20 | `0.37` (loose) |
| `l.width+utf8.DecodeRune` | 20 | `0.38` (loose) |
| `logg.LevelDebug+fmt.Errorf` | 20 | `0.44` (loose) |
| `m.Kind+kinds.KindSection` | 20 | `0.53` (settled) |
| `os.Getenv+t.Helper` | 20 | `0.38` (loose) |
| `p.buildState+p.p` | 20 | `0.44` (loose) |
| `parse.ChainNode+parse.IdentifierNode` | 20 | `0.47` (loose) |
| `resource.StaleVersion+p.c` | 20 | `0.40` (loose) |
| `t.rangeDepth+t.pipeline` | 20 | `0.43` (loose) |
| `unicode.IsSpace+unicode` | 20 | `0.43` (loose) |
| `l.w+l.content+l.start` | 19 | `0.32` (loose) |
| `m.Owner+m.gomod` | 19 | `0.37` (loose) |
| `overlayfs.OverlayFs+b.p` | 19 | `0.50` (loose) |
| `p.Base+p.Path` | 19 | `0.44` (loose) |
| `ps.File+ps.Path` | 19 | `0.60` (settled) |
| `sh.Run+page_generate` | 19 | `0.47` (loose) |
| `t.Tree+t.Name` | 19 | `0.52` (settled) |
| `Data.Params+Params.Height` | 18 | `0.36` (loose) |
| `MediaType.Type+collections` | 18 | `0.51` (settled) |
| `PartialDecoratorIDStack.Get+Context.PartialDecoratorIDS…` | 18 | `0.54` (settled) |
| `attr.Value+attr.Name` | 18 | `0.41` (loose) |
| `cast.ToFloat32+cast.ToInt` | 18 | `0.65` (settled) |
| `iofs.DirEntry+meta.Open` | 18 | `0.45` (loose) |
| `o.h+create` | 18 | `0.67` (settled) |
| `p.IsBundle+files.ComponentFolderLayouts` | 18 | `0.42` (loose) |
| `reflect.Invalid+reflect.Uintptr` | 18 | `0.40` (loose) |
| `seqv.Index+seqv.Kind` | 18 | `0.28` (loose) |
| `templ.Name+goldmark_config` | 18 | `0.49` (loose) |
| `Content.Markup+fmt.Errorf` | 17 | `0.58` (settled) |
| `c.rootCmd+Root.Command` | 17 | `0.75` (settled) |
| `cast.ToFloat64E+cast.ToIntE` | 17 | `0.66` (settled) |
| `cfg.GetStringMap+cfg.IsSet` | 17 | `0.40` (loose) |
| `glob.Glob+hglob.GetGlob` | 17 | `0.44` (loose) |
| `hooks.LinkRenderer+lr.RenderLink+Buffer.Len` | 17 | `0.42` (loose) |
| `l.currentRightShortcodeDelim+l.ignore` | 17 | `0.38` (loose) |
| `p.trace+p.zero` | 17 | `0.33` (loose) |
| `sort.Sort+reflect` | 17 | `0.39` (loose) |
| `strings.SplitSeq+strings.TrimSpace` | 17 | `0.33` (loose) |
| `time.Since+time.Second` | 17 | `0.26` (loose) |
| `PathSpec.BaseFs+Content.Fs` | 16 | `0.43` (loose) |
| `ShouldListLocal.And+pagePredicates.KindPage+pagePredicates.ShouldListLo…` | 16 | `0.39` (loose) |
| `afero.Exists+module` | 16 | `0.45` (loose) |
| `ast.WalkSkipChildren+w.Write` | 16 | `0.40` (loose) |
| `d.warnl+imagemeta.EXIF` | 16 | `0.49` (loose) |
| `draw.Over+image.NewRGBA` | 16 | `0.38` (loose) |
| `fi.fim+fi.p` | 16 | `0.48` (loose) |
| `fmt.Sprintf+hugo.deprecationLogLevelFro…` | 16 | `0.39` (loose) |
| `images.hexStringToColorGo+gift` | 16 | `0.47` (loose) |
| `m.keys+m.values` | 16 | `0.80` (unanimous) |
| `r.shift+r.shifter` | 16 | `0.40` (loose) |
| `sitesmatrix.IntSetsConfig+sitesmatrix.NewIntSetsBuild…` | 16 | `0.39` (loose) |
| `t.peek+t.next` | 16 | `0.43` (loose) |
| `t.translateFuncs+context.Context` | 16 | `0.47` (loose) |
| `v.Major+v.Minor` | 16 | `0.73` (settled) |
| `c.jsCtx+c.jsBraceDepth` | 15 | `0.39` (loose) |
| `cmd.RunE+cmd.Long+cmd.Short` | 15 | `0.62` (settled) |
| `init.Value+lcp.init` | 15 | `0.87` (unanimous) |
| `iofs.DirEntry+iofs.ReadDirFile` | 15 | `0.43` (loose) |
| `iter.Next+parse.NumberNode` | 15 | `0.24` (loose) |
| `logg.Field+logg.Fields` | 15 | `0.40` (loose) |
| `meta.Filename+fim.Meta` | 15 | `0.30` (loose) |
| `this.handlers+this._disconnectionReason` | 15 | `0.38` (loose) |
| `vv.category+dh.compareDescriptors` | 15 | `0.29` (loose) |
| `Constructor.prototype+descriptor.configurable` | 14 | `0.90` (unanimous) |
| `WalkContext.AddEventListener+WalkContext.SendEvent` | 14 | `0.34` (loose) |
| `WrapperEnd.Low+WrapperStart.High` | 14 | `0.42` (loose) |
| `buf.WriteString+buf.Bytes` | 14 | `0.34` (loose) |
| `c.ToTOML+security.Config.ToTOML` | 14 | `0.44` (loose) |
| `cfg.Fs+filepath.Join` | 14 | `0.48` (loose) |
| `cmp+quicktest` | 14 | `0.54` (settled) |
| `fmt.Printf+runtime` | 14 | `0.44` (loose) |
| `gift.OverOperator+image.NewRGBA` | 14 | `0.43` (loose) |
| `hugio.CopyDir+hugofs.Os` | 14 | `0.46` (loose) |
| `hugio.NewReadSeekerNoOpClos…+hugio.ReadSeekCloser` | 14 | `0.37` (loose) |
| `hugoSites.GetOrCreate+hugolib.NewHugoSites` | 14 | `0.31` (loose) |
| `l.backup+l.next` | 14 | `0.31` (loose) |
| `r.metaProvider+r.init` | 14 | `0.81` (unanimous) |
| `s.lastBuildLog+s.Helper` | 14 | `0.48` (loose) |
| `spec.Cfg+Cfg.BaseURL` | 14 | `0.49` (loose) |
| `t.pipeline+t.backup` | 14 | `0.48` (loose) |
| `this.connector+this.initialized` | 14 | `0.43` (loose) |
| `types.ToStringSlicePreserve…+cmp` | 14 | `0.39` (loose) |
| `Cfg.Logger+a.Cfg` | 13 | `0.30` (loose) |
| `Lastmod.After+htime.Now` | 13 | `0.28` (loose) |
| `bytes.IndexAny+state.String` | 13 | `0.39` (loose) |
| `cfg.ConfiguredRoles+cfg.ConfiguredVersions+other.languages` | 13 | `0.33` (loose) |
| `d.debugl+debugl.Log` | 13 | `0.47` (loose) |
| `exec.LookPath+bufio` | 13 | `0.32` (loose) |
| `herrors.NewFileErrorFromName+meta.Filename` | 13 | `0.33` (loose) |
| `idm.AddIdentity+identity.WalkIdentitiesShal…` | 13 | `0.34` (loose) |
| `n.Args+t.ParseInfo` | 13 | `0.41` (loose) |
| `p.component+files` | 13 | `0.42` (loose) |
| `parser.InterfaceToConfig+metadecoders.TOML` | 13 | `0.35` (loose) |
| `pos.ColumnNumber+fe.UpdatePosition` | 13 | `0.40` (loose) |
| `ti.D+tpl.Template` | 13 | `0.40` (loose) |
| `Data.Params+Params.Height+Params.Width` | 12 | `0.23` (loose) |
| `c.contentRendered+c.mustContentPlain` | 12 | `0.56` (settled) |
| `fs.PublishDirStatic+fs.PublishDir` | 12 | `0.33` (loose) |
| `h.img+h.Assert` | 12 | `0.31` (loose) |
| `image.NewNRGBA+src.Bounds` | 12 | `0.26` (loose) |
| `item.Kind+item.Len` | 12 | `0.24` (loose) |
| `other.ForEachVector+s.HasVector` | 12 | `0.61` (settled) |
| `p.Kind+p.Path` | 12 | `0.43` (loose) |
| `pageConfig.Dates+m.pageConfig` | 12 | `0.92` (unanimous) |
| `spc.get+compare` | 12 | `0.63` (settled) |
| `strconv.Itoa+strconv` | 12 | `0.36` (loose) |
| `strings.CutPrefix+fmt.Errorf` | 12 | `0.39` (loose) |
| `t.Skipf+runtime.GOARCH` | 12 | `0.55` (settled) |
| `t.leftDelim+t.rightDelim` | 12 | `0.46` (loose) |
| `t.treeSet+t.Mode` | 12 | `0.36` (loose) |
| `this._connectionDesired+this._reconnectTimer` | 12 | `0.40` (loose) |
| `ti.noBaseOf+ti.subCategory` | 12 | `0.41` (loose) |
| `Default.UnmarshalToMap+metadecoders.Default` | 11 | `0.34` (loose) |
| `ElseList.CopyList+List.CopyList` | 11 | `0.43` (loose) |
| `PathInfo.Path+fmt.Errorf` | 11 | `0.45` (loose) |
| `PathSpec.ProcessingStats+s.PathSpec` | 11 | `0.38` (loose) |
| `c.collect+coll.err` | 11 | `0.46` (loose) |
| `e.called+e.derived` | 11 | `0.43` (loose) |
| `f.MediaType+MediaType.FirstSuffix` | 11 | `0.40` (loose) |
| `fmt.Sprint+template.indirectToStringer…` | 11 | `0.34` (loose) |
| `fsnotify.Chmod+fsnotify.Write` | 11 | `0.43` (loose) |
| `i.Type+util` | 11 | `0.73` (settled) |
| `ls.Sorted+predicate` | 11 | `0.53` (settled) |
| `p.posIdentifierBaseof+p.posNameHigh` | 11 | `0.43` (loose) |
| `q.Path+rungroup` | 11 | `0.49` (loose) |
| `r.StdOut+hugolib` | 11 | `0.34` (loose) |
| `resources.InternalResourceS…+identity.FirstIdentity` | 11 | `0.43` (loose) |
| `state.String+c.state` | 11 | `0.41` (loose) |
| `strings.LastIndex+runtime` | 11 | `0.38` (loose) |
| `strings.Repeat+fmt.Sprintf` | 11 | `0.37` (loose) |
| `time.Duration+fmt.Errorf` | 11 | `0.38` (loose) |
| `DependencyManagerScopedProv…+Context.DependencyManagerSc…` | 10 | `0.43` (loose) |
| `allconfig.ConfigLanguage.De…+i18n` | 10 | `0.51` (settled) |
| `bytes.TrimSpace+unicode` | 10 | `0.38` (loose) |
| `cast.ToFloat32+images.newFilterOpts` | 10 | `0.94` (unanimous) |
| `d.PageConfigEarly+d.PageConfigLate` | 10 | `0.52` (settled) |
| `f.name+f.fs` | 10 | `0.53` (settled) |
| `hashing.HashUint64+hashing` | 10 | `0.33` (loose) |
| `identity.Anonymous+resources.ResourceSourceDes…` | 10 | `0.40` (loose) |
| `l.currShortcodeName+l.elementStepNum` | 10 | `0.41` (loose) |
| `l.sd+mime` | 10 | `0.79` (unanimous) |
| `maps.Clone+maps` | 10 | `0.40` (loose) |
| `media.Types+allconfig.ConfigLanguage.Ge…` | 10 | `0.42` (loose) |
| `ns.castPrecisionNumber+ns.translator` | 10 | `0.38` (loose) |
| `p.Layout+p.Type` | 10 | `0.41` (loose) |
| `p.posIdentifierOutputFormat+p.posIdentifierLanguage` | 10 | `0.36` (loose) |
| `parse.WithNode+parse.IfNode` | 10 | `0.35` (loose) |
| `paths.FileAndExtNoDelimiter+pos.Filename` | 10 | `0.41` (loose) |
| `paths.ToSlashPreserveLeading+hsync` | 10 | `0.43` (loose) |
| `r.getImageOps+resources.*resourceAdapter.…` | 10 | `1.00` (unanimous) |
| `sitesmatrix.VectorStore+files` | 10 | `0.37` (loose) |
| `t.name+parse` | 10 | `0.34` (loose) |
| `this._handshakeTimeout+this._disconnectionReason` | 10 | `0.40` (loose) |
| `w.w2+w.w3` | 10 | `0.36` (loose) |
| `ConfiguredDimensions.Config…+pp.ConfiguredDimensions` | 9 | `0.38` (loose) |
| `Fi.Meta+htmltemplate` | 9 | `0.44` (loose) |
| `buff.Reset+buff.WriteRune` | 9 | `0.39` (loose) |
| `c.commands+parser` | 9 | `0.91` (unanimous) |
| `cfg.ConfiguredRoles+cfg.ConfiguredVersions` | 9 | `0.43` (loose) |
| `converter.ProviderConfig.Ma…+blockquotes` | 9 | `0.58` (settled) |
| `h.numWorkers+g.Enqueue` | 9 | `0.31` (loose) |
| `hmaps.Params.GetMergeStrate…+cast` | 9 | `0.22` (loose) |
| `hstrings.UniqueStringsReuse+hstrings` | 9 | `0.32` (loose) |
| `html.ElementNode+html.Node` | 9 | `0.30` (loose) |
| `l.rightDelim+l.input` | 9 | `0.38` (loose) |
| `math.Floor+lazycache` | 9 | `0.43` (loose) |
| `p.Draft+p.ExpiryDate` | 9 | `0.37` (loose) |
| `parse.BreakNode+parse.ContinueNode` | 9 | `0.33` (loose) |
| `passthrough+attributes` | 9 | `0.28` (loose) |
| `po.f+po.p` | 9 | `0.43` (loose) |
| `qt.Contains+qt.Not` | 9 | `0.29` (loose) |
| `s.vars+fmtsort` | 9 | `0.34` (loose) |
| `source.*File.p+glob` | 9 | `0.25` (loose) |
| `t.translateFuncs+i18n` | 9 | `0.44` (loose) |
| `types.IsNil+compare` | 9 | `0.40` (loose) |
| `v.Role+v.Version` | 9 | `0.44` (loose) |
| `Divider.IsZero+Divider.Low` | 8 | `0.48` (loose) |
| `Source.MkdirAll+s.writeSource` | 8 | `0.58` (settled) |
| `b.cf+Log.Printf` | 8 | `0.26` (loose) |
| `fmt.Println+filepath.Join` | 8 | `0.29` (loose) |
| `glob.Compile+glob` | 8 | `0.30` (loose) |
| `herrors.NewFileErrorFromFile+fe.Position` | 8 | `0.29` (loose) |
| `hex.EncodeToString+cast.ToStringE` | 8 | `0.45` (loose) |
| `htesting.SupportsAll+internal` | 8 | `0.30` (loose) |
| `html.IsDangerousURL+html.RenderAttributes` | 8 | `0.27` (loose) |
| `html.IsDangerousURL+util.URLEscape` | 8 | `0.18` (loose) |
| `htmltemplate.Template+texttemplate.Template` | 8 | `0.32` (loose) |
| `i.WithSpec+i.baseResource` | 8 | `0.38` (loose) |
| `ids.vals+blackfriday` | 8 | `0.40` (loose) |
| `main.testGoFlags+main.runCmd` | 8 | `0.50` (settled) |
| `mt.Type+hreflect.IsContextType` | 8 | `0.27` (loose) |
| `muTmpl.RLock+muTmpl.RUnlock` | 8 | `0.45` (loose) |
| `ns.createClient+cast.ToStringE` | 8 | `0.50` (settled) |
| `p.CurrentSection+hstrings` | 8 | `0.41` (loose) |
| `p.norm+p.posContainerHigh` | 8 | `0.48` (loose) |
| `p.outputFormat+s.Conf` | 8 | `0.31` (loose) |
| `parse.TextNode+parse.TemplateNode` | 8 | `0.36` (loose) |
| `strconv.FormatUint+tpl` | 8 | `0.38` (loose) |
| `strings.Cut+cut` | 8 | `0.48` (loose) |
| `strings.NewReader+regexp` | 8 | `0.35` (loose) |
| `syscall.EPERM+syscall` | 8 | `0.88` (unanimous) |
| `t.ParseName+t.Root` | 8 | `0.31` (loose) |
| `t.tree+iter` | 8 | `0.72` (settled) |
| `t.vars+t.errorf` | 8 | `0.26` (loose) |
| `treePages.LongestPrefix+pt.p+cnh.isBranchNode` | 8 | `0.33` (loose) |
| `Array.isArray+Symbol.iterator` | 7 | `0.31` (loose) |
| `BuildState.IsRebuild+h.BuildState` | 7 | `0.23` (loose) |
| `Symbol.toPrimitive+prim.call` | 7 | `1.00` (unanimous) |
| `WalkContext.AddEventListener+WalkContext.SendEvent+doctree.Event` | 7 | `0.29` (loose) |
| `bytes.ToLower+utf8` | 7 | `0.29` (loose) |
| `c.Pos+c.tr` | 7 | `0.62` (settled) |
| `cfg.taxonomyTermDisabled+viewName.plural` | 7 | `0.30` (loose) |
| `errors.Unwrap+v2` | 7 | `0.24` (loose) |
| `fi.p+source.*File.p` | 7 | `0.70` (settled) |
| `fn.Call+mt.Type` | 7 | `0.36` (loose) |
| `fs.WorkingDirReadOnly+s.fs` | 7 | `0.45` (loose) |
| `fs.rootMapToReal+radix.WalkFn` | 7 | `0.34` (loose) |
| `h.SourceSpec+SourceSpec.IgnoreFile` | 7 | `0.56` (settled) |
| `hexec.InPath+fmt.Errorf` | 7 | `0.26` (loose) |
| `lf.matcher+gzip` | 7 | `0.56` (settled) |
| `m.Renderer+renderer.WithNodeRenderers` | 7 | `0.37` (loose) |
| `n.forEeachContentNode+sitesmatrix.Vector` | 7 | `0.46` (loose) |
| `p.Base+files` | 7 | `0.38` (loose) |
| `p.Ctx+p.Page` | 7 | `0.93` (unanimous) |
| `paths.TargetPath+l.paths` | 7 | `0.42` (loose) |
| `pco.c+hugolib.*pageContentOutput.c` | 7 | `0.71` (settled) |
| `pco.mustRender+hugolib.*pageContentOutput.…` | 7 | `0.82` (unanimous) |
| `qt.Equals+qt.Commentf` | 7 | `0.38` (loose) |
| `r.field+postpub.*PostPublishResourc…` | 7 | `0.67` (settled) |
| `t.Mode+t.treeSet+t.funcs` | 7 | `0.31` (loose) |
| `t.treeResources+t.treePages` | 7 | `0.24` (loose) |
| `treePages.LongestPrefix+pt.p` | 7 | `0.39` (loose) |
| `v.Index+v.Len` | 7 | `0.43` (loose) |
| `websocket+net` | 7 | `0.31` (loose) |
| `c.Get+cast` | 6 | `0.47` (loose) |
| `c.g+c.h` | 6 | `0.31` (loose) |
| `c.match+predicate.IndexString` | 6 | `0.68` (settled) |
| `d.ExpiryDate+d.PublishDate` | 6 | `0.14` (loose) |
| `d.IsDir+fs.WalkDir` | 6 | `0.27` (loose) |
| `fe.UpdatePosition+herrors.UnwrapFileError+fe.Position` | 6 | `0.31` (loose) |
| `i.opts+opts.Conf` | 6 | `0.61` (settled) |
| `identity.IdentityProvider+reflect` | 6 | `0.30` (loose) |
| `l.w+l.content` | 6 | `0.23` (loose) |
| `m.names+hashstructure` | 6 | `0.78` (unanimous) |
| `metadecoders.FormatFromStri…+parser` | 6 | `0.40` (loose) |
| `options.compression+options.hint` | 6 | `0.31` (loose) |
| `p.identifierAsString+paths.*Path.identifierAsStr…` | 6 | `0.64` (settled) |
| `pathInfo.IsBranchBundle+strings.HasPrefix` | 6 | `0.28` (loose) |
| `r.r+pagesfromdata` | 6 | `0.36` (loose) |
| `resource.MarkStale+sitesmatrix` | 6 | `0.39` (loose) |
| `s.cfg+s.s` | 6 | `0.44` (loose) |
| `srcBounds.Dx+srcBounds.Dy` | 6 | `0.95` (unanimous) |
| `strings.ContainsAny+unicode` | 6 | `0.42` (loose) |
| `t.Mode+t.treeSet` | 6 | `0.21` (loose) |
| `t.unexpected+token.typ` | 6 | `0.55` (settled) |
| `ShouldListLocal.And+pagePredicates.KindPage+pageMap.getPagesInSection~2` | 5 | `0.75` (unanimous) |
| `b.Pos+b.tr` | 5 | `0.57` (settled) |
| `b.cf+b.sourceFs` | 5 | `0.16` (loose) |
| `bluemonday+passthrough` | 5 | `0.46` (loose) |
| `c.contentRendered+c.prepareContext` | 5 | `0.58` (settled) |
| `d.Date+d.Lastmod` | 5 | `0.37` (loose) |
| `deps.PathSpec+ns.deps` | 5 | `0.86` (unanimous) |
| `docshelper.AddDocProviderFu…+docshelper.DocProvider` | 5 | `0.23` (loose) |
| `fe.UpdatePosition+herrors.UnwrapFileError` | 5 | `0.31` (loose) |
| `filepath.SkipDir+info.IsDir` | 5 | `0.21` (loose) |
| `h.PathSpec+PathSpec.BaseFs+BaseFs.Content` | 5 | `0.30` (loose) |
| `hr.templ+hr.templateHandler` | 5 | `0.89` (unanimous) |
| `ic.Quality+c.Compression` | 5 | `0.38` (loose) |
| `loggers.Log+htime` | 5 | `0.49` (loose) |
| `lv.Index+lv.Kind+lv.Len~2` | 5 | `0.24` (loose) |
| `memory+strconv` | 5 | `0.30` (loose) |
| `n.compareGet+n.checkComparisonArgCount` | 5 | `0.60` (settled) |
| `os.ErrNotExist+fs` | 5 | `0.59` (settled) |
| `p.PageNumber+page.*Pager.PageNumber` | 5 | `0.60` (settled) |
| `p.Path+hugofs.FileMetaInfo+strings.HasPrefix` | 5 | `0.22` (loose) |
| `p.els+urls` | 5 | `0.29` (loose) |
| `p.pagers+math` | 5 | `0.35` (loose) |
| `p.paginatedElements+math` | 5 | `0.32` (loose) |
| `parser.InterfaceToConfig+bytes.Buffer` | 5 | `0.37` (loose) |
| `parson.json_object_set_value+parson.json_value_free` | 5 | `1.00` (unanimous) |
| `paths.NormalizePathStringBa…+files` | 5 | `0.33` (loose) |
| `s.Fatalf+s.Helper` | 5 | `0.34` (loose) |
| `sb.WriteByte+sb.WriteString` | 5 | `0.67` (settled) |
| `sid.dp+sid.eqer` | 5 | `0.36` (loose) |
| `smartcrop+gift` | 5 | `0.41` (loose) |
| `sort.Stable+compare` | 5 | `0.54` (settled) |
| `t.items+t.lastPos` | 5 | `0.54` (settled) |
| `t.peekCount+t.token` | 5 | `0.18` (loose) |
| `unicode.IsDigit+unicode.IsLetter` | 5 | `0.55` (settled) |
| `v2+transform` | 5 | `0.34` (loose) |
| `weight.distance+best.isBetter` | 5 | `0.21` (loose) |
| `Fs.Create+fs.Fs` | 4 | — |
| `Fs.OpenFile+fs.Fs` | 4 | — |
| `Object.prototype+constructor.name` | 4 | — |
| `PathInfo.Path+metrics` | 4 | — |
| `b.writeHeadings+s.WriteString` | 4 | — |
| `c.mustContentPlain+c.prepareContext` | 4 | — |
| `cfg.Deps+ExecHelper.New` | 4 | — |
| `d.d+goat` | 4 | — |
| `doc+docshelper` | 4 | — |
| `errorl.Logf+l.errorl` | 4 | — |
| `fi.fim+fim.Meta` | 4 | — |
| `fim.IsDir+fim.Meta` | 4 | — |
| `h.img+h.Assert+qt.IsTrue` | 4 | — |
| `hreflect.IsMap+css` | 4 | — |
| `i.processActionSpec+resources.*imageResource.pr…` | 4 | — |
| `l.emitItem+l.thisItem` | 4 | — |
| `l.urlize+p.Slug` | 4 | — |
| `p.GetLanguagePrefix+paths.*Paths.GetLanguagePre…` | 4 | — |
| `p.Pairs+compare` | 4 | — |
| `p.SitesComplements+p.SitesMatrix` | 4 | — |
| `p.groupByDateField+time.Time` | 4 | — |
| `p.trace+trace.Log` | 4 | — |
| `parson.json_array_append_va…+parson.json_value_free` | 4 | — |
| `parson.json_array_replace_v…+parson.json_value_free` | 4 | — |
| `parson.json_object_dotset_v…+parson.json_value_free` | 4 | — |
| `paths.fileAndExt+unicode` | 4 | — |
| `pco.fail+hugolib.*pageContentOutput.…` | 4 | — |
| `pflag+log` | 4 | — |
| `pos.LineNumber+herrors.*fileError.UpdatePo…` | 4 | — |
| `reg.Register+attributes` | 4 | — |
| `replacer.Replace+strings.NewReplacer` | 4 | — |
| `runes+transform` | 4 | — |
| `scp.Page+pageparser` | 4 | — |
| `slices.Delete+slices` | 4 | — |
| `slices.Equal+slices` | 4 | — |
| `tt.tree+t.Trees` | 4 | — |
| `Format.Name+output` | 3 | — |
| `create+skeletons` | 3 | — |
| `p.IsBundle+paths.*Path.IsBundle` | 3 | — |
| `p.Layout+p.Type+p.Section` | 3 | — |
| `source.NewFileInfo+source` | 3 | — |
| `herrors.NewFileErrorFromFile+pos.Filename` | 2 | — |
| `m.term+pageMap.cfg` | 2 | — |
| `t.Skip+t.Helper` | 2 | — |
| `m.treeResources+m.treePages` | 1 | — |

Convention is how uniformly this corpus realizes a concept: `1.00` means every function carrying the tag does it the same way, and a low number means the tag covers several unrelated habits. A concept with fewer than five members is not modeled.

### Where the duplication is

Merge-worthy pairs are folded up to their packages: only pairs doppel judges worth consolidating are counted. An edge means two packages keep solving the same problem separately; a count on a node means the repetition is inside one package. Weights are **merge-worthy pairs**.

```mermaid
flowchart LR
    p0["blockquotes<br/>1 internal"]
    p1["goldmark<br/>43 internal"]
    p0 ---|"9"| p1
    p2["babel"]
    p3["cssjs<br/>5 internal"]
    p2 ---|"8"| p3
    p4["genavif<br/>1 internal"]
    p5["genwebp<br/>2 internal"]
    p4 ---|"8"| p5
    p6["langs<br/>18 internal"]
    p7["roles<br/>1 internal"]
    p6 ---|"8"| p7
    p8["versions"]
    p7 ---|"8"| p8
    p6 ---|"7"| p8
    p9["navigation<br/>3 internal"]
    p10["page<br/>339 internal"]
    p9 ---|"7"| p10
    p11["css<br/>4 internal"]
    p12["js<br/>8 internal"]
    p11 ---|"6"| p12
    p13["tables<br/>2 internal"]
    p1 ---|"6"| p13
    p14["strings<br/>32 internal"]
    p15["urls<br/>12 internal"]
    p14 ---|"6"| p15
    p0 ---|"5"| p13
    p16["collections<br/>56 internal"]
    p17["template<br/>194 internal"]
    p16 ---|"5"| p17
```

_407 further package pairs are connected by duplication and are not drawn._

### How settled each package is

A package with at least five functions gets a habitat model: doppel learns what is normal there and measures how surprising each member is against it. **Norm** is how uniform the package's practice is. A **misfit** is a function alien to its package *and* to the wider subsystem around it — one that fits its neighbours a directory up is normal for this codebase and is not reported.

```mermaid
flowchart TD
    h0["page<br/>468 functions · norm 0.66<br/>129 misfits"]
    h1["integrity<br/>7 functions · norm 0.68"]
    h2["versions<br/>12 functions · norm 0.69<br/>4 misfits"]
    h3["langs<br/>31 functions · norm 0.69<br/>10 misfits"]
    h4["blockquotes<br/>12 functions · norm 0.69"]
    h5["sitesmatrix<br/>107 functions · norm 0.70<br/>33 misfits"]
    h6["warpc<br/>54 functions · norm 0.70<br/>16 misfits"]
    h7["pagesfromdata<br/>20 functions · norm 0.70<br/>7 misfits"]
    h8["roles<br/>12 functions · norm 0.71<br/>4 misfits"]
    h9["allconfig<br/>93 functions · norm 0.71<br/>25 misfits"]
    h10["hugofs<br/>195 functions · norm 0.71<br/>54 misfits"]
    h11["identity<br/>55 functions · norm 0.71<br/>19 misfits"]
    classDef good fill:#d7ecd9,color:#1b3d20
    classDef warn fill:#fbeecb,color:#4a3a12
    classDef hot fill:#f7d6d6,color:#4a1c1c
    class h0,h1,h2,h3,h4,h5,h6,h7,h8,h9,h10,h11 warn
```

_117 further packages are modeled and not drawn._ Most uniform is `dartsass` (norm `0.98`); most varied is `page` (norm `0.66`). 416 functions are alien to their package and to the subsystem around it. A further 63 fit poorly in their package but match the wider subsystem, so they are not reported.

### How these candidates were found

Three channels propose candidates independently — shared rare *structure*, shared *concepts*, shared *calls* — and their union is what gets compared. This run: **32251 candidate pairs** (shape 7055, concept 20730, call 8634), of which 19% arrived on call evidence alone and 56% on concept evidence alone. A pair sharing none of the three is never compared, however alike it looks.

The concept signal on each compared pair is read three ways — what the taxonomy asserts, what this corpus's frequencies say, and what the two sides' learned vocabularies share with no tree in between. On **2250 of 31907** pairs the taxonomy and the vocabularies differ by at least 0.50: 161 where the vocabularies agree and the tree cannot see it, 2089 where the tree asserts a kinship the vocabularies lack. Each such pair carries a `concept views` line saying which.

Each function is also an arena where its candidate concepts compete for its evidence. 5546 functions reached an equilibrium: **3227** settled on a single concept, **2312** on a coalition, **0** hold concepts this corpus says do not go together.

### Corpus metrics

**Compression ratio:** `7.36`x — this corpus's canonical function bodies contain **362272 AST nodes** in total, which hash-cons (two nodes count as the same subtree exactly when their kind and every child match, all the way down) to **49224 distinct subtree shapes**; the ratio is nodes divided by shapes, always >= 1.0, and it never feeds any score.

**Nearest-neighbour code-shape:** of **5737 functions**, **5686** had a code-shape neighbour among the pairs retrieval actually scored — their best score's p50/p90/p99 are `0.45` / `1.00` / `1.00`, and 80% of them (4548 of 5686) already clear this run's threshold of `0.34`. This is **not an exhaustive nearest-neighbour search** (that would be a full pairwise comparison); it is bounded by the same three retrieval channels the pair list itself is bounded by, so the other 51 functions are excluded here as having no *scored* neighbour, not asserted to have none at all.

---

## Local practice

The vocabulary above says what a concept *is*. This says what one looks like when **this** codebase writes it — learned from the corpus, so it describes the house style rather than a rule from anywhere else.

### How this codebase writes each concept

Only what is **distinctive**. A feature earns a row by being carried by this concept's members at least twice as often as by the corpus at large — nearly every Go function has a `return` and an `if`, so prevalence alone would describe the language rather than this codebase. Weights are how much a channel counts toward whether a member looks normal — calls 40, control flow 20, co-occurring tags 15, role 15, package 10.

**`mnt.init+typ.In`** — 256 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `fmt.Errorf` | `█████·····` | 135 of 256 | 8.8× |
| flow ×20 | `range` | `███·······` | 86 of 256 | 2.1× |

**`ResourceSpec.FileCaches+c.isExpired`** — 242 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| flow ×20 | `funclit` | `███·······` | 72 of 242 | 2.6× |

**`C.BaseURL+conf.C`** — 233 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| cotags ×15 | `Config.Sorted+c.C` | `█████·····` | 113 of 233 | 14× |
|  | `pageOutput.pco+ps.pageOutput` | `███·······` | 77 of 233 | 13× |
| package ×10 | `page` | `█████·····` | 127 of 233 | 6.7× |

**`c.hasBeenInitialized+call.done`** — 210 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| flow ×20 | `defer` | `█████·····` | 110 of 210 | 10× |

**`Converters.GetHighlighter+c.applyTransformations`** — 207 functions

Nothing distinctive: its members do what the rest of the corpus does. The tag groups them; a shared way of writing them does not.

**`fn.IsValid+htmltemplate.Must`** — 204 functions

| Channel | Feature | | Members | vs corpus |
|---|---|---|---|---|
| calls ×40 | `reflect.ValueOf` | `███·······` | 61 of 204 | 21× |
| flow ×20 | `switch` | `███·······` | 55 of 204 | 5.3× |
| package ×10 | `template` | `███·······` | 60 of 204 | 6.1× |

_496 further concepts are modeled and not described._

### Which concepts share a function

`++` at least four times chance, `+` at least twice, `−` at most half, `never` not once. A blank cell is ordinary company — near chance, which is not culture.

_Showing 12 of 547 concepts — those in the strongest pairings, taken strongest first by lift weighted by how many functions it speaks for. Every cell between them is shown; the other 535 concepts are not on the grid._

| | `Body.Close+api.OnLoadArgs` | `C.BaseURL+conf.C` | `Config.Sorted+c.C` | `Converters.GetHighlighter+c.applyTransformations` | `ResourceSpec.FileCaches+c.isExpired` | `c.hasBeenInitialized+call.done` | `fn.IsValid+htmltemplate.Must` | `gohtml.EscapeString+h.cfg` | `gohtml.EscapeString+html.EscapeString` | `init.data+init.gitInfo` | `mnt.init+typ.In` |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **`C.BaseURL+conf.C`** | never | | | | | | | | | | |
| **`Config.Sorted+c.C`** | never | ++ | | | | | | | | | |
| **`Converters.GetHighlighter+c.applyTransformations`** | + | never | never | | | | | | | | |
| **`ResourceSpec.FileCaches+c.isExpired`** | ++ | − | never |  | | | | | | | |
| **`c.hasBeenInitialized+call.done`** | + | − | never | − |  | | | | | | |
| **`fn.IsValid+htmltemplate.Must`** | never | − | never |  | − |  | | | | | |
| **`gohtml.EscapeString+h.cfg`** |  |  |  |  | + |  |  | | | | |
| **`gohtml.EscapeString+html.EscapeString`** |  |  |  |  |  |  |  | ++ | | | |
| **`init.data+init.gitInfo`** | − | − | − | never |  | + | − |  |  | | |
| **`mnt.init+typ.In`** | + | never | never |  | + | − |  |  |  | − | |
| **`pageOutput.pco+ps.pageOutput`** | never | ++ | ++ | never | − | never | never |  |  | never | never |

### What travels with what

Co-occurrence measured against chance across every function. Only relationships at least twice — or at most half — as common as chance are reported; near-chance company is not culture. Each kind is listed separately, because there are far more call tokens than concepts and one shared list is all calls. Within a kind, strongest first means lift weighted by how many functions carry it — a 100× relationship holding for three functions is a weaker finding than a 30× one holding for thirty.

**Together more than chance — tag~tag**

- 20 of 21 `gohtml.EscapeString+html.EscapeString` functions also `gohtml.EscapeString+h.cfg` — 238× chance
- 46 of 48 `s.CheckReady+i18n` functions also `conf.OutputFormats+init.taxonomies` — 75× chance
- 27 of 29 `Source.MkdirAll+s.atomicSavedFiles` functions also `Source.MkdirAll+s.C` — 134× chance
- 58 of 58 `s.s+s.Language` functions also `Config.Sorted+c.C+c.Languages` — 53× chance
- 44 of 59 `b.Grow+b.WriteString` functions also `b.Grow+b.WriteString+utf8.DecodeRuneInString` — 70× chance
- 27 of 32 `Meta.Rename+filepath.Split` functions also `Meta.InclusionFilter+Meta.Rename` — 118× chance
- _1027 more not listed_

**Together more than chance — tag~role**

- 23 of 25 `d.Site+d.Conf` functions also `orchestrator` — 5.8× chance
- 12 of 12 `c.contentRendered+c.mustContentPlain` functions also `orchestrator` — 6.3× chance
- 7 of 22 `b.String+bp.GetBuffer` functions also `passthrough` — 6.9× chance
- 4 of 8 `herrors.NewFileErrorFromFile+fe.Position` functions also `passthrough` — 11× chance
- 5 of 13 `herrors.NewFileErrorFromName+meta.Filename` functions also `passthrough` — 8.3× chance
- 10 of 16 `t.peek+t.next` functions also `utility` — 4.4× chance
- _151 more not listed_

**Together more than chance — tag~call**

- 25 of 25 `d.Site+d.Conf` functions also `internal.AddTemplateFuncsNamespace` — 164× chance
- 21 of 25 `d.Site+d.Conf` functions also `internal.*TemplateFuncsNamespace.AddMethodMapping` — 166× chance
- 14 of 14 `Constructor.prototype+descriptor.configurable` functions also `./_redefine.defineProperty` — 337× chance
- 16 of 17 `cast.ToFloat64E+cast.ToIntE` functions also `github.com/spf13/cast.ToFloat64E` — 245× chance
- 19 of 24 `strings.EqualFold+strings.Contains` functions also `strings.EqualFold` — 182× chance
- 20 of 32 `hstrings.GetOrCompileRegexp+cast.ToIntE` functions also `github.com/spf13/cast.ToFloat64E` — 163× chance
- _1094 more not listed_

**Apart more than chance — tag~tag**

- **no** `C.BaseURL+conf.C` function has `mnt.init+typ.In` — chance alone would give about 10 of 233
- **no** `Config.Sorted+c.C` function has `mnt.init+typ.In` — chance alone would give about 9 of 203
- **no** `Config.Sorted+c.C` function has `ResourceSpec.FileCaches+c.isExpired` — chance alone would give about 9 of 203
- **no** `C.BaseURL+conf.C` function has `Converters.GetHighlighter+c.applyTransformations` — chance alone would give about 8 of 233
- **no** `Config.Sorted+c.C` function has `c.hasBeenInitialized+call.done` — chance alone would give about 7 of 203
- **no** `Config.Sorted+c.C` function has `Converters.GetHighlighter+c.applyTransformations` — chance alone would give about 7 of 203
- _123 more not listed_

**Apart more than chance — tag~role**

- **no** `t.rangeDepth+t.pipeline` function has `leaf` — chance alone would give about 13 of 20
- **no** `hstrings.GetOrCompileRegexp+cast.ToIntE+hreflect.Indirect` function has `orchestrator` — chance alone would give about 10 of 65
- **no** `Config.Sorted+c.C` function has `passthrough` — chance alone would give about 9 of 203
- **no** `s.s+s.Language` function has `orchestrator` — chance alone would give about 9 of 58
- **no** `s.s+s.Language` function has `utility` — chance alone would give about 8 of 58
- **no** `c.contentRendered+c.mustContentPlain` function has `leaf` — chance alone would give about 8 of 12
- _162 more not listed_

### Functions drifting from their own concept

These carry a tag but look nothing like the other functions carrying it. Typicality is measured against the concept's own median, so a genuinely varied concept lowers its own bar and a tight one can flag nobody.

| Function | Concept | Typicality | Concept median | |
|---|---|---:|---:|---|
| `js.errors` <br/>`internal/warpc/js/renderkatex.bundle.js:2` | `cast.ToFloat64E+cast.ToIntE` | `0.14` | `0.81` | no near-duplicate |
| `hugolib.*pageFinder.getPageForRefs` <br/>`hugolib/pagecollections.go:79` | `ShouldListLocal.And+pagePredicates.KindPage+pageMap.getPagesInSection` | `0.14` | `0.69` | no near-duplicate |
| `chromalexers.Get` <br/>`markup/highlight/chromalexers/chromalexers.go:28` | `gohtml.EscapeString+h.cfg` | `0.10` | `0.63` | no near-duplicate |
| `hqt.*stringChecker.Check` <br/>`htesting/hqt/checkers.go:113` | `cmp+quicktest` | `0.14` | `0.65` | no near-duplicate |
| `create.*Client.getOrCreateFileResource` <br/>`resources/resource_factories/create/create.go:201` | `pathInfo.Unnormalized+kinds.KindTerm` | `0.09` | `0.59` | no near-duplicate |
| `esbuild.*BatcherClient.New` <br/>`internal/js/esbuild/batch.go:203` | `depsFinder.Contains+identity.FinderConfig` | `0.09` | `0.58` | no near-duplicate |
| `js.errors` <br/>`internal/warpc/js/renderkatex.bundle.js:2` | `lv.Index+lv.Kind+lv.Len~2` | `0.07` | `0.56` | no near-duplicate |
| `doctree.*NodeShiftTreeWalker[T].Walk` <br/>`hugolib/doctree/nodeshifttree.go:526` | `doctree.LockTypeRead+m.treePages+doctree.NodeShiftTreeWalker` | `0.09` | `0.57` | no near-duplicate |
| `sitesmatrix.VectorIteratorToStore` <br/>`hugolib/sitesmatrix/dimensions.go:241` | `Conf.NewIdentityManager+s.Conf` | `0.11` | `0.58` | no near-duplicate |
| `identity.WalkIdentitiesDeep` <br/>`identity/identity.go:113` | `depsFinder.Contains+identity.FinderConfig` | `0.10` | `0.58` | no near-duplicate |

_1117 more unusual realizations not listed._

A row marked _no near-duplicate_ appears in no reported pair: nothing else in this report explains it, which makes it drift rather than duplication.

---

## Match #1 — Code-shape: `0.8148`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `internal/warpc/genavif/avif.c:79` | `genavif.parse_input_message` | `(?)` | options.compression+options.hint 0.53 |
| **B** | `internal/warpc/genwebp/webp.c:326` | `genwebp.parse_input_message` | `(?)` | options.compression+options.hint 0.51 |

**Explain:** differs by four extra if, three extra assign, 45 extra selector, and 8 more kinds

**Profile A:** `options.compression+options.hint` 1.00 (dominance)

**Profile B:** `options.compression+options.hint` 1.00 (dominance)

**Code similarity:** `wl 0.69  flow 1.00  nesting 1.00  sig 1.00  size 0.79`

**Containment:** `0.92`

**Evidence:** `3145.44` (shape 3141.59, concept 3.85, call 0.00)

**Trophic:** `0.86`

**Shared structure:**

- `75.79` — `depth-1 IF` ×13
- `69.10` — `depth-2 IF` ×11
- `61.57` — `depth-3 IF` ×9

**Concept views:** shape `1.00`, corpus `0.97`, feature `0.97`, a-in-b `0.97`, b-in-a `1.00`

**Shared vocabulary:** `id:strncpy`, `sel:options.compression`, `sel:options.hint`

**Structural overlap:** `0.58` (merge-worthy)

- share 13 callees: [fprintf, json_array_get_count, json_array_get_number, json_object_get_array, json_object_get_number, json_object_get_object, json_object_get_string, json_parse_string, json_value_free, json_value_get_object, json_value_get_type, malloc, strncpy]
- share patterns: [options.compression+options.hint]
- both are leaf functions
- same visibility
- same receiver type: plain functions

---

## Match #2 — Code-shape: `0.7659`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `tpl/internal/go_templates/texttemplate/exec.go:772` | `template.*state.evalCallOld` | `(reflect.Value, reflect.Value, bool, parse.Node, string, []parse.Node, reflect.Value) (reflect.Value)` | fn.IsValid+htmltemplate.Must 0.54, mnt.init+typ.In 0.50 |
| **B** | `tpl/internal/go_templates/texttemplate/hugo_template.go:294` | `template.*state.evalCall` | `(reflect.Value, reflect.Value, bool, parse.Node, string, []parse.Node, reflect.Value, ...reflect.Value) (reflect.Value)` | fn.IsValid+htmltemplate.Must 0.50, mnt.init+typ.In 0.48 |

**Kind:** diverged copy — `*state.evalCallOld` and `*state.evalCall` share the stem `evalCall` in package `template`

**Explain:** differs by seven extra assign, five extra if, one extra defer, and 16 more kinds

**Profile A:** `fn.IsValid+htmltemplate.Must` 1.00 (dominance)

**Profile B:** `fn.IsValid+htmltemplate.Must` 1.00 (dominance)

**Code similarity:** `wl 0.65  flow 1.00  nesting 1.00  sig 0.86  size 0.75`

**Containment:** `0.92` — most of the smaller body's shape is inside the larger

**Evidence:** `2525.60` (shape 2471.14, concept 4.09, call 50.37)

**Trophic:** `0.91`

**Shared structure:**

- `39.71` — `depth-3 CALL` ×6
- `39.71` — `depth-2 CALL` ×6
- `33.98` — `depth-1 CALL` ×6

**Culture:** B realizes `mnt.init+typ.In` atypically (typicality 0.12, concept median 0.25, convention 0.61)

**Concept views:** shape `1.00`, corpus `0.95`, feature `0.95`, a-in-b `0.95`, b-in-a `1.00`

**Shared vocabulary:** `call:github.com/gohugoio/hugo/tpl/internal/go_templates/htmltemplate.Must`, `call:github.com/gohugoio/hugo/tpl/internal/go_templates/texttemplate.Must`, `call:paths.ValidateIdentifier`

**Structural overlap:** `0.65` (merge-worthy)

- share 25 callees: [Elem, Interface, String, append, final.Equal, final.String, fun.Type, goodFunc, isMissing, len, make, reflect.ValueOf, s.at, s.errorf, s.evalArg, s.validateType, safeCall, t.Elem, truth, typ.In, typ.IsVariadic, typ.NumIn, unwrap, v.Interface, v.Type]
- overlapping call-graph neighborhoods (0.65): 36 shared
- share patterns: [fn.IsValid+htmltemplate.Must, mnt.init+typ.In]
- related roles: orchestrator ≈ passthrough (both high fan-out, 0.50)
- same package
- callees do related work (0.69): [mt.Type+hreflect.IsContextType, iter.Next+parse.NumberNode, parse.DotNode+parse.NilNode, parse.StringNode+parse.PipeNode, fn.IsValid+htmltemplate.Must, mnt.init+typ.In]
- same visibility
- same receiver type: state
- call into same packages: [template]

---

## Match #3 — Code-shape: `0.8672`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `internal/warpc/genavif/avif.c:198` | `genavif.write_output_message` | `(?)` | options.compression+options.hint 0.51 |
| **B** | `internal/warpc/genwebp/webp.c:477` | `genwebp.write_output_message` | `(?)` | options.compression+options.hint 0.49 |

**Explain:** differs by 10 extra selector, seven extra call, five extra literal, and 3 more kinds

**Profile A:** `options.compression+options.hint` 1.00 (dominance)

**Profile B:** `options.compression+options.hint` 1.00 (dominance)

**Code similarity:** `wl 0.78  flow 1.00  nesting 1.00  sig 1.00  size 0.82`

**Containment:** `0.96`

**Evidence:** `2073.78` (shape 2070.07, concept 3.71, call 0.00)

**Trophic:** `0.90`

**Shared structure:**

- `57.55` — `depth-3 SEL` ×11
- `57.55` — `depth-2 SEL` ×11
- `50.25` — `depth-3 SEL` ×8

**Concept views:** shape `1.00`, corpus `0.96`, feature `0.96`, a-in-b `0.96`, b-in-a `1.00`

**Shared vocabulary:** `id:strncpy`, `sel:options.compression`, `sel:options.hint`

**Structural overlap:** `0.59` (merge-worthy)

- share 13 callees: [fflush, fprintf, json_array_append_number, json_free_serialized_string, json_object_set_number, json_object_set_string, json_object_set_value, json_serialize_to_string, json_value_free, json_value_get_array, json_value_get_object, json_value_init_array, json_value_init_object]
- share patterns: [options.compression+options.hint]
- both are leaf functions
- same visibility
- same receiver type: plain functions

---

## Match #4 — Code-shape: `0.8294`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `tpl/internal/go_templates/texttemplate/exec.go:682` | `template.*state.evalFieldOld` | `(reflect.Value, string, parse.Node, []parse.Node, reflect.Value, reflect.Value) (reflect.Value)` | fn.IsValid+htmltemplate.Must 0.57 |
| **B** | `tpl/internal/go_templates/texttemplate/hugo_template.go:156` | `template.*state.evalField` | `(reflect.Value, string, parse.Node, []parse.Node, reflect.Value, reflect.Value) (reflect.Value)` | fn.IsValid+htmltemplate.Must 0.54 |

**Kind:** diverged copy — `*state.evalFieldOld` and `*state.evalField` share the stem `evalField` in package `template`

**Explain:** differs by five extra assign, four extra if, three extra declaration, and 9 more kinds

**Profile A:** `fn.IsValid+htmltemplate.Must` 1.00 (dominance)

**Profile B:** `fn.IsValid+htmltemplate.Must` 1.00 (dominance)

**Code similarity:** `wl 0.72  flow 1.00  nesting 1.00  sig 1.00  size 0.75`

**Containment:** `0.93`

**Evidence:** `1752.11` (shape 1736.23, concept 2.36, call 13.53)

**Trophic:** `0.91`

**Shared structure:**

- `39.99` — `depth-1 EXPRSTMT` ×9
- `30.55` — `depth-2 BLOCK` ×6
- `24.59` — `depth-3 EXPRSTMT` ×4

**Concept views:** shape `1.00`, corpus `0.95`, feature `0.95`, a-in-b `0.95`, b-in-a `1.00`

**Shared vocabulary:** `call:github.com/gohugoio/hugo/tpl/internal/go_templates/htmltemplate.Must`, `call:github.com/gohugoio/hugo/tpl/internal/go_templates/texttemplate.Must`, `call:paths.ValidateIdentifier`

**Structural overlap:** `0.73` (merge-worthy)

- share 26 callees: [AssignableTo, Elem, FieldByName, Key, etyp.FieldByName, etyp.Kind, indirect, len, method.IsValid, nameVal.Type, panic, ptr.Addr, ptr.CanAddr, ptr.Kind, ptr.MethodByName, receiver.FieldByIndexErr, receiver.IsValid, receiver.Kind, receiver.MapIndex, receiver.Type, reflect.ValueOf, reflect.Zero, result.IsValid, s.errorf, s.evalCall, tField.IsExported]
- overlapping call-graph neighborhoods (0.59): 13 shared
- share patterns: [fn.IsValid+htmltemplate.Must]
- both are orchestrator functions
- same package
- callees do related work (0.45): [fn.IsValid+htmltemplate.Must, mnt.init+typ.In, hreflect.ToFloat64E+hreflect.ToStringE ≈ fn.Call+mt.Type, parse.DotNode+parse.NilNode ≈ iter.Next+reflect.New]
- same visibility
- same receiver type: state
- call into same packages: [template]

---

## Match #5 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `livereload/livereload.js:2561` | `livereload._createForOfIteratorHelper` | `(?, ?)` | Array.isArray+Symbol.iterator 0.50 |
| **B** | `livereload/livereload.js:2824` | `livereload._createForOfIteratorHelper` | `(?, ?)` | Array.isArray+Symbol.iterator 0.50 |

**Explain:** identical structure, no normalization needed

**Profile A:** `Array.isArray+Symbol.iterator` 1.00 (dominance)

**Profile B:** `Array.isArray+Symbol.iterator` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `830.06` (shape 803.65, concept 3.74, call 22.67)

**Trophic:** `1.00`

**Shared structure:**

- `12.87` — `depth-1 RETURN` ×2
- `11.04` — `depth-1 BLOCK` ×2
- `7.13` — `depth-3 EXPRSTMT`

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `call:./_core.call`, `call:./_core.next`, `call:livereload.e`

**Structural overlap:** `0.76` (merge-worthy)

- share 10 callees: [Array.isArray, F, TypeError, _unsupportedIterableToArray, e, f, it.call, it.next, n, s]
- overlapping call-graph neighborhoods (1.00): 1 shared
- share patterns: [Array.isArray+Symbol.iterator]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions
- call into same packages: [livereload]

---

## Match #6 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `livereload/livereload.js:2561` | `livereload._createForOfIteratorHelper` | `(?, ?)` | Array.isArray+Symbol.iterator 0.50 |
| **B** | `livereload/livereload.js:3096` | `livereload._createForOfIteratorHelper` | `(?, ?)` | Array.isArray+Symbol.iterator 0.50 |

**Explain:** identical structure, no normalization needed

**Profile A:** `Array.isArray+Symbol.iterator` 1.00 (dominance)

**Profile B:** `Array.isArray+Symbol.iterator` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `830.06` (shape 803.65, concept 3.74, call 22.67)

**Trophic:** `1.00`

**Shared structure:**

- `12.87` — `depth-1 RETURN` ×2
- `11.04` — `depth-1 BLOCK` ×2
- `7.13` — `depth-3 EXPRSTMT`

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `call:./_core.call`, `call:./_core.next`, `call:livereload.e`

**Structural overlap:** `0.76` (merge-worthy)

- share 10 callees: [Array.isArray, F, TypeError, _unsupportedIterableToArray, e, f, it.call, it.next, n, s]
- overlapping call-graph neighborhoods (1.00): 1 shared
- share patterns: [Array.isArray+Symbol.iterator]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions
- call into same packages: [livereload]

---

## Match #7 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `livereload/livereload.js:2824` | `livereload._createForOfIteratorHelper` | `(?, ?)` | Array.isArray+Symbol.iterator 0.50 |
| **B** | `livereload/livereload.js:3096` | `livereload._createForOfIteratorHelper` | `(?, ?)` | Array.isArray+Symbol.iterator 0.50 |

**Explain:** identical structure, no normalization needed

**Profile A:** `Array.isArray+Symbol.iterator` 1.00 (dominance)

**Profile B:** `Array.isArray+Symbol.iterator` 1.00 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `830.06` (shape 803.65, concept 3.74, call 22.67)

**Trophic:** `1.00`

**Shared structure:**

- `12.87` — `depth-1 RETURN` ×2
- `11.04` — `depth-1 BLOCK` ×2
- `7.13` — `depth-3 EXPRSTMT`

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `call:./_core.call`, `call:./_core.next`, `call:livereload.e`

**Structural overlap:** `0.76` (merge-worthy)

- share 10 callees: [Array.isArray, F, TypeError, _unsupportedIterableToArray, e, f, it.call, it.next, n, s]
- overlapping call-graph neighborhoods (1.00): 1 shared
- share patterns: [Array.isArray+Symbol.iterator]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions
- call into same packages: [livereload]

---

## Match #8 — Code-shape: `0.9282`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `tpl/internal/go_templates/texttemplate/exec.go:892` | `template.*state._validateType` | `(reflect.Value, reflect.Type) (reflect.Value)` | fn.IsValid+htmltemplate.Must 0.66 |
| **B** | `tpl/internal/go_templates/texttemplate/hugo_template.go:435` | `template.*state.validateType` | `(reflect.Value, reflect.Type) (reflect.Value)` | fn.IsValid+htmltemplate.Must 0.64, Converters.GetHighlighter+c.applyTransformations 0.39 |

**Kind:** diverged copy — `*state._validateType` and `*state.validateType` share the stem `validateType` in package `template`

**Explain:** differs by two extra assign, one extra if, one extra call, and 3 more kinds

**Profile A:** `fn.IsValid+htmltemplate.Must` 1.00 (dominance)

**Profile B:** `fn.IsValid+htmltemplate.Must` 1.00 (dominance)

**Code similarity:** `wl 0.88  flow 1.00  nesting 0.99  sig 1.00  size 0.92`

**Containment:** `0.97`

**Evidence:** `1172.15` (shape 1148.59, concept 2.76, call 20.79)

**Trophic:** `0.97`

**Shared structure:**

- `24.59` — `depth-1 CALL` ×4
- `22.97` — `depth-0 CALL` ×4
- `15.07` — `depth-3 BLOCK` ×2

**Concept views:** shape `0.50`, corpus `0.63`, feature `0.59`, a-in-b `0.97`, b-in-a `0.60`

**Shared vocabulary:** `call:github.com/gohugoio/hugo/tpl/internal/go_templates/htmltemplate.Must`, `call:github.com/gohugoio/hugo/tpl/internal/go_templates/texttemplate.Must`, `call:paths.ValidateIdentifier`

**Structural overlap:** `0.54` (merge-worthy)

- share 14 callees: [AssignableTo, Elem, canBeNil, reflect.PointerTo, reflect.ValueOf, reflect.Zero, s.errorf, value.Addr, value.CanAddr, value.Elem, value.IsNil, value.IsValid, value.Kind, value.Type]
- overlapping call-graph neighborhoods (0.07): 3 shared
- share patterns: [fn.IsValid+htmltemplate.Must]
- same package
- callees do related work (0.71): [hreflect.ToFloat64E+hreflect.ToStringE, iter.Next+reflect.New, fn.IsValid+htmltemplate.Must]
- same visibility
- same receiver type: state
- call into same packages: [template]

---

## Match #9 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `resources/image.go:198` | `resources.*imageResource.getImageMetaInfoCacheTargetPath` | `() (string)` | i.getResourcePaths+i.hash 0.53, ResourceSpec.FileCaches+c.isExpired 0.52, i.ReadSeekCloser+i.Format 0.50 |
| **B** | `resources/image.go:501` | `resources.*imageResource.getImageMetaCacheTargetPath` | `() (string)` | i.getResourcePaths+i.hash 0.54, ResourceSpec.FileCaches+c.isExpired 0.52, i.ReadSeekCloser+i.Format 0.50 |

**Explain:** identical after rename

**Profile A:** `i.ReadSeekCloser+i.Format` 0.90, `i.getResourcePaths+i.hash` 0.10 (dominance)

**Profile B:** `i.ReadSeekCloser+i.Format` 0.90, `i.getResourcePaths+i.hash` 0.10 (dominance)

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 0.99`

**Containment:** `1.00`

**Evidence:** `478.63` (shape 435.26, concept 7.80, call 35.56)

**Trophic:** `1.00`

**Shared structure:**

- `7.53` — `depth-3 CALL`
- `7.53` — `depth-3 ASSIGN`
- `7.53` — `depth-3 CALL`

**Concept views:** shape `1.00`, corpus `1.00`, feature `1.00`, a-in-b `1.00`, b-in-a `1.00`

**Shared vocabulary:** `sel:i.getResourcePaths`, `sel:i.hash`, `sel:i.ReadSeekCloser`

**Structural overlap:** `0.88` (merge-worthy)

- share 8 callees: [df.TargetPath, fmt.Sprintf, hashing.HashStringHex, i.getResourcePaths, i.getSpec, i.hash, i.size, paths.FileAndExt]
- overlapping call-graph neighborhoods (0.92): 23 shared
- share patterns: [ResourceSpec.FileCaches+c.isExpired, i.ReadSeekCloser+i.Format, i.getResourcePaths+i.hash]
- both are orchestrator functions
- same package
- callers do related work (0.98): [i.ReadSeekCloser+i.Format, Body.Close+api.OnLoadArgs, ResourceSpec.FileCaches+c.isExpired]
- callees do related work (1.00): [paths.fileAndExt+unicode, strconv.FormatUint+tpl, paths.TargetPath+l.paths, hashing.HashUint64+hashing, spec.Cfg+Cfg.BaseURL, path.Split+path.Join, l.spec+mime, fw.Close+helpers.OpenFilesForWriting]
- same visibility
- same receiver type: imageResource
- called from same packages: [resources]
- call into same packages: [hashing, paths, resources]

---

## Match #10 — Code-shape: `1.0000`

| | Location | Function | Signature | Concepts |
|---|---|---|---|---|
| **A** | `internal/warpc/js/greet.bundle.js:1` | `js.w` | `(?)` | — |
| **B** | `internal/warpc/js/renderkatex.bundle.js:1` | `js.Wt` | `(?)` | — |

**Explain:** identical structure, no normalization needed

**Code similarity:** `wl 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Containment:** `1.00`

**Evidence:** `856.14` (shape 856.14, concept 0.00, call 0.00)

**Trophic:** `1.00`

**Shared structure:**

- `18.44` — `depth-3 NODE` ×3
- `18.44` — `depth-2 NODE` ×3
- `18.44` — `depth-1 NODE` ×3

**Concept views:** shape `0.00`, corpus `0.00`, feature `0.00`, a-in-b `0.00`, b-in-a `0.00`

**Structural overlap:** `0.47` (merge-worthy)

- share 9 callees: [Error, JSON.parse, TextDecoder, Uint8Array, console.log, decode, includes, r, readSync]
- both are leaf functions
- same package
- same visibility
- same receiver type: plain functions
- call into same packages: [js]

---

## Families

515 families, 1894 functions in a family, largest 49 members; 6390 edges scored here that retrieval never proposed

### Family 1 — 34 members, every pair `>= 0.34` code-shape, evidence `114410`  (317 edges scored here)

_Not drawn: 34 members is 561 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `config/allconfig/docshelper.go:23` | `allconfig.init` | `()` | docshelper.AddDocProviderFu…+docshelper.DocProvider 0.50 |
| `tpl/cast/docshelper.go:26` | `cast.init` | `()` | docshelper.AddDocProviderFu…+docshelper.DocProvider 0.48 |
| `tpl/cast/init.go:25` | `cast.init` | `()` | d.Site+d.Conf 0.50, h.PathSpec+PathSpec.BaseFs 0.29 |
| `tpl/collections/init.go:25` | `collections.init` | `()` | — |
| `tpl/compare/init.go:26` | `compare.init` | `()` | d.Site+d.Conf 0.46 |
| `tpl/crypto/init.go:25` | `crypto.init` | `()` | d.Site+d.Conf 0.52, h.PathSpec+PathSpec.BaseFs 0.29 |
| `tpl/css/css.go:229` | `css.init` | `()` | ResourceSpec.FileCaches+c.isExpired 0.48, Converters.GetHighlighter+c.applyTransformations 0.43 |
| `tpl/debug/init.go:25` | `debug.init` | `()` | d.Site+d.Conf 0.53, openapi3+resources 0.33, h.PathSpec+PathSpec.BaseFs 0.31 |
| `tpl/diagrams/init.go:26` | `diagrams.init` | `()` | d.Site+d.Conf 0.57, s.TemplateStore+context.Background 0.45, openapi3+resources 0.41 |
| `tpl/encoding/init.go:25` | `encoding.init` | `()` | d.Site+d.Conf 0.55, openapi3+resources 0.33, h.PathSpec+PathSpec.BaseFs 0.31 |

_24 more members not listed._

### Family 2 — 40 members, every pair `>= 0.35` code-shape, evidence `15002`  (690 edges scored here)

_Not drawn: 40 members is 780 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `internal/warpc/deps/parson/parson.c:309` | `parson.hex_char_to_int` | `(?)` | — |
| `internal/warpc/deps/parson/parson.c:320` | `parson.parse_utf16_hex` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:351` | `parson.verify_utf8_sequence` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:456` | `parson.json_object_init` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:526` | `parson.json_object_grow_and_rehash` | `(?)` | — |
| `internal/warpc/deps/parson/parson.c:556` | `parson.json_object_get_cell_ix` | `(?, ?, ?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:587` | `parson.json_object_add` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:641` | `parson.json_object_remove_internal` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:701` | `parson.json_object_dotremove_internal` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:747` | `parson.json_array_resize` | `(?, ?)` | — |

_30 more members not listed._

### Family 3 — 37 members, every pair `>= 0.34` code-shape, evidence `14706`  (582 edges scored here)

_Not drawn: 37 members is 666 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `internal/warpc/deps/parson/parson.c:320` | `parson.parse_utf16_hex` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:351` | `parson.verify_utf8_sequence` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:456` | `parson.json_object_init` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:526` | `parson.json_object_grow_and_rehash` | `(?)` | — |
| `internal/warpc/deps/parson/parson.c:556` | `parson.json_object_get_cell_ix` | `(?, ?, ?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:587` | `parson.json_object_add` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:641` | `parson.json_object_remove_internal` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:701` | `parson.json_object_dotremove_internal` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:747` | `parson.json_array_resize` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:808` | `parson.parse_utf16` | `(?, ?)` | — |

_27 more members not listed._

### Family 4 — 33 members, every pair `>= 0.35` code-shape, evidence `14607`  (459 edges scored here)

_Not drawn: 33 members is 528 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `internal/warpc/deps/parson/parson.c:320` | `parson.parse_utf16_hex` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:351` | `parson.verify_utf8_sequence` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:456` | `parson.json_object_init` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:526` | `parson.json_object_grow_and_rehash` | `(?)` | — |
| `internal/warpc/deps/parson/parson.c:556` | `parson.json_object_get_cell_ix` | `(?, ?, ?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:587` | `parson.json_object_add` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:641` | `parson.json_object_remove_internal` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:701` | `parson.json_object_dotremove_internal` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:747` | `parson.json_array_resize` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:808` | `parson.parse_utf16` | `(?, ?)` | — |

_23 more members not listed._

### Family 5 — 36 members, every pair `>= 0.34` code-shape, evidence `14419`  (553 edges scored here)

_Not drawn: 36 members is 630 connections. Every one of them holds — that is what makes this a family._

| Location | Function | Signature | Concepts |
|---|---|---|---|
| `internal/warpc/deps/parson/parson.c:309` | `parson.hex_char_to_int` | `(?)` | — |
| `internal/warpc/deps/parson/parson.c:320` | `parson.parse_utf16_hex` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:351` | `parson.verify_utf8_sequence` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:456` | `parson.json_object_init` | `(?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:526` | `parson.json_object_grow_and_rehash` | `(?)` | — |
| `internal/warpc/deps/parson/parson.c:556` | `parson.json_object_get_cell_ix` | `(?, ?, ?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:587` | `parson.json_object_add` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:641` | `parson.json_object_remove_internal` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:701` | `parson.json_object_dotremove_internal` | `(?, ?, ?)` | — |
| `internal/warpc/deps/parson/parson.c:747` | `parson.json_array_resize` | `(?, ?)` | — |

_26 more members not listed._

_510 more families not listed._

_3 component(s) too large or too dense to enumerate (sizes 139, 145, 642); their families are not reported._


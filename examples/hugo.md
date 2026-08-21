# hugo

static site generator; a large monolith with heavy template and resource subsystems

**What this rung shows:** habitats — many packages large enough to have a temperature of their own

| | |
|---|---|
| Corpus | [hugo](https://github.com/gohugoio/hugo) |
| Pinned at | `v0.165.0` (`76a5e1880ab46688155b02e99bab9be2a6134492`) |
| Project since | 2013 |
| doppel | `e61ea20` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 8 concepts modeled, 350 associations, 57 unusual realizations
Habitats: 126 modeled, 665 misfits; most uniform partials (norm 0.98), most diverse page (norm 0.61)
Conventions: strongest error_wrapping (0.64), loosest serialization (0.52)
Ecosystems: 1999 profiled (1410 dominance, 589 coalition, 0 conflict, 0 weak)
Found 5460 functions. Retrieving candidates...
Retrieval: shape 2263, concept 4068, call 8309 -> 13852 unique pairs
  concept-only 28.1%  call-only 54.5%  suppressed-shape functions: 71  large identity buckets: 0  surviving patterns: 29668
Running structural comparison on 13852 pairs...
```

# Code Similarity Report

**Functions analyzed:** 5460 | **Threshold:** 0.60 | **Pairs found:** 10

---

## Match #1 — Code-shape: `0.8459`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tpl/internal/go_templates/texttemplate/exec.go:772` | `template.*state.evalCallOld` | ` ` | validation, error_wrapping |
| **B** | `tpl/internal/go_templates/texttemplate/hugo_template.go:294` | `template.*state.evalCall` | ` ` | validation, error_wrapping |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.78  flow 1.00  nesting 1.00  sig 0.86  size 0.75`

**Evidence:** `3378.70` (shape 3324.07, concept 4.61, call 50.02)

**Trophic:** `0.87`

**Shared structure:**

- `20.68` — `if(call:IsVariadic)`
- `17.98` — `do(call:errorf)`
- `15.17` — `assign=(call:evalArg)`

**Culture:** B realizes `error_wrapping` atypically (typicality 0.16, concept median 0.36, convention 0.64)

**Culture:** B realizes `validation` atypically (typicality 0.16, concept median 0.34, convention 0.59)

**Structural overlap:** `0.68` (merge-worthy)

- share 25 callees: [Elem, Interface, String, append, final.Equal, final.String, fun.Type, goodFunc, isMissing, len, make, reflect.ValueOf, s.at, s.errorf, s.evalArg, s.validateType, safeCall, t.Elem, truth, typ.In, typ.IsVariadic, typ.NumIn, unwrap, v.Interface, v.Type]
- overlapping call-graph neighborhoods (0.65): 36 shared
- share patterns: [error_wrapping, validation]
- related roles: orchestrator ≈ passthrough (both high fan-out, 0.50)
- same package
- callees do related work (1.00): [validation, mapping]
- same visibility
- same receiver type: state
- call into same packages: [template]

---

## Match #2 — Code-shape: `0.9248`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tpl/internal/go_templates/texttemplate/exec.go:682` | `template.*state.evalFieldOld` | ` ` | validation |
| **B** | `tpl/internal/go_templates/texttemplate/hugo_template.go:156` | `template.*state.evalField` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.88  flow 1.00  nesting 1.00  sig 1.00  size 0.75`

**Evidence:** `1942.39` (shape 1926.48, concept 2.49, call 13.43)

**Trophic:** `0.90`

**Shared structure:**

- `40.45` — `do(call:errorf)`
- `12.97` — `assign:=(call:FieldByName)`
- `7.59` — `seq[ assign:=(call:FieldByIndexErr) ; if(unary) ]`

**Structural overlap:** `0.77` (merge-worthy)

- share 26 callees: [AssignableTo, Elem, FieldByName, Key, etyp.FieldByName, etyp.Kind, indirect, len, method.IsValid, nameVal.Type, panic, ptr.Addr, ptr.CanAddr, ptr.Kind, ptr.MethodByName, receiver.FieldByIndexErr, receiver.IsValid, receiver.Kind, receiver.MapIndex, receiver.Type, reflect.ValueOf, reflect.Zero, result.IsValid, s.errorf, s.evalCall, tField.IsExported]
- overlapping call-graph neighborhoods (0.59): 13 shared
- share patterns: [validation]
- both are orchestrator functions
- same package
- callees do related work (1.00): [validation, error_wrapping]
- same visibility
- same receiver type: state
- call into same packages: [template]

---

## Match #3 — Code-shape: `0.8498`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tpl/math/init.go:26` | `math.init` | `—` | — |
| **B** | `tpl/strings/init.go:25` | `strings.init` | `—` | — |

**Code similarity:** `ast 0.77  flow 0.94  nesting 0.95  sig 1.00  size 0.92`

**Evidence:** `3171.02` (shape 3160.74, concept 0.00, call 10.29)

**Trophic:** `0.81`

**Shared structure:**

- `149.49` — `do(call:AddMethodMapping)`
- `146.74` — `seq[ do(call:AddMethodMapping) ; do(call:AddMethodMapping) ]`
- `5.06` — `seq[ assign:=(call:New) ; assign:=(unary) ]`

**Habitat:** A fits poorly in `math` (fit 0.24, package norm 0.87)

**Habitat:** B fits poorly in `strings` (fit 0.17, package norm 0.84)

**Structural overlap:** `0.48` (merge-worthy)

- share 3 callees: [New, internal.AddTemplateFuncsNamespace, ns.AddMethodMapping]
- overlapping call-graph neighborhoods (0.97): 32 shared
- both are orchestrator functions
- same visibility
- same receiver type: plain functions
- call into same packages: [internal]

---

## Match #4 — Code-shape: `0.9564`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tpl/collections/init.go:25` | `collections.init` | `—` | — |
| **B** | `tpl/strings/init.go:25` | `strings.init` | `—` | — |

**Code similarity:** `ast 0.93  flow 1.00  nesting 1.00  sig 1.00  size 0.86`

**Evidence:** `2847.05` (shape 2836.76, concept 0.00, call 10.29)

**Trophic:** `0.75`

**Shared structure:**

- `134.54` — `do(call:AddMethodMapping)`
- `131.56` — `seq[ do(call:AddMethodMapping) ; do(call:AddMethodMapping) ]`
- `5.06` — `seq[ assign:=(call:New) ; assign:=(unary) ]`

**Habitat:** B fits poorly in `strings` (fit 0.17, package norm 0.84)

**Structural overlap:** `0.52` (merge-worthy)

- share 3 callees: [New, internal.AddTemplateFuncsNamespace, ns.AddMethodMapping]
- overlapping call-graph neighborhoods (0.91): 32 shared
- both are orchestrator functions
- callees do related work (1.00): [caching]
- same visibility
- same receiver type: plain functions
- call into same packages: [internal]

---

## Match #5 — Code-shape: `0.7425`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `resources/image.go:82` | `resources.*imageResource.newExifInfoFn` | ` ` | caching, serialization, file_io |
| **B** | `resources/image.go:125` | `resources.*imageResource.newMetaInfoFn` | ` ` | caching, serialization, file_io |

**Profile A:** `file_io` 0.94, `caching` 0.06 (dominance)

**Profile B:** `file_io` 0.94, `caching` 0.06 (dominance)

**Code similarity:** `ast 0.82  flow 1.00  nesting 1.00  sig 0.00  size 0.96`

**Evidence:** `1154.93` (shape 1094.14, concept 7.07, call 53.72)

**Trophic:** `0.96`

**Shared structure:**

- `7.59` — `seq[ assign:=(call:NewEncoder) ; return(call:Encode) ]`
- `7.59` — `seq[ assign:=(call:ReadOrCreate) ; return(id,id) ]`
- `7.59` — `seq[ assign:=(call:ToImageMetaImageFormatFormat) ; if(bin:==(id,unary)) ]`

**Habitat:** A fits poorly in `resources` (fit 0.27, package norm 0.75)

**Habitat:** B fits poorly in `resources` (fit 0.29, package norm 0.75)

**Structural overlap:** `0.89` (merge-worthy)

- share 14 callees: [InternalResourceSourcePathBestEffort, ReadOrCreate, ToImageMetaImageFormatFormat, Warnf, enc.Encode, f.Close, i.Key, i.ReadSeekCloser, i.getSpec, io.ReadAll, json.NewEncoder, json.Unmarshal, sync.OnceValues, w.Close]
- share 1 callers: [resources.newImageResource]
- overlapping call-graph neighborhoods (0.55): 24 shared
- share patterns: [caching, file_io, serialization]
- both are orchestrator functions
- same package
- callees do related work (1.00): [caching, concurrency]
- same visibility
- same receiver type: imageResource
- called from same packages: [resources]
- call into same packages: [filecache, images, resources]

---

## Match #6 — Code-shape: `0.9533`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tpl/internal/go_templates/texttemplate/exec.go:892` | `template.*state._validateType` | ` ` | validation |
| **B** | `tpl/internal/go_templates/texttemplate/hugo_template.go:435` | `template.*state.validateType` | ` ` | validation, mapping |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.92  flow 1.00  nesting 0.99  sig 1.00  size 0.92`

**Evidence:** `1333.03` (shape 1309.90, concept 2.49, call 20.65)

**Trophic:** `0.97`

**Shared structure:**

- `13.48` — `do(call:errorf)`
- `12.67` — `return(call:ValueOf)`
- `10.67` — `assign=(call:Elem)`

**Structural overlap:** `0.49` (merge-worthy)

- share 14 callees: [AssignableTo, Elem, canBeNil, reflect.PointerTo, reflect.ValueOf, reflect.Zero, s.errorf, value.Addr, value.CanAddr, value.Elem, value.IsNil, value.IsValid, value.Kind, value.Type]
- overlapping call-graph neighborhoods (0.07): 3 shared
- share patterns: [validation]
- same package
- same visibility
- same receiver type: state
- call into same packages: [template]

---

## Match #7 — Code-shape: `0.8113`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `resources/resource_transformers/babel/babel.go:115` | `babel.*babelTransformation.Transform` | ` ` | mapping, concurrency, error_wrapping, file_io |
| **B** | `resources/resource_transformers/cssjs/postcss.go:146` | `cssjs.*postcssTransformation.Transform` | ` ` | mapping, concurrency, file_io |

**Profile A:** `file_io` 1.00 (dominance)

**Profile B:** `file_io` 1.00 (dominance)

**Code similarity:** `ast 0.69  flow 1.00  nesting 1.00  sig 1.00  size 0.91`

**Evidence:** `2034.59` (shape 1921.78, concept 5.81, call 106.99)

**Trophic:** `0.77`

**Shared structure:**

- `14.73` — `seq[ assign=(call:append) ; assign=(call:append) ]`
- `14.36` — `assign=(call:ResolveJSConfigFile)`
- `14.36` — `if(call:IsNotFound)`

**Culture:** A realizes `concurrency` atypically (typicality 0.13, concept median 0.35, convention 0.60)

**Culture:** B realizes `concurrency` atypically (typicality 0.13, concept median 0.35, convention 0.60)

**Culture:** A realizes `mapping` atypically (typicality 0.14, concept median 0.31, convention 0.63)

**Culture:** B realizes `mapping` atypically (typicality 0.14, concept median 0.31, convention 0.63)

**Habitat:** A fits poorly in `babel` (fit 0.35, package norm 0.85)

**Structural overlap:** `0.53` (merge-worthy)

- share 23 callees: [BaseConfig, InfoCommand, ResolveJSConfigFile, append, cmd.Run, cmd.StdinPipe, errBuf.String, ex.Npx, filepath.Clean, filepath.IsAbs, fmt.Errorf, hexec.IsNotFound, hexec.WithDir, hexec.WithEnviron, hexec.WithStderr, hexec.WithStdout, hugo.GetExecEnviron, infol.Logf, io.Copy, io.MultiWriter, len, loggers.LevelLoggerToWriter, stdin.Close]
- overlapping call-graph neighborhoods (0.39): 34 shared
- share patterns: [concurrency, file_io, mapping]
- both are orchestrator functions
- callees do related work (0.40): [caching]
- same visibility
- both are methods, on *babelTransformation and *postcssTransformation
- call into same packages: [allconfig, filesystems, hexec, hugo, loggers]

---

## Match #8 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `resources/image.go:198` | `resources.*imageResource.getImageMetaInfoCacheTargetPath` | ` ` | caching |
| **B** | `resources/image.go:501` | `resources.*imageResource.getImageMetaCacheTargetPath` | ` ` | caching |

**Profile A:** `caching` 1.00 (dominance)

**Profile B:** `caching` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 0.99`

**Evidence:** `597.02` (shape 560.06, concept 1.65, call 35.31)

**Trophic:** `1.00`

**Shared structure:**

- `7.59` — `seq[ assign:=(call:FileAndExt) ; assign:=(call:hash) ]`
- `7.59` — `seq[ assign:=(call:HashStringHex) ; assign=(call:Sprintf) ]`
- `7.59` — `seq[ assign:=(call:getResourcePaths) ; assign:=(call:FileAndExt) ]`

**Structural overlap:** `0.83` (merge-worthy)

- share 8 callees: [df.TargetPath, fmt.Sprintf, hashing.HashStringHex, i.getResourcePaths, i.getSpec, i.hash, i.size, paths.FileAndExt]
- overlapping call-graph neighborhoods (0.92): 23 shared
- share patterns: [caching]
- both are orchestrator functions
- same package
- callers do related work (1.00): [serialization, file_io, caching]
- same visibility
- same receiver type: imageResource
- called from same packages: [resources]
- call into same packages: [hashing, paths, resources]

---

## Match #9 — Code-shape: `0.7598`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `resources/resource_transformers/cssjs/postcss.go:146` | `cssjs.*postcssTransformation.Transform` | ` ` | mapping, concurrency, file_io |
| **B** | `resources/resource_transformers/cssjs/tailwindcss.go:80` | `cssjs.*tailwindcssTransformation.Transform` | ` ` | mapping, concurrency, file_io |

**Profile A:** `file_io` 1.00 (dominance)

**Profile B:** `file_io` 1.00 (dominance)

**Code similarity:** `ast 0.61  flow 0.98  nesting 0.99  sig 1.00  size 0.75`

**Evidence:** `1583.07` (shape 1466.20, concept 5.81, call 111.05)

**Trophic:** `0.74`

**Shared structure:**

- `14.73` — `seq[ assign=(call:append) ; assign=(call:append) ]`
- `14.36` — `if(call:IsNotFound)`
- `7.59` — `seq[ assign:=(call:LevelLoggerToWriter) ; assign:=(sel) ]`

**Culture:** A realizes `concurrency` atypically (typicality 0.13, concept median 0.35, convention 0.60)

**Culture:** B realizes `concurrency` atypically (typicality 0.13, concept median 0.35, convention 0.60)

**Culture:** A realizes `mapping` atypically (typicality 0.14, concept median 0.31, convention 0.63)

**Culture:** B realizes `mapping` atypically (typicality 0.15, concept median 0.31, convention 0.63)

**Structural overlap:** `0.72` (merge-worthy)

- share 21 callees: [BaseConfig, InfoCommand, append, cmd.Run, cmd.StdinPipe, errBuf.String, ex.Npx, hexec.IsNotFound, hexec.WithDir, hexec.WithEnviron, hexec.WithStderr, hexec.WithStdout, hugo.GetExecEnviron, imp.resolve, imp.toFileError, io.Copy, io.MultiWriter, loggers.LevelLoggerToWriter, newImportResolver, options.toArgs, stdin.Close]
- overlapping call-graph neighborhoods (0.90): 44 shared
- share patterns: [concurrency, file_io, mapping]
- both are orchestrator functions
- same package
- callees do related work (1.00): [file_io, caching]
- same visibility
- both are methods, on *postcssTransformation and *tailwindcssTransformation
- call into same packages: [allconfig, cssjs, hexec, hugo, loggers]

---

## Match #10 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `internal/js/esbuild/batch.go:1021` | `esbuild.*scriptGroup.Runner` | ` ` | validation, concurrency |
| **B** | `internal/js/esbuild/batch.go:1050` | `esbuild.*scriptGroup.Script` | ` ` | validation, concurrency |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `507.68` (shape 496.69, concept 3.99, call 7.00)

**Trophic:** `1.00`

**Shared structure:**

- `13.34` — `return(call:Get)`
- `7.59` — `seq[ assign:=(call:scriptID) ; if(id) ]`
- `7.59` — `seq[ defer(call:Unlock) ; assign:=(call:scriptID) ]`

**Structural overlap:** `0.81` (merge-worthy)

- share 8 callees: [Get, Lock, Unlock, ValidateBatchID, panic, s.key, scriptID, v.Get]
- overlapping call-graph neighborhoods (1.00): 4 shared
- share patterns: [concurrency, validation]
- both are leaf functions
- same package
- callees do related work (1.00): [validation]
- same visibility
- same receiver type: scriptGroup
- call into same packages: [esbuild]

---


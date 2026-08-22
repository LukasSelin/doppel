# hugo

static site generator; a large monolith with heavy template and resource subsystems

**What this rung shows:** habitats — many packages large enough to have a temperature of their own

| | |
|---|---|
| Corpus | [hugo](https://github.com/gohugoio/hugo) |
| Pinned at | `v0.165.0` (`76a5e1880ab46688155b02e99bab9be2a6134492`) |
| Project since | 2013 |
| doppel | `c6cea0e` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 8 concepts modeled, 350 associations, 57 unusual realizations
Habitats: 126 modeled, 538 misfits (121 excused by subsystem), 31 subsystems; most uniform partials (norm 0.98), most diverse page (norm 0.61)
Conventions: strongest error_wrapping (0.64), loosest serialization (0.52)
Ecosystems: 1997 profiled (1408 dominance, 589 coalition, 0 conflict, 0 weak)
Found 5438 functions. Retrieving candidates...
Retrieval: shape 2195, concept 4057, call 8265 -> 13753 unique pairs
  concept-only 28.2%  call-only 54.7%  suppressed-shape functions: 70  large identity buckets: 0  surviving patterns: 31542
Running structural comparison on 13753 pairs...
Families: 306 over 480 components, 939 functions in a family, 623 edges completed
```

# Code Similarity Report

**Functions analyzed:** 5438 | **Threshold:** 0.60 | **Pairs found:** 10

---

## Match #1 — Code-shape: `0.8459`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `tpl/internal/go_templates/texttemplate/exec.go:772` | `template.*state.evalCallOld` | `(reflect.Value, reflect.Value, bool, parse.Node, string, []parse.Node, reflect.Value) (reflect.Value)` | validation, error_wrapping |
| **B** | `tpl/internal/go_templates/texttemplate/hugo_template.go:294` | `template.*state.evalCall` | `(reflect.Value, reflect.Value, bool, parse.Node, string, []parse.Node, reflect.Value, ...reflect.Value) (reflect.Value)` | validation, error_wrapping |

**Kind:** diverged copy — `*state.evalCallOld` and `*state.evalCall` share the stem `evalCall` in package `template`

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.78  flow 1.00  nesting 1.00  sig 0.86  size 0.75`

**Evidence:** `3741.11` (shape 3686.51, concept 4.60, call 50.00)

**Trophic:** `0.88`

**Shared structure:**

- `41.32` — `flow:call:Type→call:NumIn`
- `34.44` — `flow:call:Type→call:In`
- `22.74` — `flow:call:Interface→return`

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
| **A** | `tpl/internal/go_templates/texttemplate/exec.go:682` | `template.*state.evalFieldOld` | `(reflect.Value, string, parse.Node, []parse.Node, reflect.Value, reflect.Value) (reflect.Value)` | validation |
| **B** | `tpl/internal/go_templates/texttemplate/hugo_template.go:156` | `template.*state.evalField` | `(reflect.Value, string, parse.Node, []parse.Node, reflect.Value, reflect.Value) (reflect.Value)` | validation |

**Kind:** diverged copy — `*state.evalFieldOld` and `*state.evalField` share the stem `evalField` in package `template`

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.88  flow 1.00  nesting 1.00  sig 1.00  size 0.75`

**Evidence:** `2230.52` (shape 2214.61, concept 2.49, call 13.42)

**Trophic:** `0.89`

**Shared structure:**

- `40.40` — `do(call:errorf)`
- `35.22` — `flow:param→call:errorf`
- `34.44` — `flow:param→call:evalCall`

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
| **A** | `tpl/math/init.go:26` | `math.init` | `()` | — |
| **B** | `tpl/strings/init.go:25` | `strings.init` | `()` | — |

**Code similarity:** `ast 0.77  flow 0.94  nesting 0.95  sig 1.00  size 0.92`

**Evidence:** `3172.59` (shape 3162.31, concept 0.00, call 10.28)

**Trophic:** `0.81`

**Shared structure:**

- `149.32` — `do(call:AddMethodMapping)`
- `146.58` — `seq[ do(call:AddMethodMapping) ; do(call:AddMethodMapping) ]`
- `5.05` — `seq[ assign:=(call:New) ; assign:=(unary) ]`

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
| **A** | `tpl/collections/init.go:25` | `collections.init` | `()` | — |
| **B** | `tpl/strings/init.go:25` | `strings.init` | `()` | — |

**Code similarity:** `ast 0.93  flow 1.00  nesting 1.00  sig 1.00  size 0.86`

**Evidence:** `2848.97` (shape 2838.69, concept 0.00, call 10.28)

**Trophic:** `0.75`

**Shared structure:**

- `134.39` — `do(call:AddMethodMapping)`
- `131.42` — `seq[ do(call:AddMethodMapping) ; do(call:AddMethodMapping) ]`
- `5.05` — `seq[ assign:=(call:New) ; assign:=(unary) ]`

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
| **A** | `resources/image.go:82` | `resources.*imageResource.newExifInfoFn` | `() (func() (*meta.ExifInfo, error))` | caching, serialization, file_io |
| **B** | `resources/image.go:125` | `resources.*imageResource.newMetaInfoFn` | `() (func() (*meta.MetaInfo, error))` | caching, serialization, file_io |

**Profile A:** `file_io` 0.94, `caching` 0.06 (dominance)

**Profile B:** `file_io` 0.94, `caching` 0.06 (dominance)

**Code similarity:** `ast 0.82  flow 1.00  nesting 1.00  sig 0.00  size 0.96`

**Evidence:** `1230.71` (shape 1169.95, concept 7.07, call 53.69)

**Trophic:** `0.97`

**Shared structure:**

- `17.63` — `flow:call:ReadAll→cond`
- `17.63` — `flow:call:ReadAll→return`
- `7.58` — `seq[ assign:=(call:NewEncoder) ; return(call:Encode) ]`

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
| **A** | `tpl/internal/go_templates/texttemplate/exec.go:892` | `template.*state._validateType` | `(reflect.Value, reflect.Type) (reflect.Value)` | validation |
| **B** | `tpl/internal/go_templates/texttemplate/hugo_template.go:435` | `template.*state.validateType` | `(reflect.Value, reflect.Type) (reflect.Value)` | validation, mapping |

**Kind:** diverged copy — `*state._validateType` and `*state.validateType` share the stem `validateType` in package `template`

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.92  flow 1.00  nesting 0.99  sig 1.00  size 0.92`

**Evidence:** `1457.44` (shape 1434.32, concept 2.49, call 20.63)

**Trophic:** `0.96`

**Shared structure:**

- `27.07` — `flow:param→call:Type`
- `25.93` — `flow:param→call:AssignableTo`
- `13.47` — `do(call:errorf)`

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
| **A** | `resources/resource_transformers/babel/babel.go:115` | `babel.*babelTransformation.Transform` | `(*resources.ResourceTransformationCtx) (error)` | mapping, concurrency, error_wrapping, file_io |
| **B** | `resources/resource_transformers/cssjs/postcss.go:146` | `cssjs.*postcssTransformation.Transform` | `(*resources.ResourceTransformationCtx) (error)` | mapping, concurrency, file_io |

**Kind:** interface implementations — both implement `Transform(*resources.ResourceTransformationCtx) (error)` on `*babelTransformation` and `*postcssTransformation`, sibling packages `babel` and `cssjs`

**Profile A:** `file_io` 1.00 (dominance)

**Profile B:** `file_io` 1.00 (dominance)

**Code similarity:** `ast 0.69  flow 1.00  nesting 1.00  sig 1.00  size 0.91`

**Evidence:** `2165.06` (shape 2052.31, concept 5.82, call 106.93)

**Trophic:** `0.77`

**Shared structure:**

- `30.32` — `flow:call:ResolveJSConfigFile→cond`
- `14.72` — `seq[ assign=(call:append) ; assign=(call:append) ]`
- `14.35` — `assign=(call:ResolveJSConfigFile)`

**Culture:** A realizes `concurrency` atypically (typicality 0.13, concept median 0.35, convention 0.60)

**Culture:** B realizes `concurrency` atypically (typicality 0.13, concept median 0.35, convention 0.60)

**Culture:** A realizes `mapping` atypically (typicality 0.14, concept median 0.31, convention 0.63)

**Culture:** B realizes `mapping` atypically (typicality 0.14, concept median 0.31, convention 0.63)

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
| **A** | `resources/image.go:198` | `resources.*imageResource.getImageMetaInfoCacheTargetPath` | `() (string)` | caching |
| **B** | `resources/image.go:501` | `resources.*imageResource.getImageMetaCacheTargetPath` | `() (string)` | caching |

**Profile A:** `caching` 1.00 (dominance)

**Profile B:** `caching` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 0.99`

**Evidence:** `633.62` (shape 596.69, concept 1.64, call 35.29)

**Trophic:** `1.00`

**Shared structure:**

- `7.58` — `seq[ assign:=(call:FileAndExt) ; assign:=(call:hash) ]`
- `7.58` — `seq[ assign:=(call:HashStringHex) ; assign=(call:Sprintf) ]`
- `7.58` — `seq[ assign:=(call:getResourcePaths) ; assign:=(call:FileAndExt) ]`

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
| **A** | `resources/resource_transformers/cssjs/postcss.go:146` | `cssjs.*postcssTransformation.Transform` | `(*resources.ResourceTransformationCtx) (error)` | mapping, concurrency, file_io |
| **B** | `resources/resource_transformers/cssjs/tailwindcss.go:80` | `cssjs.*tailwindcssTransformation.Transform` | `(*resources.ResourceTransformationCtx) (error)` | mapping, concurrency, file_io |

**Kind:** interface implementations — both implement `Transform(*resources.ResourceTransformationCtx) (error)` on `*postcssTransformation` and `*tailwindcssTransformation`, in package `cssjs`

**Profile A:** `file_io` 1.00 (dominance)

**Profile B:** `file_io` 1.00 (dominance)

**Code similarity:** `ast 0.61  flow 0.98  nesting 0.99  sig 1.00  size 0.75`

**Evidence:** `1661.62` (shape 1544.81, concept 5.82, call 110.99)

**Trophic:** `0.73`

**Shared structure:**

- `14.72` — `seq[ assign=(call:append) ; assign=(call:append) ]`
- `14.35` — `if(call:IsNotFound)`
- `7.58` — `seq[ assign:=(call:LevelLoggerToWriter) ; assign:=(sel) ]`

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
| **A** | `internal/js/esbuild/batch.go:1021` | `esbuild.*scriptGroup.Runner` | `(string) (js.OptionsSetter)` | validation, concurrency |
| **B** | `internal/js/esbuild/batch.go:1050` | `esbuild.*scriptGroup.Script` | `(string) (js.OptionsSetter)` | validation, concurrency |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  nesting 1.00  sig 1.00  size 1.00`

**Evidence:** `562.20` (shape 551.22, concept 3.99, call 6.99)

**Trophic:** `1.00`

**Shared structure:**

- `13.33` — `return(call:Get)`
- `7.58` — `seq[ assign:=(call:scriptID) ; if(id) ]`
- `7.58` — `seq[ defer(call:Unlock) ; assign:=(call:scriptID) ]`

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

## Families

306 families, 939 functions in a family, largest 27 members; 623 edges scored here that retrieval never proposed

### Family 1 — 27 members, every pair `>= 0.61` code-shape, evidence `102861`  (195 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `tpl/cast/init.go:25` | `cast.init` | `()` | — |
| `tpl/collections/init.go:25` | `collections.init` | `()` | — |
| `tpl/compare/init.go:26` | `compare.init` | `()` | — |
| `tpl/crypto/init.go:25` | `crypto.init` | `()` | — |
| `tpl/css/css.go:229` | `css.init` | `()` | mapping, caching |
| `tpl/debug/init.go:25` | `debug.init` | `()` | — |
| `tpl/diagrams/init.go:26` | `diagrams.init` | `()` | — |
| `tpl/encoding/init.go:25` | `encoding.init` | `()` | — |
| `tpl/fmt/init.go:25` | `fmt.init` | `()` | — |
| `tpl/hash/hash.go:58` | `hash.init` | `()` | — |

_17 more members not listed._

### Family 2 — 24 members, every pair `>= 0.61` code-shape, evidence `99661`  (139 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `tpl/cast/init.go:25` | `cast.init` | `()` | — |
| `tpl/collections/init.go:25` | `collections.init` | `()` | — |
| `tpl/compare/init.go:26` | `compare.init` | `()` | — |
| `tpl/crypto/init.go:25` | `crypto.init` | `()` | — |
| `tpl/debug/init.go:25` | `debug.init` | `()` | — |
| `tpl/encoding/init.go:25` | `encoding.init` | `()` | — |
| `tpl/fmt/init.go:25` | `fmt.init` | `()` | — |
| `tpl/hash/hash.go:58` | `hash.init` | `()` | — |
| `tpl/images/init.go:25` | `images.init` | `()` | — |
| `tpl/inflect/init.go:25` | `inflect.init` | `()` | — |

_14 more members not listed._

### Family 3 — 27 members, every pair `>= 0.61` code-shape, evidence `98543`  (197 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `tpl/cast/init.go:25` | `cast.init` | `()` | — |
| `tpl/collections/init.go:25` | `collections.init` | `()` | — |
| `tpl/compare/init.go:26` | `compare.init` | `()` | — |
| `tpl/crypto/init.go:25` | `crypto.init` | `()` | — |
| `tpl/debug/init.go:25` | `debug.init` | `()` | — |
| `tpl/diagrams/init.go:26` | `diagrams.init` | `()` | — |
| `tpl/encoding/init.go:25` | `encoding.init` | `()` | — |
| `tpl/fmt/init.go:25` | `fmt.init` | `()` | — |
| `tpl/hash/hash.go:58` | `hash.init` | `()` | — |
| `tpl/hugo/init.go:26` | `hugo.init` | `()` | — |

_17 more members not listed._

### Family 4 — 27 members, every pair `>= 0.61` code-shape, evidence `92704`  (202 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `tpl/cast/init.go:25` | `cast.init` | `()` | — |
| `tpl/collections/init.go:25` | `collections.init` | `()` | — |
| `tpl/compare/init.go:26` | `compare.init` | `()` | — |
| `tpl/crypto/init.go:25` | `crypto.init` | `()` | — |
| `tpl/debug/init.go:25` | `debug.init` | `()` | — |
| `tpl/diagrams/init.go:26` | `diagrams.init` | `()` | — |
| `tpl/encoding/init.go:25` | `encoding.init` | `()` | — |
| `tpl/fmt/init.go:25` | `fmt.init` | `()` | — |
| `tpl/hash/hash.go:58` | `hash.init` | `()` | — |
| `tpl/hugo/init.go:26` | `hugo.init` | `()` | — |

_17 more members not listed._

### Family 5 — 27 members, every pair `>= 0.61` code-shape, evidence `88027`  (204 edges scored here)

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `tpl/cast/init.go:25` | `cast.init` | `()` | — |
| `tpl/collections/init.go:25` | `collections.init` | `()` | — |
| `tpl/compare/init.go:26` | `compare.init` | `()` | — |
| `tpl/crypto/init.go:25` | `crypto.init` | `()` | — |
| `tpl/debug/init.go:25` | `debug.init` | `()` | — |
| `tpl/diagrams/init.go:26` | `diagrams.init` | `()` | — |
| `tpl/encoding/init.go:25` | `encoding.init` | `()` | — |
| `tpl/fmt/init.go:25` | `fmt.init` | `()` | — |
| `tpl/hash/hash.go:58` | `hash.init` | `()` | — |
| `tpl/hugo/init.go:26` | `hugo.init` | `()` | — |

_17 more members not listed._

_301 more families not listed._


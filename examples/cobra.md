# cobra

CLI framework; one dominant type with a long method set, plus shell-completion generators

**What this rung shows:** the receiver and role signals, and per-shell generator siblings

| | |
|---|---|
| Corpus | [cobra](https://github.com/spf13/cobra) |
| Pinned at | `v1.10.2` (`88b30ab89da2d0d0abb153818746c5a2d30eccec`) |
| Project since | 2015 |
| doppel | `706150c` |
| Command | `doppel analyze . --tests exclude --top 10` |

Run from the corpus root, so every path below is corpus-relative.
Regenerate with `task examples`.

## Run diagnostics

The corpus-level models doppel builds before ranking anything, as printed to stderr:

```
Scanning . ...
Generating concept documents...
Culture: 1 concepts modeled, 7 associations, 0 unusual realizations
Habitats: 2 modeled, 0 misfits; most uniform doc (norm 0.91), most diverse cobra (norm 0.91)
Conventions: strongest validation (0.44), loosest validation (0.44)
Ecosystems: 108 profiled (108 dominance, 0 coalition, 0 conflict, 0 weak)
Found 269 functions. Retrieving candidates...
Retrieval: shape 117, concept 45, call 712 -> 810 unique pairs
  concept-only 5.1%  call-only 80.0%  suppressed-shape functions: 0  large identity buckets: 0  surviving patterns: 1257
Running structural comparison on 810 pairs...
Families: 18 over 43 components, 63 functions in a family, 3 edges completed
```

# Code Similarity Report

**Functions analyzed:** 269 | **Threshold:** 0.60 | **Pairs found:** 10

---

## Match #1 — Code-shape: `0.8207`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `doc/md_docs.go:57` | `doc.GenMarkdownCustom` | ` ` | — |
| **B** | `doc/rest_docs.go:62` | `doc.GenReSTCustom` | ` ` | — |

**Code similarity:** `ast 0.80  flow 1.00  sig 0.60  size 0.86`

**Evidence:** `929.98` (shape 872.01, concept 0.00, call 57.97)

**Trophic:** `0.84`

**Shared structure:**

- `26.32` — `do(call:WriteString)`
- `14.51` — `do(call:Fprintf)`
- `9.09` — `seq[ assign=(call:ReplaceAll) ; do(call:Fprintf) ]`

**Structural overlap:** `0.58` (merge-worthy)

- share 25 callees: [Format, buf.WriteString, buf.WriteTo, byName, child.IsAdditionalHelpTopicCommand, child.IsAvailableCommand, child.Name, cmd.CommandPath, cmd.Commands, cmd.HasParent, cmd.InitDefaultHelpCmd, cmd.InitDefaultHelpFlag, cmd.Parent, cmd.Runnable, cmd.UseLine, cmd.VisitParents, fmt.Fprintf, hasSeeAlso, len, linkHandler, new, parent.CommandPath, sort.Sort, strings.ReplaceAll, time.Now]
- overlapping call-graph neighborhoods (0.95): 87 shared
- both are passthrough functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [doc]
- call into same packages: [cobra, doc]

---

## Match #2 — Code-shape: `0.9806`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `doc/md_docs.go:32` | `doc.printOptions` | ` ` | — |
| **B** | `doc/rest_docs.go:30` | `doc.printOptionsReST` | ` ` | — |

**Code similarity:** `ast 0.97  flow 1.00  sig 1.00  size 0.86`

**Evidence:** `291.88` (shape 278.00, concept 0.00, call 13.88)

**Trophic:** `0.93`

**Shared structure:**

- `13.16` — `do(call:WriteString)`
- `9.09` — `seq[ do(call:PrintDefaults) ; do(call:WriteString) ]`
- `9.09` — `seq[ do(call:SetOutput) ; if(call:HasAvailableFlags) ]`

**Structural overlap:** `0.60` (merge-worthy)

- share 9 callees: [buf.WriteString, cmd.InheritedFlags, cmd.NonInheritedFlags, flags.HasAvailableFlags, flags.PrintDefaults, flags.SetOutput, parentFlags.HasAvailableFlags, parentFlags.PrintDefaults, parentFlags.SetOutput]
- overlapping call-graph neighborhoods (0.90): 36 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [doc]
- call into same packages: [cobra]

---

## Match #3 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `flag_groups.go:33` | `cobra.*Command.MarkFlagsRequiredTogether` | `—` | validation |
| **B** | `flag_groups.go:49` | `cobra.*Command.MarkFlagsOneRequired` | `—` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `192.74` (shape 181.34, concept 0.53, call 10.87)

**Trophic:** `1.00`

**Shared structure:**

- `6.89` — `do(call:panic)`
- `4.14` — `range{ call:Lookup call:Flags call:panic call:Sprintf call:SetAnnotation call:append call:Join }`
- `4.14` — `seq[ do(call:mergePersistentFlags) ; range ]`

**Structural overlap:** `0.76` (merge-worthy)

- share 8 callees: [Lookup, SetAnnotation, append, c.Flags, c.mergePersistentFlags, fmt.Sprintf, panic, strings.Join]
- overlapping call-graph neighborhoods (1.00): 34 shared
- share patterns: [validation]
- both are orchestrator functions
- same package
- same visibility
- same receiver type: Command
- call into same packages: [cobra]

---

## Match #4 — Code-shape: `0.8777`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `doc/rest_docs.go:145` | `doc.GenReSTTreeCustom` | ` ` | — |
| **B** | `doc/yaml_docs.go:60` | `doc.GenYamlTreeCustom` | ` ` | — |

**Code similarity:** `ast 0.85  flow 1.00  sig 0.80  size 1.00`

**Evidence:** `307.14` (shape 280.36, concept 0.00, call 26.78)

**Trophic:** `0.93`

**Shared structure:**

- `7.34` — `if(bin:!=(id,nil))`
- `4.14` — `seq[ assign:=(bin) ; assign:=(call:Join) ]`
- `4.14` — `seq[ defer(call:Close) ; if(bin:!=(id,nil)) ]`

**Structural overlap:** `0.56` (merge-worthy)

- share 10 callees: [c.IsAdditionalHelpTopicCommand, c.IsAvailableCommand, cmd.CommandPath, cmd.Commands, f.Close, filePrepender, filepath.Join, io.WriteString, os.Create, strings.ReplaceAll]
- overlapping call-graph neighborhoods (0.83): 35 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [doc]
- call into same packages: [cobra, doc]

---

## Match #5 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `flag_groups.go:33` | `cobra.*Command.MarkFlagsRequiredTogether` | `—` | validation |
| **B** | `flag_groups.go:65` | `cobra.*Command.MarkFlagsMutuallyExclusive` | `—` | — |

**Profile A:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `192.21` (shape 181.34, concept 0.00, call 10.87)

**Trophic:** `1.00`

**Shared structure:**

- `6.89` — `do(call:panic)`
- `4.14` — `range{ call:Lookup call:Flags call:panic call:Sprintf call:SetAnnotation call:append call:Join }`
- `4.14` — `seq[ do(call:mergePersistentFlags) ; range ]`

**Structural overlap:** `0.58` (merge-worthy)

- share 8 callees: [Lookup, SetAnnotation, append, c.Flags, c.mergePersistentFlags, fmt.Sprintf, panic, strings.Join]
- overlapping call-graph neighborhoods (1.00): 34 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: Command
- call into same packages: [cobra]

---

## Match #6 — Code-shape: `1.0000`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `flag_groups.go:49` | `cobra.*Command.MarkFlagsOneRequired` | `—` | validation |
| **B** | `flag_groups.go:65` | `cobra.*Command.MarkFlagsMutuallyExclusive` | `—` | — |

**Profile A:** `validation` 1.00 (dominance)

**Code similarity:** `ast 1.00  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `192.21` (shape 181.34, concept 0.00, call 10.87)

**Trophic:** `1.00`

**Shared structure:**

- `6.89` — `do(call:panic)`
- `4.14` — `range{ call:Lookup call:Flags call:panic call:Sprintf call:SetAnnotation call:append call:Join }`
- `4.14` — `seq[ do(call:mergePersistentFlags) ; range ]`

**Structural overlap:** `0.58` (merge-worthy)

- share 8 callees: [Lookup, SetAnnotation, append, c.Flags, c.mergePersistentFlags, fmt.Sprintf, panic, strings.Join]
- overlapping call-graph neighborhoods (1.00): 34 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: Command
- call into same packages: [cobra]

---

## Match #7 — Code-shape: `0.8425`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `doc/md_docs.go:133` | `doc.GenMarkdownTreeCustom` | ` ` | — |
| **B** | `doc/rest_docs.go:145` | `doc.GenReSTTreeCustom` | ` ` | — |

**Code similarity:** `ast 0.79  flow 1.00  sig 0.80  size 1.00`

**Evidence:** `292.76` (shape 265.99, concept 0.00, call 26.78)

**Trophic:** `0.89`

**Shared structure:**

- `7.34` — `if(bin:!=(id,nil))`
- `4.14` — `seq[ assign:=(bin) ; assign:=(call:Join) ]`
- `4.14` — `seq[ defer(call:Close) ; if(bin:!=(id,nil)) ]`

**Structural overlap:** `0.56` (merge-worthy)

- share 10 callees: [c.IsAdditionalHelpTopicCommand, c.IsAvailableCommand, cmd.CommandPath, cmd.Commands, f.Close, filePrepender, filepath.Join, io.WriteString, os.Create, strings.ReplaceAll]
- overlapping call-graph neighborhoods (0.90): 36 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [doc]
- call into same packages: [cobra, doc]

---

## Match #8 — Code-shape: `0.8725`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `doc/md_docs.go:133` | `doc.GenMarkdownTreeCustom` | ` ` | — |
| **B** | `doc/yaml_docs.go:60` | `doc.GenYamlTreeCustom` | ` ` | — |

**Code similarity:** `ast 0.79  flow 1.00  sig 1.00  size 1.00`

**Evidence:** `292.76` (shape 265.99, concept 0.00, call 26.78)

**Trophic:** `0.87`

**Shared structure:**

- `7.34` — `if(bin:!=(id,nil))`
- `4.14` — `seq[ assign:=(bin) ; assign:=(call:Join) ]`
- `4.14` — `seq[ defer(call:Close) ; if(bin:!=(id,nil)) ]`

**Structural overlap:** `0.56` (merge-worthy)

- share 10 callees: [c.IsAdditionalHelpTopicCommand, c.IsAvailableCommand, cmd.CommandPath, cmd.Commands, f.Close, filePrepender, filepath.Join, io.WriteString, os.Create, strings.ReplaceAll]
- overlapping call-graph neighborhoods (0.83): 35 shared
- both are orchestrator functions
- same package
- same visibility
- same receiver type: plain functions
- called from same packages: [doc]
- call into same packages: [cobra, doc]

---

## Match #9 — Code-shape: `0.8473`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `flag_groups.go:144` | `cobra.validateRequiredFlagGroups` | ` ` | validation |
| **B** | `flag_groups.go:188` | `cobra.validateExclusiveFlagGroups` | ` ` | validation |

**Profile A:** `validation` 1.00 (dominance)

**Profile B:** `validation` 1.00 (dominance)

**Code similarity:** `ast 0.75  flow 1.00  sig 1.00  size 0.97`

**Evidence:** `160.92` (shape 149.36, concept 0.53, call 11.03)

**Trophic:** `0.83`

**Shared structure:**

- `4.54` — `seq[ if(bin:\|\|(bin,bin)) ; do(call:Strings) ]`
- `4.54` — `seq[ range ; if(bin:\|\|(bin,bin)) ]`
- `4.14` — `range{ call:append call:len call:Strings call:Errorf }`

**Structural overlap:** `0.95` (merge-worthy)

- share 5 callees: [append, fmt.Errorf, len, sort.Strings, sortedKeys]
- share 1 callers: [cobra.*Command.ValidateFlagGroups]
- overlapping call-graph neighborhoods (1.00): 6 shared
- share patterns: [validation]
- both are leaf functions
- same package
- callers do related work (1.00): [validation]
- same visibility
- same receiver type: plain functions
- called from same packages: [cobra]
- call into same packages: [cobra]

---

## Match #10 — Code-shape: `0.6271`

| | Location | Function | Signature | Patterns |
|---|---|---|---|---|
| **A** | `command.go:674` | `cobra.stripFlags` | ` ` | — |
| **B** | `command.go:715` | `cobra.*Command.argsMinusFirstX` | ` ` | — |

**Code similarity:** `ast 0.51  flow 0.98  sig 0.50  size 0.94`

**Evidence:** `348.86` (shape 327.56, concept 0.00, call 21.31)

**Trophic:** `0.75`

**Shared structure:**

- `4.54` — `seq[ if(bin:==(call:len,lit:INT)) ; do(call:mergePersistentFlags) ]`
- `3.44` — `assign:=(call:Flags)`
- `2.75` — `do(call:mergePersistentFlags)`

**Structural overlap:** `0.65` (merge-worthy)

- share 8 callees: [append, c.Flags, c.mergePersistentFlags, hasNoOptDefVal, len, shortHasNoOptDefVal, strings.Contains, strings.HasPrefix]
- share 1 callers: [cobra.*Command.Find]
- overlapping call-graph neighborhoods (1.00): 42 shared
- both are orchestrator functions
- same package
- same visibility
- called from same packages: [cobra]
- call into same packages: [cobra]

---

## Families

18 families, 63 functions in a family, largest 9 members; 3 edges scored here that retrieval never proposed

### Family 1 — 9 members, every pair `>= 0.63` code-shape

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `command.go:412` | `cobra.*Command.getOut` | ` ` | — |
| `command.go:422` | `cobra.*Command.getErr` | ` ` | — |
| `command.go:432` | `cobra.*Command.getIn` | ` ` | — |
| `command.go:464` | `cobra.*Command.getUsageTemplateFunc` | ` ` | — |
| `command.go:505` | `cobra.*Command.getHelpTemplateFunc` | ` ` | — |
| `command.go:592` | `cobra.*Command.UsageTemplate` | ` ` | — |
| `command.go:605` | `cobra.*Command.HelpTemplate` | ` ` | — |
| `command.go:618` | `cobra.*Command.VersionTemplate` | ` ` | — |
| `command.go:631` | `cobra.*Command.getVersionTemplateFunc` | ` ` | — |

### Family 2 — 5 members, every pair `>= 0.81` code-shape

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `bash_completions.go:701` | `cobra.*Command.GenBashCompletionFile` | ` ` | — |
| `bash_completionsV2.go:470` | `cobra.*Command.GenBashCompletionFileV2` | ` ` | — |
| `fish_completions.go:284` | `cobra.*Command.GenFishCompletionFile` | ` ` | — |
| `powershell_completions.go:320` | `cobra.*Command.genPowerShellCompletionFile` | ` ` | — |
| `zsh_completions.go:70` | `cobra.*Command.genZshCompletionFile` | ` ` | — |

### Family 3 — 4 members, every pair `>= 0.86` code-shape

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `bash_completionsV2.go:24` | `cobra.*Command.genBashCompletion` | ` ` | — |
| `fish_completions.go:276` | `cobra.*Command.GenFishCompletion` | ` ` | — |
| `powershell_completions.go:313` | `cobra.*Command.genPowerShellCompletion` | ` ` | — |
| `zsh_completions.go:80` | `cobra.*Command.genZshCompletion` | ` ` | — |

### Family 4 — 4 members, every pair `>= 0.75` code-shape

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `args.go:74` | `cobra.MinimumNArgs` | ` ` | — |
| `args.go:84` | `cobra.MaximumNArgs` | ` ` | — |
| `args.go:94` | `cobra.ExactArgs` | ` ` | — |
| `args.go:104` | `cobra.RangeArgs` | ` ` | — |

### Family 5 — 4 members, every pair `>= 0.68` code-shape

| Location | Function | Signature | Patterns |
|---|---|---|---|
| `command.go:592` | `cobra.*Command.UsageTemplate` | ` ` | — |
| `command.go:605` | `cobra.*Command.HelpTemplate` | ` ` | — |
| `command.go:618` | `cobra.*Command.VersionTemplate` | ` ` | — |
| `command.go:643` | `cobra.*Command.ErrPrefix` | ` ` | — |

_13 more families not listed._


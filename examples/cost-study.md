# Do the pairs doppel flags cost maintainers anything?

A measurement, not a scoring change: nothing in the doppel module was modified
to produce it. The git walk lives in `scripts/` (`scripts/cost-study.sh`,
`scripts/historylabel study` / `summarize`), because the module must not know
git exists.

## Pre-registration

Written and committed before any study number was looked at. Nothing below this
section may change the design, the metrics or the rule.

### Design

- **Time split.** For each corpus, T is the latest first-parent commit at or
  before the pinned tag's commit date minus two years. If fewer than 100
  non-merge commits touching `*.go` fall in T..pin, T steps back one year at a
  time, at most to four years. doppel analyses the tree at T only; every
  outcome is read from the commits in T..pin. doppel never sees the window.
- **Operating point.** `doppel analyze . --languages go --top 0 --max-per-func 0
  --families 0` at T, at the shipped defaults (calibration on at rate 0.01). The
  calibrated threshold and struct-min are recorded per corpus from the
  snapshot's `params`. One run per corpus; the text report gives rank and pair
  kind, the `--format json` snapshot of the same run gives the units.
- **Treatment groups.** `top50`: report ranks 1-50 (`analyzer.SortForReport`
  order). `tail`: ranks 51-500.
- **Control group.** For every treated pair, 3 pairs of functions drawn from the
  same T snapshot, matched on
  - locality: same file / same directory (package), other file / same top-level
    directory, other package / different top-level directory; and
  - size: the unordered pair of log2 buckets of each side's body line count
    (1, 2-3, 4-7, 8-15, 16-31, 32-63, 64+), measured on the T tree.

  A control may not be any pair doppel reported at T (any rank), nor a pair of
  same-named functions in one file. Sampling is deterministic: one LCG per
  treated pair, seeded by FNV-1a over corpus, T and the pair's keys.
- **Outcomes** (all within T..pin, over non-merge commits, test files excluded,
  with `historylabel`'s existing exclusions for sweeps, campaigns and
  mechanical commits):
  - `edits` per side: non-noisy modifications of that function's body.
  - `cochanges`: commits that applied the same change to both sides
    (historylabel's co-change evidence).
  - `lagged`, `unpropagated`, `extracted`, `consolidated`: historylabel's
    evidence kinds, restricted to the window.
  - `gone`: a side absent at the pin (deleted, renamed or moved file name) —
    counted, never dropped.

### Primary metrics

- **M1, normalised co-change**: per pair `cochanges / min(editsA, editsB)`
  (0 when either side was never edited), averaged over the group.
- **M2, maintenance-miss rate**: fraction of pairs with at least one `lagged`
  or `unpropagated` event.

For each, the ratio is the treated group's mean over the mean of its matched
controls, with a 95% percentile bootstrap CI (2000 resamples, fixed seed;
a treated pair and its controls are resampled together).

### Decision rule

A corpus **shows cost** when, for the `top50` group, M1 or M2 has a ratio
>= 2.0 **and** a CI lower bound > 1.0. A ratio with a zero control mean and a
non-zero treated mean counts as >= 2.0; its CI is read as computed.

**Evidence of cost** = at least 4 of the corpora run show cost. Anything else is
reported as **no evidence of cost**, without reinterpretation. The tail group,
the per-kind split, the pooled figures and every other rate are descriptive
only and cannot change the verdict.

Planned corpora, in order: cobra, gin (pipeline debugging, still counted),
prometheus, hugo, then moby if time allows.

## Results

Everything below was produced after the pre-registration above was committed
(`6270396`). Reproduce with `task cost-study CORPUS=<name>` per corpus, then
`task cost-study-summary`. The generated tables are in
[cost-study/summary.md](cost-study/summary.md). The raw per-pair rows are in
`cost-study/<corpus>.cost.json`: every treated pair, every control, and every
piece of evidence with its commit. The hand check below read from
[cost-study/samples.md](cost-study/samples.md). Two runs of the same corpus
are byte-identical.

One pipeline bug was found and fixed while debugging on cobra and gin. That
was before any summary table for prometheus, hugo or moby had been read. The
bug: doppel names a generic method `*Stack[T].Pop`, but historylabel's parser
rendered it `*Stack.Pop`, so every generic method was silently untrackable
(111 hugo units). `histName` now maps one name onto the other, and the fix
also applies to `history-labels`. All five corpora were re-run after it, with
0 unresolved units and 0 treated pairs short of controls.

### Operating points

| corpus | T | T date | pin date | window commits replayed | functions at T | threshold | struct-min | pairs reported |
| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: |
| cobra | `9e1d6f1c2` | 2021-11-16 | 2025-12-03 | 113 | 259 | 0.46 | 0.52 | 229 |
| gin | `ecdbbbe94` | 2024-02-19 | 2026-02-28 | 132 | 417 | 0.42 | 0.50 | 490 |
| prometheus | `e86e4ed87` | 2024-08-16 | 2026-08-17 | 1402 | 4086 | 0.33 | 0.35 | 8910 |
| hugo | `2192cf7ec` | 2024-08-12 | 2026-08-12 | 779 | 4631 | 0.34 | 0.31 | 10745 |
| moby | `02011af7b` | 2023-11-04 | 2025-11-05 | 1885 | 7299 | 0.36 | 0.31 | 16595 |

The thresholds are the calibrated values (rate 0.01) doppel derived at T.
cobra had 39 `*.go` commits in two years and 65 in three, so the rule stepped
T back to four years. Every other corpus used two. Each corpus has 50 top50
pairs and 3 controls per treated pair. cobra's tail is 179 pairs (229 reported
in all); the others' tails are 440-450.

### Decision

| corpus | M1 ratio [95% CI] | M2 ratio [95% CI] | shows cost | top50 pairs with an M1/M2 event | distinct commits behind them |
| --- | --- | --- | --- | ---: | ---: |
| cobra | 15.00 [0.00, ∞] | — (no events either side) | no | 1 | 1 |
| gin | ∞ [∞, ∞] | 0.00 [0.00, 0.00] | **yes** | 6 | 1 |
| prometheus | ∞ [∞, ∞] | 6.00 [0.00, ∞] | **yes** | 17 | 28 |
| hugo | ∞ [∞, ∞] | ∞ [∞, ∞] | **yes** | 7 | 6 |
| moby | ∞ [∞, ∞] | ∞ [∞, ∞] | **yes** | 4 | 3 |

**4 of 5 corpora show cost. Under the pre-registered rule this is evidence of
cost.** Every "yes" comes from M1. M2 never clears the bar on its own.

The verdict stands as the rule computes it. What it rests on is narrower than
the word "evidence" suggests. These are facts about the result, not
reinterpretations of it:

- **The controls almost never produce an event.** Over all five corpora, the
  750 top50 controls produced one co-change, and four corpora have a control
  M1 of exactly zero. The ratio is then ∞. The percentile bootstrap returns
  [∞, ∞] because no resample can contain a control event that does not exist.
  So the rule's "CI lower bound > 1" was met by the absence of control events,
  not by a precise estimate: the bootstrap cannot express uncertainty about a
  zero rate. A rule-of-three bound on the control side (≤ 3/150 at 95%) would
  not support a 2× claim on gin or moby by itself.
- **The events are clustered.** gin's "yes" is one commit, counted as six
  pairs: 93ff771e6, a gosec fix that rewrote all four `Engine.Run*` methods.
  moby's top 50 rests on three commits and hugo's on six. The bootstrap
  resamples pairs, so it treats those six gin pairs as six independent
  observations. prometheus (17 pairs, 28 commits) is the only corpus where the
  top-50 result does not hinge on one or two maintenance events.
- **The instrument favours alike bodies by construction.** "The same change
  landed in both" is far likelier for two similar bodies than for two random
  neighbours, whatever detector found them. The controls are matched on
  locality and size, not on similarity. So this study shows that doppel's
  pairs cost more than matched random pairs. It does not show that doppel
  finds costly pairs better than a cheaper similarity detector would. That
  needs a different control: pairs that a naive token-Jaccard or a same-name
  rule flags and doppel does not.

### Absolute rates, and decay with rank

Pooled over the five corpora (from [summary.md](cost-study/summary.md)):

| | top50 | its controls | tail (51-500) | its controls |
| --- | ---: | ---: | ---: | ---: |
| M1 normalised co-change | 0.105 | 0.0003 | 0.027 | 0.000 |
| M2 lagged or unpropagated | 2.0% (5 of 250) | 0.4% (3 of 750) | 0.8% | 0.1% |
| any co-change | 12.4% (31 of 250) | 0.1% | 3.2% | 0.0% |
| synced (≥ 2 co-changes) | 3.6% | 0% | 0.8% | 0% |
| extracted or consolidated | 2.4% | 0% | 0.3% | 0% |
| both sides edited at all | 22.8% | 18.7% | 8.8% | 6.9% |
| mean edits per side | 0.55 | 1.17 | 0.31 | 0.40 |
| a side gone by the pin | 22.0% | 13.3% | 8.3% | 13.7% |

- **The cost is real but rare.** Seven in eight top-50 pairs show no shared
  maintenance at all over two years (four years on cobra). Where a pair does
  carry a cost, it is usually one edit made twice, not repeated drift.
- **Signal decays with rank.** The tail's co-change rate is about a quarter of
  the top 50's, and its M2 rate about two fifths. Both stay above their
  controls: tail M2 is 15× [4.7, ∞], the one tail figure whose CI is not
  degenerate.
- **Normalising by churn makes the result sharper, not weaker.** Treated
  functions are edited *less* than their controls (0.55 against 1.17 edits per
  side), so the co-changes do not come from busier functions.
- **"Gone" is not consolidation.** At T, 28 of cobra's top 50 pairs were one
  family: the license-template `init*` functions of `cobra/cobra`. All of them
  were removed when the cobra-cli generator moved to its own repository. That
  is why cobra's top 50 shows `gone` at 0.56 against 0.15, and why its window
  had almost nothing to measure.

### By kind and locality (descriptive, ranks 1-500 pooled)

| kind | pairs | M1 treated / control | M2 treated / control |
| --- | ---: | --- | --- |
| none | 1322 | 0.048 / 0.000 | 0.5% / 0.1% |
| interface implementations | 410 | 0.027 / 0.000 | **2.9% / 0.2%** (18× [5.4, ∞]) |
| thin wrappers | 324 | 0.000 / 0.000 | 0.3% / 0% |
| different calls | 117 | 0.019 / 0.000 | 0% / 0% |
| mirror operations | 24 | 0.056 / 0.000 | 0% / 0% |
| diverged copy | 10 | 0 / 0 | 0 / 0 |
| subsystem copies | 8 | 0 / 0 | 0 / 0 |

- **Interface implementations carry the drift.** They are the one kind whose
  lagged-and-unpropagated rate stands well clear of its controls. Nearly all
  of it is moby's network drivers: `ipvlan` and `macvlan` `Join` and
  `CreateNetwork` were given the same feature one commit apart, three times in
  two years. The kind is meant to mark a pair as unactionable, yet history
  shows these are the pairs whose maintenance lags.
- **Thin wrappers cost nothing measurable.** None of the 324 pairs co-changed.
  That fits the kind's claim that the consolidation has already happened.
- **Mirror operations co-change** (0.056) but never lag. Inverse operations
  are kept in step, which is coupling, not duplication.
- Most co-change sits in pairs with no kind at all.
- By locality, same-package pairs carry the most (M1 0.053, M2 2.5%).
  Same-file controls never co-change, which is the confound the locality
  match exists to remove.

`diverged copy` (10 pairs) and `subsystem copies` (8) are too few to read.

### Hand check

[samples.md](cost-study/samples.md) lists, per corpus, the best-ranked treated
pairs carrying any cost event, up to 10. Each was read against its commit
diffs. Only cobra and gin had fewer than 10 to check.

| corpus | checked | real duplicated change | real but mechanical | not a cost |
| --- | ---: | ---: | ---: | ---: |
| cobra | 2 | 2 | 0 | 0 |
| gin | 6 (one commit) | 0 | 6 | 0 |
| prometheus | 10 | 10 | 0 | 0 |
| hugo | 10 | 6 | 2 | 2 |
| moby | 10 | 6 | 3 | 1 |

- **cobra.** `InitDefaultHelpFlag` and `InitDefaultVersionFlag` were given the
  same completion annotation (212ea4078). The bash nounset fix (4ba5566f5)
  landed only in `genBashComp` (V2). V1's `writePreamble` still declares
  `local fullFilter` unset at the pin, so the bug is still live there.
- **gin.** All six samples are one gosec-driven commit (93ff771e6). It
  replaced `http.ListenAndServe*`/`http.Serve` with an explicit `http.Server`
  in `Run`, `RunTLS`, `RunListener` and `RunUnix`. That is a real edit made
  four times by hand, but it is lint-driven, and it is one event, not a
  pattern.
- **prometheus.** Every sample is a real duplicated change, and several are
  bug fixes made twice:
  - `NewEndpoints`/`NewEndpointSlice` got the same `DeletedFinalStateUnknown`
    handling (b1c356bee) and the same linked-controller support.
  - `AppendHistograms`/`AppendFloatHistograms` got the NHCB drop (43c1535bd)
    and the ST plumbing.
  - `sendSamplesWithBackoff`/`sendV2SamplesWithBackoff` got a metrics fix
    (897ba10d1). The maintainers then partly DRY'd them ("create common
    struct and function to DRY", e1cb29bf8).
  - The two chunk appenders lagged each other twice ("make
    HistogramAppender.appendHistogram reusable", then the same for Float).
  - `funcHistogramStdDev`/`StdVar` were merged into `histogramVariance`
    ("[REFACTOR] PromQL: DRY").
  - The two parsers' `Metric` got the same label normalisation (8bcb4d865).
  - `buildTimeSeries`/V2 got the same refactor, `addH`/`addFH` the same ST
    field, `NewIngress`/`NewService` the same namespace metadata, and the two
    moby discoveries the same timeout fix.
- **hugo.** Real:
  - `renderLink`/`renderImage`/`renderHeading` (three pairs) were each
    rewritten twice alike: for table render hooks (f738669a4) and for the
    goldmark 1.7.8 upgrade.
  - `renderLinkDefault`/`renderImageDefault` got the same XSS escaping fix
    (479fe6c65). A security fix made twice is the plainest cost in the
    sample.
  - babel/postcss `Transform` got the same config-variant support, and the
    codeblock/passthrough renderers got the render-hook change.

  Mechanical: `gofmt`/`goimports`/`rewrite` in a build script dropped
  `safeexec` alike. Not a cost:
  - `math.init` ↔ `strings.init` read as "lagged" because two unrelated
    template functions were registered months apart.
  - The tailwind "unpropagated fix" (a03a245f0) does not apply to postcss.
    postcss's `Transform` already guards the same call with
    `options.InlineImports.InlineImports`.
- **moby.** Real:
  - `ipvlan`/`macvlan` `Join` and `CreateNetwork` were given the same feature
    in paired commits, in both directions.
  - The three drivers' `initStore` changed alike when the datastore was
    shared (d21d0884a): one commit, three pairs.
  - The Linux and Windows mount parsers both got subpath validation.

  Mechanical: three `*Prune` pairs from one zero-value-return cleanup
  (7faaa3afa). Coupling rather than duplication:
  `addServiceInfoToCluster`/`deleteServiceInfoFromCluster`, an inverse pair
  edited together.

Overall, 24 of 38 checked events are real duplicated maintenance, 11 are real
but mechanical, and 3 are not a cost: two instrument false positives and one
case of inverse-pair coupling. The instrument is precise enough that the
treated rates above are not mostly noise.

### Verdict

**Evidence of cost, by the pre-registered rule: 4 of 5 corpora.** Stated
without reinterpretation, and with its reach:

- doppel's top-ranked pairs get the same maintenance edit twice far more
  often than matched pairs of neighbouring functions, which almost never do.
- Hand-checked, the edits are mostly real, and several are bug or security
  fixes that had to be made twice. One (cobra's bash nounset fix) was made
  once and is still missing from its twin.
- The effect is small in absolute terms, about one top-50 pair in eight over
  two years, and it decays with rank.
- On gin and moby the result rests on one and three commits. Wherever the
  controls had no events, the CIs are degenerate.
- The study cannot say whether doppel finds these pairs better than a simpler
  similarity measure would. That is the next control group to run.

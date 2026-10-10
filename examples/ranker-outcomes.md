# Which ranking finds the pairs that later cost maintainers something?

A measurement, not a scoring change: nothing in doppel's ranking, scoring or
defaults is touched to produce it. It joins the two studies before it.

- `examples/baselines.md` ranked pairs with doppel and with simple baselines,
  and scored them against history labels taken **at the pin**. On those labels,
  retrieval mass alone ranked the "refactor" pairs better than the full key on
  4 of 5 corpora.
- A diagnosis afterwards found that every history "refactor" label is an
  `extracted` verdict: maintainers moved code out of both functions into a new
  helper **before** the pin. In about 78 of the 96 labels, both sides call that
  helper at the pin. So the labels describe duplication that had already been
  fixed. The shared helper is a call-channel token, and the emptied bodies read
  low on code-shape and trophic. A ranker scored on them is rewarded for
  spotting finished refactors.
- `examples/cost-study.md` avoided that by ranking at a revision T and reading
  outcomes from T..pin. It compared doppel's pairs only with matched random
  pairs. It said the open question was whether a cheaper ranker would find the
  costly pairs as well.

This study asks that question. Every ranker sees only the tree at T, and every
outcome comes from commits after T.

## Pre-registration

Written and committed **before** any ranking at T was scored against any
outcome. Nothing in this section may change after results exist. Results and
the verdict go in sections below it.

### Design

- **Corpora and T.** cobra, gin, prometheus, hugo, moby, with exactly the T and
  pin the cost study used (its committed `examples/cost-study/<corpus>.cost.json`
  `since` and `pin`). No new choice of T.
- **Ranking at T.** `TestRankingsAt` (`internal/bench`) analyses the T tree with
  the bench pipeline at the production defaults. Calibration is on at rate
  0.01, re-retrieving at the calibrated threshold. The population is the
  cost study's: Go files only, test files and generated files excluded.
- **Pool.** Every method ranks the same pool: doppel's retrieval union at the
  calibrated threshold (setting A-union in `examples/baselines.md`). So this
  compares **ranking functions**, not retrieval. A pair that retrieval never
  proposed is in no list.
- **Methods.**

  | method | key |
  | --- | --- |
  | doppel | `analyzer.SortForReport` order, no `--max-per-func` cap |
  | doppel (struct-min filtered) | the same, keeping pairs at or above the calibrated struct-min |
  | token clones | Jaccard over `Fingerprint.Shingles` |
  | code-shape | `SimilarPair.Score` |
  | retrieval mass | `Retrieval.Total` |
  | call mass | `Retrieval.Call`, the channel the diagnosis found carrying the leaked labels |
  | overlap | `Evidence.OverlapScore` |
  | name heuristic | as in `examples/baselines.md` (same package only) |
  | size | `min(Nodes)` of the two sides, the size/churn confound as a ranker of its own |
  | random | seeds 1, 2, 3, pooled |

  Ties break on `(AIdx, BIdx)` for every method. Each method's list is cut at
  rank 500.
- **Outcomes.** `historylabel compare` replays T..pin with the same walk, the
  same `judge` and the same exclusions (sweeps, campaigns, mechanical commits)
  as the cost study. Each distinct pair is judged once, whichever lists hold
  it. Per pair it records `editsA`, `editsB`, `cochanges`, `lagged`,
  `unpropagated`, `extracted`, `consolidated` and the commits behind them.
  `extracted` and `consolidated` are now outcomes: a refactor that happens
  after T is a cost the ranking could have foreseen.

### Metrics

For a method's top K pairs:

- **M1**: mean over pairs of `cochanges / min(editsA, editsB)`, 0 when either
  side was never edited. This is the cost study's metric. Normalising by the
  less-edited side keeps a ranker from winning by picking busy or large
  functions.
- **M2**: fraction of pairs with at least one `lagged` or `unpropagated` event.
- **E**: fraction of pairs with any event (`cochanges`, `lagged`,
  `unpropagated`, `extracted`, `consolidated`).
- **C**: number of distinct commits behind those events. One commit that
  touches six pairs counts once.

### Primary comparison and decision rule

- **Primary metric: M1 at K = 100.**
- For each baseline X on each corpus: the difference `M1(doppel) − M1(X)` at
  K = 100, with a 95% percentile bootstrap CI (2000 resamples, fixed seed). Each
  list's 100 pairs are resampled independently. For random, the list is the
  300 pairs of the three seeds' top 100.
- doppel **beats** X on a corpus when the CI's lower bound is > 0. X **beats**
  doppel on a corpus when the upper bound is < 0. Otherwise the two are **not
  distinguishable** there.
- Over the five corpora: doppel **beats** X when it beats X on at least 3 and X
  beats doppel on none. X **beats** doppel under the same rule the other way
  round. Anything else is reported as **not distinguishable**, without
  reinterpretation.
- Everything else (K = 50, ranks 101-500, M2, E, C, the clustered sensitivity
  below, per-kind splits) is descriptive and cannot change the verdict.

### Clustered sensitivity (descriptive)

The cost study found that one commit can produce several co-changes (gin's one
gosec commit touched six pairs). As a sensitivity check, M1 is also computed with
each co-change commit's credit divided among the pairs of the same top-K list
that it touches.

### Validity checks, stated in advance

- **Agreement with the cost study.** "doppel (struct-min filtered)" at T should
  reproduce the cost study's treated list, which came from the binary at the
  same T. The overlap of the two top-50s is reported. Below 90% the bench and
  binary populations differ, and the results say so.
- **Unresolved units.** Units `historylabel` cannot find at T are reported per
  corpus. A pair with an unresolved side is dropped from every list and counted.

### Expectations

1. If the at-pin result was leakage, **retrieval mass and call mass will not
   beat doppel** at T. If either does beat it on M1@100, the at-pin finding was
   not only leakage, and the trade the rank key makes against refactor-shaped
   pairs costs real outcomes.
2. **size** will look strong on E and weak on M1, because M1 normalises by
   edits.
3. Power is low. The cost study saw any co-change in about one top-50 pair in
   eight, so **not distinguishable** is the likely result for most baselines.
   That would mean these outcomes cannot separate the rankers. It would not
   mean doppel's ranking is worthless.

## Results

Not yet run.

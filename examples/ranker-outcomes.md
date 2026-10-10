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

Everything below was produced after the pre-registration above was committed
(`05459cb`). Reproduce with `task ranker-outcomes CORPUS=<name>` per corpus, then
`task ranker-outcomes-summary`. The generated tables are in
[ranker-outcomes/summary.md](ranker-outcomes/summary.md). The raw per-pair rows
are in `ranker-outcomes/<corpus>.outcomes.json`: every list, every judged pair,
and every piece of evidence with its commit. Two runs of the same corpus are
byte-identical, and `historylabel study` still reproduces the committed cobra
cost-study file byte for byte after the two subcommands were made to share
their judging and unit-resolution code.

### Validity checks

- **Agreement with the cost study: 50 of 50 on every corpus.** The bench
  pipeline at T reproduces the binary's treated top 50 exactly, along with its
  calibrated threshold, struct-min and report size.
- **Unresolved units: 0 on every corpus.** One pair was dropped from three of
  hugo's lists (name heuristic, call mass, one random seed) because both sides
  are same-named functions in one file. Each of those lists still has 100 pairs.

### Verdict, by the pre-registered rule

| baseline | doppel wins | baseline wins | verdict |
| --- | ---: | ---: | --- |
| token clones | 4 | 0 | **doppel beats it** |
| code-shape | 4 | 0 | **doppel beats it** |
| name heuristic | 4 | 0 | **doppel beats it** |
| random (3 seeds) | 3 | 0 | **doppel beats it** |
| retrieval mass | 0 | 0 | not distinguishable |
| call mass | 1 | 0 | not distinguishable |
| overlap | 1 | 0 | not distinguishable |
| size | 2 | 0 | not distinguishable |
| doppel (struct-min filtered) | 0 | 0 | not distinguishable |

No baseline beats doppel on any corpus: no CI upper bound falls below zero
anywhere in the table. cobra separates nothing. Its four-year window holds one
co-change in doppel's top 100, and the cost study already showed that its top
pairs were removed wholesale with the cobra-cli move.

Against the expectations stated in advance:

1. **Retrieval mass and call mass do not beat doppel at T.** The at-pin result in
   `examples/baselines.md` (mass ranks history "refactor" pairs better on 4 of 5
   corpora) does not survive a time-correct outcome. Measured before the fact,
   mass's M1@100 is below doppel's on prometheus (0.187 vs 0.216) and moby
   (0.029 vs 0.040), equal on gin and within one pair on cobra and hugo. That
   fits the leakage diagnosis. But **doppel does not beat mass either**. So on
   these outcomes, the factors the key multiplies onto mass (overlap, shape²,
   trophic²) are not shown to add anything. They are not shown to cost anything
   either.
2. **size is not strong on E.** Its E@100 is below doppel's on gin,
   prometheus and hugo, and ties it on cobra (0.01) and moby (0.06). The size confound found in the at-pin "coupled"
   labels does not carry over to an outcome normalised by edits, and only
   partly to the unnormalised one.
3. **Power is low, as predicted.** Four of the eight baselines are not
   distinguishable. The ones doppel does beat are beaten because they score
   almost exactly zero, not because doppel's margin is large.

### What the numbers show (descriptive, post-hoc)

- **Pure similarity rankers find pairs nobody maintains.** Over doppel's
  retrieval union, code-shape alone and token clones put trivial bodies first.
  Their top 100 has a median smallest side of 3 lines on four corpora. On
  prometheus, hugo and moby, not one of those pairs had both sides edited in two
  years. code-shape's M1@100 is 0.000 on four of five corpora, and token
  clones' on three. That is the case for
  multiplying by mass: identical one-liners are a finding nobody pays for.
- **doppel and mass reach the same rate through different pairs.** Their top
  100s share only 32-55 pairs. Mass prefers larger functions (median smallest
  side 9-45 lines against doppel's 6-27), which get edited more often. Because
  M1 normalises by edits, the two end up level.
- **prometheus carries the signal.** doppel's top 100 there has events on 33 pairs
  from 47 distinct commits, and the clustered M1 (0.206) barely moves from the
  plain one (0.216). gin's is the opposite. Its 0.065 rests on two commits and
  falls to 0.019 when each commit is counted once, which is the gosec commit
  the cost study already flagged.
- **The struct-min filter changes nothing here.** "doppel" and "doppel
  (struct-min filtered)" score the same M1@100 on four corpora. They differ
  only on gin (0.065 against 0.055).
- **Signal decays with depth for every ranker.** For doppel, M1 at ranks 101-500 is
  a quarter of M1@100 on prometheus and hugo, matching the cost study's tail. On
  moby it is about the same as the top 100.

### What this means for the plan that motivated it

- The history-label finding that "mass beats the key" was **substantially an
  artefact of scoring at the pin**. Measured before the fact, the key is never
  behind mass by a margin the bootstrap can see.
- There is **no outcome evidence for changing the rank key** in either direction.
  Removing trophic² or shape² to favour refactor-shaped pairs is not supported:
  mass alone is the extreme of that move, and it does not do better.
- What *is* supported is that the **mass factor is load-bearing** and the pure
  similarity factors are not enough on their own. This is consistent with the
  key's design, and now measured on outcomes rather than on doppel-drawn labels.
- The open questions are the ones low power leaves: whether overlap, shape² and
  trophic² add value over mass. Answering them needs denser outcomes (more
  corpora or longer windows) or hand labels drawn from a pool of every method's
  top K. That is step 5 of the plan.

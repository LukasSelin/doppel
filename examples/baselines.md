# Does doppel's ranking beat simple baselines?

A measurement, not a change: nothing in the doppel module's ranking, scoring or
defaults was touched to produce it. `TestBaselines` (`internal/bench/baselines_test.go`,
guard `DOPPEL_BENCH_BASELINES=1`, `task baselines`) is the harness.

## Pre-registration

Written and committed **before** the first run. Nothing in this section was
edited after results existed; the results and verdict sections below are the
only parts written afterwards.

### Methods

Every method produces one ranked pair list per corpus. Ties are broken by
`(AIdx, BIdx)` ascending for every method alike, so a list is deterministic.

| # | method | key |
| --- | --- | --- |
| 1 | **doppel** | `analyzer.SortForReport` order, `RankKey = Total × Overlap × Score² × Trophic²` (× CallSim on test pairs), **no** `--max-per-func` cap |
| 1b | doppel, struct-min filtered | method 1, keeping only pairs at or above the calibrated `--struct-min` — what `doppel analyze` reports at `--top 0 --max-per-func 0` |
| 2 | random | seeded shuffle; 20 seeds, mean and standard deviation reported |
| 3 | token clones | Jaccard over `Fingerprint.Shingles` (token 3-gram hashes) |
| 4 | code-shape | `SimilarPair.Score` (`fingerprint.Similarity`) alone |
| 5 | retrieval mass | `Retrieval.Total` alone |
| 6 | overlap | `Evidence.OverlapScore` alone |
| 7 | name heuristic | `max(stem Jaccard, 1 − edit distance / max length)` over the bare function/method names, × `min(nodes)/max(nodes)`; **same package only** — cross-package pairs are not ranked |
| 8 | external clone detector (`dupl`/`jscpd`) | optional; not done if it is not installable without friction |

### Operating point and settings

The pipeline runs at the production default: `--calibrate 0.01` derives the
code-shape threshold (and the struct-min for method 1b) from each corpus's own
null; a corpus whose calibration declines keeps the static defaults. Population
is whatever the labels file declares (`exclude` for every label set here).

- **Setting A — re-rank one pool.** Every method ranks the same candidate pool, so
  only ranking quality differs. Two pools are reported:
  - **A-union**: doppel's retrieval union at the calibrated threshold (every pair the comparator scores).
  - **A-report**: A-union filtered at the calibrated struct-min — the pool
    `doppel analyze --top 0 --max-per-func 0` emits, which is the pool the
    history labels were drawn from.
- **Setting B — end to end, all pairs.** On cobra, chi, conc and gin, every
  `parser.SameBuildUnit` pair in the population is a candidate. Methods that can
  score any pair (random, token clones, code-shape, overlap, name heuristic)
  rank all of them; methods that only exist on retrieved pairs (doppel, doppel
  filtered, retrieval mass) rank their pool and leave the rest unranked, so
  retrieval recall is part of what is measured. A label a method leaves
  unranked counts as a miss for hits@K, and for mean rank takes the expected
  rank of a random completion, `(len(list) + 1 + N_pairs) / 2`.

### Metrics

- Mean rank per label class (`merge`, `refactor`, `false_positive`, `coupled`) over labels present.
- **Relevant** = merge + refactor. `merge+refactor mean rank` pools both classes.
- precision@10/20/50 = relevant labels in the top K / K (unlabelled pairs count as not relevant — labels are partial, so absolute values are low; only the comparison means anything).
- The three hard assertions on cobra (merges retrieved, no false positive above the worst merge, no false positive in the top 20).
- Setting B: labelled relevant pairs in the top 50 and top 100.
- History labels contain refactor and coupled only (no merge, no false positive), so on them "merge+refactor" is refactor alone; coupled is reported beside it as a secondary class and does not enter the verdict.

### Evidence

- Committed hand labels: `examples/labels/cobra.labels.json` (18 pairs).
- History-derived labels (`task history-labels`), already generated for gin, prometheus, hugo, moby (and chi, cobra), read from the history cache.
- Private labels if provided by env; reported only as aggregates, never named.

### Decision rule

doppel **beats** a baseline when **both** hold:

1. **cobra hand labels, A-union**: doppel has a strictly higher precision@20, a strictly lower merge+refactor mean rank, and a false-positive mean rank no lower (a lower number would mean false positives ranked earlier) than the baseline.
2. **History labels, A-report, at least 3 of gin, prometheus, hugo, moby**: doppel has a strictly higher precision@20 and a strictly lower refactor mean rank than the baseline.

For random, the baseline's value is the mean over the 20 seeds.

Because labels are sparse, precision@20 will often tie (frequently at 0). A tie
does **not** count as beating. A secondary, weaker reading — precision@20 not
lower instead of strictly higher — is also reported, and is labelled as such
wherever it is used; the primary verdict is the strict one above.

Setting B is a secondary check and does not enter the verdict: the cobra hand
labels were drawn from doppel's own ranked output, and the history labels from
doppel's reported pairs, so on all-pairs rankings both favour doppel's
retrieval by construction. Setting A on the history labels is fair only in the
sense that every method sees the same pool; A-report is the pool those labels
came from, which is why condition 2 uses it.

### Expectation

The hardest baseline is expected to be **code-shape alone** (method 4): it is one
of doppel's own ranking factors, squared in the key, and `TestOverlapRank`
found code-shape to be the factor that separates merges from false positives.
If doppel does not beat code-shape alone under the rule above, then overlap,
trophic and retrieval mass are **not shown to add value** on these labels, and
the write-up will say so in those words. Second hardest is expected to be token
clones on the merge class (cobra's merges are near-verbatim), and the weakest
random and the name heuristic.

## Results

Run on 2026-10-10 against the pinned ladder, at the production operating point.
The run is deterministic: two runs gave byte-identical tables. Calibration:
moby 0.36 / 0.30 (threshold / struct-min), prometheus 0.33 / 0.34, hugo
0.34 / 0.31, gin 0.41 / 0.49, cobra 0.44 / 0.53, chi 0.45 / 0.51. Every history
label and every cobra hand label is in both A pools, so in setting A no method
leaves a label unranked (method 1b and the name heuristic excepted, by
construction).

Labels used: cobra hand (6 merge, 9 refactor, 3 false positive); history labels
generated on 2026-10-03 for moby (33 refactor, 67 coupled), prometheus (43 / 73),
hugo (9 / 12), gin (7 / 7), chi (6 / 2) and cobra (4 coupled). conc's history file
is empty and conc has no hand labels, so conc is in the export only. kubernetes'
history labels predate the refactor/coupled split and kubernetes is not a ladder
rung, so it was left out. No private labels were provided for this run.

### Verdict, by the pre-registered rule

**doppel beats none of the six baselines**, under the strict rule or the weak
one. That includes random. The rule fails for two reasons, and both of them are
weaknesses of the rule as I wrote it. I am stating them, not using them to set
the result aside:

1. **Condition 2 cannot be met by any method.** precision@20 is 0.00 for every
   method on all four larger corpora (the random seed mean is between 0.00 and
   0.01). The history labels are 7–43 relevant pairs in pools of 570–18 545, so
   no ranking puts one in the top 20. A strict "higher P@20" can never hold.
   Under the weak reading (P@20 not lower), doppel wins on refactor mean rank
   against token clones on 4/4 corpora, code-shape 3/4, overlap 3/4, name
   heuristic 2/4, retrieval mass 0/4. Random scores 1/4 only because its seed
   mean P@20 is a few thousandths above doppel's 0.
2. **Condition 1 fails on the false-positive clause against every baseline,
   random included.** doppel puts cobra's three false positives at mean rank
   513.5 of 1 401. Every baseline puts them later (540–1 024). None of them is in
   doppel's top 20 or above its worst merge, and all three hard assertions pass.
   But these three pairs were labelled *because* they look alike, so any ranker
   that does better than random on similarity pulls them up. "No worse
   false-positive mean" therefore works against exactly what makes a ranker
   good. On cobra's other two clauses doppel wins against every baseline: P@20
   0.70 against at most 0.60, merge+refactor mean 11.0 against at least 15.3.

The prediction was that code-shape alone would be the hardest baseline, and the
rule said what to conclude if it lost. **doppel does not beat code-shape alone
under the pre-registered rule, so these labels do not show that overlap,
trophic or retrieval mass add value.** The descriptive numbers below say more
than that, but they are post-hoc and do not replace the verdict.

### What the numbers do show (descriptive, post-hoc)

- **On the only labels that judge mergeability (cobra hand), the full key is
  clearly the best ranker.** A-union: P@10 0.80 and P@20 0.70, against 0.60 /
  0.60 for the best single factor. Merge mean rank 4.8, against 14.7 at best
  (token clones, A-report). The hardest baseline here is **overlap alone**, not
  code-shape. Overlap beats doppel on the refactor class (13.9 vs 15.1), but it
  puts a false positive in the top 20 and above the worst merge. Code-shape
  alone is weak here: P@20 0.20, merge mean 21.7. Two caveats limit how much this
  says. There are 18 labels. And they were drawn from doppel's own ranked output,
  so they favour whatever doppel already puts near the top.
- **On history labels, retrieval mass alone ranks the history-relevant pairs
  better than the full key.** In A-report (the pool the labels came from), mass
  alone has the lower refactor mean on 4 of 5 corpora: moby 3 910 vs 5 058,
  prometheus 2 670 vs 3 163, hugo 2 769 vs 2 912, gin 47 vs 62. It also has the
  lower coupled mean on moby, prometheus and hugo. Pooled normalised refactor
  rank is 0.211 for mass and 0.183 for doppel, so doppel stays ahead on average
  only because of chi (mass 43.5 vs 9.5). This is the cost side of a trade the
  ranking already made on purpose: CLAUDE.md records that `ShapePower` 2 was
  adopted with a known refactor penalty ("merges before refactors"). History
  labels contain no merges, so they measure only the cost. It is still the
  measurement most worth taking seriously. On maintainer behaviour, multiplying
  mass by overlap × shape² × trophic² moves co-maintained pairs **down**.
- **Code-shape alone versus the full key on history (A-report):** doppel is
  ahead on moby (5 058 vs 7 852), prometheus (3 163 vs 4 059) and gin (62 vs 404),
  and behind on hugo (2 912 vs 2 781) and chi (9.5 vs 4.7). Pooled normalised rank
  is 0.183 vs 0.337. As description, the extra factors do help over code-shape on
  most corpora. The rule cannot certify it, because P@20 is zero everywhere.
- **The name heuristic is a stronger baseline than expected on history labels.**
  It ranks the history-relevant pairs it can rank (same package only) ahead of
  doppel on moby, hugo and chi. Maintainers keep similarly named functions in
  one package in step, and history labels reward that. On cobra's hand labels it
  is the second-worst method after random.
- **Setting B (all pairs, retrieval recall included): doppel's retrieval loses
  none of the labelled pairs that matter.** 24 of 28 labelled relevant pairs are
  in its top 50 and 27 in its top 100. That compares with overlap 20 / 24,
  code-shape 9 / 19, token clones 6 / 18, mass 12 / 21 and name heuristic 8 / 11.
  Run over all pairs, code-shape and token clones fill the top with trivial
  bodies (on cobra, 3 and 0 labelled pairs in the top 50). This setting favours
  doppel by construction, because every label here came from doppel's own
  output. It shows that retrieval does not throw labelled pairs away. It does
  not show that the method is better.
- **Random** is beaten on mean rank by every method on every label set, by one to
  two orders of magnitude (cobra A-union: 665.7 ± 117.1 against 11.0).

### Biases to keep in mind

- **The cobra hand labels came from reviewing doppel's ranked output.** Pairs
  that doppel never surfaced cannot be labelled at all. That inflates doppel's
  top-K numbers in both settings, most of all in B.
- **The history labels came from `doppel analyze --top 0 --max-per-func 0`**, so
  every one of them cleared the calibrated struct-min. In A-union this selection
  favours overlap and every method that includes it, which is why "doppel
  (struct-min filtered)" beats plain doppel in the A-union pooled table. A-report
  removes the advantage, which is why the verdict reads it. In setting B the bias
  is total, since no baseline can find a pair that doppel's retrieval did not
  propose first.
- **History labels carry refactor and coupled only.** They say which pairs
  maintainers kept in step or factored apart, not which ones should be merged.
  A ranker tuned to put merges first is expected to do worse on them.

### What would settle it

- More hand labels on a second corpus (gin or chi, as noted throughout
  CLAUDE.md), reviewed from a pooled list that draws on every method's top K,
  not doppel's alone, so the labels stop favouring doppel.
- A **new**, separately pre-registered rule on a denser metric (normalised mean
  rank, or AUC over labelled against unlabelled pairs) in place of P@20 on sparse
  labels, with the false-positive clause scoped to "no false positive in the top
  20 or above the worst merge" rather than a mean. Changing the rule now and
  re-reading these numbers under it would be fitting the rule to the result, so
  this write-up does not do that.

### Not done

- **External clone detector (method 8).** Neither `dupl` nor `jscpd` is
  installed. Installing one means a Go toolchain install of a third-party module
  or an npm global, and the pre-registration said to skip it if it added
  friction. Token clones (method 3) is the in-repo stand-in. *(Done afterwards:
  see the post-registration addendum at the end of this file.)*

### Hand-off to the cost study

`DOPPEL_BENCH_BASELINES_EXPORT=<dir>` (or `task baselines EXPORT=<dir>`) writes
`<corpus>.<pool>.rankings.json` for every public corpus: moby, prometheus, hugo,
gin, cobra and chi in both A pools, and conc in A-union. Each file is a list of
`{corpus, pool, method, pairs: [{rank, a, b}]}`, where `a` and `b` are
`snapshot.Unit.Key` values (sorted within a pair, as a snapshot orders them). It
covers every method above, with random as seed 1. Each list holds the top 500 by
default (`DOPPEL_BENCH_BASELINES_EXPORT_TOP`, 0 for the whole list). The keys are
the identities `doppel analyze --format json` writes, so the time-split cost
study can score each baseline's top pairs on co-change, lagged and unpropagated
fixes exactly as it scores doppel's. That is an outcome no label set here can
supply, and it is not biased toward doppel's output. Private corpora are never
exported.

### Full tables

Generated by `TestBaselines` with `DOPPEL_BENCH_BASELINES_MD`. "ranked/n" is how
many of the n relevant labels the list ranks. An unranked label takes the
expected rank of a random completion. † median is post-hoc.

#### moby/history — A-union (38415 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 38415 | 7534.4 (33/33) | 4757.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 7534.4 | - | 2228.1 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 18545 | 5058.1 (33/33) | 4320.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 5058.1 | - | 1853.6 | 0 | 0 | 0 |
| token clones | 38415 | 10439.2 (33/33) | 7493.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 10439.2 | - | 5542.0 | 0 | 0 | 0 |
| code-shape | 38415 | 11837.8 (33/33) | 10101.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 11837.8 | - | 5487.9 | 0 | 0 | 0 |
| retrieval mass | 38415 | 6237.9 (33/33) | 3709.0 | 0.00 | 0.00 | 0.02 | 1 | 2 | - | 6237.9 | - | 2010.8 | 0 | 0 | 0 |
| overlap | 38415 | 6847.0 (33/33) | 6019.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 6847.0 | - | 3198.8 | 0 | 0 | 0 |
| name heuristic | 16091 | 4640.7 (33/33) | 2895.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 4640.7 | - | 5003.2 | 0 | 0 | 0 |
| random (20 seeds, mean) | 38415 | 19396.2 ±1475.1 (33/33) | 20008.6 | 0.00 | 0.00 ±0.01 | 0.00 | 0 | 0 | - | 19396.2 | - | 19055.2 | 0 | 0 | 0 |

#### moby/history — A-report (18545 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 18545 | 5058.1 (33/33) | 4320.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 5058.1 | - | 1853.6 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 18545 | 5058.1 (33/33) | 4320.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 5058.1 | - | 1853.6 | 0 | 0 | 0 |
| token clones | 18545 | 7286.2 (33/33) | 6285.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 7286.2 | - | 4430.2 | 0 | 0 | 0 |
| code-shape | 18545 | 7851.8 (33/33) | 7678.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 7851.8 | - | 4333.5 | 0 | 0 | 0 |
| retrieval mass | 18545 | 3909.7 (33/33) | 2975.0 | 0.00 | 0.00 | 0.02 | 1 | 2 | - | 3909.7 | - | 1517.7 | 0 | 0 | 0 |
| overlap | 18545 | 6847.0 (33/33) | 6019.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 6847.0 | - | 3198.8 | 0 | 0 | 0 |
| name heuristic | 12795 | 4068.8 (33/33) | 2763.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 4068.8 | - | 3854.5 | 0 | 0 | 0 |
| random (20 seeds, mean) | 18545 | 9316.7 ±1056.8 (33/33) | 9606.1 | 0.01 | 0.00 ±0.01 | 0.00 | 0 | 0 | - | 9316.7 | - | 9370.4 | 0 | 0 | 0 |

#### prometheus/history — A-union (28916 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 28916 | 4904.2 (43/43) | 2369.0 | 0.00 | 0.00 | 0.04 | 2 | 4 | - | 4904.2 | - | 2079.5 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 13027 | 3163.3 (43/43) | 2276.0 | 0.00 | 0.00 | 0.04 | 2 | 4 | - | 3163.3 | - | 1612.5 | 0 | 0 | 0 |
| token clones | 28916 | 4749.9 (43/43) | 3853.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 4749.9 | - | 4776.5 | 0 | 0 | 0 |
| code-shape | 28916 | 5240.7 (43/43) | 3862.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 5240.7 | - | 5676.6 | 0 | 0 | 0 |
| retrieval mass | 28916 | 4283.4 (43/43) | 1838.0 | 0.00 | 0.00 | 0.02 | 1 | 3 | - | 4283.4 | - | 1657.1 | 0 | 0 | 0 |
| overlap | 28916 | 3099.6 (43/43) | 2093.0 | 0.00 | 0.00 | 0.04 | 2 | 2 | - | 3099.6 | - | 3433.7 | 0 | 0 | 0 |
| name heuristic | 14554 | 4355.2 (42/43) | 2561.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 4355.2 | - | 4277.0 | 0 | 0 | 0 |
| random (20 seeds, mean) | 28916 | 14125.2 ±1099.6 (43/43) | 14001.2 | 0.01 | 0.01 ±0.02 | 0.01 | 0 | 0 | - | 14125.2 | - | 14448.3 | 0 | 0 | 0 |

#### prometheus/history — A-report (13027 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 13027 | 3163.3 (43/43) | 2276.0 | 0.00 | 0.00 | 0.04 | 2 | 4 | - | 3163.3 | - | 1612.5 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 13027 | 3163.3 (43/43) | 2276.0 | 0.00 | 0.00 | 0.04 | 2 | 4 | - | 3163.3 | - | 1612.5 | 0 | 0 | 0 |
| token clones | 13027 | 3964.3 (43/43) | 3635.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 3964.3 | - | 3995.7 | 0 | 0 | 0 |
| code-shape | 13027 | 4059.0 (43/43) | 3675.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 4059.0 | - | 4201.6 | 0 | 0 | 0 |
| retrieval mass | 13027 | 2670.4 (43/43) | 1534.0 | 0.00 | 0.00 | 0.04 | 2 | 3 | - | 2670.4 | - | 1184.8 | 0 | 0 | 0 |
| overlap | 13027 | 3099.6 (43/43) | 2093.0 | 0.00 | 0.00 | 0.04 | 2 | 2 | - | 3099.6 | - | 3433.7 | 0 | 0 | 0 |
| name heuristic | 10422 | 3578.1 (42/43) | 2470.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 3578.1 | - | 3532.4 | 0 | 0 | 0 |
| random (20 seeds, mean) | 13027 | 6445.8 ±490.2 (43/43) | 6426.9 | 0.01 | 0.01 ±0.02 | 0.01 | 0 | 0 | - | 6445.8 | - | 6571.6 | 0 | 0 | 0 |

#### hugo/history — A-union (31895 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 31895 | 5335.4 (9/9) | 2895.0 | 0.00 | 0.00 | 0.00 | 0 | 1 | - | 5335.4 | - | 934.2 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 13571 | 2911.7 (9/9) | 2590.0 | 0.00 | 0.00 | 0.00 | 0 | 1 | - | 2911.7 | - | 894.6 | 0 | 0 | 0 |
| token clones | 31895 | 4846.4 (9/9) | 3819.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 4846.4 | - | 3594.5 | 0 | 0 | 0 |
| code-shape | 31895 | 3575.2 (9/9) | 2492.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 3575.2 | - | 3547.7 | 0 | 0 | 0 |
| retrieval mass | 31895 | 5241.4 (9/9) | 2793.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 5241.4 | - | 531.3 | 0 | 0 | 0 |
| overlap | 31895 | 4750.2 (9/9) | 2255.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 4750.2 | - | 2701.2 | 0 | 0 | 0 |
| name heuristic | 13685 | 2889.4 (9/9) | 2034.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 2889.4 | - | 3138.8 | 0 | 0 | 0 |
| random (20 seeds, mean) | 31895 | 14950.3 ±2901.9 (9/9) | 14364.7 | 0.01 | 0.00 ±0.01 | 0.00 | 0 | 0 | - | 14950.3 | - | 16086.5 | 0 | 0 | 0 |

#### hugo/history — A-report (13571 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 13571 | 2911.7 (9/9) | 2590.0 | 0.00 | 0.00 | 0.00 | 0 | 1 | - | 2911.7 | - | 894.6 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 13571 | 2911.7 (9/9) | 2590.0 | 0.00 | 0.00 | 0.00 | 0 | 1 | - | 2911.7 | - | 894.6 | 0 | 0 | 0 |
| token clones | 13571 | 3550.7 (9/9) | 3226.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 3550.7 | - | 2864.2 | 0 | 0 | 0 |
| code-shape | 13571 | 2781.3 (9/9) | 2261.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 2781.3 | - | 2716.8 | 0 | 0 | 0 |
| retrieval mass | 13571 | 2769.2 (9/9) | 2078.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 2769.2 | - | 471.3 | 0 | 0 | 0 |
| overlap | 13571 | 4750.2 (9/9) | 2255.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 4750.2 | - | 2701.2 | 0 | 0 | 0 |
| name heuristic | 10365 | 2576.8 (9/9) | 1896.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 2576.8 | - | 2748.8 | 0 | 0 | 0 |
| random (20 seeds, mean) | 13571 | 6784.3 ±983.5 (9/9) | 7375.6 | 0.00 | 0.00 ±0.00 | 0.00 | 0 | 0 | - | 6784.3 | - | 6875.9 | 0 | 0 | 0 |

#### gin/history — A-union (2116 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 2116 | 80.9 (7/7) | 83.0 | 0.00 | 0.00 | 0.06 | 3 | 6 | - | 80.9 | - | 32.0 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 570 | 61.7 (7/7) | 70.0 | 0.00 | 0.00 | 0.06 | 3 | 6 | - | 61.7 | - | 24.7 | 0 | 0 | 0 |
| token clones | 2116 | 518.9 (7/7) | 531.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 518.9 | - | 365.1 | 0 | 0 | 0 |
| code-shape | 2116 | 652.7 (7/7) | 662.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 652.7 | - | 324.9 | 0 | 0 | 0 |
| retrieval mass | 2116 | 89.6 (7/7) | 85.0 | 0.00 | 0.00 | 0.00 | 0 | 6 | - | 89.6 | - | 69.0 | 0 | 0 | 0 |
| overlap | 2116 | 153.9 (7/7) | 91.0 | 0.00 | 0.00 | 0.00 | 0 | 4 | - | 153.9 | - | 142.3 | 0 | 0 | 0 |
| name heuristic | 1803 | 1043.0 (7/7) | 864.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 1043.0 | - | 412.0 | 0 | 0 | 0 |
| random (20 seeds, mean) | 2116 | 1107.1 ±264.0 (7/7) | 1108.7 | 0.01 | 0.00 ±0.01 | 0.00 | 0 | 0 | - | 1107.1 | - | 991.3 | 0 | 0 | 0 |

#### gin/history — A-report (570 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 570 | 61.7 (7/7) | 70.0 | 0.00 | 0.00 | 0.06 | 3 | 6 | - | 61.7 | - | 24.7 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 570 | 61.7 (7/7) | 70.0 | 0.00 | 0.00 | 0.06 | 3 | 6 | - | 61.7 | - | 24.7 | 0 | 0 | 0 |
| token clones | 570 | 380.7 (7/7) | 417.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 380.7 | - | 301.7 | 0 | 0 | 0 |
| code-shape | 570 | 403.7 (7/7) | 429.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 403.7 | - | 278.6 | 0 | 0 | 0 |
| retrieval mass | 570 | 47.3 (7/7) | 47.0 | 0.00 | 0.00 | 0.12 | 6 | 7 | - | 47.3 | - | 26.1 | 0 | 0 | 0 |
| overlap | 570 | 153.9 (7/7) | 91.0 | 0.00 | 0.00 | 0.00 | 0 | 4 | - | 153.9 | - | 142.3 | 0 | 0 | 0 |
| name heuristic | 570 | 455.6 (7/7) | 459.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 455.6 | - | 305.9 | 0 | 0 | 0 |
| random (20 seeds, mean) | 570 | 282.3 ±49.8 (7/7) | 286.6 | 0.02 | 0.02 ±0.03 | 0.02 | 1 | 1 | - | 282.3 | - | 283.6 | 0 | 0 | 0 |

#### gin/history — B-all-pairs (123256 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 2116 | 80.9 (7/7) | 83.0 | 0.00 | 0.00 | 0.06 | 3 | 6 | - | 80.9 | - | 32.0 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 570 | 61.7 (7/7) | 70.0 | 0.00 | 0.00 | 0.06 | 3 | 6 | - | 61.7 | - | 24.7 | 0 | 0 | 0 |
| token clones | 123256 | 2484.7 (7/7) | 1929.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 2484.7 | - | 1260.6 | 0 | 0 | 0 |
| code-shape | 123256 | 7759.7 (7/7) | 2345.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 7759.7 | - | 713.3 | 0 | 0 | 0 |
| retrieval mass | 2116 | 89.6 (7/7) | 85.0 | 0.00 | 0.00 | 0.00 | 0 | 6 | - | 89.6 | - | 69.0 | 0 | 0 | 0 |
| overlap | 123256 | 231.0 (7/7) | 115.0 | 0.00 | 0.00 | 0.00 | 0 | 3 | - | 231.0 | - | 214.7 | 0 | 0 | 0 |
| name heuristic | 56845 | 22172.6 (7/7) | 10450.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 22172.6 | - | 1171.9 | 0 | 0 | 0 |
| random (20 seeds, mean) | 123256 | 64018.2 ±15031.6 (7/7) | 62241.6 | 0.00 | 0.00 ±0.00 | 0.00 | 0 | 0 | - | 64018.2 | - | 60641.8 | 0 | 0 | 0 |

#### cobra/hand — A-union (1401 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 1401 | 11.0 (15/15) | 9.0 | 0.80 | 0.70 | 0.30 | 15 | 15 | 4.8 | 15.1 | 513.5 | - | 0 | 0 | 0 |
| doppel (struct-min filtered) | 239 | 10.8 (15/15) | 9.0 | 0.80 | 0.70 | 0.30 | 15 | 15 | 4.8 | 14.8 | 566.7 | - | 0 | 0 | 0 |
| token clones | 1401 | 33.7 (15/15) | 23.0 | 0.00 | 0.25 | 0.26 | 13 | 14 | 17.7 | 44.4 | 567.2 | - | 0 | 0 | 0 |
| code-shape | 1401 | 29.5 (15/15) | 25.0 | 0.00 | 0.20 | 0.26 | 13 | 15 | 21.7 | 34.8 | 563.2 | - | 0 | 0 | 0 |
| retrieval mass | 1401 | 29.7 (15/15) | 24.0 | 0.20 | 0.30 | 0.24 | 12 | 15 | 20.3 | 35.9 | 540.5 | - | 0 | 0 | 0 |
| overlap | 1401 | 15.3 (15/15) | 15.0 | 0.60 | 0.60 | 0.28 | 14 | 15 | 17.3 | 13.9 | 623.2 | - | 0 | 1 | 1 |
| name heuristic | 1211 | 100.4 (15/15) | 68.0 | 0.00 | 0.10 | 0.10 | 5 | 9 | 65.8 | 123.4 | 865.8 | - | 0 | 1 | 0 |
| random (20 seeds, mean) | 1401 | 665.7 ±117.1 (15/15) | 694.4 | 0.01 | 0.01 ±0.03 | 0.02 | 1 | 2 | 684.0 | 653.5 | 1024.1 | - | 0 | 2 | 0 |

#### cobra/hand — A-report (239 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 239 | 10.8 (15/15) | 9.0 | 0.80 | 0.70 | 0.30 | 15 | 15 | 4.8 | 14.8 | 179.3 | - | 0 | 0 | 0 |
| doppel (struct-min filtered) | 239 | 10.8 (15/15) | 9.0 | 0.80 | 0.70 | 0.30 | 15 | 15 | 4.8 | 14.8 | 179.3 | - | 0 | 0 | 0 |
| token clones | 239 | 28.0 (15/15) | 20.0 | 0.20 | 0.40 | 0.26 | 13 | 15 | 14.7 | 36.9 | 191.0 | - | 0 | 0 | 0 |
| code-shape | 239 | 24.3 (15/15) | 21.0 | 0.00 | 0.35 | 0.30 | 15 | 15 | 18.2 | 28.4 | 191.3 | - | 0 | 0 | 0 |
| retrieval mass | 239 | 20.4 (15/15) | 18.0 | 0.20 | 0.50 | 0.30 | 15 | 15 | 15.3 | 23.8 | 188.0 | - | 0 | 0 | 0 |
| overlap | 239 | 15.3 (15/15) | 15.0 | 0.60 | 0.60 | 0.28 | 14 | 15 | 17.3 | 13.9 | 162.0 | - | 0 | 1 | 1 |
| name heuristic | 239 | 63.3 (15/15) | 55.0 | 0.10 | 0.10 | 0.14 | 7 | 13 | 50.0 | 72.1 | 185.7 | - | 0 | 1 | 0 |
| random (20 seeds, mean) | 239 | 117.4 ±15.2 (15/15) | 122.5 | 0.05 | 0.08 ±0.05 | 0.07 | 4 | 6 | 121.3 | 114.8 | 199.6 | - | 0 | 1 | 0 |

#### cobra/hand — B-all-pairs (36046 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 1401 | 11.0 (15/15) | 9.0 | 0.80 | 0.70 | 0.30 | 15 | 15 | 4.8 | 15.1 | 6287.7 | - | 0 | 0 | 0 |
| doppel (struct-min filtered) | 239 | 10.8 (15/15) | 9.0 | 0.80 | 0.70 | 0.30 | 15 | 15 | 4.8 | 14.8 | 12115.0 | - | 0 | 0 | 0 |
| token clones | 36046 | 96.7 (15/15) | 84.0 | 0.00 | 0.00 | 0.00 | 0 | 12 | 77.2 | 109.7 | 192.3 | - | 0 | 1 | 0 |
| code-shape | 36046 | 77.5 (15/15) | 83.0 | 0.00 | 0.00 | 0.06 | 3 | 13 | 53.7 | 93.4 | 149.3 | - | 0 | 1 | 1 |
| retrieval mass | 1401 | 29.7 (15/15) | 24.0 | 0.20 | 0.30 | 0.24 | 12 | 15 | 20.3 | 35.9 | 6314.7 | - | 0 | 0 | 0 |
| overlap | 36046 | 16.0 (15/15) | 16.0 | 0.60 | 0.55 | 0.28 | 14 | 15 | 18.3 | 14.4 | 357.0 | - | 0 | 1 | 1 |
| name heuristic | 28876 | 491.8 (15/15) | 131.0 | 0.00 | 0.00 | 0.04 | 2 | 5 | 151.5 | 718.7 | 20153.2 | - | 0 | 1 | 0 |
| random (20 seeds, mean) | 36046 | 18673.1 ±2538.8 (15/15) | 18701.5 | 0.00 | 0.00 ±0.00 | 0.00 | 0 | 0 | 18777.8 | 18603.3 | 18788.2 | - | 0 | 3 | 0 |

#### cobra/history — A-union (1401 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 1401 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 113.5 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 239 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 86.5 | 0 | 0 | 0 |
| token clones | 1401 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 87.2 | 0 | 0 | 0 |
| code-shape | 1401 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 510.0 | 0 | 0 | 0 |
| retrieval mass | 1401 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 90.5 | 0 | 0 | 0 |
| overlap | 1401 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 81.8 | 0 | 0 | 0 |
| name heuristic | 1211 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 288.8 | 0 | 0 | 0 |
| random (20 seeds, mean) | 1401 | 0.0 ±0.0 (0/0) | 0.0 | 0.00 | 0.00 ±0.00 | 0.00 | 0 | 0 | - | - | - | 713.3 | 0 | 0 | 0 |

#### cobra/history — A-report (239 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 239 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 86.5 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 239 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 86.5 | 0 | 0 | 0 |
| token clones | 239 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 67.2 | 0 | 0 | 0 |
| code-shape | 239 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 128.0 | 0 | 0 | 0 |
| retrieval mass | 239 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 51.5 | 0 | 0 | 0 |
| overlap | 239 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 81.8 | 0 | 0 | 0 |
| name heuristic | 239 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 122.8 | 0 | 0 | 0 |
| random (20 seeds, mean) | 239 | 0.0 ±0.0 (0/0) | 0.0 | 0.00 | 0.00 ±0.00 | 0.00 | 0 | 0 | - | - | - | 126.8 | 0 | 0 | 0 |

#### cobra/history — B-all-pairs (36046 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 1401 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 113.5 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 239 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 86.5 | 0 | 0 | 0 |
| token clones | 36046 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 178.0 | 0 | 0 | 0 |
| code-shape | 36046 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 9355.8 | 0 | 0 | 0 |
| retrieval mass | 1401 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 90.5 | 0 | 0 | 0 |
| overlap | 36046 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 95.5 | 0 | 0 | 0 |
| name heuristic | 28876 | 0.0 (0/0) | 0.0 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | - | - | 3486.8 | 0 | 0 | 0 |
| random (20 seeds, mean) | 36046 | 0.0 ±0.0 (0/0) | 0.0 | 0.00 | 0.00 ±0.00 | 0.00 | 0 | 0 | - | - | - | 18181.4 | 0 | 0 | 0 |

#### chi/history — A-union (814 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 814 | 9.5 (6/6) | 9.5 | 0.40 | 0.30 | 0.12 | 6 | 6 | - | 9.5 | - | 2.5 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 124 | 9.5 (6/6) | 9.5 | 0.40 | 0.30 | 0.12 | 6 | 6 | - | 9.5 | - | 2.5 | 0 | 0 | 0 |
| token clones | 814 | 7.7 (6/6) | 7.5 | 0.50 | 0.30 | 0.12 | 6 | 6 | - | 7.7 | - | 60.5 | 0 | 0 | 0 |
| code-shape | 814 | 4.7 (6/6) | 4.5 | 0.60 | 0.30 | 0.12 | 6 | 6 | - | 4.7 | - | 48.0 | 0 | 0 | 0 |
| retrieval mass | 814 | 209.5 (6/6) | 209.5 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 209.5 | - | 6.5 | 0 | 0 | 0 |
| overlap | 814 | 36.7 (6/6) | 36.5 | 0.00 | 0.00 | 0.12 | 6 | 6 | - | 36.7 | - | 40.5 | 0 | 0 | 0 |
| name heuristic | 735 | 4.7 (6/6) | 4.5 | 0.60 | 0.30 | 0.12 | 6 | 6 | - | 4.7 | - | 42.0 | 0 | 0 | 0 |
| random (20 seeds, mean) | 814 | 401.2 ±99.3 (6/6) | 419.0 | 0.01 | 0.01 ±0.02 | 0.01 | 1 | 1 | - | 401.2 | - | 382.3 | 0 | 0 | 0 |

#### chi/history — A-report (124 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 124 | 9.5 (6/6) | 9.5 | 0.40 | 0.30 | 0.12 | 6 | 6 | - | 9.5 | - | 2.5 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 124 | 9.5 (6/6) | 9.5 | 0.40 | 0.30 | 0.12 | 6 | 6 | - | 9.5 | - | 2.5 | 0 | 0 | 0 |
| token clones | 124 | 5.7 (6/6) | 5.5 | 0.60 | 0.30 | 0.12 | 6 | 6 | - | 5.7 | - | 53.5 | 0 | 0 | 0 |
| code-shape | 124 | 4.7 (6/6) | 4.5 | 0.60 | 0.30 | 0.12 | 6 | 6 | - | 4.7 | - | 47.0 | 0 | 0 | 0 |
| retrieval mass | 124 | 43.5 (6/6) | 43.5 | 0.00 | 0.00 | 0.12 | 6 | 6 | - | 43.5 | - | 5.0 | 0 | 0 | 0 |
| overlap | 124 | 36.7 (6/6) | 36.5 | 0.00 | 0.00 | 0.12 | 6 | 6 | - | 36.7 | - | 40.5 | 0 | 0 | 0 |
| name heuristic | 123 | 4.7 (6/6) | 4.5 | 0.60 | 0.30 | 0.12 | 6 | 6 | - | 4.7 | - | 22.0 | 0 | 0 | 0 |
| random (20 seeds, mean) | 124 | 61.2 ±14.8 (6/6) | 62.3 | 0.04 | 0.05 ±0.04 | 0.05 | 3 | 5 | - | 61.2 | - | 58.5 | 0 | 0 | 0 |

#### chi/history — B-all-pairs (16653 pairs)

| method | ranked | merge+refactor mean (ranked/n) | median† | P@10 | P@20 | P@50 | hits@50 | hits@100 | merge | refactor | false positive | coupled | merges missing | fp above merge | fp in top 20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| doppel | 814 | 9.5 (6/6) | 9.5 | 0.40 | 0.30 | 0.12 | 6 | 6 | - | 9.5 | - | 2.5 | 0 | 0 | 0 |
| doppel (struct-min filtered) | 124 | 9.5 (6/6) | 9.5 | 0.40 | 0.30 | 0.12 | 6 | 6 | - | 9.5 | - | 2.5 | 0 | 0 | 0 |
| token clones | 16653 | 19.7 (6/6) | 19.5 | 0.00 | 0.15 | 0.12 | 6 | 6 | - | 19.7 | - | 81.5 | 0 | 0 | 0 |
| code-shape | 16653 | 7.7 (6/6) | 7.5 | 0.50 | 0.30 | 0.12 | 6 | 6 | - | 7.7 | - | 69.0 | 0 | 0 | 0 |
| retrieval mass | 814 | 209.5 (6/6) | 209.5 | 0.00 | 0.00 | 0.00 | 0 | 0 | - | 209.5 | - | 6.5 | 0 | 0 | 0 |
| overlap | 16653 | 43.7 (6/6) | 43.5 | 0.00 | 0.00 | 0.12 | 6 | 6 | - | 43.7 | - | 47.5 | 0 | 0 | 0 |
| name heuristic | 8491 | 5.7 (6/6) | 5.5 | 0.60 | 0.30 | 0.12 | 6 | 6 | - | 5.7 | - | 91.5 | 0 | 0 | 0 |
| random (20 seeds, mean) | 16653 | 7986.7 ±2260.8 (6/6) | 7959.7 | 0.00 | 0.00 ±0.00 | 0.00 | 0 | 0 | - | 7986.7 | - | 7407.6 | 0 | 0 | 0 |

#### Decision rule, applied

| baseline | cobra hand, A-union | history wins (A-report), strict | history wins, weak (P@20 not lower) | beaten (strict) | beaten (weak) |
| --- | --- | --- | --- | --- | --- |
| token clones | P@20 0.70 vs 0.25 ✓; rel 11.0 vs 33.7 ✓; fp 513.5 vs 567.2 ✗ — strict no, weak no | 0/4  | 4/4 gin,prometheus,hugo,moby | **no** | no |
| code-shape | P@20 0.70 vs 0.20 ✓; rel 11.0 vs 29.5 ✓; fp 513.5 vs 563.2 ✗ — strict no, weak no | 0/4  | 3/4 gin,prometheus,moby | **no** | no |
| retrieval mass | P@20 0.70 vs 0.30 ✓; rel 11.0 vs 29.7 ✓; fp 513.5 vs 540.5 ✗ — strict no, weak no | 0/4  | 0/4  | **no** | no |
| overlap | P@20 0.70 vs 0.60 ✓; rel 11.0 vs 15.3 ✓; fp 513.5 vs 623.2 ✗ — strict no, weak no | 0/4  | 3/4 gin,hugo,moby | **no** | no |
| name heuristic | P@20 0.70 vs 0.10 ✓; rel 11.0 vs 100.4 ✓; fp 513.5 vs 865.8 ✗ — strict no, weak no | 0/4  | 2/4 gin,prometheus | **no** | no |
| random | P@20 0.70 vs 0.01 ✓; rel 11.0 vs 665.7 ✓; fp 513.5 vs 1024.1 ✗ — strict no, weak no | 0/4  | 1/4 hugo | **no** | no |

#### Pooled (post-hoc) — history labels, A-union

Mean rank divided by pool size, averaged over the corpora listed; 0 is the top of the list, 0.5 is what random ordering gives.

| method | refactor (normalised mean) | coupled (normalised mean) | corpora where it beats doppel on refactor mean |
| --- | ---: | ---: | --- |
| doppel | 0.117 | 0.043 | — |
| doppel (struct-min filtered) | 0.075 | 0.035 | 4/5 moby, prometheus, hugo, gin |
| token clones | 0.169 | 0.122 | 3/5 prometheus, hugo, chi |
| code-shape | 0.183 | 0.171 | 2/5 hugo, chi |
| retrieval mass | 0.155 | 0.039 | 3/5 moby, prometheus, hugo |
| overlap | 0.110 | 0.077 | 3/5 moby, prometheus, hugo |
| name heuristic | 0.172 | 0.138 | 4/5 moby, prometheus, hugo, chi |
| random | 0.496 | 0.491 | 0/5  |

#### Pooled (post-hoc) — history labels, A-report

Mean rank divided by pool size, averaged over the corpora listed; 0 is the top of the list, 0.5 is what random ordering gives.

| method | refactor (normalised mean) | coupled (normalised mean) | corpora where it beats doppel on refactor mean |
| --- | ---: | ---: | --- |
| doppel | 0.183 | 0.119 | — |
| doppel (struct-min filtered) | 0.183 | 0.119 | 0/5  |
| token clones | 0.334 | 0.333 | 1/5 chi |
| code-shape | 0.337 | 0.360 | 2/5 hugo, chi |
| retrieval mass | 0.211 | 0.085 | 4/5 moby, prometheus, hugo, gin |
| overlap | 0.305 | 0.259 | 1/5 prometheus |
| name heuristic | 0.304 | 0.318 | 3/5 moby, hugo, chi |
| random | 0.497 | 0.503 | 0/5  |

#### Pooled — setting B, labelled relevant pairs in the top 50 / top 100

Summed over every label set scored all-pairs (cobra hand, and the cobra, chi and gin history labels).

| method | top 50 | top 100 | of |
| --- | ---: | ---: | ---: |
| doppel | 24 | 27 | 28 |
| doppel (struct-min filtered) | 24 | 27 | 28 |
| token clones | 6 | 18 | 28 |
| code-shape | 9 | 19 | 28 |
| retrieval mass | 12 | 21 | 28 |
| overlap | 20 | 24 | 28 |
| name heuristic | 8 | 11 | 28 |
| random | 0 | 0 | 28 |

## Note (post-hoc): the history-label result is largely leakage

Added after the addendum below existed. Nothing above was changed. Every
history "refactor" label is an `extracted` verdict: the extraction happened
before the pin, and the pair is scored in its state at the pin. In about 78 of
the 96 such labels, both sides call the extracted helper there. That helper is a
call-channel token, and the emptied bodies read low on code-shape and trophic.
So "retrieval mass alone ranks history-relevant pairs better" mostly measures
recognising refactors that have already happened. The factor that pushes these
pairs down is mainly trophic², not shape². `examples/ranker-outcomes.md` ranks at
an earlier revision T and judges the pairs on what happened afterwards. There,
mass and the full key are not distinguishable on any corpus, and the key beats
code-shape, token clones, the name heuristic and random.

## Addendum (post-registration): method 8, an external clone detector

Added on 2026-10-10, after every result above existed. The pre-registration
section is unchanged. This method is scored under the **same rule, metrics,
settings and imputation** as methods 1–7, and nothing above was re-run
differently to make room for it. With the clone files left out, `TestBaselines`
reproduces the tables above byte for byte. With them included, the only change
is the new rows, and two runs are byte-identical.

### What was run

- **Detector: `dupl` v1.1.0** (`github.com/mibk/dupl`), a token-sequence clone
  finder over the Go AST. It installed with a single `go install` into GOBIN. `jscpd` was
  not tried, because a second detector would have added an npm toolchain to
  answer the same question. `go.mod` is untouched.
- **It runs outside the module.** `scripts/clone-baseline.sh` (`task
  clone-baseline`) runs dupl on each fetched ladder rung. It writes
  `<corpus>.dupl-t<N>.clones.json` (clone groups as file, start line, end line)
  to a cache directory outside the repo. The bench reads those files only when
  `DOPPEL_BENCH_BASELINES_CLONES=<dir>` is set (`task baselines CLONES=<dir>`).
  Without that variable the method does not appear in any table. It is never
  scored as an empty list.
- **Population**: non-test `.go` files. The script skips the directories the
  pipeline's walk skips (`parser.DefaultExcludes`, read out of the source) and
  files carrying Go's generated-code marker. A fragment in a file outside
  doppel's population maps to no function, so a difference between the two
  populations can never add pairs.
- **Two thresholds, both fixed before the scoring run.** The method-8 row is
  `dupl (t=100, default)`, the tool's own operating point and the counterpart of
  doppel at its production default. `dupl (t=25)` is a sensitivity row. At its
  default, dupl reports 0–3 clone groups on each of chi, cobra, conc and gin,
  which would leave the comparison nearly empty on the small rungs.

### Mapping clones to function pairs

A fragment covers every function whose `[StartLine, end]` span it overlaps. The
`end` line is the closing brace: `StartLine` plus the number of newlines in
`Body`. Within one clone group, each pair of fragments yields every cross pair
of the functions the two fragments cover. That pair is keyed by the **smaller of
the two overlaps, in lines**: the detector's clone size, clipped to the two
functions. dupl reports no token count, so lines are the only size it gives. A
pair reached through several fragment pairs keeps its largest key.

Ties break on `(AIdx, BIdx)`, as for every other method. Pairs outside one build
unit (`parser.SameBuildUnit`) are dropped. In setting A the list is restricted
to the pool. In both settings a pair the detector never reported is
**unranked**, and its label gets the imputed rank `(len(list) + 1 + N) / 2`,
the same as an unranked label under any other method.

| corpus | t=100: groups → function pairs | t=25: groups → function pairs |
| --- | ---: | ---: |
| moby | 43 → 943 | 991 → 10 590 |
| prometheus | 80 → 114 | 1 022 → 7 159 |
| hugo | 5 → 3 | 501 → 1 533 |
| gin | 3 → 120 | 35 → 554 |
| cobra | 1 → 1 | 37 → 86 |
| chi | 0 → 0 | 19 → 61 |
| conc | 0 → 0 | 6 → 17 |

### Verdict, by the pre-registered rule

**doppel beats neither dupl row**, under the strict rule or the weak one. The
reasons are the same two as for every other baseline:

1. **Condition 1 fails on the false-positive clause**, and here the failure is
   degenerate. dupl reports none of cobra's three false positives, and at t=100
   it reports only one cobra pair in total. So all three take the imputed rank:
   701.5 at t=100 and 733.5 at t=25, against doppel's 513.5. A method that ranks
   almost nothing passes this clause by abstaining. doppel wins the other two
   clauses easily: P@20 0.70 against 0.05 and 0.20, and merge+refactor mean 11.0
   against 654.8 and 262.6.
2. **Condition 2: strict 0/4, weak 3/4 (gin, hugo, moby) for both rows.**
   Prometheus is the exception, and this result is new. **dupl reaches P@20 0.10
   on prometheus's history labels, against 0.00 for doppel.** No other method in
   this study scores a non-zero P@20 on any of the four larger history label
   sets. At t=100 its top 20 there holds two refactor labels
   (`FloatHistogram.Add ↔ Sub` and `memSeries.appendFloatHistogram ↔
   appendHistogram`) and three coupled labels. Much of the rest is unlabelled
   float/int histogram twins.

The verdict section above already describes the rule's weaknesses. This method
makes the false-positive one sharper. That is stated here, not used as a reason
to set the result aside.

### What the numbers show (descriptive, post-hoc)

- **dupl is precise but finds very few of these labels.** The four larger
  corpora carry 92 history-relevant pairs. t=100 ranks 3 of them and t=25 ranks
  31 (moby 7/33, prometheus 21/43, hugo 3/9, gin 0/7). In A-report, the pooled
  normalised refactor rank is 0.498 at t=100 and 0.365 at t=25, against 0.183
  for doppel and 0.497 for random. For most pairs that maintainers kept in step
  or factored apart, dupl finds no shared sequence of 25 tokens or more.
- **Where it does report a pair, it is good.** At t=25 it ranks all six of
  cobra's merges, with the best merge mean of any baseline: 14.2 in A-union,
  against 17.3 for overlap and 17.7 for token clones (doppel 4.8). Its P@10 is
  0.40. It ranks only 4 of the 9 refactors, which puts its merge+refactor mean
  at 262.6. On chi's history labels t=25 is the best method outright, with an
  A-report mean of 6.5 against doppel's 9.5. Code-shape, token clones and the
  name heuristic also beat doppel there.
- **Setting B (all pairs)**: summed over the cobra hand labels and the chi and
  gin history labels, labelled relevant pairs in the top 50 / top 100 are 1 / 1
  for t=100 and 14 / 16 for t=25. doppel has 24 / 27, overlap 20 / 24 and token
  clones 6 / 18. The bias noted above applies: every label here came from
  doppel's output.
- **Most of dupl's pairs fall outside doppel's retrieval union.** At t=25 the
  union holds 2 190 of moby's 10 590 dupl pairs, 1 782 of prometheus's 7 159 and
  200 of gin's 554. These pairs are unlabelled, so this measurement cannot say
  whether they are recall doppel lacks or fragment overlaps that do not make two
  functions alike. The export now carries dupl's ranked lists, so the cost
  study's outcome can answer that without hand labels.

### Tables (dupl rows only; every other row is as above)

The columns and the imputation are the same as in the full tables. The last
column repeats doppel's merge+refactor mean and P@20 for the same label set and
setting. cobra/history has no relevant labels and is left out.

| label set — setting | method | ranked | merge+refactor mean (ranked/n) | P@10 | P@20 | hits@50 | merge | refactor | false positive | coupled | doppel: mean, P@20 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| moby/history — A-union (38415) | dupl (t=100, default) | 147 | 19281.5 (0/33) | 0.00 | 0.00 | 0 | - | 19281.5 | - | 15832.7 | 7534.4, 0.00 |
| moby/history — A-union (38415) | dupl (t=25) | 2190 | 16200.5 (7/33) | 0.00 | 0.00 | 0 | - | 16200.5 | - | 11987.6 | 7534.4, 0.00 |
| moby/history — A-report (18545) | dupl (t=100, default) | 145 | 9345.5 (0/33) | 0.00 | 0.00 | 0 | - | 9345.5 | - | 7676.2 | 5058.1, 0.00 |
| moby/history — A-report (18545) | dupl (t=25) | 1989 | 8282.2 (7/33) | 0.00 | 0.00 | 0 | - | 8282.2 | - | 6135.9 | 5058.1, 0.00 |
| prometheus/history — A-union (28916) | dupl (t=100, default) | 88 | 13492.6 (3/43) | 0.10 | 0.10 | 2 | - | 13492.6 | - | 12520.7 | 4904.2, 0.00 |
| prometheus/history — A-union (28916) | dupl (t=25) | 1782 | 8257.6 (21/43) | 0.10 | 0.10 | 2 | - | 8257.6 | - | 7075.4 | 4904.2, 0.00 |
| prometheus/history — A-report (13027) | dupl (t=100, default) | 88 | 6102.3 (3/43) | 0.10 | 0.10 | 2 | - | 6102.3 | - | 5664.5 | 3163.3, 0.00 |
| prometheus/history — A-report (13027) | dupl (t=25) | 1636 | 4135.9 (21/43) | 0.10 | 0.10 | 2 | - | 4135.9 | - | 3544.0 | 3163.3, 0.00 |
| hugo/history — A-union (31895) | dupl (t=100, default) | 3 | 15949.5 (0/9) | 0.00 | 0.00 | 0 | - | 15949.5 | - | 15949.5 | 5335.4, 0.00 |
| hugo/history — A-union (31895) | dupl (t=25) | 920 | 11093.8 (3/9) | 0.00 | 0.00 | 0 | - | 11093.8 | - | 9652.3 | 5335.4, 0.00 |
| hugo/history — A-report (13571) | dupl (t=100, default) | 2 | 6787.0 (0/9) | 0.00 | 0.00 | 0 | - | 6787.0 | - | 6787.0 | 2911.7, 0.00 |
| hugo/history — A-report (13571) | dupl (t=25) | 857 | 4958.3 (3/9) | 0.00 | 0.00 | 0 | - | 4958.3 | - | 4286.3 | 2911.7, 0.00 |
| gin/history — A-union (2116) | dupl (t=100, default) | 34 | 1075.5 (0/7) | 0.00 | 0.00 | 0 | - | 1075.5 | - | 1075.5 | 80.9, 0.00 |
| gin/history — A-union (2116) | dupl (t=25) | 200 | 1158.5 (0/7) | 0.00 | 0.00 | 0 | - | 1158.5 | - | 1158.5 | 80.9, 0.00 |
| gin/history — A-report (570) | dupl (t=100, default) | 16 | 293.5 (0/7) | 0.00 | 0.00 | 0 | - | 293.5 | - | 293.5 | 61.7, 0.00 |
| gin/history — A-report (570) | dupl (t=25) | 162 | 366.5 (0/7) | 0.00 | 0.00 | 0 | - | 366.5 | - | 366.5 | 61.7, 0.00 |
| gin/history — B-all-pairs (123256) | dupl (t=100, default) | 120 | 61688.5 (0/7) | 0.00 | 0.00 | 0 | - | 61688.5 | - | 61688.5 | 80.9, 0.00 |
| gin/history — B-all-pairs (123256) | dupl (t=25) | 554 | 61905.5 (0/7) | 0.00 | 0.00 | 0 | - | 61905.5 | - | 61905.5 | 80.9, 0.00 |
| cobra/hand — A-union (1401) | dupl (t=100, default) | 1 | 654.8 (1/15) | 0.10 | 0.05 | 1 | 584.8 | 701.5 | 701.5 | - | 11.0, 0.70 |
| cobra/hand — A-union (1401) | dupl (t=25) | 65 | 262.6 (10/15) | 0.40 | 0.20 | 8 | 14.2 | 428.3 | 733.5 | - | 11.0, 0.70 |
| cobra/hand — A-report (239) | dupl (t=100, default) | 1 | 112.5 (1/15) | 0.10 | 0.05 | 1 | 100.6 | 120.5 | 120.5 | - | 10.8, 0.70 |
| cobra/hand — A-report (239) | dupl (t=25) | 56 | 65.1 (10/15) | 0.40 | 0.20 | 9 | 12.5 | 100.1 | 148.0 | - | 10.8, 0.70 |
| cobra/hand — B-all-pairs (36046) | dupl (t=100, default) | 1 | 16822.5 (1/15) | 0.10 | 0.05 | 1 | 15020.2 | 18024.0 | 18024.0 | - | 11.0, 0.70 |
| cobra/hand — B-all-pairs (36046) | dupl (t=25) | 86 | 6041.7 (10/15) | 0.40 | 0.20 | 8 | 14.2 | 10060.1 | 12072.7 | - | 11.0, 0.70 |
| chi/history — A-union (814) | dupl (t=100, default) | 0 | 407.5 (0/6) | 0.00 | 0.00 | 0 | - | 407.5 | - | 407.5 | 9.5, 0.30 |
| chi/history — A-union (814) | dupl (t=25) | 50 | 7.5 (6/6) | 0.60 | 0.30 | 6 | - | 7.5 | - | 216.8 | 9.5, 0.30 |
| chi/history — A-report (124) | dupl (t=100, default) | 0 | 62.5 (0/6) | 0.00 | 0.00 | 0 | - | 62.5 | - | 62.5 | 9.5, 0.30 |
| chi/history — A-report (124) | dupl (t=25) | 48 | 6.5 (6/6) | 0.60 | 0.30 | 6 | - | 6.5 | - | 43.8 | 9.5, 0.30 |
| chi/history — B-all-pairs (16653) | dupl (t=100, default) | 0 | 8327.0 (0/6) | 0.00 | 0.00 | 0 | - | 8327.0 | - | 8327.0 | 9.5, 0.30 |
| chi/history — B-all-pairs (16653) | dupl (t=25) | 61 | 7.5 (6/6) | 0.60 | 0.30 | 6 | - | 7.5 | - | 4179.2 | 9.5, 0.30 |

The decision rule rows, as `TestBaselines` prints them:

| baseline | cobra hand, A-union | history wins (A-report), strict | history wins, weak (P@20 not lower) | beaten (strict) | beaten (weak) |
| --- | --- | --- | --- | --- | --- |
| dupl (t=100, default) | P@20 0.70 vs 0.05 ✓; rel 11.0 vs 654.8 ✓; fp 513.5 vs 701.5 ✗ — strict no, weak no | 0/4  | 3/4 gin,hugo,moby | **no** | no |
| dupl (t=25) | P@20 0.70 vs 0.20 ✓; rel 11.0 vs 262.6 ✓; fp 513.5 vs 733.5 ✗ — strict no, weak no | 0/4  | 3/4 gin,hugo,moby | **no** | no |

Pooled normalised mean rank on the history labels (post-hoc):

| method | A-union refactor | A-union coupled | A-report refactor | A-report coupled |
| --- | ---: | ---: | ---: | ---: |
| doppel | 0.117 | 0.043 | 0.183 | 0.119 |
| dupl (t=100, default) | 0.495 | 0.476 | 0.498 | 0.479 |
| dupl (t=25) | 0.322 | 0.345 | 0.365 | 0.400 |
| random | 0.496 | 0.491 | 0.497 | 0.503 |

### Export

When `DOPPEL_BENCH_BASELINES_CLONES` is set, `DOPPEL_BENCH_BASELINES_EXPORT` also
writes `dupl (t=100, default)` and `dupl (t=25)` into every
`<corpus>.<pool>.rankings.json`. Each of these lists holds only the pool pairs
the detector reported, so it is often shorter than the top-500 cut.

### Reproduce

```bash
go install github.com/mibk/dupl@v1.1.0
task corpora
task clone-baseline OUT=<dir>
task baselines CLONES=<dir> MD=<results.md> EXPORT=<rankings dir>
```

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

_Not yet run._

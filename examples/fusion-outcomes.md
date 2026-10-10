# Does fusing doppel with a clone detector beat either one?

A measurement, not a scoring change: nothing in doppel's ranking, scoring or
defaults is touched to produce it. It follows `examples/clone-outcomes.md`,
which ranked each corpus at the cost study's revision T under doppel and under
clone detectors that search the whole tree, and judged each listed pair on
what happened over T..pin. That study found doppel **not distinguishable** from
dupl, at either threshold, on M1@100.

It also found that the two reach different pairs:

- doppel reaches small function families below any clone floor (gin's
  `Engine.Run*`, 10-16 lines, which one commit changed together);
- dupl reaches large families that one sweeping commit touches (moby's
  `client.*Prune`, six pairs at M1 1.0, none of them in doppel's top 100).

If each list holds costly pairs the other misses, merging them could beat both.
Reciprocal Rank Fusion (Cormack, Clarke & Büttcher, SIGIR 2009) merges ranked
lists with no training and no score calibration, which is what two lists with
incomparable keys (doppel's evidence key, dupl's clone size in lines) need.

The question: **does an RRF fusion of doppel with a clone detector find pairs
that later cost maintainers duplicated work better than doppel alone, and
better than the clone detector alone?**

## Pre-registration

Written and committed **before** any fused list was judged against any outcome.
Nothing in this section may change after results exist. Results and the
verdict go in sections below it.

### Design

- **Corpora, T, pin and population.** Exactly those of
  `examples/clone-outcomes.md`: cobra, gin, prometheus, hugo, moby, with the
  `since` and `pin` of `examples/cost-study/<corpus>.cost.json`; Go files only,
  test and generated files excluded.
- **Inputs.** Four lists, reproduced from the clone-outcome study and not
  re-derived: `doppel`, `dupl (t=100, default)`, `dupl (t=50)` and
  `token clones (≥50 nodes)`, each exactly as `cloneOutcomeLists` builds it
  (`internal/bench/clone_rankings_test.go`), over the same dupl runs at T. Each
  is cut at rank 500 before fusion, as the clone-outcome study cut it.
- **Fused methods.**

  | method | inputs |
  | --- | --- |
  | `RRF(doppel, dupl (t=100, default))` | doppel and dupl at its default threshold |
  | `RRF(doppel, dupl (t=50))` | doppel and dupl at the sensitivity threshold |
  | `RRF(doppel, token clones (≥50 nodes))` | doppel and the floored all-pairs token detector |

- **Fusion.** For each pair in either input list, `score = Σ 1/(k + rank)` over
  the inputs that list it, with `rank` 1-based within that input's 500-cut
  list and **k = 60 fixed** (the paper's value; not tuned). A pair absent from
  an input contributes nothing for it. The sum is taken in input order (doppel
  first), so it is the same float on every run.
- **Tie-break,** in order: higher score; then the better (smaller) best single
  rank over the inputs; then unit index `(a, b)` ascending, the tie-break every
  earlier list used. The fused list is cut at rank 500.
- **Why RRF and why k = 60.** RRF needs only ranks, so no key is rescaled; k
  damps the advantage of the very top ranks, and 60 is the published default.
  With k = 60 a pair both inputs list at rank 30 (2/90 ≈ 0.022) outranks a pair
  one input lists at rank 1 (1/61 ≈ 0.016): agreement is rewarded, and below
  that the fused list interleaves the two inputs rank by rank. No other k, and
  no weighted fusion, is scored.
- **Outcomes.** `historylabel compare`, unchanged: the cost study's walk, judge
  and exclusions over T..pin, each distinct pair judged once.

### Metrics

As in `examples/clone-outcomes.md`: **M1** (mean `cochanges / min(editsA,
editsB)`, 0 when a side was never edited), M2, E, C, clustered M1, and the
overlap table (overlap with doppel's top 100, median smallest side in lines at
T, pairs with an event).

Added, descriptive: for each fused list, the **provenance of its top 100** —
how many pairs are in both inputs' 500-cut lists, in doppel's only, and in the
clone detector's only.

### Primary comparisons and decision rule

- **Primary metric: M1 at K = 100**, over the pairs a list has after dropped
  pairs; nothing is padded. A fused list always has doppel's 500 pairs to draw
  on, so it always has 100.
- **Two contrasts per fused method:** fused minus doppel, and fused minus its
  clone-detector input. Each per corpus, with the existing independent
  percentile bootstrap of `historylabel compare-summary` (`diffCI`: 2000
  resamples, fixed seed per comparison, each list resampled independently),
  95% CI. The fused method **beats** the input on a corpus when the lower bound
  is > 0; the input **beats** the fused method when the upper bound is < 0;
  otherwise **not distinguishable**. An input with no pairs on a corpus is not
  scored there.
- **Over the five corpora:** a fused method beats an input when it beats it on
  at least 3 corpora and loses on none; the input beats the fused method under
  the same rule the other way round; anything else is not distinguishable,
  reported without reinterpretation.
- **Fusion "beats either input"** only for a fused method that beats doppel
  **and** beats its clone-detector input by that rule. Three fused methods are
  scored with no multiple-comparison correction, as in the earlier studies; a
  single fused method beating doppel is reported as that one result, not as
  "fusion beats doppel".
- **Secondary, if available.** A sibling study (`examples/outcome-reanalysis.md`)
  is adding paired / clustered bootstrap options to `compare-summary`. If that
  flag is on `master` before these lists are scored, the same two contrasts are
  also reported under it, as **secondary**. It cannot change the verdict. If it
  has not merged, it is not reported.
- Everything else — K = 50, ranks 101-500, M2, E, C, clustered M1, overlap,
  provenance, the doppel-minus-input table `compare-summary` also prints — is
  descriptive and cannot change the verdict.

### Validity checks, stated in advance

- **Inputs reproduce the clone-outcome study.** Each of the four input lists
  must equal, pair for pair and in order, the list of the same name in the
  committed `examples/clone-outcomes/<corpus>.clone-outcomes.json`. The four
  inputs' rate rows must equal their rows in `examples/clone-outcomes/summary.md`
  (same pairs, same judge). A mismatch stops the study.
- **Unresolved units** reported per corpus, as before.
- **Determinism.** Two runs of the same corpus produce byte-identical rankings
  and outcomes (checked on cobra and gin, as before).
- **Earlier studies still reproduce.** `TestCloneRankingsAt` is not changed.
  `compare-summary` over the committed clone-outcome and ranker-outcome files,
  with each study's own flags, must reproduce their committed `summary.md`
  byte for byte.
- **The fusion function** is unit-tested on a hand-computed example, on each
  tie-break level, and on determinism.

### Expectations

1. **Fusion will not beat doppel.** Where dupl is sparse (cobra, gin, hugo at
   t=100: 3, 16 and 1 pairs), the fused top 100 is doppel's top 100 with those
   few pairs interleaved near the top, so its M1@100 is within a few
   thousandths of doppel's. All three fused methods will be **not
   distinguishable** from doppel.
2. **moby is where fusion can gain.** Fusing with dupl will pull the
   `client.*Prune` pairs into the top 100, so the point estimate of
   `RRF(doppel, dupl …)` on moby will be above doppel's (0.040) and near or
   above dupl's (0.090-0.105). With a CI that wide it will not separate.
3. **Fusion against the clone input looks like doppel against it.** Not
   distinguishable against dupl at either threshold, and against floored token
   clones, mirroring `examples/clone-outcomes.md`: at most 2 wins, no losses.
4. **Provenance.** Where the clone list has at least 100 pairs (prometheus,
   moby, and token clones everywhere), the fused top 100 will hold 50-80 of
   doppel's top 100; with a sparse dupl list it will hold at least 80.
5. **Power is low.** The likely answer is "not shown" for every contrast, which
   would mean these outcomes cannot tell fusion from its inputs — not that
   fusion is no better.

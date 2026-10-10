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

## Results

Everything below was produced after the pre-registration above was committed
(`06dcc5a`), and nothing above this heading was edited afterwards. Reproduce
with `task fusion-outcomes CORPUS=<name>` per corpus, then
`task fusion-outcomes-summary`. The generated tables are in
[fusion-outcomes/summary.md](fusion-outcomes/summary.md), and the per-pair rows
in `fusion-outcomes/<corpus>.fusion-outcomes.json`.

### Validity checks

- **Input lists: pass.** On all five corpora each of the four input lists is
  the clone-outcome study's list of the same name, pair for pair and in order
  (the first table of the summary).
- **Input rate rows: fail on hugo and moby.** cobra, gin and prometheus
  reproduce `clone-outcomes/summary.md` row for row. On hugo, doppel's M1@100
  reads 0.0550 here against 0.0600 there (clustered 0.0333 against 0.0383),
  and token clones' M1 at ranks 101-500 0.0150 against 0.0163. On moby,
  doppel's M1 at ranks 101-500 reads 0.0397 against 0.0372. The pairs are the
  same; their judgments are not. Of the pairs both files judged, 0 differ on
  cobra and gin, 4 on prometheus, 6 on hugo and 2 on moby. All but one differ
  only in an edit count (the M1 denominator); one moby pair
  (`loggertest.Reader.TestFollow` / `testTail`) gains a co-change.
- **Unresolved units: 0** on every corpus.
- **Determinism: pass.** Two runs of cobra and of gin wrote byte-identical
  rankings and outcomes.
- **Earlier studies: pass.** `TestCloneRankingsAt` is unchanged, and
  `compare-summary` with each earlier study's flags reproduces
  `ranker-outcomes/summary.md` and `clone-outcomes/summary.md` byte for byte.
- **Fusion function: pass.** `TestRRFFuse` pins a hand-computed example, both
  tie-break levels, the cut, and determinism.

**Why the rate rows moved (diagnosed after the fact).** `historylabel compare`
judges each pair against a walk of the directories that the listed functions
live in, and a commit counts as a sweep — excluded from edits and co-changes —
when it modifies more than 10 functions *in the walked directories*
(`history.go`, `c.sweep = modified > sweepFuncs`). A different set of lists
walks a different set of directories: hugo's window has 512 walked commits here
against 764 in the clone-outcome run, moby's 1594 against 1810. So a commit
that touched 6 functions here and 6 elsewhere is a sweep in one run and an
ordinary edit in the other, and a pair's outcome depends on which other pairs
were judged beside it. This is a property of `compare` shared by every outcome
study, not of the fusion code. It was not anticipated in any earlier
pre-registration, and it was not fixed here, since the task was to judge with
`compare` unchanged.

### Verdict, by the pre-registered rule

**No verdict.** The pre-registration says a mismatch in the input rate rows
stops the study, and it does on hugo and moby. The fusion comparison below is
therefore **descriptive only**, even though it was computed exactly as
pre-registered:

| fused | input | fused wins | input wins | would-be verdict |
| --- | --- | ---: | ---: | --- |
| RRF(doppel, dupl (t=100, default)) | doppel | 0 | 0 | not distinguishable |
| RRF(doppel, dupl (t=100, default)) | dupl (t=100, default) | 2 | 0 | not distinguishable |
| RRF(doppel, dupl (t=50)) | doppel | 0 | 0 | not distinguishable |
| RRF(doppel, dupl (t=50)) | dupl (t=50) | 1 | 0 | not distinguishable |
| RRF(doppel, token clones (≥50 nodes)) | doppel | 0 | 0 | not distinguishable |
| RRF(doppel, token clones (≥50 nodes)) | token clones (≥50 nodes) | 2 | 0 | not distinguishable |

Fused minus doppel, M1@100, by corpus (cobra, gin, prometheus, hugo, moby):
dupl t=100 `+0.000, +0.000, −0.018, +0.000, +0.050`; dupl t=50 `+0.000,
+0.000, −0.014, −0.010, +0.060`; token clones `+0.002, −0.010, −0.033, −0.010,
+0.050`. No interval excludes zero.

Read plainly: **fusion does not beat doppel on any corpus, and its point
estimate is below doppel's on prometheus and hugo.** It gains only on moby,
where it takes in dupl's `client.*Prune` pairs. Nothing here would have been a
win for fusion under the rule even if the study had not stopped, and with
contrasts this far from separating, the judge's context dependence (it moved
M1@100 by 0.005 here) could not have produced one. That is an observation about
these numbers, not a replacement verdict.

The secondary paired/clustered bootstrap is not reported: that option of
`compare-summary` had not reached `master` when these lists were scored.

Against the expectations stated in advance:

1. **Fusion did not beat doppel**, as expected. Where dupl is sparse (cobra,
   gin and hugo at t=100), the fused M1@100 equals doppel's exactly.
2. **moby is where fusion gains**, as expected: 0.090 (t=100) and 0.100 (t=50)
   against doppel's 0.040, level with dupl's 0.090 and 0.105, and not
   separated. Clustered M1 is 0.020 and 0.030, against doppel's 0.020: the gain
   is the one `client.*Prune` commit, as it was for dupl alone.
3. **Fusion against its clone input looks like doppel against it**, as
   expected: 2, 1 and 2 wins, no losses.
4. **Provenance, mostly as expected.** With a sparse dupl list the fused top
   100 holds 82-100 of doppel's top 100 (cobra 100/98, gin 85/82, hugo t=100
   100). With a clone list of 100 or more it holds 63-78 on prometheus and
   moby, and 57 and 61 for token clones on cobra and hugo, but 48 on gin,
   below the expected 50.
5. **Power is low**, as expected: every contrast is not distinguishable.

### What the numbers show (descriptive, post-hoc)

- **RRF at k = 60 mostly interleaves.** Pairs both inputs hold rise to the top,
  then the fused list alternates between the two inputs. On prometheus and
  moby fused with token clones, all 100 of the fused top 100 are in both
  inputs' 500-cut lists, so fusion reorders agreed pairs rather than adding new
  ones.
- **Fusion pays for what it adds with what it drops.** On gin, fusing with
  token clones replaces half of doppel's top 100 with floored token pairs and
  loses part of the `Engine.Run*` family (M1 0.065 → 0.055, clustered 0.019 →
  0.009). On hugo, fusing with dupl t=50 drops one of doppel's nine pairs with
  an event (0.055 → 0.045). On moby it gains the `Prune` sweep. The two
  reaches the clone-outcome study described do not add up: each input's
  distinctive pairs displace the other's.
- **The judge's context dependence can move a corpus call.** In this file's
  own doppel-minus-dupl (t=50) row, hugo's lower bound is −0.0002, where the
  clone-outcome study's was positive (a ✓). Same lists, same pairs, a different
  set of neighbours in the walk. Any later study that compares numbers across
  separately judged files should judge them together, or fix the sweep rule to
  count functions corpus-wide.

### What this means

- **Fusing doppel with a clone detector is not shown to help.** It does not
  beat doppel anywhere, and on two of five corpora its point estimate is lower.
  Its only gain is moby's one sweeping commit, which dupl alone already had.
- The study itself is **not valid as pre-registered**: an input check failed,
  for a reason in the shared judge rather than in the lists. The finding worth
  carrying forward is that one: `historylabel compare` outcomes depend on what
  else is judged in the same run.

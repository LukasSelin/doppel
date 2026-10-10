# Re-analysing the two outcome studies with paired and clustered bootstraps

A re-analysis, not a new study: no corpus is ranked again, no history is walked
again, and nothing in doppel's ranking, scoring or defaults is touched. It reads
the committed outcome files of two earlier studies unchanged:

- `examples/ranker-outcomes/<corpus>.outcomes.json` (`examples/ranker-outcomes.md`);
- `examples/clone-outcomes/<corpus>.clone-outcomes.json` (`examples/clone-outcomes.md`).

Both studies decided each comparison with a 95% percentile bootstrap CI on
`M1@100(doppel) − M1@100(X)`, resampling **each list independently**. That
bootstrap has two flaws, both known before this file was written:

1. **Shared pairs.** doppel's top 100 and a baseline's top 100 often hold the
   same pairs (52–66 of them on prometheus and 43–50 on moby for the floored
   detectors in the clone study). An
   independent resample draws a shared pair on one side and not on the other,
   so a pair that cannot contribute to the difference contributes to its
   variance. The CI is wider than the data warrant.
2. **Clustered events.** One commit can co-change many pairs at once: gin's
   `93ff771` over the six `Engine.Run*` pairs, moby's `7faaa3a` over the six
   `client.*Prune` pairs. Resampling pairs as if they were independent treats
   one maintainer action as six observations. The CI is narrower than the data
   warrant.

The two flaws push in opposite directions. This file asks how much each
matters, and whether either changes a verdict.

**The original verdicts stand as recorded.** Each study pre-registered the
independent bootstrap, and its result sections report what that rule decided.
This is a separately pre-registered sensitivity analysis, reported alongside
them and never in place of them. Neither earlier file is edited.

## Pre-registration

Written and committed **before** any number in this file was computed. Nothing
in this section may change after results exist. Results go in sections below it.

### Inputs

- The ten committed outcome files listed above, read as they are at this
  commit. No new runs.
- The same lists, depths and pair sets as the original summaries: every
  method `compare-summary` scores there (pooled random as its three lists
  concatenated), each cut at K = 100 exactly as `methodView.slice(1, 100)`
  cuts it. A list shorter than 100 (dupl lists hold 1–100 pairs) is scored
  over the pairs it has, as before. A list with no pairs is not scored on that
  corpus, as before.

### Statistic

Unchanged: `D = M1(A) − M1(B)` with A = doppel and B = the baseline, where
`M1(L)` is the mean over L's top-100 entries of `cochanges / min(editsA,
editsB)` (0 when a side was never edited). The point estimate is therefore
the one the original summaries print, digit for digit. Only the interval
changes.

Written over distinct pairs: let U be the distinct pairs in the union of the
two top-100 lists, and `a_i`, `b_i` the number of times pair i appears in A's
and in B's list (0 or 1, except pooled random, whose three lists may repeat a
pair). Then `M1(A) = Σ a_i·m1_i / Σ a_i` and likewise for B.

### Resampling schemes

Both schemes resample **clusters** of distinct pairs from U, and compute each
replicate's `D` as the ratio of sums above over the drawn clusters, each drawn
cluster counted once per draw. They differ only in what a cluster is.

- **paired**: every pair in U is its own cluster. A shared pair is drawn for
  both lists at once, so it moves both means together.
- **paired-clustered**: clusters are the connected components of the graph on
  U in which two pairs are joined when their recorded evidence shares a
  commit. "Recorded evidence" is every evidence entry the outcome file keeps
  on the pair — co-change, lagged-sync, unpropagated-fix, extracted,
  consolidated — and every commit SHA listed on it. A pair with no evidence is
  a singleton. Components are computed over U for that one comparison, not
  over the whole file. With no shared commit anywhere, this scheme is exactly
  the paired one, draw for draw.

**Stratified by list membership.** Each cluster is put in one of three strata:
holds only pairs of A's list; holds only pairs of B's list; holds at least one
pair of each (which includes every shared pair). Each replicate resamples, with
replacement, as many clusters from each stratum as that stratum holds, strata
in that order (A-only, B-only, both), clusters within a stratum ordered by the
smallest pair index (into the file's `pairs`) they contain. Stratifying keeps
the composition of the two lists — how much is shared, how much is not — fixed
across replicates, which is what the independent bootstrap's fixed list sizes
did. It also guarantees that neither side is ever empty in a replicate, so no
replicate is undefined and none is discarded.

Unequal list lengths need no special handling: the denominators `Σ a_i` and
`Σ b_i` are each side's own count of drawn entries. A one-pair dupl list is a
mean over one pair, as it was in the original summary.

Two consequences follow by construction and are tested:

- identical lists give every replicate `D = 0`, hence a CI of exactly
  `[0, 0]` under both schemes;
- the same inputs give the same CI on every run.

### Constants

- **2000 resamples** per comparison, as before (`bootReps`).
- **Seed**: the existing 64-bit LCG (`study.go`), seeded per comparison with
  `seedOf("outcome-reanalysis bootstrap", <scheme>, <corpus>, <baseline>)`.
  A fresh seed rather than the original one, because the draws are of
  different objects.
- **Percentiles**: the original's nearest-rank indices over the sorted 2000
  replicates, `⌊0.025·1999⌋` and `⌈0.975·1999⌉`.

### Decision rule

Unchanged from both studies: per baseline per corpus, doppel **wins** when the
CI lower bound is > 0 and **loses** when the upper bound is < 0; over the five
corpora, doppel **beats** X with at least 3 wins and no loss, X **beats**
doppel with at least 3 losses and no win, anything else is **not
distinguishable**. Applied separately under each scheme, to each study.

### What is reported

For each study and each scheme: the primary comparison table in the original
format, the verdict per baseline beside the original verdict, and a
descriptive table of the union size, shared pairs, cluster count and largest
cluster per comparison. Plus a plain statement of every verdict that differs
from the original, in either direction.

### Validity checks, stated in advance

- `compare-summary` without the new flag reproduces both committed
  `summary.md` files byte for byte, with the Taskfile's flags
  (`-cost-dir` for the ranker study, `-reference … -overlap` for the clone
  study).
- Every point estimate under both schemes equals the original's.

### Known limitation, stated in advance

Pairs that share a *function* rather than a commit are also dependent — they
share an edit count in the denominator of M1 — and neither scheme clusters
them. Clustering on shared functions would join most families of siblings into
one component and leave very few clusters on the small corpora. This analysis
is about the two flaws above; the shared-function dependence is named here and
left alone.

### Expectations

1. **paired narrows the CIs**, most where overlap is high: prometheus and moby
   in the clone study, and the mass-like baselines (retrieval mass, call mass,
   overlap, the struct-min filtered doppel list) in the ranker study, whose
   lists share much of doppel's.
2. **paired-clustered widens them again** relative to paired, most on gin
   (`93ff771`) and moby (`7faaa3a`), where one commit carries a large part of
   a list's co-changes. On cobra and hugo, where events are sparse and spread
   over different commits, it will change little.
3. **Verdicts.** Under paired, a few "not distinguishable" verdicts may become
   decided, in whichever direction the point estimates already lean. Under
   paired-clustered, a win that rests on gin may be lost, and with it a
   3-of-5 verdict that needed gin. The most likely outcome is that few or no
   verdicts change under either scheme, because most original intervals were
   far from zero or straddled it widely.

## Results

Everything below was computed after the pre-registration above was committed
(`b72d52d`), and nothing above this heading was edited afterwards. Reproduce
with `task outcome-reanalysis`. The generated summaries are in
`outcome-reanalysis/`, one per study and scheme:
`ranker-outcomes.paired.md`, `ranker-outcomes.paired-clustered.md`,
`clone-outcomes.paired.md` and `clone-outcomes.paired-clustered.md`. Each is the
study's own summary with the primary table's intervals replaced and a
resampling-frame table appended. Everything above the primary table is
identical to the study's committed `summary.md`.

### Validity checks

- `compare-summary` without `-bootstrap` reproduces
  `ranker-outcomes/summary.md` (with `-cost-dir`) and
  `clone-outcomes/summary.md` (with `-reference … -overlap`) byte for byte.
- Every point estimate under both schemes equals the original's, in all 75
  cells of the two primary tables.
- The percentile step is now one helper shared with the cost study's
  `summarize`; `examples/cost-study/summary.md` and `samples.md` still
  reproduce byte for byte.

### Verdicts

Wins and losses are doppel's, over the five corpora. A verdict that differs
from the original is in bold.

**Ranker-outcome study** (`examples/ranker-outcomes.md`):

| baseline | original (independent) | paired | paired-clustered |
| --- | --- | --- | --- |
| doppel (struct-min filtered) | not distinguishable (0–0) | not distinguishable (0–0) | not distinguishable (0–0) |
| token clones | doppel beats it (4–0) | doppel beats it (4–0) | **not distinguishable (1–0)** |
| code-shape | doppel beats it (4–0) | doppel beats it (4–0) | **not distinguishable (1–0)** |
| retrieval mass | not distinguishable (0–0) | not distinguishable (0–0) | not distinguishable (0–0) |
| overlap | not distinguishable (1–0) | **doppel beats it (3–0)** | not distinguishable (0–0) |
| name heuristic | doppel beats it (4–0) | doppel beats it (4–0) | **not distinguishable (1–0)** |
| call mass | not distinguishable (1–0) | not distinguishable (1–0) | not distinguishable (1–0) |
| size | not distinguishable (2–0) | not distinguishable (2–0) | not distinguishable (1–0) |
| random (3 seeds) | doppel beats it (3–0) | doppel beats it (3–0) | **not distinguishable (1–0)** |

**Clone-outcome study** (`examples/clone-outcomes.md`):

| baseline | original (independent) | paired | paired-clustered |
| --- | --- | --- | --- |
| dupl (t=100, default) | not distinguishable (2–0) | **doppel beats it (3–0)** | not distinguishable (1–0) |
| dupl (t=50) | not distinguishable (2–0) | not distinguishable (2–1) | not distinguishable (0–0) |
| token clones (≥100 nodes) | doppel beats it (3–0) | doppel beats it (3–0) | **not distinguishable (1–0)** |
| token clones (≥50 nodes) | not distinguishable (2–0) | **doppel beats it (3–0)** | not distinguishable (2–0) |
| token clones (no floor) | doppel beats it (4–0) | doppel beats it (4–0) | **not distinguishable (2–0)** |
| code-shape (≥50 nodes) | not distinguishable (1–0) | not distinguishable (1–1) | not distinguishable (1–0) |

**Under the paired-clustered bootstrap, every "doppel beats it" verdict in
both studies becomes "not distinguishable".** That is six verdicts lost: token
clones, code-shape, the name heuristic and random in the ranker study; token
clones at 100 nodes and with no floor in the clone study. Nothing beats doppel
under any scheme, and the clustered scheme records no loss anywhere.

Under the paired bootstrap, three "not distinguishable" verdicts become "doppel
beats it" (overlap; dupl at t=100; token clones at 50 nodes), none is lost, and
**doppel records its first two losses**: on moby, against dupl at t=50
(−0.065 [−0.120, −0.015]) and against floored code-shape (−0.050 [−0.100,
−0.010]). Neither changes a verdict, because both baselines also lose to
doppel elsewhere.

The original verdicts stand as recorded. This analysis says they rest on an
independence assumption the data do not support.

### Why: the events are a handful of maintainer actions

Linking the pairs in doppel's own top 100 that carry M1 > 0 by shared evidence
commits:

| corpus | pairs with M1 > 0 | independent clusters | largest cluster's share of doppel's M1 |
| --- | ---: | ---: | ---: |
| cobra | 1 | 1 | 1.00 |
| gin | 7 | 2 | 0.85 (6 pairs, `93ff771`) |
| prometheus | 27 | 26 | 0.09 |
| hugo | 6 | 3 | 0.60 (4 pairs) |
| moby | 4 | 2 | 0.75 (3 pairs) |

On gin, hugo and moby, doppel's whole M1@100 is two or three maintainer
actions. A resample of 100 clusters misses all of them with probability about
`e^−2` ≈ 14% (gin, moby) or `e^−3` ≈ 5% (hugo), well above the 2.5% the lower
bound reads. So against a baseline whose list scores 0, the lower bound is
**exactly 0** on those corpora — not a win under the strict `lo > 0` rule.
That is what the clustered intervals show: a lower bound of exactly
`+0.0000` on gin in every comparison of both studies, and on hugo and moby
against the ranker study's token clones, code-shape and name heuristic. Where
the baseline has events of its own (hugo in the clone study), the bound can
clear zero, and four hugo cells do. Prometheus, with 26 independent clusters,
is the corpus where most differences are decided: in the ranker study it
decides for doppel against every baseline the original rule said doppel beat;
in the clone study, against token clones with no floor and at 50 nodes, not
at 100. One corpus is not three.

### Against the expectations stated in advance

1. **paired narrows the CIs: yes.** Most where overlap is high, as expected:
   prometheus in the clone study (dupl at t=50 from about ±0.10 to ±0.045), and the
   mass-like lists in the ranker study (retrieval mass on prometheus from
   [−0.072, +0.132] to [−0.036, +0.095]; the struct-min filtered list
   collapses to [0, 0] wherever it equals doppel's top 100). It also narrowed
   the moby intervals enough to show two losses, which was not foreseen.
2. **paired-clustered widens them again: yes, on gin and moby as expected,
   and on hugo, which was not.** gin's upper bounds go from about +0.115 to
   +0.168 and its lower bounds to 0; moby's from about [−0.10, 0] to
   [−0.16, +0.03] against the floored detectors. "Little change on hugo" was
   wrong: four of doppel's six hugo event pairs share commits. cobra changed
   little, as expected.
3. **Verdicts: the expectation was wrong.** "Few or no verdicts change" held
   only for the paired scheme, and even there three verdicts moved. Under the
   clustered scheme every "beats" verdict was lost, not only those needing gin.

### Deviations from the pre-registration

- The pre-registration says the clustered scheme with no shared commit "is
  exactly the paired one, draw for draw". The frame is identical (tested by
  `TestClusteredWithoutSharedCommitsIsPaired`), but the seed includes the
  scheme name, as the same pre-registration also specifies, so the draws are
  not. The two schemes' cobra cells in the ranker study (where no two pairs
  share a commit) differ in the last digit for that reason alone: seed noise,
  not a difference between the schemes. The seed rule was followed; the
  "draw for draw" sentence was not literally true of it.
- Nothing else deviated.

### What this means

- **The original studies' wins are not robust to clustered outcomes.** Under
  their own pre-registered rule they stand. Under a bootstrap that counts one
  commit once, no win survives on three of five corpora, because there are
  only two or three independent events behind doppel's list there.
- **The paired scheme is not a rescue.** It removes the noise of shared pairs
  and adds three wins, but it keeps each event pair independent, which is the
  larger flaw on these data. Its extra wins rest on the same few commits.
- **No baseline beats doppel under any scheme.** The clustered result is
  "these outcomes cannot separate the methods", not "a baseline is better".
- **The next study needs more independent events, not more pairs.** On
  gin, hugo and moby, adding origins T or corpora adds clusters; deepening K
  mostly adds pairs that share the same commits. A decision rule for future
  outcome studies should resample commit clusters from the start, and should
  pre-register what to do when a corpus has fewer independent events than the
  rule can resolve (here, with three or fewer clusters the lower bound
  against a zero-scoring baseline is expected to sit at exactly zero).

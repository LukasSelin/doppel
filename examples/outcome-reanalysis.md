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

# Do the outcome comparisons hold at earlier origins?

A measurement, not a scoring change: nothing in doppel's ranking, scoring or
defaults is touched to produce it. It re-runs the two outcome comparisons
before it — `examples/ranker-outcomes.md` (doppel against simple rankers over
its own retrieval union) and `examples/clone-outcomes.md` (doppel against clone
detectors that search the tree) — at more than one revision per corpus.

Every outcome study so far has had one T per corpus, the cost study's. Most
verdicts came out **not distinguishable**, and several per-corpus results rest
on one or two commits (gin's on one gosec commit, moby's dupl lead on one
`Prune*` commit). Low power is the main limit. Rolling-origin evaluation
(Tashman 2000) evaluates a forecaster at several origins, each judged only on
what happened after it. Here a forecaster is a ranking taken at T_k, and what
happened after is the window T_k..T_{k−1}. Several origins per corpus multiply
the outcome data without changing what one observation means.

## Pre-registration

Written and committed **before** any ranking at a new origin was taken and
before any new window was judged. Nothing in this section may change after
results exist. Results and the verdict go in sections below it.

### Origins

- **Window length.** Each corpus keeps the window length the cost study chose
  for it, as a nominal length L = `years × 365 days`, where `years` is the step
  the cost study's rule settled on (`scripts/cost-study.sh`): 4 for cobra, 2
  for gin, prometheus, hugo and moby.
- **Choice of T_k.** With `pin_ts` the committer timestamp of the pinned
  commit, T_k is the latest first-parent commit at or before `pin_ts − k·L`:
  `git rev-list --first-parent -1 --before=<pin_ts − k·L> <pin>`, the cost
  study's command with `k·years` in place of `years`. T_0 is the pin. T_1 is
  then the cost study's T by construction, and the script asserts it equals the
  `since` in `examples/cost-study/<corpus>.cost.json`.
- **Windows.** Window k is T_k..T_{k−1}: the commits reachable from T_{k−1} and
  not from T_k. The ranking is taken on the tree at T_k and judged on window k
  only, with T_{k−1} as the window's end (what `historylabel compare` calls
  the pin). Consecutive windows share no commit, so no outcome is counted at
  two origins. A pair of functions can appear at several origins; its
  outcomes there come from disjoint windows.
- **Which origins.** k = 1 (the existing T, already judged), k = 2 and k = 3:
  two earlier windows per corpus. The count is fixed here, not chosen after
  looking at history depth. An inadmissible origin is reported and not
  replaced by a deeper one.
- **Admissibility.** An origin k ≥ 2 is admissible when all three hold, every
  one decided before its window is judged:
  1. window k holds at least 100 non-merge commits touching `*.go`
     (`git rev-list --count --no-merges T_k..T_{k−1} -- '*.go'`, the cost
     study's count);
  2. `TestRankingsAt` and `TestCloneRankingsAt` both complete on the tree at
     T_k;
  3. doppel's list at T_k has at least 100 pairs, so M1@100 is a mean over the
     full depth for the method every comparison is about.

  A declined calibration is allowed (the run uses the static fallbacks, as
  doppel would) and reported. An inadmissible origin is not judged at all.

### Methods

Unchanged at every origin, by the same code:

- the ranker-outcome set, `TestRankingsAt`: doppel, doppel (struct-min
  filtered), token clones, code-shape, retrieval mass, overlap, name heuristic,
  call mass, size, random (seeds 1-3, pooled to 300 pairs), all over doppel's
  retrieval union at T_k's calibrated threshold;
- the clone-outcome set, `TestCloneRankingsAt` with dupl (v1.1.0) run on the
  tree at T_k: doppel, dupl (t=100, default), dupl (t=50), token clones
  (≥100 / ≥50 nodes / no floor), code-shape (≥50 nodes), each searching the
  whole population.

Population, tie-breaks, the cut at 500 and the judging (`historylabel
compare`, unchanged: walk, judge, exclusions) are the earlier studies'. That
is 15 baselines, each compared with doppel.

### Units

A **unit** is one (corpus, admissible origin). The k = 1 units are the
committed `examples/ranker-outcomes/<corpus>.outcomes.json` and
`examples/clone-outcomes/<corpus>.clone-outcomes.json`, unchanged. A baseline
with no pairs at a unit is not scored at that unit; a list shorter than 100 is
scored over the pairs it has, as before.

### Metric

**M1@100**, as before: mean over a list's top 100 of `cochanges /
min(editsA, editsB)`, 0 when a side was never edited. Per unit,
`d = M1@100(doppel) − M1@100(X)`.

### Pooled decision rule

For each baseline X, the pooled difference is

    D(X) = mean over corpora c of [ mean over c's scored units of d ]

— every corpus with at least one scored unit counts once, and its origins are
averaged inside it. A 95% percentile bootstrap CI comes from 2000 replicates,
stratified by unit: in each replicate, every unit's doppel list and X list are
resampled with replacement independently (the earlier studies' resampling,
unit by unit), D is recomputed, and nothing moves between units. The seed is
fixed (FNV-1a over the study name and X).

- doppel **beats** X when the CI's lower bound is > 0;
- X **beats** doppel when the upper bound is < 0;
- otherwise **not distinguishable**, reported without reinterpretation.

No correction for 15 comparisons, as in the earlier studies; the verdicts are
read one baseline at a time.

**Why this rule and not a count of unit wins.** The per-corpus rule ("wins on
at least 3 corpora, loses none") was built for five units. Extended to unit
counts ("wins at least half the admissible units, loses none") it throws away
exactly what more origins add: each unit's own CI is about as wide as before,
so most units stay individually not distinguishable however many there are,
and one noisy loss vetoes everything. A stratified bootstrap of the mean pools
the evidence while keeping each unit's pairs inside that unit. Corpora rather
than units are weighted equally because origins of one corpus share code,
maintainers and habits: three moby origins are not three independent
projects, and letting a deep history outvote a short one would count one
project thrice.

**What it does not fix.** Pairs inside a unit are resampled as if
independent, but one commit can co-change several pairs (gin's gosec commit),
so the CI is too narrow where events cluster. The mean is dominated in size by
corpora whose M1 is large (prometheus). Both are reported below the verdict,
not corrected inside it.

**The per-corpus verdicts at the original T are untouched.** They stay as
`examples/ranker-outcomes.md` and `examples/clone-outcomes.md` report them.

### Descriptive (cannot change the verdict)

- the per-unit table: d with each unit's own 95% CI (the earlier per-corpus
  bootstrap, seeded per unit);
- the count rule above, for contrast;
- **replication**: the pooled D and CI over the k ≥ 2 units only — whether the
  original T's results reappear on windows the earlier studies never saw;
- **leave one corpus out**: D and CI with each corpus removed in turn;
- **clustered M1**: D with each co-change commit's credit divided among the
  pairs of the list it touches, without a CI;
- M2, E, C and event counts per unit and method.

### Secondary analysis

A sibling study (`examples/outcome-reanalysis.md`) is adding paired and
paired-clustered bootstrap options to `historylabel compare-summary`. If that
has merged into this branch's base before the pooled verdict is computed, its
options are run over the same units and reported as a secondary analysis. They
cannot change the primary verdict. If it has not merged, this is stated and
nothing is substituted for it.

### Validity checks, stated in advance

- **The original T is reproduced.** At k = 1 the new origin path (explicit
  since/pin) must write rankings whose lists are identical, pair for pair and
  in order, to those in the committed outcome files, for both method sets on
  all five corpora. The judged outcome files themselves must be byte-identical
  to the committed ones on cobra and gin, where a replay is cheap. A mismatch
  means the pipeline or the T tree moved, and the study stops.
- **T_1 equals the cost study's `since`** on every corpus.
- **Unresolved units** are reported per unit; a pair with an unresolved side is
  dropped from every list and counted, as before.
- **doppel's list is the same in both method sets** at every origin.
- **Determinism.** At least one new origin (cobra or gin, k = 2) is ranked and
  judged twice, and the files must be byte-identical.
- **Shared code.** If code shared with the earlier studies changes,
  `compare-summary` must still reproduce `ranker-outcomes/summary.md` and
  `clone-outcomes/summary.md` byte for byte, and `historylabel study` the
  committed cobra cost-study file.

### Expectations

1. **Earlier windows are denser for the old, busy corpora.** moby and
   prometheus were more active before their current T; cobra's k = 3 window
   starts when the project was weeks old and may be inadmissible.
2. **The similarity rankers without a floor stay near zero** at every origin,
   so doppel beats token clones, code-shape, name heuristic and random pooled,
   as at the original T.
3. **Retrieval mass stays level with doppel.** If pooling separates them, the
   pooled CI will be narrow around a small difference; the most likely
   verdict is still not distinguishable.
4. **Against dupl and the floored detectors, pooling decides little.** The
   original T had point estimates on both sides; with more units the pooled
   difference will sit near zero. A verdict either way would be the new
   information this study exists for, and one against doppel is reported as
   plainly as one for it.
5. **Compute.** Each new origin costs one bench ranking per method set (dupl
   and the all-pairs search included) and two history replays. moby's replays
   are the slowest; the budget is one working day of wall clock with corpora
   run in parallel. An admissible origin that has not finished by then is
   reported as not run, never silently dropped.

## Results

Everything below was produced after the pre-registration above was committed
(`633bef8`; written as `d623e84` and rebased unchanged onto `master` when #78
merged), and nothing above this heading was edited afterwards. Reproduce with
`task rolling-origin CORPUS=<name>` per corpus, then
`task rolling-origin-summary`. The generated tables are in
[rolling-origin/summary.md](rolling-origin/summary.md). The judged files of the
new origins are `rolling-origin/<corpus>.o<k>.outcomes.json` and
`.clone-outcomes.json`, and each corpus's origins are in
`rolling-origin/<corpus>.origins.tsv`. The k = 1 units are the committed files
of the two earlier studies.

### Origins

| corpus | window | k = 2 | k = 3 |
| --- | --- | --- | --- |
| cobra | 4 years | 2017-12-04, 153 Go commits | 2013-11-05, 348 Go commits |
| gin | 2 years | 2022-02-14, **80 Go commits: inadmissible** | 2020-02-26, 140 Go commits |
| prometheus | 2 years | 2022-08-18, 1488 | 2020-08-18, 842 |
| hugo | 2 years | 2022-08-12, 850 | 2020-08-12, 725 |
| moby | 2 years | 2021-11-03, 2670 | 2019-11-07, 2487 |

Nine new units, fourteen in all. Every new origin except gin k = 2 was
admissible. cobra k = 3 is the project two months old (48 functions). Its
calibration declined (406 null pairs, 1 000 needed), so it ran at the static
defaults, and doppel's list there is the whole 133-pair union. That clears the
100-pair bar, so it is a unit, as the rule says.

### Validity checks

- **The original T is reproduced.** At k = 1 the explicit-window path wrote
  outcome files **byte-identical** to the committed ones, for both method sets
  on all five corpora. That is stronger than the pre-registered check, which
  asked for byte identity on cobra and gin and identical lists elsewhere.
- **T_1 equals the cost study's `since`** on all five corpora (the script
  asserts it).
- **Unresolved units: 0** at every unit, in both files.
- **doppel's list is identical in both method sets** at every unit.
- **Determinism.** cobra k = 2 was ranked and judged twice; rankings and
  outcomes were byte-identical.
- **Shared code.** `compare-summary` still reproduces
  `ranker-outcomes/summary.md` and `clone-outcomes/summary.md` byte for byte.
  The only change to shared code is a `-min-doppel` flag on `compare`, off by
  default; `study` is untouched.

### Pooled verdict, by the pre-registered rule

| baseline | units | D [95% CI] | verdict | at T alone (earlier study) |
| --- | ---: | --- | --- | --- |
| doppel (struct-min filtered) | 14 | +0.0010 [−0.0148, +0.0170] | not distinguishable | not distinguishable |
| token clones | 14 | +0.0551 [+0.0426, +0.0674] | **doppel beats it** | doppel beats it |
| code-shape | 14 | +0.0575 [+0.0453, +0.0706] | **doppel beats it** | doppel beats it |
| retrieval mass | 14 | −0.0029 [−0.0186, +0.0143] | not distinguishable | not distinguishable |
| overlap | 14 | +0.0240 [+0.0084, +0.0386] | **doppel beats it** | not distinguishable |
| name heuristic | 14 | +0.0495 [+0.0367, +0.0626] | **doppel beats it** | doppel beats it |
| call mass | 14 | +0.0142 [−0.0013, +0.0294] | not distinguishable | not distinguishable |
| size | 14 | +0.0457 [+0.0327, +0.0594] | **doppel beats it** | not distinguishable |
| random (3 seeds) | 14 | +0.0592 [+0.0478, +0.0707] | **doppel beats it** | doppel beats it |
| dupl (t=100, default) | 13 | −0.0255 [−0.0764, +0.0186] | not distinguishable | not distinguishable |
| dupl (t=50) | 14 | −0.0455 [−0.0617, −0.0288] | **dupl beats doppel** | not distinguishable |
| token clones (≥100 nodes) | 14 | +0.0285 [+0.0145, +0.0430] | **doppel beats it** | doppel beats it |
| token clones (≥50 nodes) | 14 | +0.0168 [+0.0014, +0.0327] | **doppel beats it** | not distinguishable |
| token clones (no floor) | 14 | +0.0598 [+0.0468, +0.0726] | **doppel beats it** | doppel beats it |
| code-shape (≥50 nodes) | 14 | +0.0126 [−0.0034, +0.0297] | not distinguishable | not distinguishable |

D is M1@100(doppel) − M1@100(X), averaged over a corpus's units and then over
the five corpora. The right-hand column is the earlier studies' per-corpus rule
at the original T, which stands unchanged.

**By the rule as written, dupl at t=50 beats doppel.** That verdict stands as
computed. What it rests on is one pair at one unit. These are facts about the
result, not reinterpretations of it:

- At cobra k = 3, dupl t=50 lists **one** function pair,
  `*Command.Flags ↔ *Command.PersistentFlags`, which one commit (`b655df6`)
  later co-changed: M1@100 = 1.000 over a list of one. doppel ranks the same
  pair **first**; its M1@100 over that pair and the 99 below it is 0.029. So
  d = −0.971 at that unit, cobra's corpus mean is −0.309, and that alone moves
  the pooled D by −0.062. A list of one resamples only to itself, so its
  bootstrap has no spread.
- **Leave cobra out and the sign reverses**: D = +0.0203 [+0.0006, +0.0398],
  doppel ahead. With any other corpus left out, dupl t=50 stays ahead, because
  cobra k = 3 stays in.
- The pre-registration let a short list be scored over the pairs it has, and
  weighted corpora equally. Together those let a one-pair list on a 48-function
  tree decide a pooled verdict. That is a flaw in the rule that the data
  exposed. It is reported here, not repaired: a rule changed after seeing this
  would no longer be pre-registered.

dupl at its default threshold has the same shape at hugo k = 2 and k = 3 (4
and 3 pairs, M1 0.250 and 0.500). It ends not distinguishable only because
those lists have more than one pair to resample.

### Did more data change the answers?

- **Against the simple rankers over doppel's union: slightly, and only
  towards doppel.** overlap and size, not distinguishable at T, are beaten
  pooled. Retrieval mass and call mass stay level. Retrieval mass's D is
  −0.003 with a CI narrow around zero, so pooling did not hide a difference; it
  measured its absence. The factors the key multiplies onto mass still add
  nothing measurable on M1@100.
- **Against a configured clone detector: no.** Not shown before, not shown
  now. doppel beats floored token Jaccard at both floors, but at 50 nodes the
  margin is small and is gone when gin, hugo or prometheus is left out, and in
  the replication table. Floored code-shape and dupl at its default are not
  distinguishable. dupl at t=50 beats doppel by the rule, on the single pair
  above. Without cobra, doppel is ahead of it, with a lower bound of 0.0006.
- **Apart from that one pair, the added data does not go against doppel.** No
  other unit has a CI entirely below zero for any baseline.

### What the new windows show (descriptive)

- **Replication on the k ≥ 2 units alone** gives the pooled table's verdicts
  for token clones, code-shape, name heuristic, size, random, token clones
  ≥100 and no floor (doppel ahead), and for retrieval mass, call mass,
  struct-min, dupl t=100 and code-shape ≥50 (level). overlap and token clones
  ≥50 fall back to not distinguishable. dupl t=50 still beats doppel there,
  for the cobra k = 3 reason.
- **prometheus carries the signal at every origin.** doppel's M1@100 there is
  0.216, 0.146 and 0.147, with 47, 27 and 20 distinct commits behind its
  events. Every other unit sits between 0.010 and 0.065, with 1 to 10 commits
  behind it. Leaving prometheus out halves most pooled margins and removes
  the overlap and token ≥50 wins.
- **At earlier origins dupl at t=50 is not sparse on the large trees**: 50
  to 92 pairs on prometheus and hugo. It is behind doppel on both new
  prometheus origins (0.118 and 0.095 against 0.146 and 0.147) and within
  0.012 of it on hugo.
- **Clustered M1** (each co-change commit's credit divided among the pairs it
  touches) keeps every sign of the pooled table. It shrinks most of doppel's
  margins, by up to a third, and widens both dupl leads.

### Caveats that apply to every number above

- **Commit clustering.** The bootstrap resamples pairs, and one commit can
  co-change several pairs in a list. The sibling reanalysis (#79, paired and
  paired-clustered bootstrap for `compare-summary`) had **not merged** when
  this was scored, so the pre-registered secondary analysis was not run, and
  nothing was substituted for it. Its reported finding on the earlier studies:
  under paired-clustered resampling every "doppel beats it" becomes not
  distinguishable, because gin, hugo and moby rest on two or three independent
  commit clusters. The new origins add clusters rather than depth, but most
  units still have fewer than ten distinct commits behind doppel's events. Read
  the "beats" verdicts above as an upper bound on what a clustered analysis
  would grant.
- **The sweep exclusion depends on what else is judged.** `history.go` marks
  a commit a sweep by how many functions it modifies *in the directories
  walked*, and those are the union over every listed pair. So a pair's outcome
  can differ between two `compare` runs with different lists. The "window
  commits replayed" column of the summary differs between the two files of
  most units for this reason. This study never compares a pair across runs:
  each baseline is compared with the doppel list judged in its own file. A fix
  is pending outside this branch. When it lands, every file here needs
  re-judging, which `scripts/rolling-origin.sh -c <corpus> -k "<k>"` does per
  corpus and origin.
- **One look before the new data.** While `rolling-summary` was being tested
  on the committed k = 1 files only, its pooled table over those five units
  was read before any new origin was summarised. Those files were already
  public, the rule was already committed, and nothing in the design moved.

### Deviations from the pre-registration

- None in the design. The k = 1 reproduction was run in full on all five
  corpora instead of as a list comparison on three.
- The summary tool first labelled a unit by its rank among the judged
  origins, which called gin's k = 3 "o2". It now reads k from the origin table.
  No pooled number changed, because the pooled seeds do not depend on k.

### Compute

All nine new origins plus the five k = 1 reproductions, both method sets,
corpora in parallel on 24 threads, took about 8 minutes of wall clock (moby
7m48s, prometheus 5m28s, hugo 4m00s, cobra 1m29s, gin 0m35s). The
one-working-day budget was nowhere near reached.

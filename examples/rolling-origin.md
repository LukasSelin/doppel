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

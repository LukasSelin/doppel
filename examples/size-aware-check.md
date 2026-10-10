# Can K-size make doppel worse?

A safety check, not a scoring change: no production default moves.
`examples/size-aware-rank.md` found one size-aware rank key, **K-size**
(`analyzer.RankKey × Breakdown.Containment × min(nodes_A, nodes_B)`), beating
production doppel on held-out maintenance outcomes, +0.010 [+0.004, +0.018]
on M1@100 over nine rolling-origin windows, with the gain concentrated in
prometheus k = 2 and moby k = 2. It failed that study's cobra label guardrail
by one rank swap (merge mean 4.8 → 5.0). Before it is proposed for production,
this study asks where it could make doppel worse. It does not re-ask whether
K-size helps.

The main risk is not measured yet. On the private labelled corpora,
`TestLensRank` and `TestOverlapRank` found most of the labelled top 20 on two
of three corpora to be false positives, overwhelmingly same-receiver mirror
and sibling methods (Get/Set/Delete, Encode/Decode, Read/Write, Min/Max). A
size reward multiplies the key by how much code two bodies share. That
discounts small accessor pairs, but it could lift **large** mirror and sibling
pairs, whose bodies are long and alike in skeleton. The two ladder rungs the
outcome studies never covered, chi and conc, are also unmeasured.

## Pre-registration

Written and committed **before** any of the three checks below was scored.
Nothing in this section may change after results exist. Results and the
verdict go in sections below it.

### What was seen before this was written

- `examples/size-aware-rank.md` and its two result tables, in full, and
  `internal/bench/size_rank_test.go`.
- The private labels directory's README and its per-file class counts: four
  label sets (three private corpora and doppel's own source), each private
  corpus holding false positives, refactors and at least one merge, and
  doppel's own set a single label. No labelled pair was read, and no ranking of a private corpus
  under any key was taken for this study.
- The origin listing `scripts/rolling-origin.sh -l` prints for chi and conc
  (commit counts only, no ranking, no judgment). chi: the cost study's rule
  steps back to 4 years, T_1 = `b6a2c5a90` (2022-08-12), and the window to the
  pin holds **92** non-merge Go commits; the next origin, k = 2 (2018-08-16 to
  2022-08-12), holds 146; there is no k = 3. conc: **no commit exists** 2, 3
  or 4 years before its pin (the repository starts in 2023), so it has no
  origin at all.
- The committed `examples/{chi,conc,cobra,gin}.md` reports (top 10 each), which
  are public and were read before, in earlier sessions.

### The key under test, and the secondary rows

`K-size` is `sizeVariants[0]` in `internal/bench/size_rank_test.go` at
`a3d53c4`, unchanged. `K-subtree` and `K-floor` (`sizeVariants[1]`, `[2]`)
are reported as secondary rows everywhere and decide nothing. Every check
ranks through `analyzer.SortForReportBy`, the seam production calls with its
own key. The verdict reads K-size against production only.

### Check 1: the private labels

- **Harness.** A new bench test, `TestSizeCheckLabels` (guard
  `DOPPEL_BENCH_SIZECHECK=1`, labels from `DOPPEL_BENCH_LENSES_LABELS` and
  `DOPPEL_BENCH_LENSES_EXTRA`, exactly as `TestOverlapRank`). Per labelled
  corpus it loads the labels' population, runs `Analyze` at retrieval defaults
  (the run `TestOverlapRank`, `TestLensRank` and `TestSizeVariantsGolden`
  score), and scores the labels with `ScoreBy` under production's
  `RankKey` and under each size variant's key (`sizeVariant.memo`, reused, not
  copied). `ScoreBy` ranks the whole union with `--max-per-func 2`, as
  `TestGoldenCorpora` does.
- **Corpora.** The four label sets in the private labels directory: three
  private corpora (A, B, C here) and doppel's own source. cobra's committed
  labels run through the same test for completeness. The pooled criteria read
  the four private sets; the per-corpus merge criterion reads every set with a
  present merge, cobra included.
- **Reported per corpus and pooled:** violations (merges never retrieved +
  false positives above the worst merge + false positives in the top 20, the
  sum `violations()` already defines), false positives in the top 20, false
  positives above the worst merge, and the merge, refactor and false-positive
  mean ranks. Pooled counts are sums; pooled means are the mean of per-corpus
  means, as `TestOverlapRank` pools them.
- **The mirror check.** For every labelled false positive whose rank rises
  under K-size (a smaller rank number), its rank under both keys, its node
  counts, and whether it is a same-receiver mirror or sibling method. That
  list stays out of the repository (privacy, below); the committed results
  carry only counts.

**Criterion 1, all three parts required:**

- **1a.** Pooled violations under K-size ≤ under production.
- **1b.** Pooled false positives in the top 20 under K-size ≤ under production.
- **1c.** On every labelled corpus with a present merge, K-size's merge mean
  is at most 1.0 rank worse than production's.

These are the suggested criteria unchanged. False positives above the worst
merge are reported but do not decide: they are already a part of 1a.

### Check 2: chi and conc maintenance outcomes

- **Units.** chi and conc at the rolling-origin origins k = 1, 2, 3, where
  T_1 is the cost study's rule (latest first-parent commit at or before the
  pin − 2 years, stepping back a year at a time up to 4 while the window holds
  fewer than 100 non-merge Go commits) and T_k is k such steps back, as
  `scripts/rolling-origin.sh` computes them. That script reads T_1 from a
  committed cost study; chi and conc have none, so it is extended to derive
  T_1 by the same rule from the ladder pin when no cost study exists, and to
  report a missing origin rather than fail. With a cost study present it is
  unchanged.
- **Including k = 2 and 3 is a change from the brief** ("choose T by the cost
  study's rule"), and the reason is the listing above: by that rule chi's only
  T has a 92-commit window, inadmissible under rolling-origin's ≥ 100 rule.
  The earlier origins are rolling-origin's own design, and on chi and conc no
  key was developed on any window, so none of them is in-sample.
- **Admissibility** is rolling-origin's, unchanged and applied by the same
  script: ≥ 100 non-merge Go commits in the window, and doppel's list ≥ 100
  pairs at T (`MIN_DOPPEL=100`, exit 3). An inadmissible unit is reported with
  its reason and not judged.
- **Method set and judge** are size-aware-rank's test phase exactly:
  `scripts/rolling-origin.sh -s size` (which calls `ranker-outcomes.sh -s
  size`): doppel, the six detector lists and the three variants, every list of
  a unit in one `historylabel compare -sweep-scope commit` run. Scored with
  `historylabel size-summary`: matched-depth M1@100, the paired-clustered
  bootstrap (2000 replicates, seeded as #82 seeded it), the floor of 3
  independent event clusters per comparison frame.

**Criterion 2:** no admissible unit that clears the event-cluster floor has a
K-size − doppel 95% CI whose upper bound is below 0. A unit below the floor is
reported and not judged, as #82's test phase treated one. If no unit is
admissible, or none clears the floor, criterion 2 **passes vacuously**, and
the result says so in the verdict as a coverage gap rather than as evidence.

### Check 3: label-free top-20 inspection

- **Corpora and trees.** chi, conc, cobra and gin at their ladder pins, the
  trees `task examples` analyses.
- **Pool.** What the report ranks: a new bench test, `TestSizeCheckTop20`
  (guard `DOPPEL_BENCH_SIZECHECK_TOP20=<corpus,...>`), loads the default
  population (tests excluded, every language, as `doppel analyze` reads it),
  calibrates at rate 0.01 and keeps the union at or above the calibrated
  struct-min (`prepareBaselineRun`, unchanged). It ranks that pool with
  `SortForReportBy`, `--max-per-func 2`, top 20, under production's key and
  under K-size's.
- **Validity check.** Production's top 10 equals the committed
  `examples/<corpus>.md` top 10 pair for pair and in order. A mismatch is
  reported as a deviation, and the bench list is what is compared.
- **Entering and leaving.** A pair *enters* when it is in K-size's top 20 and
  not production's, and *leaves* the other way round.
- **Classification, blind to direction.** The test writes the entering and
  leaving pairs of all four corpora as one list sorted by corpus and name,
  with file:line for both sides and no mark of which list a pair came from,
  and writes the directions to a separate file. Each pair's two bodies are
  read and classified before that file is opened. Classes:
  - **merge-worthy**: one function could replace both, with at most a
    parameter or a trivial difference (bodies the same up to names and
    literals).
  - **refactor**: a substantial shared part could be extracted into one
    helper, but the two functions differ in meaningful logic.
  - **mirror or sibling**: two operations of one type or API, inverse or
    parallel verbs (Get/Set, Encode/Decode, Read/Write, Min/Max), or two
    implementations of one interface method doing different work, whose shared
    shape is the API's skeleton rather than duplicated logic.
  - **other false positive**: the shared shape is a language idiom or a
    coincidence.

**Criterion 3:** summed over the four corpora, the number of entering pairs
classified mirror/sibling or other false positive is at most the number of
leaving pairs with those classes.

### Verdict rule

K-size is **safe to propose** only if criteria 1a, 1b, 1c, 2 and 3 all pass.
Otherwise the verdict names each criterion that failed, with the numbers. A
vacuous pass of criterion 2 is reported as one.

### Expectations, stated in advance

1. **Check 1 is the real test, and I expect it to be close.** K-size
   multiplies production by containment × the smaller node count. Labelled
   merges are mostly exact or near-exact clones (containment near 1), so they
   should keep or improve their rank unless they are small, and small-bodied
   merges will drop. Small accessor false positives (Get/Set one-liners) should
   drop. Large Encode/Decode-style mirror pairs, long and alike in skeleton,
   should rise. Which effect dominates in the top 20 is not predictable from
   what I know; I put roughly even odds on 1b failing on at least one private
   corpus and on the pooled count, and expect 1c to hold except where a
   corpus's merges are a few small clones.
2. **Check 2 will most likely say nothing.** chi k = 1 is inadmissible on
   commit count (92), and conc has no origin. chi k = 2 has 146 commits; on a
   2018 chi tree doppel's list may fall under 100 pairs, and if it does not,
   the unit will most likely fall below the event-cluster floor, as cobra and
   gin units mostly did in #82. So criterion 2 most likely passes vacuously,
   and the coverage gap for small corpora remains.
3. **Check 3 will move few pairs.** #82 found K-size's top 100 sharing 94-100
   pairs with production's on the small corpora. At the top 20, with the
   diversity cap, I expect 0-4 entering pairs per corpus. On cobra I expect
   `GenMarkdownCustom`/`GenReSTCustom` (a refactor) to hold its place near the
   top and small exact clones to slide. I expect criterion 3 to pass, mainly
   because entering pairs on these corpora are larger, partly alike pairs that
   read as refactors.
4. If K-size does make doppel worse, it will be on the private corpora, by
   lifting large mirror pairs into the top 20, and not on the public ladder.

### Privacy

No name, path, pair, function or distinctive statistic of the three private
corpora or of the private labels directory is committed, in this file, in a
commit message, in the PR or in `CLAUDE.md`. Committed private results are
aggregates, and per-corpus rows appear only as "private corpus A/B/C" counts
that name nothing.

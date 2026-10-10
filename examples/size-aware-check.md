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

## Results

Everything below was produced after the pre-registration above was committed
(`db08150`, pushed before any check was scored), and nothing above this
heading was edited afterwards. Reproduce with `task size-aware-check-labels
LABELS=<dir> EXTRA='<roots>'` (check 1; its log names private pairs),
`task size-aware-check-outcomes` (check 2) and `task size-aware-check-top20`
(check 3). The public outputs are in [size-aware-check/](size-aware-check/):
`chi.md` and `chi.o2.size-outcomes.json` (check 2), the origin tables, and for
check 3 `top20.txt` (both top 20s per corpus, all three variants),
`blind.tsv`, `top20-classes.tsv` (the classification, written before
`directions.tsv` was opened) and `directions.tsv`.

### Verdict, by the pre-registered rule

| criterion | production | K-size | result |
| --- | --- | --- | --- |
| 1a pooled violations, four private label sets | 45 | 47 | **fail** |
| 1b pooled false positives in the top 20 | 20 | 20 | pass |
| 1c merge mean at most 1.0 rank worse, every corpus with a merge | — | +1.0, +18.0, +21.0 on private A, B, C; +0.2 on cobra | **fail** (B, C) |
| 2 no judged chi/conc unit with CI upper bound < 0 | — | chi k = 2: +0.000 [+0.000, +0.000], 2 event clusters, below the floor | pass, **vacuously** |
| 3 entering FP-class pairs ≤ leaving, four ladder rungs | — | 1 entering against 6 leaving | pass |

**K-size is not safe to propose.** Criteria 1a and 1c fail. Criterion 2
passes only because no unit could judge it. Criterion 3 passes.

### Check 1: the private labels

Production against K-size, per label set. Means are given as K-size's change,
so no private corpus's rank distribution is published.

| label set | violations | FP in top 20 | FP above worst merge | merge mean | refactor mean | FP mean |
| --- | --- | --- | --- | --- | --- | --- |
| private A | 24 → 22 | 13 → 12 | 11 → 10 | +1.0 | +0.8 | +0.7 |
| private B | 9 → 14 | 5 → 7 | 4 → 7 | +18.0 | +12.6 | −1.4 |
| private C | 11 → 10 | 1 → 0 | 10 → 10 | +21.0 | +20.1 | +91.0 |
| doppel's own (one label) | 1 → 1 | 1 → 1 | 0 → 0 | — | — | +2.0 |
| **pooled** | **45 → 47** | **20 → 20** | **25 → 27** | **+13.3** | **+11.1** | **+23.1** |
| cobra (public, not pooled) | 0 → 0 | 0 → 0 | 0 → 0 | +0.2 | −0.3 | −1.0 |

The secondary rows, pooled over the same four sets: K-subtree violations 40,
false positives in the top 20 17, above the worst merge 23; K-floor 40, 18 and
22. Both are below production on all three counts. On the per-corpus merge
criterion K-subtree is +9.0 on B and +12.2 on C, and K-floor +6.2 on C (−3.0
on B, −1.5 on A), so both would fail 1c too.

**The mirror check.** Eleven labelled false positives across the private sets
rank higher under K-size than under production. Read by hand: **four are
inverse mirror pairs** (encode/decode, read/write twice, one conversion each
way), **five are sibling methods or parallel implementations** of one API, and
two are other false positives (two separate programs' operation tables, and a
read path's internal opener against its public entry point). Eight of the
eleven sit in the top 20 under K-size:

- **The predicted failure happened.** On private A a large encode/decode
  method pair (a few hundred nodes a side) rises from 3 to **1**, displacing a
  labelled merge, and a large read/write pair rises 16 → 8. On private B a
  large read/write pair rises 8 → 6.
- **The largest single move is not a mirror.** On private B two separate
  programs' operation tables, over a thousand nodes each, rise from 22 to
  **2**.
- **What leaves is small.** The two false positives K-size pushes out of the
  top 20 are short sibling pairs (under 80 nodes a side). Inside the top 20 the
  same holds: small accessor and arithmetic siblings drop (B's top pair, a
  min/max pair of about 60 nodes, falls 1 → 13).

So the number of false positives in the top 20 does not move (criterion 1b),
but their **size and rank** do: K-size swaps small sibling false positives for
large mirror and multi-program pairs, and puts one at rank 1 and one at rank 2.

**The merges that drop are mid-sized exact clones.** On B, the corpus's one
merge (about 50 nodes a side) falls 14 → 32; on C, a merge of about 140 nodes
falls 4 → 13 and one of about 25 nodes 54 → 139. Containment × the smaller
node count rewards size, so a large, partly alike pair (refactor or false
positive) outranks a mid-sized exact clone. That is the cobra mechanism
(`GenMarkdownCustom`/`GenReSTCustom` above `MarkFlags*`): 0.2 ranks of merge
mean there, 18 and 21 on two private corpora.

### Check 2: chi and conc maintenance outcomes

| unit | window | Go commits | admissible | result |
| --- | --- | ---: | --- | --- |
| chi k = 1 | 2022-08-12 .. 2026-08-20 (`b6a2c5a90`, the cost study's rule at 4 years) | 92 | **no**, under 100 | — |
| chi k = 2 | 2018-08-16 .. 2022-08-12 (`44932d207`) | 146 (132 replayed) | yes; doppel lists 500 pairs of a 512-pair union | K-size − doppel **+0.000 [+0.000, +0.000]**, 2 event clusters, below the floor of 3: reported, not judged |
| chi k = 3 | none (no commit 12 years before the pin) | 0 | no | — |
| conc k = 1, 2, 3 | none: the repository starts in 2023, under 2 years before its pin | 0 | **no** | — |

At chi k = 2 production and all three variants score M1@100 0.015 (2 pairs
with an event, 6 commits). K-size's top 100 shares 95 pairs with production's
and drops 5 of production's 95 small pairs, none with an event. **Criterion 2
passes vacuously: no chi or conc unit can measure a difference.** Small
corpora remain outside what the outcome studies can say, as cobra and gin
mostly were in #82. Full tables in [size-aware-check/chi.md](size-aware-check/chi.md).

### Check 3: label-free top-20 inspection

Validity: production's top 10 equals the committed `examples/<corpus>.md` top
10, pair for pair and in order, on all four rungs.

| corpus | entering (K-size) | leaving (production) |
| --- | --- | --- |
| chi | `ClientIPFromHeader`/`ClientIPFromXFFTrustedProxies` (mirror/sibling), `RedirectSlashes`/`URLFormat` (refactor) | `URLParam`/`URLParamFromCtx` (merge-worthy), `compressResponseWriter.Push`/`.Close` (mirror/sibling) |
| conc | none | none |
| cobra | `AddCommand`/`RemoveCommand` (refactor), `genMan`/`GenMarkdownCustom` and `genMan`/`GenReSTCustom` (refactor) | `SetUsageTemplate`/`SetHelpTemplate`/`SetVersionTemplate`, three pairs (mirror/sibling) |
| gin | `LoadHTMLGlob`/`LoadHTMLFS` (refactor), `ProtoBuf.Render`/`TOML.Render` (merge-worthy) | `Context.Done`/`.Err`, `StaticFile`/`StaticFileFS` (mirror/sibling) |
| **sum** | 7: 1 FP-class, 5 refactor, 1 merge-worthy | 7: 6 FP-class, 1 merge-worthy |

The notes are in `top20-classes.tsv`. **On public code K-size does the
opposite of what it does on the private corpora**: what enters is larger and
mostly refactor material, and what leaves is short accessor and forwarding
siblings. It costs one merge-worthy pair on chi (`URLParam`/`URLParamFromCtx`,
22 and 19 nodes, rank 15 → out) and gains one on gin. No large mirror pair is
in these four trees' top 20s under either key, which is why the public check
could not catch the private failure. The secondary variants move 1-2 pairs per
corpus (`top20.txt`).

### Against the expectations

1. **Check 1 was the real test, as expected**, and two of its three parts
   failed. 1b held, which I had put at even odds. 1c failed on two corpora,
   worse than expected: the merges that dropped were not all small (one is
   about 140 nodes), so "small merges drop" understated it. The large mirror
   pairs rose as predicted.
2. **Check 2 said nothing, as expected**, but at chi k = 2 for another reason
   than the one I thought likelier: doppel's list cleared 100 pairs and the
   unit was admissible, then fell below the event-cluster floor.
3. **Check 3 moved 0-3 pairs per corpus**, inside the 0-4 expected, and passed
   as expected. `GenMarkdownCustom`/`GenReSTCustom` rose to rank 1 on cobra.
4. **K-size made doppel worse on the private corpora and not on the public
   ladder**, as expected.

### Deviations from the pre-registration

- None in the design. One addition: check 3's harness also logs the two
  secondary variants' top 20s.
- doppel's own label set is scored against the working tree of the main
  checkout at the time of the run, so its numbers are not reproducible from a
  commit.

### What this means

**K-size can make doppel worse, in one place and by a measurable amount.** On
the private labelled corpora it lifts large mirror and multi-program pairs to
the top of the report (ranks 1 and 2 on two corpora) while the number of false
positives in the top 20 stays the same, and it pushes mid-sized exact merges
down, worsening the merge mean by 18 and 21 ranks on two of three corpora. Pooled violations rise 45 → 47. On
the public ladder it does the opposite, trading short sibling pairs for
refactor-sized ones, and on held-out maintenance outcomes it was already
measured better (#82). The two findings share a mechanism: a size reward ranks
large, partly alike pairs above mid-sized exact ones. Maintenance outcomes
reward that, because such pairs do get edited together. Reviewers do not,
because they label large mirror pairs false positives.

What would have to change before a proposal: a size term that does not reward
a large pair for being a mirror (the `mirror operations` kind detects them but
by design never ranks, see *Pair kinds*), or a size term bounded above so that
it cannot carry a partly alike pair past an exact clone. Neither was measured
here.

# Does a size-aware rank close doppel's gap to clone detectors?

A measurement, not a scoring change: no production default moves. doppel's
rank key, `ShapePower`, `TrophicPower` and every other default are untouched.
The variants live in `internal/bench` (`size_rank_test.go`) behind a seam,
`analyzer.SortForReportBy`, which production calls only with its own key.
Whether any variant is worth proposing for production is the user's decision
afterwards.

## Why

`examples/clone-outcomes.md` and `examples/rolling-origin.md` found doppel
**not distinguishable** from configured clone detectors (dupl, size-floored
token and code-shape search) on maintenance outcomes, and pooled over origins
dupl at t=50 nominally ahead. The orchestrating session then collected the 24
pairs, at the original T, that were in a detector's top 100, carried a later
maintenance event, and were not in doppel's top 100. 19 of the 24 were
already in doppel's retrieval union, at ranks 101-420: moby's four
`client.*Prune` methods (six pairs) at 121-126; prometheus's histogram
iterators, kubernetes `NewEndpoints`/`NewPod`, aws `UnmarshalYAML`,
`parseLVals` and the `scrapeCache` setters at 107-419. A few were beyond 500
(moby `containerSpecFromGRPC`/`containerToGRPC`, 118 and 132 lines). **So the
gap is ranking, not recall.**

Hypothesised causes:

1. Detectors rank by the absolute amount of duplicated code. doppel's key
   squares two similarity *ratios* (code-shape², trophic²), so a large pair
   that is partly alike loses to a small pair that is wholly alike, while
   maintenance cost scales with how much code is duplicated.
2. Detectors match fragments inside otherwise different functions; doppel
   compares whole functions. Containment exists but never ranks.
3. Detectors report clone groups. A minor effect, not chased here.

The strength to protect: doppel reaches small families below any clone floor.
gin's 10-16-line `Engine.Run*` family (one commit, M1 0.065 on gin at T) is its
main edge, and a size term may cost it.

## Phase 0: a pair-local judge

`historylabel compare` marked a commit a **sweep** when it modified more than
10 functions *in the directories walked*, and the walked directories are the
union over every pair judged in the run. So a pair's outcome depended on what
else was judged beside it (`examples/fusion-outcomes.md` measured 9 of 684 hugo
pairs judged differently between two studies). Two other exclusions had the
same dependency: the per-commit **family** count (`family`: how many functions
a commit changed the same way, read over the commit's edits in walked
directories) and the extraction **adopter** count, and the history-wide
**campaign** tally (`deltaFuncs`, counted over the edits of walked commits
only).

`historylabel compare -sweep-scope commit` (`history.go`, `scopeCommit`)
replays every non-merge commit in the window that touches a non-test `.go`
file, and each commit's edits, sweep count, family and adopter counts and its
contribution to the campaign tally are taken over its **whole Go diff**, every
directory. Nothing a pair's judgment reads then depends on the other pairs.
The default, `walked`, is the old judge, unchanged; a file judged under
`commit` records `"sweepScope": "commit"`, and a default file records nothing,
so its bytes are unchanged.

Verified:

- **The default reproduces every committed file.** `compare` under the default
  re-judged all ten original-T files of the ranker- and clone-outcome studies
  (five corpora each, via `scripts/ranker-outcomes.sh`): byte-identical to the
  committed `examples/ranker-outcomes/*.outcomes.json` and
  `examples/clone-outcomes/*.clone-outcomes.json`. `compare-summary`
  reproduces the ranker, clone and fusion summaries and all four
  outcome-reanalysis summaries byte for byte, and `rolling-summary`
  reproduces `examples/rolling-origin/summary.md` (both after the refactor
  that exposes the paired bootstrap's replicates, `pairedReps`).
- **Pair-locality, as a test.** `TestCommitScopeIsPairLocal` builds a
  repository where one commit modifies the pair and nine functions in another
  directory, and another makes a campaign. Under `walked` the pair reads 3
  co-changes judged alone and 1 judged beside a pair in the other directory;
  under `commit` it reads 1 both ways, field for field.
- **Pair-locality, on real data.** On all five corpora at T, the development
  run that also listed the whole retrieval union (2 518 to 37 827 distinct
  pairs) and the final development run without it (1 675 to 2 104) judged
  every shared pair identically: 0 of 9 049 differ.

Commit scope is stricter about what counts as a sweep: a commit is now a
sweep when it touches more than ten functions anywhere, including vendored
code and generated files outside the population, and a sweep's edits count
neither as co-changes nor as edits. So numbers judged under it differ from the
earlier studies' files (at T, doppel's M1@100 on hugo is 0.050 here against
0.060 there) and are **not comparable** with them. Every number in this
study is judged under it, all lists of a unit in one run.

## Development: the five original-T windows

Everything in this section used only the five original-T windows (the cost
study's T..pin per corpus), whose outcomes earlier studies had already
published.

**Harness.** `TestSizeRankingsAt` (guard `DOPPEL_BENCH_SIZERANK_AT`, driven by
`scripts/ranker-outcomes.sh -s size`) lists the clone-outcome study's seven
methods (`cloneOutcomeLists`, reproduced) beside the variants, and `compare
-sweep-scope commit` judges them. With `SIZE_DEV=1` it also lists the whole
retrieval union and writes each listed pair's rank-key factors and size
features, so any candidate key over the union can be scored offline from one
judging run per corpus. That shortcut is legitimate only because the judge is
pair-local now.

**Size features** (all in `size_rank_test.go`):

- *contained nodes*: `Breakdown.Containment × min(nodesA, nodesB)`: how many of
  the smaller body's nodes the larger also has, by the corpus-weighted WL
  containment. Both factors are on every production pair already.
- *covered nodes (≥ m)*: the nodes of b lying in maximal canonical subtrees of
  at least m nodes that a holds verbatim (hash-cons identity via
  `fingerprint.Cons`, matched as a multiset, top down), the smaller of the two
  directions. An exact, rename-proof clone length summed over every clone the
  two bodies share, fragments included (cause 2).
- *smaller nodes*: `min(nodesA, nodesB)`.

### Development log

Development M1@100 per corpus under commit scope (offline, over the union;
the three locked variants were then re-run through the full harness and
matched these numbers exactly). "prod" is doppel's production key. Mean is the
plain mean of the five.

| key | cobra | gin | prometheus | hugo | moby | mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| prod (shape² trophic²) | 0.010 | 0.065 | 0.216 | 0.050 | 0.040 | 0.076 |
| shape¹ | 0.010 | 0.065 | 0.221 | 0.055 | 0.050 | 0.080 |
| shape¹ × contained nodes (cause 1 as first stated) | 0.012 | 0.065 | 0.248 | 0.055 | 0.037 | 0.083 |
| **shape² × contained nodes (K-size)** | 0.012 | 0.065 | 0.268 | 0.065 | 0.057 | 0.093 |
| shape¹ × covered (≥10) | 0.010 | 0.055 | 0.259 | 0.065 | 0.063 | 0.091 |
| shape² × covered (≥10) | 0.010 | 0.055 | 0.251 | 0.065 | 0.113 | 0.099 |
| **shape² × (covered (≥10) + 10) (K-subtree)** | 0.010 | 0.065 | 0.251 | 0.065 | 0.113 | 0.101 |
| shape² × (covered (≥5) + 10) | 0.010 | 0.065 | 0.264 | 0.065 | 0.057 | 0.092 |
| shape² × (covered (≥20) + 10) | 0.010 | 0.065 | 0.219 | 0.055 | 0.093 | 0.089 |
| shape¹ × covered (≥20), no pseudo-count | 0.010 | 0.000 | 0.233 | 0.055 | 0.113 | 0.082 |
| shape² × √(covered (≥10)) | 0.010 | 0.055 | 0.251 | 0.055 | 0.100 | 0.094 |
| shape² × log(1 + covered (≥10)) | 0.010 | 0.055 | 0.228 | 0.050 | 0.060 | 0.081 |
| shape² × covered (≥10) / (covered + 200) | 0.010 | 0.055 | 0.251 | 0.065 | 0.113 | 0.099 |
| shape² × (contained + covered (≥10) + 10) | 0.010 | 0.065 | 0.262 | 0.065 | 0.057 | 0.092 |
| overlap × covered (≥10), no mass or shape | 0.010 | 0.055 | 0.272 | 0.027 | 0.073 | 0.087 |
| shape² × min(1, smaller nodes / 100) | 0.010 | 0.065 | 0.231 | 0.055 | 0.110 | 0.094 |
| shape² × min(1, (covered (≥10) + 10) / 100) | 0.010 | 0.065 | 0.238 | 0.055 | 0.110 | 0.096 |
| **shape² × min(1, (covered (≥10) + 10) / 50) (K-floor)** | 0.010 | 0.065 | 0.215 | 0.055 | 0.050 | 0.079 |

About a hundred further keys were scored the same way (saturating, additive and
floored forms of each feature at several constants, the largest shared
subtree, WL shared-label counts per round). None beat K-subtree's mean, and
the pattern above held throughout:

- **Keep code-shape squared.** Replacing shape² by shape × size (cause 1 as
  first stated) gains less than multiplying the production key by size. The
  ratio is not the problem; a size-blind key is.
- **A size reward reaches prometheus and moby.** On prometheus the entering
  pairs are the histogram iterators' `Next`/`AtFloatHistogram`, the
  kubernetes endpoint builders, `loadWAL`/`loadWBL`, `parseLVals` and
  `API.query`/`queryRange` (11 event pairs in, 1 out for K-size). On moby only
  the *covered* forms bring in the six `client.*Prune` pairs: covered nodes
  counts the identical statements those 19-line bodies share exactly, where
  containment discounts them by the corpus weighting.
- **gin's `Engine.Run*` family survives every additive or saturating form**
  (M1 0.065 throughout). A *hard* clone floor kills it: covered nodes without
  a pseudo-count, at m = 20, drops gin to 0.000, which is the clone
  detectors' own failure. The pseudo-count (+10) is what keeps a pair sharing
  no large subtree from ranking 0; it is not tuned (5, 10 and 20 score
  identically).
- **Moby's gain is one commit.** The six `Prune` pairs share `7faaa3a`, so
  under clustered resampling they are one observation. prometheus's gain is
  spread over many commits.

**Golden-label guardrails** (`TestSizeVariantsGolden`, the cobra hand labels
through `ScoreBy`, the scorer and cap `TestGoldenCorpora` uses; hard
assertions: merges retrieved, no false positive above the worst merge or in
the top 20):

| key | merge mean | refactor mean | FP mean | hard assertions |
| --- | ---: | ---: | ---: | --- |
| production | 4.8 | 14.9 | 52.5 | green |
| shape¹ | 5.3 | 12.8 | 50.5 | green |
| K-size | 5.0 | 14.6 | 51.5 | green |
| K-subtree | 5.8 | 18.0 | 62.5 | green |
| shape² × min(1, smaller nodes / 100) | 4.7 | 13.7 | 47.5 | green |
| shape² × min(1, (covered + 10) / 100) | 5.8 | 18.0 | 62.5 | green |
| K-floor | **4.3** | 18.1 | **62.5** | green |

Every size *reward* lifts one refactor pair, `GenMarkdownCustom`/`GenReSTCustom`
(318 and 370 nodes, shape 0.65), from rank 4 to rank 1, above the three
`MarkFlags*` merges (68-node exact clones, key 234 against its 228). That one
swap moves every merge down a rank and the merge mean from 4.8 to 5.0-5.8.
The labels and the outcomes disagree here by construction: a hand review says
a large, partly alike refactor pair ranks below small exact clones, and a
size reward says the opposite. A floor that discounts below `MarkFlags*`'
size (68 nodes, 78 with the pseudo-count) is the only family that keeps the
merges on top, and it cannot reach moby's `Prune` pairs, which are about the
same size: on development a covered-node floor admits them only between 80 and
90 nodes. K-floor, at 50
nodes, is the one candidate that held every guardrail.

**Through the full harness and the statistics pre-registered below**
([size-aware-rank/dev.md](size-aware-rank/dev.md), the five development
units), K-size reads +0.028 [+0.011, +0.045] against doppel, K-subtree +0.041
[+0.012, +0.078] and K-floor +0.005 [−0.001, +0.013]. cobra and gin fall below
the event-cluster floor there, so those pools rest on prometheus, hugo and
moby. These numbers are not evidence: the keys were chosen on these windows.

**Locked.** Three variants go forward: K-size (cause 1, production-cheap),
K-subtree (cause 2, the best development mean), K-floor (the one candidate
inside the guardrails).

## Pre-registration

Written and committed **before** any ranking at a test origin was taken and
before any test window was judged under this study's judge. Nothing in this
section may change after results exist. Results and the verdict go in sections
below it.

### What was seen of the test windows before this was written

The nine test units are the rolling-origin study's earlier windows, and its
results are public. Before writing this I had read `examples/rolling-origin.md`
in full: its pooled table, and the per-unit facts it quotes (doppel's M1@100 on
prometheus k = 2 and 3, 0.146 and 0.147; dupl t=50's one-pair list on cobra
k = 3; dupl t=100's 4 and 3 pairs on hugo k = 2 and 3; dupl t=50's 50-92 pairs
on the earlier prometheus and hugo origins). I had not opened
`examples/rolling-origin/summary.md` or any per-pair outcome file of those
windows (`*.o2.*`, `*.o3.*`). Two reproduction checks read them through the
tools only (`rolling-summary` rendered the committed summary and `cmp`
compared it, without the output being read). Those files were judged under the
walked scope; this study re-judges every unit under commit scope, so none of
their numbers is one of this study's.

### Variants, frozen

`key_prod` is `analyzer.RankKey` under `DefaultRankOptions`: retrieval
`Total × Score² × TrophicSim² × OverlapScore`, × `CallSim` for test pairs (none
here: tests are excluded). With `C` = `Breakdown.Containment`, `n` =
`Fingerprint.Nodes`, and `S10` the covered nodes at m = 10 (`sharedSubtree(a,
b, 10)`: maximal canonical subtrees of at least 10 nodes, hash-cons identity,
multiset-matched top down, the smaller of the two directions):

| variant | key |
| --- | --- |
| `variant: K-size` | `key_prod × C × min(n_A, n_B)` |
| `variant: K-subtree` | `key_prod × (S10 + 10)` |
| `variant: K-floor` | `key_prod × min(1, (S10 + 10) / 50)` |

Each ranks doppel's retrieval union at the calibrated threshold (the pool
doppel's own list ranks) with `analyzer.SortForReportBy`: key descending, then
`Score` descending, then `AIdx`, `BIdx`; no diversity cap, as doppel's list
has none; cut at 500. The code is `sizeVariants` in
`internal/bench/size_rank_test.go` at this commit.

### Methods compared

At every unit, in one rankings file and one judging run: `doppel` (production,
as every outcome study listed it), `dupl (t=100, default)`, `dupl (t=50)`,
`token clones (≥100 nodes)`, `token clones (≥50 nodes)`, `token clones (no
floor)`, `code-shape (≥50 nodes)` (the clone-outcome study's lists,
`cloneOutcomeLists`, unchanged), and the three variants.

### Units and judge

- **Test units.** The rolling-origin study's admissible earlier origins, nine:
  cobra k = 2 and 3, gin k = 3, prometheus k = 2 and 3, hugo k = 2 and 3, moby
  k = 2 and 3. Origins and windows exactly as `examples/rolling-origin/
  <corpus>.origins.tsv` gives them (window k = T_k..T_{k−1}); gin k = 2 stays
  inadmissible. The admissibility rules are rolling-origin's (≥ 100 Go commits;
  doppel's list ≥ 100 pairs), applied again by the same script.
- **Judge.** `historylabel compare -sweep-scope commit`, every list of a unit
  in one run (`scripts/rolling-origin.sh -s size`, which calls
  `ranker-outcomes.sh -s size`).

### Metric and interval

- **Primary metric: M1@100 at matched depth.** For a comparison of X with Y at
  a unit, both lists are cut at `min(nX, nY, 100)`, so a one-pair list is
  compared with the other list's top one, never its top hundred. `d =
  M1(X) − M1(Y)` at that depth.
- **Interval: the paired-clustered bootstrap** of `examples/outcome-
  reanalysis.md` (`pairedReps`, `compare_paired.go`): clusters are pairs joined
  by a shared evidence commit, stratified by list membership, 2000
  replicates, LCG seeded by `seedOf("size-aware-rank", corpus, k, X, Y,
  depth)`, nearest-rank 2.5% and 97.5% points.
- **Floor on independent events.** A unit counts in a comparison only if the
  comparison's frame (the union of the two cut lists) holds **at least 3
  independent event clusters**: clusters holding a pair with M1 > 0. Below
  three, the clustered lower bound against a zero-scoring list sits at exactly
  0 (outcome-reanalysis), so the unit is reported and not pooled.
- **Pooling.** `D = mean over corpora of (mean over that corpus's counting
  units of d)`: each corpus with at least one counting unit weighs once,
  however many origins it has. The CI is the percentile interval of the same
  combination taken replicate by replicate (replicate r of every unit, pooled
  the same way). A comparison with no counting unit is not scored.
- **Decision rule, per comparison.** X **beats** Y when the pooled lower bound
  is above 0; Y beats X when the upper bound is below 0; otherwise **not
  distinguishable**, reported without reinterpretation. No correction for
  multiple comparisons.

### Success criteria, all required for a variant to count as improving in the right direction

- **(a)** The variant beats doppel on the held-out pooled M1@100.
- **(b)** It closes the gap to the clone detectors. For each detector X of the
  five — dupl t=100, dupl t=50, token clones ≥ 100 and ≥ 50, code-shape ≥ 50 —
  the **gap change** is `M1(variant) − M1(doppel)` at X's matched depth
  (`min(nX, 100)`), pooled over every unit where X lists a pair, with no
  cluster floor. (b) holds when the gap change's pooled point estimate is
  above 0 for **at least 4 of the 5** detectors **and** against no detector is the variant's verdict worse
  than doppel's (beats > not distinguishable > beaten). Whether "doppel beats
  it" becomes reachable is read off the variant's own verdicts.
- **(c)** Golden-label guardrails: hard assertions green, merge mean not worse
  than production's 4.8, false-positive mean not lower than 52.5. These are
  measured on the cobra hand labels at the pin and do not depend on any
  window, so they are **already known**, above: K-size fails (merge 5.0, FP
  51.5), K-subtree fails (merge 5.8; FP 62.5 passes), K-floor passes (merge
  4.3, FP 62.5; refactor 18.1 against 14.9, which the criterion does not
  read).
- **(d)** The small-family cost is reported: per unit, the pairs in doppel's
  top 100 whose smaller side is under 23 lines at T (the median length of a
  100-node function, dupl's default floor, over the five development trees)
  and how many of them, and how many with an event, each variant drops.

**So only K-floor can meet all four.** For K-size and K-subtree the test still
decides (a), (b) and (d): whether a size term buys held-out outcomes at the
measured label cost, which is what a proposal would have to argue.

Each variant's verdict is reported as **improved** (beats doppel),
**regressed** (doppel beats it) or **not distinguishable**, and separately
whether the doppel-versus-detector gap closed by (b).

### Expectations, stated in advance

1. **Shrinkage.** The development means were read off the same windows the
   keys were chosen on, from about a hundred candidates, so the held-out effects
   will be smaller. K-subtree's development margin over production was 0.025,
   K-size's 0.017, K-floor's 0.003 (plain means).
2. **prometheus decides.** It holds most of the independent events at every
   origin, as before. The variants' gains there were spread over many commits
   in development, so they should replicate in sign. moby's development gain
   was one commit and may not.
3. **(a):** K-size and K-subtree have a fair chance of beating doppel pooled;
   K-floor will most likely be not distinguishable (it changes little).
4. **(b):** the gap change will be positive for most detectors for K-size and
   K-subtree, small for K-floor.
5. **cobra and gin will mostly fall below the event-cluster floor**, so the
   pooled numbers rest on prometheus, hugo and moby.
6. **(d):** the variants drop small pairs on the large corpora (development,
   on prometheus, hugo and moby: K-size 23-40 per unit, K-subtree 17-30,
   K-floor 6-15), almost none with an event (one, on prometheus).

### Validity checks, stated in advance

- **Production lists reproduce.** At every test unit, doppel's list and the six
  detector lists equal, pair for pair and in order, the lists of the same name
  in `examples/rolling-origin/<corpus>.o<k>.clone-outcomes.json`. A mismatch
  means the tree or the pipeline moved, and the study stops.
- **The committed rolling-origin files reproduce under the default judge.**
  `scripts/rolling-origin.sh` (both method sets, default scope) re-judges the
  nine units and its outputs are byte-identical to the committed ones.
- **Determinism.** cobra k = 2 is ranked and judged twice under this study's
  harness; rankings and outcomes are byte-identical.
- **Origins.** The origin tables the script writes equal the committed
  rolling-origin ones.
- **Unresolved units** are reported per unit.

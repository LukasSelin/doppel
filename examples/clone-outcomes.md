# Does doppel beat a clone detector that finds its own pairs?

A measurement, not a scoring change: nothing in doppel's ranking, scoring or
defaults is touched to produce it. It follows `examples/ranker-outcomes.md`,
which ranked each corpus at the cost study's revision T and judged each listed
pair on what happened over T..pin. That study found doppel ahead of token
clones, code-shape, a name heuristic and random on M1@100, and not
distinguishable from retrieval mass, call mass, overlap or size.

Three weaknesses of that study are the reason for this one:

1. **Every baseline re-ranked doppel's retrieval union.** None searched the
   tree. A real clone detector finds its own pairs, including pairs doppel's
   retrieval never proposed.
2. **The similarity baselines had no minimum size.** Their top 100 was mostly
   3-line accessors nobody edits. Every practical clone detector has a floor
   (dupl's default is 100 tokens). Part of doppel's win was against a
   handicapped baseline.
3. **dupl was never judged on outcomes.** `examples/baselines.md` scored it
   only on history labels taken at the pin, which leak (see that file's
   post-hoc note).

The question: **does doppel's top-ranked list find pairs that later cost
maintainers duplicated work better than a properly configured, off-the-shelf
clone detector that finds its own pairs?**

## Pre-registration

Written and committed **before** any list in this study was judged against any
outcome. Nothing in this section may change after results exist. Results and
the verdict go in sections below it.

### Design

- **Corpora, T and pin.** cobra, gin, prometheus, hugo, moby, with exactly the
  `since` and `pin` of `examples/cost-study/<corpus>.cost.json`. No new choice
  of T.
- **Population.** The cost study's: Go files only, test files and generated
  files excluded, the pipeline's directory skips applied. Every method searches
  this population of the tree at T, and only it.
- **Pools.** Each method finds its own pairs. doppel's pool is its retrieval
  union at the calibrated threshold, as in the ranker-outcome study. dupl's
  pool is the pairs its clone groups imply. The all-pairs detectors' pool is
  every same-build-unit pair (`parser.SameBuildUnit`) of the population that
  passes the method's size floor. No method is restricted to doppel's union.
- **Methods.**

  | method | pairs and key |
  | --- | --- |
  | doppel | the ranker-outcome study's doppel list: `analyzer.SortForReport` over the union, no `--max-per-func` cap |
  | dupl (t=100, default) | `dupl -t 100` (github.com/mibk/dupl v1.1.0, installed into GOBIN) over the population's files at T. Clone groups map to function pairs exactly as `internal/bench/clones_test.go` does; key = clone size in lines, the smaller of the two sides' overlap with the fragment |
  | dupl (t=50) | the same at `-t 50` |
  | token clones (≥100 nodes) | Jaccard over `Fingerprint.Shingles` (the ranker-outcome study's token-clone key), all pairs with both sides at `Fingerprint.Nodes` ≥ 100 |
  | token clones (≥50 nodes) | the same, both sides ≥ 50 nodes |
  | token clones (no floor) | the same, no floor: the contrast that shows what the floor buys |
  | code-shape (≥50 nodes) | `fingerprint.Similarity(...).Score` under the run's own WL weights, all pairs with both sides ≥ 50 nodes |

  Ties break on unit index `(a, b)` for every method, as before. Each list is
  cut at rank 500. An all-pairs method lists only pairs with a key above 0.
- **Why the floors are in nodes.** dupl's "tokens" are serialized Go AST nodes
  (its `syntax.Serialize` emits one element per node), not lexical tokens.
  `Fingerprint.Nodes` is doppel's syntax-node count of the same body, so
  "both sides ≥ 100 nodes" is the all-pairs analogue of dupl's default floor:
  a 100-token clone needs two functions at least that large. The two node
  counts are close, not identical. 50 is the sensitivity floor, matching the
  dupl sensitivity threshold.
- **Why t=50 and not t=25.** `examples/baselines.md` used t=25 because dupl's
  default reports almost nothing on chi and conc. Neither is in this study, and
  50 keeps the dupl and floor sensitivities at one number.
- **Exact search, not sampling.** The all-pairs methods score every eligible
  pair (about 26M on moby with no floor) across cores and keep an exact top 500
  under the shared tie-break. The shingle and WL merges are linear in the two
  sorted lists, so this is affordable without an index. An inverted index would
  only prune pairs with no shared shingle, which cannot enter a top 500 anyway.
- **No random list.** A uniform draw from all pairs is almost entirely
  cross-package pairs that never co-change, which would say nothing the
  ranker-outcome study's random rows did not.
- **Outcomes.** `historylabel compare`, unchanged: the cost study's walk,
  judge and exclusions over T..pin, each distinct pair judged once whichever
  lists hold it.

### Metrics

As in `examples/ranker-outcomes.md`: **M1** (mean `cochanges /
min(editsA, editsB)`, 0 when a side was never edited), **M2** (share with a
lagged or unpropagated event), **E** (share with any event), **C** (distinct
commits behind those events), and M1 with each co-change commit's credit
divided among the pairs of the list it touches (clustered).

Added, all descriptive:

- **overlap with doppel**: how many of a method's top 100 are in doppel's top
  100;
- **median smallest side** of the top 100, in lines at T;
- **pairs with an event**: the count, not the share, in each top 100. A sparse
  detector can have a high rate over few pairs; the count says how many costly
  pairs it actually put in front of a reader.

### Primary comparison and decision rule

Unchanged from `examples/ranker-outcomes.md`:

- **Primary metric: M1 at K = 100.** A list shorter than 100 after dropped
  pairs is scored over the pairs it has; its `n` is reported, and nothing is
  padded.
- Per baseline X per corpus: `M1(doppel) − M1(X)` at K = 100 with a 95%
  percentile bootstrap CI (2000 resamples, fixed seed, each list resampled
  independently). doppel **beats** X on a corpus when the lower bound is > 0;
  X **beats** doppel when the upper bound is < 0; otherwise **not
  distinguishable**. A method with no pairs on a corpus is not scored there.
- Over the five corpora: doppel **beats** X when it beats X on at least 3 and
  loses on none; X **beats** doppel under the same rule the other way round;
  anything else is **not distinguishable**, reported without reinterpretation.
- Everything else — K = 50, ranks 101-500, M2, E, C, clustered M1, overlap,
  median size, event counts — is descriptive and cannot change the verdict.

### Validity checks, stated in advance

- **doppel's list reproduces the earlier study exactly.** The 500-pair doppel
  list per corpus must equal, pair for pair and in order, the doppel list in
  the committed `examples/ranker-outcomes/<corpus>.outcomes.json`. A mismatch
  means the T tree or the pipeline moved, and the study stops there.
- **Unresolved units** are reported per corpus; a pair with an unresolved side
  is dropped from every list and counted.
- **Determinism.** Two runs of the same corpus produce byte-identical rankings
  and outcomes.
- **Shared code.** If code shared with the earlier studies changes,
  `historylabel study` and `compare-summary` over the committed files must
  still reproduce them byte for byte, and `TestRankingsAt` must still write the
  same rankings.
- **dupl coverage.** Per corpus and threshold: clone groups, and the function
  pairs they map to.

### Expectations

1. **dupl will be precise but sparse.** At t=100 it will list fewer than 100
   pairs on cobra and gin, its pairs will be large functions, and its E rate
   may match or exceed doppel's while its event count stays lower. With few
   pairs, its CI will be wide, so **not distinguishable** is the likely
   verdict.
2. **The floor removes most of the trivial-body failure.** Floored token
   clones and floored code-shape will score well above the no-floor token
   list, which will look like the ranker-outcome study's token clones (near
   0). If doppel's earlier win over similarity baselines was mostly the
   missing floor, **floored methods will be not distinguishable from doppel**.
   If doppel beats them anyway, its evidence beyond similarity is earning the
   margin.
3. **doppel will beat token clones (no floor).**
4. **Overlap will be low.** Under a fifth of any all-pairs method's top 100
   will be in doppel's: the detectors and doppel look at different pairs.
5. **Power is low**, as before. Not distinguishable is the most likely result
   for every floored method and both dupl rows, and would mean these outcomes
   cannot separate doppel from a configured clone detector — not that one is
   better.

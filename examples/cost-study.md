# Do the pairs doppel flags cost maintainers anything?

A measurement, not a scoring change: nothing in the doppel module was modified
to produce it. The git walk lives in `scripts/` (`scripts/cost-study.sh`,
`scripts/historylabel study` / `summarize`), because the module must not know
git exists.

## Pre-registration

Written and committed before any study number was looked at. Nothing below this
section may change the design, the metrics or the rule.

### Design

- **Time split.** For each corpus, T is the latest first-parent commit at or
  before the pinned tag's commit date minus two years. If fewer than 100
  non-merge commits touching `*.go` fall in T..pin, T steps back one year at a
  time, at most to four years. doppel analyses the tree at T only; every
  outcome is read from the commits in T..pin. doppel never sees the window.
- **Operating point.** `doppel analyze . --languages go --top 0 --max-per-func 0
  --families 0` at T, at the shipped defaults (calibration on at rate 0.01). The
  calibrated threshold and struct-min are recorded per corpus from the
  snapshot's `params`. One run per corpus; the text report gives rank and pair
  kind, the `--format json` snapshot of the same run gives the units.
- **Treatment groups.** `top50`: report ranks 1-50 (`analyzer.SortForReport`
  order). `tail`: ranks 51-500.
- **Control group.** For every treated pair, 3 pairs of functions drawn from the
  same T snapshot, matched on
  - locality: same file / same directory (package), other file / same top-level
    directory, other package / different top-level directory; and
  - size: the unordered pair of log2 buckets of each side's body line count
    (1, 2-3, 4-7, 8-15, 16-31, 32-63, 64+), measured on the T tree.

  A control may not be any pair doppel reported at T (any rank), nor a pair of
  same-named functions in one file. Sampling is deterministic: one LCG per
  treated pair, seeded by FNV-1a over corpus, T and the pair's keys.
- **Outcomes** (all within T..pin, over non-merge commits, test files excluded,
  with `historylabel`'s existing exclusions for sweeps, campaigns and
  mechanical commits):
  - `edits` per side: non-noisy modifications of that function's body.
  - `cochanges`: commits that applied the same change to both sides
    (historylabel's co-change evidence).
  - `lagged`, `unpropagated`, `extracted`, `consolidated`: historylabel's
    evidence kinds, restricted to the window.
  - `gone`: a side absent at the pin (deleted, renamed or moved file name) —
    counted, never dropped.

### Primary metrics

- **M1, normalised co-change**: per pair `cochanges / min(editsA, editsB)`
  (0 when either side was never edited), averaged over the group.
- **M2, maintenance-miss rate**: fraction of pairs with at least one `lagged`
  or `unpropagated` event.

For each, the ratio is the treated group's mean over the mean of its matched
controls, with a 95% percentile bootstrap CI (2000 resamples, fixed seed;
a treated pair and its controls are resampled together).

### Decision rule

A corpus **shows cost** when, for the `top50` group, M1 or M2 has a ratio
>= 2.0 **and** a CI lower bound > 1.0. A ratio with a zero control mean and a
non-zero treated mean counts as >= 2.0; its CI is read as computed.

**Evidence of cost** = at least 4 of the corpora run show cost. Anything else is
reported as **no evidence of cost**, without reinterpretation. The tail group,
the per-kind split, the pooled figures and every other rate are descriptive
only and cannot change the verdict.

Planned corpora, in order: cobra, gin (pipeline debugging, still counted),
prometheus, hugo, then moby if time allows.

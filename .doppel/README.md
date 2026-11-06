# doppel labels

`labels.json` is a human-style review of the top 60 pairs `doppel analyze .` reported
for this repository on 2026-09-30 (default settings, tests excluded). Each pair is
`merge`, `refactor` or `false_positive`; a false positive also carries a `kind`
(mirror, entrypoint, separate-programs, skeleton, already-factored, accessor-family,
vocabulary, other) naming why it is not worth acting on. Contested pairs were left out.

It is ground truth for doppel's golden benchmark, not configuration: doppel's walk skips
dot-directories, so nothing here affects an analysis. Score it from a doppel checkout:

    DOPPEL_BENCH_CORPUS=<path to this repo> DOPPEL_BENCH_LABELS=<path to this repo>/.doppel/labels.json \
      go test ./internal/bench/ -run TestGoldenRanking -v -count=1

Baseline (mean rank per class), re-measured 2026-09-30 after doppel squared code-shape in its rank key: merge 21.9 (15/15 retrieved, 15 in top 50) · refactor 22.3 · false_positive 41.6 · 8 false positives in the top 20.
The benchmark's hard assertions fail at this baseline; that failure is what a
false-positive fix is measured against.

A side may pin its file with `aFile`/`bFile` (slash-separated, relative to the repository
root); a pair whose two sides share a qualified name — two `main.main`s, two `init`s, a
helper duplicated across two scripts — must pin both.

`verdicts.json`, when present, is written by `doppel verdict` each time somebody decides what to do
with a pair: merged, left (with a `kind` or `blocker` saying why) or deferred, with who, when, the
commit the tree sat on, and the reason. It is a labels file too, so it scores the same way:

    DOPPEL_BENCH_CORPUS=<path to this repo> DOPPEL_BENCH_LABELS=<path to this repo>/.doppel/verdicts.json \
      go test ./internal/bench/ -run TestGoldenRanking -v -count=1

A merged pair weighs 3 in its class's mean rank and a merge left alone 0.5. Each decision names its
own commit, and a pair merged since then survives only as wrappers, so score it against that commit.

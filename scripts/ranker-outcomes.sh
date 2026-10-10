#!/usr/bin/env bash
#
# Rank one corpus at the cost study's revision T under doppel and every
# baseline, then judge each ranked pair on what happened over T..pin.
#
#   scripts/ranker-outcomes.sh [-c <corpus>] [-o <out dir>] [-s ranker|clones]
#
# T and the pin are read from the committed cost study
# (examples/cost-study/<corpus>.cost.json), never chosen again. The ranking is
# internal/bench's TestRankingsAt over the T tree; the judging is
# `historylabel compare`. Like cost-study.sh, the git walk lives here and in
# scripts/historylabel, never in the doppel module. See
# examples/ranker-outcomes.md.
#
# Writes <out dir>/<corpus>.rankings.json and <corpus>.outcomes.json.
# `historylabel compare-summary` turns the outcomes into the tables.
#
# -s clones is the follow-up study (examples/clone-outcomes.md): dupl runs on
# the T tree through scripts/clone-baseline.sh -r, internal/bench's
# TestCloneRankingsAt lists doppel beside dupl and the all-pairs detectors, and
# the same `historylabel compare` judges them. Its files are
# <corpus>.clone-rankings.json and <corpus>.clone-outcomes.json.
set -euo pipefail

CORPUS="cobra"
OUT_DIR=""
STUDY="ranker"
while getopts ":c:o:s:" opt; do
  case "$opt" in
    c) CORPUS=$OPTARG ;;
    o) OUT_DIR=$OPTARG ;;
    s) STUDY=$OPTARG ;;
    *) echo "usage: $0 [-c corpus] [-o out-dir] [-s ranker|clones]" >&2; exit 2 ;;
  esac
done
case "$STUDY" in
  ranker) PREFIX=""; DEFAULT_DIR="ranker-outcomes" ;;
  clones) PREFIX="clone-"; DEFAULT_DIR="clone-outcomes" ;;
  *) echo "unknown study $STUDY: ranker or clones" >&2; exit 2 ;;
esac

MODULE=$(cd "$(dirname "$0")/.." && pwd)
CACHE=$(go env GOCACHE)/..
HIST_ROOT="${DOPPEL_HISTORY:-$CACHE/doppel-git-history}"
HIST="$HIST_ROOT/$CORPUS"
OUT_DIR="${OUT_DIR:-$HIST_ROOT/$DEFAULT_DIR}"
mkdir -p "$OUT_DIR"
OUT_DIR=$(cd "$OUT_DIR" && pwd)

COST="$MODULE/examples/cost-study/$CORPUS.cost.json"
if [ ! -f "$COST" ]; then
  echo "no cost study for $CORPUS at $COST: T is taken from it" >&2
  exit 1
fi
field() { sed -n "s/^  \"$1\": \"\([0-9a-f]*\)\",\$/\1/p" "$COST" | head -1; }
SINCE=$(field since)
PIN=$(field pin)
if [ -z "$SINCE" ] || [ -z "$PIN" ]; then
  echo "could not read since/pin from $COST" >&2
  exit 1
fi
if [ ! -d "$HIST/.git" ]; then
  echo "no full-history clone at $HIST; run scripts/cost-study.sh -c $CORPUS first" >&2
  exit 1
fi

WORK=$(mktemp -d)
WT="$WORK/t"
cleanup() {
  git -C "$HIST" worktree remove --force "$WT" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

go build -C "$MODULE/scripts/historylabel" -o "$WORK/historylabel" .
git -C "$HIST" -c core.longpaths=true worktree add --detach -q "$WT" "$SINCE"

RANKINGS="$OUT_DIR/$CORPUS.${PREFIX}rankings.json"
OUTCOMES="$OUT_DIR/$CORPUS.${PREFIX}outcomes.json"
echo "$CORPUS: ranking the tree at ${SINCE:0:9}" >&2
if [ "$STUDY" = clones ]; then
  "$MODULE/scripts/clone-baseline.sh" -r "$WT" -o "$WORK/clones" -t 100,50 "$CORPUS"
  (cd "$MODULE" && DOPPEL_BENCH_CLONERANK_AT="$WT" DOPPEL_BENCH_CLONERANK_CORPUS="$CORPUS" \
    DOPPEL_BENCH_CLONERANK_CLONES="$WORK/clones" DOPPEL_BENCH_CLONERANK_OUT="$RANKINGS" \
    go test ./internal/bench/ -run '^TestCloneRankingsAt$' -count=1 -v -timeout 120m | grep -E '^\s+[a-z_]+_test\.go|^(ok|FAIL|---)' >&2)
else
  (cd "$MODULE" && DOPPEL_BENCH_RANKINGS_AT="$WT" DOPPEL_BENCH_RANKINGS_CORPUS="$CORPUS" \
    DOPPEL_BENCH_RANKINGS_OUT="$RANKINGS" \
    go test ./internal/bench/ -run '^TestRankingsAt$' -count=1 -v -timeout 60m | grep -E '^\s+rankings_at_test|^(ok|FAIL|---)' >&2)
fi

"$WORK/historylabel" compare -repo "$HIST" -rankings "$RANKINGS" \
  -since "$SINCE" -pin "$PIN" -out "$OUTCOMES"
echo "wrote $OUTCOMES" >&2

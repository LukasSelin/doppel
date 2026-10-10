#!/usr/bin/env bash
#
# Rank one corpus at the cost study's revision T under doppel and every
# baseline, then judge each ranked pair on what happened over T..pin.
#
#   scripts/ranker-outcomes.sh [-c <corpus>] [-o <out dir>]
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
set -euo pipefail

CORPUS="cobra"
OUT_DIR=""
while getopts ":c:o:" opt; do
  case "$opt" in
    c) CORPUS=$OPTARG ;;
    o) OUT_DIR=$OPTARG ;;
    *) echo "usage: $0 [-c corpus] [-o out-dir]" >&2; exit 2 ;;
  esac
done

MODULE=$(cd "$(dirname "$0")/.." && pwd)
CACHE=$(go env GOCACHE)/..
HIST_ROOT="${DOPPEL_HISTORY:-$CACHE/doppel-git-history}"
HIST="$HIST_ROOT/$CORPUS"
OUT_DIR="${OUT_DIR:-$HIST_ROOT/ranker-outcomes}"
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

echo "$CORPUS: ranking the tree at ${SINCE:0:9}" >&2
(cd "$MODULE" && DOPPEL_BENCH_RANKINGS_AT="$WT" DOPPEL_BENCH_RANKINGS_CORPUS="$CORPUS" \
  DOPPEL_BENCH_RANKINGS_OUT="$OUT_DIR/$CORPUS.rankings.json" \
  go test ./internal/bench/ -run '^TestRankingsAt$' -count=1 -v -timeout 60m | grep -E '^\s+rankings_at_test|^(ok|FAIL|---)' >&2)

"$WORK/historylabel" compare -repo "$HIST" -rankings "$OUT_DIR/$CORPUS.rankings.json" \
  -since "$SINCE" -pin "$PIN" -out "$OUT_DIR/$CORPUS.outcomes.json"
echo "wrote $OUT_DIR/$CORPUS.outcomes.json" >&2

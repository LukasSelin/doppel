#!/usr/bin/env bash
#
# Analyse a series of git revisions and render them as one doppel timeline.
#
# This script is the deliberate split in `doppel timeline`'s design. Doppel
# reads no git history — that is a construction constraint of the tool, not an
# omission, and everything it claims rests on it — so producing the series is a
# caller's job and lives here rather than inside the Go module.
#
#   scripts/timeline.sh [-r <repo>] [-n <count>] [-o <out.html>] [-d <runs dir>] [rev-list args...]
#
# Every revision is analysed at ONE pinned operating point. That is not a
# nicety: calibration is on by default, so a series of independently calibrated
# runs derives a different threshold per revision and its pair sets are not
# comparable step to step. `doppel timeline` refuses such a series outright.
# --top 0 --max-per-func 0 matter for the same class of reason — `analyze
# --format json` otherwise stores the ranked report list rather than the full
# candidate set, and the pair half of the page would be bounded at the source.
set -euo pipefail

REPO="."
COUNT=20
OUT="timeline.html"
RUNS=""
THRESHOLD="${DOPPEL_THRESHOLD:-0.38}"
STRUCT_MIN="${DOPPEL_STRUCT_MIN:-0.0}"

while getopts ":r:n:o:d:" opt; do
  case "$opt" in
    r) REPO=$OPTARG ;;
    n) COUNT=$OPTARG ;;
    o) OUT=$OPTARG ;;
    d) RUNS=$OPTARG ;;
    *) echo "usage: $0 [-r repo] [-n count] [-o out.html] [-d runs-dir] [rev-list args...]" >&2; exit 2 ;;
  esac
done
shift $((OPTIND - 1))

REPO=$(cd "$REPO" && pwd)
KEEP=1
if [ -z "$RUNS" ]; then
  RUNS=$(mktemp -d)
  KEEP=0
fi
mkdir -p "$RUNS"

BIN=$(mktemp -d)/doppel
go build -o "$BIN" .

# A worktree per revision rather than `git checkout`: the tree being analysed
# must not be the tree the script is running from, and a detached worktree
# leaves the caller's working copy alone.
WT_ROOT=$(mktemp -d)
cleanup() {
  if [ -n "${WT:-}" ]; then git -C "$REPO" worktree remove --force "$WT" 2>/dev/null || true; fi
  rm -rf "$WT_ROOT"
  [ "$KEEP" = 0 ] && rm -rf "$RUNS" || true
}
trap cleanup EXIT

i=0
# --reverse so argument order is chronological: `doppel timeline` takes the
# series in the order it is given, since a snapshot carries no timestamp.
for rev in $(git -C "$REPO" rev-list --reverse -n "$COUNT" "${@:-HEAD}"); do
  WT="$WT_ROOT/$rev"
  git -C "$REPO" worktree add --detach -q "$WT" "$rev"
  printf -v n "%04d" "$i"
  short=$(git -C "$REPO" rev-parse --short "$rev")
  echo "analysing $short" >&2
  "$BIN" analyze "$WT" --format json \
    --threshold "$THRESHOLD" --struct-min "$STRUCT_MIN" \
    --top 0 --max-per-func 0 > "$RUNS/$n-$short.json" 2>/dev/null
  git -C "$REPO" worktree remove --force "$WT"
  WT=""
  i=$((i + 1))
done

if [ "$i" -lt 2 ]; then
  echo "need at least two revisions, analysed $i" >&2
  exit 1
fi

"$BIN" timeline "$RUNS"/*.json --target "$(basename "$REPO")" -o "$OUT"

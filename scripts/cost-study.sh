#!/usr/bin/env bash
#
# Measure whether the pairs doppel flags at revision T cost their maintainers
# anything over T..pin, against matched pairs doppel did not flag.
#
#   scripts/cost-study.sh [-c <corpus>] [-o <out dir>] [-k <controls per pair>]
#
# T is chosen by the pre-registered rule in examples/cost-study.md: two years
# before the pinned commit's date, stepped back a year at a time (at most to
# four) until the window holds 100 non-merge commits touching *.go. doppel
# analyses the tree at T and nothing else; `historylabel study` then replays
# T..pin. Like history-labels.sh, the git walk lives here and in
# scripts/historylabel, never in the doppel module.
#
# Writes <out dir>/<corpus>.cost.json. `historylabel summarize` turns one or
# more of those into the tables (task cost-study-summary).
set -euo pipefail

CORPUS="cobra"
OUT_DIR=""
K=3
while getopts ":c:o:k:" opt; do
  case "$opt" in
    c) CORPUS=$OPTARG ;;
    o) OUT_DIR=$OPTARG ;;
    k) K=$OPTARG ;;
    *) echo "usage: $0 [-c corpus] [-o out-dir] [-k controls]" >&2; exit 2 ;;
  esac
done

MODULE=$(cd "$(dirname "$0")/.." && pwd)
CACHE=$(go env GOCACHE)/..
HIST_ROOT="${DOPPEL_HISTORY:-$CACHE/doppel-git-history}"
HIST="$HIST_ROOT/$CORPUS"
OUT_DIR="${OUT_DIR:-$HIST_ROOT}"
mkdir -p "$OUT_DIR"
OUT="$OUT_DIR/$CORPUS.cost.json"

MANIFEST="$MODULE/internal/bench/corpora.go"
field() { awk -v n="\"$CORPUS\"" -v f="$1" '$0 ~ "Name:" && index($0, n) {on=1} on && $0 ~ f":" {gsub(/[",]/, "", $2); print $2; exit}' "$MANIFEST"; }
REPO_URL=$(field Repo)
PIN=$(field Commit)
if [ -z "$REPO_URL" ] || [ -z "$PIN" ]; then
  echo "corpus $CORPUS is not in $MANIFEST" >&2
  exit 1
fi

if [ ! -d "$HIST/.git" ]; then
  echo "cloning full history of $CORPUS into $HIST" >&2
  mkdir -p "$HIST_ROOT"
  git clone -q --no-checkout "$REPO_URL" "$HIST"
elif ! git -C "$HIST" cat-file -e "$PIN^{commit}" 2>/dev/null; then
  git -C "$HIST" fetch -q origin
fi

# The pre-registered choice of T.
PIN_TS=$(git -C "$HIST" log -1 --format=%ct "$PIN")
SINCE=""
for years in 2 3 4; do
  SINCE=$(git -C "$HIST" rev-list --first-parent -1 --before=$((PIN_TS - years * 365 * 86400)) "$PIN")
  n=$(git -C "$HIST" rev-list --count --no-merges "$SINCE..$PIN" -- '*.go')
  echo "$CORPUS: T $years years back is ${SINCE:0:9}, $n non-merge *.go commits in the window" >&2
  [ "$n" -ge 100 ] && break
done

WORK=$(mktemp -d)
WT="$WORK/t"
cleanup() {
  git -C "$HIST" worktree remove --force "$WT" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

go build -C "$MODULE" -o "$WORK/doppel" .
go build -C "$MODULE/scripts/historylabel" -o "$WORK/historylabel" .

git -C "$HIST" -c core.longpaths=true worktree add --detach -q "$WT" "$SINCE"
# The shipped defaults (calibration on), Go only, the full ranked list. The
# text report carries rank and pair kind; the snapshot carries the units.
FLAGS=(--languages go --top 0 --max-per-func 0 --families 0)
(cd "$WT" && "$WORK/doppel" analyze . --format json "${FLAGS[@]}" > "$WORK/t.json" 2>"$WORK/t.err")
(cd "$WT" && "$WORK/doppel" analyze . "${FLAGS[@]}" > "$WORK/t.txt" 2>/dev/null)
grep -i "calibrat" "$WORK/t.err" >&2 || true

"$WORK/historylabel" study -repo "$HIST" -snapshot "$WORK/t.json" -report "$WORK/t.txt" \
  -corpus "$CORPUS" -since "$SINCE" -pin "$PIN" -controls "$K" -out "$OUT"
echo "wrote $OUT" >&2

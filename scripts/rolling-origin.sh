#!/usr/bin/env bash
#
# Rank one corpus at earlier origins T_k and judge each ranking on its own
# window T_k..T_{k-1}, under both outcome studies' method sets.
#
#   scripts/rolling-origin.sh [-c <corpus>] [-o <out dir>] [-k "2 3"] [-s "ranker clones"] [-l]
#
# T_k is chosen by the pre-registered rule in examples/rolling-origin.md: the
# cost study's window length L (its year step), and T_k the latest
# first-parent commit at or before pin - k*L. k=1 is the cost study's T, which
# is asserted. -l lists the origins and their window sizes and runs nothing.
#
# An origin is run only when its window holds >= 100 non-merge *.go commits;
# the judging then refuses (exit 3) when doppel's list at T_k has under 100
# pairs. Both are decided before the window is read. Like cost-study.sh, the
# git walk lives here and in scripts/historylabel, never in the doppel module.
#
# Writes <out dir>/<corpus>.origins.tsv and, per origin k and study,
# <corpus>.o<k>.rankings.json / .outcomes.json (ranker) and
# <corpus>.o<k>.clone-rankings.json / .clone-outcomes.json (clones).
set -euo pipefail

CORPUS="cobra"
OUT_DIR=""
ORIGINS="2 3"
STUDIES="ranker clones"
LIST=0
while getopts ":c:o:k:s:l" opt; do
  case "$opt" in
    c) CORPUS=$OPTARG ;;
    o) OUT_DIR=$OPTARG ;;
    k) ORIGINS=$OPTARG ;;
    s) STUDIES=$OPTARG ;;
    l) LIST=1 ;;
    *) echo "usage: $0 [-c corpus] [-o out-dir] [-k origins] [-s studies] [-l]" >&2; exit 2 ;;
  esac
done

MODULE=$(cd "$(dirname "$0")/.." && pwd)
CACHE=$(go env GOCACHE)/..
HIST_ROOT="${DOPPEL_HISTORY:-$CACHE/doppel-git-history}"
HIST="$HIST_ROOT/$CORPUS"
OUT_DIR="${OUT_DIR:-$HIST_ROOT/rolling-origin}"
mkdir -p "$OUT_DIR"
OUT_DIR=$(cd "$OUT_DIR" && pwd)

COST="$MODULE/examples/cost-study/$CORPUS.cost.json"
field() { sed -n "s/^  \"$1\": \"\([0-9a-f]*\)\",\$/\1/p" "$COST" | head -1; }
COST_SINCE=$(field since)
PIN=$(field pin)
if [ -z "$COST_SINCE" ] || [ -z "$PIN" ]; then
  echo "could not read since/pin from $COST" >&2
  exit 1
fi
if [ ! -d "$HIST/.git" ]; then
  echo "no full-history clone at $HIST; run scripts/cost-study.sh -c $CORPUS first" >&2
  exit 1
fi

# The cost study's year step, recovered by replaying its rule, so that T_1 is
# its T by construction rather than by a second choice.
PIN_TS=$(git -C "$HIST" log -1 --format=%ct "$PIN")
origin() { git -C "$HIST" rev-list --first-parent -1 --before=$((PIN_TS - $1 * 365 * 86400)) "$PIN"; }
YEARS=""
for years in 2 3 4; do
  t=$(origin "$years")
  n=$(git -C "$HIST" rev-list --count --no-merges "$t..$PIN" -- '*.go')
  if [ "$n" -ge 100 ] || [ "$years" -eq 4 ]; then
    YEARS=$years
    break
  fi
done
if [ "$(origin "$YEARS")" != "$COST_SINCE" ]; then
  echo "$CORPUS: T_1 $(origin "$YEARS") is not the cost study's since $COST_SINCE" >&2
  exit 1
fi

# T_0 is the pin; T_k is k windows of YEARS back.
ORIG="$OUT_DIR/$CORPUS.origins.tsv"
printf 'corpus\tk\tsince\tsince_date\tend\tend_date\tgo_commits\n' > "$ORIG"
declare -A T
T[0]=$PIN
MAXK=1
for k in $ORIGINS; do [ "$k" -gt "$MAXK" ] && MAXK=$k; done
for ((k = 1; k <= MAXK; k++)); do
  T[$k]=$(origin $((k * YEARS)))
  if [ -z "${T[$k]}" ]; then
    printf '%s\t%d\t-\t-\t%s\t-\t0\n' "$CORPUS" "$k" "${T[$((k - 1))]}" >> "$ORIG"
    continue
  fi
  n=$(git -C "$HIST" rev-list --count --no-merges "${T[$k]}..${T[$((k - 1))]}" -- '*.go')
  printf '%s\t%d\t%s\t%s\t%s\t%s\t%d\n' "$CORPUS" "$k" "${T[$k]}" \
    "$(git -C "$HIST" log -1 --format=%cs "${T[$k]}")" "${T[$((k - 1))]}" \
    "$(git -C "$HIST" log -1 --format=%cs "${T[$((k - 1))]}")" "$n" >> "$ORIG"
done
cat "$ORIG" >&2
[ "$LIST" -eq 1 ] && exit 0

for k in $ORIGINS; do
  row=$(awk -F'\t' -v k="$k" 'NR > 1 && $2 == k' "$ORIG")
  n=$(printf '%s' "$row" | cut -f7)
  if [ -z "${T[$k]:-}" ] || [ "${n:-0}" -lt 100 ]; then
    echo "$CORPUS o$k: inadmissible, ${n:-0} non-merge *.go commits in the window" >&2
    continue
  fi
  for s in $STUDIES; do
    status=0
    MIN_DOPPEL=100 "$MODULE/scripts/ranker-outcomes.sh" -c "$CORPUS" -o "$OUT_DIR" -s "$s" \
      -t "${T[$k]}" -p "${T[$((k - 1))]}" -n "o$k" || status=$?
    case $status in
      0) ;;
      3) echo "$CORPUS o$k: inadmissible, doppel lists under 100 pairs" >&2; break ;;
      *) echo "$CORPUS o$k $s: failed ($status)" >&2; exit "$status" ;;
    esac
  done
done

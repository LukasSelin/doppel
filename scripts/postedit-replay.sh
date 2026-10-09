#!/usr/bin/env bash
#
# Replay a repository's history through `doppel hook post-edit`, to measure how
# often the hook would fire and on what.
#
#   scripts/postedit-replay.sh -g <git dir> [-n <commits>] [-o <out dir>] [-b <doppel binary>] [-m <min shape>]
#
# Each non-merge commit is treated as one session: the hook's baseline is taken
# at the parent (`doppel hook session-start`), the tree is moved to the commit,
# and every Go file the commit added or modified is handed to `hook post-edit`
# as one edit. That is the hook's own code path end to end — baseline, pinned
# operating point, probe selection, ledger, digest — so the firing rate it
# reports is the rate a session making that commit would have seen. A real
# session edits a file several times and the ledger then keeps later edits
# quiet, so per-file is if anything an overcount of interruptions.
#
# Matches are recorded down to -m (default 0.40) rather than at the shipped
# floor, so the summary can price several floors from one run. At the end the
# fired pairs are checked against HEAD: a pair one of whose sides no longer
# exists there was at least touched later — consolidated, renamed or deleted —
# which is a rough signal of a real finding and no more.
#
# Like scripts/timeline.sh this is the git side of the tool and lives outside
# the module; doppel itself never reads history.
set -euo pipefail

GITDIR=""
N=200
OUT=""
BIN="doppel"
MIN="0.40"
while getopts ":g:n:o:b:m:" opt; do
  case "$opt" in
    g) GITDIR=$OPTARG ;;
    n) N=$OPTARG ;;
    o) OUT=$OPTARG ;;
    b) BIN=$OPTARG ;;
    m) MIN=$OPTARG ;;
    *) echo "usage: $0 -g <git dir> [-n commits] [-o out dir] [-b doppel binary] [-m min shape]" >&2; exit 2 ;;
  esac
done
[ -n "$GITDIR" ] || { echo "-g <git dir> is required" >&2; exit 2; }
OUT="${OUT:-$(mktemp -d)}"
mkdir -p "$OUT"
TIP=$(git -C "$GITDIR" rev-parse HEAD)
WT=$(mktemp -d)/tree
git -C "$GITDIR" worktree add --detach -f "$WT" HEAD >/dev/null 2>&1
cleanup() { git -C "$GITDIR" worktree remove --force "$WT" >/dev/null 2>&1 || true; }
trap cleanup EXIT

# Native path form for the hook payload (the binary may be a Windows build).
native() { if command -v cygpath >/dev/null; then cygpath -m "$1"; else printf '%s' "$1"; fi; }
ROOT=$(native "$WT")
# Matches the hook's own baseline path: sha256 of the session id, first 16 bytes.
baseline_of() {
  local h; h=$(printf '%s' "$1" | sha256sum | cut -c1-32)
  printf '%s/doppel-baselines/%s.json' "$(native "${TMPDIR:-${TEMP:-/tmp}}")" "$h"
}

EDITS="$OUT/edits.tsv"   # commit  file  fired(0/1 at -m)  seconds
FIRES="$OUT/fires.tsv"   # commit  file  probe  state  match  match-file  shape  containment  locality  kind
printf 'commit\tfile\tfired\tseconds\n' > "$EDITS"
printf 'commit\tfile\tprobe\tstate\tmatch\tmatch_file\tshape\tcontainment\tlocality\tkind\n' > "$FIRES"

commits=$(git -C "$GITDIR" rev-list --no-merges -n "$N" "$TIP")
total=$(printf '%s\n' "$commits" | grep -c . || true)
i=0
for c in $commits; do
  i=$((i + 1))
  parent=$(git -C "$WT" rev-parse --verify -q "$c^" || true)
  [ -n "$parent" ] || continue
  files=$(git -C "$WT" diff --name-only --diff-filter=AM "$parent" "$c" -- '*.go' | grep -v '_test\.go$' || true)
  [ -n "$files" ] || continue

  git -C "$WT" checkout -q -f --detach "$parent"
  printf '{"hook-probe":"on"}\n' > "$WT/.doppel.json"
  sid="postedit-replay-$c"
  printf '{"session_id":"%s","cwd":"%s"}' "$sid" "$ROOT" | "$BIN" hook session-start >/dev/null 2>&1 || true
  git -C "$WT" checkout -q -f --detach "$c"
  printf '{"hook-probe":"on"}\n' > "$WT/.doppel.json"

  for f in $files; do
    [ -f "$WT/$f" ] || continue
    start=$(date +%s.%N)
    out=$(printf '{"session_id":"%s","cwd":"%s","tool_name":"Edit","tool_input":{"file_path":"%s/%s"}}' \
      "$sid" "$ROOT" "$ROOT" "$f" | "$BIN" hook post-edit --min-shape "$MIN" 2>/dev/null || true)
    secs=$(awk -v s="$start" -v e="$(date +%s.%N)" 'BEGIN{printf "%.2f", e-s}')
    fired=0
    [ -n "$out" ] && fired=1
    printf '%s\t%s\t%s\t%s\n' "${c:0:10}" "$f" "$fired" "$secs" >> "$EDITS"
    [ -n "$out" ] || continue
    # Unescape the additionalContext text and read the digest's own lines back.
    printf '%s' "$out" | sed 's/.*"additionalContext":"//; s/","hookEventName.*//; s/\\n/\n/g' |
      awk -v c="${c:0:10}" -v f="$f" '
        /^  [^ ~]/ { flush(); probe=$1; state=$2; gsub(/[(,]/, "", state); next }
        /^    ~ / { flush(); m=$2; mf=$3; shape=$5; cont=$7; loc=$9; kind=""; pending=1; next }
        /^      kind: / { sub(/^      kind: /, ""); kind=$0; next }
        END { flush() }
        function flush() { if (pending) printf "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", c, f, probe, state, m, mf, shape, cont, loc, kind; pending=0 }
      ' >> "$FIRES"
  done
  rm -f "$(baseline_of "$sid")"
  printf '\r%d/%d commits' "$i" "$total" >&2
done
printf '\n' >&2

# Which fired pairs still exist at HEAD: both sides' keys, read off a plain
# snapshot of the tip.
git -C "$WT" checkout -q -f --detach "$TIP"
rm -f "$WT/.doppel.json"
(cd "$WT" && "$BIN" analyze . --format json --top 0 --max-per-func 0 2>/dev/null) |
  grep -o '"key": *"[^"]*"' | sed 's/.*"key": *"//; s/"$//' | sort -u > "$OUT/head-keys.txt"

# Files are passed as operands, never through -v: awk reads escapes in a -v
# value, and a Windows temp path is full of backslashes.
awk -F'\t' '
  FNR == NR { live[$0]=1; next }
  FNR == 1 { next }
  { print $0 "\t" (($3 in live) && ($5 in live) ? "both-live" : "touched-later") }
' "$OUT/head-keys.txt" "$FIRES" > "$OUT/fires-at-head.tsv"

# Firing rate per edit and per commit at a ladder of floors.
for floor in 0.40 0.50 0.60 0.70 0.80 0.90; do
  awk -F'\t' -v fl="$floor" '
    FNR == NR { if (FNR > 1) { ne++; nc[$1]=1 } next }
    FNR > 1 && $7 + 0 >= fl { e[$1 "\t" $2]=1; cm[$1]=1; m++ }
    END {
      ce=0; for (k in e) ce++; cc=0; for (k in cm) cc++; tc=0; for (k in nc) tc++
      printf "floor %.2f: %d/%d edits fire (%.1f%%), %d/%d commits (%.1f%%), %d matches\n", fl, ce, ne, (ne ? 100*ce/ne : 0), cc, tc, (tc ? 100*cc/tc : 0), m
    }' "$EDITS" "$FIRES"
done | tee "$OUT/summary.txt"
echo "results in $OUT" >&2

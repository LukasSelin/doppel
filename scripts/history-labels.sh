#!/usr/bin/env bash
#
# Derive golden labels for one corpus from its git history.
#
#   scripts/history-labels.sh [-c <corpus>] [-r <repo url>] [-p <pin>] [-u <until>] [-o <out.json>] [-w]
#
# A hand review says two functions look alike; history says what maintainers
# did with them — replaced one by the other, applied one change to both, moved
# shared code out of both. That is evidence doppel never reads, which is what
# makes it an oracle for doppel rather than a mirror of it. See
# scripts/historylabel/main.go for the rules.
#
# Pairs come from `doppel analyze` at the pin (the benchmark's pinned commit),
# so the labels score against the same ladder rung `task golden` uses. The
# history clone is separate from the corpus cache: that one is shallow and its
# HEAD is verified against the manifest, so it must not be unshallowed here.
set -euo pipefail

CORPUS="cobra"
REPO_URL=""
PIN=""
UNTIL=""
OUT=""
WEAK=""
while getopts ":c:r:p:u:o:w" opt; do
  case "$opt" in
    c) CORPUS=$OPTARG ;;
    r) REPO_URL=$OPTARG ;;
    p) PIN=$OPTARG ;;
    u) UNTIL=$OPTARG ;;
    o) OUT=$OPTARG ;;
    w) WEAK="-weak" ;;
    *) echo "usage: $0 [-c corpus] [-r repo-url] [-p pin] [-u until] [-o out.json] [-w]" >&2; exit 2 ;;
  esac
done

MODULE=$(cd "$(dirname "$0")/.." && pwd)
CACHE=$(go env GOCACHE)/..
HIST_ROOT="${DOPPEL_HISTORY:-$CACHE/doppel-git-history}"
HIST="$HIST_ROOT/$CORPUS"
OUT="${OUT:-$HIST_ROOT/$CORPUS.history.labels.json}"

# Repo URL and pinned commit come from the bench manifest, the one place they
# are written down.
# A corpus outside the ladder names its own repo (-r) and must name its pin:
# there is no manifest commit to fall back on, and no benchmark rung to score
# the labels against, so the pin is simply the revision whose pairs get judged.
MANIFEST="$MODULE/internal/bench/corpora.go"
field() { awk -v n="\"$CORPUS\"" -v f="$1" '$0 ~ "Name:" && index($0, n) {on=1} on && $0 ~ f":" {gsub(/[",]/, "", $2); print $2; exit}' "$MANIFEST"; }
REPO_URL="${REPO_URL:-$(field Repo)}"
PIN="${PIN:-$(field Commit)}"
if [ -z "$REPO_URL" ] || [ -z "$PIN" ]; then
  echo "corpus $CORPUS is not in $MANIFEST; pass -r <repo url> and -p <pin>" >&2
  exit 1
fi

if [ ! -d "$HIST/.git" ]; then
  echo "cloning full history of $CORPUS into $HIST" >&2
  mkdir -p "$HIST_ROOT"
  git clone -q "$REPO_URL" "$HIST"
else
  git -C "$HIST" fetch -q origin
fi
UNTIL="${UNTIL:-$(git -C "$HIST" symbolic-ref --short refs/remotes/origin/HEAD)}"

WORK=$(mktemp -d)
WT="$WORK/pin"
cleanup() {
  git -C "$HIST" worktree remove --force "$WT" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

go build -C "$MODULE" -o "$WORK/doppel" .
go build -C "$MODULE/scripts/historylabel" -o "$WORK/historylabel" .

git -C "$HIST" worktree add --detach -q "$WT" "$PIN"
# --top 0 --max-per-func 0: the full candidate set, not the ranked report.
(cd "$WT" && "$WORK/doppel" analyze . --format json --top 0 --max-per-func 0 > "$WORK/pin.json" 2>/dev/null)

HAND="$MODULE/examples/labels/$CORPUS.labels.json"
LABELS=()
[ -f "$HAND" ] && LABELS=(-labels "$HAND")

"$WORK/historylabel" -repo "$HIST" -snapshot "$WORK/pin.json" -pin "$PIN" -until "$UNTIL" \
  "${LABELS[@]}" -out "$OUT" $WEAK
echo "wrote $OUT" >&2

#!/usr/bin/env bash
#
# Run an external clone detector over the fetched ladder corpora and write its
# clone groups as JSON, for TestBaselines' method 8.
#
#   scripts/clone-baseline.sh [-o <out dir>] [-t <tokens>[,<tokens>...]] [corpus ...]
#
# The detector is dupl (github.com/mibk/dupl), a token-sequence clone finder
# over the Go AST. It is installed into GOBIN, never into this module:
#
#   go install github.com/mibk/dupl@v1.1.0
#
# It runs outside the doppel module for the reason scripts/historylabel and
# timeline.sh do: it is tooling about the tool, and go.mod keeps Cobra as its
# only direct dependency. The bench side reads only the JSON this writes.
#
# Population: non-test .go files, skipping the directories the pipeline's walk
# skips (dot- and underscore-prefixed, plus parser.DefaultExcludes, read out of
# internal/parser/exclude.go so the two cannot drift) and files carrying Go's
# generated-code marker. The bench side maps fragments onto the population's
# own functions, so a file this filter lets through and doppel does not simply
# contributes no pairs.
#
# Output, one file per corpus and threshold:
#   <out>/<corpus>.dupl-t<tokens>.clones.json
#   {"tool":"dupl","version":"...","threshold":N,"corpus":"...",
#    "groups":[{"fragments":[{"file":"a/b.go","start":10,"end":20},...]},...]}
# Paths are corpus-relative and slash-separated; fragments are sorted within a
# group and groups are sorted, so the file is byte-identical across runs.
# dupl reports no token count, only line spans; the bench side keys on lines.
set -euo pipefail

MODULE=$(cd "$(dirname "$0")/.." && pwd)
CACHE_ROOT=$(go env GOCACHE)/..
OUT="${DOPPEL_CLONES:-$CACHE_ROOT/doppel-clones}"
THRESHOLDS="100,25"
while getopts ":o:t:" opt; do
  case "$opt" in
    o) OUT=$OPTARG ;;
    t) THRESHOLDS=$OPTARG ;;
    *) echo "usage: $0 [-o out-dir] [-t tokens[,tokens...]] [corpus ...]" >&2; exit 2 ;;
  esac
done
shift $((OPTIND - 1))

DUPL=$(command -v dupl || true)
if [ -z "$DUPL" ]; then
  GOBIN=$(go env GOBIN)
  GOBIN=${GOBIN:-$(go env GOPATH)/bin}
  for c in "$GOBIN/dupl" "$GOBIN/dupl.exe"; do
    [ -x "$c" ] && DUPL=$c
  done
fi
if [ -z "$DUPL" ]; then
  echo "dupl not found; install it with: go install github.com/mibk/dupl@v1.1.0" >&2
  exit 1
fi
# dupl prints no version; record the installed module version from its build info.
VERSION=$( { go version -m "$DUPL" 2>/dev/null || go version -m "$DUPL.exe" 2>/dev/null || true; } | awk '$1 == "mod" {print $3; exit}')
VERSION=${VERSION:-unknown}

CORPORA_ROOT="${DOPPEL_CORPORA:-$CACHE_ROOT/doppel-corpora}"
MANIFEST="$MODULE/internal/bench/corpora.go"
if [ $# -eq 0 ]; then
  set -- $(awk '/Name:/ {gsub(/[",]/, "", $2); print $2}' "$MANIFEST")
fi

# DefaultExcludes, lowercased: every quoted name inside the var block.
EXCLUDES=$(awk '/^var DefaultExcludes = \[\]string\{/ {on=1; next} on && /^\}/ {exit}
  on { while (match($0, /"[^"]+"/)) { print tolower(substr($0, RSTART+1, RLENGTH-2)); $0 = substr($0, RSTART+RLENGTH) } }' \
  "$MODULE/internal/parser/exclude.go")
if [ -z "$EXCLUDES" ]; then
  echo "could not read DefaultExcludes from internal/parser/exclude.go" >&2
  exit 1
fi

mkdir -p "$OUT"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# listFiles prints the population's files under the current directory,
# corpus-relative, sorted.
listFiles() {
  find . -mindepth 1 -type d \( -name '.*' -o -name '_*' \) -prune -o -type f -name '*.go' ! -name '*_test.go' -print |
    sed 's|^\./||' |
    awk -v ex="$EXCLUDES" 'BEGIN { n = split(ex, e, "\n"); for (i = 1; i <= n; i++) skip[e[i]] = 1 }
      { k = split($0, p, "/"); ok = 1; for (i = 1; i < k; i++) if (tolower(p[i]) in skip) ok = 0; if (ok) print }' |
    while IFS= read -r f; do
      # Go's marker: a line comment "// Code generated ... DO NOT EDIT." before the code.
      if ! grep -qE '^// Code generated .* DO NOT EDIT\.\s*$' "$f"; then
        printf '%s\n' "$f"
      fi
    done | LC_ALL=C sort
}

# toJSON turns dupl's text report ("found N clones:" then indented
# "path:start,end" lines) into the sorted JSON document.
toJSON() {
  local corpus=$1 threshold=$2
  tr -d '\r' |
    awk '
      function flush(   i, j, t, line) {
        if (n < 2) { n = 0; return }
        # insertion sort by (file, start, end)
        for (i = 2; i <= n; i++) { t = f[i]; j = i - 1
          while (j >= 1 && less(t, f[j])) { f[j+1] = f[j]; j-- }
          f[j+1] = t }
        line = ""
        for (i = 1; i <= n; i++) line = line (i > 1 ? "|" : "") f[i]
        print line
        n = 0
      }
      function less(x, y,   a, b) {
        split(x, a, "\t"); split(y, b, "\t")
        if (a[1] != b[1]) return a[1] < b[1]
        if (a[2] + 0 != b[2] + 0) return a[2] + 0 < b[2] + 0
        return a[3] + 0 < b[3] + 0
      }
      /^found [0-9]+ clones:/ { flush(); next }
      /^  / {
        s = substr($0, 3); gsub(/\\/, "/", s)
        k = match(s, /:[0-9]+,[0-9]+$/)
        if (!k) next
        file = substr(s, 1, k - 1); split(substr(s, k + 1), r, ",")
        f[++n] = file "\t" r[1] "\t" r[2]
        next
      }
      END { flush() }
    ' |
    LC_ALL=C sort -u |
    awk -v corpus="$corpus" -v threshold="$threshold" -v version="$VERSION" '
      function esc(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); return s }
      BEGIN { printf "{\"tool\":\"dupl\",\"version\":\"%s\",\"threshold\":%d,\"corpus\":\"%s\",\"groups\":[", version, threshold, corpus }
      {
        printf "%s\n{\"fragments\":[", (NR > 1 ? "," : "")
        m = split($0, fr, "|")
        for (i = 1; i <= m; i++) { split(fr[i], p, "\t")
          printf "%s{\"file\":\"%s\",\"start\":%d,\"end\":%d}", (i > 1 ? "," : ""), esc(p[1]), p[2], p[3] }
        printf "]}"
      }
      END { print "\n]}" }
    '
}

for corpus in "$@"; do
  dir="$CORPORA_ROOT/$corpus"
  if [ ! -d "$dir/.git" ]; then
    echo "[$corpus] not fetched under $CORPORA_ROOT; skipping (task corpora)" >&2
    continue
  fi
  (cd "$dir" && listFiles) > "$TMP/files"
  echo "[$corpus] $(wc -l < "$TMP/files") files" >&2
  IFS=',' read -ra ts <<< "$THRESHOLDS"
  for t in "${ts[@]}"; do
    out="$OUT/$corpus.dupl-t$t.clones.json"
    (cd "$dir" && "$DUPL" -t "$t" -files < "$TMP/files") | toJSON "$corpus" "$t" > "$TMP/out.json"
    mv "$TMP/out.json" "$out"
    echo "[$corpus] t=$t: $(grep -c '^.{"fragments"\|^{"fragments"' "$out") groups -> $out" >&2
  done
done

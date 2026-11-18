#!/usr/bin/env bash
#
# Fail if anything under plugin/ changed and the plugin's version did not go up.
#
#   scripts/plugin-version-check.sh [base]
#
# Claude Code decides whether an installed plugin needs updating from the
# version in plugin/.claude-plugin/plugin.json, and caches each install in a
# directory named after it. A change under plugin/ that leaves the version
# alone never reaches a machine that already has the plugin, with or without
# auto-update: the update finds the same version and does nothing.
#
# This compares the tree, uncommitted changes included, against its merge base
# with base (default origin/master). It fails when a file under plugin/
# differs and the version does not sort strictly higher than the base's.
#
# Every file under plugin/ counts, README and tests included, because the
# whole directory is what an install copies. The marketplace manifest
# (.claude-plugin/marketplace.json) does not: Claude Code refreshes a
# marketplace directly, with no version to compare.
#
# It cannot see the doppel binary. The hooks run whatever doppel is on PATH,
# so a new binary still needs go install, whatever this says.

set -euo pipefail

cd "$(dirname "$0")/.."

base=${1:-origin/master}
manifest=plugin/.claude-plugin/plugin.json

err() {
	if [ -n "${GITHUB_ACTIONS:-}" ]; then
		echo "::error::$*" >&2
	else
		echo "$*" >&2
	fi
}

if ! mb=$(git merge-base "$base" HEAD 2>/dev/null); then
	err "plugin-version-check: no merge base with $base (fetch it first)"
	exit 2
fi

changed=$(git diff --name-only "$mb" -- plugin/)
if [ -z "$changed" ]; then
	echo "plugin-version-check: nothing under plugin/ changed since $base"
	exit 0
fi

# plugin.json carries one "version" key, on a line of its own.
version_of() {
	sed -n 's/^[[:space:]]*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1
}

old=$(git show "$mb:$manifest" 2>/dev/null | version_of || true)
new=$(version_of < "$manifest")

if [ -z "$new" ]; then
	err "plugin-version-check: no version in $manifest"
	exit 1
fi

# A plugin that did not exist at the base has nothing to be ahead of.
if [ -z "$old" ]; then
	echo "plugin-version-check: $manifest is new, at version $new"
	exit 0
fi

highest=$(printf '%s\n%s\n' "$old" "$new" | sort -V | tail -n1)
if [ "$new" != "$old" ] && [ "$highest" = "$new" ]; then
	echo "plugin-version-check: plugin/ changed and its version went $old -> $new"
	exit 0
fi

err "plugin-version-check: plugin/ changed since $base, but $manifest is at $new, not above the base's $old. Raise \"version\" above $old, or installed copies will never update."
echo "Changed under plugin/:" >&2
printf '%s\n' "$changed" | sed 's/^/  /' >&2
exit 1

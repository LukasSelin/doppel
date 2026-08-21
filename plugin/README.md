# doppel — Claude Code plugin

Two hooks around a Go repo:

- **At session start**, an inventory of the concepts the codebase already contains, so "is there
  already something that does this?" has an answer before anything gets written.
- **At the end of each turn**, what the session so far did to the repo's duplication surface —
  which near-duplicate pairs it introduced, which it removed.

## Install

`doppel` must be on your `PATH`:

```bash
go install github.com/LukasSelin/doppel@latest
```

Then add the marketplace and install the plugin:

```bash
claude plugin marketplace add LukasSelin/doppel
```

```bash
claude plugin install doppel@doppel
```

If the binary is not on your `PATH`, set the `doppel_binary` option to an absolute path when the
plugin asks, or in the plugin's configuration afterwards.

## What each hook does

### SessionStart

Runs a full analysis and adds a short factual summary to the conversation:

```
doppel corpus snapshot of myrepo — 261 Go functions (test functions excluded).
Concept tags present: validation 5, caching 3, error_wrapping 3, mapping 3.
Concept tags with no occurrence in this corpus: concurrency, db_access, http_call, retry, transaction.
Structural roles: leaf 148, orchestrator 49, passthrough 11, utility 55.
Concepts by package: cmd — caching, error_wrapping, mapping, validation; retriever — caching.
Near-duplicate pairs reported at threshold 0.60: 552, of which 101 are merge-worthy.
  parser.TagSignals.AnyIdent <-> parser.TagSignals.AnyImport  shape 1.00  overlap 0.72
  ...
```

The absent-tags line is usually the most useful: "nothing here is tagged `retry`" is a direct answer
where the present-tags list only narrows the search.

It also records a **baseline** — the measurement origin the Stop hook compares against. The baseline
is written once per session; SessionStart also fires on resume and after compaction, and re-recording
then would quietly move the origin so the impact report described the last few minutes instead of the
session.

### Stop

Re-runs the analysis and reports the difference against the baseline:

```
doppel impact this session: functions 261 -> 262, candidate pairs 551 -> 560, merge-worthy 100 -> 101.
  NEW  parser.sortedSet <-> snapshot.probeSortedNames  shape 0.90  overlap 0.43  (merge-worthy)
  NEW  culture.sortedStrings <-> snapshot.probeSortedNames  shape 1.00  overlap 0.30
  (4 more pair changes from functions edited this session, not listed)
  1 further pair changes involve no function edited this session (retrieval re-ranking).
  Full delta: /tmp/doppel-baselines/1f23b7da….impact.json
```

It is **cumulative**, not per-turn: every turn compares against the same session-start baseline, so
the report answers "what has this session done so far". When nothing changed it prints nothing at
all — a "no changes" line after every turn would only train you to stop reading.

The digest goes to you, not to the agent. A `Stop` hook that injects agent-visible context keeps the
conversation running, which would mean ending every turn by handing the agent its own report and
telling it to carry on working.

## Reading the output honestly

The report separates what it can prove from what it cannot, and so should you:

- **Functions added, removed, and bodies changed** are solid. They come from names and from a hash
  of each function's own AST, so nothing outside that function can move them.
- **Pair changes** carry an attribution bit. doppel retrieves a bounded number of candidate
  neighbours per function, so a pair can enter or leave the set without either side being edited.
  Changes traced to a function you actually edited are listed; the rest are only counted, as
  "retrieval re-ranking".
- **Scores are corpus-relative.** Concept weighting is derived from the tag frequencies of the tree
  being analysed, and structural roles from its call-graph degree distribution. Adding unrelated code
  moves both. This is why the impact report leads with identity and not with score movement.

Hook runs deliberately ignore the presentation settings in `.doppel.json` — `top`, `max-per-func` and
`struct-min` — and diff the full candidate set. A pair that fell past rank 20 has not changed, and
reporting it as your session's impact would be wrong. Everything that defines the *corpus*
(`threshold`, `min-nodes`, `channel-k`, `tests`) is honoured.

## Cost

Both hooks run a full analysis of the repository, which is O(n²) in candidate scoring with no
blocking. On a few hundred functions this is well under a second. On a very large repository the
Stop hook runs on every turn, so if it becomes noticeable, raise `threshold` or lower `channel-k`
in `.doppel.json`, or remove the Stop hook and keep only SessionStart.

## State

The baseline and the last delta live in `<temp>/doppel-baselines/`, named by a hash of the session
id. They are never read by `doppel analyze` and never feed any score; losing one costs you a delta
and nothing else. Files older than seven days are swept at session start.

## Running the hooks by hand

Each subcommand reads a hook payload on stdin and writes a hook response on stdout:

```bash
echo '{"session_id":"test","cwd":"'"$PWD"'","source":"startup"}' | doppel hook session-start
```

```bash
echo '{"session_id":"test","cwd":"'"$PWD"'"}' | doppel hook stop
```

Neither ever exits non-zero or writes to stderr: a measurement must not be able to break a session.
Silence means "nothing to report" — which is also what you get when there is no baseline yet.

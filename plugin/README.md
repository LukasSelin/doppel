# doppel — Claude Code plugin

[doppel](https://github.com/LukasSelin/doppel) measures architectural erosion: the gap between the
structure a project intends and the one it has, opened one locally reasonable edit at a time. An
agent is where those edits now get made, and it has the same blind spot a human reviewer does — it
sees the change, not the corpus. These hooks close it by putting the corpus in front of the model at
each of the four moments it can still act on: before writing, when the target is named, at the last
responsible moment, and after the fact.

Four hooks around a Go repo, ordered by when they fire:

- **At session start**, an inventory of the concepts the codebase contains — the session-stable
  framing: which tags exist here, which are absent, the role distribution.
- **At each prompt**, the duplication facts scoped to the packages the prompt mentions — the
  moment the target is first known and no edit exists yet.
- **Right before each Edit or Write**, an advisory naming the merge-worthy twins of the file
  about to change — the last responsible moment.
- **At the end of each turn**, what the session did to the duplication surface — which
  near-duplicate pairs it introduced, which it removed.

## Install

The plugin shells out to the `doppel` binary. It does not bundle one and never downloads one, so
install the binary first — either a prebuilt archive from the
[latest release](https://github.com/LukasSelin/doppel/releases/latest) (linux, darwin and windows,
amd64 and arm64), or:

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

The plugin's `doppel_binary` option defaults to the bare name `doppel`, resolved through your
`PATH` — so extracting an archive is not by itself enough. Either put the extracted binary on your
`PATH`, or set `doppel_binary` to its absolute path (`~/tools/doppel`, `C:\tools\doppel.exe` —
include the `.exe` on Windows) when the plugin asks, or in the plugin's configuration afterwards.

Check that the binary the plugin will run works:

```bash
doppel version
```

For the argument behind these placements — why SessionStart is deliberately vague, why the Stop
note costs a turn, why the pre-edit advisory reads session-start facts on purpose — see
[Hooks and the Causal Window](../.github/wiki/hooks.md).

## What each hook does

### SessionStart

Runs a full analysis and adds a short factual summary to the conversation:

```
doppel corpus snapshot of myrepo — 261 Go functions (test functions excluded).
Concept tags present: validation 5, caching 3, error_wrapping 3, mapping 3.
Concept tags with no occurrence in this corpus: concurrency, db_access, http_call, retry, transaction.
Structural roles: leaf 148, orchestrator 49, passthrough 11, utility 55.
Concepts by package: cmd — caching, error_wrapping, mapping, validation; retriever — caching.
Near-duplicate pairs reported at threshold 0.48: 552, of which 101 are merge-worthy.
  parser.TagSignals.AnyIdent <-> parser.TagSignals.AnyImport  shape 1.00  overlap 0.72
  ...
```

The absent-tags line is usually the most useful: "nothing here is tagged `retry`" is a direct answer
where the present-tags list only narrows the search.

The digest also names `doppel query`, which is the agent's follow-up tool: it can pipe a draft of a
function it intends to write into `doppel query --near <package> <root>` and get back the corpus
functions most related to it, nearest-by-call-graph first — answering "does something like this
already exist, and where" before the code is written rather than measuring the duplication after.

It also records a **baseline** — the measurement origin the Stop hook compares against. The baseline
is written once per session; SessionStart also fires on resume and after compaction, and re-recording
then would quietly move the origin so the impact report described the last few minutes instead of the
session.

### UserPromptSubmit

Extracts package mentions from your message — `@internal/billing` style paths and bare
package names, confirmed against the corpus — and emits only that package's facts:

```
doppel: internal/billing (41 functions)
  merge-worthy pairs within this package:
    Service.notifyClaimOpened <-> Service.notifyClaimClosed  shape 1.00  overlap 0.86
  3 merge-worthy pairs connect this package to others.
```

Silent when the prompt mentions nothing the corpus knows, which is most prompts. This runs a full
analysis once per prompt — same cost as the Stop hook.

### PreToolUse (Edit|Write)

Right before a file is edited, names the merge-worthy twins of the functions in it:

```
doppel (as of session start): functions in internal/billing/service.go have merge-worthy twins:
  Service.notifyClaimOpened <-> Service.notifyClaimClosed  shape 1.00  overlap 0.86
```

The facts come from the session-start baseline — the label says so — which is what makes this hook
millisecond-fast on every edit instead of costing a full analysis each time. Each file's advisory
fires **once per session**. It is advisory-only: it never blocks an edit, because a blocking dedupe
hook that misfires on a genuine near-duplicate would be worse than none.

### Stop

Re-runs the analysis and reports the difference against the baseline. It leads with **what happened
to each function** — matched by body, so a rename reads as a rename — and then the pairs those
changes created or dissolved:

```
doppel delta since the session baseline: split 1, moved 1, renamed 1, new 1, deleted 1; pairs created 74, dissolved 70.
  cobra.defaultUsageFunc (command.go:1974)  -> 2 bodies
      -> cobra.usageBody (command.go:1983)  containment 0.9921
      -> cobra.usageHeader (command.go:1974)  containment 0.9022
  cobra.stringInSlice (cobra.go:225) -> sliceutil.stringInSlice (sliceutil/slice.go:3)
      jaccard 1.0000  containment 1.0000  digests equal
  cobra.MinimumNArgs (args.go:74) -> cobra.AtLeastNArgs (args.go:74)
      jaccard 1.0000  containment 1.0000  digests equal
  cobra.ExactlyNArgs (args.go:104)  (no counterpart above the match floor)
  cobra.ExactValidArgs (args.go:129)  (no counterpart above the match floor)
  NEW  cobra.ExactArgs <-> cobra.ExactlyNArgs  shape 1.00  overlap 0.71  (merge-worthy)
      cobra.ExactlyNArgs new
      explain: identical after rename
  NEW  cobra.ExactlyNArgs <-> cobra.MaximumNArgs  shape 0.79  overlap 0.71  (merge-worthy)
      cobra.ExactlyNArgs new
      explain: differs by two extra binary
  NEW  cobra.AtLeastNArgs <-> cobra.ExactArgs  shape 0.79  overlap 0.71  (merge-worthy)
      cobra.AtLeastNArgs renamed
      explain: differs by two extra binary
  (71 more, not listed)
  GONE cobra.MaximumNArgs <-> cobra.MinimumNArgs  shape 0.79  overlap 0.71  (merge-worthy)
      cobra.MinimumNArgs renamed
      explain: differs by two extra binary
  ...
doppel impact this session: functions 269 -> 270, candidate pairs 1155 -> 1159, merge-worthy 157 -> 161.
  NEW  cobra.ExactArgs <-> cobra.ExactlyNArgs  shape 1.00  overlap 0.71  (merge-worthy)
  NEW  cobra.ExactlyNArgs <-> cobra.MaximumNArgs  shape 0.79  overlap 0.71  (merge-worthy)
  NEW  cobra.AtLeastNArgs <-> cobra.ExactArgs  shape 0.79  overlap 0.71  (merge-worthy)
  (52 more pair changes from functions edited this session, not listed)
  86 further pair changes involve no function edited this session (retrieval re-ranking).
  Full delta: …/doppel-baselines/44e17b81….impact.json
```

(Real output, from a session over cobra in which one function was split, one moved to another
package, one renamed, one deleted and one added as a copy. Trimmed at the `...` and at the temp
path; nothing else is changed.)

Each pair line carries the class of the function that explains it and the stored `explain:` sentence
saying what canonicalization did for that pair — `identical after rename` means the two canonical
trees agree and renaming is what got them there. Note the pair counts: one rename re-keys every pair
its function held, so 74 created and 70 dissolved is largely one fact restated, which is why only a
bounded head is printed.

It is **cumulative**, not per-turn: every turn compares against the same session-start baseline, so
the report answers "what has this session done so far". When nothing changed it prints nothing at
all — a "no changes" line after every turn would only train you to stop reading.

Two things come out of it, and they are not the same report.

**You** get the digest above: the classification, the pair changes both ways, the counts, the
re-ranking line, the path to the full delta.

**The agent** gets a much shorter note, and only when the session produced something worth
interrupting for — the delta scoreboard, three delta findings, then the notable pairs:

```
doppel matched this session's functions against the session baseline: split 1, moved 1, renamed 1, new 1, deleted 1; pairs created 74, dissolved 70.
  cobra.defaultUsageFunc (command.go:1974)  -> 2 bodies
      -> cobra.usageBody (command.go:1983)  containment 0.9921
      -> cobra.usageHeader (command.go:1974)  containment 0.9022
  cobra.stringInSlice (cobra.go:225) -> sliceutil.stringInSlice (sliceutil/slice.go:3)
      jaccard 1.0000  containment 1.0000  digests equal
  cobra.MinimumNArgs (args.go:74) -> cobra.AtLeastNArgs (args.go:74)
      jaccard 1.0000  containment 1.0000  digests equal
  (146 further delta findings not listed)
doppel measured this session's effect on the repository's duplication surface and found 9 new near-duplicate findings:
  cobra.ExactArgs <-> cobra.ExactlyNArgs  shape 1.00  overlap 0.71
  cobra.ExactlyNArgs <-> cobra.MaximumNArgs  shape 0.79  overlap 0.71
  cobra.AtLeastNArgs <-> cobra.ExactArgs  shape 0.79  overlap 0.71
  (6 further findings not listed)
This is a measurement, not a request. No change is required.
```

The classification never buys the turn: `additionalContext` is emitted only when the *notable* list
is non-empty, and its bar is unchanged. What identity buys, once a notable finding has already
justified the note, is the attribution the note is read through.

**This costs one extra turn.** A Stop hook cannot put text in the model's context without the
conversation continuing — that is how the harness works, not a choice doppel makes. So the bar for
saying anything to the agent is deliberately high: a *new* near-duplicate that is merge-worthy and
traceable to a function edited this session, or a pair that crossed the merge-worthy line because
you edited one of its sides. Count movements, look-alikes below the line, and pairs that shifted
because the corpus shifted never reach it. Each finding is reported **once** per session, so a
duplicate you leave in place does not interrupt you again on every later turn.

Set `hook-notify` in `.doppel.json` to change this:

| Value | Behaviour |
| --- | --- |
| `agent` (default) | the note above, plus your digest; costs a turn when there is a finding |
| `user` | your digest only, never continues a turn |
| `off` | silence |

## Reading the output honestly

The report separates what it can prove from what it cannot, and so should you:

- **The classification is solid.** Every class comes from a function's own key, its own fingerprint
  digest, or the label bag of its own body — never from anything the corpus does — so nothing
  outside a function can move it. `jaccard`, `containment` and `digests equal` are printed on the
  line precisely so you can check one by opening two files. The same goes for functions added,
  removed and bodies changed in the impact half.
- **Pair changes** carry an attribution bit. doppel retrieves a bounded number of candidate
  neighbours per function, so a pair can enter or leave the set without either side being edited.
  Changes traced to a classified function are listed and say which class; the rest read
  "no classified change on either side (retrieval re-ranking)" and sort last.
- **One rename restates itself.** A rename re-keys every pair its function held, so the created and
  dissolved counts run far ahead of the number of things that actually happened. The head of each
  list is where the new duplication is; the tail is the same rename seen from other pairs.
- **Scores are corpus-relative.** Concept weighting is derived from the learned concept frequencies
  of the tree being analysed, and structural roles from its call-graph degree distribution. Adding
  unrelated code moves both. This is why the report leads with identity and not with score movement.

Hook runs deliberately ignore the presentation settings in `.doppel.json` — `top`, `max-per-func` and
`struct-min` — and diff the full candidate set. A pair that fell past rank 20 has not changed, and
reporting it as your session's impact would be wrong. Everything that defines the *corpus*
(`threshold`, `min-nodes`, `channel-k`, `tests`, `calibrate`) is honoured.

The similarity floors are calibrated from the repo itself, and **derived once per session**: the
SessionStart baseline records the operating point and every later hook reuses it. Recalibrating
each turn would let your own edits move the threshold by a hundredth mid-session, which makes the
baseline incomparable and silences the Stop hook for a turn that nothing was wrong with. Pinning
`threshold` in `.doppel.json` turns calibration off, for hooks as for everything else.

## Cost

Both hooks run a full analysis of the repository. Candidate retrieval is sub-quadratic — inverted
indexes with a bounded top-K per function, not an all-pairs scan — and nothing blocks. On a few
hundred functions this is well under a second; a few thousand still lands in a couple of seconds.
The Stop hook runs on every turn, so on a very large repository it is the first thing to feel.

If it becomes noticeable, **`channel-k` is the lever that works**. It is the per-function per-channel
top-K, so it bounds the candidate set directly; dropping it from 5 to 2 roughly halves the pairs.
`threshold` looks like the obvious knob and mostly is not — it gates admission to the *structural*
channel only, and the concept and call channels bypass it entirely, so raising it prunes far less
than the number suggests. Failing that, remove the Stop hook and keep only SessionStart.

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

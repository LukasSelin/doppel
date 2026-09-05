# doppel

**doppel measures architectural erosion in a codebase — the widening gap between the structure a
project intends and the one it actually has.**

Nobody erodes an architecture deliberately. Someone needs a retry loop and writes one, because
finding the existing one costs more than writing it. Someone forks a handler for a second provider,
and a year later only one of the two carries the bug fix. Every such edit is defensible on its own
and invisible in review, because review sees a diff and erosion is a property of the whole corpus.
That gap is what doppel is pointed at, and it is why the tool reads every function in the repo at
once rather than reading a change.

Concretely: it fingerprints each function from its AST and cross-checks matches against call-graph
context, so it finds pairs that share *shape and role* rather than string overlap — the kind
text-based clone detection stops seeing once two copies have drifted apart. The output is a ranked
list of merge candidates with the evidence behind each one.

Everything runs locally and offline: no models, no network, no cache. The same source always produces the same report.

**[Open the interactive report for this repository](https://lukasselin.github.io/doppel/report.html)** —
doppel's own source, analysed by itself and regenerated from `master` on every push. The map
screen draws every package as a region sized by its share of the functions, with duplication
painted on the borders between them; the neighbourhood screen puts any two candidate bodies
side by side with the evidence for the match. ([What the screens show](https://lukasselin.github.io/doppel/).)
The same report on seven pinned public projects — moby, prometheus, hugo, gin, cobra, chi,
conc — is at **[the ladder](https://lukasselin.github.io/doppel/examples/)**.

For a detailed breakdown of the pipeline internals, see [How Doppel Works](.github/wiki/how-it-works.md).

## What erodes, and what the report calls it

| What happened to the code | What doppel reports |
| --- | --- |
| Two functions doing the same work, written independently | a ranked **pair**, with its shared structure and shared architectural context |
| Several of them | a **family** — a maximal clique, so every member is similar to every other |
| A copy that was forked and then diverged | the pair kind **`diverged copy`** |
| A function that claims a concept but realizes it unlike any peer | a **culture note** — drift *within* a shared idea rather than duplication of it |
| A function that no longer fits the package it lives in | a **habitat misfit**, excused when its wider subsystem still explains it |

The scope is worth stating as plainly as the target: **doppel measures the code against the corpus's
own practice, not against a declared architecture.** It has no model of what you intended, so it
cannot flag a layering violation — that is a linter's job. It reads no history, no config and no org
chart, so deployment drift (running state vs. declared state) and team drift are out of scope by
construction. What it can do, without being told anything, is notice that this repo has two of
something and that one of them is odd.

## Quick Start

**Prerequisites:** [Go 1.25+](https://go.dev/dl/)

```bash
go run . analyze .
```

This scans the current directory, fingerprints every function, and prints the most similar pairs with the evidence behind each match. Add `--output report.md` to also write a Markdown report.

Go is read with `go/ast` at full fidelity. Thirteen other languages — Python, TypeScript,
JavaScript, Rust, Java, C#, Kotlin, Swift, PHP, Ruby, C, C++, Scala — are read by a frontend
that has no grammar for them, only a tokenizer and a block rule; it finds 99.4–99.9% of the
functions `go/ast` finds when both are pointed at Go, at 100% precision, losing types and
therefore the signature component of the score. Functions in different languages are never
paired with each other. `--languages go` (or a `languages` key in `.doppel.json`) narrows the
corpus to one language.

The corpus is your repository's own code, not its dependencies. Doppel skips the directories
every ecosystem installs into — `node_modules`, `vendor`, `site-packages`, `Pods`, `Carthage`,
`third_party` — and the ones it builds into — `target`, `dist`, `build`, `out`, `obj`,
`coverage` — along with anything dot- or underscore-prefixed, as the go tool does.
Names that would shadow real source in a language doppel reads are deliberately absent:
`deps` is a Go package in hugo, `packages` is where a pnpm workspace keeps its own code, and
`bin` is where an npm package keeps its CLI. That matters
more than a walk rule usually does here: every judgment doppel makes is corpus-relative, so a
`node_modules` in the population means the learned vocabulary is named after somebody else's
code and every score is weighted against it. `--exclude` adds to the list, and `--exclude '!name'`
takes one back.

## Real-world examples

[`examples/`](examples/README.md) holds doppel's actual output on seven pinned
public Go projects, ordered old-and-complex to new-and-narrow — from
[moby](examples/moby.md) (8003 functions, a decade of accretion) down to
[conc](examples/conc.md) (81 functions, one idea, written recently). Each report
is regenerated by `task examples` — and by CI on every push to `master` — never
hand-edited, and every pair it names
can be opened in the corresponding pinned tree. The interactive counterpart of each is
published at [lukasselin.github.io/doppel/examples/](https://lukasselin.github.io/doppel/examples/).

The same directory carries the per-stage performance numbers across that ladder
and a committed [golden labels file](examples/labels/cobra.labels.json): a
human review of cobra's ranking that `task golden` scores the current pipeline
against.

## Installation

Download a prebuilt binary from the [latest release](https://github.com/LukasSelin/doppel/releases/latest)
— `linux`, `darwin` and `windows`, on `amd64` and `arm64`. Extract it and put `doppel` on your `PATH`:

```bash
tar -xzf doppel_*_linux_amd64.tar.gz
sudo install doppel /usr/local/bin/
```

Every release also ships `checksums.txt`:

```bash
sha256sum -c checksums.txt --ignore-missing
```

Or install with Go:

```bash
go install github.com/LukasSelin/doppel@latest
```

Or build from source:

```bash
git clone https://github.com/LukasSelin/doppel
cd doppel
go build -o doppel .
```

`doppel version` prints the build identity. That string is recorded in every snapshot, and a
session baseline is discarded when it changes — so upgrading mid-session correctly invalidates
the baseline instead of comparing measurements across a scoring change.

## Usage

```bash
doppel analyze <path> [flags]
```

### Examples

```bash
# Analyze current directory with defaults
doppel analyze .

# Catch more subtle similarities: admit 5% of random pairs instead of 1%
doppel analyze ./src --calibrate 0.05

# Pin the floors by hand instead of calibrating, and save a report
doppel analyze . --threshold 0.50 --struct-min 0.4 --output report.md

# Print the vocabulary scoring is based on, and check it is consistent
doppel ontology --defs

# Before writing a function, ask whether the repo already has one like it
doppel query --near billing . < draft.go

# Census: every group of 3+ mutually similar functions, not just pairs
doppel families .

# Show what the score reads for one function, or the label merge two of them score on
doppel fingerprint . mapper.sortedKeys
doppel fingerprint . mapper.sortedKeys lexicon.sortedKeys

# What happened to each function between two runs — renames and moves included
doppel analyze . --format json > before.json
doppel analyze . --format json > after.json
doppel diff before.json after.json
```

### Querying before you write

`doppel query` reads a Go snippet — the function you are about to write — and reports the corpus
functions most related to it by structure, learned concepts and calls:

```
query: cmd.validateHookSetup — concepts: validateMode+fmt.Errorf 0.71
  role: orchestrator   resolved calls: 3

Corpus: 304 functions. 5 related functions:

#1  cmd.hookParams  cmd/config.go:130
    evidence: 69.7 nats (shape 59.1, concept 1.4, call 9.2)  code-shape: 0.49  locality: 1.00
    concepts: validateMode+fmt.Errorf 0.64   role: orchestrator
```

Matches are ranked by evidence boosted with **locality** — the fraction of the snippet's resolved
call neighborhood the match inhabits — so architecturally near code outranks equally-similar code
from elsewhere. `--near` names the package the function will live in: a bare snippet is wrapped in
it, and its bare-name calls resolve to that package's functions, which is what locality is built
from. Include the snippet's imports — calls into imported packages only count as evidence when the
import that binds them is present.

### Checking a score by hand

`doppel fingerprint` shows the raw material behind a code-shape number. For one function it prints
the control-flow and nesting histograms, the type set, the canonicalization rules that fired, and
the Weisfeiler-Lehman label bag weighted by this corpus, heaviest label first. For two functions it
prints every component of the score with its blend weight, and the bag merge as three partitions —
shared, only in A, only in B — whose masses are the Jaccard and the containment:

```
  code-shape: 0.6277
    wl       0.5951 × 0.60 = 0.3571
    flow     0.8660 × 0.20 = 0.1732
    nesting  0.9487 × 0.05 = 0.0474
    sig      0.3333 × 0.15 = 0.0500
  containment: 0.8562 — most of the smaller body's shape is inside the larger

  label merge (weights ln(N/df), N = 950 bodies)
    mass A 131.3  mass B 101.4  shared 86.8  union 145.9
    jaccard = shared / union = 0.5951
    containment = shared / min(A, B) = 0.8562

  shared: 56 labels, 86.8 nats
        mass  weight    A    B    df  label
        3.39    3.39    1    1    32  depth-3 RANGE  #80e9c3fe3ce5ff64
        ...
```

The totals on the page add up to the numbers the report prints, which is the point: a score you
disagree with can be traced to the labels that produced it. Names are `package.Name` or
`package.*Receiver.Method`; a bare name works when it is unique. `--labels N` bounds the rows per
section (`0` for all).

A label is still a hash. `--label <hex>` (repeatable, the `#` optional) shows the node or nodes
that produced one — the subtree as Go text, and the exact extent the label hashed as an outline
truncated at the label's depth:

```
$ doppel fingerprint . mapper.sortedKeys lexicon.sortedKeys --label 80e9c3fe3ce5ff64
...
  label #80e9c3fe3ce5ff64: depth-3 RANGE  df 33  weight 3.38  A ×1  B ×1
    A mapper.sortedKeys
      node 21 of 38: range, subtree 4 deep (10 nodes); depth-3 folds 3 of those 4 levels
        for x2 := range x0 {
        	x1 = append(x1, x2)
        }
        hashed extent (3 levels; identifier names and literal values are not part of it):
          RANGE
            ID
            ID
            BLOCK
              ASSIGN/=
                ID
                CALL/append
```

The code shown is the **canonical form** — identifiers renamed to `x0, x1, …`, commutative operands
sorted, guards flipped — because that is the tree the bag was built over, and the canonical tree
keeps no source positions to map back to. A depth-3 label folds in three levels below its node and
nothing further, so the outline, not the code, is the exact claim; the code is printed because it is
what a reader recognizes.

### Two scores per pair

Every reported pair carries two independent numbers:

- **Code similarity** (`Score`, gated by `--threshold`, printed as `code-shape:`) — how alike the two bodies are, from the structural fingerprint. The report breaks it into its components: `wl` (corpus-weighted overlap of Weisfeiler-Lehman label bags over the canonical body), `flow` (control-flow shape), `nesting` (how deeply that control flow sits), `sig` (parameter and result types), and `size` (relative body size, shown for context but not scored). `containment` is shown beside them and also not scored — it asks how much of the *smaller* body's shape the larger one has, which is a different finding.
- **Structural overlap** (gated by `--struct-min`) — how much architectural context the two share: callees, callers, concepts, role, package, and what their own callers and callees do. Concepts, roles and receiver types are matched through a concept hierarchy rather than compared as strings, so two functions doing related work — one hitting a database, the other a cache — score partial credit instead of zero. Every graded match comes with an evidence line saying which ancestor relates the two and how strongly.

  The concepts themselves are **learned from your codebase**, not read off a fixed list. doppel finds
  the groups of functions that share a way of being written, names each after the evidence that
  identified it — `store.Get+store.Decode`, `json.Marshal+Unmarshal` — and reports membership as a
  confidence rather than a yes or no. That is why a repository whose database wrapper is called
  `store` gets a concept for it with no rule anywhere naming `store`.

A high code score with low structural overlap means two lookalike bodies in unrelated parts of the system. High on both is the real merge candidate.

When a naming rule can say what a pair *is*, a `kind:` line says it — `interface implementations`
(same method, same signature, different types: a `Validate` per provider) or `diverged copy` (alike
bodies whose names agree once version markers are stripped: `evalCall` beside `evalCallOld`). Both
are labels only; neither filters a pair or moves it in the ranking.

### Families

A pair is two functions. `doppel families` reports the *groups*: every set of three or more
functions in which **every member** is at least `--family-min` alike to every other member.

```
F1    7 members   every pair >= 0.75 code-shape  (3 edges scored here)
      culture.sortedStrings                           internal/culture/culture.go:214
      culture.sortedCountKeys                         internal/culture/ecology.go:175
      ...
```

Two things make that claim checkable. Families are **cliques**, not chains: A being similar to B
and B to C says nothing about A and C, and clustering that follows such links produces "families"
whose two ends have nothing in common. And because candidate retrieval keeps a bounded number of
neighbours per function, some edges inside a real family are never proposed — doppel scores those
directly before grouping, and says how many it added, so a family never rests on an edge you cannot
find in the pair list without being told.

A function can belong to more than one family; the counts report distinct functions. A family
whose every member pair satisfies one of the pair kinds says so on its line (`kind: interface
implementations of Flush()`), and in the JSON as `kind`. `analyze` shows the most informative few
inline; `doppel families` is the whole census, with `--format json` for a machine.

### Comparing two runs

`doppel diff` takes two snapshots written by `doppel analyze --format json` and matches their
functions to each other by **body**, not by name — so a rename reads as a rename rather than as a
deletion and an unrelated arrival. Every function lands in exactly one of eight classes and every
line prints the evidence that produced it:

```
$ doppel diff before.json after.json
Delta since the baseline
========================
269 functions before, 269 after
split 1, merged 0, moved 1, renamed 2, edited 0, new 0, deleted 1, unchanged 264

split 1
  cobra.defaultUsageFunc (command.go:1974)  -> 2 bodies
      -> cobra.usageBody (command.go:1994)  containment 0.9906
      -> cobra.usageHeader (command.go:1974)  containment 0.9670

moved 1
  cobra.stringInSlice (cobra.go:225) -> sliceutil.stringInSlice (sliceutil/slice.go:3)
      jaccard 1.0000  containment 1.0000  digests equal

renamed 2
  cobra.GetActiveHelpConfig (active_help.go:47) -> cobra.ActiveHelpConfig (active_help.go:47)  (body edited)
      jaccard 0.6798  containment 0.9082  digests differ
  cobra.OnlyValidArgs (args.go:51) -> cobra.ValidateArgs (args.go:51)
      jaccard 1.0000  containment 1.0000  digests equal

deleted 1
  cobra.ExactValidArgs (args.go:129)  (no counterpart above the match floor)

unchanged 264

pairs created 1, dissolved 1

pairs created 1
  cobra.ValidateArgs <-> cobra.usageBody  shape 1.00  overlap 0.70  (merge-worthy)
      cobra.ValidateArgs renamed
      explain: identical after rename

pairs dissolved 1
  cobra.ExactValidArgs <-> cobra.OnlyValidArgs  shape 1.00  overlap 0.68  (merge-worthy)
      cobra.ExactValidArgs deleted, cobra.OnlyValidArgs renamed
      explain: identical after rename
```

The second half is the one a plain diff cannot produce: the near-duplicate **pairs those changes
created or dissolved**, each attributed to the classified function that explains it and each
carrying the stored sentence saying what the canonicalizer did for it. A pair that moved with
nothing classified on either side says so — that is retrieval re-ranking around your change rather
than a consequence of it — and sorts after everything that is attributable.

`--output <file.md>` writes the same report as markdown. There is no HTML form for `diff` itself:
the dashboard describes one run, and a page over runs is a different artifact — which is what
`doppel timeline` below is.

A function that both moved and was renamed carries one class — the move — with the rename printed
alongside it, and the same goes for a rename that also edited the body. `--unchanged` lists the
unchanged functions instead of only counting them; `--format json` emits the whole result,
unchanged findings included and both pair lists appended, with every slice already in a total
order.

Exit codes: **0** compared, **1** a file could not be read, **2** the two snapshots refuse
comparison — a different schema, or a different canonicalization rule set, either of which would
make the same untouched bodies look different. A different threshold, ontology or doppel build does
*not* refuse: matching reads only each function's own body, so those are noted in the report and the
comparison runs.

This is a different question from what the Claude Code hooks answer. Those measure a session's
impact on the pair list and deliberately claim nothing they cannot attribute to an edit.

### Stepping through many runs

`doppel timeline` takes the same snapshots, N of them instead of two, and renders one page you step
through with the arrow keys — the classification at each revision, the pairs it created or
dissolved, and every function's lifeline across the whole series.

```bash
doppel timeline runs/*.json -o timeline.html
```

Argument order is series order. A snapshot deliberately carries no timestamp, so nothing in the
tool can sort them for you — and doppel reads no git history at all, which is why producing the
series is a script's job rather than the tool's. `scripts/timeline.sh` (or `task timeline`) is the
worked example: it walks `git rev-list`, checks each revision out into a throwaway worktree,
analyses it, and calls the command.

**Every revision must be analysed at the same operating point**, and the command refuses a series
that was not. Calibration is on by default, so runs left to derive their own thresholds are answers
to different questions and their pair counts do not belong on one axis. Analyse the series with an
explicit `--threshold` and `--struct-min` — which turns calibration off — plus `--top 0
--max-per-func 0`, so each snapshot stores the full candidate set rather than its twenty-pair
report list. The script does all of this; if you roll your own and forget the caps, the command
says so.

A function's **track** is joined only through one-to-one matches, so a rename or a move continues
the same line and you can watch one function across a refactor. A split or a merge ends a track
rather than continuing arbitrarily into one part — nothing decides which piece inherits the
lifeline, so the page labels the ending instead of guessing.

Even pinned, the learned concept vocabulary, roles, habitat fit and the nearest-neighbour
percentiles are properties of each revision's own corpus. The page reports them per step and says
so; it does not draw a trend line through them.

Exit codes match `doppel diff`: **0** rendered, **1** a file could not be read, **2** the series
refuses comparison.

### Flags

| Flag                | Default | Description                                                                 |
| ------------------- | ------- | --------------------------------------------------------------------------- |
| `--calibrate`       | `0.01`  | **The one knob that sets the others.** Fraction of random unrelated pairs the thresholds may admit. Doppel measures what an unrelated pair scores *in your repo* and derives `--threshold`, `--struct-min` and `--family-min` from it, so the setting means the same thing on 80 functions and on 8000. `0` falls back to the fixed defaults below |
| `-n`, `--top`       | `20`    | Maximum number of pairs to show (`0` for no limit)                          |
| `-t`, `--threshold` | *(calibrated)* | Pin the minimum code similarity to report (0.0–1.0). Setting it turns calibration off for the run, so the number you give is the number used. Falls back to `0.38` on a corpus too small to calibrate — the median of the six ladder corpora that do calibrate |
| `--struct-min`      | *(calibrated)* | Pin the minimum structural overlap (0.0–1.0) to keep a pair. Turns calibration off; falls back to `0.0` |
| `--min-nodes`       | `16`    | Skip functions whose body has fewer than this many AST nodes. A separate knob from `--calibrate`, which derives a *score* floor: this is an eligibility rule about which functions the shape channel indexes at all, and it guards against one-line accessors, which match each other perfectly and would otherwise flood the channel |
| `-o`, `--output`    | *(disabled)* | Write a report to this file. **A `.html` path renders the interactive dashboard** — one self-contained page that opens from `file://`, showing your packages as a political map (each region's area is its share of the functions, and a painted border is duplication crossing it) plus a per-function neighbourhood view with both bodies side by side. Any other extension writes Markdown, which opens with what doppel understands about the corpus — learned concepts, duplication map, package habitats — as mermaid diagrams, then how this codebase *writes* things. The stdout report is still printed |
| `--format`          | `text`  | Stdout format: `text` or `json`. The JSON form is a deterministic snapshot of the whole run — every function, its concepts with confidence, its role, its Weisfeiler-Lehman label bag, and every reported pair with its containment, its rule-attributed explanation and its concept views (the concept signal read through the taxonomy, through this corpus's frequencies, and through the learned vocabularies alone, plus the direction of the last) |
| `--families`        | `5`     | Near-duplicate families to show after the pair list (`0` removes the section) |
| `--family-min`      | *(calibrated)* | Pin the code similarity every two members of a family must reach. Turns calibration off; falls back to `0.60` |
| `--exclude`         | *(none, on top of the built-in blocklist)* | Directory patterns to skip. Doppel already skips the dependency and build-output directories of every ecosystem it reads — `node_modules`, `vendor`, `site-packages`, `Pods`, `Carthage`, `third_party`, `target`, `dist`, `build`, `out`, `obj`, `coverage` — because code your repo did not write is not a merge candidate, and counting it changes every corpus-relative number in the report. This adds to that list: a glob over a directory name, or over its path relative to the analysis root when it contains a `/`. A leading `!` takes one back, for a repo whose real source lives in a directory the defaults call build output (`--exclude '!dist'`) |
| `--config`          | `.doppel.json` if present | Path to a JSON config file                                |

`--min-nodes`, `--channel-k` and `--max-per-func` are retrieval and report
budgets rather than judgments about your code. They still work and still take
config keys, but they are hidden from `--help` because there is no question
about a codebase whose answer tells you what to set them to.

**Why the three similarity floors are calibrated rather than fixed.** A code
similarity of `0.60` is not one standard: on a small library almost nothing
reaches it, and on a large one a great deal of unrelated code does. Doppel draws
a sample of random, unrelated pairs from your repo, scores them, and takes the
quantile at which only `--calibrate` of them would be admitted — so "admit 1% of
random pairs" is the same question everywhere. Measured at the default rate the
derived floor is 0.45 on moby, 0.53 on cobra, 0.85 on conc. The calibration is
deterministic (a fixed seed derived from the corpus itself), it is printed on
stderr at the start of every run, and it is recorded in the JSON snapshot, so a
run always states the operating point it used. On a corpus too small to sample
from, doppel says so and keeps the fixed defaults.

### Configuration

Any flag above except `--config` can be set in a `.doppel.json` at the repo root. Keys are kebab-case, mirroring the flag names, and an explicit CLI flag always wins over the file:

```json
{
  "calibrate": 0.01,
  "top": 10,
  "families": 5,
  "output": "doppel-report.md",
  "exclude": ["generated", "!bin", "internal/proto/*"]
}
```

A missing config file is not an error; malformed JSON is. A malformed `exclude`
glob is an error too, rather than a pattern that quietly matches nothing: an
exclusion decides what the corpus is, and a corpus changed by a typo is not
something you notice until the report is already wrong.

Setting `threshold`, `struct-min` or `family-min` here turns calibration off for
every run in the repo, exactly as passing the flag would — pinning a floor and
deriving it are the same decision made two ways, and doppel will not do half of
each. Set `calibrate` explicitly if you want a pinned key and a calibrated rate
to coexist; the rate wins.

One key has no flag behind it: `hook-notify` (`agent` | `user` | `off`) decides who the plugin's
Stop hook reports to. See [plugin/README.md](plugin/README.md) — reaching the agent costs an extra
turn, so it is worth understanding before leaving it on the default.

## Use as a Claude Code plugin

The same analysis can run automatically around a coding session, answering questions it is otherwise
easy to skip: *does this codebase already have a concept for what I am about to write*, *does the
file I am about to edit have twins*, and *what did I just do to its duplication surface*.

The plugin shells out to the `doppel` binary and does not bundle one, so install it first —
a [release download](https://github.com/LukasSelin/doppel/releases/latest) or:

```bash
go install github.com/LukasSelin/doppel@latest
```

```bash
claude plugin marketplace add LukasSelin/doppel
```

```bash
claude plugin install doppel@doppel
```

Four hooks, placed by when a fact can still change what gets written:

- **SessionStart** — the corpus inventory: the concepts doppel learned from this repo and how many
  functions carry each, the kinds of work it found no practice for, the role distribution.
- **UserPromptSubmit** — the duplication facts for the packages your message mentions, and nothing
  else. Silent when it recognises none.
- **PreToolUse** on `Edit`/`Write` — immediately before a file changes, the merge-worthy twins of
  the functions in it. Advisory only; it never blocks an edit.
- **Stop** — what the session has done to the duplication surface, leading with the pairs it can
  trace to a function you actually edited. Prints nothing on turns that changed nothing.

Each is driven by a `doppel hook <name>` subcommand reading a Claude Code hook payload on stdin and
writing a hook response on stdout. None ever exits non-zero: a measurement must not be able to break
a session.

See [plugin/README.md](plugin/README.md) for what the output means and how to read it honestly, and
[Hooks and the Causal Window](.github/wiki/hooks.md) for why each hook fires where it does.

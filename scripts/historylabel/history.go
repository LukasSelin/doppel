package main

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Thresholds. Every one is a property of this labeller, never of doppel: the
// whole point of history labels is to be an oracle doppel's numbers did not
// produce, so nothing here reads a doppel score.
const (
	// parallelFloor is how alike two edits must be — weighted Jaccard over
	// their signed token deltas — to count as the same change made twice.
	parallelFloor = 0.5
	// minDelta is the fewest changed tokens an edit needs to be compared at
	// all; below it a "parallel" match is a one-token coincidence.
	minDelta = 3
	// sweepFuncs: a commit modifying more functions than this is a sweep (a
	// lint pass, interface{} -> any, a rename) whose edits are parallel by
	// construction and say nothing about any one pair.
	sweepFuncs = 10
	// renameSim: a function removed while a body at least this alike is added
	// in the same commit was renamed, not consolidated.
	renameSim = 0.8
	// copySim: a function born while its partner already existed, at least
	// this alike, began as a copy.
	copySim = 0.6
	// divergeDrop: how far two bodies' token similarity must fall from first
	// coexistence to the pin before independent editing reads as divergence.
	divergeDrop = 0.15
	// minInformativeLine: a removed line shorter than this ("}", "return nil")
	// is not evidence that the partner still carries the fixed code.
	minInformativeLine = 12
	// maxFamily: a change applied alike to more functions than this in one
	// commit is cross-cutting — every accessor of a struct gaining the same
	// line — and says the functions share a field, not that they are copies.
	maxFamily = 4
	// minMoved: the fewest tokens each side must lose into a new helper for
	// the commit to count as extracting shared code rather than swapping one
	// call for another.
	minMoved = 5
	// movedInto: how much of what each side lost must reappear in the helper.
	movedInto = 0.5
)

var fixSubject = regexp.MustCompile(`(?i)\b(fix(es|ed)?|bug|correct(s|ed)?|typo|wrong|panic|crash|regression)\b`)

// mechanicalSubject marks commits whose edits are parallel because a tool or a
// find-and-replace made them: a lint autofix applies the same rewrite to every
// copy of a pattern, duplicated or not, so its parallel edits are evidence
// about the pattern and never about a pair. Counting functions alone does not
// catch these — a lint pass over one package can touch fewer than sweepFuncs.
var mechanicalSubject = regexp.MustCompile(`(?i)\b(lint|golangci|gofmt|gofumpt|vet|staticcheck|modernize|autofix(es)?|replace[sd]?|rename[sd]?|deprecat\w*|cleanup|clean up|typos?|spelling|whitespace|style|cosmetics?|revert(s|ed)?)\b`)

type commit struct {
	sha, parent, subject string
	idx                  int
	post                 bool // after the pin
	sweep                bool
	edits                []edit   // every function this commit modified, tracked or not
	dirs                 []string // tracked directories this commit touched
}

// edit is one change to one function in one commit. old nil is a birth, new
// nil a removal.
type edit struct {
	c        *commit
	old, new *fn
}

func (e edit) modified() bool { return e.old != nil && e.new != nil }

// life is one tracked function's history, under its name at the pin. A rename
// ends it: what the function was called before is not followed.
type life struct {
	key, dir, name, file string
	edits                []edit
}

func (l *life) bare() string {
	if i := strings.LastIndexByte(l.name, '.'); i >= 0 {
		return l.name[i+1:]
	}
	return l.name
}

type history struct {
	r       *repo
	ts      *tables
	commits []*commit
	pin     *commit // last commit at or before the pin
	pinSHA  string
	lives   map[string]*life
}

// walk replays every non-merge commit touching a tracked directory, oldest
// first, and records each tracked function's edits. Merges are skipped
// because each of their changes already arrived in a non-merge commit;
// comparing a merge against its first parent would replay a whole branch as
// one edit.
//
// Each commit visits only the tracked directories its own diff touched. On a
// small corpus checking every directory per commit is free; on one with
// thousands of directories and a hundred thousand commits it is the whole
// cost.
func (h *history) walk(pin, until string, dirs []string) error {
	tracked := map[string]bool{}
	for _, d := range dirs {
		tracked[d] = true
	}
	pre, err := h.revList(pin, tracked)
	if err != nil {
		return err
	}
	post, err := h.revList(pin+".."+until, tracked)
	if err != nil {
		return err
	}
	for _, c := range post {
		c.post = true
	}
	h.commits = append(pre, post...)
	for i, c := range h.commits {
		c.idx = i
	}
	if len(pre) > 0 {
		h.pin = pre[len(pre)-1]
	}

	byDir := map[string][]*life{}
	for _, l := range h.lives {
		byDir[l.dir] = append(byDir[l.dir], l)
	}
	for i, c := range h.commits {
		if i > 0 && i%5000 == 0 {
			fmt.Fprintf(os.Stderr, "  %d/%d commits\n", i, len(h.commits))
			h.ts.trim()
		}
		modified := 0
		for _, dir := range c.dirs {
			newT, newSHA, err := h.ts.at(c.sha, dir)
			if err != nil {
				return err
			}
			var oldT table
			oldSHA := ""
			if c.parent != "" {
				if oldT, oldSHA, err = h.ts.at(c.parent, dir); err != nil {
					return err
				}
			}
			if newSHA == oldSHA {
				continue
			}
			for _, name := range newT.names() {
				for _, n := range newT[name] {
					if o := oldT.lookup(name, n.File); o != nil && o.Body != n.Body {
						modified++
						c.edits = append(c.edits, edit{c: c, old: o, new: n})
					}
				}
			}
			for _, l := range byDir[dir] {
				o, n := oldT.lookup(l.name, l.file), newT.lookup(l.name, l.file)
				if o == nil && n == nil || o != nil && n != nil && o.Body == n.Body {
					continue
				}
				l.edits = append(l.edits, edit{c: c, old: o, new: n})
			}
		}
		c.sweep = modified > sweepFuncs || mechanicalSubject.MatchString(c.subject)
	}
	return nil
}

// revList lists the non-merge commits in rng that touch a .go file in a
// tracked directory, each with the tracked directories it touched. The
// pathspec is a glob rather than the directory list: a large corpus tracks
// more directories than a Windows command line can carry. --no-renames makes
// a moved file show up under both its directories, so a function leaving one
// is seen leaving.
func (h *history) revList(rng string, tracked map[string]bool) ([]*commit, error) {
	out, err := h.r.git("log", "--reverse", "--topo-order", "--no-merges", "--no-renames", "--name-only",
		"--format=%x00%H%x09%P%x09%s", rng, "--", "*.go")
	if err != nil {
		return nil, err
	}
	var cs []*commit
	for _, rec := range strings.Split(out, "\x00") {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		lines := strings.Split(rec, "\n")
		f := strings.SplitN(lines[0], "\t", 3)
		c := &commit{sha: f[0]}
		if len(f) > 1 {
			c.parent, _, _ = strings.Cut(f[1], " ")
		}
		if len(f) > 2 {
			c.subject = f[2]
		}
		seen := map[string]bool{}
		for _, p := range lines[1:] {
			p = strings.TrimSpace(p)
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				continue
			}
			if d := path.Dir(p); tracked[d] && !seen[d] {
				seen[d] = true
				c.dirs = append(c.dirs, d)
			}
		}
		sort.Strings(c.dirs)
		if len(c.dirs) > 0 {
			cs = append(cs, c)
		}
	}
	return cs, nil
}

// state is l's body at the end of commit sha ("" parent means before history).
func (h *history) state(l *life, sha string) *fn {
	if sha == "" {
		return nil
	}
	t, _, err := h.ts.at(sha, l.dir)
	if err != nil {
		return nil
	}
	return t.lookup(l.name, l.file)
}

func (h *history) atPin(l *life) *fn { return h.state(l, h.pinSHA) }

// ---- similarity ----------------------------------------------------------

func counts(toks []string) map[string]int {
	m := map[string]int{}
	for _, t := range toks {
		m[t]++
	}
	return m
}

func jaccard(a, b map[string]int) float64 {
	num, den := 0, 0
	for k, x := range a {
		y := b[k]
		num += min(x, y)
		den += max(x, y)
	}
	for k, y := range b {
		if _, ok := a[k]; !ok {
			den += y
		}
	}
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

func sim(a, b *fn) float64 {
	if a == nil || b == nil {
		return 0
	}
	return jaccard(counts(a.Tokens()), counts(b.Tokens()))
}

// delta is an edit as a signed token multiset: "+x" for each token the edit
// added, "-x" for each it removed. Two edits applying the same change to two
// bodies have alike deltas even when the bodies around them differ.
func delta(e edit) map[string]int {
	o, n := map[string]int{}, map[string]int{}
	if e.old != nil {
		o = counts(e.old.Tokens())
	}
	if e.new != nil {
		n = counts(e.new.Tokens())
	}
	d := map[string]int{}
	for k, x := range n {
		if x > o[k] {
			d["+"+k] = x - o[k]
		}
	}
	for k, x := range o {
		if x > n[k] {
			d["-"+k] = x - n[k]
		}
	}
	return d
}

func size(m map[string]int) int {
	s := 0
	for _, v := range m {
		s += v
	}
	return s
}

// lostTokens is the multiset of tokens an edit removed from the body.
func lostTokens(e edit) map[string]int {
	o, n := counts(e.old.Tokens()), counts(e.new.Tokens())
	out := map[string]int{}
	for k, x := range o {
		if x > n[k] {
			out[k] = x - n[k]
		}
	}
	return out
}

// family counts the functions the edit's commit changed the same way,
// the edit's own function included.
func family(e edit) int {
	n := 0
	for _, x := range e.c.edits {
		if parallel(e, x) >= parallelFloor {
			n++
		}
	}
	return n
}

func parallel(a, b edit) float64 {
	da, db := delta(a), delta(b)
	if size(da) < minDelta || size(db) < minDelta {
		return 0
	}
	return jaccard(da, db)
}

// removedLines is what a modification took out of the body, informative lines
// only.
func removedLines(e edit) []string {
	keep := map[string]int{}
	for _, l := range e.new.Lines() {
		keep[l]++
	}
	var out []string
	for _, l := range e.old.Lines() {
		if keep[l] > 0 {
			keep[l]--
			continue
		}
		if len(l) >= minInformativeLine {
			out = append(out, l)
		}
	}
	return out
}

func carries(f *fn, lines []string) int {
	if f == nil {
		return 0
	}
	have := map[string]int{}
	for _, l := range f.Lines() {
		have[l]++
	}
	n := 0
	for _, l := range lines {
		if have[l] > 0 {
			have[l]--
			n++
		}
	}
	return n
}

func mentions(f *fn, name string) int {
	if f == nil {
		return 0
	}
	n := 0
	for _, t := range f.Tokens() {
		if t == name {
			n++
		}
	}
	return n
}

// ---- pair verdicts -------------------------------------------------------

// Verdicts, strongest evidence first. The order is the precedence: a pair
// carrying several kinds of evidence takes the first.
const (
	vConsolidated = "consolidated" // one side removed, its callers sent to the other
	vExtracted    = "extracted"    // both sides rewritten to call one new helper
	vSynced       = "synced"       // the same change applied to both, together or later
	vUnpropagated = "unpropagated" // a fix to one side whose old code the other still carries
	vDiverged     = "diverged"     // edited independently and grew apart (weak)
	vNone         = ""             // history says nothing
)

var verdictOrder = []string{vConsolidated, vExtracted, vSynced, vUnpropagated, vDiverged}

type evidence struct {
	Kind    string   `json:"kind"`
	Commits []string `json:"commits"`
	Detail  string   `json:"detail"`
}

type verdict struct {
	Verdict  string     `json:"verdict"`
	Origin   string     `json:"origin"`
	Evidence []evidence `json:"evidence"`
}

func short(c *commit) string { return c.sha[:9] }

func (h *history) judge(a, b *life) verdict {
	var v verdict
	v.Origin = h.origin(a, b)
	add := func(kind, detail string, cs ...*commit) {
		e := evidence{Kind: kind, Detail: detail}
		for _, c := range cs {
			e.Commits = append(e.Commits, short(c))
		}
		v.Evidence = append(v.Evidence, e)
	}

	byCommit := func(l *life) map[*commit]edit {
		m := map[*commit]edit{}
		for _, e := range l.edits {
			m[e.c] = e
		}
		return m
	}
	ea, eb := byCommit(a), byCommit(b)
	used := map[*commit]bool{} // edits already explained as half of a sync

	// Co-change: both sides edited in one commit, the same way.
	for _, e := range a.edits {
		f, ok := eb[e.c]
		if !ok || !e.modified() || !f.modified() || e.c.sweep {
			continue
		}
		if p := parallel(e, f); p >= parallelFloor && family(e) <= maxFamily {
			add("co-change", fmt.Sprintf("both edited alike (%.2f): %s", p, e.c.subject), e.c)
			used[e.c] = true
		}
	}

	// Lagged sync: one side edited alone, the other given the same change
	// in a later commit.
	lagged := func(x, y *life, ey map[*commit]edit) {
		for _, e := range x.edits {
			if !e.modified() || e.c.sweep || used[e.c] || family(e) > maxFamily {
				continue
			}
			if _, both := ey[e.c]; both || h.state(y, e.c.parent) == nil {
				continue
			}
			for _, f := range y.edits {
				if f.c.idx <= e.c.idx || !f.modified() || f.c.sweep || used[f.c] || family(f) > maxFamily {
					continue
				}
				if p := parallel(e, f); p >= parallelFloor {
					add("lagged-sync", fmt.Sprintf("%s changed first, %s caught up (%.2f): %q then %q",
						x.key, y.key, p, e.c.subject, f.c.subject), e.c, f.c)
					used[e.c], used[f.c] = true, true
					break
				}
			}
		}
	}
	lagged(a, b, eb)
	lagged(b, a, ea)

	// Unpropagated fix: a commit that calls itself a fix rewrote lines of one
	// side that the other carried then and still carries at the pin.
	unprop := func(x, y *life, ey map[*commit]edit) {
		for _, e := range x.edits {
			if !e.modified() || e.c.sweep || used[e.c] || e.c.post || !fixSubject.MatchString(e.c.subject) {
				continue
			}
			if _, both := ey[e.c]; both {
				continue
			}
			rm := removedLines(e)
			if len(rm) == 0 {
				continue
			}
			before := carries(h.state(y, e.c.parent), rm)
			if before == 0 || 2*before < len(rm) {
				continue
			}
			if carries(h.atPin(y), rm) < before {
				continue
			}
			add("unpropagated-fix", fmt.Sprintf("fixed in %s only; %s still carries %d of %d replaced lines: %s",
				x.key, y.key, before, len(rm), e.c.subject), e.c)
		}
	}
	unprop(a, b, eb)
	unprop(b, a, ea)

	// Extraction: one commit adds helper H, both sides start calling it, and
	// each side's lost code reappears in H — shared logic moved out, not one
	// call swapped for another.
	for _, e := range a.edits {
		f, ok := eb[e.c]
		// Both sides must exist before and after: a birth trivially "starts
		// calling" everything, and two removals are a deletion, not a share.
		if !ok || e.c.sweep || !e.modified() || !f.modified() {
			continue
		}
		for _, hn := range h.addedIn(e.c, a.dir) {
			if hn == a.name || hn == b.name {
				continue
			}
			bare := hn
			if i := strings.LastIndexByte(hn, '.'); i >= 0 {
				bare = hn[i+1:]
			}
			t, _, _ := h.ts.at(e.c.sha, a.dir)
			helper := t.lookup(hn, "")
			if helper == nil {
				continue
			}
			ht := counts(helper.Tokens())
			moved := func(x edit) bool {
				if mentions(x.new, bare) <= mentions(x.old, bare) {
					return false
				}
				lost := lostTokens(x)
				n := size(lost)
				if n < minMoved {
					return false
				}
				in := 0
				for k, v := range lost {
					in += min(v, ht[k])
				}
				return float64(in) >= movedInto*float64(n)
			}
			if moved(e) && moved(f) {
				add("extracted", fmt.Sprintf("both moved code into new %s: %s", hn, e.c.subject), e.c)
				break
			}
		}
	}

	// Consolidation: one side removed while the other lives on, and the
	// commit trades calls to the removed name for calls to the survivor.
	consol := func(x, y *life) {
		for _, e := range x.edits {
			if e.new != nil || e.old == nil || e.c.sweep || h.state(y, e.c.sha) == nil {
				continue
			}
			renamed := false
			for _, hn := range h.addedIn(e.c, x.dir) {
				t, _, _ := h.ts.at(e.c.sha, x.dir)
				if g := t.lookup(hn, ""); g != nil && sim(e.old, g) >= renameSim {
					renamed = true
				}
			}
			if renamed {
				continue
			}
			diff, err := h.r.git("show", "--format=", "-U0", e.c.sha)
			if err != nil {
				continue
			}
			lostX, gainY := false, false
			for _, l := range strings.Split(diff, "\n") {
				switch {
				case strings.HasPrefix(l, "---"), strings.HasPrefix(l, "+++"):
				case strings.HasPrefix(l, "-") && strings.Contains(l, x.bare()+"("):
					lostX = true
				case strings.HasPrefix(l, "+") && strings.Contains(l, y.bare()+"("):
					gainY = true
				}
			}
			if lostX && gainY {
				add("consolidated", fmt.Sprintf("%s removed, its calls now go to %s: %s", x.key, y.key, e.c.subject), e.c)
			}
		}
	}
	consol(a, b)
	consol(b, a)

	// Divergence: no shared change at all, both edited, and the bodies grew
	// apart. Weak: a copy that drifted apart by neglect looks the same.
	if len(v.Evidence) == 0 {
		start := h.coexistStart(a, b)
		na, nb := 0, 0
		for _, e := range a.edits {
			if e.modified() && start != nil && e.c.idx > start.idx && !e.c.post {
				na++
			}
		}
		for _, e := range b.edits {
			if e.modified() && start != nil && e.c.idx > start.idx && !e.c.post {
				nb++
			}
		}
		if start != nil && na > 0 && nb > 0 {
			s0 := sim(h.state(a, start.sha), h.state(b, start.sha))
			s1 := sim(h.atPin(a), h.atPin(b))
			if s0-s1 >= divergeDrop {
				add("diverged", fmt.Sprintf("similarity %.2f -> %.2f over %d+%d independent edits", s0, s1, na, nb), start)
			}
		}
	}

	for _, want := range verdictOrder {
		for _, e := range v.Evidence {
			if kindVerdict(e.Kind) == want {
				v.Verdict = want
				return v
			}
		}
	}
	return v
}

func kindVerdict(kind string) string {
	switch kind {
	case "co-change", "lagged-sync":
		return vSynced
	case "unpropagated-fix":
		return vUnpropagated
	}
	return kind
}

// addedIn lists functions present in dir after c and absent before it.
func (h *history) addedIn(c *commit, dir string) []string {
	newT, _, _ := h.ts.at(c.sha, dir)
	var oldT table
	if c.parent != "" {
		oldT, _, _ = h.ts.at(c.parent, dir)
	}
	var out []string
	for _, n := range newT.names() {
		if len(oldT[n]) == 0 {
			out = append(out, n)
		}
	}
	return out
}

// birth is l's last arrival at or before the pin.
func (h *history) birth(l *life) *commit {
	var b *commit
	for _, e := range l.edits {
		if e.old == nil && e.new != nil && !e.c.post {
			b = e.c
		}
	}
	return b
}

// coexistStart is the first commit after which both sides exist.
func (h *history) coexistStart(a, b *life) *commit {
	ba, bb := h.birth(a), h.birth(b)
	if ba == nil || bb == nil {
		return nil
	}
	if ba.idx > bb.idx {
		return ba
	}
	return bb
}

// origin says how the pair came to exist. It is reported, never a verdict:
// a copy that was made deliberately and one made in a hurry start the same.
func (h *history) origin(a, b *life) string {
	ba, bb := h.birth(a), h.birth(b)
	switch {
	case ba == nil || bb == nil:
		return "unknown"
	case ba == bb:
		return "born together in " + short(ba)
	}
	older, newer, c := a, b, bb
	if ba.idx > bb.idx {
		older, newer, c = b, a, ba
	}
	if s := sim(h.state(older, c.sha), h.state(newer, c.sha)); s >= copySim {
		return fmt.Sprintf("%s began as a copy of %s (%.2f) in %s", newer.key, older.key, s, short(c))
	}
	return "written separately"
}

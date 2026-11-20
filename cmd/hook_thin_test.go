package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The session-start tree: one notifier method that sends its own message, a
// helper it calls, and enough unrelated code for the corpus weights to carry
// information.
const thinPagerBefore = `package notify

import (
	"context"
	"fmt"
	"strings"
)

type Pager struct {
	hook   string
	client poster
}

type poster interface {
	Post(ctx context.Context, hook, text string) error
}

// ReportOutage pages an outage.
func (s *Pager) ReportOutage(ctx context.Context, text string) error {
	if s.hook == "" {
		return nil
	}
	if err := s.sendWithRetry(ctx, strings.TrimSpace(text)); err != nil {
		return fmt.Errorf("failed to send outage alert: %w", err)
	}
	return nil
}

func (s *Pager) sendWithRetry(ctx context.Context, text string) error {
	var err error
	for i := 0; i < 3; i++ {
		if err = s.client.Post(ctx, s.hook, text); err == nil {
			return nil
		}
	}
	return err
}
`

// The session's edit: a second notifier arrives, and both are reduced to one
// line over a shared helper, differing only in the literal that names the
// message. This is the shape a real session's Stop note reported as a new
// merge-worthy pair at code-shape 1.00 — after the merge had been done.
const thinPagerAfter = `package notify

import (
	"context"
	"fmt"
	"strings"
)

type Pager struct {
	hook   string
	client poster
}

type poster interface {
	Post(ctx context.Context, hook, text string) error
}

// ReportOutage pages an outage.
func (s *Pager) ReportOutage(ctx context.Context, text string) error {
	return s.page(ctx, text, "outage alert")
}

// ReportRecovery pages that an outage is over.
func (s *Pager) ReportRecovery(ctx context.Context, text string) error {
	return s.page(ctx, text, "recovery note")
}

// page pages with retries; what names the message in the error.
func (s *Pager) page(ctx context.Context, text, what string) error {
	if s.hook == "" {
		return nil
	}
	if err := s.sendWithRetry(ctx, strings.TrimSpace(text)); err != nil {
		return fmt.Errorf("failed to send %s: %w", what, err)
	}
	return nil
}

func (s *Pager) sendWithRetry(ctx context.Context, text string) error {
	var err error
	for i := 0; i < 3; i++ {
		if err = s.client.Post(ctx, s.hook, text); err == nil {
			return nil
		}
	}
	return err
}
`

const thinOtherBefore = `package other

import (
	"errors"
	"sort"
	"strconv"
	"strings"
)

func ParseList(s string) ([]int, error) {
	var out []int
	for _, f := range strings.Split(s, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		n, err := strconv.Atoi(f)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func Max(xs []int) (int, error) {
	if len(xs) == 0 {
		return 0, errors.New("empty")
	}
	m := xs[0]
	for _, x := range xs[1:] {
		if x > m {
			m = x
		}
	}
	return m, nil
}

func SortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func Count(s, sub string) int {
	n := 0
	for {
		i := strings.Index(s, sub)
		if i < 0 {
			return n
		}
		n++
		s = s[i+len(sub):]
	}
}
`

// The control: the same session also pastes a verbatim copy of an existing
// function, which is a merge the agent note must still name.
const thinOtherAfter = thinOtherBefore + `
func Occurrences(s, sub string) int {
	n := 0
	for {
		i := strings.Index(s, sub)
		if i < 0 {
			return n
		}
		n++
		s = s[i+len(sub):]
	}
}
`

// TestStopHookLeavesOutConsolidatedWrappers drives the hooks through a
// consolidation: two notifier methods end the session as one-line calls into
// one shared helper, differing in a string literal. The pair is new,
// attributable, merge-worthy and reads code-shape 1.00, so only its kind keeps
// it out of the agent note. The real case this pins was reported by a binary
// from before snapshots carried a kind (schema 12), and an agent spent a turn
// deferring a merge that was already done.
//
// The user digest still lists the pair, kind attached: it costs no turn. A
// verbatim copy made in the same session is the control, so the test cannot
// pass on an agent note that is empty for some other reason.
func TestStopHookLeavesOutConsolidatedWrappers(t *testing.T) {
	dir := t.TempDir()
	writeCorpus(t, dir, "notify/pager.go", thinPagerBefore)
	writeCorpus(t, dir, "other/other.go", thinOtherBefore)

	sessionID := "doppel-thin-wrappers-" + filepath.Base(dir)
	baseline := baselinePath(sessionID)
	t.Cleanup(func() {
		os.Remove(baseline)
		os.Remove(deltaPathFor(baseline))
		os.Remove(reviewPathFor(deltaPathFor(baseline)))
	})

	payload := hookPayload(t, sessionID, dir)
	if _, _, errOut := runHook(t, "session-start", payload); errOut != "" {
		t.Fatalf("session-start wrote to stderr: %q", errOut)
	}
	writeCorpus(t, dir, "notify/pager.go", thinPagerAfter)
	writeCorpus(t, dir, "other/other.go", thinOtherAfter)

	resp, _, errOut := runHook(t, "stop", payload)
	if errOut != "" {
		t.Fatalf("stop wrote to stderr: %q", errOut)
	}

	const wrappers = "notify.*Pager.ReportOutage <-> notify.*Pager.ReportRecovery"
	msg := resp.SystemMessage
	line := ""
	for l := range strings.SplitSeq(msg, "\n") {
		if strings.Contains(l, "NEW  "+wrappers) {
			line = l
		}
	}
	if line == "" {
		t.Fatalf("the user digest must list the wrapper pair as new:\n%s", msg)
	}
	if !strings.Contains(line, "shape 1.00") || !strings.Contains(line, "(merge-worthy)  [thin wrappers]") {
		t.Errorf("the wrapper pair should read shape 1.00, merge-worthy, thin wrappers:\n%s", line)
	}

	note := resp.HookSpecificOutput.AdditionalContext
	if !strings.Contains(note, "other.Count") || !strings.Contains(note, "other.Occurrences") {
		t.Fatalf("the agent note must still name the verbatim copy:\n%s", note)
	}
	if strings.Contains(note, "ReportRecovery") {
		t.Errorf("the agent note names wrappers that already share one helper:\n%s", note)
	}
}

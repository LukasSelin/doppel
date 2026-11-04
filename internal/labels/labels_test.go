package labels

import (
	"strings"
	"testing"
)

func parseOne(t *testing.T, label string) (File, error) {
	t.Helper()
	return Parse([]byte(`{"corpus":"x","reviewed":"2026-01-01","labels":[` + label + `]}`))
}

// TestDecisionAgreesWithClass pins the rules that keep a recorded decision
// honest: a merged pair is a merge or a refactor and was not blocked, and a
// real duplicate left alone says why in a word a scorecard can group by.
func TestDecisionAgreesWithClass(t *testing.T) {
	cases := []struct {
		name, label, wantErr string
	}{
		{"merged merge", `{"a":"p.A","b":"p.B","class":"merge","note":"",
			"decision":{"outcome":"merged","by":"agent","date":"2026-11-04","commit":"18a0af3","reason":"one helper now"}}`, ""},
		{"merged refactor with target", `{"a":"p.A","b":"p.B","class":"refactor","note":"","target":"shared helper in p",
			"decision":{"outcome":"merged","by":"user","date":"2026-11-04","reason":"extracted"}}`, ""},
		{"left false positive", `{"a":"p.A","b":"p.B","class":"false_positive","kind":"mirror","note":"",
			"decision":{"outcome":"left","by":"agent","date":"2026-11-04","reason":"inverse operations"}}`, ""},
		{"left merge with blocker", `{"a":"p.A","b":"q.A","class":"merge","blocker":"layering","note":"",
			"decision":{"outcome":"left","by":"agent","date":"2026-11-04","reason":"q may not import p"}}`, ""},
		{"deferred merge", `{"a":"p.A","b":"p.B","class":"merge","note":"",
			"decision":{"outcome":"deferred","by":"user","date":"2026-11-04","reason":"after the release"}}`, ""},
		{"merged false positive", `{"a":"p.A","b":"p.B","class":"false_positive","note":"",
			"decision":{"outcome":"merged","by":"agent","date":"2026-11-04","reason":"x"}}`, "merged pair is a merge or a refactor"},
		{"merged but blocked", `{"a":"p.A","b":"p.B","class":"merge","blocker":"no-home","note":"",
			"decision":{"outcome":"merged","by":"agent","date":"2026-11-04","reason":"x"}}`, "was not blocked"},
		{"left merge without blocker", `{"a":"p.A","b":"p.B","class":"merge","note":"",
			"decision":{"outcome":"left","by":"agent","date":"2026-11-04","reason":"x"}}`, "needs a blocker"},
		{"blocker on false positive", `{"a":"p.A","b":"p.B","class":"false_positive","blocker":"design","note":""}`, "blocker"},
		{"unknown blocker", `{"a":"p.A","b":"p.B","class":"merge","blocker":"lazy","note":""}`, "invalid blocker"},
		{"unknown outcome", `{"a":"p.A","b":"p.B","class":"merge","note":"",
			"decision":{"outcome":"maybe","by":"agent","date":"2026-11-04","reason":"x"}}`, "invalid outcome"},
		{"unknown decider", `{"a":"p.A","b":"p.B","class":"merge","note":"",
			"decision":{"outcome":"merged","by":"bot","date":"2026-11-04","reason":"x"}}`, "invalid decider"},
		{"bad date", `{"a":"p.A","b":"p.B","class":"merge","note":"",
			"decision":{"outcome":"merged","by":"agent","date":"4 Nov","reason":"x"}}`, "YYYY-MM-DD"},
		{"no reason", `{"a":"p.A","b":"p.B","class":"merge","note":"",
			"decision":{"outcome":"merged","by":"agent","date":"2026-11-04","reason":" "}}`, "needs a reason"},
		{"bad commit", `{"a":"p.A","b":"p.B","class":"merge","note":"",
			"decision":{"outcome":"merged","by":"agent","date":"2026-11-04","commit":"HEAD","reason":"x"}}`, "hexadecimal"},
		{"bad earlier", `{"a":"p.A","b":"p.B","class":"merge","note":"",
			"earlier":[{"outcome":"left","by":"agent","date":"2026-11-04","reason":""}]}`, "earlier decision"},
	}
	for _, c := range cases {
		_, err := parseOne(t, c.label)
		switch {
		case c.wantErr == "" && err != nil:
			t.Errorf("%s: rejected: %v", c.name, err)
		case c.wantErr != "" && err == nil:
			t.Errorf("%s: accepted, want an error containing %q", c.name, c.wantErr)
		case c.wantErr != "" && !strings.Contains(err.Error(), c.wantErr):
			t.Errorf("%s: error %q, want it to contain %q", c.name, err, c.wantErr)
		}
	}
}

// TestEffectiveWeight pins how a decision turns into a weight: up for a merge
// somebody carried out, down for one somebody left, never down for a false
// positive somebody left (that is a confirmation), and an explicit weight
// always wins.
func TestEffectiveWeight(t *testing.T) {
	d := func(outcome string) *Decision {
		return &Decision{Outcome: outcome, By: "agent", Date: "2026-11-04", Reason: "r"}
	}
	cases := []struct {
		name string
		l    Label
		want float64
	}{
		{"plain", Label{Class: "merge"}, 1},
		{"merged", Label{Class: "merge", Decision: d("merged")}, MergedWeight},
		{"merged refactor", Label{Class: "refactor", Decision: d("merged")}, MergedWeight},
		{"left merge", Label{Class: "merge", Blocker: "design", Decision: d("left")}, LeftWeight},
		{"deferred", Label{Class: "merge", Decision: d("deferred")}, 1},
		{"left false positive", Label{Class: "false_positive", Decision: d("left")}, 1},
		{"explicit wins", Label{Class: "merge", Weight: 2, Decision: d("merged")}, 2},
	}
	for _, c := range cases {
		if got := c.l.EffectiveWeight(); got != c.want {
			t.Errorf("%s: weight %g, want %g", c.name, got, c.want)
		}
	}
}

// TestTreeRoundTrips checks the file-level commit survives a parse.
func TestTreeRoundTrips(t *testing.T) {
	lf, err := Parse([]byte(`{"corpus":"x","reviewed":"2026-01-01","tree":"ebcfa1e","labels":[
		{"a":"p.A","b":"p.B","class":"merge","note":"n"}]}`))
	if err != nil || lf.Tree != "ebcfa1e" {
		t.Fatalf("tree lost: %v %q", err, lf.Tree)
	}
}

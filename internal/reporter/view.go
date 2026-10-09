package reporter

import (
	"strings"

	"github.com/LukasSelin/doppel/internal/identity"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// SessionView is the user-side form of the Stop hook's measurement: one line
// for the band above the prompt and the full delta report for the pane behind
// it, both rendered here so the plugin's mod draws text it never composes.
//
// It exists because of a constraint the Stop hook cannot get around: text a
// Stop hook puts in front of the model continues the turn. A mod draws in the
// UI and never reaches the model's context, so it is a channel to the user that
// costs no token and no turn — which is why it can afford to show everything
// the session did, where the agent note shows only what clears Notable.
//
// Nothing here is a second rendering. The band is deltaScoreboard, the line
// that heads DeltaSection; the report is identity.PrintDelta, the text `doppel
// diff` prints. One rendering, three surfaces.
type SessionView struct {
	Band   string `json:"band"`
	Report string `json:"report"`
}

// SessionViewOf renders the view, or false exactly when SessionDigest would
// return "" — the band is silent when the user digest is, so the two user-side
// surfaces can never disagree about whether the session has done anything.
//
// The identity delta is preferred because it attributes; when it is
// unavailable (identityDelta degraded to its zero value) the view falls back to
// the impact half, untruncated, rather than showing a "not comparable" line for
// a comparison that snapshot.Diff did make.
func SessionViewOf(id identity.Delta, d snapshot.Delta) (SessionView, bool) {
	impact := impactBody(d, "")
	if impact == "" || !d.Comparable {
		return SessionView{}, false
	}
	if id.Comparable && !id.Empty() {
		var b strings.Builder
		identity.PrintDelta(&b, id, false)
		return SessionView{
			Band:   "doppel: " + deltaScoreboard(id) + " — since session start",
			Report: b.String(),
		}, true
	}
	return SessionView{
		Band:   "doppel: " + strings.Join(scoreboard(d), ", ") + " — since session start",
		Report: impact,
	}, true
}

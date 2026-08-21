package reporter

import (
	"encoding/json"
	"io"

	"github.com/LukasSelin/doppel/internal/snapshot"
)

// PrintJSON writes a snapshot as JSON.
//
// Every JSON write in the tool goes through encodeJSON, and that is
// deliberate: a baseline written by the hook and a report written by
// `analyze --format json` must be byte-comparable, which they are not if the
// two paths configure their encoders differently.
func PrintJSON(w io.Writer, s snapshot.Snapshot) error {
	return encodeJSON(w, s)
}

// PrintDeltaJSON writes an impact delta as JSON.
func PrintDeltaJSON(w io.Writer, d snapshot.Delta) error {
	return encodeJSON(w, d)
}

// encodeJSON is the single JSON encoder configuration.
//
// HTML escaping is off because evidence reason strings contain characters the
// default encoder would rewrite (`<`, `>`, `&` appear in role and pair
// descriptions), turning readable output into escape sequences for a browser
// context that does not exist here.
func encodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

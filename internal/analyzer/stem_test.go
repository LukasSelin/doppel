package analyzer

import "testing"

// The stem rule, pinned on the names that motivated it and the near-misses
// that must stay apart.
func TestStemPair(t *testing.T) {
	cases := []struct {
		x, y string
		stem string
		ok   bool
	}{
		{"evalCallOld", "evalCall", "evalCall", true},
		{"scrapeLoopAppenderV2", "scrapeLoopAppender", "scrapeLoopAppender", true},
		{"appendV1", "appendV2", "append", true}, // v-prefixed runs are versions, not numbering
		{"foo_old", "foo", "foo", true},
		{"NewClient", "Client", "Client", true},
		{"handler2", "handler", "handler", true},
		{"LegacyParse", "Parse", "Parse", true},
		{"decodeToml", "decodeYAML", "", false}, // no marker: stems are the names
		{"loadWAL", "loadWBL", "", false},
		{"sha256", "sha512", "", false}, // numbered variants, guard one
		{"utf8", "utf16", "", false},
		{"Threshold", "Thresh", "", false}, // lower-case "old" is not a marker
		{"Newton", "ton", "", false},
		{"newline", "line", "", false},
		{"oldest", "est", "", false},
		{"GoOld", "Go", "", false}, // remainder below minStem
		{"same", "same", "", false},
	}
	for _, tc := range cases {
		stem, ok := stemPair(tc.x, tc.y)
		if ok != tc.ok || stem != tc.stem {
			t.Errorf("stemPair(%q, %q) = (%q, %v), want (%q, %v)", tc.x, tc.y, stem, ok, tc.stem, tc.ok)
		}
		if rstem, rok := stemPair(tc.y, tc.x); rok != ok || rstem != stem {
			t.Errorf("stemPair is asymmetric on (%q, %q)", tc.x, tc.y)
		}
	}
}

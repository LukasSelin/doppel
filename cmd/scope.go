package cmd

import (
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/LukasSelin/doppel/internal/reporter"
	"github.com/LukasSelin/doppel/internal/snapshot"
)

// scopeMaxPackages bounds how many mentioned packages one prompt can scope.
// A prompt that names ten packages is a survey, not a target, and the digest
// must not become the corpus dump it exists to replace.
const scopeMaxPackages = 3

// harnessEnvelopes match the tags Claude Code wraps around text the user did
// not type: a background task finishing, a subagent's or another session's
// message, a system reminder. UserPromptSubmit fires on those deliveries just
// as it does on the user's own words. They made up 17% of this hook's firings
// over a month of sessions, and none of those firings led anywhere: the text
// inside names packages because the reported work lives there, not because
// anyone is about to start on them. An envelope with no closing tag runs to
// the end of the prompt, since everything after its opening tag is still the
// harness talking.
var harnessEnvelopes = func() []*regexp.Regexp {
	var out []*regexp.Regexp
	for _, tag := range []string{"task-notification", "system-reminder", "agent-message", "cross-session-message"} {
		out = append(out, regexp.MustCompile(`(?s)<`+tag+`\b[^>]*>(?:.*?</`+tag+`>|.*$)`))
	}
	return out
}()

// typedText is the part of a prompt the user typed: the prompt with every
// harness envelope cut out.
func typedText(prompt string) string {
	for _, re := range harnessEnvelopes {
		prompt = re.ReplaceAllString(prompt, " ")
	}
	return prompt
}

// mention is one path-shaped token of a prompt: as written, for the digest's
// header, and slash-normalized, for matching.
type mention struct {
	raw  string
	norm string
}

// pathMentions returns the path-shaped tokens of the part of a prompt the
// user typed, in order: a token containing '/' or '\', with an optional @
// and surrounding punctuation stripped, that is not a URL.
//
// A bare word is never a mention, though it was until the hook was measured.
// Package names are ordinary words: across a month of sessions this hook fired
// 88 times on "git" and 62 on "main", alongside "verify", "web", "agent" and
// "email", and none of its notes led to a merge. A path is what someone types
// when they mean a place in the tree.
func pathMentions(prompt string) []mention {
	var out []mention
	for _, raw := range strings.Fields(typedText(prompt)) {
		tok := trimMention(raw)
		if !strings.ContainsAny(tok, "/\\") || strings.Contains(tok, "://") {
			continue
		}
		norm := strings.Trim(path.Clean(strings.ReplaceAll(tok, "\\", "/")), "/")
		if norm == "" || norm == "." {
			continue
		}
		out = append(out, mention{raw: tok, norm: norm})
	}
	return out
}

// scopedPackages resolves a prompt's path mentions against a snapshot: a
// mention scopes every package whose directory it names, from the repo root or
// from deeper up the tree, or whose directory holds the file it names. The
// corpus confirms each one, so a path that names nothing analyzed is prose.
// First-mention order is kept, capped at scopeMaxPackages, one entry per
// package even when mentioned twice.
func scopedPackages(prompt string, s snapshot.Snapshot) []reporter.ScopedPackage {
	mentions := pathMentions(prompt)
	if len(mentions) == 0 || len(s.Units) == 0 {
		return nil
	}

	// Directory → package, from the snapshot, so the matcher confirms
	// mentions against what was analyzed rather than against the filesystem.
	dirPkg := make(map[string]string)
	for _, u := range s.Units {
		if u.Package != "" {
			dirPkg[path.Dir(u.File)] = u.Package
		}
	}

	seen := make(map[string]bool)
	var out []reporter.ScopedPackage
	for _, m := range mentions {
		// Matches are collected and sorted before adding — dirPkg is a map,
		// and map order deciding which package wins the cap would make the
		// digest differ between runs of the same prompt.
		var hits []string
		for dir, pkg := range dirPkg {
			// Four alignments: the mention names the directory exactly,
			// names it from deeper up the tree (suffix), names a parent
			// of it (prefix), or names a file inside it.
			if dir == m.norm || strings.HasSuffix(dir, "/"+m.norm) ||
				strings.HasPrefix(dir, m.norm+"/") || strings.HasPrefix(m.norm, dir+"/") {
				hits = append(hits, pkg)
			}
		}
		sort.Strings(hits)
		for _, pkg := range hits {
			if seen[pkg] || len(out) >= scopeMaxPackages {
				continue
			}
			seen[pkg] = true
			out = append(out, reporter.ScopedPackage{Package: pkg, Mention: m.raw})
		}
	}
	return out
}

// trimMention strips the decoration a prompt wraps around a path: a leading
// @ (file mentions), and surrounding quotes, brackets and trailing prose
// punctuation ("internal/culture," or "(svc/billing)").
func trimMention(tok string) string {
	tok = strings.TrimPrefix(tok, "@")
	tok = strings.Trim(tok, "\"'`([{")
	return strings.TrimRight(tok, ".,;:!?)]}\"'`")
}

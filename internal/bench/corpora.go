package bench

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Corpus is one pinned public Go repository in the reference ladder.
//
// The ladder is ordered old-and-complex first, new-and-narrow last: an
// aging container engine whose packages have accreted for a decade, down to
// a single-purpose library written in one sitting. Nothing about a corpus is
// committed except its coordinates — the trees themselves are fetched on
// demand into Root() and never enter this repository.
type Corpus struct {
	Name   string // directory name under Root(), and the example report's basename
	Repo   string // clone URL
	Tag    string // pinned release tag — the fetch is --depth 1 --branch Tag
	Commit string // the tag's *peeled* commit (annotated tags point at a tag object,
	// not a commit), verified after fetch so a moved tag is loud
	Since     int    // year the project started: the "old ... new" axis
	Character string // what kind of codebase this is
	Exercises string // what it stresses in doppel
}

// Corpora is the ladder, ordered old-and-complex to new-and-narrow.
//
// Choosing public, pinned, widely-read Go projects is deliberate: an example
// report is only useful if the reader can open the two functions it names.
var Corpora = []Corpus{
	{
		Name:      "moby",
		Repo:      "https://github.com/moby/moby.git",
		Tag:       "v28.5.2",
		Commit:    "89c5e8fd66634b6128fc4c0e6f1236e2540e46e0",
		Since:     2013,
		Character: "container engine; a decade of accretion across daemon, API, and plugin layers",
		Exercises: "scale, df caps, and the common-idiom suppression the retrieval channels exist for",
	},
	{
		Name:      "prometheus",
		Repo:      "https://github.com/prometheus/prometheus.git",
		Tag:       "v3.14.0",
		Commit:    "d7598b7141418fa35be2b5ec5d0fefb634199610",
		Since:     2012,
		Character: "monitoring system; storage engine, query language, and scrape pipeline in one tree",
		Exercises: "deep call graphs and role classification on a genuinely layered corpus",
	},
	{
		Name:      "hugo",
		Repo:      "https://github.com/gohugoio/hugo.git",
		Tag:       "v0.165.0",
		Commit:    "76a5e1880ab46688155b02e99bab9be2a6134492",
		Since:     2013,
		Character: "static site generator; a large monolith with heavy template and resource subsystems",
		Exercises: "habitats — many packages large enough to have a temperature of their own",
	},
	{
		Name:      "gin",
		Repo:      "https://github.com/gin-gonic/gin.git",
		Tag:       "v1.12.0",
		Commit:    "73726dc606796a025971fe451f0aa6f1b9b847f6",
		Since:     2014,
		Character: "HTTP framework; a small core surrounded by generated-looking binding and render variants",
		Exercises: "family clones — the case corroborated ranking was tuned to separate",
	},
	{
		Name:      "cobra",
		Repo:      "https://github.com/spf13/cobra.git",
		Tag:       "v1.10.2",
		Commit:    "88b30ab89da2d0d0abb153818746c5a2d30eccec",
		Since:     2015,
		Character: "CLI framework; one dominant type with a long method set, plus shell-completion generators",
		Exercises: "the receiver and role signals, and per-shell generator siblings",
	},
	{
		Name:      "chi",
		Repo:      "https://github.com/go-chi/chi.git",
		Tag:       "v5.3.2",
		Commit:    "38939062c5df4d3e8814aad1a488983112627ced",
		Since:     2015,
		Character: "HTTP router; a narrow core with a middleware package beside it",
		Exercises: "a corpus small enough to read end to end, where every reported pair can be checked",
	},
	{
		Name:      "conc",
		Repo:      "https://github.com/sourcegraph/conc.git",
		Tag:       "v0.3.0",
		Commit:    "7b8c8f2875cb861bb61844c9bcaa1aed070adbd4",
		Since:     2023,
		Character: "structured concurrency library; generics-heavy, one idea, written recently and at once",
		Exercises: "the small-corpus floor: 85 functions, where IC and df caps have almost nothing to work with",
	},
}

// Find returns the named corpus from the ladder.
func Find(name string) (Corpus, bool) {
	for _, c := range Corpora {
		if c.Name == name {
			return c, true
		}
	}
	return Corpus{}, false
}

// Root is where fetched corpora live: $DOPPEL_CORPORA, else a directory in
// the user cache. Deliberately outside the repository — a tree under the
// working directory would be walked by `doppel analyze .` on this repo.
//
// On Windows the path length matters: several of these corpora carry test
// fixtures with names long enough to break a checkout under a deep root.
func Root() (string, error) {
	if r := os.Getenv("DOPPEL_CORPORA"); r != "" {
		return r, nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("no cache dir and DOPPEL_CORPORA unset: %w", err)
	}
	return filepath.Join(cache, "doppel-corpora"), nil
}

// Path is where this corpus is or would be checked out.
func Path(c Corpus) (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, c.Name), nil
}

// Present reports whether the corpus is already checked out.
func Present(c Corpus) bool {
	p, err := Path(c)
	if err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(p, ".git"))
	return err == nil && info.IsDir()
}

// Fetch shallow-clones the corpus at its pinned tag if it is not already
// present, then verifies the checked-out commit against the manifest. A tag
// that has been moved upstream fails here rather than silently changing
// every number in the example reports.
func Fetch(c Corpus) (string, error) {
	dir, err := Path(c)
	if err != nil {
		return "", err
	}
	if !Present(c) {
		if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
			return "", err
		}
		// core.longpaths is a no-op off Windows and the difference between a
		// working checkout and a failed one on it.
		clone := exec.Command("git", "-c", "core.longpaths=true", "clone",
			"--depth", "1", "--branch", c.Tag, "--quiet", c.Repo, dir)
		clone.Stderr = os.Stderr
		if err := clone.Run(); err != nil {
			return "", fmt.Errorf("clone %s at %s: %w", c.Repo, c.Tag, err)
		}
	}
	head, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse %s: %w", dir, err)
	}
	if got := strings.TrimSpace(string(head)); got != c.Commit {
		return "", fmt.Errorf("%s: checked out %s, manifest pins %s (moved tag, or a stale checkout — delete %s and refetch)",
			c.Name, got, c.Commit, dir)
	}
	return dir, nil
}

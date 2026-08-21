package bench

import (
	"os"
	"regexp"
	"testing"
)

var sha1Hex = regexp.MustCompile(`^[0-9a-f]{40}$`)

// The manifest is data, and data that nobody checks rots. These assertions
// run offline in CI: they cost nothing and catch the copy-paste mistakes that
// would otherwise surface as a mysterious clone failure months later.
func TestCorporaManifest(t *testing.T) {
	if len(Corpora) < 2 {
		t.Fatal("the ladder needs at least two rungs to be a ladder")
	}
	seen := map[string]bool{}
	for _, c := range Corpora {
		if c.Name == "" || c.Repo == "" || c.Tag == "" {
			t.Errorf("%+v: name, repo and tag are all required", c)
		}
		if seen[c.Name] {
			t.Errorf("%s: duplicate corpus name", c.Name)
		}
		seen[c.Name] = true
		if !sha1Hex.MatchString(c.Commit) {
			t.Errorf("%s: commit %q is not a full 40-hex sha", c.Name, c.Commit)
		}
		if c.Since < 2007 || c.Since > 2100 {
			t.Errorf("%s: implausible start year %d (Go was announced in 2009)", c.Name, c.Since)
		}
		if c.Character == "" || c.Exercises == "" {
			t.Errorf("%s: a rung with no explanation is not worth fetching", c.Name)
		}
		if _, ok := Find(c.Name); !ok {
			t.Errorf("%s: Find cannot find a corpus that is in the ladder", c.Name)
		}
	}
	if _, ok := Find("no-such-corpus"); ok {
		t.Error("Find invented a corpus")
	}
}

func TestRootHonoursEnv(t *testing.T) {
	t.Setenv("DOPPEL_CORPORA", "/tmp/somewhere")
	got, err := Root()
	if err != nil || got != "/tmp/somewhere" {
		t.Fatalf("Root() = %q, %v; want the env override", got, err)
	}
	t.Setenv("DOPPEL_CORPORA", "")
	if _, err := Root(); err != nil {
		t.Fatalf("Root() with no override: %v", err)
	}
}

// TestFetchCorpora is the one network-touching test in the module. It is
// guarded rather than skipped-by-detection because a surprise 300 MB clone in
// somebody's `go test ./...` is a bad day.
func TestFetchCorpora(t *testing.T) {
	if os.Getenv("DOPPEL_BENCH_FETCH") == "" {
		t.Skip("set DOPPEL_BENCH_FETCH=1 to clone the reference corpora (several hundred MB)")
	}
	for _, c := range Corpora {
		t.Run(c.Name, func(t *testing.T) {
			dir, err := Fetch(c)
			if err != nil {
				t.Fatalf("fetch %s: %v", c.Name, err)
			}
			units, err := Load(dir, PopExclude)
			if err != nil {
				t.Fatalf("load %s: %v", c.Name, err)
			}
			if len(units) == 0 {
				t.Fatalf("%s: no Go functions found at %s", c.Name, dir)
			}
			t.Logf("%s %s: %d production functions", c.Name, c.Tag, len(units))
		})
	}
}

package bench

import (
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestGeneratePagesSite renders the published site's examples section: one
// interactive dashboard per rung of the ladder, plus the index that lists
// them.
//
// It belongs here rather than in cmd for the same reason the Markdown example
// generator does — the corpus manifest lives in this package, and the site
// must describe exactly the ladder the manifest pins. DOPPEL_BENCH_PAGES
// names the site root (`_site`), so the workflow and `task pages` drive the
// same code.
//
// Unlike TestGenerateExamples this **fails** on a corpus that is not fetched.
// A partially built ladder is fine as a local measurement and wrong as a
// published page: the site would quietly claim the ladder is four rungs long.
func TestGeneratePagesSite(t *testing.T) {
	outRoot := os.Getenv("DOPPEL_BENCH_PAGES")
	if outRoot == "" {
		t.Skip("set DOPPEL_BENCH_PAGES=<site root> to render the published examples section")
	}
	root := repoRoot(t)
	// A test binary runs in its own package directory, so a relative site
	// root would land under internal/bench. Both callers — the workflow and
	// `task pages` — mean it relative to the module.
	if !filepath.IsAbs(outRoot) {
		outRoot = filepath.Join(root, outRoot)
	}
	bin := filepath.Join(t.TempDir(), "doppel")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = root
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("build doppel: %v", err)
	}

	outDir := filepath.Join(outRoot, "examples")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var cards strings.Builder
	for _, c := range Corpora {
		if !Present(c) {
			t.Fatalf("%s is not fetched; the published ladder must be complete — run `task corpora`", c.Name)
		}
		// The numbers on the card come from the committed Markdown report
		// rather than from a second analysis: one run per corpus, and the
		// card cannot disagree with the report it links to.
		body, err := os.ReadFile(filepath.Join(root, "examples", c.Name+".md"))
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		diag, err := reportDiagnostics(string(normalizeEOL(body)))
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		row, err := parseDiagnostics(c, diag)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}

		// The page size is on the card because these are honest numbers a
		// reader on a phone wants before tapping: moby's dashboard inlines
		// every body it can afford and is megabytes.
		var size int64
		t.Run(c.Name, func(t *testing.T) {
			dir, err := Path(c)
			if err != nil {
				t.Fatal(err)
			}
			dst := filepath.Join(outDir, c.Name+".html")
			// The .html extension is what selects the dashboard; the same
			// command with a .md path writes the committed report.
			cmd := exec.Command(bin, "analyze", ".", "--tests", "exclude",
				"--top", fmt.Sprint(exampleTop), "--output", dst)
			cmd.Dir = dir
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("analyze %s: %v", c.Name, err)
			}
			info, err := os.Stat(dst)
			if err != nil {
				t.Fatal(err)
			}
			size = info.Size()
			t.Logf("wrote %s (%d KB)", dst, size/1024)
		})
		cards.WriteString(renderCard(row, size))
	}

	shell, err := os.ReadFile(filepath.Join(root, ".github", "pages", "examples.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := strings.Replace(string(normalizeEOL(shell)), "@@CARDS@@", strings.TrimRight(cards.String(), "\n"), 1)
	if strings.Contains(page, "@@CARDS@@") || !strings.Contains(page, "@@COMMIT@@") {
		// @@COMMIT@@ and @@BUILT@@ are the workflow's to substitute, exactly
		// as on the landing page. Losing them here would publish the
		// placeholders.
		t.Fatal(".github/pages/examples.html: expected one @@CARDS@@ slot and the provenance placeholders")
	}
	index := filepath.Join(outDir, "index.html")
	if err := os.WriteFile(index, []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", index)
}

// renderCard is one corpus's entry on the examples index. The card is a div
// rather than a link because it carries three destinations — the dashboard,
// the Markdown report and the pinned source — and an anchor cannot nest.
func renderCard(r ladderRow, size int64) string {
	c := r.Corpus
	esc := html.EscapeString
	var b strings.Builder
	fmt.Fprintf(&b, "      <div class=\"card\">\n")
	fmt.Fprintf(&b, "        <div class=\"card-kicker\">%d · %s</div>\n", c.Since, esc(c.Tag))
	fmt.Fprintf(&b, "        <div class=\"card-title\"><a href=\"%s.html\">%s</a></div>\n", esc(c.Name), esc(c.Name))
	fmt.Fprintf(&b, "        <p class=\"card-body\">%s</p>\n", esc(c.Character))
	fmt.Fprintf(&b, "        <div class=\"card-meta\">%d functions · %d pairs compared · %d kept · code-shape floor %s · %s</div>\n",
		r.Funcs, r.Pairs, r.Kept, esc(r.Floor), pageSize(size))
	fmt.Fprintf(&b, "        <div class=\"card-meta\">"+
		"<a href=\"https://github.com/LukasSelin/doppel/blob/master/examples/%s.md\">Markdown report</a> · "+
		"<a href=\"%s\">source at %s</a></div>\n",
		esc(c.Name), esc(strings.TrimSuffix(c.Repo, ".git")), esc(c.Tag))
	fmt.Fprintf(&b, "      </div>\n")
	return b.String()
}

// pageSize renders a dashboard's weight the way a reader deciding whether to
// tap it would want it: one significant figure, and never a byte count.
func pageSize(n int64) string {
	if n >= 1<<20 {
		return fmt.Sprintf("%.0f MB page", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%d KB page", n/1024)
}

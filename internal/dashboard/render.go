package dashboard

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

// assetsFS is the whole page: the shell, this package's own CSS and JS, and
// the vendored files. Embedded so a built binary carries the dashboard with it
// and a report needs nothing to open.
//
//go:embed assets
var assetsFS embed.FS

// DevAssetsEnv names a directory to read the assets from instead of the
// embedded copy.
//
// It exists for the iteration loop the old renderer did not have: with it set,
// editing app.js and re-running a prebuilt binary is the whole cycle. Nothing
// ships depending on it — the embedded copy is the only path a release takes,
// and tests must leave it unset.
const DevAssetsEnv = "DOPPEL_DASHBOARD_ASSETS"

// The asset names, relative to the assets root, in the order they are inlined.
const (
	shellAsset    = "shell.html"
	broadsheetCSS = "vendor/broadsheet.css"
	appCSS        = "app.css"
	appJS         = "app.js"
)

// source is where assets are read from: a filesystem plus the prefix the
// assets root sits at within it.
//
// The prefix is what lets the embedded copy (rooted at "assets/") and a dev
// directory (which *is* the assets root) answer the same names.
type source struct {
	fsys   fs.FS
	prefix string
}

// view is what the shell template sees.
//
// The typed fields are typed so html/template inlines them rather than
// escaping them as text. That is safe for each for a different reason: the CSS
// and the JS are files in this repo, not analysed source, and Data is JSON
// marshalled with HTML escaping on — so `<`, `>` and `&` reach the page as
// <, > and &, and no function name or body can close the script
// element that carries it.
type view struct {
	Target string
	CSS    template.CSS
	Data   template.JS
	Script template.JS
}

// Print writes the payload as one self-contained HTML document.
//
// There is no vendored JavaScript. The map is a power diagram drawn as plain
// SVG and the neighbourhood screen is DOM, so the graph library this started
// with earned nothing but its own 373KB in every report written.
//
// Self-contained is the contract the previous report established and this one
// keeps: everything is inlined, there is no fetch, and the page opens from
// file://. The only external reference is the design system's font import,
// which falls back to system-ui offline.
func Print(w io.Writer, p Payload) error {
	src, err := assetSource()
	if err != nil {
		return err
	}

	shell, err := src.read(shellAsset)
	if err != nil {
		return err
	}
	css, err := src.readAll(broadsheetCSS, appCSS)
	if err != nil {
		return err
	}
	script, err := src.read(appJS)
	if err != nil {
		return err
	}

	data, err := marshalPayload(p)
	if err != nil {
		return err
	}

	tmpl, err := template.New("dashboard").Parse(shell)
	if err != nil {
		return fmt.Errorf("parse %s: %w", shellAsset, err)
	}
	return tmpl.Execute(w, view{
		Target: p.Target,
		CSS:    template.CSS(css),
		Data:   template.JS(data),
		Script: template.JS(script),
	})
}

// marshalPayload encodes the payload for embedding in a script element.
//
// HTML escaping is left ON, which is the opposite of reporter's snapshot
// encoder and deliberately so: that one turns it off to keep a baseline
// byte-comparable, and this one needs a `</script>` inside an analysed function
// body to come out as </script> rather than ending the page.
func marshalPayload(p Payload) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(p); err != nil {
		return "", fmt.Errorf("encode payload: %w", err)
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// assetSource is the embedded copy, or the dev directory when one is named.
func assetSource() (source, error) {
	dir := os.Getenv(DevAssetsEnv)
	if dir == "" {
		return source{fsys: assetsFS, prefix: "assets"}, nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		return source{}, fmt.Errorf("%s=%s: %w", DevAssetsEnv, dir, err)
	}
	if !info.IsDir() {
		return source{}, fmt.Errorf("%s=%s: not a directory", DevAssetsEnv, dir)
	}
	// The directory named *is* the assets root, so no prefix applies.
	return source{fsys: os.DirFS(dir)}, nil
}

func (s source) read(name string) (string, error) {
	full := name
	if s.prefix != "" {
		full = path.Join(s.prefix, name)
	}
	b, err := fs.ReadFile(s.fsys, full)
	if err != nil {
		return "", fmt.Errorf("read asset %s: %w", name, err)
	}
	// An asset that can close a script element would break every page it is
	// inlined into, and would do it silently. Checked rather than escaped:
	// these are files in this repo, so the fix is to edit the file. The shell
	// is exempt — it is the document, and its own script tags are the point.
	if name != shellAsset && strings.Contains(strings.ToLower(string(b)), "</script") {
		return "", fmt.Errorf("asset %s contains </script, which cannot be inlined", name)
	}
	return string(b), nil
}

func (s source) readAll(names ...string) (string, error) {
	var b strings.Builder
	for _, name := range names {
		text, err := s.read(name)
		if err != nil {
			return "", err
		}
		b.WriteString(text)
		b.WriteString("\n")
	}
	return b.String(), nil
}

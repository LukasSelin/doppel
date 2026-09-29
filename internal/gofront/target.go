package gofront

import (
	"go/ast"
	"go/build/constraint"
	"path/filepath"
	"strings"
)

// Targets is the set of ports a Go file can be compiled for, as a bitset
// over ports (see syntax.File.Targets for the contract). It reads the two
// things the toolchain itself reads — the GOOS/GOARCH filename suffix and the
// file's build constraint — and nothing else: no path shape, no name
// heuristic, the same refusal every other rule in the tool makes.
//
// Tags other than GOOS, GOARCH and unix (cgo, purego, go1.N, custom tags)
// are free: a port is in the set when *some* assignment of them satisfies
// the constraint there. That is the permissive reading, and it is chosen so
// the only error available is the old behaviour — two files that could never
// share a build but differ only by a custom tag (`amd64 && !purego` against
// `!amd64 || purego`) still both reach amd64 and are still paired.
func Targets(filename string, f *ast.File) uint64 {
	name, nameOK := fileNameConstraint(filepath.Base(filename))
	expr := buildExpr(f)
	if !nameOK && expr == nil {
		return 0
	}
	free := freeTags(expr)
	if len(free) > maxFreeTags {
		// Past this the enumeration stops being cheap, and a file this
		// conditional is rare enough that treating its constraint as
		// satisfiable everywhere costs nothing a reader would notice. The
		// filename alone still binds.
		expr, free = nil, nil
		if !nameOK {
			return 0
		}
	}
	var set uint64
	for i, p := range ports {
		if nameOK && !name(p) {
			continue
		}
		if expr == nil || satisfiable(expr, p, free) {
			set |= 1 << uint(i)
		}
	}
	if set == 0 {
		// Constrained, but to no port on the list: a known OS or arch with
		// no first-class port (mips64p32, sparc64), or an unsatisfiable
		// expression. It still builds somewhere, so it must not read as
		// unconstrained (0) — it gets the one bit no listed port uses.
		set = offList
	}
	return set
}

// maxFreeTags bounds the per-file enumeration at 2^8 assignments per port.
const maxFreeTags = 8

// offList is the target bit for "only on a port this table does not list".
// Two such files are treated as coexisting, which fails toward pairing them.
const offList = uint64(1) << 63

// port is one GOOS/GOARCH pair.
type port struct{ os, arch string }

// ports is `go tool dist list` for Go 1.25, in its own order. The bit a port
// occupies is its index here; nothing outside a run compares these bits, so
// the order may change freely between builds.
var ports = []port{
	{"aix", "ppc64"},
	{"android", "386"}, {"android", "amd64"}, {"android", "arm"}, {"android", "arm64"},
	{"darwin", "amd64"}, {"darwin", "arm64"},
	{"dragonfly", "amd64"},
	{"freebsd", "386"}, {"freebsd", "amd64"}, {"freebsd", "arm"}, {"freebsd", "arm64"}, {"freebsd", "riscv64"},
	{"illumos", "amd64"},
	{"ios", "amd64"}, {"ios", "arm64"},
	{"js", "wasm"},
	{"linux", "386"}, {"linux", "amd64"}, {"linux", "arm"}, {"linux", "arm64"}, {"linux", "loong64"},
	{"linux", "mips"}, {"linux", "mips64"}, {"linux", "mips64le"}, {"linux", "mipsle"},
	{"linux", "ppc64"}, {"linux", "ppc64le"}, {"linux", "riscv64"}, {"linux", "s390x"},
	{"netbsd", "386"}, {"netbsd", "amd64"}, {"netbsd", "arm"}, {"netbsd", "arm64"},
	{"openbsd", "386"}, {"openbsd", "amd64"}, {"openbsd", "arm"}, {"openbsd", "arm64"},
	{"openbsd", "ppc64"}, {"openbsd", "riscv64"},
	{"plan9", "386"}, {"plan9", "amd64"}, {"plan9", "arm"},
	{"solaris", "amd64"},
	{"wasip1", "wasm"},
	{"windows", "386"}, {"windows", "amd64"}, {"windows", "arm64"},
}

// knownOS, knownArch and unixOS mirror go/build's syslist.go, which the
// standard library does not export. A filename suffix is a constraint only
// when it names one of these, exactly as the toolchain decides.
var knownOS = set("aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos",
	"ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1",
	"windows", "zos")

var knownArch = set("386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be",
	"loong64", "mips", "mipsle", "mips64", "mips64le", "mips64p32", "mips64p32le",
	"ppc", "ppc64", "ppc64le", "riscv", "riscv64", "s390", "s390x", "sparc", "sparc64",
	"wasm")

var unixOS = set("aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos",
	"ios", "linux", "netbsd", "openbsd", "solaris")

func set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// platformTag reports whether tag is decided by the port alone, and if so
// its value there — go/build's matchTag, restricted to what a port fixes.
// The three implications (android is linux, illumos is solaris, ios is
// darwin) are the toolchain's, and they apply to filename suffixes too.
func platformTag(tag string, p port) (value, decided bool) {
	switch {
	case tag == "unix":
		return unixOS[p.os], true
	case knownOS[tag]:
		return tag == p.os ||
			(tag == "linux" && p.os == "android") ||
			(tag == "solaris" && p.os == "illumos") ||
			(tag == "darwin" && p.os == "ios"), true
	case knownArch[tag]:
		return tag == p.arch, true
	}
	return false, false
}

// fileNameConstraint is go/build's goodOSArchFile as a predicate over ports:
// `name_GOOS`, `name_GOARCH` or `name_GOOS_GOARCH`, before the extension and
// any `_test`. ok is false when the name constrains nothing. A bare
// `linux.go` is not a constraint — the toolchain requires the underscore.
func fileNameConstraint(base string) (match func(port) bool, ok bool) {
	name := strings.TrimSuffix(base, filepath.Ext(base))
	i := strings.Index(name, "_")
	if i < 0 {
		return nil, false
	}
	l := strings.Split(name[i:], "_")
	if n := len(l); n >= 2 && l[n-1] == "test" {
		l = l[:n-1]
	}
	n := len(l)
	if n >= 2 && knownOS[l[n-2]] && knownArch[l[n-1]] {
		osTag, archTag := l[n-2], l[n-1]
		return func(p port) bool {
			a, _ := platformTag(osTag, p)
			b, _ := platformTag(archTag, p)
			return a && b
		}, true
	}
	if n >= 1 && (knownOS[l[n-1]] || knownArch[l[n-1]]) {
		tag := l[n-1]
		return func(p port) bool {
			v, _ := platformTag(tag, p)
			return v
		}, true
	}
	return nil, false
}

// buildExpr is the file's build constraint: the //go:build line when there
// is one, else the conjunction of its legacy // +build lines — go/build's
// own precedence. Only comments before the package clause count.
func buildExpr(f *ast.File) constraint.Expr {
	if f == nil {
		return nil
	}
	var plus constraint.Expr
	for _, g := range f.Comments {
		if g.Pos() >= f.Package {
			break
		}
		for _, c := range g.List {
			switch {
			case constraint.IsGoBuild(c.Text):
				if x, err := constraint.Parse(c.Text); err == nil {
					return x
				}
			case constraint.IsPlusBuild(c.Text):
				if x, err := constraint.Parse(c.Text); err == nil {
					if plus == nil {
						plus = x
					} else {
						plus = &constraint.AndExpr{X: plus, Y: x}
					}
				}
			}
		}
	}
	return plus
}

// freeTags lists, in first-appearance order, the tags a port does not decide.
func freeTags(x constraint.Expr) []string {
	var out []string
	seen := map[string]bool{}
	var walk func(constraint.Expr)
	walk = func(x constraint.Expr) {
		switch x := x.(type) {
		case *constraint.TagExpr:
			if _, decided := platformTag(x.Tag, port{}); !decided && !seen[x.Tag] {
				seen[x.Tag] = true
				out = append(out, x.Tag)
			}
		case *constraint.NotExpr:
			walk(x.X)
		case *constraint.AndExpr:
			walk(x.X)
			walk(x.Y)
		case *constraint.OrExpr:
			walk(x.X)
			walk(x.Y)
		}
	}
	if x != nil {
		walk(x)
	}
	return out
}

// satisfiable reports whether some assignment of the free tags makes x true
// on port p.
func satisfiable(x constraint.Expr, p port, free []string) bool {
	for mask := 0; mask < 1<<uint(len(free)); mask++ {
		ok := x.Eval(func(tag string) bool {
			if v, decided := platformTag(tag, p); decided {
				return v
			}
			for i, f := range free {
				if f == tag {
					return mask&(1<<uint(i)) != 0
				}
			}
			return false
		})
		if ok {
			return true
		}
	}
	return false
}

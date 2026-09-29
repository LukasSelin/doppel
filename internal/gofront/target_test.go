package gofront

import (
	"go/parser"
	"go/token"
	"testing"
)

func targetsOf(t *testing.T, name, src string) uint64 {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return Targets(name, f)
}

func portBit(t *testing.T, os, arch string) uint64 {
	t.Helper()
	for i, p := range ports {
		if p.os == os && p.arch == arch {
			return 1 << uint(i)
		}
	}
	t.Fatalf("no port %s/%s", os, arch)
	return 0
}

// The toolchain's two sources of truth, and nothing else: an unconstrained
// file is 0, so every language and every plain Go file keeps the old
// behaviour through the zero value.
func TestTargets(t *testing.T) {
	const pkg = "package p\n"
	amd64 := targetsOf(t, "dir/kern_amd64.go", pkg)
	arm64 := targetsOf(t, "dir/kern_arm64.go", pkg)
	generic := targetsOf(t, "dir/kern_generic.go", "//go:build !amd64 && !arm64\n\n"+pkg)
	plain := targetsOf(t, "dir/kern.go", pkg)
	linux := targetsOf(t, "dir/sys_linux.go", pkg)
	windows := targetsOf(t, "dir/sys_other.go", "//go:build windows\n\n"+pkg)
	linuxAmd64Test := targetsOf(t, "dir/sys_linux_amd64_test.go", pkg)

	if plain != 0 {
		t.Errorf("unconstrained file = %#x, want 0", plain)
	}
	if amd64 == 0 || arm64 == 0 || amd64&arm64 != 0 {
		t.Errorf("amd64 %#x and arm64 %#x must be non-zero and disjoint", amd64, arm64)
	}
	if generic&amd64 != 0 || generic&arm64 != 0 || generic == 0 {
		t.Errorf("generic fallback %#x must miss both amd64 and arm64", generic)
	}
	if linux&windows != 0 {
		t.Errorf("linux %#x and windows %#x overlap", linux, windows)
	}
	if linux&amd64 == 0 {
		t.Errorf("a _linux file and an _amd64 file share linux/amd64")
	}
	if want := portBit(t, "linux", "amd64") | portBit(t, "android", "amd64"); linuxAmd64Test != want {
		t.Errorf("_linux_amd64_test.go = %#x, want linux/amd64 and android/amd64 %#x", linuxAmd64Test, want)
	}
}

// The go/build rules that are easy to get wrong.
func TestTargetsToolchainRules(t *testing.T) {
	const pkg = "package p\n"
	// No underscore, no constraint: `linux.go` is an ordinary file.
	if got := targetsOf(t, "linux.go", pkg); got != 0 {
		t.Errorf("linux.go = %#x, want 0", got)
	}
	// A plain _test suffix is not a platform.
	if got := targetsOf(t, "x_test.go", pkg); got != 0 {
		t.Errorf("x_test.go = %#x, want 0", got)
	}
	// ios implies darwin; unix covers darwin but not windows.
	darwin := targetsOf(t, "a_darwin.go", pkg)
	if darwin&portBit(t, "ios", "arm64") == 0 {
		t.Errorf("_darwin file must build for ios")
	}
	unix := targetsOf(t, "a.go", "//go:build unix\n\n"+pkg)
	if unix&portBit(t, "darwin", "arm64") == 0 || unix&portBit(t, "windows", "amd64") != 0 {
		t.Errorf("unix = %#x: want darwin, not windows", unix)
	}
	// Legacy +build lines apply when there is no //go:build.
	if got := targetsOf(t, "a.go", "// +build windows\n\n"+pkg); got != portBit(t, "windows", "386")|portBit(t, "windows", "amd64")|portBit(t, "windows", "arm64") {
		t.Errorf("+build windows = %#x", got)
	}
	// A constraint after the package clause is not a constraint.
	if got := targetsOf(t, "a.go", pkg+"\n//go:build windows\n"); got != 0 {
		t.Errorf("post-package constraint = %#x, want 0", got)
	}
	// A known arch with no port still builds somewhere: not unconstrained.
	if got := targetsOf(t, "a_sparc64.go", pkg); got != offList {
		t.Errorf("_sparc64 = %#x, want the off-list bit", got)
	}
}

// Free tags are the permissive reading: a port counts when some assignment of
// them works, so a custom tag can widen a set but never empty it.
func TestTargetsFreeTags(t *testing.T) {
	const pkg = "package p\n"
	asm := targetsOf(t, "a.go", "//go:build amd64 && !purego\n\n"+pkg)
	fallback := targetsOf(t, "b.go", "//go:build !amd64 || purego\n\n"+pkg)
	if asm&fallback == 0 {
		t.Errorf("purego split must still read as co-buildable (both reach amd64): %#x %#x", asm, fallback)
	}
	if asm&portBit(t, "linux", "arm64") != 0 {
		t.Errorf("amd64 && !purego must not reach arm64")
	}
}

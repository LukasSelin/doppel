package cmd

import "runtime/debug"

// version identifies the doppel build. It exists because a baseline snapshot
// must be discarded when the binary that wrote it differs from the one reading
// it: scoring constants live in code, so a rebuild can move every score without
// a single source file in the analysed repo changing.
//
// Override at build time with:
//
//	go build -ldflags "-X github.com/LukasSelin/doppel/cmd.version=v0.2.0" .
var version = ""

// buildVersion resolves the build identity, preferring an ldflags override and
// falling back to the module version the toolchain recorded. A plain `go build`
// yields "(devel)", which is honest: two dev builds are indistinguishable, so
// the fallback deliberately does not invent a value that would let a stale
// baseline pass the comparability check.
func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}

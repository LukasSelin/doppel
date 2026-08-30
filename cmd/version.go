package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version identifies the doppel build. It exists because a baseline snapshot
// must be discarded when the binary that wrote it differs from the one reading
// it: scoring constants live in code, so a rebuild can move every score without
// a single source file in the analysed repo changing.
//
// Override at build time with:
//
//	go build -ldflags "-X github.com/LukasSelin/doppel/cmd.version=v0.2.0" .
//
// It stays the empty string in source, deliberately, and a scoring change is
// not a reason to put a base version here. A constant would make every plain
// `go build` of every tree claim one identity, which is the same
// stale-baseline-passes-comparability failure `.goreleaser.yaml` avoids by
// stamping `v{{ .Version }}` rather than `{{ .Tag }}` — it would replace a
// coarse answer with a confidently wrong one. Baseline invalidation does not
// depend on this string in any case: `snapshot.Schema` (6) hard-invalidates
// anything older, and `snapshot.Params` carries Threshold and MinNodes, so a
// baseline taken under different floors is already incomparable with a stated
// reason.
var version = ""

// buildVersion resolves the build identity, preferring an ldflags override and
// falling back to the module version the toolchain recorded, then "(devel)".
// The fallback deliberately does not invent a value that would let a stale
// baseline pass the comparability check.
//
// Two things to know about that fallback. A modern toolchain usually resolves
// Main.Version to a VCS pseudo-version rather than "(devel)", so a plain build
// often does carry a commit — but built inside a `git worktree` it resolves
// against the shared git directory and reports the *main* worktree's HEAD, not
// the code being compiled. It is coarse where it works and wrong where it does
// not, which is why `task build-stamped` exists.
func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}

// versionCmd exists because a downloaded binary that cannot say what it is
// makes the comparability story unusable — without it the build identity is
// visible only inside a `--format json` snapshot's `doppel` field.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the doppel build identity",
	Long: "Print the build identity recorded in every snapshot. A session baseline\n" +
		"written by a different build is discarded, so this is the string that\n" +
		"decides whether two runs are comparable at all.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), buildVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

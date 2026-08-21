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

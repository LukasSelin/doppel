package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "doppel",
	Short: "Measure architectural erosion: where a codebase drifted from its own structure",
	Long: `doppel measures architectural erosion in a Go codebase — the gap between the
structure a project intends and the one it actually has.

Nobody erodes an architecture deliberately. Someone writes a second retry loop
because finding the first costs more than writing it; a handler is forked for
another provider and the two age apart. Each edit is defensible on its own and
invisible in review, because review sees a diff and erosion is a property of the
whole corpus. So doppel reads every function in the repo at once and reports what
repeats, what forked, and what no longer fits where it lives.

The corpus is the only norm it has: it reads no declared architecture, no git
history and no deploy state, so layering violations, ownership and configuration
drift are all out of scope.`,
	// Cobra turns this into a --version flag. Evaluated at package-var init,
	// which is safe: the linker sets `version` before any Go init runs.
	Version: buildVersion(),
}

func Execute() error {
	// Deferred rather than run from a PersistentPostRunE: cobra skips Post
	// hooks when a command returns an error, and a failing run is exactly the
	// one worth having a profile of.
	defer stopProfiling()
	return rootCmd.Execute()
}

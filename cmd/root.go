package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "doppel",
	Short: "Find structurally similar functions across a codebase",
	// Cobra turns this into a --version flag. Evaluated at package-var init,
	// which is safe: the linker sets `version` before any Go init runs.
	Version: buildVersion(),
}

func Execute() error {
	return rootCmd.Execute()
}

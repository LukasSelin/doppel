package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "doppel",
	Short: "Find semantically similar functions across a codebase",
}

func Execute() error {
	return rootCmd.Execute()
}

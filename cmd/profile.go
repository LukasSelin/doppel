package cmd

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/spf13/cobra"
)

// --cpuprofile and --memprofile are persistent flags on the root command, so
// every subcommand can be profiled without each one registering its own pair.
//
// Hidden, for the same reason --channel-k and --min-nodes are: no question
// about a codebase is answered by setting them. They exist because the
// alternative was recompiling the tool with a profiling main every time a
// stage got slow, and a measurement seam nobody can reach from the CLI is one
// nobody uses.
var (
	cpuProfilePath string
	memProfilePath string
	cpuProfileFile *os.File
)

func init() {
	rootCmd.PersistentFlags().StringVar(&cpuProfilePath, "cpuprofile", "", "Write a CPU profile to this path")
	rootCmd.PersistentFlags().StringVar(&memProfilePath, "memprofile", "", "Write a heap profile to this path when the command finishes")
	for _, name := range []string{"cpuprofile", "memprofile"} {
		_ = rootCmd.PersistentFlags().MarkHidden(name)
	}
	// No subcommand defines a PersistentPreRunE, so cobra's nearest-ancestor
	// rule leaves this one reachable from every one of them. A subcommand that
	// grows its own must call startProfiling itself, or profiling goes silently
	// dead for that command alone.
	rootCmd.PersistentPreRunE = func(*cobra.Command, []string) error { return startProfiling() }
}

func startProfiling() error {
	if cpuProfilePath == "" {
		return nil
	}
	f, err := os.Create(cpuProfilePath)
	if err != nil {
		return fmt.Errorf("create cpu profile: %w", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return fmt.Errorf("start cpu profile: %w", err)
	}
	cpuProfileFile = f
	return nil
}

// stopProfiling is deferred from Execute rather than run in a PersistentPostRunE
// because cobra skips Post hooks when RunE returns an error — and a run that
// failed is exactly the one worth profiling. Errors here are reported and never
// returned: a profile that could not be written must not change a command's
// exit code, or --memprofile would turn a clean analysis into a failed one.
func stopProfiling() {
	if cpuProfileFile != nil {
		pprof.StopCPUProfile()
		if err := cpuProfileFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "doppel: close cpu profile: %v\n", err)
		}
		cpuProfileFile = nil
	}
	if memProfilePath == "" {
		return
	}
	f, err := os.Create(memProfilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "doppel: create mem profile: %v\n", err)
		return
	}
	defer f.Close()
	// The heap profile is a snapshot, so it is taken after the work rather
	// than alongside it, and after a GC so what it shows is live rather than
	// whatever had not been collected yet.
	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		fmt.Fprintf(os.Stderr, "doppel: write mem profile: %v\n", err)
	}
}

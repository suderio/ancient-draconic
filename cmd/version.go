package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version is injected by GoReleaser via ldflags at build time
	Version = "dev"
	// Commit is injected by GoReleaser via ldflags at build time
	Commit = "none"
	// BuildDate is injected by GoReleaser via ldflags at build time
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the application version",
	Long:  `Displays the current running version of dndsl alongside the build metadata.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("dndsl Engine version %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Build date: %s\n", BuildDate)
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

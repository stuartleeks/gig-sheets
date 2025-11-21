package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build information variables (set by main package from GoReleaser)
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print the version, commit hash, and build date of gigsheets",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gigsheets version %s\n", Version)
		fmt.Printf("Built from commit: %s\n", Commit)
		fmt.Printf("Built on: %s\n", Date)
	},
}

package main

import (
	"os"

	"gigsheets/cmd"
)

// Build information, set by GoReleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set build information in cmd package
	cmd.Version = version
	cmd.Commit = commit
	cmd.Date = date

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

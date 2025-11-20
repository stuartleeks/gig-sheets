package cmd
package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gigsheets",
	Short: "A CLI tool for generating PDF song sheets from YAML configurations",
	Long: `Gigsheets is a CLI tool that reads configuration and gig YAML files
to generate PDF files containing song sheets for musical performances.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
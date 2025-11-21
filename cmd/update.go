package cmd

import (
	"bufio"
	"fmt"
	"os"

	"gigsheets/internal/pkg/update"

	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update gigsheets CLI",
	Long:  "Check for and apply updates to the gigsheets CLI tool",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// do nothing - suppress root PersistentPreRun which does periodic update check
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		checkOnly, _ := cmd.Flags().GetBool("check-only")
		yes, _ := cmd.Flags().GetBool("yes")

		latest, err := update.CheckForUpdate(Version)
		if err != nil {
			return fmt.Errorf("error occurred while checking for updates: %v", err)
		}

		if latest == nil {
			fmt.Println("No updates available")
			return nil
		}

		fmt.Printf("\n\n UPDATE AVAILABLE: %s \n \n Release notes: %s\n", latest.Version, latest.ReleaseNotes)

		if checkOnly {
			return nil
		}

		fmt.Print("Do you want to update? (y/n): ")
		if !yes {
			input, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil || (input != "y\n" && input != "y\r\n") {
				// error or something other than `y`
				return err
			}
		}
		fmt.Println("Applying...")

		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not locate executable path: %v", err)
		}
		if err := selfupdate.UpdateTo(latest.AssetURL, exe); err != nil {
			return fmt.Errorf("error occurred while updating binary: %v", err)
		}
		fmt.Printf("Successfully updated to version %s\n", latest.Version)
		return nil
	},
}

func init() {
	updateCmd.Flags().Bool("check-only", false, "Check for an update without applying")
	updateCmd.Flags().BoolP("yes", "y", false, "Automatically apply any updates (i.e. answer yes)")
}

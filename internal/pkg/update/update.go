package update

import (
	"fmt"
	"os"
	"time"

	"gigsheets/internal/pkg/status"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

func CheckForUpdate(currentVersion string) (*selfupdate.Release, error) {

	latest, found, err := selfupdate.DetectLatest("stuartleeks/gig-sheets")
	if err != nil {
		return nil, fmt.Errorf("error occurred while detecting version: %v", err)
	}

	// If current version is "dev" or not a valid semver, treat it as outdated
	// This allows local builds to check for updates
	if currentVersion == "dev" || currentVersion == "" {
		if found {
			return latest, nil
		}
		return nil, nil
	}

	v, err := semver.Parse(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("error occurred while parsing version: %v", err)
	}

	if !found || latest.Version.LTE(v) {
		return nil, nil
	}
	return latest, nil
}

func PeriodicCheckForUpdate(currentVersion string) {
	const checkInterval time.Duration = 24 * time.Hour

	if os.Getenv("GIGSHEETS_SKIP_UPDATE") != "" {
		// Skip update check
		return
	}

	lastCheck := status.GetLastUpdateCheck()

	if time.Now().Before(lastCheck.Add(checkInterval)) {
		return
	}
	fmt.Println("Checking for updates...")
	latest, err := CheckForUpdate(currentVersion)
	if err != nil {
		fmt.Printf("Error checking for updates: %s", err)
	}

	status.SetLastUpdateCheck(time.Now())
	if err = status.SaveStatus(); err != nil {
		fmt.Printf("Error saving last update check time: %s\n", err)
	}

	if latest == nil {
		return
	}

	fmt.Printf("\n\n UPDATE AVAILABLE: %s \n \n Release notes: %s\n", latest.Version, latest.ReleaseNotes)
	fmt.Printf("Run `gigsheets update` to apply the update\n\n")
}

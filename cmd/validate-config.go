package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	validateConfigFile string
	addMissing         bool
)

var validateConfigCmd = &cobra.Command{
	Use:   "validate-config",
	Short: "Validate that all images in the config file exist",
	Long: `Validate that all images referenced in the config file exist on disk.
Optionally add missing images from the image folder to the config file using --add-missing flag.`,
	Run: runValidateConfig,
}

func init() {
	validateConfigCmd.Flags().StringVarP(&validateConfigFile, "config", "c", "config.yaml", "Path to config YAML file")
	validateConfigCmd.Flags().BoolVarP(&addMissing, "add-missing", "a", false, "Add missing images from image folder to config file")
}

func runValidateConfig(cmd *cobra.Command, args []string) {
	// Load configuration
	config, err := loadConfig(validateConfigFile)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	// Get the config directory for resolving relative paths
	configDir := filepath.Dir(validateConfigFile)
	imageDir := filepath.Join(configDir, config.ImageFolder)

	// Validate that image folder exists
	if _, err := os.Stat(imageDir); os.IsNotExist(err) {
		log.Fatalf("Image folder does not exist: %s", imageDir)
	}

	// Track validation results
	var missingImages []string
	var validImages []string

	// Validate existing images in config
	for _, song := range config.Songs {
		// Handle backward compatibility - single image
		if song.Image != "" {
			imagePath := filepath.Join(imageDir, song.Image)
			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				missingImages = append(missingImages, fmt.Sprintf("Song '%s': %s", song.Nickname, song.Image))
			} else {
				validImages = append(validImages, fmt.Sprintf("Song '%s': %s", song.Nickname, song.Image))
			}
		}

		// Handle multiple images
		if song.Images != nil {
			for variant, imageName := range song.Images {
				imagePath := filepath.Join(imageDir, imageName)
				if _, err := os.Stat(imagePath); os.IsNotExist(err) {
					missingImages = append(missingImages, fmt.Sprintf("Song '%s' variant '%s': %s", song.Nickname, variant, imageName))
				} else {
					validImages = append(validImages, fmt.Sprintf("Song '%s' variant '%s': %s", song.Nickname, variant, imageName))
				}
			}
		}
	}

	// Print validation results
	fmt.Printf("Validation Results:\n")
	fmt.Printf("==================\n")

	if len(validImages) > 0 {
		fmt.Printf("\nValid images (%d):\n", len(validImages))
		for _, img := range validImages {
			fmt.Printf("  ✓ %s\n", img)
		}
	}

	if len(missingImages) > 0 {
		fmt.Printf("\nMissing images (%d):\n", len(missingImages))
		for _, img := range missingImages {
			fmt.Printf("  ✗ %s\n", img)
		}
	}

	// Handle --add-missing flag
	if addMissing {
		fmt.Printf("\nScanning for images to add...\n")

		// Get all image files from the image directory
		imageFiles, err := filepath.Glob(filepath.Join(imageDir, "*"))
		if err != nil {
			log.Fatalf("Error scanning image directory: %v", err)
		}

		// Build a set of existing image files in config for quick lookup
		existingImages := make(map[string]bool)
		for _, song := range config.Songs {
			if song.Image != "" {
				existingImages[song.Image] = true
			}
			for _, imageName := range song.Images {
				existingImages[imageName] = true
			}
		}

		// Find images that aren't in config
		var newSongs []Song
		supportedExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true}

		for _, imagePath := range imageFiles {
			if info, err := os.Stat(imagePath); err == nil && !info.IsDir() {
				imageName := filepath.Base(imagePath)
				ext := strings.ToLower(filepath.Ext(imageName))

				// Check if it's a supported image format and not already in config
				if supportedExts[ext] && !existingImages[imageName] {
					// Use filename without extension as song nickname
					nickname := strings.TrimSuffix(imageName, filepath.Ext(imageName))

					newSong := Song{
						Nickname: nickname,
						Image:    imageName,
					}
					newSongs = append(newSongs, newSong)
				}
			}
		}

		if len(newSongs) > 0 {
			fmt.Printf("\nAdding %d new songs to config:\n", len(newSongs))
			for _, song := range newSongs {
				fmt.Printf("  + %s -> %s\n", song.Nickname, song.Image)
				config.Songs = append(config.Songs, song)
			}

			// Write updated config back to file
			err := writeConfig(config, validateConfigFile)
			if err != nil {
				log.Fatalf("Error writing updated config file: %v", err)
			}

			fmt.Printf("\nSuccessfully updated config file: %s\n", validateConfigFile)
		} else {
			fmt.Printf("\nNo new images found to add.\n")
		}
	}

	// Exit with error code if there are missing images
	if len(missingImages) > 0 && !addMissing {
		fmt.Printf("\nValidation failed: %d missing images\n", len(missingImages))
		fmt.Printf("Use --add-missing flag to automatically add missing images from the image folder.\n")
		os.Exit(1)
	} else if len(missingImages) == 0 {
		fmt.Printf("\n✓ All images in config exist!\n")
	}
}

// writeConfig writes the config struct back to a YAML file
func writeConfig(config *Config, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	validateConfigFile string
	addMissing         bool
	sortSongs          bool
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
	validateConfigCmd.Flags().BoolVarP(&sortSongs, "sort", "s", false, "Sort songs alphabetically by nickname")
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
	configChanged := false

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

		supportedExts := map[string]bool{".png": true, ".jpg": true, ".jpeg": true}

		// Group images by base name
		// First pass: collect all image names without extensions
		type imageInfo struct {
			fileName       string
			nameWithoutExt string
		}
		var allImages []imageInfo

		for _, imagePath := range imageFiles {
			if info, err := os.Stat(imagePath); err == nil && !info.IsDir() {
				imageName := filepath.Base(imagePath)
				ext := strings.ToLower(filepath.Ext(imageName))

				// Check if it's a supported image format and not already in config
				if supportedExts[ext] && !existingImages[imageName] {
					nameWithoutExt := strings.TrimSuffix(imageName, filepath.Ext(imageName))
					allImages = append(allImages, imageInfo{
						fileName:       imageName,
						nameWithoutExt: nameWithoutExt,
					})
				}
			}
		}

		// Sort images alphabetically to process base names before variants
		sort.Slice(allImages, func(i, j int) bool {
			return allImages[i].nameWithoutExt < allImages[j].nameWithoutExt
		})

		// Second pass: group images that share a common base
		// Key: base name, Value: map of variant -> filename
		imageGroups := make(map[string]map[string]string)
		used := make(map[int]bool) // Track which images have been grouped

		for i, img1 := range allImages {
			if used[i] {
				continue
			}

			// Start a new group with this image
			baseName := img1.nameWithoutExt
			variants := make(map[string]string)
			variants["default"] = img1.fileName
			used[i] = true

			// Look for other images that match this base with a suffix
			for j, img2 := range allImages {
				if i == j || used[j] {
					continue
				}

				// Check if img2 starts with img1's name followed by a hyphen
				if strings.HasPrefix(img2.nameWithoutExt, img1.nameWithoutExt+"-") {
					suffix := img2.nameWithoutExt[len(img1.nameWithoutExt)+1:]
					if suffix != "" {
						variants[suffix] = img2.fileName
						used[j] = true
					}
				}
			}

			imageGroups[baseName] = variants
		}

		// Create new songs from grouped images
		var newSongs []Song
		for baseName, variants := range imageGroups {
			if len(variants) == 1 {
				// Single image - check if it's the default variant
				for variant, imageName := range variants {
					if variant == "default" {
						// Use simple image format
						newSongs = append(newSongs, Song{
							Nickname: baseName,
							Image:    imageName,
						})
					} else {
						// Use images map format with the variant
						newSongs = append(newSongs, Song{
							Nickname: baseName,
							Images:   variants,
						})
					}
				}
			} else {
				// Multiple images - use images map format
				newSongs = append(newSongs, Song{
					Nickname: baseName,
					Images:   variants,
				})
			}
		}

		if len(newSongs) > 0 {
			fmt.Printf("\nAdding %d new songs to config:\n", len(newSongs))
			for _, song := range newSongs {
				if song.Image != "" {
					fmt.Printf("  + %s -> %s\n", song.Nickname, song.Image)
				} else {
					fmt.Printf("  + %s -> %v\n", song.Nickname, song.Images)
				}
				config.Songs = append(config.Songs, song)
			}
			configChanged = true
		} else {
			fmt.Printf("\nNo new images found to add.\n")
		}
	}

	// Handle --sort flag
	if sortSongs {
		// Check if songs are already sorted
		alreadySorted := slices.IsSortedFunc(config.Songs, func(a, b Song) int {
			return strings.Compare(strings.ToLower(a.Nickname), strings.ToLower(b.Nickname))
		})

		if !alreadySorted {
			fmt.Printf("\nSorting songs alphabetically by nickname...\n")

			slices.SortFunc(config.Songs, func(a, b Song) int {
				return strings.Compare(strings.ToLower(a.Nickname), strings.ToLower(b.Nickname))
			})

			configChanged = true
		} else {
			fmt.Printf("\nSongs are already sorted alphabetically.\n")
		}
	}

	// Write config back to file if there were any changes
	if configChanged {
		err := writeConfig(config, validateConfigFile)
		if err != nil {
			log.Fatalf("Error writing updated config file: %v", err)
		}

		fmt.Printf("\nSuccessfully updated config file: %s\n", validateConfigFile)
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

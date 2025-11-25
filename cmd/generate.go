package cmd

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jung-kurt/gofpdf"
	"github.com/spf13/cobra"
	"golang.org/x/image/draw"
	"gopkg.in/yaml.v3"
)

// Config represents the structure of config.yaml
type Config struct {
	ImageFolder  string   `yaml:"imageFolder"`
	GigsFolder   string   `yaml:"gigsFolder"`
	OutputFolder string   `yaml:"outputFolder"`
	Spacing      *float64 `yaml:"spacing,omitempty"` // Optional spacing between images
	Songs        []Song   `yaml:"songs"`
}

// Song represents a song configuration
type Song struct {
	Nickname string            `yaml:"nickname"`
	Image    string            `yaml:"image,omitempty"`  // For backward compatibility - single image
	Images   map[string]string `yaml:"images,omitempty"` // For multiple named images
}

// Gig represents the structure of gig.yaml
type Gig struct {
	Name string `yaml:"name"`
	Sets []Set  `yaml:"sets"`
}

// Set represents a set of songs
type Set struct {
	Name  string   `yaml:"name"`
	Songs []string `yaml:"songs"`
}

var (
	configFile     string
	watchMode      bool
	spacingFlag    *float64 // Pointer to distinguish between unset and 0
	imageOverride  string   // Override image name to use if it exists
	outputOverride string   // Override output folder path
	allSongs       bool     // Generate _all.pdf with all songs from config
	debugMode      bool     // Enable debug logging
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate PDF song sheets from configuration and gig files",
	Long: `Generate a PDF file containing song sheets based on the songs specified
in the gig file, using image paths from the configuration file.`,
	Run: runGenerate,
}

func init() {
	generateCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to config YAML file")
	generateCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch for changes and regenerate automatically")
	generateCmd.Flags().StringVarP(&imageOverride, "image-override", "i", "", "Image name to use for all songs if it exists, otherwise use the one specified in gig YAML")
	generateCmd.Flags().StringVarP(&outputOverride, "output", "o", "", "Override output folder path from config file")
	generateCmd.Flags().BoolVarP(&allSongs, "all-songs", "a", false, "Generate _all.pdf containing all songs from config (uses default image unless image-override is set)")
	generateCmd.Flags().BoolVarP(&debugMode, "debug", "d", false, "Enable debug logging")

	// Use a local variable for the flag, then assign to spacingFlag in runGenerate
	generateCmd.Flags().Float64P("spacing", "s", -1, "Spacing between images in mm (default: 5.0, or value from config)")
}

func runGenerate(cmd *cobra.Command, args []string) {
	// Get the spacing flag value
	spacingValue, _ := cmd.Flags().GetFloat64("spacing")
	if spacingValue >= 0 {
		spacingFlag = &spacingValue
	}

	if watchMode {
		runGenerateWatch()
	} else {
		runGenerateOnce()
	}
}

func runGenerateOnce() {
	err := generateAllGigs()
	if err != nil {
		log.Fatalf("Error generating PDFs: %v", err)
	}
}

func runGenerateWatch() {
	// Initial generation
	fmt.Println("Initial PDF generation...")
	err := generateAllGigs()
	if err != nil {
		log.Printf("Error during initial generation: %v", err)
	}

	// Get the config directory for resolving relative paths
	configDir := filepath.Dir(configFile)

	// Load config to get gigs folder
	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	gigsDir := filepath.Join(configDir, config.GigsFolder)

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error creating watcher: %v", err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.Printf("Error closing watcher: %v", err)
		}
	}()

	// Watch config file
	err = watcher.Add(configFile)
	if err != nil {
		log.Fatalf("Error watching config file: %v", err)
	}

	// Watch gigs directory
	err = watcher.Add(gigsDir)
	if err != nil {
		log.Fatalf("Error watching gigs directory: %v", err)
	}

	fmt.Printf("\nWatching for changes...\n")
	fmt.Printf("  Config: %s\n", configFile)
	fmt.Printf("  Gigs:   %s\n", gigsDir)
	fmt.Println("\nPress Ctrl+C to stop")

	// Debounce timer to avoid multiple regenerations for rapid file changes
	var debounceTimer *time.Timer
	debounceDuration := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Only process write and create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// Cancel existing timer if any
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				// Set new timer
				debounceTimer = time.AfterFunc(debounceDuration, func() {
					fmt.Printf("\n[%s] Change detected in: %s\n", time.Now().Format("15:04:05"), filepath.Base(event.Name))
					fmt.Println("Regenerating PDFs...")

					err := generateAllGigs()
					if err != nil {
						log.Printf("Error generating PDFs: %v", err)
					} else {
						fmt.Println("✓ PDFs regenerated successfully")
					}
				})
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// resolveSpacing determines the spacing value to use based on priority:
// 1. Command-line flag (if set)
// 2. Config file value (if set)
// 3. Default value (5.0)
func resolveSpacing(config *Config) float64 {
	// Priority 1: Command-line flag
	if spacingFlag != nil {
		return *spacingFlag
	}

	// Priority 2: Config file
	if config.Spacing != nil {
		return *config.Spacing
	}

	// Priority 3: Default
	return 5.0
}

func generateAllGigs() error {
	// Load configuration
	config, err := loadConfig(configFile)
	if err != nil {
		return fmt.Errorf("error loading config file: %w", err)
	}

	// Resolve the spacing value
	spacing := resolveSpacing(config)

	// Get the config directory for resolving relative paths
	configDir := filepath.Dir(configFile)

	// Resolve paths relative to config file
	gigsDir := filepath.Join(configDir, config.GigsFolder)

	// Use outputOverride if set, otherwise use config value
	var outputDir string
	if outputOverride != "" {
		if filepath.IsAbs(outputOverride) {
			outputDir = outputOverride
		} else {
			outputDir = filepath.Join(configDir, outputOverride)
		}
	} else {
		outputDir = filepath.Join(configDir, config.OutputFolder)
	}

	imagesDir := filepath.Join(configDir, config.ImageFolder)

	// Create output directory if it doesn't exist
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Read all gig files from the gigs directory (both .yaml and .yml)
	yamlFiles, err := filepath.Glob(filepath.Join(gigsDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("error reading gig files: %w", err)
	}
	ymlFiles, err := filepath.Glob(filepath.Join(gigsDir, "*.yml"))
	if err != nil {
		return fmt.Errorf("error reading gig files: %w", err)
	}
	gigFiles := append(yamlFiles, ymlFiles...)

	if len(gigFiles) == 0 {
		log.Printf("No gig files found in %s", gigsDir)
		return nil
	}

	fmt.Printf("Found %d gig file(s) in %s\n", len(gigFiles), gigsDir)

	// Process each gig file
	for _, gigFile := range gigFiles {
		// Load gig
		gig, err := loadGig(gigFile)
		if err != nil {
			log.Printf("Error loading gig file %s: %v", gigFile, err)
			continue
		}

		// Generate output filename
		gigBasename := filepath.Base(gigFile)
		gigName := strings.TrimSuffix(gigBasename, filepath.Ext(gigBasename))
		outputFile := filepath.Join(outputDir, gigName+".pdf")

		// Generate PDF
		err = generatePDF(config, gig, outputFile, imagesDir, gigFile, spacing, imageOverride)
		if err != nil {
			log.Printf("Error generating PDF for %s: %v", gigFile, err)
			continue
		}

		fmt.Printf("Successfully generated PDF: %s\n", outputFile)
	}

	// Generate _all.pdf if --all-songs flag is set
	if allSongs {
		// Build filename based on image override
		allSongsFilename := "_all.pdf"
		if imageOverride != "" {
			allSongsFilename = fmt.Sprintf("_all_%s.pdf", imageOverride)
		}
		allSongsFile := filepath.Join(outputDir, allSongsFilename)

		// Create an in-memory gig with all songs from config
		allSongsGig := &Gig{
			Name: "All Songs",
			Sets: []Set{
				{
					Name:  "All Songs",
					Songs: make([]string, len(config.Songs)),
				},
			},
		}

		// Populate the songs list (use default image unless image-override is set)
		for i, song := range config.Songs {
			allSongsGig.Sets[0].Songs[i] = song.Nickname
		}

		err = generatePDF(config, allSongsGig, allSongsFile, imagesDir, "config", spacing, imageOverride)
		if err != nil {
			log.Printf("Error generating _all.pdf: %v", err)
		} else {
			fmt.Printf("Successfully generated PDF: %s\n", allSongsFile)
		}
	}

	return nil
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	return &config, nil
}

func loadGig(filename string) (*Gig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read gig file: %w", err)
	}

	var gig Gig
	err = yaml.Unmarshal(data, &gig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gig YAML: %w", err)
	}

	return &gig, nil
}

// isWhiteOrTransparent checks if a pixel is white or transparent
func isWhiteOrTransparent(c color.Color) bool {
	r, g, b, a := c.RGBA()

	// Check if transparent (alpha = 0)
	if a == 0 {
		return true
	}

	// Convert to 8-bit values for easier comparison
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)

	// Check if white (or very close to white)
	threshold := uint8(240) // Allow slight variations from pure white
	return r8 >= threshold && g8 >= threshold && b8 >= threshold
}

// findLeftMostContent finds the leftmost non-white, non-transparent column
func findLeftMostContent(img image.Image) int {
	bounds := img.Bounds()

	// Scan from left to right
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		// Check column from top to bottom
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			if !isWhiteOrTransparent(img.At(x, y)) {
				return x
			}
		}
	}

	// If no content found, return original left boundary
	return bounds.Min.X
}

// findTopMostContent finds the topmost non-white, non-transparent row
func findTopMostContent(img image.Image) int {
	bounds := img.Bounds()

	// Scan from top to bottom
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		// Check row from left to right
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !isWhiteOrTransparent(img.At(x, y)) {
				return y
			}
		}
	}

	// If no content found, return original top boundary
	return bounds.Min.Y
}

// findBottomMostContent finds the bottommost non-white, non-transparent row
func findBottomMostContent(img image.Image) int {
	bounds := img.Bounds()

	// Scan from bottom to top
	for y := bounds.Max.Y - 1; y >= bounds.Min.Y; y-- {
		// Check row from left to right
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !isWhiteOrTransparent(img.At(x, y)) {
				return y + 1 // Return the position after the last content row
			}
		}
	}

	// If no content found, return original bottom boundary
	return bounds.Max.Y
}

// findRightMostContent finds the rightmost non-white, non-transparent column
func findRightMostContent(img image.Image) int {
	bounds := img.Bounds()

	// Scan from right to left
	for x := bounds.Max.X - 1; x >= bounds.Min.X; x-- {
		// Check column from top to bottom
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			if !isWhiteOrTransparent(img.At(x, y)) {
				return x + 1 // Return the position after the last content column
			}
		}
	}

	// If no content found, return original right boundary
	return bounds.Max.X
}

// cropImage crops the image from all edges (top, left, bottom, right) up to the content boundaries
func cropImage(imagePath string, songName string) (image.Image, error) {
	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log the error but don't fail the operation since we're in defer
			log.Printf("Warning: failed to close file %s: %v", imagePath, err)
		}
	}()

	// Decode the image based on file extension
	var img image.Image
	ext := strings.ToLower(filepath.Ext(imagePath))
	switch ext {
	case ".png":
		img, err = png.Decode(file)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	default:
		// Try to decode as generic image
		img, _, err = image.Decode(file)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Find the content boundaries
	bounds := img.Bounds()
	leftMostX := findLeftMostContent(img)
	topMostY := findTopMostContent(img)
	bottomMostY := findBottomMostContent(img)
	rightMostX := findRightMostContent(img)

	// If no cropping needed, return original image
	if leftMostX <= bounds.Min.X && topMostY <= bounds.Min.Y && bottomMostY >= bounds.Max.Y && rightMostX >= bounds.Max.X {
		if debugMode {
			log.Printf("[DEBUG] Image '%s' - no cropping needed: %dx%d", songName, bounds.Dx(), bounds.Dy())
		}
		return img, nil
	}

	// Calculate the cropped dimensions
	croppedWidth := rightMostX - leftMostX
	croppedHeight := bottomMostY - topMostY

	// Validate dimensions
	if croppedWidth <= 0 || croppedHeight <= 0 {
		// If dimensions are invalid, return original image
		if debugMode {
			log.Printf("[DEBUG] Image '%s' - invalid crop dimensions, using original", songName)
		}
		return img, nil
	}

	if debugMode {
		originalWidth := bounds.Dx()
		originalHeight := bounds.Dy()
		croppedLeft := leftMostX - bounds.Min.X
		croppedTop := topMostY - bounds.Min.Y
		croppedRight := bounds.Max.X - rightMostX
		croppedBottom := bounds.Max.Y - bottomMostY
		log.Printf("[DEBUG] Image '%s' - cropping: original=%dx%d, cropped=%dx%d, removed: left=%d, top=%d, right=%d, bottom=%d",
			songName, originalWidth, originalHeight, croppedWidth, croppedHeight,
			croppedLeft, croppedTop, croppedRight, croppedBottom)
	}

	// Create cropped image with new dimensions
	croppedBounds := image.Rect(0, 0, croppedWidth, croppedHeight)
	croppedImg := image.NewRGBA(croppedBounds)

	// Copy the cropped portion
	srcRect := image.Rect(leftMostX, topMostY, rightMostX, bottomMostY)
	draw.Copy(croppedImg, croppedImg.Bounds().Min, img, srcRect, draw.Src, nil)

	return croppedImg, nil
}

// addErrorText adds red error text to the PDF at the current position
func addErrorText(pdf *gofpdf.Fpdf, currentY *float64, pageWidth, pageHeight, margin, footerHeight, spacing float64, errorMsg string, addFooter func()) {
	errorHeight := 10.0 // Height for error message

	// Calculate available space on current page
	remainingHeight := pageHeight - footerHeight - margin - *currentY

	// Check if we need a new page
	if remainingHeight < errorHeight+spacing {
		pdf.AddPage()
		addFooter()
		*currentY = margin
	}

	// Set red text color (RGB: 255, 0, 0)
	pdf.SetTextColor(255, 0, 0)
	pdf.SetFont("Arial", "B", 12)

	// Add the error message
	pdf.SetXY(margin, *currentY)
	pdf.MultiCell(pageWidth-2*margin, errorHeight, errorMsg, "", "L", false)

	// Reset text color to black for subsequent content
	pdf.SetTextColor(0, 0, 0)

	// Update current Y position
	*currentY += errorHeight + spacing
}

func generatePDF(config *Config, gig *Gig, outputPath string, imagesDir string, gigFile string, spacing float64, imageOverride string) error {
	// Create a map for quick song lookup that supports both single and multiple images
	songMap := make(map[string]map[string]string)
	for _, song := range config.Songs {
		imageMap := make(map[string]string)

		// Handle backward compatibility - if single image is specified
		if song.Image != "" {
			imageMap["default"] = song.Image
		}

		// Handle multiple images
		if song.Images != nil {
			for name, path := range song.Images {
				imageMap[name] = path
			}
		}

		songMap[song.Nickname] = imageMap
	}

	// No need for temp files cleanup anymore since we're working in-memory

	// Create PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)

	// Page dimensions and layout constants
	pageWidth, pageHeight := pdf.GetPageSize()
	margin := 10.0
	footerHeight := 15.0
	availableWidth := pageWidth - 2*margin

	// Track current page position
	currentY := margin
	pageNum := 0

	// Add footer function
	addFooter := func() {
		pageNum++
		pdf.SetY(pageHeight - footerHeight)
		pdf.SetFont("Arial", "", 8)
		pdf.Cell(0, 5, fmt.Sprintf("%s - Page %d", gig.Name, pageNum))
	}

	// Start first page
	pdf.AddPage()
	addFooter()

	// Process each set
	for setIndex, set := range gig.Sets {
		// Add set separator (start new page if not the first set and not at top of page)
		if setIndex > 0 && currentY > margin {
			pdf.AddPage()
			addFooter()
			currentY = margin
		}

		// Add songs for this set
		for _, songName := range set.Songs {
			// Parse song name and image name
			parts := strings.Split(songName, "#")
			actualSongName := parts[0]
			imageName := "default"
			if len(parts) > 1 {
				imageName = parts[1]
			}

			// Look up the song in the map
			imageMap, exists := songMap[actualSongName]
			if !exists {
				errorMsg := fmt.Sprintf("ERROR: No configuration found for song '%s'", actualSongName)
				log.Printf("%s: Warning: %s", gigFile, errorMsg)
				addErrorText(pdf, &currentY, pageWidth, pageHeight, margin, footerHeight, spacing, errorMsg, addFooter)
				continue
			}

			// Apply image override if specified
			if imageOverride != "" {
				// Check if the override image exists for this song
				if _, exists := imageMap[imageOverride]; exists {
					imageName = imageOverride
				}
				// Otherwise, keep the imageName from the gig YAML
			}

			// Look up the specific image
			imagePath, exists := imageMap[imageName]
			if !exists {
				errorMsg := fmt.Sprintf("ERROR: No image '%s' found for song '%s'", imageName, actualSongName)
				log.Printf("%s: Warning: %s", gigFile, errorMsg)
				addErrorText(pdf, &currentY, pageWidth, pageHeight, margin, footerHeight, spacing, errorMsg, addFooter)
				continue
			}

			// Make image path relative to images directory if it's not absolute
			if !filepath.IsAbs(imagePath) {
				imagePath = filepath.Join(imagesDir, imagePath)
			}

			// Check if image file exists
			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				errorMsg := fmt.Sprintf("ERROR: Image file not found: %s", imagePath)
				log.Printf("%s: Warning: %s", gigFile, errorMsg)
				addErrorText(pdf, &currentY, pageWidth, pageHeight, margin, footerHeight, spacing, errorMsg, addFooter)
				continue
			}

			// Crop the image to remove white/transparent space from top, left, and bottom
			croppedImg, err := cropImage(imagePath, songName)
			if err != nil {
				log.Printf("%s: Warning: Could not crop image %s: %v", gigFile, imagePath, err)
				croppedImg = nil // Will use original file path below
			}

			var imageInfo *gofpdf.ImageInfoType
			var finalImagePath string

			if croppedImg != nil {
				// Convert cropped image to bytes buffer for gofpdf
				var buf bytes.Buffer
				ext := strings.ToLower(filepath.Ext(imagePath))
				switch ext {
				case ".png":
					err = png.Encode(&buf, croppedImg)
				case ".jpg", ".jpeg":
					err = jpeg.Encode(&buf, croppedImg, &jpeg.Options{Quality: 90})
				default:
					err = png.Encode(&buf, croppedImg) // Default to PNG
				}

				if err != nil {
					log.Printf("%s: Warning: Could not encode cropped image %s: %v", gigFile, imagePath, err)
					// Fall back to original file
					imageInfo = pdf.RegisterImage(imagePath, "")
					finalImagePath = imagePath
				} else {
					// Register the cropped image from bytes buffer
					croppedImageName := fmt.Sprintf("cropped_%s", songName)
					imageType := ""
					switch ext {
					case ".png":
						imageType = "PNG"
					case ".jpg", ".jpeg":
						imageType = "JPG"
					default:
						imageType = "PNG"
					}
					imageInfo = pdf.RegisterImageReader(croppedImageName, imageType, &buf)
					finalImagePath = croppedImageName
				}
			} else {
				// Use original file
				imageInfo = pdf.RegisterImage(imagePath, "")
				finalImagePath = imagePath
			}
			if imageInfo == nil {
				log.Printf("%s: Warning: Could not process image: %s", gigFile, imagePath)
				continue
			}

			// Get natural image dimensions in points, then convert to mm
			naturalWidth, naturalHeight := imageInfo.Extent()
			// Convert from points to mm (1 point = 0.352778 mm)
			imageWidth := naturalWidth * 0.352778
			imageHeight := naturalHeight * 0.352778

			// Calculate pixel dimensions (1 point = 1.333... pixels at 96 DPI)
			imageWidthPx := int(naturalWidth * 1.333333)
			imageHeightPx := int(naturalHeight * 1.333333)

			// Only scale down if image is wider than available width
			if imageWidth > availableWidth {
				scale := availableWidth / imageWidth
				if debugMode {
					log.Printf("[DEBUG] Image '%s' - scaling: original=%dx%d (%.2fmm x %.2fmm), scale=%.4f, final=%dx%d (%.2fmm x %.2fmm)",
						songName, imageWidthPx, imageHeightPx, imageWidth, imageHeight, scale,
						int(float64(imageWidthPx)*scale), int(float64(imageHeightPx)*scale), availableWidth, imageHeight*scale)
				}
				imageWidth = availableWidth
				imageHeight *= scale
			} else if debugMode {
				log.Printf("[DEBUG] Image '%s' - no scaling needed: %dx%d (%.2fmm x %.2fmm), available width: %.2fmm",
					songName, imageWidthPx, imageHeightPx, imageWidth, imageHeight, availableWidth)
			} // Calculate available space on current page
			remainingHeight := pageHeight - footerHeight - margin - currentY

			// Check if we have enough space for the image
			if remainingHeight < imageHeight+spacing {
				pdf.AddPage()
				addFooter()
				currentY = margin
			}

			// Add image without scaling (unless it was too wide)
			pdf.ImageOptions(finalImagePath, margin, currentY, imageWidth, imageHeight, false, gofpdf.ImageOptions{}, 0, "")
			currentY += imageHeight + spacing
		}
	}

	// Save PDF
	err := pdf.OutputFileAndClose(outputPath)
	if err != nil {
		return fmt.Errorf("failed to save PDF: %w", err)
	}

	return nil
}

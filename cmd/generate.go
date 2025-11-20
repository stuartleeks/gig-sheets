package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jung-kurt/gofpdf"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Config represents the structure of config.yaml
type Config struct {
	Songs []Song `yaml:"songs"`
}

// Song represents a song configuration
type Song struct {
	Nickname string `yaml:"nickname"`
	Image    string `yaml:"image"`
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
	configFile string
	gigFile    string
	outputFile string
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
	generateCmd.Flags().StringVarP(&gigFile, "gig", "g", "gig.yaml", "Path to gig YAML file")
	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "example-output.pdf", "Output PDF file path")
}

func runGenerate(cmd *cobra.Command, args []string) {
	// Load configuration
	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	// Load gig
	gig, err := loadGig(gigFile)
	if err != nil {
		log.Fatalf("Error loading gig file: %v", err)
	}

	// Generate PDF
	err = generatePDF(config, gig, outputFile)
	if err != nil {
		log.Fatalf("Error generating PDF: %v", err)
	}

	fmt.Printf("Successfully generated PDF: %s\n", outputFile)
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

func generatePDF(config *Config, gig *Gig, outputPath string) error {
	// Create a map for quick song lookup
	songMap := make(map[string]string)
	for _, song := range config.Songs {
		songMap[song.Nickname] = song.Image
	}

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
			imagePath, exists := songMap[songName]
			if !exists {
				log.Printf("Warning: No image found for song '%s'", songName)
				continue
			}

			// Make image path relative to config file directory if it's not absolute
			if !filepath.IsAbs(imagePath) {
				configDir := filepath.Dir(configFile)
				imagePath = filepath.Join(configDir, imagePath)
			}

			// Check if image file exists
			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				log.Printf("Warning: Image file not found: %s", imagePath)
				continue
			}

			// Get image info to determine natural dimensions
			imageInfo := pdf.RegisterImage(imagePath, "")
			if imageInfo == nil {
				log.Printf("Warning: Could not process image: %s", imagePath)
				continue
			}

			// Get natural image dimensions in points, then convert to mm
			naturalWidth, naturalHeight := imageInfo.Extent()
			// Convert from points to mm (1 point = 0.352778 mm)
			imageWidth := naturalWidth * 0.352778
			imageHeight := naturalHeight * 0.352778

			// Only scale down if image is wider than available width
			if imageWidth > availableWidth {
				scale := availableWidth / imageWidth
				imageWidth = availableWidth
				imageHeight *= scale
			}

			// Calculate available space on current page
			remainingHeight := pageHeight - footerHeight - margin - currentY
			spacing := 5.0

			// Check if we have enough space for the image
			if remainingHeight < imageHeight+spacing {
				pdf.AddPage()
				addFooter()
				currentY = margin
			}

			// Add image without scaling (unless it was too wide)
			pdf.ImageOptions(imagePath, margin, currentY, imageWidth, imageHeight, false, gofpdf.ImageOptions{}, 0, "")
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

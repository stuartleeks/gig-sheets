package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	schemaConfigFile string
	schemaOutputFile string
)

var generateSchemaCmd = &cobra.Command{
	Use:   "generate-schema",
	Short: "Generate JSON Schema for gig YAML files from config",
	Long: `Generate a JSON Schema file that provides VS Code autocomplete for gig YAML files.
The schema includes completion for song nicknames and image variants based on the config file.`,
	Run: runGenerateSchema,
}

func init() {
	generateSchemaCmd.Flags().StringVarP(&schemaConfigFile, "config", "c", "config.yaml", "Path to config YAML file")
	generateSchemaCmd.Flags().StringVarP(&schemaOutputFile, "output", "o", "gig-schema.json", "Output JSON Schema file path")
}

func runGenerateSchema(cmd *cobra.Command, args []string) {
	// Load configuration to extract song nicknames and image variants
	config, err := loadConfig(schemaConfigFile)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	// Generate the JSON Schema
	schema, err := generateJSONSchema(config)
	if err != nil {
		log.Fatalf("Error generating schema: %v", err)
	}

	// Write schema to file
	err = writeSchemaFile(schema, schemaOutputFile)
	if err != nil {
		log.Fatalf("Error writing schema file: %v", err)
	}

	fmt.Printf("Successfully generated JSON Schema: %s\n", schemaOutputFile)
	fmt.Println("To use in VS Code, add this to your settings.json:")

	absPath, err := filepath.Abs(schemaOutputFile)
	if err != nil {
		absPath = schemaOutputFile
	}
	fmt.Printf(`"yaml.schemas": {
  "%s": "*.yaml"
}`, absPath)
}

// JSONSchema represents the structure of a JSON Schema
type JSONSchema struct {
	Schema      string                 `json:"$schema"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Properties  map[string]interface{} `json:"properties"`
	Required    []string               `json:"required,omitempty"`
}

func generateJSONSchema(config *Config) (*JSONSchema, error) {
	// Extract song nicknames and their image variants
	songNicknames := make([]string, 0, len(config.Songs))
	songCompletions := make([]string, 0)

	for _, song := range config.Songs {
		songNicknames = append(songNicknames, song.Nickname)
		songCompletions = append(songCompletions, song.Nickname)

		// Add image variants for songs with multiple images
		if len(song.Images) > 1 {
			for variant := range song.Images {
				if variant != "default" {
					songCompletions = append(songCompletions, fmt.Sprintf("%s#%s", song.Nickname, variant))
				}
			}
		}
	}

	// Create the schema
	schema := &JSONSchema{
		Schema:      "http://json-schema.org/draft-07/schema#",
		Title:       "Gig Configuration Schema",
		Description: "Schema for gigsheets gig YAML files with autocomplete for songs and image variants",
		Type:        "object",
		Properties: map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the gig",
			},
			"sets": map[string]interface{}{
				"type":        "array",
				"description": "List of sets in the gig",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the set",
						},
						"songs": map[string]interface{}{
							"type":        "array",
							"description": "List of songs in the set",
							"items": map[string]interface{}{
								"type":        "string",
								"description": "Song nickname, optionally with image variant (e.g., 'song1' or 'song1#v2')",
								"enum":        songCompletions,
								"examples":    songCompletions[:min(10, len(songCompletions))], // Limit examples to first 10
							},
						},
					},
					"required": []string{"name", "songs"},
				},
			},
		},
		Required: []string{"name", "sets"},
	}

	return schema, nil
}

func writeSchemaFile(schema *JSONSchema, filename string) error {
	// Create output directory if it doesn't exist
	dir := filepath.Dir(filename)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Marshal schema to JSON with pretty formatting
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}

	// Write to file
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

# Gigsheets AI Agent Instructions

## Architecture Overview

**Gigsheets** is a CLI tool that generates PDF songsheets for musicians from YAML configurations. Two-file system:
- `config.yaml`: Maps song nicknames to image file paths (supports single `image` or multiple `images` with variants)
- `gig.yaml`: Defines performance structure with named sets containing song references (supports `song` or `song#variant` syntax)

**Core Components:**
- `main.go`: Entry point, delegates to cmd package
- `cmd/root.go`: Cobra CLI setup, registers all commands
- `cmd/generate.go`: PDF generation engine with intelligent image cropping
- `cmd/generate-schema.go`: JSON Schema generator for VS Code YAML autocomplete

## Data Model Patterns

**Song Configuration Evolution:**
```go
type Song struct {
    Nickname string            `yaml:"nickname"`
    Image    string            `yaml:"image,omitempty"`    // Legacy single image
    Images   map[string]string `yaml:"images,omitempty"`  // New multi-image support
}
```

**Image Resolution Logic:** Songs support both formats. The system builds a `songMap[nickname][imageName]string` where `imageName` defaults to `"default"`. Gig files reference songs as `songname` or `songname#variant`.

## Critical Workflows

**Development:**
```bash
go build -o gigsheets                    # Build binary
make example                            # Test with sample data
make build && make example              # Full rebuild + test
make lint                               # Ensure no linter errors
```

**Schema Generation Workflow:**
```bash
./gigsheets generate-schema --config config.yaml --output schema.json
# Outputs VS Code settings.json snippet for YAML autocomplete
```

## Image Processing Intelligence

**Smart Cropping Algorithm:** `cmd/generate.go` implements pixel-level analysis:
- `isWhiteOrTransparent()`: Detects background pixels (white/transparent with threshold=240)
- `findLeftMostContent()`, `findTopMostContent()`, `findBottomMostContent()`: Boundary detection
- `cropImage()`: In-memory cropping without temp files, supports PNG/JPEG/generic formats

**PDF Layout Logic:** Dynamically fits images per page, auto-starts new pages for sets or space constraints.

## Command Architecture Patterns

**Adding New Commands:**
1. Create `cmd/new-command.go` following the pattern in `generate-schema.go`
2. Define command with `cobra.Command` struct, flags via `init()`
3. Register in `cmd/root.go`: `rootCmd.AddCommand(newCmd)`
4. Use shared types from `generate.go` (`Config`, `Song`, `Gig`, `Set`)
5. **Update `README.md`**: Add usage documentation for the new command in the Usage section

**Modifying Existing Commands:** When changing command flags, arguments, or behavior, always update the corresponding usage examples and documentation in `README.md`.

**Error Handling Convention:** Use `log.Fatalf()` for user-facing errors, `fmt.Errorf()` with `%w` verb for wrapped errors.

## Schema Generation Specifics

**JSON Schema Generation:** `generateJSONSchema()` extracts song completions by:
1. Iterating config songs to build base nicknames
2. Adding `song#variant` entries for songs with multiple images (excluding "default")
3. Generating VS Code-compatible JSON Schema with `enum` and `examples` arrays

**VS Code Integration:** Generated schemas include exact settings.json configuration in command output using `filepath.Abs()` for correct path resolution.

## File Organization

- `example/`: Complete working sample (config.yaml, gig.yaml, images/)
- `cmd/`: All CLI commands as separate files
- `Makefile`: Standard build/test/example targets
- `SCHEMA_USAGE.md`: VS Code setup documentation

## Dependencies & Build

**Core Stack:**
- `github.com/spf13/cobra`: CLI framework
- `github.com/jung-kurt/gofpdf`: PDF generation
- `gopkg.in/yaml.v3`: YAML parsing
- `golang.org/x/image/draw`: Image manipulation

**No Test Framework:** Project currently has no test files or test patterns established.

## Key Implementation Details

**Backward Compatibility:** Always check both `song.Image` (string) and `song.Images` (map) when processing songs. Build unified `songMap` structure for consistent lookup.

**Image Variant Syntax:** `#` delimiter in gig files (`song2#v2`) maps to `Images["v2"]` in config. Missing variants fall back gracefully with warnings.

**Path Resolution:** Image paths in config can be relative (to config file directory) or absolute. Use `filepath.Join(configDir, imagePath)` for relative paths.
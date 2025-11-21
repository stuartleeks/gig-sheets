# Gigsheets CLI

A command-line tool for generating PDF song sheets from YAML configuration files.

## Overview

Gigsheets reads two YAML files:
- **Config file**: Maps song nicknames to image file paths
- **Gig file**: Defines sets of songs for a performance

It then generates a PDF containing all the song sheets, organized by sets.

## Installation

### Prerequisites
- Go 1.25.1 or later

### From Source
1. Clone this repository
2. Build the CLI:
   ```bash
   go build -o gigsheets
   ```

### From GitHub Releases
Download the latest pre-built binary from the [Releases page](../../releases) for your platform.


## Installation

Head to the [latest release page](https://github.com/stuartleeks/gig-sheets/releases/latest) and download the archive for your platform.

Extract `gigsheets` from the archive and place in a folder in your `PATH`.

Or if you just don't care and are happy to run random scripts from the internet:

```bash
export OS=linux # also darwin
export ARCH=amd64 # also 386
wget https://raw.githubusercontent.com/stuartleeks/gig-sheets/main/scripts/install.sh
chmod +x install.sh
sudo -E ./install.sh
```

## Enabling bash completion

To enable bash completion, add the following to you `~/.bashrc` file:

```bash
source <(gigsheets completion bash)
```

## Usage

### Basic Usage

```bash
./gigsheets generate --config config.yaml --gig gig.yaml --output example/output.pdf
```

### Command Options

- `--config, -c`: Path to config YAML file (default: "config.yaml")
- `--gig, -g`: Path to gig YAML file (default: "gig.yaml")
- `--output, -o`: Output PDF file path (default: "output.pdf")

### VS Code Autocomplete Support

Generate a JSON Schema for intelligent autocomplete when editing gig YAML files:

```bash
./gigsheets generate-schema --config config.yaml --output gig-schema.json
```

This creates a schema file that enables VS Code to provide:
- Autocomplete for song nicknames
- Autocomplete for image variants (e.g., `song#variant`)

To use the schema in VS Code, add to your settings.json:
```json
{
  "yaml.schemas": {
    "./gig-schema.json": "*.yaml"
  }
}
```

See [SCHEMA_USAGE.md](SCHEMA_USAGE.md) for detailed setup instructions.

### Configuration File Format

The config file maps song nicknames to image paths. Supports both single images and multiple image variants:

```yaml
songs:
  - nickname: song1
    image: images/song1.png  # Single image (backward compatibility)
  - nickname: song2
    images:  # Multiple named images
      default: images/song2.png
      v2: images/song2-v2.png
      simplified: images/song2-simple.png
```

### Gig File Format

The gig file defines sets of songs and includes the gig name:

```yaml
name: Sample Gig
sets:
  - name: set1
    songs:
      - song1
      - song2#v2  # Use specific image variant
  - name: set2
    songs:
      - song3
      - song4
```

## Example

See the `example/` directory for sample configuration and gig files.

To generate a PDF from the example files:

```bash
./gigsheets generate --config example/config.yaml --gig example/gig.yaml
```

## Features

- **Smart image cropping**: Automatically removes white/transparent space from the top, left, and bottom edges of images (in-memory processing)
- Combines song images efficiently on pages to save space
- Automatically starts new pages when switching sets or when space is insufficient
- Adds footers with gig name and page numbers
- Only scales images when they exceed page width (preserves natural dimensions)
- Supports PNG, JPEG, and other common image formats
- Provides clear error messages for missing files or songs
- Organizes songs by sets without separate title pages
- No temporary files created - all processing done in memory

## Release Process

This project uses [GoReleaser](https://goreleaser.com/) for automated builds and releases:

- **Automatic releases**: Every push to the `main` branch triggers a release with version `v0.1.<build_number>` (e.g., `v0.1.42`)
- **Cross-platform builds**: Automatically builds binaries for Linux, macOS, and Windows (both amd64 and arm64)
- **GitHub Releases**: Releases are automatically published to GitHub with checksums and changelog

### Creating a Release

Simply push to main to trigger an automatic release:
```bash
git push origin main
```

Each push will create a new release with an incrementing build number.

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [YAML v3](https://gopkg.in/yaml.v3) - YAML parsing
- [gofpdf](https://github.com/jung-kurt/gofpdf) - PDF generation
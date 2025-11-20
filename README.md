# Gigsheets CLI

A command-line tool for generating PDF song sheets from YAML configuration files.

## Overview

Gigsheets reads two YAML files:
- **Config file**: Maps song nicknames to image file paths
- **Gig file**: Defines sets of songs for a performance

It then generates a PDF containing all the song sheets, organized by sets.

## Installation

1. Clone this repository
2. Build the CLI:
   ```bash
   go build -o gigsheets
   ```

## Usage

### Basic Usage

```bash
./gigsheets generate --config config.yaml --gig gig.yaml --output example-output.pdf
```

### Command Options

- `--config, -c`: Path to config YAML file (default: "config.yaml")
- `--gig, -g`: Path to gig YAML file (default: "gig.yaml")
- `--output, -o`: Output PDF file path (default: "example-output.pdf")

### Configuration File Format

The config file maps song nicknames to image paths:

```yaml
songs:
  - nickname: song1
    image: images/song1.png
  - nickname: song2
    image: images/song2.png
```

### Gig File Format

The gig file defines sets of songs and includes the gig name:

```yaml
name: Sample Gig
sets:
  - name: set1
    songs:
      - song1
      - song2
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

- Combines song images efficiently on pages to save space
- Automatically starts new pages when switching sets or when space is insufficient
- Adds footers with gig name and page numbers
- Automatically scales images to fit available space
- Supports PNG, JPEG, and other common image formats
- Provides clear error messages for missing files or songs
- Organizes songs by sets without separate title pages

## Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [YAML v3](https://gopkg.in/yaml.v3) - YAML parsing
- [gofpdf](https://github.com/jung-kurt/gofpdf) - PDF generation
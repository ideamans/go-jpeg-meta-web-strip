# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go library specifically designed for optimizing JPEG images for web deployment. The library removes privacy-sensitive and unnecessary metadata while preserving essential information needed for proper web display. It uses a blacklist approach to selectively strip metadata that increases file sizes and may expose personal information, making images safer and more efficient for web usage.

## Build and Development Commands

```bash
# Build the package
go build ./...

# Run all tests
go test -v ./...

# Run a specific test
go test -v -run TestStrip/Remove_EXIF_thumbnail

# Generate test coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run linting (requires golangci-lint)
golangci-lint run --timeout=5m

# Generate test data (requires ImageMagick and ExifTool)
make data
# or
go run datacreator/cmd/main.go

# Clean test data
make clean
```

## Architecture

### Core Processing Flow

The library follows a segment-based JPEG processing approach:

1. **Parse JPEG Structure**: Uses `go-jpeg-image-structure` to parse JPEG into segments
2. **Segment Processing**: Each segment is evaluated by `processSegment()` function
3. **Selective Removal**: Segments are either kept, removed, or modified based on type
4. **Reconstruction**: Valid segments are reassembled into a new JPEG

### Key Components

- **strip.go**: Main processing logic for web optimization
  - `Strip()`: Entry point that orchestrates the metadata stripping process
  - `processSegment()`: Evaluates each JPEG segment
  - `processAPP1Segment()`: Handles EXIF/XMP segments specifically
  - `cleanExifSegment()`: Modifies EXIF data to remove thumbnails, GPS, and camera info
  - Binary EXIF parsing functions for TIFF/IFD structure manipulation

- **datacreator/**: Test data generation utility
  - Creates 18+ different JPEG variations with various metadata combinations
  - Uses ImageMagick for basic image operations
  - Uses ExifTool for metadata embedding (thumbnails, GPS, XMP, IPTC)
  - Creates test cases that simulate real-world web optimization scenarios

### Metadata Handling Strategy

**Removed (Blacklist)**:
- APP1: EXIF thumbnails (IFD1), GPS data (GPS IFD), camera info (Make/Model tags)
- APP1: XMP data (identified by Adobe namespace header)
- APP13: Photoshop IRB/IPTC data
- COM: Comment markers

**Preserved**:
- APP1: Core EXIF data (orientation, resolution)
- APP2: ICC color profiles
- APP14: Adobe color transform information
- All image data segments (SOF, DQT, DHT, SOS, SOI, EOI)

### Binary EXIF Processing

The library directly manipulates EXIF binary data:
- Detects endianness (big/little) from TIFF header
- Navigates IFD (Image File Directory) structures
- Sets IFD1 offset to 0 to remove thumbnails
- Zeros out specific tag entries for GPS/camera removal

## Code Quality Standards

The project enforces strict linting via `.golangci.yml`:
- Cyclomatic complexity limit: 15
- Required formatting: gofmt -s, goimports, gofumpt
- Security scanning with gosec
- Comprehensive static analysis

When modifying EXIF parsing code, be aware of:
- Proper endianness handling for multi-byte values
- IFD offset calculations must account for TIFF header position
- Tag removal by zeroing entries (not deleting) to maintain structure

## Testing Approach

Tests verify both metadata removal and image integrity:
- Metadata verification uses ExifTool output parsing
- Image integrity verified via pixel data MD5 checksums
- Test data covers edge cases (mixed metadata, ICC+thumbnail, etc.)

## Module Naming Note

The module is named `go-jpeg-meta-web-strip` in go.mod and the package is `jpegmetawebstrip`. The main function is `Strip()` which returns `(*Result, error)`. The tool is specifically designed for preparing JPEG images for web deployment by removing unnecessary metadata while preserving display-critical information.
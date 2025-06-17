package jpegmetawebstrip

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStrip(t *testing.T) {
	testCases := []struct {
		name           string
		inputFile      string
		shouldRemove   []string
		shouldPreserve []string
	}{
		{
			name:           "Remove EXIF thumbnail",
			inputFile:      "with_exif_thumbnail.jpg",
			shouldRemove:   []string{"ThumbnailImage"},
			shouldPreserve: []string{"Orientation", "ColorSpace"},
		},
		{
			name:           "Remove GPS data",
			inputFile:      "with_gps.jpg",
			shouldRemove:   []string{"GPS"},
			shouldPreserve: []string{"Orientation", "ColorSpace"},
		},
		{
			name:           "Remove camera info",
			inputFile:      "with_camera_info.jpg",
			shouldRemove:   []string{"Make", "Model"},
			shouldPreserve: []string{"Orientation", "ColorSpace"},
		},
		{
			name:           "Remove XMP metadata",
			inputFile:      "with_xmp.jpg",
			shouldRemove:   []string{"XMP"},
			shouldPreserve: []string{"Orientation", "ColorSpace"},
		},
		{
			name:           "Remove IPTC metadata",
			inputFile:      "with_iptc.jpg",
			shouldRemove:   []string{"IPTC"},
			shouldPreserve: []string{"Orientation", "ColorSpace"},
		},
		{
			name:           "Remove comment",
			inputFile:      "with_comment.jpg",
			shouldRemove:   []string{"Comment"},
			shouldPreserve: []string{"Orientation", "ColorSpace"},
		},
		{
			name:           "Preserve orientation",
			inputFile:      "with_orientation.jpg",
			shouldRemove:   []string{},
			shouldPreserve: []string{"Orientation"},
		},
		{
			name:           "Preserve ICC profile",
			inputFile:      "with_icc_profile_srgb.jpg",
			shouldRemove:   []string{},
			shouldPreserve: []string{"ProfileDescription", "ColorSpace"},
		},
		{
			name:           "Preserve DPI",
			inputFile:      "with_dpi.jpg",
			shouldRemove:   []string{},
			shouldPreserve: []string{"XResolution", "YResolution"},
		},
		{
			name:           "Preserve gamma",
			inputFile:      "with_gamma.jpg",
			shouldRemove:   []string{},
			shouldPreserve: []string{"Gamma"},
		},
		{
			name:           "Mixed metadata",
			inputFile:      "with_mixed_metadata.jpg",
			shouldRemove:   []string{"GPS", "XMP"},
			shouldPreserve: []string{"ProfileDescription", "ColorSpace"},
		},
		{
			name:           "Comprehensive mixed metadata",
			inputFile:      "with_comprehensive_mixed.jpg",
			shouldRemove:   []string{"ThumbnailImage", "GPS", "Make", "Model", "Lens", "XMP", "IPTC"},
			shouldPreserve: []string{"XResolution", "YResolution", "ImageWidth", "ImageHeight"},
		},
		{
			name:           "Thumbnail with ICC profile",
			inputFile:      "with_thumbnail_and_icc.jpg",
			shouldRemove:   []string{"ThumbnailImage", "ThumbnailOffset", "ThumbnailLength"},
			shouldPreserve: []string{"ProfileDescription", "ProfileClass", "ProfileCreator", "ColorSpace"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Read test file
			inputPath := filepath.Join("testdata", tc.inputFile)
			jpegData, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("Failed to read test file %s: %v", tc.inputFile, err)
			}

			// Get original metadata
			originalMeta := getImageMetadata(t, jpegData)
			t.Logf("Original metadata keys: %v", getMetadataKeys(originalMeta))

			// Process with Strip
			cleanedData, result, err := Strip(jpegData)
			if err != nil {
				t.Fatalf("Strip failed: %v", err)
			}

			// Get cleaned metadata
			cleanedMeta := getImageMetadata(t, cleanedData)
			t.Logf("Cleaned metadata keys: %v", getMetadataKeys(cleanedMeta))

			// Log removal results
			t.Logf("Removal results: ExifThumbnail=%d, GPS=%d, Camera=%d, XMP=%d, IPTC=%d, PhotoshopIRB=%d, Comments=%d",
				result.Removed.ExifThumbnail, result.Removed.ExifGPS, result.Removed.CameraInfo,
				result.Removed.XMP, result.Removed.IPTC, result.Removed.PhotoshopIRB, result.Removed.Comments)
			t.Logf("Total removed: %d bytes", result.Total)

			// Verify that data was removed
			for _, tag := range tc.shouldRemove {
				if containsMetadata(cleanedMeta, tag) {
					t.Errorf("Expected %s to be removed, but it still exists", tag)
				}
			}

			// Verify that important data was preserved
			for _, tag := range tc.shouldPreserve {
				if !containsMetadata(cleanedMeta, tag) && containsMetadata(originalMeta, tag) {
					t.Errorf("Expected %s to be preserved, but it was removed", tag)
				}
			}

			// Verify the cleaned JPEG is still valid
			if !isValidJPEG(cleanedData) {
				t.Error("Cleaned JPEG is not valid")
			}

			// Check file size reduction
			originalSize := len(jpegData)
			cleanedSize := len(cleanedData)
			if cleanedSize > originalSize {
				t.Errorf("Cleaned size (%d) is larger than original (%d)", cleanedSize, originalSize)
			}
			t.Logf("Size reduction: %d bytes (%.2f%%)",
				originalSize-cleanedSize,
				float64(originalSize-cleanedSize)/float64(originalSize)*100)
		})
	}
}

// getImageMetadata uses exiftool to get metadata (if available)
func getImageMetadata(t *testing.T, jpegData []byte) string {
	// Check if exiftool is available
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("exiftool not found, skipping metadata verification")
		return ""
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "test-*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(jpegData); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Run exiftool
	cmd := exec.Command("exiftool", tmpFile.Name())
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run exiftool: %v", err)
	}

	return string(output)
}

// getMetadataKeys extracts metadata field names from exiftool output
func getMetadataKeys(metadata string) []string {
	keys := []string{}
	lines := strings.Split(metadata, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// containsMetadata checks if metadata contains a specific tag
func containsMetadata(metadata, tag string) bool {
	lines := strings.Split(metadata, "\n")
	for _, line := range lines {
		// Check if line starts with the tag name (considering exiftool format)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) >= 1 {
			fieldName := strings.TrimSpace(parts[0])
			if strings.Contains(fieldName, tag) {
				return true
			}
		}
	}
	return false
}

// isValidJPEG checks if the data is a valid JPEG
func isValidJPEG(data []byte) bool {
	// Check JPEG magic numbers
	if len(data) < 2 {
		return false
	}
	// JPEG starts with 0xFFD8
	return data[0] == 0xFF && data[1] == 0xD8
}

func TestStripInvalidData(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{"Empty data", []byte{}},
		{"Not JPEG", []byte("This is not a JPEG")},
		{"Truncated JPEG", []byte{0xFF, 0xD8}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Strip(tc.data)
			if err == nil {
				t.Error("Expected error for invalid data, but got nil")
			}
		})
	}
}

// TestJpegDecodeIntegrity verifies that JPEG decoding produces identical results before and after metadata removal
func TestJpegDecodeIntegrity(t *testing.T) {
	testFiles := []string{
		"basic_copy.jpg",
		"with_exif_thumbnail.jpg",
		"with_gps.jpg",
		"with_camera_info.jpg",
		"with_xmp.jpg",
		"with_iptc.jpg",
		"with_comment.jpg",
		"with_orientation.jpg",
		"with_icc_profile_srgb.jpg",
		"with_dpi.jpg",
		"with_gamma.jpg",
		"with_comprehensive_mixed.jpg",
		"with_thumbnail_and_icc.jpg",
	}

	for _, filename := range testFiles {
		t.Run(filename, func(t *testing.T) {
			// Read test file
			inputPath := filepath.Join("testdata", filename)
			jpegData, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("Failed to read test file %s: %v", filename, err)
			}

			// Decode original JPEG
			originalChecksum, err := getJPEGPixelChecksum(jpegData)
			if err != nil {
				t.Fatalf("Failed to decode original JPEG: %v", err)
			}

			// Process with Strip
			cleanedData, _, err := Strip(jpegData)
			if err != nil {
				t.Fatalf("Strip failed: %v", err)
			}

			// Decode cleaned JPEG
			cleanedChecksum, err := getJPEGPixelChecksum(cleanedData)
			if err != nil {
				t.Fatalf("Failed to decode cleaned JPEG: %v", err)
			}

			// Compare checksums
			if originalChecksum != cleanedChecksum {
				t.Errorf("Pixel data checksum mismatch: original=%s, cleaned=%s", originalChecksum, cleanedChecksum)
			} else {
				t.Logf("✓ Pixel data integrity preserved (checksum: %s)", originalChecksum)
			}
		})
	}
}

// getJPEGPixelChecksum decodes a JPEG and returns MD5 checksum of pixel data
func getJPEGPixelChecksum(jpegData []byte) (string, error) {
	// Decode JPEG
	img, err := jpeg.Decode(bytes.NewReader(jpegData))
	if err != nil {
		return "", fmt.Errorf("failed to decode JPEG: %w", err)
	}

	// Get image bounds
	bounds := img.Bounds()

	// Create checksum of pixel data
	hasher := md5.New()

	// Write dimensions to hasher to ensure size consistency
	fmt.Fprintf(hasher, "%d,%d", bounds.Dx(), bounds.Dy())

	// Iterate through all pixels
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// Write pixel values to hasher
			fmt.Fprintf(hasher, ",%d,%d,%d,%d", r, g, b, a)
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

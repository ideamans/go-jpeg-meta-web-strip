package datacreator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	originalImage = "original.jpg"
	testdataDir   = "./testdata"
	// ThumbnailImageTag is the ExifTool tag for setting thumbnail images
	ThumbnailImageTag = "-ThumbnailImage<="
)

type TestImage struct {
	Name        string
	Description string
	Command     []string
	UseExiftool bool
}

func Run() error {
	if err := ensureTestdataDir(); err != nil {
		return fmt.Errorf("failed to ensure testdata directory: %w", err)
	}

	originalPath := filepath.Join("datacreator", originalImage)
	if _, err := os.Stat(originalPath); err != nil {
		return fmt.Errorf("original image not found at %s: %w", originalPath, err)
	}

	images := getTestImages()
	for _, img := range images {
		if err := generateImage(originalPath, img); err != nil {
			return fmt.Errorf("failed to generate %s: %w", img.Name, err)
		}
		fmt.Printf("Generated: %s - %s\n", img.Name, img.Description)
	}

	// Generate EXIF thumbnail separately
	if err := generateExifThumbnail(originalPath); err != nil {
		fmt.Printf("Warning: Could not generate EXIF thumbnail: %v\n", err)
	}

	// Generate XMP and IPTC metadata using exiftool
	if err := generateXMPAndIPTC(originalPath); err != nil {
		fmt.Printf("Warning: Could not generate XMP/IPTC metadata: %v\n", err)
	}

	// Generate ICC profile variations
	if err := generateICCProfiles(originalPath); err != nil {
		fmt.Printf("Warning: Could not generate ICC profile variations: %v\n", err)
	}

	// Generate comprehensive mixed metadata test
	if err := generateComprehensiveMixedMetadata(originalPath); err != nil {
		fmt.Printf("Warning: Could not generate comprehensive mixed metadata: %v\n", err)
	}

	// Generate thumbnail with ICC profile test
	if err := generateThumbnailWithICC(originalPath); err != nil {
		fmt.Printf("Warning: Could not generate thumbnail with ICC test: %v\n", err)
	}

	return nil
}

func ensureTestdataDir() error {
	if err := os.MkdirAll(testdataDir, 0o755); err != nil {
		return fmt.Errorf("failed to create testdata directory: %w", err)
	}
	return nil
}

func generateImage(originalPath string, img TestImage) error {
	outputPath := filepath.Join(testdataDir, img.Name)

	// Build the command properly
	args := append([]string{originalPath}, img.Command...)
	args = append(args, outputPath)

	cmd := exec.Command("magick", args...)
	cmd.Dir = "." // Run from current directory
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ImageMagick command failed: %w\nOutput: %s", err, output)
	}

	return nil
}

func getTestImages() []TestImage {
	return []TestImage{
		// Basic copy for testing
		{
			Name:        "basic_copy.jpg",
			Description: "Basic copy of original",
			Command:     []string{},
		},

		// Images with metadata to be removed
		{
			Name:        "with_gps.jpg",
			Description: "JPEG with GPS data",
			Command:     []string{"-set", "EXIF:GPSLatitude", "40.7142", "-set", "EXIF:GPSLongitude", "-74.0064"},
		},
		{
			Name:        "with_camera_info.jpg",
			Description: "JPEG with camera information",
			Command:     []string{"-set", "EXIF:Make", "Canon", "-set", "EXIF:Model", "EOS 5D Mark IV"},
		},
		{
			Name:        "with_comment.jpg",
			Description: "JPEG with comment",
			Command:     []string{"-comment", "This is a test comment"},
		},

		// Images with metadata to be preserved
		{
			Name:        "with_orientation.jpg",
			Description: "JPEG with orientation (should be preserved)",
			Command:     []string{"-rotate", "90"},
		},
		{
			Name:        "with_dpi.jpg",
			Description: "JPEG with DPI settings (should be preserved)",
			Command:     []string{"-density", "300x300", "-units", "PixelsPerInch"},
		},
		{
			Name:        "with_colorspace.jpg",
			Description: "JPEG with specific colorspace (should be preserved)",
			Command:     []string{"-colorspace", "sRGB"},
		},
		{
			Name:        "with_quality.jpg",
			Description: "JPEG with specific quality",
			Command:     []string{"-quality", "95"},
		},
		{
			Name:        "with_gamma.jpg",
			Description: "JPEG with gamma value (should be preserved)",
			Command:     []string{"-set", "gamma", "2.2"},
		},
	}
}

func generateExifThumbnail(originalPath string) error {
	outputPath := filepath.Join(testdataDir, "with_exif_thumbnail.jpg")
	tempThumb := filepath.Join(testdataDir, "temp_thumb.jpg")

	// First, copy the original
	copyCmd := exec.Command("magick", originalPath, outputPath)
	if output, err := copyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy original: %w\nOutput: %s", err, output)
	}

	// Create a small thumbnail
	thumbCmd := exec.Command("magick", originalPath, "-thumbnail", "160x120", tempThumb)
	if output, err := thumbCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create thumbnail: %w\nOutput: %s", err, output)
	}

	// Try to embed thumbnail using exiftool if available
	if _, err := exec.LookPath("exiftool"); err == nil {
		// #nosec G204 - exiftool is a trusted tool and tempThumb is generated internally
		exifCmd := exec.Command("exiftool", ThumbnailImageTag+tempThumb, "-overwrite_original", outputPath)
		if output, err := exifCmd.CombinedOutput(); err != nil {
			// Clean up temp file
			os.Remove(tempThumb)
			return fmt.Errorf("failed to embed thumbnail with exiftool: %w\nOutput: %s", err, output)
		}
		fmt.Printf("Generated: with_exif_thumbnail.jpg - JPEG with EXIF thumbnail\n")
	} else {
		// If exiftool is not available, try alternative method with ImageMagick
		// This creates a JPEG with embedded thumbnail in the EXIF data
		embedCmd := exec.Command("magick", originalPath,
			"-write", "mpr:orig",
			"-thumbnail", "160x120",
			"-write", tempThumb,
			"+delete",
			"mpr:orig",
			"-set", "profile:exif-thumbnail", tempThumb,
			outputPath)
		if _, err := embedCmd.CombinedOutput(); err != nil {
			// If this also fails, just keep the file without thumbnail
			fmt.Printf("Note: Could not embed EXIF thumbnail (exiftool not found)\n")
		} else {
			fmt.Printf("Generated: with_exif_thumbnail.jpg - JPEG with EXIF thumbnail (via ImageMagick)\n")
		}
	}

	// Clean up temp file
	os.Remove(tempThumb)

	return nil
}

func generateXMPAndIPTC(originalPath string) error {
	// Check if exiftool is available
	if _, err := exec.LookPath("exiftool"); err != nil {
		return fmt.Errorf("exiftool not found")
	}

	// Generate JPEG with XMP metadata
	xmpOutput := filepath.Join(testdataDir, "with_xmp.jpg")
	copyCmd := exec.Command("magick", originalPath, xmpOutput)
	if output, err := copyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy for XMP: %w\nOutput: %s", err, output)
	}

	xmpCmd := exec.Command("exiftool",
		"-XMP:Creator=Test Creator",
		"-XMP:CreatorTool=Adobe Photoshop",
		"-XMP:CreateDate=2024-01-01T12:00:00",
		"-XMP:ModifyDate=2024-01-01T14:00:00",
		"-XMP:MetadataDate=2024-01-01T14:00:00",
		"-XMP:Label=Test Label",
		"-XMP:Rating=5",
		"-XMP:Subject=test,sample,jpeg",
		"-overwrite_original",
		xmpOutput)
	if output, err := xmpCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add XMP metadata: %w\nOutput: %s", err, output)
	}
	fmt.Printf("Generated: with_xmp.jpg - JPEG with XMP metadata\n")

	// Generate JPEG with IPTC metadata
	iptcOutput := filepath.Join(testdataDir, "with_iptc.jpg")
	copyCmd2 := exec.Command("magick", originalPath, iptcOutput)
	if output, err := copyCmd2.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy for IPTC: %w\nOutput: %s", err, output)
	}

	iptcCmd := exec.Command("exiftool",
		"-IPTC:Caption-Abstract=Test Caption",
		"-IPTC:Keywords=test,sample,jpeg",
		"-IPTC:By-line=Test Photographer",
		"-IPTC:CopyrightNotice=Copyright 2024 Test",
		"-IPTC:City=Tokyo",
		"-IPTC:Country-PrimaryLocationName=Japan",
		"-IPTC:DateCreated=2024:01:01",
		"-IPTC:TimeCreated=12:00:00",
		"-overwrite_original",
		iptcOutput)
	if output, err := iptcCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add IPTC metadata: %w\nOutput: %s", err, output)
	}
	fmt.Printf("Generated: with_iptc.jpg - JPEG with IPTC metadata\n")

	// Generate JPEG with Photoshop IRB metadata
	irbOutput := filepath.Join(testdataDir, "with_photoshop_irb.jpg")
	copyCmd3 := exec.Command("magick", originalPath, irbOutput)
	if output, err := copyCmd3.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy for IRB: %w\nOutput: %s", err, output)
	}

	irbCmd := exec.Command("exiftool",
		"-Photoshop:IPTCDigest=00000000000000000000000000000000",
		"-Photoshop:PhotoshopQuality=12",
		"-Photoshop:PhotoshopFormat=Standard",
		"-Photoshop:ProgressiveScans=3",
		"-overwrite_original",
		irbOutput)
	if output, err := irbCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add Photoshop IRB metadata: %w\nOutput: %s", err, output)
	}
	fmt.Printf("Generated: with_photoshop_irb.jpg - JPEG with Photoshop IRB metadata\n")

	// Generate JPEG with all removable metadata combined
	allOutput := filepath.Join(testdataDir, "with_all_removable.jpg")
	copyCmd4 := exec.Command("magick", originalPath, allOutput)
	if output, err := copyCmd4.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy for all metadata: %w\nOutput: %s", err, output)
	}

	// First create thumbnail
	tempThumb := filepath.Join(testdataDir, "temp_thumb2.jpg")
	thumbCmd := exec.Command("magick", originalPath, "-thumbnail", "160x120", tempThumb)
	if output, err := thumbCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create thumbnail for all: %w\nOutput: %s", err, output)
	}

	// #nosec G204 - exiftool is a trusted tool and tempThumb is generated internally
	allCmd := exec.Command("exiftool",
		ThumbnailImageTag+tempThumb,
		"-GPS:GPSLatitude=40.7142",
		"-GPS:GPSLongitude=-74.0064",
		"-EXIF:Make=Canon",
		"-EXIF:Model=EOS 5D Mark IV",
		"-XMP:CreatorTool=Test Tool",
		"-IPTC:Caption-Abstract=Test Caption",
		"-Photoshop:PhotoshopQuality=12",
		"-Comment=Test Comment",
		"-overwrite_original",
		allOutput)
	if output, err := allCmd.CombinedOutput(); err != nil {
		os.Remove(tempThumb)
		return fmt.Errorf("failed to add all metadata: %w\nOutput: %s", err, output)
	}
	os.Remove(tempThumb)
	fmt.Printf("Generated: with_all_removable.jpg - JPEG with all removable metadata\n")

	return nil
}

func generateICCProfiles(originalPath string) error {
	// Generate JPEG with sRGB ICC profile
	srgbProfile := filepath.Join("datacreator", "sRGB-v2-micro.icc")
	srgbOutput := filepath.Join(testdataDir, "with_icc_profile_srgb.jpg")

	srgbCmd := exec.Command("magick", originalPath, "-profile", srgbProfile, srgbOutput)
	if output, err := srgbCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to embed sRGB ICC profile: %w\nOutput: %s", err, output)
	}
	fmt.Printf("Generated: with_icc_profile_srgb.jpg - JPEG with sRGB ICC profile (should be preserved)\n")

	// Generate JPEG with Display P3 ICC profile
	p3Profile := filepath.Join("datacreator", "DisplayP3-v2-micro.icc")
	p3Output := filepath.Join(testdataDir, "with_icc_profile_p3.jpg")

	p3Cmd := exec.Command("magick", originalPath, "-profile", p3Profile, p3Output)
	if output, err := p3Cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to embed Display P3 ICC profile: %w\nOutput: %s", err, output)
	}
	fmt.Printf("Generated: with_icc_profile_p3.jpg - JPEG with Display P3 ICC profile (should be preserved)\n")

	// Generate JPEG with mixed metadata (removable + ICC profile to keep)
	mixedOutput := filepath.Join(testdataDir, "with_mixed_metadata.jpg")
	mixedCmd := exec.Command("magick", originalPath,
		"-profile", srgbProfile,
		"-set", "comment", "Test comment to remove",
		"-set", "EXIF:Make", "Test Camera",
		mixedOutput)
	if output, err := mixedCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create mixed metadata: %w\nOutput: %s", err, output)
	}

	// Add more removable metadata using exiftool if available
	if _, err := exec.LookPath("exiftool"); err == nil {
		exifCmd := exec.Command("exiftool",
			"-GPS:GPSLatitude=35.6762",
			"-GPS:GPSLongitude=139.6503",
			"-XMP:CreatorTool=Test Tool",
			"-IPTC:Caption-Abstract=Test Caption",
			"-overwrite_original",
			mixedOutput)
		if _, err := exifCmd.CombinedOutput(); err != nil {
			fmt.Printf("Warning: Could not add additional metadata to mixed file: %v\n", err)
		}
	}

	fmt.Printf("Generated: with_mixed_metadata.jpg - JPEG with both removable and preservable metadata\n")

	return nil
}

func generateComprehensiveMixedMetadata(originalPath string) error {
	outputPath := filepath.Join(testdataDir, "with_comprehensive_mixed.jpg")

	// First, create image with orientation and DPI
	cmd := exec.Command("magick", originalPath,
		"-rotate", "90",
		"-density", "300x300",
		"-units", "PixelsPerInch",
		outputPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create base image: %w\nOutput: %s", err, output)
	}

	// Add EXIF thumbnail using exiftool
	if _, err := exec.LookPath("exiftool"); err == nil {
		// Create thumbnail
		tempThumb := filepath.Join(testdataDir, "temp_thumb_mixed.jpg")
		thumbCmd := exec.Command("magick", originalPath, "-thumbnail", "160x120", tempThumb)
		if output, err := thumbCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create thumbnail: %w\nOutput: %s", err, output)
		}

		// Add comprehensive metadata including thumbnail
		// #nosec G204 - exiftool is a trusted tool and tempThumb is generated internally
		exifCmd := exec.Command("exiftool",
			ThumbnailImageTag+tempThumb,
			"-GPS:GPSLatitude=51.5074",
			"-GPS:GPSLongitude=-0.1278",
			"-GPS:GPSAltitude=100",
			"-EXIF:Make=TestCamera",
			"-EXIF:Model=TestModel X1",
			"-EXIF:LensModel=TestLens 50mm",
			"-XMP:CreatorTool=TestSoftware",
			"-IPTC:Caption-Abstract=Test Caption",
			"-Comment=Comprehensive test",
			"-overwrite_original",
			outputPath)
		if output, err := exifCmd.CombinedOutput(); err != nil {
			os.Remove(tempThumb)
			return fmt.Errorf("failed to add metadata: %w\nOutput: %s", err, output)
		}
		os.Remove(tempThumb)

		fmt.Printf("Generated: with_comprehensive_mixed.jpg - JPEG with comprehensive mixed metadata (removable + preservable)\n")
	}

	return nil
}

func generateThumbnailWithICC(originalPath string) error {
	outputPath := filepath.Join(testdataDir, "with_thumbnail_and_icc.jpg")
	srgbProfile := filepath.Join("datacreator", "sRGB-v2-micro.icc")

	// First, create image with ICC profile
	cmd := exec.Command("magick", originalPath,
		"-profile", srgbProfile,
		outputPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create image with ICC: %w\nOutput: %s", err, output)
	}

	// Add EXIF thumbnail using exiftool
	if _, err := exec.LookPath("exiftool"); err == nil {
		// Create thumbnail
		tempThumb := filepath.Join(testdataDir, "temp_thumb_icc.jpg")
		thumbCmd := exec.Command("magick", originalPath, "-thumbnail", "160x120", tempThumb)
		if output, err := thumbCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create thumbnail: %w\nOutput: %s", err, output)
		}

		// Add thumbnail while preserving ICC profile
		// #nosec G204 - exiftool is a trusted tool and tempThumb is generated internally
		exifCmd := exec.Command("exiftool",
			ThumbnailImageTag+tempThumb,
			"-overwrite_original",
			outputPath)
		if output, err := exifCmd.CombinedOutput(); err != nil {
			os.Remove(tempThumb)
			return fmt.Errorf("failed to add thumbnail: %w\nOutput: %s", err, output)
		}
		os.Remove(tempThumb)

		fmt.Printf("Generated: with_thumbnail_and_icc.jpg - JPEG with EXIF thumbnail and ICC profile\n")
	} else {
		fmt.Printf("Generated: with_thumbnail_and_icc.jpg - JPEG with ICC profile (no thumbnail, exiftool not found)\n")
	}

	return nil
}

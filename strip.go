package jpegmetawebstrip

import (
	"bytes"
	"encoding/binary"
	"fmt"

	jpegstructure "github.com/dsoprea/go-jpeg-image-structure/v2"
)

const (
	// ExifHeader is the standard EXIF header
	ExifHeader = "Exif\x00\x00"
)

// Result contains information about removed metadata
type Result struct {
	Removed struct {
		ExifThumbnail int64
		ExifGPS       int64
		CameraInfo    int64
		XMP           int64
		IPTC          int64
		PhotoshopIRB  int64
		Comments      int64
	}
	Total int64
}

// Strip removes unnecessary metadata from JPEG data for web optimization while preserving display-critical information
func Strip(jpegData []byte) ([]byte, *Result, error) {
	result := &Result{}

	// Parse JPEG structure
	jmp := jpegstructure.NewJpegMediaParser()
	intfc, err := jmp.ParseBytes(jpegData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse JPEG: %w", err)
	}

	sl, ok := intfc.(*jpegstructure.SegmentList)
	if !ok {
		return nil, nil, fmt.Errorf("failed to get segment list")
	}

	// Create new segment list for cleaned JPEG
	newSegments := make([]*jpegstructure.Segment, 0)

	// Iterate through segments and filter out unwanted metadata
	for _, segment := range sl.Segments() {
		processedSegment, keep := processSegment(segment, result)
		if keep {
			newSegments = append(newSegments, processedSegment)
		}
	}

	// Create new segment list
	newSl := jpegstructure.NewSegmentList(newSegments)

	// Write cleaned JPEG
	b := new(bytes.Buffer)
	err = newSl.Write(b)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write cleaned JPEG: %w", err)
	}

	return b.Bytes(), result, nil
}

// processSegment processes a single JPEG segment and determines if it should be kept
func processSegment(segment *jpegstructure.Segment, result *Result) (*jpegstructure.Segment, bool) {
	removedSize := int64(len(segment.Data))

	switch segment.MarkerId {
	case jpegstructure.MARKER_APP1: // EXIF/XMP
		return processAPP1Segment(segment, result, removedSize)

	case jpegstructure.MARKER_APP13: // Photoshop IRB/IPTC
		result.Removed.PhotoshopIRB += removedSize
		result.Total += removedSize
		return segment, false

	case jpegstructure.MARKER_COM: // Comment
		result.Removed.Comments += removedSize
		result.Total += removedSize
		return segment, false

	case jpegstructure.MARKER_APP2, // ICC Profile
		jpegstructure.MARKER_APP14,                                                      // Adobe
		jpegstructure.MARKER_SOF0, jpegstructure.MARKER_SOF1, jpegstructure.MARKER_SOF2, // Start of Frame
		jpegstructure.MARKER_DQT, jpegstructure.MARKER_DHT, // Quantization and Huffman tables
		jpegstructure.MARKER_SOS,                           // Start of Scan
		jpegstructure.MARKER_SOI, jpegstructure.MARKER_EOI: // Start/End of Image
		// Keep these segments
		return segment, true

	default:
		// Keep unknown segments by default
		return segment, true
	}
}

// processAPP1Segment processes APP1 segments (EXIF/XMP)
func processAPP1Segment(segment *jpegstructure.Segment, result *Result, removedSize int64) (*jpegstructure.Segment, bool) {
	if isXMPSegment(segment) {
		// Remove XMP metadata
		result.Removed.XMP += removedSize
		result.Total += removedSize
		return segment, false
	}

	if isExifSegment(segment) {
		// Process EXIF data to remove thumbnails and other unwanted data
		cleanedExif, modified, removedBytes := cleanExifSegment(segment.Data, result)
		if modified {
			// Create new segment with cleaned EXIF data
			newSegment := &jpegstructure.Segment{
				MarkerId:   segment.MarkerId,
				MarkerName: segment.MarkerName,
				Offset:     segment.Offset,
				Data:       cleanedExif,
			}
			result.Total += removedBytes
			return newSegment, true
		}
		return segment, true
	}

	// Keep other APP1 segments
	return segment, true
}

// isExifSegment checks if the APP1 segment contains EXIF data
func isExifSegment(segment *jpegstructure.Segment) bool {
	if len(segment.Data) < 6 {
		return false
	}
	// Check for EXIF header
	return bytes.HasPrefix(segment.Data, []byte(ExifHeader))
}

// isXMPSegment checks if the APP1 segment contains XMP data
func isXMPSegment(segment *jpegstructure.Segment) bool {
	if len(segment.Data) < 29 {
		return false
	}
	// Check for "http://ns.adobe.com/xap/1.0/\x00" header
	return bytes.HasPrefix(segment.Data, []byte("http://ns.adobe.com/xap/1.0/\x00"))
}

// cleanExifSegment removes unwanted data from EXIF segment
func cleanExifSegment(exifData []byte, result *Result) ([]byte, bool, int64) {
	// First try to remove thumbnail
	cleanedData, thumbRemoved, thumbSize, err := removeThumbnailFromExif(exifData)
	if err != nil {
		// If error, return original data
		return exifData, false, 0
	}

	totalRemoved := int64(0)
	if thumbRemoved {
		result.Removed.ExifThumbnail += thumbSize
		totalRemoved += thumbSize
		exifData = cleanedData
	}

	// Then remove GPS data
	cleanedData, gpsRemoved, gpsSize := removeGPSFromExif(exifData)
	if gpsRemoved {
		result.Removed.ExifGPS += gpsSize
		totalRemoved += gpsSize
		exifData = cleanedData
	}

	// Remove camera-specific data
	cleanedData, camRemoved, camSize := removeCameraInfoFromExif(exifData)
	if camRemoved {
		result.Removed.CameraInfo += camSize
		totalRemoved += camSize
		exifData = cleanedData
	}

	return exifData, totalRemoved > 0, totalRemoved
}

// removeThumbnailFromExif removes thumbnail from EXIF segment data
func removeThumbnailFromExif(exifData []byte) ([]byte, bool, int64, error) {
	if len(exifData) < 6 || string(exifData[0:6]) != ExifHeader {
		return exifData, false, 0, fmt.Errorf("invalid EXIF header")
	}
	// Simple implementation: just set IFD1 offset to 0
	// TIFF header starts from byte 6
	pos := 6
	if len(exifData) < pos+8 {
		return exifData, false, 0, fmt.Errorf("invalid TIFF header")
	}
	byteOrder := binary.BigEndian.Uint16(exifData[pos : pos+2])
	littleEndian := byteOrder == 0x4949
	var readUint16 func([]byte) uint16
	var readUint32 func([]byte) uint32
	if littleEndian {
		readUint16 = func(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
		readUint32 = func(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
	} else {
		readUint16 = func(b []byte) uint16 { return binary.BigEndian.Uint16(b) }
		readUint32 = func(b []byte) uint32 { return binary.BigEndian.Uint32(b) }
	}
	ifd0Offset := int(readUint32(exifData[pos+4 : pos+8]))
	ifd0Pos := pos + ifd0Offset
	if len(exifData) < ifd0Pos+2 {
		return exifData, false, 0, fmt.Errorf("invalid IFD0")
	}
	entryCount := int(readUint16(exifData[ifd0Pos : ifd0Pos+2]))
	ifd1OffsetPos := ifd0Pos + 2 + entryCount*12
	if len(exifData) < ifd1OffsetPos+4 {
		return exifData, false, 0, fmt.Errorf("invalid IFD1 offset")
	}
	ifd1Offset := int(readUint32(exifData[ifd1OffsetPos : ifd1OffsetPos+4]))
	if ifd1Offset == 0 {
		return exifData, false, 0, nil
	}
	// Estimate thumbnail size: from IFD1 start to end of EXIF data
	thumbStart := pos + ifd1Offset
	thumbSize := int64(len(exifData) - thumbStart)
	// Set IFD1 offset to 0
	result := make([]byte, len(exifData))
	copy(result, exifData)
	if littleEndian {
		binary.LittleEndian.PutUint32(result[ifd1OffsetPos:], 0)
	} else {
		binary.BigEndian.PutUint32(result[ifd1OffsetPos:], 0)
	}
	// Remove data after IFD1
	if thumbStart < len(result) {
		result = result[:thumbStart]
	}
	return result, true, thumbSize, nil
}

// removeGPSFromExif removes GPS IFD from EXIF data
func removeGPSFromExif(exifData []byte) ([]byte, bool, int64) {
	if len(exifData) < 6 || string(exifData[0:6]) != ExifHeader {
		return exifData, false, 0
	}

	// TIFF header starts from byte 6
	pos := 6
	if len(exifData) < pos+8 {
		return exifData, false, 0
	}

	byteOrder := binary.BigEndian.Uint16(exifData[pos : pos+2])
	littleEndian := byteOrder == 0x4949
	var readUint16 func([]byte) uint16
	var readUint32 func([]byte) uint32
	var writeUint32 func([]byte, uint32)
	if littleEndian {
		readUint16 = func(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
		readUint32 = func(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
		writeUint32 = func(b []byte, v uint32) { binary.LittleEndian.PutUint32(b, v) }
	} else {
		readUint16 = func(b []byte) uint16 { return binary.BigEndian.Uint16(b) }
		readUint32 = func(b []byte) uint32 { return binary.BigEndian.Uint32(b) }
		writeUint32 = func(b []byte, v uint32) { binary.BigEndian.PutUint32(b, v) }
	}

	ifd0Offset := int(readUint32(exifData[pos+4 : pos+8]))
	ifd0Pos := pos + ifd0Offset
	if len(exifData) < ifd0Pos+2 {
		return exifData, false, 0
	}

	result := make([]byte, len(exifData))
	copy(result, exifData)

	entryCount := int(readUint16(exifData[ifd0Pos : ifd0Pos+2]))
	gpsIFDOffset := uint32(0)
	gpsTagFound := false

	// Look for GPS IFD pointer tag (0x8825)
	for i := 0; i < entryCount; i++ {
		entryPos := ifd0Pos + 2 + i*12
		if len(exifData) < entryPos+12 {
			break
		}
		tag := readUint16(exifData[entryPos : entryPos+2])
		if tag == 0x8825 { // GPS IFD Pointer
			gpsTagFound = true
			// Get GPS IFD offset
			gpsIFDOffset = readUint32(exifData[entryPos+8 : entryPos+12])
			// Set GPS IFD pointer to 0
			writeUint32(result[entryPos+8:entryPos+12], 0)
			break
		}
	}

	if !gpsTagFound || gpsIFDOffset == 0 {
		return exifData, false, 0
	}

	// Estimate GPS data size (rough estimation)
	gpsDataSize := int64(200) // Typical GPS IFD size

	return result, true, gpsDataSize
}

// removeCameraInfoFromExif removes camera-specific tags from EXIF data
func removeCameraInfoFromExif(exifData []byte) ([]byte, bool, int64) {
	if len(exifData) < 6 || string(exifData[0:6]) != ExifHeader {
		return exifData, false, 0
	}

	// TIFF header starts from byte 6
	pos := 6
	if len(exifData) < pos+8 {
		return exifData, false, 0
	}

	byteOrder := binary.BigEndian.Uint16(exifData[pos : pos+2])
	littleEndian := byteOrder == 0x4949
	var readUint16 func([]byte) uint16
	var readUint32 func([]byte) uint32
	if littleEndian {
		readUint16 = func(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
		readUint32 = func(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
	} else {
		readUint16 = func(b []byte) uint16 { return binary.BigEndian.Uint16(b) }
		readUint32 = func(b []byte) uint32 { return binary.BigEndian.Uint32(b) }
	}

	ifd0Offset := int(readUint32(exifData[pos+4 : pos+8]))
	ifd0Pos := pos + ifd0Offset
	if len(exifData) < ifd0Pos+2 {
		return exifData, false, 0
	}

	result := make([]byte, len(exifData))
	copy(result, exifData)

	entryCount := int(readUint16(exifData[ifd0Pos : ifd0Pos+2]))
	removedSize := int64(0)

	// Tags to remove (camera-specific)
	cameraTagsToRemove := map[uint16]bool{
		0x010F: true, // Make
		0x0110: true, // Model
		0x927C: true, // MakerNote
		0xA005: true, // Interoperability IFD
	}

	// Mark tags for removal by setting their type to 0
	for i := 0; i < entryCount; i++ {
		entryPos := ifd0Pos + 2 + i*12
		if len(exifData) < entryPos+12 {
			break
		}
		tag := readUint16(exifData[entryPos : entryPos+2])
		if cameraTagsToRemove[tag] {
			// Get data size for this tag
			tagType := readUint16(exifData[entryPos+2 : entryPos+4])
			count := readUint32(exifData[entryPos+4 : entryPos+8])
			dataSize := getTagDataSize(tagType, count)
			removedSize += dataSize

			// Zero out the tag entry
			for j := 0; j < 12; j++ {
				result[entryPos+j] = 0
			}
		}
	}

	if removedSize == 0 {
		return exifData, false, 0
	}

	return result, true, removedSize
}

// getTagDataSize calculates the data size for a tag
func getTagDataSize(tagType uint16, count uint32) int64 {
	var typeSize int64
	switch tagType {
	case 1, 2, 6, 7: // BYTE, ASCII, SBYTE, UNDEFINED
		typeSize = 1
	case 3, 8: // SHORT, SSHORT
		typeSize = 2
	case 4, 9, 11: // LONG, SLONG, FLOAT
		typeSize = 4
	case 5, 10, 12: // RATIONAL, SRATIONAL, DOUBLE
		typeSize = 8
	default:
		typeSize = 1
	}
	return typeSize * int64(count)
}

// ReadJpegFile is a helper function to read JPEG file (not implemented)
func ReadJpegFile(path string) ([]byte, error) {
	// This is just for compatibility with the design
	// In real usage, the caller should handle file reading
	return nil, fmt.Errorf("not implemented - please read file externally")
}

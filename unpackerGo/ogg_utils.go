// OGG Utility Functions - CRC calculation, validation, and helper functions
package main

import (
	"bytes"
)

// OGG CRC table for checksum calculation - initialized once
var oggCRCTable [256]uint32

// CalculateOGGPageCRC calculates the CRC32 checksum for an OGG page header
// Uses the standard OGG CRC polynomial and algorithm
func CalculateOGGPageCRC(data []byte) uint32 {
	// CRC polynomial: 0x04C11DB7 (standard for OGG)
	const polynomial = 0x04C11DB7

	// Initialize CRC to all ones
	crc := uint32(0xFFFFFFFF)

	// Process each byte in the page header (excluding CRC field)
	for i := 0; i < len(data); i++ {
		// Skip CRC field (bytes 22-25)
		if i >= 22 && i <= 25 {
			continue
		}

		// XOR current byte with CRC
		crc ^= uint32(data[i]) << 24

		// Process each bit
		for j := 0; j < 8; j++ {
			if crc&0x80000000 != 0 {
				crc = (crc << 1) ^ polynomial
			} else {
				crc <<= 1
			}
		}
	}

	// Return one's complement of final CRC
	return ^crc
}

// CalculateOGGPageCRCCorrect calculates the CRC32 checksum for an OGG page using the correct algorithm
// This version matches the OGG specification more precisely
func CalculateOGGPageCRCCorrect(data []byte) uint32 {
	// Use the bit-by-bit method for accurate CRC calculation
	return CalculateOGGPageCRCBitwise(data)
}

// CalculateOGGPageCRCBitwise calculates CRC using bit-by-bit method (more reliable)
func CalculateOGGPageCRCBitwise(data []byte) uint32 {
	// CRC polynomial: 0x04C11DB7 (standard for OGG)
	const polynomial = 0x04C11DB7

	// Initialize CRC register
	crc := uint32(0x00000000)

	// Process each byte in the data
	for i := 0; i < len(data); i++ {
		// Skip CRC field (bytes 22-25) by setting them to 0
		var currentByte byte
		if i >= 22 && i <= 25 {
			currentByte = 0x00
		} else {
			currentByte = data[i]
		}

		// XOR byte into top byte of CRC
		crc ^= uint32(currentByte) << 24

		// Process each bit
		for bit := 0; bit < 8; bit++ {
			if crc&0x80000000 != 0 {
				crc = (crc << 1) ^ polynomial
			} else {
				crc <<= 1
			}
		}
	}

	return crc
}

// ValidateOGGPageCRC validates the CRC checksum of an OGG page
func ValidateOGGPageCRC(data []byte) bool {
	if len(data) < 26 {
		return false
	}

	// Extract stored CRC
	storedCRC := uint32(data[22]) |
		uint32(data[23])<<8 |
		uint32(data[24])<<16 |
		uint32(data[25])<<24

	// Calculate expected CRC
	calculatedCRC := CalculateOGGPageCRCBitwise(data)

	return storedCRC == calculatedCRC
}

// isValidOGGWithCorrectCRC checks if the data has a valid OGG structure with correct CRC
// This function specifically checks for already-valid OGG files that don't need reconstruction
func isValidOGGWithCorrectCRC(data []byte) bool {
	if len(data) < 27 { // Minimum size for OGG page header with CRC
		return false
	}

	// Check OggS signature at start
	if !bytes.HasPrefix(data, []byte("OggS")) {
		return false
	}

	// Check stream version (should be 0)
	if data[4] != 0x00 {
		return false
	}

	// Validate the CRC of the first page
	// We need to find the end of the first page to validate its CRC
	if len(data) >= 58 { // Check for typical first page size
		firstPage := data[0:58]
		if ValidateOGGPageCRC(firstPage) {
			return true
		}
	}

	// If 58 bytes doesn't work, try to find the actual page boundary
	// Look for segment table and calculate actual page size
	if len(data) >= 27 {
		numSegments := int(data[26])
		if len(data) >= 27+numSegments {
			pageSize := 27 + numSegments
			for i := 0; i < numSegments; i++ {
				pageSize += int(data[27+i])
			}

			if len(data) >= pageSize {
				firstPage := data[0:pageSize]
				return ValidateOGGPageCRC(firstPage)
			}
		}
	}

	return false
}

// GenerateOGGHeaderTemplate creates the standard OGG header template for reconstruction
func GenerateOGGHeaderTemplate() []byte {
	// Standard OGG header template based on analysis of working files
	template := []byte{
		// OggS signature
		0x4F, 0x67, 0x67, 0x53, // "OggS"

		// Stream structure version (always 0)
		0x00,

		// Header type flags (0x02 = first page of logical bitstream)
		0x02,

		// Granule position (8 bytes) - position of last packet ending in this page
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

		// Bitstream serial number (4 bytes) - will be filled from original data
		0x2A, 0x00, 0x00, 0x00, // Default value from analysis

		// Page sequence number (4 bytes) - for first page, this is usually 0
		0x00, 0x00, 0x00, 0x00,

		// CRC checksum (4 bytes) - will be calculated after template creation
		0x00, 0x00, 0x00, 0x00,

		// Page segments count (1 byte) - number of segments in segment table
		0x1E, // 30 segments for our template

		// Segment table (30 bytes) - each byte represents length of a segment
		// Standard values based on typical Vorbis stream structure
		0x01, 0x1E, 0x01, 0x76, 0x6F, 0x72, 0x62, 0x69,
		0x73, 0x00, 0x00, 0x00, 0x00, 0x02, 0x44, 0xAC,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
	}

	return template
}

// initOGGCRCTable initializes the CRC lookup table for faster calculations
func initOGGCRCTable() {
	for i := 0; i < 256; i++ {
		r := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if r&0x80000000 != 0 {
				r = (r << 1) ^ 0x04c11db7
			} else {
				r <<= 1
			}
		}
		oggCRCTable[i] = r
	}
}

// calculateOGGCRC calculates CRC using lookup table method
func calculateOGGCRC(data []byte) uint32 {
	crc := uint32(0)
	for _, b := range data {
		crc = (crc << 8) ^ oggCRCTable[((crc>>24)&0xff)^uint32(b)]
	}
	return crc
}

// SafeOGGReconstructionMode enables extra safety checks during reconstruction
var SafeOGGReconstructionMode = true

// OGGDebugVerbose controls the verbosity of OGG debugging output
var OGGDebugVerbose = false

// EnableOGGDebugMode enables verbose debugging for OGG processing
func EnableOGGDebugMode() {
	OGGDebugVerbose = true
}

// DisableOGGDebugMode disables verbose debugging for OGG processing
func DisableOGGDebugMode() {
	OGGDebugVerbose = false
}

// checkExpectedStructureAtPosition checks if valid OGG structure exists at given position
func checkExpectedStructureAtPosition(data []byte, pos int) bool {
	if pos < 0 || len(data) < pos+58 {
		return false
	}

	// Check if we have OggS at the position
	if !bytes.Equal(data[pos:pos+4], []byte("OggS")) {
		return false
	}

	// Check stream version
	if data[pos+4] != 0x00 {
		return false
	}

	// Check for vorbis signature within reasonable range
	searchStart := pos + 25
	searchEnd := pos + 45
	if searchEnd > len(data) {
		searchEnd = len(data)
	}

	vorbisFound := bytes.Index(data[searchStart:searchEnd], []byte("vorbis")) != -1
	return vorbisFound
}

// FixOGGHeaderCRC fixes the CRC checksum in an existing OGG header
// This is useful when the header structure is correct but the CRC is wrong
func FixOGGHeaderCRC(data []byte) []byte {
	if len(data) < 26 {
		return data // Not enough data for OGG header
	}

	// Find first OggS position
	oggSPos := bytes.Index(data, []byte("OggS"))
	if oggSPos == -1 {
		return data // No OggS found
	}

	// Make a copy of the data to avoid modifying the original
	result := make([]byte, len(data))
	copy(result, data)

	// Calculate correct CRC for the first page
	// We need to find the end of the first page to calculate CRC properly
	firstPageSize := 58 // Default template size
	if oggSPos+firstPageSize <= len(result) {
		// Extract just the first page
		firstPage := result[oggSPos : oggSPos+firstPageSize]

		// Calculate correct CRC
		crc := CalculateOGGPageCRCBitwise(firstPage)

		// Update CRC in the result data
		result[oggSPos+22] = byte(crc & 0xFF)
		result[oggSPos+23] = byte((crc >> 8) & 0xFF)
		result[oggSPos+24] = byte((crc >> 16) & 0xFF)
		result[oggSPos+25] = byte((crc >> 24) & 0xFF)
	}

	return result
}

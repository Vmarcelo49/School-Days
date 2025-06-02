// OGG Debug and Analysis Functions - debugging, validation, and analysis utilities
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

// OGGDebugInfo contains detailed information about OGG data transformations
type OGGDebugInfo struct {
	OriginalSize     int
	CorrectedSize    int
	SizeChange       int
	OriginalHex      string
	CorrectedHex     string
	FirstOggSPos     int
	SecondOggSPos    int
	VorbisPos        int
	DataLossDetected bool
	ChangedBytes     []ByteChange
}

// ByteChange represents a change in a specific byte position
type ByteChange struct {
	Position    int
	OriginalVal byte
	NewVal      byte
	Description string
}

// HexDump creates a formatted hex dump of data with addresses
func HexDump(data []byte, maxBytes int) string {
	if len(data) == 0 {
		return "No data"
	}

	limit := len(data)
	if maxBytes > 0 && limit > maxBytes {
		limit = maxBytes
	}

	var result strings.Builder
	for i := 0; i < limit; i += 16 {
		// Address
		result.WriteString(fmt.Sprintf("%08X: ", i))

		// Hex bytes
		for j := 0; j < 16; j++ {
			if i+j < limit {
				result.WriteString(fmt.Sprintf("%02X ", data[i+j]))
			} else {
				result.WriteString("   ")
			}
		}

		// ASCII representation
		result.WriteString(" |")
		for j := 0; j < 16 && i+j < limit; j++ {
			b := data[i+j]
			if b >= 32 && b <= 126 {
				result.WriteByte(b)
			} else {
				result.WriteByte('.')
			}
		}
		result.WriteString("|\n")
	}

	if maxBytes > 0 && len(data) > maxBytes {
		result.WriteString(fmt.Sprintf("... (truncated, total %d bytes)\n", len(data)))
	}

	return result.String()
}

// CompareOGGData compares original and corrected OGG data and provides detailed analysis
func CompareOGGData(original, corrected []byte, filename string) *OGGDebugInfo {
	debug := &OGGDebugInfo{
		OriginalSize:  len(original),
		CorrectedSize: len(corrected),
		SizeChange:    len(corrected) - len(original),
		ChangedBytes:  make([]ByteChange, 0),
	}

	// Create hex dumps for comparison (first 256 bytes)
	debug.OriginalHex = HexDump(original, 256)
	debug.CorrectedHex = HexDump(corrected, 256)

	// Analyze structure positions
	debug.FirstOggSPos = bytes.Index(original, []byte("OggS"))
	debug.SecondOggSPos = -1
	if debug.FirstOggSPos != -1 {
		secondPos := bytes.Index(original[debug.FirstOggSPos+4:], []byte("OggS"))
		if secondPos != -1 {
			debug.SecondOggSPos = debug.FirstOggSPos + 4 + secondPos
		}
	}
	debug.VorbisPos = bytes.Index(original, []byte("vorbis"))

	// Detect data loss
	if len(corrected) < len(original) {
		debug.DataLossDetected = true
	}

	// Compare byte by byte to find changes
	minLen := len(original)
	if len(corrected) < minLen {
		minLen = len(corrected)
	}

	for i := 0; i < minLen; i++ {
		if original[i] != corrected[i] {
			change := ByteChange{
				Position:    i,
				OriginalVal: original[i],
				NewVal:      corrected[i],
			}

			// Add description based on position
			if i >= 0 && i <= 3 {
				change.Description = "OggS signature"
			} else if i >= 14 && i <= 17 {
				change.Description = "Serial number"
			} else if i >= 22 && i <= 25 {
				change.Description = "CRC checksum"
			} else if i >= 25 && i <= 30 && bytes.Contains(original[i:i+6], []byte("vorbis")) {
				change.Description = "Vorbis signature area"
			} else {
				change.Description = "Header data"
			}

			debug.ChangedBytes = append(debug.ChangedBytes, change)
		}
	}

	// Check for size changes
	if len(corrected) > len(original) {
		for i := len(original); i < len(corrected); i++ {
			change := ByteChange{
				Position:    i,
				OriginalVal: 0x00, // No original byte
				NewVal:      corrected[i],
				Description: "Added byte",
			}
			debug.ChangedBytes = append(debug.ChangedBytes, change)
		}
	}

	return debug
}

// PrintOGGDebugInfo prints detailed debugging information about OGG data changes
func PrintOGGDebugInfo(debug *OGGDebugInfo, filename string) {
	fmt.Printf("\n=== OGG DEBUG INFO for %s ===\n", filename)
	fmt.Printf("Original size: %d bytes\n", debug.OriginalSize)
	fmt.Printf("Corrected size: %d bytes\n", debug.CorrectedSize)
	fmt.Printf("Size change: %+d bytes\n", debug.SizeChange)

	if debug.DataLossDetected {
		fmt.Printf("⚠️  WARNING: Data loss detected! Corrected file is smaller than original.\n")
	}

	fmt.Printf("First OggS position: %d\n", debug.FirstOggSPos)
	fmt.Printf("Second OggS position: %d\n", debug.SecondOggSPos)
	fmt.Printf("Vorbis signature position: %d\n", debug.VorbisPos)

	if len(debug.ChangedBytes) > 0 {
		fmt.Printf("\n--- BYTE CHANGES (%d total) ---\n", len(debug.ChangedBytes))
		for i, change := range debug.ChangedBytes {
			if i >= 20 { // Limit output for readability
				fmt.Printf("... and %d more changes\n", len(debug.ChangedBytes)-i)
				break
			}
			fmt.Printf("  Pos %04X: %02X -> %02X (%s)\n",
				change.Position, change.OriginalVal, change.NewVal, change.Description)
		}
	} else {
		fmt.Printf("No byte changes detected\n")
	}

	fmt.Printf("\n--- ORIGINAL DATA (first 256 bytes) ---\n")
	fmt.Print(debug.OriginalHex)

	fmt.Printf("\n--- CORRECTED DATA (first 256 bytes) ---\n")
	fmt.Print(debug.CorrectedHex)

	fmt.Printf("=== END DEBUG INFO ===\n\n")
}

// AnalyzeOGGCorruption analyzes potential corruption in OGG data
func AnalyzeOGGCorruption(data []byte) {
	fmt.Printf("\n--- OGG CORRUPTION ANALYSIS ---\n")

	// Check for multiple OggS signatures
	oggSPositions := findAllOggSPositions(data)
	fmt.Printf("OggS signatures found at positions: %v\n", oggSPositions)

	// Check for vorbis signatures
	vorbisPos := bytes.Index(data, []byte("vorbis"))
	fmt.Printf("Vorbis signature at position: %d\n", vorbisPos)

	// Check header structure at each OggS position
	for i, pos := range oggSPositions {
		fmt.Printf("\nOggS #%d at position %d:\n", i+1, pos)
		if pos+26 <= len(data) {
			fmt.Printf("  Stream version: %02X\n", data[pos+4])
			fmt.Printf("  Header type: %02X\n", data[pos+5])

			// Extract and display serial number
			if pos+18 <= len(data) {
				serial := data[pos+14 : pos+18]
				fmt.Printf("  Serial number: %02X %02X %02X %02X\n", serial[0], serial[1], serial[2], serial[3])
			}

			// Extract and display CRC
			if pos+26 <= len(data) {
				crc := data[pos+22 : pos+26]
				fmt.Printf("  CRC checksum: %02X %02X %02X %02X\n", crc[0], crc[1], crc[2], crc[3])
			}

			// Check if this looks like a valid header
			if pos+58 <= len(data) {
				isValid := isValidOGGStructure(data[pos:])
				fmt.Printf("  Structure valid: %t\n", isValid)
			}
		}
	}

	// Look for orphaned vorbis signatures
	start := 0
	vorbisCount := 0
	for {
		pos := bytes.Index(data[start:], []byte("vorbis"))
		if pos == -1 {
			break
		}
		vorbisCount++
		actualPos := start + pos
		fmt.Printf("Vorbis signature #%d at position %d\n", vorbisCount, actualPos)

		// Check context around vorbis signature
		contextStart := actualPos - 10
		if contextStart < 0 {
			contextStart = 0
		}
		contextEnd := actualPos + 16
		if contextEnd > len(data) {
			contextEnd = len(data)
		}

		fmt.Printf("  Context: %s\n", HexDump(data[contextStart:contextEnd], 32))
		start = actualPos + 6
	}

	fmt.Printf("--- END CORRUPTION ANALYSIS ---\n\n")
}

// ValidateOGGFile performs comprehensive validation of OGG file structure
func ValidateOGGFile(data []byte, filename string) bool {
	fmt.Printf("    [OGG Validation] Validating %s\n", filename)

	if len(data) < 58 {
		fmt.Printf("    [OGG Validation] ❌ File too small (%d bytes)\n", len(data))
		return false
	}

	// Check first OggS signature
	if !bytes.HasPrefix(data, []byte("OggS")) {
		fmt.Printf("    [OGG Validation] ❌ Missing OggS signature at start\n")
		return false
	}

	// Validate stream version
	if data[4] != 0x00 {
		fmt.Printf("    [OGG Validation] ⚠️  Unusual stream version: %02X\n", data[4])
	}

	// Validate header type
	if data[5] != 0x02 {
		fmt.Printf("    [OGG Validation] ⚠️  Unusual header type: %02X (expected 0x02)\n", data[5])
	}

	// Check for vorbis signature
	vorbisPos := bytes.Index(data[25:45], []byte("vorbis"))
	if vorbisPos == -1 {
		fmt.Printf("    [OGG Validation] ❌ Vorbis signature not found in expected range\n")
		return false
	}

	actualVorbisPos := 25 + vorbisPos
	fmt.Printf("    [OGG Validation] ✓ Vorbis signature found at position %d\n", actualVorbisPos)

	// Validate CRC if possible
	crcValid := ValidateOGGPageCRC(data[:58]) // Check first page only
	if crcValid {
		fmt.Printf("    [OGG Validation] ✓ CRC checksum is valid\n")
	} else {
		fmt.Printf("    [OGG Validation] ⚠️  CRC checksum validation failed\n")
	}

	// Check for second OggS
	secondOggSPos := bytes.Index(data[58:80], []byte("OggS"))
	if secondOggSPos != -1 {
		fmt.Printf("    [OGG Validation] ✓ Second OggS found at position %d\n", 58+secondOggSPos)
	} else {
		fmt.Printf("    [OGG Validation] ⚠️  Second OggS not found in expected range\n")
	}

	// Basic structure validation passed
	fmt.Printf("    [OGG Validation] ✓ Basic structure validation passed\n")
	return true
}

// fixOGGCRC tests and fixes CRC issues in OGG files - utility function for debugging
func fixOGGCRC() {
	initOGGCRCTable()

	fmt.Println("=== OGG CRC Fix Tool ===")

	// Test both files
	files := []string{
		"extracted/SD_BGM/SDBGM01_INT.OGG",
		"extracted_fixed/SD_BGM/SDBGM01_INT.OGG",
	}

	for _, filename := range files {
		fmt.Printf("\nAnalyzing: %s\n", filename)

		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			continue
		}

		if len(data) < 27 {
			fmt.Printf("File too small (%d bytes)\n", len(data))
			continue
		}

		if string(data[:4]) != "OggS" {
			fmt.Printf("Not an OGG file (doesn't start with OggS)\n")
			continue
		}

		// Parse header
		headerType := data[5]
		pageSegments := data[26]
		storedCRC := binary.LittleEndian.Uint32(data[22:26])

		fmt.Printf("Header type: 0x%02X\n", headerType)
		fmt.Printf("Page segments: %d\n", pageSegments)
		fmt.Printf("Stored CRC: 0x%08X\n", storedCRC)

		if 27+int(pageSegments) > len(data) {
			fmt.Printf("Incomplete segment table\n")
			continue
		}

		// Calculate page size
		segmentTable := data[27 : 27+pageSegments]
		pageDataSize := 0
		for _, segment := range segmentTable {
			pageDataSize += int(segment)
		}

		totalPageSize := 27 + int(pageSegments) + pageDataSize
		fmt.Printf("Total page size: %d bytes\n", totalPageSize)

		if totalPageSize > len(data) {
			fmt.Printf("Page extends beyond file (%d > %d)\n", totalPageSize, len(data))
			continue
		}

		// Calculate CRC with checksum field zeroed
		pageData := make([]byte, totalPageSize)
		copy(pageData, data[:totalPageSize])

		// Zero out CRC field
		copy(pageData[22:26], []byte{0, 0, 0, 0})

		calculatedCRC := calculateOGGCRC(pageData)
		fmt.Printf("Calculated CRC: 0x%08X\n", calculatedCRC)

		if calculatedCRC == storedCRC {
			fmt.Printf("✅ CRC is correct\n")
		} else {
			fmt.Printf("❌ CRC MISMATCH!\n")

			// Create fixed version
			fixedData := make([]byte, len(data))
			copy(fixedData, data)
			binary.LittleEndian.PutUint32(fixedData[22:26], calculatedCRC)

			fixedFilename := filename[:len(filename)-4] + "_crc_fixed.OGG"
			err = os.WriteFile(fixedFilename, fixedData, 0644)
			if err != nil {
				fmt.Printf("Error writing fixed file: %v\n", err)
			} else {
				fmt.Printf("💾 Saved CRC-fixed version: %s\n", fixedFilename)
			}
		}
	}
}

// fixAllOGGPages fixes CRC issues in all pages of OGG files - comprehensive repair utility
func fixAllOGGPages() {
	fmt.Println("=== OGG All Pages CRC Fix Tool ===")

	// Test the header-fixed files
	files := []string{
		"extracted_fixed/SD_BGM/SDBGM01_INT.OGG",
		"extracted_fixed/SD_BGM/SDBGM01_LOOP.OGG",
		"extracted_fixed/SD_BGM/SDBGM03_INT.OGG",
	}

	for _, filename := range files {
		fmt.Printf("\n=== Processing: %s ===\n", filename)

		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			continue
		}

		if len(data) < 27 {
			fmt.Printf("File too small (%d bytes)\n", len(data))
			continue
		}

		pos := 0
		pageCount := 0
		pagesFixed := 0
		fileModified := false

		// Process all pages in the file
		for pos < len(data)-26 {
			// Look for OggS signature
			if pos+4 <= len(data) && string(data[pos:pos+4]) == "OggS" {
				fmt.Printf("\nPage %d at position %d\n", pageCount, pos)

				// Parse page header
				if pos+27 > len(data) {
					fmt.Printf("Incomplete header at end of file\n")
					break
				}

				headerType := data[pos+5]
				sequenceNumber := binary.LittleEndian.Uint32(data[pos+18 : pos+22])
				pageSegments := data[pos+26]
				storedCRC := binary.LittleEndian.Uint32(data[pos+22 : pos+26])

				fmt.Printf("  Header type: 0x%02X, Sequence: %d, Segments: %d\n",
					headerType, sequenceNumber, pageSegments)
				fmt.Printf("  Stored CRC: 0x%08X\n", storedCRC)

				if pos+27+int(pageSegments) > len(data) {
					fmt.Printf("  Incomplete segment table\n")
					break
				}

				// Calculate page size
				segmentTable := data[pos+27 : pos+27+int(pageSegments)]
				pageDataSize := 0
				for _, segment := range segmentTable {
					pageDataSize += int(segment)
				}

				totalPageSize := 27 + int(pageSegments) + pageDataSize
				fmt.Printf("  Page size: %d bytes\n", totalPageSize)

				if pos+totalPageSize > len(data) {
					fmt.Printf("  Page extends beyond file (%d > %d)\n",
						pos+totalPageSize, len(data))
					break
				}

				// Calculate CRC with checksum field zeroed
				pageData := make([]byte, totalPageSize)
				copy(pageData, data[pos:pos+totalPageSize])

				// Zero out CRC field
				binary.LittleEndian.PutUint32(pageData[22:26], 0)

				calculatedCRC := CalculateOGGPageCRCBitwise(pageData)
				fmt.Printf("  Calculated CRC: 0x%08X\n", calculatedCRC)

				if calculatedCRC == storedCRC {
					fmt.Printf("  ✅ CRC is correct\n")
				} else {
					fmt.Printf("  ❌ CRC MISMATCH! Fixing...\n")
					binary.LittleEndian.PutUint32(data[pos+22:pos+26], calculatedCRC)
					pagesFixed++
					fileModified = true
				}

				pos += totalPageSize
				pageCount++
			} else {
				pos++
			}
		}

		fmt.Printf("\nSummary: Found %d pages, fixed %d pages\n", pageCount, pagesFixed)

		if fileModified {
			fixedFilename := filename[:len(filename)-4] + "_all_pages_fixed.OGG"
			err = os.WriteFile(fixedFilename, data, 0644)
			if err != nil {
				fmt.Printf("Error writing fixed file: %v\n", err)
			} else {
				fmt.Printf("💾 Saved fully fixed version: %s\n", fixedFilename)
			}
		} else {
			fmt.Printf("✅ No fixes needed - all pages have correct CRCs\n")
		}
	}
}

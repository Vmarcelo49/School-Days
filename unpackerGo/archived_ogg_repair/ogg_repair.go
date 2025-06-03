// OGG Repair and Reconstruction Functions - comprehensive OGG file repair
package main

import (
	"bytes"
	"fmt"
	"os"
)

// ReconstructOGGHeader reconstructs an OGG header with proper CRC calculation
func ReconstructOGGHeader(data []byte) []byte {
	// Create header template
	header := GenerateOGGHeaderTemplate()

	// Fill in known values from original data if available
	if len(data) >= 58 {
		// Try to extract and preserve valid fields from original data
		analysis := AnalyzeOGGStructure(data)

		if analysis.FirstOggSPos != -1 {
			// Copy serial number if available
			serialNumber := ExtractSerialNumberAtPosition(data, analysis.FirstOggSPos)
			copy(header[14:18], serialNumber)

			// Copy granule position if it seems valid
			if analysis.FirstOggSPos+14 < len(data) {
				granulePos := data[analysis.FirstOggSPos+6 : analysis.FirstOggSPos+14]
				copy(header[6:14], granulePos)
			}

			// Copy page sequence number if available
			if analysis.FirstOggSPos+22 < len(data) {
				pageSeq := data[analysis.FirstOggSPos+18 : analysis.FirstOggSPos+22]
				copy(header[18:22], pageSeq)
			}
		}
	}

	// Calculate proper CRC over the reconstructed header
	crc := CalculateOGGPageCRCBitwise(header)

	// Set CRC in header (little-endian format)
	header[22] = byte(crc & 0xFF)
	header[23] = byte((crc >> 8) & 0xFF)
	header[24] = byte((crc >> 16) & 0xFF)
	header[25] = byte((crc >> 24) & 0xFF)

	return header
}

// ReconstructOGGWithTemplate reconstructs an OGG file using the template and original data
// This is a public wrapper function that uses the new CRC calculation methods
func ReconstructOGGWithTemplate(originalData []byte) []byte {
	// Use the improved header reconstruction with proper CRC
	header := ReconstructOGGHeader(originalData)

	// Find the second OggS in the original data for the remaining content
	oggSPos := bytes.Index(originalData, []byte("OggS"))
	if oggSPos != -1 {
		// Look for second OggS
		secondOggSPos := bytes.Index(originalData[oggSPos+4:], []byte("OggS"))
		if secondOggSPos != -1 {
			secondOggSPos += oggSPos + 4 // Adjust for search offset

			// Append everything from the second OggS onwards
			remainingData := originalData[secondOggSPos+4:] // Skip the "OggS" we already have in header
			header = append(header, remainingData...)
		}
	}

	return header
}

// RepairOGGFile attempts comprehensive OGG file repair including structure and CRC
func RepairOGGFile(data []byte) []byte {
	if len(data) < 4 {
		return data
	}

	// First check if it's just a CRC issue
	if IsOGGHeaderCorrect(data) {
		// Structure seems okay, just fix CRC
		if !ValidateOGGPageCRC(data) {
			fmt.Printf("    [OGG Repair] Fixing CRC checksum\n")
			return FixOGGHeaderCRC(data)
		}
		// Already valid
		return data
	}

	// Use the full reconstruction process
	analysis := AnalyzeOGGStructure(data)
	fmt.Printf("    [OGG Repair] %s\n", analysis.Description)

	switch analysis.Status {
	case OGGStatusValid:
		return data
	case OGGStatusMissingFirstOggS:
		fmt.Printf("    [OGG Repair] Adding missing first OggS\n")
		return ReconstructOGGWithTemplate(data)
	default:
		fmt.Printf("    [OGG Repair] Full reconstruction needed\n")
		return ReconstructOGGWithTemplate(data)
	}
}

// reconstructOGGWithTemplate performs the actual template-based reconstruction (from ogg_processing.go)
// This is an enhanced version that supplements the existing implementation
func reconstructOGGWithTemplateEnhanced(data []byte, filename string) []byte {
	// Extract components from original data
	serialNumber := ExtractSerialNumber(data)
	crcChecksum := ExtractCRCChecksum(data)

	// Generate the header template
	template := GenerateOGGHeaderTemplate()

	// Insert the extracted serial number at position 14-17
	copy(template[14:18], serialNumber)

	// Insert the extracted CRC checksum at position 22-25
	copy(template[22:26], crcChecksum)

	// Find vorbis data in original
	vorbisPos := bytes.Index(data, []byte("vorbis"))
	if vorbisPos == -1 {
		fmt.Printf("    [Template Enhanced] ⚠️ No vorbis signature found in %s\n", filename)
		return data
	}

	// Construct final result
	result := make([]byte, 0, len(template)+len(data)-vorbisPos)
	result = append(result, template...)
	result = append(result, data[vorbisPos:]...)

	// Recalculate CRC for the reconstructed header
	if len(result) >= 58 {
		correctedCRC := CalculateOGGPageCRCBitwise(result[:58])
		result[22] = byte(correctedCRC & 0xFF)
		result[23] = byte((correctedCRC >> 8) & 0xFF)
		result[24] = byte((correctedCRC >> 16) & 0xFF)
		result[25] = byte((correctedCRC >> 24) & 0xFF)
	}

	fmt.Printf("    [Template Enhanced] ✓ Reconstructed %s with enhanced template (size: %d bytes)\n", filename, len(result))
	return result
}

// SafeReconstructOGGWithTemplateV2 provides an alternative reconstruction method with additional safety
func SafeReconstructOGGWithTemplateV2(data []byte, filename string) []byte {
	// Safety check: don't modify already valid files
	if isValidOGGWithCorrectCRC(data) {
		fmt.Printf("    [Safe OGG V2] ✓ File %s has valid CRC, no reconstruction needed\n", filename)
		return data
	}

	// Create debug info for comparison
	original := make([]byte, len(data))
	copy(original, data)

	// Use enhanced reconstruction
	result := reconstructOGGWithTemplateEnhanced(data, filename)

	// Safety validation
	if len(result) < len(original)/2 {
		fmt.Printf("    [Safe OGG V2] ⚠️ Reconstruction result too small for %s, using fallback\n", filename)
		return RepairOGGFile(data)
	}

	// Validate the reconstruction
	if ValidateOGGFile(result, filename) {
		fmt.Printf("    [Safe OGG V2] ✓ Reconstruction validation passed for %s\n", filename)
		return result
	} else {
		fmt.Printf("    [Safe OGG V2] ⚠️ Reconstruction validation failed for %s, using original\n", filename)
		return data
	}
}

// SmartOGGRepair performs intelligent OGG repair based on detected issues
func SmartOGGRepair(data []byte, filename string) []byte {
	if len(data) < 27 {
		fmt.Printf("    [Smart Repair] File %s too small for repair\n", filename)
		return data
	}

	// Analyze the file to determine the best repair strategy
	analysis := AnalyzeOGGStructure(data)

	fmt.Printf("    [Smart Repair] %s: %s\n", filename, analysis.Description)

	switch analysis.Status {
	case OGGStatusValid:
		// Check if CRC needs fixing
		if !isValidOGGWithCorrectCRC(data) {
			fmt.Printf("    [Smart Repair] Fixing CRC for %s\n", filename)
			return FixOGGHeaderCRC(data)
		}
		return data

	case OGGStatusMissingFirstOggS:
		// Simple case: just add the missing OggS
		fmt.Printf("    [Smart Repair] Adding missing OggS for %s\n", filename)
		if analysis.FirstOggSPos > 0 && analysis.HasValidStructure {
			// Prepend OggS to the data starting from the found position
			result := make([]byte, 0, len(data)+4)
			result = append(result, []byte("OggS")...)
			result = append(result, data[analysis.FirstOggSPos:]...)
			return result
		}
		// Fallback to template reconstruction
		return SafeReconstructOGGWithTemplate(data, filename)

	case OGGStatusCorruptedHeader:
		// More complex repair needed
		fmt.Printf("    [Smart Repair] Comprehensive header repair for %s\n", filename)
		return SafeReconstructOGGWithTemplateV2(data, filename)

	default:
		// Unknown issue, use most comprehensive repair
		fmt.Printf("    [Smart Repair] Full reconstruction for %s\n", filename)
		return SafeReconstructOGGWithTemplate(data, filename)
	}
}

// BatchRepairOGGFiles repairs multiple OGG files with progress reporting
func BatchRepairOGGFiles(files []string) {
	fmt.Printf("=== Batch OGG Repair ===\n")
	fmt.Printf("Processing %d files...\n\n", len(files))

	repaired := 0
	skipped := 0
	failed := 0

	for i, filename := range files {
		fmt.Printf("[%d/%d] Processing: %s\n", i+1, len(files), filename)

		// Read file
		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("    ❌ Error reading file: %v\n", err)
			failed++
			continue
		}

		// Analyze and repair
		originalSize := len(data)
		repairedData := SmartOGGRepair(data, filename)

		if bytes.Equal(data, repairedData) {
			fmt.Printf("    ✓ No repair needed\n")
			skipped++
		} else {
			// Save repaired file
			repairedFilename := filename[:len(filename)-4] + "_repaired.OGG"
			err = os.WriteFile(repairedFilename, repairedData, 0644)
			if err != nil {
				fmt.Printf("    ❌ Error saving repaired file: %v\n", err)
				failed++
			} else {
				sizeChange := len(repairedData) - originalSize
				fmt.Printf("    ✓ Repaired and saved as %s (size change: %+d bytes)\n",
					repairedFilename, sizeChange)
				repaired++
			}
		}
		fmt.Println()
	}

	fmt.Printf("=== Batch Repair Complete ===\n")
	fmt.Printf("Repaired: %d\n", repaired)
	fmt.Printf("Skipped: %d\n", skipped)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Total: %d\n", len(files))
}

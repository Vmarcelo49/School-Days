// OGG Data Processing and Reconstruction Functions
package main

import (
	"bytes"
)

// processOGGDataWithDebug analyzes and fixes OGG file header structure with detailed debugging
func (g *GPK) processOGGDataWithDebug(data []byte, filename string) []byte {
	if len(data) < 4 {
		return data
	}

	// Store original data for comparison
	originalData := make([]byte, len(data))
	copy(originalData, data)
	// Analyze potential corruption first
	DebugPrintf("    [OGG Debug] Analyzing corruption patterns for %s\n", filename)
	AnalyzeOGGCorruption(data)

	// First, analyze the current structure
	analysis := AnalyzeOGGStructure(data)

	DebugPrintf("    [OGG Analysis] %s\n", analysis.Description)

	var result []byte

	switch analysis.Status {
	case OGGStatusValid:
		// File is already valid, return as-is
		result = data
	case OGGStatusMissingFirstOggS:
		// Missing first OggS, but has valid structure - use simple prepend
		VerbosePrintf(LogVerbose, "    [OGG Fix] Adding missing first OggS signature\n")
		result = g.addMissingFirstOggS(data, analysis)

	case OGGStatusCorruptedHeader:
		// Header is corrupted, try template reconstruction
		VerbosePrintf(LogVerbose, "    [OGG Fix] Attempting template-based reconstruction\n")
		result = g.reconstructWithTemplate(data)

	case OGGStatusNoOggS:
		// No OggS found at all - try template reconstruction as last resort
		VerbosePrintf(LogVerbose, "    [OGG Fix] No OggS found, attempting full reconstruction\n")
		result = g.reconstructWithTemplate(data)

	default:
		// Unknown status, return original data
		result = data
	}

	// Compare original vs corrected data
	if !bytes.Equal(originalData, result) {
		debug := CompareOGGData(originalData, result, filename)
		PrintOGGDebugInfo(debug, filename)
		// Additional validation
		if debug.DataLossDetected {
			VerbosePrintf(LogVerbose, "    ⚠️  [WARNING] Data loss detected in %s! Consider manual review.\n", filename)
		}

		if len(debug.ChangedBytes) > 50 {
			VerbosePrintf(LogVerbose, "    ⚠️  [WARNING] Extensive changes (%d bytes) in %s! Verify integrity.\n",
				len(debug.ChangedBytes), filename)
		}
	} else {
		DebugPrintf("    [OGG Debug] No changes made to %s (file was already valid)\n", filename)
	}

	return result
}

// addMissingFirstOggS adds the missing first OggS when structure is otherwise valid
func (g *GPK) addMissingFirstOggS(data []byte, analysis *OGGAnalysis) []byte {
	if analysis.FirstOggSPos == -1 {
		return data
	}

	fixedData := make([]byte, 0, len(data)+4)
	fixedData = append(fixedData, []byte("OggS")...)
	fixedData = append(fixedData, data[:analysis.FirstOggSPos]...)
	fixedData = append(fixedData, data[analysis.FirstOggSPos:]...)

	return fixedData
}

// reconstructWithTemplate uses the template-based reconstruction for severely corrupted files
func (g *GPK) reconstructWithTemplate(data []byte) []byte { // Safety check: if the file already has valid CRC, don't reconstruct
	if isValidOGGWithCorrectCRC(data) {
		DebugPrintf("    [Safe OGG] ✓ File has valid CRC, skipping template reconstruction\n")
		return data
	}

	// Use safe reconstruction mode for better data integrity
	return SafeReconstructOGGWithTemplate(data, "unknown")
}

// SafeReconstructOGGWithTemplate reconstructs an OGG file with additional safety checks
// This function includes validation to prevent data loss and extensive changes
func SafeReconstructOGGWithTemplate(data []byte, filename string) []byte {
	if len(data) < 4 {
		DebugPrintf("    [Safe OGG] ⚠️  Data too small for reconstruction (%d bytes)\n", len(data))
		return data
	}

	// Store original data for comparison
	originalData := make([]byte, len(data))
	copy(originalData, data)

	// Analyze original data first
	analysis := AnalyzeOGGStructure(data)
	// Safety check: If we have some valid structure, be conservative
	if analysis.VorbisPos != -1 && analysis.FirstOggSPos != -1 {
		vorbisDistance := analysis.VorbisPos - analysis.FirstOggSPos
		if vorbisDistance > 0 && vorbisDistance < 100 {
			DebugPrintf("    [Safe OGG] ✓ Found valid vorbis at reasonable distance (%d bytes), using conservative fix\n", vorbisDistance)

			// Try simple prepend first if missing first OggS
			if analysis.Status == OGGStatusMissingFirstOggS {
				result := make([]byte, 0, len(data)+4)
				result = append(result, []byte("OggS")...)
				result = append(result, data...)
				return result
			}
		}
	}

	// Continue with template reconstruction for severely corrupted files
	return reconstructOGGWithTemplate(data, filename)
}

// reconstructOGGWithTemplate performs the actual template-based reconstruction
func reconstructOGGWithTemplate(data []byte, filename string) []byte {
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
		DebugPrintf("    [Template] ⚠️ No vorbis signature found in %s\n", filename)
		return data
	}

	// Construct final result
	result := make([]byte, 0, len(template)+len(data)-vorbisPos)
	result = append(result, template...)
	result = append(result, data[vorbisPos:]...)

	DebugPrintf("    [Template] ✓ Reconstructed %s with template (size: %d bytes)\n", filename, len(result))
	return result
}

// OGG File Analysis Functions
package main

import (
	"bytes"
	"fmt"
)

// AnalyzeOGGStructure analyzes the structure of OGG data and determines what fixes are needed
func AnalyzeOGGStructure(data []byte) *OGGAnalysis {
	analysis := &OGGAnalysis{
		Status:        OGGStatusUnknown,
		FirstOggSPos:  -1,
		SecondOggSPos: -1,
		VorbisPos:     -1,
	}

	if len(data) < 4 {
		analysis.Status = OGGStatusNoOggS
		analysis.Description = "Data too short for OGG analysis"
		return analysis
	}

	// Find all OggS positions
	oggSPositions := findAllOggSPositions(data)

	if len(oggSPositions) == 0 {
		analysis.Status = OGGStatusNoOggS
		analysis.Description = "No OggS signatures found"
		return analysis
	}

	analysis.FirstOggSPos = oggSPositions[0]
	if len(oggSPositions) > 1 {
		analysis.SecondOggSPos = oggSPositions[1]
	}

	// Find vorbis signature
	vorbisPos := bytes.Index(data, []byte("vorbis"))
	if vorbisPos != -1 {
		analysis.VorbisPos = vorbisPos
	}

	// Check if structure matches our expected pattern
	if analysis.FirstOggSPos == 0 {
		// OggS at position 0 - first check if it has valid CRC (most important)
		if isValidOGGWithCorrectCRC(data) {
			analysis.Status = OGGStatusValid
			analysis.Description = "Valid OGG structure with correct CRC detected"
			analysis.HasValidStructure = true
		} else if isValidOGGStructure(data) {
			analysis.Status = OGGStatusValid
			analysis.Description = "Valid OGG structure detected (CRC may need fixing)"
			analysis.HasValidStructure = true
		} else {
			analysis.Status = OGGStatusCorruptedHeader
			analysis.Description = "OggS at position 0 but header structure is corrupted"
		}
	} else if analysis.FirstOggSPos > 0 {
		// OggS found but not at position 0
		expectedStructure := checkExpectedStructureAtPosition(data, analysis.FirstOggSPos)
		if expectedStructure {
			analysis.Status = OGGStatusMissingFirstOggS
			analysis.Description = fmt.Sprintf("Missing first OggS, found valid structure at position %d", analysis.FirstOggSPos)
			analysis.HasValidStructure = true
		} else {
			analysis.Status = OGGStatusCorruptedHeader
			analysis.Description = fmt.Sprintf("OggS found at position %d but structure appears corrupted", analysis.FirstOggSPos)
		}
	}

	return analysis
}

// findAllOggSPositions finds all positions where "OggS" appears in the data
func findAllOggSPositions(data []byte) []int {
	var positions []int
	oggS := []byte("OggS")

	start := 0
	for {
		pos := bytes.Index(data[start:], oggS)
		if pos == -1 {
			break
		}
		positions = append(positions, start+pos)
		start = start + pos + 4
	}

	return positions
}

// isValidOGGStructure checks if the data follows the expected OGG structure starting at position 0
func isValidOGGStructure(data []byte) bool {
	if len(data) < 58 { // Minimum size for our expected header
		return false
	}

	// Check OggS signature
	if !bytes.HasPrefix(data, []byte("OggS")) {
		return false
	}

	// Check stream version (should be 0)
	if data[4] != 0x00 {
		return false
	}

	// Check header type flags (should be 0x02 for first page)
	if data[5] != 0x02 {
		return false
	}

	// Check for vorbis signature at expected position
	vorbisPos := bytes.Index(data[25:35], []byte("vorbis"))
	if vorbisPos == -1 {
		return false
	}

	// Look for second OggS within reasonable range (58-70 bytes)
	secondOggS := bytes.Index(data[58:70], []byte("OggS"))
	return secondOggS != -1
}

// checkExpectedStructureAtPosition is implemented in ogg_utils.go
// This function has been moved to ogg_utils.go for better organization

// isValidOGGWithCorrectCRC is implemented in ogg_utils.go
// This function has been moved to ogg_utils.go for better organization
